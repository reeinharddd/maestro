package routing_test

import (
	"context"
	"testing"
	"github.com/reeinharrrd/opencode-kit/internal/db"
	"github.com/reeinharrrd/opencode-kit/internal/routing"
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

	s := routing.New(d)
	budget := models.BudgetConfig{PreferredTier: "free_only"}
	rule, err := s.SelectBestModel("coding_fast", budget)
	if err != nil {
		t.Fatalf("select best model: %v", err)
	}
	if rule.TaskKey != "coding_fast" {
		t.Errorf("got task key %q", rule.TaskKey)
	}
	if rule.CurrentModelID == "" {
		t.Error("expected non-empty model ID")
	}
}

func TestSelectBestModel_UnknownTask(t *testing.T) {
	d := openTestDB(t)
	s := routing.New(d)
	_, err := s.SelectBestModel("nonexistent_task", models.BudgetConfig{})
	if err == nil {
		t.Fatal("expected error for unknown task type")
	}
}

func TestSelectBestModel_NoSuitableModel(t *testing.T) {
	d := openTestDB(t)
	addProvider(t, d, "p")
	addModel(t, d, "no-fc", "p", 1000, false, false, "free", "active")

	s := routing.New(d)
	_, err := s.SelectBestModel("coding_complex", models.BudgetConfig{PreferredTier: "free_only"})
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
	rule, err := s.SelectBestModel("coding_complex", budget)
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
	rule, err := s.SelectBestModel("vision", models.BudgetConfig{PreferredTier: "free_only"})
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
	if err := s.ReassignAll(context.Background()); err != nil {
		t.Fatalf("reassign all: %v", err)
	}
}

func TestNewService(t *testing.T) {
	d := openTestDB(t)
	s := routing.New(d)
	if s == nil {
		t.Fatal("expected non-nil service")
	}
}
