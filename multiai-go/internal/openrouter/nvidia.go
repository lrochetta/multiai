package openrouter

// NVIDIA build.nvidia.com discovery: same package as the OpenRouter
// discovery because both share ModelInfo, the cache envelope and the
// discover.go helpers. Differences vs OpenRouter:
//   - the hosted catalog is 100% free (no per-token billing at NVIDIA at
//     all; production = self-hosted NIM or DGX Cloud, licensed per GPU);
//   - /v1/models exposes no pricing/context metadata (OpenAI shape);
//   - Claude Code and Codex CLI cannot reach the endpoint directly (no
//     Anthropic /v1/messages, no /v1/responses): generated profiles for
//     those tools point at the local LiteLLM bridge on port 4000
//     (scripts/nvidia-bridge.ps1); OpenCode connects directly.

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/lrochetta/multiai/internal/fsutil"
)

// nvidiaBridgeURL is the local Anthropic/Responses translation bridge
// (LiteLLM) required by Claude Code and Codex CLI profiles.
const nvidiaBridgeURL = "http://localhost:4000"

// nvidiaKeysURL is where the user generates an nvapi- key.
const nvidiaKeysURL = "https://build.nvidia.com/settings/api-keys"

// InteractiveNvidiaMenu runs the NVIDIA discovery menu (list, search,
// dynamic profile creation). It is the entry point wired to the main
// interactive loop (option 5). It returns when the user picks "0".
func InteractiveNvidiaMenu() error {
	return runNvidiaMenu(bufio.NewReader(os.Stdin), os.Stdout)
}

// runNvidiaMenu is the testable core of InteractiveNvidiaMenu.
func runNvidiaMenu(in *bufio.Reader, out io.Writer) error {
	profilesDir, dirErr := ActiveProfilesDir()
	for {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "NVIDIA build.nvidia.com -- Modeles GRATUITS (NIM)")
		fmt.Fprintln(out, strings.Repeat("-", 58))
		fmt.Fprintln(out, "  Catalogue web : https://build.nvidia.com/models")
		fmt.Fprintln(out, "  Cle API (nvapi-...) : "+nvidiaKeysURL)
		fmt.Fprintln(out, "  Tarif : tout le catalogue heberge est GRATUIT (~40 req/min,")
		fmt.Fprintln(out, "          jusqu'a 200 sur demande). NVIDIA ne vend pas d'API")
		fmt.Fprintln(out, "          payante par token ; modeles payants -> OpenRouter (menu 4).")
		fmt.Fprintln(out, "  Claude Code : pont Anthropic->OpenAI INTEGRE (automatique, rien a installer)")
		fmt.Fprintln(out, "  Codex : pont LiteLLM local port 4000 (multiai-go/scripts/nvidia-bridge.ps1)")
		fmt.Fprintln(out, "  OpenCode : acces direct")
		if dirErr == nil {
			fmt.Fprintf(out, "  Profils : %s\n", profilesDir)
		}
		fmt.Fprintln(out)
		fmt.Fprintln(out, "1. Lister les modeles (tri nom / editeur)")
		fmt.Fprintln(out, "2. Rechercher un modele")
		fmt.Fprintln(out, "3. Creer un profil dynamique (claude/codex via pont, opencode direct)")
		fmt.Fprintln(out, "0. Retour")
		fmt.Fprintln(out)
		fmt.Fprint(out, "Choix : ")

		choice, ok := readLine(in)
		if !ok {
			return nil // EOF: behave like "back"
		}
		switch choice {
		case "0":
			return nil
		case "1":
			nvidiaMenuList(in, out)
		case "2":
			nvidiaMenuSearch(in, out)
		case "3":
			nvidiaMenuCreate(in, out, profilesDir, dirErr)
		default:
			fmt.Fprintln(out, "[!] Choix invalide. Options : 1, 2, 3, 0")
		}
	}
}

// SortNvidia returns a sorted copy of models. Valid keys: "nom" (default,
// by id) and "editeur" (by owner then id). The NVIDIA endpoint returns a
// constant created timestamp, so a "recent" sort would be meaningless.
func SortNvidia(models []ModelInfo, sortKey string) ([]ModelInfo, error) {
	out := append([]ModelInfo(nil), models...)
	switch sortKey {
	case "nom", "":
		sort.SliceStable(out, func(i, j int) bool {
			return strings.ToLower(out[i].ID) < strings.ToLower(out[j].ID)
		})
	case "editeur":
		sort.SliceStable(out, func(i, j int) bool {
			oi, oj := strings.ToLower(nvidiaOwner(out[i])), strings.ToLower(nvidiaOwner(out[j]))
			if oi != oj {
				return oi < oj
			}
			return strings.ToLower(out[i].ID) < strings.ToLower(out[j].ID)
		})
	default:
		return nil, fmt.Errorf("tri inconnu %q (valides : nom, editeur)", sortKey)
	}
	return out, nil
}

// nvidiaOwner resolves the model owner: the owned_by field when present,
// otherwise the id prefix before "/".
func nvidiaOwner(m ModelInfo) string {
	if m.OwnedBy != "" {
		return m.OwnedBy
	}
	prefix, _, _ := strings.Cut(m.ID, "/")
	return prefix
}

// RenderNvidiaModelTable writes NVIDIA models as an aligned table. The
// endpoint has no pricing/context metadata: every hosted model is free,
// stated once in the header instead of a redundant per-row column.
func RenderNvidiaModelTable(w io.Writer, models []ModelInfo) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tEDITEUR\tNOM")
	fmt.Fprintln(tw, "--\t-------\t---")
	for _, m := range models {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", m.ID, orDash(nvidiaOwner(m)), orDash(truncate(m.Name, 40)))
	}
	tw.Flush()
}

func nvidiaMenuList(in *bufio.Reader, out io.Writer) {
	fmt.Fprint(out, "Tri (nom/editeur) [nom] : ")
	sortKey, ok := readLine(in)
	if !ok {
		return
	}
	cat := GetNvidiaModels(context.Background(), false)
	models, err := SortNvidia(cat.Models, sortKey)
	if err != nil {
		fmt.Fprintf(out, "[X] %v\n", err)
		return
	}
	fmt.Fprintln(out)
	RenderNvidiaModelTable(out, models)
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%d modeles, tous GRATUITS (rate limit ~40 req/min).\n", len(models))
	printCatalogNotice(out, cat)
}

func nvidiaMenuSearch(in *bufio.Reader, out io.Writer) {
	fmt.Fprint(out, "Termes de recherche : ")
	query, ok := readLine(in)
	if !ok || query == "" {
		return
	}
	cat := GetNvidiaModels(context.Background(), false)
	results := Search(cat.Models, query)
	if len(results) == 0 {
		fmt.Fprintf(out, "[!] Aucun modele ne correspond a : %s\n", query)
		printCatalogNotice(out, cat)
		return
	}
	shown := results
	if len(shown) > 25 {
		shown = shown[:25]
	}
	fmt.Fprintln(out)
	RenderNvidiaModelTable(out, shown)
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%d resultat(s) affiches sur %d.\n", len(shown), len(results))
	printCatalogNotice(out, cat)
}

// nvidiaMenuCreate mirrors the OpenRouter quick-add flow for the NVIDIA
// backend: display name, slug, CLI, then profile generation.
func nvidiaMenuCreate(in *bufio.Reader, out io.Writer, profilesDir string, dirErr error) {
	if dirErr != nil {
		fmt.Fprintf(out, "[X] Dossier profils indisponible : %v\n", dirErr)
		return
	}
	fmt.Fprint(out, "Nom du modele (ex: GLM 5.2, vide = annuler) : ")
	name, ok := readLine(in)
	if !ok || name == "" {
		return
	}
	fmt.Fprint(out, "Slug NVIDIA (ex: z-ai/glm-5.2) : ")
	slug, ok := readLine(in)
	if !ok || slug == "" {
		return
	}
	fmt.Fprint(out, "CLI (claude/codex/opencode) [opencode] : ")
	tool, ok := readLine(in)
	if !ok {
		return
	}
	if tool == "" {
		tool = "opencode"
	}

	spec := ProfileSpec{DisplayName: name, ModelSlug: slug, Tool: tool}
	path, err := CreateNvidiaProfile(profilesDir, spec, false)
	if errors.Is(err, ErrProfileExists) {
		fmt.Fprintf(out, "[!] Le fichier existe deja : %s\n", path)
		fmt.Fprint(out, "Ecraser ? (o/N) : ")
		confirm, cok := readLine(in)
		if !cok || !strings.EqualFold(confirm, "o") {
			fmt.Fprintln(out, "Annule.")
			return
		}
		path, err = CreateNvidiaProfile(profilesDir, spec, true)
	}
	if err != nil {
		fmt.Fprintf(out, "[X] %v\n", err)
		return
	}
	fmt.Fprintf(out, "[OK] Profil cree : %s [%s] -> %s\n", strings.TrimSpace(name), NvidiaShortcut(name), filepath.Base(path))
	fmt.Fprintln(out, "     Configurer la cle : multiai config --provider nvidia")
	if tool == "claude" {
		fmt.Fprintln(out, "     Pont integre : demarre automatiquement au lancement, rien a installer.")
	}
	if tool == "codex" {
		fmt.Fprintln(out, "     Pont requis avant lancement : LiteLLM port 4000 (repo multiai : multiai-go/scripts/nvidia-bridge.ps1)")
	}
	fmt.Fprintln(out, "     Le profil apparaitra au prochain affichage de la liste.")
}

// NvidiaShortcut derives the profile shortcut from a display name:
// "nv-" + alphanumerics lowercased, truncated to 12 characters total.
func NvidiaShortcut(displayName string) string {
	s := "nv-" + strings.ToLower(shortcutStrip.ReplaceAllString(displayName, ""))
	if len(s) > 12 {
		s = s[:12]
	}
	return s
}

// NvidiaProfileFileName returns the .env file name for a display name
// ("98-<shortcut>.env"; dynamic OpenRouter profiles use "99-").
func NvidiaProfileFileName(displayName string) string {
	return "98-" + NvidiaShortcut(displayName) + ".env"
}

// RenderNvidia produces the .env content for a dynamic NVIDIA profile.
// claude and codex route through the local LiteLLM bridge (the hosted
// NVIDIA endpoint has neither the Anthropic API nor the Responses API);
// opencode connects directly with an inline provider config.
func RenderNvidia(spec ProfileSpec) (string, error) {
	// Slug check first with NVIDIA wording: the shared validate() reports
	// slug failures as OpenRouter errors, which would mislead here.
	if !slugPattern.MatchString(spec.ModelSlug) {
		return "", fmt.Errorf("slug NVIDIA invalide : %q (attendu : editeur/modele, ex. z-ai/glm-5.2)", spec.ModelSlug)
	}
	if err := spec.validate(); err != nil {
		return "", err
	}
	sc := NvidiaShortcut(spec.DisplayName)
	lines := []string{
		"PROFILE_ID=" + sc,
		"SHORTCUT=" + sc,
		"TOOL=" + spec.Tool,
		"TOOL_LABEL=" + spec.Tool,
		"DISPLAY_NAME=" + strings.TrimSpace(spec.DisplayName) + " (via NVIDIA)",
		"DESCRIPTION=NVIDIA build.nvidia.com (gratuit): " + spec.ModelSlug,
		"ORDER=51",
		"COMMAND=" + spec.Tool,
		"CLEAR_ENV=true",
		"REQUIRED_SECRETS=NVIDIA_API_KEY",
		"NVIDIA_API_KEY=PASTE_NVIDIA_API_KEY_HERE",
	}
	switch spec.Tool {
	case "claude":
		lines = append(lines,
			"NOTES=Pont Anthropic->OpenAI integre au binaire multiai (demarrage automatique)",
			"BRIDGE=anthropic-openai",
			"BRIDGE_TARGET="+nvidiaAPIBase,
			"BRIDGE_KEY_VAR=NVIDIA_API_KEY",
			"ANTHROPIC_AUTH_TOKEN=%NVIDIA_API_KEY%",
			"ANTHROPIC_MODEL="+spec.ModelSlug,
			"ANTHROPIC_API_KEY=",
		)
	case "codex":
		lines = append(lines,
			"NOTES=Pont LiteLLM requis sur le port 4000 (repo multiai : multiai-go/scripts/nvidia-bridge.ps1)",
			"ARGS=-c model_provider=nvidia -c model="+spec.ModelSlug+
				" -c model_providers.nvidia.name=NVIDIA"+
				" -c model_providers.nvidia.base_url="+nvidiaBridgeURL+
				" -c model_providers.nvidia.env_key=NVIDIA_API_KEY"+
				" -c model_providers.nvidia.wire_api=responses",
		)
	case "opencode":
		cfg, err := nvidiaOpencodeConfig(spec)
		if err != nil {
			return "", err
		}
		lines = append(lines, "OPENCODE_CONFIG_CONTENT="+cfg)
	}
	return strings.Join(lines, "\r\n") + "\r\n", nil
}

// nvidiaOpencodeConfig builds the inline OpenCode provider JSON for one
// NVIDIA model. Built with encoding/json so the display name is escaped.
func nvidiaOpencodeConfig(spec ProfileSpec) (string, error) {
	cfg := map[string]any{
		"$schema": "https://opencode.ai/config.json",
		"model":   "nvidia/" + spec.ModelSlug,
		"provider": map[string]any{
			"nvidia": map[string]any{
				"npm":  "@ai-sdk/openai-compatible",
				"name": "NVIDIA NIM (gratuit)",
				"options": map[string]any{
					"baseURL":      nvidiaAPIBase,
					"apiKey":       "{env:NVIDIA_API_KEY}",
					"timeout":      600000,
					"chunkTimeout": 120000,
				},
				"models": map[string]any{
					spec.ModelSlug: map[string]any{"name": strings.TrimSpace(spec.DisplayName)},
				},
			},
		},
		"share": "manual",
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("generation de la config OpenCode impossible: %w", err)
	}
	return string(data), nil
}

// CreateNvidiaProfile writes the generated .env into dir (created when
// missing), with the same overwrite protocol as CreateProfile. The derived
// shortcut must not be claimed by another profile of the directory: the
// bundled 84-claude-nvidia.env ships SHORTCUT=nv-cc inside the same "nv-"
// namespace, so e.g. a model named "CC" would otherwise make both profiles
// unlaunchable by shortcut ("plusieurs profils correspondent").
func CreateNvidiaProfile(dir string, spec ProfileSpec, overwrite bool) (string, error) {
	content, err := RenderNvidia(spec)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creation du dossier profils impossible: %w", err)
	}
	fileName := NvidiaProfileFileName(spec.DisplayName)
	if owner := shortcutOwner(dir, NvidiaShortcut(spec.DisplayName), fileName); owner != "" {
		return "", fmt.Errorf("le raccourci %q est deja pris par le profil %s : choisis un autre nom de modele", NvidiaShortcut(spec.DisplayName), owner)
	}
	path := filepath.Join(dir, fileName)
	if !overwrite {
		if _, statErr := os.Stat(path); statErr == nil {
			return path, fmt.Errorf("%w : %s", ErrProfileExists, path)
		}
	}
	if err := fsutil.WriteFileAtomic(path, []byte(content), 0o600); err != nil {
		return "", fmt.Errorf("ecriture du profil impossible: %w", err)
	}
	return path, nil
}

// shortcutOwner returns the file name of the .env profile in dir that
// already declares SHORTCUT=shortcut, ignoring selfName (overwriting the
// same generated file is legitimate). Empty string when the shortcut is
// free. Errors are treated as "free": the launcher re-validates uniqueness
// at load time, this is a best-effort early warning.
func shortcutOwner(dir, shortcut, selfName string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	needle := "SHORTCUT=" + shortcut
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".env") || e.Name() == selfName {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
			if strings.TrimSpace(line) == needle {
				return e.Name()
			}
		}
	}
	return ""
}
