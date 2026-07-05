---
title: Decisions
description: ADR log
created: "2026-06-23"
project: "multiai"
---

# Decisions

## 2026-07-06 — Auto-update via GitHub Releases
- **Context**: Les utilisateurs npm/go install doivent réinstaller manuellement pour obtenir la dernière version. Aucune notification de mise à jour.
- **Decision**: Ajouter `internal/update/` — au lancement, vérifie l'API GitHub Releases (cache 1h), télécharge le nouveau binaire si plus récent, vérifie SHA256, extrait, re-exec. Tout est silencieux (timeout 5s, jamais bloquant).
- **Rationale**: Maintient les utilisateurs à jour sans friction. Pas de dépendance externe (stdlib uniquement).
- **Consequences**: `update.Check(version)` dans `main()`, package `internal/update/`, cache dans `UserConfigDir/multiai/update-check.json`.
- **Status**: active

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
