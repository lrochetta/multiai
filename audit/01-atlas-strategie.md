# Audit Strategique — Atlas (Miriam)

**Projet :** multiai — Routeur multi-IA  
**Version auditee :** 0.4.0-dev (Go), v0.3.0 (PowerShell, legacy)  
**Date :** 2026-07-05  
**Auditeur :** Atlas (Strategiste / Product Manager)  

---

## 1. Resume Executif

**multiai** est un CLI cross-platform (Go) 100% open-source qui unifie l'acces a 13+ fournisseurs d'IA - Anthropic, DeepSeek, OpenAI, Z.ai, OpenRouter, MiniMax, StepFun, Qwen, Kimi, SiliconFlow, MiMo, Requesty, LiteLLM - via un unique point d'entree. Il gere pour vous les variables d'environnement, les cles API et le routage entre les CLI d'IA de codage (Claude Code, Codex CLI, OpenCode), avec chiffrement AES-256-GCM, chaines de fallback, catalogues data-driven, et profils embarques.

### Proposition de valeur unique

> **"Un seul outil pour lancer n'importe quel CLI d'IA avec n'importe quel fournisseur, sans fuite de cles, sans configuration manuelle, sans pollution d'environnement."**

### Marche cible

- **Primaire :** Developpeurs individuels qui utilisent plusieurs IA de codage (Claude Code, Codex CLI, OpenCode) et veulent basculer entre fournisseurs sans se prendre la tete avec les variables d'environnement.
- **Secondaire :** Equipes qui standardisent sur un routeur pour garantir l'isolation des cles API et la conformite entre postes de travail.
- **Tertiaire :** Utilisateurs de BMAD+ (framework d'augmentation multi-agent) qui exploitent le routage d'IA comme infrastructure sous-jacente.

### Etat actuel (score : 8/10)

Le projet a accompli une migration spectaculaire de PowerShell vers Go en ~15 jours, atteignant la parite fonctionnelle avec 37 profils embarques, un credential store natif, un catalogue extensible et des fonctionnalites avancees (fallback, expansion `%VAR%`, decouverte OpenRouter). La qualite du code et de la securite est excellente. Les lacunes residuelles sont : la couverture de test partielle, l'absence de releve de couts reel, le credential store natif OS pas encore implemente, et un ecosysteme de packaging pas totalement deploye (Homebrew/Scoop en skip_upload, AUR avec SHA256 manquant).

---

## 2. Description du Produit

### Qu'est-ce que multiai ?

multiai est un couteau suisse pour l'IA de codage. Il agit comme un **routeur d'environnement** : il prepare et isole les variables d'environnement necessaires a chaque CLI d'IA, puis lance le processus enfant avec le bon jeu de cles, de parametres et de configuration.

```
multiai launch -p ds
   │
   ├── Charge le profil `ds.env` (DeepSeek V4 Pro via Claude Code)
   ├── Resout le credential store (cle dechiffree)
   ├── Isole l'environnement (30 variables systeme seulement)
   ├── Injecte les variables du profil
   └── Lance `claude` (DeepSeek V4 Pro)
```

### Fonctionnalites cles

| Fonctionnalite | Description | Statut |
|---|---|---|
| Menu interactif | Lancement outil→profil avec navigation complete | 🟢 Stable |
| 37 profils embarques | Claude Code (17), Codex CLI (8), OpenCode (12) | 🟢 Stable |
| Lancement direct | `multiai launch -p <shortcut>` | 🟢 Stable |
| Catalogue 13 fournisseurs | Data-driven (YAML embarque), 32 shortcuts | 🟢 Stable |
| Chaines de fallback | Fallback automatique si le processus enfant echoue | 🟢 Stable |
| Expansion `%VAR%` | Resolution d'indirection entre variables de profil | 🟢 Stable |
| Credential store | AES-256-GCM, sentinelle dans .env, resolution au launch | 🟢 Stable |
| Isolation d'env | Liste blanche ~30 variables systeme | 🟢 Stable |
| Decouverte OpenRouter | `models`, `search`, `compare` avec cache 1h | 🟢 Stable |
| Journal de sessions | `sessions.jsonl` sans secrets | 🟢 Stable |
| Onboarding premier demarrage | Assistant de configuration au premier run | 🟢 Stable |
| Profils YAML | Support `.yaml`/`.multiai.yaml` par projet avec heritage | 🟢 Stable |
| Plugin hooks | `before_launch`/`after_launch` avec template variables | 🟢 Stable |
| Shell completion | bash, zsh, fish, PowerShell | 🟢 Stable |
| BMAD+ integre | Detection, version, packs, menu update | 🟢 Stable |
| Effacement de cles | Par fournisseur ou global, confirmation `oui` | 🟢 Stable |
| Erase store + .env | Purge credential store + remise du placeholder | 🟢 Stable |
| Profils dynamiques | Ajout/suppression de modeles OpenRouter a la volee | 🟡 Go-only |
| Cost tracking | Journalise duree, pas de cout reel (limitation API) | 🟡 Neutre |

### Architecture technique

```
multiai-go/                   → Module Go 1.23
├── cmd/multiai/
│   ├── main.go               → Point d'entree, switch de sous-commandes
│   └── cmd_openrouter.go     → Sous-commandes models/search/compare (init)
├── internal/
│   ├── assets/               → Embedding des 37 profils .env
│   ├── catalog/              → Catalogue fournisseurs (providers.yaml)
│   ├── cli/                  → Launcher, display, fallback, hooks, completion
│   ├── config/               → Wizard interactif + erase menu
│   ├── env/                  → Isolation whitelist + expansion %VAR%
│   ├── fsutil/               → Ecriture atomique de fichiers
│   ├── logging/              → Logger structure + journal sessions
│   ├── menu/                 → Menus interactifs (top, tool, profile)
│   ├── onboarding/           → Premier demarrage
│   ├── openrouter/           → Client API, cache, search, compare
│   ├── profile/              → Chargement .env, YAML, projet
│   └── secret/               → AES-256-GCM + stores natifs OS (stubs)
└── pkg/dotenv/               → Parser .env robuste
```

---

## 3. Analyse du Marche

### Positionnement

```
                    Prix
                    haut
                     │
          Claude Code        Codex CLI
          (officiel)         (officiel)
                     │
              ┌──────┴──────┐
              │   multiai   │
              │  (routeur)  │
              └──────┬──────┘
                     │
          OpenRouter API     LiteLLM
          (gateway pur)      (proxy local)
                     │
                    bas
```

### Concurrents directs

| Concurrent | Type | Forces | Faiblesses |
|---|---|---|---|
| **OpenRouter** (API) | Gateway multi-modele | 300+ modeles, Fusion panel, pricing transparent | Pas de gestion d'env, pas d'isolation, API-only |
| **Requesty** | Gateway EU | RGPD-friendly, load balancing, gratuit 200 req/j | Pas de CLI router, moins de providers |
| **LiteLLM** | Proxy local | Proxy Docker, OpenAI-compatible | Pas de CLI native, pas d'isolation d'env, Docker requis |
| **Direct (manuel)** | Aucun outil | Zero cout | Risque de fuite de cles, pas d'isolation, perte de temps |

### Concurrents indirects

| Concurrent | Type | Menace |
|---|---|---|
| **IDE integre** (Cursor, Windsurf) | IDE avec IA embarquee | Faible - ne remplace pas Claude Code/Codex CLI |
| **`env` manuel** | Export bash/PowerShell | Eleve - les devs font ca depuis 20 ans |
| **dotenv** | Gestionnaire .env | Tres faible - ne lance pas de processus, pas d'isolation |

### Opportunites de marche

1. **Marche porteur :** Explosion des CLI d'IA (Claude Code, Codex CLI, OpenCode) - chaque nouveau CLI augmente la valeur de multiai exponentiellement.
2. **Verrouillage par reseau :** Plus il y a de profils, plus l'outil est utile. Les 37 profils actuels sont un moat defensif.
3. **Differentiation securite :** Aucun concurrent ne propose d'isolation d'environnement et de credential store natif.
4. **BMAD+ ecosystem :** Positionne comme couche d'infrastructure pour le framework d'augmentation.

### Positionnement recommande

> **"Le seul routeur de CLI d'IA avec isolation securisee, credential store natif, et 37 profils preconfigures."**

---

## 4. Completeness Fonctionnelle

### Parite Go vs PowerShell

| Fonctionnalite | PowerShell v0.3.0 | Go v0.4.0-dev | Statut |
|---|---|---|---|
| 37 profils embarques | 🟢 | 🟢 | Parite |
| 13 fournisseurs + 32 shortcuts | 🟢 | 🟢 (14 dont Requesty ceu gap) | Parite+ |
| Menu interactif | 🟢 | 🟢 | Parite |
| Lancement direct (-p) | 🟢 | 🟢 | Parite |
| Sortie JSON | 🟢 | 🟢 | Parite |
| Dry-run | 🟢 | 🟢 | Parite |
| Credential store | 🟢 (SecureString) | 🟢 (AES-256-GCM) | Superieure |
| Isolation d'environnement | 🟢 | 🟢 | Parite |
| Fallback chains | 🟢 (L1135-1163) | 🟢 | Parite |
| Expansion `%VAR%` | 🟢 | 🟢 | Parite |
| Chiffrement | 🟢 (PowerShell SecureString) | 🟢 (AES-256-GCM + PBKDF2) | Superieure |
| `models`/`search`/`compare` | 🟡 (ecran d'aide statique) | 🟢 (API reseau + cache) | Superieure |
| Profils dynamiques | 🟢 | 🟢 | Parite |
| Erase keys | 🟢 | 🟢 (store + .env) | Parite+ |
| Journal sessions | 🟢 (costs.log) | 🟢 (sessions.jsonl) | Renommee |
| Onboarding | 🔴 | 🟢 | Go-only |
| Profils YAML + projet | 🔴 | 🟢 | Go-only |
| Plugin hooks | 🔴 | 🟢 | Go-only |
| Shell completion | 🔴 | 🟢 (4 shells) | Go-only |
| BMAD+ menu | 🔴 | 🟢 | Go-only |

**Verdict : Parite fonctionnelle depassee.** Le Go est strictement superieur sur 10 aspects, et la version PowerShell est archivee.

### Couverture de test par package

| Package | Couverture | Verdict |
|---|---|---|
| `pkg/dotenv` | 93.9% | 🟢 Excellent |
| `internal/env` | 86.2% | 🟢 Tres bien |
| `internal/secret` | 77.1% | 🟢 Bien |
| `internal/assets` | 73.7% | 🟡 Acceptable |
| `internal/profile` | 27.2% | 🔴 Insuffisant |
| `internal/config` | 15.2% | 🔴 Critique |
| `internal/cli` | 7.1% | 🔴 Critique |
| `internal/menu` | 0% | 🔴 Critique |
| `internal/openrouter` | 0% | 🔴 Critique |
| `internal/logging` | 0% | 🔴 Critique |
| `internal/onboarding` | 0% | 🔴 Critique |
| `cmd/multiai` | 0% | 🔴 Critique |

**Probleme critique :** 6 packages ont 0% de couverture. Les tests d'integration dans `tests/` couvrent partiellement certains chemins, mais sans tests unitaires modifies, le refactoring est risque.

### Fonctionnalites manquantes vs vision produit

| Feature | Priorite | Raison |
|---|---|---|
| **Cost tracking reel** | Haute | Impossible sans API des fournisseurs (token usage). Solution : intercepter stderr du child |
| **Credential store natif OS** (Windows Credential Manager, macOS Keychain, libsecret) | Haute | Les stubs existent, le fallback fichier fonctionne, mais la securite reelle necessite les stores natifs |
| **Internationalisation (i18n)** | Basse | Tout est en francais avec accents supprimes (CP850). L'anglais serait pertinent pour le marche global |
| **Telemetrie anonyme** | Optionnelle | Optionnel, pour comprendre l'usage (adoption, profils populaires) |
| **Mode daemon / watch** | Basse | Surveiller un repertoire et lancer automatiquement |
| **Plugin system** | Moyenne | Permettre aux utilisateurs d'ecrire leurs propres hooks/plugins |
| **Dashboard web** | Basse | Interface graphique pour visualiser les profils et les sessions |

---

## 5. Roadmap Recommandee

### Priorites 🔴🟡🟢

Legende : 🔴 Critique | 🟡 Important | 🟢 Souhaitable

---

### Phase 1 : v0.4.0 "Stabilisation" (Sprint 1-2, 1-2 semaines)

| ID | Priorite | Tache | Impact |
|---|---|---|---|
| P1 | 🔴 | **Augmenter couverture de test** des packages critiques (config, cli, menu, openrouter, logging, onboarding). Cible : min 50% | Qualite / Confiance |
| P2 | 🔴 | **Credential store natif OS** : implementer Windows Credential Manager (Go: `go-credentialstore` ou syscall direct) | Securite |
| P3 | 🔴 | **Credential store natif macOS Keychain** (via CDSA ou CLI `security`) | Securite |
| P4 | 🔴 | **Credential store natif Linux libsecret** (via D-Bus) | Securite |
| P5 | 🔴 | **Release v0.4.0 officielle** : publier sur GitHub, activer Homebrew (skip_upload→false), Scoop (skip_upload→false), AUR (fixer SHA256) | Distribution |
| P6 | 🟡 | **Mettre a jour SHA256 dans AUR** via script de mise a jour | Distribution |
| P7 | 🟡 | **Publier npm avec le binaire Go** (le package npm actuel distribue encore le PowerShell) | Distribution |
| P8 | 🟡 | **Creer les repositories Homebrew tap et Scoop bucket** | Distribution |
| P9 | 🟡 | **Ajouter tests CI specifiques aux stores natifs OS** | Qualite |

**Criteres de sortie v0.4.0 :**
- [ ] Couverture de test > 40% sur les 6 packages critiques
- [ ] Credential store natif sur au moins 1 OS (Windows recommande)
- [ ] Homebrew tap public, Scoop bucket public
- [ ] AUR SHA256 verifie
- [ ] npm package distribue le binaire Go
- [ ] `goreleaser check` OK, release pipeline testee

---

### Phase 2 : v0.5.0 "Intelligence" (Sprint 3-4, 2-3 semaines)

| ID | Priorite | Tache | Impact |
|---|---|---|---|
| P10 | 🔴 | **Cost tracking avec parsing de sortie** : intercepter stderr des CLI pour extraire les tokens (Claude Code: `"Tokens": {"input":..., "output":...}`) | Valeur utilisateur |
| P11 | 🔴 | **Fusion panel OpenRouter** : implementer le multi-modele avec synthese automatique (aujourd'hui, le profil or-fusion lance Claude Code avec 1 modele) | Differenciation |
| P12 | 🟡 | **Support de nouveaux CLI d'IA** : Cursor, Windsurf, Continue.dev, Aider | Portee |
| P13 | 🟡 | **Mode `multiai watch <dir>`** : lancement automatique a la detection de changement | Productivite |
| P14 | 🟡 | **Export de profil** : `multiai export -p <shortcut> --json` pour CI/CD | DevOps |
| P15 | 🟡 | **Migration de .golangci.yml vers v2** (format de configuration) | Qualite |
| P16 | 🟡 | **Tests de performance (benchmark) en CI** : comparer les versions, detecter les regressions | Qualite |

---

### Phase 3 : v1.0.0 "Plateforme" (Sprint 5-6, 3-4 semaines)

| ID | Priorite | Tache | Impact |
|---|---|---|---|
| P17 | 🔴 | **Internationalisation (EN + FR)** : messages, aide, menus | Marche global |
| P18 | 🔴 | **Site vitrine** : landing page, documentation, use cases | Acquisition |
| P19 | 🔴 | **Plugin system v2** : hooks avances, custom providers, scripts utilisateur | Extensibilite |
| P20 | 🔴 | **Mode batch** : `multiai batch profiles.txt` lancer plusieurs profils sequentiellement | Productivite |
| P21 | 🟡 | **Dashboard web local** : visualiser les sessions, les couts, l'usage | Monitoring |
| P22 | 🟡 | **CI/CD complet** : integration GitHub Actions avec coverage tracking | Qualite / Confiance |
| P23 | 🟡 | **Sponsor/OSS funding** : GitHub Sponsors, OpenCollective | Perennite |
| P24 | 🟢 | **Telemetrie anonyme opt-in** : adoption, popularite des profils | Produit |
| P25 | 🟢 | **CLI `multiai suggest`** : recommander le meilleur fournisseur/modele pour une tache | IA-native |

---

### Diagramme de la roadmap

```
S1-2           S3-4            S5-6            S7-8
v0.4.0 ────── v0.5.0 ─────── v1.0.0-rc ───── v1.0.0
│              │               │               │
├─Tests        ├─Cost tracking ├─i18n          ├─Sponsors
├─Stores OS    ├─Fusion panel  ├─Landing page  ├─Telemetrie
├─Release      ├─New CLI supp  ├─Plugin v2     ├─Mature
├─Homebrew     ├─Export        ├─Batch mode    ├─Enterprise
└─Scoop        └─Watch mode    └─Dashboard     └─Ecosystem
```

---

## 6. Qualite du Packaging / DX

### Analyse des canaux de distribution

| Canal | Statut | Notes |
|---|---|---|
| **npm** (`npx multiai install`) | 🟡 A finaliser | Package npm pointe encore vers PowerShell. La config Go est prete (package.json, install.js avec SHA256). A basculer a la release v0.4.0. Script `scan-secrets.js` anti-fuite en prepublishOnly. |
| **Go install** | 🟢 OK | `go install github.com/lrochetta/multiai@latest` fonctionne immediatement. Les profils sont embarques (embed) et materialises au premier run. |
| **Homebrew** | 🔴 Pas de publique | `.goreleaser.yaml` configure un cask Homebrew, mais `skip_upload: true` en attendant la creation du tap `lrochetta/homebrew-tap`. Post-install hook pour enlever la quarantine macOS. |
| **Scoop** | 🔴 Pas de publique | Meme chose : `skip_upload: true` en attendant `lrochetta/scoop-bucket`. |
| **Debian (.deb)** | 🟢 OK | GoReleaser construit le .deb via nfpm avec postinst pour les completions. Script local `build-deb.sh` comme fallback dev. |
| **AUR (Arch Linux)** | 🟡 SHA256 a fixer | PKGBUILD source-build parfait, mais `sha256sums` est `SKIP` - doit etre mis a jour via `scripts/update-aur-checksums.sh` avant chaque release. |
| **Install.sh** (curl pipe bash) | 🟢 OK | Installation universelle macOS/Linux avec verification SHA256, resolution de la derniere version via GitHub redirect. |
| **Windows (setup-go.ps1)** | 🟢 OK | Script d'installation complet avec telechargement Go + build + test + cross-compilation. |
| **macOS/Linux (setup-go.sh)** | 🟢 OK | Equivalent bash avec Homebrew fallback pour Go. |

### Qualite du code et CI/CD

| Aspect | Evaluation | Notes |
|---|---|---|
| **Structure du code** | 🟢 Excellent | Packages bien separes, responsabilites claires, interfaces propres (Store interface, catalog extensible) |
| **Securite** | 🟢 Excellent | Actions CI pinnees par SHA, gosec, govulncheck, Cosign keyless, checksums, sentinelle secret, zeroisation memoire, atomic writes, YAML bomb protection, escape shell |
| **CI/CD pipelines** | 🟢 Excellent | lint -> test (3 OS) -> security -> benchmark -> build -> release-check. Release : GoReleaser + Cosign + GitHub provenance attestation |
| **Dependabot** | 🟢 Configure | gomod + GitHub Actions + npm, hebdomadaire |
| **Changelog** | 🟢 Excellent | CHANGELOG.md complet, structure par version, divergences documentees |
| **Documentation** | 🟢 README complet | README.md avec tableaux, exemples, toutes les commandes documentees |
| **Site VitePress** | 🟢 Partiel | `docs/` existe avec 16 pages, mais non-audite dans ce rapport |
| **CLAUDE.md** | 🟢 Excellent | Instructions de projet completes, protocole memoire Karpathy Guardrails |
| **BMAD+ integration** | 🟢 Excellent | Agents, skills, role triggers, manifests complets |

### Notes de qualite supplementaires

**Points forts notables :**
- Modele de menace honnete dans la documentation du package `secret` (pas d'illusion de securite)
- Gestion des races conditions sur le master key (`O_CREATE|O_EXCL`)
- Separation claire metadata/environment dans `profile.go` (MetadataKeys)
- Gestion du BOM UTF-8 dans le parser `.env` (les fichiers PowerShell l'ajoutent)
- Signal forwarding SIGINT/SIGTERM au processus enfant avec detection Ctrl+C
- Verrouillage de concurrence avec `sync.Mutex` dans `session.go` et `secret.go`
- Protection XSS dans le profile loading (CWD pas autorise sans `MULTIAI_DEV`)
- Script scan-secrets.js pour empecher la fuite de cles dans npm

**Points a ameliorer :**
- `golangci-lint` desactive a cause du format v1 de `.golangci.yml`
- Pas de coverage tracking en CI (upload du artifact mais pas de visualisation)
- Pas de `Makefile` cible pour le benchmark individuel
- Les tests `TestSessionEvent_NoSecretFields` sont un design guard elegant mais fragile

---

## 7. Recommandations Strategiques

### Top 5 moves pour faire de multiai le meilleur outil de sa categorie

---

#### Recommandation n°1 🔴 : Passer le credential store natif OS de "planifie" a "implemente"

**Constat :** Les trois stores natifs (Windows Credential Manager, macOS Keychain, Linux libsecret) sont en etat de stubs qui deleguent au fichier AES. Le modele de menace est honnetement documente, mais c'est une dette de securite reelle.

**Action :** Implementer le Windows Credential Manager en priorite (plus grand marche, API accessible via `syscall`). Le Go n'a pas de binding standard, mais une approche via CLI `cmdkey` ou un package comme `github.com/AllenDang/go-credentialstore` pourrait fonctionner. Pour macOS, la CLI `security` est suffisante. Pour Linux, `secret-tool` (libsecret) est standard.

**Impact :** 🔴 Critique - transformation du modele de menace de "securite de fichier" a "securite OS"

**Effort estime :** 3-5 jours par store natif (recherche + implementation + tests)

---

#### Recommandation n°2 🔴 : Atteindre > 50% de couverture de test sur tous les packages

**Constat :** 6 packages a 0%, 2 packages sous 30%. Les tests d'integration couvrent l'essentiel mais ne suffisent pas pour le refactoring.

**Action :** Par priorite :
1. `internal/config` (wizard, erase, updateEnvFile) - 15.2% → priorite 1
2. `internal/cli` (launcher, fallback, hooks, display, completion) - 7.1% → priorite 2
3. `internal/menu` (interactive.go) - 0% → priorite 3
4. `internal/openrouter` (client, cache, search, profilegen) - 0% → priorite 4
5. `internal/logging` (logger, session) - 0% → priorite 5
6. `internal/onboarding` (wizard) - 0% → priorite 5

**Impact :** 🔴 Critique - confiance dans le refactoring, prevention de regressions

**Effort estime :** 4-6 jours pour atteindre 50% sur tous les packages

---

#### Recommandation n°3 🟡 : Activer la distribution complete (Homebrew, Scoop, AUR, npm Go)

**Constat :** Le .goreleaser.yaml prepare tout (casks, manifests, PKGBUILD), mais tout est en `skip_upload: true`. Les repositories Homebrew tap et Scoop bucket n'existent pas. Le AUR a SHA256=SKIP. npm distribue encore le PowerShell.

**Action :**
1. Creer `lrochetta/homebrew-tap` et `lrochetta/scoop-bucket`
2. Gerer les secrets `TAP_GITHUB_TOKEN`
3. Passer `skip_upload: false` dans `.goreleaser.yaml`
4. Mettre a jour SHA256 dans le PKGBUILD AUR
5. Basculer le package npm sur le binaire Go

**Impact :** 🟡 Important - chaque canal de distribution est un point d'entree utilisateur. Sans eux, la croissance est freinee.

**Effort estime :** 2-3 jours

---

#### Recommandation n°4 🟡 : Implementer le vrai cost tracking

**Constat :** Le journal de sessions est honnete sur ses limites ("le routeur ne voit pas les tokens consommes"), mais c'est la fonctionnalite la plus demandee par les utilisateurs de CLI d'IA. Les CLI comme Claude Code affichent leur consommation de tokens sur stderr.

**Action :** Intercepter stderr du processus enfant avec un pipe, parser les lignes contenant des metriques de token (format JSON dans Claude Code), agreger par session, et stocker dans le journal. Optionnellement, utiliser les prix catalogues d'OpenRouter du cache (`ModelInfo.Pricing`) pour estimer les couts.

**Impact :** 🟡 Important - valeur utilisateur immediate, differenciation concurrentielle

**Effort estime :** 3-5 jours (recherche des patterns de sortie des CLI, parsing, stockage)

---

#### Recommandation n°5 🟡 : Preparer l'internationalisation et le positionnement global

**Constat :** Tout le projet est en francais (messages, README, menus, aide). C'est un choix delibere et coherent avec le marche francophone initial, mais ca limite l'adoption globale. La convention CP850 (ascii-only) est un heritage PowerShell.

**Action :**
1. Extraire toutes les chaines utilisateur dans un systeme de messages (inspiration : `golang.org/x/text/message`)
2. Creer des bundles EN + FR
3. Detecter la langue via `LANG` ou une variable `MULTIAI_LANG`
4. Creer un README en anglais comme point d'entree principal
5. Positionner le site/documentation en anglais

**Impact :** 🟡 Important - adresse le marche global (90%+ des developpeurs)

**Effort estime :** 5-8 jours (refactoring des chaines, traduction, test)

---

### Matrice Impact / Effort

```
Impact
  ^
  │ P1-P4        P11
H │ (stores OS)  (Fusion panel)
  │
  │ P2              P10
  │ (tests 50%)     (cost tracking)
  │
  │ P5-P8            P17
  │ (distribution)   (i18n)
  │
  │ P9              P12-P14
  │ (CI tests OS)   (new CLI, export)
  │
  │ P15-P16          P18-P19
  │ (golangci, perf) (site, plugins)
  │
  └───────────────────────────> Effort
    Faible                  Eleve
```

**Conclusion strategique :** Les 5 moves ci-dessus sont ordonnes par priorite. Les 3 premiers (stores natifs, tests, distribution) sont les pre-requis pour considerer multiai comme un outil professionnel. Les 2 suivants (cost tracking, i18n) sont les vrais differentiateurs pour dominer la categorie.

---

## Annexe A : Metriques Produit

| Metrique | Valeur |
|---|---|
| Lignes de code Go | ~3 500 (estimation) |
| Fichiers Go | ~45 |
| Profils embarques | 37 |
| Fournisseurs supportes | 13 (14 avec Requesty ceu gap) |
| Shortcuts | 32 (+ 5 keyless = 37) |
| Tests unitaires | 45 (43 tests + 2 benchmarks) |
| Couverture moyenne | ~35% (estimation ponderee) |
| Cibles de build | 5 (win/amd64, darwin/amd64, darwin/arm64, linux/amd64, linux/arm64) |
| Canaux de distribution | 7 (npm, Go install, Homebrew, Scoop, deb, AUR, script) |
| Shells supportes | 4 (bash, zsh, fish, PowerShell) |
| Temps de developpement | ~15 jours (du premier commit Go a la parite) |

## Annexe B : Vulnerabilites de Securite Residuelles

| Risque | Niveau | Status |
|---|---|---|
| Master key AES cote fichier (pas de passphrase) | Medium | Accepte (documente), planifie (store OS) |
| Pas de verification d'integrite du binaire au runtime | Low | Compense par signature Cosign + checksums SHA256 |
| Pas de rate limiting sur le wizard de config | Low | Menace : brute-force local des cles (faible) |
| `MULTIAI_PROFILES_DIR` non restreint | Low | Usage administrateur/reseau, documente |
| Pas de revocation de cles | Low | Fonctionnalite erase couvre le retrait |

---

*Rapport genere par Atlas (Miriam), agent Strategiste/Product Manager BMAD+.*  
*Base sur l'exploration complete du codebase : multiai-go/, CHANGELOG.md, README.md, .goreleaser.yaml, workflows CI/CD, packaging et scripts.*
