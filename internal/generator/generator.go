package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/reeinharrrd/opencode-kit/internal/db"
)

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

func (s *Service) GenerateConfig() error {
	providers, err := s.db.ListProviders()
	if err != nil {
		return fmt.Errorf("providers: %w", err)
	}

	cfg := map[string]interface{}{
		"$schema": "https://opencode.ai/config.json",
	}

	providerSection := make(map[string]interface{})
	for _, p := range providers {
		models, err := s.db.ListModelsByProvider(p.ID)
		if err != nil {
			continue
		}
		var whitelist []string
			for _, m := range models {
				if m.Status != "error" {
					whitelist = append(whitelist, m.DisplayName)
				}
			}
		if len(whitelist) > 0 {
			providerSection[p.ID] = map[string]interface{}{
				"whitelist": whitelist,
			}
		}
	}
	if len(providerSection) > 0 {
		cfg["provider"] = providerSection
	}

	agents, err := s.db.ListAgents()
	if err == nil && len(agents) > 0 {
		agentSection := make(map[string]interface{})
		for _, a := range agents {
			if a.Status != "active" {
				continue
			}
			entry := map[string]interface{}{
				"description": a.Description,
				"mode":        a.Mode,
			}
			if a.CurrentModelID != "" {
				entry["model"] = a.CurrentModelID
			}
			if a.Temperature > 0 {
				entry["temperature"] = a.Temperature
			}
			if a.Color != "" {
				entry["color"] = a.Color
			}
			if a.Permission != "" {
				entry["permission"] = a.Permission
			}
			if a.MaxSteps > 0 {
				entry["steps"] = a.MaxSteps
			}
			agentSection[a.ID] = entry
		}
		if len(agentSection) > 0 {
			cfg["agent"] = agentSection
		}
	}

	commands, err := s.db.ListCommands()
	if err == nil && len(commands) > 0 {
		cmdSection := make(map[string]interface{})
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
			cmdSection[c.ID] = entry
		}
		if len(cmdSection) > 0 {
			cfg["command"] = cmdSection
		}
	}

	mcps, err := s.db.ListMCPs()
	if err == nil && len(mcps) > 0 {
		mcpSection := make(map[string]interface{})
		for _, m := range mcps {
			entry := map[string]interface{}{
				"type":    m.Type,
				"enabled": false,
			}
			if m.Type == "local" && m.Command != "" {
				var cmdArr []string
				json.Unmarshal([]byte(m.Command), &cmdArr)
				entry["command"] = cmdArr
			}
			if m.Type == "remote" && m.URL != "" {
				entry["url"] = m.URL
			}
			mcpSection[m.ID] = entry
		}
		cfg["mcp"] = mcpSection
	}

	existing := s.readExistingConfig()
	merged := s.mergeWithExisting(cfg, existing)

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

func (s *Service) mergeWithExisting(generated, existing map[string]interface{}) map[string]interface{} {
	if existing == nil {
		return generated
	}
	result := make(map[string]interface{})
	for k, v := range existing {
		result[k] = v
	}
	for k, v := range generated {
		result[k] = v
	}
	return result
}
