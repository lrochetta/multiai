#Requires -Version 5.1
<#
.SYNOPSIS
  Starts the local LiteLLM bridge for NVIDIA build.nvidia.com on port 4000.

.DESCRIPTION
  Required by the multiai profiles nv-cc (Claude Code) and codex-nv
  (Codex CLI), plus dynamic nv-* claude/codex profiles: the hosted NVIDIA
  endpoint is OpenAI chat/completions only, so Claude Code (Anthropic
  /v1/messages) and Codex 2026 (Responses API) need this translation
  layer. OpenCode profiles connect to NVIDIA directly and do not need it.

  Get a free key (nvapi-...): https://build.nvidia.com/settings/api-keys

.EXAMPLE
  .\nvidia-bridge.ps1                    # key from NVIDIA_NIM_API_KEY / NVIDIA_API_KEY or prompt
  .\nvidia-bridge.ps1 -ApiKey nvapi-...  # explicit key
  .\nvidia-bridge.ps1 -Port 4000
#>
param(
    [string]$ApiKey = $env:NVIDIA_NIM_API_KEY,
    [int]$Port = 4000
)

if (-not $ApiKey) { $ApiKey = $env:NVIDIA_API_KEY }
if (-not $ApiKey) {
    $sec = Read-Host "Cle NVIDIA (nvapi-...)" -AsSecureString
    $ApiKey = [Runtime.InteropServices.Marshal]::PtrToStringAuto(
        [Runtime.InteropServices.Marshal]::SecureStringToBSTR($sec))
}
if (-not $ApiKey) {
    Write-Error "Cle NVIDIA manquante. Genere-la sur https://build.nvidia.com/settings/api-keys"
    exit 1
}

if (-not (Get-Command litellm -ErrorAction SilentlyContinue)) {
    Write-Error 'litellm introuvable. Installe-le avec : pip install "litellm[proxy]" (eviter les versions 1.82.7/1.82.8, compromises sur PyPI).'
    exit 1
}

$config = Join-Path $PSScriptRoot "nvidia-litellm.yaml"
if (-not (Test-Path $config)) {
    Write-Error "Config introuvable : $config"
    exit 1
}

$env:NVIDIA_NIM_API_KEY = $ApiKey
Write-Host "Pont NVIDIA (LiteLLM) sur http://127.0.0.1:$Port - Ctrl+C pour arreter."
Write-Host "Profils multiai servis : nv-cc (Claude Code), codex-nv (Codex CLI), nv-* dynamiques."
litellm --config $config --port $Port --host 127.0.0.1
