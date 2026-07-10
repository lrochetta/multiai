# Reddit post — r/LocalLLaMA

> Target: https://www.reddit.com/r/LocalLLaMA/submit

---

## Title

**multiai: a router that lets you use Claude Code with any provider (OpenRouter, DeepSeek, Z.ai, Qwen, MiniMax...) — 37 profiles, secure credential isolation**

## Post body

We all know the pain: Claude Code is locked to Anthropic's API by default. But what if you want to run it with DeepSeek V4, or route through OpenRouter to compare models, or use it with a provider that gives better pricing?

I built **multiai** — an open-source CLI router that decouples AI coding agents from their API providers.

**What it does:**
You keep using Claude Code (or Codex CLI, or OpenCode) as your editor agent. multiai sits in front and injects the right environment variables for whatever provider you choose. 37 profiles across 13 providers.

**Providers currently supported:**
| Provider | Claude Code | Codex CLI | OpenCode |
|---|---|---|---|
| Anthropic | co, ca | — | oceanthropic |
| DeepSeek V4 | ds, dsf | — | ocdeepseek |
| OpenAI | — | codex55, codex54, codexmini | ocdefault, ocopenai |
| OpenRouter Fusion | or-fusion | codex-fusion | oc-fusion |
| Z.ai GLM-5.2 | cg, cgalt | — | oczai |
| MiniMax M3 | mm | — | ocminimax |
| Qwen | — | codex-qwen | ocqwen |
| StepFun | stepfun | — | — |
| Xiaomi MiMo | mimo | — | ocmimo |
| SiliconFlow | — | codex-siliconflow | — |
| Kimi | — | — | ockimi |
| Requesty EU | req-cc | req-codex | req-oc |
| LiteLLM | litellm | — | — |

**Why this matters for r/LocalLLaMA:**
- OpenRouter Fusion profile gives you a multi-model expert panel with automatic synthesis
- You can test the same Claude Code task across different providers to compare quality and cost
- Fallback chains: if OpenRouter is down, it falls through to your next provider automatically
- `multiai search "claude"` finds the best OpenRouter model for your use case

**Security:**
- AES-256-GCM encrypted credential store (keys never in plaintext .env files)
- Process-level environment isolation (whitelist approach)
- Cosign-signed releases

**Install:**
```bash
npx multiai install
# or
go install github.com/lrochetta/multiai/multiai-go/cmd/multiai@latest
```

GitHub: https://github.com/lrochetta/multiai

Curious what provider combinations the community here uses — any requests for additional profiles?
