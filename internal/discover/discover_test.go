package discover_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/reeinharrrd/maestro/internal/db"
	"github.com/reeinharrrd/maestro/internal/discover"
	"github.com/reeinharrrd/maestro/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Mock DB for discover tests ────────────────────────────────────

type mockDB struct {
	providers     []models.Provider
	modelsByProv  map[string][]models.Model
	upsertedProv  []models.Provider
	upsertedMod   []models.Model
	upsertProvErr error
	upsertModErr  error
	listProvErr   error
}

func (m *mockDB) ListProviders() ([]models.Provider, error) {
	if m.listProvErr != nil {
		return nil, m.listProvErr
	}
	return m.providers, nil
}

func (m *mockDB) UpsertProvider(p *models.Provider) error {
	if m.upsertProvErr != nil {
		return m.upsertProvErr
	}
	m.upsertedProv = append(m.upsertedProv, *p)
	return nil
}

func (m *mockDB) UpsertModel(mod *models.Model) error {
	if m.upsertModErr != nil {
		return m.upsertModErr
	}
	m.upsertedMod = append(m.upsertedMod, *mod)
	return nil
}

func (m *mockDB) ListModelsByProvider(pid string) ([]models.Model, error) {
	if m.modelsByProv != nil {
		return m.modelsByProv[pid], nil
	}
	return nil, nil
}

func (m *mockDB) GetModel(id string) (*models.Model, error) {
	if m.modelsByProv != nil {
		for _, mods := range m.modelsByProv {
			for _, mod := range mods {
				if mod.ID == id {
					return &mod, nil
				}
			}
		}
	}
	return nil, assert.AnError
}

func (m *mockDB) DeleteModel(id string) error              { return nil }
func (m *mockDB) DeleteProvider(id string) error            { return nil }
func (m *mockDB) Query(query string, args ...any) (*sql.Rows, error)   { return nil, nil }
func (m *mockDB) Exec(query string, args ...any) (sql.Result, error)   { return nil, nil }
func (m *mockDB) ListModels(opts ...db.ModelFilter) ([]models.Model, error) { return nil, nil }

func (m *mockDB) GetProvider(id string) (*models.Provider, error) {
	for _, p := range m.providers {
		if p.ID == id {
			return &p, nil
		}
	}
	return nil, assert.AnError
}

// Stubs
func (m *mockDB) UpsertCommand(c *models.Command) error                          { return nil }
func (m *mockDB) ListCommands() ([]models.Command, error)                         { return nil, nil }
func (m *mockDB) UpsertMCP(mcp *models.MCPServer) error                           { return nil }
func (m *mockDB) ListMCPs() ([]models.MCPServer, error)                           { return nil, nil }
func (m *mockDB) UpsertSkill(s *models.Skill) error                               { return nil }
func (m *mockDB) ListSkills() ([]models.Skill, error)                             { return nil, nil }
func (m *mockDB) UpsertSourceItem(si *models.SourceItem) error                    { return nil }
func (m *mockDB) ListSourceItems() ([]models.SourceItem, error)                   { return nil, nil }
func (m *mockDB) GetSourceItem(id string) (*models.SourceItem, error)             { return nil, assert.AnError }
func (m *mockDB) DeleteSourceItem(id string) error                                { return nil }
func (m *mockDB) UpsertLSPServer(l *models.LSPServer) error                       { return nil }
func (m *mockDB) ListLSPServers() ([]models.LSPServer, error)                     { return nil, nil }
func (m *mockDB) GetLSPServer(id string) (*models.LSPServer, error)               { return nil, assert.AnError }
func (m *mockDB) DeleteLSPServer(id string) error                                 { return nil }
func (m *mockDB) UpsertConfigFragment(f *models.ConfigFragment) error             { return nil }
func (m *mockDB) ListConfigFragments(limit int) ([]models.ConfigFragment, error)   { return nil, nil }
func (m *mockDB) GetConfigFragment(id string) (*models.ConfigFragment, error)      { return nil, assert.AnError }
func (m *mockDB) UpsertModelProfile(p *models.ModelProfile) error                 { return nil }
func (m *mockDB) ListModelProfiles() ([]models.ModelProfile, error)               { return nil, nil }
func (m *mockDB) GetModelProfile(modelID string) (*models.ModelProfile, error)     { return nil, assert.AnError }
func (m *mockDB) UpsertSource(src *models.Source) error                           { return nil }
func (m *mockDB) GetSource(id string) (*models.Source, error)                     { return nil, assert.AnError }
func (m *mockDB) DeleteSource(id string) error                                    { return nil }
func (m *mockDB) ListSources() ([]models.Source, error)                           { return nil, nil }
func (m *mockDB) UpsertAgent(a *models.Agent) error                               { return nil }
func (m *mockDB) ListAgents() ([]models.Agent, error)                             { return nil, nil }
func (m *mockDB) InsertRoutingEvent(e *models.RoutingEvent) error                 { return nil }
func (m *mockDB) ListRoutingRules() ([]models.RoutingRule, error)                 { return nil, nil }
func (m *mockDB) UpsertRoutingRule(r *models.RoutingRule) error                   { return nil }
func (m *mockDB) GetBudget() (*models.BudgetConfig, error)                        { return nil, assert.AnError }
func (m *mockDB) SetPreference(key, value string) error                           { return nil }
func (m *mockDB) GetPreference(key string) (string, error)                        { return "", nil }
func (m *mockDB) ListPreferences() (map[string]string, error)                     { return nil, nil }
func (m *mockDB) DeletePreference(key string) error                               { return nil }
func (m *mockDB) GetSmallFastModels(ctx context.Context) ([]models.Model, error)  { return nil, nil }
func (m *mockDB) DBPath() string                                                  { return ":memory:" }
func (m *mockDB) SearchModels(query string) ([]models.Model, error) { return nil, nil }
func (m *mockDB) GetStats() (map[string]int, error) { panic("unexpected") }
func (m *mockDB) DeleteCommand(id string) error { return nil }
func (m *mockDB) DeleteMCP(id string) error { return nil }
func (m *mockDB) DeleteSkill(id string) error { return nil }
func (m *mockDB) UpdateSkillMeta(id string, updates map[string]any) error { return nil }
func (m *mockDB) SearchSkills(query string) ([]models.Skill, error) { return nil, nil }
func (m *mockDB) GetAgent(id string) (*models.Agent, error) { return nil, assert.AnError }
func (m *mockDB) DeleteAgent(id string) error { return nil }
func (m *mockDB) GetRoutingRule(key string) (*models.RoutingRule, error) { return nil, assert.AnError }
func (m *mockDB) DeleteRoutingRule(key string) error { return nil }
func (m *mockDB) ListRoutingEvents(limit int) ([]models.RoutingEvent, error) { return nil, nil }
func (m *mockDB) UpsertBudget(b *models.BudgetConfig) error { return nil }
func (m *mockDB) CleanupProviderPrefs() (int, error) { return 0, nil }
func (m *mockDB) CleanupInvalidPreferences() (int, error) { return 0, nil }
func (m *mockDB) InsertSyncLog(phase, status, details string, durationMs int64) error { return nil }
func (m *mockDB) ListSyncLogs(limit int) ([]models.SyncLog, error) { return nil, nil }
func (m *mockDB) InsertExecLog(l *models.ExecLog) error { return nil }
func (m *mockDB) ListExecLogs(limit int) ([]models.ExecLog, error) { return nil, nil }
func (m *mockDB) InsertSnapshot(hash, content string) error { return nil }
func (m *mockDB) ListSnapshots(limit int) ([]models.Snapshot, error) { return nil, nil }
func (m *mockDB) GetSnapshot(id int64) (*models.Snapshot, error) { return nil, assert.AnError }
func (m *mockDB) DeleteSnapshot(id int64) error { return nil }
func (m *mockDB) UpsertProject(p *models.Project) error { return nil }
func (m *mockDB) ListProjects() ([]models.Project, error) { return nil, nil }
func (m *mockDB) GetProject(id string) (*models.Project, error) { return nil, nil }
func (m *mockDB) DeleteProject(id string) error { return nil }
func (m *mockDB) UpsertDetectedStack(d *models.DetectedStack) error { return nil }
func (m *mockDB) ListDetectedStacks(projectID string) ([]models.DetectedStack, error) { return nil, nil }
func (m *mockDB) DeleteDetectedStacks(projectID string) error { return nil }
func (m *mockDB) UpsertProjectConfig(p *models.ProjectConfig) error { return nil }
func (m *mockDB) ListProjectConfigs(projectID string) ([]models.ProjectConfig, error) { return nil, nil }
func (m *mockDB) GetProjectConfig(projectID, configType string) (*models.ProjectConfig, error) { return nil, nil }
func (m *mockDB) DeleteProjectConfigs(projectID string) error { return nil }
func (m *mockDB) UpdateSourceItemStatus(id, status string) error { return nil }
func (m *mockDB) UpdateSourceItemTarget(id, targetPath string) error { return nil }
func (m *mockDB) ListSourceItemsBySource(sourceID string) ([]models.SourceItem, error) { return nil, nil }

// ── DetectAvailableProviders ──────────────────────────────────────

func TestDetectAvailableProviders(t *testing.T) {
	t.Run("returns providers with matching env key", func(t *testing.T) {
		t.Setenv("OK_TEST_KEY", "sk-test123")
		querier := &mockDB{
			providers: []models.Provider{
				{ID: "prov-a", Name: "Provider A", KeyEnv: "OK_TEST_KEY"},
				{ID: "prov-b", Name: "Provider B", KeyEnv: "OK_MISSING_KEY"},
			},
		}
		got := discover.DetectAvailableProviders(querier)
		assert.Equal(t, []string{"prov-a"}, got)
	})

	t.Run("returns empty when no keys set", func(t *testing.T) {
		querier := &mockDB{
			providers: []models.Provider{
				{ID: "prov-a", KeyEnv: "OK_NONEXISTENT"},
			},
		}
		got := discover.DetectAvailableProviders(querier)
		assert.Empty(t, got)
	})

	t.Run("returns empty on list error", func(t *testing.T) {
		querier := &mockDB{listProvErr: assert.AnError}
		got := discover.DetectAvailableProviders(querier)
		assert.Empty(t, got)
	})

	t.Run("all providers with keys are returned", func(t *testing.T) {
		t.Setenv("OK_KEY_A", "key-a")
		t.Setenv("OK_KEY_B", "key-b")
		querier := &mockDB{
			providers: []models.Provider{
				{ID: "a", Name: "A", KeyEnv: "OK_KEY_A"},
				{ID: "b", Name: "B", KeyEnv: "OK_KEY_B"},
				{ID: "c", Name: "C", KeyEnv: "OK_MISSING"},
			},
		}
		got := discover.DetectAvailableProviders(querier)
		assert.ElementsMatch(t, []string{"a", "b"}, got)
	})
}

// ── NewService ────────────────────────────────────────────────────

func TestNewService(t *testing.T) {
	m := &mockDB{}
	s := discover.NewService(discover.NewServiceParams{DB: m})
	require.NotNil(t, s)
}

// ── Service.ActivateUntestedFreeModels ────────────────────────────

func TestActivateUntestedFreeModels(t *testing.T) {
	t.Run("activates untested models from free providers", func(t *testing.T) {
		m := &mockDB{
			providers: []models.Provider{
				{ID: "free-prov", Name: "Free", IsFree: true},
				{ID: "paid-prov", Name: "Paid", IsFree: false},
			},
			modelsByProv: map[string][]models.Model{
				"free-prov": {
					{ID: "free-prov/model-a", DisplayName: "model-a", Status: "untested", ContextWindow: 0},
					{ID: "free-prov/model-b", DisplayName: "model-b", Status: "untested"},
				},
				"paid-prov": {
					{ID: "paid-prov/model-c", DisplayName: "model-c", Status: "untested"},
				},
			},
		}
		m.upsertedMod = nil
		s := discover.NewService(discover.NewServiceParams{DB: m})

		err := s.ActivateUntestedFreeModels()
		require.NoError(t, err)

		assert.Equal(t, 2, len(m.upsertedMod))
		for _, mod := range m.upsertedMod {
			assert.Equal(t, "active", mod.Status)
			assert.Equal(t, "free", mod.Tier)
			assert.Greater(t, mod.ContextWindow, 0, "context window should be > 0")
		}
	})

	t.Run("skips already active models", func(t *testing.T) {
		m := &mockDB{
			providers: []models.Provider{
				{ID: "free-prov", Name: "Free", IsFree: true},
			},
			modelsByProv: map[string][]models.Model{
				"free-prov": {
					{ID: "free-prov/active-model", DisplayName: "active-model", Status: "active"},
				},
			},
		}
		s := discover.NewService(discover.NewServiceParams{DB: m})
		err := s.ActivateUntestedFreeModels()
		require.NoError(t, err)
		assert.Equal(t, 0, len(m.upsertedMod))
	})

	t.Run("handles ListProviders error", func(t *testing.T) {
		m := &mockDB{listProvErr: assert.AnError}
		s := discover.NewService(discover.NewServiceParams{DB: m})
		err := s.ActivateUntestedFreeModels()
		require.Error(t, err)
	})
}

// ── Service.DeduplicateProviders ──────────────────────────────────

func TestDeduplicateProviders(t *testing.T) {
	t.Run("merges providers with same base URL", func(t *testing.T) {
		m := &mockDB{
			providers: []models.Provider{
				{ID: "winner", Name: "Winner", BaseURL: "https://api.example.com", KeyEnv: "OK_WINNER_KEY", Source: "seed"},
				{ID: "loser", Name: "Loser", BaseURL: "https://api.example.com", KeyEnv: "OK_MISSING_KEY", Source: "auto"},
			},
			modelsByProv: map[string][]models.Model{
				"loser": {
					{ID: "loser/model-a", DisplayName: "model-a", Status: "untested"},
				},
			},
		}
		t.Setenv("OK_WINNER_KEY", "sk-test")
		s := discover.NewService(discover.NewServiceParams{DB: m})

		err := s.DeduplicateProviders()
		require.NoError(t, err)
		assert.Equal(t, 1, len(m.upsertedMod))
		assert.Equal(t, "winner/model-a", m.upsertedMod[0].ID)
		assert.Equal(t, "winner", m.upsertedMod[0].ProviderID)
	})

	t.Run("prefers seed source when no keys set", func(t *testing.T) {
		m := &mockDB{
			providers: []models.Provider{
				{ID: "auto-prov", BaseURL: "https://api.example.com", Source: "auto"},
				{ID: "seed-prov", BaseURL: "https://api.example.com", Source: "seed"},
			},
			modelsByProv: map[string][]models.Model{},
		}
		s := discover.NewService(discover.NewServiceParams{DB: m})
		err := s.DeduplicateProviders()
		require.NoError(t, err)
	})

	t.Run("no-op when no duplicate URLs", func(t *testing.T) {
		m := &mockDB{
			providers: []models.Provider{
				{ID: "prov-a", BaseURL: "https://api.a.com"},
				{ID: "prov-b", BaseURL: "https://api.b.com"},
			},
		}
		s := discover.NewService(discover.NewServiceParams{DB: m})
		err := s.DeduplicateProviders()
		require.NoError(t, err)
		assert.Equal(t, 0, len(m.upsertedMod))
	})

	t.Run("no-op when no base URLs", func(t *testing.T) {
		m := &mockDB{
			providers: []models.Provider{{ID: "prov-a"}, {ID: "prov-b"}},
		}
		s := discover.NewService(discover.NewServiceParams{DB: m})
		err := s.DeduplicateProviders()
		require.NoError(t, err)
	})

	t.Run("handles ListProviders error", func(t *testing.T) {
		m := &mockDB{listProvErr: assert.AnError}
		s := discover.NewService(discover.NewServiceParams{DB: m})
		err := s.DeduplicateProviders()
		require.Error(t, err)
	})
}

// ── Service.Discover (HTTP integration via httptest.Server) ───────

func TestDiscover_FetchesAndProcessesModels(t *testing.T) {
	// Set up a test HTTP server that returns catalog models
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify auth header
		assert.Equal(t, "Bearer sk-test-api-key", r.Header.Get("Authorization"))

		resp := map[string][]discover.CatalogModel{
			"data": {
				{
					ID:          "gpt-4o",
					Object:      "model",
					Created:     1700000000,
					OwnedBy:     "openai",
					Description: "GPT-4o flagship model",
					Capabilities: mustRawMessage(`{"completion_chat": true, "function_calling": true, "vision": true}`),
					MaxContext:  128000,
					Pricing:     mustRawMessage(`{"prompt": 2.50, "completion": 10.00}`),
				},
				{
					ID:          "text-embedding-3-small",
					Object:      "model",
					Description: "Embedding model",
					Capabilities: mustRawMessage(`{"completion_chat": false}`),
				},
				{
					ID:          "gpt-3.5-turbo",
					Object:      "model",
					Capabilities: mustRawMessage(`{"completion_chat": true}`),
					MaxContext:  16384,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	t.Setenv("OK_DISCOVER_KEY", "sk-test-api-key")

	m := &mockDB{
		providers: []models.Provider{
			{
				ID:         "test-prov",
				Name:       "Test Provider",
				CatalogURL: server.URL,
				KeyEnv:     "OK_DISCOVER_KEY",
				Source:     "manual",
				Priority:   1,
			},
		},
	}
	s := discover.NewService(discover.NewServiceParams{DB: m})

	err := s.Discover(context.Background())
	require.NoError(t, err)

	// Should have upserted provider (from Discover's UpsertProvider call)
	assert.True(t, len(m.upsertedProv) > 0)

	// Should have upserted models: gpt-4o (chat) and gpt-3.5-turbo (chat),
	// but NOT text-embedding-3-small (not chat). Both chat models have
	// nonChatKeywords-free IDs.
	assert.Equal(t, 2, len(m.upsertedMod))

	ids := make(map[string]bool)
	for _, mod := range m.upsertedMod {
		ids[mod.ID] = true
	}
	assert.True(t, ids["test-prov/gpt-4o"], "gpt-4o should be upserted")
	assert.True(t, ids["test-prov/gpt-3.5-turbo"], "gpt-3.5-turbo should be upserted")
	assert.False(t, ids["test-prov/text-embedding-3-small"], "embedding model should be filtered out")
}

func TestDiscover_HandlesHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	t.Setenv("OK_DISCOVER_ERR", "key")

	m := &mockDB{
		providers: []models.Provider{
			{ID: "err-prov", Name: "Error", CatalogURL: server.URL, KeyEnv: "OK_DISCOVER_ERR"},
		},
	}
	s := discover.NewService(discover.NewServiceParams{DB: m})
	err := s.Discover(context.Background())
	// The HTTP error is logged and skipped, not propagated
	require.NoError(t, err)
}

func TestDiscover_SkipsProviderWithNoCatalogURL(t *testing.T) {
	m := &mockDB{
		providers: []models.Provider{
			{ID: "prov-no-catalog", Name: "No Catalog", CatalogURL: ""},
		},
	}
	s := discover.NewService(discover.NewServiceParams{DB: m})
	err := s.Discover(context.Background())
	require.NoError(t, err)
}

func TestDiscover_SkipsProviderWithNoKey(t *testing.T) {
	m := &mockDB{
		providers: []models.Provider{
			{ID: "prov-no-key", Name: "No Key", KeyEnv: "OK_NONEXISTENT_KEY", CatalogURL: "https://api.example.com/models"},
		},
	}
	s := discover.NewService(discover.NewServiceParams{DB: m})
	err := s.Discover(context.Background())
	require.NoError(t, err)
}

func TestDiscover_ReturnsErrorOnListFailure(t *testing.T) {
	m := &mockDB{listProvErr: assert.AnError}
	s := discover.NewService(discover.NewServiceParams{DB: m})
	err := s.Discover(context.Background())
	require.Error(t, err)
}

func TestDiscover_SkipsMultipleWithoutKeys(t *testing.T) {
	m := &mockDB{
		providers: []models.Provider{
			{ID: "p1", KeyEnv: "MISSING_1", CatalogURL: "https://a.com/models"},
			{ID: "p2", KeyEnv: "MISSING_2", CatalogURL: "https://b.com/models"},
			{ID: "p3", KeyEnv: "MISSING_3", CatalogURL: ""},
		},
	}
	s := discover.NewService(discover.NewServiceParams{DB: m})
	err := s.Discover(context.Background())
	require.NoError(t, err)
}

// ── Helpers ───────────────────────────────────────────────────────

func mustRawMessage(v string) json.RawMessage {
	return json.RawMessage(v)
}
