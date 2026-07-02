CREATE TABLE projects (
    id TEXT PRIMARY KEY,
    path TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL DEFAULT '',
    detected_at INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at INTEGER NOT NULL DEFAULT (unixepoch()),
    status TEXT NOT NULL DEFAULT 'active',
    source TEXT NOT NULL DEFAULT 'scan'
);

CREATE TABLE detected_stacks (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    language TEXT NOT NULL,
    framework TEXT NOT NULL DEFAULT '',
    version TEXT NOT NULL DEFAULT '',
    builder TEXT NOT NULL DEFAULT '',
    test_runner TEXT NOT NULL DEFAULT '',
    linter TEXT NOT NULL DEFAULT '',
    detected_at INTEGER NOT NULL DEFAULT (unixepoch()),
    confidence REAL NOT NULL DEFAULT 1.0,
    UNIQUE(project_id, language)
);

CREATE TABLE project_configs (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    config_type TEXT NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    generated_at INTEGER NOT NULL DEFAULT (unixepoch()),
    hash TEXT NOT NULL DEFAULT '',
    UNIQUE(project_id, config_type)
);
