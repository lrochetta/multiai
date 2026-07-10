# Hooks before/after launch

Les hooks permettent d'exécuter des scripts automatiquement **avant** (`before_launch`) et **après** (`after_launch`) le lancement d'un CLI. Ils sont configurables dans les profils YAML, la configuration projet, ou la configuration globale.

## Cas d'usage

| Usage | Exemple |
|-------|---------|
| 🔒 **Vérification VPN** | Bloque le lancement si le VPN est inactif |
| 🔑 **Rotation de clés** | Récupère une clé API depuis un vault (HashiCorp, Azure Key Vault) |
| 🔔 **Notifications** | Envoie une notification Slack/Discord au début/fin de session |
| 📝 **Logging** | Enregistre chaque lancement dans un fichier de log horodaté |
| ✅ **Validation d'environnement** | Vérifie que les dépendances sont installées |
| 🧹 **Ménage** | Nettoie des fichiers temporaires après la session |
| 📦 **Mise à jour** | Vérifie et installe les mises à jour avant le lancement |

## Configuration globale

Dans `~/.multiai/config.yaml` :

```yaml
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
    display_name: "Profil sécurisé"
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

## Variables d'environnement transmises

Les hooks reçoivent ces variables pour contextualiser leur exécution :

| Variable | Description | Disponible dans |
|----------|-------------|-----------------|
| `MULTIAI_PROFILE` | Nom du profil utilisé | before + after |
| `MULTIAI_TOOL` | CLI lancé (`claude`, `codex`, `opencode`) | before + after |
| `MULTIAI_EXIT_CODE` | Code de sortie du CLI | after uniquement |
| `MULTIAI_PROJECT_DIR` | Répertoire du projet (si `.multiai.yaml` trouvé) | before + after |
| `MULTIAI_START_TIME` | Timestamp Unix du début du lancement | before + after |

## Exemples concrets

### Vérification VPN (before_launch)

```bash
#!/bin/bash
# ~/.multiai/hooks/check-vpn.sh
# Bloque le lancement si le VPN n'est pas actif

if ! curl -s --max-time 5 https://api.ipify.org | grep -q "212.80"; then
  echo "Erreur : VPN non détecté. Active ton VPN avant de lancer multiai."
  exit 1  # Un code non-zero bloque le lancement
fi

echo "VPN détecté. Lancement autorisé."
```

### Récupération de clé depuis Vault (before_launch)

```bash
#!/bin/bash
# ~/.multiai/hooks/vault-key.sh
# Récupère la clé API depuis HashiCorp Vault

VAULT_ADDR="https://vault.company.com"
VAULT_TOKEN="$(cat ~/.vault-token)"

API_KEY=$(curl -s --header "X-Vault-Token: $VAULT_TOKEN" \
  "$VAULT_ADDR/v1/secret/data/multiai" | jq -r '.data.data.anthropic_key')

if [ -n "$API_KEY" ] && [ "$API_KEY" != "null" ]; then
  export ANTHROPIC_API_KEY="$API_KEY"
  echo "Clé récupérée depuis Vault."
else
  echo "Erreur : impossible de récupérer la clé depuis Vault."
  exit 1
fi
```

### Notification Slack (after_launch)

```bash
#!/bin/bash
# ~/.multiai/hooks/notify-slack.sh
# Envoie une notification Slack en fin de session

WEBHOOK_URL="https://hooks.slack.com/services/xxx/yyy/zzz"
DURATION=$(( $(date +%s) - MULTIAI_START_TIME ))

curl -s -X POST -H 'Content-type: application/json' \
  --data "{
    \"text\": \"Session multiai terminée :\nProfil: $MULTIAI_PROFILE\nOutil: $MULTIAI_TOOL\nDurée: ${DURATION}s\nCode: $MULTIAI_EXIT_CODE\"
  }" "$WEBHOOK_URL" > /dev/null
```

### Logging horodaté (after_launch)

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
# Vérifie que les dépendances sont installées

REQUIRED_CMDS=("node" "npm" "git" "curl")
for cmd in "${REQUIRED_CMDS[@]}"; do
  if ! command -v "$cmd" &> /dev/null; then
    echo "Erreur : $cmd n'est pas installé."
    exit 1
  fi
done

echo "Toutes les dépendances sont présentes."
```

## Ordre d'exécution

1. `before_launch` global (depuis `config.yaml`)
2. `before_launch` projet (depuis `.multiai.yaml`)
3. `before_launch` profil (depuis `profiles.yaml`)
4. **Lancement du CLI** (claude, codex, opencode)
5. `after_launch` profil
6. `after_launch` projet
7. `after_launch` global

## Gestion des erreurs

- Si un hook `before_launch` retourne un code non-zero, le lancement est **bloqué**
- Si un hook `after_launch` échoue, le code de sortie du CLI est conservé
- Les hooks sont exécutés dans l'ordre défini ci-dessus

```bash
# before_launch échoue → lancement annulé
$ multiai launch -p securise
Erreur : VPN non détecté. Active ton VPN avant de lancer multiai.
# Code de sortie : 1 (le CLI n'est pas lancé)
```

## Bonnes pratiques

1. **Chemins absolus** — utilise des chemins absolus dans la configuration pour éviter les surprises
2. **Droits d'exécution** — n'oublie pas `chmod +x` sur tes scripts
3. **Exit codes** — `before_launch` doit retourner 0 pour autoriser le lancement
4. **Shebang** — utilise `#!/bin/bash` ou `#!/bin/sh` pour la portabilité
5. **Silencieux** — les hooks peuvent ne rien afficher si les vérifications sont OK
6. **Timeout** — évite les opérations longues qui retardent le lancement

## Voir aussi

- [Configuration projet (.multiai.yaml)](/advanced/project-config) — hooks dans la config projet
- [Profils YAML](/advanced/yaml-profiles) — configuration des hooks par profil
- [Configuration](/guide/configuration) — configuration globale
