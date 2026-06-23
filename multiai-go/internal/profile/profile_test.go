package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func createTempProfile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestLoadDir_ValidProfiles(t *testing.T) {
	content := `PROFILE_ID=test-profile
SHORTCUT=tp
TOOL=claude
DISPLAY_NAME=Test Profile
ORDER=10
COMMAND=claude
ANTHROPIC_API_KEY=sk-ant-test123
`
	dir := createTempProfile(t, "10-test.env", content)

	profiles, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(profiles))
	}

	p := profiles[0]
	if p.ID != "test-profile" {
		t.Errorf("ID: got %q, want %q", p.ID, "test-profile")
	}
	if p.Shortcut != "tp" {
		t.Errorf("Shortcut: got %q, want %q", p.Shortcut, "tp")
	}
	if p.Tool != "claude" {
		t.Errorf("Tool: got %q, want %q", p.Tool, "claude")
	}
	if p.Command != "claude" {
		t.Errorf("Command: got %q, want %q", p.Command, "claude")
	}
	if p.Order != 10 {
		t.Errorf("Order: got %d, want 10", p.Order)
	}
}

func TestLoadDir_MissingDir(t *testing.T) {
	_, err := LoadDir("/nonexistent/dir")
	if err == nil {
		t.Fatal("expected error for missing directory")
	}
}

func TestLoadDir_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	profiles, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 0 {
		t.Errorf("expected 0 profiles, got %d", len(profiles))
	}
}

func TestFindByShortcut_ExactMatch(t *testing.T) {
	profiles := []Profile{
		{ID: "deepseek", Shortcut: "ds", Tool: "claude"},
		{ID: "codex55", Shortcut: "c55", Tool: "codex"},
	}

	p, err := FindByShortcut(profiles, "ds")
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != "deepseek" {
		t.Errorf("got %q, want deepseek", p.ID)
	}
}

func TestFindByShortcut_NotFound(t *testing.T) {
	profiles := []Profile{
		{ID: "deepseek", Shortcut: "ds", Tool: "claude"},
	}

	_, err := FindByShortcut(profiles, "nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown profile")
	}
}
