# Variables d'environnement

Liste complete des variables d'environnement utilisees par les profils multiai, classees par fournisseur.

## Anthropic (Claude Code)

Ces variables sont utilisees par les profils qui utilisent Claude Code comme CLI (`co`, `ca`, `ds`, `dsf`, `cg`, `nv-cc`, `litellm`, ...).

| Variable | Obligatoire | Description |
|----------|-------------|-------------|
| `ANTHROPIC_API_KEY` | Oui | Cle API Anthropic pour les appels Claude |
| `ANTHROPIC_BASE_URL` | Non | URL de base de l'API (pour proxies ou fournisseurs tiers) |
| `ANTHROPIC_MODEL` | Non | Modele a utiliser (defaut : `claude-sonnet-4-20250514`) |
| `CLAUDE_CODE_VERSION` | Non | Version de Claude Code a utiliser |
| `CLAUDE_CODE_DIR` | Non | Repertoire de travail pour Claude Code |
| `MCP_PORT` | Non | Port pour le serveur MCP (defaut : dynamique) |

### Fournisseurs compatibles

Ces profils utilisent `ANTHROPIC_BASE_URL` pour router vers d'autres fournisseurs :

| Profil | `ANTHROPIC_BASE_URL` |
|--------|----------------------|
| `ds` / `dsf` (DeepSeek) | `https://api.deepseek.com/anthropic` |
| `cg` (Z.ai) | `https://api.z.ai/api/anthropic` |
| `or-fusion` (OpenRouter) | `https://openrouter.ai/api` |
| `nv-cc` (NVIDIA, pont integre) | injecte automatiquement (`http://127.0.0.1:<port ephemere>`) |
| `litellm` | `http://localhost:4000` |

---

## OpenAI (Codex CLI)

Ces variables sont utilisees par les profils qui utilisent Codex CLI (`codex55`, `codex54`, `codexmini`, `codex-qwen`, `codex-sf`, `codex-nv`, ...).

| Variable | Obligatoire | Description |
|----------|-------------|-------------|
| `OPENAI_API_KEY` | Oui | Cle API OpenAI |
| `OPENAI_BASE_URL` | Non | URL de base de l'API OpenAI |
| `OPENAI_MODEL` | Non | Modele a utiliser |
| `CODEX_VERSION` | Non | Version de Codex CLI |
| `CODEX_DIR` | Non | Repertoire de travail pour Codex |

---

## OpenRouter

Ces variables sont utilisees par les profils OpenRouter (`or-fusion`, `codex-fusion`, `oc-fusion`, `ocqwen`, `ockimi`, `ocminimax`).

| Variable | Obligatoire | Description |
|----------|-------------|-------------|
| `OPENROUTER_API_KEY` | Oui | Cle API OpenRouter |
| `OPENROUTER_BASE_URL` | Non | URL de base (defaut : `https://openrouter.ai/api/v1`) |
| `OPENROUTER_MODEL` | Non | Modele a utiliser via OpenRouter |

---

## NVIDIA build.nvidia.com (NIM)

Variable utilisee par les profils NVIDIA (`nv-cc`, `codex-nv`, `ocnvidia` et les profils dynamiques `nv-*`). Cle gratuite (format `nvapi-...`) : <https://build.nvidia.com/settings/api-keys>.

| Variable | Obligatoire | Description |
|----------|-------------|-------------|
| `NVIDIA_API_KEY` | Oui | Cle API build.nvidia.com — catalogue 100% gratuit (~40 req/min) |
| `NVIDIA_NIM_API_KEY` | Non | Lue par le pont LiteLLM (`multiai-go/scripts/nvidia-bridge.ps1`) |

Le menu 5 (decouverte NVIDIA) lit aussi `NVIDIA_API_KEY` pour interroger `/v1/models` (fonctionne sans cle, envoyee quand disponible).

---

## OpenCode

Ces variables sont utilisees par les profils qui utilisent OpenCode (`ocdefault`, `ocopenai`, `ocdeepseek`, `oczai`, `ocnvidia`, ...). Les profils a fournisseur custom passent par `OPENCODE_CONFIG_CONTENT` (JSON inline).

| Variable | Obligatoire | Description |
|----------|-------------|-------------|
| `OPENAI_API_KEY` | Oui | Cle API OpenAI (ou fournisseur compatible) |
| `OPENAI_BASE_URL` | Non | URL de base de l'API |
| `OPENAI_MODEL` | Non | Modele a utiliser |
| `OPENCODE_DIR` | Non | Repertoire de travail pour OpenCode |

---

## Configuration multiai

Variables reellement lues par le binaire (le reste du comportement passe par les
flags de commande, pas par des variables d'environnement).

| Variable | Description | Valeur par defaut |
|----------|-------------|-------------------|
| `MULTIAI_PROFILES_DIR` | Repertoire des profils `.env` (prioritaire) | `<config>/multiai/profiles` |
| `MULTIAI_SECRETS_DIR` | Repertoire du credential store chiffre | `~/.config/multiai/secrets` |
| `MULTIAI_CACHE_DIR` | Cache OpenRouter (`models`/`search`) | `<config>/multiai/cache` |
| `MULTIAI_LOGS_DIR` | Journal de sessions (`sessions.jsonl`) | `<config>/multiai/logs` |
| `MULTIAI_DEV` | Si defini, autorise le chargement des profils depuis `./configs/profiles` (dev uniquement — desactive par defaut pour raison de securite) | *(non defini)* |
| `NO_COLOR` | Si defini (n'importe quelle valeur), desactive les couleurs ANSI | *(non defini)* |

`<config>` = `os.UserConfigDir()` (Windows : `%AppData%` ; macOS : `~/Library/Application Support` ; Linux : `~/.config`).

Cote installeur npm (`packaging/npm/install.js`) : `MULTIAI_SKIP_DOWNLOAD=1` saute le
telechargement du binaire, `MULTIAI_INSTALL_DIR` copie aussi le binaire verifie
dans ce repertoire.

Ordre de resolution des profils : `MULTIAI_PROFILES_DIR` > `<dir de l'exe>/configs/profiles` > `./configs/profiles` (si `MULTIAI_DEV`) > `<config>/multiai/profiles` (materialise au premier lancement).

---

## Variables de surcharge de profil

Tu peux surcharger n'importe quelle variable d'un profil en la definissant directement dans l'environnement avant le lancement :

```bash
# Surcharger le modele pour un lancement
ANTHROPIC_MODEL=claude-opus-4-20250514 multiai launch -p co

# Changer l'URL de base temporairement
ANTHROPIC_BASE_URL=https://custom-proxy.example.com multiai launch -p co
```

## Profils YAML personnalises

Quand tu definis des profils dans `~/.multiai/profiles.yaml`, tu peux utiliser n'importe quelle variable d'environnement comme reference :

```yaml
profiles:
  mon-profil:
    tool: claude
    env:
      ANTHROPIC_API_KEY: "${ANTHROPIC_API_KEY}"
      CUSTOM_VAR: "valeur personnalisee"
```

## Voir aussi

- [Profils disponibles](/guide/profiles) — les 40 profils inclus
- [Profils personnalises](/advanced/custom-profiles) — creer ses propres profils
- [Profils YAML](/advanced/yaml-profiles) — format YAML avance
