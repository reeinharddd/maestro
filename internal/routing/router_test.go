package routing_test

import (
	"context"
	"encoding/json"
	"github.com/reeinharrrd/opencode-kit/internal/db"
	"github.com/reeinharrrd/opencode-kit/internal/routing"
	"github.com/reeinharrrd/opencode-kit/pkg/models"
	"testing"
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

func addProvider(t *testing.T, d *db.DB, id string) {
	t.Helper()
	if err := d.UpsertProvider(&models.Provider{
		ID: id, Name: id, Source: "auto", Status: "active",
	}); err != nil {
		t.Fatalf("add provider: %v", err)
	}
}

func addModel(t *testing.T, d *db.DB, id, provID string, ctx int, fc, vision bool, tier, status string) {
	t.Helper()
	if err := d.UpsertModel(&models.Model{
		ID: id, ProviderID: provID, DisplayName: id,
		ContextWindow: ctx, FunctionCalling: fc, Vision: vision,
		Tier: tier, Status: status, Source: "discovered",
	}); err != nil {
		t.Fatalf("add model: %v", err)
	}
}

func TestSelectBestModel_CodingFast(t *testing.T) {
	d := openTestDB(t)
	addProvider(t, d, "test-provider")
	addModel(t, d, "fast-model", "test-provider", 50000, true, false, "free", "active")
	addModel(t, d, "slow-model", "test-provider", 1000000, true, false, "free", "active")
	if err := d.UpsertModel(&models.Model{ID: "slow-model", ProviderID: "test-provider", DisplayName: "slow-model", ContextWindow: 1000000, FunctionCalling: true, Tier: "free", Status: "active", PricingPrompt: 0.2}); err != nil {
		t.Fatalf("set cost: %v", err)
	}

	s := routing.New(d)
	budget := models.BudgetConfig{PreferredTier: "free_only"}
	rule, err := s.SelectBestModel("coding_fast", budget, false)
	if err != nil {
		t.Fatalf("select best model: %v", err)
	}
	if rule.TaskKey != "coding_fast" {
		t.Errorf("got task key %q", rule.TaskKey)
	}
	if rule.CurrentModelID == "" {
		t.Error("expected non-empty model ID")
	}
	if rule.MaxCostPerCall == 0 {
		t.Error("expected max cost to be populated")
	}
	events, err := d.ListRoutingEvents(10)
	if err != nil {
		t.Fatalf("list routing events: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected routing event to be persisted")
	}
	if events[0].TaskKey != "coding_fast" {
		t.Fatalf("got task %q, want coding_fast", events[0].TaskKey)
	}
}

func TestSelectBestModel_UnknownTask(t *testing.T) {
	d := openTestDB(t)
	s := routing.New(d)
	_, err := s.SelectBestModel("nonexistent_task", models.BudgetConfig{}, false)
	if err == nil {
		t.Fatal("expected error for unknown task type")
	}
}

func TestSelectBestModel_NoSuitableModel(t *testing.T) {
	d := openTestDB(t)
	addProvider(t, d, "p")
	addModel(t, d, "no-fc", "p", 1000, false, false, "free", "active")

	s := routing.New(d)
	_, err := s.SelectBestModel("coding_complex", models.BudgetConfig{PreferredTier: "free_only"}, false)
	if err == nil {
		t.Fatal("expected error when no model meets requirements")
	}
}

func TestSelectBestModel_RespectsBudget(t *testing.T) {
	d := openTestDB(t)
	addProvider(t, d, "p")
	addModel(t, d, "free-model", "p", 100000, true, false, "free", "active")
	addModel(t, d, "paid-model", "p", 200000, true, false, "paid", "active")

	s := routing.New(d)
	budget := models.BudgetConfig{PreferredTier: "free_only"}
	rule, err := s.SelectBestModel("coding_complex", budget, false)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if rule.CurrentModelID == "paid-model" {
		t.Error("should not select paid model when budget is free_only")
	}
}

func TestSelectBestModel_VisionTask(t *testing.T) {
	d := openTestDB(t)
	addProvider(t, d, "p")
	addModel(t, d, "with-vision", "p", 100000, true, true, "free", "active")
	addModel(t, d, "no-vision", "p", 200000, true, false, "free", "active")

	s := routing.New(d)
	rule, err := s.SelectBestModel("vision", models.BudgetConfig{PreferredTier: "free_only"}, false)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if rule.CurrentModelID != "with-vision" {
		t.Error("expected model with vision capability")
	}
}

func TestReassignAll(t *testing.T) {
	d := openTestDB(t)
	addProvider(t, d, "p")
	addModel(t, d, "all-purpose", "p", 200000, true, true, "free", "active")

	s := routing.New(d)
	if err := s.ReassignAll(context.Background(), false); err != nil {
		t.Fatalf("reassign all: %v", err)
	}
	rules, err := d.ListRoutingRules()
	if err != nil {
		t.Fatalf("list routing rules: %v", err)
	}
	if len(rules) == 0 {
		t.Fatal("expected routing rules to be stored")
	}
}

func TestReassignAll_ShadowMode(t *testing.T) {
	d := openTestDB(t)
	addProvider(t, d, "p")
	addModel(t, d, "all-purpose", "p", 200000, true, true, "free", "active")

	s := routing.New(d)
	if err := s.ReassignAll(context.Background(), true); err != nil {
		t.Fatalf("shadow reassign: %v", err)
	}
	rules, err := d.ListRoutingRules()
	if err != nil {
		t.Fatalf("list routing rules: %v", err)
	}
	if len(rules) != 6 {
		t.Fatalf("got %d routing rules, want seeded 6", len(rules))
	}
}

func TestSelectBestModel_FallbackJSON(t *testing.T) {
	d := openTestDB(t)
	addProvider(t, d, "p")
	addModel(t, d, "best", "p", 200000, true, false, "free", "active")
	addModel(t, d, "backup-1", "p", 200000, true, false, "free", "active")
	addModel(t, d, "backup-2", "p", 200000, true, false, "free", "active")

	s := routing.New(d)
	rule, err := s.SelectBestModel("coding_fast", models.BudgetConfig{PreferredTier: "free_only"}, false)
	if err != nil {
		t.Fatalf("select best model: %v", err)
	}
	var fallbacks []string
	if err := json.Unmarshal([]byte(rule.FallbackIDs), &fallbacks); err != nil {
		t.Fatalf("fallback json: %v", err)
	}
	if len(fallbacks) == 0 {
		t.Fatal("expected fallback chain")
	}
	if got := routing.FormatCandidateSummary(rule.FallbackIDs); got == "-" {
		t.Fatal("expected non-empty candidate summary")
	}
}

func TestFormatCandidateSummary(t *testing.T) {
	got := routing.FormatCandidateSummary(`[{"id":"a","score":1.25},{"id":"b","score":0.5}]`)
	if got != "a=1.25, b=0.50" {
		t.Fatalf("got %q", got)
	}
}

func TestNewService(t *testing.T) {
	d := openTestDB(t)
	s := routing.New(d)
	if s == nil {
		t.Fatal("expected non-nil service")
	}
}
