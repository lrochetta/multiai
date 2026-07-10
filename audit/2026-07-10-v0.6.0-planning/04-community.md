---

# Epic — Registre Communautaire & Visibilité (v0.6.0)

**Epic Owner :** Atlas (Strategist) + Forge (Architect-Dev)
**Version cible :** v0.6.0
**Score actuel visibilité :** 1/10
**Score cible :** 6/10
**Dépendance globale :** v0.5.0 livrée (Sprint 1-4 complété)
**KPI epic :** 100+ stars GitHub, 5+ profils communautaires, 3+ contributeurs externes

---

## S8.1 — Dépôt GitHub dédié au registre communautaire de profils

**Priorité :** BLOCKER
**Effort :** 4h
**Agent :** Forge + Nexus

### Objectif
Créer un dépôt GitHub public `github.com/lrochetta/profiles-multiai` qui sert de registre officiel aux profils soumis par la communauté, avec une structure de répertoire normalisée, un index machine-readable, et un processus de soumission par Pull Request.

### Description technique

Le dépôt `profiles-multiai` est un référentiel de profils YAML validés et approuvés par la communauté. Chaque soumission est un fichier YAML unique suivant un schéma strict (hérité de `ProfileYAML` dans `internal/profile/yaml.go`). La structure du dépôt est la suivante :

```
profiles-multiai/
├── profiles/
│   ├── anthropic/
│   │   ├── claude-sonnet-4.yaml       # Profil approuvé
│   │   └── claude-opus-4.yaml
│   ├── deepseek/
│   │   └── deepseek-v4-pro.yaml
│   ├── openrouter/
│   │   ├── openrouter-claude.yaml
│   │   └── openrouter-gpt5.yaml
│   └── community/                      # Profils non officiels, tag "community"
│       └── my-custom-vendor.yaml
├── index.json                          # Index machine-readable (regénéré à chaque merge)
├── schemas/
│   └── profile-schema.json             # JSON Schema validant les profils
├── CONTRIBUTING.md                     # Spécifique au registre
├── CODEOWNERS                          # Équipe de review des profils
└── README.md                           # Présentation + statistiques
```

L'`index.json` est le contrat d'interface entre le registre et le CLI multiai. Il contient :
```json
{
  "schema_version": 1,
  "updated_at": "2026-07-10T12:00:00Z",
  "profiles": [
    {
      "id": "claude-sonnet-4",
      "name": "Claude Sonnet 4",
      "display_name": "Claude Sonnet 4 (Anthropic)",
      "description": "Profil officiel Claude Code avec Sonnet 4",
      "author": "multiai-team",
      "provider": "anthropic",
      "tool": "claude",
      "tags": ["official", "anthropic", "claude-code"],
      "path": "profiles/anthropic/claude-sonnet-4.yaml",
      "sha256": "abc123...",
      "stars": 0,
      "installs": 0
    }
  ]
}
```

La génération de l'index est automatisée par une GitHub Action déclenchée sur chaque merge dans `main` : elle scanne `profiles/`, valide chaque YAML, calcule les SHA256, et commit le nouvel `index.json`. Une seconde Action pousse l'index vers GitHub Pages (`https://lrochetta.github.io/profiles-multiai/index.json`) pour un accès HTTPS direct sans API GitHub.

Le dépôt est configuré avec :
- Protection de branche `main` : requiert PR + review + CI verte
- `CODEOWNERS` : `@lrochetta` + 2 maintainers initiaux
- `dependabot` désactivé (pas de dépendances)
- License MIT (identique au projet parent)

### Fichiers impactés

- `https://github.com/lrochetta/profiles-multiai` (nouveau dépôt)
  - `profiles/` + sous-répertoires par provider
  - `index.json` (généré)
  - `schemas/profile-schema.json`
  - `CONTRIBUTING.md`
  - `CODEOWNERS`
  - `README.md`
  - `.github/workflows/generate-index.yml`
  - `.github/workflows/deploy-pages.yml`

### Tests attendus

- [ ] `index.json` est valide JSON Schema après génération
- [ ] Chaque profil YAML est valide contre `profile-schema.json`
- [ ] La CI échoue si un YAML est invalide (champ manquant, type incorrect)
- [ ] La CI échoue si un profil contient une vraie clé API (gitleaks)
- [ ] La CI échoue si un profil tente de définir `CLEAR_ENV=false` (sécurité)
- [ ] L'index est accessible via HTTPS : `curl https://lrochetta.github.io/profiles-multiai/index.json`

### Résultat attendu

```
✅ Dépôt profiles-multiai créé et public
✅ Structure de répertoire normalisée
✅ index.json machine-readable accessible en HTTPS
✅ CI de validation et génération d'index
✅ 3+ profils officiels seedés (Anthropic, DeepSeek, OpenRouter)
```

### Definition of Done

- [ ] Le dépôt est accessible sur `github.com/lrochetta/profiles-multiai`
- [ ] `index.json` est livré sur GitHub Pages
- [ ] 3 profils seed de démonstration sont mergés
- [ ] La CI `generate-index` est verte
- [ ] README avec badges et instructions de soumission
- [ ] CODEOWNERS configuré

### Risques

- **Nom déjà pris** : `profiles-multiai` peut être indisponible → alternatives : `multiai-profiles`, `multiai-community-profiles`
- **Maintien à long terme** : l'équipe doit review les PRs de profils dans les 72h → risque de bottleneck si popularité augmente
- **Index Pages non disponible** : GitHub Pages peut être désactivé sur le compte → fallback : utiliser `raw.githubusercontent.com`

### Dépendances

- Aucune (dépôt indépendant, peut être créé en parallèle du code CLI)
- Source d'inspiration : `homebrew/homebrew-core` pour la structure, `nixpkgs` pour le processus de review

---

## S8.2 — Validation automatique CI des profils soumis

**Priorité :** BLOCKER
**Effort :** 3h
**Agent :** Forge + Sentinel
**Dépend de :** S8.1

### Objectif
Pipeline CI dans `profiles-multiai` qui garantit que chaque profil soumis est syntaxiquement valide, ne contient pas de secrets, respecte les contraintes de sécurité, et passe un lint métier avant d'être mergé.

### Description technique

Le workflow CI `.github/workflows/validate-profiles.yml` s'exécute sur chaque PR modifiant des fichiers dans `profiles/`. Il exécute les étapes suivantes dans l'ordre, en s'arrêtant à la première erreur :

1. **Lint YAML** : vérifie que chaque fichier `.yaml` est un YAML valide (`yamllint` ou équivalent Go)
2. **Validation schéma** : chaque profil est validé contre `schemas/profile-schema.json` à l'aide de `go-jsonschema` ou d'une validation Go dédiée
3. **Vérification des champs obligatoires** :
   - `id` présent et unique dans le dépôt
   - `display_name` présent
   - `tool` dans la liste autorisée (`claude`, `codex`, `opencode`)
   - `env` non vide
   - `required_secrets` référencent des clés présentes dans `env`
4. **Scan de secrets** : `gitleaks` détecte les patterns de clés API en clair (`sk-*`, `ANTHROPIC_API_KEY=sk-...`, etc.)
5. **Scan de sécurité** :
   - `CLEAR_ENV` ne doit PAS être `false` (sécurité : un profil `clear_env=false` hérite du PATH parent, ce qui est un vecteur d'attaque)
   - `COMMAND` ne doit pas contenir de pipe, `$(...)`, `` `...` ``, `;`, `|`, `>`
   - `env` ne doit pas contenir de variable système de la whitelist (PATH, HOME, etc.) — ces variables sont déjà injectées par multiai
6. **Vérification de non-duplication** : l'`id` du profil ne doit pas déjà exister dans `index.json` ni dans un autre fichier de la PR
7. **Génération de l'index de test** : exécuter la génération `index.json` pour vérifier que l'index est toujours valide après merge
8. **Post-check commentaire** : si tout passe, le bot commente la PR avec un résumé :
   ```
   ✅ Validation réussie
   - Profils : 1
   - Schéma : OK
   - Secrets : 0 détecté
   - Index : OK
   ```

Le workflow utilise exclusivement des outils disponibles dans l'écosystème Go + GitHub Actions, sans dépendance externe (pas de Python, pas de Node). Pour la validation JSON Schema, on utilise soit `github.com/santhosh-tekuri/jsonschema` (Go pur), soit un validateur Go custom qui échoue vite et avec des messages d'erreur lisibles.

### Fichiers impactés

- `https://github.com/lrochetta/profiles-multiai/.github/workflows/validate-profiles.yml`
- `https://github.com/lrochetta/profiles-multiai/schemas/profile-schema.json`
- `https://github.com/lrochetta/profiles-multiai/.github/workflows/generate-index.yml` (peut être fusionné)
- Optionnel : `scripts/validate-profile.sh` ou `cmd/validate-profile/main.go` (outil de validation réutilisable)

### Tests attendus

- [ ] Profil valide → CI verte, commentaire posté
- [ ] YAML invalide → CI rouge, message "invalid YAML"
- [ ] Champ `id` manquant → CI rouge
- [ ] Clé API en clair → CI rouge (gitleaks détecte)
- [ ] `CLEAR_ENV=false` → CI rouge avec message "security: clear_env must be true"
- [ ] `COMMAND` avec `$(...)` → CI rouge
- [ ] `id` dupliqué avec `main` → CI rouge
- [ ] PR avec 3 profils valides → CI verte, index généré correctement

### Résultat attendu

```
✅ CI de validation complète
✅ Schéma JSON Schema publié
✅ 8 règles de validation implémentées
✅ Commentaire automatique sur PR
✅ 0 faux positif sur un profil officiel seedé
```

### Definition of Done

- [ ] Le workflow `validate-profiles.yml` est vert sur une PR de test
- [ ] `profile-schema.json` couvre tous les champs de `ProfileYAML`
- [ ] Tous les cas de rejet listés ci-dessus sont testés
- [ ] Le post-check commentaire s'affiche sur la PR
- [ ] La documentation du schéma est incluse dans `CONTRIBUTING.md` du registre

### Risques

- **Faux positifs gitleaks** : les placeholders (`PASTE_YOUR_KEY_HERE`) doivent être exclus → configurer `.gitleaks.toml` dans le dépôt registre
- **Évolution du schéma** : si le format `ProfileYAML` change dans multiai, le schéma du registre doit être mis à jour → ajouter une CI de compatibilité cross-repo ou une version de schéma
- **Temps d'exécution** : gitleaks + validation Go + génération index → sous 2 minutes sur ubuntu-latest

### Dépendances

- S8.1 (dépôt registre créé)
- Schéma JSON calqué sur `ProfileYAML` dans `internal/profile/yaml.go`

---

## S8.3 — Commande `multiai profile search <query>`

**Priorité :** HIGH
**Effort :** 5h
**Agent :** Forge
**Dépend de :** S8.1, S8.2

### Objectif
Permettre à l'utilisateur de rechercher des profils dans le registre communautaire directement depuis le terminal : `multiai profile search claude` retourne la liste des profils correspondant au terme de recherche.

### Description technique

La sous-commande `multiai profile search` est une nouvelle entrée dans le registre des commandes de `main.go`. Elle fait partie d'un groupe `profile` avec structure :

```
multiai profile search <query>        # Recherche textuelle
multiai profile search --tag claude   # Recherche par tag
multiai profile search --provider anthropic  # Recherche par fournisseur
multiai profile search --json        # Sortie JSON
multiai profile search --offline     # Utiliser le cache local uniquement
```

**Architecture :**

1. **Nouveau package** `internal/registry/` avec les fichiers :
   - `registry.go` — définition du type `Registry` et des méthodes `Search()`, `IndexURL()`, `CachePath()`
   - `index.go` — structure `ProfileIndex` mirrorant `index.json`
   - `cache.go` — cache local du registre (fichier JSON dans `~/.multiai/registry-cache.json`)
   - `client.go` — HTTP fetch de l'index depuis GitHub Pages ou fallback `raw.githubusercontent.com`

2. **Algorithme de recherche** :
   - Télécharger `index.json` depuis `https://lrochetta.github.io/profiles-multiai/index.json` (cache 1h)
   - Filtrer les profils dont `display_name`, `description`, `provider`, `tags` ou `id` contiennent le terme (case-insensitive)
   - Ordonner par : profils officiels d'abord, puis par nombre d'installations (via étoiles futures S8.4)
   - Afficher les résultats au format tableau (terminal) ou JSON (`--json`)

3. **Cache** :
   - Stocké dans `~/.multiai/registry-cache.json`
   - Durée de vie : 1 heure (configurable via `MULTIAI_REGISTRY_CACHE_TTL`)
   - Rafraîchissement silencieux en arrière-plan (goroutine avec timeout 10s)
   - `--offline` force l'utilisation du cache (pas de requête réseau)

4. **Sortie terminal** (style `multiai list`) :

   ```
   Résultats pour "claude" (8 profils)
   
   ID                       DISPLAY NAME                    PROVIDER      TOOL      INSTALLS
   claude-sonnet-4          Claude Sonnet 4 (Anthropic)     anthropic     claude    ★★★★☆
   claude-opus-4            Claude Opus 4 (Anthropic)       anthropic     claude    ★★★☆☆
   openrouter-claude        OpenRouter Claude Sonnet 4      openrouter    claude    ★★★☆☆
   ...
   
   Conseil : multiai profile install claude-sonnet-4
   ```

5. **Gestion d'erreurs** :
   - Pas de réseau → utiliser le cache, afficher un warning
   - Cache vide + pas de réseau → message "Aucun cache. Essayez plus tard ou vérifiez votre connexion."
   - Index invalide → utiliser le cache précédent si disponible, sinon erreur

**Nouveaux messages i18n :**
- `registry_search_no_results` : "Aucun profil trouvé pour \"%s\""
- `registry_search_offline` : "[i] Mode hors ligne (cache du %s)"
- `registry_search_error` : "[X] Erreur de recherche : %v"
- `registry_search_usage` : (usage text)

### Fichiers impactés

- `multiai-go/cmd/multiai/cmd_profile.go` (nouveau) — parseur `multiai profile search`
- `multiai-go/internal/registry/registry.go` (nouveau)
- `multiai-go/internal/registry/index.go` (nouveau)
- `multiai-go/internal/registry/cache.go` (nouveau)
- `multiai-go/internal/registry/client.go` (nouveau)
- `multiai-go/internal/registry/registry_test.go` (nouveau)
- `multiai-go/cmd/multiai/main.go` — enregistrer la commande `profile`
- `multiai-go/internal/i18n/` — nouveaux messages

### Tests attendus

- [ ] `Search("claude")` retourne les profils contenant "claude" dans leur nom ou description
- [ ] `Search("")` retourne tous les profils
- [ ] `Search("zzzzzzz")` retourne une liste vide
- [ ] `Search("CLAUDE")` (case-insensitive) retourne les mêmes résultats que "claude"
- [ ] Cache hit : pas de requête HTTP si cache < 1h
- [ ] Cache miss : requête HTTP + mise à jour du cache
- [ ] `--offline` : retourne les résultats depuis le cache même si expiré
- [ ] Réseau indisponible + cache vide : message d'erreur explicite
- [ ] Sortie JSON : `--json` produit un tableau JSON valide
- [ ] Timeout réseau : fallback cache avec warning

### Résultat attendu

```
✅ multiai profile search implementation
✅ Cache 1h avec rafraîchissement silencieux
✅ Sortie texte + JSON
✅ Mode offline
✅ 10+ tests unitaires passant
```

### Definition of Done

- [ ] `multiai profile search deepseek` affiche les profils DeepSeek
- [ ] `multiai profile search --json` produit du JSON valide
- [ ] Tests > 70% de couverture sur `internal/registry/`
- [ ] `go build ./...` passe
- [ ] `go vet ./...` passe
- [ ] Documentation dans `docs/reference/commands.md`

### Risques

- **Dépendance réseau** : sans accès à GitHub Pages, la commande est vide → `--offline` atténue mais le premier appel nécessite le réseau → afficher un message clair : "Téléchargement du catalogue communautaire..."
- **Latence** : requête HTTP bloquante au premier appel → timeout de 10s max, goroutine de pré-chargement au démarrage de multiai
- **Cache corrompu** : un `registry-cache.json` corrompu → détecter et recréer silencieusement

### Dépendances

- S8.1 (index.json accessible en HTTPS)
- S8.2 (index.json valide)

---

## S8.4 — Commande `multiai profile install <name>`

**Priorité :** HIGH
**Effort :** 6h
**Agent :** Forge
**Dépend de :** S8.1, S8.3

### Objectif
Permettre l'installation d'un profil distant depuis le registre communautaire en une commande : `multiai profile install claude-sonnet-4` télécharge le YAML, le valide, et l'installe dans le dossier de profils local.

### Description technique

La sous-commande `multiai profile install` complète le workflow de découverte :

```
multiai profile install <id>              # Installer par ID (depuis index.json)
multiai profile install <id> --dir <path>  # Installer dans un dossier spécifique
multiai profile install <id> --dry-run     # Simuler sans écrire
multiai profile install -                  # Lire depuis stdin (pipe)
multiai profile search claude | head -1 | multiai profile install -
```

**Workflow d'installation :**

1. Résoudre l'ID du profil dans `index.json` (déjà téléchargé/caché par S8.3)
2. Télécharger le fichier YAML depuis `https://raw.githubusercontent.com/lrochetta/profiles-multiai/main/profiles/{provider}/{id}.yaml`
3. Valider le YAML : utiliser `profile.LoadYAML()` sur le contenu téléchargé
4. Vérifier les conflits : si un profil avec le même ID existe déjà dans le dossier local (`getProfilesDir()`), demander confirmation
5. Écrire le fichier : `fsutil.WriteFileAtomic()` dans le dossier des profils
6. Enregistrer la provenance : ajouter un commentaire en tête du fichier YAML :
   ```yaml
   # Installed via multiai profile install
   # Source: github.com/lrochetta/profiles-multiai
   # Profile ID: claude-sonnet-4
   # Installed at: 2026-07-10T14:30:00Z
   ```
7. Afficher le résumé :
   ```
   ✅ Profil "Claude Sonnet 4 (Anthropic)" installé
      Fichier : ~/.multiai/profiles/claude-sonnet-4.yaml
      Utilisation : multiai launch -p claude-sonnet-4
   ```

**Gestion des dépendances entre profils :**
Certains profils peuvent définir un `extends` (projet YAML). Si le profil installé `extends` un autre profil qui n'est pas installé localement, afficher un avertissement et proposer d'installer la dépendance :
```
⚠ Ce profil étend "base-claude" qui n'est pas installé.
  Commande : multiai profile install base-claude
```

**Sécurité :**
- `--dry-run` : affiche ce qui serait installé sans rien écrire
- Confirmation avant écrasement d'un profil existant (même nom)
- Refuser l'installation si le fichier source n'est pas dans `profiles-multiai`
- Vérification SHA256 du fichier téléchargé contre `index.json` (intégrité)
- Mode `--yes` pour usage scripté (saute les confirmations)

**Nouveaux fichiers :**

```
internal/registry/
├── install.go          # Logique d'installation
├── install_test.go     # Tests
```

Nouvelles entrées dans `cmd_profile.go` pour acheminer `install`.

**Nouveaux messages i18n :**
- `registry_install_success` : "Profil \"%s\" installé"
- `registry_install_exists` : "Le profil \"%s\" existe déjà. Remplacer ? [y/N]"
- `registry_install_conflict` : "[X] Conflit : %s"
- `registry_install_dep_missing` : "⚠ %s étend \"%s\" qui n'est pas installé"
- `registry_install_sha_mismatch` : "[X] Erreur d'intégrité : SHA256 mismatch"

### Fichiers impactés

- `multiai-go/cmd/multiai/cmd_profile.go` (nouveau ou étendu)
- `multiai-go/internal/registry/install.go` (nouveau)
- `multiai-go/internal/registry/install_test.go` (nouveau)
- `multiai-go/internal/registry/client.go` (étendre avec téléchargement fichier)
- `multiai-go/internal/i18n/` — nouveaux messages
- `multiai-go/internal/profile/profile.go` — peut nécessiter `LoadYAMLFromBytes()`

### Tests attendus

- [ ] Installation d'un profil valide → fichier créé dans `getProfilesDir()`
- [ ] Installation avec conflit → confirmation demandée, écrasement si confirmé
- [ ] Installation avec `--yes` → pas de confirmation, écrasement silencieux
- [ ] `--dry-run` → pas de fichier créé, message affiché
- [ ] SHA256 mismatch → erreur, fichier non installé
- [ ] ID inexistant dans index.json → erreur "profil introuvable"
- [ ] Installation depuis stdin → `echo "..." | multiai profile install -`
- [ ] Profil avec `extends` manquant → warning affiché
- [ ] Réseau indisponible → message "impossible de télécharger"
- [ ] Tests unitaires avec HTTP mocké pour le téléchargement

### Résultat attendu

```
✅ multiai profile install <id> fonctionnel
✅ Gestion des conflits (existant, confirmation)
✅ Vérification d'intégrité SHA256
✅ Installation depuis stdin
✅ Mode --dry-run
✅ Gestion des dépendances extends
✅ 10+ tests unitaires
```

### Definition of Done

- [ ] `multiai profile install claude-sonnet-4` installe le fichier
- [ ] `multiai list` affiche le profil installé
- [ ] `multiai launch -p claude-sonnet-4` lance le profil (après config des clés)
- [ ] Tests > 70% de couverture sur `internal/registry/install.go`
- [ ] `go build ./...` et `go test ./...` passent

### Risques

- **Nom de fichier en conflit** : le registre utilise `{provider}/{id}.yaml` mais localement le dossier est plat → si deux profils ont le même ID mais des providers différents, conflit → résolution : préfixer avec le provider (`anthropic-claude-sonnet-4.yaml`)
- **Profil cassé après installation** : un profil valide au moment de la soumission peut devenir invalide après une mise à jour de multiai (changement de schéma) → ajouter une validation post-installation avec la version locale de multiai
- **Dépendances circulaires** : si A extends B et B extends A → détecter et refuser

### Dépendances

- S8.1 (index.json + fichiers YAML accessibles)
- S8.3 (package `internal/registry/` pour le cache et client HTTP)
- `internal/profile/yaml.go` (fonction `LoadYAML`)

---

## S8.5 — Documentation contributeur pour les profils

**Priorité :** HIGH
**Effort :** 3h
**Agent :** Huldah (Technical Writer)

### Objectif
Produire une documentation complète guidant les contributeurs dans la soumission, la maintenance et l'évolution des profils communautaires.

### Description technique

Deux cibles documentaires :

#### 1. `profiles-multiai/CONTRIBUTING.md` (registre)

Document autonome dans le dépôt registre, couvrant :

```
# Contribuer au registre de profils multiai

## Prérequis
- Connaître le format YAML des profils multiai (lien vers docs/)
- Avoir testé le profil localement

## Soumettre un profil

### 1. Créer le fichier
```yaml
# profiles/anthopic/my-custom-profile.yaml
id: my-custom-profile
display_name: Mon Profil Custom
tool: claude
command: claude
env:
  ANTHROPIC_API_KEY: "${ANTHROPIC_API_KEY}"
  MODEL: claude-sonnet-4-20250514
required_secrets:
  - ANTHROPIC_API_KEY
```

### 2. Valider localement
```bash
# Utiliser l'outil de validation
go run cmd/validate-profile/main.go profiles/my-provider/my-profile.yaml
```

### 3. Ouvrir une Pull Request
- Titre : `profile: ajout <provider> <modele>`
- Label : `new-profile`
- Description : lien vers le provider, cas d'usage

### Règles de validation
Voir .github/workflows/validate-profiles.yml et schemas/profile-schema.json

### Convention de nommage
- ID : kebab-case, 3-30 caractères
- Provider : dossier correspondant au provider ID
- Version dans le nom si plusieurs modèles : claude-sonnet-4, claude-opus-4

## Maintenance
- Signaler un profil cassé : ouvrir une issue avec label `broken-profile`
- Proposer une mise à jour : PR avec label `update-profile`
- Dépréciation : issue `deprecate-profile`, période de grâce 90 jours
```

#### 2. `docs/guide/community-profiles.md` (site VitePress)

Page dans la documentation principale expliquant comment utiliser le registre depuis le CLI :

```
# Profils communautaires

multiai propose un registre de profils maintenu par la communauté.

## Rechercher un profil
```bash
multiai profile search claude
```

## Installer un profil
```bash
multiai profile install claude-sonnet-4
```

## Soumettre votre profil
1. Créez votre profil YAML
2. Testez-le : `multiai launch -p mon-profil`
3. Soumettez-le : https://github.com/lrochetta/profiles-multiai
```

#### 3. Mise à jour de `docs/reference/commands.md`

Ajouter les entrées pour `multiai profile search` et `multiai profile install`.

### Fichiers impactés

- `https://github.com/lrochetta/profiles-multiai/CONTRIBUTING.md`
- `multiai-go/docs/guide/community-profiles.md` (nouveau)
- `multiai-go/docs/guide/getting-started.md` (ajouter section "Profils communautaires")
- `multiai-go/docs/reference/commands.md` (ajouter sous-commandes profile)
- `multiai-go/docs/.vitepress/config.ts` (ajouter au menu latéral)

### Tests attendus

- [ ] La documentation est compréhensible par un nouveau contributeur
- [ ] Toutes les commandes mentionnées fonctionnent (documentation vérifiée)
- [ ] Le guide de soumission est suivi par un testeur externe
- [ ] Les liens internes (docs ↔ registre) sont valides
- [ ] La page `community-profiles.md` s'affiche correctement dans le site VitePress

### Résultat attendu

```
✅ CONTRIBUTING.md du registre complet
✅ Page community-profiles.md sur le site
✅ Commandes documentées dans la référence
✅ Guide de soumission pas-à-pas
✅ Liens croisés entre documentation et registre
```

### Definition of Done

- [ ] CONTRIBUTING.md publié et mergé dans le registre
- [ ] Site VitePress avec page community-profiles
- [ ] Documentation lisible en anglais (le public cible est international)
- [ ] Un profil testé via la procédure documentée
- [ ] `docs/reference/commands.md` mis à jour avec les sous-commandes `profile`

### Risques

- **Documentation obsolète** : si le format change, la documentation n'est plus à jour → garder les instructions proches du code (CONTRIBUTING.md dans le registre est plus facile à maintenir que le site VitePress)
- **Barrière linguistique** : documentation en anglais uniquement (audience internationale) mais erreurs potentielles → relecture par un locuteur natif

### Dépendances

- S8.1 (dépôt registre existant pour CONTRIBUTING.md)
- S8.3, S8.4 (commandes documentées, existantes au moment de la rédaction)

---

## S8.6 — Programme de feedback (GitHub Discussions)

**Priorité :** MEDIUM
**Effort :** 2h
**Agent :** Atlas + Nexus

### Objectif
Activer et structurer GitHub Discussions sur le dépôt principal `lrochetta/multiai` pour centraliser les retours utilisateurs, les questions, les partages d'usage, et créer un espace communautaire.

### Description technique

#### Activation GitHub Discussions

Dans les paramètres du dépôt `lrochetta/multiai` (Settings → Features → Discussions) :

1. **Configurer les catégories** :

| Catégorie | Description | Format |
|---|---|---|
| 🎉 Show and Tell | Partagez comment vous utilisez multiai | Annonce |
| 💡 Ideas | Propositions de fonctionnalités | Discussion libre |
| 🙏 Q&A | Questions et entraide | Question/Réponse |
| 📢 Announcements | Annonces officielles (mainteneur seulement) | Annonce |
| 🐛 Bug Reports (optional) | Alternative aux Issues pour les bugs non bloquants | Discussion libre |
| 🤝 Community Profiles | Partagez vos profils personnalisés | Discussion libre |

2. **Template d'annonce** (format fixe dans chaque mois) :
   ```
   # Journal communautaire - Juillet 2026
   
   ## Nouveautés
   - v0.5.0 disponible : [changelog]
   
   ## Profils de la semaine
   - [profil] par [contributeur]
   
   ## Chiffres
   - Stars : XXX
   - Profils communautaires : XX
   - Téléchargements : XXX
   ```

3. **Règles de modération** (épinglées dans la catégorie Announcements) :
   - Pas de spam
   - Pas de clés API en clair
   - Langues acceptées : anglais, français
   - Respect du Code of Conduct (lié vers CONTRIBUTING.md)

#### Workflow Issues ↔ Discussions

- Les **bugs** vont dans Issues (traçabilité, assignation)
- Les **questions** vont dans Discussions Q&A (pas de limite, pas de closed)
- Les **feature requests** peuvent commencer en Discussion Ideas, puis migrer en Issue si acceptées
- Les **profils perso** partagés en Discussions peuvent être promus en PR vers le registre (S8.1)

#### Automatisation

Créer une GitHub Action qui :
- Ajoute un label `discussion-answered` quand un mainteneur répond à une Q&A
- Ping les maintainers si une Discussion reste sans réponse > 48h
- Poste un message de bienvenue automatique pour les nouveaux participants :
  ```
  Bienvenue dans la communauté multiai ! 
  
  📖 Documentation : https://lrochetta.github.io/multiai
  💡 Guide de soumission : https://github.com/lrochetta/profiles-multiai
  ```

#### KPIs Discussions

| Métrique | Cible J+30 | Cible J+90 |
|---|---|---|
| Discussions totales | 20 | 100 |
| Taux de réponse < 24h | 80% | 90% |
| Profils partagés | 5 | 30 |
| Nouveaux participants | 10 | 50 |

### Fichiers impactés

- `https://github.com/lrochetta/multiai/settings` (UI GitHub — activation Discussions)
- `.github/DISCUSSION_TEMPLATE/q-and-a.md` (optionnel)
- `.github/workflows/discussion-welcome.yml` (optionnel)
- `README.md` (ajouter badge "Ask questions in Discussions" + lien)

### Tests attendus

- [ ] GitHub Discussions activé sur le dépôt
- [ ] Les 6 catégories sont configurées
- [ ] Un post de test dans chaque catégorie est possible
- [ ] Le bot de bienvenue répond aux nouveaux posts
- [ ] Les labels sont appliqués automatiquement

### Résultat attendu

```
✅ GitHub Discussions activé
✅ 6 catégories configurées
✅ Automatisation de bienvenue
✅ Règles de modération publiées
✅ Badge Discussions dans le README
```

### Definition of Done

- [ ] Discussions activé et visible sur le dépôt
- [ ] Posts de test dans chaque catégorie (supprimés ensuite)
- [ ] GitHub Action de bienvenue déployée
- [ ] README mis à jour avec lien vers Discussions
- [ ] Template d'annonce mensuelle créé

### Risques

- **Spam** : Discussions activé = porte ouverte au spam → configurer les filtres GitHub + modérateurs supplémentaires (2 co-mainteneurs à trouver)
- **Dilution des Issues** : risque que les bugs soient postés en Discussions au lieu d'Issues → éduquer via le template de bienvenue
- **Communauté vide** : sans utilisateurs actifs, Discussions sera un désert → publier des posts réguliers (tips, release notes) même sans engagement pour donner l'impression d'activité

### Dépendances

- Aucune (indépendant des autres stories)
- Recommandé : avoir au moins 50 stars avant d'activer (sinon Discussions vide)

---

## S8.7 — Badges supplémentaires (Codecov, Go Report Card, Scorecard)

**Priorité :** MEDIUM
**Effort :** 1h30
**Agent :** Nexus + Sentinel

### Objectif
Ajouter les badges de qualité, sécurité et métriques manquants dans le README pour donner confiance aux visiteurs et améliorer le référencement.

### Description technique

#### État actuel des badges (root README)

```
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-0078D4)](#installation)
[![Go](https://img.shields.io/badge/Go-1.23-blue)](https://go.dev)
[![npm](https://img.shields.io/npm/v/multiai)](https://www.npmjs.com/package/multiai)
[![Score](https://img.shields.io/badge/score-10%2F10-success)]()
```

#### Badges à ajouter (ordre recommandé)

```
<!-- En-tête : qualité et sécurité -->
[![Go Report Card](https://goreportcard.com/badge/github.com/lrochetta/multiai)](https://goreportcard.com/report/github.com/lrochetta/multiai)
[![Codecov](https://codecov.io/gh/lrochetta/multiai/branch/master/graph/badge.svg)](https://codecov.io/gh/lrochetta/multiai)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/lrochetta/multiai/badge)](https://securityscorecards.dev/viewer/?uri=github.com/lrochetta/multiai)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/12345/badge)](https://bestpractices.coreinfrastructure.org/projects/12345)

<!-- Métriques -->
[![GitHub Stars](https://img.shields.io/github/stars/lrochetta/multiai?style=social)](https://github.com/lrochetta/multiai)
[![GitHub Downloads](https://img.shields.io/github/downloads/lrochetta/multiai/total)](https://github.com/lrochetta/multiai/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/lrochetta/multiai.svg)](https://pkg.go.dev/github.com/lrochetta/multiai)
```

#### Configuration Codecov

1. Activer le dépôt sur [codecov.io](https://codecov.io) avec le compte GitHub
2. Ajouter le token CODECOV_TOKEN dans les secrets du dépôt GitHub
3. Ajouter un job `upload-coverage` dans `.github/workflows/ci.yml` :

```yaml
upload-coverage:
  needs: test
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: "1.23"
    - run: go test -coverprofile=coverage.out -covermode=atomic ./...
    - uses: codecov/codecov-action@v4
      with:
        token: ${{ secrets.CODECOV_TOKEN }}
        files: coverage.out
        fail_ci_if_error: true
```

4. Ajouter le badge Codecov dans le README

#### Configuration Go Report Card

L'URL Go Report Card est déjà fonctionnelle (basée sur `go vet` + `gofmt`). Vérifier qu'elle est active :
```bash
curl -s https://goreportcard.com/report/github.com/lrochetta/multiai | grep -o "A+"
```

#### Configuration OpenSSF Scorecard

1. Installer l'action Scorecard GitHub :
```yaml
scorecard:
  runs-on: ubuntu-latest
  permissions:
    security-events: write
    id-token: write
  steps:
    - uses: actions/checkout@v4
    - uses: ossf/scorecard-action@v2
      with:
        results_file: results.sarif
        results_format: sarif
        publish_results: true
    - uses: github/codeql-action/upload-sarif@v3
      with:
        sarif_file: results.sarif
```

#### Organisation des badges dans le README

Structurer les badges en sections pour lisibilité :

```markdown
## 📊 Qualité & Sécurité
[Go Report Card] [Codecov] [Scorecard] [License]

## 📦 Distribution
[npm] [Homebrew] [Scoop] [Go Reference]

## 📈 Métriques
[Stars] [Downloads] [Platform]
```

### Fichiers impactés

- `README.md` (root et/ou `multiai-go/README.md`)
- `.github/workflows/ci.yml` (job upload-coverage)
- `.github/workflows/scorecard.yml` (nouveau)

### Tests attendus

- [ ] Codecov rapporte la couverture après chaque push sur master
- [ ] Le badge Codecov est vert (> 60% couverture)
- [ ] Go Report Card est A+ (vérifié manuellement après publication)
- [ ] Scorecard rapporte un score > 5/10
- [ ] Tous les badges sont des liens cliquables
- [ ] Les badges s'affichent correctement en dark mode (GitHub)

### Résultat attendu

```
✅ Codecov configuré et badge vert
✅ Go Report Card A+ avec badge
✅ OpenSSF Scorecard actif
✅ CI avec upload coverage
✅ README badges organisés en sections
```

### Definition of Done

- [ ] `codecov.io` rapporte la couverture
- [ ] `goreportcard.com` donne A+
- [ ] `securityscorecards.dev` donne un score
- [ ] Tous les badges sont dans le README
- [ ] La CI inclut le job `upload-coverage`
- [ ] La CI inclut le job `scorecard`

### Risques

- **Codecov token** : doit être ajouté aux secrets GitHub → faire par l'UI, pas par commit
- **Scorecard permissions** : nécessite `id-token: write` qui peut être refusé par la politique de sécurité du compte → fallback : badge seulement, sans action CI
- **Go Report Card latence** : le badge peut mettre quelques minutes à se mettre à jour après un push → c'est normal
- **Badge `score-10/10` actuel** : supprimer ce badge trompeur (il n'est lié à aucune métrique réelle)

### Dépendances

- Aucune (tâche indépendante, peut être faite immédiatement)
- Codecov : nécessite que le repo soit public (déjà le cas) et que le propriétaire active le service

---

## Synthèse de l'Epic

### Dépendances entre stories

```
S8.1 ──→ S8.2 (registre avant validation CI)
S8.1 ──→ S8.3 (index.json avant commande search)
S8.3 ──→ S8.4 (package registry avant commande install)
S8.1 ──→ S8.5 (registre avant doc contributeur)
S8.1 ──→ S8.6 (optionnel, peut être indépendant)
S8.7 ──→ (indépendant)
```

### Effort total estimé

| Story | Effort | Dépendances |
|---|---|---|
| S8.1 | 4h | — |
| S8.2 | 3h | S8.1 |
| S8.3 | 5h | S8.1, S8.2 |
| S8.4 | 6h | S8.1, S8.3 |
| S8.5 | 3h | S8.1 |
| S8.6 | 2h | — |
| S8.7 | 1h30 | — |
| **Total** | **~24h30** | |

### Score KPI cible (v0.6.0)

| Métrique | Actuel | Cible v0.6.0 |
|---|---|---|
| Stars GitHub | 0 | 100+ |
| Profils communautaires | 0 | 5+ |
| Contributeurs externes | 0 | 3+ |
| Commandes profil (search/install) | 0 | 2 |
| GitHub Discussions actives | 0 | 20 |
| Badges README | 5 | 12 |
| Score visibilité (auto-évaluation) | 1/10 | 6/10 |

### Risques globaux

- **Adoption du registre** : sans contributeurs, le registre reste vide → seed avec 10 profils officiels dès le lancement
- **Maintien** : le registre et les commandes CLI ajoutent une surface de maintenance → documenter le processus de review et définir des SLAs
- **Sécurité** : un profil malveillant dans le registre pourrait exécuter des commandes arbitraires → la CI de validation (S8.2) bloque les `COMMAND` dangereux, et la vérification SHA256 (S8.4) garantit l'intégrité