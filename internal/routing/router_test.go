package routing_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/reeinharrrd/maestro/internal/db"
	"github.com/reeinharrrd/maestro/internal/routing"
	"github.com/reeinharrrd/maestro/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDB implements db.DBInterface for routing tests.
type mockDB struct {
	models       []models.Model
	budget       *models.BudgetConfig
	events       []models.RoutingEvent
	lastRule     *models.RoutingRule
	returnErr    bool
	budgetErr    bool
	modelsErr    bool
	modelsFn     func() ([]models.Model, error)
	noopEventFn  func(*models.RoutingEvent) error
}

func (m *mockDB) ListModels(opts ...db.ModelFilter) ([]models.Model, error) {
	if m.modelsFn != nil {
		return m.modelsFn()
	}
	if m.modelsErr {
		return nil, assert.AnError
	}
	return m.models, nil
}

func (m *mockDB) GetBudget() (*models.BudgetConfig, error) {
	if m.budgetErr {
		return nil, assert.AnError
	}
	if m.budget != nil {
		return m.budget, nil
	}
	return &models.BudgetConfig{ID: "default", DailyGlobalUSD: 0.50, PreferredTier: "free_only"}, nil
}

func (m *mockDB) InsertRoutingEvent(e *models.RoutingEvent) error {
	if m.noopEventFn != nil {
		return m.noopEventFn(e)
	}
	m.events = append(m.events, *e)
	if m.returnErr {
		return assert.AnError
	}
	return nil
}

func (m *mockDB) UpsertRoutingRule(r *models.RoutingRule) error {
	m.lastRule = r
	return nil
}

func (m *mockDB) DeleteRoutingRule(key string) error {
	return nil
}

// Unimplemented methods — should not be called by routing code paths.
func (m *mockDB) UpsertProvider(p *models.Provider) error                      { panic("unexpected") }
func (m *mockDB) ListProviders() ([]models.Provider, error)                     { panic("unexpected") }
func (m *mockDB) GetProvider(id string) (*models.Provider, error)               { panic("unexpected") }
func (m *mockDB) DeleteProvider(id string) error                                { panic("unexpected") }
func (m *mockDB) UpsertModel(mdl *models.Model) error                           { panic("unexpected") }
func (m *mockDB) ListModelsByProvider(providerID string) ([]models.Model, error) { panic("unexpected") }
func (m *mockDB) GetModel(id string) (*models.Model, error)                     { panic("unexpected") }
func (m *mockDB) DeleteModel(id string) error                                   { panic("unexpected") }
func (m *mockDB) UpsertCommand(c *models.Command) error                         { panic("unexpected") }
func (m *mockDB) ListCommands() ([]models.Command, error)                       { panic("unexpected") }
func (m *mockDB) UpsertMCP(mcp *models.MCPServer) error                         { panic("unexpected") }
func (m *mockDB) ListMCPs() ([]models.MCPServer, error)                         { panic("unexpected") }
func (m *mockDB) UpsertSkill(s *models.Skill) error                             { panic("unexpected") }
func (m *mockDB) ListSkills() ([]models.Skill, error)                           { panic("unexpected") }
func (m *mockDB) UpsertSourceItem(s *models.SourceItem) error                   { panic("unexpected") }
func (m *mockDB) ListSourceItems() ([]models.SourceItem, error)                 { panic("unexpected") }
func (m *mockDB) GetSourceItem(id string) (*models.SourceItem, error)           { panic("unexpected") }
func (m *mockDB) DeleteSourceItem(id string) error                              { panic("unexpected") }
func (m *mockDB) UpsertLSPServer(l *models.LSPServer) error                     { panic("unexpected") }
func (m *mockDB) ListLSPServers() ([]models.LSPServer, error)                   { panic("unexpected") }
func (m *mockDB) GetLSPServer(id string) (*models.LSPServer, error)             { panic("unexpected") }
func (m *mockDB) DeleteLSPServer(id string) error                               { panic("unexpected") }
func (m *mockDB) UpsertConfigFragment(f *models.ConfigFragment) error           { panic("unexpected") }
func (m *mockDB) ListConfigFragments(limit int) ([]models.ConfigFragment, error) { panic("unexpected") }
func (m *mockDB) GetConfigFragment(id string) (*models.ConfigFragment, error)   { panic("unexpected") }
func (m *mockDB) UpsertModelProfile(p *models.ModelProfile) error               { panic("unexpected") }
func (m *mockDB) ListModelProfiles() ([]models.ModelProfile, error)             { panic("unexpected") }
func (m *mockDB) GetModelProfile(modelID string) (*models.ModelProfile, error)  { panic("unexpected") }
func (m *mockDB) UpsertSource(src *models.Source) error                         { panic("unexpected") }
func (m *mockDB) GetSource(id string) (*models.Source, error)                   { panic("unexpected") }
func (m *mockDB) DeleteSource(id string) error                                  { panic("unexpected") }
func (m *mockDB) ListSources() ([]models.Source, error)                         { panic("unexpected") }
func (m *mockDB) UpsertAgent(a *models.Agent) error                             { panic("unexpected") }
func (m *mockDB) ListAgents() ([]models.Agent, error)                           { panic("unexpected") }
func (m *mockDB) ListRoutingRules() ([]models.RoutingRule, error)               { return nil, nil }
func (m *mockDB) SetPreference(key, value string) error                         { panic("unexpected") }
func (m *mockDB) ListPreferences() (map[string]string, error)                   { panic("unexpected") }
func (m *mockDB) Query(query string, args ...any) (*sql.Rows, error)            { panic("unexpected") }
func (m *mockDB) Exec(query string, args ...any) (sql.Result, error)            { panic("unexpected") }
func (m *mockDB) GetSmallFastModels(ctx context.Context) ([]models.Model, error) { panic("unexpected") }
func (m *mockDB) DBPath() string                                                { panic("unexpected") }
func (m *mockDB) SearchModels(query string) ([]models.Model, error) { panic("unexpected") }
func (m *mockDB) GetStats() (map[string]int, error) { panic("unexpected") }
func (m *mockDB) DeleteCommand(id string) error { panic("unexpected") }
func (m *mockDB) DeleteMCP(id string) error { panic("unexpected") }
func (m *mockDB) DeleteSkill(id string) error { panic("unexpected") }
func (m *mockDB) UpdateSkillMeta(id string, updates map[string]any) error { panic("unexpected") }
func (m *mockDB) SearchSkills(query string) ([]models.Skill, error) { panic("unexpected") }
func (m *mockDB) GetAgent(id string) (*models.Agent, error) { panic("unexpected") }
func (m *mockDB) DeleteAgent(id string) error { panic("unexpected") }
func (m *mockDB) GetRoutingRule(key string) (*models.RoutingRule, error) { panic("unexpected") }
func (m *mockDB) ListRoutingEvents(limit int) ([]models.RoutingEvent, error) { panic("unexpected") }
func (m *mockDB) UpsertBudget(b *models.BudgetConfig) error { panic("unexpected") }
func (m *mockDB) GetPreference(key string) (string, error) { panic("unexpected") }
func (m *mockDB) DeletePreference(key string) error { panic("unexpected") }
func (m *mockDB) CleanupProviderPrefs() (int, error) { panic("unexpected") }
func (m *mockDB) CleanupInvalidPreferences() (int, error) { panic("unexpected") }
func (m *mockDB) InsertSyncLog(phase, status, details string, durationMs int64) error { panic("unexpected") }
func (m *mockDB) ListSyncLogs(limit int) ([]models.SyncLog, error) { panic("unexpected") }
func (m *mockDB) InsertExecLog(l *models.ExecLog) error { panic("unexpected") }
func (m *mockDB) ListExecLogs(limit int) ([]models.ExecLog, error) { panic("unexpected") }
func (m *mockDB) InsertSnapshot(hash, content string) error { panic("unexpected") }
func (m *mockDB) ListSnapshots(limit int) ([]models.Snapshot, error) { panic("unexpected") }
func (m *mockDB) GetSnapshot(id int64) (*models.Snapshot, error) { panic("unexpected") }
func (m *mockDB) DeleteSnapshot(id int64) error { panic("unexpected") }
func (m *mockDB) UpsertProject(p *models.Project) error { panic("unexpected") }
func (m *mockDB) ListProjects() ([]models.Project, error) { panic("unexpected") }
func (m *mockDB) GetProject(id string) (*models.Project, error) { panic("unexpected") }
func (m *mockDB) DeleteProject(id string) error { panic("unexpected") }
func (m *mockDB) UpsertDetectedStack(d *models.DetectedStack) error { panic("unexpected") }
func (m *mockDB) ListDetectedStacks(projectID string) ([]models.DetectedStack, error) { panic("unexpected") }
func (m *mockDB) DeleteDetectedStacks(projectID string) error { panic("unexpected") }
func (m *mockDB) UpsertProjectConfig(p *models.ProjectConfig) error { panic("unexpected") }
func (m *mockDB) ListProjectConfigs(projectID string) ([]models.ProjectConfig, error) { panic("unexpected") }
func (m *mockDB) GetProjectConfig(projectID, configType string) (*models.ProjectConfig, error) { panic("unexpected") }
func (m *mockDB) DeleteProjectConfigs(projectID string) error { panic("unexpected") }
func (m *mockDB) UpdateSourceItemStatus(id, status string) error { panic("unexpected") }
func (m *mockDB) UpdateSourceItemTarget(id, targetPath string) error { panic("unexpected") }
func (m *mockDB) ListSourceItemsBySource(sourceID string) ([]models.SourceItem, error) { return nil, nil }

func TestFormatCandidateSummary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			want:  "-",
		},
		{
			name:  "empty array",
			input: "[]",
			want:  "-",
		},
		{
			name:  "single candidate",
			input: `[{"id":"gpt-4","score":3.5}]`,
			want:  "gpt-4=3.50",
		},
		{
			name:  "multiple candidates",
			input: `[{"id":"gpt-4","score":3.5},{"id":"claude-3","score":2.0}]`,
			want:  "gpt-4=3.50, claude-3=2.00",
		},
		{
			name:  "invalid json returns raw",
			input: `{invalid json}`,
			want:  "{invalid json}",
		},
		{
			name:  "high precision score",
			input: `[{"id":"model-a","score":1.234567}]`,
			want:  "model-a=1.23",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := routing.FormatCandidateSummary(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSelectBestModel(t *testing.T) {
	t.Parallel()

	baseBudget := models.BudgetConfig{DailyGlobalUSD: 0.50, PreferredTier: "free_only"}

	t.Run("unknown task type returns error", func(t *testing.T) {
		t.Parallel()
		m := &mockDB{models: []models.Model{}}
		s := routing.New(m)
		_, err := s.SelectBestModel("nonexistent_task", baseBudget, false)
		assert.ErrorContains(t, err, "unknown task type")
	})

	t.Run("no eligible models returns error", func(t *testing.T) {
		t.Parallel()
		m := &mockDB{
			models: []models.Model{
				{ID: "paid-only", Status: "active", Tier: "paid", ContextWindow: 1000},
			},
			budget: &baseBudget,
		}
		s := routing.New(m)
		_, err := s.SelectBestModel("fastest", models.BudgetConfig{PreferredTier: "free_only"}, false)
		assert.ErrorContains(t, err, "no suitable model found")
	})

	t.Run("selects highest scored model", func(t *testing.T) {
		t.Parallel()
		models := []models.Model{
			{ID: "slow-free", Status: "active", Tier: "free", ContextWindow: 8000, LatencyP50Ms: 2000, PricingPrompt: 0, PricingCompletion: 0},
			{ID: "fast-free", Status: "active", Tier: "free", ContextWindow: 8000, LatencyP50Ms: 100, PricingPrompt: 0, PricingCompletion: 0},
		}
		m := &mockDB{models: models, budget: &baseBudget}
		s := routing.New(m)
		rule, err := s.SelectBestModel("fastest", baseBudget, false)
		require.NoError(t, err)
		assert.Equal(t, "fast-free", rule.CurrentModelID)
	})

	t.Run("prefers models with function calling for coding task", func(t *testing.T) {
		t.Parallel()
		models := []models.Model{
			{
				ID: "no-fc", Status: "active", Tier: "free", ContextWindow: 100000,
				FunctionCalling: false, LatencyP50Ms: 200, PricingPrompt: 0, PricingCompletion: 0,
			},
		}
		m := &mockDB{models: models, budget: &baseBudget}
		s := routing.New(m)
		_, err := s.SelectBestModel("coding_complex", baseBudget, false)
		// no-fc is not eligible because coding_complex needs FC
		assert.ErrorContains(t, err, "no suitable model found")
	})

	t.Run("shadow mode does not upsert rule", func(t *testing.T) {
		t.Parallel()
		models := []models.Model{
			{ID: "model-a", Status: "active", Tier: "free", ContextWindow: 8000, LatencyP50Ms: 100, PricingPrompt: 0, PricingCompletion: 0},
		}
		m := &mockDB{models: models, budget: &baseBudget}
		s := routing.New(m)
		rule, err := s.SelectBestModel("fastest", baseBudget, true)
		require.NoError(t, err)
		assert.NotNil(t, rule)
		assert.Nil(t, m.lastRule) // shadow mode does not upsert
	})

	t.Run("logs routing event", func(t *testing.T) {
		t.Parallel()
		models := []models.Model{
			{ID: "model-a", Status: "active", Tier: "free", ContextWindow: 8000, LatencyP50Ms: 100, PricingPrompt: 0, PricingCompletion: 0},
		}
		m := &mockDB{models: models, budget: &baseBudget}
		s := routing.New(m)
		_, err := s.SelectBestModel("fastest", baseBudget, false)
		require.NoError(t, err)
		require.Len(t, m.events, 1)
		assert.Equal(t, "fastest", m.events[0].TaskKey)
		assert.Equal(t, "model-a", m.events[0].SelectedModel)
		assert.False(t, m.events[0].Shadow)
	})

	t.Run("returned rule has correct fields", func(t *testing.T) {
		t.Parallel()
		models := []models.Model{
			{ID: "smart-model", Status: "active", Tier: "free", ContextWindow: 200000,
				FunctionCalling: true, Reasoning: true, LatencyP50Ms: 300,
				PricingPrompt: 0.01, PricingCompletion: 0.02},
		}
		m := &mockDB{models: models, budget: &baseBudget}
		s := routing.New(m)
		rule, err := s.SelectBestModel("reasoning", baseBudget, false)
		require.NoError(t, err)
		assert.Equal(t, "reasoning", rule.TaskKey)
		assert.Equal(t, "smart-model", rule.CurrentModelID)
		assert.False(t, rule.NeedsFC) // reasoning task does not require FC
		assert.False(t, rule.NeedsVision)
		assert.Equal(t, 100000, rule.MinContext)
		// FallbackIDs should be valid JSON array
		var fallbacks []string
		assert.NoError(t, json.Unmarshal([]byte(rule.FallbackIDs), &fallbacks))
		assert.Contains(t, fallbacks, "smart-model")
	})

	t.Run("model exceeds max cost gets penalty", func(t *testing.T) {
		t.Parallel()
		// coding_fast has MaxCostPerCall=0.03
		expensive := models.Model{
			ID: "pricey", Status: "active", Tier: "free", ContextWindow: 100000,
			FunctionCalling: true, LatencyP50Ms: 100,
			PricingPrompt: 0.10, PricingCompletion: 0.10, // total 0.20 > 0.03
		}
		cheap := models.Model{
			ID: "affordable", Status: "active", Tier: "free", ContextWindow: 50000,
			FunctionCalling: true, LatencyP50Ms: 200,
			PricingPrompt: 0.005, PricingCompletion: 0.005, // total 0.01 < 0.03
		}
		m := &mockDB{models: []models.Model{expensive, cheap}, budget: &baseBudget}
		s := routing.New(m)
		rule, err := s.SelectBestModel("coding_fast", baseBudget, false)
		require.NoError(t, err)
		assert.Equal(t, "affordable", rule.CurrentModelID)
	})

	t.Run("models with circuit breaker are excluded", func(t *testing.T) {
		t.Parallel()
		broken := models.Model{
			ID: "broken-model", Status: "active", Tier: "free", ContextWindow: 8000,
			FailCount: 5, LastTested: 0, // FailCount >= 3 and no last test → circuit open
			LatencyP50Ms: 100, PricingPrompt: 0, PricingCompletion: 0,
		}
		healthy := models.Model{
			ID: "health-model", Status: "active", Tier: "free", ContextWindow: 8000,
			FailCount: 0, LatencyP50Ms: 150, PricingPrompt: 0, PricingCompletion: 0,
		}
		m := &mockDB{models: []models.Model{broken, healthy}, budget: &baseBudget}
		s := routing.New(m)
		rule, err := s.SelectBestModel("fastest", baseBudget, false)
		require.NoError(t, err)
		assert.Equal(t, "health-model", rule.CurrentModelID)
	})
}

func TestReassignAll(t *testing.T) {
	t.Parallel()

	baseBudget := models.BudgetConfig{DailyGlobalUSD: 0.50, PreferredTier: "free_only"}

	t.Run("reassigns all task types", func(t *testing.T) {
		t.Parallel()
		mdls := []models.Model{
			{ID: "all-purpose", Status: "active", Tier: "free", ContextWindow: 500000,
				FunctionCalling: true, Vision: true, Reasoning: true,
				LatencyP50Ms: 100, PricingPrompt: 0, PricingCompletion: 0},
		}
		m := &mockDB{models: mdls, budget: &baseBudget}
		s := routing.New(m)
		err := s.ReassignAll(context.Background(), false)
		require.NoError(t, err)
		assert.NotNil(t, m.lastRule)
	})

	t.Run("shadow mode does not upsert", func(t *testing.T) {
		t.Parallel()
		mdls := []models.Model{
			{ID: "m", Status: "active", Tier: "free", ContextWindow: 500000,
				FunctionCalling: true, Vision: true, Reasoning: true,
				LatencyP50Ms: 100, PricingPrompt: 0, PricingCompletion: 0},
		}
		m := &mockDB{models: mdls, budget: &baseBudget}
		s := routing.New(m)
		err := s.ReassignAll(context.Background(), true)
		require.NoError(t, err)
		assert.Nil(t, m.lastRule) // shadow → no upsert
	})

	t.Run("no models available returns error on first task", func(t *testing.T) {
		t.Parallel()
		m := &mockDB{models: []models.Model{}, budget: &baseBudget}
		s := routing.New(m)
		// ReassignAll prints warnings but doesn't return error unless UPSERT fails
		err := s.ReassignAll(context.Background(), false)
		require.NoError(t, err)
	})
}
