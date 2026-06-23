package cli

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/lrochetta/multiai/internal/env"
	"github.com/lrochetta/multiai/internal/profile"
	"github.com/lrochetta/multiai/pkg/dotenv"
)

// AllowedCommands is the immutable list of binaries allowed by default.
var AllowedCommands = []string{"claude", "codex", "opencode"}

// IsCommandAllowed checks whether a command is in the whitelist.
func IsCommandAllowed(cmd string) bool {
	for _, allowed := range AllowedCommands {
		if cmd == allowed {
			return true
		}
	}
	return false
}

// LaunchOptions configures how a CLI is launched.
type LaunchOptions struct {
	DryRun             bool
	NoLaunch           bool
	ShowEnv            bool
	JSON               bool
	AllowCustomCommand bool
	ExtraArgs          []string
	Hooks              *profile.HooksConfig
}

// LaunchResult contains information about the launched process.
type LaunchResult struct {
	Profile   string `json:"profile"`
	Shortcut  string `json:"shortcut"`
	Tool      string `json:"tool"`
	Command   string `json:"command"`
	Args      string `json:"args"`
	Status    string `json:"status"`
	PID       int    `json:"pid,omitempty"`
	ExitCode  int    `json:"exit_code,omitempty"`
	Timestamp string `json:"timestamp"`
}

// ValidateAndLaunch checks the profile and launches the CLI.
func ValidateAndLaunch(prof *profile.Profile, opts LaunchOptions) (*LaunchResult, error) {
	// 1. Validate command whitelist
	if !IsCommandAllowed(prof.Command) {
		if opts.AllowCustomCommand {
			fmt.Fprintf(os.Stderr, "⚠ Commande custom autorisee : %s\n", prof.Command)
		} else {
			return nil, fmt.Errorf("commande non autorisee : '%s'. Utilise -AllowCustomCommand pour autoriser.", prof.Command)
		}
	}

	// 2. Check command exists in PATH
	if _, err := exec.LookPath(prof.Command); err != nil {
		return nil, fmt.Errorf("commande introuvable : '%s'. Installe le CLI correspondant.", prof.Command)
	}

	// 3. Validate required secrets
	if err := validateSecrets(prof); err != nil {
		return nil, err
	}

	// 4. Build environment
	cmdEnv := env.BuildCleanEnv(prof.Env)

	// 5. Prepare command
	allArgs := append(prof.Args, opts.ExtraArgs...)

	result := &LaunchResult{
		Profile:   prof.DisplayName,
		Shortcut:  prof.Shortcut,
		Tool:      prof.ToolLabel,
		Command:   fmt.Sprintf("%s %s", prof.Command, strings.Join(allArgs, " ")),
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// 6. Show env if requested
	if opts.ShowEnv {
		ShowEffectiveEnv(prof, opts.JSON)
	}

	// 7. Dry run
	if opts.DryRun {
		result.Status = "dry_run"
		fmt.Printf("\n[DRY RUN] Simulation sans lancement\n")
		fmt.Printf("[DRY RUN] Commande : %s\n", result.Command)
		return result, nil
	}

	// 8. No launch
	if opts.NoLaunch {
		result.Status = "no_launch"
		return result, nil
	}

	// 8.5. Run before-launch hooks
	if opts.Hooks != nil {
		if err := RunBeforeHooks(opts.Hooks, prof); err != nil {
			return nil, err
		}
	}

	// 9. Launch

	// Set up signal forwarding to child process
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	defer close(sigCh)

	cmd := exec.Command(prof.Command, allArgs...)
	cmd.Env = cmdEnv
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("\nLancement : %s\n", result.Command)

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("erreur de lancement : %w", err)
	}

	// Forward signals to child process
	go func() {
		for sig := range sigCh {
			if cmd.Process != nil {
				cmd.Process.Signal(sig)
			}
		}
	}()

	result.Status = "launched"
	result.PID = cmd.Process.Pid

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.Status = fmt.Sprintf("exited_%d", exitErr.ExitCode())
			result.ExitCode = exitErr.ExitCode()
			// Run after-launch hooks even on error
			if opts.Hooks != nil {
				RunAfterHooks(opts.Hooks, prof, err)
			}
			return result, nil
		}
		// Run after-launch hooks on error
		if opts.Hooks != nil {
			RunAfterHooks(opts.Hooks, prof, err)
		}
		return nil, fmt.Errorf("erreur processus : %w", err)
	}

	result.Status = "completed"
	result.ExitCode = cmd.ProcessState.ExitCode()
	// Run after-launch hooks on success
	if opts.Hooks != nil {
		RunAfterHooks(opts.Hooks, prof, nil)
	}
	return result, nil
}

// ShowEffectiveEnv displays the environment that would be set.
func ShowEffectiveEnv(prof *profile.Profile, asJSON bool) {
	if asJSON {
		fmt.Printf("{\n  \"profile\": \"%s\",\n  \"shortcut\": \"%s\",\n  \"tool\": \"%s\",\n  \"command\": \"%s %s\",\n  \"env\": {\n",
			prof.DisplayName, prof.Shortcut, prof.ToolLabel, prof.Command, strings.Join(prof.Args, " "))
		keys := make([]string, 0, len(prof.Env))
		for k := range prof.Env {
			keys = append(keys, k)
		}
		for i, k := range keys {
			v := prof.Env[k]
			if env.IsSecretKey(k) {
				v = env.MaskSecret(v)
			}
			comma := ","
			if i == len(keys)-1 {
				comma = ""
			}
			fmt.Printf("    \"%s\": \"%s\"%s\n", k, v, comma)
		}
		fmt.Printf("  }\n}\n")
		return
	}

	fmt.Println()
	fmt.Printf("Profil : %s [%s]\n", prof.DisplayName, prof.Shortcut)
	fmt.Printf("Outil  : %s\n", prof.ToolLabel)
	fmt.Printf("Commande : %s %s\n", prof.Command, strings.Join(prof.Args, " "))
	fmt.Println()
	for k, v := range prof.Env {
		display := v
		if env.IsSecretKey(k) {
			display = env.MaskSecret(v)
		}
		fmt.Printf("%s=%s\n", k, display)
	}
	fmt.Println()
}

// jsonError returns a JSON-formatted error string.
func jsonError(msg string) string {
	return `{"status":"error","error":"` + strings.ReplaceAll(msg, `"`, `\"`) + `"}`
}

func validateSecrets(prof *profile.Profile) error {
	if len(prof.RequiredSecrets) == 0 {
		return nil
	}
	for _, secret := range prof.RequiredSecrets {
		value := prof.Env[secret]
		if dotenv.IsPlaceholder(value) {
			return fmt.Errorf("secret obligatoire non configure pour le profil '%s' : %s\n  Edite : %s\n  Ou lance : multiai config",
				prof.DisplayName, secret, prof.Path)
		}
	}
	return nil
}
