package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/lrochetta/multiai/internal/profile"
)

// RunBeforeHooks executes all before_launch hooks.
// If any hook fails, the launch is aborted.
func RunBeforeHooks(hooks *profile.HooksConfig, prof *profile.Profile) error {
	if hooks == nil || len(hooks.BeforeLaunch) == 0 {
		return nil
	}

	fmt.Fprintf(os.Stderr, "\n[Hooks] Executing before_launch hooks (%d)...\n", len(hooks.BeforeLaunch))

	for i, hook := range hooks.BeforeLaunch {
		cmdStr := expandHookVars(hook.Command, prof)
		fmt.Fprintf(os.Stderr, "  [%d/%d] %s\n", i+1, len(hooks.BeforeLaunch), cmdStr)

		shell := hook.Shell
		if shell == "" {
			shell = defaultShell()
		}

		var cmd *exec.Cmd
		switch shell {
		case "powershell", "pwsh":
			cmd = exec.Command("powershell", "-NoProfile", "-Command", cmdStr)
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
		fmt.Fprintf(os.Stderr, "  [%d/%d] %s\n", i+1, len(hooks.AfterLaunch), cmdStr)

		shell := hook.Shell
		if shell == "" {
			shell = defaultShell()
		}

		var cmd *exec.Cmd
		switch shell {
		case "powershell", "pwsh":
			cmd = exec.Command("powershell", "-NoProfile", "-Command", cmdStr)
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
func expandHookVars(cmd string, prof *profile.Profile) string {
	result := cmd
	result = strings.ReplaceAll(result, "{{.Profile.ID}}", prof.ID)
	result = strings.ReplaceAll(result, "{{.Profile.Shortcut}}", prof.Shortcut)
	result = strings.ReplaceAll(result, "{{.Profile.DisplayName}}", prof.DisplayName)
	result = strings.ReplaceAll(result, "{{.Profile.Tool}}", prof.Tool)
	result = strings.ReplaceAll(result, "{{.Profile.Command}}", prof.Command)
	result = os.ExpandEnv(result)
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
