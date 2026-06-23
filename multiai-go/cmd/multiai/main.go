// multiai is the main entry point for the multiai CLI.
// It routes AI CLI commands (Claude Code, Codex, OpenCode) with
// isolated environment profiles.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lrochetta/multiai/internal/cli"
	"github.com/lrochetta/multiai/internal/config"
	"github.com/lrochetta/multiai/internal/menu"
	"github.com/lrochetta/multiai/internal/profile"
)

const version = "0.2.0"

// getProfilesDir returns the path to the profiles directory.
func getProfilesDir() string {
	// Try relative to executable first
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Join(filepath.Dir(exe), "configs", "profiles")
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	// Fallback: relative to working directory
	dir := filepath.Join("configs", "profiles")
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		return dir
	}
	// Last resort
	return dir
}

func printHelp() {
	fmt.Println(`multiai — Routeur multi-IA (Claude Code, Codex CLI, OpenCode)

Usage:
  multiai                         Menu interactif
  multiai launch                  Menu de lancement
  multiai launch -p <profile>     Lancement direct
  multiai launch -p ds --json     Lancement + sortie JSON
  multiai launch -p ds --dry-run  Simulation sans lancer
  multiai list                    Liste des profils
  multiai list --json             Liste en JSON
  multiai config                  Configurer les clés API
  multiai config --provider <id>  Configurer un fournisseur specifique
  multiai completion <shell>      Completion shell (bash/zsh/fish/powershell)
  multiai version                 Afficher la version
  multiai help                    Cette aide

Exemples:
  multiai launch -p ds
  multiai launch -p codex55 -- --dangerously-skip-permissions
  multiai list --json | jq .

Code: https://github.com/lrochetta/multiai`)
}

func main() {
	if len(os.Args) < 2 {
		runInteractive()
		return
	}

	switch os.Args[1] {
	case "version", "--version", "-V":
		fmt.Printf("multiai %s\n", version)

	case "help", "--help", "-h":
		printHelp()

	case "list":
		profiles, err := profile.LoadDir(getProfilesDir())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
			os.Exit(2)
		}
		asJSON := hasFlag(os.Args, "--json", "-j")
		if err := cli.ListProfiles(profiles, asJSON); err != nil {
			fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
			os.Exit(1)
		}

	case "launch":
		profilesDir := getProfilesDir()
		profiles, err := profile.LoadDir(profilesDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
			os.Exit(2)
		}
		runLaunch(profiles)

	case "config":
		profiles, err := profile.LoadDir(getProfilesDir())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
			os.Exit(2)
		}
		if err := config.InteractiveConfig(profiles); err != nil {
			fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
			os.Exit(1)
		}

	case "completion":
		shell := "bash"
		if len(os.Args) > 2 {
			shell = os.Args[2]
		}
		if err := cli.GenerateCompletion(shell); err != nil {
			fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "Commande inconnue : %s\n", os.Args[1])
		printHelp()
		os.Exit(1)
	}
}

func runInteractive() {
	profiles, err := profile.LoadDir(getProfilesDir())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
		os.Exit(2)
	}

	for {
		choice := menu.ShowTopMenu()
		switch choice {
		case "1":
			runLaunch(profiles)
			return
		case "2":
			if err := config.InteractiveConfig(profiles); err != nil {
				fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
			}
			return
		case "3":
			fmt.Println("\nBMAD+ — lancement de npx bmad-plus install...")
			fmt.Println("(non implemente dans la version Go — utilise le script shell)")
			return
		default:
			fmt.Println("Choix invalide.")
		}
	}
}

func runLaunch(profiles []profile.Profile) {
	var selected *profile.Profile

	// Check for -p / --profile flag
	profileName := getFlagValue(os.Args, "-p", "--profile")
	// Check for -t / --tool flag
	toolName := getFlagValue(os.Args, "-t", "--tool")

	if profileName != "" {
		var err error
		selected, err = profile.FindByShortcut(profiles, profileName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
			os.Exit(1)
		}
	} else if toolName != "" {
		var err error
		selected, err = menu.SelectProfile(profiles, toolName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
			os.Exit(1)
		}
	} else {
		tool, err := menu.SelectTool(profiles)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
			os.Exit(1)
		}
		selected, err = menu.SelectProfile(profiles, tool)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
			os.Exit(1)
		}
	}

	// Parse extra args (everything after --)
	extraArgs := getExtraArgs(os.Args)

	opts := cli.LaunchOptions{
		DryRun:             hasFlag(os.Args, "--dry-run"),
		NoLaunch:           hasFlag(os.Args, "--no-launch"),
		ShowEnv:            hasFlag(os.Args, "--show-env", "--env"),
		JSON:               hasFlag(os.Args, "--json", "-j"),
		AllowCustomCommand: hasFlag(os.Args, "--allow-custom-command"),
		ExtraArgs:          extraArgs,
	}

	result, err := cli.ValidateAndLaunch(selected, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
		os.Exit(1)
	}

	if opts.JSON && result != nil {
		fmt.Printf("{\"status\": \"%s\"}\n", result.Status)
	}
	if result != nil && result.PID > 0 {
		os.Exit(0)
	}
}

// --- Helper functions ---

func hasFlag(args []string, flags ...string) bool {
	for _, arg := range args {
		for _, flag := range flags {
			if arg == flag {
				return true
			}
		}
	}
	return false
}

func getFlagValue(args []string, flags ...string) string {
	for i, arg := range args {
		for _, flag := range flags {
			if arg == flag && i+1 < len(args) {
				return args[i+1]
			}
		}
	}
	return ""
}

func getExtraArgs(args []string) []string {
	for i, arg := range args {
		if arg == "--" {
			if i+1 < len(args) {
				return args[i+1:]
			}
		}
	}
	return nil
}
