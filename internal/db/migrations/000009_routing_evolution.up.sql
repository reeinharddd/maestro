ALTER TABLE routing_rules ADD COLUMN priority_weight INTEGER DEFAULT 0;
ALTER TABLE routing_rules ADD COLUMN enabled INTEGER DEFAULT 1;
ALTER TABLE routing_rules ADD COLUMN created_at TEXT;
ALTER TABLE routing_rules ADD COLUMN updated_at TEXT;

CREATE TABLE IF NOT EXISTS model_budgets (
    model_id TEXT PRIMARY KEY REFERENCES models(id) ON DELETE CASCADE,
    max_cost_per_period REAL NOT NULL DEFAULT 0,
    period_hours INTEGER NOT NULL DEFAULT 24,
    current_cost REAL NOT NULL DEFAULT 0,
    reset_at TEXT NOT NULL DEFAULT (datetime('now')),
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
