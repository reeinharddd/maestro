# Maestro Roadmap — OpenCode-First Completion

> Goal: Ship v1.0 with OpenCode infra management complete, then evaluate multi-platform expansion.

## Current State

- 25k LOC Go, SQLite (modernc.org/sqlite), Cobra CLI
- ~26 test files, some with failures
- 2 packages failing to build: `internal/profile`, `internal/sync`
- 2 open specs not closed: `config-path-unification`, `snapshot-schema`

---

## Phase 1 — Build Green (days 1-2)

**#1 Fix internal/profile build failure**
  - `profile_test.go:71:18`: mockDB.UpsertModelProfile duplicate declaration
  - Remove duplicate method, ensure interface matches db package

**#2 Fix internal/sync build failure**
  - Run `go build ./internal/sync/...` and diagnose errors
  - Most likely missing interface method or import

**#3 Fix internal/mcp test failure**
  - TestHandleExecute_ClassifyTask_InvalidArgs expects 400 but getting 200
  - Fix error handling in classify handler — invalid JSON should reject

**#4 Run full suite**
  - `go test -race -coverprofile=coverage.out ./...`
  - Target: >60% coverage (per openspec config)

## Phase 2 — Close Open Specs (days 3-5)

**#5 Config path unification**
  - Extract `cli.OpenCodeConfigDir()` into `internal/config`
  - Make `db.DefaultPath()` delegate to it
  - Update all callers
  - Already spec'd in `openspec/specs/config-path-unification/`

**#6 Snapshot schema migration**
  - Migration `000002`: rename `snapshots.data` → `snapshots.content`
  - Add down migration
  - Already spec'd in `openspec/specs/snapshot-schema/`

**#7 CLI commands (agents subcommand)**
  - `maestro agents list|get|delete`
  - Already spec'd in `openspec/specs/cli-commands/`

## Phase 3 — Polish & Docs (days 6-7)

**#8 Test coverage increases**
  - Focus on uncovered packages: skill, mcp, discover

**#9 CLI docs / --help polish**
  - Ensure all cobra commands have descriptions
  - Verify help text consistency

**#10 README update with installation, examples, screenshots**
  - Current README is good but needs usage examples

---

## Future: Multi-Platform Expansion (post-v1)

After v1 ships, consider expanding beyond OpenCode:
- `.cursorrules` import/export
- `CLAUDE.md` / `.windsurfrules` management
- `~/.config/github-copilot/` integration
- Unified provider registry across all AI agents

Not before v1 ships.
