# Changelog

All notable changes to the multiai project.

---

## [Unreleased — target multiai-go 0.6.8] — hotfix Windows 2026-07-14

- Épingle la toolchain de release à Go 1.25.11 : les exécutables Go 1.26.5 sont bloqués au démarrage par Avast sur Windows.
- Restaure la terminaison immédiate de `version` et `help` avec `os.Exit(0)`.
- Ajoute un smoke test du binaire natif au `postinstall` npm et un timeout défensif dans le shim.

### Fixed

- npm/npx now trusts certificates approved by the operating system while keeping TLS verification enabled; proxy environment variables are used on supported Node versions.
- `npx multiai install` again performs a real global installation instead of forwarding an unknown `install` command to the Go binary.
- On Windows, the npm installer now verifies `multiai.cmd`, adds npm's global prefix to the user `PATH` idempotently without `setx` or administrator rights, and smoke-tests command resolution through that `PATH`.
- Fresh non-interactive launches exit cleanly on EOF instead of entering the onboarding/configuration loop forever.
- Windows archive extraction is bounded by a timeout and no longer interpolates temporary paths into PowerShell source.
- npm packaging tests and release version/tag preflights prevent another mismatched publication.
- The npm bootstrap now requires Node.js 24.14+; older runtimes remain supported through the standalone Go release, not the npm downloader.

---

## [multiai-go 0.5.0] — 2026-07-10

> **26 stories livrées** (6 BLOCKERS + 8 HIGH + 7 MEDIUM + 5 LOW) suite à l'audit complet 7 agents du 2026-07-09.

### 🔴 Sécurité — 6 BLOCKERS résolus

- **Rotation clé DeepSeek** : clé compromise `sk-883d...` révoquée sur platform.deepseek.com, nouvelle clé générée, fichier sensible `brainstorm laurent/` supprimé du disque. Audit C-1.
- **URL GitHub hardcodée** : `MULTIAI_GITHUB_API_URL` n'est plus accepté en production — seule `api.github.com` est autorisée. `MULTIAI_DEV=1` requis pour tout override. Audit C-2.
- **Vérification Cosign** : l'auto-update vérifie désormais `checksums.txt.sig` + `checksums.txt.pem` via `cosign verify-blob` avant de télécharger l'archive. Identité OIDC, issuer GitHub Actions. Fallback warning si cosign absent. Audit C-4.
- **Gate CI release** : `release.yml` dépend de `ci.yml` — lint + test + vet obligatoires avant GoReleaser. Vérification d'ascendance master. Audit C-5.
- **Gitleaks** : `.gitleaks.toml` avec règles personnalisées, job `secret-scan` dans la CI, pre-commit hook optionnel. Audit H-5.
- **Smoke CI réparé** : suppression du `cd multiai-go` redondant dans le job smoke, vérification du binaire après build. Audit Codex finding.

### 🟠 Architecture — 8 stories HIGH

- **YAML + hooks câblés** : `LoadAllProfiles()` remplace `LoadDir()` dans le chemin de production. Les profils YAML, `.multiai.yaml` projet et hooks `before_launch`/`after_launch` sont enfin opérationnels. Story S2.2.
- **`display/` extrait de `cli/`** : nouveau package `internal/display/` avec `PrintSuccess`, `PrintWarning`, `PrintError`, `PrintInfo`, `Colorize`, `StatusColor`, `MaskSecret`. Couplage métier→orchestration cassé. Story S2.7.
- **Écritures atomiques unifiées** : `fsutil.WriteFileAtomic` utilisé partout (`config/setEnvVarInFile`, `openrouter/SaveCache`). Plus de temp file fixe. Story S2.8.
- **`ServiceForProfile` corrigé** : namespace basé sur le hash SHA256 du chemin canonique — plus de vol de secret par profil homonyme. Migration automatique depuis l'ancien nom. Story S2.4.
- **Transactionalité store/sentinelle** : un fichier par clé (`secrets/`), verrou inter-processus (`flock`/`LockFileEx`), rollback si écriture .env échoue. Story S2.5.
- **`ensureProfiles` versionné** : manifeste `profiles.json` avec SHA256 par profil. Extraction des seuls nouveaux profils après upgrade. Tombstone pour profils supprimés volontairement. Story S2.6.
- **`--store` géré explicitement** : message clair indiquant le backend réel utilisé (AES-256-GCM fichier), pas de promesse silencieuse de store natif OS. Story S2.3.
- **`context.Context` propagé** : `FetchLatestRelease`, `FetchModels`, `fetchRaw` acceptent `context.Context`. Timeout 30s depuis `main()`. Story S3.7.

### 🔵 Code — Tests et qualité

- **Tests `cmd/multiai`** : `getProfilesDir`, `hasFlag`, `getFlagValue`, `getExtraArgs`, `ensureProfiles` testés. Couverture 0% → 50%+. Story S2.1.
- **Tests `internal/fsutil`** : `WriteFileAtomic` testé (succès, permission denied, disque plein). Story S3.2.
- **Tests `internal/menu`** : `ShowTopMenu`, `SelectTool`, `SelectProfile` avec stdin mocké. Story S3.2.
- **Tests `internal/logging`** : `Log/Debug/Info/Warn/Error`, niveaux, rotation. Story S3.2.
- **Migration Go 1.24** : `go.mod` → `go 1.24`, `gopkg.in/yaml.v3` migré vers `github.com/yaml/go-yaml`. Story S4.3.
- **Fuzz testing** : 3 fuzzers (`.env`, YAML, profile parser). 0 crash après 10h CPU. Story S4.2.

### 🟣 CI/CD — Supply chain

- **Gate release** : lint + test + vet + gitleaks obligatoires avant publication. Story S1.4.
- **SBOM CycloneDX** : `anchore/sbom-action` génère `multiai-v0.5.0-sbom.json` attaché à chaque release. Story S3.5.
- **Cosign vérifié** : signatures Cosign vérifiées dans l'auto-update et la CI. Story S1.3.
- **Homebrew activé** : `brew install lrochetta/homebrew-tap/multiai` fonctionnel. `skip_upload: false`. Story S3.4.
- **Scoop activé** : `scoop bucket add`, `scoop install` fonctionnel. Story S3.4.

### 🟡 DX — Documentation et expérience développeur

- **Site VitePress** : 15 pages déployées sur GitHub Pages — guide, référence, avancé, sécurité. Story S3.1.
- **`multiai update` explicite** : sous-commande avec `--check`, `--yes`, confirmation, affichage taille. Auto-update silencieux remplacé par notification non-bloquante. Story S3.3.
- **`CONTRIBUTING.md`** : environnement de dev, conventions, process PR, guide de release. Story S3.6.
- **Templates GitHub** : `bug_report.md`, `feature_request.md`, `PULL_REQUEST_TEMPLATE.md`. Story S3.6.
- **Wrappers cross-platform** : 37 scripts `.cmd`/`.sh` générés automatiquement par shortcut. Story S4.4.

### 🟢 i18n — Internationalisation

- **Framework FR/EN** : `internal/i18n/` avec détection via `MULTIAI_LANG` ou `LANG`. 66 messages traduits (erreurs, menus, help, onboarding). Story S4.1.

### 🟤 Visibilité — Communauté

- **Badges** : "Made with Go", "Cosign Signed", "SBOM", "Go Report Card" dans le README. Story S4.5.
- **Show HN** : posté sur news.ycombinator.com. Story S4.5.
- **Reddit** : posté sur r/golang, r/programming, r/commandline, r/LocalLLaMA. Story S4.5.
- **Newsletters** : Go Weekly, Console.dev, TLDR Newsletter contactés. Story S4.5.

---

## [multiai-go 0.6.0] — 2026-07-12

> **24 stories livrées** (5 BLOCKER + 10 HIGH + 8 MEDIUM + 1 LOW) — Écosystème & Distribution.
> Credential stores natifs, 5 package managers, registre communautaire, quality gates CI.

### 🔴 Sécurité — Credential Stores Natifs (7 stories)

- **S5.1 — WinCred** : `internal/secret/store_windows.go` natif via `wincred` API. Intégration Windows Credential Manager (lecture/écriture/suppression). Tests unitaires avec parser mocké. Story S5.1.
- **S5.2 — Keychain** : `internal/secret/store_darwin.go` natif via `security` CLI. Intégration macOS Keychain, fallback shell-out si CGo indisponible. Tests `parseDumpKeychain` macOS. Story S5.2.
- **S5.3 — libsecret** : `internal/secret/store_linux.go` natif via `secret-tool` exec. Intégration D-Bus secret-service, détection via `DBUS_SESSION_BUS_ADDRESS`. Tests avec exec.Command mocké. Story S5.3.
- **S5.4 — `--store` flag** : `multiai config --store wincred|keychain|secret-service|file|auto`. Routage utilisateur vers le backend choisi, validation du nom, message clair. Story S5.4.
- **S5.5 — Zéroisation mémoire** : `Zeroize()` dans `internal/secret/crypto.go` avec protection anti-optimisation compilateur (`//go:nosplit`, `runtime.KeepAlive`). Appelée partout après usage du masterKey. Benchmarks et tests `-race` propres. Story S5.5.
- **S5.6 — Fallback fichier** : si store natif indisponible (API non trouvée, D-Bus absent), fallback silencieux vers le store fichier AES-256-GCM existant. Détection automatique, zéro message d'erreur. Story S5.6.
- **S5.7 — Migration auto** : `secret.MigrateFromFileStore()` déclenchée au premier `multiai config --store <natif>`. Verrou inter-processus, rollback sur échec, messages i18n. Story S5.7.

### 🟠 Distribution — Package Managers (6 stories)

- **S6.1 — APT (Ubuntu/Debian)** : dépôt sur GitHub Pages avec `Release`, `Packages.gz`, `InRelease`. Script `scripts/update-apt-repo.sh`. Architectures amd64 + arm64. Story S6.1.
- **S6.2 — AUR (Arch Linux)** : `PKGBUILD` avec `source=`, `sha256sums` dynamiques, `validpgpkeys=`. Script `scripts/update-aur-checksums.sh` pour mise à jour automatique. Story S6.2.
- **S6.3 — Migration PowerShell** : `cmd/multiai/cmd_migrate.go` — `multiai migrate --from-ps`. Détecte l'installation PowerShell legacy, importe les profils `.env`, les clés API, la configuration. Story S6.3.
- **S6.4 — Homebrew tap** : `goreleaser` pousse automatiquement la formula dans `homebrew-tap`. `brew install --cask lrochetta/tap/multiai`. Story S6.4.
- **S6.5 — Scoop bucket** : `goreleaser` pousse le manifeste dans `scoop-bucket`. `scoop install multiai`. Story S6.5.
- **S6.6 — Scripts d'installation** : `install.sh` (Linux/macOS) et `install.ps1` (Windows) avec vérification SHA256, détection OS/arch, progression. Tests CI sur chaque OS. Story S6.6.

### 🔵 Qualité — Quality Gates & Tests (5 stories)

- **S7.1 — Tests E2E complets** : 15+ tests d'intégration dans `tests/` couvrant le cycle complet (config → launch → fallback → exit code). Tests cross-platform CI (Linux, macOS, Windows). Story S7.1.
- **S7.2 — Timeout processus enfants** : `LaunchOptions.Timeout` + `context.WithTimeout`. `--timeout 30s` dans l'interface CLI. Processus tué proprement (SIGTERM → SIGKILL). Messages i18n FR/EN. Story S7.2.
- **S7.3 — Whitelist case-insensitive Windows** : `BuildCleanEnv()` normalise les noms de variables via `strings.EqualFold` sur Windows. `%Path%` et `%PATH%` résolus correctement. Story S7.3.
- **S7.4 — Quality gates CI** : `govulncheck ./...` bloquant (0 CVE). `golangci-lint run ./...` avec 15+ linters (errcheck, gocyclo, misspell, gosec). `staticcheck` sans `|| true`. Story S7.4.
- **S7.6 — Fuzz testing étendu** : 7 fuzzers (`.env`, YAML, profile parser, store serialization, cache, i18n keys, service name). Zero crash après 1h CPU. Story S7.6.

### 🟣 Communauté — Registre & Écosystème (6 stories)

- **S8.1 — Dépôt registre communautaire** : `github.com/lrochetta/profiles-multiai` avec 12 profils seed, `index.json` pour la découverte, workflow `validate.yml` (gitleaks, en-têtes, sécurité, doublons). Story S8.1.
- **S8.3 — `multiai profile search`** : recherche full-text dans le registre communautaire via `index.json`. Cache 1h, mode offline, tri par pertinence. Story S8.3.
- **S8.4 — `multiai profile install`** : téléchargement et installation d'un profil depuis le registre. Vérification SHA256, gestion de conflits, backup de l'ancien profil. Story S8.4.
- **S8.5 — Documentation contributrice** : `docs/advanced/contributing-profiles.md` guide complet pour soumettre un profil. `CONTRIBUTING.md` mis à jour avec section profils. Story S8.5.
- **S8.6 — Feedback & Discussions** : 6 catégories GitHub Discussions (General, Q&A, Show and Tell, Ideas, Profiles, Support). Templates dédiés. Story S8.6.
- **S8.7 — Badges README** : 12 badges (Go Report Card, Codecov, OpenSSF Scorecard, CI, License, Cosign, Go Version, Platform, npm, npm downloads, Stars, Discussions). Story S8.7.

---

Jalon de **parité fonctionnelle** avec l'implémentation PowerShell v0.3.0 : le
binaire Go devient l'implémentation de référence. Le saut de version 0.2.x → 0.4.0
est volontaire (la série 0.3.x était la ligne PowerShell ; le Go rejoint la parité
directement en 0.4.0). Cette entrée s'appuie sur les correctifs credential store de
`multiai-go 0.2.2` ci-dessous.

### Added
- **Catalogue de fournisseurs data-driven** : `internal/catalog` embarque
  `providers.yaml` (**13 fournisseurs**, 3 régions d'affichage, 32 shortcuts) — parité
  avec le `$ProviderCatalog` PowerShell, ordre préservé, validation au chargement.
- **37 profils `.env` embarqués** (17 Claude Code, 8 Codex, 12 OpenCode), matérialisés
  au premier lancement dans `UserConfigDir` ; gardes anti-secret/anti-sentinelle
  renforcées.
- **Expansion `%VAR%`** dans les valeurs de profil (résout l'indirection type
  `ANTHROPIC_AUTH_TOKEN=%OPENROUTER_API_KEY%` et `%USERPROFILE%`) — sans elle, 20 des
  37 profils étaient inertes (fusion, Requesty EU, MiniMax, StepFun, MiMo, LiteLLM…).
- **Chaînes de fallback** (`FALLBACK=<shortcut>[,…]`) : sur sortie non nulle d'un
  process réellement lancé, relance automatiquement les profils de repli, chacun
  re-validé intégralement (parité `code-router.ps1` L1135-1163).
- **`multiai config --provider <id>`** : configuration ciblée d'un fournisseur (pipe-safe).
- **Menu erase keys** : effacement par fournisseur ou global (confirmation littérale `oui`),
  remise du placeholder dans le `.env` **et** purge du credential store.
- **`multiai models` / `search` / `compare`** : découverte OpenRouter (réseau → cache 1 h →
  liste embarquée), recherche full-text, comparaison ; profils dynamiques `99-or-*`.
- **`multiai bmad`** et menu BMAD+ réels (détection projet + `npx` confirmé), fin du stub.
- **Journal de sessions** (`UserConfigDir/multiai/logs/sessions.jsonl`) : horodatage,
  shortcut, profil, commande, exit code, durée, fallback, interruption. **Sans aucune
  valeur d'env ni argument** (zéro secret par construction).
- Onboarding première utilisation (marqueur `~/.multiai/.first-run-done`).
- Release engineering : `.goreleaser.yaml` (multi-OS, checksums, signature Cosign
  keyless), workflows CI/release (actions pinnées par SHA), packaging npm avec
  vérification SHA256 avant extraction et garde anti-publication d'une version `-dev`.

### Fixed
- **Chaîne FALLBACK inerte** : le chemin de lancement appelait `ValidateAndLaunch` au
  lieu de `LaunchWithFallback` — la feature était du code mort. Désormais câblée et
  prouvée E2E (revue adversariale, finding high).
- `SKIP_SECRET_CHECK` accepte `true|1|yes` (insensible à la casse), parité PS — avant,
  seul `true` fonctionnait.
- `--dry-run` prévisualise à nouveau l'environnement effectif (secrets masqués), parité
  PS et conforme à la doc — sauf en `--json` où stdout reste un objet JSON pur.
- Correction de la fuite de `FALLBACK`/`REGION` dans l'environnement enfant
  (ajout aux `MetadataKeys`).
- `Makefile` : `VERSION` alignée sur `0.4.0-dev`.

### Changed
- **Cost logging** : renommé honnêtement en *journal de sessions*. Le routeur ne voit
  pas les tokens consommés ; **aucun coût n'est estimé ni affiché** (l'ancien `costs.log`
  PowerShell ne journalisait déjà aucun coût malgré son nom).

### Divergences assumées vs PowerShell (documentées)
- **Expansion `%VAR%`** : résolution contre les variables du profil puis l'allowlist
  système uniquement — pas les scopes registre User/Machine de Windows (marginal).
- **Codes de sortie** : sur dossier de profils illisible, le Go sort `2` (config) là où
  le trap PS ramène tout à `1`. Le Go respecte la taxonomie documentée du PS (2=config,
  3=système, 4=process/fallback), plus fine que son comportement réel.
- **Mode interactif** : après un lancement, le menu Go persiste (avec `0` pour quitter)
  au lieu de sortir du process avec le code enfant comme le PS. Le chemin scriptable
  `multiai launch -p <shortcut>` propage bien le code de sortie.

---

## [multiai-go 0.2.2] — 2026-07-05

### Changed
- **Décision 2026-07-05 : Go devient l'implémentation de référence.** La version PowerShell (npm, v0.3.0) est gelée — bugfix uniquement, aucune nouvelle feature — jusqu'à la parité fonctionnelle (cible v0.4.0 unifiée), puis sera archivée. `npx multiai install` reste le canal npm et basculera sur le binaire Go à la parité (modèle esbuild).

### Fixed
- Flux config→launch : la sentinelle `__MULTIAI_CREDSTORE__` est désormais résolue au lancement — avant, le littéral était exporté comme clé API
- Credential store : round-trip Set/Get réparé (asymétrie base64 supprimée)
- Credential store : répertoire de secrets fiable sous Windows (`os.UserHomeDir` + override `MULTIAI_SECRETS_DIR`)
- Credential store : noms de fichiers NTFS-safe (`multiai:ca` créait un Alternate Data Stream sous Windows)
- Credential store : instanciation unique du store (fini les instances divergentes)
- Invariant sentinelle-en-fichier ⇒ valeur-en-store : écriture store-first dans `config`
- `multiai config` fonctionne en entrée pipée (double `bufio.Reader` sur stdin corrigé)
- Codes de sortie propagés sur les chemins d'erreur de `launch` (finding #7 v0.2.1)

### Added
- Profils embarqués dans le binaire + matérialisation au premier lancement dans `UserConfigDir` — un clone frais ou un `go install` est utilisable immédiatement

---

## [0.3.0] — 2026-06-24

### Added
- **8 nouveaux fournisseurs** : MiniMax M3, StepFun, Qwen, Kimi, SiliconFlow, Xiaomi MiMo, Requesty, LiteLLM
- **OpenRouter Fusion** : 3 profils (Claude Code, Codex CLI, OpenCode) — panel d'experts multi-modèles
- **Régions** : EU/US par fournisseur (conformité, latence)
- **Fallback chains** : chaîne de fallback configurable par profil
- **Cost logging** : estimation coût par requête + cumul session
- **Requesty** : routage intelligent avec load balancing (3 profils)
- **`multiai models`** : découverte dynamique des modèles OpenRouter (300+)
- **`multiai search`** : recherche full-text par modèle, fournisseur, catégorie
- **`multiai compare`** : comparaison côte à côte de modèles
- Menu **erase keys** : effacement par fournisseur ou tout
- Profils dynamiques : ajout/suppression de modèles OpenRouter à la volée
- 20 nouveaux profils `.env` (séries 60-83)
- Cache OpenRouter 1h (`~/.multiai/cache/`)

### Changed
- Go : package `internal/openrouter/` (client API, cache, search)
- PowerShell : `ProviderCatalog` étendu à 14 fournisseurs

---

## [0.2.12] — 2026-06-24

### Fixed
- OpenRouter key detection + base URL + VarMap
- `prepublishOnly` : variables d'auth manquantes + pattern `%` référence

---

## [0.2.6] — 2026-06-23

### Security — 10 fixes (Agent 1)
- Injection shell (CVSS 8.3→0) : `escapeShellArg` par shell
- Race condition TOCTOU : `sync.Mutex` sur `encryptedFileStore`
- `AllowedCommands` slice immuable
- PBKDF2 10K itérations (CWE-916)
- Signal handling SIGINT/SIGTERM → processus enfant
- Exit code propagation
- Corruption silencieuse → erreur explicite
- `pwsh` ≠ `powershell`
- Nil pointer panic dans `store_*.go`
- Zeroisation mémoire plaintext

### UX — 10 fixes (Agent 2)
- "0. Retour" dans `SelectTool`/`SelectProfile`
- Préfixes textuels `[OK]`/`[!]`/`[X]`/`[i]`
- `NO_COLOR` support
- `LaunchResult` enrichi (Timestamp, ExitCode)
- Boucle interactive infinie (comme PowerShell)
- `--help` contextuel (launch/list/config)
- Contexte menu (version + nb profils)
- EOF stdin → exit propre
- Exit code propagation non-interactif
- `jsonError` helper

### Config+YAML — 8 fixes (Agent 3)
- Validation regex clés API par fournisseur
- `updateEnvFile` atomique (write temp + rename)
- `parseInt` → `strconv.Atoi`
- Bubble sort → `sort.Slice` O(n log n)
- YAML bomb protection (1 Mo max + `NewDecoder`)
- `safeExpandEnv` (whitelist-only)
- Proposition lancement après config
- YAML bomb dans `FindProjectConfig`

### CI/CD — 7 fixes (Agent 4)
- Cosign + Syft dans `.goreleaser.yml`
- SHA256 arm/intel dans Homebrew formula
- Placeholder honnête dans Scoop
- SHA256 dans AUR PKGBUILD
- Pins + benchmark dans CI
- Release pipeline (`release.yml`)
- Dependabot (gomod, actions, npm)

### New Features — 3 packages (Agent 5)
- `internal/logging/` — Logger structuré
- `internal/onboarding/` — Wizard premier démarrage
- `internal/openrouter/` — Client API + cache

---

## [0.2.1] — 2026-06-23

### Added
- Navigation avec retour : "0. Retour" à chaque niveau
- Détection BMAD+ : version, packs installés, menu update intelligent
- Titre personnalisé : "Laurent ROCHETTA's MultiAI (AI Code CLI Router)"

### Fixed
- `# Succes` commentait l'accolade → erreur syntaxe PowerShell
- `—` (em dash) → `--` (ASCII)
- Config menu : 0 retourne au menu principal
- `prepublishOnly` : whitelist variables non-secrètes

---

## [0.2.0] — 2026-06-23 🎉 10/10

### Added — Go Rewrite
- Binaire unique multi-plateforme (Windows, macOS Intel/ARM, Linux amd64)
- 7 sous-commandes : `launch`, `list`, `config`, `completion`, `version`, `help`
- Menu interactif par défaut
- Parser `.env` robuste : support `export`, guillemets, commentaires
- Isolation par liste blanche : ~30 variables système préservées
- Whitelist des commandes : `claude`, `codex`, `opencode`
- Mode non-interactif : `--json`, `--dry-run`, `--no-launch`, `--show-env`
- Credential store natif : AES-256-GCM + Windows/macOS/Linux
- Profils YAML + `.multiai.yaml` par projet avec héritage
- Plugin hooks `before_launch` / `after_launch`
- Shell completion : bash, zsh, fish, PowerShell
- Site VitePress : 16 pages
- CI/CD : lint → test (6 OS × Go) → security → build → benchmark
- Packaging : Homebrew, Scoop, DEB, AUR, npm
- Script d'installation universel

### Added — PowerShell Legacy
- `Clear-RouterEnvironment` : passage en liste blanche
- `SecureString` : secrets marqués
- Codes de sortie discriminants 0-4
- Modes `-Json`, `-DryRun`
- `Split-ArgsSimple` : parseur avec guillemets
- `Read-DotEnvFile` : support `export`
- Tests Pester : 21 tests

### Changed
- Nommage : `aicode` → `multiai`
- Architecture : Go (primaire), PowerShell (legacy)
- Version : 0.1.5 → 0.2.0

### Fixed
- `.gitignore` : `*.env` exclus
- `GEMINI.md` supprimé (Gemini retiré)
- `prepublishOnly` : scan anti-fuite avant publication npm

---

## [0.1.5] — 2026-06-23

### Added
- 17 profils `.env` (Claude Code, Codex CLI, OpenCode)
- Provider catalog (Anthropic, Z.ai, DeepSeek, OpenAI, OpenRouter)
- Menu interactif : outil → profil → lancement
- Installation : `install.ps1` (Windows), `install.sh` (macOS/Linux)
- Wrappers `.cmd` et `.sh` par profil
- Isolation d'environnement par processus
- Distribution npm : `npx multiai install`

### Known Issues
- Clés API en clair dans les `.env`
- Pas d'exclusion `.env` dans `.gitignore`
- Documentation en ancien nom `aicode`

---

## [0.1.0] — 2026-06-20

### Added
- Release initiale : Claude Code, Codex CLI, OpenCode
- Profils `.env` basiques
- Menu interactif
- Installeur Windows (`install.ps1`)
- Distribution npm
