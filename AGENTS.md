# maestro — Agent Instructions

> Go CLI tool managing OpenCode configuration (Cobra + SQLite + concurrent API calls).

## Project Overview

- **Module**: `github.com/reeinharrrd/maestro`
- **Binary**: `maestro`
- **Go**: 1.25
- **Stack**: Cobra CLI + modernc.org/sqlite (CGo-free, WAL mode) + golang-migrate + golang.org/x/sync
- **Keyring**: `zalando/go-keyring` (credential storage)
- **Logging**: `slog` (text handler, stderr output)
- **Linter**: golangci-lint v2 (see `.golangci.yml` — strict revive, sloglint, testifylint)
- **Pre-commit**: `.githooks/` — enabled via `git config core.hooksPath .githooks`

## Product Vision
> Maestro es un director de orquesta — no sabe tocar todos los instrumentos,
> pero sabe cómo suenan, sabe dirigir a las personas que los tocan para que
> puedan coordinarse y sonar bien. Coordina agents, modelos, skills, commands,
> MCPs, prompts, harnesses.
>
> **Before Maestro**: Harnesses sobrecargados, skills sin uso, contexto quemado,
> flujo a suerte, herramientas infrautilizadas.
> **After Maestro**: Costos bajan, resultados consistentes, convicción en las
> herramientas, setup portable a cualquier entorno.

### What Maestro IS

- **Central configuration database** — almacena todas las configs en una BD centralizada, accesible desde cualquier IDE o lugar
- **Capability analyzer** — sabe de cada modelo: contexto, funciones, arquitectura, tier, costos, modalidades. Sabe de cada skill, MCP, harness, comando
- **Message optimizer** — construye mensajes óptimos según las limitantes del modelo receptor, comprime, enruta, aplica metodologías fijas
- **Workspace scanner** — al abrir un proyecto, escanea stack, tecnologías, versiones, patrones, reglas. Genera configs específicas del proyecto
- **Router inteligente** — elige el mejor modelo para cada tarea, reformula mensajes para el modelo destino
- **Hybrid CLI + Daemon** — CLI para gestión, daemon para runtime proxy
- **100% open source** — independiente de modelo, servicio, proveedor, IDE

### What Maestro IS NOT

- **No es un IDE** — no compite con OpenCode, VS Code, Cursor, etc.
- **No es un runtime de agentes** — no ejecuta agentes, los configura y dirige
- **No es un framework** — no impone cómo escribir código
- **No almacena prompts del usuario** — gestiona metadata y configs

### Roadmap

| Phase | Name | Description |
|-------|------|-------------|
| **1** | Foundation: Central Config DB | CRUD completo, sync bidireccional unificado, classification scaffolding, CLI completion |
| **2** | Intelligence: Scanner + Router | Escáner de proyectos, daemon proxy, enrutamiento runtime, reformulación de mensajes, compresión automática |
| **3** | Ecosystem: Everywhere | Integración con cualquier IDE/CLI, plugins, marketplace de configs compartidas |

### Design Principles

1. **Data-first** — la BD centralizada es el core. Todo se deriva de tener datos completos y precisos
2. **Model-aware** — cada decisión considera las capacidades y limitantes del modelo real
3. **Portable** — configs y setup funcionan igual en cualquier máquina
4. **Progressive** — Fase 1 resuelve el problema de un dev. Fase 2+ escala
5. **Stateless CLI, stateful daemon** — CLI para gestión, daemon para runtime
6. **Opinionated, not restrictive** — metodologías fijas, pero no bloquea al usuario

## Architecture — 4-Layer

```
CLI (internal/cli/) → Service (sync/routing/heal/...) → Repository (internal/db/) → SQLite
```

- **CLI layer**: Cobra commands in `internal/cli/`, one file per command group (36 files).
  - Entry: `cmd/maestro/main.go` sets up slog, calls `cli.NewRootCmd().Execute()`.
- **Service layer**: `sync`, `routing`, `heal`, `discover`, `audit`, `generator`, `profile`, `mcp`, `classifier`, `compress`.
- **Repository layer**: `internal/db/` — `DBInterface` (40+ methods), `DB` struct implementing it.
  - All services depend on `db.DBInterface`, not concrete `DB`.
- **Models**: `pkg/models/types.go` — all domain types (Provider, Model, Agent, Skill, etc.).

## Package Index (16 internal packages)

| Package | Purpose | Key files |
|---------|---------|-----------|
| `internal/cli/` | All cobra commands | `root.go`, `*_cmd.go`, `models.go`, `providers.go`, etc. |
| `internal/db/` | SQLite data access | `db.go`, `interface.go`, `models.go`, `crud.go`, `routing.go`, `upsert.go`, `agents.go` |
| `internal/sync/` | Import/export OpenCode config | `sync.go` |
| `internal/routing/` | Model selection & routing logic | `router.go` |
| `internal/heal/` | Auto-heal: stale models, missing providers | `heal.go` |
| `internal/discover/` | Model discovery from APIs | `discover.go` |
| `internal/audit/` | Model testing/benchmarking | `audit.go` |
| `internal/config/` | XDG path resolution | `paths.go` |
| `internal/credentials/` | Keyring, file, Bitwarden credential stores | (multiple files) |
| `internal/classifier/` | Model classification | `classifier.go` |
| `internal/generator/` | Config generation | `generator.go` |
| `internal/compress/` | Compress/decompress data | `compress.go` |
| `internal/mcp/` | MCP server integration | `mcp.go` |
| `internal/profile/` | Model profiles | `profile.go` |
| `internal/sources/` | Source data management | `sources.go` |
| `internal/util/` | Utilities (JSONC stripping) | `util.go` |
| `pkg/models/` | Domain types | `types.go` |

## Design Decisions (Codebase Patterns)

### Error handling
- **Wrap ALL errors** with `fmt.Errorf("context: %w", err)`
- Lowercase message, no trailing punctuation
- Sentinel errors only where caller needs type assertion
- Example: `fmt.Errorf("model %q not found", id)`

### DB layer conventions
- **`upsertRow`** — generic CRUD helper in `db/upsert.go`. Accepts table name, id column, and `[]upsertCol`. Handles INSERT ON CONFLICT DO UPDATE automatically. Used for most tables.
- `// kept manual:` comment when upsertRow can't handle something (e.g. `datetime('now')` in `UpsertModel`).
- **COALESCE** in column selection strings (e.g. `COALESCE(api_base,'')`) to avoid NULL scanning issues.
- **`boolToInt`** helper for SQLite boolean columns (maps Go `bool` ↔ `int` 0/1).
- **`scanProvider`/`scanModels`** — dedicated scanner functions that coerce int→bool after scanning.
- **`ModelFilter`** — functional options pattern for `ListModels()`: `StatusActive()`, `HasFC()`, `MinContext(min)`, `Tier(tier)`, etc.

### Service layer conventions
- **Constructor**: `func New(database db.DBInterface) *Service`
- **Compile-time check**: `var _ db.DBInterface = (*DB)(nil)` in `db/db.go`
- Services never import `internal/db` concrete — only `db.DBInterface`.
- Exported methods only, no internal helpers exported.

### CLI conventions
- Root command in `root.go`, subcommands in `*_cmd.go` files.
- `genericCmd("name", "short", "long", runFunc)` — shared helper for simple commands.
- Config loaded via `loadConfig()` in `root.go` — tries CLI flag → env var → XDG default.
- ALWAYS check errors from `cmd.Execute()` and exit via `os.Exit(1)`.
- Slog logger set up once in `main.go`, passed to CLI via context.

### Testing conventions
- **External test packages**: `package cli_test`, `package db_test`
- **SQLite in-memory**: `db.Open(":memory:")`
- **Temp dirs**: `t.TempDir()` for config dirs
- **Cleanup**: `t.Cleanup()` for teardown
- **No mocking framework** — interface-based mocking manually
- **Table-driven** with descriptive names: `TestFuncName_Scenario_Expected`
- `t.Parallel()` on test suite level

### Coverage targets (progressive)
| Tier | Threshold | Packages |
|------|-----------|----------|
| Critical | 80% | `internal/db/`, `internal/sync/` |
| Core | 60% | `internal/routing/`, `internal/heal/`, `internal/discover/`, `internal/audit/` |
| CLI | 40% | `internal/cli/` |
| Generated/thin | 0% | `cmd/`, `pkg/models/`, `internal/config/`, `internal/util/` |

## Methodology

### SDD-TDD Hybrid
| Change size | Process |
|-------------|---------|
| >3 files, feature | SDD cycle: explore → design → tasks → TDD apply → verify → archive |
| 1-2 files, fix | Direct TDD: test → impl → verify → commit |

### TDD Rules (strict)
- NEVER write all tests first (horizontal slices anti-pattern)
- ONE test → ONE impl → repeat (vertical tracer bullets)
- Test public interfaces only
- Enough code to pass current test — no speculative features

### Definition of Done
- [ ] `go build ./...` passes
- [ ] `go vet ./...` passes
- [ ] `go test -race ./...` passes
- [ ] Coverage doesn't decrease below per-package thresholds (see `check-coverage`)
- [ ] No TODOs, stubs, placeholders, or dead code
- [ ] Follows existing code patterns (error wrapping, boolToInt, functional options, etc.)
- [ ] Conventional commit: `type(scope): description in present tense`
- [ ] Saved to Engram with `mem_save`

## Commands

```bash
make verify    # build + vet + test-race + coverage — RUN BEFORE EVERY COMMIT
make test      # go test -v ./...
make lint      # go vet ./... (golangci-lint)
make build     # go build -o maestro ./cmd/maestro/
make precommit # quick build + vet
make coverage  # go test -race -coverprofile=coverage.out
make install-hooks # git config core.hooksPath .githooks
```

## Security & Config

- **Keyring**: `go-keyring` for API keys — NOT in env vars or files.
- **Credentials package**: `internal/credentials/` supports keyring, JSON file, and Bitwarden backends.
- **Paths**: XDG Base Directory — `XDG_CONFIG_HOME`/maestro/, `XDG_DATA_HOME`/maestro/.
- **DB location**: `~/.local/share/maestro/maestro.db` by default (WAL mode).
- **No secrets in config files** — keyring is the primary credential store.

## Verification

Do NOT claim work is done without running verification. Always:
1. Run `lsp_diagnostics` on changed files
2. Run `make verify`
3. Check output for errors
4. Only then mark complete

See `CLAUDE.md` and `.opencode/skills/maestro-dev/SKILL.md` for full dev workflow.
See `DEVELOPMENT.md` for complete spec.

## Brand & Visual Identity

> **Brand system**: `docs/brand/logo-system.html` — complete brand reference.

- **Direction**: Ascending — batuta ascendente a +30° (icono principal)
- **Color**: Maestro Gold `#c8a064` (primary), `#e8d5a3` (highlight), `#a08050` (shadow)
- **Typography**: Playfair Display italic (wordmark), Inter (UI), DM Mono (terminal)
- **Asset types**: icon SVG, wordmark lockup, favicon (SVG + dark/light), terminal ASCII
- **Guidelines**: Do/don't usage cards in `docs/brand/logo-system.html`
- **Variants**: dark, light, mono for every logo variant
- **Exploration history**: `docs/brand/logo-direction-variants.html` (4 direction options)
- All new visual assets MUST follow the Maestro Gold palette and Ascending direction.
