# Deploy GitHub Actions workflows + dependabot config to the repo root.
#
# WHY: GitHub only executes workflows located in <repo root>/.github/workflows/
# and only reads <repo root>/.github/dependabot.yml. This monorepo maintains
# them next to the Go module (multiai-go/.github/) so that release engineering
# stays versioned with the code it releases. Run this script after any change
# under multiai-go/.github/ and commit the synced copies at the repo root.
#
# Usage (from anywhere):
#   powershell -NoProfile -File multiai-go/scripts/sync-workflows.ps1
#   powershell -NoProfile -File multiai-go/scripts/sync-workflows.ps1 -Check
#
# -Check compares source and destination without writing (CI-friendly:
# exits 1 when the repo root copies are stale).

[CmdletBinding()]
param(
    [switch]$Check
)

$ErrorActionPreference = 'Stop'

$moduleDir = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$repoRoot = Split-Path -Parent $moduleDir
$sourceDir = Join-Path $moduleDir '.github'
$destDir = Join-Path $repoRoot '.github'

if (-not (Test-Path $sourceDir)) {
    throw "Source directory not found: $sourceDir"
}

# Everything under multiai-go/.github/ is deployed with the same relative path.
$sources = Get-ChildItem -Path $sourceDir -Recurse -File

$stale = @()
foreach ($src in $sources) {
    $relative = $src.FullName.Substring($sourceDir.Length).TrimStart('\', '/')
    $dest = Join-Path $destDir $relative

    $needsCopy = $true
    if (Test-Path $dest) {
        $srcHash = (Get-FileHash -Path $src.FullName -Algorithm SHA256).Hash
        $destHash = (Get-FileHash -Path $dest -Algorithm SHA256).Hash
        $needsCopy = ($srcHash -ne $destHash)
    }

    if (-not $needsCopy) {
        Write-Host "ok      .github/$relative"
        continue
    }

    if ($Check) {
        $stale += $relative
        Write-Host "STALE   .github/$relative"
        continue
    }

    $destParent = Split-Path -Parent $dest
    if (-not (Test-Path $destParent)) {
        New-Item -ItemType Directory -Force -Path $destParent | Out-Null
    }
    Copy-Item -Path $src.FullName -Destination $dest -Force
    Write-Host "synced  .github/$relative"
}

if ($Check -and $stale.Count -gt 0) {
    Write-Host ''
    Write-Host "$($stale.Count) file(s) out of sync. Run: multiai-go/scripts/sync-workflows.ps1"
    exit 1
}

Write-Host ''
if ($Check) {
    Write-Host 'Repo root .github/ is up to date.'
} else {
    Write-Host "Deployed to $destDir. Review with 'git status' and commit the changes."
}
