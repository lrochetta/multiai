---
layout: home
title: multiai
titleTemplate: Routeur multi-IA

hero:
  name: multiai
  text: Routeur multi-IA
  tagline: Un seul outil pour lancer Claude Code, Codex CLI et OpenCode — avec des profils isolés par fournisseur.
  image:
    src: /logo.svg
    alt: multiai
  actions:
    - theme: brand
      text: Démarrer
      link: /guide/getting-started
    - theme: alt
      text: Installation
      link: /guide/installation

features:
  - icon: 🚀
    title: Lancement unifié
    details: Un menu interactif pour choisir ton outil et ton profil. Ou lance directement avec multiai launch -p ds.
  - icon: 🔐
    title: Isolation des environnements
    details: Chaque profil a ses propres variables d'environnement. Pas de fuite de clés entre fournisseurs.
  - icon: 🖥️
    title: Cross-platform
    details: Windows, macOS, Linux. Binaire unique sans dépendance. Installe en une commande.
  - icon: ⚡
    title: 17 profils inclus
    details: Anthropic, Z.ai, DeepSeek, OpenAI, OpenRouter — déjà configurés pour toi.
  - icon: 🔧
    title: Extensible
    details: Crée tes propres profils en .env ou .yaml. Surcharge par projet avec .multiai.yaml.
  - icon: 🧩
    title: Plugin hooks
    details: Scripts before/after launch. VPN check, notifications, logging personnalisé.
---

## Quick Start

```bash
# macOS / Linux
curl -fsSL https://rochetta.fr/multiai/install.sh | bash

# Windows (PowerShell)
irm https://rochetta.fr/multiai/install.ps1 | iex

# Ou via Go
go install github.com/lrochetta/multiai@latest
```

```bash
# Configurer ses clés API
multiai config

# Lancer DeepSeek V4 Pro via Claude Code
multiai launch -p ds

# Lister tous les profils
multiai list --json
```
