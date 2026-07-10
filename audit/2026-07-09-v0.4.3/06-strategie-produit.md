# Audit Stratégie Produit — multiai v0.4.3

**Date :** 2026-07-09
**Score global : 6.5/10**

---

## Scores par dimension

| Dimension | Note |
|---|---|
| Positionnement & Marché | 7/10 |
| Produit & Fonctionnalités | 7.5/10 |
| Sécurité | 7/10 |
| Qualité & Tests | 5.5/10 |
| Distribution & Packaging | 6/10 |
| Communauté & Adoption | 1/10 |
| Stratégie & Vision | 8/10 |

---

## Proposition de valeur

> "Un seul outil pour lancer n'importe quel CLI d'IA avec n'importe quel fournisseur, sans fuite de clés, sans configuration manuelle."

Positionnement clair et défendable. **Aucun concurrent direct** ne couvre exactement ce segment (routeur d'environnement pour CLI d'IA avec isolation).

---

## Matrice concurrentielle

| Concurrent | Type | vs multiai |
|---|---|---|
| OpenRouter API | Gateway multi-modèle | API-only, pas de gestion d'env |
| LiteLLM | Proxy local | Docker requis, pas d'isolation env |
| cc-switch | GUI switch provider | Mono-CLI (Claude Code), pas de fallback |
| Usage manuel (export) | Aucun outil | Risque de fuite, perte de temps |

---

## Forces et faiblesses

### Forces
- Isolation d'environnement par liste blanche (unique sur le marché)
- 37 profils pré-configurés, 13 fournisseurs
- Fallback chains automatiques
- 1 seule dépendance Go — surface d'attaque minimale
- Architecture data-driven (ajout fournisseur sans code)

### Faiblesses
- **0 stars, 0 forks, 0 issues** — projet invisible
- **100% français** — bloque l'adoption internationale
- Pas de cost tracking réel (tokens, coût)
- Roadmap désynchronisée de la réalité
- Fonctionnalités promises non implémentées dans le README

---

## Top 5 actions stratégiques

| # | Action | Urgence | Impact |
|---|---|---|---|
| 1 | **Sortir de 0 stars** — Publier HN, Reddit, newsletters | CRITIQUE | Survie du projet |
| 2 | **Sécuriser la supply chain** — Hardcoder URL GitHub, Cosign, gitleaks | CRITIQUE | Bloque v0.5.0 |
| 3 | **Synchroniser ROADMAP.md** avec la réalité | HAUTE | Crédibilité |
| 4 | **50% couverture de test** sur tous les packages | HAUTE | Refactoring safe |
| 5 | **Déployer Homebrew/Scoop/AUR** — `skip_upload: false` | MOYENNE | Adoption macOS/Linux |

---

## Verdict

**Projet techniquement excellent, commercialement invisible.** Le code est bon (7-8/10) mais sans adoption, la question de la stratégie reste académique. La priorité absolue : se faire connaître.
