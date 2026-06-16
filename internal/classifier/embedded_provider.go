// Package classifier provides the Task Classifier for okit.
// It implements a hybrid model selection system for classifying user tasks.
//
// Copyright 2026 OpenCode Foundation
// SPDX-License-Identifier: Apache-2.0

package classifier

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"math"
	"sync"

	"github.com/reeinharddd/okit/internal/classifier/tokenizer"
	"github.com/reeinharddd/okit/pkg/onnx"
)

//go:embed embedded/*
var embeddedFS embed.FS

// EmbeddedProvider implements the Provider interface for embedded models.
type EmbeddedProvider struct {
	model     *onnx.Model
	tokenizer *tokenizer.Tokenizer
	mu        sync.Mutex
}

// NewEmbeddedProvider creates a new EmbeddedProvider.
func NewEmbeddedProvider() (*EmbeddedProvider, error) {
	// Load the ONNX model.
	modelData, err := embeddedFS.ReadFile("embedded/model.onnx")
	if err != nil {
		return nil, fmt.Errorf("read model.onnx: %w", err)
	}

	model, err := onnx.LoadModel(modelData)
	if err != nil {
		return nil, fmt.Errorf("load model: %w", err)
	}

	// Load the tokenizer.
	tokenizerData, err := embeddedFS.ReadFile("embedded/tokenizer.json")
	if err != nil {
		return nil, fmt.Errorf("read tokenizer.json: %w", err)
	}

	tok, err := tokenizer.NewTokenizer(tokenizerData)
	if err != nil {
		return nil, fmt.Errorf("load tokenizer: %w", err)
	}

	return &EmbeddedProvider{
		model:     model,
		tokenizer: tok,
	}, nil
}

// ID returns the unique identifier for the provider.
func (p *EmbeddedProvider) ID() string {
	return "embedded"
}

// Name returns the human-readable name of the provider.
func (p *EmbeddedProvider) Name() string {
	return "Embedded Provider"
}

// Models returns the list of models available from this provider.
func (p *EmbeddedProvider) Models(ctx context.Context) ([]Model, error) {
	return []Model{
		{
			ID:          "embedded-distilbert",
			Name:        "DistilBERT (Embedded)",
			Provider:     p.ID(),
			Latency:     100, // Estimated latency in milliseconds
			Cost:        0,   // No cost for embedded models
			IsFreeTier:  true,
		},
	}, nil
}

// Classify classifies a task using the embedded model.
func (p *EmbeddedProvider) Classify(ctx context.Context, task Task, model Model) (ClassificationResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if model.Provider != p.ID() {
		return ClassificationResult{}, errors.New("model not supported by this provider")
	}

	// Tokenize the input.
	tokens, err := p.tokenizer.Encode(task.Input)
	if err != nil {
		return ClassificationResult{}, fmt.Errorf("tokenize input: %w", err)
	}

	// Run inference.
	inputs := map[string][]float32{"input_ids": tokens}
	outputs, err := p.model.Run(inputs)
	if err != nil {
		return ClassificationResult{}, fmt.Errorf("inference: %w", err)
	}

	// Parse model output: apply softmax to logits, select top class.
	logits, ok := outputs["output"]
	if !ok || len(logits) == 0 {
		return ClassificationResult{}, errors.New("model output missing or empty")
	}

	confidence, intent := parseOutput(logits)

	return ClassificationResult{
		TaskID:     task.ID,
		ModelID:    model.ID,
		Intent:     intent,
		Confidence: confidence,
		Entities:   make(map[string]string),
		Latency:    100,
	}, nil
}

// intentLabels maps class indices to task classification intents.
// Order must match the embedded model's output layer.
var intentLabels = []string{
	"coding_complex",
	"coding_fast",
	"reasoning",
	"vision",
	"long_context",
	"fastest",
}

func softmax(logits []float32) (int, float64) {
	var maxLogit float32
	for _, v := range logits {
		if v > maxLogit {
			maxLogit = v
		}
	}
	var sum float64
	exps := make([]float64, len(logits))
	for i, v := range logits {
		exps[i] = math.Exp(float64(v - maxLogit))
		sum += exps[i]
	}
	maxIdx, maxProb := 0, 0.0
	for i, p := range exps {
		prob := p / sum
		if prob > maxProb {
			maxIdx, maxProb = i, prob
		}
	}
	return maxIdx, maxProb
}

func parseOutput(logits []float32) (float64, string) {
	idx, confidence := softmax(logits)
	if idx < len(intentLabels) {
		return confidence, intentLabels[idx]
	}
	return confidence, "unknown"
}