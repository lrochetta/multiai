# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.2.x   | :white_check_mark: |
| 0.1.x   | :x: (PowerShell)   |

## Reporting a Vulnerability

**Ne PAS ouvrir d'issue publique pour les vulnerabilites de securite.**

Signaler par email a : **laurent@rochetta.fr**

Delai de reponse : 48 heures ouvre.es.

Processus :
1. Vous signalez la vulnerabilite par email
2. Nous accusons reception dans les 48h
3. Nous investigons et proposons un correctif
4. Nous publions un advisory de securite + correctif
5. Credit public (si souhaite)

## Security Best Practices for Users

1. **Ne jamais commiter vos fichiers .env** — ils sont dans .gitignore par defaut
2. **Utiliser le credential store natif** (v0.3.0+) plutot que les fichiers .env
3. **Verifier les signatures** avec Cosign : `cosign verify-blob --signature multiai.sig multiai`
4. **Mettre a jour regulierement** : `multiai update`

## Supply Chain

- Binaires signes avec Cosign (Sigstore)
- SBOM genere a chaque release (Syft) — au format CycloneDX, disponible dans les assets de chaque release GitHub
- Dependances scannees : gosec (SAST), govulncheck (vulnerabilites)
- Builds reproductibles via goreleaser
