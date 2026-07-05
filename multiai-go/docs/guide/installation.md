# Installation

Cette page détaille toutes les méthodes d'installation de multiai.

## go install (recommandé)

Si tu as Go installé (version 1.22 ou ultérieure) :

```bash
go install github.com/lrochetta/multiai@latest
```

Le binaire sera placé dans `$GOPATH/bin` (par défaut `~/go/bin`). Assure-toi que ce dossier est dans ton `$PATH`.

Vérifie l'installation :

```bash
multiai version
```

## Script d'installation universel

### macOS / Linux

```bash
curl -fsSL https://rochetta.fr/multiai/install.sh | bash
```

Ce script :
1. Détecte l'architecture de ta machine (amd64 / arm64)
2. Télécharge la dernière version depuis GitHub Releases
3. Place le binaire dans `/usr/local/bin`

### Windows (PowerShell)

```powershell
irm https://rochetta.fr/multiai/install.ps1 | iex
```

Le binaire est placé dans un dossier ajouté au `PATH` utilisateur.

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

## npm wrapper

Pour les environnements Node.js, un wrapper npm est disponible :

```bash
npm install -g multiai-cli
```

Le wrapper télécharge et exécute le binaire natif. Il suit les versions de multiai automatiquement.

## Compilation manuelle

Tu peux compiler depuis les sources :

```bash
git clone https://github.com/lrochetta/multiai.git
cd multiai
go build -o multiai .
```

Pour compiler pour une plateforme spécifique :

```bash
# Linux amd64
GOOS=linux GOARCH=amd64 go build -o multiai-linux-amd64 .

# macOS arm64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o multiai-darwin-arm64 .

# Windows amd64
GOOS=windows GOARCH=amd64 go build -o multiai.exe .
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

# Script
curl -fsSL https://rochetta.fr/multiai/install.sh | bash

# Homebrew
brew upgrade multiai

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
