package profile

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ProfileYAML is the YAML representation of a profile (for .yaml/.multiai.yaml files).
type ProfileYAML struct {
	ID              string            `yaml:"id" json:"id"`
	Shortcut        string            `yaml:"shortcut" json:"shortcut"`
	Tool            string            `yaml:"tool" json:"tool"`
	ToolLabel       string            `yaml:"tool_label,omitempty" json:"tool_label,omitempty"`
	DisplayName     string            `yaml:"display_name" json:"display_name"`
	Description     string            `yaml:"description,omitempty" json:"description,omitempty"`
	Order           int               `yaml:"order,omitempty" json:"order,omitempty"`
	Command         string            `yaml:"command,omitempty" json:"command,omitempty"`
	Args            []string          `yaml:"args,omitempty" json:"args,omitempty"`
	Env             map[string]string `yaml:"env" json:"env"`
	ClearEnv        *bool             `yaml:"clear_env,omitempty" json:"clear_env,omitempty"`
	RequiredSecrets []string          `yaml:"required_secrets,omitempty" json:"required_secrets,omitempty"`
	Provider        string            `yaml:"provider,omitempty" json:"provider,omitempty"`

	// Project profile (.multiai.yaml)
	Extends   string            `yaml:"extends,omitempty" json:"extends,omitempty"`
	Overrides map[string]string `yaml:"overrides,omitempty" json:"overrides,omitempty"`

	// Plugin hooks
	Hooks *HooksConfig `yaml:"hooks,omitempty" json:"hooks,omitempty"`
}

// HooksConfig defines commands to run before/after launch.
type HooksConfig struct {
	BeforeLaunch []HookCommand `yaml:"before_launch,omitempty" json:"before_launch,omitempty"`
	AfterLaunch  []HookCommand `yaml:"after_launch,omitempty" json:"after_launch,omitempty"`
}

// HookCommand is a single hook execution command.
type HookCommand struct {
	Command string `yaml:"command" json:"command"`
	Shell   string `yaml:"shell,omitempty" json:"shell,omitempty"`
}

// LoadYAML loads a single YAML profile file.
func LoadYAML(path string) (*Profile, error) {
	if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
		return nil, fmt.Errorf("not a YAML file: %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", path, err)
	}

	const maxYAMLSize = 1 << 20 // 1 Mo max
	if len(data) > maxYAMLSize {
		return nil, fmt.Errorf("YAML file too large: %s (%d bytes, max %d)", path, len(data), maxYAMLSize)
	}

	var py ProfileYAML
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&py); err != nil {
		return nil, fmt.Errorf("cannot parse %s: %w", path, err)
	}

	return yamlToProfile(&py, path), nil
}

// LoadDirYAML loads all .yaml/.yml profiles from a directory.
func LoadDirYAML(dir string) ([]Profile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot read profiles directory %s: %w", dir, err)
	}

	var profiles []Profile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
			fullPath := filepath.Join(dir, name)
			p, err := LoadYAML(fullPath)
			if err != nil {
				continue // skip invalid files
			}
			profiles = append(profiles, *p)
		}
	}
	return profiles, nil
}

// LoadAllProfiles loads both .env and .yaml profiles from a directory.
func LoadAllProfiles(dir string) ([]Profile, error) {
	var all []Profile

	envProfiles, err := LoadDir(dir)
	if err == nil {
		all = append(all, envProfiles...)
	}

	yamlProfiles, err := LoadDirYAML(dir)
	if err == nil {
		all = append(all, yamlProfiles...)
	}

	// Sort by Tool, Order, DisplayName
	sortProfilesSlice(all)
	return all, nil
}

// sortProfilesSlice sorts profiles in place.
func sortProfilesSlice(profiles []Profile) {
	sort.Slice(profiles, func(i, j int) bool {
		pi, pj := profiles[i], profiles[j]
		if pi.Tool != pj.Tool { return pi.Tool < pj.Tool }
		if pi.Order != pj.Order { return pi.Order < pj.Order }
		return pi.DisplayName < pj.DisplayName
	})
}

// yamlToProfile converts a YAML profile to the internal Profile type.
func yamlToProfile(py *ProfileYAML, path string) *Profile {
	p := &Profile{
		Path:            path,
		ID:              py.ID,
		Shortcut:        py.Shortcut,
		Tool:            py.Tool,
		ToolLabel:       py.ToolLabel,
		DisplayName:     py.DisplayName,
		Description:     py.Description,
		Order:           py.Order,
		Command:         py.Command,
		Args:            py.Args,
		Env:             py.Env,
		RequiredSecrets: py.RequiredSecrets,
	}

	// Apply defaults
	if p.ToolLabel == "" {
		p.ToolLabel = p.Tool
	}
	if p.Command == "" {
		p.Command = p.Tool
	}
	if p.ID == "" {
		p.ID = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	if p.Shortcut == "" {
		p.Shortcut = p.ID
	}
	if p.DisplayName == "" {
		p.DisplayName = p.ID
	}
	p.ClearEnv = true
	if py.ClearEnv != nil {
		p.ClearEnv = *py.ClearEnv
	}
	if p.Env == nil {
		p.Env = make(map[string]string)
	}

	return p
}
