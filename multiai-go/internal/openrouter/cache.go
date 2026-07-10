package openrouter

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lrochetta/multiai/internal/fsutil"
)

// cacheFileName is the on-disk name of the cached /models payload.
const cacheFileName = "openrouter-models.json"

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

// LoadCache reads the cached model list and its fetch timestamp.
// It returns an error when the cache is missing, unreadable or empty;
// staleness is the caller's decision (see GetModels).
func LoadCache() ([]ModelInfo, time.Time, error) {
	dir, err := CacheDir()
	if err != nil {
		return nil, time.Time{}, err
	}
	data, err := os.ReadFile(filepath.Join(dir, cacheFileName))
	if err != nil {
		return nil, time.Time{}, err
	}
	var cc cachedCatalog
	if err := json.Unmarshal(data, &cc); err != nil {
		return nil, time.Time{}, fmt.Errorf("cache OpenRouter illisible: %w", err)
	}
	if len(cc.Models) == 0 {
		return nil, time.Time{}, errors.New("cache OpenRouter vide")
	}
	return cc.Models, cc.FetchedAt, nil
}

// SaveCache writes the model list to the cache file atomically, creating the
// cache directory when needed.
func SaveCache(models []ModelInfo) error {
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
	path := filepath.Join(dir, cacheFileName)
	return fsutil.WriteFileAtomic(path, data, 0o644)
}
