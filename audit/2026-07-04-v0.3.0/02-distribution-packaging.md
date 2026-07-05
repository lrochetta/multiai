# Audit v0.3.0 — Distribution & packaging

Date 2026-07-04 · Auditeur Atlas (Strategist — go-to-market & distribution) · Score : 2.5/10 · Methode : audit BMAD+ parallele + contre-verification adversariale.

---

# Audit Distribution & Packaging — multiai (post-v0.3.0)

**Auditeur** : Atlas (Strategist, BMAD+) — mandat Nexus
**Date** : 2026-07-05
**Référence delta** : audit v0.2.1 du 2026-06-23 (`audit/07-audit-v0.2.1-synthese.md`)
**Note : 2.5/10**

---

## Resume

La chaîne de distribution de multiai est un **village Potemkine**. Le README racine (`README.md:71-77`) vend 5 méthodes d'installation (npm, Go, Homebrew, Scoop, script curl) et un badge « score 9.5/10 » (`README.md:9`) ; la vérification factuelle montre que **seul `npx multiai install` fonctionne**, et il installe la version **PowerShell legacy** (npm `multiai@0.3.0`, publiée 2026-06-24, vérifiée via `npm view multiai version` → `0.3.0`), pas le binaire Go présenté comme l'implémentation primaire.

Preuves structurantes :
- **Le repo GitHub `lrochetta/multiai` est PRIVÉ, sans aucune release ni tag distant** (`gh repo view` → `{"isPrivate":true,"latestRelease":null}` ; `git ls-remote --tags origin` → vide ; tags locaux v0.2.1/v0.2.6 jamais poussés). Toute URL `github.com/lrochetta/multiai/releases/download/...` pointe donc vers le néant.
- **Les workflows CI/Release ne tournent JAMAIS** : ils sont trackés sous `multiai-go/.github/workflows/` (`git ls-tree HEAD` racine : `.gitignore, CHANGELOG.md, CLAUDE.md, README.md, multiai-go, multiai-powershell, push-github.ps1` — pas de `.github/` racine). GitHub ne lit les workflows qu'à la racine. La « CI/CD complète » créditée comme force dans l'audit v0.2.1 est donc une illusion.
- **`https://rochetta.fr/multiai/install.sh` → HTTP 404** (testé). `install.ps1` documenté (`multiai-go/docs/guide/installation.md:37`) n'existe même pas dans `multiai-go/scripts/`.
- **`lrochetta/homebrew-tap` et `lrochetta/scoop-bucket` n'existent pas** (`git ls-remote` → « Repository not found » pour les deux).

## Forces

- **Le canal npm PowerShell fonctionne réellement** : `multiai@0.3.0` publié sur registry.npmjs.org (2026-06-24T11:31Z), 13 versions publiées depuis 0.2.x, `bin/multiai.js` propre (help, install, détection pwsh), garde-fou `prepublishOnly` anti-fuite de clés étendu aux 8 nouveaux fournisseurs (`multiai-powershell/package.json`, script scannant STEPFUN/SILICONFLOW/MIMO/LITELLM/REQUESTY/DASHSCOPE_API_KEY).
- **`.goreleaser.yml` bien conçu sur le papier** (`multiai-go/.goreleaser.yml`) : builds 5 plateformes CGO_ENABLED=0 + trimpath (l.8-27), checksums.txt (l.40-41), **signature Cosign keyless** (l.43-52), **SBOM Syft spdx-json** (l.54-62), publication brews/scoops automatisée (l.64-78). Répond conceptuellement au problème #6 v0.2.1.
- **`release.yml` structurellement moderne** (`multiai-go/.github/workflows/release.yml:8-33`) : permissions `id-token: write` pour Cosign OIDC, cosign-installer, sbom-action, goreleaser-action v6.
- **Couverture multi-gestionnaires ambitieuse** : manifestes npm, Homebrew, Scoop (avec `checkver`/`autoupdate` corrects, `multiai.json:18-30`), AUR, deb (+ postinst générant les complétions) — le squelette de la roadmap v0.5.0 Distribution existe.
- **`packaging/deb/build-deb.sh`** : script fonctionnel, injection version/arch par sed, dpkg-deb standard.

## Constats detailles

### 1. Pipeline de release mort-né (CRITIQUE)
- Workflows dans `multiai-go/.github/workflows/{ci,release}.yml` mais **aucun `.github/` à la racine du repo** (`git ls-tree HEAD` — 7 entrées, pas de `.github`). GitHub Actions ne les exécutera jamais. CI (lint/test 6×/gosec/govulncheck/benchmark, `ci.yml:13-99`) et Release n'ont jamais tourné sur GitHub.
- `release.yml:46` : `cd ../../multiai-powershell` — sort de `$GITHUB_WORKSPACE` ; le job npm-publish échouerait même si le workflow tournait, et publierait le package PS sur un tag Go.
- `release.yml:58-62` : le job « Update Homebrew Tap » est un **placeholder assumé** — `echo "Homebrew formula SHA256 auto-update ready"` ne fait rien.
- `.goreleaser.yml:64-78` : `brews`/`scoops` poussent vers `lrochetta/homebrew-tap` et `lrochetta/scoop-bucket` avec `secrets.GITHUB_TOKEN` (`release.yml:33`) — ce token ne peut pas écrire dans un autre repo (il faut un PAT), et ces deux repos **n'existent pas**.

### 2. Repo privé, zéro release, zéro tag distant (CRITIQUE)
- `gh repo view lrochetta/multiai` → `PRIVATE`, `latestRelease: null`. `git ls-remote --tags` → vide. `push-github.ps1:10` confirme l'intention : `gh repo create lrochetta/multiai --private`.
- Conséquence en cascade : toutes les URLs de téléchargement — `packaging/npm/install.js:61`, `scripts/install.sh:24`, `packaging/scoop/multiai.json:9,13`, `packaging/homebrew/multiai.rb:8`, `packaging/aur/PKGBUILD:10` — pointent vers `releases/download/v0.5.0/...` qui n'a jamais existé et serait de toute façon inaccessible publiquement.

### 3. Chaos de versions — 4 vérités contradictoires (CRITIQUE)

| Source | Version |
|---|---|
| `multiai-go/cmd/multiai/main.go:18` (`const version`) | **0.2.1** |
| `multiai-go/internal/menu/interactive.go:18` (bannière codée en dur) | **v0.2.1** |
| `CHANGELOG.md:7` + npm `multiai` publié + `multiai-powershell/package.json` | **0.3.0** |
| `packaging/npm/package.json:3`, `install.js:10`, `multiai.rb:8`, `multiai.json:3`, `PKGBUILD:3`, `deb/control:2`, `scripts/install.sh:6` | **0.5.0** |
| `multiai-go/ROADMAP.md:9` | v0.2.0 « en cours » |

Le packaging référence une version 0.5.0 que la roadmap elle-même (`ROADMAP.md:41-47`) classe comme phase future non cochée.

### 4. `-X main.version` inopérant : `version` est une `const` (HAUTE)
`main.go:18` déclare `const version = "0.2.1"` alors que `.goreleaser.yml:15`, `multiai.rb:22` et `PKGBUILD:15` injectent `-X main.version=...`. Le linker Go n'écrase que des **var** string ; sur une const l'injection est silencieusement ignorée. Tout binaire releasé afficherait « multiai 0.2.1 » à perpétuité. Doublé par la bannière codée en dur `interactive.go:18`.

### 5. Binaire Go non fonctionnel après installation (CRITIQUE)
- Aucun `go:embed` dans tout `multiai-go/` (grep vide). `main.go:21-37` cherche `configs/profiles` à côté de l'exécutable ou dans le cwd ; `profile.LoadDir` (`internal/profile/profile.go:41-44`) retourne une erreur si le dossier manque → `multiai list`/`launch` sortent en code 2.
- Or **aucun canal ne livre les profils** : `.goreleaser.yml:29-38` n'a pas de clause `files:` pour `configs/`, `install.js:83` copie uniquement le binaire, `install.sh:36` idem, `build-deb.sh` copie binaire + complétions seulement. Un utilisateur installant via n'importe quel canal obtient un outil cassé (0 profil).

### 6. Checksums placeholders — problème #4 v0.2.1 PERSISTANT (HAUTE)
- `packaging/aur/PKGBUILD:11` : `sha256sums=('REPLACE_WITH_ACTUAL_SHA256')` — la chaîne exacte pointée par l'audit v0.2.1.
- `packaging/homebrew/multiai.rb:9-10` : `PLACEHOLDER_ARM64_SHA256` / `PLACEHOLDER_AMD64_SHA256` — de plus, syntaxe invalide : deux sha256 arm/intel pour **une seule** URL de tarball source.
- `packaging/scoop/multiai.json:11,14` : `PLACEHOLDER_SHA256` ×2. Le commentaire « SHA256 will be filled by goreleaser » (l.2) est faux : goreleaser génère ses propres manifestes dans le bucket, il ne réécrit jamais ces fichiers trackés.

### 7. Packages source structurellement incompilables (HAUTE)
Le module Go vit dans le sous-dossier `multiai-go/` (`git ls-tree HEAD` racine) alors que `go.mod:1` déclare `module github.com/lrochetta/multiai` :
- `go install github.com/lrochetta/multiai@latest` (`README.md:73`, `multiai-go/README.md:130`, `docs/guide/installation.md:10`) échoue triplement : repo privé, mismatch chemin module/emplacement, pas de package main à la racine du module (main dans `./cmd/multiai`).
- `multiai.rb:8` télécharge le tarball du repo racine puis `go build ./cmd/multiai/` (l.22-23) → ce chemin n'existe pas à la racine du tarball. Formule incompilable.
- `PKGBUILD:10,15` : même défaut, aggravé par `install -Dm644 LICENSE` (l.21) alors qu'**aucun LICENSE n'est tracké à la racine** (le badge MIT de `README.md:5` pointe aussi vers ce fichier absent).
- `docs/guide/installation.md:89` : `go build -o multiai .` depuis la racine clonée → échec (pas de go.mod à la racine).

### 8. Mismatch de nommage des archives : dash vs underscore (HAUTE)
`.goreleaser.yml:31-35` produit `multiai_0.5.0_linux_amd64.tar.gz` (underscores). Mais `install.js:16-23,60` construit `multiai_0.5.0_linux-amd64.tar.gz` et `install.sh:22-23` `${PLATFORM}-${ARCH}` (dashes) → **404 garanti même si une release existait**. Seul le manifeste Scoop est cohérent avec goreleaser.

### 9. Trois identités npm concurrentes (HAUTE)
- `multiai` : publié (PowerShell, 0.3.0).
- `multiai-installer` : `packaging/npm/package.json:2` — **jamais publié** (`npm view` → E404) ; son `bin` mappe `multiai` sur `install.js` (l.5-7), donc chaque invocation post-install relancerait le téléchargement ; `install.js` ignore ses arguments, ne vérifie **aucun checksum**, ne teste pas le statut HTTP (une page 404 serait écrite comme archive) et ne suit qu'un seul niveau de redirection (l.29-40).
- `multiai-cli` : `docs/guide/installation.md:77` (`npm install -g multiai-cli`) — jamais publié non plus.

Le badge npm des deux README (`README.md:8`, `multiai-go/README.md:8`) affiche la version du package PowerShell sur la vitrine du produit Go.

### 10. Instructions Scoop/Homebrew incohérentes entre elles (MOYENNE)
- Scoop, trois variantes : `README.md:75` (`scoop install multiai` sans bucket — impossible), `multiai-go/README.md:140` (`bucket add lrochetta .../scoop-bucket`), `docs/guide/installation.md:62` (`bucket add multiai .../scoop-multiai`). Aucun de ces buckets n'existe.
- `multiai.json:13` déclare une URL windows-arm64 alors que `.goreleaser.yml:23-25` **ignore** windows/arm64 : cette archive ne sera jamais produite.

### 11. Onboarding : vitrine sans branchement (MOYENNE) — problème #13 v0.2.1 PERSISTANT
- `internal/onboarding/wizard.go` (IsFirstRun, RunWelcome, marqueur `~/.multiai/.first-run-done`) est du **code mort** : grep « onboarding » dans tout `multiai-go/` ne renvoie que sa propre déclaration de package — jamais importé, ni dans `main.go` ni ailleurs.
- Incohérences internes : le wizard annonce « 5 fournisseurs » (`wizard.go:40`) vs « 14+ » revendiqués ; `markFirstRunDone` écrit un marqueur (l.68-73) que `IsFirstRun` ne lit jamais (l.18-27).
- `internal/install/` et `internal/update/` sont des **répertoires vides** (0 fichier). Il n'existe **aucune commande `multiai install`** dans le Go (`main.go:126-182` : version/help/list/launch/config/completion). Le `npx multiai install` du README est servi par le package PowerShell.

### 12. Docs publiques VitePress : soignées mais mensongères (MOYENNE)
- `docs/index.md:44-50` : les 3 commandes du Quick Start sont mortes (curl 404, irm inexistant, go install cassé). « 17 profils inclus » (l.35) reflète le Go (17 fichiers dans `multiai-go/configs/profiles`, séries 00-57) mais pas la v0.3.0 vendue (38 profils côté PS, séries 60-83 absentes du Go).
- `docs/guide/installation.md:31-32` : « télécharge la dernière version… dans /usr/local/bin » — faux deux fois : `install.sh:6` épingle 0.5.0 (pas de résolution latest) et installe dans `~/.local/bin` (l.7).
- `docs/reference/commands.md` ne documente ni `models`, ni `search`, ni `compare` — cohérent avec le binaire Go mais contredisant `CHANGELOG.md` v0.3.0 qui les annonce (aucun `case "models"` dans tout le repo ; ces features sont un menu PS, `code-router.ps1:994`).
- Aucun workflow de déploiement docs (seulement ci.yml/release.yml) : le site n'est publié nulle part.
- Le `docs/` à la racine du projet est un répertoire **vide** non tracké.

### 13. Divergence Go/PS côté distribution (MOYENNE) — prolonge #12 v0.2.1
Le produit « v0.3.0 » distribué (npm) est le PowerShell (38 profils, OpenRouter Fusion, menu 4 entrées) ; le Go — présenté partout comme l'implémentation primaire — est resté à 0.2.1/17 profils/menu 3 entrées (`interactive.go:21-23` vs le menu 4 entrées montré dans `README.md:33-36`). Le CHANGELOG unique masque que deux produits divergents portent le même nom et le même numéro de version.

## Statut des problemes v0.2.1

| # v0.2.1 | Problème | Statut | Preuve |
|---|---|---|---|
| #4 | Checksums placeholders brew/scoop/AUR | **PERSISTE** (identique) | `PKGBUILD:11` (`REPLACE_WITH_ACTUAL_SHA256`), `multiai.rb:9-10`, `multiai.json:11,14` |
| #6 | Aucune signature Cosign | **PERSISTE de facto** | Config présente (`.goreleaser.yml:43-52`, `release.yml:22-25`) mais pipeline jamais exécutable (workflows hors racine, repo privé, 0 tag distant) → aucun artefact signé n'a jamais existé |
| #12 | Incohérence Go vs PS | **PERSISTE / AGGRAVÉ** (angle distribution) | npm livre PS 0.3.0/38 profils ; Go bloqué à 0.2.1/17 profils (`main.go:18`, `configs/profiles` = 17 fichiers) |
| #13 | Aucun wizard d'onboarding | **PERSISTE** (trompe-l'œil) | `wizard.go` écrit mais jamais importé (grep : 1 seul hit = sa déclaration) ; `internal/install/`, `internal/update/` vides |
| Force créditée « CI/CD complète » | — | **INVALIDÉE** | Workflows sous `multiai-go/.github/` jamais lus par GitHub (pas de `.github/` racine dans `git ls-tree HEAD`) |

## Recommandations priorisees

1. **[P0] Restructurer le repo publié** : soit faire de `multiai-go/` la racine du repo GitHub (résout d'un coup workflows + go install + tarballs brew/AUR + LICENSE), soit déplacer `.github/` à la racine et changer `go.mod` en `github.com/lrochetta/multiai/multiai-go`. Ajouter LICENSE à la racine. (~2 h)
2. **[P0] Décider public/privé** : tant que le repo est privé, retirer du README/docs toutes les méthodes d'installation basées sur GitHub Releases — chaque commande affichée doit fonctionner pour un inconnu. (1 h de vérité)
3. **[P0] Une seule source de version** : `var version = "dev"` (var, pas const) injectée par `-X`, bannière lisant cette var, script de release vérifiant l'égalité main.go/CHANGELOG/packaging. Supprimer les « 0.5.0 » anticipés. (2 h)
4. **[P0] Livrer les profils avec le binaire** : `go:embed configs/profiles` avec extraction vers `~/.multiai/profiles` au premier lancement (ce qui donne enfin un vrai `multiai install`/first-run et branche `wizard.go`). (1 jour)
5. **[P1] Premier release réel** : pousser un tag, laisser goreleaser produire archives+checksums+cosign+SBOM, vérifier que `install.sh`/`install.js` téléchargent (corriger dash→underscore, ajouter vérification checksum et statut HTTP dans install.js). Créer réellement `homebrew-tap`/`scoop-bucket` avec un PAT dédié, ou supprimer ces sections. (2 jours)
6. **[P1] Un seul nom npm** : trancher entre `multiai` (PS) et le futur installeur Go — publier le binaire Go sous `multiai` en optionalDependencies par plateforme (pattern esbuild) plutôt que 3 noms fantômes. (1 jour)
7. **[P1] Corriger release.yml** : supprimer le `cd ../../multiai-powershell`, implémenter ou supprimer le job homebrew-update placeholder. (2 h)
8. **[P2] Héberger ou supprimer** `rochetta.fr/multiai/install.sh|ps1` ; déployer le site VitePress (GitHub Pages workflow) et aligner installation.md sur les canaux réels. (1 jour)
9. **[P2] Retirer le badge « score 9.5/10 »** auto-attribué (`README.md:9`) — un badge codecov/goreportcard réel a plus de valeur marketing qu'un badge invérifiable. (10 min)
10. **[P2] Nettoyer la racine du working tree** : zips, `brainstorm laurent`, profils 60-62 en vrac à la racine — bruit qui finirait dans tout tarball source. (1 h)

---

**Verdict distribution : 2.5/10.** L'outillage (goreleaser, cosign, SBOM, multi-gestionnaires) est celui d'un projet mature, mais aucun maillon de la chaîne Go n'a jamais fonctionné de bout en bout, et le seul canal vivant distribue le legacy que le projet prétend remplacer. Pour « le meilleur routeur multi-IA du marché », la priorité absolue n'est pas une feature de plus : c'est un premier `git push --tags` qui produit une release installable et vérifiable par un inconnu.

---

## Findings contre-verifies

Aucun finding n'a été réfuté. Les verdicts PARTIAL correspondent à des faits confirmés mais à une sévérité initiale recalibrée (affichée ci-dessous).

| ID | Severite | Titre | Verdict | Note |
|---|---|---|---|---|
| 02-01 | high | Workflows GitHub Actions jamais exécutés (pas de `.github/` à la racine du repo) | PARTIAL | Faits confirmés ; sévérité recalibrée critical → high (pas de vecteur d'attaque, mais force « CI/CD complète » v0.2.1 invalidée ; aggravant : trigger `branches: [main]` vs branche `master`) |
| 02-02 | medium | Repo GitHub privé, zéro release, zéro tag distant : URLs de téléchargement mortes | PARTIAL | Faits confirmés ; sévérité recalibrée critical → medium (aucun utilisateur n'emprunte ces chemins aujourd'hui : `multiai-installer` non publié, canal npm réel indépendant des releases GitHub — release blocker latent, pas panne live) |
| 02-03 | medium | Chaos de versions : 0.2.1 / 0.3.0 / 0.5.0 / ROADMAP « v0.2.0 en cours » | PARTIAL | Faits confirmés (9 emplacements 0.5.0) ; sévérité recalibrée critical → medium (manifests 0.5.0 = templates pré-release ; impact live limité à la confusion Go 0.2.1 vs npm 0.3.0) |
| 02-04 | high | Binaire Go installé inutilisable : profils ni embarqués ni livrés par aucun canal | PARTIAL | Faits confirmés sur les 7 canaux ; sévérité recalibrée critical → high (release blocker garanti à 100 % pour v0.5.0, mais zéro utilisateur Go atteignable aujourd'hui ; le canal npm PS livre bien ses profils) |
| 02-05 | high | Checksums placeholders AUR/Homebrew/Scoop — problème #4 v0.2.1 inchangé | CONFIRMED | Vérifié ; aggravant : pas de section `aurs:` dans goreleaser, syntaxe `arm:/intel:` invalide en Formula, `.SRCINFO` divergent (`SKIP`) |
| 02-06 | high | Injection `-X main.version` inopérante : `version` est une `const` | CONFIRMED | Vérifié ; touche aussi Makefile:4, go-build.ps1:60 et User-Agent codé en dur (`openrouter/client.go:38`) |
| 02-07 | high | Packages source incompilables : module Go en sous-dossier vs go.mod racine | CONFIRMED | Vérifié point par point (go install, Homebrew, AUR, build manuel tous cassés ; LICENSE absent) |
| 02-08 | high | Mismatch nommage archives : goreleaser underscore vs installeurs dash — 404 garanti | CONFIRMED | Vérifié ; aggravant : `install.js` ne teste pas le statusCode et écrirait la page 404 comme archive |
| 02-09 | high | Trois identités npm : `multiai` (PS publié), `multiai-installer` et `multiai-cli` (inexistants) | CONFIRMED | Vérifié sur le registre npm (2026-07-05) ; badge npm PS affiché sur la vitrine Go |
| 02-10 | high | release.yml cassé : `cd` hors workspace + job Homebrew placeholder + GITHUB_TOKEN cross-repo | CONFIRMED | Vérifié ; aggravant : le workflow ne se déclenchera de toute façon jamais (hors racine) |
| 02-11 | high | URLs publiques mortes : rochetta.fr/multiai/install.sh → 404, install.ps1 inexistant | CONFIRMED | Vérifié en live 2026-07-05 (404, et 403 sur User-Agent curl) ; 4 fichiers de doc affectés |
| 02-12 | medium | Onboarding fantôme : wizard.go jamais importé, internal/install et internal/update vides | non contre-vérifié | — |
| 02-13 | medium | npm install.js fragile : pas de checksum, statut HTTP ignoré, bin mappé sur l'installeur | non contre-vérifié | — |
| 02-14 | medium | install.sh contredit la doc : version épinglée 0.5.0 et ~/.local/bin vs /usr/local/bin | non contre-vérifié | — |
| 02-15 | medium | Manifeste Scoop déclare une archive windows-arm64 jamais produite par goreleaser | non contre-vérifié | — |
| 02-16 | medium | CHANGELOG v0.3.0 annonce des commandes Go inexistantes (models/search/compare) et 20 profils absents du Go | non contre-vérifié | — |
| 02-17 | low | Badge auto-attribué « score 9.5/10 » sur le README public | non contre-vérifié | — |
| 02-18 | low | ROADMAP obsolète et contradictoire avec le packaging | non contre-vérifié | — |
