//go:build linux

package secret

import (
	"fmt"
	"os/exec"
	"strings"
)

// execCommand is overridden in tests.
var execCommand = exec.Command

// libsecretStore stores credentials via the system secret-service D-Bus API
// using the secret-tool CLI (libsecret). The actual secret is piped through
// stdin on store, never passed as an argument.
//
// If secret-tool is not available in PATH, newPlatformStore returns an error.
// The caller (S5.6) is responsible for falling back to the encrypted file store.
type libsecretStore struct{}

// secretToolCheckPath wraps exec.LookPath so tests can mock availability checks.
var secretToolLookPath = exec.LookPath

func newPlatformStore() (Store, error) {
	if _, err := secretToolLookPath("secret-tool"); err != nil {
		return nil, fmt.Errorf("secret-tool not found in PATH: %w", err)
	}
	return &libsecretStore{}, nil
}

// newNamedStore returns the requested named backend on Linux.
func newNamedStore(backend string) (Store, error) {
	switch backend {
	case "secret-service":
		return newPlatformStore()
	default:
		return nil, fmt.Errorf("unsupported backend on this platform: %s (supported: secret-service, file, auto)", backend)
	}
}

// Get retrieves a credential via "secret-tool lookup".
// Returns an error (not found) if secret-tool exits non-zero or the value is empty.
func (s *libsecretStore) Get(service, key string) (string, error) {
	cmd := execCommand("secret-tool", "lookup", "service", service, "key", key)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("credential not found: %s/%s", service, key)
	}
	val := strings.TrimRight(string(out), "\n\r")
	if val == "" {
		return "", fmt.Errorf("credential not found: %s/%s", service, key)
	}
	return val, nil
}

// Set stores a credential via "secret-tool store". The secret value is piped
// through stdin to avoid leaking it into the process argument list.
func (s *libsecretStore) Set(service, key, value string) error {
	cmd := execCommand("secret-tool", "store", "--label=multiai", "service", service, "key", key)
	cmd.Stdin = strings.NewReader(value)
	if _, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("secret-tool store %s/%s: %w", service, key, err)
	}
	return nil
}

// Delete removes a credential via "secret-tool clear".
func (s *libsecretStore) Delete(service, key string) error {
	cmd := execCommand("secret-tool", "clear", "service", service, "key", key)
	if _, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("secret-tool clear %s/%s: %w", service, key, err)
	}
	return nil
}

// List returns all key→value pairs stored under the given service via
// "secret-tool search".
func (s *libsecretStore) List(service string) (map[string]string, error) {
	cmd := execCommand("secret-tool", "search", "service", service)
	out, err := cmd.Output()
	if err != nil {
		// secret-tool search exits with code 3 or 1 when no secrets match.
		// Treat this as an empty result set, not a hard error.
		return make(map[string]string), nil
	}
	return parseSecretToolSearch(string(out)), nil
}

// parseSecretToolSearch parses the INI-like output of "secret-tool search".
//
// Example output:
//
//	[/org/freedesktop/secrets/collection/login/1]
//	label = multiai
//	service = multiai:ca-a1b2c3d4
//	key = ANTHROPIC_API_KEY
//	value = sk-ant-...
//
//	[/org/freedesktop/secrets/collection/login/2]
//	label = multiai
//	service = multiai:ca-a1b2c3d4
//	key = OPENAI_API_KEY
//	value = sk-openai-...
//
// Each entry block is delimited by a [...] header. Only "key" and "value"
// attributes are extracted; "service", "label", and other attributes are ignored.
func parseSecretToolSearch(output string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(output, "\n")

	var currentKey, currentValue string
	inEntry := false

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		trimmed := strings.TrimSpace(line)

		// Entry header: [...] — flush previous entry and start a new one.
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			if inEntry && currentKey != "" {
				result[currentKey] = currentValue
			}
			inEntry = true
			currentKey = ""
			currentValue = ""
			continue
		}

		if !inEntry || trimmed == "" {
			continue
		}

		// Extract key=value attributes. Only care about "key" and "value".
		before, after, found := strings.Cut(trimmed, " = ")
		if !found {
			continue
		}
		switch before {
		case "key":
			currentKey = after
		case "value":
			currentValue = after
		}
	}

	// Flush the last entry.
	if inEntry && currentKey != "" {
		result[currentKey] = currentValue
	}

	return result
}
