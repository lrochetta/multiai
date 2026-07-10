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
	b.WriteString(fmt.Sprintf("Installation detectee : %s\n", r.Detected.RootDir))
	if r.Detected.Version != "" {
		b.WriteString(fmt.Sprintf("Version legacy        : v%s\n", r.Detected.Version))
	}
	b.WriteString(fmt.Sprintf("Profiles trouves      : %d\n", r.Detected.ProfileCount))
	b.WriteString("\n")

	if r.DryRun {
		b.WriteString("[SIMULATION] Aucun fichier n'a ete modifie.\n\n")
	} else if r.BackupPath != "" {
		b.WriteString(fmt.Sprintf("Sauvegarde effectuee dans : %s\n", r.BackupPath))
	}
	b.WriteString(fmt.Sprintf("Destination : %s\n", r.TargetDir))
	b.WriteString("\n")

	if len(r.Migrated) > 0 {
		b.WriteString(fmt.Sprintf("Migres  (%d) :\n", len(r.Migrated)))
		for _, name := range r.Migrated {
			b.WriteString(fmt.Sprintf("  + %s\n", name))
		}
		b.WriteString("\n")
	}

	if len(r.Skipped) > 0 {
		b.WriteString(fmt.Sprintf("Ignores (%d) :\n", len(r.Skipped)))
		for _, name := range r.Skipped {
			b.WriteString(fmt.Sprintf("  - %s (deja present)\n", name))
		}
		b.WriteString("\n")
	}

	if !r.DryRun {
		b.WriteString(fmt.Sprintf("Migration terminee : %d copie(s), %d ignore(s).\n",
			len(r.Migrated), len(r.Skipped)))
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
