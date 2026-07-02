# Maestro — Vision

> An orchestra conductor doesn't play every instrument.  
> They know how each one sounds, when it should play, and how to direct the musicians so everything comes together in harmony.  
> **Maestro does the same for your AI development environment.**

---

## The Problem

Today, working with AI-assisted development tools means juggling:

- **Multiple models** — each with different context windows, capabilities, architectures, and costs
- **Many harnesses** — OpenCode, Claude Code, Codex, VS Code extensions, custom scripts
- **Dozens of skills and MCPs** — most unused, all loaded anyway, burning context
- **Project-specific needs** — every repo has its own stack, patterns, conventions, rules
- **No central source of truth** — configs scattered, duplicated, inconsistent across environments

The result: **bloated context, wasted tokens, inconsistent results, and tools that exist but never get proper use.** Workflows are luck-based, not engineered. Users pay for capabilities they never fully leverage.

---

## The Solution: Maestro

Maestro is a **central orchestrator** for your AI development environment. It doesn't replace your tools — it makes them work together, consistently, optimally.

### Core: Central Config Database

Everything starts with a **SQLite database** (the existing maestro DB) that becomes the single source of truth for:

- **Providers** — API keys, base URLs, availability
- **Models** — capabilities, context limits, function-calling support, architecture, cost
- **Agents** — definitions, prompts, model bindings
- **Skills** — what they do, when they trigger, what they need
- **MCPs** — servers, tools, resources, which projects use them
- **Commands** — slash commands, routes
- **Prompts & templates** — reusable message structures
- **Harnesses** — runtime configurations for different environments

This database is **portable** — not tied to any single IDE or tool. It lives wherever you go.

### CLI (Setup & Control)

```bash
maestro init                  # Initialize config
maestro project scan          # Analyze current project
maestro project generate      # Generate optimal configs for this project
maestro config export         # Export to OpenCode, Codex, Claude Code, etc.
maestro model list            # Query model capabilities
maestro status                # See what's active
```

### Daemon (Intelligent Proxy)

A lightweight daemon runs in the background and acts as a **selective proxy** between you and your AI tools:

- **Simple messages** pass through untouched (zero latency overhead)
- **Complex requests** get processed: model selection, message reformulation, context compression, routing
- Knows each model's strengths and limits — formats messages accordingly
- Can compress context when the target model has tight constraints
- Routes to the best available model for each task

---

## How It Works

### Global Scope (Always Active)

Core methodologies, rules, and system prompts that apply everywhere, every project, every environment. These are the **anchor principles** — they never change.

### Project Scope (On Enter)

When you open a project, Maestro:

1. **Scans** the entire project — technologies, stack, versions, patterns, rules, conventions
2. **Generates** specific configurations for this project:
   - Which agents to create
   - Which MCPs to activate (and which to ignore)
   - Which skills to include (and which to discard)
   - Optimal system prompts for the detected stack
   - Consumable files for the AI tools in use
3. **Activates** project scope on top of global scope — keeping context lean and relevant

### Runtime (On Request)

When you send a message:

1. **Analyze** the task, complexity, domain
2. **Select** the best model from all available providers
3. **Reformulate** the message for the target model's capabilities and constraints
4. **Compress** if the model has tight context limits
5. **Route** to the right handler
6. **Learn** from the result to improve future decisions

---

## What Maestro IS

| Capability | Description |
|-----------|-------------|
| **Central config store** | Single source of truth for all AI tooling config |
| **Project scanner** | Analyzes repos and generates optimal configs |
| **Model router** | Selects best model per task based on capabilities |
| **Message optimizer** | Reformulates and compresses for target model |
| **Portability layer** | Exports/imports config across environments |
| **Knowledge base** | Knows every component's capabilities and limits |

## What Maestro IS NOT

| Boundary | Why |
|----------|-----|
| **Not an IDE / editor** | Maestro works with VS Code, Helix, Neovim, etc. — doesn't compete |
| **Not an agent runtime** | Maestro doesn't execute agents — it configures and directs them |
| **Not a framework** | Not LangChain, not an SDK — no vendor lock-in |
| **Not model-dependent** | Independent of any provider or model — works with all |
| **Not a replacement for OpenCode** | Maestro complements and enhances, it doesn't replace |

---

## Before vs After

| | Before Maestro | After Maestro |
|---|---|---|
| **Configuration** | Scattered, duplicated, inconsistent | Centralized, portable, optimized |
| **Context usage** | Bloated, all skills loaded always | Lean, project-relevant only |
| **Cost** | Wasted tokens, unused capabilities | Optimized model selection, minimal waste |
| **Tool usage** | Many tools installed, few used properly | Every tool used where it fits best |
| **Consistency** | Luck-based, varies per session | Engineered, repeatable results |
| **Portability** | Tied to one IDE/harness | Works everywhere |

---

## Design Principles

1. **Portable first** — Configs live in the database, not in any tool's config file
2. **Progressive** — CLI first, daemon later, each step adds value independently
3. **Transparent** — You always know what Maestro is doing and why
4. **Lightweight** — Minimal overhead, zero impact on simple flows
5. **Offline-capable** — Core functionality works without internet
6. **Single-user focused** — Optimized for one person's environment and workflow
7. **Open core** — 100% open source, community-driven

---

## Target User

A single developer who:

- Uses multiple AI coding tools (OpenCode, Claude Code, Codex, VS Code extensions)
- Works across multiple projects with different stacks
- Cares about token efficiency and cost
- Wants consistent, repeatable AI-assisted workflows
- Values portability — same setup everywhere
- Invests in their tooling and wants maximum ROI

---

## Roadmap

### Phase 1: Central Config Database (Current → Next)
Strengthen the existing SQLite database as the source of truth:
- Complete CRUD for all entity types (providers, models, agents, skills, MCPs)
- Robust sync with OpenCode configuration
- Model capability classification (context, functions, architecture, tier)
- Query and analysis capabilities

### Phase 2: Project Scanner + Generator
- Full project analysis (stack, dependencies, patterns, conventions)
- Auto-generation of project-specific configs
- Intelligent MCP/skill selection per project
- Export to multiple target formats (OpenCode, Codex, Claude Code, VS Code)

### Phase 3: Daemon + Intelligent Proxy
- Lightweight background daemon
- Selective message interception and processing
- Model selection engine
- Message reformulation and compression
- Routing automation

---

## Brand

**Name**: Maestro (it. *master*, orchestra conductor)  
**Direction**: Ascending — batuta ascendente a +30°  
**Color**: Maestro Gold `#c8a064`  
**Typography**: Playfair Display italic (wordmark), Inter (UI), DM Mono (terminal)  

Full brand system at `docs/brand/logo-system.html`.

---

## License

100% open source. Community-driven development.

---

*This vision document captures the long-term product direction. Implementation is incremental — each phase delivers standalone value. See DEVELOPMENT.md for current state and next steps.*
