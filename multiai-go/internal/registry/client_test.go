package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// TestFetchIndex
// ---------------------------------------------------------------------------

func TestFetchIndexFromServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
			"version": 1,
			"updated_at": "2026-07-10T12:00:00Z",
			"total": 2,
			"profiles": [
				{
					"name": "ds",
					"title": "DeepSeek V4 Pro",
					"description": "Configuration for DeepSeek V4 Pro",
					"author": "lrochetta",
					"stars": 5,
					"tags": ["deepseek", "claude"]
				},
				{
					"name": "codex55",
					"title": "Codex CLI 55",
					"description": "Codex CLI with GPT-5.5",
					"author": "community",
					"stars": 3
				}
			]
		}`)
	}))
	defer srv.Close()

	// Temporarily override the index URL to point at our test server.
	origURL := indexURL
	indexURL = srv.URL
	defer func() { indexURL = origURL }()

	ctx := context.Background()
	idx, err := FetchIndexNoCache(ctx)
	if err != nil {
		t.Fatalf("FetchIndex() unexpected error: %v", err)
	}

	if idx.Version != 1 {
		t.Errorf("Version = %d, want 1", idx.Version)
	}
	if idx.Total != 2 {
		t.Errorf("Total = %d, want 2", idx.Total)
	}
	if len(idx.Profiles) != 2 {
		t.Fatalf("len(Profiles) = %d, want 2", len(idx.Profiles))
	}
	if idx.Profiles[0].Name != "ds" {
		t.Errorf("Profile[0].Name = %q, want %q", idx.Profiles[0].Name, "ds")
	}
	if idx.Profiles[0].Stars != 5 {
		t.Errorf("Profile[0].Stars = %d, want 5", idx.Profiles[0].Stars)
	}
	if len(idx.Profiles[0].Tags) != 2 {
		t.Errorf("Profile[0].Tags len = %d, want 2", len(idx.Profiles[0].Tags))
	}
}

func TestFetchIndexHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"message": "rate limited"}`)
	}))
	defer srv.Close()

	origURL := indexURL
	indexURL = srv.URL
	defer func() { indexURL = origURL }()

	ctx := context.Background()
	_, err := FetchIndexNoCache(ctx)
	if err == nil {
		t.Fatal("expected error for HTTP 403")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("error should mention HTTP status, got: %v", err)
	}
}

func TestFetchIndexTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"version":1,"profiles":[]}`)
	}))
	defer srv.Close()

	origTimeout := httpTimeout
	httpTimeout = 100 * time.Millisecond
	defer func() { httpTimeout = origTimeout }()

	origURL := indexURL
	indexURL = srv.URL
	defer func() { indexURL = origURL }()

	ctx := context.Background()
	_, err := FetchIndexNoCache(ctx)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestFetchIndexInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{invalid json`)
	}))
	defer srv.Close()

	origURL := indexURL
	indexURL = srv.URL
	defer func() { indexURL = origURL }()

	ctx := context.Background()
	_, err := FetchIndexNoCache(ctx)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestFetchIndexEmptyProfiles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"version":1,"total":0}`)
	}))
	defer srv.Close()

	origURL := indexURL
	indexURL = srv.URL
	defer func() { indexURL = origURL }()

	ctx := context.Background()
	idx, err := FetchIndexNoCache(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx.Profiles == nil {
		t.Fatal("Profiles should be non-nil (empty slice)")
	}
	if len(idx.Profiles) != 0 {
		t.Errorf("len(Profiles) = %d, want 0", len(idx.Profiles))
	}
}

// ---------------------------------------------------------------------------
// TestSearchProfiles
// ---------------------------------------------------------------------------

func TestSearchProfiles(t *testing.T) {
	idx := &Index{
		Profiles: []ProfileEntry{
			{Name: "ds", Title: "DeepSeek V4 Pro", Description: "Claude Code profile for DeepSeek", Author: "lrochetta", Stars: 5, Tags: []string{"deepseek", "claude"}},
			{Name: "codex55", Title: "Codex CLI 55", Description: "GPT-5.5 for Codex CLI", Author: "community", Stars: 3, Tags: []string{"openai", "gpt"}},
			{Name: "oc-qwen", Title: "OpenCode Qwen", Description: "Qwen model via OpenCode", Author: "contributor", Stars: 2},
		},
	}

	tests := []struct {
		name  string
		query string
		want  int
		first string
	}{
		{"match name", "ds", 1, "ds"},
		{"match title", "deepseek", 1, "ds"},
		{"match description", "codex", 1, "codex55"},
		{"match author", "lrochetta", 1, "ds"},
		{"match tag", "openai", 1, "codex55"},
		{"match multiple", "claude", 1, "ds"},
		{"match all", "o", 3, ""},
		{"no match", "nonexistent", 0, ""},
		{"empty query", "", 0, ""},
		{"case insensitive", "DEEPSEEK", 1, "ds"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := SearchProfiles(idx, tt.query)
			if len(results) != tt.want {
				t.Errorf("SearchProfiles(%q) = %d results, want %d", tt.query, len(results), tt.want)
			}
			if tt.first != "" && len(results) > 0 && results[0].Name != tt.first {
				t.Errorf("first result.Name = %q, want %q", results[0].Name, tt.first)
			}
		})
	}
}

func TestSearchProfilesNilIndex(t *testing.T) {
	results := SearchProfiles(nil, "test")
	if results != nil {
		t.Errorf("expected nil for nil index, got %v", results)
	}
}

// ---------------------------------------------------------------------------
// TestFindProfileByName
// ---------------------------------------------------------------------------

func TestFindProfileByName(t *testing.T) {
	idx := &Index{
		Profiles: []ProfileEntry{
			{Name: "ds", Title: "DeepSeek V4 Pro", Author: "lrochetta"},
			{Name: "codex55", Title: "Codex CLI 55", Author: "community"},
		},
	}

	tests := []struct {
		name  string
		query string
		found bool
	}{
		{"exact match", "ds", true},
		{"case insensitive", "DS", true},
		{"not found", "missing", false},
		{"empty query", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := FindProfileByName(idx, tt.query)
			if tt.found && p == nil {
				t.Errorf("FindProfileByName(%q) = nil, want profile", tt.query)
			}
			if !tt.found && p != nil {
				t.Errorf("FindProfileByName(%q) = %v, want nil", tt.query, p)
			}
		})
	}
}

func TestFindProfileByNameNilIndex(t *testing.T) {
	p := FindProfileByName(nil, "ds")
	if p != nil {
		t.Errorf("expected nil for nil index, got %v", p)
	}
}

// ---------------------------------------------------------------------------
// TestCacheReadWrite
// ---------------------------------------------------------------------------

func TestCacheReadWrite(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_CACHE_DIR", dir)

	idx := &Index{
		Version: 1,
		Total:   1,
		Profiles: []ProfileEntry{
			{Name: "ds", Title: "DeepSeek", Author: "lrochetta", Stars: 5},
		},
	}

	if err := writeCache(idx); err != nil {
		t.Fatalf("writeCache() error: %v", err)
	}

	cached, err := readCache()
	if err != nil {
		t.Fatalf("readCache() error: %v", err)
	}
	if cached == nil {
		t.Fatal("readCache() returned nil")
	}
	if cached.Version != 1 {
		t.Errorf("Version = %d, want 1", cached.Version)
	}
	if len(cached.Profiles) != 1 {
		t.Errorf("len(Profiles) = %d, want 1", len(cached.Profiles))
	}
	if cached.Profiles[0].Name != "ds" {
		t.Errorf("Profile.Name = %q, want %q", cached.Profiles[0].Name, "ds")
	}
}

func TestCacheExpiry(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_CACHE_DIR", dir)

	idx := &Index{Version: 1, Profiles: []ProfileEntry{}}
	if err := writeCache(idx); err != nil {
		t.Fatalf("writeCache() error: %v", err)
	}

	// Cache should be valid immediately.
	cached, err := readCache()
	if err != nil {
		t.Fatalf("readCache() error: %v", err)
	}
	if cached == nil {
		t.Fatal("readCache() returned nil for fresh cache")
	}

	// Modify the cache file to have an old timestamp.
	path := cacheFilePath()
	entry := struct {
		FetchedAt time.Time `json:"fetched_at"`
		Index     *Index    `json:"index"`
	}{
		FetchedAt: time.Now().Add(-2 * cacheTTL),
		Index:     idx,
	}
	data, _ := json.Marshal(entry)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Cache should now be expired.
	cached, err = readCache()
	if err != nil {
		t.Fatalf("readCache() error: %v", err)
	}
	if cached != nil {
		t.Fatal("readCache() should return nil for expired cache")
	}
}

func TestCacheCorruptedFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_CACHE_DIR", dir)

	// Write garbage.
	path := filepath.Join(dir, "registry-index.json")
	if err := os.WriteFile(path, []byte("{bad json"), 0644); err != nil {
		t.Fatal(err)
	}

	cached, err := readCache()
	if err != nil {
		t.Fatalf("readCache() error: %v", err)
	}
	if cached != nil {
		t.Fatal("readCache() should return nil for corrupted cache")
	}
}

func TestCacheNoDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_CACHE_DIR", dir)

	// No cache file exists yet.
	cached, err := readCache()
	if err != nil {
		t.Fatalf("readCache() error: %v", err)
	}
	if cached != nil {
		t.Fatal("readCache() should return nil when no cache exists")
	}
}

// ---------------------------------------------------------------------------
// TestFetchIndexWithCache
// ---------------------------------------------------------------------------

func TestFetchIndexWithCache(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_CACHE_DIR", dir)

	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"version":1,"total":1,"profiles":[{"name":"ds","title":"DeepSeek","author":"lrochetta"}]}`)
	}))
	defer srv.Close()

	origURL := indexURL
	indexURL = srv.URL
	defer func() { indexURL = origURL }()

	ctx := context.Background()

	// First call: should hit the server.
	idx1, err := FetchIndex(ctx)
	if err != nil {
		t.Fatalf("first FetchIndex() error: %v", err)
	}
	if idx1 == nil {
		t.Fatal("first FetchIndex() returned nil")
	}
	if requestCount != 1 {
		t.Errorf("first call: requestCount = %d, want 1", requestCount)
	}

	// Second call: should use cache, no server request.
	idx2, err := FetchIndex(ctx)
	if err != nil {
		t.Fatalf("second FetchIndex() error: %v", err)
	}
	if idx2 == nil {
		t.Fatal("second FetchIndex() returned nil")
	}
	if requestCount != 1 {
		t.Errorf("second call (cached): requestCount = %d, want 1", requestCount)
	}

	// Both should return the same data.
	if idx1.Version != idx2.Version {
		t.Errorf("version mismatch: %d vs %d", idx1.Version, idx2.Version)
	}
	if len(idx1.Profiles) != len(idx2.Profiles) {
		t.Errorf("profile count mismatch: %d vs %d", len(idx1.Profiles), len(idx2.Profiles))
	}
}

// ---------------------------------------------------------------------------
// TestResolveURL
// ---------------------------------------------------------------------------

func TestResolveURL(t *testing.T) {
	t.Run("default URL", func(t *testing.T) {
		orig := indexURL
		indexURL = defaultIndexURL
		defer func() { indexURL = orig }()

		url := resolveURL()
		if url != defaultIndexURL {
			t.Errorf("resolveURL() = %q, want %q", url, defaultIndexURL)
		}
	})

	t.Run("custom URL requires DEV", func(t *testing.T) {
		t.Setenv("MULTIAI_DEV", "")
		t.Setenv("MULTIAI_REGISTRY_URL", "https://example.com/index.json")

		orig := indexURL
		indexURL = defaultIndexURL
		defer func() { indexURL = orig }()

		url := resolveURL()
		if url != defaultIndexURL {
			t.Errorf("resolveURL() = %q, want default (no MULTIAI_DEV)", url)
		}
	})

	t.Run("custom URL with DEV", func(t *testing.T) {
		t.Setenv("MULTIAI_DEV", "1")
		t.Setenv("MULTIAI_REGISTRY_URL", "https://example.com/index.json")

		orig := indexURL
		indexURL = defaultIndexURL
		defer func() { indexURL = orig }()

		url := resolveURL()
		if url != "https://example.com/index.json" {
			t.Errorf("resolveURL() = %q, want custom URL", url)
		}
	})
}

// ---------------------------------------------------------------------------
// Test for FindProfileByName edge cases
// ---------------------------------------------------------------------------

func TestProfileEntryFields(t *testing.T) {
	idx := &Index{
		Profiles: []ProfileEntry{
			{Name: "test", Title: "Test Profile", Description: "A test", Author: "tester", Stars: 42, Tags: []string{"a", "b"}},
		},
	}

	p := FindProfileByName(idx, "test")
	if p == nil {
		t.Fatal("FindProfileByName returned nil")
	}
	if p.Title != "Test Profile" {
		t.Errorf("Title = %q, want %q", p.Title, "Test Profile")
	}
	if p.Description != "A test" {
		t.Errorf("Description = %q, want %q", p.Description, "A test")
	}
	if p.Author != "tester" {
		t.Errorf("Author = %q, want %q", p.Author, "tester")
	}
	if p.Stars != 42 {
		t.Errorf("Stars = %d, want 42", p.Stars)
	}
	if len(p.Tags) != 2 {
		t.Errorf("len(Tags) = %d, want 2", len(p.Tags))
	}
}
