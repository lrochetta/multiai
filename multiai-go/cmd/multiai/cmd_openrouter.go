// cmd_openrouter.go wires the OpenRouter model discovery subcommands
// (models, search, compare) into the subcommand registry. The heavy lifting
// lives in internal/openrouter; this file only parses flags and maps results
// to exit codes:
//
//	0 success · 1 user error (bad flag, no match) · 3 output failure
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lrochetta/multiai/internal/openrouter"
)

func init() {
	register("models", cmdModels)
	register("search", cmdSearch)
	register("compare", cmdCompare)
}

// orOptions holds the flags shared by the three OpenRouter subcommands.
type orOptions struct {
	json       bool
	offline    bool
	free       bool
	help       bool
	top        int
	sortKey    string
	category   string
	vendor     string
	positional []string
}

// orParseArgs hand-parses flags in the style of the rest of the CLI
// (no flag package, so `--` passthrough conventions stay uniform).
func orParseArgs(args []string) (*orOptions, error) {
	o := &orOptions{top: 20, sortKey: openrouter.SortRecent}
	need := func(name string, i *int) (string, error) {
		*i++
		if *i >= len(args) {
			return "", fmt.Errorf("valeur manquante pour %s", name)
		}
		return args[*i], nil
	}
	for i := 0; i < len(args); i++ {
		switch a := args[i]; a {
		case "--json", "-j":
			o.json = true
		case "--offline":
			o.offline = true
		case "--free":
			o.free = true
		case "--help", "-h":
			o.help = true
		case "--top", "-n":
			v, err := need(a, &i)
			if err != nil {
				return nil, err
			}
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 {
				return nil, fmt.Errorf("valeur invalide pour %s : %q", a, v)
			}
			o.top = n
		case "--sort":
			v, err := need(a, &i)
			if err != nil {
				return nil, err
			}
			o.sortKey = v
		case "--category":
			v, err := need(a, &i)
			if err != nil {
				return nil, err
			}
			o.category = v
		case "--vendor":
			v, err := need(a, &i)
			if err != nil {
				return nil, err
			}
			o.vendor = v
		default:
			if strings.HasPrefix(a, "-") {
				return nil, fmt.Errorf("option inconnue : %s", a)
			}
			o.positional = append(o.positional, a)
		}
	}
	return o, nil
}

// orCatalog loads the model catalog and reports degraded sources on stderr,
// keeping stdout clean for --json consumers.
func orCatalog(o *orOptions) *openrouter.Catalog {
	cat := openrouter.GetModels(o.offline)
	if cat.Warning != "" {
		fmt.Fprintf(os.Stderr, "[!] %s\n", cat.Warning)
	}
	return cat
}

// orPrintJSON emits the catalog subset as indented JSON on stdout.
func orPrintJSON(cat *openrouter.Catalog, models []openrouter.ModelInfo) int {
	if models == nil {
		models = []openrouter.ModelInfo{} // JSON [] instead of null
	}
	var fetchedAt *time.Time
	if !cat.FetchedAt.IsZero() {
		t := cat.FetchedAt
		fetchedAt = &t
	}
	out := struct {
		Source    string                 `json:"source"`
		FetchedAt *time.Time             `json:"fetched_at,omitempty"`
		Count     int                    `json:"count"`
		Models    []openrouter.ModelInfo `json:"models"`
	}{string(cat.Source), fetchedAt, len(models), models}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
		return 3
	}
	return 0
}

func orPrintModelsHelp() {
	fmt.Println(`Usage:
  multiai models [options]           Top des modeles OpenRouter

Options:
  --top <n>, -n <n>    Nombre de modeles affiches (defaut : 20)
  --sort <cle>         Tri : recent | prix | contexte | nom (defaut : recent)
  --free               Ne garder que les modeles gratuits
  --category <mod>     Filtre par modalite (ex: text, image)
  --vendor <id>        Filtre par fournisseur (ex: anthropic, deepseek)
  --offline            Cache local / liste embarquee uniquement (aucun reseau)
  --json, -j           Sortie JSON

Le classement par usage n'est pas expose par l'API publique OpenRouter ;
les tris disponibles sont le prix, le contexte, la recence et le nom.
Cache : <config utilisateur>/multiai/cache/openrouter-models.json (1h).`)
}

func orPrintSearchHelp() {
	fmt.Println(`Usage:
  multiai search <termes...> [options]   Recherche full-text (id, nom, description)

Options:
  --top <n>, -n <n>    Nombre maximal de resultats (defaut : 20)
  --sort <cle>         Tri : recent | prix | contexte | nom (defaut : recent)
  --offline            Cache local / liste embarquee uniquement (aucun reseau)
  --json, -j           Sortie JSON

Exemple:
  multiai search deepseek free
Code de sortie 1 quand aucun modele ne correspond.`)
}

func orPrintCompareHelp() {
	fmt.Println(`Usage:
  multiai compare <modele1> <modele2> [options]   Comparaison cote a cote

Les modeles sont resolus par id exact, sinon par correspondance unique
sur l'id ou le nom (ex: "gpt-5.5" ou "openai/gpt-5.5").

Options:
  --offline            Cache local / liste embarquee uniquement (aucun reseau)
  --json, -j           Sortie JSON

Exemple:
  multiai compare anthropic/claude-sonnet-4.6 deepseek/deepseek-v4-pro`)
}

func cmdModels(args []string) int {
	o, err := orParseArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
		return 1
	}
	if o.help {
		orPrintModelsHelp()
		return 0
	}
	cat := orCatalog(o)
	models := cat.Models
	if o.vendor != "" {
		models = openrouter.FilterVendor(models, o.vendor)
	}
	if o.category != "" {
		models = openrouter.FilterModality(models, o.category)
	}
	if o.free {
		models = openrouter.FilterFree(models)
	}
	total := len(models)
	top, err := openrouter.Top(models, o.sortKey, o.top)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
		return 1
	}
	if o.json {
		code := orPrintJSON(cat, top)
		if code == 0 && len(top) == 0 {
			return 1 // no match: parity with the non-JSON path + documented taxonomy
		}
		return code
	}
	if len(top) == 0 {
		fmt.Fprintln(os.Stderr, "[!] Aucun modele apres filtrage.")
		return 1
	}
	openrouter.RenderModelTable(os.Stdout, top)
	fmt.Printf("\n%d/%d modele(s) - tri : %s - source : %s\n", len(top), total, o.sortKey, cat.SourceLabel())
	return 0
}

func cmdSearch(args []string) int {
	o, err := orParseArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
		return 1
	}
	if o.help {
		orPrintSearchHelp()
		return 0
	}
	if len(o.positional) == 0 {
		fmt.Fprintln(os.Stderr, "Erreur: terme de recherche manquant.")
		orPrintSearchHelp()
		return 1
	}
	query := strings.Join(o.positional, " ")
	cat := orCatalog(o)
	results := openrouter.Search(cat.Models, query)
	total := len(results)
	results, err = openrouter.Top(results, o.sortKey, o.top)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
		return 1
	}
	if o.json {
		code := orPrintJSON(cat, results)
		if code == 0 && total == 0 {
			return 1 // grep convention: no match
		}
		return code
	}
	if total == 0 {
		fmt.Fprintf(os.Stderr, "[!] Aucun modele ne correspond a : %s\n", query)
		return 1
	}
	openrouter.RenderModelTable(os.Stdout, results)
	fmt.Printf("\n%d/%d resultat(s) - source : %s\n", len(results), total, cat.SourceLabel())
	return 0
}

func cmdCompare(args []string) int {
	o, err := orParseArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
		return 1
	}
	if o.help {
		orPrintCompareHelp()
		return 0
	}
	if len(o.positional) != 2 {
		fmt.Fprintln(os.Stderr, "Erreur: il faut exactement deux modeles a comparer.")
		orPrintCompareHelp()
		return 1
	}
	cat := orCatalog(o)
	a, err := openrouter.FindModel(cat.Models, o.positional[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
		return 1
	}
	b, err := openrouter.FindModel(cat.Models, o.positional[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
		return 1
	}
	if o.json {
		return orPrintJSON(cat, []openrouter.ModelInfo{*a, *b})
	}
	openrouter.RenderComparison(os.Stdout, *a, *b)
	fmt.Printf("\nSource : %s\n", cat.SourceLabel())
	return 0
}
