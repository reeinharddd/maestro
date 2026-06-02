package heal

import (
	"context"
	"fmt"
	"time"

	"github.com/reeinharddd/okit/internal/db"
)

type Service struct {
	db db.DBInterface
}

func New(database db.DBInterface) *Service {
	return &Service{db: database}
}

type HealReport struct {
	IssuesFound int
	IssuesFixed int
	Issues      []HealIssue
}

type HealIssue struct {
	Severity  string
	Component string
	Message   string
	Action    string
	Fixed     bool
}

func (s *Service) Run(ctx context.Context) (*HealReport, error) {
	report := &HealReport{}

	issues, err := s.checkModelsWithError()
	if err != nil {
		return nil, err
	}
	for _, iss := range issues {
		report.Issues = append(report.Issues, iss)
		report.IssuesFound++
		if iss.Fixed {
			report.IssuesFixed++
		}
	}

	issues, err = s.checkProviders()
	if err != nil {
		return nil, err
	}
	for _, iss := range issues {
		report.Issues = append(report.Issues, iss)
		report.IssuesFound++
		if iss.Fixed {
			report.IssuesFixed++
		}
	}

	issues, err = s.checkStaleModels()
	if err != nil {
		return nil, err
	}
	for _, iss := range issues {
		report.Issues = append(report.Issues, iss)
		report.IssuesFound++
		if iss.Fixed {
			report.IssuesFixed++
		}
	}

	issues, err = s.checkDBIntegrity()
	if err != nil {
		return nil, err
	}
	for _, iss := range issues {
		report.Issues = append(report.Issues, iss)
		report.IssuesFound++
		if iss.Fixed {
			report.IssuesFixed++
		}
	}

	return report, nil
}

func (s *Service) checkModelsWithError() ([]HealIssue, error) {
	rows, err := s.db.Query(`SELECT id, fail_count FROM models WHERE status='error' AND fail_count > 3`)
	if err != nil {
		return nil, err
	}
	type failedModel struct {
		id        string
		failCount int
	}
	var candidates []failedModel
	for rows.Next() {
		var fm failedModel
		if err := rows.Scan(&fm.id, &fm.failCount); err != nil {
			continue
		}
		candidates = append(candidates, fm)
	}
	rows.Close()

	var issues []HealIssue
	for _, fm := range candidates {
		_, err := s.db.Exec(`UPDATE models SET status='deprecated' WHERE id=?`, fm.id)
		fixed := err == nil
		msg := fmt.Sprintf("Model %s deprecated (%d consecutive failures)", fm.id, fm.failCount)
		issues = append(issues, HealIssue{
			Severity: "warning", Component: "models",
			Message: msg, Action: "status→deprecated", Fixed: fixed,
		})
	}
	return issues, nil
}

func (s *Service) checkProviders() ([]HealIssue, error) {
	var issues []HealIssue
	providers, err := s.db.ListProviders()
	if err != nil {
		return nil, err
	}
	for _, p := range providers {
		models, _ := s.db.ListModelsByProvider(p.ID)
		hasActive := false
		for _, m := range models {
			if m.Status == "active" {
				hasActive = true
				break
			}
		}
		if !hasActive {
			issues = append(issues, HealIssue{
				Severity: "warning", Component: "providers",
				Message: fmt.Sprintf("Provider %s has no active models", p.ID),
				Action:  "manual review needed", Fixed: false,
			})
		}
	}
	return issues, nil
}

func (s *Service) checkStaleModels() ([]HealIssue, error) {
	var issues []HealIssue
	weekAgo := time.Now().Add(-7 * 24 * time.Hour).Unix()
	rows, err := s.db.Query(`SELECT id FROM models WHERE status='active' AND last_tested > 0 AND last_tested < ?`, weekAgo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		issues = append(issues, HealIssue{
			Severity: "info", Component: "models",
			Message: fmt.Sprintf("Model %s needs re-test (last tested >7 days ago)", id),
			Action:  "scheduled for re-test", Fixed: true,
		})
	}
	return issues, nil
}

func (s *Service) checkDBIntegrity() ([]HealIssue, error) {
	var issues []HealIssue
	rows, err := s.db.Query(`PRAGMA integrity_check`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var result string
		if err := rows.Scan(&result); err == nil && result != "ok" {
			issues = append(issues, HealIssue{
				Severity: "critical", Component: "config",
				Message: fmt.Sprintf("DB integrity: %s", result),
				Action:  "restore from backup", Fixed: false,
			})
		}
	}
	return issues, nil
}
