---
title: Lessons
description: Things that burned us
created: "2026-06-23"
project: "multiai"
---

# Lessons

## Incident Avast/CyberCapture — 2026-07-14

### `CommandContext` ne borne pas toujours `CreateProcess` sous Windows
- **Impact**: CyberCapture peut retenir le syscall avant que `exec.Cmd.Start` retourne. L'annulation de contexte n'est alors pas encore armée et un test apparemment borné reste gelé.
- **Lesson**: Exécuter `cmd.Run()` dans un goroutine contrôleur, sélectionner explicitement sur une deadline et ne lire les buffers qu'après le retour du processus. Garder le smoke de l'artefact officiel comme gate de release.

### Un téléchargement vérifié par SHA256 peut rester inutilisable
- **Impact**: npm 0.6.7 vérifiait correctement l'asset puis annonçait une installation réussie alors qu'Avast empêchait son démarrage.
- **Lesson**: Après extraction, exécuter une probe bornée qui vérifie la version exacte avant d'annoncer le succès; un shim doit aussi transformer le gel d'une probe en erreur explicite.

## Audit BMAD+ complet — 2026-07-14

### Un smoke test par chemin interne peut certifier une installation inutilisable
- **Impact**: L'ancien bootstrap lançait directement le JavaScript sous `npm root --global`; il passait même lorsque le shell ne trouvait pas `multiai`.
- **Lesson**: Le test d'acceptation d'un outil CLI doit appeler son nom public avec l'environnement persistant d'une nouvelle session et vérifier le premier shim résolu.

### Les outils Windows n'ont pas tous la même sortie texte
- **Impact**: `where.exe` a remplacé un caractère Unicode d'un chemin temporaire lors du décodage, rendant une comparaison canonique non fiable.
- **Lesson**: Garder la résolution de chemins et la sérialisation dans une couche Unicode maîtrisée ; ici PowerShell produit du JSON UTF-8, puis Node valide strictement le résultat.

### Une allowlist npm doit embarquer tout helper transitif
- **Impact**: Un correctif valide dans le worktree peut disparaître du tarball si `package.json.files` n'inclut pas le module JavaScript et le script PowerShell.
- **Lesson**: Toute évolution du bootstrap doit finir par `npm pack --dry-run --json` et un test d'installation du tarball exact.

### Un timeout n'est jamais un test vert
- **Impact**: La suite Go complète et certains sous-processus se bloquent localement sous Windows ; arrêter l'exécution ne prouve ni succès ni absence de vulnérabilité.
- **Lesson**: Rapporter l'état comme inconclusif, borner chaque enfant avec un contexte et réserver la décision de release à une matrice CI complète et reproductible.

## Session pause release v0.6.7 — 2026-07-13

### Une CI partiellement verte ne prouve pas la portabilité
- **Impact**: Les tests npm, Windows, le lint, la sécurité, GoReleaser et les cross-compilations étaient verts, mais les tests réels macOS et Ubuntu ont encore détecté des défauts de test et d'isolation.
- **Lesson**: Pour une distribution multi-OS, attendre la matrice complète. Une cross-compilation ne remplace pas l'exécution avec Keychain, libsecret et la locale du runner.

### Les tests localisés et les stores natifs doivent être déterministes
- **Impact**: macOS retournait des messages anglais alors que des assertions attendaient le français; deux tests partageaient le Keychain et `MULTIAI_SECRETS_DIR` n'isolait pas le backend natif.
- **Lesson**: Vérifier des identifiants invariants plutôt que des traductions, utiliser des services uniques avec nettoyage, et forcer le backend fichier pour les tests unitaires de migration.

### Les doubles de processus doivent reproduire le vrai argv
- **Impact**: Le faux `secret-tool` incluait d'abord le nom de l'exécutable, puis sa correction a révélé qu'il ne simulait pas l'erreur de la sous-commande `search`.
- **Lesson**: Tester explicitement l'argv transmis et chaque code de sortie simulé pour les wrappers de commandes externes.

## Session v0.6.7 — 2026-07-12

### `https.get` Node brut n'hérite pas de la confiance npm/OS
- **Impact**: npm atteignait son registre, mais le `postinstall` échouait vers GitHub avec `unable to verify the first certificate`; npm masquant la sortie des lifecycle scripts, l'utilisateur percevait un gel.
- **Lesson**: Pour un bootstrap natif, tester avec un cache npx vierge et le vrai magasin de certificats de l'OS. Fusionner CA par défaut + CA système via API feature-détectée; ne jamais utiliser `rejectUnauthorized=false`.

### Une commande documentée est un contrat exécutable
- **Impact**: `npx multiai install` était documenté partout mais transmis au binaire Go, qui répondait `Commande inconnue : install`.
- **Lesson**: Ajouter un test contractuel pour chaque quick-start public. Ici le shim npm restaure l'installation globale historique.

### Ne jamais publier npm depuis un worktree sale
- **Impact**: le tag `v0.6.6` pointe vers un commit contenant `package.json@0.6.5`, alors que npm a reçu 0.6.6 depuis une modification locale.
- **Lesson**: Bloquer le publish si le worktree est sale ou si `HEAD` n'a pas le tag exact `v<package.version>`; vérifier aussi cette égalité dans le workflow GitHub Release.

### EOF ignoré dans une boucle interactive = boucle infinie
- **Impact**: un premier lancement sans TTY traitait EOF comme la réponse par défaut « oui », puis le menu de configuration rebouclait indéfiniment sur une entrée vide.
- **Lesson**: Toute boucle interactive doit traiter EOF comme une sortie normale et posséder un test avec un reader vide.

### Une évolution de signature doit mettre à jour tous les tests
- **Impact**: l'ajout du paramètre `secret.Store` avait laissé plusieurs tests et fuzzers non compilables.
- **Lesson**: Lancer au minimum les tests des packages touchés après toute modification de signature; la CI ne doit jamais accepter des tests qui ne compilent plus.

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
