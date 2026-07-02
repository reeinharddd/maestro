package classifier

import (
	"fmt"
	"strings"

	"github.com/reeinharrrd/maestro/internal/db"
	"github.com/reeinharrrd/maestro/pkg/models"
)

// ClassResult holds the classification result for a model.
type ClassResult struct {
	Architecture   string
	Tier           string
	RecommendedUse string
}

// Service provides model classification by analyzing model metadata
// and applying heuristic rules based on known model families and capabilities.
type Service struct {
	db db.DBInterface
}

// NewService creates a new classifier Service.
func NewService(database db.DBInterface) *Service {
	return &Service{db: database}
}

// knownArchitectures maps recognized model family prefixes to their architecture type.
// This is the primary source for architecture identification.
var knownArchitectures = map[string]string{
	// OpenAI
	"gpt-4o":        "transformer",
	"gpt-4-turbo":   "transformer",
	"gpt-4":         "transformer",
	"gpt-3.5-turbo": "transformer",
	"gpt-3.5":       "transformer",
	"o1":            "reasoning_transformer",
	"o1-mini":       "reasoning_transformer",
	"o3":            "reasoning_transformer",
	"o3-mini":       "reasoning_transformer",

	// Anthropic
	"claude-3.5": "transformer",
	"claude-3":   "transformer",
	"claude-4":   "transformer",
	"claude":     "transformer",

	// Google
	"gemini-2.0":   "transformer",
	"gemini-1.5":   "transformer",
	"gemini-1.0":   "transformer",
	"gemini-pro":   "transformer",
	"gemini-ultra": "transformer",
	"gemini-flash": "transformer",
	"gemini":       "transformer",

	// Meta
	"llama-4":   "transformer",
	"llama-3.2": "transformer",
	"llama-3.1": "transformer",
	"llama-3":   "transformer",
	"llama-2":   "transformer",
	"llama":     "transformer",
	"codellama": "transformer",

	// Mistral
	"mistral-large":  "transformer",
	"mistral-medium": "transformer",
	"mistral-small":  "transformer",
	"mistral":        "transformer",
	"mixtral":        "mixture_of_experts",

	// DeepSeek
	"deepseek-r1": "reasoning_transformer",
	"deepseek-v3": "transformer",
	"deepseek-v2": "transformer",
	"deepseek":    "transformer",

	// Other
	"command-r-plus": "transformer",
	"command-r":      "transformer",
	"command":        "transformer",
	"dbrx":           "mixture_of_experts",
	"phi-4":          "transformer",
	"phi-3":          "transformer",
	"phi":            "transformer",
	"qwen2.5":        "transformer",
	"qwen2":          "transformer",
	"qwen":           "transformer",
}

// Classify analyzes a model and returns its Architecture, Tier, and RecommendedUse.
// It uses known model family mapping as the primary source, then falls back to
// heuristic rules based on capabilities and context window size.
func (s *Service) Classify(model *models.Model) (ClassResult, error) {
	if model == nil {
		return ClassResult{}, fmt.Errorf("model is nil")
	}

	return ClassResult{
		Architecture:   classifyArchitecture(model),
		Tier:           classifyTier(model),
		RecommendedUse: classifyRecommendedUse(model),
	}, nil
}

// classifyArchitecture determines the model architecture.
// Priority: known family mapping → capability heuristics → "unknown".
func classifyArchitecture(model *models.Model) string {
	// 1. Check known family mapping.
	family := strings.ToLower(strings.TrimSpace(model.Family))
	if family != "" {
		if arch, ok := knownArchitectures[family]; ok {
			return arch
		}
		// Check partial prefix match (e.g. "gpt-4-turbo-preview" → "gpt-4").
		for prefix, arch := range knownArchitectures {
			if strings.HasPrefix(family, prefix) {
				return arch
			}
		}
	}

	// 2. Check by model ID (which often contains the family name).
	id := strings.ToLower(strings.TrimSpace(model.ID))
	if id != "" {
		if arch, ok := knownArchitectures[id]; ok {
			return arch
		}
		for prefix, arch := range knownArchitectures {
			if strings.HasPrefix(id, prefix) {
				return arch
			}
		}
	}

	// 3. Check by display name.
	name := strings.ToLower(strings.TrimSpace(model.DisplayName))
	if name != "" {
		if arch, ok := knownArchitectures[name]; ok {
			return arch
		}
		for prefix, arch := range knownArchitectures {
			if strings.HasPrefix(name, prefix) || strings.Contains(name, prefix) {
				return arch
			}
		}
	}

	// 4. Capability-based heuristics.
	if model.Reasoning {
		return "reasoning"
	}
	if model.Vision && model.FunctionCalling {
		return "multimodal"
	}
	if model.ContextWindow > 100000 {
		return "large_context"
	}

	return "unknown"
}

// classifyTier determines the model pricing tier.
// Priority: model.Tier → context window heuristics → "free".
func classifyTier(model *models.Model) string {
	// 1. Use model.Tier directly if set and not empty.
	tier := strings.ToLower(strings.TrimSpace(model.Tier))
	if tier != "" && tier != "unknown" {
		return tier
	}

	// 2. Context-window based heuristic fallback.
	switch {
	case model.ContextWindow >= 128000:
		return "premium"
	case model.ContextWindow >= 32000:
		return "standard"
	case model.ContextWindow >= 8000:
		return "budget"
	default:
		return "free"
	}
}

// classifyRecommendedUse determines the best use case for the model
// based on its capabilities. Returns the most specific match.
func classifyRecommendedUse(model *models.Model) string {
	if model.FunctionCalling && model.StructuredOutput {
		return "agent"
	}
	if model.Reasoning {
		return "reasoning"
	}
	if model.Vision {
		return "vision"
	}
	if model.Audio {
		return "audio"
	}
	if model.ContextWindow > 100000 {
		return "long_context"
	}
	if model.FunctionCalling {
		return "tool_use"
	}
	return "general"
}
