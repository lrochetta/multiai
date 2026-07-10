# Reddit post — r/programming

> Target: https://www.reddit.com/r/programming/submit

---

## Title

**multiai: a secure CLI router for AI coding agents — manage Claude Code, Codex CLI, and OpenCode with 37 profiles**

## Post body

If you use multiple AI coding CLIs, you know the pain: each one needs different environment variables, and a single `export` mistake can leak credentials across sessions.

**multiai** is an open-source CLI that solves this by acting as a secure router between AI agents and their API providers.

**How it works:**
1. You choose a CLI (Claude Code, Codex CLI, or OpenCode)
2. You choose a profile (37 pre-configured, covering 13 providers)
3. multiai injects the right environment variables into the current process — nothing leaks, nothing persists

**Security model:**
- Process-level environment whitelist: only ~30 system variables survive, everything else is scoped to the session
- AES-256-GCM encrypted credential store on disk
- Sentinel pattern: `.env` files never contain real API keys
- Cosign keyless signature verification for all releases
- npm `prepublishOnly` scans for leaked `.env` files

**Why this matters:**
AI coding tools are proliferating fast. Developers routinely juggle 5+ API keys across multiple CLIs. The current workflow (copy-pasting `.env` files, managing shell exports) is fragile and risky. multiai treats credentials as a security concern from day one.

**Tech:** Go binary, cross-platform (Windows/macOS/Linux), distributed via npm/Homebrew/Scoop/Go install.

**Links:**
- GitHub: https://github.com/lrochetta/multiai
- npm: `npx multiai install`

Would love to hear feedback on the isolation approach — especially from anyone doing similar work with AI agent sandboxing.
