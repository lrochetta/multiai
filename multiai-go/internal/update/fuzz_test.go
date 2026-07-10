package update

import (
	"testing"
)

// FuzzVersionParse fuzzes the IsNewer semver comparison function with random
// version strings. It must never panic for any input.
//
//go:noinline
func FuzzVersionParse(f *testing.F) {
	// Seed corpus: representative version strings covering the semver space
	seeds := []struct {
		current string
		latest  string
	}{
		{"0.4.0", "0.4.1"},
		{"v0.4.0", "v0.4.1"},
		{"1.0.0", "2.0.0"},
		{"", "1.0.0"},
		{"0.4.1-beta", "0.4.1"},
		{"0.4.1", "0.4.1-beta"},
		{"abc", "1.0"},
		{"1.2.3", "1.2.3"},
		{"v1.2.3-alpha", "v1.2.3-beta"},
		{"10.20.30", "10.20.31"},
		{"0.0.0", "0.0.1"},
		{"1", "2"},
		{"1.0", "1.1"},
		{"v1.0.0-rc1", "v1.0.0"},
		{"v1.0.0", "v1.0.0-rc2"},
		{"999.999.999", "1000.0.0"},
	}

	for _, s := range seeds {
		f.Add(s.current, s.latest)
	}

	f.Fuzz(func(t *testing.T, current, latest string) {
		// IsNewer must never panic for any input.
		result := IsNewer(current, latest)

		// Basic invariants:
		// - IsNewer(a, b) should be false for identical strings.
		if current == latest && result {
			t.Errorf("IsNewer(%q, %q) = true for identical versions", current, latest)
		}

		// - IsNewer must be deterministic (calling twice gives same result).
		result2 := IsNewer(current, latest)
		if result != result2 {
			t.Errorf("IsNewer(%q, %q) non-deterministic: %v then %v", current, latest, result, result2)
		}

		// - Asymmetry: IsNewer(a, b) && IsNewer(b, a) must never both be true.
		reverse := IsNewer(latest, current)
		if result && reverse {
			t.Errorf("IsNewer(%q, %q) and IsNewer(%q, %q) both returned true", current, latest, latest, current)
		}
	})
}

// FuzzParseVersionPart fuzzes the internal parseVersionPart function directly.
// It must never panic for any input string.
//
//go:noinline
func FuzzParseVersionPart(f *testing.F) {
	seeds := []string{
		"1", "42", "0", "1-beta", "2-alpha", "abc",
		"", "10.20", "v1", "1.2.3-rc1+build42",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		// Must never panic.
		num, suffix, ok := parseVersionPart(s)

		// Basic invariants:
		if ok {
			if num < 0 {
				t.Errorf("parseVersionPart(%q) returned negative number %d", s, num)
			}
			if suffix != "" && len(suffix) >= len(s) {
				t.Errorf("parseVersionPart(%q) suffix %q is longer than input", s, suffix)
			}
		} else {
			if num != 0 || suffix != "" {
				t.Errorf("parseVersionPart(%q) returned ok=false but num=%d suffix=%q", s, num, suffix)
			}
		}

		// Determinism.
		num2, suffix2, ok2 := parseVersionPart(s)
		if ok != ok2 || num != num2 || suffix != suffix2 {
			t.Errorf("parseVersionPart(%q) non-deterministic", s)
		}
	})
}
