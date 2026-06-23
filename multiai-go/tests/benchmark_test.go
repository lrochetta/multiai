package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lrochetta/multiai/internal/profile"
)

func BenchmarkLoadDir(b *testing.B) {
	dir := b.TempDir()
	for i := 0; i < 17; i++ {
		content := "PROFILE_ID=bench-" + string(rune('a'+i)) + "\nTOOL=claude\nCOMMAND=claude\nCLAUDE_CONFIG_DIR=/tmp/test\n"
		os.WriteFile(filepath.Join(dir, string(rune('0'+i/10))+string(rune('0'+i%10))+"-bench.env"), []byte(content), 0644)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		profile.LoadDir(dir)
	}
}

func BenchmarkFindByShortcut(b *testing.B) {
	profiles := make([]profile.Profile, 17)
	shortcuts := []string{"ds", "dsf", "ca", "cg", "cgalt", "co", "codex55", "codex54", "codexmini", "oc", "ocanthropic", "ocdeepseek", "ocopenai", "ocqwen", "ockimi", "ocminimax", "oczai"}
	for i, sc := range shortcuts {
		profiles[i] = profile.Profile{ID: "profile-" + sc, Shortcut: sc, Tool: "claude"}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		profile.FindByShortcut(profiles, "ocdefault")
	}
}
