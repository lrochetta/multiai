Voici les 6 stories detaillees pour Documentation Complete et Architecture v1.0.0.

---

# Stories — Documentation Complete + Architecture v1.0.0

## Story 1 — Documentation Architecture (ADR + Diagrammes)

**Titre :** Architecture Decision Records et diagrammes d'architecture multiai

**Priorite :** P0 (BLOCKER)

**Objectif :** Creer un dossier d'architecture formel comprenant les ADR (Architecture Decision Records) couvrant les decisions fondatrices du projet, ainsi que des diagrammes C4 model (System Context, Container, Component) pour outiller les contributeurs et la maintenance long-terme.

**Specs :**

1. Creer `docs/architecture/adr/` avec les ADR suivants :
   - **ADR-001** — Choix du store AES-256-GCM fichier comme backend par defaut (raisons : zero-dependency, portabilite, simplicite). Date, contexte, options ecartees (Keychain SDK, libsecret direct), consequences.
   - **ADR-002** — Architecture credential store avec `Store` interface et backends natifs OS (Windows Credential Manager via `advapi32.dll`, macOS Keychain via `security` CLI, Linux via `secret-tool`). Decision de ne PAS utiliser cgo pour macOS (preferer exec).
   - **ADR-003** — Routage multi-outils par profiles .env vs YAML vs projet `.multiai.yaml`. Pourquoi les trois coexistent et l'ordre de precedence.
   - **ADR-004** — Modele d'isolation des processus : whitelist env, sentinel pattern, zeroization memoire. Justification de l'approche liste-blanche vs conteneurisation.
   - **ADR-005** — Distribution multi-plateforme (GoReleaser, Cosign, SBOM, packages APT/AUR/Homebrew/Scoop). Pourquoi chaque canal est necessaire.
   - **ADR-006** — Architecture du registre communautaire (depot Git comme source de verite, index.json, cache 1h, SHA256 verification).
   - **ADR-007** — Migration PowerShell vers Go : detect, backup, copy, report. Pourquoi la migration est unidirectionnelle et conservatrice.

2. Chaque ADR suit le format standard :
   - Titre, statut (Proposed/Accepted/Deprecated/Superseded), date
   - Contexte et probleme
   - Options considerees
   - Decision retenue avec justification
   - Consequences (positives et negatives)
   - Liens vers les ADR lies (s'il y en a)

3. Creer `docs/architecture/diagrams/` avec des diagrammes PlantUML :
   - **System Context** — multiai et ses interactions (CLIs IA, GitHub API, registre communautaire, stores OS, utilisateur)
   - **Container** — decomposition du binaire Go : `cmd/multiai` + packages internes + stores + registre
   - **Component** — par package cle :
     - `internal/secret/` : Store interface, encryptedFileStore, winCredStore, darwinKeychainStore, linuxSecretStore, fallback
     - `internal/profile/` : Profile, YAML, Project, Merge, LoadAllProfiles
     - `internal/registry/` : Client, Cache, Install, Index
     - `internal/cli/` : Launcher, Display, Fallback, Hooks
     - `internal/migration/powershell/` : Detect, Migrate, Report
   - **Data Flow** — Sentinal pattern : config ecrit le sentinel → launch le resolve via Store.Get
   - **Sequence** — `multiai launch -p ds` : parse flags → load profiles → resolve sentinels → apply hooks → exec CLI

4. Generer les diagrammes en PNG via PlantUML et les referencer dans la documentation.

5. Ajouter une page `docs/architecture/index.md` sommaire liant ADR, diagrammes et description des decisions.

**Livrables :**
- `docs/architecture/index.md` — page d'accueil architecture
- `docs/architecture/adr/ADR-001.md` a `ADR-007.md` — 7 decisions architecturales
- `docs/architecture/diagrams/` — fichiers `.puml` sources + PNG generes
- Section dans `docs/.vitepress/config.ts` pour la sidebar Architecture

**Definition of Done (DoD) :**
- Chaque ADR existe, est date, complete, et relu par un pair technique
- Les diagrammes PlantUML compilent sans erreur
- Les PNG sont generes et visibles dans le site VitePress
- Les ADR sont lies entre eux par des references croisees
- Commit conventionnel : `docs: ADR-001 a ADR-007 + diagrammes C4`

---

## Story 2 — Documentation API (Godoc, Interfaces Publiques, Exemples)

**Titre :** Documentation API complete des packages internes multiai

**Priorite :** P0 (BLOCKER)

**Objectif :** Produire une documentation godoc exhaustive pour chaque package interne, couvrant les interfaces publiques (`Store`, `Profile`, `ProviderCatalog`...), les fonctions exportees, et des exemples executable pour les cas d'usage principaux. Publier sur `pkg.go.dev` via un module Go bien forme.

**Specs :**

1. **Godoc package-level** — Verifier et completer la documentation en-tete de chaque package :
   - `internal/secret/` — Threat model documente en tete (deja present), ajouter les exemples `ExampleStore_file`, `ExampleStore_wincred`, `ExampleServiceForProfile`
   - `internal/registry/` — Documentation de l'interface client, cache, installation
   - `internal/profile/` — Documentation des formats supportes, ordre de precedence, merge avec projet
   - `internal/cli/` — Documentation LaunchOptions, LaunchResult, hooks lifecycle
   - `internal/migration/powershell/` — Documentation du flux detect → migrate → report
   - `internal/catalog/` — Documentation du catalogue data-driven (providers.yaml)
   - `internal/fsutil/` — Documentation atomic writes guarantees
   - `internal/i18n/` — Documentation du systeme de traduction FR/EN
   - `internal/display/` — Documentation des helpers d'affichage colore

2. **Interfaces publiques** — Chaque interface doit avoir un exemple concret :
   - `Store` interface : exemple mock + exemple avec chaque backend
   - `Profile` struct : exemple chargement, recherche, lancement
   - `Provider` : exemple de resolution de fournisseur
   - `Index` / `ProfileEntry` : exemple de recherche et installation

3. **Exemples executable (`_test.go`)** — Creer des `ExampleXxx` functions pour :
   - `ExampleSecretStore_file` — stocke et recupere une cle
   - `ExampleProfile_LoadAll` — charge tous les profils
   - `ExampleRegistry_Search` — recherche un profil communautaire
   - `ExampleLauncher_Launch` — lance un profil (dry-run)
   - Ce sont des examples tests (compiles et executes par `go test`)

4. **README.go** — Ajouter un fichier `doc.go` dans chaque package avec la documentation package-level si elle n'existe pas deja :
   - `internal/secret/doc.go` — "Package secret provides secure credential storage..."
   - `internal/profile/doc.go` — "Package profile manages launch profiles..."
   - `internal/cli/doc.go` — "Package cli orchestrates CLI launch..."

5. **Publication pkg.go.dev** — Verifier que `go.dev` indexe correctement le module :
   - `go.mod` doit avoir un `module` valide
   - Les tags de version doivent etre presents (v0.5.0+)
   - Lancer `go list -m github.com/lrochetta/multiai@v0.5.0` pour verifier

6. **Surface API** — Generer `docs/api-surface.md` listant toutes les fonctions/structures exportees par package, avec leur signature et une phrase de description.

**Livrables :**
- Fichiers `doc.go` pour chaque package interne (7 packages)
- 8-10 `ExampleXxx` functions testees
- `docs/api-surface.md` — inventaire de la surface API
- Module visible sur `pkg.go.dev/github.com/lrochetta/multiai`
- Section API dans le site VitePress (`docs/reference/api.md`)

**DoD :**
- `go doc ./internal/secret/...` affiche une documentation complete avec exemples
- `go test -run Example -v ./internal/secret/` passe sans erreur
- La page `docs/reference/api.md` liste toutes les interfaces et leurs signatures
- pkg.go.dev indexe le module avec la doc complete
- Aucune fonction exportee sans commentaire godoc

---

## Story 3 — Mise a jour VitePress (Stores Natifs, Registry, Migration)

**Titre :** Nouvelles pages VitePress pour les features v0.6.0

**Priorite :** P0 (BLOCKER)

**Objectif :** Ajouter 5 nouvelles pages au site VitePress couvrant les fonctionnalites majeures de la v0.6.0 : stores natifs OS, migration PowerShell, registre communautaire, scripts d'installation, et options `--store`/`--timeout`.

**Specs :**

1. **Page stores natifs** — `docs/advanced/stores.md` :
   - Comparaison des 4 backends : `file` (AES-256-GCM), `wincred`, `keychain`, `secret-service`
   - Tableau des fonctionnalites par backend (chiffrement, isolement, multi-utilisateur, backup)
   - Guide choix : quand utiliser quel backend
   - Exemple `multiai config --store wincred`
   - Migration store natif (avec `--migrate-force`)
   - Securite : threat model detaille pour chaque backend

2. **Page migration PowerShell** — `docs/guide/migration.md` :
   - Quand migrer (v0.3.x PS vers v0.5.x+ Go)
   - Commande `multiai migrate`
   - Options `--from-ps`, `--dry-run`, `--json`
   - Backups automatiques
   - Compatibilite des profils (format .env identique)
   - Rollback manual si necessaire

3. **Page registre communautaire** — `docs/guide/community.md` :
   - Qu'est-ce que le registre communautaire
   - `multiai profile search <query>` — recherche
   - `multiai profile install <name>` — installation
   - Liste `--remote` vs local
   - Contribution : fork, YAML, PR
   - Mettre a jour la sidebar avec `{ text: 'Registre communautaire', link: '/guide/community' }`

4. **Page options avancees** — Mettre a jour `docs/reference/commands.md` :
   - Ajouter `--store` dans le tableau flags de `multiai config`
   - Ajouter `--timeout` dans le tableau flags de `multiai launch`
   - Ajouter les sous-commandes `profile`, `migrate`, `update`
   - Ajouter les codes de sortie manquants (5 = timeout, 6 = store error)

5. **Page scripts d'installation** — Mettre a jour `docs/guide/installation.md` :
   - Ajouter APT repository (Ubuntu/Debian)
   - Ajouter AUR (Arch Linux)
   - Ajouter scripts `install.sh` et `install.ps1`
   - Ajouter verification Cosign des binaires
   - Ajouter verification SHA256 checksums

6. **Navigation** — Mettre a jour `docs/.vitepress/config.ts` :
   - Ajouter les nouvelles pages dans la sidebar
   - Verifier que tous les liens internes sont valides
   - Ajouter les entrees manquantes dans le nav principal si necessaire

7. **Redirection** — Ajouter les redirections de l'ancien `troubleshooting.md` racine vers `guide/troubleshooting.md` pour les bookmarks existants.

**Livrables :**
- `docs/advanced/stores.md` — page backends de stockage
- `docs/guide/migration.md` — guide migration PowerShell
- `docs/guide/community.md` — registre communautaire
- Mise a jour de `docs/reference/commands.md` — nouveaux flags
- Mise a jour de `docs/guide/installation.md` — nouveaux canaux
- Mise a jour de `docs/.vitepress/config.ts` — sidebar

**DoD :**
- Site VitePress build sans erreur (`npx vitepress build docs/`)
- Tous les nouveaux liens sont valides (0 dead link)
- 20+ pages total sur le site
- Navigation cohérente : section stores, registry, et migration accessibles en 2 clics
- Les commandes `multiai config --store wincred`, `multiai migrate`, etc. sont documentees avec exemples

---

## Story 4 — Tutoriel Video "De Zero a Hero avec multiai" (Script + Storyboard)

**Titre :** Script et storyboard pour tutoriel video d'onboarding

**Priorite :** P1 (HIGH)

**Objectif :** Produire un script narratif et un storyboard detailles pour un tutoriel video de 10-15 minutes montrant le parcours complet : installation, configuration, premier lancement, profils avances, et contribution.

**Specs :**

1. **Structure du tutoriel** — 6 segments de 1,5 a 3 minutes :
   - **Segment 1** (1:30) — Qu'est-ce que multiai ? Problematique : jongler entre Claude Code, Codex CLI, OpenCode. Solution : routeur multi-IA.
   - **Segment 2** (2:00) — Installation : `npm install -g multiai`, verification, `multiai version`. Script install.sh sur Linux/macOS, install.ps1 sur Windows.
   - **Segment 3** (3:00) — Premier lancement : `multiai` menu interactif, `multiai launch -p ds`, `--dry-run`. Configuration des cles API avec `multiai config`.
   - **Segment 4** (2:30) — Gestion des profils : `multiai list`, `multiai list --json`, profils YAML personnalises, hooks before/after, `.multiai.yaml` projet.
   - **Segment 5** (2:00) — Securite : stores natifs (`--store keychain`), sentinel pattern, zeroization, auto-update, Cosign verification.
   - **Segment 6** (1:30) — Communaute : registre de profils, `multiai profile search`, contribution, GitHub Discussions.
   - **Conclusion** (0:30) — Roadmap v1.0.0, appels a contribution, liens.

2. **Script narratif** — Pour chaque segment :
   - Texte lu par le narrateur (FR, ton professionnel mais accessible)
   - Commandes tapees a l'ecran
   - Resultats attendus
   - Annotations et callouts visuels
   - Transitions entre segments

3. **Storyboard visuel** — Tableau avec colonnes :
   - Temps (timestamp debut-fin)
   - Action / visuel a l'ecran
   - Texte narrateur
   - Effets / transitions
   - Notes techniques (resolutions, polices, logos)

4. **Environnement de demo** — Specifier :
   - Terminal : Windows Terminal (Windows) / iTerm2 (macOS) avec theme sombre
   - Shell : PowerShell 7+ ou bash 5+
   - Resolution : 1920x1080, 60fps
   - Police : JetBrains Mono Nerd Font 14pt
   - Theme : One Dark Pro ou Dracula

5. **Supports complementaires** :
   - Fiche recapitulative PDF (commandes essentielles)
   - Lien vers la doc VitePress pour chaque segment
   - Fichier `.env` exemple pour les spectateurs
   - Mini-site `multiai.dev/tutorial` avec la video et les ressources

6. **Localisation** — Le script existe en deux versions :
   - FR : public francophone (prioritaire)
   - EN : public international (traduction)

**Livrables :**
- `docs/tutorial/script-fr.md` — script narratif FR complet (10-15 min)
- `docs/tutorial/storyboard-fr.md` — storyboard avec timing et visuels
- `docs/tutorial/script-en.md` — version anglaise
- `docs/tutorial/cheatsheet.md` — fiche recapitulative
- Plan de production : materiel necessaire (micro, enregistrement, montage)

**DoD :**
- Script lu a voix haute tient en 10-15 minutes (verification chronometree)
- Chaque commande donnee dans le script a son equivalent verifie dans la doc
- Storyboard executable par un monteur video sans contexte technique
- Cheatsheet tient sur une page A4

---

## Story 5 — CONTRIBUTING.md v2 (Guide Complet pour Contributeurs)

**Titre :** Rewrite complet du CONTRIBUTING.md pour code + profils

**Priorite :** P1 (HIGH)

**Objectif :** Transformer le CONTRIBUTING.md actuel (1 page, 6 sections) en un guide complet couvrant le code (Go), les profils (YAML/.env), la documentation (VitePress), la securite, et les bonnes pratiques CI/CD. Vise a abaisser la barriere d'entree pour les nouveaux contributeurs.

**Specs :**

1. **Architecture du document** — 8 sections claires avec table des matieres interactive :
   - **1. Introduction** — Philosophie du projet, code de conduite, attentes
   - **2. Quick Start** — Fork, clone, build, premiere PR en 5 min
   - **3. Environnement de developpement** — Go 1.24, outils (golangci-lint, gosec, govulncheck, gitleaks), IDE config
   - **4. Guide du code** — Structure des packages, conventions Go, godoc, tests, fuzzing
   - **5. Guide des profils** — Format YAML, .env, variables, validation, registry
   - **6. Guide de la documentation** — VitePress, conventions editoriales, diagrammes, i18n
   - **7. Workflow de contribution** — Branches, commits conventionnels, PR checklist, review
   - **8. Securite** — Signalement de vulnerabilite, threat model, secrets, supply chain

2. **Pour chaque section :**
   - Sous-sections claires avec des exemples concrets
   - Fichiers de reference (`go.mod`, `.golangci.yml`, `.goreleaser.yaml`) expliques
   - Commandes exactes a copier-coller
   - Checklist de verification

3. **Guide des profils (section 5) — Nouveau** :
   - Anatomie d'un profil YAML : chaque champ explique
   - Variables d'environnement : `$VAR`, `${VAR}`, valeurs par defaut
   - Validation locale : `bash tests/validate.sh`
   - Contribution au registre communautaire
   - Exemples de profils (minimal, complet, avec hooks)

4. **Guide de la documentation (section 6) — Nouveau** :
   - Installer et lancer VitePress en local
   - Conventions editoriales : ton, structure, exemples de code
   - Ajouter une page : frontmatter, sidebar, navigation
   - i18n : ajouter des messages FR/EN (comment contribuer des traductions)
   - Diagrammes : comment ajouter un PlantUML, generer le PNG

5. **Workflow (section 7) — Ameliore** :
   - Branches : `feat/`, `fix/`, `docs/`, `security/`
   - Commits conventionnels : templates et examples pour chaque type
   - PR template expliquee point par point
   - Processus de review : qui review quoi, comment reagir aux commentaires
   - Squash merge vs merge commit

6. **Securite (section 8) — Nouveau** :
   - Ouverture d'un security advisory vs issue publique
   - Processus de divulgation responsable
   - Bonnes pratiques : ne jamais commiter de cle, verifier les `.env`
   - Voir aussi `SECURITY.md` pour la politique complete

7. **Templates et automatisation** : Ajouter des fichiers supports :
   - `.github/ISSUE_TEMPLATE/config.yml` — redirection vers Discussions
   - `.github/workflows/pr-checks.yml` — checks automatiques sur chaque PR
   - `docs/contributing/` — pages de guide contribueur dans VitePress

**Livrables :**
- `CONTRIBUTING.md` complet (8 sections, 20+ pages rendues)
- `docs/contributing/` dossier avec 3 pages VitePress (quickstart, profiles, docs)
- Mise a jour sidebar VitePress

**DoD :**
- Un nouveau contributeur peut faire sa premiere PR en <30 minutes en suivant le guide
- Chaque commande et exemple du guide est teste et fonctionne
- `CONTRIBUTING.md` fait moins de 100 lignes de TOC + liens (le contenu detaille est dans VitePress)
- Les templates de PR, bug, feature sont coherents avec le guide
- Relu par au moins un contributeur externe (ou simulateur)

---

## Story 6 — CHANGELOG v1.0.0 (Automatisation depuis Commits Conventionnels)

**Titre :** Generation automatique du CHANGELOG depuis les commits conventionnels

**Priorite :** P1 (HIGH)

**Objectif :** Mettre en place un pipeline de generation automatique du CHANGELOG.md a chaque release, en parsant les commits conventionnels (`feat:`, `fix:`, `docs:`, `security:`, etc.) pour produire un changelog structure, categorise, et lisible. Le pipeline doit fonctionner en CI et en local.

**Specs :**

1. **Outil de generation** — Choisir et integrer un outil :
   - Option A : `git-chglog` (Go, configuration YAML, templates custom)
   - Option B : `semantic-release` (Node.js, plus lourd, auto-versioning)
   - Option C : script Go interne `internal/changelog/generate.go` (zero-dependency, maitrise totale)
   - **Decision recommandee** : Option A (`git-chglog`) car ecrit en Go, compatible CI, templates flexibles. Fallback : Option C si on veut eviter une dependance externe.

2. **Configuration `chglog`** :
   - Fichier `.chglog/config.yml` avec :
     - Styles par type de commit : `feat` → `### Features`, `fix` → `### Bug Fixes`, `security` → `### Security`, `docs` → `### Documentation`, `refactor` → `### Code Refactoring`, `test` → `### Tests`, `chore` → `### Build & CI`
     - Sections optionnelles : BREAKING CHANGE en rouge
     - Tri par type puis par scope
   - Template markdown : `chglog/CHANGELOG.tpl.md`

3. **Commits conventionnels strictes** :
   - Definir les types autorises dans `.github/settings.yml` ou un badge de statut
   - Ajouter un hook de pre-commit ou CI check validant le format du message
   - Formats acceptes : `type(scope): description` et `type: description`
   - Types valides : `feat`, `fix`, `security`, `docs`, `refactor`, `test`, `chore`, `perf`, `style`, `ci`

4. **Integration CI** — Script de generation automatique :
   - Nouvelle etape dans `release.yml` :
     ```yaml
     - name: Generate changelog
       run: |
         git-chglog -o CHANGELOG.md --next-tag v1.0.0
     ```
   - Le CHANGELOG.md genere est commite dans la release
   - L'etape verifie que le format est valide avant generation

5. **Script local** — `scripts/generate-changelog.sh` pour usage local :
   ```bash
   ./scripts/generate-changelog.sh [--next-tag v1.0.0]
   ```
   - Detecte la derniere release taggee
   - Genere le changelog depuis les commits entre les tags
   - Si `--next-tag` fourni, inclus les commits HEAD→tag

6. **Format du CHANGELOG** :
   ```
   # Changelog

   ## [v1.0.0] — 2026-07-31

   ### Breaking Changes
   - ...

   ### Security
   - ...

   ### Features
   - ...

   ### Bug Fixes
   - ...

   ### Documentation
   - ...

   ### Build & CI
   - ...

   ### Tests
   - ...
   ```

7. **Migration du CHANGELOG existant** :
   - Preserver les entrees historiques (v0.2.0 a v0.5.0) en-tete
   - A partir de v1.0.0, le contenu est genere automatiquement
   - Ajouter une note expliquant la transition dans le CHANGELOG

8. **Validation** — Ajouter un check CI qui verifie :
   - Le CHANGELOG est a jour avant merge sur master
   - Pas de `[Unreleased]` en double
   - Tous les commits HEAD→last tag sont references

**Livrables :**
- `.chglog/config.yml` — configuration git-chglog
- `chglog/CHANGELOG.tpl.md` — template markdown
- `scripts/generate-changelog.sh` — script local
- Mise a jour de `release.yml` — etape de generation auto
- Mise a jour de `ci.yml` — validation format commit
- `docs/contributing/commits.md` — guide de commits conventionnels
- Mise a jour `CONTRIBUTING.md` (section commits)

**DoD :**
- `./scripts/generate-changelog.sh` genere un CHANGELOG valide
- La CI verboten les commits non-conventionnels sur master
- Le changelog genere est lisible et categorise correctement
- Les entrees historiques (pre-v1.0.0) sont preservees
- La release v1.0.0 CI a un CHANGELOG genere automatiquement

---

## Resume des Priorites

| Story | Priorite | Effort estime | Depend de |
|-------|----------|---------------|-----------|
| 1. ADR + Diagrammes | P0 - BLOCKER | 3 jours | Aucune |
| 2. Documentation API | P0 - BLOCKER | 2 jours | Story 1 (slightly) |
| 3. Mise a jour VitePress | P0 - BLOCKER | 2 jours | Stories 1, 2 |
| 4. Tutoriel video | P1 - HIGH | 5 jours | Story 3 |
| 5. CONTRIBUTING.md v2 | P1 - HIGH | 2 jours | Aucune |
| 6. CHANGELOG automation | P1 - HIGH | 2 jours | Aucune |

## Metriques de succes v1.0.0

- 20+ pages VitePress avec architecture + API + stores + registry
- 7 ADR couvrant les decisions fondatrices
- 7 diagrammes C4 model (System Context, Container, Component)
- 100% des fonctions exportees documentees (godoc)
- pkg.go.dev indexe avec exemples
- Site VitePress build sans erreur, 0 dead link
- CHANGELOG genere automatiquement en CI
- CONTRIBUTING.md v2 operationnel (PR en <30 min)
- Script tutoriel pret a filmer (10-15 min)