# Reddit post — r/golang

> Target: https://www.reddit.com/r/golang/submit

---

## Title

**multiai: a Go-based CLI router for Claude Code, Codex CLI, and OpenCode — with AES-256-GCM credential isolation**

## Post body

I built a CLI tool in Go that routes credentials securely between AI coding agents (Claude Code, Codex CLI, OpenCode) and 13 different API providers.

**Why Go?** Single binary, no runtime deps, cross-compile to all platforms. The binary has exactly one external dependency (yaml.v3).

**Architecture highlights:**
- Process-level environment isolation via whitelist (~30 system vars survive, everything else is scoped per-session)
- Credential store encrypted with AES-256-GCM at `~/.config/multiai/secrets`
- 37 pre-configured profiles (YAML-driven catalog, not hardcoded)
- Fallback chains: `FALLBACK=ds,dsf,or-fusion` auto-retries on profile failure
- Data-driven provider catalog in `providers.yaml` — adding a new provider is a config change, not a code change

**What it does:**
```
$ multiai launch -p ds          # Launch Claude Code with DeepSeek V4 Pro
$ multiai launch --dry-run -p or-fusion  # Preview env before launching
$ multiai list --json           # All profiles in JSON
$ multiai config                # Encrypted credential wizard
```

**Stats:**
- 1 external dep, 0 CVEs
- 13 Go packages, all tested
- GoReleaser + Cosign keyless signing for releases
- Homebrew, Scoop, npm, and direct `go install` distributions

The credential isolation approach was the hardest part — I'd love feedback on the process env whitelist strategy vs. other sandboxing approaches.

Repo: https://github.com/lrochetta/multiai
