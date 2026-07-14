package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/lrochetta/multiai/internal/profile"
)

func init() {
	register("project", cmdProject)
}

func printProjectHelp(output io.Writer) {
	fmt.Fprintln(output, `Usage:
  multiai project status [--json]  Afficher le fichier, son SHA256 et son etat
  multiai project trust [--json]   Approuver le chemin canonique et les octets actuels
  multiai project untrust          Revoquer l'approbation

Une modification de .multiai.yaml invalide automatiquement la confiance.
Aucun hook ni override projet n'est applique avant approbation explicite.`)
}

func cmdProject(args []string) int {
	return runProject(args, os.Stdout, os.Stderr)
}

func runProject(args []string, output, errorOutput io.Writer) int {
	action := "status"
	asJSON := false
	actionSeen := false
	for _, arg := range args {
		switch arg {
		case "status", "trust", "untrust":
			if actionSeen {
				fmt.Fprintf(errorOutput, "Erreur: action projet multiple ou inattendue: %s\n", arg)
				return 2
			}
			action = arg
			actionSeen = true
		case "--json":
			asJSON = true
		case "--help", "-h":
			printProjectHelp(output)
			return 0
		default:
			fmt.Fprintf(errorOutput, "Erreur: argument projet inconnu: %s\n", arg)
			return 2
		}
	}

	configPath, err := profile.FindProjectConfigPath()
	if err != nil {
		fmt.Fprintf(errorOutput, "Erreur: recherche de .multiai.yaml: %v\n", err)
		return 1
	}
	if configPath == "" {
		fmt.Fprintln(errorOutput, "Erreur: aucun .multiai.yaml ou .multiai.yml trouve dans ce dossier ou ses parents.")
		return 1
	}

	switch action {
	case "trust":
		if warnings, validateErr := profile.ValidateProfileYAML(configPath); validateErr != nil {
			fmt.Fprintf(errorOutput, "Erreur: configuration projet invalide: %v\n", validateErr)
			return 1
		} else if len(warnings) > 0 {
			for _, warning := range warnings {
				fmt.Fprintf(errorOutput, "[!] %s\n", warning)
			}
		}
		status, trustErr := profile.TrustProjectConfig(configPath)
		if trustErr != nil {
			fmt.Fprintf(errorOutput, "Erreur: approbation refusee: %v\n", trustErr)
			return 1
		}
		return writeProjectTrustStatus(output, errorOutput, status, asJSON)
	case "untrust":
		status, inspectErr := profile.InspectProjectConfigTrust(configPath)
		if inspectErr != nil {
			fmt.Fprintf(errorOutput, "Erreur: inspection impossible: %v\n", inspectErr)
			return 1
		}
		if err := profile.UntrustProjectConfig(configPath); err != nil {
			fmt.Fprintf(errorOutput, "Erreur: revocation impossible: %v\n", err)
			return 1
		}
		status.State = profile.ProjectTrustUntrusted
		status.TrustedFingerprint = ""
		status.TrustedAt = ""
		return writeProjectTrustStatus(output, errorOutput, status, asJSON)
	default:
		status, inspectErr := profile.InspectProjectConfigTrust(configPath)
		if inspectErr != nil {
			fmt.Fprintf(errorOutput, "Erreur: inspection impossible: %v\n", inspectErr)
			return 1
		}
		return writeProjectTrustStatus(output, errorOutput, status, asJSON)
	}
}

func writeProjectTrustStatus(output, errorOutput io.Writer, status profile.ProjectTrustStatus, asJSON bool) int {
	if asJSON {
		encoder := json.NewEncoder(output)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(status); err != nil {
			fmt.Fprintf(errorOutput, "Erreur: sortie JSON: %v\n", err)
			return 1
		}
		return 0
	}
	fmt.Fprintf(output, "Fichier : %s\n", status.CanonicalPath)
	fmt.Fprintf(output, "Etat    : %s\n", status.State)
	fmt.Fprintf(output, "SHA256  : %s\n", status.Fingerprint)
	if status.TrustedFingerprint != "" {
		fmt.Fprintf(output, "Approuve: %s\n", status.TrustedFingerprint)
	}
	return 0
}
