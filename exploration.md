## Exploration: Foundation Phase — okit Codebase Assessment

### Current State
The okit codebase at `/home/erik/projects/okit` is a Go CLI tool managing OpenCode configuration (cobra + SQLite + concurrent API calls). The HANDOFF.md describes 7 Foundation Phase items, but the **codebase is significantly ahead of the HANDOFF document** — most items are already implemented.

**Module**: `github.com/reeinharddd/okit` (module name contains the old username)

---

### (a) Config Path Parametrization

**Status: PARTIALLY DONE**

`internal/cli/configpath.go` already implements `OpenCodeConfigDir()` with proper priority:
```
OPENCODE_CONFIG_DIR env → XDG_CONFIG_HOME + "opencode" → HOME/.config/opencode
```

However, `internal/db/db.go`'s `DefaultPath()` uses `os.UserConfigDir()` directly — NOT `OpenCodeConfigDir()`:

```go
// db.go line 28-39 — does NOT use configpath.go's OpenCodeConfigDir()
func DefaultPath() string {
    configDir, err := os.UserConfigDir()
    // ...
    base := filepath.Join(configDir, "opencode")
    return filepath.Join(base, "opencode-kit.db")
}
```

This means: if someone sets `OPENCODE_CONFIG_DIR=/custom/path`, the config file path (`OpenCodeConfigPath()`) correctly resolves to `/custom/path/opencode.json`, but the DB path (`DefaultPath()`) still goes to `$HOME/.config/opencode/opencode-kit.db`.

**Affected files:**
- `internal/db/db.go` — `DefaultPath()` doesn't use `OpenCodeConfigDir()`
- `internal/cli/configpath.go` — correct implementation, just not wired into db.go
- `internal/cli/doctor.go` — lines 48-49 use both `OpenCodeConfigDir()` and `OpenCodeDBPath()` (correct)
- `internal/cli/init.go` — line 28 uses `OpenCodeConfigDir()` (correct)
- `internal/cli/skills_cli.go` — line 73 uses `db.DefaultPath()` directly (should use `OpenCodeConfigDir()`)
- `internal/cli/providers.go` — line 223 uses `filepath.Dir(d.DBPath())` for config dir (may be inconsistent)

**No hardcoded `/home/reeinharrrd/` found in any Go source file.** The HANDOFF.md is stale on this point.

---

### (b) DB Interface Extraction

**Status: FULLY DONE**

- `internal/db/interface.go` — complete `DBInterface` with all 40+ methods
- `internal/db/db.go` line 26: `var _ DBInterface = (*DB)(nil)` — compile-time conformance check
- All internal packages already use `db.DBInterface`:

| Package | Field Type | File |
|---------|-----------|------|
| routing | `db.DBInterface` | router.go:17 |
| discover | `db.DBInterface` | discover.go:17 |
| profile | `db.DBInterface` | profile.go:20 |
| heal | `db.DBInterface` | heal.go:12 |
| sync | `db.DBInterface` | sync.go:16 |
| generator | `db.DBInterface` | generator.go:18 |
| audit | `db.DBInterface` | audit.go:20 |

Only 4 references to concrete `*db.DB` remain in the CLI layer:
- `internal/cli/root.go:86` — `openDB()` returns `*db.DB` (factory function, acceptable)
- `internal/cli/root.go:97` — `runHeal()` takes `*db.DB`
- `internal/cli/root.go:560` — `findConfigPath()` takes `*db.DB`
- `internal/cli/providers.go:222` — `syncConfig()` takes `*db.DB`

These are CLI wiring functions and are fine — they construct the concrete type then pass `DBInterface` to service constructors.

---

### (c) DB Migrations

**Status: FULLY DONE**

- `golang-migrate` v4.19.1 is already in `go.mod`
- `internal/db/db.go` lines 84-101 — `Migrate()` function uses `github.com/golang-migrate/migrate/v4` with `iofs` embedded source and `sqlite` driver
- Migration files exist at `internal/db/migrations/`:
  - `000001_init.up.sql` (265 lines, 23 tables)
  - `000001_init.down.sql` (23 DROP TABLE statements)
- Migrations run automatically in `Open()` via `if err := Migrate(db); err != nil { ... }`

**BUG FOUND — Schema mismatch in snapshots table:**

The migration `000001_init.up.sql` line 205 creates:
```sql
CREATE TABLE IF NOT EXISTS snapshots (
    ...
    data TEXT NOT NULL DEFAULT '{}',
    ...
);
```

But Go code in `internal/db/routing.go` uses column `content` everywhere:
```go
// line 153
INSERT INTO snapshots (hash, content) VALUES (?, ?)
// line 161
SELECT id, hash, content, COALESCE(created_at,'') FROM snapshots
// line 179
SELECT id, hash, content, COALESCE(created_at,'') FROM snapshots WHERE id=?
```

This is a **runtime error** — queries will fail with "no such column: content". The column should be `content` or the migration needs a new version.

---

### (d) Dead Code

**Status: FULLY CLEANED**

| Item | Found in code? | Evidence |
|------|---------------|----------|
| `ModelScore` struct | ❌ Not in current code | Only referenced in HANDOFF.md |
| `TaskRequirement` struct | ❌ Not in current code | Only referenced in HANDOFF.md |
| Duplicate `stripJSONC` | ❌ Only one implementation | `internal/util/util.go:7` — used by sync, validate, mcpprofile, doctor |
| Unused `import _ "time"` | ❌ Not found | `time` in routing.go IS used by `circuitOpen()` |
| Stale test files | ❌ Already deleted | `db_test.go`, `heal_test.go`, `router_test.go` don't exist on disk |

The `STALE_TESTS.md` file references APIs (`routing.TaskRequirements`, `db.CreateDB`) that no longer exist — the file itself is now stale documentation of already-deleted files.

---

### (e) Backup Implementation

**Status: FULLY DONE** (code was moved from non-existent `internal/daily/daily.go` to `internal/cli/root.go`)

- `createBackup()` at `root.go:736-765` — full `.tar.gz` backup of SQLite DB file
- `cleanupOldBackups()` at `root.go:767-782` — removes backups older than 30 days
- Called from daily pipeline at `root.go:214-222`

**Note:** `internal/daily/daily.go` does NOT exist — the daily pipeline is implemented entirely in `root.go`'s `newDailyCmd()`.

---

### (f) Missing CLI Commands

**Status: ALL 8 ARE IMPLEMENTED**

HANDOFF lists 8 tables without CLI coverage. Current code covers ALL of them:

| Table | CLI Command | File | Commands |
|-------|------------|------|----------|
| budget_config | `okit budget` | `budget.go` | `show`, `set` |
| lsp_servers | `okit lsp` | `lsp.go` | `list` |
| snapshots | `okit snapshots` | `snapshots.go` | `list`, `show`, `delete` |
| preferences | `okit prefs` | `prefs.go` | `list`, `get`, `set`, `delete` |
| skills | `okit skills` | `skills_cli.go` | `list`, `report`, `sync` |
| source_items | `okit source-items` | `sourceitems.go` | `list`, `import`, `report` |
| exec_log | `okit exec-log` | `exec.go` | `list` |
| model_profiles | `okit profiles` | `profiles_cli.go` | `list` |

All are registered in `root.go` lines 53-81.

**Unrelated gap discovered**: Agents table has no dedicated CLI command (`newAgentsCmd` doesn't exist anywhere). Agents are only managed through `sync import`.

---

### (g) Persist Profiles

**Status: FULLY DONE**

`internal/profile/profile.go` line 85 already persists to DB:
```go
if err := s.db.UpsertModelProfile(prof); err != nil {
    fmt.Printf("  Warning: save profile for %s: %v\n", model.ID, err)
}
```

Both `ProfileModel()` (single model) and `ProfileAll()` (all models) persist profiles. The CLI can read them back via `okit profiles list`.

---

### (h) Current Test State

- No `_test.go` files exist in `internal/db/`, `internal/heal/`, or `internal/routing/` — all stale tests have been deleted
- Existing tests: `internal/cli/cli_test.go`, `internal/sync/sync_test.go`, `internal/compress/compress_test.go`, `internal/generator/generator_test.go`, `internal/audit/live_test.go`
- `STALE_TESTS.md` still exists but describes already-deleted files

---

### (i) Module Name

- Module: `github.com/reeinharddd/okit` (typo: `reeinharddd` with 3 d's)
- Current user is `erik` at `/home/erik/` — the module name doesn't match the actual GitHub org/user
- All imports reference `github.com/reeinharddd/okit/...`

---

### Dependency Order Between Items

```
Item 1 (Config path) ─────────────────────────── (no deps)
Item 2 (DB interface) ─── already done ───────── (no deps)
Item 3 (Migrations) ───── already done ───────── (no deps)
Item 4 (Dead code) ────── already cleaned ────── (no deps)
Item 5 (Backup) ───────── already done ───────── (no deps)
Item 6 (Missing CLI) ──── already done ───────── (no deps)
Item 7 (Persist profiles) already done ───────── (no deps)

Real items remaining (not in HANDOFF):
A. Fix snapshot schema mismatch ─ depends on Item 3 (needs new migration)
B. Wire db.DefaultPath() to OpenCodeConfigDir() ─ depends on Item 1
C. Add agents CLI command ─ new independent work
D. Clean up STALE_TESTS.md ─ depends on verifying tests pass
```

---

### Risks and Edge Cases Found

1. **Snapshot schema mismatch (CRITICAL)** — migration has `data` column, Go code uses `content`. Any code path that inserts or queries snapshots will panic with SQL error.

2. **Config path inconsistency** — setting `OPENCODE_CONFIG_DIR` makes config files route correctly but leaves DB in default location. This could cause "config says one thing, DB says another" confusion.

3. **No agents CLI** — the `agents` table has full CRUD operations in `db/agents.go` but no way to manage agents from the CLI. Only accessible via `sync import`.

4. **Module name mismatch** — `github.com/reeinharddd/okit` won't match actual deployments. This affects `go install` and any future CI/CD.

5. **Test coverage gap** — zero tests for the core DB CRUD operations, routing service, backup, or healing. Only 5 test files exist, none exercise the main business logic.

6. **STALE_TESTS.md is misleading** — describes tests as still present and recommends deletion, but they're already gone. Someone following the doc would be confused.

7. **The entire Foundation Phase is already done** — the HANDOFF is a strategic document that's now out of sync with the actual codebase. Implementing any of its 7 items would be wasted effort.

### Ready for Proposal
**Yes**, but with a critical caveat: The Foundation Phase items in HANDOFF.md are already implemented. The orchestrator should inform the user that:
- Items 1-7 from HANDOFF are substantially complete
- The real remaining work is: fixing the snapshot schema migration mismatch, wiring `DefaultPath()` to `OpenCodeConfigDir()`, adding an agents CLI command, and cleaning up the stale HANDOFF/STALE_TESTS documentation
- The next phase should focus on test coverage and the items actually remaining
