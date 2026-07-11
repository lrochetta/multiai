// Package registry implements the community profile registry client for
// multiai. It fetches the profile index from the community registry GitHub
// repository (github.com/lrochetta/profiles-multiai), caches it locally with a
// 1-hour TTL, and provides search and display helpers.
package registry

import "time"

// Index represents the community profile registry index.json structure.
type Index struct {
	Version   int            `json:"version"`
	UpdatedAt time.Time      `json:"updated_at"`
	Total     int            `json:"total"`
	Profiles  []ProfileEntry `json:"profiles"`
}

// ProfileEntry represents a single community-contributed profile in the index.
type ProfileEntry struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Stars       int      `json:"stars"`
	Tags        []string `json:"tags,omitempty"`
	// DownloadURL overrides the default download URL for the profile .env file.
	// When empty the client constructs the URL from the registry base and name.
	DownloadURL string `json:"download_url,omitempty"`
	// SHA256 is the expected hex-encoded SHA-256 checksum of the profile .env
	// file. When set, the installer verifies the download against this value.
	SHA256 string `json:"sha256,omitempty"`
}
