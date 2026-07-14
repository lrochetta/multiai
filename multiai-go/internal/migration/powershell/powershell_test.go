package powershell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// fakePSInstallation creates a minimal PowerShell legacy directory structure
// in a temp directory and returns the root path.
func fakePSInstallation(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "code-router.ps1"), `# fake code-router.ps1`)
	mustWrite(t, filepath.Join(root, "multiai.cmd"), `@echo off`)
	mustWrite(t, filepath.Join(root, "package.json"), `{"name":"multiai","version":"0.3.0"}`)

	profilesDir := filepath.Join(root, "configs", "profiles")
	mustMkdir(t, profilesDir)

	mustWrite(t, filepath.Join(profilesDir, "10-claude-official.env"),
		`PROFILE_ID=claude-official
SHORTCUT=co
TOOL=claude
DISPLAY_NAME=Claude Official
ORDER=10
COMMAND=claude
ANTHROPIC_API_KEY=sk-ant-test
`)

	mustWrite(t, filepath.Join(profilesDir, "30-claude-deepseek-v4-pro.env"),
		`PROFILE_ID=claude-deepseek-v4-pro
SHORTCUT=ds
TOOL=claude
DISPLAY_NAME=DeepSeek V4 Pro 1M
ORDER=40
COMMAND=claude
CLAUDE_CONFIG_DIR=%USERPROFILE%\.claude-deepseek-v4pro
ANTHROPIC_BASE_URL=https://api.deepseek.com/anthropic
ANTHROPIC_AUTH_TOKEN=PASTE_DEEPSEEK_API_KEY_HERE
ANTHROPIC_MODEL=deepseek-v4-pro[1m]
`)

	mustWrite(t, filepath.Join(profilesDir, "40-codex-gpt55.env"),
		`PROFILE_ID=codex-gpt55
SHORTCUT=codex55
TOOL=codex
DISPLAY_NAME=Codex GPT-5.5
ORDER=50
COMMAND=codex
OPENAI_API_KEY=sk-test123
`)

	return root
}

// fakePSInstallationMinimal creates a PS installation with no profiles (edge
// case).
func fakePSInstallationMinimal(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "multiai.cmd"), `@echo off`)
	mustMkdir(t, filepath.Join(root, "configs", "profiles"))
	return root
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

// ---------------------------------------------------------------------------
// TestDetect
// ---------------------------------------------------------------------------

func TestDetect_Found(t *testing.T) {
	root := fakePSInstallation(t)

	result, err := Detect(root)
	if err != nil {
		t.Fatalf("Detect() unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Detect() returned nil, expected a result")
	}

	if result.RootDir != root {
		t.Errorf("RootDir = %q, want %q", result.RootDir, root)
	}
	wantProfilesDir := filepath.Join(root, "configs", "profiles")
	if result.ProfilesDir != wantProfilesDir {
		t.Errorf("ProfilesDir = %q, want %q", result.ProfilesDir, wantProfilesDir)
	}
	if result.Version != "0.3.0" {
		t.Errorf("Version = %q, want 0.3.0", result.Version)
	}
	if result.ProfileCount != 3 {
		t.Errorf("ProfileCount = %d, want 3", result.ProfileCount)
	}
	if len(result.ProfileNames) != 3 {
		t.Fatalf("ProfileNames has %d entries, want 3", len(result.ProfileNames))
	}
	// Verify the expected profiles are present.
	names := make(map[string]bool)
	for _, n := range result.ProfileNames {
		names[n] = true
	}
	for _, want := range []string{
		"10-claude-official.env",
		"30-claude-deepseek-v4-pro.env",
		"40-codex-gpt55.env",
	} {
		if !names[want] {
			t.Errorf("ProfileNames missing %q", want)
		}
	}
}

func TestDetect_NotFound(t *testing.T) {
	// Empty directory with no PS markers.
	empty := t.TempDir()
	result, err := Detect(empty)
	if err != nil {
		t.Fatalf("Detect() unexpected error: %v", err)
	}
	if result != nil {
		t.Fatalf("Detect() returned non-nil result for empty dir: %+v", result)
	}
}

func TestDetect_NotFoundNonExistent(t *testing.T) {
	result, err := Detect("/nonexistent/ps/install")
	if err != nil {
		t.Fatalf("Detect() unexpected error: %v", err)
	}
	if result != nil {
		t.Fatal("Detect() should return nil for nonexistent path")
	}
}

func TestDetect_Minimal(t *testing.T) {
	root := fakePSInstallationMinimal(t)
	result, err := Detect(root)
	if err != nil {
		t.Fatalf("Detect() unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Detect() returned nil for minimal install")
	}
	if result.ProfileCount != 0 {
		t.Errorf("ProfileCount = %d, want 0", result.ProfileCount)
	}
	if result.Version != "" {
		t.Errorf("Version = %q, want empty (no package.json)", result.Version)
	}
}

func TestDetect_MultipleExtraDirs(t *testing.T) {
	root := fakePSInstallation(t)
	// Passing multiple dirs, only one should match.
	result, err := Detect("/tmp", root, "/nonexistent")
	if err != nil {
		t.Fatalf("Detect() unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Detect() should find the installation")
	}
}

func TestDetect_ProbeDirWithOnlyCmdFile(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "multiai.cmd"), `@echo off`)
	mustMkdir(t, filepath.Join(root, "configs", "profiles"))
	mustWrite(t, filepath.Join(root, "configs", "profiles", "test.env"), "PROFILE_ID=test\n")

	result, err := Detect(root)
	if err != nil {
		t.Fatalf("Detect() unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("Detect() returned nil")
	}
	if result.ProfileCount != 1 {
		t.Errorf("ProfileCount = %d, want 1", result.ProfileCount)
	}
}

// ---------------------------------------------------------------------------
// TestParseLegacyProfiles
// ---------------------------------------------------------------------------

func TestParseLegacyProfiles(t *testing.T) {
	root := fakePSInstallation(t)
	profilesDir := filepath.Join(root, "configs", "profiles")

	profiles, err := ParseLegacyProfiles(profilesDir)
	if err != nil {
		t.Fatalf("ParseLegacyProfiles() unexpected error: %v", err)
	}
	if len(profiles) != 3 {
		t.Fatalf("got %d profiles, want 3", len(profiles))
	}

	// Check a specific profile.
	found := false
	for _, p := range profiles {
		if p.ProfileID == "claude-deepseek-v4-pro" {
			found = true
			if p.Shortcut != "ds" {
				t.Errorf("Shortcut = %q, want %q", p.Shortcut, "ds")
			}
			if p.Tool != "claude" {
				t.Errorf("Tool = %q, want %q", p.Tool, "claude")
			}
			if p.EnvVars["ANTHROPIC_MODEL"] != "deepseek-v4-pro[1m]" {
				t.Errorf("ANTHROPIC_MODEL = %q, want %q", p.EnvVars["ANTHROPIC_MODEL"], "deepseek-v4-pro[1m]")
			}
			break
		}
	}
	if !found {
		t.Error("profile claude-deepseek-v4-pro not found")
	}
}

func TestParseLegacyProfiles_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	profiles, err := ParseLegacyProfiles(dir)
	if err != nil {
		t.Fatalf("ParseLegacyProfiles() unexpected error: %v", err)
	}
	if len(profiles) != 0 {
		t.Errorf("got %d profiles, want 0", len(profiles))
	}
}

func TestParseLegacyProfiles_NonExistentDir(t *testing.T) {
	_, err := ParseLegacyProfiles("/nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}

// ---------------------------------------------------------------------------
// TestMigrateRun
// ---------------------------------------------------------------------------

func TestRunMigration_Full(t *testing.T) {
	srcRoot := fakePSInstallation(t)
	detected, err := Detect(srcRoot)
	if err != nil {
		t.Fatalf("Detect() error: %v", err)
	}

	dstDir := t.TempDir()

	report, err := RunMigration(detected, dstDir, MigrateOptions{})
	if err != nil {
		t.Fatalf("RunMigration() error: %v", err)
	}

	if report.Detected == nil {
		t.Fatal("report.Detected is nil")
	}
	if len(report.Migrated) != 3 {
		t.Errorf("Migrated count = %d, want 3", len(report.Migrated))
	}
	if len(report.Skipped) != 0 {
		t.Errorf("Skipped count = %d, want 0", len(report.Skipped))
	}
	if report.BackupPath == "" {
		t.Error("BackupPath should not be empty")
	}
	if report.DryRun {
		t.Error("DryRun should be false")
	}
	if report.TargetDir != dstDir {
		t.Errorf("TargetDir = %q, want %q", report.TargetDir, dstDir)
	}

	// Verify the profiles were actually copied.
	for _, name := range []string{"10-claude-official.env", "30-claude-deepseek-v4-pro.env", "40-codex-gpt55.env"} {
		path := filepath.Join(dstDir, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("profile %s not copied: %v", name, err)
		}
	}

	// Verify backup was created.
	backupEntries, err := os.ReadDir(report.BackupPath)
	if err != nil {
		t.Fatalf("cannot read backup dir: %v", err)
	}
	if len(backupEntries) != 3 {
		t.Errorf("backup has %d files, want 3", len(backupEntries))
	}
}

func TestRunMigration_DryRun(t *testing.T) {
	srcRoot := fakePSInstallation(t)
	detected, err := Detect(srcRoot)
	if err != nil {
		t.Fatalf("Detect() error: %v", err)
	}

	dstDir := t.TempDir()

	report, err := RunMigration(detected, dstDir, MigrateOptions{DryRun: true})
	if err != nil {
		t.Fatalf("RunMigration() error: %v", err)
	}

	if !report.DryRun {
		t.Error("DryRun should be true")
	}
	if len(report.Migrated) != 3 {
		t.Errorf("Migrated count = %d, want 3", len(report.Migrated))
	}

	// No files should have been copied.
	for _, name := range report.Migrated {
		path := filepath.Join(dstDir, name)
		if _, err := os.Stat(path); err == nil {
			t.Errorf("dry-run should not copy files, but %s exists", name)
		}
	}
	if report.BackupPath != "" {
		t.Errorf("dry-run should not create backup, got %s", report.BackupPath)
	}
}

func TestRunMigrationRejectsTraversalProfile(t *testing.T) {
	srcDir := t.TempDir()
	detected := &DetectResult{
		ProfilesDir:  srcDir,
		ProfileNames: []string{".." + string(filepath.Separator) + "outside.env"},
	}
	if _, err := RunMigration(detected, t.TempDir(), MigrateOptions{}); err == nil {
		t.Fatal("RunMigration() accepted a traversal profile name")
	}
}

func TestRunMigration_WithExistingTarget(t *testing.T) {
	srcRoot := fakePSInstallation(t)
	detected, err := Detect(srcRoot)
	if err != nil {
		t.Fatalf("Detect() error: %v", err)
	}

	dstDir := t.TempDir()

	// Pre-create one profile in the target.
	mustWrite(t, filepath.Join(dstDir, "10-claude-official.env"), "EXISTING=1")

	report, err := RunMigration(detected, dstDir, MigrateOptions{})
	if err != nil {
		t.Fatalf("RunMigration() error: %v", err)
	}

	// The existing file should be skipped.
	if len(report.Skipped) != 1 || report.Skipped[0] != "10-claude-official.env" {
		t.Errorf("Skipped = %v, want [10-claude-official.env]", report.Skipped)
	}
	if len(report.Migrated) != 2 {
		t.Errorf("Migrated count = %d, want 2", len(report.Migrated))
	}

	// Verify the existing file was NOT overwritten.
	data, err := os.ReadFile(filepath.Join(dstDir, "10-claude-official.env"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != "EXISTING=1" {
		t.Errorf("existing file was overwritten, content: %s", string(data))
	}
}

func TestRunMigration_NilSource(t *testing.T) {
	dstDir := t.TempDir()
	report, err := RunMigration(nil, dstDir, MigrateOptions{})
	if err != nil {
		t.Fatalf("RunMigration() error: %v", err)
	}
	if report.Detected != nil {
		t.Error("report.Detected should be nil")
	}
	if len(report.Migrated) != 0 {
		t.Errorf("Migrated count = %d, want 0", len(report.Migrated))
	}
}

func TestRunMigration_CustomBackupDir(t *testing.T) {
	srcRoot := fakePSInstallation(t)
	detected, err := Detect(srcRoot)
	if err != nil {
		t.Fatalf("Detect() error: %v", err)
	}

	dstDir := t.TempDir()
	customBackup := filepath.Join(t.TempDir(), "my-backup")

	report, err := RunMigration(detected, dstDir, MigrateOptions{BackupDir: customBackup})
	if err != nil {
		t.Fatalf("RunMigration() error: %v", err)
	}

	if report.BackupPath != customBackup {
		t.Errorf("BackupPath = %q, want %q", report.BackupPath, customBackup)
	}

	// Verify backup exists.
	if _, err := os.Stat(customBackup); err != nil {
		t.Errorf("custom backup directory not found: %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestReportString and JSON
// ---------------------------------------------------------------------------

func TestReportString_NoDetection(t *testing.T) {
	r := &MigrationReport{}
	s := r.String()
	if !strings.Contains(s, "Aucune") {
		t.Errorf("expected French 'Aucune', got: %s", s)
	}
}

func TestReportString_FullReport(t *testing.T) {
	r := &MigrationReport{
		Detected: &DetectResult{
			RootDir:      "C:\\npm\\multiai",
			Version:      "0.3.0",
			ProfileCount: 3,
			ProfileNames: []string{"a.env", "b.env", "c.env"},
		},
		Migrated:   []string{"a.env", "b.env"},
		Skipped:    []string{"c.env"},
		BackupPath: "C:\\backup\\profiles-backup-20260710-120000",
		TargetDir:  "C:\\Users\\laurent\\AppData\\Roaming\\multiai\\profiles",
	}
	s := r.String()
	if !strings.Contains(s, "Migration PowerShell") {
		t.Errorf("expected section header, got: %s", s)
	}
	if !strings.Contains(s, "2 copie") {
		t.Errorf("expected migration count, got: %s", s)
	}
	if !strings.Contains(s, "1 ignore") {
		t.Errorf("expected skip count, got: %s", s)
	}
}

func TestReportJSON(t *testing.T) {
	r := &MigrationReport{
		Detected: &DetectResult{
			RootDir:      "/usr/lib/node_modules/multiai",
			Version:      "0.3.0",
			ProfileCount: 1,
			ProfileNames: []string{"test.env"},
		},
		Migrated: []string{"test.env"},
		DryRun:   false,
	}
	j := r.ToJSON()
	if !strings.Contains(j, `"root_dir"`) {
		t.Errorf("JSON missing root_dir field: %s", j)
	}
	if !strings.Contains(j, `"migrated"`) {
		t.Errorf("JSON missing migrated field: %s", j)
	}
	if strings.Contains(j, `"error"`) {
		t.Errorf("JSON contains error: %s", j)
	}
}

// ---------------------------------------------------------------------------
// TestDetectWithVersionParsing
// ---------------------------------------------------------------------------

func TestReadPSVersion(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"valid", `{"name":"multiai","version":"0.3.0"}`, "0.3.0"},
		{"no version", `{"name":"multiai"}`, ""},
		{"invalid JSON", `not json`, ""},
		{"empty", ``, ""},
		{"v prefix preserved", `{"version":"v0.3.0"}`, "v0.3.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			pkgPath := filepath.Join(dir, "package.json")
			if tt.content != "" {
				mustWrite(t, pkgPath, tt.content)
			}
			got := readPSVersion(pkgPath)
			if got != tt.want {
				t.Errorf("readPSVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestExistingProfiles
// ---------------------------------------------------------------------------

func TestExistingProfiles(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "a.env"), "A=1")
	mustWrite(t, filepath.Join(dir, "b.env"), "B=2")
	mustWrite(t, filepath.Join(dir, "c.txt"), "not a profile")

	existing := existingProfiles(dir)
	if len(existing) != 2 {
		t.Errorf("existingProfiles() = %v, want 2 entries", existing)
	}
	if !existing["a.env"] {
		t.Error("a.env should exist")
	}
	if !existing["b.env"] {
		t.Error("b.env should exist")
	}
	if existing["c.txt"] {
		t.Error("c.txt should not be counted")
	}
}

func TestExistingProfiles_EmptyDir(t *testing.T) {
	existing := existingProfiles(t.TempDir())
	if len(existing) != 0 {
		t.Errorf("expected empty, got %v", existing)
	}
}

func TestExistingProfiles_NonExistentDir(t *testing.T) {
	existing := existingProfiles("/nonexistent")
	if len(existing) != 0 {
		t.Errorf("expected empty for nonexistent dir, got %v", existing)
	}
}

// ---------------------------------------------------------------------------
// TestEdgeCases
// ---------------------------------------------------------------------------

func TestDetect_ProfilesDirAtRoot(t *testing.T) {
	// Some legacy installations might have profiles at the root instead of
	// in configs/profiles/.
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "code-router.ps1"), "# fake")
	mustWrite(t, filepath.Join(root, "profile.env"), "PROFILE_ID=root-profile\n")

	result, err := Detect(root)
	if err != nil {
		t.Fatalf("Detect() error: %v", err)
	}
	if result == nil {
		t.Fatal("Detect() returned nil")
	}
	if result.ProfileCount != 1 {
		t.Errorf("ProfileCount = %d, want 1", result.ProfileCount)
	}
	if result.ProfilesDir != root {
		t.Errorf("ProfilesDir = %q, want %q", result.ProfilesDir, root)
	}
}

// ---------------------------------------------------------------------------
// Benchmark
// ---------------------------------------------------------------------------

func BenchmarkDetect(b *testing.B) {
	root := b.TempDir()
	mustWriteT(b, filepath.Join(root, "code-router.ps1"), `# fake`)
	mustWriteT(b, filepath.Join(root, "configs", "profiles", "p.env"), "PROFILE_ID=test\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Detect(root)
	}
}

// mustWriteT is a helper for writing files in benchmarks.
func mustWriteT(b interface{ TempDir() string }, path, content string) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		panic(err)
	}
}
