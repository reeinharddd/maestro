# Contributing to maestro

Thank you for your interest in contributing to **maestro**, a Go CLI tool for managing
OpenCode configuration. This guide covers setup, workflow, and standards.

## Table of Contents

- [Development Setup](#development-setup)
- [Project Structure](#project-structure)
- [Available Commands](#available-commands)
- [Testing](#testing)
- [Coding Standards](#coding-standards)
- [Pull Request Process](#pull-request-process)
- [Pre-commit Hooks](#pre-commit-hooks)

---

## Development Setup

### Requirements

- **Go 1.25+** (see [go.mod](./go.mod) for exact version)
- **SQLite** (build-time dependency via `modernc.org/sqlite` — pure Go, no CGO needed)
- **Make** (for build targets)

### Clone and Build

```bash
git clone https://github.com/reeinharrrd/maestro.git
cd maestro
make build
```

The binary is placed at `./maestro`. Install it with `make install` (copies to `$GOPATH/bin/`).

### Verify Your Setup

```bash
make verify
```

This runs the full verification pipeline: build, lint, tests with race detector,
and coverage check. All gates must pass before committing.

---

## Project Structure

```
cmd/
  maestro/              # Main entry point
internal/
  audit/             # Configuration audit logic
  classifier/        # Model/agent classification
  cli/               # CLI commands (cobra-based)
  compress/          # Context compression
  config/            # Configuration management
  credentials/       # Secure credential storage
  db/                # SQLite database layer
  discover/          # Capability discovery
  generator/         # Code/config generation
  heal/              # Self-healing logic
  mcp/               # MCP protocol handling
  profile/           # Model profiling
  routing/           # Model routing logic
  sources/           # Data source abstraction
  sync/              # Concurrent API coordination
  util/              # Shared utilities
pkg/
  models/            # Shared domain types
```

---

## Available Commands

| Command | Description |
|---------|-------------|
| `make build` | Build for current platform (`./maestro`) |
| `make test` | Run all tests verbosely |
| `make lint` | Run `go vet ./...` |
| `make verify` | Full gate: build → lint → test-race → coverage |
| `make precommit` | Quick check: build → lint → test (no coverage) |
| `make coverage` | Run tests with race + coverage, show per-function report |
| `make clean` | Remove build artifacts |
| `make fmt` | Format all Go source files |
| `make tidy` | Run `go mod tidy` |
| `make install-hooks` | Enable pre-commit hooks from `.githooks/` |
| `make install` | Build and copy binary to `$GOPATH/bin/` |
| `make build-all` | Cross-compile for all supported platforms |

### Verification Pipeline

Run `make verify` before every commit. The pipeline executes in order:

1. `go build ./...` — must compile
2. `go vet ./...` — zero warnings
3. `go test -race -coverprofile=coverage.out ./...` — tests pass with no races
4. Coverage advisory (see [coverage thresholds](#coverage-thresholds))

### Coverage Thresholds

| Tier | Threshold | Packages |
|------|-----------|----------|
| Critical | 80% | `internal/db/`, `internal/sync/` |
| Core | 60% | `internal/routing/`, `internal/heal/` |
| CLI | 40% | `internal/cli/` |
| Generated / thin | 0% | `cmd/`, `pkg/models/` |

Coverage is advisory in local development. CI enforces the thresholds.

---

## Testing

maestro follows **Test-Driven Development (TDD)** with a red-green-refactor cycle.
Write one test, implement minimally to pass it, then refactor.

### Test Conventions

- **Table-driven tests** for all functions with multiple inputs or outputs.
- **Descriptive names**: `TestFuncName_Scenario_Expected` (e.g. `TestUpsertProvider_EmptyName_Error`).
- **Subtests** via `t.Run()` to organize related scenarios within a table.
- **External test packages** — test files use `package db_test` (not `package db`) to
  exercise the public API only. This keeps tests decoupled from internal details.
- **Interface-based mocking** — no mocking framework. Define interfaces in the
  consumer package and provide test implementations.
- **SQLite in-memory** for database tests:
  ```go
  db, err := db.Open(":memory:")
  ```
- **No `time.Sleep()`** in tests — use channels, conditions, or `require.Eventually()`.
- Use `t.Helper()` for shared test helpers and `t.Cleanup()` for resource teardown.
- Use `t.TempDir()` for temporary file creation.

### Running Tests

```bash
# All tests
make test

# With race detector
make test-race

# Coverage report
make coverage

# Specific package
go test -v ./internal/db/...

# Specific test
go test -v -run TestUpsertProvider ./internal/db/
```

---

## Coding Standards

### Go Idioms

- Follow standard Go conventions from [Effective Go](https://go.dev/doc/effective_go)
  and [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments).
- Use `slog` for structured logging throughout.
- Errors are typed and wrapped with `fmt.Errorf("...: %w", err)`. Avoid stringly-typed
  errors and panics in production code.
- Public API must have clear, self-documenting names — no magic values or opaque arguments.
- Run `go vet ./...` (or `make lint`) before every commit.

### Conventional Commits

```
type(scope): short summary in present tense

[optional body explaining why, not what]
```

**Types:** `feat`, `fix`, `test`, `refactor`, `docs`, `chore`, `perf`

**Scopes:** `db`, `cli`, `routing`, `sync`, `heal`, `discover`, `audit`, `models`, `generator`, `profile`

Examples:
```
feat(db): add ProviderID filter for ListModels
fix(routing): handle empty model list in SelectBestModel
test(heal): add integration test for Run()
refactor(db): extract scanModel helper from ListModels
```

During TDD cycles, use checkpoint commits:
```
test(db): add failing test for UpsertProvider
feat(db): implement UpsertProvider
refactor(db): extract common query pattern
```

### Linting

Current linting is `go vet` (zero-config, ships with Go). Plans to add
`golangci-lint` with `errcheck`, `staticcheck`, `gofmt`, and `goimports`
are tracked separately.

---

## Pull Request Process

1. **Branch from `main`**. Use a descriptive branch name:
   `feat/add-model-filter`, `fix/empty-list-panic`.

2. **Keep changes focused**. Each PR should address one concern. If your change
   touches multiple areas, consider splitting into stacked PRs.

3. **Run `make verify` before pushing.** All gates must pass.

4. **Write a clear PR description** explaining what changed and why. Reference
   any related issues.

5. **Request review** from a maintainer. Address all feedback before merging.

6. **Squash merge** into `main` with a clean conventional commit message.

---

## Pre-commit Hooks

The repository includes a pre-commit hook at `.githooks/pre-commit` that runs:

1. `go build ./...`
2. `go vet ./...`
3. `go test -race -coverprofile=coverage.out ./...`
4. Advisory coverage check

Enable it with:

```bash
make install-hooks
```

This sets `git config core.hooksPath .githooks`. The hook stashes unstaged
changes so only staged code is tested, then pops the stash afterward.

---

## Additional Resources

- [DEVELOPMENT.md](./DEVELOPMENT.md) — full development workflow specification,
  including SDD-TDD hybrid process, agent routing, and decision log.
- [Go testing package](https://pkg.go.dev/testing)
- [Table-driven tests](https://go.dev/wiki/TableDrivenTests)
- [Cobra CLI user guide](https://github.com/spf13/cobra/blob/main/user_guide.md)
