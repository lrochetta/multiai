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

func checksumHex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

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
		SHA256:      checksumHex(profileContent),
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
	entry := &ProfileEntry{
		Name:   "test",
		Title:  "Test",
		SHA256: strings.Repeat("0", sha256.Size*2),
	}

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

	entry := &ProfileEntry{
		Name:   "nested-test",
		Title:  "Nested",
		SHA256: checksumHex(profileContent),
	}
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
	entry := &ProfileEntry{
		Name:   "empty-test",
		Title:  "Empty",
		SHA256: checksumHex(nil),
	}
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

func TestValidateProfileNamePortablePolicy(t *testing.T) {
	valid := []string{
		"a",
		"profile-1",
		"profile_name",
		"a" + strings.Repeat("b", 63),
	}
	for _, name := range valid {
		t.Run("valid_"+name, func(t *testing.T) {
			if err := ValidateProfileName(name); err != nil {
				t.Errorf("ValidateProfileName(%q) unexpected error: %v", name, err)
			}
		})
	}

	invalid := []string{
		"",
		".",
		"..",
		"../escape",
		`..\escape`,
		"/tmp/escape",
		`C:\escape`,
		"C:/escape",
		`\\server\share`,
		"//server/share",
		`\\?\C:\escape`,
		"a/b",
		`a\b`,
		"a:b",
		"a..b",
		"Uppercase",
		"-leading",
		"_leading",
		"with space",
		"profil" + string(rune(0x00e9)),
		"a" + strings.Repeat("b", 64),
	}
	for _, name := range invalid {
		t.Run(fmt.Sprintf("invalid_%q", name), func(t *testing.T) {
			if err := ValidateProfileName(name); err == nil {
				t.Errorf("ValidateProfileName(%q) expected error", name)
			}
		})
	}
}

func TestResolveInstallPathConfinesDestination(t *testing.T) {
	root := t.TempDir()
	entry := &ProfileEntry{
		Name:   "safe-profile",
		SHA256: strings.Repeat("0", sha256.Size*2),
	}

	dest, err := ResolveInstallPath(entry, root)
	if err != nil {
		t.Fatalf("ResolveInstallPath() unexpected error: %v", err)
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(rootAbs, dest)
	if err != nil {
		t.Fatal(err)
	}
	if rel != "safe-profile.env" {
		t.Fatalf("destination relative path = %q, want safe-profile.env", rel)
	}
	if filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		t.Fatalf("destination escaped root: root=%q dest=%q rel=%q", rootAbs, dest, rel)
	}
}

func TestInstallProfileRejectsTraversalBeforeDownloadOrWrite(t *testing.T) {
	orig := DownloadProfileContent
	defer func() { DownloadProfileContent = orig }()

	downloadCalls := 0
	DownloadProfileContent = func(_ context.Context, _ *ProfileEntry) ([]byte, error) {
		downloadCalls++
		return []byte("unexpected"), nil
	}

	for _, name := range []string{"../escape", `..\escape`, "/tmp/escape", `C:\escape`, `\\server\share`} {
		t.Run(fmt.Sprintf("%q", name), func(t *testing.T) {
			profilesDir := filepath.Join(t.TempDir(), "profiles")
			entry := &ProfileEntry{
				Name:   name,
				SHA256: strings.Repeat("0", sha256.Size*2),
			}
			if _, err := InstallProfile(context.Background(), entry, profilesDir); err == nil {
				t.Fatalf("InstallProfile(%q) expected error", name)
			}
			if _, err := os.Stat(profilesDir); !os.IsNotExist(err) {
				t.Fatalf("profiles directory should not be created, stat error: %v", err)
			}
		})
	}
	if downloadCalls != 0 {
		t.Fatalf("unsafe names triggered %d download(s)", downloadCalls)
	}
}

func TestInstallProfileRequiresChecksumBeforeDownloadOrWrite(t *testing.T) {
	orig := DownloadProfileContent
	defer func() { DownloadProfileContent = orig }()

	downloadCalls := 0
	DownloadProfileContent = func(_ context.Context, _ *ProfileEntry) ([]byte, error) {
		downloadCalls++
		return []byte("unexpected"), nil
	}

	profilesDir := filepath.Join(t.TempDir(), "profiles")
	_, err := InstallProfile(context.Background(), &ProfileEntry{Name: "safe"}, profilesDir)
	if err == nil || !IsChecksumError(err) {
		t.Fatalf("InstallProfile() error = %v, want checksum metadata error", err)
	}
	if downloadCalls != 0 {
		t.Fatalf("missing checksum triggered %d download(s)", downloadCalls)
	}
	if _, err := os.Stat(profilesDir); !os.IsNotExist(err) {
		t.Fatalf("profiles directory should not be created, stat error: %v", err)
	}
}

func TestInstallProfileRejectsOversizedContent(t *testing.T) {
	orig := DownloadProfileContent
	defer func() { DownloadProfileContent = orig }()

	content := make([]byte, maxProfileBytes+1)
	DownloadProfileContent = func(_ context.Context, _ *ProfileEntry) ([]byte, error) {
		return content, nil
	}

	profilesDir := filepath.Join(t.TempDir(), "profiles")
	entry := &ProfileEntry{Name: "large", SHA256: checksumHex(content)}
	if _, err := InstallProfile(context.Background(), entry, profilesDir); err == nil || !strings.Contains(err.Error(), "maximum size") {
		t.Fatalf("InstallProfile() error = %v, want maximum size error", err)
	}
	if _, err := os.Stat(profilesDir); !os.IsNotExist(err) {
		t.Fatalf("profiles directory should not be created, stat error: %v", err)
	}
}

func TestResolveInstallPathRejectsMalformedChecksum(t *testing.T) {
	for _, checksum := range []string{"", "abc", strings.Repeat("z", sha256.Size*2)} {
		entry := &ProfileEntry{Name: "safe", SHA256: checksum}
		if _, err := ResolveInstallPath(entry, t.TempDir()); err == nil || !IsChecksumError(err) {
			t.Errorf("ResolveInstallPath(checksum=%q) error = %v, want checksum error", checksum, err)
		}
	}
}

func TestInstallProfileRejectsSymlinkedProfilesDirectoryBeforeDownload(t *testing.T) {
	base := t.TempDir()
	realDir := filepath.Join(base, "real")
	if err := os.Mkdir(realDir, 0755); err != nil {
		t.Fatal(err)
	}
	linkedDir := filepath.Join(base, "profiles")
	if err := os.Symlink(realDir, linkedDir); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	original := DownloadProfileContent
	defer func() { DownloadProfileContent = original }()
	downloads := 0
	content := []byte("PROFILE_ID=unsafe\n")
	DownloadProfileContent = func(context.Context, *ProfileEntry) ([]byte, error) {
		downloads++
		return content, nil
	}

	entry := &ProfileEntry{Name: "unsafe", SHA256: checksumHex(content)}
	if _, err := InstallProfile(context.Background(), entry, linkedDir); err == nil {
		t.Fatal("InstallProfile() accepted a symlinked profiles directory")
	}
	if downloads != 0 {
		t.Fatalf("unsafe path triggered %d download(s)", downloads)
	}
	if _, err := os.Stat(filepath.Join(realDir, "unsafe.env")); !os.IsNotExist(err) {
		t.Fatalf("file escaped through symlink, stat error: %v", err)
	}
}

func TestInstallProfileRejectsSymlinkDestinationBeforeDownload(t *testing.T) {
	profilesDir := t.TempDir()
	target := filepath.Join(t.TempDir(), "outside.env")
	if err := os.WriteFile(target, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(profilesDir, "unsafe.env")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	original := DownloadProfileContent
	defer func() { DownloadProfileContent = original }()
	downloads := 0
	content := []byte("changed")
	DownloadProfileContent = func(context.Context, *ProfileEntry) ([]byte, error) {
		downloads++
		return content, nil
	}

	entry := &ProfileEntry{Name: "unsafe", SHA256: checksumHex(content)}
	if _, err := InstallProfile(context.Background(), entry, profilesDir); err == nil {
		t.Fatal("InstallProfile() accepted a symlink destination")
	}
	if downloads != 0 {
		t.Fatalf("unsafe destination triggered %d download(s)", downloads)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "keep" {
		t.Fatalf("outside target changed: data=%q err=%v", data, err)
	}
}
