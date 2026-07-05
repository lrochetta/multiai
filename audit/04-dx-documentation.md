# Rapport Audit DX & Documentation — AI CLI Launcher / multiai

**Projet** : `D:\travail\DEV\multiai`  
**Fichiers audités** : README.md, COMMANDS.md, PROVIDERS.md, INSTALL-CLI.md, SECURITY.md, README-CONFIGS.txt, CLAUDE.md, AGENTS.md, GEMINI.md, package.json, LICENSE  
**Date** : 2026-06-23

---

## Note DX globale : **5/10**

| Critère | Note |
|---|---|
| Onboarding (première expérience) | 5/10 |
| Documentation | 4/10 |
| Nommage et branding | 2/10 |
| Interface utilisateur | 7/10 |
| Contribution (dev) | 2/10 |

---

## DX1 — Onboarding

### Parcours `npx multiai install` → première commande

**Étapes** : 7 étapes, 2-3 minutes si tous les prérequis sont présents.

**Problèmes** :
- `install.sh` référence `aicode.sh` dans le message post-install (n'existe plus)
- Aucune vérification que les CLIs cibles (`claude`, `codex`, `opencode`) sont installées — découverte au lancement seulement
- Sur macOS/Linux sans pwsh : `install.sh` copie les fichiers mais le routeur ne fonctionnera PAS. L'utilisateur a une installation "fantôme"

### Messages d'erreur prérequis manquant
- **Node manquant** : `npx` échoue avec erreur Node standard, aucun message personnalisé
- **PowerShell manquant (macOS/Linux)** : message clair dans `install.sh` ✅
- **CLI cible manquant** : message en français "Commande introuvable" — correct mais incohérent avec README anglais
- **Clé API manquante** : référence l'ancien nom `aicode -Configure`

---

## DX2 — Documentation

### README.md — 6.5/10
- ✅ Structure claire (problème → solution → install → tableau → usage)
- ✅ Badges (npm, license, platform, Node)
- ✅ Tableau des profils complet et bien formaté
- ❌ Aucune section troubleshooting
- ❌ Aucune section FAQ ou Contributing
- ❌ Tableau montre 7 profils mais dit "17 profiles total"

### Docs/ — 3/10
- **COMMANDS.md** : complet mais **entièrement en français** et **entièrement avec l'ancien nom `aicode`** (21 occurrences). Totalement décalé du README.
- **PROVIDERS.md** : court, correct, en français
- **INSTALL-CLI.md** : minimaliste (3 commandes). Aucune info sur comment installer les CLIs
- **SECURITY.md** : décrit le stockage mais manque politique de sécurité, procédure de reporting

### Sujets manquants
- ❌ Troubleshooting
- ❌ FAQ
- ❌ Guide de contribution
- ❌ CHANGELOG
- ❌ Guide de migration `aicode` → `multiai`

### Mélange anglais/français
Le README est en anglais, les 4 docs/ en français, les messages du routeur en français. **Dissonance majeure** : un utilisateur anglophone lit le README, tape une commande, reçoit une erreur en français ("Fichier introuvable").

---

## DX3 — Nommage et branding — **PROBLÈME MAJEUR**

### 6 noms différents pour le même produit

| Nom | Où | Statut |
|---|---|---|
| `multiai` | npm, README, nouveaux .cmd/.sh | **Officiel actuel** |
| `aicode` | COMMANDS.md (21 fois), code-router.ps1 | **Ancien, encore partout** |
| `code-cli-router` | Dossier d'installation, scripts | Nom technique |
| `multiai` | Dossier projet, `_bmad/config.yaml` | Nom projet |
| `AI Code CLI Router` | Messages menu, en-têtes | Nom affichage |
| `powerai` | `package.json` repository URL | Nom dépôt GitHub |

### Impact
- L'utilisateur lit COMMANDS.md → tape `aicode` → erreur
- L'installateur Linux dit `aicode.sh` → n'existe plus
- Google: "multiai" vs "aicode" vs "ai cli launcher" → 3 identités différentes

---

## DX4 — Interface utilisateur — 7/10

### Points forts
- Menu interactif clair avec couleurs
- Statut de configuration `[OK]`/`[~~]`/`[--]`
- URLs directes vers les pages de création de clés
- Masquage des clés dans le debug (`PAST...HERE`)

### Points faibles
- Messages d'erreur en français dans un README anglais
- Références à des commandes inexistantes (`cc -List`, `aicode -Configure`)
- Si aucun profil n'est configuré, le menu s'affiche mais l'erreur référence l'ancien nom

---

## DX5 — Développeur (contribuer) — 2/10

### Infrastructure inexistante
- ❌ Aucun guide de contribution (CONTRIBUTING.md)
- ❌ Aucun CHANGELOG
- ❌ Zéro test — vérification manuelle uniquement
- ❌ Aucune CI/CD — pas de GitHub Actions, pas de badge build
- ❌ Aucun linting — pas de `.editorconfig`, `PSScriptAnalyzer`
- ❌ `docs/` exclu du git — les docs ne sont pas versionnées !
- ❌ `.gitignore` n'exclut PAS les vrais fichiers `.env` — risque de commit de clés

---

## DX6 — BMAD+

### Intégration
- Option 3 du menu (`multiai -Bmad`) → `npx bmad-plus install`
- Intégration discrète, ne gêne pas l'utilisateur final
- Message "Astuce : lance cc depuis le dossier du projet" — `cc` non documenté

### Fichiers CLAUDE.md / AGENTS.md / GEMINI.md
**Ils sont identiques** (copies exactes, 2 873 octets chacun). Problèmes :
- `GEMINI.md` est trompeur — Gemini a été retiré du projet (PROVIDERS.md)
- 3 fichiers identiques créent de la confusion : lequel est l'autorité ?
- Suggère un artifact de génération automatique non nettoyé

---

## Top 5 problèmes DX

| # | Problème | Impact |
|---|---|---|
| **1** | **Crise identitaire** — 6 noms, doc en `aicode`, README en `multiai` | Confusion totale |
| **2** | **Zéro troubleshooting** — pas de FAQ, pas de guide de dépannage | Utilisateur bloqué sans aide |
| **3** | **Documentation incohérente** — README anglais, docs français, messages français | Dissonance |
| **4** | **Zéro infrastructure contributeur** — pas de tests, CI, linting, CONTRIBUTING.md | Barrière à la contribution |
| **5** | **macOS/Linux mal supporté** — install.sh référence `aicode.sh`, pas de test Linux | Expérience dégradée |

---

## Fichiers à corriger en priorité

| Fichier | Problème | Priorité |
|---|---|---|
| `docs/COMMANDS.md` | 21× `aicode` au lieu de `multiai` | **CRITIQUE** |
| `code-router.ps1:225` | `cc -List` au lieu de `multiai -List` | **CRITIQUE** |
| `code-router.ps1:288` | `aicode -Configure` au lieu de `multiai -Configure` | **CRITIQUE** |
| `install.sh:122-124` | `aicode.sh` au lieu de `multiai.sh` | **CRITIQUE** |
| `.gitignore` | `docs/` exclu, `.env` non exclu | HAUTE |
| `README.md` | Tableau profils 7/17 incohérent | MOYENNE |
| `GEMINI.md` | Fichier identique à CLAUDE.md, nom trompeur | MOYENNE |
| `docs/INSTALL-CLI.md` | Trop court, pas de vrai guide | MOYENNE |

---

## Recommandations

1. **Unifier le nom** — `multiai` partout. Renommer toutes les références `aicode`. Supprimer `GEMINI.md`.
2. **Créer une section troubleshooting** — erreurs courantes, solutions
3. **Choisir une langue** — tout en anglais (standard open source) ou tout en français
4. **Ajouter l'infrastructure minimale** — CONTRIBUTING.md, CHANGELOG.md, `.editorconfig`, CI GitHub Actions
5. **Tester macOS/Linux** — `multiai.sh` fonctionnel, messages post-install corrects
