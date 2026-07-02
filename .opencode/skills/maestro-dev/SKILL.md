---
name: maestro-dev
description: maestro development methodology — SDD-TDD hybrid, Definition of Done, verification pipeline. Load when developing maestro.
---

## Workflow

### For features / changes >3 files — SDD-TDD hybrid
1. **SDD Explore** — search engram, understand codebase, read relevant files
2. **SDD Design** — define interfaces, types, architecture decisions
3. **SDD Tasks** — break into implementation tasks
4. **TDD Apply** — ONE test → ONE impl → repeat per task
5. **SDD Verify** — `make verify` (build + vet + test + coverage)
6. **SDD Archive** — save to engram

### For small fixes / 1-2 files — direct TDD
1. Read relevant code
2. Write test FIRST (TDD)
3. Implement
4. `make verify`
5. Conventional commit
6. Save to engram

## TDD Rules
- NEVER write all tests first (horizontal slices anti-pattern)
- ONE test → ONE impl → repeat (vertical slices via tracer bullets)
- Test behavior through public interfaces, not implementation details
- Only enough code to pass current test — no speculative features

## Definition of Done (ALL must pass)
- [ ] `go build ./...` passes
- [ ] `go vet ./...` passes
- [ ] `go test -race ./...` passes
- [ ] Coverage doesn't decrease below per-package thresholds
- [ ] No TODOs, stubs, placeholders, or dead code added
- [ ] Follows existing code patterns and idioms
- [ ] Conventional commit: `type(scope): description`
- [ ] Saved to Engram with `mem_save`

## Test Conventions
- Table-driven tests with descriptive names
- External test packages: `package db_test`
- SQLite in-memory for DB tests: `db.Open(":memory:")`
- Interface-based mocking (no framework)
- Coverage targets: db/sync 80%, routing/heal 60%, cli 40%

## Verification Commands
```makefile
make verify    # build + vet + test-race + coverage check
make test      # go test -v ./...
make lint      # go vet ./...
make build     # go build -o maestro ./cmd/maestro/
make precommit # quick check before commit
```

## Commit Conventions
```
type(scope): brief summary in present tense
```
Types: feat, fix, test, refactor, docs, chore, perf
Scope: db, cli, routing, sync, heal, discover, audit, models, generator, profile

## Engram Memory
- Save after every significant change: `mem_save`
- Use topic_key for evolving topics: `architecture/maestro-<feature>`
- Session summary before ending: `mem_session_summary`
