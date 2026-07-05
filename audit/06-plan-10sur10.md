# Plan 10/10 — AI CLI Launcher (multiai)

**Objectif** : Amener les 4 dimensions de l'audit à 10/10  
**État initial** : 5.25/10 (moyenne pondérée : Sécurité 4, Qualité 6.5, Architecture 5.5, DX 5)  
**Durée estimée** : 6 phases sur ~6 mois (pour un développeur solo à temps partiel)  
**Principe directeur** : Chaque phase incrémente la note visée d'au moins 1 point sur chaque dimension

---

## Vision 10/10

| Dimension | Aujourd'hui | Cible | Delta |
|---|---|---|---|
| Sécurité | 4/10 | 10/10 | +6 |
| Qualité Code | 6.5/10 | 10/10 | +3.5 |
| Architecture | 5.5/10 | 10/10 | +4.5 |
| DX | 5/10 | 10/10 | +5 |

**Décision architecturale clé** : Réécriture en **Go** en Phase 3. C'est le pivot qui débloque simultanément la sécurité (credential store natif), l'architecture (binaire unique multi-plateforme), la qualité (typage fort, tests natifs), et la DX (une seule commande `go install`, plus de pwsh).

---

# Phase 1 — Fondations (2 semaines)

**Objectif** : Sécurité 7/10, Qualité 8/10, Architecture 6/10, DX 7/10  
**Portée** : Corrections critiques sans changement d'architecture

## Semaine 1 — Sécurité urgente

### 1.1 Protéger les secrets sur disque
- [ ] Ajouter `configs/profiles/*.env` dans `.gitignore` **(CRITIQUE)**
- [ ] Ajouter `.env` dans `.npmignore` pour exclure de la publication npm
- [ ] Ajouter un script `prepublishOnly` dans `package.json` qui scanne les `.env` et bloque si une vraie clé est détectée (pas de placeholder)
- [ ] Dans `install.ps1` et `install.sh`, après copie : `chmod 600` sur tous les `.env` (Linux/macOS) et `icacls` restrictif (Windows)

### 1.2 Durcir l'exécution
- [ ] Valider `COMMAND` avant exécution (whitelist : `claude`, `codex`, `opencode`)
- [ ] Si `COMMAND` n'est pas dans la whitelist → refuser de lancer avec un message explicite
- [ ] Ajouter option `-AllowCustomCommand` pour les utilisateurs avancés

### 1.3 Corriger les bugs bloquants
- [ ] `Read-DotEnvFile` : supporter le préfixe `export` (format Unix standard)
- [ ] `Split-ArgsSimple` : implémenter un parseur respectant les guillemets
- [ ] `Clear-RouterEnvironment` : étendre `$KnownEnvVars` avec `AWS_*`, `AZURE_*`, `GITHUB_TOKEN`, `NPM_TOKEN`, `SSH_*`, etc.

## Semaine 2 — Assainissement

### 1.4 Unifier le nommage
- [ ] Remplacer toutes les occurrences de `aicode` → `multiai` dans :
  - `docs/COMMANDS.md` (21 occurrences)
  - `code-router.ps1` (lignes 225, 288)
  - `install.sh` (lignes 122-124)
  - `install.ps1` (messages post-install)
- [ ] Renommer `GEMINI.md` → supprimer ou remplacer par un fichier qui explique pourquoi Gemini a été retiré
- [ ] Vérifier que `AGENTS.md` et `CLAUDE.md` ne sont pas des doublons inutiles

### 1.5 Ajouter l'infrastructure de qualité minimale
- [ ] Créer `.editorconfig` (PowerShell + JavaScript + Markdown)
- [ ] Ajouter `scripts` dans `package.json` : `lint`, `test`, `prepublishOnly`
- [ ] Ajouter `devDependencies` : `pester` (tests PowerShell), `markdownlint-cli`
- [ ] Créer `tests/unit/RouterFunctions.Tests.ps1` avec Pester pour :
  - `Test-IsPlaceholder` (10+ cas)
  - `Read-DotEnvFile` (8+ cas : standard, avec export, commentaires, guillemets, vide)
  - `Split-ArgsSimple` (6+ cas : simple, avec guillemets, imbriqué)
  - `Expand-RouterValue` (5+ cas)
- [ ] Créer `.github/workflows/test.yml` :
  ```yaml
  name: Test
  on: [push, pull_request]
  jobs:
    test-powershell:
      runs-on: ${{ matrix.os }}
      strategy:
        matrix:
          os: [windows-latest, ubuntu-latest, macos-latest]
      steps:
        - uses: actions/checkout@v4
        - name: Run Pester tests
          run: pwsh -Command "Install-Module Pester -Force; Invoke-Pester tests/"
    lint:
      runs-on: ubuntu-latest
      steps:
        - uses: actions/checkout@v4
        - run: npx markdownlint-cli *.md docs/*.md
  ```

### 1.6 Documentation immédiate
- [ ] Ajouter section **Troubleshooting** dans README.md (5 erreurs courantes + solutions)
- [ ] Ajouter section **FAQ** (5 questions)
- [ ] Créer `CHANGELOG.md` avec l'historique connu (v0.1.0 → v0.1.5)
- [ ] Créer `CONTRIBUTING.md` (comment installer, tester, build, submit PR)
- [ ] Décider une langue unique pour la documentation → **Recommandation : tout en anglais**

### Livrables Phase 1
```
✅ Sécurité 4→7 : .gitignore, whitelist COMMAND, chmod 600, prepublishOnly
✅ Qualité 6.5→8 : bugs corrigés, tests Pester, CI/CD, linting
✅ Architecture 5.5→6 : nommage unifié, split-args robuste
✅ DX 5→7 : troubleshooting, FAQ, CHANGELOG, CONTRIBUTING, nom unique
```

---

# Phase 2 — Robustesse et finition PowerShell (3 semaines)

**Objectif** : Sécurité 8/10, Qualité 9/10, Architecture 7/10, DX 8/10

## Semaine 3 — Refactoring PowerShell

### 2.1 Modulariser code-router.ps1
- [ ] Extraire `Read-DotEnvFile` → `lib/DotEnv.psm1` (module)
- [ ] Extraire `Get-Profiles`, `Find-Profile`, `Select-Tool`, `Select-Profile` → `lib/ProfileManager.psm1`
- [ ] Extraire `Apply-ProfileEnv`, `Clear-RouterEnvironment`, `Expand-RouterValue` → `lib/EnvManager.psm1`
- [ ] Extraire `Show-ConfigMenu`, `Invoke-ConfigureProvider`, `Set-ProfileSecret` → `lib/ConfigWizard.psm1`
- [ ] Extraire `Show-TopMenu`, `Show-BmadMenu` → `lib/Menu.psm1`
- [ ] Garder `code-router.ps1` comme point d'entrée fin (100-150 lignes max)
- [ ] Ajouter `using module` au lieu de dot-sourcing

### 2.2 Améliorer la gestion d'erreurs
- [ ] Remplacer les `throw` nus par `try/catch` avec messages utilisateur formatés
- [ ] Ajouter `-ErrorAction Continue` sur les opérations non critiques
- [ ] Codes de sortie discriminants :
  - `0` : succès
  - `1` : erreur utilisateur (profil introuvable, CLI manquant, clé manquante)
  - `2` : erreur configuration (fichier .env corrompu)
  - `3` : erreur système (permissions, dossier inaccessible)
  - `4` : erreur processus enfant (CLI crash)
- [ ] Logger les erreurs dans `$env:TEMP/multiai-error.log` avec timestamp

### 2.3 Ajouter le mode non-interactif complet
- [ ] `multiai -Profile ds -Json` → sortie JSON structurée :
  ```json
  {
    "profile": "claude-deepseek-v4-pro",
    "shortcut": "ds",
    "tool": "claude",
    "command": "claude",
    "env": {"ANTHROPIC_BASE_URL": "...", "ANTHROPIC_MODEL": "deepseek-v4-pro[1m]"},
    "status": "launched",
    "pid": 12345
  }
  ```
- [ ] `multiai -Profile ds -DryRun` → simule sans lancer, retourne ce qui serait exécuté
- [ ] `multiai -Profile ds -Quiet` → pas de output sauf erreurs

## Semaine 4 — Sécurité avancée

### 2.4 Isolation d'environnement renforcée
- [ ] Remplacer `Clear-RouterEnvironment` par une liste blanche au lieu d'une liste noire :
  ```powershell
  $AllowedEnvVars = @('PATH', 'HOME', 'USER', 'USERPROFILE', 'TEMP', 'TMP',
                       'SHELL', 'LANG', 'LC_ALL', 'DISPLAY', 'WAYLAND_DISPLAY',
                       'TERM', 'COLORTERM', 'SSH_AUTH_SOCK')
  # Supprimer TOUT sauf la liste blanche
  Get-ChildItem Env: | Where-Object { $_.Name -notin $AllowedEnvVars } |
    Remove-Item -ErrorAction SilentlyContinue
  ```
- [ ] Ajouter une option `-PreserveEnv` pour garder l'environnement complet (opt-in)

### 2.5 Protection des secrets en mémoire
- [ ] Utiliser `SecureString` pour stocker les clés API après lecture :
  ```powershell
  $secureKey = ConvertTo-SecureString $value -AsPlainText -Force
  ```
- [ ] Injecter les clés dans le processus enfant via un pipe anonyme temporaire plutôt que via les variables d'environnement (quand le CLI le supporte)
- [ ] Alternative : créer un fichier temporaire avec `chmod 600`, le passer au CLI, le supprimer immédiatement après lancement

### 2.6 Vérification d'intégrité
- [ ] Ajouter un hash SHA256 de `code-router.ps1` et des modules dans le dépôt
- [ ] Vérifier le hash au lancement (alerter si modifié)
- [ ] Signer les scripts PowerShell avec un certificat code-signing (optionnel, nécessite un certificat)

## Semaine 5 — Tests et finition

### 2.7 Couverture de tests étendue
- [ ] Tests unitaires : toutes les fonctions pures (100% coverage cible)
- [ ] Tests d'intégration :
  - Créer un dossier temporaire avec des faux profils .env
  - Tester `Get-Profiles` → parsing correct
  - Tester `Apply-ProfileEnv` → variables injectées dans le scope
  - Tester `Find-Profile` → match exact, partiel, inexistant
  - Tester `Test-RequiredSecrets` → placeholder détecté, clé valide
- [ ] Tests end-to-end :
  - `multiai -Profile test-dummy -NoLaunch` → pas d'erreur
  - `multiai -List` → sortie contient les profils
  - `multiai -List -Json` → JSON valide

### 2.8 Linting et qualité
- [ ] Ajouter `PSScriptAnalyzer` dans la CI
- [ ] Corriger tous les warnings PSScriptAnalyzer
- [ ] Ajouter `shellcheck` pour `install.sh`
- [ ] Ajouter `eslint` pour `bin/multiai.js`

### Livrables Phase 2
```
✅ Sécurité 7→8 : SecureString, liste blanche env, vérification intégrité
✅ Qualité 8→9 : code modulaire, tests intégration/E2E, PSScriptAnalyzer
✅ Architecture 6→7 : modules PowerShell, mode non-interactif complet
✅ DX 7→8 : sortie JSON, DryRun, messages d'erreur utilisateur propres
```

---

# Phase 3 — Réécriture Go (6 semaines) 🔄 PIVOT

**Objectif** : Sécurité 9/10, Qualité 9/10, Architecture 9/10, DX 9/10  
**Décision** : Réécrire le cœur du routeur en Go. Le code PowerShell est gelé en maintenance.

## Pourquoi Go
| Besoin | Go | PowerShell actuel |
|---|---|---|
| Multi-plateforme natif | ✅ `go build` → 1 binaire par OS/arch | ❌ Nécessite pwsh sur macOS/Linux |
| Zéro dépendance runtime | ✅ Binaire statique | ❌ PowerShell 5.1+ ou pwsh |
| Gestion sécurisée des secrets | ✅ `golang.org/x/term`, keychain | ❌ Env vars uniquement |
| Tests | ✅ `go test` natif, race detector | ⚠️ Pester (Windows-centric) |
| Performance | ✅ Compilé, goroutines | ❌ Interprété, mono-thread |
| Cross-compilation | ✅ `GOOS=linux GOARCH=arm64 go build` | ❌ Non applicable |
| Distribution | ✅ `go install`, `brew`, `scoop`, `apt` | ⚠️ npm + PowerShell |

## Semaine 6-7 — Fondations Go

### 3.1 Structure du projet Go
```
multiai/
├── cmd/
│   └── multiai/
│       └── main.go              // Point d'entrée
├── internal/
│   ├── profile/
│   │   ├── profile.go           // Structure Profile
│   │   ├── loader.go            // Chargement depuis .env / YAML
│   │   ├── validator.go         // Validation de schéma
│   │   └── manager.go           // CRUD profils
│   ├── env/
│   │   ├── env.go               // Gestion environnement isolé
│   │   ├── cleanup.go           // Nettoyage liste blanche
│   │   └── injector.go          // Injection processus enfant
│   ├── cli/
│   │   ├── launcher.go          // Lancement processus
│   │   ├── adapter.go           // Interface CLIs (claude, codex, opencode)
│   │   └── args.go              // Parsing d'arguments avec guillemets
│   ├── config/
│   │   ├── wizard.go            // Menu interactif de config
│   │   ├── provider.go          // Catalogue de fournisseurs
│   │   └── secret.go            // Gestion sécurisée des secrets
│   ├── menu/
│   │   ├── top.go               // Menu principal
│   │   └── select.go            // Sélection outil/profil
│   ├── install/
│   │   ├── install.go           // Logique d'installation
│   │   └── path.go              // Gestion PATH cross-platform
│   └── update/
│       └── update.go            // Auto-update
├── pkg/
│   └── dotenv/
│       └── dotenv.go            // Parseur .env (export standard)
├── configs/
│   └── profiles/                // Les 17 profils .env (conservés)
├── docs/
│   └── ...                      // Documentation
├── tests/
│   ├── profile_test.go
│   ├── env_test.go
│   ├── dotenv_test.go
│   └── integration_test.go
├── .goreleaser.yml              // Release multi-plateforme
├── go.mod
├── go.sum
├── README.md
├── CHANGELOG.md
├── CONTRIBUTING.md
└── Makefile
```

### 3.2 Structures de données fondamentales
```go
// Profile représente un profil de lancement
type Profile struct {
    ID          string            `yaml:"id" json:"id"`
    Shortcut    string            `yaml:"shortcut" json:"shortcut"`
    Tool        string            `yaml:"tool" json:"tool"`        // claude, codex, opencode
    DisplayName string            `yaml:"display_name" json:"display_name"`
    Description string            `yaml:"description" json:"description,omitempty"`
    Order       int               `yaml:"order" json:"order"`
    Command     string            `yaml:"command" json:"command"`
    Args        []string          `yaml:"args" json:"args,omitempty"`
    Env         map[string]string `yaml:"env" json:"env"`
    ClearEnv    bool              `yaml:"clear_env" json:"clear_env"`
    Secrets     []string          `yaml:"required_secrets" json:"required_secrets,omitempty"`
}

// Provider définit un fournisseur de clés API
type Provider struct {
    ID        string              `yaml:"id"`
    Display   string              `yaml:"display"`
    URL       string              `yaml:"url"`          // URL page de création de clé
    Shortcuts []string            `yaml:"shortcuts"`    // Profils associés
    VarMap    map[string]string   `yaml:"var_map"`      // shortcut → variable d'env
    Note      string              `yaml:"note,omitempty"`
}
```

### 3.3 Parseur .env robuste
```go
// Parse lit un fichier .env et retourne un map[string]string
// Supporte :
//   - export KEY=value
//   - KEY="value with spaces"
//   - KEY='single quoted'
//   - # commentaires
//   - lignes vides
//   - valeurs multilignes (guillemets non fermés = continuation)
func Parse(r io.Reader) (map[string]string, error)
```

### 3.4 Gestion sécurisée des secrets
```go
// CredentialStore abstrait le stockage sécurisé par plateforme
type CredentialStore interface {
    Get(service, key string) (string, error)
    Set(service, key, value string) error
    Delete(service, key string) error
}

// Implémentations :
// - Windows: wincred (Windows Credential Manager)
// - macOS:   keychain (macOS Keychain)
// - Linux:   freedesktop secret-service (ou fallback fichier chiffré)
```

## Semaine 8-9 — Logique métier

### 3.5 Lancement de processus isolé
```go
// Launch lance un CLI avec environnement isolé
func Launch(profile Profile, extraArgs []string) error {
    // 1. Construire environnement vierge (liste blanche uniquement)
    env := buildCleanEnv()
    // 2. Ajouter les variables du profil
    for k, v := range profile.Env {
        env[k] = v
    }
    // 3. Résoudre les placeholders %VAR%
    env = expandEnvVars(env)
    // 4. Vérifier les secrets requis
    if err := validateSecrets(profile, env); err != nil {
        return fmt.Errorf("secret manquant: %w", err)
    }
    // 5. Récupérer les secrets du credential store si non définis dans .env
    // 6. Lancer le processus
    cmd := exec.Command(profile.Command, append(profile.Args, extraArgs...)...)
    cmd.Env = mapToEnvList(env)
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
}
```

### 3.6 Dérivation automatique du ProviderCatalog
```go
// BuildProviderCatalog scanne les profils et construit le catalogue automatiquement
// Plus de double source de vérité : tout vient des fichiers .env
func BuildProviderCatalog(profiles []Profile) map[string]Provider {
    catalog := make(map[string]Provider)
    for _, p := range profiles {
        // Déduire le provider depuis les variables d'env
        provider := detectProvider(p)
        catalog[provider.ID] = provider
    }
    return catalog
}
```

### 3.7 Commande `multiai` unifiée
```go
// Usage:
//   multiai                          # menu interactif
//   multiai launch                   # menu lancement
//   multiai launch -p ds             # lancement direct
//   multiai launch -p ds -- --dangerously-skip-permissions
//   multiai config                   # config clés
//   multiai config --provider deepseek
//   multiai list                     # liste profils
//   multiai list --json
//   multiai install                  # installation
//   multiai install --dir /opt/multiai
//   multiai update                   # mise à jour automatique
//   multiai version                  # version
```

## Semaine 10-11 — CLI, packaging, tests

### 3.8 Distribution
- [ ] `.goreleaser.yml` configuré pour :
  - Windows: `amd64`, `arm64` (`.exe` + scoop manifest)
  - macOS: `amd64`, `arm64` (`.tar.gz` + brew formula)
  - Linux: `amd64`, `arm64` (`.tar.gz` + `.deb` + `.rpm` + AUR)
- [ ] `go install github.com/lrochetta/multiai@latest` fonctionnel
- [ ] `brew install lrochetta/tap/multiai`
- [ ] `scoop bucket add lrochetta/multiai`
- [ ] Garder le package npm `multiai` comme alias → télécharge le binaire Go si disponible, sinon message

### 3.9 Tests Go complets
```go
// tests/dotenv_test.go - 20+ cas
func TestParse_Standard(t *testing.T) { ... }
func TestParse_Export(t *testing.T) { ... }
func TestParse_QuotedValues(t *testing.T) { ... }
func TestParse_Comments(t *testing.T) { ... }
func TestParse_Multiline(t *testing.T) { ... }
func TestParse_EmptyFile(t *testing.T) { ... }

// tests/profile_test.go - 15+ cas
func TestLoadProfiles_All(t *testing.T) { ... }
func TestLoadProfiles_MissingDir(t *testing.T) { ... }
func TestFindProfile_ExactMatch(t *testing.T) { ... }
func TestFindProfile_PartialMatch(t *testing.T) { ... }
func TestBuildProviderCatalog(t *testing.T) { ... }

// tests/env_test.go - 10+ cas
func TestBuildCleanEnv(t *testing.T) { ... }
func TestExpandEnvVars(t *testing.T) { ... }
func TestValidateSecrets(t *testing.T) { ... }

// tests/integration_test.go - 5+ cas
func TestLaunch_DryRun(t *testing.T) { ... }
func TestLaunch_NoLaunch(t *testing.T) { ... }
```

### 3.10 Migration automatique
- [ ] Commande `multiai migrate` : détecte l'ancienne installation PowerShell, migre les profils .env, supprime les wrappers .cmd/.sh obsolètes
- [ ] Message post-install clair pour les utilisateurs existants

### Livrables Phase 3
```
✅ Sécurité 8→9 : credential store natif, env vierge, plus de fuite mémoire
✅ Qualité 9→9 : Go type-safe, go test, race detector
✅ Architecture 7→9 : binaire unique, modules clairs, ProviderCatalog dérivé
✅ DX 8→9 : go install, brew, scoop, migration auto, commande unifiée
```

---

# Phase 4 — Industrialisation (3 semaines)

**Objectif** : Sécurité 10/10, Qualité 10/10, Architecture 9/10, DX 9/10

## Semaine 12 — Sécurité maximale

### 4.1 Chiffrement au repos
- [ ] Implémenter `SecretManager` avec chiffrement AES-256-GCM
- [ ] Clé de chiffrement dérivée de la machine (DPAPI Windows, Keychain macOS, libsecret Linux)
- [ ] Les fichiers .env ne contiennent JAMAIS de clés en clair
- [ ] `multiai config` stocke automatiquement dans le credential store
- [ ] Export/import chiffré des profils (`multiai export --encrypt`)

### 4.2 Supply chain
- [ ] Signer les binaires avec Cosign (Sigstore)
- [ ] Générer SBOM (Software Bill of Materials) avec Syft à chaque release
- [ ] `multiai update` vérifie la signature avant d'installer
- [ ] `.github/workflows/release.yml` :
  ```yaml
  - goreleaser
  - cosign sign
  - syft sbom
  - upload to GitHub Releases
  - update scoop/brew/homebrew taps
  ```

### 4.3 Audit et conformité
- [ ] Intégrer `gosec` dans la CI (analyse statique de sécurité)
- [ ] Intégrer `govulncheck` (vulnérabilités connues dans les dépendances)
- [ ] Politique de sécurité (`SECURITY.md`) :
  - Comment signaler une vulnérabilité
  - Délai de réponse (48h)
  - Versions supportées
- [ ] Exécuter un audit externe (ou simulé par un agent) avant chaque release majeure

## Semaine 13 — Qualité maximale

### 4.4 CI/CD complet
```yaml
# .github/workflows/ci.yml
name: CI
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: golangci/golangci-lint-action@v6
      - run: go vet ./...
      - uses: securego/gosec@master

  test:
    needs: lint
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
        go: ['1.22', '1.23']
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '${{ matrix.go }}' }
      - run: go test -race -coverprofile=coverage.out ./...
      - run: go test -bench=. ./...  # benchmarks
      - uses: codecov/codecov-action@v4

  security:
    needs: lint
    runs-on: ubuntu-latest
    steps:
      - run: govulncheck ./...
      - run: gosec ./...

  build:
    needs: test
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
        arch: [amd64, arm64]
    steps:
      - run: make build
      - uses: actions/upload-artifact@v4
```

### 4.5 Métriques de qualité
- [ ] Couverture de test : cible ≥ 90%
- [ ] `golangci-lint` : 0 warning
- [ ] `gosec` : 0 issue haute/critique
- [ ] `govulncheck` : 0 vulnérabilité connue
- [ ] Benchmark : lancement de profil < 50ms

## Semaine 14 — Configuration déclarative

### 4.6 Format de profil YAML (en plus du .env)
```yaml
# configs/profiles/claude-deepseek-v4-pro.yaml
id: claude-deepseek-v4-pro
shortcut: ds
tool: claude
display_name: DeepSeek V4 Pro 1M
description: DeepSeek V4 Pro 1M via endpoint Anthropic-compatible
order: 40
command: claude
env:
  CLAUDE_CONFIG_DIR: "${HOME}/.claude-deepseek-v4pro"
  ANTHROPIC_BASE_URL: "https://api.deepseek.com/anthropic"
  ANTHROPIC_MODEL: deepseek-v4-pro[1m]
  ANTHROPIC_DEFAULT_OPUS_MODEL: deepseek-v4-pro[1m]
  ANTHROPIC_DEFAULT_SONNET_MODEL: deepseek-v4-pro[1m]
  ANTHROPIC_DEFAULT_HAIKU_MODEL: deepseek-v4-flash
  CLAUDE_CODE_SUBAGENT_MODEL: deepseek-v4-flash
  CLAUDE_CODE_AUTO_COMPACT_WINDOW: 1000000
  API_TIMEOUT_MS: 3000000
  CLAUDE_CODE_EFFORT_LEVEL: max
clear_env: true
required_secrets:
  - ANTHROPIC_AUTH_TOKEN
provider: deepseek   # ← clé étrangère vers le catalogue provider
```

- [ ] Support des deux formats (.env ET .yaml), migration automatique `.env` → `.yaml`
- [ ] Validation de schéma JSON Schema : `multiai validate` vérifie tous les profils

### 4.7 Profils par projet
```yaml
# .multiai.yaml à la racine d'un projet
extends: ds                    # hérite du profil global "ds"
overrides:
  CLAUDE_CONFIG_DIR: "${HOME}/.myproject-claude"
  ANTHROPIC_MODEL: deepseek-v4-flash  # moins cher pour ce projet
```

### Livrables Phase 4
```
✅ Sécurité 9→10 : chiffrement au repos, cosign, SBOM, gosec, govulncheck
✅ Qualité 9→10 : CI/CD complet, ≥90% coverage, linting 0 warning
✅ Architecture 9→9 : config YAML, profils par projet, schema validation
✅ DX 9→9 : export chiffré, validate, .multiai.yaml local
```

---

# Phase 5 — Excellence DX (2 semaines)

**Objectif** : Architecture 10/10, DX 10/10

## Semaine 15 — Documentation de classe mondiale

### 5.1 Site de documentation
- [ ] Choisir un générateur statique (Docusaurus, VitePress, ou mdBook)
- [ ] Structure :
  - **Getting Started** : installation par plateforme, premier lancement, configuration des clés
  - **Guides** : comment ajouter un nouveau fournisseur, migrer depuis l'ancienne version, CI/CD
  - **Reference** : chaque commande documentée avec flags, exemples, codes de sortie
  - **Profiles** : catalogue visuel de tous les profils disponibles
  - **FAQ** : 20+ questions réelles
  - **Troubleshooting** : 15+ erreurs communes avec solutions
- [ ] Captures d'écran / GIFs du menu interactif
- [ ] Vidéo "Get started in 2 minutes" (optionnel)

### 5.2 Documentation intégrée
- [ ] `multiai help` → aide riche avec exemples
- [ ] `multiai help launch` → aide spécifique
- [ ] `multiai help config` → aide spécifique
- [ ] Messages d'erreur avec `--help` suggéré automatiquement
- [ ] `multiai doctor` → diagnostique l'installation, vérifie les prérequis, suggère des corrections

### 5.3 Programme de feedback
- [ ] Issue templates GitHub (bug report, feature request, profile request)
- [ ] `multiai feedback` → ouvre un sondage rapide ou le template issue
- [ ] Discord ou GitHub Discussions pour la communauté

## Semaine 16 — Extensibilité et écosystème

### 5.4 Plugin hooks
```go
// Hooks permettent d'injecter du comportement avant/après le lancement
type Hooks struct {
    BeforeLaunch func(profile Profile) error   // ex: vérifier VPN
    AfterLaunch  func(profile Profile, err error)  // ex: notifier Slack
}
// Configurable dans .multiai.yaml :
// hooks:
//   before_launch:
//     - command: "vpn-check.sh"
//     - command: "notify.sh 'Starting {{.Profile.DisplayName}}'"
```

### 5.5 Registre communautaire de profils
- [ ] Dépôt GitHub `multiai-profiles` avec profils contribués par la communauté
- [ ] `multiai search qwen` → cherche dans le registre
- [ ] `multiai install-profile lrochetta/qwen-custom` → installe un profil depuis GitHub
- [ ] `multiai publish-profile` → crée un fichier de profil, propose de le PR dans le registre

### 5.6 Shell completion
- [ ] `multiai completion bash` → autocomplétion bash
- [ ] `multiai completion zsh` → autocomplétion zsh
- [ ] `multiai completion fish` → autocomplétion fish
- [ ] `multiai completion powershell` → autocomplétion PowerShell
- [ ] Intégration automatique dans `.bashrc`/`.zshrc` via `multiai install`

### Livrables Phase 5
```
✅ Architecture 9→10 : plugin hooks, registre communautaire, extensibilité
✅ DX 9→10 : documentation site, multiai doctor, shell completion, help riche
```

---

# Phase 6 — Maintenance continue (permanent)

**Objectif** : Maintenir 10/10 dans la durée

### 6.1 Release process
- [ ] Versionnage sémantique strict (MAJOR.MINOR.PATCH)
- [ ] CHANGELOG mis à jour automatiquement via Conventional Commits
- [ ] Release notes automatiques via `goreleaser`
- [ ] Canal `beta` : `go install github.com/lrochetta/multiai@beta`
- [ ] Canal `nightly` : builds automatiques sur main

### 6.2 Monitoring
- [ ] Télémétrie anonyme opt-in (version, OS, profils utilisés, erreurs) → jamais de clés
- [ ] Crash reporting via Sentry (opt-in)
- [ ] Dashboard de métriques (taux d'erreur, profils populaires, OS distribution)

### 6.3 Gouvernance
- [ ] `CODEOWNERS` : @lrochetta
- [ ] `CODE_OF_CONDUCT.md`
- [ ] `GOVERNANCE.md` (comment les décisions sont prises)
- [ ] Process de review : ≥1 approve avant merge sur main
- [ ] Branch protection : require CI, require review, no direct push

### 6.4 Roadmap publique
- [ ] `ROADMAP.md` dans le repo
- [ ] GitHub Project board public
- [ ] Tags `help-wanted`, `good-first-issue` pour attirer les contributeurs

---

# Récapitulatif des phases

| Phase | Durée | Sécurité | Qualité | Architecture | DX | Moyenne |
|---|---|---|---|---|---|---|
| **Aujourd'hui** | — | 4 | 6.5 | 5.5 | 5 | 5.25 |
| **1 — Fondations** | 2 sem | 7 | 8 | 6 | 7 | 7.0 |
| **2 — Robustesse** | 3 sem | 8 | 9 | 7 | 8 | 8.0 |
| **3 — Réécriture Go** | 6 sem | 9 | 9 | 9 | 9 | 9.0 |
| **4 — Industrialisation** | 3 sem | 10 | 10 | 9 | 9 | 9.5 |
| **5 — Excellence DX** | 2 sem | 10 | 10 | 10 | 10 | **10.0** |
| **6 — Maintenance** | ∞ | 10 | 10 | 10 | 10 | **10.0** |

**Durée totale estimée** : 16 semaines (4 mois) de travail effectif, étalable sur 6 mois à temps partiel.

---

# Budget estimé (si externalisé)

| Phase | Jours estimés | TJM indicatif | Coût |
|---|---|---|---|
| 1 — Fondations | 10 j | 600€ | 6 000€ |
| 2 — Robustesse | 15 j | 600€ | 9 000€ |
| 3 — Réécriture Go | 30 j | 700€ | 21 000€ |
| 4 — Industrialisation | 15 j | 700€ | 10 500€ |
| 5 — Excellence DX | 10 j | 600€ | 6 000€ |
| **Total** | **80 jours** | — | **52 500€** |

---

# Risques et mitigations

| Risque | Probabilité | Impact | Mitigation |
|---|---|---|---|
| La réécriture Go introduit des bugs de régression | Moyenne | Haut | Tests exhaustifs + migration par shadowing (les deux versions tournent en parallèle) |
| Les utilisateurs existants rejettent le nouveau nom/format | Faible | Moyen | Migration automatique transparente, période de compatibilité `.env` + `.yaml` |
| `brew`/`scoop` refusent le package | Faible | Faible | Fallback `go install`, npm comme canal secondaire |
| Manque de temps pour la Phase 3 (Go) | Haute | Critique | Phase 3 peut être découpée en 3B1 (core minimal), 3B2 (features), 3B3 (polish) |

---

# Indicateurs de succès par phase

### Phase 1 ✅
- [ ] 0 fichier `.env` tracké par git
- [ ] 100% des références `aicode` remplacées
- [ ] 4 fonctions pures testées (≥80% coverage)
- [ ] CI passe sur Windows + Linux + macOS

### Phase 2 ✅
- [ ] Code PowerShell modularisé (5 modules)
- [ ] `multiai -Json` retourne du JSON valide
- [ ] ≥90% coverage sur les fonctions PowerShell
- [ ] 0 warning PSScriptAnalyzer

### Phase 3 ✅
- [ ] `go install github.com/lrochetta/multiai@latest` fonctionne
- [ ] Tous les tests PowerShell migrés en Go + nouveaux tests
- [ ] `multiai migrate` convertit une installation PowerShell proprement
- [ ] Cross-compilation Windows/macOS/Linux amd64/arm64

### Phase 4 ✅
- [ ] Secrets stockés dans le credential store natif
- [ ] Binaires signés avec Cosign
- [ ] ≥90% test coverage Go
- [ ] 0 issue gosec/govulncheck

### Phase 5 ✅
- [ ] Site de documentation en ligne
- [ ] `multiai doctor` diagnostique et répare
- [ ] Shell completion pour bash/zsh/fish/powershell
- [ ] ≥1 contribution externe acceptée

---

*Plan généré le 2026-06-23 dans le cadre de l'audit complet du projet AI CLI Launcher.*
