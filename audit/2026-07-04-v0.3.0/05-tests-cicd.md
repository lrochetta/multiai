# Audit v0.3.0 — Tests & CI/CD

Date 2026-07-04 · Auditeur Sentinel (Quality — tests & CI/CD) · Score : 3/10 · Méthode : audit BMAD+ parallèle + contre-vérification adversariale.

---

# Audit Tests & CI/CD — multiai (état au 2026-07-05)

**Auditeur** : Sentinel (BMAD+ Quality)
**Périmètre** : multiai-go/ (Go, primaire) + multiai-powershell/ (legacy npm)
**Version auditée** : ambiguë — CHANGELOG.md:7 et package.json:3 annoncent 0.3.0, mais le binaire Go est resté à `0.2.1` (multiai-go/cmd/multiai/main.go:18)
**Outillage** : go1.26.4 windows/amd64 — `go vet`, `go test -cover`, `go build`, `gofmt -l` exécutés réellement

---

## Résumé

Le projet affiche une façade de qualité (« 45+ tests », « CI/CD : lint → test (6 OS × Go) → security → build → benchmark », badge « score 9.5/10 ») qui ne résiste pas à la vérification. Le point le plus grave : **l'intégralité de la CI/CD n'a jamais tourné et ne peut pas tourner** — les workflows sont dans `multiai-go/.github/workflows/` alors que la racine du repo git est `D:/travail/DEV/multiai/` (GitHub Actions ne lit que `.github/workflows/` à la racine), et ils se déclenchent sur `branches: [main]` (ci.yml:5) alors que la seule branche est `master`. Dependabot est inactif pour la même raison d'emplacement. Le pipeline de release (goreleaser + Cosign + SBOM), correctif clé de l'audit v0.2.1, n'a jamais été déclenché (aucun tag v0.3.0, workflow mort) et est cassé par construction. Les tests existants sont réels et passent tous, mais couvrent essentiellement les parseurs ; ~60 % du code de production (lancement, hooks, config, menu) est à 0 % de couverture, y compris les correctifs des vulnérabilités critiques v0.2.1 qui n'ont aucun test de régression.

**Note : 3/10.** Base de tests authentique mais étroite, chaîne de vérification fantôme, claims de qualité contredits par les mesures.

---

## Forces

- **Les tests existants passent tous** : `go test ./... -cover` → ok sur les 5 packages testés, 0 échec (exécuté le 2026-07-05).
- **`go vet ./...` : 0 warning** — le claim README.md:201 est exact.
- **pkg/dotenv réellement bien testé** : 93.9 % mesuré, conforme au claim README.md:202 ; 11 tests couvrant export, quotes, commentaires, lignes malformées, `=` dans les valeurs (pkg/dotenv/dotenv_test.go:8-150).
- **Bons réflexes de test** : `t.TempDir()` systématique, un vrai table-driven avec sous-tests (tests/integration_test.go:49-74 ; pkg/dotenv/dotenv_test.go:121-150).
- **Le fix #3 v0.2.1 (AllowedCommands) est corrigé ET testé** : slice immuable + accesseur (internal/cli/launcher.go:18-28), test positif/négatif (tests/integration_test.go:77-89).
- **Des tests Pester existent** pour le legacy PowerShell : 21 blocs `It` sur 3 fonctions pures (multiai-powershell/tests/unit/RouterFunctions.Tests.ps1:8-142) — le claim « 21 tests » de CHANGELOG.md:131 est exact.
- **La configuration goreleaser est sérieuse sur le papier** : Cosign keyless (multiai-go/.goreleaser.yml:43-52), SBOM Syft (:54-62), checksums (:40-41), `go test -race` en hook (:6).
- **dependabot.yml bien rédigé** (gomod + actions + npm, multiai-go/.github/dependabot.yml:1-20) — mais inactif, voir constats.

---

## Constats détaillés

### 1. CI/CD : 100 % inopérante — aucun job n'a jamais tourné (CRITIQUE)

- Racine du repo git = `D:/travail/DEV/multiai/` (`.git` à la racine, confirmé) ; il n'y a **aucun dossier `.github/` à la racine**. Les workflows sont dans `multiai-go/.github/workflows/ci.yml` et `release.yml`. GitHub Actions ne lit les workflows que dans `<racine>/.github/workflows/` → **ces fichiers sont ignorés**.
- Deuxième verrou : `on.push.branches: [main]` (multiai-go/.github/workflows/ci.yml:4-7) alors que la seule branche est `master` (git branch -a : `master`, `origin/master`). Même déplacés à la racine, les workflows ne se déclencheraient pas.
- Conséquence : les claims README.md:200 (« CI/CD : lint → test (6 OS × Go) → security → build → benchmark ») et multiai-go/README.md:247 décrivent une chaîne qui n'a **jamais exécuté un seul job**. gosec (ci.yml:27-30) et govulncheck (ci.yml:64-65) sont configurés mais n'ont jamais scanné le code.

### 2. Même relocalisés, les workflows échoueraient (HAUTE)

- Aucun job ne définit `working-directory` : `go vet ./...` (ci.yml:22), `go test` (ci.yml:47), `go build` (ci.yml:94) tourneraient à la racine du repo **où il n'y a pas de go.mod** (go.mod est dans multiai-go/) → échec immédiat de tous les jobs.
- `.golangci.yml` est au format v1 (multiai-go/.golangci.yml:1-22, pas de clé `version: "2"`, linters `gofmt`/`gosimple` obsolètes en v2) alors que `golangci-lint-action@v7` (ci.yml:24) installe golangci-lint v2 → erreur de configuration fatale.
- `gofmt -l` échoue sur 3 fichiers (vérifié) : internal/config/wizard.go, internal/openrouter/client.go, internal/profile/yaml.go → le linter gofmt casserait le job lint de toute façon.
- Bug d'expression : `matrix.os == 'windows'` (ci.yml:94, :99) n'est jamais vrai (valeurs `windows-latest`) → l'artefact Windows ne serait jamais suffixé `.exe`.

### 3. Pipeline de release : jamais déclenché et cassé par construction (HAUTE)

- Tags existants : v0.2.1, v0.2.6 — **pas de tag v0.3.0** alors que CHANGELOG.md:7 date la release au 2026-06-24. La v0.3.0 npm (package.json:3) a donc été publiée manuellement, sans Cosign, sans SBOM, sans checksums.
- release.yml, job goreleaser (multiai-go/.github/workflows/release.yml:14-33) : s'exécuterait à la racine, sans `.goreleaser.yml` ni go.mod visibles → échec.
- Job npm-publish : `cd ../../multiai-powershell` (release.yml:46) sort **au-dessus** du workspace de checkout → chemin inexistant. Le bon chemin depuis la racine serait `multiai-powershell/`.
- Job homebrew-update : placeholder assumé — `echo "Homebrew formula SHA256 auto-update ready"` (release.yml:59-62).
- Checksums packaging toujours factices : `PLACEHOLDER_ARM64_SHA256` (packaging/homebrew/multiai.rb:9-10), `PLACEHOLDER_SHA256` (packaging/scoop/multiai.json:10,14), `REPLACE_WITH_ACTUAL_SHA256` (packaging/aur/PKGBUILD:11), `SKIP` (packaging/aur/.SRCINFO:11). Délégués à un goreleaser qui n'a jamais tourné.

### 4. Dependabot inactif (HAUTE)

- multiai-go/.github/dependabot.yml:1-20 : bien écrit (gomod `/multiai-go`, actions, npm) mais **doit être à `<racine>/.github/dependabot.yml`** pour être lu par GitHub → inactif. Le correctif #14 v0.2.1 est cosmétique.

### 5. Actions épinglées par tag, pas par SHA — #14 v0.2.1 persiste (MOYENNE)

- ci.yml:17 (`actions/checkout@v4`), :18 (`setup-go@v5`), :24 (`golangci-lint-action@v7`), :28 (`securego/gosec@v2.22.3`), :49 (`codecov-action@v5`), :76,:96 (`upload-artifact@v4`) ; release.yml:18,:21,:23 (`cosign-installer@v3`), :25 (`sbom-action@v0`), :27 (`goreleaser-action@v6`), :41 (`setup-node@v4`) — **aucun SHA**. CHANGELOG.md:79 (« Pins + benchmark dans CI ») est trompeur : ce sont des tags mutables.

### 6. Inventaire réel des tests : le claim « 45+ tests » est faux (HAUTE)

Comptage exhaustif des 8 fichiers `*_test.go` (grep `^func (Test|Benchmark)`) :

| Fichier | Test* | Benchmark* |
|---|---|---|
| pkg/dotenv/dotenv_test.go | 11 | 0 |
| internal/env/env_test.go | 3 | 0 |
| internal/profile/profile_test.go | 5 | 0 |
| internal/secret/secret_test.go | 4 | 0 |
| tests/integration_test.go | 5 | 0 |
| tests/validation_test.go | 3 | 0 |
| tests/config_test.go | 1 | 0 |
| tests/benchmark_test.go | 0 | 2 |
| **Total** | **32** | **2** |

34 fonctions au total (+18 sous-tests `t.Run`). Le claim « 45+ tests » (README.md:199, multiai-go/README.md:246) n'est atteignable qu'en comptant les lignes de tables — au sens standard Go, c'est 32.

### 7. Couverture réelle vs annoncée (HAUTE)

`go test ./... -cover` (2026-07-05) :

| Package | Mesuré | Annoncé | Verdict |
|---|---|---|---|
| pkg/dotenv | **93.9 %** | 93.9 % (README.md:202) | ✅ exact |
| internal/env | **86.2 %** | 96.0 % (README.md:202) | ❌ périmé (−9.8 pts) |
| internal/secret | **64.7 %** | 61.2 % (multiai-go/README.md:249) | ~ périmé |
| internal/profile | **27.2 %** | non annoncé | — |
| cmd/multiai (323 loc) | **0 %** | — | ❌ |
| internal/cli (565 loc) | **0 %** | — | ❌ |
| internal/config (298 loc) | **0 %** | — | ❌ |
| internal/menu (141 loc) | **0 %** | — | ❌ |
| internal/openrouter (96 loc) | **0 %** | — | ❌ |
| internal/onboarding (83 loc) | **0 %** | — | ❌ |
| internal/logging (72 loc) | **0 %** | — | ❌ |

≈ 60 % du code de production (~1 580 loc sur ~2 620) est dans des packages à **zéro test**. Risques concrets : `internal/cli` contient le launcher (whitelist, validation secrets, propagation exit code, forwarding signaux — launcher.go:55-171) et les hooks shell (hooks.go:39-137), c'est-à-dire toute la surface d'attaque et le chemin critique du produit. `internal/config` contient `updateEnvFile` (écriture des clés API, wizard.go:257-297). Nuance : profile monte un peu au-dessus de 27.2 % si l'on compte les tests inter-packages de `tests/` (non attribués sans `-coverpkg`).

Note : `go test -race` est impossible localement (`-race requires cgo`, vérifié) ; seul un CI l'exécuterait — et il est mort. Le mutex anti-TOCTOU (#2 v0.2.1) n'a donc jamais été validé par le race detector nulle part.

### 8. Les fixes critiques v0.2.1 n'ont aucun test de régression (HAUTE)

- **Injection hooks (#1, le CRITIQUE de v0.2.1)** : le fix `escapeShellArg` (internal/cli/hooks.go:14-37) a **zéro test** (package cli : 0 %). Pire, l'ordre des opérations est suspect : le commentaire hooks.go:56 (« Expand env AFTER escaping, so injected env vars can't add shell metacharacters ») affirme l'inverse de l'effet réel — `os.ExpandEnv` (hooks.go:57) injecte les valeurs d'environnement **après** l'échappement, donc leurs métacaractères ne sont jamais échappés avant `bash -c`/`cmd /c` (hooks.go:62-77). Un test de régression d'injection (payload `; rm -rf` dans une valeur expansée) aurait immédiatement exposé ce doute. Sans test, impossible d'affirmer que le CVSS 8.3 annoncé « → 0 » (CHANGELOG.md:41) est réel.
- **updateEnvFile atomique (#9/#16)** : corrigé (temp + rename, wizard.go:278-287) mais non exporté, dans un package à 0 %, et le seul test au nom prometteur est une façade (voir constat 9).
- **Exit code (#7)** : propagé (main.go:153-155) mais le chemin d'erreur de `runLaunch` retourne `nil` (main.go:276-279) → `multiai launch -p inexistant` sort avec le code 0, contredisant la table « Codes de sortie 0-4 » (multiai-go/README.md:208-216). Aucun test.
- **Signaux (#8)** : forwarding ajouté (launcher.go:117-142). Aucun test.
- **TOCTOU (#2)** : mutex ajouté (secret.go:39,110-147). Aucun test de concurrence dans secret_test.go (:79-118 : happy path séquentiel Set/Get/List/Delete uniquement — pas de goroutines, pas de clé corrompue, pas de mauvaise clé de déchiffrement).

### 9. Test de façade : TestConfig_UpdateEnvFile (MOYENNE)

tests/config_test.go:12-31 : malgré son nom, ce test ne touche jamais `updateEnvFile` — il écrit un .env, appelle `profile.LoadDir` et vérifie qu'un placeholder est un placeholder. L'écriture atomique, le remplacement `__MULTIAI_CREDSTORE__` (wizard.go:269) et le chemin d'erreur « variable non trouvée » (wizard.go:274-276) ne sont testés nulle part.

### 10. Claims v0.3.0 fantômes côté Go (HAUTE — claims vs code)

- `multiai models`, `search`, `compare` annoncés shipped (CHANGELOG.md:16-18 ; README.md:89-90, :147-152) : **absents** du switch de main.go:126-182 (sous-commandes réelles : version, help, list, launch, config, completion). `internal/openrouter/client.go` est du **code mort** — jamais importé hors de lui-même (grep `openrouter.` : seule autre occurrence = une URL dans wizard.go:89).
- « Régions EU/US », « fallback chains », « cost logging » (CHANGELOG.md:12-14, README.md:160-163) : **introuvables dans le code Go** (grep region/fallback/cost : uniquement le fallback du credential store). Présents seulement partiellement dans le legacy PowerShell (41 occurrences dans code-router.ps1).
- Le binaire s'annonce `0.2.1` (main.go:18) et le menu affiche « v0.2.1 » en dur (internal/menu/interactive.go:18) alors que npm publie 0.3.0 (package.json:3). Le menu Go a 3 options (main.go:192-211, option 3 = stub « BMAD+ n'est pas encore integre ») quand le README racine en montre 4 dont OpenRouter (README.md:33-36). Aucun test ne verrouille la cohérence version/CHANGELOG — c'est exactement le rôle d'un test de release.
- Badge « score 9.5/10 » (README.md:9) et « 10/10 » (multiai-go/README.md:9) : auto-attribués, sans aucune mesure automatisée derrière (CI morte).

### 11. Legacy PowerShell : tests jamais exécutés automatiquement (MOYENNE)

- Les 21 tests Pester (RouterFunctions.Tests.ps1) couvrent 3 fonctions pures sur ~30 fonctions du code-router.ps1 (1 165 lignes) : menus, lancement, config, isolation env, régions/fallback/cost — non testés.
- **Aucun mécanisme ne les exécute** : pas de script `"test"` dans package.json:40-42 (seul `prepublishOnly` existe), aucun job Pester dans ci.yml. Le paquet npm distribué (celui que les utilisateurs installent réellement via `npx multiai install`) part en production sans qu'un seul test ne tourne.

### 12. Divers (BASSE)

- Makefile:2 : `VERSION = 0.2.0-dev` périmé (troisième version différente dans le repo).
- Benchmarks minimalistes : 2 fonctions (benchmark_test.go:11-33), aucune comparaison de référence ; `BenchmarkFindByShortcut` cherche `"ocdefault"` qui n'est pas dans la liste construite (:25-31) — il benchmarke le chemin d'échec.
- Job benchmark CI : upload d'artefact sans garde de régression (ci.yml:67-79).
- codecov-action@v5 (ci.yml:49) sans `CODECOV_TOKEN` : échouerait sur repo privé.

---

## Statut des problèmes v0.2.1 (dimension Tests & CI/CD + fixes vérifiés en code)

| # v0.2.1 | Problème | Statut | Preuve |
|---|---|---|---|
| #14 | Actions non pinnées SHA + dependabot | **PERSISTE** | ci.yml:17-28,49,76,96 et release.yml:18-41 : tags mutables ; dependabot.yml mal placé (multiai-go/.github/) → inactif |
| #6 | Pas de signature Cosign | **PERSISTE de facto** | Config présente (.goreleaser.yml:43-52, release.yml:22-23) mais jamais exécutée : pas de tag v0.3.0, workflow ignoré par GitHub |
| #4 | Checksums placeholders brew/scoop/AUR | **PERSISTE** | homebrew/multiai.rb:9-10, scoop/multiai.json:10,14, aur/PKGBUILD:11 — toujours des placeholders ; goreleaser censé les remplir n'a jamais tourné |
| #2 | Race TOCTOU encryptedFileStore | **CORRIGÉ** (sans test) | secret.go:39 (sync.Mutex), :110-147 — aucun test de concurrence, -race jamais exécuté |
| #3 | AllowedCommands map mutable | **CORRIGÉ + testé** | launcher.go:18-28 ; tests/integration_test.go:77-89 |
| #7 | Exit code non propagé | **PARTIEL** (sans test) | main.go:153-155 propage ; mais erreur de lancement → `nil` → exit 0 (main.go:276-279), contredit multiai-go/README.md:208-216 |
| #8 | Pas de signal handling | **CORRIGÉ** (sans test) | launcher.go:117-142 (forwarding SIGINT/SIGTERM) |
| #9 | updateEnvFile non atomique | **CORRIGÉ** (sans test) | wizard.go:278-287 (temp + rename) ; test homonyme = façade (config_test.go:12-31) |
| #1 | Injection shell hooks | **PARTIEL, non prouvé** | hooks.go:14-37 (escapeShellArg) mais ExpandEnv après échappement (hooks.go:55-57) laisse les valeurs env non échappées ; 0 test |
| #13 | Pas de wizard onboarding | **PERSISTE (déguisé)** | internal/onboarding/wizard.go existe mais n'est appelé nulle part (grep `onboarding.` : 0 usage externe) — code mort |
| Force « CI/CD complète » | — | **RÉGRESSION (jamais vraie)** | Workflows hors racine + branche main vs master : aucune exécution possible |
| Force « 45+ tests » | — | **INFIRMÉE** | 32 Test* + 2 Benchmark* comptés |

---

## Recommandations priorisées

### Immédiat (< 1 jour)
1. **Déplacer `.github/` à la racine du repo** (workflows + dependabot) et corriger `branches: [main]` → `[master]` (ou renommer la branche). Ajouter `defaults.run.working-directory: multiai-go` + `working-directory` sur les actions setup-go/goreleaser. C'est le déblocage de TOUT le reste.
2. **Corriger release.yml** : `cd multiai-powershell` (pas `../../`), goreleaser avec `workdir: multiai-go`, supprimer le job homebrew-update placeholder (goreleaser `brews:` le fait déjà).
3. **Mettre à jour ou supprimer les claims** : « 45+ tests », « env 96.0% », badge 9.5/10, « CI/CD complète », commandes `models/search/compare` — tant que ce n'est pas vrai, chaque ligne du README est une dette de crédibilité pour « le meilleur routeur du marché ».
4. **Aligner la version** : main.go:18 et interactive.go:18 → injecter `-X main.version` partout, un seul SSOT.

### Haute priorité (semaine)
5. **Tests de régression sécurité** : injection hooks (payloads métacaractères dans templates ET valeurs env), exit code sur erreur de lancement, updateEnvFile (atomicité + variable absente), concurrence sur encryptedFileStore (goroutines + `-race` en CI).
6. **Migrer .golangci.yml au format v2** (`version: "2"`), passer `gofmt -w` sur les 3 fichiers non formatés.
7. **Pinner les actions par SHA** (dependabot les tiendra à jour une fois actif).
8. **Job CI Pester** (windows-latest, `Invoke-Pester`) + script `"test"` dans package.json — le paquet npm est le produit distribué, il part aujourd'hui sans aucune vérification.
9. **Décider du sort du code mort** : wirer `internal/openrouter` + `internal/onboarding` dans main.go, ou les retirer et corriger le CHANGELOG v0.3.0.

### Moyen terme (mois)
10. Couverture cible : cli ≥ 70 % (launcher testable en injectant un exécuteur), config ≥ 60 %, profile ≥ 60 % (yaml.go/project.go à 0 % en couverture intra-package). Seuil de couverture bloquant en CI (codecov `fail_ci_if_error` + target).
11. Premier tag de release via le pipeline réparé (v0.3.1) pour produire enfin binaires signés Cosign + SBOM + checksums réels — condition pour retirer les placeholders packaging.
12. Test E2E minimal : `multiai list --json`, `multiai launch -p X --dry-run --json` sur binaire compilé, en matrice 3 OS.

---

## Justification de la note : 3/10

| Critère | Évaluation |
|---|---|
| Tests existants | Réels, passent tous, dotenv exemplaire (+) |
| Couverture | 0 % sur ~60 % du code, dont tout le chemin critique (−−) |
| Régression sur fixes critiques | Inexistante (−−) |
| CI | Jamais exécutée, doublement inopérante (−−−) |
| Release engineering | Pipeline jamais déclenché, cassé par construction, artefacts non signés (−−) |
| Véracité des claims | « 45+ tests », couvertures, badge, features v0.3.0 : infirmés (−−) |

Un 10/10 = CI verte publique, couverture mesurée et publiée, releases signées reproductibles, tests de régression sur chaque CVE interne. Le projet en est loin ; la base de tests honnête sur les parseurs évite le 2.

---

## Findings contre-vérifiés

| ID | Sévérité | Titre | Verdict | Note |
|---|---|---|---|---|
| 05-01 | high (corrigée depuis critical) | CI/CD entièrement inopérante : workflows hors racine du repo + branche main inexistante | PARTIAL | Cœur confirmé par l'API GitHub (aucun run CI/Release jamais enregistré) ; erreur factuelle : release.yml se déclenche sur les tags `v*`, pas sur `main` — il est inopérant uniquement par son emplacement. Sévérité recalibrée high (pas d'impact runtime direct). |
| 05-03 | high | 7 packages sur 12 à 0 % de couverture, dont tout le chemin critique de lancement | CONFIRMED | LOC et pourcentages vérifiés au fichier près ; 60,3 % du code de production sans test, incluant whitelist, secrets, hooks, exit codes. |
| 05-04 | high | Fix injection shell des hooks (#1 v0.2.1) sans test de régression, ordre escape→expand douteux | CONFIRMED | Exploit résiduel reproductible via valeur d'env contenant des métacaractères (ExpandEnv après échappement) ; commentaire in-code littéralement inversé ; 0 test. |
| 05-05 | high | Pipeline de release jamais déclenché et cassé par construction | CONFIRMED | Aucun tag v0.3.0, npm 0.3.0 publié hors pipeline ; workflow de toute façon hors racine ; chemins goreleaser/npm cassés ; placeholders packaging référencent même v0.5.0. |
| 05-06 | high | Features v0.3.0 annoncées mais absentes du binaire Go ; version bloquée à 0.2.1 | CONFIRMED | models/search/compare n'existent nulle part (ni Go ni PowerShell) ; openrouter et onboarding = code mort (0 import) ; nuance : npm ships le PowerShell, qui a bien regions/fallback/cost. |
| 05-07 | medium | Actions GitHub épinglées par tags mutables, pas par SHA | non contre-vérifié | Risque supply-chain type tj-actions ; #14 v0.2.1 persiste. |
| 05-08 | medium | Dependabot inactif : fichier placé hors de la racine du repo | non contre-vérifié | Fichier correct mais jamais lu par GitHub. |
| 05-09 | medium | Lint job condamné : .golangci.yml v1 avec golangci-lint-action@v7 (v2) + 3 fichiers non gofmt | non contre-vérifié | gofmt -l vérifié localement le 2026-07-05. |
| 05-10 | medium | TestConfig_UpdateEnvFile est un test de façade | non contre-vérifié | updateEnvFile jamais appelé par le test homonyme. |
| 05-11 | medium | Tests Pester jamais exécutés : pas de script npm test, pas de job CI PowerShell | non contre-vérifié | Le paquet npm distribué part en prod sans aucun test. |
| 05-12 | medium | Exit code 0 sur erreur de lancement : table « Codes de sortie 0-4 » ni implémentée ni testée | non contre-vérifié | runLaunch retourne nil sur erreur (main.go:276-279). |
| 05-02 | low (corrigée depuis low, comptage requalifié) | Claim « 45+ tests » : 32 Test* + 2 Benchmark* comptés | PARTIAL | Le « 45+ » devient atteignable en comptant les 18 sous-tests t.Run (50 « === RUN » mesurés) → titre requalifié ; reste vrai : couverture env annoncée 96,0 % vs 86,2 % mesurée (doc périmée). |
| 05-13 | low | Fixes concurrence (#2) validés nulle part : pas de -race possible localement, pas de test de concurrence | non contre-vérifié | Mutex présent mais jamais passé au race detector. |
| 05-14 | low | Bugs mineurs CI/build : expression windows jamais vraie, Makefile périmé, benchmark du chemin d'échec | non contre-vérifié | matrix.os == 'windows' jamais vrai ; VERSION = 0.2.0-dev. |

Aucun finding REFUTED : les 14 findings survivent, dont 4 CONFIRMED, 2 PARTIAL (sévérité/portée recalibrées), 8 non contre-vérifiés (medium/low).
