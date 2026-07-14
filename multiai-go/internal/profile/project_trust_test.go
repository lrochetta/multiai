package profile

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const testProjectYAML = "extends: base\noverrides:\n  MULTIAI_TEST: enabled\n"

func writeProjectConfig(t *testing.T, dir, content string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, ".multiai.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func newTestProjectTrustStore(t *testing.T) *ProjectTrustStore {
	t.Helper()
	return NewProjectTrustStore(filepath.Join(t.TempDir(), "config", "multiai", "trusted-projects.json"))
}

func TestProjectTrustCanonicalPathAndFingerprint(t *testing.T) {
	root := t.TempDir()
	path := writeProjectConfig(t, root, testProjectYAML)
	store := newTestProjectTrustStore(t)

	status, err := store.Inspect(filepath.Join(root, "unused", "..", filepath.Base(path)))
	if err != nil {
		t.Fatal(err)
	}
	wantPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	wantPath, err = filepath.Abs(wantPath)
	if err != nil {
		t.Fatal(err)
	}
	if status.CanonicalPath != filepath.Clean(wantPath) {
		t.Fatalf("canonical path = %q, want %q", status.CanonicalPath, filepath.Clean(wantPath))
	}
	digest := sha256.Sum256([]byte(testProjectYAML))
	if status.Fingerprint != hex.EncodeToString(digest[:]) {
		t.Fatalf("fingerprint = %q, want %q", status.Fingerprint, hex.EncodeToString(digest[:]))
	}
	if status.State != ProjectTrustUntrusted {
		t.Fatalf("state = %q, want %q", status.State, ProjectTrustUntrusted)
	}
	if err := store.Check(path); !errors.Is(err, ErrProjectConfigUntrusted) {
		t.Fatalf("Check error = %v, want ErrProjectConfigUntrusted", err)
	}
}

func TestProjectTrustFingerprintChangeInvalidatesApproval(t *testing.T) {
	path := writeProjectConfig(t, t.TempDir(), testProjectYAML)
	store := newTestProjectTrustStore(t)

	trusted, err := store.Trust(path)
	if err != nil {
		t.Fatal(err)
	}
	if !trusted.Trusted() {
		t.Fatalf("state after Trust = %q, want trusted", trusted.State)
	}
	if err := store.Check(path); err != nil {
		t.Fatalf("Check trusted config: %v", err)
	}

	if err := os.WriteFile(path, []byte(testProjectYAML+"args: [--dangerous]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := store.Inspect(path)
	if err != nil {
		t.Fatal(err)
	}
	if changed.State != ProjectTrustChanged {
		t.Fatalf("state after modification = %q, want %q", changed.State, ProjectTrustChanged)
	}
	if changed.Fingerprint == changed.TrustedFingerprint {
		t.Fatal("modified config kept the approved fingerprint")
	}
	if err := store.Check(path); !errors.Is(err, ErrProjectConfigChanged) {
		t.Fatalf("Check error = %v, want ErrProjectConfigChanged", err)
	}
}

func TestProjectTrustSymlinkUsesTargetCanonicalPath(t *testing.T) {
	root := t.TempDir()
	target := writeProjectConfig(t, filepath.Join(root, "real"), testProjectYAML)
	aliasDir := filepath.Join(root, "alias")
	if err := os.MkdirAll(aliasDir, 0o755); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(aliasDir, ".multiai.yaml")
	if err := os.Symlink(target, alias); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}

	store := newTestProjectTrustStore(t)
	status, err := store.Trust(alias)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatal(err)
	}
	want, err = filepath.Abs(want)
	if err != nil {
		t.Fatal(err)
	}
	if status.CanonicalPath != filepath.Clean(want) {
		t.Fatalf("canonical path = %q, want target %q", status.CanonicalPath, filepath.Clean(want))
	}
	if err := store.Check(target); err != nil {
		t.Fatalf("target should share symlink approval: %v", err)
	}
}

func TestProjectTrustCorruptStoreFailsClosed(t *testing.T) {
	path := writeProjectConfig(t, t.TempDir(), testProjectYAML)
	store := newTestProjectTrustStore(t)
	if err := os.MkdirAll(filepath.Dir(store.Path()), 0o700); err != nil {
		t.Fatal(err)
	}
	original := []byte(`{"version":1,"projects":{},"unexpected":true}`)
	if err := os.WriteFile(store.Path(), original, 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := store.Inspect(path); !errors.Is(err, ErrProjectTrustStoreCorrupt) {
		t.Fatalf("Inspect error = %v, want ErrProjectTrustStoreCorrupt", err)
	}
	if err := store.Check(path); !errors.Is(err, ErrProjectTrustStoreCorrupt) {
		t.Fatalf("Check error = %v, want ErrProjectTrustStoreCorrupt", err)
	}
	if _, err := store.Trust(path); !errors.Is(err, ErrProjectTrustStoreCorrupt) {
		t.Fatalf("Trust error = %v, want ErrProjectTrustStoreCorrupt", err)
	}
	after, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(original) {
		t.Fatal("Trust overwrote a corrupt trust store")
	}
}

func TestProjectTrustAndUntrustAreIdempotent(t *testing.T) {
	path := writeProjectConfig(t, t.TempDir(), testProjectYAML)
	store := newTestProjectTrustStore(t)

	first, err := store.Trust(path)
	if err != nil {
		t.Fatal(err)
	}
	sentinel := time.Unix(1_700_000_000, 0)
	if err := os.Chtimes(store.Path(), sentinel, sentinel); err != nil {
		t.Fatal(err)
	}
	second, err := store.Trust(path)
	if err != nil {
		t.Fatal(err)
	}
	if first.TrustedAt != second.TrustedAt {
		t.Fatalf("idempotent Trust changed timestamp: %q -> %q", first.TrustedAt, second.TrustedAt)
	}
	info, err := os.Stat(store.Path())
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(sentinel) {
		t.Fatalf("idempotent Trust rewrote store: modtime = %s, want %s", info.ModTime(), sentinel)
	}

	if err := store.Untrust(path); err != nil {
		t.Fatal(err)
	}
	if err := store.Check(path); !errors.Is(err, ErrProjectConfigUntrusted) {
		t.Fatalf("Check after Untrust = %v, want ErrProjectConfigUntrusted", err)
	}
	if err := os.Chtimes(store.Path(), sentinel, sentinel); err != nil {
		t.Fatal(err)
	}
	if err := store.Untrust(path); err != nil {
		t.Fatal(err)
	}
	info, err = os.Stat(store.Path())
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(sentinel) {
		t.Fatalf("idempotent Untrust rewrote store: modtime = %s, want %s", info.ModTime(), sentinel)
	}
}

func TestProjectTrustStorePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose POSIX permission bits consistently")
	}
	path := writeProjectConfig(t, t.TempDir(), testProjectYAML)
	store := newTestProjectTrustStore(t)
	if _, err := store.Trust(path); err != nil {
		t.Fatal(err)
	}
	fileInfo, err := os.Stat(store.Path())
	if err != nil {
		t.Fatal(err)
	}
	if got := fileInfo.Mode().Perm(); got != 0o600 {
		t.Fatalf("store permissions = %o, want 600", got)
	}
	dirInfo, err := os.Stat(filepath.Dir(store.Path()))
	if err != nil {
		t.Fatal(err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Fatalf("store directory permissions = %o, want 700", got)
	}
}

func TestProjectTrustRejectsOversizedFiles(t *testing.T) {
	t.Run("project configuration", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), ".multiai.yaml")
		if err := os.WriteFile(path, make([]byte, maxTrustedProjectYAMLSize+1), 0o644); err != nil {
			t.Fatal(err)
		}
		store := newTestProjectTrustStore(t)
		if _, err := store.Inspect(path); err == nil || !strings.Contains(err.Error(), "exceeds") {
			t.Fatalf("Inspect oversized config error = %v, want size rejection", err)
		}
	})

	t.Run("trust store", func(t *testing.T) {
		store := newTestProjectTrustStore(t)
		if err := os.MkdirAll(filepath.Dir(store.Path()), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(store.Path(), make([]byte, maxProjectTrustStoreSize+1), 0o600); err != nil {
			t.Fatal(err)
		}
		path := writeProjectConfig(t, t.TempDir(), testProjectYAML)
		if _, err := store.Inspect(path); err == nil || !strings.Contains(err.Error(), "exceeds") {
			t.Fatalf("Inspect with oversized store error = %v, want size rejection", err)
		}
	})
}

func TestFindProjectConfigFailsClosedUntilTrusted(t *testing.T) {
	root := t.TempDir()
	path := writeProjectConfig(t, root, testProjectYAML)
	nested := filepath.Join(root, "nested", "deeper")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	store := newTestProjectTrustStore(t)

	config, foundPath, err := findProjectConfigFrom(nested, store)
	if config != nil || !errors.Is(err, ErrProjectConfigUntrusted) {
		t.Fatalf("untrusted discovery = (%v, %q, %v), want nil config and untrusted error", config, foundPath, err)
	}
	if foundPath == "" {
		t.Fatal("untrusted discovery did not report the canonical path")
	}

	if _, err := store.Trust(path); err != nil {
		t.Fatal(err)
	}
	config, foundPath, err = findProjectConfigFrom(nested, store)
	if err != nil {
		t.Fatal(err)
	}
	if config == nil || config.Extends != "base" || config.Overrides["MULTIAI_TEST"] != "enabled" {
		t.Fatalf("trusted config was not parsed: %#v", config)
	}
	if foundPath == "" || !filepath.IsAbs(foundPath) {
		t.Fatalf("trusted discovery path = %q, want absolute canonical path", foundPath)
	}

	if err := os.WriteFile(path, []byte(testProjectYAML+"clear_env: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	config, _, err = findProjectConfigFrom(nested, store)
	if config != nil || !errors.Is(err, ErrProjectConfigChanged) {
		t.Fatalf("changed discovery = (%v, %v), want nil config and changed error", config, err)
	}
}
