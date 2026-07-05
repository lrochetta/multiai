package openrouter

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lrochetta/multiai/internal/profile"
)

func TestShortcut(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"DeepSeek V4 Pro", "or-deepseekv"},    // stripped, lowercased, cut at 12
		{"Owl", "or-owl"},                      // short names keep everything
		{"GPT-5.5", "or-gpt55"},                // punctuation stripped
		{"Modele Eteint 9000", "or-modeleete"}, // non-ASCII letters kept only if [a-zA-Z0-9]
		{"a b c", "or-abc"},
	}
	for _, tt := range tests {
		if got := Shortcut(tt.in); got != tt.want {
			t.Errorf("Shortcut(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestProfileFileName(t *testing.T) {
	if got := ProfileFileName("DeepSeek V4 Pro"); got != "99-or-deepseekv.env" {
		t.Errorf("ProfileFileName = %q", got)
	}
}

func TestActiveProfilesDirEnvOverride(t *testing.T) {
	t.Setenv("MULTIAI_PROFILES_DIR", `X:\somewhere\profiles`)
	dir, err := ActiveProfilesDir()
	if err != nil {
		t.Fatal(err)
	}
	if dir != `X:\somewhere\profiles` {
		t.Errorf("dir = %q", dir)
	}
}

func TestRenderClaudeProfileExactContent(t *testing.T) {
	got, err := Render(ProfileSpec{DisplayName: "DeepSeek V4 Pro", ModelSlug: "deepseek/deepseek-v4-pro", Tool: "claude"})
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		"PROFILE_ID=or-deepseekv",
		"SHORTCUT=or-deepseekv",
		"TOOL=claude",
		"TOOL_LABEL=claude",
		"DISPLAY_NAME=DeepSeek V4 Pro (via OR)",
		"DESCRIPTION=OpenRouter: deepseek/deepseek-v4-pro",
		"ORDER=50",
		"COMMAND=claude",
		"CLEAR_ENV=true",
		"REQUIRED_SECRETS=OPENROUTER_API_KEY",
		"OPENROUTER_API_KEY=PASTE_OPENROUTER_API_KEY_HERE",
		"ANTHROPIC_AUTH_TOKEN=%OPENROUTER_API_KEY%",
		"ANTHROPIC_BASE_URL=https://openrouter.ai/api", // no /v1: Claude Code appends /v1/messages
		"ANTHROPIC_MODEL=deepseek/deepseek-v4-pro",
		"ANTHROPIC_API_KEY=",
	}, "\r\n") + "\r\n"
	if got != want {
		t.Errorf("claude profile content mismatch:\ngot:\n%q\nwant:\n%q", got, want)
	}
}

func TestRenderCodexProfile(t *testing.T) {
	got, err := Render(ProfileSpec{DisplayName: "GPT 5.5", ModelSlug: "openai/gpt-5.5", Tool: "codex"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "OPENAI_BASE_URL=https://openrouter.ai/api/v1\r\n") {
		t.Errorf("codex profile must use /api/v1 base URL:\n%s", got)
	}
	if !strings.Contains(got, "OPENAI_API_KEY=%OPENROUTER_API_KEY%\r\n") {
		t.Errorf("codex profile must reference the OpenRouter key:\n%s", got)
	}
	if strings.Contains(got, "ANTHROPIC") {
		t.Errorf("codex profile must not carry Anthropic variables:\n%s", got)
	}
}

func TestRenderOpencodeProfileHasNoToolLines(t *testing.T) {
	got, err := Render(ProfileSpec{DisplayName: "Owl", ModelSlug: "openrouter/owl-alpha", Tool: "opencode"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "ANTHROPIC") || strings.Contains(got, "OPENAI_BASE_URL") {
		t.Errorf("opencode profile must only keep OPENROUTER_API_KEY:\n%s", got)
	}
	if !strings.HasSuffix(got, "OPENROUTER_API_KEY=PASTE_OPENROUTER_API_KEY_HERE\r\n") {
		t.Errorf("opencode profile must end after the key placeholder:\n%q", got)
	}
}

func TestRenderValidation(t *testing.T) {
	tests := []struct {
		name    string
		spec    ProfileSpec
		wantErr string
	}{
		{"empty name", ProfileSpec{"", "a/b", "claude"}, "nom de modele vide"},
		{"no alphanumerics", ProfileSpec{"---", "a/b", "claude"}, "aucun caractere alphanumerique"},
		{"control char injection", ProfileSpec{"Evil\r\nCOMMAND=powershell", "a/b", "claude"}, "caractere de controle"},
		{"bad slug no slash", ProfileSpec{"Foo", "badslug", "claude"}, "slug OpenRouter invalide"},
		{"bad slug spaces", ProfileSpec{"Foo", "a b/c", "claude"}, "slug OpenRouter invalide"},
		{"bad tool", ProfileSpec{"Foo", "a/b", "vim"}, "outil inconnu"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Render(tt.spec)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("want error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestCreateProfileRefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	spec := ProfileSpec{DisplayName: "DeepSeek V4 Pro", ModelSlug: "deepseek/deepseek-v4-pro", Tool: "claude"}

	path1, err := CreateProfile(dir, spec, false)
	if err != nil {
		t.Fatalf("first CreateProfile: %v", err)
	}
	if filepath.Base(path1) != "99-or-deepseekv.env" {
		t.Errorf("file name = %s", filepath.Base(path1))
	}

	path2, err := CreateProfile(dir, spec, false)
	if !errors.Is(err, ErrProfileExists) {
		t.Fatalf("want ErrProfileExists, got %v", err)
	}
	if path2 != path1 {
		t.Errorf("conflict path = %s, want %s", path2, path1)
	}

	if _, err := CreateProfile(dir, spec, true); err != nil {
		t.Fatalf("overwrite=true should succeed: %v", err)
	}
}

func TestCreateProfileCreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "profiles")
	spec := ProfileSpec{DisplayName: "Owl", ModelSlug: "openrouter/owl-alpha", Tool: "opencode"}
	if _, err := CreateProfile(dir, spec, false); err != nil {
		t.Fatalf("CreateProfile with missing dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "99-or-owl.env")); err != nil {
		t.Fatalf("profile file missing: %v", err)
	}
}

// TestCreatedProfileLoadsAsValidProfile guards the integration contract: the
// generated .env must round-trip through the real profile loader.
func TestCreatedProfileLoadsAsValidProfile(t *testing.T) {
	dir := t.TempDir()
	spec := ProfileSpec{DisplayName: "DeepSeek V4 Pro", ModelSlug: "deepseek/deepseek-v4-pro", Tool: "claude"}
	if _, err := CreateProfile(dir, spec, false); err != nil {
		t.Fatal(err)
	}

	profiles, err := profile.LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir: %v", err)
	}
	if len(profiles) != 1 {
		t.Fatalf("got %d profiles, want 1", len(profiles))
	}
	p := profiles[0]
	if p.Shortcut != "or-deepseekv" {
		t.Errorf("shortcut = %q", p.Shortcut)
	}
	if p.Tool != "claude" || p.Command != "claude" {
		t.Errorf("tool/command = %q/%q", p.Tool, p.Command)
	}
	if p.DisplayName != "DeepSeek V4 Pro (via OR)" {
		t.Errorf("display name = %q", p.DisplayName)
	}
	if !p.ClearEnv {
		t.Error("ClearEnv should be true")
	}
	if len(p.RequiredSecrets) != 1 || p.RequiredSecrets[0] != "OPENROUTER_API_KEY" {
		t.Errorf("required secrets = %v", p.RequiredSecrets)
	}
	if p.Env["ANTHROPIC_BASE_URL"] != "https://openrouter.ai/api" {
		t.Errorf("ANTHROPIC_BASE_URL = %q", p.Env["ANTHROPIC_BASE_URL"])
	}
	if p.Env["ANTHROPIC_AUTH_TOKEN"] != "%OPENROUTER_API_KEY%" {
		t.Errorf("ANTHROPIC_AUTH_TOKEN = %q", p.Env["ANTHROPIC_AUTH_TOKEN"])
	}
	if v, ok := p.Env["ANTHROPIC_API_KEY"]; !ok || v != "" {
		t.Errorf("ANTHROPIC_API_KEY should be present and empty, got %q (ok=%v)", v, ok)
	}
}
