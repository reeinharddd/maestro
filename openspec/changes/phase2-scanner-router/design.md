# SDD Design: Phase 2 — Intelligence: Scanner + Router

> **Based on**: `docs/VISION.md`, `openspec/changes/vision-to-design/design.md`
> **Evidence**: `.omo/evidence/phase2-routing-state.md`, `.omo/evidence/phase2-routing-data.md`, `.omo/evidence/phase2-scanner-landscape.md`
> **Status**: Reviewed

---

## 1. Overview

Phase 2 transforms Maestro from a **passive config registry** (Phase 1) into an **active intelligence layer** that scans projects, routes optimally, and proxies runtime decisions. Three pillars:

1. **Scanner** — Auto-triggered on project open, detect tech stack, framework, versions, patterns, rules. Generate project-specific AI configs (agents, MCPs, skills).
2. **Router** — Evolve from 6 hardcoded `taskDefs` to a DB-backed, feedback-driven routing engine with budget enforcement and historical learning.
3. **Daemon Proxy** — Long-running process that intercepts API calls, reformulates messages based on target model constraints, and routes to the optimal model.

> **Metaphor**: Phase 1 built the music stand and tuned every instrument. Phase 2 gives the conductor X-ray vision (scanner), a perfect ear (router), and a baton that conducts itself (daemon).

### 1.1 Goals

| Goal | Success Criteria |
|------|-----------------|
|| Project scanner | `maestro scan` detects tech stack, framework, versions, patterns via dynamic evidence registry — auto-triggered on project open |
| Project-aware config | Scanner generates agent files, MCP filters, skill selection per project |
| DB-backed routing rules | RoutingRules stored in DB, not hardcoded `taskDefs` |
| ExecLog feedback loop | Router consumes ExecLog history to weight models by actual performance |
| Budget enforcement | BudgetConfig enforced at routing time (daily/monthly caps, per-task budgets) |
| Dynamic latency | LatencyP50Ms refreshed from real usage, not static config |
| Daemon proxy | Long-lived process intercepts API requests, reformulates, routes |
| Message reformulation | Messages compressed/expanded based on target model context window |
| Classifier integration | Router uses Classify() from Phase 1 for architecture-aware routing |

### 1.2 Scope — In

- `internal/scanner/` — new package: project detection, tech stack identification, config generation
- `internal/db/` — new tables: `projects`, `detected_stacks`, `project_configs`
- `pkg/models/` — new types: `Project`, `DetectedStack`, `ScannerResult`, `DaemonConfig`
- `internal/routing/` — DB-backed taskDefs, ExecLog feedback, budget enforcement, classifier integration
- `internal/daemon/` — new package: long-running proxy, message reformulation, runtime routing
- `internal/compress/` — model-aware message compression (not session observation compression)
- CLI: `maestro scan`, `maestro daemon start/stop/status`, `maestro route list/test/benchmark`

### 1.3 Scope — Not In

- Non-OpenCode integrations (Phase 3 — Cursor, VS Code, Claude Code, Codex)
- Plugin marketplace (Phase 3)
- Shared config sync across machines (Phase 3)
- Live provider API probing beyond current `discover` (Phase 1)
- Full agent runtime execution (Maestro configures, doesn't run agents)

---

## 2. Scanner Service

### 2.1 New Types

```go
package models

type Project struct {
    ID          string    `json:"id"`
    Path        string    `json:"path"`
    Name        string    `json:"name"`
    DetectedAt  int64     `json:"detected_at"`
    UpdatedAt   int64     `json:"updated_at"`
    Status      string    `json:"status"` // active, stale, archived
    Source      string    `json:"source"` // scan, manual, import
}

type DetectedStack struct {
    ID          string    `json:"id"`
    ProjectID   string    `json:"project_id"`
    Language    string    `json:"language"`    // go, typescript, python, rust, etc.
    Framework   string    `json:"framework"`   // echo, nextjs, fastapi, actix, etc.
    Version     string    `json:"version"`     // detected version
    Builder     string    `json:"builder"`     // go build, npm, cargo, uv, etc.
    TestRunner  string    `json:"test_runner"` // go test, vitest, pytest, cargo test
    Linter      string    `json:"linter"`      // golangci-lint, biome, ruff, clippy
    DetectedAt  int64     `json:"detected_at"`
    Confidence  float64   `json:"confidence"`  // 0.0-1.0
}

type ScannerResult struct {
    ProjectID   string            `json:"project_id"`
    Stacks      []DetectedStack   `json:"stacks"`
    Configs     map[string]string `json:"configs"` // generated config blobs
    Errors      []string          `json:"errors,omitempty"`
}
```

### 2.2 New DB Tables

```sql
CREATE TABLE projects (
    id TEXT PRIMARY KEY,
    path TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    detected_at INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at INTEGER NOT NULL DEFAULT (unixepoch()),
    status TEXT NOT NULL DEFAULT 'active',
    source TEXT NOT NULL DEFAULT 'scan'
);

CREATE TABLE detected_stacks (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    language TEXT NOT NULL,
    framework TEXT DEFAULT '',
    version TEXT DEFAULT '',
    builder TEXT DEFAULT '',
    test_runner TEXT DEFAULT '',
    linter TEXT DEFAULT '',
    detected_at INTEGER NOT NULL DEFAULT (unixepoch()),
    confidence REAL NOT NULL DEFAULT 0.0
);

CREATE TABLE project_configs (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    config_type TEXT NOT NULL, -- 'agents', 'mcps', 'skills', 'lsp'
    content TEXT NOT NULL,
    generated_at INTEGER NOT NULL DEFAULT (unixepoch()),
    hash TEXT NOT NULL
);
```

### 2.3 Scanner Heuristics — Dynamic Evidence Registry

Detection is NOT hardcoded to a fixed language list. A **registry of evidence matchers** maps filesystem patterns to detection results. New matchers can be added without code changes:

```go
type EvidenceMatcher struct {
    Pattern     string // glob, e.g. "go.mod", "Cargo.toml", "package.json"
    ParseFunc   func(path string) (*DetectedStack, error)
    Priority    int    // higher wins on conflicts
}

var defaultMatchers = []EvidenceMatcher{
    {Pattern: "go.mod",        Priority: 10, ParseFunc: parseGoMod},
    {Pattern: "package.json",  Priority: 10, ParseFunc: parsePackageJSON},
    {Pattern: "Cargo.toml",    Priority: 10, ParseFunc: parseCargoToml},
    {Pattern: "pyproject.toml", Priority: 10, ParseFunc: parsePyproject},
    {Pattern: "requirements.txt", Priority: 5, ParseFunc: parseRequirements},
    {Pattern: "composer.json",  Priority: 10, ParseFunc: parseComposerJSON},
    {Pattern: "*.sln",         Priority: 8,  ParseFunc: parseDotNetSolution},
    {Pattern: "build.gradle*",  Priority: 8,  ParseFunc: parseGradle},
    {Pattern: "*.csproj",      Priority: 8,  ParseFunc: parseDotNetProject},
    {Pattern: "CMakeLists.txt", Priority: 8,  ParseFunc: parseCMake},
    {Pattern: "mix.exs",       Priority: 10, ParseFunc: parseElixir},
    {Pattern: "shard.yml",     Priority: 10, ParseFunc: parseCrystal},
    {Pattern: "*.zig",         Priority: 5,  ParseFunc: parseZig},
    // Evidence patterns (not language-specific)
    {Pattern: ".github/workflows/*.yml", Priority: 3, ParseFunc: parseGitHubActions},
    {Pattern: "Dockerfile",              Priority: 3, ParseFunc: parseDocker},
    {Pattern: "Makefile",                Priority: 3, ParseFunc: parseMakefile},
    {Pattern: "Justfile",                Priority: 3, ParseFunc: parseJustfile},
    {Pattern: "Taskfile.yml",            Priority: 3, ParseFunc: parseTaskfile},
}
```

The registry is extensible: users can add custom matchers via config or plugin.
### 2.4 Config Generation Rules

Per detected stack, generate opinionated configs:

| Language | Agents | MCPs | Skills |
|----------|--------|------|--------|
| Go | golang-pro, go-testing | filesystem | maestro-dev, tdd |
| TypeScript | typescript-pro, angular/nestjs/react/vue | filesystem, playwright | test-master |
| Python | python-pro, fastapi/django | filesystem | tdd |
| Rust | rust-engineer | filesystem | tdd |

### 2.5 CLI Commands

```go
maestro scan [path]                    // Scan project, detect stack, generate configs
maestro scan list                      // List all scanned projects
maestro scan status <id>               // Show project config status
maestro scan regenerate <id>           // Regenerate configs for a project
```

### 2.6 DBInterface Methods

```go
// Project
UpsertProject(project *models.Project) error
ListProjects() ([]models.Project, error)
GetProject(id string) (*models.Project, error)
DeleteProject(id string) error

// DetectedStack
UpsertDetectedStack(stack *models.DetectedStack) error
ListDetectedStacks(projectID string) ([]models.DetectedStack, error)
DeleteDetectedStacks(projectID string) error

// ProjectConfig
UpsertProjectConfig(cfg *models.ProjectConfig) error
ListProjectConfigs(projectID string) ([]models.ProjectConfig, error)
GetProjectConfig(projectID, configType string) (*models.ProjectConfig, error)
DeleteProjectConfigs(projectID string) error
```

---

## 3. Router Enhancement

### 3.1 Current Limitations

| Issue | Current State | Target |
|-------|--------------|--------|
| taskDefs | Hardcoded in router.go | DB-backed RoutingRules |
| ExecLog | Written but never read | Feedback loop: real performance → routing weights |
| Budget | Soft limit in SelectBestModel | Hard cap per task + daily/monthly |
| Classifier | Not integrated | Use Architecture/RecommendedUse in scoring |
| Latency | Static LatencyP50Ms | Dynamic from ExecLog |
| Scoring | Arbitrary linear weights | Normalized dimension scores |
| GetSmallFastModels | Exists, unused | Used for priority="speed" tasks |

### 3.2 DB-Backed RoutingRules

Current `taskDefs` map becomes DB rows:

```sql
-- Already exists: routing_rules (task_key PK, description, needs_fc, needs_vision, etc.)
-- Enhancement: add priority_weight, enabled, created_at
ALTER TABLE routing_rules ADD COLUMN priority_weight REAL NOT NULL DEFAULT 1.0;
ALTER TABLE routing_rules ADD COLUMN enabled INTEGER NOT NULL DEFAULT 1;
ALTER TABLE routing_rules ADD COLUMN created_at INTEGER NOT NULL DEFAULT (unixepoch());
```

New CLI commands:
```go
maestro route list                          // List all routing rules
maestro route add --task-key <key> ...       // Add custom routing rule
maestro route remove <task-key>              // Remove custom rule
maestro route test --task <key>              // Dry-run: what model would be selected
maestro route benchmark <provider>           // Benchmark models for all task types
```

### 3.3 ExecLog Feedback Loop (Periodic + On-Demand)

```go
// New method on routing.Service
func (s *Service) SyncExecLogs() error
```

Two modes:
1. **Automatic periodic**: runs every N routing calls (configurable, default every 100)
2. **On-demand manual**: `maestro route benchmark <provider>` triggers sync + re-evaluation

How it works:
1. `SyncExecLogs()` queries `InsertExecLog` records since last sync
2. Per model per task type: computes avg latency, success rate, token efficiency
3. Updates `Model` fields: `LatencyP50Ms`, `FailCount`, `LastTested`
4. Circuit breaker state refreshed automatically
### 3.4 Score Model Normalization

Current scoring mixes arbitrary weights (ctx*2, FC+3, latency 0-5, cost -cost*25). Target: normalized 0-100 scale with configurable dimension weights.

```go
type ScoringConfig struct {
    ContextWeight    float64 // weight for context adequacy (0-1)
    CapabilityWeight float64 // weight for capability match (0-1)
    CostWeight       float64 // weight for cost efficiency (0-1)
    LatencyWeight    float64 // weight for speed (0-1)
    ReliabilityWeight float64 // weight for circuit breaker state (0-1)
}
```

Scoring becomes:
```go
func (s *Service) scoreModel(m models.Model, def taskDef, cfg ScoringConfig) float64 {
    score := 0.0
    score += normalizedContextScore(m, def) * cfg.ContextWeight
    score += normalizedCapabilityScore(m, def) * cfg.CapabilityWeight
    score += normalizedCostScore(m, def) * cfg.CostWeight
    score += normalizedLatencyScore(m) * cfg.LatencyWeight
    score += normalizedReliabilityScore(m) * cfg.ReliabilityWeight
    return score
}
```

Default config has equal weights (1.0 each). Users can override via `maestro route config set`.

### 3.5 Classifier Integration

Phase 1 classifier is wired into `discover.go` but NOT into `router.go`. Add:

```go
// At routing startup, if model.Architecture == "":
classResult := classifier.NewService(s.db).Classify(&model)
model.Architecture = classResult.Architecture
model.RecommendedUse = classResult.RecommendedUse
```

Scoring bonus for architecture-task alignment:
```go
// reasoning tasks prefer reasoning_transformer
if def.NeedsReasoning && m.Architecture == "reasoning_transformer" {
    score += 5
}
// fast tasks prefer lightweight architectures
if def.Priority == "speed" && m.Architecture == "transformer" && m.ContextWindow <= 64000 {
    score += 3
}
```

---

## 4. Daemon Proxy Architecture

### 4.1 Overview

The daemon is a long-running process (`maestro daemon start`, CLI foreground) that:

1. **Intercepts** API calls via local HTTP endpoint (not forward proxy)
2. **Analyzes** the request to determine task type, target model, message size
3. **Routes** to the optimal model if no target specified, or validates the chosen model
4. **Reformulates** messages based on target model constraints (context window, pricing)
5. **Compresses** context when approaching model limits
6. **Logs** every execution for feedback loop

### 4.2 Architecture

```
┌─────────────────────────────────────────────────┐
│                 OpenCode/Client                  │
│  ┌───────────────────────────────────────────┐   │
│  │  API Request → Daemon Proxy (localhost)    │   │
│  └──────────────┬────────────────────────────┘   │
└─────────────────┼────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────┐
│              Maestro Daemon                      │
│                                                  │
│  ┌─────────┐  ┌──────────┐  ┌───────────────┐   │
│  │ Proxy   │→ │ Analyzer │→ │ Reformulator   │   │
│  │ Server  │  │ (task    │  │ (compression,  │   │
│  │ (HTTP)  │  │  type,   │  │  expansion,    │   │
│  │         │  │  intent,  │  │  restructuring) │   │
│  └─────────┘  │  size)   │  └───────┬───────┘   │
│               └──────────┘          │           │
│                                     ▼           │
│  ┌─────────┐  ┌──────────┐  ┌───────────────┐   │
│  │ Logger  │← │ Router   │← │ Compressor    │   │
│  │ (ExecLog│  │ (model   │  │ (context      │   │
│  │  + stats│  │  select)  │  │  optimization)│   │
│  └─────────┘  └────┬─────┘  └───────────────┘   │
│                    │                             │
│                    ▼                             │
│           ┌────────────────┐                     │
│           │ API Call       │                     │
│           │ (to real       │                     │
│           │  provider)     │                     │
│           └───────┬────────┘                     │
└───────────────────┼──────────────────────────────┘
                    │
                    ▼
         ┌────────────────────┐
         │ Provider API       │
         │ (OpenAI, Anthropic,│
         │  Google, etc.)     │
         └────────────────────┘
```

### 4.3 Daemon Server

```go
package daemon

type Config struct {
    Addr        string   // listen address (default: localhost:4096)
    AdminAddr   string   // admin API (default: localhost:4097)
    DBPath      string   // path to maestro.db
    LogDir      string   // execution logs
    UpstreamURL string   // default provider URL if none specified
    MaxConcurrency int   // max concurrent proxy requests
    BufferSize  int      // response buffer in KB
}

type Daemon struct {
    cfg       Config
    db        db.DBInterface
    classifier *classifier.Service
    router    *routing.Service
    httpSrv   *http.Server
    adminSrv  *http.Server
    started   time.Time
}
```

### 4.4 Message Reformulation

The reformulator adapts messages based on the target model's constraints:

```go
type ReformulationStrategy int

const (
    StrategyNone      ReformulationStrategy = iota // pass through
    StrategyCompress                                // compress verbose sections
    StrategyTruncate                                // trim oldest messages
    StrategyRestructure                              // reorganize for context efficiency
    StrategySplit                                    // split into multiple calls
)

func (d *Daemon) reformulate(req *ProxyRequest, model *models.Model) (*ProxyRequest, error) {
    switch {
    case estimateTokens(req.Body) > model.ContextWindow * 0.8:
        // Near context limit → compress or truncate
        if model.RecommendedUse == "quality" {
            return d.compressStrategy(req)     // Keep quality, lose size
        }
        return d.truncateStrategy(req)         // Keep recent, lose old
    case estimateTokens(req.Body) < model.ContextWindow * 0.1:
        // Far below limit → can expand with context
        if model.Tier == "paid" {
            return req, nil                     // Don't waste paid tokens
        }
        return d.restructureForExpansion(req)  // Add methodology context
    default:
        return req, nil                         // Pass through
    }
}
```

### 4.5 Proxy Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/chat/completions` | POST | OpenAI-compatible chat completion proxy |
| `/v1/messages` | POST | Anthropic-compatible messages proxy |
| `/health` | GET | Daemon health check |
| `/stats` | GET | Runtime stats (routes today, tokens proxied, avg latency) |

### 4.6 CLI Commands

```go
maestro daemon start [--port 4096]    // Start daemon (foreground)
maestro daemon stop                    // Graceful shutdown
maestro daemon status                  // Is daemon running? Stats
maestro daemon logs [--tail 50]        // View recent daemon logs
```

### 4.7 Admin API

Runs on a separate port for operational control:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/admin/config` | GET/PUT | Read/update runtime config |
| `/admin/routes` | GET | Current routing table |
| `/admin/executions` | GET | Recent execution log |
| `/admin/circuits` | GET | Circuit breaker status per model |
| `/admin/budget` | GET/PUT | Budget status and caps |
| `/admin/shutdown` | POST | Graceful stop |

---

## 5. Compress Enhancement

### 5.1 Current State

`internal/compress/compress.go` only handles session observation compression (reducing stored context size). No LLM message compression.

### 5.2 New: LLM Message Compression

```go
package compress

type MessageCompressor struct {
    maxTokens     int
    strategy      CompressionStrategy
}

type CompressionStrategy int

const (
    StrategySelective CompressionStrategy = iota // Keep essential, summarize filler
    StrategyProgressive                          // Compress oldest messages more
    StrategyStructural                           // Restructure for token efficiency
)

func (mc *MessageCompressor) CompressMessages(msgs []Message, targetTokens int) ([]Message, error)
func (mc *MessageCompressor) EstimateTokens(content string) int
func (mc *MessageCompressor) SelectMessagesForTruncation(msgs []Message, budget int) []int
```

---

## 6. Implementation Waves

### Wave 1: Scanner Foundation

| Task | Description | Files |
|------|-------------|-------|
| 1.1 | New types: Project, DetectedStack, ProjectConfig, ScannerResult | `pkg/models/types.go` |
| 1.2 | Migration: projects, detected_stacks, project_configs tables | `internal/db/migrations/000008_*.sql` |
| 1.3 | DBInterface + crud: Project CRUD (6 methods) | `internal/db/interface.go`, `internal/db/crud.go` |
| 1.4 | DBInterface + crud: DetectedStack CRUD (4 methods) | `internal/db/interface.go`, `internal/db/crud.go` |
| 1.5 | DBInterface + crud: ProjectConfig CRUD (4 methods) | `internal/db/interface.go`, `internal/db/crud.go` |

### Wave 2: Scanner Engine

| Task | Description | Files |
|------|-------------|-------|
| 2.1 | Scanner service: Detect() method with language/stack heuristics | `internal/scanner/detect.go` |
| 2.2 | Scanner service: Config generation per detected stack | `internal/scanner/generate.go` |
| 2.3 | CLI: `maestro scan` with subcommands | `internal/cli/scan_cmd.go` |
| 2.4 | Generator extension: GenerateProjectConfig using scanner | `internal/generator/generator.go` |

### Wave 3: Router Evolution

| Task | Description | Files |
|------|-------------|-------|
| 3.1 | Migration: routing_rules enhancements (priority_weight, enabled) | `internal/db/migrations/000009_*.sql` |
| 3.2 | CLI: `maestro route list/add/remove/test/benchmark` | `internal/cli/routing_cmd.go` |
| 3.3 | ExecLog feedback loop | `internal/routing/router.go` |
| 3.4 | Normalized scoring with ScoringConfig | `internal/routing/router.go` |
| 3.5 | Classifier integration in routing | `internal/routing/router.go` |
| 3.6 | Budget enforcement (hard caps) | `internal/routing/router.go` |

### Wave 4: Daemon Foundation

| Task | Description | Files |
|------|-------------|-------|
| 4.1 | Daemon types: Config, Daemon struct | `pkg/models/types.go` |
| 4.2 | Daemon server: HTTP proxy + health/admin endpoints | `internal/daemon/server.go` |
| 4.3 | Daemon CLI: `maestro daemon start/stop/status/logs` | `internal/cli/daemon_cmd.go` |
| 4.4 | Message reformulator: strategies + token estimation | `internal/daemon/reformulate.go` |

### Wave 5: Daemon Intelligence

| Task | Description | Files |
|------|-------------|-------|
| 5.1 | LLM message compression service | `internal/compress/compress.go` |
| 5.2 | Runtime routing in daemon | `internal/daemon/router.go` |
| 5.3 | ExecLog recording in daemon | `internal/daemon/logger.go` |
| 5.4 | Admin API: runtime config, routes, circuits, budget | `internal/daemon/admin.go` |

### Wave 6: Integration & Polish

| Task | Description | Files |
|------|-------------|-------|
| 6.1 | End-to-end test: scan → generate configs → route → proxy | `internal/scanner/scanner_test.go` |
| 6.2 | End-to-end test: daemon proxy with mock provider | `internal/daemon/daemon_test.go` |
| 6.3 | CLI integration tests for scan, route, daemon commands | `internal/cli/cli_test.go` |
| 6.4 | Full `make verify` pass | All |

---

## 7. Dependency Matrix

```
Scan CLI     → Scanner Service → DBInterface (Project, DetectedStack, ProjectConfig)
Route CLI    → Router Service  → DBInterface (RoutingRule, ExecLog, Budget, Model)
Daemon CLI   → Daemon Service  → Router + Classifier + Compress + Logger
Scanner      → Generator       → opencode.jsonc (project-specific sections)
Router       → ExecLog (DB)    → feedback loop updates Model fields
Daemon       → Compress        → message optimization
Daemon       → Classifier      → architecture-aware routing
```

## 8. TDD Wave Sequencing

```
Wave 1 (Scanner types+DB) ───────────────┐
                                          ▼
Wave 2 (Scanner engine) ────────────┐     │
                                    ▼     ▼
Wave 3 (Router evolution) ────┐     Integration 1 (scan+route)
                              ▼     ▼
Wave 4 (Daemon foundation) ── Integration 2 (daemon boot) ──┐
                                                             ▼
Wave 5 (Daemon intelligence) ── Integration 3 (full proxy) ──┐
                                                             ▼
Wave 6 (Integration + polish) ───────────────────────────────┘
```

Waves 1-2 are sequential (need types before scanner engine).
Wave 3 can start after Wave 1 (independent — needs types, not scanner).
Wave 4 needs Wave 3 (daemon wraps router).
Waves 5-6 need Wave 4.

## 9. Key Decisions

| Decision | Choice | Rationale |
||----------|--------|-----------|
|| Scanner config storage | Separate `project_configs` table | Avoids polluting existing generator; scanner output is project-specific, not global |
|| Scanner trigger | Auto on project open + manual | Zero friction: scan happens when you enter a project dir |
|| Scanner detection | Dynamic evidence registry | Not limited to 4 languages; users extend via config |
|| Router taskDefs | DB-backed with migration from hardcoded | Existing `routing_rules` table already has task_key PK |
|| Scoring normalization | Configurable weights, equal defaults | Backward compatible; power users can tune |
|| Daemon proxy format | Local endpoint (not forward proxy) | Simpler — point client to localhost:4096 directly |
|| Daemon admin API | Separate port | Security: admin APIs shouldn't be exposed to proxy clients |
|| Daemon startup | CLI foreground (`maestro daemon start`) | Manual start; systemd user service is future |
|| Message compression | Selective (keep essential, summarize filler) | Progressive compresses oldest more |
|| ExecLog feedback | Periodic (auto every 100 routes) + on-demand | Warm data without manual intervention |
|| Scanner heuristics | Filesystem evidence over convention | Self-documenting: what files exist → what we detect |

## 10. Migration Plan

```sql
-- 000008_project_tables.up.sql
CREATE TABLE projects (...);
CREATE TABLE detected_stacks (...);
CREATE TABLE project_configs (...);

-- 000009_routing_enhancements.up.sql
ALTER TABLE routing_rules ADD COLUMN priority_weight REAL NOT NULL DEFAULT 1.0;
ALTER TABLE routing_rules ADD COLUMN enabled INTEGER NOT NULL DEFAULT 1;
ALTER TABLE routing_rules ADD COLUMN created_at INTEGER NOT NULL DEFAULT (unixepoch());

-- 000010_daemon_config.up.sql
CREATE TABLE IF NOT EXISTS daemon_config (
    id TEXT PRIMARY KEY,
    addr TEXT NOT NULL DEFAULT 'localhost:4096',
    admin_addr TEXT NOT NULL DEFAULT 'localhost:4097',
    max_concurrency INTEGER NOT NULL DEFAULT 10,
    buffer_size_kb INTEGER NOT NULL DEFAULT 512,
    auto_start INTEGER NOT NULL DEFAULT 0,
    log_dir TEXT DEFAULT '',
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at INTEGER NOT NULL DEFAULT (unixepoch())
);
```
