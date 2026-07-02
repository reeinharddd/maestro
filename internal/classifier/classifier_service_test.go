package classifier_test

import (
	"testing"

	"github.com/reeinharrrd/maestro/internal/classifier"
	"github.com/reeinharrrd/maestro/internal/db"
	"github.com/reeinharrrd/maestro/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_Classify_NilModel(t *testing.T) {
	t.Parallel()

	d, err := db.Open(":memory:")
	require.NoError(t, err)
	defer d.Close()

	s := classifier.NewService(d)
	_, err = s.Classify(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "model is nil")
}

func TestService_Classify_Tier(t *testing.T) {
	t.Parallel()

	d, err := db.Open(":memory:")
	require.NoError(t, err)
	defer d.Close()

	s := classifier.NewService(d)

	tests := []struct {
		name     string
		model    *models.Model
		wantTier string
	}{
		{
			name:     "already classified premium",
			model:    &models.Model{ID: "gpt-4o", Tier: "premium"},
			wantTier: "premium",
		},
		{
			name:     "already classified budget",
			model:    &models.Model{ID: "gpt-4o-mini", Tier: "budget"},
			wantTier: "budget",
		},
		{
			name:     "tier unknown falls through to context",
			model:    &models.Model{ID: "test", Tier: "unknown", ContextWindow: 200000},
			wantTier: "premium",
		},
		{
			name:     "context >= 128k is premium",
			model:    &models.Model{ID: "test", ContextWindow: 128000},
			wantTier: "premium",
		},
		{
			name:     "context >= 32k is standard",
			model:    &models.Model{ID: "test", ContextWindow: 32000},
			wantTier: "standard",
		},
		{
			name:     "context >= 8k is budget",
			model:    &models.Model{ID: "test", ContextWindow: 8000},
			wantTier: "budget",
		},
		{
			name:     "context below 8k is free",
			model:    &models.Model{ID: "test", ContextWindow: 4096},
			wantTier: "free",
		},
		{
			name:     "zero context is free",
			model:    &models.Model{ID: "test", ContextWindow: 0},
			wantTier: "free",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res, err := s.Classify(tt.model)
			require.NoError(t, err)
			assert.Equal(t, tt.wantTier, res.Tier)
		})
	}
}

func TestService_Classify_Architecture(t *testing.T) {
	t.Parallel()

	d, err := db.Open(":memory:")
	require.NoError(t, err)
	defer d.Close()

	s := classifier.NewService(d)

	tests := []struct {
		name            string
		model           *models.Model
		wantArch        string
	}{
		{
			name:     "known family exact match",
			model:    &models.Model{ID: "gpt-4o-2024-08-06", Family: "gpt-4o"},
			wantArch: "transformer",
		},
		{
			name:     "reasoning transformer family",
			model:    &models.Model{ID: "o1-preview", Family: "o1"},
			wantArch: "reasoning_transformer",
		},
		{
			name:     "prefix match on family",
			model:    &models.Model{ID: "x", Family: "gpt-4-turbo-preview"},
			wantArch: "transformer",
		},
		{
			name:     "fallback to id when family unknown",
			model:    &models.Model{ID: "gemini-2.0-flash", Family: ""},
			wantArch: "transformer",
		},
		{
			name:     "fallback to name when family and id unknown",
			model:    &models.Model{ID: "custom-model", DisplayName: "claude-4-opus"},
			wantArch: "transformer",
		},
		{
			name:     "prefix match on name",
			model:    &models.Model{ID: "custom", DisplayName: "mistral-large-1234"},
			wantArch: "transformer",
		},
		{
			name:     "mixture of experts",
			model:    &models.Model{ID: "mixtral-8x7b", Family: "mixtral"},
			wantArch: "mixture_of_experts",
		},
		{
			name:     "reasoning fallback",
			model:    &models.Model{ID: "custom-reasoner", Reasoning: true},
			wantArch: "reasoning",
		},
		{
			name:     "multimodal fallback",
			model:    &models.Model{ID: "custom-vision", Vision: true, FunctionCalling: true},
			wantArch: "multimodal",
		},
		{
			name:     "large context fallback",
			model:    &models.Model{ID: "custom-long", ContextWindow: 200000},
			wantArch: "large_context",
		},
		{
			name:     "unknown architecture",
			model:    &models.Model{ID: "random-obscure-model-v7"},
			wantArch: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res, err := s.Classify(tt.model)
			require.NoError(t, err)
			assert.Equal(t, tt.wantArch, res.Architecture)
		})
	}
}

func TestService_Classify_RecommendedUse(t *testing.T) {
	t.Parallel()

	d, err := db.Open(":memory:")
	require.NoError(t, err)
	defer d.Close()

	s := classifier.NewService(d)

	tests := []struct {
		name  string
		model *models.Model
		want  string
	}{
		{
			name:  "agent use",
			model: &models.Model{ID: "gpt-4o", FunctionCalling: true, StructuredOutput: true},
			want:  "agent",
		},
		{
			name:  "reasoning use",
			model: &models.Model{ID: "o1", Reasoning: true},
			want:  "reasoning",
		},
		{
			name:  "vision use",
			model: &models.Model{ID: "gpt-4-vision", Vision: true},
			want:  "vision",
		},
		{
			name:  "audio use",
			model: &models.Model{ID: "gpt-4o-audio", Audio: true},
			want:  "audio",
		},
		{
			name:  "long context use",
			model: &models.Model{ID: "claude-3", ContextWindow: 200000},
			want:  "long_context",
		},
		{
			name:  "tool use",
			model: &models.Model{ID: "gpt-4", FunctionCalling: true},
			want:  "tool_use",
		},
		{
			name:  "general use",
			model: &models.Model{ID: "basic-model"},
			want:  "general",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res, err := s.Classify(tt.model)
			require.NoError(t, err)
			assert.Equal(t, tt.want, res.RecommendedUse)
		})
	}
}

func TestService_Classify_CompleteResult(t *testing.T) {
	t.Parallel()

	d, err := db.Open(":memory:")
	require.NoError(t, err)
	defer d.Close()

	s := classifier.NewService(d)

	model := &models.Model{
		ID:                "gpt-4o-2024-08-06",
		Family:            "gpt-4o",
		ContextWindow:     128000,
		FunctionCalling:   true,
		StructuredOutput:  true,
		Vision:            true,
	}

	res, err := s.Classify(model)
	require.NoError(t, err)
	assert.Equal(t, "transformer", res.Architecture)
	assert.Equal(t, "premium", res.Tier)
	assert.Equal(t, "agent", res.RecommendedUse)
}
