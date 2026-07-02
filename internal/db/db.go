package db

import (
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/reeinharrrd/maestro/internal/config"
	"github.com/reeinharrrd/maestro/pkg/models"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

//go:embed seed/providers.json
var seedProvidersJSON embed.FS

type DB struct {
	*sql.DB
	Path string
}

var _ DBInterface = (*DB)(nil)

func DefaultPath() string {
	base := config.ConfigDir()
	os.MkdirAll(base, 0755)
	return filepath.Join(base, "opencode-kit.db")
}

func Open(path string) (*DB, error) {
	if path == "" {
		path = DefaultPath()
	}
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := Migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	d := &DB{DB: db, Path: path}
	if err := d.SeedDefaults(); err != nil {
		return nil, fmt.Errorf("seed: %w", err)
	}

	return d, nil
}

func (d *DB) Close() error {
	return d.DB.Close()
}

func (d *DB) DBPath() string {
	return d.Path
}

func (d *DB) Now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func (d *DB) ExecLog(syncType, status, message string, _ time.Duration) error {
	_, err := d.Exec(
		`INSERT INTO sync_log (phase, status, details) VALUES (?, ?, ?)`,
		syncType, status, message,
	)
	return err
}

func Migrate(db *sql.DB) error {
	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("migration source: %w", err)
	}
	driver, err := sqlite.WithInstance(db, &sqlite.Config{})
	if err != nil {
		return fmt.Errorf("migration driver: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", source, "sqlite", driver)
	if err != nil {
		return fmt.Errorf("migrate instance: %w", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

func loadSeedProviders() ([]models.Provider, error) {
	data, err := seedProvidersJSON.ReadFile("seed/providers.json")
	if err != nil {
		return nil, fmt.Errorf("read seed providers: %w", err)
	}
	var providers []models.Provider
	if err := json.Unmarshal(data, &providers); err != nil {
		return nil, fmt.Errorf("unmarshal seed providers: %w", err)
	}
	return providers, nil
}

func (d *DB) SeedDefaults() error {
	_, err := d.Exec(`INSERT OR IGNORE INTO budget_config (id, daily_global_usd, preferred_tier) VALUES ('default', 0.50, 'free_only')`)
	if err != nil {
		return fmt.Errorf("seed budget: %w", err)
	}
	_, err = d.Exec(`INSERT OR IGNORE INTO routing_rules (task_key, description, min_context, needs_fc, needs_vision, max_cost_per_call, current_model_id, last_assigned) VALUES
		('coding_complex', 'Complex coding tasks with function calling', 100000, 1, 0, 0, '', 0),
		('coding_fast', 'Fast coding with function calling', 50000, 1, 0, 0, '', 0),
		('reasoning', 'Deep reasoning and analysis', 100000, 0, 0, 0, '', 0),
		('vision', 'Vision and image understanding', 100000, 0, 1, 0, '', 0),
		('long_context', 'Long context research and analysis', 500000, 0, 0, 0, '', 0),
		('fastest', 'Simple tasks, maximum speed', 0, 0, 0, 0, '', 0)`)
	if err != nil {
		return fmt.Errorf("seed routing rules: %w", err)
	}
	providers, err := loadSeedProviders()
	if err != nil {
		return fmt.Errorf("load seed providers: %w", err)
	}
	for _, p := range providers {
		if removed, _ := d.GetPreference("seed_removed:" + p.ID); removed == "1" {
			continue
		}
		if _, err := d.GetProvider(p.ID); err != nil {
			if err := d.UpsertProvider(&p); err != nil {
				return fmt.Errorf("seed provider %s: %w", p.ID, err)
			}
		}
	}
	return nil
}
