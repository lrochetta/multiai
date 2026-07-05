package cli

import (
	"fmt"
	"time"

	"github.com/lrochetta/multiai/internal/profile"
)

// LaunchWithFallback runs ValidateAndLaunch and, when the launched process
// exits non-zero and the profile declares a fallback chain
// (FALLBACK=<shortcut>[,<shortcut>...]), tries each fallback profile in the
// declared order. Semantics mirror code-router.ps1 L1135-1163:
//
//   - a pre-validation failure of the initial profile (command missing,
//     secret missing, profile invalid) returns the error and never triggers
//     the chain — parity with the PS trap that exits 1;
//   - dry-run and no-launch never trigger the chain;
//   - each fallback attempt is fully re-validated (whitelist, PATH lookup,
//     credential-store resolution, required secrets) and receives the same
//     ExtraArgs as the initial launch;
//   - an attempt that cannot start (profile introuvable, cle manquante, CLI
//     absent) prints a red message, records exit code 4 and moves on to the
//     next shortcut;
//   - the chain is non-recursive: a fallback profile's own FALLBACK is
//     ignored;
//   - the returned result carries the exit code of the last process
//     launched (or a synthetic exit 4 result when the last attempt failed
//     before launching), so callers can propagate it as the router's exit
//     code, like `exit $exitCode` in the PS reference.
//
// Documented divergence from the PS reference: a launch interrupted by the
// user (Ctrl+C / SIGINT — detected via the forwarded signal or exit codes
// 130 / STATUS_CONTROL_C_EXIT) does NOT trigger the chain; relaunching
// another CLI after a deliberate interrupt is hostile. An interrupt during
// a fallback attempt stops the chain for the same reason.
func LaunchWithFallback(profiles []profile.Profile, prof *profile.Profile, opts LaunchOptions) (*LaunchResult, error) {
	result, err := ValidateAndLaunch(prof, opts)
	if err != nil {
		return nil, err
	}
	if opts.DryRun || opts.NoLaunch {
		return result, nil
	}
	if result.ExitCode == 0 || result.Interrupted || len(prof.Fallback) == 0 {
		return result, nil
	}

	// Message parity with the PS reference: the failure line always names
	// the initial command, while the exit code tracks the latest attempt.
	command := prof.Command
	exitCode := result.ExitCode

	for _, shortcut := range prof.Fallback {
		fmt.Println()
		PrintWarning(fmt.Sprintf("Echec (%s, exit=%d) -> fallback vers '%s'...", command, exitCode, shortcut))

		fbProf, ferr := profile.FindByShortcut(profiles, shortcut)
		if ferr != nil {
			PrintError(fmt.Sprintf("Fallback '%s' a echoue : %v", shortcut, ferr))
			exitCode = 4
			result = fallbackFailureResult(shortcut)
			continue
		}

		fbOpts := opts
		fbOpts.fallbackAttempt = true
		fbResult, ferr := ValidateAndLaunch(fbProf, fbOpts)
		if ferr != nil {
			PrintError(fmt.Sprintf("Fallback '%s' a echoue : %v", shortcut, ferr))
			exitCode = 4
			result = fallbackFailureResult(shortcut)
			continue
		}

		result = fbResult
		exitCode = fbResult.ExitCode
		if exitCode == 0 || fbResult.Interrupted {
			break
		}
	}
	return result, nil
}

// isInterruptExit reports whether an exit code corresponds to a user
// interruption rather than a CLI failure:
//   - 130: POSIX convention 128+SIGINT
//   - 0xC000013A: Windows STATUS_CONTROL_C_EXIT
func isInterruptExit(code int) bool {
	return code == 130 || uint32(code) == 0xC000013A
}

// fallbackFailureResult builds the synthetic result reported when a
// fallback attempt fails before its process could be launched. Exit code 4
// matches the PS reference ("erreur processus enfant / fallback KO").
// Nothing is written to the session journal for these attempts (parity:
// Write-CostLog only runs after a real launch).
func fallbackFailureResult(shortcut string) *LaunchResult {
	return &LaunchResult{
		Shortcut:  shortcut,
		Status:    "fallback_error",
		ExitCode:  4,
		Timestamp: time.Now().Format(time.RFC3339),
	}
}
