# Audit produit & fonctionnalites — multiai

**Date :** 2026-07-14

**Auditeur :** Atlas (BMAD+ Strategist — Analyste puis Product Manager)

**Perimetre :** proposition de valeur, personas/JTBD, parcours CLI, couverture fonctionnelle, coherence documentation-code, distribution, adoption et roadmap produit.

**Revision auditee :** `5808769` (`master`, egalement `origin/master` au debut de l'audit).
**Score produit/fonctionnalites : 6,1/10.**

> Verdict : le coeur de multiai est devenu un produit reel et differenciant, mais le premier contact n'est pas fiable. Le routeur local, les 37 profils, l'isolation d'environnement, les stores natifs, les fallbacks et la decouverte OpenRouter forment une excellente base. En revanche, l'installation Windows peut annoncer un succes sans rendre `multiai` invocable, plusieurs canaux declares disponibles ne le sont pas, le registre communautaire pointe vers un depot absent, et des contrats documentes (YAML, projet, hooks, JSON, timeout) ne correspondent pas au code. Il faut d'abord rendre chaque promesse executable avant d'accelerer la croissance.

---

## 1. Methode et limites

- Inspection du depot, des surfaces utilisateur, des tests de contrat npm, de la documentation et de l'historique Git local.
- Etat public verifie avec `npm view multiai version engines dist-tags --json`, `git ls-remote` sur les depots annonces et les tags distants.
- Benchmark limite a des sources officielles : npm, Go, OpenCode, OpenRouter, LiteLLM et le depot officiel de Claude Code Router.
- Les changements deja presents dans le worktree ont ete conserves. Au debut de l'audit, `git status --short --branch` signalait notamment des modifications dans la memoire projet, `CHANGELOG.md`, des tests Go et deux guides.
- Cet audit ne remplace ni l'audit architecture/code ni l'audit securite executes en parallele. Il juge la valeur livree et les contrats exposes aux utilisateurs.

### Echelle de severite

| Niveau | Definition produit |
|---|---|
| **BLOQUANT** | Empeche l'activation ou le parcours principal d'un segment majeur. |
| **HAUTE** | Fonction phare indisponible, contrat public faux, ou automatisation non fiable. |
| **MOYENNE** | Friction importante, dette de coherence ou frein a l'adoption. |
| **BASSE** | Opportunite d'amelioration sans rupture immediate du parcours principal. |

---

## 2. Description claire du projet

### Definition factuelle

**multiai est un plan de controle local pour les CLI de developpement assiste par IA.** Il lance Claude Code, Codex CLI ou OpenCode avec un profil qui choisit le fournisseur, le modele, les arguments et les variables d'environnement. Les secrets sont resolus depuis un credential store, l'environnement enfant est isole par liste blanche, et le processus reste celui du CLI d'origine.

Le produit n'est donc ni un modele, ni un agent de code, ni un proxy LLM generaliste. C'est la couche locale qui relie :

```text
developpeur -> multiai -> profil/politique locale -> CLI natif -> fournisseur/modeles
```

La proposition actuelle est visible dans `README.md:1-3`. Le coeur autorise explicitement les trois commandes `claude`, `codex` et `opencode` (`multiai-go/internal/cli/launcher.go:21-31`), charge les profils embarques dans le repertoire utilisateur (`multiai-go/cmd/multiai/main.go:40-76`) et construit un environnement nettoye (`multiai-go/internal/env/env.go:83-108`).

### Proposition de valeur recommandee

> **Une commande pour lancer le bon agent de code sur le bon fournisseur, avec les bons secrets — isoles localement.**

Version internationale :

> **Run the right coding agent on the right provider, with locally isolated credentials — in one command.**

Cette formulation vend le resultat — choix, securite et vitesse — plutot que le mecanisme de « routeur multi-IA ».

### Positionnement a assumer

- **Categorie :** local-first AI coding CLI control plane.
- **Promesse principale :** une politique de lancement portable entre plusieurs CLI, sans export global de secrets.
- **Differenciateur :** combinaison multi-CLI + profils + credential store OS + isolation par processus + fallback de session.
- **Complementarite :** multiai orchestre localement OpenRouter ou LiteLLM ; il ne doit pas pretendre remplacer leur routage par requete.
- **Non-objectifs conseilles :** devenir un proxy LLM complet, une interface de chat ou un orchestrateur d'agents generaliste.

---

## 3. Personas et Jobs-to-be-Done

Ces personas sont des **hypotheses produit derivees du code et de la documentation**, pas le resultat d'entretiens utilisateurs. Il faut les valider avant d'investir dans une croissance large.

| Priorite | Persona | Job-to-be-Done | Resultat attendu | Preuve de besoin dans le produit |
|---|---|---|---|---|
| P1 | Developpeur « poly-CLI » | « Quand je change d'agent ou de fournisseur, je veux lancer le bon contexte sans reconfigurer mon shell. » | Un raccourci stable et un lancement en moins de 10 secondes. | Profils et choix d'outil : `multiai-go/internal/profile/profile.go:13-36`, `multiai-go/internal/menu/interactive.go`. |
| P1 | Developpeur sensible aux secrets | « Je veux eviter qu'une cle destinee a un fournisseur fuite dans un autre CLI. » | Aucun secret global, valeurs masquees, stockage OS. | Isolation : `multiai-go/internal/env/env.go:10-24,83-108`; resolution du store : `multiai-go/internal/cli/launcher.go:300-331`. |
| P2 | Lead/plateforme d'equipe | « Je veux versionner une politique de projet sans versionner les cles. » | Schema valide, comportement deterministe, onboarding reproductible. | Intention `.multiai.yaml` : `multiai-go/internal/profile/project.go:12-73`; contrat actuellement deficient, voir P-04. |
| P2 | Developpeur cout/resilience | « Je veux choisir un modele selon prix, contexte ou disponibilite et basculer en cas d'echec. » | Comparaison explicable et fallback controle. | OpenRouter : `multiai-go/cmd/multiai/cmd_openrouter.go:182-299`; fallback : `multiai-go/internal/cli/fallback.go:10-77`. |
| P3 | Auteur de profils/mainteneur | « Je veux publier, valider, installer et mettre a jour un profil de confiance. » | Cycle add/validate/publish/install/remove complet. | Registre et installation : `multiai-go/cmd/multiai/cmd_registry.go:397-499`; endpoint absent, voir P-03. |
| Niche | Utilisateur BMAD+ | « Je veux gerer BMAD+ sans quitter mon routeur. » | Integration contextuelle, sans encombrer le produit generique. | Menu BMAD+ : `multiai-go/internal/i18n/i18n.go:105-110`; commande : `multiai-go/cmd/multiai/main.go:321-324`. |

### Besoin primaire a optimiser

Le parcours gagnant est : **installer -> verifier -> configurer une cle -> lancer un profil -> relancer le meme profil demain**. Toutes les autres fonctions sont secondaires tant que ce funnel n'est pas mesurable et fiable.

---

## 4. Parcours CLI actuel

| Etape | Experience actuelle | Evaluation | Preuve |
|---|---|---:|---|
| Decouverte | README riche, pitch comprehensible, mais beaucoup de badges et de claims avant la preuve. | 6/10 | `README.md:1-30,82-103`. |
| Installation npm | Telecharge et verifie le binaire, puis lance une installation npm globale. Sous Windows, ne corrige pas le PATH. | 3/10 | `multiai-go/packaging/npm/install.js:1-12,163-195`; `multiai-go/packaging/npm/bin/multiai.js:47-87`. |
| Premier demarrage | Profils materialises automatiquement et wizard propose la configuration. | 8/10 | `multiai-go/cmd/multiai/main.go:84-100,346-353`; `multiai-go/internal/onboarding/wizard.go:53-99`. |
| Configuration | Catalogue de 13 fournisseurs, propagation d'une cle a plusieurs profils, stores OS/fichier. | 8/10 | `multiai-go/internal/catalog/providers.yaml:39-192`; `multiai-go/internal/config/wizard.go:98-209,283-309`. |
| Selection/lancement | Menu outil -> profil, lancement direct, isolation, arguments, hooks et fallback. | 8/10 | `multiai-go/cmd/multiai/main.go:395-472`; `multiai-go/internal/cli/launcher.go:65-217`. |
| Automatisation | `list/models/search/compare/update/profile/migrate` proposent du JSON, mais `launch --json` et le parsing de flags ne sont pas des contrats fiables. | 4/10 | `multiai-go/cmd/multiai/main.go:443-470`; constats P-05/P-06. |
| Extension | YAML, projet, hooks, registre et profils OpenRouter existent en code, mais leurs contrats sont incomplets ou incompatibles avec les guides. | 4/10 | `multiai-go/internal/profile/yaml.go`; constats P-03/P-04/P-13. |
| Support/diagnostic | Guides de depannage, mais pas de `doctor`, `paths`, `history` ni verificateur global. | 4/10 | Commandes enregistrees : `multiai-go/cmd/multiai/cmd_*.go`; recherche `rg -n "doctor|diagnos" multiai-go/cmd multiai-go/internal` sans commande produit. |

---

## 5. Couverture fonctionnelle reelle

### Capacites validees

| Capacite | Etat | Preuve |
|---|---|---|
| 3 CLI autorises | Livre | `multiai-go/internal/cli/launcher.go:21-31`. |
| 37 profils embarques, 15 Claude + 7 Codex + 15 OpenCode | Livre | Les 37 fichiers sous `multiai-go/internal/assets/profiles/`; le catalogue comptabilise les 37 shortcuts a `multiai-go/internal/catalog/providers.yaml:192-193`. |
| 13 fournisseurs configurables | Livre | `multiai-go/internal/catalog/providers.yaml:39-190`. |
| Isolation de l'environnement par liste blanche | Livre | `multiai-go/internal/env/env.go:10-24,83-108`. |
| Credential stores WinCred, Keychain, libsecret + fallback chiffre | Livre dans le code | `multiai-go/internal/secret/secret.go:126-149`; implementations `store_windows.go`, `store_darwin.go`, `store_linux.go`. |
| Fallback de profils au niveau processus | Livre | `multiai-go/internal/cli/fallback.go:10-77`. |
| Decouverte OpenRouter : models/search/compare, cache/offline | Livre | Enregistrement : `multiai-go/cmd/multiai/cmd_openrouter.go:21-24`; handlers : `:182-299`. |
| Onboarding, update explicite et migration PowerShell | Livre | `multiai-go/cmd/multiai/main.go:346-353`; `cmd_update.go:18-131`; `cmd_migrate.go:14-106`. |
| Sorties JSON pour listes et decouverte | Livre | `multiai-go/internal/cli/display.go:33-58`; `multiai-go/cmd/multiai/cmd_openrouter.go:110-132`. |

### Maturite par domaine

| Domaine | Note | Risque dominant |
|---|---:|---|
| Coeur launch/config/isolation | 8,0/10 | Contrats de flags et projet. |
| Catalogue fournisseurs/profils | 8,0/10 | Documentation de profils obsolete. |
| Installation/distribution | 3,0/10 | PATH Windows et canaux annonces absents. |
| Automatisation/JSON | 4,0/10 | Sortie enfant melangee au JSON, options ignorees. |
| Extensibilite projet/YAML/hooks | 4,0/10 | Schema documente incompatible et champs ignores. |
| OpenRouter/model discovery | 7,5/10 | Pas encore relie a une politique de selection explicable. |
| Registre communautaire | 1,0/10 | Depot par defaut inexistant. |
| Documentation/aide/completion | 3,0/10 | Plusieurs generations de contrats contradictoires. |
| Internationalisation | 4,0/10 | Traduction partielle et sorties mixtes. |
| Feedback produit/diagnostic | 3,0/10 | Pas de doctor ni de funnel d'activation mesurable. |

---

## 6. Benchmark officiel et choix strategique

| Reference | Capacite officielle pertinente | Implication pour multiai |
|---|---|---|
| [OpenCode — Providers](https://opencode.ai/docs/providers/) | OpenCode sait deja connecter de nombreux fournisseurs, gerer OAuth/headless et declarer des fournisseurs compatibles OpenAI. | Pour un utilisateur OpenCode seul, « plus de fournisseurs » ne suffit pas. La valeur de multiai doit etre **cross-CLI, politique et isolation**. |
| [Claude Code Router](https://github.com/musistudio/claude-code-router) | Routage par tache/modele, transformations requete/reponse, changement dynamique et plugins autour de Claude Code. | multiai doit nommer clairement son fallback **au niveau session/processus** et gagner par sa portabilite multi-CLI, pas imiter un proxy mono-CLI. |
| [OpenRouter — Model fallbacks](https://openrouter.ai/docs/guides/routing/model-fallbacks) et [provider routing](https://openrouter.ai/docs/guides/routing/provider-selection) | Fallback par requete, ordre des fournisseurs et criteres de latence/debit. | multiai peut devenir le plan de controle local qui genere ou applique ces politiques, sans recreer leur infrastructure. |
| [LiteLLM — documentation officielle](https://docs.litellm.ai/) | Proxy/gateway unifie avec routage, retries et fallbacks entre deployments. | Positionner LiteLLM comme backend compatible renforce multiai ; le produit doit rester leger et sans daemon. |
| [npm — Folders](https://docs.npmjs.com/files/folders) | Sous Windows, les executables globaux sont lies directement dans `{prefix}` et ce chemin doit etre present dans `PATH`. | Le correctif PATH doit utiliser le **prefix npm**, pas `npm root --global`, et le verifier dans un nouveau processus. |
| [Go Modules Reference](https://go.dev/ref/mod) | Le chemin de module est le prefixe des packages et le module root est le dossier contenant `go.mod`. | Le monorepo actuel doit aligner emplacement, module path, package public et tags avant de promettre `go install`. |

**Inference Atlas :** l'espace defensable n'est pas « le routeur LLM le plus riche ». C'est **le plan de controle local de reference pour tous les agents de code en CLI** : profils portables, secrets OS, politiques projet, diagnostic et compatibilite avec les gateways existantes.

---

## 7. Constats detailles

### P-01 — BLOQUANT — L'installation npm Windows ne rend pas toujours `multiai` accessible

**Preuves**

- `buildGlobalInstallArgs` lance bien `npm install --global` (`multiai-go/packaging/npm/bin/multiai.js:20-28`).
- Apres installation, le code demande `npm root --global`, construit directement le chemin du module et smoke-teste `node <module>/bin/multiai.js --version` (`multiai-go/packaging/npm/bin/multiai.js:58-83`). Ce test contourne totalement la resolution par `PATH`.
- Le message de succes se contente de « Open a new terminal » (`multiai-go/packaging/npm/bin/multiai.js:86`) ; aucune lecture ni mise a jour du PATH utilisateur n'existe dans le fichier.
- Les tests couvrent arguments, prefix et invocation npm, jamais la presence/deduplication du PATH ni la resolution de `multiai` (`multiai-go/packaging/npm/bin/multiai.test.js:15-74`).
- `git diff 141120b^ 141120b -- multiai-go/packaging/npm/bin/multiai.js multiai-go/packaging/npm/bin/multiai.test.js` montre que ce faux smoke test a ete introduit avec la restauration du contrat `npx ... install`.
- npm confirme officiellement que, sous Windows, le shim global est place directement dans `{prefix}` et que ce chemin doit etre dans `PATH` : [npm Folders](https://docs.npmjs.com/files/folders).

**Impact**

Le parcours recommande peut annoncer une installation reussie alors que la commande suivante du README echoue avec « commande introuvable ». C'est une rupture d'activation P0, particulierement grave car le test interne donne un faux sentiment de succes.

**Recommandation / criteres d'acceptation**

1. Obtenir le prefix reel avec l'equivalent de `npm prefix --global`, en preservant un eventuel `--prefix` personnalise.
2. Calculer `binDir = prefix` sous Windows et `binDir = prefix/bin` sous Unix.
3. Sous Windows, lire le PATH utilisateur, normaliser les chemins, dedupliquer sans tenir compte de la casse ni des slashs, puis ajouter `binDir` de facon idempotente via PowerShell/.NET (`[Environment]::SetEnvironmentVariable`) ; ne pas utiliser `setx`, qui peut tronquer ou developper la valeur.
4. Ne jamais ecraser le PATH existant. Si l'ecriture echoue, afficher une commande manuelle exacte et retourner un echec d'installation explicite.
5. Verifier dans un **nouveau processus** que `multiai --version` se resout par son nom, et non par un chemin interne.
6. Couvrir : PATH vide, deja present avec casse/slash differents, prefix avec espaces/Unicode, prefix custom, absence de droits, valeur longue, et nouvelle console PowerShell/cmd.

### P-02 — HAUTE — La matrice de distribution declare des canaux indisponibles

**Preuves**

- Le README marque Go, Homebrew, Scoop, APT, AUR et script comme disponibles (`README.md:86-92`) et les recompte comme distribution livree (`README.md:288`).
- GoReleaser indique au contraire que Homebrew et Scoop sont desactives et leurs sections sont commentees (`multiai-go/.goreleaser.yaml:115-148`).
- Le PKGBUILD AUR est reste en `0.4.0` avec `sha256sums=('SKIP')` (`multiai-go/packaging/aur/PKGBUILD:16-27`).
- `git ls-remote https://github.com/lrochetta/homebrew-tap.git HEAD` et la meme commande sur `scoop-bucket.git` repondent `Repository not found`.
- `git ls-remote origin refs/heads/gh-pages` ne retourne aucune branche, alors que le README promet un depot APT GitHub Pages.
- A la date de l'audit, `npm view multiai version engines dist-tags --json` renvoie `latest: 0.6.6` et `node >=18`, tandis que le package local cible `0.6.7` et Node `>=24.14.0` (`multiai-go/packaging/npm/package.json:3,40-42`). `git ls-remote origin refs/tags/v0.6.7 refs/tags/v0.6.6` ne retourne que `v0.6.6`.

**Impact**

Les utilisateurs choisissent une methode selon leur gestionnaire de paquets, rencontrent un depot absent, puis perdent confiance dans toutes les autres promesses. Les guides melangent en plus l'etat public 0.6.6 et le correctif local 0.6.7 non publie.

**Recommandation / criteres d'acceptation**

- Remplacer immediatement chaque coche par `disponible`, `beta` ou `planifie`, determine par un smoke test public.
- Ne publier 0.6.7 qu'apres la matrice CI complete verte, conformement a `.agents/memory/decisions.md`.
- Creer effectivement tap, bucket et branche APT, ou retirer les commandes jusqu'a leur disponibilite.
- Ajouter une CI quotidienne qui installe depuis **chaque canal public**, dans un environnement vierge, puis execute `multiai version`, `multiai list --json` et `multiai launch -p co --dry-run --json`.
- Faire du README le reflet genere de cette matrice, pas une declaration manuelle.

### P-03 — HAUTE — Le registre communautaire pointe vers un depot inexistant

**Preuves**

- Le client utilise `https://raw.githubusercontent.com/lrochetta/profiles-multiai/main/index.json` (`multiai-go/internal/registry/client.go:15-22`) et construit les telechargements dans le meme depot (`:134-143`).
- Le README promeut `profile search`, `profile install` et les contributions vers ce registre (`README.md:120-121,306`).
- `git ls-remote https://github.com/lrochetta/profiles-multiai.git HEAD` repond `Repository not found`.
- Le code des commandes est pourtant expose (`multiai-go/cmd/multiai/cmd_registry.go:21-27,397-499`).

**Impact**

Deux commandes phares et le principal volant communautaire echouent par construction. Les auteurs potentiels ne peuvent ni soumettre ni installer un profil.

**Recommandation / criteres d'acceptation**

- Soit creer le depot public avec index versionne, schema, exemples, gouvernance, checksums et CI ; soit masquer les commandes et le claim jusqu'a sa disponibilite.
- Ajouter `multiai profile registry status` ou integrer ce controle a `multiai doctor`.
- Refuser un index invalide, imposer HTTPS et checksum, afficher source/version/date de l'index.
- KPI de sortie : recherche et installation reelles depuis une machine vierge, au moins trois profils maintenus hors du coeur et un processus de contribution documente de bout en bout.

### P-04 — HAUTE — Les schemas YAML, projet et hooks documentes ne correspondent pas a l'implementation

**Preuves**

- Le guide promet un fichier unique `~/.multiai/profiles.yaml` contenant une map `profiles:` et plusieurs profils (`multiai-go/docs/advanced/yaml-profiles.md:5-40,42-86`).
- Le chargeur reel parcourt des fichiers `.yaml/.yml` individuels dans le repertoire des profils et decode chacun directement dans un seul `ProfileYAML` (`multiai-go/internal/profile/yaml.go:15-37,77-117`).
- Le guide promet interpolation `${VAR}`, priorite YAML et hooks sous forme de chaines (`multiai-go/docs/advanced/yaml-profiles.md:88-102,137-146`). Le code n'expanse que les references `%VAR%` des profils ordinaires (`multiai-go/internal/env/env.go:31-81`) ; `HooksConfig` attend des listes de `HookCommand`, pas des chaines (`multiai-go/internal/profile/yaml.go:40-48`).
- Le guide projet utilise `project.default_profile` et `profiles.<name>.extends` (`multiai-go/docs/advanced/project-config.md:15-45,60-82,115-154`). Le decodeur lit un unique `ProfileYAML`, puis `MergeProjectConfig` n'applique que `display_name`, `overrides`, `clear_env`, `args` et `hooks` (`multiai-go/internal/profile/project.go:12-38,50-75`). `Extends` est declare (`multiai-go/internal/profile/yaml.go:33`) mais jamais resolu en production.
- Les decodeurs YAML ne demandent pas `KnownFields(true)` (`multiai-go/internal/profile/yaml.go:69-70`; `project.go:32-33`) : des champs documentes mais inconnus peuvent etre ignores silencieusement.

**Impact**

Une equipe peut croire qu'un profil, un hook VPN ou une politique projet est actif alors qu'il est ignore. La fonction d'extensibilite la plus strategique est donc a la fois difficile a utiliser et dangereuse par son silence.

**Recommandation / criteres d'acceptation**

1. Choisir et versionner **un schema v1 unique** ; publier son JSON Schema.
2. Activer le rejet des champs inconnus et ajouter `multiai profile validate <file>` / `multiai project validate`.
3. Implementer reellement `extends`, profil par defaut, merge de hooks et interpolation, ou retirer ces promesses.
4. Fournir une migration automatique depuis les schemas deja documentes.
5. Tester tous les exemples de documentation comme fixtures executables.
6. Pour les hooks de securite, un schema invalide doit bloquer le lancement avec un message explicite, jamais etre ignore.

### P-05 — HAUTE — Les options inconnues sont ignorees et `--timeout` est une fausse garantie

**Preuves**

- Le README recommande `multiai launch -p or-fusion --timeout 120s` (`README.md:112`).
- Aucun timeout de lancement n'existe dans `LaunchOptions` (`multiai-go/internal/cli/launcher.go:34-47`) ni dans `runLaunch`, qui ne lit qu'une liste de drapeaux connus (`multiai-go/cmd/multiai/main.go:437-449`).
- Le parsing repose sur `hasFlag` / `getFlagValue` sans validation des arguments restants (`multiai-go/cmd/multiai/main.go:505-540`). Une option inconnue avant `--` est donc silencieusement ignoree.
- La reference documente egalement `-n`, `--verbose` et `-v` (`multiai-go/docs/reference/commands.md:42-50`), absents du code.

**Impact**

Un utilisateur ou un script pense avoir impose une limite de temps ou un mode verbeux alors que le comportement n'a pas change. Une option de surete ignoree silencieusement est pire qu'une erreur.

**Recommandation / criteres d'acceptation**

- Introduire un parseur strict par sous-commande ; toute option inconnue avant `--` doit produire exit 2 et une aide ciblee.
- Implementer un vrai timeout avec propagation/termination propre du processus enfant, ou retirer immediatement le claim.
- Generer l'aide, les completions et la reference depuis les memes definitions de flags.
- Ajouter des snapshots de contrat pour chaque commande et des tests de faute de frappe (`--timeot`, valeur manquante, doublons, `--`).

### P-06 — HAUTE — `launch --json` ne garantit pas un flux JSON valide

**Preuves**

- L'aide promet « Lancement + sortie JSON » (`multiai-go/cmd/multiai/main.go:124-165`) et le README presente JSON comme integration scriptable (`README.md:203-207`).
- Meme en mode JSON, le processus enfant herite de `os.Stdout` et `os.Stderr` (`multiai-go/internal/cli/launcher.go:157-165`).
- Apres la fin du CLI enfant, le resultat JSON est encode sur ce meme stdout (`multiai-go/cmd/multiai/main.go:467-471`). Toute sortie du CLI precede donc le document JSON.
- Les erreurs de prevalidation restent du texte humain dans `main.go:266-270`; la fonction `jsonError` existe mais n'est pas appelee (`multiai-go/internal/cli/launcher.go:295-298`).

**Impact**

`jq` et les integrations CI ne peuvent pas traiter de facon fiable `launch --json`. Le contrat varie selon que le CLI enfant ecrit ou non sur stdout et selon l'endroit de l'echec.

**Recommandation / criteres d'acceptation**

- Definir explicitement deux usages incompatibles : TTY interactif et resultat machine.
- Option minimale sure : autoriser `--json` uniquement avec `--dry-run` ou `--no-launch`, et proposer `--report-json <fichier>` pour le rapport post-session d'un CLI interactif.
- Si la capture complete est voulue, fournir un mode non-TTY distinct avec champs `child_stdout`/`child_stderr`, limites de taille et codes documentes.
- Toute branche succes/erreur d'une commande JSON doit produire exactement un document JSON sur stdout ; diagnostics sur stderr.

### P-07 — HAUTE — La documentation de reference decrit une ancienne generation du produit

**Preuves**

- Le guide annonce 17 profils et des shortcuts tels que `za`, `son40`, `oa5`, `codex45`, `or`, `oc4` (`multiai-go/docs/guide/profiles.md:1-23`), alors que le produit embarque 37 profils et que le catalogue les comptabilise (`multiai-go/internal/catalog/providers.yaml:192-193`).
- Le guide « premiers pas » renvoie encore vers « les 17 profils » (`multiai-go/docs/guide/getting-started.md:52-57`).
- La reference des commandes ne liste que launch/list/config/completion/version/help (`multiai-go/docs/reference/commands.md:13-22`), alors que models/search/compare/update/profile/migrate sont enregistrees (`multiai-go/cmd/multiai/cmd_openrouter.go:21-24`, `cmd_update.go:18-20`, `cmd_registry.go:21-23`, `cmd_migrate.go:14-16`).
- L'exemple JSON documente expose `name/provider/model` (`multiai-go/docs/reference/commands.md:111-131`), mais `list --json` renvoie `tool/shortcut/display_name/description/command/args` (`multiai-go/internal/cli/display.go:33-58`).
- Le guide dit que le script Unix installe dans `/usr/local/bin` (`multiai-go/docs/guide/installation.md:39-43`), tandis que le script utilise par defaut `$HOME/.local/bin` et ne fait qu'avertir si ce dossier est absent du PATH (`multiai-go/scripts/install.sh`, variables `INSTALL_DIR` et section `PATH check`).

**Impact**

Les utilisateurs copient des commandes qui echouent, les integrateurs codent contre un JSON inexistant et le support doit expliquer quelle page est vraie. Cette dette neutralise une partie substantielle des fonctions deja livrees.

**Recommandation / criteres d'acceptation**

- Declarer le CLI et le manifeste de profils comme sources de verite et generer tableaux, exemples JSON et completions.
- Ajouter un job `docs-contract` qui execute chaque bloc de commandes non destructif et compare les schemas JSON.
- Archiver ou supprimer les pages dupliquees (`troubleshooting.md` existe a deux emplacements, tout comme les guides hooks/projet).
- Bloquer une release si un shortcut documente n'existe pas ou si un profil embarque n'apparait pas dans la reference generee.

### P-08 — MOYENNE — La « source unique » de version ne l'est pas

**Preuves**

- `main.go` affirme que `version` est la source unique, puis fixe `0.6.0` (`multiai-go/cmd/multiai/main.go:28-31`).
- Le package npm local est `0.6.7` (`multiai-go/packaging/npm/package.json:3`) ; le dernier tag public est `v0.6.6` (commande `git tag --list --sort=-version:refname`).
- Le README presente toujours l'implementation comme v0.6.0 (`README.md:100-103,247`).
- Seuls les builds GoReleaser remplacent la valeur par ldflags (`multiai-go/.goreleaser.yaml:39-53`) ; une compilation manuelle ou un eventuel `go install` affiche la valeur obsolete.

**Impact**

Les diagnostics, l'auto-update et les demandes de support peuvent rapporter une version differente du code execute. La documentation ne permet pas de distinguer public, cible et snapshot.

**Recommandation / criteres d'acceptation**

- Introduire un manifeste de release unique ou deriver la version de `debug.ReadBuildInfo`, avec un fallback `dev` explicite.
- Verifier en CI l'egalite tag/package npm/changelog/docs/artefacts.
- Afficher `version`, `commit`, `channel` et `dirty` dans `multiai version --json`.

### P-09 — MOYENNE — Les donnees utilisateur sont dispersees entre plusieurs racines

**Preuves**

- Les profils vivent sous `os.UserConfigDir()/multiai/profiles` (`multiai-go/cmd/multiai/main.go:68-76`).
- Le fallback de secrets utilise `UserHomeDir()/.config/multiai/secrets`, y compris sous Windows (`multiai-go/internal/secret/secret.go:161-178`).
- Le marqueur de premier lancement utilise `~/.multiai/.first-run-done` (`multiai-go/internal/onboarding/wizard.go:103-107`).
- Le logger texte utilise `~/.multiai/logs`, tandis que le journal de sessions utilise `UserConfigDir()/multiai/logs` (`multiai-go/internal/logging/logger.go:36-55`; `session.go:41-51`).
- La desinstallation conseille uniquement `rm -rf ~/.multiai` (`multiai-go/docs/guide/installation.md:178-187`).

**Impact**

Sauvegarde, support, migration et desinstallation sont imprevisibles. Sous Windows, un utilisateur peut supprimer une racine et laisser secrets, profils ou journaux ailleurs.

**Recommandation / criteres d'acceptation**

- Centraliser les chemins dans un package unique conforme a chaque OS, avec migration idempotente des anciennes racines.
- Ajouter `multiai paths --json`, `multiai doctor` et `multiai uninstall --purge --dry-run`.
- La desinstallation doit enumerer exactement ce qui sera supprime et ne jamais effacer sans confirmation.

### P-10 — MOYENNE — L'internationalisation est partielle et l'interface reste personnalisee au mainteneur

**Preuves**

- La langue EN est officiellement detectee via `MULTIAI_LANG`/`LANG` (`multiai-go/internal/i18n/i18n.go:10-15,38-58`).
- L'aide principale est pourtant un bloc francais code en dur (`multiai-go/cmd/multiai/main.go:121-151`).
- OpenRouter et update contiennent de nombreux messages francais directs (`multiai-go/cmd/multiai/cmd_openrouter.go:128-294`; `cmd_update.go:52-122`).
- Les titres FR **et EN** portent `Laurent ROCHETTA's MultiAI` (`multiai-go/internal/i18n/i18n.go:105,146,225,266`).

**Impact**

Le mode anglais produit une interface mixte et freine l'adoption internationale. Le nom personnel dans chaque menu donne l'impression d'un outil prive plutot que d'un projet communautaire, meme si le credit auteur est legitime.

**Recommandation / criteres d'acceptation**

- Faire passer 100 % des messages utilisateur par le catalogue i18n et tester FR/EN par snapshots.
- Utiliser `multiai` comme marque dans l'interface ; conserver l'auteur dans `about`, README et licence.
- Ajouter `multiai config language fr|en` ou documenter clairement `MULTIAI_LANG`.

### P-11 — MOYENNE — Aide et completions ne rendent pas les fonctions livrees decouvrables

**Preuves**

- `migrate` est enregistre (`multiai-go/cmd/multiai/cmd_migrate.go:14-16`) mais absent de l'aide generale (`multiai-go/cmd/multiai/main.go:121-151`).
- Les completions bash/zsh/fish/PowerShell listent launch/list/config/models/search/compare/bmad/version/help/completion, mais omettent update/profile/migrate (`multiai-go/internal/cli/completion.go:14,34,50-59,75`).
- Elles ne proposent pas `--store` ni `--migrate-force`, bien que le code les lise (`multiai-go/cmd/multiai/main.go:282-304`).
- `multiai help launch` documente a `multiai-go/docs/reference/commands.md:229-246` n'est pas le protocole implemente ; le code intercepte `multiai launch --help` (`multiai-go/cmd/multiai/main.go:212-235`).

**Impact**

Les fonctions recentes sont invisibles sauf lecture du README ou du code, et l'autocompletion encourage un contrat incomplet.

**Recommandation / criteres d'acceptation**

- Enregistrer commandes et options dans une structure commune utilisee par dispatch, aide, completions et docs.
- Supporter de facon coherente `multiai help <commande>` et `<commande> --help`.
- Tester que chaque commande enregistree apparait dans les quatre completions et l'aide.

### P-12 — MOYENNE — Le message `--allow-plaintext` propose une option impossible

**Preuves**

- En cas d'indisponibilite du store, le code demande d'utiliser `--allow-plaintext` (`multiai-go/internal/config/wizard.go:348-350`).
- L'appel force toujours `allowPlaintext=false` (`multiai-go/internal/config/wizard.go:296-299`).
- `main.go` ne lit jamais cette option ; ses flags config sont `--store`, `--migrate-force` et `--provider` (`multiai-go/cmd/multiai/main.go:282-316`).

**Impact**

L'utilisateur suit la remediation affichee et reste bloque. Pour un produit centre sur la gestion de secrets, ce type d'impasse nuit fortement a la confiance.

**Recommandation / criteres d'acceptation**

- Choisir une politique : soit implementer l'option avec avertissement/confirmation explicite et tests, soit supprimer toute suggestion de downgrade en clair.
- La recommandation securite est de garder le blocage par defaut et de rendre l'exception tres visible, limitee au profil vise et auditable.

### P-13 — MOYENNE — Le cycle de vie des profils est incomplet et la suppression annoncee n'existe pas

**Preuves**

- Le README promet « ajout/suppression de modeles OpenRouter a la volee » (`README.md:231-235`).
- Le menu OpenRouter cree et peut ecraser un profil (`multiai-go/internal/openrouter/menu.go:172-195`; `profilegen.go:134-166`).
- La recherche `rg -n "Remove|Delete|Supprim|effacer" multiai-go/internal/openrouter` ne trouve aucune fonction de suppression.
- Le registre ajoute des profils, mais n'offre pas update/remove/validate/export (`multiai-go/cmd/multiai/cmd_registry.go:27-50,397-499`).

**Impact**

Les profils s'accumulent et l'utilisateur doit manipuler les fichiers a la main, precisement ce que le produit cherche a eviter.

**Recommandation / criteres d'acceptation**

- Livrer `profile add|show|edit|validate|update|remove|export` avec confirmation, dry-run et JSON.
- Distinguer profil embarque, utilisateur et registre ; ne jamais supprimer silencieusement un profil modifie.
- Rendre les operations idempotentes et afficher la provenance/checksum.

### P-14 — MOYENNE — Il manque une boucle operationnelle « verifier, diagnostiquer, apprendre »

**Preuves**

- Le journal local ne conserve que timestamp, profil, commande, code de sortie et duree (`multiai-go/internal/logging/session.go:12-34`) ; aucun `history` ne l'expose.
- Aucune commande `doctor`, `paths` ou `status` globale n'est enregistree ; les commandes additionnelles sont models/search/compare/update/profile/migrate (`multiai-go/cmd/multiai/cmd_*.go`, commandes `register(...)`).
- La documentation demande encore d'executer manuellement `which claude/codex/opencode` (`multiai-go/docs/guide/troubleshooting.md:100-102`).

**Impact**

Le support ne peut pas obtenir un diagnostic standard, l'utilisateur ne sait pas si son PATH, ses CLI, ses stores, ses profils et le registre sont coherents, et l'equipe produit n'a aucun funnel d'activation fiable.

**Recommandation / criteres d'acceptation**

- Ajouter `multiai doctor [--fix] [--json]` : version, canal, PATH, CLI presents, stores, permissions, profils, schema projet, registre, update et connectivite optionnelle.
- Ajouter `multiai history` pour le journal local, sans telemetrie par defaut.
- Mesurer de facon volontaire et respectueuse de la vie privee : installation reussie, premier config, premier dry-run, premier launch, taux de succes et retention W1. Publier exactement les donnees collectees et permettre un opt-out total.

### P-15 — BASSE — BMAD+ occupe une place de premier niveau dans un produit generique

**Preuves**

- Le menu principal reserve l'option 3 a BMAD+ (`multiai-go/internal/i18n/i18n.go:105-110,225-230`).
- Cette entree apparait pour tous les utilisateurs, alors que le produit est presente comme routeur generique pour trois CLI (`README.md:1-3`).

**Impact**

Pour un utilisateur qui ne connait pas BMAD+, l'information concurrence la tache centrale et brouille la categorie. Pour un utilisateur BMAD+, l'integration reste utile.

**Recommandation / criteres d'acceptation**

- Afficher BMAD+ contextuellement lorsqu'une installation est detectee, ou le placer dans `integrations`.
- Conserver `multiai bmad` comme commande explicite et mesurer son usage avant d'en faire un pilier de marque.

---

## 8. Delta depuis les audits precedents

L'historique montre une progression technique majeure depuis l'audit v0.3.0 (`audit/2026-07-04-v0.3.0/01-produit-fonctionnalites.md`) :

- **Corrige :** implementation Go devenue primaire, 37 profils embarques, round-trip credential store, `%VAR%`, profils utilisateurs, onboarding, hooks branches, OpenRouter models/search/compare, auto-update, stores natifs et migration.
- **Partiellement corrige :** YAML/projet/hooks sont branches, mais le schema public reste incompatible.
- **Toujours present :** suppression de profils absente, `go install`/distribution incoherents, documentation en retard et claims superieurs a l'etat public.
- **Nouveau risque :** le contrat `npx ... install` restaure en `141120b` ne gere pas le PATH Windows et son smoke test contourne la commande globale.

La conclusion n'est donc plus « produit fictif ». Elle est : **produit solide, enveloppe de livraison non fiable**.

---

## 9. Roadmap pour devenir la reference

### P0 — Verite et activation (0-7 jours)

| Priorite | Livraison | Definition of Done | KPI |
|---:|---|---|---|
| 1 | PATH Windows automatique | Ajout idempotent du prefix npm au PATH utilisateur + resolution `multiai --version` dans un nouveau processus. | 100 % des cas E2E Windows propres. |
| 2 | Matrice d'installation honnete | Seuls les canaux publiquement testables sont coches ; liens absents retires. | 0 commande d'installation morte dans README/docs. |
| 3 | Parsing CLI strict | Option inconnue = exit 2 ; `--timeout` implemente ou retire. | 100 % des flags documentes couverts par tests. |
| 4 | Contrat JSON | `launch --json` limite a un mode coherent ou remplace par `--report-json`. | Toute commande JSON valide par parseur dans CI. |
| 5 | Documentation de source unique | Profils, commandes, flags et exemples JSON generes/executes. | 100 % des snippets non destructifs smoke-testes. |
| 6 | Registre | Depot public fonctionnel ou fonctionnalite masquee. | Search/install E2E verts. |

**Gate :** aucune campagne de lancement avant ces six points et avant la CI multi-OS entierement verte.

### P1 — Plan de controle fiable (2-4 semaines)

1. Schema v1 strict pour profils/YAML/projet/hooks, JSON Schema et commandes `validate`.
2. `multiai doctor`, `paths`, version JSON et migration vers une racine de donnees coherente.
3. Cycle complet `profile add/show/edit/validate/update/remove/export`.
4. Aide, completions et docs generees depuis le registre interne de commandes.
5. Internationalisation FR/EN complete et nom de produit neutre dans l'interface.
6. Profil projet par defaut explicite et selection non interactive deterministe.

**KPI cibles :** mediane installation -> premier dry-run < 2 min ; taux premier lancement reussi > 85 % ; zero ticket « commande introuvable » apres installation ; 100 % des exemples docs executes en CI.

### P2 — Leadership de categorie (1-2 mois)

1. Politiques de projet portables : contraintes de fournisseur, region, budget indicatif, contexte et fallback eligible par type d'erreur.
2. Generation de politiques OpenRouter/LiteLLM plutot que duplication d'un proxy.
3. Registre communautaire signe, provenance, revue automatique, compatibilite/version minimale et gouvernance publique.
4. SDK d'adaptateur pour ajouter un nouveau CLI sans modifier le coeur, avec capability matrix.
5. `history` et diagnostic local ; telemetrie uniquement opt-in, documentee et minimale.
6. Trois etudes de cas verifiees avant les objectifs de croissance : solo poly-CLI, equipe avec `.multiai.yaml`, environnement entreprise/proxy Windows.

**KPI cibles :** retention W4 > 35 % parmi les utilisateurs actives ; > 20 % utilisent au moins deux CLI ; > 10 profils communautaires valides ; taux de lancement sans erreur > 95 % ; cinq contributeurs externes actifs.

---

## 10. Score detaille

| Dimension | Poids | Note |
|---|---:|---:|
| Vision et proposition de valeur | 15 % | 8,5/10 |
| Coeur fonctionnel launch/config/isolation | 20 % | 8,5/10 |
| Installation et activation | 20 % | 4,0/10 |
| Verite documentaire et contrats | 15 % | 4,0/10 |
| Extensibilite et cycle de vie | 10 % | 6,0/10 |
| Automatisation, JSON et diagnostic | 10 % | 6,0/10 |
| Ecosysteme, communaute et validation | 5 % | 4,0/10 |
| Internationalisation | 5 % | 6,0/10 |
| **Score pondere** | **100 %** | **6,1/10** |

Le score de potentiel apres P0/P1 est **8,5/10**. Le gain ne demande pas d'ajouter beaucoup de fonctions : il vient surtout de rendre les fonctions existantes fiables, strictes et decouvrables.

---

## 11. Top priorites executives

1. **Corriger automatiquement le PATH Windows et remplacer le faux smoke test.**
2. **Arreter de declarer disponibles les canaux, depots et versions qui ne le sont pas.**
3. **Rendre le CLI strict : options inconnues refusees, timeout reel ou retire.**
4. **Reparer le contrat JSON avant toute promesse d'integration scriptable.**
5. **Choisir un schema YAML/projet/hooks unique, strict et valide par le CLI.**
6. **Generer documentation, aide et completions depuis le code.**
7. **Livrer `multiai doctor` et unifier les chemins de donnees.**
8. **Faire fonctionner le registre ou le masquer.**
9. **Completer le cycle de vie des profils.**
10. **Valider les personas et l'activation avant d'investir dans la croissance.**

---

## Conclusion Atlas

multiai ne doit pas chercher a gagner par le nombre de fournisseurs : OpenCode, OpenRouter et LiteLLM jouent deja ce jeu a une autre couche. Il peut gagner en devenant **la facon la plus sure, la plus simple et la plus reproductible de passer d'un agent de code en CLI a un autre**.

Le principe directeur pour les prochaines releases doit etre :

> **Aucune promesse sans commande executable, aucun succes d'installation sans resolution par le PATH, aucun schema accepte silencieusement, aucun mode JSON qui ne soit pas du JSON.**
