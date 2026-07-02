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

func seedProviders(t *testing.T, dbPath string) {
	t.Helper()
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	defer d.Close()
	for _, p := range []*models.Provider{
		{ID: "openai", Name: "OpenAI", Source: "discovered", Status: "active"},
		{ID: "anthropic", Name: "Anthropic", Source: "custom", Status: "active"},
	} {
		if err := d.UpsertProvider(p); err != nil {
			t.Fatalf("upsert provider %s: %v", p.ID, err)
		}
	}
}

func seedModels(t *testing.T, dbPath string) {
	t.Helper()
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	defer d.Close()
	for _, m := range []*models.Model{
		{ID: "gpt-4", ProviderID: "openai", DisplayName: "GPT-4", ContextWindow: 8192, Status: "active", Source: "discovered"},
		{ID: "claude-3", ProviderID: "anthropic", DisplayName: "Claude 3", ContextWindow: 100000, Status: "active", Source: "discovered"},
	} {
		if err := d.UpsertModel(m); err != nil {
			t.Fatalf("upsert model %s: %v", m.ID, err)
		}
	}
}

func seedCommands(t *testing.T, dbPath string) {
	t.Helper()
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	defer d.Close()
	if err := d.UpsertCommand(&models.Command{ID: "/test", Template: "# test\n", Description: "Test command", Status: "active"}); err != nil {
		t.Fatalf("upsert command: %v", err)
	}
}

func seedAgents(t *testing.T, dbPath string) {
	t.Helper()
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	defer d.Close()
	if err := d.UpsertAgent(&models.Agent{ID: "agent1", Description: "Agent One", Mode: "auto", Status: "active"}); err != nil {
		t.Fatalf("upsert agent: %v", err)
	}
}

// ---------------------------------------------------------------------------
// providers_cmd.go tests
// ---------------------------------------------------------------------------

func TestProvidersCmd_List_Empty(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, _ := db.Open(dbPath)
	d.Close()

	cmd := newProvidersCmd(&dbPath)
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	if err := listCmd.RunE(listCmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestProvidersCmd_List_WithData(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedProviders(t, dbPath)

	cmd := newProvidersCmd(&dbPath)
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	if err := listCmd.RunE(listCmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestProvidersCmd_Add(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, _ := db.Open(dbPath)
	d.Close()

	cmd := newProvidersCmd(&dbPath)
	addCmd, _, err := cmd.Find([]string{"add"})
	if err != nil {
		t.Fatal(err)
	}
	addCmd.SetArgs([]string{
		"--id", "test-provider",
		"--name", "Test Provider",
		"--api-base", "https://api.test.com",
		"--key-env", "TEST_API_KEY",
	})
	if err := addCmd.RunE(addCmd, nil); err != nil {
		t.Fatal(err)
	}

	// Verify via DB
	d2, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer d2.Close()
	p, err := d2.GetProvider("test-provider")
	if err != nil {
		t.Fatalf("provider not found after add: %v", err)
	}
	if p.Name != "Test Provider" {
		t.Errorf("name = %q, want %q", p.Name, "Test Provider")
	}
	if p.BaseURL != "https://api.test.com" {
		t.Errorf("base URL = %q, want %q", p.BaseURL, "https://api.test.com")
	}
	if p.KeyEnv != "TEST_API_KEY" {
		t.Errorf("key env = %q, want %q", p.KeyEnv, "TEST_API_KEY")
	}
}

func TestProvidersCmd_Add_MissingRequired(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, _ := db.Open(dbPath)
	d.Close()

	cmd := newProvidersCmd(&dbPath)
	addCmd, _, err := cmd.Find([]string{"add"})
	if err != nil {
		t.Fatal(err)
	}
	addCmd.SetArgs([]string{"--id", "partial"})
	if err := addCmd.RunE(addCmd, nil); err == nil {
		t.Fatal("expected error for missing required flags")
	}
}

func TestProvidersCmd_Update(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedProviders(t, dbPath)

	cmd := newProvidersCmd(&dbPath)
	updateCmd, _, err := cmd.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	updateCmd.SetArgs([]string{"--id", "openai", "--name", "OpenAI Updated"})
	if err := updateCmd.RunE(updateCmd, nil); err != nil {
		t.Fatal(err)
	}

	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	p, err := d.GetProvider("openai")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "OpenAI Updated" {
		t.Errorf("name = %q, want %q", p.Name, "OpenAI Updated")
	}
}

func TestProvidersCmd_Update_NoChanges(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedProviders(t, dbPath)

	cmd := newProvidersCmd(&dbPath)
	updateCmd, _, err := cmd.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	updateCmd.SetArgs([]string{"--id", "openai"})
	if err := updateCmd.RunE(updateCmd, nil); err == nil {
		t.Fatal("expected error for no changes")
	}
}

func TestProvidersCmd_Update_NonExistent(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, _ := db.Open(dbPath)
	d.Close()

	cmd := newProvidersCmd(&dbPath)
	updateCmd, _, err := cmd.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	updateCmd.SetArgs([]string{"--id", "ghost", "--name", "Ghost"})
	if err := updateCmd.RunE(updateCmd, nil); err == nil {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestProvidersCmd_Remove(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedProviders(t, dbPath)

	cmd := newProvidersCmd(&dbPath)
	removeCmd, _, err := cmd.Find([]string{"remove"})
	if err != nil {
		t.Fatal(err)
	}
	removeCmd.SetArgs([]string{"--id", "anthropic"})
	if err := removeCmd.RunE(removeCmd, nil); err != nil {
		t.Fatal(err)
	}

	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if _, err := d.GetProvider("anthropic"); err == nil {
		t.Fatal("expected provider to be deleted")
	}
}

func TestProvidersCmd_Remove_NonExistent(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, _ := db.Open(dbPath)
	d.Close()

	cmd := newProvidersCmd(&dbPath)
	removeCmd, _, err := cmd.Find([]string{"remove"})
	if err != nil {
		t.Fatal(err)
	}
	removeCmd.SetArgs([]string{"--id", "ghost"})
	if err := removeCmd.RunE(removeCmd, nil); err == nil {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestProvidersCmd_Remove_MissingID(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, _ := db.Open(dbPath)
	d.Close()

	cmd := newProvidersCmd(&dbPath)
	removeCmd, _, err := cmd.Find([]string{"remove"})
	if err != nil {
		t.Fatal(err)
	}
	if err := removeCmd.RunE(removeCmd, nil); err == nil {
		t.Fatal("expected error for missing --id")
	}
}

// ---------------------------------------------------------------------------
// models_cmd.go tests
// ---------------------------------------------------------------------------

func TestModelsCmd_List_Empty(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, _ := db.Open(dbPath)
	d.Close()

	cmd := newModelsCmd(&dbPath)
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	if err := listCmd.RunE(listCmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestModelsCmd_List_WithData(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedModels(t, dbPath)

	cmd := newModelsCmd(&dbPath)
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	if err := listCmd.RunE(listCmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestModelsCmd_List_Paid(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedModels(t, dbPath)

	cmd := newModelsCmd(&dbPath)
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	listCmd.SetArgs([]string{"--paid"})
	if err := listCmd.RunE(listCmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestModelsCmd_Search_Found(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedModels(t, dbPath)

	cmd := newModelsCmd(&dbPath)
	searchCmd, _, err := cmd.Find([]string{"search"})
	if err != nil {
		t.Fatal(err)
	}
	if err := searchCmd.RunE(searchCmd, []string{"gpt"}); err != nil {
		t.Fatal(err)
	}
}

func TestModelsCmd_Search_NotFound(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, _ := db.Open(dbPath)
	d.Close()

	cmd := newModelsCmd(&dbPath)
	searchCmd, _, err := cmd.Find([]string{"search"})
	if err != nil {
		t.Fatal(err)
	}
	if err := searchCmd.RunE(searchCmd, []string{"nonexistent"}); err != nil {
		t.Fatal(err)
	}
}

func TestModelsCmd_Search_MissingArgs(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, _ := db.Open(dbPath)
	d.Close()

	cmd := newModelsCmd(&dbPath)
	searchCmd, _, err := cmd.Find([]string{"search"})
	if err != nil {
		t.Fatal(err)
	}
	if err := searchCmd.RunE(searchCmd, nil); err == nil {
		t.Fatal("expected error for missing args")
	}
}

func TestModelsCmd_Info(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedModels(t, dbPath)

	cmd := newModelsCmd(&dbPath)
	infoCmd, _, err := cmd.Find([]string{"info"})
	if err != nil {
		t.Fatal(err)
	}
	if err := infoCmd.RunE(infoCmd, []string{"gpt-4"}); err != nil {
		t.Fatal(err)
	}
}

func TestModelsCmd_Info_NonExistent(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, _ := db.Open(dbPath)
	d.Close()

	cmd := newModelsCmd(&dbPath)
	infoCmd, _, err := cmd.Find([]string{"info"})
	if err != nil {
		t.Fatal(err)
	}
	if err := infoCmd.RunE(infoCmd, []string{"ghost-model"}); err == nil {
		t.Fatal("expected error for nonexistent model")
	}
}

func TestModelsCmd_Add(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedProviders(t, dbPath)

	cmd := newModelsCmd(&dbPath)
	addCmd, _, err := cmd.Find([]string{"add"})
	if err != nil {
		t.Fatal(err)
	}
	addCmd.SetArgs([]string{
		"--id", "openai/gpt-5",
		"--provider-id", "openai",
		"--display-name", "GPT-5",
		"--context", "16384",
		"--function-calling",
		"--vision",
	})
	if err := addCmd.RunE(addCmd, nil); err != nil {
		t.Fatal(err)
	}

	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	m, err := d.GetModel("openai/gpt-5")
	if err != nil {
		t.Fatalf("model not found after add: %v", err)
	}
	if m.DisplayName != "GPT-5" {
		t.Errorf("display name = %q, want %q", m.DisplayName, "GPT-5")
	}
	if m.ContextWindow != 16384 {
		t.Errorf("context window = %d, want 16384", m.ContextWindow)
	}
	if !m.FunctionCalling {
		t.Error("function calling should be true")
	}
	if !m.Vision {
		t.Error("vision should be true")
	}
}

func TestModelsCmd_Add_MissingRequired(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, _ := db.Open(dbPath)
	d.Close()

	cmd := newModelsCmd(&dbPath)
	addCmd, _, err := cmd.Find([]string{"add"})
	if err != nil {
		t.Fatal(err)
	}
	// Missing --provider-id
	addCmd.SetArgs([]string{"--id", "orphan"})
	if err := addCmd.RunE(addCmd, nil); err == nil {
		t.Fatal("expected error for missing --provider-id")
	}
}

func TestModelsCmd_Update(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedModels(t, dbPath)

	cmd := newModelsCmd(&dbPath)
	updateCmd, _, err := cmd.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	updateCmd.SetArgs([]string{"--id", "gpt-4", "--display-name", "GPT-4 Turbo"})
	if err := updateCmd.RunE(updateCmd, nil); err != nil {
		t.Fatal(err)
	}

	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	m, err := d.GetModel("gpt-4")
	if err != nil {
		t.Fatal(err)
	}
	if m.DisplayName != "GPT-4 Turbo" {
		t.Errorf("display name = %q, want %q", m.DisplayName, "GPT-4 Turbo")
	}
}

func TestModelsCmd_Update_NonExistent(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, _ := db.Open(dbPath)
	d.Close()

	cmd := newModelsCmd(&dbPath)
	updateCmd, _, err := cmd.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	updateCmd.SetArgs([]string{"--id", "ghost", "--display-name", "Ghost"})
	if err := updateCmd.RunE(updateCmd, nil); err == nil {
		t.Fatal("expected error for nonexistent model")
	}
}

func TestModelsCmd_Update_NoChanges(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedModels(t, dbPath)

	cmd := newModelsCmd(&dbPath)
	updateCmd, _, err := cmd.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	updateCmd.SetArgs([]string{"--id", "gpt-4"})
	if err := updateCmd.RunE(updateCmd, nil); err == nil {
		t.Fatal("expected error for no changes")
	}
}

func TestModelsCmd_Remove(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedModels(t, dbPath)

	cmd := newModelsCmd(&dbPath)
	removeCmd, _, err := cmd.Find([]string{"remove"})
	if err != nil {
		t.Fatal(err)
	}
	removeCmd.SetArgs([]string{"--id", "gpt-4"})
	if err := removeCmd.RunE(removeCmd, nil); err != nil {
		t.Fatal(err)
	}

	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if _, err := d.GetModel("gpt-4"); err == nil {
		t.Fatal("expected model to be deleted")
	}
}

func TestModelsCmd_Remove_NonExistent(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, _ := db.Open(dbPath)
	d.Close()

	cmd := newModelsCmd(&dbPath)
	removeCmd, _, err := cmd.Find([]string{"remove"})
	if err != nil {
		t.Fatal(err)
	}
	removeCmd.SetArgs([]string{"--id", "ghost"})
	if err := removeCmd.RunE(removeCmd, nil); err == nil {
		t.Fatal("expected error for nonexistent model")
	}
}

func TestModelsCmd_Classify(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedProviders(t, dbPath)
	seedModels(t, dbPath)

	cmd := newModelsCmd(&dbPath)
	classifyCmd, _, err := cmd.Find([]string{"classify"})
	if err != nil {
		t.Fatal(err)
	}
	if err := classifyCmd.RunE(classifyCmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestModelsCmd_Classify_ByProvider(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedProviders(t, dbPath)
	seedModels(t, dbPath)

	cmd := newModelsCmd(&dbPath)
	classifyCmd, _, err := cmd.Find([]string{"classify"})
	if err != nil {
		t.Fatal(err)
	}
	classifyCmd.SetArgs([]string{"--provider", "openai"})
	if err := classifyCmd.RunE(classifyCmd, nil); err != nil {
		t.Fatal(err)
	}
}

// ---------------------------------------------------------------------------
// commands_cmd.go tests
// ---------------------------------------------------------------------------

func TestCommandsCmd_List_Empty(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, _ := db.Open(dbPath)
	d.Close()

	cmd := newCommandsCmd(&dbPath)
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	if err := listCmd.RunE(listCmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestCommandsCmd_List_WithData(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedCommands(t, dbPath)

	cmd := newCommandsCmd(&dbPath)
	listCmd, _, err := cmd.Find([]string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	if err := listCmd.RunE(listCmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestCommandsCmd_Add(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, _ := db.Open(dbPath)
	d.Close()

	cmd := newCommandsCmd(&dbPath)
	addCmd, _, err := cmd.Find([]string{"add"})
	if err != nil {
		t.Fatal(err)
	}
	addCmd.SetArgs([]string{
		"--id", "/my-command",
		"--template", "Do something with {input}",
		"--description", "My custom command",
		"--agent", "coder",
	})
	if err := addCmd.RunE(addCmd, nil); err != nil {
		t.Fatal(err)
	}

	d2, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer d2.Close()
	commands, err := d2.ListCommands()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range commands {
		if c.ID == "/my-command" {
			found = true
			if c.Template != "Do something with {input}" {
				t.Errorf("template = %q, want %q", c.Template, "Do something with {input}")
			}
			if c.Description != "My custom command" {
				t.Errorf("description = %q, want %q", c.Description, "My custom command")
			}
			if c.Agent != "coder" {
				t.Errorf("agent = %q, want %q", c.Agent, "coder")
			}
			break
		}
	}
	if !found {
		t.Fatal("command not found after add")
	}
}

func TestCommandsCmd_Add_MissingRequired(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, _ := db.Open(dbPath)
	d.Close()

	cmd := newCommandsCmd(&dbPath)
	addCmd, _, err := cmd.Find([]string{"add"})
	if err != nil {
		t.Fatal(err)
	}
	addCmd.SetArgs([]string{"--id", "/incomplete"})
	if err := addCmd.RunE(addCmd, nil); err == nil {
		t.Fatal("expected error for missing --template")
	}
}

func TestCommandsCmd_Update(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedCommands(t, dbPath)

	cmd := newCommandsCmd(&dbPath)
	updateCmd, _, err := cmd.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	updateCmd.SetArgs([]string{"--id", "/test", "--description", "Updated description"})
	if err := updateCmd.RunE(updateCmd, nil); err != nil {
		t.Fatal(err)
	}

	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	commands, err := d.ListCommands()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range commands {
		if c.ID == "/test" {
			found = true
			if c.Description != "Updated description" {
				t.Errorf("description = %q, want %q", c.Description, "Updated description")
			}
			break
		}
	}
	if !found {
		t.Fatal("command not found after update")
	}
}

func TestCommandsCmd_Update_NonExistent(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, _ := db.Open(dbPath)
	d.Close()

	cmd := newCommandsCmd(&dbPath)
	updateCmd, _, err := cmd.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	updateCmd.SetArgs([]string{"--id", "/ghost", "--description", "Ghost"})
	if err := updateCmd.RunE(updateCmd, nil); err == nil {
		t.Fatal("expected error for nonexistent command")
	}
}

func TestCommandsCmd_Remove(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedCommands(t, dbPath)

	cmd := newCommandsCmd(&dbPath)
	removeCmd, _, err := cmd.Find([]string{"remove"})
	if err != nil {
		t.Fatal(err)
	}
	removeCmd.SetArgs([]string{"--id", "/test"})
	if err := removeCmd.RunE(removeCmd, nil); err != nil {
		t.Fatal(err)
	}

	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	commands, err := d.ListCommands()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range commands {
		if c.ID == "/test" {
			t.Fatal("expected command to be deleted")
		}
	}
}

func TestCommandsCmd_Remove_NonExistent(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, _ := db.Open(dbPath)
	d.Close()

	cmd := newCommandsCmd(&dbPath)
	removeCmd, _, err := cmd.Find([]string{"remove"})
	if err != nil {
		t.Fatal(err)
	}
	removeCmd.SetArgs([]string{"--id", "/ghost"})
	if err := removeCmd.RunE(removeCmd, nil); err == nil {
		t.Fatal("expected error for nonexistent command")
	}
}

// ---------------------------------------------------------------------------
// agents_cmd.go tests (add / update)
// ---------------------------------------------------------------------------

func TestAgentsCmd_Add(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, _ := db.Open(dbPath)
	d.Close()

	cmd := newAgentsCmd(&dbPath)
	addCmd, _, err := cmd.Find([]string{"add"})
	if err != nil {
		t.Fatal(err)
	}
	addCmd.SetArgs([]string{
		"--id", "new-agent",
		"--task-type", "coding",
		"--description", "A new coding agent",
		"--model", "gpt-4",
		"--mode", "subagent",
		"--color", "blue",
	})
	if err := addCmd.RunE(addCmd, nil); err != nil {
		t.Fatal(err)
	}

	d2, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer d2.Close()
	a, err := d2.GetAgent("new-agent")
	if err != nil {
		t.Fatalf("agent not found after add: %v", err)
	}
	if a.Description != "A new coding agent" {
		t.Errorf("description = %q, want %q", a.Description, "A new coding agent")
	}
	if a.CurrentModelID != "gpt-4" {
		t.Errorf("model = %q, want %q", a.CurrentModelID, "gpt-4")
	}
	if a.Mode != "subagent" {
		t.Errorf("mode = %q, want %q", a.Mode, "subagent")
	}
	if a.Color != "blue" {
		t.Errorf("color = %q, want %q", a.Color, "blue")
	}
}

func TestAgentsCmd_Add_MissingID(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, _ := db.Open(dbPath)
	d.Close()

	cmd := newAgentsCmd(&dbPath)
	addCmd, _, err := cmd.Find([]string{"add"})
	if err != nil {
		t.Fatal(err)
	}
	if err := addCmd.RunE(addCmd, nil); err == nil {
		t.Fatal("expected error for missing --id")
	}
}

func TestAgentsCmd_Update(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedAgents(t, dbPath)

	cmd := newAgentsCmd(&dbPath)
	updateCmd, _, err := cmd.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	updateCmd.SetArgs([]string{
		"--id", "agent1",
		"--description", "Updated Agent",
		"--model", "claude-3",
		"--mode", "primary",
	})
	if err := updateCmd.RunE(updateCmd, nil); err != nil {
		t.Fatal(err)
	}

	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	a, err := d.GetAgent("agent1")
	if err != nil {
		t.Fatal(err)
	}
	if a.Description != "Updated Agent" {
		t.Errorf("description = %q, want %q", a.Description, "Updated Agent")
	}
	if a.CurrentModelID != "claude-3" {
		t.Errorf("model = %q, want %q", a.CurrentModelID, "claude-3")
	}
	if a.Mode != "primary" {
		t.Errorf("mode = %q, want %q", a.Mode, "primary")
	}
}

func TestAgentsCmd_Update_NonExistent(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, _ := db.Open(dbPath)
	d.Close()

	cmd := newAgentsCmd(&dbPath)
	updateCmd, _, err := cmd.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	updateCmd.SetArgs([]string{"--id", "ghost", "--description", "Ghost"})
	if err := updateCmd.RunE(updateCmd, nil); err == nil {
		t.Fatal("expected error for nonexistent agent")
	}
}

func TestAgentsCmd_Update_NoChanges(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	seedAgents(t, dbPath)

	cmd := newAgentsCmd(&dbPath)
	updateCmd, _, err := cmd.Find([]string{"update"})
	if err != nil {
		t.Fatal(err)
	}
	updateCmd.SetArgs([]string{"--id", "agent1"})
	if err := updateCmd.RunE(updateCmd, nil); err == nil {
		t.Fatal("expected error for no changes")
	}
}
