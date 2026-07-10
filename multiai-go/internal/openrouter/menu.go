package openrouter

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// InteractiveMenu runs the OpenRouter discovery menu (top models, search,
// comparison, dynamic profile creation). It is the entry point wired to the
// main interactive loop (option 4). It returns when the user picks "0".
func InteractiveMenu() error {
	return runMenu(bufio.NewReader(os.Stdin), os.Stdout)
}

// runMenu is the testable core of InteractiveMenu: input and output are
// injected so tests can script the whole flow without a terminal.
func runMenu(in *bufio.Reader, out io.Writer) error {
	profilesDir, dirErr := ActiveProfilesDir()
	for {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "OpenRouter -- Decouvrir et ajouter des modeles")
		fmt.Fprintln(out, strings.Repeat("-", 58))
		fmt.Fprintln(out, "  Catalogue web : https://openrouter.ai/models")
		if dirErr == nil {
			fmt.Fprintf(out, "  Profils      : %s\n", profilesDir)
		}
		fmt.Fprintln(out)
		fmt.Fprintln(out, "1. Top modeles (prix / contexte / recent / nom)")
		fmt.Fprintln(out, "2. Rechercher un modele")
		fmt.Fprintln(out, "3. Comparer deux modeles")
		fmt.Fprintln(out, "4. Creer un profil dynamique (ajout rapide)")
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
			menuTop(in, out)
		case "2":
			menuSearch(in, out)
		case "3":
			menuCompare(in, out)
		case "4":
			menuCreate(in, out, profilesDir, dirErr)
		default:
			fmt.Fprintln(out, "[!] Choix invalide. Options : 1, 2, 3, 4, 0")
		}
	}
}

// readLine reads one trimmed line; ok is false on EOF with no data.
func readLine(in *bufio.Reader) (string, bool) {
	line, err := in.ReadString('\n')
	line = strings.TrimSpace(line)
	if err != nil && line == "" {
		return "", false
	}
	return line, true
}

// printCatalogNotice shows the degradation warning and the data source.
func printCatalogNotice(out io.Writer, cat *Catalog) {
	if cat.Warning != "" {
		fmt.Fprintf(out, "[!] %s\n", cat.Warning)
	}
	fmt.Fprintf(out, "Source : %s\n", cat.SourceLabel())
}

func menuTop(in *bufio.Reader, out io.Writer) {
	fmt.Fprintf(out, "Tri (%s) [%s] : ", strings.Join(SortKeys(), "/"), SortRecent)
	sortKey, ok := readLine(in)
	if !ok {
		return
	}
	if sortKey == "" {
		sortKey = SortRecent
	}
	cat := GetModels(context.Background(), false)
	top, err := Top(cat.Models, sortKey, 15)
	if err != nil {
		fmt.Fprintf(out, "[X] %v\n", err)
		return
	}
	fmt.Fprintln(out)
	RenderModelTable(out, top)
	fmt.Fprintln(out)
	printCatalogNotice(out, cat)
}

func menuSearch(in *bufio.Reader, out io.Writer) {
	fmt.Fprint(out, "Termes de recherche : ")
	query, ok := readLine(in)
	if !ok || query == "" {
		return
	}
	cat := GetModels(context.Background(), false)
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
	RenderModelTable(out, shown)
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%d resultat(s) affiches sur %d.\n", len(shown), len(results))
	printCatalogNotice(out, cat)
}

func menuCompare(in *bufio.Reader, out io.Writer) {
	fmt.Fprint(out, "Modele 1 (slug ou nom) : ")
	first, ok := readLine(in)
	if !ok || first == "" {
		return
	}
	fmt.Fprint(out, "Modele 2 (slug ou nom) : ")
	second, ok := readLine(in)
	if !ok || second == "" {
		return
	}
	cat := GetModels(context.Background(), false)
	a, err := FindModel(cat.Models, first)
	if err != nil {
		fmt.Fprintf(out, "[X] %v\n", err)
		return
	}
	b, err := FindModel(cat.Models, second)
	if err != nil {
		fmt.Fprintf(out, "[X] %v\n", err)
		return
	}
	fmt.Fprintln(out)
	RenderComparison(out, *a, *b)
	fmt.Fprintln(out)
	printCatalogNotice(out, cat)
}

// menuCreate is the quick-add flow of the PowerShell reference
// (Show-OpenRouterMenu L925-937): display name, slug, CLI, then profile
// generation. Divergences: the slug is validated and an existing file is
// never overwritten without confirmation.
func menuCreate(in *bufio.Reader, out io.Writer, profilesDir string, dirErr error) {
	if dirErr != nil {
		fmt.Fprintf(out, "[X] Dossier profils indisponible : %v\n", dirErr)
		return
	}
	fmt.Fprint(out, "Nom du modele (ex: DeepSeek V4 Pro, vide = annuler) : ")
	name, ok := readLine(in)
	if !ok || name == "" {
		return
	}
	fmt.Fprint(out, "Slug OpenRouter (ex: deepseek/deepseek-v4-pro) : ")
	slug, ok := readLine(in)
	if !ok || slug == "" {
		return
	}
	fmt.Fprint(out, "CLI (claude/codex/opencode) [claude] : ")
	tool, ok := readLine(in)
	if !ok {
		return
	}
	if tool == "" {
		tool = "claude"
	}

	spec := ProfileSpec{DisplayName: name, ModelSlug: slug, Tool: tool}
	path, err := CreateProfile(profilesDir, spec, false)
	if errors.Is(err, ErrProfileExists) {
		fmt.Fprintf(out, "[!] Le fichier existe deja : %s\n", path)
		fmt.Fprint(out, "Ecraser ? (o/N) : ")
		confirm, cok := readLine(in)
		if !cok || !strings.EqualFold(confirm, "o") {
			fmt.Fprintln(out, "Annule.")
			return
		}
		path, err = CreateProfile(profilesDir, spec, true)
	}
	if err != nil {
		fmt.Fprintf(out, "[X] %v\n", err)
		return
	}
	fmt.Fprintf(out, "[OK] Profil cree : %s [%s] -> %s\n", strings.TrimSpace(name), Shortcut(name), filepath.Base(path))
	fmt.Fprintln(out, "     Configurer la cle : multiai config --provider openrouter")
	fmt.Fprintln(out, "     Le profil apparaitra au prochain affichage de la liste.")
}
