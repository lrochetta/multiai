# Contributing to multiai

Merci de contribuer ! / Thanks for contributing!

## Setup

### Prerequisites
- Windows: PowerShell 5.1+ (built-in)
- macOS/Linux: [PowerShell Core 7+](https://learn.microsoft.com/powershell/scripting/install/installing-powershell)
- Node.js 16+ (for npm distribution only)
- Pester (PowerShell testing framework)

### Install dev environment

```powershell
# Clone the repo
git clone https://github.com/lrochetta/powerai.git
cd powerai

# Install Pester for testing
Install-Module Pester -Force -SkipPublisherCheck
```

## Testing

```powershell
# Run all tests
Invoke-Pester tests/

# Run specific test file
Invoke-Pester tests/unit/RouterFunctions.Tests.ps1

# With coverage
Invoke-Pester tests/ -CodeCoverage code-router.ps1
```

## Code Style

- PowerShell: Follow [PowerShell Best Practices](https://docs.microsoft.com/powershell/scripting/learn/ps101/09-functions)
  - Use `[CmdletBinding()]` on all functions
  - Use `-LiteralPath` for file operations
  - Use `[ordered]@{}` for ordered hashtables
  - Messages in French, code in English
- Bash: Follow [Google Shell Style Guide](https://google.github.io/styleguide/shellguide.html)
  - Use `set -euo pipefail`
  - Use `[[ ]]` for tests, not `[ ]`

## Pull Requests

1. Run all tests: `Invoke-Pester tests/`
2. Ensure no PSScriptAnalyzer warnings
3. Update CHANGELOG.md under `[Unreleased]`
4. PR titles: `type: description` (e.g. `feat: add new provider`, `fix: export prefix parsing`)

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):
- `feat:` new feature
- `fix:` bug fix
- `docs:` documentation
- `security:` security fix
- `refactor:` code restructuring
- `test:` adding tests

## Questions?

Open an issue on GitHub: https://github.com/lrochetta/powerai/issues
