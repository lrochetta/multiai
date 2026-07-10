// Package display provides terminal output formatting utilities:
// colored messages (success, warning, error, info) and status helpers.
package display

import (
	"fmt"
	"os"
)

// colorsEnabled is false when the NO_COLOR env var is set, disabling ANSI
// escape codes.
var colorsEnabled = os.Getenv("NO_COLOR") == ""

// Colorize wraps text in the given ANSI color code, respecting NO_COLOR.
func Colorize(color, text string) string {
	if !colorsEnabled {
		return text
	}
	return color + text + "\033[0m"
}

// StatusColor returns an ANSI color code for a configuration status.
//  [OK] -> green, [~~] -> yellow, [--] -> dim/grey
func StatusColor(configured, total int) string {
	if configured == total && total > 0 {
		return "\033[32m" // green
	}
	if configured > 0 {
		return "\033[33m" // yellow
	}
	return "\033[90m" // dim grey
}

// PrintSuccess prints a green success message with [OK] prefix.
func PrintSuccess(msg string) {
	fmt.Printf("%s\n", Colorize("\033[32m", "[OK] "+msg))
}

// PrintWarning prints a yellow warning message with [!] prefix.
func PrintWarning(msg string) {
	fmt.Printf("%s\n", Colorize("\033[33m", "[!] "+msg))
}

// PrintError prints a red error message with [X] prefix.
func PrintError(msg string) {
	fmt.Printf("%s\n", Colorize("\033[31m", "[X] "+msg))
}

// PrintInfo prints a cyan info message with [i] prefix.
func PrintInfo(msg string) {
	fmt.Printf("%s\n", Colorize("\033[36m", "[i] "+msg))
}