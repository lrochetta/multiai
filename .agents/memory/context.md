---
title: Project Context
description: Living state — updated by Zecher
created: "2026-06-23"
project: "multiai"
last_updated: "2026-07-15"
version: "0.6.10"
score: "5.8/10"
status: "0.6.10 STABLE — GITHUB/NPM LATEST — NORMAL REFS SANITIZED; GITHUB SUPPORT PENDING"
---

# Project Context — multiai v0.6.10 stable

## Point de reprise sécurité/release — 2026-07-15 01h (autoritatif)

- **Canaux publics sûrs** : GitHub `latest` et npm `latest` restent en 0.6.6. v0.6.7 et v0.6.8 sont des prereleases; npm 0.6.8 est uniquement sous `next`.
- **Défaut restant de 0.6.8** : les timeouts synchrones Node ne bornent pas un `CreateProcess` retenu par Avast. Le postinstall isolé pouvait donc encore geler avant que Node ne puisse appliquer son timer.
- **Correctif 0.6.10 local** : contrôleur Windows externe à deux processus pour le postinstall et les probes `version`; timeout 124 explicite, fallback lorsque `taskkill` est refusé, aucune limite ajoutée aux commandes interactives. Le bootstrap borne aussi hôtes, redirections et tailles de téléchargement.
- **Preuves boîte noire** : faux EXE dormant une heure interrompu en environ 2,9 s pour une limite de 2 s, sans PID résiduel; 40/40 tests npm verts; tarball dry-run de 9 fichiers contenant le module et le contrôleur; scan secrets et synchronisation workflows verts.
- **Go local** : tous les paquets `cmd`, `internal` et `pkg` passent. Le paquet d'intégration `tests` compile mais son lancement est retenu localement par Avast avant le runtime; la validation d'exécution reste à fournir par GitHub Actions.
- **Distribution 0.6.10** : master nettoyé `4851c2e`; CI finale `29378761550` et release `29378795691` vertes, AUR ignoré. À la demande explicite de Laurent, GitHub v0.6.10 est la release stable `latest` non-prerelease avec 11 assets et npm `latest`/`next` pointent sur `multiai@0.6.10`; npm 0.6.8 est dépréciée et 0.6.9 n'existe pas sur npm.
- **Qualification Windows** : tarball npm public et asset Windows final vérifiés par SHA-256. Une première probe Avast a échoué proprement avec `(no version)`, puis le même asset a réussi après acquisition de réputation : postinstall vert, `multiai 0.6.10` en 4,7 s et aucun nouveau processus résiduel. Après promotion, le smoke public `multiai@latest version` répond 0.6.10 et Laurent a validé une installation globale réelle : checksum officiel, postinstall, probe 0.6.10 et menu interactif sans freeze. Poursuivre néanmoins les essais sur plusieurs PC vierges/AV.
- **Audit credentials** : la valeur DeepSeek révoquée était encore présente dans trois blobs du HEAD, pas seulement dans l'historique. `git-filter-repo` a réécrit 80 commits; branches et tags normaux du dépôt principal et des deux forks publics sont nettoyés, exception Gitleaks supprimée, CI Gitleaks verte et clones frais à empreinte zéro. Résidu public : l'ancien SHA reste résolvable via les refs internes des PR fermées #3/#5/#6 ou le cache du réseau de forks; une purge GitHub Support est requise pour l'éradication côté hébergeur.
- **Nettoyage local** : les 30 worktrees temporaires ont été sauvegardés sous forme de patches/fichiers texte assainis dans `D:\tmp\multiai-worktree-recovery-20260715-024435`, puis supprimés. Le dépôt principal local est réaligné sur `4851c2e`, ses reflogs/objets antérieurs ont été purgés et un miroir assaini séparé conserve les 31 historiques de branches locales.
- **Auth release** : utiliser seulement le pointeur du coffre partagé hors dépôt et injecter le PAT en mémoire; ne jamais copier ni journaliser sa valeur.

## Point de reprise P0 Avast/CyberCapture — 2026-07-14 23h (archive)

- **Incident confirmé** : l'asset Windows 0.6.7 et les nouveaux exécutables locaux sont retenus par Avast CyberCapture dans `CreateProcess`, avant le runtime Go et `main`.
- **Rollback effectué** : npm `latest` pointe de nouveau sur 0.6.6; l'installation globale locale est 0.6.6 et `multiai version` répond. La release GitHub 0.6.7 est marquée prerelease.
- **0.6.7 dépréciée** : avertissement npm complet publié après validation 2FA.
- **Hotfix local 0.6.8** : Go release piné à 1.25.12 (correctif GO-2026-5856), `os.Exit(0)` restauré pour version/help, smoke postinstall 20s, timeout shim uniquement sur probes version, E2E et test Windows bornés autour de `Run`.
- **Validation** : 32/32 tests npm verts, scan secrets vert, workflows synchronisés, diff check vert. Le test Windows anti-gel échoue proprement en 10s sous Avast local, ce qui confirme le diagnostic.
- **Gate** : commit/push et CI sont autorisés. Après CI verte, v0.6.8 reste une GitHub prerelease et npm reste sous `next` pour qualifier les hashes exacts. Aucune promotion stable/`latest` avant essai Avast/CyberCapture ou whitelisting.
- **Stable publique** : 0.6.6. Ne pas promouvoir 0.6.7.
- **Auth release** : le PAT GitHub partagé est référencé uniquement par le coffre hors dépôt. Il a été utilisé en mémoire pour pousser `079019c`, sans valeur copiée dans le projet ni persistance dans `gh`.
- **CI `079019c`** : tous les jobs fonctionnels et multi-OS sont verts; seul govulncheck a refusé Go 1.25.11 pour GO-2026-5856. Reprise sur Go 1.25.12 avant toute release.

## Point de reprise Nexus — 2026-07-14 (autoritatif)

- **Audit complet** : Atlas, Forge et Sentinel ont travaillé en parallèle. Index et synthèse dans `audit/2026-07-14-bmad-plus-complete/`.
- **Description cible** : multiai est le plan de contrôle local pour Claude Code, Codex CLI et OpenCode — profils reproductibles, secrets isolés, sans proxy LLM ni lock-in.
- **Score consolidé** : produit 6,1 ; architecture 5,9 ; qualité/sécurité 5,4 ; maturité moyenne **5,8/10**. Décision **NO-GO v0.6.7**.
- **PATH Windows corrigé localement** : le parcours explicite `npx --yes --allow-scripts=multiai multiai@latest install` résout le préfixe npm, persiste le PATH utilisateur sans `setx` ni admin, refuse UNC/device et détecte le premier shim réel.
- **Preuves Windows** : 25/25 tests npm, syntaxe Node, tarball dry-run avec les deux helpers, préfixes espaces/Unicode, idempotence et conflit de shim validés. Aucun vrai PATH utilisateur n'a été muté pendant l'audit.
- **Gate Windows restante** : installer le tarball exact en mode `Apply` dans une VM Windows vierge, fermer l'installateur, ouvrir de nouvelles consoles cmd/PowerShell et vérifier le shim par son nom.
- **Quatre bloqueurs sécurité/release** : confiance implicite de `.multiai.yaml`, traversal du registre, updater non persistant/fail-open, workflow de release racine divergent et non suffisamment gated.
- **Contrats produit prioritaires** : YAML/projet/hooks, CLI/JSON/timeout, versions, canaux de distribution et documentation doivent être alignés sur le code réellement livré.
- **Validation locale** : `go vet ./...` et tests ciblés profile/registry/update/cli verts ; suite `go test ./...` et test `cmd/multiai` inconclusifs par timeout Windows. Un timeout ne vaut pas succès.
- **État public** : GitHub/npm restent en 0.6.6 ; package local en 0.6.7. Aucun commit, tag, release ou publish n'a été créé durant l'audit.
- **Ordre de reprise** : fermer la roadmap P0, borner les tests de sous-processus, obtenir la matrice CI complète verte, exécuter l'E2E Windows, puis seulement préparer une release signée et vérifiée.
- **Worktree** : préserver les modifications préexistantes, notamment `multiai-go/cmd/multiai/main_test.go` et les changements mémoire/docs déjà présents avant l'audit.

## Archive Zecher — 2026-07-13

- **Publication suspendue à la demande de Laurent** : reprendre plus tard; ne créer aucun tag et ne lancer aucun `npm publish` avant reprise explicite.
- **État public** : GitHub/npm restent en `0.6.6`. La cible `0.6.7` n'est pas publiée.
- **Git** : `master` et `origin/master` pointent sur `5808769`.
- **Commits déjà poussés** : `141120b` (installation npm/npx), `22fdf23` (lint), `29180f4` (assertions de signatures), `5808769` (CI cross-platform et Go 1.26.5).
- **CI** : run `29213384824` terminé en échec. Verts : lint, tests Windows, sécurité/govulncheck, GoReleaser check et six cross-compilations. Rouges : tests macOS et Ubuntu; build, smoke et benchmark ignorés en aval.
- **Worktree de reprise** : `multiai-go/cmd/multiai/main_test.go` contient déjà une assertion d'erreur indépendante de la langue, non commitée. La conserver.
- **Blocages restants documentés** : assertions localisées macOS, isolation du Keychain entre tests, migration à tester avec le backend fichier, et faux `secret-tool` qui ne simule pas encore l'erreur `search`.
- **Validation locale** : les exécutables de test Go peuvent rester bloqués avant `main` sur ce poste Windows; utiliser les runners GitHub comme validation cross-platform définitive.
- **Ordre de reprise** : lire `.agents/memory/sessions/2026-07-13-v0.6.7-release-blocked.md`, finir les quatre correctifs de tests, commit/push, attendre toute la CI verte, puis seulement tag `v0.6.7`, release GitHub et publication npm avec 2FA.

## Archive précédente — 2026-07-12 (avant les commits)

- **Version publique actuelle** : npm + GitHub `0.6.6`.
- **Incident confirmé** : `postinstall` n'utilisait pas la CA système Node 24; `npx multiai install` atteignait ensuite une commande Go inexistante; un lancement frais sans TTY pouvait boucler sur EOF.
- **Correctif local** : `0.6.7` préparée, tests Node 12/12, tests Go ciblés verts, installation réelle depuis tarball/cache vierge validée sur Windows sans `--use-system-ca`.
- **Publication** : non effectuée. Il faut commit → tag `v0.6.7` → attendre les artefacts GitHub → `npm publish` manuel (2FA).
- **Limitation de validation locale** : certains `.exe` Go fraîchement compilés restent parfois bloqués avant `main` dans l'environnement Windows; la suite complète peut expirer, tandis que les packages modifiés passent séparément.

## Project Identity

- **Name** : multiai
- **Version** : **0.6.6 publiée**, **0.6.7 préparée**
- **Path** : `D:\travail\DEV\multiai`
- **Go module** : `github.com/lrochetta/multiai` (multiai-go/)
- **npm** : `multiai@0.6.6` publié; `0.6.7` en attente de release
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
| 2026-07-12 | v0.6.6 publiée après plusieurs correctifs install/release, mais depuis un worktree sale |
| 2026-07-12 | v0.6.7 préparée : CA système, contrat npx install, EOF, tests/preflights |

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

## Build Status historique (v0.4.x — non autoritatif)

```
✅ go vet: 0 warning
✅ go build: OK
✅ go test: 13/13 packages OK
✅ govulncheck: 0 vulnérabilité
✅ npm publish: v0.4.2
✅ GitHub Release: v0.4.2 (GoReleaser + Cosign)
✅ Branch protection: 6 status checks
```

## Release v0.6.7 (2026-07-14)

- Tag annoté `v0.6.7` sur `4180aa4` (signature GPG indisponible sur la machine de publication).
- CI et workflow Release verts : race tests multi-OS, lint, gosec, Gitleaks, govulncheck, cross-builds et smoke test.
- GitHub Release publique avec archives Windows/macOS/Linux, paquets DEB, checksums SHA-256, bundle Cosign, provenance et SBOM CycloneDX.
- `multiai@0.6.7` publié sur npm et installation isolée validée ; le binaire npm est identique à l'asset Windows de la release.
- Métadonnées AUR `0.6.7` et checksum source canonique enregistrés dans le dépôt. Le push AUR externe reste désactivé tant que `AUR_SSH_KEY` et `AUR_KNOWN_HOSTS` ne sont pas configurés.
- Le lancement local des exécutables Go reste affecté par le blocage Windows global `CreateProcessWithLogonW failed: 2`; les runners GitHub confirment le fonctionnement du binaire.

## Open Questions

- [ ] Credential stores natifs OS (Windows Credential Manager, macOS Keychain, libsecret)
- [ ] Homebrew tap + Scoop bucket (skip_upload → false)
- [ ] Cost tracking réel (parsing stderr des CLI enfants)
- [ ] i18n anglais pour marché global
- [ ] Site VitePress déployé
