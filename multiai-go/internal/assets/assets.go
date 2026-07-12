// Package assets embeds the default profile templates so a freshly
// installed binary can materialize a working configuration on first run.
//
// The templates in profiles/ are sanitized copies of the launch profiles:
// every secret value is a placeholder (PASTE_..._HERE). Real keys are
// provided by the user via `multiai config` or by editing the extracted
// files.
package assets

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed profiles/*.env
var Profiles embed.FS

//go:embed profiles-manifest.json
var manifestData []byte

// ProfileManifest records the version and expected SHA256 sums of every
// embedded profile template. It is used to detect new profiles on upgrade
// and to warn when a user has modified a file.
type ProfileManifest struct {
	Version  string            `json:"version"`
	Profiles map[string]string `json:"profiles"` // filename -> "sha256:hex"
}

// ReadManifest parses the embedded profiles-manifest.json.
func ReadManifest() (*ProfileManifest, error) {
	var m ProfileManifest
	if err := json.Unmarshal(manifestData, &m); err != nil {
		return nil, fmt.Errorf("cannot parse embedded manifest: %w", err)
	}
	if m.Version == "" {
		return nil, fmt.Errorf("embedded manifest has empty version")
	}
	return &m, nil
}

// ManifestPath returns the path of the installed manifest file inside dir.
func ManifestPath(dir string) string {
	return filepath.Join(dir, ".profiles-manifest.json")
}

// ReadInstalledManifest reads the manifest previously written to dir.
// It returns nil, nil when the file does not exist (fresh install / legacy).
func ReadInstalledManifest(dir string) (*ProfileManifest, error) {
	p := ManifestPath(dir)
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var m ProfileManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("cannot parse installed manifest at %s: %w", p, err)
	}
	return &m, nil
}

// WriteManifest persists m to the installed manifest file inside dir.
func WriteManifest(dir string, m *ProfileManifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ManifestPath(dir), data, 0o600)
}

// ExtractProfiles materializes the embedded profile templates into destDir.
// The directory is created (0700) if needed, files are written with 0600.
// Existing files are never overwritten: user modifications take precedence.
// It returns the number of files actually written.
func ExtractProfiles(destDir string) (int, error) {
	if err := os.MkdirAll(destDir, 0o700); err != nil {
		return 0, fmt.Errorf("cannot create profiles directory %s: %w", destDir, err)
	}

	entries, err := fs.ReadDir(Profiles, "profiles")
	if err != nil {
		return 0, fmt.Errorf("cannot read embedded profiles: %w", err)
	}

	written := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		dest := filepath.Join(destDir, entry.Name())
		if _, err := os.Stat(dest); err == nil {
			continue // keep the user's version
		}
		data, err := Profiles.ReadFile("profiles/" + entry.Name())
		if err != nil {
			return written, fmt.Errorf("cannot read embedded profile %s: %w", entry.Name(), err)
		}
		if err := os.WriteFile(dest, data, 0o600); err != nil {
			return written, fmt.Errorf("cannot write profile %s: %w", dest, err)
		}
		written++
	}
	return written, nil
}

// ExtractMissingProfiles extracts only those embedded profiles that are not
// yet tracked in the installed manifest, or that were added in a newer
// version.  User-modified profiles (mismatched SHA) are warned about on
// stderr and skipped.  After extraction the full embedded manifest is saved
// to destDir so future calls know which profiles are current.
func ExtractMissingProfiles(destDir string, installed *ProfileManifest) (int, error) {
	embed, err := ReadManifest()
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(destDir, 0o700); err != nil {
		return 0, fmt.Errorf("cannot create profiles directory %s: %w", destDir, err)
	}

	written := 0
	for name, embedHash := range embed.Profiles {
		if installed != nil {
			if installedHash, known := installed.Profiles[name]; known {
				if installedHash == embedHash {
					continue // already installed and unchanged
				}
				// Profile exists but differs from embedded → user-modified.
				fmt.Fprintf(os.Stderr,
					"Avertissement: %s a ete modifie — conserver la version utilisateur\n", name)
				continue
			}
		}
		// Not tracked by the installed manifest → candidate for extraction.
		dest := filepath.Join(destDir, name)
		if _, err := os.Stat(dest); err == nil {
			// File exists on disk without being tracked in the manifest
			// (legacy install, or untracked file). Never overwrite.
			continue
		}
		data, err := Profiles.ReadFile("profiles/" + name)
		if err != nil {
			return written, fmt.Errorf("cannot read embedded profile %s: %w", name, err)
		}
		if err := os.WriteFile(dest, data, 0o600); err != nil {
			return written, fmt.Errorf("cannot write profile %s: %w", dest, err)
		}
		written++
	}

	// Save the full embedded manifest so future upgrades know what is current.
	if err := WriteManifest(destDir, embed); err != nil {
		return written, fmt.Errorf("cannot save manifest: %w", err)
	}

	return written, nil
}

// HashFile computes "sha256:<hex>" for the file at path.
func HashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(h[:]), nil
}
