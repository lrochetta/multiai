# Audit de Securite -- Scan des Secrets (Repo Public)

**Date :** 2026-07-05  
**Perimetre :** `lrochetta/multiai` (passage en public)  
**Scanneurs :** 4 (fichiers .env/config, code source, historique git, CI/npm/scripts)  
**Score global :** 8/10 (1 CRITICAL, 1 MEDIUM, 2 LOW, reste OK)

---

## RESUME EXECUTIF

| Scanner | Statut | Trouvailles |
|---|---|---|
| 1. Fichiers .env et config | TERMINE | 1 CRITICAL, reste OK |
| 2. Code source (.go, .ps1, .js, .sh) | TERMINE | Aucun secret expose |
| 3. Historique git | TERMINE | Aucun secret dans l'historique |
| 4. Workflows CI, npm, scripts | TERMINE | 1 MEDIUM, 1 LOW |

**Aucune cle API exposee dans les fichiers suivis par git.** L'unique cle reelle trouvee est dans un fichier ignore par .gitignore (jamais commite).

---

## TROUVAILLE CRITIQUE

### S-1 : Cle API DeepSeek en clair sur disque

| Champ | Valeur |
|---|---|
| **Severite** | **CRITICAL** |
| **Fichier** | `brainstorm laurent/clé deepseek ne pas mettre dans le repo.txt:1` |
| **Contenu** | `sk-xxxx...xxxx` (36 car., prefixe `sk-` + 32 hex, masquee dans le rapport) |
| **Statut git** | Ignore via `.gitignore` (`/brainstorm*`). JAMAIS commitee. |
| **Type** | Cle API DeepSeek valide |

**Risque :** Le fichier est en clair dans l'arborescence du depot. Bien qu'ignore par git, il pourrait etre :
- Capture par un `git add -f` ou `git add --all` force
- Synchronise par un service de cloud backup (Dropbox, Google Drive, etc.)
- Accessible a un tiers ayant un acces local a la machine

**Correctif :**
1. **ROTATION IMMEDIATE** de la cle sur https://platform.deepseek.com/api_keys
2. **SUPPRIMER** le fichier du disque
3. Stocker la cle dans un gestionnaire de mots de passe (Bitwarden, 1Password, etc.)
4. Ne jamais stocker de secrets dans l'arborescence du depot, meme ignores

---

## TROUVAILLES MOYENNES

### S-2 : AUR PKGBUILD avec sha256sums=('SKIP')

| Champ | Valeur |
|---|---|
| **Severite** | **MEDIUM** |
| **Fichier** | `multiai-go/packaging/aur/PKGBUILD:16` |
| **Contenu** | `sha256sums=('SKIP')` |
| **Description** | La verification SHA256 du tarball source est desactivee. Si publie sur l'AUR avec SKIP, un attaquant pourrait remplacer le tarball source sans detection. |

**Fichier concerne aussi :**
- `multiai-go/packaging/aur/.SRCINFO:11` : `sha256sums = SKIP`

**Correctif :**
- Remplacer `SKIP` par un vrai checksum avant publication (script `scripts/update-aur-checksums.sh` existe)
- Ajouter un guard CI ou pre-commit hook qui refuse `SKIP` dans les fichiers AUR
- Generer `.SRCINFO` via `makepkg --printsrcinfo` apres correction

---

## TROUVAILLES BASSE

### S-3 : Version hardcodee dans go-build.ps1

| Champ | Valeur |
|---|---|
| **Severite** | **LOW** |
| **Fichier** | `multiai-go/go-build.ps1:60` |
| **Contenu** | `go build -ldflags="-s -w -X main.version=0.2.0" -o $out ./cmd/multiai/` |
| **Description** | Version `0.2.0` hardcodee alors que le projet est en `0.4.0-dev`. Les builds avec ce script auront une version erronee. |

**Correctif :** Remplacer par `main.version=$env:VERSION` ou synchroniser avec la version du projet.

### S-4 : Nom de fichier corrompu dans l'historique git (commits 72c862c / 37b9898)

| Champ | Valeur |
|---|---|
| **Severite** | **LOW** |
| **Description** | Un fichier `.env` avec nom corrompu (caracteres Unicode de controle + commande `cp` inline) a ete introduit puis supprime dans les premiers commits. Le contenu etait un template placeholder, pas de vrai secret. |
| **Statut** | Fichier deja supprime dans `70eb802` (Zecher memory cleanup). |
| **Correctif** | Aucune action necessaire. |

---

## ZONES VERIFIEES -- OK

### Profils .env (tous templates avec placeholders)

| Zone | Fichiers | Placeholders |
|---|---|---|
| `multiai-go/internal/assets/profiles/` | 37 profils | `PASTE_..._HERE`, `%VAR%`, lignes commentees |
| `multiai-go/configs/profiles/` | ~24 profils (non suivis) | Placeholders uniquement |
| `multiai-powershell/configs/profiles/` | ~52 profils (non suivis) | Placeholders uniquement |
| `_local-archive/sprint1-snapshot/` | 17 profils archive | Placeholders uniquement |
| Racine du repo (60/61/62-*.env) | 3 fichiers | `PASTE_OPENROUTER_API_KEY_HERE` |
| `multiai-go/internal/assets/profiles/` | Aucun `__MULTIAI_CREDSTORE__` | Conforme |

### Fichiers de test Go -- fausses cles (acceptables)

Les fichiers `*_test.go` contiennent des cles factices :
- `internal/secret/secret_test.go:15` : `sk-ant-api-03-test-secret-key-123456`
- `internal/config/wizard_test.go` : `sk-ant-api03-wizard-test-*`
- `internal/env/env_test.go` : `sk-ant-test123`
- `internal/onboarding/wizard_test.go:45` : `sk-abcdef1234567890`
- `pkg/dotenv/dotenv_test.go` : `sk-xxxx`, `sk-ant-api-03-abc123def456`
- `tests/integration_test.go:23` : `sk-ant-test123`

**Verdict :** Valeurs de test inoffensives (pattern `-test-`, `abcdef`, `xxxx`).  
**Recommandation (INFO) :** Renommer avec prefixe explicite `TEST_SK_` pour eviter faux positifs de scanners automatises.

### Fichiers .yaml, .json, .toml (aucun secret)

- `internal/catalog/providers.yaml` : Noms de variables uniquement, pas de secrets
- `.github/workflows/*.yml` : `${{ secrets.GITHUB_TOKEN }}` -- reference correcte
- `.goreleaser.yaml` : `{{ .Env.TAP_GITHUB_TOKEN }}` -- variable d'env (commentaire, code desactive)
- `internal/openrouter/testdata/models.json` : Donnees de test uniquement
- `.claude/settings.local.json` : Permissions d'outils uniquement

### Workflows GitHub Actions

- **Actions toutes epinglees par SHA commit** (audit v0.2.1, finding #14)
- `GITHUB_TOKEN` utilise correctement via `${{ secrets.GITHUB_TOKEN }}`
- Permissions explicites : `contents: read` sur CI, `contents: write` sur release uniquement
- Cosign keyless signing avec `id-token: write`
- Scan de securite : gosec + govulncheck dans le pipeline CI

### Scripts d'installation et packaging

- `packaging/npm/install.js` : SHA256 verification du binaire, pas de token hardcode
- `packaging/npm/scan-secrets.js` : Regex couvrant tous les formats de cles, bloque publication npm si vrai secret detecte
- `scripts/install.sh` : SHA256 verification, pas de token hardcode
- `scripts/setup-go.sh`, `scripts/setup-go.ps1` : Pas de secrets

### Code source (.go, .ps1, .js, .sh)

- Aucune cle API hardcodee dans le code
- Les cles sont toujours lues via `os.Getenv()` ou stockees dans le credential store AES-256-GCM
- Le credential store (`internal/secret/`) utilise AES-256-GCM avec derive PBKDF2

### Historique git

- 17 commits analyses (78d4705 a d7dfd51)
- Aucune cle API reelle trouvee dans l'historique
- Aucun fichier `.env` supprime (qui aurait pu contenir des secrets)
- `.gitignore` present des le premier commit, excluant `*.env`
- Commit v0.4.0 (f8e8923) : 4 correctifs de securite appliques

---

## STATUT .gitignore

Le fichier `.gitignore` exclut correctement :
- `*.env` (tous les fichiers .env)
- `configs/profiles/*.env`
- `.credentials/`
- `log/`
- `multiai-go/dist/`
- `multiai-go/build/`
- `node_modules/`
- `_bmad/`, `_bmad-output/`, `_local*`, etc.

Exception explicite pour les templates :
- `!multiai-go/internal/assets/profiles/*.env` (whitelist des templates suivis)

---

## TABLEAU RECAPITULATIF

| ID | Fichier | Severite | Type | Correctif |
|---|---|---|---|---|
| **S-1** | `brainstorm laurent/cle deepseek ne pas mettre dans le repo.txt:1` | **CRITICAL** | Cle DeepSeek en clair (`sk-xxxx...xxxx`) — rotation recommandee | Rotation + suppression du fichier |
| **S-2** | `multiai-go/packaging/aur/PKGBUILD:16` | **MEDIUM** | sha256sums=('SKIP') vulnerable supply-chain | Remplacer par vrai checksum |
| **S-3** | `multiai-go/go-build.ps1:60` | LOW | Version hardcodee 0.2.0 | Synchroniser avec version projet |
| **S-4** | Historique git (commit 37b9898) | LOW | Fichier .env corrompu (deja supprime) | Aucun |
| S-5 | Profils .env (130+ fichiers) | INFO | Placeholders OK | Aucun |
| S-6 | Tests Go (10+ fichiers) | INFO | Cles de test factices | Renommer avec prefixe TEST_ |
| S-7 | Workflows CI | INFO | Secrets references correctement | Aucun |
| S-8 | .gitignore | INFO | Patterns secrets exclus | Aucun |
| S-9 | Credential store | INFO | AES-256-GCM + PBKDF2 | Aucun |

---

## RECOMMANDATIONS PRIORITAIRES

1. **[URGENT]** Rotation immediate de la cle DeepSeek sur https://platform.deepseek.com/api_keys
2. **[Urgent]** Suppression du fichier `brainstorm laurent/cle deepseek ne pas mettre dans le repo.txt`
3. **[Avant publication AUR]** Remplacer `sha256sums=('SKIP')` par un vrai checksum dans PKGBUILD
4. **[Faible priorite]** Ajouter `TEST_` prefix aux cles factices dans les fichiers de test pour eviter faux positifs de scanners automatises
5. **[Faible priorite]** De-traquer les 3 fichiers `.env` de la racine (`60-claude-fusion.env`, `61-codex-fusion.env`, `62-opencode-fusion.env`) avec `git rm --cached` et les deplacer dans `configs/profiles/` ou `docs/examples/`. Bien qu'ils contiennent actuellement des placeholders, le fait qu'ils soient traques dans git presente un risque si quelqu'un y ecrit une vraie cle par erreur.
6. **[Faible priorite]** Etendre `scan-secrets.js` pour couvrir aussi les repertoires `scripts/` et les fichiers `*.ps1` (actuellement limite aux dossiers de profils .env)
7. **[Continu]** Executer un scanner automatise (truffleHog, GitLeaks, Gitleaks) avant chaque release

---

*Rapport genere par audit multi-agent BMAD+ (Scanner 1-4)*
