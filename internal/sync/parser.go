package sync

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/reeinharrrd/maestro/internal/util"
)

// ── Config types matching opencode.jsonc entity sections ──────────────

// ProviderConfig represents a provider entry in opencode.jsonc.
type ProviderConfig struct {
	Name      string                 `json:"name,omitempty"`
	NPM       string                 `json:"npm,omitempty"`
	Options   map[string]interface{} `json:"options,omitempty"`
	Whitelist []string               `json:"whitelist,omitempty"`
	Models    map[string]interface{} `json:"models,omitempty"`
}

// AgentConfig represents an agent entry in opencode.jsonc.
type AgentConfig struct {
	Model       string                 `json:"model,omitempty"`
	Description string                 `json:"description,omitempty"`
	Mode        string                 `json:"mode,omitempty"`
	Temperature float64                `json:"temperature,omitempty"`
	Color       string                 `json:"color,omitempty"`
	Steps       float64                `json:"steps,omitempty"`
	Prompt      string                 `json:"prompt,omitempty"`
	Permission  map[string]interface{} `json:"permission,omitempty"`
}

// CommandConfig represents a command entry in opencode.jsonc.
type CommandConfig struct {
	Template    string `json:"template,omitempty"`
	Description string `json:"description,omitempty"`
	Agent       string `json:"agent,omitempty"`
	Model       string `json:"model,omitempty"`
	Subtask     bool   `json:"subtask,omitempty"`
}

// MCPConfig represents an MCP server entry in opencode.jsonc.
// Both "environment" (hand-written configs) and "env" (generated configs) are captured.
type MCPConfig struct {
	Type        string                 `json:"type,omitempty"`
	Command     []interface{}          `json:"command,omitempty"`
	URL         string                 `json:"url,omitempty"`
	Enabled     bool                   `json:"enabled,omitempty"`
	Timeout     float64                `json:"timeout,omitempty"`
	Environment map[string]interface{} `json:"environment,omitempty"`
	Env         map[string]interface{} `json:"env,omitempty"`
}

// LSPConfig represents an LSP server entry in opencode.jsonc.
type LSPConfig struct {
	Command        []interface{}          `json:"command,omitempty"`
	Extensions     []interface{}          `json:"extensions,omitempty"`
	Env            map[string]interface{} `json:"env,omitempty"`
	Initialization string                 `json:"initialization,omitempty"`
	Disabled       bool                   `json:"disabled,omitempty"`
}

// OpenCodeConfig represents the full opencode.jsonc structure with typed access
// for every entity section and a Raw map for meta preference iteration.
type OpenCodeConfig struct {
	Schema     string                    `json:"$schema,omitempty"`
	Providers  map[string]ProviderConfig `json:"provider,omitempty"`
	Agents     map[string]AgentConfig    `json:"agent,omitempty"`
	Commands   map[string]CommandConfig  `json:"command,omitempty"`
	MCPServers map[string]MCPConfig      `json:"mcp,omitempty"`

	// LSP can be a boolean (true/false) or an object of server configs.
	LSP json.RawMessage `json:"lsp,omitempty"`

	Skills     map[string]interface{} `json:"skills,omitempty"`
	Compaction map[string]interface{} `json:"compaction,omitempty"`

	// Meta fields — kept as interface{} because their types vary.
	Autoupdate        interface{} `json:"autoupdate,omitempty"`
	DisabledProviders interface{} `json:"disabled_providers,omitempty"`
	EnabledProviders  interface{} `json:"enabled_providers,omitempty"`
	Model             interface{} `json:"model,omitempty"`
	SmallModel        interface{} `json:"small_model,omitempty"`
	Share             interface{} `json:"share,omitempty"`
	Plugin            interface{} `json:"plugin,omitempty"`

	// Raw holds the full decoded config map for meta-key iteration.
	// Populated by ParseOpenCodeConfig.
	Raw map[string]interface{}
}

// LSPEnabled returns true + the boolean value when "lsp" is a boolean.
// Returns false, false when "lsp" is absent or an object.
func (c *OpenCodeConfig) LSPEnabled() (bool, bool) {
	if c.LSP == nil {
		return false, false
	}
	var b bool
	if err := json.Unmarshal(c.LSP, &b); err == nil {
		return b, true
	}
	return false, false
}

// LSPServers returns the parsed LSP server map when "lsp" is an object.
// Returns nil when "lsp" is absent or a boolean.
func (c *OpenCodeConfig) LSPServers() map[string]LSPConfig {
	if c.LSP == nil {
		return nil
	}
	var servers map[string]LSPConfig
	if err := json.Unmarshal(c.LSP, &servers); err == nil {
		return servers
	}
	return nil
}

// ParseOpenCodeConfig reads, strips JSONC comments, and parses an opencode.jsonc file
// into an OpenCodeConfig with full typed access.
func ParseOpenCodeConfig(path string) (*OpenCodeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cleaned := util.StripJSONC(data)

	// First pass: decode into raw map for full access.
	var raw map[string]interface{}
	if err := json.Unmarshal(cleaned, &raw); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Second pass: decode into the typed struct.
	var cfg OpenCodeConfig
	if err := json.Unmarshal(cleaned, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.Raw = raw

	return &cfg, nil
}
