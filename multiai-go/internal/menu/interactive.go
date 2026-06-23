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
)

// ShowTopMenu displays the main menu and returns the user's choice.
func ShowTopMenu(profileCount int) string {
	fmt.Println()
	cli.PrintInfo(fmt.Sprintf("Laurent ROCHETTA's MultiAI (v0.2.1) — %d profils", profileCount))
	fmt.Println(strings.Repeat("─", 58))
	fmt.Println()
	fmt.Println("1. Lancer")
	fmt.Println("2. Configurer les clés API")
	fmt.Println("3. BMAD+ — installer dans un projet")
	fmt.Println()
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
	// Group by tool
	toolMap := make(map[string]struct {
		Label string
		Count int
	})
	for _, p := range profiles {
		t := toolMap[p.Tool]
		t.Label = p.ToolLabel
		t.Count++
		toolMap[p.Tool] = t
	}

	// Get ordered list of tools
	var tools []struct {
		ID    string
		Label string
		Count int
	}
	for id, t := range toolMap {
		tools = append(tools, struct {
			ID    string
			Label string
			Count int
		}{id, t.Label, t.Count})
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
		fmt.Printf("%d. %s [%s]\n", i+1, p.DisplayName, p.Shortcut)
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
