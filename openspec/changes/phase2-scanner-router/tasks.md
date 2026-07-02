# Phase 2 — Implementation Tasks: Scanner + Router + Daemon

> **Source**: `openspec/changes/phase2-scanner-router/design.md`
> **Methodology**: SDD-TDD hybrid — ONE test → ONE impl → repeat per task
> **Verification**: Each wave passes `make verify` (build + vet + test-race + coverage)
> **Dependency ordering**: Sequential within waves; parallel between independent waves

---

## Wave 1: Scanner Foundation (Types + DB)

**Goal**: New domain types and DB tables for project scanning.
**Dependency**: None (foundation wave).
**Verification**: `var _ db.DBInterface = (*DB)(nil)` compiles. Migrations run.

### 1.1 — Add Project, DetectedStack, ProjectConfig, ScannerResult types

- **Files**: `pkg/models/types.go`
- **Change**: Add 4 new structs following existing Model pattern:
  ```go
  type Project struct {
      ID          string `json:"id"`
      Path        string `json:"path"`
      Name        string `json:"name"`
      DetectedAt  int64  `json:"detected_at"`
      UpdatedAt   int64  `json:"updated_at"`
      Status      string `json:"status"` // active, stale, archived
      Source      string `json:"source"` // scan, manual, import
  }
  type DetectedStack struct {
      ID          string  `json:"id"`
      ProjectID   string  `json:"project_id"`
      Language    string  `json:"language"`
      Framework   string  `json:"framework"`
      Version     string  `json:"version"`
      Builder     string  `json:"builder"`
      TestRunner  string  `json:"test_runner"`
      Linter      string  `json:"linter"`
      DetectedAt  int64   `json:"detected_at"`
      Confidence  float64 `json:"confidence"`
  }
  type ProjectConfig struct {
      ID          string `json:"id"`
      ProjectID   string `json:"project_id"`
      ConfigType  string `json:"config_type"` // agents, mcps, skills, lsp
      Content     string `json:"content"`
      GeneratedAt int64  `json:"generated_at"`
      Hash        string `json:"hash"`
  }
  type ScannerResult struct {
      ProjectID string            `json:"project_id"`
      Stacks    []DetectedStack   `json:"stacks"`
      Configs   map[string]string `json:"configs"`
      Errors    []string          `json:"errors,omitempty"`
  }
  ```
- **Acceptance**: `go build ./...` passes. Types greppable in types.go.
- **Size**: S

### 1.2 — Create migration 000008: projects, detected_stacks, project_configs

- **Files**: `internal/db/migrations/000008_project_tables.up.sql`, `000008_project_tables.down.sql`
- **Change**: Follow existing migration pattern (000006, 000007). Up creates 3 tables with exact columns from design doc. Down drops all 3. Use `unixepoch()` for timestamps, ON DELETE CASCADE for FK refs.
- **Acceptance**: `000008_project_tables.up.sql` contains CREATE TABLE for projects, detected_stacks, project_configs. `down.sql` drops all 3.
- **Size**: S

### 1.3 — Add DBInterface + crud: Project CRUD (4 methods)

- **Files**: `internal/db/interface.go`, `internal/db/crud.go` (or new `internal/db/projects.go`)
- **Change**: Add `UpsertProject`, `ListProjects`, `GetProject`, `DeleteProject` to both DBInterface and *DB. Follow existing `UpsertProvider` pattern with `upsertRow` helper. UpsertProject uses path as UNIQUE key for conflict detection.
- **Test**: In-memory SQLite: UpsertProject → GetProject matches. ListProjects returns all. DeleteProject removes it. Table-driven.
- **Size**: M

### 1.4 — Add DBInterface + crud: DetectedStack CRUD (3 methods)

- **Files**: `internal/db/interface.go`, `internal/db/crud.go` (or `internal/db/projects.go`)
- **Change**: `UpsertDetectedStack`, `ListDetectedStacks(projectID)`, `DeleteDetectedStacks(projectID)`. Language+ProjectID as unique constraint. Cascade delete with project.
- **Test**: Upsert → List by projectID matches. Delete stacks for project → empty list.
- **Size**: M

### 1.5 — Add DBInterface + crud: ProjectConfig CRUD (4 methods)

- **Files**: `internal/db/interface.go`, `internal/db/crud.go` (or `internal/db/projects.go`)
- **Change**: `UpsertProjectConfig`, `ListProjectConfigs(projectID)`, `GetProjectConfig(projectID, configType)`, `DeleteProjectConfigs(projectID)`. ConfigType+ProjectID unique.
- **Test**: Upsert → List → Get by type matches. Delete all → empty list. Cascade with project.
- **Size**: M

**Wave 1 verification**: `make verify` passes. New migrations apply cleanly. CRUD works with in-memory SQLite.

---

## Wave 2: Scanner Engine

**Goal**: Detect project stack via dynamic evidence registry, generate project-specific configs.
**Dependency**: Wave 1 (needs DB types + CRUD).
**Sequential**: After Wave 1.

### 2.1 — Create internal/scanner package with EvidenceMatcher registry

- **Files**: `internal/scanner/detect.go` (new), `internal/scanner/scanner.go` (new)
- **Change**: Service with Detect(path) method:
  ```go
  type Service struct { db db.DBInterface }
  func New(database db.DBInterface) *Service
  func (s *Service) Detect(path string) (*models.Project, error)
  ```
  EvidenceMatcher registry with 17 default matchers: go.mod, package.json, Cargo.toml, pyproject.toml, requirements.txt, composer.json, *.sln, build.gradle*, *.csproj, CMakeLists.txt, mix.exs, shard.yml, *.zig + evidence patterns (.github/workflows/*.yml, Dockerfile, Makefile, Justfile, Taskfile.yml).
  Each matcher: glob pattern + parse function + priority. Higher priority wins on conflicts.
  Detect() walks project root, matches files against registry, calls ParseFunc, builds DetectedStack list. Upserts Project + Stacks to DB.
- **Test**: Mock project with go.mod + Makefile → detects Go language. Mock with package.json + biome.json → detects TypeScript + Biome linter. No evidence → empty stacks, no crash.
- **Size**: L

### 2.2 — Config generation per detected stack

- **Files**: `internal/scanner/generate.go` (new)
- **Change**: Generate() method reads detected stacks, produces ProjectConfig entries:
  ```go
  func (s *Service) Generate(projectID string) (*models.ScannerResult, error)
  ```
  Per-language config rules (table from design doc §2.4): Go→golang-pro/go-testing + filesystem MCP, TypeScript→typescript-pro + react/vue/angular/nestjs + filesystem/playwright, Python→python-pro/fastapi/django + filesystem, Rust→rust-engineer + filesystem.
  Output: agent lines for skills enable/disable, MCP filters, LSP selection. Stored as ProjectConfig rows.
- **Test**: Stacks=[{Language:"Go", Framework:"echo"}] → generated configs contain golang-pro, go-testing, filesystem MCP. Stacks=[{Language:"Python"}] → python-pro, fastapi expert.
- **Size**: M

### 2.3 — CLI: `maestro scan` with subcommands

- **Files**: `internal/cli/scan_cmd.go` (new), `cmd/maestro/main.go` (register)
- **Change**:
  ```go
  maestro scan [path]                    // Detect + generate + upsert
  maestro scan list                      // List all scanned projects
  maestro scan status <id>               // Show project + configs
  maestro scan regenerate <id>           // Re-run detect + generate
  ```
  `maestro scan` without path uses `os.Getwd()`. Follow providers.go command pattern. `scan list` shows table with ID, Name, Path, Status, DetectedAt.
- **Test**: `maestro scan .` on maestro repo itself → detects Go project. `maestro scan list` returns rows.
- **Size**: M

### 2.4 — Wire scanner into generator for project-specific sections

- **Files**: `internal/generator/generator.go`
- **Change**: Extend GenerateConfig to accept optional projectID. When present, read ProjectConfigs for that project and emit project-specific sections alongside global config. Merge strategy: project sections override global sections for same entity type.
- **Test**: Generate with projectID → output contains merged config. Generate without → existing behavior unchanged.
- **Size**: M

**Wave 2 verification**: `make verify` passes. `maestro scan .` on maestro repo detects Go.

---

## Wave 3: Router Evolution

**Goal**: DB-backed RoutingRules, ExecLog feedback, normalized scoring, classifier integration, budget enforcement.
**Dependency**: Wave 1 (needs DBInterface stability). Can start after Wave 1 (parallel to Wave 2).
**Parallel**: Can run alongside Wave 2.

### 3.1 — Migration 000009: routing_rules enhancements

- **Files**: `internal/db/migrations/000009_routing_enhancements.up.sql`, `000009_routing_enhancements.down.sql`
- **Change**: ALTER TABLE routing_rules ADD COLUMNS: `priority_weight REAL NOT NULL DEFAULT 1.0`, `enabled INTEGER NOT NULL DEFAULT 1`, `created_at INTEGER NOT NULL DEFAULT (unixepoch())`. Down drops all 3.
- **Acceptance**: Migration applies. Existing routing_rules rows get defaults.
- **Size**: XS

### 3.2 — CLI: `maestro route list/add/remove/test/benchmark`

- **Files**: `internal/cli/routing_cmd.go`
- **Change**:
  ```go
  maestro route list                    // Show all routing rules (DB-backed)
  maestro route add --task-key <key>    // Add custom rule (flags: description, needs-fc, needs-vision, etc.)
  maestro route remove <task-key>       // Delete custom rule
  maestro route test --task <key>       // Dry-run: show which model would be selected
  maestro route benchmark <provider>    // Run benchmark + update ExecLog
  ```
  `route list` reads from DB, shows task_key, description, enabled, priority_weight. `route test` calls SelectBestModel with current DB state. `route benchmark` triggers SyncExecLogs + re-evaluation.
- **Test**: `route add --task-key test-task --description "test"` → `route list` shows it. `route test --task coding_complex` returns best model name.
- **Size**: M

### 3.3 — ExecLog feedback loop

- **Files**: `internal/routing/router.go`
- **Change**: Add SyncExecLogs() method.
  ```go
  func (s *Service) SyncExecLogs() error
  ```
  Queries ExecLog records since last sync. Per model per task type: computes avg latency (→ LatencyP50Ms), success rate (→ FailCount), token efficiency. Updates Model fields. Refreshes circuit breaker state.
  Two modes: automatic periodic (triggered every 100 SelectBestModel calls) + on-demand (via `route benchmark`). Track last sync time internally.
  Safe on empty ExecLog (no crash, no update).
- **Test**: Insert ExecLog records → SyncExecLogs → Model.LatencyP50Ms updated. Empty ExecLog → no change.
- **Size**: L

### 3.4 — Normalized scoring with ScoringConfig

- **Files**: `internal/routing/router.go`
- **Change**: Add ScoringConfig type and normalized scoring:
  ```go
  type ScoringConfig struct {
      ContextWeight     float64 // weight for context adequacy (0-1)
      CapabilityWeight  float64 // weight for capability match (0-1)
      CostWeight        float64 // weight for cost efficiency (0-1)
      LatencyWeight     float64 // weight for speed (0-1)
      ReliabilityWeight float64 // weight for circuit breaker state (0-1)
  }
  ```
  Each dimension score normalized 0-100.
  - contextScore: `min(model.ContextWindow / requiredContext, 1.0) * 100`
  - capabilityScore: FC(+25), Vision(+25), Reasoning(+25), Audio(+15), OCR(+10). Sum capped at 100.
  - costScore: `100 - min(cost * 100, 100)` where cost = prompt+completion price per 1K tokens
  - latencyScore: `100 - min(latencyP50Ms / 20, 100)` (0ms=100, 2000ms+=0)
  - reliabilityScore: `circuitOpen ? 0 : 100 - min(FailCount * 30, 90)`
  Final = weighted average.
  Default ScoringConfig has 1.0 for all weights (backward compatible).
  Store ScoringConfig in DB or as package-level default. CLI: `maestro route config set --context-weight 1.5`.
- **Test**: Model with 100K ctx, needs FC → contextScore=100, capabilityScore≥25. Slow model (latency=5000ms) → latencyScore=0. Circuit open → reliabilityScore=0.
- **Size**: L

### 3.5 — Classifier integration in routing

- **Files**: `internal/routing/router.go`
- **Change**: At routing-time, if model.Architecture == "":
  ```go
  classResult := classifier.NewService(s.db).Classify(&model)
  model.Architecture = classResult.Architecture
  model.RecommendedUse = classResult.RecommendedUse
  ```
  Then add architecture-task alignment bonus to score:
  - reasoning tasks → +10 if Architecture == "reasoning_transformer"
  - speed tasks → +5 if lightweight (transformer + ≤64K context)
  - vision tasks → +5 if Architecture supports vision
- **Test**: Model with unknown architecture → Classifier fills it. Reasoning task prefers reasoning_transformer models.
- **Size**: M

### 3.6 — Budget enforcement (hard caps)

- **Files**: `internal/routing/router.go`
- **Change**: New method CheckBudget() called before SelectBestModel returns:
  ```go
  func (s *Service) CheckBudget(def taskDef, model models.Model) error
  ```
  BudgetConfig fields: DailySpend, MonthlySpend, PerTaskBudget, TaskKey, ProviderID.
  If model would exceed daily/monthly/per-task cap → skip model (eligibility filter, not score penalty).
  Track spend in BudgetConfig table incrementally.
- **Test**: Set daily cap of $0.01, route after exceeding → model filtered out. Per-task cap exceeded → model ineligible for that task.
- **Size**: M

**Wave 3 verification**: `make verify` passes. Routing uses DB-backed rules. ExecLog feedback updates model stats. Scoring is normalized. Classifier fills architecture gaps.

---

## Wave 4: Daemon Foundation

**Goal**: Long-running HTTP proxy server, message reformulation, CLI lifecycle.
**Dependency**: Wave 3 (daemon wraps router). Cannot start before Wave 3.
**Sequential**: After Wave 3.

### 4.1 — Add DaemonConfig type + daemon_config table

- **Files**: `pkg/models/types.go`, `internal/db/migrations/000010_daemon_config.up.sql`
- **Change**:
  ```go
  type DaemonConfig struct {
      ID              string `json:"id"`
      Addr            string `json:"addr"`
      AdminAddr       string `json:"admin_addr"`
      MaxConcurrency  int    `json:"max_concurrency"`
      BufferSizeKB    int    `json:"buffer_size_kb"`
      AutoStart       bool   `json:"auto_start"`
      LogDir          string `json:"log_dir"`
      CreatedAt       int64  `json:"created_at"`
      UpdatedAt       int64  `json:"updated_at"`
  }
  ```
  Migration 000010: CREATE TABLE daemon_config with columns matching struct.
  DBInterface: GetDaemonConfig, UpsertDaemonConfig.
- **Size**: S

### 4.2 — Create internal/daemon package: HTTP proxy + health endpoints

- **Files**: `internal/daemon/server.go` (new), `internal/daemon/daemon.go` (new)
- **Change**:
  ```go
  type Config struct { ... }  // Addr, AdminAddr, DBPath, LogDir, UpstreamURL, MaxConcurrency, BufferSize
  type Daemon struct {
      cfg       Config
      db        db.DBInterface
      classifier *classifier.Service
      router    *routing.Service
      httpSrv   *http.Server
      adminSrv  *http.Server
      started   time.Time
  }
  func New(cfg Config, database db.DBInterface) *Daemon
  func (d *Daemon) Start() error    // Start both HTTP + admin servers, block
  func (d *Daemon) Stop() error     // Graceful shutdown with context timeout
  func (d *Daemon) IsRunning() bool
  ```
  HTTP server: endpoints `/v1/chat/completions` (POST, OpenAI-compatible), `/v1/messages` (POST, Anthropic-compatible), `/health` (GET), `/stats` (GET).
  Proxy handler reads request body, forwards to router for model selection (if no model specified), then forwards to real provider API. Returns streaming or non-streaming response.
  Starting point: simple pass-through proxy (no reformulation yet — that's Wave 5).
- **Test**: Start daemon → `/health` returns 200. `/stats` returns JSON with uptime. Stop daemon gracefully.
- **Size**: L

### 4.3 — CLI: `maestro daemon start/stop/status/logs`

- **Files**: `internal/cli/daemon_cmd.go` (new), `cmd/maestro/main.go` (register)
- **Change**:
  ```go
  maestro daemon start [--port 4096]    // Start daemon (foreground, blocks)
  maestro daemon stop                    // POST /admin/shutdown → graceful stop
  maestro daemon status                  // GET /health → running or not
  maestro daemon logs [--tail 50]        // Read log file, last N lines
  ```
  `start` foreground only (no daemonize). Prints "Maestro daemon listening on :4096" and blocks until SIGINT/SIGTERM.
  `stop` sends HTTP POST to admin port /admin/shutdown.
  `status` hits /health. `logs` reads from LogDir.
- **Test**: `daemon start &` → daemon starts. `daemon status` shows running. `daemon stop` stops it.
- **Size**: M

### 4.4 — Message reformulator: strategies + token estimation

- **Files**: `internal/daemon/reformulate.go` (new)
- **Change**:
  ```go
  type ReformulationStrategy int
  const (
      StrategyNone      ReformulationStrategy = iota
      StrategyCompress
      StrategyTruncate
      StrategyRestructure
      StrategySplit
  )
  func estimateTokens(body string) int  // rough: len/4 or len/3.5 for non-English
  ```
  Reformulation logic from design doc §4.4:
  - Near context limit (>80%): compress if quality model, truncate otherwise
  - Far below limit (<10%): expand if free tier, pass-through if paid
  - Default: pass-through
  At this wave, reformulator is wired but strategies are basic (strategyNone = pass-through is the actual behavior). Full implementations come in Wave 5.
- **Test**: estimateTokens works within 20% of actual. reformulate with no-op returns same request.
- **Size**: M

**Wave 4 verification**: `make verify` passes. Daemon starts, responds to /health, proxies requests, stops gracefully.

---

## Wave 5: Daemon Intelligence

**Goal**: LLM message compression, runtime routing in daemon, ExecLog recording, admin API.
**Dependency**: Wave 4 (daemon server exists). Cannot start before Wave 4.
**Sequential**: After Wave 4.

### 5.1 — LLM message compression service

- **Files**: `internal/compress/compress.go`
- **Change**: Extend existing Compressor with message compression:
  ```go
  type MessageCompressor struct {
      maxTokens int
      strategy  CompressionStrategy
  }
  type CompressionStrategy int
  const (
      StrategySelective  CompressionStrategy = iota // Keep essential, summarize filler
      StrategyProgressive                           // Compress oldest messages more
      StrategyStructural                            // Restructure for token efficiency
  )
  func (mc *MessageCompressor) CompressMessages(msgs []Message, targetTokens int) ([]Message, error)
  func (mc *MessageCompressor) EstimateTokens(content string) int
  func (mc *MessageCompressor) SelectMessagesForTruncation(msgs []Message, budget int) []int
  ```
  StrategySelective: keep system message + last user/assistant pair, summarize everything before that.
  StrategyProgressive: compress ratio increases with message age (oldest get 80% reduction, newest 0%).
  StrategyStructural: merge consecutive same-role messages, remove redundant content.
- **Test**: Messages exceeding target token count → compressed under target. Empty messages → empty result. Single message within limit → unchanged.
- **Size**: L

### 5.2 — Runtime routing in daemon

- **Files**: `internal/daemon/router.go` (new)
- **Change**: Wire router into proxy pipeline:
  ```go
  func (d *Daemon) resolveTargetModel(req *ProxyRequest) (*models.Model, error)
  ```
  If request specifies a model → validate it's available and circuit isn't open.
  If no model → call routing.Service.SelectBestModel with auto-detected task type (analyze request path + message content).
  Task type detection heuristic: `/v1/chat/completions` → coding_fast or coding_complex based on message length. `/v1/messages` → same.
- **Test**: Request with model "gpt-4" → validates availability. Request without model → routing selects best.
- **Size**: M

### 5.3 — ExecLog recording in daemon

- **Files**: `internal/daemon/logger.go` (new)
- **Change**:
  ```go
  func (d *Daemon) recordExecution(req *ProxyRequest, resp *ProxyResponse, modelID string, duration time.Duration)
  ```
  Captures: task_key, model_id, prompt_tokens, completion_tokens, duration_ms, success, error_message.
  Inserts via DBInterface.InsertExecLog().
  Runs async (goroutine) to avoid blocking proxy. Uses sync.Pool for log buffers if hot path.
- **Test**: Proxy request completes → ExecLog row exists with correct duration and token counts.
- **Size**: M

### 5.4 — Admin API: runtime config, routes, circuits, budget

- **Files**: `internal/daemon/admin.go` (new)
- **Change**:
  ```
  GET    /admin/config          → Current DaemonConfig
  PUT    /admin/config          → Update runtime config (Addr, MaxConcurrency, etc.)
  GET    /admin/routes          → Current routing table (taskDefs → best model per)
  GET    /admin/executions      → Recent ExecLog entries (last 100)
  GET    /admin/circuits        → Circuit breaker status per model
  GET    /admin/budget          → Current budget spend vs caps
  PUT    /admin/budget          → Update budget caps
  POST   /admin/shutdown        → Graceful shutdown
  ```
  Admin server runs on AdminAddr (default :4097), separate from proxy port for security.
- **Test**: GET /admin/routes returns JSON array. GET /admin/circuits shows circuit state. POST /admin/shutdown stops daemon.
- **Size**: M

**Wave 5 verification**: `make verify` passes. Daemon routes, compresses, logs, and exposes admin API.

---

## Wave 6: Integration & Polish

**Goal**: End-to-end tests, CLI integration tests, full verification.
**Dependency**: All previous waves.
**Sequential**: After Waves 4+5.

### 6.1 — End-to-end test: scan → generate configs → route

- **Files**: `internal/scanner/scanner_test.go`
- **Test**: Create temp project with go.mod + package.json. Run scanner.Detect() → 2 detected stacks. Run Generate() → project configs contain relevant agents/MCPs per stack. Upsert to DB → ListProjectConfigs matches.
- **Size**: M

### 6.2 — End-to-end test: daemon proxy with mock provider

- **Files**: `internal/daemon/daemon_test.go`
- **Test**: Start daemon with mock config. Send POST /v1/chat/completions with sample messages. Verify proxy forwards to mock provider. Verify ExecLog recorded. Verify /health returns 200. Stop daemon.
- **Size**: M

### 6.3 — CLI integration tests for scan, route, daemon

- **Files**: `internal/cli/scan_cmd_test.go`, `internal/cli/routing_cmd_test.go`, `internal/cli/daemon_cmd_test.go`
- **Test**: `maestro scan .` on known directory returns expected output. `maestro route list` shows rules. `maestro daemon status` works (no daemon running = clean message).
- **Size**: M

### 6.4 — Full `make verify` pass

- **Change**: Build + vet + test-race + coverage all pass. Coverage thresholds maintained.
- **Size**: XS

**Wave 6 verification**: `make verify` passes. All E2E tests pass.

---

## Implementation Order

```
Wave 1 (Scanner types+DB) ─────────────────────────────────────┐
                                                                ▼
Wave 2 (Scanner engine) ────────────────────────────┐     │
                  │                                  ▼     ▼
                  └── Wave 3 (Router evolution) ── Integration 1 (scan+route)
                                                              │
                                                              ▼
                                              Wave 4 (Daemon foundation)
                                                              │
                                                              ▼
                                              Wave 5 (Daemon intelligence)
                                                              │
                                                              ▼
                                              Wave 6 (Integration + polish)
```

Waves 1→2 sequential (need types before scanner engine).
Wave 3 starts after Wave 1 (parallel to Wave 2 — independent, needs DBInterface not scanner).
Wave 4 needs Wave 3 (daemon wraps router).
Waves 5→6 sequential after Wave 4.

## Verification Checklist (per wave)

Each wave MUST pass before starting the next dependent wave:
- [ ] `go build ./...` passes
- [ ] `go vet ./...` passes
- [ ] `go test -race ./...` passes
- [ ] Coverage ≥ thresholds (db/sync 80%, routing/heal 60%, cli 40%)
- [ ] No TODOs, stubs, placeholders, dead code
- [ ] Conventional commit per task
- [ ] `mem_save` with topic_key `architecture/maestro-phase2-<wave>`
