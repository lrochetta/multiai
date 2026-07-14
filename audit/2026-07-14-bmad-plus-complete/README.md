# Audit BMAD+ complet — 2026-07-14

Cet audit a été piloté par **Nexus** en mode parallèle :

- **Atlas** : produit, fonctionnalités, parcours et positionnement ;
- **Forge** : architecture, code, maintenabilité et installation Windows ;
- **Sentinel** : qualité, sécurité, supply chain et revue adversariale ;
- **Nexus** : correctif PATH, arbitrage, roadmap et consolidation.

## Décision

| Sujet | État |
|---|---|
| Correctif PATH Windows | **Implémenté et testé localement** |
| Tests npm | **25/25 verts** |
| Publication v0.6.7 | **NO-GO** |
| Prochaine preuve Windows | E2E `Apply` du tarball final dans une VM propre, puis nouvelle console |

## Rapports

1. [Synthèse exécutive](00-synthese-executive.md) — verdict, description du
   projet, risques, gates et cible.
2. [Produit et fonctionnalités](01-produit-fonctionnalites.md) — personas,
   parcours, benchmark, écarts et score Atlas.
3. [Architecture, code et Windows](02-architecture-code-windows.md) — structure,
   dette, contrats, updater, release et revue du correctif PATH.
4. [Qualité et sécurité](03-qualite-securite.md) — modèle de menace, preuves,
   matrice de risques, gates et score Sentinel.
5. [Correctif PATH Windows](04-correctif-path-windows.md) — cause racine,
   architecture, protections, tests et E2E restant.
6. [Roadmap priorisée](05-roadmap-priorisee.md) — P0/P1/P2, propriétaires,
   Definition of Done et KPI.

Le rapport produit fige le défaut PATH tel qu'il existait au début de l'audit.
Les documents `00`, `02`, `03` et `04` décrivent l'état **après**
correctif et font foi pour son statut final.

## Périmètre couvert

- proposition de valeur, fonctionnalités et UX CLI ;
- architecture Go, modularité, fiabilité et portabilité ;
- packaging npm/npx et installation Windows ;
- secrets, hooks, configuration projet et registre ;
- updater, workflows CI/release, signatures et dépendances ;
- tests, documentation, versions et canaux de distribution ;
- roadmap pour faire de multiai un plan de contrôle local de référence.

## Artefacts du correctif

- `multiai-go/packaging/npm/bin/multiai.js`
- `multiai-go/packaging/npm/lib/windows-path.js`
- `multiai-go/packaging/npm/scripts/ensure-user-path.ps1`
- `multiai-go/packaging/npm/bin/multiai.test.js`
- `multiai-go/packaging/npm/lib/windows-path.test.js`

## Règle de sortie

Ne pas créer de tag, release GitHub ou publication npm tant que la gate P0 de
la synthèse n'est pas entièrement fermée et que la matrice CI du même SHA n'est
pas verte sur macOS, Ubuntu et Windows.
