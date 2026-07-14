# Mirror the authoritative root workflows next to the Go module.
#
# WHY: GitHub only executes workflows located in <repo root>/.github/workflows/
# and only reads <repo root>/.github/dependabot.yml. The root is therefore the
# single source of truth. A checked mirror under multiai-go/.github/ keeps the
# module self-describing, and CI rejects any drift.
#
# Usage (from anywhere):
#   powershell -NoProfile -File multiai-go/scripts/sync-workflows.ps1
#   powershell -NoProfile -File multiai-go/scripts/sync-workflows.ps1 -Check
#
# -Check compares source and mirror without writing (CI-friendly: exits 1
# when the module copies are stale).

[CmdletBinding()]
param(
    [switch]$Check
)

$ErrorActionPreference = 'Stop'

$moduleDir = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$repoRoot = Split-Path -Parent $moduleDir
$sourceDir = Join-Path $repoRoot '.github'
$destDir = Join-Path $moduleDir '.github'

if (-not (Test-Path $sourceDir)) {
    throw "Source directory not found: $sourceDir"
}

# Only executable release-engineering configuration is mirrored. Repository
# templates remain root-only and cannot accidentally become a second source.
$relativePaths = @(
    'dependabot.yml',
    'workflows/ci.yml',
    'workflows/release.yml'
)
$sources = foreach ($relative in $relativePaths) {
    $source = Join-Path $sourceDir $relative
    if (-not (Test-Path -LiteralPath $source -PathType Leaf)) {
        throw "Authoritative file not found: $source"
    }
    Get-Item -LiteralPath $source
}

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
    Write-Host 'Module workflow mirrors are up to date.'
} else {
    Write-Host "Mirrored root release configuration to $destDir. Review with 'git status'."
}
