package registry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lrochetta/multiai/internal/fsutil"
)

// InstallProfile downloads a profile .env file from the registry, verifies its
// SHA256 checksum (when available on the entry), and writes it atomically into
// the profiles directory. It returns the destination path on success.
//
// Errors are of three kinds, all distinguishable by the caller:
//   - network/download errors (wrapped as-is)
//   - checksum mismatch (detectable via IsChecksumError)
//   - filesystem errors (wrapped as-is)
func InstallProfile(ctx context.Context, entry *ProfileEntry, profilesDir string) (string, error) {
	// Download the profile .env content.
	data, err := DownloadProfileContent(ctx, entry)
	if err != nil {
		return "", fmt.Errorf("download %s: %w", entry.Name, err)
	}

	// Verify SHA256 checksum when the entry carries one.
	if entry.SHA256 != "" {
		if err := verifyChecksum(data, entry.SHA256); err != nil {
			return "", err
		}
	}

	// Ensure the profiles directory exists.
	if err := os.MkdirAll(profilesDir, 0755); err != nil {
		return "", fmt.Errorf("create profiles dir: %w", err)
	}

	// Write the .env file atomically.
	destPath := filepath.Join(profilesDir, entry.Name+".env")
	if err := fsutil.WriteFileAtomic(destPath, data, 0644); err != nil {
		return "", fmt.Errorf("write %s: %w", destPath, err)
	}

	return destPath, nil
}

// verifyChecksum checks that data matches the expected hex-encoded SHA-256
// checksum. It returns a ChecksumError on mismatch.
func verifyChecksum(data []byte, expectedHex string) error {
	expectedHex = stripWhitespace(expectedHex)
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if got != expectedHex {
		return &ChecksumError{Expected: expectedHex, Got: got}
	}
	return nil
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

func (e *ChecksumError) Error() string {
	return fmt.Sprintf("checksum mismatch: expected %s, got %s", e.Expected, e.Got)
}

// IsChecksumError reports whether err is a ChecksumError (possibly wrapped).
func IsChecksumError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*ChecksumError)
	if ok {
		return true
	}
	// Check wrapped errors.
	type cser interface{ Unwrap() error }
	for err != nil {
		if _, ok := err.(*ChecksumError); ok {
			return true
		}
		u, ok := err.(cser)
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}
