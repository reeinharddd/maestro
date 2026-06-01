# opencode-kit: Comprehensive Handoff for Parallel Implementation

## Vision & Strategy

opencode-kit manages OpenCode configuration (cobra + SQLite + concurrent API calls).
Current state: functional skeleton with ~20 CLI commands covering ~60% of available DB tables.
**Goal**: Transform from config-file manager into a full agent context management platform.

### Three Strategic Pillars

1. **Agent Memory & Context** — context compression, cross-agent memory, cache alignment
2. **Model Routing & Failover** — cost-aware routing, circuit breakers, fallback chains
3. **Ecosystem Integration** — plug into gstack/claude-skills/ECC skill ecosystem

---

## PART 1: Current Codebase Audit Summary

### Internal Packages (10)
| Package | Files | Purpose | Gaps |
|---------|-------|---------|------|
| `cmd/` | 1 | CLI entry, ~20 subcommands | No completion, no help groups |
| `internal/cli/` | 6 | CLI implementations | 8 of 16 DB tables uncovered |
| `internal/db/` | 6 | SQLite (database/sql + mattn/go-sqlite3) | No migrations, no FK enforcement |
| `internal/discover/` | 3 | Model discovery from providers | No rate limiting, no caching |
| `internal/audit/` | 2 | Model testing (latency, throughput) | No persistence of results |
| `internal/routing/` | 1 | Model scoring (`scoreModel`) | Dead code: `ModelScore`, `TaskRequirement` |
| `internal/profile/` | 1 | Model profiling | Computed but never persisted |
| `internal/generator/` | 1 | Config generation | No template customization |
| `internal/heal/` | 1 | Auto-healing | No tests, no rollback |
| `internal/sync/`` | 2 | Bidirectional sync | No conflict resolution |
| `internal/mcp/` | 1 | MCP server management | Minimal implementation |
| `internal/keys/` | 1 | API key management | No key rotation, no encryption |
| `internal/sources/` | 1 | Skill sources management | No source validation |
| `internal/doctor/` | 1 | System diagnostics | No auto-repair |
| `internal/daily/` | 1 | Daily pipeline | `backup()` is a no-op |
| `pkg/models/` | 1 | Shared types | Duplicate types in pkg vs internal |

### DB Tables (16 SQLite tables)
| Table | CLI Coverage | Status |
|-------|------------|--------|
| providers | ✅ `providers` | Working |
| models | ✅ `models` | Working |
| agents | ✅ `agents` | Working |
| routes | ✅ `routes` | Working |
| commands | ✅ `commands` | Working |
| api_keys | ✅ `api_keys` | Working |
| mcp_servers | ✅ `mcp` | Working |
| config_history | ✅ `history` | Working |
| rate_limits | ✅ `limits` | Working |
| system_info | ✅ `doctor` | Working |
| budget_config | ❌ No CLI | Missing |
| lsp_servers | ❌ No CLI | Missing |
| snapshots | ❌ No CLI | Missing |
| preferences | ❌ No CLI | Missing |
| skills | ❌ No CLI | Missing |
| source_items | ❌ No CLI | Missing |
| exec_log | ❌ No CLI | Missing |
| model_profiles | ❌ No CLI | Missing (profiles computed but not saved) |

### Key Dead Code
- `ModelScore` struct in `routing.go` — never used after scoring
- `TaskRequirement` struct — same file, never referenced
- `import _ "time"` guard in `daily.go` — only to prevent unused import on no-op `backup()`
- Two `stripJSONC` functions — one in `util.go`, one in `discover.go`, duplicate implementation

### Config Path Hardcoding
- `/home/reeinharrrd/.config/opencode/` hardcoded in multiple places
- Should use `$XDG_CONFIG_HOME` + `$OPencode_CONFIG_DIR` override

### No Interfaces for Testability
- `DB` is concrete — no `DBInterface`
- `ProviderFetcher` in discover is concrete
- `ModelTester` in audit is concrete
- All packages depend on concrete db instance

---

## PART 2: Starred Repos — Ecosystem Analysis

### Direct Competitors / Alternatives
| Repo | Stars | Relevance |
|------|-------|-----------|
| **vibecode-pro-max-kit** (withkynam) | — | Spec-driven coding harness, 12 agents, 32 skills, context memory. Most similar vision to opencode-kit |
| **gstack** (garrytan) | — | 23 opinionated tools (CEO, Designer, Eng Manager, Release Manager) |
| **ECC** (affaan-m) | — | Agent harness optimization, skills, instincts, memory, security |
| **agentmemory** (rohitg00) | ★#1 | #1 persistent memory for AI coding agents |
| **code-review-graph** (tirth8205) | — | Local-first code intelligence graph, MCP, context reduction |

### Skill Ecosystems
| Repo | Assets |
|------|--------|
| **claude-skills** (Jeffallan) | 66 specialized skills for full-stack dev |
| **skills** (mattpocock) | Engineering workflow skills (tdd, diagnose, etc.) |
| **taste-skill** (Leonxlnx) | Design taste for AI |
| **caveman** (JuliusBrussee) | Token compression skill (-65%) |
| **ui-ux-pro-max-skill** (nextlevelbuilder) | UI/UX design intelligence |
| **awesome-agent-skills** (heilcheng) | Curated skill directory |
| **auto-skill** (MaTriXy) | Create skills while working |

### Agent Orchestration
| Repo | Relevance |
|------|-----------|
| **agent-orchestrator** (ComposioHQ) | Parallel coding agents, git worktrees, multi-agent |
| **ClawTeam-OpenClaw** (win4r) | Multi-agent swarm coordination |
| **build-your-own-openclaw** (czl9707) | Tutorial to build own agent |

### Other Relevant
- **agentic-ai-apis** (cporter202) — 2,036 production-ready APIs
- **openpencil** (ZSeven-W) — AI-native vector design, agent teams
- **awesome-ai-agents** (korchasa) — Curated agent tools list
- **codeflow** (braedonsaunders) — GitHub → architecture map visualization
- **Feynman** (companion-inc) — AI for complex reasoning
- **page-agent** (alibaba) — JS in-page GUI agent

---

## PART 3: Academic Research — Context Compression

### [Acon — Agent Context Optimization](https://arxiv.org/pdf/2510.00615) (Microsoft, 2025)
- **Mechanism**: LLM-based compression with guideline optimization via failure analysis
- **Key insight**: When full context succeeds but compressed fails, analyze why → update compression guideline
- **Results**: 26-54% memory reduction, preserves 95% accuracy when distilled
- **Architecture**: 
  - Compressor LLM with optimization guideline P
  - Failure analysis → natural language gradient
  - Distillation into smaller compressor (0.6B)
- **Impact**: Smaller LMs improve 46% as agents with Acon compression
- **Code**: https://github.com/microsoft/acon
- **Relevance for opencode-kit**: Directly applicable — add Acon-style compression as optional middleware

### [Focus — Active Context Compression](https://arxiv.org/abs/2601.07190v1) (2026)
- **Mechanism**: Agent autonomously decides when to compress and prune history
- **Inspiration**: Slime mold (Physarum polycephalum) — retracts from dead ends, leaves markers
- **Results**: 22.7% token reduction, maintains accuracy, 6 compressions/task on average
- **Key finding**: Aggressive prompting (compress every 10-15 calls) eliminates accuracy tradeoff
- **Exploration-heavy tasks**: 50-57% savings
- **Relevance**: Active compression is complementary to Acon — agent-driven vs LLM-driven

### [SWE-Pruner](https://arxiv.org/pdf/2601.16746) (2026)
- **Mechanism**: Task-aware line-level pruning, 0.6B neural skimmer
- **Results**: 23-54% token reduction on SWE-Bench, actually *improves* success rates
- **Key feature**: Dynamic, query-conditioned thresholding
- **Up to 14.84× compression** on single-turn tasks
- **Relevance**: Most practical for code-specific compression

### [Implicit Compression (ICAE)](https://arxiv.org/html/2605.11051v1) (2026)
- **Mechanism**: Encode context as continuous embeddings
- **Finding**: Works on single-shot, **FAILS** on multi-step agentic tasks
- **Root cause**: Error accumulation + failure to preserve long-range dependencies
- **Lesson**: Token-level compression not sufficient for agents; need semantic preservation

### [ContextEvolve](https://arxiv.org/html/2602.02597v1) (2026)
- **Mechanism**: 3-agent decomposition (Summarizer, Navigator, Sampler)
- **Result**: 33.3% improvement, 29% token reduction
- **RL isomorphism**: Maps to state representation, policy gradient, experience replay
- **Relevance**: Multi-agent compression architecture pattern

### [CodeComp — Structural KV Cache](https://arxiv.org/pdf/2604.10235) (2026)
- **Mechanism**: Static program analysis (Code Property Graph via Joern) guides KV cache compression
- **Result**: Outperforms attention-only compression, matches full-context quality
- **Key**: Structure-aware budget allocation, span-level protection
- **Relevance**: Code-specific cache optimization — training-free

### [TACO — Terminal Agent Compression](https://arxiv.org/pdf/2604.19572) (2026)
- **Mechanism**: Self-evolving compression rules from interaction trajectories
- **Result**: 1-4% accuracy gains, reduces token consumption
- **Key**: Training-free, plug-and-play

### Key Takeaways for Implementation
1. **Acon + Focus together**: Guideline-optimized LLM compression + agent-driven active compression
2. **Line-level pruning** (SWE-Pruner style) best for code-specific compression
3. **KV cache compression** (CodeComp) for inference optimization — training-free
4. **Implicit compression fails** on multi-step — avoid ICAE approach
5. **Rule-based compression** (TACO) for terminal output filtering

---

## PART 4: Academic & Industry Research — Model Routing

### [RouteLLM](https://github.com/lmsys/routeLLM) (ICLR 2025, LMSYS)
- **Mechanism**: BERT-based router trained on Chatbot Arena preferences
- **Results**: 48-75% cost reduction at fixed quality
- **Open-source**: https://github.com/lmsys/routeLLM

### [Cascade Routing](https://arxiv.org/abs/2406.02069) (ETH Zurich, 2024)
- **Mechanism**: Cheap model → verifier → escalate to strong model on failure
- **Results**: 70-80% cost reduction
- **Key insight**: Cascade pays latency tax but eliminates need for accurate complexity classifier
- **Trade-off**: +600-900ms tail latency — not for interactive UIs

### [Speculative-Race](https://www.swfte.com/blog/mixture-of-routers-llm-routing-techniques-2026) (2026)
- **Mechanism**: Fire cheap + strong model simultaneously, verifier picks first acceptable
- **Results**: 30-60% latency reduction, slight cost increase
- **When**: Interactive UIs, chat, code completion, voice

### [Mixture-of-Routers (MoR)](https://www.swfte.com/blog/mixture-of-routers-llm-routing-techniques-2026) (2026)
- **Mechanism**: Ensemble of specialized routers voting on destination
- **Dimensions**: Cost router, latency router, accuracy router
- **Results**: 75-85% cost reduction, multi-objective Pareto front
- **Architecture**: Cache → route → cascade/speculate → verify → log

### [MasRouter](https://arxiv.org/abs/2502.11133) (ACL 2025)
- **Mechanism**: Cascaded controller: collaboration mode → role allocator → LLM router
- **Results**: 1.8-8.2% improvement, 52% overhead reduction

### [Router-R1](https://arxiv.org/abs/2503.0xxxx) (NeurIPS 2025)
- **Mechanism**: Router-LLM interleaves think + route actions
- **Key**: Dynamic mid-chain model invocation, not static ensemble

### [BaRP — Bandit Routing](https://arxiv.org/abs/2505.xxxxx) (2025)
- **Mechanism**: Bandit framework — only observes quality of chosen model
- **Results**: 12.46% better than offline routers, 2.45% better than largest LLM

### [vLLM Semantic Router](https://blog.vllm.ai/2025/09/semantic-router) (Sep 2025)
- **Mechanism**: ModernBERT classifier → reasoning or direct answer path
- **Results**: 50% latency and token savings with single model
- **LoRA-based**: v0.1 Iris (Jan 2026) shares computation across classification tasks

### [PILOT](https://arxiv.org/abs/2505.xxxxx) (2025)
- **Mechanism**: Contextual bandit — shared embedding space, online refinement
- **Result**: Dial cost/quality trade-off at inference time

### [LiteLLM Proxy](https://github.com/BerriAI/litellm) (2024-2026)
- **Most deployed** open-source LLM proxy
- **Fallbacks**: `fallbacks`, `context_window_fallbacks`, `content_policy_fallbacks`
- **Load balancing**: round-robin, least-busy, latency-based, cost-based

### [Bifrost — Enterprise AI Gateway](https://www.getmaxim.ai) (2026)
- **Automatic fallback**: Sequential chains, circuit breakers, adaptive load balancing
- **Hedging**: Parallel requests for latency-sensitive paths
- **Governance**: Virtual keys, approved providers, SOC 2 / HIPAA audit logs
- **Key metrics**: 11µs overhead at 5K req/s

### [Agent Runtime Patterns](https://arxiv.org/html/2605.20173v1) (2026)
- **Stochastic-Deterministic Boundary** (SDB): proposer → verifier → commit → reject
- **6 patterns**: P1 (Orchestrator), P2 (Saga), P3 (Event Log), P4 (Gate), P5 (State Machine), P6 (Control Plane)
- **Replay divergence**: Same input → different output under model version change
- **Relevance**: SDB is the pattern to follow for opencode-kit's routing

### Key Takeaways for Routing Implementation
1. **Sequential fallback** is baseline — circuit breakers needed for production
2. **Multi-signal routing** beats single-signal — combine complexity, task type, latency SLA
3. **Shadow routing** for validation — never deploy routing changes directly
4. **Cost distributions are heavy-tailed** — prioritize routing decisions for high-cost requests
5. **MoR architecture**: cache → route → cascade/speculate → verify → log

---

## PART 5: Implementation Plan — Ordered by Priority

### Phase 1: Foundation (Week 1)
**Priority: HIGH — prerequisites for everything else**

1. **Config path parametrization** — replace hardcoded `/home/reeinharrrd/.config/opencode/` with `$XDG_CONFIG_HOME` + `$OPencode_CONFIG_DIR` override
2. **DB interface extraction** — create `DBInterface`, make concrete DB implement it, all packages depend on interface
3. **DB migrations** — add golang-migrate, version-controlled schema changes
4. **Remove dead code** — `ModelScore`, `TaskRequirement`, duplicate `stripJSONC`, unused time import
5. **Fix backup()** — implement actual backup to `.tar.gz`
6. **Add missing CLI commands** — CLI for all 8 uncovered tables (budget_config, lsp_servers, snapshots, preferences, skills, source_items, exec_log, model_profiles)
7. **Persist profiles** — save computed model profiles to `model_profiles` table

### Phase 2: Context Compression (Week 2)
**Priority: HIGH — core differentiator**

8. **Observation compression** — Acon-style LLM compression for agent interaction history
   - Compressor prompt with guideline optimization
   - Trajectory comparison (full success vs compressed fail → feedback)
   - Configurable compression ratio per agent
9. **Selective pruning** — SWE-Pruner-style line-level pruning for terminal output
   - Lightweight binary relevance classifier
   - Command output: keep errors/warnings, strip success noise
   - Preserve structural integrity (no mid-line truncation)
10. **Active compression triggers** — Focus-inspired agent-driven compression
    - `start_focus` / `complete_focus` tool pattern
    - Auto-trigger every N steps (configurable)
    - Knowledge block for preserved learnings

### Phase 3: Model Routing (Week 2-3)
**Priority: HIGH — production-critical**

11. **Fallback routing** — implement `FallbackIDs` field in Agent struct
    - Sequential fallback chain (primary → fallback1 → fallback2)
    - Circuit breaker: N failures → cool-down → half-open → closed
    - Provider outage detection + failover
12. **Cost-aware routing** — RouteLLM-inspired
    - Complexity classifier (lightweight model)
    - Task-type classification
    - Budget-aware: route to cheaper models unless complexity justifies cost
13. **Shadow mode** — route decisions logged but not executed for safe validation
14. **Fill pricing/latency/tags fields** — currently NULL in DB

### Phase 4: Ecosystem Integration (Week 3-4)
**Priority: MEDIUM — expansion**

15. **Skill import** — import skills from claude-skills/mattpocock-skillstaste-skill/caveman into source_items table
16. **Agent registry** — register all available agents with their model assignments
17. **MCP integration** — expose routing decisions as MCP resources/tools
18. **gstack compatibility** — implement gstack's tool contract for slot claims

### Phase 5: Production Hardening (Week 4)
**Priority: MEDIUM — reliability**

19. **CI/CD** — GitHub Actions: lint (`go vet`), test (race), build, coverage
20. **Integration tests** — `test-suite.sh` needs expansion
21. **Completion** — bash/zsh/fish completion scripts
22. **Encrypted key storage** — API keys in `api_keys` table need encryption at rest
23. **Config validation** — validate generated configs before writing

---

## PART 6: Architecture Decisions

### Database
- **Current**: `database/sql` + `mattn/go-sqlite3`
- **Keep**: SQLite matches OpenCode's approach
- **Add**: golang-migrate for schema migrations
- **Add**: Foreign key constraints
- **Add**: `DBInterface` for testability

### Routing
- **Current**: Simple `scoreModel()` with dead code
- **Target**: LiteLLM-inspired proxy with:
  - Sequential fallback chain
  - Circuit breaker per provider
  - Cost-aware routing
  - Shadow mode for validation
  - Observability (latency, cost, fallback rate per provider)

### Compression
- **Current**: None
- **Target**: 3-tier compression:
  1. Observation compression (Acon-style, LLM-based)
  2. Terminal output pruning (SWE-Pruner-style, lightweight classifier)
  3. Active compression (Focus-style, agent-driven)

### CLI
- **Current**: Flat cobra commands
- **Keep**: cobra for CLI
- **Add**: Command groups / `--help` with categories
- **Add**: Shell completion

---

## PART 7: Parallel Work Units

These can be dispatched to parallel agents:

### Agent A: Config Path + DB Interface
- Files: `internal/cli/cli.go`, `internal/db/db.go`, `internal/db/ops.go`
- Tasks: Parametrize config path, extract DBInterface, add migrations

### Agent B: Dead Code + Missing CLI
- Files: `internal/routing/routing.go`, `internal/daily/daily.go`, `internal/discover/util.go`
- Tasks: Remove dead code, fix backup(), add CLI for missing tables, persist profiles

### Agent C: Compression Core
- New files: `internal/compress/compress.go`, `internal/compress/acon.go`, `internal/compress/prune.go`
- Tasks: Implement Acon-style observation compression, SWE-Pruner-style pruning

### Agent D: Routing Engine
- New files: `internal/router/router.go`, `internal/router/fallback.go`, `internal/router/circuit.go`
- Tasks: Fallback chain, circuit breaker, cost-aware routing, shadow mode

### Agent E: Ecosystem + Tests
- Tasks: Skill import, agent registry, CI config, integration tests, shell completions

---

## PART 8: Quick Reference — Key Files

| File | Purpose | Lines | Key Functions |
|------|---------|-------|--------------|
| `cmd/okit/main.go` | CLI entry | ~50 | `main()`, subcommand registration |
| `internal/cli/cli.go` | CLI dispatch | ~300 | `RegisterCommands()`, init cobra |
| `internal/cli/provider.go` | Provider commands | ~100 | CRUD for providers table |
| `internal/cli/agent.go` | Agent commands | ~100 | CRUD for agents table |
| `internal/cli/route.go` | Route commands | ~80 | CRUD for routes table |
| `internal/db/db.go` | DB init | ~80 | `Open()`, `Init()`, `Close()` |
| `internal/db/models.go` | Table defs | ~80 | 16 CREATE TABLE statements |
| `internal/db/ops.go` | CRUD ops | ~200 | Insert/Get/List/Update/Delete |
| `internal/discover/discover.go` | Model discovery | ~120 | `DiscoverModels()`, API calls |
| `internal/audit/bench.go` | Model testing | ~150 | `BenchmarkModel()`, test prompt |
| `internal/routing/routing.go` | Scoring | ~80 | `scoreModel()`, Ranking |
| `internal/profile/profile.go` | Profiling | ~80 | `ProfileModel()` |
| `internal/generator/generate.go` | Config gen | ~100 | `GenerateConfig()` |
| `internal/heal/heal.go` | Auto-heal | ~50 | `HealConfig()` |
| `internal/daily/daily.go` | Daily run | ~100 | `RunDaily()`, no-op backup |
| `pkg/models/models.go` | Shared types | ~120 | Provider, Model, Agent, Route |

---

## PART 9: Open Questions

1. **Compression approach**: Acon-style (guideline-optimized LLM) vs Focus-style (agent-driven)? **Answer**: Both — they are complementary
2. **Routing granularity**: Per-request vs per-agent vs per-task-type? **Answer**: Per-request with per-agent defaults
3. **Storage for compression**: In SQLite or separate KV store? **Answer**: SQLite for now, add KV later
4. **Integration depth**: How tightly to integrate with gstack/ECC? **Answer**: Loose coupling — expose MCP resources
5. **Pricing data**: How to keep pricing/latency current? **Answer**: Periodic fetch + fallback to cached

---

## Research Sources

### Papers Accessed
- Acon (Microsoft, 2025) — https://arxiv.org/pdf/2510.00615
- Focus (Active Compression, 2026) — https://arxiv.org/abs/2601.07190v1
- SWE-Pruner (2026) — https://arxiv.org/pdf/2601.16746
- ICAE Implicit Compression (2026) — https://arxiv.org/html/2605.11051v1
- ContextEvolve (2026) — https://arxiv.org/html/2602.02597v1
- CodeComp (2026) — https://arxiv.org/pdf/2604.10235
- TACO (2026) — https://arxiv.org/pdf/2604.19572
- Agent Runtime Patterns (2026) — https://arxiv.org/html/2605.20173v1

### Articles Accessed
- Zylos Model Routing Survey (Mar 2026) — https://zylos.ai/research/2026-03-02-ai-agent-model-routing
- Mixture-of-Routers (Swfte, May 2026) — https://www.swfte.com/blog/mixture-of-routers-llm-routing-techniques-2026
- Bifrost Fallback Routing (Maxim AI, Apr/May 2026) — https://www.getmaxim.ai/articles/failover-routing-strategies-for-llms-in-enterprise-ai-applications/
- Model Routing as System Design (Apr 2026) — https://tianpan.co/blog/2026-04-16-model-routing-system-design-llm
- Fallback Chain Pattern — https://www.agentpatternscatalog.org/patterns/fallback-chain/
