# Audit v0.3.0 — Qualite de code

Date 2026-07-04 · Auditeur Forge (Architect-Dev — qualite de code) · Score : 4.5/10 · Methode : audit BMAD+ parallele + contre-verification adversariale.

---

# Audit Qualité de Code — multiai (Go primaire, PowerShell legacy)

**Auditeur** : Forge (BMAD+ Architect-Dev) — mandaté par Nexus
**Date** : 2026-07-05
**Périmètre** : `multiai-go/` (24 fichiers Go, 3 072 lignes) + `multiai-powershell/code-router.ps1` (1 165 lignes)
**Outillage** : `go build ./...` OK, `go vet ./...` OK, `gofmt -l` → **3 fichiers non conformes** (go1.26.4 windows/amd64)
**Référence delta** : `audit/07-audit-v0.2.1-synthese.md` (2026-06-23, code 5.5/10)

---

## Résumé

Le binaire Go compile et passe `go vet`, la base est petite et lisible, et une partie des correctifs annoncés dans le CHANGELOG v0.2.6 est réelle. Mais trois bugs **critiques de correction fonctionnelle** rendent les flux primaires inutilisables : (1) le flux `multiai config` → `multiai launch` est cassé de bout en bout — la clé saisie est remplacée dans le `.env` par un sentinel `__MULTIAI_CREDSTORE__` que **rien ne résout jamais** au lancement ; (2) les profils `.env` livrés avec le Go utilisent la syntaxe `%USERPROFILE%` que `os.Expand` n'expanse jamais ; (3) la whitelist d'environnement est case-sensitive alors que Windows stocke `Path`, `SystemRoot`, `ComSpec`, `windir` en casse mixte — vérifié empiriquement sur la machine d'audit — donc le processus enfant démarre **sans PATH** sous Windows. S'y ajoutent une CI/CD totalement inerte (workflows dans `multiai-go/.github/`, un emplacement que GitHub ignore, avec trigger sur `main` alors que la branche est `master`), ~1/3 de code mort (hooks, YAML, onboarding, openrouter, logging) et un écart massif entre le CHANGELOG v0.3.0 et le code réel. **Note : 4.5/10** (calibrage sévère ; le badge « 9.5/10 » du README est un claim marketing sans rapport avec la réalité).

---

## Forces

- **Build sain** : `go build ./...` et `go vet ./...` passent sans erreur ; une seule dépendance externe (`gopkg.in/yaml.v3`, `go.mod:5`).
- **Correctifs v0.2.1 réels** : mutex sur `encryptedFileStore` (`internal/secret/secret.go:39,110,123,134,145`), propagation du code de sortie enfant (`internal/cli/launcher.go:147-165` + `cmd/multiai/main.go:153-155`), écriture atomique temp+rename (`internal/config/wizard.go:278-287`), navigation « 0. Retour » (`internal/menu/interactive.go:73,119`), préfixes textuels `[OK]/[!]/[X]/[i]` + `NO_COLOR` (`internal/cli/display.go:14,61-78`).
- **Crypto correcte** : AES-256-GCM avec nonce aléatoire préfixé (`internal/secret/crypto.go:56-94`) ; l'implémentation PBKDF2 maison est conforme au RFC (`crypto.go:25-53`).
- **Parser dotenv simple et testé** (`pkg/dotenv/dotenv.go:20-65`) : export, guillemets, commentaires.
- **PowerShell legacy solide sur certains points** : whitelist env case-insensitive via `-in` (`code-router.ps1:395-412`), expansion `%VAR%` fonctionnelle (`code-router.ps1:414-427`), `exit $exitCode` final (`code-router.ps1:1165`), fallback chains réellement implémentées (`code-router.ps1:1136-1163`).

---

## Constats détaillés

### A. Bugs critiques de correction fonctionnelle (Go)

**A1. Flux config→launch cassé : le sentinel `__MULTIAI_CREDSTORE__` n'est jamais résolu — CRITIQUE**
`updateEnvFile` remplace la valeur dans le `.env` par `varName + "=__MULTIAI_CREDSTORE__"` (`internal/config/wizard.go:269`) et pousse la vraie clé dans le credential store (`wizard.go:295`). Mais un grep exhaustif montre que `__MULTIAI_CREDSTORE__` n'apparaît que deux fois dans tout le dépôt : à l'écriture (`wizard.go:269`) et dans un test d'onboarding mort (`internal/onboarding/wizard.go:21`). Ni `validateSecrets` (`internal/cli/launcher.go:217-229`), ni `BuildCleanEnv` (`internal/env/env.go:34-60`), ni `LoadDir` ne relisent le store. Conséquences : après `multiai config`, le CLI enfant reçoit littéralement `ANTHROPIC_AUTH_TOKEN=__MULTIAI_CREDSTORE__` (et `IsPlaceholder` ne le détecte pas, `pkg/dotenv/dotenv.go:73-93`, donc aucune erreur). Pire : si `secret.NewStore()` échoue, `updateEnvFile` retourne `nil` (succès) **sans stocker la clé nulle part** (`wizard.go:290-293`) et `_ = store.Set(...)` avale aussi l'erreur (`wizard.go:295`) — la clé que l'utilisateur vient de taper est perdue silencieusement. La fonctionnalité centrale du produit est donc cassée de bout en bout en Go.

**A2. `%USERPROFILE%` jamais expansé — les profils livrés sont incompatibles avec le binaire Go — CRITIQUE**
Les profils livrés dans `multiai-go/configs/profiles/` utilisent la syntaxe Windows : `CLAUDE_CONFIG_DIR=%USERPROFILE%\.claude-deepseek-v4pro` (`configs/profiles/30-claude-deepseek-v4-pro.env:12`) et le pattern référence `%OPENROUTER_API_KEY%`. Or `safeExpandEnv` s'appuie sur `os.Expand` qui ne connaît que `$VAR`/`${VAR}` (`internal/env/env.go:24-31`). Le processus enfant reçoit donc la chaîne littérale `%USERPROFILE%\...`. Le PowerShell, lui, gère `%VAR%` correctement (`code-router.ps1:414-427`). De plus, même en syntaxe `$VAR`, les références inter-variables du profil ne marcheraient pas : `safeExpandEnv` n'expanse que la whitelist système, pas les autres variables du profil.

**A3. Whitelist d'environnement case-sensitive → enfant sans PATH sous Windows — CRITIQUE**
`BuildCleanEnv` filtre avec `AllowedEnvVars[key]` (lookup map case-sensitive, `internal/env/env.go:44`) alors que la whitelist est en MAJUSCULES (`env.go:9-21`). Vérification empirique sur la machine d'audit : le process env contient `Path`, `ComSpec`, `SystemRoot`, `windir` (casse mixte) → **aucun ne matche** `PATH`/`COMSPEC`/`SYSTEMROOT`/`WINDIR`. Le CLI enfant (shim `claude.cmd` → node) démarre sans `PATH` ni `SystemRoot` : résolution de `node` et spawn de sous-processus cassés. La whitelist omet aussi `APPDATA`, `LOCALAPPDATA`, `ProgramFiles`, `NUMBER_OF_PROCESSORS` que le PS inclut partiellement (`code-router.ps1:397-406`, avec comparaison `-in` insensible à la casse — le Go a régressé par rapport au PS).

### B. Infrastructure et claims

**B1. CI/CD entièrement inerte — CRITIQUE (infra)**
Les workflows vivent dans `multiai-go/.github/workflows/` (`ci.yml`, `release.yml`) alors que la racine du dépôt git est `D:/travail/DEV/multiai/` (remote `github.com/lrochetta/multiai.git`) — GitHub ne lit les workflows **qu'à la racine** `.github/workflows/`. Double verrou : les triggers visent `branches: [main]` (`ci.yml:5-7`) alors que la branche est `master`. Donc lint, tests 6×, gosec, govulncheck, benchmark, goreleaser+Cosign : **rien ne s'exécute jamais**. La « force » CI/CD saluée par l'audit v0.2.1 est fictive. Preuve indirecte : `gofmt -l` échoue sur 3 fichiers, ce que la moindre exécution de golangci-lint aurait attrapé.

**B2. `go install github.com/lrochetta/multiai@latest` ne peut pas fonctionner — HAUTE**
Le module est déclaré `module github.com/lrochetta/multiai` (`multiai-go/go.mod:1`) mais vit dans le sous-dossier `multiai-go/` du dépôt. Le chemin d'import ne correspond pas à l'emplacement ; il faudrait `github.com/lrochetta/multiai/multiai-go`. Les instructions d'installation du README racine (`README.md:69-75` : go install, brew, scoop, curl) sont inopérantes — et le dépôt est privé (`push-github.ps1:10`).

**B3. CHANGELOG v0.3.0 : fonctionnalités annoncées inexistantes — HAUTE**
`CHANGELOG.md:16-18` annonce « `multiai models` : découverte dynamique (300+) », « `multiai search` », « `multiai compare` ». Le switch de `main.go:126-182` ne connaît que `version/help/list/launch/config/completion`. Le PS n'a pas non plus ces commandes : `Show-OpenRouterMenu` est une liste statique de 10 slugs hardcodés (`code-router.ps1:911-921`), aucun `Invoke-RestMethod` dans tout le script. « Cache OpenRouter 1h » (`CHANGELOG.md:22`) : les fonctions de cache existent (`internal/openrouter/client.go:61-96`) mais **rien ne les appelle**. « Cost logging : estimation coût par requête + cumul session » (`CHANGELOG.md:14`) : `Write-CostLog` ne logge ni coût ni cumul — juste shortcut/exit/durée (`code-router.ps1:1010-1021`). « Menu erase keys » : PS uniquement (`code-router.ps1:561-622`), absent du wizard Go (`internal/config/wizard.go:107-171`). Le badge « score 9.5/10 » (`README.md:9`) et « 10/10 » (`multiai-go/README.md:9`) contredisent l'auto-audit à 5.5/10.

**B4. Chaos de versions : 5 sources divergentes + injection ldflags inopérante — MOYENNE**
`const version = "0.2.1"` (`cmd/multiai/main.go:18`), « v0.2.1 » hardcodé dans le menu (`internal/menu/interactive.go:18`), User-Agent « multiai/0.2.1 » (`internal/openrouter/client.go:38`), `VERSION = 0.2.0-dev` (`Makefile:2`), `"version": "0.3.0"` (`multiai-powershell/package.json`). De plus, `-X main.version=...` (`Makefile:4`, `.goreleaser.yml:15`) cible une **const** — l'injection à l'édition de liens ne fonctionne que sur une `var` : elle est silencieusement sans effet.

### C. Code mort massif (Go)

**C1. Hooks : parsés puis jetés — HAUTE**
`yamlToProfile` ne copie jamais `py.Hooks` dans `Profile` (`internal/profile/yaml.go:130-171` — le struct `Profile` n'a même pas de champ Hooks) et `opts.Hooks` n'est jamais assigné par aucun appelant (`cmd/multiai/main.go:266-273`). Tout `internal/cli/hooks.go` (164 lignes) est mort. Le README v0.2.0 revendique « Plugin hooks before_launch/after_launch » (`CHANGELOG.md:117`).

**C2. Profils YAML et config projet : jamais branchés — HAUTE**
Toutes les commandes utilisent `profile.LoadDir` (.env uniquement — `main.go:134,147,158,187`). `LoadAllProfiles`, `LoadDirYAML`, `LoadYAML` ne sont appelés que par les tests (`tests/integration_test.go:126`). `FindProjectConfig`/`MergeProjectConfig`/`ValidateProfileYAML` (`internal/profile/project.go:13,51,76`) : zéro appelant. Le claim « Profils YAML + .multiai.yaml par projet avec héritage » (`CHANGELOG.md:117`) est faux pour le binaire réel. Notons que `MergeProjectConfig` utilise `os.ExpandEnv` sans restriction (`project.go:62`), incohérent avec `safeExpandEnv`.

**C3. Onboarding, logging, openrouter : trois packages morts — MOYENNE**
`onboarding.IsFirstRun`/`RunWelcome` : aucun appelant (le wizard premier démarrage « corrigé » en v0.2.6 n'est jamais invoqué) ; le marqueur `.first-run-done` est écrit (`internal/onboarding/wizard.go:68-73`) mais jamais lu. `logging` n'est importé que par onboarding (donc mort), son `init()` crée pourtant `~/.multiai/logs` à chaque exécution (`internal/logging/logger.go:35-40`) et ce n'est pas un « logger structuré » (lignes texte). `openrouter` : aucun appelant.

**C4. `ClearEnv` parsé mais jamais consulté — MOYENNE**
`CLEAR_ENV` est parsé (`internal/profile/profile.go:112-116`, `yaml.go:162-164`) mais `ValidateAndLaunch` appelle `env.BuildCleanEnv` inconditionnellement (`launcher.go:76`). `CLEAR_ENV=false` est silencieusement ignoré en Go ; le PS l'honore (`code-router.ps1:432-436`). Divergence de comportement Go/PS sur un même fichier profil.

**C5. `jsonError` défini, jamais utilisé** (`internal/cli/launcher.go:212-215`) — et les erreurs en mode `--json` sortent en texte brut, cassant le contrat JSON.

### D. Concurrence et signaux (Go)

**D1. `defer close(sigCh)` avant `signal.Stop` → panic potentielle — HAUTE**
`launcher.go:120-121` : `defer signal.Stop(sigCh)` puis `defer close(sigCh)`. LIFO : `close` s'exécute **avant** `Stop`. Entre les deux, un SIGINT entrant provoque un send sur canal fermé dans la goroutine du package os/signal → panic. Fenêtre courte mais réelle, précisément au moment où l'utilisateur martèle Ctrl+C. Ordre correct : `Stop` puis `close` (ou ne pas fermer).

**D2. Forwarding de signaux inefficace/nuisible — MOYENNE**
Sous Windows, `cmd.Process.Signal(syscall.SIGTERM/SIGINT)` n'est pas supporté (no-op avec erreur ignorée, `launcher.go:136-142`). Sous Unix, l'enfant partage le groupe de processus du terminal : il reçoit déjà SIGINT nativement, puis le parent le lui **renvoie** → double SIGINT (pour Claude Code : 1er = interrompre, 2e = quitter). Aucun `context.Context` dans tout le projet (un seul `go func`, grep vérifié).

**D3. Mutex neutralisé par instanciation par opération — MOYENNE**
Les wrappers Windows/macOS créent un `newEncryptedFileStore()` **à chaque appel** (`store_windows.go:26,32,40,48,56` ; `store_darwin.go:19,27,35,43`) : le `sync.Mutex` (fix v0.2.1 #2) ne protège plus rien entre deux opérations, et il n'y a aucun verrou inter-processus (pas de lockfile).

### E. Credential store : stubs et bug d'encodage

**E1. « Credential store natif » fictif + round-trip cassé — HAUTE**
`store_windows.go` prétend utiliser Windows Credential Manager mais `Get` lance `cmdkey /list` puis retombe sur le fichier chiffré **dans les deux branches** (`store_windows.go:22-36` — le if/else est identique, pur théâtre). Pire : `Set` encode la valeur en **base64** (`store_windows.go:44`, `store_darwin.go:31`) mais `Get` ne décode jamais → `Set("sk-abc")` puis `Get` retourne `c2stYWJj`. Round-trip cassé sur Windows/macOS (Linux est cohérent, `store_linux.go:23-25`) — preuve que ce chemin n'a jamais été exercé. L'item v0.2.1 « remplacer stubs par vraies implémentations » (plan #7 haute priorité) n'est pas fait ; les README continuent de revendiquer le store natif (`multiai-go/README.md:73`).

**E2. `HOME` au lieu de `os.UserHomeDir` — MOYENNE**
`newEncryptedFileStore` utilise `os.Getenv("HOME")` (`internal/secret/secret.go:45`) : vide sous Windows PowerShell classique → chemin relatif `\.config\multiai\secrets` sur la racine du lecteur courant. Incohérent avec `openrouter/client.go:62` et `logging/logger.go:36` qui utilisent correctement `os.UserHomeDir()`.

### F. Comportement CLI (Go)

**F1. Les échecs de lancement sortent avec exit 0 — HAUTE**
`runLaunch` retourne `nil` sur toute erreur (profil introuvable `main.go:227-229`, commande non autorisée, secret manquant `main.go:276-279`) et `main` ne sort non-zéro que si `result != nil && result.ExitCode != 0` (`main.go:153-155`). Donc `multiai launch -p inexistant && echo OK` affiche OK. Le fix v0.2.1 #7 ne couvre que le code de l'enfant, pas les échecs du routeur. Le PS documente et implémente des codes 0-4 (`code-router.ps1:1024-1029`) — divergence de contrat.

**F2. Binaire installé inutilisable : résolution des profils trop fragile — HAUTE**
`getProfilesDir` ne cherche que `<dir exe>/configs/profiles` puis `./configs/profiles` (`main.go:21-37`). Un binaire installé via go install/brew/scoop ne trouvera jamais de profils, et il n'existe aucun fallback utilisateur (`~/.config/multiai/profiles` ou `~/.multiai/profiles`). Contradiction directe avec les 5 méthodes d'installation du README.

**F3. JSON fabriqué à la main sans échappement → invalide — MOYENNE**
`ShowEffectiveEnv` construit le JSON par `fmt.Printf` (`launcher.go:176-193`). Toute valeur contenant `\` ou `"` casse le JSON — or les profils livrés contiennent `%USERPROFILE%\.claude-...` : `multiai launch -p ds --show-env --json | jq` échoue systématiquement. `encoding/json` est pourtant utilisé trois lignes plus haut dans le même package (`display.go:46`). Ordre des clés non déterministe en prime (itération de map, `launcher.go:179-182`).

**F4. Menu outils non déterministe — MOYENNE**
`SelectTool` construit la liste en itérant une map (`interactive.go:58-64`) : l'ordre « 1. Claude Code / 2. Codex » peut changer **entre deux exécutions**. Le PS trie (`code-router.ps1:313`).

**F5. Messages d'erreur avec la syntaxe PowerShell dans le binaire Go — MOYENNE**
« Utilise -AllowCustomCommand » (`launcher.go:61`) et « Lance 'multiai -List' » (`profile.go:163`) : ces flags n'existent pas en Go (`--allow-custom-command`, `multiai list`). Copier-collé du PS non adapté.

**F6. Divers** : `list --json` émet `null` au lieu de `[]` pour zéro profil (`display.go:35-45`) ; complétions shell avec 18 shortcuts hardcodés qui divergent déjà des profils réels (`completion.go:18,64`) ; `--help` de sous-commande seulement détecté en position exacte `os.Args[2]` (`main.go:112`) ; parsing d'arguments artisanal `hasFlag`/`getFlagValue` sans validation des flags inconnus (`main.go:292-323`) ; menu option 3 = stub « BMAD+ n'est pas encore integre » (`main.go:206-208`) alors que le README racine montre 4 options.

### G. Sécurité du code des hooks (recoupement dimension sécurité)

**G1. L'« échappement » des hooks est mal ordonné : réintroduit l'injection et casse les hooks légitimes — HAUTE**
`hooks.go:55-57` : `escapeShellArg` est appliqué à la **commande entière** (cassant pipes, quotes et `$VAR` légitimes de l'auteur du hook), puis `os.ExpandEnv` est appelé **après** l'échappement. Le commentaire prétend que c'est pour empêcher l'injection — c'est l'inverse : les valeurs substituées par `ExpandEnv` arrivent brutes dans la chaîne exécutée par `sh -c`/`cmd /c`. Exemple : env var contenant `x; rm -rf /` → `\x; rm -rf /` → le `;` non échappé sépare les commandes. Sous Windows, `$` n'est pas échappé par la branche cmd.exe (`hooks.go:30-35`) donc la valeur passe telle quelle. Le fix v0.2.1 #1 est donc incorrect (heureusement neutralisé par le fait que les hooks sont du code mort, cf. C1). `RunBeforeHooks`/`RunAfterHooks` sont par ailleurs dupliqués à ~95 % (`hooks.go:41-89` vs `92-137`).

### H. Robustesse parsing et divers

- `LoadDir` ignore silencieusement les `.env` illisibles ou corrompus (`profile.go:56,61`) ; idem `LoadDirYAML` (`yaml.go:91-93`) — l'utilisateur ne sait pas qu'un profil manque.
- `LoadYAML` lit le fichier **entier** avant le check 1 Mo (`yaml.go:56-64`) : la « YAML bomb protection » n'empêche pas l'épuisement mémoire d'un fichier géant ; utiliser `os.Stat` ou `io.LimitReader` d'abord.
- `dotenv.Parse` : pas de commentaires inline (`KEY=val # comment` → le commentaire entre dans la valeur), limite de 64 Ko/ligne du `bufio.Scanner` non gérée.
- `openrouter.FetchModels` : erreur de `http.NewRequest` ignorée (`client.go:36`) ; `CacheModels` avale les erreurs de `MkdirAll`/`Marshal` et écrit sans temp+rename (`client.go:61-68`).
- Duplication : `isAllowedCommand` (`project.go:101-104`) duplique `cli.AllowedCommands` (`launcher.go:18`) ; tri des profils dupliqué (`profile.go:140-148` vs `yaml.go:120-127`) ; `MaskSecret` dupliqué dans wizard (`wizard.go:204-209` vs `env.go:74-82`).
- gofmt non conforme : `internal/config/wizard.go`, `internal/openrouter/client.go`, `internal/profile/yaml.go` (ifs mono-ligne `yaml.go:123-125`).
- Encodage console : mélange incohérent de chaînes ASCII-strippées (« termine », « cles ») et de caractères non-ASCII (`─` `interactive.go:19`, `wizard.go:110` ; `é` `interactive.go:22` ; `⚠` `launcher.go:59`) sans aucune gestion du codepage Windows (pas de SetConsoleOutputCP) : sous conhost CP850, affichage garbled ; le strip des accents est donc incomplet, pas une politique.

### I. PowerShell legacy (v0.3.0 de fait)

- `Set-ProfileSecret` non atomique (`Get-Content`+`Set-Content` direct, `code-router.ps1:527-542`) — le fix atomique n'a été porté qu'en Go (**v0.2.1 #9 persiste en PS**), et le PS écrit la clé **en clair** dans le `.env` (pas de credential store), comportement divergent du Go.
- `Get-RouterHash` : `$MyInvocation.MyCommand.Path` **dans une fonction** ne pointe pas le script → toujours `$null` → le « hash d'intégrité SHA256 » revendiqué par le README n'est jamais loggé (`code-router.ps1:445-453`, consommé `1087-1090`). Utiliser `$PSCommandPath`.
- Claim « SecureString : clés jamais en clair en mémoire » (`multiai-powershell/README.md:74`) : **aucun** usage de SecureString dans le code (grep vérifié) ; `Read-Host` en clair (`code-router.ps1:662`).
- Fallback chain déclenchée sur **tout** exit ≠ 0 (`code-router.ps1:1136`), y compris une interruption volontaire (Ctrl+C, exit 130) → relance surprise sur un autre fournisseur (et une autre facturation).
- `throw 'Choix invalide.'` dans `Select-Tool`/`Select-Profile` (`code-router.ps1:327,353`) + `trap` global → une faute de frappe au menu tue le programme (exit 1) au lieu de redemander.
- Fonctionnalités v0.3.0 (14 fournisseurs, régions, fallback, profils 60-83) **uniquement en PS** : le Go plafonne à 5 fournisseurs (`internal/config/wizard.go:58-94`) et 17 profils (00-57). Le « primaire Go » du positionnement produit a 2 versions mineures de retard fonctionnel sur le « legacy ».

---

## Statut des problèmes v0.2.1

| # v0.2.1 | Problème | Statut | Preuve |
|---|---|---|---|
| **#2** | Race TOCTOU `encryptedFileStore.Set/Delete` | 🟡 **Partiellement corrigé** | `sync.Mutex` ajouté et pris dans Get/Set/Delete/List (`internal/secret/secret.go:39,110,123,134,145`) — MAIS les wrappers Windows/macOS créent un store neuf **par opération** (`store_windows.go:26,32,40,48,56`), rendant le mutex inopérant entre opérations ; aucun verrou inter-processus. |
| **#3** | `AllowedCommands` map mutable sans sync | 🟢 **Corrigé** | Map → slice + `IsCommandAllowed` (`internal/cli/launcher.go:17-28`). Reliquats mineurs : `var` exportée donc toujours mutable ; logique dupliquée dans `project.go:101-104`. |
| **#7** | Exit code du fils non propagé | 🟡 **Partiellement corrigé** | Code enfant propagé (`launcher.go:147-165`, `main.go:153-155`). MAIS toute erreur du routeur lui-même (profil introuvable, secret manquant, commande interdite) sort avec **exit 0** (`main.go:215-288` : `runLaunch` retourne `nil`, main ne teste que `result.ExitCode`). |
| **#8** | Pas de context.Context / SIGINT orphelin | 🟡 **Partiellement corrigé** | Forwarding SIGINT/SIGTERM ajouté (`launcher.go:117-142`). MAIS : ordre des defer inversé → `close(sigCh)` avant `signal.Stop` = panic « send on closed channel » possible (`launcher.go:120-121`) ; `Process.Signal` est un no-op sous Windows ; double-SIGINT sous Unix (l'enfant le reçoit déjà du terminal) ; toujours **zéro** `context.Context` dans le projet. |
| **#9** | `updateEnvFile` non atomique | 🟡 **Corrigé en Go, mais remplacé par pire ; persiste en PS** | Temp+rename OK (`internal/config/wizard.go:278-287`). MAIS le nouveau flux écrit un sentinel `__MULTIAI_CREDSTORE__` jamais résolu au lancement et perd la clé si le store échoue (`wizard.go:269,290-296`) — la « correction » a cassé la fonctionnalité. En PS, `Set-ProfileSecret` reste non atomique (`code-router.ps1:527-542`). |
| #1 (séc.) | Injection shell hooks | 🔴 **Mal corrigé + code mort** | `escapeShellArg` sur la commande entière puis `os.ExpandEnv` **après** échappement réinjecte des métacaractères non échappés (`hooks.go:55-57`) ; de toute façon les hooks ne sont jamais exécutés (yaml.go:130-171 jette `py.Hooks`). |

---

## Recommandations priorisées

### P0 — Cette semaine (le produit Go ne fonctionne pas)
1. **Réparer config→launch** : soit résoudre `__MULTIAI_CREDSTORE__` au chargement (`LoadDir` : si valeur == sentinel → `store.Get("multiai:"+profileID, varName)`), soit écrire la clé en clair dans le `.env` comme le PS. Ne jamais retourner `nil` quand la clé n'a été persistée nulle part (`wizard.go:290-296`).
2. **Expansion `%VAR%`** : ajouter le support `%VAR%` dans `safeExpandEnv` (regex comme `Expand-RouterValue` PS) + résolution des références intra-profil, ou convertir les profils Go en syntaxe `${VAR}` — mais il faut de toute façon gérer les fichiers partagés avec PS.
3. **Whitelist env case-insensitive sous Windows** : normaliser avec `strings.ToUpper(key)` avant lookup (sur Windows uniquement, ou globalement pour la whitelist) et ajouter `APPDATA`, `LOCALAPPDATA`, `ProgramFiles`, `HOMEDRIVE/HOMEPATH`, `NUMBER_OF_PROCESSORS`.
4. **Réactiver la CI** : déplacer `.github/` à la racine du dépôt (avec `working-directory: multiai-go`), trigger sur `master`, puis corriger les 3 fichiers gofmt.
5. **Inverser les defer** `signal.Stop`/`close` (`launcher.go:120-121`).

### P1 — Ce mois
6. Exit non-zéro sur échec de lancement (retourner une erreur depuis `runLaunch`, `os.Exit(1)` dans main) + sortie JSON d'erreur via `jsonError` en mode `--json`.
7. Brancher ou supprimer le code mort : hooks (copier `py.Hooks` dans `Profile`, câbler `opts.Hooks`), YAML/`.multiai.yaml` (utiliser `LoadAllProfiles` + `FindProjectConfig` dans main), onboarding (appeler `IsFirstRun` au démarrage), openrouter (implémenter `multiai models` annoncé) — ou retirer ces claims du README/CHANGELOG.
8. Corriger le round-trip base64 des stores win/darwin (décoder dans `Get`) et instancier le store une seule fois ; remplacer `os.Getenv("HOME")` par `os.UserHomeDir()` (`secret.go:45`).
9. Refondre l'échappement des hooks : n'échapper que les **valeurs** substituées (templates + env), jamais la commande ; supprimer `os.ExpandEnv` post-échappement ; fusionner RunBefore/RunAfter (un seul `runHooks(hooks []HookCommand, blocking bool)`).
10. Source de version unique : `var version` + `-X` (déjà dans Makefile/goreleaser), utilisée par le menu et le User-Agent.
11. Fallback profils utilisateur (`~/.config/multiai/profiles`) pour rendre le binaire installé utilisable.
12. Corriger `go.mod` (module `github.com/lrochetta/multiai/multiai-go`) ou déplacer le module à la racine pour rendre `go install` réel.

### P2 — Ce trimestre
13. Trier `SelectTool` (menu déterministe) ; JSON via `encoding/json` dans `ShowEffectiveEnv` ; honorer `ClearEnv` ; messages d'erreur alignés sur la syntaxe Go ; warnings sur profils ignorés dans `LoadDir`.
14. PS : porter l'écriture atomique dans `Set-ProfileSecret` ; `$PSCommandPath` dans `Get-RouterHash` ; ne pas déclencher le fallback sur exit 130 / interruption ; retirer les claims SecureString ou les implémenter.
15. Converger Go/PS : le Go doit rattraper le catalogue 14 fournisseurs + fallback chains + erase keys, ou la doc doit dire clairement que le PS est l'implémentation de référence v0.3.0.
16. Retirer les badges de score auto-attribués des README.

---

## Findings contre-verifies

| ID | Severite | Titre | Verdict | Note |
|---|---|---|---|---|
| 04-01 | critical | Flux config→launch casse : sentinel `__MULTIAI_CREDSTORE__` jamais resolu, cle API perdue | CONFIRMED | Preuve exhaustive : sentinel ecrit (wizard.go:269), jamais relu ; perte silencieuse de cle confirmee (wizard.go:290-295). |
| 04-02 | high (corrigee, initialement critical) | `%USERPROFILE%` jamais expanse : profils livres incompatibles avec le binaire Go | PARTIAL | Mecanisme confirme, mais le pattern `%OPENROUTER_API_KEY%` ne vit que cote PS ; impact reel = isolation de config cassee sur 6/17 profils Go, pas echec d'auth total → high. |
| 04-03 | high (corrigee, initialement critical) | Whitelist env case-sensitive : PATH/ComSpec/windir supprimes sous Windows | PARTIAL | Reproduit empiriquement (child sans PATH), mais SYSTEMROOT est re-injecte par os/exec (addCriticalEnv) et le produit npm livre est le PS, pas le Go (WIP) → high. |
| 04-04 | high (corrigee, initialement critical) | CI/CD entierement inerte : workflows hors racine + trigger sur `main` | PARTIAL | Faits integralement confirmes (double neutralisation, gofmt -l = 3 fichiers), mais repo prive et publication npm manuelle : defaillance de processus, pas vulnerabilite active → high. |
| 04-05 | high | Hooks before/after_launch : code mort — `py.Hooks` jete, `opts.Hooks` jamais assigne | CONFIRMED | Chaine complete verifiee ; hooks.go (164 lignes) inatteignable ; claim CHANGELOG.md:117 faux. |
| 04-06 | medium (corrigee, initialement high) | Echappement des hooks mal ordonne : `os.ExpandEnv` apres `escapeShellArg` reintroduit l'injection | PARTIAL | Faits exacts, mais pas de frontiere de privilege (le hook vient de la config locale) et code mort (cf. 04-05) : defaut avant tout fonctionnel → medium. |
| 04-07 | high | Credential store « natif » fictif + round-trip base64 casse (Windows/macOS) | CONFIRMED | Set encode base64, Get ne decode jamais ; branches `cmdkey` identiques ; claim README faux ; store write-only en production. |
| 04-08 | high | Les echecs de lancement sortent avec exit code 0 | CONFIRMED | `runLaunch` retourne nil sur toute erreur ; `jsonError` a zero appelant ; divergence de contrat avec les codes 0-4 du PS. |
| 04-09 | high | `defer close(sigCh)` execute avant `signal.Stop` : panic « send on closed channel » possible | CONFIRMED | Ordre LIFO verifie ; fenetre atteignable au double Ctrl+C ; forwarding no-op Windows et double-SIGINT Unix confirmes (le contre-verificateur suggere un impact plutot medium : crash de teardown, pas de corruption). |
| 04-10 | high | Binaire installe inutilisable : profils cherches uniquement pres de l'exe ou du cwd | CONFIRMED | Aucun fallback utilisateur ni profil embarque ; aucune methode de packaging ne depose configs/profiles. |
| 04-11 | high | `go install` impossible : chemin de module incoherent avec le sous-dossier `multiai-go/` | CONFIRMED | Echec certain ; aggravant : le main est dans cmd/multiai ; repo prive en prime. |
| 04-12 | high | CHANGELOG v0.3.0 : models/search/compare, cache 1h et estimation de cout inexistants | CONFIRMED | Les 5 points verifies (switch main.go, zero Invoke-RestMethod, cache jamais appele, Write-CostLog sans cout, aucune fonction search). |
| 04-13 | medium | Profils YAML et `.multiai.yaml` projet : jamais branches au CLI (code mort) | non contre-verifie | — |
| 04-14 | medium | Packages onboarding, logging et openrouter entierement morts ; marqueur first-run jamais lu | non contre-verifie | — |
| 04-15 | medium | `CLEAR_ENV` parse mais jamais honore par le lanceur Go | non contre-verifie | — |
| 04-16 | medium | Chaos de versions (5 sources divergentes) + injection ldflags `-X` inoperante sur une const | non contre-verifie | — |
| 04-17 | medium | `ShowEffectiveEnv --json` : JSON fabrique a la main sans echappement → invalide | non contre-verifie | — |
| 04-18 | medium | Menu de selection d'outil non deterministe (iteration de map Go) | non contre-verifie | — |
| 04-19 | medium | Mutex du store chiffre neutralise : store neuf par operation (win/darwin) | non contre-verifie | — |
| 04-20 | medium | `encryptedFileStore` utilise `os.Getenv("HOME")` : chemin casse sous Windows | non contre-verifie | — |
| 04-21 | medium | gofmt non conforme sur 3 fichiers (jamais detecte, CI inerte) | non contre-verifie | — |
| 04-22 | medium | Messages d'erreur du binaire Go referencant la syntaxe PowerShell | non contre-verifie | — |
| 04-23 | medium | PS : `Set-ProfileSecret` non atomique (fix v0.2.1 #9 non porte en PS) | non contre-verifie | — |
| 04-24 | low | PS : `Get-RouterHash` retourne toujours null — hash SHA256 jamais logge | non contre-verifie | — |
| 04-25 | low | Claims securite faux dans les README : SecureString (PS) et credential store natif (Go) | non contre-verifie | — |
| 04-26 | low | `LoadYAML` lit le fichier entier avant le check de taille (YAML bomb incomplete) | non contre-verifie | — |
| 04-27 | low | Profils `.env` corrompus ou illisibles ignores silencieusement | non contre-verifie | — |
| 04-28 | low | Duplication : whitelist commandes, tri profils, masquage secrets, hooks before/after | non contre-verifie | — |
| 04-29 | low | PS : fallback chain declenchee sur tout exit non-zero, y compris Ctrl+C | non contre-verifie | — |
| 04-30 | low | `multiai list --json` emet `null` au lieu de `[]` pour une liste vide | non contre-verifie | — |
| 04-31 | low | Completions shell avec shortcuts hardcodes deja divergents des profils reels | non contre-verifie | — |
| 04-32 | low | Encodage console incoherent (ASCII strippe / accents / box-drawing, pas de codepage) | non contre-verifie | — |

Aucun finding REFUTED : les 12 findings critical/high contre-verifies sont tous confirmes sur le fond ; 4 severites recalibrees (04-02, 04-03, 04-04 : critical→high ; 04-06 : high→medium).
