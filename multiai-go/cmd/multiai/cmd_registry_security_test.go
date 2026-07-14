package main

import (
	"strings"
	"testing"
)

func TestParseProfileInstallFlagsRejectsNoVerify(t *testing.T) {
	if _, err := parseProfileInstallFlags([]string{"safe", "--no-verify"}); err == nil {
		t.Fatal("parseProfileInstallFlags() accepted removed --no-verify bypass")
	}
}

func TestCmdProfileInstallRejectsUnsafeNameBeforeNetwork(t *testing.T) {
	for _, name := range []string{"../escape", `..\escape`, "/tmp/escape", `C:\escape`, `\\server\share`} {
		t.Run(name, func(t *testing.T) {
			var code int
			stderr := snapshotStderr(func() {
				code = cmdProfileInstall([]string{name})
			})
			if code != 1 {
				t.Fatalf("cmdProfileInstall(%q) exit code = %d, want 1", name, code)
			}
			if !strings.Contains(stderr, "invalid profile name") {
				t.Fatalf("cmdProfileInstall(%q) stderr = %q, want validation error", name, stderr)
			}
		})
	}
}
