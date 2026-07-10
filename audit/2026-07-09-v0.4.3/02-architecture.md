# Audit Architecture — multiai Go v0.4.3

**Date :** 2026-07-09
**Score global : 7.8/10**

---

## Structure des packages

```
cmd/multiai/        → Point d'entrée + registre de sous-commandes
internal/
  assets/           → Embed des templates .env (go:embed)
  catalog/          → Catalogue data-driven (providers.yaml intégré)
  cli/              → Lancement, hooks, fallback, affichage, completion
  config/           → Assistant config + effacement clés
  env/              → Sous-environnement (%VAR%, whitelist)
  fsutil/           → Écriture atomique
  logging/          → Journal + sessions JSONL
  menu/             → Menus interactifs (3 niveaux)
  onboarding/       → Assistant premier démarrage
  openrouter/       → Découverte modèles (API, cache, recherche)
  profile/          → Chargement .env, YAML, .multiai.yaml
  secret/           → Credential store AES-256-GCM
  update/           → Auto-update GitHub Releases
pkg/
  dotenv/           → Parseur .env générique
```

---

## Patterns architecturaux

| Pattern | Emplacement | Évaluation |
|---|---|---|
| Data-driven catalog | `catalog/providers.yaml` | ✅ Excellent — ajout sans code |
| Sentinel | `secret/sentinel.go` | ✅ Robuste — invariant vérifié par test |
| Atomic writes | `fsutil/`, `config/`, `secret/`, `openrouter/` | ⚠️ Dupliqué 4 fois |
| Hooks lifecycle | `cli/hooks.go` | ✅ Template + escape correct |
| Fallback chain | `cli/fallback.go` | ✅ Interruption-aware |
| sync.OnceValues | `catalog/catalog.go` | ✅ Chargement paresseux |
| Platform build tags | `secret/store_*.go` | ⚠️ 3 stubs identiques |

---

## Graphe d'imports

```
main → assets, catalog, cli, config, menu, onboarding, openrouter, profile, update
cli → env, profile, secret, logging, dotenv
config → catalog, cli, env, profile, secret, dotenv   ← ⚠️ dépend de cli pour l'affichage
menu → cli, profile, dotenv                            ← ⚠️ dépend de cli pour l'affichage
onboarding → catalog, cli, config, logging, profile    ← ⚠️ dépend de cli pour l'affichage
openrouter → fsutil
profile → dotenv, yaml
secret → fsutil
update → (stdlib)
```

✅ **Pas de dépendance circulaire**
⚠️ Couplage display : `config`, `menu`, `onboarding` → `cli` uniquement pour `PrintWarning/Error/Success/Info/Colorize`

---

## Qualité des interfaces

| Interface | Évaluation |
|---|---|
| `secret.Store` | ✅ Excellente — minimale, complète, build tags |
| `profile.Profile` | ✅ Struct de données, 22 champs bien documentés |
| `catalog.Catalog` | ✅ Type concret avec méthodes, pas besoin d'interface |
| `openrouter.ModelCache` | ✅ Avec `Source`, `FetchedAt`, `Warning` |
| `context.Context` | ❌ Absent — pas de propagation de cancellation |

---

## Gestion de configuration

**Ordre de résolution** (bon, sécurisé) :
1. `MULTIAI_PROFILES_DIR`
2. `<executable dir>/configs/profiles`
3. CWD-relative **uniquement si** `MULTIAI_DEV` est set
4. `<user config dir>/multiai/profiles` + extraction templates

⚠️ Logique de résolution dupliquée entre `main.go:getProfilesDir()` et `openrouter.ActiveProfilesDir()`

---

## Points d'extension

- **Ajout provider** : éditer `providers.yaml` + créer `.env` → 0 code Go ✅
- **Ajout outil** : whitelist + profilegen + templates
- **Ajout commande** : `init()` avec `register()` → pas de merge hotspot
- **Ajout backend credential** : implémenter `secret.Store` + build tag

---

## Scores détaillés

| Critère | Score |
|---|---|
| Séparation des responsabilités | 9/10 |
| Extensibilité (data-driven) | 10/10 |
| Sécurité (sentinel, atomic, whitelist) | 8/10 |
| Qualité des interfaces | 7/10 |
| Gestion des erreurs | 7/10 |
| Documentation du code | 9/10 |
| Couplage entre packages | 6/10 |
| Portabilité | 7/10 |
| Gestion de configuration | 8/10 |

---

## Recommandations prioritaires

### 🔴 Haute priorité

| # | Action | Impact |
|---|---|---|
| R1 | Extraire `display/` de `cli/` | Casse le couplage métier→orchestration |
| R2 | Unifier atomic writes → `fsutil.WriteFileAtomic` | Supprime 3 duplications |
| R3 | Centraliser `getProfilesDir()` dans `profile/` | Élimine divergence |

### 🟠 Moyenne priorité

| # | Action |
|---|---|
| R4 | Propager `context.Context` dans les appels HTTP |
| R5 | Connecter le format YAML au pipeline de lancement |
| R6 | Remplacer `init()` logging par `Init() error` |

### 🟢 Basse priorité

| # | Action |
|---|---|
| R7 | Implémenter backend natif (Windows Credential Manager d'abord) |
| R8 | Test d'intégration pour `ValidateAndLaunch` complet |
| R9 | Uniformiser les permissions fichiers en constantes |
