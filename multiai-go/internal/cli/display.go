package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/lrochetta/multiai/internal/profile"
)

// noColor disables ANSI escape codes when the NO_COLOR env var is set.
var noColor = os.Getenv("NO_COLOR") != ""

// Colorize wraps text in the given ANSI code, respecting NO_COLOR.
func Colorize(text, ansiCode string) string {
	if noColor {
		return text
	}
	return ansiCode + text + "\033[0m"
}

// colorize keeps the internal name for backwards compat within this package.
func colorize(text, ansiCode string) string {
	return Colorize(text, ansiCode)
}

// StatusColor returns an ANSI color code for a configuration status.
//  [OK] → green, [~~] → yellow, [--] → dim/grey
func StatusColor(configured, total int) string {
	if configured == total && total > 0 {
		return "\033[32m" // green
	}
	if configured > 0 {
		return "\033[33m" // yellow
	}
	return "\033[90m" // dim grey
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
func PrintSuccess(msg string) {
	fmt.Printf("%s\n", colorize("[OK] "+msg, "\033[32m"))
}

// PrintWarning prints a yellow warning message with [!] prefix.
func PrintWarning(msg string) {
	fmt.Printf("%s\n", colorize("[!] "+msg, "\033[33m"))
}

// PrintError prints a red error message with [X] prefix.
func PrintError(msg string) {
	fmt.Printf("%s\n", colorize("[X] "+msg, "\033[31m"))
}

// PrintInfo prints a cyan info message with [i] prefix.
func PrintInfo(msg string) {
	fmt.Printf("%s\n", colorize("[i] "+msg, "\033[36m"))
}
