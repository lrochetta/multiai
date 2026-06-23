# Script de publication GitHub — repo PRIVE multiai
# Usage: .\push-github.ps1

$ErrorActionPreference = 'Stop'

Write-Host "multiai — Publication GitHub" -ForegroundColor Cyan

# 1. Créer le repo privé + push
Write-Host "[1/3] Création repo privé + push..." -ForegroundColor Yellow
gh repo create lrochetta/multiai --private --source=. --push --description "Route multiple AI CLIs with isolated env profiles"

# 2. Tag v0.2.0
Write-Host "[2/3] Tag v0.2.0..." -ForegroundColor Yellow
git tag v0.2.0
git push origin v0.2.0

# 3. Publier npm
Write-Host "[3/3] Publication npm..." -ForegroundColor Yellow
Push-Location "multiai-powershell"
npm publish
Pop-Location

Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "  PUBLICATION TERMINEE" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""
Write-Host "Repo  : https://github.com/lrochetta/multiai (PRIVE)"
Write-Host "npm   : npx multiai install"
Write-Host "Go    : go install github.com/lrochetta/multiai@v0.2.0"
