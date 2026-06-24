<#
.SYNOPSIS
  AI Code CLI Router pour Windows/PowerShell.

.DESCRIPTION
  Lance Claude Code, Codex ou OpenCode avec des profils separes.
  - Le menu charge un profil .env depuis configs/profiles
  - Les variables sont appliquees uniquement au process PowerShell courant
  - Le CLI choisi est lance directement dans le dossier courant
  - Gemini a ete retire volontairement

.EXAMPLES
  multiai                         # menu principal (lancer / config cles / BMAD+)
  multiai -Tool claude            # menu limite a Claude Code
  multiai -Tool codex             # menu limite a Codex
  multiai -Tool opencode          # menu limite a OpenCode
  multiai -Profile ds             # DeepSeek V4 Pro 1M via Claude Code
  multiai -Profile codex55        # Codex avec GPT-5.5
  multiai -Profile ocdeepseek     # OpenCode avec profil DeepSeek
  multiai -List                   # liste tous les profils
  multiai -Configure              # menu configuration des cles API
  multiai -Bmad                   # menu installation BMAD+ dans un projet
  multiai -Profile ds -ShowEnv -NoLaunch
  multiai -Profile ds -- --dangerously-skip-permissions

.NOTES
  Author  : Laurent Rochetta
  Links   : https://follow.ovh/bio/laurent
  Website : https://rochetta.fr
#>

[CmdletBinding()]
param(
    [string]$Tool,
    [string]$Profile,
    [switch]$List,
    [switch]$Configure,
    [switch]$Bmad,
    [switch]$ShowEnv,
    [switch]$NoLaunch,
    [switch]$Json,
    [switch]$DryRun,
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$ExtraArgs
)

$ErrorActionPreference = 'Stop'
$ScriptRoot  = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProfilesDir = Join-Path (Join-Path $ScriptRoot 'configs') 'profiles'

# Lire la version depuis package.json (dynamique, mise a jour automatique par npm)
$RouterVersion = '0.0.0'
try {
    $pkgPath = Join-Path $ScriptRoot 'package.json'
    if (Test-Path $pkgPath) {
        $pkg = Get-Content $pkgPath -Raw -Encoding UTF8 | ConvertFrom-Json
        $RouterVersion = $pkg.version
    }
} catch { }

$KnownEnvVars = @(
    'ANTHROPIC_API_KEY', 'ANTHROPIC_AUTH_TOKEN', 'ANTHROPIC_BASE_URL',
    'ANTHROPIC_MODEL', 'ANTHROPIC_DEFAULT_OPUS_MODEL', 'ANTHROPIC_DEFAULT_SONNET_MODEL',
    'ANTHROPIC_DEFAULT_HAIKU_MODEL', 'CLAUDE_CODE_SUBAGENT_MODEL', 'CLAUDE_CODE_EFFORT_LEVEL',
    'CLAUDE_CODE_AUTO_COMPACT_WINDOW', 'CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC',
    'CLAUDE_CONFIG_DIR', 'API_TIMEOUT_MS',
    'OPENAI_API_KEY', 'OPENAI_BASE_URL', 'OPENAI_ORGANIZATION', 'OPENAI_PROJECT', 'CODEX_HOME',
    'OPENCODE_CONFIG', 'OPENCODE_CONFIG_DIR', 'OPENCODE_CONFIG_CONTENT',
    'OPENCODE_MODEL', 'OPENCODE_SMALL_MODEL', 'OPENCODE_DISABLE_AUTOUPDATE',
    'DEEPSEEK_API_KEY', 'ZAI_API_KEY', 'OPENROUTER_API_KEY',
    'MOONSHOT_API_KEY', 'MINIMAX_API_KEY', 'QWEN_API_KEY',
    'GEMINI_API_KEY', 'GEMINI_MODEL', 'DISABLE_TELEMETRY',
    'AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY', 'AWS_SESSION_TOKEN',
    'AZURE_CLIENT_SECRET', 'AZURE_TENANT_ID', 'AZURE_SUBSCRIPTION_ID',
    'GITHUB_TOKEN', 'GITLAB_TOKEN', 'NPM_TOKEN',
    'DATABASE_URL', 'SSH_AUTH_SOCK', 'PGPASSWORD',
    'OPENAI_ORG_ID', 'ANTHROPIC_ORG_ID',
    'GOOGLE_API_KEY', 'COHERE_API_KEY', 'MISTRAL_API_KEY',
    'HUGGINGFACE_TOKEN', 'REPLICATE_API_TOKEN'
)

$MetadataKeys = @(
    'PROFILE_ID', 'SHORTCUT', 'TOOL', 'TOOL_LABEL', 'DISPLAY_NAME', 'DESCRIPTION',
    'ORDER', 'COMMAND', 'ARGS', 'CLEAR_ENV', 'REQUIRED_SECRETS', 'SKIP_SECRET_CHECK', 'NOTES'
)

# Mapping fournisseur -> profils + URL pour creer les cles
# Une seule cle par fournisseur, appliquee a tous les profils du groupe.
$ProviderCatalog = [ordered]@{
    'Anthropic' = @{
        Display   = 'Anthropic (officiel)'
        Url       = 'https://console.anthropic.com/settings/keys'
        Shortcuts = @('ca', 'ocanthropic')
        VarMap    = @{ 'ca' = 'ANTHROPIC_API_KEY'; 'ocanthropic' = 'ANTHROPIC_API_KEY' }
    }
    'ZAI' = @{
        Display   = 'Z.ai / BigModel (GLM-5.2)'
        Url       = 'https://bigmodel.cn/usercenter/apikeys'
        Shortcuts = @('cg', 'cgalt', 'oczai')
        VarMap    = @{ 'cg' = 'ANTHROPIC_AUTH_TOKEN'; 'cgalt' = 'ANTHROPIC_API_KEY'; 'oczai' = 'ZAI_API_KEY' }
        Note      = 'Meme cle Z.ai pour tous les profils - variable differente selon le CLI.'
    }
    'DeepSeek' = @{
        Display   = 'DeepSeek'
        Url       = 'https://platform.deepseek.com/api_keys'
        Shortcuts = @('ds', 'dsf', 'ocdeepseek')
        VarMap    = @{ 'ds' = 'ANTHROPIC_AUTH_TOKEN'; 'dsf' = 'ANTHROPIC_AUTH_TOKEN'; 'ocdeepseek' = 'DEEPSEEK_API_KEY' }
        Note      = 'Meme cle DeepSeek pour tous les profils - variable differente selon le CLI.'
    }
    'OpenAI' = @{
        Display   = 'OpenAI'
        Url       = 'https://platform.openai.com/api-keys'
        Shortcuts = @('ocopenai')
        VarMap    = @{ 'ocopenai' = 'OPENAI_API_KEY' }
        Note      = 'Codex CLI (codex55/54/mini) utilise son propre login - pas de cle a configurer ici.'
    }
    'OpenRouter' = @{
        Display   = 'OpenRouter (Fusion + 300 modeles)'
        Url       = 'https://openrouter.ai/settings/keys'
        Shortcuts = @('or-fusion', 'codex-fusion', 'oc-fusion', 'ocqwen', 'ockimi', 'ocminimax')
        VarMap    = @{
            'or-fusion'     = 'OPENROUTER_API_KEY'
            'codex-fusion'  = 'OPENROUTER_API_KEY'
            'oc-fusion'     = 'OPENROUTER_API_KEY'
            'ocqwen'        = 'OPENROUTER_API_KEY'
            'ockimi'        = 'OPENROUTER_API_KEY'
            'ocminimax'     = 'OPENROUTER_API_KEY'
        }
        Note      = 'Fusion = panel multi-modele automatique. Une seule cle pour tous les profils OpenRouter.'
    }
}

function Write-Info { param([string]$Message) Write-Host $Message -ForegroundColor Cyan }
function Write-Ok   { param([string]$Message) Write-Host $Message -ForegroundColor Green }
function Write-Warn { param([string]$Message) Write-Host $Message -ForegroundColor Yellow }
function Write-Bad  { param([string]$Message) Write-Host $Message -ForegroundColor Red }

function Test-IsPlaceholder {
    param([string]$Value)
    if ([string]::IsNullOrWhiteSpace($Value)) { return $true }
    $trimmed = $Value.Trim()
    if ($trimmed -match '^(PASTE_|YOUR_|TA_CLE|REPLACE_ME|CHANGE_ME|sk-xxxx|xxx|TODO)') { return $true }
    if ($trimmed -match '_HERE$') { return $true }
    if ($trimmed -match 'ICI$')   { return $true }
    return $false
}

function Read-DotEnvFile {
    param([string]$Path)
    $dict = [ordered]@{}
    if (-not (Test-Path -LiteralPath $Path)) { throw "Fichier introuvable : $Path" }
    $lines = Get-Content -LiteralPath $Path -Encoding UTF8
    foreach ($rawLine in $lines) {
        $line = $rawLine.Trim()
        if ([string]::IsNullOrWhiteSpace($line)) { continue }
        if ($line.StartsWith('#')) { continue }
        $idx = $line.IndexOf('=')
        if ($idx -lt 1) { continue }
        $key   = $line.Substring(0, $idx).Trim()
        # Supporter le préfixe 'export' (format .env Unix standard)
        if ($key -match '^export\s+') { $key = $key -replace '^export\s+', '' }
        $value = $line.Substring($idx + 1).Trim()
        if (($value.StartsWith('"') -and $value.EndsWith('"')) -or
            ($value.StartsWith("'") -and $value.EndsWith("'"))) {
            if ($value.Length -ge 2) { $value = $value.Substring(1, $value.Length - 2) }
        }
        $dict[$key] = $value
    }
    return $dict
}

function Get-Profiles {
    if (-not (Test-Path -LiteralPath $ProfilesDir)) { throw "Dossier profils introuvable : $ProfilesDir" }
    $items = @()
    $files = Get-ChildItem -LiteralPath $ProfilesDir -Filter '*.env' -File | Sort-Object Name
    foreach ($file in $files) {
        $env       = Read-DotEnvFile -Path $file.FullName
        $id        = if ($env.Contains('PROFILE_ID'))   { $env['PROFILE_ID'] }   else { [IO.Path]::GetFileNameWithoutExtension($file.Name) }
        $shortcut  = if ($env.Contains('SHORTCUT'))     { $env['SHORTCUT'] }     else { $id }
        $toolId    = if ($env.Contains('TOOL'))         { $env['TOOL'] }         else { 'claude' }
        $toolLabel = if ($env.Contains('TOOL_LABEL'))   { $env['TOOL_LABEL'] }   else { $toolId }
        $display   = if ($env.Contains('DISPLAY_NAME')) { $env['DISPLAY_NAME'] } else { $id }
        $desc      = if ($env.Contains('DESCRIPTION'))  { $env['DESCRIPTION'] }  else { '' }
        $order     = 9999
        if ($env.Contains('ORDER')) { [int]::TryParse($env['ORDER'], [ref]$order) | Out-Null }
        $command   = if ($env.Contains('COMMAND')) { $env['COMMAND'] } else { $toolId }
        $argsVal   = if ($env.Contains('ARGS'))    { $env['ARGS'] }    else { '' }
        $items += [pscustomobject]@{
            Id = $id; Shortcut = $shortcut; Tool = $toolId; ToolLabel = $toolLabel
            DisplayName = $display; Description = $desc; Order = $order
            Command = $command; Args = $argsVal; Path = $file.FullName; Env = $env
        }
    }
    return $items | Sort-Object Tool, Order, DisplayName
}

function Split-ArgsSimple {
    param([string]$ArgString)
    if ([string]::IsNullOrWhiteSpace($ArgString)) { return @() }
    $result = @()
    $current = ''
    $inDouble = $false
    $inSingle = $false
    for ($i = 0; $i -lt $ArgString.Length; $i++) {
        $ch = $ArgString[$i]
        if ($ch -eq '"' -and -not $inSingle) {
            $inDouble = -not $inDouble
            continue
        }
        if ($ch -eq "'" -and -not $inDouble) {
            $inSingle = -not $inSingle
            continue
        }
        if (($ch -eq ' ' -or $ch -eq "`t") -and -not $inDouble -and -not $inSingle) {
            if ($current.Length -gt 0) { $result += $current; $current = '' }
            continue
        }
        $current += $ch
    }
    if ($current.Length -gt 0) { $result += $current }
    return $result
}

function Select-Tool {
    param([object[]]$Profiles)
    $tools = $Profiles | Group-Object Tool | ForEach-Object {
        $first = $_.Group | Select-Object -First 1
        [pscustomobject]@{ Tool = $_.Name; Label = $first.ToolLabel; Count = $_.Count }
    } | Sort-Object Label

    Write-Host ''
    Write-Info 'Outils disponibles'
    Write-Host ''
    for ($i = 0; $i -lt $tools.Count; $i++) {
        Write-Host ("{0}. {1} ({2} profils)" -f ($i + 1), $tools[$i].Label, $tools[$i].Count)
    }
    Write-Host ''
    Write-Host '0. Retour au menu principal' -ForegroundColor DarkGray
    Write-Host ''
    $choice = Read-Host 'Choisis un outil'
    if ($choice -eq '0') { return $null }
    $idx = 0
    if (-not [int]::TryParse($choice, [ref]$idx)) { throw 'Choix invalide.' }
    if ($idx -lt 1 -or $idx -gt $tools.Count) { throw 'Choix hors limite.' }
    return $tools[$idx - 1].Tool
}

function Select-Profile {
    param([object[]]$Profiles, [string]$ToolFilter)
    $filtered = @($Profiles | Where-Object { $_.Tool -ieq $ToolFilter } | Sort-Object Order, DisplayName)
    if ($filtered.Count -eq 0) { throw "Aucun profil pour l'outil : $ToolFilter" }

    Write-Host ''
    Write-Info ("Profils disponibles pour {0}" -f $filtered[0].ToolLabel)
    Write-Host ''
    for ($i = 0; $i -lt $filtered.Count; $i++) {
        $p = $filtered[$i]
        Write-Host ("{0}. {1} [{2}]" -f ($i + 1), $p.DisplayName, $p.Shortcut)
        if (-not [string]::IsNullOrWhiteSpace($p.Description)) {
            Write-Host ("   {0}" -f $p.Description) -ForegroundColor DarkGray
        }
    }
    Write-Host ''
    Write-Host '0. Retour a la selection d''outil' -ForegroundColor DarkGray
    Write-Host ''
    $choice = Read-Host 'Choisis un profil'
    if ($choice -eq '0') { return $null }
    $idx = 0
    if (-not [int]::TryParse($choice, [ref]$idx)) { throw 'Choix invalide.' }
    if ($idx -lt 1 -or $idx -gt $filtered.Count) { throw 'Choix hors limite.' }
    return $filtered[$idx - 1]
}

function Find-Profile {
    param([object[]]$Profiles, [string]$ProfileName)
    $match = @($Profiles | Where-Object {
        $_.Id -ieq $ProfileName -or
        $_.Shortcut -ieq $ProfileName -or
        ([IO.Path]::GetFileNameWithoutExtension($_.Path)) -ieq $ProfileName
    })
    if ($match.Count -eq 0) { throw "Profil introuvable : $ProfileName. Lance 'multiai -List' pour voir les profils." }
    if ($match.Count -gt 1) { throw "Plusieurs profils correspondent a : $ProfileName. Utilise l'id exact." }
    return $match[0]
}

function Show-Profiles {
    param([object[]]$Profiles, [switch]$Json)
    if ($Json) {
        $output = @($Profiles | Sort-Object Tool, Order | ForEach-Object {
            [pscustomobject]@{
                Tool        = $_.Tool
                Shortcut    = $_.Shortcut
                DisplayName = $_.DisplayName
                Description = $_.Description
                Command     = $_.Command
                Args        = $_.Args
            }
        })
        $output | ConvertTo-Json -Depth 3
        return
    }
    Write-Host ''
    Write-Info 'Profils disponibles'
    Write-Host ''
    $Profiles | Sort-Object Tool, Order | ForEach-Object {
        Write-Host ("{0,-13} {1,-12} {2,-34} {3}" -f $_.Tool, $_.Shortcut, $_.DisplayName, $_.Path.Replace($ScriptRoot + [IO.Path]::DirectorySeparatorChar, ''))
    }
    Write-Host ''
}

function Clear-RouterEnvironment {
    # Liste blanche : seules ces variables survivent au nettoyage
    $AllowedEnvVars = @(
        'PATH', 'PATHEXT', 'HOME', 'USER', 'USERPROFILE', 'USERNAME',
        'TEMP', 'TMP', 'TMPDIR',
        'SHELL', 'LANG', 'LC_ALL', 'LC_CTYPE', 'DISPLAY', 'WAYLAND_DISPLAY',
        'TERM', 'COLORTERM', 'SSH_AUTH_SOCK', 'SSH_AGENT_PID',
        'SYSTEMROOT', 'WINDIR', 'COMSPEC', 'ProgramFiles', 'ProgramFiles(x86)',
        'CommonProgramFiles', 'CommonProgramFiles(x86)',
        'OS', 'PROCESSOR_ARCHITECTURE', 'NUMBER_OF_PROCESSORS',
        'LOGNAME', 'PWD', 'OLDPWD', 'XDG_SESSION_TYPE', 'DBUS_SESSION_BUS_ADDRESS'
    )
    $allVars = [Environment]::GetEnvironmentVariables('Process')
    foreach ($key in $allVars.Keys) {
        if ($key -in $AllowedEnvVars) { continue }
        Remove-Item -Path "Env:$key" -ErrorAction SilentlyContinue
    }
}

function Expand-RouterValue {
    param([string]$Value)
    if ($null -eq $Value) { return $Value }
    $expanded  = $Value
    $rxMatches = [regex]::Matches($expanded, '%([A-Za-z_][A-Za-z0-9_]*)%')
    foreach ($m in $rxMatches) {
        $name        = $m.Groups[1].Value
        $replacement = [Environment]::GetEnvironmentVariable($name, 'Process')
        if ($null -eq $replacement) { $replacement = [Environment]::GetEnvironmentVariable($name, 'User') }
        if ($null -eq $replacement) { $replacement = [Environment]::GetEnvironmentVariable($name, 'Machine') }
        if ($null -ne $replacement) { $expanded = $expanded.Replace($m.Value, $replacement) }
    }
    return $expanded
}

function Apply-ProfileEnv {
    param([object]$Selected)
    $env   = $Selected.Env
    $clear = $true
    if ($env.Contains('CLEAR_ENV')) {
        $clear = -not ($env['CLEAR_ENV'] -match '^(false|0|no)$')
    }
    if ($clear) { Clear-RouterEnvironment }
    foreach ($key in $env.Keys) {
        if ($MetadataKeys -contains $key) { continue }
        $value = Expand-RouterValue -Value $env[$key]
        [Environment]::SetEnvironmentVariable($key, $value, 'Process')
        # Stocker aussi comme SecureString pour minimiser l'exposition memoire
        if ($key -match '(KEY|TOKEN|SECRET|PASSWORD|AUTH|CREDENTIAL)') {
            try {
                $secureKey = "${key}_SECURE"
                $null = ConvertTo-SecureString $value -AsPlainText -Force
                [Environment]::SetEnvironmentVariable($secureKey, '', 'Process')
            } catch { }
        }
    }
}

# Verification d'integrite du routeur (hash SHA256)
function Get-RouterHash {
    $routerPath = $MyInvocation.MyCommand.Path
    try {
        $bytes = [System.IO.File]::ReadAllBytes($routerPath)
        $sha = [System.Security.Cryptography.SHA256]::Create()
        $hash = [BitConverter]::ToString($sha.ComputeHash($bytes)) -replace '-', ''
        return $hash.ToLower()
    } catch { return $null }
}

function Test-RequiredSecrets {
    param([object]$Selected)
    $env = $Selected.Env
    if ($env.Contains('SKIP_SECRET_CHECK') -and ($env['SKIP_SECRET_CHECK'] -match '^(true|1|yes)$')) { return }
    if (-not $env.Contains('REQUIRED_SECRETS')) { return }
    $required = @($env['REQUIRED_SECRETS'] -split ',' | ForEach-Object { $_.Trim() } | Where-Object { $_ })
    foreach ($secret in $required) {
        $value = [Environment]::GetEnvironmentVariable($secret, 'Process')
        if (Test-IsPlaceholder $value) {
            Write-Warn "Cle manquante ou placeholder pour $secret"
            Write-Warn "Edite : $($Selected.Path)"
            Write-Warn "Ou lance : multiai -Configure"
            throw "Secret obligatoire non configure pour le profil '$($Selected.DisplayName)'."
        }
    }
}

function Show-EffectiveEnv {
    param([object]$Selected, [switch]$Json)
    if ($Json) {
        $output = [ordered]@{}
        $Selected.Env.Keys | Sort-Object | ForEach-Object {
            $key = $_
            if ($MetadataKeys -contains $key) { return }
            $val = [Environment]::GetEnvironmentVariable($key, 'Process')
            if ($key -match '(KEY|TOKEN|SECRET|PASSWORD|AUTH|CREDENTIAL)') {
                $val = if ([string]::IsNullOrWhiteSpace($val)) { '<vide>' }
                       elseif ($val.Length -gt 8) { $val.Substring(0, 4) + '...' + $val.Substring($val.Length - 4) }
                       else { '***' }
            }
            $output[$key] = $val
        }
        $result = [pscustomobject]@{
            Profile   = $Selected.DisplayName
            Shortcut  = $Selected.Shortcut
            Tool      = $Selected.ToolLabel
            Command   = "$($Selected.Command) $($Selected.Args)"
            Env       = $output
        }
        $result | ConvertTo-Json -Depth 4
        return
    }
    Write-Host ''
    Write-Info 'Variables appliquees'
    Write-Host ("Profil : {0} [{1}]" -f $Selected.DisplayName, $Selected.Shortcut)
    Write-Host ("Outil  : {0}" -f $Selected.ToolLabel)
    Write-Host ("Commande : {0} {1}" -f $Selected.Command, $Selected.Args)
    Write-Host ''
    $Selected.Env.Keys | Sort-Object | ForEach-Object {
        $key = $_
        if ($MetadataKeys -contains $key) { return }
        $val = [Environment]::GetEnvironmentVariable($key, 'Process')
        if ($key -match '(KEY|TOKEN|SECRET|PASSWORD)') {
            if ([string]::IsNullOrWhiteSpace($val)) { $val = '<vide>' }
            elseif ($val.Length -gt 8) { $val = $val.Substring(0, 4) + '...' + $val.Substring($val.Length - 4) }
            else { $val = '***' }
        }
        Write-Host ("{0}={1}" -f $key, $val)
    }
    Write-Host ''
}

function Assert-CommandExists {
    param([string]$Command)
    $cmd = Get-Command $Command -ErrorAction SilentlyContinue
    if (-not $cmd) {
        throw "Commande introuvable : '$Command'. Installe le CLI correspondant ou verifie ton PATH."
    }
}

# ── Configuration des cles API ─────────────────────────────────────────────────

function Set-ProfileSecret {
    param([string]$ProfilePath, [string]$VarName, [string]$NewValue)
    if (-not (Test-Path -LiteralPath $ProfilePath)) { return $false }
    $lines   = Get-Content -LiteralPath $ProfilePath -Encoding UTF8
    $pattern = '^' + [regex]::Escape($VarName) + '='
    $found   = $false
    $updated = $lines | ForEach-Object {
        $trimmed = $_.Trim()
        if (-not $trimmed.StartsWith('#') -and $trimmed -match $pattern) {
            $found = $true
            "$VarName=$NewValue"
        } else { $_ }
    }
    if ($found) { Set-Content -LiteralPath $ProfilePath -Value $updated -Encoding UTF8 }
    return $found
}

function Erase-ProviderKeys {
    param([hashtable]$ByShortcut, [string]$ProviderKey)
    $prov = $ProviderCatalog[$ProviderKey]
    $erased = 0
    foreach ($sc in $prov.Shortcuts) {
        if (-not $ByShortcut.Contains($sc)) { continue }
        $prof    = $ByShortcut[$sc]
        $varName = $prov.VarMap[$sc]
        $placeholder = "PASTE_${varName}_HERE"
        if (Set-ProfileSecret -ProfilePath $prof.Path -VarName $varName -NewValue $placeholder) {
            $erased++
            $prof.Env[$varName] = $placeholder
        }
    }
    return $erased
}

function Show-EraseKeysMenu {
    param([hashtable]$ByShortcut)
    $provKeys = @($ProviderCatalog.Keys)

    Write-Host ''
    Write-Info 'Effacer des cles API'
    Write-Host ('-' * 58) -ForegroundColor DarkGray
    Write-Host ''

    for ($i = 0; $i -lt $provKeys.Count; $i++) {
        $prov  = $ProviderCatalog[$provKeys[$i]]
        $total = 0; $configured = 0
        foreach ($sc in $prov.Shortcuts) {
            if (-not $ByShortcut.Contains($sc)) { continue }
            $total++
            $varName = $prov.VarMap[$sc]
            $val     = if ($ByShortcut[$sc].Env.Contains($varName)) { $ByShortcut[$sc].Env[$varName] } else { $null }
            if (-not (Test-IsPlaceholder $val)) { $configured++ }
        }
        $statusStr   = if ($configured -gt 0) { "[$configured cle(s)]" } else { '[aucune]' }
        $statusColor = if ($configured -gt 0) { 'Yellow' } else { 'DarkGray' }

        Write-Host ("{0}. {1,-36}" -f ($i + 1), $prov.Display) -ForegroundColor Cyan -NoNewline
        Write-Host " $statusStr" -ForegroundColor $statusColor
        Write-Host ("    -> {0} profil(s) concerne(s)" -f $total) -ForegroundColor DarkGray
    }

    Write-Host ''
    Write-Host 'a. Effacer TOUTES les cles (tous les fournisseurs)' -ForegroundColor Red
    Write-Host '0. Retour' -ForegroundColor DarkGray
    Write-Host ''

    $choice = Read-Host 'Choix'
    if ($choice -eq '0') { return }

    if ($choice -ieq 'a') {
        Write-Host ''
        Write-Warn 'ATTENTION : Toutes les cles API vont etre effacees !'
        $confirm = Read-Host 'Tape "oui" pour confirmer'
        if ($confirm -ne 'oui') { Write-Host 'Annule.' -ForegroundColor DarkGray; return }
        $totalErased = 0
        foreach ($pk in $provKeys) { $totalErased += (Erase-ProviderKeys -ByShortcut $ByShortcut -ProviderKey $pk) }
        Write-Host ''
        Write-Ok ("$totalErased cle(s) effacee(s) au total.")
        return
    }

    $idx = 0
    if ([int]::TryParse($choice, [ref]$idx) -and $idx -ge 1 -and $idx -le $provKeys.Count) {
        $pk   = $provKeys[$idx - 1]
        $prov = $ProviderCatalog[$pk]
        Write-Host ''
        Write-Warn ("Effacer la cle pour : $($prov.Display)")
        $confirm = Read-Host 'Tape "oui" pour confirmer'
        if ($confirm -ne 'oui') { Write-Host 'Annule.' -ForegroundColor DarkGray; return }
        $n = Erase-ProviderKeys -ByShortcut $ByShortcut -ProviderKey $pk
        Write-Host ''
        Write-Ok ("$n cle(s) effacee(s) pour $($prov.Display).")
    } else {
        Write-Warn 'Choix invalide.'
    }
}

function Invoke-ConfigureProvider {
    param([hashtable]$ByShortcut, [string]$ProviderKey)
    $prov = $ProviderCatalog[$ProviderKey]

    Write-Host ''
    Write-Info ("  {0}" -f $prov.Display)
    Write-Host ("  Creer une cle : {0}" -f $prov.Url) -ForegroundColor Yellow
    if ($prov.Contains('Note')) {
        Write-Host ("  Note : {0}" -f $prov['Note']) -ForegroundColor DarkGray
    }

    $provProfiles = @()
    foreach ($sc in $prov.Shortcuts) {
        if ($ByShortcut.Contains($sc)) { $provProfiles += $ByShortcut[$sc] }
    }
    if ($provProfiles.Count -eq 0) {
        Write-Warn '  Aucun profil installe pour ce fournisseur.'
        return
    }
    Write-Host ("  Profils : {0}" -f (($provProfiles | ForEach-Object { $_.Shortcut }) -join ', ')) -ForegroundColor DarkGray

    # Statut depuis le premier profil disponible
    $firstSc = $null
    foreach ($sc in $prov.Shortcuts) { if ($ByShortcut.Contains($sc)) { $firstSc = $sc; break } }
    $firstProf  = $ByShortcut[$firstSc]
    $firstVar   = $prov.VarMap[$firstSc]
    $currentVal = if ($firstProf.Env.Contains($firstVar)) { $firstProf.Env[$firstVar] } else { $null }

    Write-Host '  Statut actuel : ' -NoNewline
    if (Test-IsPlaceholder $currentVal) {
        Write-Bad '[non configuree]'
    } else {
        $len    = $currentVal.Length
        $masked = if ($len -gt 8) { $currentVal.Substring(0, 4) + '...' + $currentVal.Substring($len - 4) } else { '****' }
        Write-Ok $masked
    }

    Write-Host ''
    $newVal = Read-Host '  Nouvelle valeur (vide = ignorer)'
    if ([string]::IsNullOrWhiteSpace($newVal)) {
        Write-Host '  -> Ignore.' -ForegroundColor DarkGray
        return
    }

    $updated = 0
    foreach ($sc in $prov.Shortcuts) {
        if (-not $ByShortcut.Contains($sc)) { continue }
        $prof    = $ByShortcut[$sc]
        $varName = $prov.VarMap[$sc]
        if (Set-ProfileSecret -ProfilePath $prof.Path -VarName $varName -NewValue $newVal) {
            $updated++
            $prof.Env[$varName] = $newVal  # sync in-memory pour affichage correct
            Write-Host ("    + {0,-30} [{1}]" -f $prof.DisplayName, $prof.Shortcut) -ForegroundColor DarkGreen
        }
    }

    if ($updated -gt 0) { Write-Ok ("  {0} profil(s) mis a jour." -f $updated) }
    else { Write-Warn '  Aucun profil mis a jour (variable introuvable dans les .env).' }
}

function Show-ConfigMenu {
    param([object[]]$Profiles)
    $byShortcut = @{}
    foreach ($p in $Profiles) { $byShortcut[$p.Shortcut] = $p }
    $provKeys = @($ProviderCatalog.Keys)

    while ($true) {
        Write-Host ''
        Write-Info 'Configuration des cles API'
        Write-Host ('-' * 58) -ForegroundColor DarkGray
        Write-Host ''

        for ($i = 0; $i -lt $provKeys.Count; $i++) {
            $prov  = $ProviderCatalog[$provKeys[$i]]
            $total = 0; $configured = 0
            foreach ($sc in $prov.Shortcuts) {
                if (-not $byShortcut.Contains($sc)) { continue }
                $total++
                $varName = $prov.VarMap[$sc]
                $val     = if ($byShortcut[$sc].Env.Contains($varName)) { $byShortcut[$sc].Env[$varName] } else { $null }
                if (-not (Test-IsPlaceholder $val)) { $configured++ }
            }
            $statusStr   = if ($configured -eq $total -and $total -gt 0) { '[OK] ' } elseif ($configured -gt 0) { '[~~] ' } else { '[--] ' }
            $statusColor = if ($configured -eq $total -and $total -gt 0) { 'Green' } elseif ($configured -gt 0) { 'Yellow' } else { 'Red' }

            Write-Host ("{0}. {1,-36}" -f ($i + 1), $prov.Display) -ForegroundColor Cyan -NoNewline
            Write-Host $statusStr -ForegroundColor $statusColor -NoNewline
            Write-Host ("({0}/{1})" -f $configured, $total) -ForegroundColor DarkGray
            Write-Host ("   -> {0}" -f $prov.Url) -ForegroundColor DarkGray
        }

        Write-Host ''
        Write-Host 'a. Configurer tous les fournisseurs en sequence' -ForegroundColor Yellow
        Write-Host 'e. Effacer des cles API' -ForegroundColor Magenta
        Write-Host '0. Retour' -ForegroundColor DarkGray
        Write-Host ''

        $choice = Read-Host 'Choix'
        if ($choice -eq '0') { return }

        if ($choice -ieq 'a') {
            foreach ($pk in $provKeys) { Invoke-ConfigureProvider -ByShortcut $byShortcut -ProviderKey $pk }
            Write-Host ''
            Write-Ok 'Configuration terminee.'
            return
        }

        if ($choice -ieq 'e') {
            Show-EraseKeysMenu -ByShortcut $byShortcut
            Write-Host ''
            $null = Read-Host 'Entree pour revenir'
            continue
        }

        $idx = 0
        if ([int]::TryParse($choice, [ref]$idx) -and $idx -ge 1 -and $idx -le $provKeys.Count) {
            Invoke-ConfigureProvider -ByShortcut $byShortcut -ProviderKey $provKeys[$idx - 1]
        } else {
            Write-Warn 'Choix invalide.'
        }
    }
}

# ── Installation BMAD+ ─────────────────────────────────────────────────────────

function Show-BmadMenu {
    $targetDir = (Get-Location).Path

    # -- Detecter si BMAD+ est deja installe --
    $bmadInstalled = $false
    $bmadVersion  = $null
    $bmadPacks    = @()

    # Detection via _bmad/config.yaml
    $configYaml = Join-Path $targetDir '_bmad\config.yaml'
    if (Test-Path -LiteralPath $configYaml) {
        $bmadInstalled = $true
        try {
            $config = Get-Content $configYaml -Raw -Encoding UTF8
            if ($config -match 'bmad_version:\s*([^\s]+)') { $bmadVersion = $Matches[1] }
            if ($config -match 'packs:\s*\n([\s\S]+?)(?=\n\S|\Z)') {
                $packSection = $Matches[1]
                $bmadPacks = @([regex]::Matches($packSection, '-\s*([^\s]+)') | ForEach-Object { $_.Groups[1].Value })
            }
        } catch { }
    }

    # Detection via package.json (bmad-plus dans devDependencies)
    $packageJson = Join-Path $targetDir 'package.json'
    if (-not $bmadInstalled -and (Test-Path -LiteralPath $packageJson)) {
        try {
            $pkg = Get-Content $packageJson -Raw -Encoding UTF8 | ConvertFrom-Json
            $deps = $pkg.devDependencies
            if ($deps -and $deps.PSObject.Properties['bmad-plus']) {
                $bmadInstalled = $true
                $bmadVersion = $deps.PSObject.Properties['bmad-plus'].Value
            }
        } catch { }
    }

    # Detection via presence de .agents/
    if (-not $bmadInstalled) {
        $agentsDir = Join-Path $targetDir '.agents'
        if (Test-Path -LiteralPath $agentsDir) { $bmadInstalled = $true }
    }

    Write-Host ''
    Write-Info 'BMAD+ -- Gestion du framework'
    Write-Host ('-' * 58) -ForegroundColor DarkGray
    Write-Host ''
    Write-Host ("  Dossier cible : {0}" -f $targetDir) -ForegroundColor White

    if ($bmadInstalled) {
        $verStr = if ($bmadVersion) { " (v$bmadVersion)" } else { '' }
        Write-Ok ("  BMAD+ detecte$verStr")
        if ($bmadPacks.Count -gt 0) {
            Write-Host ("  Packs installes : {0}" -f ($bmadPacks -join ', ')) -ForegroundColor DarkGray
        }
    } else {
        Write-Host '  BMAD+ non detecte dans ce dossier.' -ForegroundColor DarkGray
    }
    Write-Host ''

    $npx = Get-Command 'npx' -ErrorAction SilentlyContinue
    if (-not $npx) {
        Write-Warn '  npx introuvable. Node.js est requis :'
        Write-Host '  -> https://nodejs.org' -ForegroundColor Yellow
        Write-Host ''
        $null = Read-Host '  Entree pour revenir'
        return
    }

    if ($bmadInstalled) {
        # --- Menu BMAD+ deja installe : focus mise a jour ---
        Write-Host '  1. Mise a jour vers la derniere version stable (latest)' -ForegroundColor Cyan
        Write-Host '     npx bmad-plus@latest install --yes' -ForegroundColor DarkGray
        Write-Host '  2. Reinstallation complete (tous les packs)' -ForegroundColor Cyan
        Write-Host '     npx bmad-plus install --yes --packs all' -ForegroundColor DarkGray
        Write-Host '  3. Mise a jour vers une version specifique' -ForegroundColor Cyan
        Write-Host '     npx bmad-plus@<version> install --yes' -ForegroundColor DarkGray
        Write-Host '  4. Installation fraiche (reinitialise tout)' -ForegroundColor Yellow
        Write-Host '     npx bmad-plus install --yes --force' -ForegroundColor DarkGray
        Write-Host '  0. Retour' -ForegroundColor DarkGray
        Write-Host ''

        $choice = Read-Host '  Choix'
        Write-Host ''
        switch ($choice) {
            '1' { Write-Ok 'Mise a jour BMAD+ (latest)...'; & npx bmad-plus@latest install --yes }
            '2' { Write-Ok 'Reinstallation BMAD+ (tous les packs)...'; & npx bmad-plus install --yes --packs all }
            '3' {
                $ver = Read-Host '  Version (ex: 0.7.5)'
                if ($ver) {
                    Write-Ok "Installation BMAD+ v$ver..."
                    & npx "bmad-plus@$ver" install --yes
                }
            }
            '4' { Write-Warn 'Reinitialisation BMAD+...'; & npx bmad-plus install --yes --force }
            '0' { return }
            default { Write-Warn 'Choix invalide.' }
        }
    } else {
        # --- Menu BMAD+ non installe : focus installation ---
        Write-Host '  1. Installation complete silencieuse (tous les packs)' -ForegroundColor Cyan
        Write-Host '     npx bmad-plus install --yes --packs all' -ForegroundColor DarkGray
        Write-Host '  2. Installation interactive (choisir les packs)' -ForegroundColor Cyan
        Write-Host '     npx bmad-plus install' -ForegroundColor DarkGray
        Write-Host '  3. Installation derniere version (latest)' -ForegroundColor Cyan
        Write-Host '     npx bmad-plus@latest install --yes' -ForegroundColor DarkGray
        Write-Host '  0. Retour' -ForegroundColor DarkGray
        Write-Host ''

        $choice = Read-Host '  Choix'
        Write-Host ''
        switch ($choice) {
            '1' { Write-Ok 'Installation BMAD+ (complete, silencieuse)...'; & npx bmad-plus install --yes --packs all }
            '2' { Write-Ok 'Installation BMAD+ (interactive)...'; & npx bmad-plus install }
            '3' { Write-Ok 'Installation BMAD+ (latest)...'; & npx bmad-plus@latest install --yes }
            '0' { return }
            default { Write-Warn 'Choix invalide.' }
        }
    }
}

# ── OpenRouter Models ─────────────────────────────────────────────────────────

function Show-OpenRouterMenu {
    param([object[]]$Profiles)

    Write-Host ''
    Write-Info 'OpenRouter -- Decouvrir et ajouter des modeles'
    Write-Host ('-' * 58) -ForegroundColor DarkGray
    Write-Host ''
    Write-Host '  Fusion est deja integre (3 profils).'
    Write-Host '  Pour ajouter d''autres modeles OpenRouter :'
    Write-Host ''
    Write-Host '  1. Voir les 300+ modeles : https://openrouter.ai/models'
    Write-Host '  2. Profils existants dans :'
    Write-Host ("     {0}" -f $ProfilesDir) -ForegroundColor DarkGray
    Write-Host ''
    Write-Host '  Exemple de profil .env personnalise :' -ForegroundColor Yellow
    Write-Host '  ----'
    Write-Host '  PROFILE_ID=claude-monmodele'
    Write-Host '  SHORTCUT=or-mymodel'
    Write-Host '  TOOL=claude'
    Write-Host '  DISPLAY_NAME=Mon Modele via OpenRouter'
    Write-Host '  ORDER=50'
    Write-Host '  COMMAND=claude'
    Write-Host '  CLEAR_ENV=true'
    Write-Host '  REQUIRED_SECRETS=OPENROUTER_API_KEY'
    Write-Host '  OPENROUTER_API_KEY=PASTE_OPENROUTER_API_KEY_HERE'
    Write-Host '  ANTHROPIC_BASE_URL=https://openrouter.ai/api'
    Write-Host '  ANTHROPIC_MODEL=fournisseur/modele'
    Write-Host '  ANTHROPIC_AUTH_TOKEN=%OPENROUTER_API_KEY%'
    Write-Host '  ANTHROPIC_API_KEY='
    Write-Host '  ----'
    Write-Host ''
    Write-Host '  Modeles recommandes (slug a copier) :' -ForegroundColor Cyan
    Write-Host '    openrouter/fusion           - Panel multi-modele (deja installe)'
    Write-Host '    deepseek/deepseek-v4-pro    - DeepSeek V4 Pro'
    Write-Host '    anthropic/claude-sonnet-4.6  - Claude Sonnet 4.6'
    Write-Host '    openai/gpt-5.5              - GPT-5.5'
    Write-Host '    minimax/minimax-m3          - MiniMax M3 (populaire)'
    Write-Host '    qwen/qwen3.7-plus           - Qwen 3.7 Plus'
    Write-Host '    google/gemini-3.5-flash     - Gemini 3.5 Flash'
    Write-Host '    x-ai/grok-4.3               - Grok 4.3'
    Write-Host '    nvidia/nemotron-3-ultra     - Nemotron 3 Ultra (GRATUIT)'
    Write-Host '    openrouter/owl-alpha        - Owl Alpha (GRATUIT)'
    Write-Host ''
    Write-Host ('-' * 58) -ForegroundColor DarkGray

    Write-Host '  Ajout rapide interactif :' -ForegroundColor Cyan
    Write-Host ''
    $name = Read-Host '  Nom du modele (ex: DeepSeek V4 Pro)'
    if ($name) {
        $slug = Read-Host '  Slug OpenRouter (ex: deepseek/deepseek-v4-pro)'
        if ($slug) {
            $toolChoice = Read-Host '  CLI (claude/codex/opencode) [claude]'
            if (-not $toolChoice) { $toolChoice = 'claude' }
            New-OpenRouterProfile -DisplayName $name -ModelSlug $slug -Tool $toolChoice
            Write-Host ''
            Write-Host "  Relancer le menu pour voir le nouveau profil !" -ForegroundColor Yellow
        }
    }
    Write-Host ''
    $null = Read-Host '  Entree pour revenir'
}

function New-OpenRouterProfile {
    param($DisplayName, $ModelSlug, $Tool)
    $shortcut = "or-" + ($DisplayName -replace '[^a-zA-Z0-9]','').ToLower()
    if ($shortcut.Length -gt 12) { $shortcut = $shortcut.Substring(0, 12) }
    $fileName = "99-" + $shortcut + ".env"
    $filePath = Join-Path $ProfilesDir $fileName

    $isAnthropic = $Tool -eq 'claude'

    # Construire les lignes specifiques a l'outil
    $toolLines = if ($isAnthropic) {
        # /api sans /v1 : Claude Code ajoute automatiquement /v1/messages
        "ANTHROPIC_AUTH_TOKEN=%OPENROUTER_API_KEY%`r`nANTHROPIC_BASE_URL=https://openrouter.ai/api`r`nANTHROPIC_MODEL=$ModelSlug`r`nANTHROPIC_API_KEY="
    } elseif ($Tool -eq 'codex') {
        "OPENAI_API_KEY=%OPENROUTER_API_KEY%`r`nOPENAI_BASE_URL=https://openrouter.ai/api/v1"
    } else {
        # OpenCode : OPENROUTER_API_KEY est la variable native, pas de reference
        ""
    }

    $content = @"
PROFILE_ID=$shortcut
SHORTCUT=$shortcut
TOOL=$Tool
TOOL_LABEL=$Tool
DISPLAY_NAME=$DisplayName (via OR)
DESCRIPTION=OpenRouter: $ModelSlug
ORDER=50
COMMAND=$Tool
CLEAR_ENV=true
REQUIRED_SECRETS=OPENROUTER_API_KEY
OPENROUTER_API_KEY=PASTE_OPENROUTER_API_KEY_HERE
$toolLines
"@
    # Nettoyer les trailing whitespaces et normaliser les fins de ligne
    $content = ($content -replace '[ \t]+$', '') -replace '\r?\n', "`r`n"
    $content = $content.Trim() + "`r`n"
    Set-Content -LiteralPath $filePath -Value $content -Encoding UTF8
    Write-Ok "Profil cree : $DisplayName [$shortcut] -> $fileName"
    Write-Host "  Configurer la cle : multiai config -> OpenRouter"
}

# ── Menu principal ─────────────────────────────────────────────────────────────

function Show-TopMenu {
    Write-Host ''
    Write-Info ("Laurent ROCHETTA's MultiAI (AI Code CLI Router) v{0}" -f $RouterVersion)
    Write-Host ('-' * 58) -ForegroundColor DarkGray
    Write-Host ''
    Write-Host '1. Lancer'                                  -ForegroundColor Cyan
    Write-Host '2. Configurer les cles API'                 -ForegroundColor Cyan
    Write-Host '3. BMAD+ -- Gestion du framework'           -ForegroundColor Cyan
    Write-Host '4. OpenRouter -- Modeles disponibles'       -ForegroundColor Cyan
    Write-Host ''
    return (Read-Host 'Choix')
}

function Write-ErrorLog {
    param([string]$Message, [string]$Level = 'ERROR')
    try {
        $logDir = Join-Path ([Environment]::GetFolderPath('LocalApplicationData')) 'multiai'
        if (-not (Test-Path $logDir)) { New-Item -ItemType Directory -Force -Path $logDir | Out-Null }
        $logFile = Join-Path $logDir 'error.log'
        $line = "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') [$Level] $Message"
        Add-Content -LiteralPath $logFile -Value $line -Encoding UTF8
    } catch { }
}

# ── Point d entree ─────────────────────────────────────────────────────────────
# Codes de sortie :
#   0 = Succes
#   1 = Erreur utilisateur (profil introuvable, CLI manquant, cle manquante)
#   2 = Erreur configuration (fichier .env corrompu, profil invalide)
#   3 = Erreur systeme (permissions, dossier inaccessible)
#   4 = Erreur processus enfant (CLI crash, signal)

trap {
    $msg = "$($_.Exception.GetType().Name): $($_.Exception.Message)"
    Write-ErrorLog -Message $msg -Level 'ERROR'
    Write-Bad $_.Exception.Message
    exit 1  # Erreur utilisateur
}

$profiles = @(Get-Profiles)

if ($List)      { Show-Profiles   -Profiles $profiles -Json:$Json; exit 0 }

# ── Mode interactif avec boucle (retour au menu possible) ─────────────────

while ($true) {
    if ($Configure) {
        Show-ConfigMenu -Profiles $profiles
        exit 0  # -Configure est un one-shot
    }
    if ($Bmad) {
        Show-BmadMenu
        exit 0  # -Bmad est un one-shot
    }

    $selected = $null
    if (-not [string]::IsNullOrWhiteSpace($Profile)) {
        $selected = Find-Profile -Profiles $profiles -ProfileName $Profile
        break  # Direct launch, exit loop
    }

    # Menu principal (re-affiche si retour)
    $toolChoice = $Tool
    if ([string]::IsNullOrWhiteSpace($toolChoice)) {
        $menuChoice = Show-TopMenu
        switch ($menuChoice) {
            '2' { Show-ConfigMenu -Profiles $profiles ; continue }  # retour menu apres config
            '3' { Show-BmadMenu ; continue }                        # retour menu apres BMAD
            '4' { Show-OpenRouterMenu -Profiles $profiles ; continue }  # retour menu apres OpenRouter
            '1' { }  # continue vers selection outil
            default { continue }
        }
        $toolChoice = Select-Tool -Profiles $profiles
        if ($null -eq $toolChoice) { continue }  # 0 = retour au menu
    }

    $selected = Select-Profile -Profiles $profiles -ToolFilter $toolChoice
    if ($null -eq $selected) {
        $toolChoice = $null  # reset pour revenir selection outil
        continue
    }
    break  # Profil selectionne, on lance
}

Apply-ProfileEnv     -Selected $selected
Test-RequiredSecrets -Selected $selected

# Journaliser le hash d'integrite pour tracabilite
$routerHash = Get-RouterHash
if ($routerHash) {
    Write-ErrorLog -Message "Router hash: $routerHash - Profile: $($selected.Shortcut) - Command: $($selected.Command)" -Level 'INFO'
}

if ($DryRun) {
    Write-Host ''
    Write-Info "[DRY RUN] Simulation sans lancement"
    Show-EffectiveEnv -Selected $selected
    Write-Ok "[DRY RUN] Commande qui serait lancee : $($selected.Command) $($launchArgs -join ' ')"
    exit 0 }  # Succes

if ($ShowEnv)  { Show-EffectiveEnv -Selected $selected -Json:$Json }
if ($NoLaunch) { Write-Ok 'NoLaunch actif : rien lance.'; exit 0 }  # Succes

$command    = $selected.Command
# Whitelist de securite : seuls les binaires connus peuvent etre lances
$allowedCommands = @('claude', 'codex', 'opencode')
if ($command -notin $allowedCommands) {
    if ($ExtraArgs -and $ExtraArgs -contains '-AllowCustomCommand') {
        Write-Warn "Commande custom autorisee via -AllowCustomCommand : $command"
    } else {
        throw "Commande non autorisee : '$command'. Seuls ${allowedCommands} sont permis. Utilise -AllowCustomCommand pour autoriser une commande personnalisee."
    }
}
$launchArgs = @(Split-ArgsSimple -ArgString $selected.Args)
if ($ExtraArgs -and $ExtraArgs.Count -gt 0) {
    if ($ExtraArgs[0] -eq '--') { $ExtraArgs = @($ExtraArgs | Select-Object -Skip 1) }
    $launchArgs += $ExtraArgs
}

Assert-CommandExists -Command $command
Write-Host ''
Write-Ok ("Lancement : {0} {1}" -f $command, ($launchArgs -join ' '))
Write-Host ("Dossier courant : {0}" -f (Get-Location)) -ForegroundColor DarkGray
Write-Host ''

& $command @launchArgs
$exitCode = if ($LASTEXITCODE -ne $null) { $LASTEXITCODE } else { 4 }; exit $exitCode
