package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lrochetta/multiai/internal/catalog"
	"github.com/lrochetta/multiai/internal/profile"
	"github.com/lrochetta/multiai/internal/secret"
)

// newReader builds a scripted input reader for menu-driven tests.
func newReader(input string) *bufio.Reader {
	return bufio.NewReader(strings.NewReader(input))
}

// writeProfile creates a minimal profile .env on disk and returns the
// matching in-memory profile, the way profile.LoadDir would build it.
func writeProfile(t *testing.T, dir, shortcut, varName, value string) profile.Profile {
	t.Helper()
	path := filepath.Join(dir, shortcut+".env")
	content := fmt.Sprintf("PROFILE_ID=%s\nSHORTCUT=%s\nDISPLAY_NAME=%s\n%s=%s\n",
		shortcut, shortcut, shortcut, varName, value)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return profile.Profile{
		ID: shortcut, Shortcut: shortcut, DisplayName: shortcut,
		Env:  map[string]string{varName: value},
		Path: path,
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// ── updateEnvFile (Sprint 1 credential store flow, preserved) ───────────────

// TestUpdateEnvFile_SentinelInvariant checks the write side of the
// config→launch flow: after updateEnvFile, the profile file holds the
// sentinel AND the credential store holds the real value.
func TestUpdateEnvFile_SentinelInvariant(t *testing.T) {
	t.Setenv("MULTIAI_SECRETS_DIR", t.TempDir())

	dir := t.TempDir()
	profPath := filepath.Join(dir, "ca.env")
	content := "PROFILE_ID=ca\nSHORTCUT=ca\nANTHROPIC_API_KEY=PASTE_YOUR_KEY_HERE\n"
	if err := os.WriteFile(profPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	const value = "sk-ant-api03-wizard-test-1234567890"
	if err := updateEnvFile(profPath, "ANTHROPIC_API_KEY", value, false, nil); err != nil {
		t.Fatalf("updateEnvFile: %v", err)
	}

	// File must contain the sentinel, never the plaintext key.
	data := readFile(t, profPath)
	if strings.Contains(data, value) {
		t.Error("plaintext key leaked into the profile file")
	}
	if !strings.Contains(data, "ANTHROPIC_API_KEY="+secret.Sentinel) {
		t.Errorf("sentinel missing from profile file:\n%s", data)
	}

	// Store must return the real value for the same service name the
	// launcher will derive from the profile path.
	store, err := secret.NewStore()
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(secret.ServiceForProfile(profPath), "ANTHROPIC_API_KEY")
	if err != nil {
		t.Fatalf("store.Get: %v", err)
	}
	if got != value {
		t.Errorf("store value mismatch: got %q, want %q", got, value)
	}
}

func TestUpdateEnvFile_VariableNotFound(t *testing.T) {
	t.Setenv("MULTIAI_SECRETS_DIR", t.TempDir())

	dir := t.TempDir()
	profPath := filepath.Join(dir, "ca.env")
	if err := os.WriteFile(profPath, []byte("SHORTCUT=ca\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := updateEnvFile(profPath, "MISSING_VAR", "x", false, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(profPath)
	if !strings.Contains(string(data), "MISSING_VAR=") {
		t.Errorf("variable not appended:\n%s", string(data))
	}
}

func TestSetEnvVarInFile_SkipsComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "p.env")
	content := "# ANTHROPIC_API_KEY=commented\nANTHROPIC_API_KEY=old\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	if err := setEnvVarInFile(path, "ANTHROPIC_API_KEY", "new"); err != nil {
		t.Fatal(err)
	}
	got := readFile(t, path)
	if !strings.Contains(got, "# ANTHROPIC_API_KEY=commented") {
		t.Error("commented line was rewritten")
	}
	if !strings.Contains(got, "ANTHROPIC_API_KEY=new") {
		t.Errorf("value not replaced:\n%s", got)
	}
}

// ── catalog wiring ──────────────────────────────────────────────────────────

// TestDefaultProvidersComeFromCatalog guards against reintroducing a
// hardcoded provider list: config must reflect the embedded catalog.
func TestDefaultProvidersComeFromCatalog(t *testing.T) {
	provs := DefaultProviders()
	if len(provs) != len(catalog.Default().Providers) {
		t.Fatalf("DefaultProviders = %d entries, catalog has %d", len(provs), len(catalog.Default().Providers))
	}
	if provs[0].ID != "openrouter" {
		t.Errorf("first provider = %q, want openrouter (menu order)", provs[0].ID)
	}
}

func TestValidateAPIKey(t *testing.T) {
	noPattern := Provider{ID: "acme"}
	withPattern := Provider{ID: "acme", KeyPattern: "^sk-or-"}
	tests := []struct {
		name  string
		prov  Provider
		key   string
		valid bool
	}{
		{"placeholder", noPattern, "PASTE_YOUR_KEY_HERE", false},
		{"empty-ish placeholder", noPattern, "   ", false},
		{"too short", noPattern, "sk-12", false},
		{"no pattern accepts", noPattern, "any-long-enough-key", true},
		{"pattern match", withPattern, "sk-or-abcdef123456", true},
		{"pattern mismatch", withPattern, "sk-ant-abcdef123456", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, msg := validateAPIKey(tt.prov, tt.key)
			if valid != tt.valid {
				t.Errorf("validateAPIKey(%q) = %v (%s), want %v", tt.key, valid, msg, tt.valid)
			}
		})
	}
}

// ── ConfigureProviderByID (contract consumed by cmd/multiai) ────────────────

func TestConfigureProviderByID_UnknownProvider(t *testing.T) {
	err := ConfigureProviderByID(nil, "doesnotexist", newReader(""), nil)
	if err == nil {
		t.Fatal("expected error for unknown provider id")
	}
	if !strings.Contains(err.Error(), "doesnotexist") {
		t.Errorf("error should name the bad id: %v", err)
	}
	if !strings.Contains(err.Error(), "openrouter") {
		t.Errorf("error should list valid ids: %v", err)
	}
}

// TestConfigureProviderByID_PropagatesToWholeGroup checks the core catalog
// semantic: one key, written to every installed profile of the group, with
// the store-first sentinel flow.
func TestConfigureProviderByID_PropagatesToWholeGroup(t *testing.T) {
	t.Setenv("MULTIAI_SECRETS_DIR", t.TempDir())
	dir := t.TempDir()

	// Two profiles of the anthropic group (ca, ocanthropic).
	profiles := []profile.Profile{
		writeProfile(t, dir, "ca", "ANTHROPIC_API_KEY", "PASTE_ANTHROPIC_API_KEY_HERE"),
		writeProfile(t, dir, "ocanthropic", "ANTHROPIC_API_KEY", "PASTE_ANTHROPIC_API_KEY_HERE"),
	}

	const key = "sk-ant-api03-group-propagation-test-123"
	if err := ConfigureProviderByID(profiles, "anthropic", newReader(key+"\n"), nil); err != nil {
		t.Fatalf("ConfigureProviderByID: %v", err)
	}

	store, err := secret.NewStore()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range profiles {
		data := readFile(t, p.Path)
		if strings.Contains(data, key) {
			t.Errorf("%s: plaintext key leaked into file", p.Shortcut)
		}
		if !strings.Contains(data, "ANTHROPIC_API_KEY="+secret.Sentinel) {
			t.Errorf("%s: sentinel missing:\n%s", p.Shortcut, data)
		}
		got, err := store.Get(secret.ServiceForProfile(p.Path), "ANTHROPIC_API_KEY")
		if err != nil || got != key {
			t.Errorf("%s: store value = %q, err = %v", p.Shortcut, got, err)
		}
	}
}

func TestConfigureProviderByID_EmptyInputIgnores(t *testing.T) {
	t.Setenv("MULTIAI_SECRETS_DIR", t.TempDir())
	dir := t.TempDir()
	profiles := []profile.Profile{
		writeProfile(t, dir, "ca", "ANTHROPIC_API_KEY", "PASTE_ANTHROPIC_API_KEY_HERE"),
	}

	if err := ConfigureProviderByID(profiles, "anthropic", newReader("\n"), nil); err != nil {
		t.Fatalf("ConfigureProviderByID: %v", err)
	}
	if got := readFile(t, profiles[0].Path); !strings.Contains(got, "ANTHROPIC_API_KEY=PASTE_ANTHROPIC_API_KEY_HERE") {
		t.Errorf("empty input must not modify the file:\n%s", got)
	}
}

// TestConfigureProviderByID_ShortKeyDeclined: an invalid key triggers the
// advisory warning; answering "n" aborts without writing.
func TestConfigureProviderByID_ShortKeyDeclined(t *testing.T) {
	t.Setenv("MULTIAI_SECRETS_DIR", t.TempDir())
	dir := t.TempDir()
	profiles := []profile.Profile{
		writeProfile(t, dir, "ca", "ANTHROPIC_API_KEY", "PASTE_ANTHROPIC_API_KEY_HERE"),
	}

	if err := ConfigureProviderByID(profiles, "anthropic", newReader("short\nn\n"), nil); err != nil {
		t.Fatalf("ConfigureProviderByID: %v", err)
	}
	if got := readFile(t, profiles[0].Path); !strings.Contains(got, "PASTE_ANTHROPIC_API_KEY_HERE") {
		t.Errorf("declined key must not be written:\n%s", got)
	}
}

// TestConfigureProviderByID_ShortKeyForced: answering "o" writes anyway
// (PS parity: no blocking validation).
func TestConfigureProviderByID_ShortKeyForced(t *testing.T) {
	t.Setenv("MULTIAI_SECRETS_DIR", t.TempDir())
	dir := t.TempDir()
	profiles := []profile.Profile{
		writeProfile(t, dir, "ca", "ANTHROPIC_API_KEY", "PASTE_ANTHROPIC_API_KEY_HERE"),
	}

	if err := ConfigureProviderByID(profiles, "anthropic", newReader("short\no\n"), nil); err != nil {
		t.Fatalf("ConfigureProviderByID: %v", err)
	}
	if got := readFile(t, profiles[0].Path); !strings.Contains(got, "ANTHROPIC_API_KEY="+secret.Sentinel) {
		t.Errorf("forced key should be stored (sentinel in file):\n%s", got)
	}
}

func TestConfigureProviderByID_CaseInsensitiveID(t *testing.T) {
	t.Setenv("MULTIAI_SECRETS_DIR", t.TempDir())
	// No profiles installed: the provider prompt prints a warning and returns.
	if err := ConfigureProviderByID(nil, "ANTHROPIC", newReader(""), nil); err != nil {
		t.Fatalf("uppercase id should resolve: %v", err)
	}
}

// ── menu loop ───────────────────────────────────────────────────────────────

// TestRunConfigMenu_ConfigureByNumber drives the real menu: pick anthropic
// (entry 12), enter a key, then exit.
func TestRunConfigMenu_ConfigureByNumber(t *testing.T) {
	t.Setenv("MULTIAI_SECRETS_DIR", t.TempDir())
	dir := t.TempDir()
	prof := writeProfile(t, dir, "ca", "ANTHROPIC_API_KEY", "PASTE_ANTHROPIC_API_KEY_HERE")
	byShortcut := shortcutIndex([]profile.Profile{prof})

	const key = "sk-ant-api03-menu-flow-test-99887766"
	input := "12\n" + key + "\n0\n"
	if err := runConfigMenu(catalog.Default(), byShortcut, newReader(input), nil); err != nil {
		t.Fatalf("runConfigMenu: %v", err)
	}

	if got := readFile(t, prof.Path); !strings.Contains(got, "ANTHROPIC_API_KEY="+secret.Sentinel) {
		t.Errorf("key not persisted via menu:\n%s", got)
	}
	// In-memory profile must be synced for the next menu display.
	if byShortcut["ca"].Env["ANTHROPIC_API_KEY"] != key {
		t.Error("in-memory profile not updated")
	}
}

// TestRunConfigMenu_InvalidChoiceThenExit: bad input warns and loops.
func TestRunConfigMenu_InvalidChoiceThenExit(t *testing.T) {
	t.Setenv("MULTIAI_SECRETS_DIR", t.TempDir())
	if err := runConfigMenu(catalog.Default(), nil, newReader("zz\n99\n0\n"), nil); err != nil {
		t.Fatalf("runConfigMenu: %v", err)
	}
}

func TestRunConfigMenu_EOFReturns(t *testing.T) {
	t.Setenv("MULTIAI_SECRETS_DIR", t.TempDir())
	if err := runConfigMenu(catalog.Default(), nil, newReader(""), nil); err != nil {
		t.Fatalf("runConfigMenu on EOF: %v", err)
	}
}

// TestRunConfigMenu_AllSequenceWithEmptyInput: "a" walks every provider;
// empty answers skip them all; the final launch prompt consumes one line.
func TestRunConfigMenu_AllSequenceWithEmptyInput(t *testing.T) {
	t.Setenv("MULTIAI_SECRETS_DIR", t.TempDir())
	dir := t.TempDir()
	prof := writeProfile(t, dir, "ca", "ANTHROPIC_API_KEY", "PASTE_ANTHROPIC_API_KEY_HERE")
	byShortcut := shortcutIndex([]profile.Profile{prof})

	// 13 empty answers (one per provider; only anthropic has a profile but
	// every provider prompts or warns) + "n" for the launch prompt.
	input := "a\n" + strings.Repeat("\n", 13) + "n\n"
	if err := runConfigMenu(catalog.Default(), byShortcut, newReader(input), nil); err != nil {
		t.Fatalf("runConfigMenu: %v", err)
	}
	if got := readFile(t, prof.Path); !strings.Contains(got, "PASTE_ANTHROPIC_API_KEY_HERE") {
		t.Errorf("empty inputs must leave placeholders untouched:\n%s", got)
	}
}

// ── erase keys ──────────────────────────────────────────────────────────────

// configureGroup seeds a configured provider group: sentinel in files,
// value in store.
func configureGroup(t *testing.T, dir string, providerID string, key string, shortcuts ...string) ([]profile.Profile, Provider) {
	t.Helper()
	prov, ok := catalog.Default().ProviderByID(providerID)
	if !ok {
		t.Fatalf("provider %q not in catalog", providerID)
	}
	var profiles []profile.Profile
	for _, sc := range shortcuts {
		varName := prov.VarMap[sc]
		if varName == "" {
			t.Fatalf("shortcut %q not in provider %q", sc, providerID)
		}
		profiles = append(profiles, writeProfile(t, dir, sc, varName, "PASTE_"+varName+"_HERE"))
	}
	if err := ConfigureProviderByID(profiles, providerID, newReader(key+"\n"), nil); err != nil {
		t.Fatal(err)
	}
	return profiles, prov
}

// TestEraseProviderKeys_ResetsFilesAndStore is the core erase contract:
// placeholder back in the .env AND credential gone from the store.
func TestEraseProviderKeys_ResetsFilesAndStore(t *testing.T) {
	t.Setenv("MULTIAI_SECRETS_DIR", t.TempDir())
	dir := t.TempDir()
	const key = "sk-ant-api03-erase-test-1234567890"
	profiles, prov := configureGroup(t, dir, "anthropic", key, "ca", "ocanthropic")
	byShortcut := shortcutIndex(profiles)

	if n := EraseProviderKeys(prov, byShortcut, nil); n != 2 {
		t.Fatalf("erased = %d, want 2", n)
	}

	store, err := secret.NewStore()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range profiles {
		data := readFile(t, p.Path)
		if !strings.Contains(data, "ANTHROPIC_API_KEY=PASTE_ANTHROPIC_API_KEY_HERE") {
			t.Errorf("%s: placeholder not restored:\n%s", p.Shortcut, data)
		}
		if strings.Contains(data, secret.Sentinel) {
			t.Errorf("%s: sentinel left behind after erase", p.Shortcut)
		}
		if _, err := store.Get(secret.ServiceForProfile(p.Path), "ANTHROPIC_API_KEY"); err == nil {
			t.Errorf("%s: credential still in store after erase", p.Shortcut)
		}
		if byShortcut[p.Shortcut].Env["ANTHROPIC_API_KEY"] != "PASTE_ANTHROPIC_API_KEY_HERE" {
			t.Errorf("%s: in-memory env not reset", p.Shortcut)
		}
	}
}

func TestEraseProviderKeys_SkipsMissingProfiles(t *testing.T) {
	t.Setenv("MULTIAI_SECRETS_DIR", t.TempDir())
	prov, _ := catalog.Default().ProviderByID("anthropic")
	if n := EraseProviderKeys(prov, map[string]*profile.Profile{}, nil); n != 0 {
		t.Errorf("erased = %d, want 0 with no installed profiles", n)
	}
}

// TestEraseProviderKeys_UnconfiguredCreatesNoStoreEntry: erasing a profile
// that never held a key must not create credential files as a side effect.
func TestEraseProviderKeys_UnconfiguredCreatesNoStoreEntry(t *testing.T) {
	secretsDir := t.TempDir()
	t.Setenv("MULTIAI_SECRETS_DIR", secretsDir)
	dir := t.TempDir()
	prof := writeProfile(t, dir, "ca", "ANTHROPIC_API_KEY", "PASTE_ANTHROPIC_API_KEY_HERE")
	prov, _ := catalog.Default().ProviderByID("anthropic")

	if n := EraseProviderKeys(prov, shortcutIndex([]profile.Profile{prof}), nil); n != 1 {
		t.Fatalf("erased = %d, want 1 (placeholder rewrite still counts)", n)
	}
	entries, err := filepath.Glob(filepath.Join(secretsDir, "*.enc"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("erase created store entries for an unconfigured profile: %v", entries)
	}
}

// TestEraseProviderKeys_StoreUnavailableStillCleansFiles: when the credential
// store cannot open (secrets dir is a file), erase must degrade to file-only
// cleanup without panicking (NewStore can return a typed-nil Store with the
// error).
func TestEraseProviderKeys_StoreUnavailableStillCleansFiles(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MULTIAI_SECRETS_DIR", filepath.Join(blocker, "sub"))

	dir := t.TempDir()
	prof := writeProfile(t, dir, "ca", "ANTHROPIC_API_KEY", "sk-ant-plaintext-erase-me-123456")
	prov, _ := catalog.Default().ProviderByID("anthropic")
	byShortcut := shortcutIndex([]profile.Profile{prof})

	if n := EraseProviderKeys(prov, byShortcut, nil); n != 1 {
		t.Fatalf("erased = %d, want 1", n)
	}
	if got := readFile(t, prof.Path); !strings.Contains(got, "ANTHROPIC_API_KEY=PASTE_ANTHROPIC_API_KEY_HERE") {
		t.Errorf("file not cleaned despite store failure:\n%s", got)
	}
}

// TestRunEraseMenu_SingleProviderWithConfirmation drives the menu path:
// pick provider 12 (anthropic), confirm with "oui".
func TestRunEraseMenu_SingleProviderWithConfirmation(t *testing.T) {
	t.Setenv("MULTIAI_SECRETS_DIR", t.TempDir())
	dir := t.TempDir()
	const key = "sk-ant-api03-erase-menu-test-123456"
	profiles, _ := configureGroup(t, dir, "anthropic", key, "ca")
	byShortcut := shortcutIndex(profiles)

	runEraseMenu(catalog.Default(), byShortcut, newReader("12\noui\n"), nil)

	if got := readFile(t, profiles[0].Path); !strings.Contains(got, "PASTE_ANTHROPIC_API_KEY_HERE") {
		t.Errorf("key not erased via menu:\n%s", got)
	}
}

// TestRunEraseMenu_RefusedConfirmationKeepsKeys: anything but "oui" aborts.
func TestRunEraseMenu_RefusedConfirmationKeepsKeys(t *testing.T) {
	t.Setenv("MULTIAI_SECRETS_DIR", t.TempDir())
	dir := t.TempDir()
	const key = "sk-ant-api03-erase-refuse-test-1234"
	profiles, _ := configureGroup(t, dir, "anthropic", key, "ca")
	byShortcut := shortcutIndex(profiles)

	runEraseMenu(catalog.Default(), byShortcut, newReader("12\nnon\n"), nil)

	if got := readFile(t, profiles[0].Path); !strings.Contains(got, "ANTHROPIC_API_KEY="+secret.Sentinel) {
		t.Errorf("refused confirmation must keep the key:\n%s", got)
	}
	store, err := secret.NewStore()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(secret.ServiceForProfile(profiles[0].Path), "ANTHROPIC_API_KEY"); err != nil {
		t.Error("credential must survive a refused erase")
	}
}

// TestRunEraseMenu_EraseAll erases every configured provider group at once.
func TestRunEraseMenu_EraseAll(t *testing.T) {
	t.Setenv("MULTIAI_SECRETS_DIR", t.TempDir())
	dir := t.TempDir()
	anthProfiles, _ := configureGroup(t, dir, "anthropic", "sk-ant-api03-eraseall-test-123456", "ca")
	orProfiles, _ := configureGroup(t, dir, "openrouter", "sk-or-eraseall-test-123456789", "or-fusion", "ocqwen")

	all := append(append([]profile.Profile{}, anthProfiles...), orProfiles...)
	byShortcut := shortcutIndex(all)

	runEraseMenu(catalog.Default(), byShortcut, newReader("a\noui\n"), nil)

	for _, p := range all {
		data := readFile(t, p.Path)
		if strings.Contains(data, secret.Sentinel) {
			t.Errorf("%s: still configured after erase all:\n%s", p.Shortcut, data)
		}
		if !strings.Contains(data, "_HERE") {
			t.Errorf("%s: placeholder missing after erase all:\n%s", p.Shortcut, data)
		}
	}
}

// TestRunConfigMenu_EraseFlowEndToEnd goes through the config menu ("e"),
// the erase menu, the confirmation, and back.
func TestRunConfigMenu_EraseFlowEndToEnd(t *testing.T) {
	t.Setenv("MULTIAI_SECRETS_DIR", t.TempDir())
	dir := t.TempDir()
	profiles, _ := configureGroup(t, dir, "openrouter", "sk-or-config-erase-e2e-123456", "or-fusion")
	byShortcut := shortcutIndex(profiles)

	// e -> erase menu; 1 -> openrouter; oui -> confirm; \n -> "Entree pour
	// revenir"; 0 -> exit config menu.
	input := "e\n1\noui\n\n0\n"
	if err := runConfigMenu(catalog.Default(), byShortcut, newReader(input), nil); err != nil {
		t.Fatalf("runConfigMenu: %v", err)
	}
	if got := readFile(t, profiles[0].Path); !strings.Contains(got, "PASTE_OPENROUTER_API_KEY_HERE") {
		t.Errorf("erase via config menu failed:\n%s", got)
	}
}
