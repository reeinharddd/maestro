// Package db provides database operations for okit.
//
// Copyright 2026 OpenCode Foundation
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"github.com/reeinharddd/okit/pkg/models"
)

// DBInterface defines the interface for database operations.
type DBInterface interface {
	// UpsertCommand upserts a command into the database.
	UpsertCommand(c *models.Command) error
	// ListCommands lists all commands in the database.
	ListCommands() ([]models.Command, error)
	// UpsertMCP upserts an MCP server into the database.
	UpsertMCP(m *models.MCPServer) error
	// ListMCPs lists all MCP servers in the database.
	ListMCPs() ([]models.MCPServer, error)
	// UpsertSkill upserts a skill into the database.
	UpsertSkill(s *models.Skill) error
	// ListSkills lists all skills in the database.
	ListSkills() ([]models.Skill, error)
	// UpsertSourceItem upserts a source item into the database.
	UpsertSourceItem(s *models.SourceItem) error
	// ListSourceItems lists all source items in the database.
	ListSourceItems() ([]models.SourceItem, error)
	// GetSourceItem retrieves a source item by ID.
	GetSourceItem(id string) (*models.SourceItem, error)
	// DeleteSourceItem deletes a source item by ID.
	DeleteSourceItem(id string) error
	// UpsertLSPServer upserts an LSP server into the database.
	UpsertLSPServer(l *models.LSPServer) error
	// ListLSPServers lists all LSP servers in the database.
	ListLSPServers() ([]models.LSPServer, error)
	// GetLSPServer retrieves an LSP server by ID.
	GetLSPServer(id string) (*models.LSPServer, error)
	// DeleteLSPServer deletes an LSP server by ID.
	DeleteLSPServer(id string) error
	// UpsertConfigFragment upserts a config fragment into the database.
	UpsertConfigFragment(f *models.ConfigFragment) error
	// ListConfigFragments lists all config fragments in the database.
	ListConfigFragments(limit int) ([]models.ConfigFragment, error)
	// GetConfigFragment retrieves a config fragment by ID.
	GetConfigFragment(id string) (*models.ConfigFragment, error
	// UpsertModelProfile upserts a model profile into the database.
	UpsertModelProfile(p *models.ModelProfile) error
	// ListModelProfiles lists all model profiles in the database.
	ListModelProfiles() ([]models.ModelProfile, error)
	// GetModelProfile retrieves a model profile by model ID.
	GetModelProfile(modelID string) (*models.ModelProfile, error)
	// UpsertSource upserts a source into the database.
	UpsertSource(src *models.Source) error
	// ListSources lists all sources in the database.
	ListSources() ([]models.Source, error)
	// GetSmallFastModels retrieves small, fast models for classification.
	GetSmallFastModels(ctx context.Context) ([]models.Model, error)
}

// GetSmallFastModels retrieves small, fast models for classification.
func (d *DB) GetSmallFastModels(ctx context.Context) ([]models.Model, error) {
	rows, err := d.QueryContext(ctx, `
		SELECT id, name, provider, latency_ms, cost_microcents, is_free_tier
		FROM models
		WHERE latency_ms < 500 AND cost_microcents <= 1000
		ORDER BY is_free_tier DESC, cost_microcents ASC, latency_ms ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models []models.Model
n	for rows.Next() {
		var m models.Model
		var latency int
		var cost int
		var isFree int
		if err := rows.Scan(&m.ID, &m.Name, &m.Provider, &latency, &cost, &isFree); err != nil {
			return nil, err
		}
		m.Latency = latency
		m.Cost = cost
		m.IsFreeTier = isFree != 0
		models = append(models, m)
	}
	return models, nil
}

// MockDB is a mock implementation of DBInterface for testing.
type MockDB struct {
	GetSmallFastModelsFunc func(ctx context.Context) ([]models.Model, error)
}

func (m *MockDB) GetSmallFastModels(ctx context.Context) ([]models.Model, error) {
	if m.GetSmallFastModelsFunc != nil {
		return m.GetSmallFastModelsFunc(ctx)
	}
	return nil, nil
}

func (m *MockDB) UpsertCommand(c *models.Command) error {
	return nil
}

func (m *MockDB) ListCommands() ([]models.Command, error) {
	return nil, nil
}

func (m *MockDB) UpsertMCP(mcp *models.MCPServer) error {
	return nil
}

func (m *MockDB) ListMCPs() ([]models.MCPServer, error) {
	return nil, nil
}

func (m *MockDB) UpsertSkill(s *models.Skill) error {
	return nil
}

func (m *MockDB) ListSkills() ([]models.Skill, error) {
	return nil, nil
}

func (m *MockDB) UpsertSourceItem(s *models.SourceItem) error {
	return nil
}

func (m *MockDB) ListSourceItems() ([]models.SourceItem, error) {
	return nil, nil
}

func (m *MockDB) GetSourceItem(id string) (*models.SourceItem, error) {
	return nil, nil
}

func (m *MockDB) DeleteSourceItem(id string) error {
	return nil
}

func (m *MockDB) UpsertLSPServer(l *models.LSPServer) error {
	return nil
}

func (m *MockDB) ListLSPServers() ([]models.LSPServer, error) {
	return nil, nil
}

func (m *MockDB) GetLSPServer(id string) (*models.LSPServer, error) {
	return nil, nil
}

func (m *MockDB) DeleteLSPServer(id string) error {
	return nil
}

func (m *MockDB) UpsertConfigFragment(f *models.ConfigFragment) error {
	return nil
}

func (m *MockDB) ListConfigFragments(limit int) ([]models.ConfigFragment, error) {
	return nil, nil
}

func (m *MockDB) GetConfigFragment(id string) (*models.ConfigFragment, error) {
	return nil, nil
}

func (m *MockDB) UpsertModelProfile(p *models.ModelProfile) error {
	return nil
}

func (m *MockDB) ListModelProfiles() ([]models.ModelProfile, error) {
	return nil, nil
}

func (m *MockDB) GetModelProfile(modelID string) (*models.ModelProfile, error) {
	return nil, nil
}

func (m *MockDB) UpsertSource(src *models.Source) error {
	return nil
}

func (m *MockDB) ListSources() ([]models.Source, error) {
	return nil, nil
}