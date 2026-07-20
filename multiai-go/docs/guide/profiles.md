# Profils disponibles

multiai inclut **40 profils prêts à l'emploi** couvrant **14 fournisseurs** et 3 CLI (16 Claude Code, 8 Codex CLI, 16 OpenCode). Liste vivante : `multiai list` (ou `multiai list --json`).

## Claude Code (16 profils)

| Shortcut | Display Name | Fournisseur / clé |
|----------|--------------|-------------------|
| `co` | Claude Code officiel | Login Claude (sans clé) |
| `ca` | Anthropic API officielle | `ANTHROPIC_API_KEY` |
| `cp` | Code Pro (premium) | DeepSeek — `ANTHROPIC_AUTH_TOKEN` |
| `cf` | Code Fast (economique) | DeepSeek — `ANTHROPIC_AUTH_TOKEN` |
| `ceu` | Code EU (Requesty RGPD) | `REQUESTY_API_KEY` |
| `cg` | Z.ai GLM-5.2 Coding Plan | `ANTHROPIC_AUTH_TOKEN` |
| `cgalt` | Z.ai GLM-5.2 endpoint alternatif | `ANTHROPIC_API_KEY` |
| `ds` | DeepSeek V4 Pro 1M | `ANTHROPIC_AUTH_TOKEN` |
| `dsf` | DeepSeek V4 Flash | `ANTHROPIC_AUTH_TOKEN` |
| `or-fusion` | OpenRouter Fusion (Multi-Model) | `OPENROUTER_API_KEY` |
| `mm` | MiniMax M3 (1M ctx) | `MINIMAX_API_KEY` |
| `stepfun` | StepFun Step Plan | `STEPFUN_API_KEY` |
| `mimo` | Xiaomi MiMo V2.5 Pro | `MIMO_API_KEY` |
| `req-cc` | Claude Code via Requesty | `REQUESTY_API_KEY` |
| `litellm` | Claude Code via LiteLLM proxy | `LITELLM_API_KEY` |
| `nv-cc` | NVIDIA GLM-5.2 gratuit (pont LiteLLM) | `NVIDIA_API_KEY` |

## Codex CLI (8 profils)

| Shortcut | Display Name | Fournisseur / clé |
|----------|--------------|-------------------|
| `codex55` | Codex GPT-5.5 | Login Codex (sans clé) |
| `codex54` | Codex GPT-5.4 | Login Codex (sans clé) |
| `codexmini` | Codex GPT-5.4 mini | Login Codex (sans clé) |
| `codex-fusion` | Codex via OpenRouter Fusion | `OPENROUTER_API_KEY` |
| `codex-qwen` | Codex Qwen via DashScope | `DASHSCOPE_API_KEY` |
| `codex-sf` | Codex via SiliconFlow | `SILICONFLOW_API_KEY` |
| `req-codex` | Codex via Requesty | `REQUESTY_API_KEY` |
| `codex-nv` | Codex NVIDIA GLM-5.2 gratuit (pont LiteLLM) | `NVIDIA_API_KEY` |

## OpenCode (16 profils)

| Shortcut | Display Name | Fournisseur / clé |
|----------|--------------|-------------------|
| `ocdefault` | OpenCode default / connect | `/connect` (sans clé) |
| `ocopenai` | OpenCode OpenAI GPT-5.5 | `OPENAI_API_KEY` |
| `ocanthropic` | OpenCode Anthropic Claude | `ANTHROPIC_API_KEY` |
| `ocdeepseek` | OpenCode DeepSeek V4 Pro | `DEEPSEEK_API_KEY` |
| `oczai` | OpenCode Z.ai GLM-5.2 | `ZAI_API_KEY` |
| `oc-fusion` | OpenCode via OpenRouter Fusion | `OPENROUTER_API_KEY` |
| `ocqwen` | OpenCode Qwen via OpenRouter | `OPENROUTER_API_KEY` |
| `ockimi` | OpenCode Kimi via OpenRouter | `OPENROUTER_API_KEY` |
| `ocminimax` | OpenCode MiniMax via OpenRouter | `OPENROUTER_API_KEY` |
| `ocmini` | OpenCode MiniMax M3 direct | `MINIMAX_API_KEY` |
| `ocqwen-direct` | OpenCode Qwen direct (DashScope) | `DASHSCOPE_API_KEY` |
| `ockimi-direct` | OpenCode Kimi direct (Moonshot) | `MOONSHOT_API_KEY` |
| `oc-siliconflow` | OpenCode via SiliconFlow | `SILICONFLOW_API_KEY` |
| `ocmimo` | OpenCode Xiaomi MiMo | `MIMO_API_KEY` |
| `req-oc` | OpenCode via Requesty | `REQUESTY_API_KEY` |
| `ocnvidia` | OpenCode NVIDIA NIM gratuit (GLM-5.2 + 9 modeles) | `NVIDIA_API_KEY` |

## Profils dynamiques

En plus des 40 profils embarqués :

- **Menu 4 (OpenRouter)** → option 4 : génère un profil `or-*` (`99-*.env`) pour n'importe quel modèle OpenRouter.
- **Menu 5 (NVIDIA)** → option 3 : génère un profil `nv-*` (`98-*.env`) pour n'importe quel modèle gratuit de build.nvidia.com — voir le [guide NVIDIA](/guide/nvidia).

## Utilisation

```bash
# Lancer un profil spécifique
multiai launch -p ds
multiai launch -p nv-cc
multiai launch -p ocnvidia

# Lister tous les profils disponibles
multiai list

# Lister au format JSON
multiai list --json
```

## Voir aussi

- [Configuration](/guide/configuration) — configurer les clés API
- [NVIDIA gratuit](/guide/nvidia) — GLM 5.2 & 118 modèles gratuits
- [Fournisseurs](/reference/providers) — détail des 14 fournisseurs
- [Profils personnalisés](/advanced/custom-profiles) — créer ses propres profils
- [Profils YAML](/advanced/yaml-profiles) — alternative au format .env
