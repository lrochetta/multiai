// cmd_registry.go wires the community profile registry subcommands
// (profile search, profile list --remote, profile install) into the
// subcommand registry.
//
//	0 success · 1 user error (bad flag, no match) · 3 output failure
package main

import (
	"context"
	"encoding/json"
	"fmt"
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
// sub-subcommands: search, list, install, or help.
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
	case "list":
		return cmdProfileList(args[1:])
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
  multiai profile list [options]              Lister les profils (locaux ou --remote)
  multiai profile install <name> [options]    Installer un profil depuis le registre

Commandes:
  search    Rechercher des profils par nom, description, auteur ou tags
  list      Afficher les profils installes (--remote pour le registre)
  install   Installer un profil communautaire (telecharger et copier)

Options:
  --json, -j           Sortie JSON
  --remote             Lister les profils du registre (profile list)
  --force              Ecraser un profil existant (profile install)
  --no-verify          Sauter la verification SHA-256 (profile install)
  --help, -h           Cette aide

Exemples:
  multiai profile search deepseek
  multiai profile search claude --json
  multiai profile list --remote
  multiai profile install ds
  multiai profile install ds --force`)
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

func printProfileListHelp() {
	fmt.Println(`Usage:
  multiai profile list [options]             Lister les profils

Par defaut, liste les profils installes localement.
Avec --remote, liste les profils disponibles dans le registre communautaire.

Options:
  --json, -j           Sortie JSON
  --remote, -r         Lister les profils du registre communautaire

Exemple:
  multiai profile list
  multiai profile list --remote
  multiai profile list --remote --json`)
}

func printProfileInstallHelp() {
	fmt.Println(`Usage:
  multiai profile install <name> [options]   Installer un profil communautaire

Telecharge le fichier .env du profil depuis le registre communautaire,
verifie l'empreinte SHA-256 (si disponible), et le copie dans le dossier
des profils locaux.

Options:
  --force              Ecraser si le profil existe deja
  --no-verify          Sauter la verification SHA-256

Exemple:
  multiai profile install ds
  multiai profile install ds --force`)
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

// ── List ──────────────────────────────────────────────────────────────────────

// profileListOptions collects flags for the profile list subcommand.
type profileListOptions struct {
	json   bool
	remote bool
	help   bool
}

// cmdProfileList implements "multiai profile list" and "multiai profile list --remote".
func cmdProfileList(args []string) int {
	o, err := parseProfileListFlags(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T("error"), err)
		return 1
	}
	if o.help {
		printProfileListHelp()
		return 0
	}

	if o.remote {
		return listRemoteProfiles(o.json)
	}
	return listLocalProfiles(o.json)
}

// parseProfileListFlags hand-parses list-specific flags.
func parseProfileListFlags(args []string) (*profileListOptions, error) {
	o := &profileListOptions{}
	for _, a := range args {
		switch a {
		case "--json", "-j":
			o.json = true
		case "--remote", "-r":
			o.remote = true
		case "--help", "-h":
			o.help = true
		default:
			if strings.HasPrefix(a, "-") {
				return nil, fmt.Errorf("option inconnue : %s", a)
			}
			return nil, fmt.Errorf("argument inattendu : %s", a)
		}
	}
	return o, nil
}

// listLocalProfiles lists the .env profiles found in the local profiles directory.
func listLocalProfiles(jsonOut bool) int {
	profilesDir := getProfilesDir()
	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T("error"), err)
		return 1
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".env") {
			files = append(files, strings.TrimSuffix(e.Name(), ".env"))
		}
	}

	if jsonOut {
		out := struct {
			Count   int      `json:"count"`
			Dir     string   `json:"directory"`
			Profiles []string `json:"profiles"`
		}{len(files), profilesDir, files}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(out); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T("error"), err)
			return 3
		}
		return 0
	}

	if len(files) == 0 {
		fmt.Println(i18n.T("registry_list_remote_empty"))
		return 0
	}
	fmt.Printf("%s (%s)\n", i18n.T("registry_list_remote"), profilesDir)
	for _, f := range files {
		fmt.Printf("  %s\n", f)
	}
	fmt.Printf("\n%d profil(s) installe(s) localement\n", len(files))
	return 0
}

// listRemoteProfiles fetches the remote index and prints all available profiles.
func listRemoteProfiles(jsonOut bool) int {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	idx, err := registry.FetchIndexNoCache(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T("registry_fetch_error"), err)
		return 2
	}

	if jsonOut {
		return printProfileListJSON(idx.Profiles)
	}

	if len(idx.Profiles) == 0 {
		fmt.Println(i18n.T("registry_list_remote_empty"))
		return 0
	}

	fmt.Println(i18n.T("registry_list_remote"))
	renderProfileResults(idx.Profiles)
	fmt.Println()
	fmt.Print(i18n.T("registry_results_count", len(idx.Profiles)))
	return 0
}

// printProfileListJSON emits the profile list as indented JSON on stdout.
func printProfileListJSON(profiles []registry.ProfileEntry) int {
	if profiles == nil {
		profiles = []registry.ProfileEntry{}
	}
	out := struct {
		Count   int                       `json:"count"`
		Results []registry.ProfileEntry   `json:"results"`
	}{len(profiles), profiles}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T("error"), err)
		return 3
	}
	return 0
}

// ── Search / list output helpers ──────────────────────────────────────────────

// printProfileSearchJSON emits the search results as indented JSON on stdout.
func printProfileSearchJSON(results []registry.ProfileEntry) int {
	if results == nil {
		results = []registry.ProfileEntry{} // JSON [] instead of null
	}
	out := struct {
		Count   int                     `json:"count"`
		Results []registry.ProfileEntry `json:"results"`
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

// profileInstallOptions collects flags for the profile install subcommand.
type profileInstallOptions struct {
	force    bool
	noVerify bool
	help     bool
	name     string
}

// cmdProfileInstall implements "multiai profile install <name>".
// It fetches the index, looks up the profile by name, downloads the .env file
// from the registry repo, verifies SHA-256 (when available), checks for
// existing files, and writes it atomically into the profiles directory.
func cmdProfileInstall(args []string) int {
	o, err := parseProfileInstallFlags(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T("error"), err)
		return 1
	}
	if o.help {
		printProfileInstallHelp()
		return 0
	}
	if o.name == "" {
		fmt.Fprintf(os.Stderr, "%s: %s\n", i18n.T("error"), i18n.T("registry_search_missing"))
		printProfileInstallHelp()
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	idx, err := registry.FetchIndex(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T("registry_fetch_error"), err)
		return 2
	}

	profile := registry.FindProfileByName(idx, o.name)
	if profile == nil {
		fmt.Fprintf(os.Stderr, "[!] %s\n", i18n.T("registry_profile_not_found", o.name))
		return 1
	}

	// Check for existing profile file.
	profilesDir := getProfilesDir()
	destPath := filepath.Join(profilesDir, profile.Name+".env")
	if _, err := os.Stat(destPath); err == nil && !o.force {
		fmt.Fprintf(os.Stderr, "[!] %s\n", i18n.T("registry_install_exists", profile.Name, destPath))
		return 1
	}

	// Clear SHA256 when --no-verify is set so the installer skips verification.
	if o.noVerify {
		profile.SHA256 = ""
	}

	fmt.Fprintf(os.Stderr, "[i] %s %s...\n", i18n.T("registry_downloading"), profile.Name)

	if profile.SHA256 != "" {
		fmt.Fprintf(os.Stderr, "[i] %s\n", i18n.T("registry_checksum_verify"))
	}

	installedPath, err := registry.InstallProfile(ctx, profile, profilesDir)
	if err != nil {
		if registry.IsChecksumError(err) {
			fmt.Fprintf(os.Stderr, "[X] %s\n", i18n.T("registry_checksum_error"))
			return 2
		}
		fmt.Fprintf(os.Stderr, "[X] %s\n", i18n.T("registry_download_error"))
		return 2
	}

	fmt.Fprintf(os.Stderr, "[OK] %s\n", i18n.T("registry_installed", installedPath))
	return 0
}

// parseProfileInstallFlags hand-parses install-specific flags.
func parseProfileInstallFlags(args []string) (*profileInstallOptions, error) {
	o := &profileInstallOptions{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--force", "-f":
			o.force = true
		case "--no-verify":
			o.noVerify = true
		case "--help", "-h":
			o.help = true
		default:
			if strings.HasPrefix(args[i], "-") {
				return nil, fmt.Errorf("option inconnue : %s", args[i])
			}
			if o.name == "" {
				o.name = args[i]
			} else {
				return nil, fmt.Errorf("argument inattendu : %s", args[i])
			}
		}
	}
	return o, nil
}
