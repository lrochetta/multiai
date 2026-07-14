package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/lrochetta/multiai/internal/profile"
	"github.com/lrochetta/multiai/internal/secret"
)

func useIsolatedSecretStore(t *testing.T) {
	t.Helper()
	t.Setenv("MULTIAI_SECRETS_DIR", t.TempDir())
	original := newSecretStore
	newSecretStore = func() (secret.Store, error) {
		return secret.NewStoreWithBackend("file")
	}
	t.Cleanup(func() { newSecretStore = original })
}

// TestResolveStoredSecrets covers the config→launch flow that was broken:
// a profile whose .env carries the credential-store sentinel must launch
// with the real value, never with the literal sentinel.
func TestResolveStoredSecrets(t *testing.T) {
	useIsolatedSecretStore(t)

	const key = "ANTHROPIC_API_KEY"
	const value = "sk-ant-api03-resolve-test-1234567890"
	profPath := filepath.Join(t.TempDir(), "ca.env")

	store, err := newSecretStore()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Set(secret.ServiceForProfile(profPath), key, value); err != nil {
		t.Fatal(err)
	}

	prof := &profile.Profile{
		Shortcut: "ca",
		Path:     profPath,
		Env:      map[string]string{key: secret.Sentinel, "OTHER": "untouched"},
	}

	if err := resolveStoredSecrets(prof); err != nil {
		t.Fatalf("resolveStoredSecrets: %v", err)
	}
	if prof.Env[key] != value {
		t.Errorf("sentinel not resolved: got %q, want %q", prof.Env[key], value)
	}
	if prof.Env["OTHER"] != "untouched" {
		t.Errorf("non-sentinel value was modified: %q", prof.Env["OTHER"])
	}
}

func TestResolveStoredSecrets_MissingFromStore(t *testing.T) {
	useIsolatedSecretStore(t)
	profPath := filepath.Join(t.TempDir(), "ca.env")

	prof := &profile.Profile{
		Shortcut: "ca",
		Path:     profPath,
		Env:      map[string]string{"ANTHROPIC_API_KEY": secret.Sentinel},
	}

	err := resolveStoredSecrets(prof)
	if err == nil {
		t.Fatal("expected an error for a dangling sentinel, got nil")
	}
	if !strings.Contains(err.Error(), "multiai config") {
		t.Errorf("error should tell the user to re-run 'multiai config', got: %v", err)
	}
	if prof.Env["ANTHROPIC_API_KEY"] != secret.Sentinel {
		t.Errorf("env mutated despite error: %q", prof.Env["ANTHROPIC_API_KEY"])
	}
}

func TestResolveStoredSecrets_NoSentinel(t *testing.T) {
	// No store dir override on purpose: with no sentinel present the store
	// must not even be touched.
	prof := &profile.Profile{
		Shortcut: "ds",
		Path:     "/profiles/ds.env",
		Env:      map[string]string{"ANTHROPIC_AUTH_TOKEN": "sk-real-key-in-file-1234"},
	}
	if err := resolveStoredSecrets(prof); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prof.Env["ANTHROPIC_AUTH_TOKEN"] != "sk-real-key-in-file-1234" {
		t.Error("plain value must pass through unchanged")
	}
}
