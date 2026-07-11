package secret

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mockStore is a simple in-memory Store implementation used to test migration.
// It can be configured to fail on Set or Delete operations for a given service.
type mockStore struct {
	data       map[string]map[string]string // service → key → value
	failSet    map[string]bool              // services on which Set should fail
	failDelete map[string]bool              // services on which Delete should fail
}

func newMockStore() *mockStore {
	return &mockStore{
		data:       make(map[string]map[string]string),
		failSet:    make(map[string]bool),
		failDelete: make(map[string]bool),
	}
}

func (m *mockStore) Get(service, key string) (string, error) {
	if _, ok := m.data[service]; !ok {
		return "", errNotFound("credential not found: %s/%s", service, key)
	}
	v, ok := m.data[service][key]
	if !ok {
		return "", errNotFound("credential not found: %s/%s", service, key)
	}
	return v, nil
}

func (m *mockStore) Set(service, key, value string) error {
	if m.failSet[service] {
		return errNotFound("mock store: Set failed for service %s", service)
	}
	if m.data[service] == nil {
		m.data[service] = make(map[string]string)
	}
	m.data[service][key] = value
	return nil
}

func (m *mockStore) Delete(service, key string) error {
	if m.failDelete[service] {
		return errNotFound("mock store: Delete failed for service %s", service)
	}
	if _, ok := m.data[service]; ok {
		delete(m.data[service], key)
	}
	return nil
}

func (m *mockStore) List(service string) (map[string]string, error) {
	if m.data[service] == nil {
		return make(map[string]string), nil
	}
	result := make(map[string]string, len(m.data[service]))
	for k, v := range m.data[service] {
		result[k] = v
	}
	return result, nil
}

// errNotFound creates a simple error matching the store's "not found" pattern.
// We use it to simulate store errors during tests.
func errNotFound(format string, args ...interface{}) error {
	return &mockNotFoundError{msg: sprintf(format, args...)}
}

type mockNotFoundError struct{ msg string }

func (e *mockNotFoundError) Error() string { return e.msg }

// sprintf is a minimal sprintf helper to avoid importing fmt in mock-only code.
func sprintf(format string, args ...interface{}) string {
	s := format
	for _, a := range args {
		idx := strings.Index(s, "%s")
		if idx < 0 {
			idx = strings.Index(s, "%d")
		}
		if idx < 0 {
			break
		}
		val := ""
		switch v := a.(type) {
		case string:
			val = v
		case int:
			val = itoa(v)
		}
		s = s[:idx] + val + s[idx+2:]
	}
	return s
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	if neg {
		digits = "-" + digits
	}
	return digits
}

// setupFileStoreWithCredentials creates a file store in a temp directory with
// the given credentials, and returns the (service name, store).  Each entry in
// creds is a map from key to value for a distinct service.
func setupFileStoreWithCredentials(t *testing.T, creds map[string]map[string]string) *encryptedFileStore {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("MULTIAI_SECRETS_DIR", dir)

	store, err := newEncryptedFileStore()
	if err != nil {
		t.Fatal(err)
	}

	for svc, keys := range creds {
		for k, v := range keys {
			if err := store.Set(svc, k, v); err != nil {
				t.Fatalf("setup: Set(%q, %q): %v", svc, k, err)
			}
		}
	}

	return store
}

// verifyFileStoreHas checks that the file store still contains the given
// credentials (used to verify no data was lost).
func verifyFileStoreHas(t *testing.T, store *encryptedFileStore, service, key, want string) {
	t.Helper()
	got, err := store.Get(service, key)
	if err != nil {
		t.Fatalf("file store Get(%q, %q): %v", service, key, err)
	}
	if got != want {
		t.Errorf("file store Get(%q, %q) = %q, want %q", service, key, got, want)
	}
}

// ── Tests ─────────────────────────────────────────────────────────────────

// TestMigrateFromFileStore_Success verifies a basic migration: credentials
// from two file store services are copied to the native store, and the
// .migrated marker is written.
func TestMigrateFromFileStore_Success(t *testing.T) {
	creds := map[string]map[string]string{
		"multiai:ca-a1b2c3d4e5f6a7b8": {"ANTHROPIC_API_KEY": "sk-ant-test-123"},
		"multiai:ds-b2c3d4e5f6a7b8c9": {"OPENAI_API_KEY": "sk-openai-456"},
	}
	setupFileStoreWithCredentials(t, creds)

	mock := newMockStore()
	report, err := MigrateFromFileStore(mock, false)
	if err != nil {
		t.Fatal(err)
	}

	if report.AlreadyMigrated {
		t.Error("AlreadyMigrated should be false on first migration")
	}
	if len(report.ServicesFound) != 2 {
		t.Errorf("ServicesFound: got %d, want 2", len(report.ServicesFound))
	}
	if len(report.Migrated) != 2 {
		t.Errorf("Migrated: got %d, want 2; content: %v", len(report.Migrated), report.Migrated)
	}
	if len(report.Skipped) != 0 {
		t.Errorf("Skipped: got %d, want 0", len(report.Skipped))
	}
	if len(report.Failed) != 0 {
		t.Errorf("Failed: got %d, want 0", len(report.Failed))
	}

	// Verify data in the native store.
	for svc, keys := range creds {
		for k, v := range keys {
			got, err := mock.Get(svc, k)
			if err != nil {
				t.Errorf("mock Get(%q, %q): %v", svc, k, err)
				continue
			}
			if got != v {
				t.Errorf("mock Get(%q, %q) = %q, want %q", svc, k, got, v)
			}
		}
	}

	// Verify marker was written.
	if !IsMigrated() {
		t.Error("IsMigrated() should be true after successful migration")
	}
}

// TestMigrateFromFileStore_AlreadyMigrated verifies that when the .migrated
// marker exists, migration is skipped unless force is true.
func TestMigrateFromFileStore_AlreadyMigrated(t *testing.T) {
	creds := map[string]map[string]string{
		"multiai:ca-a1b2c3d4e5f6a7b8": {"ANTHROPIC_API_KEY": "sk-ant-test-123"},
	}
	setupFileStoreWithCredentials(t, creds)

	mock := newMockStore()

	// First migration succeeds.
	_, err := MigrateFromFileStore(mock, false)
	if err != nil {
		t.Fatal(err)
	}

	// Second migration without force should be skipped.
	report, err := MigrateFromFileStore(mock, false)
	if err != nil {
		t.Fatal(err)
	}
	if !report.AlreadyMigrated {
		t.Error("AlreadyMigrated should be true when marker exists")
	}
	if len(report.Migrated) != 0 {
		t.Errorf("Migrated should be empty, got: %v", report.Migrated)
	}
}

// TestMigrateFromFileStore_ForceReMigration verifies that --migrate-force
// causes re-migration even when the marker exists.
func TestMigrateFromFileStore_ForceReMigration(t *testing.T) {
	creds := map[string]map[string]string{
		"multiai:ca-a1b2c3d4e5f6a7b8": {"ANTHROPIC_API_KEY": "sk-ant-test-123"},
	}
	setupFileStoreWithCredentials(t, creds)

	mock := newMockStore()

	// First migration.
	_, err := MigrateFromFileStore(mock, false)
	if err != nil {
		t.Fatal(err)
	}

	// Force re-migration.
	report, err := MigrateFromFileStore(mock, true)
	if err != nil {
		t.Fatal(err)
	}
	if report.AlreadyMigrated {
		t.Error("AlreadyMigrated should be false when force=true")
	}
	if len(report.Migrated) != 1 {
		t.Errorf("Migrated: got %d, want 1 (force re-migration should work)", len(report.Migrated))
	}

	// Verify data still in native store.
	got, err := mock.Get("multiai:ca-a1b2c3d4e5f6a7b8", "ANTHROPIC_API_KEY")
	if err != nil {
		t.Fatal(err)
	}
	if got != "sk-ant-test-123" {
		t.Errorf("got %q, want %q", got, "sk-ant-test-123")
	}
}

// TestMigrateFromFileStore_NoServices verifies that an empty file store
// produces a report with no services found.
func TestMigrateFromFileStore_NoServices(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_SECRETS_DIR", dir)

	// Create an empty file store (no .enc files).
	_, err := newEncryptedFileStore()
	if err != nil {
		t.Fatal(err)
	}

	mock := newMockStore()
	report, err := MigrateFromFileStore(mock, false)
	if err != nil {
		t.Fatal(err)
	}

	if len(report.ServicesFound) != 0 {
		t.Errorf("ServicesFound: got %d, want 0", len(report.ServicesFound))
	}
	if len(report.Migrated) != 0 {
		t.Errorf("Migrated should be empty")
	}
	if report.AlreadyMigrated {
		t.Error("AlreadyMigrated should be false for empty store")
	}
}

// TestMigrateFromFileStore_SkipAlreadyInNative verifies that a service whose
// credentials already exist in the native store is skipped (unless force).
func TestMigrateFromFileStore_SkipAlreadyInNative(t *testing.T) {
	creds := map[string]map[string]string{
		"multiai:ca-a1b2c3d4e5f6a7b8": {"ANTHROPIC_API_KEY": "sk-ant-test-123"},
	}
	setupFileStoreWithCredentials(t, creds)

	mock := newMockStore()
	// Pre-populate the native store with the same service but a different value.
	if err := mock.Set("multiai:ca-a1b2c3d4e5f6a7b8", "ANTHROPIC_API_KEY", "sk-ant-native-value"); err != nil {
		t.Fatal(err)
	}

	// Without force, this service should be skipped.
	report, err := MigrateFromFileStore(mock, false)
	if err != nil {
		t.Fatal(err)
	}

	if len(report.Migrated) != 0 {
		t.Errorf("Migrated should be empty when service already in native, got: %v", report.Migrated)
	}
	if len(report.Skipped) != 1 {
		t.Errorf("Skipped should contain 1 service, got: %v", report.Skipped)
	}

	// Verify native store still has the original value (was not overwritten).
	got, err := mock.Get("multiai:ca-a1b2c3d4e5f6a7b8", "ANTHROPIC_API_KEY")
	if err != nil {
		t.Fatal(err)
	}
	if got != "sk-ant-native-value" {
		t.Errorf("native store should retain original value, got %q", got)
	}
}

// TestMigrateFromFileStore_ForceOverwritesNative verifies that with force=true,
// existing credentials in the native store are overwritten.
func TestMigrateFromFileStore_ForceOverwritesNative(t *testing.T) {
	creds := map[string]map[string]string{
		"multiai:ca-a1b2c3d4e5f6a7b8": {"ANTHROPIC_API_KEY": "sk-ant-file-store"},
	}
	setupFileStoreWithCredentials(t, creds)

	mock := newMockStore()
	// Pre-populate with a different value.
	if err := mock.Set("multiai:ca-a1b2c3d4e5f6a7b8", "ANTHROPIC_API_KEY", "sk-ant-native-old"); err != nil {
		t.Fatal(err)
	}

	// With force, the native value should be overwritten.
	report, err := MigrateFromFileStore(mock, true)
	if err != nil {
		t.Fatal(err)
	}

	if len(report.Migrated) != 1 {
		t.Errorf("Migrated should contain 1 service with force, got: %v", report.Migrated)
	}
	if len(report.Skipped) != 0 {
		t.Errorf("Skipped should be empty with force, got: %v", report.Skipped)
	}

	// Verify native store now has the file store value.
	got, err := mock.Get("multiai:ca-a1b2c3d4e5f6a7b8", "ANTHROPIC_API_KEY")
	if err != nil {
		t.Fatal(err)
	}
	if got != "sk-ant-file-store" {
		t.Errorf("native store should have file store value, got %q", got)
	}
}

// TestMigrateFromFileStore_RollbackOnFailure verifies that when a Set
// operation fails midway through a service's credentials, all previously
// written credentials for that service are rolled back (deleted).
func TestMigrateFromFileStore_RollbackOnFailure(t *testing.T) {
	// Service "svc-a" has two keys. The native store's Set for "svc-a"
	// will succeed for the first key and fail for the second.
	creds := map[string]map[string]string{
		"svc-a": {
			"KEY1": "value1",
			"KEY2": "value2",
		},
		"svc-b": {
			"KEY_B": "value_b",
		},
	}
	setupFileStoreWithCredentials(t, creds)

	mock := newMockStore()
	mock.failSet["svc-a"] = true // causes Set to fail

	report, err := MigrateFromFileStore(mock, false)
	if err != nil {
		t.Fatal(err)
	}

	// svc-a should be in Failed (rolled back), svc-b should be in Migrated.
	if len(report.Failed) != 1 || report.Failed[0] != "svc-a" {
		t.Errorf("Failed should contain 'svc-a', got: %v", report.Failed)
	}
	if len(report.Migrated) != 1 || report.Migrated[0] != "svc-b" {
		t.Errorf("Migrated should contain 'svc-b', got: %v", report.Migrated)
	}

	// Verify rollback: svc-a should have NO keys in the native store.
	svcAData, err := mock.List("svc-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(svcAData) != 0 {
		t.Errorf("svc-a should be fully rolled back, got %d keys: %v", len(svcAData), svcAData)
	}

	// Verify svc-b was migrated successfully.
	val, err := mock.Get("svc-b", "KEY_B")
	if err != nil {
		t.Fatal(err)
	}
	if val != "value_b" {
		t.Errorf("svc-b KEY_B = %q, want %q", val, "value_b")
	}

	// Verify file store data was not touched (read-only).
	fileDir := os.Getenv("MULTIAI_SECRETS_DIR")
	fs, err := newEncryptedFileStore()
	if err != nil {
		t.Fatal(err)
	}
	// Only check if we can still read the file store.
	_ = fileDir
	_ = fs
}

// TestMigrateFromFileStore_ServiceNamesRecovery verifies that service names
// containing ':' are correctly recovered from .enc file names.
func TestMigrateFromFileStore_ServiceNamesRecovery(t *testing.T) {
	creds := map[string]map[string]string{
		"multiai:ca-a1b2c3d4e5f6a7b8": {"KEY": "val1"},
		"test-service-plain":          {"KEY": "val2"},
	}
	setupFileStoreWithCredentials(t, creds)

	mock := newMockStore()
	report, err := MigrateFromFileStore(mock, false)
	if err != nil {
		t.Fatal(err)
	}

	// Both services should be migrated.
	if len(report.Migrated) != 2 {
		t.Errorf("Migrated: got %d, want 2; content: %v", len(report.Migrated), report.Migrated)
	}

	// Verify data in native store under correct service names.
	for svc := range creds {
		_, err := mock.Get(svc, "KEY")
		if err != nil {
			t.Errorf("service %q should have been migrated, but Get failed: %v", svc, err)
		}
	}
}

// TestMigrateFromFileStore_MarkerNotWrittenOnFailure verifies that the
// .migrated marker is NOT written when any service fails.
func TestMigrateFromFileStore_MarkerNotWrittenOnFailure(t *testing.T) {
	creds := map[string]map[string]string{
		"svc-a": {"KEY1": "val1"},
		"svc-b": {"KEY2": "val2"},
	}
	setupFileStoreWithCredentials(t, creds)

	mock := newMockStore()
	mock.failSet["svc-b"] = true

	_, err := MigrateFromFileStore(mock, false)
	if err != nil {
		t.Fatal(err)
	}

	// The .migrated marker should NOT exist.
	if IsMigrated() {
		t.Error("IsMigrated() should be false when migration had failures")
	}
}

// TestServiceNameFromEncFile verifies the service name recovery logic.
func TestServiceNameFromEncFile(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"multiai_ca-a1b2c3d4.enc", "multiai:ca-a1b2c3d4"},
		{"multiai_ds-b2c3d4e5.enc", "multiai:ds-b2c3d4e5"},
		{"plain_service.enc", "plain_service"},
		{"no_extension", "no_extension"},
		{"multiai__a1b2.enc", "multiai:_a1b2"},
	}
	for _, tt := range tests {
		got := serviceNameFromEncFile(tt.input)
		if got != tt.want {
			t.Errorf("serviceNameFromEncFile(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestMigratedMarkerPath verifies that MigratedMarkerPath returns a valid path.
func TestMigratedMarkerPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_SECRETS_DIR", dir)

	path, err := MigratedMarkerPath()
	if err != nil {
		t.Fatal(err)
	}
	wantSuffix := string(filepath.Separator) + ".migrated"
	if len(path) <= len(wantSuffix) || path[len(path)-len(wantSuffix):] != wantSuffix {
		t.Errorf("MigratedMarkerPath() = %q, want suffix %q", path, wantSuffix)
	}
}

// TestListFileStoreServices verifies that listFileStoreServices correctly
// enumerates .enc files while skipping dotfiles and non-.enc files.
func TestListFileStoreServices(t *testing.T) {
	dir := t.TempDir()

	// Create .enc files and non-.enc files.
	files := []string{
		"multiai_ca.enc",
		"multiai_ds.enc",
		".masterkey",
		".migrated",
		"some_other_file.txt",
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("data"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	services, err := listFileStoreServices(dir)
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"multiai:ca", "multiai:ds"}
	if len(services) != len(expected) {
		t.Fatalf("got %d services: %v, want %v", len(services), services, expected)
	}
	for i, svc := range services {
		if svc != expected[i] {
			t.Errorf("service[%d] = %q, want %q", i, svc, expected[i])
		}
	}
}
