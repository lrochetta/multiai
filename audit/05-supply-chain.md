# Audit Supply-Chain — multiai

**Date :** 2026-07-05
**Repo :** `lrochetta/multiai` (public)
**Auditeur :** Agent Securite Supply-Chain

---

## 1. Dependances vulnerables

### 1.1 Go — `go.mod`

| Dependance | Version | Statut |
|---|---|---|
| `gopkg.in/yaml.v3` | v3.0.1 | Aucune CVE connue |

**Verification :**
- `govulncheck ./...` retourne : **"No vulnerabilities found."**
- CVE-2022-28948 (DoS via Unmarshal) et SNYK-GOLANG-GOPKGINYAMLV3-2952714 (NULL pointer dereference) sont **corrigees dans v3.0.1** — la version utilisee est bien la version patchee.

**Alerte :** Le depot officiel `gopkg.in/yaml.v3` a ete archive en avril 2024. Il ne recevra plus de correctifs de securite. Bien que v3.0.1 soit sain aujourd'hui, toute CVE future ne sera pas corrigee. Recommandation : migrer vers `github.com/go-yaml/yaml` (fork maintenu).

### 1.2 npm — docs (`multiai-go/docs/`)

| Dependance | Version | Risque |
|---|---|---|
| `vitepress` | `^1.0.0` (dev) | Plage large, pas de lockfile |

**Alerte :** Pas de `package-lock.json` ni `npm-shrinkwrap.json`. Le `^1.0.0` permet des mises a jour automatiques vers n'importe quelle 1.x, incluant des versions non auditees.

### 1.3 npm — installer (`multiai-go/packaging/npm/`)

**Aucune dependance npm runtime.** Le `postinstall` telecharge un binaire Go natif. Pas de lockfile necessaire, mais l'absence de `npm-shrinkwrap.json` ouvre un risque theorique d'injection dans le registre npm si le package venait a etre compromis.

### 1.4 npm — legacy (`multiai-powershell/`)

**Aucune dependance npm.** Projet PowerShell uniquement. Pas de risque.

---

## 2. Workflows CI/CD

### 2.1 Fichiers audites

- `.github/workflows/ci.yml` (x2 : racine + multiai-go/.github/workflows/)
- `.github/workflows/release.yml` (x2 : racine + multiai-go/.github/workflows/)

> Note : GitHub n'execute les workflows que depuis la racine. Les fichiers dans `multiai-go/.github/workflows/` sont synchronises par `sync-workflows.ps1`. Les deux copies sont identiques, signe d'une bonne discipline de synchronisation.

### 2.2 Permissions

| Workflow | Permissions globales | Permissions job | Commentaire |
|---|---|---|---|
| CI | `contents: read` | — | **Minimales** |
| Release | `contents: read` | `contents: write`, `id-token: write`, `attestations: write` | **Justifie** (release + Cosign + attestations) |

### 2.3 Pinning des actions

Toutes les actions sont **pinnees par SHA complet** avec un commentaire indiquant la version lisible. Exemple :
```yaml
uses: actions/checkout@df4cb1c069e1874edd31b4311f1884172cec0e10 # v6.0.3
```

**Alerte :** `golangci-lint-action` utilise `version: latest` dans CI. Ce n'est pas un SHA — le binaire golangci-lint lui-meme n'est pas pinne et peut changer entre les runs. A remplacer par une version semver explicite (`version: "~> v2"` comme pour GoReleaser).

### 2.4 Secrets references

| Secret | Workflow | Risque |
|---|---|---|
| `GITHUB_TOKEN` | release.yml | Automatique, scope au workflow, **aucun risque** |
| `TAP_GITHUB_TOKEN` | Jamais utilise (commente dans `.goreleaser.yaml`) | **Aucun** — pas de secret configure |

### 2.5 Vecteurs d'injection

- Pas de `pull_request_target` — les PRs s'executent avec le contexte securise par defaut
- Pas de `workflow_run` — pas de workflows chaines
- Les `paths` filters limitent le declenchement aux fichiers modifies dans `multiai-go/`

### 2.6 Securite du pipeline npm

- `npm publish` est **manuel** (pas automatise dans CI) — bonne pratique
- `prepublishOnly` execute `scan-secrets.js` qui rejette :
  - Les versions `-dev`
  - Les fichiers .env avec de vraies cles API
  - Le sentinel `__MULTIAI_CREDSTORE__`

---

## 3. Scripts d'installation

### 3.1 `multiai-go/scripts/install.sh`

**Score :** 9/10

| Element | Present ? | Detail |
|---|---|---|
| HTTPS download | Oui | `curl -fsSL` vers `github.com/...` |
| SHA256 verification | Oui | `sha256sum --check` ou `shasum -a 256` |
| Cosign verification | **Non** | Aucune verification de signature cosign |
| Temp dir cleanup | Oui | `trap 'rm -rf "${TMPDIR}"' EXIT` |
| PATH check | Oui | Verifie que le dossier d'install est dans PATH |
| Set -euo pipefail | Oui | Bonnes pratiques bash |
| Version selection | Oui | `MULTIAI_VERSION` env var ou derniere release |

**Risque :** Le pipe `curl ... | bash` est recommande dans le script (`Usage: curl -fsSL https://rochetta.fr/multiai/install.sh | bash`). C'est un vecteur standard mais qui repose entierement sur la securite de la page web et du CDN. Ajouter une verification Cosign de `checksums.txt.sig` serait un renforcement supplementaire.

### 3.2 `multiai-go/packaging/npm/install.js`

**Score :** 9.5/10

| Element | Present ? | Detail |
|---|---|---|
| HTTPS download | Oui | `https.get` avec User-Agent |
| SHA256 verification | Oui | Verifie AVANT extraction |
| Cosign verification | **Non** | Aucune verification de signature cosign |
| Redirection handling | Oui | max 5 redirects suivis |
| Temp dir cleanup | Oui | `try/finally` avec `rmSync` |
| Dev version safe | Oui | Skip avec message si `-dev` |
| Escape hatch | Oui | `MULTIAI_SKIP_DOWNLOAD=1` |
| Platform detection | Oui | Mapping explicite os/arch |
| Error handling | Oui | `main().catch()` avec message utilsateur |
| Max redirects | Oui | 5, evite les boucles infinies |
| Checksums first | Oui | Telecharge checksums.txt AVANT l'archive |

**Risque :** Pas de verification de la signature Cosign de `checksums.txt`. Si le CDN GitHub etait compromis, l'attaquant pourrait fournir un `checksums.txt` falsifie. Cependant, le risque est limite car GitHub Releases est un CDN signe et les releases sont immutables.

### 3.3 `multiai-go/packaging/npm/scan-secrets.js`

**Score :** 9/10

| Protection | Present ? |
|---|---|
| Rejet version -dev | Oui |
| Scan des templates .env embarques | Oui (`internal/assets/profiles/`) |
| Scan des configs locales | Oui (`configs/profiles/`) |
| Scan du package npm | Oui (`__dirname`) |
| Regex secrets (API_KEY/_TOKEN/_SECRET/) | Oui |
| Pattern de valeurs vivantes | Oui (sk-ant-, sk-proj-, ghp_, AKIA, etc.) |
| Detection placeholders (PASTE_, YOUR_) | Oui |
| Detection sentinel credstore | Oui (`__MULTIAI_CREDSTORE__`) |
| Whitelist de cles de configuration | Oui (liste `meta` de ~70 cles autorisees) |

**Alerte mineure :** La regex `LIVE_VALUE_RE` couvre les formats connus mais n'inclut pas `zai_` (Z.ai), `minimax_`, `stepfun_`, `dashscope_`, `siliconflow_`, `mimo_`, `requesty_`, `litellm_` comme formats de cles detectables. Cependant, ces cles sont capturees par la regle `SECRET_KEY_RE` + longueur >= 20.

---

## 4. Fichiers sensibles exposes

### 4.1 `.gitignore` (racine)

**Couverture :** Bonne

```
*.env
configs/profiles/*.env
.credentials/
dist/
log/
*.out
multiai.exe
```

**Exception justifiee :** `!multiai-go/internal/assets/profiles/*.env` — les templates de profils embarques sont volontairement tracks car ce sont des placeholders.

### 4.2 `.gitignore` (multiai-powershell)

```
configs/profiles/*.env
*.env
.credentials/
docs/
.agents/
```

Les fichiers `.env` dans `configs/profiles/` du projet PowerShell sont **tracks dans git** malgre le `.gitignore`. Ce sont des templates avec des placeholders (`PASTE_*`). Aucun risque mais incohérence : le gitignore dit de les ignorer mais ils sont deja suivis.

### 4.3 Verification des fichiers commités

| Type de fichier | Trouvé dans git ? | Risque |
|---|---|---|
| `multiai.exe` (binaires) | **Non** | Aucun |
| `*.log` | **Non** | Aucun |
| `coverage.out` | **Non** | Aucun |
| `benchmark.txt` | **Non** | Aucun |
| `.env` avec secrets reels | **Non** | Tous les .env suivis ont des placeholders |
| Fichiers de credentials | **Non** | `.credentials/` est ignore |
| Build artifacts | **Non** | `dist/`, `build/`, `vendor/` ignores |
| Logs sessions | **Non** | `log/` ignore ; contenu actuel = documents d'audit uniquement |

### 4.4 Fichiers de test avec valeurs `sk-ant-*`

13 fichiers de test Go utilisent des valeurs factices (`sk-ant-test123`, `sk-ant-api03-...`). Ces valeurs **ne sont pas** de vraies cles API :
- Le prefixe `test123` et le motif `api03-` (inexistant dans le schema de cle Anthropic) sont clairement factices
- Aucune de ces cles n'est fonctionnelle

### 4.5 Verification git historique

Les seuls commits touchant des fichiers `.env` sont :
1. `c476d64` — feat: v0.3.0 (ajout des templates de profils)
2. `72c862c` — fix: track OpenRouter fusion profile templates
3. `f8e8923` — feat: audit v0.4.0

Aucun commit contenant de vraies cles API n'a ete trouve dans l'historique.

---

## 5. GoReleaser — `.goreleaser.yaml`

### 5.1 Securite des builds

| Element | Present ? | Detail |
|---|---|---|
| SHA256 checksums | Oui | Fichier `checksums.txt` genere automatiquement |
| Cosign keyless signing | Oui | `cosign sign-blob` avec OIDC GitHub Actions |
| Build provenance | Oui | `actions/attest-build-provenance` (public uniquement) |
| CGO disabled | Oui | `CGO_ENABLED=0` — pas de dependance C dynamique |
| Trimpath | Oui | `-trimpath` — pas de chemins de build dans le binaire |
| Strip symbols | Oui | `-s -w` dans ldflags |
| Mod timestamp | Oui | `{{ .CommitTimestamp }}` — reproducibilite |

### 5.2 Empaquetage

| Canal | skip_upload | Token expose ? |
|---|---|---|
| Homebrew Cask | `true` (manuel) | Aucun |
| Scoop | `true` (manuel) | Aucun |
| Debian (nfpm) | Publie dans la release | Aucun |

`TAP_GITHUB_TOKEN` est mentionne dans les commentaires comme preparation pour une future automatisation, mais n'est jamais utilise aujourd'hui.

---

## 6. Dependabot

Fichier : `multiai-go/.github/dependabot.yml`

| Ecosystème | Directory | Frequence |
|---|---|---|
| gomod | `/multiai-go` | weekly |
| github-actions | `/` | weekly |
| npm | `/multiai-powershell` | weekly |

**Note :** Dependabot ne scanne que le fichier a la racine du depot (monorepo constraint). La synchronisation est manuelle. Le npm ecosystem scanne `multiai-powershell` mais pas `multiai-go/docs/` (vitepress) ni `multiai-go/packaging/npm/` (aucune dependance, mais non couvert).

---

## 7. Scores et recommandations

### Scores par domaine

| Domaine | Score | Raison |
|---|---|---|
| Dependances Go | 9/10 | v3.0.1 patche mais archive |
| Dependances npm | 7/10 | Pas de lockfile, plage large |
| CI/CD workflows | 9/10 | Pinning SHA, permissions minimes, 1 alerte |
| Scripts installation | 8/10 | SHA256 ok, Cosign manquant |
| Fichiers sensibles | 9.5/10 | Rien de compromis, gitignore solide |
| GoReleaser | 10/10 | Checksums + Cosign + provenance |
| **Global** | **8.8/10** | |

### Recommandations

1. **Haute :** Migrer de `gopkg.in/yaml.v3` vers `github.com/go-yaml/yaml` (fork maintenu)
2. **Haute :** Ajouter `package-lock.json` ou `npm-shrinkwrap.json` pour `multiai-go/docs/` (vitepress)
3. **Moyenne :** Pinner `golangci-lint` avec `version: "~> v2"` au lieu de `latest`
4. **Moyenne :** Ajouter un npm ecosystem Dependabot pour `multiai-go/docs/`
5. **Basse :** Ajouter une verification Cosign dans `install.sh` et `install.js` (verifier la signature de checksums.txt)
6. **Basse :** Ajouter les formats de cles manquants a `LIVE_VALUE_RE` dans `scan-secrets.js` (zai_, minimax_, stepfun_, etc.)
7. **Basse :** Uniformiser `.gitignore` : retirer l'exception `configs/profiles/*.env` dans `multiai-powershell/.gitignore` si les templates sont volontairement suivis
