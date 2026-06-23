# Setup script for multiai Go development
# This script installs Go and builds the project for all platforms

$ErrorActionPreference = 'Stop'

Write-Host "multiai Go Setup" -ForegroundColor Cyan
Write-Host "===============" -ForegroundColor Cyan
Write-Host ''

# ── Check if Go is installed ──────────────────────────────────────────────
$goPath = Get-Command go -ErrorAction SilentlyContinue
if (-not $goPath) {
    Write-Host "Go non trouve. Installation..." -ForegroundColor Yellow

    $goVersion = "1.23.2"
    $goZip = "$env:TEMP\go$goVersion.zip"
    $goUrl = "https://go.dev/dl/go$goVersion.windows-amd64.zip"

    Write-Host "Telechargement de Go $goVersion..." -ForegroundColor Cyan
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    Invoke-WebRequest -Uri $goUrl -OutFile $goZip -UseBasicParsing

    Write-Host "Extraction dans C:\Go..." -ForegroundColor Cyan
    Expand-Archive -Path $goZip -DestinationPath "C:\" -Force

    $env:Path = "C:\Go\bin;$env:Path"
    [Environment]::SetEnvironmentVariable('Path', "C:\Go\bin;$([Environment]::GetEnvironmentVariable('Path', 'User'))", 'User')

    Write-Host "Go $goVersion installe." -ForegroundColor Green
} else {
    Write-Host "Go trouve : $(go version)" -ForegroundColor Green
}

# ── Build multiai ─────────────────────────────────────────────────────────
$projectDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$projectDir = Split-Path -Parent $projectDir

Write-Host ''
Write-Host "Build du projet..." -ForegroundColor Cyan
Push-Location $projectDir

Write-Host "1/4 go mod tidy..." -ForegroundColor Cyan
go mod tidy

Write-Host "2/4 go vet..." -ForegroundColor Cyan
go vet ./...
if ($LASTEXITCODE -ne 0) { throw "go vet a echoue" }

Write-Host "3/4 go build (Windows)..." -ForegroundColor Cyan
New-Item -ItemType Directory -Force -Path build | Out-Null
go build -o build/multiai.exe ./cmd/multiai/
if ($LASTEXITCODE -ne 0) { throw "go build a echoue" }
Write-Host "  -> build/multiai.exe" -ForegroundColor Green

Write-Host "4/4 go test..." -ForegroundColor Cyan
go test -race -v ./...

Write-Host ''
Write-Host "Cross-compilation..." -ForegroundColor Cyan
$env:CGO_ENABLED = '0'
GOOS=linux   GOARCH=amd64 go build -ldflags="-s -w" -o build/multiai-linux-amd64   ./cmd/multiai/
GOOS=darwin  GOARCH=amd64 go build -ldflags="-s -w" -o build/multiai-darwin-amd64  ./cmd/multiai/
GOOS=darwin  GOARCH=arm64 go build -ldflags="-s -w" -o build/multiai-darwin-arm64  ./cmd/multiai/

Write-Host ''
Write-Host "Binaires generes :" -ForegroundColor Green
Get-ChildItem build/ | ForEach-Object { Write-Host "  $($_.Name) ($([math]::Round($_.Length/1KB, 1)) KB)" }

Pop-Location
Write-Host ''
Write-Host "Setup termine avec succes !" -ForegroundColor Green
Write-Host "Lance : .\build\multiai.exe help" -ForegroundColor Cyan
