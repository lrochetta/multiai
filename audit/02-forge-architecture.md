# Audit Architecture & Code — Forge (Bezalel)

**Projet :** multiai (Go CLI router multi-IA)
**Version :** 0.4.0-dev
**Date :** 2026-07-05

---

## 1. Vue d'ensemble de l'architecture

### Diagramme textuel des composants

```
┌──────────────────────────────────────────────────────────────────┐
│                       cmd/multiai/main.go                        │
│              Point d'entree CLI + registre commandes             │
│              (commands map: register/call pattern)               │
└──────────────────┬──────────────────────────────────────┬────────┘
                   │                                      │
         ┌─────────▼──────────┐           ┌──────────────▼──────┐
         │  internal/menu/    │           │  cmd_openrouter.go  │
         │  TUI interactif    │           │  (models/search/    │
         │  (bmad,launch)     │           │   compare)          │
         └─────────┬──────────┘           └──────────┬───────────┘
                   │                                  │
    ┌──────────────▼──────────────────────────────────▼───────────┐
    │                    internal/openrouter/                      │
    │  FetchModels → cache → GetModels → Top/Search/Find/Filters  │
    │  + menu interactif + profilegen (CreateProfile)             │
    └────────┬──────────┬──────────────┬──────────────────────────┘
             │          │              │
    ┌────────▼──┐  ┌────▼────┐  ┌─────▼────────┐
    │ cli/     │  │ config/ │  │ onboarding/  │
    │ launcher │  │ wizard  │  │ first-run    │
    │ fallback │  │ erase   │  │ welcome      │
    │ hooks    │  │         │  │              │
    │ display  │  └─────────┘  └──────────────┘
    │          │
    └────┬─────┘
         │
    ┌────▼────────────────────────────────────────────────────────┐
    │                     profile/                                │
    │  LoadDir (.env) / LoadDirYAML / LoadAllProfiles             │
    │  + project.go (.multiai.yaml) + yaml.go (YAML structs)     │
    └────┬────────────────────────────────────────────────────────┘
         │
    ┌────▼────┐  ┌───────────┐  ┌────────────┐  ┌──────────────┐
    │ env/   │  │ secret/   │  │ logging/   │  │  catalog/    │
    │ Expand │  │ AES-256   │  │ Logger     │  │  providers   │
    │ Build  │  │ Store     │  │ Session    │  │  .yaml       │
    │ Mask   │  │ (3 OS)    │  │ JSONL      │  │  (embarque)  │
    └────────┘  └───────────┘  └────────────┘  └──────────────┘
         │
    ┌────▼────┐  ┌───────────┐
    │ dotenv/ │  │ fsutil/  │
    │ Parse   │  │ Atomic   │
    │ pholder │  │ Write    │
    └─────────┘  └───────────┘
```

### Flux de donnees principal

```
1. main.go → profile.LoadDir() → dotenv.Parse() → []profile.Profile
2. main.go → menu.SelectTool() / SelectProfile()
3. main.go → cli.LaunchWithFallback()
   ├─ resolveStoredSecrets() → secret.Store.Get()
   ├─ validateSecrets() → dotenv.IsPlaceholder()
   ├─ buildProcessEnv() → env.BuildCleanEnv() / env.ExpandProfileEnv()
   ├─ RunBeforeHooks() → exec.Command(shell)
   ├─ cmd.Start() / Wait() → signal forwarding (SIGINT/SIGTERM)
   ├─ logLaunchSession() → logging.LogSession() [JSONL]
   └─ RunAfterHooks()
```

### Couplage entre packages

| Package | Dependances internes | Dependances externes |
|---------|---------------------|---------------------|
| `cmd/multiai` | assets, catalog, cli, config, menu, onboarding, openrouter, profile | standard |
| `cli` | env, logging, profile, secret, pkg/dotenv | standard |
| `config` | catalog, cli, profile, secret, pkg/dotenv | standard |
| `profile` | pkg/dotenv, go-yaml | gopkg.in/yaml.v3 |
| `env` | (aucune) | standard |
| `secret` | fsutil | standard |
| `openrouter` | fsutil, profile | standard |
| `catalog` | (aucune) | gopkg.in/yaml.v3 |
| `logging` | (aucune) | standard |
| `menu` | cli, profile | standard |
| `onboarding` | catalog, cli, config, logging, profile, dotenv | standard |
| `assets` | (aucune) | embed, standard |
| `fsutil` | (aucune) | standard |
| `pkg/dotenv` | (aucune) | standard |

**Aucune dependance cyclique detectee** — le graphe est oriente et acyclique.

---

## 2. Qualite du code Go

### Points forts

- **Gestion d'erreurs cohérente** : toutes les fonctions retournent `error` ; les erreurs sont wraps avec `%w` pour `errors.Is`/`errors.As` (ex: `fmt.Errorf("catalog: %w", err)`)
- **Thread-safety** : `sync.Mutex` dans `secret.encryptedFileStore`, `sync.OnceValues` dans `catalog.Default()`, `sync.Mutex` dans `logging.Logger`
- **Signal forwarding** : `signal.Notify` + goroutine + `atomic.Bool` pour detecter les interruptions utilisateur — robuste
- **Zeroize secrets** : `crypto.go` zeroize plaintext avec `for i := range plaintext { plaintext[i] = 0 }` apres decrypt
- **Atomic writes** : `fsutil.WriteFileAtomic` utilise `os.CreateTemp` (nom unique, pas de race) + fsync + rename
- **Pas de `init()` sauvage** : seul `cmd_openrouter.go` utilise `init()` pour le registre de commandes, et `logging/logger.go` pour initialiser le logger par defaut

### Problemes identifies

| Fichier:ligne | Probleme | Severite |
|---------------|----------|----------|
| `cli/hooks.go:57` | `os.ExpandEnv(cmdStr)` apres `escapeShellArg` — l'ordre est correct mais la combinaison peut produire des resultats surprenants si une variable d'env contient des caracteres d'echappement | 🟡 |
| `cli/hooks.go:84,133` | `RunBeforeHooks` et `RunAfterHooks` utilisent `cmd.Stdin = os.Stdin` — si stdin est un pipe, les hooks peuvent consommer des donnees destinees au processus enfant | 🟡 |
| `cli/launcher.go:150-155` | Deferred `close(sigCh)` puis `defer signal.Stop(sigCh)` — bien que l'ordre LIFO soit documente (signal.Stop avant close), c'est subtil et meriterait un commentaire plus explicite | 🟢 |
| `internal/logging/logger.go:38` | `init()` utilise `os.UserHomeDir()` sans fallback — si `HOME` et `USERPROFILE` sont absents, le logger reçoit un chemin vide | 🟡 |
| `internal/secret/secret.go:121-143` | Generation de cle maitre : `rand.Read(key)` — correct, mais `defer f.Close()` ignore l'erreur de Close | 🟢 |
| `internal/secret/crypto.go:44-48` | `binary.Write(mac, binary.BigEndian, uint32(blockNum))` — cette fonction panique si `mac.Write` echoue (jamais le cas avec HMAC, mais non verifie) | 🟢 |
| `internal/config/wizard.go:138,152,163,221,232` | Appels `reader.ReadString('\n')` — les erreurs ignorees sont documentees mais constituent un pattern dangereux en cas de pipe EOF | 🟢 |
| `internal/openrouter/menu.go:64-69` | `readLine` ignore l'erreur de `ReadString` — si EOF arrive avec des donnees avant le `\n`, les dernieres donnees sont retournees, si EOF vide, `ok=false` | 🟢 |

### Detection de fuites memoires/goroutines

- **Goroutine leak potentielle** : `cli/launcher.go:175-182` — la goroutine de forwarding de signaux lit depuis `sigCh`. Si le processus enfant se termine rapidement, `cmd.Wait()` retourne, puis `defer signal.Stop(sigCh)` et `defer close(sigCh)` sont executes. Mais si une interruption survient entre `signal.Stop` et `close`, la goroutine peut recevoir un signal apres le stop et tenter d'ecrire sur un canal ferme. Le defer LIFO garantit `Stop` avant `close`, donc pas de send-on-closed. Cependant, si le processus enfant se termine et que le main goroutine sort de `cmd.Wait()` avant que la goroutine de signal ne soit terminee, la goroutine peut survivre brievement — mais sans consequence car le canal est ferme en dernier. **Aucun leak critique.**
- **Aucune fuite memoire evidente** dans les profils charges (le GC gere les `[]Profile`)

---

## 3. Structure du projet

### Respect du standard Go layout

| Critere | Evaluation |
|---------|-----------|
| `cmd/` pour les binaires | ✅ `cmd/multiai/main.go` + `cmd_openrouter.go` |
| `internal/` pour les packages prives | ✅ 11 packages internes |
| `pkg/` pour les packages partageables | ✅ `pkg/dotenv/` |
| `tests/` pour les tests d'integration | ✅ 4 fichiers de test (mais pas de `testdata/` a la racine) |
| `go.mod` module path correct | ✅ `github.com/lrochetta/multiai` |
| Pas de `src/` | ✅ |
| `Makefile` | ✅ |
| `.goreleaser.yaml` | ✅ |

### Cohérence des packages

- **`internal/cli/`** : le package le plus charge — melange lancement, fallback, affichage, completion shell et hooks. Envisager de le scinder (`cli/launcher`, `cli/display`, `cli/hooks`, `cli/completion`).
- **`internal/openrouter/`** : bien decoupe en 7 fichiers (client, cache, discover, menu, profilegen, source, menu_test)
- **`internal/secret/`** : bonne separation cross-platform (`store_windows.go`, `store_darwin.go`, `store_linux.go`)
- **`internal/profile/`** : 3 fichiers (profile, yaml, project) — excellent decoupage

### Problemes structurels

| Fichier | Probleme | Severite |
|---------|----------|----------|
| `cmd/multiai/main.go:32` | `commands` map globale + `register()` — pattern fonctionnel mais empeche les tests unitaires de `main.go` | 🟡 |
| `cmd/multiai/main.go:27` | `var version = "0.4.0-dev"` — ldflags-friendly, mais pas de fichier `VERSION` ni de `version/` package | 🟢 |
| `tests/` (root) | Les tests d'integration sont dans `package tests` (pas `_test.go` dans les packages) — convention valide mais isolee du source, ne peut pas tester les fonctions non exportees | 🟢 |
| `internal/openrouter/source.go:120-133` | `fallbackModels` est une package-level var (pas const) — pourrait etre `var` convertie en fonction pour eviter toute mutation accidentelle (bien que `embeddedModels()` retourne une copie) | 🟢 |

---

## 4. Tests

### Couverture globale

```
31 fichiers de test
~150 tests individuels
Benchmarks : 2 (LoadDir, FindByShortcut)
```

### Repartition par package

| Package | Tests | Type | Assertions |
|---------|-------|------|------------|
| `catalog` | 10 | Unitaires + validation | `reflect.DeepEqual`, `strings.Contains` |
| `cli` | 15 | Integration (vrai processus) + fallback | `errors.Is`, `os.Stat`, JSONL parsing |
| `config` | 16 | Integration (store + files) | lecture fichier, `secret.NewStore()` round-trip |
| `profile` | 14 | Unitaires + parsing | table-driven |
| `openrouter/client` | 6 | Integration HTTP (httptest) | table-driven |
| `openrouter/cache` | 6 | Integration fichiers | table-driven |
| `openrouter/source` | 12 | Integration (cache + reseau) | table-driven |
| `openrouter/discover` | 7 | Unitaires | table-driven |
| `openrouter/profilegen` | 8 | Unitaires + integration | table-driven |
| `openrouter/menu` | 7 | Integration (scripted stdin) | table-driven |
| `openrouter` (total) | 39 | — | — |
| `env` | 6 | Unitaires | table-driven |
| `secret` | 9 | Unitaires + integration | table-driven |
| `logging` | 5 | Integration (fichiers) | table-driven |
| `menu` | 3 | Unitaires | table-driven |
| `onboarding` | 2 | Unitaires | table-driven |
| `assets` | 5 | Integration (fichiers + embed) | table-driven |
| `dotenv` | 10 | Unitaires | table-driven |
| `tests/` | 7 | Integration end-to-end | table-driven |

### Points forts des tests

- **Tests scriptés du TUI** : `menu_test.go` injecte stdin scripté et capture stdout — excellent pattern pour tester les menus interactifs sans terminal
- **Fallback tests avec vrai processus** : `fallback_test.go` utilise `TestHelperProcess` (pattern stdlib `os/exec`) — robuste et realiste
- **Atomic write tests** : `config/wizard_test.go` verifie que le sentinel et la store sont coherents apres config/erase
- **Table-driven tests systematiques** : ~80% des tests utilisent le pattern table-driven
- **Test de non-regression CP850** : `catalog_test.go` verifie que les chaines UI sont ASCII-safe
- **Test de securite des templates** : `assets_test.go` verifie qu'aucune vraie cle API n'est embarquee dans les binaires

### Lacunes

| Lacune | Impact | Severite |
|--------|--------|----------|
| Pas de test pour `main.go` (interactive loop, flag parsing) | La logique de routage principal n'est pas testee | 🟡 |
| Pas de test pour `cli.LaunchWithFallback` avec `opts.JSON` | Le chemin JSON n'a pas de couverture de test specifique | 🟢 |
| Pas de test pour `cli.RunBeforeHooks` / `cli.RunAfterHooks` | Les hooks de cycle de vie ne sont pas testes | 🟡 |
| `tests/config_test.go` minimal (1 test) | Ce fichier semble etre un placeholder — peu de valeur ajoutee | 🟢 |
| Pas de benchmark pour `dotenv.Parse` | Le parsing .env est un goulot potentiel avec 37 profils | 🟢 |
| Couverture `openrouter.GetModels` : les chemins `network->cache write` sont testes mais pas le cas `cache write fail` | Le warning de cache non ecrit n'est pas verifiable dans les tests actuels | 🟢 |

---

## 5. Gestion des erreurs

### Verdict : Robuste

- Tous les retours d'erreur sont verifies (aucun `_ = fn()` sauf dans les chemins best-effort documentes)
- Les erreurs sont wraps avec `%w` pour la compatibilite `errors.Is`/`errors.As`
- Pas de `panic` nu — seul `catalog.Default()` panique, ce qui est documente comme une erreur de programmation (embeded YAML invalide)
- Les chemins d'echec sont explicitement geres : store indisponible degrade vers fichier clair avec warning, cache indisponible degrade vers liste embarque, etc.

### Patterns de robustesse

```
// internal/secret/store_windows.go:6-7
// "until then we use one honest, working backend instead of a half-stub"
→ Honnete sur les limitations

// internal/config/wizard.go:262-274
// store FIRST, then file — invariant documente
→ Ordre d'ecriture protege contre les crashes

// cli/launcher.go:150-155
// signal.Stop() AVANT close(sigCh) — ordre LIFO des defers
→ Prevention de panic "send on closed channel"
```

### Problemes residuels

| Fichier:ligne | Probleme | Severite |
|---------------|----------|----------|
| `logging/logger.go:56-67` | Erreur `WriteString` ignoree — certes best-effort, mais pas de diagnostic si le disque est plein | 🟢 |
| `logging/session.go:57-79` | Erreurs entierement ignorees dans `LogSession` — documente (parity PS) mais peut masquer des problemes | 🟢 |
| `cli/hooks.go:133` | Erreur `RunAfterHooks` non transmise — documente best-effort, mais le message WARN va sur stderr pendant que le processus principal continue | 🟢 |
| `env/env.go:73` | `strings.Index(kv, "=")` — si la variable d'env contient plusieurs `=`, les valeurs apres le premier `=` sont correctes mais les valeurs AVANT aussi (ex: `KEY=val=ue`) | 🟢 |

---

## 6. Documentation

### GoDoc (package-level)

| Package | Documentation | Qualite |
|---------|---------------|---------|
| `catalog` | ✅ Complete — explique le role, le mirroring PS, la data-driven philosophy | Excellente |
| `secret` | ✅ Complete — explique le threat model, les limitations, le roadmap | Excellente |
| `cli` | ✅ Complete — chaque fonction exportee documentee | Bonne |
| `cli/fallback` | ✅ Long commentaire expliquant les divergences PS | Excellente |
| `logging/session` | ✅ Compare au PS, explique l'honnete "no cost estimation" | Excellente |
| `env` | ✅ Documentee | Bonne |
| `openrouter` | ✅ Documentee | Bonne |
| `profile` | ✅ Documentee | Bonne |
| `menu` | ✅ Documentee | Correcte |
| `onboarding` | ✅ Documentee | Correcte |
| `config` | ✅ Documentee | Bonne |
| `assets` | ✅ Documentee | Correcte |
| `fsutil` | ✅ Complete — explique l'atomicite, les races, le cleanup | Excellente |
| `dotenv` | ✅ Documentee | Correcte |

### README et documentation externe

- `README.md` dans la racine du repo et `multiai-go/README.md` — ✅
- `multiai-go/docs/` — dossier present mais non audite
- `CLAUDE.md` — present, dedie a la configuration BMAD+ des agents

### Commentaires dans le code

Le code est exceptionnellement bien commente. Chaque divergence du PS est documentee, chaque invariant est explique, chaque edge case est note. Exemples :

```go
// internal/secret/secret.go:8-12
// Threat model (be honest about it): ...
```

```go
// internal/config/wizard.go:262-263
// Credential store FIRST: the file only receives the sentinel when the
// store write succeeded (invariant: sentinel in file => value in store)
```

```go
// internal/secret/crypto.go:17-20
// RESERVED, not yet wired: ...
```

---

## 7. Dependances (`go.mod`)

```
module github.com/lrochetta/multiai

go 1.22

require gopkg.in/yaml.v3 v3.0.1
```

### Analyse

| Aspect | Evaluation |
|--------|------------|
| Nombre de dependances | **1 seule** (yaml.v3) — excellent |
| Dependances transitives | 0 (yaml.v3 n'a pas de dependances) |
| Version minimale Go | 1.22 (publique, supportee) |
| Dependances non utilisees | Aucune |
| Dependances de securite | Aucune dans le graphe (pas de `golang.org/x/` ni `google.golang.org/`) |
| Taille du binaire | ~10MB (CGO_ENABLED=0, -trimpath, -s -w) |

Ceci est un point fort majeur du projet : **zero dependance externe** a part yaml.v3, qui est stable, largement audite, et sans dependances transitives.

### Recommandations

- Envisager de remplacer `gopkg.in/yaml.v3` par `gopkg.in/yaml.v3` rien a faire — il n'y a pas d'alternative plus legère qui supporte le mapping preserve-order (necessaire pour le catalogue)
- `gopkg.in/yaml.v3` a 0 CVE publiees a ce jour — la dependance est sure

---

## 8. CI/CD & Release

### GitHub Actions (`ci.yml`)

| Job | Outils | Commentaire |
|-----|--------|-------------|
| `lint` | gofmt + go vet + gosec | ✅ gosec exclut G104 (prints/exit) |
| `test` | go test -race -coverprofile (3 OS) | ✅ Ubuntu + macOS + Windows |
| `security` | govulncheck | ✅ Scan de vulnerabilites |
| `benchmark` | go test -bench=. -benchmem | ✅ Benchmarks CI |
| `build` | go build (3 OS) | ✅ Upload d'artefact |
| `release-check` | goreleaser check + snapshot | ✅ Valide la config GoReleaser |

### Release (`release.yml`)

- GoReleaser v2 avec builds cross-platform (windows/darwin/linux × amd64/arm64)
- Cosign keyless signing (OIDC via GitHub)
- GitHub build provenance attestation
- Homebrew Cask + Scoop manifest generes (skip_upload: true)
- NFPMS pour packages deb
- Preflight check pour `var version` (pas `const`)

### Points forts CI/CD

- Toutes les actions sont pinees par SHA complet (audit finding #14)
- `fail-fast: false` pour les tests multi-OS (un echec n'arrete pas les autres)
- Concurrence CI avec `cancel-in-progress: true`
- Working directory `multiai-go` pour le monorepo

### Faiblesses CI/CD

| Probleme | Severite |
|----------|----------|
| `golangci-lint` explicitement non utilise (`.golangci.yml` en v1, action rejecte le format v1) — gofmt + go vet + gosec ne couvrent pas tout (`staticcheck`, `unparam`, `prealloc`, `nakedret`) | 🟡 |
| Pas de `go mod verify` ou `go mod tidy -diff` dans la CI | 🟢 |
| Cosign signe seulement `checksums.txt`, pas les binaires individuels | 🟢 |
| Homebrew Cask et Scoop ont `skip_upload: true` — publication manuelle, pas automatisee | 🟡 |
| Pas de smoke test/fonctionnel CI (lancer `multiai version`, `multiai list`, etc.) | 🟡 |

### Packaging

- **GoReleaser** : configuration complete, bien commentee, formats tar.gz/zip, checksums SHA256
- **npm** : `packaging/npm/` — script d'installation (download binary), pas de publication automatisee
- **Debian** : `packaging/deb/` — postinst, control, build script (remplace par nfpms GoReleaser mais garde comme fallback)
- **AUR** : `packaging/aur/` — PKGBUILD, .SRCINFO
- **Homebrew/Scoop** : generes par GoReleaser dans `dist/` (skip_upload)

---

## 9. Points forts et faiblesses

### Tableau detaille

| # | Categorie | Observation | Severite | Fichier(s) concerne(s) |
|---|-----------|-------------|----------|----------------------|
| **F1** | Dependances | 1 seule dependance (yaml.v3) — zero bloat, surface d'attaque minimale | 🟢 | `go.mod` |
| **F2** | Documentation | Commentaires exceptionnels — chaque divergence PS documentee, chaque invariant explique | 🟢 | `internal/*` |
| **F3** | Securite | AES-256-GCM, zeroize plaintext, credential store sentinel, lock files | 🟢 | `internal/secret/` |
| **F4** | Tests | ~150 tests, table-driven pattern dominant, tests TUI scriptes, fallback avec vrai processus | 🟢 | `*_test.go` |
| **F5** | Robustesse | Atomic writes, graceful degradation (cache→embarque, store→plaintext warning) | 🟢 | `fsutil/`, `openrouter/source.go` |
| **F6** | Qualite Go | Idiomatique, pas de dependent injection lourde, gestion d'erreurs exemplaire | 🟢 | Tous |
| **F7** | Cross-platform | Builds Windows/macOS/Linux × amd64/arm64, platform-specific secret stores | 🟢 | `.goreleaser.yaml`, `store_*.go` |
| **F8** | Catalog data-driven | providers.yaml — ajouter un provider = editer le YAML, pas de code | 🟢 | `internal/catalog/` |
| **W1** | Pas de `golangci-lint` CI | Seulement gofmt+govet+gosec — manque staticcheck, unparam, prealloc | 🟡 | `.github/workflows/ci.yml` |
| **W2** | Code mort potentiel | `DeriveKey`/`GenerateSalt`/`pbkdf2HMACSHA256` non utilises (roadmap 1.10) | 🟢 | `internal/secret/crypto.go` |
| **W3** | Couverture `main.go` | `runInteractiveLoop()`, `runLaunch()`, flag parsing non testes | 🟡 | `cmd/multiai/main.go` |
| **W4** | CLI hooks non testes | `RunBeforeHooks`/`RunAfterHooks` — code non teste (shell escaping, exec, stderr) | 🟡 | `internal/cli/hooks.go` |
| **W5** | Pas de smoketest CI | Aucun test fonctionnel qui lance le binaire compile | 🟡 | `.github/workflows/ci.yml` |
| **W6** | Thread-safety Logger | `init()` global — `defaultLogger` partage un Mutex, mais les packages tiers ne peuvent pas creer leur propre logger | 🟢 | `internal/logging/logger.go` |
| **W7** | `commands` map globale | `register()` dans `init()` — pas testable unitairement, depend de l'ordre d'initialisation | 🟡 | `cmd/multiai/main.go:32-34` |
| **W8** | Homebrew/Scoop manuels | `skip_upload: true` — les packages Homebrew et Scoop ne sont pas publies automatiquement | 🟡 | `.goreleaser.yaml` |
| **W9** | Pas d'endpoint telemetrie | `LogSession` ecrit local JSONL — pas de metriques consolidees, pas d'alerte | 🟢 | `internal/logging/session.go` |
| **W10** | Aucun benchmark critique | Pas de benchmark pour dotenv.Parse, profile.LoadDir (37 profils), resolveStoredSecrets | 🟢 | `tests/benchmark_test.go` |

---

## 10. Recommandations techniques

### Top 10 actions concretes

| # | Action | Justification | Effort | Impact |
|---|--------|---------------|--------|--------|
| **1** | Migrer `.golangci.yml` en v2 et activer `golangci-lint` dans la CI | `gofmt + go vet` ne remplacent pas `staticcheck`, `unparam`, `prealloc`, `nakedret`. Le fichier `.golangci.yml` existe deja mais est bloque par la CI | 1h | 🔴 |
| **2** | Ajouter un smoketest dans la CI (`go build && ./multiai version && ./multiai list`) | Verifier que le binaire compile et s'execute correctement — attrape les regressions avant release | 1h | 🔴 |
| **3** | Decoupler `internal/cli/` en sous-packages | `cli/` melange 5 responsabilites : launch, fallback, display, hooks, completion | 3h | 🟡 |
| **4** | Tester `main.go` (interactive loop + flag parsing) | Extraire la logique de routage dans un package `cmd/multiai/testable` ou utiliser des tests d'integration | 4h | 🟡 |
| **5** | Tester `cli/hooks.go` | Les hooks before/after launch executent des commandes shell — doivent etre testes (injection, PATH, timeout) | 2h | 🟡 |
| **6** | Finaliser les backends natifs credential store (roadmap 1.10) | Le threat model du store AES est explicite : pas de protection contre un attaquant local. Windows Credential Manager, macOS Keychain, libsecret | 8h | 🟡 |
| **7** | Automatiser la publication Homebrew/Scoop | Creer les repositories `lrochetta/homebrew-tap` et `lrochetta/scoop-bucket`, ajouter le secret `TAP_GITHUB_TOKEN` | 2h | 🟢 |
| **8** | Ajouter `go mod verify` dans la CI | Verifier que `go.sum` est coherent — detecte les modifications non intentionnelles des dependances | 30min | 🟢 |
| **9** | Benchmark + optimisation du parsing .env | 37 profils × ~15 lignes = ~555 lignes parsees a chaque lancement — pas critique mais mesurable | 2h | 🟢 |
| **10** | Supprimer le code mort (`DeriveKey`, `GenerateSalt`, `pbkdf2HMACSHA256`, `store_linux.go` stub libsecret) | Code non utilise depuis la migration du Sprint 1. Remettre quand les backends natifs sont implementes | 30min | 🟢 |

---

## Statistiques finales

| Metrique | Valeur |
|----------|--------|
| Fichiers Go audites | 53 (30+ obligatoires couverts) |
| Lignes de code (estimation) | ~4500 LoC Go |
| Dependances externes | 1 (gopkg.in/yaml.v3) |
| Tests | ~150 tests, 2 benchmarks |
| Probleme grave | 0 |
| Probleme moyen (🟡) | 8 |
| Probleme mineur (🟢) | 11 |
| Recommandations 🔴 | 2 |
| Recommandations 🟡 | 4 |
| Recommandations 🟢 | 4 |

---

*Audit realise par Forge (Bezalel) — Agent Architecture & Developpement BMAD+*
*Relecture QA recommandee : Sentinel*
