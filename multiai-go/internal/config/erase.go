package config

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"

	"github.com/lrochetta/multiai/internal/catalog"
	"github.com/lrochetta/multiai/internal/display"
	"github.com/lrochetta/multiai/internal/profile"
	"github.com/lrochetta/multiai/internal/secret"
	"github.com/lrochetta/multiai/pkg/dotenv"
)

// EraseProviderKeys resets the provider's key in every installed profile of
// the group: the .env line goes back to the `PASTE_<VAR>_HERE` placeholder
// and the credential-store entry is deleted (parity with PS
// Erase-ProviderKeys L544-559, extended to purge the Sprint 1 store).
// It returns the number of profiles erased.
func EraseProviderKeys(prov Provider, byShortcut map[string]*profile.Profile) int {
	store, storeErr := secret.NewStore()
	if storeErr != nil {
		// Do not touch store below: NewStore may return a typed-nil Store
		// alongside the error, so guard on storeErr, not on store != nil.
		display.PrintWarning(fmt.Sprintf("  Credential store indisponible (%v) : les fichiers seront nettoyes, pas le store.", storeErr))
	}

	erased := 0
	for _, sc := range prov.Shortcuts {
		p, ok := byShortcut[sc]
		if !ok {
			continue
		}
		varName := prov.VarMap[sc]
		prev := p.Env[varName]
		placeholder := "PASTE_" + varName + "_HERE"
		if err := setEnvVarInFile(p.Path, varName, placeholder); err != nil {
			continue
		}
		p.Env[varName] = placeholder
		erased++
		// Purge the credential store whenever the profile held a configured
		// value (sentinel or plaintext). Skipping placeholders avoids creating
		// empty store entries for never-configured profiles.
		if storeErr == nil && !dotenv.IsPlaceholder(prev) {
			if err := store.Delete(secret.ServiceForProfile(p.Path), varName); err != nil {
				display.PrintWarning(fmt.Sprintf("  Store non purge pour %s/%s : %v", p.Shortcut, varName, err))
			}
		}
	}
	return erased
}

// confirmErase asks for the literal "oui" (case-insensitive, like the PS
// Read-Host confirmation) before any destructive action.
func confirmErase(reader *bufio.Reader) bool {
	fmt.Print("Tape \"oui\" pour confirmer : ")
	confirm, _ := reader.ReadString('\n')
	if strings.EqualFold(strings.TrimSpace(confirm), "oui") {
		return true
	}
	fmt.Println("Annule.")
	return false
}

// runEraseMenu shows the erase menu: one entry per catalog provider plus
// "a" to erase everything. Every action requires an explicit confirmation.
func runEraseMenu(cat *catalog.Catalog, byShortcut map[string]*profile.Profile, reader *bufio.Reader) {
	fmt.Println()
	display.PrintInfo("Effacer des cles API")
	fmt.Println(strings.Repeat("-", 58))
	fmt.Println()

	for i, prov := range cat.Providers {
		configured, total := providerStatus(prov, byShortcut)
		status := "[aucune]"
		if configured > 0 {
			status = fmt.Sprintf("[%d cle(s)]", configured)
		}
		fmt.Printf("%d. %-36s %s\n", i+1, prov.Display, status)
		fmt.Printf("    -> %d profil(s) concerne(s)\n", total)
	}

	fmt.Println()
	fmt.Println("a. Effacer TOUTES les cles (tous les fournisseurs)")
	fmt.Println("0. Retour")
	fmt.Println()
	fmt.Print("Choix : ")

	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	if choice == "0" {
		return
	}

	if strings.EqualFold(choice, "a") {
		fmt.Println()
		display.PrintWarning("ATTENTION : Toutes les cles API vont etre effacees !")
		if !confirmErase(reader) {
			return
		}
		totalErased := 0
		for _, prov := range cat.Providers {
			totalErased += EraseProviderKeys(prov, byShortcut)
		}
		fmt.Println()
		display.PrintSuccess(fmt.Sprintf("%d cle(s) effacee(s) au total.", totalErased))
		return
	}

	idx, err := strconv.Atoi(choice)
	if err != nil || idx < 1 || idx > len(cat.Providers) {
		display.PrintWarning("Choix invalide.")
		return
	}
	prov := cat.Providers[idx-1]
	fmt.Println()
	display.PrintWarning(fmt.Sprintf("Effacer la cle pour : %s", prov.Display))
	if !confirmErase(reader) {
		return
	}
	n := EraseProviderKeys(prov, byShortcut)
	fmt.Println()
	display.PrintSuccess(fmt.Sprintf("%d cle(s) effacee(s) pour %s.", n, prov.Display))
}
