# Rapport d'Audit Architecture — Projet "AI CLI Launcher" (multiai v0.1.5)

**Auditeur** : Agent spécialisé architecture — revue exhaustive du design et des choix techniques  
**Date** : 2026-06-23  
**Périmètre** : Stack technique, design patterns, profils, BMAD+, vision produit

---

## Note architecture globale : **5.5/10**

| Critère | Évaluation |
|---|---|
| Choix technologiques | Moyen — PowerShell défendable sur Windows, inadapté pour macOS/Linux |
| Design patterns | Correct — implicites mais présents, manque de séparation modulaire |
| Extensibilité | Moyen — ajout de fournisseur = 3-4 fichiers à toucher + code |
| Robustesse | Faible — zéro test, dépendance critique à pwsh, pas de fallback |
| Vision produit | Faible — identité confuse (5 noms différents), positionnement flou |

---

## A1 — Choix technologiques

### PowerShell comme langage principal

**Pour Windows** : pertinent. PowerShell 5.1 est natif sur Windows 10/11, couvre ~95% des machines. Le code tire parti de `[Environment]::SetEnvironmentVariable(..., 'Process')` pour l'isolation.

**Pour macOS/Linux** : architecturalement contestable. L'exigence `pwsh` (PowerShell Core) est une taxe d'entrée considérable. 99% des developers macOS/Linux n'ont pas pwsh installé. Le script `install.sh` admet ce problème : si pwsh est absent, les fichiers sont copiés mais le routeur ne fonctionnera pas.

### npm comme canal de distribution

Concept intéressant mais mal exécuté. `bin/multiai.js` ne gère QUE `install`. Si un utilisateur tape `npx multiai -Profile ds`, il obtient `Commande inconnue : -Profile`. Le package npm est un installeur à usage unique — après installation, Node.js n'est plus jamais utilisé.

### Pourquoi pas Go, Rust, Python, ou bash pur ?

| Alternative | Avantage |
|---|---|
| **Go** | Binaire statique, zéro dépendance, multi-plateforme natif. Le routeur complet tiendrait en <300 lignes. |
| **Rust** | Même avantage, sécurité mémoire en plus. Overkill probablement. |
| **Python** | Plus universel que PowerShell, mais nécessite un interprète. |
| **Bash pur** | Aucune dépendance sur macOS/Linux. L'auteur sait écrire du bash (cf. `install.sh`). |

**La chaîne d'appel actuelle :**
```
utilisateur → npx multiai install → multiai.js → powershell/pwsh → install.ps1 → copie fichiers
utilisateur → ds.cmd → powershell → code-router.ps1 → lit .env → lance CLI
```

C'est une indirection inutile. Un binaire Go unique (`go install github.com/lrochetta/multiai@latest`) remplacerait toute cette chaîne.

---

## A2 — Design patterns

### Patterns identifiés
1. **Strategy pattern** (implicite) : chaque profil .env définit une stratégie de lancement
2. **Template Method** (implicite) : flux standard (charger → filtrer → appliquer → vérifier → lancer)
3. **Registry pattern** (partiel) : `$ProviderCatalog` comme registre de fournisseurs
4. **Command pattern** : chaque `.cmd`/`.sh` est un wrapper Command

### Patterns absents
1. **Factory** : la création de profil est ad-hoc avec des fallbacks en cascade
2. **Observer/Event** : pas de hooks, pas de pipeline extensible
3. **Adapter** : chaque CLI a son propre protocole, mais pas d'adaptateur formel

### Séparation des concerns

`code-router.ps1` mélange 5 responsabilités dans 557 lignes : parsing de profils, affichage menu, configuration de clés, installation BMAD+, lancement CLI. La séparation est fonctionnelle mais pas modulaire (tout dans le même fichier).

### Extensibilité

Ajouter un fournisseur à un CLI existant nécessite :
1. Créer un fichier `.env`
2. Ajouter l'entrée dans `$ProviderCatalog` (code PowerShell)
3. Optionnellement créer un `.cmd` de raccourci

**Problème** : `$ProviderCatalog` (code) et les fichiers `.env` (données) sont deux sources de vérité qui doivent rester synchronisées manuellement.

---

## A3 — Architecture des profils

### Numérotation (00-, 10-, 20-, 30-, 40-, 50-)

Système fragile qui code 3 dimensions dans un seul espace : CLI (00/10/20/30=Claude, 40=Codex, 50+=OpenCode), provider (10=Anthropic, 20=Z.ai, 30=DeepSeek), et ordre d'affichage.

**Le tri par `Sort-Object Tool, Order, DisplayName` n'utilise PAS le préfixe numérique** — la numérotation ne sert qu'à l'organisation visuelle dans le filesystem. C'est du bruit.

### 17 fichiers .env vs config centralisée

| .env individuels | YAML centralisé |
|---|---|
| ✅ Indépendants, faciles à copier/partager | ❌ Conflits de merge possibles |
| ❌ 17 lectures disque à chaque lancement | ✅ Une seule lecture |
| ❌ Duplication (même clé DeepSeek dans 3 fichiers) | ✅ Héritage (`extends`) |
| ❌ Pas de validation structurelle automatique | ✅ Schéma JSON/CUE |

---

## A4 — Robustesse architecturale

### CLI non installé

`Assert-CommandExists` vérifie la présence du binaire via `Get-Command`. Si absent → exception fatale. Pas de fallback (ex: proposer de l'installer). Le check est fait APRÈS que l'utilisateur a choisi son profil.

### Dépendance à PowerShell

- **Windows 10/11** : PowerShell 5.1 pré-installé → ~95% couverts
- **macOS** : 0% ont pwsh par défaut → `brew install powershell/tap/powershell`
- **Linux** : 0% ont pwsh par défaut → `sudo apt-get install -y powershell` (Ubuntu/Debian seulement)

Le marché cible (développeurs IA) utilise massivement macOS (60-70%). L'exigence pwsh réduit significativement le marché potentiel.

### Fallback install.sh

`install.sh` délègue à `pwsh install.ps1` si pwsh est installé. Si pwsh est absent, il copie les fichiers mais émet un avertissement. Ce n'est PAS un fallback fonctionnel — le routeur ne fonctionnera pas sans pwsh.

---

## A5 — BMAD+ integration

### Pertinence

L'intégration BMAD+ se résume à un point de menu (option 3) qui appelle `npx bmad-plus install`. C'est un raccourci pratique mais sans rapport avec le routage CLI.

### Couplage

Quasi-nul techniquement (simple appel `npx`). Mais confusion identitaire : le projet utilise BMAD+ pour son propre développement, ET propose d'installer BMAD+ dans les projets des utilisateurs. Mélange des genres entre outil de build et fonctionnalité livrée.

---

## A6 — Vision produit

### Positionnement

Cible : **développeurs IA expérimentés** jonglant entre plusieurs CLI et fournisseurs. Le problème résolu est réel. Mais :
- Documentation bilingue français/anglais incohérente
- Messages en français → marché international limité
- Option BMAD+ → public très niche

### Cohérence du nommage — **PROBLÈME MAJEUR**

| Nom | Où |
|---|---|
| `multiai` | npm, package.json, README, binaire |
| `aicode` | Ancien nom, encore dans COMMANDS.md, install.sh |
| `code-cli-router` | Dossier d'installation (`C:\AI\code-cli-router`) |
| `multiai` | Projet dans `_bmad/config.yaml` |
| `powerai` | Dépôt GitHub (`github.com/lrochetta/powerai`) |

**5 noms différents pour le même produit.** L'install.ps1 nettoie les anciens `aicode.cmd` (preuve de transition), mais COMMANDS.md référence encore `aicode` partout.

### Ce qui manque

1. Tests (zéro)
2. CI/CD
3. Mise à jour automatique (`multiai self-update`)
4. Documentation pas-à-pas pour nouveaux utilisateurs
5. i18n (choix de langue)
6. Logging / mode debug
7. Sortie JSON pour intégration scriptée
8. Profils par projet (`.multiai.yaml` local)

---

## Top 5 problèmes architecturaux

| # | Problème | Impact |
|---|---|---|
| **1** | **Surcharge Node.js/npm inutile** — ne sert qu'à l'installation, jamais utilisé pour le routage | Complexité, points de panne |
| **2** | **Double source de vérité** — `$ProviderCatalog` + fichiers .env doivent rester sync | Fragilité, bugs silencieux |
| **3** | **Dépendance pwsh pour macOS/Linux** — inutilisable sans installation préalable | Barrière à l'adoption |
| **4** | **17 fichiers .cmd redondants** — chaque profil = un fichier wrapper | Non-maintenable à l'échelle |
| **5** | **5 noms différents** pour le même produit — `multiai`/`aicode`/`code-cli-router`/`multiai`/`powerai` | Confusion utilisateur, SEO |

---

## Feuille de route recommandée

### Phase 1 — Quick wins
- Uniformiser le nommage : `multiai` partout
- Remplacer les 17 `.cmd` par un seul `multiai.cmd` + dispatcher
- Dériver `$ProviderCatalog` automatiquement des fichiers .env
- Ajouter `--json` pour intégration scriptée

### Phase 2 — Refonte architecture
- Réécrire le routeur en **Go** : `go install github.com/lrochetta/multiai@latest`
- Un binaire unique : installation + routing + configuration + mise à jour
- Supprimer la dépendance npm/Node.js

### Phase 3 — Industrialisation
- Schéma de validation pour les profils (JSON Schema ou CUE)
- Tests unitaires et d'intégration
- CI/CD avec compilation multi-plateforme
- Gestion sécurisée des secrets (keychain macOS, Windows Credential Manager)

### Phase 4 — Produit
- Plugin hooks (avant/après lancement)
- Catalogue communautaire de profils
- Profils par projet (`.multiai.yaml`)
