package secret

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lrochetta/multiai/internal/fsutil"
)

// MigratedMarkerName is the file name written to the secrets directory after
// a successful migration from the encrypted file store to a native credential
// store (wincred, keychain, secret-service). Its presence prevents re-migration
// unless --migrate-force is used.
const MigratedMarkerName = ".migrated"

// FileStoreMigrationReport describes the result of a migration from the
// encrypted file store to a native credential store.
type FileStoreMigrationReport struct {
	// ServicesFound lists the service names that had .enc files in the store.
	ServicesFound []string `json:"services_found,omitempty"`
	// Migrated lists service names that were successfully copied.
	Migrated []string `json:"migrated,omitempty"`
	// Skipped lists service names that were skipped (already present in the
	// native store, or empty).
	Skipped []string `json:"skipped,omitempty"`
	// Failed lists service names whose migration failed and were rolled back.
	Failed []string `json:"failed,omitempty"`
	// Force indicates whether --migrate-force was used.
	Force bool `json:"force"`
	// AlreadyMigrated is true when the .migrated marker already existed and
	// force was false.
	AlreadyMigrated bool `json:"already_migrated,omitempty"`
}

// SecretsDir returns the path to the encrypted file store directory.
// Exported so that main.go can display the path to the user during migration.
func SecretsDir() (string, error) {
	return secretsDir()
}

// MigratedMarkerPath returns the path to the .migrated marker file inside the
// secrets directory.
func MigratedMarkerPath() (string, error) {
	dir, err := secretsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, MigratedMarkerName), nil
}

// IsMigrated checks whether a previous file-to-native migration has completed
// successfully (the .migrated marker file exists).
func IsMigrated() bool {
	markerPath, err := MigratedMarkerPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(markerPath)
	return err == nil
}

// listFileStoreServices enumerates credential service names that have data in
// the encrypted file store by scanning for .enc files in the secrets directory.
// It skips dot-files (.masterkey, .migrated, etc.).
func listFileStoreServices(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("cannot read secrets directory: %w", err)
	}

	var services []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".enc") {
			continue
		}
		if strings.HasPrefix(name, ".") {
			continue
		}
		svc := serviceNameFromEncFile(name)
		services = append(services, svc)
	}
	return services, nil
}

// serviceNameFromEncFile recovers the original service name from a sanitized
// .enc file name. The file store's sanitizeFileName replaces all
// non-alphanumeric characters (except '.', '-', '_') with '_'. The only
// character affected in practice is ':' — "multiai:ca-a1b2c3d4" becomes
// "multiai_ca-a1b2c3d4.enc". We reverse by replacing "multiai_" with
// "multiai:".
//
// For service names that contain no ':' we return the name as-is.
func serviceNameFromEncFile(fileName string) string {
	name := strings.TrimSuffix(fileName, ".enc")
	if strings.HasPrefix(name, "multiai_") {
		return "multiai:" + name[8:]
	}
	return name
}

// MigrateFromFileStore migrates all credentials from the encrypted file store
// to the given native store (dstStore). It returns a report describing what
// was migrated, skipped, or failed.
//
// When force is false and the .migrated marker already exists, the function
// returns immediately with AlreadyMigrated=true in the report.
//
// The migration is transactional per service: if any credential write for a
// given service fails, all previously written credentials for that service are
// deleted from the native store (rollback). A service whose native store
// already has credentials is skipped unless force is true.
//
// On success (at least one service migrated and no failures), the .migrated
// marker is written atomically to the secrets directory.
func MigrateFromFileStore(dstStore Store, force bool) (*FileStoreMigrationReport, error) {
	report := &FileStoreMigrationReport{Force: force}

	// Check if the marker already exists.
	if !force && IsMigrated() {
		report.AlreadyMigrated = true
		return report, nil
	}

	// Open the file store to get its directory and master key.
	fileStore, err := newEncryptedFileStore()
	if err != nil {
		return nil, fmt.Errorf("cannot open file store: %w", err)
	}
	dir := fileStore.dir

	services, err := listFileStoreServices(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot list file store services: %w", err)
	}
	report.ServicesFound = services

	if len(services) == 0 {
		return report, nil
	}

	// Migrate each service.
	for _, svc := range services {
		// Read all credentials from the file store.
		creds, err := fileStore.List(svc)
		if err != nil {
			report.Failed = append(report.Failed, svc)
			continue
		}
		if len(creds) == 0 {
			report.Skipped = append(report.Skipped, svc)
			continue
		}

		// Check whether the native store already has data for this service.
		existing, err := dstStore.List(svc)
		if err != nil {
			report.Failed = append(report.Failed, svc)
			continue
		}
		if len(existing) > 0 && !force {
			report.Skipped = append(report.Skipped, svc)
			continue
		}

		// Transaction: write all credentials, then commit or rollback.
		written := make([]string, 0, len(creds))
		ok := true
		for k, v := range creds {
			if err := dstStore.Set(svc, k, v); err != nil {
				ok = false
				break
			}
			written = append(written, k)
		}

		if !ok {
			// Rollback: delete every key we just wrote.
			for _, k := range written {
				_ = dstStore.Delete(svc, k)
			}
			report.Failed = append(report.Failed, svc)
			continue
		}

		report.Migrated = append(report.Migrated, svc)
	}

	// Write the .migrated marker if at least one service was migrated and
	// none failed (partial migration is not marked complete).
	if len(report.Migrated) > 0 && len(report.Failed) == 0 {
		if err := writeMigratedMarker(dir); err != nil {
			return report, fmt.Errorf("migration succeeded but marker write failed: %w", err)
		}
	}

	return report, nil
}

// writeMigratedMarker atomically writes the .migrated marker into the secrets
// directory.
func writeMigratedMarker(dir string) error {
	markerPath := filepath.Join(dir, MigratedMarkerName)
	return fsutil.WriteFileAtomic(markerPath, []byte("migrated"), 0600)
}
