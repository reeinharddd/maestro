package db_test

import (
	"testing"
	"github.com/reeinharrrd/opencode-kit/internal/db"
	"github.com/reeinharrrd/opencode-kit/pkg/models"
)

func openTestDB(t *testing.T) *db.DB {
	t.Helper()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func TestOpen_InMemory(t *testing.T) {
	d := openTestDB(t)
	if d == nil {
		t.Fatal("expected non-nil db")
	}
}

func TestUpsertProvider(t *testing.T) {
	d := openTestDB(t)
	p := &models.Provider{
		ID:       "test-provider",
		Name:     "Test Provider",
		BaseURL:  "https://test.com",
		KeyEnv:   "TEST_KEY",
		Source:   "custom",
		Status:   "active",
		Priority: 50,
	}
	if err := d.UpsertProvider(p); err != nil {
		t.Fatalf("upsert provider: %v", err)
	}
	got, err := d.GetProvider("test-provider")
	if err != nil {
		t.Fatalf("get provider: %v", err)
	}
	if got.Name != "Test Provider" {
		t.Errorf("got name %q, want %q", got.Name, "Test Provider")
	}
	if got.BaseURL != "https://test.com" {
		t.Errorf("got base URL %q", got.BaseURL)
	}
}

func TestUpsertProvider_Duplicate(t *testing.T) {
	d := openTestDB(t)
	p := &models.Provider{ID: "dup", Name: "Original", BaseURL: "https://orig.com", Source: "custom", Status: "active"}
	if err := d.UpsertProvider(p); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	p2 := &models.Provider{ID: "dup", Name: "Updated", BaseURL: "https://updated.com", Source: "custom", Status: "active"}
	if err := d.UpsertProvider(p2); err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	got, err := d.GetProvider("dup")
	if err != nil {
		t.Fatalf("get provider: %v", err)
	}
	if got.Name != "Updated" {
		t.Errorf("got name %q, want %q", got.Name, "Updated")
	}
}

func TestGetProvider_NotFound(t *testing.T) {
	d := openTestDB(t)
	_, err := d.GetProvider("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent provider")
	}
}

func TestListProviders(t *testing.T) {
	d := openTestDB(t)
	providers := []*models.Provider{
		{ID: "p1", Name: "Provider 1", Source: "custom", Status: "active"},
		{ID: "p2", Name: "Provider 2", Source: "custom", Status: "active"},
	}
	for _, p := range providers {
		if err := d.UpsertProvider(p); err != nil {
			t.Fatalf("upsert: %v", err)
		}
	}
	list, err := d.ListProviders()
	if err != nil {
		t.Fatalf("list providers: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("got %d providers, want 2", len(list))
	}
}

func TestDeleteProvider(t *testing.T) {
	d := openTestDB(t)
	p := &models.Provider{ID: "del-me", Name: "Delete Me", Source: "custom", Status: "active"}
	if err := d.UpsertProvider(p); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if err := d.DeleteProvider("del-me"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err := d.GetProvider("del-me")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestDeleteProvider_CascadeDeletesModels(t *testing.T) {
	d := openTestDB(t)
	p := &models.Provider{ID: "cascade-provider", Name: "Cascade Test", Source: "custom", Status: "active"}
	if err := d.UpsertProvider(p); err != nil {
		t.Fatalf("upsert provider: %v", err)
	}
	m := &models.Model{
		ID: "cascade-model", ProviderID: "cascade-provider",
		DisplayName: "Cascade Model", Status: "untested", Source: "test",
	}
	if err := d.UpsertModel(m); err != nil {
		t.Fatalf("upsert model: %v", err)
	}

	if err := d.DeleteProvider("cascade-provider"); err != nil {
		t.Fatalf("delete provider: %v", err)
	}

	models, err := d.ListModels()
	if err != nil {
		t.Fatalf("list models: %v", err)
	}
	for _, m := range models {
		if m.ID == "cascade-model" {
			t.Fatal("model should have been deleted with provider")
		}
	}
}

func TestUpsertModel(t *testing.T) {
	d := openTestDB(t)
	p := &models.Provider{ID: "test-provider", Name: "Test", Source: "custom", Status: "active"}
	if err := d.UpsertProvider(p); err != nil {
		t.Fatalf("upsert provider: %v", err)
	}
	m := &models.Model{
		ID:             "test-model",
		ProviderID:     "test-provider",
		DisplayName:    "Test Model",
		ContextWindow:  100000,
		FunctionCalling: true,
		Vision:         false,
		Status:         "active",
		Source:         "discovered",
	}
	if err := d.UpsertModel(m); err != nil {
		t.Fatalf("upsert model: %v", err)
	}
	got, err := d.GetModel("test-model")
	if err != nil {
		t.Fatalf("get model: %v", err)
	}
	if got.DisplayName != "Test Model" {
		t.Errorf("got display name %q", got.DisplayName)
	}
	if !got.FunctionCalling {
		t.Error("expected function_calling=true")
	}
}

func TestGetModel_NotFound(t *testing.T) {
	d := openTestDB(t)
	_, err := d.GetModel("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent model")
	}
}

func TestListModelsByProvider(t *testing.T) {
	d := openTestDB(t)
	p := &models.Provider{ID: "prov1", Name: "Prov1", Source: "custom", Status: "active"}
	if err := d.UpsertProvider(p); err != nil {
		t.Fatalf("upsert provider: %v", err)
	}
	models := []*models.Model{
		{ID: "m1", ProviderID: "prov1", DisplayName: "Model 1", Status: "active", Source: "discovered"},
		{ID: "m2", ProviderID: "prov1", DisplayName: "Model 2", Status: "active", Source: "discovered"},
	}
	for _, m := range models {
		if err := d.UpsertModel(m); err != nil {
			t.Fatalf("upsert model: %v", err)
		}
	}
	list, err := d.ListModelsByProvider("prov1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("got %d models, want 2", len(list))
	}
}

func TestListModels_FilterByStatus(t *testing.T) {
	d := openTestDB(t)
	p := &models.Provider{ID: "p", Name: "P", Source: "custom", Status: "active"}
	if err := d.UpsertProvider(p); err != nil {
		t.Fatalf("upsert provider: %v", err)
	}
	models := []*models.Model{
		{ID: "m1", ProviderID: "p", DisplayName: "Active", Status: "active", Source: "discovered"},
		{ID: "m2", ProviderID: "p", DisplayName: "Error", Status: "error", Source: "discovered"},
	}
	for _, m := range models {
		if err := d.UpsertModel(m); err != nil {
			t.Fatalf("upsert model: %v", err)
		}
	}
	active, err := d.ListModels(db.StatusActive())
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("got %d active, want 1", len(active))
	}
	if active[0].ID != "m1" {
		t.Errorf("got model %q", active[0].ID)
	}
}

func TestSearchModels(t *testing.T) {
	d := openTestDB(t)
	p := &models.Provider{ID: "p", Name: "P", Source: "custom", Status: "active"}
	if err := d.UpsertProvider(p); err != nil {
		t.Fatalf("upsert provider: %v", err)
	}
	models := []*models.Model{
		{ID: "gpt-4", ProviderID: "p", DisplayName: "GPT-4", Status: "active", Source: "discovered"},
		{ID: "claude-3", ProviderID: "p", DisplayName: "Claude 3", Status: "active", Source: "discovered"},
		{ID: "gpt-3.5", ProviderID: "p", DisplayName: "GPT-3.5", Status: "active", Source: "discovered"},
	}
	for _, m := range models {
		if err := d.UpsertModel(m); err != nil {
			t.Fatalf("upsert model: %v", err)
		}
	}
	results, err := d.SearchModels("gpt")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
}

func TestGetStats(t *testing.T) {
	d := openTestDB(t)
	p := &models.Provider{ID: "p", Name: "P", Source: "custom", Status: "active"}
	if err := d.UpsertProvider(p); err != nil {
		t.Fatalf("upsert provider: %v", err)
	}
	models := []*models.Model{
		{ID: "m1", ProviderID: "p", DisplayName: "M1", Status: "active", Source: "discovered"},
		{ID: "m2", ProviderID: "p", DisplayName: "M2", Status: "error", Source: "discovered"},
	}
	for _, m := range models {
		if err := d.UpsertModel(m); err != nil {
			t.Fatalf("upsert model: %v", err)
		}
	}
	stats, err := d.GetStats()
	if err != nil {
		t.Fatalf("get stats: %v", err)
	}
	if stats["active"] != 1 {
		t.Errorf("got active=%d, want 1", stats["active"])
	}
	if stats["error"] != 1 {
		t.Errorf("got error=%d, want 1", stats["error"])
	}
}

func TestSeedDefaults(t *testing.T) {
	d := openTestDB(t)
	var count int
	err := d.QueryRow("SELECT COUNT(*) FROM routing_rules").Scan(&count)
	if err != nil {
		t.Fatalf("count routing_rules: %v", err)
	}
	if count == 0 {
		t.Error("expected seeded routing rules")
	}
}

func TestExecLog(t *testing.T) {
	d := openTestDB(t)
	if err := d.ExecLog("test", "ok", "testing exec log", 0); err != nil {
		t.Fatalf("exec log: %v", err)
	}
	var count int
	d.QueryRow("SELECT COUNT(*) FROM sync_log WHERE phase='test'").Scan(&count)
	if count != 1 {
		t.Errorf("got %d log entries, want 1", count)
	}
}

func TestUpsertModelProfile(t *testing.T) {
	d := openTestDB(t)
	p := &models.Provider{ID: "prov", Name: "Prov", Source: "custom", Status: "active"}
	if err := d.UpsertProvider(p); err != nil {
		t.Fatalf("upsert provider: %v", err)
	}
	m := &models.Model{ID: "test-model", ProviderID: "prov", DisplayName: "Test", Status: "active", Source: "discovered"}
	if err := d.UpsertModel(m); err != nil {
		t.Fatalf("upsert model: %v", err)
	}
	mp := &models.ModelProfile{
		ModelID:       "test-model",
		RealContext:   128000,
		MaxOutput:     4096,
		SupportsStream: true,
		SupportsSO:    true,
		StreamTPS:     150.5,
		ProfiledAt:    1700000000,
	}
	if err := d.UpsertModelProfile(mp); err != nil {
		t.Fatalf("upsert model profile: %v", err)
	}
	got, err := d.GetModelProfile("test-model")
	if err != nil {
		t.Fatalf("get model profile: %v", err)
	}
	if got.RealContext != 128000 {
		t.Errorf("got real_context=%d, want 128000", got.RealContext)
	}
	if !got.SupportsStream {
		t.Error("expected supports_stream=true")
	}
	if !got.SupportsSO {
		t.Error("expected supports_so=true")
	}
}

func TestUpsertModelProfile_NotFound(t *testing.T) {
	d := openTestDB(t)
	_, err := d.GetModelProfile("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent model profile")
	}
}

func TestListModelProfiles(t *testing.T) {
	d := openTestDB(t)
	p := &models.Provider{ID: "prov", Name: "Prov", Source: "custom", Status: "active"}
	if err := d.UpsertProvider(p); err != nil {
		t.Fatalf("upsert provider: %v", err)
	}
	for _, id := range []string{"m1", "m2"} {
		m := &models.Model{ID: id, ProviderID: "prov", DisplayName: id, Status: "active", Source: "discovered"}
		if err := d.UpsertModel(m); err != nil {
			t.Fatalf("upsert model %s: %v", id, err)
		}
		if err := d.UpsertModelProfile(&models.ModelProfile{ModelID: id, RealContext: 1000 * len(id)}); err != nil {
			t.Fatalf("upsert profile %s: %v", id, err)
		}
	}
	list, err := d.ListModelProfiles()
	if err != nil {
		t.Fatalf("list model profiles: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("got %d profiles, want 2", len(list))
	}
}

func TestUpsertSourceItem(t *testing.T) {
	d := openTestDB(t)
	src := &models.Source{ID: "test-source", RemoteURL: "https://example.com", Status: "active"}
	if err := d.UpsertSource(src); err != nil {
		t.Fatalf("upsert source: %v", err)
	}
	si := &models.SourceItem{
		ID:         "item1",
		SourceID:   "test-source",
		Type:       "skill",
		SourcePath: "/path/to/skill",
		TargetPath: "/target/skill",
		Hash:       "abc123",
		Status:     "active",
	}
	if err := d.UpsertSourceItem(si); err != nil {
		t.Fatalf("upsert source item: %v", err)
	}
	got, err := d.GetSourceItem("item1")
	if err != nil {
		t.Fatalf("get source item: %v", err)
	}
	if got.Type != "skill" {
		t.Errorf("got type %q, want 'skill'", got.Type)
	}
	if got.Hash != "abc123" {
		t.Errorf("got hash %q, want 'abc123'", got.Hash)
	}
}

func TestUpsertSourceItem_Duplicate(t *testing.T) {
	d := openTestDB(t)
	src := &models.Source{ID: "src", RemoteURL: "https://example.com", Status: "active"}
	if err := d.UpsertSource(src); err != nil {
		t.Fatalf("upsert source: %v", err)
	}
	si := &models.SourceItem{ID: "item1", SourceID: "src", Type: "skill", Status: "active"}
	if err := d.UpsertSourceItem(si); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	si2 := &models.SourceItem{ID: "item1", SourceID: "src", Type: "plugin", Status: "active"}
	if err := d.UpsertSourceItem(si2); err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	got, err := d.GetSourceItem("item1")
	if err != nil {
		t.Fatalf("get source item: %v", err)
	}
	if got.Type != "plugin" {
		t.Errorf("got type %q, want 'plugin'", got.Type)
	}
}

func TestGetSourceItem_NotFound(t *testing.T) {
	d := openTestDB(t)
	_, err := d.GetSourceItem("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent source item")
	}
}

func TestListSourceItems(t *testing.T) {
	d := openTestDB(t)
	src := &models.Source{ID: "src", RemoteURL: "https://example.com", Status: "active"}
	if err := d.UpsertSource(src); err != nil {
		t.Fatalf("upsert source: %v", err)
	}
	items := []*models.SourceItem{
		{ID: "i1", SourceID: "src", Type: "skill", Status: "active"},
		{ID: "i2", SourceID: "src", Type: "plugin", Status: "active"},
	}
	for _, si := range items {
		if err := d.UpsertSourceItem(si); err != nil {
			t.Fatalf("upsert source item: %v", err)
		}
	}
	list, err := d.ListSourceItems()
	if err != nil {
		t.Fatalf("list source items: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("got %d items, want 2", len(list))
	}
}

func TestDeleteSourceItem(t *testing.T) {
	d := openTestDB(t)
	src := &models.Source{ID: "src", RemoteURL: "https://example.com", Status: "active"}
	if err := d.UpsertSource(src); err != nil {
		t.Fatalf("upsert source: %v", err)
	}
	si := &models.SourceItem{ID: "del-me", SourceID: "src", Type: "skill", Status: "active"}
	if err := d.UpsertSourceItem(si); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if err := d.DeleteSourceItem("del-me"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err := d.GetSourceItem("del-me")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestUpsertLSPServer(t *testing.T) {
	d := openTestDB(t)
	l := &models.LSPServer{
		ID:             "lsp-test",
		Command:        `["gopls"]`,
		Extensions:     `["go"]`,
		Env:            `{"GOFLAGS":"-mod=mod"}`,
		Initialization: `{"maxCompletionItems":100}`,
		Disabled:       false,
	}
	if err := d.UpsertLSPServer(l); err != nil {
		t.Fatalf("upsert LSP server: %v", err)
	}
	got, err := d.GetLSPServer("lsp-test")
	if err != nil {
		t.Fatalf("get LSP server: %v", err)
	}
	if got.Command != `["gopls"]` {
		t.Errorf("got command %q", got.Command)
	}
	if got.Disabled {
		t.Error("expected disabled=false")
	}
}

func TestUpsertLSPServer_Duplicate(t *testing.T) {
	d := openTestDB(t)
	l := &models.LSPServer{ID: "lsp-dup", Command: `["gopls"]`, Disabled: false}
	if err := d.UpsertLSPServer(l); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	l2 := &models.LSPServer{ID: "lsp-dup", Command: `["typescript-language-server"]`, Disabled: true}
	if err := d.UpsertLSPServer(l2); err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	got, err := d.GetLSPServer("lsp-dup")
	if err != nil {
		t.Fatalf("get LSP server: %v", err)
	}
	if got.Command != `["typescript-language-server"]` {
		t.Errorf("got command %q", got.Command)
	}
	if !got.Disabled {
		t.Error("expected disabled=true")
	}
}

func TestGetLSPServer_NotFound(t *testing.T) {
	d := openTestDB(t)
	_, err := d.GetLSPServer("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent LSP server")
	}
}

func TestListLSPServers(t *testing.T) {
	d := openTestDB(t)
	servers := []*models.LSPServer{
		{ID: "lsp1", Command: `["gopls"]`},
		{ID: "lsp2", Command: `["typescript-language-server"]`},
	}
	for _, l := range servers {
		if err := d.UpsertLSPServer(l); err != nil {
			t.Fatalf("upsert LSP: %v", err)
		}
	}
	list, err := d.ListLSPServers()
	if err != nil {
		t.Fatalf("list LSP servers: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("got %d servers, want 2", len(list))
	}
}

func TestDeleteLSPServer(t *testing.T) {
	d := openTestDB(t)
	l := &models.LSPServer{ID: "lsp-del", Command: `["gopls"]`}
	if err := d.UpsertLSPServer(l); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if err := d.DeleteLSPServer("lsp-del"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err := d.GetLSPServer("lsp-del")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}
