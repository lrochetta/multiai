// cmd_migrate.go implements the "multiai migrate" subcommand for migrating
// from a PowerShell legacy installation to the Go version.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/lrochetta/multiai/internal/i18n"
	"github.com/lrochetta/multiai/internal/migration/powershell"
)

func init() {
	register("migrate", cmdMigrate)
}

// migrateOptions collects flags for the migrate subcommand.
type migrateOptions struct {
	fromPS string
	dryRun bool
	json   bool
	help   bool
}

// parseMigrateFlags hand-parses migrate-specific flags.
func parseMigrateFlags(args []string) (*migrateOptions, error) {
	o := &migrateOptions{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--from-ps":
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				o.fromPS = args[i]
			}
		case "--dry-run":
			o.dryRun = true
		case "--json", "-j":
			o.json = true
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

// printMigrateHelp shows the migrate subcommand usage on stdout.
func printMigrateHelp() {
	fmt.Println(i18n.T("migrate_help_usage"))
	fmt.Println()
	fmt.Println(i18n.T("migrate_help_options"))
	fmt.Println()
	fmt.Println(i18n.T("migrate_help_examples"))
}

// cmdMigrate is the subcommand handler registered as "migrate".
func cmdMigrate(args []string) int {
	o, err := parseMigrateFlags(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T("error"), err)
		return 1
	}
	if o.help {
		printMigrateHelp()
		return 0
	}

	// Detection: search the specified path, or default locations.
	var searchDirs []string
	if o.fromPS != "" {
		searchDirs = append(searchDirs, o.fromPS)
	}

	detected, err := powershell.Detect(searchDirs...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[X] %s: %v\n", i18n.T("migrate_detect_error"), err)
		return 2
	}

	if detected == nil {
		fmt.Fprintln(os.Stderr, i18n.T("migrate_no_legacy"))
		return 0
	}

	// Resolve the Go profiles destination directory.
	dstDir := getProfilesDir()

	// Run migration.
	opts := powershell.MigrateOptions{
		DryRun: o.dryRun,
	}
	report, err := powershell.RunMigration(detected, dstDir, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[X] %s: %v\n", i18n.T("migrate_failed"), err)
		return 2
	}

	// Output.
	if o.json {
		fmt.Println(report.ToJSON())
	} else {
		fmt.Println(report.String())
	}

	return 0
}
