# Commandes

Reference complete de toutes les commandes multiai.

## multiai

Commande principale. Sans argument, lance le menu interactif.

```bash
multiai [commande] [flags]
```

### Sous-commandes

| Commande | Description |
|----------|-------------|
| `launch` | Lance un CLI avec un profil specifique |
| `list` | Liste les profils disponibles |
| `config` | Configure les cles API |
| `completion` | Genere les scripts de completion pour le shell |
| `version` | Affiche la version |
| `help` | Affiche l'aide |

### Flags globaux

| Flag | Description |
|------|-------------|
| `--help` | Affiche l'aide |
| `--version` | Affiche la version |
| `--json` | Sortie au format JSON (pour list) |

---

## multiai launch

Lance un CLI avec un profil specifique. Si aucun profil n'est specifie, utilise le profil par defaut ou affiche un menu.

```bash
multiai launch [flags]
```

### Flags

| Flag | Court | Description |
|------|-------|-------------|
| `--profile` | `-p` | Nom du profil a utiliser (ex: `ds`, `co`) |
| `--tool` | `-t` | CLI a utiliser (`claude`, `codex`, `opencode`) |
| `--dry-run` | `-n` | Affiche les variables d'environnement sans lancer le CLI |
| `--verbose` | `-v` | Mode verbeux : affiche le chemin et les variables |
| `--help` | `-h` | Affiche l'aide |

### Exemples

```bash
# Lancer avec le profil DeepSeek
multiai launch -p ds

# Lancer Claude Code avec le profil Anthropic
multiai launch -p co

# Simuler un lancement (affiche les variables, ne lance rien)
multiai launch -p ds --dry-run

# Mode verbeux
multiai launch -p codex55 -v

# Specifier le CLI directement
multiai launch -t codex -p oa5

# Avec variables d'environnement inline
ANTHROPIC_API_KEY=sk-ant-... multiai launch -p co
```

### Codes de sortie

| Code | Signification |
|------|---------------|
| `0` | Succes |
| `1` | Erreur generique |
| `2` | Profil introuvable |
| `3` | CLI introuvable |
| `4` | Cle API manquante |

---

## multiai list

Liste tous les profils disponibles.

```bash
multiai list [flags]
```

### Flags

| Flag | Description |
|------|-------------|
| `--json` | Sortie au format JSON |
| `--help` | Affiche l'aide |

### Exemples

```bash
# Liste simple
multiai list

# Sortie JSON
multiai list --json
```

### Exemple de sortie JSON

```json
[
  {
    "name": "co",
    "display_name": "Anthropic Claude",
    "tool": "claude",
    "provider": "Anthropic",
    "model": "claude-sonnet-4-20250514",
    "description": "Claude Code officiel — modele par defaut"
  },
  {
    "name": "ds",
    "display_name": "DeepSeek",
    "tool": "claude",
    "provider": "DeepSeek (via Anthropic)",
    "model": "claude-sonnet-4-20250514",
    "description": "DeepSeek V4 Pro chez Anthropic"
  }
]
```

---

## multiai config

Configure les cles API via un menu interactif.

```bash
multiai config [flags]
```

### Flags

| Flag | Description |
|------|-------------|
| `--store` | Methode de stockage (`file`, `keychain`, `wincred`, `secret-service`, `auto`) |
| `--help` | Affiche l'aide |

### Exemples

```bash
# Lancer le menu interactif
multiai config

# Utiliser le credential store du systeme
multiai config --store keychain

# Forcer le stockage fichier
multiai config --store file
```

---

## multiai completion

Genere les scripts de completion pour le shell.

```bash
multiai completion [shell]
```

### Shells supportes

- `bash`
- `zsh`
- `fish`
- `powershell`

### Exemples

```bash
# Bash
source <(multiai completion bash)

# Zsh
source <(multiai completion zsh)

# Fish
multiai completion fish | source

# Powershell
multiai completion powershell | Out-String | Invoke-Expression
```

### Installation permanente

```bash
# Bash
echo 'source <(multiai completion bash)' >> ~/.bashrc

# Zsh
echo 'source <(multiai completion zsh)' >> ~/.zshrc

# Fish
mkdir -p ~/.config/fish/completions
multiai completion fish > ~/.config/fish/completions/multiai.fish
```

---

## multiai version

Affiche la version installee de multiai.

```bash
multiai version
```

### Exemple de sortie

```
multiai version 1.0.0 (commit abc1234, 2026-06-23)
```

---

## multiai help

Affiche l'aide generale ou pour une sous-commande.

```bash
multiai help [commande]
```

### Exemples

```bash
# Aide generale
multiai help

# Aide pour une commande specifique
multiai help launch
multiai help config
```
