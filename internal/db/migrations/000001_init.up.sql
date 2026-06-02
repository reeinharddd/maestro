CREATE TABLE IF NOT EXISTS providers (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    api_base TEXT,
    catalog_url TEXT,
    key_env TEXT,
    source TEXT NOT NULL DEFAULT 'auto',
    status TEXT NOT NULL DEFAULT 'active',
    enabled INTEGER NOT NULL DEFAULT 1,
    priority INTEGER NOT NULL DEFAULT 0,
    last_synced INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS models (
    id TEXT PRIMARY KEY,
    provider_id TEXT NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    display_name TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    context_window INTEGER NOT NULL DEFAULT 0,
    function_calling INTEGER NOT NULL DEFAULT 0,
    vision INTEGER NOT NULL DEFAULT 0,
    streaming INTEGER NOT NULL DEFAULT 1,
    structured_outputs INTEGER NOT NULL DEFAULT 0,
    latency_p50_ms REAL NOT NULL DEFAULT 0,
    latency_p95_ms REAL NOT NULL DEFAULT 0,
    tokens_per_sec REAL NOT NULL DEFAULT 0,
    pricing_prompt REAL NOT NULL DEFAULT 0,
    pricing_completion REAL NOT NULL DEFAULT 0,
    pricing_cache_read REAL NOT NULL DEFAULT 0,
    tier TEXT NOT NULL DEFAULT 'unknown',
    status TEXT NOT NULL DEFAULT 'untested',
    error_message TEXT NOT NULL DEFAULT '',
    tags TEXT NOT NULL DEFAULT '',
    last_tested INTEGER NOT NULL DEFAULT 0,
    test_count INTEGER NOT NULL DEFAULT 0,
    fail_count INTEGER NOT NULL DEFAULT 0,
    source TEXT NOT NULL DEFAULT 'discovered',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS model_profiles (
    model_id TEXT PRIMARY KEY REFERENCES models(id) ON DELETE CASCADE,
    real_context INTEGER NOT NULL DEFAULT 0,
    max_output INTEGER NOT NULL DEFAULT 0,
    supports_stream INTEGER NOT NULL DEFAULT 0,
    supports_so INTEGER NOT NULL DEFAULT 0,
    stream_tps REAL NOT NULL DEFAULT 0,
    profiled_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS agents (
    id TEXT PRIMARY KEY,
    task_type TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    current_model_id TEXT NOT NULL DEFAULT '',
    fallback_ids TEXT NOT NULL DEFAULT '',
    prompt_file TEXT NOT NULL DEFAULT '',
    temperature REAL NOT NULL DEFAULT 0.7,
    max_steps INTEGER NOT NULL DEFAULT 0,
    permission TEXT NOT NULL DEFAULT '',
    color TEXT NOT NULL DEFAULT '',
    mode TEXT NOT NULL DEFAULT 'subagent',
    hidden INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'active',
    source TEXT NOT NULL DEFAULT 'auto',
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS capabilities (
    agent_id TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (agent_id, name)
);

CREATE TABLE IF NOT EXISTS routes (
    task_key TEXT PRIMARY KEY,
    description TEXT NOT NULL DEFAULT '',
    min_context INTEGER NOT NULL DEFAULT 0,
    needs_fc INTEGER NOT NULL DEFAULT 0,
    needs_vision INTEGER NOT NULL DEFAULT 0,
    max_cost_per_call REAL NOT NULL DEFAULT 0,
    current_model_id TEXT NOT NULL DEFAULT '',
    last_assigned INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS commands (
    id TEXT PRIMARY KEY,
    command_template TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    agent TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    subtask INTEGER NOT NULL DEFAULT 0,
    source TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active'
);

CREATE TABLE IF NOT EXISTS mcp_servers (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL DEFAULT 'local',
    command TEXT,
    url TEXT NOT NULL DEFAULT '',
    enabled INTEGER NOT NULL DEFAULT 0,
    env_vars TEXT NOT NULL DEFAULT '',
    timeout_ms INTEGER NOT NULL DEFAULT 5000,
    source TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS mcp_tools (
    mcp_id TEXT NOT NULL REFERENCES mcp_servers(id) ON DELETE CASCADE,
    tool_name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    schema_json TEXT NOT NULL DEFAULT '{}',
    PRIMARY KEY (mcp_id, tool_name)
);

CREATE TABLE IF NOT EXISTS skills (
    id TEXT PRIMARY KEY,
    source TEXT NOT NULL DEFAULT '',
    source_path TEXT NOT NULL DEFAULT '',
    target_path TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL DEFAULT 'skill',
    status TEXT NOT NULL DEFAULT 'active',
    hash TEXT NOT NULL DEFAULT '',
    last_synced INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS sources (
    id TEXT PRIMARY KEY,
    remote_url TEXT NOT NULL DEFAULT '',
    local_path TEXT NOT NULL DEFAULT '',
    "commit" TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active',
    last_synced INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS source_items (
    id TEXT NOT NULL,
    source_id TEXT NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    type TEXT NOT NULL DEFAULT '',
    source_path TEXT NOT NULL DEFAULT '',
    target_path TEXT NOT NULL DEFAULT '',
    hash TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active',
    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS lsp_servers (
    id TEXT PRIMARY KEY,
    command TEXT NOT NULL DEFAULT '',
    extensions TEXT NOT NULL DEFAULT '',
    env TEXT NOT NULL DEFAULT '',
    initialization TEXT NOT NULL DEFAULT '',
    disabled INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS budget_config (
    id TEXT PRIMARY KEY,
    daily_global_usd REAL NOT NULL DEFAULT 0,
    preferred_tier TEXT NOT NULL DEFAULT 'fast',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS config_fragments (
    id TEXT PRIMARY KEY,
    config_type TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL DEFAULT '',
    hash TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS sync_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    phase TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending',
    details TEXT NOT NULL DEFAULT '',
    duration_ms INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS exec_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    agent TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    task TEXT NOT NULL DEFAULT '',
    tokens_in INTEGER NOT NULL DEFAULT 0,
    tokens_out INTEGER NOT NULL DEFAULT 0,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    success INTEGER NOT NULL DEFAULT 1,
    error TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    hash TEXT NOT NULL DEFAULT '',
    data TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS preferences (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS rate_limits (
    provider_id TEXT NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    requests_per_min INTEGER NOT NULL DEFAULT 60,
    tokens_per_min INTEGER NOT NULL DEFAULT 100000,
    PRIMARY KEY (provider_id)
);

CREATE TABLE IF NOT EXISTS system_info (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS heal_actions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    action_type TEXT NOT NULL DEFAULT '',
    target TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending',
    message TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS routing_rules (
    task_key TEXT PRIMARY KEY,
    description TEXT NOT NULL DEFAULT '',
    min_context INTEGER NOT NULL DEFAULT 0,
    needs_fc INTEGER NOT NULL DEFAULT 0,
    needs_vision INTEGER NOT NULL DEFAULT 0,
    max_cost_per_call REAL NOT NULL DEFAULT 0,
    current_model_id TEXT NOT NULL DEFAULT '',
    fallback_ids TEXT NOT NULL DEFAULT '',
    last_assigned INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS routing_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_key TEXT NOT NULL DEFAULT '',
    selected_model TEXT NOT NULL DEFAULT '',
    candidates TEXT NOT NULL DEFAULT '[]',
    reason TEXT NOT NULL DEFAULT '',
    shadow INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_models_provider ON models(provider_id);
CREATE INDEX IF NOT EXISTS idx_exec_log_executed ON exec_log(created_at);
CREATE INDEX IF NOT EXISTS idx_sync_log_created ON sync_log(created_at);
CREATE INDEX IF NOT EXISTS idx_routing_events_created ON routing_events(created_at);
CREATE INDEX IF NOT EXISTS idx_source_items_source ON source_items(source_id);
CREATE INDEX IF NOT EXISTS idx_heal_actions_status ON heal_actions(status);
