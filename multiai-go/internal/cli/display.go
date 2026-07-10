// Package cli provides the CLI entry-point utilities including the main menu
// display functions. The colored-print functions are now deprecated wrappers
// around internal/display — kept for callers in cmd/multiai/main.go.
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/lrochetta/multiai/internal/display"
	"github.com/lrochetta/multiai/internal/profile"
)

// Colorize wraps text in the given ANSI code, respecting NO_COLOR.
// Deprecated: use display.Colorize (note: arg order is color, text).
func Colorize(text, ansiCode string) string {
	return display.Colorize(ansiCode, text)
}

// StatusColor returns an ANSI color code for a configuration status.
//  [OK] -> green, [~~] -> yellow, [--] -> dim/grey
// Deprecated: use display.StatusColor.
func StatusColor(configured, total int) string {
	return display.StatusColor(configured, total)
}

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
		out := make([]profileJSON, 0, len(profiles))
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

// PrintSuccess prints a green success message with [OK] prefix.
// Deprecated: use display.PrintSuccess.
func PrintSuccess(msg string) {
	display.PrintSuccess(msg)
}

// PrintWarning prints a yellow warning message with [!] prefix.
// Deprecated: use display.PrintWarning.
func PrintWarning(msg string) {
	display.PrintWarning(msg)
}

// PrintError prints a red error message with [X] prefix.
// Deprecated: use display.PrintError.
func PrintError(msg string) {
	display.PrintError(msg)
}

// PrintInfo prints a cyan info message with [i] prefix.
// Deprecated: use display.PrintInfo.
func PrintInfo(msg string) {
	display.PrintInfo(msg)
}
