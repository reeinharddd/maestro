package db

import (
	"database/sql"
	"time"

	"github.com/reeinharrrd/opencode-kit/pkg/models"
)

type DBInterface interface {
	// General
	Close() error
	DBPath() string
	Query(query string, args ...interface{}) (*sql.Rows, error)
	Exec(query string, args ...interface{}) (sql.Result, error)
	Now() string
	ExecLog(phase, status, details string, dur time.Duration) error

	// Provider
	UpsertProvider(p *models.Provider) error
	ListProviders() ([]models.Provider, error)
	GetProvider(id string) (*models.Provider, error)
	DeleteProvider(id string) error

	// Model
	UpsertModel(m *models.Model) error
	ListModels(opts ...ModelFilter) ([]models.Model, error)
	ListModelsByProvider(providerID string) ([]models.Model, error)
	GetModel(id string) (*models.Model, error)
	DeleteModel(id string) error
	SearchModels(query string) ([]models.Model, error)
	GetStats() (map[string]int, error)

	// Agent
	UpsertAgent(a *models.Agent) error
	ListAgents() ([]models.Agent, error)
	GetAgent(id string) (*models.Agent, error)
	DeleteAgent(id string) error

	// Command
	UpsertCommand(c *models.Command) error
	ListCommands() ([]models.Command, error)

	// MCP
	UpsertMCP(m *models.MCPServer) error
	ListMCPs() ([]models.MCPServer, error)

	// Skill
	UpsertSkill(s *models.Skill) error
	ListSkills() ([]models.Skill, error)

	// Source
	UpsertSource(s *models.Source) error
	ListSources() ([]models.Source, error)

	// SourceItem
	UpsertSourceItem(s *models.SourceItem) error
	ListSourceItems() ([]models.SourceItem, error)
	GetSourceItem(id string) (*models.SourceItem, error)
	DeleteSourceItem(id string) error

	// LSPServer
	UpsertLSPServer(l *models.LSPServer) error
	ListLSPServers() ([]models.LSPServer, error)
	GetLSPServer(id string) (*models.LSPServer, error)
	DeleteLSPServer(id string) error

	// ModelProfile
	UpsertModelProfile(p *models.ModelProfile) error
	ListModelProfiles() ([]models.ModelProfile, error)
	GetModelProfile(modelID string) (*models.ModelProfile, error)

	// Budget
	UpsertBudget(b *models.BudgetConfig) error
	GetBudget() (*models.BudgetConfig, error)

	// Routing
	UpsertRoutingRule(r *models.RoutingRule) error
	ListRoutingRules() ([]models.RoutingRule, error)
	GetRoutingRule(key string) (*models.RoutingRule, error)

	// Sync Log
	InsertSyncLog(phase, status, details string, durationMs int64) error
	ListSyncLogs(limit int) ([]models.SyncLog, error)

	// Exec Log
	InsertExecLog(l *models.ExecLog) error
	ListExecLogs(limit int) ([]models.ExecLog, error)

	// Snapshot
	InsertSnapshot(hash, content string) error
	ListSnapshots(limit int) ([]models.Snapshot, error)
	GetSnapshot(id int64) (*models.Snapshot, error)
	DeleteSnapshot(id int64) error

	// Preference
	SetPreference(key, value string) error
	GetPreference(key string) (string, error)
	ListPreferences() (map[string]string, error)
	DeletePreference(key string) error
}
