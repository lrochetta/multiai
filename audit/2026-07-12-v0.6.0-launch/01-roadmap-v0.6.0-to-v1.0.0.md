# Plan de convergence — v0.6.0 → v1.0.0

> **Nexus (Orchestrator) — Synthèse 5 agents BMAD+**
> Généré le 2026-07-12 — Sprint S9/S10/M1-M3

---

## Resume executif

**31 stories** consolidees depuis 5 agents (Forge, Sentinel, Atlas, Huldah+Bezalel, Nexus) en **25 stories uniques** apres deduplication des overlaps.

| Metrique | Valeur |
|----------|--------|
| Stories totales (brutes) | 31 |
| Stories apres fusion | 25 |
| Sprints | 5 (A → E) |
| Charge estimee totale | ~45 JH |
| Chemin critique | 12 stories (BLOCKER/CRITICAL) |
| Deadline v0.6.0 | J+7 (19 juillet 2026) |
| Deadline v1.0.0 | J+30 (11 aout 2026) |

### Decoupages et fusions effectuees

| Stories fusionnees | Raison |
|--------------------|--------|
| S9.3 (Forge) + S10.3 (Sentinel) | Qualite CI: golangci-lint bloquant — meme objectif |
| S9.2 (Forge) + S10.5 (Sentinel) | Tests stores natifs + cross-platform — memes fichiers |
| M3.1 (Nexus) + Huldah Story 3 | Quickstart + VitePress stores/registry/migration |
| M3.2 (Nexus) + Huldah Story 4 | Tutoriel video "Zero a Hero" — contenu identique a 80% |
| M1.0 (Nexus) + Atlas S1 | Campagne marketing + Lancement PH/HN/Reddit |
| Huldah Story 5 + Atlas S5 | CONTRIBUTING.md v2 + Contributor ladder — docs+process |

---

## Sprint A — CI Foundation (J0 → J+5)

**Objectif :** CI stable, tests cross-platform, quality gates bloquants.

| ID | Titre | Priorite | JH | Depend de | Agent |
|----|-------|----------|----|-----------|-------|
| **A1** | Fix CI: gofmt + compilation + i18n | **BLOCKER** | 0.5 | — | Forge |
| **A2** | Quality gates: govulncheck + golangci-lint + gosec bloquants | **HIGH** | 1.5 | A1 | Forge + Sentinel |
| **A3** | Tests stores natifs + cross-platform CI | **HIGH** | 2 | A1 | Forge + Sentinel |
| **A4** | 0 vulnerabilite govulncheck (audit stdlib + dependances) | **CRITICAL** | 1 | — | Sentinel |

**Total Sprint A : 5 JH**

### Details des fusions

**A2** = S9.3 (Forge: rendre gosec/golangci-lint/govulncheck bloquants) + S10.3 (Sentinel: 0 warning, reactiver errcheck, gocyclo, misspell). Output: `.golangci.yml` avec linters complets, `|| true` supprime, `ci-check` Makefile target.

**A3** = S9.2 (Forge: tests WinCred + macOS Keychain + libsecret en CI) + S10.5 (Sentinel: audit build tags, tests OS-specifiques, gitleaks). Output: job `test-native-stores` dans CI, `store_windows_parser.go` sans build tag, tests `parseDumpKeychain` macOS.

### Livrables Sprint A
- [ ] CI verte sur lint + test + build (3 jobs, 5 OS)
- [ ] `gofmt -l .` vide dans CI et en local
- [ ] `golangci-lint run ./...` exit code 1 = rouge
- [ ] `govulncheck` integre dans lint job
- [ ] `gosec` sans `|| true`
- [ ] Tests WinCred parsing sur tous les OS
- [ ] Tests macOS Keychain parsing sur macos-latest
- [ ] Job `test-native-stores` avec libsecret-tools installe
- [ ] `store_windows_parser.go` extrait (sans build tag)
- [ ] `TestI18nKeysExist` cree et passe
- [ ] Pre-commit hook gofmt documente
- [ ] Aucune CVE connue (govulncheck passe)

---

## Sprint B — Release v0.6.0 (J+5 → J+10)

**Objectif :** Publier v0.6.0 avec ecosysteme complet (registre, distribution, CHANGELOG).

| ID | Titre | Priorite | JH | Depend de | Agent |
|----|-------|----------|----|-----------|-------|
| **B1** | Depot registre communautaire + CI validation profils | **BLOCKER** | 2 | — | Nexus |
| **B2** | CHANGELOG automation (git-chglog + commits conventionnels) | **HIGH** | 2 | — | Huldah |
| **B3** | Homebrew/Scoop: TAP_GITHUB_TOKEN + .goreleaser.yaml | **MEDIUM** | 1 | — | Forge |
| **B4** | Discord server (setup AVANT lancement) | **HIGH** | 1 | — | Atlas |
| **B5** | Release v0.6.0: tag, GH Release, npm publish, artifacts | **BLOCKER** | 1.5 | A1, A2, B1, B2 | Forge |

**Total Sprint B : 7.5 JH** (parallele possible: B1, B2, B3, B4 en parallele)

### Dependances
- B5 (Release) bloque par A1 + A2 (CI verte)
- B1 (Registre) independant mais prerequis pour lancer la commu
- B4 (Discord) doit etre fait AVANT le lancement (Sprint E)

### Livrables Sprint B
- [ ] `github.com/lrochetta/profiles-multiai` cree avec 12 profils seed
- [ ] `index.json` genere automatiquement + GitHub Pages
- [ ] Workflow `validate.yml` (gitleaks, en-tetes, securite, doublons)
- [ ] `git-chglog` configure (`.chglog/config.yml`, template)
- [ ] `scripts/generate-changelog.sh` operationnel
- [ ] `release.yml` genere CHANGELOG automatiquement
- [ ] Repos `homebrew-tap` et `scoop-bucket` crees
- [ ] `TAP_GITHUB_TOKEN` configure dans secrets
- [ ] `brews` et `scoops` decommentes dans `.goreleaser.yaml`
- [ ] Serveur Discord cree (5 canaux: #general, #help, #profiles, #dev, #releases)
- [ ] **git tag v0.6.0 pushe sur master**
- [ ] GitHub Release avec 8 archives + checksums + Cosign + SBOM
- [ ] APT repo mis a jour sur gh-pages
- [ ] AUR PKGBUILD mis a jour
- [ ] `npm publish` avec binaire Go v0.6.0
- [ ] `multiai version` retourne `0.6.0`

---

## Sprint C — Documentation v1.0.0 (J+10 → J+18)

**Objectif :** Documentation complete pour v1.0.0 (architecture, API, guides, VitePress).

| ID | Titre | Priorite | JH | Depend de | Agent |
|----|-------|----------|----|-----------|-------|
| **C1** | ADR (7 decisions) + Diagrammes C4 model (PlantUML) | **BLOCKER** | 3 | — | Huldah |
| **C2** | Documentation API (godoc, doc.go, exemples, pkg.go.dev) | **BLOCKER** | 2 | C1 (partiel) | Huldah |
| **C3** | Mise a jour VitePress: stores, registry, migration, Quickstart | **BLOCKER** | 3 | C1, C2 | Huldah + Bezalel |
| **C4** | CONTRIBUTING.md v2 + Contributor ladder (6 echelons) | **HIGH** | 3 | C3 | Huldah + Atlas |

**Total Sprint C : 11 JH**

### Details des fusions

**C3** = Huldah Story 3 (5 nouvelles pages VitePress: stores, migration, community, commands, installation) + M3.1 (Quickstart 5 minutes en 6 commandes). Output: 20+ pages VitePress, 0 dead link, sidebar complete.

**C4** = Huldah Story 5 (CONTRIBUTING.md 8 sections, templates, guide profils) + Atlas S5 (6 echelons: first issue → maintainer, good first issues, mentoring). Output: CONTRIBUTING.md v2 + `docs/contributing/` dossier + good first issues labels.

### Livrables Sprint C
- [ ] 7 ADR dans `docs/architecture/adr/ADR-001.md` a `ADR-007.md`
- [ ] Diagrammes PlantUML (System Context, Container, Component, Data Flow, Sequence)
- [ ] PNG generes et visibles dans VitePress
- [ ] `doc.go` dans chaque package interne (7 packages)
- [ ] 8-10 `ExampleXxx` fonctions testees
- [ ] `docs/api-surface.md` — inventaire complet
- [ ] Module indexe sur `pkg.go.dev/github.com/lrochetta/multiai`
- [ ] 20+ pages VitePress, build sans erreur
- [ ] Page stores natifs avec comparaison des 4 backends
- [ ] Page migration PowerShell
- [ ] Page registre communautaire
- [ ] Page Quickstart 5 minutes
- [ ] COMMANDS.md mis a jour (--store, --timeout, profile, migrate, update)
- [ ] INSTALLATION.md mis a jour (APT, AUR, Cosign, SHA256)
- [ ] CONTRIBUTING.md v2 (8 sections)
- [ ] `docs/contributing/` dossier avec quickstart, profiles, docs
- [ ] 10 good first issues creees et labelisees
- [ ] Contributor ladder documentee (6 echelons)

---

## Sprint D — Qualite v1.0.0 (J+18 → J+25)

**Objectif :** Qualite industrielle: 90% coverage, 0 CVE, benchmarks, securite.

| ID | Titre | Priorite | JH | Depend de | Agent |
|----|-------|----------|----|-----------|-------|
| **D1** | Test coverage 4.9% → 90% (13 sous-stories par package) | **CRITICAL** | 5 | A1, A3 (CI stable) | Sentinel |
| **D2** | Performance benchmarks (2 → 15+, benchstat en CI) | **MEDIUM** | 1 | D1 | Sentinel |
| **D3** | Security audit final (6 vecteurs, pen-test, Cosign, score >=9.5) | **CRITICAL** | 3 | D1 | Sentinel |

**Total Sprint D : 9 JH**

### Risques
- D1 est le plus gros morceau : ~5 JH pour passer de 50% a 90% (en realite l'etat actuel est estime a 4.9% sur `cmd/multiai`, 0% sur `display` et `i18n`). **Accepte de descendre a 75%** si certains packages sont trop complexes a tester (interfaces systeme).
- D3 active Cosign dans `.goreleaser.yaml` — verifier que le workflow `attest-build-provenance` n'est pas en conflit.

### Livrables Sprint D
- [ ] Coverage >= 75% sur tous les packages (minimum 90% sur `internal/secret/`, `internal/i18n/`)
- [ ] 15+ benchmarks reproductibles
- [ ] `benchstat` integration CI (regression detection)
- [ ] 0 vulnerabilite connue (govulncheck + audit stdlib)
- [ ] 6 exclusions gosec analysees et documentees
- [ ] Pen-test sur 6 vecteurs: aucun critique trouve
- [ ] Cosign active dans `release.yml`
- [ ] Score securite >= 9.0/10
- [ ] `internal/security/vuln.go` cree pour exemptions documentees
- [ ] `SECURITY.md` avec politique de divulgation 90 jours
- [ ] Cosign verification documentee dans installation.md

---

## Sprint E — Community Launch (J+18 → J+45)

**Objectif :** 500+ stars, 10+ contributeurs, ecosysteme actif. Demarrage parallele a Sprint C/D.

| ID | Titre | Priorite | JH | Depend de | Agent |
|----|-------|----------|----|-----------|-------|
| **E1** | Campagne lancement: Product Hunt + Show HN + Reddit + newsletters | **CRITICAL** | 4 | B4, B5, B1 | Atlas + Nexus |
| **E2** | Tutoriel video "De Zero a Hero" (script + storyboard + production) | **HIGH** | 6 | C3 (VitePress pret) | Atlas + Huldah |
| **E3** | Content marketing: 6 articles blog (calendrier editorial) | **HIGH** | 5 | E1 | Atlas |
| **E4** | Programme ambassadeur: 10 createurs, kit, outreach 4 phases | **MEDIUM** | 3 | E1 | Atlas |
| **E5** | Partnerships: OpenCode, Anthropic, OpenRouter, providers | **LOW** | 3 | E1, E3 | Atlas |
| **E6** | Case studies: temoignages, benchmarks, equipes | **LOW** | 2 | E1, E3 | Atlas |
| **E7** | KPI dashboard public + badges README | **LOW** | 0.5 | — | Atlas |

**Total Sprint E : 23.5 JH** (execution fortement parallele)

### Details des fusions

**E1** = M1.0 (Calendrier J-7 a J+30: Show HN, 4 Reddit, 3 newsletters, Twitter, dev.to) + Atlas S1 (Product Hunt + Show HN + Reddit). Timeline unique: J0=Show HN, J+1=Product Hunt, J+2/J+3=Reddit, J+3/5/7=newsletters, J+7=Twitter thread, J+14=dev.to post-mortem.

**E2** = M3.2 (Video demo 2-3 min, script 5 scenes) + Huldah Story 4 (Tutoriel 10-15 min, 6 segments, storyboard). Resolution: produire les DEUX versions — une demo courte (3 min) pour le lancement, un tutoriel complet (12 min) pour la doc.

### Timeline E1 (Lancement)

| J | Action | Canal | Cible |
|---|--------|-------|-------|
| J0 | Show HN + Reddit r/golang + Twitter/X | Hacker News, Reddit, X | Debut du buzz |
| J+1 | Product Hunt launch + Reddit r/programming | Product Hunt, Reddit | 100+ stars |
| J+2 | Reddit r/commandline + r/LocalLLaMA | Reddit | Niche communities |
| J+3 | Newsletter Go Weekly | Email | Devs Go |
| J+5 | Newsletter Console.dev | Email | Devs SaaS |
| J+7 | Newsletter TLDR + Twitter recap | Email, X | General devs |
| J+14 | Post-mortem sur dev.to | dev.to | Retention |
| J+30 | Bilan KPI + Ajustements | Tous canaux | Iteration |

### Livrables Sprint E
- [ ] Product Hunt launch (post + comments + replies)
- [ ] Show HN post avec commentaire technique
- [ ] 4 posts Reddit (r/golang, r/programming, r/commandline, r/LocalLLaMA)
- [ ] 3 newsletters (Go Weekly, Console.dev, TLDR)
- [ ] Thread Twitter/X multi-tweets
- [ ] Article dev.to post-mortem
- [ ] ~~500 stars GitHub~~ (cible a J+90, pas J+30)
- [ ] Video demo 3 min pour lancement
- [ ] Tutoriel 12 min avec storyboard
- [ ] Cheatsheet PDF dans `docs/tutorial/cheatsheet.md`
- [ ] 6 articles de blog publies (calendrier editorial)
- [ ] Kit ambassadeur (email, one-pager, media kit)
- [ ] 10 createurs identifies et contactes
- [ ] Outreach log `outreach-log.md`
- [ ] 4 partenariats actifs (OpenCode, Anthropic, OpenRouter, 1 provider)
- [ ] 5 temoignages publics collectes
- [ ] KPI dashboard dans README (badges: stars, contributors, downloads, coverage, govulncheck)

---

## Chemin critique

```
J0 ── A1 (Fix CI) ──┬── A2 (Quality gates) ──┐
                     ├── A3 (Tests natifs) ───┤
                     └── A4 (Vuln check) ─────┘
                                               │
J+5 ── B1 (Registre) ──┬── B5 (RELEASE v0.6.0) ──┐
       B2 (CHANGELOG) ──┘                          │
       B3 (Homebrew) ──── indépendant              │
       B4 (Discord) ─────→ requis par E1           │
                                                    │
J+10 ── C1 (ADR) ──→ C2 (Doc API) ──→ C3 (VitePress) ──→ C4 (CONTRIBUTING)
                                                    │
J+18 ── D1 (Coverage) ──→ D3 (Security audit) ────┤
        D2 (Benchmarks) ──→ indépendant            │
                                                    │
J+18 ── E1 (Launch) ──┬── E2 (Video) ──────────────┤
        E3 (Content) ──┤                            │
        E4 (Ambassadeur) ── indépendant             │
                                                     │
J+45 ── v1.0.0 RELEASE ──────────────────────────────┘
```

**Chemin critique absolu** (9 stories, ~16 JH):
```
A1(0.5) → A2(1.5) → B5(1.5) → C1(3) → C2(2) → C3(3) → D3(3) → E1(4) total: ~18j calendaires
```

**Point de bascule critique :** B5 (Release v0.6.0) a J+10. Si A1 ou B5 glisse, tout le planning derape. Mitigation: geler les features 48h avant la release.

---

## Diagramme de Gantt (vue synthetique)

```
Sprint A  [██████████████░░░░░░░░░░░░░░░░░░░░░░░░]  J0→J+5
  A1      [██░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░]  0.5j
  A2      [░░██████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░]  1.5j
  A3      [░░░░████████░░░░░░░░░░░░░░░░░░░░░░░░░░]  2j
  A4      [░░██░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░]  1j

Sprint B  [░░░░░░████████████████░░░░░░░░░░░░░░░░]  J+5→J+10
  B1      [░░░░░░████████░░░░░░░░░░░░░░░░░░░░░░░░]  2j
  B2      [░░░░░░░░████████░░░░░░░░░░░░░░░░░░░░░░]  2j
  B3      [░░░░░░░░░░████░░░░░░░░░░░░░░░░░░░░░░░░]  1j
  B4      [░░░░░░░░░░░░████░░░░░░░░░░░░░░░░░░░░░░]  1j
  B5      [░░░░░░░░░░░░░░██████░░░░░░░░░░░░░░░░░░]  1.5j

Sprint C  [░░░░░░░░░░░░░░░░██████████████████░░░░]  J+10→J+18
  C1      [░░░░░░░░░░░░░░░░████████████░░░░░░░░░░]  3j
  C2      [░░░░░░░░░░░░░░░░░░░░░░████████░░░░░░░░]  2j
  C3      [░░░░░░░░░░░░░░░░░░░░░░░░░░████████████]  3j
  C4      [░░░░░░░░░░░░░░░░░░░░░░░░░░░░██████████]  3j

Sprint D  [░░░░░░░░░░░░░░░░░░░░░░░░░░░░██████████]  J+18→J+25
  D1      [░░░░░░░░░░░░░░░░░░░░░░░░░░░░██████████]  5j
  D2      [░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░████░░]  1j
  D3      [░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░████]  3j

Sprint E  [░░░░░░░░░░░░░░░░░░░░░░░░░░░░████████████████████████████████]  J+18→J+45
  E1      [░░░░░░░░░░░░░░░░░░░░░░░░░░░░████████████████░░░░░░░░░░░░░░░░]  4j
  E2      [░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░████████████████████░░]  6j
  E3      [░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░██████████████████]  5j
  E4      [░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░████████████░░]  3j
  E5      [░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░██████████]  3j
  E6      [░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░████████]  2j
  E7      [░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░██░░░░]  0.5j
```

---

## Risques et mitigations

| Risque | Prob. | Impact | Mitigation |
|--------|-------|--------|------------|
| CI cassee bloque tout (A1) | Haute | Critique | Branch freeze 48h pre-release, fix en hotfix branch |
| Coverage 90% irrealiste (D1) | Haute | Moyen | Accepter 75%, focus sur `secret/` et `i18n/` d'abord |
| TAP_GITHUB_TOKEN non configure | Moyenne | Bloquant B3, B5 | Creer PAT classic JOUR 1 avant tout |
| npm publish echoue (OTP manquant) | Moyenne | Bloquant B5 | Prevoir la presence de laurent a J+10 |
| Product Hunt down / HN rate limit | Faible | Moyen | Plan de repli: Reddit en priorite, news en backup |
| Aucun contributeur externe en J+30 | Moyenne | Haut (KPI rate) | Discord mentoring, good first issues, outreach direct |
| goreleaser v2 API change (brews vs casks) | Faible | Bloquant B3 | `goreleaser check` + snapshot test AVANT release |
| Cosign signing conflict avec attest-build-provenance | Faible | Bloquant D3 | Tester en snapshot, desactiver si conflit |
| Surcharge: 45 JH sur 30j (1.5 FTE) | Haute | Moyen | Prioritiser le chemin critique, reporter E5/E6/E7 |

---

## Definition of Done — v0.6.0

- [ ] A1 — CI verte (lint + test + build)
- [ ] A2 — Quality gates bloquants
- [ ] A3 — Tests stores natifs en CI (3 OS)
- [ ] A4 — 0 vulnerabilite connue (govulncheck)
- [ ] B1 — Depot registre communautaire operationnel
- [ ] B2 — CHANGELOG automatise (git-chglog)
- [ ] B3 — Homebrew + Scoop actifs
- [ ] B4 — Discord serveur cree
- [ ] B5 — Release taggee + artifacts publies
- [ ] **`multiai version` → `0.6.0`**

## Definition of Done — v1.0.0

Tout le DoD v0.6.0 +:

- [ ] C1 — 7 ADR + diagrammes C4 (architecture/)
- [ ] C2 — 7 doc.go + godoc complet + pkg.go.dev
- [ ] C3 — 20+ pages VitePress (build ok, 0 dead link)
- [ ] C4 — CONTRIBUTING.md v2 + 10 good first issues
- [ ] D1 — Coverage >= 75% (90% sur packages critiques)
- [ ] D2 — 15+ benchmarks avec benchstat CI
- [ ] D3 — Score securite >= 9.0/10, Cosign actif
- [ ] E1 — Campagne lancement executee (PH, HN, Reddit, newsletters)
- [ ] E2 — Video demo (3 min) + tutoriel (12 min)
- [ ] E3 — 6 articles de blog publies
- [ ] E4 — Programme ambassadeur lance (10 createurs)
- [ ] E5/E6/E7 — Partnerships, case studies, KPIs (si temps)
- [ ] **500+ stars GitHub** (J+90, pas J+30)
- [ ] **10+ contributeurs** (J+90)

---

## KPIs de succes

| KPI | Seuil J+7 | Seuil J+30 | Seuil J+90 |
|-----|-----------|------------|------------|
| Stars GitHub | 50 | 200 | **500** |
| Contributeurs | 2 | 5 | **10** |
| Forks | 5 | 20 | 50 |
| npm downloads/semaine | 100 | 500 | 2000 |
| Docker pulls/semaine | — | 100 | 500 |
| Membres Discord | 50 | 200 | **500** |
| Pages VitePress | 15 | **20** | 25 |
| Coverage | — | 50% | **75%** |
| Govulncheck | **0** | **0** | **0** |
| CI status | **verte** | **verte** | **verte** |
| Product Hunt votes | — | 50 | — |
| Newsletter mentions | 2 | 3 | 5 |
| Partenariats actifs | — | 2 | **4** |
| Temoignages publics | — | 2 | **5** |

---

## Budget temps par agent (estimation)

| Agent | Sprint A | Sprint B | Sprint C | Sprint D | Sprint E | Total |
|-------|----------|----------|----------|----------|----------|-------|
| **Forge** | A1(0.5) + A2(1) + A3(1.5) = 3 | B3(1) + B5(1.5) = 2.5 | — | — | — | **5.5 JH** |
| **Sentinel** | A2(0.5) + A4(1) = 1.5 | — | — | D1(5) + D2(1) + D3(3) = 9 | — | **10.5 JH** |
| **Huldah+Bezalel** | — | B2(2) = 2 | C1(3) + C2(2) + C3(3) + C4(1.5) = 9.5 | — | E2(3) = 3 | **14.5 JH** |
| **Atlas** | — | B4(1) = 1 | C4(1.5) = 1.5 | — | E1(4) + E2(3) + E3(5) + E4(3) + E5(3) + E6(2) + E7(0.5) = 20.5 | **23 JH** |
| **Nexus** | — | B1(2) = 2 | — | — | E1(0) (coordination) | **2 JH** |
| **Total** | **5 JH** | **7.5 JH** | **11 JH** | **9 JH** | **23.5 JH** | **~56 JH** |

Note: certains JH en parallele — Atlast peut travailler sur E3/E4/E5 pendant que Huldah fait C1/C2. Charge reelle ~45 JH sur 30-45 jours calendaires.

---

## Stories reportees (post-v1.0.0)

Ces stories des agents ne sont PAS dans le perimetre v1.0.0 :

| Story | Raison |
|-------|--------|
| Credential stores natifs OS (v0.6.0 original) | Deja livre dans S5.x |
| Auto-update (v0.5.0) | Deja livre |
| i18n English | Deja livre (66 messages) |
| Registry community profiles (v0.6.0) | Deplace dans Sprint B1 |
| Badges Codecov / Go Report Card | Trop tot, depend de D1 coverage |
| Community registry web UI | Post-v1.0.0 (v1.1.0) |
| Multi-org support | Post-v1.0.0 (v1.2.0) |

---

## Plan d'action immediate (prochaines 24h)

1. **Laurent** : Creer PAT classic (scope repo) → TAP_GITHUB_TOKEN dans secrets repo
2. **Laurent** : Creer repos `homebrew-tap` et `scoop-bucket` (public)
3. **Laurent** : Creer `profiles-multiai` (public, si pas deja fait)
4. **Forge** : Demarrer A1 (Fix CI gofmt + compilation + i18n)
5. **Nexus** : Demarrer B1 (Depot registre communautaire)
6. **Atlas** : Preparer B4 (Discord server) + drafts E1 (posts HN/PH/Reddit)

---

*Document genere par Nexus (Orchestrator) — 2026-07-12*
*Sources: Forge S9.x, Sentinel S10.x, Atlas M1-M3+S1-S7, Huldah+Bezalel S1-S6*
