package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lrochetta/multiai/internal/assets"
)

// snapshotStderr calls fn while capturing stderr, then returns the captured
// output.
func snapshotStderr(fn func()) string {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	fn()
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stderr = old
	return buf.String()
}

// TestEnsureProfiles_Upgrade simulates an existing installation with 30
// profiles and a manifest, then verifies that only the 7 new profiles are
// extracted after the embedded set grows.
func TestEnsureProfiles_Upgrade(t *testing.T) {
	// Read the full embedded manifest to know what we are working with.
	embed, err := assets.ReadManifest()
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	const totalEmbedded = 37

	// Pick 30 profiles to simulate an "old" installation.
	profileNames := make([]string, 0, totalEmbedded)
	for name := range embed.Profiles {
		profileNames = append(profileNames, name)
	}
	if len(profileNames) != totalEmbedded {
		t.Fatalf("expected %d embedded profiles, got %d", totalEmbedded, len(profileNames))
	}
	oldSet := profileNames[:30]
	newSet := profileNames[30:] // 7 profiles not yet installed

	dir := t.TempDir()

	// Copy the old profiles into the temp dir (no manifest yet).
	for _, name := range oldSet {
		data, err := assets.Profiles.ReadFile("profiles/" + name)
		if err != nil {
			t.Fatalf("read embedded %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	// Write an installed manifest that only knows about the old 30 profiles.
	oldManifest := &assets.ProfileManifest{
		Version:  "0.4.3",
		Profiles: make(map[string]string, len(oldSet)),
	}
	for _, name := range oldSet {
		oldManifest.Profiles[name] = embed.Profiles[name]
	}
	if err := assets.WriteManifest(dir, oldManifest); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}

	// --- ensureProfiles runs the upgrade logic ---
	ensureProfiles(dir)

	// Verify all 37 .env files now exist on disk.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	envCount := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".env") {
			envCount++
		}
	}
	if envCount != totalEmbedded {
		t.Errorf("expected %d .env files after upgrade, got %d", totalEmbedded, envCount)
	}

	// Verify the 7 new profiles were actually written.
	for _, name := range newSet {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Errorf("new profile %s not found: %v", name, err)
			continue
		}
		want, err := assets.Profiles.ReadFile("profiles/" + name)
		if err != nil {
			t.Fatalf("cannot read embedded %s: %v", name, err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s content differs from embedded", name)
		}
	}

	// Verify the 30 old profiles were NOT modified.
	for _, name := range oldSet {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Errorf("old profile %s missing: %v", name, err)
			continue
		}
		want, err := assets.Profiles.ReadFile("profiles/" + name)
		if err != nil {
			t.Fatalf("cannot read embedded %s: %v", name, err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s was unexpectedly modified", name)
		}
	}

	// Verify the installed manifest was updated to cover all 37 profiles.
	installed, err := assets.ReadInstalledManifest(dir)
	if err != nil {
		t.Fatalf("ReadInstalledManifest after upgrade: %v", err)
	}
	if len(installed.Profiles) != totalEmbedded {
		t.Errorf("installed manifest has %d entries, want %d", len(installed.Profiles), totalEmbedded)
	}
}

// TestEnsureProfiles_UserModified verifies that a user-modified profile is
// not overwritten.  We simulate what happens after a binary upgrade where
// the embedded manifest has a different SHA for a profile than the one
// stored in the installed manifest — indicating the template changed (and
// the user may have modified the file too).
func TestEnsureProfiles_UserModified(t *testing.T) {
	embed, err := assets.ReadManifest()
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}

	dir := t.TempDir()

	// Write ALL profiles from the embedded set.
	for name := range embed.Profiles {
		data, err := assets.Profiles.ReadFile("profiles/" + name)
		if err != nil {
			t.Fatalf("read embedded %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	// Write an installed manifest matching the embedded hash for every
	// profile, simulating a completed first install.
	fullManifest := &assets.ProfileManifest{
		Version:  embed.Version,
		Profiles: make(map[string]string, len(embed.Profiles)),
	}
	for name, hash := range embed.Profiles {
		fullManifest.Profiles[name] = hash
	}
	if err := assets.WriteManifest(dir, fullManifest); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}

	// "User" modifies one profile.
	modifiedName := "10-claude-anthropic-api.env"
	userContent := []byte("# User-modified profile\nPROFILE_ID=user-edit\n")
	if err := os.WriteFile(filepath.Join(dir, modifiedName), userContent, 0o600); err != nil {
		t.Fatalf("write user modification: %v", err)
	}

	// Simulate an upgrade: the installed manifest now has a DIFFERENT hash
	// for the modified profile than the embedded one (as would happen when
	// the binary is updated with a new template version).
	fullManifest.Profiles[modifiedName] = "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	if err := assets.WriteManifest(dir, fullManifest); err != nil {
		t.Fatalf("WriteManifest with modified hash: %v", err)
	}

	// --- ensureProfiles runs ---
	stderr := snapshotStderr(func() {
		ensureProfiles(dir)
	})

	// Verify the modified file was NOT overwritten.
	got, err := os.ReadFile(filepath.Join(dir, modifiedName))
	if err != nil {
		t.Fatalf("read modified file: %v", err)
	}
	if !bytes.Equal(got, userContent) {
		t.Errorf("user-modified profile was overwritten; got content:\n%s", got)
	}

	// Verify a warning was printed mentioning the file.
	if !strings.Contains(stderr, modifiedName) {
		t.Errorf("expected stderr warning mentioning %q, got:\n%s", modifiedName, stderr)
	}
}

// --- Tests pour --store flag (S2.3) ---

func TestHandleStoreFlag_NoFlag(t *testing.T) {
	msg, err := handleStoreFlag([]string{"multiai", "config"})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if msg != "" {
		t.Errorf("expected empty message, got: %q", msg)
	}
}

func TestHandleStoreFlag_InvalidBackend(t *testing.T) {
	msg, err := handleStoreFlag([]string{"multiai", "config", "--store", "not-a-real-backend"})
	if err == nil {
		t.Fatal("expected an error for invalid backend")
	}
	if msg != "" {
		t.Errorf("expected empty message on error, got: %q", msg)
	}
	if !strings.Contains(err.Error(), "invalide") {
		t.Errorf("error should mention 'invalide', got: %v", err)
	}
}

func TestHandleStoreFlag_ValidBackend(t *testing.T) {
	backends := []string{"keychain", "wincred", "secret-service"}
	for _, backend := range backends {
		t.Run(backend, func(t *testing.T) {
			msg, err := handleStoreFlag([]string{"multiai", "config", "--store", backend})
			if err != nil {
				t.Errorf("expected no error for %s, got: %v", backend, err)
			}
			if !strings.Contains(msg, "Le backend natif '"+backend+"'") {
				t.Errorf("expected notice about %s in message, got: %q", backend, msg)
			}
			if !strings.Contains(msg, "n'est pas encore implemente") {
				t.Errorf("expected 'pas encore implemente' in message, got: %q", msg)
			}
			if !strings.Contains(msg, "AES-256-GCM") {
				t.Errorf("expected 'AES-256-GCM' in message, got: %q", msg)
			}
			if !strings.Contains(msg, "issues/42") {
				t.Errorf("expected 'issues/42' in message, got: %q", msg)
			}
		})
	}
}
