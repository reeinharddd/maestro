package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/reeinharddd/okit/internal/db"
	"github.com/reeinharddd/okit/pkg/models"
)

type Service struct {
	db db.DBInterface
}

func New(database db.DBInterface) *Service {
	return &Service{db: database}
}

func (s *Service) Sync(ctx context.Context) error {
	sources, err := s.db.ListSources()
	if err != nil {
		return fmt.Errorf("list sources: %w", err)
	}
	for _, src := range sources {
		if err := s.syncSource(ctx, src); err != nil {
			fmt.Printf("  Warning: sync %s: %v\n", src.ID, err)
		}
	}
	return nil
}

func (s *Service) syncSource(ctx context.Context, source models.Source) error {
	if _, err := os.Stat(source.LocalPath); os.IsNotExist(err) {
		cmd := exec.CommandContext(ctx, "git", "clone", source.RemoteURL, source.LocalPath)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("clone %s: %w", source.RemoteURL, err)
		}
	} else {
		cmd := exec.CommandContext(ctx, "git", "-C", source.LocalPath, "pull")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("pull %s: %w", source.LocalPath, err)
		}
	}

	out, err := exec.Command("git", "-C", source.LocalPath, "rev-parse", "HEAD").Output()
	if err != nil {
		return fmt.Errorf("get commit: %w", err)
	}
	commit := string(out[:40])

	source.Commit = commit
	if err := s.db.UpsertSource(&source); err != nil {
		return err
	}

	_, err = s.DiscoverItems(ctx, source)
	return err
}

func (s *Service) DiscoverItems(ctx context.Context, source models.Source) ([]models.SourceItem, error) {
	var items []models.SourceItem

	for _, subdir := range []string{"skills", "agents", "commands"} {
		dir := filepath.Join(source.LocalPath, subdir)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			name := strings.TrimSuffix(entry.Name(), ".md")
			itemType := subdir[:len(subdir)-1]

			item := models.SourceItem{
				ID:         source.ID + "-" + name,
				SourceID:   source.ID,
				Type:       itemType,
				SourcePath: filepath.Join(dir, entry.Name()),
				Status:     "active",
			}
			items = append(items, item)
		}
	}

	for _, item := range items {
		skill := models.Skill{
			ID:         item.ID,
			Source:     source.ID,
			SourcePath: item.SourcePath,
			Type:       item.Type,
			Status:     item.Status,
		}
		_ = s.db.UpsertSkill(&skill)
		_ = s.db.UpsertSourceItem(&item)
	}

	return items, nil
}

func (s *Service) ImportSourceRegistry(registryPath string) error {
	data, err := os.ReadFile(registryPath)
	if err != nil {
		return fmt.Errorf("read registry: %w", err)
	}

	var registry struct {
		Sources map[string]struct {
			Commit string `json:"commit"`
			Items  struct {
				Skills   []string `json:"skills"`
				Agents   []string `json:"agents"`
				Commands []string `json:"commands"`
				MCPs     []string `json:"mcps"`
			} `json:"items"`
		} `json:"sources"`
	}

	if err := json.Unmarshal(data, &registry); err != nil {
		return fmt.Errorf("parse registry: %w", err)
	}

	for id, srcData := range registry.Sources {
		source := models.Source{
			ID:     id,
			RemoteURL: "",
			LocalPath: "",
			Commit: srcData.Commit,
			Status: "active",
		}
		if err := s.db.UpsertSource(&source); err != nil {
			return fmt.Errorf("upsert source %s: %w", id, err)
		}

		for _, skillID := range srcData.Items.Skills {
			item := models.SourceItem{ID: skillID, SourceID: id, Type: "skill", Status: "active"}
			_ = s.db.UpsertSourceItem(&item)
			_ = s.db.UpsertSkill(&models.Skill{ID: skillID, Source: id, Type: "skill", Status: "active"})
		}

		for _, agentID := range srcData.Items.Agents {
			item := models.SourceItem{ID: agentID, SourceID: id, Type: "agent", Status: "active"}
			_ = s.db.UpsertSourceItem(&item)
			_ = s.db.UpsertAgent(&models.Agent{ID: agentID, Source: id, Mode: "subagent", Status: "active"})
		}

		for _, cmdID := range srcData.Items.Commands {
			item := models.SourceItem{ID: cmdID, SourceID: id, Type: "command", Status: "active"}
			_ = s.db.UpsertSourceItem(&item)
			_ = s.db.UpsertCommand(&models.Command{ID: cmdID, Source: id, Status: "active"})
		}

		for _, mcpID := range srcData.Items.MCPs {
			item := models.SourceItem{ID: mcpID, SourceID: id, Type: "mcp", Status: "active"}
			_ = s.db.UpsertSourceItem(&item)
			_ = s.db.UpsertMCP(&models.MCPServer{ID: mcpID, Source: id, Type: "local", Enabled: true})
		}
	}

	fmt.Printf("  Imported %d sources from registry\n", len(registry.Sources))
	return nil
}
