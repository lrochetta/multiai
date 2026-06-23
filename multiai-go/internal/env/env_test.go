package env

import (
	"os"
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
