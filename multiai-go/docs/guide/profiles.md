# Profils disponibles

multiai inclut **17 profils prêts à l'emploi**, couvrant les principaux fournisseurs d'IA et CLI.

## Tableau des profils

| Shortcut | Tool | Display Name | Provider | Description |
|----------|------|-------------|----------|-------------|
| `co` | claude | Anthropic Claude | Anthropic | Claude Code officiel — modèle par défaut |
| `ds` | claude | DeepSeek | DeepSeek (via Anthropic) | DeepSeek V4 Pro chez Anthropic |
| `za` | claude | Z.ai | Z.ai | Z.ai Claude chez Anthropic |
| `son40` | claude | Claude Sonnet 4.0 | Anthropic | Claude Sonnet 4.0 |
| `son41` | claude | Claude Sonnet 4.1 | Anthropic | Claude Sonnet 4.1 |
| `ha40` | claude | Claude Haiku 4.0 | Anthropic | Claude Haiku 4.0 |
| `oa` | codex | OpenAI GPT | OpenAI | OpenAI officiel via Codex CLI |
| `oa4` | codex | OpenAI GPT-4 | OpenAI | GPT-4 via Codex CLI |
| `oa5` | codex | OpenAI GPT-5 | OpenAI | GPT-5 via Codex CLI |
| `codex` | codex | Codex default | OpenAI | Codex CLI — modèle par défaut |
| `codex4` | codex | Codex GPT-4 | OpenAI | Codex CLI avec GPT-4 |
| `codex45` | codex | Codex GPT-4.5 | OpenAI | Codex CLI avec GPT-4.5 |
| `codex55` | codex | Codex GPT-5.5 | OpenAI | Codex CLI avec GPT-5.5 |
| `or` | codex | OpenRouter | OpenRouter | OpenRouter via Codex CLI |
| `oc` | opencode | OpenCode default | OpenCode | OpenCode — modèle par défaut |
| `oc4` | opencode | OpenCode GPT-4 | OpenAI | OpenCode avec GPT-4 |
| `oc5` | opencode | OpenCode GPT-5 | OpenAI | OpenCode avec GPT-5 |

## Détails par fournisseur

### Anthropic (Claude Code)

Les profils `co`, `ds`, `za`, `son40`, `son41` et `ha40` utilisent tous **Claude Code** comme CLI, mais avec des modèles et fournisseurs différents.

| Profil | Variable clé | Modèle |
|--------|-------------|--------|
| `co` | `ANTHROPIC_API_KEY` | claude-sonnet-4-20250514 |
| `ds` | `ANTHROPIC_API_KEY` | claude-sonnet-4-20250514 (via DeepSeek) |
| `za` | `ANTHROPIC_API_KEY` | claude-sonnet-4-20250514 (via Z.ai) |
| `son40` | `ANTHROPIC_API_KEY` | claude-sonnet-4-20250514 |
| `son41` | `ANTHROPIC_API_KEY` | claude-sonnet-4-20250513 |
| `ha40` | `ANTHROPIC_API_KEY` | claude-haiku-4-20250514 |

### OpenAI (Codex CLI)

Les profils `oa`, `oa4`, `oa5`, `codex`, `codex4`, `codex45`, `codex55` et `or` utilisent **Codex CLI**.

| Profil | Variable clé | Modèle |
|--------|-------------|--------|
| `oa` | `OPENAI_API_KEY` | gpt-5 |
| `oa4` | `OPENAI_API_KEY` | gpt-4 |
| `oa5` | `OPENAI_API_KEY` | gpt-5 |
| `codex` | `OPENAI_API_KEY` | (défaut Codex) |
| `codex4` | `OPENAI_API_KEY` | gpt-4 |
| `codex45` | `OPENAI_API_KEY` | gpt-4.5 |
| `codex55` | `OPENAI_API_KEY` | gpt-5.5 |
| `or` | `OPENROUTER_API_KEY` | (modèle OpenRouter) |

### OpenCode

Les profils `oc`, `oc4` et `oc5` utilisent **OpenCode**.

| Profil | Variable clé | Modèle |
|--------|-------------|--------|
| `oc` | `OPENAI_API_KEY` | (défaut OpenCode) |
| `oc4` | `OPENAI_API_KEY` | gpt-4 |
| `oc5` | `OPENAI_API_KEY` | gpt-5 |

## Utilisation

```bash
# Lancer un profil spécifique
multiai launch -p ds
multiai launch -p codex55
multiai launch -p oc

# Lister tous les profils disponibles
multiai list

# Lister au format JSON
multiai list --json
```

## Voir aussi

- [Configuration](/guide/configuration) — configurer les clés API
- [Profils personnalisés](/advanced/custom-profiles) — créer ses propres profils
- [Profils YAML](/advanced/yaml-profiles) — alternative au format .env
