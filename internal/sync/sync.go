package sync

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/reeinharddd/okit/internal/db"
	"github.com/reeinharddd/okit/internal/util"
	"github.com/reeinharddd/okit/pkg/models"
)

const metaPref = "config/"

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
		for provID, provVal := range provSection {
			if provCfg, ok := provVal.(map[string]interface{}); ok {
				if !existingMap[provID] {
					_ = s.db.UpsertProvider(&models.Provider{
						ID:     provID,
						Name:   provID,
						Source: "opencode",
						Status: "active",
					})
					diff.AddedProviders = append(diff.AddedProviders, provID)
				}
				importModels := func(names []string) {
					for _, name := range names {
						modelID := provID + "/" + name
						if err := s.db.UpsertModel(&models.Model{
							ID:          modelID,
							ProviderID:  provID,
							DisplayName: name,
							Source:      "opencode",
							Status:      "untested",
						}); err == nil {
							diff.AddedModels = append(diff.AddedModels, modelID)
						}
						_ = s.db.UpsertModelProfile(&models.ModelProfile{
							ModelID: modelID,
						})
					}
				}
				if whitelist, ok := provCfg["whitelist"].([]interface{}); ok {
					var names []string
					for _, w := range whitelist {
						if name, ok := w.(string); ok {
							names = append(names, name)
						}
					}
					importModels(names)
				}
				if modelsSection, ok := provCfg["models"].(map[string]interface{}); ok {
					var names []string
					for modelName := range modelsSection {
						names = append(names, modelName)
					}
					importModels(names)
				}
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

	if mcpSection, ok := cfg["mcp"].(map[string]interface{}); ok {
		for id, val := range mcpSection {
			entry, _ := val.(map[string]interface{})
			m := models.MCPServer{ID: id, Source: "opencode"}
			if t, _ := entry["type"].(string); t != "" {
				m.Type = t
			}
			if cmd, _ := entry["command"].([]interface{}); len(cmd) > 0 {
				b, _ := json.Marshal(cmd)
				m.Command = string(b)
			}
			if u, _ := entry["url"].(string); u != "" {
				m.URL = u
			}
			if en, _ := entry["enabled"].(bool); en {
				m.Enabled = true
			}
			if env, _ := entry["environment"].(map[string]interface{}); len(env) > 0 {
				b, _ := json.Marshal(env)
				m.EnvVars = string(b)
			}
			if to, _ := entry["timeout"].(float64); to > 0 {
				m.Timeout = int(to)
			}
			_ = s.db.UpsertMCP(&m)
			diff.AddedMCPs = append(diff.AddedMCPs, id)
		}
	}

	if lspBool, isBool := cfg["lsp"].(bool); isBool {
		_ = s.db.SetPreference(metaPref+"lsp", fmt.Sprintf("%t", lspBool))
	} else if lspObj, isObj := cfg["lsp"].(map[string]interface{}); isObj {
		_ = s.db.SetPreference(metaPref+"lsp", "object")
		for id, val := range lspObj {
			entry, _ := val.(map[string]interface{})
			l := models.LSPServer{ID: id}
			if cmd, _ := entry["command"].([]interface{}); len(cmd) > 0 {
				b, _ := json.Marshal(cmd)
				l.Command = string(b)
			}
			if ext, _ := entry["extensions"].([]interface{}); len(ext) > 0 {
				b, _ := json.Marshal(ext)
				l.Extensions = string(b)
			}
			if env, _ := entry["env"].(map[string]interface{}); len(env) > 0 {
				b, _ := json.Marshal(env)
				l.Env = string(b)
			}
			if init, _ := entry["initialization"].(string); init != "" {
				l.Initialization = init
			}
			if dis, _ := entry["disabled"].(bool); dis {
				l.Disabled = true
			}
			_ = s.db.UpsertLSPServer(&l)
		}
	}

	setJSONPref := func(key string, val interface{}) {
		b, _ := json.Marshal(val)
		_ = s.db.SetPreference(key, string(b))
	}
	for _, key := range []string{"autoupdate", "disabled_providers", "model", "small_model", "share", "plugin"} {
		if v, exists := cfg[key]; exists {
			setJSONPref(metaPref+key, v)
		}
	}
	if skills, ok := cfg["skills"].(map[string]interface{}); ok {
		for sk, sv := range skills {
			setJSONPref(metaPref+"skills_"+sk, sv)
		}
	}
	if comp, ok := cfg["compaction"].(map[string]interface{}); ok {
		for ck, cv := range comp {
			setJSONPref(metaPref+"compaction_"+ck, cv)
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
		providerModels, _ := s.db.ListModelsByProvider(p.ID)
		var whitelist []string
		for _, m := range providerModels {
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
