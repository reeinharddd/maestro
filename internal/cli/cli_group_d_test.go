package cli

import (
	"path/filepath"
	"testing"

	"github.com/reeinharrrd/maestro/internal/db"
	"github.com/reeinharrrd/maestro/pkg/models"
)

// ---------------------------------------------------------------------------
// Seed helpers
// ---------------------------------------------------------------------------

func seedSkills(t *testing.T, dbPath string) {
	t.Helper()
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	defer d.Close()
	for _, s := range []*models.Skill{
		{ID: "coder", Source: "registry", Type: "skill", Status: "active"},
		{ID: "reviewer", Source: "manual", Type: "agent", Status: "active"},
		{ID: "triage-script", Source: "filesystem", Type: "command", Status: "inactive"},
	} {
		if err := d.UpsertSkill(s); err != nil {
			t.Fatalf("upsert skill %s: %v", s.ID, err)
		}
	}
}

func seedMCPServers(t *testing.T, dbPath string) {
	t.Helper()
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	defer d.Close()
	for _, s := range []*models.MCPServer{
		{ID: "filesystem", Type: "local", Command: `["npx","-y","@modelcontextprotocol/server-filesystem"]`, Enabled: true, Timeout: 60000, Source: "manual"},
		{ID: "context7", Type: "remote", URL: "http://localhost:3000/mcp", Enabled: false, Timeout: 30000, Source: "manual"},
	} {
		if err := d.UpsertMCP(s); err != nil {
			t.Fatalf("upsert mcp %s: %v", s.ID, err)
		}
	}
}

func seedLSPServers(t *testing.T, dbPath string) {
	t.Helper()
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	defer d.Close()
	for _, l := range []*models.LSPServer{
		{ID: "gopls", Command: `["gopls"]`, Extensions: `[".go"]`},
		{ID: "typescript", Command: `["typescript-language-server","--stdio"]`, Extensions: `[".ts",".tsx"]`, Disabled: true},
	} {
		if err := d.UpsertLSPServer(l); err != nil {
			t.Fatalf("upsert lsp %s: %v", l.ID, err)
		}
	}
}

func seedExecLogs(t *testing.T, dbPath string) {
	t.Helper()
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	defer d.Close()
	for _, l := range []*models.ExecLog{
		{Agent: "agent1", Model: "gpt-4", Task: "code review", TokensIn: 500, TokensOut: 200, DurationMs: 3000, Success: true},
		{Agent: "agent2", Model: "claude-3", Task: "refactor", TokensIn: 1000, TokensOut: 500, DurationMs: 5000, Success: true},
		{Agent: "agent1", Model: "gpt-4", Task: "test", TokensIn: 100, TokensOut: 50, DurationMs: 1000, Success: false, Error: "timeout"},
	} {
		if err := d.InsertExecLog(l); err != nil {
			t.Fatalf("insert exec log: %v", err)
		}
	}
}

func seedModelProfiles(t *testing.T, dbPath string) {
	t.Helper()
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	defer d.Close()
	for _, p := range []*models.ModelProfile{
		{ModelID: "gpt-4", RealContext: 8192, MaxOutput: 4096, SupportsStream: true, SupportsSO: true, StreamTPS: 45.2, ProfiledAt: 1700000000},
		{ModelID: "claude-3", RealContext: 100000, MaxOutput: 8192, SupportsStream: true, SupportsSO: false, StreamTPS: 32.1, ProfiledAt: 1700000001},
	} {
		if err := d.UpsertModelProfile(p); err != nil {
			t.Fatalf("upsert profile %s: %v", p.ModelID, err)
		}
	}
}

func seedModelsForView(t *testing.T, dbPath string) {
	t.Helper()
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	defer d.Close()
	for _, m := range []*models.Model{
		{ID: "gpt-4", ProviderID: "openai", DisplayName: "GPT-4", ContextWindow: 8192, FunctionCalling: true, Vision: false, Tier: "standard", Status: "active", Source: "discovered"},
		{ID: "claude-3", ProviderID: "anthropic", DisplayName: "Claude 3", ContextWindow: 100000, FunctionCalling: true, Vision: true, Tier: "premium", Status: "active", Source: "discovered"},
	} {
		if err := d.UpsertModel(m); err != nil {
			t.Fatalf("upsert model %s: %v", m.ID, err)
		}
	}
}

// seedMinSync seeds the minimum required data so syncConfig (called by add/update/remove)
// does not fail. Based on validate test seed pattern.
func seedMinSync(t *testing.T, dbPath string) {
	t.Helper()
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	defer d.Close()
	d.UpsertProvider(&models.Provider{ID: "p", Name: "P", Source: "custom", Status: "active"})
	d.UpsertModel(&models.Model{ID: "m", ProviderID: "p", DisplayName: "M", Status: "active", Source: "discovered"})
}

// ============================================================================
// Skills
// ============================================================================

func TestSkillsList_Empty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newSkillsCmd(&dbPath)
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	if err := listCmd.RunE(listCmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestSkillsList_PrintsAll(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedSkills(t, dbPath)

	cmd := newSkillsCmd(&dbPath)
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	if err := listCmd.RunE(listCmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestSkillsReport_ShowsSummary(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedSkills(t, dbPath)

	cmd := newSkillsCmd(&dbPath)
	reportCmd, _, err := cmd.Find([]string{"report"})
	if err != nil {
		t.Fatal(err)
	}
	if err := reportCmd.RunE(reportCmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestSkillsAdd_AddsSkill(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedMinSync(t, dbPath)

	cmd := newSkillAddCmd(&dbPath)
	cmd.Flags().String("id", "", "")
	cmd.Flags().String("source", "", "")
	cmd.Flags().String("type", "", "")
	cmd.Flags().String("source-path", "", "")
	cmd.Flags().String("target-path", "", "")
	cmd.Flags().String("status", "active", "")
	cmd.Flags().Set("id", "my-skill")
	cmd.Flags().Set("source", "manual")
	cmd.Flags().Set("type", "skill")

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}

	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	skills, err := d.ListSkills()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range skills {
		if s.ID == "my-skill" && s.Source == "manual" && s.Type == "skill" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("skill not found after add")
	}
}

func TestSkillsAdd_MissingFlags(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newSkillAddCmd(&dbPath)
	cmd.Flags().String("id", "", "")
	cmd.Flags().String("source", "", "")
	cmd.Flags().String("type", "", "")
	// No flags set — should error

	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("expected error for missing flags")
	}
}

func TestSkillsUpdate_UpdatesSkill(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedSkills(t, dbPath)
	seedMinSync(t, dbPath)

	cmd := newSkillUpdateCmd(&dbPath)
	cmd.Flags().String("id", "", "")
	cmd.Flags().String("source", "", "")
	cmd.Flags().String("status", "", "")
	cmd.Flags().Set("id", "coder")
	cmd.Flags().Set("source", "updated-source")
	cmd.Flags().Set("status", "inactive")

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}

	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	skills, err := d.ListSkills()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range skills {
		if s.ID == "coder" {
			if s.Source != "updated-source" || s.Status != "inactive" {
				t.Fatalf("skill not updated: %+v", s)
			}
			return
		}
	}
	t.Fatal("skill coder not found")
}

func TestSkillsUpdate_NoFlags(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedSkills(t, dbPath)

	cmd := newSkillUpdateCmd(&dbPath)
	cmd.Flags().String("id", "", "")
	cmd.Flags().Set("id", "coder")
	// No other flags changed — should error

	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("expected error when no flags changed")
	}
}

func TestSkillsUpdate_NotFound(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newSkillUpdateCmd(&dbPath)
	cmd.Flags().String("id", "", "")
	cmd.Flags().String("source", "", "")
	cmd.Flags().Set("id", "nonexistent")
	cmd.Flags().Set("source", "val")

	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("expected error for nonexistent skill")
	}
}

func TestSkillsRemove_RemovesSkill(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedSkills(t, dbPath)
	seedMinSync(t, dbPath)

	cmd := newSkillRemoveCmd(&dbPath)
	cmd.Flags().String("id", "", "")
	cmd.Flags().Set("id", "triage-script")

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}

	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if _, err := d.ListSkills(); err != nil {
		t.Fatal(err)
	}
	skills, _ := d.ListSkills()
	for _, s := range skills {
		if s.ID == "triage-script" {
			t.Fatal("skill was not removed")
		}
	}
}

func TestSkillsRemove_MissingFlags(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newSkillRemoveCmd(&dbPath)
	cmd.Flags().String("id", "", "")
	// No id set

	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("expected error for missing --id")
	}
}

func TestSkillsRemove_NotFound(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedSkills(t, dbPath)

	cmd := newSkillRemoveCmd(&dbPath)
	cmd.Flags().String("id", "", "")
	cmd.Flags().Set("id", "nonexistent")

	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("expected error for nonexistent skill")
	}
}

// ============================================================================
// MCP Servers
// ============================================================================

func TestMCPServersList_Empty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newMCPServersCmd(&dbPath)
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	if err := listCmd.RunE(listCmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestMCPServersList_PrintsAll(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedMCPServers(t, dbPath)

	cmd := newMCPServersCmd(&dbPath)
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	if err := listCmd.RunE(listCmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestMCPServersAdd_Stdio(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedMinSync(t, dbPath)

	cmd := newMCPServerAddCmd(&dbPath)
	cmd.Flags().String("id", "", "")
	cmd.Flags().String("type", "", "")
	cmd.Flags().String("command", "", "")
	cmd.Flags().String("url", "", "")
	cmd.Flags().Bool("enabled", true, "")
	cmd.Flags().String("env-vars", "", "")
	cmd.Flags().Int("timeout", 60, "")
	cmd.Flags().Set("id", "my-server")
	cmd.Flags().Set("type", "stdio")
	cmd.Flags().Set("command", `["npx","-y","my-pkg"]`)

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}

	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	servers, err := d.ListMCPs()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range servers {
		if s.ID == "my-server" && s.Type == "local" && s.Command == `["npx","-y","my-pkg"]` {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("MCP server not found after add")
	}
}

func TestMCPServersAdd_URL(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedMinSync(t, dbPath)

	cmd := newMCPServerAddCmd(&dbPath)
	cmd.Flags().String("id", "", "")
	cmd.Flags().String("type", "", "")
	cmd.Flags().String("command", "", "")
	cmd.Flags().String("url", "", "")
	cmd.Flags().Bool("enabled", true, "")
	cmd.Flags().String("env-vars", "", "")
	cmd.Flags().Int("timeout", 30, "")
	cmd.Flags().Set("id", "remote-server")
	cmd.Flags().Set("type", "url")
	cmd.Flags().Set("url", "http://localhost:9090/mcp")

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}

	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	servers, err := d.ListMCPs()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range servers {
		if s.ID == "remote-server" && s.Type == "remote" && s.URL == "http://localhost:9090/mcp" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("MCP server not found after add")
	}
}

func TestMCPServersAdd_MissingFlags(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newMCPServerAddCmd(&dbPath)
	cmd.Flags().String("id", "", "")
	cmd.Flags().String("type", "", "")
	cmd.Flags().String("command", "", "")
	cmd.Flags().String("url", "", "")
	cmd.Flags().Bool("enabled", true, "")
	cmd.Flags().String("env-vars", "", "")
	cmd.Flags().Int("timeout", 60, "")
	// Missing id and type

	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("expected error for missing flags")
	}
}

func TestMCPServersUpdate_UpdatesServer(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedMCPServers(t, dbPath)
	seedMinSync(t, dbPath)

	cmd := newMCPServerUpdateCmd(&dbPath)
	cmd.Flags().String("id", "", "")
	cmd.Flags().String("type", "", "")
	cmd.Flags().String("command", "", "")
	cmd.Flags().String("url", "", "")
	cmd.Flags().Bool("enabled", true, "")
	cmd.Flags().String("env-vars", "", "")
	cmd.Flags().Int("timeout", 120, "")
	cmd.Flags().Set("id", "filesystem")
	cmd.Flags().Set("timeout", "120")

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}

	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	s, err := d.ListMCPs()
	if err != nil {
		t.Fatal(err)
	}
	for _, sv := range s {
		if sv.ID == "filesystem" && sv.Timeout == 120000 {
			return
		}
	}
	t.Fatal("MCP server not updated")
}

func TestMCPServersUpdate_NotFound(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newMCPServerUpdateCmd(&dbPath)
	cmd.Flags().String("id", "", "")
	cmd.Flags().Set("id", "nonexistent")
	cmd.Flags().Int("timeout", 60, "")
	cmd.Flags().Set("timeout", "30")

	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("expected error for nonexistent MCP server")
	}
}

func TestMCPServersRemove_RemovesServer(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedMCPServers(t, dbPath)
	seedMinSync(t, dbPath)

	cmd := newMCPServerRemoveCmd(&dbPath)
	cmd.Flags().String("id", "", "")
	cmd.Flags().Set("id", "context7")

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}

	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	servers, err := d.ListMCPs()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range servers {
		if s.ID == "context7" {
			t.Fatal("MCP server was not removed")
		}
	}
}

func TestMCPServersRemove_NotFound(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedMCPServers(t, dbPath)

	cmd := newMCPServerRemoveCmd(&dbPath)
	cmd.Flags().String("id", "", "")
	cmd.Flags().Set("id", "nonexistent")

	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("expected error for nonexistent MCP server")
	}
}

// ============================================================================
// Budget
// ============================================================================

func TestBudgetShow_Default(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	// Seed a budget so GetBudget doesn't error
	if err := d.UpsertBudget(&models.BudgetConfig{
		ID: "default", DailyGlobalUSD: 5.0, PreferredTier: "budget",
	}); err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newBudgetCmd(&dbPath)
	showCmd, _, err := cmd.Find([]string{"show"})
	if err != nil {
		t.Fatal(err)
	}
	if err := showCmd.RunE(showCmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestBudgetSet_SetsValues(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertBudget(&models.BudgetConfig{
		ID: "default", DailyGlobalUSD: 5.0, PreferredTier: "budget",
	}); err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newBudgetCmd(&dbPath)
	setCmd, _, err := cmd.Find([]string{"set"})
	if err != nil {
		t.Fatal(err)
	}
	setCmd.Flags().Set("daily", "10.0")
	setCmd.Flags().Set("tier", "quality")

	if err := setCmd.RunE(setCmd, nil); err != nil {
		t.Fatal(err)
	}

	d, err = db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	b, err := d.GetBudget()
	if err != nil {
		t.Fatal(err)
	}
	if b.DailyGlobalUSD != 10.0 {
		t.Fatalf("expected daily 10.0, got %.2f", b.DailyGlobalUSD)
	}
	if b.PreferredTier != "quality" {
		t.Fatalf("expected tier quality, got %s", b.PreferredTier)
	}
}

// ============================================================================
// LSP Servers
// ============================================================================

func TestLSPServersList_Empty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newLSPServersCmd(&dbPath)
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	if err := listCmd.RunE(listCmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestLSPServersList_PrintsAll(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedLSPServers(t, dbPath)

	cmd := newLSPServersCmd(&dbPath)
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	if err := listCmd.RunE(listCmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestLSPServersAdd_AddsServer(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newLSPAddCmd(&dbPath)
	cmd.Flags().String("id", "", "")
	cmd.Flags().String("command", "", "")
	cmd.Flags().String("extensions", "", "")
	cmd.Flags().String("env", "", "")
	cmd.Flags().String("init", "", "")
	cmd.Flags().Bool("disabled", false, "")
	cmd.Flags().Set("id", "rust-analyzer")
	cmd.Flags().Set("command", `["rust-analyzer"]`)
	cmd.Flags().Set("extensions", `[".rs"]`)

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}

	d, err = db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	s, err := d.GetLSPServer("rust-analyzer")
	if err != nil {
		t.Fatal(err)
	}
	if s.ID != "rust-analyzer" || s.Command != `["rust-analyzer"]` {
		t.Fatalf("LSP server not correct: %+v", s)
	}
}

func TestLSPServersAdd_MissingFlags(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newLSPAddCmd(&dbPath)
	cmd.Flags().String("id", "", "")
	cmd.Flags().String("command", "", "")
	// Missing flag values

	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("expected error for missing flags")
	}
}

func TestLSPServersUpdate_UpdatesServer(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedLSPServers(t, dbPath)

	cmd := newLSPUpdateCmd(&dbPath)
	cmd.Flags().String("id", "", "")
	cmd.Flags().String("command", "", "")
	cmd.Flags().String("extensions", "", "")
	cmd.Flags().Bool("disabled", false, "")
	cmd.Flags().Set("id", "gopls")
	cmd.Flags().Set("extensions", `[".go",".proto"]`)
	cmd.Flags().Set("disabled", "true")

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}

	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	s, err := d.GetLSPServer("gopls")
	if err != nil {
		t.Fatal(err)
	}
	if s.Extensions != `[".go",".proto"]` || !s.Disabled {
		t.Fatalf("LSP server not updated: %+v", s)
	}
}

func TestLSPServersRemove_RemovesServer(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedLSPServers(t, dbPath)

	cmd := newLSPRemoveCmd(&dbPath)
	cmd.Flags().String("id", "", "")
	cmd.Flags().Set("id", "typescript")

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}

	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if _, err := d.GetLSPServer("typescript"); err == nil {
		t.Fatal("LSP server was not removed")
	}
}

// ============================================================================
// Exec Log
// ============================================================================

func TestExecLogList_Empty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newExecLogCmd(&dbPath)
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	if err := listCmd.RunE(listCmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestExecLogList_PrintsAll(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedExecLogs(t, dbPath)

	cmd := newExecLogCmd(&dbPath)
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	if err := listCmd.RunE(listCmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestExecLogList_Limit(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedExecLogs(t, dbPath)

	cmd := newExecLogCmd(&dbPath)
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	listCmd.Flags().Set("limit", "2")

	if err := listCmd.RunE(listCmd, nil); err != nil {
		t.Fatal(err)
	}
}

// ============================================================================
// Model Profiles
// ============================================================================

func TestModelProfilesList_Empty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newModelProfilesCmd(&dbPath)
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	if err := listCmd.RunE(listCmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestModelProfilesList_PrintsAll(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedProviders(t, dbPath)
	seedModelsForView(t, dbPath)
	seedModelProfiles(t, dbPath)

	cmd := newModelProfilesCmd(&dbPath)
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	if err := listCmd.RunE(listCmd, nil); err != nil {
		t.Fatal(err)
	}
}

// ============================================================================
// Models View
// ============================================================================

func TestModelsView_Empty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newModelsViewCmd(&dbPath)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestModelsView_PrintsAll(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedProviders(t, dbPath)
	seedModelsForView(t, dbPath)

	cmd := newModelsViewCmd(&dbPath)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
}
