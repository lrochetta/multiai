# Contre-vérification adversariale — procès-verbal

**Projet** : multiai v0.3.0 — **Audit** : 2026-07-04 — **Rapports couverts** : 01 à 07
**Rédigé par** : Sentinel (BMAD+ Quality)

---

## 1. La méthode

Chaque rapport de dimension (01 à 07) a été soumis à une phase de contre-vérification adversariale avant consolidation :

- **Findings critical** : 2 réfutateurs adversariaux indépendants, à angles différents —
  1. *exactitude technique* (les faits allégués sont-ils vrais, ligne par ligne ?) ;
  2. *exploitabilité / sévérité* (l'impact réel justifie-t-il la sévérité attribuée ?).
- **Findings high** : 1 réfutateur adversarial.
- **Cap** : 14 vérifications maximum par dimension ; tout surplus est marqué « non contre-vérifié ». (Sur ce run, aucune dimension n'a atteint le cap : de 6 à 12 findings vérifiés par dimension.)
- **Findings medium/low** : non vérifiés **par conception** (exception ponctuelle : 05-02, low, contre-vérifié à l'occasion de la requalification d'un comptage).
- **Mission des réfutateurs** : RÉFUTER le finding — le doute profite à la réfutation, pas à l'auditeur.
- **Fusion des votes** :
  - tous les réfutateurs votent REFUTED → finding **écarté** ;
  - au moins un vote REFUTED ou PARTIAL → verdict **PARTIAL**, avec sévérité recalibrée à la valeur **la moins alarmiste** proposée ;
  - sinon → **CONFIRMED**.

**Chiffres globaux du run** : 135 findings émis, 41 CONFIRMED, 0 REFUTED, 99 agents mobilisés au total (auditeurs + réfutateurs).

---

## 2. Tableau consolidé complet (135 findings)

Dimensions : 01 Produit & fonctionnalités · 02 Distribution & packaging · 03 Architecture · 04 Qualité de code · 05 Tests & CI/CD · 06 UX / DX / Documentation · 07 Sécurité.
« NCV » = non contre-vérifié. La colonne Sévérité finale indique entre parenthèses la sévérité initiale lorsqu'elle a été recalibrée.

| ID | Dimension | Sévérité finale | Titre | Verdict | Note |
|---|---|---|---|---|---|
| 01-02 | 01 Produit | critical | Flux config→launch cassé en Go : le littéral `__MULTIAI_CREDSTORE__` est exporté comme clé API | CONFIRMED | Chaîne complète vérifiée (wizard.go:269/:295, aucun `store.Get` en production, IsPlaceholder aveugle au marqueur) ; aggravant : échec dès la même session (rechargement disque main.go:187) |
| 01-03 | 01 Produit | high (← critical) | v0.3.0 inexistante en Go : toutes les features livrées en PowerShell « legacy » uniquement | PARTIAL | Faits intégralement confirmés (diff Go vide, 17 profils, or-fusion introuvable) mais sévérité recalibrée : le canal npm headline livre la v0.3.0 PS fonctionnelle ; impact limité aux canaux Go déjà cassés |
| 01-04 | 01 Produit | high | « Cost logging : estimation coût + cumul session » — aucune estimation de coût n'existe | CONFIRMED | Write-CostLog = simple log de lancement (timestamp/exit/durée) ; ModelPricing Go jamais utilisé |
| 01-05 | 01 Produit | high | 4 canaux d'installation sur 5 morts : go install impossible, brew/scoop/AUR placeholders, repo privé, aucune release | CONFIRMED | Vérifié sur code + état live GitHub/npm ; même sous-estimé (manifests référencent un tag v0.5.0 fantôme, tap Homebrew inexistant) |
| 01-06 | 01 Produit | high | CI/CD annoncée jamais exécutée : workflows hors de la racine du repo | CONFIRMED | Workflows uniquement sous multiai-go/.github/workflows/, aucun .github/ racine — jamais exécutables par GitHub Actions |
| 01-07 | 01 Produit | high | Profils YAML, .multiai.yaml, héritage `extends` et hooks before/after_launch : code mort non câblé | CONFIRMED | ~200 lignes de code mort ; zéro appelant production de LoadAllProfiles/FindProjectConfig/Hooks ; `Extends` jamais résolu |
| 01-08 | 01 Produit | high | Expansion `%VAR%` non supportée en Go : CLAUDE_CONFIG_DIR passé littéralement, isolation des configs cassée | CONFIRMED | os.Expand ($VAR only) ; les 6 profils Claude Go utilisent %USERPROFILE% → transmis littéralement, isolation silencieusement inopérante |
| 01-09 | 01 Produit | high | Binaire Go inutilisable hors du repo : aucune stratégie de répertoire de profils utilisateur | CONFIRMED | getProfilesDir ne teste que 2 emplacements ; aucun canal de distribution ne livre les profils → exit 2 sur toutes les commandes |
| 01-01 | 01 Produit | medium (← critical) | Commandes `models`/`search`/`compare` annoncées mais inexistantes (vaporware) | PARTIAL | Faits confirmés (switch sans ces cases, client OpenRouter jamais importé, menu PS statique) mais écart doc/implémentation sans impact sécurité ni perte de données : l'utilisateur reçoit une erreur propre |
| 01-10 | 01 Produit | medium | Wizard onboarding écrit mais jamais appelé ; internal/install et internal/update vides | NCV | — |
| 01-11 | 01 Produit | medium | « Profils dynamiques : ajout/suppression à la volée » — la suppression n'existe nulle part | NCV | — |
| 01-12 | 01 Produit | medium | Config Go limitée à 5 fournisseurs et sans menu erase keys, contre 14+ annoncés | NCV | — |
| 01-13 | 01 Produit | medium | Chaos de versions : 0.2.1 (Go) vs 0.3.0 (npm) vs 0.5.0 (packaging/npm) vs roadmap contradictoire | NCV | — |
| 01-14 | 01 Produit | medium | Menu Go : BMAD+ stub et pas d'entrée OpenRouter, contrairement à la démo du README | NCV | — |
| 01-15 | 01 Produit | low | Docs de référence et complétions désynchronisées ; métriques gonflées (tests, pages, Go 1.23) | NCV | — |
| 01-16 | 01 Produit | low | Fichiers profils fusion orphelins à la racine du repo | NCV | — |
| 02-01 | 02 Distribution | high (← critical) | Workflows GitHub Actions jamais exécutés (pas de `.github/` à la racine du repo) | PARTIAL | Faits confirmés ; sévérité recalibrée critical → high (pas de vecteur d'attaque, mais force « CI/CD complète » v0.2.1 invalidée ; aggravant : trigger `branches: [main]` vs branche `master`) |
| 02-02 | 02 Distribution | medium (← critical) | Repo GitHub privé, zéro release, zéro tag distant : URLs de téléchargement mortes | PARTIAL | Faits confirmés ; sévérité recalibrée critical → medium (aucun utilisateur n'emprunte ces chemins aujourd'hui : `multiai-installer` non publié, canal npm réel indépendant des releases GitHub — release blocker latent, pas panne live) |
| 02-03 | 02 Distribution | medium (← critical) | Chaos de versions : 0.2.1 / 0.3.0 / 0.5.0 / ROADMAP « v0.2.0 en cours » | PARTIAL | Faits confirmés (9 emplacements 0.5.0) ; sévérité recalibrée critical → medium (manifests 0.5.0 = templates pré-release ; impact live limité à la confusion Go 0.2.1 vs npm 0.3.0) |
| 02-04 | 02 Distribution | high (← critical) | Binaire Go installé inutilisable : profils ni embarqués ni livrés par aucun canal | PARTIAL | Faits confirmés sur les 7 canaux ; sévérité recalibrée critical → high (release blocker garanti à 100 % pour v0.5.0, mais zéro utilisateur Go atteignable aujourd'hui ; le canal npm PS livre bien ses profils) |
| 02-05 | 02 Distribution | high | Checksums placeholders AUR/Homebrew/Scoop — problème #4 v0.2.1 inchangé | CONFIRMED | Vérifié ; aggravant : pas de section `aurs:` dans goreleaser, syntaxe `arm:/intel:` invalide en Formula, `.SRCINFO` divergent (`SKIP`) |
| 02-06 | 02 Distribution | high | Injection `-X main.version` inopérante : `version` est une `const` | CONFIRMED | Vérifié ; touche aussi Makefile:4, go-build.ps1:60 et User-Agent codé en dur (`openrouter/client.go:38`) |
| 02-07 | 02 Distribution | high | Packages source incompilables : module Go en sous-dossier vs go.mod racine | CONFIRMED | Vérifié point par point (go install, Homebrew, AUR, build manuel tous cassés ; LICENSE absent) |
| 02-08 | 02 Distribution | high | Mismatch nommage archives : goreleaser underscore vs installeurs dash — 404 garanti | CONFIRMED | Vérifié ; aggravant : `install.js` ne teste pas le statusCode et écrirait la page 404 comme archive |
| 02-09 | 02 Distribution | high | Trois identités npm : `multiai` (PS publié), `multiai-installer` et `multiai-cli` (inexistants) | CONFIRMED | Vérifié sur le registre npm (2026-07-05) ; badge npm PS affiché sur la vitrine Go |
| 02-10 | 02 Distribution | high | release.yml cassé : `cd` hors workspace + job Homebrew placeholder + GITHUB_TOKEN cross-repo | CONFIRMED | Vérifié ; aggravant : le workflow ne se déclenchera de toute façon jamais (hors racine) |
| 02-11 | 02 Distribution | high | URLs publiques mortes : rochetta.fr/multiai/install.sh → 404, install.ps1 inexistant | CONFIRMED | Vérifié en live 2026-07-05 (404, et 403 sur User-Agent curl) ; 4 fichiers de doc affectés |
| 02-12 | 02 Distribution | medium | Onboarding fantôme : wizard.go jamais importé, internal/install et internal/update vides | NCV | — |
| 02-13 | 02 Distribution | medium | npm install.js fragile : pas de checksum, statut HTTP ignoré, bin mappé sur l'installeur | NCV | — |
| 02-14 | 02 Distribution | medium | install.sh contredit la doc : version épinglée 0.5.0 et ~/.local/bin vs /usr/local/bin | NCV | — |
| 02-15 | 02 Distribution | medium | Manifeste Scoop déclare une archive windows-arm64 jamais produite par goreleaser | NCV | — |
| 02-16 | 02 Distribution | medium | CHANGELOG v0.3.0 annonce des commandes Go inexistantes (models/search/compare) et 20 profils absents du Go | NCV | — |
| 02-17 | 02 Distribution | low | Badge auto-attribué « score 9.5/10 » sur le README public | NCV | — |
| 02-18 | 02 Distribution | low | ROADMAP obsolète et contradictoire avec le packaging | NCV | — |
| 03-01 | 03 Architecture | critical | Flux config→launch cassé : sentinelle `__MULTIAI_CREDSTORE__` écrite mais jamais relue (credential store write-only) | CONFIRMED | Vérifié ligne par ligne ; aggravant : si `secret.NewStore()` échoue, la clé saisie est perdue définitivement (sentinelle déjà écrite sur disque). |
| 03-04 | 03 Architecture | critical | Profils .env et doc VitePress exclus de git : clone frais inutilisable | CONFIRMED | Exit code 2 sur list/launch/config vérifié ; nuance mineure : les profils PS 01-03 sont trackés, 15 restent absents. |
| 03-06 | 03 Architecture | high | Aucun canal d'installation ne livre les profils (distribution incompatible avec le modèle de données) | CONFIRMED | Vérifié pour install.sh, deb, npm, brew, go install ; le finding est même sous-estimé (repo privé + releases jamais produites). |
| 03-07 | 03 Architecture | high | Chaos de versions : 5 sources contradictoires + injection ldflags sur une const (no-op silencieux) | CONFIRMED | Toutes les versions vérifiées ; un binaire releasé v0.5.0 afficherait 0.2.1. |
| 03-09 | 03 Architecture | high | Stores « natifs » = façades, contrat Set/Get asymétrique (base64 non décodé) | CONFIRMED | Bug latent (aucun Get en production), mais données base64 déjà écrites sur disque sous Windows/macOS ; aggravant : `os.Getenv("HOME")` vide sous Windows. |
| 03-03 | 03 Architecture | high (← critical) | v0.3.0 livrée uniquement en PowerShell « legacy », Go « primaire » stagne à 0.2.1 | PARTIAL | Cœur confirmé ; corrections : catalogue PS = 13 providers (pas 14), et `models/search/compare` n'existent dans AUCUNE implémentation (même pas en PS). Pas de faille sécurité → high. |
| 03-05 | 03 Architecture | high | CI/CD fantôme : workflows hors racine, branche `main` inexistante | PARTIAL | Cœur confirmé (workflows jamais exécutables) ; détail réfuté : release.yml déclenche sur tags `v*`, pas sur `main` — son seul blocage est la localisation hors racine. Sévérité maintenue high. |
| 03-02 | 03 Architecture | medium (← critical) | Code mort d'`internal/` vendu comme features (YAML, héritage, hooks, onboarding, openrouter, logging) | PARTIAL | Mécanique confirmée point par point, mais chiffre recalibré ~31% (pas ~40%) et aucun impact runtime/sécurité : écart doc/produit → medium. |
| 03-08 | 03 Architecture | medium (corrigée) | Contrat de profil divergent Go vs PS : `%VAR%` non expansé, FALLBACK/REGION exportées | PARTIAL | Leg `%USERPROFILE%` actif (6/17 profils Go cassés) ; les legs FALLBACK/REGION et SKIP_SECRET_CHECK sont latents (aucun profil Go livré ne les contient). Sévérité corrigée medium. |
| 03-10 | 03 Architecture | medium | PBKDF2 ajouté mais code mort ; master key en clair dans `.masterkey` | NCV | — |
| 03-11 | 03 Architecture | medium | `internal/cli` fourre-tout : présentation couplée à l'exécution | NCV | — |
| 03-12 | 03 Architecture | medium | Duplications structurelles : whitelist, catalogue providers, shortcuts en 3-5 exemplaires | NCV | — |
| 03-13 | 03 Architecture | medium | Doc VitePress décrit une API imaginaire (`~/.multiai/config.yaml`, hooks string) | NCV | — |
| 03-14 | 03 Architecture | medium | Ordre du menu outils non déterministe (itération de map Go) | NCV | — |
| 03-15 | 03 Architecture | medium | Claims de tests gonflées (34 réels vs « 45+ »), zéro test sur launcher/wizard/stores | NCV | — |
| 03-16 | 03 Architecture | medium | `--allow-custom-command` contourne la whitelist sans validation | NCV | — |
| 03-17 | 03 Architecture | low | Hooks inatteignables + escaping appliqué à la commande entière | NCV | — |
| 03-18 | 03 Architecture | low | Hygiène du dépôt : zips, profils orphelins, ROADMAP contredisant le CHANGELOG | NCV | — |
| 04-01 | 04 Qualité de code | critical | Flux config→launch cassé : sentinel `__MULTIAI_CREDSTORE__` jamais résolu, clé API perdue | CONFIRMED | Preuve exhaustive : sentinel écrit (wizard.go:269), jamais relu ; perte silencieuse de clé confirmée (wizard.go:290-295). |
| 04-02 | 04 Qualité de code | high (← critical) | `%USERPROFILE%` jamais expansé : profils livrés incompatibles avec le binaire Go | PARTIAL | Mécanisme confirmé, mais le pattern `%OPENROUTER_API_KEY%` ne vit que côté PS ; impact réel = isolation de config cassée sur 6/17 profils Go, pas échec d'auth total → high. |
| 04-03 | 04 Qualité de code | high (← critical) | Whitelist env case-sensitive : PATH/ComSpec/windir supprimés sous Windows | PARTIAL | Reproduit empiriquement (child sans PATH), mais SYSTEMROOT est ré-injecté par os/exec (addCriticalEnv) et le produit npm livré est le PS, pas le Go (WIP) → high. |
| 04-04 | 04 Qualité de code | high (← critical) | CI/CD entièrement inerte : workflows hors racine + trigger sur `main` | PARTIAL | Faits intégralement confirmés (double neutralisation, gofmt -l = 3 fichiers), mais repo privé et publication npm manuelle : défaillance de processus, pas vulnérabilité active → high. |
| 04-05 | 04 Qualité de code | high | Hooks before/after_launch : code mort — `py.Hooks` jeté, `opts.Hooks` jamais assigné | CONFIRMED | Chaîne complète vérifiée ; hooks.go (164 lignes) inatteignable ; claim CHANGELOG.md:117 faux. |
| 04-06 | 04 Qualité de code | medium (← high) | Échappement des hooks mal ordonné : `os.ExpandEnv` après `escapeShellArg` réintroduit l'injection | PARTIAL | Faits exacts, mais pas de frontière de privilège (le hook vient de la config locale) et code mort (cf. 04-05) : défaut avant tout fonctionnel → medium. |
| 04-07 | 04 Qualité de code | high | Credential store « natif » fictif + round-trip base64 cassé (Windows/macOS) | CONFIRMED | Set encode base64, Get ne décode jamais ; branches `cmdkey` identiques ; claim README faux ; store write-only en production. |
| 04-08 | 04 Qualité de code | high | Les échecs de lancement sortent avec exit code 0 | CONFIRMED | `runLaunch` retourne nil sur toute erreur ; `jsonError` a zéro appelant ; divergence de contrat avec les codes 0-4 du PS. |
| 04-09 | 04 Qualité de code | high | `defer close(sigCh)` exécuté avant `signal.Stop` : panic « send on closed channel » possible | CONFIRMED | Ordre LIFO vérifié ; fenêtre atteignable au double Ctrl+C ; forwarding no-op Windows et double-SIGINT Unix confirmés (le contre-vérificateur suggère un impact plutôt medium : crash de teardown, pas de corruption). |
| 04-10 | 04 Qualité de code | high | Binaire installé inutilisable : profils cherchés uniquement près de l'exe ou du cwd | CONFIRMED | Aucun fallback utilisateur ni profil embarqué ; aucune méthode de packaging ne dépose configs/profiles. |
| 04-11 | 04 Qualité de code | high | `go install` impossible : chemin de module incohérent avec le sous-dossier `multiai-go/` | CONFIRMED | Échec certain ; aggravant : le main est dans cmd/multiai ; repo privé en prime. |
| 04-12 | 04 Qualité de code | high | CHANGELOG v0.3.0 : models/search/compare, cache 1h et estimation de coût inexistants | CONFIRMED | Les 5 points vérifiés (switch main.go, zéro Invoke-RestMethod, cache jamais appelé, Write-CostLog sans coût, aucune fonction search). |
| 04-13 | 04 Qualité de code | medium | Profils YAML et `.multiai.yaml` projet : jamais branchés au CLI (code mort) | NCV | — |
| 04-14 | 04 Qualité de code | medium | Packages onboarding, logging et openrouter entièrement morts ; marqueur first-run jamais lu | NCV | — |
| 04-15 | 04 Qualité de code | medium | `CLEAR_ENV` parsé mais jamais honoré par le lanceur Go | NCV | — |
| 04-16 | 04 Qualité de code | medium | Chaos de versions (5 sources divergentes) + injection ldflags `-X` inopérante sur une const | NCV | — |
| 04-17 | 04 Qualité de code | medium | `ShowEffectiveEnv --json` : JSON fabriqué à la main sans échappement → invalide | NCV | — |
| 04-18 | 04 Qualité de code | medium | Menu de sélection d'outil non déterministe (itération de map Go) | NCV | — |
| 04-19 | 04 Qualité de code | medium | Mutex du store chiffré neutralisé : store neuf par opération (win/darwin) | NCV | — |
| 04-20 | 04 Qualité de code | medium | `encryptedFileStore` utilise `os.Getenv("HOME")` : chemin cassé sous Windows | NCV | — |
| 04-21 | 04 Qualité de code | medium | gofmt non conforme sur 3 fichiers (jamais détecté, CI inerte) | NCV | — |
| 04-22 | 04 Qualité de code | medium | Messages d'erreur du binaire Go référençant la syntaxe PowerShell | NCV | — |
| 04-23 | 04 Qualité de code | medium | PS : `Set-ProfileSecret` non atomique (fix v0.2.1 #9 non porté en PS) | NCV | — |
| 04-24 | 04 Qualité de code | low | PS : `Get-RouterHash` retourne toujours null — hash SHA256 jamais loggé | NCV | — |
| 04-25 | 04 Qualité de code | low | Claims sécurité faux dans les README : SecureString (PS) et credential store natif (Go) | NCV | — |
| 04-26 | 04 Qualité de code | low | `LoadYAML` lit le fichier entier avant le check de taille (YAML bomb incomplète) | NCV | — |
| 04-27 | 04 Qualité de code | low | Profils `.env` corrompus ou illisibles ignorés silencieusement | NCV | — |
| 04-28 | 04 Qualité de code | low | Duplication : whitelist commandes, tri profils, masquage secrets, hooks before/after | NCV | — |
| 04-29 | 04 Qualité de code | low | PS : fallback chain déclenchée sur tout exit non-zéro, y compris Ctrl+C | NCV | — |
| 04-30 | 04 Qualité de code | low | `multiai list --json` émet `null` au lieu de `[]` pour une liste vide | NCV | — |
| 04-31 | 04 Qualité de code | low | Complétions shell avec shortcuts hardcodés déjà divergents des profils réels | NCV | — |
| 04-32 | 04 Qualité de code | low | Encodage console incohérent (ASCII strippé / accents / box-drawing, pas de codepage) | NCV | — |
| 05-01 | 05 Tests & CI/CD | high (← critical) | CI/CD entièrement inopérante : workflows hors racine du repo + branche main inexistante | PARTIAL | Cœur confirmé par l'API GitHub (aucun run CI/Release jamais enregistré) ; erreur factuelle : release.yml se déclenche sur les tags `v*`, pas sur `main` — il est inopérant uniquement par son emplacement. Sévérité recalibrée high (pas d'impact runtime direct). |
| 05-03 | 05 Tests & CI/CD | high | 7 packages sur 12 à 0 % de couverture, dont tout le chemin critique de lancement | CONFIRMED | LOC et pourcentages vérifiés au fichier près ; 60,3 % du code de production sans test, incluant whitelist, secrets, hooks, exit codes. |
| 05-04 | 05 Tests & CI/CD | high | Fix injection shell des hooks (#1 v0.2.1) sans test de régression, ordre escape→expand douteux | CONFIRMED | Exploit résiduel reproductible via valeur d'env contenant des métacaractères (ExpandEnv après échappement) ; commentaire in-code littéralement inversé ; 0 test. |
| 05-05 | 05 Tests & CI/CD | high | Pipeline de release jamais déclenché et cassé par construction | CONFIRMED | Aucun tag v0.3.0, npm 0.3.0 publié hors pipeline ; workflow de toute façon hors racine ; chemins goreleaser/npm cassés ; placeholders packaging référencent même v0.5.0. |
| 05-06 | 05 Tests & CI/CD | high | Features v0.3.0 annoncées mais absentes du binaire Go ; version bloquée à 0.2.1 | CONFIRMED | models/search/compare n'existent nulle part (ni Go ni PowerShell) ; openrouter et onboarding = code mort (0 import) ; nuance : npm ships le PowerShell, qui a bien regions/fallback/cost. |
| 05-07 | 05 Tests & CI/CD | medium | Actions GitHub épinglées par tags mutables, pas par SHA | NCV | Risque supply-chain type tj-actions ; #14 v0.2.1 persiste. |
| 05-08 | 05 Tests & CI/CD | medium | Dependabot inactif : fichier placé hors de la racine du repo | NCV | Fichier correct mais jamais lu par GitHub. |
| 05-09 | 05 Tests & CI/CD | medium | Lint job condamné : .golangci.yml v1 avec golangci-lint-action@v7 (v2) + 3 fichiers non gofmt | NCV | gofmt -l vérifié localement le 2026-07-05. |
| 05-10 | 05 Tests & CI/CD | medium | TestConfig_UpdateEnvFile est un test de façade | NCV | updateEnvFile jamais appelé par le test homonyme. |
| 05-11 | 05 Tests & CI/CD | medium | Tests Pester jamais exécutés : pas de script npm test, pas de job CI PowerShell | NCV | Le paquet npm distribué part en prod sans aucun test. |
| 05-12 | 05 Tests & CI/CD | medium | Exit code 0 sur erreur de lancement : table « Codes de sortie 0-4 » ni implémentée ni testée | NCV | runLaunch retourne nil sur erreur (main.go:276-279). |
| 05-02 | 05 Tests & CI/CD | low (comptage requalifié) | Claim « 45+ tests » : 32 Test* + 2 Benchmark* comptés | PARTIAL | Le « 45+ » devient atteignable en comptant les 18 sous-tests t.Run (50 « === RUN » mesurés) → titre requalifié ; reste vrai : couverture env annoncée 96,0 % vs 86,2 % mesurée (doc périmée). |
| 05-13 | 05 Tests & CI/CD | low | Fixes concurrence (#2) validés nulle part : pas de -race possible localement, pas de test de concurrence | NCV | Mutex présent mais jamais passé au race detector. |
| 05-14 | 05 Tests & CI/CD | low | Bugs mineurs CI/build : expression windows jamais vraie, Makefile périmé, benchmark du chemin d'échec | NCV | matrix.os == 'windows' jamais vrai ; VERSION = 0.2.0-dev. |
| 06-03 | 06 UX/DX/Docs | critical | Dépôt git sans aucun profil Go + profils PS de base manquants — first-run cassé depuis GitHub | CONFIRMED | Exit 2 sur toutes les commandes après clone+build ; aucun go:embed ni scaffolding ; aucun canal (npm installer Go, scoop, brew) n'embarque les profils |
| 06-04 | 06 UX/DX/Docs | high | Wizard onboarding écrit mais jamais appelé (code mort) | CONFIRMED | Aucun import hors du package ; marqueur `.first-run-done` écrit jamais lu ; CHANGELOG v0.2.6 le déclare livré |
| 06-05 | 06 UX/DX/Docs | high | Profils YAML, .multiai.yaml et hooks annoncés mais débranchés du binaire | CONFIRMED | `LoadDirYAML`/`FindProjectConfig`/`Hooks` uniquement appelés par les tests ou jamais ; 3 pages docs inaccessibles |
| 06-06 | 06 UX/DX/Docs | high | Features v0.3.0 (régions, fallback, cost log, erase keys, 14 fournisseurs) uniquement en PowerShell legacy | CONFIRMED | 0 occurrence FALLBACK/REGION/costs.log/erase côté Go ; `%VAR%` non expansé en Go ; profils 60-83 absents du Go |
| 06-07 | 06 UX/DX/Docs | high | Claim « Cost logging : estimation coût + cumul session » faux | CONFIRMED | `Write-CostLog` ne logge que timestamp/shortcut/exit/durée ; aucun prix, aucun cumul, aucun agrégateur |
| 06-08 | 06 UX/DX/Docs | high | Badges de score auto-proclamés (9.5/10 et 10/10) sur un projet audité à 5.5/10 | CONFIRMED | Badges statiques à lien vide, contredits par audit/07 (5.5/10, 3 critiques ouverts) et contradictoires entre eux |
| 06-09 | 06 UX/DX/Docs | high | `go install github.com/lrochetta/multiai@latest` cassé (module dans un sous-dossier) | CONFIRMED | go.mod dans `multiai-go/` avec chemin de module racine + package main dans `cmd/multiai` : double échec ; documenté à 8+ endroits |
| 06-01 | 06 UX/DX/Docs | medium (← critical) | Commandes `multiai models`/`search`/`compare` annoncées mais inexistantes | PARTIAL | Faits confirmés dans les deux implémentations (openrouter/client.go = dead code sans search/compare) ; sévérité recalibrée : échec propre, ni sécurité ni perte de données — écart docs/implémentation sur package publié |
| 06-02 | 06 UX/DX/Docs | low (← critical) | Binaire Go déclaré v0.2.1 alors que la release est v0.3.0 | PARTIAL | Faits confirmés (+ bug aggravant : `version` est un const, l'injection ldflags de goreleaser/Makefile/AUR/brew est inopérante) ; mais le binaire Go n'est pas distribué dans la release npm 0.3.0 et aucun tag v0.3.0 n'existe → drift cosmétique d'un composant non shippé |
| 06-10 | 06 UX/DX/Docs | medium | Mode --json non scriptable : sortie humaine mêlée au JSON, JSON fabriqué non échappé | NCV | — |
| 06-11 | 06 UX/DX/Docs | medium | Messages d'erreur Go utilisant la syntaxe de flags PowerShell | NCV | — |
| 06-12 | 06 UX/DX/Docs | medium | `multiai config --provider` documenté mais non implémenté | NCV | — |
| 06-13 | 06 UX/DX/Docs | medium | Docs VitePress hors dépôt, non déployées, périmées, flags inexistants documentés | NCV | — |
| 06-14 | 06 UX/DX/Docs | medium | Exit codes 0-4 documentés mais non respectés (Go et PS) | NCV | — |
| 06-15 | 06 UX/DX/Docs | medium | Aucune option Quitter dans les menus interactifs Go et PowerShell | NCV | — |
| 06-16 | 06 UX/DX/Docs | medium | Typographie Go incohérente : em dash/box-drawing réintroduits, accents mélangés | NCV | — |
| 06-17 | 06 UX/DX/Docs | medium | Claim « Fusion — panel d'experts avec synthèse automatique » sans aucun code | NCV | — |
| 06-18 | 06 UX/DX/Docs | low | Fichiers parasites à la racine, script interne tracké, clé API sur disque | NCV | — |
| 06-19 | 06 UX/DX/Docs | low | ROADMAP.md obsolète et en contradiction avec le CHANGELOG | NCV | — |
| 06-20 | 06 UX/DX/Docs | low | Complétion shell figée sur 18 shortcuts, sans les profils v0.3.0 | NCV | — |
| 06-21 | 06 UX/DX/Docs | low | Chiffres gonflés dans le README (tests, sous-commandes, version Go) | NCV | — |
| 07-03 | 07 Sécurité | high | Intégration credential-store cassée : marqueur `__MULTIAI_CREDSTORE__` transmis comme clé API | CONFIRMED | Chaîne complète vérifiée (wizard.go:269 → dotenv.go:73-93 → launcher.go:76,124) : aucune relecture du store, échec d'auth garanti après `multiai config`. |
| 07-04 | 07 Sécurité | high | Installeur npm sans vérification checksum/signature | CONFIRMED | install.js:26-87 : aucun hash ni Cosign consommé alors que goreleaser les produit ; vecteur release compromise atteignable, MITM atténué par TLS. |
| 07-06 | 07 Sécurité | high | CI jamais déclenchée (trigger `main` vs branche `master`) | CONFIRMED | Pire que décrit : le workflow vit dans `multiai-go/.github/workflows/` (sous-répertoire, jamais lu par GitHub Actions) ; corriger la branche ne suffirait pas. |
| 07-01 | 07 Sécurité | high (← critical) | Master key AES-256 en clair à côté du ciphertext | PARTIAL | Faits confirmés sur les 3 OS (stores tous fallback fichier, DeriveKey jamais appelé), mais modèle d'attaque = lecture du home, équivalent baseline `~/.ssh` à 0600 → sévérité recalibrée CRITICAL→HIGH ; « security theater » réel. |
| 07-02 | 07 Sécurité | medium (corrigée) | Credential stores « natifs » = stubs (claim README faux) | PARTIAL | Stubs confirmés (aucun wincred/Keychain/libsecret), mais le détail « base64 non chiffré » est réfuté : le fichier est bien chiffré AES-256-GCM au repos ; claim README trompeur, pas de secrets en clair. |
| 07-05 | 07 Sécurité | low (← high) | Checksums packaging en placeholder, CHANGELOG « faussement corrigé » | PARTIAL | Placeholders confirmés, mais l'accusation de mensonge CHANGELOG est réfutée (CHANGELOG.md:77 annonce « Placeholder honnête ») ; fichiers = templates, goreleaser génère les vrais manifestes brew/scoop → impact faible. |
| 07-07 | 07 Sécurité | medium | Échappement hooks PowerShell/pwsh incomplet → injection latente | NCV | Atténuant : feature non câblée (opts.Hooks toujours nil). |
| 07-08 | 07 Sécurité | medium | Clé DeepSeek réelle en clair dans l'arborescence | NCV | Gitignorée, absente de l'historique ; rotation recommandée. |
| 07-09 | 07 Sécurité | medium | `--allow-custom-command` bypass whitelist sans validation | NCV | Persiste depuis v0.2.1 (#15). |
| 07-10 | 07 Sécurité | medium | PBKDF2 implémenté mais jamais utilisé (code mort) | NCV | Correctif CHANGELOG « CWE-916 » inopérant sur le chemin réel. |
| 07-11 | 07 Sécurité | medium | Cacophonie de versions (0.2.1 / 0.3.0 / 0.5.0) | NCV | install.js pointe vers des assets v0.5.0 potentiellement inexistants. |
| 07-12 | 07 Sécurité | medium | Fonctionnalités annoncées non câblées (code mort étendu) | NCV | models/search/compare, hooks, onboarding, config projet : absents du flux réel. |
| 07-13 | 07 Sécurité | low | Lecture de réponse HTTP OpenRouter non bornée | NCV | Code non câblé aujourd'hui ; borner avant exposition. |
| 07-14 | 07 Sécurité | low | Dossier secrets relatif au cwd si HOME absent (Windows) | NCV | Utiliser `os.UserHomeDir()`. |
| 07-15 | 07 Sécurité | low | Actions GitHub non épinglées par SHA + `@latest` flottants | NCV | Dependabot atténue partiellement. |
| 07-16 | 07 Sécurité | low | Étape npm-publish de release cassée (chemin hors checkout) | NCV | `cd ../../multiai-powershell` invalide dans le runner. |

---

## 3. Bilan chiffré (recompté depuis les 7 tableaux)

| Dimension | Findings | CONFIRMED | PARTIAL | REFUTED | Non contre-vérifiés |
|---|---:|---:|---:|---:|---:|
| 01 — Produit & fonctionnalités | 16 | 7 | 2 | 0 | 7 |
| 02 — Distribution & packaging | 18 | 7 | 4 | 0 | 7 |
| 03 — Architecture | 18 | 5 | 4 | 0 | 9 |
| 04 — Qualité de code | 32 | 8 | 4 | 0 | 20 |
| 05 — Tests & CI/CD | 14 | 4 | 2 | 0 | 8 |
| 06 — UX / DX / Documentation | 21 | 7 | 2 | 0 | 12 |
| 07 — Sécurité | 16 | 3 | 3 | 0 | 10 |
| **Total** | **135** | **41** | **21** | **0** | **73** |

Détail complémentaire :

- **62 findings contre-vérifiés** (41 CONFIRMED + 21 PARTIAL) ; le cap de 14 par dimension n'a jamais été atteint (maximum : 12 en dimension 04).
- **0 finding écarté** : aucun réfutateur n'a obtenu un REFUTED intégral.
- Sur les 21 PARTIAL : **19 recalibrations de sévérité à la baisse** (dont 15 des 20 criticals initiaux contre-vérifiés, rétrogradés en high, medium ou low), **1 sévérité maintenue** avec détail factuel réfuté (03-05), **1 requalification de portée** sans changement de sévérité (05-02).
- Les **5 criticals survivants** après contre-vérification (01-02, 03-01, 03-04, 04-01, 06-03) sont tous CONFIRMED — trois d'entre eux décrivent le même défaut racine (sentinelle `__MULTIAI_CREDSTORE__` write-only) vu par trois dimensions indépendantes, ce qui vaut triple confirmation croisée.

### Lecture — fiabilité de l'audit

Zéro REFUTED sur 62 tentatives de réfutation à charge : les auditeurs sont factuellement précis — aucun finding ne repose sur un fait faux, et plusieurs réfutateurs ont même trouvé des éléments *aggravants* (02-05, 02-08, 03-06, 07-06). En revanche, un tiers des findings vérifiés (21/62) a vu sa sévérité ou sa portée recalibrée à la baisse, presque toujours de critical vers high/medium : les auditeurs sur-cotent l'impact, en particulier en confondant « cassé par construction » et « exploité en production » (le canal npm PS, seul canal vivant, échappe à la plupart des pannes Go). Limite structurelle de la méthode : 73 findings sur 135 (54 %), tous medium/low, n'ont subi aucune contre-vérification par conception — leur exactitude est plausible par contagion (les auditeurs se sont montrés fiables sur le vérifié) mais reste non prouvée. Enfin, la contre-vérification teste la véracité des findings émis, pas l'exhaustivité de l'audit : un faux négatif (défaut manqué) resterait invisible à ce dispositif.
