# Profils personnalises

Tu peux creer tes propres profils en utilisant des fichiers `.env` personnalises. C'est utile pour ajouter un fournisseur non inclus par defaut ou pour creer des variantes avec des modeles specifiques.

## Structure d'un profil .env

Un profil est un fichier `.env` place dans `~/.multiai/profiles/`. Le nom du fichier (sans l'extension `.env`) est le nom du profil.

```
~/.multiai/profiles/
├── co.env      # Profil "co"
├── ds.env      # Profil "ds"
└── mon-profil.env   # Profil "mon-profil"
```

## Format du fichier .env

```env
# ~/.multiai/profiles/mon-profil.env
TOOL=claude
MODEL=claude-sonnet-4-20250514
ANTHROPIC_API_KEY=sk-ant-...
ANTHROPIC_BASE_URL=https://api.custom-provider.com/v1
DESCRIPTION=Mon fournisseur personnalise
DISPLAY_NAME=Custom Provider
```

### Variables requises

| Variable | Description |
|----------|-------------|
| `TOOL` | CLI a utiliser : `claude`, `codex`, ou `opencode` |
| `ANTHROPIC_API_KEY` ou `OPENAI_API_KEY` | Cle API selon le fournisseur |

### Variables optionnelles

| Variable | Description |
|----------|-------------|
| `MODEL` | Modele a utiliser |
| `ANTHROPIC_BASE_URL` | URL de base API (pour fournisseurs compatibles) |
| `DISPLAY_NAME` | Nom affiche dans le menu et `multiai list` |
| `DESCRIPTION` | Description affichee dans `multiai list` |
| `PROVIDER` | Nom du fournisseur |

## Exemple : Fournisseur compatible Anthropic

```env
# ~/.multiai/profiles/mistral.env
TOOL=claude
MODEL=claude-sonnet-4-20250514
ANTHROPIC_API_KEY=votre-cle-mistral
ANTHROPIC_BASE_URL=https://api.mistral.ai/v1
DISPLAY_NAME=Mistral AI
DESCRIPTION=Mistral AI via Claude Code
PROVIDER=Mistral
```

## Exemple : Modele OpenAI personnalise

```env
# ~/.multiai/profiles/oa-custom.env
TOOL=codex
MODEL=gpt-4-turbo
OPENAI_API_KEY=sk-proj-...
DISPLAY_NAME=OpenAI GPT-4 Turbo
DESCRIPTION=GPT-4 Turbo via Codex CLI
PROVIDER=OpenAI
```

## Exemple : Serveur local (Ollama / LM Studio)

```env
# ~/.multiai/profiles/local.env
TOOL=codex
MODEL=llama3
OPENAI_API_KEY=not-needed
OPENAI_BASE_URL=http://localhost:1234/v1
DISPLAY_NAME=LM Studio Local
DESCRIPTION=Modele local via LM Studio
PROVIDER=Local
```

## Utilisation

Une fois le fichier cree, le profil est automatiquement disponible :

```bash
multiai list
multiai launch -p mon-profil
```

## Bonnes pratiques

1. **Nommage** : utilises des noms courts, sans espaces, en kebab-case (`mon-profil`)
2. **Clefs API** : ne commit jamais un fichier `.env` contenant des clefs. Utilise le credential store pour la production
3. **Un seul profil par fichier** : un fichier `.env` = un profil
4. **Description** : ajoute toujours `DISPLAY_NAME` et `DESCRIPTION` pour que le profil soit identifiable dans `multiai list`

## Voir aussi

- [Profils YAML](/advanced/yaml-profiles) — alternative avancee au format .env
- [Profils par projet](/advanced/project-profiles) — configuration .multiai.yaml par projet
- [Variables d'environnement](/reference/env-variables) — reference de toutes les variables
