# Planification & Stories — multiai v0.4.3 → v0.5.0

**Date :** 2026-07-09
**Cible :** v0.5.0 (score 8.5/10)
**Source :** Audit complet 7 agents (5 Claude + 1 Codex adversarial + 1 sécurité)

---

## Résumé exécutif

| Sprint | Stories | Effort estimé | Score après |
|---|---|---|---|
| 🔴 Sprint 1 — BLOCKERS | 6 stories | ~5h | 7.5→8.0/10 |
| 🟠 Sprint 2 — HIGH | 8 stories | ~20h | 8.0→8.5/10 |
| 🟡 Sprint 3 — MEDIUM | 7 stories | ~25h | 8.5→9.0/10 |
| 🟢 Sprint 4 — LOW | 5 stories | ~20h | 9.0→9.5/10 |

---

# 🔴 Sprint 1 — BLOCKERS (avant toute release)

> **Objectif :** 0 vulnérabilité CRITICAL, 0 vulnérabilité HIGH bloquante
> **Gate de sortie :** CI verte, govulncheck propre, gitleaks clean, Cosign vérifié

---

## Story 1.1 — Rotation clé DeepSeek + suppression fichier sensible

**Priorité :** 🔴 CRITICAL
**Effort :** 5 min
**Source :** Audit sécurité C-1

### Contexte
Une clé API DeepSeek réelle (`sk-883d82d5eeaf4a8e88e33ca3e87ba330`) est présente en clair dans `brainstorm laurent/clé deepseek ne pas mettre dans le repo.txt`. Le fichier est dans `.gitignore` mais reste sur le disque, synchronisable par backup cloud.

### Ce qu'il faut faire
1. Aller sur https://platform.deepseek.com → révoquer la clé compromise
2. Créer une nouvelle clé API DeepSeek
3. Mettre à jour le credential store multiai avec la nouvelle clé
4. Supprimer le dossier `brainstorm laurent/` du disque
5. Vérifier qu'aucune autre clé n'est en clair sur le disque

### Commande de vérification
```bash
rg -n 'sk-[a-zA-Z0-9]{20,}' --glob '!.git' --glob '!node_modules' --glob '!log' --glob '!dist'
```

### Tests
- [ ] `rg` ne trouve plus aucune clé API réelle hors credential store
- [ ] La nouvelle clé DeepSeek fonctionne : `multiai launch -p ds`

### Résultat attendu
- 0 clé API en clair sur le disque
- Ancienne clé révoquée (ne peut plus être utilisée si volée)

---

## Story 1.2 — Hardcoder l'URL de l'API GitHub dans l'auto-update

**Priorité :** 🔴 CRITICAL
**Effort :** 30 min
**Fichier :** `multiai-go/internal/update/update.go`
**Source :** Audit sécurité C-2 + Codex critical finding

### Contexte
L'auto-update utilise `os.Getenv("MULTIAI_GITHUB_API_URL")` qui peut être détourné vers un serveur malveillant. Comme l'archive et `checksums.txt` viennent du même endpoint, la vérification SHA256 ne protège pas.

### Ce qu'il faut faire
1. Supprimer la variable d'environnement `MULTIAI_GITHUB_API_URL` comme source de l'URL
2. Hardcoder l'URL : `https://api.github.com/repos/lrochetta/multiai/releases/latest`
3. Garder une variable `MULTIAI_GITHUB_API_URL` utilisable UNIQUEMENT si `MULTIAI_DEV=1` est set (mode développement)
4. Si l'URL ne commence pas par `https://api.github.com`, refuser avec une erreur
5. Documenter le changement dans le commentaire de fonction

### Code attendu
```go
func FetchLatestRelease() (*Release, error) {
    apiURL := "https://api.github.com/repos/lrochetta/multiai/releases/latest"
    if devURL := os.Getenv("MULTIAI_GITHUB_API_URL"); devURL != "" {
        if os.Getenv("MULTIAI_DEV") != "1" {
            return nil, errors.New("MULTIAI_GITHUB_API_URL requires MULTIAI_DEV=1")
        }
        if !strings.HasPrefix(devURL, "https://api.github.com") {
            return nil, errors.New("MULTIAI_GITHUB_API_URL must start with https://api.github.com")
        }
        apiURL = devURL
    }
    // ... suite inchangée
}
```

### Tests
- [ ] `MULTIAI_GITHUB_API_URL` ignoré sans `MULTIAI_DEV=1`
- [ ] `MULTIAI_GITHUB_API_URL=https://evil.com/api` rejeté (pas `api.github.com`)
- [ ] `MULTIAI_DEV=1 MULTIAI_GITHUB_API_URL=https://api.github.com/repos/test/releases/latest` accepté
- [ ] Sans variable → utilise l'URL hardcodée

### Résultat attendu
- Impossible de détourner l'auto-update vers un serveur non-GitHub en production
- Mode dev préservé pour les tests

---

## Story 1.3 — Vérifier la signature Cosign dans l'auto-update

**Priorité :** 🔴 CRITICAL
**Effort :** 2h
**Fichier :** `multiai-go/internal/update/update.go`
**Source :** Audit sécurité C-2, Codex critical finding

### Contexte
Actuellement, `downloadAndVerifyRelease` vérifie uniquement le SHA256 de l'archive contre `checksums.txt`. Mais `checksums.txt` est sur le même serveur que l'archive → un attaquant peut remplacer les deux. La release GitHub inclut `checksums.txt.sig` et `checksums.txt.pem` (signature Cosign keyless) qui doivent être vérifiés AVANT le SHA256.

### Ce qu'il faut faire
1. Télécharger `checksums.txt`, `checksums.txt.sig`, `checksums.txt.pem` (pas seulement l'archive)
2. Vérifier la signature Cosign de `checksums.txt` avec :
   - Certificat : `checksums.txt.pem`
   - Signature : `checksums.txt.sig`
   - Identité : `https://github.com/lrochetta/multiai/.github/workflows/release.yml@refs/tags/v*`
   - Issuer OIDC : `https://token.actions.githubusercontent.com`
3. Si la vérification Cosign échoue → refuser la mise à jour (ne pas télécharger l'archive)
4. Vérifier le SHA256 de l'archive téléchargée contre `checksums.txt` (existant)
5. Option `MULTIAI_SKIP_COSIGN=true` pour bypass (développement uniquement)
6. Fallback : si `cosign` n'est pas installé, afficher un avertissement et proposer `multiai update --skip-cosign`

### Implémentation
- Méthode 1 (recommandée) : invoquer `cosign verify-blob` via `os/exec`
- Méthode 2 (fallback) : implémenter la vérification en Go pur avec `crypto/x509` (plus complexe mais pas de dépendance externe)
- Méthode 3 (hybride) : méthode 1 si `cosign` dans le PATH, sinon avertissement + fallback SHA256-only avec warning

### Tests
- [ ] Mock HTTP : archive + checksums valides, signature Cosign invalide → refus
- [ ] Mock HTTP : tout valide → mise à jour acceptée
- [ ] `MULTIAI_SKIP_COSIGN=true` : bypass explicite
- [ ] Absence de `cosign` dans PATH : avertissement affiché
- [ ] Release réelle v0.4.3 : vérification OK

### Résultat attendu
- Auto-update ne peut plus être détournée même avec contrôle du serveur
- Vérification cryptographique de bout en bout (Cosign → SHA256 → extraction)

---

## Story 1.4 — Ajouter un gate CI dans le workflow de release

**Priorité :** 🔴 CRITICAL
**Effort :** 1h
**Fichier :** `multiai-go/.github/workflows/release.yml`
**Source :** Codex high finding

### Contexte
Tout tag `v*` déclenche GoReleaser directement. Aucun test, lint, govulncheck ou vérification n'est exécuté avant la publication. Une release cassée ou malveillante serait signée et distribuée.

### Ce qu'il faut faire
1. Modifier le workflow `release.yml` pour qu'il dépende du workflow `ci.yml`
2. Ou : exécuter les étapes minimales suivantes dans `release.yml` avant GoReleaser :
   - `go test -race ./...` (tous les OS)
   - `go vet ./...`
   - `govulncheck ./...`
   - `golangci-lint run`
   - `gitleaks detect`
3. Vérifier que le tag est un descendant de `master` :
   ```bash
   git merge-base --is-ancestor ${{ github.ref }} origin/master
   ```
4. Exiger l'approbation d'un reviewer sur la release (environnement protégé GitHub)
5. Bloquer la release si la couverture est sous 30% sur un package critique

### Tests
- [ ] Push d'un tag sur une branche sans CI verte → release bloquée
- [ ] Push d'un tag non descendant de master → release bloquée
- [ ] Push d'un tag valide sur master avec CI verte → release OK
- [ ] Simulation : test qui échoue → GoReleaser non exécuté

### Résultat attendu
- Impossible de publier une release sans tests verts
- Protection contre les tags accidentels

---

## Story 1.5 — Installer gitleaks + pre-commit hook + job CI

**Priorité :** 🔴 CRITICAL
**Effort :** 1h
**Fichiers :** `.gitleaks.toml`, `.pre-commit-config.yaml`, `.github/workflows/ci.yml`
**Source :** Audit sécurité H-5

### Ce qu'il faut faire
1. Installer gitleaks dans le projet :
   ```bash
   go install github.com/gitleaks/gitleaks/v8@latest
   ```
2. Créer `.gitleaks.toml` avec :
   - Règles pour les patterns de clés API (OpenAI, Anthropic, DeepSeek, GitHub, HuggingFace)
   - Exclusion des fichiers de test (placeholders `PASTE_*_HERE`, `-test-`)
   - Exclusion du dossier `audit/` (contient des exemples de clés dans les rapports)
3. Ajouter un job `secret-scan` dans `ci.yml` :
   ```yaml
   secret-scan:
     runs-on: ubuntu-latest
     steps:
       - uses: actions/checkout@<SHA>
       - uses: gitleaks/gitleaks-action@v2
         with:
           config-path: .gitleaks.toml
   ```
4. Optionnel : configurer pre-commit hook local

### Tests
- [ ] `gitleaks detect` → 0 finding sur le code source
- [ ] Simulation : ajout d'une fausse clé `sk-test12345678901234567890` → gitleaks détecte
- [ ] `audit/` correctement exclu
- [ ] Fichiers de test avec `PASTE_*_HERE` non détectés

### Résultat attendu
- Scan de secrets à chaque push et PR
- Bloque le merge si une clé réelle est détectée

---

## Story 1.6 — Réparer le job smoke de la CI

**Priorité :** 🔴 CRITICAL
**Effort :** 5 min
**Fichier :** `.github/workflows/ci.yml`
**Source :** Codex medium finding

### Contexte
Le job `smoke` a un `working-directory: multiai-go` mais exécute `cd multiai-go` → il cherche `multiai-go/multiai-go` qui n'existe pas. Le build échoue silencieusement.

### Ce qu'il faut faire
1. Supprimer la ligne `cd multiai-go` du script smoke
2. Vérifier que le binaire est fonctionnel après build :
   ```yaml
   - run: go build -o multiai.exe ./cmd/multiai
   - run: ./multiai.exe version
   - run: ./multiai.exe list --json
   ```

### Tests
- [ ] Job smoke passe sur ubuntu, macos, windows
- [ ] `multiai version` retourne un code 0
- [ ] `multiai list --json` retourne du JSON valide

### Résultat attendu
- CI smoke fonctionnelle sur les 3 OS
- Preuve que le binaire démarre et répond

---

# 🟠 Sprint 2 — HIGH (avant v0.5.0)

> **Objectif :** 0 vulnérabilité HIGH, couverture > 50% sur tous les packages, fonctionnalités promises câblées
> **Gate de sortie :** Tests passent, `LoadAllProfiles` câblé, `--store` géré

---

## Story 2.1 — Tests pour `cmd/multiai` (0% → 50%)

**Priorité :** 🟠 HIGH
**Effort :** 2h
**Fichier :** `multiai-go/cmd/multiai/main_test.go` (nouveau)
**Source :** Audit qualité #1

### Ce qu'il faut tester

#### `getProfilesDir()`
| Test | Entrée | Résultat attendu |
|---|---|---|
| MULTIAI_PROFILES_DIR set | `MULTIAI_PROFILES_DIR=/custom` | `/custom` |
| DEV mode CWD | `MULTIAI_DEV=1`, pas de var | `configs/profiles` relatif au CWD |
| Sans DEV, sans var | rien | `<UserConfigDir>/multiai/profiles` |
| DEV non set, CWD profiles existe | `configs/profiles/` présent | NE PAS utiliser (sécurité) |

#### `hasFlag()` / `getFlagValue()` / `getExtraArgs()`
| Test | Entrée | Résultat |
|---|---|---|
| Flag présent | `["-p", "ds", "launch"]` | `hasFlag("-p")` → true |
| Flag absent | `["launch"]` | `hasFlag("-p")` → false |
| Valeur flag | `["-p", "ds"]` | `getFlagValue("-p")` → "ds" |
| Double flag | `["-p", "ds", "-p", "oc"]` | Dernier gagne → "oc" |
| Extra args après -- | `["launch", "--", "arg1", "arg2"]` | `["arg1", "arg2"]` |

#### `ensureProfiles()`
| Test | Entrée | Résultat |
|---|---|---|
| Dossier vide | 0 fichier | Extraction des 37 profils |
| 1 fichier existant | `00-claude-official.env` présent | Pas d'extraction, retour OK |
| Dossier verrouillé | Permission denied | Erreur + code sortie 2 |

### Objectif
- Couverture de `cmd/multiai` passe de 0% à ≥ 50%
- Toutes les fonctions de parsing et résolution sont testées

---

## Story 2.2 — Câbler `LoadAllProfiles` (YAML + hooks) dans le chemin de production

**Priorité :** 🟠 HIGH
**Effort :** 3h
**Fichiers :** `multiai-go/cmd/multiai/main.go`, `multiai-go/internal/profile/`
**Source :** Codex high finding, Audit architecture R5

### Contexte
`main.go` utilise `profile.LoadDir()` qui ne charge que les `.env`. Les profils `.yaml`/`.yml`, la config projet `.multiai.yaml` et les hooks `before_launch`/`after_launch` sont implémentés mais **jamais appelés en production**. Fonctionnalités documentées mais absentes.

### Ce qu'il faut faire
1. Dans `main.go:runLaunch()`, remplacer `profile.LoadDir()` par `profile.LoadAllProfiles()`
2. Transmettre les hooks à `cli.LaunchOptions` :
   ```go
   opts.BeforeHooks = prof.BeforeHooks
   opts.AfterHooks = prof.AfterHooks
   ```
3. Activer l'exécution des hooks dans `runLaunch()` (commentée actuellement) :
   ```go
   if len(opts.BeforeHooks) > 0 && !opts.DryRun {
       cli.RunBeforeHooks(opts.BeforeHooks, prof)
   }
   ```
4. Charger et fusionner la config projet `.multiai.yaml` :
   ```go
   projectConfig, _ := profile.FindProjectConfig()
   if projectConfig != nil {
       profile.MergeProjectConfig(prof, projectConfig)
   }
   ```
5. Logger un warning si YAML ou hooks sont présents mais `NO_HOOKS=1`

### Tests
- [ ] `multiai list` affiche les profils YAML (pas seulement .env)
- [ ] `multiai launch -p test-yaml` lance un profil défini en YAML
- [ ] Hook `before_launch` exécuté avant le processus enfant
- [ ] Hook `after_launch` exécuté après (même si enfant échoue)
- [ ] `.multiai.yaml` dans un dossier parent → config fusionnée
- [ ] `MULTIAI_NO_HOOKS=1` désactive les hooks
- [ ] Test E2E : création profil YAML → lancement → hook exécuté

### Résultat attendu
- Fonctionnalités promises dans le README enfin opérationnelles
- Les hooks permettent des cas d'usage réels (VPN check, notification, cleanup)

---

## Story 2.3 — Gérer explicitement `--store` (implémenter ou refuser)

**Priorité :** 🟠 HIGH
**Effort :** 4h (si implémentation Windows) ou 30 min (si refus explicite)
**Fichiers :** `multiai-go/internal/secret/store_windows.go`, `multiai-go/cmd/multiai/main.go`
**Source :** Codex high finding, Audit sécurité H-4

### Contexte
- `config --store keychain|wincred|secret-service` est accepté mais **ignoré** : le wizard normal s'ouvre
- Les 3 fichiers `store_windows.go`, `store_darwin.go`, `store_linux.go` sont des **stubs identiques** qui délèguent au `encryptedFileStore`
- L'utilisateur pense utiliser le store natif mais utilise le fichier AES sans le savoir
- La clé maître AES est stockée à côté des ciphertexts

### Option A : Implémentation Windows Credential Manager (recommandé)
1. Utiliser `golang.org/x/sys/windows` pour appeler `CredReadW`/`CredWriteW`/`CredDeleteW`
2. Implémenter `windowsCredStore` qui satisfait `secret.Store`
3. La clé maître AES est stockée dans Windows Credential Manager
4. Les ciphertexts restent sur disque (ne peuvent pas être déchiffrés sans Credential Manager)
5. Garder `encryptedFileStore` comme fallback si Credential Manager échoue

### Option B : Refus explicite (pragmatique)
1. Modifier `config --store` pour afficher :
   ```
   [i] Le backend natif n'est pas encore implémenté.
       Le store chiffré par fichier (AES-256-GCM) est utilisé.
       Suivez https://github.com/lrochetta/multiai/issues/X
   ```
2. Retourner `exit 0` (pas d'erreur) mais logger le vrai backend utilisé
3. Mettre à jour `SECURITY.md` pour documenter l'état réel

### Tests
- [ ] `multiai config --store wincred` → message clair + fallback fichier (Option B)
- [ ] `multiai config --store keychain` → idem macOS
- [ ] Le secret reste stocké et récupérable
- [ ] `multiai config --store invalid` → erreur + exit ≠ 0

### Résultat attendu
- L'utilisateur sait exactement quel backend est utilisé
- Pas de promesse non tenue silencieuse

---

## Story 2.4 — Corriger le namespace `ServiceForProfile` (vol de secret par profil homonyme)

**Priorité :** 🟠 HIGH
**Effort :** 1h
**Fichier :** `multiai-go/internal/secret/secret.go`
**Source :** Codex high finding

### Contexte
`ServiceForProfile` dérive le nom du service à partir du **basename** du fichier profil. Si deux profils ont le même nom dans des répertoires différents (ex: `~/.multiai/profiles/00-claude-official.env` et `/tmp/evil/00-claude-official.env`), ils partagent le même secret. Un attaquant peut créer un profil homonyme qui récupère le secret légitime.

### Ce qu'il faut faire
1. Modifier `ServiceForProfile` pour inclure un hash du chemin canonique :
   ```go
   func ServiceForProfile(profilePath string) string {
       abs, _ := filepath.Abs(profilePath)
       canonical := filepath.Clean(abs)
       h := sha256.Sum256([]byte(canonical))
       base := strings.TrimSuffix(filepath.Base(canonical), filepath.Ext(canonical))
       return fmt.Sprintf("%s-%x", base, h[:8])
   }
   ```
2. Ajouter une migration automatique : si un secret existe sous l'ancien nom, le migrer vers le nouveau
3. Refuser la résolution de sentinel pour les profils hors des racines de confiance (`MULTIAI_PROFILES_DIR`, `<user config>`) sans confirmation

### Tests
- [ ] Deux profils homonymes dans des dossiers différents → services différents
- [ ] Même profil après migration → même service
- [ ] Migration auto : ancien nom → nouveau nom
- [ ] Profil hors racine de confiance → avertissement

### Résultat attendu
- Impossible de voler un secret via un profil homonyme
- Rétrocompatibilité avec les secrets existants

---

## Story 2.5 — Transactionalité store/sentinelle (race condition)

**Priorité :** 🟠 HIGH
**Effort :** 4h
**Fichiers :** `multiai-go/internal/secret/`, `multiai-go/internal/config/`
**Source :** Codex high finding

### Contexte
- Deux processus multiai simultanés peuvent corrompre le store : `Get` + `Set` ne sont pas atomiques
- Le wizard efface le secret AVANT de remplacer la sentinelle → crash = perte irrécupérable
- `encryptedFileStore` lit tout le fichier, modifie une entrée, réécrit tout → deux processus s'écrasent

### Ce qu'il faut faire
1. **Une entrée par clé** : remplacer le fichier unique chiffré par un fichier par clé (dossier `secrets/` avec `sk-ant-*`, `sk-or-*`, etc.)
2. **Verrou inter-processus** : utiliser `flock` (Linux/macOS) / `LockFileEx` (Windows) sur le fichier
3. **Transaction store + profil** :
   - Sauvegarder le secret dans le store
   - Écrire la sentinelle dans le `.env` (atomique, déjà fait)
   - Si l'écriture .env échoue → rollback (supprimer le secret du store)
   - Si le store échoue → ne pas toucher au .env
4. **Ne jamais effacer avant d'avoir écrit** : l'ordre doit être CREATE NEW → REPLACE OLD → DELETE OLD

### Tests
- [ ] Test multi-processus : 10 lancements simultanés → 0 corruption
- [ ] Test crash au milieu de l'écriture → secret préservé
- [ ] Test disque plein → erreur, état cohérent
- [ ] Test `Ctrl+C` pendant l'écriture → rollback

### Résultat attendu
- Aucune perte de secret possible
- Tolérance aux pannes (crash, disque plein, concurrence)

---

## Story 2.6 — Migration versionnée `ensureProfiles` (nouveaux profils après upgrade)

**Priorité :** 🟠 HIGH
**Effort :** 2h
**Fichier :** `multiai-go/cmd/multiai/main.go`
**Source :** Codex high finding

### Contexte
`ensureProfiles` retourne dès qu'un seul `.env` existe. Après une mise à jour (v0.4.3 → v0.5.0), les nouveaux profils embarqués ne sont **jamais** extraits. L'utilisateur ne voit pas les nouveaux fournisseurs.

### Ce qu'il faut faire
1. Créer un fichier manifeste `profiles.json` dans le dossier de profils :
   ```json
   {"version": "0.4.3", "profiles": {"00-claude-official.env": "sha256:abc123...", ...}}
   ```
2. À chaque démarrage, comparer le manifeste embarqué (binaire) avec le manifeste installé
3. Extraire uniquement les nouveaux profils ou ceux modifiés
4. **Ne jamais écraser** un profil modifié par l'utilisateur (comparer le SHA256)
5. Profils supprimés volontairement → créer un tombstone `.00-deleted.env.removed` pour ne pas les ré-extraire

### Tests
- [ ] Première installation → 37 profils extraits + manifeste créé
- [ ] Upgrade avec 2 nouveaux profils → seuls les 2 nouveaux extraits
- [ ] Profil modifié par l'utilisateur → non écrasé (SHA différent)
- [ ] Profil supprimé volontairement → non ré-extrait (tombstone)
- [ ] Rollback : ancienne version → manifeste plus récent → avertissement

### Résultat attendu
- Les utilisateurs reçoivent les nouveaux profils à chaque upgrade
- Les modifications utilisateur sont préservées

---

## Story 2.7 — Extraire `display/` de `cli/` (couplage)

**Priorité :** 🟠 HIGH
**Effort :** 2h
**Fichiers :** `multiai-go/internal/cli/display.go` → `multiai-go/internal/display/display.go`
**Source :** Audit architecture R1

### Contexte
`config`, `menu`, `onboarding` importent `cli` **uniquement** pour les helpers d'affichage (`PrintWarning`, `PrintError`, `PrintSuccess`, `PrintInfo`, `Colorize`, `StatusColor`). Ça crée un couplage artificiel "métier → orchestration".

### Ce qu'il faut faire
1. Créer `internal/display/` avec les fonctions copiées depuis `cli/display.go` :
   - `PrintSuccess()`, `PrintWarning()`, `PrintError()`, `PrintInfo()`
   - `Colorize()`, `StatusColor()`, `MaskSecret()`
2. Remplacer les imports de `cli.PrintX` par `display.PrintX` dans :
   - `internal/config/wizard.go`, `internal/config/erase.go`
   - `internal/menu/interactive.go`, `internal/menu/bmad.go`
   - `internal/onboarding/wizard.go`
3. Garder des alias dans `cli/display.go` en dépréciation (pour `main.go`) ou migrer `main.go` aussi
4. Supprimer les alias après migration complète

### Tests
- [ ] `go build ./...` passe
- [ ] `go test ./...` passe (tous les tests qui utilisent `Print*`)
- [ ] Les imports de `cli` dans `config/menu/onboarding` ne contiennent plus `Print*`

### Résultat attendu
- `config`, `menu`, `onboarding` ne dépendent plus de `cli`
- Graphe d'imports plus propre

---

## Story 2.8 — Unifier les écritures atomiques vers `fsutil.WriteFileAtomic`

**Priorité :** 🟠 HIGH
**Effort :** 1h
**Fichiers :** `multiai-go/internal/fsutil/`, `multiai-go/internal/openrouter/`, `multiai-go/internal/config/`
**Source :** Audit architecture R2

### Contexte
Le pattern temp-file + fsync + rename est implémenté 4 fois différemment :
- `fsutil.WriteFileAtomic()` — la version canonique
- `config/setEnvVarInFile()` — son propre temp file avec `os.CreateTemp`
- `secret/save()` — via `fsutil.WriteFileAtomic` (OK)
- `openrouter/SaveCache()` — nom de temp fixe `path + ".tmp"` (dangereux en concurrence)

### Ce qu'il faut faire
1. Remplacer le temp file dans `config/setEnvVarInFile()` par un appel à `fsutil.WriteFileAtomic`
2. Remplacer le temp file fixe dans `openrouter/SaveCache()` par un appel à `fsutil.WriteFileAtomic`
3. Vérifier qu'aucun autre appel ne duplique le pattern
4. Ajouter un test direct pour `WriteFileAtomic` (cf Story 3.2)

### Tests
- [ ] `config/setEnvVarInFile` → le fichier est écrit atomiquement (pas de fichier temporaire résiduel)
- [ ] `openrouter/SaveCache` → idem + test de concurrence
- [ ] `secret/save` → déjà OK, juste vérifier

### Résultat attendu
- 1 seule implémentation d'écriture atomique
- Plus de temp file fixe (race condition)

---

# 🟡 Sprint 3 — MEDIUM (v0.6.0)

> **Objectif :** Documentation, tests, distribution, DX
> **Gate de sortie :** docs/ en ligne, Homebrew fonctionnel, couverture > 70%

---

## Story 3.1 — Documentation utilisateur dans `docs/` (VitePress)

**Priorité :** 🟡 MEDIUM
**Effort :** 3h
**Fichiers :** `multiai-go/docs/`
**Source :** Audit DX #1

### Pages à créer
```
docs/
├── index.md                      ← Landing page
├── guide/
│   ├── getting-started.md        ← Installation + premier lancement
│   ├── profiles.md               ← Format .env et .yaml
│   ├── configuration.md          ← Configurer les clés API
│   ├── fallback.md               ← Chaînes de fallback
│   └── troubleshooting.md        ← Erreurs courantes
├── reference/
│   ├── commands.md               ← Toutes les sous-commandes
│   ├── env-variables.md          ← Variables d'environnement
│   ├── providers.md              ← Catalogue des fournisseurs
│   └── exit-codes.md             ← Codes de sortie
├── advanced/
│   ├── custom-profiles.md        ← Profils .env et YAML
│   ├── project-config.md         ← .multiai.yaml
│   ├── hooks.md                  ← before_launch / after_launch
│   └── credential-store.md       ← Architecture AES-256-GCM
└── security/
    ├── threat-model.md           ← Modèle de menace
    └── supply-chain.md           ← Cosign, SBOM, attestation
```

### Résultat attendu
- Site VitePress déployé sur GitHub Pages
- Chaque page couvre un sujet spécifique
- Exemples de code fonctionnels

---

## Story 3.2 — Tests pour les packages à 0%

**Priorité :** 🟡 MEDIUM
**Effort :** 4h
**Fichiers :** nouveaux `*_test.go`

### Packages cibles
| Package | Tests minimum |
|---|---|
| `internal/fsutil` | `WriteFileAtomic` : succès, permission denied, disque plein, crash |
| `internal/menu` | `ShowTopMenu`, `SelectTool`, `SelectProfile` avec stdin mocké |
| `internal/logging` | `Logger.Log/Debug/Info/Warn/Error`, niveaux, rotation |
| `internal/cli` | Hooks, completion, `escapeShellArg` |

### Résultat attendu
- Plus aucun package à 0%
- Couverture globale > 60%

---

## Story 3.3 — `multiai update` explicite (remplacer l'auto-update silencieux)

**Priorité :** 🟡 MEDIUM
**Effort :** 2h
**Fichier :** `multiai-go/cmd/multiai/cmd_update.go` (nouveau)
**Source :** Audit DX #5

### Ce qu'il faut faire
1. Créer la sous-commande `multiai update`
2. Comportement :
   - Affiche version courante et version disponible
   - Affiche la taille du téléchargement
   - Vérifie Cosign + SHA256 (affiché)
   - Demande confirmation
   - Télécharge et remplace le binaire
3. Changer `Check()` dans `main()` pour afficher un message non-bloquant :
   ```
   [i] v0.5.0 disponible (vous avez v0.4.3). Lancez 'multiai update'.
   ```
   Au lieu de télécharger silencieusement.
4. Ajouter `multiai update --check` (vérifie sans installer)
5. Ajouter `multiai update --yes` (non-interactif, pour CI/scripts)

### Résultat attendu
- L'utilisateur contrôle quand mettre à jour
- Plus de remplacement silencieux du processus

---

## Story 3.4 — Homebrew tap + Scoop bucket (`skip_upload: false`)

**Priorité :** 🟡 MEDIUM
**Effort :** 3h
**Fichiers :** `.goreleaser.yaml`
**Source :** Audit supply chain

### Ce qu'il faut faire
1. Créer le repo `github.com/lrochetta/homebrew-tap`
2. Créer le repo `github.com/lrochetta/scoop-bucket`
3. Configurer les tokens dans les secrets GitHub :
   - `TAP_GITHUB_TOKEN` (classic PAT avec `public_repo`)
4. Passer `skip_upload: false` dans `.goreleaser.yaml`
5. Tester : `goreleaser release --snapshot --clean` → vérifier les fichiers dans `dist/`
6. Valider la formule Homebrew : `brew audit --strict dist/homebrew/Casks/multiai.rb`
7. Valider le manifest Scoop : `scoop checkver dist/scoop/multiai.json`

### Résultat attendu
- `brew install lrochetta/homebrew-tap/multiai` fonctionnel
- `scoop bucket add multiai https://github.com/lrochetta/scoop-bucket && scoop install multiai` fonctionnel

---

## Story 3.5 — SBOM CycloneDX dans les releases

**Priorité :** 🟡 MEDIUM
**Effort :** 1h
**Fichier :** `.github/workflows/release.yml`
**Source :** Audit supply chain

### Ce qu'il faut faire
```yaml
- uses: anchore/sbom-action@v0
  with:
    path: ./multiai-go
    format: cyclonedx-json
- name: Upload SBOM
  uses: actions/upload-release-asset@v1
  with:
    upload_url: ${{ steps.create_release.outputs.upload_url }}
    asset_path: ./sbom.cyclonedx.json
    asset_name: multiai-${{ github.ref_name }}-sbom.json
    asset_content_type: application/json
```

### Résultat attendu
- `multiai-v0.5.0-sbom.json` dans chaque release
- Liste complète des dépendances (1 directe + 1 transitive)

---

## Story 3.6 — `CONTRIBUTING.md` + templates GitHub

**Priorité :** 🟡 MEDIUM
**Effort :** 1h
**Fichiers :** `CONTRIBUTING.md`, `.github/ISSUE_TEMPLATE/`, `.github/PULL_REQUEST_TEMPLATE.md`

### Contenu de CONTRIBUTING.md
- Environnement de dev : Go 1.22+, `go build ./cmd/multiai`, `go test ./...`
- Conventions : Conventional Commits, `gofmt`, `go vet`, tests obligatoires
- Process : fork → branch → PR → review → merge
- Guide de release : goreleaser, npm publish, Homebrew

### Templates
- `bug_report.md` : version, OS, comportement attendu, comportement observé
- `feature_request.md` : problème, solution proposée, alternatives
- `PULL_REQUEST_TEMPLATE.md` : checklist (tests, lint, changelog)

---

## Story 3.7 — `context.Context` dans les appels HTTP

**Priorité :** 🟡 MEDIUM
**Effort :** 2h
**Fichiers :** `update/update.go`, `openrouter/client.go`
**Source :** Audit architecture R4

### Ce qu'il faut faire
1. Ajouter `ctx context.Context` en paramètre de :
   - `FetchLatestRelease(ctx context.Context)`
   - `FetchModels(ctx context.Context)`
   - `fetchRaw(ctx context.Context, ...)`
2. Utiliser `req.WithContext(ctx)` dans les requêtes HTTP
3. Propager depuis `main()` avec `context.WithTimeout(30 * time.Second)`
4. Gérer `ctx.Err()` dans les boucles de retry

---

# 🟢 Sprint 4 — LOW (backlog)

> **Objectif :** Internationalisation, visibilité, polish
> **Gate de sortie :** 50+ stars GitHub, i18n EN fonctionnel

---

## Story 4.1 — Internationalisation anglais (framework minimal)

**Priorité :** 🟢 LOW
**Effort :** 4h
**Fichiers :** `multiai-go/internal/i18n/` (nouveau)

### Approche
1. Externaliser les messages utilisateur dans un fichier JSON (ou Go map)
2. Détecter la langue via `MULTIAI_LANG` ou `LANG` (fallback: `fr`)
3. Traduire en priorité : erreurs, menus, help, onboarding
4. Les commentaires de code et noms techniques restent en anglais

---

## Story 4.2 — Fuzz testing (.env, YAML)

**Priorité :** 🟢 LOW
**Effort :** 3h
**Fichiers :** `pkg/dotenv/fuzz_test.go`, `internal/profile/yaml_fuzz_test.go`

### Ce qu'il faut faire
```go
func FuzzParse(f *testing.F) {
    f.Add("KEY=value")
    f.Fuzz(func(t *testing.T, input string) {
        p, _ := Parse(strings.NewReader(input))
        // Ne doit jamais paniquer
        _ = p.Get("KEY")
    })
}
```

---

## Story 4.3 — Mettre à jour `go.mod` → go 1.24 + migrer yaml

**Priorité :** 🟢 LOW
**Effort :** 1h
**Fichiers :** `go.mod`, `internal/catalog/catalog.go`, `internal/profile/yaml.go`, `internal/profile/project.go`

### Ce qu'il faut faire
1. `go mod edit -go=1.24`
2. Remplacer `gopkg.in/yaml.v3` par `github.com/yaml/go-yaml` (API compatible)
3. `go test ./...`

---

## Story 4.4 — Script de lancement direct par shortcut

**Priorité :** 🟢 LOW
**Effort :** 2h
**Fichier :** `scripts/generate-wrappers.sh`

### Ce qu'il faut faire
Générer automatiquement des wrappers par shortcut (comme les `.cmd` PowerShell mais cross-platform) :
```bash
#!/bin/bash
# multiai-ds → multiai launch -p ds
exec multiai launch -p ds "$@"
```

---

## Story 4.5 — Publication & visibilité

**Priorité :** 🟢 LOW
**Effort :** 3h
**Source :** Audit stratégie #1

### Actions
1. Publier sur Hacker News ("Show HN: multiai — routeur multi-IA avec isolation sécurisée")
2. Poster sur r/golang, r/programming, r/commandline
3. Contacter les newsletters Go Weekly, Console.dev, TLDR Newsletter
4. Ajouter un badge "Made with Go" + "Cosign Signed" dans le README
5. Créer un compte Twitter/X pour le projet

### KPI
- 50+ stars GitHub
- 3+ issues de vrais utilisateurs

---

# Synthèse par rôle

## Pour l'agent DEV (Forge/Claude Code)
Stories : 1.2, 1.3, 1.4, 1.6, 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 2.8, 3.2, 3.3, 3.5, 3.7, 4.2, 4.3, 4.4
**Priorité :** Commencer par Sprint 1 (BLOCKERS) puis Sprint 2

## Pour l'agent QA (Sentinel)
Stories : 1.1, 1.5, 2.1 (tests), 3.2, 4.2
**Focus :** gitleaks, rotation clés, couverture de tests

## Pour l'agent DOCS (Huldah)
Stories : 3.1, 3.6, 4.1
**Focus :** VitePress, CONTRIBUTING.md, i18n

## Pour l'agent OPS (Nexus)
Stories : 1.4, 1.6, 3.4, 3.5, 4.5
**Focus :** CI/CD, Homebrew/Scoop, SBOM, visibilité

---

## Dépendances entre stories

```
1.2 ──→ 1.3 (hardcoded URL avant Cosign)
1.4 ──→ dépendant de 1.5 (gitleaks) et 1.6 (smoke fix)
2.1 ──→ prérequis pour 2.2 (tests avant câblage YAML/hooks)
2.3 ──→ prérequis pour 2.5 (store natif avant transactionnalité)
2.7 ──→ prérequis pour 3.2 (display extrait avant tests menu)
3.4 ──→ dépend de 1.4 (CI gate avant distribution)
```

---

**Prochaine action :** `Story 1.1` (rotation clé DeepSeek) — 5 min, peut être faite immédiatement.
