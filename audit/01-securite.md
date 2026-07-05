# Rapport d'Audit de Sécurité — Projet "AI CLI Launcher" (multiai v0.1.5)

**Auditeur** : Agent spécialisé sécurité — revue exhaustive du code source  
**Date** : 2026-06-23  
**Périmètre** : 4 scripts critiques + 17 fichiers .env + package.json + .gitignore + wrappers .cmd/.sh  
**Fichiers audités** : 24 fichiers, ~650 lignes de code

---

## Note de sécurité globale : **4/10**

| Critère | Points perdus |
|---|---|
| Pas de chiffrement au repos, clés en clair dans des fichiers .env | -3 |
| Pas d'exclusion `.env` dans `.gitignore` (risque de commit accidentel) | -1 |
| Clés API visibles dans l'environnement du processus enfant (fuite possible) | -1 |
| Périmètre de nettoyage incomplet (`$KnownEnvVars` limité à 30 variables) | -0.5 |
| `ExecutionPolicy Bypass` systématique / permissions filesystem absentes | -0.5 |

---

## S1 — Gestion des secrets

### 1.1 Stockage : clés API en clair dans des fichiers .env

Toutes les clés API sont stockées en **plaintext** dans 17 fichiers `.env` sous `configs/profiles/`. Exemples :

- `10-claude-anthropic-api.env` ligne 13 : `ANTHROPIC_API_KEY=PASTE_ANTHROPIC_API_KEY_HERE`
- `30-claude-deepseek-v4-pro.env` ligne 14 : `ANTHROPIC_AUTH_TOKEN=PASTE_DEEPSEEK_API_KEY_HERE`

**Aucun chiffrement au repos.** Aucun DPAPI, aucune gestion de keyring, aucun mécanisme de chiffrement.

### 1.2 Lecture des secrets : `Read-DotEnvFile` (lignes 121-141)

Parsing robuste via `IndexOf('=')` + `Substring(idx+1)`. Gère les guillemets. Aucun logging d'accès. Pas de gestion des valeurs multilignes.

### 1.3 Injection dans le processus enfant (ligne 555)

```powershell
Apply-ProfileEnv -Selected $selected
Test-RequiredSecrets -Selected $selected
& $command @launchArgs
```

**Problème majeur** : Les clés API sont exposées dans l'environnement du processus enfant. Vecteurs de fuite :
- **Linux** : `/proc/<PID>/environ` lisible par le même utilisateur
- **Windows** : `GetEnvironmentVariable` accessible à tout processus du même compte
- **Core dumps** : les clés figurent dans le dump si le processus crash
- **Swap** : les pages mémoire peuvent être swappées en clair

### 1.4 Fuite via arguments de ligne de commande

Risque minime : les clés ne sont pas passées en arguments. Cependant `--dangerously-skip-permissions` apparaît dans `ps aux` / `wmic process`.

### 1.5 Détection des placeholders : `Test-IsPlaceholder` (lignes 111-119)

```powershell
if ($trimmed -match '^(PASTE_|YOUR_|TA_CLE|REPLACE_ME|CHANGE_ME|sk-xxxx|xxx|TODO)') { return $true }
```

Couverture correcte mais incomplète : ne détecte pas les clés partielles, les variantes comme `your-api-key-goes-here`.

### 1.6 Masquage dans `Show-EffectiveEnv` (lignes 306-309)

La regex `(KEY|TOKEN|SECRET|PASSWORD)` ne couvre pas `AUTH_TOKEN`, `CREDENTIALS`, `PRIVATE_KEY`.

---

## S2 — Isolation

### 2.1 Nettoyage d'environnement : `Clear-RouterEnvironment` (lignes 241-244)

**Problème** : ne supprime que les 30 variables listées dans `$KnownEnvVars`. Variables non couvertes :
- `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`
- `AZURE_CLIENT_SECRET`, `GITHUB_TOKEN`, `NPM_TOKEN`
- `DATABASE_URL`, `SSH_AUTH_SOCK`, `PGPASSWORD`

**Impact** : secrets système hérités et transmis au CLI enfant → accessibles au modèle AI distant.

### 2.2 Efficacité du `CLEAR_ENV`

Efficace dans son périmètre : purge puis applique le nouveau profil. Mais limité aux variables connues.

### 2.3 Isolation des `CLAUDE_CONFIG_DIR`

Mécanisme étanche : chaque profil a son propre dossier (`~/.claude-deepseek-v4pro`, etc.). Le contenu (credentials, tokens de session) persiste entre exécutions.

### 2.4 `Expand-RouterValue` : fuite inter-profil (lignes 247-260)

Cherche `Process` → `User` → `Machine`. Si une variable n'est pas dans le profil .env, elle peut être résolue depuis un scope supérieur contenant un secret.

---

## S3 — Surface d'attaque

### 3.1 Injection de commandes via `COMMAND` (ligne 542) — **CRITIQUE**

```powershell
$command = $selected.Command  # provient du fichier .env
& $command @launchArgs
```

Toute personne pouvant écrire dans `configs/profiles/` peut exécuter du code arbitraire.

### 3.2 Injection via `ARGS`

Les arguments passent par l'opérateur `&` (pas d'interprétation shell), mais pourraient exploiter des vulnérabilités dans le CLI cible.

### 3.3 Validation des entrées utilisateur

| Point d'entrée | Validation | Risque |
|---|---|---|
| `-Tool` | Filtré via `Select-Profile` (`-ieq`) | Non exploitable |
| `-Profile` | `Find-Profile` par ID/Shortcut/filename | Pas d'injection possible |
| Choix `Read-Host` | `TryParse` en entier | Sécurisé |
| `ExtraArgs` | Aucune validation | Flags dangereux possibles |

### 3.4 `Split-ArgsSimple` (lignes 168-172)

Bug fonctionnel (pas de gestion des guillemets) : `-m "gpt 4.5"` → 3 arguments au lieu de 2.

### 3.5 `-ExecutionPolicy Bypass` (tous les .cmd)

Combine avec S3.1 : un attaquant qui écrit dans le dossier d'installation peut substituer `code-router.ps1`.

### 3.6 `Set-ProfileSecret` — pas de validation (lignes 326-341)

Écrit la valeur utilisateur directement dans le .env. Des retours à la ligne dans la valeur peuvent corrompre le format ou injecter de nouvelles variables.

---

## S4 — Bonnes pratiques

### 4.1 Fichiers .env dans le dépôt git : **PROBLÈME CRITIQUE**

Le `.gitignore` ne contient **PAS** de règle pour `*.env`. Les clés configurées peuvent être commitées accidentellement et exposées à perpétuité.

### 4.2 Package npm : les .env sont publiés

`"files": ["configs/", ...]` inclut tous les .env dans le package npm. Si un fichier avec une vraie clé est publié → clé dans le registre npm.

### 4.3 Gestion des erreurs : fuites d'information

Les messages d'erreur divulguent des chemins absolus : `"Fichier introuvable : $Path"`, `"Edite : $($Selected.Path)"`.

### 4.4 Permissions des fichiers de configuration

Aucun `chmod 600` appliqué. Windows : permissions NTFS par défaut (`Users` en lecture). Linux : tout processus du même utilisateur peut lire.

### 4.5 BMAD+ dans le code

`npx bmad-plus install` exécute du code depuis npm sans vérification de signature — risque standard de supply chain npm.

---

## S5 — npm / chaîne d'approvisionnement

### 5.1 Package `multiai` v0.1.5

- Binaire `bin/multiai.js` — pas de dépendances npm
- `spawnSync` avec `shell: false` — bonne pratique
- `configs/` inclus dans la publication → 17 fichiers .env avec placeholders
- Pas de `.npmignore`, pas de `prepublishOnly` de vérification

### 5.2 Aucune vérification d'intégrité

Pas de checksum, pas de signature, pas de lockfile.

---

## Top 5 vulnérabilités

| # | Sévérité | Titre | Fichier : Ligne |
|---|---|---|---|
| **1** | **CRITIQUE** | `.env` absent du `.gitignore` — risque de commit de clés | `.gitignore` |
| **2** | **CRITIQUE** | Exécution arbitraire via champ `COMMAND` des profils .env | `code-router.ps1` : 542, 555 |
| **3** | **HAUTE** | Clés API exposées dans l'environnement du processus enfant | `code-router.ps1` : 273, 555 |
| **4** | **HAUTE** | Périmètre `$KnownEnvVars` incomplet — fuite de secrets système | `code-router.ps1` : 49-61 |
| **5** | **MOYENNE** | Aucune protection des fichiers .env sur le disque | `install.ps1` : 57-73 |

---

## Recommandations

1. **IMMÉDIAT** : Ajouter `configs/profiles/*.env` dans `.gitignore`
2. **IMMÉDIAT** : Valider que `COMMAND` est un binaire connu (`claude`, `codex`, `opencode`)
3. **HAUTE** : Remplacer `Clear-RouterEnvironment` par un nettoyage complet (sauf PATH, HOME)
4. **HAUTE** : Appliquer `chmod 600` / `icacls` restrictif sur les fichiers .env
5. **MOYENNE** : Utiliser `SecureString` PowerShell et injecter les clés via pipe plutôt que via env
6. **MOYENNE** : Ajouter `.npmignore` excluant `*.env` et script `prepublishOnly`
7. **FAIBLE** : Valider les entrées dans `Set-ProfileSecret` contre l'injection de newlines
8. **FAIBLE** : Remplacer `Split-ArgsSimple` par un parseur respectant les guillemets
