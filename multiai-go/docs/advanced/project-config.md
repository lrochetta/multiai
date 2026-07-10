# Configuration projet (.multiai.yaml)

multiai supporte la configuration par projet via un fichier `.multiai.yaml` à la racine de ton projet. Cela permet de partager une configuration commune entre les membres de l'équipe tout en laissant chacun utiliser ses propres clés API.

## Principe

1. Tu places un fichier `.multiai.yaml` à la racine de ton projet
2. Ce fichier est versionné dans Git
3. Chaque développeur garde ses clés API dans `~/.multiai/profiles/`
4. multiai fusionne les deux sources : projet + utilisateur

## Exemple minimal

```yaml
# .multiai.yaml à la racine du projet
profiles:
  default:
    extends: co
    env:
      PROJECT_DIR: "${PWD}"
```

```bash
# Dans le projet, `multiai launch` utilise automatiquement la config projet
cd ~/projects/mon-projet
multiai launch
```

## Structure complète

```yaml
project:
  name: "mon-projet"              # Nom du projet
  description: "API Gateway"      # Description optionnelle
  default_profile: staging        # Profil par défaut (optionnel)

profiles:
  # Référence à un profil utilisateur avec surcharge
  default:
    extends: co
    env:
      CLAUDE_CODE_DIR: "${PWD}/src"
      MCP_PORT: "8080"

  # Profil défini entièrement dans le projet
  staging:
    tool: claude
    display_name: "Staging"
    description: "Lancement avec variables de staging"
    env:
      ANTHROPIC_API_KEY: "${ANTHROPIC_API_KEY}"
      ANTHROPIC_BASE_URL: "https://staging-api.company.com/v1"
      NODE_ENV: "staging"
      DEBUG: "true"
```

## Héritage avec `extends`

Le mot-clé `extends` permet de partir d'un profil existant et de le personnaliser :

```yaml
profiles:
  default:
    extends: co                    # Hérite de ~/.multiai/profiles/co.env
    env:
      MODEL: "claude-sonnet-4-20250514"
      PROJECT_ROOT: "${PWD}"

  deepseek-projet:
    extends: ds                    # Hérite du profil DeepSeek
    env:
      MODEL: "claude-opus-4-20250514"
```

Les variables définies dans le projet surchargent celles du profil parent.

## Ordre de priorité

1. **Variables d'environnement du shell** (priorité la plus haute)
2. **Profil `.multiai.yaml`** du projet
3. **Profil `~/.multiai/profiles.yaml`**
4. **Profil `~/.multiai/profiles/<name>.env`**

## Utilisation

```bash
# Dans le répertoire du projet
cd ~/projects/mon-projet

# Lance le profil par défaut du projet
multiai launch

# Lance un profil spécifique du projet
multiai launch -p staging
multiai launch -p production
```

## Détection automatique

multiai détecte automatiquement la présence d'un fichier `.multiai.yaml` dans le répertoire courant (et les répertoires parents). Tu n'as rien à configurer pour l'activer.

```bash
# Dans le projet, multiai utilise .multiai.yaml automatiquement
cd ~/projects/mon-projet
multiai launch -p default

# En dehors du projet, multiai utilise la configuration globale
cd /tmp
multiai launch -p co
```

## Exemple concret : projet avec 3 environnements

```yaml
# .multiai.yaml
project:
  name: "api-gateway"
  description: "API Gateway - Équipe backend"
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

## Hooks par projet

Tu peux aussi définir des hooks before/after launch dans le `.multiai.yaml` :

```yaml
profiles:
  default:
    extends: co
    hooks:
      before_launch: "./scripts/check-env.sh"
      after_launch: "./scripts/cleanup.sh"
```

## Bonnes pratiques

1. **Ne pas versionner les clés API** — le fichier ne contient que des références `${VAR}`
2. **Documenter les prérequis** — ajoute un commentaire en haut du fichier
3. **Ignorer les `.env`** dans `.gitignore` :

```text
# .gitignore
.env
*.env
```

4. **Ajouter un `.env.example`** dans le dépôt :

```text
# .env.example
ANTHROPIC_API_KEY=sk-ant-...
OPENAI_API_KEY=sk-proj-...
```

5. **Définir un profil par défaut** avec `project.default_profile` pour que `multiai launch` sans argument utilise le bon profil

## Voir aussi

- [Profils YAML](/advanced/yaml-profiles) — format YAML détaillé
- [Hooks before/after launch](/advanced/hooks) — scripts d'automatisation
- [Profils personnalisés](/advanced/custom-profiles) — créer ses propres profils .env
