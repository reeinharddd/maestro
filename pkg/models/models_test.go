package models

import (
	"testing"
)

func TestProviderCreation(t *testing.T) {
	p := Provider{
		ID:      "openai",
		Name:    "OpenAI",
		BaseURL: "https://api.openai.com",
		Enabled: true,
		Source:  "builtin",
		Status:  "active",
	}

	if p.ID != "openai" {
		t.Errorf("ID = %q, want %q", p.ID, "openai")
	}
	if p.Name != "OpenAI" {
		t.Errorf("Name = %q, want %q", p.Name, "OpenAI")
	}
	if p.BaseURL != "https://api.openai.com" {
		t.Errorf("BaseURL = %q, want %q", p.BaseURL, "https://api.openai.com")
	}
	if !p.Enabled {
		t.Error("Enabled should be true")
	}
	if p.Source != "builtin" {
		t.Errorf("Source = %q, want %q", p.Source, "builtin")
	}
}

func TestProviderWithOptionalFields(t *testing.T) {
	p := Provider{
		ID:              "anthropic",
		Name:            "Anthropic",
		CatalogURL:      "https://docs.anthropic.com/models",
		KeyEnv:          "ANTHROPIC_API_KEY",
		TimeoutMs:       30000,
		EnterpriseURL:   "https://enterprise.anthropic.com",
		APIPackage:      "anthropic",
		EnvList:         `["ANTHROPIC_API_KEY"]`,
		LastSynced:      1700000000,
		Priority:        10,
		HeaderTimeoutMs: 5000,
		ChunkTimeoutMs:  1000,
		Source:          "config",
		Status:          "active",
	}

	if p.CatalogURL != "https://docs.anthropic.com/models" {
		t.Errorf("CatalogURL = %q", p.CatalogURL)
	}
	if p.KeyEnv != "ANTHROPIC_API_KEY" {
		t.Errorf("KeyEnv = %q", p.KeyEnv)
	}
	if p.TimeoutMs != 30000 {
		t.Errorf("TimeoutMs = %d", p.TimeoutMs)
	}
	if p.Priority != 10 {
		t.Errorf("Priority = %d", p.Priority)
	}
}

func TestModelCreation(t *testing.T) {
	m := Model{
		ID:            "gpt-4o",
		ProviderID:    "openai",
		DisplayName:   "GPT-4o",
		ContextWindow: 128000,
		MaxOutput:     4096,
		Tier:          "premium",
		Status:        "active",
		Source:        "catalog",
	}

	if m.ID != "gpt-4o" {
		t.Errorf("ID = %q, want %q", m.ID, "gpt-4o")
	}
	if m.ProviderID != "openai" {
		t.Errorf("ProviderID = %q", m.ProviderID)
	}
	if m.ContextWindow != 128000 {
		t.Errorf("ContextWindow = %d", m.ContextWindow)
	}
	if m.Tier != "premium" {
		t.Errorf("Tier = %q", m.Tier)
	}
}

func TestModelCapabilities(t *testing.T) {
	m := Model{
		ID:               "claude-3-opus",
		ProviderID:       "anthropic",
		ContextWindow:    200000,
		FunctionCalling:  true,
		Vision:           true,
		Streaming:        true,
		StructuredOutput: true,
		Tier:             "premium",
		Status:           "active",
		Source:           "catalog",
	}

	if !m.FunctionCalling {
		t.Error("FunctionCalling should be true")
	}
	if !m.Vision {
		t.Error("Vision should be true")
	}
	if !m.Streaming {
		t.Error("Streaming should be true")
	}
	if !m.StructuredOutput {
		t.Error("StructuredOutput should be true")
	}
}

func TestModelPricing(t *testing.T) {
	m := Model{
		ID:                "gpt-4o-mini",
		ProviderID:        "openai",
		PricingPrompt:     0.00015,
		PricingCompletion: 0.0006,
		PricingCacheRead:  0.000075,
		PricingCacheWrite: 0.0003,
		ContextWindow:     128000,
		Tier:              "budget",
		Status:            "active",
		Source:            "catalog",
	}

	if m.PricingPrompt != 0.00015 {
		t.Errorf("PricingPrompt = %f", m.PricingPrompt)
	}
	if m.PricingCompletion != 0.0006 {
		t.Errorf("PricingCompletion = %f", m.PricingCompletion)
	}
	if m.PricingCacheRead != 0.000075 {
		t.Errorf("PricingCacheRead = %f", m.PricingCacheRead)
	}
	if m.PricingCacheWrite != 0.0003 {
		t.Errorf("PricingCacheWrite = %f", m.PricingCacheWrite)
	}
}

func TestModelLatency(t *testing.T) {
	m := Model{
		ID:            "claude-sonnet",
		ProviderID:    "anthropic",
		LatencyP50Ms:  1200.5,
		LatencyP95Ms:  3500.7,
		TokensPerSec:  85.3,
		ContextWindow: 200000,
		DefaultTemp:   1.0,
		Tier:          "premium",
		Status:        "active",
		Source:        "catalog",
	}

	if m.LatencyP50Ms != 1200.5 {
		t.Errorf("LatencyP50Ms = %f", m.LatencyP50Ms)
	}
	if m.TokensPerSec != 85.3 {
		t.Errorf("TokensPerSec = %f", m.TokensPerSec)
	}
	if m.DefaultTemp != 1.0 {
		t.Errorf("DefaultTemp = %f", m.DefaultTemp)
	}
}

func TestModelExtraFields(t *testing.T) {
	m := Model{
		ID:               "gpt-4o",
		ProviderID:       "openai",
		Reasoning:        false,
		Audio:            true,
		OCR:              false,
		FineTuning:       true,
		Classification:   false,
		Moderation:       true,
		Tags:             `["gpt4","chat"]`,
		Aliases:          `["gpt4"]`,
		Family:           "gpt-4",
		ReleaseDate:      "2024-05-13",
		Deprecation:      "2026-01-01",
		Experimental:     false,
		ModalitiesInput:  "text,image",
		ModalitiesOutput: "text,audio",
		OwnedBy:          "openai",
		Architecture:     "transformer",
		RecommendedUse:   "general conversation, coding",
		ContextWindow:    128000,
		Tier:             "premium",
		Status:           "active",
		Source:           "catalog",
	}

	if m.Reasoning != false {
		t.Error("Reasoning should be false")
	}
	if !m.Audio {
		t.Error("Audio should be true")
	}
	if !m.FineTuning {
		t.Error("FineTuning should be true")
	}
	if !m.Moderation {
		t.Error("Moderation should be true")
	}
	if m.OwnedBy != "openai" {
		t.Errorf("OwnedBy = %q", m.OwnedBy)
	}
}

func TestAgentCreation(t *testing.T) {
	a := Agent{
		ID:          "code-assistant",
		TaskType:    "coding",
		Description: "General coding assistant",
		Mode:        "primary",
		Status:      "active",
		Source:      "builtin",
	}

	if a.ID != "code-assistant" {
		t.Errorf("ID = %q", a.ID)
	}
	if a.Mode != "primary" {
		t.Errorf("Mode = %q", a.Mode)
	}
	if a.Status != "active" {
		t.Errorf("Status = %q", a.Status)
	}
}

func TestAgentOptionalFields(t *testing.T) {
	a := Agent{
		ID:             "expert-reviewer",
		CurrentModelID: "gpt-4o",
		FallbackIDs:    `["claude-3-opus","gpt-4o-mini"]`,
		PromptFile:     "prompts/review.md",
		Temperature:    0.3,
		MaxSteps:       25,
		Permission:     `{"allow_fs_read":true}`,
		Color:          "yellow",
		Hidden:         true,
		Mode:           "subagent",
		Status:         "active",
		Source:         "config",
	}

	if a.CurrentModelID != "gpt-4o" {
		t.Errorf("CurrentModelID = %q", a.CurrentModelID)
	}
	if a.Temperature != 0.3 {
		t.Errorf("Temperature = %f", a.Temperature)
	}
	if a.MaxSteps != 25 {
		t.Errorf("MaxSteps = %d", a.MaxSteps)
	}
	if !a.Hidden {
		t.Error("Hidden should be true")
	}
}

func TestSkillCreation(t *testing.T) {
	s := Skill{
		ID:          "tdd",
		Source:      "builtin",
		SourcePath:  "skills/tdd.md",
		TargetPath:  ".opencode/skills/tdd.md",
		Type:        "workflow",
		Status:      "installed",
		Hash:        "abc123",
		Description: "Test-driven development workflow",
		Category:    "process",
		Tags:        "testing,tdd,workflow",
		Triggers:    "test,tdd",
		SizeBytes:   2048,
		Filename:    "tdd.md",
	}

	if s.ID != "tdd" {
		t.Errorf("ID = %q", s.ID)
	}
	if s.Type != "workflow" {
		t.Errorf("Type = %q", s.Type)
	}
	if s.Status != "installed" {
		t.Errorf("Status = %q", s.Status)
	}
	if s.SizeBytes != 2048 {
		t.Errorf("SizeBytes = %d", s.SizeBytes)
	}
	if s.Category != "process" {
		t.Errorf("Category = %q", s.Category)
	}
}

func TestMCPServerCreation(t *testing.T) {
	m := MCPServer{
		ID:      "filesystem",
		Type:    "local",
		Command: `["npx","-y","@modelcontextprotocol/server-filesystem"]`,
		Enabled: true,
		Timeout: 30000,
	}

	if m.ID != "filesystem" {
		t.Errorf("ID = %q", m.ID)
	}
	if m.Type != "local" {
		t.Errorf("Type = %q", m.Type)
	}
	if !m.Enabled {
		t.Error("Enabled should be true")
	}
}

func TestMCPServerRemote(t *testing.T) {
	m := MCPServer{
		ID:      "web-search",
		Type:    "remote",
		URL:     "https://mcp.example.com/search",
		EnvVars: `{"API_KEY":"env:FIRE_API_KEY"}`,
		Enabled: true,
		Timeout: 60000,
		Source:  "registry",
	}

	if m.Type != "remote" {
		t.Errorf("Type = %q", m.Type)
	}
	if m.URL != "https://mcp.example.com/search" {
		t.Errorf("URL = %q", m.URL)
	}
	if m.EnvVars != `{"API_KEY":"env:FIRE_API_KEY"}` {
		t.Errorf("EnvVars = %q", m.EnvVars)
	}
}

func TestLSPServerCreation(t *testing.T) {
	l := LSPServer{
		ID:         "typescript",
		Command:    `["typescript-language-server","--stdio"]`,
		Extensions: `["ts","tsx","js","jsx"]`,
		Disabled:   false,
	}

	if l.ID != "typescript" {
		t.Errorf("ID = %q", l.ID)
	}
	if l.Extensions != `["ts","tsx","js","jsx"]` {
		t.Errorf("Extensions = %q", l.Extensions)
	}
	if l.Disabled {
		t.Error("Disabled should be false")
	}
}

func TestLSPServerWithEnv(t *testing.T) {
	l := LSPServer{
		ID:             "rust-analyzer",
		Command:        `["rust-analyzer"]`,
		Env:            `{"RUST_LOG":"info"}`,
		Initialization: `{"checkOnSave":true}`,
	}

	if l.Env != `{"RUST_LOG":"info"}` {
		t.Errorf("Env = %q", l.Env)
	}
	if l.Initialization != `{"checkOnSave":true}` {
		t.Errorf("Initialization = %q", l.Initialization)
	}
}

func TestCommandCreation(t *testing.T) {
	c := Command{
		ID:          "test",
		Template:    "go test -v ./...",
		Description: "Run all tests",
		Agent:       "default",
		Model:       "gpt-4o",
		Subtask:     false,
		Status:      "active",
	}

	if c.ID != "test" {
		t.Errorf("ID = %q", c.ID)
	}
	if c.Template != "go test -v ./..." {
		t.Errorf("Template = %q", c.Template)
	}
	if c.Description != "Run all tests" {
		t.Errorf("Description = %q", c.Description)
	}
}

func TestCommandSubtask(t *testing.T) {
	c := Command{
		ID:       "review-code",
		Template: "review the code at {{path}}",
		Subtask:  true,
		Source:   "builtin",
		Status:   "active",
	}

	if !c.Subtask {
		t.Error("Subtask should be true")
	}
	if c.Source != "builtin" {
		t.Errorf("Source = %q", c.Source)
	}
}

func TestContextEstimate(t *testing.T) {
	ce := ContextEstimate{
		TotalBytes:  15000,
		TotalSkills: 5,
		BySource:    map[string]int64{"builtin": 10000, "registry": 5000},
		ByCategory:  map[string]int64{"process": 8000, "utility": 7000},
		Heaviest: []SkillSizeEntry{
			{ID: "tdd", Source: "builtin", SizeBytes: 5000},
			{ID: "debug", Source: "builtin", SizeBytes: 3000},
		},
	}

	if ce.TotalBytes != 15000 {
		t.Errorf("TotalBytes = %d", ce.TotalBytes)
	}
	if ce.TotalSkills != 5 {
		t.Errorf("TotalSkills = %d", ce.TotalSkills)
	}
	if len(ce.Heaviest) != 2 {
		t.Errorf("len(Heaviest) = %d, want 2", len(ce.Heaviest))
	}
	// Verify BySource map
	if v, ok := ce.BySource["builtin"]; !ok || v != 10000 {
		t.Errorf(`BySource["builtin"] = %d, want 10000`, v)
	}
}

func TestSkillSizeEntry(t *testing.T) {
	e := SkillSizeEntry{
		ID:        "tdd",
		Source:    "builtin",
		Category:  "process",
		SizeBytes: 5000,
	}

	if e.ID != "tdd" {
		t.Errorf("ID = %q", e.ID)
	}
	if e.SizeBytes != 5000 {
		t.Errorf("SizeBytes = %d", e.SizeBytes)
	}
}

func TestRoutingRule(t *testing.T) {
	r := RoutingRule{
		TaskKey:        "code-review",
		Description:    "Route code reviews to best model",
		NeedsFC:        true,
		NeedsVision:    false,
		CurrentModelID: "gpt-4o",
		FallbackIDs:    `["claude-3-opus"]`,
		Enabled:        true,
		PriorityWeight: 10,
	}

	if r.TaskKey != "code-review" {
		t.Errorf("TaskKey = %q", r.TaskKey)
	}
	if !r.NeedsFC {
		t.Error("NeedsFC should be true")
	}
	if !r.Enabled {
		t.Error("Enabled should be true")
	}
	if r.PriorityWeight != 10 {
		t.Errorf("PriorityWeight = %d", r.PriorityWeight)
	}
}

func TestRoutingEvent(t *testing.T) {
	e := RoutingEvent{
		ID:            1,
		TaskKey:       "code-review",
		SelectedModel: "gpt-4o",
		Candidates:    `["gpt-4o","claude-3-opus"]`,
		Reason:        "best quality for task",
		Shadow:        false,
		CreatedAt:     "2025-01-01T00:00:00Z",
	}

	if e.ID != 1 {
		t.Errorf("ID = %d", e.ID)
	}
	if e.TaskKey != "code-review" {
		t.Errorf("TaskKey = %q", e.TaskKey)
	}
	if e.Reason != "best quality for task" {
		t.Errorf("Reason = %q", e.Reason)
	}
}

func TestModelProfile(t *testing.T) {
	p := ModelProfile{
		ModelID:        "gpt-4o",
		RealContext:    128000,
		MaxOutput:      4096,
		SupportsStream: true,
		SupportsSO:     true,
		StreamTPS:      120.5,
		ProfiledAt:     1700000000,
	}

	if p.ModelID != "gpt-4o" {
		t.Errorf("ModelID = %q", p.ModelID)
	}
	if !p.SupportsStream {
		t.Error("SupportsStream should be true")
	}
	if !p.SupportsSO {
		t.Error("SupportsSO should be true")
	}
	if p.StreamTPS != 120.5 {
		t.Errorf("StreamTPS = %f", p.StreamTPS)
	}
}

func TestBudgetConfig(t *testing.T) {
	b := BudgetConfig{
		ID:             "team-budget",
		DailyGlobalUSD: 50.0,
		PreferredTier:  "budget",
		UpdatedAt:      "2025-01-01T00:00:00Z",
	}

	if b.ID != "team-budget" {
		t.Errorf("ID = %q", b.ID)
	}
	if b.DailyGlobalUSD != 50.0 {
		t.Errorf("DailyGlobalUSD = %f", b.DailyGlobalUSD)
	}
	if b.PreferredTier != "budget" {
		t.Errorf("PreferredTier = %q", b.PreferredTier)
	}
}

func TestSyncLog(t *testing.T) {
	l := SyncLog{
		ID:         1,
		Phase:      "download",
		Status:     "completed",
		Details:    "Downloaded 42 skills",
		DurationMs: 1500,
		CreatedAt:  "2025-01-01T00:00:00Z",
	}

	if l.Phase != "download" {
		t.Errorf("Phase = %q", l.Phase)
	}
	if l.Status != "completed" {
		t.Errorf("Status = %q", l.Status)
	}
	if l.DurationMs != 1500 {
		t.Errorf("DurationMs = %d", l.DurationMs)
	}
}

func TestExecLog(t *testing.T) {
	e := ExecLog{
		ID:         1,
		Agent:      "code-assistant",
		Model:      "gpt-4o",
		Task:       "review PR #42",
		TokensIn:   1500,
		TokensOut:  500,
		DurationMs: 12000,
		Success:    true,
		CreatedAt:  "2025-01-01T00:00:00Z",
	}

	if e.Agent != "code-assistant" {
		t.Errorf("Agent = %q", e.Agent)
	}
	if e.TokensIn != 1500 {
		t.Errorf("TokensIn = %d", e.TokensIn)
	}
	if e.TokensOut != 500 {
		t.Errorf("TokensOut = %d", e.TokensOut)
	}
	if !e.Success {
		t.Error("Success should be true")
	}
}

func TestSnapshot(t *testing.T) {
	s := Snapshot{
		ID:        1,
		Hash:      "def456",
		Content:   "state snapshot content",
		CreatedAt: "2025-01-01T00:00:00Z",
	}

	if s.Hash != "def456" {
		t.Errorf("Hash = %q", s.Hash)
	}
	if s.Content != "state snapshot content" {
		t.Errorf("Content = %q", s.Content)
	}
}

func TestSource(t *testing.T) {
	s := Source{
		ID:         "github-maestro",
		RemoteURL:  "https://github.com/user/maestro",
		LocalPath:  "/home/user/maestro",
		Commit:     "abc123",
		Status:     "synced",
		LastSynced: 1700000000,
	}

	if s.ID != "github-maestro" {
		t.Errorf("ID = %q", s.ID)
	}
	if s.RemoteURL != "https://github.com/user/maestro" {
		t.Errorf("RemoteURL = %q", s.RemoteURL)
	}
	if s.Status != "synced" {
		t.Errorf("Status = %q", s.Status)
	}
}

func TestSourceItem(t *testing.T) {
	si := SourceItem{
		ID:         "item-1",
		SourceID:   "github-maestro",
		Type:       "skill",
		SourcePath: "skills/tdd.md",
		TargetPath: ".opencode/skills/tdd.md",
		Hash:       "abc123",
		Status:     "linked",
	}

	if si.SourceID != "github-maestro" {
		t.Errorf("SourceID = %q", si.SourceID)
	}
	if si.Type != "skill" {
		t.Errorf("Type = %q", si.Type)
	}
	if si.Status != "linked" {
		t.Errorf("Status = %q", si.Status)
	}
}

func TestConfigFragment(t *testing.T) {
	cf := ConfigFragment{
		ID:         "cf-1",
		ConfigType: "skill",
		Content:    "name: test\nversion: 1",
		Source:     "registry",
		Hash:       "abc123",
		CreatedAt:  "2025-01-01T00:00:00Z",
		UpdatedAt:  "2025-01-02T00:00:00Z",
	}

	if cf.ConfigType != "skill" {
		t.Errorf("ConfigType = %q", cf.ConfigType)
	}
	if cf.Content != "name: test\nversion: 1" {
		t.Errorf("Content = %q", cf.Content)
	}
}

func TestTool(t *testing.T) {
	tool := Tool{
		ID:          "search",
		Description: "Search the web",
		Parameters: map[string]ToolParameter{
			"query": {
				Type:        "string",
				Description: "Search query",
			},
		},
	}

	if tool.ID != "search" {
		t.Errorf("ID = %q", tool.ID)
	}
	if tool.Description != "Search the web" {
		t.Errorf("Description = %q", tool.Description)
	}
	if len(tool.Parameters) != 1 {
		t.Errorf("len(Parameters) = %d, want 1", len(tool.Parameters))
	}
}

func TestToolParameter(t *testing.T) {
	tp := ToolParameter{
		Type:        "object",
		Description: "User profile",
		Properties: map[string]ToolParameter{
			"name": {Type: "string", Description: "Full name"},
			"age":  {Type: "integer", Description: "Age in years"},
		},
		Required: []string{"name"},
	}

	if tp.Type != "object" {
		t.Errorf("Type = %q", tp.Type)
	}
	if len(tp.Properties) != 2 {
		t.Errorf("len(Properties) = %d, want 2", len(tp.Properties))
	}
	if len(tp.Required) != 1 || tp.Required[0] != "name" {
		t.Errorf("Required = %v, want [name]", tp.Required)
	}
}

func TestProject(t *testing.T) {
	p := Project{
		ID:         "proj-1",
		Path:       "/home/user/maestro",
		Name:       "maestro",
		DetectedAt: 1700000000,
		UpdatedAt:  1700000001,
		Status:     "active",
		Source:     "scan",
	}

	if p.Name != "maestro" {
		t.Errorf("Name = %q", p.Name)
	}
	if p.Status != "active" {
		t.Errorf("Status = %q", p.Status)
	}
	if p.Source != "scan" {
		t.Errorf("Source = %q", p.Source)
	}
}

func TestDetectedStack(t *testing.T) {
	ds := DetectedStack{
		ID:         "stack-1",
		ProjectID:  "proj-1",
		Language:   "Go",
		Framework:  "Chi",
		Version:    "1.22",
		Builder:    "go build",
		TestRunner: "go test",
		Linter:     "golangci-lint",
		DetectedAt: 1700000000,
		Confidence: 0.95,
	}

	if ds.Language != "Go" {
		t.Errorf("Language = %q", ds.Language)
	}
	if ds.Framework != "Chi" {
		t.Errorf("Framework = %q", ds.Framework)
	}
	if ds.Confidence != 0.95 {
		t.Errorf("Confidence = %f", ds.Confidence)
	}
}

func TestProjectConfig(t *testing.T) {
	pc := ProjectConfig{
		ID:          "pc-1",
		ProjectID:   "proj-1",
		ConfigType:  "agents",
		Content:     `{"agents":[]}`,
		GeneratedAt: 1700000000,
		Hash:        "abc123",
	}

	if pc.ConfigType != "agents" {
		t.Errorf("ConfigType = %q", pc.ConfigType)
	}
	if pc.Content != `{"agents":[]}` {
		t.Errorf("Content = %q", pc.Content)
	}
}

func TestScannerResult(t *testing.T) {
	sr := ScannerResult{
		ProjectID: "proj-1",
		Stacks: []DetectedStack{
			{ID: "stack-1", Language: "Go", Confidence: 0.95},
		},
		Configs: map[string]string{
			"builder": "go build",
		},
		Errors: []string{},
	}

	if sr.ProjectID != "proj-1" {
		t.Errorf("ProjectID = %q", sr.ProjectID)
	}
	if len(sr.Stacks) != 1 {
		t.Errorf("len(Stacks) = %d, want 1", len(sr.Stacks))
	}
	if sr.Stacks[0].Language != "Go" {
		t.Errorf("Stack Language = %q", sr.Stacks[0].Language)
	}
}
