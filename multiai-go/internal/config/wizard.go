// Package config implements the interactive API key configuration wizard,
// driven by the embedded provider catalog (internal/catalog).
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/lrochetta/multiai/internal/catalog"
	"github.com/lrochetta/multiai/internal/cli"
	"github.com/lrochetta/multiai/internal/env"
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
		return false, "placeholder non configure"
	}
	if len(key) < 10 {
		return false, "cle trop courte (min 10 caracteres)"
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
		return false, fmt.Sprintf("format invalide pour %s (attendu: %s)", prov.ID, prov.KeyPattern)
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
// by its catalog id (e.g. "openrouter"), skipping the menu — backs
// `multiai config --provider <id>`. A nil reader defaults to stdin.
func ConfigureProviderByID(profiles []profile.Profile, providerID string, reader *bufio.Reader) error {
	cat := catalog.Default()
	prov, ok := cat.ProviderByID(providerID)
	if !ok {
		return fmt.Errorf("fournisseur inconnu : %q (valides : %s)",
			providerID, strings.Join(cat.ProviderIDs(), ", "))
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
		cli.PrintInfo("Configuration des cles API")
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
			color := cli.StatusColor(configured, total)
			line := fmt.Sprintf("  %d. %-36s %s", i+1, prov.Display, status)
			if configured > 0 {
				line += fmt.Sprintf(" (%d/%d)", configured, total)
			}
			fmt.Println(cli.Colorize(line, color))
			fmt.Printf("     -> %s\n", prov.URL)
		}

		fmt.Println()
		fmt.Println("a. Configurer tous les fournisseurs en sequence")
		fmt.Println("e. Effacer des cles API")
		fmt.Println("0. Retour")
		fmt.Println()
		fmt.Print("Choix : ")

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch {
		case choice == "0":
			return nil

		case strings.EqualFold(choice, "a"):
			for _, prov := range cat.Providers {
				configureProvider(prov, byShortcut, reader)
			}
			cli.PrintSuccess("Configuration terminee.")
			fmt.Println()
			fmt.Print("Voulez-vous lancer un profil maintenant ? (o/N) : ")
			choice2, _ := reader.ReadString('\n')
			if strings.TrimSpace(strings.ToLower(choice2)) == "o" {
				fmt.Println("Utilisez 'multiai launch -p <shortcut>' pour lancer.")
				fmt.Println("Lancez 'multiai list' pour voir les profils disponibles.")
			}
			return nil

		case strings.EqualFold(choice, "e"):
			runEraseMenu(cat, byShortcut, reader)
			fmt.Println()
			fmt.Print("Entree pour revenir : ")
			_, _ = reader.ReadString('\n')

		default:
			idx, err := strconv.Atoi(choice)
			if err != nil || idx < 1 || idx > len(cat.Providers) {
				cli.PrintWarning("Choix invalide.")
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
	cli.PrintInfo(fmt.Sprintf("  %s", prov.Display))
	fmt.Printf("  Creer une cle : %s\n", prov.URL)
	if prov.Note != "" {
		fmt.Printf("  Note : %s\n", prov.Note)
	}

	// Collect installed profiles of the group, keeping catalog order.
	var shortcuts []string
	for _, sc := range prov.Shortcuts {
		if _, ok := byShortcut[sc]; ok {
			shortcuts = append(shortcuts, sc)
		}
	}
	if len(shortcuts) == 0 {
		cli.PrintWarning("  Aucun profil installe pour ce fournisseur.")
		return
	}
	fmt.Printf("  Profils : %s\n", strings.Join(shortcuts, ", "))

	// Determine the key variable for the first installed shortcut.
	varName := prov.VarMap[shortcuts[0]]
	p := byShortcut[shortcuts[0]]
	currentValue := p.Env[varName]

	isSentinel := currentValue == secret.Sentinel

	// Show current value (masked).
	masked := env.MaskSecret(currentValue)
	fmt.Printf("  Variable  : %s\n", varName)
	if isSentinel {
		fmt.Printf("  Valeur    : %s (stockee dans le credential store)\n", cli.Colorize(masked, "\033[32m"))
	} else if !dotenv.IsPlaceholder(currentValue) {
		fmt.Printf("  Valeur    : %s (en clair dans le .env)\n", cli.Colorize(masked, "\033[33m"))
	} else {
		fmt.Printf("  Valeur    : %s\n", cli.Colorize(masked, "\033[90m"))
	}
	fmt.Println()

	// Build the key prompt.
	prompt := fmt.Sprintf("Nouvelle cle API (vide = inchanger) : ")
	if isSentinel {
		prompt = fmt.Sprintf("Nouvelle cle API (vide = inchanger, effacer = e) : ")
	}

	fmt.Print(prompt)
	newKey, _ := reader.ReadString('\n')
	newKey = strings.TrimSpace(newKey)

	if newKey == "" {
		fmt.Println("  Aucune modification.")
		return
	}

	if newKey == "e" && isSentinel {
		// Erase from store and restore placeholder.
		store, err := secret.NewStore()
		if err != nil {
			cli.PrintWarning(fmt.Sprintf("  Credential store inaccessible : %v", err))
			return
		}
		isSentinel = false
		for _, sc := range shortcuts {
			v := prov.VarMap[sc]
			if err := store.Delete(secret.ServiceForProfile(byShortcut[sc].Path), v); err != nil {
				cli.PrintWarning(fmt.Sprintf("  Erreur effacement %s : %v", sc, err))
				continue
			}
			if err := setEnvVarInFile(byShortcut[sc].Path, v, "PASTE_" + v + "_HERE"); err != nil {
				cli.PrintWarning(fmt.Sprintf("  Erreur mise a jour %s : %v", sc, err))
				continue
			}
			// Reload in-memory so the menu status refreshes.
			byShortcut[sc].Env[v] = "PASTE_" + v + "_HERE"
		}
		cli.PrintSuccess("Cle effacee du credential store.")
		return
	}

	// Validate format.
	if valid, msg := validateAPIKey(prov, newKey); !valid {
		cli.PrintWarning(fmt.Sprintf("  Format invalide : %s", msg))
		fmt.Print("  Confirmer quand meme ? (o/N) : ")
		confirm, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(confirm)) != "o" {
			fmt.Println("  Annule.")
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
			cli.PrintWarning(fmt.Sprintf("  %s : %v", sc, err))
			continue
		}
		// Reload in-memory.
		pp.Env[v] = newKey
		updated++
	}
	if updated > 0 {
		cli.PrintSuccess(fmt.Sprintf("  %d profil(s) mis a jour.", updated))
	} else {
		cli.PrintWarning("  Aucun profil mis a jour (variable introuvable dans les .env).")
	}
}

// updateEnvFile persists a secret for one profile: real value in the
// credential store, sentinel in the .env file.
// When allowPlaintext is false and the credential store is inaccessible, it
// returns an error to prevent silent plaintext downgrade.
func updateEnvFile(path, varName, newValue string, allowPlaintext bool) error {
	// Credential store FIRST: the file only receives the sentinel when the
	// store write succeeded (invariant: sentinel in file => value in store,
	// otherwise launch would export a dangling sentinel as the API key).
	fileValue := newValue
	storeErrMsg := ""
	if store, err := secret.NewStore(); err == nil {
		if err := store.Set(secret.ServiceForProfile(path), varName, newValue); err == nil {
			fileValue = secret.Sentinel
		} else {
			storeErrMsg = fmt.Sprintf("  Credential store inaccessible : %v", err)
		}
	} else {
		storeErrMsg = fmt.Sprintf("  Credential store inaccessible : %v", err)
	}
	if fileValue != secret.Sentinel {
		if !allowPlaintext {
			return fmt.Errorf("%s. Utilisez --allow-plaintext pour forcer l'ecriture en clair de %s dans %s",
				storeErrMsg, varName, path)
		}
		cli.PrintWarning(fmt.Sprintf("  Credential store indisponible : %s sera ecrit EN CLAIR dans %s", varName, path))
	}
	return setEnvVarInFile(path, varName, fileValue)
}

// setEnvVarInFile replaces the value of the first non-commented `VAR=` line
// in a .env file, atomically (temp file + rename). It writes the value
// verbatim — secret handling is the caller's business.
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

	// Atomic write via temp file + rename (fsync on temp file before rename).
	tmpFile, err := os.CreateTemp(filepath.Dir(path), ".tmp-*.env")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(newContent); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}