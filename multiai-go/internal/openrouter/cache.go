package openrouter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lrochetta/multiai/internal/fsutil"
)

// cacheFileName is the on-disk name of the cached OpenRouter /models payload.
const cacheFileName = "openrouter-models.json"

// nvidiaCacheFileName is the on-disk name of the cached NVIDIA /models payload.
const nvidiaCacheFileName = "nvidia-models.json"

// CacheDir resolves the local cache directory, in priority order:
//  1. MULTIAI_CACHE_DIR environment variable (tests, portable setups)
//  2. <user config dir>/multiai/cache
func CacheDir() (string, error) {
	if dir := os.Getenv("MULTIAI_CACHE_DIR"); dir != "" {
		return dir, nil
	}
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("dossier de configuration utilisateur introuvable: %w", err)
	}
	return filepath.Join(cfg, "multiai", "cache"), nil
}

// cachedCatalog is the JSON envelope written to the cache file.
type cachedCatalog struct {
	FetchedAt time.Time   `json:"fetched_at"`
	Models    []ModelInfo `json:"models"`
}

// LoadCache reads the cached OpenRouter model list and its fetch timestamp.
// It returns an error when the cache is missing, unreadable or empty;
// staleness is the caller's decision (see GetModels).
func LoadCache() ([]ModelInfo, time.Time, error) {
	return loadCacheNamed(cacheFileName, "OpenRouter")
}

// LoadNvidiaCache reads the cached NVIDIA model list (same envelope).
func LoadNvidiaCache() ([]ModelInfo, time.Time, error) {
	return loadCacheNamed(nvidiaCacheFileName, "NVIDIA")
}

// loadCacheNamed is the shared cache reader; label names the backend in
// error messages.
func loadCacheNamed(name, label string) ([]ModelInfo, time.Time, error) {
	dir, err := CacheDir()
	if err != nil {
		return nil, time.Time{}, err
	}
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return nil, time.Time{}, err
	}
	var cc cachedCatalog
	if err := json.Unmarshal(data, &cc); err != nil {
		return nil, time.Time{}, fmt.Errorf("cache %s illisible: %w", label, err)
	}
	if len(cc.Models) == 0 {
		return nil, time.Time{}, fmt.Errorf("cache %s vide", label)
	}
	return cc.Models, cc.FetchedAt, nil
}

// SaveCache writes the OpenRouter model list to the cache file atomically,
// creating the cache directory when needed.
func SaveCache(models []ModelInfo) error {
	return saveCacheNamed(cacheFileName, models)
}

// SaveNvidiaCache writes the NVIDIA model list to its own cache file.
func SaveNvidiaCache(models []ModelInfo) error {
	return saveCacheNamed(nvidiaCacheFileName, models)
}

// saveCacheNamed is the shared atomic cache writer.
func saveCacheNamed(name string, models []ModelInfo) error {
	dir, err := CacheDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creation du dossier cache impossible: %w", err)
	}
	data, err := json.Marshal(cachedCatalog{FetchedAt: time.Now().UTC(), Models: models})
	if err != nil {
		return err
	}
	path := filepath.Join(dir, name)
	return fsutil.WriteFileAtomic(path, data, 0o644)
}
