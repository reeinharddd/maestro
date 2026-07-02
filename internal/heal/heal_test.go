package heal_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/reeinharrrd/maestro/internal/db"
	"github.com/reeinharrrd/maestro/internal/heal"
	"github.com/reeinharrrd/maestro/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

// mockHealDB uses a real in-memory SQLite for Query/Exec and canned data for ListProviders/ListModelsByProvider.
type mockHealDB struct {
	db         *sql.DB
	providers  []models.Provider
	modelsByPID map[string][]models.Model
}

func newMockHealDB(t *testing.T, seedModels bool) *mockHealDB {
	t.Helper()
	sqldb, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { sqldb.Close() })

	m := &mockHealDB{db: sqldb, modelsByPID: make(map[string][]models.Model)}

	// Create minimal models table matching the schema heal queries expect.
	_, err = m.db.Exec(`
		CREATE TABLE models (
			id TEXT PRIMARY KEY,
			provider_id TEXT,
			display_name TEXT,
			status TEXT,
			error_message TEXT,
			last_tested INTEGER,
			fail_count INTEGER,
			context_window INTEGER,
			function_calling INTEGER,
			vision INTEGER,
			streaming INTEGER,
			latency_p50_ms REAL,
			pricing_prompt REAL,
			pricing_completion REAL,
			tier TEXT,
			source TEXT
		)
	`)
	require.NoError(t, err)

	if seedModels {
		now := time.Now().Unix()
		weekAgo := now - 8*24*3600 // >7 days ago

		// errored model with high fail count
		_, err = m.db.Exec(`INSERT INTO models (id, status, fail_count) VALUES ('broken-model', 'error', 5)`)
		require.NoError(t, err)
		// active stale model (last_tested >7 days ago)
		_, err = m.db.Exec(`INSERT INTO models (id, status, last_tested) VALUES ('stale-model', 'active', ?)`, weekAgo)
		require.NoError(t, err)
		// active fresh model
		_, err = m.db.Exec(`INSERT INTO models (id, status, last_tested) VALUES ('fresh-model', 'active', ?)`, now)
		require.NoError(t, err)
	}

	return m
}

func (m *mockHealDB) Query(query string, args ...any) (*sql.Rows, error) {
	return m.db.Query(query, args...)
}

func (m *mockHealDB) Exec(query string, args ...any) (sql.Result, error) {
	return m.db.Exec(query, args...)
}

func (m *mockHealDB) ListProviders() ([]models.Provider, error) {
	return m.providers, nil
}

func (m *mockHealDB) ListModelsByProvider(providerID string) ([]models.Model, error) {
	return m.modelsByPID[providerID], nil
}

// Methods not called by heal code paths.
func (m *mockHealDB) UpsertProvider(p *models.Provider) error                      { panic("unexpected") }
func (m *mockHealDB) ListRoutingRules() ([]models.RoutingRule, error)              { panic("unexpected") }
func (m *mockHealDB) GetProvider(id string) (*models.Provider, error)              { panic("unexpected") }
func (m *mockHealDB) DeleteProvider(id string) error                               { panic("unexpected") }
func (m *mockHealDB) UpsertModel(mdl *models.Model) error                          { panic("unexpected") }
func (m *mockHealDB) ListModels(opts ...db.ModelFilter) ([]models.Model, error)    { panic("unexpected") }
func (m *mockHealDB) GetModel(id string) (*models.Model, error)                    { panic("unexpected") }
func (m *mockHealDB) DeleteModel(id string) error                                  { panic("unexpected") }
func (m *mockHealDB) UpsertCommand(c *models.Command) error                        { panic("unexpected") }
func (m *mockHealDB) ListCommands() ([]models.Command, error)                      { panic("unexpected") }
func (m *mockHealDB) UpsertMCP(mcp *models.MCPServer) error                        { panic("unexpected") }
func (m *mockHealDB) ListMCPs() ([]models.MCPServer, error)                        { panic("unexpected") }
func (m *mockHealDB) UpsertSkill(s *models.Skill) error                            { panic("unexpected") }
func (m *mockHealDB) ListSkills() ([]models.Skill, error)                          { panic("unexpected") }
func (m *mockHealDB) UpsertSourceItem(s *models.SourceItem) error                  { panic("unexpected") }
func (m *mockHealDB) ListSourceItems() ([]models.SourceItem, error)                { panic("unexpected") }
func (m *mockHealDB) GetSourceItem(id string) (*models.SourceItem, error)          { panic("unexpected") }
func (m *mockHealDB) DeleteSourceItem(id string) error                             { panic("unexpected") }
func (m *mockHealDB) UpsertLSPServer(l *models.LSPServer) error                    { panic("unexpected") }
func (m *mockHealDB) ListLSPServers() ([]models.LSPServer, error)                  { panic("unexpected") }
func (m *mockHealDB) GetLSPServer(id string) (*models.LSPServer, error)            { panic("unexpected") }
func (m *mockHealDB) DeleteLSPServer(id string) error                              { panic("unexpected") }
func (m *mockHealDB) UpsertConfigFragment(f *models.ConfigFragment) error          { panic("unexpected") }
func (m *mockHealDB) ListConfigFragments(limit int) ([]models.ConfigFragment, error) { panic("unexpected") }
func (m *mockHealDB) GetConfigFragment(id string) (*models.ConfigFragment, error)  { panic("unexpected") }
func (m *mockHealDB) UpsertModelProfile(p *models.ModelProfile) error              { panic("unexpected") }
func (m *mockHealDB) ListModelProfiles() ([]models.ModelProfile, error)            { panic("unexpected") }
func (m *mockHealDB) GetModelProfile(modelID string) (*models.ModelProfile, error) { panic("unexpected") }
func (m *mockHealDB) UpsertSource(src *models.Source) error                        { panic("unexpected") }
func (m *mockHealDB) GetSource(id string) (*models.Source, error)                  { panic("unexpected") }
func (m *mockHealDB) DeleteSource(id string) error                                 { panic("unexpected") }
func (m *mockHealDB) ListSources() ([]models.Source, error)                        { panic("unexpected") }
func (m *mockHealDB) UpsertAgent(a *models.Agent) error                            { panic("unexpected") }
func (m *mockHealDB) ListAgents() ([]models.Agent, error)                          { panic("unexpected") }
func (m *mockHealDB) InsertRoutingEvent(e *models.RoutingEvent) error              { panic("unexpected") }
func (m *mockHealDB) UpsertRoutingRule(r *models.RoutingRule) error                { panic("unexpected") }
func (m *mockHealDB) GetBudget() (*models.BudgetConfig, error)                     { panic("unexpected") }
func (m *mockHealDB) SetPreference(key, value string) error                        { panic("unexpected") }
func (m *mockHealDB) ListPreferences() (map[string]string, error)                  { panic("unexpected") }
func (m *mockHealDB) GetSmallFastModels(ctx context.Context) ([]models.Model, error) { panic("unexpected") }
func (m *mockHealDB) DBPath() string                                               { panic("unexpected") }
func (m *mockHealDB) SearchModels(query string) ([]models.Model, error) { panic("unexpected") }
func (m *mockHealDB) GetStats() (map[string]int, error) { panic("unexpected") }
func (m *mockHealDB) DeleteCommand(id string) error { panic("unexpected") }
func (m *mockHealDB) DeleteMCP(id string) error { panic("unexpected") }
func (m *mockHealDB) DeleteSkill(id string) error { panic("unexpected") }
func (m *mockHealDB) UpdateSkillMeta(id string, updates map[string]any) error { panic("unexpected") }
func (m *mockHealDB) SearchSkills(query string) ([]models.Skill, error) { panic("unexpected") }
func (m *mockHealDB) GetAgent(id string) (*models.Agent, error) { panic("unexpected") }
func (m *mockHealDB) DeleteAgent(id string) error { panic("unexpected") }
func (m *mockHealDB) GetRoutingRule(key string) (*models.RoutingRule, error) { panic("unexpected") }
func (m *mockHealDB) DeleteRoutingRule(key string) error { panic("unexpected") }
func (m *mockHealDB) ListRoutingEvents(limit int) ([]models.RoutingEvent, error) { panic("unexpected") }
func (m *mockHealDB) UpsertBudget(b *models.BudgetConfig) error { panic("unexpected") }
func (m *mockHealDB) GetPreference(key string) (string, error) { panic("unexpected") }
func (m *mockHealDB) DeletePreference(key string) error { panic("unexpected") }
func (m *mockHealDB) CleanupProviderPrefs() (int, error) { panic("unexpected") }
func (m *mockHealDB) CleanupInvalidPreferences() (int, error) { panic("unexpected") }
func (m *mockHealDB) InsertSyncLog(phase, status, details string, durationMs int64) error { panic("unexpected") }
func (m *mockHealDB) ListSyncLogs(limit int) ([]models.SyncLog, error) { panic("unexpected") }
func (m *mockHealDB) InsertExecLog(l *models.ExecLog) error { panic("unexpected") }
func (m *mockHealDB) ListExecLogs(limit int) ([]models.ExecLog, error) { panic("unexpected") }
func (m *mockHealDB) InsertSnapshot(hash, content string) error { panic("unexpected") }
func (m *mockHealDB) ListSnapshots(limit int) ([]models.Snapshot, error) { panic("unexpected") }
func (m *mockHealDB) GetSnapshot(id int64) (*models.Snapshot, error) { panic("unexpected") }
func (m *mockHealDB) DeleteSnapshot(id int64) error { panic("unexpected") }
func (m *mockHealDB) UpsertProject(p *models.Project) error { panic("unexpected") }
func (m *mockHealDB) ListProjects() ([]models.Project, error) { panic("unexpected") }
func (m *mockHealDB) GetProject(id string) (*models.Project, error) { panic("unexpected") }
func (m *mockHealDB) DeleteProject(id string) error { panic("unexpected") }
func (m *mockHealDB) UpsertDetectedStack(d *models.DetectedStack) error { panic("unexpected") }
func (m *mockHealDB) ListDetectedStacks(projectID string) ([]models.DetectedStack, error) { panic("unexpected") }
func (m *mockHealDB) DeleteDetectedStacks(projectID string) error { panic("unexpected") }
func (m *mockHealDB) UpsertProjectConfig(p *models.ProjectConfig) error { panic("unexpected") }
func (m *mockHealDB) ListProjectConfigs(projectID string) ([]models.ProjectConfig, error) { panic("unexpected") }
func (m *mockHealDB) GetProjectConfig(projectID, configType string) (*models.ProjectConfig, error) { panic("unexpected") }
func (m *mockHealDB) DeleteProjectConfigs(projectID string) error { panic("unexpected") }
func (m *mockHealDB) UpdateSourceItemStatus(id, status string) error { panic("unexpected") }
func (m *mockHealDB) UpdateSourceItemTarget(id, targetPath string) error { panic("unexpected") }
func (m *mockHealDB) ListSourceItemsBySource(sourceID string) ([]models.SourceItem, error) { return nil, nil }

func TestRun(t *testing.T) {
	t.Parallel()

	t.Run("reports issues and fixes them", func(t *testing.T) {
		t.Parallel()
		m := newMockHealDB(t, true)
		svc := heal.New(m)

		report, err := svc.Run(context.Background())
		require.NoError(t, err)
		require.NotNil(t, report)

		// Should find: broken-model (error + high fail count) and stale-model (>7 days old)
		assert.GreaterOrEqual(t, report.IssuesFound, 2)

		// broken-model should be deprecated (fixed=true → status changed to deprecated)
		var brokenStatus string
		err = m.db.QueryRow(`SELECT status FROM models WHERE id='broken-model'`).Scan(&brokenStatus)
		require.NoError(t, err)
		assert.Equal(t, "deprecated", brokenStatus)

		// stale-model should be marked as needs re-test (fixed=true from checkStaleModels)
		var staleStatus string
		err = m.db.QueryRow(`SELECT status FROM models WHERE id='stale-model'`).Scan(&staleStatus)
		require.NoError(t, err)
		assert.Equal(t, "active", staleStatus) // stale check doesn't change status, just reports
	})

	t.Run("integrity check passes on clean db", func(t *testing.T) {
		t.Parallel()
		m := newMockHealDB(t, false) // no seed data
		svc := heal.New(m)

		report, err := svc.Run(context.Background())
		require.NoError(t, err)
		assert.NotNil(t, report)
		// Clean DB should have no critical issues from integrity check
		for _, iss := range report.Issues {
			if iss.Component == "config" {
				assert.Fail(t, "unexpected integrity issue: "+iss.Message)
			}
		}
	})

	t.Run("provider with no active models reported", func(t *testing.T) {
		t.Parallel()
		m := newMockHealDB(t, false)

		// Add a provider with no active models
		m.providers = []models.Provider{
			{ID: "orphan-provider", Name: "Orphan Provider", Status: "active"},
		}
		m.modelsByPID["orphan-provider"] = []models.Model{
			{ID: "dep-model", Status: "deprecated"},
		}

		svc := heal.New(m)
		report, err := svc.Run(context.Background())
		require.NoError(t, err)

		found := false
		for _, iss := range report.Issues {
			if iss.Component == "providers" {
				found = true
				assert.Contains(t, iss.Message, "orphan-provider")
				assert.Equal(t, "warning", iss.Severity)
				assert.False(t, iss.Fixed)
				break
			}
		}
		assert.True(t, found, "expected provider issue in report")
	})

	t.Run("provider with active models not reported", func(t *testing.T) {
		t.Parallel()
		m := newMockHealDB(t, false)

		m.providers = []models.Provider{
			{ID: "healthy-p", Name: "Healthy Provider", Status: "active"},
		}
		m.modelsByPID["healthy-p"] = []models.Model{
			{ID: "active-m", Status: "active"},
		}

		svc := heal.New(m)
		report, err := svc.Run(context.Background())
		require.NoError(t, err)

		for _, iss := range report.Issues {
			if iss.Component == "providers" {
				assert.Fail(t, "unexpected provider issue: "+iss.Message)
			}
		}
	})
}
