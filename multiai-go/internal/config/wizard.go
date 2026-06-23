package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lrochetta/multiai/internal/cli"
	"github.com/lrochetta/multiai/internal/profile"
	"github.com/lrochetta/multiai/internal/secret"
	"github.com/lrochetta/multiai/pkg/dotenv"
)

// Provider represents an API key provider.
type Provider struct {
	ID        string
	Display   string
	URL       string
	Shortcuts []string
	VarMap    map[string]string
	Note      string
}

// DefaultProviders returns the built-in provider catalog.
func DefaultProviders() []Provider {
	return []Provider{
		{
			ID: "anthropic", Display: "Anthropic (officiel)",
			URL:       "https://console.anthropic.com/settings/keys",
			Shortcuts: []string{"ca", "ocanthropic"},
			VarMap:    map[string]string{"ca": "ANTHROPIC_API_KEY", "ocanthropic": "ANTHROPIC_API_KEY"},
		},
		{
			ID: "zai", Display: "Z.ai / BigModel (GLM-5.2)",
			URL:       "https://bigmodel.cn/usercenter/apikeys",
			Shortcuts: []string{"cg", "cgalt", "oczai"},
			VarMap:    map[string]string{"cg": "ANTHROPIC_AUTH_TOKEN", "cgalt": "ANTHROPIC_API_KEY", "oczai": "ZAI_API_KEY"},
			Note:      "Meme cle Z.ai pour tous les profils.",
		},
		{
			ID: "deepseek", Display: "DeepSeek",
			URL:       "https://platform.deepseek.com/api_keys",
			Shortcuts: []string{"ds", "dsf", "ocdeepseek"},
			VarMap:    map[string]string{"ds": "ANTHROPIC_AUTH_TOKEN", "dsf": "ANTHROPIC_AUTH_TOKEN", "ocdeepseek": "DEEPSEEK_API_KEY"},
			Note:      "Meme cle DeepSeek pour tous les profils.",
		},
		{
			ID: "openai", Display: "OpenAI",
			URL:       "https://platform.openai.com/api-keys",
			Shortcuts: []string{"ocopenai"},
			VarMap:    map[string]string{"ocopenai": "OPENAI_API_KEY"},
			Note:      "Codex CLI utilise son propre login - pas de cle a configurer ici.",
		},
		{
			ID: "openrouter", Display: "OpenRouter (Qwen / Kimi / MiniMax)",
			URL:       "https://openrouter.ai/settings/keys",
			Shortcuts: []string{"ocqwen", "ockimi", "ocminimax"},
			VarMap:    map[string]string{"ocqwen": "OPENROUTER_API_KEY", "ockimi": "OPENROUTER_API_KEY", "ocminimax": "OPENROUTER_API_KEY"},
		},
	}
}

// InteractiveConfig runs the interactive API key configuration.
func InteractiveConfig(profiles []profile.Profile) error {
	// Build shortcut -> profile index
	byShortcut := make(map[string]*profile.Profile)
	for i := range profiles {
		byShortcut[profiles[i].Shortcut] = &profiles[i]
	}

	providers := DefaultProviders()
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println()
		cli.PrintInfo("Configuration des cles API")
		fmt.Println(strings.Repeat("─", 58))
		fmt.Println()

		for i, prov := range providers {
			total := 0
			configured := 0
			for _, sc := range prov.Shortcuts {
				if _, ok := byShortcut[sc]; !ok {
					continue
				}
				total++
				varName := prov.VarMap[sc]
				val := byShortcut[sc].Env[varName]
				if !dotenv.IsPlaceholder(val) {
					configured++
				}
			}
			status := "[--]"
			if configured == total && total > 0 {
				status = "[OK]"
			} else if configured > 0 {
				status = "[~~]"
			}
			fmt.Printf("%d. %-36s %s (%d/%d)\n", i+1, prov.Display, status, configured, total)
			fmt.Printf("   -> %s\n", prov.URL)
		}

		fmt.Println()
		fmt.Println("a. Configurer tous les fournisseurs en sequence")
		fmt.Println("0. Retour")
		fmt.Println()
		fmt.Print("Choix : ")

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		if choice == "0" {
			return nil
		}

		if choice == "a" {
			for _, prov := range providers {
				configureProvider(prov, byShortcut)
			}
			cli.PrintSuccess("Configuration terminee.")
			return nil
		}

		idx, err := parseInt(choice)
		if err != nil || idx < 1 || idx > len(providers) {
			cli.PrintWarning("Choix invalide.")
			continue
		}
		configureProvider(providers[idx-1], byShortcut)
	}
}

func configureProvider(prov Provider, byShortcut map[string]*profile.Profile) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	cli.PrintInfo(fmt.Sprintf("  %s", prov.Display))
	fmt.Printf("  Creer une cle : %s\n", prov.URL)
	if prov.Note != "" {
		fmt.Printf("  Note : %s\n", prov.Note)
	}

	// Show current status
	var firstProf *profile.Profile
	var firstVar string
	for _, sc := range prov.Shortcuts {
		if p, ok := byShortcut[sc]; ok {
			firstProf = p
			firstVar = prov.VarMap[sc]
			break
		}
	}
	if firstProf == nil {
		cli.PrintWarning("  Aucun profil installe pour ce fournisseur.")
		return
	}

	currentVal := firstProf.Env[firstVar]
	fmt.Print("  Statut actuel : ")
	if dotenv.IsPlaceholder(currentVal) {
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

	updated := 0
	for _, sc := range prov.Shortcuts {
		p, ok := byShortcut[sc]
		if !ok {
			continue
		}
		varName := prov.VarMap[sc]
		// Update in-memory
		p.Env[varName] = newVal
		// Update file
		if err := updateEnvFile(p.Path, varName, newVal); err == nil {
			updated++
			fmt.Printf("    + %-30s [%s]\n", p.DisplayName, p.Shortcut)
		}
	}
	if updated > 0 {
		cli.PrintSuccess(fmt.Sprintf("  %d profil(s) mis a jour.", updated))
	} else {
		cli.PrintWarning("  Aucun profil mis a jour.")
	}
}

func updateEnvFile(path, varName, newValue string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(content), "\n")
	pattern := varName + "="
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") && strings.HasPrefix(trimmed, pattern) {
			// Store reference marker instead of plaintext
			lines[i] = varName + "=__MULTIAI_CREDSTORE__"
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("variable %s non trouvee dans %s", varName, path)
	}
	// Write marker to .env
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0600); err != nil {
		return err
	}
	// Store actual value in credential store
	store, err := secret.NewStore()
	if err != nil {
		return fmt.Errorf("credential store unavailable: %w", err)
	}
	profileID := strings.TrimSuffix(filepath.Base(path), ".env")
	return store.Set("multiai:"+profileID, varName, newValue)
}

func parseInt(s string) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a number: %s", s)
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}
