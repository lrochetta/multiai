# multiai — Routeur multi-IA

> **Un seul outil pour lancer Claude Code, Codex CLI et OpenCode — avec des profils d'environnement isolés par fournisseur.**

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-0078D4)](#installation)
[![Go](https://img.shields.io/badge/Go-1.23-blue)](https://go.dev)
[![npm](https://img.shields.io/npm/v/multiai)](https://www.npmjs.com/package/multiai)
[![Score](https://img.shields.io/badge/score-10%2F10-success)]()

---

## Le problème

Tu jongles entre **Claude Code**, **Codex CLI** et **OpenCode** avec **5+ fournisseurs** d'API — Anthropic, DeepSeek, Z.ai, OpenAI, OpenRouter.

Chaque CLI a besoin de variables d'environnement différentes. Un mauvais `export` et ta clé `ANTHROPIC_API_KEY` fuit dans la mauvaise session. Tu passes ton temps à vérifier quel `.env` est chargé.

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
4. Z.ai GLM-5.2 endpoint alternatif [cgalt]
5. DeepSeek V4 Pro 1M [ds]
6. DeepSeek V4 Flash [dsf]

Choisis un profil : 5

Lancement : claude
```

---

## Fonctionnalités

### 🚀 Lancement unifié
- Menu interactif : outil → profil → lancement
- Lancement direct : `multiai launch -p ds`
- Mode dry-run : `--dry-run` pour simuler sans lancer
- Sortie JSON : `--json` pour l'intégration scriptée
- Navigation avec retour à chaque niveau

### 🔐 Sécurité
- **Isolation par liste blanche** : seul ~30 variables système survivent
- **Credential store natif** : chiffrement AES-256-GCM + Windows Credential Manager / macOS Keychain / libsecret Linux
- **Whitelist des commandes** : seuls `claude`, `codex`, `opencode` sont exécutables
- **SecureString** (PowerShell) : clés jamais en clair en mémoire
- **Vérification d'intégrité SHA256** : hash du routeur loggé
- **Anti-fuite npm** : `prepublishOnly` scan les `.env` avant publication

### 🖥️ Cross-platform
| Plateforme | Architecture | Installation |
|-----------|-------------|-------------|
| **Windows** | amd64 | `scoop install multiai` / `go install` |
| **macOS Intel** | amd64 | `brew install lrochetta/tap/multiai` |
| **macOS Apple Silicon** | arm64 | `brew install lrochetta/tap/multiai` |
| **Linux** | amd64, arm64 | `go install` / `.deb` / AUR |

### ⚡ 17 profils inclus

| Shortcut | Tool | Display Name | Provider |
|----------|------|-------------|----------|
| `co` | Claude Code | Claude Code officiel | — |
| `ca` | Claude Code | Anthropic API officielle | Anthropic |
| `cg` | Claude Code | Z.ai GLM-5.2 Coding Plan | Z.ai |
| `cgalt` | Claude Code | Z.ai GLM-5.2 endpoint alt | Z.ai |
| `ds` | Claude Code | DeepSeek V4 Pro 1M | DeepSeek |
| `dsf` | Claude Code | DeepSeek V4 Flash | DeepSeek |
| `codex55` | Codex CLI | Codex GPT-5.5 | OpenAI |
| `codex54` | Codex CLI | Codex GPT-5.4 | OpenAI |
| `codexmini` | Codex CLI | Codex GPT-5.4 mini | OpenAI |
| `ocdefault` | OpenCode | OpenCode default | — |
| `ocopenai` | OpenCode | OpenCode OpenAI GPT-5.5 | OpenAI |
| `ocanthropic` | OpenCode | OpenCode Anthropic Claude | Anthropic |
| `ocdeepseek` | OpenCode | OpenCode DeepSeek V4 Pro | DeepSeek |
| `oczai` | OpenCode | OpenCode Z.ai GLM-5.2 | Z.ai |
| `ocqwen` | OpenCode | OpenCode Qwen via OpenRouter | OpenRouter |
| `ockimi` | OpenCode | OpenCode Kimi via OpenRouter | OpenRouter |
| `ocminimax` | OpenCode | OpenCode MiniMax via OpenRouter | OpenRouter |

### 🔧 Extensibilité
- **Profils YAML** : en complément du `.env`, avec validation de schéma
- **Profils par projet** : `.multiai.yaml` avec héritage (`extends`) et surcharges (`overrides`)
- **Plugin hooks** : `before_launch` / `after_launch` avec template variables
- **Shell completion** : bash, zsh, fish, PowerShell

### 🧠 BMAD+ intégré
- Détection automatique de BMAD+ dans le projet courant
- Version et packs affichés
- Menu mise à jour : latest, version spécifique, réinstallation, reset

---

## Installation

### Quick (npm)
```bash
npx multiai install
```

### Go (recommandé)
```bash
go install github.com/lrochetta/multiai@latest
```

### Homebrew (macOS)
```bash
brew install lrochetta/tap/multiai
```

### Scoop (Windows)
```powershell
scoop bucket add lrochetta https://github.com/lrochetta/scoop-bucket
scoop install multiai
```

### Installation universelle vérifiée
```bash
npx --yes --allow-scripts=multiai multiai@0.6.6 install
```

---

## Usage

```bash
# Menu interactif
multiai

# Lancement direct par profil
multiai launch -p ds              # DeepSeek V4 Pro
multiai launch -p codex55         # Codex GPT-5.5
multiai launch -p ocanthropic     # OpenCode + Anthropic

# Liste des profils
multiai list
multiai list --json | jq .

# Configuration des clés API
multiai config

# Shell completion
multiai completion bash > /etc/bash_completion.d/multiai

# Debug / simulation
multiai launch -p ds --dry-run --json
multiai launch -p ds --show-env --no-launch

# Passer des arguments au CLI
multiai launch -p codex55 -- --ask-for-approval never
```

---

## Commandes

| Commande | Description |
|----------|------------|
| `multiai` | Menu interactif |
| `multiai launch` | Menu de lancement |
| `multiai launch -p <shortcut>` | Lancement direct |
| `multiai list` | Liste des profils |
| `multiai list --json` | Liste en JSON |
| `multiai config` | Configurer les clés API |
| `multiai completion <shell>` | Script de completion |
| `multiai version` | Afficher la version |
| `multiai help` | Aide |

### Flags

| Flag | Description |
|------|------------|
| `-p, --profile <id>` | Profil par ID ou shortcut |
| `-t, --tool <id>` | Filtrer par outil |
| `--json, -j` | Sortie JSON |
| `--dry-run` | Simulation sans lancer |
| `--no-launch` | Ne pas lancer |
| `--show-env` | Afficher l'environnement |
| `--allow-custom-command` | Autoriser commande hors whitelist |

### Codes de sortie

| Code | Signification |
|------|-------------|
| 0 | Succès |
| 1 | Erreur utilisateur (profil/clé introuvable) |
| 2 | Erreur configuration (fichier corrompu) |
| 3 | Erreur système (permissions) |
| 4 | Erreur processus enfant (crash) |

---

## Projet

### Stack
- **Go 1.23** — binaire unique cross-platform (primaire)
- **PowerShell 5.1+** — version legacy maintenue

### Structure
```
multiai-go/
├── cmd/multiai/main.go              # CLI (7 sous-commandes)
├── internal/
│   ├── cli/                         # Launcher, display, completion, hooks
│   ├── config/                      # Wizard interactif + ProviderCatalog
│   ├── env/                         # Isolation whitelist
│   ├── menu/                        # Menus interactifs
│   ├── profile/                     # .env + YAML loader, project config
│   └── secret/                      # AES-256-GCM, credential store
├── pkg/dotenv/                      # Parser .env robuste
├── configs/profiles/                # 17 profils (.env)
├── docs/                            # Site VitePress 16 pages
├── packaging/                       # brew, scoop, deb, aur, npm
├── tests/                           # Tests unitaires + intégration + benchmark
└── scripts/                         # setup-go, install universel
```

### Qualité
- **45+ tests** (unitaires, intégration, benchmark, validation)
- **CI/CD** GitHub Actions : lint → test (6 OS × Go) → security → build → benchmark
- **go vet** : 0 warning
- **Couverture** : dotenv 93.9%, env 96.0%, secret 61.2%

### Licence
MIT — [Laurent Rochetta](https://rochetta.fr) • [follow.ovh/bio/laurent](https://follow.ovh/bio/laurent)
