# Profils par projet

multiai supporte la configuration par projet via un fichier `.multiai.yaml` a la racine de ton projet. Cela permet de partager une configuration commune entre les membres de l'equipe tout en laissant chacun utiliser ses propres cles API.

## Principe

1. Tu places un fichier `.multiai.yaml` a la racine de ton projet
2. Ce fichier est versionne dans Git
3. Chaque developpeur garde ses cles API dans `~/.multiai/profiles/`
4. multiai fusionne les deux sources : projet + utilisateur

## Structure du fichier

```yaml
# .multiai.yaml (a la racine du projet)
project:
  name: "mon-projet"
  description: "Configuration multiai pour mon projet"

profiles:
  # Reference a un profil utilisateur
  default:
    extends: co
    env:
      CLAUDE_CODE_DIR: "${PROJECT_DIR}"
      CUSTOM_PROJECT_VAR: "mon-projet"

  # Profil specifique au projet
  staging:
    tool: claude
    display_name: "Staging Env"
    description: "Lancement avec variables de staging"
    env:
      ANTHROPIC_API_KEY: "${ANTHROPIC_API_KEY}"
      NODE_ENV: "staging"
      DEBUG: "true"
```

## Heritage et surcharge

La configuration par projet supporte l'heritage avec `extends`. Le profil projet herite de toutes les variables du profil parent et peut les surcharger.

### Ordre de priorite

1. Variables d'environnement du shell (priorite la plus haute)
2. Profil `.multiai.yaml` du projet
3. Profil `~/.multiai/profiles.yaml`
4. Profil `~/.multiai/profiles/<name>.env`

### Exemple d'heritage

```yaml
# .multiai.yaml
profiles:
  # Herite du profil "co" defini dans ~/.multiai/profiles/co.env
  default:
    extends: co
    env:
      # Surcharge la variable MODEL pour ce projet
      MODEL: "claude-sonnet-4-20250514"
      # Ajoute des variables specifiques au projet
      PROJECT_ROOT: "${PWD}"

  # Herite du profil "ds" mais avec un modele different
  deepseek-projet:
    extends: ds
    env:
      MODEL: "claude-opus-4-20250514"
```

## Exemple complet

```yaml
# .multiai.yaml
project:
  name: "api-gateway"
  description: "API Gateway - Equipe backend"
  default_profile: staging

profiles:
  default:
    extends: co
    env:
      CLAUDE_CODE_DIR: "${PWD}/src"
      MCP_PORT: "8080"

  staging:
    extends: ds
    display_name: "Staging Claude"
    env:
      ANTHROPIC_BASE_URL: "https://staging-api.company.com/v1"
      DEBUG: "true"
      LOG_LEVEL: "debug"

  production:
    extends: co
    display_name: "Production Claude"
    env:
      ANTHROPIC_BASE_URL: "https://api.company.com/v1"
      LOG_LEVEL: "info"
```

## Utilisation

```bash
# Dans le repertoire du projet
cd ~/projects/api-gateway

# Lance le profil par defaut du projet
multiai launch

# Lance un profil specifique du projet
multiai launch -p staging
multiai launch -p production
```

## Bonnes pratiques

1. **Ne pas versionner les cles API** : le `.multiai.yaml` ne contient que des references `${VAR}` ou des variables non-sensibles
2. **Documenter les prerequis** : ajoute un commentaire en haut du fichier pour indiquer les variables d'environnement requises
3. **Fichier .gitignore** : ajoute `!*.env` si tu veux versionner des exemples, mais jamais les cles reelles
4. **Profil par defaut** : definis `project.default_profile` pour que `multiai launch` sans argument utilise le bon profil

### Exemple .gitignore

```gitignore
# Ne pas versionner les fichiers de configuration personnels
.env
*.env
```

### Exemple .env.example

```env
# .env.example — copie vers .env et remplis tes cles
ANTHROPIC_API_KEY=sk-ant-...
OPENAI_API_KEY=sk-proj-...
```

## Detection automatique

multiai detecte automatiquement la presence d'un fichier `.multiai.yaml` dans le repertoire courant (et les parents). Tu n'as rien a configurer pour l'activer.

```bash
# Dans le repertoire du projet, multiai utilise automatiquement .multiai.yaml
cd ~/projects/mon-projet
multiai launch -p default

# En dehors, multiai utilise la configuration globale
cd /tmp
multiai launch -p co
```

## Voir aussi

- [Profils YAML](/advanced/yaml-profiles) — format YAML detaille
- [Profils personnalises](/advanced/custom-profiles) — creer ses propres profils
- [Plugin Hooks](/advanced/plugin-hooks) — scripts before/after dans le projet
