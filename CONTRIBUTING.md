# Contributing to multiai

Thanks for considering contributing to multiai! 🚀

---

## Table of Contents

- [Environment](#environment)
- [Conventions](#conventions)
- [Development Workflow](#development-workflow)
- [Process](#process)
- [Release Guide](#release-guide)
- [Questions?](#questions)

---

## Environment

### Prerequisites

- **Go** 1.22+
- **Make** (optional, for convenience targets)
- **Git** with signed commits recommended

### Setup

```bash
# Clone the repository
git clone https://github.com/lrochetta/multiai.git
cd multiai/multiai-go

# Build the binary
go build ./cmd/multiai/

# Run all tests
go test ./...

# Run lint checks
gofmt -l .          # check formatting
go vet ./...         # static analysis
```

> **Note:** The Go module lives in `multiai-go/`. All Go commands should be run from that directory.

---

## Conventions

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

| Prefix       | Usage                     |
|--------------|---------------------------|
| `feat:`      | New feature               |
| `fix:`       | Bug fix                   |
| `docs:`      | Documentation             |
| `security:`  | Security fix              |
| `refactor:`  | Code restructuring        |
| `test:`      | Adding or updating tests  |
| `chore:`     | Tooling, CI, dependencies |
| `perf:`      | Performance improvement   |

Example:

```
feat: add Azure OpenAI provider with regional fallback
```

### Code Style

- **Go**: run `gofmt` before committing — CI rejects unformatted code.
- Run `go vet ./...` — zero warnings required.
- All tests must pass before opening a PR.
- Follow idiomatic Go conventions (see [Effective Go](https://go.dev/doc/effective_go)).
- Code and technical documentation in English; user-facing strings in French.

### Testing

```bash
# Run all tests with race detection and coverage
go test -race -coverprofile=coverage.out -covermode=atomic ./...

# Run benchmarks
go test -bench=. -benchmem ./tests/

# Security scanning
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
go run github.com/securego/gosec/v2/cmd/gosec@latest -exclude=G104 ./...
```

### Branch Naming

```
<type>/<short-description>
```

Examples: `feat/azure-provider`, `fix/empty-fallback`, `docs/api-readme`.

---

## Development Workflow

1. **Fork** the repository on GitHub.
2. **Create a branch** from `master`:
   ```bash
   git checkout -b feat/my-feature
   ```
3. **Make your changes** — keep them focused and atomic.
4. **Run checks** locally:
   ```bash
   gofmt -l -d .
   go vet ./...
   go test -race ./...
   ```
5. **Commit** using Conventional Commits:
   ```bash
   git commit -m "feat: add Azure OpenAI provider"
   ```
6. **Push** your branch:
   ```bash
   git push origin feat/my-feature
   ```
7. **Open a Pull Request** against `master`.

---

## Process

### Pull Request Checklist

Before submitting, ensure:

- [ ] `gofmt` produces no diffs
- [ ] `go vet ./...` passes with zero warnings
- [ ] `go test -race ./...` passes
- [ ] New code includes tests where applicable
- [ ] CHANGELOG.md is updated under `[Unreleased]`
- [ ] `README.md` is updated if user-facing behavior changed
- [ ] Security implications are considered (credential handling, env isolation)

### Review

- At least one maintainer review required.
- CI must be green (lint → test → security → build).
- Squash merge preferred — keep history clean.

---

## Release Guide

Releases are automated via **GoReleaser** with Cosign keyless signing and GitHub attestations.

### Steps

1. **Update version** in `multiai-go/cmd/multiai/main.go` (variable `version`).
2. **Update CHANGELOG.md** with the new version section.
3. **Commit and tag**:
   ```bash
   git commit -m "chore: bump version v0.x.y"
   git tag v0.x.y
   git push origin master --tags
   ```
4. **GitHub Action** triggers automatically:
   - GoReleaser builds for Windows, macOS (Intel + Apple Silicon), Linux (amd64 + arm64)
   - Checksums + Cosign keyless signatures + GitHub build provenance attachements
   - GitHub Release created with all artifacts
5. **Manual npm publish** (if applicable):
   ```bash
   cd multiai-go/packaging/npm
   npm publish
   ```
6. **Homebrew / Scoop** (first time only):
   - Submit PR to `lrochetta/homebrew-tap` and `lrochetta/scoop-bucket`.

### Configuration

| File | Purpose |
|------|---------|
| `multiai-go/.goreleaser.yaml` | GoReleaser build matrix and release config |
| `.github/workflows/ci.yml` | CI pipeline (lint → test → security → build → smoke) |
| `.github/workflows/release.yml` | Release pipeline (GoReleaser + Cosign + attestations) |
| `multiai-go/packaging/npm/` | npm distribution files |

---

## Questions?

Open a [Discussion](https://github.com/lrochetta/multiai/discussions) or an [Issue](https://github.com/lrochetta/multiai/issues).

For security vulnerabilities, see [SECURITY.md](SECURITY.md) (if present) or email the maintainer directly.
