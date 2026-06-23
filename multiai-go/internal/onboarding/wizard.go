package onboarding

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lrochetta/multiai/internal/cli"
	"github.com/lrochetta/multiai/internal/config"
	"github.com/lrochetta/multiai/internal/logging"
	"github.com/lrochetta/multiai/internal/profile"
	"github.com/lrochetta/multiai/pkg/dotenv"
)

// IsFirstRun returns true if no profiles have configured API keys.
func IsFirstRun(profiles []profile.Profile) bool {
	for _, p := range profiles {
		for k, v := range p.Env {
			if isSecretLike(k) && !dotenv.IsPlaceholder(v) && v != "__MULTIAI_CREDSTORE__" {
				return false // At least one key is configured
			}
		}
	}
	return true
}

// RunWelcome displays the welcome wizard and optionally configures keys.
func RunWelcome(profiles []profile.Profile) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	cli.PrintInfo("========================================")
	cli.PrintInfo("  Bienvenue dans multiai !")
	cli.PrintInfo("========================================")
	fmt.Println()
	fmt.Println("Il semble que ce soit votre premiere utilisation.")
	fmt.Printf("  %d profils disponibles\n", len(profiles))
	fmt.Println("  5 fournisseurs : Anthropic, DeepSeek, Z.ai, OpenAI, OpenRouter")
	fmt.Println()
	fmt.Println("Etapes recommandees :")
	fmt.Println("  1. Configurer vos cles API")
	fmt.Println("  2. Lancer votre premier profil")
	fmt.Println()
	fmt.Print("Commencer la configuration ? (O/n) : ")

	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(strings.ToLower(choice))

	if choice == "" || choice == "o" || choice == "y" {
		fmt.Println()
		if err := config.InteractiveConfig(profiles); err != nil {
			logging.Error("wizard config failed: %v", err)
		}
		fmt.Println()
		cli.PrintSuccess("Configuration terminee !")
		fmt.Println()
		fmt.Println("Pour lancer un profil : multiai launch -p <shortcut>")
		fmt.Println("Pour voir les profils  : multiai list")
		fmt.Println("Menu interactif        : multiai")
	}

	// Mark first run as done
	markFirstRunDone()
}

func markFirstRunDone() {
	home, _ := os.UserHomeDir()
	markerDir := filepath.Join(home, ".multiai")
	os.MkdirAll(markerDir, 0700)
	os.WriteFile(filepath.Join(markerDir, ".first-run-done"), []byte("1"), 0600)
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
