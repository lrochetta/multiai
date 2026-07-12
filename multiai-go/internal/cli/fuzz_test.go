package cli

import (
	"strings"
	"testing"

	"github.com/lrochetta/multiai/internal/profile"
)

// FuzzLaunchArgs fuzzes CLI argument parsing, command validation, and
// string-escaping functions. It must never panic for any input.
//
//go:noinline
func FuzzLaunchArgs(f *testing.F) {
	// Seed corpus: representative command names, args, and shell types
	seeds := []struct {
		cmd   string
		shell string
		arg   string
	}{
		{"claude", "bash", "--help"},
		{"codex", "powershell", "--model gpt4"},
		{"opencode", "zsh", "-v"},
		{"unknown-tool", "fish", ""},
		{"", "cmd", strings.Repeat("A", 100)},
		{"claude", "", "\"; rm -rf /; \""},
		{"codex", "pwsh", "`$PATH`"},
	}

	for _, s := range seeds {
		f.Add(s.cmd, s.shell, s.arg)
	}

	f.Fuzz(func(t *testing.T, cmd, shell, arg string) {
		// ---- Part 1: IsCommandAllowed ----
		// Must never panic and be deterministic.
		result1 := IsCommandAllowed(cmd)
		result2 := IsCommandAllowed(cmd)
		if result1 != result2 {
			t.Errorf("IsCommandAllowed(%q) non-deterministic: %v then %v", cmd, result1, result2)
		}

		// Known commands must always be allowed.
		if cmd == "claude" || cmd == "codex" || cmd == "opencode" {
			if !result1 {
				t.Errorf("IsCommandAllowed(%q) = false, want true", cmd)
			}
		}

		// ---- Part 2: GenerateCompletion ----
		// Must never panic for any shell name.
		_ = GenerateCompletion(shell)

		// ---- Part 3: escapeShellArg ----
		// Must never panic for any shell type.
		for _, s := range []string{"bash", "zsh", "powershell", "pwsh", "cmd", "", shell} {
			escaped := escapeShellArg(arg, s)
			// Escaped string must not be empty if input is not empty.
			if arg != "" && escaped == "" {
				t.Errorf("escapeShellArg(%q, %q) returned empty string", arg, s)
			}
			// Determinism check.
			escaped2 := escapeShellArg(arg, s)
			if escaped != escaped2 {
				t.Errorf("escapeShellArg(%q, %q) non-deterministic", arg, s)
			}
		}

		// ---- Part 4: jsonError ----
		// Must never panic and always return valid JSON-ish output.
		jsonStr := jsonError(cmd + arg)
		if !strings.HasPrefix(jsonStr, `{"status":"error","error":"`) {
			t.Errorf("jsonError(%q) = %q, expected JSON error format", cmd+arg, jsonStr)
		}
		if !strings.HasSuffix(jsonStr, `"}`) {
			t.Errorf("jsonError(%q) = %q, missing closing", cmd+arg, jsonStr)
		}

		// ---- Part 5: isInterruptExit ----
		// Must never panic for any int (exit code).
		_ = isInterruptExit(len(cmd) + len(arg))

		// ---- Part 6: expandHookVars ----
		// Must never panic with random template strings.
		prof := &profile.Profile{
			ID:          cmd,
			Shortcut:    shell,
			DisplayName: arg,
			Tool:        cmd + "_tool",
			Command:     cmd,
		}
		_ = expandHookVars(arg, prof)
	})
}

// FuzzHooksTemplates fuzzes expandHookVars with random template strings
// and profile data, ensuring it never panics.
//
//go:noinline
func FuzzHooksTemplates(f *testing.F) {
	seeds := []string{
		"{{.Profile.ID}}",
		"{{.Profile.Shortcut}}",
		"{{.Profile.DisplayName}}",
		"{{.Profile.Tool}}",
		"{{.Profile.Command}}",
		"before launch for {{.Profile.Shortcut}}",
		"{{.Profile.ID}}-{{.Profile.Shortcut}}",
		"echo {{.Profile.Command}} --verbose",
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, template string) {
		prof := &profile.Profile{
			ID:          "test",
			Shortcut:    "t",
			DisplayName: "Test Profile",
			Tool:        "claude",
			Command:     "claude",
		}
		result := expandHookVars(template, prof)
		// Must never return empty when input is non-empty and contains
		// only template variables.
		if template != "" && strings.TrimSpace(result) == "" {
			t.Errorf("expandHookVars(%q) returned empty result", template)
		}
	})
}
