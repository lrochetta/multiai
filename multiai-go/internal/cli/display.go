package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/lrochetta/multiai/internal/profile"
)

// ListProfiles displays all profiles.
func ListProfiles(profiles []profile.Profile, asJSON bool) error {
	if asJSON {
		type profileJSON struct {
			Tool        string `json:"tool"`
			Shortcut    string `json:"shortcut"`
			DisplayName string `json:"display_name"`
			Description string `json:"description,omitempty"`
			Command     string `json:"command"`
			Args        string `json:"args,omitempty"`
		}
		var out []profileJSON
		for _, p := range profiles {
			out = append(out, profileJSON{
				Tool:        p.Tool,
				Shortcut:    p.Shortcut,
				DisplayName: p.DisplayName,
				Description: p.Description,
				Command:     p.Command,
				Args:        strings.Join(p.Args, " "),
			})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Tool\tShortcut\tDisplay Name\tCommand")
	fmt.Fprintln(w, "----\t--------\t------------\t-------")
	for _, p := range profiles {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.Tool, p.Shortcut, p.DisplayName, p.Command)
	}
	return w.Flush()
}

// PrintSuccess prints a green success message.
func PrintSuccess(msg string) {
	fmt.Printf("\033[32m%s\033[0m\n", msg)
}

// PrintWarning prints a yellow warning message.
func PrintWarning(msg string) {
	fmt.Printf("\033[33m%s\033[0m\n", msg)
}

// PrintError prints a red error message.
func PrintError(msg string) {
	fmt.Printf("\033[31m%s\033[0m\n", msg)
}

// PrintInfo prints a cyan info message.
func PrintInfo(msg string) {
	fmt.Printf("\033[36m%s\033[0m\n", msg)
}
