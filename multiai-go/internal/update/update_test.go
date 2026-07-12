package update

import (
	"context"
	"crypto/sha256"
	"encoding/json"
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

// ---------------------------------------------------------------------------
// TestIsNewer
// ---------------------------------------------------------------------------

func TestIsNewer(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    bool
	}{
		{"equal versions", "0.4.0", "0.4.0", false},
		{"v prefix on both", "v0.4.1", "v0.4.0", false},
		{"newer patch", "0.4.0", "0.4.1", true},
		{"newer minor", "0.4.0", "0.5.0", true},
		{"newer major", "0.4.0", "1.0.0", true},
		{"older patch", "0.4.1", "0.4.0", false},
		{"mixed v/no-v", "v0.4.0", "0.5.0", true},
		{"invalid versions", "abc", "1.0", false},
		{"empty current", "", "0.4.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNewer(tt.current, tt.latest)
			if got != tt.want {
				t.Errorf("IsNewer(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestGetTarget
// ---------------------------------------------------------------------------

func TestGetTarget(t *testing.T) {
	got := GetTarget()

	parts := strings.SplitN(got, "_", 2)
	if len(parts) != 2 {
		t.Fatalf("GetTarget() = %q, want format os_arch", got)
	}
	if parts[0] == "" || parts[1] == "" {
		t.Fatalf("GetTarget() = %q, both OS and arch must be non-empty", got)
	}
}

func TestGetTargetMatchesRuntime(t *testing.T) {
	want := runtime.GOOS + "_" + runtime.GOARCH
	got := GetTarget()
	if got != want {
		t.Errorf("GetTarget() = %q, want %q (matching runtime.GOOS/_GOARCH)", got, want)
	}
}

// ---------------------------------------------------------------------------
// TestReadWriteCache
// ---------------------------------------------------------------------------

func TestReadWriteCache(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_CACHE_DIR", dir)

	c := Cache{
		LastCheck:     time.Now().Truncate(time.Second),
		LatestVersion: "1.2.3",
	}

	if err := WriteCache(c); err != nil {
		t.Fatalf("WriteCache() unexpected error: %v", err)
	}

	got, err := ReadCache()
	if err != nil {
		t.Fatalf("ReadCache() unexpected error: %v", err)
	}

	if !got.LastCheck.Equal(c.LastCheck) {
		t.Errorf("ReadCache().LastCheck = %v, want %v", got.LastCheck, c.LastCheck)
	}
	if got.LatestVersion != c.LatestVersion {
		t.Errorf("ReadCache().LatestVersion = %q, want %q", got.LatestVersion, c.LatestVersion)
	}
}

func TestReadCacheMissingFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_CACHE_DIR", dir)

	_, err := ReadCache()
	if err == nil {
		t.Fatal("ReadCache() expected error for missing cache file")
	}
}

func TestReadCacheCorruptedFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_CACHE_DIR", dir)

	// Ensure the directory matches what CacheFilePath would return.
	cacheDir := filepath.Join(dir, "multiai")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write garbage.
	if err := os.WriteFile(filepath.Join(cacheDir, "update-check.json"), []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ReadCache()
	if err == nil {
		t.Fatal("ReadCache() expected error for corrupted cache file")
	}
}

// ---------------------------------------------------------------------------
// TestCacheFilePath
// ---------------------------------------------------------------------------

func TestCacheFilePath(t *testing.T) {
	path := CacheFilePath()
	if !strings.HasSuffix(path, "update-check.json") {
		t.Errorf("CacheFilePath() = %q, want path ending with update-check.json", path)
	}
}

func TestCacheFilePathUsesEnvVar(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_CACHE_DIR", dir)

	path := CacheFilePath()
	if !strings.HasPrefix(path, dir) {
		t.Errorf("CacheFilePath() = %q, want prefix %q", path, dir)
	}
	if !strings.HasSuffix(path, "update-check.json") {
		t.Errorf("CacheFilePath() = %q, want suffix update-check.json", path)
	}
}

func TestCacheFilePathDefaultDir(t *testing.T) {
	// Unset env var so the default user cache dir is used.
	t.Setenv("MULTIAI_CACHE_DIR", "")

	path := CacheFilePath()
	if path == "" {
		t.Fatal("CacheFilePath() returned empty string")
	}
	if !strings.HasSuffix(path, "update-check.json") {
		t.Errorf("CacheFilePath() = %q, want suffix update-check.json", path)
	}
}

// ---------------------------------------------------------------------------
// TestShouldCheck
// ---------------------------------------------------------------------------

func TestShouldCheck(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T) // prepares the cache, returns nothing
		want  bool
	}{
		{
			name: "no cache file",
			setup: func(t *testing.T) {
				t.Setenv("MULTIAI_CACHE_DIR", t.TempDir())
			},
			want: true,
		},
		{
			name: "cache older than one hour",
			setup: func(t *testing.T) {
				dir := t.TempDir()
				t.Setenv("MULTIAI_CACHE_DIR", dir)
				c := Cache{
					LastCheck:     time.Now().Add(-2 * time.Hour),
					LatestVersion: "0.1.0",
				}
				if err := WriteCache(c); err != nil {
					t.Fatal(err)
				}
			},
			want: true,
		},
		{
			name: "cache newer than one hour",
			setup: func(t *testing.T) {
				dir := t.TempDir()
				t.Setenv("MULTIAI_CACHE_DIR", dir)
				c := Cache{
					LastCheck:     time.Now().Add(-30 * time.Minute),
					LatestVersion: "0.1.0",
				}
				if err := WriteCache(c); err != nil {
					t.Fatal(err)
				}
			},
			want: false,
		},
		{
			name: "cache just under one hour",
			setup: func(t *testing.T) {
				dir := t.TempDir()
				t.Setenv("MULTIAI_CACHE_DIR", dir)
				c := Cache{
					LastCheck:     time.Now().Add(-59 * time.Minute),
					LatestVersion: "0.1.0",
				}
				if err := WriteCache(c); err != nil {
					t.Fatal(err)
				}
			},
			want: false,
		},
		{
			name: "cache exactly one hour",
			setup: func(t *testing.T) {
				dir := t.TempDir()
				t.Setenv("MULTIAI_CACHE_DIR", dir)
				c := Cache{
					LastCheck:     time.Now().Add(-1 * time.Hour),
					LatestVersion: "0.1.0",
				}
				if err := WriteCache(c); err != nil {
					t.Fatal(err)
				}
			},
			want: false, // exactly 1h is still within the window
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(t)
			got := ShouldCheck()
			if got != tt.want {
				t.Errorf("ShouldCheck() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestFetchLatestRelease
// ---------------------------------------------------------------------------

func TestFetchLatestRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
			"tag_name": "v1.2.3",
			"assets": [
				{"name": "multiai_windows_amd64.zip", "browser_download_url": "https://example.com/multiai_windows_amd64.zip"},
				{"name": "multiai_linux_amd64.tar.gz", "browser_download_url": "https://example.com/multiai_linux_amd64.tar.gz"}
			]
		}`)
	}))
	defer srv.Close()

	release, err := fetchLatestReleaseWithURL(srv.URL + "/repos/lrochetta/multiai/releases/latest")
	if err != nil {
		t.Fatalf("fetchLatestRelease() unexpected error: %v", err)
	}

	if release.TagName != "v1.2.3" {
		t.Errorf("TagName = %q, want %q", release.TagName, "v1.2.3")
	}
	if len(release.Assets) != 2 {
		t.Fatalf("len(Assets) = %d, want 2", len(release.Assets))
	}
	if release.Assets[0].Name != "multiai_windows_amd64.zip" {
		t.Errorf("Asset[0].Name = %q, want %q", release.Assets[0].Name, "multiai_windows_amd64.zip")
	}
	if release.Assets[0].BrowserDownloadURL != "https://example.com/multiai_windows_amd64.zip" {
		t.Errorf("Asset[0].BrowserDownloadURL = %q, want %q", release.Assets[0].BrowserDownloadURL, "https://example.com/multiai_windows_amd64.zip")
	}
}

func TestFetchLatestReleaseWithTargetAsset(t *testing.T) {
	target := GetTarget()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{
			"tag_name": "v1.2.3",
			"assets": [
				{"name": "multiai_%s.zip", "browser_download_url": "https://example.com/multiai_%s.zip"},
				{"name": "multiai_linux_arm64.tar.gz", "browser_download_url": "https://example.com/multiai_linux_arm64.tar.gz"}
			]
		}`, target, target)
	}))
	defer srv.Close()

	release, err := fetchLatestReleaseWithURL(srv.URL + "/repos/lrochetta/multiai/releases/latest")
	if err != nil {
		t.Fatalf("fetchLatestRelease() unexpected error: %v", err)
	}

	if len(release.Assets) != 2 {
		t.Fatalf("len(Assets) = %d, want 2", len(release.Assets))
	}
}

func TestFetchLatestReleaseInvalidResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"this is not the expected schema`)
	}))
	defer srv.Close()

	_, err := fetchLatestReleaseWithURL(srv.URL + "/repos/lrochetta/multiai/releases/latest")
	if err == nil {
		t.Fatal("fetchLatestRelease() expected error for invalid JSON response")
	}
}

func TestFetchLatestReleaseHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"message": "API rate limit exceeded"}`)
	}))
	defer srv.Close()

	_, err := fetchLatestReleaseWithURL(srv.URL + "/repos/lrochetta/multiai/releases/latest")
	if err == nil {
		t.Fatal("fetchLatestRelease() expected error for HTTP 403")
	}
}

func TestFetchLatestReleaseTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sleep longer than the client timeout.
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"tag_name":"v1.0.0","assets":[]}`)
	}))
	defer srv.Close()

	t.Setenv("MULTIAI_UPDATE_TIMEOUT", "100ms")

	_, err := fetchLatestReleaseWithURL(srv.URL + "/repos/lrochetta/multiai/releases/latest")
	if err == nil {
		t.Fatal("fetchLatestRelease() expected timeout error")
	}
}

// ---------------------------------------------------------------------------
// TestDownloadAndVerify
// ---------------------------------------------------------------------------

func TestDownloadAndVerify(t *testing.T) {
	payload := []byte("this is a fake binary content for testing")
	hash := sha256.Sum256(payload)
	checksum := fmt.Sprintf("%x", hash)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(payload) //nolint:errcheck
	}))
	defer srv.Close()

	ctx := context.Background()
	dest := filepath.Join(t.TempDir(), "downloaded.zip")
	err := DownloadAndVerify(ctx, srv.URL, checksum, dest)
	if err != nil {
		t.Fatalf("DownloadAndVerify() unexpected error: %v", err)
	}

	// Verify the file exists and has the correct content.
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(payload) {
		t.Errorf("downloaded content = %q, want %q", string(data), string(payload))
	}
}

func TestDownloadAndVerifyWrongChecksum(t *testing.T) {
	payload := []byte("real content")
	wrongChecksum := fmt.Sprintf("%x", sha256.Sum256([]byte("wrong content")))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(payload) //nolint:errcheck
	}))
	defer srv.Close()

	ctx := context.Background()
	dest := filepath.Join(t.TempDir(), "downloaded.zip")
	err := DownloadAndVerify(ctx, srv.URL, wrongChecksum, dest)
	if err == nil {
		t.Fatal("DownloadAndVerify() expected error for checksum mismatch")
	}
}

func TestDownloadAndVerifyServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ctx := context.Background()
	dest := filepath.Join(t.TempDir(), "downloaded.zip")
	err := DownloadAndVerify(ctx, srv.URL, "abc123", dest)
	if err == nil {
		t.Fatal("DownloadAndVerify() expected error for HTTP 500")
	}
}

func TestDownloadAndVerifyEmptyBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx := context.Background()
	dest := filepath.Join(t.TempDir(), "empty.zip")
	err := DownloadAndVerify(ctx, srv.URL, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", dest)
	if err != nil {
		t.Fatalf("DownloadAndVerify() unexpected error for empty body: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Test that FetchLatestRelease validates MULTIAI_GITHUB_API_URL
// ---------------------------------------------------------------------------

func TestFetchLatestReleaseURLBuilding(t *testing.T) {
	ctx := context.Background()

	t.Run("requires MULTIAI_DEV when MULTIAI_GITHUB_API_URL is set", func(t *testing.T) {
		t.Setenv("MULTIAI_GITHUB_API_URL", "https://api.github.com/repos/x/y/releases/latest")
		_, err := FetchLatestRelease(ctx)
		if err == nil {
			t.Fatal("expected error: MULTIAI_GITHUB_API_URL requires MULTIAI_DEV=1")
		}
	})

	t.Run("rejects non-GitHub URL even with MULTIAI_DEV", func(t *testing.T) {
		t.Setenv("MULTIAI_GITHUB_API_URL", "https://evil.com/repos/x/y/releases/latest")
		t.Setenv("MULTIAI_DEV", "1")
		_, err := FetchLatestRelease(ctx)
		if err == nil {
			t.Fatal("expected error: URL must start with https://api.github.com/")
		}
	})

	t.Run("default URL does not trigger validation", func(t *testing.T) {
		// When MULTIAI_GITHUB_API_URL is unset, the function uses the
		// hardcoded production URL and does not check MULTIAI_DEV.
		// We expect a network error (not a validation error).
		_, err := FetchLatestRelease(ctx)
		if err != nil && strings.Contains(err.Error(), "MULTIAI_GITHUB_API_URL") {
			t.Fatalf("unexpected validation error: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// Helper: fetchLatestReleaseWithURL wraps FetchLatestRelease with a
// configurable GitHub API base URL for testing.
// ---------------------------------------------------------------------------

func fetchLatestReleaseWithURL(apiURL string) (*Release, error) {
	return fetchReleaseFromURL(context.Background(), apiURL)
}

// ---------------------------------------------------------------------------
// Ensure exported symbols compile (interface check)
// ---------------------------------------------------------------------------

// TestExportedFunctions is a compile-time assertion that the expected public
// API of this package exists with the correct signatures.
func TestExportedFunctions(t *testing.T) {
	t.Run("IsNewer signature", func(t *testing.T) {
		var fn func(string, string) bool = IsNewer
		_ = fn
	})

	t.Run("GetTarget signature", func(t *testing.T) {
		var fn func() string = GetTarget
		_ = fn
	})

	t.Run("ReadCache signature", func(t *testing.T) {
		var fn func() (*Cache, error) = ReadCache
		_ = fn
	})

	t.Run("WriteCache signature", func(t *testing.T) {
		var fn func(Cache) error = WriteCache
		_ = fn
	})

	t.Run("CacheFilePath signature", func(t *testing.T) {
		var fn func() string = CacheFilePath
		_ = fn
	})

	t.Run("ShouldCheck signature", func(t *testing.T) {
		var fn func() bool = ShouldCheck
		_ = fn
	})

	t.Run("FetchLatestRelease signature", func(t *testing.T) {
		var fn func(context.Context) (*Release, error) = FetchLatestRelease
		_ = fn
	})

	t.Run("DownloadAndVerify signature", func(t *testing.T) {
		var fn func(context.Context, string, string, string) error = DownloadAndVerify
		_ = fn
	})
}

// ---------------------------------------------------------------------------
// Cache test helpers
// ---------------------------------------------------------------------------

// TestWriteCacheRoundTripJSON ensures the serialization/deserialization
// of Cache is stable.
func TestWriteCacheRoundTripJSON(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_CACHE_DIR", dir)

	original := Cache{
		LastCheck:     time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		LatestVersion: "0.5.0",
	}

	if err := WriteCache(original); err != nil {
		t.Fatalf("WriteCache() error: %v", err)
	}

	read, err := ReadCache()
	if err != nil {
		t.Fatalf("ReadCache() error: %v", err)
	}

	if !read.LastCheck.Equal(original.LastCheck) {
		t.Errorf("LastCheck: got %v, want %v", read.LastCheck, original.LastCheck)
	}
	if read.LatestVersion != original.LatestVersion {
		t.Errorf("LatestVersion: got %q, want %q", read.LatestVersion, original.LatestVersion)
	}
}

// ---------------------------------------------------------------------------
// Concurrency test for cache
// ---------------------------------------------------------------------------

func TestWriteCacheConcurrentSafe(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_CACHE_DIR", dir)

	done := make(chan struct{})
	for i := 0; i < 5; i++ {
		go func(v string) {
			_ = WriteCache(Cache{
				LastCheck:     time.Now(),
				LatestVersion: v,
			})
			done <- struct{}{}
		}(fmt.Sprintf("v1.%d.0", i))
	}
	for i := 0; i < 5; i++ {
		<-done
	}

	// Should not panic, and at least one write should have succeeded.
	_, err := ReadCache()
	if err != nil {
		t.Fatalf("ReadCache() after concurrent writes: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Helper: downloadReader is used in TestDownloadAndVerify variants
// ---------------------------------------------------------------------------

func TestDownloadAndVerifyCustomDir(t *testing.T) {
	payload := []byte("custom directory download test")
	hash := sha256.Sum256(payload)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(payload) //nolint:errcheck
	}))
	defer srv.Close()

	ctx := context.Background()
	destDir := t.TempDir()
	dest := filepath.Join(destDir, "output", "nested", "binary.exe")
	err := DownloadAndVerify(ctx, srv.URL, fmt.Sprintf("%x", hash), dest)
	if err != nil {
		t.Fatalf("DownloadAndVerify() with nested dir: %v", err)
	}
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		t.Fatal("DownloadAndVerify() did not create the destination file")
	}
}

// ---------------------------------------------------------------------------
// Edge cases for IsNewer
// ---------------------------------------------------------------------------

func TestIsNewerEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    bool
	}{
		{"both empty", "", "", false},
		{"only latest empty", "1.0.0", "", false},
		{"pre-release suffix", "1.0.0", "1.0.1-beta", true},
		{"pre-release current", "1.0.1-beta", "1.0.1", true},
		{"three-digit", "1.2.3", "1.2.4", true},
		{"long version", "10.20.30", "10.20.31", true},
		{"current higher", "2.0.0", "1.9.9", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNewer(tt.current, tt.latest)
			if got != tt.want {
				t.Errorf("IsNewer(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestFetchLatestReleasePartialData — e.g. missing assets field
// ---------------------------------------------------------------------------

func TestFetchLatestReleasePartialData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"tag_name": "v0.0.1"}`) // no "assets" key
	}))
	defer srv.Close()

	release, err := fetchLatestReleaseWithURL(srv.URL + "/repos/lrochetta/multiai/releases/latest")
	if err != nil {
		t.Fatalf("fetchLatestRelease() unexpected error: %v", err)
	}
	if release.TagName != "v0.0.1" {
		t.Errorf("TagName = %q, want %q", release.TagName, "v0.0.1")
	}
	if release.Assets == nil {
		t.Fatal("Assets should be non-nil (empty slice)")
	}
}

// ---------------------------------------------------------------------------
// TestFetchLatestReleaseWithHeader simulates rate-limit headers
// ---------------------------------------------------------------------------

func TestFetchLatestReleaseRateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(10*time.Minute).Unix()))
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"message":"API rate limit exceeded"}`)
	}))
	defer srv.Close()

	_, err := fetchLatestReleaseWithURL(srv.URL + "/repos/lrochetta/multiai/releases/latest")
	if err == nil {
		t.Fatal("fetchLatestRelease() expected error for rate limit")
	}
}

// ---------------------------------------------------------------------------
// Benchmark: IsNewer
// ---------------------------------------------------------------------------

func BenchmarkIsNewer(b *testing.B) {
	versions := []struct{ cur, lat string }{
		{"0.4.0", "0.4.0"},
		{"0.4.0", "0.4.1"},
		{"0.4.0", "0.5.0"},
		{"0.4.0", "1.0.0"},
		{"1.0.0", "0.9.9"},
		{"abc", "1.0"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := versions[i%len(versions)]
		IsNewer(v.cur, v.lat)
	}
}

// ---------------------------------------------------------------------------
// Benchmark: JSON marshal/unmarshal of Cache
// ---------------------------------------------------------------------------

func BenchmarkCacheRoundTrip(b *testing.B) {
	c := Cache{
		LastCheck:     time.Now(),
		LatestVersion: "1.2.3",
	}

	b.Run("marshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = json.Marshal(c)
		}
	})

	data, _ := json.Marshal(c)
	b.Run("unmarshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var out Cache
			_ = json.Unmarshal(data, &out)
		}
	})
}
