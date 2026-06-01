package sync

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/reeinharrrd/opencode-kit/internal/db"
	"github.com/reeinharrrd/opencode-kit/internal/util"
	"github.com/reeinharrrd/opencode-kit/pkg/models"
)

type Service struct {
	db db.DBInterface
}

type Diff struct {
	AddedProviders   []string
	RemovedProviders []string
	AddedModels      []string
	RemovedModels    []string
	AddedAgents      []string
	RemovedAgents    []string
	AddedCommands    []string
	AddedMCPs        []string
}

func New(database db.DBInterface) *Service {
	return &Service{db: database}
}

func (s *Service) ImportFromOpenCodeConfig(configPath string) (*Diff, error) {
	diff := &Diff{}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cleaned := util.StripJSONC(data)
	var cfg map[string]interface{}
	if err := json.Unmarshal(cleaned, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	existing, _ := s.db.ListProviders()
	existingMap := make(map[string]bool)
	for _, p := range existing {
		existingMap[p.ID] = true
	}

	if provSection, ok := cfg["provider"].(map[string]interface{}); ok {
		for provID := range provSection {
			if !existingMap[provID] {
				_ = s.db.UpsertProvider(&models.Provider{
					ID:     provID,
					Name:   provID,
					Source: "opencode",
					Status: "active",
				})
				diff.AddedProviders = append(diff.AddedProviders, provID)
			}
		}
	}

	if agentSection, ok := cfg["agent"].(map[string]interface{}); ok {
		for agentID, agentVal := range agentSection {
			if agentMap, ok := agentVal.(map[string]interface{}); ok {
				model, _ := agentMap["model"].(string)
				desc, _ := agentMap["description"].(string)
				mode, _ := agentMap["mode"].(string)
				t, _ := agentMap["temperature"].(float64)
				color, _ := agentMap["color"].(string)

				_ = s.db.UpsertAgent(&models.Agent{
					ID:             agentID,
					Description:    desc,
					CurrentModelID: model,
					Mode:           mode,
					Temperature:    t,
					Color:          color,
					Source:         "opencode",
					Status:         "active",
				})
				diff.AddedAgents = append(diff.AddedAgents, agentID)
			}
		}
	}

	if cmdSection, ok := cfg["command"].(map[string]interface{}); ok {
		for cmdID, cmdVal := range cmdSection {
			if cmdMap, ok := cmdVal.(map[string]interface{}); ok {
				tpl, _ := cmdMap["template"].(string)
				desc, _ := cmdMap["description"].(string)

				_ = s.db.UpsertCommand(&models.Command{
					ID:          cmdID,
					Template:    tpl,
					Description: desc,
					Source:      "opencode",
					Status:      "active",
				})
				diff.AddedCommands = append(diff.AddedCommands, cmdID)
			}
		}
	}

	return diff, nil
}

func (s *Service) ExportToOpenCodeConfig(configPath string) error {
	providers, err := s.db.ListProviders()
	if err != nil {
		return err
	}

	cfg := map[string]interface{}{
		"$schema": "https://opencode.ai/config.json",
	}

	provSection := make(map[string]interface{})
	for _, p := range providers {
		models, _ := s.db.ListModelsByProvider(p.ID)
		var whitelist []string
		for _, m := range models {
			if m.Status != "error" && m.DisplayName != "" {
				whitelist = append(whitelist, m.DisplayName)
			}
		}
		if len(whitelist) > 0 {
			provSection[p.ID] = map[string]interface{}{
				"whitelist": whitelist,
			}
		}
	}
	cfg["provider"] = provSection

	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, out, 0644)
}
