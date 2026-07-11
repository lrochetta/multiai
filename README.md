# multiai — Routeur multi-IA

> **Un seul outil pour lancer Claude Code, Codex CLI et OpenCode — avec des profils d'environnement isolés par fournisseur.**

<!-- Qualite et securite -->
[![Go Report Card](https://goreportcard.com/badge/github.com/lrochetta/multiai)](https://goreportcard.com/report/github.com/lrochetta/multiai)
[![Codecov](https://codecov.io/gh/lrochetta/multiai/branch/master/graph/badge.svg)](https://codecov.io/gh/lrochetta/multiai)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/lrochetta/multiai/badge)](https://securityscorecards.dev/viewer/?uri=github.com/lrochetta/multiai)
[![CI](https://github.com/lrochetta/multiai/actions/workflows/ci.yml/badge.svg)](https://github.com/lrochetta/multiai/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Cosign](https://img.shields.io/badge/signed-Cosign%20keyless-2ea44f)](https://github.com/lrochetta/multiai)

<!-- Distribution -->
[![Go Version](https://img.shields.io/badge/Go-1.22-blue)](https://go.dev)
[![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-0078D4)](#installation)
[![npm](https://img.shields.io/npm/v/multiai)](https://www.npmjs.com/package/multiai)
[![npm downloads](https://img.shields.io/npm/dm/multiai)](https://www.npmjs.com/package/multiai)

<!-- Communaute et metriques -->
[![GitHub Stars](https://img.shields.io/github/stars/lrochetta/multiai?style=social)](https://github.com/lrochetta/multiai)
[![GitHub Downloads](https://img.shields.io/github/downloads/lrochetta/multiai/total)](https://github.com/lrochetta/multiai/releases)
[![GitHub Discussions](https://img.shields.io/badge/GitHub-Discussions-181717?logo=github)](https://github.com/lrochetta/multiai/discussions)

```bash
npx multiai install
```

---

## Le problème

Tu jongles entre **Claude Code**, **Codex CLI** et **OpenCode** avec **13+ fournisseurs** d'API — Anthropic, DeepSeek, Z.ai, OpenAI, OpenRouter, MiniMax, StepFun, Qwen, Kimi, SiliconFlow, MiMo, Requesty, LiteLLM et plus.

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
1. Claude Code (15 profils)
2. Codex CLI (7 profils)
3. OpenCode (15 profils)

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
| **npm** | `npx multiai install` | ✅ v0.4.2 (binaire Go natif, SHA256 vérifié) |
| **Go** | `go install github.com/lrochetta/multiai/multiai-go/cmd/multiai@latest` | ✅ maintenant |
| **Homebrew** | `brew install --cask lrochetta/tap/multiai` | ✅ v0.4.3 (auto-upload GoReleaser) |
| **Scoop** | `scoop bucket add lrochetta https://github.com/lrochetta/scoop-bucket && scoop install lrochetta/multiai` | ✅ v0.4.3 (auto-upload GoReleaser) |
| **Script** | `curl -fsSL https://rochetta.fr/multiai/install.sh \| bash` | v0.4.2 |

---

## Implémentation

| Composant | Version | Rôle |
|-----------|---------|------|
| `multiai-go/` | **v0.4.2** | Implémentation de référence : binaire Go natif, 37 profils, 13 fournisseurs, fallback chains, credential store AES-256-GCM, menus colorés, Cosign keyless |
| `multiai-powershell/` | v0.3.0 (gelée) | Version d'origine, archivée — le package npm a basculé sur le binaire Go en v0.4.0 |

---

## Usage rapide

```bash
multiai                        # Menu interactif
multiai launch -p ds           # DeepSeek V4 Pro via Claude Code
multiai launch -p or-fusion    # OpenRouter Fusion (panel multi-modele)
multiai launch -p codex55      # Codex GPT-5.5
multiai list --json            # Liste tous les profils en JSON
multiai config                 # Configurer les cles API (menu colore vert/jaune/gris)
multiai launch -t claude        # Choisir un profil Claude Code (menu colore)
multiai models                 # Decouvrir les modeles OpenRouter (300+)
multiai search "claude"        # Rechercher un modele
multiai completion bash        # Autocompletion bash
```

---

## 37 profils inclus

### Claude Code (15 profils)
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
| `req-cc` | Requesty EU |
| `litellm` | LiteLLM |

### Codex CLI (7 profils)
| Shortcut | Provider |
|----------|----------|
| `codex55`, `codex54`, `codexmini` | OpenAI |
| `codex-fusion` | OpenRouter Fusion |
| `codex-qwen` | Qwen |
| `codex-siliconflow` | SiliconFlow |
| `req-codex` | Requesty EU |

### OpenCode (15 profils)
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
| `req-oc` | Requesty EU |

---

## Raccourcis directs (wrappers)

Génère un exécutable par profil pour lancer `multiai launch -p <shortcut>` sans taper la commande complète.

```bash
# Générer tous les wrappers (37 profils → 74 fichiers)
cd multiai-go && make wrappers

# Ou depuis la racine du projet
bash scripts/generate-wrappers.sh

# Usage
./wrappers/multiai-ds          # DeepSeek V4 Pro
./wrappers/multiai-ds.cmd      # Version Windows (.cmd)
./wrappers/multiai-codex55     # Codex GPT-5.5

# Ajouter au PATH
export PATH="$PATH:/chemin/vers/wrappers"
```

Chaque profil `.env` avec `SHORTCUT=` produit deux fichiers :
- `wrappers/multiai-<shortcut>` — bash (Linux, macOS, Git-Bash)
- `wrappers/multiai-<shortcut>.cmd` — cmd (Windows natif)

| Variable | Défaut | Description |
|----------|--------|-------------|
| `MULTIAI_PROFILES_DIR` | `multiai-go/internal/assets/profiles/` | Répertoire des profils `.env` |
| `WRAPPER_OUTPUT_DIR` | `wrappers/` | Répertoire de sortie |
| `MULTIAI_CMD` | `multiai` | Commande à exécuter |

---

## Fonctionnalités

### 🚀 Lancement unifié
- Menu interactif avec navigation complète et retour à chaque niveau
- Lancement direct : `multiai launch -p ds`
- Mode dry-run : `--dry-run` pour simuler sans lancer
- Sortie JSON : `--json` pour l'intégration scriptée

### 🔍 OpenRouter intégré
- **`multiai models`** — top modèles par usage, catégorie, prix
- **`multiai search`** — recherche full-text par mot-clé, fournisseur
- **`multiai compare`** — comparaison côte à côte de 2 modèles
- **Fusion** — panel d'experts multi-modèles avec synthèse automatique
- Cache 1h, fallback offline

### 🔐 Sécurité
- **Isolation par liste blanche** : seul ~30 variables système survivent
- **Credential store** : chiffrement AES-256-GCM dans `~/.config/multiai/secrets`
- **Sentinel pattern** : les fichiers `.env` ne contiennent jamais de clés réelles
- **Whitelist des commandes** : `claude`, `codex`, `opencode` uniquement
- **Anti-fuite npm** : `prepublishOnly` scan les `.env` avant publication
- **Cosign keyless** : signatures des releases vérifiables

### 🌍 Chaînes de fallback
- `FALLBACK=<shortcut>[,…]` : relance automatique sur profil de repli
- Journal de sessions (`sessions.jsonl`) : usage horodaté, **sans secrets**

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
.
├── multiai-go/                  → Go v0.4.0 (implémentation de référence)
│   ├── cmd/multiai/             → Point d'entrée CLI (7 sous-commandes)
│   ├── internal/
│   │   ├── assets/              → 37 profils .env embarqués
│   │   ├── catalog/             → 13 fournisseurs (data-driven YAML)
│   │   ├── cli/                 → Launcher, fallback, hooks, display, completion
│   │   ├── config/              → Wizard interactif + erase keys
│   │   ├── env/                 → Isolation + expansion %VAR%
│   │   ├── logging/             → Journal de sessions JSONL
│   │   │   ├── menu/                → Menus interactifs colorés (top, tool, profile, BMAD)
│   │   ├── onboarding/          → Assistant premier démarrage
│   │   ├── openrouter/          → Client API, cache, search, compare, profilegen
│   │   ├── profile/             → Chargement .env, YAML, projet
│   │   └── secret/              → AES-256-GCM + credential store natif
│   ├── pkg/dotenv/              → Parser .env robuste
│   ├── packaging/               → npm, deb, AUR
│   ├── tests/                   → Tests d'intégration
│   └── scripts/                 → setup, install, sync
│
├── scripts/                     → generate-wrappers.sh (fabrication de wrappers par shortcut)
├── wrappers/                    → Wrappers générés (gitignored)
├── multiai-powershell/          → PowerShell v0.3.0 (gelée)
└── audit/                       → Rapports d'audit BMAD+ (v0.3.0, v0.4.0)
```

---

## Qualité & Audit

**Score global BMAD+ : 8.5/10** (audit 3 agents — 2026-07-05)

| Métrique | Valeur |
|----------|--------|
| Tests Go | 13 packages, tous verts |
| go vet | 0 warning |
| govulncheck | 0 vulnérabilité |
| Dépendances Go | 1 seule (yaml.v3), 0 CVE |
| CI/CD | lint → test (3 OS) → security → build → release (GoReleaser + Cosign) |
| Release signing | SHA256 + Cosign keyless + GitHub provenance |

Rapports d'audit complets dans [`audit/`](audit/).

---

## Communaute

### GitHub Discussions
Posez vos questions, partagez vos profils, proposez des idees dans [GitHub Discussions](https://github.com/lrochetta/multiai/discussions).

| Categorie | Description |
|-----------|-------------|
| [Show and Tell](https://github.com/lrochetta/multiai/discussions/categories/show-and-tell) | Partagez vos profils, vos usages, vos decouvertes |
| [Q&A](https://github.com/lrochetta/multiai/discussions/categories/q-a) | Posez vos questions et obtenez de l'aide |
| [Ideas](https://github.com/lrochetta/multiai/discussions/categories/ideas) | Proposez des fonctionnalites et des ameliorations |

### Contribuer
- **Profils communautaires** : soumettez vos profils YAML via le [registre communautaire](https://github.com/lrochetta/profiles-multiai) ou partagez-les dans [Show and Tell](https://github.com/lrochetta/multiai/discussions/categories/show-and-tell)
- **Code** : consultez [CONTRIBUTING.md](CONTRIBUTING.md) pour les regles de contribution
- **Bug reports** : ouvrez une [issue](https://github.com/lrochetta/multiai/issues/new/choose) avec le template dedie

### Soutenir
[![GitHub Sponsors](https://img.shields.io/badge/sponsor-30363D?logo=github-sponsors)](https://github.com/sponsors/lrochetta)

💡 Vous utilisez multiai au quotidien ? Une etoile sur GitHub, un partage ou un sponsor sont les bienvenus !

---

MIT — [Laurent Rochetta](https://rochetta.fr) • [github.com/lrochetta/multiai](https://github.com/lrochetta/multiai)
