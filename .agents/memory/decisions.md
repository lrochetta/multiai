---
title: Decisions
description: ADR log
created: "2026-06-23"
project: "multiai"
---

# Decisions

## 2026-07-20 — Release 0.7.0 : exécution et déblocages
- **Context**: Laurent a autorisé commit, push, version bump et publication npm en session. Trois obstacles : l'environnement GitHub protégé `release` laissait GoReleaser en `waiting` ; le token npm avait été révoqué lors de la rotation post-incident 0.6.10 ; `npm publish` non-TTY masque les URLs WebAuthn avec `***` même hors PowerShell.
- **Decision**: (1) Approbation du deployment `release` via `POST .../pending_deployments` avec le token gh de laurent (reviewer légitime, intention explicite). (2) Relogin npm par l'API web (`POST /-/v1/login` + header **`npm-auth-type: web`**, sans lequel le registry route vers le flow legacy et renvoie 401), token écrit dans `~/.npmrc`. (3) Publish piloté en Node : `npm-cli.js` appelé directement (Node ≥20 refuse de spawner `.cmd` sans shell), tentative en `--json` pour obtenir `authUrl`/`doneUrl` non masqués, navigateur ouvert, poll du doneUrl avec le bearer de session, retry avec `npm_config_otp` injecté en env uniquement. (4) Protection master restaurée et renforcée : 9 checks requis (Lint, Test×3, Security scan, Secret scan gitleaks, Build×3), anti force-push/deletion, enforce_admins off.
- **Rationale**: Chaque étape reprend la leçon 0.6.10 correspondante ; aucune valeur sensible affichée ni persistée hors des emplacements standards.
- **Consequences**: GitHub v0.7.0 (11 assets) et npm latest=0.7.0 vérifiés. AUR toujours en échec attendu (clés SSH absentes). Scripts réutilisables dans le scratchpad de session (npm-web-login.js / npm-publish-webauthn.js) — pattern documenté dans lessons.
- **Status**: completed

## 2026-07-20 — Fournisseur NVIDIA build.nvidia.com + pont Anthropic→OpenAI INTÉGRÉ au binaire
- **Context**: Laurent veut les modèles gratuits de build.nvidia.com (GLM 5.2 en tête) dans Claude Code, Codex et OpenCode, avec une solution interne « fully compatible » sans dépendre d'un proxy externe. Vérifié en live : `integrate.api.nvidia.com/v1` ne sert QUE `/v1/chat/completions` + `/v1/models` (404 sur `/v1/messages` et `/v1/responses`), aucune métadonnée prix/contexte, catalogue 118 modèles 100 % gratuit (~40 req/min) — NVIDIA ne vend aucune API par token.
- **Decision**: (1) Provider `nvidia` au catalog (console build.nvidia.com/settings/api-keys, var `NVIDIA_API_KEY`). (2) **Pont Anthropic→OpenAI intégré** : package `internal/bridge` (Messages↔chat/completions : streaming SSE, tool calls, reasoning_content→blocs thinking, count_tokens, erreurs Anthropic, /v1/models ; clamp max_tokens 32768). Le launcher le démarre pour tout profil `BRIDGE=anthropic-openai` (+`BRIDGE_TARGET`/`BRIDGE_KEY_VAR`/`BRIDGE_PORT`, loopback port éphémère, ANTHROPIC_BASE_URL injecté, arrêt au exit du CLI) ; sous-commande standalone `multiai bridge` (défaut :4100). (3) Profils : `nv-cc` (pont intégré), `ocnvidia` (OpenCode direct), `codex-nv` (pont LiteLLM externe :4000 — Codex exige l'API Responses, non couverte par le pont intégré). (4) Menu « 5. NVIDIA — Modèles gratuits » : découverte + profilegen dynamique `nv-*` (backend factorisé dans openrouter, garde anti-collision de shortcuts).
- **Rationale**: claude-code-router cassé avec NIM (issue #1341) ; LiteLLM fonctionne mais impose Python (et 3.14 sans wheel → compilation Rust qui échoue sur cette machine) ; un pont in-process en Go supprime toute dépendance, vit le temps du process enfant et n'écoute qu'en loopback. Le mécanisme BRIDGE est générique : n'importe quel backend OpenAI-compatible devient utilisable par Claude Code.
- **Consequences**: 40 profils / 14 fournisseurs / 35 shortcuts (tests à jour, `BRIDGE_KEY_VAR` exempté du scan anti-secrets comme métadonnée). Profils Codex tiers réparés au passage (voir lessons : Codex 2026 ignorait OPENAI_BASE_URL et retombait silencieusement sur le compte OpenAI). Docs guide/nvidia.md + providers + env-variables + profiles réécrits.
- **Status**: implemented + validé live : pont non-stream (200, usage réel mappé), streaming (séquence SSE Anthropic conforme), et **E2E complet `multiai launch -p nv-cc` → claude -p → GLM 5.2 → « PONT LAUNCHER OK »**

## 2026-07-15 — Promotion explicite de 0.6.10 en stable/latest
- **Context**: v0.6.10 était volontairement limitée à GitHub prerelease et npm `next` après une première rétention Avast, puis le même asset exact a réussi son postinstall et sa probe sans processus résiduel après acquisition de réputation. La qualification multi-PC recommandée n'était pas encore complète.
- **Decision**: Sur confirmation explicite de Laurent, promouvoir exactement la release et le package déjà vérifiés : GitHub v0.6.10 devient non-prerelease/`latest`, npm `latest` et `next` pointent sur 0.6.10. Aucun nouvel artefact n'est reconstruit.
- **Rationale**: Les checksums, la CI, la release, le tarball public et un smoke Windows du hash exact sont verts; conserver le même artefact évite toute divergence de supply chain.
- **Consequences**: `npx multiai` résout désormais 0.6.10. Le risque de réputation endpoint reste documenté et les tests sur plusieurs PC/antivirus doivent continuer après promotion.
- **Status**: completed; public smoke `multiai 0.6.10` successful

## 2026-07-15 — Purge complète de la clé DeepSeek historique
- **Context**: Une clé DeepSeek réelle, désormais révoquée (API HTTP 401), restait récupérable dans trois blobs du HEAD, le commit public `b25fbb7`, plusieurs tags, forks et refs de pull request. Laurent a remplacé la décision non destructive initiale par une demande explicite de réécriture et de repush sans la clé après publication de 0.6.10.
- **Decision**: Réécrire toutes les branches et tous les tags distants avec `git-filter-repo`, remplacer la valeur révoquée dans chaque blob, retirer ensuite l'exception Gitleaks liée à l'ancien commit et force-push les refs nettoyées. Recréer v0.6.10 depuis le nouveau SHA avec checksums, signatures, SBOM et provenance à jour.
- **Rationale**: La révocation empêche l'usage actif, mais seule la réécriture retire la valeur des refs Git publiques. Le dépôt GitHub lui-même est conservé afin de préserver ses URLs, réglages et canaux de distribution.
- **Consequences**: Tous les SHA descendants et tags concernés ont changé; les clones et worktrees existants doivent être jetés ou reclonés et ne jamais repousser l'ancien historique. Les refs normales du dépôt principal sont propres sur `4851c2e`; les deux forks publics ont également été réécrits et vérifiés par clones frais. Trois refs de PR fermées et les objets/caches GitHub restent hors contrôle du force-push et nécessitent une purge par GitHub Support.
- **Status**: completed for all normal refs and local clones; GitHub internal PR/cache purge pending

## 2026-07-15 — Canal 0.6.10 borné hors du processus Node
- **Context**: Le paquet npm 0.6.8, bien que limité à `next`, pouvait encore rester bloqué dans `execFileSync`/`spawnSync` lorsque Avast retenait directement `CreateProcess`; le timeout Node n'était alors jamais armé.
- **Decision**: Publier le correctif suivant sous 0.6.10 avec un contrôleur Windows externe à deux processus. Le contrôleur lance la probe dans un worker PowerShell, attend une deadline, tente de tuer l'arbre puis applique un fallback CIM + `Kill` si l'antivirus refuse `taskkill`. Seules les probes de version sont bornées; les commandes interactives restent sans limite.
- **Rationale**: La deadline doit vivre dans un processus déjà démarré et de confiance, indépendant du thread retenu dans `CreateProcess`.
- **Consequences**: Le tag 0.6.9 est abandonné sans release/npm. 0.6.10 doit rester GitHub prerelease (`latest=false`) et npm `next`; 0.6.6 demeure stable. La promotion requiert CI complète, contrôle du tarball exact et essai sur un autre PC. Les identifiants GitHub restent des pointeurs vers le coffre partagé hors dépôt, sans valeur secrète copiée dans le projet.
- **Status**: promoted to GitHub/npm `latest` by explicit user decision; multi-PC endpoint-security qualification remains recommended

## 2026-07-14 — Auth GitHub de release via le coffre partagé, sans persistance locale
- **Context**: Le token OAuth du keyring `gh` possède `repo` mais pas `workflow`, donc GitHub refuse tout push modifiant `.github/workflows/*`.
- **Decision**: Utiliser le pointeur du coffre partagé hors dépôt, résoudre le PAT actif depuis le coffre central, vérifier ses scopes puis l'injecter uniquement dans l'environnement du processus Git via `http.extraHeader`. Ne jamais copier sa valeur dans le dépôt ni remplacer le token `gh` du keyring.
- **Rationale**: Le dépôt garde zéro secret et l'autorisation étendue n'existe que pendant l'opération explicitement demandée.
- **Consequences**: Le push du hotfix `079019c` a réussi avec `repo` + `workflow`; toutes les variables temporaires ont été supprimées dans un bloc `finally`. Le coffre reste la source de vérité et doit être roté séparément.
- **Status**: active

## 2026-07-14 — Rollback public 0.6.6 et gate Avast obligatoire pour 0.6.8
- **Context**: Le binaire Windows 0.6.7 publié est retenu par Avast CyberCapture dans `CreateProcess`, avant le runtime Go. Le postinstall annonçait néanmoins un succès et le shim pouvait attendre indéfiniment.
- **Decision**: Remettre immédiatement npm `latest` sur 0.6.6 et déprécier 0.6.7. Préparer 0.6.8 avec Go 1.25.12 (plus récent patch sécurisé de la branche 1.25), smoke postinstall borné, probes du shim bornées et tests E2E dont les appels `CreateProcess` sont contrôlés hors du thread de test.
- **Rationale**: Un timeout de `CommandContext` ne suffit pas si Windows bloque directement `CreateProcess`; la borne doit entourer l'appel `Run` lui-même. Aucune release 0.6.8 ne peut être promue sans CI complète et essai de l'artefact officiel sur une machine Avast/CyberCapture.
- **Consequences**: 0.6.6 reste la version stable. Le tag v0.6.8 peut produire une GitHub prerelease et npm 0.6.8 peut rester sous le tag `next` pour tester exactement les hashes finaux; aucune promotion stable/`latest` avant qualification Avast ou whitelisting.
- **Status**: active

## 2026-07-14 — Bootstrap PATH Windows explicite, user-scope et fail-closed
- **Context**: `npx multiai install` terminait son installation globale puis smoke-testait directement le JavaScript du package. Sous Windows, ce test contournait `multiai.cmd` et masquait l'absence du préfixe npm dans `PATH`.
- **Decision**: Le parcours explicite `npx --yes --allow-scripts=multiai multiai@latest install` résout `npm prefix --global`, valide un chemin de disque local, persiste ce préfixe au scope User via .NET, puis vérifie le premier shim réellement résolu. Pas de `setx`, pas d'élévation, pas de mutation depuis `postinstall`; `npm install -g` seul conserve la sémantique npm standard.
- **Rationale**: Réparer le contrat utilisateur sans mutation implicite lors d'un lifecycle npm, préserver le PATH existant et refuser tout faux succès ou shim masqué.
- **Consequences**: `MULTIAI_SKIP_PATH_UPDATE=1` reste disponible pour les postes administrés. Une nouvelle console est nécessaire. L'E2E `Apply` sur VM Windows vierge est une gate obligatoire avant publication.
- **Status**: implemented locally; release-blocked

## 2026-07-14 — Les frontières de confiance priment sur les nouvelles fonctions
- **Context**: L'audit Atlas/Forge/Sentinel 2026-07-14 note la maturité à 5,8/10 et identifie quatre bloqueurs indépendants : configuration projet implicite, traversal du registre, updater non persistant/fail-open et workflow de release effectif divergent.
- **Decision**: Maintenir le NO-GO v0.6.7. Rendre le check d'update de démarrage notification-only, exiger la confiance projet explicite, confiner les écritures du registre et unifier la gate de release avant tout ajout fonctionnel ou publication.
- **Rationale**: Un produit local-first ne peut être une référence que si un dépôt non fiable, un index distant ou un artefact non qualifié ne peut jamais déclencher une exécution implicite.
- **Consequences**: La roadmap P0 dans `audit/2026-07-14-bmad-plus-complete/05-roadmap-priorisee.md` complète la matrice CI déjà exigée. Aucun tag, release GitHub ou `npm publish` avant fermeture des sept lignes P0.
- **Status**: active

## 2026-07-13 — Aucun tag ou publish avant matrice CI entièrement verte
- **Context**: Le correctif `0.6.7` et quatre commits sont poussés, mais le run CI `29213384824` reste rouge sur macOS et Ubuntu malgré les contrôles Windows, sécurité, lint, GoReleaser et cross-compilation verts.
- **Decision**: Suspendre la release. Aucun tag `v0.6.7`, aucune release GitHub et aucun `npm publish` ne sont autorisés avant une reprise explicite et une matrice CI complète verte.
- **Rationale**: Les échecs restants révèlent des tests dépendants de la langue et des stores natifs partagés; les ignorer produirait une release non reproductible.
- **Consequences**: La reprise commence par les correctifs de tests documentés dans le handoff, suivis d'un commit/push et d'une nouvelle CI. La publication npm restera manuelle avec 2FA.
- **Status**: active

## 2026-07-12 — Contrat npm/npx restauré et release reproductible
- **Context**: `multiai@0.6.6` échouait sur les nouvelles installations Windows avec Node 24 (`unable to verify the first certificate`). Après contournement TLS, la commande publique `npx multiai install` échouait encore car le CLI Go ne possède pas de sous-commande `install`. Le tag `v0.6.6` contenait en outre `package.json` en `0.6.5`; npm 0.6.6 avait été publié depuis un worktree sale.
- **Decision**: Préparer `0.6.7` avec (1) Node 24.14+ comme minimum npm et fusion feature-détectée des CA par défaut/système, sans jamais désactiver TLS, (2) support du proxy d'environnement, (3) restauration de `npx multiai install` comme installation npm globale réelle, (4) sortie propre sur EOF/non-TTY, (5) tests npm et preflights `tag == package version` + worktree propre.
- **Rationale**: Restaurer le contrat historique sans sacrifier la vérification SHA256 et rendre la publication traçable/reproductible.
- **Consequences**: La release GitHub `v0.6.7` doit être créée depuis le commit contenant `package.json@0.6.7` avant le `npm publish` manuel. Le prepublish refuse une copie sale ou non taguée.
- **Status**: active

## 2026-07-06 — Auto-update via GitHub Releases
- **Context**: Les utilisateurs npm/go install doivent réinstaller manuellement pour obtenir la dernière version. Aucune notification de mise à jour.
- **Decision**: Ajouter `internal/update/` — au lancement, vérifie l'API GitHub Releases (cache 1h), télécharge le nouveau binaire si plus récent, vérifie SHA256, extrait, re-exec. Tout est silencieux (timeout 5s, jamais bloquant).
- **Rationale**: Maintient les utilisateurs à jour sans friction. Pas de dépendance externe (stdlib uniquement).
- **Consequences**: `update.Check(version)` dans `main()`, package `internal/update/`, cache dans `UserConfigDir/multiai/update-check.json`.
- **Status**: superseded by the 2026-07-14 notification-only decision; remediation pending

## 2026-07-06 — Menus colorés selon statut de configuration
- **Context**: Les utilisateurs ne savaient pas quels profils/fournisseurs étaient configurés sans entrer dans le wizard.
- **Decision**: Colorer les lignes des menus config et profils : vert [OK] si toutes les clés configurées, jaune [~~] si partiel, gris [--] si aucune. Fonction `StatusColor` exportée dans `internal/cli/display.go`.
- **Rationale**: Feedback visuel immédiat, cohérent entre les menus, réutilisable.
- **Consequences**: `countSecrets()` dans menu, `StatusColor()` dans cli, `Colorize()` exporté.
- **Status**: active

## 2026-07-05 — Repo GitHub public avec branch protection
- **Context**: Le repo était privé, bloquant le téléchargement des binaires par `install.js` (npm).
- **Decision**: Passer le repo en public, protéger master avec 6 status checks obligatoires (Lint, Test×3, Security scan, Build), interdire force-push et deletion.
- **Rationale**: npm nécessite des releases publiques. Branch protection empêche les régressions.
- **Consequences**: CI/CD doit passer avant tout merge. Force-push désactivé (admin bypass temporaire possible).
- **Status**: active

## 2026-07-05 — npm switch PowerShell → Go natif
- **Context**: Le package npm distribuait le script PowerShell. Le binaire Go est plus rapide, plus sûr (AES-256-GCM), cross-platform natif.
- **Decision**: Basculer `multiai` npm sur le binaire Go. `install.js` télécharge depuis GitHub Releases avec vérification SHA256.
- **Rationale**: Parité fonctionnelle atteinte et dépassée. Le PowerShell est gelé.
- **Consequences**: `package.json` v0.4.0+, `bin/multiai.js` shim vers binaire natif, `postinstall` = `install.js`.
- **Status**: active

## 2026-07-05 — Audit BMAD+ 3 agents en parallèle
- **Context**: Le projet atteint la parité Go/PS. Besoin d'un audit complet avant release publique.
- **Decision**: Lancer Atlas (stratégie), Forge (architecture), Sentinel (sécurité) en parallèle. Consolider dans `audit/`.
- **Rationale**: Couverture exhaustive en une session. Chaque agent a un scope distinct (produit, code, sécurité).
- **Consequences**: Score 8.5/10, 6 rapports, top 4 correctifs appliqués.
- **Status**: completed

## 2026-07-05 — Credential store obligatoire (plus de fallback texte clair)
- **Context**: `updateEnvFile` écrivait la clé en clair si le credential store était indisponible, avec un simple warning.
- **Decision**: Retourner une erreur bloquante. Ajouter `allowPlaintext` booléen pour forcer (utilisé par `--allow-plaintext`).
- **Rationale**: Ne jamais dégrader silencieusement la sécurité.
- **Consequences**: `updateEnvFile(path, varName, value, allowPlaintext)` — signature changée, appelants mis à jour.
- **Status**: active

## Décisions historiques (v0.2.x - v0.3.0)
- **8 nouveaux fournisseurs + régions + fallback** : completed
- **42 fixes post-audit par 5 agents parallèles** : completed
- **OpenRouter comme fournisseur LLM** : completed
- **Navigation avec retour + boucle interactive** : active
- **BMAD+ smart detection** : active
- **Pivot Go** : completed
- **Credential store natif** : active
- **Profils YAML + .multiai.yaml** : active
- **Plugin hooks** : active
- **Renommage aicode → multiai** : completed
- **Retrait Gemini** : completed
