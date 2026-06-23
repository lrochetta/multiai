<#
.SYNOPSIS
  Installe AI Code CLI Router et l'ajoute au PATH utilisateur.
  Windows : C:\AI\multiai + PATH Registry.
  macOS / Linux : ~/.local/share/multiai + shell profile.

.DESCRIPTION
  - Copie le projet vers le dossier d'installation.
  - Preserve les .env existants qui contiennent deja tes cles.
  - Les nouvelles versions de .env sont copiees en .new si le fichier existe deja.
  - Ajoute le dossier d'installation au PATH utilisateur.
  - macOS / Linux : necessite pwsh (PowerShell Core) installe.

.EXAMPLES
  # Windows
  .\install.ps1
  .\install.ps1 -InstallDir 'D:\tools\multiai'

  # macOS / Linux
  pwsh install.ps1
  pwsh install.ps1 -InstallDir '/opt/multiai'

.NOTES
  Author  : Laurent Rochetta
  Links   : https://follow.ovh/bio/laurent
  Website : https://rochetta.fr
#>

[CmdletBinding()]
param(
    [string]$InstallDir = ''
)

$ErrorActionPreference = 'Stop'
$SourceDir = Split-Path -Parent $MyInvocation.MyCommand.Path

# ── Detection plateforme ──────────────────────────────────────────────────────
# $IsWindows / $IsMacOS / $IsLinux sont des built-ins PS 6+.
# Sur Windows PowerShell 5.1, seul $IsWindows peut manquer -> on le deduit.
$onWindows = if (Test-Path variable:IsWindows) { $IsWindows } else { $true }
$onMacOS   = if (Test-Path variable:IsMacOS)   { $IsMacOS }   else { $false }

if ([string]::IsNullOrWhiteSpace($InstallDir)) {
    $InstallDir = if ($onWindows) {
        'C:\AI\multiai'
    } else {
        Join-Path $HOME '.local' 'share' 'multiai'
    }
}

Write-Host "Installation dans : $InstallDir" -ForegroundColor Cyan
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

# ── Copie des fichiers ────────────────────────────────────────────────────────
$files = Get-ChildItem -LiteralPath $SourceDir -Recurse -File
foreach ($file in $files) {
    $relative = $file.FullName.Substring($SourceDir.Length).TrimStart('\', '/')
    if ($relative -like '*.zip') { continue }

    $dest    = Join-Path $InstallDir $relative
    $destDir = Split-Path -Parent $dest
    New-Item -ItemType Directory -Force -Path $destDir | Out-Null

    $isProfile = ($relative -like ('configs' + [IO.Path]::DirectorySeparatorChar + 'profiles' + [IO.Path]::DirectorySeparatorChar + '*.env')) -or
                 ($relative -like 'configs\profiles\*.env') -or
                 ($relative -like 'configs/profiles/*.env')

    if ($isProfile -and (Test-Path -LiteralPath $dest)) {
        Copy-Item -LiteralPath $file.FullName -Destination ($dest + '.new') -Force
        Write-Host "Profil existant preserve : $relative ; nouvelle version copiee en .new" -ForegroundColor Yellow
    } else {
        Copy-Item -LiteralPath $file.FullName -Destination $dest -Force
    }
}

# ── Nettoyage anciens noms (aicode -> multiai) ───────────────────────────────
foreach ($old in @('aicode.cmd', 'aicode.sh')) {
    $oldPath = Join-Path $InstallDir $old
    if (Test-Path -LiteralPath $oldPath) {
        Remove-Item -LiteralPath $oldPath -Force
        Write-Host "Ancien fichier supprime : $old" -ForegroundColor DarkGray
    }
}

# ── Permissions Unix ─────────────────────────────────────────────────────────
if (-not $onWindows) {
    $shFiles = Get-ChildItem -LiteralPath $InstallDir -Filter '*.sh' -File -ErrorAction SilentlyContinue
    foreach ($sh in $shFiles) {
        & chmod +x $sh.FullName
    }
    # Normalise les fins de ligne CRLF -> LF dans les .sh (securite si crees sur Windows)
    foreach ($sh in $shFiles) {
        $content = [System.IO.File]::ReadAllText($sh.FullName) -replace "`r`n", "`n" -replace "`r", "`n"
        [System.IO.File]::WriteAllText($sh.FullName, $content, [System.Text.Encoding]::UTF8)
    }
    Write-Host "Permissions .sh appliquees." -ForegroundColor Green
}

# ── Gestion PATH ──────────────────────────────────────────────────────────────
if ($onWindows) {
    $currentPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    $paths = @()
    if (-not [string]::IsNullOrWhiteSpace($currentPath)) {
        $paths = $currentPath -split ';' | Where-Object { -not [string]::IsNullOrWhiteSpace($_) }
    }
    $alreadyInPath = $false
    foreach ($p in $paths) {
        if ($p.TrimEnd('\', '/') -ieq $InstallDir.TrimEnd('\', '/')) {
            $alreadyInPath = $true
            break
        }
    }
    if (-not $alreadyInPath) {
        $newPath = if ([string]::IsNullOrWhiteSpace($currentPath)) { $InstallDir } else { "$currentPath;$InstallDir" }
        [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
        $env:Path += ";$InstallDir"
        Write-Host "Ajoute au PATH utilisateur (Windows Registry) : $InstallDir" -ForegroundColor Green
    } else {
        Write-Host 'Deja present dans le PATH utilisateur.' -ForegroundColor Green
    }
} else {
    # macOS / Linux : modification du shell profile
    $shellEnv     = [Environment]::GetEnvironmentVariable('SHELL', 'Process')
    $shellProfile = Join-Path $HOME '.profile'
    if ($shellEnv -like '*zsh*') {
        $shellProfile = Join-Path $HOME '.zshrc'
    } elseif ($shellEnv -like '*bash*') {
        if ($onMacOS -and (Test-Path (Join-Path $HOME '.bash_profile'))) {
            $shellProfile = Join-Path $HOME '.bash_profile'
        } else {
            $shellProfile = Join-Path $HOME '.bashrc'
        }
    }

    $exportLine     = "export PATH=`"`$PATH:$InstallDir`""
    $profileContent = if (Test-Path $shellProfile) { Get-Content $shellProfile -Raw -ErrorAction SilentlyContinue } else { '' }

    if ($null -eq $profileContent) { $profileContent = '' }

    if (-not $profileContent.Contains($InstallDir)) {
        Add-Content -Path $shellProfile -Value "`n# AI Code CLI Router`n$exportLine"
        Write-Host "Ajoute au PATH dans : $shellProfile" -ForegroundColor Green
    } else {
        Write-Host 'Deja present dans le PATH.' -ForegroundColor Green
    }
}

# ── Messages post-installation ────────────────────────────────────────────────
Write-Host ''
Write-Host 'Installation terminee.' -ForegroundColor Green

if ($onWindows) {
    Write-Host 'Ferme et rouvre PowerShell, puis utilise :' -ForegroundColor Cyan
} else {
    $shellEnv2 = [Environment]::GetEnvironmentVariable('SHELL', 'Process')
    $reloadCmd = if ($shellEnv2 -like '*zsh*') { 'source ~/.zshrc' } elseif ($onMacOS) { 'source ~/.bash_profile' } else { 'source ~/.bashrc' }
    Write-Host ("Recharge ton shell avec : $reloadCmd") -ForegroundColor Cyan
    Write-Host 'Puis utilise :' -ForegroundColor Cyan
}

Write-Host '  multiai        # menu principal (lancer / config cles / BMAD+)'
Write-Host '  config        # configurer les cles API par fournisseur'
Write-Host '  bmad          # installer BMAD+ dans un projet'

if (-not $onWindows) {
    Write-Host ''
    Write-Host '  Sur macOS/Linux, utilise multiai.sh / config.sh / bmad.sh.' -ForegroundColor DarkGray
}

Write-Host ''
Write-Host '  co            # Claude Code officiel'
Write-Host '  cg            # Claude Code via Z.ai GLM-5.2'
Write-Host '  ds            # Claude Code via DeepSeek V4 Pro 1M'
Write-Host '  codex55       # Codex GPT-5.5'
Write-Host '  oc            # OpenCode menu / default'
Write-Host '  ocdeepseek    # OpenCode DeepSeek'
Write-Host ''
Write-Host 'Configure tes cles (premiere etape recommandee) :' -ForegroundColor Cyan
Write-Host '  config'
Write-Host ''
Write-Host 'Ou manuellement dans :' -ForegroundColor DarkGray
Write-Host "  $InstallDir$(  [IO.Path]::DirectorySeparatorChar)configs$(  [IO.Path]::DirectorySeparatorChar)profiles$(  [IO.Path]::DirectorySeparatorChar)"
