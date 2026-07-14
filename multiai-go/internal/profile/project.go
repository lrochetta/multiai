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
	store, err := DefaultProjectTrustStore()
	if err != nil {
		return nil, "", err
	}
	return FindProjectConfigWithTrustStore(store)
}

// FindProjectConfigWithTrustStore is FindProjectConfig with an explicit trust
// store. It is primarily useful for tests and embedded callers. A discovered
// configuration is returned only when its canonical path and exact SHA-256 are
// approved; untrusted, changed, or corrupt-store states fail closed.
func FindProjectConfigWithTrustStore(store *ProjectTrustStore) (*ProfileYAML, string, error) {
	if store == nil {
		return nil, "", fmt.Errorf("project trust store is nil")
	}
	dir, err := os.Getwd()
	if err != nil {
		return nil, "", err
	}
	return findProjectConfigFrom(dir, store)
}

// FindProjectConfigPath locates the nearest project configuration without
// parsing or trusting it. It exists so the CLI can show, trust, or untrust the
// exact file before any project-controlled value is applied.
func FindProjectConfigPath() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return findProjectConfigPathFrom(dir)
}

func findProjectConfigFrom(dir string, store *ProjectTrustStore) (*ProfileYAML, string, error) {
	configPath, err := findProjectConfigPathFrom(dir)
	if err != nil || configPath == "" {
		return nil, configPath, err
	}

	status, data, inspectErr := store.inspectWithContent(configPath)
	if inspectErr != nil {
		return nil, status.CanonicalPath, inspectErr
	}
	if trustErr := trustError(status); trustErr != nil {
		return nil, status.CanonicalPath, trustErr
	}

	var py ProfileYAML
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&py); err != nil {
		return nil, status.CanonicalPath, fmt.Errorf("cannot parse %s: %w", status.CanonicalPath, err)
	}
	return &py, status.CanonicalPath, nil
}

func findProjectConfigPathFrom(dir string) (string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve project search directory: %w", err)
	}
	for {
		for _, name := range []string{".multiai.yaml", ".multiai.yml"} {
			configPath := filepath.Join(dir, name)
			if _, statErr := os.Stat(configPath); statErr != nil {
				if os.IsNotExist(statErr) {
					continue
				}
				return configPath, fmt.Errorf("cannot inspect %s: %w", configPath, statErr)
			}
			return configPath, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached root
		}
		dir = parent
	}

	return "", nil // no config found
}

// MergeProjectConfig merges a project config override on top of a base profile.
func MergeProjectConfig(base *Profile, project *ProfileYAML) *Profile {
	merged := *base
	merged.Args = append([]string(nil), base.Args...)
	merged.RequiredSecrets = append([]string(nil), base.RequiredSecrets...)
	merged.Fallback = append([]string(nil), base.Fallback...)
	if base.Env != nil {
		merged.Env = make(map[string]string, len(base.Env))
		for key, value := range base.Env {
			merged.Env[key] = value
		}
	}
	merged.Hooks = cloneHooksConfig(base.Hooks)

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
		merged.Args = append([]string(nil), project.Args...)
	}
	if project.Hooks != nil {
		merged.Hooks = cloneHooksConfig(project.Hooks)
	}

	return &merged
}

func cloneHooksConfig(source *HooksConfig) *HooksConfig {
	if source == nil {
		return nil
	}
	return &HooksConfig{
		BeforeLaunch: append([]HookCommand(nil), source.BeforeLaunch...),
		AfterLaunch:  append([]HookCommand(nil), source.AfterLaunch...),
	}
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
