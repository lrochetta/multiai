// Package openrouter implements the OpenRouter model discovery features
// of multiai: fetching the public /models catalog, caching it locally,
// full-text search, model comparison and dynamic launch profile creation.
//
// Note: these capabilities go beyond the PowerShell reference (which only
// shows a static help screen plus a .env generator); the network-backed
// discovery is a Go-only feature.
package openrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultAPIBase = "https://openrouter.ai/api/v1"
	userAgent      = "multiai"
	httpTimeout    = 10 * time.Second

	// CacheTTL is how long the cached model list is considered fresh.
	CacheTTL = time.Hour
)

// apiBase is a variable so tests can point the client at a local server.
var apiBase = defaultAPIBase

// maxResponseBytes caps the size of the /models payload. Variable so
// tests can exercise the limit without generating tens of megabytes.
var maxResponseBytes int64 = 32 << 20

// ModelInfo mirrors one entry of the OpenRouter /models response.
type ModelInfo struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	Created       int64        `json:"created"`
	Description   string       `json:"description"`
	ContextLength int          `json:"context_length"`
	Architecture  Architecture `json:"architecture"`
	Pricing       ModelPricing `json:"pricing"`
	TopProvider   TopProvider  `json:"top_provider"`
}

// Architecture describes the model modality and tokenizer.
type Architecture struct {
	Modality  string `json:"modality"`
	Tokenizer string `json:"tokenizer"`
}

// ModelPricing holds per-token USD prices as decimal strings, as returned
// by the API (e.g. "0.000003" per token = 3.00 USD per million tokens).
type ModelPricing struct {
	Prompt     string `json:"prompt"`
	Completion string `json:"completion"`
}

// TopProvider mirrors the top_provider block of the API.
type TopProvider struct {
	ContextLength       int  `json:"context_length"`
	MaxCompletionTokens int  `json:"max_completion_tokens"`
	IsModerated         bool `json:"is_moderated"`
}

// FetchModels retrieves the model catalog from the OpenRouter API.
// The /models endpoint is public: apiKey may be empty; when provided it
// is sent as a Bearer token.
func FetchModels(ctx context.Context, apiKey string) ([]ModelInfo, error) {
	req, err := http.NewRequest(http.MethodGet, apiBase+"/models", nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("api OpenRouter inaccessible: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api OpenRouter: statut HTTP %d", resp.StatusCode)
	}

	lr := &io.LimitedReader{R: resp.Body, N: maxResponseBytes + 1}
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, fmt.Errorf("lecture de la reponse OpenRouter impossible: %w", err)
	}
	if lr.N <= 0 {
		return nil, fmt.Errorf("reponse OpenRouter trop volumineuse (limite %d octets)", maxResponseBytes)
	}

	var result struct {
		Data []ModelInfo `json:"data"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("reponse OpenRouter illisible: %w", err)
	}
	return result.Data, nil
}
