Now I have a complete picture of the codebase. Let me write all seven stories.

---

# Epic: Credential Stores Natifs OS (v0.6.0)

## Contexte general

L'architecture actuelle repose sur un store fichier AES-256-GCM (`internal/secret/`). Les trois plateformes (`store_windows.go`, `store_darwin.go`, `store_linux.go`) sont des stubs qui deleguent toutes au fallback fichier. Le flag `--store` est parsé dans `main.go` (ligne 472) mais retourne un message "pas encore implementé". L'interface `Store` (4 methodes) est utilisée par 5 packages : `internal/config/`, `internal/cli/`, `internal/secret/` (tests), `internal/onboarding/` et `internal/profile/`.

**Contrainte zero-dep** : `go.mod` n'a que `gopkg.in/yaml.v3` comme dependance directe. L'ajout de dependances doit etre justifie et minimise. Pas de CGo aujourd'hui dans `internal/secret/`.

---

## S5.1 — Store natif Windows Credential Manager (wincred)

**Priorite** : HIGH

**Objectif** : Implementer `winCredStore` en appellant l'API Win32 Credential Manager via `syscall` sur `advapi32.dll`, sans CGo ni dependance externe.

**Description technique** : Remplacer le stub `store_windows.go` par une implementation reelle utilisant `syscall.NewLazyDLL("advapi32.dll")` pour charger `CredWriteW`, `CredReadW`, `CredDeleteW`, `CredEnumerateW` et `CredFree`. La structure `CREDENTIALW` (taille, type, TargetName, CredentialBlob, Persist, UserName) sera definie en pur Go. Le `service` (format `"multiai:ca-a1b2c3d4"`) mappe sur `TargetName` dans le Credential Manager, et les cles/valeurs du service mappent sur des entrees individuelles (ou un unique blob JSON si le CM ne permet pas de lister toutes les cles d'un service). `CRED_PERSIST_LOCAL_MACHINE` est utilise pour que la duree de vie soit liee a l'utilisateur Windows. Le `List()` sera implemente via `CredEnumerateW` avec un filtre sur le prefixe `"multiai:"`.

**Fichiers impactes** :
- `internal/secret/store_windows.go` (replacement complet du stub)
- `internal/secret/secret.go` (optionnel : ajout `NewStoreWithBackend()` si on refactor la factory)
- `internal/secret/secret_test.go` (tests conditionnels `//go:build windows`)
- `cmd/multiai/main.go` (desactiver le message "not implemented" pour wincred)

**Tests attendus** :
- Unit : test de la structure CREDENTIALW et marshaling bytes → string (sans appeler le vrai Win32)
- Integration (Windows uniquement) : Set/Get/Delete/List avec un nom de service temporaire, nettoyage en fin de test (marque `//go:build windows`, skip sur CI si pas de credential manager)
- Fuzz : buffers mal formes ne font pas planter le blob parsing

**Resultat attendu** :
- `multiai config --store wincred` store et retrieve les secrets via le Credential Manager
- Les entrees apparaissent dans `cmdkey /list` ou l'interface graphique Credential Manager
- Les entrees sont persistees entre les sessions Windows
- Pas de fuite memoire (CredFree appelle systematiquement)

**Definition of Done** :
- [ ] CREDENTIALW structure et appels syscall implementes sans CGo
- [ ] Set/Get/Delete/List passent sur Windows 10/11
- [ ] tests integration passent (marques `windows`)
- [ ] `multiai config --store wincred` ne montre plus le message "not implemented"
- [ ] goreleaser cross-compile Windows sans erreur CGo
- [ ] `-race` clean

**Risques** :
- `CredEnumerateW` peut etre lent avec des milliers d'entrees Credential Manager existantes (filtrage cote Go apres enumeration)
- Les versions de Windows armored (Windows 10 S, certaines configs GPO) peuvent bloquer l'acces au Credential Manager
- La persistence du blob JSON (si on utilise un blob unique par service) complexifie le merge concurrent

**Dependances** : Aucune (peut etre developpee en parallele de S5.2 et S5.3)

---

## S5.2 — Store natif macOS Keychain (keychain)

**Priorite** : HIGH

**Objectif** : Implementer `keychainStore` en appellant le Security Framework macOS via fichiers CGo separes (`store_darwin_cgo.go` + `store_darwin.go`), avec une minimisation du code C.

**Description technique** : Creer un petit fichier `store_darwin_cgo.go` avec `//go:build darwin && cgo` contenant les appels C (`#cgo LDFLAGS: -framework Security`) a `SecKeychainAddGenericPassword`, `SecKeychainFindGenericPassword`, `SecKeychainDeleteGenericPassword`, `SecKeychainCopyDefault`. Le service name mappe sur le `serviceName` de l'API Keychain, les cles individuelles mappent sur `accountName`. Si CGo est indisponible (`//go:build darwin && !cgo`), basculer sur une implementation shell-out vers `/usr/bin/security` (moins securisee mais fonctionnelle). Le `List()` est complexe avec le Security Framework : utiliser `SecKeychainSearchCreateFromAttributes` ou implementer un fallback qui appelle `security dump-keychain` et parse le output.

**Fichiers impactes** :
- `internal/secret/store_darwin.go` (replacement du stub)
- `internal/secret/store_darwin_cgo.go` (nouveau : appels C directs)
- `internal/secret/store_darwin_nocgo.go` (nouveau : fallback shell-out `security`)

**Tests attendus** :
- Unit : tests de parsing du output de `security dump-keychain`
- Integration (macOS uniquement) : Set/Get/Delete/List sur un keychain temporaire, cleanup
- CGo compile test : `CGO_ENABLED=1 go build ./internal/secret/` sur macOS
- No-CGo compile test : `CGO_ENABLED=0 go build ./internal/secret/` sur macOS

**Resultat attendu** :
- `multiai config --store keychain` stocke les secrets dans le trousseau macOS (Keychain.app)
- Les secrets survivent a un redemarrage
- CGo enabled : appels directs au Security Framework
- CGo disabled : shell-out a `/usr/bin/security` (avec message warning)
- Les secrets sont isoles par service name `"multiai:..."`

**Definition of Done** :
- [ ] Fichier CGo avec appels au Security Framework ecrit et compile
- [ ] Fallback shell-out `security` pour les builds sans CGo
- [ ] tests integration macOS passent
- [ ] `multiai config --store keychain` operationnel
- [ ] goreleaser : build macOS inclut CGo, build Linux sans CGo ignore ce fichier

**Risques** :
- CGo empeche la cross-compilation simple de macOS → Linux dans goreleaser (deja gere par les build constraints)
- `SecKeychainSearchCreateFromAttributes` est deprecated depuis macOS 10.10 (remplace par SecItemCopyMatching) — utiliser la version moderne via `Security.framework` et `kSecClass`, `kSecAttrService`, `kSecAttrAccount`
- Le shell-out vers `security` expose les secrets dans les arguments ps (visible par `ps aux`) — mitigation : pipe via stdin
- Les builds sur CI GitHub Actions macOS ont CGo active par defaut (OK)

**Dependances** : Aucune (parallele avec S5.1 et S5.3)

---

## S5.3 — Store natif Linux libsecret / D-Bus (secret-service)

**Priorite** : HIGH

**Objectif** : Implementer `libsecretStore` en utilisant l'API D-Bus `org.freedesktop.Secret` via un appel a l'outil CLI `secret-tool` (libsecret-1-0 + libsecret-tools), sans dependance D-Bus Go supplementaire.

**Description technique** : Utiliser `os/exec` pour appeler `secret-tool store --label='multiai:...' service <service> key <key> <value>` et `secret-tool lookup service <service> key <key>`, `secret-tool clear service <service> key <key>`. Le `List()` est le defi principal : `secret-tool` n'a pas de commande `list` native. Implementer via `secret-tool search service <service>` qui retourne toutes les entrees d'un service (requete D-Bus native). En alternative, utiliser directement D-Bus via le package `github.com/godbus/dbus/v5` si la precision le requiert (a evaluer). Le fallback fichier AES-256-GCM est utilise si `secret-tool` n'est pas dans `$PATH` ou que le service D-Bus `org.freedesktop.secrets` n'est pas disponible.

**Fichiers impactes** :
- `internal/secret/store_linux.go` (replacement complet du stub)
- `internal/secret/store_linux_test.go` (nouveau : tests avec mock secret-tool)

**Tests attendus** :
- Unit : mock de `exec.Command` pour simuler secret-tool (via une fonction de remplacement injectable ou un test helper qui remplace la commande)
- Integration (Linux uniquement, necessite `gnome-keyring-daemon` ou `keepassxc` comme backend secret-service) : Set/Get/Delete/Clear
- Fallback test : `secret-tool` absent → fallback fichier silencieux
- D-Bus disconnection handling

**Resultat attendu** :
- `multiai config --store secret-service` stocke les secrets dans le service de secret D-Bus
- Compatible avec GNOME Keyring, KDE Wallet, KeePassXC (tout backend implementant org.freedesktop.Secret)
- Si `secret-tool` absent → message informatif + fallback fichier AES-256-GCM automatique
- Les entrees sont isolees par attribut `service=multiai:ca-a1b2c3d4`

**Definition of Done** :
- [ ] Implementation basee sur `secret-tool` (store/lookup/clear/search)
- [ ] Detection de disponibilite D-Bus au moment de `NewStore()`
- [ ] Fallback fichier automatique si store natif indisponible (pas de hard error)
- [ ] tests unitaires (mock command) et integration (Linux D-Bus)
- [ ] `multiai config --store secret-service` operationnel
- [ ] Documentation des prerequis : `apt install libsecret-tools` ou equivalent

**Risques** :
- `secret-tool search` peut etre lent (>500ms) sur des sessions avec beaucoup de secrets
- Certains backends (Gonkey, KDE) peuvent demander un unlock du trousseau via D-Bus prompt — bloquant en CLI non-interactive
- `secret-tool` n'est pas installe par defaut sur les images Docker minimales, Ubuntu Desktop, Fedora Workstation — doit etre documente
- `killall gnome-keyring-daemon` pendant une operation store peut laisser l'appel bloque

**Dependances** : S5.6 (fallback fichier est le comportement attendu quand le D-Bus n'est pas disponible)

---

## S5.4 — Commande `multiai config --store <backend>` implementee

**Priorite** : BLOCKER (depend de S5.1, S5.2, S5.3)

**Objectif** : Remplacer le message "not implemented" de `handleStoreFlag` par le routage effectif vers le backend natif selectionne, avec validation et messages utilisateur clairs.

**Description technique** : Modifier `NewStore()` en `NewStore(backend string)` (ou creer `NewStoreWithBackend(backend string)` pour preservers l'API existante). Si `--store` est fourni : `wincred` → `newWinCredStore()`, `keychain` → `newKeychainStore()`, `secret-service` → `newLibsecretStore()`, `file` → `newEncryptedFileStore()`, `auto` ou absent → `newPlatformStore()` (auto-detection). Si le backend demande (par ex. `keychain` sous Windows) → erreur explicite "backend non disponible sur cette plateforme". Le flag `--store` est propage depuis `main.go` (`config` case) jusqu'a `config.InteractiveConfig()` et `config.ConfigureProviderByID()` via une option struct ou un parametre supplementaire. Le store choisi est utilise pour toutes les operations de Set/Get/Delete dans le wizard. En mode interactif, un message confirme le backend utilise : "Credential store : Windows Credential Manager" / "Credential store : Fichier chiffre AES-256-GCM".

Ajouter la valeur `auto` comme comportement par defaut dans l'aide et la completion.

**Fichiers impactes** :
- `cmd/multiai/main.go` (modification du `config` case, modification de `handleStoreFlag`, mise a jour de `printConfigHelp`)
- `internal/secret/secret.go` (nouvelle fonction `NewStoreWithBackend(backend string)`, refactor `NewStore()` comme alias de `NewStoreWithBackend("auto")`)
- `internal/secret/store_windows.go` (exporter `newWinCredStore`)
- `internal/secret/store_darwin.go` (exporter `newKeychainStore`)
- `internal/secret/store_linux.go` (exporter `newLibsecretStore`)
- `internal/config/wizard.go` (passer `Store` au lieu d'appeler `secret.NewStore()` en interne — injection de dependance)
- `internal/config/erase.go` (idem)
- `internal/config/wizard_test.go` (adapter les appels)
- `internal/i18n/i18n.go` (ajouter messages FR/EN pour "store backend selected", "store not available on this platform")
- `cmd/multiai/cmd_update.go` (verifier qu'il n'utilise pas le secret store)

**Tests attendus** :
- Unit : `TestNewStoreWithBackend` — chaque backend valide retourne le type attendu, backend invalide → erreur
- Unit : `TestNewStoreWithBackend_PlatformCheck` — keychain sous Linux → erreur, secret-service sous Windows → erreur
- Integration : `multiai config --store file --provider openrouter` (flux complet avec store fichier force)
- E2E : `multiai config --store auto` (auto-detection)
- Regression : `multiai config` sans `--store` fonctionne comme avant

**Resultat attendu** :
- `multiai config --store wincred` utilise Windows Credential Manager (Windows)
- `multiai config --store keychain` utilise macOS Keychain (macOS)
- `multiai config --store secret-service` utilise libsecret (Linux)
- `multiai config --store file` force le store fichier existant (toute plateforme)
- `multiai config --store auto` detecte automatiquement le meilleur backend disponible
- `multiai config --store invalid` → message d'erreur explicite "Stores valides: wincred, keychain, secret-service, file, auto"
- `multiai config --store keychain` sous Linux → message d'erreur "keychain non disponible sur cette plateforme"
- Messages i18n (FR/EN) pour tous les nouveaux textes
- Le store choisi est affiche dans l'interface utilisateur

**Definition of Done** :
- [ ] `NewStoreWithBackend(backend)` implementee et testee
- [ ] handleStoreFlag ne retourne plus "not implemented" pour les backends valides sur la bonne plateforme
- [ ] Erreur specifique pour backend invalide ou backend non supporte sur la plateforme courante
- [ ] wizard.go et erase.go utilisent le store injecte (pas de `secret.NewStore()` interne)
- [ ] Messages i18n ajoutes (FR + EN)
- [ ] Tests unitaires et E2E passent
- [ ] `printConfigHelp` mis a jour pour montrer le flag `--store`

**Risques** :
- Changement de signature de `NewStore()` impacte 5 fichiers + tests — necessite une migration soigneuse
- L'injection de dependance dans `wizard.go` et `erase.go` change le flux actuel ou `NewStore()` est appele localement
- La completion shell doit etre mise a jour pour le flag `--store` (package `cli/completion.go`)
- Le flag `--store` doit etre parse dans le `case "config"` AVANT l'appel a `InteractiveConfig()` et `ConfigureProviderByID()`

**Dependances** : S5.1, S5.2, S5.3 (store natifs implementes), S5.6 (fallback)

---

## S5.5 — Zeroisation memoire complete des secrets apres usage

**Priorite** : HIGH

**Objectif** : Garantir qu'aucun secret (API key, master key, token) ne reste dans la memoire du processus apres usage, en etendant la zeroisation au-dela de la fonction `load()` existante.

**Description technique** : L'architecture actuelle zero deja le `plaintext` dans `load()` (lignes 258-262), mais plusieurs vecteurs ne sont pas couverts. Implementer :
1. Une fonction `Zeroize(buf []byte)` dans `internal/secret/crypto.go` qui ecrit `0` sur chaque octet et empeche l'optimisation du compilateur (`runtime.KeepAlive` apres l'ecriture, ou `golang.org/x/sys/cpu.Memclr` si disponible — attention aux dep).
2. Zeroisation du master key apres chaque operation Get/Set/Delete/List (ne JAMAIS garder la cle en memoire plus longtemps que necessaire ; actuellement elle est stockee dans `encryptedFileStore.masterKey` pour toute la duree de vie du store).
3. Zeroisation des valeurs retournees par `store.Get()` cote appelant (`resolveStoredSecrets` dans `launcher.go`, `config.wizard.go`).
4. Zeroisation via `defer` systematique dans tout chemin qui manipule un secret en clair.
5. Utilisation de `MULTIAI_SECRETS_DIR` ≠ `HOME` pour que les tests ne laissent pas de traces memoire.
6. Reviewer `sodium_memzero`-like pattern : pour les buffers alloues dynamiquement, le Go GC peut deplacer les references ; envisager `runtime.Pin` ou `unsafe` pour les buffers critiques (a discuter selon le threat model).

La zeroisation ne doit JAMAIS etre optimisee par le compilateur. Utiliser une fonction no-op externe appellee par `go test -gcflags=-d=checkptr` pour verifier.

**Fichiers impactes** :
- `internal/secret/crypto.go` (nouvelle fonction `Zeroize`, modification de `encrypt`, `decrypt`, `DeriveKey`, `GenerateSalt`)
- `internal/secret/secret.go` (zeroisation du master key dans `encryptedFileStore` apres chaque operation, zeroisation du plaintext dans `load()`, zeroisation dans `save()`)
- `internal/cli/launcher.go` (zeroisation dans `resolveStoredSecrets`)
- `internal/config/wizard.go` (zeroisation dans `configureProvider` et `updateEnvFile`)
- `internal/secret/secret_test.go` (test unitaire que `Zeroize` efface bien la memoire et que le compilateur ne l'optimise pas)
- `internal/secret/crypto_test.go` (nouveau : test de zeroization)

**Tests attendus** :
- Unit : `TestZeroize` — verifier que `buffer[i] == 0` pour tout i apres Zeroize, et que la function n'est pas elidee (utiliser `go test -gcflags=-m` pour verifier l'absence d'inline elision)
- Unit : `TestMasterKeyZeroized` — creer un store, faire un Set, verifier que `store.masterKey` n'est plus accessible (via un getter de test)
- Race : `-race` clean — la zeroisation concurrente ne cause pas de data race
- Benchmark : mesurer le cout de zeroization (nanoseconds par KB) — doit etre <1% du temps de l'operation

**Resultat attendu** :
- `Zeroize()` implementee et verifiee contre l'optimisation compilateur
- Master key zeroise apres chaque Get/Set/Delete/List (recharge depuis le fichier a chaque appel)
- Valeurs retournees par `store.Get()` zeroisees cote appelant
- Aucun secret en clair dans la memoire du processus apres la resolution/configuration
- Le benchmark montre un cout negligeable (<1%)

**Definition of Done** :
- [ ] `Zeroize()` implementee avec protection anti-optimisation compilateur
- [ ] Tous les chemins de code qui manipulent des secrets en clair ont un `defer Zeroize()`
- [ ] master key zeroise et reload entre chaque operation (pas garde en memoire)
- [ ] `go test -race -count=1 ./internal/secret/` propre
- [ ] `go test -gcflags=-m ./internal/secret/` ne montre pas d'elision de Zeroize
- [ ] Test unitaire qui verifie l'effacement effectif (lecture du buffer apres Zeroize)

**Risques** :
- `runtime.KeepAlive` n'est pas une garantie absolue contre l'optimisation (le compilateur peut prouver que le buffer n'est jamais relu apres la zeroisation et elider l'ecriture)
- Le GC Go peut deplacer les references de byte slices, rendant la zeroisation du slice header insuffisante — besoin potentiel de `runtime.Pin` (Go 1.21+) ou d'`unsafe` pour les cas critiques
- Zeroizer le master key apres chaque operation signifie le relire du fichier a chaque appel — impact performance (lecture disque, AES decrypt du master key) → mitigation : cache avec TTL court (1 seconde max)

**Dependances** : Aucune (independante, peut etre developpee en parallele)

---

## S5.6 — Fallback fichier AES-256-GCM si store natif indisponible

**Priorite** : MEDIUM

**Objectif** : Assurer que le systeme ne plante jamais si le store natif n'est pas disponible (D-Bus absent, pas de keychain, credential manager bloque), en retombant silencieusement sur le store fichier existant avec un message informatif.

**Description technique** : Modifier chaque constructeur de store natif (`newWinCredStore`, `newKeychainStore`, `newLibsecretStore`) pour qu'en cas d'echec d'initialisation du backend natif (pas de DLL, pas de service D-Bus, pas de `secret-tool` dans PATH, CGo disabled sur macOS), il retourne un **wrapper fallback** qui encapsule un `encryptedFileStore` et emet un warning. Ce wrapper implemente l'interface `Store` et delegue toutes les operations au `encryptedFileStore`. Le message de warning est emis UNE FOIS par session (sync.Once) pour eviter le spam. Le comportement est deterministe : si `--store <backend>` est explicite et que le backend est indisponible, une erreur est retournee (pas de fallback silencieux). Si `--store auto` ou pas de flag, le fallback est automatique avec un [i] message.

**Fichiers impactes** :
- `internal/secret/store_windows.go` (detection echec CredWriteW, construction du fallback)
- `internal/secret/store_darwin.go` (detection echec SecKeychainAddGenericPassword ou CGo desactive)
- `internal/secret/store_linux.go` (detection echec exec secret-tool)
- `internal/secret/fallback.go` (nouveau : type `fallbackStoreWrapper` avec warning Once)
- `internal/secret/secret.go` (ajout message i18n pour warning unique)
- `internal/display/` (reutilisation de `PrintWarning` pour le message)
- `internal/i18n/i18n.go` (nouveau message `"store_fallback"` FR/EN)

**Tests attendus** :
- Unit : `testFallbackWrapper` — le wrapper delegue correctement au sous-jacent
- Unit : `testFallbackOnlyOnce` — le warning n'est emis qu'une seule fois (sync.Once)
- Integration : sur une machine sans backend natif (Linux sans D-Bus, macOS headless), `multiai config --store auto` utilise le fichier sans erreur
- Integration : `multiai config --store wincred` sur Linux → erreur explicite (pas fallback silencieux)

**Resultat attendu** :
- Store natif indisponible en mode `auto` → message "[i] Keychain indisponible, utilisation du store fichier AES-256-GCM" (1 fois)
- Store natif indisponible en mode explicite `--store wincred` → erreur "[X] WinCred non disponible sur ce systeme"
- Toutes les operations (Get/Set/Delete/List) fonctionnent via le fallback fichier
- Le message de fallback n'est pas duplique en mode interactif

**Definition of Done** :
- [ ] `fallbackStoreWrapper` implemente l'interface `Store`
- [ ] Detection du backend natif dans chaque constructeur de plateforme
- [ ] Warning unique par session (sync.Once)
- [ ] Mode auto → fallback silencieux
- [ ] Mode explicite → erreur si backend indisponible
- [ ] Tests unitaires et integration

**Risques** :
- Detection de disponibilite D-Bus sous Linux peut necessiter un appel a `dbus-send` ou la verification de `$DBUS_SESSION_BUS_ADDRESS` — la variable peut etre absente meme si le service est disponible
- Sur macOS sans CGo, le fallback shell-out `security` peut aussi echouer → fallback fichier (double fallback)
- Le message "[i]" peut derouter les utilisateurs qui ont explicitement installe un backend natif qui ne fonctionne pas — amelioration possible : ajouter `--verbose` pour plus de details

**Dependances** : S5.1, S5.2, S5.3 (besoin des constructeurs natifs pour les wrapper), S5.4 (integration du --store flag)

---

## S5.7 — Migration automatique des secrets existants vers le store natif

**Priorite** : MEDIUM

**Objectif** : Lors du premier demarrage avec un store natif, detecter les secrets presents dans le store fichier (`.enc` + `.masterkey`), les transferer automatiquement vers le store natif, et nettoyer l'ancien store fichier.

**Description technique** : Ajouter une fonction `MigrateFromFileStore(nativeStore Store) (int, error)` qui :
1. Verifie si un store fichier existe (presence du repertoire `secretsDir()` contenant des `.enc` fichiers)
2. Initialise un `encryptedFileStore` temporaire en lecture seule
3. Liste tous les services presents (fichiers `.enc`), decrypte chaque service
4. Pour chaque cle/valeur, appelle `nativeStore.Set(service, key, value)`
5. Si la copie reussit pour toutes les entrees d'un service, supprime le fichier `.enc` et `.lock` correspondants
6. Si la migration echoue a mi-chemin, elle est **transactionnelle** par service : pas de perte de donnees (l'ancien fichier n'est supprime que si la copie vers le natif a reussi)
7. A la fin, si tous les fichiers ont ete migres, supprime le fichier `.masterkey`

Le point d'appel est dans `main.go` dans le bloc `config` case, APRES la selection du store et AVANT le wizard interactif. Un message resume le nombre de secrets migres.

La migration est **one-shot** : un marker file `.migrated` est ecrit dans le repertoire des secrets pour eviter une re-migration au prochain demarrage. Si l'utilisateur re-cree un store fichier plus tard, une nouvelle migration peut etre forcee avec `--migrate-force`.

**Fichiers impactes** :
- `internal/secret/secret.go` (nouvelle fonction `MigrateFromFileStore`, detection pre-migration)
- `internal/secret/migration.go` (nouveau : logique de migration transactionnelle par service)
- `internal/secret/secret_test.go` (tests de migration)
- `cmd/multiai/main.go` (appel a `MigrateFromFileStore` avant le wizard config, flags `--migrate` et `--migrate-force`)
- `internal/i18n/i18n.go` (messages FR/EN pour les resultats de migration)
- `docs/reference/commands.md` (documentation des flags de migration)

**Tests attendus** :
- Unit : `TestMigrateFromFileStore` — creer 3 services dans le store fichier, migrer vers un store mock, verifier que les donnees sont copiees et les fichiers supprimes
- Unit : `TestMigrateFromFileStore_Rollback` — simuler un echec sur le 2eme service, verifier que le premier service est integre et que les fichiers ne sont pas supprimes
- Unit : `TestMigrateFromFileStore_AlreadyMigrated` — marker present → no-op
- Integration : migration reelle du store fichier vers le store natif avec verification de chaque secret
- E2E : `multiai config --store wincred` avec des secrets existants dans le store fichier

**Resultat attendu** :
- Premier `multiai config --store wincred|keychain|secret-service` → "[i] Migration de 3 secrets depuis le store fichier vers Windows Credential Manager"
- Les secrets migres sont accessibles via le store natif
- Les fichiers `.enc` et `.masterkey` sont supprimes apres migration reussie
- `multiai launch` avec les memes profils fonctionne (les sentinelles `__MULTIAI_CREDSTORE__` sont resolues via le store natif)
- Un second appel ne remigre pas (marker present)
- En cas d'echec partiel, les donnees ne sont pas perdues

**Definition of Done** :
- [ ] `MigrateFromFileStore` implementee et testee
- [ ] Transaction par service (old file supprime seulement apres write natif reussi)
- [ ] Marker `.migrated` empeche la re-migration
- [ ] Flag `--migrate-force` pour forcer une re-migration
- [ ] Appel automatique au premier `config --store <natif>`
- [ ] Messages utilisateur clairs (FR/EN) sur le nombre de secrets migres
- [ ] Si le store natif contient deja des entrees avec les memes services/cles, ne pas ecraser sans confirmation

**Risques** :
- Migration concurrente : si deux processus `multiai config` sont lances simultanement, ils peuvent tous deux tenter la migration → utiliser le lock inter-processus existant sur un fichier de lock de migration dedie
- Tres gros volumes : si l'utilisateur a des centaines de cles, la migration peut prendre du temps → ajouter une barre de progression ou un compteur
- Le marker `.migrated` peut rester orphelin si l'utilisateur supprime le store natif et revient au store fichier manuellement → documenter `--migrate-force` et la possibilite de supprimer le marker manuellement

**Dependances** : S5.4 (commande --store implementee pour declencher la migration), S5.6 (fallback si store natif indisponible pendant la migration)

---

## Reseau des dependances entre stories

```
S5.1 (wincred) ──────┐
S5.2 (keychain) ──────┤
S5.3 (secret-svc) ────┤
S5.5 (zeroization) ───┤
                      ▼
            S5.4 (--store flag) ──── S5.6 (fallback)
                      │
                      ▼
            S5.7 (migration) ──── S5.6
```

**Ordre de sprint recommande** :
1. S5.5 (zeroization) — independant, fondation securite
2. S5.1, S5.2, S5.3 (stores natifs) — en parallele, 3 devs ou 1 dev sequentiel
3. S5.6 (fallback) — apres les 3 stores natifs
4. S5.4 (--store flag) — apres les stores natifs ET le fallback
5. S5.7 (migration) — en dernier, depend de tout