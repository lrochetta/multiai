# Audit Sentinel — Sécurité, Qualité, Conformité & UX

**Projet :** multiai (CLI Go — routeur multi-IA)  
**Version auditee :** 0.4.0-dev  
**Date :** 2026-07-05  
**Agent :** Sentinel (Qualite/Securite/QA)  
**Commits :** 70eb802 (HEAD), 476d64d (v0.3.0), 3912c3a, 72c862c, 97bc484  
**Go :** 1.26.4  
**`go vet ./...` :** PROPRE (aucun avertissement)  
**`go test ./...` :** 16 paquets OK, 0 echec

---

## Sommaire

1. [VOLET SECURITE](#vollet-securite)
   - 1. Gestion des secrets
   - 2. Fuite de secrets
   - 3. Surface d'attaque
   - 4. Dependances
   - 5. Securite du binaire
   - 6. Distribution
2. [VOLET QUALITE](#volet-qualite)
   - 7. Robustesse
   - 8. Tests
   - 9. Logging & observabilite
   - 10. Gestion d'erreurs UX
3. [VOLET CONFORMITE](#volet-conformite)
   - 11. Licences
   - 12. RGPD/Privacy
   - 13. SLSA / Supply chain
4. [VOLET UX](#volet-ux)
   - 14. Onboarding
   - 15. Messages d'erreur
   - 16. Interface CLI
   - 17. Menu interactif

---

## VOLET SECURITE

### 1. Gestion des secrets

#### 1.1 Chiffrement AES-256-GCM

- Fichiers : `internal/secret/crypto.go`, `internal/secret/secret.go`
- Mecanisme : AES-256-GCM avec nonce aleatoire de 12 octets, cle maitre de 32 octets
- Derivation PBKDF2-HMAC-SHA2500 (10000 iterations) reservee pour usage futur (`DeriveKey`)
- Zeroisation memoire apres decrypt : `internal/secret/secret.go:179-182`

**Forces :**
- Implementation sans dependance externe (stdlib crypto)
- Nonce aleatoire a chaque chiffrement (GCM)
- Troncature robuste de la cle maitre a 32 octets (tolere les fichiers de cle plus longs)
- Race condition O_CREATE|O_EXCL gere correctement

**Faiblesses :**

| # | Severite | Description | Fichier:Ligne |
|---|----------|-------------|---------------|
| 1 | 🟡 Élevée | **Cle maitre stockee a cote du ciphertext** — le master key (.masterkey) et les secrets chiffres (.enc) vivent dans le meme repertoire avec les memes permissions 0600. Tout attaquant lisant le repertoire peut dechiffrer. | `secret.go:97` |
| 2 | 🟢 Moyenne | **Absence de liaison machine** — pas de TPM, Secure Enclave, ou DPAPI. Le fichier .masterkey est portable et dechiffrable sur n'importe quelle machine. | `secret.go:6-13` (documente) |
| 3 | ⚪ Faible | **Absence de passphrase** — aucun secret utilisateur ne protege la cle maitre. | `secret.go:18` (documente) |

#### 1.2 Stores platformes (tous stubs)

Les trois plateformes (`store_windows.go`, `store_darwin.go`, `store_linux.go`) utilisent le fichier chiffre de fallback. Les stores natifs (Windows Credential Manager, macOS Keychain, libsecret D-Bus) sont declares "planned — roadmap 1.10".

```go
// store_windows.go:5-7  -- Delegue au fichier AES
// store_darwin.go:5-7   -- Delegue au fichier AES
// store_linux.go:9-16   -- Tentative D-Bus commentee, fallback fichier
```

#### 1.3 Sentinel pattern (forteresse)

- Le sentinel `__MULTIAI_CREDSTORE__` remplace la vraie cle dans les fichiers .env
- `resolveStoredSecrets()` : resolve les sentinels avant lancement (`launcher.go:303-326`)
- `updateEnvFile()` : ecrit le sentinel dans le fichier APRES avoir stocke dans le credential store (`wizard.go:258-273`)
- Test d'invariant : `TestUpdateEnvFile_SentinelInvariant` verifie que la cle n'apparait jamais en clair dans le fichier

**Probleme : fail-unsafe degrade**

| # | Severite | Description | Fichier:Ligne |
|---|----------|-------------|---------------|
| 4 | 🟢 Moyenne | **Degradation silencieuse en texte clair** — si le credential store est indisponible, `updateEnvFile` ecrit la cle API en CLAIR dans le .env avec seulement un avertissement. Aucune validation utilisateur demandee, aucune option --force requise. | `config/wizard.go:268-271` |

**Scenario d'attaque :** Un utilisateur lance `multiai config` alors que son dossier `~/.config/multiai/secrets/` est verrouillé (permissions, noyau, conteneur). La cle est ecrite en clair dans le .env sans que l'utilisateur ne confirme explicitement.

**Correctif recommande :** Rendre le credential store obligatoire pour ecrire le sentinel. Si le store est indisponible, afficher un message d'erreur bloquant : "Credential store inaccessible. La cle ne peut pas etre stockee en toute securite. Corrigez les permissions ou definissez MULTIAI_SECRETS_DIR." Ajouter `--allow-plaintext` pour forcer l'ecriture en clair avec double confirmation.

---

### 2. Fuite de secrets

#### 2.1 Logs session JSONL

Fichier : `internal/logging/session.go`

- Schema `SessionEvent` explicitement concu pour NE PAS contenir de secrets : seulement timestamp, shortcut, profile_path, command (nom du binaire), exit_code, duration
- Test de garde : `TestSessionEvent_NoSecretFields` (`session_test.go:121-149`) — echoue si un champ non autorise est ajoute
- Ecriture silencieuse des erreurs (silent failure) — pas de risque de planter sur un echec de log

**Propre.** Aucune fuite via le journal de session.

#### 2.2 Logger fichier

Fichier : `internal/logging/logger.go`

- Niveaux : DEBUG, INFO, WARN, ERROR
- MinLevel par defaut : INFO (DEBUG non ecrit sauf explicitement active)
- Sortie fichier + stderr pour WARN/ERROR

**Observation :** Il n'y a AUCUN appel a `logging.Debug()` dans la base de code. Le niveau DEBUG est donc inutilise. C'est une bonne chose du point de vue securite car aucun secret ne peut fuiter via DEBUG, mais cela indique que le logger est peu utilise dans le code (seulement 1 appel : `onboarding/wizard.go:73`).

| # | Severite | Description | Fichier:Ligne |
|---|----------|-------------|---------------|
| 5 | ⚪ Faible | **Logger sous-utilise** — un seul appel a logging.Error dans tout le projet. Les erreurs sont principalement `fmt.Fprintf(os.Stderr)`. | `logger.go` |

#### 2.3 Expansion de variables et masquage

Fichier : `internal/env/env.go`

- `IsSecretKey()` : detecte les cles contenant KEY, TOKEN, SECRET, PASSWORD, AUTH, CREDENTIAL
- `MaskSecret()` : affiche les 4 premiers + 4 derniers caracteres, ou `***`, ou `<vide>`
- `maskedEffectiveEnv()` dans `launcher.go:267-277` : utilise ces deux fonctions pour le rendu JSON

**Mecanisme solide.** Aucune fuite identifiee dans les affichages.

#### 2.4 Passage aux processus enfants

Fichier : `internal/cli/launcher.go`

- `buildProcessEnv()` construit l'environnement du processus enfant
- Mode CLEAR_ENV=true : seules les variables systeme autorisees + les vars du profil
- Mode CLEAR_ENV=false : environnement courant + vars du profil
- Les profils peuvent definir des variables avec `%VAR%` fusion inter-variables

**Alerte :** Tout processus enfant recoit TOUTES les variables du profil en clair, y compris les cles API. C'est le comportement attendu (le CLI a besoin des cles), mais cela signifie que tout processus enfant compromis expose les secrets.

| # | Severite | Description | Fichier:Ligne |
|---|----------|-------------|---------------|
| 6 | ⚪ Faible | **Secrets exposes aux processus enfants** — les cles API sont transmises comme variables d'environnement aux processus enfants (necessaire, mais documenter le risque). | `launcher.go:226-244` |

**Ce n'est pas un bug** — c'est le fonctionnement prevu. A documenter dans les consignes de securite.

---

### 3. Surface d'attaque

#### 3.1 Injection de commandes via hooks

Fichier : `internal/cli/hooks.go`

Les hooks executent des commandes shell arbitraires definies dans les profils YAML avec la chaine suivante :

1. `expandHookVars()` : remplace `{{.Profile.X}}` par les valeurs du profil
2. `escapeShellArg()` : echappe les metacaracteres shell
3. `os.ExpandEnv()` : **Etend les variables d'environnement SYSTEME APRES echappement**

| # | Severite | Description | Fichier:Ligne |
|---|----------|-------------|---------------|
| 7 | 🟡 **Élevée** | **Injection via `os.ExpandEnv` apres echappement shell** — `os.ExpandEnv()` est appelee APRES `escapeShellArg()`. Si une variable d'environnement systeme contient des metacaracteres shell (; rm -rf /, $(malicious), etc.), ceux-ci sont inseres sans echappement dans la commande. | `hooks.go:57` et `hooks.go:108` |

**Scenario d'attaque :** Une variable d'environnement `$PROMPT` ou `$MY_CUSTOM_VAR` contient `; rm -rf $HOME`. Le hook `echo {{.Profile.DisplayName}}` est converti en `echo mon-profil` (echappe), puis `os.ExpandEnv` transforme une reference `$PROMPT` dans la valeur de `DISPLAY_NAME` (ou n'importe quelle variable shell dans la commande) en `echo mon-profil; rm -rf $HOME`.

**Correctif recommande :** Inverser l'ordre : appeler `os.ExpandEnv()` AVANT `escapeShellArg()`, ou ne pas utiliser `os.ExpandEnv` du tout (les hooks devraient utiliser l'expansions de variables internes `expandHookVars` uniquement). Solution plus robuste : remplacer `os.ExpandEnv` par une expansion controlee des seules variables internes.

#### 3.2 Commande personnalisee

Fichier : `internal/cli/launcher.go:68-73`

- `AllowedCommands = ["claude", "codex", "opencode"]`
- `--allow-custom-command` flag pour outrepasser

| # | Severite | Description | Fichier:Ligne |
|---|----------|-------------|---------------|
| 8 | 🟢 Moyenne | **--allow-custom-command abaisse la securite** — un profil malveillant avec COMMAND=/chemin/vers/binaire peut etre lance. A documenter comme un risque. | `launcher.go:69` |

**Correctif :** Actuellement correct (flag explicite requis). Ajouter un message d'avertissement quand il est utilise.

#### 3.3 Parsing .env et validation des entrées

Fichier : `pkg/dotenv/dotenv.go`

- Parser robuste : gere BOM UTF-8, `export`, guillemets simples/doubles, commentaires
- Aucune validation de contenu au-dela du parsing
- `IsPlaceholder()` : detection des valeurs non configurees

**Propre.** Le parser .env est standard et ne presente pas de risques d'injection specifiques puisque les fichiers .env ne sont pas interpretes par un shell.

#### 3.4 Resolution des profils CWD

Fichier : `cmd/multiai/main.go:59-63`

```go
if os.Getenv("MULTIAI_DEV") != "" {
    if dir := filepath.Join("configs", "profiles"); isDir(dir) {
        return dir
    }
}
```

| # | Severite | Description | Fichier:Ligne |
|---|----------|-------------|---------------|
| 9 | 🟢 Moyenne | **Profils relatifs au CWD avec MULTIAI_DEV** — documente comme une surface d'attaque (commentaire ligne 56-58). La resolution des sentinels utilise `ServiceForProfile(prof.Path)` derive du nom de fichier, donc un attaquant pourrait creer `./configs/profiles/ca.env` pour faire resolver le sentinel d'un autre profil. | `main.go:59-63` |

**Correctif :** Actuellement correct (opt-in via `MULTIAI_DEV`). Ajouter un message d'avertissement a l'ecran quand ce mode est actif : `"⚠ Mode MULTIAI_DEV actif — les profils du repertoire courant sont charges."`

#### 3.5 Parsing de fichiers YAML sans limite de profondeur

Fichier : `internal/profile/yaml.go`

- Limite de taille : 1 Mo (bonne pratique)
- Pas de limite de profondeur de struct YAML (attaque `billion laughs` possible)

| # | Severite | Description | Fichier:Ligne |
|---|----------|-------------|---------------|
| 10 | ⚪ Faible | **Pas de limite de profondeur YAML** — `yaml.Decoder` par defaut peut exploser en memoire avec un fichier profondement imbrique (billion laughs). Probabilite faible car les fichiers YAML sont locaux. | `yaml.go:69-72` |

**Correctif :** Utiliser `decoder.KnownFields(true)` et limiter la profondeur avec un wrapper.

---

### 4. Dependances

#### 4.1 Analyse go.mod

```
module github.com/lrochetta/multiai
go 1.22
require gopkg.in/yaml.v3 v3.0.1
```

- **1 seule dependance directe** — excellent pour la surface d'attaque
- `gopkg.in/yaml.v3 v3.0.1` : Apache 2.0, pas de CVE connue, version stable
- **Aucune dependance transitive** (go.sum minimal)
- Chiffrement : stdlib Go uniquement (`crypto/aes`, `crypto/cipher`, etc.)
- HTTP : stdlib Go uniquement

| # | Severite | Description | Fichier:Ligne |
|---|----------|-------------|---------------|
| 11 | ⚪ Faible | **Aucun outil d'analyse de vulnerabilites** — pas de `go vulncheck`, Dependabot Go, ou Snyk configures. | `go.mod` |

**Recommandation :** Ajouter une etape `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` dans la CI.

---

### 5. Securite du binaire

#### 5.1 Flags de compilation

Fichier : `Makefile`

```makefile
LDFLAGS = -ldflags="-s -w -X main.version=$(VERSION)"
```

- `-s` : strip symbol table (bon pour la taille, reduit la surface d'attaque)
- `-w` : strip DWARF debug info
- `-X` : injection du numero de version
- PIE (Position Independent Executable) : par defaut avec Go 1.22+ sur les plateformes modernes — verifier avec `go build -buildmode=pie`

#### 5.2 Signatures et code signing

| # | Severite | Description |
|---|----------|-------------|
| 12 | 🟢 Moyenne | **Pas de signature Cosign/GPG** — les binaires releases ne sont pas signes. Aucune attestation SLSA. |
| 13 | ⚪ Faible | **Pas de signature macOS (codesign)** — les binaires darwin ne sont pas notarises. |

#### 5.3 SBOM

| # | Severite | Description |
|---|----------|-------------|
| 14 | ⚪ Faible | **Pas de SBOM genere** — ni SPDX ni CycloneDX. |

---

### 6. Distribution

#### 6.1 install.sh

Fichier : `scripts/install.sh`

- **Forces :** Verification sha256 contre checksums.txt, installation dans `~/.local/bin`, nettoyage du repertoire temporaire (`trap`)
- **Faiblesse :** Pas de signature GPG sur checksums.txt — une compromission du CDN GitHub permettrait de remplacer checksums.txt et le binaire

| # | Severite | Description | Fichier:Ligne |
|---|----------|-------------|---------------|
| 15 | 🟢 Moyenne | **checksums.txt non signe** — l'installateur verifie le sha256 mais pas de signature GPG/Cosign sur la somme de controle. Mitige par le transport HTTPS. | `install.sh:48-59` |

#### 6.2 Packaging

- **npm** (`packaging/npm/`) : integrity npm standard
- **AUR** (`packaging/aur/`) : PKGBUILD avec checksums
- **deb** (`packaging/deb/`) : postinst
- **Homebrew et Scoop** : FICHIERS SUPPRIMES (`.goreleaser.yml` supprime aussi, remplace par `.goreleaser.yaml`)

| # | Severite | Description |
|---|----------|-------------|
| 16 | 🟢 Moyenne | **Homebrew et Scoop non maintenus** — les formules ont ete supprimees du depot. Les utilisateurs de ces plateformes ne recoivent pas de mise a jour. |

---

## VOLET QUALITE

### 7. Robustesse

#### 7.1 Gestion des erreurs

- Retour d'erreurs avec `%w` (wrapping) partout — bon pour le debugging
- Echecs silencieux volontaires : log, first-run marker, extraction de profils
- Timeouts : 10s HTTP timeout pour OpenRouter API
- Limites de taille : 32MB reponse OpenRouter, 1MB fichiers YAML
- Signal SIGINT/SIGTERM transfere au processus enfant avec `atomic.Bool` pour marquer l'interruption

#### 7.2 Race conditions

- `sync.Mutex` dans le store de secrets : protege la lecture/ecriture concurrence (`secret.go:63`)
- `sync.Mutex` dans le logger : protege l'acces fichier (`logger.go:28`)
- `sync.Mutex` pour le journal de sessions : serialise les appends (`session.go:37`)
- `sync.OnceValues` pour le chargement du catalogue (`catalog.go:123`)
- `atomic.Bool` pour le flag d'interruption dans la boucle de signaux (`launcher.go:174`)
- **Race condition corrigee dans `secretsDir()`** : l'utilisation de `os.UserHomeDir()` evite les problemes de `HOME` sur Windows

#### 7.3 Gestion des cas limites

- Fichier .env vide : gere (`profile.go`)
- Repertoire de profils manquant : gere avec fallback sur `configs/profiles`
- Marqueur first-run absent : `FirstRunMarkerExists()` retourne true si pas de home (ne jamais relancer)
- Cycle d'expansion `%VAR%` : profondeur max 10, pas de boucle infinie
- JSON null vs [] : converti (`openrouter/models`)

#### 7.4 Points faibles

| # | Severite | Description | Fichier:Ligne |
|---|----------|-------------|---------------|
| 17 | 🟢 Moyenne | **setEnvVarInFile non atomique** — utilise un temp file + rename, MAIS pas fsync sur le fichier temporaire, et pas de verification que le rename est atomique sur tous les FS (NFS, overlay, etc.). | `wizard.go:309-317` |
| 18 | ⚪ Faible | **Pas de verrouillage fichier sur le journal de sessions** — 2 processus `multiai` simultanes ecrivant dans le meme fichier JSONL peuvent entrelacer les lignes. | `session.go:67-78` |
| 19 | ⚪ Faible | **Cache OpenRouter non atomique** — `SaveCache()` utilise tmp+rename sans Sync. Contraste avec `WriteFileAtomic` dans `fsutil` qui est atomic. | `openrouter/cache.go:72-78` |

---

### 8. Tests

#### 8.1 Couverture globale : EXCELLENTE

16 paquets, tous OK, 21 fichiers de test.

**Tests de securite presents :**
- `TestEncryptDecrypt` et `TestEncryptDecrypt_EmptySlice` — crypto
- `TestPlatformStoreRoundTrip` — store complet config -> launch
- `TestServiceNameIsFileSafe` — pas d'ADS NTFS
- `TestUpdateEnvFile_SentinelInvariant` — sentinel pattern
- `TestResolveStoredSecrets` — resolution du sentinel
- `TestSessionEvent_NoSecretFields` — garde de schema
- `TestEmbeddedProfilesContainNoRealSecrets` — pas de secrets dans les templates
- `TestEraseProviderKeys_ResetsFilesAndStore` — effacement complet

**Trous dans la couverture :**

| # | Zone non testee | Fichier | Risque |
|---|-----------------|---------|--------|
| 20 | **Hooks execution** — `RunBeforeHooks` et `RunAfterHooks` n'ont AUCUN test. | `cli/hooks.go` | C'est la fonction avec le plus haut risque de securite (injection). |
| 21 | `resolveStoredSecrets` avec store indisponible | `cli/launcher.go:303-326` | Degradation inattendue si le store echoue. |
| 22 | `buildProcessEnv` avec `CLEAR_ENV=false` | `cli/launcher.go:226-244` | Teste indirectement via `TestBuildProcessEnv_RespectsClearEnv` dans `fallback_test.go:350`. OK en verite. |
| 23 | Integrations reelles — pas de test qui lance un vrai CLI (claude, codex) | `tests/` | Testable uniquement avec mock. |
| 24 | `LoadDirYAML` et `LoadAllProfiles` | `profile/yaml.go` | Les tests YAML sont minimaux. |

#### 8.2 Qualite des tests

- Les tests sont **bien ecrits** : `t.Helper()`, `t.TempDir()`, `t.Setenv()` utilises partout
- Les tests de fallback utilisent un vrai sous-processus (le binaire de test lui-meme) — excellent
- Les tests de menu utilisent des readers pipes — reproductibles, pas d'interaction humaine
- Pas de `t.Parallel()` (certains tests mutent des globales comme `apiBase`, `maxResponseBytes`)

---

### 9. Logging & observabilite

#### 9.1 Duplication des chemins de logs

| # | Severite | Description | Fichier |
|---|----------|-------------|---------|
| 25 | ⚪ Faible | **Deux destinations de logs differentes** : le logger ecrit dans `~/.multiai/logs/multiai.log` tandis que le journal de sessions utilise `<UserConfigDir>/multiai/logs/sessions.jsonl`. Inconsistance du repertoire de base. | `logger.go:37` vs `session.go:51` |

#### 9.2 Verdict logging

- Le logger fichier est fonctionnel mais a peine utilise
- Le journal de sessions est bien concu et secure
- **Recommandation :** Uniformiser les repertoires de logs et utiliser `<UserConfigDir>/multiai/logs/` partout

---

### 10. Gestion d'erreurs UX

#### 10.1 Messages utilisateur

- Tous les messages sont en francais (conforme aux specs)
- Messages actionnables : "Edite : fichier.env" ou "Lance : multiai config"
- Sortie coloree (ANSI) avec `NO_COLOR` respecte
- Codes de retour documentes (0-130+)
- Pas de stacktrace brute exposee a l'utilisateur

#### 10.2 Points a ameliorer

| # | Severite | Description | Fichier:Ligne |
|---|----------|-------------|---------------|
| 26 | ⚪ Faible | **Codes de retour non standardises** — 2 = config error, 3 = output failure, 4 = fallback error. Il manque une specification formelle des codes. | `main.go` (implicite) |
| 27 | ⚪ Faible | **jsonError() utilise un string replace** (`strings.ReplaceAll(msg, "\"", "\\\"")`) pour echapper le JSON. Risque d'injection JSON si le message contient des caracteres de controle. | `launcher.go:296-298` |

---

## VOLET CONFORMITE

### 11. Licences

| Dependance | Licence | Compatible |
|------------|---------|------------|
| `gopkg.in/yaml.v3 v3.0.1` | Apache License 2.0 | Oui |
| Go standard library | BSD-style | Oui |

| # | Severite | Description |
|---|----------|-------------|
| 28 | 🟡 Élevée | **Aucun fichier LICENSE trouve a la racine du depot.** Le fichier LICENSE est absent (`D:/travail/DEV/multiai/LICENSE` n'existe pas, ni ses variantes). Le projet est publie sans licence explicite, ce qui cree une ambiguite legale pour les contributeurs et utilisateurs. |

### 12. RGPD/Privacy

| Aspect | Verdict |
|--------|---------|
| Telemetrie | **Absente.** Aucun appel reseau en dehors des commandes OpenRouter explicites. |
| Journal local | `sessions.jsonl` reste local, contient seulement metadonnees (pas de secrets). |
| Premier lancement | Marqueur local `~/.multiai/.first-run-done`. |
| Installation | `install.sh` telecharge depuis GitHub releases seulement. |
| npx BMAD+ | Confirmation demandee avant execution. |

**Verdict :** Excellent. Aucune collecte de donnees sans consentement identifiee.

### 13. SLSA / Supply chain

| Niveau | Atteint |
|--------|---------|
| SLSA 1 (documentation du build) | Partiel (Makefile present, .goreleaser.yaml present) |
| SLSA 2 (build hosted) | Non (CI non verifiee) |
| SLSA 3 (build hermetic) | Non |
| SLSA 4 (reproducible build) | Non |

| Controle | Statut |
|----------|--------|
| go.sum present | Oui |
| Build reproducible | Non |
| Cosign signature | Non |
| SBOM | Non |
| Provenance attestation | Non |
| Dependabot config | Oui (npm) mais pas pour Go |

---

## VOLET UX

### 14. Onboarding

**Points forts :**
- Message de bienvenue clair et chaleureux avec instructions etapes par etapes
- Detection du premier demarrage : verifie si au moins une cle API est configuree
- Detection des `%VAR%` indirections pour ne pas etre trompe par les templates vierges
- Marqueur first-run-done : ne repete jamais le wizard
- Transition fluide vers le menu de configuration

**Points faibles :**

| # | Severite | Description | Fichier:Ligne |
|---|----------|-------------|---------------|
| 29 | ⚪ Faible | **Le fil RSS/WhatsNew "Afficher les nouveautes" n'existe pas** — l'utilisateur ne sait pas ce qui a change entre les versions. | `onboarding/wizard.go` |
| 30 | ⚪ Faible | **Pas de progression visuelle lors de la configuration en masse** ("a" = tous les fournisseurs) — l'utilisateur ne sait pas combien de fournisseurs restent. | `config/wizard.go:146-148` |

### 15. Messages d'erreur

**Points forts :**
- Messages en francais, actionnables, clairs
- Codes couleur pour le niveau de gravite (vert/jaune/rouge/cyan)
- Respect de `NO_COLOR` pour l'accessibilite
- `--json` mode pour consommation programmee
- Erreurs avec nom du fichier concerne et commande de correction

**Points faibles :**
- Le message d'erreur du store indisponible pendant `multiai config` est un simple warning, pas une erreur bloquante (voir Securite #4)

### 16. Interface CLI

**Points forts :**
- Structure de commandes coherente (`multiai launch`, `multiai config`, `multiai list`, etc.)
- Completion shell pour 4 shells (bash, zsh, fish, powershell) avec contenus dynamiques (profiles list)
- `--help` sur chaque sous-commande
- `--json` disponible partout
- Tabwriter pour l'affichage table

**Points faibles :**

| # | Severite | Description | Fichier:Ligne |
|---|----------|-------------|---------------|
| 31 | ⚪ Faible | **Parsing manuel des flags** — pas de `flag` package. `getFlagValue()` ne supporte pas `--flag=VALUE` (seulement `--flag VALUE`). | `main.go:433-441` |
| 32 | ⚪ Faible | **Help text :** `multiai -V` affiche la version mais n'est pas documente dans le help. | `main.go:111-136` |
| 33 | ⚪ Faible | **Completion bash** — evalue `$(multiai list 2>/dev/null)` a chaque completion, ce qui peut etre lent. | `completion.go:18` |

### 17. Menu interactif

**Points forts :**
- Navigation intuitive (0 = retour/quit)
- Organisation claire en regions pour la configuration des fournisseurs
- Partage du reader entre le wizard et le menu (pas de perte de buffer)
- Confirmation explicite ("oui/OUI") avant les actions destructrices

**Points faibles :**

| # | Severite | Description | Fichier:Ligne |
|---|----------|-------------|---------------|
| 34 | ⚪ Faible | **Pas de scroll/recherche** — les menus avec beaucoup d'entrees (config, erase) n'ont pas de pagination ou de fonctionnalite de recherche. | `config/wizard.go` |

---

## SYNTHESE DES VULNERABILITES

### Par severite

| Severite | Denombrement |
|----------|-------------|
| 🔴 Critique | 0 |
| 🟡 Élevée | 2 |
| 🟢 Moyenne | 8 |
| ⚪ Faible | 14 |
| **Total** | **24** |

### Top 5 correctifs prioritaires

| Priorite | Vuln # | Titre | Impact |
|----------|--------|-------|--------|
| 1 | #7 | Injection via `os.ExpandEnv` apres echappement shell dans les hooks | Execution de code arbitraire via variables d'environnement |
| 2 | #28 | Fichier LICENSE absent du depot | Ambiguite legale |
| 3 | #4 | Degradation silencieuse en texte clair si credential store indisponible | Fuite de secrets API |
| 4 | #1 | Cle maitre stockee a cote du ciphertext | Dechiffrement possible avec acces fichier |
| 5 | #12-14 | Pas de signature, pas de SBOM, pas de codesign | Securite de la chaine de distribution |

### Forteresses identifiees

- **Sentinel pattern** : conception robuste de stockage des secrets
- **Journal de sessions** : schema securise par conception avec test de garde
- **Masquage des secrets** : IsSecretKey + MaskSecret systematiques
- **Zeroisation memoire** : apres decrypt des secrets
- **Atomic writes** : WriteFileAtomic partout (sauf cache OpenRouter)
- **Command whitelist** : seuls claude/codex/opencode autorises par defaut
- **Pas de telemetrie** : aucune collecte de donnees
- **Dependance unique** : surface d'attaque minimale

---

## ANNEXE : Fichiers lus (37 fichiers)

```
cmd/multiai/main.go                      cmd/multiai/cmd_openrouter.go
internal/assets/assets.go                internal/assets/assets_test.go
internal/catalog/catalog.go             internal/cli/completion.go
internal/cli/display.go                 internal/cli/fallback.go
internal/cli/fallback_test.go           internal/cli/hooks.go
internal/cli/launcher.go                internal/cli/resolve_test.go
internal/config/erase.go                internal/config/wizard.go
internal/config/wizard_test.go          internal/env/env.go
internal/env/env_test.go                internal/fsutil/atomic.go
internal/logging/logger.go              internal/logging/session.go
internal/logging/session_test.go        internal/menu/bmad.go
internal/menu/interactive.go            internal/onboarding/wizard.go
internal/openrouter/cache.go            internal/openrouter/client.go
internal/profile/profile.go             internal/profile/profile_test.go
internal/profile/project.go             internal/profile/yaml.go
internal/secret/crypto.go              internal/secret/secret.go
internal/secret/secret_test.go          internal/secret/store_darwin.go
internal/secret/store_linux.go          internal/secret/store_windows.go
pkg/dotenv/dotenv.go                    pkg/dotenv/dotenv_test.go
Makefile                                 go.mod
scripts/install.sh                      scripts/setup-go.sh
internal/assets/profiles/10-claude-anthropic-api.env (exemple)
```

**`go vet ./...` :** PROPRE (aucun avertissement)  
**`go test ./...` :** 16/16 paquets OK, 0 echec
