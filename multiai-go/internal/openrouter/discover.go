package openrouter

import (
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
)

// Sort keys accepted by Top. The public OpenRouter /models endpoint does not
// expose usage/popularity rankings, so "usage" is deliberately not offered.
const (
	SortRecent  = "recent"
	SortPrice   = "prix"
	SortContext = "contexte"
	SortName    = "nom"
)

// SortKeys lists the valid sort keys, in help/menu order.
func SortKeys() []string {
	return []string{SortRecent, SortPrice, SortContext, SortName}
}

// Top returns a sorted copy of models truncated to n entries (n <= 0 means
// no truncation). An unknown sort key is an error; "" means SortRecent.
func Top(models []ModelInfo, sortKey string, n int) ([]ModelInfo, error) {
	out := append([]ModelInfo(nil), models...)
	switch sortKey {
	case SortRecent, "":
		sort.SliceStable(out, func(i, j int) bool { return out[i].Created > out[j].Created })
	case SortPrice:
		sort.SliceStable(out, func(i, j int) bool {
			pi, ci := sortPrices(out[i])
			pj, cj := sortPrices(out[j])
			if pi != pj {
				return pi < pj
			}
			if ci != cj {
				return ci < cj
			}
			return out[i].ID < out[j].ID
		})
	case SortContext:
		sort.SliceStable(out, func(i, j int) bool { return out[i].ContextLength > out[j].ContextLength })
	case SortName:
		sort.SliceStable(out, func(i, j int) bool {
			return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
		})
	default:
		return nil, fmt.Errorf("tri inconnu %q (valides : %s)", sortKey, strings.Join(SortKeys(), ", "))
	}
	if n > 0 && len(out) > n {
		out = out[:n]
	}
	return out, nil
}

// sortPrices returns the prompt and completion per-token prices for sorting;
// unknown prices sort last.
func sortPrices(m ModelInfo) (float64, float64) {
	p, okP := parsePrice(m.Pricing.Prompt)
	c, okC := parsePrice(m.Pricing.Completion)
	if !okP {
		p = math.Inf(1)
	}
	if !okC {
		c = math.Inf(1)
	}
	return p, c
}

// Search does a case-insensitive full-text search over model id, name and
// description. All whitespace-separated terms of query must match.
func Search(models []ModelInfo, query string) []ModelInfo {
	terms := strings.Fields(strings.ToLower(query))
	if len(terms) == 0 {
		return nil
	}
	var out []ModelInfo
	for _, m := range models {
		hay := strings.ToLower(m.ID + " " + m.Name + " " + m.Description)
		match := true
		for _, t := range terms {
			if !strings.Contains(hay, t) {
				match = false
				break
			}
		}
		if match {
			out = append(out, m)
		}
	}
	return out
}

// FindModel resolves a model by exact id first (case-insensitive), then by
// unique substring match over id and name. Zero or multiple matches are
// errors with actionable messages.
func FindModel(models []ModelInfo, query string) (*ModelInfo, error) {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil, errors.New("nom de modele vide")
	}
	for i := range models {
		if strings.ToLower(models[i].ID) == q {
			m := models[i]
			return &m, nil
		}
	}
	var matches []ModelInfo
	for _, m := range models {
		if strings.Contains(strings.ToLower(m.ID), q) || strings.Contains(strings.ToLower(m.Name), q) {
			matches = append(matches, m)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("modele introuvable : %q (essaie 'multiai search %s')", query, query)
	case 1:
		m := matches[0]
		return &m, nil
	default:
		ids := make([]string, 0, 5)
		for i, m := range matches {
			if i == 5 {
				ids = append(ids, "...")
				break
			}
			ids = append(ids, m.ID)
		}
		return nil, fmt.Errorf("plusieurs modeles correspondent a %q : %s (utilise l'id exact)",
			query, strings.Join(ids, ", "))
	}
}

// FilterFree keeps only models whose prompt and completion prices both parse
// to zero. Models with unknown prices are not considered free.
func FilterFree(models []ModelInfo) []ModelInfo {
	var out []ModelInfo
	for _, m := range models {
		p, okP := parsePrice(m.Pricing.Prompt)
		c, okC := parsePrice(m.Pricing.Completion)
		if okP && okC && p == 0 && c == 0 {
			out = append(out, m)
		}
	}
	return out
}

// FilterModality keeps models whose architecture modality contains the given
// category (case-insensitive), e.g. "image" matches "text+image->text".
func FilterModality(models []ModelInfo, category string) []ModelInfo {
	cat := strings.ToLower(strings.TrimSpace(category))
	if cat == "" {
		return models
	}
	var out []ModelInfo
	for _, m := range models {
		if strings.Contains(strings.ToLower(m.Architecture.Modality), cat) {
			out = append(out, m)
		}
	}
	return out
}

// FilterVendor keeps models whose id prefix (before "/") equals the given
// vendor, case-insensitive (e.g. "anthropic", "deepseek").
func FilterVendor(models []ModelInfo, vendor string) []ModelInfo {
	v := strings.TrimSpace(vendor)
	if v == "" {
		return models
	}
	var out []ModelInfo
	for _, m := range models {
		prefix, _, _ := strings.Cut(m.ID, "/")
		if strings.EqualFold(prefix, v) {
			out = append(out, m)
		}
	}
	return out
}

// parsePrice parses a per-token USD price string from the API. Empty,
// malformed or negative (OpenRouter uses -1 for dynamic pricing) values are
// reported as unknown.
func parsePrice(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	// Reject NaN/Inf too: strconv parses "NaN"/"Inf"/"Infinity", and a NaN
	// price silently corrupts the `--sort prix` ordering (NaN comparisons).
	if err != nil || v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	return v, true
}

// FormatPricePerMTok renders a per-token price string as USD per million
// tokens: "3.00", "gratuit" or "n/d" when unknown.
func FormatPricePerMTok(perToken string) string {
	v, ok := parsePrice(perToken)
	if !ok {
		return "n/d"
	}
	if v == 0 {
		return "gratuit"
	}
	return strconv.FormatFloat(v*1e6, 'f', 2, 64)
}

// formatContext renders a context length compactly ("128k") or "n/d".
func formatContext(n int) string {
	if n <= 0 {
		return "n/d"
	}
	if n >= 1000 {
		return strconv.Itoa(n/1000) + "k"
	}
	return strconv.Itoa(n)
}

// formatCount renders a plain integer or "n/d" when zero/unknown.
func formatCount(n int) string {
	if n <= 0 {
		return "n/d"
	}
	return strconv.Itoa(n)
}

// orDash substitutes "n/d" for empty strings.
func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "n/d"
	}
	return s
}

// truncate shortens s to max runes, appending "..." when cut.
func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 3 {
		return string(r[:max])
	}
	return string(r[:max-3]) + "..."
}

// RenderModelTable writes models as an aligned table. Prices are USD per
// million tokens.
func RenderModelTable(w io.Writer, models []ModelInfo) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tCTX\tIN $/M\tOUT $/M\tNOM")
	fmt.Fprintln(tw, "--\t---\t------\t-------\t---")
	for _, m := range models {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			m.ID,
			formatContext(m.ContextLength),
			FormatPricePerMTok(m.Pricing.Prompt),
			FormatPricePerMTok(m.Pricing.Completion),
			truncate(m.Name, 40))
	}
	tw.Flush()
}

// RenderComparison writes a side-by-side comparison of two models.
func RenderComparison(w io.Writer, a, b ModelInfo) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	row := func(label, va, vb string) {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", label, va, vb)
	}
	row("", a.ID, b.ID)
	row("", strings.Repeat("-", len(a.ID)), strings.Repeat("-", len(b.ID)))
	row("Nom", orDash(a.Name), orDash(b.Name))
	row("Contexte", formatContext(a.ContextLength), formatContext(b.ContextLength))
	row("Prix entree ($/M)", FormatPricePerMTok(a.Pricing.Prompt), FormatPricePerMTok(b.Pricing.Prompt))
	row("Prix sortie ($/M)", FormatPricePerMTok(a.Pricing.Completion), FormatPricePerMTok(b.Pricing.Completion))
	row("Modalite", orDash(a.Architecture.Modality), orDash(b.Architecture.Modality))
	row("Tokenizer", orDash(a.Architecture.Tokenizer), orDash(b.Architecture.Tokenizer))
	row("Max completion", formatCount(a.TopProvider.MaxCompletionTokens), formatCount(b.TopProvider.MaxCompletionTokens))
	row("Moderation", boolFR(a.TopProvider.IsModerated), boolFR(b.TopProvider.IsModerated))
	tw.Flush()
}

// boolFR renders a boolean in French, without accents.
func boolFR(b bool) string {
	if b {
		return "oui"
	}
	return "non"
}
