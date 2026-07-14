[CmdletBinding()]
param(
    [ValidateSet('Apply', 'Plan')]
    [string]$Mode = 'Apply'
)

$ErrorActionPreference = 'Stop'
$maxEnvironmentLength = 32767
[Console]::OutputEncoding = [Text.UTF8Encoding]::new($false)

function Normalize-PathEntry {
    param([AllowNull()][string]$Value)

    if ([string]::IsNullOrWhiteSpace($Value)) {
        return $null
    }

    $candidate = $Value.Trim()
    if ($candidate.Length -ge 2 -and $candidate[0] -eq '"' -and $candidate[$candidate.Length - 1] -eq '"') {
        $candidate = $candidate.Substring(1, $candidate.Length - 2)
    }
    $candidate = [Environment]::ExpandEnvironmentVariables($candidate)

    try {
        if ([IO.Path]::IsPathRooted($candidate)) {
            $candidate = [IO.Path]::GetFullPath($candidate)
        }
    }
    catch {
        # Preserve unusual existing entries. The install target is validated separately.
    }

    $root = [IO.Path]::GetPathRoot($candidate)
    while ($candidate.Length -gt $root.Length -and ($candidate.EndsWith('\') -or $candidate.EndsWith('/'))) {
        $candidate = $candidate.Substring(0, $candidate.Length - 1)
    }
    return $candidate
}

function Test-PathListContains {
    param(
        [AllowNull()][string]$PathValue,
        [string]$NormalizedTarget
    )

    if ([string]::IsNullOrWhiteSpace($PathValue)) {
        return $false
    }

    foreach ($entry in ($PathValue -split ';')) {
        $normalized = Normalize-PathEntry $entry
        if ($null -ne $normalized -and [string]::Equals(
            $normalized,
            $NormalizedTarget,
            [StringComparison]::OrdinalIgnoreCase
        )) {
            return $true
        }
    }
    return $false
}

function Add-PathEntry {
    param(
        [AllowNull()][string]$PathValue,
        [string]$Entry
    )

    if ([string]::IsNullOrWhiteSpace($PathValue)) {
        return $Entry
    }
    if ($PathValue.EndsWith(';')) {
        return $PathValue + $Entry
    }
    return $PathValue + ';' + $Entry
}

function Get-PersistentPaths {
    if ($Mode -eq 'Plan') {
        return @{
            User = $env:MULTIAI_TEST_USER_PATH
            Machine = $env:MULTIAI_TEST_MACHINE_PATH
        }
    }
    return @{
        User = [Environment]::GetEnvironmentVariable('Path', 'User')
        Machine = [Environment]::GetEnvironmentVariable('Path', 'Machine')
    }
}

function Join-PersistentPath {
    param(
        [AllowNull()][string]$MachinePath,
        [AllowNull()][string]$UserPath
    )

    $machine = [Environment]::ExpandEnvironmentVariables([string]$MachinePath)
    $user = [Environment]::ExpandEnvironmentVariables([string]$UserPath)
    if ([string]::IsNullOrWhiteSpace($machine)) {
        return $user
    }
    if ([string]::IsNullOrWhiteSpace($user)) {
        return $machine
    }
    if ($machine.EndsWith(';')) {
        return $machine + $user
    }
    return $machine + ';' + $user
}

function Find-FirstCommandShim {
    param(
        [AllowNull()][string]$EffectivePath,
        [string]$CommandName
    )

    $directories = @((Get-Location).Path)
    if (-not [string]::IsNullOrWhiteSpace($EffectivePath)) {
        $directories += $EffectivePath -split ';'
    }
    foreach ($directory in $directories) {
        $normalizedDirectory = Normalize-PathEntry $directory
        if ([string]::IsNullOrWhiteSpace($normalizedDirectory)) {
            continue
        }
        try {
            $candidate = Join-Path $normalizedDirectory $CommandName
            if (Test-Path -LiteralPath $candidate -PathType Leaf) {
                return [IO.Path]::GetFullPath($candidate)
            }
        }
        catch {
            # Invalid or unavailable PATH entries are ignored just like command lookup.
        }
    }
    return $null
}

function Write-ApplyResult {
    param(
        [string]$Status,
        [hashtable]$Paths
    )

    $effectivePath = Join-PersistentPath $Paths.Machine $Paths.User
    [pscustomobject]@{
        status = $Status
        effectivePath = $effectivePath
        resolvedShim = Find-FirstCommandShim $effectivePath 'multiai.cmd'
    } | ConvertTo-Json -Compress
}

$target = $env:MULTIAI_PATH_ENTRY
if ([string]::IsNullOrWhiteSpace($target)) {
    throw 'MULTIAI_PATH_ENTRY is required.'
}
$target = $target.Trim()
if ($target.IndexOfAny(@([char]0, [char]13, [char]10, [char]';')) -ge 0) {
    throw 'The npm prefix contains a character that cannot be stored safely in PATH.'
}
if ($target -notmatch '^[A-Za-z]:[\\/]') {
    throw 'The npm prefix must be an absolute local drive path.'
}
$target = [IO.Path]::GetFullPath($target)

$commandShim = Join-Path $target 'multiai.cmd'
if (-not (Test-Path -LiteralPath $commandShim -PathType Leaf)) {
    throw "The npm command shim does not exist: $commandShim"
}

$normalizedTarget = Normalize-PathEntry $target
$paths = Get-PersistentPaths
if (Test-PathListContains $paths.Machine $normalizedTarget) {
    Write-ApplyResult 'present:machine' $paths
    exit 0
}
if (Test-PathListContains $paths.User $normalizedTarget) {
    Write-ApplyResult 'present:user' $paths
    exit 0
}

if ($Mode -eq 'Plan') {
    $planned = Add-PathEntry $paths.User $target
    if ($planned.Length -ge $maxEnvironmentLength) {
        throw 'The resulting user PATH would exceed the Windows environment limit.'
    }
    Write-Output ("planned`t" + $planned)
    exit 0
}

$mutex = [Threading.Mutex]::new($false, 'Local\multiai-user-path-update')
$acquired = $false
try {
    try {
        $acquired = $mutex.WaitOne([TimeSpan]::FromSeconds(15))
    }
    catch [Threading.AbandonedMutexException] {
        $acquired = $true
    }
    if (-not $acquired) {
        throw 'Timed out while another multiai installer was updating PATH.'
    }

    # Re-read after acquiring the lock so concurrent installers remain idempotent.
    $paths = Get-PersistentPaths
    if (Test-PathListContains $paths.Machine $normalizedTarget) {
        Write-ApplyResult 'present:machine' $paths
        exit 0
    }
    if (Test-PathListContains $paths.User $normalizedTarget) {
        Write-ApplyResult 'present:user' $paths
        exit 0
    }

    $updated = Add-PathEntry $paths.User $target
    if ($updated.Length -ge $maxEnvironmentLength) {
        throw 'The resulting user PATH would exceed the Windows environment limit.'
    }

    # User scope requires no administrator rights. .NET also broadcasts
    # WM_SETTINGCHANGE so new terminals launched by Explorer inherit the value.
    [Environment]::SetEnvironmentVariable('Path', $updated, 'User')

    $persisted = [Environment]::GetEnvironmentVariable('Path', 'User')
    if (-not (Test-PathListContains $persisted $normalizedTarget)) {
        throw 'The user PATH update could not be verified after writing it.'
    }
    $paths.User = $persisted
    Write-ApplyResult 'added' $paths
}
finally {
    if ($acquired) {
        $mutex.ReleaseMutex()
    }
    $mutex.Dispose()
}
