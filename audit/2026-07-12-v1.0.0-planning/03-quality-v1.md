Les 6 stories sont ecrites dans le fichier `multiai-go/docs/sprint10-v1.0.0-stories.md` (chemin absolu : `D:\travail\DEV\multiai\.claude\worktrees\wf_9ebf162e-5c6-3\multiai-go\docs\sprint10-v1.0.0-stories.md`).

## Resume des 6 stories

### S10.1 — Test coverage 50% → 90% (CRITICAL)
- 13 sous-stories couvrant chaque package avec etat des lieux, delta, et methodes de test
- BLOCKER : `internal/config` ne compile pas (10 appels a corriger avec `secret.Store` param)
- Priorite : `cmd/multiai` (4.9%), `internal/display` (0%), `internal/i18n` (0%)
- Nouveaux fichiers : `display_test.go`, `i18n_test.go`
- Suppression code mort : eliminer `isSecretLike()` vs `IsSecretKey()` duplication

### S10.2 — 0 vulnerabilite govulncheck (CRITICAL)
- Audit sur 3 OS, fix stdlib et dependances, programme de gestion
- NOUVEAU : `internal/security/vuln.go` pour exemptions documentees
- NOUVEAU : `.github/SECURITY.md` avec politique de divulgation 90 jours

### S10.3 — 0 warning golangci-lint (HIGH)
- Re-activer `errcheck` progressivement (packages par packages)
- Exclusions fines (fmt.Fprint*, Close, Write) au lieu de desactivation totale
- Ajouter `gocyclo`, `misspell`, `whitespace` comme linters supplementaires

### S10.4 — Performance benchmarks (MEDIUM)
- 4 nouveaux fichiers de benchmark : lancement, isolation env, stockage, OpenRouter
- De 2 benchmarks a >=15
- Regression detection dans CI avec `benchstat`

### S10.5 — Tests cross-platform CI (HIGH)
- Audit des build tags, suppression des `|| true` dans CI
- Re-activation de `secret-scan` (gitleaks sans licence)
- Tests OS-specifiques pour Windows Credential Manager, macOS Keychain, libsecret

### S10.6 — Security audit final (CRITICAL)
- Penetration testing sur 6 vecteurs (injection, escalade, divulgation, DoS, race conditions)
- Activer Cosign dans `.goreleaser.yaml`
- Analyser les 6 exclusions gosec, ajouter invariants de securite automatisés
- Score cible >= 9.5/10