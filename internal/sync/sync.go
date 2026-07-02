package sync

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/reeinharrrd/maestro/internal/db"
	"github.com/reeinharrrd/maestro/pkg/models"
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

	cfg, err := ParseOpenCodeConfig(configPath)
	if err != nil {
		return nil, err
	}

	existing, _ := s.db.ListProviders()
	existingMap := make(map[string]bool)
	for _, p := range existing {
		existingMap[p.ID] = true
	}

	// ── Providers ──
	for provID, provCfg := range cfg.Providers {
		provName := provCfg.Name
		if provName == "" {
			provName = provID
		}

		provBaseURL := ""
		if len(provCfg.Options) > 0 {
			if bu, _ := provCfg.Options["baseURL"].(string); bu != "" {
				provBaseURL = bu
			}
		}

		upserted := !existingMap[provID]
		p := &models.Provider{
			ID:      provID,
			Name:    provName,
			BaseURL: provBaseURL,
			Source:  "opencode",
			Status:  "active",
		}
		// Preserve existing KeyEnv — don't wipe what discover set
		if existing, err := s.db.GetProvider(provID); err == nil && existing.KeyEnv != "" {
			p.KeyEnv = existing.KeyEnv
			p.CatalogURL = existing.CatalogURL
		}
		_ = s.db.UpsertProvider(p)
		if upserted {
			diff.AddedProviders = append(diff.AddedProviders, provID)
		}

		// Store npm and options as preferences
		if provCfg.NPM != "" && provCfg.NPM != "null" && provCfg.NPM != "NULL" {
			_ = s.db.SetPreference("config/provider_npm_"+provID, provCfg.NPM)
		}
		if len(provCfg.Options) > 0 {
			b, _ := json.Marshal(provCfg.Options)
			_ = s.db.SetPreference("config/provider_options_"+provID, string(b))
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
		if len(provCfg.Whitelist) > 0 {
			importModels(provCfg.Whitelist)
		}
		if len(provCfg.Models) > 0 {
			var names []string
			for modelName := range provCfg.Models {
				names = append(names, modelName)
			}
			importModels(names)
		}
	}

	// ── Agents ──
	for agentID, agentCfg := range cfg.Agents {
		id := normalizeAgentID(agentID)
		_ = s.db.UpsertAgent(&models.Agent{
			ID:             id,
			Description:    agentCfg.Description,
			CurrentModelID: agentCfg.Model,
			Mode:           agentCfg.Mode,
			Temperature:    agentCfg.Temperature,
			Color:          agentCfg.Color,
			Source:         "opencode",
			Status:         "active",
		})
		diff.AddedAgents = append(diff.AddedAgents, id)
	}

	// ── Commands ──
	for cmdID, cmdCfg := range cfg.Commands {
		_ = s.db.UpsertCommand(&models.Command{
			ID:          cmdID,
			Template:    cmdCfg.Template,
			Description: cmdCfg.Description,
			Source:      "opencode",
			Status:      "active",
		})
		diff.AddedCommands = append(diff.AddedCommands, cmdID)
	}

	// ── MCP ──
	for id, mcpCfg := range cfg.MCPServers {
		m := models.MCPServer{ID: id, Source: "opencode"}
		if mcpCfg.Type != "" {
			m.Type = mcpCfg.Type
		}
		if len(mcpCfg.Command) > 0 {
			b, _ := json.Marshal(mcpCfg.Command)
			m.Command = string(b)
		}
		if mcpCfg.URL != "" {
			m.URL = mcpCfg.URL
		}
		if mcpCfg.Enabled {
			m.Enabled = true
		}
		env := mcpCfg.Environment
		if len(env) == 0 {
			env = mcpCfg.Env
		}
		if len(env) > 0 {
			b, _ := json.Marshal(env)
			m.EnvVars = string(b)
		}
		if mcpCfg.Timeout > 0 {
			m.Timeout = int(mcpCfg.Timeout)
		}
		_ = s.db.UpsertMCP(&m)
		diff.AddedMCPs = append(diff.AddedMCPs, id)
	}

	// ── LSP ──
	if lspBool, ok := cfg.LSPEnabled(); ok {
		_ = s.db.SetPreference(metaPref+"lsp", fmt.Sprintf("%t", lspBool))
	}
	for id, lspCfg := range cfg.LSPServers() {
		l := models.LSPServer{ID: id}
		if len(lspCfg.Command) > 0 {
			b, _ := json.Marshal(lspCfg.Command)
			l.Command = string(b)
		}
		if len(lspCfg.Extensions) > 0 {
			b, _ := json.Marshal(lspCfg.Extensions)
			l.Extensions = string(b)
		}
		if len(lspCfg.Env) > 0 {
			b, _ := json.Marshal(lspCfg.Env)
			l.Env = string(b)
		}
		if lspCfg.Initialization != "" {
			l.Initialization = lspCfg.Initialization
		}
		if lspCfg.Disabled {
			l.Disabled = true
		}
		_ = s.db.UpsertLSPServer(&l)
	}

	// ── Meta preferences ──
	setJSONPref := func(key string, val interface{}) {
		b, _ := json.Marshal(val)
		_ = s.db.SetPreference(key, string(b))
	}
	for _, key := range []string{"autoupdate", "disabled_providers", "model", "small_model", "share", "plugin"} {
		if v, exists := cfg.Raw[key]; exists {
			setJSONPref(metaPref+key, v)
		}
	}
	for sk, sv := range cfg.Skills {
		setJSONPref(metaPref+"skills_"+sk, sv)
	}
	for ck, cv := range cfg.Compaction {
		setJSONPref(metaPref+"compaction_"+ck, cv)
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

	prefs, _ := s.db.ListPreferences()

	// ── Providers ─────────────────────────────────────────────────
	provSection := make(map[string]interface{})
	for _, p := range providers {
		providerModels, _ := s.db.ListModelsByProvider(p.ID)
		whitelist := make([]string, 0, len(providerModels))
		modelEntries := make(map[string]interface{})
		for _, m := range providerModels {
			if m.Status != "error" && m.DisplayName != "" {
				whitelist = append(whitelist, m.DisplayName)
				modelEntries[m.DisplayName] = map[string]interface{}{
					"name":  m.DisplayName,
					"limit": map[string]interface{}{"context": 128000, "output": 8192},
				}
			}
		}
		entry := map[string]interface{}{}
		if len(whitelist) > 0 {
			entry["whitelist"] = whitelist
		}
		if len(modelEntries) > 0 {
			entry["models"] = modelEntries
		}
		if p.Name != "" {
			entry["name"] = p.Name
		}
		if npm, ok := prefs["config/provider_npm_"+p.ID]; ok && npm != "" && npm != "null" && npm != "NULL" {
			entry["npm"] = npm
		}
		if optsJSON, ok := prefs["config/provider_options_"+p.ID]; ok && optsJSON != "" {
			var opts map[string]interface{}
			if json.Unmarshal([]byte(optsJSON), &opts) == nil {
				entry["options"] = opts
			}
		} else if p.BaseURL != "" {
			entry["options"] = map[string]interface{}{"baseURL": p.BaseURL}
		}
		if len(entry) > 0 {
			provSection[p.ID] = entry
		}
	}
	cfg["provider"] = provSection

	// ── Agents ─────────────────────────────────────────────────────
	agents, err := s.db.ListAgents()
	if err == nil && len(agents) > 0 {
		agentSection := make(map[string]interface{})
		for _, a := range agents {
			if a.Status != "active" {
				continue
			}
			entry := map[string]interface{}{}
			if a.Description != "" {
				entry["description"] = a.Description
			}
			if a.CurrentModelID != "" {
				entry["model"] = a.CurrentModelID
			}
			if a.Mode != "" {
				entry["mode"] = a.Mode
			}
			if a.Temperature > 0 {
				entry["temperature"] = a.Temperature
			}
			if a.Color != "" {
				entry["color"] = a.Color
			}
			if a.MaxSteps > 0 {
				entry["steps"] = a.MaxSteps
			}
			if a.PromptFile != "" {
				entry["prompt"] = a.PromptFile
			}
			if a.Permission != "" {
				var permMap map[string]interface{}
				if json.Unmarshal([]byte(a.Permission), &permMap) == nil {
					entry["permission"] = permMap
				}
			}
			if len(entry) > 0 {
				agentSection[a.ID] = entry
			}
		}
		if len(agentSection) > 0 {
			cfg["agent"] = agentSection
		}
	}

	// ── Commands ──────────────────────────────────────────────────
	commands, err := s.db.ListCommands()
	if err == nil && len(commands) > 0 {
		cmdSection := make(map[string]interface{})
		for _, c := range commands {
			if c.Status != "active" {
				continue
			}
			entry := map[string]interface{}{"template": c.Template}
			if c.Description != "" {
				entry["description"] = c.Description
			}
			if c.Agent != "" {
				entry["agent"] = c.Agent
			}
			if c.Model != "" {
				entry["model"] = c.Model
			}
			if c.Subtask {
				entry["subtask"] = true
			}
			cmdSection[c.ID] = entry
		}
		cfg["command"] = cmdSection
	}

	// ── MCP ────────────────────────────────────────────────────────
	mcps, err := s.db.ListMCPs()
	if err == nil && len(mcps) > 0 {
		mcpSection := make(map[string]interface{})
		for _, m := range mcps {
			entry := map[string]interface{}{}
			if m.Type == "local" {
				entry["type"] = "local"
				if m.Command != "" {
					var cmdArr []string
					if json.Unmarshal([]byte(m.Command), &cmdArr) == nil {
						entry["command"] = cmdArr
					}
				}
			} else if m.Type == "remote" {
				entry["type"] = "remote"
				if m.URL != "" {
					entry["url"] = m.URL
				}
			}
			if m.Enabled {
				entry["enabled"] = true
			}
			if m.Timeout > 0 {
				entry["timeout"] = m.Timeout
			}
			if m.EnvVars != "" {
				var envObj map[string]interface{}
				if json.Unmarshal([]byte(m.EnvVars), &envObj) == nil {
					entry["environment"] = envObj
				}
			}
			if len(entry) > 0 {
				mcpSection[m.ID] = entry
			}
		}
		cfg["mcp"] = mcpSection
	}

	// ── Meta preferences ─────────────────────────────────────────
	for k, v := range prefs {
		if !strings.HasPrefix(k, metaPref) {
			continue
		}
		stem := k[len(metaPref):]
		// Skip provider-specific and internal keys
		if strings.HasPrefix(stem, "provider_") {
			continue
		}
		if strings.HasPrefix(stem, "skills_") {
			continue
		}
		if strings.HasPrefix(stem, "compaction_") {
			continue
		}
		if stem == "lsp" {
			continue
		}
		var val interface{}
		_ = json.Unmarshal([]byte(v), &val)
		cfg[stem] = val
	}

	// ── Write ──────────────────────────────────────────────────────
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, out, 0644)
}
