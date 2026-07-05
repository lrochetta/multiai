# Synthèse d'Audit — AI CLI Launcher (multiai v0.1.5)

**Date** : 2026-06-23  
**Périmètre** : Code source complet, sécurité, architecture, qualité, documentation, DX  
**Méthodologie** : 4 agents spécialisés en parallèle — revue exhaustive de tous les fichiers

---

## Notes globales

| Dimension | Note | Auditeur |
|---|---|---|
| 🔴 Sécurité | **4/10** | Agent sécurité |
| 🟡 Qualité Code | **6.5/10** | Agent qualité |
| 🔵 Architecture | **5.5/10** | Agent architecture |
| 🟢 DX & Documentation | **5/10** | Agent DX |
| 📊 **Moyenne pondérée** | **5.25/10** | — |

---

## Résumé exécutif

**AI CLI Launcher** (publié sous le nom `multiai`) est un routeur CLI multi-IA fonctionnel qui résout un vrai problème : jongler entre Claude Code, Codex CLI et OpenCode avec des clés API isolées par fournisseur. Le concept est bon, la réalisation est **fonctionnelle mais fragile**.

Le projet souffre de **3 problèmes systémiques** qui se répercutent dans toutes les dimensions de l'audit :

1. **Crise d'identité produit** — 6 noms différents, doc et code qui référencent l'ancien nom `aicode`
2. **Choix technologique en demi-teinte** — PowerShell excellent sur Windows, barrière à l'adoption sur macOS/Linux
3. **Absence totale d'infrastructure de qualité** — zéro test, zéro CI, pas de linting

---

## Top 10 problèmes toutes catégories

| # | Catégorie | Sévérité | Problème |
|---|---|---|---|
| 1 | 🔴 Sécurité | **CRITIQUE** | `.gitignore` n'exclut pas `*.env` — risque de commit de clés API |
| 2 | 🔴 Sécurité | **CRITIQUE** | Exécution arbitraire via champ `COMMAND` des profils .env |
| 3 | 🔴 Sécurité | **HAUTE** | Clés API exposées dans l'environnement du processus enfant |
| 4 | 🟡 Code | **HAUTE** | `Read-DotEnvFile` ignore le préfixe `export` (format Unix standard) |
| 5 | 🔵 Architecture | **HAUTE** | Double source de vérité : `$ProviderCatalog` + fichiers .env |
| 6 | 🟢 DX | **CRITIQUE** | 6 noms pour le même produit ; doc entière en `aicode` (nom abandonné) |
| 7 | 🔵 Architecture | **HAUTE** | Dépendance à pwsh bloque l'adoption sur macOS/Linux |
| 8 | 🟡 Code | **MOYENNE** | Zéro test — aucune couverture, modifications risquées |
| 9 | 🔴 Sécurité | **HAUTE** | Périmètre `$KnownEnvVars` incomplet — fuite de secrets système |
| 10 | 🟢 DX | **HAUTE** | Zéro troubleshooting — utilisateur bloqué sans aide |

---

## Forces du projet

- ✅ **Concept pertinent** — répond à un vrai besoin des développeurs IA multi-CLI/multi-fournisseur
- ✅ **Isolation fonctionnelle** — `CLAUDE_CONFIG_DIR` séparé par profil, `CLEAR_ENV` efficace dans son périmètre
- ✅ **Code PowerShell propre** — idiomes respectés, `-LiteralPath` systématique, `shell: false` dans Node.js
- ✅ **Profils .env bien pensé** — simple, lisible, versionnable, facile à éditer
- ✅ **Expérience interactive agréable** — menu coloré, statuts `[OK]/[~~]/[--]`, URLs directes vers les clés
- ✅ **npm propre** — zéro dépendance, `spawnSync` sécurisé, gestion cross-platform
- ✅ **Installateur robuste** — préservation des .env existants, backup `.new`, nettoyage anciens noms

---

## Recommandations prioritaires

### 🔴 Immédiat (cette semaine)

1. **Ajouter `configs/profiles/*.env` dans `.gitignore`**
   - Fichier : `.gitignore`
   - Risque : commit accidentel de clés API dans l'historique git

2. **Uniformiser le nom `multiai` partout**
   - Fichiers : `docs/COMMANDS.md` (21 occurrences), `code-router.ps1:225,288`, `install.sh:122-124`
   - Remplacer `aicode`, `cc`, `aicode.sh` par `multiai`, `multiai.sh`

3. **Valider `COMMAND` avant exécution**
   - Fichier : `code-router.ps1:542`
   - Whitelist : `claude`, `codex`, `opencode` uniquement

### 🟠 Haute priorité (ce mois)

4. **Corriger `Read-DotEnvFile` pour le préfixe `export`**
   - Fichier : `code-router.ps1:132`, +2 lignes

5. **Remplacer `Clear-RouterEnvironment` par un nettoyage complet**
   - Au lieu de 30 variables connues, nettoyer tout sauf PATH, HOME, USER

6. **Appliquer `chmod 600` sur les fichiers .env après installation**
   - Fichier : `install.ps1`, fonction post-install

7. **Créer une section troubleshooting dans le README**

### 🟡 Moyenne priorité (ce trimestre)

8. **Ajouter tests Pester pour les 4 fonctions pures**
   - `Test-IsPlaceholder`, `Read-DotEnvFile`, `Split-ArgsSimple`, `Expand-RouterValue`

9. **Dériver `$ProviderCatalog` automatiquement des fichiers .env**
   - Supprimer la double source de vérité

10. **Remplacer `Split-ArgsSimple` par un parseur respectant les guillemets**

11. **Corriger les messages post-install** (`install.ps1:162-163`, `install.sh:122-124`)

12. **Choisir une langue unique pour la documentation** (tout anglais ou tout français)

### 🔵 Long terme (6 mois)

13. **Réécrire le routeur en Go** — binaire unique, zéro runtime, multi-plateforme natif
14. **Gestion sécurisée des secrets** — intégration keychain macOS, Windows Credential Manager
15. **CI/CD** — GitHub Actions, compilation multi-plateforme, tests automatisés
16. **Catalogue communautaire de profils** — `multiai search deepseek`

---

## Verdict

**AI CLI Launcher est un prototype fonctionnel prometteur, pas encore un produit industrialisable.**

Le cœur du problème (isolation des environnements multi-CLI) est bien identifié et la solution est techniquement correcte sur Windows. Mais les choix techniques (PowerShell + npm wrapper), l'absence de tests, la confusion de nommage, et les lacunes de sécurité (stockage des secrets, isolation incomplète) empêchent le projet d'atteindre un niveau de qualité production.

Les corrections prioritaires (`.gitignore`, uniformisation du nom, validation COMMAND) peuvent être faites en **une journée** et élèveraient significativement la qualité perçue et la sécurité.

À plus long terme, une réécriture en Go résoudrait la plupart des problèmes architecturaux d'un coup : plus de dépendance runtime, compilation multi-plateforme native, gestion d'environnement intégrée, et un vrai `go install` en une commande.
