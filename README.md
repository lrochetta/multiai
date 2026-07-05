# multiai — Routeur multi-IA

> **Un seul outil pour lancer Claude Code, Codex CLI et OpenCode — avec des profils d'environnement isolés par fournisseur.**

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-0078D4)](#installation)
[![Go](https://img.shields.io/badge/Go-1.23-blue)](https://go.dev)
[![npm](https://img.shields.io/npm/v/multiai)](https://www.npmjs.com/package/multiai)

```bash
npx multiai install
```

---

## Le problème

Tu jongles entre **Claude Code**, **Codex CLI** et **OpenCode** avec **14+ fournisseurs** d'API — Anthropic, DeepSeek, Z.ai, OpenAI, OpenRouter, MiniMax, StepFun, Qwen, Kimi, SiliconFlow, MiMo, Requesty, LiteLLM et plus.

Chaque CLI a besoin de variables d'environnement différentes. Un mauvais `export` et ta clé `ANTHROPIC_API_KEY` fuit dans la mauvaise session.

## La solution

**multiai** charge les bonnes clés, pour le bon CLI, pour le bon fournisseur — **dans le processus courant uniquement**. Rien ne fuit. Rien ne persiste.

```
$ multiai

Laurent ROCHETTA's MultiAI (AI Code CLI Router)
----------------------------------------------------------

1. Lancer
2. Configurer les cles API
3. BMAD+ -- Gestion du framework
4. OpenRouter -- Decouvrir les modeles

Choix : 1

Outils disponibles

0. Retour au menu principal
1. Claude Code (8 profils)
2. Codex CLI (5 profils)
3. OpenCode (10 profils)

Choisis un outil : 1

Profils disponibles pour Claude Code

0. Retour a la selection d'outil
1. Claude Code officiel [co]
2. Anthropic API officielle [ca]
3. Z.ai GLM-5.2 Coding Plan [cg]
4. DeepSeek V4 Pro 1M [ds]
5. DeepSeek V4 Flash [dsf]
6. OpenRouter Fusion [or-fusion]
7. MiniMax M3 [mm]
8. Xiaomi MiMo-V2.5 [mimo]

Choisis un profil : 6

Lancement : claude (OpenRouter Fusion — panel multi-modele)
```

---

## Installation

| Méthode | Commande | Disponibilité |
|---------|----------|---------------|
| **npm** | `npx multiai install` | ✅ maintenant (version PowerShell v0.3.0) |
| **Go** | `go install github.com/lrochetta/multiai@latest` | ✅ maintenant |
| **Homebrew** | `brew install --cask lrochetta/tap/multiai` | à partir de la release v0.4.0 (tap requis) |
| **Scoop** | `scoop install multiai` | à partir de la release v0.4.0 (bucket requis) |
| **Script** | `curl -fsSL https://rochetta.fr/multiai/install.sh \| bash` | à partir de la release v0.4.0 |

---

## Deux implémentations

| Implémentation | Version | Rôle |
|----------------|---------|------|
| `multiai-powershell/` | **v0.3.0** | Version encore distribuée sur npm (`npx multiai install`) : 37 profils, 13 fournisseurs, régions, fallback chains, journal de sessions |
| `multiai-go/` | **v0.4.0-dev** | Implémentation de référence (décision 2026-07-05) : **parité fonctionnelle atteinte** — 37 profils, 13 fournisseurs, `models`/`search`/`compare`, fallback chains, expansion `%VAR%`, credential store |

- Les fonctionnalités marquées **v0.3.0** ci-dessous (`models`/`search`/`compare`, régions, fallback, journal de sessions) sont désormais portées à l'identique dans le binaire Go.
- La version PowerShell est gelée (bugfix uniquement) jusqu'à la parité, cible **v0.4.0 unifiée** — `npx multiai install` basculera alors sur le binaire Go, puis la version PowerShell sera archivée.

---

## Usage rapide

```bash
multiai                        # Menu interactif
multiai launch -p ds           # DeepSeek V4 Pro via Claude Code
multiai launch -p or-fusion    # OpenRouter Fusion (panel multi-modele)
multiai launch -p codex55      # Codex GPT-5.5
multiai list --json            # Liste tous les profils en JSON
multiai config                 # Configurer les cles API
multiai models                 # Decouvrir les modeles OpenRouter (300+)
multiai search "claude"        # Rechercher un modele
multiai completion bash        # Autocompletion bash
```

---

## 20+ profils inclus

### Claude Code
| Shortcut | Provider |
|----------|----------|
| `co` | Claude Code officiel |
| `ca` | Anthropic API |
| `cg`, `cgalt` | Z.ai GLM-5.2 |
| `ds`, `dsf` | DeepSeek V4 |
| `or-fusion` | OpenRouter Fusion |
| `mm` | MiniMax M3 |
| `stepfun` | StepFun |
| `mimo` | Xiaomi MiMo |

### Codex CLI
| Shortcut | Provider |
|----------|----------|
| `codex55`, `codex54`, `codexmini` | OpenAI |
| `codex-fusion` | OpenRouter Fusion |
| `codex-qwen` | Qwen |
| `codex-siliconflow` | SiliconFlow |

### OpenCode
| Shortcut | Provider |
|----------|----------|
| `ocdefault`, `ocopenai` | OpenAI |
| `ocanthropic` | Anthropic |
| `ocdeepseek` | DeepSeek |
| `oczai` | Z.ai |
| `oc-fusion` | OpenRouter Fusion |
| `ocminimax` | MiniMax |
| `ocqwen` | Qwen |
| `ockimi` | Kimi |
| `ocmimo` | MiMo |

### Routing
| Shortcut | Service |
|----------|---------|
| `req-cc`, `req-codex`, `req-oc` | Requesty (load balancing) |
| `litellm` | LiteLLM |

---

## Fonctionnalités

### 🚀 Lancement unifié
- Menu interactif avec navigation complète et retour à chaque niveau
- Lancement direct : `multiai launch -p ds`
- Mode dry-run : `--dry-run` pour simuler sans lancer
- Sortie JSON : `--json` pour l'intégration scriptée

### 🔍 OpenRouter intégré (v0.3.0)
- **`multiai models`** — top modèles par usage, catégorie, prix
- **`multiai search`** — recherche full-text par mot-clé, fournisseur
- **`multiai compare`** — comparaison côte à côte de 2 modèles
- **Fusion** — panel d'experts multi-modèles avec synthèse automatique
- Cache 1h, fallback offline

### 🔐 Sécurité
- **Isolation par liste blanche** : seul ~30 variables système survivent
- **Credential store** : fichier chiffré AES-256-GCM dans `~/.config/multiai/secrets` (override : `MULTIAI_SECRETS_DIR` ; stores natifs OS Windows/macOS prévus, pas encore implémentés)
- **Whitelist des commandes** : `claude`, `codex`, `opencode` uniquement
- **Anti-fuite npm** : `prepublishOnly` scan les `.env`

### 🌍 Régions & Fallback (v0.3.0)
- Régions EU/US : en-têtes d'affichage du menu de config (regroupement par zone)
- Chaînes de fallback configurables (`FALLBACK=<shortcut>[,…]`)
- Journal de sessions (`sessions.jsonl`) : usage horodaté, **sans estimation de coût**
  (le routeur ne voit pas les tokens consommés) et sans aucun secret

### 🖥️ Cross-platform
Windows amd64 • macOS Intel • macOS Apple Silicon • Linux amd64/arm64

### 🔧 Extensibilité
- Profils YAML + `.multiai.yaml` par projet avec héritage
- Plugin hooks `before_launch` / `after_launch`
- Shell completion bash, zsh, fish, PowerShell
- Profils dynamiques : ajout/suppression de modèles OpenRouter à la volée

### 🧠 BMAD+ intégré
- Détection automatique, version, packs
- Menu mise à jour (latest, version spécifique, réinstall, reset)

---

## Structure du projet

```
multiai-go/          → Go v0.2.2 (implémentation de référence, cross-platform)
├── cmd/multiai/     → CLI 7 sous-commandes
├── internal/        → cli, config, env, menu, profile, secret, openrouter
├── pkg/dotenv/      → Parser .env
├── configs/profiles/ → 17 profils
├── packaging/       → brew, scoop, deb, aur, npm
├── tests/           → tests d'intégration
└── scripts/         → setup, install

multiai-powershell/ → PowerShell v0.3.0 (version npm actuelle, gelée hors bugfix)
```

---

## Qualité

Chiffres mesurés le 2026-07-05 sur `multiai-go/` (`go test ./... -cover`) :

- **45 fonctions de test** (43 tests + 2 benchmarks), toutes vertes
- **Couverture par package** : dotenv 93.9% · env 86.2% · secret 77.1% · assets 73.7% · profile 27.2% · config 15.2% · cli 7.1%
- **Non couverts à ce jour** (0%) : `menu`, `openrouter`, `logging`, `onboarding`, `cmd/multiai`
- **go vet** : 0 warning
- **CI/CD** : lint → test (6 OS × Go) → security → build → benchmark

---

MIT — [Laurent Rochetta](https://rochetta.fr) • [follow.ovh/bio/laurent](https://follow.ovh/bio/laurent) • [github.com/lrochetta/multiai](https://github.com/lrochetta/multiai)
