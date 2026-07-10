package menu

import (
	"testing"

	"github.com/lrochetta/multiai/internal/profile"
)

func TestCountSecrets_AllSet(t *testing.T) {
	p := &profile.Profile{
		Env: map[string]string{
			"OPENAI_API_KEY":     "sk-real-key-12345",
			"ANTHROPIC_API_KEY":  "sk-ant-real-key",
		},
	}
	got, total := countSecrets(p)
	if got != 2 {
		t.Errorf("configured = %d, want 2", got)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if got != total {
		t.Errorf("all keys should be configured, but configured (%d) != total (%d)", got, total)
	}
}

func TestCountSecrets_NoneSet(t *testing.T) {
	p := &profile.Profile{
		Env: map[string]string{
			"OPENAI_API_KEY":     "paste_your_key_here",
			"ANTHROPIC_API_KEY":  "",
			"OTHER_SECRET":       "your_value_here",
		},
	}
	got, total := countSecrets(p)
	if got != 0 {
		t.Errorf("configured = %d, want 0", got)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
}

func TestCountSecrets_Partial(t *testing.T) {
	p := &profile.Profile{
		Env: map[string]string{
			"OPENAI_API_KEY":     "sk-real-key-12345",
			"ANTHROPIC_API_KEY":  "ta_cle_ici",
			"OTHER_SECRET":       "",
			"AZURE_API_KEY":      "valid-key",
		},
	}
	got, total := countSecrets(p)
	if got != 2 {
		t.Errorf("configured = %d, want 2 (OPENAI_API_KEY and AZURE_API_KEY)", got)
	}
	if total != 4 {
		t.Errorf("total = %d, want 4", total)
	}
}

func TestCountSecrets_MetadataKeysIgnored(t *testing.T) {
	p := &profile.Profile{
		Env: map[string]string{
			"API_KEY":           "valid",
			"PROFILE_ID":        "ignored",
			"SHORTCUT":          "ig",
			"TOOL":              "ignored",
			"TOOL_LABEL":        "ignored",
			"DISPLAY_NAME":      "ignored",
			"ORDER":             "1",
			"COMMAND":           "ignored",
		},
	}
	got, total := countSecrets(p)
	if got != 1 {
		t.Errorf("configured = %d, want 1 (only API_KEY)", got)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1 (metadata keys excluded)", total)
	}
}
