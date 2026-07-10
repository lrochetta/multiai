package openrouter

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSaveAndLoadCache(t *testing.T) {
	t.Setenv("MULTIAI_CACHE_DIR", t.TempDir())
	models := loadFixture(t)

	if err := SaveCache(models); err != nil {
		t.Fatalf("SaveCache: %v", err)
	}
	loaded, fetchedAt, err := LoadCache()
	if err != nil {
		t.Fatalf("LoadCache: %v", err)
	}
	if len(loaded) != len(models) {
		t.Errorf("loaded %d models, want %d", len(loaded), len(models))
	}
	if time.Since(fetchedAt) > time.Minute {
		t.Errorf("fetchedAt too old: %v", fetchedAt)
	}
}

func TestLoadCacheMissing(t *testing.T) {
	t.Setenv("MULTIAI_CACHE_DIR", t.TempDir())
	if _, _, err := LoadCache(); err == nil {
		t.Fatal("want error on missing cache")
	}
}

func TestLoadCacheCorrupt(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_CACHE_DIR", dir)
	if err := os.WriteFile(filepath.Join(dir, cacheFileName), []byte("{oops"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadCache(); err == nil || !strings.Contains(err.Error(), "illisible") {
		t.Fatalf("want unreadable-cache error, got %v", err)
	}
}

func TestLoadCacheEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_CACHE_DIR", dir)
	if err := os.WriteFile(filepath.Join(dir, cacheFileName), []byte(`{"models":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadCache(); err == nil || !strings.Contains(err.Error(), "vide") {
		t.Fatalf("want empty-cache error, got %v", err)
	}
}

// writeCacheAged writes a cache file whose fetched_at is `age` in the past.
func writeCacheAged(t *testing.T, dir string, models []ModelInfo, age time.Duration) {
	t.Helper()
	data, err := json.Marshal(cachedCatalog{FetchedAt: time.Now().UTC().Add(-age), Models: models})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, cacheFileName), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGetModelsOfflineNoCacheFallsBackToEmbedded(t *testing.T) {
	t.Setenv("MULTIAI_CACHE_DIR", t.TempDir())

	cat := GetModels(context.Background(), true)
	if cat.Source != SourceEmbedded {
		t.Fatalf("source = %s, want %s", cat.Source, SourceEmbedded)
	}
	if len(cat.Models) != len(fallbackModels) {
		t.Errorf("got %d embedded models, want %d", len(cat.Models), len(fallbackModels))
	}
	if cat.Warning == "" {
		t.Error("want a degradation warning for the embedded list")
	}
}

func TestGetModelsOfflineFreshCache(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_CACHE_DIR", dir)
	writeCacheAged(t, dir, loadFixture(t), 5*time.Minute)

	cat := GetModels(context.Background(), true)
	if cat.Source != SourceCache {
		t.Fatalf("source = %s, want %s", cat.Source, SourceCache)
	}
	if cat.Warning != "" {
		t.Errorf("fresh cache should have no warning, got %q", cat.Warning)
	}
}

func TestGetModelsOfflineStaleCache(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_CACHE_DIR", dir)
	writeCacheAged(t, dir, loadFixture(t), 3*time.Hour)

	cat := GetModels(context.Background(), true)
	if cat.Source != SourceStale {
		t.Fatalf("source = %s, want %s", cat.Source, SourceStale)
	}
	if !strings.Contains(cat.Warning, "perime") {
		t.Errorf("warning should mention staleness, got %q", cat.Warning)
	}
}

func TestGetModelsOnlineFreshCacheSkipsNetwork(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_CACHE_DIR", dir)
	writeCacheAged(t, dir, loadFixture(t), time.Minute)
	setAPIBase(t, deadServer(t)) // any network call would fail loudly

	cat := GetModels(context.Background(), false)
	if cat.Source != SourceCache {
		t.Fatalf("source = %s, want %s (no network expected)", cat.Source, SourceCache)
	}
}

func TestGetModelsOnlineFetchesAndWritesCache(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_CACHE_DIR", dir)
	srv := newFixtureServer(t, nil)
	setAPIBase(t, srv.URL)

	cat := GetModels(context.Background(), false)
	if cat.Source != SourceNetwork {
		t.Fatalf("source = %s, want %s", cat.Source, SourceNetwork)
	}
	if len(cat.Models) != 8 {
		t.Errorf("got %d models, want 8", len(cat.Models))
	}
	if _, err := os.Stat(filepath.Join(dir, cacheFileName)); err != nil {
		t.Errorf("cache file should have been written: %v", err)
	}
}

func TestGetModelsOnlineNetworkDownUsesStaleCache(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("MULTIAI_CACHE_DIR", dir)
	writeCacheAged(t, dir, loadFixture(t), 3*time.Hour)
	setAPIBase(t, deadServer(t))

	cat := GetModels(context.Background(), false)
	if cat.Source != SourceStale {
		t.Fatalf("source = %s, want %s", cat.Source, SourceStale)
	}
	if !strings.Contains(cat.Warning, "cache local du") {
		t.Errorf("warning should mention the cache fallback, got %q", cat.Warning)
	}
}

func TestGetModelsOnlineNetworkDownNoCacheUsesEmbedded(t *testing.T) {
	t.Setenv("MULTIAI_CACHE_DIR", t.TempDir())
	setAPIBase(t, deadServer(t))

	cat := GetModels(context.Background(), false)
	if cat.Source != SourceEmbedded {
		t.Fatalf("source = %s, want %s", cat.Source, SourceEmbedded)
	}
	if !strings.Contains(cat.Warning, "embarquee") {
		t.Errorf("warning should mention the embedded list, got %q", cat.Warning)
	}
	// The embedded list must include the two free models of the PS reference.
	free := FilterFree(cat.Models)
	if len(free) != 2 {
		t.Errorf("embedded free models = %d, want 2", len(free))
	}
}

func TestEmbeddedModelsReturnsACopy(t *testing.T) {
	a := embeddedModels()
	a[0].ID = "mutated/mutated"
	b := embeddedModels()
	if b[0].ID == "mutated/mutated" {
		t.Fatal("embeddedModels must return a copy, not the shared slice")
	}
}
