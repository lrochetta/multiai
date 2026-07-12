// Package profile defines the Profile type and loading logic.
package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/lrochetta/multiai/pkg/dotenv"
)

// Profile represents a launch profile loaded from a .env file.
type Profile struct {
	ID              string            `json:"id"`
	Shortcut        string            `json:"shortcut"`
	Tool            string            `json:"tool"` // claude, codex, opencode
	ToolLabel       string            `json:"tool_label"`
	DisplayName     string            `json:"display_name"`
	Description     string            `json:"description,omitempty"`
	Order           int               `json:"order"`
	Command         string            `json:"command"`
	Args            []string          `json:"args,omitempty"`
	Env             map[string]string `json:"env"`
	ClearEnv        bool              `json:"clear_env"`
	RequiredSecrets []string          `json:"required_secrets,omitempty"`
	// Fallback lists profile shortcuts/ids to try, in order, when the
	// launched process exits non-zero (FALLBACK=ds,cg). Single level:
	// a fallback profile's own chain is never followed.
	Fallback []string `json:"fallback,omitempty"`
	// Region is informational metadata (REGION=eu). No routing logic reads
	// it; it exists so the key never leaks into the child environment and
	// stays available for display grouping.
	Region string `json:"region,omitempty"`
	// SkipSecretCheck bypasses the REQUIRED_SECRETS placeholder validation
	// when SKIP_SECRET_CHECK=true (parity with code-router.ps1 L455-458).
	SkipSecretCheck bool   `json:"skip_secret_check,omitempty"`
	Path            string `json:"path"` // filesystem path to the .env file
	// Holds before_launch/after_launch hook definitions (YAML profiles only).
	Hooks *HooksConfig `json:"hooks,omitempty"`
}

// MetadataKeys are .env keys that are metadata, not environment variables.
var MetadataKeys = map[string]bool{
	"PROFILE_ID": true, "SHORTCUT": true, "TOOL": true, "TOOL_LABEL": true,
	"DISPLAY_NAME": true, "DESCRIPTION": true, "ORDER": true, "COMMAND": true,
	"ARGS": true, "CLEAR_ENV": true, "REQUIRED_SECRETS": true,
	"SKIP_SECRET_CHECK": true, "NOTES": true,
	"FALLBACK": true, "REGION": true,
}

// LoadDir loads all .env profiles from a directory.
func LoadDir(dir string) ([]Profile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot read profiles directory %s: %w", dir, err)
	}

	var profiles []Profile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".env") {
			continue
		}

		fullPath := filepath.Join(dir, entry.Name())
		f, err := os.Open(fullPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[!] profil ignore (illisible) : %s : %v\n", entry.Name(), err)
			continue
		}
		envMap, err := dotenv.Parse(f)
		f.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[!] profil ignore (malforme) : %s : %v\n", entry.Name(), err)
			continue
		}

		p := Profile{Path: fullPath}
		p.ID = entry.Name()
		if id, ok := envMap["PROFILE_ID"]; ok {
			p.ID = id
		} else {
			p.ID = strings.TrimSuffix(entry.Name(), ".env")
		}

		p.Shortcut = p.ID
		if sc, ok := envMap["SHORTCUT"]; ok {
			p.Shortcut = sc
		}

		p.Tool = "claude"
		if t, ok := envMap["TOOL"]; ok {
			p.Tool = t
		}

		p.ToolLabel = p.Tool
		if tl, ok := envMap["TOOL_LABEL"]; ok {
			p.ToolLabel = tl
		}

		p.DisplayName = p.ID
		if dn, ok := envMap["DISPLAY_NAME"]; ok {
			p.DisplayName = dn
		}

		if desc, ok := envMap["DESCRIPTION"]; ok {
			p.Description = desc
		}

		p.Order = 9999
		if ord, ok := envMap["ORDER"]; ok {
			if n, err := strconv.Atoi(ord); err == nil {
				p.Order = n
			}
		}

		p.Command = p.Tool
		if cmd, ok := envMap["COMMAND"]; ok {
			p.Command = cmd
		}

		if args, ok := envMap["ARGS"]; ok && args != "" {
			p.Args = splitArgs(args)
		}

		p.ClearEnv = true
		if ce, ok := envMap["CLEAR_ENV"]; ok {
			low := strings.ToLower(ce)
			p.ClearEnv = low != "false" && low != "0" && low != "no"
		}

		if rs, ok := envMap["REQUIRED_SECRETS"]; ok {
			p.RequiredSecrets = splitCSV(rs)
		}

		if fb, ok := envMap["FALLBACK"]; ok {
			p.Fallback = splitCSV(fb)
		}

		if region, ok := envMap["REGION"]; ok {
			p.Region = strings.TrimSpace(region)
		}

		if skip, ok := envMap["SKIP_SECRET_CHECK"]; ok {
			// Parity with Test-RequiredSecrets (code-router.ps1 L458):
			// '^(true|1|yes)$', case-insensitive.
			low := strings.ToLower(strings.TrimSpace(skip))
			p.SkipSecretCheck = low == "true" || low == "1" || low == "yes"
		}

		// Store non-metadata keys as environment variables
		p.Env = make(map[string]string)
		for k, v := range envMap {
			if MetadataKeys[k] {
				continue
			}
			p.Env[k] = v
		}

		profiles = append(profiles, p)
	}

	// Sort by Tool, Order, DisplayName
	sort.Slice(profiles, func(i, j int) bool {
		if profiles[i].Tool != profiles[j].Tool {
			return profiles[i].Tool < profiles[j].Tool
		}
		if profiles[i].Order != profiles[j].Order {
			return profiles[i].Order < profiles[j].Order
		}
		return profiles[i].DisplayName < profiles[j].DisplayName
	})

	return profiles, nil
}

// FindByShortcut finds a profile by its ID, shortcut, or filename (case-insensitive).
func FindByShortcut(profiles []Profile, name string) (*Profile, error) {
	var matches []Profile
	for _, p := range profiles {
		if strings.EqualFold(p.ID, name) || strings.EqualFold(p.Shortcut, name) ||
			strings.EqualFold(strings.TrimSuffix(filepath.Base(p.Path), ".env"), name) {
			matches = append(matches, p)
		}
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("profil introuvable : %s. Lance 'multiai -List' pour voir les profils", name)
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("plusieurs profils correspondent a : %s. Utilise l'id exact", name)
	}
	return &matches[0], nil
}

// splitCSV splits a comma-separated metadata value, trimming whitespace and
// dropping empty items (parity with the PS reference: split, trim, filter).
func splitCSV(s string) []string {
	var out []string
	for _, item := range strings.Split(s, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

// splitArgs parses a string of arguments respecting quoted substrings.
func splitArgs(s string) []string {
	var result []string
	var current strings.Builder
	inDouble := false
	inSingle := false

	for _, ch := range s {
		switch {
		case ch == '"' && !inSingle:
			inDouble = !inDouble
		case ch == '\'' && !inDouble:
			inSingle = !inSingle
		case (ch == ' ' || ch == '\t') && !inDouble && !inSingle:
			if current.Len() > 0 {
				result = append(result, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(ch)
		}
	}
	if current.Len() > 0 {
		result = append(result, current.String())
	}
	return result
}
