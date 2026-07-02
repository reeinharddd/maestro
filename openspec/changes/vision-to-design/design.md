# SDD Design: Phase 1 — Foundation: Central Config DB

> **Based on**: `docs/VISION.md`
> **Evidence**: `.omo/evidence/crud-matrix.md`, `.omo/evidence/sync-generator-report.md`
> **Status**: Draft

---

## 1. Overview

Phase 1 establishes the **foundation** — a centralized configuration database with complete CRUD across all entities, a unified sync pipeline, and basic model intelligence. This phase does NOT include the daemon proxy, scanner, or non-OpenCode integrations (those are Phases 2-3 in VISION.md).

> **Metaphor**: First we build the music stand (DB) and tune every instrument (CRUD), then we ensure the sheet music is consistent (sync), and finally we teach the conductor to recognize each musician's range (classification).

### 1.1 Goals

| Goal | Success Criteria |
|------|------------------|
| Complete DBInterface | All 19 entity methods declared in `DBInterface` (currently 43 declared, 19 missing) |
| Complete CLI CRUD | Every entity has `list/add/update/remove` CLI commands reading from DB |
| Unified sync | Single config parser used by both `sync.ImportFromOpenCodeConfig` and `generator.SyncExistingToDB` |
| Safe sync | Delete detection, preview, backup before overwrite |
| Generator completeness | LSP, Plugin, and proper MCP merge in `opencode.jsonc` output |
| Model classification foundation | Capture provider-intrinsic metadata + fallback strategy |

### 1.2 Scope — In

- All 19 structs in `pkg/models/types.go` that have DB methods (all except Tool, ToolParameter)
- `internal/db/interface.go` — add missing method declarations
- `internal/cli/` — add missing CRUD commands for Command, MCP, LSP, Skill, Agent
- `internal/sync/sync.go` — refactor shared parser, add delete detection, preview, backup
- `internal/generator/generator.go` — add LSP/Plugin export, fix MCP merge, add format validation
- `internal/classifier/` — model classification scaffolding (fallback-only for Phase 1)

### 1.3 Scope — Not In

- Daemon proxy / runtime routing (Phase 2)
- Project scanner (Phase 2)
- Non-OpenCode integrations (Phase 3)
- Message reformulation / compression / routing (Phase 2)
- Live provider API probing for model classification (manual/discover-based only)

---

## 2. DBInterface Completion

### 2.1 Current State

```
DBInterface: 43 methods declared
NOT in interface (but implemented on *DB): 19 methods
Completely missing from interface: 3 entities (SyncLog, ExecLog, Snapshot)
```

### 2.2 Methods to Add

**Agent (2):**
```go
GetAgent(id string) (*models.Agent, error)
DeleteAgent(id string) error
```

**Model (2):**
```go
SearchModels(query string, filter ...ModelFilter) ([]models.Model, error)
GetStats() (*models.Stats, error)
```

**Routing (3):**
```go
GetRoutingRule(id string) (*models.RoutingRule, error)
ListRoutingEvents(limit int) ([]models.RoutingEvent, error)
UpsertBudget(budget *models.BudgetConfig) error
```

**SyncLog (2):**
```go
InsertSyncLog(log *models.SyncLog) error
ListSyncLogs(limit int) ([]models.SyncLog, error)
```

**ExecLog (2):**
```go
InsertExecLog(log *models.ExecLog) error
ListExecLogs(limit int) ([]models.ExecLog, error)
```

**Snapshot (4):**
```go
InsertSnapshot(snapshot *models.Snapshot) error
ListSnapshots(limit int) ([]models.Snapshot, error)
GetSnapshot(id string) (*models.Snapshot, error)
DeleteSnapshot(id string) error
```

**Preferences (4):**
```go
GetPreference(key string) (string, error)
DeletePreference(key string) error
CleanupProviderPrefs(providerID string) error
CleanupInvalidPreferences() error
```

**Total: 19 methods**

### 2.3 Design Decision: Interface Growth

`DBInterface` will grow from 43 to 62 methods. This is intentional — the interface IS the contract for the repository layer. Every service depends on `DBInterface`, not concrete `*DB`. If the interface becomes too large in the future, we can split into domain-specific sub-interfaces (e.g., `AgentDB`, `ModelDB`, `SyncDB`), but that's a future concern.

> **Rationale**: A single large interface is simpler to mock for services that need multiple domains. Splitting prematurely adds complexity without proven need.

### 2.4 Compile-Time Verification

Add a compile-time check (already present for `*DB`, extend pattern):
```go
var _ db.DBInterface = (*DB)(nil)  // exists in db/db.go
```

After adding methods, this line will fail to compile if any method signature mismatches — instant verification.

---

## 3. CLI CRUD Completion

### 3.1 Current CLI Gaps

| Entity | Has DB CRUD | Has CLI | Gap |
|--------|-------------|---------|-----|
| Command | Full | **None** | No `maestro commands` subcommand |
| MCPServer | Full | Partial | `mcp start filesystem` bypasses DB; no `mcp list/add/update/remove` |
| LSPServer | Full | Partial | Only `lsp list` exists; no add/update/remove |
| Skill | Full | Partial | Only `skills list/report/sync`; no add/update/remove |
| Agent | Full | Partial | Only `agents list/get/delete`; no add/update |

### 3.2 Design: CLI Pattern

Follow the established `providers.go` pattern for new commands:

```
maestro commands list        → ListCommands (DBInterface)
maestro commands add         → UpsertCommand
maestro commands update      → UpsertCommand
maestro commands remove      → DeleteCommand (needs to be added to DBInterface)

maestro mcps list            → ListMCPs
maestro mcps add             → UpsertMCP
maestro mcps update          → UpsertMCP
maestro mcps remove          → DeleteMCP (needs new)
maestro mcps start filesystem → keep existing (bypasses DB for runtime)

maestro lsp add              → UpsertLSPServer
maestro lsp update           → UpsertLSPServer
maestro lsp remove           → DeleteLSPServer

maestro skills add           → UpsertSkill
maestro skills update        → UpsertSkill
maestro skills remove        → DeleteSkill (needs new)
maestro skills sync          → keep existing

maestro agents add           → UpsertAgent
maestro agents update        → UpsertAgent
```

### 3.3 New DBInterface Methods Needed for CLI

These entities don't have `Delete` methods in DB (DB + interface):

- `DeleteCommand(id string) error` — new
- `DeleteMCP(id string) error` — new
- `DeleteSkill(id string) error` — new
- `DeleteAgent(id string) error` — exists in `*DB`, add to interface

### 3.4 Test Strategy

Each new CLI command group gets a `*_test.go` following `agents_test.go` pattern:
- `list` returns entities from DB
- `add` creates entity + verifies via list
- `update` modifies + verifies
- `remove` deletes + verifies empty
- Error cases: missing args, invalid IDs

---

## 4. Sync Unification

### 4.1 The Problem

Two separate import paths with different field coverage:

| Feature | `sync.ImportFromOpenCodeConfig` | `generator.SyncExistingToDB` |
|---------|-------------------------------|------------------------------|
| Models imported | ✅ From whitelist/models | ❌ Not imported |
| Provider base URL dedup | ❌ | ✅ Merges by base URL |
| Timeout options | ❌ | ✅ Parsed |
| EnterpriseURL | ❌ | ✅ Parsed |
| Steps/PromptFile/Permission | ❌ | ✅ Parsed for agents |
| MCP Source | `"opencode"` | `"sync"` |
| Plugin import | ✅ Stored as pref | ❌ Not handled |

### 4.2 Design: Shared Parser

Extract config parsing into `internal/sync/parser.go`:

```go
// Parser handles reading + parsing opencode config files.
// Used by both sync.ImportFromOpenCodeConfig and generator.SyncExistingToDB.
type Parser struct{}

func (p *Parser) ParseConfig(path string) (*ParsedConfig, error)
func (p *Parser) ParseProviders(raw map[string]interface{}) ([]ParsedProvider, error)
func (p *Parser) ParseAgents(raw map[string]interface{}) ([]ParsedAgent, error)
func (p *Parser) ParseCommands(raw map[string]interface{}) ([]ParsedCommand, error)
func (p *Parser) ParseMCPs(raw map[string]interface{}) ([]ParsedMCP, error)
func (p *Parser) ParseLSPServers(raw map[string]interface{}) ([]ParsedLSPServer, error)
func (p *Parser) ParseMeta(raw map[string]interface{}) (*ParsedMeta, error)
```

`ParsedConfig` struct aggregates all parsed sections. Both `sync.ImportFromOpenCodeConfig` and `generator.SyncExistingToDB` call the same parser, then apply their own upsert logic.

Parse target: **field-union** — cover all fields from both implementations. The union includes:
- `timeout`, `headerTimeout`, `chunkTimeout` from options
- `enterpriseUrl`, `setCacheKey` from options
- `steps`, `promptFile`, `permission` from agents
- `models` map entries (model limits, capabilities, cost, modalities)

### 4.3 Source Field Unification

`sync.ImportFromOpenCodeConfig` uses `"opencode"`, `generator.SyncExistingToDB` uses `"sync"`. Use `"opencode"` as the canonical source for all config-file imports. The `"sync"` value was arbitrary.

---

## 5. Sync Gap Fixes

### 5.1 Delete Detection

**Problem**: `Diff.Removed*` fields are declared but never populated. Import is additive-only.

**Solution**: Before importing, snapshot the current DB state (list all entity IDs). After import, compute the diff. Entities that existed before but aren't in the config file were removed.

```go
type Diff struct {
    AddedProviders   []string
    RemovedProviders []string  // ← newly populated
    AddedModels      []string
    RemovedModels    []string  // ← newly populated
    AddedAgents      []string
    RemovedAgents    []string  // ← newly populated
    AddedCommands    []string
    RemovedCommands  []string  // ← newly populated
    AddedMCPs        []string
    RemovedMCPs      []string  // ← new field
}
```

### 5.2 Preview / Dry-Run Mode

Add `DryRun bool` field to `ImportFromOpenCodeConfig`. When true, compute the diff and return it without writing to DB. CLI:

```
maestro sync import --dry-run   # show what would change
```

### 5.3 Backup Before Overwrite

Before `ExportToOpenCodeConfig` writes, copy the existing file:
```go
if _, err := os.Stat(configPath); err == nil {
    os.Copy(configPath, configPath+".bak")
}
```

### 5.4 Fix AddedAgents Bug

Line 151 in `sync.go`: `AddedAgents` is always appended regardless of whether the agent already existed. Fix: track existing agents the same way providers do (build `existingMap` before import, only append if `!existingMap[agentID]`).

### 5.5 New Diff Field: AddedMCPs

`Diff.RemovedMCPs` is declared but `Diff.AddedMCPs` is NOT. The MCP section tracks `AddedAgents`-style (always appends). Add `AddedMCPs []string` to `Diff`.

---

## 6. Generator Enhancements

### 6.1 LSP Export

**Problem**: `generator.go:435-437` explicitly skips LSP keys. LSP servers in DB are never written to `opencode.jsonc`.

**Solution**: Build an `lsp` section in the output:
```json
{
  "lsp": {
    "typescript": { "command": "typescript-language-server", "args": ["--stdio"] }
  }
}
```

`buildMetaFromDB` already reads `config/*` preferences. Add a dedicated `buildLSPSection()` that reads LSPServers from DB and writes them as a top-level `lsp` key.

### 6.2 Plugin Export

**Problem**: Plugin config is imported and stored as preferences but never re-exported. `generatorManagedKeys` has no "plugin".

**Solution**: Read plugin preferences and include them in the output. `plugin` is already a top-level key in `opencode.jsonc` schema — just missing from the generator's managed keys list.

### 6.3 Proper MCP Merge (Delete Detection)

**Problem**: `mergeMCP` is additive-only. MCPs deleted from DB stay in the existing config output.

**Solution**: After building generated MCPs, filter the existing config's MCP section: only keep MCPs that are NOT in the generated set AND were not in the DB (i.e., user-added MCPs that the generated list doesn't know about). Remove MCPs that were in the DB but were deleted.

### 6.4 Format Validation

Add a lightweight validation step before writing:
```go
func validateOpenCodeConfig(config map[string]interface{}) error
```
Checks: required top-level keys present, no `nil` values where `object` expected, provider format valid. This prevents writing invalid `opencode.jsonc` that would break OpenCode on next load.

---

## 7. Model Classification Foundation

### 7.1 Strategy

Phase 1 does NOT do live API probing. Instead:

1. **Provider metadata**: Enrich model records with whatever provider APIs already expose during `discover` (architecture, tier, context limits, function calling support, pricing)
2. **Fallback**: If provider returns incomplete metadata, mark capabilities as `unknown` (never crash)
3. **Storage**: Extend `pkg/models/types.go` `Model` struct with classification fields if missing:
   - `Architecture` (e.g., "dense", "mixture_of_experts")
   - `Tier` (e.g., "frontier", "fast", "cheap")
   - `RecommendedUse` (e.g., "coding", "reasoning", "creative")

### 7.2 Classifier Service

New file `internal/classifier/classifier.go` matching the project's service convention:

```go
type Service struct {
    db db.DBInterface
}

func New(database db.DBInterface) *Service

// Classify updates model entries with capability metadata.
// Uses provider API data first; falls back to heuristic defaults.
func (s *Service) Classify(model *models.Model) (*models.ModelClassification, error)
```

### 7.3 Test Strategy

- Mock provider returns incomplete metadata → classifier marks as `unknown`, no crash
- Valid provider metadata → classifier correctly maps fields
- Provider returns no metadata → all capabilities default to `unknown`

---

## 8. TDD Implementation Waves

> **Methodology**: ONE test → ONE impl → repeat (vertical tracer bullets per wave).
> Each wave is independently verifiable via `make verify`.

### Wave 1: DBInterface (Foundation)

| Task | Details |
|------|---------|
| 1a | Add 19 missing methods to `DBInterface` in `internal/db/interface.go` |
| 1b | Verify all concrete implementations on `*DB` match the interface |
| 1c | Run compile-time check: `var _ db.DBInterface = (*DB)(nil)` must pass |
| **Tests** | Existing tests already cover the implementations. Interface declaration is the change. |

### Wave 2: CLI CRUD Completion

| Task | Details |
|------|---------|
| 2a | Add `DeleteCommand`, `DeleteMCP`, `DeleteSkill` to `DB` + `DBInterface` |
| 2b | Create `internal/cli/commands_cmd.go` (list/add/update/remove) |
| 2c | Create `internal/cli/mcps_cmd.go` (list/add/update/remove) — refactor existing `mcp start` |
| 2d | Extend `internal/cli/lsp_cmd.go` (add/update/remove) |
| 2e | Extend `internal/cli/skills_cmd.go` (add/update/remove) |
| 2f | Extend `internal/cli/agents_cmd.go` (add/update) |
| **Tests** | Each new CLI subcommand gets table-driven test in `internal/cli/*_test.go` |

### Wave 3: Sync Unification

| Task | Details |
|------|---------|
| 3a | Create `internal/sync/parser.go` with shared config parser |
| 3b | Refactor `sync.ImportFromOpenCodeConfig` to use shared parser |
| 3c | Refactor `generator.SyncExistingToDB` to use shared parser |
| 3d | Add `DryRun` mode to import |
| 3e | Add delete detection (populate `Removed*` fields) |
| 3f | Add backup before overwrite in export |
| 3g | Fix `AddedAgents` always-report bug |
| **Tests** | Test parser with union of both field sets. Test delete detection with before/after snapshot. Test backup creates `.bak` file. |

### Wave 4: Generator Fixes

| Task | Details |
|------|---------|
| 4a | Add LSP section to generator output |
| 4b | Add Plugin to generator output |
| 4c | Fix MCP merge for delete detection |
| 4d | Add format validation before write |
| **Tests** | Assert LSP section appears in output. Assert Plugin section appears. Assert deleted MCP gone. Assert invalid config rejected. |

### Wave 5: Model Classification

| Task | Details |
|------|---------|
| 5a | Create `internal/classifier/classifier.go` |
| 5b | Implement provider metadata enrichment |
| 5c | Implement fallback to `unknown` |
| **Tests** | Table-driven: valid metadata → correct classification; incomplete → graceful `unknown`; empty → all `unknown` |

### Wave 6: Classification Integration

| Task | Details |
|------|---------|
| 6a | Wire classifier into `discover` or `sync` pipeline |
| 6b | Run on `maestro discover` or `maestro sync import` |
| **Tests** | Integration test: discover populates classification fields in DB |

---

## 9. Dependency Matrix

```
Wave 1 (DBInterface) ─┬─→ Wave 2 (CLI CRUD) ──→ Wave 4 (Generator Fixes)
                       │
                       └─→ Wave 3 (Sync) ────────→ Wave 5 (Classification)
                                                          │
                                                          └──→ Wave 6 (Integration)
```

- Wave 1 is the single blocker — nothing depends on CLI/Sync/Generator results
- Wave 2 depends on Wave 1 (needs complete DBInterface for new CLI commands)
- Wave 3 depends on Wave 1 (needs all entity access for delete detection)
- Wave 4 depends on Wave 2 (needs CLI for LSP/Plugin management)
- Wave 5 depends on Wave 3 (needs unified sync to classify during import)
- Wave 6 depends on Wave 5 + Wave 3 (wires classification into pipeline)

Parallel execution: Waves 2 and 3 can proceed in parallel after Wave 1.

---

## 10. Key Decisions Log

| # | Decision | Rationale |
|---|----------|-----------|
| D1 | Keep single `DBInterface` (62 methods) rather than splitting | Simpler to mock for multi-domain services; splitting is premature |
| D2 | Shared parser in `internal/sync/` (not a new package) | Parser reads config files — natural fit with sync package |
| D3 | No live API probing for Phase 1 classification | Cost/benefit: server-based probing wouldn't pass through CLI; provider catalog metadata is sufficient |
| D4 | Sequential waves within each track | Each wave produces verifiable output; dependencies are clear |
| D5 | Source field unified to `"opencode"` | The `"sync"` value was arbitrary and inconsistent |
