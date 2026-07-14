// cmd_update.go implements explicit, package-manager-owned updates.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/lrochetta/multiai/internal/update"
)

const (
	updateCheckTimeout   = 15 * time.Second
	updateInstallTimeout = 5 * time.Minute
)

var (
	fetchUpdateRelease   = update.FetchLatestRelease
	installUpdateRelease = update.InstallRelease
)

func init() {
	register("update", cmdUpdate)
}

type updateOptions struct {
	check bool
	yes   bool
	help  bool
}

func parseUpdateFlags(args []string) (*updateOptions, error) {
	options := &updateOptions{}
	for _, arg := range args {
		switch arg {
		case "--check":
			options.check = true
		case "--yes", "-y":
			options.yes = true
		case "--help", "-h":
			options.help = true
		default:
			if strings.HasPrefix(arg, "-") {
				return nil, fmt.Errorf("option inconnue : %s", arg)
			}
			return nil, fmt.Errorf("argument inattendu : %s", arg)
		}
	}
	return options, nil
}

func printUpdateHelp() {
	printUpdateHelpTo(os.Stdout)
}

func printUpdateHelpTo(output io.Writer) {
	fmt.Fprintln(output, `Usage:
  multiai update [options]      Verifier et installer une mise a jour persistante

Options:
  --check                       Verifier uniquement (JSON, aucune installation)
  --yes, -y                     Confirmer sans question interactive

Contrat de securite:
  - le check de demarrage ne fait qu'une notification;
  - l'installation est deleguee au gestionnaire npm detecte;
  - aucune version temporaire n'est executee;
  - les installations manuelles ou inconnues sont refusees.

Exemples:
  multiai update
  multiai update --check
  multiai update --yes`)
}

func cmdUpdate(args []string) int {
	return runUpdate(args, os.Stdin, os.Stdout, os.Stderr)
}

func runUpdate(args []string, input io.Reader, output, errorOutput io.Writer) int {
	options, err := parseUpdateFlags(args)
	if err != nil {
		fmt.Fprintf(errorOutput, "Erreur: %v\n", err)
		return 1
	}
	if options.help {
		printUpdateHelpTo(output)
		return 0
	}

	checkCtx, cancelCheck := context.WithTimeout(context.Background(), updateCheckTimeout)
	release, err := fetchUpdateRelease(checkCtx)
	cancelCheck()
	if err != nil {
		fmt.Fprintf(errorOutput, "[X] Impossible de verifier les mises a jour : %v\n", err)
		return 2
	}

	hasUpdate := update.IsNewer(version, release.TagName)
	latestVersion := strings.TrimPrefix(release.TagName, "v")
	if options.check {
		result := struct {
			CurrentVersion string `json:"current_version"`
			LatestVersion  string `json:"latest_version"`
			HasUpdate      bool   `json:"has_update"`
		}{
			CurrentVersion: version,
			LatestVersion:  latestVersion,
			HasUpdate:      hasUpdate,
		}
		encoder := json.NewEncoder(output)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			fmt.Fprintf(errorOutput, "[X] Erreur de sortie JSON : %v\n", err)
			return 3
		}
		return 0
	}

	if !hasUpdate {
		fmt.Fprintln(output, "[i] Deja a jour")
		return 0
	}

	fmt.Fprintf(output, "Version actuelle   : %s\n", version)
	fmt.Fprintf(output, "Version disponible : %s\n", latestVersion)
	if !options.yes {
		fmt.Fprint(output, "Installer cette version via le gestionnaire detecte ? [O/n] ")
		reader := bufio.NewReader(input)
		answer, readErr := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			fmt.Fprintf(errorOutput, "[X] Impossible de lire la confirmation : %v\n", readErr)
			return 1
		}
		if errors.Is(readErr, io.EOF) && answer == "" {
			fmt.Fprintln(output, "Mise a jour annulee : confirmation absente.")
			return 0
		}
		if answer == "n" || answer == "non" {
			fmt.Fprintln(output, "Mise a jour annulee.")
			return 0
		}
	}

	fmt.Fprintln(output, "Installation persistante en cours...")
	installCtx, cancelInstall := context.WithTimeout(context.Background(), updateInstallTimeout)
	result, err := installUpdateRelease(installCtx, release)
	cancelInstall()
	if err != nil {
		fmt.Fprintf(errorOutput, "[X] Mise a jour non installee : %v\n", err)
		if errors.Is(err, update.ErrUnsupportedInstall) {
			fmt.Fprintf(errorOutput, "[i] Reinstallation recommandee : npx --yes --allow-scripts=multiai multiai@%s install\n", latestVersion)
		}
		return 2
	}

	_ = update.WriteCache(update.Cache{LastCheck: time.Now(), LatestVersion: release.TagName})
	fmt.Fprintf(output, "[OK] multiai %s installe de maniere persistante via %s.\n", result.Version, result.Manager)
	return 0
}
