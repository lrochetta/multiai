# Roadmap v0.3.0 → « le meilleur »

Date 2026-07-05 · Auteurs Atlas (Strategist) + Forge (Architect-Dev) — BMAD+ · Basé sur les 7 rapports d'audit v0.3.0 (01→07, findings contre-vérifiés adversarialement, aucun REFUTED).

---

## 1. Rappel du verdict

**Moyenne : 3.7/10 — régression nette vs l'audit v0.2.1 (5.5/10).**

| Dimension | v0.2.1 | v0.3.0 | Tendance |
|---|---|---|---|
| Produit & fonctionnalités | — | 3.5 | ↓ |
| Distribution & packaging | — | 2.5 | ↓↓ |
| Architecture | — | 3.5 | ↓ |
| Qualité de code | 5.5 | 4.5 | ↓ |
| Tests & CI/CD | — | 3.0 | ↓↓ |
| UX / DX / Docs | — | 4.5 | ↔ (UX ↑, docs ↓↓) |
| Sécurité | 5.5 | 4.5 | ↓ |
| **Moyenne** | **5.5** | **3.7** | **↓** |

La régression n'est pas due au code livré (le PowerShell v0.3.0 sur npm fonctionne) mais à **trois failles structurelles** :

1. **Le flux central config→launch du binaire Go est cassé** : la sentinelle `__MULTIAI_CREDSTORE__` est écrite dans le `.env` mais jamais relue — le CLI enfant reçoit ce littéral comme clé API, et la clé saisie peut même être perdue silencieusement (01-02, 03-01, 04-01, 07-03 — tous CONFIRMED).
2. **Un clone frais est inutilisable** : les profils `.env` et la doc VitePress sont exclus de git par des motifs non ancrés (`*.env`, `docs/`) — exit 2 sur toutes les commandes Go après `git clone` (03-04, 06-03 — CONFIRMED).
3. **~40 % des claims du README sont invérifiables ou faux** : commandes vaporware, badge « 9.5/10 » auto-décerné, CI fantôme, code mort vendu comme features (01-01, 02-01, 04-12, 05-06, 06-08…).

L'ambition « LE meilleur routeur multi-IA du marché » est atteignable — les différenciateurs réels existent (voir §2) — mais elle exige d'abord de **réparer ce qui casse l'utilisateur**, puis de **rendre chaque claim vérifiable par un inconnu**.

---

## 2. Définition MESURABLE de « le meilleur »

Concurrence identifiée (01-produit, section « Positionnement concurrentiel ») : **claude-code-router** (routage par requête + UI web), **cc-switch** (GUI de bascule providers, communauté), **LiteLLM proxy** (vrai cost tracking, 100+ providers, retry/fallback par requête), **profils natifs OpenCode / direnv / mise** (isolation env par dossier).

Différenciateurs réels et défendables de multiai (confirmés par l'audit) : isolation env par liste blanche au lancement, **fallback chains au niveau lanceur** (aucun concurrent lanceur ne le fait), couverture **3 CLIs × 14 providers** en un menu, UX française soignée.

« Le meilleur » = les 8 critères suivants, tous **mesurables et vérifiables par un tiers** :

| # | Critère | Cible mesurable | Concurrent battu |
|---|---|---|---|
| M1 | **Installation qui marche pour un inconnu** | `git clone` + 1 commande → premier launch réussi sur Windows/macOS/Linux, vérifié par un job E2E en CI publique | cc-switch (parité), claude-code-router (parité) |
| M2 | **Time-to-first-launch** | < 2 minutes de l'install au premier CLI lancé (wizard first-run branché) | Tous (aucun ne mesure ça) |
| M3 | **Confiance vérifiable** | Repo public, CI verte publique, releases signées Cosign + SBOM, badges réels (codecov/goreportcard) — **0 claim README sans code exécutable derrière** | claude-code-router, cc-switch (aucun ne signe) |
| M4 | **Couverture multi-CLI × providers** | 3 CLIs × 14+ providers × fallback chains, identique sur toutes les plateformes (une seule implémentation de référence) | claude-code-router (mono-CLI), OpenCode natif (mono-CLI) |
| M5 | **Fallback chains + sélection par projet** | Relance auto sur profil de secours + `.multiai.yaml` par dossier projet réellement câblé | Personne côté lanceur |
| M6 | **Visibilité coût réelle** | Estimation coût/session croisant durée × prix OpenRouter (ModelPricing existe déjà), cumul affichable | Parité minimale avec LiteLLM sur le cas d'usage lanceur |
| M7 | **Découverte de modèles** | `multiai models` / `search` réels sur l'API OpenRouter (client déjà écrit, à brancher), cache 1h effectif | Personne côté lanceur |
| M8 | **Adoption mesurable** | ≥ 500 téléchargements npm/semaine, ≥ 100 stars, ≥ 3 contributeurs externes à +6 mois | cc-switch (à rattraper) |

**Anti-critère** : tout claim non tenu est une dette de confiance qui coûte plus cher que la feature manquante — un dev exigeant vérifie avant d'adopter (01, section concurrentielle : « rédhibitoire »).

---

## 3. Phases

### Phase 0 — Quick wins : réparer ce qui casse l'utilisateur (3-5 jours)

**Règle d'ordre : les items 0.1 et 0.2 d'abord — ce sont les deux bugs qui rendent le produit inutilisable.**

| # | Action concrète | Fichiers concernés | Findings | Effort | Impact |
|---|---|---|---|---|---|
| 0.1 | **Réparer le credential store write-only** : résoudre `__MULTIAI_CREDSTORE__` via `store.Get` au chargement (dans `LoadDir` ou `BuildCleanEnv`), reconnaître la sentinelle dans `IsPlaceholder` pour échouer proprement, et ne JAMAIS retourner `nil` quand la clé n'a été persistée nulle part (`wizard.go:290-296`). Ajouter le test d'intégration config→launch qui aurait attrapé le bug | `internal/config/wizard.go:269,290-296`, `internal/env/env.go:34-60`, `internal/cli/launcher.go:71-76`, `pkg/dotenv/dotenv.go:73-93`, `tests/` | **01-02, 03-01, 04-01, 07-03** | 0.5 j | Produit +1, Code +0.5, Sécurité +0.5 |
| 0.2 | **Tracker profils + docs dans git** : remplacer `*.env` et `docs/` par des motifs ancrés dans `.gitignore`, `git add -f` des 17 templates Go + profils PS 00-57 manquants (le garde-fou `prepublishOnly` prouve que ce sont des placeholders), ancrer `/docs` | `.gitignore:2,31`, `multiai-go/configs/profiles/*.env`, `multiai-powershell/configs/profiles/00-57`, `multiai-go/docs/` | **03-04, 06-03** | 0.5 j | Produit +0.5, Distribution +0.5, UX/Docs +0.5 |
| 0.3 | **Purger README/CHANGELOG de tout claim non implémenté** : retirer `models`/`search`/`compare`, « cache 1h », « estimation coût + cumul », « héritage YAML », « hooks », « SecureString », « credential store natif », « Fusion panel d'experts », « CI/CD complète », badges « 9.5/10 » et « 10/10 », « 45+ tests », « Go 1.23 », « 16 pages » — ou les marquer explicitement « Roadmap ». Section Fonctionnalités en deux colonnes « PowerShell (npm) » / « Go (beta) » | `README.md`, `CHANGELOG.md`, `multiai-go/README.md` | **01-01, 01-04, 02-16, 02-17, 03-15, 04-12, 04-25, 05-02, 05-06, 06-01, 06-07, 06-08, 06-17, 06-21** | 0.5 j | Produit +0.5, UX/Docs +1 (meilleur ROI crédibilité/effort du projet) |
| 0.4 | **Rotation immédiate de la clé DeepSeek** trouvée en clair sur disque + suppression du fichier hors de l'arborescence git | `brainstorm laurent/clé deepseek….txt` | **07-08** | 0.5 h | Sécurité +0.5 |
| 0.5 | **Whitelist env case-insensitive** : normaliser `strings.ToUpper(key)` avant lookup + ajouter `APPDATA`, `LOCALAPPDATA`, `ProgramFiles`, `HOMEDRIVE/HOMEPATH` (l'enfant démarre sans PATH sous Windows) | `internal/env/env.go:9-21,44` | **04-03** | 0.5 j | Code +0.5 |
| 0.6 | **Support `%VAR%` dans `safeExpandEnv`** (ou migration des profils vers `${VAR}`) + références intra-profil — sinon l'isolation `CLAUDE_CONFIG_DIR` est silencieusement cassée en Go | `internal/env/env.go:24-31`, `configs/profiles/*.env` | **01-08, 03-08, 04-02** | 0.5 j | Code +0.5, Produit +0.5 |
| 0.7 | **Déplacer `.github/` à la racine du repo** (workflows + dependabot), trigger `master` (pas `main`), ajouter `working-directory: multiai-go` sur tous les jobs — le déblocage de TOUT le reste | `multiai-go/.github/` → `.github/`, `ci.yml:5`, `release.yml:46` | **01-06, 02-01, 03-05, 04-04, 05-01, 05-08, 07-06** | 0.5 j | Tests/CI +1, Distribution +0.5 |
| 0.8 | **Une seule source de version** : `const version` → `var` (sinon `-X main.version` est un no-op silencieux), bannière et User-Agent lisant cette var, supprimer les « 0.5.0 » anticipés du packaging | `cmd/multiai/main.go:18`, `internal/menu/interactive.go:18`, `internal/openrouter/client.go:38`, `Makefile:2`, `packaging/*` | **01-13, 02-03, 02-06, 03-07, 04-16, 06-02, 07-11** | 0.5 j | Distribution +0.5, UX/Docs +0.5 |
| 0.9 | **Exit non-zéro sur échec de lancement** (`runLaunch` retourne `nil` sur toute erreur aujourd'hui) + inverser les defer `signal.Stop`/`close(sigCh)` (panic possible au double Ctrl+C) | `cmd/multiai/main.go:215-288`, `internal/cli/launcher.go:120-121` | **04-08, 04-09, 05-12, 06-14** | 0.5 j | Code +0.5 |

**Total Phase 0 : ~4 jours. Livrable : un produit qui ne ment plus et ne casse plus son propre utilisateur.**

---

### Phase 1 — Fondations : installable, vérifiable, testé (3-4 semaines)

| # | Action concrète | Fichiers concernés | Findings | Effort | Impact |
|---|---|---|---|---|---|
| 1.1 | **DÉCISION STRATÉGIQUE (bloquante, jour 1)** : trancher l'implémentation primaire. Option A (recommandée) : porter la v0.3.0 en Go et geler le PS en maintenance. Option B : assumer PS comme produit, requalifier Go en beta. L'état actuel — « primaire » en retard de 2 versions sur « legacy » distribuée sous le même nom npm — est la pire option | Décision d'architecture (ADR dans le repo) | **01-03, 02-16, 03-03, 05-06, 06-06** (#12 v0.2.1 aggravé) | 0.5 j | Conditionne tout le reste |
| 1.2 | **`go:embed` des profils + répertoire utilisateur** `~/.multiai/profiles` prioritaire, extraction au first-run — règle d'un coup clone frais, `go install`, brew, deb, npm | `cmd/multiai/main.go:21-37`, nouveau `internal/profile/embed.go`, `configs/profiles/` | **01-09, 02-04, 03-06, 04-10** | 2 j | Produit +1, Distribution +1 |
| 1.3 | **Restructurer le repo publié** : soit `multiai-go/` devient la racine d'un repo dédié, soit `go.mod` → `github.com/lrochetta/multiai/multiai-go`. Ajouter LICENSE à la racine (badge MIT pointe vers un fichier absent) | `multiai-go/go.mod:1`, structure du repo, `LICENSE` | **02-07, 04-11, 06-09** | 1 j | Distribution +1 |
| 1.4 | **Passer le repo en PUBLIC + premier tag + première release goreleaser réelle** : archives + checksums + Cosign + SBOM. Tant que le repo est privé : zéro communauté, zéro confiance vérifiable | Repo GitHub, tags, `.goreleaser.yml` | **02-02, 02-05, 05-05, 07-05** | 2 j | Distribution +1.5, M3/M8 débloqués |
| 1.5 | **Corriger la chaîne d'install** : nommage archives dash→underscore aligné sur goreleaser, `install.js` vérifie statut HTTP + checksum + signature Cosign avant exécution, créer réellement `homebrew-tap`/`scoop-bucket` avec un PAT (ou supprimer ces sections) | `packaging/npm/install.js`, `scripts/install.sh`, `.goreleaser.yml:64-78`, `release.yml` | **02-08, 02-10, 02-13, 07-04, 07-16** | 2 j | Distribution +1, Sécurité +0.5 |
| 1.6 | **Une seule identité npm** : trancher entre `multiai` (PS publié) et le binaire Go — pattern optionalDependencies par plateforme (esbuild) ; supprimer `multiai-installer` et `multiai-cli` fantômes | `packaging/npm/package.json`, `multiai-powershell/package.json` | **02-09** | 1 j | Distribution +0.5 |
| 1.7 | **Tests de régression sécurité + couverture du chemin critique** : injection hooks (payloads métacaractères), config→launch roundtrip, exit codes, concurrence store (`-race` en CI), round-trip base64 des stores. Cibles : `internal/cli` ≥ 70 %, `internal/config` ≥ 60 % (60 % du code de production est aujourd'hui à 0 %) | `internal/cli/`, `internal/config/`, `internal/secret/`, `tests/` | **05-03, 05-04, 05-10, 05-13, 03-15, 04-19** | 4 j | Tests/CI +1.5 |
| 1.8 | **Réparer la CI pour qu'elle passe** : `.golangci.yml` migré v2, `gofmt -w` sur les 3 fichiers non conformes, actions épinglées par SHA, `matrix.os == 'windows-latest'`, job Pester + script `npm test` (le paquet npm part aujourd'hui en prod sans un seul test exécuté) | `.golangci.yml`, `ci.yml`, `multiai-powershell/package.json`, `tests/unit/` | **04-21, 05-07, 05-09, 05-11, 05-14, 07-15** | 2 j | Tests/CI +1 |
| 1.9 | **Contrat de profil formel partagé Go/PS** : spec versionnée du format `.env` (clés métadonnées `FALLBACK`/`REGION`/`SKIP_SECRET_CHECK`/`CLEAR_ENV`, sémantique d'expansion) + tests de conformité des deux implémentations | Nouveau `PROFILE-SPEC.md`, `internal/profile/profile.go:33-38`, `code-router.ps1:87` | **03-08, 04-15** | 2 j | Architecture +1 |
| 1.10 | **Durcir le stockage des secrets** : master key dérivée d'un secret non stocké sur disque (le `DeriveKey` PBKDF2 existe déjà, code mort) ou keyring OS ; corriger le round-trip base64 Set/Get ; `os.UserHomeDir()` au lieu de `os.Getenv("HOME")` ; instancier le store une seule fois | `internal/secret/secret.go:45,51-63,80,102`, `store_windows.go:44`, `store_darwin.go:31`, `crypto.go:18` | **03-09, 03-10, 04-07, 04-19, 04-20, 07-01, 07-10, 07-14** | 2 j | Sécurité +1 |
| 1.11 | **Brancher l'onboarding** (le code est écrit, jamais appelé — #13 v0.2.1 « faux-corrigé ») : `IsFirstRun`/`RunWelcome` dans `runInteractiveLoop`, marqueur first-run réellement lu | `cmd/multiai/main.go:185-213`, `internal/onboarding/wizard.go` | **01-10, 02-12, 06-04** | 1 j | UX/Docs +0.5, M2 débloqué |

**Total Phase 1 : ~3-4 semaines. Livrable : un inconnu peut installer, vérifier et faire confiance.**

---

### Phase 2 — Différenciation produit (1-2 mois)

| # | Action concrète | Fichiers concernés | Findings | Effort | Impact |
|---|---|---|---|---|---|
| 2.1 | **Porter la v0.3.0 dans l'implémentation de référence** (si option A retenue en 1.1) : catalogue 14 providers **data-driven** (YAML embarqué, comme le `$ProviderCatalog` PS — pas 5 hardcodés dans 5 fichiers), régions, fallback chains, erase keys, profils 60-83 | `internal/config/wizard.go:58-94`, nouveau `internal/catalog/`, `configs/profiles/` | **01-03, 01-12, 03-03, 03-16 (C16), 06-06** | 1-2 sem | Produit +1.5, Architecture +1 |
| 2.2 | **Câbler `.multiai.yaml` par projet + héritage `Extends` réel** — LA feature quotidienne que personne n'a côté lanceur (le code existe, il manque ~50 lignes dans main.go + la résolution d'héritage) | `internal/profile/project.go`, `yaml.go:31`, `cmd/multiai/main.go` | **01-07, 03-02, 04-13, 06-05** | 3 j | Produit +1 (M5) |
| 2.3 | **Implémenter réellement `multiai models`/`search`/`compare`** sur l'API OpenRouter : le client + cache existent (`client.go`), les brancher au switch, borner la lecture HTTP (`io.LimitReader`), complétions shell générées dynamiquement depuis les profils | `internal/openrouter/client.go`, `cmd/multiai/main.go:126-182`, `internal/cli/completion.go:18` | **01-01, 04-12, 04-31, 06-01, 06-20, 07-13** | 1 sem | Produit +1 (M7) |
| 2.4 | **Cost logging honnête** : croiser durée/modèle avec `ModelPricing` (déjà dans le client) pour une vraie estimation + cumul session — ou renommer la feature « launch log » tant que ce n'est pas fait | `code-router.ps1:1010-1021`, `internal/openrouter/client.go:24-27`, nouveau `internal/cost/` | **01-04, 06-07** | 1 sem | Produit +1 (M6, critère d'achat face à LiteLLM) |
| 2.5 | **Profil par défaut + `multiai doctor` + suppression de profils** (diagnostic clés/CLIs installés ; « ajout/suppression à la volée » est annoncé mais la suppression n'existe pas) | `cmd/multiai/main.go`, `code-router.ps1` | **01-11**, gaps produit 01 §2/4/6 | 3 j | Produit +0.5 (M2) |
| 2.6 | **Refondre les hooks AVANT de les câbler** : n'échapper que les valeurs substituées (jamais la commande entière), supprimer `os.ExpandEnv` post-échappement, compléter l'échappement PowerShell (`;`, `|`, `&`, `$()`), copier `py.Hooks` dans `Profile`, fusionner RunBefore/RunAfter | `internal/cli/hooks.go:14-37,55-57`, `internal/profile/yaml.go:130-171` | **03-17, 04-05, 04-06, 05-04, 07-07** | 3 j | Sécurité +0.5, Code +0.5 |
| 2.7 | **Credential stores natifs réels** (wincred/Keychain/libsecret via keyring) — ou renommer honnêtement le claim en « fichier chiffré local » | `internal/secret/store_*.go` | **03-09, 04-07, 07-02** | 1 sem | Sécurité +1 |
| 2.8 | **Fallback chain intelligente** : ne pas relancer sur exit 130/Ctrl+C (relance surprise = facturation surprise), portage dans l'implémentation de référence | `code-router.ps1:1136`, équivalent Go | **04-29** | 1 j | Produit +0.5 |
| 2.9 | **DX scriptable** : JSON via `encoding/json` (le JSON artisanal est invalide dès qu'une valeur contient `\`), zéro sortie humaine sur stdout en `--json`, `jsonError` utilisé, flags parsés seulement avant `--`, `config --provider` implémenté ou retiré, messages Go sans syntaxe PS, `list --json` → `[]`, option « q. Quitter », typographie unique | `internal/cli/launcher.go:176-194`, `display.go`, `cmd/multiai/main.go:292-312`, `profile.go:163` | **04-17, 04-22, 04-30, 04-32, 06-10, 06-11, 06-12, 06-15, 06-16** | 1 sem | UX/Docs +1, Code +0.5 |
| 2.10 | **Assainir la structure** : scinder `internal/cli` (ui/launch/hooks), unifier whitelist commandes et catalogues (aujourd'hui en 3-5 exemplaires), menu déterministe (slice, pas map), `--allow-custom-command` → whitelist extensible par config validée | `internal/cli/`, `internal/profile/project.go:101-104`, `internal/menu/interactive.go:41-64`, `launcher.go:57-63` | **03-11, 03-12, 03-14, 03-16, 04-18, 04-28, 07-09** | 1 sem | Architecture +1, Code +0.5 |

**Total Phase 2 : ~6-8 semaines. Livrable : les différenciateurs M4-M7 réels, plus aucun concurrent lanceur à parité.**

---

### Phase 3 — Adoption & distribution (2-3 mois, en parallèle partiel de la fin de Phase 2)

| # | Action concrète | Fichiers concernés | Findings | Effort | Impact |
|---|---|---|---|---|---|
| 3.1 | **Canaux d'install réellement vivants** : brew tap + scoop bucket alimentés par goreleaser (PAT dédié), AUR avec checksums réels, retirer windows-arm64 fantôme du manifeste Scoop, cohérence des 3 variantes d'instructions | `packaging/`, `.goreleaser.yml`, repos tap/bucket | **02-05, 02-10, 02-15**, C5 (01) | 1 sem | Distribution +1.5 (M1) |
| 3.2 | **Site VitePress versionné, à jour et déployé** (GitHub Pages en CI) : purger les flags inexistants (`-n`, `-v`), l'API imaginaire (`~/.multiai/config.yaml`), aligner sur les canaux réels ; héberger ou supprimer `rochetta.fr/multiai/install.*` | `multiai-go/docs/`, `docs/reference/commands.md:44-49`, `.github/workflows/docs.yml` | **02-11, 02-14, 03-13, 06-13**, C14 (01) | 1 sem | UX/Docs +1 |
| 3.3 | **E2E en CI publique sur 3 OS** : `multiai list --json`, `launch --dry-run --json`, install depuis chaque canal — le test qui rend M1 vérifiable en continu | `.github/workflows/e2e.yml` | reco 05 §12 | 3 j | Tests/CI +1, Distribution +0.5 |
| 3.4 | **Kit communauté** : CONTRIBUTING à la racine Go, templates d'issues, README EN + FR, ROADMAP réécrite et synchronisée avec le CHANGELOG, guide « ajouter un provider en 1 fichier » (possible grâce au catalogue data-driven de 2.1) | `CONTRIBUTING.md`, `.github/ISSUE_TEMPLATE/`, `ROADMAP.md` | **02-18, 06-19**, C16 (03) | 1 sem | Produit +0.5 (M8) |
| 3.5 | **Exploration routage par requête (mode proxy opt-in)** : le manque bloquant identifié face à claude-code-router — spike d'abord, décision ensuite. Ne PAS lancer avant que M1-M3 soient verts | Nouveau module, ADR préalable | gap concurrentiel 01 | 2-3 sem (spike 3 j) | Produit +1 potentiel — seule vraie extension de périmètre justifiée |
| 3.6 | **Preuves publiques** : badges réels (codecov, goreportcard), benchmarks publiés avec méthodologie, comparatif honnête vs claude-code-router/cc-switch/LiteLLM dans la doc | `README.md`, `docs/` | **02-17, 06-08** (remplacement des badges auto-décernés) | 3 j | UX/Docs +0.5, M3/M8 |

**Total Phase 3 : ~6-8 semaines. Livrable : un projet qu'on peut recommander publiquement, adoption mesurable.**

---

## 4. Ce qu'il faut ARRÊTER de faire

1. **Arrêter de maintenir deux implémentations divergentes sous le même nom et le même numéro de version.** La v0.3.0 est 100 % PowerShell, le Go « primaire » stagne à 0.2.1 avec 17 profils contre 38, une sémantique `%VAR%` différente et des menus différents — c'est le problème #12 de v0.2.1 **aggravé** (01-03, 02-16, 03-03, 03-08, 05-06, 06-06). Tant que la décision 1.1 n'est pas prise, chaque feature livrée creuse l'écart. Une implémentation de référence, une seule.

2. **Arrêter les claims marketing non tenus.** Chaque ligne du README doit pointer vers du code exécutable. À supprimer immédiatement :
   - badge « score 9.5/10 » et « 10/10 » auto-décernés — contredits par les audits internes eux-mêmes (02-17, 06-08) ;
   - « 45+ tests » — comptage réel : 32 fonctions Test + 2 benchmarks (03-15, 05-02, 06-21) ;
   - `multiai models`/`search`/`compare` « livrés » — vaporware dans les DEUX implémentations (01-01, 04-12, 05-06, 06-01) ;
   - « Cost logging : estimation coût + cumul session » — c'est un log de lancement renommé (01-04, 06-07) ;
   - « Credential store natif Windows/macOS/Linux » — trois stubs qui retombent sur le même fichier (04-07, 07-02) ;
   - « SecureString : clés jamais en clair » — zéro usage de SecureString dans le code (04-25) ;
   - « Fusion — panel d'experts avec synthèse automatique » — aucun code, juste un slug de modèle (06-17) ;
   - « CI/CD complète lint→test 6 OS→security→build→benchmark » — jamais exécutée, jamais exécutable en l'état (02-01, 05-01) ;
   - « Cache 1h, fallback offline », « héritage YAML », « hooks », badge « Go 1.23 », « 16 pages » (01-07, 06-01, 06-21).

3. **Arrêter d'écrire du code mort vendu comme feature.** ~31 % d'`internal/` n'est jamais atteint par main.go : openrouter, onboarding, logging, YAML/projet, hooks, PBKDF2 — dont certains portent des vulnérabilités latentes (injection hooks) et tous génèrent des claims faux (01-07, 01-10, 03-02, 04-05, 04-13, 04-14, 07-10, 07-12). Règle à adopter : **pas de merge sans câblage dans main + test + ligne de doc** — sinon le code va dans une branche, pas dans master.

4. **Arrêter le packaging anticipé.** Neuf fichiers référencent une v0.5.0 qui n'existe nulle part, des repos tap/bucket inexistants, des checksums placeholders depuis deux audits, trois identités npm dont deux fantômes (02-03, 02-05, 02-09, 02-18). Le packaging suit la release, jamais l'inverse : goreleaser génère, on ne pré-écrit pas.

5. **Arrêter de polluer le dépôt** : `push-github.ps1` (script interne mentionnant « repo PRIVE ») tracké et destiné au public, zips dupliqués, profils 60-62 orphelins à la racine, `docs/` racine vide, clé API vivante dans un dossier de brainstorm, ROADMAP contradictoire avec le CHANGELOG (01-16, 02-18, 03-18, 06-18, 06-19, 07-08).

6. **Arrêter l'auto-évaluation dans les livrables.** Les scores, c'est l'audit ou les badges tiers qui les donnent (codecov, goreportcard, utilisateurs). Un badge invérifiable sur un repo public est un boomerang de crédibilité (06-08) — exactement l'inverse du critère M3.

---

## 5. Projection des scores par phase

Même grille que l'audit v0.2.1 pour la continuité. Hypothèse : phases exécutées dans l'ordre, décision 1.1 = option A (Go référence), pas de nouvelle dette introduite.

| Phase | Durée cumulée | Produit | Distribution | Architecture | Code | Tests/CI | UX/Docs | Sécurité | **Moyenne** |
|---|---|---|---|---|---|---|---|---|---|
| **v0.3.0 (état actuel)** | — | 3.5 | 2.5 | 3.5 | 4.5 | 3.0 | 4.5 | 4.5 | **3.7** |
| **Phase 0** — quick wins | +1 semaine | 5.0 | 3.5 | 4.0 | 6.0 | 4.0 | 6.0 | 5.5 | **4.9** |
| **Phase 1** — fondations | +1 mois | 6.0 | 6.0 | 5.5 | 6.5 | 6.5 | 6.5 | 6.5 | **6.2** |
| **Phase 2** — différenciation | +3 mois | 7.5 | 6.5 | 7.0 | 7.5 | 7.0 | 7.5 | 7.5 | **7.2** |
| **Phase 3** — adoption | +5-6 mois | 8.5 | 8.5 | 7.5 | 8.0 | 8.0 | 8.5 | 8.0 | **8.1** |

Lecture :
- **Phase 0 rend 1.2 point de moyenne en ~4 jours** — le meilleur ROI du projet, parce qu'elle répare les deux casseurs d'utilisateur (0.1, 0.2) et supprime la dette de confiance documentaire (0.3). C'est aussi la seule phase qui **repasse au-dessus de rien** : elle arrête l'hémorragie.
- **Phase 1 repasse au-dessus du score v0.2.1 (5.5)** : c'est le seuil de dignité — installable, vérifiable, testé sur le chemin critique.
- **Phase 2 franchit le seuil « recommandable »** (7+) : les différenciateurs M4-M7 sont réels et démontrables.
- **Phase 3 vise le podium** (8+) : la note ne monte au-delà que par la preuve publique (CI verte, releases signées, adoption M8) — plus aucun point ne peut venir d'un claim.
- Le 10/10 n'est pas un objectif de roadmap : c'est le badge qu'on ne s'auto-décerne plus (§4.6).

---

## Prochaine action

Lancer la Phase 0, item 0.1 (credential store) et 0.2 (profils dans git) — 1 jour de travail, et le produit cesse de casser ses utilisateurs. La décision 1.1 (Go vs PS comme implémentation de référence) doit être prise par laurent avant d'engager la Phase 1.
