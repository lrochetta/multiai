package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lrochetta/multiai/internal/profile"
)

func TestConfig_UpdateEnvFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.env")
	content := "ANTHROPIC_API_KEY=PASTE_YOUR_KEY_HERE\nOTHER_VAR=value\n"
	os.WriteFile(path, []byte(content), 0644)

	// Test that we can detect placeholder
	profiles, err := profile.LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 1 {
		t.Fatal("expected 1 profile")
	}

	val := profiles[0].Env["ANTHROPIC_API_KEY"]
	if !strings.Contains(val, "PASTE_") {
		t.Errorf("expected placeholder, got %q", val)
	}
}
