package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultIndexURL = "https://raw.githubusercontent.com/lrochetta/profiles-multiai/main/index.json"
	userAgent       = "multiai-registry"
)

// indexURL and httpTimeout are variables so tests can override them.
var (
	indexURL    = defaultIndexURL
	httpTimeout = 10 * time.Second
)

// FetchIndex retrieves the community profile index from the registry. It tries
// the local cache first; on a cache miss or expiration it fetches from the
// remote URL, caches the result, and returns it.
func FetchIndex(ctx context.Context) (*Index, error) {
	// Try cache first.
	if cached, err := readCache(); err == nil && cached != nil {
		return cached, nil
	}

	// Cache miss or expired — fetch from remote.
	idx, err := fetchFromURL(ctx, resolveURL())
	if err != nil {
		return nil, err
	}

	// Best-effort cache write (failure is non-fatal).
	_ = writeCache(idx)
	return idx, nil
}

// FetchIndexNoCache fetches the community profile index from the remote
// registry, bypassing the local cache entirely.
func FetchIndexNoCache(ctx context.Context) (*Index, error) {
	return fetchFromURL(ctx, resolveURL())
}

// resolveURL returns the default index URL, overridden by the
// MULTIAI_REGISTRY_URL env var when MULTIAI_DEV=1 is also set.
func resolveURL() string {
	url := indexURL
	if devURL := os.Getenv("MULTIAI_REGISTRY_URL"); devURL != "" {
		if os.Getenv("MULTIAI_DEV") == "1" {
			url = devURL
		}
	}
	return url
}

// fetchFromURL performs the HTTP GET and decodes the index JSON.
func fetchFromURL(ctx context.Context, url string) (*Index, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned HTTP %d", resp.StatusCode)
	}

	var idx Index
	if err := json.NewDecoder(resp.Body).Decode(&idx); err != nil {
		return nil, err
	}
	if idx.Profiles == nil {
		idx.Profiles = []ProfileEntry{}
	}
	return &idx, nil
}

// SearchProfiles performs a case-insensitive full-text search over the profile
// index. It matches against the profile Name, Title, Description, Author, and
// Tags fields. Results are returned in no guaranteed order.
func SearchProfiles(idx *Index, query string) []ProfileEntry {
	if idx == nil || query == "" {
		return nil
	}
	q := strings.ToLower(query)
	var results []ProfileEntry
	for _, p := range idx.Profiles {
		if matches(p, q) {
			results = append(results, p)
		}
	}
	return results
}

// matches checks whether a profile entry matches the lowercase query string.
func matches(p ProfileEntry, q string) bool {
	if strings.Contains(strings.ToLower(p.Name), q) {
		return true
	}
	if strings.Contains(strings.ToLower(p.Title), q) {
		return true
	}
	if strings.Contains(strings.ToLower(p.Description), q) {
		return true
	}
	if strings.Contains(strings.ToLower(p.Author), q) {
		return true
	}
	for _, tag := range p.Tags {
		if strings.Contains(strings.ToLower(tag), q) {
			return true
		}
	}
	return false
}

// FindProfileByName looks up a profile in the index by its exact name (case-insensitive).
func FindProfileByName(idx *Index, name string) *ProfileEntry {
	if idx == nil || name == "" {
		return nil
	}
	n := strings.ToLower(name)
	for _, p := range idx.Profiles {
		if strings.ToLower(p.Name) == n {
			return &p
		}
	}
	return nil
}
