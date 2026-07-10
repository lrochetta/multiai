// Package catalog holds the data-driven provider catalog embedded in the
// binary (providers.yaml). It is the single source of truth consumed by
// config, onboarding, erase and validation: adding a provider means editing
// the YAML, no code change.
//
// The catalog mirrors the PowerShell reference ($ProviderCatalog,
// code-router.ps1 L93-215) for functional parity with v0.3.0.
package catalog

import (
	_ "embed"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed providers.yaml
var embeddedYAML []byte

// SchemaVersion is the providers.yaml schema version this loader understands.
const SchemaVersion = 1

// Region is a display group for the config/erase menus.
type Region struct {
	ID    string `yaml:"id"`
	Label string `yaml:"label"`
	// Flag is an optional emoji; UIs targeting CP850 consoles may ignore it.
	Flag string `yaml:"flag"`
}

// Provider describes one API key provider: where the user creates the key
// and which env var receives it in each profile of the group.
type Provider struct {
	ID      string
	Display string
	Region  string
	URL     string
	Note    string
	// KeyPattern is an optional regex validating the key format. It is empty
	// for every provider today (the PS reference only detects placeholders);
	// the field is reserved for future opt-in validation.
	KeyPattern string
	// Shortcuts lists the profile shortcuts of the group, in YAML document
	// order (menus and status displays iterate this order).
	Shortcuts []string
	// VarMap maps each shortcut to the env var name that receives the key
	// in that profile's .env file.
	VarMap map[string]string
}

// Catalog is the parsed, validated providers.yaml.
type Catalog struct {
	Version   int        `yaml:"version"`
	Regions   []Region   `yaml:"regions"`
	Providers []Provider `yaml:"providers"`
}

// rawProvider keeps shortcuts as a yaml.Node so their document order can be
// preserved (a plain map[string]string would lose it).
type rawProvider struct {
	ID         string    `yaml:"id"`
	Display    string    `yaml:"display"`
	Region     string    `yaml:"region"`
	ConsoleURL string    `yaml:"console_url"`
	Note       string    `yaml:"note"`
	KeyPattern string    `yaml:"key_pattern"`
	Shortcuts  yaml.Node `yaml:"shortcuts"`
}

// UnmarshalYAML decodes a provider entry, preserving shortcut order.
func (p *Provider) UnmarshalYAML(node *yaml.Node) error {
	var raw rawProvider
	if err := node.Decode(&raw); err != nil {
		return err
	}
	p.ID = raw.ID
	p.Display = raw.Display
	p.Region = raw.Region
	p.URL = raw.ConsoleURL
	p.Note = raw.Note
	p.KeyPattern = raw.KeyPattern
	p.Shortcuts = nil
	p.VarMap = make(map[string]string)

	if raw.Shortcuts.Kind == 0 || raw.Shortcuts.Tag == "!!null" {
		return nil // validated later (shortcuts required)
	}
	if raw.Shortcuts.Kind != yaml.MappingNode {
		return fmt.Errorf("provider %q: shortcuts must be a mapping", raw.ID)
	}
	for i := 0; i+1 < len(raw.Shortcuts.Content); i += 2 {
		name := raw.Shortcuts.Content[i].Value
		varName := raw.Shortcuts.Content[i+1].Value
		if _, dup := p.VarMap[name]; dup {
			return fmt.Errorf("provider %q: duplicate shortcut %q", raw.ID, name)
		}
		p.Shortcuts = append(p.Shortcuts, name)
		p.VarMap[name] = varName
	}
	return nil
}

// Parse decodes and validates a providers.yaml document.
func Parse(data []byte) (*Catalog, error) {
	var c Catalog
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("catalog: cannot parse providers.yaml: %w", err)
	}
	if err := c.validate(); err != nil {
		return nil, fmt.Errorf("catalog: invalid providers.yaml: %w", err)
	}
	return &c, nil
}

// Load parses and validates the embedded providers.yaml.
func Load() (*Catalog, error) {
	return Parse(embeddedYAML)
}

var defaultCatalog = sync.OnceValues(Load)

// Default returns the embedded catalog, parsed once. It panics if the
// embedded YAML is invalid â€” a programmer error caught by the package tests
// at build time, never by end users.
func Default() *Catalog {
	c, err := defaultCatalog()
	if err != nil {
		panic(err)
	}
	return c
}

// ProviderByID returns the provider with the given id (case-insensitive).
func (c *Catalog) ProviderByID(id string) (Provider, bool) {
	id = strings.TrimSpace(id)
	for _, p := range c.Providers {
		if strings.EqualFold(p.ID, id) {
			return p, true
		}
	}
	return Provider{}, false
}

// ProviderIDs returns all provider ids in menu order.
func (c *Catalog) ProviderIDs() []string {
	ids := make([]string, len(c.Providers))
	for i, p := range c.Providers {
		ids[i] = p.ID
	}
	return ids
}

// RegionLabel returns the display label for a region id, falling back to the
// id itself if unknown.
func (c *Catalog) RegionLabel(id string) string {
	for _, r := range c.Regions {
		if r.ID == id {
			return r.Label
		}
	}
	return id
}

var (
	idRe  = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	varRe = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
)

// validate enforces the schema invariants the consumers rely on.
func (c *Catalog) validate() error {
	if c.Version != SchemaVersion {
		return fmt.Errorf("unsupported schema version %d (want %d)", c.Version, SchemaVersion)
	}
	if len(c.Regions) == 0 {
		return fmt.Errorf("no regions defined")
	}
	regionIDs := make(map[string]bool, len(c.Regions))
	for _, r := range c.Regions {
		if r.ID == "" {
			return fmt.Errorf("region with empty id")
		}
		if regionIDs[r.ID] {
			return fmt.Errorf("duplicate region id %q", r.ID)
		}
		regionIDs[r.ID] = true
		if strings.TrimSpace(r.Label) == "" {
			return fmt.Errorf("region %q: empty label", r.ID)
		}
		if !isCP850Safe(r.Label) {
			return fmt.Errorf("region %q: label contains non-ASCII characters (CP850 console convention)", r.ID)
		}
	}

	if len(c.Providers) == 0 {
		return fmt.Errorf("no providers defined")
	}
	providerIDs := make(map[string]bool, len(c.Providers))
	shortcutOwner := make(map[string]string) // shortcut -> provider id
	for _, p := range c.Providers {
		if !idRe.MatchString(p.ID) {
			return fmt.Errorf("provider id %q: must match %s", p.ID, idRe)
		}
		if providerIDs[p.ID] {
			return fmt.Errorf("duplicate provider id %q", p.ID)
		}
		providerIDs[p.ID] = true
		if strings.TrimSpace(p.Display) == "" {
			return fmt.Errorf("provider %q: empty display", p.ID)
		}
		if !isCP850Safe(p.Display) || !isCP850Safe(p.Note) {
			return fmt.Errorf("provider %q: display/note contains non-ASCII characters (CP850 console convention)", p.ID)
		}
		if !regionIDs[p.Region] {
			return fmt.Errorf("provider %q: unknown region %q", p.ID, p.Region)
		}
		if !strings.HasPrefix(p.URL, "https://") && !strings.HasPrefix(p.URL, "http://") {
			return fmt.Errorf("provider %q: console_url %q is not http(s)", p.ID, p.URL)
		}
		if len(p.Shortcuts) == 0 {
			return fmt.Errorf("provider %q: no shortcuts", p.ID)
		}
		for _, sc := range p.Shortcuts {
			if !idRe.MatchString(sc) {
				return fmt.Errorf("provider %q: shortcut %q must match %s", p.ID, sc, idRe)
			}
			if owner, taken := shortcutOwner[sc]; taken {
				return fmt.Errorf("shortcut %q claimed by both %q and %q", sc, owner, p.ID)
			}
			shortcutOwner[sc] = p.ID
			if !varRe.MatchString(p.VarMap[sc]) {
				return fmt.Errorf("provider %q: shortcut %q: env var %q must match %s", p.ID, sc, p.VarMap[sc], varRe)
			}
		}
		if p.KeyPattern != "" {
			if _, err := regexp.Compile(p.KeyPattern); err != nil {
				return fmt.Errorf("provider %q: invalid key_pattern: %v", p.ID, err)
			}
		}
	}
	return nil
}

// isCP850Safe reports whether s only contains printable ASCII (safe on
// Windows CP850 consoles, per the product convention "French, no accents").
func isCP850Safe(s string) bool {
	for _, r := range s {
		if r < 0x20 || r > 0x7E {
			return false
		}
	}
	return true
}
