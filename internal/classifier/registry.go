// Package classifier provides the Task Classifier for okit.
// It implements a hybrid model selection system for classifying user tasks.
//
// Copyright 2026 OpenCode Foundation
// SPDX-License-Identifier: Apache-2.0

package classifier

import (
	"context"
	"sync"
)

// providerRegistry implements the ProviderRegistry interface.
type providerRegistry struct {
	providers map[string]Provider
	mu        sync.RWMutex
}

// NewProviderRegistry creates a new ProviderRegistry.
func NewProviderRegistry() ProviderRegistry {
	return &providerRegistry{
		providers: make(map[string]Provider),
	}
}

// Register registers a provider with the registry.
func (r *providerRegistry) Register(provider Provider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.providers[provider.ID()] = provider
	return nil
}

// Providers returns the list of registered providers.
func (r *providerRegistry) Providers() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]Provider, 0, len(r.providers))
	for _, provider := range r.providers {
		providers = append(providers, provider)
	}
	return providers
}

// Provider returns the provider with the given ID.
func (r *providerRegistry) Provider(id string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, ok := r.providers[id]
	return provider, ok
}

// DefaultProviders returns the default list of providers.
func DefaultProviders(ctx context.Context, db DBInterface) ([]Provider, error) {
	var providers []Provider

	// Add the EmbeddedProvider.
	embedded, err := NewEmbeddedProvider()
	if err != nil {
		return nil, fmt.Errorf("new embedded provider: %w", err)
	}
	providers = append(providers, embedded)

	// Add user models from the database.
	userModels, err := db.GetSmallFastModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("get user models: %w", err)
	}
	if len(userModels) > 0 {
		userProvider := NewUserModelsProvider(userModels)
		providers = append(providers, userProvider)
	}

	// TODO: Add OllamaProvider if Ollama is installed.

	return providers, nil
}

// UserModelsProvider implements the Provider interface for user models.
type UserModelsProvider struct {
	models []Model
}

// NewUserModelsProvider creates a new UserModelsProvider.
func NewUserModelsProvider(models []Model) *UserModelsProvider {
	return &UserModelsProvider{models: models}
}

// ID returns the unique identifier for the provider.
func (p *UserModelsProvider) ID() string {
	return "user_models"
}

// Name returns the human-readable name of the provider.
func (p *UserModelsProvider) Name() string {
	return "User Models"
}

// Models returns the list of models available from this provider.
func (p *UserModelsProvider) Models(ctx context.Context) ([]Model, error) {
	return p.models, nil
}

// Classify classifies a task using a user model.
func (p *UserModelsProvider) Classify(ctx context.Context, task Task, model Model) (ClassificationResult, error) {
	// TODO: Implement classification using user models (e.g., via MCP).
	return ClassificationResult{}, errors.New("not implemented")
}