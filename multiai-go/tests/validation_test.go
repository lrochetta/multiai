package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lrochetta/multiai/internal/profile"
)

func TestValidateProfileYAML_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "valid.yaml")
	yaml := `id: valid-prof
shortcut: vp
tool: claude
command: claude
env:
  KEY: value
`
	os.WriteFile(path, []byte(yaml), 0644)

	warnings, err := profile.ValidateProfileYAML(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) > 0 {
		t.Errorf("expected 0 warnings, got %v", warnings)
	}
}

func TestValidateProfileYAML_NoID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "noid.yaml")
	yaml := `tool: claude
env: {}
`
	os.WriteFile(path, []byte(yaml), 0644)

	warnings, err := profile.ValidateProfileYAML(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) == 0 {
		t.Error("expected warnings for missing id")
	}
}

func TestValidateProfileYAML_BadCommand(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "badcmd.yaml")
	yaml := `id: bad
tool: claude
command: rm
env: {}
`
	os.WriteFile(path, []byte(yaml), 0644)

	warnings, err := profile.ValidateProfileYAML(path)
	if err != nil {
		t.Fatal(err)
	}
	hasBadCmd := false
	for _, w := range warnings {
		if contains(w, "whitelist") {
			hasBadCmd = true
			break
		}
	}
	if !hasBadCmd {
		t.Error("expected warning about command not in whitelist")
	}
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
