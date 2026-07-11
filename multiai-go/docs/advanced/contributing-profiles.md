# Contribuer un profil au registre communautaire

multiai dispose d'un **registre communautaire** heberge sur GitHub : [github.com/lrochetta/profiles-multiai](https://github.com/lrochetta/profiles-multiai). Tout le monde peut soumettre un profil pour le partager avec la communaute.

Un profil communautaire est un fichier YAML valide, teste et documente, qui definit une configuration de lancement pour un CLI (`claude`, `codex`, `opencode`) chez un fournisseur donne.

---

## Table des matieres

- [Prerequis](#prerequis)
- [Structure du depot](#structure-du-depot)
- [Convention de nommage](#convention-de-nommage)
- [Template de profil](#template-de-profil)
- [Variables et interpolation](#variables-et-interpolation)
- [Shortcuts et organisation](#shortcuts-et-organisation)
- [Validation locale](#validation-locale)
- [Tests a passer](#tests-a-passer)
- [Processus de soumission PR](#processus-de-soumission-pr)
- [Bonnes pratiques](#bonnes-pratiques)
- [FAQ](#faq)
- [Voir aussi](#voir-aussi)

---

## Prerequis

Avant de soumettre un profil :

- **multiai v0.5.0+** installe (`multiai version`)
- **Git** configure avec un compte GitHub
- **YAML** : connaissance de base du format
- **Compte GitHub** pour fork et PR

---

## Structure du depot

Le depot communautaire [profiles-multiai](https://github.com/lrochetta/profiles-multiai) est organise ainsi :

```
profiles-multiai/
├── index.json                 # Index genere automatiquement (ne pas editer)
├── profiles/
│   ├── anthropic/
│   │   └── co.yaml            # Profil Claude officiel
│   ├── deepseek/
│   │   └── ds.yaml            # Profil DeepSeek V4 Pro
│   ├── openrouter/
│   │   └── or-fusion.yaml     # Profil OpenRouter Fusion
│   └── communaute/            # Profils soumis par la communaute
│       ├── <fournisseur>/
│       │   └── <profil>.yaml
│       └── ...
├── tests/                     # Scripts de validation CI
│   └── validate.sh
├── CONTRIBUTING.md            # Ce document
└── LICENSE
```

Les profils officiels sont dans `profiles/<fournisseur>/`. Les contributions communautaires vont dans `profiles/communaute/<fournisseur>/`.

---

## Convention de nommage

### Nom du fichier

Le nom du fichier YAML est le **shortcut** du profil, en **kebab-case** :

```
<shortcut>.yaml
```

Regles :
- Lettres minuscules, chiffres et tirets (`-`) uniquement
- Pas d'underscores, pas d'espaces
- Maximum 24 caracteres
- Commence par une lettre
- Significatif et facile a retenir

Exemples valides :
```
co.yaml              # Claude officiel
ds.yaml              # DeepSeek V4 Pro
codex55.yaml         # Codex GPT-5.5
or-fusion.yaml       # OpenRouter Fusion
groq-llama.yaml      # Groq + Llama
perplexity-pro.yaml  # Perplexity Pro
```

Exemples invalides :
```
Mon_Profil.yaml      # Underscore interdit
3profil.yaml         # Commence par un chiffre
trop-long-pour-etre-valide.yaml  # Trop long (>24 caracteres)
```

### Dossier fournisseur

Le dossier parent suit le nom du fournisseur en kebab-case :

```
profiles/communaute/perplexity/perplexity-pro.yaml
profiles/communaute/groq/groq-llama.yaml
profiles/communaute/together/together-mixtral.yaml
```

### ID du profil

Dans le fichier YAML, le champ `id` doit correspondre au nom de fichier (sans l'extension `.yaml`) :

```yaml
id: perplexity-pro   # fichier : perplexity-pro.yaml
```

---

## Template de profil

### Template minimal

```yaml
# profiles/communaute/<fournisseur>/<shortcut>.yaml
id: <shortcut>
tool: <claude|codex|opencode>
display_name: "<Nom affiche dans le menu>"
description: "<Description courte du profil>"
provider: "<Nom du fournisseur>"
env:
  API_KEY_VAR: "${VARIABLE_ENV}"
  BASE_URL: "https://api.example.com/v1"
```

### Template complet

```yaml
# profiles/communaute/mon-fournisseur/mon-profil.yaml
id: mon-profil
shortcut: mp
tool: claude
tool_label: Claude
display_name: "Mon Fournisseur"
description: "Description detaillee pour multiai list"
provider: "MonFournisseur"
order: 50
command: claude
args: ["--model", "claude-sonnet-4-20250514"]
clear_env: true
region: eu
fallback: ["co", "ds"]
required_secrets:
  - MON_CLE_API
skip_secret_check: false
env:
  ANTHROPIC_API_KEY: "${MON_CLE_API}"
  ANTHROPIC_BASE_URL: "https://api.mon-fournisseur.com/v1"
  CUSTOM_VAR: "valeur-personnalisee"
hooks:
  before_launch:
    - command: "./scripts/check-vpn.sh"
      shell: bash
  after_launch:
    - command: "./scripts/notify.sh"
      shell: bash
```

### Champs disponibles

| Champ | Requis | Description |
|-------|--------|-------------|
| `id` | Oui | Identifiant unique du profil (kebab-case, correspond au nom de fichier) |
| `shortcut` | Non | Alias court pour `multiai launch -p <shortcut>` (defaut = `id`) |
| `tool` | Oui | CLI a utiliser : `claude`, `codex`, ou `opencode` |
| `tool_label` | Non | Libelle affiche pour le CLI |
| `display_name` | Oui | Nom affiche dans le menu et `multiai list` |
| `description` | Recommande | Description affichee dans `multiai list` |
| `provider` | Recommande | Nom du fournisseur |
| `order` | Non | Ordre d'affichage dans le menu (defaut: 9999) |
| `command` | Non | Binaire a executer (defaut = `tool`) |
| `args` | Non | Arguments par defaut passes au CLI |
| `clear_env` | Non | Nettoyer l'environnement avant lancement (defaut: true) |
| `region` | Non | Code region pour le groupement (`eu`, `us`, `cn`, etc.) |
| `fallback` | Non | Liste de shortcuts de repli en cas d'echec (ex: `["co", "ds"]`) |
| `required_secrets` | Non | Liste des variables secretes obligatoires |
| `skip_secret_check` | Non | Desactiver la verification des secrets (defaut: false) |
| `env` | Oui | Variables d'environnement du profil |
| `hooks` | Non | Scripts before/after launch (voir [Hooks](/advanced/hooks)) |

---

## Variables et interpolation

### Variables obligatoires

Chaque profil definit au minimum la variable contenant la cle API :
- `ANTHROPIC_API_KEY` pour les profils `tool: claude`
- `OPENAI_API_KEY` pour les profils `tool: codex` ou `tool: opencode`

### Interpolation depuis l'environnement

Utilise la syntaxe `${VAR}` pour referencer une variable d'environnement existante :

```yaml
env:
  ANTHROPIC_API_KEY: "${MA_CLE_FOURNISSEUR}"
  ANTHROPIC_BASE_URL: "https://api.fournisseur.com/v1"
```

multiai remplace automatiquement `${MA_CLE_FOURNISSEUR}` par la valeur de la variable d'environnement au moment du lancement.

### Variables d'environnement exposees aux hooks

Les hooks recoivent automatiquement ces variables contextuelles :

| Variable | Description |
|----------|-------------|
| `MULTIAI_PROFILE` | Nom du profil utilise |
| `MULTIAI_TOOL` | CLI lance (`claude`, `codex`, `opencode`) |
| `MULTIAI_EXIT_CODE` | Code de sortie du CLI (after uniquement) |
| `MULTIAI_PROJECT_DIR` | Repertoire du projet (si `.multiai.yaml` trouve) |
| `MULTIAI_START_TIME` | Timestamp Unix du debut du lancement |

### Interdiction des secrets en dur

Les fichiers de profils **ne doivent jamais contenir de cle API en clair**. Utilisez toujours la syntaxe `${VAR}`. Les secrets sont stockes dans le credential store natif (`multiai config --store native`) ou dans un fichier `.env` local non versionne.

```yaml
# CORRECT
env:
  ANTHROPIC_API_KEY: "${ANTHROPIC_API_KEY}"

# INCORRECT — Ne JAMAIS commit une cle en dur
env:
  ANTHROPIC_API_KEY: "sk-ant-12345abcde..."
```

---

## Shortcuts et organisation

### Regles de shortcut

- 2 a 8 caracteres de preference
- Kebab-case (`or-fusion`, `codex55`, `ocqwen`)
- Unique dans tout le registre (pas de doublon entre profils)
- Evocateur du fournisseur ou du modele

### Conventions par type

| Type | Convention | Exemple |
|------|------------|---------|
| Claude Code | 2-4 lettres | `co`, `ds`, `cg` |
| Codex CLI | `codex` + modele | `codex55`, `codex-qwen` |
| OpenCode | `oc` + fournisseur | `oc5`, `ocqwen`, `ocmini` |
| OpenRouter | `or-` + modele | `or-fusion`, `or-codex` |
| Requesty | `req-` + CLI | `req-cc`, `req-codex` |
| Communaute | `<fournisseur>-<modele>` | `groq-llama`, `perplexity-pro` |

### Organisation dans le registre

Les profils officiels sont ranges par fournisseur dans `profiles/<fournisseur>/`. Les profils communautaires sont ranges sous `profiles/communaute/<fournisseur>/`. Un meme fournisseur peut avoir plusieurs profils :

```
profiles/communaute/groq/
├── groq-llama.yaml          # Groq + Llama via Codex
└── groq-mixtral.yaml        # Groq + Mixtral via Codex
```

---

## Validation locale

Avant de soumettre, validez votre profil localement avec les outils fournis.

### 1. Validation YAML de base

```bash
# Verifier que le YAML est bien forme
python -c "import yaml; yaml.safe_load(open('mon-profil.yaml'))"
```

### 2. Validation avec multiai

```bash
# Verifier que multiai charge correctement le profil
multiai list --json | jq '.[] | select(.id=="mon-profil")'

# Tester le lancement (sans cle API, verifier que la structure est correcte)
multiai launch -p mon-profil 2>&1 || true
```

### 3. Validation structurelle

Le script `tests/validate.sh` du depot registre verifie :

- Que le YAML est valide
- Que tous les champs requis sont presents
- Que les valeurs sont du type attendu
- Que le shortcut est unique

```bash
# Depuis la racine du depot registre
bash tests/validate.sh profiles/communaute/mon-fournisseur/mon-profil.yaml
```

### 4. Tests unitaires (contributeurs avances)

Si vous contribuez aussi des modifications au code Go de multiai, executez la suite complete de tests :

```bash
# Depuis multiai-go/
go test -race -count=1 ./internal/profile/...
go test -race -count=1 ./internal/registry/...
```

---

## Tests a passer

Avant d'ouvrir une PR, votre profil doit satisfaire ces conditions :

### Checklist de validation

- [ ] Le fichier YAML est syntaxiquement valide
- [ ] `tool` est l'un de : `claude`, `codex`, `opencode`
- [ ] `id` correspond au nom de fichier (sans `.yaml`)
- [ ] `id` est en kebab-case, lettres minuscules, max 24 caracteres
- [ ] `display_name` est present et non vide
- [ ] `env` contient au moins une variable de cle API (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, ou une variable personnalisee)
- [ ] Aucune valeur de cle API en dur (toutes les cles sont en `${VAR}`)
- [ ] Le shortcut est unique dans le registre
- [ ] La description est renseignee
- [ ] Le dossier parent suit la convention `communaute/<fournisseur>/`
- [ ] `tests/validate.sh` passe sans erreur
- [ ] `multiai list --json` affiche le profil correctement

### Tests CI automatiques

Quand vous ouvrez une PR sur le depot registre, la CI execute automatiquement :

1. **Lint YAML** — `yamllint` sur tous les fichiers
2. **Validation structurelle** — `tests/validate.sh` verifie les champs requis
3. **Deduplication** — Verifie que le shortcut n'existe pas deja
4. **Integration** — Monte un environnement de test avec multiai et tente de charger le profil

Voir [le workflow CI du registre](https://github.com/lrochetta/profiles-multiai/blob/main/.github/workflows/ci.yml) pour les details.

---

## Processus de soumission PR

### Etape 1 : Fork et clone

```bash
# Forker le depot sur GitHub, puis :
git clone https://github.com/<votre-utilisateur>/profiles-multiai.git
cd profiles-multiai
git remote add upstream https://github.com/lrochetta/profiles-multiai.git
```

### Etape 2 : Creer une branche

```bash
git checkout -b feat/<shortcut>-profile
```

Utilisez le prefixe `feat/` suivi du shortcut du profil.

### Etape 3 : Ajouter le profil

Creez le fichier dans le dossier approprie :

```bash
mkdir -p profiles/communaute/<fournisseur>
# Creez et editez le fichier YAML
```

### Etape 4 : Valider localement

```bash
bash tests/validate.sh profiles/communaute/<fournisseur>/<shortcut>.yaml
multiai list --json | grep <shortcut>
```

### Etape 5 : Commit et push

Le message de commit suit la convention [Conventional Commits](https://www.conventionalcommits.org/) :

```bash
git add profiles/communaute/<fournisseur>/<shortcut>.yaml

# Exemples de messages de commit valides :
git commit -m "feat(profile): add Groq Llama profile"
git commit -m "feat(profile): add Perplexity Pro via Codex CLI"
```

Format : `feat(profile): add <Description courte>`

### Etape 6 : Ouvrir la Pull Request

1. Poussez votre branche : `git push origin feat/<shortcut>-profile`
2. Ouvrez une PR sur [github.com/lrochetta/profiles-multiai](https://github.com/lrochetta/profiles-multiai)
3. Remplissez le template de PR avec :

```markdown
## Description

Ajout du profil <nom> pour <fournisseur>, utilisant <CLI>.

## Provider

- Nom : <Fournisseur>
- URL de creation de cle : https://...
- URL de l'API : https://api.example.com/v1

## Checklist

- [ ] YAML valide (`yamllint`)
- [ ] `tests/validate.sh` passe
- [ ] Aucune cle API en dur
- [ ] Shortcut unique dans le registre
- [ ] Documentation du fournisseur incluse

## Test

```bash
multiai launch -p <shortcut>
```
```

### Etape 7 : Review et merge

- Un mainteneur examine la PR sous 48h ouvrées
- La CI doit etre verte
- Une fois approuvee, la PR est `squash merge` dans `main`
- Le profil apparait dans l'index `index.json` apres le prochain rebuild automatique

---

## Bonnes pratiques

### Documentation du profil

Ajoutez un commentaire d'en-tete dans le fichier YAML expliquant le profil :

```yaml
# Profil Groq Llama 3.3 70B
# - CLI : Codex
# - Cle API : https://console.groq.com/keys
# - Modele : llama-3.3-70b-versatile
# - Configuration : API compatible OpenAI
```

### Variables d'environnement documentees

Si le profil necessite des variables specifiques autre que la cle API, documentez-les dans un commentaire :

```yaml
# Variables requises :
#   GROQ_API_KEY       - Cle API Groq (https://console.groq.com/keys)
#   GROQ_BASE_URL      - (optionnel) URL de base, defaut: https://api.groq.com/openai/v1
```

### Profil sans cle API

Certains profils utilisent l'authentification integree du CLI (ex: `co` avec `claude login`, `codex55` avec Codex login). Dans ce cas, mettez `skip_secret_check: true` et ne definissez pas de variable de cle API :

```yaml
id: codex55
tool: codex
display_name: "Codex GPT-5.5"
description: "Codex CLI avec GPT-5.5 (authentification integree)"
skip_secret_check: true
env: {}
```

### Heritage de configuration

Vous pouvez utiliser `extends` pour heriter d'un profil existant, ce qui evite de dupliquer des champs communs :

```yaml
id: groq-pro
extends: co
display_name: "Groq Pro"
description: "Groq Llama via Claude Code"
env:
  ANTHROPIC_API_KEY: "${GROQ_API_KEY}"
  ANTHROPIC_BASE_URL: "https://api.groq.com/openai/v1"
```

### Tests manuels

Avant de soumettre, testez le profil dans differents contextes :

```bash
# Test d'affichage dans la liste
multiai list --json | jq '.[] | select(.id=="<shortcut>")'

# Test de lancement en mode dry-run (necessite une cle valide)
multiai launch -p <shortcut> --dry-run

# Test avec timeout court
multiai launch -p <shortcut> --timeout 10s

# Test de fallback (si defini)
multiai launch -p <shortcut> && echo "OK" || echo "Fallback..."
```

---

## FAQ

### Puis-je soumettre un profil pour un fournisseur deja existant ?

Oui. Si vous ajoutez une variante (modele different, CLI different), placez-la dans le dossier du fournisseur avec un nom distinct. Par exemple, si Groq a deja `groq-llama.yaml`, vous pouvez ajouter `groq-mixtral.yaml`.

### Mon profil est rejete par la CI. Que faire ?

Consultez les logs de la CI :
1. **YAML invalide** — utilisez `yamllint` pour corriger la syntaxe
2. **Shortcut duplique** — changez le shortcut pour qu'il soit unique
3. **Champ obligatoire manquant** — ajoutez `tool`, `display_name` ou `env`
4. **Cle API en dur** — remplacez la valeur en clair par `${MA_VARIABLE}`

### Puis-je soumettre un profil sans multiai installe ?

Non, la validation CI necessite `multiai list --json`. Utilisez l'image Docker fournie ou installez multiai d'abord.

### Mon profil est accepte. Quand sera-t-il disponible ?

Apres le merge de la PR, l'index est reconstruit automatiquement dans l'heure. Le profil est alors disponible via `multiai profile search <query>` et `multiai profile install <name>`.

### Comment mettre a jour mon profil existant ?

Ouvrez une nouvelle PR modifiant le fichier YAML existant. Utilisez le prefixe `fix(profile):` ou `feat(profile):` selon le changement.

---

## Voir aussi

- [Profils YAML](/advanced/yaml-profiles) — format YAML detaille
- [Profils personnalises (.env)](/advanced/custom-profiles) — format .env
- [Configuration projet (.multiai.yaml)](/advanced/project-config)
- [Hooks before/after launch](/advanced/hooks)
- [Commandes de reference](/reference/commands)
- [Variables d'environnement](/reference/env-variables)
- [Registre communautaire](https://github.com/lrochetta/profiles-multiai)
- [Guide des profils](/guide/profiles)
