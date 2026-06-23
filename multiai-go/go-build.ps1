# ============================================================================
# multiai — Build & Test complet pour Windows
# Usage : .\go-build.ps1
# ============================================================================
$ErrorActionPreference = 'Stop'
Write-Host "multiai Build Suite" -ForegroundColor Cyan
Write-Host "==================" -ForegroundColor Cyan

$projectDir = Split-Path -Parent $MyInvocation.MyCommand.Path

# ── Étape 0 : Installer Go si absent ─────────────────────────────────────
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "[0/5] Installation de Go 1.23.2..." -ForegroundColor Yellow
    $goUrl = "https://go.dev/dl/go1.23.2.windows-amd64.zip"
    $goZip = "$env:TEMP\go.zip"
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    Invoke-WebRequest -Uri $goUrl -OutFile $goZip -UseBasicParsing
    Expand-Archive -Path $goZip -DestinationPath "C:\" -Force
    Remove-Item $goZip
    $env:Path = "C:\Go\bin;$env:Path"
    [Environment]::SetEnvironmentVariable('Path', "C:\Go\bin;$([Environment]::GetEnvironmentVariable('Path', 'User'))", 'User')
    Write-Host "    Go installe : C:\Go\bin" -ForegroundColor Green
}

# ── Étape 1 : go mod tidy ────────────────────────────────────────────────
Write-Host "[1/5] go mod tidy..." -ForegroundColor Cyan
Push-Location $projectDir
go mod tidy
Write-Host "    OK" -ForegroundColor Green

# ── Étape 2 : go vet ─────────────────────────────────────────────────────
Write-Host "[2/5] go vet..." -ForegroundColor Cyan
go vet ./...
if ($LASTEXITCODE -ne 0) { throw "go vet a echoue" }
Write-Host "    OK - 0 warning" -ForegroundColor Green

# ── Étape 3 : go test ────────────────────────────────────────────────────
Write-Host "[3/5] go test (race + coverage)..." -ForegroundColor Cyan
New-Item -ItemType Directory -Force -Path build | Out-Null
go test -race -v -coverprofile=build/coverage.out ./... 2>&1 | Tee-Object -FilePath build/test-output.txt
if ($LASTEXITCODE -ne 0) { Write-Host "    Certains tests ont echoue" -ForegroundColor Red }
Write-Host "    Coverage: build/coverage.out" -ForegroundColor Green

# ── Étape 4 : go build + cross-compile ───────────────────────────────────
Write-Host "[4/5] go build (all platforms)..." -ForegroundColor Cyan
$env:CGO_ENABLED = '0'

$targets = @(
    @{OS="windows"; ARCH="amd64"; Ext=".exe"},
    @{OS="linux";   ARCH="amd64"; Ext=""},
    @{OS="darwin";  ARCH="amd64"; Ext=""},
    @{OS="darwin";  ARCH="arm64"; Ext=""}
)

foreach ($t in $targets) {
    $out = "build/multiai-$($t.OS)-$($t.ARCH)$($t.Ext)"
    Write-Host "    Building $out..."
    $env:GOOS = $t.OS
    $env:GOARCH = $t.ARCH
    go build -ldflags="-s -w -X main.version=0.2.0" -o $out ./cmd/multiai/
    if ($LASTEXITCODE -ne 0) { Write-Host "    ECHEC : $out" -ForegroundColor Red }
    else { Write-Host "    OK : $out ($([math]::Round((Get-Item $out).Length/1KB,1)) KB)" -ForegroundColor Green }
}

# ── Étape 5 : Benchmark ──────────────────────────────────────────────────
Write-Host "[5/5] go benchmark..." -ForegroundColor Cyan
go test -bench=. -benchmem ./tests/ 2>&1 | Tee-Object -FilePath build/benchmark-output.txt
Write-Host "    Benchmark: build/benchmark-output.txt" -ForegroundColor Green

Pop-Location

# ── Résumé ───────────────────────────────────────────────────────────────
Write-Host ""
Write-Host "=============================================" -ForegroundColor Cyan
Write-Host "  BUILD TERMINE" -ForegroundColor Green
Write-Host "=============================================" -ForegroundColor Cyan

$bins = Get-ChildItem build/ -Filter "multiai-*" | ForEach-Object {
    Write-Host "  $($_.Name) ($([math]::Round($_.Length/1KB,1)) KB)"
}

Write-Host ""
Write-Host "Lance le binaire Windows :" -ForegroundColor Cyan
Write-Host "  .\build\multiai-windows-amd64.exe help" -ForegroundColor Yellow
Write-Host ""
Write-Host "Documentation :" -ForegroundColor Cyan
Write-Host "  cd docs && npm install && npm run dev" -ForegroundColor Yellow
