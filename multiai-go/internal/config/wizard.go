// Package config implements the interactive API key configuration wizard,
// driven by the embedded provider catalog (internal/catalog).
package config

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/lrochetta/multiai/internal/catalog"
	"github.com/lrochetta/multiai/internal/display"
	"github.com/lrochetta/multiai/internal/env"
	"github.com/lrochetta/multiai/internal/fsutil"
	"github.com/lrochetta/multiai/internal/i18n"
	"github.com/lrochetta/multiai/internal/profile"
	"github.com/lrochetta/multiai/internal/secret"
	"github.com/lrochetta/multiai/pkg/dotenv"
)

// Provider is the catalog provider type, re-exported for this package's
// consumers.
type Provider = catalog.Provider

// DefaultProviders returns the provider catalog in menu order. The list is
// data-driven (internal/catalog, embedded providers.yaml): adding a provider
// means editing the YAML, not this package.
func DefaultProviders() []Provider {
	return catalog.Default().Providers
}

// validateAPIKey validates an API key against the provider's optional
// KeyPattern from the catalog. Failures are advisory: the caller lets the
// user override (parity with PS, which has no format validation at all).
func validateAPIKey(prov Provider, key string) (bool, string) {
	if dotenv.IsPlaceholder(key) {
		return false, i18n.T("placeholder_unconfigured")
	}
	if len(key) < 10 {
		return false, i18n.T("key_too_short")
	}
	if prov.KeyPattern == "" {
		return true, "" // no pattern in the catalog: accept
	}
	re, err := regexp.Compile(prov.KeyPattern)
	if err != nil {
		// The catalog validates patterns at load time; never block on this.
		return true, ""
	}
	if !re.MatchString(key) {
		return false, fmt.Sprintf("%s", i18n.T("invalid_format", prov.ID, prov.KeyPattern))
	}
	return true, ""
}

// shortcutIndex maps profile shortcuts to their profile, so a provider's
// key can be propagated to every installed profile of the group.
func shortcutIndex(profiles []profile.Profile) map[string]*profile.Profile {
	byShortcut := make(map[string]*profile.Profile, len(profiles))
	for i := range profiles {
		byShortcut[profiles[i].Shortcut] = &profiles[i]
	}
	return byShortcut
}

// providerStatus counts the installed profiles of a provider group and how
// many of them hold a configured (non-placeholder) key. The credential-store
// sentinel counts as configured.
func providerStatus(prov Provider, byShortcut map[string]*profile.Profile) (configured, total int) {
	for _, sc := range prov.Shortcuts {
		p, ok := byShortcut[sc]
		if !ok {
			continue
		}
		total++
		if !dotenv.IsPlaceholder(p.Env[prov.VarMap[sc]]) {
			configured++
		}
	}
	return configured, total
}

// InteractiveConfig runs the interactive API key configuration menu.
func InteractiveConfig(profiles []profile.Profile) error {
	return runConfigMenu(catalog.Default(), shortcutIndex(profiles), bufio.NewReader(os.Stdin))
}

// ConfigureProviderByID runs the key prompt for a single provider selected
// by its catalog id (e.g. "openrouter"), skipping the menu â€” backs
// `multiai config --provider <id>`. A nil reader defaults to stdin.
func ConfigureProviderByID(profiles []profile.Profile, providerID string, reader *bufio.Reader) error {
	cat := catalog.Default()
	prov, ok := cat.ProviderByID(providerID)
	if !ok {
		return fmt.Errorf("%s", i18n.T("unknown_provider",
			providerID, strings.Join(cat.ProviderIDs(), ", ")))
	}
	if reader == nil {
		reader = bufio.NewReader(os.Stdin)
	}
	configureProvider(prov, shortcutIndex(profiles), reader)
	return nil
}

// runConfigMenu is the menu loop, split from InteractiveConfig so tests can
// drive it with a scripted reader.
func runConfigMenu(cat *catalog.Catalog, byShortcut map[string]*profile.Profile, reader *bufio.Reader) error {
	for {
		fmt.Println()
		display.PrintInfo(i18n.T("config_title"))
		fmt.Println(strings.Repeat("-", 58))

		currentRegion := ""
		for i, prov := range cat.Providers {
			if prov.Region != currentRegion {
				currentRegion = prov.Region
				fmt.Println()
				fmt.Printf("  %s\n", cat.RegionLabel(currentRegion))
				fmt.Println("  " + strings.Repeat("-", 48))
			}
			configured, total := providerStatus(prov, byShortcut)
			status := "[--]"
			if configured == total && total > 0 {
				status = "[OK]"
			} else if configured > 0 {
				status = "[~~]"
			}
			color := display.StatusColor(configured, total)
			line := fmt.Sprintf("  %d. %-36s %s", i+1, prov.Display, status)
			if configured > 0 {
				line += fmt.Sprintf(" (%d/%d)", configured, total)
			}
			fmt.Println(display.Colorize(color, line))
			fmt.Printf("     -> %s\n", prov.URL)
		}

		fmt.Println()
		fmt.Println(i18n.T("config_all"))
		fmt.Println(i18n.T("erase_keys"))
		fmt.Println(i18n.T("back_option"))
		fmt.Println()
		fmt.Print(i18n.T("choice_prompt"))

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch {
		case choice == "0":
			return nil

		case strings.EqualFold(choice, "a"):
			for _, prov := range cat.Providers {
				configureProvider(prov, byShortcut, reader)
			}
			display.PrintSuccess(i18n.T("config_done"))
			fmt.Println()
			fmt.Print(i18n.T("launch_now_prompt"))
			choice2, _ := reader.ReadString('\n')
			if strings.TrimSpace(strings.ToLower(choice2)) == "o" {
				fmt.Println(i18n.T("launch_help"))
				fmt.Println(i18n.T("list_help"))
			}
			return nil

		case strings.EqualFold(choice, "e"):
			runEraseMenu(cat, byShortcut, reader)
			fmt.Println()
			fmt.Print(i18n.T("enter_return"))
			_, _ = reader.ReadString('\n')

		default:
			idx, err := strconv.Atoi(choice)
			if err != nil || idx < 1 || idx > len(cat.Providers) {
				display.PrintWarning(i18n.T("invalid_choice"))
				continue
			}
			configureProvider(cat.Providers[idx-1], byShortcut, reader)
		}
	}
}

// configureProvider shares the caller's reader: a second bufio.Reader on
// os.Stdin would lose whatever the first one already buffered (piped input
// like `printf "1\nsk-...\n0\n" | multiai config` would silently read EOF).
func configureProvider(prov Provider, byShortcut map[string]*profile.Profile, reader *bufio.Reader) {
	fmt.Println()
	display.PrintInfo("  " + prov.Display)
	fmt.Printf("  %s\n", i18n.T("create_key_at", prov.URL))
	if prov.Note != "" {
		fmt.Printf("  %s\n", i18n.T("note", prov.Note))
	}

	// Collect installed profiles of the group, keeping catalog order.
	var shortcuts []string
	for _, sc := range prov.Shortcuts {
		if _, ok := byShortcut[sc]; ok {
			shortcuts = append(shortcuts, sc)
		}
	}
	if len(shortcuts) == 0 {
		display.PrintWarning("  " + i18n.T("no_prof_provider"))
		return
	}
	fmt.Printf("  %s\n", i18n.T("profiles_label", strings.Join(shortcuts, ", ")))

	// Determine the key variable for the first installed shortcut.
	varName := prov.VarMap[shortcuts[0]]
	p := byShortcut[shortcuts[0]]
	currentValue := p.Env[varName]

	isSentinel := currentValue == secret.Sentinel

	// Show current value (masked).
	masked := env.MaskSecret(currentValue)
	fmt.Printf("  %s\n", i18n.T("variable_label", varName))
	if isSentinel {
		fmt.Printf("  Valeur    : %s (stockee dans le credential store)\n", display.Colorize("\033[32m", masked))
	} else if !dotenv.IsPlaceholder(currentValue) {
		fmt.Printf("  Valeur    : %s (en clair dans le .env)\n", display.Colorize("\033[33m", masked))
	} else {
		fmt.Printf("  Valeur    : %s\n", display.Colorize("\033[90m", masked))
	}
	fmt.Println()

	// Build the key prompt.
	var prompt string
	if isSentinel {
		prompt = i18n.T("key_prompt_erase")
	} else {
		prompt = i18n.T("key_prompt")
	}

	fmt.Print(prompt)
	newKey, _ := reader.ReadString('\n')
	newKey = strings.TrimSpace(newKey)

	if newKey == "" {
		fmt.Println("  " + i18n.T("no_change"))
		return
	}

	if newKey == "e" && isSentinel {
		// Erase from store and restore placeholder.
		store, err := secret.NewStore()
		if err != nil {
			display.PrintWarning(fmt.Sprintf("  %s", i18n.T("cred_store_unavailable", err)))
			return
		}
		isSentinel = false
		for _, sc := range shortcuts {
			v := prov.VarMap[sc]
			if err := store.Delete(secret.ServiceForProfile(byShortcut[sc].Path), v); err != nil {
				display.PrintWarning(fmt.Sprintf("  Erreur effacement %s : %v", sc, err))
				continue
			}
			if err := setEnvVarInFile(byShortcut[sc].Path, v, "PASTE_"+v+"_HERE"); err != nil {
				display.PrintWarning(fmt.Sprintf("  Erreur mise a jour %s : %v", sc, err))
				continue
			}
			// Reload in-memory so the menu status refreshes.
			byShortcut[sc].Env[v] = "PASTE_" + v + "_HERE"
		}
		display.PrintSuccess(i18n.T("key_erased"))
		return
	}

	// Validate format.
	if valid, msg := validateAPIKey(prov, newKey); !valid {
		display.PrintWarning(fmt.Sprintf("  %s: %s", i18n.T("invalid_format_simple"), msg))
		fmt.Print("  " + i18n.T("confirm_anyway"))
		confirm, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(confirm)) != "o" {
			fmt.Println("  " + i18n.T("cancelled"))
			return
		}
	}

	// Apply to all installed profiles.
	updated := 0
	for _, sc := range shortcuts {
		v := prov.VarMap[sc]
		pp := byShortcut[sc]

		// Use the allow-plaintext flag when the credential store is
		// unavailable so the user intent is explicit.
		if err := updateEnvFile(pp.Path, v, newKey, false); err != nil {
			display.PrintWarning(fmt.Sprintf("  %s : %v", sc, err))
			continue
		}
		// Reload in-memory.
		pp.Env[v] = newKey
		updated++
	}
	if updated > 0 {
		display.PrintSuccess(i18n.T("profiles_updated", updated))
	} else {
		display.PrintWarning("  " + i18n.T("profiles_not_updated"))
	}
}

// updateEnvFile persists a secret for one profile: real value in the
// credential store, sentinel in the .env file.
// When allowPlaintext is false and the credential store is inaccessible, it
// returns an error to prevent silent plaintext downgrade.
//
// ORDER (crash-safe):
//  1. Write the secret into the credential store.
//  2. Write the sentinel into the .env file.
//  3. If step 2 fails, rollback step 1 (delete from store) so that a
//     dangling credential never accumulates without its sentinel.
func updateEnvFile(path, varName, newValue string, allowPlaintext bool) error {
	store, storeErr := secret.NewStore()
	storeAvailable := storeErr == nil

	fileValue := newValue
	storeSetDone := false
	storeErrMsg := ""
	if storeAvailable {
		if err := store.Set(secret.ServiceForProfile(path), varName, newValue); err == nil {
			fileValue = secret.Sentinel
			storeSetDone = true
		} else {
			storeErrMsg = fmt.Sprintf("  %s", i18n.T("cred_store_unavailable", err))
		}
	} else {
		storeErrMsg = fmt.Sprintf("  %s", i18n.T("cred_store_unavailable", storeErr))
	}
	if fileValue != secret.Sentinel {
		if !allowPlaintext {
			return fmt.Errorf("%s. Utilisez --allow-plaintext pour forcer l'ecriture en clair de %s dans %s",
				storeErrMsg, varName, path)
		}
		display.PrintWarning(fmt.Sprintf("  Credential store indisponible : %s %s dans %s", varName, i18n.T("will_be_in_plaintext"), path))
	}

	if err := setEnvVarInFile(path, varName, fileValue); err != nil {
		// Rollback: the sentinel write failed, so remove the credential
		// we just stored to prevent a dangling unreferenced entry.
		if storeSetDone {
			_ = store.Delete(secret.ServiceForProfile(path), varName)
		}
		return err
	}
	return nil
}

// setEnvVarInFile replaces the value of the first non-commented `VAR=` line
// in a .env file, atomically. It writes the value
// verbatim â€” secret handling is the caller's business.
func setEnvVarInFile(path, varName, value string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		// Match the key tolerating spaces around '=' (dotenv.Parse accepts
		// "VAR = value"); an exact "VAR=" prefix would miss such a line and
		// silently orphan a credential-store entry.
		eq := strings.IndexByte(trimmed, '=')
		if eq < 0 {
			continue
		}
		if strings.TrimSpace(trimmed[:eq]) == varName {
			lines[i] = varName + "=" + value
			found = true
			break
		}
	}
	if !found {
		// Append the variable at the end.
		lines = append(lines, "")
		lines = append(lines, varName+"="+value)
	}

	newContent := strings.Join(lines, "\n")
	return fsutil.WriteFileAtomic(path, []byte(newContent), 0644)
}
