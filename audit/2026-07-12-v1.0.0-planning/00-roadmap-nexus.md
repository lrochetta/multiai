Fichier ecrit : `D:\travail\DEV\multiai\audit\2026-07-12-v0.6.0-launch\01-roadmap-v0.6.0-to-v1.0.0.md`

## Synthese pour laurent

**31 stories brutes consolidees en 25 uniques** apres 7 fusions (S9.3+S10.3, S9.2+S10.5, M3.1+Huldah3, M3.2+Huldah4, M1.0+AtlasS1, Huldah5+AtlasS5, S4+Discord).

### 5 sprints, ~45 JH

| Sprint | Jours | JH | Objectif |
|--------|-------|----|----------|
| **A** — CI Foundation | J0→J+5 | 5 | CI stable, quality gates bloquants, tests cross-platform |
| **B** — Release v0.6.0 | J+5→J+10 | 7.5 | Registre, CHANGELOG auto, Homebrew/Scoop, **tag v0.6.0** |
| **C** — Documentation v1.0.0 | J+10→J+18 | 11 | ADR, godoc, VitePress 20+ pages, CONTRIBUTING.md v2 |
| **D** — Qualite v1.0.0 | J+18→J+25 | 9 | Coverage 75%+, benchmarks, securite audit >=9.0/10 |
| **E** — Community Launch | J+18→J+45 | 23.5 | PH+HN+Reddit+newsletters, video, content, ambassadeurs, partnerships |

### Chemin critique (9 stories, ~18j calendaires)
```
A1(Fix CI) → A2(Quality gates) → B5(Release v0.6.0) → C1(ADR) → C2(Doc API) → C3(VitePress) → D3(Security audit) → E1(Launch campaign)
```

### Point de bascule
**B5 (Release v0.6.0) a J+10** — si la CI n'est pas verte ou les secrets manquants, tout derape.

### 3 actions immediates pour toi
1. Creer PAT classic (scope repo) → `TAP_GITHUB_TOKEN` dans secrets du repo GitHub
2. Creer repos `homebrew-tap` et `scoop-bucket` (publics)
3. Valider que `profiles-multiai` est cree (ou le creer)

### Budget par agent
- **Forge** : 5.5 JH (CI + release)
- **Sentinel** : 10.5 JH (qualite + tests)
- **Huldah+Bezalel** : 14.5 JH (documentation)
- **Atlas** : 23 JH (community + growth)
- **Nexus** : 2 JH (registre + coordination)

Le chemin critique fait 18j calendaires si execute sequentiellement. En pratique, les sprints C/D/E peuvent demarrer en parallele de B (les agents Atlas et Huldah ne dependent pas de Forge). La release v0.6.0 est realisee a J+10, la v1.0.0 cible J+30 a J+45.