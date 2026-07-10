# Newsletters — contact list & submission text

---

## 1. Go Weekly

**Website:** https://goweekly.com/
**Submit:** https://goweekly.com/submit
**Format:** Short description + link
**Audience:** Go developers, ~25k subscribers

### Submission text

**multiai — secure multi-AI CLI router built in Go**

multiai is a Go-based CLI that securely routes credentials between AI coding agents (Claude Code, Codex CLI, OpenCode) and 13+ API providers. It features process-level environment isolation via whitelist, AES-256-GCM encrypted credential storage, 37 pre-configured profiles, and fallback chains. Built with zero runtime dependencies (only yaml.v3), cross-compiled for Windows/macOS/Linux, and signed with Cosign keyless.

`go install github.com/lrochetta/multiai/multiai-go/cmd/multiai@latest`

https://github.com/lrochetta/multiai

---

## 2. Console.dev

**Website:** https://console.dev/
**Submit:** https://console.dev/submit-tool/
**Format:** Tool review format (name, tagline, description, category, pricing, links)
**Audience:** Developers discovering tools, ~40k subscribers

### Submission

**Tool name:** multiai
**Tagline:** Secure multi-AI CLI router — one tool to launch Claude Code, Codex CLI, and OpenCode with any provider
**Category:** CLI / Developer Tools / AI
**Pricing:** Free & open source (MIT)
**Description:**
multiai is a CLI router that sits between developers and their AI coding agents. Instead of manually managing environment variables for each combination of CLI (Claude Code, Codex CLI, OpenCode) and API provider (Anthropic, DeepSeek, OpenAI, OpenRouter, Z.ai, MiniMax, Qwen, and 7+ more), multiai injects the correct credentials into the current process with strict isolation. It features an AES-256-GCM encrypted credential store, process-level environment whitelisting, 37 pre-configured profiles, fallback chains, and OpenRouter model discovery. Built in Go with minimal dependencies, distributed as a single binary via npm, Homebrew, Scoop, and Go install.
**Links:**
- GitHub: https://github.com/lrochetta/multiai
- npm: `npx multiai install`
- Docs: README on GitHub

---

## 3. TLDR (tldr.tech)

**Website:** https://tldr.tech/
**Submit:** https://tldr.tech/submit
**Format:** One-liner + link (very concise)
**Audience:** General developers, ~500k+ subscribers

### Submission text

**multiai** (https://github.com/lrochetta/multiai) — An open-source CLI router that securely manages API keys across Claude Code, Codex CLI, and OpenCode. Supports 13 providers, 37 profiles, AES-256-GCM encrypted storage, and process-level credential isolation. Go binary, MIT license. Install via `npx multiai install` or `go install`.
