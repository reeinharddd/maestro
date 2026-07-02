package generator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/reeinharrrd/maestro/internal/db"
	"github.com/reeinharrrd/maestro/pkg/models"
)

func TestGenerateConfig_NoIllegalTopLevelKeys(t *testing.T) {
	tmp := t.TempDir()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	if err := d.UpsertProvider(&models.Provider{ID: "mistral", Name: "Mistral", BaseURL: "https://api.mistral.ai/v1", Status: "active"}); err != nil {
		t.Fatalf("upsert provider: %v", err)
	}
	if err := d.UpsertModel(&models.Model{ID: "codestral-2508", DisplayName: "codestral-2508", ProviderID: "mistral", ContextWindow: 256000, FunctionCalling: true, Tier: "free", Status: "active"}); err != nil {
		t.Fatalf("upsert model: %v", err)
	}

	svc := NewService(d, tmp)
	if err := svc.GenerateConfig(); err != nil {
		t.Fatalf("generate: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(tmp, "opencode.jsonc"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("parse: %v", err)
	}

	if _, ok := cfg["maestro"]; ok {
		t.Error("opencode.jsonc must not contain an 'maestro' top-level key — opencode schema rejects it")
	}

	allowed := map[string]bool{
		"$schema":          true,
		"provider":         true,
		"agent":            true,
		"command":          true,
		"mcp":              true,
		"permission":       true,
		"experimental":     true,
		"skills":           true,
		"share":            true,
		"autoupdate":       true,
		"compaction":       true,
		"plugin":           true,
		"watcher":          true,
		"snapshot":         true,
		"default_agent":    true,
		"model":            true,
		"small_model":      true,
		"disabled_providers": true,
		"enabled_providers":  true,
		"tools":            true,
		"theme":            true,
		"keybinds":         true,
		"logLevel":         true,
		"server":           true,
		"shell":            true,
		"instructions":     true,
		"reference":        true,
		"formatter":        true,
		"lsp":              true,
		"attachment":       true,
		"enterprise":       true,
		"tool_output":      true,
		"username":         true,
		"mode":             true,
		"layout":           true,
	}
	for k := range cfg {
		if !allowed[k] {
			t.Errorf("illegal top-level key %q in opencode.jsonc", k)
		}
	}
}

func TestGenerateConfig_ModelSchemaConformsToOpenCode(t *testing.T) {
	tmp := t.TempDir()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	if err := d.UpsertProvider(&models.Provider{ID: "groq", Name: "Groq", Status: "active"}); err != nil {
		t.Fatalf("upsert provider: %v", err)
	}
	if err := d.UpsertModel(&models.Model{
		ID: "llama-3.3-70b-versatile", DisplayName: "llama-3.3-70b-versatile",
		ProviderID: "groq", ContextWindow: 131072, FunctionCalling: true,
		Streaming: true, Vision: true, Tier: "free", Status: "active",
		PricingPrompt: 0.59, PricingCompletion: 0.79,
	}); err != nil {
		t.Fatalf("upsert model: %v", err)
	}
	if err := d.UpsertModelProfile(&models.ModelProfile{
		ModelID: "llama-3.3-70b-versatile", MaxOutput: 32768, SupportsStream: true,
	}); err != nil {
		t.Fatalf("upsert profile: %v", err)
	}

	svc := NewService(d, tmp)
	if err := svc.GenerateConfig(); err != nil {
		t.Fatalf("generate: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(tmp, "opencode.jsonc"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("parse: %v", err)
	}

	prov, _ := cfg["provider"].(map[string]interface{})
	groq, _ := prov["groq"].(map[string]interface{})
	modelsMap, _ := groq["models"].(map[string]interface{})
	llama, _ := modelsMap["llama-3.3-70b-versatile"].(map[string]interface{})

	if _, ok := llama["capabilities"]; ok {
		t.Error("model entry must not have a top-level 'capabilities' key — opencode schema uses 'limit' and 'tool_call' directly")
	}
	if _, ok := llama["context_window"]; ok {
		t.Error("model entry must not have 'context_window' — use 'limit.context' instead")
	}
	if _, ok := llama["function_calling"]; ok {
		t.Error("model entry must not have 'function_calling' — use 'tool_call' boolean instead")
	}
	if _, ok := llama["pricing"]; ok {
		t.Error("model entry must not have 'pricing' — use 'cost.input' and 'cost.output' instead")
	}

	limit, ok := llama["limit"].(map[string]interface{})
	if !ok {
		t.Fatal("model.limit is required by the opencode schema")
	}
	if int(limit["context"].(float64)) != 131072 {
		t.Errorf("limit.context = %v, want 131072", limit["context"])
	}
	if int(limit["output"].(float64)) != 32768 {
		t.Errorf("limit.output = %v, want 32768 (from profile)", limit["output"])
	}

	if llama["tool_call"] != true {
		t.Errorf("tool_call = %v, want true", llama["tool_call"])
	}
	if llama["attachment"] != true {
		t.Errorf("attachment = %v, want true (vision=true)", llama["attachment"])
	}

	cost, ok := llama["cost"].(map[string]interface{})
	if !ok {
		t.Fatal("model.cost is required when emitting cost")
	}
	if _, ok := cost["input"]; !ok {
		t.Error("cost.input required")
	}
	if _, ok := cost["output"]; !ok {
		t.Error("cost.output required")
	}

	modalities, _ := llama["modalities"].(map[string]interface{})
	inputMods, _ := modalities["input"].([]interface{})
	hasImage := false
	for _, m := range inputMods {
		if m.(string) == "image" {
			hasImage = true
		}
	}
	if !hasImage {
		t.Error("modalities.input should include 'image' when vision=true")
	}
}

func TestGenerateConfig_NeverEmitsStatusActive(t *testing.T) {
	tmp := t.TempDir()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	if err := d.UpsertProvider(&models.Provider{ID: "mistral", Status: "active"}); err != nil {
		t.Fatalf("upsert provider: %v", err)
	}
	if err := d.UpsertModel(&models.Model{ID: "ok-1", DisplayName: "ok-1", ProviderID: "mistral", Status: "active", ContextWindow: 1000}); err != nil {
		t.Fatalf("upsert model: %v", err)
	}
	if err := d.UpsertModel(&models.Model{ID: "dep-1", DisplayName: "dep-1", ProviderID: "mistral", Status: "deprecated", ContextWindow: 1000}); err != nil {
		t.Fatalf("upsert model: %v", err)
	}

	svc := NewService(d, tmp)
	if err := svc.GenerateConfig(); err != nil {
		t.Fatalf("generate: %v", err)
	}

	raw, _ := os.ReadFile(filepath.Join(tmp, "opencode.jsonc"))
	var cfg map[string]interface{}
	_ = json.Unmarshal(raw, &cfg)
	modelsMap, _ := cfg["provider"].(map[string]interface{})["mistral"].(map[string]interface{})["models"].(map[string]interface{})

	if m, ok := modelsMap["ok-1"].(map[string]interface{}); ok {
		if s, has := m["status"]; has && s == "active" {
			t.Error("active models must omit the 'status' field — opencode enum is alpha|beta|deprecated, active is implicit")
		}
	}
	if m, ok := modelsMap["dep-1"].(map[string]interface{}); ok {
		if m["status"] != "deprecated" {
			t.Errorf("deprecated model should emit status='deprecated', got %v", m["status"])
		}
	}
}

func TestGenerateConfig_DefaultsAppliedWhenContextUnknown(t *testing.T) {
	tmp := t.TempDir()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	if err := d.UpsertProvider(&models.Provider{ID: "openai", Status: "active"}); err != nil {
		t.Fatalf("upsert provider: %v", err)
	}
	if err := d.UpsertModel(&models.Model{
		ID: "mystery-model", DisplayName: "mystery-model",
		ProviderID: "openai", ContextWindow: 0, FunctionCalling: true, Status: "active",
	}); err != nil {
		t.Fatalf("upsert model: %v", err)
	}

	svc := NewService(d, tmp)
	if err := svc.GenerateConfig(); err != nil {
		t.Fatalf("generate: %v", err)
	}

	raw, _ := os.ReadFile(filepath.Join(tmp, "opencode.jsonc"))
	var cfg map[string]interface{}
	_ = json.Unmarshal(raw, &cfg)

	prov, _ := cfg["provider"].(map[string]interface{})
	openai, _ := prov["openai"].(map[string]interface{})
	modelsMap, _ := openai["models"].(map[string]interface{})
	mystery, _ := modelsMap["mystery-model"].(map[string]interface{})
	limit, _ := mystery["limit"].(map[string]interface{})

	if int(limit["context"].(float64)) != defaultContextWindow {
		t.Errorf("limit.context default = %v, want %d", limit["context"], defaultContextWindow)
	}
	if int(limit["output"].(float64)) != defaultMaxOutput {
		t.Errorf("limit.output default = %v, want %d", limit["output"], defaultMaxOutput)
	}
}

func TestGenerateConfig_ExcludesErrorModelsAndWritesState(t *testing.T) {
	tmp := t.TempDir()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	if err := d.UpsertProvider(&models.Provider{ID: "mistral", Name: "Mistral", Status: "active"}); err != nil {
		t.Fatalf("upsert provider: %v", err)
	}
	if err := d.UpsertModel(&models.Model{ID: "codestral-2508", DisplayName: "codestral-2508", ProviderID: "mistral", ContextWindow: 256000, FunctionCalling: true, Status: "active"}); err != nil {
		t.Fatalf("upsert model: %v", err)
	}
	if err := d.UpsertModel(&models.Model{ID: "pixtral-broken", DisplayName: "pixtral-broken", ProviderID: "mistral", Status: "error", ErrorMessage: "rate_limit"}); err != nil {
		t.Fatalf("upsert model: %v", err)
	}
	if err := d.UpsertRoutingRule(&models.RoutingRule{TaskKey: "coding_complex", CurrentModelID: "mistral/codestral-2508", Description: "Complex coding", NeedsFC: true, MinContext: 100000}); err != nil {
		t.Fatalf("upsert route: %v", err)
	}
	if err := d.UpsertSkill(&models.Skill{ID: "global-rules", Status: "active"}); err != nil {
		t.Fatalf("upsert skill: %v", err)
	}
	if err := d.UpsertSkill(&models.Skill{ID: "disabled-skill", Status: "deprecated"}); err != nil {
		t.Fatalf("upsert skill: %v", err)
	}

	svc := NewService(d, tmp)
	if err := svc.GenerateConfig(); err != nil {
		t.Fatalf("generate: %v", err)
	}

	raw, _ := os.ReadFile(filepath.Join(tmp, "opencode.jsonc"))
	var cfg map[string]interface{}
	_ = json.Unmarshal(raw, &cfg)

	prov, _ := cfg["provider"].(map[string]interface{})
	mistral, _ := prov["mistral"].(map[string]interface{})
	whitelist, _ := mistral["whitelist"].([]interface{})
	joined := ""
	for _, w := range whitelist {
		joined += w.(string) + ","
	}
	if strings.Contains(joined, "pixtral-broken") {
		t.Error("error models should not be in whitelist")
	}
	if !strings.Contains(joined, "codestral-2508") {
		t.Error("active models should be in whitelist")
	}

	stateRaw, err := os.ReadFile(filepath.Join(tmp, "maestro-state.json"))
	if err != nil {
		t.Fatalf("sidecar state file missing: %v", err)
	}
	var state map[string]interface{}
	if err := json.Unmarshal(stateRaw, &state); err != nil {
		t.Fatalf("state parse: %v", err)
	}
	if int(state["active_models"].(float64)) != 1 {
		t.Errorf("state.active_models = %v, want 1", state["active_models"])
	}
	if int(state["error_models"].(float64)) != 1 {
		t.Errorf("state.error_models = %v, want 1", state["error_models"])
	}
	skills, _ := state["skills"].([]interface{})
	if len(skills) != 1 {
		t.Errorf("state.skills count = %d, want 1 (active only)", len(skills))
	}
	routes, _ := state["routes"].(map[string]interface{})
	if _, ok := routes["coding_complex"]; !ok {
		t.Error("state.routes missing coding_complex")
	}
}

func TestGenerateConfig_CommandSectionConformsToSchema(t *testing.T) {
	tmp := t.TempDir()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	if err := d.UpsertCommand(&models.Command{
		ID: "review", Template: "Review the following code:\n$1",
		Description: "Code review", Agent: "reviewer", Model: "groq/llama-3.3-70b-versatile", Subtask: true,
		Status: "active",
	}); err != nil {
		t.Fatalf("upsert command: %v", err)
	}

	svc := NewService(d, tmp)
	if err := svc.GenerateConfig(); err != nil {
		t.Fatalf("generate: %v", err)
	}

	raw, _ := os.ReadFile(filepath.Join(tmp, "opencode.jsonc"))
	var cfg map[string]interface{}
	_ = json.Unmarshal(raw, &cfg)
	cmds, _ := cfg["command"].(map[string]interface{})
	review, _ := cmds["review"].(map[string]interface{})

	if _, ok := review["template"]; !ok {
		t.Error("command.template is required by the opencode schema")
	}
	if _, ok := review["command_template"]; ok {
		t.Error("command must use 'template' key, not 'command_template'")
	}
}

func TestGenerateConfig_MCPLocalCommandIsArray(t *testing.T) {
	tmp := t.TempDir()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	if err := d.UpsertMCP(&models.MCPServer{
		ID: "filesystem", Type: "local",
		Command: `["npx","-y","@modelcontextprotocol/server-filesystem","/tmp"]`,
		Enabled: true, Timeout: 10000,
	}); err != nil {
		t.Fatalf("upsert mcp: %v", err)
	}

	svc := NewService(d, tmp)
	if err := svc.GenerateConfig(); err != nil {
		t.Fatalf("generate: %v", err)
	}

	raw, _ := os.ReadFile(filepath.Join(tmp, "opencode.jsonc"))
	var cfg map[string]interface{}
	_ = json.Unmarshal(raw, &cfg)
	mcps, _ := cfg["mcp"].(map[string]interface{})
	fs, _ := mcps["filesystem"].(map[string]interface{})

	if fs["type"] != "local" {
		t.Errorf("mcp.type = %v, want 'local'", fs["type"])
	}
	cmd, ok := fs["command"].([]interface{})
	if !ok {
		t.Fatal("local mcp.command must be an array per opencode schema")
	}
	if len(cmd) != 4 || cmd[0].(string) != "npx" {
		t.Errorf("mcp.command = %v, want npx array", cmd)
	}
	if fs["enabled"] != true {
		t.Errorf("mcp.enabled = %v, want true", fs["enabled"])
	}
}

func TestMerge_ManagedKeysOverrideExisting(t *testing.T) {
	gen := map[string]interface{}{
		"$schema":            "https://opencode.ai/config.json",
		"provider":           map[string]interface{}{"groq": "x"},
		"agent":              map[string]interface{}{"build": "y"},
		"command":            map[string]interface{}{"foo": "z"},
		"mcp":                map[string]interface{}{"a": true},
		"permission":         map[string]interface{}{},
		"experimental":       map[string]interface{}{},
		"lsp":                true,
		"plugin":             []string{"opencode-notify"},
		"skills":             map[string]interface{}{"paths": []string{"/sync/skills"}},
		"autoupdate":         false,
		"disabled_providers": []string{"sync"},
		"small_model":        "sync/model",
		"model":              "sync/codestral",
		"share":              "auto",
		"compaction":         map[string]interface{}{"auto": false},
	}
	existing := map[string]interface{}{
		"provider":           map[string]interface{}{"groq": "OLD_SHOULD_LOSE"},
		"unknown_user_key":   "must_survive",
		"lsp":                "OLD_SHOULD_LOSE",
		"autoupdate":         "OLD_SHOULD_LOSE",
		"small_model":        "OLD_SHOULD_LOSE",
	}
	s := &Service{}
	merged := s.mergeWithExisting(gen, existing, nil)
	if merged["unknown_user_key"] != "must_survive" {
		t.Errorf("unknown_user_key should survive, got %v", merged["unknown_user_key"])
	}
	if merged["provider"].(map[string]interface{})["groq"] != "x" {
		t.Errorf("generator-managed provider should win, got %v", merged["provider"])
	}
	for _, k := range []string{"lsp", "plugin", "skills", "autoupdate", "disabled_providers", "small_model", "model", "share", "compaction"} {
		if v, exists := merged[k]; !exists || v == "OLD_SHOULD_LOSE" {
			t.Errorf("managed key %q should be generator value, got %v", k, v)
		}
	}
	if _, ok := merged["$schema"]; !ok {
		t.Errorf("$schema should be present")
	}
}

func TestGenerateConfig_DisabledProviderFilter(t *testing.T) {
	tmp := t.TempDir()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	if err := d.UpsertProvider(&models.Provider{ID: "openai", Name: "OpenAI", Status: "active"}); err != nil {
		t.Fatalf("upsert provider: %v", err)
	}
	if err := d.UpsertModel(&models.Model{ID: "openai/gpt-4o", DisplayName: "gpt-4o", ProviderID: "openai", ContextWindow: 128000, Status: "active"}); err != nil {
		t.Fatalf("upsert model: %v", err)
	}
	if err := d.SetPreference("config/disabled_providers", `["openai"]`); err != nil {
		t.Fatalf("set pref: %v", err)
	}

	svc := NewService(d, tmp)
	if err := svc.GenerateConfig(); err != nil {
		t.Fatalf("generate: %v", err)
	}

	raw, _ := os.ReadFile(filepath.Join(tmp, "opencode.jsonc"))
	var cfg map[string]interface{}
	_ = json.Unmarshal(raw, &cfg)
	prov, _ := cfg["provider"].(map[string]interface{})
	if _, ok := prov["openai"]; ok {
		t.Error("disabled provider 'openai' must not appear in config")
	}
}

func TestGenerateConfig_EnabledProviderFilter(t *testing.T) {
	tmp := t.TempDir()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	if err := d.UpsertProvider(&models.Provider{ID: "openai", Name: "OpenAI", Status: "active"}); err != nil {
		t.Fatalf("upsert provider: %v", err)
	}
	if err := d.UpsertModel(&models.Model{ID: "openai/gpt-4o", DisplayName: "gpt-4o", ProviderID: "openai", ContextWindow: 128000, Status: "active"}); err != nil {
		t.Fatalf("upsert model: %v", err)
	}
	if err := d.SetPreference("config/enabled_providers", `["mistral"]`); err != nil {
		t.Fatalf("set pref: %v", err)
	}

	svc := NewService(d, tmp)
	if err := svc.GenerateConfig(); err != nil {
		t.Fatalf("generate: %v", err)
	}

	raw, _ := os.ReadFile(filepath.Join(tmp, "opencode.jsonc"))
	var cfg map[string]interface{}
	_ = json.Unmarshal(raw, &cfg)
	prov, _ := cfg["provider"].(map[string]interface{})
	if _, ok := prov["openai"]; ok {
		t.Error("non-enabled provider 'openai' must not appear when enabled_providers filter is set")
	}
}

func TestGenerateConfig_ProviderWithOptions(t *testing.T) {
	tmp := t.TempDir()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	if err := d.UpsertProvider(&models.Provider{
		ID: "custom", Name: "Custom", Status: "active",
		BaseURL: "https://custom.example.com/v1", TimeoutMs: 60000,
		HeaderTimeoutMs: 30000, ChunkTimeoutMs: 10000,
		EnterpriseURL: "https://enterprise.custom.example.com", SetCacheKey: true,
	}); err != nil {
		t.Fatalf("upsert provider: %v", err)
	}
	if err := d.UpsertModel(&models.Model{ID: "custom/model-1", DisplayName: "model-1", ProviderID: "custom", ContextWindow: 1000, Status: "active"}); err != nil {
		t.Fatalf("upsert model: %v", err)
	}

	svc := NewService(d, tmp)
	if err := svc.GenerateConfig(); err != nil {
		t.Fatalf("generate: %v", err)
	}

	raw, _ := os.ReadFile(filepath.Join(tmp, "opencode.jsonc"))
	var cfg map[string]interface{}
	_ = json.Unmarshal(raw, &cfg)
	p, _ := cfg["provider"].(map[string]interface{})["custom"].(map[string]interface{})
	opts, _ := p["options"].(map[string]interface{})
	if opts == nil {
		t.Fatal("provider options must be present")
	}
	if opts["baseURL"] != "https://custom.example.com/v1" {
		t.Errorf("baseURL = %v", opts["baseURL"])
	}
	if int(opts["timeout"].(float64)) != 60000 {
		t.Errorf("timeout = %v", opts["timeout"])
	}
	if opts["setCacheKey"] != true {
		t.Errorf("setCacheKey missing")
	}
}

func TestGenerateConfig_AgentSection(t *testing.T) {
	tmp := t.TempDir()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	if err := d.UpsertAgent(&models.Agent{
		ID: "build", Description: "Build agent", CurrentModelID: "openai/gpt-4o",
		Mode: "primary", Temperature: 0.3, Color: "blue", MaxSteps: 50,
		Status: "active", Source: "test",
	}); err != nil {
		t.Fatalf("upsert agent: %v", err)
	}
	if err := d.UpsertAgent(&models.Agent{
		ID: "inactive-agent", Status: "deprecated", Source: "test",
	}); err != nil {
		t.Fatalf("upsert inactive agent: %v", err)
	}

	svc := NewService(d, tmp)
	if err := svc.GenerateConfig(); err != nil {
		t.Fatalf("generate: %v", err)
	}

	raw, _ := os.ReadFile(filepath.Join(tmp, "opencode.jsonc"))
	var cfg map[string]interface{}
	_ = json.Unmarshal(raw, &cfg)

	agents, _ := cfg["agent"].(map[string]interface{})
	build, _ := agents["build"].(map[string]interface{})
	if build == nil {
		t.Fatal("agent 'build' must be present")
	}
	if build["model"] != "openai/gpt-4o" {
		t.Errorf("agent model = %v", build["model"])
	}
	if build["mode"] != "primary" {
		t.Errorf("agent mode = %v", build["mode"])
	}
	if _, ok := agents["inactive-agent"]; ok {
		t.Error("inactive agent must not appear in config")
	}
}

func TestGenerateConfig_AgentWithPermission(t *testing.T) {
	tmp := t.TempDir()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	if err := d.UpsertAgent(&models.Agent{
		ID: "restricted", Description: "Restricted agent",
		Mode: "subagent", Status: "active", Source: "test",
		Permission: `{"allow": ["read", "write"]}`,
	}); err != nil {
		t.Fatalf("upsert agent: %v", err)
	}

	svc := NewService(d, tmp)
	if err := svc.GenerateConfig(); err != nil {
		t.Fatalf("generate: %v", err)
	}

	raw, _ := os.ReadFile(filepath.Join(tmp, "opencode.jsonc"))
	var cfg map[string]interface{}
	_ = json.Unmarshal(raw, &cfg)
	ag, _ := cfg["agent"].(map[string]interface{})["restricted"].(map[string]interface{})
	perm, _ := ag["permission"].(map[string]interface{})
	if perm == nil {
		t.Fatal("agent permission must be present")
	}
	allow, _ := perm["allow"].([]interface{})
	if len(allow) != 2 {
		t.Errorf("permission.allow length = %d, want 2", len(allow))
	}
}

func TestGenerateConfig_RemoteMCP(t *testing.T) {
	tmp := t.TempDir()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	if err := d.UpsertMCP(&models.MCPServer{
		ID: "remote-api", Type: "remote",
		URL: "https://example.com/mcp", Enabled: true, Timeout: 15000,
	}); err != nil {
		t.Fatalf("upsert mcp: %v", err)
	}

	svc := NewService(d, tmp)
	if err := svc.GenerateConfig(); err != nil {
		t.Fatalf("generate: %v", err)
	}

	raw, _ := os.ReadFile(filepath.Join(tmp, "opencode.jsonc"))
	var cfg map[string]interface{}
	_ = json.Unmarshal(raw, &cfg)
	mcps, _ := cfg["mcp"].(map[string]interface{})
	m, _ := mcps["remote-api"].(map[string]interface{})
	if m == nil {
		t.Fatal("remote MCP must be present")
	}
	if m["type"] != "remote" {
		t.Errorf("type = %v, want remote", m["type"])
	}
	if m["url"] != "https://example.com/mcp" {
		t.Errorf("url = %v", m["url"])
	}
}

func TestGenerateConfig_LSPSection(t *testing.T) {
	tmp := t.TempDir()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	if err := d.UpsertLSPServer(&models.LSPServer{
		ID: "gopls",
		Command: `["gopls","serve"]`,
		Extensions: `[".go"]`,
		Initialization: `{"go_version": "1.25"}`,
		Disabled: true,
	}); err != nil {
		t.Fatalf("upsert lsp: %v", err)
	}

	svc := NewService(d, tmp)
	if err := svc.GenerateConfig(); err != nil {
		t.Fatalf("generate: %v", err)
	}

	raw, _ := os.ReadFile(filepath.Join(tmp, "opencode.jsonc"))
	var cfg map[string]interface{}
	_ = json.Unmarshal(raw, &cfg)

	lsps, _ := cfg["lsp"].(map[string]interface{})
	gopls, _ := lsps["gopls"].(map[string]interface{})
	if gopls == nil {
		t.Fatal("lsp section must contain gopls")
	}
	if gopls["disabled"] != true {
		t.Errorf("disabled = %v, want true", gopls["disabled"])
	}
	cmd, _ := gopls["command"].([]interface{})
	if len(cmd) != 2 || cmd[0] != "gopls" {
		t.Errorf("command = %v", cmd)
	}
}

func TestGenerateConfig_LSPWithEnv(t *testing.T) {
	tmp := t.TempDir()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	if err := d.UpsertLSPServer(&models.LSPServer{
		ID: "ruff",
		Command: `["ruff","server"]`,
		Env: `{"RUFF_TRACE": "true"}`,
	}); err != nil {
		t.Fatalf("upsert lsp: %v", err)
	}

	svc := NewService(d, tmp)
	if err := svc.GenerateConfig(); err != nil {
		t.Fatalf("generate: %v", err)
	}

	raw, _ := os.ReadFile(filepath.Join(tmp, "opencode.jsonc"))
	var cfg map[string]interface{}
	_ = json.Unmarshal(raw, &cfg)
	lsps, _ := cfg["lsp"].(map[string]interface{})
	ruff, _ := lsps["ruff"].(map[string]interface{})
	env, _ := ruff["env"].(map[string]interface{})
	if env == nil || env["RUFF_TRACE"] != "true" {
		t.Errorf("lsp env = %v, want RUFF_TRACE=true", env)
	}
}

func TestGenerateConfig_PluginSection(t *testing.T) {
	tmp := t.TempDir()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	if err := d.UpsertSkill(&models.Skill{
		ID: "my-plugin", Type: "plugin", Status: "active",
		Source: "npm", SourcePath: "@org/plugin",
	}); err != nil {
		t.Fatalf("upsert plugin skill: %v", err)
	}
	if err := d.UpsertSkill(&models.Skill{
		ID: "my-skill", Type: "skill", Status: "active",
	}); err != nil {
		t.Fatalf("upsert non-plugin skill: %v", err)
	}

	svc := NewService(d, tmp)
	if err := svc.GenerateConfig(); err != nil {
		t.Fatalf("generate: %v", err)
	}

	raw, _ := os.ReadFile(filepath.Join(tmp, "opencode.jsonc"))
	var cfg map[string]interface{}
	_ = json.Unmarshal(raw, &cfg)

	plugins, _ := cfg["plugin"].(map[string]interface{})
	p, _ := plugins["my-plugin"].(map[string]interface{})
	if p == nil {
		t.Fatal("plugin section must contain my-plugin")
	}
	if p["source"] != "npm" {
		t.Errorf("plugin source = %v, want npm", p["source"])
	}
	if _, ok := plugins["my-skill"]; ok {
		t.Error("non-plugin skill must not appear in plugin section")
	}
}

func TestGenerateConfig_ModelWithStatusPaid(t *testing.T) {
	tmp := t.TempDir()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	if err := d.UpsertProvider(&models.Provider{ID: "openai", Status: "active"}); err != nil {
		t.Fatalf("upsert provider: %v", err)
	}
	if err := d.UpsertModel(&models.Model{
		ID: "openai/gpt-4o", DisplayName: "gpt-4o", ProviderID: "openai",
		ContextWindow: 128000, Status: "paid", Tier: "paid",
	}); err != nil {
		t.Fatalf("upsert model: %v", err)
	}

	svc := NewService(d, tmp)
	if err := svc.GenerateConfig(); err != nil {
		t.Fatalf("generate: %v", err)
	}

	raw, _ := os.ReadFile(filepath.Join(tmp, "opencode.jsonc"))
	var cfg map[string]interface{}
	_ = json.Unmarshal(raw, &cfg)
	m, _ := cfg["provider"].(map[string]interface{})["openai"].(map[string]interface{})["models"].(map[string]interface{})["gpt-4o"].(map[string]interface{})
	if m["status"] != "alpha" {
		t.Errorf("paid model status = %v, want alpha", m["status"])
	}
}

func TestGenerateConfig_ModelWithFullFields(t *testing.T) {
	tmp := t.TempDir()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	if err := d.UpsertProvider(&models.Provider{ID: "test", Status: "active"}); err != nil {
		t.Fatalf("upsert provider: %v", err)
	}
	if err := d.UpsertModel(&models.Model{
		ID: "test/full", DisplayName: "full", ProviderID: "test",
		ContextWindow: 100000, MaxOutput: 16384,
		Description: "A full model", Architecture: "transformer",
		RecommendedUse: "coding", Family: "gpt", ReleaseDate: "2025-01-01",
		Aliases: `["full-v1","full-latest"]`,
		Experimental: true, Reasoning: true, DefaultTemp: 1.0,
		PricingPrompt: 10, PricingCompletion: 30,
		PricingCacheRead: 5, PricingCacheWrite: 15,
		Vision: true, Audio: false, OCR: true,
		ModalitiesInput: `["text","image"]`,
		ModalitiesOutput: `["text"]`,
		Tier: "paid", Status: "beta",
	}); err != nil {
		t.Fatalf("upsert model: %v", err)
	}

	svc := NewService(d, tmp)
	if err := svc.GenerateConfig(); err != nil {
		t.Fatalf("generate: %v", err)
	}

	raw, _ := os.ReadFile(filepath.Join(tmp, "opencode.jsonc"))
	var cfg map[string]interface{}
	_ = json.Unmarshal(raw, &cfg)
	m, _ := cfg["provider"].(map[string]interface{})["test"].(map[string]interface{})["models"].(map[string]interface{})["full"].(map[string]interface{})

	if m["description"] != "A full model" {
		t.Errorf("description = %v", m["description"])
	}
	if m["architecture"] != "transformer" {
		t.Errorf("architecture = %v", m["architecture"])
	}
	if m["experimental"] != true {
		t.Errorf("experimental missing")
	}
	if m["reasoning"] != true {
		t.Errorf("reasoning missing")
	}
	aliases, _ := m["aliases"].([]interface{})
	if len(aliases) != 2 {
		t.Errorf("aliases = %v", aliases)
	}
	cost, _ := m["cost"].(map[string]interface{})
	if cost == nil || cost["cache_read"] == nil || cost["cache_write"] == nil {
		t.Errorf("cost missing cache fields: %v", cost)
	}
	if m["status"] != "beta" {
		t.Errorf("status = %v, want beta", m["status"])
	}
}

func TestGenerateConfig_ModelWithDeprecation(t *testing.T) {
	tmp := t.TempDir()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	if err := d.UpsertProvider(&models.Provider{ID: "test", Status: "active"}); err != nil {
		t.Fatalf("upsert provider: %v", err)
	}
	if err := d.UpsertModel(&models.Model{
		ID: "test/old", DisplayName: "old", ProviderID: "test",
		ContextWindow: 4096, Status: "deprecated",
		Deprecation: `{"reason": "replaced", "alternative": "test/new"}`,
	}); err != nil {
		t.Fatalf("upsert model: %v", err)
	}

	svc := NewService(d, tmp)
	if err := svc.GenerateConfig(); err != nil {
		t.Fatalf("generate: %v", err)
	}

	raw, _ := os.ReadFile(filepath.Join(tmp, "opencode.jsonc"))
	var cfg map[string]interface{}
	_ = json.Unmarshal(raw, &cfg)
	m, _ := cfg["provider"].(map[string]interface{})["test"].(map[string]interface{})["models"].(map[string]interface{})["old"].(map[string]interface{})

	if m["status"] != "deprecated" {
		t.Errorf("status = %v, want deprecated", m["status"])
	}
	dep, _ := m["deprecation"].(map[string]interface{})
	if dep == nil || dep["reason"] != "replaced" {
		t.Errorf("deprecation = %v", dep)
	}
}

func TestGenerateConfig_ModelWithInterleaved(t *testing.T) {
	tmp := t.TempDir()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	if err := d.UpsertProvider(&models.Provider{ID: "test", Status: "active"}); err != nil {
		t.Fatalf("upsert provider: %v", err)
	}
	if err := d.UpsertModel(&models.Model{
		ID: "test/interleaved", DisplayName: "interleaved", ProviderID: "test",
		ContextWindow: 100000, Status: "active",
		Interleaved: `{"field": "reasoning_content"}`,
	}); err != nil {
		t.Fatalf("upsert model: %v", err)
	}

	svc := NewService(d, tmp)
	if err := svc.GenerateConfig(); err != nil {
		t.Fatalf("generate: %v", err)
	}

	raw, _ := os.ReadFile(filepath.Join(tmp, "opencode.jsonc"))
	var cfg map[string]interface{}
	_ = json.Unmarshal(raw, &cfg)
	m, _ := cfg["provider"].(map[string]interface{})["test"].(map[string]interface{})["models"].(map[string]interface{})["interleaved"].(map[string]interface{})

	iv, _ := m["interleaved"].(map[string]interface{})
	if iv == nil || iv["field"] != "reasoning_content" {
		t.Errorf("interleaved = %v", iv)
	}
}

func TestGenerateConfig_MCPWithEnvVars(t *testing.T) {
	tmp := t.TempDir()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	if err := d.UpsertMCP(&models.MCPServer{
		ID: "github", Type: "local",
		Command: `["node","server.js"]`,
		Enabled: true,
		EnvVars: `{"GITHUB_TOKEN": "$GITHUB_TOKEN"}`,
	}); err != nil {
		t.Fatalf("upsert mcp: %v", err)
	}

	svc := NewService(d, tmp)
	if err := svc.GenerateConfig(); err != nil {
		t.Fatalf("generate: %v", err)
	}

	raw, _ := os.ReadFile(filepath.Join(tmp, "opencode.jsonc"))
	var cfg map[string]interface{}
	_ = json.Unmarshal(raw, &cfg)
	m, _ := cfg["mcp"].(map[string]interface{})["github"].(map[string]interface{})
	env, _ := m["environment"].(map[string]interface{})
	if env == nil || env["GITHUB_TOKEN"] != "$GITHUB_TOKEN" {
		t.Errorf("mcp environment = %v", env)
	}
}

func TestGenerateAgents(t *testing.T) {
	tmp := t.TempDir()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	if err := d.UpsertAgent(&models.Agent{
		ID: "coder", Description: "Coding agent", CurrentModelID: "groq/llama",
		Mode: "primary", Temperature: 0.2, Color: "green",
		Permission: `{"allow": ["bash"]}`,
		Status: "active", Source: "test",
	}); err != nil {
		t.Fatalf("upsert agent: %v", err)
	}

	svc := NewService(d, tmp)
	if err := svc.GenerateAgents(); err != nil {
		t.Fatalf("GenerateAgents: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(tmp, "agents", "coder.md"))
	if err != nil {
		t.Fatalf("read agent file: %v", err)
	}
	content := string(raw)
	if !strings.Contains(content, "description: Coding agent") {
		t.Errorf("missing description in frontmatter: %s", content)
	}
	if !strings.Contains(content, "model: groq/llama") {
		t.Errorf("missing model in frontmatter: %s", content)
	}
	if !strings.Contains(content, "temperature: 0.2") {
		t.Errorf("missing temperature in frontmatter: %s", content)
	}
	if !strings.Contains(content, "color: green") {
		t.Errorf("missing color in frontmatter: %s", content)
	}
	if !strings.Contains(content, "allow:") || !strings.Contains(content, "bash") {
		t.Errorf("missing permission in frontmatter: %s", content)
	}
}

func TestGenerateCommands(t *testing.T) {
	tmp := t.TempDir()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	if err := d.UpsertCommand(&models.Command{
		ID: "deploy", Template: "Deploy to $1",
		Description: "Deploy command", Agent: "deployer", Model: "groq/llama",
		Status: "active", Source: "test",
	}); err != nil {
		t.Fatalf("upsert command: %v", err)
	}

	svc := NewService(d, tmp)
	if err := svc.GenerateCommands(); err != nil {
		t.Fatalf("GenerateCommands: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(tmp, "commands", "deploy.md"))
	if err != nil {
		t.Fatalf("read command file: %v", err)
	}
	content := string(raw)
	if !strings.Contains(content, "description: Deploy command") {
		t.Errorf("missing description in frontmatter: %s", content)
	}
	if !strings.Contains(content, "agent: deployer") {
		t.Errorf("missing agent in frontmatter: %s", content)
	}
	if !strings.Contains(content, "Deploy to") {
		t.Errorf("missing template in body: %s", content)
	}
}

func TestParseStringList(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  map[string]bool
	}{
		{"empty", "", map[string]bool{}},
		{"null", "null", map[string]bool{}},
		{"single", `["openai"]`, map[string]bool{"openai": true}},
		{"multiple", `["a","b"]`, map[string]bool{"a": true, "b": true}},
		{"invalid", "not-json", map[string]bool{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseStringList(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("parseStringList(%q) = %v, want %v", tt.input, got, tt.want)
			}
			for k := range tt.want {
				if !got[k] {
					t.Errorf("parseStringList(%q) missing key %q", tt.input, k)
				}
			}
		})
	}
}

func TestMerge_RemovesDeletedMCP(t *testing.T) {
	gen := map[string]interface{}{
		"mcp": map[string]interface{}{
			"filesystem": map[string]interface{}{"type": "local"},
		},
	}
	existing := map[string]interface{}{
		"mcp": map[string]interface{}{
			"filesystem":   map[string]interface{}{"type": "local"},
			"deleted-mcp":  map[string]interface{}{"type": "remote"},
			"user-added":   map[string]interface{}{"type": "local"},
		},
	}
	// Simulate that "deleted-mcp" was in the DB pre-sync, so it should be removed
	preSyncIDs := map[string]bool{"filesystem": true, "deleted-mcp": true}

	s := &Service{}
	merged := s.mergeWithExisting(gen, existing, preSyncIDs)
	mcp := merged["mcp"].(map[string]interface{})

	if _, ok := mcp["deleted-mcp"]; ok {
		t.Error("deleted-mcp should be removed (was in pre-sync, not in generated)")
	}
	if _, ok := mcp["user-added"]; !ok {
		t.Error("user-added mcp should be kept (not in pre-sync)")
	}
	if _, ok := mcp["filesystem"]; !ok {
		t.Error("filesystem mcp should be present")
	}
}

func TestMerge_NonManagedKeysPreserved(t *testing.T) {
	existing := map[string]interface{}{
		"custom_key": map[string]interface{}{"nested": true},
		"theme":      "dark",
		"tools":      []string{"a"},
	}
	gen := map[string]interface{}{
		"$schema": "https://opencode.ai/config.json",
	}
	s := &Service{}
	merged := s.mergeWithExisting(gen, existing, nil)
	if merged["custom_key"] == nil {
		t.Error("custom_key should be preserved")
	}
	if merged["theme"] != "dark" {
		t.Errorf("theme = %v, want dark", merged["theme"])
	}
}

func TestGenerateConfig_MCPEnvVarsWithRemote(t *testing.T) {
	tmp := t.TempDir()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	if err := d.UpsertMCP(&models.MCPServer{
		ID: "mixed", Type: "remote",
		URL: "https://mcp.example.com", Enabled: true,
		EnvVars: `{"API_KEY": "sekret"}`,
	}); err != nil {
		t.Fatalf("upsert mcp: %v", err)
	}

	svc := NewService(d, tmp)
	if err := svc.GenerateConfig(); err != nil {
		t.Fatalf("generate: %v", err)
	}

	raw, _ := os.ReadFile(filepath.Join(tmp, "opencode.jsonc"))
	var cfg map[string]interface{}
	_ = json.Unmarshal(raw, &cfg)

	mcps, _ := cfg["mcp"].(map[string]interface{})
	m, _ := mcps["mixed"].(map[string]interface{})
	if m["type"] != "remote" {
		t.Errorf("type = %v", m["type"])
	}
	env, _ := m["environment"].(map[string]interface{})
	if env == nil || env["API_KEY"] != "sekret" {
		t.Errorf("environment = %v", env)
	}
}

func TestGenerateConfig_CommandWithoutOptionalFields(t *testing.T) {
	tmp := t.TempDir()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	if err := d.UpsertCommand(&models.Command{
		ID: "test", Template: "just a template",
		Status: "active", Source: "test",
	}); err != nil {
		t.Fatalf("upsert command: %v", err)
	}

	svc := NewService(d, tmp)
	if err := svc.GenerateConfig(); err != nil {
		t.Fatalf("generate: %v", err)
	}

	raw, _ := os.ReadFile(filepath.Join(tmp, "opencode.jsonc"))
	var cfg map[string]interface{}
	_ = json.Unmarshal(raw, &cfg)

	cmds, _ := cfg["command"].(map[string]interface{})
	c, _ := cmds["test"].(map[string]interface{})
	if c == nil {
		t.Fatal("command must be present")
	}
	if c["template"] != "just a template" {
		t.Errorf("template = %v", c["template"])
	}
	if _, ok := c["agent"]; ok {
		t.Error("agent should not be present for optional field")
	}
}

func TestMerge_PreservesUserMCPAndAddsDBMCP(t *testing.T) {
	gen := map[string]interface{}{
		"mcp": map[string]interface{}{
			"from_db": map[string]interface{}{"type": "local"},
		},
	}
	existing := map[string]interface{}{
		"mcp": map[string]interface{}{
			"user_manual": map[string]interface{}{"type": "local"},
			"from_db":     map[string]interface{}{"type": "OLD_SHOULD_LOSE"},
		},
	}
	s := &Service{}
	merged := s.mergeWithExisting(gen, existing, nil)
	mcp := merged["mcp"].(map[string]interface{})
	if _, ok := mcp["user_manual"]; !ok {
		t.Errorf("user_manual mcp was lost")
	}
	if _, ok := mcp["from_db"]; !ok {
		t.Errorf("from_db mcp was lost")
	}
	if mcp["from_db"].(map[string]interface{})["type"] != "local" {
		t.Errorf("from_db mcp should be regenerated, got %v", mcp["from_db"])
	}
}
