# Tester un bundle Windows autonome

Ce bundle permet de transférer et tester `multiai` sur un autre PC Windows
64 bits sans publier de package npm ou de release GitHub. Le PC cible n'a
besoin ni de Go ni de Node.js.

## Construire le bundle

Depuis `multiai-go`, avec Go installé :

```powershell
.\scripts\build-test-bundle.ps1
```

Le script compile `multiai` 0.6.7 pour `windows/amd64`, désactive CGO,
injecte la version, exécute un smoke test hors ligne borné à 15 secondes,
puis produit :

```text
dist/test-bundle/multiai_0.6.7_windows_amd64.zip
dist/test-bundle/checksums.txt
```

La compilation est également bornée à 60 secondes. En cas de timeout ou
d'échec d'une vérification, le script s'arrête sans produire de ZIP validé.

Copiez ces deux fichiers sur le PC cible.

## Vérifier et tester sur le PC cible

Placez les deux fichiers dans le même dossier, puis exécutez PowerShell :

```powershell
$expected = (Get-Content .\checksums.txt |
  Where-Object { $_ -match 'multiai_0\.6\.7_windows_amd64\.zip$' } |
  ForEach-Object { ($_ -split '\s+')[0] })
$actual = (Get-FileHash .\multiai_0.6.7_windows_amd64.zip -Algorithm SHA256).Hash
if (-not $expected -or $actual -ne $expected) { throw 'SHA256 invalide' }

Expand-Archive .\multiai_0.6.7_windows_amd64.zip -DestinationPath .\multiai-test
$env:MULTIAI_SKIP_UPDATE = '1'
.\multiai-test\multiai.exe --version
.\multiai-test\multiai.exe help
```

Le test utilise directement l'exécutable extrait et ne modifie pas le `PATH`
du PC. Conservez `MULTIAI_SKIP_UPDATE=1` pendant les essais hors ligne.
