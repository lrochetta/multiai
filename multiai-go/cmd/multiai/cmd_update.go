// cmd_update.go implements the "multiai update" subcommand for explicit
// self-update checking and installation, complementing the silent auto-update
// in main.go with an interactive / CI-friendly flow.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lrochetta/multiai/internal/update"
)

func init() {
	register("update", cmdUpdate)
}

// updateOptions collects flags for the update subcommand.
type updateOptions struct {
	check bool
	yes   bool
	help  bool
}

// parseUpdateFlags hand-parses update-specific flags.
func parseUpdateFlags(args []string) (*updateOptions, error) {
	o := &updateOptions{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--check":
			o.check = true
		case "--yes", "-y":
			o.yes = true
		case "--help", "-h":
			o.help = true
		default:
			if strings.HasPrefix(args[i], "-") {
				return nil, fmt.Errorf("option inconnue : %s", args[i])
			}
			return nil, fmt.Errorf("argument inattendu : %s", args[i])
		}
	}
	return o, nil
}

// printUpdateHelp shows the update subcommand usage on stdout.
func printUpdateHelp() {
	fmt.Println(`Usage:
  multiai update [options]              Verifier et installer les mises a jour

Options:
  --check              Verifier la disponibilite (sortie JSON, sans installation)
  --yes, -y            Installation sans confirmation (mode CI)

Exemples:
  multiai update
  multiai update --check
  multiai update --yes`)
}

// cmdUpdate is the subcommand handler registered as "update".
func cmdUpdate(args []string) int {
	o, err := parseUpdateFlags(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
		return 1
	}
	if o.help {
		printUpdateHelp()
		return 0
	}

	// Fetch latest release from GitHub.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rel, err := update.FetchLatestRelease(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[X] Impossible de verifier les mises a jour : %v\n", err)
		return 2
	}

	hasUpdate := update.IsNewer(version, rel.TagName)

	// --check: JSON output, no installation.
	if o.check {
		out := struct {
			CurrentVersion string `json:"current_version"`
			LatestVersion  string `json:"latest_version"`
			HasUpdate      bool   `json:"has_update"`
		}{
			CurrentVersion: version,
			LatestVersion:  strings.TrimPrefix(rel.TagName, "v"),
			HasUpdate:      hasUpdate,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(out); err != nil {
			fmt.Fprintf(os.Stderr, "[X] Erreur de sortie JSON : %v\n", err)
			return 3
		}
		return 0
	}

	if !hasUpdate {
		fmt.Println("[i] Deja a jour")
		return 0
	}

	// Interactive / --yes path.
	fmt.Printf("Version actuelle   : %s\n", version)
	fmt.Printf("Version disponible : %s\n", strings.TrimPrefix(rel.TagName, "v"))

	if !o.yes {
		fmt.Print("Voulez-vous installer la mise a jour ? [O/n] ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer == "n" || answer == "non" {
			fmt.Println("Mise a jour annulee.")
			return 0
		}
	}

	fmt.Println("Telechargement et installation...")
	newExe, err := update.DownloadRelease(ctx, version, rel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[X] Erreur lors du telechargement : %v\n", err)
		return 2
	}

	fmt.Println("[OK] Mise a jour telechargee. Redemarrage...")
	update.ExecBinary(newExe)
	// Never reaches here (ExecBinary calls os.Exit(0)).
	return 0
}
