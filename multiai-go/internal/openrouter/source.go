package openrouter

import (
	"fmt"
	"os"
	"time"
)

// Source identifies where a model catalog came from. The string values are
// stable identifiers exposed in the --json output.
type Source string

const (
	// SourceNetwork means the models were just fetched from the API.
	SourceNetwork Source = "reseau"
	// SourceCache means a fresh local cache (younger than CacheTTL) was used.
	SourceCache Source = "cache"
	// SourceStale means an expired local cache was used (offline or API down).
	SourceStale Source = "cache-perime"
	// SourceEmbedded means the small static list shipped in the binary was
	// used (no network and no cache). Data is limited: no prices, no context.
	SourceEmbedded Source = "embarque"
)

// Catalog is a model list plus provenance metadata, so callers can be honest
// about how fresh (or degraded) the data is.
type Catalog struct {
	Models    []ModelInfo
	Source    Source
	FetchedAt time.Time // zero for SourceEmbedded
	Warning   string    // human-readable notice when the source is degraded
}

// SourceLabel returns a short human-readable description of the source.
func (c *Catalog) SourceLabel() string {
	switch c.Source {
	case SourceNetwork:
		return "reseau OpenRouter"
	case SourceCache:
		return fmt.Sprintf("cache local du %s", c.FetchedAt.Local().Format("2006-01-02 15:04"))
	case SourceStale:
		return fmt.Sprintf("cache local perime du %s", c.FetchedAt.Local().Format("2006-01-02 15:04"))
	case SourceEmbedded:
		return fmt.Sprintf("liste statique embarquee (%d modeles, donnees limitees)", len(c.Models))
	}
	return string(c.Source)
}

// GetModels returns the best available model catalog. It never fails: when
// the network and the cache are both unavailable it falls back to the small
// embedded list, with Warning explaining the degradation.
//
// Resolution order:
//   - offline: cache (fresh or stale) -> embedded list
//   - online:  fresh cache -> network (then cache write) -> stale cache
//     -> embedded list
func GetModels(offline bool) *Catalog {
	cached, fetchedAt, cacheErr := LoadCache()

	if offline {
		if cacheErr == nil {
			if time.Since(fetchedAt) > CacheTTL {
				return &Catalog{
					Models: cached, Source: SourceStale, FetchedAt: fetchedAt,
					Warning: fmt.Sprintf("Cache local perime (du %s, TTL %s). Relance sans --offline pour rafraichir.",
						fetchedAt.Local().Format("2006-01-02 15:04"), CacheTTL),
				}
			}
			return &Catalog{Models: cached, Source: SourceCache, FetchedAt: fetchedAt}
		}
		return &Catalog{
			Models: embeddedModels(), Source: SourceEmbedded,
			Warning: fmt.Sprintf("Mode hors-ligne, %s : liste statique embarquee (donnees limitees). Relance sans --offline pour construire le cache.", cacheStateNote(cacheErr)),
		}
	}

	if cacheErr == nil && time.Since(fetchedAt) <= CacheTTL {
		return &Catalog{Models: cached, Source: SourceCache, FetchedAt: fetchedAt}
	}

	fetched, fetchErr := FetchModels(os.Getenv("OPENROUTER_API_KEY"))
	if fetchErr == nil {
		cat := &Catalog{Models: fetched, Source: SourceNetwork, FetchedAt: time.Now()}
		if saveErr := SaveCache(fetched); saveErr != nil {
			cat.Warning = fmt.Sprintf("Cache local non ecrit (%v) : la prochaine execution refera la requete.", saveErr)
		}
		return cat
	}

	if cacheErr == nil {
		return &Catalog{
			Models: cached, Source: SourceStale, FetchedAt: fetchedAt,
			Warning: fmt.Sprintf("%v : cache local du %s utilise.", fetchErr, fetchedAt.Local().Format("2006-01-02 15:04")),
		}
	}

	return &Catalog{
		Models: embeddedModels(), Source: SourceEmbedded,
		Warning: fmt.Sprintf("%v, et %s : liste statique embarquee (donnees limitees).", fetchErr, cacheStateNote(cacheErr)),
	}
}

// cacheStateNote describes why the local cache was not usable, distinguishing
// an absent cache from a present-but-corrupt one so the message is honest.
func cacheStateNote(cacheErr error) string {
	switch {
	case cacheErr == nil:
		return "cache local ignore"
	case os.IsNotExist(cacheErr):
		return "aucun cache local"
	default:
		return fmt.Sprintf("cache local illisible (%v)", cacheErr)
	}
}

// fallbackModels is the static list shown when neither the network nor the
// cache is available. It mirrors the "modeles recommandes" screen of the
// PowerShell reference (code-router.ps1 L911-921); prices and context are
// unknown except for the two models the reference marks as free.
var fallbackModels = []ModelInfo{
	{ID: "openrouter/fusion", Name: "Fusion", Description: "Panel multi-modele (deja installe)"},
	{ID: "deepseek/deepseek-v4-pro", Name: "DeepSeek V4 Pro", Description: "DeepSeek V4 Pro"},
	{ID: "anthropic/claude-sonnet-4.6", Name: "Claude Sonnet 4.6", Description: "Claude Sonnet 4.6"},
	{ID: "openai/gpt-5.5", Name: "GPT-5.5", Description: "GPT-5.5"},
	{ID: "minimax/minimax-m3", Name: "MiniMax M3", Description: "MiniMax M3 (populaire)"},
	{ID: "qwen/qwen3.7-plus", Name: "Qwen 3.7 Plus", Description: "Qwen 3.7 Plus"},
	{ID: "google/gemini-3.5-flash", Name: "Gemini 3.5 Flash", Description: "Gemini 3.5 Flash"},
	{ID: "x-ai/grok-4.3", Name: "Grok 4.3", Description: "Grok 4.3"},
	{ID: "nvidia/nemotron-3-ultra", Name: "Nemotron 3 Ultra", Description: "Nemotron 3 Ultra (GRATUIT)",
		Pricing: ModelPricing{Prompt: "0", Completion: "0"}},
	{ID: "openrouter/owl-alpha", Name: "Owl Alpha", Description: "Owl Alpha (GRATUIT)",
		Pricing: ModelPricing{Prompt: "0", Completion: "0"}},
}

// embeddedModels returns a copy of the static fallback list, so callers can
// sort or filter without mutating the package-level slice.
func embeddedModels() []ModelInfo {
	return append([]ModelInfo(nil), fallbackModels...)
}
