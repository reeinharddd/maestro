package db

import (
	"fmt"
	"strings"

	"github.com/reeinharrrd/maestro/pkg/models"
)

func (d *DB) UpsertCommand(c *models.Command) error {
	return d.upsertRow("commands", "id", []upsertCol{
		{"id", c.ID},
		{"command_template", c.Template},
		{"description", c.Description},
		{"agent", c.Agent},
		{"model", c.Model},
		{"subtask", boolToInt(c.Subtask)},
		{"source", c.Source},
		{"status", c.Status},
	})
}

func (d *DB) ListCommands() ([]models.Command, error) {
	rows, err := d.Query(`SELECT id, COALESCE(command_template,''), COALESCE(description,''), COALESCE(agent,''), COALESCE(model,''), COALESCE(subtask,0), COALESCE(source,''), COALESCE(status,'active') FROM commands ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Command
	for rows.Next() {
		var c models.Command
		var sub int
		if err := rows.Scan(&c.ID, &c.Template, &c.Description, &c.Agent, &c.Model, &sub, &c.Source, &c.Status); err != nil {
			return nil, err
		}
		c.Subtask = sub != 0
		out = append(out, c)
	}
	return out, nil
}

func (d *DB) DeleteCommand(id string) error {
	_, err := d.Exec(`DELETE FROM commands WHERE id=?`, id)
	return err
}

func (d *DB) UpsertMCP(m *models.MCPServer) error {
	return d.upsertRow("mcp_servers", "id", []upsertCol{
		{"id", m.ID},
		{"type", m.Type},
		{"command", m.Command},
		{"url", m.URL},
		{"enabled", boolToInt(m.Enabled)},
		{"env_vars", m.EnvVars},
		{"timeout_ms", m.Timeout},
		{"source", m.Source},
	})
}

func (d *DB) ListMCPs() ([]models.MCPServer, error) {
	rows, err := d.Query(`SELECT id, COALESCE(type,'local'), COALESCE(command,''), COALESCE(url,''), COALESCE(enabled,0), COALESCE(env_vars,''), COALESCE(timeout_ms,5000), COALESCE(source,'') FROM mcp_servers ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.MCPServer
	for rows.Next() {
		var m models.MCPServer
		var en int
		if err := rows.Scan(&m.ID, &m.Type, &m.Command, &m.URL, &en, &m.EnvVars, &m.Timeout, &m.Source); err != nil {
			return nil, err
		}
		m.Enabled = en != 0
		out = append(out, m)
	}
	return out, nil
}

func (d *DB) DeleteMCP(id string) error {
	_, err := d.Exec(`DELETE FROM mcp_servers WHERE id=?`, id)
	return err
}

func (d *DB) UpsertSkill(s *models.Skill) error {
	return d.upsertRow("skills", "id", []upsertCol{
		{"id", s.ID},
		{"source", s.Source},
		{"source_path", s.SourcePath},
		{"target_path", s.TargetPath},
		{"type", s.Type},
		{"status", s.Status},
		{"hash", s.Hash},
		{"last_synced", s.LastSynced},
		{"description", s.Description},
		{"category", s.Category},
		{"tags", s.Tags},
		{"triggers", s.Triggers},
		{"size_bytes", s.SizeBytes},
		{"filename", s.Filename},
	})
}

func (d *DB) ListSkills() ([]models.Skill, error) {
	rows, err := d.Query(`SELECT id, COALESCE(source,''), COALESCE(source_path,''), COALESCE(target_path,''), COALESCE(type,'skill'), COALESCE(status,'active'), COALESCE(hash,''), COALESCE(last_synced,0), COALESCE(description,''), COALESCE(category,''), COALESCE(tags,''), COALESCE(triggers,''), COALESCE(size_bytes,0), COALESCE(filename,'') FROM skills ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Skill
	for rows.Next() {
		var s models.Skill
		if err := rows.Scan(&s.ID, &s.Source, &s.SourcePath, &s.TargetPath, &s.Type, &s.Status, &s.Hash, &s.LastSynced, &s.Description, &s.Category, &s.Tags, &s.Triggers, &s.SizeBytes, &s.Filename); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func (d *DB) DeleteSkill(id string) error {
	_, err := d.Exec(`DELETE FROM skills WHERE id=?`, id)
	return err
}

func (d *DB) UpdateSkillMeta(id string, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}
	var sb strings.Builder
	sb.WriteString("UPDATE skills SET ")
	first := true
	args := make([]any, 0, len(updates)+1)
	for k, v := range updates {
		if !first {
			sb.WriteString(", ")
		}
		sb.WriteString(k)
		sb.WriteString("=?")
		args = append(args, v)
		first = false
	}
	sb.WriteString(" WHERE id=?")
	args = append(args, id)
	_, err := d.Exec(sb.String(), args...)
	return err
}

func (d *DB) SearchSkills(query string) ([]models.Skill, error) {
	pattern := "%" + query + "%"
	rows, err := d.Query(`SELECT id, COALESCE(source,''), COALESCE(source_path,''), COALESCE(target_path,''), COALESCE(type,'skill'), COALESCE(status,'active'), COALESCE(hash,''), COALESCE(last_synced,0), COALESCE(description,''), COALESCE(category,''), COALESCE(tags,''), COALESCE(triggers,''), COALESCE(size_bytes,0), COALESCE(filename,'') FROM skills WHERE id LIKE ? OR description LIKE ? OR tags LIKE ? OR category LIKE ? ORDER BY id`, pattern, pattern, pattern, pattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Skill
	for rows.Next() {
		var s models.Skill
		if err := rows.Scan(&s.ID, &s.Source, &s.SourcePath, &s.TargetPath, &s.Type, &s.Status, &s.Hash, &s.LastSynced, &s.Description, &s.Category, &s.Tags, &s.Triggers, &s.SizeBytes, &s.Filename); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func (d *DB) UpsertSourceItem(s *models.SourceItem) error {
	return d.upsertRow("source_items", "id", []upsertCol{
		{"id", s.ID},
		{"source_id", s.SourceID},
		{"type", s.Type},
		{"source_path", s.SourcePath},
		{"target_path", s.TargetPath},
		{"hash", s.Hash},
		{"status", s.Status},
	})
}

func (d *DB) ListSourceItems() ([]models.SourceItem, error) {
	rows, err := d.Query(`SELECT id, COALESCE(source_id,''), COALESCE(type,''), COALESCE(source_path,''), COALESCE(target_path,''), COALESCE(hash,''), COALESCE(status,'active') FROM source_items ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.SourceItem
	for rows.Next() {
		var s models.SourceItem
		if err := rows.Scan(&s.ID, &s.SourceID, &s.Type, &s.SourcePath, &s.TargetPath, &s.Hash, &s.Status); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func (d *DB) GetSourceItem(id string) (*models.SourceItem, error) {
	var s models.SourceItem
	err := d.QueryRow(`SELECT id, COALESCE(source_id,''), COALESCE(type,''), COALESCE(source_path,''), COALESCE(target_path,''), COALESCE(hash,''), COALESCE(status,'active') FROM source_items WHERE id=?`, id).
		Scan(&s.ID, &s.SourceID, &s.Type, &s.SourcePath, &s.TargetPath, &s.Hash, &s.Status)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (d *DB) DeleteSourceItem(id string) error {
	_, err := d.Exec(`DELETE FROM source_items WHERE id=?`, id)
	return err
}

func (d *DB) UpdateSourceItemStatus(id, status string) error {
	_, err := d.Exec(`UPDATE source_items SET status=? WHERE id=?`, status, id)
	return err
}

func (d *DB) UpdateSourceItemTarget(id, targetPath string) error {
	_, err := d.Exec(`UPDATE source_items SET target_path=? WHERE id=?`, targetPath, id)
	return err
}

func (d *DB) ListSourceItemsBySource(sourceID string) ([]models.SourceItem, error) {
	rows, err := d.Query(`SELECT id, COALESCE(source_id,''), COALESCE(type,''), COALESCE(source_path,''), COALESCE(target_path,''), COALESCE(hash,''), COALESCE(status,'active') FROM source_items WHERE source_id=? ORDER BY id`, sourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.SourceItem
	for rows.Next() {
		var s models.SourceItem
		if err := rows.Scan(&s.ID, &s.SourceID, &s.Type, &s.SourcePath, &s.TargetPath, &s.Hash, &s.Status); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func (d *DB) UpsertLSPServer(l *models.LSPServer) error {
	return d.upsertRow("lsp_servers", "id", []upsertCol{
		{"id", l.ID},
		{"command", l.Command},
		{"extensions", l.Extensions},
		{"env", l.Env},
		{"initialization", l.Initialization},
		{"disabled", boolToInt(l.Disabled)},
	})
}

// kept manual: upsertRow can't handle datetime('now') expressions
func (d *DB) UpsertConfigFragment(f *models.ConfigFragment) error {
	_, err := d.Exec(`INSERT INTO config_fragments (id, config_type, content, source, hash, updated_at)
		VALUES (?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(id) DO UPDATE SET
		config_type=excluded.config_type, content=excluded.content,
		source=excluded.source, hash=excluded.hash, updated_at=datetime('now')`,
		f.ID, f.ConfigType, f.Content, f.Source, f.Hash)
	return err
}

func (d *DB) ListConfigFragments(limit int) ([]models.ConfigFragment, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := d.Query(`SELECT id, COALESCE(config_type,''), COALESCE(content,''), COALESCE(source,''), COALESCE(hash,''), COALESCE(created_at,''), COALESCE(updated_at,'') FROM config_fragments ORDER BY updated_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ConfigFragment
	for rows.Next() {
		var f models.ConfigFragment
		if err := rows.Scan(&f.ID, &f.ConfigType, &f.Content, &f.Source, &f.Hash, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, nil
}

func (d *DB) GetConfigFragment(id string) (*models.ConfigFragment, error) {
	var f models.ConfigFragment
	err := d.QueryRow(`SELECT id, COALESCE(config_type,''), COALESCE(content,''), COALESCE(source,''), COALESCE(hash,''), COALESCE(created_at,''), COALESCE(updated_at,'') FROM config_fragments WHERE id=?`, id).
		Scan(&f.ID, &f.ConfigType, &f.Content, &f.Source, &f.Hash, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (d *DB) ListLSPServers() ([]models.LSPServer, error) {
	rows, err := d.Query(`SELECT id, COALESCE(command,''), COALESCE(extensions,''), COALESCE(env,''), COALESCE(initialization,''), COALESCE(disabled,0) FROM lsp_servers ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.LSPServer
	for rows.Next() {
		var l models.LSPServer
		var dis int
		if err := rows.Scan(&l.ID, &l.Command, &l.Extensions, &l.Env, &l.Initialization, &dis); err != nil {
			return nil, err
		}
		l.Disabled = dis != 0
		out = append(out, l)
	}
	return out, nil
}

func (d *DB) GetLSPServer(id string) (*models.LSPServer, error) {
	var l models.LSPServer
	var dis int
	err := d.QueryRow(`SELECT id, COALESCE(command,''), COALESCE(extensions,''), COALESCE(env,''), COALESCE(initialization,''), COALESCE(disabled,0) FROM lsp_servers WHERE id=?`, id).
		Scan(&l.ID, &l.Command, &l.Extensions, &l.Env, &l.Initialization, &dis)
	if err != nil {
		return nil, err
	}
	l.Disabled = dis != 0
	return &l, nil
}

func (d *DB) DeleteLSPServer(id string) error {
	_, err := d.Exec(`DELETE FROM lsp_servers WHERE id=?`, id)
	return err
}

func (d *DB) UpsertModelProfile(p *models.ModelProfile) error {
	return d.upsertRow("model_profiles", "model_id", []upsertCol{
		{"model_id", p.ModelID},
		{"real_context", p.RealContext},
		{"max_output", p.MaxOutput},
		{"supports_stream", boolToInt(p.SupportsStream)},
		{"supports_so", boolToInt(p.SupportsSO)},
		{"stream_tps", p.StreamTPS},
		{"profiled_at", p.ProfiledAt},
	})
}

func (d *DB) ListModelProfiles() ([]models.ModelProfile, error) {
	rows, err := d.Query(`SELECT model_id, COALESCE(real_context,0), COALESCE(max_output,0), COALESCE(supports_stream,0), COALESCE(supports_so,0), COALESCE(stream_tps,0), COALESCE(profiled_at,0) FROM model_profiles ORDER BY model_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ModelProfile
	for rows.Next() {
		var p models.ModelProfile
		var str, so int
		if err := rows.Scan(&p.ModelID, &p.RealContext, &p.MaxOutput, &str, &so, &p.StreamTPS, &p.ProfiledAt); err != nil {
			return nil, err
		}
		p.SupportsStream = str != 0
		p.SupportsSO = so != 0
		out = append(out, p)
	}
	return out, nil
}

func (d *DB) GetModelProfile(modelID string) (*models.ModelProfile, error) {
	var p models.ModelProfile
	var str, so int
	err := d.QueryRow(`SELECT model_id, COALESCE(real_context,0), COALESCE(max_output,0), COALESCE(supports_stream,0), COALESCE(supports_so,0), COALESCE(stream_tps,0), COALESCE(profiled_at,0) FROM model_profiles WHERE model_id=?`, modelID).
		Scan(&p.ModelID, &p.RealContext, &p.MaxOutput, &str, &so, &p.StreamTPS, &p.ProfiledAt)
	if err != nil {
		return nil, err
	}
	p.SupportsStream = str != 0
	p.SupportsSO = so != 0
	return &p, nil
}

func (d *DB) UpsertSource(src *models.Source) error {
	return d.upsertRow("sources", "id", []upsertCol{
		{"id", src.ID},
		{"remote_url", src.RemoteURL},
		{"local_path", src.LocalPath},
		{`"commit"`, src.Commit},
		{"status", src.Status},
		{"last_synced", src.LastSynced},
	})
}

func (d *DB) ListSources() ([]models.Source, error) {
	rows, err := d.Query(`SELECT id, COALESCE(remote_url,''), COALESCE(local_path,''), COALESCE("commit",''), COALESCE(status,'active'), COALESCE(last_synced,0) FROM sources ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Source
	for rows.Next() {
		var s models.Source
		if err := rows.Scan(&s.ID, &s.RemoteURL, &s.LocalPath, &s.Commit, &s.Status, &s.LastSynced); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func (d *DB) GetSource(id string) (*models.Source, error) {
	row := d.QueryRow(`SELECT id, remote_url, local_path, "commit", status, last_synced FROM sources WHERE id=?`, id)
	var s models.Source
	err := row.Scan(&s.ID, &s.RemoteURL, &s.LocalPath, &s.Commit, &s.Status, &s.LastSynced)
	if err != nil {
		return nil, fmt.Errorf("source %q not found: %w", id, err)
	}
	return &s, nil
}

func (d *DB) DeleteSource(id string) error {
	_, err := d.Exec(`DELETE FROM sources WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("delete source %q: %w", id, err)
	}
	_, _ = d.Exec(`DELETE FROM source_items WHERE source_id=?`, id)
	_, _ = d.Exec(`DELETE FROM skills WHERE source=?`, id)
	_, _ = d.Exec(`DELETE FROM agents WHERE source=?`, id)
	_, _ = d.Exec(`DELETE FROM commands WHERE source=?`, id)
	_, _ = d.Exec(`DELETE FROM mcp_servers WHERE source=?`, id)
	return nil
}
