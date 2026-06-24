# BMAD+ — AI Agent Configuration

## Project Context
This project uses BMAD+, an augmented AI-driven development framework.
Based on BMAD-METHOD v6.6.0 with multi-role agents, autopilot mode, and parallel execution.

## Agents
To activate an agent, say its name or persona:
- **Atlas** (Strategist) — Business analysis + Product management
- **Forge** (Architect-Dev) — Architecture + Development + Documentation
- **Sentinel** (Quality) — QA + UX review
- **Nexus** (Orchestrator) — Sprint management + Autopilot + Parallel execution

## Skills
- Load skills from `.agents/skills/`
- Each agent has a SKILL.md with capabilities, activation protocol, and role-switching rules
- Auto-activation triggers: `.agents/data/role-triggers.yaml`

## Key Commands
- `bmad-help` — Show all available agents and skills
- `autopilot` — Launch Nexus in full pipeline mode
- `parallel` — Enable parallel multi-agent execution

## Communication
- User name: laurent
- Default language: French for user-facing content, English for code and technical docs.
