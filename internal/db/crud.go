package db

import "github.com/reeinharrrd/opencode-kit/pkg/models"

func (d *DB) UpsertCommand(c *models.Command) error {
	_, err := d.Exec(`INSERT INTO commands (id, command_template, description, agent, model, subtask, source, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
		command_template=excluded.command_template, description=excluded.description,
		agent=excluded.agent, model=excluded.model, subtask=excluded.subtask,
		source=excluded.source, status=excluded.status`,
		c.ID, c.Template, c.Description, c.Agent, c.Model, boolToInt(c.Subtask), c.Source, c.Status)
	return err
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

func (d *DB) UpsertMCP(m *models.MCPServer) error {
	_, err := d.Exec(`INSERT INTO mcp_servers (id, type, command, url, enabled, env_vars, timeout_ms, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
		type=excluded.type, command=excluded.command, url=excluded.url,
		enabled=excluded.enabled, env_vars=excluded.env_vars,
		timeout_ms=excluded.timeout_ms, source=excluded.source`,
		m.ID, m.Type, m.Command, m.URL, boolToInt(m.Enabled), m.EnvVars, m.Timeout, m.Source)
	return err
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

func (d *DB) UpsertSkill(s *models.Skill) error {
	_, err := d.Exec(`INSERT INTO skills (id, source, source_path, target_path, type, status, hash, last_synced)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
		source=excluded.source, source_path=excluded.source_path,
		target_path=excluded.target_path, type=excluded.type,
		status=excluded.status, hash=excluded.hash, last_synced=excluded.last_synced`,
		s.ID, s.Source, s.SourcePath, s.TargetPath, s.Type, s.Status, s.Hash, s.LastSynced)
	return err
}

func (d *DB) ListSkills() ([]models.Skill, error) {
	rows, err := d.Query(`SELECT id, COALESCE(source,''), COALESCE(source_path,''), COALESCE(target_path,''), COALESCE(type,'skill'), COALESCE(status,'active'), COALESCE(hash,''), COALESCE(last_synced,0) FROM skills ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Skill
	for rows.Next() {
		var s models.Skill
		if err := rows.Scan(&s.ID, &s.Source, &s.SourcePath, &s.TargetPath, &s.Type, &s.Status, &s.Hash, &s.LastSynced); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func (d *DB) UpsertSourceItem(s *models.SourceItem) error {
	_, err := d.Exec(`INSERT INTO source_items (id, source_id, type, source_path, target_path, hash, status)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
		source_id=excluded.source_id, type=excluded.type,
		source_path=excluded.source_path, target_path=excluded.target_path,
		hash=excluded.hash, status=excluded.status`,
		s.ID, s.SourceID, s.Type, s.SourcePath, s.TargetPath, s.Hash, s.Status)
	return err
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

func (d *DB) UpsertLSPServer(l *models.LSPServer) error {
	_, err := d.Exec(`INSERT INTO lsp_servers (id, command, extensions, env, initialization, disabled)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
		command=excluded.command, extensions=excluded.extensions,
		env=excluded.env, initialization=excluded.initialization,
		disabled=excluded.disabled`,
		l.ID, l.Command, l.Extensions, l.Env, l.Initialization, boolToInt(l.Disabled))
	return err
}

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
	_, err := d.Exec(`INSERT INTO model_profiles (model_id, real_context, max_output, supports_stream, supports_so, stream_tps, profiled_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(model_id) DO UPDATE SET
		real_context=excluded.real_context, max_output=excluded.max_output,
		supports_stream=excluded.supports_stream, supports_so=excluded.supports_so,
		stream_tps=excluded.stream_tps, profiled_at=excluded.profiled_at`,
		p.ModelID, p.RealContext, p.MaxOutput, boolToInt(p.SupportsStream), boolToInt(p.SupportsSO), p.StreamTPS, p.ProfiledAt)
	return err
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
	_, err := d.Exec(`INSERT INTO sources (id, remote_url, local_path, "commit", status, last_synced)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
		remote_url=excluded.remote_url, local_path=excluded.local_path,
		"commit"=excluded."commit", status=excluded.status, last_synced=excluded.last_synced`,
		src.ID, src.RemoteURL, src.LocalPath, src.Commit, src.Status, src.LastSynced)
	return err
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
