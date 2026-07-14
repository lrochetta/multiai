# Premiers pas

Bienvenue dans multiai ! Ce guide t'aide à lancer ton premier CLI en moins de 2 minutes.

## Prérequis

- Un des CLI supportés : [Claude Code](https://docs.anthropic.com/en/docs/claude-code), [Codex CLI](https://github.com/openai/codex), [OpenCode](https://github.com/opencode-ai/opencode)
- Des clés API pour les fournisseurs que tu souhaites utiliser

## 1. Installation

### npm / npx (recommandé avec Node.js 24.14+)

```bash
npx --yes --allow-scripts=multiai multiai@latest install
```

Sous Windows, l'installeur ajoute automatiquement le préfixe global npm au
`PATH` utilisateur s'il manque. Ouvre un nouveau terminal avant de poursuivre.

### Autres méthodes

```bash
# npm stable épinglé (Windows, macOS et Linux)
npx --yes --allow-scripts=multiai multiai@0.6.6 install

# Go
go install github.com/lrochetta/multiai@latest
```

## 2. Configuration des clés

```bash
multiai config
```

Choisis un fournisseur, suis le lien pour créer une clé API, puis colle-la.

## 3. Premier lancement

```bash
# Menu interactif
multiai

# Ou lancement direct
multiai launch -p co        # Claude Code officiel
multiai launch -p ds        # DeepSeek V4 Pro via Claude Code
multiai launch -p codex55   # Codex GPT-5.5
multiai launch -p oc        # OpenCode
```

## 4. Et ensuite ?

- [Guide d'installation détaillé](/guide/installation) — toutes les méthodes
- [Configuration des profils](/guide/configuration) — comprendre les .env
- [Profils disponibles](/guide/profiles) — les 17 profils inclus
- [Dépannage](/guide/troubleshooting) — solutions aux erreurs courantes
