package logging

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureStderr runs fn while capturing anything written to os.Stderr,
// then returns the captured output as a string.
func captureStderr(fn func()) string {
	r, w, _ := os.Pipe()
	old := os.Stderr
	os.Stderr = w

	fn()

	w.Close()
	os.Stderr = old

	data, _ := io.ReadAll(r)
	r.Close()
	return string(data)
}

func TestLogger_Info(t *testing.T) {
	dir := t.TempDir()
	l := &Logger{logDir: dir, minLevel: DEBUG}

	l.Log(INFO, "test info %s", "message")

	data, err := os.ReadFile(filepath.Join(dir, "multiai.log"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "[INFO] test info message") {
		t.Errorf("log file missing INFO entry:\n%s", string(data))
	}
}

func TestLogger_Warn(t *testing.T) {
	dir := t.TempDir()
	l := &Logger{logDir: dir, minLevel: DEBUG}

	stderr := captureStderr(func() {
		l.Log(WARN, "test warning %d", 42)
	})

	// Check log file
	data, err := os.ReadFile(filepath.Join(dir, "multiai.log"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "[WARN] test warning 42") {
		t.Errorf("log file missing WARN entry:\n%s", string(data))
	}
	// WARN should also appear on stderr
	if !strings.Contains(stderr, "[WARN] test warning 42") {
		t.Errorf("stderr missing WARN entry:\n%s", stderr)
	}
}

func TestLogger_Error(t *testing.T) {
	dir := t.TempDir()
	l := &Logger{logDir: dir, minLevel: DEBUG}

	stderr := captureStderr(func() {
		l.Log(ERROR, "test error fatal")
	})

	data, err := os.ReadFile(filepath.Join(dir, "multiai.log"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "[ERROR] test error fatal") {
		t.Errorf("log file missing ERROR entry:\n%s", string(data))
	}
	if !strings.Contains(stderr, "[ERROR] test error fatal") {
		t.Errorf("stderr missing ERROR entry:\n%s", stderr)
	}
}

func TestLogger_MinLevelFilters(t *testing.T) {
	dir := t.TempDir()
	l := &Logger{logDir: dir, minLevel: ERROR} // only ERROR passes

	l.Log(INFO, "should be filtered")
	l.Log(WARN, "should be filtered too")
	l.Log(ERROR, "should appear")

	data, err := os.ReadFile(filepath.Join(dir, "multiai.log"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "[INFO]") {
		t.Error("INFO entry was not filtered by minLevel=ERROR")
	}
	if strings.Contains(string(data), "[WARN]") {
		t.Error("WARN entry was not filtered by minLevel=ERROR")
	}
	if !strings.Contains(string(data), "[ERROR] should appear") {
		t.Error("ERROR entry was filtered out despite minLevel=ERROR")
	}
}

func TestLogSession_ValidJSON(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_LOGS_DIR", dir)

	LogSession(SessionEvent{
		Timestamp:       "2026-07-10T14:30:00Z",
		Shortcut:        "val",
		Profile:         "Validation Test",
		ProfilePath:     "/profiles/test.env",
		Command:         "cli-tool",
		ExitCode:        1,
		DurationSeconds: 2.5,
		Fallback:        true,
		Interrupted:     false,
	})

	data, err := os.ReadFile(filepath.Join(dir, "sessions.jsonl"))
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 JSONL line, got %d", len(lines))
	}

	var ev SessionEvent
	if err := json.Unmarshal([]byte(lines[0]), &ev); err != nil {
		t.Fatalf("invalid JSONL line: %v\nraw: %s", err, lines[0])
	}

	// Verify every field round-trips correctly
	if ev.Timestamp != "2026-07-10T14:30:00Z" {
		t.Errorf("timestamp = %q, want 2026-07-10T14:30:00Z", ev.Timestamp)
	}
	if ev.Shortcut != "val" {
		t.Errorf("shortcut = %q, want val", ev.Shortcut)
	}
	if ev.Profile != "Validation Test" {
		t.Errorf("profile = %q, want Validation Test", ev.Profile)
	}
	if ev.ProfilePath != "/profiles/test.env" {
		t.Errorf("profile_path = %q, want /profiles/test.env", ev.ProfilePath)
	}
	if ev.Command != "cli-tool" {
		t.Errorf("command = %q, want cli-tool", ev.Command)
	}
	if ev.ExitCode != 1 {
		t.Errorf("exit_code = %d, want 1", ev.ExitCode)
	}
	if ev.DurationSeconds != 2.5 {
		t.Errorf("duration_seconds = %v, want 2.5", ev.DurationSeconds)
	}
	if !ev.Fallback {
		t.Error("fallback = false, want true")
	}
	if ev.Interrupted {
		t.Error("interrupted = true, want false")
	}
}
