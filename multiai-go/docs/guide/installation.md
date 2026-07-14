# Installation

Cette page détaille toutes les méthodes d'installation de multiai.

## npm / npx (recommandé)

Prérequis : Node.js 24.14 ou ultérieur.

```bash
npx --yes --allow-scripts=multiai multiai@latest install
# Sous Windows, ouvre ensuite un nouveau terminal
multiai version
```

Cette commande installe globalement le binaire Go natif et vérifie son SHA256.
Sous Windows, elle détecte le préfixe npm, vérifie le shim `multiai.cmd` et
ajoute le dossier au `PATH` utilisateur de façon idempotente, sans droits
administrateur. Le processus d'installation ne peut pas modifier le terminal
déjà ouvert : ferme-le puis ouvre-en un nouveau. Pour un poste géré, définis
`MULTIAI_SKIP_PATH_UPDATE=1` afin de conserver un `PATH` administré
manuellement.

## go install (alternative)

Si tu as Go installé (version 1.22 ou ultérieure) :

```bash
go install github.com/lrochetta/multiai@latest
```

Le binaire sera placé dans `$GOPATH/bin` (par défaut `~/go/bin`). Assure-toi que ce dossier est dans ton `$PATH`.

Vérifie l'installation :

```bash
multiai version
```

## Installation universelle vérifiée

Utilise la version npm stable explicitement épinglée. Cette commande conserve
la vérification SHA-256 du binaire et n'exécute aucun script téléchargé depuis
un domaine tiers :

```bash
npx --yes --allow-scripts=multiai multiai@0.6.6 install
```

## Homebrew (macOS)

Si tu utilises Homebrew :

```bash
brew tap lrochetta/tap
brew install multiai
```

Mise à jour :

```bash
brew upgrade multiai
```

## APT (Ubuntu/Debian)

Si tu utilises une distribution basée sur Debian (Ubuntu, Debian, Linux Mint, Pop!_OS) :

```bash
# 1. Importer la clé GPG
sudo mkdir -p /usr/share/keyrings
sudo curl -fsSL https://lrochetta.github.io/multiai/apt/gpg-public.key \
  -o /usr/share/keyrings/multiai-archive-keyring.gpg

# 2. Ajouter le dépôt
echo "deb [signed-by=/usr/share/keyrings/multiai-archive-keyring.gpg] https://lrochetta.github.io/multiai/apt stable main" \
  | sudo tee /etc/apt/sources.list.d/multiai.list

# 3. Installer
sudo apt update
sudo apt install multiai
```

Mise à jour :

```bash
sudo apt update
sudo apt upgrade multiai
```

Vérification de la signature (optionnel) :

```bash
gpg --verify /usr/share/keyrings/multiai-archive-keyring.gpg
```

L'empreinte de la clé est `C386 EBA7 DD7F 742B 36C9 4BEE A6D9 99AB 129B 8351`.

## Scoop (Windows)

Si tu utilises Scoop :

```bash
scoop bucket add multiai https://github.com/lrochetta/scoop-multiai
scoop install multiai
```

Mise à jour :

```bash
scoop update multiai
```

## Exécution ponctuelle avec npm

Avec Node.js 24.14 ou ultérieur :

```bash
npx --yes --allow-scripts=multiai multiai@latest
```

Cette commande lance multiai sans installation globale.

## Compilation manuelle

Tu peux compiler depuis les sources :

```bash
git clone https://github.com/lrochetta/multiai.git
cd multiai/multiai-go
go build -o multiai ./cmd/multiai/
```

Pour compiler pour une plateforme spécifique :

```bash
# Linux amd64
GOOS=linux GOARCH=amd64 go build -o multiai-linux-amd64 ./cmd/multiai/

# macOS arm64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o multiai-darwin-arm64 ./cmd/multiai/

# Windows amd64
GOOS=windows GOARCH=amd64 go build -o multiai.exe ./cmd/multiai/
```

## Vérification

Quelle que soit la méthode, vérifie que tout fonctionne :

```bash
multiai version
multiai list
```

## Mise à jour

```bash
# go install
go install github.com/lrochetta/multiai@latest

# npm stable épinglé
npx --yes --allow-scripts=multiai multiai@0.6.6 install

# Homebrew
brew upgrade multiai

# APT
sudo apt update && sudo apt upgrade multiai

# Scoop
scoop update multiai
```

## Désinstallation

Supprime simplement le binaire `multiai` de ton PATH, ainsi que les dossiers de configuration :

```bash
# Configuration utilisateur
rm -rf ~/.multiai

# Sur Linux/macOS, supprime le binaire
rm $(which multiai)
```
