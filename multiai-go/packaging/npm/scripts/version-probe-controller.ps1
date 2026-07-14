[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][string]$BinaryPath,
    [ValidateRange(1000, 120000)][int]$TimeoutMilliseconds = 20000
)

$ErrorActionPreference = 'Stop'
$binary = [IO.Path]::GetFullPath($BinaryPath)
if (-not (Test-Path -LiteralPath $binary -PathType Leaf)) {
    throw "Binary not found: $binary"
}

$inner = $null
try {
    $encoded = [Convert]::ToBase64String([Text.Encoding]::Unicode.GetBytes(
        '& $env:MULTIAI_PROBE_BINARY --version; exit $LASTEXITCODE'
    ))
    $env:MULTIAI_PROBE_BINARY = $binary

    $systemDirectory = [Environment]::SystemDirectory
    $powershell = Join-Path $systemDirectory 'WindowsPowerShell\v1.0\powershell.exe'
    if (-not (Test-Path -LiteralPath $powershell -PathType Leaf)) {
        throw 'Trusted Windows PowerShell was not found.'
    }

    # Avoid Start-Process: Windows PowerShell 5 can fail while normalizing an
    # inherited environment that contains both Path and PATH entries.
    $startInfo = [Diagnostics.ProcessStartInfo]::new()
    $startInfo.FileName = $powershell
    $startInfo.Arguments = "-NoProfile -NonInteractive -EncodedCommand $encoded"
    $startInfo.UseShellExecute = $false
    $startInfo.CreateNoWindow = $true
    $startInfo.RedirectStandardOutput = $true
    $startInfo.RedirectStandardError = $true

    $inner = [Diagnostics.Process]::new()
    $inner.StartInfo = $startInfo
    if (-not $inner.Start()) {
        throw 'Could not start the version probe worker.'
    }

    if (-not $inner.WaitForExit($TimeoutMilliseconds)) {
        # Kill the complete worker tree. A child which starts and then hangs
        # must not survive npm or the multiai shim as an orphan process.
        $taskkill = Join-Path $systemDirectory 'taskkill.exe'
        if (Test-Path -LiteralPath $taskkill -PathType Leaf) {
            $previousErrorAction = $ErrorActionPreference
            try {
                # taskkill writes access failures to the native error stream;
                # never let that bypass the fallback cleanup below.
                $ErrorActionPreference = 'Continue'
                & $taskkill /PID $inner.Id /T /F 2>$null | Out-Null
            }
            catch { }
            finally { $ErrorActionPreference = $previousErrorAction }
        }
        if (-not $inner.HasExited) {
            # Stop direct children first when taskkill is denied by endpoint
            # security, then terminate the PowerShell worker itself.
            try {
                Get-CimInstance Win32_Process -Filter "ParentProcessId = $($inner.Id)" |
                    ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }
            }
            catch { }
        }
        if (-not $inner.HasExited) {
            $inner.Kill()
        }
        $inner.WaitForExit()
        [Console]::Error.WriteLine("version probe timed out after $TimeoutMilliseconds ms")
        exit 124
    }

    $out = $inner.StandardOutput.ReadToEnd()
    $err = $inner.StandardError.ReadToEnd()
    if ($out) { [Console]::Out.Write($out) }
    if ($err) { [Console]::Error.Write($err) }
    exit $inner.ExitCode
}
finally {
    if ($null -ne $inner) {
        $inner.Dispose()
    }
}
