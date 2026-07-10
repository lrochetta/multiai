J'ai maintenant une vision complete de l'etat actuel. Voici les 6 stories detaillees pour l'epic Distribution & Packaging (v0.6.0).

---

# Epic: Distribution & Packaging (v0.6.0)

## S6.1 — Depôt APT (Ubuntu/Debian)

**Priorite** : HIGH

**Objectif** : Mettre a disposition des utilisateurs Debian/Ubuntu un depôt APT officiel installe via `apt install multiai`, avec signature GPG et mise a jour automatique a chaque release.

**Description technique** :
GoReleaser produit deja les `.deb` via nfpm (`multiai-go/.goreleaser.yaml`, section `nfpms`) dans `dist/` a chaque tag `v*`. Il manque un depôt APT indexe (Release + Packages.gz + InRelease) heberge sur GitHub Pages (`lrochetta.github.io/multiai/apt`) ou un espace de stockage accessible. La solution recommandee est d'ajouter une section `repos` dans `.goreleaser.yaml`, qui genere l'arborescence `pool/` + `dists/stable/` avec les metadonnees APT. Le depôt sera pousse vers la branche `gh-pages` du repo, ou vers un repo `multiai-apt` dedie. Une cle GPG (`multiai@rochetta.fr`) sera generee une fois et ajoutee au trousseau de la CI (secret `APT_GPG_KEY`) pour signer le `Release`. La page d'installation dans la doc (`installation.md`) decrit `curl -fsSL https://lrochetta.github.io/multiai/apt/key.gpg | sudo gpg --dearmor -o /usr/share/keyrings/multiai.gpg` puis `echo "deb [signed-by=/usr/share/keyrings/multiai.gpg] https://lrochetta.github.io/multiai/apt stable main" | sudo tee /etc/apt/sources.list.d/multiai.list`.

**Fichiers impactes** :
- `multiai-go/.goreleaser.yaml` — ajouter section `repos` avec configuration APT
- `multiai-go/.github/workflows/release.yml` — ajouter permission `contents: write` sur gh-pages, ou workflow de publish dedie
- `multiai-go/packaging/apt/key.gpg` — cle publique GPG (ajoutee au repo, pas la privee)
- `multiai-go/docs/guide/installation.md` — section APT
- `multiai-go/README.md` — badge APT + instructions
- `.github/workflows/ci.yml` — eventuel job de verification du .deb

**Tests attendus** :
1. Sur une VM Ubuntu 22.04/24.04 vierge : `apt update && apt install multiai` -> binaire installe + fonctionnel
2. `multiai version` retourne la version de la release
3. Mise a jour : `apt update && apt upgrade multiai` apres une nouvelle release
4. Verification de la signature GPG : `apt-get update` ne produit pas d'avertissement `NO_PUBKEY`
5. Verification de l'integrite du .deb genere : `dpkg-deb --info multiai_*.deb`

**Resultat attendu** : Un utilisateur Debian/Ubuntu peut installer et tenir a jour multiai via le gestionnaire de paquets natif, avec verification de signature.

**Definition of Done** :
- [ ] Section `repos` activee dans `.goreleaser.yaml` generant l'arborescence APT
- [ ] Cle GPG generee, cle privee dans GitHub Secrets (`APT_GPG_KEY`), cle publique dans le repo
- [ ] Workflow CI/Release pousse le depôt vers `gh-pages` ou `multiai-apt`
- [ ] Integration testee sur Ubuntu 22.04 et 24.04 en CI
- [ ] Documentation d'installation mise a jour
- [ ] Release v0.6.0 produite avec paquet .bt installe via le depôt

**Risques** :
- GoReleaser `repos` necessite goreleaser Pro (payant) ou une solution custom via `ghp-import`/`aptly` -> verifier la licence
- GitHub Pages impose une limite de taille (1 GB) : le depôt APT (fichiers .deb + metadonnees) peut croitre avec les versions
- La cle GPG doit etre stockee de maniere securisee (secret scope = environnement `release`)

**Dependances** :
- Aucune (peut etre developpee independamment)
- Necessite l'existence d'un token `GITHUB_TOKEN` avec permissions sur gh-pages

---

## S6.2 — Paquet AUR (Arch Linux) avec verification SHA256

**Priorite** : HIGH

**Objectif** : Publier un `PKGBUILD` officiel sur l'AUR (Arch User Repository) dont la somme SHA256 du tarball source est verifiee automatiquement a chaque release, et mettre a jour le `.SRCINFO` correspondant.

**Description technique** :
Le `PKGBUILD` actuel (`multiai-go/packaging/aur/PKGBUILD`) a une version epinglee a 0.4.0 et un checksum `SKIP` qui desactive toute verification d'integrite. Il faut creer un pipeline qui, a chaque tag `v*`, calcule le SHA256 du tarball source GitHub (`multiai-<version>.tar.gz`) et met a jour a la fois le `PKGBUILD` et le `.SRCINFO` dans le repo. Le script existant `scripts/update-aur-checksums.sh` fait deja une partie du travail mais doit etre integre au workflow de release. Le PKGBUILD doit aussi corriger le chemin du module Go : le tarball source contient la racine du repo, donc le `go build` doit cibler `./multiai-go/cmd/multiai/`. Le binaire est compile depuis les sources (source build, pas precompile), ce qui est l'usage standard sur l'AUR. Le paquet cree `multiai` et `multiai-go` dans le repertoire source.

**Fichiers impactes** :
- `multiai-go/packaging/aur/PKGBUILD` — corriger version, makedepends, build(), package(), checksums
- `multiai-go/packaging/aur/.SRCINFO` — regenere automatiquement
- `multiai-go/scripts/update-aur-checksums.sh` — verifier/qu'il soit operationnel
- `multiai-go/.github/workflows/release.yml` — ajouter une etape de mise a jour AUR
- `multiai-go/docs/guide/installation.md` — instructions AUR
- `multiai-go/packaging/aur/README.md` — a creer, explications pour les maintainers

**Tests attendus** :
1. `makepkg -si` depuis le PKGBUILD frais -> compilation + installation reussies
2. `multiai version` affiche la version correcte
3. `makepkg --verifysource` echoue si le SHA256 ne correspond pas
4. Sur une instance Arch Linux vierge (Docker) : `yay -S multiai` ou `paru -S multiai`
5. Le `.SRCINFO` est valide (verifiable par `makepkg --printsrcinfo > .SRCINFO && diff`)

**Resultat attendu** : Tout utilisateur Arch Linux peut installer multiai via `yay -S multiai`, `paru -S multiai`, ou `git clone && makepkg -si`. L'integrite du code source est garantie par la verification SHA256.

**Definition of Done** :
- [ ] `PKGBUILD` mis a jour avec le chemin correct du sous-module `multiai-go/`
- [ ] Checksum SHA256 dynamique injecte par le workflow de release (plus de `SKIP`)
- [ ] Workflow CI pousse les modifications vers le repo AUR via `ssh` ou `aurpublish`
- [ ] `makepkg` reussi en environnement CI
- [ ] Documentation d'installation AUR mise a jour
- [ ] `CONTRIBUTING.md` mentionne la procedure de mise a jour AUR

**Risques** :
- L'AUR n'accepte que des mises a jour manuelles (pas d'API automatisee sans outil tiers comme `aurpublish` ou `aur-update`)
- Un `PKGBUILD` malforme est rejete par les maintainers AUR sans preavis
- La compilation depuis les sources prend du temps (1-3 min sur Arch) ; certains utilisateurs preferent un binaire precompile (AUR ne permet pas les binaires, mais il existe `multiai-bin`)

**Dependances** :
- S6.6 (install.sh) — le script de release doit verifier que le checksum est calculable facilement
- Necessite un compte AUR (`lrochetta`) et une cle SSH pour pousser

---

## S6.3 — Migration automatique depuis l'ancienne version PowerShell

**Priorite** : MEDIUM

**Objectif** : Detecter une installation existante de la version PowerShell legacy (`multiai-powershell/`) et migrer automatiquement les profils .env, les cles API et la configuration vers le format Go de v0.6.0, sans perte de donnees.

**Description technique** :
La version PowerShell legacy stocke ses profils dans `C:\AI\multiai\configs\profiles\*.env` (Windows) ou `~/.local/share/multiai/configs/profiles/*.env` (macOS/Linux), avec eventuellement des cles API non chiffrees dans les fichiers `.env`. La version Go utilise `~/.multiai/env/` ou le credential store AES-256-GCM. Il faut implementer une commande `multiai migrate --from-ps` (ou un wizard detectant automatiquement l'ancienne installation) qui :
1. Localise l'installation PowerShell (repertoire d'installation, PATH, ou variable d'environnement)
2. Parcourt les profils `.env` et extrait les paires `CLE=VALEUR`
3. Importe chaque cle dans le credential store chiffre de la version Go
4. Cree les profils correspondants dans le format Go (`~/.multiai/profiles/`)
5. Cree les wrappers `.cmd`/`.sh` equivalents si l'utilisateur les utilisait
6. Sauvegarde l'ancienne installation (copie vers `~/.multiai/backup-ps-<date>/`)
7. Affiche un rapport de migration (N profils migres, N cles importees, avertissements si certains champs n'ont pas d'equivalent Go)

L'architecture : un package `internal/migration/` avec un sous-package `powershell/` pour la logique de lecture du format legacy. La detection se fait par la presence de `code-router.ps1` dans le PATH ou de `~/.local/share/multiai/code-router.ps1`. Sur Windows, on cherche `C:\AI\multiai\code-router.ps1`.

**Fichiers impactes** :
- `multiai-go/cmd/multiai/main.go` — ajouter la sous-commande `migrate`
- `multiai-go/cmd/multiai/cmd_migrate.go` — nouvelle commande (a creer)
- `multiai-go/internal/migration/` — nouveau package (lecture PS, mapping vers Go)
- `multiai-go/internal/migration/powershell/` — detection + parsing
- `multiai-go/internal/secret/secret.go` — eventuellement une methode d'import en masse
- `multiai-go/internal/profile/profile.go` — eventuellement une methode d'import de profils
- `multiai-go/docs/guide/migration.md` — guide de migration
- `multiai-go/CHANGELOG.md` — mentionner la migration

**Tests attendus** :
1. Creer une installation PowerShell factice avec N profils et cles, lancer `multiai migrate` -> tous les profils et cles sont importes
2. `multiai migrate --dry-run` affiche ce qui serait importe sans rien changer
3. Sans installation PowerShell detectee : message clair "aucune installation legacy trouvee"
4. Sur une installation existante reelle (machine de dev) : migration complete sans perte
5. Apres migration, `multiai list` montre tous les profils et `multiai config list` montre toutes les cles
6. Test de rollback : si la migration echoue a mi-chemin, l'etat original est preserve

**Resultat attendu** : Un utilisateur de la version PowerShell peut basculer vers la version Go en une commande (`multiai migrate`), sans perdre sa configuration, ses cles API ni ses profils personnalises.

**Definition of Done** :
- [ ] Package `internal/migration/` implemente avec detection multi-plateforme
- [ ] Sous-commande `multiai migrate [--from-ps] [--dry-run]` fonctionnelle
- [ ] Import des profils .env vers le format Go + credential store
- [ ] Backup automatique de l'ancienne installation
- [ ] Tests unitaires (>=70% couverture du package migration)
- [ ] Guide de migration (`docs/guide/migration.md`)
- [ ] Test manuel valide sur installation PowerShell reelle

**Risques** :
- La version PowerShell legacy a pu etre installee via npm (`npx multiai`) ou via clone git ; les chemins varient
- Les fichiers .env legacy peuvent contenir des cles pour des fournisseurs que le Go ne supporte pas encore (ex: StepFun, Mimo, LiteLLM) -> warning mais pas d'erreur bloquante
- Le format des profils Go (YAML) est different du format .env PowerShell (cle=valeur) ; certains champs comme `ANTHROPIC_BASE_URL` custom ne sont pas directement portables
- Tests impossibles sans installation PowerShell de reference : creer un jeu de donnees de test synthetique

**Dependances** :
- Aucune (story independante)
- Idealement apres la stabilisation du credential store et du profile loader (v0.5.x)

---

## S6.4 — Homebrew tap (reactivation)

**Priorite** : HIGH

**Objectif** : Creer le repository `lrochetta/homebrew-tap`, generer et publier automatiquement la cask Homebrew a chaque release via GoReleaser, permettant `brew install --cask lrochetta/tap/multiai`.

**Description technique** :
GoReleaser a une section `homebrew_casks` commentee dans `.goreleaser.yaml` (lignes 124-128) avec `skip_upload: true`. Pour reactiver :
1. Creer le repo GitHub `lrochetta/homebrew-tap` (public, avec README minimal)
2. Generer un PAT (classic, scope `public_repo`) depuis un compte ayant push sur ce repo
3. Ajouter le PAT comme secret GitHub `TAP_GITHUB_TOKEN` (scope = environnement `release`)
4. Decommenter la section `homebrew_casks` dans `.goreleaser.yaml`, en remplacant `repository.token: "{{ .Env.TAP_GITHUB_TOKEN }}"` pour le push cross-repo
5. Configurer le `name`, `homepage`, `description`, `license` et le `post_install` (suppression de l'attribut de quarantaine macOS)
6. Le binaire etant precompile (pas de build depuis les sources), la cask declare `arch arm64` et `arch x86_64` avec le sha256 correspondant

La formule generee est un fichier `.rb` dans `dist/homebrew/Casks/multiai.rb` que GoReleaser pousse vers `lrochetta/homebrew-tap` via le PAT.

**Fichiers impactes** :
- `multiai-go/.goreleaser.yaml` — decommenter et configurer `homebrew_casks`
- `multiai-go/packaging/homebrew/README.md` — mettre a jour avec les instructions finales
- `multiai-go/.github/workflows/release.yml` — s'assurer que `TAP_GITHUB_TOKEN` est passe dans `env`
- `multiai-go/docs/guide/installation.md` — section Homebrew
- `multiai-go/README.md` — badge Homebrew + instructions

**Tests attendus** :
1. Sur macOS Intel : `brew tap lrochetta/tap && brew install --cask multiai` -> binaire installe + fonctionnel
2. Sur macOS Apple Silicon : meme test
3. `multiai version` apres installation retourne la version de la release
4. `brew upgrade multiai` apres une nouvelle release met a jour le binaire
5. Le sha256 du binaire telecharge correspond a celui de checksums.txt

**Resultat attendu** : Tout utilisateur macOS avec Homebrew peut installer et mettre a jour multiai via `brew install --cask lrochetta/tap/multiai`.

**Definition of Done** :
- [ ] `lrochetta/homebrew-tap` cree et public
- [ ] `TAP_GITHUB_TOKEN` configure dans les secrets GitHub (environnement `release`)
- [ ] Section `homebrew_casks` activee et testee dans `.goreleaser.yaml`
- [ ] Release v0.6.0 : cask generee dans dist/ et poussee vers le tap
- [ ] Installation testee sur macOS Intel + Apple Silicon
- [ ] Documentation mise a jour
- [ ] Le `post_install` gere correctement l'attribut de quarantaine : `sudo xattr -dr com.apple.quarantine /usr/local/bin/multiai`

**Risques** :
- macOS Gatekeeper peut bloquer le binaire non signe Apple (solution : notarization couteuse, ou guide utilisateur pour contourner)
- Le `TAP_GITHUB_TOKEN` doit etre un PAT classique (pas un token fine-grained) pour pouvoir pousser vers un autre repo ; GitHub deprecie les PAT classiques progressivement
- Homebrew Casks pour binaires non signes peuvent declencher des alerts de securite macOS

**Dependances** :
- Creation du repo `lrochetta/homebrew-tap`
- Generation du PAT et ajout aux secrets GitHub
- Configuration reseau : le workflow release doit avoir `id-token: write` (deja present)

---

## S6.5 — Scoop bucket (reactivation)

**Priorite** : MEDIUM

**Objectif** : Creer le repository `lrochetta/scoop-bucket` et publier automatiquement le manifest Scoop a chaque release via GoReleaser, permettant `scoop bucket add lrochetta https://github.com/lrochetta/scoop-bucket && scoop install multiai`.

**Description technique** :
Structure similaire a S6.4 (Homebrew) mais pour Windows/Scoop. La section `scoops` de `.goreleaser.yaml` est commentee (`skip_upload: true`). Le manifest Scoop est genere dans `dist/scoop/multiai.json` avec le sha256 de l'archive Windows. Pour reactiver :
1. Creer le repo GitHub `lrochetta/scoop-bucket` (public)
2. Le meme `TAP_GITHUB_TOKEN` (PAT classique, scope `public_repo`) est utilise pour pousser
3. Decommenter la section `scoops` dans `.goreleaser.yaml`
4. Verifier que l'URL de l'archive est correcte : l'archive Windows est `.zip`, nommee `multiai_<version>_windows_amd64.zip`
5. S'assurer que `windows/arm64` n'est PAS declare (goreleaser l'ignore deja, mais le manifest ne doit pas referencer une archive inexistante)
6. Verifier la presence de `checkver` et `autoupdate` dans le manifest pour les mises a jour automatiques

Le manifest genere par GoReleaser suit le format Scoop standard : `version`, `description`, `homepage`, `license`, `architecture.{64bit,arm64}.url+hash`, `bin` (nom du binaire), `checkver` (github releases), `autoupdate` (template URL + hash).

**Fichiers impactes** :
- `multiai-go/.goreleaser.yaml` — decommenter et configurer `scoops`
- `multiai-go/packaging/scoop/README.md` — mettre a jour avec les instructions finales
- `multiai-go/.github/workflows/release.yml` — `TAP_GITHUB_TOKEN` deja partage
- `multiai-go/docs/guide/installation.md` — section Scoop (corriger le nom du bucket)
- `multiai-go/README.md` — badge Scoop + instructions

**Tests attendus** :
1. Sur Windows 10/11 : `scoop bucket add lrochetta https://github.com/lrochetta/scoop-bucket && scoop install multiai` -> binaire installe + fonctionnel
2. `multiai version` apres installation retourne la version de la release
3. `scoop update multiai` apres une nouvelle release met a jour
4. Le hash sha256 du manifest correspond a checksums.txt
5. `scoop install multiai` ne produit pas d'erreur de hash mismatch

**Resultat attendu** : Tout utilisateur Windows avec Scoop peut installer et mettre a jour multiai via la commande standard Scoop.

**Definition of Done** :
- [ ] `lrochetta/scoop-bucket` cree et public
- [ ] Section `scoops` activee et testee dans `.goreleaser.yaml`
- [ ] Release v0.6.0 : manifest genere dans dist/ et pousse vers le bucket
- [ ] Installation testee sur Windows 10/11
- [ ] Documentation mise a jour (correction des incoherences de nom de bucket)
- [ ] Le manifest ne declare pas windows/arm64

**Risques** :
- Meme risque PAT que Homebrew (S6.4)
- Scoop requiert que le binaire soit un `.exe` signe ou au minimum non bloque par Windows Defender / SmartScreen -> peut generer un faux positif
- Le `autoupdate` de Scoop repose sur le format de l'URL GitHub release ; si le format change, les utilisateurs ne recoivent pas les mises a jour

**Dependances** :
- Creation du repo `lrochetta/scoop-bucket`
- Partage du `TAP_GITHUB_TOKEN` avec la config Homebrew

---

## S6.6 — Scripts d'installation cross-platform (install.sh / install.ps1)

**Priorite** : BLOCKER

**Objectif** : Fournir des scripts d'installation fonctionnels, verifies et heberges pour macOS/Linux (`install.sh`) et Windows (`install.ps1`) qui telechargent la derniere release du binaire Go, verifient son integrite SHA256, et l'installent avec les permissions correctes.

**Description technique** :
Le script `multiai-go/scripts/install.sh` existe mais a plusieurs defauts critiques identifies par l'audit :
1. Il reference `rochetta.fr/multiai/install.sh` qui retourne HTTP 404
2. Il eplngt la version 0.5.0 en dur (pas de resolution `latest`)
3. Il ne verifie pas le statut HTTP du telechargement
4. Sa section de resolution `latest` utilise `curl -fsSLI` qui peut echouer silencieusement
5. Il installe dans `~/.local/bin` mais la doc dit `/usr/local/bin`

Pour Windows, aucun `install.ps1` pour le binaire Go n'existe (celui dans `multiai-powershell/` installe la version PowerShell). Il faut creer un script PowerShell qui :
- Detecte l'architecture (amd64)
- Telecharge le zip depuis GitHub Releases
- Verifie le checksum SHA256
- Extrait et place `multiai.exe` dans un dossier (ex: `$env:LOCALAPPDATA\multiai` ou `$env:ProgramFiles\multiai`)
- Ajoute au PATH utilisateur
- Supporte `-InstallDir`, `-Version` (pin), `-SkipChecksum` (debug)

Les deux scripts doivent etre heberges accessiblement : soit sur GitHub Pages (`https://lrochetta.github.io/multiai/install.sh`), soit directement depuis le repo (`https://raw.githubusercontent.com/lrochetta/multiai/master/multiai-go/scripts/install.sh`). La deuxieme option est preferable pendant le developpement.

Les scripts doivent partager les conventions de nommage d'archive de GoReleaser (`multiai_<version>_<os>_<arch>.tar.gz` pour Linux/macOS, `.zip` pour Windows).

**Fichiers impactes** :
- `multiai-go/scripts/install.sh` — reecriture complete (version dynamique, verification SHA256, gestion d'erreur)
- `multiai-go/scripts/install.ps1` — nouveau fichier pour le binaire Go (a cote de celui du PowerShell legacy)
- `multiai-go/docs/guide/installation.md` — mise a jour des URLs et chemins d'installation
- `multiai-go/README.md` — badges + commandes curl/irm
- `multiai-go/packaging/npm/install.js` — aligner les conventions de nommage d'archive (verifier consistency)
- `.github/workflows/ci.yml` — eventuel smoke test des scripts

**Tests attendus** :
1. `curl -fsSL https://raw.githubusercontent.com/lrochetta/multiai/master/multiai-go/scripts/install.sh | bash` sur Ubuntu 22.04, Ubuntu 24.04, macOS Intel, macOS Apple Silicon -> binaire installe + fonctionnel
2. `irm https://raw.githubusercontent.com/lrochetta/multiai/master/multiai-go/scripts/install.ps1 | iex` sur Windows 10/11 -> binaire installe + PATH mis a jour
3. `MULTIAI_VERSION=0.5.0 curl ... | bash` installe cette version specifique
4. Simulation d'un checksum mismatch -> le script echoue avec un message clair, pas d'installation corrompue
5. `MULTIAI_SKIP_CHECKSUM=1` bypass (debug)
6. `MULTIAI_INSTALL_DIR=/opt/multiai` fonctionne
7 - Le script continue si `sha256sum` ou `shasum` n'est pas installe (fallback vers `openssl sha256`)
8. Sur Windows, verification que le PATH est bien mis a jour dans le registre (persistant) et immediatement

**Resultat attendu** : Les deux commandes one-liner (`curl | bash` et `irm | iex`) fonctionnent de bout en bout : telechargement, verification SHA256, installation, PATH, et `multiai version` fonctionnel immediatement.

**Definition of Done** :
- [ ] `install.sh` reecrit : resolution `latest`, verification SHA256, fallback hash tool, `MULTIAI_VERSION`, `MULTIAI_INSTALL_DIR`, `MULTIAI_SKIP_CHECKSUM`
- [ ] `install.ps1` cree : detection architecture, telechargement, verification SHA256, PATH registre, `-InstallDir`, `-Version`
- [ ] Les deux scripts sont tests sur leurs plateformes cibles
- [ ] Les URLs de la doc (`installation.md`, `README.md`) pointent vers des ressources accessibles (raw.githubusercontent.com)
- [ ] Les scripts sont integres au smoke test CI (execution en environnement isole)
- [ ] Les conventions de nommage d'archive sont alignees entre scripts et GoReleaser

**Risques** :
- `curl | bash` est un pattern de securite sensible ; le script doit etre le plus transparent possible (verification SHA256 obligatoire)
- GitHub raw.githubusercontent.com peut avoir du cache ou etre bloque dans certaines entreprises
- Windows `irm | iex` est bloque par la politique d'execution PowerShell par defaut (Restricted) -> la doc doit mentionner `Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass`
- Le script doit etre compatible avec `sh` et `bash` (pas de syntaxe bash-only)

**Dependances** :
- Aucune (priorite maximale, peut etre developpee en parallele)
- Sert de fondation aux stories S6.1, S6.2, S6.4, S6.5 (les scripts sont la methode d'installation universelle)

---

## Resume des priorites et dependances

```
S6.6 (BLOCKER) — Install scripts
  |
  ├── S6.1 (HIGH) — APT repo        (depend de scripts pour la doc)
  ├── S6.2 (HIGH) — AUR package     (depend de scripts pour le checksum pipeline)
  ├── S6.4 (HIGH) — Homebrew tap    (partage TAP_GITHUB_TOKEN avec S6.5)
  |
  ├── S6.5 (MEDIUM) — Scoop bucket  (partage TAP_GITHUB_TOKEN avec S6.4)
  |
  └── S6.3 (MEDIUM) — Migration PS  (independante)
```

Les 6 stories couvrent l'ensemble de l'epic "Distribution & Packaging" identifie dans la roadmap v0.6.0. Les stories S6.1, S6.2, S6.4, S6.5 et S6.6 sont les pre-requis pour que le binaire Go soit installable via tous les gestionnaires de paquets courants. S6.3 (migration PowerShell) assure la continuite pour les utilisateurs existants du legacy.