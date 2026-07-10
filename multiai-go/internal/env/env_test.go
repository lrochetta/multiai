package env

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestBuildCleanEnv(t *testing.T) {
	// Set a test var
	os.Setenv("TEST_KEEP_VAR", "keep_me")
	defer os.Unsetenv("TEST_KEEP_VAR")

	profileEnv := map[string]string{
		"ANTHROPIC_API_KEY": "sk-ant-test123",
	}

	result := BuildCleanEnv(profileEnv)

	// Convert result slice to map for easier testing
	envMap := make(map[string]string)
	for _, kv := range result {
		idx := strings.Index(kv, "=")
		if idx < 0 {
			continue
		}
		envMap[kv[:idx]] = kv[idx+1:]
	}

	// PATH should be preserved
	if envMap["PATH"] == "" {
		t.Error("PATH should be preserved")
	}

	// Profile var should be present
	if envMap["ANTHROPIC_API_KEY"] != "sk-ant-test123" {
		t.Errorf("ANTHROPIC_API_KEY not set correctly: got %q", envMap["ANTHROPIC_API_KEY"])
	}

	// Random test var should NOT be preserved (not in allowlist)
	if _, ok := envMap["TEST_KEEP_VAR"]; ok {
		t.Error("TEST_KEEP_VAR should have been cleaned (not in allowlist)")
	}
}

func TestExpandProfileEnv(t *testing.T) {
	os.Setenv("USERPROFILE", `C:\Users\test`)
	defer os.Unsetenv("USERPROFILE")

	profileEnv := map[string]string{
		// fusion-style indirection: the auth token points at another key
		"OPENROUTER_API_KEY":   "sk-or-real-value",
		"ANTHROPIC_AUTH_TOKEN": "%OPENROUTER_API_KEY%",
		// allow-listed system var
		"CLAUDE_CONFIG_DIR": `%USERPROFILE%\.claude-fusion`,
		// unknown reference: kept literal, like PS
		"KEEP_LITERAL": "%NOT_A_KNOWN_VAR%",
		// no reference: untouched
		"ANTHROPIC_MODEL": "openrouter/fusion",
	}

	got := ExpandProfileEnv(profileEnv)

	if got["ANTHROPIC_AUTH_TOKEN"] != "sk-or-real-value" {
		t.Errorf("indirection not resolved: got %q", got["ANTHROPIC_AUTH_TOKEN"])
	}
	if got["CLAUDE_CONFIG_DIR"] != `C:\Users\test\.claude-fusion` {
		t.Errorf("system var not resolved: got %q", got["CLAUDE_CONFIG_DIR"])
	}
	if got["KEEP_LITERAL"] != "%NOT_A_KNOWN_VAR%" {
		t.Errorf("unknown var should stay literal: got %q", got["KEEP_LITERAL"])
	}
	if got["ANTHROPIC_MODEL"] != "openrouter/fusion" {
		t.Errorf("plain value altered: got %q", got["ANTHROPIC_MODEL"])
	}
	// The source map must not be mutated.
	if profileEnv["ANTHROPIC_AUTH_TOKEN"] != "%OPENROUTER_API_KEY%" {
		t.Error("ExpandProfileEnv mutated its input")
	}
}

func TestExpandWindowsVarsCycleGuard(t *testing.T) {
	// A -> B -> A must not loop; depth cap leaves a literal, never hangs.
	profileEnv := map[string]string{
		"A": "%B%",
		"B": "%A%",
	}
	got := ExpandProfileEnv(profileEnv) // must return, not deadlock
	if got["A"] == "" {
		t.Error("cyclic expansion should degrade to a literal, not empty")
	}
}

func TestBuildCleanEnvResolvesIndirection(t *testing.T) {
	profileEnv := map[string]string{
		"OPENROUTER_API_KEY":   "sk-or-xyz",
		"ANTHROPIC_AUTH_TOKEN": "%OPENROUTER_API_KEY%",
	}
	envMap := make(map[string]string)
	for _, kv := range BuildCleanEnv(profileEnv) {
		if idx := strings.Index(kv, "="); idx >= 0 {
			envMap[kv[:idx]] = kv[idx+1:]
		}
	}
	if envMap["ANTHROPIC_AUTH_TOKEN"] != "sk-or-xyz" {
		t.Errorf("BuildCleanEnv did not resolve indirection: got %q", envMap["ANTHROPIC_AUTH_TOKEN"])
	}
}

func TestIsSecretKey(t *testing.T) {
	tests := []struct {
		key      string
		expected bool
	}{
		{"ANTHROPIC_API_KEY", true},
		{"GITHUB_TOKEN", true},
		{"DB_PASSWORD", true},
		{"ANTHROPIC_AUTH_TOKEN", true},
		{"MY_CREDENTIAL", true},
		{"AWS_SECRET_ACCESS_KEY", true},
		{"PATH", false},
		{"HOME", false},
		{"DISPLAY_NAME", false},
	}

	for _, tt := range tests {
		result := IsSecretKey(tt.key)
		if result != tt.expected {
			t.Errorf("IsSecretKey(%q) = %v, want %v", tt.key, result, tt.expected)
		}
	}
}

func TestMaskSecret(t *testing.T) {
	tests := []struct {
		value    string
		expected string
	}{
		{"sk-ant-api-03-abc123def456", "sk-a...f456"},
		{"short", "***"},
		{"ab", "***"},
		{"", "<vide>"},
		{"123456789", "1234...6789"},
	}

	for _, tt := range tests {
		result := MaskSecret(tt.value)
		if result != tt.expected {
			t.Errorf("MaskSecret(%q) = %q, want %q", tt.value, result, tt.expected)
		}
	}
}

func TestWhitelistCaseInsensitiveWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test: case-insensitive env var matching")
	}

	cases := []struct {
		key  string
		want bool
	}{
		{"PATH", true},
		{"Path", true},
		{"path", true},
		{"PATH", true},
		{"pAtH", true},
		{"TEMP", true},
		{"Temp", true},
		{"USERPROFILE", true},
		{"UserProfile", true},
		{"userprofile", true},
		{"SYSTEMROOT", true},
		{"SystemRoot", true},
		{"systemroot", true},
		{"COMSPEC", true},
		{"ComSpec", true},
		{"comspec", true},
		// Non-allowlisted
		{"MY_CUSTOM_VAR", false},
		{"", false},
	}

	for _, tc := range cases {
		got := isAllowed(tc.key)
		if got != tc.want {
			t.Errorf("isAllowed(%q) on Windows = %v, want %v", tc.key, got, tc.want)
		}
	}
}

func TestWhitelistCaseSensitiveLinux(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("not applicable on Windows")
	}

	// Uppercase must still pass
	if !isAllowed("PATH") {
		t.Error("isAllowed(\"PATH\") should be true on non-Windows")
	}
	if !isAllowed("TEMP") {
		t.Error("isAllowed(\"TEMP\") should be true on non-Windows")
	}
	if !isAllowed("USERPROFILE") {
		t.Error("isAllowed(\"USERPROFILE\") should be true on non-Windows")
	}

	// Mixed/lower case must NOT pass on case-sensitive platforms
	if isAllowed("Path") {
		t.Error("isAllowed(\"Path\") should be false on non-Windows")
	}
	if isAllowed("path") {
		t.Error("isAllowed(\"path\") should be false on non-Windows")
	}
	if isAllowed("temp") {
		t.Error("isAllowed(\"temp\") should be false on non-Windows")
	}
	if isAllowed("userprofile") {
		t.Error("isAllowed(\"userprofile\") should be false on non-Windows")
	}
}

func TestAllWindowsSystemVars(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only test")
	}

	// All common Windows system vars must pass regardless of casing
	vars := []string{
		"PATH", "Path", "path", "PATHEXT", "pathext",
		"TEMP", "Temp", "temp", "TMP", "Tmp",
		"USERPROFILE", "UserProfile", "userprofile",
		"USERNAME", "UserName", "username",
		"SYSTEMROOT", "SystemRoot", "systemroot",
		"WINDIR", "Windir", "windir",
		"COMSPEC", "ComSpec", "comspec",
		"OS", "os",
		"PROCESSOR_ARCHITECTURE", "processor_architecture",
	}

	for _, key := range vars {
		if !isAllowed(key) {
			t.Errorf("isAllowed(%q) = false, want true (Windows system var)", key)
		}
	}
}
