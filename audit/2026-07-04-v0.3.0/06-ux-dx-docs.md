# Audit v0.3.0 — UX / DX / Documentation

Date 2026-07-04 · Auditeur Sentinel (Quality — UX/DX & documentation) · Score : 4.5/10 · Méthode : audit BMAD+ parallèle + contre-vérification adversariale.

---

## Résumé

Le paradoxe de cette version : le **produit interactif s'est amélioré** (les 4 quick wins UX de v0.2.1 sont réellement livrés dans le code Go), mais la **documentation a décroché de la réalité** à un point qui invalide la promesse du projet. Le README racine décrit un produit qui n'existe pas : commandes `multiai models` / `search` / `compare` introuvables dans les deux implémentations, fonctionnalités v0.3.0 (régions, fallback, cost logging) présentes uniquement dans le PowerShell « legacy », packages Go entiers jamais branchés au binaire (`onboarding`, `openrouter`, YAML/hooks), binaire qui s'annonce `multiai 0.2.1` alors que le CHANGELOG publie 0.3.0. Enfin, un `git clone` du dépôt livre **zéro profil Go** et un jeu PowerShell amputé de ses profils de base : le first-run depuis GitHub est cassé. Pour un projet qui ambitionne d'être « LE meilleur routeur multi-IA du marché », la crédibilité documentaire est aujourd'hui son pire ennemi.

---

## Forces

- **Navigation Go corrigée** (#10 v0.2.1) : « 0. Retour » à chaque niveau — `multiai-go/internal/menu/interactive.go:73` et `:119`, `internal/config/wizard.go:139`, avec boucle de retour outil→profil dans `cmd/multiai/main.go:241-260`.
- **Accessibilité corrigée** (#11 v0.2.1) : préfixes textuels `[OK]`/`[!]`/`[X]`/`[i]` (`internal/cli/display.go:60-78`) + support `NO_COLOR` (`display.go:14`).
- **Boucle interactive** : le menu Go revient au menu principal après chaque action (`main.go:185-213`), aligné sur le comportement PowerShell (`code-router.ps1:1044-1081`).
- **`--help` contextuel** par sous-commande (`main.go:112-124`, `printLaunchHelp`/`printListHelp`/`printConfigHelp`).
- **Messages d'erreur avec suggestion** sur secret manquant : « Edite : <path> / Ou lance : multiai config » (`internal/cli/launcher.go:224-225`).
- **Wizard de configuration des clés soigné** : statut [OK]/[~~]/[--] par fournisseur, URL de création de clé, masquage `sk-a...x7f2`, validation regex par fournisseur avec confirmation en cas d'échec (`internal/config/wizard.go:107-255`).
- **EOF stdin → exit propre** (`interactive.go:29-31`), exit code enfant propagé (`main.go:153-155`).
- **PowerShell réellement enrichi** : menu erase keys (`code-router.ps1:566-613`), groupement par région dans la config (`:696-704`), chaîne de fallback fonctionnelle (`:1135-1163`), log d'usage (`:1010-1021`).
- **UX PS install** : préservation des `.env` existants en `.new`, nettoyage des anciens noms `aicode`, gestion PATH (`multiai-powershell/install.ps1:70-144`).

---

## Constats détaillés

### 1. Documentation vs code : des commandes annoncées qui n'existent pas (CRITIQUE)

- `README.md:88-90` (« Usage rapide ») annonce `multiai models`, `multiai search "claude"`, et `README.md:148-151` détaille `models`/`search`/`compare` comme livrés en v0.3.0. `CHANGELOG.md:16-18` les liste comme « Added ».
- **Réalité Go** : le switch de `cmd/multiai/main.go:126-182` ne connaît que `version`, `help`, `list`, `launch`, `config`, `completion`. `multiai models` → « Commande inconnue : models » + exit 1.
- **Réalité PowerShell** : le bloc `param()` de `code-router.ps1:33-46` n'a aucun switch models/search/compare. Le menu « 4. OpenRouter » (`:880-940`) est un écran statique qui affiche l'URL https://openrouter.ai/models et propose de créer un profil à la main — aucune découverte dynamique, aucune recherche, aucune comparaison, aucun « top modèles par usage, catégorie, prix ».
- `internal/openrouter/client.go` existe (FetchModels, cache) mais **n'est importé nulle part** — dead code. Il ne contient d'ailleurs ni fonction de recherche ni de comparaison, contrairement à `CHANGELOG.md:25` (« client API, cache, search »).

### 2. Version : le binaire ment sur lui-même (CRITIQUE)

- `cmd/multiai/main.go:18` : `const version = "0.2.1"` → `multiai version` affiche 0.2.1.
- `internal/menu/interactive.go:18` : titre codé en dur « Laurent ROCHETTA's MultiAI (v0.2.1) » — deuxième source de vérité divergente.
- `internal/openrouter/client.go:38` : User-Agent « multiai/0.2.1 ».
- Pendant ce temps `CHANGELOG.md:7` publie [0.3.0] — 2026-06-24 et `multiai-powershell/package.json:3` est à 0.3.0. Le PS, lui, lit sa version dynamiquement depuis package.json (`code-router.ps1:52-59`) — la bonne pratique existe dans le repo mais pas côté Go.

### 3. Un clone GitHub est inutilisable : profils absents du dépôt (CRITIQUE)

- `.gitignore:2` (`*.env`) ignore tous les profils. `git ls-files multiai-go/configs/` = **0 fichier** : l'implémentation « primaire » Go est publiée **sans aucun profil** (vérifié via `git check-ignore` : `multiai-go/configs/profiles/00-claude-official.env` ignoré par `.gitignore:2`).
- Côté PowerShell, seuls 20 `.env` sont trackés (01-03 + 60-83) ; les profils de base 00, 10, 20-57 (`co`, `ca`, `cg`, `ds`, `dsf`, `codex55`, `ocdeepseek`…) sont absents du dépôt alors que 37 fichiers existent localement (`multiai-powershell/configs/profiles/`).
- Conséquence : `git clone` + `go build` → `LoadDir` échoue (« cannot read profiles directory », `internal/profile/profile.go:42-44`) et le programme sort en code 2 à chaque lancement (`main.go:187-190`). Le README (`README.md:96`) promet « 20+ profils inclus ».

### 4. Packages Go fantômes : onboarding, openrouter, logging, YAML, hooks (HAUTE)

- **Onboarding (#13 v0.2.1)** : `internal/onboarding/wizard.go` existe (IsFirstRun, RunWelcome) mais **aucun appel** dans `main.go` ni ailleurs (grep `onboarding` : seules auto-références). Le marqueur `.first-run-done` est écrit (`wizard.go:68-73`) mais jamais lu — même `IsFirstRun` ne le consulte pas. `CHANGELOG.md:86` le présente comme livré (« internal/onboarding/ — Wizard premier démarrage ») : c'est du code mort. Le wizard affiche en outre « 5 fournisseurs » (`wizard.go:40`) quand le README vend « 14+ ».
- **YAML / .multiai.yaml / hooks** : `README.md:168-170` promet « Profils YAML + .multiai.yaml par projet avec héritage » et « Plugin hooks before_launch/after_launch ». Or `profile.LoadDir` ne lit que les `.env` (`profile.go:49`), `FindProjectConfig` (`internal/profile/project.go:13`) n'est jamais appelé depuis `main.go`, et `LaunchOptions.Hooks` n'est jamais renseigné (`main.go:266-273` construit les options sans Hooks). Trois pages de docs (`multiai-go/docs/advanced/yaml-profiles.md`, `project-profiles.md`, `plugin-hooks.md`) documentent des fonctionnalités inaccessibles dans le binaire.
- **Logging** : `internal/logging/logger.go` n'est référencé que par le package onboarding, lui-même mort.
- `jsonError` (`internal/cli/launcher.go:212-215`) : helper revendiqué comme fix UX v0.2.6 (`CHANGELOG.md:62`), défini mais jamais utilisé — les erreurs en mode `--json` restent du texte brut sur stderr.

### 5. Incohérence Go vs PowerShell aggravée (#12 v0.2.1) (HAUTE)

Le README racine présente un produit unique ; il y en a deux, très inégaux :

| Capacité | PowerShell (legacy) | Go (« primaire ») |
|---|---|---|
| Menu principal | 4 options avec OpenRouter (`code-router.ps1:991-994`) | 3 options (`main.go:206-211`) |
| BMAD+ | Menu complet détection/update (`ps1:757-876`) | Stub « pas encore integre » (`main.go:206-208`) |
| Fournisseurs config | 14 avec régions (`ps1:87-209`) | 5 (`wizard.go:58-94`) |
| Profils | 37 locaux | 17 (aucun profil 60-83) |
| Erase keys | Oui (`ps1:566-613`) | Non (menu wizard.go:137-141 : 1-5/a/0) |
| Fallback chain | Oui (`ps1:1135-1163`) | Non (aucune occurrence FALLBACK) |
| Cost log / régions | Oui | Non |
| Expansion `%VAR%` dans .env | Oui (`ps1:414-418`) | Non — `safeExpandEnv` ne gère que `$VAR` (`internal/env/env.go:24-31`) : les profils 60-83 utilisant `%OPENROUTER_API_KEY%` seraient silencieusement cassés en Go |

Le transcript du README (`README.md:28-64`) montre le menu PowerShell (4 options, profils or-fusion/mm/mimo) présenté comme LE produit — un utilisateur Go n'obtiendra jamais cet écran. Pire, les messages d'erreur Go renvoient la syntaxe PowerShell : « Lance 'multiai -List' » (`internal/profile/profile.go:163`) et « Utilise -AllowCustomCommand » (`internal/cli/launcher.go:61`) au lieu de `multiai list` / `--allow-custom-command`.

### 6. Claims marketing non vérifiables ou faux (HAUTE)

- **Badge « score 9.5/10 »** (`README.md:9`) et « score 10/10 » (`multiai-go/README.md:9`) : auto-proclamés, sans lien, alors que l'audit interne du 2026-06-23 donnait 5.5/10. Sur un dépôt public, c'est un boomerang de crédibilité.
- **« Cost logging : estimation coût par requête + cumul session »** (`README.md:163`, `CHANGELOG.md:14`) : `Write-CostLog` (`code-router.ps1:1010-1021`) n'enregistre que timestamp, shortcut, exit code et durée. Aucune estimation de coût, aucun cumul, aucun prix nulle part.
- **« Fusion — panel d'experts multi-modèles avec synthèse automatique »** (`README.md:151`) : aucun code de fusion/synthèse dans le repo ; le profil (`multiai-powershell/configs/profiles/60-claude-fusion.env:13-17`) pointe simplement `ANTHROPIC_MODEL=openrouter/fusion` — tout repose sur un slug OpenRouter dont l'existence n'est étayée nulle part.
- **« Cache 1h, fallback offline »** (`README.md:152`) : le seul code de cache est le client Go mort (`openrouter/client.go:60-96`) ; le PS n'a aucun cache.
- **« BMAD+ intégré — détection automatique, version, packs »** (`README.md:174-176` et `multiai-go/README.md:114-117`) : vrai en PS, faux en Go — le README Go documente cette feature dans le même fichier où elle n'existe pas.
- **Chiffres gonflés** : « 45+ tests » (`README.md:199`) — comptage réel : 32 fonctions `Test*` + 2 `t.Run` + 2 benchmarks ≈ 36. « CLI 7 sous-commandes » (`README.md:184`) — il y en a 6. Badge « Go 1.23 » (`README.md:7`) vs `go 1.22` (`multiai-go/go.mod:3`). « Site VitePress : 16 pages » (`CHANGELOG.md:120`) — 13 fichiers .md.
- **`go install github.com/lrochetta/multiai@latest`** (`README.md:74`, `docs/index.md`, `docs/guide/getting-started.md`) : cassé deux fois — le `go.mod` vit dans le sous-dossier `multiai-go/` (pas à la racine du module annoncé) et le package main est dans `cmd/multiai` (il faudrait `.../multiai-go/cmd/multiai@latest` avec un go.mod à la racine du repo).

### 7. DX scripting : le mode --json n'est pas scriptable proprement (MOYENNE)

- En mode `--json --dry-run`, `ValidateAndLaunch` imprime « [DRY RUN] Simulation sans lancement » et la commande sur **stdout** (`launcher.go:97-98`) avant que `main.go:281-285` n'émette le JSON → `multiai launch -p ds --dry-run --json | jq` casse.
- `--show-env --json` : JSON fabriqué à la main par `fmt.Printf` sans échappement (`launcher.go:176-194`). Les valeurs contenant `\` (ex. `CLAUDE_CONFIG_DIR=%USERPROFILE%\.claude-fusion`) ou `"` produisent un JSON invalide ; et combiné au LaunchResult on obtient deux documents JSON concaténés sur stdout.
- `hasFlag`/`getFlagValue` (`main.go:292-312`) scannent tous les args y compris après `--` : `multiai launch -p ds -- --json` active le mode JSON de multiai en plus de passer le flag à l'enfant.
- Les erreurs en `--json` restent en texte brut sur stderr (le helper `jsonError` est mort, cf. constat 4).
- `multiai config --provider <id>` : documenté dans l'aide (`main.go:51` et `:94-95`) mais le case `config` (`main.go:157-166`) ne parse jamais `--provider` — flag fantôme.

### 8. Documentation VitePress : hors dépôt, hors CI, périmée (MOYENNE)

- `.gitignore:31` (`docs/`) ignore **tout** dossier docs → les 13 pages de `multiai-go/docs/` ne sont pas trackées (`git ls-files` ne retourne aucun fichier docs). Le site n'est ni versionné, ni buildé, ni déployé — aucun workflow ne mentionne vitepress/pages (`.github/workflows/ci.yml`, `release.yml`).
- Contenu figé v0.2.0 : « 17 profils inclus » (`docs/index.md`, features), rien sur v0.3.0.
- `docs/reference/commands.md:44-45` documente des flags inexistants : `--dry-run | -n` (le raccourci `-n` n'existe pas) et `--verbose | -v` (aucune occurrence dans main.go).
- Le dossier `docs/` **à la racine du repo** est vide (0 fichier) — coquille abandonnée.

### 9. Exit codes documentés non respectés (MOYENNE)

- `multiai-go/README.md:208-216` et `code-router.ps1:1024-1029` documentent une grille 0-4 (1=user, 2=config, 3=système, 4=processus enfant).
- PS : le `trap` global (`ps1:1031-1036`) sort en `exit 1` pour **toutes** les exceptions — les codes 2 et 3 ne sont jamais émis.
- Go : seuls 0, 1 et 2 sont utilisés (`main.go:137,143,150,160,165,175,181`) ; 3 et 4 jamais.

### 10. Encodage / typographie : régression partielle (#11 résiduel, MOYENNE-BASSE)

- v0.2.1 avait éradiqué le em dash (`CHANGELOG.md:99` : « — → -- ASCII ») ; le Go le réintroduit : « — %d profils » (`interactive.go:18`) et ligne `strings.Repeat("─", 58)` (`interactive.go:19`, aussi `wizard.go:110`) — mojibake garanti sous conhost CP850, alors que le PS utilise `('-' * 58)` ASCII (`ps1:989`).
- Accents incohérents dans la même UI Go : « Configurer les clés API » (`interactive.go:22`) vs « Configuration des cles API » (`wizard.go:109`), « Retour a la selection d'outil » (`interactive.go:119`). Ce n'est pas un choix CP850 délibéré (sinon zéro accent partout comme en PS), c'est de la négligence.
- `launcher.go:59` : « ⚠ Commande custom autorisee » — symbole Unicode seul, sans préfixe texte `[!]`, dernier vestige du problème #11.
- Français uniforme par ailleurs (UI, README, docs) : la dissonance EN/FR de v0.1.5 est résolue.

### 11. Menus sans porte de sortie (MOYENNE)

- Go : `runInteractiveLoop` (`main.go:185-213`) boucle infinie, options « 1, 2, 3 », aucun « 0. Quitter » / « q » ; seuls Ctrl+C ou EOF sortent.
- PS : idem, menu 1-4 sans option quitter (`ps1:991-996`), `default { continue }` (`ps1:1069`).
- Un choix invalide en Go affiche « Choix invalide. Options : 1, 2, 3 » (`main.go:210`) sans mentionner comment quitter.

### 12. Complétion shell figée (BASSE)

- 4 shells livrés comme annoncé (`internal/cli/completion.go:6-84`) — conforme.
- Mais les shortcuts sont hardcodés (18 profils, `completion.go:18` et `:64`) : aucun des profils v0.3.0 (or-fusion, mm, stepfun, req-*, litellm…), pas de génération dynamique depuis `LoadDir`, et divergence garantie à chaque ajout de profil.

### 13. Hygiène de la racine du dépôt (BASSE mais très visible)

- **Trackés (publiés sur GitHub)** : `push-github.ps1` — script interne de publication mentionnant « repo PRIVE », tag v0.2.0, workflow npm perso. N'a rien à faire dans un dépôt destiné au public.
- **Non trackés mais présents localement** (pollution de l'espace de travail, risque de fuite) : 5 zips dont doublons `claude-code-zai-pack (1).zip` / `claude-code-zai-pack(1).zip`, dossier `brainstorm laurent/` contenant `clé deepseek ne pas mettre dans le repo.txt` (le nom du fichier admet qu'une clé réelle traîne sur disque), `60-62-*.env` dupliqués à la racine, `brainstorm-openrouter.md`, `.commit-msg`, `docs/` vide.
- `CLAUDE.md` est tracké et référence `.agents/skills/` et `.agents/data/role-triggers.yaml`… qui sont gitignorés (`.gitignore:33`) — références mortes pour tout consommateur du dépôt public.
- `multiai-go/ROADMAP.md` : « v0.2.0 (en cours) » avec cases non cochées et « v0.3.0 — Securisation » à faire — contredit frontalement le CHANGELOG qui publie v0.3.0 avec un tout autre contenu.

---

## Statut des problèmes v0.2.1

| # v0.2.1 | Problème | Statut | Preuve |
|---|---|---|---|
| #10 | Navigation sans retour en Go | ✅ **CORRIGÉ** | `interactive.go:73,86-88` et `:119,132-134` ; boucle `main.go:241-260` |
| #11 | Couleurs sans préfixe texte | ✅ **CORRIGÉ (résidu)** | `display.go:60-78` [OK]/[!]/[X]/[i] + NO_COLOR `:14` ; résidu « ⚠ » seul `launcher.go:59` |
| #12 | Incohérence Go vs PowerShell | ❌ **AGGRAVÉ** | Toutes les features v0.3.0 sont PS-only ; menus 3 vs 4 options ; 5 vs 14 fournisseurs ; messages Go avec flags PS (`profile.go:163`, `launcher.go:61`) ; `%VAR%` non supporté en Go (`env.go:24-31`) |
| #13 | Wizard onboarding inexistant | ❌ **PERSISTE** | `internal/onboarding/wizard.go` écrit mais jamais appelé (aucune référence hors du package) ; marqueur `.first-run-done` écrit (`wizard.go:68-73`) jamais lu ; CHANGELOG v0.2.6:86 le déclare pourtant livré |
| — (19) | Messages d'erreur avec suggestions | ✅ partiellement | `launcher.go:224-225` (bon) ; mais « commande introuvable » sans suggestion d'install (`launcher.go:67`) |
| — (18) | NO_COLOR | ✅ | `display.go:14` |
| — (20) | JSON enrichi PID/timestamp/exit_code | ✅ | `launcher.go:42-52` |

---

## Recommandations priorisées

### P0 — Rétablir l'intégrité documentaire (2-3 h, avant toute pub)
1. **Purger le README racine** de tout ce qui n'existe pas : `models`/`search`/`compare` (`README.md:88-90,148-151`), « estimation coût », « synthèse automatique », « Cache 1h ». Réécrire la section Fonctionnalités en deux colonnes explicites « Go » / « PowerShell (npm) ».
2. **Supprimer les badges de score auto-proclamés** (`README.md:9`, `multiai-go/README.md:9`) — ou les remplacer par de vrais badges CI/couverture.
3. **Corriger la version** : une seule source (`main.go:18` → 0.3.x via `-ldflags`), supprimer le hardcode de `interactive.go:18`, aligner `openrouter/client.go:38`.
4. **Tracker les profils** : `git add -f multiai-go/configs/profiles/*.env` (ce sont des templates avec placeholders, le `prepublishOnly` le prouve) + compléter les profils PS manquants (00-57). Sans ça, le dépôt public ne fonctionne pas.

### P1 — Brancher ou supprimer le code mort (1-2 j)
5. Appeler `onboarding.IsFirstRun`/`RunWelcome` dans `runInteractiveLoop` (ou supprimer le package et retirer la ligne du CHANGELOG).
6. Brancher `openrouter.FetchModels` sur de vraies sous-commandes `models`/`search` — ou retirer les claims.
7. Câbler `FindProjectConfig` + `Hooks` dans `main.go`/`runLaunch`, ou dépublier les 3 pages docs `advanced/`.
8. Porter en Go : catalogue 14 fournisseurs, erase keys, fallback chain, régions — c'est la condition pour que « Go primaire » soit vrai (#12).

### P2 — DX scriptable (0.5 j)
9. En mode `--json`, supprimer toute sortie humaine sur stdout (`launcher.go:97-98,129`) et générer `ShowEffectiveEnv` via `encoding/json` (`launcher.go:176-194`).
10. Utiliser (ou supprimer) `jsonError` ; émettre les erreurs JSON sur stdout en mode `--json`.
11. Ne parser les flags multiai qu'avant `--` (`main.go:292-312`).
12. Implémenter `config --provider` ou le retirer de l'aide (`main.go:51,94`).
13. Corriger les messages Go utilisant la syntaxe PS (`profile.go:163`, `launcher.go:61`).

### P3 — Finitions UX/docs (1 j)
14. Option « q. Quitter » dans les menus Go et PS.
15. Politique typographique unique : ASCII sans accents partout (comme PS) ou UTF-8 assumé avec configuration console Windows — pas le mélange actuel.
16. Sortir `docs/` VitePress du `.gitignore` (pattern trop large `docs/`), le mettre à jour v0.3.0, corriger `commands.md` (flags `-n`/`-v` inexistants), ajouter un deploy Pages en CI.
17. Nettoyer la racine : supprimer `push-github.ps1` du tracking, zips, `brainstorm laurent/` (et révoquer/déplacer la clé DeepSeek mentionnée), `docs/` vide, `.env` racine dupliqués.
18. Mettre à jour `ROADMAP.md` et aligner les chiffres du README (tests, sous-commandes, version Go).
19. Documenter le vrai chemin `go install` (module dans `multiai-go/`) ou déplacer le go.mod à la racine.

---

## Justification de la note (4.5/10)

| Critère | Note | Justification |
|---|---|---|
| UX interactive (menus, erreurs, accessibilité) | 6.5/10 | Quick wins v0.2.1 réellement livrés ; reste : pas de quitter, typographie incohérente, wizard mort |
| DX scriptable (--json, flags, complétion, exit codes) | 4.5/10 | JSON pollué/invalide, flags fantômes, complétion figée, exit codes non conformes |
| Cohérence Go/PS | 3/10 | Aggravée : v0.3.0 est PS-only, le « primaire » Go est en retard de deux versions fonctionnelles |
| Documentation (véracité, fraîcheur, complétude) | 2.5/10 | Commandes inexistantes en tête de README, badges auto-proclamés, docs hors dépôt, ROADMAP contradictoire, first-run cassé depuis git |
| **Global dimension** | **4.5/10** | La documentation est le produit d'appel d'un CLI open-source ; ici elle décrit un produit imaginaire |

---

## Findings contre-vérifiés

Contre-vérification adversariale appliquée aux findings critical/high ; aucun finding REFUTED (donc aucun écarté). Les verdicts PARTIAL affichent la sévérité corrigée.

| ID | Sévérité | Titre | Verdict | Note |
|---|---|---|---|---|
| 06-03 | critical | Dépôt git sans aucun profil Go + profils PS de base manquants — first-run cassé depuis GitHub | CONFIRMED | Exit 2 sur toutes les commandes après clone+build ; aucun go:embed ni scaffolding ; aucun canal (npm installer Go, scoop, brew) n'embarque les profils |
| 06-04 | high | Wizard onboarding écrit mais jamais appelé (code mort) | CONFIRMED | Aucun import hors du package ; marqueur `.first-run-done` écrit jamais lu ; CHANGELOG v0.2.6 le déclare livré |
| 06-05 | high | Profils YAML, .multiai.yaml et hooks annoncés mais débranchés du binaire | CONFIRMED | `LoadDirYAML`/`FindProjectConfig`/`Hooks` uniquement appelés par les tests ou jamais ; 3 pages docs inaccessibles |
| 06-06 | high | Features v0.3.0 (régions, fallback, cost log, erase keys, 14 fournisseurs) uniquement en PowerShell legacy | CONFIRMED | 0 occurrence FALLBACK/REGION/costs.log/erase côté Go ; `%VAR%` non expansé en Go ; profils 60-83 absents du Go |
| 06-07 | high | Claim « Cost logging : estimation coût + cumul session » faux | CONFIRMED | `Write-CostLog` ne logge que timestamp/shortcut/exit/durée ; aucun prix, aucun cumul, aucun agrégateur |
| 06-08 | high | Badges de score auto-proclamés (9.5/10 et 10/10) sur un projet audité à 5.5/10 | CONFIRMED | Badges statiques à lien vide, contredits par audit/07 (5.5/10, 3 critiques ouverts) et contradictoires entre eux |
| 06-09 | high | `go install github.com/lrochetta/multiai@latest` cassé (module dans un sous-dossier) | CONFIRMED | go.mod dans `multiai-go/` avec chemin de module racine + package main dans `cmd/multiai` : double échec ; documenté à 8+ endroits |
| 06-01 | medium (corrigée, initialement critical) | Commandes `multiai models`/`search`/`compare` annoncées mais inexistantes | PARTIAL | Faits confirmés dans les deux implémentations (openrouter/client.go = dead code sans search/compare) ; sévérité recalibrée : échec propre, ni sécurité ni perte de données — écart docs/implémentation sur package publié |
| 06-02 | low (corrigée, initialement critical) | Binaire Go déclaré v0.2.1 alors que la release est v0.3.0 | PARTIAL | Faits confirmés (+ bug aggravant : `version` est un const, l'injection ldflags de goreleaser/Makefile/AUR/brew est inopérante) ; mais le binaire Go n'est pas distribué dans la release npm 0.3.0 et aucun tag v0.3.0 n'existe → drift cosmétique d'un composant non shippé |
| 06-10 | medium | Mode --json non scriptable : sortie humaine mêlée au JSON, JSON fabriqué non échappé | non contre-vérifié | |
| 06-11 | medium | Messages d'erreur Go utilisant la syntaxe de flags PowerShell | non contre-vérifié | |
| 06-12 | medium | `multiai config --provider` documenté mais non implémenté | non contre-vérifié | |
| 06-13 | medium | Docs VitePress hors dépôt, non déployées, périmées, flags inexistants documentés | non contre-vérifié | |
| 06-14 | medium | Exit codes 0-4 documentés mais non respectés (Go et PS) | non contre-vérifié | |
| 06-15 | medium | Aucune option Quitter dans les menus interactifs Go et PowerShell | non contre-vérifié | |
| 06-16 | medium | Typographie Go incohérente : em dash/box-drawing réintroduits, accents mélangés | non contre-vérifié | |
| 06-17 | medium | Claim « Fusion — panel d'experts avec synthèse automatique » sans aucun code | non contre-vérifié | |
| 06-18 | low | Fichiers parasites à la racine, script interne tracké, clé API sur disque | non contre-vérifié | |
| 06-19 | low | ROADMAP.md obsolète et en contradiction avec le CHANGELOG | non contre-vérifié | |
| 06-20 | low | Complétion shell figée sur 18 shortcuts, sans les profils v0.3.0 | non contre-vérifié | |
| 06-21 | low | Chiffres gonflés dans le README (tests, sous-commandes, version Go) | non contre-vérifié | |
