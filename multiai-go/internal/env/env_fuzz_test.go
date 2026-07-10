package env

import (
	"strings"
	"testing"
)

// FuzzExpandProfileEnv fuzzes ExpandProfileEnv with a map built from
// newline-separated KEY=VALUE pairs. It must never panic for any input.
//
//go:noinline
func FuzzExpandProfileEnv(f *testing.F) {
	// Seed corpus: representative environment variable patterns
	seeds := []string{
		// Standard variables
		"API_KEY=sk-test123",
		// With %VAR% reference
		"AUTH_TOKEN=%API_KEY%",
		// Nested references
		"A=hello\nB=%A% world\nC=%B%!",
		// Unknown variable — kept literal
		"PATH=%UNKNOWN_VAR%",
		// Multiple lines with a mix
		"KEY1=val1\nKEY2=%KEY1%\nKEY3=plain",
		// Self-reference (cyclic)
		"A=%A%",
		// Mutual cycle
		"A=%B%\nB=%A%",
		// Empty value
		"EMPTY=",
		// Single line
		"SINGLE=value",
		// With allowed system var reference
		"MY_PATH=%PATH%",
		// Special characters in values
		"SPECIAL=hello world! @#$%^&*()",
		// Multiple % in one value
		"MULTI=%A%_%B%_%C%",
		// Deep nesting
		"A=1\nB=%A%\nC=%B%\nD=%C%\nE=%D%\nF=%E%\nG=%F%\nH=%G%\nI=%H%\nJ=%I%\nK=%J%",
		// Empty input
		"",
		// Only whitespace
		"   \n  \n",
		// Invalid key format
		"=value\n123invalid=test\n",
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, raw string) {
		// Build profileEnv from newline-separated KEY=VALUE lines.
		profileEnv := make(map[string]string)
		lines := strings.Split(raw, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			idx := strings.Index(line, "=")
			if idx < 1 {
				continue
			}
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])
			if key == "" {
				continue
			}
			profileEnv[key] = value
		}

		// ExpandProfileEnv must never panic for any input map.
		result := ExpandProfileEnv(profileEnv)

		// Basic invariant: result must have the same keys as input,
		// and no key must be missing or empty in an unexpected way.
		if len(result) != len(profileEnv) {
			t.Errorf("result has %d keys, input has %d", len(result), len(profileEnv))
		}
		for k := range profileEnv {
			if _, ok := result[k]; !ok {
				t.Errorf("key %q missing from result", k)
			}
		}
	})
}
