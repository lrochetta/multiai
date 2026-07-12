---

# Stories de stabilisation CI et préparation release v0.6.0

---

## S9.1 — Fix CI: gofmt Go 1.25, compilation errors i18n/main.go

**Priorité**: BLOCKER (Sprint 0 — pre-v0.6.0)

### Objectif
Rendre la CI verte de manière fiable en éliminant les divergences de formatage gofmt entre versions Go (1.25 CI vs 1.26 local) et en garantissant que la compilation cross-platform passe sans erreur après les merges récents (i18n.go, cmd_update.go, main_test.go).

### Spécifications techniques

#### 1. Pin gofmt à la version Go de la CI

La CI utilise `go-version: '1.25'` mais `gofmt` évolue entre versions mineures de Go (les changements de formatage dans go 1.26+ peuvent produire un output différent). Solution : forcer le formatage via le binaire Go de la CI plutôt que le gofmt système.

Deux approches possibles — retenir la plus robuste :

**Approche A (recommandée) : utiliser `gofmt` du Go installé**
Remplacer dans `ci.yml` :
```yaml
- name: gofmt
  run: |
    unformatted="$(gofmt -l .)"
    if [ -n "$unformatted" ]; then
      echo "gofmt needed on:"
      echo "$unformatted"
      exit 1
    fi
```
Le `gofmt` invoqué est celui du PATH, qui est celui de `actions/setup-go` (Go 1.25). Le problème se produit uniquement si un runner a une version Go système pré-installée différente. Ajouter un guard explicite :
```yaml
- name: gofmt
  run: |
    go version
    unformatted="$(gofmt -l .)"
    if [ -n "$unformatted" ]; then
      echo "gofmt needed on:"
      echo "$unformatted"
      exit 1
    fi
```

Ajouter un pre-commit hook `.githooks/pre-commit` qui exécute `gofmt -l .` avec la version Go du projet (`go 1.24` dans go.mod) en prévention locale, documenté dans CONTRIBUTING.md.

#### 2. Vérifier les symboles i18n manquants

Après l'ajout de `cmd_update.go` et `cmd_migrate.go`, plusieurs clés i18n sont utilisées mais doivent être vérifiées :
- `cmd_update.go` utilise des chaînes FR/EN *hardcodées* (pas via i18n.T) — c'est intentionnel mais doit être audité pour s'assurer qu'au moins la version EN est disponible
- `cmd_migrate.go` utilise bien `i18n.T()` — vérifier que les clés sont toutes dans `messages[FR]` ET `messages[EN]`

Ajouter un test de compilation :
```go
// TestI18nKeysCompile vérifie que toutes les clés référencées dans le code existent
func TestI18nKeysExist(t *testing.T) {
    keys := []string{
        "store_fallback",          // utilisé dans fallback.go
        "store_already_migrated",  // utilisé dans main.go
        "store_migrate_failed",    // utilisé dans main.go
        "store_migrated",          // utilisé dans main.go
        "store_migrate_skip",      // utilisé dans main.go
    }
    for _, key := range keys {
        if i18n.T(key) == key {
            t.Errorf("i18n key %q missing in both FR and EN", key)
        }
    }
}
```

#### 3. Vérifier compilation cross-platform

Le `go build ./cmd/multiai/` compile OK sur linux/amd64, mais le `go vet ./...` peut détecter des problèmes. Ajouter un job `compile-check` dans la matrice CI (`go build ./...` sur linux + windows) après lint et avant test.

**Commande**:
```yaml
- name: Cross-compile check
  run: |
    GOOS=linux GOARCH=amd64 go build ./cmd/multiai/
    GOOS=windows GOARCH=amd64 go build ./cmd/multiai/
    GOOS=darwin GOARCH=amd64 go build ./cmd/multiai/
    GOOS=darwin GOARCH=arm64 go build ./cmd/multiai/
    GOOS=linux GOARCH=arm64 go build ./cmd/multiai/
```

### Fichiers impactés
- `.github/workflows/ci.yml` (dans `multiai-go/.github/workflows/` ET racine `.github/workflows/`)
- `multiai-go/.golangci.yml` (ajuster la config si besoin)
- `multiai-go/internal/i18n/i18n.go` (ajouter les clés manquantes si détectées)
- `multiai-go/internal/i18n/i18n_test.go` (créer : test couverture des clés)
- `multiai-go/CONTRIBUTING.md` (documenter le pre-commit hook gofmt)

### Tests
- Ajouter `TestI18nKeysExist` dans `internal/i18n/` — vérifie que toutes les clés référencées dans le code source existent dans les deux langues
- Le test de compilation cross-platform est dans le workflow CI lui-même
- Vérifier que `go vet ./...` passe sur les 3 plateformes

### Résultat attendu
- CI `lint` job passe au vert sans faux positif gofmt
- Toutes les clés i18n utilisées existent en FR et EN
- Compilation garantie sur windows/amd64, darwin/amd64, darwin/arm64, linux/amd64, linux/arm64

### Définition of Done
- [ ] `gofmt -l .` retourne vide dans le job lint CI (Go 1.25)
- [ ] `TestI18nKeysExist` passe (toutes les clés existent)
- [ ] Cross-compile check job créé et vert
- [ ] Pre-commit hook gofmt optionnel documenté dans CONTRIBUTING.md

### Risques
- Les runners GitHub Actions peuvent avoir une version Go pré-installée qui shadow `setup-go` — le `go version` dans le step le détecte immédiatement
- L'ajout de `store_fallback` dans i18n (utilisé dans `fallback.go` ligne 23) — vérifier que cette clé existe dans les deux langues (elle n'existe pas actuellement !)

### Dépendances
- Aucune

---

## S9.2 — Tests stores natifs en CI (build tags, skip conditionnel)

**Priorité**: HIGH (Sprint 0 — pre-v0.6.0)

### Objectif
Garantir que les trois stores natifs (Windows Credential Manager, macOS Keychain, Linux libsecret) sont testés automatiquement dans leurs environnements respectifs, avec des tests unitaires mockés qui tournent partout et des tests d'intégration conditionnels qui s'activent uniquement quand le backing service est disponible.

### Contexte

État actuel des tests par plateforme :

| Store | Fichier | Build tag | Tests unitaires | Tests CI |
|-------|---------|-----------|-----------------|----------|
| Encrypted file | `secret_test.go` | aucun | Oui (complets) | Oui (tous OS) |
| Windows CredMan | `store_windows.go` | `windows` | **Aucun** | **Non** |
| macOS Keychain CGo | `store_darwin_cgo.go` | `darwin && cgo` | **Aucun** | **Non** |
| macOS Keychain nocgo | `store_darwin_nocgo.go` | `darwin && !cgo` | **Aucun** | **Non** |
| Linux libsecret | `store_linux.go` | `linux` | Oui (mocks) | Oui sur ubuntu |
| Fallback wrapper | `fallback.go` | aucun | Oui | Oui (tous OS) |

### Spécifications techniques

#### 1. Tests mockés pour Windows Credential Manager

Créer `store_windows_test.go` avec `//go:build windows` :
- Mock des appels Win32 (`CredWriteW`, `CredReadW`, `CredDeleteW`, `CredEnumerateW`) via `syscall` — ou plus pragmatiquement, tester la logique parser uniquement
- Tester `extractKey()` — parsing du format `mti_<hex>_<key>` — ce test est **sans build tag** car c'est de la logique pure
- Tester `targetName()` et `serviceFilter()` — construction des chaînes
- Tester `newPlatformStore()` avec `MULTIAI_SECRETS_DIR` set → retourne file store
- Déplacer `extractKey()` et `targetName()` et `serviceFilter()` dans un fichier `store_windows_parser.go` **sans build tag** pour pouvoir les tester sur tous les OS

#### 2. Tests pour macOS Keychain

Les appels CGo ne sont pas testables facilement. Stratégie :
- Tester `parseDumpKeychain()` (dans `store_darwin.go`, accessible depuis `store_darwin_nocgo.go` aussi) — logique pure de parsing, pas de build tag nécessaire
- Ajouter un test `store_darwin_parser_test.go` sans build tag qui teste `parseDumpKeychain`, `extractQuotedValue`, `splitLines`, `joinLines`, `trimSpace`
- Tester `keychainAvailable()` avec mock de `exec.LookPath` (exiger que ce soit testable)
- CGo path : pas de test unitaire (trop complexe à mocker), mais un test d'intégration conditionnel

#### 3. Tests Linux libsecret — améliorer la couverture CI

Les tests existants dans `store_linux_test.go` sont bons (mock de `execCommand` via subprocess). Vérifier que :
- Ils tournent bien sur ubuntu-latest dans CI (oui, grâce à `//go:build linux`)
- Ajouter le test `TestNewPlatformStore_Available` (cas normal, retourne `libsecretStore`) — mais il faudrait conditionner sur la présence de `secret-tool`. Ce test est déjà géré via le mock de `secretToolLookPath`.

Problème : `TestLibsecretStore_RoundTrip` (intégration réelle) est skip quand `secret-tool` n'est pas dans PATH. C'est correct. Mais on pourrait l'activer explicitement dans CI en installant `libsecret-tools` dans un job dédié.

#### 4. CI : installer libsecret-tools sur un job dédié

Ajouter un job `test-native-stores` dans la matrice CI :
```yaml
test-native-stores:
  name: Test native stores
  needs: lint
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@...
    - uses: actions/setup-go@...
      with:
        go-version: '1.25'
    - name: Install libsecret
      run: sudo apt-get update -qq && sudo apt-get install -y -qq libsecret-tools
    - name: Test libsecret integration
      run: go test -race -v -run 'TestLibsecretStore_RoundTrip' ./internal/secret/
```

Pour Windows : impossible d'installer un service de Credential Manager en CI GitHub Actions (nécessite une session interactive). Les tests unitaires mockés suffisent.

Pour macOS : `security` CLI est toujours présente. Ajouter un test d'intégration macOS simple (vérifier que `security find-generic-password -h` marche).

### Fichiers impactés
- `multiai-go/internal/secret/store_windows.go` (extraire `extractKey`, `targetName`, `serviceFilter`)
- `multiai-go/internal/secret/store_windows_parser.go` (créer, sans build tag)
- `multiai-go/internal/secret/store_windows_test.go` (créer)
- `multiai-go/internal/secret/store_darwin_parser_test.go` (créer, sans build tag)
- `multiai-go/internal/secret/store_linux_test.go` (ajouter cas de test)
- `multiai-go/.github/workflows/ci.yml` (ajouter job test-native-stores)
- `multiai-go/.github/workflows/ci.yml` dans la racine (sync)

### Tests
- `TestExtractKey` / `TestTargetName` / `TestServiceFilter` — logique de nommage WinCred, sans build tag
- `TestParseDumpKeychain` / `TestExtractQuotedValue` — parsing macOS dump
- `TestNewPlatformStore_Windows_WithEnv` — `MULTIAI_SECRETS_DIR` force file store
- `TestSecretToolRoundTripCI` — intégration libsecret (skip si secret-tool absent)

### Résultat attendu
- Les tests de logique pure WinCred tournent sur tous les OS dans la CI
- Les tests Linux mockés tournent sur ubuntu-latest
- Le job `test-native-stores` sur ubuntu-latest installe `secret-tool` et lance le round-trip réel
- Les tests macOS `parseDumpKeychain` tournent sur macos-latest

### Définition of Done
- [ ] `extractKey`/`targetName`/`serviceFilter` en logique pure sans build tag
- [ ] Tests unitaires WinCred créés et verts sur windows-latest
- [ ] Tests parsing macOS Keychain créés et verts sur macos-latest
- [ ] Job `test-native-stores` créé dans CI avec libsecret installé
- [ ] `TestLibsecretStore_RoundTrip` passe sur ce job
- [ ] Aucune régression sur les tests existants

### Risques
- Le job `test-native-stores` nécessite `sudo apt-get install` — les runners GitHub Actions le supportent
- Windows Credential Manager ne peut pas être mocké au niveau Win32 syscall — se contenter de tester la logique de nommage
- Le test d'intégration `secret-tool` écrit/récupère/supprime un secret — pas de persistance entre runs

### Dépendances
- S9.1 (CI doit déjà être verte)

---

## S9.3 — Quality gates: govulncheck + golangci-lint bloquants

**Priorité**: HIGH (Sprint 0 — pre-v0.6.0)

### Objectif
Tous les outils de qualité (gosec, govulncheck, golangci-lint, go vet) doivent être **bloquants** dans la CI — pas de `|| true`, pas de `--issues-exit-code=0`. Une vulnérabilité ou un lint non résolu = CI rouge.

### Contexte

État actuel des quality gates dans `ci.yml` :

| Outil | Status | Exit code bloquant |
|-------|--------|-------------------|
| `gofmt` | OK | Oui (`exit 1` explicite) |
| `go vet` | OK | Oui (built-in) |
| `gosec` | **NON bloquant** | `|| true` ligne 57 |
| `golangci-lint` | **NON bloquant** | action sans `--issues-exit-code=0` explicite, mais `version: latest` peut introduire des nouvelles règles |
| `govulncheck` | Séparé | Job `security` indépendant — ne bloque pas `test` ni `build` |

### Spécifications techniques

#### 1. Rendre gosec bloquant

Supprimer le `|| true` à la ligne 57 de `ci.yml` :
```yaml
# Avant
run: go run github.com/securego/gosec/v2/cmd/gosec@latest -exclude=G104 ./... || true

# Après
run: go run github.com/securego/gosec/v2/cmd/gosec@latest -exclude=G104 ./...
```

Vérifier que le code passe gosec sans warning non-exclu. Si de nouveaux warnings apparaissent (liés aux stores natifs contenant des appels syscall/unsafe), les ajouter dans `.gosec.json` avec documentation de la décision.

#### 2. Rendre golangci-lint bloquant explicitement

`golangci-lint-action` utilise par défaut `--issues-exit-code=1` quand `--issues-exit-code` n'est pas fourni (le comportement par défaut du CLI). **Vérifier** que c'est bien le cas pour l'action `v7`. Si l'action a un comportement différent (certains modes CI passent à 0), l'expliciter :
```yaml
- name: golangci-lint
  uses: golangci/golangci-lint-action@v7
  with:
    working-directory: multiai-go
    version: latest
    args: --issues-exit-code=1
```

Problème existant : la config `.golangci.yml` est très minimaliste :
```yaml
version: "2"
linters:
  disable:
    - errcheck
run:
  timeout: 5m
```

Il faut au minimum activer les linters essentiels :
```yaml
version: "2"
linters:
  default: all
  disable:
    - errcheck      # G104 already excluded in gosec
    - wrapcheck     # trop strict pour un CLI
    - gochecknoglobals  # Version var est intentionnelle
  disable:
    - errcheck
run:
  timeout: 5m
  issues-exit-code: 1
```

**Étape préalable** : lancer `golangci-lint run ./...` en local et corriger les warnings existants AVANT de rendre le gate bloquant. Les violations détectées peuvent inclure :
- `ineffassign` : variables assignées mais non utilisées
- `unused` : fonctions/variables exportées non utilisées
- `govet` : déjà dans un step séparé mais peut être redondant

#### 3. Rendre govulncheck bloquant

Déplacer govulncheck dans la dépendance de `test` (ou le faire dépendre de `lint` ET être `needed-by` `test`). Actuellement le job `security` a `needs: lint` mais `test` a aussi `needs: lint` — sans dépendance entre `test` et `security`. Soit :

**Option A** : Fusionner govulncheck dans le job `lint` (recommandé — rapide, < 10s) :
```yaml
- name: govulncheck
  run: go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

**Option B** : Ajouter `needs: [lint, security]` au job `test` :
```yaml
test:
  name: Test (${{ matrix.os }})
  needs: [lint, security]
```

L'option A est plus simple et évite d'allonger le temps total du pipeline (govulncheck est rapide et n'a pas besoin de dépendre de `lint`). L'option B préserve la séparation des préoccupations mais allonge le temps d'exécution.

**Recommandé** : Option A — intégrer govulncheck dans le job `lint`.

#### 4. Ajouter un `Makefile` target `ci-check`

Pour faciliter la vérification locale avant push :
```makefile
ci-check: lint
	go vet ./...
	go run github.com/securego/gosec/v2/cmd/gosec@latest -exclude=G104 ./...
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...
	golangci-lint run ./...
```

### Fichiers impactés
- `multiai-go/.github/workflows/ci.yml` (supprimer `|| true`, ajouter govulncheck dans lint)
- `.github/workflows/ci.yml` (sync)
- `multiai-go/.golangci.yml` (activer `issues-exit-code: 1`, configurer les linters)
- `multiai-go/.gosec.json` (ajouter exclusions si besoin pour syscall/unsafe)
- `multiai-go/Makefile` (ajouter `ci-check` target)

### Tests
- Vérifier que `golangci-lint run ./...` passe avec `issues-exit-code: 1` **avant** de merger
- Vérifier que `gosec` passe sans le `|| true`
- Vérifier que `govulncheck` ne remonte pas de CVE connue (si oui, bloquant — priorité de fix)

### Résultat attendu
- Un warning gosec fait échouer le job `lint`
- Une vulnérabilité connue fait échouer le job `lint`
- Un lint golangci fait échouer le job `lint`
- CI entièrement bloquante sur la qualité du code

### Définition of Done
- [ ] `|| true` supprimé de la commande gosec
- [ ] `golangci-lint` configuré avec `issues-exit-code: 1`
- [ ] `govulncheck` intégré dans le job `lint`
- [ ] Aucune violation existante non résolue (golangci-lint passe)
- [ ] `ci-check` target dans le Makefile

### Risques
- `golangci-lint` peut remonter des dizaines de warnings mineurs — prévoir un correctif séparé si trop volumineux
- `gosec` peut avoir des faux positifs sur les stores natifs (unsafe, syscall) — les exclure proprement
- `govulncheck` peut remonter une vulnérabilité dans la stdlib (ex: golang.org/x/net) — nécessite `go get` de la version patchée

### Dépendances
- S9.1 (CI stable avant de rendre les gates bloquantes)
- S9.2 facultatif (les tests stores natifs n'impactent pas la qualité)

---

## S9.4 — Release v0.6.0: tag, GitHub Release, npm publish

**Priorité**: BLOCKER (release)

### Objectif
Publier la version 0.6.0 de multiai avec tous les artefacts : GitHub Release, 8 archives (5 plateformes), checksums SHA256 + signature Cosign, SBOM CycloneDX, APT repo, AUR, binaire npm, et changelog complet.

### Spécifications techniques

#### 1. Bump de version

Dans `multiai-go/cmd/multiai/main.go` :
```go
var version = "0.6.0"
```

Dans `multiai-go/Makefile` :
```makefile
VERSION = 0.6.0
```

Dans `multiai-go/.goreleaser.yaml` : pas de version hardcodée, GoReleaser utilise `{{ .Version }}` depuis le tag git.

#### 2. Mettre à jour le CHANGELOG

Ajouter une section `[multiai-go 0.6.0]` dans `CHANGELOG.md` listant toutes les stories livrées depuis v0.5.0 (26 juin — 12 juillet 2026). Les stories S5.x (stores natifs), S6.x (distribution), S7.x (qualité), S8.x (registre), S9.x (cette CI) doivent être documentées.

Organisation :
```
## [multiai-go 0.6.0] — 2026-07-12

### 🔴 Stories S5 — Stores natifs OS (5 stories)
### 🟠 Stories S8 — Registre communautaire (7 stories)
### 🟡 Stories S6 — Distribution (5 stories)
### 🟢 Stories S7 — Qualité (6 stories)
### 🔵 Stories S9 — CI/Release (5 stories)
```

#### 3. Tag et GitHub Release

Commande de release :
```bash
# Depuis master, après merge de toutes les branches de feature
git tag -a v0.6.0 -m "v0.6.0 — Ecosysteme & Distribution"
git push origin v0.6.0
```

Le workflow `release.yml` déclenché par le tag effectue :
- Preflight : vérification que le tag est sur master, `go test -race ./...`, `go vet ./...`
- GoReleaser : build des 8 archives (windows/amd64, darwin/amd64, darwin/arm64, linux/amd64, linux/arm64), checksums.txt, packages .deb, snapshot
- Attestation de provenance GitHub
- SBOM CycloneDX
- Upload APT repo vers gh-pages
- AUR : mise à jour du PKGBUILD (via `scripts/update-aur-checksums.sh`)

#### 4. npm publish

Le package npm `multiai` dans `multiai-go/packaging/npm/` doit être mis à jour et publié manuellement (pas automatisé dans release.yml par conception — voir audit C-3).

Étapes :
```bash
cd multiai-go/packaging/npm

# Mettre à jour package.json version
# Vérifier install.js pointe vers les bons assets v0.6.0

# Publier (nécessite npm login avec OTP)
npm publish --otp=<code>
```

Le `prepublishOnly` script vérifie automatiquement :
- La version n'est pas `-dev`
- Les SHA256 des assets distants correspondent
- Aucun secret/placeholder n'est exposé

**Configuration `install.js`** : vérifier que l'URL de téléchargement (format `https://github.com/lrochetta/multiai/releases/download/v<VERSION>/multiai_<VERSION>_<OS>_<ARCH>.tar.gz|.zip`) est bien correcte. Le format actuel dans `.goreleaser.yaml` est :
```
multiai_{{ .Version }}_{{ .Os }}_{{ .Arch }}.tar.gz
```
donc `multiai_0.6.0_linux_amd64.tar.gz`.

Vérifier les patterns dans `install.js` (si existant) ou `packaging/npm/install.js`.

#### 5. Vérification des artifacts après release

Après la release, vérifier :
```bash
# Télécharger et vérifier le binaire Linux
curl -LO https://github.com/lrochetta/multiai/releases/download/v0.6.0/multiai_0.6.0_linux_amd64.tar.gz
curl -LO https://github.com/lrochetta/multiai/releases/download/v0.6.0/checksums.txt
sha256sum --check --ignore-missing checksums.txt
tar xzf multiai_0.6.0_linux_amd64.tar.gz
./multiai version
# Doit afficher : multiai 0.6.0

# Vérifier le SBOM
gh release download v0.6.0 --pattern "sbom.cyclonedx.json"
# Vérifier la signature Cosign
cosign verify-blob \
  --certificate checksums.txt.pem \
  --signature checksums.txt.sig \
  --certificate-identity-regexp 'https://github.com/lrochetta/multiai' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  checksums.txt

# Vérifier le package deb
apt-get install ./multiai_0.6.0_amd64.deb
multiai version
```

#### 6. Post-release : npm installer node

Mettre à jour le package npm multi-platform :
```bash
npx multiai install
# Vérifie que le binaire Go natif est téléchargé et installé
```

### Fichiers impactés
- `multiai-go/cmd/multiai/main.go` (version bump)
- `multiai-go/Makefile` (version bump)
- `CHANGELOG.md` (nouvelle section v0.6.0)
- `multiai-go/packaging/npm/package.json` (version bump si existant)
- `multiai-go/.goreleaser.yaml` (aucun changement de config, vérifier que tout est correct)

### Tests
- Vérifier que `go build -ldflags="-X main.version=0.6.0" ./cmd/multiai/` produit un binaire avec la bonne version
- Vérifier que `goreleaser check` passe dans le job `release-check`
- Vérifier que `goreleaser build --snapshot --clean` produit les 8 archives
- Vérifier `sha256sum --check checksums.txt` sur les archives produites en snapshot

### Résultat attendu
- GitHub Release v0.6.0 créée avec tous les artefacts
- Archives téléchargeables et vérifiables
- APT repo mis à jour sur gh-pages
- AUR PKGBUILD mis à jour avec les nouveaux checksums
- Package npm publié (version + binaire Go)
- `multiai version` affiche `0.6.0` partout

### Définition of Done
- [ ] `git tag v0.6.0` pushé sur master
- [ ] GitHub Release créée avec artefacts (vérifié manuellement)
- [ ] `checksums.txt` présent et vérifiable
- [ ] SBOM attaché à la release
- [ ] APT repo mis à jour (`sudo apt install multiai` fonctionne)
- [ ] AUR PKGBUILD mis à jour
- [ ] npm package publié avec binaire Go v0.6.0
- [ ] `multiai version` retourne `0.6.0` après installation npm

### Risques
- Rate limiting GitHub API (attendre 1 min entre les requêtes de vérification)
- Le job `apt-repo` dépend de `APT_GPG_KEY` secret — vérifier qu'il est configuré
- Le Cosign signing est commenté dans `.goreleaser.yaml` — la signature est faite par le workflow (`attest-build-provenance`), pas par GoReleaser lui-même. Vérifier que le workflow ne casse pas.
- npm publish nécessite un OTP 2FA — prévoir d'être devant le terminal
- `npm publish` échoue si `prepublishOnly` détecte des problèmes — avoir les SHA256 des assets à jour

### Dépendances
- S9.1 (CI verte)
- S9.3 (quality gates verts)
- Branches de feature S5, S6, S7, S8 mergées dans master

---

## S9.5 — Homebrew/Scoop: reactivation avec TAP_GITHUB_TOKEN

**Priorité**: MEDIUM (Sprint 0 — pre-v0.6.0)

### Objectif
Réactiver la publication automatique des formules Homebrew et des manifests Scoop via GoReleaser, en configurant le secret `TAP_GITHUB_TOKEN` et en créant les repositories de tap/bucket.

### Contexte

Dans `.goreleaser.yaml`, les sections `homebrew_casks` et `scoops` sont **commentées** avec la mention :
```
# Homebrew cask and Scoop manifest — disabled until TAP_GITHUB_TOKEN secret is
# configured in repo settings.
```

Le CHANGELOG v0.5.0 mentionne "Homebrew activé" et "Scoop activé" mais en réalité ils ne le sont pas (le résultat de `brew install lrochetta/homebrew-tap/multiai` n'a jamais été testé en production).

### Spécifications techniques

#### 1. Créer les repositories GitHub

- `github.com/lrochetta/homebrew-tap` — tap Homebrew (doit commencer par `homebrew-`)
- `github.com/lrochetta/scoop-bucket` — bucket Scoop

Les repositories doivent être **publics** et initialisés avec un README.md mentionnant la procédure d'installation.

#### 2. Créer le token d'accès (TAP_GITHUB_TOKEN)

- Créer un Personal Access Token (classic) avec scope `repo` sur un compte qui a push access aux deux repositories
- **Compte technique** : créer un utilisateur dédié `multiai-bot` (ou utiliser le compte laurent)
- Ajouter le token dans les secrets du repository `lrochetta/multiai` → Settings → Secrets and variables → Actions → New repository secret
- Nom du secret : `TAP_GITHUB_TOKEN`

#### 3. Décommenter et configurer `.goreleaser.yaml`

Section Homebrew :
```yaml
brews:
  - name: multiai
    repository:
      owner: lrochetta
      name: homebrew-tap
    commit_msg_template: "chore: update multiai formula to {{ .Tag }}"
    homepage: "https://rochetta.fr"
    description: "Route multiple AI CLIs with isolated env profiles"
    license: "MIT"
    skip_upload: false
    install: |
      bin.install "multiai"
    test: |
      system "#{bin}/multiai", "version"
    # Caveats / deprecation ne sont pas nécessaires pour v1
```

Section Scoop :
```yaml
scoops:
  - name: multiai
    repository:
      owner: lrochetta
      name: scoop-bucket
    commit_msg_template: "chore: update multiai manifest to {{ .Tag }}"
    homepage: "https://rochetta.fr"
    description: "Route multiple AI CLIs with isolated env profiles"
    license: "MIT"
    skip_upload: false
    persist: []
```

**Important** : GoReleaser v2 utilise `brews` et `scoops` (pas `homebrew_casks` comme mentionné dans le commentaire — vérifier la version GoReleaser dans le workflow : `version: "~> v2"`). Depuis GoReleaser 2.10, `brews` est déprécié et les casks sont le nouveau standard. Le commentaire dans `.goreleaser.yaml` est obsolète — vérifier la documentation de GoReleaser v2.x.

#### 4. Mettre à jour le workflow release.yml

Ajouter l'injection du token :
```yaml
- name: Run GoReleaser
  uses: goreleaser/goreleaser-action@...
  with:
    distribution: goreleaser
    version: "~> v2"
    workdir: multiai-go
    args: release --clean
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    TAP_GITHUB_TOKEN: ${{ secrets.TAP_GITHUB_TOKEN }}
```

#### 5. Tester en snapshot

Avant la release réelle, tester avec `goreleaser release --snapshot --clean` et inspecter `dist/` :
```bash
cd multiai-go
goreleaser release --snapshot --clean 2>&1 | tail -50
ls dist/homebrew/  # Doit contenir multiai.rb
ls dist/scoop/     # Doit contenir multiai.json
```

### Fichiers impactés
- `multiai-go/.goreleaser.yaml` (décommenter et configurer brews + scoops)
- `multiai-go/.github/workflows/release.yml` (ajouter TAP_GITHUB_TOKEN aux env)
- `.github/workflows/release.yml` (sync)
- `docs/guide/installation.md` (mettre à jour les commandes Homebrew/Scoop si besoin)

### Tests
- `goreleaser check` — validation de la config
- `goreleaser release --snapshot --clean` — vérifier que les fichiers sont bien générés dans `dist/`
- Vérifier le contenu des fichiers générés (`dist/homebrew/multiai.rb`, `dist/scoop/multiai.json`) — le SHA256 doit être présent et correct
- Test d'installation local :
  ```bash
  # Homebrew
  brew tap lrochetta/homebrew-tap
  brew install multiai
  multiai version
  
  # Scoop
  scoop bucket add multiai https://github.com/lrochetta/scoop-bucket
  scoop install multiai
  multiai version
  ```

### Résultat attendu
- GoReleaser pousse automatiquement la formule Homebrew vers `lrochetta/homebrew-tap` à chaque release
- GoReleaser pousse automatiquement le manifest Scoop vers `lrochetta/scoop-bucket` à chaque release
- Les SHA256 sont corrects (vérifiés par GoReleaser)
- Les utilisateurs peuvent installer via `brew install lrochetta/homebrew-tap/multiai`

### Définition of Done
- [ ] Repos `homebrew-tap` et `scoop-bucket` créés et publics
- [ ] `TAP_GITHUB_TOKEN` secret configuré dans les settings du repo
- [ ] Sections `brews` et `scoops` décommentées et configurées dans `.goreleaser.yaml`
- [ ] `TAP_GITHUB_TOKEN` injecté dans le workflow `release.yml`
- [ ] Snapshot test validé (formule + manifest générés correctement)
- [ ] `goreleaser check` passe
- [ ] Documentation d'installation mise à jour

### Risques
- `TAP_GITHUB_TOKEN` nécessite un PAT (Personal Access Token) classic — les fine-grained tokens ne supportent pas le scope `repo` complet. Créer un classic token.
- Le compte propriétaire du PAT doit avoir `Write` accès au repo `homebrew-tap` et `scoop-bucket`
- GoReleaser v2 peut avoir changé l'API pour `brews` vs `casks` — **vérifier** la documentation officielle `goreleaser.com` avant de décommenter
- Homebrew casks vs brews : le commentaire dit "casks are the supported way to ship prebuilt binaries" mais la syntaxe exacte a changé entre v1 et v2

### Dépendances
- Création des repos `homebrew-tap` et `scoop-bucket` (prérequis manuel par laurent)
- Génération du PAT classic (prérequis manuel)
- S9.4 (la release doit être prête — mais peut être testée en snapshot avant)

---

## Annexes

### Dépendances entre stories

```
S9.1 (Fix CI)
  ├── S9.3 (Quality gates) → dépend de S9.1 (CI stable avant d'activer les gates)
  └── S9.2 (Tests natifs)  → dépend de S9.1 (CI stable)
  
S9.4 (Release) → dépend de S9.1 + S9.3 (CI verte + quality gates bloquants)
S9.5 (Homebrew/Scoop) → dépend de S9.4
```

### Priorité d'exécution recommandée

1. **S9.1** — Fix gofmt + compilation + i18n (BLOCKER : CI cassée → tout bloqué)
2. **S9.3** — Quality gates blocquants (HIGH : sécurise la CI)
3. **S9.2** — Tests stores natifs (HIGH : couverture CI des stores)
4. **S9.5** — Homebrew/Scoop (MEDIUM : peut être fait en parallèle de S9.4)
5. **S9.4** — Release v0.6.0 (BLOCKER : dernière étape)

### Timeline estimée

| Story | Effort | Risque |
|-------|--------|--------|
| S9.1 | 2h | Faible — corrections mineures |
| S9.2 | 4h | Moyen — création de tests + CI job |
| S9.3 | 3h | Faible — principalement de la config |
| S9.4 | 2h | Faible — procédure bien rodée |
| S9.5 | 3h | Moyen — dépend de création de repos + token |

### Ce qui n'est PAS dans le scope

- Dépôt dédié au registre communautaire (S8.1 — déjà en cours)
- Badges Codecov/Go Report Card (S8.7 — priorité séparée)
- Validation automatique CI des profils soumis (S8.2 — nécessite S8.1 fini)