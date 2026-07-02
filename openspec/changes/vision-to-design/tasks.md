# Phase 1 — Implementation Tasks

> **Source**: `openspec/changes/vision-to-design/design.md`
> **Methodology**: SDD-TDD hybrid — ONE test → ONE impl → repeat per task
> **Verification**: Each task passes `make verify` (build + vet + test-race + coverage)
> **Dependency ordering**: Sequential within waves; parallel between independent waves after Wave 1

---

## Wave 1: DBInterface Completion (Foundation)

**Goal**: All 19 missing methods declared in `DBInterface` for complete interface coverage.
**Dependency**: None (foundation wave).
**Verification**: `var _ db.DBInterface = (*DB)(nil)` compile-time check passes.

### 1.1 — Add GetAgent, DeleteAgent to DBInterface

- **Files**: `internal/db/interface.go`, `internal/db/agents.go`
- **Change**: Add `GetAgent(id string) (*models.Agent, error)` and `DeleteAgent(id string) error` to `DBInterface` declaration.
- **Note**: Implementations already exist on `*DB` in `agents.go` — interface declaration is the only delta.
- **Acceptance**: `go build ./...` passes; `grep 'GetAgent' internal/db/interface.go` matches.
- **Size**: XS

### 1.2 — Add SearchModels, GetStats to DBInterface

- **Files**: `internal/db/interface.go`, `internal/db/models.go`
- **Change**: Add `SearchModels(filter ModelFilter) ([]models.Model, error)` and `GetStats() (*models.ModelStats, error)` to `DBInterface`.
- **Note**: Implementations exist on `*DB`. Verify signature matches exactly.
- **Acceptance**: `go build ./...` passes; `grep 'SearchModels\|GetStats' internal/db/interface.go` matches.
- **Size**: XS

### 1.3 — Add GetRoutingRule, ListRoutingEvents, UpsertBudget to DBInterface

- **Files**: `internal/db/interface.go`, `internal/db/routing.go`
- **Change**: Add `GetRoutingRule(id string) (*models.RoutingRule, error)`, `ListRoutingEvents(filter ...interface{}) ([]models.RoutingEvent, error)`, `UpsertBudget(cfg *models.BudgetConfig) error`.
- **Note**: Implementations exist. Match signatures exactly.
- **Acceptance**: `go build ./...` passes; each method greppable in interface.go.
- **Size**: XS

### 1.4 — Add SyncLog, ExecLog, Snapshot methods to DBInterface

- **Files**: `internal/db/interface.go`, `internal/db/sync_log.go`, `internal/db/exec_log.go`, `internal/db/snapshot.go`
- **Change**: Add all 10 methods for 3 entities:
  - `InsertSyncLog(log *models.SyncLog) error`, `ListSyncLogs(limit int) ([]models.SyncLog, error)`
  - `InsertExecLog(log *models.ExecLog) error`, `ListExecLogs(limit int) ([]models.ExecLog, error)`
  - `InsertSnapshot(snap *models.Snapshot) error`, `ListSnapshots(limit int) ([]models.Snapshot, error)`, `GetSnapshot(id string) (*models.Snapshot, error)`, `DeleteSnapshot(id string) error`
- **Note**: All implementations exist. This completes 3 entities that were completely absent from DBInterface.
- **Acceptance**: `go build ./...` passes; all 10 methods greppable in interface.go.
- **Size**: S

### 1.5 — Add DeleteCommand, DeleteMCP, DeleteSkill to DB + DBInterface

- **Files**: `internal/db/interface.go`, `internal/db/commands.go`, `internal/db/mcps.go`, `internal/db/skills.go`
- **Change**: Add `DeleteCommand`, `DeleteMCP`, `DeleteSkill` to both `DBInterface` and `*DB` implementations.
- **Note**: Unlike 1.1-1.4, these need new SQL + implementation. Follow existing pattern from `DeleteProvider` or `DeleteAgent`.
- **Acceptance**: `go build ./...` passes; each delete method callable from interface.
- **Size**: S

### 1.6 — Fix UpsertBudget interface bug

- **Files**: `internal/db/interface.go`
- **Note**: `UpsertBudget` may have signature mismatch. Verify existing implementation signature in `routing.go` matches the interface declaration added in 1.3.
- **Acceptance**: `go vet ./...` passes; no unused method warnings.
- **Size**: XS

**Wave 1 verification**: `make verify` passes. `var _ db.DBInterface = (*DB)(nil)` in `db.go` compiles.

---

## Wave 2: CLI CRUD Completion

**Goal**: Every entity has complete CRUD CLI commands or is explicitly exempt.
**Dependency**: Wave 1 (needs complete DBInterface).
**Parallel**: Can run alongside Wave 3.

### 2.1 — Add agents add/update CLI commands

- **Files**: `internal/cli/agents_cmd.go`
- **Change**: Add `maestro agents add` (flags: id, model, mode, description, temperature, color) and `maestro agents update` (same flags).
- **Pattern**: Follow `internal/cli/providers_cmd.go` or `models_cmd.go` structure.
- **Test**: `maestro agents add --id test-agent --model openai/gpt-4 --mode subagent` succeeds. `agents list | grep test-agent` matches. `agents update --id test-agent --temperature 0.5` updates. Table-driven test in `agents_test.go`.
- **Size**: M

### 2.2 — Add skills add/update/remove CLI commands

- **Files**: `internal/cli/skills_cmd.go`
- **Change**: Add `maestro skills add` (flags: id, source, type), `maestro skills update`, `maestro skills remove`.
- **Pattern**: Follow existing `skills list/report/sync` convention.
- **Test**: CRUD cycle via CLI. Table-driven in `skills_test.go`.
- **Size**: M

### 2.3 — Add mcps add/update/remove CLI commands (refactor mcp start)

- **Files**: `internal/cli/mcps_cmd.go` (new or enhanced)
- **Change**: Add `maestro mcps add` (flags: id, type, command, url), `maestro mcps update`, `maestro mcps remove`. Refactor existing `mcp start filesystem` to use DB-read path instead of direct JSON manipulation.
- **Test**: CRUD cycle via CLI. `maestro mcps list` reads from DB. Table-driven in `mcps_test.go`.
- **Size**: L (refactor of `mcp start` is non-trivial)

### 2.4 — Add lsp add/update/remove CLI commands

- **Files**: `internal/cli/lsp_cmd.go`
- **Change**: Add `maestro lsp add` (flags: id, command, args), `maestro lsp update`, `maestro lsp remove`. Currently only `lsp list` exists.
- **Pattern**: Follow existing LSP patterns.
- **Test**: CRUD cycle via CLI. Table-driven in `lsp_test.go`.
- **Size**: M

### 2.5 — Add commands list CLI command

- **Files**: `internal/cli/commands_cmd.go` (new file)
- **Change**: Add `maestro commands list` — currently has no CLI presence despite full DB support.
- **Extension**: Optionally add `maestro commands add/update/remove` if needed by Wave 4.
- **Test**: `commands list` returns expected output. Table-driven.
- **Size**: S

**Wave 2 verification**: Every entity has `maestro <entity> list/add/update/remove` or explicit rationale for exemption. `make verify` passes.

---

## Wave 3: Sync Unification

**Goal**: Single shared config parser, delete detection, dry-run mode, backup safety.
**Dependency**: Wave 1 (needs complete entity access for delete detection).
**Parallel**: Can run alongside Wave 2.

### 3.1 — Create shared config parser in internal/sync/parser.go

- **Files**: `internal/sync/parser.go` (new)
- **Change**: Extract common config-parsing logic from both `sync.ImportFromOpenCodeConfig` and `generator.SyncExistingToDB` into a shared `ParseOpenCodeConfig(data []byte) (*ParsedConfig, error)` function.
- **Coverage**: Must handle ALL fields from both paths: timeout, headerTimeout, chunkTimeout, enterpriseUrl, setCacheKey, steps, promptFile, permission, baseURL dedup, MCP source.
- **Test**: Test with union of field sets from both import paths. Assert no field lost in translation.
- **Size**: L

### 3.2 — Refactor sync.ImportFromOpenCodeConfig to use shared parser

- **Files**: `internal/sync/sync.go`
- **Change**: Replace inline config parsing with call to `parser.ParseOpenCodeConfig()`. Preserve existing diff tracking and DB write logic.
- **Test**: Existing `sync_test.go` import tests continue to pass.
- **Size**: M

### 3.3 — Refactor generator.SyncExistingToDB to use shared parser

- **Files**: `internal/generator/generator.go` (lines 101-326)
- **Change**: Replace inline config parsing with call to `parser.ParseOpenCodeConfig()`. This eliminates the dual-import-path bug.
- **Critical**: Ensure no regression in field coverage — test the union.
- **Test**: New `SyncExistingToDB` unit test with mock config file. Compare output against known-correct entity state.
- **Size**: L

### 3.4 — Add DryRun mode to import

- **Files**: `internal/sync/sync.go`, `internal/sync/parser.go`
- **Change**: Add `DryRun bool` option to import flow. When true, compute diff without writing to DB.
- **CLI**: `maestro sync import --dry-run` returns diff without mutation.
- **Test**: DryRun returns correct Diff with zero DB changes.
- **Size**: S

### 3.5 — Add delete detection (populate Removed* fields)

- **Files**: `internal/sync/sync.go`
- **Change**: Before import, snapshot existing entity IDs. After import, diff to find entities in pre-snapshot but not in config → populate `Diff.Removed*`. Handle: Providers, Models, Agents, Commands, MCPs.
- **Test**: Config with 2 providers → import → config with 1 provider → import shows 1 RemovedProvider. Assert `RemovedProviders` populated.
- **Size**: M

### 3.6 — Add backup before overwrite in export

- **Files**: `internal/sync/sync.go` (ExportToOpenCodeConfig), `internal/generator/generator.go` (GenerateConfig)
- **Change**: Before `os.WriteFile`, copy existing config to `{path}.bak` with timestamp. Handle edge case where no existing config exists.
- **Test**: Export overwrites → `*.bak` file exists with original content.
- **Size**: XS

### 3.7 — Fix AddedAgents always-report bug

- **Files**: `internal/sync/sync.go` (line 151)
- **Change**: Track existing agents (like providers do with `existingMap`). Only append to `AddedAgents` if agent was NOT already in DB.
- **Test**: Re-import same config → `Diff.AddedAgents` is empty.
- **Size**: XS

**Wave 3 verification**: `make verify` passes. Dual import paths eliminated. Delete detection works. Backup files created.

---

## Wave 4: Generator Fixes

**Goal**: LSP/Plugin export, proper MCP merge, format validation.
**Dependency**: Wave 2 (needs CLI for LSP/Plugin management).
**Sequential**: After Wave 2.

### 4.1 — Add LSP section to generator output

- **Files**: `internal/generator/generator.go` (around line 380-440)
- **Change**: Build `lsp` top-level section from DB LSPServers. Write alongside `provider`, `agent`, `command`, `mcp` sections.
- **Test**: Config generated with LSP entries → `lsp` key present in output JSON.
- **Size**: M

### 4.2 — Add Plugin to generator output

- **Files**: `internal/generator/generator.go` (around line 884-901)
- **Change**: Read plugin preferences and include in output. Add "plugin" to `generatorManagedKeys`.
- **Test**: Plugin entries in DB → `plugin` key present in generated config.
- **Size**: S

### 4.3 — Fix MCP merge for delete detection

- **Files**: `internal/generator/generator.go` (mergeMCP function, lines 922-933)
- **Change**: After building generated MCP list, filter existing config MCPs: keep only MCPs that are NOT in generated set AND were not tracked in DB (user-added). Remove MCPs that are in DB but absent from generated list (deleted).
- **Test**: 3 MCPs in DB, delete 1 from DB, generate → output has 2 MCPs. User-added MCP (not in DB) survives.
- **Size**: M

### 4.4 — Add format validation before write

- **Files**: `internal/generator/generator.go`
- **Change**: Add `validateOpenCodeConfig(config map[string]interface{}) error` that checks: required top-level keys present, no nil values where object expected, provider format valid. Call before `os.WriteFile`.
- **Test**: Invalid config (nil provider, missing key) → error returned. Valid config → no error.
- **Size**: S

**Wave 4 verification**: `make verify` passes. LSP, Plugin in output. MCP merge handles deletions. Invalid configs rejected.

---

## Wave 5: Model Classification Foundation

**Goal**: Classifier service that enriches model records with capability metadata from provider APIs.
**Dependency**: Wave 3 (needs unified sync to classify during import).
**Sequential**: After Wave 3.

### 5.1 — Create internal/classifier/classifier.go

- **Files**: `internal/classifier/classifier.go` (new), `pkg/models/types.go` (extend Model if needed)
- **Change**: New service matching project conventions:
  ```go
  type Service struct { db db.DBInterface }
  func New(database db.DBInterface) *Service
  func (s *Service) Classify(model *models.Model) (*models.ModelClassification, error)
  ```
- **Fields**: Architecture, Tier, RecommendedUse. Fallback to `unknown` for any missing data.
- **Test**: Valid metadata → correct classification. Incomplete → marks missing as `unknown`. Empty → all `unknown`.
- **Size**: M

### 5.2 — Implement provider metadata enrichment

- **Files**: `internal/classifier/classifier.go`
- **Change**: For each model, read whatever provider API returned during `discover`. Map to classification fields.
- **Provider-specific mapping**: OpenAI (architecture from model name prefix), Anthropic (tier from model family), etc.
- **Test**: Provider-specific mapping logic with known model names.
- **Size**: M

### 5.3 — Implement fallback to unknown

- **Files**: `internal/classifier/classifier.go`
- **Change**: Graceful handling — if a provider returns no metadata for a model, set all classification fields to `unknown`. Never panic, never crash.
- **Test**: Provider returns empty capabilities → all fields `unknown`. Provider returns partial → known fields set, missing = `unknown`.
- **Size**: S

**Wave 5 verification**: `make verify` passes. Classifier handles all metadata quality levels without crashing.

---

## Wave 6: Classification Integration

**Goal**: Wire classifier into the pipeline so classification happens automatically.
**Dependency**: Wave 5 + Wave 3 (needs unified sync + classifier).
**Sequential**: After both Wave 5 and Wave 3.

### 6.1 — Wire classifier into discover or sync pipeline

- **Files**: `internal/discover/discover.go` or `internal/sync/sync.go`
- **Change**: After import/discover completes, run classifier on each model. Store classification in DB.
- **Position**: Best triggered after `maestro discover` (new models added) or `maestro sync import` (config changes).
- **Test**: Discover populates classification fields in DB. Re-classify updates existing.
- **Size**: M

### 6.2 — Add CLI flag for classification

- **Files**: `internal/cli/discover_cmd.go`, `internal/cli/sync_cmd.go`
- **Change**: Add `--classify` flag or make it default behavior (design decision). If default, provide `--no-classify` to skip.
- **Test**: `maestro discover --classify` runs classifier. `maestro discover --no-classify` skips.
- **Size**: S

### 6.3 — Integration test for classification pipeline

- **Files**: New test file or add to existing discover_test.go
- **Test**: End-to-end: seed provider + models → run discover with classification → assert classification fields populated in DB.
- **Size**: S

**Wave 6 verification**: `make verify` passes. Discover + classify pipeline produces classified models in DB.

---

## Implementation Order

```
Wave 1 (DBInterface) ─┬─→ Wave 2 (CLI CRUD)
                       │       └─→ Wave 4 (Generator Fixes)
                       │
                       └─→ Wave 3 (Sync Unification)
                               └─→ Wave 5 (Classification)
                                       └─→ Wave 6 (Integration)
```

1. **Wave 1** first (blocker for everything)
2. **Waves 2 + 3** in parallel (independent after Wave 1)
3. **Wave 4** after Wave 2
4. **Wave 5** after Wave 3
5. **Wave 6** after both Wave 5 + Wave 3

## Verification Checklist (per wave)

Each wave MUST pass before starting the next dependent wave:
- [ ] `go build ./...` passes
- [ ] `go vet ./...` passes
- [ ] `go test -race ./...` passes
- [ ] Coverage ≥ thresholds (db/sync 80%, routing/heal 60%, cli 40%)
- [ ] No TODOs, stubs, placeholders, dead code
- [ ] Conventional commit per task
- [ ] `mem_save` with topic_key `architecture/maestro-phase1-<wave>`
