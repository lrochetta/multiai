package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/lrochetta/multiai/internal/env"
	"github.com/lrochetta/multiai/internal/profile"
	"github.com/lrochetta/multiai/pkg/dotenv"
)

// AllowedCommands are the only binaries that can be launched by default.
var AllowedCommands = map[string]bool{
	"claude":   true,
	"codex":    true,
	"opencode": true,
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
	Profile  string `json:"profile"`
	Shortcut string `json:"shortcut"`
	Tool     string `json:"tool"`
	Command  string `json:"command"`
	Args     string `json:"args"`
	Status   string `json:"status"`
	PID      int    `json:"pid,omitempty"`
}

// ValidateAndLaunch checks the profile and launches the CLI.
func ValidateAndLaunch(prof *profile.Profile, opts LaunchOptions) (*LaunchResult, error) {
	// 1. Validate command whitelist
	if !AllowedCommands[prof.Command] {
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
		Profile:  prof.DisplayName,
		Shortcut: prof.Shortcut,
		Tool:     prof.ToolLabel,
		Command:  fmt.Sprintf("%s %s", prof.Command, strings.Join(allArgs, " ")),
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
	cmd := exec.Command(prof.Command, allArgs...)
	cmd.Env = cmdEnv
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("\nLancement : %s\n", result.Command)

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("erreur de lancement : %w", err)
	}

	result.Status = "launched"
	result.PID = cmd.Process.Pid

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.Status = fmt.Sprintf("exited_%d", exitErr.ExitCode())
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
