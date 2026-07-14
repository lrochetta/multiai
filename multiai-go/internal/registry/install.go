package registry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lrochetta/multiai/internal/fsutil"
)

const maxProfileBytes = 1 << 20 // 1 MiB is ample for a .env profile.

var profileNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

// ValidateProfileName accepts only portable, registry-safe profile IDs. The
// explicit path checks make the boundary fail closed on every host OS: a
// Windows path must still be rejected when multiai is tested on Linux, and
// vice versa.
func ValidateProfileName(name string) error {
	if name == "" {
		return fmt.Errorf("profile name is required")
	}
	if path.IsAbs(name) || filepath.IsAbs(name) || filepath.VolumeName(name) != "" || hasWindowsVolumePrefix(name) {
		return fmt.Errorf("invalid profile name %q: absolute and volume paths are forbidden", name)
	}
	if strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("invalid profile name %q: path separators are forbidden", name)
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("invalid profile name %q: parent traversal is forbidden", name)
	}
	if !profileNamePattern.MatchString(name) {
		return fmt.Errorf("invalid profile name %q: expected [a-z0-9][a-z0-9_-]{0,63}", name)
	}
	return nil
}

func hasWindowsVolumePrefix(name string) bool {
	if strings.HasPrefix(name, `\`) || strings.HasPrefix(name, "//") {
		return true
	}
	if len(name) < 2 || name[1] != ':' {
		return false
	}
	c := name[0]
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// ResolveInstallPath validates all remotely controlled install metadata and
// proves that the resulting destination is a direct child of profilesDir.
// Callers must use this before any filesystem test or write.
func ResolveInstallPath(entry *ProfileEntry, profilesDir string) (string, error) {
	if entry == nil {
		return "", fmt.Errorf("profile entry is required")
	}
	if err := ValidateProfileName(entry.Name); err != nil {
		return "", err
	}
	if _, err := normalizeChecksum(entry.SHA256); err != nil {
		return "", err
	}
	if strings.TrimSpace(profilesDir) == "" {
		return "", fmt.Errorf("profiles directory is required")
	}

	root, err := filepath.Abs(profilesDir)
	if err != nil {
		return "", fmt.Errorf("resolve profiles directory: %w", err)
	}
	dest, err := filepath.Abs(filepath.Join(root, entry.Name+".env"))
	if err != nil {
		return "", fmt.Errorf("resolve profile destination: %w", err)
	}
	rel, err := filepath.Rel(root, dest)
	if err != nil {
		return "", fmt.Errorf("confine profile destination: %w", err)
	}
	wantRel := entry.Name + ".env"
	if filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel != wantRel {
		return "", fmt.Errorf("profile destination escapes profiles directory")
	}
	return dest, nil
}

// InstallProfile downloads a profile .env file from the registry, verifies its
// mandatory SHA256 checksum, and writes it atomically into
// the profiles directory. It returns the destination path on success.
//
// Errors are of three kinds, all distinguishable by the caller:
//   - network/download errors (wrapped as-is)
//   - checksum mismatch (detectable via IsChecksumError)
//   - filesystem errors (wrapped as-is)
func InstallProfile(ctx context.Context, entry *ProfileEntry, profilesDir string) (string, error) {
	destPath, err := ResolveInstallPath(entry, profilesDir)
	if err != nil {
		return "", err
	}
	if err := validateInstallPath(destPath); err != nil {
		return "", err
	}

	// Download the profile .env content.
	data, err := DownloadProfileContent(ctx, entry)
	if err != nil {
		return "", fmt.Errorf("download %s: %w", entry.Name, err)
	}
	if len(data) > maxProfileBytes {
		return "", fmt.Errorf("profile %s exceeds maximum size of %d bytes", entry.Name, maxProfileBytes)
	}

	// Every registry installation is authenticated by its index checksum.
	if err := verifyChecksum(data, entry.SHA256); err != nil {
		return "", err
	}

	// Ensure the profiles directory exists.
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return "", fmt.Errorf("create profiles dir: %w", err)
	}
	if err := validateInstallPath(destPath); err != nil {
		return "", err
	}

	// Write the .env file atomically.
	if err := fsutil.WriteFileAtomic(destPath, data, 0644); err != nil {
		return "", fmt.Errorf("write %s: %w", destPath, err)
	}

	return destPath, nil
}

// validateInstallPath rejects filesystem indirection in the destination and
// every existing ancestor. Windows reparse points include junctions that are
// not necessarily exposed as ordinary symlinks by os.Lstat.
func validateInstallPath(destPath string) error {
	destPath = filepath.Clean(destPath)
	current := destPath
	for {
		info, err := os.Lstat(current)
		switch {
		case err == nil:
			if info.Mode()&os.ModeSymlink != 0 || isReparsePoint(info) {
				return fmt.Errorf("unsafe registry install path %q: symlink or reparse point is forbidden", current)
			}
			if current == destPath && !info.Mode().IsRegular() {
				return fmt.Errorf("unsafe registry destination %q: existing target is not a regular file", current)
			}
		case os.IsNotExist(err):
		default:
			return fmt.Errorf("inspect registry install path %q: %w", current, err)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
		current = parent
	}
}

// verifyChecksum checks that data matches the expected hex-encoded SHA-256
// checksum. It returns a ChecksumError on mismatch.
func verifyChecksum(data []byte, expectedHex string) error {
	expectedHex, err := normalizeChecksum(expectedHex)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if got != expectedHex {
		return &ChecksumError{Expected: expectedHex, Got: got}
	}
	return nil
}

func normalizeChecksum(expectedHex string) (string, error) {
	expectedHex = strings.ToLower(stripWhitespace(expectedHex))
	if expectedHex == "" {
		return "", &ChecksumMetadataError{Reason: "SHA-256 checksum is required"}
	}
	if len(expectedHex) != sha256.Size*2 {
		return "", &ChecksumMetadataError{Reason: "SHA-256 checksum must contain exactly 64 hexadecimal characters"}
	}
	if _, err := hex.DecodeString(expectedHex); err != nil {
		return "", &ChecksumMetadataError{Reason: "SHA-256 checksum is not hexadecimal"}
	}
	return expectedHex, nil
}

// stripWhitespace removes whitespace characters from s (handles leading,
// trailing, and inline spaces/tabs/newlines in checksum files).
func stripWhitespace(s string) string {
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			b = append(b, c)
		}
	}
	return string(b)
}

// ChecksumError is returned when a downloaded profile's SHA-256 checksum does
// not match the expected value. Callers can detect it with a type assertion or
// errors.As.
type ChecksumError struct {
	Expected string
	Got      string
}

// ChecksumMetadataError reports absent or malformed checksum metadata from the
// registry index. It is classified as a checksum error so the CLI can explain
// the trust failure without attempting a download or write.
type ChecksumMetadataError struct {
	Reason string
}

func (e *ChecksumMetadataError) Error() string { return e.Reason }

func (e *ChecksumError) Error() string {
	return fmt.Sprintf("checksum mismatch: expected %s, got %s", e.Expected, e.Got)
}

// IsChecksumError reports whether err is a ChecksumError (possibly wrapped).
func IsChecksumError(err error) bool {
	var mismatch *ChecksumError
	var metadata *ChecksumMetadataError
	return errors.As(err, &mismatch) || errors.As(err, &metadata)
}
