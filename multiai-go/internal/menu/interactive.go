package menu

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/lrochetta/multiai/internal/display"
	"github.com/lrochetta/multiai/internal/i18n"
	"github.com/lrochetta/multiai/internal/profile"
	"github.com/lrochetta/multiai/pkg/dotenv"
)

// ShowTopMenu displays the main menu and returns the user's choice.
// The version comes from the caller (single source in main, ldflags-friendly)
// so the title can never drift from `multiai version` again.
func ShowTopMenu(version string, profileCount int) string {
	fmt.Println()
	display.PrintInfo(i18n.T("menu_title", version, profileCount))
	fmt.Println(strings.Repeat("-", 58))
	fmt.Println()
	fmt.Println(i18n.T("menu_launch"))
	fmt.Println(i18n.T("menu_config"))
	fmt.Println(i18n.T("menu_bmad"))
	fmt.Println(i18n.T("menu_models"))
	fmt.Println()
	fmt.Println(i18n.T("menu_quit"))
	fmt.Print(i18n.T("menu_choice"))
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			os.Exit(0)
		}
		fmt.Fprint(os.Stderr, i18n.T("read_error", err))
		return ""
	}
	return strings.TrimSpace(input)
}

// SelectTool lets the user choose a tool from the available ones.
func SelectTool(profiles []profile.Profile) (string, error) {
	// Group by tool, preserving first-appearance order (profiles are already
	// sorted by tool/order) so the numbering is stable across runs â€” a map
	// iteration here would shuffle the menu on every launch.
	type toolEntry struct {
		ID    string
		Label string
		Count int
	}
	var tools []toolEntry
	index := make(map[string]int)
	for _, p := range profiles {
		if i, ok := index[p.Tool]; ok {
			tools[i].Count++
			continue
		}
		index[p.Tool] = len(tools)
		tools = append(tools, toolEntry{ID: p.Tool, Label: p.ToolLabel, Count: 1})
	}

	fmt.Println()
	display.PrintInfo(i18n.T("tools_available"))
	fmt.Println()
	for i, t := range tools {
		fmt.Printf("%d. %s (%d %s)\n", i+1, t.Label, t.Count, i18n.T("profiles_count"))
	}
	fmt.Println()
	fmt.Println(i18n.T("back_main"))
	fmt.Print(i18n.T("choose_tool"))

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			os.Exit(0)
		}
		return "", fmt.Errorf("%s", i18n.T("read_error_profile", err))
	}
	input = strings.TrimSpace(input)

	if input == "0" {
		return "", nil
	}

	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(tools) {
		return "", fmt.Errorf("%s", i18n.T("invalid_choice_lower"))
	}
	return tools[idx-1].ID, nil
}

// SelectProfile lets the user choose a profile for a given tool.
func SelectProfile(profiles []profile.Profile, toolFilter string) (*profile.Profile, error) {
	var filtered []profile.Profile
	for _, p := range profiles {
		if strings.EqualFold(p.Tool, toolFilter) {
			filtered = append(filtered, p)
		}
	}
	if len(filtered) == 0 {
		return nil, fmt.Errorf("%s", i18n.T("no_profile_tool", toolFilter))
	}

	fmt.Println()
	display.PrintInfo(i18n.T("profiles_available", filtered[0].ToolLabel))
	fmt.Println()
	for i, p := range filtered {
		configured, total := countSecrets(&p)
		color := display.StatusColor(configured, total)
		line := fmt.Sprintf("%d. %s [%s]", i+1, p.DisplayName, p.Shortcut)
		if configured > 0 {
			line += fmt.Sprintf(" (%d/%d)", configured, total)
		}
		fmt.Println(display.Colorize(color, line))
		if p.Description != "" {
			fmt.Printf("   %s\n", p.Description)
		}
	}
	fmt.Println()
	fmt.Println(i18n.T("back_tool_sel"))
	fmt.Print(i18n.T("choose_profile"))

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			os.Exit(0)
		}
		return nil, fmt.Errorf("%s", i18n.T("read_error_profile", err))
	}
	input = strings.TrimSpace(input)

	if input == "0" {
		return nil, nil
	}

	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(filtered) {
		return nil, fmt.Errorf("%s", i18n.T("invalid_choice_lower"))
	}
	return &filtered[idx-1], nil
}

// countSecrets counts how many secret keys (non-metadata) are configured
// (non-placeholder) in a profile, and the total number of secret keys.
func countSecrets(p *profile.Profile) (configured, total int) {
	for k, v := range p.Env {
		if profile.MetadataKeys[k] {
			continue
		}
		total++
		if !dotenv.IsPlaceholder(v) {
			configured++
		}
	}
	return
}
