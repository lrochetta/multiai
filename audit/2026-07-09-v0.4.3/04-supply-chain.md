# Audit Supply Chain & Distribution — multiai v0.4.3

**Date :** 2026-07-09
**Score global : 7.3/10**

---

## Résumé par catégorie

| Catégorie | Score | Poids |
|---|---|---|
| Dépendances Go | 9.0/10 | 15% |
| Packaging | 7.0/10 | 20% |
| CI/CD | 9.5/10 | 20% |
| Signatures | 8.5/10 | 15% |
| Build reproductible | 9.0/10 | 10% |
| Publication | 5.0/10 | 10% |
| SBOM | 1.0/10 | 5% |
| Dependabot | 8.0/10 | 5% |

---

## 1. Dépendances Go (9/10)

- **1 dépendance directe** : `gopkg.in/yaml.v3 v3.0.1`
- `go.sum` complet et vérifiable
- Go 1.22 (fin de support à prévoir → migrer vers 1.24)
- CI : `gosec`, `govulncheck`, `golangci-lint` (13 linters)

⚠️ `gosec` et `govulncheck` utilisent `@latest` non pinné dans la CI

---

## 2. Packaging (7/10)

### npm (Go binaire) — Bon
- `postinstall` : téléchargement + vérification SHA256
- `prepublishOnly` : `scan-secrets.js` bloque si clé réelle détectée
- Pas de vérification Cosign dans `install.js`

### npm (PowerShell legacy) — Fragile
- `prepublishOnly` inline dans package.json (~60 variables)
- Pas de `postinstall` automatique

### Homebrew / Scoop / AUR — Non publiés
- `skip_upload: true` dans `.goreleaser.yaml`
- Repos `lrochetta/homebrew-tap` et `lrochetta/scoop-bucket` inexistants
- PKGBUILD avec `sha256sums=('SKIP')`

---

## 3. CI/CD (9.5/10)

- Toutes les actions GitHub pinnées par SHA complet
- Permissions `contents: read` par défaut
- `concurrency` avec `cancel-in-progress: true`
- 7 jobs : lint, test (3 OS), security, benchmark, build, smoke, release-check

⚠️ `golangci-lint-action` utilise `version: latest`

---

## 4. Signatures (8.5/10)

- Cosign keyless (OIDC via Fulcio) sur `checksums.txt`
- Attestation GitHub (SLSA Build L1)
- Commande de vérification documentée dans le footer de release

⚠️ Identité regex Cosign trop large (`lrochetta/multiai` sans restriction de workflow/tag)

---

## 5. Build Reproductible (9/10)

Tous les flags standards : `CGO_ENABLED=0`, `-trimpath`, `-s -w`, `-X main.version=`, `mod_timestamp`

Manque : pas de vérification de reproductibilité dans le CI

---

## 6. Publication (5/10)

| Canal | Statut | Automatisé |
|---|---|---|
| GitHub Releases | Actif | Oui |
| npm (Go) | Manuel | Non |
| npm (PowerShell) | Manuel | Non |
| Homebrew | Non publié | Non |
| Scoop | Non publié | Non |
| Debian (.deb) | Inclus dans release | Oui |
| AUR | Non publié | Semi |

---

## 7. SBOM (1/10)

Aucun SBOM généré. Recommandation : `anchore/sbom-action` (Syft) en CycloneDX JSON.

---

## 8. Dependabot (8/10)

- 3 ecosystems : gomod, github-actions, npm
- Fréquence hebdomadaire
- Pas de `groups:` pour réduire le bruit

---

## Recommandations prioritaires

### 🔴 Immédiat
1. Créer `lrochetta/homebrew-tap` et `lrochetta/scoop-bucket`
2. Automatiser `npm publish` depuis le CI (avec `NPM_TOKEN`)
3. Pinner `gosec`/`govulncheck` à des versions spécifiques
4. Générer un SBOM dans le workflow de release

### 🟠 Haute priorité
5. Extraire `scan-secrets` du package.json PowerShell vers un fichier .js dédié
6. Vérifier Cosign dans `install.js` en plus du SHA256
7. Resserrer l'identité regex Cosign
8. Ne jamais pousser PKGBUILD avec `SKIP` sur AUR

### 🟡 Moyenne
9. Ajouter `CODEOWNERS` pour `.github/workflows/` et `.goreleaser.yaml`
10. Supprimer `packaging/deb/build-deb.sh` une fois GoReleaser stable
11. Signer le `.deb` avec GPG ou Cosign
