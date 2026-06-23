package openrouter

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const baseURL = "https://openrouter.ai/api/v1"

type ModelInfo struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	ContextLength int           `json:"context_length"`
	Architecture  string        `json:"architecture"`
	Pricing       ModelPricing  `json:"pricing"`
	TopProvider   ProviderInfo  `json:"top_provider"`
	Description   string        `json:"description"`
}

type ModelPricing struct {
	Prompt     string `json:"prompt"`
	Completion string `json:"completion"`
}

type ProviderInfo struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// FetchModels retrieves available models from OpenRouter API.
func FetchModels(apiKey string) ([]ModelInfo, error) {
	req, _ := http.NewRequest("GET", baseURL+"/models", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("User-Agent", "multiai/0.2.1")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API OpenRouter inaccessible: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API OpenRouter error %d", resp.StatusCode)
	}

	var result struct {
		Data []ModelInfo `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("cannot parse OpenRouter response: %w", err)
	}
	return result.Data, nil
}

// CacheModels caches models locally to avoid hitting the API too often.
func CacheModels(models []ModelInfo) error {
	cacheDir, _ := os.UserHomeDir()
	cacheDir = filepath.Join(cacheDir, ".multiai", "cache")
	os.MkdirAll(cacheDir, 0700)

	data, _ := json.MarshalIndent(models, "", "  ")
	return os.WriteFile(filepath.Join(cacheDir, "openrouter-models.json"), data, 0600)
}

// LoadCachedModels loads models from cache.
func LoadCachedModels() ([]ModelInfo, error) {
	cacheDir, _ := os.UserHomeDir()
	cachePath := filepath.Join(cacheDir, ".multiai", "cache", "openrouter-models.json")

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var models []ModelInfo
	if err := json.Unmarshal(data, &models); err != nil {
		return nil, err
	}
	return models, nil
}

// IsCacheValid returns true if the cache is less than maxAge old.
func IsCacheValid(maxAge time.Duration) bool {
	cacheDir, _ := os.UserHomeDir()
	cachePath := filepath.Join(cacheDir, ".multiai", "cache", "openrouter-models.json")
	info, err := os.Stat(cachePath)
	if err != nil {
		return false
	}
	return time.Since(info.ModTime()) < maxAge
}
