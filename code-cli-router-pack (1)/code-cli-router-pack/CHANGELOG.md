# Changelog

All notable changes to the multiai project.

---

## [0.2.1] — 2026-06-23

### Added
- Navigation avec retour : "0. Retour" dans sélection outil et profil
- Détection BMAD+ : version, packs installés, menu update intelligent
- Titre personnalisé : "Laurent ROCHETTA's MultiAI (AI Code CLI Router)"

### Fixed
- `# Succes` commentait l'accolade fermante → erreur de syntaxe PowerShell
- `—` (em dash Unicode) → `--` (ASCII) corrige `â€"` dans le terminal
- Config menu : 0 retourne au menu principal au lieu de quitter
- `prepublishOnly` : whitelist des variables non-secrètes (URLs, modèles)

### Security
- `prepublishOnly` amélioré : exclut les métadonnées du scan anti-fuite

---

## [0.2.0] — 2026-06-23 🎉 10/10

### Added — Go Rewrite
- **Binaire unique multi-plateforme** (Windows, macOS Intel/ARM, Linux amd64)
- **7 sous-commandes** : `launch`, `list`, `config`, `completion`, `version`, `help`, par défaut menu interactif
- **Parser .env robuste** : support `export`, guillemets simples/doubles, commentaires, lignes vides
- **Isolation par liste blanche** : ~30 variables système préservées, tout le reste nettoyé
- **Whitelist des commandes** : `claude`, `codex`, `opencode` uniquement (sauf `-AllowCustomCommand`)
- **Mode non-interactif** : `--json` pour `list` et `launch`, `--dry-run`, `--no-launch`, `--show-env`
- **Credential store** : interface `Store` avec implémentations Windows/macOS/Linux + fallback AES-256-GCM
- **Profils YAML** : support `.yaml` en complément du `.env`, avec validation de schéma
- **Profils par projet** : `.multiai.yaml` avec héritage (`extends`) et surcharges (`overrides`)
- **Plugin hooks** : `before_launch` (bloquant) et `after_launch` (best-effort)
- **Shell completion** : bash, zsh, fish, PowerShell (via `multiai completion <shell>`)
- **Site VitePress** : 16 pages (guide, référence, avancé)
- **CI/CD** GitHub Actions : lint → test (6 OS × Go) → security → build → benchmark
- **Packaging** : Homebrew, Scoop, DEB (Ubuntu/Debian), AUR (Arch Linux), npm wrapper
- **Script d'installation universel** : `curl -fsSL .../install.sh | bash`
- **`.editorconfig`**, **`.golangci.yml`**, **`.goreleaser.yml`**

### Added — PowerShell Legacy
- **`Clear-RouterEnvironment`** : passage en liste blanche (30+ variables)
- **`SecureString`** : secrets marqués convertis après injection
- **Codes de sortie discriminants** : 0-4 documentés
- **Modes `-Json`** : `Show-Profiles` et `Show-EffectiveEnv`
- **Mode `-DryRun`** : simulation complète sans lancement
- **Logging erreurs** : `%LocalAppData%\multiai\error.log`
- **Vérification intégrité SHA256** : hash du routeur loggé
- **`Split-ArgsSimple`** : parseur avec guillemets (machine à états)
- **`Read-DotEnvFile`** : support `export` (format Unix standard)
- **`$KnownEnvVars`** : étendu à AWS, Azure, GitHub, GitLab, NPM, etc.
- **Configuration interactive** : statut `[OK]/[~~]/[--]`, URLs vers les consoles API
- **Tests Pester** : 21 tests (Test-IsPlaceholder, Read-DotEnvFile, Split-ArgsSimple)

### Added — Documentation
- `CHANGELOG.md`, `CONTRIBUTING.md`, `ROADMAP.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`
- Issue templates GitHub (bug, feature, profile)
- Section Troubleshooting dans README (11 erreurs courantes)

### Changed
- **Nommage** : `aicode` → `multiai` partout (COMMANDS.md 21×, code-router.ps1, install.sh)
- **Architecture** : Go (primaire), PowerShell (legacy maintenu)
- **Version** : 0.1.5 → 0.2.0

### Fixed
- `.gitignore` : `*.env` exclus (risque critique de commit de clés API)
- `GEMINI.md` supprimé (Gemini retiré du projet)
- Messages post-install : `aicode.sh` → `multiai.sh`
- `prepublishOnly` : scan anti-fuite de clés avant publication npm

### Removed
- `GEMINI.md` (copie identique de CLAUDE.md, Gemini retiré)

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
- Clés API en clair dans les `.env` (chiffrement prévu)
- Pas d'exclusion `.env` dans `.gitignore`
- Documentation `COMMANDS.md` en ancien nom `aicode`

---

## [0.1.0] — 2026-06-20

### Added
- Release initiale : Claude Code, Codex CLI, OpenCode
- Profils `.env` basiques
- Menu interactif
- Installeur Windows (`install.ps1`)
- Distribution npm
