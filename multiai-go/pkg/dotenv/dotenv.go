// Package dotenv implements a robust .env file parser.
// Supports: export prefix, double/single quotes, comments, empty lines.
package dotenv

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Parse reads a .env file from r and returns a map of key-value pairs.
// Lines starting with # are treated as comments.
// Empty lines are skipped.
// Supports:
//   - KEY=value
//   - export KEY=value
//   - KEY="value with spaces"
//   - KEY='value with spaces'
func Parse(r io.Reader) (map[string]string, error) {
	result := make(map[string]string)
	scanner := bufio.NewScanner(r)
	lineNum := 0
	first := true

	for scanner.Scan() {
		lineNum++
		raw := scanner.Text()
		if first {
			// Strip a leading UTF-8 BOM: files saved by Notepad, VS Code
			// "UTF-8 with BOM", or PowerShell Out-File/Set-Content -Encoding
			// utf8 start with U+FEFF, which TrimSpace does NOT remove — it
			// would otherwise mangle the first key (e.g. U+FEFF prefixing TOOL).
			raw = strings.TrimPrefix(raw, "\uFEFF")
			first = false
		}
		line := strings.TrimSpace(raw)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Remove export prefix
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimPrefix(line, "export ")
			line = strings.TrimSpace(line)
		}

		// Find first = (key cannot contain =)
		idx := strings.Index(line, "=")
		if idx < 1 {
			continue // malformed line, skip
		}

		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])

		// Remove surrounding quotes
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		result[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading .env at line %d: %w", lineNum, err)
	}

	return result, nil
}

// ParseBytes parses .env content from a byte slice.
func ParseBytes(data []byte) (map[string]string, error) {
	return Parse(strings.NewReader(string(data)))
}

// IsPlaceholder checks if a value looks like an unconfigured placeholder.
func IsPlaceholder(value string) bool {
	v := strings.TrimSpace(value)
	if v == "" {
		return true
	}
	lower := strings.ToLower(v)
	if strings.HasPrefix(lower, "paste_") ||
		strings.HasPrefix(lower, "your_") ||
		strings.HasPrefix(lower, "ta_cle") ||
		strings.HasPrefix(lower, "replace_me") ||
		strings.HasPrefix(lower, "change_me") ||
		strings.HasPrefix(lower, "sk-xxxx") ||
		strings.HasPrefix(lower, "xxx") ||
		strings.HasPrefix(lower, "todo") {
		return true
	}
	if strings.HasSuffix(lower, "_here") || strings.HasSuffix(lower, "ici") {
		return true
	}
	return false
}
