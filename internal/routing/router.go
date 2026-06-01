package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/reeinharrrd/opencode-kit/internal/db"
	"github.com/reeinharrrd/opencode-kit/pkg/models"
)

type Service struct {
	db db.DBInterface
}

func New(database db.DBInterface) *Service {
	return &Service{db: database}
}

type taskDef struct {
	Description string
	MinContext  int
	NeedsFC     bool
	NeedsVision bool
	Priority    string
}

var taskDefs = map[string]taskDef{
	"coding_complex": {"Complex coding tasks with function calling", 100000, true, false, "quality"},
	"coding_fast":    {"Fast coding with function calling", 50000, true, false, "speed"},
	"reasoning":      {"Deep reasoning and analysis", 100000, false, false, "quality"},
	"vision":         {"Vision and image understanding", 100000, false, true, "quality"},
	"long_context":   {"Long context research and analysis", 500000, false, false, "cost"},
	"fastest":        {"Simple tasks, maximum speed", 0, false, false, "speed"},
}

func (s *Service) SelectBestModel(taskType string, budget models.BudgetConfig) (*models.RoutingRule, error) {
	def, ok := taskDefs[taskType]
	if !ok {
		return nil, fmt.Errorf("unknown task type: %s", taskType)
	}

	allModels, err := s.db.ListModels()
	if err != nil {
		return nil, fmt.Errorf("list models: %w", err)
	}

	var best models.Model
	bestScore := -1.0
	found := false

	for _, m := range allModels {
		if m.Status != "active" {
			continue
		}
		if budget.PreferredTier == "free_only" && m.Tier == "paid" {
			continue
		}
		if m.ContextWindow < def.MinContext {
			continue
		}
		if def.NeedsFC && !m.FunctionCalling {
			continue
		}
		if def.NeedsVision && !m.Vision {
			continue
		}

		score := scoreModel(m, def)
		if score > bestScore {
			bestScore = score
			best = m
			found = true
		}
	}

	if !found {
		return nil, fmt.Errorf("no suitable model found for task: %s", taskType)
	}

	fallbacks := make([]string, 0, 3)
	if best.ID != "" {
		fallbacks = append(fallbacks, best.ID)
	}
	if alt, ok := s.bestFallbackModels(def, best, allModels); ok {
		fallbacks = append(fallbacks, alt...)
	}
	fallbackJSON, _ := json.Marshal(fallbacks)

	return &models.RoutingRule{
		TaskKey:        taskType,
		Description:    def.Description,
		MinContext:     def.MinContext,
		NeedsFC:        def.NeedsFC,
		NeedsVision:    def.NeedsVision,
		CurrentModelID: best.ID,
		FallbackIDs:    string(fallbackJSON),
	}, nil
}

func (s *Service) bestFallbackModels(def taskDef, best models.Model, allModels []models.Model) ([]string, bool) {
	var out []string
	for _, m := range allModels {
		if m.ID == best.ID || m.Status != "active" {
			continue
		}
		if def.NeedsFC && !m.FunctionCalling {
			continue
		}
		if def.NeedsVision && !m.Vision {
			continue
		}
		if m.ContextWindow < def.MinContext/2 {
			continue
		}
		if best.Tier == "free" && m.Tier == "paid" {
			continue
		}
		out = append(out, m.ID)
		if len(out) == 2 {
			break
		}
	}
	return out, len(out) > 0
}

func scoreModel(m models.Model, def taskDef) float64 {
	score := 0.0

	ctxScore := math.Min(float64(m.ContextWindow)/100000, 5.0)
	score += ctxScore * 2

	if m.FunctionCalling {
		score += 3
	}
	if m.Vision {
		score += 2
	}

	latencyScore := 0.0
	if m.LatencyP50Ms > 0 {
		latencyScore = math.Max(0, 5-m.LatencyP50Ms/500)
	} else {
		latencyScore = 1
	}
	score += latencyScore

	isPaid := m.Tier == "paid"
	if isPaid {
		score -= 2
	}

	if m.ProviderID == "mistral" && (m.ContextWindow >= 200000) {
		score += 2
	}

	switch def.Priority {
	case "speed":
		score += latencyScore * 2
	case "quality":
		score += ctxScore * 1.5
		if m.FunctionCalling {
			score += 2
		}
	case "cost":
		if !isPaid {
			score += 3
		}
	}

	return score
}

func (s *Service) ReassignAll(ctx context.Context) error {
	budget, err := s.db.GetBudget()
	if err != nil {
		budget = &models.BudgetConfig{ID: "default", DailyGlobalUSD: 0.50, PreferredTier: "free_only"}
	}

	for taskType := range taskDefs {
		rule, err := s.SelectBestModel(taskType, *budget)
		if err != nil {
			fmt.Printf("  Warning: no model for %s: %v\n", taskType, err)
			continue
		}
		if err := s.db.UpsertRoutingRule(rule); err != nil {
			return fmt.Errorf("upsert rule %s: %w", taskType, err)
		}
		fmt.Printf("  Route: %s → %s\n", taskType, rule.CurrentModelID)
	}
	return nil
}
