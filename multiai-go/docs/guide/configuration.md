# Configuration

multiai utilise des fichiers `.env` pour stocker les clés API et les paramètres de chaque profil.

## Menu interactif

La manière la plus simple de configurer multiai est d'utiliser le menu interactif :

```bash
multiai config
```

Ce menu te guide à travers chaque fournisseur :
1. Sélectionne un fournisseur dans la liste
2. multiai ouvre le lien pour créer une clé API
3. Colle la clé API dans le terminal
4. La clé est sauvegardée dans le fichier `.env` correspondant

Tu peux configurer un seul fournisseur ou tous les fournisseurs à la suite.

## Structure des fichiers de configuration

```
~/.multiai/
├── config.yaml            # Configuration globale
├── profiles/
│   ├── co.env             # Claude Code (Anthropic)
│   ├── ds.env             # DeepSeek
│   ├── za.env             # Z.ai
│   ├── oa.env             # OpenAI
│   ├── or.env             # OpenRouter
│   └── ...                # Autres profils
└── profiles.yaml          # Profils personnalisés (optionnel)
```

Chaque fichier `.env` contient les variables d'environnement pour un profil :

```env
# ~/.multiai/profiles/ds.env
ANTHROPIC_API_KEY=sk-ant-...
MODEL=claude-sonnet-4-20250514
PROVIDER=anthropic
```

## Édition manuelle

Tu peux modifier les fichiers `.env` directement avec ton éditeur préféré :

```bash
# Éditer le profil Claude Code
code ~/.multiai/profiles/co.env

# Éditer le profil DeepSeek
nano ~/.multiai/profiles/ds.env
```

## Stockage sécurisé (credential store)

Pour ne pas exposer tes clés API en clair sur le disque, multiai supporte le stockage sécurisé via le credential store de ton système d'exploitation.

### macOS (Keychain)

Les clés peuvent être stockées dans le Keychain macOS :

```bash
# Stocker une clé
multiai config --store keychain

# ou directement
security add-generic-password -s "multiai-<profil>" -a "<clé>" -w
```

### Windows (Credential Manager)

```bash
multiai config --store wincred
```

### Linux (Secret Service / libsecret)

```bash
multiai config --store secret-service
```

### Détection automatique

Si tu omets `--store`, multiai détecte automatiquement le credential store disponible sur ta plateforme. Tu peux toujours forcer le mode fichier `.env` avec :

```bash
multiai config --store file
```

## Configuration globale

Le fichier `~/.multiai/config.yaml` contient les paramètres généraux :

```yaml
# ~/.multiai/config.yaml
default_profile: co
credential_store: auto  # auto, file, keychain, wincred, secret-service
theme: default           # default, dark, light
language: fr
```

| Option | Description | Valeurs |
|--------|-------------|---------|
| `default_profile` | Profil par défaut pour `multiai launch` | `co`, `ds`, `za`, ... |
| `credential_store` | Méthode de stockage des clés | `auto`, `file`, `keychain`, `wincred`, `secret-service` |
| `theme` | Thème du menu interactif | `default`, `dark`, `light` |
| `language` | Langue de l'interface | `fr`, `en` |

## Variables d'environnement directes

Tu peux aussi passer les variables directement sans fichier de configuration. multiai lit les variables d'environnement standards de chaque fournisseur :

```bash
ANTHROPIC_API_KEY=sk-ant-... multiai launch -p co
```

## Voir aussi

- [Profils disponibles](/guide/profiles) — la liste complète des profils inclus
- [Profils personnalisés](/advanced/custom-profiles) — créer ses propres profils
- [Variables d'environnement](/reference/env-variables) — référence complète
