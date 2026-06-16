// Package db provides database operations for okit.
//
// Copyright 2026 OpenCode Foundation
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"database/sql"
	"github.com/reeinharddd/okit/pkg/models"
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

	// Command operations
	UpsertCommand(c *models.Command) error
	ListCommands() ([]models.Command, error)

	// MCP operations
	UpsertMCP(m *models.MCPServer) error
	ListMCPs() ([]models.MCPServer, error)

	// Skill operations
	UpsertSkill(s *models.Skill) error
	ListSkills() ([]models.Skill, error)

	// Source operations
	UpsertSourceItem(s *models.SourceItem) error
	ListSourceItems() ([]models.SourceItem, error)
	GetSourceItem(id string) (*models.SourceItem, error)
	DeleteSourceItem(id string) error

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
	ListSources() ([]models.Source, error)

	// Agent operations
	UpsertAgent(a *models.Agent) error
	ListAgents() ([]models.Agent, error)

	// Routing operations
	InsertRoutingEvent(e *models.RoutingEvent) error
	ListRoutingRules() ([]models.RoutingRule, error)
	UpsertRoutingRule(r *models.RoutingRule) error
	GetBudget() (*models.BudgetConfig, error)

	// Preference operations
	SetPreference(key, value string) error
	ListPreferences() (map[string]string, error)

	// Raw query operations (for heal/migration)
	Query(query string, args ...any) (*sql.Rows, error)
	Exec(query string, args ...any) (sql.Result, error)

	// Small fast models for classification
	GetSmallFastModels(ctx context.Context) ([]models.Model, error)

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

