# Audit v0.3.0 — Produit & fonctionnalites

Date 2026-07-04 · Auditeur Atlas (Strategist — business analysis & product management) · Score : 3.5/10 · Methode : audit BMAD+ parallele + contre-verification adversariale.

---

# Audit Produit & Fonctionnalités — multiai v0.3.0 (npm) / v0.2.1 (Go)

**Auditeur** : Atlas (Strategist, BMAD+) — mandaté par Nexus
**Date** : 2026-07-05
**Périmètre** : complétude fonctionnelle réelle vs claims, features v0.3.0, positionnement concurrentiel, gaps produit
**Référence delta** : audit/07-audit-v0.2.1-synthese.md (2026-06-23, 5.5/10)

---

## Résumé

L'ambition affichée — « un seul outil pour lancer Claude Code, Codex CLI et OpenCode avec des profils isolés » — est **partiellement tenue, mais pas par le produit que le README décrit**. La réalité vérifiée dans le code :

1. **La v0.3.0 n'existe pas en Go.** Le commit v0.3.0 (`c476d64`) ne touche QUE `multiai-powershell/` (36 fichiers, 0 fichier Go). `git diff dd8d9c7..HEAD -- multiai-go/internal multiai-go/cmd multiai-go/configs` est vide : aucun changement de code Go depuis la v0.2.1 du 2026-06-23. Le binaire Go affiche `version = "0.2.1"` (multiai-go/cmd/multiai/main.go:18). Le commit `70eb802` documente lui-même la manœuvre : « Update README: 14 providers, 20+ profiles, v0.3.0 features, score 9.5/10 » — le README a été mis à jour sans le code.
2. **Les commandes vedettes annoncées sont du vaporware.** `multiai models`, `multiai search`, `multiai compare` (README.md:88-90, README.md:147-152, CHANGELOG.md:16-18) n'existent nulle part : le switch de main.go:126-182 ne connaît que `version/help/list/launch/config/completion` — ces commandes tombent dans `default:` → « Commande inconnue ». Côté PowerShell, le menu « 4. OpenRouter » (code-router.ps1:880-940) est un écran d'aide statique qui affiche un lien vers openrouter.ai/models et une liste de slugs codée en dur.
3. **Le flux central config→launch du binaire Go est cassé** (voir constat C2) : configurer une clé via `multiai config` rend le profil inutilisable.
4. **La distribution est à 80 % fictive** : sur les 5 méthodes du README (README.md:73-77), seul npm fonctionne — et il livre le PowerShell. `go install` est impossible (module dans un sous-dossier d'un repo privé), brew/scoop/AUR ont des checksums placeholders, aucune release GitHub n'existe (tags : v0.2.1, v0.2.6 uniquement).

Ce qui marche vraiment : le routeur PowerShell v0.3.0 publié sur npm (37 profils, 14 fournisseurs, régions, fallback chains, erase keys) et le lanceur Go v0.2.1 en usage local (menu, launch, list, isolation env, dry-run/JSON).

**Note : 3.5/10** (calibrage sévère ; le badge « score 9.5/10 » du README.md:9 et « 10/10 » de multiai-go/README.md:9 sont des claims auto-décernés sans base).

---

## Forces

| Force | Preuve |
|---|---|
| Coeur de promesse tenu en PS : menu → outil → profil → lancement avec env isolé | code-router.ps1:395 (`Clear-RouterEnvironment`), :429-442 (`Apply-ProfileEnv`), :1130 (`& $command @launchArgs`) |
| **Fallback chains réelles et différenciantes** (relance auto sur un autre profil en cas d'échec) | code-router.ps1:1135-1160, profil 01-code-pro.env (`FALLBACK=cf`) |
| **14 fournisseurs + régions** réels en PS : catalogue groupé Global/Chine/USA | code-router.ps1:93-210 (`$ProviderCatalog` ordered + `Region`), :696-704 (affichage groupé) |
| 37 profils PS réels dont Fusion (60-62), MiniMax/StepFun/Qwen/Kimi/SiliconFlow/MiMo (70-79), Requesty (80-82), LiteLLM (83), alias cp/cf/ceu (01-03) | multiai-powershell/configs/profiles/ (listing complet vérifié) |
| Menu erase keys (par fournisseur ou tout) en PS | code-router.ps1:547-613, :727 |
| Ajout rapide de profil OpenRouter par slug | code-router.ps1:942-982 (`New-OpenRouterProfile`) |
| Lanceur Go v0.2.1 sain : whitelist commandes, propagation exit code, forwarding SIGINT/SIGTERM, navigation « 0. Retour », préfixes [OK]/[X], NO_COLOR, JSON/dry-run | launcher.go:18, :147-165, :117-142 ; interactive.go:73, :119 ; display.go:14, :61-78 |
| Expansion `%VAR%` correcte en PS avec dictionnaire ordonné (les références intra-profil fonctionnent) | code-router.ps1:414-426, :234 (`[ordered]@{}`) |
| npm réellement publié et fonctionnel (multiai@0.3.0, latest, 2026-06-24) | `npm view multiai` + multiai-powershell/package.json |
| Garde-fou anti-fuite `prepublishOnly` étendu aux nouvelles variables (STEPFUN, REQUESTY, DASHSCOPE…) | multiai-powershell/package.json (script prepublishOnly) |

---

## Constats détaillés

### C1 — CRITIQUE : `models` / `search` / `compare` sont du vaporware pur
- **Claims** : README.md:88-90 (« Usage rapide »), README.md:147-152 (« OpenRouter intégré (v0.3.0) : multiai models — top modèles par usage, catégorie, prix ; multiai search — recherche full-text ; multiai compare — comparaison côte à côte ; Cache 1h, fallback offline »), CHANGELOG.md:16-18.
- **Réalité Go** : le switch de main.go:126-182 n'a aucun case `models`/`search`/`compare` → « Commande inconnue ». Le package `internal/openrouter/client.go` (97 lignes : FetchModels, CacheModels, IsCacheValid) n'est **importé par personne** (grep sur tout multiai-go : zéro import hors du package lui-même). Aucune fonction de recherche ni de comparaison n'existe même en code mort.
- **Réalité PS** : `Show-OpenRouterMenu` (code-router.ps1:880-940) affiche un texte statique (« Voir les 300+ modeles : https://openrouter.ai/models ») et 10 slugs codés en dur (:911-921). Aucun appel API, aucun cache, aucune recherche.
- **« Cache 1h, fallback offline »** : code mort en Go (client.go:61-96), inexistant en PS (grep « cache » : une seule occurrence, dans une note descriptive Requesty, :118).

### C2 — CRITIQUE : le flux config→launch du binaire Go casse les clés configurées
- `multiai config` (Go) remplace la valeur dans le .env par le littéral `__MULTIAI_CREDSTORE__` (internal/config/wizard.go:269) et stocke la vraie clé dans le credential store (wizard.go:295, `store.Set`).
- **Aucun code ne relit jamais le store au lancement** : `store.Get` n'apparaît que dans secret_test.go:93,114. `ValidateAndLaunch` (internal/cli/launcher.go:71-76) valide via `dotenv.IsPlaceholder` — qui ne reconnaît PAS `__MULTIAI_CREDSTORE__` (pkg/dotenv/dotenv.go:73-93 : préfixes paste_/your_/xxx…) — puis exporte l'env du profil tel quel (`env.BuildCleanEnv`, env.go:34-60).
- **Conséquence** : le CLI enfant reçoit `ANTHROPIC_AUTH_TOKEN=__MULTIAI_CREDSTORE__` → échec d'authentification garanti. Toute clé configurée via le wizard Go rend le profil inutilisable. C'est le parcours n°1 d'un nouvel utilisateur (config → launch) qui est cassé.

### C3 — CRITIQUE : la v0.3.0 est PowerShell-only, l'architecture « Go primaire / PS legacy » est inversée dans les faits
- README.md:183-192 présente la structure avec `multiai-go/ → Go (primaire)` et `configs/profiles/ → 20+ profils` (:187). Réalité : multiai-go/configs/profiles/ contient **17 profils** (séries 00-57) ; les séries 60-83 (Fusion, MiniMax, StepFun, Qwen, Kimi, SiliconFlow, MiMo, Requesty, LiteLLM) n'existent que dans multiai-powershell/configs/profiles/.
- Les shortcuts du tableau « 20+ profils inclus » (README.md:96-136) `or-fusion`, `mm`, `stepfun`, `mimo`, `codex-fusion`, `codex-qwen`, `codex-siliconflow`, `oc-fusion`, `ocmimo`, `req-cc`, `req-codex`, `req-oc`, `litellm` sont **introuvables dans les profils Go** — `multiai launch -p or-fusion` (exemple README.md:85) échoue avec « profil introuvable » sur le binaire Go.
- Le menu Go (interactive.go:16-36) n'a que 3 entrées et l'option BMAD+ est un stub : main.go:206-208 affiche « (BMAD+ n'est pas encore integre dans la version Go) » — alors que la démo du README (:28-45) montre 4 entrées dont « 4. OpenRouter » et que README.md:174-177 vante « Détection automatique, version, packs, menu mise à jour » (PS uniquement, code-router.ps1:1049-1067).
- Fichiers parasites : `60-claude-fusion.env`, `61-codex-fusion.env`, `62-opencode-fusion.env` traînent à la **racine du repo** (gitignorés, statut `!!`) — vestiges d'une mauvaise manipulation.

### C4 — HAUTE : « Cost logging : estimation coût par requête + cumul session » = faux
- Claims : README.md:163, CHANGELOG.md:14.
- Réalité : `Write-CostLog` (code-router.ps1:1010-1021) écrit `timestamp | shortcut | DisplayName | exit=N | duree=Ns | profile=path` dans costs.log. **Aucun prix, aucune estimation, aucun cumul de session, aucun comptage de tokens.** C'est un log de lancement renommé « cost logging ». Côté Go : rien du tout.

### C5 — HAUTE : 4 canaux d'installation sur 5 sont morts
- README.md:73-77 : npm / Go / Homebrew / Scoop / Script.
- `go install github.com/lrochetta/multiai@latest` (:74) est doublement impossible : le module `github.com/lrochetta/multiai` (go.mod) vit dans le sous-dossier `multiai-go/` (le chemin module ≠ emplacement réel), le package main est sous `cmd/multiai`, et le repo GitHub est **privé** (push-github.ps1:10 : `gh repo create lrochetta/multiai --private`).
- Homebrew : packaging/homebrew/multiai.rb:9-10 = `PLACEHOLDER_ARM64_SHA256` / `PLACEHOLDER_AMD64_SHA256`. Scoop : packaging/scoop/multiai.json:10,14 = `PLACEHOLDER_SHA256`. AUR : packaging/aur/PKGBUILD:11 = `REPLACE_WITH_ACTUAL_SHA256`. Aucune release GitHub n'existe (tags locaux : v0.2.1, v0.2.6 — **pas de tag v0.3.0** malgré la publication npm).
- Le wrapper npm binaire (packaging/npm/package.json, nom `multiai-installer`, version **0.5.0** ⁉) n'est pas publié ; le paquet npm réel `multiai@0.3.0` = PowerShell, qui **exige PowerShell 5.1+/pwsh sur macOS/Linux** (bin/multiai.js:36-38) — friction majeure contredisant la promesse « binaire unique cross-platform ».

### C6 — HAUTE : la CI/CD annoncée ne s'exécute jamais
- README.md:200 : « CI/CD : lint → test (6 OS × Go) → security → build → benchmark ».
- Les workflows vivent dans `multiai-go/.github/workflows/ci.yml` et `release.yml` — **pas à la racine du repo** (`git ls-files | grep workflows` : uniquement sous multiai-go/). GitHub Actions ne lit que `.github/workflows/` à la racine : ces pipelines n'ont jamais tourné et ne tourneront jamais en l'état. Idem pour `.goreleaser.yml` (jamais déclenché, aucune release).

### C7 — HAUTE : YAML, `.multiai.yaml`, héritage et hooks = code mort non câblé
- Claims : README.md:168-171 (« Profils YAML + .multiai.yaml par projet avec héritage ; Plugin hooks before_launch / after_launch »), CHANGELOG.md:116-117.
- `LoadAllProfiles`/`LoadYAML`/`LoadDirYAML` (internal/profile/yaml.go:51-117) ne sont appelées **que par les tests** (tests/integration_test.go:108,126). main.go n'appelle que `profile.LoadDir` (.env) — main.go:134,147,158,187.
- `FindProjectConfig`/`MergeProjectConfig` (internal/profile/project.go:13-73) : **zéro appelant**, même pas les tests.
- Le champ `Extends` (yaml.go:31) n'est **résolu nulle part** : aucune logique d'héritage n'existe, même en code mort. `MergeProjectConfig` utilise d'ailleurs `os.ExpandEnv` brut (project.go:62), contournant le `safeExpandEnv` vanté au CHANGELOG:70.
- Hooks : `LaunchOptions.Hooks` n'est jamais renseigné par main.go (main.go:266-273) → `RunBeforeHooks`/`RunAfterHooks` (cli/hooks.go) ne s'exécutent jamais. Et même s'ils l'étaient, `escapeShellArg` appliqué à la commande entière (hooks.go:55) mangle toute commande légitime contenant quotes ou pipes.

### C8 — HAUTE : le binaire Go est inutilisable hors du dossier de dev
- `getProfilesDir` (main.go:21-37) ne cherche que `<dir exe>/configs/profiles` puis `./configs/profiles`. Pas de répertoire utilisateur (`~/.multiai/profiles`), pas de `go:embed`, pas de variable d'environnement de config. Un binaire installé (go install, brew hypothétique, copie dans PATH) échoue avec « cannot read profiles directory » — exit 2. Le produit Go n'a pas de story d'installation viable.

### C9 — HAUTE : expansion `%VAR%` non supportée en Go — profils partagés incompatibles
- Les profils Go utilisent la syntaxe Windows `%USERPROFILE%` (ex. configs/profiles/00-claude-official.env:12 : `CLAUDE_CONFIG_DIR=%USERPROFILE%\.claude-official`, idem 10/20/21/30/31).
- `safeExpandEnv` (internal/env/env.go:24-31) utilise `os.Expand` (syntaxe `$VAR`/`${VAR}`) : `%USERPROFILE%` est passé **littéralement** au processus enfant, sur toutes les plateformes. L'isolation des répertoires de config Claude (feature clé anti-collision de sessions) est silencieusement cassée en Go, alors que PS l'expanse correctement (code-router.ps1:414-426). Même format de profil, deux sémantiques différentes.

### C10 — MOYENNE : wizard d'onboarding écrit mais jamais branché
- `internal/onboarding/wizard.go` (IsFirstRun, RunWelcome, markFirstRunDone) est complet mais **aucun import** dans main.go ni ailleurs (grep : seul onboarding importe logging). Le CHANGELOG:86 (« internal/onboarding — Wizard premier démarrage ») laisse croire que le problème v0.2.1 #13 est réglé : fonctionnellement, il persiste. Idem `internal/logging/` (logger complet, jamais utilisé par le produit) — et `internal/install/` + `internal/update/` sont des **dossiers vides**.

### C11 — MOYENNE : « Profils dynamiques : ajout/suppression à la volée » — la suppression n'existe pas
- Claim : README.md:172, CHANGELOG.md:20. Réalité : `New-OpenRouterProfile` (code-router.ps1:942-982) crée un profil ; **aucune fonction de suppression de profil** n'existe (grep Remove/Supprimer/delete : seul `Remove-Item -Path "Env:$key"` :410, qui efface des variables d'env). Côté Go : ni ajout ni suppression.

### C12 — MOYENNE : configuration Go limitée à 5 fournisseurs, sans erase
- `DefaultProviders` (internal/config/wizard.go:58-94) : anthropic, zai, deepseek, openai, openrouter — 5 fournisseurs, alors que le pitch produit dit « 14+ fournisseurs » (README.md:19). Le menu config Go (wizard.go:137-141) n'a pas l'option « effacer des clés » (CHANGELOG.md:19) — elle n'existe qu'en PS (code-router.ps1:727, :547-613).

### C13 — MOYENNE : chaos de versions
- Go : `0.2.1` (main.go:18), version en dur dans le menu (interactive.go:18) et le User-Agent (client.go:38). npm publié : `0.3.0`. packaging/npm : `0.5.0`. Tags git : v0.2.1, v0.2.6 — pas de v0.3.0. ROADMAP.md:25-31 décrit une « v0.3.0 — Securisation » (Cosign, SBOM, credential store) sans rapport avec la v0.3.0 du CHANGELOG (fournisseurs) — roadmap jamais mise à jour.

### C14 — BASSE : documentation de référence désynchronisée
- docs/reference/commands.md:44-49 documente `--dry-run | -n` et `--verbose | -v` : ni `-n` ni `--verbose` n'existent (main.go ne parse que les flags longs listés :267-272). Les complétions shell (internal/cli/completion.go:18) codent en dur une liste de shortcuts figée v0.2.x (pas de or-fusion/mm/req-*) au lieu de la générer depuis les profils. README.md:199 : « 45+ tests » — décompte réel : 32 `func Test` + 2 `t.Run` ≈ 34. « Site VitePress : 16 pages » (CHANGELOG.md:119) : 13 fichiers .md. Badge « Go 1.23 » (README.md:7) vs `go 1.22` (go.mod:3). La démo du README (:40-64) ne correspond à aucune des deux implémentations (compte de profils et entrées de menu fabriqués).

---

## Statut des problèmes v0.2.1 (dimension produit)

| # v0.2.1 | Problème | Statut | Preuve |
|---|---|---|---|
| #4 | Checksums placeholders brew/scoop/AUR | **PERSISTE** (et CHANGELOG:76-78 prétend le contraire) | multiai.rb:9-10, multiai.json:10,14, PKGBUILD:11 |
| #6 | Pas de signature Cosign | **PERSISTE de facto** : cosign dans .goreleaser.yml mais aucune release, workflows jamais exécutés (hors racine) | multiai-go/.github/workflows/, tags git |
| #7 | Exit code non propagé | **CORRIGÉ** | launcher.go:147-165, main.go:153-155 |
| #8 | Pas de signal handling | **CORRIGÉ** (forwarding SIGINT/SIGTERM ; toujours pas de context.Context) | launcher.go:117-142 |
| #9 | updateEnvFile non-atomique | **CORRIGÉ** (temp+rename) mais remplacé par une **RÉGRESSION pire** : roundtrip credstore cassé (C2) | wizard.go:278-287 vs :269/:295 |
| #10 | Navigation sans retour (Go) | **CORRIGÉ** | interactive.go:73, :86-88, :119, :132-134 |
| #11 | Couleurs sans texte | **CORRIGÉ** ([OK]/[!]/[X]/[i] + NO_COLOR) | display.go:14, :61-78 |
| #12 | Incohérence Go vs PS | **AGGRAVÉ** : v0.3.0 100 % PS, Go figé 0.2.1, expansion %VAR% divergente (C9), menus différents | git diff dd8d9c7..HEAD (vide sur Go), C3/C9 |
| #13 | Aucun wizard onboarding | **PERSISTE fonctionnellement** (code écrit, jamais appelé) | onboarding/wizard.go sans appelant |
| #15 | --allow-custom-command sans validation | **PERSISTE** (simple bypass avec warning) | launcher.go:57-63 |

---

## Positionnement concurrentiel

| Concurrent | Ce qu'il fait | multiai vs lui |
|---|---|---|
| **claude-code-router** (musistudio) | Proxy de routage par requête pour Claude Code : switch de modèle dynamique, transformers par provider, UI web, routage par type de tâche | multiai ne route pas par requête, seulement par session/processus. Différenciateur multiai : multi-CLI (claude+codex+opencode) et isolation env. Manque bloquant : pas de routage dynamique, pas d'UI |
| **cc-switch** | GUI de bascule de providers pour Claude Code/Codex, gestion centralisée des configs | Périmètre similaire avec GUI mature et communauté ; multiai se distingue par le menu CLI, les fallback chains et la whitelist env — mais sans repo public ni releases, impossible de rivaliser en adoption |
| **LiteLLM proxy** | Passerelle 100+ providers : vrai cost tracking, load balancing, retry/fallback par requête, budgets | multiai est un lanceur, pas un proxy (il a même un profil `litellm`, complémentaire). Mais multiai *prétend* faire du cost logging que seul LiteLLM fait réellement |
| **Profils natifs OpenCode / env managers (direnv, mise)** | OpenCode gère nativement N providers dans opencode.json ; direnv isole l'env par dossier | Pour un utilisateur OpenCode pur, multiai apporte peu ; sa valeur est la cohérence cross-CLI + secrets hors des .env projet |

**Différenciateurs réels et défendables** : (1) isolation d'environnement par liste blanche au lancement — angle sécurité unique ; (2) fallback chains au niveau lanceur (code-router.ps1:1135-1160) — aucun concurrent lanceur ne le fait ; (3) couverture 3 CLIs × 14 providers en un menu ; (4) UX française soignée.

**Manques bloquants pour être « LE meilleur routeur »** : repo privé (zéro communauté, zéro confiance vérifiable), implémentation primaire Go non distribuable et cassée sur son flux config, pas de routage par requête ni de coût réel, dépendance PowerShell sur macOS/Linux pour le seul canal fonctionnel, et un README dont ~40 % des claims fonctionnels sont invérifiables ou faux — rédhibitoire pour un dev exigeant qui vérifie avant d'adopter.

---

## Gaps produit (adoption quotidienne par un dev exigeant)

1. **Confiance** : le parcours « npx multiai install → config → launch » fonctionne en PS, mais tout utilisateur du binaire Go casse ses profils au premier `multiai config` (C2).
2. **Pas de mémoire du dernier profil / profil par défaut** — chaque session repasse par le menu ou exige le shortcut.
3. **Pas de sélection automatique par projet** — le code `.multiai.yaml` existe mais n'est pas branché (C7) ; c'est LA feature qui ferait gagner du temps au quotidien.
4. **Pas de gestion de profils en CLI** (add/remove/edit/doctor) côté Go ; en PS uniquement ajout OpenRouter.
5. **Aucune visibilité coût/usage réelle** malgré le claim (C4) — face à LiteLLM c'est un critère d'achat.
6. **Pas de self-update** (`internal/update/` vide) ni de `multiai doctor` (diagnostic clés/CLIs installés).
7. **Complétions shell figées** (completion.go:18) — l'autocomplétion propose des profils v0.2.x.
8. **Découverte de modèles inexistante** malgré 3 commandes annoncées (C1) — le brainstorm (brainstorm-openrouter.md) décrit la vision, rien n'est construit.

---

## Recommandations priorisées

### P0 — Rétablir l'intégrité (1 semaine)
1. **Purger le README/CHANGELOG de tout claim non implémenté** (models/search/compare, cache 1h, estimation coût, héritage YAML, hooks, badge 9.5/10) ou les marquer « roadmap ». Chaque claim doit pointer vers du code exécutable. Effort : 2 h. C'est le fix au meilleur ROI crédibilité/effort.
2. **Fixer le roundtrip credstore Go** : résoudre `__MULTIAI_CREDSTORE__` via `store.Get` dans `ValidateAndLaunch` (ou cesser d'écrire ce marqueur). Effort : 2 h.
3. **Décider et assumer l'implémentation primaire** : soit porter la v0.3.0 en Go (providers, régions, fallback, profils 60-83), soit officialiser PS comme primaire et requalifier Go en beta. L'état actuel (README Go-first, réalité PS-first) est le mensonge structurel du produit.

### P1 — Rendre le produit installable et vérifiable (2-3 semaines)
4. Déplacer `.github/workflows/` à la racine du repo (ou faire de multiai-go la racine d'un repo dédié) pour que la CI tourne réellement.
5. Passer le repo en public + tagger v0.3.0 + première release goreleaser (résout checksums brew/scoop/AUR, `go install`, et le wrapper npm binaire).
6. Donner au binaire Go un répertoire de profils utilisateur (`~/.config/multiai/profiles` + `go:embed` des templates) — sans ça, aucune distribution Go n'a de sens (C8).
7. Supporter `%VAR%` (ou migrer les profils vers `${VAR}`) dans `safeExpandEnv` pour unifier la sémantique Go/PS (C9).

### P2 — Combler l'écart concurrentiel (1-2 mois)
8. Câbler ce qui existe déjà : `.multiai.yaml` par projet + hooks + onboarding (le code est écrit, il manque ~50 lignes dans main.go).
9. Implémenter réellement `multiai models`/`search` sur l'API OpenRouter (le client existe, client.go:35-58) + complétions dynamiques.
10. Cost logging honnête : croiser durée/modèle avec les prix OpenRouter (déjà dans ModelPricing, client.go:24-27) pour une vraie estimation, ou renommer la feature « launch log ».
11. Profil par défaut + `multiai doctor` + suppression de profils dynamiques.

---

## Findings contre-verifies

| ID | Severite | Titre | Verdict | Note |
|---|---|---|---|---|
| 01-02 | critical | Flux config→launch cassé en Go : le littéral `__MULTIAI_CREDSTORE__` est exporté comme clé API | CONFIRMED | Chaîne complète vérifiée (wizard.go:269/:295, aucun `store.Get` en production, IsPlaceholder aveugle au marqueur) ; aggravant : échec dès la même session (rechargement disque main.go:187) |
| 01-03 | high (corrigée, initialement critical) | v0.3.0 inexistante en Go : toutes les features livrées en PowerShell « legacy » uniquement | PARTIAL | Faits intégralement confirmés (diff Go vide, 17 profils, or-fusion introuvable) mais sévérité recalibrée : le canal npm headline livre la v0.3.0 PS fonctionnelle ; impact limité aux canaux Go déjà cassés |
| 01-04 | high | « Cost logging : estimation coût + cumul session » — aucune estimation de coût n'existe | CONFIRMED | Write-CostLog = simple log de lancement (timestamp/exit/durée) ; ModelPricing Go jamais utilisé |
| 01-05 | high | 4 canaux d'installation sur 5 morts : go install impossible, brew/scoop/AUR placeholders, repo privé, aucune release | CONFIRMED | Vérifié sur code + état live GitHub/npm ; même sous-estimé (manifests référencent un tag v0.5.0 fantôme, tap Homebrew inexistant) |
| 01-06 | high | CI/CD annoncée jamais exécutée : workflows hors de la racine du repo | CONFIRMED | Workflows uniquement sous multiai-go/.github/workflows/, aucun .github/ racine — jamais exécutables par GitHub Actions |
| 01-07 | high | Profils YAML, .multiai.yaml, héritage `extends` et hooks before/after_launch : code mort non câblé | CONFIRMED | ~200 lignes de code mort ; zéro appelant production de LoadAllProfiles/FindProjectConfig/Hooks ; `Extends` jamais résolu |
| 01-08 | high | Expansion `%VAR%` non supportée en Go : CLAUDE_CONFIG_DIR passé littéralement, isolation des configs cassée | CONFIRMED | os.Expand ($VAR only) ; les 6 profils Claude Go utilisent %USERPROFILE% → transmis littéralement, isolation silencieusement inopérante |
| 01-09 | high | Binaire Go inutilisable hors du repo : aucune stratégie de répertoire de profils utilisateur | CONFIRMED | getProfilesDir ne teste que 2 emplacements ; aucun canal de distribution ne livre les profils → exit 2 sur toutes les commandes |
| 01-01 | medium (corrigée, initialement critical) | Commandes `models`/`search`/`compare` annoncées mais inexistantes (vaporware) | PARTIAL | Faits confirmés (switch sans ces cases, client OpenRouter jamais importé, menu PS statique) mais écart doc/implémentation sans impact sécurité ni perte de données : l'utilisateur reçoit une erreur propre |
| 01-10 | medium | Wizard onboarding écrit mais jamais appelé ; internal/install et internal/update vides | non contre-verifie | — |
| 01-11 | medium | « Profils dynamiques : ajout/suppression à la volée » — la suppression n'existe nulle part | non contre-verifie | — |
| 01-12 | medium | Config Go limitée à 5 fournisseurs et sans menu erase keys, contre 14+ annoncés | non contre-verifie | — |
| 01-13 | medium | Chaos de versions : 0.2.1 (Go) vs 0.3.0 (npm) vs 0.5.0 (packaging/npm) vs roadmap contradictoire | non contre-verifie | — |
| 01-14 | medium | Menu Go : BMAD+ stub et pas d'entrée OpenRouter, contrairement à la démo du README | non contre-verifie | — |
| 01-15 | low | Docs de référence et complétions désynchronisées ; métriques gonflées (tests, pages, Go 1.23) | non contre-verifie | — |
| 01-16 | low | Fichiers profils fusion orphelins à la racine du repo | non contre-verifie | — |

Aucun finding n'a reçu de verdict REFUTED : aucun n'est écarté.
