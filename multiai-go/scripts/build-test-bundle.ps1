[CmdletBinding()]
param(
    [ValidatePattern('^\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?$')]
    [string]$Version = '0.6.7'
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$projectDir = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$outputDir = Join-Path $projectDir 'dist\test-bundle'
$binaryName = 'multiai.exe'
$archiveName = "multiai_${Version}_windows_amd64.zip"
$binaryPath = Join-Path $outputDir $binaryName
$archivePath = Join-Path $outputDir $archiveName
$checksumsPath = Join-Path $outputDir 'checksums.txt'

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw 'Go is required to build the Windows test bundle.'
}

New-Item -ItemType Directory -Force -Path $outputDir | Out-Null
foreach ($path in @($binaryPath, $archivePath, $checksumsPath)) {
    if (Test-Path -LiteralPath $path) {
        Remove-Item -LiteralPath $path -Force
    }
}

$previousCgo = $env:CGO_ENABLED
$previousGoos = $env:GOOS
$previousGoarch = $env:GOARCH
$previousSkipUpdate = $env:MULTIAI_SKIP_UPDATE

Push-Location $projectDir
try {
    $env:CGO_ENABLED = '0'
    $env:GOOS = 'windows'
    $env:GOARCH = 'amd64'

    $buildInfo = [Diagnostics.ProcessStartInfo]::new()
    $buildInfo.FileName = (Get-Command go).Source
    $quotedBinaryPath = $binaryPath.Replace('"', '\"')
    $buildInfo.Arguments = "build -trimpath -ldflags `"-s -w -X main.version=$Version`" -o `"$quotedBinaryPath`" ./cmd/multiai"
    $buildInfo.UseShellExecute = $false
    $buildInfo.CreateNoWindow = $true

    $buildProcess = [Diagnostics.Process]::new()
    $buildProcess.StartInfo = $buildInfo
    try {
        if (-not $buildProcess.Start()) {
            throw 'go build could not be started.'
        }
        if (-not $buildProcess.WaitForExit(60000)) {
            $buildProcess.Kill()
            throw 'go build timed out after 60 seconds.'
        }
        if ($buildProcess.ExitCode -ne 0) {
            throw "go build failed with exit code $($buildProcess.ExitCode)."
        }
    }
    finally {
        $buildProcess.Dispose()
    }
    if (-not (Test-Path -LiteralPath $binaryPath -PathType Leaf)) {
        throw "go build did not create $binaryPath."
    }

    $env:MULTIAI_SKIP_UPDATE = '1'
    $startInfo = [Diagnostics.ProcessStartInfo]::new()
    $startInfo.FileName = $binaryPath
    $startInfo.Arguments = '--version'
    $startInfo.UseShellExecute = $false
    $startInfo.CreateNoWindow = $true
    $startInfo.RedirectStandardOutput = $true
    $startInfo.RedirectStandardError = $true

    $process = [Diagnostics.Process]::new()
    $process.StartInfo = $startInfo
    try {
        if (-not $process.Start()) {
            throw 'The bundled binary could not be started.'
        }
        $stdoutTask = $process.StandardOutput.ReadToEndAsync()
        $stderrTask = $process.StandardError.ReadToEndAsync()
        if (-not $process.WaitForExit(15000)) {
            $process.Kill()
            throw 'The bundled binary timed out during its --version smoke test.'
        }
        $versionOutput = ($stdoutTask.GetAwaiter().GetResult() + $stderrTask.GetAwaiter().GetResult()).Trim()
        if ($process.ExitCode -ne 0) {
            throw "The bundled binary failed its --version smoke test: $versionOutput"
        }
        if ($versionOutput -notmatch [regex]::Escape($Version)) {
            throw "The bundled binary did not report version $Version. Output: $versionOutput"
        }
    }
    finally {
        $process.Dispose()
    }

    Compress-Archive -LiteralPath $binaryPath -DestinationPath $archivePath -CompressionLevel Optimal
    if (-not (Test-Path -LiteralPath $archivePath -PathType Leaf)) {
        throw "Archive creation failed: $archivePath"
    }

    $binaryHash = (Get-FileHash -LiteralPath $binaryPath -Algorithm SHA256).Hash.ToLowerInvariant()
    $archiveHash = (Get-FileHash -LiteralPath $archivePath -Algorithm SHA256).Hash.ToLowerInvariant()
    @(
        "$binaryHash  $binaryName"
        "$archiveHash  $archiveName"
    ) | Set-Content -LiteralPath $checksumsPath -Encoding ascii

    if (-not (Test-Path -LiteralPath $checksumsPath -PathType Leaf)) {
        throw "Checksum creation failed: $checksumsPath"
    }

    Write-Host "Windows test bundle created in $outputDir"
    Write-Host "  $archiveName"
    Write-Host '  checksums.txt'
}
finally {
    Pop-Location
    $env:CGO_ENABLED = $previousCgo
    $env:GOOS = $previousGoos
    $env:GOARCH = $previousGoarch
    $env:MULTIAI_SKIP_UPDATE = $previousSkipUpdate
}
