package config

import (
	"testing"

	"github.com/lrochetta/multiai/internal/catalog"
	"github.com/lrochetta/multiai/internal/profile"
)

// FuzzConfigWizard fuzzes the config wizard input-parsing and validation
// paths that are safe to run without side effects:
//   - validateAPIKey with random keys against known provider patterns
//   - shortcutIndex with random shortcut strings
//   - providerStatus with random profile/env mappings
//
// It must never panic for any input.
//
//go:noinline
func FuzzConfigWizard(f *testing.F) {
	// Seed corpus: representative user inputs
	seeds := []struct {
		shortcut string
		key      string
	}{
		{"ds", "sk-ant-api03-abcdefghijklmnopqrstuvwxyzABCDEFG"},
		{"or", "sk-or-v1-1234567890abcdef"},
		{"ch", ""},
		{"xx", "__MULTIAI_CREDSTORE__"},
		{"ab", "too-short"},
		{"", "sk-valid-but-unknown-provider-12345678"},
		{"999", "PASTE_API_KEY_HERE"},
		{"\n", "\n"},
		{"PATH", "value with spaces"},
		{"special!@#", "line1\nline2"},
	}

	for _, s := range seeds {
		f.Add(s.shortcut, s.key)
	}

	f.Fuzz(func(t *testing.T, shortcut, key string) {
		cat := catalog.Default()

		// ---- Part 1: validateAPIKey with random keys ----
		for _, prov := range cat.Providers {
			valid, msg := validateAPIKey(prov, key)
			// Must never panic, never return empty msg when valid=false
			// and msg is advisory (human-readable).
			if !valid && msg == "" {
				t.Errorf("validateAPIKey(%q, %q) returned valid=false but empty message", prov.ID, key)
			}
			// validateAPIKey must be deterministic.
			valid2, msg2 := validateAPIKey(prov, key)
			if valid != valid2 || msg != msg2 {
				t.Errorf("validateAPIKey(%q, %q) non-deterministic", prov.ID, key)
			}
		}

		// ---- Part 2: shortcutIndex with random shortcut-like data ----
		profiles := []profile.Profile{
			{Shortcut: shortcut, Env: map[string]string{"KEY": key}},
			{Shortcut: key, Env: map[string]string{"OTHER": shortcut}},
		}
		byShortcut := shortcutIndex(profiles)
		// Must never panic and must be deterministic.
		byShortcut2 := shortcutIndex(profiles)
		if len(byShortcut) != len(byShortcut2) {
			t.Errorf("shortcutIndex non-deterministic: %d vs %d", len(byShortcut), len(byShortcut2))
		}

		// ---- Part 3: providerStatus with random profiles ----
		for _, prov := range cat.Providers {
			configured, total := providerStatus(prov, byShortcut)
			if configured < 0 || total < 0 {
				t.Errorf("providerStatus returned negative counts: configured=%d total=%d", configured, total)
			}
			if configured > total {
				t.Errorf("providerStatus: configured=%d > total=%d", configured, total)
			}
			// providerStatus must be deterministic.
			configured2, total2 := providerStatus(prov, byShortcut)
			if configured != configured2 || total != total2 {
				t.Errorf("providerStatus(%q) non-deterministic", prov.ID)
			}
		}
	})
}

// FuzzConfigEraseKeys fuzzes the erase-key menu logic with random provider
// IDs and profile configurations. It requires the credential store, so it
// must never panic but may return errors.
//
//go:noinline
func FuzzConfigEraseKeys(f *testing.F) {
	seeds := []struct {
		provID      string
		profileName string
	}{
		{"openrouter", "or"},
		{"anthropic", "ds"},
		{"openai", "ch"},
		{"unknown", "xx"},
		{"", ""},
	}

	for _, s := range seeds {
		f.Add(s.provID, s.profileName)
	}

	f.Fuzz(func(t *testing.T, provID, profileName string) {
		cat := catalog.Default()

		// Build a mini profile map that may or may not match the provider.
		profiles := []profile.Profile{
			{Shortcut: profileName, Env: map[string]string{"API_KEY": profileName}},
		}
		byShortcut := shortcutIndex(profiles)

		// Find the provider by ID.
		prov, ok := cat.ProviderByID(provID)
		if !ok {
			return // nothing to test
		}

		// EraseProviderKeys must never panic (may fail gracefully).
		_ = EraseProviderKeys(prov, byShortcut, nil)
	})
}
