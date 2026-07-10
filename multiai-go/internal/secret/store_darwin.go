//go:build darwin

package secret

// keychainStore stores credentials in the macOS Keychain.
//
// When CGo is available (the normal case on macOS) it uses the Security
// Framework directly via store_darwin_cgo.go.  When CGo is disabled the
// package falls back to shelling out to /usr/bin/security (store_darwin_nocgo.go).
//
// Service names are mapped to kSecAttrService, credential keys to
// kSecAttrAccount, and values to kSecValueData.
type keychainStore struct{}

func newPlatformStore() (Store, error) {
	if keychainAvailable() {
		return &keychainStore{}, nil
	}
	return newEncryptedFileStore()
}

func (s *keychainStore) Get(service, key string) (string, error) {
	return keychainGet(service, key)
}

func (s *keychainStore) Set(service, key, value string) error {
	return keychainSet(service, key, value)
}

func (s *keychainStore) Delete(service, key string) error {
	return keychainDelete(service, key)
}

func (s *keychainStore) List(service string) (map[string]string, error) {
	return keychainList(service)
}

// ── Dump-keychain parser (used by the shell-out backend) ──────────────

// dumpEntry represents a parsed generic-password entry from
// "security dump-keychain -r genp".
type dumpEntry struct {
	Account string
	Service string
}

// parseDumpKeychain parses the output of "security dump-keychain -r genp"
// and returns every generic-password entry found.
//
// Entry-separator heuristic: a line that starts with "keychain:" begins a
// new record.  The format on macOS 13+ looks like:
//
//	keychain: "/path/to/login.keychain-db"
//	version: 512
//	class: "genp"
//	attributes:
//	    0x00000007 <blob> = "account-name"   (kSecAttrAccount)
//	    0x00000008 <blob> = "service-name"   (kSecAttrService)
//	    ...
func parseDumpKeychain(out string) []dumpEntry {
	var entries []dumpEntry
	lines := splitLines(out)
	if len(lines) == 0 {
		return nil
	}

	// Collect blocks separated by "keychain:" headers.
	var blocks []string
	var buf []string
	for _, line := range lines {
		if len(buf) > 0 && isKeychainHeader(line) {
			blocks = append(blocks, joinLines(buf))
			buf = buf[:0]
		}
		buf = append(buf, line)
	}
	if len(buf) > 0 {
		blocks = append(blocks, joinLines(buf))
	}

	for _, block := range blocks {
		if !containsGenp(block) {
			continue
		}
		entry := dumpEntry{}
		for _, line := range splitLines(block) {
			line = trimSpace(line)
			switch {
			case hasPrefix(line, "0x00000007 <blob> = "):
				entry.Account = extractQuotedValue(line)
			case hasPrefix(line, "0x00000008 <blob> = "):
				entry.Service = extractQuotedValue(line)
			}
		}
		if entry.Service != "" && entry.Account != "" {
			entries = append(entries, entry)
		}
	}
	return entries
}

// extractQuotedValue extracts the content between the first and last
// double-quote pair on the line: `0x00000007 <blob> = "hello"` → "hello".
func extractQuotedValue(line string) string {
	start := indexByte(line, '"')
	if start < 0 {
		return ""
	}
	end := lastIndexByte(line, '"')
	if end <= start {
		return ""
	}
	return line[start+1 : end]
}

// ── Minimal string helpers (no imports beyond built-in) ───────────────

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func joinLines(lines []string) string {
	var out string
	for i, l := range lines {
		if i > 0 {
			out += "\n"
		}
		out += l
	}
	return out
}

func trimSpace(s string) string {
	lo, hi := 0, len(s)
	for lo < hi && (s[lo] == ' ' || s[lo] == '\t') {
		lo++
	}
	for hi > lo && (s[hi-1] == ' ' || s[hi-1] == '\t') {
		hi--
	}
	return s[lo:hi]
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func lastIndexByte(s string, b byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func isKeychainHeader(line string) bool {
	return hasPrefix(trimSpace(line), "keychain:")
}

func containsGenp(block string) bool {
	return indexOf(block, `class: "genp"`) >= 0
}

func indexOf(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
