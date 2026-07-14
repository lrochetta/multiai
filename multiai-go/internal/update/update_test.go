package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func restoreHooks(t *testing.T) {
	t.Helper()
	originalExecutablePath := executablePath
	originalLookPath := lookPath
	originalNow := now
	originalRunCommand := runCommand
	originalFetchLatest := fetchLatest
	t.Cleanup(func() {
		executablePath = originalExecutablePath
		lookPath = originalLookPath
		now = originalNow
		runCommand = originalRunCommand
		fetchLatest = originalFetchLatest
	})
}

func TestIsNewer(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    bool
	}{
		{"equal", "0.6.7", "0.6.7", false},
		{"newer patch", "0.6.6", "0.6.7", true},
		{"newer minor", "0.6.7", "0.7.0", true},
		{"newer major", "0.6.7", "1.0.0", true},
		{"older", "1.0.0", "0.9.9", false},
		{"v prefix", "v0.6.6", "v0.6.7", true},
		{"stable after prerelease", "1.0.0-rc1", "1.0.0", true},
		{"invalid current", "dev", "1.0.0", false},
		{"invalid latest", "1.0.0", "latest", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsNewer(test.current, test.latest); got != test.want {
				t.Fatalf("IsNewer(%q, %q) = %v, want %v", test.current, test.latest, got, test.want)
			}
		})
	}
}

func TestNormalizeReleaseVersionRejectsPackageSpecInjection(t *testing.T) {
	invalid := []string{
		"",
		"latest",
		"1.2",
		"1.2.3 --global evil",
		"1.2.3/../../evil",
		"1.2.3;calc.exe",
		"1.2.3+build",
		"v1.2.3\n--force",
	}
	for _, version := range invalid {
		if _, err := normalizeReleaseVersion(version); err == nil {
			t.Errorf("normalizeReleaseVersion(%q) unexpectedly succeeded", version)
		}
	}

	valid := []string{"0.6.7", "v1.2.3", "2.0.0-rc.1"}
	for _, version := range valid {
		if _, err := normalizeReleaseVersion(version); err != nil {
			t.Errorf("normalizeReleaseVersion(%q): %v", version, err)
		}
	}
}

func TestCheckLatestIsMetadataOnlyAndTestable(t *testing.T) {
	restoreHooks(t)
	fetchLatest = func(ctx context.Context) (*Release, error) {
		return &Release{TagName: "v0.7.0"}, nil
	}

	result, err := CheckLatest(context.Background(), "0.6.7")
	if err != nil {
		t.Fatalf("CheckLatest(): %v", err)
	}
	if result.LatestVersion != "0.7.0" || !result.HasUpdate {
		t.Fatalf("CheckLatest() = %+v", result)
	}
}

func TestCheckLatestHonorsCallerDeadline(t *testing.T) {
	restoreHooks(t)
	fetchLatest = func(ctx context.Context) (*Release, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := CheckLatest(ctx, "0.6.7")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("CheckLatest() error = %v, want deadline exceeded", err)
	}
}

func TestReadWriteCache(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_CACHE_DIR", dir)
	want := Cache{
		LastCheck:     time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC),
		LatestVersion: "0.6.7",
	}
	if err := WriteCache(want); err != nil {
		t.Fatalf("WriteCache(): %v", err)
	}
	got, err := ReadCache()
	if err != nil {
		t.Fatalf("ReadCache(): %v", err)
	}
	if !got.LastCheck.Equal(want.LastCheck) || got.LatestVersion != want.LatestVersion {
		t.Fatalf("ReadCache() = %+v, want %+v", got, want)
	}
}

func TestReadCacheErrors(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		t.Setenv("MULTIAI_CACHE_DIR", t.TempDir())
		if _, err := ReadCache(); err == nil {
			t.Fatal("ReadCache() unexpectedly succeeded")
		}
	})

	t.Run("corrupt", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("MULTIAI_CACHE_DIR", dir)
		if err := os.WriteFile(filepath.Join(dir, "update-check.json"), []byte("{bad"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadCache(); err == nil {
			t.Fatal("ReadCache() unexpectedly accepted corrupt JSON")
		}
	})
}

func TestShouldCheckDeterministic(t *testing.T) {
	restoreHooks(t)
	fixedNow := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	now = func() time.Time { return fixedNow }

	tests := []struct {
		name string
		age  time.Duration
		want bool
	}{
		{"recent", 30 * time.Minute, false},
		{"boundary", time.Hour, false},
		{"stale", time.Hour + time.Second, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("MULTIAI_CACHE_DIR", dir)
			if err := WriteCache(Cache{LastCheck: fixedNow.Add(-test.age)}); err != nil {
				t.Fatal(err)
			}
			if got := ShouldCheck(); got != test.want {
				t.Fatalf("ShouldCheck() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestShouldCheckWithoutCache(t *testing.T) {
	t.Setenv("MULTIAI_CACHE_DIR", t.TempDir())
	if !ShouldCheck() {
		t.Fatal("ShouldCheck() = false without cache")
	}
}

func TestCacheFilePathUsesOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_CACHE_DIR", dir)
	want := filepath.Join(dir, "update-check.json")
	if got := CacheFilePath(); got != want {
		t.Fatalf("CacheFilePath() = %q, want %q", got, want)
	}
}

func TestFetchReleaseMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s", r.Method)
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept = %q", r.Header.Get("Accept"))
		}
		fmt.Fprint(w, `{"tag_name":"v1.2.3","assets":[{"name":"checksums.txt"}]}`)
	}))
	defer server.Close()

	release, err := fetchReleaseFromURL(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("fetchReleaseFromURL(): %v", err)
	}
	if release.TagName != "v1.2.3" || len(release.Assets) != 1 {
		t.Fatalf("release = %+v", release)
	}
}

func TestFetchReleaseRejectsUntrustedURL(t *testing.T) {
	if _, err := fetchReleaseFromURL(context.Background(), "https://example.com/release"); err == nil {
		t.Fatal("fetchReleaseFromURL() accepted an untrusted host")
	}
}

func TestFetchReleaseRejectsInvalidMetadata(t *testing.T) {
	tests := []string{
		`{"assets":[]}`,
		`{"tag_name":"latest"}`,
		`{"tag_name":"v1.2.3;evil"}`,
		`{broken`,
	}
	for _, body := range tests {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, body)
			}))
			defer server.Close()
			if _, err := fetchReleaseFromURL(context.Background(), server.URL); err == nil {
				t.Fatal("invalid metadata unexpectedly accepted")
			}
		})
	}
}

func TestFetchReleaseHTTPErrorAndTimeout(t *testing.T) {
	t.Run("status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer server.Close()
		if _, err := fetchReleaseFromURL(context.Background(), server.URL); err == nil {
			t.Fatal("HTTP error unexpectedly accepted")
		}
	})

	t.Run("timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(200 * time.Millisecond)
			fmt.Fprint(w, `{"tag_name":"v1.2.3"}`)
		}))
		defer server.Close()
		t.Setenv("MULTIAI_UPDATE_TIMEOUT", "20ms")
		if _, err := fetchReleaseFromURL(context.Background(), server.URL); err == nil {
			t.Fatal("timeout unexpectedly succeeded")
		}
	})
}

func TestFetchLatestReleaseURLOverrideIsRestricted(t *testing.T) {
	t.Setenv("MULTIAI_GITHUB_API_URL", "https://evil.example/releases/latest")
	t.Setenv("MULTIAI_DEV", "1")
	if _, err := FetchLatestRelease(context.Background()); err == nil {
		t.Fatal("non-GitHub override unexpectedly accepted")
	}
}

func TestIsNPMInstallPath(t *testing.T) {
	binaryName := "multiai"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	valid := filepath.Join(t.TempDir(), "node_modules", "multiai", "bin", "native", binaryName)
	if !isNPMInstallPath(valid) {
		t.Fatalf("official npm path rejected: %q", valid)
	}
	invalid := []string{
		filepath.Join(t.TempDir(), binaryName),
		filepath.Join(t.TempDir(), "node_modules", "other", "bin", "native", binaryName),
		filepath.Join(t.TempDir(), "node_modules", "multiai", "bin", binaryName),
	}
	for _, path := range invalid {
		if isNPMInstallPath(path) {
			t.Errorf("non-npm path accepted: %q", path)
		}
	}
}

func TestInstallReleaseDelegatesExactVersionAndVerifiesPersistentBinary(t *testing.T) {
	restoreHooks(t)
	binaryName := "multiai"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	currentExe := filepath.Join(t.TempDir(), "node_modules", "multiai", "bin", "native", binaryName)
	npmExe := filepath.Join(t.TempDir(), "npm.exe")
	executablePath = func() (string, error) { return currentExe, nil }
	lookPath = func(file string) (string, error) {
		if file != "npm" {
			t.Fatalf("lookPath(%q)", file)
		}
		return npmExe, nil
	}

	type invocation struct {
		name string
		args []string
	}
	var calls []invocation
	runCommand = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		calls = append(calls, invocation{name: name, args: append([]string(nil), args...)})
		if strings.HasSuffix(strings.Join(args, " "), "root --global") {
			return []byte(filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(currentExe)))) + "\n"), nil
		}
		if name == currentExe {
			return []byte("multiai 1.2.3\n"), nil
		}
		return []byte("updated 1 package"), nil
	}

	result, err := InstallRelease(context.Background(), &Release{TagName: "v1.2.3"})
	if err != nil {
		t.Fatalf("InstallRelease(): %v", err)
	}
	if result.Manager != "npm" || result.Version != "1.2.3" || result.ExecutablePath != currentExe {
		t.Fatalf("result = %+v", result)
	}
	if len(calls) != 3 {
		t.Fatalf("calls = %d, want ownership check + manager + verification", len(calls))
	}
	if got := strings.Join(calls[0].args, " "); !strings.HasSuffix(got, "root --global") {
		t.Fatalf("ownership call args = %q", got)
	}
	joined := strings.Join(calls[1].args, " ")
	if !strings.Contains(joined, "install --global") || !strings.Contains(joined, "multiai@1.2.3") {
		t.Fatalf("npm args = %q", joined)
	}
	if calls[2].name != currentExe || strings.Join(calls[2].args, " ") != "--version" {
		t.Fatalf("verification call = %+v", calls[2])
	}
}

func TestInstallReleaseRefusesNPMFromDifferentGlobalRoot(t *testing.T) {
	restoreHooks(t)
	binaryName := "multiai"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	currentExe := filepath.Join(t.TempDir(), "node_modules", "multiai", "bin", "native", binaryName)
	executablePath = func() (string, error) { return currentExe, nil }
	lookPath = func(string) (string, error) { return filepath.Join(t.TempDir(), "npm.exe"), nil }

	var calls int
	runCommand = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		calls++
		return []byte(filepath.Join(t.TempDir(), "node_modules") + "\n"), nil
	}

	result, err := InstallRelease(context.Background(), &Release{TagName: "v1.2.3"})
	if result != nil || err == nil || !strings.Contains(err.Error(), "possible PATH hijack") {
		t.Fatalf("result=%+v err=%v, want PATH hijack refusal", result, err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want ownership check only", calls)
	}
}

func TestInstallReleaseRefusesUnknownInstallationWithoutRunningAnything(t *testing.T) {
	restoreHooks(t)
	executablePath = func() (string, error) { return filepath.Join(t.TempDir(), "multiai.exe"), nil }
	called := false
	runCommand = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		called = true
		return nil, nil
	}

	result, err := InstallRelease(context.Background(), &Release{TagName: "v1.2.3"})
	if result != nil || !errors.Is(err, ErrUnsupportedInstall) {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if called {
		t.Fatal("a command was run for an unsupported installation")
	}
}

func TestInstallReleaseFailsClosedOnManagerOrVerificationError(t *testing.T) {
	tests := []struct {
		name       string
		managerErr error
		verifyOut  string
		verifyErr  error
	}{
		{"manager error", errors.New("npm failed"), "", nil},
		{"wrong version", nil, "multiai 1.2.2", nil},
		{"invalid output", nil, "something else", nil},
		{"verification error", nil, "", errors.New("cannot start")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			restoreHooks(t)
			binaryName := "multiai"
			if runtime.GOOS == "windows" {
				binaryName += ".exe"
			}
			currentExe := filepath.Join(t.TempDir(), "node_modules", "multiai", "bin", "native", binaryName)
			executablePath = func() (string, error) { return currentExe, nil }
			lookPath = func(string) (string, error) { return filepath.Join(t.TempDir(), "npm.exe"), nil }
			call := 0
			runCommand = func(ctx context.Context, name string, args ...string) ([]byte, error) {
				call++
				if call == 1 {
					return []byte(filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(currentExe)))) + "\n"), nil
				}
				if call == 2 {
					return []byte("manager output"), test.managerErr
				}
				return []byte(test.verifyOut), test.verifyErr
			}

			result, err := InstallRelease(context.Background(), &Release{TagName: "v1.2.3"})
			if result != nil || err == nil {
				t.Fatalf("result=%+v err=%v, want fail-closed", result, err)
			}
		})
	}
}

func TestInstallReleaseRejectsInvalidReleaseBeforeExecuting(t *testing.T) {
	restoreHooks(t)
	called := false
	runCommand = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		called = true
		return nil, nil
	}
	if _, err := InstallRelease(context.Background(), &Release{TagName: "latest;evil"}); err == nil {
		t.Fatal("invalid release unexpectedly accepted")
	}
	if called {
		t.Fatal("command executed for invalid release")
	}
}

func TestResolveNPMCommandAvoidsWindowsBatchExecution(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific npm shim contract")
	}
	restoreHooks(t)
	dir := t.TempDir()
	npmPath := filepath.Join(dir, "npm.cmd")
	nodePath := filepath.Join(dir, "node.exe")
	cliPath := filepath.Join(dir, "node_modules", "npm", "bin", "npm-cli.js")
	if err := os.MkdirAll(filepath.Dir(cliPath), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{npmPath, nodePath, cliPath} {
		if err := os.WriteFile(path, []byte("test"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	lookPath = func(string) (string, error) { return npmPath, nil }

	command, args, err := resolveNPMCommand()
	if err != nil {
		t.Fatalf("resolveNPMCommand(): %v", err)
	}
	if command != nodePath || len(args) != 1 || args[0] != cliPath {
		t.Fatalf("command=%q args=%q", command, args)
	}
}

func TestWriteCacheConcurrentSafe(t *testing.T) {
	t.Setenv("MULTIAI_CACHE_DIR", t.TempDir())
	done := make(chan struct{})
	for i := 0; i < 5; i++ {
		go func(version int) {
			_ = WriteCache(Cache{LastCheck: time.Now(), LatestVersion: fmt.Sprintf("1.%d.0", version)})
			done <- struct{}{}
		}(i)
	}
	for i := 0; i < 5; i++ {
		<-done
	}
	if _, err := ReadCache(); err != nil {
		t.Fatalf("ReadCache() after concurrent writes: %v", err)
	}
}

func TestExportedContractsCompile(t *testing.T) {
	_ = (func(context.Context, string))(Check)
	_ = (func(context.Context, string) (CheckResult, error))(CheckLatest)
	_ = (func(context.Context) (*Release, error))(FetchLatestRelease)
	_ = (func(context.Context, *Release) (*InstallResult, error))(InstallRelease)
	_ = (func(string, string) bool)(IsNewer)
	_ = (func() bool)(ShouldCheck)
	_ = (func() (*Cache, error))(ReadCache)
	_ = (func(Cache) error)(WriteCache)
}

func BenchmarkIsNewer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		IsNewer("0.6.7", "1.0.0")
	}
}

func BenchmarkCacheJSON(b *testing.B) {
	cache := Cache{LastCheck: time.Now(), LatestVersion: "1.2.3"}
	for i := 0; i < b.N; i++ {
		data, _ := json.Marshal(cache)
		var decoded Cache
		_ = json.Unmarshal(data, &decoded)
	}
}
