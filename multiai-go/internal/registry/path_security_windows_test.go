//go:build windows

package registry

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestValidateInstallPathRejectsWindowsJunction(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "target")
	junction := filepath.Join(base, "junction")
	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("cmd.exe", "/d", "/c", "mklink", "/J", junction, target).CombinedOutput(); err != nil {
		t.Skipf("cannot create test junction: %v (%s)", err, output)
	}
	if err := validateInstallPath(filepath.Join(junction, "profile.env")); err == nil {
		t.Fatal("validateInstallPath() accepted a Windows junction")
	}
}
