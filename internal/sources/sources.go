package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/reeinharrrd/maestro/internal/config"
	"github.com/reeinharrrd/maestro/internal/db"
	"github.com/reeinharrrd/maestro/pkg/models"
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

func (s *Service) AddSource(ctx context.Context, remoteURL string) (*models.Source, error) {
	id := sourceIDFromURL(remoteURL)
	localPath := filepath.Join(config.SourcesDir(), id)

	existing, err := s.db.GetSource(id)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("source %q already exists (url: %s)", id, existing.RemoteURL)
	}

	if err := os.MkdirAll(config.SourcesDir(), 0755); err != nil {
		return nil, fmt.Errorf("create sources dir: %w", err)
	}

	source := &models.Source{
		ID:         id,
		RemoteURL:  remoteURL,
		LocalPath:  localPath,
		Status:     "active",
		LastSynced: 0,
	}

	if err := s.db.UpsertSource(source); err != nil {
		return nil, fmt.Errorf("upsert source: %w", err)
	}

	if err := s.syncSource(ctx, *source); err != nil {
		fmt.Printf("  Warning: initial sync of %s failed: %v\n", id, err)
	}

	return source, nil
}

func (s *Service) RemoveSource(ctx context.Context, id string) error {
	source, err := s.db.GetSource(id)
	if err != nil {
		return fmt.Errorf("source %q not found", id)
	}

	if source.LocalPath != "" {
		os.RemoveAll(source.LocalPath)
	}

	if err := s.db.DeleteSource(id); err != nil {
		return fmt.Errorf("delete source %q: %w", id, err)
	}

	fmt.Printf("  Removed source %s (was: %s)\n", id, source.RemoteURL)
	return nil
}

func (s *Service) SyncSourceByID(ctx context.Context, id string) error {
	source, err := s.db.GetSource(id)
	if err != nil {
		return fmt.Errorf("source %q not found", id)
	}
	return s.syncSource(ctx, *source)
}

func (s *Service) SourceByID(id string) (*models.Source, error) {
	return s.db.GetSource(id)
}

func (s *Service) AllSources() ([]models.Source, error) {
	return s.db.ListSources()
}

func sourceIDFromURL(rawURL string) string {
	u := rawURL

	u = strings.TrimPrefix(u, "https://")
	u = strings.TrimPrefix(u, "http://")
	u = strings.TrimPrefix(u, "git@")
	u = strings.TrimPrefix(u, "ssh://")

	u = strings.ReplaceAll(u, ":", "-")
	u = strings.ReplaceAll(u, "/", "-")
	u = strings.ReplaceAll(u, ".", "-")

	u = strings.TrimSuffix(u, "-git")

	re := regexp.MustCompile(`-+`)
	u = re.ReplaceAllString(u, "-")

	u = strings.Trim(u, "-")

	return u
}

func (s *Service) syncSource(ctx context.Context, source models.Source) error {
	if source.RemoteURL == "" {
		return fmt.Errorf("source %q has no remote URL", source.ID)
	}

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
	source.LastSynced = time.Now().Unix()
	if err := s.db.UpsertSource(&source); err != nil {
		return err
	}

	_, err = s.DiscoverItems(ctx, source)
	return err
}


func (s *Service) DiscoverItems(ctx context.Context, source models.Source) ([]models.SourceItem, error) {
	scanned, err := ScanAll(source.LocalPath)
	if err != nil {
		return nil, fmt.Errorf("scan source %s: %w", source.ID, err)
	}

	var items []models.SourceItem
	for _, si := range scanned {
		item := models.SourceItem{
			ID:         source.ID + "-" + si.ID,
			SourceID:   source.ID,
			Type:       si.Type,
			SourcePath: si.SourcePath,
			Status:     "active",
		}

		s.upsertEntity(source.ID, si)

		if err := s.db.UpsertSourceItem(&item); err != nil {
			fmt.Printf("  Warning: upsert source item %s: %v\n", item.ID, err)
			continue
		}
		items = append(items, item)
	}

	return items, nil
}

func (s *Service) upsertEntity(sourceID string, item ScannedItem) {
	switch item.Type {
	case "skill":
		_ = s.db.UpsertSkill(&models.Skill{
			ID:         sourceID + "-" + item.ID,
			Source:     sourceID,
			SourcePath: item.SourcePath,
			Status:     "active",
		})
	case "agent":
		_ = s.db.UpsertAgent(&models.Agent{
			ID:     sourceID + "-" + item.ID,
			Source: sourceID,
			Mode:   "subagent",
			Status: "active",
		})
	case "command":
		_ = s.db.UpsertCommand(&models.Command{
			ID:     sourceID + "-" + item.ID,
			Source: sourceID,
			Status: "active",
		})
	case "mcp":
		_ = s.db.UpsertMCP(&models.MCPServer{
			ID:      sourceID + "-" + item.ID,
			Source:  sourceID,
			Type:    "local",
			Enabled: true,
		})
	}
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
			ID:        id,
			RemoteURL: "",
			LocalPath: "",
			Commit:    srcData.Commit,
			Status:    "active",
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
