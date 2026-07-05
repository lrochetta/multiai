# Claude Code — Afficher la fenêtre de contexte et les tokens en temps réel

## Objectif

Afficher en permanence dans Claude Code :
- Le pourcentage de contexte utilisé
- Le nombre de tokens consommés / disponibles
- Le modèle actif

## Méthode 1 — Slash command (recommandée)

Dans une session Claude Code :

```
/statusline show model, context usage, and token count
```

Claude Code génère le script, met à jour `~/.claude/settings.json` automatiquement et active la status line immédiatement.

## Méthode 2 — Configuration manuelle

Editer `~/.claude/settings.json` (`C:\Users\<user>\.claude\settings.json` sur Windows) :

```json
{
  "statusLine": {
    "type": "command",
    "command": "jq -r '\"Context: \\(.context_window.used_percentage)% | Tokens: \\(.context_window.used_tokens)/\\(.context_window.max_tokens) | Restant: \\(.context_window.remaining_tokens)\"'"
  }
}
```

## Champs disponibles

| Champ | Description |
|---|---|
| `context_window.used_percentage` | % du contexte utilisé |
| `context_window.used_tokens` | Tokens actuellement consommés |
| `context_window.max_tokens` | Taille totale de la fenêtre |
| `context_window.remaining_tokens` | Tokens encore disponibles |
| `context_window.model` | Modèle actif |

## Exemples de formats

**Minimaliste**
```json
"command": "jq -r '\"\\(.context_window.used_percentage)% — \\(.context_window.remaining_tokens) restants\"'"
```

**Complet**
```json
"command": "jq -r '\"[\\(.context_window.model)] \\(.context_window.used_tokens)/\\(.context_window.max_tokens) tokens (\\(.context_window.used_percentage)%)\"'"
```

## Comportement

- La status line s'affiche en bas de la fenêtre Claude Code
- Elle se rafraîchit automatiquement à chaque échange
- Persiste pour toute la durée de la session

## Prérequis

- `jq` installé sur la machine
  - Windows : `winget install jqlang.jq` ou `choco install jq`
  - macOS : `brew install jq`
  - Linux : `sudo apt-get install jq`
