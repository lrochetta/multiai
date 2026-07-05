package logging

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Session journal — honest port of Write-CostLog (code-router.ps1 L1010-1021).
//
// The PowerShell reference appends launch records to a file named costs.log,
// but it records no cost: no tokens, no prices, no estimation of any kind.
// The router never sees token usage from the child CLIs, so any cost figure
// it produced would be fabricated. The Go port therefore keeps the same
// facts (timestamp, profile, exit code, duration) under an honest name
// (sessions.jsonl) and deliberately adds no cost estimation.

// SessionEvent is one launch record in the usage journal.
//
// Security invariant: events must never carry environment values or
// command-line arguments — both can contain secrets. Only identifiers,
// file paths, exit codes and durations are recorded, so nothing here ever
// needs masking.
type SessionEvent struct {
	Timestamp       string  `json:"timestamp"` // RFC3339
	Shortcut        string  `json:"shortcut"`
	Profile         string  `json:"profile"` // display name
	ProfilePath     string  `json:"profile_path,omitempty"`
	Command         string  `json:"command,omitempty"` // binary name only, never args
	ExitCode        int     `json:"exit_code"`
	DurationSeconds float64 `json:"duration_seconds"`      // wall clock, rounded to 0.1s
	Fallback        bool    `json:"fallback,omitempty"`    // launched as a fallback attempt
	Interrupted     bool    `json:"interrupted,omitempty"` // user Ctrl+C / SIGINT
}

// sessionMu serializes appends from concurrent goroutines in one process.
var sessionMu sync.Mutex

// SessionLogPath returns the path of the JSONL usage journal:
// $MULTIAI_LOGS_DIR/sessions.jsonl when the override is set (tests,
// portable setups), otherwise <UserConfigDir>/multiai/logs/sessions.jsonl.
func SessionLogPath() (string, error) {
	if dir := os.Getenv("MULTIAI_LOGS_DIR"); dir != "" {
		return filepath.Join(dir, "sessions.jsonl"), nil
	}
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg, "multiai", "logs", "sessions.jsonl"), nil
}

// LogSession appends one event to the usage journal. Failures are silent:
// logging must never break or delay a launch (parity with the PS reference,
// whose Write-CostLog swallows every error).
func LogSession(ev SessionEvent) {
	path, err := SessionLogPath()
	if err != nil {
		return
	}
	line, err := json.Marshal(ev)
	if err != nil {
		return
	}

	sessionMu.Lock()
	defer sessionMu.Unlock()

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(line, '\n'))
}
