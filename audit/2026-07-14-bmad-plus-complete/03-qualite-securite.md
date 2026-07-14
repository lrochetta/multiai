# Audit qualité, sécurité et chaîne d'approvisionnement — multiai

**Date :** 2026-07-14
**Auditeur :** Sentinel (BMAD+ Quality Guardian — QA, UX et sécurité)
**Révision de départ :** `5808769` (`master`) avec un worktree déjà modifié et conservé
**Périmètre :** code Go, CLI, tests, installation npm, correctif PATH Windows, secrets, configuration projet, registre communautaire, auto-update, CI/CD et dépendances
**Score de préparation à une release : 5,4/10**
**Décision : NO-GO pour publier v0.6.7 en l'état.**

> Le cœur local de multiai est sérieux : peu de dépendances d'exécution, liste blanche des CLI, isolation d'environnement, credential stores natifs, écriture atomique, tests nombreux et package npm sans dépendance runtime. Le correctif PATH Windows ajouté pendant l'audit résout le défaut initial dans le parcours explicite `npx ... multiai install` et passe 25 tests. Il reste toutefois quatre bloqueurs P1 indépendants : une configuration de projet non approuvée peut exécuter du code ou détourner des secrets, le registre permet une sortie de répertoire, l'auto-update exécute un binaire temporaire sans authenticité obligatoire ni installation persistante, et le workflow de release réellement exécuté n'impose ni tests ni synchronisation avec sa copie renforcée.

---

## 1. Méthode, hypothèses et sévérités

L'audit combine lecture de code orientée menaces, revue adversariale du correctif Windows, exécution locale de tests, contrôle du tarball npm, analyse de dépendances et comparaison des workflows GitHub Actions. Aucun secret, package ou tag n'a été publié. Aucune écriture persistante du PATH utilisateur n'a été effectuée : le helper PowerShell a été exercé en mode `Plan` par les tests.

Le modèle de menace retient notamment :

- un dépôt cloné ou un répertoire de travail non fiable ;
- un index de registre, un cache ou un compte GitHub compromis ;
- une machine Windows avec un PATH ancien, personnalisé ou hostile ;
- un poste d'entreprise derrière proxy et autorité de certification locale ;
- une erreur de mainteneur lors du tag ou de la synchronisation des workflows.

| Niveau | Définition | Gate |
|---|---|---|
| **P0 — critique** | Exploitation immédiate et triviale avec impact systémique, ou perte/destruction majeure. | Arrêt immédiat. |
| **P1 — bloquant release** | RCE, exfiltration, écriture hors périmètre, chaîne de livraison non fiable ou promesse centrale cassée. | Corriger avant v0.6.7. |
| **P2 — important** | Défense en profondeur absente, test non déterministe, vulnérabilité de développement ou forte friction UX. | Corriger dans le sprint suivant ; certains P2 Windows sont recommandés avant release. |
| **P3 — amélioration** | Durcissement ou dette maintenable sans rupture immédiate. | Backlog borné. |

---

## 2. Résultats exécutés

### 2.1 Synthèse des contrôles

| Contrôle | Résultat | Interprétation |
|---|---:|---|
| `go build ./...` | **PASS** | Le module compile localement. |
| `go vet ./...` | **PASS** | Aucun défaut remonté par `vet`. |
| Tests ciblés `pkg/dotenv`, `internal/registry`, `internal/update`, `internal/cli`, `internal/profile` | **PASS** | Les unités principales ciblées sont vertes. |
| `go test -short ./internal/secret` | **PASS** | Le chemin court évite le test récursif problématique. |
| `go test -count=1 -timeout=90s ./...` | **INCONCLUSIF** | Aucun résultat final ; exécution locale Windows interrompue après environ 124 s. |
| `go test ./internal/secret` | **TIMEOUT** | Blocage dans `TestZeroizeNotOptimized`, qui lance un autre `go test` sans contexte ni délai (`internal/secret/secret_test.go:466-485`). |
| Couverture large `internal/...`, `pkg/...`, `tests` | **TIMEOUT** | Blocage dans `TestLaunchWithFallback_NoFallbackOnInterrupt`, puis dans l'E2E `TestE2E_ListProfiles`. |
| `npm test` dans `multiai-go/packaging/npm` | **PASS 25/25** | Inclut les nouveaux tests Windows PowerShell exécutés sur Windows. |
| `node --check` sur shim/helper/tests PATH | **PASS** | Syntaxe JavaScript valide. |
| `npm pack --dry-run --json` | **PASS** | Le tarball contient `lib/windows-path.js` et `scripts/ensure-user-path.ps1`. |
| `node scan-secrets.js` | **PASS** | 54 fichiers `.env` contrôlés, aucune clé réelle détectée. Ce scan n'analyse ni tout le source ni l'historique Git. |
| `npm audit --json` dans `multiai-go/docs` | **3 alertes** | 1 haute et 2 modérées, toutes dans l'outillage VitePress/Vite/esbuild de développement. |
| `govulncheck ./...` local | **INCONCLUSIF** | La commande n'a pas produit de verdict avant blocage ; ne pas conclure à l'absence de vulnérabilité. |
| `gofmt -l .` | **14 fichiers signalés** | Principalement cohérent avec les conversions CRLF locales ; absence d'un `.gitattributes` racine imposant LF. |
| `git diff --check` | **PASS avec avertissements CRLF** | Pas d'erreur d'espaces, mais de nombreux avertissements LF → CRLF. |

Le dépôt possède **41 fichiers de tests Go**, ce qui est un bon socle. La faiblesse n'est pas l'absence de tests, mais leur terminaison et leur représentativité sur les frontières processus/OS.

### 2.2 Couverture partielle observée avant timeout

Ces valeurs ne constituent pas une mesure globale, puisque l'exécution a été interrompue :

| Package | Couverture partielle |
|---|---:|
| `internal/catalog` | 94,9 % |
| `internal/env` | 95,6 % |
| `internal/migration/powershell` | 89,3 % |
| `pkg/dotenv` | 97,4 % |
| `internal/registry` | 78,7 % |
| `internal/onboarding` | 71,0 % |
| `internal/secret` | 67,0 % |
| `internal/profile` | 50,2 % |
| `internal/cli` | 42,4 % avant blocage |
| `internal/update` | 38,6 % |
| `internal/assets` | 20,0 % |
| `internal/menu` | 17,2 % |
| `internal/display`, `internal/i18n` | 0 % dans cette exécution |

Une release doit produire une couverture complète, reproductible et assortie d'un seuil. L'upload actuel de `coverage.out` sans seuil ne constitue pas une gate.

---

## 3. Revue du correctif PATH Windows

### 3.1 État du problème initial

Sous Windows, npm place le shim global directement dans son préfixe. Si ce préfixe n'est pas dans `PATH`, `npm install -g multiai` peut installer le package sans rendre `multiai` résoluble. Le smoke test historique lançait directement le JavaScript du package et masquait donc le défaut.

Un correctif non encore publié a été ajouté au worktree pendant cet audit :

- `multiai-go/packaging/npm/bin/multiai.js` traite l'installation explicite avant de réclamer le binaire natif temporaire ;
- `lib/windows-path.js` récupère le préfixe global npm et appelle un helper sûr ;
- `scripts/ensure-user-path.ps1` ajoute de façon idempotente ce préfixe au PATH **utilisateur** ;
- `package.json` embarque les deux fichiers et exécute les nouveaux tests.

### 3.2 Contrôles positifs validés

| Propriété | Preuve | Évaluation |
|---|---|---|
| Préfixe correct | `npm prefix --global`, avec propagation du `--prefix` personnalisé (`lib/windows-path.js:9-19`, `bin/multiai.js:87-93`). | Correct. |
| Portée non administrateur | `[Environment]::SetEnvironmentVariable(..., 'User')` (`ensure-user-path.ps1:222`). | Correct. |
| Pas de `setx` | Écriture .NET, donc pas de troncature/expansion destructive par `setx`. | Correct. |
| Entrée validée | Lecteur local `^[A-Za-z]:[\\/]`, refus de `;`, NUL, CR/LF, présence de `multiai.cmd` (`windows-path.js:55-67`, PS1 `:153-169`). | Correct, y compris contre UNC/device. |
| Appel sans injection | Le préfixe passe par `MULTIAI_PATH_ENTRY`, jamais concaténé à la ligne PowerShell (`windows-path.js:72-82`). | Correct. |
| Exécutables système fiables | PowerShell et `cmd.exe` sont résolus depuis `%SystemRoot%\System32` (`windows-path.js:21-45`). | Correct. |
| Idempotence | Normalisation casse/slash/guillemets/variables et contrôle des PATH User + Machine (`ensure-user-path.ps1:11-59,171-180`). | Correct. |
| Concurrence | Mutex puis relecture avant écriture (`ensure-user-path.ps1:190-213`). | Correct. |
| Vérification après écriture | Le PATH utilisateur est relu et contrôlé avant de renvoyer l'état effectif (`ensure-user-path.ps1:222-229`). | Correct. |
| Échappatoire entreprise | `MULTIAI_SKIP_PATH_UPDATE=1`, avec instruction manuelle (`windows-path.js:54`, `bin/multiai.js:100-102`). | Correct. |
| Échec fermé | Code PowerShell non nul, JSON malformé ou résultat incomplet déclenche une erreur (`windows-path.js:85-104`). | Correct. |
| Packaging | `npm pack --dry-run --json` contient les deux helpers ; 7 fichiers, 10 319 octets compressés. | Correct. |
| Tests | 25/25, dont PowerShell réel sur Windows, Unicode, casse, variable développée, PATH machine, conflit de shim et résolution par `cmd.exe`. | Bon socle. |

**Conclusion PATH :** le défaut initial ainsi que les deux réserves W-01/W-02 ont été corrigés pendant l'audit pour le parcours recommandé `npx --yes --allow-scripts=multiai multiai@latest install`. Le design évite volontairement une mutation persistante dans `postinstall`, ce qui est raisonnable. Il reste à valider le cycle réel dans une VM Windows vierge.

### W-01 — CORRIGÉ PENDANT L'AUDIT — Détection du shim réellement prioritaire

**État initial.** Le smoke test plaçait artificiellement le nouveau préfixe en tête de PATH. Il pouvait donc réussir malgré un ancien shim prioritaire.

**Correctif revu.** Le helper reconstruit désormais le PATH persistant dans l'ordre Machine puis User (`ensure-user-path.ps1:91-109`), recherche `multiai.cmd` dans le CWD puis chaque entrée (`:111-136`) et renvoie en JSON UTF-8 le PATH et le premier shim (`:9,139-151`). Le shim Node remplace son PATH de smoke par cette valeur exacte, compare le résultat au `<prefix>\multiai.cmd` attendu et échoue avec les deux chemins en cas de conflit (`bin/multiai.js:115-142`). Il exécute ensuite `cmd.exe /c multiai.cmd --version` dans ce même environnement (`:142-150`).

**Validation.** `npm test` repassé après correction : **25/25**. Le test PowerShell matérialise un shim concurrent antérieur et vérifie qu'il est renvoyé (`lib/windows-path.test.js:158-164`). Le test de smoke utilise un répertoire Unicode et le PATH fourni sans préfixage caché (`:170-190`).

**Statut.** Risque W-01 fermé par revue de code et tests locaux. La validation d'une nouvelle console réelle reste suivie par W-04.

### W-02 — CORRIGÉ PENDANT L'AUDIT — Refus des chemins UNC et device

**État initial.** `path.win32.isAbsolute` et `[IO.Path]::IsPathRooted` acceptaient également UNC et device paths.

**Correctif revu.** Les deux couches exigent maintenant un chemin commençant par un lecteur local : `/^[A-Za-z]:[\\/]/` côté Node (`windows-path.js:55`) et `^[A-Za-z]:[\\/]` côté PowerShell (`ensure-user-path.ps1:161-164`). Les contrôles existants sur séparateur, NUL, CR/LF et présence du shim restent actifs.

**Validation.** Les tests refusent explicitement `\\serveur\partage`, `\\?\C:\device` et `\\.\C:\device` avant tout spawn (`lib/windows-path.test.js:52-71`). **Statut W-02 fermé.**

### W-03 — P2 UX — Le contrat dépend de la commande d'installation

`npx ... multiai install` corrige le PATH ; un simple `npm install -g multiai` ne le fait pas, car le `postinstall` télécharge seulement le binaire. Le README doit dire exactement quelle commande automatise le PATH. À terme, `multiai doctor --fix-path` ou un bootstrap `npx multiai setup-path` permettrait de réparer une installation globale déjà présente.

### W-04 — Gate manquante — Pas encore d'E2E sur Windows vierge

Les tests actuels couvrent la logique, le conflit et l'Unicode en mode `Plan`, mais pas le cycle réel registre utilisateur → fermeture du processus installateur → nouvelle console → résolution par nom. Avant publication : VM Windows propre, PATH sans préfixe npm, préfixe avec espaces/Unicode, utilisateur standard, installation, nouvelle session `cmd` et PowerShell, puis `where multiai` et `multiai version`.

---

## 4. Bloqueurs P1 sécurité et release

### S-01 — Configuration `.multiai.yaml` non approuvée : RCE et détournement de secrets

**Preuves.** `FindProjectConfig` recherche automatiquement `.multiai.yaml/.yml` dans le dossier courant puis tous ses parents (`internal/profile/project.go:12-47`). `MergeProjectConfig` applique sans approbation `overrides`, `clear_env`, `args` et `hooks` (`:50-73`). Les hooks sont transmis par défaut ; seul `MULTIAI_NO_HOOKS=1` les désactive (`cmd/multiai/main.go:451-460`). Ils sont ensuite exécutés via PowerShell, cmd, bash, zsh ou sh (`internal/cli/hooks.go:39-88,91-136`).

La résolution des secrets du credential store a lieu avant le lancement du CLI (`internal/cli/launcher.go:81-93,300-331`). Un projet peut donc changer une URL de fournisseur tout en laissant la sentinelle du secret intacte : le vrai secret sera résolu puis envoyé par le CLI vers l'endpoint imposé. Un hook simple peut aussi exécuter n'importe quel programme avec les droits de l'utilisateur.

**Scénario.** L'utilisateur clone un dépôt, se place dans le dossier et exécute sa commande habituelle `multiai launch -p ...`. Aucune confirmation n'indique qu'un fichier du dépôt ou d'un parent va modifier l'environnement et exécuter un hook.

**Impact.** Exécution de code utilisateur, exfiltration de credentials, altération du comportement réseau. La liste blanche `claude/codex/opencode` ne protège pas les hooks ni les endpoints.

**Remédiation.**

1. Considérer toute configuration projet comme non fiable à la première rencontre ; afficher chemin canonique, source et diff, puis stocker une décision de confiance liée au chemin et à une empreinte.
2. Désactiver les hooks par défaut dans un dépôt non approuvé et en non-interactif ; exiger `--allow-project-hooks` ou une confiance persistée explicite.
3. Protéger les variables sensibles de routage/authentification, ou au minimum exiger une confirmation dédiée lorsqu'une config projet les modifie.
4. Deep-copier `Env` et les slices : `merged := *base` partage actuellement la map et peut modifier le profil source en mémoire (`project.go:51-63`).
5. Ajouter `multiai project inspect|trust|untrust` et un mode CI fail-closed.

**Tests.** Dépôt non fiable, config dans un parent, mode non interactif, changement de base URL, hook avant/après, empreinte modifiée, révocation, symlink/junction et absence totale d'accès au credential store avant confiance.

### S-02 — Traversée de répertoire via le registre communautaire

**Preuves.** Le nom d'un profil provient de l'index distant et est utilisé directement dans `filepath.Join(profilesDir, entry.Name+".env")` (`internal/registry/install.go:22-43`). Le handler calcule le même chemin avant l'installation (`cmd/multiai/cmd_registry.go:426-451`). Aucune regex de nom, aucun `filepath.Rel` ni contrôle de confinement ne bloque `..`, séparateurs, volume ou chemin absolu.

L'index peut aussi fournir un `download_url` arbitraire (`internal/registry/client.go:132-149`), et le profil est lu sans limite (`:156-171`). Le SHA-256 est optionnel et provient du même index ; `--no-verify` le désactive explicitement (`cmd_registry.go:440-448`).

**Impact.** Un index/cache compromis peut créer ou, avec `--force`, écraser un fichier `.env` hors du répertoire des profils, dans tout emplacement accessible à l'utilisateur. Le checksum du même canal ne fournit aucune provenance.

**Remédiation.**

- imposer par exemple `^[a-z0-9][a-z0-9_-]{0,63}$` à `ProfileEntry.Name` dès le décodage ;
- nettoyer puis vérifier `filepath.Rel(profilesDir, dest)` : refuser résultat absolu, `..` ou changement de volume ;
- limiter le téléchargement (par exemple 1 MiB) et le JSON d'index ;
- imposer HTTPS et, en production, une origine autorisée ;
- signer/versionner l'index, rendre le checksum obligatoire et supprimer `--no-verify` du parcours standard ;
- ajouter des tests Windows et Unix pour `../`, `..\`, UNC, drive-relative, Unicode confusable et liens de répertoire.

### S-03 — Auto-update exécutable sans authenticité obligatoire et sans installation persistante

**Preuves d'authenticité.** GoReleaser annonce une signature Cosign mais le bloc `signs:` est commenté (`multiai-go/.goreleaser.yaml:13-24,71-75`). L'updater ne vérifie Cosign que si les deux assets existent (`internal/update/update.go:251-316`) ; même alors, il ignore l'absence du binaire `cosign` sauf si l'utilisateur a défini `MULTIAI_REQUIRE_COSIGN=1` (`:361-388`). En pratique, le SHA-256 et l'archive viennent du même release GitHub. `packaging/npm/install.js:181-195` applique le même modèle sans vérification de signature.

**Preuves fonctionnelles.** L'archive est extraite dans un dossier temporaire (`update.go:334-358`) puis `ExecBinary` démarre ce fichier et appelle `os.Exit(0)` (`:391-403,498-502`). Aucun remplacement atomique de l'exécutable installé n'existe. Au prochain lancement, l'ancien binaire repart ; le cache peut empêcher une nouvelle tentative pendant une heure. De plus, `go update.Check(...)` est lancé dans une goroutine à chaque démarrage (`cmd/multiai/main.go:203-207`) : un téléchargement terminé peut appeler `os.Exit` au milieu d'un menu ou d'une session enfant.

**Impact.** Compromettre le canal de release ou un compte de publication permet l'exécution de code. Sans compromission, la fonction intitulée « installation » ne persiste pas la version et peut interrompre brutalement une commande active. Les dossiers temporaires de succès ne sont pas nettoyés.

**Remédiation.**

1. Désactiver l'auto-exécution en arrière-plan ; au démarrage, vérifier/notifier seulement.
2. Rendre `multiai update` conscient du canal : `npm`, Homebrew, Scoop, APT/AUR doivent déléguer au gestionnaire. Pour un binaire autonome, utiliser remplacement atomique, rollback et vérification de l'exécutable réellement installé.
3. Produire systématiquement signature Cosign et provenance ; vérifier de façon fail-closed avec identité/issuer pinés. La vérification ne doit pas dépendre d'un `cosign` préinstallé silencieusement absent.
4. Appliquer la même racine de confiance au bootstrap npm ; un checksum téléchargé à côté de l'archive ne suffit pas.
5. Limiter tailles téléchargées et extraites ; interdire une archive inattendue.

**Tests.** Mise à jour npm/standalone, redémarrage ultérieur, crash entre copie et rename, rollback, signature absente/invalide, identité Cosign incorrecte, release incomplète, session longue jamais interrompue, nettoyage des temporaires.

### R-01 — Le workflow de release exécuté n'est pas la copie renforcée

**Preuves.** GitHub n'exécute que `.github/workflows`. La copie source `multiai-go/.github/workflows/release.yml` possède un job `preflight`, vérifie que le tag descend de `master`, exécute `go test -race ./...` et `go vet ./...`, impose `needs: [preflight]` et un environnement `release` (`:31-59`). La copie racine réellement exécutée commence directement par `goreleaser` et ne contient pas ces gates (`.github/workflows/release.yml:31-39`). Les deux fichiers diffèrent de 160 lignes. Aucun job CI n'exécute `scripts/sync-workflows.ps1 -Check` malgré le mécanisme prévu (`sync-workflows.ps1:1-14,35-69`).

Autres écarts de chaîne :

- commentaire « toutes les actions sont pinées » mais `anchore/sbom-action@v0.18.0` ne l'est pas par SHA (`release.yml:87-92`) ;
- CI : `gosec@latest ... || true` n'est pas bloquant (`.github/workflows/ci.yml:57-64`) ;
- `govulncheck@latest` et `version: latest` rendent les runs non reproductibles (`:60-64,100-101`) ;
- Gitleaks est désactivé par `if: false` (`:200-210`) ;
- le workflow n'installe pas explicitement Node 24.14 avant `npm test`, alors que le package l'exige ;
- des expressions de tag sont injectées directement dans des scripts shell de release ; utiliser des variables `env:` intermédiaires ;
- la première connexion AUR utilise `StrictHostKeyChecking accept-new` plutôt qu'une clé hôte pinée (`release.yml:142-153`).

**Impact.** Un simple tag peut publier des artefacts sans preuve que le commit a passé les tests attendus. Les mainteneurs peuvent renforcer la mauvaise copie et croire la production protégée. Les scanners donnent une impression de couverture supérieure à la réalité.

**Remédiation.** Synchroniser les workflows, ajouter `sync-workflows.ps1 -Check` comme première gate racine, exiger tests multi-OS verts pour le SHA tagué, protéger l'environnement de release, pinner actions et outils par SHA/version, rendre gosec et Gitleaks bloquants, ajouter `setup-node` 24.14.x, puis tester le workflow avec un tag de pré-release non publiant.

---

## 5. Constats P2/P3

### Q-01 — Tests de processus non bornés et suite locale non déterministe — P2

`TestZeroizeNotOptimized` lance récursivement `go test` avec `exec.Command` sans `context` (`internal/secret/secret_test.go:466-485`). Les helpers CLI/E2E démarrent également des processus sans délai global (`internal/cli/fallback_test.go:206+`, `tests/e2e_test.go:35,73`). Sous couverture Windows, des enfants restent bloqués au démarrage ou à l'attente.

**Action :** utiliser `exec.CommandContext`, délais par test, `t.Cleanup` qui tue l'arbre de processus, protocole helper explicite et logs capturés en cas de timeout. Séparer tests unitaires, intégration OS et E2E ; aucun job ne doit pouvoir dépasser son budget sans diagnostic.

### Q-02 — Scanners CI incomplets ou non bloquants — P2

Le scan `.env` local est utile mais étroit. Avec Gitleaks désactivé, ni le source complet ni l'historique Git ne sont protégés. `gosec || true` ne peut empêcher une release. `govulncheck` local n'ayant pas terminé, l'état des CVE Go reste inconnu.

**Action :** Gitleaks CLI open source piné, gosec bloquant avec baseline justifiée, govulncheck piné, SARIF/artefacts, politique d'exception datée et gate release qui exige leurs succès.

### Q-03 — Dépendances docs vulnérables, sans correctif stable disponible — P2

`npm audit` sur `multiai-go/docs` remonte :

- **haute** : Vite `server.fs.deny` bypass sur chemins Windows alternatifs, GHSA-fx2h-pf6j-xcff ;
- **modérée** : traversal/fuite via sourcemaps de dépendances optimisées, GHSA-4w7w-66w2-5vf9 ;
- **modérée** : divulgation NTLMv2 via chemin UNC dans `launch-editor`, GHSA-v6wh-96g9-6wx3 ;
- **modérée transitive** : serveur de développement esbuild, GHSA-67mh-4wv8-2f99.

VitePress 1.6.4 est direct et aucun `fixAvailable` n'est proposé dans l'audit observé. Ce sont des dépendances de développement, pas du binaire distribué.

**Action :** ne jamais exposer le serveur docs sur une interface réseau non fiable, ne pas l'exécuter sur un dépôt non approuvé, surveiller la migration de toolchain et ajouter Dependabot pour `/multiai-go/docs`. La configuration actuelle ne surveille que Go, Actions et `/multiai-powershell` (`.github/dependabot.yml:6-24`).

### Q-04 — Téléchargements non bornés en mémoire — P2

`io.ReadAll` est utilisé pour profils et assets d'update (`internal/registry/client.go:167`, `internal/update/update.go:520-538`) ; l'installateur npm accumule tous les chunks puis `Buffer.concat` (`packaging/npm/install.js:72-104`). L'index JSON n'a pas non plus de limite. Une réponse volumineuse peut épuiser mémoire/disque avant même le contrôle de checksum.

**Action :** vérifier `Content-Length`, lire via limite dure + 1 octet, plafonner taille compressée et décompressée, limiter le nombre de fichiers d'archive, supprimer le temporaire sur toute branche et tester les dépassements.

### Q-05 — Résolution de shells par PATH dans d'autres surfaces — P2

Le nouveau helper PATH choisit correctement PowerShell depuis `%SystemRoot%`, mais l'extracteur npm appelle encore `execFileSync('powershell', ...)` (`packaging/npm/install.js:119-132`) et les hooks recherchent `powershell`, `cmd`, `bash` ou `sh` par nom (`internal/cli/hooks.go:61-76,112-127`). Une entrée PATH antérieure peut donc détourner ces exécutables.

**Action :** sous Windows, utiliser les chemins système fiables pour PowerShell/cmd. Pour les hooks explicitement approuvés, afficher le shell résolu et son chemin ; en contexte entreprise, permettre une allowlist administrée.

### Q-06 — Masquage de secret trop révélateur — P3

`MaskSecret` expose les quatre premiers et quatre derniers caractères dès que la valeur dépasse huit caractères (`internal/env/env.go:124-132`). Un secret de neuf caractères est donc révélé à 8/9.

**Action :** n'afficher que les quatre derniers caractères, ou deux premiers/deux derniers uniquement au-delà d'une longueur élevée ; sinon remplacer toute la valeur. Tester Unicode et chaînes courtes.

### Q-07 — Écriture `.env` en 0644 et remédiation `--allow-plaintext` inexistante — P2

Le wizard conseille `--allow-plaintext` en cas d'échec du store (`internal/config/wizard.go:348-350`), mais l'appel force `allowPlaintext=false` et aucun parseur CLI n'expose ce flag (`:296-299`, `cmd/multiai/main.go:281-316`). Si le chemin plaintext était activé, `setEnvVarInFile` écrit avec `0644` (`wizard.go:369-402`), trop permissif sur un poste Unix multi-utilisateur.

**Action :** conserver le refus du plaintext par défaut ; soit supprimer la fausse instruction, soit implémenter une exception explicite avec confirmation. Utiliser `0600` pour tout fichier susceptible de contenir un secret et préserver/réduire les permissions existantes.

### Q-08 — Fins de ligne Windows et outillage — P3

Le module demande LF dans `multiai-go/.editorconfig`, mais aucun `.gitattributes` racine ne le garantit. Sur cette copie Windows, `gofmt -l .` signale 14 fichiers et Git avertit de conversions CRLF.

**Action :** ajouter au niveau racine une politique ciblée, par exemple `*.go text eol=lf`, ainsi que YAML/JS/PS1 selon les besoins ; renormaliser dans un commit dédié après vérification. Cela évite de confondre bruit de ligne et défaut Go réel.

### Q-09 — Contrat Go/documentation/version incohérent — P2

Le module déclare Go 1.24 (`multiai-go/go.mod:3`), le CI utilise 1.26.5, tandis que le README module annonce encore Go 1.23 (`multiai-go/README.md:228`). Le README racine propose `go install github.com/lrochetta/multiai/multiai-go/cmd/multiai@latest`, alors que plusieurs guides proposent `go install github.com/lrochetta/multiai@latest`. Le binaire compilé hors GoReleaser conserve par défaut `version = "0.6.0"` (`cmd/multiai/main.go:28-31`).

**Action :** choisir et tester une seule commande Go dans un environnement vierge, aligner module/tags/emplacement, définir la version minimale Go unique et dériver la version du build plutôt que d'un fallback de release ancien.

---

## 6. Matrice de risques

| ID | Risque | Probabilité | Impact | Priorité | Propriétaire suggéré |
|---|---|---:|---:|---:|---|
| S-01 | Projet non fiable → hook/RCE ou endpoint hostile | Haute | Critique | **P1** | Architecture + sécurité |
| S-02 | Index registre → écriture hors répertoire | Moyenne | Élevé | **P1** | Développement |
| S-03 | Release compromise → binaire exécuté sans signature obligatoire | Moyenne | Critique | **P1** | Release engineering |
| S-03b | Auto-update temporaire/interruption de session | Haute | Élevé | **P1** | Développement |
| R-01 | Tag publié sans gates de la copie renforcée | Moyenne | Critique | **P1** | DevOps |
| W-01 | Ancien shim prioritaire masqué par le smoke | — | — | **Fermé pendant audit** | Packaging Windows |
| W-02 | UNC/device persisté dans PATH | — | — | **Fermé pendant audit** | Packaging Windows |
| Q-01 | Tests enfants bloqués, faux signal CI | Haute | Moyen | **P2** | QA |
| Q-03 | Vite dev server vulnérable | Faible en prod | Élevé en dev exposé | **P2** | Documentation |
| Q-04 | Réponse réseau volumineuse → DoS local | Moyenne | Moyen | **P2** | Développement |
| Q-05 | Shell détourné via PATH | Faible à moyenne | Élevé | **P2** | Sécurité |

---

## 7. Gates obligatoires avant v0.6.7

### Gate A — Sécurité applicative

- [ ] Confiance explicite pour `.multiai.yaml`, hooks désactivés tant que non approuvés.
- [ ] Deep copy du profil et tests de non-contamination.
- [ ] Validation stricte des noms du registre + preuve de confinement du chemin.
- [ ] Limites de taille réseau minimales sur index/profil/update/bootstrap.

### Gate B — Mise à jour et chaîne de livraison

- [ ] Aucun `os.Exit` depuis une vérification d'update en arrière-plan.
- [ ] Mise à jour réellement persistante ou simple notification jusqu'à implémentation correcte.
- [ ] Artefacts signés ; vérification obligatoire et documentée pour updater et npm.
- [ ] Workflow racine synchronisé, tests/race/vet obligatoires, environnement release protégé.
- [ ] Gosec et secret scan bloquants ; outils/actions pinés.

### Gate C — Installation Windows

- [x] Ajout idempotent du préfixe npm au PATH utilisateur.
- [x] Pas de `setx`, pas d'admin, entrée transmise hors ligne de commande.
- [x] Helper présent dans le tarball ; 25 tests npm verts.
- [x] Refus UNC/device dans Node et PowerShell.
- [x] Détection du premier shim réellement résolu sans préfixer artificiellement PATH.
- [ ] E2E Windows vierge, nouvelle console cmd + PowerShell, utilisateur standard.
- [x] Documentation explicite : seul le parcours `npx ... install` répare automatiquement PATH.

### Gate D — Qualité reproductible

- [ ] `go test -race ./...` vert sur Ubuntu, macOS et Windows pour le même SHA.
- [ ] Tous les tests de sous-processus bornés et diagnostiquables.
- [ ] `govulncheck`, `gosec`, Gitleaks et `npm audit` exécutés avec politique d'exception.
- [ ] Couverture complète calculée ; seuil initial réaliste, puis hausse progressive.
- [ ] `npm pack` puis installation du **tarball**, pas seulement test des sources.

La mémoire projet impose déjà de ne pas taguer/publier avant une matrice macOS/Ubuntu entièrement verte. Cet audit renforce cette décision et ajoute les gates Windows et sécurité ci-dessus.

---

## 8. Plan de tests prioritaire

### 0–2 jours

1. Tests négatifs registre : traversal Windows/Unix, absolus, volumes, UNC, symlinks et confinement final.
2. Tests confiance projet : aucun hook/override sensible avant approbation ; fail-closed en CI.
3. Compléter l'E2E PATH dans une VM : application réelle, conflit CWD/Machine/User et nouvelle console.
4. Remplacer les `exec.Command` de tests par `CommandContext` avec diagnostics de timeout.

### 3–7 jours

1. Installer le tarball npm dans une VM Windows vierge et vérifier une nouvelle console.
2. Tests de mise à jour par canal et redémarrage de l'ancien chemin installé.
3. Vérification de signature absente, invalide, mauvaise identité et bonne identité.
4. Tests de limites HTTP/archives et nettoyage des fichiers temporaires.
5. Dry-run complet du workflow de release synchronisé.

### Après stabilisation

- fuzzing des parseurs d'index, chemins, checksums et archives ;
- snapshots de contrats CLI/JSON/i18n ;
- tests d'installation publics par canal ;
- tests de panne : proxy, CA entreprise, GitHub indisponible, disque plein, PATH proche de la limite, verrou concurrent.

---

## 9. Score détaillé

| Dimension | Poids | Note | Justification |
|---|---:|---:|---|
| Qualité du code et conception locale | 20 % | 7,0 | Go lisible, dépendances limitées, écritures atomiques, séparation interne correcte. |
| Tests et reproductibilité | 20 % | 6,0 | 41 fichiers de tests et bonnes unités, mais suite complète/coverage non terminantes localement. |
| Sécurité applicative | 25 % | 4,0 | Isolation/secrets solides, neutralisés par la confiance projet et le registre. |
| Supply chain et release | 20 % | 3,0 | Signature désactivée, updater fail-open, workflow racine divergent, scanners incomplets. |
| Installation et UX Windows | 15 % | 8,0 | Mutation et résolution persistantes corrigées ; seul l'E2E d'une console vierge reste à exécuter. |
| **Score pondéré** | **100 %** | **5,4/10** | Potentiel post-P1 estimé à 8+/10. |

---

## 10. Limites de l'audit

- Le rapport porte sur la révision locale et les changements non commités visibles pendant l'audit ; le correctif PATH n'est pas encore un artefact public.
- Aucun test `Apply` n'a modifié le PATH utilisateur réel ; cette mutation doit être validée dans une VM jetable.
- `govulncheck` et la suite Go complète n'ont pas terminé localement : leur statut est **inconnu**, pas vert.
- L'audit n'est pas un pentest réseau du compte GitHub, des secrets Actions, des stores OS ou des futurs dépôts de packages.
- Les vulnérabilités npm sont celles retournées le 2026-07-14 et peuvent évoluer.

---

## Conclusion Sentinel

multiai possède les ingrédients d'un excellent routeur local : son périmètre d'exécution est compréhensible, les secrets ne sont pas volontairement globalisés, et le correctif Windows traite le vrai préfixe npm sans `setx` ni droits administrateur. Pendant l'audit, il a aussi été durci pour refuser UNC/device et vérifier le premier shim réellement résolu dans le PATH persistant exact.

La release ne doit toutefois pas partir tant que des données de dépôt non approuvées peuvent déclencher des hooks, qu'un index distant peut sortir du répertoire des profils, et qu'un updater non authentifié peut arrêter le processus pour exécuter une copie temporaire. La priorité n'est pas d'ajouter des fonctions : elle est de rendre les frontières de confiance explicites et la livraison vérifiable.

> **Critère de sortie : un dépôt non fiable ne s'exécute jamais implicitement, un chemin distant ne sort jamais de son répertoire, un artefact non signé ne s'exécute jamais, et une installation Windows n'annonce jamais un succès sans résoudre exactement le shim attendu dans une nouvelle session.**
