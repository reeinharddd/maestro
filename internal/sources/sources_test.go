package sources_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/reeinharrrd/maestro/internal/db"
	"github.com/reeinharrrd/maestro/internal/sources"
	"github.com/reeinharrrd/maestro/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDB implements db.DBInterface for sources tests.
type mockDB struct {
	sources     []models.Source
	skills      []models.Skill
	items       []models.SourceItem
	agents      []models.Agent
	commands    []models.Command
	mcpServers  []models.MCPServer
}

func (m *mockDB) ListSources() ([]models.Source, error) { return m.sources, nil }

func (m *mockDB) GetSource(id string) (*models.Source, error) {
	for _, s := range m.sources {
		if s.ID == id {
			return &s, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockDB) DeleteSource(id string) error {
	for i, s := range m.sources {
		if s.ID == id {
			m.sources = append(m.sources[:i], m.sources[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockDB) UpsertSource(src *models.Source) error {
	for i, s := range m.sources {
		if s.ID == src.ID {
			m.sources[i] = *src
			return nil
		}
	}
	m.sources = append(m.sources, *src)
	return nil
}

func (m *mockDB) UpsertSkill(s *models.Skill) error           { m.skills = append(m.skills, *s); return nil }
func (m *mockDB) UpsertSourceItem(s *models.SourceItem) error { m.items = append(m.items, *s); return nil }
func (m *mockDB) UpsertAgent(a *models.Agent) error           { m.agents = append(m.agents, *a); return nil }
func (m *mockDB) UpsertCommand(c *models.Command) error       { m.commands = append(m.commands, *c); return nil }
func (m *mockDB) UpsertMCP(mcp *models.MCPServer) error       { m.mcpServers = append(m.mcpServers, *mcp); return nil }

// Unimplemented methods.
func (m *mockDB) UpsertProvider(p *models.Provider) error                                    { panic("unexpected") }
func (m *mockDB) ListProviders() ([]models.Provider, error)                                   { panic("unexpected") }
func (m *mockDB) GetProvider(id string) (*models.Provider, error)                              { panic("unexpected") }
func (m *mockDB) DeleteProvider(id string) error                                               { panic("unexpected") }
func (m *mockDB) UpsertModel(mdl *models.Model) error                                         { panic("unexpected") }
func (m *mockDB) ListModels(opts ...db.ModelFilter) ([]models.Model, error)                    { panic("unexpected") }
func (m *mockDB) ListModelsByProvider(providerID string) ([]models.Model, error)               { panic("unexpected") }
func (m *mockDB) GetModel(id string) (*models.Model, error)                                   { panic("unexpected") }
func (m *mockDB) DeleteModel(id string) error                                                  { panic("unexpected") }
func (m *mockDB) ListCommands() ([]models.Command, error)                                     { panic("unexpected") }
func (m *mockDB) ListMCPs() ([]models.MCPServer, error)                                       { panic("unexpected") }
func (m *mockDB) ListSkills() ([]models.Skill, error)                                         { panic("unexpected") }
func (m *mockDB) ListSourceItems() ([]models.SourceItem, error)                               { panic("unexpected") }
func (m *mockDB) GetSourceItem(id string) (*models.SourceItem, error)                         { panic("unexpected") }
func (m *mockDB) DeleteSourceItem(id string) error                                            { panic("unexpected") }
func (m *mockDB) UpsertLSPServer(l *models.LSPServer) error                                   { panic("unexpected") }
func (m *mockDB) ListLSPServers() ([]models.LSPServer, error)                                 { panic("unexpected") }
func (m *mockDB) GetLSPServer(id string) (*models.LSPServer, error)                           { panic("unexpected") }
func (m *mockDB) DeleteLSPServer(id string) error                                             { panic("unexpected") }
func (m *mockDB) UpsertConfigFragment(f *models.ConfigFragment) error                         { panic("unexpected") }
func (m *mockDB) ListConfigFragments(limit int) ([]models.ConfigFragment, error)               { panic("unexpected") }
func (m *mockDB) GetConfigFragment(id string) (*models.ConfigFragment, error)                 { panic("unexpected") }
func (m *mockDB) UpsertModelProfile(p *models.ModelProfile) error                             { panic("unexpected") }
func (m *mockDB) ListModelProfiles() ([]models.ModelProfile, error)                           { panic("unexpected") }
func (m *mockDB) GetModelProfile(modelID string) (*models.ModelProfile, error)                { panic("unexpected") }
func (m *mockDB) ListAgents() ([]models.Agent, error)                                         { panic("unexpected") }
func (m *mockDB) InsertRoutingEvent(e *models.RoutingEvent) error                             { panic("unexpected") }
func (m *mockDB) ListRoutingRules() ([]models.RoutingRule, error)                             { panic("unexpected") }
func (m *mockDB) UpsertRoutingRule(r *models.RoutingRule) error                               { panic("unexpected") }
func (m *mockDB) GetBudget() (*models.BudgetConfig, error)                                    { panic("unexpected") }
func (m *mockDB) SetPreference(key, value string) error                                       { panic("unexpected") }
func (m *mockDB) ListPreferences() (map[string]string, error)                                 { panic("unexpected") }
func (m *mockDB) Query(query string, args ...any) (*sql.Rows, error)                          { panic("unexpected") }
func (m *mockDB) Exec(query string, args ...any) (sql.Result, error)                          { panic("unexpected") }
func (m *mockDB) GetSmallFastModels(ctx context.Context) ([]models.Model, error)               { panic("unexpected") }
func (m *mockDB) DBPath() string                                                              { panic("unexpected") }
func (m *mockDB) SearchModels(query string) ([]models.Model, error) { panic("unexpected") }
func (m *mockDB) GetStats() (map[string]int, error) { panic("unexpected") }
func (m *mockDB) DeleteCommand(id string) error { panic("unexpected") }
func (m *mockDB) DeleteMCP(id string) error { panic("unexpected") }
func (m *mockDB) DeleteSkill(id string) error { panic("unexpected") }
func (m *mockDB) UpdateSkillMeta(id string, updates map[string]any) error { panic("unexpected") }
func (m *mockDB) SearchSkills(query string) ([]models.Skill, error) { panic("unexpected") }
func (m *mockDB) GetAgent(id string) (*models.Agent, error) { panic("unexpected") }
func (m *mockDB) DeleteAgent(id string) error { panic("unexpected") }
func (m *mockDB) GetRoutingRule(key string) (*models.RoutingRule, error) { panic("unexpected") }
func (m *mockDB) DeleteRoutingRule(key string) error { panic("unexpected") }
func (m *mockDB) ListRoutingEvents(limit int) ([]models.RoutingEvent, error) { panic("unexpected") }
func (m *mockDB) UpsertBudget(b *models.BudgetConfig) error { panic("unexpected") }
func (m *mockDB) GetPreference(key string) (string, error) { panic("unexpected") }
func (m *mockDB) DeletePreference(key string) error { panic("unexpected") }
func (m *mockDB) CleanupProviderPrefs() (int, error) { panic("unexpected") }
func (m *mockDB) CleanupInvalidPreferences() (int, error) { panic("unexpected") }
func (m *mockDB) InsertSyncLog(phase, status, details string, durationMs int64) error { panic("unexpected") }
func (m *mockDB) ListSyncLogs(limit int) ([]models.SyncLog, error) { panic("unexpected") }
func (m *mockDB) InsertExecLog(l *models.ExecLog) error { panic("unexpected") }
func (m *mockDB) ListExecLogs(limit int) ([]models.ExecLog, error) { panic("unexpected") }
func (m *mockDB) InsertSnapshot(hash, content string) error { panic("unexpected") }
func (m *mockDB) ListSnapshots(limit int) ([]models.Snapshot, error) { panic("unexpected") }
func (m *mockDB) GetSnapshot(id int64) (*models.Snapshot, error) { panic("unexpected") }
func (m *mockDB) DeleteSnapshot(id int64) error { panic("unexpected") }
func (m *mockDB) UpsertProject(p *models.Project) error { panic("unexpected") }
func (m *mockDB) ListProjects() ([]models.Project, error) { panic("unexpected") }
func (m *mockDB) GetProject(id string) (*models.Project, error) { panic("unexpected") }
func (m *mockDB) DeleteProject(id string) error { panic("unexpected") }
func (m *mockDB) UpsertDetectedStack(d *models.DetectedStack) error { panic("unexpected") }
func (m *mockDB) ListDetectedStacks(projectID string) ([]models.DetectedStack, error) { panic("unexpected") }
func (m *mockDB) DeleteDetectedStacks(projectID string) error { panic("unexpected") }
func (m *mockDB) UpsertProjectConfig(p *models.ProjectConfig) error { panic("unexpected") }
func (m *mockDB) ListProjectConfigs(projectID string) ([]models.ProjectConfig, error) { panic("unexpected") }
func (m *mockDB) GetProjectConfig(projectID, configType string) (*models.ProjectConfig, error) { panic("unexpected") }
func (m *mockDB) DeleteProjectConfigs(projectID string) error { panic("unexpected") }
func (m *mockDB) UpdateSourceItemStatus(id, status string) error {
	for i, item := range m.items {
		if item.ID == id {
			m.items[i].Status = status
			return nil
		}
	}
	return nil
}
func (m *mockDB) UpdateSourceItemTarget(id, targetPath string) error {
	for i, item := range m.items {
		if item.ID == id {
			m.items[i].TargetPath = targetPath
			return nil
		}
	}
	return nil
}
func (m *mockDB) ListSourceItemsBySource(sourceID string) ([]models.SourceItem, error) {
	var out []models.SourceItem
	for _, item := range m.items {
		if item.SourceID == sourceID {
			out = append(out, item)
		}
	}
	return out, nil
}

func TestDiscoverItems(t *testing.T) {
	t.Parallel()

	t.Run("discovers md files in skills/agents/commands dirs", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()

		dirs := []string{"skills", "agents", "commands", "mcps", "plugins"}
		files := map[string]string{
			"skills/my-skill.md":      "# My Skill",
			"agents/my-agent.md":      "# My Agent",
			"commands/my-cmd.md":      "# My Command",
			"mcps/my-mcp.md":          "# My MCP",
			"plugins/my-plugin.md":    "# My Plugin",
		}

		for _, d := range dirs {
			require.NoError(t, os.MkdirAll(filepath.Join(tmp, d), 0755))
		}
		for path, content := range files {
			require.NoError(t, os.WriteFile(filepath.Join(tmp, path), []byte(content), 0644))
		}

		src := models.Source{
			ID:        "test-source",
			LocalPath: tmp,
			Status:    "active",
		}

		m := &mockDB{}
		svc := sources.New(m)
		items, err := svc.DiscoverItems(context.Background(), src)
		require.NoError(t, err)
		require.Len(t, items, 5)
		itemByID := make(map[string]models.SourceItem)
		for _, item := range items {
			itemByID[item.ID] = item
		}

		assert.Equal(t, "skill", itemByID["test-source-skills-my-skill"].Type)
		assert.Equal(t, "agent", itemByID["test-source-agents-my-agent"].Type)
		assert.Equal(t, "command", itemByID["test-source-commands-my-cmd"].Type)
		assert.Equal(t, "mcp", itemByID["test-source-mcps-my-mcp"].Type)
		assert.Equal(t, "plugin", itemByID["test-source-plugins-my-plugin"].Type)

		// Entity routing: each type maps to correct entity, not all Skills
		assert.Len(t, m.skills, 1)
		assert.Len(t, m.agents, 1)
		assert.Len(t, m.commands, 1)
		assert.Len(t, m.mcpServers, 1)
	})

	t.Run("discovers SKILL.md and .opencode at root", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()

		require.NoError(t, os.WriteFile(filepath.Join(tmp, "SKILL.md"), []byte("# Root Skill"), 0644))
		require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".opencode"), 0755))

		src := models.Source{
			ID:        "test-source",
			LocalPath: tmp,
			Status:    "active",
		}

		m := &mockDB{}
		svc := sources.New(m)
		items, err := svc.DiscoverItems(context.Background(), src)
		require.NoError(t, err)
		assert.Len(t, items, 2)
	})

	t.Run("skips non-md files and directories", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()

		require.NoError(t, os.MkdirAll(filepath.Join(tmp, "skills"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(tmp, "skills", "notes.txt"), []byte("text"), 0644))
		require.NoError(t, os.MkdirAll(filepath.Join(tmp, "skills", "subdir"), 0755))

		src := models.Source{
			ID:        "test-source",
			LocalPath: tmp,
			Status:    "active",
		}

		m := &mockDB{}
		svc := sources.New(m)
		items, err := svc.DiscoverItems(context.Background(), src)
		require.NoError(t, err)
		assert.Empty(t, items)
	})

	t.Run("missing subdirs are silently skipped", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()

		src := models.Source{
			ID:        "test-source",
			LocalPath: tmp,
			Status:    "active",
		}

		m := &mockDB{}
		svc := sources.New(m)
		items, err := svc.DiscoverItems(context.Background(), src)
		require.NoError(t, err)
		assert.Empty(t, items)
	})
}

func TestAddSource(t *testing.T) {
	t.Parallel()

	t.Run("adds a source to the database", func(t *testing.T) {
		t.Parallel()
		m := &mockDB{}
		svc := sources.New(m)

		src, err := svc.AddSource(context.Background(), "https://github.com/user/my-skills.git")
		require.NoError(t, err)
		require.NotNil(t, src)
		assert.Equal(t, "github-com-user-my-skills", src.ID)
		assert.Equal(t, "https://github.com/user/my-skills.git", src.RemoteURL)
		assert.Equal(t, "active", src.Status)
	})

	t.Run("returns error for duplicate source", func(t *testing.T) {
		t.Parallel()
		m := &mockDB{}
		svc := sources.New(m)

		_, err := svc.AddSource(context.Background(), "https://github.com/user/my-skills.git")
		require.NoError(t, err)

		_, err = svc.AddSource(context.Background(), "https://github.com/user/my-skills.git")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("generates unique IDs for different URLs", func(t *testing.T) {
		t.Parallel()
		m := &mockDB{}
		svc := sources.New(m)

		src1, err := svc.AddSource(context.Background(), "https://github.com/a/b")
		require.NoError(t, err)
		assert.Equal(t, "github-com-a-b", src1.ID)

		src2, err := svc.AddSource(context.Background(), "https://gitlab.com/c/d.git")
		require.NoError(t, err)
		assert.Equal(t, "gitlab-com-c-d", src2.ID)

		src3, err := svc.AddSource(context.Background(), "git@github.com:user/repo.git")
		require.NoError(t, err)
		assert.Equal(t, "github-com-user-repo", src3.ID)
		assert.Len(t, m.sources, 3)
	})
}

func TestRemoveSource(t *testing.T) {
	t.Parallel()

	t.Run("removes source from database", func(t *testing.T) {
		t.Parallel()
		m := &mockDB{}
		svc := sources.New(m)

		_, err := svc.AddSource(context.Background(), "https://github.com/user/test-repo")
		require.NoError(t, err)
		assert.Len(t, m.sources, 1)

		err = svc.RemoveSource(context.Background(), "github-com-user-test-repo")
		require.NoError(t, err)
		assert.Empty(t, m.sources)
	})

	t.Run("returns error for non-existent source", func(t *testing.T) {
		t.Parallel()
		m := &mockDB{}
		svc := sources.New(m)

		err := svc.RemoveSource(context.Background(), "non-existent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestSyncSourceByID(t *testing.T) {
	t.Parallel()

	t.Run("returns error for non-existent source", func(t *testing.T) {
		t.Parallel()
		m := &mockDB{}
		svc := sources.New(m)

		err := svc.SyncSourceByID(context.Background(), "non-existent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestImportSourceRegistry(t *testing.T) {
	t.Parallel()

	t.Run("imports registry with all item types", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()
		regPath := filepath.Join(tmp, "registry.json")

		reg := map[string]interface{}{
			"sources": map[string]interface{}{
				"github-community": map[string]interface{}{
					"commit": "abc123",
					"items": map[string]interface{}{
						"skills":   []string{"skill-one", "skill-two"},
						"agents":   []string{"agent-alpha"},
						"commands": []string{"cmd-x"},
						"mcps":     []string{"mcp-server"},
					},
				},
			},
		}
		data, err := json.Marshal(reg)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(regPath, data, 0644))

		m := &mockDB{}
		svc := sources.New(m)

		err = svc.ImportSourceRegistry(regPath)
		require.NoError(t, err)

		require.Len(t, m.items, 5)
		assert.Len(t, m.skills, 2)
		assert.Len(t, m.agents, 1)
		assert.Len(t, m.commands, 1)
		assert.Len(t, m.mcpServers, 1)

		assert.Equal(t, "skill-one", m.skills[0].ID)
		assert.Equal(t, "agent-alpha", m.agents[0].ID)
		assert.Equal(t, "cmd-x", m.commands[0].ID)
		assert.Equal(t, "mcp-server", m.mcpServers[0].ID)
	})

	t.Run("handles empty registry", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()
		regPath := filepath.Join(tmp, "empty.json")

		reg := map[string]interface{}{
			"sources": map[string]interface{}{},
		}
		data, err := json.Marshal(reg)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(regPath, data, 0644))

		m := &mockDB{}
		svc := sources.New(m)
		err = svc.ImportSourceRegistry(regPath)
		require.NoError(t, err)
		assert.Empty(t, m.items)
	})

	t.Run("returns error on missing file", func(t *testing.T) {
		t.Parallel()
		m := &mockDB{}
		svc := sources.New(m)
		err := svc.ImportSourceRegistry("/nonexistent/registry.json")
		assert.Error(t, err)
	})

	t.Run("returns error on invalid json", func(t *testing.T) {
		t.Parallel()
		tmp := t.TempDir()
		regPath := filepath.Join(tmp, "bad.json")
		require.NoError(t, os.WriteFile(regPath, []byte("{invalid"), 0644))

		m := &mockDB{}
		svc := sources.New(m)
		err := svc.ImportSourceRegistry(regPath)
		assert.Error(t, err)
	})
}

func TestInstaller_InstallUninstall(t *testing.T) {

	t.Run("install and uninstall skill via unified symlink", func(t *testing.T) {
		// Set up temp data dir — ALL types go to ~/.local/share/maestro/<type>/
		dataDir := t.TempDir()
		t.Setenv("OPENCODE_DATA_DIR", dataDir)
		require.NoError(t, os.MkdirAll(filepath.Join(dataDir, "skills"), 0755))

		// Create a fake source file to symlink
		sourceDir := t.TempDir()
		sourcePath := filepath.Join(sourceDir, "my-skill.md")
		require.NoError(t, os.WriteFile(sourcePath, []byte("# My Skill\n"), 0644))

		item := &models.SourceItem{
			ID:         "test-skill",
			Type:       "skill",
			SourcePath: sourcePath,
			SourceID:   "test-source",
			Status:     "active",
		}
		m := &mockDB{}
		m.UpsertSourceItem(item)
		inst := sources.NewInstaller(m)

		// Install
		err := inst.InstallItem(item)
		require.NoError(t, err)

		// Verify symlink is namespaced with sourceID under dataDir/skills
		// Name from item.ID ("test-skill") + extension (".md") = "test-skill.md"
		targetPath := filepath.Join(dataDir, "skills", "test-skill.md")
		info, err := os.Lstat(targetPath)
		require.NoError(t, err)
		assert.True(t, info.Mode()&os.ModeSymlink != 0, "should be a symlink")

		// Verify symlink target — may point to sanitized copy
		actualTarget, err := os.Readlink(targetPath)
		require.NoError(t, err)
		// The source file has no hardcoded paths so it points to original
		assert.Equal(t, sourcePath, actualTarget)

		// Verify status updated
		found := false
		for _, it := range m.items {
			if it.ID == item.ID && it.Status == "installed" {
				found = true
			}
		}
		assert.True(t, found, "item status should be installed")

		// Uninstall — uses targetPath (set by InstallItem)
		item.TargetPath = targetPath
		err = inst.UninstallItem(item)
		require.NoError(t, err)

		// Verify symlink removed
		_, err = os.Lstat(targetPath)
		assert.True(t, os.IsNotExist(err), "symlink should be removed")
	})

	t.Run("install agent with namespaced symlink", func(t *testing.T) {
		dataDir := t.TempDir()
		t.Setenv("OPENCODE_DATA_DIR", dataDir)

		sourceDir := t.TempDir()
		sourcePath := filepath.Join(sourceDir, "code-reviewer.md")
		require.NoError(t, os.WriteFile(sourcePath, []byte("# Code Reviewer\n"), 0644))

		item := &models.SourceItem{
			ID:         "agents-code-reviewer",
			Type:       "agent",
			SourcePath: sourcePath,
			SourceID:   "test-source",
			Status:     "active",
		}
		m := &mockDB{}
		require.NoError(t, os.MkdirAll(filepath.Join(dataDir, "agents"), 0755))

		inst := sources.NewInstaller(m)
		err := inst.InstallItem(item)
		require.NoError(t, err)

		// Name derived from item.ID + extension (the unique symlink name)
		// item.ID = "agents-code-reviewer", ext = ".md" → "agents-code-reviewer.md"
		targetPath := filepath.Join(dataDir, "agents", "agents-code-reviewer.md")
		info, err := os.Lstat(targetPath)
		require.NoError(t, err)
		assert.True(t, info.Mode()&os.ModeSymlink != 0)

		item.TargetPath = targetPath
		err = inst.UninstallItem(item)
		require.NoError(t, err)

		_, err = os.Lstat(targetPath)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("install unsupported type returns error", func(t *testing.T) {
		m := &mockDB{}
		inst := sources.NewInstaller(m)
		item := &models.SourceItem{
			ID:         "test-doc",
			Type:       "doc",
			SourcePath: "/tmp/test.md",
			SourceID:   "test",
		}
		err := inst.InstallItem(item)
		assert.ErrorContains(t, err, "unsupported item type")
	})

	t.Run("install missing source path returns error", func(t *testing.T) {
		m := &mockDB{}
		inst := sources.NewInstaller(m)
		item := &models.SourceItem{
			ID:   "test-skill",
			Type: "skill",
		}
		err := inst.InstallItem(item)
		assert.ErrorContains(t, err, "has no source path")
	})

	t.Run("InstallAll and UninstallAll with namespaced names", func(t *testing.T) {
		dataDir := t.TempDir()
		t.Setenv("OPENCODE_DATA_DIR", dataDir)
		require.NoError(t, os.MkdirAll(filepath.Join(dataDir, "skills"), 0755))
		require.NoError(t, os.MkdirAll(filepath.Join(dataDir, "agents"), 0755))
		sourceDir := t.TempDir()
		skillPath := filepath.Join(sourceDir, "skill-a.md")
		agentPath := filepath.Join(sourceDir, "agent-a.md")
		require.NoError(t, os.WriteFile(skillPath, []byte("# Skill A\n"), 0644))
		require.NoError(t, os.WriteFile(agentPath, []byte("# Agent A\n"), 0644))

		m := &mockDB{}
		_ = m.UpsertSourceItem(&models.SourceItem{
			ID: "s1-item1", SourceID: "source-1", Type: "skill",
			SourcePath: skillPath, Status: "active",
		})
		_ = m.UpsertSourceItem(&models.SourceItem{
			ID: "s1-item2", SourceID: "source-1", Type: "agent",
			SourcePath: agentPath, Status: "active",
		})

		inst := sources.NewInstaller(m)

		// InstallAll
		err := inst.InstallAll("source-1")
		require.NoError(t, err)

		// Names derived from item.ID + extension (unique names, not sourceID+barename)
		// s1-item1 (.md) → "s1-item1.md"
		_, err = os.Lstat(filepath.Join(dataDir, "skills", "s1-item1.md"))
		assert.NoError(t, err)
		// s1-item2 (.md) → "s1-item2.md"
		_, err = os.Lstat(filepath.Join(dataDir, "agents", "s1-item2.md"))
		assert.NoError(t, err)

		// Verify statuses updated
		items, err := m.ListSourceItemsBySource("source-1")
		require.NoError(t, err)
		for _, it := range items {
			assert.Equal(t, "installed", it.Status)
		}

		// UninstallAll
		err = inst.UninstallAll("source-1")
		require.NoError(t, err)

		// Verify removed (using item.ID + extension)
		_, err = os.Lstat(filepath.Join(dataDir, "skills", "s1-item1.md"))
		assert.True(t, os.IsNotExist(err))
		_, err = os.Lstat(filepath.Join(dataDir, "agents", "s1-item2.md"))
		assert.True(t, os.IsNotExist(err))
	})
}
