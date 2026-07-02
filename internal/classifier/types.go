package classifier

import (
	"context"
	"time"
)

type Task struct {
	ID        string    `json:"id"`
	Input     string    `json:"input"`
	SessionID string    `json:"session_id"`
	CreatedAt time.Time `json:"created_at"`
}

type Model struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Provider   string `json:"provider"`
	Latency    int    `json:"latency"`
	Cost       int    `json:"cost"`
	IsFreeTier bool   `json:"is_free_tier"`
}

type Provider interface {
	ID() string
	Name() string
	Models(ctx context.Context) ([]Model, error)
	Classify(ctx context.Context, task Task, model Model) (ClassificationResult, error)
}

type ClassificationResult struct {
	TaskID     string            `json:"task_id"`
	ModelID    string            `json:"model_id"`
	Intent     string            `json:"intent"`
	Confidence float64           `json:"confidence"`
	Entities   map[string]string `json:"entities"`
	Latency    int               `json:"latency"`
}

type CacheKey struct {
	SessionID string
	Hash      string
}