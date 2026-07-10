# Audit Qualité de Code — multiai Go v0.4.3

**Date :** 2026-07-09
**Score global : 7/10**
**`go vet` : 0 erreur**

---

## Couverture par package

| Package | Couverture | Score | Statut |
|---|---|---|---|
| `internal/catalog` | 94.9% | 10/10 | ✅ Excellent |
| `internal/env` | 97.4% | 10/10 | ✅ Excellent |
| `pkg/dotenv` | 94.7% | 10/10 | ✅ Excellent |
| `internal/openrouter` | 87.1% | 9/10 | ✅ Très bon |
| `internal/config` | 83.0% | 8/10 | ✅ Très bon |
| `internal/assets` | 73.7% | 7/10 | ✅ Bon |
| `internal/secret` | 70.4% | 7/10 | ✅ Bon |
| `internal/cli` | 53.9% | 6/10 | ⚠️ Moyen |
| `internal/profile` | 48.4% | 6/10 | ⚠️ Moyen |
| `internal/logging` | 47.7% | 5/10 | ⚠️ Moyen |
| `internal/onboarding` | 44.6% | 5/10 | ⚠️ Moyen |
| `internal/update` | 44.6% | 6/10 | ⚠️ Moyen |
| `internal/menu` | 14.0% | 4/10 | ❌ Faible |
| **`cmd/multiai`** | **0.0%** | **2/10** | 🔴 Critique |
| `internal/fsutil` | 0.0% | 3/10 | ❌ Faible |

---

## Points forts
- Tests table-driven systématiques
- Mocks HTTP (`httptest.NewServer`)
- Test sentinel invariant (jamais de clé réelle dans .env)
- Test de sécurité : `TestEmbeddedProfilesContainNoRealSecrets`
- Gestion d'erreurs avec `%w` systématique
- Sentinels errors (`ErrProfileExists`)

## Points faibles
- `cmd/multiai` : 0% — point d'entrée, parsing args, résolution chemins
- `internal/menu` : 14% — menus interactifs non testés
- Duplication : `RunBeforeHooks`/`RunAfterHooks` (90% identique)
- Pas de `require`/`assert` (testify)
- Pas de fuzz testing

---

## Top 10 améliorations

| # | Action | Package |
|---|---|---|
| 1 | Tests pour `getProfilesDir()`, `hasFlag()`, `ensureProfiles()` | `cmd/multiai` |
| 2 | Tests directs pour `WriteFileAtomic()` (ENOSPC, crash) | `internal/fsutil` |
| 3 | Tests pour `ShowTopMenu()`, `SelectTool()`, `SelectProfile()` | `internal/menu` |
| 4 | Refactorer `RunBeforeHooks`/`RunAfterHooks` en fonction partagée | `internal/cli` |
| 5 | Tests pour `Logger` (DEBUG/INFO/WARN/ERROR) | `internal/logging` |
| 6 | Tests pour `LoadDirYAML()`, `MergeProjectConfig()` | `internal/profile` |
| 7 | Tests pour `extractZip()`, `extractTarGz()`, `execNewBinary()` | `internal/update` |
| 8 | Supprimer duplication `isSecretLike()`/`IsSecretKey()` | multiples |
| 9 | Uniformiser `sortProfilesSlice()` dans `LoadDir()` | `internal/profile` |
| 10 | Remplacer `contains()` custom par `strings.Contains()` | `tests/` |
