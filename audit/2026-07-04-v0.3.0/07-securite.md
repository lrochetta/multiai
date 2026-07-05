# Audit v0.3.0 — Sécurité

Date 2026-07-04 · Auditeur Shadow (OSINT & Security research) · Score : 4.5/10 · Méthode : audit BMAD+ parallèle + contre-vérification adversariale.

## Résumé

Audit sécurité défensif du routeur multi-IA `multiai` (implémentation Go primaire `multiai-go/` + legacy PowerShell). Le projet manipule des clés API de 14+ fournisseurs, donc l'exigence de sécurité est élevée.

**Bonne nouvelle d'abord** : aucune clé API réelle n'est commitée dans le dépôt ni présente dans l'historique git. Les fichiers `60/61/62-*-fusion.env` à la racine, les profils PowerShell trackés, les `.zip` commités : tous ne contiennent que des placeholders (`PASTE_..._HERE`, `sk-xxxx`, `%OPENROUTER_API_KEY%`, `{env:...}`) ou des valeurs vides. Le scanner `prepublishOnly` (multiai-powershell/package.json:41) est un vrai garde-fou anti-fuite. L'isolation d'environnement par liste blanche (~30 variables) est saine et « fail closed ». Le chiffrement AES-256-GCM est correctement implémenté (nonce aléatoire 12 octets, tag d'authentification).

**Mauvaise nouvelle** : la version auditée (le binaire Go se déclare `0.2.1` alors que README=0.3.0 et packaging=0.5.0) présente un écart majeur entre les *claims* du CHANGELOG/README et la réalité du code. Plusieurs correctifs sécurité annoncés « faits » dans le CHANGELOG v0.2.6 sont fictifs ou inopérants. C'est le cœur de cet audit : au-delà des vulnérabilités techniques, il y a un problème d'intégrité documentaire qui affaiblit la confiance.

Un secret réel a été trouvé sur disque : une clé DeepSeek en clair dans `brainstorm laurent/clé deepseek ne pas mettre dans le repo.txt` (valeur `sk-883d…a330`, **rédigée**). Elle est gitignorée (`brainstorm*`) et **absente de l'historique** — donc pas de fuite publique — mais c'est une clé vivante en clair dans une arborescence git, à un `git add -f` de l'exposition.

## Forces

| Force | Preuve |
|---|---|
| Aucun secret réel commité ni dans l'historique | `git log --all -S`, `git grep` sur sk-ant/sk-or/AIza/ghp_ : seulement fixtures de test |
| Scanner anti-fuite npm robuste | multiai-powershell/package.json:41 (échoue le publish si valeur ≥20 car. non-placeholder) |
| AES-256-GCM correct | internal/secret/crypto.go:55-94 (nonce aléatoire via crypto/rand, préfixé, tag GCM) |
| Isolation env par whitelist, fail-closed | internal/env/env.go:9-31 ; `safeExpandEnv` n'expose que les vars whitelistées |
| Permissions fichiers systématiques | 0600 sur secrets/logs/cache, 0700 sur les dossiers (secret.go:46,60,106 ; logging/logger.go:38,56 ; openrouter/client.go:64,67) |
| Empreinte de dépendances minimale | go.mod : 1 seule dépendance (`gopkg.in/yaml.v3 v3.0.1`), go.sum présent |
| Timeout HTTP défini | openrouter/client.go:40 (10 s) |
| Protection YAML bomb | profile/yaml.go:61-64 et project.go:27-30 (1 Mo max + `NewDecoder`) |
| Dependabot présent | .github/dependabot.yml (gomod + github-actions + npm) |
| Logs ne consignent pas les valeurs de secrets | logging/logger.go ; `MaskSecret` utilisé à l'affichage (env.go:74) |

## Constats détaillés

### 1. Master key AES stockée en clair, à côté du ciphertext — CRITIQUE (v0.2.1 #5 PERSISTE)
`internal/secret/secret.go:51-63` : la clé maître 32 octets est générée aléatoirement puis écrite en clair dans `~/.config/multiai/secrets/.masterkey` (0600), et relue telle quelle. Le fichier chiffré `<service>.enc` vit dans le **même dossier**. Quiconque peut lire le home de l'utilisateur (malware userland, backup non chiffré, sync cloud, autre process) obtient clé + ciphertext côte à côte → le chiffrement n'apporte quasiment aucune protection. C'est exactement le finding #5 de v0.2.1, non corrigé.

### 2. « Credential stores natifs » = stubs, claim README faux — HAUTE (lié v0.2.1 plan #7, PERSISTE)
Le README annonce « Credential store natif : AES-256-GCM + Windows/macOS/Linux ». La réalité :
- **Windows** (store_windows.go:20-45) : fait un `cmdkey /list` cosmétique puis **retombe toujours** sur `encryptedFileStore`. Ne stocke jamais dans le Credential Manager. Valeurs juste base64-encodées (pas chiffrées à ce niveau). Commentaire ligne 19 : « In production, this would use golang.org/x/sys/windows ».
- **macOS** (store_darwin.go:18-48) : fallback `encryptedFileStore` systématique, Keychain jamais utilisé. Commentaire : « In production, this would use CGO + Security.framework ».
- **Linux** (store_linux.go:9-33) : « Attempt D-Bus connection » en commentaire mais utilise directement le fallback fichier.

Les trois « stores natifs » sont donc le même fichier chiffré avec master key en clair (constat #1). Le claim est trompeur.

### 3. Intégration credential-store cassée : clé jamais relue — HAUTE (fonctionnel + sécurité)
`internal/config/wizard.go:269` écrit `VAR=__MULTIAI_CREDSTORE__` dans le `.env` et stocke la vraie clé dans le store (wizard.go:295). Mais **rien ne relit** le store au lancement : `ValidateAndLaunch` → `env.BuildCleanEnv(prof.Env)` (launcher.go:76) utilise `prof.Env` tel quel. Or `dotenv.IsPlaceholder("__MULTIAI_CREDSTORE__")` renvoie **false** (dotenv.go:73-93 : aucun préfixe/suffixe ne matche) → `validateSecrets` (launcher.go:217-228) passe, et la chaîne littérale `__MULTIAI_CREDSTORE__` est transmise comme clé API au CLI enfant. Conséquence : après `multiai config`, tout lancement envoie une fausse clé. Le store est en écriture seule, déconnecté du flux de lancement → la fonctionnalité de sécurité centrale est inopérante.

### 4. Installeur npm : téléchargement de binaire sans vérification d'intégrité — HAUTE (lié v0.2.1 #4/#6)
`multiai-go/packaging/npm/install.js:26-41` télécharge `multiai_<ver>_<platform>` depuis GitHub Releases et l'installe (ligne 70-87) **sans aucune vérification de checksum ni de signature**. Le `checksums.txt` et la signature Cosign produits par goreleaser ne sont jamais récupérés ni comparés. Un MITM sur le CDN ou une release compromise = exécution de binaire arbitraire. C'est le principal chemin d'installation (`npx`).

### 5. Checksums de packaging encore en placeholder — HAUTE (v0.2.1 #4 PERSISTE ; CHANGELOG faux)
Le CHANGELOG v0.2.6 prétend « SHA256 arm/intel dans Homebrew », « SHA256 dans AUR PKGBUILD ». Réalité :
- homebrew/multiai.rb:9-10 → `PLACEHOLDER_ARM64_SHA256`, `PLACEHOLDER_AMD64_SHA256`
- aur/PKGBUILD:11 → `sha256sums=('REPLACE_WITH_ACTUAL_SHA256')`
- scoop/multiai.json:10,14 → `PLACEHOLDER_SHA256`

Les affirmations « fixed » du CHANGELOG sont fausses.

### 6. La CI ne s'exécute jamais — HAUTE (claim README faux)
`.github/workflows/ci.yml:4-7` se déclenche sur `push`/`pull_request` vers **`main`**, or la branche par défaut du dépôt est **`master`** (`git symbolic-ref refs/remotes/origin/HEAD` → master). Aucun run n'est donc déclenché. Les claims README « CI/CD complète (lint, test 6×, security, benchmark) », « go vet : 0 warning », « Couverture : dotenv 93.9%, env 96.0% » ne sont pas vérifiables — la pipeline est dormante. Fausse assurance de qualité.

### 7. Échappement des hooks PowerShell incomplet → injection latente — MOYENNE (v0.2.1 #1 partiellement corrigé)
`internal/cli/hooks.go:14-37` : `escapeShellArg` échappe correctement bash/zsh/cmd, mais pour `powershell`/`pwsh` (lignes 16-20) il n'échappe que le backtick et `"`. Il laisse passer `;`, `|`, `&`, `$(...)`, les retours ligne — tous des séparateurs/opérateurs PowerShell. Un `DISPLAY_NAME=foo; Remove-Item C:\... -Recurse` injecté via un template de profil s'exécute. De plus, l'ordre `escape` puis `os.ExpandEnv` (lignes 55-57 / 106-108) et l'échappement de la commande **entière** casseraient les hooks légitimes (leurs propres `&&`, `|`). **Atténuant** : `opts.Hooks` n'est jamais renseigné dans main.go (toujours nil) → les hooks ne sont pas câblés dans le CLI livré, donc actuellement non atteignable. Mais la feature est annoncée au README (« Plugin hooks before_launch / after_launch ») et le code vulnérable est prêt à être branché.

### 8. Clé DeepSeek réelle en clair dans l'arborescence — MOYENNE
`brainstorm laurent/clé deepseek ne pas mettre dans le repo.txt` contient une clé live `sk-883d…a330` (**rédigée**). Gitignorée et **absente de l'historique** (vérifié via `git log --all -S` + `git grep HEAD`), donc pas de fuite publique — mais un secret vivant en clair dans un répertoire git reste un risque (add forcé, erreur .gitignore, sync). Recommander rotation + suppression + stockage hors dépôt.

### 9. `--allow-custom-command` : bypass whitelist sans validation — MOYENNE (v0.2.1 #15 PERSISTE)
`internal/cli/launcher.go:57-63` : avec `--allow-custom-command`, n'importe quelle commande contourne la whitelist `{claude,codex,opencode}` et est exécutée (LookPath + exec.Command). Aucune validation, aucune restriction par fichier de config. Inchangé depuis v0.2.1.

### 10. Actions GitHub non épinglées par SHA — MOYENNE (v0.2.1 #14 partiel)
ci.yml et release.yml utilisent des tags flottants : `actions/checkout@v4`, `setup-go@v5`, `golangci-lint-action@v7`, `codecov-action@v5`, `goreleaser-action@v6`, `cosign-installer@v3`, `sbom-action@v0` ; pire, `golangci-lint version: latest` (ci.yml:25) et `govulncheck@latest` (ci.yml:65) sont non reproductibles. Seul `securego/gosec@v2.22.3` est semi-épinglé. Dependabot atténue mais l'épinglage SHA annoncé n'est pas fait.

### 11. Cacophonie de versions — MOYENNE
`cmd/multiai/main.go:18` → `const version = "0.2.1"` (et User-Agent `multiai/0.2.1` dans openrouter/client.go:38), README badge = 0.3.0, `multiai-powershell/package.json` = 0.3.0, `packaging/npm/package.json` + install.js + homebrew/scoop/aur = 0.5.0. Cinq numéros différents. `install.js:10,60-61` pointe vers les assets de release `v0.5.0` qui peuvent ne pas exister → installation cassée. L'utilisateur ne peut pas savoir ce qu'il exécute.

### 12. PBKDF2 implémenté mais jamais utilisé (code mort) — MOYENNE (v0.2.1 #14 « fix » inopérant)
`internal/secret/crypto.go:18-53` implémente PBKDF2-HMAC-SHA256 10 000 itérations, mais `encryptedFileStore` chiffre avec la master key brute (secret.go:80,102) et n'appelle **jamais** `DeriveKey` (référencé uniquement dans crypto.go et secret_test.go). Le vrai chemin de chiffrement n'utilise aucun KDF. Le correctif annoncé « PBKDF2 10K itérations (CWE-916) » ne s'applique pas au code réel.

### 13. Fonctionnalités annoncées non câblées — MOYENNE (intégrité/confiance)
Le README/CHANGELOG v0.3.0 vendent comme livrés : `multiai models` / `search` / `compare` (absents du switch main.go:126-182 ; le package `internal/openrouter` n'est importé nulle part dans cmd/), les hooks (opts.Hooks toujours nil), le wizard d'onboarding (`RunWelcome`/`IsFirstRun` jamais appelés — vérifié), la config projet `.multiai.yaml` avec héritage (`FindProjectConfig`/`MergeProjectConfig` référencés seulement dans les tests). Du code mort qui gonfle la surface et porte des vulnérabilités latentes (hooks).

### 14. Client OpenRouter : lecture de corps de réponse non bornée — FAIBLE
`internal/openrouter/client.go:54` : `json.NewDecoder(resp.Body).Decode(...)` sans `io.LimitReader`. Un endpoint compromis peut renvoyer un corps géant → épuisement mémoire. Impact actuel faible (code non câblé) mais à corriger avant d'exposer la feature.

### 15. Dossier secrets résolu relativement au cwd si HOME absent — FAIBLE
`internal/secret/secret.go:45` : `filepath.Join(os.Getenv("HOME"), ".config", "multiai", "secrets")`. Sur Windows `HOME` est souvent vide (c'est `USERPROFILE`) → le chemin devient relatif au répertoire courant, écrivant `.config/multiai/secrets/.masterkey` dans le projet en cours. Utiliser `os.UserHomeDir()` (comme le fait logging/logger.go:36 et openrouter/client.go:62).

### 16. Étape npm-publish de release cassée — FAIBLE
`.github/workflows/release.yml:46` : `cd ../../multiai-powershell` sort de l'arborescence checkoutée → chemin invalide, publish échouerait. `ClearEnv` parsé (yaml.go:162-165) mais jamais consulté au lancement (launcher.go appelle toujours BuildCleanEnv) — pas de trou (fail-closed), mais écart comportement/claim. Zips dupliqués commités à la racine (`claude-code-zai-pack*.zip`, `code-cli-router-pack*.zip`) : hygiène de dépôt/supply-chain (ne contiennent que des templates, vérifié).

## Statut des problèmes v0.2.1

| # | Problème v0.2.1 | Statut | Preuve |
|---|---|---|---|
| #1 | Injection shell hooks via templates | **PARTIEL** — corrigé bash/zsh/cmd, **persiste en PowerShell/pwsh** ; feature non câblée (atténuant) | hooks.go:16-20 |
| #4 | Checksums placeholders packaging | **PERSISTE** (CHANGELOG prétend corrigé) | homebrew/multiai.rb:9-10 ; aur/PKGBUILD:11 ; scoop/multiai.json:10,14 |
| #5 | Master key AES en clair | **PERSISTE** | secret.go:51-63 |
| #6 | Aucune signature binaires | **PARTIEL** — Cosign+SBOM dans goreleaser, mais installeur npm ne vérifie rien | .goreleaser.yml:43-62 vs install.js:26-70 |
| #14 | KDF SHA-256 sans itérations | **INOPÉRANT** — PBKDF2 ajouté mais code mort ; chemin réel = master key brute | crypto.go:18-53 vs secret.go:80,102 |
| #15 | `--allow-custom-command` bypass | **PERSISTE** | launcher.go:57-63 |
| (plan #7) | Credential store natif (remplacer stubs) | **PERSISTE** — toujours des stubs | store_windows.go / store_darwin.go / store_linux.go |
| (plan #14) | CI pins SHA + dependabot | **PARTIEL** — dependabot ok, pins SHA absents ; + CI ne tourne pas (branche) | dependabot.yml vs ci.yml:4-25 |

## Recommandations priorisées

**Immédiat (sécurité critique)**
1. Master key : dériver d'un secret non stocké sur disque (passphrase utilisateur via `DeriveKey` déjà présent, ou DPAPI/Keychain/keyring OS) ; a minima ne pas la ranger dans le même dossier que le ciphertext (secret.go:51-63).
2. Réparer l'intégration credential-store : relire le store au lancement pour substituer `__MULTIAI_CREDSTORE__`, sinon traiter ce marqueur comme placeholder dans `IsPlaceholder` pour échouer proprement (wizard.go:269, launcher.go:76, dotenv.go:73).
3. Rotation immédiate de la clé DeepSeek `sk-883d…a330` et sortie du fichier hors de l'arborescence git.
4. Installeur npm : télécharger `checksums.txt` + signature Cosign et vérifier avant exécution (install.js).

**Haute priorité**
5. Corriger le trigger CI (`master`) et rendre les claims README vérifiables ; épingler les actions par SHA, figer `golangci-lint`/`govulncheck`.
6. Remplir les checksums réels dans homebrew/scoop/aur (ou générer 100% via goreleaser et retirer les placeholders du dépôt) ; corriger le CHANGELOG mensonger.
7. Implémenter réellement les credential stores natifs OU renommer honnêtement le claim README en « fichier chiffré local ».
8. Compléter l'échappement PowerShell des hooks (`;`, `|`, `&`, `$()`, newline) et repenser l'architecture (passer les valeurs en variables plutôt qu'en concaténation de commande) avant de câbler la feature.

**Moyenne priorité**
9. Aligner toutes les versions sur un SSOT unique (ldflags injecte déjà `main.version` — supprimer le `const version` en dur main.go:18).
10. `--allow-custom-command` → liste blanche extensible par fichier de config validé, pas un bypass total.
11. Borner la lecture des réponses HTTP (`io.LimitReader`) et utiliser `os.UserHomeDir()` partout (secret.go:45).
12. Retirer le code mort (openrouter/hooks/onboarding/project) tant qu'il n'est pas câblé, ou le câbler et le tester ; retirer les `.zip` dupliqués du dépôt.

**Note de sécurité : 4,5/10.** Les primitives (AES-GCM, whitelist, scanner anti-fuite, absence de secret commité) sont réelles et valent des points, mais la persistance de findings critiques v0.2.1 (master key en clair, checksums placeholder, bypass whitelist), l'écart massif entre claims et code (stores stubs, PBKDF2 mort, CI dormante, features fantômes), et l'intégration credential-store cassée tirent la note sous le 5,5 précédent : la confiance dans les affirmations de sécurité du projet a régressé.

## Findings contre-vérifiés

| ID | Sévérité | Titre | Verdict | Note |
|---|---|---|---|---|
| 07-03 | high | Intégration credential-store cassée : marqueur `__MULTIAI_CREDSTORE__` transmis comme clé API | CONFIRMED | Chaîne complète vérifiée (wizard.go:269 → dotenv.go:73-93 → launcher.go:76,124) : aucune relecture du store, échec d'auth garanti après `multiai config`. |
| 07-04 | high | Installeur npm sans vérification checksum/signature | CONFIRMED | install.js:26-87 : aucun hash ni Cosign consommé alors que goreleaser les produit ; vecteur release compromise atteignable, MITM atténué par TLS. |
| 07-06 | high | CI jamais déclenchée (trigger `main` vs branche `master`) | CONFIRMED | Pire que décrit : le workflow vit dans `multiai-go/.github/workflows/` (sous-répertoire, jamais lu par GitHub Actions) ; corriger la branche ne suffirait pas. |
| 07-01 | high (corrigée depuis critical) | Master key AES-256 en clair à côté du ciphertext | PARTIAL | Faits confirmés sur les 3 OS (stores tous fallback fichier, DeriveKey jamais appelé), mais modèle d'attaque = lecture du home, équivalent baseline `~/.ssh` à 0600 → sévérité recalibrée CRITICAL→HIGH ; « security theater » réel. |
| 07-02 | medium (sévérité corrigée) | Credential stores « natifs » = stubs (claim README faux) | PARTIAL | Stubs confirmés (aucun wincred/Keychain/libsecret), mais le détail « base64 non chiffré » est réfuté : le fichier est bien chiffré AES-256-GCM au repos ; claim README trompeur, pas de secrets en clair. |
| 07-05 | low (corrigée depuis high) | Checksums packaging en placeholder, CHANGELOG « faussement corrigé » | PARTIAL | Placeholders confirmés, mais l'accusation de mensonge CHANGELOG est réfutée (CHANGELOG.md:77 annonce « Placeholder honnête ») ; fichiers = templates, goreleaser génère les vrais manifestes brew/scoop → impact faible. |
| 07-07 | medium | Échappement hooks PowerShell/pwsh incomplet → injection latente | non contre-vérifié | Atténuant : feature non câblée (opts.Hooks toujours nil). |
| 07-08 | medium | Clé DeepSeek réelle en clair dans l'arborescence | non contre-vérifié | Gitignorée, absente de l'historique ; rotation recommandée. |
| 07-09 | medium | `--allow-custom-command` bypass whitelist sans validation | non contre-vérifié | Persiste depuis v0.2.1 (#15). |
| 07-10 | medium | PBKDF2 implémenté mais jamais utilisé (code mort) | non contre-vérifié | Correctif CHANGELOG « CWE-916 » inopérant sur le chemin réel. |
| 07-11 | medium | Cacophonie de versions (0.2.1 / 0.3.0 / 0.5.0) | non contre-vérifié | install.js pointe vers des assets v0.5.0 potentiellement inexistants. |
| 07-12 | medium | Fonctionnalités annoncées non câblées (code mort étendu) | non contre-vérifié | models/search/compare, hooks, onboarding, config projet : absents du flux réel. |
| 07-13 | low | Lecture de réponse HTTP OpenRouter non bornée | non contre-vérifié | Code non câblé aujourd'hui ; borner avant exposition. |
| 07-14 | low | Dossier secrets relatif au cwd si HOME absent (Windows) | non contre-vérifié | Utiliser `os.UserHomeDir()`. |
| 07-15 | low | Actions GitHub non épinglées par SHA + `@latest` flottants | non contre-vérifié | Dependabot atténue partiellement. |
| 07-16 | low | Étape npm-publish de release cassée (chemin hors checkout) | non contre-vérifié | `cd ../../multiai-powershell` invalide dans le runner. |

Aucun finding intégralement REFUTED : aucun élément écarté. Trois findings PARTIAL avec sévérité recalibrée (07-01 critical→high, 07-02 medium, 07-05 high→low).
