package logging

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readSessionLines(t *testing.T, dir string) []SessionEvent {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "sessions.jsonl"))
	if err != nil {
		t.Fatalf("cannot read session log: %v", err)
	}
	var events []SessionEvent
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var ev SessionEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Fatalf("invalid JSONL line %q: %v", line, err)
		}
		events = append(events, ev)
	}
	return events
}

func TestSessionLogPath_Override(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_LOGS_DIR", dir)

	path, err := SessionLogPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "sessions.jsonl")
	if path != want {
		t.Errorf("got %q, want %q", path, want)
	}
}

func TestSessionLogPath_Default(t *testing.T) {
	t.Setenv("MULTIAI_LOGS_DIR", "")

	path, err := SessionLogPath()
	if err != nil {
		t.Skipf("no user config dir on this system: %v", err)
	}
	if filepath.Base(path) != "sessions.jsonl" {
		t.Errorf("unexpected file name: %q", path)
	}
	if !strings.Contains(path, "multiai") {
		t.Errorf("default path should live under a multiai dir: %q", path)
	}
}

func TestLogSession_AppendsJSONL(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_LOGS_DIR", dir)

	events := []SessionEvent{
		{
			Timestamp:       "2026-07-05T10:00:00Z",
			Shortcut:        "cp",
			Profile:         "Code Pro (premium)",
			ProfilePath:     "/profiles/01-code-pro.env",
			Command:         "claude",
			ExitCode:        3,
			DurationSeconds: 1.5,
		},
		{
			Timestamp:       "2026-07-05T10:00:05Z",
			Shortcut:        "cf",
			Profile:         "Code Fast",
			Command:         "claude",
			ExitCode:        0,
			DurationSeconds: 0.2,
			Fallback:        true,
		},
	}
	for _, ev := range events {
		LogSession(ev)
	}

	got := readSessionLines(t, dir)
	if len(got) != 2 {
		t.Fatalf("expected 2 events, got %d", len(got))
	}
	if got[0].Shortcut != "cp" || got[0].ExitCode != 3 || got[0].Fallback {
		t.Errorf("event 0 mismatch: %+v", got[0])
	}
	if got[1].Shortcut != "cf" || got[1].ExitCode != 0 || !got[1].Fallback {
		t.Errorf("event 1 mismatch: %+v", got[1])
	}
	if got[0].DurationSeconds != 1.5 {
		t.Errorf("duration: got %v, want 1.5", got[0].DurationSeconds)
	}
}

// TestLogSession_SilentFailure: logging must never break a launch, even
// when the log directory cannot be created (parity with the PS reference).
func TestLogSession_SilentFailure(t *testing.T) {
	dir := t.TempDir()
	// Point MULTIAI_LOGS_DIR at a path whose parent is a regular file, so
	// MkdirAll fails.
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MULTIAI_LOGS_DIR", filepath.Join(blocker, "sub"))

	// Must not panic and must not return an error (there is none to return).
	LogSession(SessionEvent{Shortcut: "cp", ExitCode: 1})
}

// TestSessionEvent_NoSecretFields is a design guard: the event schema must
// never grow fields that could carry env values or CLI arguments.
func TestSessionEvent_NoSecretFields(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_LOGS_DIR", dir)

	LogSession(SessionEvent{
		Timestamp: "2026-07-05T10:00:00Z",
		Shortcut:  "ds",
		Profile:   "DeepSeek",
		Command:   "claude",
	})

	data, err := os.ReadFile(filepath.Join(dir, "sessions.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(data))), &raw); err != nil {
		t.Fatal(err)
	}
	for key := range raw {
		switch key {
		case "timestamp", "shortcut", "profile", "profile_path", "command",
			"exit_code", "duration_seconds", "fallback", "interrupted":
			// allowed
		default:
			t.Errorf("unexpected field %q in session event: extend the allowlist only if it cannot carry secrets", key)
		}
	}
}
