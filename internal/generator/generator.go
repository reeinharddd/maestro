package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/reeinharrrd/opencode-kit/internal/db"
	"github.com/reeinharrrd/opencode-kit/pkg/models"
)

const metaPrefix = "config/"

type Service struct {
	db        db.DBInterface
	outputDir string
}

func NewService(db db.DBInterface, outputDir string) *Service {
	if outputDir == "" {
		outputDir = filepath.Dir(db.DBPath())
	}
	return &Service{db: db, outputDir: outputDir}
}

const (
	defaultContextWindow = 128000
	defaultMaxOutput     = 8192
)

func (s *Service) GenerateConfig() error {
	if err := s.SyncExistingToDB(); err != nil {
		fmt.Printf("  Warning: sync existing config: %v\n", err)
	}

	providers, err := s.db.ListProviders()
	if err != nil {
		return fmt.Errorf("providers: %w", err)
	}

	profiles, err := s.loadProfiles()
	if err != nil {
		return fmt.Errorf("profiles: %w", err)
	}

	cfg := map[string]interface{}{
		"$schema": "https://opencode.ai/config.json",
	}

	totalActive, totalError, providerSection := s.buildProviderSection(providers, profiles)
	if len(providerSection) > 0 {
		cfg["provider"] = providerSection
	}

	if section, err := s.buildAgentSection(); err == nil {
		cfg["agent"] = section
	}

	cfg["permission"] = map[string]interface{}{}
	cfg["experimental"] = map[string]interface{}{}

	if err := s.writeStateFile(totalActive, totalError, len(providerSection)); err != nil {
		return fmt.Errorf("state file: %w", err)
	}

	if section, err := s.buildCommandSection(); err == nil {
		cfg["command"] = section
	}

	if section, err := s.buildMCPSection(); err == nil {
		cfg["mcp"] = section
	}

	meta := s.buildMetaFromDB()
	for k, v := range meta {
		cfg[k] = v
	}

	merged := s.mergeWithExisting(cfg, s.readExistingConfig())

	out, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	configPath := filepath.Join(s.outputDir, "opencode.jsonc")
	if err := os.WriteFile(configPath, out, 0644); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	fmt.Printf("  Generated: %s\n", configPath)
	return nil
}

func (s *Service) SyncExistingToDB() error {
	cfg := s.readExistingConfig()
	if cfg == nil {
		return nil
	}

	syncMCP := func() {
		mcpRaw, ok := cfg["mcp"].(map[string]interface{})
		if !ok {
			return
		}
		for id, val := range mcpRaw {
			entry, ok := val.(map[string]interface{})
			if !ok {
				continue
			}
			m := models.MCPServer{ID: id, Source: "sync"}
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
		}
	}

	syncCommands := func() {
		cmdsRaw, ok := cfg["command"].(map[string]interface{})
		if !ok {
			return
		}
		for id, val := range cmdsRaw {
			entry, ok := val.(map[string]interface{})
			if !ok {
				continue
			}
			c := models.Command{ID: id, Source: "sync", Status: "active"}
			if t, _ := entry["template"].(string); t != "" {
				c.Template = t
			}
			if desc, _ := entry["description"].(string); desc != "" {
				c.Description = desc
			}
			if ag, _ := entry["agent"].(string); ag != "" {
				c.Agent = ag
			}
			if mod, _ := entry["model"].(string); mod != "" {
				c.Model = mod
			}
			if st, _ := entry["subtask"].(bool); st {
				c.Subtask = st
			}
			_ = s.db.UpsertCommand(&c)
		}
	}

	syncLSP := func() {
		lspBool, isBool := cfg["lsp"].(bool)
		lspObj, isObj := cfg["lsp"].(map[string]interface{})
		if isBool {
			_ = s.db.SetPreference(metaPrefix+"lsp", fmt.Sprintf("%t", lspBool))
		} else if isObj {
			_ = s.db.SetPreference(metaPrefix+"lsp", "object")
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
					l.Disabled = dis
				}
				_ = s.db.UpsertLSPServer(&l)
			}
		}
	}

	setJSONPref := func(key string, val interface{}) {
		b, _ := json.Marshal(val)
		_ = s.db.SetPreference(key, string(b))
	}

	syncMeta := func() {
		simple := []string{"autoupdate", "disabled_providers", "model", "small_model", "share", "plugin"}
		for _, key := range simple {
			if v, exists := cfg[key]; exists {
				setJSONPref(metaPrefix+key, v)
			}
		}

		if skills, ok := cfg["skills"].(map[string]interface{}); ok {
			for sk, sv := range skills {
				setJSONPref(metaPrefix+"skills_"+sk, sv)
			}
		}

		if comp, ok := cfg["compaction"].(map[string]interface{}); ok {
			for ck, cv := range comp {
				setJSONPref(metaPrefix+"compaction_"+ck, cv)
			}
		}
	}

	syncMCP()
	syncCommands()
	syncLSP()
	syncMeta()
	return nil
}

func (s *Service) loadProfiles() (map[string]models.ModelProfile, error) {
	list, err := s.db.ListModelProfiles()
	if err != nil {
		return nil, err
	}
	out := make(map[string]models.ModelProfile, len(list))
	for _, p := range list {
		out[p.ModelID] = p
	}
	return out, nil
}

func (s *Service) buildProviderSection(providers []models.Provider, profiles map[string]models.ModelProfile) (int, int, map[string]interface{}) {
	section := make(map[string]interface{})
	totalActive := 0
	totalError := 0
	for _, p := range providers {
		models, err := s.db.ListModelsByProvider(p.ID)
		if err != nil {
			continue
		}
		whitelist := make([]string, 0, len(models))
		modelEntries := make(map[string]interface{})
		for _, m := range models {
			if m.Status == "error" {
				totalError++
				continue
			}
			totalActive++
			whitelist = append(whitelist, m.DisplayName)
			entry := buildModelEntry(m, profiles[m.ID])
			modelEntries[m.DisplayName] = entry
		}
		entry := map[string]interface{}{}
		if len(whitelist) > 0 {
			entry["whitelist"] = whitelist
		}
		if len(modelEntries) > 0 {
			entry["models"] = modelEntries
		}
		if p.BaseURL != "" {
			entry["api"] = p.BaseURL
		}
		if p.Name != "" {
			entry["name"] = p.Name
		}
		if len(entry) > 0 {
			section[p.ID] = entry
		}
	}
	return totalActive, totalError, section
}

func buildModelEntry(m models.Model, profile models.ModelProfile) map[string]interface{} {
	context := m.ContextWindow
	if context <= 0 {
		context = defaultContextWindow
	}
	maxOutput := defaultMaxOutput
	if profile.MaxOutput > 0 {
		maxOutput = profile.MaxOutput
	}

	entry := map[string]interface{}{}
	if m.DisplayName != "" {
		entry["name"] = m.DisplayName
	}
	entry["limit"] = map[string]interface{}{
		"context": context,
		"output":  maxOutput,
	}
	entry["tool_call"] = m.FunctionCalling
	entry["attachment"] = m.Vision
	entry["cost"] = map[string]interface{}{
		"input":  m.PricingPrompt,
		"output": m.PricingCompletion,
	}
	inputModalities := []string{"text"}
	if m.Vision {
		inputModalities = append(inputModalities, "image")
	}
	entry["modalities"] = map[string]interface{}{
		"input":  inputModalities,
		"output": []string{"text"},
	}
	switch m.Status {
	case "deprecated":
		entry["status"] = "deprecated"
	case "beta":
		entry["status"] = "beta"
	case "alpha":
		entry["status"] = "alpha"
	}
	return entry
}

func (s *Service) buildAgentSection() (map[string]interface{}, error) {
	agents, err := s.db.ListAgents()
	if err != nil {
		return nil, err
	}
	section := make(map[string]interface{})
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
			if err := json.Unmarshal([]byte(a.Permission), &permMap); err == nil {
				entry["permission"] = permMap
			}
		}
		if len(entry) > 0 {
			section[a.ID] = entry
		}
	}
	return section, nil
}

func (s *Service) buildCommandSection() (map[string]interface{}, error) {
	commands, err := s.db.ListCommands()
	if err != nil {
		return nil, err
	}
	section := make(map[string]interface{})
	for _, c := range commands {
		if c.Status != "active" {
			continue
		}
		entry := map[string]interface{}{
			"template": c.Template,
		}
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
		section[c.ID] = entry
	}
	return section, nil
}

func (s *Service) buildMCPSection() (map[string]interface{}, error) {
	mcps, err := s.db.ListMCPs()
	if err != nil {
		return nil, err
	}
	section := make(map[string]interface{})
	for _, m := range mcps {
		entry := map[string]interface{}{}
		if m.Type == "local" {
			entry["type"] = "local"
			if m.Command != "" {
				var cmdArr []string
				if err := json.Unmarshal([]byte(m.Command), &cmdArr); err == nil {
					entry["command"] = cmdArr
				}
			}
		} else if m.Type == "remote" {
			entry["type"] = "remote"
			if m.URL != "" {
				entry["url"] = m.URL
			}
		}
		entry["enabled"] = m.Enabled
		if m.Timeout > 0 {
			entry["timeout"] = m.Timeout
		}
		if m.EnvVars != "" {
			var envObj map[string]interface{}
			if err := json.Unmarshal([]byte(m.EnvVars), &envObj); err == nil {
				entry["environment"] = envObj
			}
		}
		if len(entry) > 0 {
			section[m.ID] = entry
		}
	}
	return section, nil
}

func (s *Service) writeStateFile(active, errored, providers int) error {
	state := map[string]interface{}{
		"active_models": active,
		"error_models":  errored,
		"providers":     providers,
		"generated_at":  time.Now().UTC().Format(time.RFC3339),
		"source":        "opencode-kit",
	}
	if routes, err := s.db.ListRoutingRules(); err == nil && len(routes) > 0 {
		routeEntries := make(map[string]interface{})
		for _, r := range routes {
			entry := map[string]interface{}{}
			if r.CurrentModelID != "" {
				entry["model"] = r.CurrentModelID
			}
			if r.FallbackIDs != "" {
				entry["fallback"] = r.FallbackIDs
			}
			if r.Description != "" {
				entry["description"] = r.Description
			}
			if r.MinContext > 0 {
				entry["min_context"] = r.MinContext
			}
			if r.NeedsFC {
				entry["needs_fc"] = true
			}
			if r.NeedsVision {
				entry["needs_vision"] = true
			}
			if r.MaxCostPerCall > 0 {
				entry["max_cost"] = r.MaxCostPerCall
			}
			routeEntries[r.TaskKey] = entry
		}
		state["routes"] = routeEntries
	}
	if skills, err := s.db.ListSkills(); err == nil && len(skills) > 0 {
		skillEntries := make([]string, 0, len(skills))
		for _, sk := range skills {
			if sk.Status == "active" {
				skillEntries = append(skillEntries, sk.ID)
			}
		}
		state["skills"] = skillEntries
	}

	out, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(s.outputDir, "okit-state.json")
	if err := os.WriteFile(path, out, 0644); err != nil {
		return err
	}
	fmt.Printf("  Generated: %s\n", path)
	return nil
}

func (s *Service) GenerateAgents() error {
	agents, err := s.db.ListAgents()
	if err != nil {
		return err
	}

	agentsDir := filepath.Join(s.outputDir, "agents")
	os.MkdirAll(agentsDir, 0755)

	for _, a := range agents {
		if a.Status != "active" {
			continue
		}
		path := filepath.Join(agentsDir, a.ID+".md")
		fm := fmt.Sprintf("---\ndescription: %s\nmode: %s\n", a.Description, a.Mode)
		if a.CurrentModelID != "" {
			fm += fmt.Sprintf("model: %s\n", a.CurrentModelID)
		}
		if a.Temperature > 0 {
			fm += fmt.Sprintf("temperature: %.1f\n", a.Temperature)
		}
		if a.Color != "" {
			fm += fmt.Sprintf("color: %s\n", a.Color)
		}
		if a.Permission != "" {
			fm += fmt.Sprintf("permission:\n")
			var permMap map[string]interface{}
			if err := json.Unmarshal([]byte(a.Permission), &permMap); err == nil {
				for k, v := range permMap {
					fm += fmt.Sprintf("  %s: %v\n", k, v)
				}
			}
		}
		fm += "---\n\n"

		prompt := "# " + a.Description + "\n"
		if a.PromptFile != "" {
			prompt += "\n" + a.PromptFile + "\n"
		}
		content := fm + prompt

		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("write agent %s: %w", a.ID, err)
		}
		fmt.Printf("  Agent: %s\n", path)
	}
	return nil
}

func (s *Service) GenerateCommands() error {
	commands, err := s.db.ListCommands()
	if err != nil {
		return err
	}

	cmdsDir := filepath.Join(s.outputDir, "commands")
	os.MkdirAll(cmdsDir, 0755)

	for _, c := range commands {
		if c.Status != "active" {
			continue
		}
		path := filepath.Join(cmdsDir, c.ID+".md")
		fm := "---\n"
		if c.Description != "" {
			fm += fmt.Sprintf("description: %s\n", c.Description)
		}
		if c.Agent != "" {
			fm += fmt.Sprintf("agent: %s\n", c.Agent)
		}
		if c.Model != "" {
			fm += fmt.Sprintf("model: %s\n", c.Model)
		}
		fm += "---\n\n"

		tpl := c.Template
		if tpl == "" {
			tpl = "# " + c.Description
		}
		content := fm + tpl + "\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("write command %s: %w", c.ID, err)
		}
		fmt.Printf("  Command: %s\n", path)
	}
	return nil
}

func (s *Service) readExistingConfig() map[string]interface{} {
	path := filepath.Join(s.outputDir, "opencode.jsonc")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil
	}
	return cfg
}

func (s *Service) buildMetaFromDB() map[string]interface{} {
	prefs, err := s.db.ListPreferences()
	if err != nil {
		return nil
	}
	result := make(map[string]interface{})
	compactionKeys := make(map[string]interface{})

	for k, v := range prefs {
		if !strings.HasPrefix(k, metaPrefix) {
			continue
		}
		stem := k[len(metaPrefix):]

		if strings.HasPrefix(stem, "compaction_") {
			var val interface{}
			_ = json.Unmarshal([]byte(v), &val)
			compactionKeys[stem[len("compaction_"):]] = val
			continue
		}
		if strings.HasPrefix(stem, "skills_") {
			sk := stem[len("skills_"):]
			if _, exists := result["skills"]; !exists {
				result["skills"] = make(map[string]interface{})
			}
			obj := result["skills"].(map[string]interface{})
			var val interface{}
			_ = json.Unmarshal([]byte(v), &val)
			obj[sk] = val
			continue
		}
		var val interface{}
		_ = json.Unmarshal([]byte(v), &val)
		result[stem] = val
	}
	if len(compactionKeys) > 0 {
		result["compaction"] = compactionKeys
	}
	return result
}

var generatorManagedKeys = map[string]bool{
	"$schema":           true,
	"provider":          true,
	"agent":             true,
	"command":           true,
	"mcp":               true,
	"permission":        true,
	"experimental":      true,
	"lsp":               true,
	"plugin":            true,
	"skills":            true,
	"autoupdate":        true,
	"disabled_providers": true,
	"model":             true,
	"small_model":       true,
	"share":             true,
	"compaction":        true,
}

func (s *Service) mergeWithExisting(generated, existing map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range existing {
		if !generatorManagedKeys[k] {
			result[k] = v
		}
	}
	for k, v := range generated {
		if k == "mcp" {
			genMcp, _ := v.(map[string]interface{})
			existingMcp, _ := existing["mcp"].(map[string]interface{})
			result["mcp"] = mergeMCP(genMcp, existingMcp)
			continue
		}
		result[k] = v
	}
	return result
}

func mergeMCP(generated, existing map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{}, len(generated)+len(existing))
	for k, v := range generated {
		merged[k] = v
	}
	for k, v := range existing {
		if _, covered := merged[k]; !covered {
			merged[k] = v
		}
	}
	return merged
}
