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

func TestLoadDir_FallbackParsing(t *testing.T) {
	tests := []struct {
		name string
		line string
		want []string
	}{
		{"single shortcut", "FALLBACK=cf", []string{"cf"}},
		{"chain", "FALLBACK=ds,cg", []string{"ds", "cg"}},
		{"spaces and empties", "FALLBACK= ds , cg ,, ", []string{"ds", "cg"}},
		{"empty value", "FALLBACK=", nil},
		{"absent", "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := "PROFILE_ID=fb-test\nSHORTCUT=fbt\nTOOL=claude\nCOMMAND=claude\n"
			if tt.line != "" {
				content += tt.line + "\n"
			}
			dir := createTempProfile(t, "01-fb.env", content)

			profiles, err := LoadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			if len(profiles) != 1 {
				t.Fatalf("expected 1 profile, got %d", len(profiles))
			}
			p := profiles[0]
			if len(p.Fallback) != len(tt.want) {
				t.Fatalf("Fallback: got %v, want %v", p.Fallback, tt.want)
			}
			for i := range tt.want {
				if p.Fallback[i] != tt.want[i] {
					t.Errorf("Fallback[%d]: got %q, want %q", i, p.Fallback[i], tt.want[i])
				}
			}
			// FALLBACK is metadata: it must never leak into the child env.
			if _, leaked := p.Env["FALLBACK"]; leaked {
				t.Error("FALLBACK leaked into profile Env")
			}
		})
	}
}

func TestLoadDir_RegionMetadata(t *testing.T) {
	content := `PROFILE_ID=eu-test
SHORTCUT=ceu
TOOL=claude
COMMAND=claude
REGION=eu
ANTHROPIC_BASE_URL=https://router.eu.requesty.ai
`
	dir := createTempProfile(t, "03-eu.env", content)

	profiles, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	p := profiles[0]
	if p.Region != "eu" {
		t.Errorf("Region: got %q, want %q", p.Region, "eu")
	}
	if _, leaked := p.Env["REGION"]; leaked {
		t.Error("REGION leaked into profile Env")
	}
	if p.Env["ANTHROPIC_BASE_URL"] != "https://router.eu.requesty.ai" {
		t.Error("regular env var missing after metadata filtering")
	}
}

func TestLoadDir_SkipSecretCheck(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{"true", "SKIP_SECRET_CHECK=true", true},
		{"true uppercase", "SKIP_SECRET_CHECK=TRUE", true},
		{"one", "SKIP_SECRET_CHECK=1", true},   // PS parity: ^(true|1|yes)$
		{"yes", "SKIP_SECRET_CHECK=yes", true}, // PS parity: ^(true|1|yes)$
		{"false", "SKIP_SECRET_CHECK=false", false},
		{"garbage", "SKIP_SECRET_CHECK=maybe", false},
		{"absent", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := "PROFILE_ID=ssc\nSHORTCUT=ssc\nTOOL=claude\nCOMMAND=claude\n"
			if tt.line != "" {
				content += tt.line + "\n"
			}
			dir := createTempProfile(t, "00-ssc.env", content)

			profiles, err := LoadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			p := profiles[0]
			if p.SkipSecretCheck != tt.want {
				t.Errorf("SkipSecretCheck: got %v, want %v", p.SkipSecretCheck, tt.want)
			}
			if _, leaked := p.Env["SKIP_SECRET_CHECK"]; leaked {
				t.Error("SKIP_SECRET_CHECK leaked into profile Env")
			}
		})
	}
}

func TestYAMLProfile_FallbackAndRegion(t *testing.T) {
	dir := t.TempDir()
	content := `id: yaml-fb
shortcut: yfb
tool: claude
command: claude
fallback:
  - ds
  - cg
region: eu
env: {}
`
	path := filepath.Join(dir, "yaml-fb.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	p, err := LoadYAML(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Fallback) != 2 || p.Fallback[0] != "ds" || p.Fallback[1] != "cg" {
		t.Errorf("Fallback: got %v, want [ds cg]", p.Fallback)
	}
	if p.Region != "eu" {
		t.Errorf("Region: got %q, want %q", p.Region, "eu")
	}
}

func TestSplitArgs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"empty", "", nil},
		{"single", "--verbose", []string{"--verbose"}},
		{"multiple", "-a -b -c", []string{"-a", "-b", "-c"}},
		{"double quotes", `--flag "two words" plain`, []string{"--flag", "two words", "plain"}},
		{"single quotes", "--flag 'two words' plain", []string{"--flag", "two words", "plain"}},
		{"tabs", "-a\t-b", []string{"-a", "-b"}},
		{"quote inside quote", `--m "it's ok"`, []string{"--m", "it's ok"}},
		{"whitespace only", "   ", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitArgs(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v (%d), want %v (%d)", got, len(got), tt.want, len(tt.want))
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("arg[%d]: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSplitCSV(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"empty", "", nil},
		{"single", "cf", []string{"cf"}},
		{"chain", "ds,cg", []string{"ds", "cg"}},
		{"messy", " a ,, b , ", []string{"a", "b"}},
		{"only commas", ",,,", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitCSV(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("item[%d]: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
