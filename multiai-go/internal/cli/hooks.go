package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/lrochetta/multiai/internal/profile"
)

// escapeShellArg escapes shell-special characters in a string for the given shell.
func escapeShellArg(s string, shell string) string {
	switch shell {
	case "powershell", "pwsh":
		// PowerShell: escape backticks and double-quotes
		s = strings.ReplaceAll(s, "`", "``")
		s = strings.ReplaceAll(s, "\"", "`\"")
		return s
	case "bash", "zsh":
		// Bash/zsh: escape shell metacharacters
		replacer := strings.NewReplacer(
			";", "\\;", "&", "\\&", "|", "\\|",
			"$", "\\$", "`", "\\`", "\\", "\\\\",
			"\"", "\\\"", "'", "\\'",
		)
		return replacer.Replace(s)
	default:
		// cmd.exe: escape & | < > ^ %
		replacer := strings.NewReplacer(
			"&", "^&", "|", "^|", "<", "^<", ">", "^>",
			"^", "^^", "%", "%%",
		)
		return replacer.Replace(s)
	}
}

// RunBeforeHooks executes all before_launch hooks.
// If any hook fails, the launch is aborted.
func RunBeforeHooks(hooks *profile.HooksConfig, prof *profile.Profile) error {
	if hooks == nil || len(hooks.BeforeLaunch) == 0 {
		return nil
	}

	fmt.Fprintf(os.Stderr, "\n[Hooks] Executing before_launch hooks (%d)...\n", len(hooks.BeforeLaunch))

	for i, hook := range hooks.BeforeLaunch {
		cmdStr := expandHookVars(hook.Command, prof)
		shell := hook.Shell
		if shell == "" {
			shell = defaultShell()
		}
		// Expand env FIRST, so injected env var values are escaped below
		cmdStr = os.ExpandEnv(cmdStr)
		// Escape shell metacharacters to prevent injection (including expanded env values)
		cmdStr = escapeShellArg(cmdStr, shell)

		fmt.Fprintf(os.Stderr, "  [%d/%d] %s\n", i+1, len(hooks.BeforeLaunch), cmdStr)

		var cmd *exec.Cmd
		switch shell {
		case "powershell":
			cmd = exec.Command("powershell", "-NoProfile", "-Command", cmdStr)
		case "pwsh":
			cmd = exec.Command("pwsh", "-NoProfile", "-Command", cmdStr)
		case "bash":
			cmd = exec.Command("bash", "-c", cmdStr)
		case "zsh":
			cmd = exec.Command("zsh", "-c", cmdStr)
		default:
			if runtime.GOOS == "windows" {
				cmd = exec.Command("cmd", "/c", cmdStr)
			} else {
				cmd = exec.Command("sh", "-c", cmdStr)
			}
		}

		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("hook before_launch[%d] failed: %w\n  Command: %s", i, err, cmdStr)
		}
	}

	return nil
}

// RunAfterHooks executes all after_launch hooks (best-effort, errors are logged).
func RunAfterHooks(hooks *profile.HooksConfig, prof *profile.Profile, launchErr error) {
	if hooks == nil || len(hooks.AfterLaunch) == 0 {
		return
	}

	fmt.Fprintf(os.Stderr, "\n[Hooks] Executing after_launch hooks (%d)...\n", len(hooks.AfterLaunch))

	for i, hook := range hooks.AfterLaunch {
		cmdStr := expandHookVars(hook.Command, prof)
		shell := hook.Shell
		if shell == "" {
			shell = defaultShell()
		}
		// Expand env FIRST, so injected env var values are escaped below
		cmdStr = os.ExpandEnv(cmdStr)
		// Escape shell metacharacters to prevent injection (including expanded env values)
		cmdStr = escapeShellArg(cmdStr, shell)

		fmt.Fprintf(os.Stderr, "  [%d/%d] %s\n", i+1, len(hooks.AfterLaunch), cmdStr)

		var cmd *exec.Cmd
		switch shell {
		case "powershell":
			cmd = exec.Command("powershell", "-NoProfile", "-Command", cmdStr)
		case "pwsh":
			cmd = exec.Command("pwsh", "-NoProfile", "-Command", cmdStr)
		case "bash":
			cmd = exec.Command("bash", "-c", cmdStr)
		case "zsh":
			cmd = exec.Command("zsh", "-c", cmdStr)
		default:
			if runtime.GOOS == "windows" {
				cmd = exec.Command("cmd", "/c", cmdStr)
			} else {
				cmd = exec.Command("sh", "-c", cmdStr)
			}
		}

		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "  [WARN] after_launch hook failed (non-blocking): %v\n", err)
		}
	}
}

// expandHookVars expands template variables in hook commands.
// Note: os.ExpandEnv is called BEFORE escapeShellArg, not here,
// so that any injected shell metacharacters in env values get escaped.
func expandHookVars(cmd string, prof *profile.Profile) string {
	result := cmd
	result = strings.ReplaceAll(result, "{{.Profile.ID}}", prof.ID)
	result = strings.ReplaceAll(result, "{{.Profile.Shortcut}}", prof.Shortcut)
	result = strings.ReplaceAll(result, "{{.Profile.DisplayName}}", prof.DisplayName)
	result = strings.ReplaceAll(result, "{{.Profile.Tool}}", prof.Tool)
	result = strings.ReplaceAll(result, "{{.Profile.Command}}", prof.Command)
	return result
}

// defaultShell returns the default shell for the current OS.
func defaultShell() string {
	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("powershell"); err == nil {
			return "powershell"
		}
		return "cmd"
	}
	if _, err := exec.LookPath("bash"); err == nil {
		return "bash"
	}
	return "sh"
}
