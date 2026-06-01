package heal_test

import (
	"context"
	"testing"
	"github.com/reeinharrrd/opencode-kit/internal/db"
	"github.com/reeinharrrd/opencode-kit/internal/heal"
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

func TestHealRun_Empty(t *testing.T) {
	d := openTestDB(t)
	s := heal.New(d)
	report, err := s.Run(context.Background())
	if err != nil {
		t.Fatalf("heal run: %v", err)
	}
	if report == nil {
		t.Fatal("expected non-nil report")
	}
}

func TestHealRun_DeprecatesFailedModels(t *testing.T) {
	d := openTestDB(t)
	p := &models.Provider{ID: "p", Name: "P", Source: "auto", Status: "active"}
	if err := d.UpsertProvider(p); err != nil {
		t.Fatalf("upsert provider: %v", err)
	}
	m := &models.Model{ID: "broken", ProviderID: "p", DisplayName: "Broken", Status: "error", FailCount: 10, Source: "discovered"}
	if err := d.UpsertModel(m); err != nil {
		t.Fatalf("upsert model: %v", err)
	}
	s := heal.New(d)
	report, err := s.Run(context.Background())
	if err != nil {
		t.Fatalf("heal run: %v", err)
	}
	if report.IssuesFound < 1 {
		t.Error("expected at least 1 issue (deprecated model)")
	}
	fixed := false
	for _, issue := range report.Issues {
		if issue.Component == "models" && issue.Fixed {
			fixed = true
			break
		}
	}
	if !fixed {
		t.Error("expected a model issue to be fixed")
	}
	updated, _ := d.GetModel("broken")
	if updated != nil && updated.Status != "deprecated" {
		t.Errorf("expected model status 'deprecated', got %q", updated.Status)
	}
}

func TestHealRun_WarnsProviderWithoutActiveModels(t *testing.T) {
	d := openTestDB(t)
	p := &models.Provider{ID: "lonely", Name: "Lonely", Source: "auto", Status: "active"}
	if err := d.UpsertProvider(p); err != nil {
		t.Fatalf("upsert provider: %v", err)
	}
	m := &models.Model{ID: "only-model", ProviderID: "lonely", DisplayName: "Only", Status: "error", Source: "discovered"}
	if err := d.UpsertModel(m); err != nil {
		t.Fatalf("upsert model: %v", err)
	}
	s := heal.New(d)
	report, err := s.Run(context.Background())
	if err != nil {
		t.Fatalf("heal run: %v", err)
	}
	foundWarning := false
	for _, issue := range report.Issues {
		if issue.Component == "providers" && !issue.Fixed {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Error("expected a warning for provider with no active models")
	}
}

func TestHealReport_Fields(t *testing.T) {
	d := openTestDB(t)
	s := heal.New(d)
	report, err := s.Run(context.Background())
	if err != nil {
		t.Fatalf("heal run: %v", err)
	}
	if report.IssuesFound < 0 {
		t.Error("IssuesFound should be >= 0")
	}
	if report.IssuesFixed < 0 {
		t.Error("IssuesFixed should be >= 0")
	}
}
