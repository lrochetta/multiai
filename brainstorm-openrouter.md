# 🧠 Brainstorming — Intégration OpenRouter dans multiai v0.3.0

> Date : 2026-06-23

## Vision

OpenRouter donne accès à **300+ modèles** via une API unique. multiai peut devenir le routeur ultime en découvrant dynamiquement les modèles disponibles, triés par popularité, avec **Fusion** en premier.

---

## 1. État des lieux OpenRouter

### API de découverte des modèles
```
GET https://openrouter.ai/api/v1/models
Headers: Authorization: Bearer OPENROUTER_API_KEY
```

### Paramètres de tri disponibles
| Paramètre | Description |
|-----------|------------|
| `sort=top-weekly` | Plus utilisés cette semaine |
| `sort=most-popular` | Plus populaires |
| `sort=pricing-low-to-high` | Moins chers d'abord |
| `sort=context-high-to-low` | Plus grand contexte |
| `category=programming` | Modèles coding |
| `category=tool-calling` | Modèles tool use |
| `max_price=0` | Gratuits uniquement |
| `arch=Claude` | Architecture spécifique |

### `openrouter/fusion` — Le modèle à mettre en avant
- **URL** : https://openrouter.ai/fusion
- **Concept** : "Small multi-model deliberation" — panel d'experts qui débat et synthétise
- **Fonctionnement** : Envoie la requête à plusieurs modèles en parallèle, agrège les réponses, et produit une synthèse optimale
- **Dashboard** : Interface web pour voir l'historique des "runs" Fusion
- **Contexte** : 1,000,000 tokens
- **Pricing** : Variable (routing dynamique)
- **Positionnement** : Choix intelligent du meilleur modèle sans que l'utilisateur ait à décider

---

## 2. Architecture proposée : Profils OpenRouter dynamiques

### 2.1 Nouvelle commande : `multiai models`

```bash
# Lister les modèles OpenRouter (depuis l'API, avec cache)
multiai models

# Filtrer
multiai models --category programming
multiai models --free
multiai models --sort top-weekly
multiai models --limit 20

# JSON
multiai models --json
```

### 2.2 Cache intelligent
- **Durée** : 1 heure par défaut (configurable)
- **Stockage** : `~/.multiai/cache/openrouter-models.json`
- **Fallback** : Si API inaccessible, utiliser le cache

### 2.3 Structure de profil dynamique

```yaml
# configs/profiles/openrouter-fusion.yaml
id: openrouter-fusion
shortcut: or-fusion
tool: claude
display_name: "OpenRouter Fusion (Auto)"
description: "Panel d'experts multi-modèles — le meilleur modèle choisi automatiquement"
order: 5
provider: openrouter
dynamic: true
api_endpoint: https://openrouter.ai/api/v1
env:
  OPENROUTER_API_KEY: __MULTIAI_CREDSTORE__
```

### 2.4 Génération de profils à la demande

```bash
# Créer un profil depuis n'importe quel modèle OpenRouter
multiai models --add or-fusion    # Ajoute Fusion aux profils
multiai models --add deepseek/v4  # Ajoute DeepSeek V4
multiai models --add --top 5      # Ajoute les 5 modèles les plus populaires
```

---

## 3. Interface UI/UX

### 3.1 Menu principal → nouvelle option

```
Laurent ROCHETTA's MultiAI (AI Code CLI Router)
----------------------------------------------------------

1. Lancer
2. Configurer les cles API
3. BMAD+ -- Gestion du framework
4. OpenRouter -- Decouvrir les modeles    ← NOUVEAU

Choix :
```

### 3.2 Menu OpenRouter

```
OpenRouter — 300+ modeles disponibles
----------------------------------------------------------

🌟 Recommandés pour toi (top weekly) :
  1. MiniMax M3                    $0.30/M  | Agentic, multimodal
  2. DeepSeek V4 Flash             $0.09/M  | Fast coding
  3. Xiaomi MiMo-V2.5              $0.14/M  | Agentic, multimodal
  4. Claude Opus 4.7               $5/M     | Long-running agents
  5. Owl Alpha                     GRATUIT  | Agentic workloads

🔥 Fusion : Choix automatique du meilleur modèle (panel d'experts)

Filtres :
  f. Gratuits uniquement
  c. Coding / Programming
  t. Tool calling
  s. Rechercher un modèle...

  0. Retour
  a. Ajouter un modèle à mes profils
```

### 3.3 Affichage d'un modèle

```
DeepSeek V4 Flash
─────────────────────────────────────────
Fournisseur  : DeepSeek (via OpenRouter)
Prix         : $0.09/M input | $0.18/M output
Contexte     : 1,048,576 tokens
Catégories   : coding, tool-calling, agents
Description  : Fast coding model optimized for high-throughput tasks

Actions :
  1. Lancer avec ce modèle
  2. Ajouter à mes profils permanents
  0. Retour
```

---

## 4. Top 10 modèles à intégrer (ordre par défaut)

Basé sur les données OpenRouter de juin 2026 :

| # | Modèle | Prix Input | Pourquoi |
|---|--------|-----------|----------|
| 🌟 | **Fusion** | Variable | Choix auto du meilleur modèle |
| 1 | MiniMax M3 | $0.30 | Agentic multimodal, #1 usage |
| 2 | DeepSeek V4 Flash | $0.09 | Fast coding, rapport qualité/prix |
| 3 | Xiaomi MiMo-V2.5 | $0.14 | Agentic multimodal |
| 4 | Claude Opus 4.8 | $5 | Plus capable, multi-step |
| 5 | Claude Sonnet 4.6 | $3 | Coding, agents, pro |
| 6 | DeepSeek V4 Pro | $0.435 | Raisonnement avancé |
| 7 | Owl Alpha | Gratuit | Agentic, gratuit |
| 8 | NVIDIA Nemotron 3 Ultra | Gratuit | 55B params, gratuit |
| 9 | Qwen 3.7 Plus | $0.32 | 1M contexte |
| 10 | Gemini 3 Flash | $0.50 | Agentic multimodal |

---

## 5. Plan d'implémentation

### Phase A — Fondation (2h)
- [ ] Nouveau package Go : `internal/openrouter/`
- [ ] Fonction `FetchModels(apiKey, sort, category)` → `[]Model`
- [ ] Cache JSON 1h dans `~/.multiai/cache/`
- [ ] Commande `multiai models`

### Phase B — Intégration (2h)
- [ ] Menu OpenRouter dans le menu principal
- [ ] Affichage formaté des modèles (nom, prix, contexte, catégories)
- [ ] Tri avec **Fusion toujours en premier**
- [ ] Filtres (gratuit, coding, tool-calling)

### Phase C — Profils dynamiques (2h)
- [ ] `multiai models --add <model-id>` crée un profil `.yaml`
- [ ] `multiai models --add --top 5` ajoute les 5 plus populaires
- [ ] Les profils dynamiques apparaissent dans `multiai list` et le menu launch

---

## 7. Recherche avancée de modèles

### 7.1 Commande `multiai search`

```bash
# Recherche par mot-clé (nom du modèle ou fournisseur)
multiai search "claude"          # Tous les modèles contenant "claude"
multiai search "deepseek"        # Tous les modèles DeepSeek
multiai search "gratuit"         # Modèles gratuits

# Recherche par fournisseur (filtre exact)
multiai search --provider anthropic
multiai search --provider deepseek
multiai search --provider openai

# Recherche avec filtres combinés
multiai search "flash" --provider deepseek --max-price 0.50
multiai search --free --category coding
multiai search --min-context 1000000 --sort pricing-low-to-high

# Format
multiai search "claude" --json
multiai search "claude" --json | jq '.[] | {id, name, pricing}'
```

### 7.2 API OpenRouter utilisée

```
GET https://openrouter.ai/api/v1/models?q=<recherche>
```

Le paramètre `q` de l'API OpenRouter fait une recherche full-text sur :
- Nom du modèle (`deepseek-v4-pro`)
- Slug du modèle (`deepseek/deepseek-v4-pro`)
- Nom du fournisseur (`DeepSeek`)
- Architecture (`GPT`, `Claude`, `Gemini`)

### 7.3 Menu interactif de recherche

```
Recherche de modeles OpenRouter
----------------------------------------------------------
Rechercher : claude▊

Résultats pour "claude" (8 trouvés) :

  1. Anthropic: Claude Opus 4.8          $5/M     | 1M ctx | ★★★★☆
  2. Anthropic: Claude Sonnet 4.6        $3/M     | 1M ctx | ★★★★☆
  3. Anthropic: Claude Opus 4.7          $5/M     | 1M ctx | ★★★☆☆
  4. Anthropic: Claude Fable 5           $10/M    | 1M ctx | ★★★★★
  5. Anthropic: Claude Fable Latest      Variable | 1M ctx | —
  6. OpenRouter: Claude via OpenRouter   Variable | 1M ctx | —
  7. Claude Opus 4.8 (Fast)              $10/M    | 1M ctx | ★★★★☆
  8. Claude Opus 4.7 (Fast)              $30/M    | 1M ctx | ★★★☆☆

Filtres disponibles :
  f. Gratuits uniquement
  c. Coding / Programming
  t. Tool calling
  p. Par prix max
  s. Nouvelle recherche

Actions par modèle :
  [1-8] Détail du modèle
  0. Retour
```

### 7.4 Détail d'un modèle (après sélection)

```
Anthropic: Claude Opus 4.8
──────────────────────────────────────────────────────────
ID           : anthropic/claude-opus-4.8
Fournisseur  : Anthropic (via OpenRouter)
Prix         : $5.00/M input | $25.00/M output
Contexte     : 1,000,000 tokens
Architecture : Claude
Catégories   : coding, tool-calling, agents, reasoning
Description  : Most capable Claude model for multi-step reasoning,
               long-running agents, and complex code generation.

🌟 Note communauté : ★★★★☆ (4.2/5 — basé sur 12 847 avis)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Prix estimé pour une session typique (10K input / 2K output) :
  → ~$0.10 par requête
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Actions :
  1. Lancer avec ce modèle (one-shot)
  2. Ajouter à mes profils permanents
  3. Comparer avec un autre modèle
  0. Retour aux résultats
```

### 7.5 Recherche par fournisseur

```
Recherche par fournisseur
----------------------------------------------------------
Fournisseurs disponibles (via OpenRouter) :

  Anthropic     → 12 modèles   (Claude Opus, Sonnet, Haiku, Fable)
  Google        → 18 modèles   (Gemini Pro, Flash, Nano)
  OpenAI        → 15 modèles   (GPT-5.5, GPT-5.4, GPT-OSS)
  DeepSeek      → 6 modèles    (V4 Pro, V4 Flash, V3.2)
  Qwen          → 8 modèles    (Qwen3.7 Max, Plus, Flash)
  Meta          → 5 modèles    (Llama 4, Llama 3)
  Mistral       → 4 modèles    (Medium, Large, Small)
  xAI           → 3 modèles    (Grok 4.3, Grok Build)
  NVIDIA        → 4 modèles    (Nemotron 3 Ultra, Nano)
  Cohere        → 3 modèles    (North, Command)
  MiniMax       → 2 modèles    (M3, M2)
  Kimi          → 3 modèles    (K2.7, K2.6, K2.5)
  Z.ai          → 2 modèles    (GLM 5.2, GLM 5.1)
  OpenRouter    → 2 modèles    (Fusion, Owl Alpha)
  ...et 50+ autres fournisseurs

Tape le nom d'un fournisseur (ou 0 pour retour) : anthropic▊

Modèles Anthropic via OpenRouter :
  1. Claude Opus 4.8           $5/M     | 1M ctx | coding, agents
  2. Claude Sonnet 4.6         $3/M     | 1M ctx | coding, agents
  3. Claude Opus 4.7           $5/M     | 1M ctx | agents
  4. Claude Fable 5            $10/M    | 1M ctx | frontier
  5. Claude Haiku 4.5          $1/M     | 500K   | fast, cheap
  ...
  0. Retour
```

### 7.6 Comparaison de modèles

```
Comparaison : Claude Opus 4.8 vs DeepSeek V4 Pro
──────────────────────────────────────────────────────────
Critère              │ Opus 4.8          │ DeepSeek V4 Pro
──────────────────────┼───────────────────┼────────────────
Prix input (1M tok)  │ $5.00             │ $0.44
Prix output (1M tok) │ $25.00            │ $0.87
Contexte             │ 1 000 000         │ 1 048 576
Architecture         │ Claude            │ DeepSeek
Coding               │ ★★★★★ (5.0)      │ ★★★★☆ (4.2)
Agents               │ ★★★★★ (4.9)      │ ★★★★☆ (4.0)
Reasoning            │ ★★★★★ (5.0)      │ ★★★★★ (4.8)
Vitesse              │ ★★★☆☆ (3.5)      │ ★★★★★ (4.8)
──────────────────────┴───────────────────┴────────────────
Recommandé pour :      Tâches complexes    Usage quotidien
                       Haute précision     Budget serré
                       Agents long terme   Fast coding

  1. Lancer Opus 4.8
  2. Lancer DeepSeek V4 Pro
  0. Retour
```

### 7.7 Implémentation technique

```go
// internal/openrouter/search.go
package openrouter

type SearchParams struct {
    Query      string  // free-text search
    Provider   string  // filter by provider slug
    Category   string  // programming, tool-calling, vision, roleplay
    MaxPrice   float64 // max input price $/M tokens
    FreeOnly   bool    // free models only
    MinContext int     // minimum context length
    Sort       string  // top-weekly, most-popular, pricing-low-to-high, newest
    Limit      int     // max results (default 20)
}

type ModelResult struct {
    ID             string  `json:"id"`
    Name           string  `json:"name"`
    Provider       string  `json:"provider"`
    ProviderName   string  `json:"provider_name"`
    ContextLength  int     `json:"context_length"`
    Pricing        Pricing `json:"pricing"`
    Architecture   string  `json:"architecture"`
    Categories     []string `json:"categories"`
    Description    string  `json:"description"`
    IsFree         bool    `json:"is_free"`
    WeeklyRank     int     `json:"weekly_rank,omitempty"`
}

type Pricing struct {
    Input      string `json:"input"`       // "$0.44/M"
    Output     string `json:"output"`      // "$0.87/M"
    InputUSD   float64 `json:"input_usd"`
    OutputUSD  float64 `json:"output_usd"`
}

// FetchModels retrieves models from OpenRouter API with search/filter.
func FetchModels(apiKey string, params SearchParams) ([]ModelResult, error)

// GetProviders returns a deduplicated list of available providers.
func GetProviders(apiKey string) ([]Provider, error)

// SearchInteractive runs the interactive search TUI.
func SearchInteractive(apiKey string, cacheDir string) error
```

---

## 8. Résumé des commandes OpenRouter

| Commande | Description |
|----------|------------|
| `multiai models` | Top modèles (triés par usage hebdomadaire) |
| `multiai models --json` | Liste complète en JSON |
| `multiai search "claude"` | Recherche par mot-clé |
| `multiai search --provider anthropic` | Tous les modèles d'un fournisseur |
| `multiai search "claude" --free` | Modèles Claude gratuits |
| `multiai search --category coding --sort pricing` | Moins chers pour le code |
| `multiai providers` | Liste des fournisseurs disponibles |
| `multiai compare opus-4.8 deepseek-v4` | Comparaison côte à côte |
| `multiai models --add deepseek-v4-pro` | Ajouter un modèle aux profils |
| `multiai models --add --top 5` | Ajouter les 5 plus populaires |

## 6. Sécurité

- **Clé API** : stockée dans le credential store (pas en clair)
- **Rate limiting** : max 1 appel API / heure (cache)
- **Timeout** : 10 secondes sur l'appel API
- **Fallback** : cache offline si API inaccessible
