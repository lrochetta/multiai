package openrouter

import (
	"context"
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
	Backend   string    // "" (OpenRouter, historical default) or "NVIDIA"
}

// backendLabel names the backend in user-facing source labels.
func (c *Catalog) backendLabel() string {
	if c.Backend == "" {
		return "OpenRouter"
	}
	return c.Backend
}

// SourceLabel returns a short human-readable description of the source.
func (c *Catalog) SourceLabel() string {
	switch c.Source {
	case SourceNetwork:
		return "reseau " + c.backendLabel()
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
func GetModels(ctx context.Context, offline bool) *Catalog {
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

	fetched, fetchErr := FetchModels(ctx, os.Getenv("OPENROUTER_API_KEY"))
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

// GetNvidiaModels returns the best available NVIDIA build.nvidia.com model
// catalog, with the same never-fail resolution order as GetModels: fresh
// cache -> network -> stale cache -> embedded list. Every hosted NVIDIA
// model is free (rate-limited ~40 req/min); the endpoint exposes no pricing
// or context metadata, so those columns render as "n/d".
func GetNvidiaModels(ctx context.Context, offline bool) *Catalog {
	cached, fetchedAt, cacheErr := LoadNvidiaCache()

	if offline {
		if cacheErr == nil {
			if time.Since(fetchedAt) > CacheTTL {
				return &Catalog{
					Models: cached, Source: SourceStale, FetchedAt: fetchedAt, Backend: "NVIDIA",
					Warning: fmt.Sprintf("Cache local perime (du %s, TTL %s). Relance sans --offline pour rafraichir.",
						fetchedAt.Local().Format("2006-01-02 15:04"), CacheTTL),
				}
			}
			return &Catalog{Models: cached, Source: SourceCache, FetchedAt: fetchedAt, Backend: "NVIDIA"}
		}
		return &Catalog{
			Models: embeddedNvidiaModels(), Source: SourceEmbedded, Backend: "NVIDIA",
			Warning: fmt.Sprintf("Mode hors-ligne, %s : liste statique embarquee (donnees limitees). Relance sans --offline pour construire le cache.", cacheStateNote(cacheErr)),
		}
	}

	if cacheErr == nil && time.Since(fetchedAt) <= CacheTTL {
		return &Catalog{Models: cached, Source: SourceCache, FetchedAt: fetchedAt, Backend: "NVIDIA"}
	}

	fetched, fetchErr := FetchNvidiaModels(ctx, os.Getenv("NVIDIA_API_KEY"))
	if fetchErr == nil {
		cat := &Catalog{Models: fetched, Source: SourceNetwork, FetchedAt: time.Now(), Backend: "NVIDIA"}
		if saveErr := SaveNvidiaCache(fetched); saveErr != nil {
			cat.Warning = fmt.Sprintf("Cache local non ecrit (%v) : la prochaine execution refera la requete.", saveErr)
		}
		return cat
	}

	if cacheErr == nil {
		return &Catalog{
			Models: cached, Source: SourceStale, FetchedAt: fetchedAt, Backend: "NVIDIA",
			Warning: fmt.Sprintf("%v : cache local du %s utilise.", fetchErr, fetchedAt.Local().Format("2006-01-02 15:04")),
		}
	}

	return &Catalog{
		Models: embeddedNvidiaModels(), Source: SourceEmbedded, Backend: "NVIDIA",
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

// nvidiaFallbackModels is the static NVIDIA list shown when neither the
// network nor the cache is available. Coding-relevant subset of the hosted
// catalog (live /v1/models, 2026-07-20); everything hosted on
// build.nvidia.com is free, hence the zero prices.
var nvidiaFallbackModels = []ModelInfo{
	{ID: "z-ai/glm-5.2", Name: "GLM 5.2 (1M ctx)", OwnedBy: "z-ai",
		Description: "Z.ai GLM 5.2 753B MoE - coding/agentic", Pricing: ModelPricing{Prompt: "0", Completion: "0"}},
	{ID: "deepseek-ai/deepseek-v4-pro", Name: "DeepSeek V4 Pro", OwnedBy: "deepseek-ai",
		Pricing: ModelPricing{Prompt: "0", Completion: "0"}},
	{ID: "deepseek-ai/deepseek-v4-flash", Name: "DeepSeek V4 Flash", OwnedBy: "deepseek-ai",
		Pricing: ModelPricing{Prompt: "0", Completion: "0"}},
	{ID: "moonshotai/kimi-k2.6", Name: "Kimi K2.6", OwnedBy: "moonshotai",
		Pricing: ModelPricing{Prompt: "0", Completion: "0"}},
	{ID: "minimaxai/minimax-m3", Name: "MiniMax M3", OwnedBy: "minimaxai",
		Pricing: ModelPricing{Prompt: "0", Completion: "0"}},
	{ID: "qwen/qwen3.5-397b-a17b", Name: "Qwen 3.5 397B", OwnedBy: "qwen",
		Pricing: ModelPricing{Prompt: "0", Completion: "0"}},
	{ID: "qwen/qwen3-next-80b-a3b-instruct", Name: "Qwen3 Next 80B", OwnedBy: "qwen",
		Pricing: ModelPricing{Prompt: "0", Completion: "0"}},
	{ID: "openai/gpt-oss-120b", Name: "GPT-OSS 120B", OwnedBy: "openai",
		Pricing: ModelPricing{Prompt: "0", Completion: "0"}},
	{ID: "nvidia/nemotron-3-super-120b-a12b", Name: "Nemotron 3 Super 120B", OwnedBy: "nvidia",
		Pricing: ModelPricing{Prompt: "0", Completion: "0"}},
	{ID: "nvidia/nemotron-3-ultra-550b-a55b", Name: "Nemotron 3 Ultra 550B", OwnedBy: "nvidia",
		Pricing: ModelPricing{Prompt: "0", Completion: "0"}},
	{ID: "mistralai/mistral-large-3-675b-instruct-2512", Name: "Mistral Large 3", OwnedBy: "mistralai",
		Pricing: ModelPricing{Prompt: "0", Completion: "0"}},
	{ID: "meta/llama-4-maverick-17b-128e-instruct", Name: "Llama 4 Maverick", OwnedBy: "meta",
		Pricing: ModelPricing{Prompt: "0", Completion: "0"}},
}

// embeddedNvidiaModels returns a copy of the static NVIDIA fallback list.
func embeddedNvidiaModels() []ModelInfo {
	return append([]ModelInfo(nil), nvidiaFallbackModels...)
}
