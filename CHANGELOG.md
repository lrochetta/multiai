# Changelog

All notable changes to the multiai project.

---

## [multiai-go 0.4.0-dev] — Unreleased

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
