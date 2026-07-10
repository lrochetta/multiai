// Package powershell detects and migrates a PowerShell legacy installation
// of multiai (the frozen multiai-powershell/ reference implementation).
//
// The PS version stores its profiles as .env files in configs/profiles/
// alongside a code-router.ps1 entry point and a package.json manifest.
// Profile format is identical to the Go version, so migration is primarily
// a file copy with backup and reporting.
package powershell

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DetectResult describes a legacy PowerShell installation found on disk.
type DetectResult struct {
	// RootDir is the detected installation root (the directory containing
	// code-router.ps1 or multiai.cmd).
	RootDir string `json:"root_dir"`
	// ProfilesDir is the resolved path to configs/profiles/.
	ProfilesDir string `json:"profiles_dir"`
	// Version is the version string from package.json, or "" if unknown.
	Version string `json:"version"`
	// ProfileCount is the number of .env files found.
	ProfileCount int `json:"profile_count"`
	// ProfileNames lists the .env filenames found (base names only).
	ProfileNames []string `json:"profile_names"`
}

// Detect looks for a PowerShell legacy installation at the given paths or
// in well-known locations. It returns nil (not an error) when nothing is
// found.
func Detect(searchDirs ...string) (*DetectResult, error) {
	candidates := collectCandidates(searchDirs)

	for _, dir := range candidates {
		result, err := probeDir(dir)
		if err != nil {
			return nil, err
		}
		if result != nil {
			return result, nil
		}
	}
	return nil, nil
}

// collectCandidates gathers directories to probe, combining user-supplied
// search dirs with well-known default locations.
func collectCandidates(extra []string) []string {
	seen := map[string]bool{}
	var dirs []string

	add := func(d string) {
		if d == "" || seen[d] {
			return
		}
		seen[d] = true
		dirs = append(dirs, d)
	}

	// 1. Extra search dirs from the caller (e.g. --from-ps flag).
	for _, d := range extra {
		add(d)
	}

	// 2. Common npm global installation paths (Windows).
	if appData := os.Getenv("APPDATA"); appData != "" {
		add(filepath.Join(appData, "npm", "node_modules", "multiai"))
		add(filepath.Join(appData, "npm", "node_modules", "multiai-powershell"))
	}
	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		add(filepath.Join(localAppData, "multiai"))
	}

	// 3. Executable-relative (npm global on Unix or portable installs).
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		add(filepath.Join(exeDir, "..", "lib", "node_modules", "multiai"))
		add(filepath.Dir(exeDir)) // sibling of bin/
	}

	// 4. Relative to CWD.
	if cwd, err := os.Getwd(); err == nil {
		add(filepath.Join(cwd, "multiai-powershell"))
		add(cwd)
		parent := filepath.Dir(cwd)
		add(filepath.Join(parent, "multiai-powershell"))
	}

	return dirs
}

// probeDir checks whether dir contains a PS legacy installation.
func probeDir(dir string) (*DetectResult, error) {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return nil, nil
	}

	hasPS := fileExists(filepath.Join(dir, "code-router.ps1"))
	hasCMD := fileExists(filepath.Join(dir, "multiai.cmd"))
	if !hasPS && !hasCMD {
		return nil, nil
	}

	profilesDir := filepath.Join(dir, "configs", "profiles")
	if !isDir(profilesDir) {
		profilesDir = dir
	}

	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", profilesDir, err)
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".env") {
			names = append(names, e.Name())
		}
	}

	version := readPSVersion(filepath.Join(dir, "package.json"))

	return &DetectResult{
		RootDir:      dir,
		ProfilesDir:  profilesDir,
		Version:      version,
		ProfileCount: len(names),
		ProfileNames: names,
	}, nil
}

// readPSVersion reads the version field from a legacy package.json.
func readPSVersion(pkgPath string) string {
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return ""
	}
	var meta struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return ""
	}
	return meta.Version
}

// fileExists reports whether path is a regular file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// isDir reports whether path is a directory.
func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
