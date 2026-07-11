package registry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// TestVerifyChecksum
// ---------------------------------------------------------------------------

func TestVerifyChecksumValid(t *testing.T) {
	data := []byte("PROFILE_ID=test\nTOOL=claude\n")
	sum := sha256.Sum256(data)
	hexSum := hex.EncodeToString(sum[:])

	err := verifyChecksum(data, hexSum)
	if err != nil {
		t.Errorf("verifyChecksum() unexpected error: %v", err)
	}
}

func TestVerifyChecksumInvalid(t *testing.T) {
	data := []byte("PROFILE_ID=test\nTOOL=claude\n")
	// Deliberately wrong checksum.
	wrongHex := "0000000000000000000000000000000000000000000000000000000000000000"

	err := verifyChecksum(data, wrongHex)
	if err == nil {
		t.Fatal("verifyChecksum() expected error, got nil")
	}
	var cerr *ChecksumError
	if !errors.As(err, &cerr) {
		t.Fatalf("verifyChecksum() error type = %T, want *ChecksumError", err)
	}
	if cerr.Expected != wrongHex {
		t.Errorf("Expected = %q, want %q", cerr.Expected, wrongHex)
	}
}

func TestVerifyChecksumWithWhitespace(t *testing.T) {
	data := []byte("PROFILE_ID=test\nTOOL=claude\n")
	sum := sha256.Sum256(data)
	hexSum := hex.EncodeToString(sum[:])

	// Checksum with whitespace like a sha256sum output.
	wsHex := "  " + hexSum + "\n"
	err := verifyChecksum(data, wsHex)
	if err != nil {
		t.Errorf("verifyChecksum() unexpected error with whitespace: %v", err)
	}
}

func TestVerifyChecksumEmptyData(t *testing.T) {
	data := []byte{}
	sum := sha256.Sum256(data)
	hexSum := hex.EncodeToString(sum[:])

	err := verifyChecksum(data, hexSum)
	if err != nil {
		t.Errorf("verifyChecksum(empty) unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestStripWhitespace
// ---------------------------------------------------------------------------

func TestStripWhitespace(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"abc123", "abc123"},
		{"  abc123\n", "abc123"},
		{"abc 123", "abc123"},
		{"\t\n\r", ""},
		{"a b\tc\nd\r", "abcd"},
	}
	for _, tt := range tests {
		got := stripWhitespace(tt.input)
		if got != tt.want {
			t.Errorf("stripWhitespace(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// TestIsChecksumError
// ---------------------------------------------------------------------------

func TestIsChecksumError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"bare ChecksumError", &ChecksumError{Expected: "a", Got: "b"}, true},
		{"wrapped ChecksumError", &installError{msg: "wrapped", inner: &ChecksumError{Expected: "a", Got: "b"}}, true},
		{"other error", errors.New("other"), false},
		{"double wrapped", fmt.Errorf("outer: %w", &installError{msg: "inner", inner: &ChecksumError{Expected: "a", Got: "b"}}), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsChecksumError(tt.err)
			if got != tt.want {
				t.Errorf("IsChecksumError() = %v, want %v", got, tt.want)
			}
		})
	}
}

// installError is a test helper implementing unwrappable error.
type installError struct {
	msg   string
	inner error
}

func (e *installError) Error() string { return e.msg }
func (e *installError) Unwrap() error { return e.inner }

// ---------------------------------------------------------------------------
// TestProfileDownloadURL
// ---------------------------------------------------------------------------

func TestProfileDownloadURL(t *testing.T) {
	t.Run("custom download URL", func(t *testing.T) {
		entry := &ProfileEntry{
			Name:        "test",
			DownloadURL: "https://example.com/test.env",
		}
		url := profileDownloadURL(entry)
		if url != "https://example.com/test.env" {
			t.Errorf("profileDownloadURL() = %q, want custom URL", url)
		}
	})

	t.Run("default URL from name", func(t *testing.T) {
		entry := &ProfileEntry{Name: "ds"}
		url := profileDownloadURL(entry)
		want := "https://raw.githubusercontent.com/lrochetta/profiles-multiai/main/profiles/ds.env"
		if url != want {
			t.Errorf("profileDownloadURL() = %q, want %q", url, want)
		}
	})

	t.Run("URL encoding safety", func(t *testing.T) {
		entry := &ProfileEntry{Name: "my-profile"}
		url := profileDownloadURL(entry)
		if !strings.Contains(url, "/my-profile.env") {
			t.Errorf("profileDownloadURL() = %q, should contain profile name", url)
		}
	})
}

// ---------------------------------------------------------------------------
// TestInstallProfile — mock HTTP round-trip
// ---------------------------------------------------------------------------

func TestInstallProfileSuccess(t *testing.T) {
	// Save and restore DownloadProfileContent.
	orig := DownloadProfileContent
	defer func() { DownloadProfileContent = orig }()

	// Stub the download function.
	profileContent := []byte("PROFILE_ID=test\nTOOL=claude\nDESCRIPTION=Test profile\n")
	DownloadProfileContent = func(_ context.Context, entry *ProfileEntry) ([]byte, error) {
		return profileContent, nil
	}

	// Create a temporary profiles directory.
	profilesDir := t.TempDir()

	entry := &ProfileEntry{
		Name:        "test-profile",
		Title:       "Test Profile",
		Description: "A test profile for unit testing",
		Author:      "tester",
	}

	destPath, err := InstallProfile(context.Background(), entry, profilesDir)
	if err != nil {
		t.Fatalf("InstallProfile() unexpected error: %v", err)
	}

	// Check the destination path.
	wantPath := filepath.Join(profilesDir, "test-profile.env")
	if destPath != wantPath {
		t.Errorf("InstallProfile() destPath = %q, want %q", destPath, wantPath)
	}

	// Check the file was written.
	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("cannot read installed profile: %v", err)
	}
	if string(data) != string(profileContent) {
		t.Errorf("installed content = %q, want %q", string(data), string(profileContent))
	}
}

func TestInstallProfileWithChecksum(t *testing.T) {
	orig := DownloadProfileContent
	defer func() { DownloadProfileContent = orig }()

	profileContent := []byte("PROFILE_ID=checksum-test\nTOOL=codex\n")
	sum := sha256.Sum256(profileContent)
	hexSum := hex.EncodeToString(sum[:])

	DownloadProfileContent = func(_ context.Context, entry *ProfileEntry) ([]byte, error) {
		return profileContent, nil
	}

	profilesDir := t.TempDir()
	entry := &ProfileEntry{
		Name:   "checksum-profile",
		Title:  "Checksum Test",
		SHA256: hexSum,
	}

	destPath, err := InstallProfile(context.Background(), entry, profilesDir)
	if err != nil {
		t.Fatalf("InstallProfile() with valid checksum unexpected error: %v", err)
	}

	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("cannot read installed profile: %v", err)
	}
	if string(data) != string(profileContent) {
		t.Errorf("installed content = %q, want %q", string(data), string(profileContent))
	}
}

func TestInstallProfileChecksumMismatch(t *testing.T) {
	orig := DownloadProfileContent
	defer func() { DownloadProfileContent = orig }()

	profileContent := []byte("PROFILE_ID=bad-checksum\nTOOL=opencode\n")
	wantSum := "0000000000000000000000000000000000000000000000000000000000000000"

	DownloadProfileContent = func(_ context.Context, entry *ProfileEntry) ([]byte, error) {
		return profileContent, nil
	}

	profilesDir := t.TempDir()
	entry := &ProfileEntry{
		Name:   "bad-checksum-profile",
		Title:  "Bad Checksum",
		SHA256: wantSum,
	}

	_, err := InstallProfile(context.Background(), entry, profilesDir)
	if err == nil {
		t.Fatal("InstallProfile() expected checksum error, got nil")
	}
	if !IsChecksumError(err) {
		t.Fatalf("InstallProfile() error type = %T, want ChecksumError", err)
	}
}

func TestInstallProfileDownloadError(t *testing.T) {
	orig := DownloadProfileContent
	defer func() { DownloadProfileContent = orig }()

	DownloadProfileContent = func(_ context.Context, entry *ProfileEntry) ([]byte, error) {
		return nil, errors.New("connection refused")
	}

	profilesDir := t.TempDir()
	entry := &ProfileEntry{Name: "test", Title: "Test"}

	_, err := InstallProfile(context.Background(), entry, profilesDir)
	if err == nil {
		t.Fatal("InstallProfile() expected download error, got nil")
	}
	if IsChecksumError(err) {
		t.Fatal("InstallProfile() should not be ChecksumError for download failure")
	}
}

func TestInstallProfileCreatesDir(t *testing.T) {
	orig := DownloadProfileContent
	defer func() { DownloadProfileContent = orig }()

	profileContent := []byte("PROFILE_ID=test\nTOOL=claude\n")
	DownloadProfileContent = func(_ context.Context, entry *ProfileEntry) ([]byte, error) {
		return profileContent, nil
	}

	// Use a nested path that doesn't exist yet.
	baseDir := t.TempDir()
	nestedDir := filepath.Join(baseDir, "sub", "dir", "profiles")

	entry := &ProfileEntry{Name: "nested-test", Title: "Nested"}
	destPath, err := InstallProfile(context.Background(), entry, nestedDir)
	if err != nil {
		t.Fatalf("InstallProfile() with nested dir error: %v", err)
	}

	if _, err := os.Stat(destPath); err != nil {
		t.Errorf("installed profile not found: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Test for empty profile content
// ---------------------------------------------------------------------------

func TestInstallProfileEmptyContent(t *testing.T) {
	orig := DownloadProfileContent
	defer func() { DownloadProfileContent = orig }()

	DownloadProfileContent = func(_ context.Context, entry *ProfileEntry) ([]byte, error) {
		return []byte{}, nil
	}

	profilesDir := t.TempDir()
	entry := &ProfileEntry{Name: "empty-test", Title: "Empty"}
	destPath, err := InstallProfile(context.Background(), entry, profilesDir)
	if err != nil {
		t.Fatalf("InstallProfile() empty content error: %v", err)
	}

	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("cannot read installed profile: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected empty file, got %d bytes", len(data))
	}
}
