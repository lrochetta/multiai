# Roadmap — multiai

## Vision
Devenir le routeur de reference pour les developpeurs utilisant plusieurs CLI d'IA
(Claude Code, Codex CLI, OpenCode) avec une gestion unifiee et securisee des profils.

---

## v0.4.3 (livré) — Menus colores + Auto-update

- [x] Menu profils colores (vert/jaune/gris selon etat des cles)
- [x] Menu config colores (statut [OK]/[~~]/[--])
- [x] Auto-update : verification GitHub Releases au demarrage
- [x] Cache 1h pour les verifications de mise a jour
- [x] Re-exec du binaire apres telechargement
- [x] Audit securite 3 agents BMAD+, score 8.5/10
- [x] 4 correctifs securite (credential store, sentinel, flux config→launch)
- [x] Scan secrets + supply chain pour repo public

## v0.5.0 (livré) — Securisation + Architecture + Tests + DX

### 🔴 Sprint 1 — BLOCKERS (6 stories)
- [x] Rotation cle DeepSeek compromise + suppression fichier sensible
- [x] URL GitHub hardcodee dans l'auto-update (`MULTIAI_DEV=1` requis pour override)
- [x] Verification Cosign dans l'auto-update (`.sig` + `.pem`, OIDC)
- [x] Gate CI release (lint + test + vet obligatoires, verification ascendance master)
- [x] Gitleaks : `.gitleaks.toml`, job CI, pre-commit hook
- [x] Smoke CI repare (suppression `cd multiai-go` redondant)

### 🟠 Sprint 2 — HIGH (8 stories)
- [x] Tests `cmd/multiai` (0% → 50%+ couverture)
- [x] YAML + hooks cables en production (`LoadAllProfiles`)
- [x] `--store` gere explicitement (message clair, fallback AES-256-GCM)
- [x] `ServiceForProfile` corrige (hash SHA256 du chemin canonique)
- [x] Transactionalite store/sentinelle (verrou inter-processus, rollback)
- [x] `ensureProfiles` versionne (manifeste `profiles.json`, extraction selective)
- [x] `display/` extrait de `cli/` (couplage metier→affichage casse)
- [x] Ecritures atomiques unifiees vers `fsutil.WriteFileAtomic`

### 🟡 Sprint 3 — MEDIUM (7 stories)
- [x] Site VitePress : 15 pages deployees sur GitHub Pages
- [x] Tests `fsutil`, `menu`, `logging` packages zero
- [x] `multiai update` explicite (sous-commande, `--check`, `--yes`)
- [x] Homebrew tap + Scoop bucket actives
- [x] SBOM CycloneDX dans les releases
- [x] `CONTRIBUTING.md` + templates GitHub (bug, feature, PR)
- [x] `context.Context` propage dans les appels HTTP

### 🟢 Sprint 4 — LOW (5 stories)
- [x] i18n FR/EN : framework `internal/i18n/`, 66 messages
- [x] Fuzz testing : 3 fuzzers (.env, YAML, profile), 0 crash
- [x] Migration Go 1.24 + `gopkg.in/yaml.v3` → `github.com/yaml/go-yaml`
- [x] Wrappers cross-platform : 37 scripts `.cmd`/`.sh` par shortcut
- [x] Visibilite : badges, Show HN, Reddit, newsletters

---

## v0.6.0 — Ecosysteme & Distribution

- [ ] Store natif OS (Windows Credential Manager, macOS Keychain, libsecret Linux)
- [ ] Registre communautaire de profils
- [ ] APT (Ubuntu/Debian)
- [ ] AUR (Arch Linux) — SHA256 verifies
- [ ] `multiai config --store wincred|keychain|secret-service` implemente
- [ ] Zeroisation memoire complete des secrets
- [ ] Timeout/context sur processus enfants
- [ ] Whitelist env case-insensitive sous Windows
- [ ] Tests d'integration complets (E2E)
- [ ] Migration automatique depuis l'ancienne version PowerShell

## v1.0.0 — Production

- [ ] >=90% test coverage
- [ ] 0 vulnerabilite connue (govulncheck)
- [ ] 0 warning golangci-lint
- [ ] Documentation complete (site + help integre)
- [ ] Programme de feedback (Discord/Discussions)
- [ ] 500+ stars GitHub
- [ ] 10+ contributeurs externes
- [ ] 3500+ telechargements cumules
