# Tests unitaires pour les fonctions pures de code-router.ps1
# Usage: Invoke-Pester tests/unit/RouterFunctions.Tests.ps1

$ScriptRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$RouterPath = Join-Path (Join-Path (Join-Path $ScriptRoot '..') '..') 'code-router.ps1'
. $RouterPath

Describe 'Test-IsPlaceholder' {
    It 'Returns true for null or whitespace' {
        Test-IsPlaceholder -Value $null | Should -BeTrue
        Test-IsPlaceholder -Value '' | Should -BeTrue
        Test-IsPlaceholder -Value '   ' | Should -BeTrue
    }

    It 'Returns true for PASTE_ prefix' {
        Test-IsPlaceholder -Value 'PASTE_ANTHROPIC_API_KEY_HERE' | Should -BeTrue
    }

    It 'Returns true for YOUR_ prefix' {
        Test-IsPlaceholder -Value 'YOUR_API_KEY' | Should -BeTrue
    }

    It 'Returns true for sk-xxxx' {
        Test-IsPlaceholder -Value 'sk-xxxx' | Should -BeTrue
    }

    It 'Returns true for TODO' {
        Test-IsPlaceholder -Value 'TODO' | Should -BeTrue
    }

    It 'Returns true for _HERE suffix' {
        Test-IsPlaceholder -Value 'PASTE_YOUR_KEY_HERE' | Should -BeTrue
    }

    It 'Returns true for _ICI suffix' {
        Test-IsPlaceholder -Value 'TA_CLE_ICI' | Should -BeTrue
    }

    It 'Returns false for a real key' {
        Test-IsPlaceholder -Value 'sk-ant-api-03-abc123def456' | Should -BeFalse
    }

    It 'Returns false for a real token' {
        Test-IsPlaceholder -Value 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.xxx' | Should -BeFalse
    }
}

Describe 'Read-DotEnvFile' {
    It 'Parses standard key=value' {
        $tmp = New-TemporaryFile
        "KEY1=value1`nKEY2=value2" | Set-Content $tmp -Encoding UTF8
        $result = Read-DotEnvFile -Path $tmp.FullName
        $result['KEY1'] | Should -Be 'value1'
        $result['KEY2'] | Should -Be 'value2'
        Remove-Item $tmp
    }

    It 'Supports export prefix (Unix format)' {
        $tmp = New-TemporaryFile
        "export KEY1=value1`nexport KEY2=value2" | Set-Content $tmp -Encoding UTF8
        $result = Read-DotEnvFile -Path $tmp.FullName
        $result['KEY1'] | Should -Be 'value1'
        $result['KEY2'] | Should -Be 'value2'
        Remove-Item $tmp
    }

    It 'Ignores comments' {
        $tmp = New-TemporaryFile
        "# This is a comment`nKEY1=value1`n# Another comment" | Set-Content $tmp -Encoding UTF8
        $result = Read-DotEnvFile -Path $tmp.FullName
        $result.Count | Should -Be 1
        $result['KEY1'] | Should -Be 'value1'
        Remove-Item $tmp
    }

    It 'Handles double-quoted values' {
        $tmp = New-TemporaryFile
        'KEY1="value with spaces"' | Set-Content $tmp -Encoding UTF8
        $result = Read-DotEnvFile -Path $tmp.FullName
        $result['KEY1'] | Should -Be 'value with spaces'
        Remove-Item $tmp
    }

    It 'Handles single-quoted values' {
        $tmp = New-TemporaryFile
        "KEY1='value with spaces'" | Set-Content $tmp -Encoding UTF8
        $result = Read-DotEnvFile -Path $tmp.FullName
        $result['KEY1'] | Should -Be 'value with spaces'
        Remove-Item $tmp
    }

    It 'Ignores empty lines' {
        $tmp = New-TemporaryFile
        "`n`nKEY1=value1`n`n" | Set-Content $tmp -Encoding UTF8
        $result = Read-DotEnvFile -Path $tmp.FullName
        $result.Count | Should -Be 1
        Remove-Item $tmp
    }

    It 'Throws on missing file' {
        { Read-DotEnvFile -Path '/nonexistent/file.env' } | Should -Throw
    }
}

Describe 'Split-ArgsSimple' {
    It 'Returns empty array for null/whitespace' {
        Split-ArgsSimple -ArgString $null | Should -HaveCount 0
        Split-ArgsSimple -ArgString '' | Should -HaveCount 0
        Split-ArgsSimple -ArgString '   ' | Should -HaveCount 0
    }

    It 'Splits on spaces' {
        $result = Split-ArgsSimple -ArgString '-p ds --verbose'
        $result.Count | Should -Be 3
        $result[0] | Should -Be '-p'
        $result[1] | Should -Be 'ds'
        $result[2] | Should -Be '--verbose'
    }

    It 'Preserves double-quoted strings' {
        $result = Split-ArgsSimple -ArgString '-m "deepseek v4 pro" --flag'
        $result.Count | Should -Be 3
        $result[0] | Should -Be '-m'
        $result[1] | Should -Be 'deepseek v4 pro'
        $result[2] | Should -Be '--flag'
    }

    It 'Preserves single-quoted strings' {
        $result = Split-ArgsSimple -ArgString "-m 'deepseek v4 pro' --flag"
        $result.Count | Should -Be 3
        $result[0] | Should -Be '-m'
        $result[1] | Should -Be 'deepseek v4 pro'
        $result[2] | Should -Be '--flag'
    }

    It 'Handles mixed quotes' {
        $result = Split-ArgsSimple -ArgString '--name "Jean Dupont" --city Paris'
        $result.Count | Should -Be 4
        $result[1] | Should -Be 'Jean Dupont'
        $result[3] | Should -Be 'Paris'
    }
}
