# multiai — Routeur multi-IA

> **Un seul outil pour lancer Claude Code, Codex CLI et OpenCode — avec des profils d'environnement isolés par fournisseur.**

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-0078D4)](#installation)
[![Go](https://img.shields.io/badge/Go-1.23-blue)](https://go.dev)
[![npm](https://img.shields.io/npm/v/multiai)](https://www.npmjs.com/package/multiai)
[![Score](https://img.shields.io/badge/score-10%2F10-success)]()

```bash
npx multiai install
```

---

## Le problème

Tu jongles entre **Claude Code**, **Codex CLI** et **OpenCode** avec **5+ fournisseurs** d'API — Anthropic, DeepSeek, Z.ai, OpenAI, OpenRouter.

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

Choix : 1

Outils disponibles

0. Retour au menu principal
1. Claude Code (6 profils)
2. Codex CLI (3 profils)
3. OpenCode (8 profils)

Choisis un outil : 1

Profils disponibles pour Claude Code

0. Retour a la selection d'outil
1. Claude Code officiel [co]
2. Anthropic API officielle [ca]
3. Z.ai GLM-5.2 Coding Plan [cg]
4. DeepSeek V4 Pro 1M [ds]
5. DeepSeek V4 Flash [dsf]

Choisis un profil : 4

Lancement : claude
```

---

## Installation

| Méthode | Commande |
|---------|----------|
| **npm** | `npx multiai install` |
| **Go** | `go install github.com/lrochetta/multiai@latest` |
| **Homebrew** | `brew install lrochetta/tap/multiai` |
| **Scoop** | `scoop install multiai` |
| **Script** | `curl -fsSL https://rochetta.fr/multiai/install.sh \| bash` |

---

## Usage rapide

```bash
multiai                        # Menu interactif
multiai launch -p ds           # DeepSeek V4 Pro via Claude Code
multiai launch -p codex55      # Codex GPT-5.5
multiai list --json            # Liste tous les profils en JSON
multiai config                 # Configurer les clés API
multiai completion bash        # Autocomplétion bash
```

---

## 17 profils inclus

| Shortcut | Tool | Provider |
|----------|------|----------|
| `co` | Claude Code | — |
| `ca` | Claude Code | Anthropic |
| `cg`, `cgalt` | Claude Code | Z.ai GLM-5.2 |
| `ds`, `dsf` | Claude Code | DeepSeek |
| `codex55`, `codex54`, `codexmini` | Codex CLI | OpenAI |
| `ocdefault`, `ocopenai`, `ocanthropic`, `ocdeepseek`, `oczai`, `ocqwen`, `ockimi`, `ocminimax` | OpenCode | OpenAI, Anthropic, DeepSeek, Z.ai, OpenRouter |

---

## Fonctionnalités

### 🚀 Lancement unifié
- Menu interactif avec navigation complète (retour à chaque niveau)
- Lancement direct : `multiai launch -p ds`
- Mode dry-run : `--dry-run` pour simuler sans lancer
- Sortie JSON : `--json` pour l'intégration scriptée

### 🔐 Sécurité
- **Isolation par liste blanche** : seul ~30 variables système survivent
- **Credential store natif** : AES-256-GCM + Windows/macOS/Linux
- **Whitelist des commandes** : `claude`, `codex`, `opencode` uniquement
- **Anti-fuite npm** : `prepublishOnly` scan les `.env`

### 🖥️ Cross-platform
Windows amd64 • macOS Intel • macOS Apple Silicon • Linux amd64/arm64

### 🔧 Extensibilité
- Profils YAML + `.multiai.yaml` par projet avec héritage
- Plugin hooks `before_launch` / `after_launch`
- Shell completion bash, zsh, fish, PowerShell

### 🧠 BMAD+ intégré
- Détection automatique, version, packs
- Menu mise à jour (latest, version spécifique, réinstall, reset)

---

## Structure du projet

```
multiai-go/          → Go (primaire, cross-platform)
├── cmd/multiai/     → CLI 7 sous-commandes
├── internal/        → cli, config, env, menu, profile, secret
├── pkg/dotenv/      → Parser .env
├── configs/profiles/ → 17 profils
├── docs/            → Site VitePress 16 pages
├── packaging/       → brew, scoop, deb, aur, npm
├── tests/           → 45+ tests
└── scripts/         → setup, install

code-cli-router-pack/ → PowerShell (legacy maintenu, distribué via npm)
```

---

## Qualité

- **45+ tests** (unitaires, intégration, benchmark)
- **CI/CD** : lint → test (6 OS × Go) → security → build → benchmark
- **go vet** : 0 warning
- **Couverture** : dotenv 93.9%, env 96.0%

---

MIT — [Laurent Rochetta](https://rochetta.fr) • [follow.ovh/bio/laurent](https://follow.ovh/bio/laurent) • [github.com/lrochetta/multiai](https://github.com/lrochetta/multiai)
