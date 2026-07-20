# NVIDIA build.nvidia.com — modèles gratuits

NVIDIA héberge ~118 modèles sur [build.nvidia.com](https://build.nvidia.com/models) (catalogue NIM), dont **GLM 5.2, DeepSeek V4 Pro/Flash, Kimi K2.6, MiniMax M3, Qwen 3.5, GPT-OSS 120B, Mistral Large 3 et la famille Nemotron**. Tout le catalogue hébergé est **100 % gratuit**.

## Gratuit vs payant — comment ça marche

| | |
|---|---|
| **Gratuit** | Tout le catalogue hébergé `integrate.api.nvidia.com` : pas de facturation par token, pas de carte bancaire. Limite ~**40 req/min** par compte (jusqu'à ~200 sur demande auprès de NVIDIA). HTTP 429 en cas de dépassement. |
| **« Payant » NVIDIA** | NVIDIA ne vend **pas** d'API par token. La production passe par **NIM self-host** (licence NVIDIA AI Enterprise, par GPU) ou **DGX Cloud Serverless** (facturation GPU, contact commercial). |
| **Modèles payants par token** | Passer par **OpenRouter** (menu 4 de multiai) : mêmes modèles ouverts + modèles propriétaires, avec prix affichés par million de tokens. |

⚠️ Aux heures de pointe, les modèles populaires (GLM 5.2, DeepSeek V4 Pro) peuvent être **lents ou mis en file d'attente** — c'est la contrepartie du gratuit.

## 1. Générer la clé (gratuite)

1. Va sur **<https://build.nvidia.com/settings/api-keys>** (compte NVIDIA requis, vérification téléphone possible, pas de CB).
2. Génère une clé — format `nvapi-...`, affichée **une seule fois**.
3. Configure-la dans multiai (stockée chiffrée dans le credential store) :

```bash
multiai config --provider nvidia
```

## 2. Découvrir les modèles

Menu interactif → option **5. NVIDIA — Modèles gratuits (build.nvidia.com)** :

- **Lister** les ~118 modèles (tri par nom ou éditeur), tous gratuits ;
- **Rechercher** (`glm`, `deepseek`, `coder`…) ;
- **Créer un profil dynamique** pour n'importe quel modèle du catalogue.

L'endpoint `/v1/models` de NVIDIA n'expose ni prix ni contexte (tout est gratuit) ; cache local 1 h, fallback hors-ligne sur une liste embarquée.

## 3. Utiliser les modèles par CLI

### OpenCode — direct, zéro friction ✅

```bash
multiai launch -p ocnvidia
```

Le profil `ocnvidia` connecte OpenCode **directement** à `https://integrate.api.nvidia.com/v1` avec GLM 5.2 par défaut + 9 modèles pré-déclarés (`/models` pour changer).

### Claude Code — pont intégré, zéro installation ✅

NVIDIA n'expose pas l'API Anthropic (`/v1/messages` → 404), mais multiai embarque **son propre pont de traduction Anthropic→OpenAI dans le binaire Go** : le launcher le démarre automatiquement sur `127.0.0.1` (port éphémère), injecte `ANTHROPIC_BASE_URL`, et l'arrête quand Claude Code se ferme. **Rien à installer, rien à démarrer :**

```powershell
multiai launch -p nv-cc             # GLM 5.2 (opus/sonnet) + DeepSeek V4 Flash (haiku)
```

Le pont intégré traduit : streaming SSE, tool calls, reasoning (`reasoning_content` → blocs thinking), `count_tokens`, erreurs au format Anthropic, `/v1/models`. Il plafonne `max_tokens` à 32 768 (limite NVIDIA). Déclaration côté profil : `BRIDGE=anthropic-openai` + `BRIDGE_TARGET` + `BRIDGE_KEY_VAR` — n'importe quel backend OpenAI-compatible peut en profiter.

Usage standalone (pour un Claude Code lancé hors multiai) :

```powershell
$env:NVIDIA_API_KEY = "nvapi-..."
multiai bridge                      # ecoute sur 127.0.0.1:4100
# puis : ANTHROPIC_BASE_URL=http://127.0.0.1:4100
```

> Pourquoi pas claude-code-router ? Son transformer NIM est cassé depuis avril 2026 (issue #1341, crashs exit 143). Et LiteLLM (alternative valable) impose Python — le pont intégré supprime cette dépendance pour Claude Code.

### Codex CLI — via le pont LiteLLM 🌉

Codex 2026 a supprimé `wire_api="chat"` et exige l'API Responses, que NVIDIA ne sert pas (`/v1/responses` → 404). Le pont intégré de multiai ne parle que l'API Anthropic pour l'instant — Codex passe donc par **LiteLLM** (Python 3.10–3.13 ; 3.14 sans wheel précompilée) :

```powershell
pip install "litellm[proxy]"            # une fois — eviter 1.82.7/1.82.8 (compromises PyPI)
multiai-go\scripts\nvidia-bridge.ps1    # pont LiteLLM sur 127.0.0.1:4000 (config wildcard nvidia_nim/*)
multiai launch -p codex-nv              # provider injecte par flags -c, config.toml intact
```

### Gemini CLI — non supporté ❌

Gemini CLI n'accepte que les backends Google (pas d'endpoint OpenAI-compatible, feature requests ouvertes non implémentées). Utilise OpenCode pour la même expérience avec les modèles NVIDIA.

## Récapitulatif des profils

| Profil | CLI | Chemin | Modèle par défaut |
|--------|-----|--------|-------------------|
| `nv-cc` | Claude Code | **pont intégré** (automatique) | `z-ai/glm-5.2` (haiku → `deepseek-ai/deepseek-v4-flash`) |
| `codex-nv` | Codex CLI | pont LiteLLM :4000 | `z-ai/glm-5.2` |
| `ocnvidia` | OpenCode | direct | `z-ai/glm-5.2` + 9 modèles |
| `nv-*` | au choix | selon CLI (claude → pont intégré) | générés via le menu 5 → option 3 |

## Limites connues

- **40 req/min partagées** : les sous-agents parallèles de Claude Code consomment vite le quota — le profil `nv-cc` active `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1` pour l'économiser.
- **GLM 5.2 hébergé** : contexte 1M tokens en entrée, mais sortie plafonnée à 32 768 tokens par appel.
- **Modèles « gated »** : certains modèles listés peuvent renvoyer 404 selon le compte (ex. rapporté : `moonshotai/kimi-k2.6`).
- **Tool calls** : la fiabilité varie selon le modèle open-source ; GLM 5.2 et DeepSeek V4 sont les plus solides en agentique.
