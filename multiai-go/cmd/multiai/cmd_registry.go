// cmd_registry.go wires the community profile registry subcommands
// (profile search, profile install) into the subcommand registry.
//
//	0 success · 1 user error (bad flag, no match) · 3 output failure
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lrochetta/multiai/internal/i18n"
	"github.com/lrochetta/multiai/internal/registry"
)

func init() {
	register("profile", cmdProfile)
}

// cmdProfile is the top-level handler for "multiai profile". It dispatches to
// sub-subcommands: search, install, or help.
func cmdProfile(args []string) int {
	if len(args) == 0 || hasFlag(args, "--help", "-h") {
		printProfileHelp()
		return 0
	}

	// Parse subcommand.
	sub := strings.ToLower(args[0])
	switch sub {
	case "search":
		return cmdProfileSearch(args[1:])
	case "install":
		return cmdProfileInstall(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "%s: %s\n", i18n.T("error"), i18n.T("registry_unknown_subcmd", args[0]))
		printProfileHelp()
		return 1
	}
}

// ── Help ──────────────────────────────────────────────────────────────────────

// printProfileHelp shows the profile subcommand usage on stdout.
func printProfileHelp() {
	fmt.Println(`Usage:
  multiai profile search <query> [options]    Rechercher un profil dans le registre communautaire
  multiai profile install <name> [options]    Installer un profil depuis le registre

Commandes:
  search    Rechercher des profils par nom, description, auteur ou tags
  install   Installer un profil communautaire (telecharger et copier)

Options:
  --json, -j           Sortie JSON
  --help, -h           Cette aide

Exemples:
  multiai profile search deepseek
  multiai profile search claude --json
  multiai profile install ds`)
}

func printProfileSearchHelp() {
	fmt.Println(`Usage:
  multiai profile search <query> [options]   Rechercher un profil

Le terme de recherche est insensible a la casse et cherche dans le nom,
le titre, la description, l'auteur et les tags des profils.

Options:
  --json, -j           Sortie JSON

Exemple:
  multiai profile search deepseek
  multiai profile search claude --json`)
}

func printProfileInstallHelp() {
	fmt.Println(`Usage:
  multiai profile install <name>              Installer un profil communautaire

Telecharge le fichier .env du profil depuis le registre communautaire
et le copie dans le dossier des profils locaux.

Exemple:
  multiai profile install ds`)
}

// ── Search ────────────────────────────────────────────────────────────────────

// profileSearchOptions collects flags for the profile search subcommand.
type profileSearchOptions struct {
	json  bool
	help  bool
	query string
}

// cmdProfileSearch implements "multiai profile search <query>".
func cmdProfileSearch(args []string) int {
	o, err := parseProfileSearchFlags(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T("error"), err)
		return 1
	}
	if o.help {
		printProfileSearchHelp()
		return 0
	}
	if o.query == "" {
		fmt.Fprintf(os.Stderr, "%s\n", i18n.T("registry_search_missing"))
		printProfileSearchHelp()
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	idx, err := registry.FetchIndex(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T("registry_fetch_error"), err)
		return 2
	}

	results := registry.SearchProfiles(idx, o.query)

	if o.json {
		return printProfileSearchJSON(results)
	}

	if len(results) == 0 {
		fmt.Fprintf(os.Stderr, "[!] %s\n", i18n.T("registry_no_results", o.query))
		return 1
	}

	renderProfileResults(results)
	fmt.Println()
	fmt.Print(i18n.T("registry_results_count", len(results)))
	return 0
}

// parseProfileSearchFlags hand-parses search-specific flags.
func parseProfileSearchFlags(args []string) (*profileSearchOptions, error) {
	o := &profileSearchOptions{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json", "-j":
			o.json = true
		case "--help", "-h":
			o.help = true
		default:
			if strings.HasPrefix(args[i], "-") {
				return nil, fmt.Errorf("option inconnue : %s", args[i])
			}
			if o.query == "" {
				o.query = args[i]
			} else {
				o.query += " " + args[i]
			}
		}
	}
	return o, nil
}

// printProfileSearchJSON emits the search results as indented JSON on stdout.
func printProfileSearchJSON(results []registry.ProfileEntry) int {
	if results == nil {
		results = []registry.ProfileEntry{} // JSON [] instead of null
	}
	out := struct {
		Count   int                       `json:"count"`
		Results []registry.ProfileEntry   `json:"results"`
	}{len(results), results}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T("error"), err)
		return 3
	}
	return 0
}

// renderProfileResults prints the search results as a formatted list.
func renderProfileResults(results []registry.ProfileEntry) {
	for _, p := range results {
		stars := renderStars(p.Stars)
		tags := ""
		if len(p.Tags) > 0 {
			tags = " [" + strings.Join(p.Tags, ", ") + "]"
		}
		fmt.Println()
		fmt.Printf("  %s%s\n", p.Name, stars)
		if p.Title != "" {
			fmt.Printf("  %s\n", p.Title)
		}
		if p.Description != "" {
			fmt.Printf("  %s\n", p.Description)
		}
		fmt.Printf("  %s: %s%s\n", i18n.T("registry_author"), p.Author, tags)
	}
}

// renderStars returns a star-rating string for display.
func renderStars(n int) string {
	if n <= 0 {
		return ""
	}
	s := strings.Repeat("★ ", n)
	return " " + strings.TrimRight(s, " ")
}

// ── Install ───────────────────────────────────────────────────────────────────

// cmdProfileInstall implements "multiai profile install <name>".
// It fetches the index, looks up the profile by name, downloads the .env file
// from the registry repo, and copies it into the user's profiles directory.
func cmdProfileInstall(args []string) int {
	if len(args) == 0 || hasFlag(args, "--help", "-h") {
		printProfileInstallHelp()
		return 0
	}
	name := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	idx, err := registry.FetchIndex(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T("registry_fetch_error"), err)
		return 2
	}

	profile := registry.FindProfileByName(idx, name)
	if profile == nil {
		fmt.Fprintf(os.Stderr, "[!] %s\n", i18n.T("registry_profile_not_found", name))
		return 1
	}

	// Build the download URL for the profile's .env file.
	downloadURL := fmt.Sprintf(
		"https://raw.githubusercontent.com/lrochetta/profiles-multiai/main/profiles/%s.env",
		profile.Name,
	)

	fmt.Fprintf(os.Stderr, "[i] %s %s...\n", i18n.T("registry_downloading"), profile.Name)

	// Download the profile .env file.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T("error"), err)
		return 2
	}
	req.Header.Set("User-Agent", "multiai-registry")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[X] %s: %v\n", i18n.T("registry_download_error"), err)
		return 2
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "[X] %s (HTTP %d)\n", i18n.T("registry_download_error"), resp.StatusCode)
		return 2
	}

	// Read the profile content.
	profileData, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[X] %s: %v\n", i18n.T("registry_download_error"), err)
		return 2
	}

	// Resolve profiles directory and write the file.
	profilesDir := getProfilesDir()
	destPath := filepath.Join(profilesDir, profile.Name+".env")
	if err := os.WriteFile(destPath, profileData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T("error"), err)
		return 2
	}

	fmt.Fprintf(os.Stderr, "[OK] %s\n", i18n.T("registry_installed", destPath))
	return 0
}
