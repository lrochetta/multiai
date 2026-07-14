//go:build windows

package tests

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"
)

// TestWindowsStartupIsBounded guards against security software holding a
// freshly built executable before the Go runtime reaches main. Each command
// has its own deadline and a controller goroutine, so even CreateProcess being
// held before Start returns cannot stall the whole CI job.
func TestWindowsStartupIsBounded(t *testing.T) {
	const attempts = 3
	const startupTimeout = 10 * time.Second
	versionPattern := regexp.MustCompile(`^multiai \d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?$`)

	for attempt := 1; attempt <= attempts; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), startupTimeout)
		var stdout, stderr bytes.Buffer
		cmd := exec.CommandContext(ctx, multiaiBin, "--version")
		cmd.Env = append(os.Environ(), "MULTIAI_SKIP_UPDATE=1")
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		cmd.WaitDelay = 2 * time.Second

		started := time.Now()
		result := make(chan error, 1)
		go func() { result <- cmd.Run() }()

		var err error
		select {
		case err = <-result:
			cancel()
		case <-ctx.Done():
			// CommandContext cannot interrupt Windows CreateProcess while Start
			// itself is blocked. Keeping Run in a controller goroutine ensures
			// the test still reports the freeze within its fixed budget.
			t.Fatalf("attempt %d: multiai --version remained blocked in process startup for %s", attempt, startupTimeout)
		}
		elapsed := time.Since(started)

		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			t.Fatalf("attempt %d: multiai --version exceeded %s (elapsed %s)", attempt, startupTimeout, elapsed)
		}
		if err != nil {
			t.Fatalf("attempt %d: multiai --version failed after %s: %v (stdout=%q, stderr=%q)",
				attempt, elapsed, err, stdout.String(), stderr.String())
		}
		if output := strings.TrimSpace(stdout.String()); !versionPattern.MatchString(output) {
			t.Fatalf("attempt %d: unexpected version output %q (stderr=%q)", attempt, output, stderr.String())
		}
	}
}
