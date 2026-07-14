// Package tests contains integration and end-to-end tests for the multiai CLI.
// E2E tests build the real binary and exercise it as a black-box CLI.
package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// TestMain — build the real binary once for all E2E tests
// ---------------------------------------------------------------------------

var multiaiBin string

func TestMain(m *testing.M) {
	// Build the multiai binary into a temp location.
	tmpBin := filepath.Join(os.TempDir(), "multiai-e2e-test")
	if runtime.GOOS == "windows" {
		tmpBin += ".exe"
	}

	// Resolve the cmd/multiai package path relative to the tests/ directory.
	repoRoot := findRepoRoot()
	pkgPath := filepath.Join(repoRoot, "cmd", "multiai")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	cmd := exec.CommandContext(ctx, "go", "build", "-o", tmpBin, "-ldflags=-s -w", pkgPath)
	cmd.Stderr = os.Stderr
	cmd.WaitDelay = 2 * time.Second
	result := make(chan error, 1)
	go func() { result <- cmd.Run() }()

	var buildErr error
	select {
	case buildErr = <-result:
		cancel()
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, "[E2E] multiai build remained blocked for 2m")
		os.Exit(1)
	}
	if buildErr != nil {
		fmt.Fprintf(os.Stderr, "[E2E] failed to build multiai binary: %v\n", buildErr)
		os.Exit(1)
	}

	multiaiBin = tmpBin
	code := m.Run()

	_ = os.Remove(tmpBin)
	os.Exit(code)
}

// findRepoRoot walks up from the test working directory looking for go.mod.
func findRepoRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Last resort: assume we are in multiai-go/tests/
			return filepath.Dir(dir)
		}
		dir = parent
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// runMultiai runs the multiai binary with the given args and optional env
// overrides. It returns stdout, stderr, and the exit code.
func runMultiai(t *testing.T, args []string, env map[string]string) (stdout, stderr string, exitCode int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, multiaiBin, args...)
	cmd.Dir = findRepoRoot() // consistent working directory
	cmd.WaitDelay = 2 * time.Second

	// Inherit current environment, then overlay test-specific vars.
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	result := make(chan error, 1)
	go func() { result <- cmd.Run() }()

	var err error
	select {
	case err = <-result:
	case <-ctx.Done():
		// On Windows, security software can hold CreateProcess before Start
		// returns. Running it behind this controller keeps the test bounded.
		return "", "multiai process startup timed out after 15s", -1
	}
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			exitCode = exit.ExitCode()
		} else {
			exitCode = -1 // failed to start
		}
	} else {
		exitCode = 0
	}

	return outBuf.String(), errBuf.String(), exitCode
}

// writeProfile creates a profile .env file in dir.
func writeProfile(t *testing.T, dir, filename, content string) string {
	t.Helper()
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writeProfile %s: %v", filename, err)
	}
	return path
}

// requireExitCode fails the test if the exit code is not the expected value.
func requireExitCode(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("exit code = %d, want %d", got, want)
	}
}

// requireContains fails the test if s does not contain substr.
func requireContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected output to contain %q\n--- got:\n%s", substr, s)
	}
}

// requireJSONValid fails the test if s cannot be unmarshalled into v.
func requireJSONValid(t *testing.T, s string, v interface{}) {
	t.Helper()
	if err := json.Unmarshal([]byte(s), v); err != nil {
		t.Errorf("invalid JSON in output:\n  %v\n--- raw:\n%s", err, s)
	}
}

// ---------------------------------------------------------------------------
// 1. TestE2E_ListProfiles  —  multiai list --json
// ---------------------------------------------------------------------------

func TestE2E_ListProfiles(t *testing.T) {
	profDir := t.TempDir()

	// Create two test profiles with unique shortcuts to avoid collision with
	// the 37 embedded profiles that ensureProfiles extracts on first access.
	writeProfile(t, profDir, "01-codex.env", `PROFILE_ID=test-codex
SHORTCUT=e2e-tx
TOOL=codex
TOOL_LABEL=Codex CLI
DISPLAY_NAME=Test Codex
ORDER=10
COMMAND=codex
CLEAR_ENV=true
`)
	writeProfile(t, profDir, "02-claude.env", `PROFILE_ID=test-claude
SHORTCUT=e2e-tc
TOOL=claude
TOOL_LABEL=Claude Code
DISPLAY_NAME=Test Claude
ORDER=20
COMMAND=claude
CLEAR_ENV=true
`)

	stdout, stderr, exitCode := runMultiai(t, []string{"list", "--json"}, map[string]string{
		"MULTIAI_PROFILES_DIR": profDir,
	})

	requireExitCode(t, exitCode, 0)
	if stderr != "" {
		t.Logf("stderr: %s", stderr)
	}

	// Verify output is valid JSON array.
	var profiles []map[string]interface{}
	requireJSONValid(t, stdout, &profiles)

	if len(profiles) < 2 {
		t.Fatalf("expected at least 2 profiles in JSON output, got %d", len(profiles))
	}

	// Verify our two test profiles are present among the (possibly many)
	// embedded profiles that ensureProfiles also extracts.
	shortcuts := make(map[string]bool)
	for _, p := range profiles {
		if sc, ok := p["shortcut"].(string); ok {
			shortcuts[sc] = true
		}
	}
	if !shortcuts["e2e-tx"] {
		t.Errorf("missing shortcut 'e2e-tx' in JSON output")
	}
	if !shortcuts["e2e-tc"] {
		t.Errorf("missing shortcut 'e2e-tc' in JSON output")
	}
}

// ---------------------------------------------------------------------------
// 2. TestE2E_Version  —  multiai version
// ---------------------------------------------------------------------------

func TestE2E_Version(t *testing.T) {
	stdout, stderr, exitCode := runMultiai(t, []string{"version"}, nil)

	requireExitCode(t, exitCode, 0)
	if stderr != "" {
		t.Logf("stderr: %s", stderr)
	}

	// Version output must start with "multiai " and contain a version number.
	requireContains(t, stdout, "multiai")
	// Must contain a semver-like pattern (digit.digit.digit). Release builds
	// may override the development version through ldflags.
	if !regexp.MustCompile(`\b\d+\.\d+\.\d+\b`).MatchString(stdout) {
		t.Fatalf("expected semver-like version output, got:\n%s", stdout)
	}
}

// ---------------------------------------------------------------------------
// 3. TestE2E_Help  —  multiai help
// ---------------------------------------------------------------------------

func TestE2E_Help(t *testing.T) {
	stdout, stderr, exitCode := runMultiai(t, []string{"help"}, nil)

	requireExitCode(t, exitCode, 0)
	if stderr != "" {
		t.Logf("stderr: %s", stderr)
	}

	// Help output must contain usage information.
	requireContains(t, stdout, "multiai")
	requireContains(t, stdout, "Usage")
	requireContains(t, stdout, "Routeur")
	requireContains(t, stdout, "help")
}

// ---------------------------------------------------------------------------
// 4. TestE2E_LaunchDryRun  —  multiai launch -p e2eds --allow-custom-command
//    --dry-run
// ---------------------------------------------------------------------------

func TestE2E_LaunchDryRun(t *testing.T) {
	profDir := t.TempDir()

	// Use a unique shortcut ("e2eds") that does NOT collide with any of the
	// 37 embedded profiles that ensureProfiles deposits in the temp dir.
	// The command is "go" because it is guaranteed to be in PATH wherever
	// Go tests run.  --allow-custom-command bypasses the builtin whitelist.
	writeProfile(t, profDir, "90-e2eds.env", `PROFILE_ID=test-e2eds
SHORTCUT=e2eds
TOOL=claude
TOOL_LABEL=Claude Code
DISPLAY_NAME=Test E2E DS
ORDER=90
COMMAND=go
CLEAR_ENV=true
SKIP_SECRET_CHECK=true
`)

	stdout, stderr, exitCode := runMultiai(t, []string{
		"launch", "-p", "e2eds",
		"--allow-custom-command",
		"--dry-run",
	}, map[string]string{
		"MULTIAI_PROFILES_DIR": profDir,
	})

	requireExitCode(t, exitCode, 0)
	requireContains(t, stdout, "DRY RUN")
	requireContains(t, stdout, "e2eds")
	// The command "go" should appear in the dry-run output.
	requireContains(t, stdout, "go")

	if stderr != "" && !strings.Contains(stderr, "custom") {
		// Allow "custom command" warnings on stderr.
		t.Logf("stderr: %s", stderr)
	}
}

// ---------------------------------------------------------------------------
// 5. TestE2E_LaunchShowEnv  —  multiai launch -p e2eds --allow-custom-command
//    --show-env --dry-run
//
// --dry-run is needed alongside --show-env because otherwise the binary
// proceeds to launch the profile command (which may exit non-zero).
// ---------------------------------------------------------------------------

func TestE2E_LaunchShowEnv(t *testing.T) {
	profDir := t.TempDir()

	writeProfile(t, profDir, "90-e2eds.env", `PROFILE_ID=test-e2eds
SHORTCUT=e2eds
TOOL=claude
TOOL_LABEL=Claude Code
DISPLAY_NAME=Test E2E DS
ORDER=90
COMMAND=go
CLEAR_ENV=true
SKIP_SECRET_CHECK=true
MY_CUSTOM_VAR=hello-world
`)

	stdout, stderr, exitCode := runMultiai(t, []string{
		"launch", "-p", "e2eds",
		"--allow-custom-command",
		"--show-env",
		"--dry-run",
	}, map[string]string{
		"MULTIAI_PROFILES_DIR": profDir,
	})

	requireExitCode(t, exitCode, 0)
	// Should show the profile name and the custom variable.
	requireContains(t, stdout, "Test E2E DS")
	requireContains(t, stdout, "MY_CUSTOM_VAR")
	requireContains(t, stdout, "hello-world")
	// DRY RUN marker should appear since --dry-run is active.
	requireContains(t, stdout, "DRY RUN")

	if stderr != "" && !strings.Contains(stderr, "custom") {
		t.Logf("stderr: %s", stderr)
	}
}

// ---------------------------------------------------------------------------
// 6. TestE2E_ConfigStore  —  multiai config --store file
//    (pipe "0" = back to exit the interactive menu)
// ---------------------------------------------------------------------------

func TestE2E_ConfigStore(t *testing.T) {
	profDir := t.TempDir()
	secretsDir := t.TempDir()

	// At least one profile must exist so the config menu has content.
	writeProfile(t, profDir, "01-claude.env", `PROFILE_ID=test-claude
SHORTCUT=tc
TOOL=claude
TOOL_LABEL=Claude Code
DISPLAY_NAME=Test Claude
ORDER=10
COMMAND=claude
CLEAR_ENV=true
`)

	bin := multiaiBin
	cmd := exec.Command(bin, "config", "--store", "file")
	cmd.Dir = findRepoRoot()
	cmd.Env = append(os.Environ(),
		"MULTIAI_PROFILES_DIR="+profDir,
		"MULTIAI_SECRETS_DIR="+secretsDir,
	)

	// Pipe "0\n" to exit the interactive menu immediately.
	cmd.Stdin = bytes.NewReader([]byte("0\n"))

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			exitCode = exit.ExitCode()
		} else {
			exitCode = -1
		}
	}

	requireExitCode(t, exitCode, 0)

	// Assert stable catalog content rather than a localized menu label. The
	// command selects French or English from the host locale (notably macOS CI).
	requireContains(t, outBuf.String(), "Anthropic")

	// Verify the file store was created in the temp secrets dir.
	entries, err := os.ReadDir(secretsDir)
	if err != nil {
		t.Fatalf("cannot read secrets dir: %v", err)
	}
	if len(entries) == 0 {
		t.Error("secrets dir is empty after --store file -- file store should have created a master key")
	}

	// We expect at least .masterkey to exist (and possibly .enc files).
	foundMasterKey := false
	for _, e := range entries {
		if e.Name() == ".masterkey" {
			foundMasterKey = true
			break
		}
	}
	if !foundMasterKey {
		t.Errorf("expected .masterkey in secrets dir %s, got: %v", secretsDir, listNames(entries))
	}

	if errBuf.Len() > 0 {
		t.Logf("stderr: %s", errBuf.String())
	}
}

// listNames returns a list of entry names for error messages.
func listNames(entries []os.DirEntry) []string {
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name()
	}
	return names
}

// ---------------------------------------------------------------------------
// 7. TestE2E_Models  —  multiai models --offline
// ---------------------------------------------------------------------------

func TestE2E_Models(t *testing.T) {
	stdout, stderr, exitCode := runMultiai(t, []string{"models", "--offline"}, nil)

	requireExitCode(t, exitCode, 0)

	// The embedded model list should include these well-known entries.
	requireContains(t, stdout, "Fusion")
	requireContains(t, stdout, "DeepSeek")
	requireContains(t, stdout, "Claude Sonnet")
	requireContains(t, stdout, "GPT-5.5")

	// A summary line should be printed with source info.
	requireContains(t, stdout, "embarqu")

	// Stderr may contain a warning about degraded/cache mode; that is fine.
	if stderr != "" {
		t.Logf("stderr: %s", stderr)
	}
}

// ---------------------------------------------------------------------------
// 8. TestE2E_UpdateCheck  —  multiai update --check
//
// This test works both online and offline:
//   - Online  (exit 0): JSON output with version info.
//   - Offline (exit 2): error message on stderr.
// In both cases we verify the output is well-formed and the binary does not
// crash or hang.
// ---------------------------------------------------------------------------

func TestE2E_UpdateCheck(t *testing.T) {
	// Isolate the update cache so this test never interferes with the user's.
	cacheDir := t.TempDir()

	stdout, stderr, exitCode := runMultiai(t, []string{"update", "--check"}, map[string]string{
		"MULTIAI_CACHE_DIR": cacheDir,
	})

	// Exit code must be 0 (success, check completed) or 2 (network error).
	if exitCode != 0 && exitCode != 2 {
		t.Fatalf("exit code = %d; expected 0 (online) or 2 (offline)", exitCode)
	}

	switch exitCode {
	case 0:
		// JSON output with version info.
		var result struct {
			CurrentVersion string `json:"current_version"`
			LatestVersion  string `json:"latest_version"`
			HasUpdate      bool   `json:"has_update"`
		}
		requireJSONValid(t, stdout, &result)
		if result.CurrentVersion == "" {
			t.Error("current_version is empty in JSON output")
		}
		if result.LatestVersion == "" {
			t.Error("latest_version is empty in JSON output")
		}
		t.Logf("update --check online: current=%s latest=%s has_update=%v",
			result.CurrentVersion, result.LatestVersion, result.HasUpdate)

	case 2:
		// Offline — must have an error message on stderr.
		requireContains(t, stderr, "Impossible")
		requireContains(t, stderr, "mises a jour")
		t.Log("update --check offline (no network): verified error output")
	}
}

// ---------------------------------------------------------------------------
// 9. Bonus: TestE2E_ListNoJSON  —  multiai list (tabular output)
// ---------------------------------------------------------------------------

func TestE2E_ListNoJSON(t *testing.T) {
	profDir := t.TempDir()

	writeProfile(t, profDir, "01-test.env", `PROFILE_ID=bonus-test
SHORTCUT=e2ebt
TOOL=claude
TOOL_LABEL=Claude Code
DISPLAY_NAME=Bonus Test
ORDER=10
COMMAND=claude
CLEAR_ENV=true
`)

	stdout, stderr, exitCode := runMultiai(t, []string{"list"}, map[string]string{
		"MULTIAI_PROFILES_DIR": profDir,
	})

	requireExitCode(t, exitCode, 0)
	requireContains(t, stdout, "Tool")
	requireContains(t, stdout, "Shortcut")
	requireContains(t, stdout, "e2ebt")
	requireContains(t, stdout, "Bonus Test")

	if stderr != "" {
		t.Logf("stderr: %s", stderr)
	}
}

// ---------------------------------------------------------------------------
// 10. Bonus: TestE2E_HelpShortcut  —  multiai --help / -h
// ---------------------------------------------------------------------------

func TestE2E_HelpShortcut(t *testing.T) {
	// Test both --help and -h variants.
	for _, flag := range []string{"--help", "-h"} {
		t.Run(flag, func(t *testing.T) {
			stdout, stderr, exitCode := runMultiai(t, []string{flag}, nil)
			requireExitCode(t, exitCode, 0)
			requireContains(t, stdout, "multiai")
			requireContains(t, stdout, "Usage")
			if stderr != "" {
				t.Logf("stderr: %s", stderr)
			}
		})
	}
}
