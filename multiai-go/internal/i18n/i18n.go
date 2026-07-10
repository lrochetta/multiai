// Package i18n provides a minimal internationalization framework for Multiai.
//
// Usage:
//
//	import "github.com/lrochetta/multiai/internal/i18n"
//
//	// In any function:
//	i18n.T("menu_launch")           // "1. Lancer" (FR) or "1. Launch" (EN)
//	i18n.T("profiles_updated", 3)   // "3 profil(s) mis a jour." or "3 profile(s) updated."
//
// Language detection (in order of priority):
//  1. MULTIAI_LANG environment variable
//  2. LANG environment variable
//  3. Default: French
package i18n

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// Lang represents a language code.
type Lang string

const (
	FR Lang = "fr"
	EN Lang = "en"
)

var (
	currentLang Lang
	langOnce    sync.Once
)

// detectLang detects the language from environment variables.
// Priority: MULTIAI_LANG > LANG > default FR.
func detectLang() Lang {
	langOnce.Do(func() {
		if l := os.Getenv("MULTIAI_LANG"); l != "" {
			lang := strings.ToLower(strings.TrimSpace(l))
			switch lang {
			case "en", "english":
				currentLang = EN
			default:
				currentLang = FR
			}
			return
		}
		if l := os.Getenv("LANG"); l != "" {
			lang := strings.ToLower(strings.TrimSpace(l))
			if strings.HasPrefix(lang, "en") {
				currentLang = EN
				return
			}
		}
		currentLang = FR
	})
	return currentLang
}

// SetLang forces the language for the current process.  Useful in tests.
// Without calling SetLang the language is auto-detected on the first call
// to T.
func SetLang(l Lang) {
	currentLang = l
}

// T looks up key in the current language's translation table and formats the
// result with the given arguments (if any).  If the key is missing from the
// target language it falls back to French; if still missing it returns the
// raw key as a last resort.
func T(key string, args ...interface{}) string {
	lang := detectLang()

	template := messages[lang][key]
	if template == "" {
		// Fallback to French.
		template = messages[FR][key]
	}
	if template == "" {
		return key
	}
	if len(args) > 0 {
		return fmt.Sprintf(template, args...)
	}
	return template
}

// messages is the translation map.  The French entries mirror the original
// hardcoded strings so that existing tests continue to pass.
var messages = map[Lang]map[string]string{
	FR: {
		// -- Core / priority (requis S4.1) -----------------------------------------------
		"profile_not_found":       "profil introuvable",
		"required_secret_missing": "secret obligatoire non configure",
		"will_be_in_plaintext":    "sera ecrit EN CLAIR",
		"warning":                 "Avertissement",
		"error":                   "Erreur",
		"version":                 "Version",
		"launching":               "Lancement de",

		// -- Main menu --------------------------------------------------------------------
		"menu_title":           "Laurent ROCHETTA's MultiAI (AI Code CLI Router) v%s - %d profils",
		"menu_launch":          "1. Lancer",
		"menu_config":          "2. Configurer les cles API",
		"menu_bmad":            "3. BMAD+ -- Gestion du framework",
		"menu_models":          "4. OpenRouter -- Decouvrir les modeles",
		"menu_quit":            "0. Quitter",
		"menu_choice":          "Choix : ",
		"menu_invalid":         "Choix invalide. Options : 1-4, 0 pour quitter",
		"read_error":           "[X] Erreur de lecture: %v",
		"read_error_profile":   "erreur de lecture: %v",
		"back_main":            "0. Retour au menu principal",
		"back_option":          "0. Retour",
		"profiles_count":       "profils",
		"invalid_choice_lower": "choix invalide",
		"launch_help":          "Utilisez 'multiai launch -p <shortcut>' pour lancer.",
		"list_help":            "Lancez 'multiai list' pour voir les profils disponibles.",

		// -- Config wizard -----------------------------------------------------------------
		"config_title":   "Configuration des cles API",
		"config_all":     "a. Configurer tous les fournisseurs en sequence",
		"erase_keys":     "e. Effacer des cles API",
		"config_done":    "Configuration terminee.",
		"enter_return":   "Entree pour revenir : ",
		"invalid_choice": "Choix invalide.",
		"choice_prompt":  "Choix : ",

		// -- API key prompts ---------------------------------------------------------------
		"key_prompt":        "Nouvelle cle API (vide = inchanger) : ",
		"key_prompt_erase":  "Nouvelle cle API (vide = inchanger, effacer = e) : ",
		"no_change":         "Aucune modification.",
		"cancelled":         "Annule.",
		"confirm_anyway":    "Confirmer quand meme ? (o/N) : ",
		"launch_now_prompt": "Voulez-vous lancer un profil maintenant ? (o/N) : ",

		// -- Tool / profile selection ------------------------------------------------------
		"tools_available":    "Outils disponibles",
		"back_tool_sel":      "0. Retour a la selection d'outil",
		"choose_tool":        "Choisis un outil : ",
		"choose_profile":     "Choisis un profil : ",
		"no_profile_tool":    "aucun profil pour l'outil : %s",
		"profiles_available": "Profils disponibles pour %s",
		"title_profiles":     "Laurent ROCHETTA's MultiAI (AI Code CLI Router) v%s - %d profils",

		// -- Status / results --------------------------------------------------------------
		"unknown_cmd":              "Commande inconnue : %s",
		"update_available":         "\n[i] v%s disponible. Lancez 'multiai update'.\n\n",
		"warning_manifest":         "Avertissement: impossible de lire le manifeste installe",
		"warning_install_profiles": "Avertissement: impossible d'installer les profils dans %s",
		"profiles_installed":       "Profils installes dans %s (%d fichiers)",
		"provider_flag_expected":   "Erreur: --provider attend un identifiant de fournisseur",
		"profiles_updated":         "%d profil(s) mis a jour.",
		"profiles_not_updated":     "Aucun profil mis a jour (variable introuvable dans les .env).",
		"key_erased":               "Cle effacee du credential store.",
		"process_exit_code":        "Le processus s'est termine avec le code: %d",
		"timeout_reached":          "Le processus a depasse le delai de %s",

		// -- Provider config ---------------------------------------------------------------
		"create_key_at":    "Creer une cle : %s",
		"note":             "Note : %s",
		"no_prof_provider": "Aucun profil installe pour ce fournisseur.",
		"profiles_label":   "Profils : %s",
		"variable_label":   "Variable  : %s",
		"unknown_provider": "fournisseur inconnu : %q (valides : %s)",

		// -- Validation --------------------------------------------------------------------
		"placeholder_unconfigured": "placeholder non configure",
		"key_too_short":            "cle trop courte (min 10 caracteres)",
		"invalid_format":           "format invalide pour %s (attendu: %s)",
		"invalid_format_simple":    "Format invalide",

		// -- Store / creds -----------------------------------------------------------------
		"cred_store_unavailable": "Credential store inaccessible : %v",
		"store_invalid_backend":  "Backend de stockage invalide : %s. Valeurs acceptees : %s",
		"store_init_error":       "Erreur d'initialisation du store '%s' : %v",
		"store_selected":         "Backend de stockage : %s",
		"store_flag_help":        "  multiai config --store <backend>    Forcer un backend de stockage (wincred, keychain, secret-service, file, auto)",

		// -- Migration --------------------------------------------------------------------
		"migrate_no_legacy":     "Aucune installation PowerShell legacy detectee.\nRecherchez avec : multiai migrate --from-ps <chemin>",
		"migrate_detect_error":  "Erreur de detection de l'installation PowerShell legacy",
		"migrate_failed":        "Echec de la migration PowerShell -> Go",
		"migrate_help_usage":    "Usage:\n  multiai migrate [options]              Migrer depuis une installation PowerShell legacy\n\nOptions:\n  --from-ps <chemin>     Chemin vers l'installation PowerShell legacy\n  --dry-run              Simulation sans ecriture (rapport seul)\n  --json, -j             Sortie au format JSON\n\nExemples:\n  multiai migrate\n  multiai migrate --from-ps C:\\Users\\laurent\\AppData\\Roaming\\npm\\node_modules\\multiai\n  multiai migrate --dry-run --json",
		"migrate_help_options":  "Options:\n  --from-ps <chemin>     Chemin vers l'installation PowerShell legacy\n  --dry-run              Simulation sans ecriture (rapport seul)\n  --json, -j             Sortie au format JSON",
		"migrate_help_examples": "Exemples:\n  multiai migrate\n  multiai migrate --from-ps <chemin>\n  multiai migrate --dry-run --json",
	},

	EN: {
		// -- Core / priority ---------------------------------------------------------------
		"profile_not_found":       "profile not found",
		"required_secret_missing": "required secret not configured",
		"will_be_in_plaintext":    "will be written IN PLAINTEXT",
		"warning":                 "Warning",
		"error":                   "Error",
		"version":                 "Version",
		"launching":               "Launching",

		// -- Main menu --------------------------------------------------------------------
		"menu_title":           "Laurent ROCHETTA's MultiAI (AI Code CLI Router) v%s - %d profiles",
		"menu_launch":          "1. Launch",
		"menu_config":          "2. Configure API keys",
		"menu_bmad":            "3. BMAD+ -- Framework management",
		"menu_models":          "4. OpenRouter -- Discover models",
		"menu_quit":            "0. Quit",
		"menu_choice":          "Choice : ",
		"menu_invalid":         "Invalid choice. Options: 1-4, 0 to quit",
		"read_error":           "[X] Read error: %v",
		"read_error_profile":   "read error: %v",
		"back_main":            "0. Back to main menu",
		"back_option":          "0. Back",
		"profiles_count":       "profiles",
		"invalid_choice_lower": "invalid choice",
		"launch_help":          "Use 'multiai launch -p <shortcut>' to launch.",
		"list_help":            "Run 'multiai list' to see available profiles.",

		// -- Config wizard -----------------------------------------------------------------
		"config_title":   "API Key Configuration",
		"config_all":     "a. Configure all providers in sequence",
		"erase_keys":     "e. Erase API keys",
		"config_done":    "Configuration complete.",
		"enter_return":   "Press Enter to return : ",
		"invalid_choice": "Invalid choice.",
		"choice_prompt":  "Choice : ",

		// -- API key prompts ---------------------------------------------------------------
		"key_prompt":        "New API key (empty = unchanged) : ",
		"key_prompt_erase":  "New API key (empty = unchanged, erase = e) : ",
		"no_change":         "No modification.",
		"cancelled":         "Cancelled.",
		"confirm_anyway":    "Confirm anyway? (y/N) : ",
		"launch_now_prompt": "Would you like to launch a profile now? (y/N) : ",

		// -- Tool / profile selection ------------------------------------------------------
		"tools_available":    "Available Tools",
		"back_tool_sel":      "0. Back to tool selection",
		"choose_tool":        "Choose a tool : ",
		"choose_profile":     "Choose a profile : ",
		"no_profile_tool":    "no profile for tool: %s",
		"profiles_available": "Profiles available for %s",
		"title_profiles":     "Laurent ROCHETTA's MultiAI (AI Code CLI Router) v%s - %d profiles",

		// -- Status / results --------------------------------------------------------------
		"unknown_cmd":              "Unknown command: %s",
		"update_available":         "\n[i] v%s available. Run 'multiai update'.\n\n",
		"warning_manifest":         "Warning: unable to read installed manifest",
		"warning_install_profiles": "Warning: unable to install profiles in %s",
		"profiles_installed":       "Profiles installed in %s (%d files)",
		"provider_flag_expected":   "Error: --provider expects a provider identifier",
		"profiles_updated":         "%d profile(s) updated.",
		"profiles_not_updated":     "No profiles updated (variable not found in .env files).",
		"key_erased":               "Key erased from credential store.",
		"process_exit_code":        "The process exited with code: %d",
		"timeout_reached":          "Process timed out after %s",

		// -- Provider config ---------------------------------------------------------------
		"create_key_at":    "Create key at: %s",
		"note":             "Note: %s",
		"no_prof_provider": "No profile installed for this provider.",
		"profiles_label":   "Profiles: %s",
		"variable_label":   "Variable: %s",
		"unknown_provider": "unknown provider: %q (valid: %s)",

		// -- Validation --------------------------------------------------------------------
		"placeholder_unconfigured": "placeholder not configured",
		"key_too_short":            "key too short (min 10 characters)",
		"invalid_format":           "invalid format for %s (expected: %s)",
		"invalid_format_simple":    "Invalid format",

		// -- Store / creds -----------------------------------------------------------------
		"cred_store_unavailable": "Credential store unavailable: %v",
		"store_invalid_backend":  "Invalid storage backend: %s. Accepted values: %s",
		"store_init_error":       "Error initializing store '%s': %v",
		"store_selected":         "Storage backend: %s",
		"store_flag_help":        "  multiai config --store <backend>    Force a storage backend (wincred, keychain, secret-service, file, auto)",

		// -- Migration --------------------------------------------------------------------
		"migrate_no_legacy":     "No PowerShell legacy installation detected.\nSearch with: multiai migrate --from-ps <path>",
		"migrate_detect_error":  "Error detecting PowerShell legacy installation",
		"migrate_failed":        "PowerShell -> Go migration failed",
		"migrate_help_usage":    "Usage:\n  multiai migrate [options]              Migrate from a PowerShell legacy installation\n\nOptions:\n  --from-ps <path>       Path to the PowerShell legacy installation\n  --dry-run              Simulation without writing (report only)\n  --json, -j             JSON output\n\nExamples:\n  multiai migrate\n  multiai migrate --from-ps /usr/local/lib/node_modules/multiai\n  multiai migrate --dry-run --json",
		"migrate_help_options":  "Options:\n  --from-ps <path>       Path to the PowerShell legacy installation\n  --dry-run              Simulation without writing (report only)\n  --json, -j             JSON output",
		"migrate_help_examples": "Examples:\n  multiai migrate\n  multiai migrate --from-ps <path>\n  multiai migrate --dry-run --json",
	},
}
