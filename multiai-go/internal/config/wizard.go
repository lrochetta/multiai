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
	"github.com/lrochetta/multiai/internal/cli"
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
			fmt.Printf("  %d. %-36s %s (%d/%d)\n", i+1, prov.Display, status, configured, total)
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

	// Current status, read from the first installed profile of the group.
	firstProf := byShortcut[shortcuts[0]]
	firstVar := prov.VarMap[shortcuts[0]]
	currentVal := firstProf.Env[firstVar]
	fmt.Print("  Statut actuel : ")
	if currentVal == secret.Sentinel {
		fmt.Println("[configuree - stockee dans le credential store]")
	} else if dotenv.IsPlaceholder(currentVal) {
		cli.PrintError("[non configuree]")
	} else {
		masked := currentVal
		if len(masked) > 8 {
			masked = masked[:4] + "..." + masked[len(masked)-4:]
		} else if len(masked) > 0 {
			masked = "****"
		}
		fmt.Println(masked)
	}

	fmt.Println()
	fmt.Print("  Nouvelle valeur (vide = ignorer) : ")
	newVal, _ := reader.ReadString('\n')
	newVal = strings.TrimSpace(newVal)

	if newVal == "" {
		fmt.Println("  -> Ignore.")
		return
	}

	if valid, msg := validateAPIKey(prov, newVal); !valid {
		cli.PrintWarning(fmt.Sprintf("  Attention: %s", msg))
		fmt.Print("  Continuer quand meme ? (o/N) : ")
		confirm, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(confirm)) != "o" {
			return
		}
	}

	updated := 0
	for _, sc := range shortcuts {
		p := byShortcut[sc]
		varName := prov.VarMap[sc]
		// Update file first, then memory, so the display stays honest.
		if err := updateEnvFile(p.Path, varName, newVal, false); err == nil {
			p.Env[varName] = newVal
			updated++
			fmt.Printf("    + %-30s [%s]\n", p.DisplayName, p.Shortcut)
		}
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
		return fmt.Errorf("variable %s non trouvee dans %s", varName, path)
	}

	newContent := []byte(strings.Join(lines, "\n"))
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, newContent, 0600); err != nil {
		return fmt.Errorf("cannot write temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("cannot replace %s: %w", path, err)
	}
	return nil
}
