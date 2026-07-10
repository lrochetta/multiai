package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lrochetta/multiai/internal/fsutil"
)

const cacheTTL = 1 * time.Hour

// cacheFileName is the file name used for the local registry index cache.
const cacheFileName = "registry-index.json"

// readCache reads the registry index from the local cache. It returns (nil,
// nil) when the cache is absent or expired — callers treat that as "needs
// refresh", not an error.
func readCache() (*Index, error) {
	path := cacheFilePath()
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil // cache miss — not an error
	}
	cacheEntry := struct {
		FetchedAt time.Time `json:"fetched_at"`
		Index     *Index    `json:"index"`
	}{}
	if err := json.Unmarshal(data, &cacheEntry); err != nil {
		return nil, nil // corrupted cache — treat as miss
	}
	if time.Since(cacheEntry.FetchedAt) > cacheTTL {
		return nil, nil // expired
	}
	return cacheEntry.Index, nil
}

// writeCache atomically writes the registry index to the local cache.
func writeCache(idx *Index) error {
	path := cacheFilePath()
	if path == "" {
		return fmt.Errorf("cannot resolve cache path")
	}
	cacheEntry := struct {
		FetchedAt time.Time `json:"fetched_at"`
		Index     *Index    `json:"index"`
	}{
		FetchedAt: time.Now(),
		Index:     idx,
	}
	data, err := json.Marshal(cacheEntry)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(path, data, 0644)
}

// cacheFilePath returns the path to the registry index cache file. Cache
// directory is determined by the MULTIAI_CACHE_DIR env var, falling back to
// os.UserConfigDir()/multiai/cache.
func cacheFilePath() string {
	dir := cacheDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, cacheFileName)
}

// cacheDir resolves the base directory for cache files.
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
