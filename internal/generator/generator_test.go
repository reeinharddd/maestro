package generator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/reeinharrrd/opencode-kit/internal/db"
	"github.com/reeinharrrd/opencode-kit/pkg/models"
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

	if _, ok := cfg["okit"]; ok {
		t.Error("opencode.jsonc must not contain an 'okit' top-level key — opencode schema rejects it")
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

	stateRaw, err := os.ReadFile(filepath.Join(tmp, "okit-state.json"))
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
	merged := s.mergeWithExisting(gen, existing)
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
	merged := s.mergeWithExisting(gen, existing)
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
