// Package classifier provides the Task Classifier for okit.
// It implements a hybrid model selection system for classifying user tasks.
//
// Copyright 2026 OpenCode Foundation
// SPDX-License-Identifier: Apache-2.0

package classifier

import (
	"context"
	"time"
)

// Task represents a user task to be classified.
type Task struct {
	// ID is a unique identifier for the task.
	ID string `json:"id"`
	// Input is the raw user input to classify.
	Input string `json:"input"`
	// SessionID is the ID of the current session.
	SessionID string `json:"session_id"`
	// CreatedAt is the timestamp when the task was created.
	CreatedAt time.Time `json:"created_at"`
}

// Model represents a model that can classify tasks.
type Model struct {
	// ID is a unique identifier for the model.
	ID string `json:"id"`
	// Name is the human-readable name of the model.
	Name string `json:"name"`
	// Provider is the provider of the model (e.g., "openai", "ollama", "embedded").
	Provider string `json:"provider"`
	// Latency is the expected latency of the model in milliseconds.
	Latency int `json:"latency"`
	// Cost is the cost per classification in micro-cents (e.g., 1000 = $0.01).
	Cost int `json:"cost"`
	// IsFreeTier indicates if the model is available in the free tier.
	IsFreeTier bool `json:"is_free_tier"`
}

// Provider represents a provider of models.
type Provider interface {
	// ID returns the unique identifier for the provider.
	ID() string
	// Name returns the human-readable name of the provider.
	Name() string
	// Models returns the list of models available from this provider.
	Models(ctx context.Context) ([]Model, error)
	// Classify classifies a task using the provider's models.
	Classify(ctx context.Context, task Task, model Model) (ClassificationResult, error)
}

// ClassificationResult represents the result of classifying a task.
type ClassificationResult struct {
	// TaskID is the ID of the classified task.
	TaskID string `json:"task_id"`
	// ModelID is the ID of the model used for classification.
	ModelID string `json:"model_id"`
	// Intent is the classified intent of the task (e.g., "code_review", "debug").
	Intent string `json:"intent"`
	// Confidence is the confidence score of the classification (0.0 to 1.0).
	Confidence float64 `json:"confidence"`
	// Entities is a map of extracted entities (e.g., "skill": "golang-pro").
	Entities map[string]string `json:"entities"`
	// Latency is the latency of the classification in milliseconds.
	Latency int `json:"latency"`
}

// CacheKey represents a key for caching classification results.
type CacheKey struct {
	// SessionID is the ID of the current session.
	SessionID string
	// Hash is the hash of the last 3 turns and the current message.
	Hash string
}