package profile

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// FuzzParseYAML fuzzes YAML parsing followed by yamlToProfile.
// It must never panic for any input.
//
//go:noinline
func FuzzParseYAML(f *testing.F) {
	// Seed corpus: representative YAML profile content
	seeds := []string{
		// Valid minimal YAML profile
		"id: test\nshortcut: t\ntool: claude\nenv:\n  KEY: val\n",
		// YAML with all fields
		`id: full
shortcut: f
tool: codex
display_name: "Full Profile"
description: "A full YAML profile"
order: 5
command: codex
args: ["--verbose", "--model", "gpt4"]
env:
  API_KEY: sk-test
  REGION: eu
hooks:
  before_launch:
    - command: echo ready
  after_launch:
    - command: echo done
`,
		// Empty YAML
		"",
		"{}",
		// Minimal env
		"id: min\nenv: {}\n",
		// Invalid YAML
		"\t...\n{{{\n",
		// Just a string â€” valid YAML but not a mapping
		"just a string",
		// Number
		"42",
		// List
		"- a\n- b\n",
		// Nested structures
		"id: nested\nenv:\n  outer:\n    inner: value\n",
	}

	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		var py ProfileYAML

		// Unmarshal the YAML into a ProfileYAML structure.
		if err := yaml.Unmarshal(data, &py); err != nil {
			// Invalid YAML or not a mapping â€” skip (LoadYAML would return
			// an error at this stage and never call yamlToProfile).
			return
		}

		// yamlToProfile must never panic for any ProfileYAML, even
		// partially populated.
		result := yamlToProfile(&py, "/fuzz/test.yaml")

		// Basic sanity: result must never be nil
		if result == nil {
			t.Error("yamlToProfile returned nil")
		}
	})
}
