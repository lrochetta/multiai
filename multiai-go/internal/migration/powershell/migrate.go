package powershell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MigrateOptions controls migration behaviour.
type MigrateOptions struct {
	// DryRun performs all detection and reporting without copying files.
	DryRun bool
	// BackupDir is where the legacy installation is backed up before
	// migration. If empty, a timestamped directory inside the user's
	// config dir or the legacy root is used.
	BackupDir string
}

// MigrationReport describes what happened during a migration.
type MigrationReport struct {
	// Detected is the legacy installation that was found (nil if none).
	Detected *DetectResult `json:"detected,omitempty"`
	// Migrated lists profiles that were copied to the target.
	Migrated []string `json:"migrated"`
	// Skipped lists profiles that already existed in the target.
	Skipped []string `json:"skipped"`
	// BackupPath is where the legacy installation was backed up, or "".
	BackupPath string `json:"backup_path,omitempty"`
	// DryRun indicates whether this was a simulation.
	DryRun bool `json:"dry_run"`
	// TargetDir is the profiles directory where files were copied.
	TargetDir string `json:"target_dir"`
}

// RunMigration executes the migration from a detected legacy PS installation
// to the Go profiles directory (dstDir). It first creates a backup, then
// copies .env files that don't already exist in dstDir.
func RunMigration(src *DetectResult, dstDir string, opts MigrateOptions) (*MigrationReport, error) {
	report := &MigrationReport{
		Detected:  src,
		TargetDir: dstDir,
		DryRun:    opts.DryRun,
	}

	if src == nil {
		return report, nil
	}

	// Ensure the destination directory exists.
	if !opts.DryRun {
		if err := os.MkdirAll(dstDir, 0o755); err != nil {
			return nil, fmt.Errorf("cannot create target directory %s: %w", dstDir, err)
		}
	}

	// Determine what profiles exist in the target.
	existing := existingProfiles(dstDir)

	// Classify profiles as migrate or skip.
	for _, name := range src.ProfileNames {
		if existing[name] {
			report.Skipped = append(report.Skipped, name)
		} else {
			report.Migrated = append(report.Migrated, name)
		}
	}

	if opts.DryRun {
		return report, nil
	}

	// Backup the legacy profiles directory (timestamped).
	backupDir := opts.BackupDir
	if backupDir == "" {
		backupDir = filepath.Join(filepath.Dir(src.ProfilesDir),
			"profiles-backup-"+time.Now().Format("20060102-150405"))
	}
	if err := backupProfiles(src.ProfilesDir, backupDir); err != nil {
		return nil, fmt.Errorf("backup failed: %w", err)
	}
	report.BackupPath = backupDir

	// Copy profiles that don't already exist in the target.
	for _, name := range report.Migrated {
		srcPath := filepath.Join(src.ProfilesDir, name)
		dstPath := filepath.Join(dstDir, name)
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return nil, fmt.Errorf("cannot read %s: %w", srcPath, err)
		}
		if err := os.WriteFile(dstPath, data, 0o600); err != nil {
			return nil, fmt.Errorf("cannot write %s: %w", dstPath, err)
		}
	}

	return report, nil
}

// backupProfiles copies all files from srcDir to backupDir.
func backupProfiles(srcDir, backupDir string) error {
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return fmt.Errorf("cannot create backup directory %s: %w", backupDir, err)
	}

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("cannot read source directory for backup %s: %w", srcDir, err)
	}

	var lastErr error
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		srcPath := filepath.Join(srcDir, e.Name())
		dstPath := filepath.Join(backupDir, e.Name())
		data, err := os.ReadFile(srcPath)
		if err != nil {
			lastErr = fmt.Errorf("cannot read %s: %w", srcPath, err)
			continue
		}
		if err := os.WriteFile(dstPath, data, 0o600); err != nil {
			lastErr = fmt.Errorf("cannot write backup %s: %w", dstPath, err)
			continue
		}
	}
	return lastErr
}

// existingProfiles returns a set of .env filenames already present in dir.
func existingProfiles(dir string) map[string]bool {
	result := make(map[string]bool)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return result
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".env") {
			result[e.Name()] = true
		}
	}
	return result
}
