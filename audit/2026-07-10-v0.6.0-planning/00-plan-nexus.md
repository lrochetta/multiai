---

# Plan de Sprint v0.6.0 — multiai

**Orchestrateur :** Nexus
**Version cible :** v0.6.0
**Perimetre :** 4 epics, 26 stories (dont 2 consolidees), 4 sprints de 2 semaines
**Charge totale estimee :** 65-85 jours/homme
**Date debut :** Semaine du 13 juillet 2026

---

## 1. Consolidation des Stories

Apres analyse croisee des 4 epics (26 stories soumises), voici les consolidations effectuees :

| Stories fusionnees | Raison | Nouvel ID |
|---|---|---|
| S8.1 (Depot registre) + S8.2 (CI validation) | La CI de validation fait partie integrante de la creation du depot — impossible d'avoir l'un sans l'autre. Un seul livrable : `github.com/lrochetta/profiles-multiai` avec CI operationnelle. | **S8.1** |
| S7.4 (Govulncheck) + S7.5 (Golangci-lint) | Les deux sont des "quality gates" CI passant de non-bloquant a bloquant. Meme fichier (`ci.yml`), meme type de travail, meme agent (Sentinel). | **S7.4** |

Les 24 autres stories restent separees car leurs domaines techniques sont distincts.

### Consolidations rejetees (justification)

- **S6.4 (Homebrew) + S6.5 (Scoop)** : Meme si elles partagent le `TAP_GITHUB_TOKEN`, les cibles (macOS vs Windows) et les formats (`.rb` vs `.json`) sont suffisamment differents pour justifier 2 stories. La configuration partagee est documentee dans les dependances.
- **S5.4 (--store flag) + S5.6 (Fallback)** : Le flag `--store` gere le routage utilisateur ; le fallback gere la resilience technique. Deux preocccupations distinctes, meme si le code interagit.
- **S8.3 (search) + S8.4 (install)** : Les deux commandes partagent `internal/registry/` mais l'installation ajoute la verification SHA256, la gestion de conflits, et les entrees-sorties fichier — suffisamment de complexite pour justifier 2 stories.

---

## 2. Priorisation Finale

| Priorite | Stories | Criteres |
|---|---|---|
| **BLOCKER** (5) | S5.4, S6.6, S7.2, S7.3, S8.1 | Processus orphelins, echecs silencieux Windows, distribution impossible, registre inexistant |
| **HIGH** (10) | S5.1, S5.2, S5.3, S5.5, S6.1, S6.2, S6.4, S7.1, S8.3, S8.4 | Securite des secrets, installation via gestionnaires, tests E2E, decouverte de profils |
| **MEDIUM** (6) | S5.6, S5.7, S6.3, S6.5, S7.6, S8.5, S8.6, S7.4 | Fallback, migration, fuzzing, documentation, Discussions, badges |
| **LOW** (0) | — | |

Soit 5 BLOCKER, 10 HIGH, 8 MEDIUM.

---

## 3. Architecture des Sprints

```
Sprint 1 (Jours 1-10)     Sprint 2 (Jours 11-20)    Sprint 3 (Jours 21-30)    Sprint 4 (Jours 31-40)
══════════════════════     ══════════════════════    ═══════════════════════    ═══════════════════════
                                                    
[S5.5 Zeroization]         [S5.1 WinCred]           [S6.1 APT repo]           [S8.3 profile search]
[S7.2 Timeout]             [S5.2 Keychain]          [S6.2 AUR package]        [S8.4 profile install]
[S7.3 Env Win CI]          [S5.3 libsecret]         [S6.4 Homebrew tap]       [S5.7 Migration auto]
[S6.6 Install scripts]     [S5.6 Fallback]          [S6.5 Scoop bucket]       [S8.5 Contrib docs]
[S8.1 Registry repo+CI]    [S5.4 --store flag]      [S6.3 PS migration]       [S8.6 Discussions]
[S7.4 Quality gates CI]    [S7.6 Fuzz testing]      [S7.1 E2E tests]          [S8.7 Badges*]

Track A (Securite):  S5.5 → S5.1|S5.2|S5.3 → S5.6 → S5.4 → S5.7
Track B (Distrib):   S6.6 → S6.1|S6.2|S6.4|S6.5
Track C (Communaute):S8.1 → S8.3 → S8.4 → S8.5|S8.6
Track D (Qualite):   S7.2|S7.3|S7.4 → S7.6 → S7.1

(S8.7 est execute dans Sprint 1 car zero dependance et impact visibilite immediat)
```

---

## 4. Tableau des Sprints

### Sprint 1 — Fondation Securite & Infrastructure (Jours 1-10)

**Objectif :** Zeroisation memoire, processus enfants proteges, CI de qualite enforcee, scripts d'installation fonctionnels, depot registre cree.

| ID | Titre | Prio | JH | Agent | Dependances |
|---|---|---|---|---|---|
| S5.5 | Zeroisation memoire complete des secrets | HIGH | 2j | Forge | Aucune |
| S7.2 | Timeout/context sur processus enfants | BLOCKER | 2j | Forge | Aucune |
| S7.3 | Whitelist env case-insensitive Windows | BLOCKER | 1j | Forge | Aucune |
| S6.6 | Scripts d'installation cross-platform | BLOCKER | 2j | Forge | Aucune |
| S8.1 | Depot registre + CI validation profils | BLOCKER | 2j | Forge+Nexus | Aucune |
| S7.4 | Quality gates CI (govulncheck+golangci-lint bloquants) | MEDIUM | 2j | Sentinel | Aucune |
| S8.7 | Badges supplementaires (Codecov, Go Report Card, Scorecard) | MEDIUM | 0.5j | Atlas+Nexus | Aucune |

**Charge :** ~11.5j — parallelisable sur 3 agents (Forge 7j, Sentinel 2j, Atlas 0.5j, Nexus 2j)

**Allocation agent recommandee :**

| Agent | Jours | Stories |
|---|---|---|
| **Forge** | 7j | S5.5 (j1-2), S7.2 (j3-4), S7.3 (j5), S6.6 (j6-7), S8.1 (j8-9, avec Nexus) |
| **Sentinel** | 2j | S7.4 (j1-2) |
| **Atlas** | 0.5j | S8.7 (j1) |
| **Nexus** | 2j | S8.1 (j8-9, ops GitHub : creation depot, discussions, tokens) |

**Livrables sprint 1 :**
- `Zeroize()` implementee avec protection anti-optimisation compilateur
- `LaunchOptions.Timeout` operationnel, processus enfants tues apres delai
- `BuildCleanEnv()` case-insensitive sur Windows
- `install.sh` + `install.ps1` fonctionnels avec verification SHA256
- `github.com/lrochetta/profiles-multiai` cree avec CI validation
- Govulncheck et golangci-lint bloquants en CI (0 warning, 0 CVE)
- README avec 12 badges (Codecov, Go Report Card, Scorecard)

---

### Sprint 2 — Credential Stores Natifs OS (Jours 11-20)

**Objectif :** Les 3 stores natifs OS implementes, fallback fichier operationnel, flag `--store` fonctionnel, fuzzing etendu.

| ID | Titre | Prio | JH | Agent | Dependances |
|---|---|---|---|---|---|
| S5.1 | Store natif Windows Credential Manager | HIGH | 3j | Forge | Aucune |
| S5.2 | Store natif macOS Keychain | HIGH | 3j | Forge | Aucune |
| S5.3 | Store natif Linux libsecret/D-Bus | HIGH | 3j | Forge | Aucune |
| S5.6 | Fallback fichier si store natif indisponible | MEDIUM | 1j | Forge | S5.1 (au moins un store natif) |
| S5.4 | Commande `--store <backend>` implementee | BLOCKER | 2j | Forge | S5.1, S5.2, S5.3, S5.6 |
| S7.6 | Fuzz testing etendu (5+ fuzzers) | MEDIUM | 3j | Sentinel | Aucune |

**Charge :** ~15j sur 2 agents (Forge 12j, Sentinel 3j)

**Allocation agent recommandee :**

| Agent | Jours | Stories |
|---|---|---|
| **Forge** | 12j | S5.1 (j1-3), S5.2 (j4-6), S5.3 (j7-9), S5.6 (j10), S5.4 (j11-12) |
| **Sentinel** | 3j | S7.6 (j1-3) |

**Note :** Les 3 stores (S5.1, S5.2, S5.3) sont developpes sequentiellement par Forge (meme pattern, 3 OS). Si un agent supplementaire est disponible (Oholiab), ils peuvent etre parallellises :
- Forge : WinCred (j1-3) + libsecret (j4-6)
- Oholiab : Keychain (j1-3)
- Forge : Fallback (j7) + `--store` flag (j8-9)

**Livrables sprint 2 :**
- `multiai config --store wincred` operationnel sur Windows 10/11
- `multiai config --store keychain` operationnel sur macOS (CGo + fallback shell-out)
- `multiai config --store secret-service` operationnel sur Linux
- `multiai config --store file` force le store fichier existant
- `multiai config --store auto` detecte automatiquement le meilleur backend
- Fallback silencieux vers fichier AES-256-GCM si store natif indisponible
- 5 nouveaux fuzzers operationnels, zero crash apres 1h CPU
- Messages i18n FR/EN pour tous les nouveaux textes

---

### Sprint 3 — Distribution & Qualite (Jours 21-30)

**Objectif :** Packages APT, AUR, Homebrew, Scoop publies. Migration PowerShell legacy operationnelle. Tests E2E complets.

| ID | Titre | Prio | JH | Agent | Dependances |
|---|---|---|---|---|---|
| S6.1 | Depot APT (Ubuntu/Debian) | HIGH | 2j | Forge | S6.6 (install scripts) |
| S6.2 | Paquet AUR (Arch Linux) | HIGH | 2j | Forge | S6.6 |
| S6.4 | Homebrew tap (macOS) | HIGH | 1j | Forge | S6.6 |
| S6.5 | Scoop bucket (Windows) | MEDIUM | 1j | Forge | S6.6 |
| S6.3 | Migration automatique depuis PowerShell legacy | MEDIUM | 2j | Forge | Aucune |
| S7.1 | Tests d'integration complets (E2E) | HIGH | 4j | Sentinel | Aucune |

**Charge :** ~12j sur 2 agents (Forge 8j, Sentinel 4j)

**Allocation agent recommandee :**

| Agent | Jours | Stories |
|---|---|---|
| **Forge** | 8j | S6.1 (j1-2), S6.2 (j3-4), S6.4 (j5), S6.5 (j6), S6.3 (j7-8) |
| **Sentinel** | 4j | S7.1 (j1-4) |

**Note sprint 3 :** Les stories S6.1, S6.2, S6.4, S6.5 partagent le meme prerequis (S6.6 fait en sprint 1) et peuvent etre executees en parallele si plusieurs agents disponibles.

**Dependances transverses :**
- `TAP_GITHUB_TOKEN` doit etre configure avant la release (PAT classique, scope `public_repo`) — preparation par Nexus en sprint 1
- Les repos `homebrew-tap` et `scoop-bucket` doivent exister — creation par Nexus en sprint 2

**Livrables sprint 3 :**
- `apt install multiai` fonctionnel sur Debian/Ubuntu
- `yay -S multiai` ou `paru -S multiai` sur Arch Linux
- `brew install --cask lrochetta/tap/multiai` sur macOS
- `scoop install multiai` sur Windows
- `multiai migrate --from-ps` fonctionnel
- Suite de 15+ tests E2E passant sur Linux, macOS, Windows

---

### Sprint 4 — Communaute & Finalisation (Jours 31-40)

**Objectif :** Commandes registre `search`/`install` operationnelles, documentation contributrice, Discussions active, migration auto des secrets.

| ID | Titre | Prio | JH | Agent | Dependances |
|---|---|---|---|---|---|
| S8.3 | Commande `profile search` | HIGH | 2j | Forge | S8.1 (registre + index.json) |
| S8.4 | Commande `profile install` | HIGH | 3j | Forge | S8.1, S8.3 |
| S5.7 | Migration auto secrets fichier → natif | MEDIUM | 2j | Forge | S5.4 (flag --store operationnel) |
| S8.5 | Documentation contributrice pour profils | MEDIUM | 1.5j | Huldah | S8.1 |
| S8.6 | Programme de feedback (GitHub Discussions) | MEDIUM | 1j | Atlas+Nexus | Aucune |

**Charge :** ~9.5j sur 3 agents (Forge 7j, Huldah 1.5j, Atlas+Nexus 1j)

**Allocation agent recommandee :**

| Agent | Jours | Stories |
|---|---|---|
| **Forge** | 7j | S8.3 (j1-2), S8.4 (j3-5), S5.7 (j6-7) |
| **Huldah** | 1.5j | S8.5 (j1-1.5) |
| **Atlas+Nexus** | 1j | S8.6 (j1) |

**Note sprint 4 :** S5.7 (migration auto) peut commencer en fin de sprint 2 si S5.4 est livre tot. Dans ce cas, le deplacer en sprint 3 pour equilibrer la charge.

**Livrables sprint 4 :**
- `multiai profile search <query>` avec cache 1h et mode offline
- `multiai profile install <id>` avec verification SHA256 et gestion conflits
- Migration automatique des secrets a l'activation d'un store natif
- `profiles-multiai/CONTRIBUTING.md` complet
- `docs/guide/community-profiles.md` sur le site VitePress
- GitHub Discussions active avec 6 categories
- Release v0.6.0 preparee

---

## 5. Diagramme de Dependances Complet

```
Sprint 1                        Sprint 2                     Sprint 3                        Sprint 4
─────────                       ─────────                    ─────────                        ─────────

S5.5 Zeroization ──────────────┐
                               │
S7.2 Timeout ──────────────────┤
                               │
S7.3 Env CI ───────────────────┤
                               │
S6.6 Install scripts ──────────┼─────────────────┬──────────┬──────────┐
                               │                 │          │          │
S8.1 Registry repo+CI ─────────┼─────────────────┼──────────┼──────────┼─────┬─────┐
                               │                 │          │          │     │     │
S7.4 Quality gates CI ────────┘                 │          │          │     │     │
                                                 │          │          │     │     │
S5.1 WinCred ──────────────┐                    │          │          │     │     │
S5.2 Keychain ─────────────┤                    │          │          │     │     │
S5.3 libsecret ────────────┤                    │          │          │     │     │
                           │                    │          │          │     │     │
S5.6 Fallback ◄────────────┘                    │          │          │     │     │
   │                                             │          │          │     │     │
S5.4 --store flag ◄─────────────────────────────┘          │          │     │     │
   │                                                       │          │     │     │
   └── S5.7 Migration ◄────────────────────────────────────┘          │     │     │
                                                                      │     │     │
S6.1 APT ◄────────────────────────────────────────────────────────────┘     │     │
S6.2 AUR ◄──────────────────────────────────────────────────────────────────┘     │
S6.4 Homebrew ◄────────────────────────────────────────────────────────────────────┘
S6.5 Scoop ◄──────────────────────────────────────────────────────────────────────┘
S6.3 PS migration ───────── (independant, peut aller dans n'importe quel sprint)

S7.6 Fuzz testing ──────── (independant)
S7.1 E2E tests ─────────── (independant)
S8.7 Badges ────────────── (independant, sprint 1)

S8.3 search ◄──────────────────────────────────────────────────────────────┐
S8.4 install ◄─────────────────────────────────────────────────────────────┼────┘
S8.5 Contrib docs ◄────────────────────────────────────────────────────────┘
S8.6 Discussions ───────── (independant)
```

---

## 6. Risques et Mitigations

| Risque | Probabilite | Impact | Mitigation |
|---|---|---|---|
| **CGo cross-compilation** : macOS Keychain necessite CGo, bloque la compilation croisee Linux→macOS | Eleve | Moyen | `//go:build darwin && cgo` + `darwin && !cgo` fallback shell-out. Deja gere par les build constraints de GoReleaser. |
| **GoReleaser Pro** : La section `repos` (APT) necessite GoReleaser Pro (payant) | Moyen | Eleve | Alternative : workflow custom avec `aptly` + `ghp-import`. Valider licence GoReleaser avant sprint 3. |
| **PAT classique deprecie** : GitHub deprecie les PAT classiques — requis pour Homebrew/Scoop cross-repo push | Eleve | Eleve | Utiliser un GitHub App token (`app:write`) ou un PAT fine-grained avec acces aux 2 repos. |
| **Windows SmartScreen** : Binaire non signe bloque par Windows Defender | Moyen | Moyen | Documenter la procedure de contournement. Envisager la signature Authenticode en v0.7.0. |
| **macOS Gatekeeper** : Binaire non notarise bloque sur macOS | Moyen | Moyen | `post_install` Homebrew : `sudo xattr -dr com.apple.quarantine`. Envisager notarization en v0.7.0. |
| **D-Bus indisponible** : Linux sans gnome-keyring ni keepassxc | Faible | Moyen | Fallback fichier AES-256-GCM automatique (S5.6). Detection via `$DBUS_SESSION_BUS_ADDRESS` + test `secret-tool`. |
| **Migration concurrente** : 2 processus `config` lances simultanement tentent la migration | Faible | Moyen | Lock inter-processus dedie pour la migration (fichier `.migration.lock` existant). |
| **Charge forfait 4 sprints** : 65-85 JH sur 40 jours ouvrés = 1.6-2.1 JH/jour — faisable avec 2-3 agents | Moyen | Moyen | Si retard, couper S8.6 (Discussions) et S6.3 (PS migration) de la release v0.6.0. |

---

## 7. KPIs de Succes v0.6.0

| KPI | Mesure | Cible | Source |
|---|---|---|---|
| **Credential store natif** | Nombre de stores implementes | 3/3 (WinCred, Keychain, libsecret) | `multiai config --store <backend>` |
| **Commandes profile** | Nombre de commandes registre | 2 (search, install) | `multiai profile search/install` |
| **Tests E2E** | Nombre de tests E2E passant | 15+ | `go test -tags=e2e ./tests/` |
| **CI quality gates** | Nombre de quality gates bloquants | 3 (govulncheck, golangci-lint, gosec) | Fichier `ci.yml` |
| **Fuzzers** | Nombre de fuzzers operationnels | 7+ | `make fuzz` |
| **Zeroization** | `go test -race -count=1` secret package | Clean | Rapport CI |
| **Package managers** | Nombre de gestionnaires couverts | 5 (APT, AUR, Homebrew, Scoop, npm) | Documentation installation |
| **Migration PS** | Commande `multiai migrate` operationnelle | Oui | Test manuel |
| **Stars GitHub** | Metrique communaute | 100+ | GitHub API |
| **Profils communautaires** | Nombre de profils dans le registre | 5+ | `index.json` |
| **Discussions** | Discussions actives | 20+ | GitHub UI |
| **Badges README** | Nombre de badges lisibles | 12 | Fichier `README.md` |
| **Zero CVE** | Govulncheck | 0 vulnerabilite | `govulncheck ./...` |
| **Zero lint** | Golangci-lint | 0 warning | `golangci-lint run ./...` |

---

## 8. MVP Sprint 1 — Definition Precise

Le MVP du sprint 1 est la fondation sur laquelle tout le reste repose. Il doit etre **complet et stable** avant de commencer le sprint 2.

### Perimetre MVP

```
BLOCKER (absolument necessaire) :
  S7.2 — Timeout/context sur processus enfants  (securite processus)
  S7.3 — Env case-insensitive Windows            (stabilite Windows)
  S6.6 — Install scripts                          (distribution)
  S8.1 — Depot registre + CI                      (communaute)

HIGH (tres fortement recommande) :
  S5.5 — Zeroisation memoire                      (securite secrets)
  S7.4 — Quality gates CI                         (qualite code)

MEDIUM (nice-to-have) :
  S8.7 — Badges                                   (visibilite)
```

### Criteres de sortie MVP

- [ ] `go test -race -count=1 ./internal/secret/` propre (S5.5)
- [ ] `multiai launch -p <profil> --timeout 30s` tue le processus apres 30s (S7.2)
- [ ] `%Path%` et `%PATH%` resolus correctement sous Windows (S7.3)
- [ ] `curl https://raw.githubusercontent.com/.../install.sh | bash` installe multiai (S6.6)
- [ ] `github.com/lrochetta/profiles-multiai` accessible avec CI verte (S8.1)
- [ ] `govulncheck ./...` retourne 0 et `golangci-lint run ./...` retourne 0 (S7.4)
- [ ] README avec 12 badges, Codecov operationnel (S8.7)
- [ ] CI globale verte sur les 3 OS

### Gate de fin de sprint 1

Avant de demarrer le sprint 2, les BLOCKERS du sprint 1 doivent etre en production sur `master`. Tout BLOCKER non livre entraine un "sprint 1 extension" de 2-3 jours avant de commencer le sprint 2.

---

## 9. Organisation des Agents par Sprint

| Sprint | Forge (Dev) | Sentinel (QA) | Atlas (Strat) | Huldah (Doc) | Nexus (Orch) |
|---|---|---|---|---|---|
| **1** | S5.5, S7.2, S7.3, S6.6, S8.1 | S7.4 | S8.7 | — | S8.1 (ops), S8.7 |
| **2** | S5.1, S5.2, S5.3, S5.6, S5.4 | S7.6 | — | — | Creer homebrew-tap + scoop-bucket |
| **3** | S6.1, S6.2, S6.4, S6.5, S6.3 | S7.1 | — | — | Configurer TAP_GITHUB_TOKEN |
| **4** | S8.3, S8.4, S5.7 | — | S8.6 | S8.5 | S8.6, Release v0.6.0 |

**Besoins en agents supplementaires :**
- Sprint 2 : un deuxieme dev (Oholiab) permettrait de parallelliser les 3 stores (gain de 5 jours)
- Sprint 4 : Huldah pour la documentation contributrice

---

## 10. Recommandations pour l'Execution

### Rythme de release
- **Sprint 1** : Pas de release — la fondation est instable. Tag `v0.6.0-alpha.1` en fin de sprint.
- **Sprint 2** : Tag `v0.6.0-beta.1` — stores natifs disponibles pour test.
- **Sprint 3** : Tag `v0.6.0-beta.2` — package managers operationnels, tests E2E.
- **Sprint 4** : Tag `v0.6.0` — release finale.

### Dependances inter-sprints à surveiller
1. `TAP_GITHUB_TOKEN` (PAT) doit etre cree AVANT le sprint 3 (Nexus le prepare en sprint 2)
2. Les repos `homebrew-tap` et `scoop-bucket` doivent exister AVANT le sprint 3
3. S5.7 (migration) peut commencer tot si S5.4 est livre en avance — opportunite de debordement positif

### Stories candidates au debordement si retard
(par ordre de priorite decroissante pour la release)
1. S6.3 (Migration PowerShell) — independant, peut etre reporte a v0.6.1
2. S8.6 (GitHub Discussions) — independant, pas de code
3. S6.5 (Scoop) — alternative : install.ps1 de S6.6 couvre deja Windows

### Verification avant release v0.6.0
- [ ] `goreleaser check` passe
- [ ] `goreleaser build --snapshot --clean` produit tous les artefacts
- [ ] Tests E2E passent sur Linux + macOS + Windows
- [ ] Govulncheck = 0 CVE
- [ ] Golangci-lint = 0 warning
- [ ] Gosek = 0 finding (exclusions documentees)
- [ ] `go test -race -count=1 ./...` = tout vert
- [ ] Documentation build VitePress sans erreur
- [ ] Cosign signature operationnelle (optionnel)

---

*Document genere par Nexus (Orchestrator) le 2026-07-10. Base sur les stories soumises par Forge, Sentinel, Atlas et Shield.*