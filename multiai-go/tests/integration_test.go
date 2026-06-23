package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lrochetta/multiai/internal/cli"
	"github.com/lrochetta/multiai/internal/profile"
)

func TestIntegration_LoadProfiles(t *testing.T) {
	dir := t.TempDir()
	content := `PROFILE_ID=test-int
SHORTCUT=ti
TOOL=claude
TOOL_LABEL=Claude Code
DISPLAY_NAME=Test Integration
ORDER=10
COMMAND=claude
CLEAR_ENV=true
REQUIRED_SECRETS=ANTHROPIC_API_KEY
ANTHROPIC_API_KEY=sk-ant-test123
`
	os.WriteFile(filepath.Join(dir, "10-test.env"), []byte(content), 0644)

	profiles, err := profile.LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(profiles))
	}
	p := profiles[0]
	if p.Tool != "claude" {
		t.Errorf("Tool = %q, want claude", p.Tool)
	}
	if len(p.RequiredSecrets) != 1 {
		t.Errorf("expected 1 required secret, got %d", len(p.RequiredSecrets))
	}
}

func TestIntegration_FindProfile(t *testing.T) {
	profiles := []profile.Profile{
		{ID: "deepseek", Shortcut: "ds", Tool: "claude", DisplayName: "DeepSeek V4 Pro", Order: 30},
		{ID: "codex55", Shortcut: "c55", Tool: "codex", DisplayName: "Codex GPT-5.5", Order: 10},
	}

	tests := []struct {
		name    string
		query   string
		wantID  string
		wantErr bool
	}{
		{"exact shortcut", "ds", "deepseek", false},
		{"exact id", "deepseek", "deepseek", false},
		{"case insensitive", "DS", "deepseek", false},
		{"not found", "nonexistent", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := profile.FindByShortcut(profiles, tt.query)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if p != nil && p.ID != tt.wantID {
				t.Errorf("got ID %q, want %q", p.ID, tt.wantID)
			}
		})
	}
}

func TestIntegration_AllowedCommands(t *testing.T) {
	for _, cmd := range cli.AllowedCommands {
		if !cli.IsCommandAllowed(cmd) {
			t.Errorf("%s should be allowed", cmd)
		}
	}
	if cli.IsCommandAllowed("rm") {
		t.Error("rm should NOT be allowed")
	}
	if cli.IsCommandAllowed("bash") {
		t.Error("bash should NOT be allowed")
	}
}

func TestIntegration_YAMLLoad(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `id: test-yaml
shortcut: ty
tool: claude
display_name: Test YAML
order: 20
command: claude
env:
  ANTHROPIC_API_KEY: sk-yaml-test
  CLAUDE_CONFIG_DIR: "${HOME}/.claude-test"
clear_env: true
required_secrets:
  - ANTHROPIC_API_KEY
`
	os.WriteFile(filepath.Join(dir, "test.yaml"), []byte(yamlContent), 0644)

	profiles, err := profile.LoadDirYAML(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(profiles))
	}
	p := profiles[0]
	if p.Tool != "claude" || p.Shortcut != "ty" {
		t.Errorf("unexpected profile: %+v", p)
	}
}

func TestIntegration_LoadAllProfiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "01-env.env"), []byte("PROFILE_ID=env-prof\nSHORTCUT=ep\nTOOL=codex\nCOMMAND=codex\n"), 0644)
	os.WriteFile(filepath.Join(dir, "02-yaml.yaml"), []byte("id: yaml-prof\nshortcut: yp\ntool: claude\ncommand: claude\nenv: {}\n"), 0644)

	profiles, err := profile.LoadAllProfiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) < 2 {
		t.Fatalf("expected at least 2 profiles, got %d", len(profiles))
	}
}
