Maintenant j'ai une vision complete du codebase. Voici les 6 stories detaillées pour l'epic "Qualite & Robustesse" (v0.6.0).

---

# Epic: Qualite & Robustesse (v0.6.0)

## S7.1 — Tests d'integration complets (E2E)

**Priorite:** HIGH

**Objectif:** Couvrir l'ensemble du pipeline CLI par des tests d'integration reproductibles, sans dependances externes.

**Description technique:**
Les tests `tests/integration_test.go` actuels couvrent le chargement de profils isolement mais n'exercent jamais la boucle complete : parsing des flags → selection du profil → resolution du credential store → build de l'environnement → lancement du processus → collecte du resultat. Il faut une suite E2E qui utilise des binaires factices (scripts shell/batch qui se comportent comme claude/codex) pour verifier les comportements attendus : exit codes propages, fallback declenche, signaux forwardes, `--dry-run`, `--show-env`, `--json`, `--no-launch`, arguments `--` transmis, variable d'environnement `MULTIAI_PROFILES_DIR`, detection de profiles malformes, profils YAML avec hooks. Il faut aussi un helper `cmd/multiai/testutil.go` qui compile le binaire et l'appelle avec des arguments controles dans un tempdir isole. Les tests utiliseront `t.Parallel()` pour chaque scenario et un cleanup systematique des fichiers temporaires.

**Fichiers impactes:**
- `tests/integration_test.go` — reecriture complete (ou `tests/e2e_test.go`)
- `cmd/multiai/main_test.go` — ajout de helpers `runMultiai()` et `runMultiaiWithEnv()`
- `cmd/multiai/testutil.go` — NOUVEAU: helpers de compilation et d'execution
- `tests/testdata/` — NOUVEAU: binaires factices et fichiers de config
- `internal/cli/launcher.go` — potentiel refactoring mineur pour injectabilite

**Tests attendus:**
- `TestE2E_LaunchDryRun` — `multiai launch -p ds --dry-run` ne lance rien, retourne 0
- `TestE2E_LaunchJSON` — `multiai launch -p ds --json` produit un JSON valide
- `TestE2E_LaunchWithExtraArgs` — args apres `--` transmis au processus enfant
- `TestE2E_LaunchFallback` — processus enfant exit 1 → fallback vers le profil suivant
- `TestE2E_LaunchFallbackInterrupt` — Ctrl+C pendant le premier processus ne declenche PAS le fallback
- `TestE2E_LaunchMissingCommand` — commande introuvable → erreur claire
- `TestE2E_LaunchMissingSecret` — sentinel non resolue → erreur
- `TestE2E_ConfigStoreRoundtrip` — `multiai config --provider deepseek` → ecrit le store → verifie la resolution
- `TestE2E_ProfilesDirOverride` — `MULTIAI_PROFILES_DIR` pointe vers un dossier custom
- `TestE2E_ListJSON` — `multiai list --json` produit un JSON valide avec tous les profils
- `TestE2E_InvalidProfile` — profile .env malforme → ignore avec warning, pas de crash
- `TestE2E_YAMLProfileLoading` — profile .yaml charge correctement avec hooks et env
- `TestE2E_ShowEnv` — `--show-env` affiche les vars resolues sans secret visible
- `TestE2E_NoLaunch` — `--no-launch` prepare sans lancer, retourne 0
- `TestE2E_SmokeVersion` — `multiai version` retourne le bon format

**Resultat attendu:** Suite de tests E2E executable en < 30s, couvrant 15+ scenarios critiques. Possibilite de remplacer le smoke test CI actuel (qui ne teste que `version` et `list --json`).

**Definition of Done:**
- 15+ tests E2E passant sur Linux, macOS, Windows
- Binaires factices dans `testdata/` signales par une convention de nommage claire
- Helper `runMultiai()` factorise dans `testutil.go`
- `go test -tags=e2e ./tests/` lance la suite complete
- CI met a jour pour inclure `-tags=e2e` dans la job `test`
- Documentation dans `CONTRIBUTING.md` sur comment ajouter un test E2E

**Risques:**
- Dependance a `go build` dans les tests : ralenti, necessite Go installe. Solution : precompiler le binaire dans `TestMain`.
- Binaires factices differents par OS (`.sh` vs `.bat`). Solution : utiliser `testdata/scripts/` avec detection OS.
- Tests non-deterministes si `os.Environ()` contient des variables interferentes. Solution : `MULTIAI_PROFILES_DIR` + `HOME` isole.

**Dependances:** Aucune (travail independant).

---

## S7.2 — Timeout/context sur processus enfants

**Priorite:** BLOCKER

**Objectif:** Ajouter un timeout configurable et une propagation de contexte aux processus enfants lances par `ValidateAndLaunch`, pour eviter les processus orphelins et les blocages infinis.

**Description technique:**
Actuellement `launcher.go:L146-L218` utilise `exec.Command` sans contexte de timeout. Si un processus enfant se bloque (API qui ne repond pas, deadlock interne), `cmd.Wait()` bloque indefiniment et multiai ne peut pas etre interrompu proprement. Il faut ajouter une option `Timeout time.Duration` dans `LaunchOptions`, utilisee via `context.WithTimeout`. Une goroutine de surveillance tue le processus enfant apres le delai (`cmd.Process.Kill()`). La valeur par defaut est 0 (pas de timeout, comportement actuel). Un signal `SIGTERM` est envoye d'abord, suivi de `SIGKILL` apres 5s de grace. Le `LaunchResult` doit reporter le timeout (`Status: "timeout"`). La propagation du `context.Context` depuis `main.go` (auto-update a deja son ctx) jusqu'a `ValidateAndLaunch` doit etre faite sans casser l'API actuelle. Penser a la compatibilite des profils YAML qui pourraient declarer un `timeout` dans leur configuration.

**Fichiers impactes:**
- `internal/cli/launcher.go` — ajout `Timeout` dans `LaunchOptions`, logique de kill dans `ValidateAndLaunch`
- `internal/cli/fallback.go` — propagation du timeout dans `LaunchWithFallback`
- `internal/cli/launcher_test.go` — NOUVEAU: tests unitaires du timeout
- `cmd/multiai/main.go` — propagation du ctx dans `runLaunch`
- `internal/profile/profile.go` — optionnel: champ `Timeout` dans la structure Profile
- `internal/profile/yaml.go` — optionnel: parsing du timeout YAML

**Tests attendus:**
- `TestLaunchTimeout_Exceeded` — processus qui dort 10s avec timeout 1s → `Status: "timeout"`
- `TestLaunchTimeout_NotExceeded` — processus court avec timeout long → termine normalement
- `TestLaunchTimeout_ZeroValue` — `Timeout: 0` → comportement actuel (pas de limite)
- `TestLaunchTimeout_FallbackRespectsTimeout` — timeout declenche pendant le premier essai → fallback tente avec le meme timeout
- `TestLaunchTimeout_PropagatedCancel` — annulation du ctx parent (via defer) → processus tue
- `TestLaunchTimeout_GracePeriod` — verifier que `SIGTERM` est envoye avant `SIGKILL`

**Resultat attendu:** Tout processus enfant est termine proprement dans les X secondes configurees. Aucun processus orphelin possible. La commande `multiai launch -p ds --timeout 30s` existe en UX.

**Definition of Done:**
- `LaunchOptions.Timeout` implemente et documente
- Signaux de terminaison en deux temps (SIGTERM → 5s → SIGKILL)
- Propagation du `context.Context` depuis le main jusqu'a l'exec.Command
- Tests unitaires pour timeout depasse, timeout respecte, timeout zero
- Tests sur les 3 OS (le comportement des signaux differe sur Windows)
- Documente dans le help `launch --help`

**Risques:**
- Windows ne gere pas SIGTERM (seulement `TerminateProcess`). Solution : `cmd.Process.Kill()` fonctionne sur Windows, mais la grace period est inutile → detection `runtime.GOOS`.
- Processus enfant qui fork : le timeout ne tue que le processus direct, pas ses enfants. Solution : documenter cette limitation, utiliser des groupes de processus si disponible.
- Retrocompatibilite : les appels existants a `LaunchOptions{}` sans `Timeout` doivent fonctionner sans changement.

**Dependances:** Aucune (independant).

---

## S7.3 — Whitelist env case-insensitive sous Windows

**Priorite:** BLOCKER

**Objectif:** Rendre la whitelist `AllowedEnvVars` case-insensitive sous Windows pour eviter les fuites de variables systeme et les comportements incoherents.

**Description technique:**
Dans `env.go:L10-L22`, `AllowedEnvVars` est un `map[string]bool` avec lookup sensible a la casse. Sur Windows, `os.Environ()` retourne `PATH=...` MAIS parfois `Path=...` (certaines applications definissent des variantes). La fonction `BuildCleanEnv` fait `AllowedEnvVars[key]` qui echoue si la casse differe. La variable `PATH` est alors exclue de l'environnement nettoye, ce qui rend le processus enfant incapable de trouver ses binaires. La solution est de normaliser les cles en majuscules pour le lookup : soit en convertissant les cles dans la map (`AllowedEnvVars[strings.ToUpper(key)]`), soit en normalisant dans `BuildCleanEnv`. Il faut aussi verifier `expandWindowsVars` qui fait `AllowedEnvVars[name]` — la resolution des references `%PATH%` doit etre case-insensitive. Le comportement sur Linux/macOS doit rester inchange (sensibilite a la casse preservee). Une constante `normalizeEnvKey(key string) string` permet de centraliser la logique.

**Fichiers impactes:**
- `internal/env/env.go` — ajout de `normalizeEnvKey()`, modification de `BuildCleanEnv()` et `expandWindowsVars()`
- `internal/env/env_test.go` — tests du comportement Windows avec `t.Setenv("Path", ...)` (note: "Path" pas "PATH")

**Tests attendus:**
- `TestBuildCleanEnv_WindowsCaseInsensitive` — `os.Setenv("Path", "/custom/bin")` → `envMap["PATH"]` present avec la valeur
- `TestBuildCleanEnv_WindowsMixedCaseKeys` — `os.Setenv("Path", "/a")` + `os.Setenv("PATH", "/b")` → une seule entree /b
- `TestAllowedEnvVars_CrossPlatform` — `normalizeEnvKey("PATH")` = `normalizeEnvKey("Path")` sur Windows, different sinon
- `TestExpandWindowsVars_CaseInsensitiveWindows` — `%path%` = `%PATH%` sur Windows, pas sur Linux
- `TestBuildCleanEnv_LinuxStaysCaseSensitive` — `os.Setenv("Path", "/a")` n'est PAS dans la whitelist sur Linux

**Resultat attendu:** `multiai launch` sur Windows preserve bien `PATH` (et toutes les vars de la whitelist) quelle que soit la casse utilisee par le processus parent ou le systeme. Les processus enfants trouvent leurs binaires.

**Definition of Done:**
- `normalizeEnvKey()` implementee avec detection `runtime.GOOS`
- `BuildCleanEnv()` utilise `normalizeEnvKey()` pour le lookup whitelist
- `expandWindowsVars()` utilise `normalizeEnvKey()` pour le lookup whitelist
- Tests unitaires couvrant Windows (cassee sensible), Linux (casse sensible), et le cas `%Path%` vs `%PATH%`
- CI verte sur les 3 OS
- Aucune regression sur les tests existants `TestBuildCleanEnv`, `TestExpandProfileEnv`

**Risques:**
- Detection `runtime.GOOS` rend le comportement lie a la compilation, pas a l'execution. Pour un binaire compile sur Linux mais invoque sous Wine, le comportement serait incorrect. Acceptable pour l'usage prevu.
- Si deux variables avec la meme casse differente existent (ex: `Path` et `PATH`), une seule est conservee. C'est coherent avec le comportement natif Windows (la derniere gagne).

**Dependances:** Aucune.

---

## S7.4 — Govulncheck CI non-bloquant → bloquant (0 vulnerabilite)

**Priorite:** MEDIUM

**Objectif:** Rendre govulncheck bloquant dans la CI apres avoir ramene a zero les vulnerabilites connues dans la stack Go.

**Description technique:**
Dans `ci.yml:L99`, la ligne `go run golang.org/x/vuln/cmd/govulncheck@latest ./... || true` ignore systematiquement les vulnerabilites. Il faut d'abord auditer l'etat actuel : `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` sans `|| true` pour collecter les CVEs. Puis mettre a jour les dependances (upgrade de modules Go, mise a jour du go.mod) pour corriger chaque CVE identifiee. Si une fausse-positive est detectee (vulnerabilite dans une dependance indirecte non utilisee), elle doit etre listee dans un fichier `.govulncheck-ignore.yaml` (ou equivalent) avec justification. Le format d'ignore n'existe pas nativement dans govulncheck — solution : wrapper `scripts/govulncheck.sh` qui filtre les resultats avec `grep -vF -f .govulncheck-ignore`. Apres mise a jour et zero CVE, remplacer `|| true` par un appel au wrapper. Le meme changement doit etre applique dans le Makefile ou une cible `make vulncheck` est presente.

**Fichiers impactes:**
- `.github/workflows/ci.yml` — ligne 99 : `|| true` → appel au wrapper
- `Makefile` — ajout cible `vulncheck` (ou mise a jour de `lint`)
- `scripts/govulncheck.sh` — NOUVEAU : wrapper avec support d'ignore
- `.govulncheck-ignore` — NOUVEAU : liste de faux-positifs (si necessaire)
- `go.mod` / `go.sum` — mise a jour des dependances

**Tests attendus:**
- Aucun test unitaire (changement CI / build tooling)
- Verification manuelle : `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` retourne 0

**Resultat attendu:** La CI echoue si une CVE est presente dans l'une des dependances Go. Les fausses-positives sont ignorees via un fichier explicite.

**Definition of Done:**
- Audit initial : `govulncheck` sans `|| true` identifie 0 CVE actuellement
- Si CVE existantes : toutes corrigees par mise a jour de dependances
- Fichier `.govulncheck-ignore` cree si necessaire avec justifications
- Wrapper `scripts/govulncheck.sh` implemente
- CI job `security` echoue quand `govulncheck` trouve une CVE non ignoree
- Cible `make vulncheck` disponible localement

**Risques:**
- `govulncheck@latest` peut avoir des regressions (faux-positifs sur des vieux modules). Solution : le wrapper filtre.
- Mise a jour de dependances peut casser la compilation. Solution : tests en amont.
- `golang.org/x/vuln/cmd/govulncheck@latest` n'est pas pinne — un changement de comportement peut casser la CI. Solution acceptable car l'outil est stable.

**Dependances:** Aucune.

---

## S7.5 — Golangci-lint non-bloquant → bloquant (0 warning)

**Priorite:** MEDIUM

**Objectif:** Ramener a zero les warnings golangci-lint et rendre le job CI bloquant.

**Description technique:**
Dans `ci.yml:L63`, l'argument `--issues-exit-code=0` neutralise la valeur de sortie de golangci-lint. Il faut d'abord identifer tous les warnings actuels en lancant `golangci-lint run ./...` (sans `--issues-exit-code=0`). Chaque warning doit etre corrige ou explicitement ignore via la configuration `.golangci.yml`. La configuration actuelle est minimale (`version: "2"`, `disable: [errcheck]`, `timeout: 5m`). Il faut la completer : ajouter les linters souhaites (govet, staticcheck, revive, gosimple, unused, ineffassign, prealloc, gocritic, bodyclose, noctx, errname, errorlint, etc.) et definir des exclusions pour les patterns acceptes (ex: `G104` pour les prints qui ignorent l'erreur de retour). Apres nettoyage, remplacer `--issues-exit-code=0` par `--issues-exit-code=1` (ou supprimer le flag, 1 etant la valeur par defaut). Le `.golangci.yml` doit etre commente pour expliquer chaque exclusion. Cibler `golangci-lint` v2 (deja configure).

**Fichiers impactes:**
- `.github/workflows/ci.yml` — ligne 63 : `--issues-exit-code=0` → `--issues-exit-code=1` (ou supprimer)
- `.golangci.yml` — configuration enrichie avec linters actifs et exclusions commentees
- Multiples fichiers `.go` — corrections des warnings (noms non conventionnels, erreurs non checked, code mort, etc.)

**Tests attendus:**
- Aucun test specifique (modifications de style / lint)
- Verification : `golangci-lint run ./...` retourne 0 sur l'ensemble du codebase

**Resultat attendu:** La CI echoue si un warning lint est present. Le codebase respecte les regles definies dans `.golangci.yml`.

**Definition of Done:**
- Audit initial : `golangci-lint run ./...` (sans `--issues-exit-code=0`) identifie tous les warnings
- Chaque warning est corrige ou exclu avec un commentaire justificatif dans `.golangci.yml`
- `.golangci.yml` configure avec la liste des linters souhaitee et les exclusions
- `--issues-exit-code=0` remplace par `--issues-exit-code=1` dans la CI
- Verification que `make lint` inclut desormais `golangci-lint` (pas seulement `go vet`)
- Documentation dans `CONTRIBUTING.md` sur les regles de lint

**Risques:**
- `golangci-lint` version `latest` (non pinne) peut avoir des nouvelles regles qui cassent la CI. Solution : soit `golangci-lint-action@v7` est pinne, soit ajouter un fichier de version.
- Corrections de lint peuvent modifier l'API publique (ex: `IsCommandAllowed` → `IsCommandAllowed` est deja correct). Les corrections sont cosmétiques uniquement.
- `errcheck` actuellement desactive — des que reactive, il y aura des centaines d'erreurs. Solution : le reactiver progressivement, package par package, avec `//nolint` temporaires.

**Dependances:** S7.4 (les deux sont des chores CI, peuvent etre traites en parallele).

---

## S7.6 — Fuzz testing étendu (5+ fuzzers supplémentaires)

**Priorite:** MEDIUM

**Objectif:** Ajouter 5 nouveaux fuzzers ciblant les entrees non encore fuzzees (launcher, config, menu, logging, secret) pour atteindre une couverture >= 70% des fonctions critiques.

**Description technique:**
Actuellement 3 fuzzers existent : dotenv.ParseBytes (pkg/dotenv/fuzz_test.go), env.ExpandProfileEnv (internal/env/env_fuzz_test.go), profile.yamlToProfile (internal/profile/yaml_fuzz_test.go). Il faut ajouter au moins 5 fuzzers supplementaires :
1. **`FuzzParseArgs`** — `profile/splitArgs()` pour les arguments entre guillemets, echappements, cas limites
2. **`FuzzBuildCleanEnv`** — `env.BuildCleanEnv()` avec variations d'env systeme et profile env
3. **`FuzzSecretEncrypt`** — `secret` package : encrypt/decrypt roundtrip avec valeurs binaires, tres longues, vides
4. **`FuzzMenuInput`** — `menu` package : parsing de choix utilisateur, caracteres speciaux, entrees vides
5. **`FuzzConfigValidation`** — `config` package : validation de formats de cle API, patterns regex
6. **`FuzzSessionLogging`** — `logging` package : ecriture de logs avec caracteres non-ASCII, tres longues lignes
7. **`FuzzHashService`** — `secret.ServiceForProfile()` avec chemins de fichiers pathologiques (très longs, caracteres unicode, chemins Windows)
Chaque fuzzer doit etre marque `//go:noinline` et suivre le pattern des fuzzers existants. Le seed corpus doit representer les entrees reelles (profiles .env, configurations utilisateur, chemins de fichiers). Apres implémentation, lancer les fuzzers pendant 1h CPU en pre-commit pour verifier zero crash.

**Fichiers impactes:**
- `internal/profile/splitargs_fuzz_test.go` — NOUVEAU : fuzzer pour `splitArgs()`
- `internal/env/env_fuzz_test.go` — AJOUT : `FuzzBuildCleanEnv`
- `internal/secret/secret_fuzz_test.go` — NOUVEAU : fuzzers encrypt/decrypt + ServiceForProfile
- `internal/menu/menu_fuzz_test.go` — NOUVEAU : fuzzer pour le parsing de choix
- `internal/config/config_fuzz_test.go` — NOUVEAU : fuzzer validation des cles API
- `internal/logging/logging_fuzz_test.go` — NOUVEAU : fuzzer pour l'ecriture de logs
- `.github/workflows/ci.yml` — AJOUT : job de fuzz testing CI (optionnel : run court)
- `Makefile` — AJOUT : cible `fuzz` pour lancer tous les fuzzers

**Tests attendus:**
- 7 fuzzers (5 nouveaux + 2 existants), zero crash apres 1h CPU
- `go test -fuzz=FuzzParseArgs -fuzztime=5m ./internal/profile/`
- `go test -fuzz=FuzzBuildCleanEnv -fuzztime=5m ./internal/env/`
- `go test -fuzz=FuzzSecretEncrypt -fuzztime=5m ./internal/secret/`
- `go test -fuzz=FuzzMenuInput -fuzztime=5m ./internal/menu/`
- `go test -fuzz=FuzzConfigValidation -fuzztime=5m ./internal/config/`
- `go test -fuzz=FuzzSessionLogging -fuzztime=5m ./internal/logging/`
- `go test -fuzz=FuzzHashService -fuzztime=5m ./internal/secret/`

**Resultat attendu:** 7 fuzzers operationnels, zero crash, 0 panics. Couverture des chemins de code critiques portant sur le parsing d'entrees utilisateur et la manipulation de fichiers.

**Definition of Done:**
- 5 nouveaux fuzzers (minimum) implementes et operationnels
- Chaque fuzzer a un seed corpus representatif (10+ seeds)
- `make fuzz` lance tous les fuzzers sequentiellement avec `-fuzztime=5m`
- Documentation du processus de fuzzing dans `CONTRIBUTING.md`
- CI run courte (1m par fuzzer) pour detection rapide, avec recommandation de run longue locale
- Zero crash apres 1h CPU par fuzzer

**Risques:**
- Fuzzers lents (surtout secret avec crypto) : limiter `-fuzztime` en CI.
- Fuzzers non-deterministes : `t.Skip()` si certaines conditions OS non remplies.
- Fuzzer de logging peut generer des fichiers volumineux sur le disque : isoler dans `t.TempDir()`.

**Dependances:** Aucune.

---

## Resume des Priorites

| Story | Priorite | Effort estimé | Dependances |
|-------|----------|---------------|-------------|
| S7.1 Tests E2E | HIGH | 5-8 jours | Aucune |
| S7.2 Timeout enfants | BLOCKER | 3-5 jours | Aucune |
| S7.3 Env case-insensitive | BLOCKER | 1-2 jours | Aucune |
| S7.4 Govulncheck bloquant | MEDIUM | 1-2 jours | Aucune |
| S7.5 Golangci-lint bloquant | MEDIUM | 3-5 jours | Aucune (parallele a S7.4) |
| S7.6 Fuzz testing etendu | MEDIUM | 4-6 jours | Aucune |

Total estimé : 20-28 jours/homme.

Les 2 BLOCKERS (S7.2, S7.3) corrigent des failles de securite et de fiabilite qui peuvent causer des processus orphelins (S7.2) et des echecs de lancement silencieux sous Windows (S7.3). Les 2 HIGH/MEDIUM de CI (S7.4, S7.5) font passer la qualite du code de "advisory" a "enforcee". Les tests (S7.1, S7.6) assurent la non-regression.