package profile

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// FindProjectConfig looks for .multiai.yaml or .multiai.yml in current and ancestor directories.
func FindProjectConfig() (*ProfileYAML, string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, "", err
	}

	for {
		for _, name := range []string{".multiai.yaml", ".multiai.yml"} {
			configPath := filepath.Join(dir, name)
			if _, err := os.Stat(configPath); err == nil {
				data, err := os.ReadFile(configPath)
				if err != nil {
					return nil, "", fmt.Errorf("cannot read %s: %w", configPath, err)
				}
				const maxYAMLSize = 1 << 20 // 1 Mo max
				if len(data) > maxYAMLSize {
					return nil, "", fmt.Errorf("yaml file too large: %s (%d bytes, max %d)", configPath, len(data), maxYAMLSize)
				}
				var py ProfileYAML
				decoder := yaml.NewDecoder(bytes.NewReader(data))
				if err := decoder.Decode(&py); err != nil {
					return nil, "", fmt.Errorf("cannot parse %s: %w", configPath, err)
				}
				return &py, configPath, nil
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached root
		}
		dir = parent
	}

	return nil, "", nil // no config found
}

// MergeProjectConfig merges a project config override on top of a base profile.
func MergeProjectConfig(base *Profile, project *ProfileYAML) *Profile {
	merged := *base // shallow copy

	if project.DisplayName != "" {
		merged.DisplayName = project.DisplayName
	}
	if len(project.Overrides) > 0 {
		if merged.Env == nil {
			merged.Env = make(map[string]string)
		}
		for k, v := range project.Overrides {
			merged.Env[k] = os.ExpandEnv(v)
		}
	}
	if project.ClearEnv != nil {
		merged.ClearEnv = *project.ClearEnv
	}
	if len(project.Args) > 0 {
		merged.Args = project.Args
	}
	if project.Hooks != nil {
		merged.Hooks = project.Hooks
	}

	return &merged
}

// ValidateProfileYAML validates a YAML profile file and returns warnings.
func ValidateProfileYAML(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var py ProfileYAML
	if err := yaml.Unmarshal(data, &py); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}

	var warnings []string
	if py.ID == "" && py.Extends == "" {
		warnings = append(warnings, "YAML: 'id' or 'extends' recommended")
	}
	if py.Tool == "" && py.Extends == "" {
		warnings = append(warnings, "YAML: 'tool' recommended (claude, codex, opencode)")
	}
	if py.Command != "" && !isAllowedCommand(py.Command) {
		warnings = append(warnings, fmt.Sprintf("YAML: command '%s' not in whitelist (claude, codex, opencode)", py.Command))
	}

	return warnings, nil
}

// isAllowedCommand checks if a command name is in the allowed list.
func isAllowedCommand(cmd string) bool {
	allowed := map[string]bool{"claude": true, "codex": true, "opencode": true}
	return allowed[cmd]
}
