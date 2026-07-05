// Package assets embeds the default profile templates so a freshly
// installed binary can materialize a working configuration on first run.
//
// The templates in profiles/ are sanitized copies of the launch profiles:
// every secret value is a placeholder (PASTE_..._HERE). Real keys are
// provided by the user via `multiai config` or by editing the extracted
// files.
package assets

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed profiles/*.env
var Profiles embed.FS

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
