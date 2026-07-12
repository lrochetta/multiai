package secret

import (
	"crypto/sha256"
	"fmt"
)

// ── Target-naming helpers ──────────────────────────────────────────────

// targetName returns the Windows Credential Manager target name for a
// (service, key) pair.
//
// Format: mti_<hex(sha256(service)[:8])>_<key>
//
// The 8-byte hash prefix keeps the name short while preventing collision
// between services that happen to share a key name.
func targetName(service, key string) string {
	h := sha256.Sum256([]byte(service))
	return fmt.Sprintf("mti_%x_%s", h[:8], key)
}

// serviceFilter returns the CredEnumerateW filter string that matches every
// credential belonging to the given service.
func serviceFilter(service string) string {
	h := sha256.Sum256([]byte(service))
	return fmt.Sprintf("mti_%x_*", h[:8])
}

// extractKey parses the key from a targetName-formatted string.
// Returns empty string when the format does not match.
func extractKey(target string) string {
	// Format: mti_<16hex>_<key>  (4 + 16 + 1 = 21 prefix chars)
	if len(target) < 22 || target[:4] != "mti_" || target[20] != '_' {
		return ""
	}
	for i := 4; i < 20; i++ {
		c := target[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return ""
		}
	}
	return target[21:]
}
