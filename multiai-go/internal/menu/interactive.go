package menu

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/lrochetta/multiai/internal/cli"
	"github.com/lrochetta/multiai/internal/profile"
	"github.com/lrochetta/multiai/pkg/dotenv"
)

// ShowTopMenu displays the main menu and returns the user's choice.
// The version comes from the caller (single source in main, ldflags-friendly)
// so the title can never drift from `multiai version` again.
func ShowTopMenu(version string, profileCount int) string {
	fmt.Println()
	cli.PrintInfo(fmt.Sprintf("Laurent ROCHETTA's MultiAI (AI Code CLI Router) v%s - %d profils", version, profileCount))
	fmt.Println(strings.Repeat("-", 58))
	fmt.Println()
	fmt.Println("1. Lancer")
	fmt.Println("2. Configurer les cles API")
	fmt.Println("3. BMAD+ -- Gestion du framework")
	fmt.Println("4. OpenRouter -- Decouvrir les modeles")
	fmt.Println()
	fmt.Println("0. Quitter")
	fmt.Print("Choix : ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "[X] Erreur de lecture: %v\n", err)
		return ""
	}
	return strings.TrimSpace(input)
}

// SelectTool lets the user choose a tool from the available ones.
func SelectTool(profiles []profile.Profile) (string, error) {
	// Group by tool, preserving first-appearance order (profiles are already
	// sorted by tool/order) so the numbering is stable across runs — a map
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
	cli.PrintInfo("Outils disponibles")
	fmt.Println()
	for i, t := range tools {
		fmt.Printf("%d. %s (%d profils)\n", i+1, t.Label, t.Count)
	}
	fmt.Println()
	fmt.Println("0. Retour au menu principal")
	fmt.Print("Choisis un outil : ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			os.Exit(0)
		}
		return "", fmt.Errorf("erreur de lecture: %v", err)
	}
	input = strings.TrimSpace(input)

	if input == "0" {
		return "", nil
	}

	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(tools) {
		return "", fmt.Errorf("choix invalide")
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
		return nil, fmt.Errorf("aucun profil pour l'outil : %s", toolFilter)
	}

	fmt.Println()
	cli.PrintInfo(fmt.Sprintf("Profils disponibles pour %s", filtered[0].ToolLabel))
	fmt.Println()
	for i, p := range filtered {
		configured, total := countSecrets(&p)
		color := cli.StatusColor(configured, total)
		line := fmt.Sprintf("%d. %s [%s]", i+1, p.DisplayName, p.Shortcut)
		if configured > 0 {
			line += fmt.Sprintf(" (%d/%d)", configured, total)
		}
		fmt.Println(cli.Colorize(line, color))
		if p.Description != "" {
			fmt.Printf("   %s\n", p.Description)
		}
	}
	fmt.Println()
	fmt.Println("0. Retour a la selection d'outil")
	fmt.Print("Choisis un profil : ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			os.Exit(0)
		}
		return nil, fmt.Errorf("erreur de lecture: %v", err)
	}
	input = strings.TrimSpace(input)

	if input == "0" {
		return nil, nil
	}

	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(filtered) {
		return nil, fmt.Errorf("choix invalide")
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
