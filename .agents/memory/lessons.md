---
title: Lessons
description: Things that burned us
created: "2026-06-23"
project: "multiai"
---

# Lessons

## Audit supply-chain 0.6.10 — 2026-07-15

### Une lecture de coffre doit être structurelle et minimale
- **Impact**: Une lecture console trop large du coffre partagé a affiché des valeurs voisines qui n'étaient pas nécessaires à l'opération GitHub; elles doivent désormais être considérées comme exposées et rotées.
- **Lesson**: Extraire uniquement le champ nommé avec une expression ancrée, ne jamais afficher le document entier et journaliser seulement le pointeur, l'usage et le résultat sans aucune valeur.

### Un clone `--mirror` pousse plus que `--all` si la remote reste en mode miroir
- **Impact**: `git push --force origin --all` a tenté de pousser les refs PR cachées et les tags parce que `remote.origin.mirror=true` venait du clone miroir; GitHub a refusé les refs internes et `master` protégé.
- **Lesson**: Après `git-filter-repo`, supprimer explicitement `remote.origin.mirror`, pousser des refspecs `refs/heads/*` et `refs/tags/*` contrôlés, désactiver le workflow de release pendant les tags historiques et gérer la protection de branche avec restauration garantie.

### Le tree courant doit être comparé blob par blob après une purge
- **Impact**: Le tree nettoyé différait du tree annoncé « propre » sur exactement trois rapports d'audit : preuve que la valeur révoquée survivait encore au HEAD.
- **Lesson**: Comparer les mappings chemin→blob avant/après sans afficher les contenus; une différence limitée aux fichiers attendus valide la purge et révèle les faux diagnostics de propreté.

### L'authentification WebAuth npm doit utiliser la réponse JSON structurée
- **Impact**: La sortie texte `EOTP` reformattée par PowerShell a produit une URL de retour invalide et interrompu deux tentatives, sans publication partielle.
- **Lesson**: Pour une publication non-TTY, demander `npm publish --json`, ouvrir uniquement `error.authUrl`, sonder `error.doneUrl` jusqu'au jeton à usage unique, puis le passer en environnement au retry sans jamais le journaliser.

### Une purge de secret ne s'arrête pas au force-push
- **Impact**: Réécrire master sans traiter les tags, clones, worktrees, releases et caches laisserait des chemins de récupération ou permettrait de réintroduire l'ancien historique.
- **Lesson**: Réécrire toutes les refs distantes dans un miroir jetable, retirer les exceptions du scanner, reconstruire la release sur le nouveau SHA, rescanner un clone frais et demander la purge des vues/caches GitHub si nécessaire.

### Une allowlist de dossier d'audit masque précisément les preuves sensibles
- **Impact**: Gitleaks passait sur tout l'historique parce que `audit/` était globalement ignoré, alors qu'un ancien rapport contenait une vraie clé révoquée.
- **Lesson**: Ne jamais exclure globalement les rapports de sécurité. Après révocation confirmée, limiter toute exception aux règles concernées et au commit exact, sans inclure la valeur du secret.

### Une URL de checksum plausible peut retourner du HTML
- **Impact**: `go.dev/dl/<archive>.sha256` répondait avec une page HTML, rendant le nouveau script d'installation systématiquement inutilisable malgré une intention de vérification correcte.
- **Lesson**: Tester l'URL et le type de contenu réels. Pour une toolchain épinglée, conserver le SHA-256 publié officiellement dans le script et refuser toute divergence avant extraction.

### Une prerelease doit être isolée de tous les canaux secondaires
- **Impact**: Le workflow v0.6.9 gardait GitHub/npm en quarantaine mais aurait encore pu pousser le paquet vers l'AUR.
- **Lesson**: La condition de prerelease doit fermer GitHub latest, npm latest et chaque canal secondaire. Exécuter les scripts de publication depuis le tag audité, jamais depuis une branche mutable.

## Incident Avast/CyberCapture — 2026-07-14

### Un timeout synchrone Node ne borne pas un `CreateProcess` retenu
- **Impact**: Le test isolé de npm 0.6.8 est resté figé au-delà de 60 secondes malgré les options `timeout` de `execFileSync` et `spawnSync`, car l'appel Windows n'avait pas encore rendu le contrôle à Node.
- **Lesson**: Placer la tentative de lancement dans un worker externe et faire porter la deadline par son parent déjà démarré. Tester avec un faux EXE qui dort, vérifier le code 124 et l'absence de processus orphelin.

### Les mécanismes Windows de repli peuvent eux-mêmes échouer
- **Impact**: `Start-Process` a refusé un environnement contenant `Path` et `PATH`; ensuite `taskkill /T /F` a renvoyé « accès refusé » sous Avast et `ErrorActionPreference=Stop` court-circuitait le nettoyage.
- **Lesson**: Utiliser `ProcessStartInfo` sans reconstruire l'environnement, neutraliser l'erreur native de `taskkill`, puis toujours exécuter un fallback CIM + `Process.Kill`. Valider ces branches en boîte noire sur Windows PowerShell 5.

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
