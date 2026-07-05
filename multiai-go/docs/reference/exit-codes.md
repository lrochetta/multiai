# Codes de sortie

multiai utilise 5 codes de sortie (0-4) pour indiquer le resultat de l'execution.

## Tableau des codes

| Code | Nom | Signification |
|------|-----|---------------|
| `0` | Success | Execution reussie |
| `1` | Error | Erreur generique |
| `2` | ProfileNotFound | Profil specifie introuvable |
| `3` | ToolNotFound | CLI (claude/codex/opencode) introuvable |
| `4` | MissingAPIKey | Cle API manquante pour le profil |

## Code 0 — Success

L'outil a ete lance avec succes ou la commande s'est terminee normalement.

**Exemples :**
```bash
multiai launch -p co
# Code de sortie : 0 (si Claude Code se lance correctement)

multiai list
# Code de sortie : 0

multiai version
# Code de sortie : 0
```

## Code 1 — Error

Une erreur generique est survenue. Consulte le message d'erreur pour plus de details.

**Causes possibles :**
- Erreur de lecture du fichier de configuration
- Erreur de parsing YAML
- Erreur interne
- Argument invalide

**Exemple :**
```bash
multiai launch --unknown-flag
# Error: unknown flag: --unknown-flag
# Code de sortie : 1
```

## Code 2 — ProfileNotFound

Le profil specifie avec l'option `--profile` / `-p` n'existe pas dans la configuration.

**Causes possibles :**
- Faute de frappe dans le nom du profil
- Profil non configure
- Fichier de profil manquant

**Exemple :**
```bash
multiai launch -p inexistant
# Error: profile "inexistant" not found
# Code de sortie : 2
```

**Solution :**
```bash
# Lister les profils disponibles
multiai list
```

## Code 3 — ToolNotFound

Le CLI requis par le profil (claude, codex, opencode) n'est pas installe ou introuvable dans le PATH.

**Causes possibles :**
- Claude Code non installe (`npm install -g @anthropic-ai/claude-code`)
- Codex CLI non installe
- OpenCode non installe
- Le binaire n'est pas dans le PATH

**Exemple :**
```bash
multiai launch -p co
# Error: executable "claude" not found in PATH
# Code de sortie : 3
```

**Solutions :**
```bash
# Verifier que le CLI est accessible
which claude
which codex
which opencode

# Installer le CLI manquant
npm install -g @anthropic-ai/claude-code
```

## Code 4 — MissingAPIKey

La cle API necessaire pour le profil n'est pas configuree.

**Causes possibles :**
- Fichier `.env` manquant pour le profil
- Variable d'environnement non definie
- Cle API vide ou invalide

**Exemple :**
```bash
multiai launch -p co
# Error: ANTHROPIC_API_KEY is not set for profile "co"
# Code de sortie : 4
```

**Solutions :**
```bash
# Configurer la cle avec le menu interactif
multiai config

# Ou definir la variable directement
export ANTHROPIC_API_KEY=sk-ant-...
# ou
ANTHROPIC_API_KEY=sk-ant-... multiai launch -p co
```

## Utiliser les codes de sortie dans des scripts

Les codes de sortie permettent d'automatiser des actions selon le resultat :

```bash
#!/bin/bash
multiai launch -p ds
case $? in
  0) echo "Lancement reussi" ;;
  2) echo "Profil introuvable, verification..." ; multiai list ;;
  3) echo "Installation du CLI necessaire" ; npm install -g @anthropic-ai/claude-code ;;
  4) echo "Configuration requise" ; multiai config ;;
  *) echo "Erreur inattendue" ; exit 1 ;;
esac
```

## Voir aussi

- [Commandes](/reference/commands) — documentation des commandes et flags
- [Depannage](/guide/troubleshooting) — solutions aux erreurs courantes
