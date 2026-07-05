# Audit v0.3.0 — Architecture

Date 2026-07-04 · Auditeur Forge (Architect-Dev — architecture logicielle) · Score : 3.5/10 · Méthode : audit BMAD+ parallèle + contre-vérification adversariale.

---

# Audit Architecture — multiai (Go primaire + PowerShell legacy)

**Auditeur** : Forge (BMAD+ Architect-Dev) — mandaté par Nexus
**Date** : 2026-07-05
**Périmètre** : `multiai-go/` (architecture), convergence `multiai-powershell/`, modèle de configuration, extensibilité, packaging, delta vs audit v0.2.1 (2026-06-23)
**Version auditée** : binaire Go `0.2.1` (`multiai-go/cmd/multiai/main.go:18`), npm publié `0.3.0` (`multiai-powershell/package.json:3`), packaging `0.5.0`

---

## Résumé

Le découpage en packages de `multiai-go/` est visuellement académique (cmd/internal/pkg, fichiers courts, zéro cycle d'imports, une seule dépendance `yaml.v3`), mais l'audit révèle une architecture **Potemkine** : une fraction majeure d'`internal/` n'est jamais atteinte par `cmd/multiai/main.go`, le flux fonctionnel central (configurer une clé → lancer un CLI) est cassé par un credential store *write-only*, et toute l'innovation v0.3.0 (14 fournisseurs, fusion OpenRouter, fallback chains, cost logging) a été livrée **uniquement dans l'implémentation PowerShell « legacy »**, faisant diverger les deux implémentations au point que la « primaire » est maintenant en retard de deux versions fonctionnelles sur la « legacy ». La chaîne de distribution (CI fantôme, profils exclus de git, packaging à trois versions différentes, formule Homebrew structurellement incapable de compiler) montre que le binaire Go n'a jamais été livré par un canal automatisé. Note sévère : **3.5/10** — le squelette est bon, presque tout le reste est claim non tenu.

---

## Forces

| Force | Preuve |
|---|---|
| Graphe d'imports sain, aucune dépendance circulaire | `main → {cli, config, menu, profile}` ; `config → {cli, profile, secret, dotenv}` ; `profile → dotenv` ; `secret`, `openrouter`, `logging` → stdlib seule |
| Hygiène de dépendances exemplaire | `multiai-go/go.mod:5` — unique require `gopkg.in/yaml.v3 v3.0.1` ; `go.sum` minimal |
| `pkg/dotenv` : vraie brique réutilisable, testée (11 tests) | `pkg/dotenv/dotenv.go:20-65`, `dotenv_test.go` |
| Modèle d'isolation env par liste blanche simple et lisible | `internal/env/env.go:9-21` (AllowedEnvVars), `env.go:34-60` (BuildCleanEnv) |
| Fichiers courts, pas de god-file (max 323 lignes : main.go) | `wc -l` : 3 351 lignes Go au total |
| Corrections v0.2.1 réelles sur le launcher : exit code propagé, forwarding signaux, whitelist immuable, mutex sur le store | `internal/cli/launcher.go:18,117-142,147-165` ; `internal/secret/secret.go:39,110-147` |
| Écriture atomique des .env (temp + rename) | `internal/config/wizard.go:278-287` |

---

## Constats détaillés

### C1 — CRITIQUE : le flux config → launch est cassé (credential store write-only)
`multiai config` remplace la valeur de la clé dans le fichier .env par une sentinelle : `internal/config/wizard.go:269` (`lines[i] = varName + "=__MULTIAI_CREDSTORE__"`), puis stocke la vraie valeur dans le credential store (`wizard.go:295`, `store.Set(...)`). Or **aucun code de production n'appelle jamais `store.Get`** : ni `env.BuildCleanEnv` (`internal/env/env.go:34-60`), ni `validateSecrets` (`internal/cli/launcher.go:217-229`) ne résolvent la sentinelle. `dotenv.IsPlaceholder` (`pkg/dotenv/dotenv.go:73-93`) ne reconnaît pas `__MULTIAI_CREDSTORE__` comme placeholder. Conséquence : après configuration, le lancement exporte littéralement `ANTHROPIC_API_KEY=__MULTIAI_CREDSTORE__` vers le CLI enfant — le produit ne fonctionne plus après son propre wizard. Le seul endroit qui connaît la sentinelle est `internal/onboarding/wizard.go:21`… qui est lui-même du code mort (voir C2).

### C2 — CRITIQUE : ~40% d'`internal/` est du code mort, vendu comme features
`main.go` charge les profils exclusivement via `profile.LoadDir` (.env seulement) : `cmd/multiai/main.go:134,147,158,187`. En conséquence :
- **Profils YAML** : `LoadYAML`/`LoadDirYAML`/`LoadAllProfiles` (`internal/profile/yaml.go:51,76,101`) ne sont appelés que par les tests (`tests/integration_test.go:108,126`).
- **Héritage `.multiai.yaml`** : `FindProjectConfig`/`MergeProjectConfig` (`internal/profile/project.go:13,51`) — zéro appelant hors tests. Le champ `Extends` (`yaml.go:31`) n'est **lu nulle part** : l'héritage annoncé n'est pas implémenté du tout.
- **Plugin hooks** : `LaunchOptions.Hooks` n'est jamais renseigné (`main.go:266-273`) et `yamlToProfile` (`yaml.go:130-171`) **perd `py.Hooks`** (le type `Profile` n'a pas de champ Hooks). Les 164 lignes de `internal/cli/hooks.go` sont inatteignables.
- **`internal/openrouter`** (client, cache 1h) : importé par aucun code de production (seule mention externe : une URL dans `wizard.go:89`).
- **`internal/onboarding`** : `RunWelcome`/`IsFirstRun` jamais appelés depuis main — le « wizard premier démarrage » (fix v0.2.1 #13) n'est jamais déclenché. Incohérence interne : `markFirstRunDone` écrit un marqueur (`onboarding/wizard.go:68-73`) que `IsFirstRun` ne consulte pas.
- **`internal/logging`** : uniquement importé par onboarding (mort) — le « logger structuré » (qui n'est d'ailleurs pas structuré : lignes texte, `logger.go:52`) ne log rien en pratique.

Le README racine annonce pourtant tout cela comme livré : « Profils YAML + .multiai.yaml par projet avec héritage », « Plugin hooks before_launch/after_launch » (`README.md:169-171`).

### C3 — CRITIQUE : la v0.3.0 n'existe pas dans l'implémentation « primaire »
`CHANGELOG.md:7-27` (v0.3.0) annonce `multiai models`, `multiai search`, `multiai compare`, fusion, régions, fallback chains, cost logging. Le switch de commandes de `main.go:126-182` ne connaît que `version/help/list/launch/config/completion`. Tout est en PowerShell : `ProviderCatalog` 14 fournisseurs (`multiai-powershell/code-router.ps1:93`), `Write-CostLog` (`:1010`), chaîne `FALLBACK` (`:1136-1156`). Le Go reste à 5 fournisseurs hardcodés (`internal/config/wizard.go:58-94`) et 17 profils (`multiai-go/configs/profiles/` : 00→57) contre 38 côté PS (00→83). Le CHANGELOG v0.3.0 attribue même mensongèrement au Go : « Go : package internal/openrouter/ (client API, cache, search) » — le package existe (96 lignes) mais sans search, sans fusion, et n'est pas branché. L'historique git le confirme : depuis `dd8d9c7` (v0.2.1), plus aucun commit substantiel ne touche `multiai-go/` alors que `multiai-powershell/` en reçoit six.

### C4 — CRITIQUE : les profils et la doc sont exclus de git — un clone frais est inutilisable
`.gitignore:2` (`*.env`, motif non ancré) ignore **tous** les profils : `git ls-files multiai-go/configs/` retourne **0 fichier** ; côté PS, les 17 profils de base 00-57 sont également non trackés (seuls 01-03 et 60-83 ont été force-addés, cf. commit `72c862c`). Un `git clone` produit un binaire Go qui sort en code 2 sur toutes les commandes (`main.go:135` : « cannot read profiles directory »). Le motif `docs/` du `.gitignore` (non ancré lui aussi) exclut de surcroît **tout le site VitePress** (`git ls-files multiai-go/docs/` = 0) pourtant annoncé « Site VitePress : 16 pages » (`CHANGELOG.md:119`).

### C5 — HAUTE : CI/CD fantôme
Les workflows vivent dans `multiai-go/.github/workflows/{ci,release}.yml` — GitHub n'exécute que `.github/` **à la racine du repo**, qui n'existe pas (`NO root .github` vérifié). Double verrou : `ci.yml:5` déclenche sur `branches: [main]` alors que le repo n'a que `master` (`git branch -a`). `release.yml:46-48` contient un `cd ../../multiai-powershell` qui sort du workspace, et le job homebrew-update est un `echo` placeholder (`release.yml:58-62`). La claim README « CI/CD : lint → test (6 OS × Go) → security → build → benchmark » (`README.md:200`) et la « force confirmée » de l'audit v0.2.1 sont donc factuellement fausses : **cette CI n'a jamais tourné**.

### C6 — HAUTE : modèle de distribution incompatible avec le modèle de données
Les profils sont cherchés à côté de l'exécutable ou dans le cwd (`main.go:21-37`), sans `go:embed` ni répertoire utilisateur (`~/.multiai/profiles`). Or aucun canal ne les installe : `scripts/install.sh:34-42` ne copie que le binaire ; `packaging/deb/postinst` n'installe que les complétions ; `packaging/npm/install.js` ne télécharge que le binaire. La formule Homebrew (`packaging/homebrew/multiai.rb:8,22-23`) télécharge le tarball du repo et lance `go build ./cmd/multiai/` **depuis la racine** — où il n'y a ni `go.mod` ni `cmd/` (ils sont sous `multiai-go/`) : compilation impossible. Pour la même raison, `go install github.com/lrochetta/multiai@latest` (`README.md:73`) ne peut pas fonctionner (module déclaré `github.com/lrochetta/multiai` dans `multiai-go/go.mod:1`, mais placé en sous-répertoire). Chaque méthode d'installation documentée est cassée.

### C7 — HAUTE : chaos de versions (5 sources de vérité)
`main.go:18` = const `0.2.1` ; `internal/menu/interactive.go:18` = « v0.2.1 » hardcodé dans le titre ; `internal/openrouter/client.go:38` = User-Agent `multiai/0.2.1` ; `Makefile:2` = `0.2.0-dev` ; npm publié = `0.3.0` (`multiai-powershell/package.json:3`) ; packaging = `0.5.0` (`packaging/deb/control:2`, `packaging/npm/package.json:3`, `packaging/npm/install.js:10`, `multiai.rb:8`, `scripts/install.sh:6`). Bonus : le badge README « Go 1.23 » (`README.md:7`) contredit `go.mod:3` (`go 1.22`). Défaut structurel aggravant : `.goreleaser.yml:15` et le Makefile injectent `-X main.version=...` sur une **const** (`main.go:18`) — l'injection ldflags est silencieusement sans effet, un binaire releasé afficherait toujours 0.2.1.

### C8 — HAUTE : le contrat de profil diverge entre Go et PowerShell (mêmes fichiers, sémantique différente)
Les .env sont censés être le format partagé, mais :
- **Expansion `%VAR%`** : PS l'implémente (`code-router.ps1:414-427`, `Expand-RouterValue`) ; Go n'expanse que `$VAR/${VAR}` (`internal/env/env.go:24-31`, `os.Expand`). Six profils Go utilisent `%USERPROFILE%` (ex. `configs/profiles/30-claude-deepseek-v4-pro.env:12`) → sous Go, le CLI enfant reçoit littéralement `CLAUDE_CONFIG_DIR=%USERPROFILE%\.claude-deepseek-v4pro`.
- **Clés métadonnées** : PS traite `FALLBACK` et `REGION` comme métadonnées (`code-router.ps1:87`) ; le `MetadataKeys` Go (`internal/profile/profile.go:33-38`) ne les connaît pas → un profil v0.3.0 chargé par Go **exporterait FALLBACK/REGION comme variables d'environnement** vers le processus enfant.
- **`SKIP_SECRET_CHECK`** : filtré comme métadonnée en Go (`profile.go:37`) mais non honoré par `validateSecrets` (`launcher.go:217-229`), alors que PS l'honore (`code-router.ps1:458`).

### C9 — HAUTE : stores « natifs » = façades, contrat asymétrique
Les trois implémentations plateforme délèguent toutes à `encryptedFileStore` avec des commentaires avouant la simulation (« In production, this would use golang.org/x/sys/windows », `store_windows.go:18-19` ; idem `store_darwin.go:16-17`). Pire : Windows/macOS `Set()` encode la valeur en **base64** (`store_windows.go:44`, `store_darwin.go:31`) mais `Get()` ne décode jamais → aller-retour corrompu ; Linux n'encode pas. `store_windows.go:20-37` exécute `cmdkey /list` puis prend **la même branche fallback dans les deux cas** (branche morte). La claim « Credential store natif : AES-256-GCM + Windows/macOS/Linux » (`README.md:156`) est fausse sur la partie « natif ». Par ailleurs `DeriveKey` PBKDF2 (`internal/secret/crypto.go:18`) n'est appelée que par les tests — la master key reste des octets aléatoires en clair dans `.masterkey` (`secret.go:51-63`).

### C10 — MOYENNE : `internal/cli` = package fourre-tout, présentation couplée partout
`internal/cli` mélange quatre responsabilités : lancement de processus (`launcher.go`), helpers d'affichage (`display.go:61-78`), exécution de hooks (`hooks.go`), scripts de complétion (`completion.go`). `config`, `menu` et `onboarding` l'importent uniquement pour `PrintInfo/PrintWarning` — la couche présentation est soudée à la couche exécution. Un renommage en `internal/ui` + `internal/launch` s'impose.

### C11 — MOYENNE : duplications structurelles (drift garanti)
- Whitelist de commandes en double : `cli/launcher.go:18` (`AllowedCommands`) vs `profile/project.go:101-104` (`isAllowedCommand`, map locale).
- Catalogue fournisseurs en triple : `config/wizard.go:58-94` (Go, 5 providers), `code-router.ps1:93` (PS, 14 providers), chaîne hardcodée « 5 fournisseurs » dans `onboarding/wizard.go:40`.
- Shortcuts hardcodés dans les scripts de complétion statiques (`cli/completion.go:18`) — déjà désynchronisés des 38 profils PS.
- Politique d'expansion incohérente : `env.safeExpandEnv` whitelist-only (`env/env.go:24`) vs `MergeProjectConfig` qui utilise `os.ExpandEnv` brut (`profile/project.go:62`), contournant la politique de non-exposition.

### C12 — MOYENNE : la documentation décrit une API imaginaire
`docs/advanced/plugin-hooks.md` et `docs/advanced/yaml-profiles.md` décrivent `~/.multiai/config.yaml` et `~/.multiai/profiles.yaml` avec un schéma `profiles:` map-par-nom et des hooks sous forme de chemins string. Rien de tout cela n'existe : `grep "config.yaml|profiles.yaml"` dans `internal/` et `cmd/` → zéro occurrence ; le schéma réel est un YAML plat par fichier (`yaml.go:15-36`) et les hooks sont des listes d'objets `{command, shell}` (`yaml.go:39-48`). Même le README racine montre un menu à 4 entrées avec « 4. OpenRouter » (`README.md:33-36`) alors que `ShowTopMenu` n'en a que 3 (`menu/interactive.go:21-23`).

### C13 — MOYENNE : ordre du menu non déterministe
`SelectTool` construit la liste des outils en itérant une map Go (`menu/interactive.go:41-64`) — l'ordre d'itération des maps étant aléatoire, la numérotation des outils change d'une exécution à l'autre. Défaut d'architecture de données (map là où il faut une slice ordonnée).

### C14 — MOYENNE : claims de tests gonflées, pans entiers non testés
Comptage réel : 34 fonctions Test/Benchmark (`grep -c "^func Test|^func Benchmark"`) vs « 45+ tests » (`README.md:199`). Zéro test pour : `launcher` (le cœur), `hooks`, `config/wizard` (le flux cassé de C1 aurait été détecté), `menu`, `onboarding`, `openrouter`, `logging`, stores plateforme. Les tests existants couvrent surtout parsing et helpers ; l'emplacement est incohérent (mi `tests/`, mi packages).

### C15 — BASSE : hygiène du dépôt racine
Archives zip qui traînent à la racine (`claude-code-zai-pack.zip`, `code-cli-router-pack.zip`…), fichiers `60/61/62-*.env` orphelins à la racine (doublons des templates PS, ignorés par git), artefact git au nom corrompu commité puis nettoyé (`37b9898` → `70eb802`), binaires + `coverage.out` dans `multiai-go/build/` sur le disque.

### C16 — Extensibilité contributeur : non conçue côté Go
Pour ajouter un fournisseur en Go, un contributeur doit : éditer la fonction hardcodée `DefaultProviders()` (`wizard.go:58`), créer un .env que git ignorera (C4), mettre à jour la regex de validation (`wizard.go:37-43`), les scripts de complétion statiques (`completion.go:18`) et la chaîne du wizard onboarding (`onboarding/wizard.go:40`) — cinq endroits, aucun data-driven. Le PS fait mieux avec son `$ProviderCatalog` déclaratif unique. Il n'existe pas de `CONTRIBUTING.md` dans `multiai-go/` (uniquement dans `multiai-powershell/`). La `ROADMAP.md` de multiai-go vit dans un univers parallèle : elle liste la v0.3.0 « Sécurisation » comme non commencée alors que le CHANGELOG racine déclare une v0.3.0 « providers » livrée.

---

## Statut des problèmes v0.2.1

| # v0.2.1 | Problème | Statut | Preuve |
|---|---|---|---|
| #1 | Injection shell hooks | ⚠️ Corrigé sur le papier, hooks **inatteignables** (code mort) ; l'escaping s'applique à la commande entière, design douteux | `cli/hooks.go:14-37,55-57` ; aucun appelant de `LaunchOptions.Hooks` |
| #2 | Race TOCTOU encryptedFileStore | ✅ Corrigé (`sync.Mutex`) | `secret/secret.go:39,110,122,133,144` |
| #3 | AllowedCommands map mutable | ✅ Corrigé (slice + accesseur) | `cli/launcher.go:18-28` |
| #4 | Checksums placeholders brew/scoop/AUR | ❌ Persiste | `packaging/homebrew/multiai.rb:9-10` (`PLACEHOLDER_ARM64_SHA256`) |
| #5 | Master key en clair `.masterkey` | ❌ Persiste ; PBKDF2 ajouté mais **code mort** (appelé uniquement par les tests) | `secret/secret.go:51-63` ; `crypto.go:18` vs `secret_test.go:50-54` |
| #6 | Pas de signature Cosign | ⚠️ Config présente mais pipeline inexécutable (workflows hors racine + branche main) | `.goreleaser.yml:43-52` ; `multiai-go/.github/workflows/release.yml` ; `ci.yml:5` |
| #7 | Exit code non propagé | ✅ Corrigé | `cli/launcher.go:147-165` ; `main.go:153-155` |
| #8 | Pas de context — enfant orphelin | ⚠️ Partiel : forwarding SIGINT/SIGTERM ajouté, pas de `context.Context` ; `syscall.SIGTERM` inopérant sous Windows | `cli/launcher.go:117-142` |
| #9 | `updateEnvFile` non atomique | ✅ Corrigé (temp+rename)… mais **régression fonctionnelle** : écrit une sentinelle jamais relue (C1) | `config/wizard.go:269,278-287` |
| #10 | Navigation sans retour (Go) | ✅ Corrigé (« 0. Retour » aux deux niveaux) | `menu/interactive.go:73,119` |
| #11 | Couleurs sans texte | ✅ Corrigé (`[OK]/[!]/[X]/[i]` + NO_COLOR) | `cli/display.go:14,61-78` |
| #12 | Incohérence Go vs PowerShell | 🔴 **AGGRAVÉ** : v0.3.0 livrée PS-only ; catalogues 5 vs 14 providers ; 17 vs 38 profils ; sémantique `%VAR%`/`FALLBACK` divergente ; npm distribue le PS en 0.3.0 pendant que le Go « primaire » reste 0.2.1 | C3, C7, C8 |
| #13 | Aucun wizard onboarding | ⚠️ Faux-corrigé : package écrit mais **jamais invoqué** par main | `onboarding/wizard.go:30` ; imports de `main.go:12-16` |
| #15 | `--allow-custom-command` sans validation | ❌ Persiste (simple warning puis exécution) | `cli/launcher.go:57-63` |

Bilan : 5 corrigés réellement, 3 faux-corrigés (code mort), 3 persistants, 1 aggravé.

---

## Recommandations priorisées

### P0 — Restaurer l'intégrité fonctionnelle et documentaire (cette semaine)
1. **Réparer le flux config→launch** : soit résoudre `__MULTIAI_CREDSTORE__` dans `BuildCleanEnv` via `store.Get`, soit (plus simple) réécrire la vraie valeur dans le .env et supprimer la sentinelle. Ajouter un test d'intégration config→launch qui aurait attrapé ce bug.
2. **Tracker les profils et la doc dans git** : remplacer `*.env` par des motifs ancrés + force-add des templates avec placeholders (le mécanisme `prepublishOnly` existe déjà pour la sécurité) ; ancrer `docs/` en `/docs/`.
3. **Purger le README/CHANGELOG de toute feature non branchée** (models/search/compare, YAML, hooks, héritage, credential store « natif », badge 9.5/10, badge Go 1.23, « 45+ tests », CI). Un README honnête vaut mieux qu'un README aspirationnel — chaque écart est une dette de confiance.
4. **Déplacer `.github/` à la racine du repo et cibler `master`** ; corriger le `cd ../../` de release.yml ; passer `const version` en `var` pour que `-X main.version` fonctionne.

### P1 — Trancher la stratégie de convergence Go/PS (ce mois)
5. Décision explicite : soit (a) porter la v0.3.0 en Go (catalogue providers data-driven en YAML embarqué, `FALLBACK`/`REGION` metadata, expansion `%VAR%`, cost log) et geler le PS en maintenance, soit (b) assumer le PS comme produit et requalifier le Go en prototype. L'état actuel — « primaire » en retard sur « legacy » distribuée sous le même nom npm — est la pire des options.
6. **Extraire un contrat de profil formel partagé** (spec du format .env : clés métadonnées, sémantique d'expansion, SKIP_SECRET_CHECK) versionné à la racine, avec tests de conformité des deux implémentations.
7. **Embarquer les profils par défaut via `go:embed`** + répertoire utilisateur `~/.multiai/profiles` prioritaire : règle le clone frais, `go install`, brew, deb, npm d'un coup. Corriger la formule Homebrew (le module est en sous-répertoire) ou déplacer `multiai-go/` à la racine du repo.

### P2 — Assainir la structure (ce trimestre)
8. Brancher ou supprimer le code mort : soit câbler YAML/hooks/onboarding/openrouter dans main (avec `Extends` réellement implémenté), soit les supprimer et les remettre dans la ROADMAP. Le code mort documenté comme feature est le pire des deux mondes.
9. Scinder `internal/cli` : `internal/ui` (Print*, couleurs, tabwriter), `internal/launch` (ValidateAndLaunch), `internal/hooks`. Unifier la whitelist de commandes (une seule définition).
10. Unifier le versioning : une seule source (ldflags via goreleaser), supprimer les versions hardcodées de `menu/interactive.go:18` et `openrouter/client.go:38`, aligner packaging/Makefile/ROADMAP/CHANGELOG.
11. Remplacer la map par une slice ordonnée dans `SelectTool` ; complétion dynamique générée depuis les profils chargés plutôt que shortcuts hardcodés.
12. Couvrir de tests le launcher, le wizard config et le contrat Store (le bug base64 Set/Get asymétrique de `store_windows.go:44` serait tombé au premier round-trip test).

---

## Note : 3.5/10

Justification : squelette de packages sain et dépendances minimales (ce qui vaut les 3.5 points), mais flux central cassé (C1), écart doc/code massif et systématique (C2, C3, C12), chaîne de livraison inopérante de bout en bout (C4, C5, C6, C7), et divergence Go/PS aggravée sans stratégie de convergence — soit l'inverse exact de l'ambition « meilleur routeur multi-IA du marché ». Le score « 9.5/10 » affiché en badge du README est un artefact d'auto-évaluation sans base vérifiable.

---

## Findings contre-vérifiés

Contre-vérification adversariale : chaque finding critical/high a été soumis à une tentative de réfutation indépendante. Aucun finding n'a été réfuté (REFUTED) ; quatre ont vu leur sévérité recalibrée (PARTIAL).

| ID | Sévérité | Titre | Verdict | Note |
|---|---|---|---|---|
| 03-01 | critical | Flux config→launch cassé : sentinelle `__MULTIAI_CREDSTORE__` écrite mais jamais relue (credential store write-only) | CONFIRMED | Vérifié ligne par ligne ; aggravant : si `secret.NewStore()` échoue, la clé saisie est perdue définitivement (sentinelle déjà écrite sur disque). |
| 03-04 | critical | Profils .env et doc VitePress exclus de git : clone frais inutilisable | CONFIRMED | Exit code 2 sur list/launch/config vérifié ; nuance mineure : les profils PS 01-03 sont trackés, 15 restent absents. |
| 03-06 | high | Aucun canal d'installation ne livre les profils (distribution incompatible avec le modèle de données) | CONFIRMED | Vérifié pour install.sh, deb, npm, brew, go install ; le finding est même sous-estimé (repo privé + releases jamais produites). |
| 03-07 | high | Chaos de versions : 5 sources contradictoires + injection ldflags sur une const (no-op silencieux) | CONFIRMED | Toutes les versions vérifiées ; un binaire releasé v0.5.0 afficherait 0.2.1. |
| 03-09 | high | Stores « natifs » = façades, contrat Set/Get asymétrique (base64 non décodé) | CONFIRMED | Bug latent (aucun Get en production), mais données base64 déjà écrites sur disque sous Windows/macOS ; aggravant : `os.Getenv("HOME")` vide sous Windows. |
| 03-03 | high *(corrigée depuis critical)* | v0.3.0 livrée uniquement en PowerShell « legacy », Go « primaire » stagne à 0.2.1 | PARTIAL | Cœur confirmé ; corrections : catalogue PS = 13 providers (pas 14), et `models/search/compare` n'existent dans AUCUNE implémentation (même pas en PS). Pas de faille sécurité → high. |
| 03-05 | high | CI/CD fantôme : workflows hors racine, branche `main` inexistante | PARTIAL | Cœur confirmé (workflows jamais exécutables) ; détail réfuté : release.yml déclenche sur tags `v*`, pas sur `main` — son seul blocage est la localisation hors racine. Sévérité maintenue high. |
| 03-02 | medium *(corrigée depuis critical)* | Code mort d'`internal/` vendu comme features (YAML, héritage, hooks, onboarding, openrouter, logging) | PARTIAL | Mécanique confirmée point par point, mais chiffre recalibré ~31% (pas ~40%) et aucun impact runtime/sécurité : écart doc/produit → medium. |
| 03-08 | medium | Contrat de profil divergent Go vs PS : `%VAR%` non expansé, FALLBACK/REGION exportées | PARTIAL | Leg `%USERPROFILE%` actif (6/17 profils Go cassés) ; les legs FALLBACK/REGION et SKIP_SECRET_CHECK sont latents (aucun profil Go livré ne les contient). Sévérité corrigée medium. |
| 03-10 | medium | PBKDF2 ajouté mais code mort ; master key en clair dans `.masterkey` | non contre-vérifié | — |
| 03-11 | medium | `internal/cli` fourre-tout : présentation couplée à l'exécution | non contre-vérifié | — |
| 03-12 | medium | Duplications structurelles : whitelist, catalogue providers, shortcuts en 3-5 exemplaires | non contre-vérifié | — |
| 03-13 | medium | Doc VitePress décrit une API imaginaire (`~/.multiai/config.yaml`, hooks string) | non contre-vérifié | — |
| 03-14 | medium | Ordre du menu outils non déterministe (itération de map Go) | non contre-vérifié | — |
| 03-15 | medium | Claims de tests gonflées (34 réels vs « 45+ »), zéro test sur launcher/wizard/stores | non contre-vérifié | — |
| 03-16 | medium | `--allow-custom-command` contourne la whitelist sans validation | non contre-vérifié | — |
| 03-17 | low | Hooks inatteignables + escaping appliqué à la commande entière | non contre-vérifié | — |
| 03-18 | low | Hygiène du dépôt : zips, profils orphelins, ROADMAP contredisant le CHANGELOG | non contre-vérifié | — |
