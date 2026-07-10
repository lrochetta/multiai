package dotenv

import (
	"testing"
)

// FuzzParse fuzzes the Parse function via ParseBytes.
// It must never panic for any input.
//
//go:noinline
func FuzzParse(f *testing.F) {
	// Seed corpus: representative .env patterns
	seeds := []string{
		// Standard KEY=value
		"KEY1=value1\nKEY2=value2\n",
		// With double quotes
		`KEY="quoted value"` + "\n",
		// With single quotes
		`KEY='quoted value'` + "\n",
		// With comments
		"# this is a comment\nKEY=value\n",
		// With export prefix
		"export KEY=value\n",
		// Multiline content (several lines)
		"KEY1=a\nKEY2=b\nKEY3=c\n",
		// Empty lines
		"\n\nKEY=val\n\n",
		// Empty file
		"",
		// Value with equals signs
		"KEY=value=with=equals\n",
		// Only comment
		"# just a comment\n",
		// Only export
		"export\n",
		// Malformed line
		"MALFORMED\nKEY=val\n",
		// BOM prefix (\xef\xbb\xbfKEY=bom\n)
		"\xef\xbb\xbfKEY=bom\n",
		// Spaces around key/value
		"  KEY  =  value  \n",
	}

	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		// Must not panic.
		result, err := ParseBytes(data)

		// err nil => result must be non-nil
		if err == nil && result == nil {
			t.Errorf("ParseBytes returned nil map with nil error for input: %q", string(data))
		}
		// err != nil => result may be nil or empty, either is fine
		// But ParseBytes never panics — that's the main invariant.
	})
}
