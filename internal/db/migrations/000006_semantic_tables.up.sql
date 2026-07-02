CREATE TABLE IF NOT EXISTS skill_capabilities (
    skill_id TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    confidence REAL NOT NULL DEFAULT 0.5,
    PRIMARY KEY (skill_id, name)
);

CREATE TABLE IF NOT EXISTS agent_capabilities (
    agent_id TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    confidence REAL NOT NULL DEFAULT 0.5,
    PRIMARY KEY (agent_id, name)
);

CREATE TABLE IF NOT EXISTS command_capabilities (
    command_id TEXT NOT NULL REFERENCES commands(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    confidence REAL NOT NULL DEFAULT 0.5,
    PRIMARY KEY (command_id, name)
);

CREATE TABLE IF NOT EXISTS mcp_capabilities (
    mcp_id TEXT NOT NULL REFERENCES mcp_servers(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    confidence REAL NOT NULL DEFAULT 0.5,
    PRIMARY KEY (mcp_id, name)
);

CREATE TABLE IF NOT EXISTS source_credentials (
    source_id TEXT NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    backend TEXT NOT NULL DEFAULT 'file',
    config TEXT NOT NULL DEFAULT '{}',
    PRIMARY KEY (source_id)
);

CREATE INDEX IF NOT EXISTS idx_skill_capabilities_skill ON skill_capabilities(skill_id);
CREATE INDEX IF NOT EXISTS idx_agent_capabilities_agent ON agent_capabilities(agent_id);
CREATE INDEX IF NOT EXISTS idx_command_capabilities_command ON command_capabilities(command_id);
CREATE INDEX IF NOT EXISTS idx_mcp_capabilities_mcp ON mcp_capabilities(mcp_id);