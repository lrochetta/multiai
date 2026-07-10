# Fournisseurs

multiai intègre **13 fournisseurs** répartis en 3 régions. Chaque fournisseur expose un ou plusieurs profils (shortcuts) que tu peux utiliser avec `multiai launch -p <shortcut>`.

## Régions

| Région | Fournisseurs |
|--------|-------------|
| 🌍 Global / Agrégateurs | OpenRouter, Requesty, LiteLLM |
| 🇨🇳 Chine | DeepSeek, Z.ai, Qwen/DashScope, MiniMax, Kimi/Moonshot, StepFun, SiliconFlow, Xiaomi MiMo |
| 🇺🇸 USA | Anthropic, OpenAI |

---

## 🌍 Global / Agrégateurs

### OpenRouter

Agrégateur de **300+ modèles** avec fusion multi-modèle automatique.

| Champ | Valeur |
|-------|--------|
| **Console** | [Créer une clé](https://openrouter.ai/settings/keys) |
| **Profils** | `or-fusion` (Claude Code), `codex-fusion` (Codex), `oc-fusion` (OpenCode), `ocqwen`, `ockimi`, `ocminimax` |
| **Variable** | `OPENROUTER_API_KEY` |
| **API** | `https://openrouter.ai/api/v1` |

```bash
multiai config --provider openrouter
multiai launch -p or-fusion
```

---

### Requesty

Gateway européen avec fallback, cache, et contrôle des coûts. Gratuit 200 req/j.

| Champ | Valeur |
|-------|--------|
| **Console** | [Créer une clé](https://app.requesty.ai/api-keys) |
| **Profils** | `req-cc` (Claude Code), `req-codex` (Codex), `req-oc` (OpenCode), `ceu` (EU Frankfurt) |
| **Variable** | `REQUESTY_API_KEY` |
| **API** | `https://router.requesty.ai/v1` |

```bash
multiai config --provider requesty
multiai launch -p req-cc
```

---

### LiteLLM

Proxy local open-source à lancer avec Docker ou WSL2. Héberge tous les modèles derrière une API unique.

| Champ | Valeur |
|-------|--------|
| **Console** | [Documentation](https://docs.litellm.ai/docs/proxy/quick_start) |
| **Profils** | `litellm` (via Claude Code) |
| **Variable** | `LITELLM_API_KEY` |
| **API** | `http://localhost:4000` |

```bash
multiai config --provider litellm
multiai launch -p litellm
```

---

## 🇨🇳 Chine

### DeepSeek (V4 Pro 1M / Flash)

Modèles V4 Pro (1M tokens de contexte) et V4 Flash. Compatible Anthropic + OpenAI.

| Champ | Valeur |
|-------|--------|
| **Console** | [Créer une clé](https://platform.deepseek.com/api_keys) |
| **Profils** | `ds` (Claude Code), `dsf` (Claude Code Flash), `ocdeepseek` (OpenCode) |
| **Variable** | `ANTHROPIC_AUTH_TOKEN` ou `DEEPSEEK_API_KEY` |
| **API** | `https://api.deepseek.com/v1` |

```bash
multiai config --provider deepseek
multiai launch -p ds    # V4 Pro via Claude Code
multiai launch -p dsf   # V4 Flash via Claude Code
```

---

### Z.ai / BigModel (GLM-5.2)

Modèles GLM-5.2 de Zhipu AI. Compatible Anthropic + OpenAI.

| Champ | Valeur |
|-------|--------|
| **Console** | [Créer une clé](https://bigmodel.cn/usercenter/apikeys) |
| **Profils** | `cg` (Claude Code endpoint standard), `cgalt` (Claude Code endpoint alternatif), `oczai` (OpenCode) |
| **Variable** | `ANTHROPIC_AUTH_TOKEN` ou `ZAI_API_KEY` |
| **API** | `https://api.z.ai/api/anthropic` ou `https://api.z.ai/api/coding/paas/v4` |

```bash
multiai config --provider zai
multiai launch -p cg
```

---

### Qwen / DashScope (Alibaba)

Modèles Qwen3-Coder d'Alibaba. Compatible OpenAI.

| Champ | Valeur |
|-------|--------|
| **Console** | [Créer une clé](https://dashscope.aliyun.com/api-key-management) |
| **Profils** | `codex-qwen` (Codex), `ocqwen-direct` (OpenCode) |
| **Variable** | `DASHSCOPE_API_KEY` |
| **API** | `https://dashscope.aliyun.com/compatible-mode/v1` |

```bash
multiai config --provider dashscope
multiai launch -p codex-qwen
```

---

### MiniMax (M3 1M)

Modèle M3 avec 1M tokens de contexte et Extended Thinking. Compatible Anthropic + OpenAI.

| Champ | Valeur |
|-------|--------|
| **Console** | [Créer une clé](https://platform.minimax.io/account/api-keys) |
| **Profils** | `mm` (Claude Code), `ocmini` (OpenCode) |
| **Variable** | `MINIMAX_API_KEY` |
| **API** | `https://api.minimax.chat/v1` |

```bash
multiai config --provider minimax
multiai launch -p mm
```

---

### Kimi / Moonshot (K2.7 Code)

Modèle K2.7 spécialisé coding de Moonshot AI. Compatible OpenAI.

| Champ | Valeur |
|-------|--------|
| **Console** | [Créer une clé](https://platform.moonshot.ai/console/api-keys) |
| **Profils** | `ockimi-direct` (OpenCode direct, pas via OpenRouter) |
| **Variable** | `MOONSHOT_API_KEY` |
| **API** | `https://api.moonshot.ai/v1` |

```bash
multiai config --provider moonshot
multiai launch -p ockimi-direct
```

---

### StepFun (Step Plan)

Modèle Step Plan. Compatible Anthropic uniquement, pour Claude Code.

| Champ | Valeur |
|-------|--------|
| **Console** | [Créer une clé](https://platform.stepfun.ai/api-keys) |
| **Profils** | `stepfun` (Claude Code) |
| **Variable** | `STEPFUN_API_KEY` |
| **API** | `https://api.stepfun.com/v1` |

```bash
multiai config --provider stepfun
multiai launch -p stepfun
```

---

### SiliconFlow (Agrégateur open-source)

Agrégateur de modèles open-source chinois (Qwen, DeepSeek, Yi, etc.). Compatible OpenAI.

| Champ | Valeur |
|-------|--------|
| **Console** | [Créer une clé](https://cloud.siliconflow.cn/api-keys) |
| **Profils** | `codex-sf` (Codex), `oc-siliconflow` (OpenCode) |
| **Variable** | `SILICONFLOW_API_KEY` |
| **API** | `https://api.siliconflow.cn/v1` |

```bash
multiai config --provider siliconflow
multiai launch -p codex-sf
```

---

### Xiaomi MiMo (V2.5 Pro)

Modèle V2.5 Pro de Xiaomi. Tier gratuit disponible (mimo-auto). Compatible Anthropic + OpenAI.

| Champ | Valeur |
|-------|--------|
| **Console** | [Créer une clé](https://api.xiaomimimo.com/dashboard/api-keys) |
| **Profils** | `mimo` (Claude Code), `ocmimo` (OpenCode) |
| **Variable** | `MIMO_API_KEY` |
| **API** | `https://api.xiaomimimo.com/v1` |

```bash
multiai config --provider mimo
multiai launch -p mimo
```

---

## 🇺🇸 USA

### Anthropic (officiel)

Fournisseur officiel Anthropic. Accès à Claude Sonnet 4, Haiku 4, etc.

| Champ | Valeur |
|-------|--------|
| **Console** | [Créer une clé](https://console.anthropic.com/settings/keys) |
| **Profils** | `ca` (Claude Code direct), `ocanthropic` (OpenCode) |
| **Variable** | `ANTHROPIC_API_KEY` |
| **API** | `https://api.anthropic.com/v1` |

```bash
multiai config --provider anthropic
multiai launch -p ca
```

> **Note :** Le profil `co` (Claude Code officiel) utilise le login Claude plutôt qu'une clé API — il n'apparaît pas dans ce catalogue.

---

### OpenAI

Fournisseur officiel OpenAI. Accès à GPT-4, GPT-5, GPT-5.5 via Codex CLI et OpenCode.

| Champ | Valeur |
|-------|--------|
| **Console** | [Créer une clé](https://platform.openai.com/api-keys) |
| **Profils** | `ocopenai` (OpenCode) |
| **Variable** | `OPENAI_API_KEY` |
| **API** | `https://api.openai.com/v1` |

```bash
multiai config --provider openai
multiai launch -p ocopenai
```

> **Note :** Les profils Codex CLI (`codex55`, `codex54`, `codexmini`) utilisent leur propre système de login intégré — pas de clé API à configurer ici.

---

## Utilisation en ligne de commande

```bash
# Configurer un fournisseur
multiai config --provider deepseek

# Lister tous les profils disponibles
multiai list

# Voir les détails en JSON
multiai list --json | jq
```

## Voir aussi

- [Guide des profils](/guide/profiles) — liste complète des 37 profils
- [Configuration](/guide/configuration) — configurer les clés API
- [Variables d'environnement](/reference/env-variables) — référence des variables par fournisseur
