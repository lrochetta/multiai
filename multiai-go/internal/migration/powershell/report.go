package powershell

import (
	"encoding/json"
	"fmt"
	"strings"
)

// String returns a human-readable summary of the migration report.
func (r *MigrationReport) String() string {
	if r.Detected == nil {
		return "Aucune installation PowerShell legacy detectee."
	}

	var b strings.Builder

	b.WriteString("=== Migration PowerShell -> Go ===\n\n")
	fmt.Fprintf(&b, "Installation detectee : %s\n", r.Detected.RootDir)
	if r.Detected.Version != "" {
		fmt.Fprintf(&b, "Version legacy        : v%s\n", r.Detected.Version)
	}
	fmt.Fprintf(&b, "Profiles trouves      : %d\n", r.Detected.ProfileCount)
	b.WriteString("\n")

	if r.DryRun {
		b.WriteString("[SIMULATION] Aucun fichier n'a ete modifie.\n\n")
	} else if r.BackupPath != "" {
		fmt.Fprintf(&b, "Sauvegarde effectuee dans : %s\n", r.BackupPath)
	}
	fmt.Fprintf(&b, "Destination : %s\n", r.TargetDir)
	b.WriteString("\n")

	if len(r.Migrated) > 0 {
		fmt.Fprintf(&b, "Migres  (%d) :\n", len(r.Migrated))
		for _, name := range r.Migrated {
			fmt.Fprintf(&b, "  + %s\n", name)
		}
		b.WriteString("\n")
	}

	if len(r.Skipped) > 0 {
		fmt.Fprintf(&b, "Ignores (%d) :\n", len(r.Skipped))
		for _, name := range r.Skipped {
			fmt.Fprintf(&b, "  - %s (deja present)\n", name)
		}
		b.WriteString("\n")
	}

	if !r.DryRun {
		fmt.Fprintf(&b, "Migration terminee : %d copie(s), %d ignore(s).\n",
			len(r.Migrated), len(r.Skipped))
	}

	return b.String()
}

// ToJSON returns the report as indented JSON.
func (r *MigrationReport) ToJSON() string {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error":%q}`, err.Error())
	}
	return string(data)
}
