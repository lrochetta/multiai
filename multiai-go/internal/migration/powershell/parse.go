package powershell

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lrochetta/multiai/pkg/dotenv"
)

// PSProfile represents a single profile loaded from a legacy PS .env file.
type PSProfile struct {
	// FileName is the base name of the .env file (e.g. "30-claude-deepseek-v4-pro.env").
	FileName string `json:"file_name"`
	// ProfileID is the PROFILE_ID value, or the filename stem if absent.
	ProfileID string `json:"profile_id"`
	// Shortcut is the SHORTCUT value, or ProfileID if absent.
	Shortcut string `json:"shortcut"`
	// Tool is the TOOL value (claude, codex, opencode).
	Tool string `json:"tool"`
	// DisplayName is the display name.
	DisplayName string `json:"display_name"`
	// EnvVars is the full raw key-value map from the .env file, including
	// metadata keys (for faithful backup).
	EnvVars map[string]string `json:"env_vars"`
}

// ParseLegacyProfiles reads all .env files from a legacy PS profiles directory.
func ParseLegacyProfiles(dir string) ([]PSProfile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot read legacy profiles directory %s: %w", dir, err)
	}

	var profiles []PSProfile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".env") {
			continue
		}

		fullPath := filepath.Join(dir, entry.Name())
		f, err := os.Open(fullPath)
		if err != nil {
			return nil, fmt.Errorf("cannot open %s: %w", entry.Name(), err)
		}

		envMap, err := dotenv.Parse(f)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("cannot parse %s: %w", entry.Name(), err)
		}

		p := PSProfile{
			FileName:  entry.Name(),
			ProfileID: strings.TrimSuffix(entry.Name(), ".env"),
			Shortcut:  strings.TrimSuffix(entry.Name(), ".env"),
			Tool:      "claude",
			EnvVars:   envMap,
		}

		if id, ok := envMap["PROFILE_ID"]; ok {
			p.ProfileID = id
		}
		if sc, ok := envMap["SHORTCUT"]; ok {
			p.Shortcut = sc
		}
		if t, ok := envMap["TOOL"]; ok {
			p.Tool = t
		}
		if dn, ok := envMap["DISPLAY_NAME"]; ok {
			p.DisplayName = dn
		}

		profiles = append(profiles, p)
	}

	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].FileName < profiles[j].FileName
	})

	return profiles, nil
}
