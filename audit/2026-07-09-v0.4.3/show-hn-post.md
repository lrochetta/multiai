# Show HN: multiai — a secure multi-AI CLI router

> **Draft for Hacker News Show HN submission**
> Target: `https://news.ycombinator.com/submit`

---

## Title (80 chars max)

**Show HN: multiai — secure multi-AI CLI router for Claude Code, Codex, OpenCode**

## URL

https://github.com/lrochetta/multiai

## Description text

I got tired of juggling API keys across Claude Code, Codex CLI, and OpenCode every time I switched providers. One wrong `export ANTHROPIC_API_KEY` in the wrong terminal and your key leaks into the wrong session. That's why I built multiai.

### The problem
If you use AI coding CLIs, you probably have 5+ API keys sitting in `.env` files or shell history. Each CLI needs different environment variables, and managing them manually is error-prone. A mistyped variable leaks production keys.

### The solution
multiai is a single CLI router that loads the right credentials, for the right CLI, for the right provider — in the current process only. Nothing leaks, nothing persists.

**Key features:**
- **37 pre-configured profiles** across Claude Code (15), Codex CLI (7), and OpenCode (15) — covering Anthropic, DeepSeek, OpenAI, OpenRouter (Fusion), Z.ai, MiniMax, StepFun, Qwen, Kimi, SiliconFlow, MiMo, Requesty, LiteLLM
- **Credential isolation** by process-level environment whitelist (~30 system vars survive, everything else is scoped)
- **AES-256-GCM encrypted store** at `~/.config/multiai/secrets` — keys never touch disk in plaintext
- **Sentinel pattern**: `.env` files never contain real keys
- **Fallback chains**: `FALLBACK=ds,dsf,or-fusion` auto-retries on failure
- **OpenRouter integration**: browse 300+ models, search, compare side-by-side, multi-model Fusion panel with auto-summary
- **Cosign keyless signing**: all releases are verifiably signed

### Tech stack
- **Go** (single binary, ~1 dependency: yaml.v3)
- Cross-platform: Windows, macOS, Linux
- Distribution: npm (Go binary), Homebrew, Scoop, direct Go install
- CI/CD: lint → test (3 OS) → security scan → GoReleaser + Cosign
- BMAD+ score: 8.5/10 from 3-agent audit

### Links
- GitHub: https://github.com/lrochetta/multiai
- npm: `npx multiai install`
- Go install: `go install github.com/lrochetta/multiai/multiai-go/cmd/multiai@latest`
- Docs: included in the README

Happy to answer questions — would love feedback on the isolation approach and the credential store design.
