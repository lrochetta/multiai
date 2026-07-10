// Package update provides self-update logic for the multiai CLI.
//
// It checks the GitHub Releases API for a newer version, downloads and
// verifies the matching platform archive, extracts the binary, and re-execs
// the new version — all silently, without blocking startup on failure.
package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/lrochetta/multiai/internal/fsutil"
)

const (
	repoOwner      = "lrochetta"
	repoName       = "multiai"
	checkInterval  = 1 * time.Hour
	requestTimeout = 5 * time.Second
)

// Cache holds the last update-check timestamp and the latest version seen.
type Cache struct {
	LastCheck     time.Time `json:"last_check"`
	LatestVersion string    `json:"latest_version,omitempty"`
}

// Release represents a GitHub release fetched from the Releases API.
type Release struct {
	TagName string         `json:"tag_name"`
	Assets  []ReleaseAsset `json:"assets"`
}

// ReleaseAsset is a single file attached to a GitHub release.
type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Check is the main entry point for auto-update. It is safe to call on every
// startup: all errors are silently logged to stderr and never block the launch.
func Check(ctx context.Context, currentVersion string) {
	if os.Getenv("MULTIAI_SKIP_UPDATE") != "" {
		return
	}
	if !ShouldCheck() {
		return
	}

	rel, err := FetchLatestRelease(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[update] fetch latest release: %v\n", err)
		return
	}
	if !IsNewer(currentVersion, rel.TagName) {
		_ = WriteCache(Cache{LastCheck: time.Now(), LatestVersion: rel.TagName})
		return
	}

	newExe, err := downloadAndVerifyRelease(ctx, currentVersion, rel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[update] download and verify: %v\n", err)
		_ = WriteCache(Cache{LastCheck: time.Now(), LatestVersion: rel.TagName})
		return
	}
	execNewBinary(newExe)
}

// ShouldCheck returns true when the last check was more than checkInterval ago
// or when no cache exists yet. The duration is truncated to second precision
// to absorb serialization and filesystem overhead at the boundary.
func ShouldCheck() bool {
	entry, err := ReadCache()
	if err != nil {
		return true
	}
	return time.Since(entry.LastCheck).Truncate(time.Second) > checkInterval
}

// FetchLatestRelease calls the GitHub Releases API and returns the latest
// release metadata. The API URL is hardcoded to the production GitHub API for
// the lrochetta/multiai repository. A different URL can be set via
// MULTIAI_GITHUB_API_URL, but only when MULTIAI_DEV=1 is also set and the
// provided URL starts with https://api.github.com/. The request timeout can
// be set via MULTIAI_UPDATE_TIMEOUT (default 5s).
func FetchLatestRelease(ctx context.Context) (*Release, error) {
	apiURL := "https://api.github.com/repos/lrochetta/multiai/releases/latest"
	if devURL := os.Getenv("MULTIAI_GITHUB_API_URL"); devURL != "" {
		if os.Getenv("MULTIAI_DEV") != "1" {
			return nil, fmt.Errorf("MULTIAI_GITHUB_API_URL requires MULTIAI_DEV=1")
		}
		if !strings.HasPrefix(devURL, "https://api.github.com/") {
			return nil, fmt.Errorf("MULTIAI_GITHUB_API_URL must start with https://api.github.com/")
		}
		apiURL = devURL
	}

	return fetchReleaseFromURL(ctx, apiURL)
}

// fetchReleaseFromURL performs the HTTP request and decodes the response.
func fetchReleaseFromURL(ctx context.Context, url string) (*Release, error) {
	timeout := requestTimeout
	if t := os.Getenv("MULTIAI_UPDATE_TIMEOUT"); t != "" {
		if d, err := time.ParseDuration(t); err == nil && d > 0 {
			timeout = d
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "multiai-updater")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	if rel.Assets == nil {
		rel.Assets = []ReleaseAsset{}
	}
	return &rel, nil
}

// IsNewer compares two semver-like version strings and returns true when
// latest > current. It strips any leading "v" prefix, compares
// major.minor.patch numerically, and treats a missing pre-release suffix as
// newer than one that has it (e.g. "1.0.1" > "1.0.1-beta"). Invalid or
// unparseable versions cause a graceful return of false.
func IsNewer(current, latest string) bool {
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")

	curParts := strings.SplitN(current, ".", 3)
	latParts := strings.SplitN(latest, ".", 3)

	// Pad short version strings to at least 3 parts.
	for len(curParts) < 3 {
		curParts = append(curParts, "0")
	}
	for len(latParts) < 3 {
		latParts = append(latParts, "0")
	}

	var curSuffix, latSuffix string
	for i := 0; i < 3; i++ {
		curNum, curRest, curOK := parseVersionPart(curParts[i])
		latNum, latRest, latOK := parseVersionPart(latParts[i])
		if !curOK || !latOK {
			return false
		}
		if curRest != "" {
			curSuffix = curRest
		}
		if latRest != "" {
			latSuffix = latRest
		}
		if latNum > curNum {
			return true
		}
		if latNum < curNum {
			return false
		}
	}

	// All three numeric parts are equal.
	// A bare release (no suffix) is newer than a pre-release of the same
	// version, e.g. "1.0.1" > "1.0.1-beta".
	if latSuffix == "" && curSuffix != "" {
		return true
	}
	return false
}

// parseVersionPart extracts the leading numeric value and the remaining
// suffix (if any) from a single semver segment. ok is false when the segment
// has no leading digits at all.
func parseVersionPart(s string) (num int, suffix string, ok bool) {
	if s == "" {
		return 0, "", false
	}
	for i, c := range s {
		if c < '0' || c > '9' {
			if i == 0 {
				return 0, "", false // no leading digits
			}
			n, err := strconv.Atoi(s[:i])
			if err != nil {
				return 0, "", false
			}
			return n, s[i:], true
		}
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, "", false
	}
	return n, "", true
}

// downloadAndVerifyRelease handles the full update pipeline: locate the
// correct platform archive and checksums in the release assets, download,
// verify SHA256, extract the binary, and return its temporary path.
func downloadAndVerifyRelease(ctx context.Context, currentVersion string, rel *Release) (string, error) {
	target := GetTarget()
	if target == "" {
		return "", fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	isWindows := runtime.GOOS == "windows"
	ext := ".tar.gz"
	if isWindows {
		ext = ".zip"
	}

	version := strings.TrimPrefix(rel.TagName, "v")
	archiveName := fmt.Sprintf("multiai_%s_%s%s", version, target, ext)
	binaryName := "multiai"
	if isWindows {
		binaryName = "multiai.exe"
	}

	// Locate archive and checksums assets.
	var archiveURL, checksumsURL, sigURL, certURL string
	for _, a := range rel.Assets {
		switch a.Name {
		case archiveName:
			archiveURL = a.BrowserDownloadURL
		case "checksums.txt":
			checksumsURL = a.BrowserDownloadURL
		case "checksums.txt.sig":
			sigURL = a.BrowserDownloadURL
		case "checksums.txt.pem":
			certURL = a.BrowserDownloadURL
		}
	}
	if archiveURL == "" {
		return "", fmt.Errorf("archive %s not found in release assets", archiveName)
	}
	if checksumsURL == "" {
		return "", fmt.Errorf("checksums.txt not found in release assets")
	}

	// Fetch checksums.
	checksumsData, err := fetchRaw(ctx, checksumsURL)
	if err != nil {
		return "", fmt.Errorf("fetch checksums: %w", err)
	}

	// Cosign signature verification.
	if sigURL != "" && certURL != "" {
		fmt.Fprintf(os.Stderr, "[update] Vérification Cosign...\n")

		sigData, err := fetchRaw(ctx, sigURL)
		if err != nil {
			return "", fmt.Errorf("fetch cosign signature: %w", err)
		}
		certData, err := fetchRaw(ctx, certURL)
		if err != nil {
			return "", fmt.Errorf("fetch cosign certificate: %w", err)
		}

		cosignTmpDir, err := os.MkdirTemp("", "multiai-cosign")
		if err != nil {
			return "", fmt.Errorf("create cosign temp dir: %w", err)
		}
		defer os.RemoveAll(cosignTmpDir)

		checksumsPath := filepath.Join(cosignTmpDir, "checksums.txt")
		sigPath := filepath.Join(cosignTmpDir, "checksums.txt.sig")
		certPath := filepath.Join(cosignTmpDir, "checksums.txt.pem")

		if err := os.WriteFile(checksumsPath, checksumsData, 0644); err != nil {
			return "", fmt.Errorf("write checksums temp file: %w", err)
		}
		if err := os.WriteFile(sigPath, sigData, 0644); err != nil {
			return "", fmt.Errorf("write sig temp file: %w", err)
		}
		if err := os.WriteFile(certPath, certData, 0644); err != nil {
			return "", fmt.Errorf("write cert temp file: %w", err)
		}

		if err := verifyCosignSignature(checksumsPath, sigPath, certPath); err != nil {
			return "", fmt.Errorf("cosign verification failed: %w", err)
		}

		fmt.Fprintf(os.Stderr, "[update] [OK] Cosign vérifié\n")
	}

	// Fetch archive.
	archiveData, err := fetchRaw(ctx, archiveURL)
	if err != nil {
		return "", fmt.Errorf("fetch archive: %w", err)
	}

	// Verify SHA256.
	expected := findChecksum(string(checksumsData), archiveName)
	if expected == "" {
		return "", fmt.Errorf("%s not found in checksums.txt", archiveName)
	}
	actual := sha256Hex(archiveData)
	if actual != expected {
		return "", fmt.Errorf("SHA256 mismatch for %s", archiveName)
	}

	// Extract to a temporary directory.
	tmpDir, err := os.MkdirTemp("", "multiai-update")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}

	var binaryPath string
	if isWindows {
		binaryPath, err = extractZip(archiveData, tmpDir, binaryName)
	} else {
		binaryPath, err = extractTarGz(archiveData, tmpDir, binaryName)
	}
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("extract: %w", err)
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(binaryPath, 0755); err != nil {
			os.RemoveAll(tmpDir)
			return "", err
		}
	}

	return binaryPath, nil
}

// verifyCosignSignature verifies the checksums blob using a Cosign signature
// and Fulcio certificate. It returns nil on success, or if cosign is not
// installed and MULTIAI_REQUIRE_COSIGN is unset.
func verifyCosignSignature(checksumsPath, sigPath, certPath string) error {
	cosignPath, err := exec.LookPath("cosign")
	if err != nil {
		if os.Getenv("MULTIAI_REQUIRE_COSIGN") == "1" {
			return fmt.Errorf("cosign not found and MULTIAI_REQUIRE_COSIGN=1: %w", err)
		}
		fmt.Fprintf(os.Stderr, "[update] cosign not found, skipping signature verification\n")
		return nil
	}

	args := []string{
		"verify-blob",
		"--certificate", certPath,
		"--signature", sigPath,
		"--certificate-identity-regexp", `https://github.com/lrochetta/multiai/.github/workflows/release.yml@refs/tags/v.*`,
		"--certificate-oidc-issuer", "https://token.actions.githubusercontent.com",
		checksumsPath,
	}

	cmd := exec.Command(cosignPath, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cosign verify-blob failed: %w\nOutput: %s", err, string(out))
	}
	return nil
}

// execNewBinary starts the new binary with the same arguments, stdin, stdout,
// and stderr as the current process, then exits the old one.
func execNewBinary(newExe string) {
	args := append([]string{newExe}, os.Args[1:]...)
	proc, err := os.StartProcess(newExe, args, &os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "[update] exec new binary: %v\n", err)
		return
	}
	proc.Release()
	os.Exit(0)
}

// GetTarget returns the platform string used in release archive names
// (e.g. "windows_amd64", "linux_arm64"). It matches the convention used by
// goreleaser and install.js.
func GetTarget() string {
	switch {
	case runtime.GOOS == "windows" && runtime.GOARCH == "amd64":
		return "windows_amd64"
	case runtime.GOOS == "darwin" && runtime.GOARCH == "amd64":
		return "darwin_amd64"
	case runtime.GOOS == "darwin" && runtime.GOARCH == "arm64":
		return "darwin_arm64"
	case runtime.GOOS == "linux" && runtime.GOARCH == "amd64":
		return "linux_amd64"
	case runtime.GOOS == "linux" && runtime.GOARCH == "arm64":
		return "linux_arm64"
	default:
		return ""
	}
}

// CacheFilePath returns the absolute path to the update-check cache file.
// The base directory can be overridden with the MULTIAI_CACHE_DIR env var;
// otherwise it defaults to os.UserConfigDir()/multiai/.
func CacheFilePath() string {
	dir := cacheDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "update-check.json")
}

// ReadCache reads the cache entry from disk. It returns an error when the
// file is missing, unreadable, or contains invalid JSON.
func ReadCache() (*Cache, error) {
	path := CacheFilePath()
	if path == "" {
		return nil, fmt.Errorf("cannot resolve cache path")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Cache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// WriteCache atomically writes a cache entry to disk.
func WriteCache(c Cache) error {
	path := CacheFilePath()
	if path == "" {
		return fmt.Errorf("cannot resolve cache path")
	}
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(path, data, 0644)
}

// DownloadAndVerify downloads a file from url, verifies its SHA256 digest
// matches checksum, and writes it to dest. It creates intermediate
// directories in dest as needed.
func DownloadAndVerify(ctx context.Context, url, checksum, dest string) error {
	data, err := fetchRaw(ctx, url)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	actual := sha256Hex(data)
	if actual != strings.ToLower(checksum) {
		return fmt.Errorf("SHA256 mismatch: got %s, expected %s", actual, checksum)
	}
	dir := filepath.Dir(dest)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(dest, data, 0644)
}

// DownloadRelease is an exported wrapper around downloadAndVerifyRelease,
// allowing external callers (e.g. cmd_update.go) to download, verify, and
// extract the release binary for the current platform.
func DownloadRelease(ctx context.Context, currentVersion string, rel *Release) (string, error) {
	return downloadAndVerifyRelease(ctx, currentVersion, rel)
}

// ExecBinary starts the new binary with the same arguments and exits the
// current process, completing the self-update cycle.
func ExecBinary(newExe string) {
	execNewBinary(newExe)
}

// ── Internal helpers ──────────────────────────────────────────────────────

// cacheDir resolves the base directory for cache files. It respects the
// MULTIAI_CACHE_DIR env var, falling back to os.UserConfigDir()/multiai
// and finally to os.TempDir()/multiai as a last resort.
func cacheDir() string {
	if dir := os.Getenv("MULTIAI_CACHE_DIR"); dir != "" {
		return dir
	}
	cfg, err := os.UserConfigDir()
	if err == nil {
		return filepath.Join(cfg, "multiai")
	}
	return filepath.Join(os.TempDir(), "multiai")
}

// fetchRaw performs an HTTP GET and returns the raw response body.
func fetchRaw(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "multiai-updater")

	client := &http.Client{Timeout: requestTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// findChecksum parses a checksums.txt body (as produced by goreleaser) and
// returns the lowercase SHA256 hex string for the given fileName.
func findChecksum(checksumsText, fileName string) string {
	for _, line := range strings.Split(checksumsText, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: "<sha256hex>  [*]<filename>"
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		name := strings.TrimPrefix(parts[1], "*")
		if name == fileName {
			return strings.ToLower(parts[0])
		}
	}
	return ""
}

// sha256Hex returns the lowercase hex SHA256 digest of data.
func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

// extractZip extracts binaryName from a zip archive into destDir.
func extractZip(data []byte, destDir, binaryName string) (string, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}
	for _, f := range r.File {
		if filepath.Base(f.Name) != binaryName {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		defer rc.Close()

		extractPath := filepath.Join(destDir, binaryName)
		out, err := os.OpenFile(extractPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return "", err
		}
		defer out.Close()

		if _, err := io.Copy(out, rc); err != nil {
			return "", err
		}
		return extractPath, nil
	}
	return "", fmt.Errorf("%s not found in archive", binaryName)
}

// extractTarGz extracts binaryName from a gzipped tar archive into destDir.
func extractTarGz(data []byte, destDir, binaryName string) (string, error) {
	gzr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if filepath.Base(hdr.Name) != binaryName {
			continue
		}
		extractPath := filepath.Join(destDir, binaryName)
		out, err := os.OpenFile(extractPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return "", err
		}
		defer out.Close()

		if _, err := io.Copy(out, tr); err != nil {
			return "", err
		}
		return extractPath, nil
	}
	return "", fmt.Errorf("%s not found in archive", binaryName)
}
