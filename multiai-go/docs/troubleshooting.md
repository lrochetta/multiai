# Dépannage

Cette page recense les erreurs les plus fréquentes et leurs solutions.

## multiai : commande introuvable

**Symptôme :** `bash: multiai: command not found`

**Causes :**
- Le binaire n'est pas installé
- Le dossier d'installation n'est pas dans le `PATH`

**Solutions :**
```bash
# Vérifier l'installation
go install github.com/lrochetta/multiai@latest

# Ajouter GOBIN au PATH
export PATH=$PATH:$(go env GOBIN)
# Si GOBIN est vide
export PATH=$PATH:~/go/bin
```

---

## Clavier qui bloque dans Claude Code

**Symptôme :** Le terminal n'accepte plus la saisie après `multiai launch`

**Causes :**
- Conflit de mode terminal
- Session interactive mal initialisée

**Solutions :**
- Ouvrir un nouveau terminal et relancer
- Utiliser `reset` dans le terminal bloqué
- Vérifier que le CLI cible (claude, codex) est correctement installé

---

## Erreur : "no such profile"

**Symptôme :**
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

# Vérifier les profils installés
ls ~/.multiai/profiles/
```

---

## Erreur API : 401 Unauthorized

**Symptôme :**
```
Error: 401 Unauthorized - invalid API key
```

**Causes :**
- Clé API manquante ou invalide
- Fichier `.env` mal configuré

**Solutions :**
```bash
# Reconfigurer la clé
multiai config

# Vérifier le fichier de profil
cat ~/.multiai/profiles/co.env

# Vérifier que la variable est exportée
echo $ANTHROPIC_API_KEY
```

---

## Erreur : "executable file not found"

**Symptôme :**
```
Error: exec: "claude": executable file not found
```

**Causes :**
- Claude Code, Codex CLI ou OpenCode n'est pas installé
- Le CLI n'est pas dans le PATH

**Solutions :**
```bash
# Installer Claude Code
npm install -g @anthropic-ai/claude-code

# Installer Codex CLI (via npm)
npm install -g @openai/codex

# Vérifier l'installation
which claude
which codex
which opencode
```

---

## Le menu interactif ne s'affiche pas

**Symptôme :** Le programme se termine sans afficher le menu

**Causes :**
- Problème de terminal (pipe, redirection)
- Variable `TERM` non définie

**Solutions :**
```bash
# Vérifier le terminal
echo $TERM

# Forcer le mode terminal
export TERM=xterm-256color
multiai
```

---

## Erreur : "permission denied"

**Symptôme :**
```
Error: permission denied
```

**Causes :**
- Le binaire n'a pas les droits d'exécution
- Le dossier de configuration n'est pas accessible

**Solutions :**
```bash
# Donner les droits d'exécution
chmod +x $(which multiai)

# Réinitialiser les permissions du dossier de config
chmod -R 700 ~/.multiai
```

---

## Erreur : "port already in use"

**Symptôme :** Conflit de port lors du lancement de Claude Code

**Causes :**
- Une instance de Claude Code est déjà en cours
- Le port est utilisé par un autre processus

**Solutions :**
```bash
# Trouver et tuer le processus
lsof -i :<port>
kill -9 <PID>

# Ou changer de port
MCP_PORT=<autre_port> multiai launch -p co
```

---

## Les profils YAML ne sont pas chargés

**Symptôme :** Les profils définis dans `profiles.yaml` n'apparaissent pas

**Causes :**
- Fichier mal formaté (indentation YAML)
- Mauvais emplacement du fichier

**Solutions :**
```bash
# Vérifier le format YAML
yamllint ~/.multiai/profiles.yaml

# Vérifier l'emplacement
ls -la ~/.multiai/profiles.yaml

# Valider avec multiai
multiai list --json
```

---

## Les hooks ne s'exécutent pas

**Symptôme :** Les scripts before_launch ou after_launch sont ignorés

**Causes :**
- Le script n'a pas les droits d'exécution
- Chemin absolu non spécifié
- Extension incorrecte (`.sh` requis sur Linux/macOS)

**Solutions :**
```bash
# Donner les droits d'exécution
chmod +x ~/.multiai/hooks/before_launch.sh

# Vérifier le chemin dans la configuration
# Le chemin doit être absolu
cat ~/.multiai/config.yaml
```

---

## Erreur Go : "cannot find module"

**Symptôme :** `go install` échoue avec une erreur de module

**Causes :**
- Version de Go trop ancienne
- Problème de proxy Go

**Solutions :**
```bash
# Vérifier la version de Go
go version
# Doit être >= 1.22

# Configurer le proxy Go
go env -w GOPROXY=https://proxy.golang.org,direct

# Nettoyer le cache et réessayer
go clean -modcache
go install github.com/lrochetta/multiai@latest
```

---

## L'auto-update ne fonctionne pas

**Symptôme :** multiai ne vérifie pas les mises à jour au démarrage

**Causes :**
- Pas de connexion internet
- Cache de version encore valide (1h)
- Binaire installé dans un dossier sans droit d'écriture

**Solutions :**
```bash
# Forcer la vérification
multiai version

# Vider le cache
rm -rf ~/.cache/multiai

# Vérifier les permissions du dossier d'installation
ls -la $(which multiai)
```

---

## Erreur de credential store

**Symptôme :** "credential store unavailable" ou "key not found"

**Causes :**
- Le service système (keychain, wincred, secret-service) n'est pas disponible
- La clé n'a jamais été stockée

**Solutions :**
```bash
# Forcer le stockage fichier
multiai config --store file

# Vérifier quel store est disponible
multiai config --store auto
```

---

## Vous ne trouvez pas la solution ?

- Ouvrez une [issue](https://github.com/lrochetta/multiai/issues) sur GitHub
- Consultez la [Configuration](/guide/configuration) pour vérifier vos fichiers
- Vérifiez la [référence des commandes](/reference/commands) pour les flags disponibles

> **Note :** multiai utilise l'encodage UTF-8. Les caractères accentués dans les commentaires et la documentation sont normaux.
