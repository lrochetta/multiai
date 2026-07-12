---
title: Patterns
description: Reusable patterns observed in this project
created: "2026-06-23"
project: "multiai"
---

# Patterns

Reusable patterns that work well in this project.

### Native npm Bootstrap with OS Trust
- **Problem**: Télécharger un binaire GitHub depuis un lifecycle npm derrière un proxy ou une CA locale sans désactiver TLS.
- **Shape**: Avec Node 24.14+ comme minimum du bootstrap npm, fusionner `tls.getCACertificates('default')` et `('system')` avec `setDefaultCACertificates` avant la première requête, puis activer le proxy d'environnement. Conserver un timeout global, une limite de redirections et la vérification SHA256 avant extraction. Garder les feature checks comme défense supplémentaire.
- **Trade-off**: Le magasin de confiance de l'OS peut contenir des CA d'entreprise supplémentaires; c'est cohérent avec la politique de confiance locale mais plus large que le bundle Mozilla de Node.
- **Status**: `validated`

## Format

```
### <pattern name>
- **Problem**: what it solves
- **Shape**: the core idea in 2-3 sentences
- **Trade-off**: what it costs
- **Status**: `candidate` / `validated` / `deprecated`
```

---

### Sentinel Credential Store
- **Problem**: Comment stocker des clés API sans qu'elles apparaissent en clair dans les fichiers .env, tout en restant simple à utiliser ?
- **Shape**: Le credential store chiffre la clé (AES-256-GCM) et écrit un sentinel `__MULTIAI_CREDSTORE__` dans le .env. Au lancement, `resolveStoredSecrets()` détecte le sentinel et va chercher la vraie valeur dans le store. Invariant : sentinel dans le fichier ⇒ valeur dans le store. L'écriture store FIRST, puis fichier.
- **Trade-off**: Sécurité renforcée mais dépend d'un fichier maître (.masterkey) stocké à côté du ciphertext. Les stores natifs OS (Windows Credential Manager, macOS Keychain) sont prévus pour renforcer.
- **Status**: `validated`

### Colored Status Menu
- **Problem**: Les utilisateurs ne savent pas d'un coup d'oeil quels fournisseurs/profils sont configurés.
- **Shape**: Une fonction `StatusColor(configured, total int) string` retourne un code ANSI : vert si tout configuré, jaune si partiel, gris si rien. Exportée pour être utilisée dans tous les menus. Respecte `NO_COLOR`. La ligne entière est colorée, pas seulement le badge.
- **Trade-off**: Dépend de la détection des placeholders (`dotenv.IsPlaceholder`). Si un placeholder n'est pas reconnu, une clé configurée peut apparaître comme non configurée.
- **Status**: `validated`

### Profile-as-EnvFile
- **Problem**: Chaque combinaison CLI×fournisseur nécessite des variables d'environnement différentes.
- **Shape**: Chaque profil est un fichier `.env` autonome avec métadonnées (PROFILE_ID, SHORTCUT, TOOL, ORDER...) et variables d'env. Le routeur lit, parse, applique dans le scope Process, lance le CLI.
- **Trade-off**: Simplicité maximale mais duplication (même clé dans plusieurs profils).
- **Status**: `validated`

### Process-Scoped Environment Isolation
- **Problem**: Lancer plusieurs CLIs IA avec des clés différentes sans contamination.
- **Shape**: Liste blanche de ~30 variables système. Le profil injecte ses variables. Rien n'est persistant.
- **Trade-off**: Efficace mais ne couvre pas les secrets système non listés.
- **Status**: `validated`

### Data-Driven Provider Catalog
- **Problem**: Ajouter un fournisseur nécessitait de modifier le code source (ProviderCatalog codé en dur).
- **Shape**: `providers.yaml` embarqué (embed.FS) définit tous les fournisseurs : ID, nom, région, URL, shortcuts, VarMap, KeyPattern. Le catalogue est chargé et validé au démarrage. Ajouter un fournisseur = éditer le YAML.
- **Trade-off**: Extensible sans code. Mais le YAML doit rester cohérent avec les profils .env correspondants.
- **Status**: `validated`

### OpenRouter Model Cache with Graceful Degradation
- **Problem**: L'API OpenRouter peut être lente ou down.
- **Shape**: Cache JSON 1h. Au lancement : retourner le cache, rafraîchir en arrière-plan si >1h. Si API down, utiliser le cache avec avertissement.
- **Trade-off**: Résilience au prix de données potentiellement périmées.
- **Status**: `candidate`

### Atomic File Write (temp + fsync + rename)
- **Problem**: Écriture de fichiers .env interrompue = corruption.
- **Shape**: `WriteFileAtomic` : écrire dans un fichier temporaire unique (CreateTemp), fsync, fermer, puis rename atomique. Le fichier original n'est jamais dans un état incohérent.
- **Trade-off**: Fonctionne sur tous les FS POSIX. Peut échouer sur certains FS réseau (NFS) où rename n'est pas atomique.
- **Status**: `validated`

### CLI Command Shim (npm native binary)
- **Problem**: Le package npm doit lancer un binaire Go natif sans compilation.
- **Shape**: `bin/multiai.js` shim Node.js : vérifie que le binaire existe dans `bin/native/`, le lance avec `spawnSync`, forward stdio et exit code. `install.js` (postinstall) télécharge le binaire depuis GitHub Releases avec vérification SHA256.
- **Trade-off**: Fonctionne sans compilation. Mais dépend de GitHub Releases (repo doit être public) et du réseau au premier install.
- **Status**: `validated`

### Interactive Provider Configuration
- **Problem**: Configurer 13 fournisseurs avec des noms de variables différents par CLI.
- **Shape**: Menu interactif groupé par région, une saisie par fournisseur propagée à tous les profils du groupe. Statut [OK]/[~~]/[--] coloré.
- **Trade-off**: Excellente UX. Le mapping fournisseur→profils est dans le catalogue YAML.
- **Status**: `validated`
