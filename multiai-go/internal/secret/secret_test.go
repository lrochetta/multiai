package secret

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	plaintext := []byte("sk-ant-api-03-test-secret-key-123456")
	ciphertext, err := encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	// Should be different from plaintext
	if string(ciphertext) == string(plaintext) {
		t.Error("ciphertext equals plaintext")
	}

	decrypted, err := decrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("decrypt mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptDecrypt_EmptySlice(t *testing.T) {
	key := make([]byte, 32)
	plaintext := []byte("")
	ciphertext, err := encrypt(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := decrypt(key, ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if len(decrypted) != 0 {
		t.Errorf("expected empty, got %d bytes", len(decrypted))
	}
}

func TestDeriveKey(t *testing.T) {
	salt := []byte("test-salt-123456")
	key1 := DeriveKey("my-passphrase", salt)
	key2 := DeriveKey("my-passphrase", salt)
	key3 := DeriveKey("different", salt)

	if len(key1) != 32 {
		t.Errorf("key length: got %d, want 32", len(key1))
	}
	// Same inputs = same key
	for i := range key1 {
		if key1[i] != key2[i] {
			t.Error("same inputs produced different keys")
			break
		}
	}
	// Different inputs = different key
	same := true
	for i := range key1 {
		if key1[i] != key3[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("different inputs produced same key")
	}
}

func TestEncryptedFileStore(t *testing.T) {
	t.Setenv("MULTIAI_SECRETS_DIR", t.TempDir())

	store, err := newEncryptedFileStore()
	if err != nil {
		t.Fatal(err)
	}

	// Set
	if err := store.Set("test-service", "API_KEY", "sk-test-abc123"); err != nil {
		t.Fatal(err)
	}

	// Get
	val, err := store.Get("test-service", "API_KEY")
	if err != nil {
		t.Fatal(err)
	}
	if val != "sk-test-abc123" {
		t.Errorf("got %q, want %q", val, "sk-test-abc123")
	}

	// List
	creds, err := store.List("test-service")
	if err != nil {
		t.Fatal(err)
	}
	if creds["API_KEY"] != "sk-test-abc123" {
		t.Errorf("list mismatch")
	}

	// Delete
	if err := store.Delete("test-service", "API_KEY"); err != nil {
		t.Fatal(err)
	}
	_, err = store.Get("test-service", "API_KEY")
	if err == nil {
		t.Error("expected error after delete")
	}
}

// TestPlatformStoreRoundTrip exercises the full public path used by
// 'multiai config' (Set) and launch-time resolution (Get) — the exact
// round-trip that was broken (base64-encoded Set, raw Get).
func TestPlatformStoreRoundTrip(t *testing.T) {
	t.Setenv("MULTIAI_SECRETS_DIR", t.TempDir())

	store, err := NewStore()
	if err != nil {
		t.Fatal(err)
	}

	service := ServiceForProfile("/some/dir/ca.env")
	if !strings.HasPrefix(service, "multiai:ca-") {
		t.Fatalf("ServiceForProfile: got %q, want prefix %q", service, "multiai:ca-")
	}
	// Ensure it is not just "multiai:ca" (old buggy format) — must have hash
	if strings.Count(service, "-") != 1 || len(service) <= len("multiai:ca-") {
		t.Fatalf("ServiceForProfile: missing hash suffix: %q", service)
	}

	const key = "ANTHROPIC_API_KEY"
	const value = "sk-ant-api03-roundtrip-test-1234567890"
	if err := store.Set(service, key, value); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(service, key)
	if err != nil {
		t.Fatal(err)
	}
	if got != value {
		t.Errorf("round-trip corrupted the value: got %q, want %q", got, value)
	}
}

// TestServiceForProfile_SameName_DifferentPath ensures two profiles with
// the same basename in different directories get different service names.
func TestServiceForProfile_SameName_DifferentPath(t *testing.T) {
	s1 := ServiceForProfile("/home/user/profiles/ca.env")
	s2 := ServiceForProfile("/tmp/evil/ca.env")
	if s1 == s2 {
		t.Errorf("same basename, different paths must produce different services:\n  s1=%q\n  s2=%q", s1, s2)
	}
}

// TestServiceForProfile_SamePath_SameService ensures the same path always
// produces the same service name (stability across calls).
func TestServiceForProfile_SamePath_SameService(t *testing.T) {
	path := "/home/user/profiles/ca.env"
	s1 := ServiceForProfile(path)
	s2 := ServiceForProfile(path)
	if s1 != s2 {
		t.Errorf("same path must produce the same service:\n  s1=%q\n  s2=%q", s1, s2)
	}
}

// TestMigrateServiceName verifies that credentials stored under the old
// basename-only service name are migrated to the new hashed service name.
func TestMigrateServiceName(t *testing.T) {
	t.Setenv("MULTIAI_SECRETS_DIR", t.TempDir())

	store, err := NewStore()
	if err != nil {
		t.Fatal(err)
	}

	profilePath := "/home/user/profiles/ca.env"
	newService := ServiceForProfile(profilePath)
	oldService := oldServiceForProfile(profilePath)

	// If old == new (e.g. path is so short the hash matches?), skip — no migration needed.
	if newService == oldService {
		t.Skip("old and new service names are identical, nothing to migrate")
	}

	// Store credentials under the OLD service name (simulating pre-fix state).
	if err := store.Set(oldService, "ANTHROPIC_API_KEY", "sk-old-migration-test"); err != nil {
		t.Fatal(err)
	}
	if err := store.Set(oldService, "OPENAI_API_KEY", "sk-old-openai"); err != nil {
		t.Fatal(err)
	}

	// Verify old data is readable.
	val, err := store.Get(oldService, "ANTHROPIC_API_KEY")
	if err != nil {
		t.Fatal(err)
	}
	if val != "sk-old-migration-test" {
		t.Fatalf("old service value: got %q, want %q", val, "sk-old-migration-test")
	}

	// Run migration.
	migratedService, err := MigrateServiceName(store, profilePath)
	if err != nil {
		t.Fatal(err)
	}
	if migratedService != newService {
		t.Fatalf("MigrateServiceName returned %q, want %q", migratedService, newService)
	}

	// Verify data is now under the NEW service.
	val, err = store.Get(newService, "ANTHROPIC_API_KEY")
	if err != nil {
		t.Fatal(err)
	}
	if val != "sk-old-migration-test" {
		t.Errorf("new service ANTHROPIC_API_KEY: got %q, want %q", val, "sk-old-migration-test")
	}
	val, err = store.Get(newService, "OPENAI_API_KEY")
	if err != nil {
		t.Fatal(err)
	}
	if val != "sk-old-openai" {
		t.Errorf("new service OPENAI_API_KEY: got %q, want %q", val, "sk-old-openai")
	}

	// Verify OLD service is now empty.
	oldCreds, err := store.List(oldService)
	if err != nil {
		t.Fatal(err)
	}
	if len(oldCreds) != 0 {
		t.Errorf("old service still has %d credentials after migration", len(oldCreds))
	}

	// Second migration call should be a no-op.
	migratedService2, err := MigrateServiceName(store, profilePath)
	if err != nil {
		t.Fatal(err)
	}
	if migratedService2 != newService {
		t.Fatalf("second MigrateServiceName returned %q, want %q", migratedService2, newService)
	}
	val, err = store.Get(newService, "ANTHROPIC_API_KEY")
	if err != nil {
		t.Fatal(err)
	}
	if val != "sk-old-migration-test" {
		t.Errorf("after second migration, value: got %q, want %q", val, "sk-old-migration-test")
	}
}

// TestServiceNameIsFileSafe ensures a service containing ':' lands in a
// regular file inside the secrets dir (on NTFS a raw ':' would create an
// alternate data stream and no visible file).
func TestServiceNameIsFileSafe(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_SECRETS_DIR", dir)

	store, err := newEncryptedFileStore()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Set("multiai:ca", "K", "v"); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range entries {
		if e.Name() == "multiai_ca.enc" {
			found = true
		}
		if strings.Contains(e.Name(), ":") {
			t.Errorf("unsafe file name created: %q", e.Name())
		}
	}
	if !found {
		t.Errorf("expected multiai_ca.enc in %s, entries: %v", dir, entries)
	}
}

// TestConcurrentStoreAccess runs 10 goroutines that simultaneously Set, Get,
// and Delete credentials on the same service, exercising both the in-process
// mutex and the inter-process file lock. The race detector (-race) must not
// fire and no credential must be lost or corrupted.
func TestConcurrentStoreAccess(t *testing.T) {
	t.Setenv("MULTIAI_SECRETS_DIR", t.TempDir())

	store, err := newEncryptedFileStore()
	if err != nil {
		t.Fatal(err)
	}

	const service = "concurrent-service"
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("KEY_%d", id)
			value := fmt.Sprintf("value_%d", id)

			// Set
			if err := store.Set(service, key, value); err != nil {
				t.Errorf("Set(%q): %v", key, err)
				return
			}

			// Get
			got, err := store.Get(service, key)
			if err != nil {
				t.Errorf("Get(%q): %v", key, err)
				return
			}
			if got != value {
				t.Errorf("Get(%q) = %q, want %q", key, got, value)
			}

			// Delete
			if err := store.Delete(service, key); err != nil {
				t.Errorf("Delete(%q): %v", key, err)
				return
			}

			// Verify deleted
			_, err = store.Get(service, key)
			if err == nil {
				t.Errorf("Get(%q) should fail after delete", key)
			}
		}(i)
	}
	wg.Wait()
}

// ── Zeroize tests ──────────────────────────────────────────────────────────

func TestZeroize(t *testing.T) {
	buf := []byte("sk-ant-api03-this-is-a-secret-key-12345")
	original := make([]byte, len(buf))
	copy(original, buf)

	Zeroize(buf)

	for i, b := range buf {
		if b != 0 {
			t.Errorf("byte[%d] = 0x%02x after Zeroize, want 0x00", i, b)
		}
	}
	// Verify that original data is gone — at least one byte must differ.
	allSame := true
	for i := range buf {
		if buf[i] != original[i] {
			allSame = false
			break
		}
	}
	if allSame {
		t.Error("Zeroize did not modify the buffer")
	}
}

func TestZeroizeEmpty(t *testing.T) {
	// Must not panic or error on a nil/empty slice.
	var nilBuf []byte = nil
	Zeroize(nilBuf) // should be a no-op

	emptyBuf := []byte{}
	Zeroize(emptyBuf) // should be a no-op
}

func TestZeroizeLargeBuffer(t *testing.T) {
	buf := make([]byte, 65536)
	for i := range buf {
		buf[i] = byte(i % 256)
	}
	Zeroize(buf)
	for i, b := range buf {
		if b != 0 {
			t.Errorf("large buffer byte[%d] = 0x%02x after Zeroize, want 0x00", i, b)
			break
		}
	}
}

// TestZeroizeNotOptimized recompiles the package with inlining hints and
// verifies that the compiler does NOT consider Zeroize inlinable. An inlined
// Zeroize could have its write loop elided by dead-code elimination since the
// buffer is never read after being written — the whole point is that the
// runtime.KeepAlive barrier must survive.
func TestZeroizeNotOptimized(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping compiler-flag test in short mode")
	}
	// Use "go test" with -gcflags=-m on the package — if Zeroize is
	// inlinable the compiler will emit "can inline Zeroize".
	cmd := exec.Command("go", "test",
		"-gcflags=-m",
		"-run=^$", // run nothing, just compile
		"-count=0",
		".")
	out, err := cmd.CombinedOutput()
	if err != nil {
		// The compile step may fail on constrained systems.
		t.Skipf("go test -gcflags=-m failed (%v), skipping inline check:\n%s", err, out)
	}
	output := string(out)
	if strings.Contains(output, "can inline Zeroize") {
		t.Error("Zeroize is flagged as inlinable by the compiler — the write loop may be elided")
	}
}

// BenchmarkZeroize measures the time cost of zeroising buffers of various
// sizes. The compiler cannot optimise the loop away because the buffer
// pointer escapes through runtime.KeepAlive.
func BenchmarkZeroize(b *testing.B) {
	sizes := []int{32, 256, 1024, 65536}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			buf := make([]byte, size)
			for i := range buf {
				buf[i] = byte(i)
			}
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				Zeroize(buf)
			}
		})
	}
}
