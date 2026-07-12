package onboarding

import (
	"bufio"
	"strings"
	"testing"

	"github.com/lrochetta/multiai/internal/profile"
	"github.com/lrochetta/multiai/internal/secret"
)

func profileWithEnv(env map[string]string) profile.Profile {
	return profile.Profile{ID: "t", Shortcut: "t", Tool: "claude", Env: env}
}

func TestIsFirstRun(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{
			name: "placeholder key means first run",
			env:  map[string]string{"DEEPSEEK_API_KEY": "PASTE_DEEPSEEK_API_KEY_HERE"},
			want: true,
		},
		{
			name: "empty key means first run",
			env:  map[string]string{"ANTHROPIC_API_KEY": ""},
			want: true,
		},
		{
			name: "pure %VAR% indirection is not a configured key (fusion-style)",
			env: map[string]string{
				"OPENROUTER_API_KEY":   "PASTE_OPENROUTER_API_KEY_HERE",
				"ANTHROPIC_AUTH_TOKEN": "%OPENROUTER_API_KEY%",
			},
			want: true,
		},
		{
			name: "credential-store sentinel counts as configured",
			env:  map[string]string{"OPENROUTER_API_KEY": secret.Sentinel},
			want: false,
		},
		{
			name: "real-looking key counts as configured",
			env:  map[string]string{"DEEPSEEK_API_KEY": "sk-abcdef1234567890"},
			want: false,
		},
		{
			name: "non-secret vars are ignored",
			env: map[string]string{
				"ANTHROPIC_BASE_URL": "https://api.deepseek.com/anthropic",
				"ANTHROPIC_MODEL":    "deepseek-chat",
			},
			want: true,
		},
		{
			name: "indirection with surrounding text is treated as a value",
			env:  map[string]string{"SOME_AUTH_HEADER": "Bearer %OPENROUTER_API_KEY%"},
			want: false,
		},
		{
			name: "no profiles at all",
			env:  nil,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var profiles []profile.Profile
			if tt.env != nil {
				profiles = []profile.Profile{profileWithEnv(tt.env)}
			}
			if got := IsFirstRun(profiles); got != tt.want {
				t.Errorf("IsFirstRun() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFirstRunMarker(t *testing.T) {
	home := t.TempDir()
	// os.UserHomeDir reads USERPROFILE on Windows and HOME elsewhere.
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	if FirstRunMarkerExists() {
		t.Fatal("marker should not exist in a fresh home dir")
	}
	markFirstRunDone()
	if !FirstRunMarkerExists() {
		t.Fatal("marker should exist after markFirstRunDone")
	}
	// Idempotent: marking again must not fail or flip the state.
	markFirstRunDone()
	if !FirstRunMarkerExists() {
		t.Fatal("marker should survive a second markFirstRunDone")
	}
}

func TestRunWelcomeEOFReturnsWithoutMarkingFirstRunDone(t *testing.T) {
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	runWelcome(nil, bufio.NewReader(strings.NewReader("")))

	if FirstRunMarkerExists() {
		t.Fatal("EOF must not mark onboarding complete")
	}
}
