# Synthèse d'Audit v0.2.1 — multiai

**Date** : 2026-06-23
**Auditeurs** : Sentinel (Code), Rachel (UX), Shield (Sécurité)
**Méthodologie** : 3 agents spécialisés en parallèle

---

## Notes globales

| Dimension | Note | Auditeur |
|-----------|------|----------|
| 🟡 Code Go | **5.5/10** | Sentinel |
| 🟢 UI/UX | **5.5/10** | Rachel |
| 🔴 Sécurité | **5.5/10** | Shield |
| **📊 Moyenne** | **5.5/10** | — |

> **Note** : L'audit précédent (v0.1.5) était à 5.25/10. Le score est stable — les forces du projet sont réelles mais les mêmes types de problèmes persistent dans la nouvelle version Go.

---

## Top 15 problèmes toutes catégories

| # | Catégorie | Sévérité | Problème |
|---|-----------|----------|----------|
| 1 | 🔴 Sécurité | **CRITIQUE** | Injection shell dans hooks via templates (`DISPLAY_NAME=; rm -rf /`) |
| 2 | 🟡 Code | **CRITIQUE** | Race condition TOCTOU sur `encryptedFileStore.Set/Delete` |
| 3 | 🟡 Code | **CRITIQUE** | `AllowedCommands` map mutable sans sync — concurrent access = fatal panic |
| 4 | 🔴 Sécurité | **HAUTE** | Checksums placeholders dans brew/scoop/AUR (`REPLACE_WITH_ACTUAL_SHA256`) |
| 5 | 🔴 Sécurité | **HAUTE** | Master key AES-256 stockée en clair dans `~/.config/multiai/secrets/.masterkey` |
| 6 | 🔴 Sécurité | **HAUTE** | Aucune signature Cosign des binaires |
| 7 | 🟡 Code | **HAUTE** | Code de sortie processus fils non propagé (toujours 0) |
| 8 | 🟡 Code | **HAUTE** | Aucun `context.Context` — processus fils orphelin au Ctrl+C |
| 9 | 🟡 Code | **HAUTE** | `updateEnvFile` non-atomique — perte de données si étape 2 échoue |
| 10 | 🟢 UX | **CRITIQUE** | Navigation sans retour en Go (pas de "0. Retour") |
| 11 | 🟢 UX | **HAUTE** | Couleurs sans texte — inaccessible aux daltoniens/lecteurs d'écran |
| 12 | 🟢 UX | **HAUTE** | Incohérence majeure Go vs PowerShell (paradigme CLI, titre, comportement) |
| 13 | 🟢 UX | **HAUTE** | Aucun wizard de premier démarrage (onboarding inexistant) |
| 14 | 🔴 Sécurité | **MOYENNE** | Key derivation SHA-256 simple (0 itérations) |
| 15 | 🔴 Sécurité | **MOYENNE** | `--allow-custom-command` contourne la whitelist sans validation |

---

## Forces confirmées

| Force | Audit |
|-------|-------|
| Architecture zero-trust (whitelist env) | Sécurité ✅ |
| Chiffrement AES-256-GCM | Sécurité ✅ |
| 1 seule dépendance externe (yaml.v3) | Sécurité + Code ✅ |
| CI/CD complète (lint, test 6×, security, benchmark) | Code ✅ |
| `prepublishOnly` anti-fuite | Sécurité ✅ |
| Configuration des clés API bien conçue | UX ✅ |
| README + CHANGELOG complets | UX ✅ |
| Profilage .env flexible | UX + Code ✅ |
| Mode non-interactif opérationnel | UX ✅ |
| 45+ tests | Code ✅ |

---

## Plan de correction priorisé

### 🔴 Immédiat (cette semaine — ~3 jours)

| # | Action | Effort | Impact |
|---|--------|--------|--------|
| 1 | **Fix injection hooks** — échapper templates avant shell | 3h | CVSS 8.3 → 0 |
| 2 | **Fix race condition** — `sync.Mutex` sur encryptedFileStore | 1h | Fatal panic → 0 |
| 3 | **Fix AllowedCommands** — slice immuable + fonction d'accès | 30min | Fatal panic → 0 |
| 4 | **Navigation "0. Retour" en Go** — comme déjà fait en PS | 1h | UX critique |
| 5 | **Préfixes textuels couleurs** — `[OK]`/`[ERR]`/`[WARN]` | 30min | Accessibilité |
| 6 | **Propager exit code** processus fils | 15min | Comportement correct |

### 🟠 Haute priorité (ce mois — ~2 semaines)

| # | Action | Effort |
|---|--------|--------|
| 7 | **Credential store natif** — remplacer stubs par vraies implémentations | 3 jours |
| 8 | **Pipeline release** — goreleaser + Cosign + checksums | 2 jours |
| 9 | **Key derivation PBKDF2** — remplacer SHA-256 simple | 4h |
| 10 | **Signal handling** — SIGINT/SIGTERM → enfant | 4h |
| 11 | **Convergence Go/PS** — uniformiser paradigme CLI | 2 jours |
| 12 | **Wizard onboarding** — détection first-run | 1 jour |
| 13 | **YAML bomb protection** — limites taille/recursion | 2h |
| 14 | **CI/CD pins** — SHAs + dependabot | 2h |

### 🟡 Moyenne priorité (ce trimestre)

| # | Action | Effort |
|---|--------|--------|
| 15 | **`--allow-custom-command`** → fichier config + validation | 1 jour |
| 16 | **`updateEnvFile` atomique** — write to temp + rename | 1h |
| 17 | **Validation clés API** — regex par fournisseur | 2h |
| 18 | **`NO_COLOR` support** | 30min |
| 19 | **Messages d'erreur avec suggestions** | 1 jour |
| 20 | **JSON enrichi** (PID, timestamp, exit_code) | 2h |
| 21 | **Zeroisation mémoire secrets** | 4h |
| 22 | **Logger structuré** | 1 jour |

---

## Roadmap vers 10/10

| Phase | Semaines | Sécurité | Code | UX | Moyenne |
|-------|----------|----------|------|-----|---------|
| **Aujourd'hui** | — | 5.5 | 5.5 | 5.5 | **5.5** |
| **Quick wins** (6 fixes) | 3 jours | 7.5 | 7.0 | 7.0 | **7.2** |
| **Haute priorité** (8 fixes) | 2 sem | 9.0 | 8.5 | 8.5 | **8.7** |
| **Moyenne priorité** (8 fixes) | 4 sem | 9.5 | 9.0 | 9.0 | **9.2** |
| **OpenRouter** (Phase A-C) | 1 sem | — | — | — | nouvelle feature |
| **Polish final** | 2 sem | 10 | 10 | 10 | **10** |

---

*Rapport généré le 2026-06-23 — 3 agents spécialisés en parallèle*
