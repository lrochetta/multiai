// multiai is the main entry point for the multiai CLI.
// It routes AI CLI commands (Claude Code, Codex, OpenCode) with
// isolated environment profiles.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lrochetta/multiai/internal/cli"
	"github.com/lrochetta/multiai/internal/config"
	"github.com/lrochetta/multiai/internal/menu"
	"github.com/lrochetta/multiai/internal/profile"
)

const version = "0.2.1"

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

func printLaunchHelp() {
	fmt.Println(`Usage:
  multiai launch                     Menu de lancement interactif
  multiai launch -p <shortcut>       Lancement direct d'un profil
  multiai launch -t <tool>           Selection interactive du profil pour un outil
  multiai launch -p ds --json        Lancement avec sortie JSON
  multiai launch -p ds --dry-run     Simulation sans lancer
  multiai launch -p ds --show-env    Afficher les variables d'environnement

Options:
  -p, --profile <shortcut>    Profil a lancer (ex: ds, ca, ocqwen)
  -t, --tool <tool>           Outil a utiliser (ex: claude, codex, opencode)
  --json, -j                  Sortie au format JSON
  --dry-run                   Simulation sans lancement
  --no-launch                 Preparation sans lancement
  --show-env, --env           Afficher les variables d'environnement
  --allow-custom-command      Autoriser les commandes personnalisees
  --                          Arguments supplementaires passes au CLI`)
}

func printListHelp() {
	fmt.Println(`Usage:
  multiai list                  Liste tous les profils disponibles
  multiai list --json           Liste au format JSON

Affiche la liste des profils configures avec leur outil, shortcut et commande.`)
}

func printConfigHelp() {
	fmt.Println(`Usage:
  multiai config                              Configuration interactive des cles API
  multiai config --provider <id>              Configurer un fournisseur specifique

Fournisseurs disponibles:
  anthropic   Anthropic (officiel)
  zai         Z.ai / BigModel
  deepseek    DeepSeek
  openai      OpenAI
  openrouter  OpenRouter`)
}

func main() {
	if len(os.Args) < 2 {
		runInteractiveLoop()
		return
	}

	// Handle subcommand --help
	if len(os.Args) >= 3 && os.Args[2] == "--help" {
		switch os.Args[1] {
		case "launch":
			printLaunchHelp()
		case "list":
			printListHelp()
		case "config":
			printConfigHelp()
		default:
			printHelp()
		}
		os.Exit(0)
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
		result := runLaunch(profiles)
		if result != nil && result.ExitCode != 0 {
			os.Exit(result.ExitCode)
		}

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

func runInteractiveLoop() {
	for {
		profiles, err := profile.LoadDir(getProfilesDir())
		if err != nil {
			fmt.Fprintf(os.Stderr, "[X] %v\n", err)
			os.Exit(2)
		}
		choice := menu.ShowTopMenu(len(profiles))
		switch choice {
		case "1":
			result := runLaunch(profiles)
			if result != nil && result.ExitCode != 0 {
				cli.PrintWarning(fmt.Sprintf("Le processus s'est termine avec le code: %d", result.ExitCode))
			}
			fmt.Println()
		case "2":
			if err := config.InteractiveConfig(profiles); err != nil {
				fmt.Fprintf(os.Stderr, "[X] %v\n", err)
			}
			fmt.Println()
		case "3":
			fmt.Println("\nBMAD+ -- lancement de npx bmad-plus install...")
			fmt.Println("(BMAD+ n'est pas encore integre dans la version Go)")
			fmt.Println()
		default:
			fmt.Println("Choix invalide. Options : 1, 2, 3")
		}
	}
}

func runLaunch(profiles []profile.Profile) *cli.LaunchResult {
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
			return nil
		}
	} else if toolName != "" {
		var err error
		selected, err = menu.SelectProfile(profiles, toolName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
			return nil
		}
		if selected == nil {
			return nil
		}
	} else {
		for {
			tool, err := menu.SelectTool(profiles)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
				return nil
			}
			if tool == "" {
				return nil // back to menu
			}

			selected, err = menu.SelectProfile(profiles, tool)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
				return nil
			}
			if selected != nil {
				break
			}
			// selected == nil -> back to tool selection, continue loop
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
		return nil
	}

	if opts.JSON && result != nil {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(result)
	}

	return result
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
