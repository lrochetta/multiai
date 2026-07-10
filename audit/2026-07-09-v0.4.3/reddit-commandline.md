# Reddit post — r/commandline

> Target: https://www.reddit.com/r/commandline/submit

---

## Title

**multiai: a terminal tool to securely manage API keys for Claude Code, Codex CLI, and OpenCode**

## Post body

I spend a lot of time in the terminal switching between AI coding CLIs (Claude Code, Codex CLI, OpenCode). The biggest pain point was managing API keys for different providers — one wrong export and you're leaking credentials.

**multiai** is a CLI router that sits between you and these tools. You pick a tool + profile, it injects the right environment, and launches the CLI — all in the current process.

**Install in one line:**
```bash
npx multiai install
```

**Usage:**
```bash
multiai                     # Interactive menu with colors
multiai launch -p ds        # Launch Claude Code with DeepSeek V4 Pro
multiai launch -p codex55   # Launch Codex CLI with GPT-5.5
multiai config              # Set up API keys (encrypted storage)
multiai models              # Browse 300+ OpenRouter models
multiai completion bash     # Shell completion
```

**What's under the hood:**
- Go binary, single file, no runtime deps
- AES-256-GCM encrypted credential store in `~/.config/multiai/secrets`
- 37 profiles across 13 providers — Anthropic, DeepSeek, OpenAI, OpenRouter (Fusion), Z.ai, MiniMax, and more
- Fallback chains: auto-retry on a different profile if one fails
- Full shell completion: bash, zsh, fish, PowerShell
- Menus with green/yellow/gray color coding (config status), dark terminal friendly

**Why I built it instead of using direnv or dotenv:**
Those tools are great for project-level env management, but they don't handle the multi-provider, multi-CLI case well. multiai is purpose-built for AI coding agents — it knows which env vars each CLI needs and isolates them properly.

https://github.com/lrochetta/multiai

Feedback welcome on the terminal UX and any features you'd like to see.
