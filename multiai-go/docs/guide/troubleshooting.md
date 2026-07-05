# Depannage

Cette page recense les erreurs les plus frequentes et leurs solutions.

## multiai : commande introuvable

**Symptome :** `bash: multiai: command not found`

**Causes :**
- Le binaire n'est pas installe
- Le dossier d'installation n'est pas dans le `PATH`

**Solutions :**
```bash
# Verifier l'installation
go install github.com/lrochetta/multiai@latest

# Ajouter GOBIN au PATH
export PATH=$PATH:$(go env GOBIN)
# Si GOBIN est vide
export PATH=$PATH:~/go/bin
```

## Clavier qui bloque dans Claude Code

**Symptome :** Le terminal n'accepte plus la saisie apres `multiai launch`

**Causes :**
- Conflit de mode terminal
- Session interactive mal initialisee

**Solutions :**
- Ouvrir un nouveau terminal et relancer
- Utiliser `reset` dans le terminal bloque
- Verifier que le CLI cible (claude, codex) est correctement installe

## Erreur : "no such profile"

**Symptome :**
```
Error: profile "xx" not found
```

**Causes :**
- Le profil n'existe pas dans la configuration
- Faute de frappe dans le nom du profil

**Solutions :**
```bash
# Lister les profils disponibles
multiai list

# Verifier les profils installes
ls ~/.multiai/profiles/
```

## Erreur API : 401 Unauthorized

**Symptome :**
```
Error: 401 Unauthorized - invalid API key
```

**Causes :**
- Cle API manquante ou invalide
- Fichier `.env` mal configure

**Solutions :**
```bash
# Reconfigurer la cle
multiai config

# Verifier le fichier de profil
cat ~/.multiai/profiles/co.env

# Verifier que la variable est exportee
echo $ANTHROPIC_API_KEY
```

## Erreur : "executable file not found"

**Symptome :**
```
Error: exec: "claude": executable file not found
```

**Causes :**
- Claude Code, Codex CLI ou OpenCode n'est pas installe
- Le CLI n'est pas dans le PATH

**Solutions :**
```bash
# Installer Claude Code
npm install -g @anthropic-ai/claude-code

# Installer Codex CLI (via npm ou Go)
npm install -g @openai/codex

# Verifier l'installation
which claude
which codex
which opencode
```

## Le menu interactif ne s'affiche pas

**Symptome :** Le programme se termine sans afficher le menu

**Causes :**
- Probleme de terminal (pipe, redirection)
- Variable `TERM` non definie

**Solutions :**
```bash
# Verifier le terminal
echo $TERM

# Forcer le mode terminal
export TERM=xterm-256color
multiai
```

## Erreur : "permission denied"

**Symptome :**
```
Error: permission denied
```

**Causes :**
- Le binaire n'a pas les droits d'execution
- Le dossier de configuration n'est pas accessible

**Solutions :**
```bash
# Donner les droits d'execution
chmod +x $(which multiai)

# Reinitialiser les permissions du dossier de config
chmod -R 700 ~/.multiai
```

## Erreur : "port already in use"

**Symptome :** Conflit de port lors du lancement de Claude Code

**Causes :**
- Une instance de Claude Code est deja en cours
- Le port est utilise par un autre processus

**Solutions :**
```bash
# Trouver et tuer le processus
lsof -i :<port>
kill -9 <PID>

# Ou changer de port
MCP_PORT=<autre_port> multiai launch -p co
```

## Les profils YAML ne sont pas charges

**Symptome :** Les profils definis dans `profiles.yaml` n'apparaissent pas

**Causes :**
- Fichier mal formatte (indentation YAML)
- Mauvais emplacement du fichier

**Solutions :**
```bash
# Verifier le format YAML
yamllint ~/.multiai/profiles.yaml

# Verifier l'emplacement
ls -la ~/.multiai/profiles.yaml

# Valider avec multiai
multiai list --json
```

## Les hooks ne s'executent pas

**Symptome :** Les scripts before_launch ou after_launch sont ignores

**Causes :**
- Le script n'a pas les droits d'execution
- Chemin absolu non specifie
- Extension de fichier incorrecte (`.sh` requis sur Linux/macOS)

**Solutions :**
```bash
# Donner les droits d'execution
chmod +x ~/.multiai/hooks/before_launch.sh

# Verifier le chemin dans la configuration
# Le chemin doit etre absolu
cat ~/.multiai/config.yaml
```

## Erreur Go : "cannot find module"

**Symptome :** `go install` echoue avec une erreur de module

**Causes :**
- Version de Go trop ancienne
- Probleme de proxy Go

**Solutions :**
```bash
# Verifier la version de Go
go version
# Doit etre >= 1.22

# Configurer le proxy Go
go env -w GOPROXY=https://proxy.golang.org,direct

# Nettoyer le cache et reessayer
go clean -modcache
go install github.com/lrochetta/multiai@latest
```

## Vous ne trouvez pas la solution ?

- Ouvrez une [issue](https://github.com/lrochetta/multiai/issues) sur GitHub
- Consultez la [Configuration](/guide/configuration) pour verifier vos fichiers

> **Note :** multiai utilise l'encodage UTF-8. Les caracteres accentues dans les commentaires et la documentation sont normaux.
