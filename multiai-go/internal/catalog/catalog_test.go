package catalog

import (
	"reflect"
	"strings"
	"testing"
)

func TestEmbeddedCatalogLoads(t *testing.T) {
	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.Version != SchemaVersion {
		t.Errorf("version = %d, want %d", c.Version, SchemaVersion)
	}
	if got := len(c.Providers); got != 14 {
		t.Errorf("providers = %d, want 14 (13 PS parity + nvidia)", got)
	}
	if got := len(c.Regions); got != 3 {
		t.Errorf("regions = %d, want 3", got)
	}
}

func TestDefaultDoesNotPanic(t *testing.T) {
	if Default() == nil {
		t.Fatal("Default returned nil")
	}
}

// TestProviderMenuOrder locks the PS menu order: the YAML list order IS the
// order of the config and erase menus.
func TestProviderMenuOrder(t *testing.T) {
	want := []string{
		"openrouter", "requesty", "litellm",
		"deepseek", "zai", "dashscope", "minimax", "moonshot",
		"stepfun", "siliconflow", "mimo",
		"anthropic", "openai", "nvidia",
	}
	if got := Default().ProviderIDs(); !reflect.DeepEqual(got, want) {
		t.Errorf("provider order:\n got %v\nwant %v", got, want)
	}
}

// TestShortcutOrderPreserved verifies the YAML mapping order survives
// parsing (a plain map would randomize it).
func TestShortcutOrderPreserved(t *testing.T) {
	p, ok := Default().ProviderByID("openrouter")
	if !ok {
		t.Fatal("openrouter not found")
	}
	want := []string{"or-fusion", "codex-fusion", "oc-fusion", "ocqwen", "ockimi", "ocminimax"}
	if !reflect.DeepEqual(p.Shortcuts, want) {
		t.Errorf("openrouter shortcuts:\n got %v\nwant %v", p.Shortcuts, want)
	}
}

// TestVarMapParity spot-checks the shortcut->var extraction against the PS
// reference, including the single-key-multiple-vars ZAI case and the ceu
// gap fix.
func TestVarMapParity(t *testing.T) {
	tests := []struct {
		provider string
		shortcut string
		wantVar  string
	}{
		{"deepseek", "ds", "ANTHROPIC_AUTH_TOKEN"},
		{"deepseek", "cf", "ANTHROPIC_AUTH_TOKEN"},
		{"deepseek", "ocdeepseek", "DEEPSEEK_API_KEY"},
		{"zai", "cg", "ANTHROPIC_AUTH_TOKEN"},
		{"zai", "cgalt", "ANTHROPIC_API_KEY"},
		{"zai", "oczai", "ZAI_API_KEY"},
		{"requesty", "ceu", "REQUESTY_API_KEY"}, // gap fix vs PS catalog
		{"openrouter", "ockimi", "OPENROUTER_API_KEY"},
		{"moonshot", "ockimi-direct", "MOONSHOT_API_KEY"},
		{"anthropic", "ca", "ANTHROPIC_API_KEY"},
		{"nvidia", "nv-cc", "NVIDIA_API_KEY"},
		{"nvidia", "ocnvidia", "NVIDIA_API_KEY"},
	}
	c := Default()
	for _, tt := range tests {
		p, ok := c.ProviderByID(tt.provider)
		if !ok {
			t.Errorf("provider %q not found", tt.provider)
			continue
		}
		if got := p.VarMap[tt.shortcut]; got != tt.wantVar {
			t.Errorf("%s/%s var = %q, want %q", tt.provider, tt.shortcut, got, tt.wantVar)
		}
	}
}

// TestShortcutCount locks the 35 catalog shortcuts (31 PS + ceu + 3 nvidia).
// The 5 keyless profiles (co, codex55/54/mini, ocdefault) stay out on purpose.
func TestShortcutCount(t *testing.T) {
	total := 0
	for _, p := range Default().Providers {
		total += len(p.Shortcuts)
		if len(p.Shortcuts) != len(p.VarMap) {
			t.Errorf("provider %q: %d shortcuts but %d varmap entries", p.ID, len(p.Shortcuts), len(p.VarMap))
		}
	}
	if total != 35 {
		t.Errorf("total shortcuts = %d, want 35", total)
	}
}

func TestKeyPatternsEmptyForPSParity(t *testing.T) {
	// The PS reference has no per-provider format validation; the field is
	// reserved. If someone fills one in, this test forces a conscious update.
	for _, p := range Default().Providers {
		if p.KeyPattern != "" {
			t.Errorf("provider %q: unexpected key_pattern %q (PS has no format validation)", p.ID, p.KeyPattern)
		}
	}
}

func TestProviderByID(t *testing.T) {
	c := Default()
	tests := []struct {
		in     string
		wantID string
		wantOK bool
	}{
		{"anthropic", "anthropic", true},
		{"ANTHROPIC", "anthropic", true},     // case-insensitive
		{" openrouter ", "openrouter", true}, // trimmed
		{"nope", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		p, ok := c.ProviderByID(tt.in)
		if ok != tt.wantOK || p.ID != tt.wantID {
			t.Errorf("ProviderByID(%q) = (%q, %v), want (%q, %v)", tt.in, p.ID, ok, tt.wantID, tt.wantOK)
		}
	}
}

func TestRegionLabel(t *testing.T) {
	c := Default()
	if got := c.RegionLabel("global"); got != "Global / Agregateurs" {
		t.Errorf("RegionLabel(global) = %q", got)
	}
	if got := c.RegionLabel("unknown"); got != "unknown" {
		t.Errorf("RegionLabel(unknown) = %q, want fallback to id", got)
	}
}

// validYAML is a minimal correct document used as the base of mutation tests.
const validYAML = `
version: 1
regions:
  - id: global
    label: "Global"
providers:
  - id: acme
    display: "Acme"
    region: global
    console_url: "https://acme.test/keys"
    shortcuts:
      ac: ACME_API_KEY
`

func TestParseValidation(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string // substring expected in the error; "" = must succeed
	}{
		{"valid minimal", validYAML, ""},
		{"not yaml", "{{{", "cannot parse"},
		{"wrong version", strings.Replace(validYAML, "version: 1", "version: 99", 1), "unsupported schema version"},
		{"no regions", strings.Replace(validYAML, "  - id: global\n    label: \"Global\"\n", "", 1), "no regions"},
		{"empty region label", strings.Replace(validYAML, `label: "Global"`, `label: ""`, 1), "empty label"},
		{"accented region label", strings.Replace(validYAML, `label: "Global"`, `label: "Général"`, 1), "non-ASCII"},
		{"duplicate region", strings.Replace(validYAML, "regions:", "regions:\n  - id: global\n    label: \"Dup\"", 1), "duplicate region"},
		{"unknown provider region", strings.Replace(validYAML, "region: global", "region: mars", 1), "unknown region"},
		{"bad provider id", strings.Replace(validYAML, "id: acme", "id: Acme!", 1), "must match"},
		{"empty display", strings.Replace(validYAML, `display: "Acme"`, `display: ""`, 1), "empty display"},
		{"accented display", strings.Replace(validYAML, `display: "Acme"`, `display: "Acmé"`, 1), "non-ASCII"},
		{"bad url", strings.Replace(validYAML, `console_url: "https://acme.test/keys"`, `console_url: "ftp://x"`, 1), "not http(s)"},
		{"no shortcuts", strings.Replace(validYAML, "    shortcuts:\n      ac: ACME_API_KEY\n", "", 1), "no shortcuts"},
		{"bad shortcut name", strings.Replace(validYAML, "ac: ACME_API_KEY", "AC!: ACME_API_KEY", 1), "must match"},
		{"bad var name", strings.Replace(validYAML, "ac: ACME_API_KEY", "ac: lower_key", 1), "must match"},
		{"bad key_pattern", strings.Replace(validYAML, `console_url:`, "key_pattern: \"([\"\n    console_url:", 1), "invalid key_pattern"},
		{
			"duplicate provider id",
			validYAML + `
  - id: acme
    display: "Acme 2"
    region: global
    console_url: "https://acme2.test/keys"
    shortcuts:
      ac2: ACME2_API_KEY
`,
			"duplicate provider id",
		},
		{
			"shortcut claimed twice across providers",
			validYAML + `
  - id: other
    display: "Other"
    region: global
    console_url: "https://other.test/keys"
    shortcuts:
      ac: OTHER_API_KEY
`,
			"claimed by both",
		},
		{
			"shortcuts as list not mapping",
			strings.Replace(validYAML, "    shortcuts:\n      ac: ACME_API_KEY\n", "    shortcuts:\n      - ac\n", 1),
			"must be a mapping",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.yaml))
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err, tt.wantErr)
			}
		})
	}
}

// TestEmbeddedUIStringsAreCP850Safe double-checks the embedded data against
// the console convention (validate() enforces it, this documents it).
func TestEmbeddedUIStringsAreCP850Safe(t *testing.T) {
	c := Default()
	for _, r := range c.Regions {
		if !isCP850Safe(r.Label) {
			t.Errorf("region %q label not ASCII: %q", r.ID, r.Label)
		}
	}
	for _, p := range c.Providers {
		if !isCP850Safe(p.Display) {
			t.Errorf("provider %q display not ASCII: %q", p.ID, p.Display)
		}
		if !isCP850Safe(p.Note) {
			t.Errorf("provider %q note not ASCII: %q", p.ID, p.Note)
		}
	}
}
