# Profils YAML

En plus du format `.env`, multiai supporte les fichiers YAML pour definir des profils. Le format YAML offre plus de flexibilite : definitions multiples, interpolation de variables, et inclusion de scripts.

## Fichier de profils YAML

Les profils YAML sont definis dans `~/.multiai/profiles.yaml`. Un seul fichier peut contenir plusieurs profils.

```yaml
# ~/.multiai/profiles.yaml
profiles:
  mon-profil-yaml:
    tool: claude
    display_name: Mon Profil YAML
    description: Profil defini en YAML
    provider: Custom
    model: claude-sonnet-4-20250514
    env:
      ANTHROPIC_API_KEY: "${ANTHROPIC_API_KEY}"
      ANTHROPIC_BASE_URL: "https://api.custom-provider.com/v1"
      CUSTOM_VAR: "valeur personnalisee"
```

## Format complet

```yaml
profiles:
  <nom-du-profil>:
    tool: claude           # claude | codex | opencode
    display_name: "..."    # Nom affiche dans le menu
    description: "..."     # Description du profil
    provider: "..."        # Nom du fournisseur
    model: "..."           # Modele par defaut
    env:                   # Variables d'environnement
      VAR1: "valeur1"
      VAR2: "${VAR_EXTERNE}"  # Interpolation depuis l'environnement
    hooks:                 # Scripts before/after (optionnel)
      before_launch: "/chemin/script.sh"
      after_launch: "/chemin/script.sh"
```

## Exemple complet : plusieurs profils

```yaml
profiles:
  # Profil OpenAI GPT-5
  gpt5:
    tool: codex
    display_name: "OpenAI GPT-5"
    description: "Codex CLI avec GPT-5"
    provider: OpenAI
    model: gpt-5
    env:
      OPENAI_API_KEY: "${OPENAI_API_KEY}"

  # Profil DeepSeek via Claude
  deepseek-pro:
    tool: claude
    display_name: "DeepSeek Pro"
    description: "DeepSeek V4 Pro via Claude Code"
    provider: DeepSeek
    model: claude-sonnet-4-20250514
    env:
      ANTHROPIC_API_KEY: "${DEEPSEEK_API_KEY}"
      ANTHROPIC_BASE_URL: "https://api.deepseek.com/v1"

  # Profil local LM Studio
  local:
    tool: codex
    display_name: "LM Studio"
    description: "Modele local Ollama/LM Studio"
    provider: Local
    env:
      OPENAI_API_KEY: "not-needed"
      OPENAI_BASE_URL: "http://localhost:1234/v1"

  # Profil Groq
  groq:
    tool: codex
    display_name: "Groq Cloud"
    description: "Groq via API compatible OpenAI"
    provider: Groq
    env:
      OPENAI_API_KEY: "${GROQ_API_KEY}"
      OPENAI_BASE_URL: "https://api.groq.com/openai/v1"
```

## Interpolation de variables

Le YAML supporte l'interpolation de variables d'environnement avec la syntaxe `${VAR}` :

```yaml
profiles:
  mon-profil:
    tool: claude
    env:
      ANTHROPIC_API_KEY: "${MA_CLE_ANTHROPIC}"
      ANTHROPIC_BASE_URL: "${URL_API}"
      TIMEOUT: "30"
```

multiai remplace `${MA_CLE_ANTHROPIC}` par la valeur de la variable d'environnement `MA_CLE_ANTHROPIC` au moment du lancement.

## Migration .env vers .yaml

Si tu as des fichiers `.env` existants, tu peux les migrer vers le format YAML.

### .env existant

```text
# ~/.multiai/profiles/mon-profil.env
TOOL=claude
MODEL=claude-sonnet-4-20250514
ANTHROPIC_API_KEY=sk-ant-...
ANTHROPIC_BASE_URL=https://api.example.com/v1
DISPLAY_NAME=Mon Profil
DESCRIPTION=Description du profil
PROVIDER=Custom
```

### Equivalent YAML

```yaml
# ~/.multiai/profiles.yaml
profiles:
  mon-profil:
    tool: claude
    display_name: "Mon Profil"
    description: "Description du profil"
    provider: Custom
    model: claude-sonnet-4-20250514
    env:
      ANTHROPIC_API_KEY: "${ANTHROPIC_API_KEY}"
      ANTHROPIC_BASE_URL: "https://api.example.com/v1"
```

### Priorite

Quand un profil existe a la fois en `.env` et en `.yaml`, le fichier `.yaml` est prioritaire.

## Avantages du format YAML

- **Plusieurs profils dans un seul fichier** : tout est centralise
- **Interpolation de variables** : utilise les variables d'environnement directement
- **Hooks integres** : definit les scripts before/after par profil
- **Structure plus lisible** : pas de flat key=value
- **Versionnable** : peut etre commit dans un repo (sans les clefs)

## Voir aussi

- [Profils personnalises](/advanced/custom-profiles) — format .env
- [Profils par projet](/advanced/project-profiles) — configuration .multiai.yaml
- [Plugin Hooks](/advanced/plugin-hooks) — scripts before/after launch
