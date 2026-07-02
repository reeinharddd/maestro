package sync

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/reeinharrrd/maestro/internal/db"
	"github.com/reeinharrrd/maestro/pkg/models"
)

func TestNew(t *testing.T) {
	t.Parallel()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	svc := New(d)
	if svc == nil {
		t.Fatal("New returned nil")
	}
}

func TestImportFromOpenCodeConfig(t *testing.T) {
	t.Parallel()

	t.Run("full config with all sections", func(t *testing.T) {
		t.Parallel()
		d, err := db.Open(":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer d.Close()

		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "opencode.jsonc")
		config := map[string]interface{}{
			"$schema": "https://opencode.ai/config.json",
			"provider": map[string]interface{}{
				"groq": map[string]interface{}{
					"whitelist": []string{"llama-3.3-70b-versatile"},
				},
			},
			"agent": map[string]interface{}{
				"build": map[string]interface{}{
					"model": "groq/llama-3.3-70b-versatile",
				},
			},
			"command": map[string]interface{}{
				"test": map[string]interface{}{
					"template": "test",
				},
			},
			"mcp": map[string]interface{}{
				"engram": map[string]interface{}{
					"type":    "local",
					"command": []string{"engram", "mcp", "--tools=agent"},
					"enabled": true,
				},
			},
			"lsp":             true,
			"autoupdate":      false,
			"disabled_providers": []string{"opencode"},
			"model":           "groq/llama-3.3-70b-versatile",
			"small_model":     "groq/llama-3.1-8b-instant",
			"share":           "manual",
			"compaction": map[string]interface{}{
				"auto": true,
			},
		}
		out, _ := json.MarshalIndent(config, "", "  ")
		if err := os.WriteFile(configPath, out, 0644); err != nil {
			t.Fatal(err)
		}

		svc := New(d)
		diff, err := svc.ImportFromOpenCodeConfig(configPath)
		if err != nil {
			t.Fatalf("import: %v", err)
		}

		if len(diff.AddedProviders) != 0 {
			t.Errorf("providers: want [], got %v", diff.AddedProviders)
		}
		if len(diff.AddedAgents) != 1 || diff.AddedAgents[0] != "build" {
			t.Errorf("agents: want [build], got %v", diff.AddedAgents)
		}
		if len(diff.AddedCommands) != 1 || diff.AddedCommands[0] != "test" {
			t.Errorf("commands: want [test], got %v", diff.AddedCommands)
		}
		if len(diff.AddedMCPs) != 1 || diff.AddedMCPs[0] != "engram" {
			t.Errorf("mcps: want [engram], got %v", diff.AddedMCPs)
		}

		mcpList, _ := d.ListMCPs()
		if len(mcpList) != 1 || mcpList[0].ID != "engram" {
			t.Errorf("mcp not imported: %v", mcpList)
		}

		prefs, _ := d.ListPreferences()
		if prefs[metaPref+"lsp"] != "true" {
			t.Errorf("lsp preference not set: %v", prefs[metaPref+"lsp"])
		}
		for _, key := range []string{"autoupdate", "disabled_providers", "model", "small_model", "share"} {
			if _, ok := prefs[metaPref+key]; !ok {
				t.Errorf("meta key %q not imported", key)
			}
		}

		modelList, _ := d.ListModelsByProvider("groq")
		if len(modelList) != 1 || modelList[0].DisplayName != "llama-3.3-70b-versatile" {
			t.Errorf("model not imported: %v", modelList)
		}

		agents, _ := d.ListAgents()
		if len(agents) != 1 || agents[0].ID != "build" {
			t.Errorf("agent not imported: %v", agents)
		}

		commands, _ := d.ListCommands()
		if len(commands) != 1 || commands[0].ID != "test" {
			t.Errorf("command not imported: %v", commands)
		}

		mcpServers, _ := d.ListMCPs()
		if len(mcpServers) != 1 || mcpServers[0].ID != "engram" {
			t.Errorf("MCP server not imported: %v", mcpServers)
		}

		lspServers, _ := d.ListLSPServers()
		if len(lspServers) != 0 {
			t.Errorf("LSP servers should not be imported when lsp is boolean, got %v", lspServers)
		}

		providers, _ := d.ListProviders()
		found := false
		for _, p := range providers {
			if p.ID == "groq" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("provider groq not found among: %v", providers)
		}

		profiles, _ := d.ListModelProfiles()
		if len(profiles) != 1 || profiles[0].ModelID != "groq/llama-3.3-70b-versatile" {
			t.Errorf("model profile not imported: %v", profiles)
		}
	})

	t.Run("empty config", func(t *testing.T) {
		t.Parallel()
		d, err := db.Open(":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer d.Close()

		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "opencode.jsonc")
		config := map[string]interface{}{
			"$schema": "https://opencode.ai/config.json",
		}
		out, _ := json.MarshalIndent(config, "", "  ")
		os.WriteFile(configPath, out, 0644)

		svc := New(d)
		diff, err := svc.ImportFromOpenCodeConfig(configPath)
		if err != nil {
			t.Fatalf("import: %v", err)
		}

		if len(diff.AddedProviders) != 0 {
			t.Errorf("expected 0 providers, got %v", diff.AddedProviders)
		}
		if len(diff.AddedAgents) != 0 {
			t.Errorf("expected 0 agents, got %v", diff.AddedAgents)
		}
		if len(diff.AddedCommands) != 0 {
			t.Errorf("expected 0 commands, got %v", diff.AddedCommands)
		}
		if len(diff.AddedMCPs) != 0 {
			t.Errorf("expected 0 MCPs, got %v", diff.AddedMCPs)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		t.Parallel()
		d, err := db.Open(":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer d.Close()

		svc := New(d)
		_, err = svc.ImportFromOpenCodeConfig("/nonexistent/path.jsonc")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("relative path", func(t *testing.T) {
		t.Parallel()
		d, err := db.Open(":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer d.Close()

		tempDir := t.TempDir()
		origDir, _ := os.Getwd()
		os.Chdir(tempDir)
		defer os.Chdir(origDir)

		config := map[string]interface{}{
			"$schema": "https://opencode.ai/config.json",
			"provider": map[string]interface{}{
				"openai": map[string]interface{}{
					"name": "OpenAI",
				},
			},
		}
		out, _ := json.MarshalIndent(config, "", "  ")
		os.WriteFile("relative.jsonc", out, 0644)

		svc := New(d)
		diff, err := svc.ImportFromOpenCodeConfig("relative.jsonc")
		if err != nil {
			t.Fatalf("import relative path: %v", err)
		}
		if len(diff.AddedProviders) != 1 || diff.AddedProviders[0] != "openai" {
			t.Errorf("expected openai provider, got %v", diff.AddedProviders)
		}
	})

	t.Run("provider with baseURL option", func(t *testing.T) {
		t.Parallel()
		d, err := db.Open(":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer d.Close()

		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "opencode.jsonc")
		config := map[string]interface{}{
			"$schema": "https://opencode.ai/config.json",
			"provider": map[string]interface{}{
				"custom": map[string]interface{}{
					"name": "Custom API",
					"options": map[string]interface{}{
						"baseURL": "https://custom.api/v1",
					},
				},
			},
		}
		out, _ := json.MarshalIndent(config, "", "  ")
		os.WriteFile(configPath, out, 0644)

		svc := New(d)
		diff, err := svc.ImportFromOpenCodeConfig(configPath)
		if err != nil {
			t.Fatalf("import: %v", err)
		}
		if len(diff.AddedProviders) != 1 || diff.AddedProviders[0] != "custom" {
			t.Errorf("expected custom provider, got %v", diff.AddedProviders)
		}

		p, err := d.GetProvider("custom")
		if err != nil {
			t.Fatal(err)
		}
		if p.BaseURL != "https://custom.api/v1" {
			t.Errorf("expected baseURL, got %q", p.BaseURL)
		}
		if p.Name != "Custom API" {
			t.Errorf("expected name 'Custom API', got %q", p.Name)
		}

		prefs, _ := d.ListPreferences()
		optsKey := metaPref + "provider_options_custom"
		if _, ok := prefs[optsKey]; !ok {
			t.Errorf("expected provider_options_custom preference")
		}
	})

	t.Run("provider with NPM", func(t *testing.T) {
		t.Parallel()
		d, err := db.Open(":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer d.Close()

		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "opencode.jsonc")
		config := map[string]interface{}{
			"$schema": "https://opencode.ai/config.json",
			"provider": map[string]interface{}{
				"npm-provider": map[string]interface{}{
					"npm": "@org/package",
				},
			},
		}
		out, _ := json.MarshalIndent(config, "", "  ")
		os.WriteFile(configPath, out, 0644)

		svc := New(d)
		_, err = svc.ImportFromOpenCodeConfig(configPath)
		if err != nil {
			t.Fatalf("import: %v", err)
		}

		prefs, _ := d.ListPreferences()
		npmKey := metaPref + "provider_npm_npm-provider"
		if prefs[npmKey] != "@org/package" {
			t.Errorf("expected NPM preference, got %q", prefs[npmKey])
		}
	})

	t.Run("preserves existing provider KeyEnv", func(t *testing.T) {
		t.Parallel()
		d, err := db.Open(":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer d.Close()

		d.UpsertProvider(&models.Provider{
			ID:      "openai",
			Name:    "OpenAI",
			KeyEnv:  "CUSTOM_OPENAI_KEY",
			Source:  "manual",
			Status:  "active",
		})

		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "opencode.jsonc")
		config := map[string]interface{}{
			"$schema": "https://opencode.ai/config.json",
			"provider": map[string]interface{}{
				"openai": map[string]interface{}{
					"name": "OpenAI",
				},
			},
		}
		out, _ := json.MarshalIndent(config, "", "  ")
		os.WriteFile(configPath, out, 0644)

		svc := New(d)
		_, err = svc.ImportFromOpenCodeConfig(configPath)
		if err != nil {
			t.Fatalf("import: %v", err)
		}

		p, err := d.GetProvider("openai")
		if err != nil {
			t.Fatal(err)
		}
		if p.KeyEnv != "CUSTOM_OPENAI_KEY" {
			t.Errorf("KeyEnv not preserved, got %q", p.KeyEnv)
		}
		if p.Source != "opencode" {
			t.Errorf("Source should be overwritten to opencode, got %q", p.Source)
		}
	})

	t.Run("LSP as object config", func(t *testing.T) {
		t.Parallel()
		d, err := db.Open(":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer d.Close()

		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "opencode.jsonc")
		config := map[string]interface{}{
			"$schema": "https://opencode.ai/config.json",
			"lsp": map[string]interface{}{
				"golang": map[string]interface{}{
					"command": []interface{}{"gopls", "serve"},
				},
			},
		}
		out, _ := json.MarshalIndent(config, "", "  ")
		os.WriteFile(configPath, out, 0644)

		svc := New(d)
		_, err = svc.ImportFromOpenCodeConfig(configPath)
		if err != nil {
			t.Fatalf("import: %v", err)
		}

		lsps, _ := d.ListLSPServers()
		if len(lsps) != 1 || lsps[0].ID != "golang" {
			t.Errorf("expected golang LSP, got %v", lsps)
		}

		prefs, _ := d.ListPreferences()
		if _, ok := prefs[metaPref+"lsp"]; ok {
			t.Errorf("lsp boolean preference should not be set for object config")
		}
	})

	t.Run("imports models from map", func(t *testing.T) {
		t.Parallel()
		d, err := db.Open(":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer d.Close()

		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "opencode.jsonc")
		config := map[string]interface{}{
			"$schema": "https://opencode.ai/config.json",
			"provider": map[string]interface{}{
				"test-provider": map[string]interface{}{
					"models": map[string]interface{}{
						"model-a": map[string]interface{}{},
						"model-b": map[string]interface{}{},
					},
				},
			},
		}
		out, _ := json.MarshalIndent(config, "", "  ")
		os.WriteFile(configPath, out, 0644)

		svc := New(d)
		diff, err := svc.ImportFromOpenCodeConfig(configPath)
		if err != nil {
			t.Fatalf("import: %v", err)
		}
		if len(diff.AddedModels) != 2 {
			t.Errorf("expected 2 models, got %v", diff.AddedModels)
		}
		models, _ := d.ListModelsByProvider("test-provider")
		if len(models) != 2 {
			t.Errorf("expected 2 models in provider, got %d", len(models))
		}
	})
}

func TestExportToOpenCodeConfig(t *testing.T) {
	t.Parallel()

	t.Run("empty DB", func(t *testing.T) {
		t.Parallel()
		d, err := db.Open(":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer d.Close()

		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "output.jsonc")

		svc := New(d)
		if err := svc.ExportToOpenCodeConfig(outputPath); err != nil {
			t.Fatalf("export: %v", err)
		}

		data, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatal(err)
		}
		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatal(err)
		}
		if result["$schema"] != "https://opencode.ai/config.json" {
			t.Errorf("missing schema in output")
		}
		// No providers should be exported from empty DB (seeded providers have no source=opencode)
		if _, ok := result["provider"]; !ok {
			t.Errorf("expected empty provider section")
		}
	})

	t.Run("with providers and models", func(t *testing.T) {
		t.Parallel()
		d, err := db.Open(":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer d.Close()

		d.UpsertProvider(&models.Provider{
			ID: "test-provider", Name: "Test Provider",
			Source: "opencode", Status: "active",
		})
		d.UpsertModel(&models.Model{
			ID: "test-provider/model-1", ProviderID: "test-provider",
			DisplayName: "model-1", Status: "active", Source: "opencode",
		})
		d.UpsertModel(&models.Model{
			ID: "test-provider/model-2", ProviderID: "test-provider",
			DisplayName: "model-2", Status: "active", Source: "opencode",
		})
		d.UpsertModelProfile(&models.ModelProfile{
			ModelID: "test-provider/model-1",
		})
		d.UpsertModelProfile(&models.ModelProfile{
			ModelID: "test-provider/model-2",
		})

		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "output.jsonc")

		svc := New(d)
		if err := svc.ExportToOpenCodeConfig(outputPath); err != nil {
			t.Fatalf("export: %v", err)
		}

		data, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatal(err)
		}
		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatal(err)
		}
		provSection, ok := result["provider"].(map[string]interface{})
		if !ok {
			t.Fatal("missing provider section")
		}
		entry, ok := provSection["test-provider"].(map[string]interface{})
		if !ok {
			t.Fatal("missing test-provider entry")
		}
		if entry["name"] != "Test Provider" {
			t.Errorf("expected name 'Test Provider', got %v", entry["name"])
		}
	})

	t.Run("excludes error models", func(t *testing.T) {
		t.Parallel()
		d, err := db.Open(":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer d.Close()

		d.UpsertProvider(&models.Provider{
			ID: "test-provider", Name: "Test",
			Source: "opencode", Status: "active",
		})
		d.UpsertModel(&models.Model{
			ID: "test-provider/good", ProviderID: "test-provider",
			DisplayName: "good", Status: "active", Source: "opencode",
		})
		d.UpsertModel(&models.Model{
			ID: "test-provider/bad", ProviderID: "test-provider",
			DisplayName: "bad", Status: "error", Source: "opencode",
		})

		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "output.jsonc")

		svc := New(d)
		if err := svc.ExportToOpenCodeConfig(outputPath); err != nil {
			t.Fatalf("export: %v", err)
		}

		data, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatal(err)
		}
		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatal(err)
		}
		provSection := result["provider"].(map[string]interface{})
		entry := provSection["test-provider"].(map[string]interface{})
		whitelist := entry["whitelist"].([]interface{})
		if len(whitelist) != 1 || whitelist[0] != "good" {
			t.Errorf("expected only 'good' in whitelist, got %v", whitelist)
		}
	})

	t.Run("with agents, commands, MCPs", func(t *testing.T) {
		t.Parallel()
		d, err := db.Open(":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer d.Close()

		d.UpsertProvider(&models.Provider{
			ID: "p", Name: "P", Source: "opencode", Status: "active",
		})
		d.UpsertAgent(&models.Agent{
			ID: "build", Description: "Build agent",
			CurrentModelID: "p/m1", Mode: "primary",
			Temperature: 0.7, Color: "blue",
			Status: "active", Source: "opencode",
		})
		d.UpsertAgent(&models.Agent{
			ID: "inactive-agent", Description: "Inactive",
			Status: "deprecated", Source: "opencode",
		})
		d.UpsertCommand(&models.Command{
			ID: "test", Template: "go test ./...",
			Description: "Run tests", Status: "active", Source: "opencode",
		})
		d.UpsertCommand(&models.Command{
			ID: "old-cmd", Template: "old",
			Status: "removed", Source: "opencode",
		})
		d.UpsertMCP(&models.MCPServer{
			ID: "local-mcp", Type: "local",
			Command: `["tool","run"]`,
			Enabled: true, Timeout: 30000,
			Source: "opencode",
		})
		d.UpsertMCP(&models.MCPServer{
			ID: "remote-mcp", Type: "remote",
			URL: "https://mcp.example.com",
			Source: "opencode",
		})

		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "output.jsonc")

		svc := New(d)
		if err := svc.ExportToOpenCodeConfig(outputPath); err != nil {
			t.Fatalf("export: %v", err)
		}

		data, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatal(err)
		}
		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatal(err)
		}

		agentSection := result["agent"].(map[string]interface{})
		if _, ok := agentSection["build"]; !ok {
			t.Error("expected build agent in export")
		}
		if _, ok := agentSection["inactive-agent"]; ok {
			t.Error("inactive agent should not be exported")
		}
		buildEntry := agentSection["build"].(map[string]interface{})
		if buildEntry["model"] != "p/m1" {
			t.Errorf("expected model 'p/m1', got %v", buildEntry["model"])
		}
		if buildEntry["mode"] != "primary" {
			t.Errorf("expected mode 'primary', got %v", buildEntry["mode"])
		}
		if buildEntry["temperature"] != 0.7 {
			t.Errorf("expected temperature 0.7, got %v", buildEntry["temperature"])
		}
		if buildEntry["color"] != "blue" {
			t.Errorf("expected color 'blue', got %v", buildEntry["color"])
		}

		cmdSection := result["command"].(map[string]interface{})
		if _, ok := cmdSection["test"]; !ok {
			t.Error("expected test command in export")
		}
		if _, ok := cmdSection["old-cmd"]; ok {
			t.Error("removed command should not be exported")
		}

		mcpSection := result["mcp"].(map[string]interface{})
		localEntry := mcpSection["local-mcp"].(map[string]interface{})
		if localEntry["type"] != "local" {
			t.Errorf("expected type local, got %v", localEntry["type"])
		}
		if localEntry["timeout"] != float64(30000) {
			t.Errorf("expected timeout 30000, got %v", localEntry["timeout"])
		}
		remoteEntry := mcpSection["remote-mcp"].(map[string]interface{})
		if remoteEntry["type"] != "remote" {
			t.Errorf("expected type remote, got %v", remoteEntry["type"])
		}
		if remoteEntry["url"] != "https://mcp.example.com" {
			t.Errorf("expected url, got %v", remoteEntry["url"])
		}
	})

	t.Run("with MCP env vars", func(t *testing.T) {
		t.Parallel()
		d, err := db.Open(":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer d.Close()

		d.UpsertMCP(&models.MCPServer{
			ID: "env-mcp", Type: "local",
			Command: `["my-mcp"]`,
			Enabled: true,
			EnvVars: `{"API_KEY":"sk-123","HOST":"localhost"}`,
			Source:  "opencode",
		})

		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "output.jsonc")

		svc := New(d)
		if err := svc.ExportToOpenCodeConfig(outputPath); err != nil {
			t.Fatalf("export: %v", err)
		}

		data, _ := os.ReadFile(outputPath)
		var result map[string]interface{}
		json.Unmarshal(data, &result)
		mcpSection := result["mcp"].(map[string]interface{})
		entry := mcpSection["env-mcp"].(map[string]interface{})
		env := entry["environment"].(map[string]interface{})
		if env["API_KEY"] != "sk-123" {
			t.Errorf("expected API_KEY in env, got %v", env)
		}
	})

	t.Run("with meta preferences", func(t *testing.T) {
		t.Parallel()
		d, err := db.Open(":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer d.Close()

		d.SetPreference(metaPref+"model", `"gpt-4"`)
		d.SetPreference(metaPref+"share", `"auto"`)
		d.SetPreference(metaPref+"provider_npm_test", "@test/pkg")
		d.SetPreference(metaPref+"skills_my-skill", `{"enabled":true}`)
		d.SetPreference(metaPref+"compaction_auto", `true`)

		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "output.jsonc")

		svc := New(d)
		if err := svc.ExportToOpenCodeConfig(outputPath); err != nil {
			t.Fatalf("export: %v", err)
		}

		data, _ := os.ReadFile(outputPath)
		var result map[string]interface{}
		json.Unmarshal(data, &result)

		if result["model"] != "gpt-4" {
			t.Errorf("expected model 'gpt-4' in export, got %v", result["model"])
		}
		if result["share"] != "auto" {
			t.Errorf("expected share 'auto' in export, got %v", result["share"])
		}
		// provider_, skills_, compaction_ prefs should be skipped
		if _, ok := result["provider_npm_test"]; ok {
			t.Errorf("provider_npm_test should not be exported as top-level key")
		}
		if _, ok := result["skills_my-skill"]; ok {
			t.Errorf("skills_my-skill should not be exported as top-level key")
		}
		if _, ok := result["compaction_auto"]; ok {
			t.Errorf("compaction_auto should not be exported as top-level key")
		}
	})

	t.Run("with agent permission and prompt", func(t *testing.T) {
		t.Parallel()
		d, err := db.Open(":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer d.Close()

		d.UpsertAgent(&models.Agent{
			ID: "agent-with-perms", Description: "Agent with permissions",
			CurrentModelID: "p/m", Mode: "subagent",
			MaxSteps: 50, PromptFile: "prompts/agent.md",
			Permission: `{"allow":["read","write"],"deny":["exec"]}`,
			Status: "active", Source: "opencode",
		})

		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "output.jsonc")

		svc := New(d)
		if err := svc.ExportToOpenCodeConfig(outputPath); err != nil {
			t.Fatalf("export: %v", err)
		}

		data, _ := os.ReadFile(outputPath)
		var result map[string]interface{}
		json.Unmarshal(data, &result)
		agentSection := result["agent"].(map[string]interface{})
		entry := agentSection["agent-with-perms"].(map[string]interface{})
		if entry["steps"] != float64(50) {
			t.Errorf("expected steps 50, got %v", entry["steps"])
		}
		if entry["prompt"] != "prompts/agent.md" {
			t.Errorf("expected prompt file, got %v", entry["prompt"])
		}
		perm := entry["permission"].(map[string]interface{})
		if len(perm["allow"].([]interface{})) != 2 {
			t.Errorf("expected 2 allow permissions, got %v", perm["allow"])
		}
	})

	t.Run("provider with NPM and options preferences", func(t *testing.T) {
		t.Parallel()
		d, err := db.Open(":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer d.Close()

		d.UpsertProvider(&models.Provider{
			ID: "custom-provider", Name: "Custom",
			Source: "opencode", Status: "active",
		})
		d.SetPreference(metaPref+"provider_npm_custom-provider", "@custom/pkg")
		d.SetPreference(metaPref+"provider_options_custom-provider", `{"baseURL":"https://custom.api","apiVersion":"v2"}`)

		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "output.jsonc")

		svc := New(d)
		if err := svc.ExportToOpenCodeConfig(outputPath); err != nil {
			t.Fatalf("export: %v", err)
		}

		data, _ := os.ReadFile(outputPath)
		var result map[string]interface{}
		json.Unmarshal(data, &result)
		provSection := result["provider"].(map[string]interface{})
		entry := provSection["custom-provider"].(map[string]interface{})

		if entry["npm"] != "@custom/pkg" {
			t.Errorf("expected npm '@custom/pkg', got %v", entry["npm"])
		}
		opts := entry["options"].(map[string]interface{})
		if opts["baseURL"] != "https://custom.api" {
			t.Errorf("expected baseURL in options, got %v", opts)
		}
		if opts["apiVersion"] != "v2" {
			t.Errorf("expected apiVersion in options, got %v", opts)
		}
	})

	t.Run("provider with BaseURL fallback when no options preference", func(t *testing.T) {
		t.Parallel()
		d, err := db.Open(":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer d.Close()

		d.UpsertProvider(&models.Provider{
			ID: "baseurl-provider", Name: "BaseURL Only",
			BaseURL: "https://api.example.com",
			Source: "opencode", Status: "active",
		})

		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "output.jsonc")

		svc := New(d)
		if err := svc.ExportToOpenCodeConfig(outputPath); err != nil {
			t.Fatalf("export: %v", err)
		}

		data, _ := os.ReadFile(outputPath)
		var result map[string]interface{}
		json.Unmarshal(data, &result)
		provSection := result["provider"].(map[string]interface{})
		entry := provSection["baseurl-provider"].(map[string]interface{})
		opts := entry["options"].(map[string]interface{})
		if opts["baseURL"] != "https://api.example.com" {
			t.Errorf("expected baseURL fallback, got %v", opts)
		}
	})

	t.Run("command with agent, model, subtask fields", func(t *testing.T) {
		t.Parallel()
		d, err := db.Open(":memory:")
		if err != nil {
			t.Fatal(err)
		}
		defer d.Close()

		d.UpsertCommand(&models.Command{
			ID: "complex-cmd", Template: "run {{.args}}",
			Description: "Complex command", Agent: "build",
			Model: "gpt-4", Subtask: true,
			Status: "active", Source: "opencode",
		})

		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "output.jsonc")

		svc := New(d)
		if err := svc.ExportToOpenCodeConfig(outputPath); err != nil {
			t.Fatalf("export: %v", err)
		}

		data, _ := os.ReadFile(outputPath)
		var result map[string]interface{}
		json.Unmarshal(data, &result)
		cmdSection := result["command"].(map[string]interface{})
		entry := cmdSection["complex-cmd"].(map[string]interface{})

		if entry["agent"] != "build" {
			t.Errorf("expected agent 'build', got %v", entry["agent"])
		}
		if entry["model"] != "gpt-4" {
			t.Errorf("expected model 'gpt-4', got %v", entry["model"])
		}
		if entry["subtask"] != true {
			t.Errorf("expected subtask true, got %v", entry["subtask"])
		}
	})
}
