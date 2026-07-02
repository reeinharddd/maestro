package db

import (
	"fmt"

	"github.com/reeinharrrd/maestro/pkg/models"
)

// ── Project CRUD ──────────────────────────────────────────────────────

func (d *DB) UpsertProject(p *models.Project) error {
	return d.upsertRow("projects", "path", []upsertCol{
		{"id", p.ID},
		{"path", p.Path},
		{"name", p.Name},
		{"detected_at", p.DetectedAt},
		{"updated_at", p.UpdatedAt},
		{"status", p.Status},
		{"source", p.Source},
	})
}

var projectCols = `id, COALESCE(path,''), COALESCE(name,''), COALESCE(detected_at,0), COALESCE(updated_at,0), COALESCE(status,'active'), COALESCE(source,'scan')`

func scanProject(scanner interface {
	Scan(dest ...interface{}) error
}) (models.Project, error) {
	var p models.Project
	err := scanner.Scan(&p.ID, &p.Path, &p.Name, &p.DetectedAt, &p.UpdatedAt, &p.Status, &p.Source)
	if err != nil {
		return p, err
	}
	return p, nil
}

func (d *DB) ListProjects() ([]models.Project, error) {
	rows, err := d.Query(`SELECT ` + projectCols + ` FROM projects ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Project
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (d *DB) GetProject(id string) (*models.Project, error) {
	p, err := scanProject(d.QueryRow(`SELECT `+projectCols+` FROM projects WHERE id=?`, id))
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (d *DB) DeleteProject(id string) error {
	_, err := d.Exec(`DELETE FROM projects WHERE id=?`, id)
	return err
}

// ── DetectedStack CRUD ────────────────────────────────────────────────

func (d *DB) UpsertDetectedStack(s *models.DetectedStack) error {
	return d.upsertRow("detected_stacks", "project_id, language", []upsertCol{
		{"id", s.ID},
		{"project_id", s.ProjectID},
		{"language", s.Language},
		{"framework", s.Framework},
		{"version", s.Version},
		{"builder", s.Builder},
		{"test_runner", s.TestRunner},
		{"linter", s.Linter},
		{"detected_at", s.DetectedAt},
		{"confidence", s.Confidence},
	})
}

var stackCols = `id, COALESCE(project_id,''), COALESCE(language,''), COALESCE(framework,''), COALESCE(version,''), COALESCE(builder,''), COALESCE(test_runner,''), COALESCE(linter,''), COALESCE(detected_at,0), COALESCE(confidence,1.0)`

func scanStack(scanner interface {
	Scan(dest ...interface{}) error
}) (models.DetectedStack, error) {
	var s models.DetectedStack
	err := scanner.Scan(&s.ID, &s.ProjectID, &s.Language, &s.Framework, &s.Version, &s.Builder, &s.TestRunner, &s.Linter, &s.DetectedAt, &s.Confidence)
	if err != nil {
		return s, err
	}
	return s, nil
}

func (d *DB) ListDetectedStacks(projectID string) ([]models.DetectedStack, error) {
	rows, err := d.Query(`SELECT `+stackCols+` FROM detected_stacks WHERE project_id=? ORDER BY language`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.DetectedStack
	for rows.Next() {
		s, err := scanStack(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func (d *DB) DeleteDetectedStacks(projectID string) error {
	_, err := d.Exec(`DELETE FROM detected_stacks WHERE project_id=?`, projectID)
	return err
}

// ── ProjectConfig CRUD ────────────────────────────────────────────────

func (d *DB) UpsertProjectConfig(pc *models.ProjectConfig) error {
	return d.upsertRow("project_configs", "project_id, config_type", []upsertCol{
		{"id", pc.ID},
		{"project_id", pc.ProjectID},
		{"config_type", pc.ConfigType},
		{"content", pc.Content},
		{"generated_at", pc.GeneratedAt},
		{"hash", pc.Hash},
	})
}

var configCols = `id, COALESCE(project_id,''), COALESCE(config_type,''), COALESCE(content,''), COALESCE(generated_at,0), COALESCE(hash,'')`

func scanProjectConfig(scanner interface {
	Scan(dest ...interface{}) error
}) (models.ProjectConfig, error) {
	var pc models.ProjectConfig
	err := scanner.Scan(&pc.ID, &pc.ProjectID, &pc.ConfigType, &pc.Content, &pc.GeneratedAt, &pc.Hash)
	if err != nil {
		return pc, err
	}
	return pc, nil
}

func (d *DB) ListProjectConfigs(projectID string) ([]models.ProjectConfig, error) {
	rows, err := d.Query(`SELECT `+configCols+` FROM project_configs WHERE project_id=? ORDER BY config_type`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ProjectConfig
	for rows.Next() {
		pc, err := scanProjectConfig(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, pc)
	}
	return out, nil
}

func (d *DB) GetProjectConfig(projectID, configType string) (*models.ProjectConfig, error) {
	pc, err := scanProjectConfig(d.QueryRow(`SELECT `+configCols+` FROM project_configs WHERE project_id=? AND config_type=?`, projectID, configType))
	if err != nil {
		return nil, fmt.Errorf("project config %s/%s not found: %w", projectID, configType, err)
	}
	return &pc, nil
}

func (d *DB) DeleteProjectConfigs(projectID string) error {
	_, err := d.Exec(`DELETE FROM project_configs WHERE project_id=?`, projectID)
	return err
}
