# Rapport Audit Qualité Code — AI CLI Launcher (multiai v0.1.5)

**Projet** : `D:\travail\DEV\multiai\code-cli-router-pack (1)\code-cli-router-pack`  
**Fichiers audités** : `code-router.ps1` (557 L), `bin/multiai.js` (93 L), `install.ps1` (183 L), `install.sh` (131 L), `package.json`, profils `.env`, wrappers `.cmd`  
**Date** : 2026-06-23

---

## Note de qualité globale : **6.5/10**

| Critère | Note |
|---|---|
| Structure et organisation | 7/10 |
| Qualité PowerShell | 6.5/10 |
| Qualité Node.js | 7.5/10 |
| Robustesse | 5/10 |
| Maintenabilité | 6/10 |
| Tests | 0/10 |

---

## Q1 — Structure et organisation

### Points positifs
- Séparation claire : routeur PowerShell (cœur), launcher Node.js (point d'entrée npm), installateurs OS-spécifiques
- `package.json` bien configuré : `"bin"`, champ `files`
- Profils `.env` bien organisés avec convention de nommage numérique (`00-`, `10-`, `20-`...)

### Incohérences
1. **Double point d'entrée latéral** : `bin/multiai.js` ne gère que `install` et `help`. Le lancement des CLIs se fait via les `.cmd`/`.sh`. Le binaire npm est inutile pour l'usage quotidien.
2. **Duplication install.ps1 / install.sh** : même logique (copie fichiers, préservation .env, PATH) dupliquée dans 2 fichiers

---

## Q2 — Qualité PowerShell (code-router.ps1)

### Bonnes pratiques observées
- `$ErrorActionPreference = 'Stop'` (ligne 45)
- `Join-Path` pour construction de chemins
- `[ordered]@{}` pour `$ProviderCatalog`
- `[pscustomobject]@{}` pour les objets profil
- `-LiteralPath` systématique
- `[Environment]::SetEnvironmentVariable(..., 'Process')` pour isolement

### Anti-patterns

**1. `$items +=` array anti-pattern (lignes 146, 159)**  
Les tableaux PowerShell sont immutables. Chaque `+=` crée un nouveau tableau → complexité O(n²).

**2. Fonctions sans `[CmdletBinding()]`** (lignes 106-109, 111)  
Perte de fonctionnalités PowerShell : `$PSCmdlet`, `-Verbose`, etc.

**3. `throw` comme unique mécanisme d'erreur**  
Aucun bloc `try/catch` dans tout le fichier. Comportement tout-ou-rien.

**4. `Read-Host` sans validation avancée** (lignes 188, 211, 487, 509)  
Si l'utilisateur tape "q", le `[int]::TryParse` échoue et `throw` est déclenché.

---

## Q3 — Qualité Node.js (multiai.js)

- `spawnSync` avec `shell: false` → bonne pratique sécurité
- `result.status != null ? result.status : 1` → gère correctement les processus tués par signal
- Normalisation CRLF→LF pour scripts bash générés sous Windows (lignes 82-87)
- Pas de validation d'existence de `install.ps1` avant `spawnSync` — erreur non interceptée si package corrompu

---

## Q4 — Robustesse

### Cas limites non gérés
- **Profils absents** : `Get-Profiles` retourne tableau vide → `Select-Tool` tente de grouper 0 profils → message d'erreur imprécis
- **Permissions fichier** : `Set-ProfileSecret` sans `try/catch` — si `.env` en lecture seule, `Set-Content` lève une erreur
- **Processus tué par signal** : `exit $LASTEXITCODE` → si `$LASTEXITCODE` est `$null`, sort avec code 0 (devrait être 1)

### Codes de sortie
- Pas de codes de sortie discriminants entre types d'erreur
- Seuls `exit 0` et `exit $LASTEXITCODE` sont utilisés

---

## Q5 — Maintenabilité

### Lisibilité
- Commentaires de section clairs (`# ── Point d entree ──`)
- Fonctions documentées (`.SYNOPSIS`, `.DESCRIPTION`, `.EXAMPLES`, `.NOTES`)
- Messages en français / code en anglais → cohérent avec CLAUDE.md

### Duplication
- `install.ps1` et `install.sh` dupliquent la même logique → 2 points de maintenance
- Chemins `configs/profiles` dupliqués conceptuellement entre `code-router.ps1` et `install.ps1`

### Tests
- **Zéro test** — aucun fichier de test, aucun dossier `test/`, `spec/`, `__tests__`
- Les fonctions pures (`Read-DotEnvFile`, `Test-IsPlaceholder`, `Split-ArgsSimple`, `Expand-RouterValue`) sont des candidates parfaites pour Pester

---

## Q6 — Bugs potentiels

### BUG-1 : `Read-DotEnvFile` ignore le préfixe `export` — **HAUTE**
**Ligne 132** : `$key = $line.Substring(0, $idx).Trim()` — si le fichier contient `export ANTHROPIC_API_KEY=sk-xxx`, la clé devient `"export ANTHROPIC_API_KEY"`, qui ne correspond à rien. La variable n'est jamais appliquée.

**Correction** : Ajouter après la ligne 132 :
```powershell
if ($key -match '^export\s+') { $key = $key -replace '^export\s+', '' }
```

### BUG-2 : `Split-ArgsSimple` ne préserve pas les guillemets — **MOYENNE**
**Ligne 171** : `$ArgString -split '\s+'` — `--prompt "Hello world"` devient 3 arguments au lieu de 2. Les guillemets littéraux dans les arguments cassent le parsing du CLI cible.

### BUG-3 : Références obsolètes `cc` et `aicode` — **FAIBLE**
- **Ligne 225** : `"Lance 'cc -List'"` — `cc` est l'ancien nom
- **Ligne 288** : `"aicode -Configure"` — `aicode` est l'ancien nom

### BUG-4 : Messages post-install invalides — **MOYENNE**
- **install.ps1:162-163** : référence `config` et `bmad` comme commandes standalone
- **install.sh:122-124** : référence `aicode.sh` (ancien nom)

### BUG-5 : Détection fragile des profils dans install.ps1 — **MOYENNE**
**Lignes 64-66** : 3 patterns différents pour couvrir les séparateurs Windows et Unix. Si `$relative` contient `/` sur Windows (checkout git avec `core.autocrlf` désactivé), la détection échoue → écrasement de profils existants.

---

## Top 5 problèmes

| # | Problème | Ligne | Sévérité |
|---|---|---|---|
| 1 | `Read-DotEnvFile` ignore le préfixe `export` | code-router.ps1:132 | HAUTE |
| 2 | `Split-ArgsSimple` ne gère pas les guillemets | code-router.ps1:171 | MOYENNE |
| 3 | Références obsolètes `cc` / `aicode` | code-router.ps1:225,288 | FAIBLE |
| 4 | Messages post-install invalides | install.ps1:162-163, install.sh:122-124 | MOYENNE |
| 5 | Zéro test / couverture | Projet entier | MOYENNE |

---

## Quick wins

1. **Corriger `export` dans `Read-DotEnvFile`** (ligne 132, +2 lignes) — tous les .env format Unix fonctionnent
2. **Corriger les références obsolètes** (lignes 225, 288) — `'cc -List'` → `'multiai -List'`, `'aicode -Configure'` → `'multiai -Configure'`
3. **Corriger messages post-install** (install.ps1:162-163, install.sh:122-124) — utiliser les noms actuels
4. **Remplacer `$items +=` par `List[object]`** (lignes 146, 159) — meilleure pratique PowerShell
5. **Ajouter tests Pester** pour les 4 fonctions pures — couvrirait ~40% de la logique
