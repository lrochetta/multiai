---
title: Project Context
description: Living state — updated by Zecher
created: "2026-06-23"
project: "multiai"
last_updated: "2026-07-06T01:30:00Z"
version: "0.4.3"
score: "8.5/10"
status: "PRODUCTION"
---

# Project Context — multiai v0.4.3

## Project Identity

- **Name** : multiai
- **Version** : **0.4.2** (npm + Go)
- **Path** : `D:\travail\DEV\multiai`
- **Go module** : `github.com/lrochetta/multiai` (multiai-go/)
- **npm** : `multiai@0.4.3` (publié, binaire Go natif)
- **Stack** : Go 1.22 (référence) + PowerShell 5.1+ (gelé/archivé)
- **Score** : **8.5/10** (audit BMAD+ 3 agents, 2026-07-05)

## Chronologie

| Date | Événement |
|------|----------|
| 2026-06-23 | BMAD+ init, audit v0.1.5, plan 10/10, 42 fixes, npm v0.2.6 |
| 2026-06-24 | v0.3.0 : 8 providers, régions, fallback, Requesty, cost logging |
| 2026-07-05 21:00 | Audit BMAD+ 3 agents (Atlas/Forge/Sentinel) → 8.5/10 |
| 2026-07-05 21:30 | 4 correctifs sécurité (hooks injection, LICENSE, credential store, CI) |
| 2026-07-05 22:00 | Repo public, audit sécurité (secrets + supply chain), branch protection |
| 2026-07-05 22:30 | Release v0.4.0 (GoReleaser + Cosign, 10 assets multi-OS) |
| 2026-07-05 22:45 | npm switch PS→Go : `multiai@0.4.0` binaire natif |
| 2026-07-05 23:00 | Menu config coloré (vert/jaune/gris) + erase key depuis configureProvider |
| 2026-07-05 23:30 | v0.4.1 : README corrigé, 37 profils, audit public |
| 2026-07-06 00:00 | Menu profils coloré (même système que config) |
| 2026-07-06 00:30 | v0.4.2 : README final, npm publié |
| 2026-07-06 01:00 | Auto-update : `internal/update/`, check GitHub Releases au lancement |
| 2026-07-06 01:30 | v0.4.3 : npm publié avec auto-update |

## v0.4.x — Nouveautés (2026-07-05/06)

### Passage en production Go
- **npm switch** : le package npm distribue le binaire Go natif (plus PowerShell)
- **SHA256 vérifié** : install.js télécharge depuis GitHub Releases avec vérification
- **Cosign keyless** : signatures des releases vérifiables

### Audit BMAD+ complet
- **3 agents parallèles** (Atlas stratégie, Forge archi, Sentinel sécurité)
- **6 rapports** dans `audit/` (stratégie, architecture, sécurité, secrets, supply-chain, synthèse)
- **Score global : 8.5/10** — 0 vulnérabilité critique

### Correctifs de sécurité
- Injection `os.ExpandEnv` après `escapeShellArg` dans hooks → ordre inversé
- LICENSE MIT ajouté (était absent)
- Credential store obligatoire — plus de fallback silencieux en texte clair
- CI/CD : `.golangci.yml` v2, golangci-lint activé, smoketest

### Auto-update (v0.4.3)
- **Check au lancement** : `update.Check(version)` dans `main()`
- GitHub Releases API, cache 1h, timeout 5s, silent fail
- Télécharge, vérifie SHA256, extrait, re-exec le nouveau binaire
- `MULTIAI_SKIP_UPDATE=1` pour désactiver

### UX améliorée
- **Menus colorés** : vert (configuré), jaune (partiel), gris (non configuré)
- Menu config : lignes colorées + option `e` pour effacer une clé
- Menu profils : lignes colorées selon clés configurées

### Repo GitHub
- Public, branch master protégée (6 status checks obligatoires)
- Workflows CI/CD synchronisés à la racine
- Historique git nettoyé (0 Co-Authored-By)

## Architecture (v0.4.2)

```
multiai-go/                  → Go v0.4.2 (référence)
├── cmd/multiai/             → CLI 7 sous-commandes
├── internal/
│   ├── assets/              → 37 profils .env embarqués
│   ├── catalog/             → 13 fournisseurs (data-driven YAML)
│   ├── cli/                 → Launcher, fallback, hooks, display, completion
│   ├── config/              → Wizard interactif + erase keys
│   ├── env/                 → Isolation + expansion %VAR%
│   ├── logging/             → Journal de sessions JSONL
│   ├── menu/                → Menus interactifs colorés
│   ├── onboarding/          → Assistant premier démarrage
│   ├── openrouter/          → Client API, cache, search, compare, profilegen
│   ├── profile/             → Chargement .env, YAML, projet
│   └── secret/              → AES-256-GCM + credential store
└── pkg/dotenv/              → Parser .env robuste
```

## Build Status

```
✅ go vet: 0 warning
✅ go build: OK
✅ go test: 13/13 packages OK
✅ govulncheck: 0 vulnérabilité
✅ npm publish: v0.4.2
✅ GitHub Release: v0.4.2 (GoReleaser + Cosign)
✅ Branch protection: 6 status checks
```

## Open Questions

- [ ] Credential stores natifs OS (Windows Credential Manager, macOS Keychain, libsecret)
- [ ] Homebrew tap + Scoop bucket (skip_upload → false)
- [ ] Cost tracking réel (parsing stderr des CLI enfants)
- [ ] i18n anglais pour marché global
- [ ] Site VitePress déployé
