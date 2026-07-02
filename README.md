# maestro — OpenCode Infrastructure Manager

`maestro` is a CLI tool for managing [OpenCode](https://github.com/opencode-ai/opencode) configuration — providers, models, agents, skills, MCP servers, routing rules, and runtime preferences. It uses a local SQLite database as the source of truth and can sync bidirectionally with `opencode.jsonc`.

## Features

- **Provider management** — add, update, remove LLM providers (OpenAI, Anthropic, Mistral, Groq, Google, GitHub Models, Cerebras, custom)
- **Model discovery** — fetch model catalogs from provider APIs and populate the local registry with capabilities, pricing, context windows, and metadata
- **Model profiling** — test models for streaming throughput, structured output support, and context window estimation
- **Config generation** — produce `opencode.jsonc` from the database, merging with existing config to preserve hand-written sections
- **Bidirectional sync** — import existing `opencode.jsonc` into the database or export the database back to config files
- **Agent management** — create and update agent definitions, generate agent prompt files
- **Skill/MCP/LSP management** — track installed skills, MCP servers, and language server configurations
- **Intelligent routing** — score and select the best model per task type (coding, reasoning, vision, long context) based on capabilities, latency, cost, and budget constraints
- **Health & healing** — auto-detect stale models, providers with no active models, and DB integrity issues
- **Budget management** — set daily spending limits and preferred pricing tier (free-only, budget, quality)
- **Source management** — import skills/agents/commands from Git-based remote registries
- **Credential management** — store and retrieve API keys via system keyring, Bitwarden, or encrypted files
- **Provider verification** — test API connectivity and compare available models against the local registry
- **Task classification** — classify user requests into task types for routing
- **Execution logging** — track agent/model usage, tokens, duration, and success/failure
- **Snapshots & compression** — capture and compress config state

## Installation

### Prerequisites

- Go 1.25.0 or later

### From source

```bash
git clone https://github.com/reeinharrrd/maestro.git
cd maestro
make build
```

The binary is built at `./maestro`. Move it to a directory in your `$PATH`:

```bash
cp maestro ~/.local/bin/
```

### Build for multiple platforms

```bash
make build-all
```

Produces binaries for Linux, macOS (Intel + Apple Silicon), and Windows (x86_64).

## Quick Start

```bash
# Initialize the database and seed default providers
maestro init

# Set your API keys (e.g. in ~/.config/opencode/opencode.env)
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."

# Inject keys from OpenCode auth.json
maestro keys

# Discover available models from providers
maestro discover

# Generate opencode.jsonc from the database
maestro generate

# Show the current status
maestro status

# Verify provider connectivity
maestro verify --live
```

## Commands

| Command | Description |
|---------|-------------|
| `init` | Bootstrap the database, run migrations, seed default providers and routing rules |
| `discover` | Fetch model catalogs from all configured providers and upsert models into the DB |
| `generate` | Generate `opencode.jsonc` and `maestro-state.json` from the database |
| `sync` | Import existing `opencode.jsonc` into the database (bidirectional) |
| `status` | Show a summary of providers, models, routes, agents, and budget |
| `query` | Run raw SQL queries against the database |
| `providers` | Manage providers (list, add, update, remove) — auto-syncs to config |
| `models` | Manage models (list, search, deprecate, tag) |
| `routes` | Manage routing rules and re-assign models per task type |
| `agents` | Manage agent definitions and generate agent prompt files |
| `skills` | List skills managed by maestro |
| `mcp` | List MCP servers managed by maestro |
| `lsp` | List LSP server configurations |
| `profile` | Profile models — test streaming throughput, structured output, context window |
| `heal` | Run diagnostics and auto-fix config issues (stale models, DB integrity) |
| `audit` | Run comprehensive live tests against provider models |
| `daily` | Run the daily health check pipeline |
| `verify` | Verify provider API connectivity and model availability |
| `classify` | Classify a task description into a routing category |
| `budget` | View or update token budget configuration |
| `prefs` | View or update preference key-value pairs |
| `credentials` | Manage API key storage (keyring, Bitwarden, file) |
| `keys` | Inject API keys from OpenCode auth.json into the environment |
| `sources` | Manage remote registry sources for skills/agents/commands |
| `source-items` | Manage individual source items (list, import registry, report) |
| `snapshots` | Manage config snapshots |
| `compress` | Compress snapshots to save space |
| `exec-log` | View execution log history |
| `doctor` | Run system diagnostics |
| `validate` | Validate the database and config integrity |
| `configpath` | Show OpenCode config file paths |
| `completion` | Generate shell completion scripts |

### Global flags

| Flag | Description |
|------|-------------|
| `--db` | Path to the SQLite database (default: `~/.config/opencode/opencode-kit.db`) |
| `--verbose` | Enable verbose logging |

## Configuration

### Database

maestro stores its state in a SQLite database at `~/.config/opencode/opencode-kit.db` by default. The path can be overridden with the `--db` flag or the `OPENCODE_KIT_DB` environment variable.

### Environment variables

| Variable | Description |
|----------|-------------|
| `OPENCODE_CONFIG_DIR` | Override the OpenCode config directory (default: `~/.config/opencode`) |
| `OPENCODE_DATA_DIR` | Override the maestro data directory (default: `~/.local/share/maestro`) |
| `OPENCODE_CACHE_DIR` | Override the maestro cache directory (default: `~/.cache/maestro`) |
| `OPENCODE_KIT_DB` | Override the database path |

API keys are read from environment variables (e.g. `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`) or injected from `~/.local/share/opencode/auth.json` via `maestro keys`.

### XDG directory layout

```
~/.config/opencode/
  opencode.jsonc         # OpenCode configuration (generated by maestro)
  opencode-kit.db        # maestro SQLite database
  maestro-state.json        # maestro state summary
  agents/                # Generated agent prompt files
  commands/              # Generated command files

~/.local/share/maestro/
  sources/               # Git-cloned remote registries
  skills/                # Installed skills
  credentials/           # Credential storage

~/.cache/maestro/           # Cache directory
```

## Architecture

maestro follows a clean layered architecture:

```
cmd/maestro/main.go         # Entry point
internal/cli/            # Cobra commands (36 files)
internal/db/             # SQLite database layer (migrations, CRUD, seeding)
internal/discover/       # Model discovery from provider APIs
internal/generator/      # opencode.jsonc generation
internal/sync/           # Bidirectional config ↔ DB sync
internal/routing/        # Intelligent model selection per task type
internal/heal/           # Auto-healing diagnostics
internal/profile/        # Model profiling (streaming, structured output, context)
internal/sources/        # Remote registry source management
internal/classifier/     # Task classification
internal/audit/          # Live model testing
internal/credentials/    # Credential storage backends
internal/compress/       # Snapshot compression
internal/config/         # XDG path resolution
internal/mcp/            # MCP server utilities
internal/util/           # Shared utilities
pkg/models/              # Shared model types (Provider, Model, Agent, Skill, MCP, etc.)
```

### Key concepts

- **Provider** — an API service that hosts models (OpenAI, Anthropic, etc.). Each provider has a base URL, catalog URL for model discovery, and an environment variable name for the API key (only the name is stored, never the key value).
- **Model** — a specific LLM identified by `provider_id/model_name`. Models have capabilities (function calling, vision, reasoning, streaming, structured output), pricing, latency metrics, and status tracking.
- **Routing rule** — maps a task type to the best available model based on scoring that considers context window, capabilities, latency, cost, and budget preferences.
- **Agent** — an OpenCode agent definition with a model assignment, mode (primary/subagent), temperature, color, and permission settings.
- **Source** — a Git remote that provides skills, agents, and commands as installable items.

## Development

### Prerequisites

- Go 1.25.0+
- Make

### Make targets

| Target | Description |
|--------|-------------|
| `make build` | Build the binary to `./maestro` |
| `make build-all` | Cross-compile for linux/darwin/windows, amd64/arm64 |
| `make test` | Run all tests |
| `make test-race` | Run tests with the race detector |
| `make lint` | Run `go vet ./...` |
| `make verify` | Full pipeline: build + vet + test-race + coverage check |
| `make precommit` | Quick: build + vet + test (for rapid iteration) |
| `make coverage` | Run tests with coverage and display per-function report |
| `make coverage-check` | Run tests with coverage and check against thresholds |
| `make clean` | Remove build artifacts |
| `make install-hooks` | Enable the pre-commit hook (`make verify` before every commit) |

### Testing

maestro uses standard Go testing with table-driven tests and external test packages. Tests use in-memory SQLite databases (`:memory:`) where applicable.

### Coverage targets

| Tier | Threshold | Packages |
|------|-----------|----------|
| Critical | 80% | `internal/db`, `internal/sync` |
| Core | 60% | `internal/routing`, `internal/heal` |
| CLI | 40% | `internal/cli` |
| Generated/thin | 0% | `cmd/`, `pkg/models/` |

### Pre-commit hook

Enable with `make install-hooks`. The hook runs `go build ./...` → `go vet ./...` → `go test -race ./...` before every commit.

## Database

The SQLite database is managed via [golang-migrate](https://github.com/golang-migrate/migrate) with embedded migration files in `internal/db/migrations/`. On first open, migrations run automatically and default data is seeded:

- **6 routing rules** — coding_complex, coding_fast, reasoning, vision, long_context, fastest
- **Default budget** — $0.50/day, free-tier preference
- **Seed providers** — embedded via `seed/providers.json`

## Security

- **API keys**: only the environment variable name is stored in the database (e.g. `MISTRAL_API_KEY`), never the actual key value
- **Credentials**: optional encrypted storage via system keyring, Bitwarden, or local encrypted files
- **Auth injection**: `maestro keys` reads from `~/.local/share/opencode/auth.json` and injects matching keys into the process environment — keys are never persisted in the maestro database

## License

MIT
