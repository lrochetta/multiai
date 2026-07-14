// Package update checks for newer multiai releases and delegates explicit
// updates to the package manager that owns the current installation.
//
// Startup checks are notification-only: this package never downloads or
// executes a release binary and never terminates the caller's process.
package update

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	checkInterval         = 1 * time.Hour
	requestTimeout        = 5 * time.Second
	maxRequestTimeout     = 10 * time.Second
	automaticCheckTimeout = 5 * time.Second
	maxResponseBytes      = 1 << 20
	maxCommandOutputBytes = 64 << 10
)

// ErrUnsupportedInstall is returned when multiai cannot prove which package
// manager owns the currently running executable. Refusing is safer than
// overwriting an arbitrary binary or running an unverified temporary one.
var ErrUnsupportedInstall = errors.New("unsupported multiai installation")

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

// CheckResult describes a notification-only release check.
type CheckResult struct {
	LatestVersion string
	HasUpdate     bool
}

// InstallResult confirms a package-manager update at its persistent path.
type InstallResult struct {
	Manager        string
	Version        string
	ExecutablePath string
}

var (
	executablePath = os.Executable
	lookPath       = exec.LookPath
	now            = time.Now
	runCommand     = runCommandBounded
	fetchLatest    = FetchLatestRelease
)

// Check is the startup entry point. It only fetches release metadata, updates
// the rate-limit cache, and prints a notification when a newer version exists.
// It never downloads an archive, starts a replacement binary, or calls os.Exit.
func Check(ctx context.Context, currentVersion string) {
	if os.Getenv("MULTIAI_SKIP_UPDATE") != "" || !ShouldCheck() {
		return
	}

	result, err := CheckLatest(ctx, currentVersion)
	if err != nil {
		// Cache failed automatic attempts to avoid a request on every command.
		_ = WriteCache(Cache{LastCheck: now()})
		return
	}
	_ = WriteCache(Cache{LastCheck: now(), LatestVersion: result.LatestVersion})
	if result.HasUpdate {
		fmt.Fprintf(os.Stderr, "[update] Nouvelle version %s disponible; lancez 'multiai update'.\n", result.LatestVersion)
	}
}

// CheckLatest performs a bounded, notification-only release metadata request.
func CheckLatest(ctx context.Context, currentVersion string) (CheckResult, error) {
	ctx, cancel := context.WithTimeout(ctx, automaticCheckTimeout)
	defer cancel()

	rel, err := fetchLatest(ctx)
	if err != nil {
		return CheckResult{}, err
	}
	version, err := normalizeReleaseVersion(rel.TagName)
	if err != nil {
		return CheckResult{}, err
	}
	return CheckResult{
		LatestVersion: version,
		HasUpdate:     IsNewer(currentVersion, version),
	}, nil
}

// ShouldCheck returns true when the cache is absent or older than one hour.
func ShouldCheck() bool {
	entry, err := ReadCache()
	if err != nil {
		return true
	}
	return now().Sub(entry.LastCheck).Truncate(time.Second) > checkInterval
}

// FetchLatestRelease calls the hardcoded GitHub Releases API endpoint. A test
// endpoint override is accepted only in explicit development mode and remains
// restricted to api.github.com.
func FetchLatestRelease(ctx context.Context) (*Release, error) {
	apiURL := "https://api.github.com/repos/lrochetta/multiai/releases/latest"
	if devURL := os.Getenv("MULTIAI_GITHUB_API_URL"); devURL != "" {
		if os.Getenv("MULTIAI_DEV") != "1" {
			return nil, fmt.Errorf("MULTIAI_GITHUB_API_URL requires MULTIAI_DEV=1") //nolint:staticcheck
		}
		if !strings.HasPrefix(devURL, "https://api.github.com/") {
			return nil, fmt.Errorf("MULTIAI_GITHUB_API_URL must start with https://api.github.com/") //nolint:staticcheck
		}
		apiURL = devURL
	}

	return fetchReleaseFromURL(ctx, apiURL)
}

func fetchReleaseFromURL(ctx context.Context, url string) (*Release, error) {
	timeout := requestTimeout
	if value := os.Getenv("MULTIAI_UPDATE_TIMEOUT"); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil && parsed > 0 {
			if parsed > maxRequestTimeout {
				parsed = maxRequestTimeout
			}
			timeout = parsed
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

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxResponseBytes {
		return nil, fmt.Errorf("release metadata exceeds %d bytes", maxResponseBytes)
	}

	var rel Release
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&rel); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("release metadata contains trailing JSON data")
	}
	if _, err := normalizeReleaseVersion(rel.TagName); err != nil {
		return nil, err
	}
	if rel.Assets == nil {
		rel.Assets = []ReleaseAsset{}
	}
	return &rel, nil
}

// InstallRelease delegates an explicit update to the package manager that owns
// the current executable. It currently supports the official npm layout. It
// reports success only after the persistent binary returns the target version.
func InstallRelease(ctx context.Context, rel *Release) (*InstallResult, error) {
	if rel == nil {
		return nil, fmt.Errorf("release is nil")
	}
	version, err := normalizeReleaseVersion(rel.TagName)
	if err != nil {
		return nil, err
	}

	currentExe, err := executablePath()
	if err != nil {
		return nil, fmt.Errorf("resolve current executable: %w", err)
	}
	currentExe, err = filepath.Abs(currentExe)
	if err != nil {
		return nil, fmt.Errorf("resolve absolute executable path: %w", err)
	}
	if !isNPMInstallPath(currentExe) {
		return nil, fmt.Errorf("%w at %q; automatic replacement refused (reinstall with npm or use your original package manager)", ErrUnsupportedInstall, currentExe)
	}

	command, prefixArgs, err := resolveNPMCommand()
	if err != nil {
		return nil, err
	}
	if err := verifyNPMOwnsInstall(ctx, command, prefixArgs, currentExe); err != nil {
		return nil, err
	}
	args := append(prefixArgs,
		"install",
		"--global",
		"--ignore-scripts=false",
		"--no-audit",
		"--no-fund",
		"multiai@"+version,
	)
	output, err := runCommand(ctx, command, args...)
	if err != nil {
		return nil, fmt.Errorf("npm did not complete the persistent update: %w%s", err, formatCommandOutput(output))
	}

	verifyOutput, err := runCommand(ctx, currentExe, "--version")
	if err != nil {
		return nil, fmt.Errorf("npm completed but the installed executable could not be verified: %w%s", err, formatCommandOutput(verifyOutput))
	}
	installedVersion, err := parseVersionOutput(string(verifyOutput))
	if err != nil {
		return nil, fmt.Errorf("npm completed but returned an unverifiable executable: %w", err)
	}
	if installedVersion != version {
		return nil, fmt.Errorf("npm completed but persistent executable reports %s instead of %s; update not confirmed", installedVersion, version)
	}

	return &InstallResult{Manager: "npm", Version: version, ExecutablePath: currentExe}, nil
}

func verifyNPMOwnsInstall(ctx context.Context, command string, prefixArgs []string, currentExe string) error {
	args := append(append([]string(nil), prefixArgs...), "root", "--global")
	output, err := runCommand(ctx, command, args...)
	if err != nil {
		return fmt.Errorf("cannot verify npm ownership of the current installation: %w%s", err, formatCommandOutput(output))
	}

	rootOutput := strings.TrimSpace(string(output))
	if rootOutput == "" || strings.ContainsAny(rootOutput, "\r\n") {
		return fmt.Errorf("npm returned an invalid global root %q; update refused", rootOutput)
	}
	npmRoot, err := filepath.Abs(rootOutput)
	if err != nil {
		return fmt.Errorf("resolve npm global root: %w", err)
	}
	wantRoot := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(currentExe))))
	if !pathsEqual(npmRoot, wantRoot) {
		return fmt.Errorf("npm at %q owns global root %q, not the current installation at %q; possible PATH hijack, update refused", command, npmRoot, currentExe)
	}
	return nil
}

func pathsEqual(a, b string) bool {
	a = filepath.Clean(a)
	b = filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func resolveNPMCommand() (string, []string, error) {
	npmPath, err := lookPath("npm")
	if err != nil {
		return "", nil, fmt.Errorf("npm installation detected but npm is unavailable: %w", err)
	}
	npmPath, err = filepath.Abs(npmPath)
	if err != nil {
		return "", nil, fmt.Errorf("resolve npm path: %w", err)
	}
	if runtime.GOOS != "windows" {
		return npmPath, nil, nil
	}

	ext := strings.ToLower(filepath.Ext(npmPath))
	if ext != ".cmd" && ext != ".bat" {
		return npmPath, nil, nil
	}

	// Avoid cmd.exe and shell parsing: execute npm-cli.js with its adjacent,
	// persistent node.exe when npm resolves to the standard Windows batch shim.
	dir := filepath.Dir(npmPath)
	nodePath := filepath.Join(dir, "node.exe")
	npmCLIPath := filepath.Join(dir, "node_modules", "npm", "bin", "npm-cli.js")
	if !isRegularFile(nodePath) || !isRegularFile(npmCLIPath) {
		return "", nil, fmt.Errorf("npm batch shim found at %q but its adjacent node.exe/npm-cli.js pair is unavailable; update refused", npmPath)
	}
	return nodePath, []string{npmCLIPath}, nil
}

func isRegularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func isNPMInstallPath(path string) bool {
	parts := splitPath(filepath.Clean(path))
	if len(parts) < 5 {
		return false
	}
	want := []string{"node_modules", "multiai", "bin", "native"}
	start := len(parts) - 5
	for i, component := range want {
		if !pathComponentEqual(parts[start+i], component) {
			return false
		}
	}
	name := parts[len(parts)-1]
	return pathComponentEqual(name, "multiai") || pathComponentEqual(name, "multiai.exe")
}

func splitPath(path string) []string {
	volume := filepath.VolumeName(path)
	path = strings.TrimPrefix(path, volume)
	return strings.FieldsFunc(path, func(char rune) bool {
		return char == '/' || char == '\\'
	})
}

func pathComponentEqual(a, b string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func parseVersionOutput(output string) (string, error) {
	fields := strings.Fields(strings.TrimSpace(output))
	if len(fields) != 2 || fields[0] != "multiai" {
		return "", fmt.Errorf("unexpected --version output %q", strings.TrimSpace(output))
	}
	return normalizeReleaseVersion(fields[1])
}

func normalizeReleaseVersion(version string) (string, error) {
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	parts := strings.SplitN(version, "-", 2)
	core := strings.Split(parts[0], ".")
	if len(core) != 3 {
		return "", fmt.Errorf("invalid release version %q", version)
	}
	for _, part := range core {
		if part == "" || !allVersionChars(part, false) {
			return "", fmt.Errorf("invalid release version %q", version)
		}
	}
	if len(parts) == 2 && (parts[1] == "" || !allVersionChars(parts[1], true)) {
		return "", fmt.Errorf("invalid release version %q", version)
	}
	return version, nil
}

func allVersionChars(value string, allowSeparators bool) bool {
	for _, char := range value {
		if char >= '0' && char <= '9' {
			continue
		}
		if allowSeparators && ((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || char == '.' || char == '-') {
			continue
		}
		return false
	}
	return true
}

func runCommandBounded(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var output bytes.Buffer
	limited := &boundedWriter{destination: &output, remaining: maxCommandOutputBytes}
	cmd.Stdout = limited
	cmd.Stderr = limited
	err := cmd.Run()
	return output.Bytes(), err
}

type boundedWriter struct {
	destination *bytes.Buffer
	remaining   int64
}

func (writer *boundedWriter) Write(data []byte) (int, error) {
	originalLength := len(data)
	if writer.remaining <= 0 {
		return originalLength, nil
	}
	if int64(len(data)) > writer.remaining {
		data = data[:writer.remaining]
	}
	written, err := writer.destination.Write(data)
	writer.remaining -= int64(written)
	if err != nil {
		return written, err
	}
	return originalLength, nil
}

func formatCommandOutput(output []byte) string {
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return ""
	}
	return ": " + strings.Map(func(char rune) rune {
		if char == '\n' || char == '\r' || char == '\t' || char >= 0x20 {
			return char
		}
		return -1
	}, trimmed)
}

// IsNewer compares semver-like version strings. Invalid versions fail closed.
func IsNewer(current, latest string) bool {
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")
	curParts := strings.SplitN(current, ".", 3)
	latParts := strings.SplitN(latest, ".", 3)
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
	return latSuffix == "" && curSuffix != ""
}

func parseVersionPart(value string) (num int, suffix string, ok bool) {
	if value == "" {
		return 0, "", false
	}
	for i, char := range value {
		if char < '0' || char > '9' {
			if i == 0 {
				return 0, "", false
			}
			number, err := strconv.Atoi(value[:i])
			if err != nil {
				return 0, "", false
			}
			return number, value[i:], true
		}
	}
	number, err := strconv.Atoi(value)
	if err != nil {
		return 0, "", false
	}
	return number, "", true
}

// CacheFilePath returns the absolute update-check cache path.
func CacheFilePath() string {
	dir := cacheDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "update-check.json")
}

// ReadCache reads the cache entry from disk.
func ReadCache() (*Cache, error) {
	path := CacheFilePath()
	if path == "" {
		return nil, fmt.Errorf("cannot resolve cache path")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cache Cache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}
	return &cache, nil
}

// WriteCache atomically writes a cache entry to disk.
func WriteCache(cache Cache) error {
	path := CacheFilePath()
	if path == "" {
		return fmt.Errorf("cannot resolve cache path")
	}
	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(path, data, 0644)
}

func cacheDir() string {
	if dir := os.Getenv("MULTIAI_CACHE_DIR"); dir != "" {
		return dir
	}
	configDir, err := os.UserConfigDir()
	if err == nil {
		return filepath.Join(configDir, "multiai")
	}
	return filepath.Join(os.TempDir(), "multiai")
}
