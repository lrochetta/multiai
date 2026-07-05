package cli

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/lrochetta/multiai/internal/env"
	"github.com/lrochetta/multiai/internal/logging"
	"github.com/lrochetta/multiai/internal/profile"
	"github.com/lrochetta/multiai/internal/secret"
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

	// fallbackAttempt marks a launch made by LaunchWithFallback while
	// walking a FALLBACK chain, so the session journal can flag it.
	// Internal: only this package sets it.
	fallbackAttempt bool
}

// LaunchResult contains information about the launched process.
type LaunchResult struct {
	Profile     string            `json:"profile"`
	Shortcut    string            `json:"shortcut"`
	Tool        string            `json:"tool"`
	Command     string            `json:"command"`
	Args        string            `json:"args"`
	Status      string            `json:"status"`
	PID         int               `json:"pid,omitempty"`
	ExitCode    int               `json:"exit_code,omitempty"`
	Interrupted bool              `json:"interrupted,omitempty"` // user Ctrl+C / SIGINT, not a CLI failure
	Timestamp   string            `json:"timestamp"`
	Env         map[string]string `json:"env,omitempty"` // populated only with --show-env --json (secrets masked)
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

	// 3. Resolve credential-store sentinels written by 'multiai config'
	if err := resolveStoredSecrets(prof); err != nil {
		return nil, err
	}

	// 3b. Validate required secrets
	if err := validateSecrets(prof); err != nil {
		return nil, err
	}

	// 4. Build environment
	cmdEnv := buildProcessEnv(prof)

	// 5. Prepare command
	allArgs := append(prof.Args, opts.ExtraArgs...)

	result := &LaunchResult{
		Profile:   prof.DisplayName,
		Shortcut:  prof.Shortcut,
		Tool:      prof.ToolLabel,
		Command:   fmt.Sprintf("%s %s", prof.Command, strings.Join(allArgs, " ")),
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// 6. Show env if requested. In JSON mode the env rides inside the single
	// LaunchResult document (result.Env) instead of being printed as a second,
	// hand-rolled JSON object — so stdout stays one valid, properly escaped doc.
	if opts.ShowEnv {
		if opts.JSON {
			result.Env = maskedEffectiveEnv(prof)
		} else {
			ShowEffectiveEnv(prof)
		}
	}

	// 7. Dry run
	if opts.DryRun {
		result.Status = "dry_run"
		// In JSON mode print nothing here: the result JSON is emitted by the
		// caller and must be the only thing on stdout.
		if !opts.JSON {
			fmt.Printf("\n[DRY RUN] Simulation sans lancement\n")
			// Parity with code-router.ps1 L1100: a dry run previews the
			// effective environment, unless --show-env already did (step 6).
			if !opts.ShowEnv {
				ShowEffectiveEnv(prof)
			}
			fmt.Printf("[DRY RUN] Commande : %s\n", result.Command)
		}
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
	// Order matters (defers are LIFO): signal.Stop must run BEFORE close, so
	// os/signal has stopped delivering before the channel is closed. Closing
	// while Notify is still armed can panic "send on closed channel".
	defer close(sigCh)
	defer signal.Stop(sigCh)

	cmd := exec.Command(prof.Command, allArgs...)
	cmd.Env = cmdEnv
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if !opts.JSON {
		fmt.Printf("\nLancement : %s\n", result.Command)
	}

	start := time.Now()
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("erreur de lancement : %w", err)
	}

	// Forward signals to child process, remembering that the user asked to
	// interrupt: a Ctrl+C must never be treated as a CLI failure (fallback).
	var interrupted atomic.Bool
	go func() {
		for sig := range sigCh {
			interrupted.Store(true)
			if cmd.Process != nil {
				cmd.Process.Signal(sig)
			}
		}
	}()

	result.Status = "launched"
	result.PID = cmd.Process.Pid

	waitErr := cmd.Wait()
	duration := time.Since(start)

	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			result.Status = fmt.Sprintf("exited_%d", exitErr.ExitCode())
			result.ExitCode = exitErr.ExitCode()
			result.Interrupted = interrupted.Load() || isInterruptExit(result.ExitCode)
			logLaunchSession(prof, result, duration, opts.fallbackAttempt)
			// Run after-launch hooks even on error
			if opts.Hooks != nil {
				RunAfterHooks(opts.Hooks, prof, waitErr)
			}
			return result, nil
		}
		// Run after-launch hooks on error
		if opts.Hooks != nil {
			RunAfterHooks(opts.Hooks, prof, waitErr)
		}
		return nil, fmt.Errorf("erreur processus : %w", waitErr)
	}

	result.Status = "completed"
	result.ExitCode = cmd.ProcessState.ExitCode()
	result.Interrupted = interrupted.Load()
	logLaunchSession(prof, result, duration, opts.fallbackAttempt)
	// Run after-launch hooks on success
	if opts.Hooks != nil {
		RunAfterHooks(opts.Hooks, prof, nil)
	}
	return result, nil
}

// buildProcessEnv honours CLEAR_ENV. The default (true) delegates to
// env.BuildCleanEnv (system allowlist + profile overlay). CLEAR_ENV=false
// keeps the full current environment and overlays the profile variables on
// top, mirroring Apply-ProfileEnv in code-router.ps1 (L429-436). In both
// modes profile values have their %NAME% references resolved (env.Expand
// ProfileEnv), so indirection profiles behave identically either way.
func buildProcessEnv(prof *profile.Profile) []string {
	if prof.ClearEnv {
		return env.BuildCleanEnv(prof.Env)
	}
	merged := make(map[string]string)
	for _, kv := range os.Environ() {
		if idx := strings.Index(kv, "="); idx > 0 {
			merged[kv[:idx]] = kv[idx+1:]
		}
	}
	for k, v := range env.ExpandProfileEnv(prof.Env) {
		merged[k] = v
	}
	out := make([]string, 0, len(merged))
	for k, v := range merged {
		out = append(out, k+"="+v)
	}
	return out
}

// logLaunchSession appends one record to the usage journal after a real
// launch (never for dry-run/no-launch, which return before this point).
// Only names, paths, codes and durations are recorded — never environment
// values or arguments, which may contain secrets.
func logLaunchSession(prof *profile.Profile, result *LaunchResult, dur time.Duration, fallback bool) {
	logging.LogSession(logging.SessionEvent{
		Timestamp:       time.Now().Format(time.RFC3339),
		Shortcut:        prof.Shortcut,
		Profile:         prof.DisplayName,
		ProfilePath:     prof.Path,
		Command:         prof.Command,
		ExitCode:        result.ExitCode,
		DurationSeconds: math.Round(dur.Seconds()*10) / 10,
		Fallback:        fallback,
		Interrupted:     result.Interrupted,
	})
}

// maskedEffectiveEnv returns the child environment as it would be set — %NAME%
// references resolved so it matches what the process actually receives — with
// secret-named keys masked. Used for the --show-env --json payload.
func maskedEffectiveEnv(prof *profile.Profile) map[string]string {
	effective := env.ExpandProfileEnv(prof.Env)
	out := make(map[string]string, len(effective))
	for k, v := range effective {
		if env.IsSecretKey(k) {
			v = env.MaskSecret(v)
		}
		out[k] = v
	}
	return out
}

// ShowEffectiveEnv prints the environment that would be set, in human form.
// Profile values have their %NAME% references resolved first, so what is shown
// matches what the child process actually receives; secret-named keys are
// masked. The JSON form goes through maskedEffectiveEnv into the LaunchResult.
func ShowEffectiveEnv(prof *profile.Profile) {
	fmt.Println()
	fmt.Printf("Profil : %s [%s]\n", prof.DisplayName, prof.Shortcut)
	fmt.Printf("Outil  : %s\n", prof.ToolLabel)
	fmt.Printf("Commande : %s %s\n", prof.Command, strings.Join(prof.Args, " "))
	fmt.Println()
	for k, v := range maskedEffectiveEnv(prof) {
		fmt.Printf("%s=%s\n", k, v)
	}
	fmt.Println()
}

// jsonError returns a JSON-formatted error string.
func jsonError(msg string) string {
	return `{"status":"error","error":"` + strings.ReplaceAll(msg, `"`, `\"`) + `"}`
}

// resolveStoredSecrets replaces credential-store sentinels in the profile
// env with the real values saved by 'multiai config'. Without this step the
// literal sentinel would be exported as the API key.
func resolveStoredSecrets(prof *profile.Profile) error {
	var pending []string
	for k, v := range prof.Env {
		if v == secret.Sentinel {
			pending = append(pending, k)
		}
	}
	if len(pending) == 0 {
		return nil
	}
	store, err := secret.NewStore()
	if err != nil {
		return fmt.Errorf("credential store indisponible pour le profil '%s' : %w", prof.Shortcut, err)
	}
	service := secret.ServiceForProfile(prof.Path)
	for _, k := range pending {
		val, err := store.Get(service, k)
		if err != nil {
			return fmt.Errorf("secret %s du profil '%s' introuvable dans le credential store.\n  Relance : multiai config (%v)", k, prof.Shortcut, err)
		}
		prof.Env[k] = val
	}
	return nil
}

func validateSecrets(prof *profile.Profile) error {
	// SKIP_SECRET_CHECK=true bypasses the placeholder validation (parity
	// with code-router.ps1 Test-RequiredSecrets L455-458). Credential-store
	// sentinels are still resolved beforehand by resolveStoredSecrets.
	if prof.SkipSecretCheck {
		return nil
	}
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
