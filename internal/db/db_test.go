package db_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/reeinharrrd/maestro/internal/db"
	"github.com/reeinharrrd/maestro/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openTestDB(t *testing.T) *db.DB {
	t.Helper()
	d, err := db.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { d.Close() })
	return d
}

// ── Open / Close / Migrate ────────────────────────────────────────

func TestOpen_DefaultPath_ReturnsDB(t *testing.T) {
	t.Parallel()
	d, err := db.Open(":memory:")
	require.NoError(t, err)
	require.NotNil(t, d)
	assert.Equal(t, ":memory:", d.DBPath())
	assert.NoError(t, d.Close())
}

func TestOpen_EmptyPath_ReturnsDefault(t *testing.T) {
	t.Parallel()
	d, err := db.Open("")
	require.NoError(t, err)
	defer d.Close()
	assert.NotEmpty(t, d.DBPath())
}

func TestMigrate_ExistingDB_NoError(t *testing.T) {
	t.Parallel()
	raw, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer raw.Close()
	assert.NoError(t, db.Migrate(raw))
	// second migrate should be no-op
	assert.NoError(t, db.Migrate(raw))
}

func TestNow_ReturnsRFC3339(t *testing.T) {
	t.Parallel()
	d := openTestDB(t)
	now := d.Now()
	assert.Regexp(t, `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$`, now)
}

func TestExecLog_WritesLogEntry(t *testing.T) {
	t.Parallel()
	d := openTestDB(t)
	err := d.ExecLog("test-phase", "ok", "all good", 0)
	assert.NoError(t, err)
}

// ── Provider CRUD ─────────────────────────────────────────────────

func TestProviderCRUD(t *testing.T) {
	t.Parallel()
	d := openTestDB(t)

	t.Run("Upsert and Get", func(t *testing.T) {
		p := &models.Provider{
			ID:     "test-provider",
			Name:   "Test Provider",
			Status: "active",
			Source: "manual",
		}
		require.NoError(t, d.UpsertProvider(p))

		got, err := d.GetProvider("test-provider")
		require.NoError(t, err)
		assert.Equal(t, p.ID, got.ID)
		assert.Equal(t, p.Name, got.Name)
		assert.Equal(t, "active", got.Status)
	})

	t.Run("Update existing", func(t *testing.T) {
		require.NoError(t, d.UpsertProvider(&models.Provider{
			ID:     "test-provider",
			Name:   "Updated Provider",
			Status: "inactive",
			Source: "manual",
		}))
		got, err := d.GetProvider("test-provider")
		require.NoError(t, err)
		assert.Equal(t, "Updated Provider", got.Name)
		assert.Equal(t, "inactive", got.Status)
	})

	t.Run("ListProviders", func(t *testing.T) {
		all, err := d.ListProviders()
		require.NoError(t, err)
		ids := make(map[string]bool)
		for _, p := range all {
			ids[p.ID] = true
		}
		assert.True(t, ids["test-provider"], "test-provider should be in list")
	})

	t.Run("Get nonexistent returns error", func(t *testing.T) {
		_, err := d.GetProvider("no-such-provider")
		assert.Error(t, err)
	})

	t.Run("DeleteProvider removes models too", func(t *testing.T) {
		prov := &models.Provider{ID: "del-prov", Name: "Delete Me", Status: "active", Source: "manual"}
		require.NoError(t, d.UpsertProvider(prov))
		mod := &models.Model{
			ID:         "del-prov/test-model",
			ProviderID: "del-prov",
			DisplayName: "test-model",
			Status:     "active",
		}
		require.NoError(t, d.UpsertModel(mod))

		require.NoError(t, d.DeleteProvider("del-prov"))
		_, err := d.GetProvider("del-prov")
		assert.Error(t, err)
		_, err = d.GetModel("del-prov/test-model")
		assert.Error(t, err)
	})
}

// ── Model CRUD ────────────────────────────────────────────────────

func TestModelCRUD(t *testing.T) {
	t.Parallel()
	d := openTestDB(t)

	// seed a provider
	require.NoError(t, d.UpsertProvider(&models.Provider{
		ID: "modeltest-prov", Name: "Model Test", Status: "active", Source: "manual",
	}))

	t.Run("Upsert and Get", func(t *testing.T) {
		m := &models.Model{
			ID:              "modeltest-prov/gpt-test",
			ProviderID:      "modeltest-prov",
			DisplayName:     "gpt-test",
			ContextWindow:    8192,
			FunctionCalling: true,
			Vision:          false,
			Streaming:       true,
			Tier:            "quality",
			Status:          "active",
			Source:          "discovered",
		}
		require.NoError(t, d.UpsertModel(m))

		got, err := d.GetModel("modeltest-prov/gpt-test")
		require.NoError(t, err)
		assert.Equal(t, m.ID, got.ID)
		assert.Equal(t, m.ProviderID, got.ProviderID)
		assert.Equal(t, 8192, got.ContextWindow)
		assert.True(t, got.FunctionCalling)
		assert.True(t, got.Streaming)
	})

	t.Run("ListModels", func(t *testing.T) {
		all, err := d.ListModels()
		require.NoError(t, err)
		assert.True(t, len(all) > 0)
	})

	t.Run("ListModelsByProvider", func(t *testing.T) {
		all, err := d.ListModelsByProvider("modeltest-prov")
		require.NoError(t, err)
		assert.Equal(t, 1, len(all))
		assert.Equal(t, "modeltest-prov/gpt-test", all[0].ID)
	})

	t.Run("Get nonexistent model", func(t *testing.T) {
		_, err := d.GetModel("no-such-model")
		assert.Error(t, err)
	})

	t.Run("DeleteModel", func(t *testing.T) {
		m := &models.Model{
			ID: "modeltest-prov/to-delete", ProviderID: "modeltest-prov",
			DisplayName: "to-delete", Status: "active",
		}
		require.NoError(t, d.UpsertModel(m))
		require.NoError(t, d.DeleteModel("modeltest-prov/to-delete"))
		_, err := d.GetModel("modeltest-prov/to-delete")
		assert.Error(t, err)
	})
}

// ── Model Filters ─────────────────────────────────────────────────

func TestListModels_Filters(t *testing.T) {
	t.Parallel()
	d := openTestDB(t)
	require.NoError(t, d.UpsertProvider(&models.Provider{
		ID: "filter-prov", Name: "Filter Test", Status: "active", Source: "manual",
	}))

	mods := []models.Model{
		{ID: "filter-prov/fc-model", ProviderID: "filter-prov", DisplayName: "fc-model", FunctionCalling: true, Tier: "quality", Status: "active"},
		{ID: "filter-prov/small", ProviderID: "filter-prov", DisplayName: "small", ContextWindow: 4000, Status: "active"},
		{ID: "filter-prov/large", ProviderID: "filter-prov", DisplayName: "large", ContextWindow: 128000, Status: "active"},
		{ID: "filter-prov/inactive", ProviderID: "filter-prov", DisplayName: "inactive", Status: "inactive"},
		{ID: "filter-prov/untested", ProviderID: "filter-prov", DisplayName: "untested", Status: "untested"},
	}
	for i := range mods {
		require.NoError(t, d.UpsertModel(&mods[i]))
	}

	t.Run("StatusActive filter", func(t *testing.T) {
		all, err := d.ListModels(db.StatusActive())
		require.NoError(t, err)
		for _, m := range all {
			assert.Equal(t, "active", m.Status)
		}
	})

	t.Run("HasFC filter", func(t *testing.T) {
		all, err := d.ListModels(db.HasFC())
		require.NoError(t, err)
		for _, m := range all {
			assert.True(t, m.FunctionCalling, "%s should have FC", m.ID)
		}
	})

	t.Run("MinContext filter", func(t *testing.T) {
		all, err := d.ListModels(db.MinContext(5000))
		require.NoError(t, err)
		for _, m := range all {
			assert.GreaterOrEqual(t, m.ContextWindow, 5000, "%s context", m.ID)
		}
	})

	t.Run("Tier filter", func(t *testing.T) {
		all, err := d.ListModels(db.Tier("quality"))
		require.NoError(t, err)
		for _, m := range all {
			assert.Equal(t, "quality", m.Tier)
		}
	})

	t.Run("StatusNot filter", func(t *testing.T) {
		all, err := d.ListModels(db.StatusNot("inactive"))
		require.NoError(t, err)
		for _, m := range all {
			assert.NotEqual(t, "inactive", m.Status, "%s should not be inactive", m.ID)
		}
	})

	t.Run("Combined filters", func(t *testing.T) {
		all, err := d.ListModels(db.StatusActive(), db.HasFC())
		require.NoError(t, err)
		for _, m := range all {
			assert.Equal(t, "active", m.Status)
			assert.True(t, m.FunctionCalling)
		}
	})
}

func TestSearchModels(t *testing.T) {
	t.Parallel()
	d := openTestDB(t)
	require.NoError(t, d.UpsertProvider(&models.Provider{
		ID: "search-prov", Name: "Search Test", Status: "active", Source: "manual",
	}))

	require.NoError(t, d.UpsertModel(&models.Model{
		ID: "search-prov/gpt-4", ProviderID: "search-prov",
		DisplayName: "GPT-4", Description: "OpenAI flagship model",
		Status: "active",
	}))
	require.NoError(t, d.UpsertModel(&models.Model{
		ID: "search-prov/claude-3", ProviderID: "search-prov",
		DisplayName: "Claude 3", Description: "Anthropic model",
		Status: "active",
	}))

	t.Run("Search by ID", func(t *testing.T) {
		res, err := d.SearchModels("gpt-4")
		require.NoError(t, err)
		assert.Equal(t, 1, len(res))
		assert.Equal(t, "search-prov/gpt-4", res[0].ID)
	})

	t.Run("Search by description", func(t *testing.T) {
		res, err := d.SearchModels("Anthropic")
		require.NoError(t, err)
		assert.Equal(t, 1, len(res))
	})

	t.Run("Search no results", func(t *testing.T) {
		res, err := d.SearchModels("zzz_nonexistent")
		require.NoError(t, err)
		assert.Equal(t, 0, len(res))
	})
}

func TestGetStats(t *testing.T) {
	t.Parallel()
	d := openTestDB(t)
	require.NoError(t, d.UpsertProvider(&models.Provider{
		ID: "stats-prov", Name: "Stats Test", Status: "active", Source: "manual",
	}))
	require.NoError(t, d.UpsertModel(&models.Model{
		ID: "stats-prov/stats-a", ProviderID: "stats-prov",
		DisplayName: "stats-a", Status: "active",
	}))
	require.NoError(t, d.UpsertModel(&models.Model{
		ID: "stats-prov/stats-b", ProviderID: "stats-prov",
		DisplayName: "stats-b", Status: "untested",
	}))

	stats, err := d.GetStats()
	require.NoError(t, err)
	assert.Equal(t, 1, stats["active"])
	assert.Equal(t, 1, stats["untested"])
	assert.Equal(t, 1, stats["providers_active"])
}

func TestGetSmallFastModels(t *testing.T) {
	t.Parallel()
	d := openTestDB(t)
	require.NoError(t, d.UpsertProvider(&models.Provider{
		ID: "fast-prov", Name: "Fast Test", Status: "active", Source: "manual",
	}))

	// This model has low latency and low cost → should appear in small-fast list
	require.NoError(t, d.UpsertModel(&models.Model{
		ID: "fast-prov/fast-model", ProviderID: "fast-prov",
		DisplayName: "Fast Model", LatencyP50Ms: 100,
		PricingPrompt: 0.001, PricingCompletion: 0.002,
		Tier: "free", Status: "active",
	}))

	// This model has high latency → should NOT appear
	require.NoError(t, d.UpsertModel(&models.Model{
		ID: "fast-prov/slow-model", ProviderID: "fast-prov",
		DisplayName: "Slow Model", LatencyP50Ms: 1000,
		PricingPrompt: 0.001, PricingCompletion: 0.002,
		Tier: "free", Status: "active",
	}))

	results, err := d.GetSmallFastModels(context.Background())
	require.NoError(t, err)
	ids := make(map[string]bool)
	for _, m := range results {
		ids[m.ID] = true
	}
	assert.True(t, ids["fast-prov/fast-model"], "fast model should be in results")
	assert.False(t, ids["fast-prov/slow-model"], "slow model should NOT be in results")
}

// ── Agent CRUD ────────────────────────────────────────────────────

func TestAgentCRUD(t *testing.T) {
	t.Parallel()
	d := openTestDB(t)

	t.Run("Upsert and List", func(t *testing.T) {
		require.NoError(t, d.UpsertAgent(&models.Agent{
			ID: "test-agent", TaskType: "coding", Description: "Test agent",
			Status: "active", Source: "manual",
		}))
		all, err := d.ListAgents()
		require.NoError(t, err)
		found := false
		for _, a := range all {
			if a.ID == "test-agent" {
				found = true
				assert.Equal(t, "coding", a.TaskType)
			}
		}
		assert.True(t, found)
	})

	t.Run("GetAgent", func(t *testing.T) {
		a, err := d.GetAgent("test-agent")
		require.NoError(t, err)
		assert.Equal(t, "Test agent", a.Description)
	})

	t.Run("Get nonexistent agent", func(t *testing.T) {
		_, err := d.GetAgent("no-such-agent")
		assert.Error(t, err)
	})

	t.Run("DeleteAgent", func(t *testing.T) {
		require.NoError(t, d.UpsertAgent(&models.Agent{
			ID: "del-agent", Status: "active", Source: "manual",
		}))
		require.NoError(t, d.DeleteAgent("del-agent"))
		_, err := d.GetAgent("del-agent")
		assert.Error(t, err)
	})

	t.Run("DeleteAgent nonexistent returns error", func(t *testing.T) {
		err := d.DeleteAgent("no-such-agent")
		assert.Error(t, err)
	})
}

// ── Command CRUD ──────────────────────────────────────────────────

func TestCommandCRUD(t *testing.T) {
	t.Parallel()
	d := openTestDB(t)

	t.Run("Upsert and List", func(t *testing.T) {
		require.NoError(t, d.UpsertCommand(&models.Command{
			ID: "test-cmd", Template: "echo {{.Input}}",
			Description: "Test command", Status: "active",
		}))
		all, err := d.ListCommands()
		require.NoError(t, err)
		found := false
		for _, c := range all {
			if c.ID == "test-cmd" {
				found = true
				assert.Equal(t, "echo {{.Input}}", c.Template)
			}
		}
		assert.True(t, found)
	})

	t.Run("Upsert updates existing", func(t *testing.T) {
		require.NoError(t, d.UpsertCommand(&models.Command{
			ID: "test-cmd", Template: "updated", Status: "inactive",
		}))
		all, _ := d.ListCommands()
		for _, c := range all {
			if c.ID == "test-cmd" {
				assert.Equal(t, "updated", c.Template)
				assert.Equal(t, "inactive", c.Status)
			}
		}
	})
}

// ── MCP CRUD ──────────────────────────────────────────────────────

func TestMCPCRUD(t *testing.T) {
	t.Parallel()
	d := openTestDB(t)

	t.Run("Upsert and List", func(t *testing.T) {
		require.NoError(t, d.UpsertMCP(&models.MCPServer{
			ID: "test-mcp", Type: "local", Command: `["echo"]`,
			Enabled: true, Timeout: 5000,
		}))
		all, err := d.ListMCPs()
		require.NoError(t, err)
		found := false
		for _, m := range all {
			if m.ID == "test-mcp" {
				found = true
				assert.Equal(t, "local", m.Type)
				assert.True(t, m.Enabled)
			}
		}
		assert.True(t, found)
	})
}

// ── Skill CRUD ────────────────────────────────────────────────────

func TestSkillCRUD(t *testing.T) {
	t.Parallel()
	d := openTestDB(t)
	s := &models.Skill{
		ID: "test-skill", Source: "github", Type: "skill", Status: "active",
	}
	require.NoError(t, d.UpsertSkill(s))
	all, err := d.ListSkills()
	require.NoError(t, err)
	found := false
	for _, sk := range all {
		if sk.ID == "test-skill" {
			found = true
		}
	}
	assert.True(t, found)
}

// ── Source CRUD ───────────────────────────────────────────────────

func TestSourceCRUD(t *testing.T) {
	t.Parallel()
	d := openTestDB(t)

	t.Run("Upsert and List", func(t *testing.T) {
		require.NoError(t, d.UpsertSource(&models.Source{
			ID: "test-source", RemoteURL: "https://example.com/repo",
			LocalPath: "/tmp/repo", Status: "active",
		}))
		all, err := d.ListSources()
		require.NoError(t, err)
		found := false
		for _, s := range all {
			if s.ID == "test-source" {
				found = true
				assert.Equal(t, "https://example.com/repo", s.RemoteURL)
			}
		}
		assert.True(t, found)
	})
}

// ── SourceItem CRUD ───────────────────────────────────────────────

func TestSourceItemCRUD(t *testing.T) {
	t.Parallel()
	d := openTestDB(t)

	require.NoError(t, d.UpsertSource(&models.Source{
		ID: "si-source", RemoteURL: "https://example.com/si", Status: "active",
	}))

	item := &models.SourceItem{
		ID: "si-1", SourceID: "si-source", Type: "skill",
		SourcePath: "skills/test", Status: "active",
	}

	t.Run("Upsert and Get", func(t *testing.T) {
		require.NoError(t, d.UpsertSourceItem(item))
		got, err := d.GetSourceItem("si-1")
		require.NoError(t, err)
		assert.Equal(t, "si-source", got.SourceID)
	})

	t.Run("List", func(t *testing.T) {
		all, err := d.ListSourceItems()
		require.NoError(t, err)
		found := false
		for _, s := range all {
			if s.ID == "si-1" {
				found = true
			}
		}
		assert.True(t, found)
	})

	t.Run("Delete", func(t *testing.T) {
		require.NoError(t, d.DeleteSourceItem("si-1"))
		_, err := d.GetSourceItem("si-1")
		assert.Error(t, err)
	})
}

// ── LSP Server CRUD ───────────────────────────────────────────────

func TestLSPServerCRUD(t *testing.T) {
	t.Parallel()
	d := openTestDB(t)

	t.Run("Upsert and List", func(t *testing.T) {
		require.NoError(t, d.UpsertLSPServer(&models.LSPServer{
			ID: "test-lsp", Command: `["gopls"]`,
			Extensions: `[".go"]`,
		}))
		all, err := d.ListLSPServers()
		require.NoError(t, err)
		found := false
		for _, l := range all {
			if l.ID == "test-lsp" {
				found = true
				assert.Equal(t, `["gopls"]`, l.Command)
			}
		}
		assert.True(t, found)
	})

	t.Run("GetLSPServer", func(t *testing.T) {
		l, err := d.GetLSPServer("test-lsp")
		require.NoError(t, err)
		assert.Equal(t, `[".go"]`, l.Extensions)
	})

	t.Run("DeleteLSPServer", func(t *testing.T) {
		require.NoError(t, d.DeleteLSPServer("test-lsp"))
		_, err := d.GetLSPServer("test-lsp")
		assert.Error(t, err)
	})
}

// ── Config Fragment CRUD ──────────────────────────────────────────

func TestConfigFragmentCRUD(t *testing.T) {
	t.Parallel()
	d := openTestDB(t)

	t.Run("Upsert and Get", func(t *testing.T) {
		require.NoError(t, d.UpsertConfigFragment(&models.ConfigFragment{
			ID: "cf-test", ConfigType: "json", Content: `{"key":"val"}`,
			Source: "manual", Hash: "abc123",
		}))
		got, err := d.GetConfigFragment("cf-test")
		require.NoError(t, err)
		assert.Equal(t, "json", got.ConfigType)
		assert.Equal(t, `{"key":"val"}`, got.Content)
	})

	t.Run("List respects limit", func(t *testing.T) {
		require.NoError(t, d.UpsertConfigFragment(&models.ConfigFragment{
			ID: "cf-2", ConfigType: "yaml", Content: "key: val", Source: "manual",
		}))
		all, err := d.ListConfigFragments(1)
		require.NoError(t, err)
		assert.Equal(t, 1, len(all))
	})
}

// ── Model Profile CRUD ────────────────────────────────────────────

func TestModelProfileCRUD(t *testing.T) {
	t.Parallel()
	d := openTestDB(t)
	require.NoError(t, d.UpsertProvider(&models.Provider{
		ID: "mp-prov", Name: "MP Test", Status: "active", Source: "manual",
	}))
	require.NoError(t, d.UpsertModel(&models.Model{
		ID: "mp-prov/mp-model", ProviderID: "mp-prov",
		DisplayName: "mp-model", Status: "active",
	}))

	prof := &models.ModelProfile{
		ModelID: "mp-prov/mp-model", RealContext: 64000,
		MaxOutput: 4096, SupportsStream: true, SupportsSO: true,
	}

	t.Run("Upsert and Get", func(t *testing.T) {
		require.NoError(t, d.UpsertModelProfile(prof))
		got, err := d.GetModelProfile("mp-prov/mp-model")
		require.NoError(t, err)
		assert.Equal(t, 64000, got.RealContext)
		assert.True(t, got.SupportsStream)
		assert.True(t, got.SupportsSO)
	})

	t.Run("List", func(t *testing.T) {
		all, err := d.ListModelProfiles()
		require.NoError(t, err)
		found := false
		for _, p := range all {
			if p.ModelID == "mp-prov/mp-model" {
				found = true
			}
		}
		assert.True(t, found)
	})
}

// ── Routing ───────────────────────────────────────────────────────

func TestRoutingRuleCRUD(t *testing.T) {
	t.Parallel()
	d := openTestDB(t)

	r := &models.RoutingRule{
		TaskKey: "test-route", Description: "Test rule",
		MinContext: 50000, NeedsFC: true,
		CurrentModelID: "model-a",
	}

	t.Run("Upsert and Get", func(t *testing.T) {
		require.NoError(t, d.UpsertRoutingRule(r))
		got, err := d.GetRoutingRule("test-route")
		require.NoError(t, err)
		assert.Equal(t, "Test rule", got.Description)
		assert.True(t, got.NeedsFC)
		assert.Equal(t, "model-a", got.CurrentModelID)
	})

	t.Run("List", func(t *testing.T) {
		all, err := d.ListRoutingRules()
		require.NoError(t, err)
		found := false
		for _, rr := range all {
			if rr.TaskKey == "test-route" {
				found = true
			}
		}
		assert.True(t, found)
	})
}

func TestRoutingEvent(t *testing.T) {
	t.Parallel()
	d := openTestDB(t)

	ev := &models.RoutingEvent{
		TaskKey: "event-test", SelectedModel: "model-a",
		Candidates: `["model-a","model-b"]`, Reason: "latency",
	}
	require.NoError(t, d.InsertRoutingEvent(ev))

	events, err := d.ListRoutingEvents(5)
	require.NoError(t, err)
	assert.Equal(t, 1, len(events))
	assert.Equal(t, "event-test", events[0].TaskKey)
	assert.Equal(t, "model-a", events[0].SelectedModel)
}

func TestListRoutingEvents_ZeroLimitDefault(t *testing.T) {
	t.Parallel()
	d := openTestDB(t)
	events, err := d.ListRoutingEvents(0)
	require.NoError(t, err)
	// default limit is 20, with no entries it returns empty list
	assert.Equal(t, 0, len(events))
}

// ── Budget ─────────────────────────────────────────────────────────

func TestBudgetCRUD(t *testing.T) {
	t.Parallel()
	d := openTestDB(t)

	t.Run("GetBudget has seed default", func(t *testing.T) {
		b, err := d.GetBudget()
		require.NoError(t, err)
		assert.Equal(t, "default", b.ID)
		assert.Equal(t, 0.50, b.DailyGlobalUSD)
		assert.Equal(t, "free_only", b.PreferredTier)
	})

	t.Run("Upsert updates budget", func(t *testing.T) {
		require.NoError(t, d.UpsertBudget(&models.BudgetConfig{
			ID: "default", DailyGlobalUSD: 1.00, PreferredTier: "quality",
		}))
		b, err := d.GetBudget()
		require.NoError(t, err)
		assert.Equal(t, 1.00, b.DailyGlobalUSD)
		assert.Equal(t, "quality", b.PreferredTier)
	})
}

// ── Preferences ───────────────────────────────────────────────────

func TestPreferenceCRUD(t *testing.T) {
	t.Parallel()
	d := openTestDB(t)

	t.Run("Set and Get", func(t *testing.T) {
		require.NoError(t, d.SetPreference("theme", "dark"))
		val, err := d.GetPreference("theme")
		require.NoError(t, err)
		assert.Equal(t, "dark", val)
	})

	t.Run("Update existing", func(t *testing.T) {
		require.NoError(t, d.SetPreference("theme", "light"))
		val, _ := d.GetPreference("theme")
		assert.Equal(t, "light", val)
	})

	t.Run("ListPreferences", func(t *testing.T) {
		all, err := d.ListPreferences()
		require.NoError(t, err)
		assert.Equal(t, "light", all["theme"])
	})

	t.Run("DeletePreference", func(t *testing.T) {
		require.NoError(t, d.DeletePreference("theme"))
		_, err := d.GetPreference("theme")
		assert.Error(t, err)
	})

	t.Run("Get nonexistent preference returns error", func(t *testing.T) {
		_, err := d.GetPreference("no-such-pref")
		assert.Error(t, err)
	})
}

func TestCleanupPreferences(t *testing.T) {
	t.Parallel()
	d := openTestDB(t)

	require.NoError(t, d.SetPreference("config/provider_test", `{"key":"val"}`))
	require.NoError(t, d.SetPreference("theme", "dark"))
	require.NoError(t, d.SetPreference("null_val", "null"))

	n, err := d.CleanupProviderPrefs()
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	n2, err := d.CleanupInvalidPreferences()
	require.NoError(t, err)
	assert.Equal(t, 1, n2)
}

// ── Sync Log ──────────────────────────────────────────────────────

func TestSyncLog(t *testing.T) {
	t.Parallel()
	d := openTestDB(t)

	t.Run("Insert and List", func(t *testing.T) {
		require.NoError(t, d.InsertSyncLog("sync-provs", "done", "All synced", 1500))
		logs, err := d.ListSyncLogs(5)
		require.NoError(t, err)
		assert.Equal(t, 1, len(logs))
		assert.Equal(t, "sync-provs", logs[0].Phase)
		assert.Equal(t, "done", logs[0].Status)
		assert.Equal(t, int64(1500), logs[0].DurationMs)
	})

	t.Run("List zero limit defaults", func(t *testing.T) {
		logs, err := d.ListSyncLogs(0)
		require.NoError(t, err)
		assert.Equal(t, 1, len(logs)) // from previous insert
	})
}

// ── Exec Log ──────────────────────────────────────────────────────

func TestExecLog(t *testing.T) {
	t.Parallel()
	d := openTestDB(t)

	t.Run("Insert and List", func(t *testing.T) {
		require.NoError(t, d.InsertExecLog(&models.ExecLog{
			Agent: "sisyphus", Model: "gpt-4", Task: "classify",
			TokensIn: 100, TokensOut: 50, DurationMs: 2000, Success: true,
		}))
		logs, err := d.ListExecLogs(5)
		require.NoError(t, err)
		assert.Equal(t, 1, len(logs))
		assert.Equal(t, "sisyphus", logs[0].Agent)
		assert.Equal(t, "gpt-4", logs[0].Model)
		assert.True(t, logs[0].Success)
	})
}

// ── Snapshots ─────────────────────────────────────────────────────

func TestSnapshotCRUD(t *testing.T) {
	t.Parallel()
	d := openTestDB(t)

	t.Run("Insert and List", func(t *testing.T) {
		require.NoError(t, d.InsertSnapshot("hash1", "content1"))
		all, err := d.ListSnapshots(5)
		require.NoError(t, err)
		assert.Equal(t, 1, len(all))
		assert.Equal(t, "hash1", all[0].Hash)
		assert.Equal(t, "content1", all[0].Content)
	})

	t.Run("GetSnapshot", func(t *testing.T) {
		all, _ := d.ListSnapshots(5)
		s, err := d.GetSnapshot(all[0].ID)
		require.NoError(t, err)
		assert.Equal(t, "hash1", s.Hash)
	})

	t.Run("DeleteSnapshot", func(t *testing.T) {
		all, _ := d.ListSnapshots(5)
		require.NoError(t, d.DeleteSnapshot(all[0].ID))
		_, err := d.GetSnapshot(all[0].ID)
		assert.Error(t, err)
	})
}

// ── DefaultPath / Edge cases ──────────────────────────────────────

func TestDefaultPath_NonEmpty(t *testing.T) {
	t.Parallel()
	p := db.DefaultPath()
	assert.Contains(t, p, "opencode-kit.db")
}

// ── Project CRUD ────────────────────────────────────────────────────

func TestProjectCRUD(t *testing.T) {
	t.Parallel()
	d := openTestDB(t)

	t.Run("Upsert and Get", func(t *testing.T) {
		p := &models.Project{
			ID:     "test-project",
			Path:   "/home/user/project",
			Name:   "Test Project",
			Status: "active",
			Source: "scan",
		}
		require.NoError(t, d.UpsertProject(p))

		got, err := d.GetProject("test-project")
		require.NoError(t, err)
		assert.Equal(t, p.ID, got.ID)
		assert.Equal(t, p.Path, got.Path)
		assert.Equal(t, p.Name, got.Name)
		assert.Equal(t, "active", got.Status)
	})

	t.Run("Upsert by path conflict updates existing", func(t *testing.T) {
		require.NoError(t, d.UpsertProject(&models.Project{
			ID:     "test-project",
			Path:   "/home/user/project",
			Name:   "Updated Project",
			Status: "stale",
			Source: "manual",
		}))
		got, err := d.GetProject("test-project")
		require.NoError(t, err)
		assert.Equal(t, "Updated Project", got.Name)
		assert.Equal(t, "stale", got.Status)
		assert.Equal(t, "manual", got.Source)
	})

	t.Run("ListProjects", func(t *testing.T) {
		all, err := d.ListProjects()
		require.NoError(t, err)
		ids := make(map[string]bool)
		for _, p := range all {
			ids[p.ID] = true
		}
		assert.True(t, ids["test-project"], "test-project should be in list")
	})

	t.Run("Get nonexistent returns error", func(t *testing.T) {
		_, err := d.GetProject("no-such-project")
		assert.Error(t, err)
	})

	t.Run("DeleteProject", func(t *testing.T) {
		require.NoError(t, d.UpsertProject(&models.Project{
			ID:     "del-project",
			Path:   "/path/to/delete",
			Name:   "Delete Me",
			Status: "active",
			Source: "scan",
		}))
		require.NoError(t, d.DeleteProject("del-project"))
		_, err := d.GetProject("del-project")
		assert.Error(t, err)
	})
}

// ── DetectedStack CRUD ──────────────────────────────────────────────

func TestDetectedStackCRUD(t *testing.T) {
	t.Parallel()
	d := openTestDB(t)

	// seed a project
	require.NoError(t, d.UpsertProject(&models.Project{
		ID: "stacktest-proj", Path: "/stack/test", Name: "Stack Test",
		Status: "active", Source: "scan",
	}))

	t.Run("Upsert and List", func(t *testing.T) {
		require.NoError(t, d.UpsertDetectedStack(&models.DetectedStack{
			ID: "stack-1", ProjectID: "stacktest-proj",
			Language: "go", Framework: "gin", Confidence: 0.95,
		}))
		all, err := d.ListDetectedStacks("stacktest-proj")
		require.NoError(t, err)
		found := false
		for _, s := range all {
			if s.ID == "stack-1" {
				found = true
				assert.Equal(t, "go", s.Language)
				assert.Equal(t, "gin", s.Framework)
				assert.Equal(t, 0.95, s.Confidence)
			}
		}
		assert.True(t, found)
	})

	t.Run("Upsert by language conflict updates existing", func(t *testing.T) {
		require.NoError(t, d.UpsertDetectedStack(&models.DetectedStack{
			ID: "stack-1", ProjectID: "stacktest-proj",
			Language: "go", Framework: "echo", Confidence: 0.80,
		}))
		all, err := d.ListDetectedStacks("stacktest-proj")
		require.NoError(t, err)
		have := 0
		for _, s := range all {
			if s.ProjectID == "stacktest-proj" {
				have++
				if s.ID == "stack-1" {
					assert.Equal(t, "echo", s.Framework)
					assert.Equal(t, 0.80, s.Confidence)
				}
			}
		}
		assert.Equal(t, 1, have, "should have exactly 1 stack for the project")
	})

	t.Run("List foreign project returns empty", func(t *testing.T) {
		all, err := d.ListDetectedStacks("no-such-project")
		require.NoError(t, err)
		assert.Equal(t, 0, len(all))
	})

	t.Run("DeleteDetectedStacks", func(t *testing.T) {
		require.NoError(t, d.UpsertDetectedStack(&models.DetectedStack{
			ID: "stack-del", ProjectID: "stacktest-proj",
			Language: "python", Confidence: 1.0,
		}))
		require.NoError(t, d.DeleteDetectedStacks("stacktest-proj"))
		all, err := d.ListDetectedStacks("stacktest-proj")
		require.NoError(t, err)
		assert.Equal(t, 0, len(all))
	})
}

// ── ProjectConfig CRUD ──────────────────────────────────────────────

func TestProjectConfigCRUD(t *testing.T) {
	t.Parallel()
	d := openTestDB(t)

	// seed a project
	require.NoError(t, d.UpsertProject(&models.Project{
		ID: "configtest-proj", Path: "/config/test", Name: "Config Test",
		Status: "active", Source: "scan",
	}))

	t.Run("Upsert and Get", func(t *testing.T) {
		require.NoError(t, d.UpsertProjectConfig(&models.ProjectConfig{
			ID: "cfg-1", ProjectID: "configtest-proj",
			ConfigType: "agents", Content: `{"agent":"test"}`,
			Hash: "abc123",
		}))
		got, err := d.GetProjectConfig("configtest-proj", "agents")
		require.NoError(t, err)
		assert.Equal(t, "cfg-1", got.ID)
		assert.Equal(t, `{"agent":"test"}`, got.Content)
		assert.Equal(t, "abc123", got.Hash)
	})

	t.Run("Upsert by config_type conflict updates existing", func(t *testing.T) {
		require.NoError(t, d.UpsertProjectConfig(&models.ProjectConfig{
			ID: "cfg-1", ProjectID: "configtest-proj",
			ConfigType: "agents", Content: `{"agent":"updated"}`,
			Hash: "def456",
		}))
		got, err := d.GetProjectConfig("configtest-proj", "agents")
		require.NoError(t, err)
		assert.Equal(t, `{"agent":"updated"}`, got.Content)
		assert.Equal(t, "def456", got.Hash)
	})

	t.Run("ListProjectConfigs", func(t *testing.T) {
		require.NoError(t, d.UpsertProjectConfig(&models.ProjectConfig{
			ID: "cfg-2", ProjectID: "configtest-proj",
			ConfigType: "mcps", Content: "{}", Hash: "zzz",
		}))
		all, err := d.ListProjectConfigs("configtest-proj")
		require.NoError(t, err)
		types := make(map[string]bool)
		for _, c := range all {
			types[c.ConfigType] = true
		}
		assert.True(t, types["agents"])
		assert.True(t, types["mcps"])
	})

	t.Run("Get nonexistent returns error", func(t *testing.T) {
		_, err := d.GetProjectConfig("configtest-proj", "skills")
		assert.Error(t, err)
	})

	t.Run("DeleteProjectConfigs", func(t *testing.T) {
		require.NoError(t, d.DeleteProjectConfigs("configtest-proj"))
		all, err := d.ListProjectConfigs("configtest-proj")
		require.NoError(t, err)
		assert.Equal(t, 0, len(all))
	})
}
