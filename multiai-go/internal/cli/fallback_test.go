package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lrochetta/multiai/internal/logging"
	"github.com/lrochetta/multiai/internal/profile"
)

// TestHelperProcess is not a real test: it is the child process launched by
// the fallback tests (stdlib os/exec pattern). It exits with
// MULTIAI_TEST_EXIT_CODE and optionally records its argv for assertions.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("MULTIAI_TEST_HELPER") != "1" {
		return
	}
	if s := os.Getenv("MULTIAI_TEST_SLEEP_MS"); s != "" {
		d, _ := time.ParseDuration(s + "ms")
		if d > 0 {
			time.Sleep(d)
		}
	}
	if file := os.Getenv("MULTIAI_TEST_ARGS_FILE"); file != "" {
		_ = os.WriteFile(file, []byte(strings.Join(os.Args, "\n")), 0644)
	}
	code := 0
	if v := os.Getenv("MULTIAI_TEST_EXIT_CODE"); v != "" {
		code, _ = strconv.Atoi(v)
	}
	os.Exit(code)
}

// testProfile builds a profile whose command is this test binary, so a real
// process is launched and exits with the requested code.
func testProfile(t *testing.T, shortcut string, exitCode int, fallback []string) profile.Profile {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return profile.Profile{
		ID:          shortcut,
		Shortcut:    shortcut,
		DisplayName: "Test " + shortcut,
		Tool:        "claude",
		ToolLabel:   "claude",
		Command:     exe,
		Args:        []string{"-test.run=^TestHelperProcess$"},
		Env: map[string]string{
			"MULTIAI_TEST_HELPER":    "1",
			"MULTIAI_TEST_EXIT_CODE": strconv.Itoa(exitCode),
		},
		ClearEnv: true,
		Fallback: fallback,
		Path:     "/profiles/" + shortcut + ".env",
	}
}

// readSessions parses the JSONL usage journal written during a test.
func readSessions(t *testing.T, dir string) []logging.SessionEvent {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "sessions.jsonl"))
	if err != nil {
		t.Fatalf("cannot read session journal: %v", err)
	}
	var events []logging.SessionEvent
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var ev logging.SessionEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Fatalf("invalid JSONL line %q: %v", line, err)
		}
		events = append(events, ev)
	}
	return events
}

func TestLaunchWithFallback_ChainSuccess(t *testing.T) {
	logsDir := t.TempDir()
	t.Setenv("MULTIAI_LOGS_DIR", logsDir)

	primary := testProfile(t, "cp", 3, []string{"missing", "cf"})
	fb := testProfile(t, "cf", 0, nil)
	profiles := []profile.Profile{primary, fb}

	result, err := LaunchWithFallback(profiles, &primary, LaunchOptions{AllowCustomCommand: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 {
		t.Errorf("final exit code: got %d, want 0", result.ExitCode)
	}
	if result.Shortcut != "cf" {
		t.Errorf("final shortcut: got %q, want cf", result.Shortcut)
	}

	// Session journal: primary launch + successful fallback launch. The
	// 'missing' attempt fails before launching and is not journaled
	// (parity with Write-CostLog).
	events := readSessions(t, logsDir)
	if len(events) != 2 {
		t.Fatalf("expected 2 session events, got %d: %+v", len(events), events)
	}
	if events[0].Shortcut != "cp" || events[0].ExitCode != 3 || events[0].Fallback {
		t.Errorf("primary event mismatch: %+v", events[0])
	}
	if events[1].Shortcut != "cf" || events[1].ExitCode != 0 || !events[1].Fallback {
		t.Errorf("fallback event mismatch: %+v", events[1])
	}
}

func TestLaunchWithFallback_AllAttemptsFail(t *testing.T) {
	logsDir := t.TempDir()
	t.Setenv("MULTIAI_LOGS_DIR", logsDir)

	primary := testProfile(t, "cp", 3, []string{"cf"})
	fb := testProfile(t, "cf", 5, nil)
	profiles := []profile.Profile{primary, fb}

	result, err := LaunchWithFallback(profiles, &primary, LaunchOptions{AllowCustomCommand: true})
	if err != nil {
		t.Fatal(err)
	}
	// Router exit code = exit code of the last process launched (PS parity).
	if result.ExitCode != 5 {
		t.Errorf("final exit code: got %d, want 5", result.ExitCode)
	}
	if len(readSessions(t, logsDir)) != 2 {
		t.Error("expected 2 session events (primary + fallback)")
	}
}

func TestLaunchWithFallback_UnknownFallbackExitCode4(t *testing.T) {
	logsDir := t.TempDir()
	t.Setenv("MULTIAI_LOGS_DIR", logsDir)

	primary := testProfile(t, "cp", 3, []string{"does-not-exist"})
	profiles := []profile.Profile{primary}

	result, err := LaunchWithFallback(profiles, &primary, LaunchOptions{AllowCustomCommand: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 4 {
		t.Errorf("final exit code: got %d, want 4 (fallback KO)", result.ExitCode)
	}
	if result.Status != "fallback_error" {
		t.Errorf("status: got %q, want fallback_error", result.Status)
	}
	// Only the primary launch is journaled.
	if events := readSessions(t, logsDir); len(events) != 1 {
		t.Errorf("expected 1 session event, got %d", len(events))
	}
}

func TestLaunchWithFallback_NoFallbackOnSuccess(t *testing.T) {
	logsDir := t.TempDir()
	t.Setenv("MULTIAI_LOGS_DIR", logsDir)

	primary := testProfile(t, "cp", 0, []string{"cf"})
	fb := testProfile(t, "cf", 0, nil)
	profiles := []profile.Profile{primary, fb}

	result, err := LaunchWithFallback(profiles, &primary, LaunchOptions{AllowCustomCommand: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 || result.Shortcut != "cp" {
		t.Errorf("unexpected result: %+v", result)
	}
	if events := readSessions(t, logsDir); len(events) != 1 {
		t.Errorf("expected 1 session event, got %d", len(events))
	}
}

func TestLaunchWithFallback_NoFallbackOnDryRun(t *testing.T) {
	logsDir := t.TempDir()
	t.Setenv("MULTIAI_LOGS_DIR", logsDir)

	primary := testProfile(t, "cp", 3, []string{"cf"})
	profiles := []profile.Profile{primary}

	result, err := LaunchWithFallback(profiles, &primary, LaunchOptions{DryRun: true, AllowCustomCommand: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "dry_run" {
		t.Errorf("status: got %q, want dry_run", result.Status)
	}
	// Dry-run never launches, so nothing may be journaled.
	if _, err := os.Stat(filepath.Join(logsDir, "sessions.jsonl")); !os.IsNotExist(err) {
		t.Error("dry-run must not write the session journal")
	}
}

// A user interrupt (exit 130 = 128+SIGINT) must not trigger the chain —
// documented divergence from the PS reference.
func TestLaunchWithFallback_NoFallbackOnInterrupt(t *testing.T) {
	logsDir := t.TempDir()
	t.Setenv("MULTIAI_LOGS_DIR", logsDir)

	primary := testProfile(t, "cp", 130, []string{"cf"})
	fb := testProfile(t, "cf", 0, nil)
	profiles := []profile.Profile{primary, fb}

	result, err := LaunchWithFallback(profiles, &primary, LaunchOptions{AllowCustomCommand: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 130 {
		t.Errorf("exit code: got %d, want 130", result.ExitCode)
	}
	if !result.Interrupted {
		t.Error("exit 130 must be flagged as interrupted")
	}
	events := readSessions(t, logsDir)
	if len(events) != 1 {
		t.Fatalf("interrupt must not trigger fallback: got %d events", len(events))
	}
	if !events[0].Interrupted {
		t.Error("session event should carry the interrupted flag")
	}
}

// The chain is single-level: a fallback profile's own FALLBACK is ignored.
func TestLaunchWithFallback_NonRecursive(t *testing.T) {
	logsDir := t.TempDir()
	t.Setenv("MULTIAI_LOGS_DIR", logsDir)

	primary := testProfile(t, "aa", 3, []string{"bb"})
	second := testProfile(t, "bb", 7, []string{"cc"})
	third := testProfile(t, "cc", 0, nil)
	profiles := []profile.Profile{primary, second, third}

	result, err := LaunchWithFallback(profiles, &primary, LaunchOptions{AllowCustomCommand: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 7 {
		t.Errorf("final exit code: got %d, want 7 (bb's, cc never tried)", result.ExitCode)
	}
	events := readSessions(t, logsDir)
	if len(events) != 2 {
		t.Fatalf("expected 2 events (aa, bb), got %d", len(events))
	}
	for _, ev := range events {
		if ev.Shortcut == "cc" {
			t.Error("recursive fallback was followed: cc must never launch")
		}
	}
}

// A pre-validation failure of the initial profile returns the error and
// never triggers the chain (PS parity: throw -> trap -> exit 1).
func TestLaunchWithFallback_NoFallbackOnValidationError(t *testing.T) {
	logsDir := t.TempDir()
	t.Setenv("MULTIAI_LOGS_DIR", logsDir)

	primary := testProfile(t, "cp", 0, []string{"cf"})
	primary.Command = "multiai-test-missing-binary-xyz"
	fb := testProfile(t, "cf", 0, nil)
	profiles := []profile.Profile{primary, fb}

	_, err := LaunchWithFallback(profiles, &primary, LaunchOptions{AllowCustomCommand: true})
	if err == nil {
		t.Fatal("expected a validation error for a missing command")
	}
	if _, statErr := os.Stat(filepath.Join(logsDir, "sessions.jsonl")); !os.IsNotExist(statErr) {
		t.Error("no launch happened, the journal must stay empty")
	}
}

// ExtraArgs given on the initial launch propagate to fallback attempts
// (PS parity, L1146-1149).
func TestLaunchWithFallback_ExtraArgsPropagate(t *testing.T) {
	logsDir := t.TempDir()
	t.Setenv("MULTIAI_LOGS_DIR", logsDir)

	argsFile := filepath.Join(t.TempDir(), "fb-args.txt")
	primary := testProfile(t, "cp", 3, []string{"cf"})
	fb := testProfile(t, "cf", 0, nil)
	fb.Env["MULTIAI_TEST_ARGS_FILE"] = argsFile
	profiles := []profile.Profile{primary, fb}

	opts := LaunchOptions{AllowCustomCommand: true, ExtraArgs: []string{"extra-marker"}}
	result, err := LaunchWithFallback(profiles, &primary, opts)
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("fallback should succeed, got exit %d", result.ExitCode)
	}
	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("fallback child did not record its argv: %v", err)
	}
	if !strings.Contains(string(data), "extra-marker") {
		t.Errorf("ExtraArgs not propagated to fallback attempt; argv:\n%s", data)
	}
}

func TestIsInterruptExit(t *testing.T) {
	tests := []struct {
		code int
		want bool
	}{
		{0, false},
		{1, false},
		{3, false},
		{4, false},
		{130, true},         // POSIX 128+SIGINT
		{-1073741510, true}, // Windows STATUS_CONTROL_C_EXIT as int32
	}

	for _, tt := range tests {
		if got := isInterruptExit(tt.code); got != tt.want {
			t.Errorf("isInterruptExit(%d): got %v, want %v", tt.code, got, tt.want)
		}
	}

	// STATUS_CONTROL_C_EXIT as reported unsigned by the platform. Computed
	// at runtime so the test also compiles on 32-bit targets.
	var winCtrlC uint32 = 0xC000013A
	if !isInterruptExit(int(winCtrlC)) {
		t.Error("unsigned STATUS_CONTROL_C_EXIT must be treated as an interrupt")
	}
}

func TestValidateAndLaunch_SkipSecretCheck(t *testing.T) {
	prof := testProfile(t, "ssc", 0, nil)
	prof.RequiredSecrets = []string{"MY_API_KEY"}
	prof.Env["MY_API_KEY"] = "PASTE_MY_API_KEY_HERE"
	opts := LaunchOptions{NoLaunch: true, AllowCustomCommand: true}

	if _, err := ValidateAndLaunch(&prof, opts); err == nil {
		t.Fatal("placeholder secret must fail validation when the check is active")
	}

	prof.SkipSecretCheck = true
	result, err := ValidateAndLaunch(&prof, opts)
	if err != nil {
		t.Fatalf("SKIP_SECRET_CHECK=true must bypass validation: %v", err)
	}
	if result.Status != "no_launch" {
		t.Errorf("status: got %q, want no_launch", result.Status)
	}
}

func TestBuildProcessEnv_RespectsClearEnv(t *testing.T) {
	t.Setenv("MULTIAI_TEST_CANARY", "canary-value")

	prof := &profile.Profile{
		ClearEnv: false,
		Env:      map[string]string{"FOO_SETTING": "bar"},
	}
	got := buildProcessEnv(prof)
	if !containsEnv(got, "MULTIAI_TEST_CANARY=canary-value") {
		t.Error("CLEAR_ENV=false must keep the current environment")
	}
	if !containsEnv(got, "FOO_SETTING=bar") {
		t.Error("profile vars must overlay the kept environment")
	}

	prof.ClearEnv = true
	got = buildProcessEnv(prof)
	if containsEnv(got, "MULTIAI_TEST_CANARY=canary-value") {
		t.Error("CLEAR_ENV=true must drop non-allowlisted variables")
	}
	if !containsEnv(got, "FOO_SETTING=bar") {
		t.Error("profile vars must survive the clean env")
	}
}

func containsEnv(envList []string, kv string) bool {
	for _, e := range envList {
		if e == kv {
			return true
		}
	}
	return false
}
