# Audit Adversarial Codex — multiai v0.4.3

**Date :** 2026-07-09
**Agent :** Codex CLI (GPT-5) via plugin `codex@openai-codex`
**Verdict :** NEEDS-ATTENTION — NO-SHIP

---

## Findings critiques

### 🔴 CRITICAL — Auto-update exécute des artefacts non authentifiés
**Fichier :** `multiai-go/internal/update/update.go:245-282`

L'archive et `checksums.txt` viennent des mêmes assets GitHub. Seule la cohérence SHA-256 est vérifiée. Le client ne récupère jamais `checksums.txt.sig`/`.pem`. Un attaquant qui contrôle `MULTIAI_GITHUB_API_URL` peut servir archive + checksum cohérents → RCE.

**Recommandation :** Vérifier la signature Cosign de `checksums.txt` AVANT le SHA256, refuser les hôtes non approuvés en production.

### 🔴 HIGH — Tout tag `v*` publie sans gate CI
**Fichier :** `.github/workflows/release.yml:20-68`

Le workflow release ne lance ni tests, ni lint, ni govulncheck, ne vérifie pas l'ascendance depuis master. Une release cassée serait signée et distribuée.

**Recommandation :** Bloquer la release sur CI verte (tests multi-OS, race, scans, couverture).

---

## Findings hautes

### 🟠 HIGH — L'update n'est pas persistée
**Fichier :** `multiai-go/internal/update/update.go:284-323`

Le nouveau binaire reste dans un répertoire temporaire, l'ancien n'est jamais remplacé. Le processus parent exit 0 sans attendre l'enfant → perte de l'exit code.

**Recommandation :** Installation atomique persistante avec backup/rollback + `exec`.

### 🟠 HIGH — Profil homonyme = vol de secret
**Fichier :** `multiai-go/internal/secret/secret.go:33-38`

`ServiceForProfile` ne namespace que par le basename du fichier. Un profil homonyme dans une racine custom récupère le secret légitime.

**Recommandation :** Dériver l'identité du secret d'une racine canonique approuvée.

### 🟠 HIGH — `--store` natif silencieusement ignoré
**Fichier :** `multiai-go/cmd/multiai/main.go:258-279`

`config --store keychain|wincred|secret-service` est ignoré, ouvrant le wizard normal. Tous les backends délèguent au fichier AES-GCM avec clé maître à côté des ciphertexts.

**Recommandation :** Implémenter ou rejeter explicitement avec exit ≠ 0.

### 🟠 HIGH — Mutations store/sentinelle = perte de credentials
**Fichier :** `multiai-go/internal/secret/secret.go:219-238`

Race condition : deux processus peuvent s'écraser. Le wizard efface le secret avant de remplacer la sentinelle → crash = perte irrécupérable.

**Recommandation :** Verrou inter-processus + transaction journalisée store + profil avec rollback.

### 🟠 HIGH — Nouvelles installations = pas de nouveaux profils après upgrade
**Fichier :** `multiai-go/cmd/multiai/main.go:76-88`

`ensureProfiles` retourne dès qu'un seul `.env` existe. Après upgrade, les nouveaux profils embarqués ne sont jamais extraits.

**Recommandation :** Migration versionnée avec manifeste et préservation des modifications.

### 🟠 HIGH — Profil OpenRouter dynamique non configurable
**Fichier :** `multiai-go/internal/openrouter/menu.go:181-199`

Le wizard `config --provider openrouter` ne parcourt que les 6 shortcuts statiques. Les profils `or-*` générés dynamiquement sont ignorés.

**Recommandation :** Associer explicitement le provider aux profils générés.

### 🟠 HIGH — Profils YAML, config projet et hooks absents du chemin de production
**Fichier :** `multiai-go/cmd/multiai/main.go:231-259`

`LoadDir` au lieu de `LoadAllProfiles`. Les hooks `before_launch`/`after_launch` ne s'exécutent jamais. Fonctionnalités documentées mais non câblées.

**Recommandation :** Unifier list/config/launch autour de `LoadAllProfiles` + transmettre les hooks.

---

## Findings moyennes

### 🟡 MEDIUM — CI smoke job utilise un répertoire inexistant
**Fichier :** `.github/workflows/ci.yml:148-153`

`cd multiai-go` dans un job où le `working-directory` est déjà `multiai-go` → `multiai-go/multiai-go` n'existe pas. Le test ne prouve rien.

### 🟡 MEDIUM — Rapports d'audit ne décrivent pas l'état livrable actuel
`audit/00-synthese.md` audite v0.4.0-dev sans SHA, annonce 16/16 tests et 0 défaut critique. La réalité v0.4.3 est différente. `ROADMAP.md` indique encore v0.2 en cours. `SECURITY.md` promet un store natif et un SBOM inexistants.

---

## Next steps (Codex)

1. **Geler la publication** et désactiver l'auto-update par défaut jusqu'à correction de l'authenticité
2. **Corriger l'identité, la transactionnalité et le backend** du credential store avec migration
3. **Unifier le pipeline de profils** et ajouter des tests E2E (upgrade, OpenRouter dynamique, YAML, hooks)
4. **Réparer la CI** puis exécuter `go test -race ./...`, `go vet`, `govulncheck` avec seuils
5. **Rebaseliner** audit/, ROADMAP.md et SECURITY.md sur le SHA livré
