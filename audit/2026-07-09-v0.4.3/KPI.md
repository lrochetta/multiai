# KPI — Métriques de suivi post-lancement

> Objectif : mesurer l'impact de la publication v0.4.3 après les campagnes Show HN / Reddit / newsletters.

---

## 1. GitHub Stars

| Métrique | Cible J+7 | Cible J+30 | Cible J+90 |
|---|---|---|---|
| Stars | 50 | 200 | 500 |
| Watchers | 5 | 15 | 30 |
| Forks | 5 | 15 | 40 |

**Où vérifier :** https://github.com/lrochetta/multiai (bouton "Insights" > Traffic)
**Outil recommandé :** [Star History](https://star-history.com/) pour la courbe d'adoption

---

## 2. Issues & Contributions

| Métrique | Cible J+30 | Cible J+90 |
|---|---|---|
| Issues ouvertes (total) | 10-20 | 30-60 |
| Issues résolues | 80% | 85% |
| PRs externes mergées | 2 | 8 |
| Contributeurs (hors laurent) | 3 | 10 |

**À surveiller :**
- Temps moyen avant première réponse sur une issue (< 24h)
- Issues "good first issue" pour attirer les contributeurs
- Ratio issues résolues / issues fermées sans résolution

---

## 3. Téléchargements

| Canal | J+7 | J+30 | J+90 |
|---|---|---|---|
| npm (cumul) | 150 | 500 | 2000 |
| npm (hebdo) | 50 | 150 | 500 |
| Homebrew (cumul) | 20 | 100 | 500 |
| Scoop (cumul) | 10 | 50 | 200 |
| Go install / releases | 30 | 200 | 800 |
| **Total estimé** | **~210** | **~850** | **~3500** |

**Où vérifier :**
- npm : `https://api.npmjs.org/downloads/point/last-week/multiai` ou tableau de bord npm
- GitHub Releases : `https://github.com/lrochetta/multiai/releases` (compteur downloads par release)
- Homebrew : `brew info multiai` (affiche les analytics)

---

## 4. Visibilité web

| Métrique | Cible J+30 |
|---|---|
| Google indexed pages | 20+ |
| Sites référents | 5+ |
| Backlinks | 10+ |

**Où vérifier :** Google Search Console, Ahrefs, ou manual `site:github.com/lrochetta/multiai`

---

## 5. Engagement communautaire

| Plateforme | Cible J+7 | Cible J+30 |
|---|---|---|
| Show HN upvotes | 30 | — |
| Show HN commentaires | 10 | — |
| Reddit r/golang upvotes | 20 | — |
| Reddit r/programming upvotes | 50 | — |
| Reddit r/commandline upvotes | 30 | — |
| Reddit r/LocalLLaMA upvotes | 40 | — |
| Newsletter mentions | 1 | 3 |

---

## 6. Suivi des KPIs

### Checklist hebdomadaire
- [ ] Vérifier les stars GitHub (nouveautés de la semaine)
- [ ] Vérifier les issues ouvertes et temps de réponse
- [ ] Vérifier les téléchargements npm (hebdomadaire)
- [ ] Vérifier les PRs en attente
- [ ] Enregistrer dans `audit/KPI-log.md`

### Outils de suivi
- **GitHub Insights** : trafic, clones, referrers
- **npm stats** : téléchargements cumulés / hebdomadaires
- **Google Alerts** : "multiai CLI" pour détecter les mentions
- **Hackernews alerts** : `https://hn.algolia.com/` pour les mentions

---

## 7. Seuils d'alerte

| Indicateur | Seuil | Action |
|---|---|---|
| Issues non traitées > 48h | 5+ | Prioriser le triage |
| Crash report utilisateur | 1 | Hotfix immédiat |
| Téléchargements en baisse 2 semaines consécutives | — | Relancer campagne (blog post, v0.5) |
| npm vulnérabilité rapportée | 1 | Correctif de sécurité sous 24h |

---

## 8. Cibles de version suivante (v0.5)

Basé sur les KPIs, la roadmap v0.5 devrait inclure :
- Les fonctionnalités les plus demandées dans les issues
- Les providers additionnels demandés par la communauté
- Améliorations de la doc demandées dans les retours

---

*Dernière mise à jour : 2026-07-10*
