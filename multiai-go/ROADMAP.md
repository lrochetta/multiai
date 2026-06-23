# Roadmap — multiai

## Vision
Devenir le routeur de reference pour les developpeurs utilisant plusieurs CLI d'IA
(Claude Code, Codex CLI, OpenCode) avec une gestion unifiee et securisee des profils.

---

## v0.2.0 (en cours) — Reecriture Go

- [x] Binaire unique multi-plateforme (Windows, macOS, Linux — amd64, arm64)
- [x] Parser .env robuste (support export, guillemets, commentaires)
- [x] Isolation d'environnement par liste blanche
- [x] Whitelist des commandes executables
- [x] Mode non-interactif (--json, --dry-run, --no-launch)
- [x] Configuration interactive des cles API (5 fournisseurs)
- [x] Tests unitaires (dotenv, profile, env)
- [x] CI/CD GitHub Actions (lint, test 6x, security, build)
- [x] Cross-compilation (goreleaser)
- [x] Shell completion (bash, zsh, fish, PowerShell)
- [ ] Validation du build Go (go build ./...)
- [ ] Tests d'integration complets
- [ ] Migration automatique depuis l'ancienne version PowerShell

## v0.3.0 — Securisation

- [ ] Chiffrement au repos (AES-256-GCM)
- [ ] Credential store natif (Windows Credential Manager, macOS Keychain, libsecret Linux)
- [ ] Signature Cosign des binaires
- [ ] SBOM (Syft)
- [ ] Audit de securite externe

## v0.4.0 — Ecosysteme

- [ ] Profils YAML (en plus du .env)
- [ ] Profils par projet (.multiai.yaml local)
- [ ] Registre communautaire de profils
- [ ] Plugin hooks (before/after launch)
- [ ] Site de documentation (VitePress ou Docusaurus)

## v0.5.0 — Distribution

- [ ] Homebrew (brew install lrochetta/tap/multiai)
- [ ] Scoop (scoop install multiai)
- [ ] APT (Ubuntu/Debian)
- [ ] AUR (Arch Linux)
- [ ] npm (wrapper de telechargement du binaire)

## v1.0.0 — Production

- [ ] >=90% test coverage
- [ ] 0 vulnerabilite connue (govulncheck)
- [ ] 0 warning golangci-lint
- [ ] Documentation complete (site + help integre)
- [ ] Programme de feedback (Discord/Discussions)
