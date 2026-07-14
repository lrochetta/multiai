package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunProjectTrustLifecycle(t *testing.T) {
	root := t.TempDir()
	configDir := t.TempDir()
	t.Setenv("APPDATA", configDir)
	t.Setenv("XDG_CONFIG_HOME", configDir)

	configPath := filepath.Join(root, ".multiai.yaml")
	if err := os.WriteFile(configPath, []byte("overrides:\n  MULTIAI_TEST: safe\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })

	var output, errorOutput bytes.Buffer
	if code := runProject([]string{"status"}, &output, &errorOutput); code != 0 {
		t.Fatalf("status code=%d stderr=%s", code, errorOutput.String())
	}
	if !strings.Contains(output.String(), "untrusted") {
		t.Fatalf("status output=%q, want untrusted", output.String())
	}

	output.Reset()
	errorOutput.Reset()
	if code := runProject([]string{"trust", "--json"}, &output, &errorOutput); code != 0 {
		t.Fatalf("trust code=%d stderr=%s", code, errorOutput.String())
	}
	if !strings.Contains(output.String(), `"state": "trusted"`) {
		t.Fatalf("trust output=%q, want trusted", output.String())
	}

	if err := os.WriteFile(configPath, []byte("overrides:\n  MULTIAI_TEST: changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	output.Reset()
	errorOutput.Reset()
	if code := runProject([]string{"status"}, &output, &errorOutput); code != 0 {
		t.Fatalf("changed status code=%d stderr=%s", code, errorOutput.String())
	}
	if !strings.Contains(output.String(), "changed") {
		t.Fatalf("status output=%q, want changed", output.String())
	}

	output.Reset()
	errorOutput.Reset()
	if code := runProject([]string{"untrust"}, &output, &errorOutput); code != 0 {
		t.Fatalf("untrust code=%d stderr=%s", code, errorOutput.String())
	}
	if !strings.Contains(output.String(), "untrusted") {
		t.Fatalf("untrust output=%q, want untrusted", output.String())
	}
}

func TestRunProjectRejectsUnknownArgument(t *testing.T) {
	var output, errorOutput bytes.Buffer
	if code := runProject([]string{"trust", "--force"}, &output, &errorOutput); code != 2 {
		t.Fatalf("code=%d, want 2", code)
	}
}
