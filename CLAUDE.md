# opencode-kit — Development Guide

## Project
Go CLI tool managing OpenCode configuration (cobra + SQLite + concurrent API calls).
Vision: **auto-orchestrator** — manages models, skills, MCPs, agents, routing, and runtime config dynamically based on user's model arsenal and harness sources.

## Methodology

**CRITICAL**: Load `maestro-dev` skill when developing maestro:
```
skill maestro-dev
```

This skill defines the full development workflow, Definition of Done, and verification pipeline.

## Commands
- `make verify` — build + vet + test-race + coverage (run before EVERY commit)
- `make test` — `go test -v ./...`
- `make lint` — `go vet ./...`
- `make build` — `go build -o maestro ./cmd/maestro/`
- `make precommit` — quick build+vet check (for rapid iteration)
- `make coverage` — `go test -race -coverprofile=coverage.out ./... && go tool cover -func=coverage.out`
- `bash test-suite.sh` — integration tests
- `make install-hooks` — enable pre-commit hook: `git config core.hooksPath .githooks`

## Workflow: SDD-TDD Hybrid
- **Features / >3 files**: SDD cycle (explore → design → tasks → TDD → verify → archive)
- **Fixes / 1-2 files**: TDD direct (test first → implement → verify)
- **ALWAYS**: `make verify` before commit

## Definition of Done (ALL must pass)
- [ ] `go build ./...` passes
- [ ] `go vet ./...` passes
- [ ] `go test -race ./...` passes
- [ ] No TODOs, stubs, placeholders, or dead code added
- [ ] Follows existing code patterns
- [ ] Conventional commit: `type(scope): description`
- [ ] Saved to Engram via `mem_save`

## TDD Rules
- NEVER write all tests first (horizontal slices anti-pattern)
- ONE test → ONE impl → repeat (vertical slices via tracer bullets)
- Test through public interfaces, not implementation details
- Only enough code to pass current test — no speculative features

## Test Conventions
- Table-driven tests with descriptive names (`TestFuncName_Scenario_Expected`)
- External test packages (`package db_test`)
- SQLite in-memory for DB tests: `db.Open(":memory:")`
- Interface-based mocking (no framework)

## Coverage Targets (progressive)
| Tier | Threshold | Packages |
|------|-----------|----------|
| Critical | 80% | `internal/db/`, `internal/sync/` |
| Core | 60% | `internal/routing/`, `internal/heal/` |
| CLI | 40% | `internal/cli/` |
| Generated/thin | 0% | `cmd/`, `pkg/models/` |

## Pre-commit Hook
Enable with: `make install-hooks`
The hook runs: `go build ./...` → `go vet ./...` → `go test -race ./...` (coverage advisory)

## Memory
Save to engram with `topic_key: architecture/maestro-workflow` or `architecture/maestro-<feature>`.
Phase completions always saved as `sessions/maestro-phase-<N>-complete`.
Session summary required on close.

## Full Reference
See [DEVELOPMENT.md](./DEVELOPMENT.md) for complete workflow specification.
See `.opencode/skills/maestro-dev/SKILL.md` for loadable skill definition.
See `.opencode/tools/verify.ts` for custom verification tool.
See `docs/development/DOD.md` for Definition of Done.
