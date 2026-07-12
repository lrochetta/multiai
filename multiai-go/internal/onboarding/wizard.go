package onboarding

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lrochetta/multiai/internal/catalog"
	"github.com/lrochetta/multiai/internal/config"
	"github.com/lrochetta/multiai/internal/display"
	"github.com/lrochetta/multiai/internal/logging"
	"github.com/lrochetta/multiai/internal/profile"
	"github.com/lrochetta/multiai/pkg/dotenv"
)

// varRefRe matches a pure PS-style %NAME% indirection. Twenty of the 37
// shipped profiles wire their auth token to another variable of the same
// file (e.g. ANTHROPIC_AUTH_TOKEN=%OPENROUTER_API_KEY%): that value is not
// a configured key, only the referenced variable is.
var varRefRe = regexp.MustCompile(`^%[A-Za-z_][A-Za-z0-9_]*%$`)

// IsFirstRun returns true if no profiles have configured API keys.
func IsFirstRun(profiles []profile.Profile) bool {
	for _, p := range profiles {
		for k, v := range p.Env {
			if !isSecretLike(k) {
				continue
			}
			// %VAR% indirections ship in the pristine templates; skip them
			// so a fresh install is still detected as a first run.
			if varRefRe.MatchString(strings.TrimSpace(v)) {
				continue
			}
			// A credential-store sentinel means the key IS configured
			// (real value lives in the store, written by 'multiai config').
			if !dotenv.IsPlaceholder(v) {
				return false // At least one key is configured
			}
		}
	}
	return true
}

// RunWelcome displays the welcome wizard and optionally configures keys.
func RunWelcome(profiles []profile.Profile) {
	runWelcome(profiles, bufio.NewReader(os.Stdin))
}

func runWelcome(profiles []profile.Profile, reader *bufio.Reader) {
	fmt.Println()
	display.PrintInfo("========================================")
	display.PrintInfo("  Bienvenue dans multiai !")
	display.PrintInfo("========================================")
	fmt.Println()
	fmt.Println("Il semble que ce soit votre premiere utilisation.")
	fmt.Printf("  %d profils disponibles, %d fournisseurs supportes\n",
		len(profiles), len(catalog.Default().Providers))
	fmt.Println("  (liste complete : multiai config)")
	fmt.Println()
	fmt.Println("Etapes recommandees :")
	fmt.Println("  1. Configurer vos cles API")
	fmt.Println("  2. Lancer votre premier profil")
	fmt.Println()
	fmt.Print("Commencer la configuration ? (O/n) : ")

	choice, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		logging.Error("wizard input failed: %v", err)
		return
	}
	if err == io.EOF && len(choice) == 0 {
		// A non-interactive npx/CI invocation has no answer. Do not interpret
		// EOF as the default "yes", and do not suppress the next real welcome.
		fmt.Println()
		return
	}
	choice = strings.TrimSpace(strings.ToLower(choice))

	if choice == "" || choice == "o" || choice == "y" {
		fmt.Println()
		if err := config.InteractiveConfig(profiles, nil); err != nil {
			logging.Error("wizard config failed: %v", err)
		}
		fmt.Println()
		display.PrintSuccess("Configuration terminee !")
		fmt.Println()
		fmt.Println("Pour lancer un profil : multiai launch -p <shortcut>")
		fmt.Println("Pour voir les profils  : multiai list")
		fmt.Println("Menu interactif        : multiai")
	}

	// Mark first run as done
	markFirstRunDone()
}

// firstRunMarkerPath returns the file recording that the welcome wizard
// already ran once (completed or declined). Empty when no home dir exists.
func firstRunMarkerPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".multiai", ".first-run-done")
}

// FirstRunMarkerExists reports whether the welcome wizard already ran.
// Callers combine it with IsFirstRun: the wizard shows only when no key is
// configured AND this marker is absent (an existing marker is respected).
func FirstRunMarkerExists() bool {
	path := firstRunMarkerPath()
	if path == "" {
		return true // no home dir: never nag, never re-prompt
	}
	_, err := os.Stat(path)
	return err == nil
}

func markFirstRunDone() {
	path := firstRunMarkerPath()
	if path == "" {
		return
	}
	// Best-effort: a failed marker write must never break the CLI.
	_ = os.MkdirAll(filepath.Dir(path), 0700)
	_ = os.WriteFile(path, []byte("1"), 0600)
}

func isSecretLike(key string) bool {
	upper := strings.ToUpper(key)
	for _, p := range []string{"KEY", "TOKEN", "SECRET", "AUTH"} {
		if strings.Contains(upper, p) {
			return true
		}
	}
	return false
}
