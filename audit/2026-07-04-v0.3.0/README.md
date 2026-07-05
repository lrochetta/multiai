# Audit complet multiai v0.3.0 — 2026-07-04

**Projet** : multiai — routeur multi-IA CLI (Claude Code / Codex / OpenCode), Go « primaire » + PowerShell « legacy » (npm)
**Version auditée** : npm `multiai@0.3.0` (PowerShell) / binaire Go `0.2.1` / packaging `0.5.0`
**Orchestrateur** : Nexus (BMAD+) · **Référence delta** : audit v0.2.1 du 2026-06-23 (5.5/10)

## Méthodologie

- **7 auditeurs BMAD+ en parallèle**, mandatés par Nexus via workflow déterministe :
  - **Atlas ×2** (Strategist) — produit & fonctionnalités, distribution & packaging
  - **Forge ×2** (Architect-Dev) — architecture, qualité de code
  - **Sentinel ×2** (Quality) — tests & CI/CD, UX/DX & documentation
  - **Shadow** (OSINT & Security) — sécurité
- **Contre-vérification adversariale** : chaque finding **critical** soumis à 2 réfutateurs indépendants à angles différents, chaque **high** à 1 réfutateur. **99 agents au total** sur le run.
- Vérifications outillées réelles : `go build/vet/test -cover`, `gofmt -l`, `git ls-files`/`ls-remote`, `npm view`, `gh repo view`, tests HTTP live des URL d'installation.

**Bilan des 135 findings** : 5 critical · 46 high · 56 medium · 28 low (sévérités finales, post-recalibrage).
**Contre-vérification** : 62 findings passés au crible adversarial (tous les critical/high initiaux) → **41 CONFIRMED**, **21 PARTIAL** (faits confirmés, sévérité ou portée recalibrée à la baisse — dont 15 critical initiaux rétrogradés), **0 REFUTED**. Les 73 medium/low restants n'ont pas été contre-vérifiés. **Aucun finding n'a été écarté.** Les 5 critical finaux sont tous CONFIRMED.

---

## Scores par dimension

| Fichier | Dimension | Auditeur | Score /10 |
|---|---|---|---|
| [01-produit-fonctionnalites.md](01-produit-fonctionnalites.md) | Produit & fonctionnalités | Atlas | 3.5 |
| [02-distribution-packaging.md](02-distribution-packaging.md) | Distribution & packaging | Atlas | **2.5** |
| [03-architecture.md](03-architecture.md) | Architecture | Forge | 3.5 |
| [04-qualite-code.md](04-qualite-code.md) | Qualité de code | Forge | 4.5 |
| [05-tests-cicd.md](05-tests-cicd.md) | Tests & CI/CD | Sentinel | 3.0 |
| [06-ux-dx-docs.md](06-ux-dx-docs.md) | UX / DX / Documentation | Sentinel | 4.5 |
| [07-securite.md](07-securite.md) | Sécurité | Shadow | 4.5 |
| **Moyenne** | | | **3.7** |

### Delta vs v0.2.1 (2026-06-23 : 5.5/10) — pourquoi −1.8 ?

La baisse est réelle mais elle mélange trois effets qu'il faut séparer honnêtement :

1. **Régressions réelles.** Le « fix » v0.2.1 #9 (updateEnvFile atomique) a introduit une régression pire que le mal : le wizard Go écrit désormais la sentinelle `__MULTIAI_CREDSTORE__` dans le .env et **plus rien ne la relit** — le flux central config→launch est cassé (critical confirmé 4 fois). L'incohérence Go/PS (#12) s'est **aggravée** : toute la v0.3.0 a été livrée dans le PowerShell « legacy », le Go « primaire » est figé à 0.2.1 avec 5 fournisseurs contre 14 et 17 profils contre 37-38. Sur les 3 dimensions comparables, la chute est de −1.0 chacune : code 5.5→4.5, UX 5.5→4.5, sécurité 5.5→4.5 (le rapport sécurité le dit explicitement : « la confiance dans les affirmations de sécurité du projet a régressé »).
2. **Périmètre élargi.** 4 dimensions nouvelles (produit 3.5, distribution 2.5, architecture 3.5, tests/CI 3.0) n'étaient pas notées en v0.2.1 ; elles moyennent à 3.1 et tirent mécaniquement la note globale sous les 4.5 des dimensions historiques. La distribution à 2.5 (« village Potemkine » : repo privé, zéro release, zéro tag distant, 4 canaux sur 5 morts) n'avait jamais été mesurée.
3. **Calibrage plus sévère.** L'audit v0.2.1 avait crédité des forces sur la foi des claims : « CI/CD complète » et « 45+ tests » figuraient dans ses forces confirmées — la contre-vérification 2026-07-04 prouve que **la CI n'a jamais exécuté un seul job** (workflows sous `multiai-go/.github/`, jamais lus par GitHub, trigger `main` vs branche `master`) et compte 32 fonctions de test. Une partie de la baisse est donc une correction de la mesure, pas une dégradation du produit.

---

## Verdict global

L'ambition affichée — « LE meilleur routeur multi-IA du marché » — repose aujourd'hui sur un seul pilier qui fonctionne : le routeur **PowerShell publié sur npm** (37 profils, 14 fournisseurs, régions, fallback chains, erase keys), précisément la partie que le projet qualifie de « legacy ». L'implémentation Go « primaire » est figée deux versions fonctionnelles en arrière, son flux central config→launch est cassé, et aucun de ses canaux de distribution n'a jamais fonctionné de bout en bout (repo privé, zéro release, CI fantôme, checksums placeholders). Le README décrit un produit imaginaire : `models`/`search`/`compare`, cost logging, cache 1h, héritage YAML, hooks — tout est vaporware ou code mort, couronné d'un badge « 9.5/10 » auto-décerné quand l'audit mesure 3.7. Les différenciateurs réels existent (isolation env par whitelist, fallback chains au lancement, couverture 3 CLIs × 14 fournisseurs) et aucun concurrent lanceur ne les combine. Mais le gap n'est pas d'abord technique : c'est un problème d'**intégrité** — ~40 % des claims fonctionnels sont invérifiables ou faux, rédhibitoire pour l'adoption par des devs exigeants. La priorité absolue n'est pas une feature de plus : réparer le flux config→launch, tracker profils et docs dans git, purger le README, et produire une première release installable par un inconnu.

---

## Top 10 des problèmes (toutes dimensions, priorisé par sévérité vérifiée)

Les doublons inter-dimensions sont regroupés ; tous les IDs sources sont cités.

| Rang | IDs | Sévérité | Verdict | Titre | Dimensions |
|---|---|---|---|---|---|
| 1 | 01-02, 03-01, 04-01, 07-03 | **critical** | CONFIRMED ×4 | Flux config→launch Go cassé : la sentinelle `__MULTIAI_CREDSTORE__` est écrite mais jamais relue (credential store write-only) — le littéral est exporté comme clé API au CLI enfant ; si le store échoue, la clé saisie est perdue silencieusement | Produit, Architecture, Code, Sécurité |
| 2 | 03-04, 06-03 | **critical** | CONFIRMED ×2 | Profils `.env` Go (0 tracké) et doc VitePress exclus de git (`*.env` et `docs/` non ancrés dans .gitignore) : un clone frais sort en exit 2 sur toutes les commandes — first-run cassé depuis GitHub | Architecture, UX/Docs |
| 3 | 01-09, 02-04, 03-06, 04-10 | high | CONFIRMED ×3, PARTIAL ×1 | Binaire Go installé inutilisable : profils cherchés uniquement près de l'exe ou du cwd, ni `go:embed` ni répertoire utilisateur, et **aucun canal ne livre les profils** | Produit, Distribution, Architecture, Code |
| 4 | 01-06, 07-06, 02-01, 03-05, 04-04, 05-01 | high | CONFIRMED ×2, PARTIAL ×4 | CI/CD fantôme : workflows sous `multiai-go/.github/` (jamais lus par GitHub) + trigger `branches: [main]` vs branche `master` — lint, tests 6×, gosec, goreleaser, Cosign, SBOM, Dependabot : **zéro exécution, jamais** | Produit, Sécurité, Distribution, Architecture, Code, Tests |
| 5 | 01-03, 03-03, 05-06, 06-06 | high | CONFIRMED ×2, PARTIAL ×2 | La v0.3.0 n'existe pas en Go : 14 fournisseurs, régions, fallback chains, erase keys, profils 60-83 livrés **uniquement en PowerShell « legacy »** ; le « primaire » Go stagne à 0.2.1 (aggravation du #12 v0.2.1) | Produit, Architecture, Tests, UX/Docs |
| 6 | 01-05, 02-02, 02-05, 02-07, 02-08, 02-11, 04-11, 06-09 | high | CONFIRMED ×6, PARTIAL ×2 | Distribution morte (4 canaux sur 5) : repo privé sans release ni tag distant, `go install` structurellement impossible (module en sous-dossier), checksums brew/scoop/AUR en placeholder, mismatch de nommage d'archives (404 garanti), `install.sh` → HTTP 404, buckets/taps inexistants | Produit, Distribution, Code, UX/Docs |
| 7 | 01-04, 04-12, 06-07, 06-08, 01-01, 06-01 | high | CONFIRMED ×4, PARTIAL ×2 | Claims faux dans README/CHANGELOG : `models`/`search`/`compare` inexistants dans les DEUX implémentations, « cost logging » = simple log de lancement sans aucun prix, « cache 1h » = code mort, badges « 9.5/10 » et « 10/10 » auto-décernés | Produit, Code, UX/Docs |
| 8 | 01-07, 04-05, 06-04, 06-05, 03-02, 04-06, 05-04, 07-12 | high | CONFIRMED ×4, PARTIAL ×2, non c-v ×2 | ~31 % d'`internal/` = code mort vendu comme features : YAML/`.multiai.yaml`/héritage `extends`, hooks (dont l'échappement, mal ordonné, réintroduirait l'injection s'ils étaient branchés), onboarding, openrouter, logging — zéro appelant en production | Produit, Architecture, Code, Tests, UX/Docs, Sécurité |
| 9 | 07-01, 07-02, 03-09, 04-07 | high | CONFIRMED ×2, PARTIAL ×2 | Credential store : master key AES en clair à côté du ciphertext (v0.2.1 #5 persiste), stores « natifs » Windows/macOS/Linux = façades sur le même fichier, round-trip base64 cassé (Set encode, Get ne décode jamais), PBKDF2 jamais appelé | Sécurité, Architecture, Code |
| 10 | 01-08, 04-02, 03-08, 04-03 | high | CONFIRMED ×1, PARTIAL ×3 | Contrat de profil cassé côté Go : `%USERPROFILE%` jamais expansé (isolation des configs Claude silencieusement inopérante sur 6/17 profils) + whitelist env case-sensitive → processus enfant **sans PATH** sous Windows (vérifié empiriquement) | Produit, Code, Architecture |

**Sous le top 10** (high confirmés, mono-dimension) : installeur npm sans vérification checksum/signature (07-04), échecs de lancement en exit 0 (04-08), `defer close(sigCh)` avant `signal.Stop` → panic possible (04-09), ~60 % du code de production à 0 % de couverture dont tout le chemin critique (05-03), release pipeline cassé par construction (05-05), chaos de versions 0.2.1/0.3.0/0.5.0 avec injection ldflags sur une `const` = no-op (03-07, 02-06, 02-03).

---

## Forces confirmées (croisées entre rapports)

| Force | Confirmée par |
|---|---|
| Canal npm réel et fonctionnel : `multiai@0.3.0` publié, `bin/multiai.js` propre, 13 versions | 01, 02 |
| Fallback chains PS réelles et différenciantes (aucun concurrent lanceur ne le fait) | 01, 04, 06 |
| 14 fournisseurs + régions + 37 profils + erase keys réels côté PowerShell | 01, 04, 06 |
| Isolation env par liste blanche, fail-closed — l'angle sécurité unique du produit | 01, 03, 07 |
| Aucun secret réel commité ni dans l'historique git ; scanner `prepublishOnly` anti-fuite robuste | 07, 02, 01 |
| AES-256-GCM correctement implémenté (nonce aléatoire, tag) ; permissions 0600/0700 systématiques | 04, 07 |
| Hygiène de dépendances exemplaire : 1 seule dépendance (`yaml.v3`), graphe d'imports sans cycle | 03, 04, 07 |
| Quick wins UX v0.2.1 réellement livrés en Go : « 0. Retour », préfixes `[OK]/[X]`, NO_COLOR, exit code enfant propagé, forwarding signaux | 01, 03, 04, 06 |
| Tests existants authentiques : 0 échec, `go vet` propre, `pkg/dotenv` à 93.9 % mesuré | 05, 03 |
| Outillage release sérieux **sur le papier** : goreleaser + Cosign keyless + SBOM Syft + dependabot bien rédigés (jamais exécutés) | 02, 05 |

---

## Statut des 15 problèmes de l'audit v0.2.1

Consolidé depuis les sections « Statut des problèmes v0.2.1 » des 7 rapports.
Bilan : **4 corrigés · 2 partiels · 7 persistants · 2 aggravés**.

| # | Problème v0.2.1 | Statut consolidé | Détail (rapports sources) |
|---|---|---|---|
| 1 | Injection shell hooks via templates (CRITIQUE) | 🔴 **Persiste (latent, mal corrigé)** | `escapeShellArg` appliqué à la commande entière puis `os.ExpandEnv` **après** échappement : l'injection reste possible via valeurs d'env, et l'échappement PowerShell laisse passer `;` `\|` `&` `$()`. Neutralisé uniquement parce que les hooks sont du code mort jamais branché (03, 04, 05, 07) |
| 2 | Race TOCTOU `encryptedFileStore` (CRITIQUE) | 🟢 **Corrigé** (avec réserves) | `sync.Mutex` réel (03, 05) ; mais wrappers win/darwin instancient un store neuf par opération (mutex inopérant entre opérations), aucun test de concurrence, `-race` jamais exécuté (04, 05) |
| 3 | `AllowedCommands` map mutable (CRITIQUE) | 🟢 **Corrigé + testé** | Slice + accesseur, test positif/négatif (03, 04, 05) |
| 4 | Checksums placeholders brew/scoop/AUR (HAUTE) | 🔴 **Persiste** (identique) | `PLACEHOLDER_*_SHA256` / `REPLACE_WITH_ACTUAL_SHA256` toujours en place ; le goreleaser censé les remplir n'a jamais tourné (01, 02, 03, 05, 07) |
| 5 | Master key AES en clair (HAUTE) | 🔴 **Persiste** | `.masterkey` en clair dans le même dossier que le ciphertext ; PBKDF2 ajouté mais **code mort** (appelé uniquement par les tests) (03, 07) |
| 6 | Aucune signature Cosign (HAUTE) | 🔴 **Persiste de facto** | Config Cosign+SBOM présente mais pipeline inexécutable (workflows hors racine, repo privé, 0 tag distant) : aucun artefact signé n'a jamais existé ; l'installeur npm ne vérifie rien (01, 02, 03, 05, 07) |
| 7 | Exit code fils non propagé (HAUTE) | 🟡 **Partiel** | Code de l'enfant propagé ; mais toute erreur du routeur lui-même (profil introuvable, secret manquant) sort en **exit 0** (01, 03, 04, 05) |
| 8 | Pas de context / SIGINT orphelin (HAUTE) | 🟡 **Partiel** | Forwarding SIGINT/SIGTERM ajouté ; mais toujours zéro `context.Context`, no-op sous Windows, double-SIGINT sous Unix, et nouveau bug : `defer close(sigCh)` avant `signal.Stop` → panic possible (01, 03, 04, 05) |
| 9 | `updateEnvFile` non atomique (HAUTE) | 🔴 **AGGRAVÉ** | Temp+rename OK, mais la « correction » écrit la sentinelle credstore jamais relue : le fix a cassé le flux central du produit (top 1 ci-dessus) ; et le PS reste non atomique (01, 03, 04, 05) |
| 10 | Navigation sans retour en Go (CRITIQUE UX) | 🟢 **Corrigé** | « 0. Retour » aux deux niveaux + boucle outil→profil (01, 03, 06) |
| 11 | Couleurs sans texte (HAUTE) | 🟢 **Corrigé** (résidu mineur) | `[OK]/[!]/[X]/[i]` + NO_COLOR ; résidu : un « ⚠ » seul, typographie Unicode réintroduite (01, 03, 06) |
| 12 | Incohérence Go vs PowerShell (HAUTE) | 🔴 **AGGRAVÉ** | v0.3.0 100 % PS-only ; 5 vs 14 fournisseurs, 17 vs 37 profils, menus 3 vs 4 entrées, sémantique `%VAR%`/`FALLBACK` divergente, messages Go avec syntaxe PS (01, 02, 03, 04, 06) |
| 13 | Aucun wizard d'onboarding (HAUTE) | 🔴 **Persiste (faux-corrigé)** | `internal/onboarding/wizard.go` écrit et déclaré livré au CHANGELOG, mais **jamais appelé** ; marqueur first-run écrit jamais lu (01, 02, 03, 05, 06) |
| 14 | KDF SHA-256 sans itérations (MOYENNE) | 🔴 **Persiste (faux-corrigé)** | PBKDF2-HMAC-SHA256 10K implémenté mais jamais appelé sur le chemin réel : le chiffrement utilise la master key brute (07, 03) |
| 15 | `--allow-custom-command` bypass whitelist (MOYENNE) | 🔴 **Persiste** | Simple warning puis exécution, aucune validation (01, 03, 07) |

À noter : deux « forces » créditées par l'audit v0.2.1 sont **invalidées** par ce run — « CI/CD complète » (jamais exécutée, cf. top 4) et « 45+ tests » (32 fonctions `Test*` comptées).

---

## Sommaire

| Fichier | Contenu |
|---|---|
| [01-produit-fonctionnalites.md](01-produit-fonctionnalites.md) | Produit & fonctionnalités (Atlas) — 3.5/10 |
| [02-distribution-packaging.md](02-distribution-packaging.md) | Distribution & packaging (Atlas) — 2.5/10 |
| [03-architecture.md](03-architecture.md) | Architecture (Forge) — 3.5/10 |
| [04-qualite-code.md](04-qualite-code.md) | Qualité de code (Forge) — 4.5/10 |
| [05-tests-cicd.md](05-tests-cicd.md) | Tests & CI/CD (Sentinel) — 3.0/10 |
| [06-ux-dx-docs.md](06-ux-dx-docs.md) | UX / DX / Documentation (Sentinel) — 4.5/10 |
| [07-securite.md](07-securite.md) | Sécurité (Shadow) — 4.5/10 |
| [08-verification-adversariale.md](08-verification-adversariale.md) | Contre-vérification adversariale — méthode, verdicts, recalibrages |
| [09-roadmap-le-meilleur.md](09-roadmap-le-meilleur.md) | Roadmap « LE meilleur routeur multi-IA du marché » |

---

*Synthèse générée par Nexus (BMAD+) le 2026-07-04 — 99 agents : 7 auditeurs parallèles + contre-vérification adversariale (2 réfutateurs par critical, 1 par high) + orchestration déterministe.*
