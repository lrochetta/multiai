# Changelog

All notable changes to the multiai project.

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
