# opencode-kit — Development Guide

## Project
Go CLI tool managing OpenCode configuration (cobra + SQLite + concurrent API calls).

## Commands
- `make test` — `go test -v ./...`
- `make lint` — `go vet ./...`
- `make build` — `go build -o okit ./cmd/okit/`
- `go test -race -coverprofile=coverage.out ./...`
- `go tool cover -func=coverage.out`
- `bash test-suite.sh` — integration tests

## Workflow (Spec → Test → Implement → Verify → Ship)

1. **Load context**: `mem_context` + `mem_search` for related work
2. **Load skill**: `golang-testing` + `golang-patterns` + `tdd` for test/impl
3. **TDD (vertical slices)**: ONE test → ONE impl → repeat. No bulk test writing.
4. **Verify**: `go vet ./... && go test -race ./... && go build ./...`
5. **Review**: `code-reviewer` skill via `reviewer` agent
6. **Ship**: Conventional commits (`feat(db):`, `fix(routing):`, `test(heal):`, etc.)
7. **Save**: `mem_session_summary` at session end

## Test Conventions
- Table-driven tests with descriptive names (`TestFuncName_Scenario_Expected`)
- External test packages (`package db_test`)
- SQLite in-memory for DB tests: `db.Open(":memory:")`
- Interface-based mocking (no framework)

## Skills per Stage
| Stage | Skills |
|-------|--------|
| Tests | `golang-testing`, `tdd` |
| Impl | `golang-patterns`, `golang-pro` |
| Review | `code-reviewer` |
| Debug | `diagnose` |

## Agents
| Task | Agent |
|------|-------|
| Implementation | `coder` |
| Code review | `reviewer` |
| Debugging | `debugger` |
| CI/config | `devops` |
| Quick lookup | `explore` |

## Stale Tests
3 stale `_test.go` files block `go test ./...`. See [STALE_TESTS.md](./STALE_TESTS.md).

## Memory
Save to engram with `topic_key: architecture/dev-workflow` or `handoff/<feature>`.

## Full Reference
See [DEVELOPMENT.md](./DEVELOPMENT.md) for the complete workflow specification.
