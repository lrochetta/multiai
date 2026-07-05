---
title: Lessons
description: Things that burned us
created: "2026-06-23"
project: "multiai"
---

# Lessons

## Session v0.4.x (2026-07-05/06)

### Repo privé = npm cassé
- **Impact**: `install.js` télécharge les binaires depuis GitHub Releases. Si le repo est privé, le téléchargement échoue (HTTP 404 pour les non-authentifiés).
- **Lesson**: Un package npm qui download des assets GitHub nécessite un repo **public**. Vérifier avant de publier.

### GitHub Actions : workflows à la racine, pas dans un sous-dossier
- **Impact**: Les workflows dans `multiai-go/.github/workflows/` n'étaient pas exécutés. GitHub ne les voit que dans `.github/workflows/` à la racine.
- **Lesson**: Dans un monorepo, maintenir les workflows à la racine. Le script `sync-workflows.ps1` est un palliatif, mais la source de vérité doit être racine.

### Force-push nécessite de désactiver la branch protection
- **Impact**: `git push --force` rejeté après avoir configuré la protection de branche.
- **Lesson**: Désactiver la protection (`gh api ... --method DELETE`), force-push, puis réactiver. Automatiser en une commande.

### npm publish nécessite 2FA même en CLI
- **Impact**: `npm publish` bloqué par `EOTP` (one-time password). Le navigateur Chrome gère l'OTP automatiquement.
- **Lesson**: Publier depuis un terminal interactif (pas via un agent). Le `prepublishOnly` scan-secrets.js est exécuté avant la publication.

### Audit de sécurité = toujours vérifier ce qui est exposé avant de passer public
- **Impact**: Une clé API DeepSeek locale (gitignorée) et le rapport d'audit contenait la clé en clair dans un commit.
- **Lesson**: Avant de passer un repo en public : (1) scanner tous les fichiers pour des secrets, (2) vérifier l'historique git, (3) vérifier les rapports d'audit eux-mêmes.

### git filter-branch pour nettoyer l'historique
- **Impact**: Besoin de retirer "Co-Authored-By: Claude" de 21 commits.
- **Lesson**: `git filter-branch --msg-filter "sed '/^Co-Authored-By: Claude/d'" -- --all` fonctionne. Penser à `--tag-name-filter cat` pour les tags.

### README non mis à jour = information trompeuse (rappel)
- **Impact**: README disait "version PowerShell v0.3.0" pour npm alors que le Go est publié. "20+ profils" au lieu de 37.
- **Lesson**: Mettre à jour le README dans le même commit que les changements. Vérifier les nombres, les versions, les liens.

## Session v0.3.0 (2026-06-24)

### Fichiers brainstorming = indésirables sur GitHub
- **Impact**: `brainstorm-openrouter.md` tracké dans git malgré `.gitignore`. Commit avant la règle.
- **Lesson**: Toujours vérifier `git ls-files` après avoir ajouté une règle `.gitignore`.

### CHANGELOG en doublon = divergence garantie
- **Impact**: Deux CHANGELOG.md identiques. Maintenir 2 copies = l'une sera obsolète.
- **Lesson**: Un seul CHANGELOG à la racine du repo.

### La mémoire projet est un passif si non maintenue
- **Impact**: `.agents/memory/context.md` gelé à v0.2.6, aucune ADR pour v0.3.0.
- **Lesson**: Après chaque release majeure, mettre à jour TOUS les fichiers mémoire.

## Session v0.2.6 (2026-06-23)

### 5 agents parallèles sans conflit = possible avec découpage par fichier
- **Impact**: 42 fixes en ~50 min sans aucun conflit.
- **Lesson**: Découper les agents par FICHIER, pas par feature. Aucun chevauchement = zéro conflit.

### Toujours exporter les symboles utilisés par les tests
- **Impact**: `AllowedCommands` renommé en privé → `tests/` ne compile plus.
- **Lesson**: Un symbole utilisé dans un package `_test` externe DOIT être exporté.

### `prepublishOnly` doit être précis
- **Impact**: Faux positifs sur DISPLAY_NAME, ANTHROPIC_BASE_URL.
- **Lesson**: Whitelist des clés de métadonnées + URLs dans le scan.

### Classifieur deepseek intermittent = fallback PowerShell
- **Impact**: 30+ commandes bloquées.
- **Lesson**: Toujours avoir un fallback PowerShell pour les commandes git/npm.

## Leçons résolues
- ✅ Renommage atomique (aicode → multiai)
- ✅ Parser .env : supporter `export`
- ✅ Isolation : liste blanche
- ✅ Injection shell → escapeShellArg
- ✅ Race condition → sync.Mutex
- ✅ YAML bomb → limite 1 Mo
- ✅ Navigation UX → boucle + "0. Retour"
- ✅ Accessibilité → préfixes texte + NO_COLOR
- ✅ npm switch PS→Go
- ✅ Branch protection
