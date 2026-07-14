// multiai is the main entry point for the multiai CLI.
// It routes AI CLI commands (Claude Code, Codex, OpenCode) with
// isolated environment profiles.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lrochetta/multiai/internal/assets"
	"github.com/lrochetta/multiai/internal/catalog"
	"github.com/lrochetta/multiai/internal/cli"
	"github.com/lrochetta/multiai/internal/config"
	"github.com/lrochetta/multiai/internal/i18n"
	"github.com/lrochetta/multiai/internal/menu"
	"github.com/lrochetta/multiai/internal/onboarding"
	"github.com/lrochetta/multiai/internal/openrouter"
	"github.com/lrochetta/multiai/internal/profile"
	"github.com/lrochetta/multiai/internal/secret"
	"github.com/lrochetta/multiai/internal/update"
)

// version is the single source of truth for the CLI version (also shown in
// the interactive menu title). Release builds override it with
// `-ldflags "-X main.version=X.Y.Z"` (goreleaser).
var version = "0.6.0"

// commands is the subcommand registry. Feature files (cmd_*.go) contribute
// commands from an init() via register(), so main.go stays free of merge
// hotspots when several features land in parallel.
var commands = map[string]func(args []string) int{}

func register(name string, fn func(args []string) int) { commands[name] = fn }

// getProfilesDir resolves the profiles directory, in priority order:
//  1. MULTIAI_PROFILES_DIR environment variable (tests, portable setups)
//  2. <executable dir>/configs/profiles when present (zip installs)
//  3. ./configs/profiles when present AND MULTIAI_DEV is set (dev mode only,
//     opt-in: loading profiles from an arbitrary CWD is an attack surface)
//  4. <user config dir>/multiai/profiles, created and seeded from the
//     embedded templates on first run (go install / npm binary).
func getProfilesDir() string {
	if dir := os.Getenv("MULTIAI_PROFILES_DIR"); dir != "" {
		ensureProfiles(dir)
		return dir
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Join(filepath.Dir(exe), "configs", "profiles")
		if isDir(dir) {
			return dir
		}
	}
	// CWD-relative profiles are OPT-IN only (MULTIAI_DEV). Honouring them
	// unconditionally is an attack surface: a hostile directory could ship
	// ./configs/profiles that shadows the user's real profiles and, via the
	// credential-store service namespace (basename-derived), exfiltrate their
	// stored secrets. The PowerShell reference never reads the CWD.
	if os.Getenv("MULTIAI_DEV") != "" {
		if dir := filepath.Join("configs", "profiles"); isDir(dir) {
			return dir
		}
	}
	cfg, err := os.UserConfigDir()
	if err != nil {
		// No user config dir available: fall back to the dev-mode path so
		// the caller reports a readable "cannot read profiles directory".
		return filepath.Join("configs", "profiles")
	}
	dir := filepath.Join(cfg, "multiai", "profiles")
	ensureProfiles(dir)
	return dir
}

// ensureProfiles seeds dir with the embedded profile templates, using the
// profiles manifest to only extract new or unknown profiles.  User-modified
// files are never overwritten (a warning is emitted instead).
// On fresh installs all profiles are extracted; on upgrades only the newly
// added ones are materialised.
func ensureProfiles(dir string) {
	installed, err := assets.ReadInstalledManifest(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T("warning_manifest"), err)
		// Fall back to legacy behaviour.
		if hasEnvFiles(dir) {
			return
		}
	}
	n, err := assets.ExtractMissingProfiles(dir, installed)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T("warning_install_profiles", dir), err)
		return
	}
	if n > 0 {
		fmt.Fprintf(os.Stderr, "%s\n", i18n.T("profiles_installed", dir, n))
	}
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func hasEnvFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".env") {
			return true
		}
	}
	return false
}

func printHelp() {
	fmt.Println(`multiai -- Routeur multi-IA (Claude Code, Codex CLI, OpenCode)

Usage:
  multiai                         Menu interactif
  multiai launch                  Menu de lancement
  multiai launch -p <profile>     Lancement direct
  multiai launch -p ds --json     Lancement + sortie JSON
  multiai launch -p ds --dry-run  Simulation sans lancer
  multiai list                    Liste des profils
  multiai list --json             Liste en JSON
  multiai config                  Configurer les cles API
  multiai config --provider <id>  Configurer un fournisseur specifique
  multiai models                  Modeles OpenRouter (top, --offline: cache)
  multiai search <terme>          Rechercher un modele OpenRouter
  multiai compare <slug> ...      Comparer des modeles OpenRouter
  multiai bmad                    Gestion BMAD+ (install/update via npx)
  multiai completion <shell>      Completion shell (bash/zsh/fish/powershell)
  multiai update                  Verifier et installer les mises a jour
  multiai project status          Inspecter la confiance de .multiai.yaml
  multiai project trust           Approuver le fichier courant et son SHA256
  multiai project untrust         Revoquer cette approbation
  multiai profile search <terme>  Rechercher un profil communautaire
  multiai profile install <nom>   Installer un profil communautaire
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

Fournisseurs disponibles:`)
	// Data-driven from the embedded catalog so this list can never go stale.
	for _, prov := range catalog.Default().Providers {
		fmt.Printf("  %-14s %s\n", prov.ID, prov.Display)
	}
}

func printBmadHelp() {
	fmt.Println(`Usage:
  multiai bmad    Gestion BMAD+ dans le dossier courant :
                  detection de l'installation (_bmad/, package.json, .agents/),
                  installation / mise a jour via npx bmad-plus
                  (confirmation demandee avant chaque execution)

Necessite Node.js (npx) : https://nodejs.org`)
}

func main() {
	// Auto-update is notification-only: the bounded, cached background check
	// never downloads, installs, executes a replacement, or exits this process.
	go update.Check(context.Background(), version)

	if len(os.Args) < 2 {
		runInteractiveLoop()
		return
	}

	// Handle subcommand --help. Registered commands (models, search,
	// compare, ...) are NOT intercepted: they own their --help output.
	if len(os.Args) >= 3 && os.Args[2] == "--help" {
		switch os.Args[1] {
		case "launch":
			printLaunchHelp()
			os.Exit(0)
		case "list":
			printListHelp()
			os.Exit(0)
		case "config":
			printConfigHelp()
			os.Exit(0)
		case "bmad":
			printBmadHelp()
			os.Exit(0)
		default:
			if _, ok := commands[os.Args[1]]; !ok {
				printHelp()
				os.Exit(0)
			}
		}
	}

	switch os.Args[1] {
	case "version", "--version", "-V":
		fmt.Printf("multiai %s\n", version)
		return

	case "help", "--help", "-h":
		printHelp()
		return

	case "list":
		profiles, err := profile.LoadAllProfiles(getProfilesDir())
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T("error"), err)
			os.Exit(2)
		}
		asJSON := hasFlag(os.Args, "--json", "-j")
		if err := cli.ListProfiles(profiles, asJSON); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T("error"), err)
			os.Exit(1)
		}

	case "launch":
		profilesDir := getProfilesDir()
		profiles, err := profile.LoadAllProfiles(profilesDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T("error"), err)
			os.Exit(2)
		}
		result, err := runLaunch(profiles)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T("error"), err)
			os.Exit(1)
		}
		if result != nil && result.ExitCode != 0 {
			os.Exit(result.ExitCode)
		}

	case "config":
		profiles, err := profile.LoadAllProfiles(getProfilesDir())
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T("error"), err)
			os.Exit(2)
		}
		store, err := handleStoreFlag(os.Args)
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(2)
		}
		// Auto-migrate from file store to native store when --store is set to
		// a native backend (wincred, keychain, secret-service).
		storeBackend := getFlagValue(os.Args, "--store")
		if store != nil && storeBackend != "" && storeBackend != "file" && storeBackend != "auto" {
			migrateForce := hasFlag(os.Args, "--migrate-force")
			report, mErr := secret.MigrateFromFileStore(store, migrateForce)
			if mErr != nil {
				fmt.Fprintf(os.Stderr, "[!] %s: %v\n", i18n.T("warning"), mErr)
			} else if report.AlreadyMigrated {
				fmt.Fprintf(os.Stderr, "%s\n", i18n.T("store_already_migrated"))
			} else if len(report.Failed) > 0 {
				fmt.Fprintf(os.Stderr, "[!] %s\n", i18n.T("store_migrate_failed", len(report.Failed)))
			} else if len(report.Migrated) > 0 {
				fmt.Fprintf(os.Stderr, "%s\n", i18n.T("store_migrated", len(report.Migrated)))
			} else if len(report.ServicesFound) > 0 && len(report.Migrated) == 0 {
				fmt.Fprintf(os.Stderr, "%s\n", i18n.T("store_migrate_skip", len(report.Skipped)))
			}
		}
		if hasFlag(os.Args, "--provider") {
			providerID := getFlagValue(os.Args, "--provider")
			if providerID == "" {
				fmt.Fprintln(os.Stderr, i18n.T("provider_flag_expected"))
				fmt.Println()
				printConfigHelp()
				os.Exit(1)
			}
			if err := config.ConfigureProviderByID(profiles, providerID, bufio.NewReader(os.Stdin), store); err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T("error"), err)
				os.Exit(1)
			}
		} else if err := config.InteractiveConfig(profiles, store); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T("error"), err)
			os.Exit(1)
		}

	case "bmad":
		// One-shot parity with the PowerShell -Bmad flag: show the menu,
		// run the chosen npx action, exit 0.
		menu.ShowBmadMenu()

	case "completion":
		shell := "bash"
		if len(os.Args) > 2 {
			shell = os.Args[2]
		}
		if err := cli.GenerateCompletion(shell); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T("error"), err)
			os.Exit(1)
		}

	default:
		if fn, ok := commands[os.Args[1]]; ok {
			os.Exit(fn(os.Args[2:]))
		}
		fmt.Fprintf(os.Stderr, "%s", i18n.T("unknown_cmd", os.Args[1]))
		printHelp()
		os.Exit(1)
	}
}

func runInteractiveLoop() {
	// First-run onboarding: interactive mode only, shown at most once.
	// An existing marker file is respected even if all keys were erased
	// later (the wizard never nags twice).
	if profiles, err := profile.LoadAllProfiles(getProfilesDir()); err == nil {
		if onboarding.IsFirstRun(profiles) && !onboarding.FirstRunMarkerExists() {
			onboarding.RunWelcome(profiles)
		}
	}

	for {
		profiles, err := profile.LoadAllProfiles(getProfilesDir())
		if err != nil {
			fmt.Fprintf(os.Stderr, "[X] %v\n", err)
			os.Exit(2)
		}
		choice := menu.ShowTopMenu(version, len(profiles))
		switch choice {
		case "1":
			result, err := runLaunch(profiles)
			if err != nil {
				cli.PrintError(fmt.Sprintf("%v", err))
			} else if result != nil && result.ExitCode != 0 {
				cli.PrintWarning(fmt.Sprintf(i18n.T("process_exit_code"), result.ExitCode))
			}
			fmt.Println()
		case "2":
			if err := config.InteractiveConfig(profiles, nil); err != nil {
				cli.PrintError(fmt.Sprintf("%v", err))
			}
			fmt.Println()
		case "3":
			menu.ShowBmadMenu()
			fmt.Println()
		case "4":
			if err := openrouter.InteractiveMenu(); err != nil {
				cli.PrintError(fmt.Sprintf("OpenRouter : %v", err))
			}
			fmt.Println()
		case "0", "q", "quit", "exit":
			return
		default:
			fmt.Println(i18n.T("menu_invalid"))
		}
	}
}

// runLaunch returns (nil, nil) when the user navigates back, and a non-nil
// error on real failures so callers can exit non-zero (v0.2.1 finding #7).
func runLaunch(profiles []profile.Profile) (*cli.LaunchResult, error) {
	var selected *profile.Profile

	// Check for -p / --profile flag
	profileName := getFlagValue(os.Args, "-p", "--profile")
	// Check for -t / --tool flag
	toolName := getFlagValue(os.Args, "-t", "--tool")

	if profileName != "" {
		var err error
		selected, err = profile.FindByShortcut(profiles, profileName)
		if err != nil {
			return nil, err
		}
	} else if toolName != "" {
		var err error
		selected, err = menu.SelectProfile(profiles, toolName)
		if err != nil {
			return nil, err
		}
		if selected == nil {
			return nil, nil
		}
	} else {
		for {
			tool, err := menu.SelectTool(profiles)
			if err != nil {
				return nil, err
			}
			if tool == "" {
				return nil, nil // back to menu
			}

			selected, err = menu.SelectProfile(profiles, tool)
			if err != nil {
				return nil, err
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

	// Project config is fail-closed. A repository-controlled file is never
	// applied until its canonical path and exact bytes were explicitly trusted.
	projectConfig, projectPath, err := profile.FindProjectConfig()
	if err != nil {
		return nil, fmt.Errorf("configuration projet bloquee (%s): %w; inspectez-la avec 'multiai project status', puis approuvez-la avec 'multiai project trust'", projectPath, err)
	}
	if projectConfig != nil {
		merged := profile.MergeProjectConfig(selected, projectConfig)
		selected = merged
	}

	// Transmit hooks for before/after launch execution
	if os.Getenv("MULTIAI_NO_HOOKS") != "1" {
		opts.Hooks = selected.Hooks
	}

	result, err := cli.LaunchWithFallback(profiles, selected, opts)
	if err != nil {
		return nil, err
	}

	if opts.JSON && result != nil {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(result)
	}

	return result, nil
}

// --- Helper functions ---

// handleStoreFlag checks the --store flag in args and returns:
//   - (nil, nil) if --store is absent (no-op)
//   - (store, nil) if the backend was created successfully
//   - (nil, err) if the backend name is invalid or creation failed
func handleStoreFlag(args []string) (secret.Store, error) {
	storeFlag := getFlagValue(args, "--store")
	if storeFlag == "" {
		return nil, nil
	}
	validBackends := []string{"wincred", "keychain", "secret-service", "file", "auto"}
	isValid := false
	for _, b := range validBackends {
		if storeFlag == b {
			isValid = true
			break
		}
	}
	if !isValid {
		return nil, fmt.Errorf("%s", i18n.T("store_invalid_backend", storeFlag, strings.Join(validBackends, ", ")))
	}
	store, err := secret.NewStoreWithBackend(storeFlag)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("store_init_error", storeFlag, err))
	}
	return store, nil
}

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
	var val string
	for i, arg := range args {
		for _, flag := range flags {
			if arg == flag && i+1 < len(args) {
				val = args[i+1]
			}
		}
	}
	return val
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
