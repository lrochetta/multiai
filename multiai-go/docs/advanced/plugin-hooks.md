# Plugin Hooks

Les hooks permettent d'executer des scripts avant (`before_launch`) et apres (`after_launch`) le lancement d'un CLI. Ils sont configurables dans les profils YAML ou la configuration globale.

## Quand utiliser les hooks ?

- **Verification VPN** : s'assurer que le VPN est actif avant de lancer un CLI
- **Rotation de cles** : recuperer une cle API depuis un vault
- **Notifications** : envoyer une notification (Slack, Discord) au debut/fin d'une session
- **Logging** : enregistrer les lancements dans un fichier de log
- **Validation d'environnement** : verifier que les prerequis sont presents
- **Menage** : nettoyer des fichiers temporaires apres la session
- **Mise a jour** : verifier et installer les mises a jour avant de lancer

## Configuration globale

Dans `~/.multiai/config.yaml` :

```yaml
# ~/.multiai/config.yaml
hooks:
  before_launch: "/home/user/.multiai/hooks/before_launch.sh"
  after_launch: "/home/user/.multiai/hooks/after_launch.sh"
```

## Configuration par profil

Dans `~/.multiai/profiles.yaml` :

```yaml
profiles:
  securise:
    tool: claude
    display_name: "Profil securise"
    env:
      ANTHROPIC_API_KEY: "${ANTHROPIC_API_KEY}"
    hooks:
      before_launch: "/home/user/.multiai/hooks/check-vpn.sh"
      after_launch: "/home/user/.multiai/hooks/notify.sh"
```

Dans `.multiai.yaml` (projet) :

```yaml
profiles:
  default:
    extends: co
    hooks:
      before_launch: "./scripts/check-env.sh"
      after_launch: "./scripts/cleanup.sh"
```

## Variables transmises aux hooks

Les hooks recoivent les variables d'environnement suivantes :

| Variable | Description |
|----------|-------------|
| `MULTIAI_PROFILE` | Nom du profil utilise |
| `MULTIAI_TOOL` | CLI lance (`claude`, `codex`, `opencode`) |
| `MULTIAI_EXIT_CODE` | Code de sortie du CLI (uniquement dans after_launch) |
| `MULTIAI_PROJECT_DIR` | Repertoire du projet (si .multiai.yaml trouve) |

## Exemples concrets

### Verification VPN (before_launch)

```bash
#!/bin/bash
# ~/.multiai/hooks/check-vpn.sh
# Bloque le lancement si le VPN n'est pas actif

if ! curl -s --max-time 5 https://api.ipify.org | grep -q "212.80"; then
  echo "Erreur : VPN non detecte. Active ton VPN avant de lancer multiai."
  exit 1  # Un code non-zero bloque le lancement
fi

echo "VPN detecte. Lancement autorise."
```

### Rotation de cle depuis un vault (before_launch)

```bash
#!/bin/bash
# ~/.multiai/hooks/vault-key.sh
# Recupere la cle API depuis HashiCorp Vault

VAULT_ADDR="https://vault.company.com"
VAULT_TOKEN="$(cat ~/.vault-token)"

API_KEY=$(curl -s --header "X-Vault-Token: $VAULT_TOKEN" \
  "$VAULT_ADDR/v1/secret/data/multiai" | jq -r '.data.data.anthropic_key')

if [ -n "$API_KEY" ] && [ "$API_KEY" != "null" ]; then
  echo "Cle recuperee depuis Vault."
  # La cle est passee via l'environnement a multiai
else
  echo "Erreur : impossible de recuperer la cle depuis Vault."
  exit 1
fi
```

### Notification Slack (after_launch)

```bash
#!/bin/bash
# ~/.multiai/hooks/notify-slack.sh
# Envoie une notification Slack en fin de session

WEBHOOK_URL="https://hooks.slack.com/services/xxx/yyy/zzz"
DURATION=$(( $(date +%s) - $MULTIAI_START_TIME ))

curl -s -X POST -H 'Content-type: application/json' \
  --data "{
    \"text\": \"Session multiai terminee :\nProfil: $MULTIAI_PROFILE\nOutil: $MULTIAI_TOOL\nDuree: ${DURATION}s\nCode: $MULTIAI_EXIT_CODE\"
  }" "$WEBHOOK_URL" > /dev/null
```

### Logging horodate (after_launch)

```bash
#!/bin/bash
# ~/.multiai/hooks/log-session.sh
# Enregistre chaque session dans un fichier de log

LOG_FILE="$HOME/.multiai/sessions.log"
TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')

echo "$TIMESTAMP | $MULTIAI_PROFILE | $MULTIAI_TOOL | code: $MULTIAI_EXIT_CODE" >> "$LOG_FILE"
```

### Validation d'environnement (before_launch)

```bash
#!/bin/bash
# ~/.multiai/hooks/check-deps.sh
# Verifie que les dependances sont installees

REQUIRED_CMDS=("node" "npm" "git" "curl")
for cmd in "${REQUIRED_CMDS[@]}"; do
  if ! command -v "$cmd" &> /dev/null; then
    echo "Erreur : $cmd n'est pas installe."
    exit 1
  fi
done

echo "Toutes les dependances sont presentes."
```

## Ordre d'execution

1. `before_launch` hooks (global, puis projet, puis profil)
2. Lancement du CLI (claude, codex, opencode)
3. `after_launch` hooks (profil, puis projet, puis global)

## Gestion des erreurs

- Si un hook `before_launch` retourne un code non-zero, le lancement est **bloque**
- Si un hook `after_launch` echoue, le code de sortie du CLI est conserve
- Les hooks sont executes dans l'ordre defini

```bash
# before_launch echoue → lancement annule
$ multiai launch -p securise
Erreur : VPN non detecte. Active ton VPN avant de lancer multiai.
# Code de sortie : 1 (le CLI n'est pas lance)
```

## Bonnes pratiques

1. **Chemins absolus** : utilise des chemins absolus dans la configuration pour eviter les surprises
2. **Droits d'execution** : n'oublie pas `chmod +x` sur tes scripts
3. **Exit codes** : `before_launch` doit retourner 0 pour autoriser le lancement
4. **Portable** : les hooks sont executes dans l'environnement du shell, utilise `#!/bin/bash` ou `#!/bin/sh`
5. **Silencieux** : les hooks peuvent etre silencieux (ne rien afficher) si le verifications sont ok
6. **Timeout** : evite les operations longues dans les hooks qui retardent le lancement

## Voir aussi

- [Profils YAML](/advanced/yaml-profiles) — configuration des hooks par profil
- [Profils par projet](/advanced/project-profiles) — hooks dans .multiai.yaml
- [Configuration](/guide/configuration) — configuration globale
