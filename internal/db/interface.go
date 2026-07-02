package db

import (
	"context"
	"database/sql"
	"github.com/reeinharrrd/maestro/pkg/models"
)

// DBInterface defines the interface for database operations.
type DBInterface interface {
	// Provider operations
	UpsertProvider(p *models.Provider) error
	ListProviders() ([]models.Provider, error)
	GetProvider(id string) (*models.Provider, error)
	DeleteProvider(id string) error

	// Model operations
	UpsertModel(m *models.Model) error
	ListModels(opts ...ModelFilter) ([]models.Model, error)
	ListModelsByProvider(providerID string) ([]models.Model, error)
	GetModel(id string) (*models.Model, error)
	DeleteModel(id string) error

	// Model search & stats
	SearchModels(query string) ([]models.Model, error)
	GetStats() (map[string]int, error)

	// Command operations
	UpsertCommand(c *models.Command) error
	ListCommands() ([]models.Command, error)

	DeleteCommand(id string) error

	// MCP operations
	UpsertMCP(m *models.MCPServer) error
	ListMCPs() ([]models.MCPServer, error)

	DeleteMCP(id string) error

	// Skill operations
	UpsertSkill(s *models.Skill) error
	ListSkills() ([]models.Skill, error)

	UpdateSkillMeta(id string, updates map[string]any) error
	SearchSkills(query string) ([]models.Skill, error)
	DeleteSkill(id string) error

	// Source operations
	UpsertSourceItem(s *models.SourceItem) error
	ListSourceItems() ([]models.SourceItem, error)
	GetSourceItem(id string) (*models.SourceItem, error)
	DeleteSourceItem(id string) error
	// Source item status tracking
	ListSourceItemsBySource(sourceID string) ([]models.SourceItem, error)
	UpdateSourceItemStatus(id, status string) error
	UpdateSourceItemTarget(id, targetPath string) error

	// LSP operations
	UpsertLSPServer(l *models.LSPServer) error
	ListLSPServers() ([]models.LSPServer, error)
	GetLSPServer(id string) (*models.LSPServer, error)
	DeleteLSPServer(id string) error

	// Config fragment operations
	UpsertConfigFragment(f *models.ConfigFragment) error
	ListConfigFragments(limit int) ([]models.ConfigFragment, error)
	GetConfigFragment(id string) (*models.ConfigFragment, error)

	// Model profile operations
	UpsertModelProfile(p *models.ModelProfile) error
	ListModelProfiles() ([]models.ModelProfile, error)
	GetModelProfile(modelID string) (*models.ModelProfile, error)

	// Source registry operations
	UpsertSource(src *models.Source) error
	GetSource(id string) (*models.Source, error)
	ListSources() ([]models.Source, error)
	DeleteSource(id string) error
	UpsertAgent(a *models.Agent) error
	ListAgents() ([]models.Agent, error)

	GetAgent(id string) (*models.Agent, error)
	DeleteAgent(id string) error

	// Routing operations
	InsertRoutingEvent(e *models.RoutingEvent) error
	ListRoutingRules() ([]models.RoutingRule, error)
	UpsertRoutingRule(r *models.RoutingRule) error
	GetBudget() (*models.BudgetConfig, error)
	GetRoutingRule(key string) (*models.RoutingRule, error)
	DeleteRoutingRule(key string) error
	ListRoutingEvents(limit int) ([]models.RoutingEvent, error)
	UpsertBudget(b *models.BudgetConfig) error
	// Preference operations
	SetPreference(key, value string) error
	ListPreferences() (map[string]string, error)

	GetPreference(key string) (string, error)
	DeletePreference(key string) error
	CleanupProviderPrefs() (int, error)
	CleanupInvalidPreferences() (int, error)

	// Sync log operations
	InsertSyncLog(phase, status, details string, durationMs int64) error
	ListSyncLogs(limit int) ([]models.SyncLog, error)

	// Exec log operations
	InsertExecLog(l *models.ExecLog) error
	ListExecLogs(limit int) ([]models.ExecLog, error)

	// Snapshot operations
	InsertSnapshot(hash, content string) error
	ListSnapshots(limit int) ([]models.Snapshot, error)
	GetSnapshot(id int64) (*models.Snapshot, error)
	DeleteSnapshot(id int64) error

	// Raw query operations (for heal/migration)
	Query(query string, args ...any) (*sql.Rows, error)
	Exec(query string, args ...any) (sql.Result, error)

	// Small fast models for classification
	GetSmallFastModels(ctx context.Context) ([]models.Model, error)


	// Project operations
	UpsertProject(p *models.Project) error
	ListProjects() ([]models.Project, error)
	GetProject(id string) (*models.Project, error)
	DeleteProject(id string) error

	// DetectedStack operations
	UpsertDetectedStack(d *models.DetectedStack) error
	ListDetectedStacks(projectID string) ([]models.DetectedStack, error)
	DeleteDetectedStacks(projectID string) error

	// ProjectConfig operations
	UpsertProjectConfig(p *models.ProjectConfig) error
	ListProjectConfigs(projectID string) ([]models.ProjectConfig, error)
	GetProjectConfig(projectID, configType string) (*models.ProjectConfig, error)
	DeleteProjectConfigs(projectID string) error

	// DB path
	DBPath() string
}

// GetSmallFastModels retrieves small, fast models for classification.
func (d *DB) GetSmallFastModels(ctx context.Context) ([]models.Model, error) {
	rows, err := d.QueryContext(ctx, `
		SELECT id, display_name, provider_id, latency_p50_ms, pricing_prompt, pricing_completion, tier
		FROM models
		WHERE latency_p50_ms < 500 AND (pricing_prompt + pricing_completion) <= 0.01
		ORDER BY 
			CASE WHEN tier = 'free' THEN 0 ELSE 1 END,
			(pricing_prompt + pricing_completion) ASC,
			latency_p50_ms ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Model
	for rows.Next() {
		var m models.Model
		var latency float64
		var promptCost, completionCost float64
		var tier string
		if err := rows.Scan(&m.ID, &m.DisplayName, &m.ProviderID, &latency, &promptCost, &completionCost, &tier); err != nil {
			return nil, err
		}
		m.LatencyP50Ms = latency
		m.PricingPrompt = promptCost
		m.PricingCompletion = completionCost
		m.Tier = tier
		result = append(result, m)
	}
	return result, nil
}

