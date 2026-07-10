# Synthèse Audit Complet — multiai v0.4.3

**Date :** 2026-07-09
**Méthodologie :** 5 agents Claude + 1 revue adversariale Codex en parallèle
**Périmètre :** ~70 fichiers Go + CI/CD + packaging + docs + stratégie

---

## Scores consolidés

| # | Audit | Score | Agent |
|---|---|---|---|
| 01 | Sécurité | **7.0/10** | Claude multi-agent |
| 02 | Architecture | **7.8/10** | Claude Explore |
| 03 | Qualité code | **7.0/10** | Claude Explore |
| 04 | Supply chain | **7.3/10** | Claude Explore |
| 05 | DX & Documentation | **6.2/10** | Claude Explore |
| 06 | Stratégie produit | **6.5/10** | Claude Explore |
| 07 | Adversarial | **NEEDS-ATTENTION** | Codex CLI (GPT-5) |

### Score global pondéré : **7.0/10**

---

## Findings CRITICAL (5)

| # | Source | Finding | Fichier |
|---|---|---|---|
| C-1 | Sécurité | Clé API DeepSeek en clair sur le disque | `brainstorm laurent/` |
| C-2 | Sécurité | SSRF + RCE via `MULTIAI_GITHUB_API_URL` | `update/update.go:96` |
| C-3 | Sécurité | Exécution de commandes via hooks | `cli/hooks.go` |
| C-4 | **Codex** | Auto-update exécute des artefacts non authentifiés malgré Cosign | `update/update.go:245` |
| C-5 | **Codex** | Tout tag `v*` publie une release sans gate CI | `release.yml:20` |

## Findings HIGH (17)

| # | Source | Finding |
|---|---|---|
| H-1/H-2 | Sécurité | `--allow-custom-command` + `CLEAR_ENV=false` contournent l'isolation |
| H-3/H-4 | Sécurité | Clé maître AES à côté des ciphertexts + stores natifs = stubs |
| H-5/H-6 | Sécurité | Pas de gitleaks + `os.ExpandEnv` dans hooks |
| H-7 | Architecture | Couplage display : config/menu/onboarding → cli |
| H-8 | Architecture | Duplication atomic writes ×4 |
| H-9 | Architecture | Résolution `getProfilesDir()` dupliquée |
| H-10 | Qualité | `cmd/multiai` : 0% couverture |
| H-11 | Qualité | `internal/menu` : 14% couverture |
| H-12 | **Codex** | L'update n'est pas persistée (reste en temp) |
| H-13 | **Codex** | Profil homonyme = vol de secret |
| H-14 | **Codex** | `--store` natif silencieusement ignoré |
| H-15 | **Codex** | Mutations store/sentinelle = perte credentials (race condition) |
| H-16 | **Codex** | Profils YAML + hooks absents du chemin de production |
| H-17 | **Codex** | Upgrade = pas de déploiement des nouveaux profils |

---

## Matrice des forces/faiblesses

### ✅ Forces majeures (à préserver)
- **1 seule dépendance Go** (yaml.v3) — surface d'attaque minimale
- **Architecture data-driven** (catalogue YAML) — ajout provider sans code
- **Credential store AES-256-GCM** avec sentinel pattern
- **Isolation d'environnement** par liste blanche (~30 vars)
- **Fallback chains** interruption-aware — unique sur le marché
- **37 profils** pré-configurés, 13 fournisseurs, 3 CLI
- **CI/CD** : actions pinnées SHA, Cosign keyless, attestation GitHub
- **Scan-secrets** : bloque la publication npm si clé réelle
- **`go vet` : 0 erreur** — code propre

### ❌ Faiblesses critiques (à corriger avant v0.5.0)
- **Auto-update non authentifiée** — risque RCE (C-2, C-4)
- **Publication sans gate CI** — release signée sans tests (C-5)
- **0% couverture sur `cmd/multiai`** — point d'entrée aveugle
- **Clé DeepSeek sur le disque** — rotation immédiate (C-1)
- **6 packages sous 30% de couverture**
- **0 stars, 0 utilisateurs** — projet invisible

---

## Plan d'action priorisé

### 🔴 BLOCKER — v0.4.4 (cette semaine)

| # | Action | Effort |
|---|---|---|
| 1 | Rotation clé DeepSeek + suppression `brainstorm laurent/` | 5 min |
| 2 | Hardcoder `MULTIAI_GITHUB_API_URL` → `api.github.com` uniquement | 30 min |
| 3 | Vérifier signature Cosign dans l'auto-update (`.sig` + `.pem`) | 2h |
| 4 | Ajouter gate CI dans `release.yml` (lint + test + vet obligatoires) | 1h |
| 5 | Installer gitleaks + pre-commit hook + job CI | 1h |
| 6 | Réparer le job smoke CI (`cd multiai-go` redondant) | 5 min |

### 🟠 HIGH — v0.5.0

| # | Action | Effort |
|---|---|---|
| 7 | Tests pour `cmd/multiai` (getProfilesDir, hasFlag, ensureProfiles) | 2h |
| 8 | Câbler `LoadAllProfiles` (YAML, hooks) dans le chemin de production | 3h |
| 9 | Implémenter `--store` natif ou le refuser explicitement | 4h |
| 10 | Corriger namespace `ServiceForProfile` (racine canonique) | 1h |
| 11 | Transactionalité store/sentinelle (verrou inter-processus) | 4h |
| 12 | Migration versionnée `ensureProfiles` (nouveaux profils après upgrade) | 2h |
| 13 | Extraire `display/` de `cli/` (casser couplage métier→affichage) | 2h |
| 14 | Unifier atomic writes → `fsutil.WriteFileAtomic` | 1h |

### 🟡 MEDIUM — v0.6.0

| # | Action |
|---|---|
| 15 | Propager `context.Context` dans les appels HTTP |
| 16 | Créer documentation `docs/` (VitePress) |
| 17 | Ajouter `multiai update` explicite |
| 18 | Homebrew tap + Scoop bucket (`skip_upload: false`) |
| 19 | SBOM CycloneDX dans les releases |
| 20 | `CONTRIBUTING.md` + templates d'issues GitHub |

### 🟢 LOW — Backlog

| # | Action |
|---|---|
| 21 | i18n anglais (framework minimal) |
| 22 | Fuzz testing (.env, YAML) |
| 23 | Publier HN/Reddit/newsletters (sortir de 0 stars) |
| 24 | Mettre à jour `go.mod` → go 1.24 |
| 25 | Migrer `gopkg.in/yaml.v3` → `github.com/yaml/go-yaml` |

---

## Verdict

**multiai v0.4.3 est un projet techniquement solide (7/10) mais avec des vulnérabilités critiques qui bloquent la release v0.5.0.**

Les 6 BLOCKERS sont corrigeables en ~5h. Une fois résolus, le projet peut viser **8.5/10** pour v0.5.0.

La revue adversariale Codex a été décisive : elle a trouvé 8 findings HIGH que les agents Claude n'avaient pas vus (profil homonyme = vol de secret, race condition store, profils YAML/hooks non câblés, upgrade sans nouveaux profils, CI smoke cassé).

**La priorité absolue reste l'adoption : 0 star = 0 utilisateur = le produit n'existe pas pour le marché.**
