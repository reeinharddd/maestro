package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/reeinharrrd/maestro/internal/audit"
	"github.com/reeinharrrd/maestro/internal/compress"
	"github.com/reeinharrrd/maestro/internal/discover"
	"github.com/reeinharrrd/maestro/internal/generator"
	"github.com/reeinharrrd/maestro/internal/heal"
	"github.com/reeinharrrd/maestro/internal/profile"
	"github.com/reeinharrrd/maestro/internal/routing"
	"github.com/spf13/cobra"
)

func newDailyCmd(dbPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "daily",
		Short: "Run full daily pipeline",
		Long: `Full daily pipeline:
  1. Backup snapshot
  2. Sync sources
  3. Discover models
  4. Audit models
  5. Generate config files
  6. Validate
  7. Heal if needed`,
		RunE: func(cmd *cobra.Command, args []string) error {
			full, _ := cmd.Flags().GetBool("full")
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()

			fmt.Println("=== Daily Pipeline ===")

			// 1. Backup
			fmt.Println("[1/9] Backup...")
			backupDir := filepath.Join(filepath.Dir(d.DBPath()), "backups")
			os.MkdirAll(backupDir, 0755)
			backupPath := filepath.Join(backupDir, fmt.Sprintf("opencode-kit-%s.tar.gz", time.Now().UTC().Format("2006-01-02T15-04-05")))
			if err := createBackup(d.DBPath(), backupPath); err != nil {
				fmt.Printf("  Warning: backup failed: %v\n", err)
			} else {
				fmt.Printf("  Backup: %s\n", backupPath)
				cleanupOldBackups(backupDir, 30*24*time.Hour)
			}

			// 2. Discover
			fmt.Println("[2/9] Discovering models...")
			dis := discover.NewService(discover.NewServiceParams{DB: d})
			if err := dis.Discover(cmd.Context()); err != nil {
				fmt.Printf("  Warning: discover: %v\n", err)
			}

			// 2a. Clean up stale preferences
			if cleaned, err := d.CleanupInvalidPreferences(); err == nil && cleaned > 0 {
				fmt.Printf("  Cleaned %d invalid preference values\n", cleaned)
			}
			if cleaned, err := d.CleanupProviderPrefs(); err == nil && cleaned > 0 {
				fmt.Printf("  Cleaned %d flat provider_* preferences\n", cleaned)
			}

			// 2b. Deduplicate providers with same base URL
			fmt.Println("[2b/9] Deduplicating providers with same base URL...")
			if err := dis.DeduplicateProviders(); err != nil {
				fmt.Printf("  Warning: dedup: %v\n", err)
			}

			// 2c. Live audit + fix drift (real /v1/models vs DB)
			fmt.Println("[2c/9] Live audit (real catalog vs DB) + auto-fix drift...")
			live := audit.NewLive(d, 4)
			fixes, err := live.FixAll(cmd.Context())
			if err != nil {
				fmt.Printf("  Warning: live fix: %v\n", err)
			} else {
				ph, ms, sk := 0, 0, 0
				for _, f := range fixes {
					if f.FetchError == "" {
						ph += f.PhantomFixed
						ms += f.MissingAdded
						sk += f.Skipped
					}
				}
				fmt.Printf("  Drift fixed: %d phantoms -> error, %d new untested, %d non-chat skipped\n",
					ph, ms, sk)
			}

			// 3. Audit
			fmt.Println("[3/9] Auditing models...")
			aud := audit.New(d, 5)
			if err := aud.Run(cmd.Context(), full); err != nil {
				fmt.Printf("  Warning: audit: %v\n", err)
			}

			// 3b. Activate untested models from known free providers
			fmt.Println("[3b/9] Activating untested free models...")
			if err := dis.ActivateUntestedFreeModels(); err != nil {
				fmt.Printf("  Warning: activate: %v\n", err)
			}

			// 4. Generate
			fmt.Println("[4/9] Generating configuration...")
			gen := generator.NewService(d, "")
			if err := gen.GenerateConfig(); err != nil {
				fmt.Printf("  Warning: generate config: %v\n", err)
			}

			// 5. Stats
			fmt.Println("[5/9] Collecting stats...")
			stats, err := d.GetStats()
			if err == nil {
				fmt.Printf("  Active models: %d\n", stats["active"])
				fmt.Printf("  Error models:  %d\n", stats["error"])
				fmt.Printf("  Untested:      %d\n", stats["untested"])
				fmt.Printf("  Providers:     %d\n", stats["providers_active"])
			}

			fmt.Println("[6/9] Profiling models...")
			prof := profile.New(d)
			if err := prof.ProfileAll(cmd.Context(), false); err != nil {
				fmt.Printf("  Warning: profile: %v\n", err)
			}

			fmt.Println("[7/9] Reassigning routing...")
			router := routing.New(d)
			if err := router.ReassignAll(cmd.Context(), false); err != nil {
				fmt.Printf("  Warning: route: %v\n", err)
			}

			fmt.Println("[8/9] Running auto-healing...")
			healer := heal.New(d)
			if report, err := healer.Run(cmd.Context()); err != nil {
				fmt.Printf("  Warning: heal: %v\n", err)
			} else if report.IssuesFound > 0 {
				fmt.Printf("  Healing: %d issues found, %d fixed\n", report.IssuesFound, report.IssuesFixed)
			}

			fmt.Println("[9/9] Compressing session observations...")
			fragments, _ := d.ListConfigFragments(50)
			obs := make([]compress.Observation, 0, len(fragments))
			step := 1
			for _, f := range fragments {
				obs = append(obs, compress.Observation{
					Source: f.Source, Step: step, Message: f.Content, Important: true,
				})
				step++
			}
			c := compress.NewWithDB(d, 12)
			if out := c.Compress(obs); out != "" {
				fmt.Printf("  Compressed: %d observation(s) -> fragment\n", len(obs))
			}

			fmt.Println("Daily pipeline complete.")
			return nil
		},
	}
	cmd.Flags().Bool("full", false, "Re-test already-active models during audit")
	return cmd
}

func createBackup(dbPath, backupPath string) error {
	buf := &bytes.Buffer{}
	gw := gzip.NewWriter(buf)
	tw := tar.NewWriter(gw)

	data, err := os.ReadFile(dbPath)
	if err != nil {
		return fmt.Errorf("read db: %w", err)
	}

	hdr := &tar.Header{
		Name: filepath.Base(dbPath),
		Size: int64(len(data)),
		Mode: 0644,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("tar header: %w", err)
	}
	if _, err := tw.Write(data); err != nil {
		return fmt.Errorf("tar write: %w", err)
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("tar close: %w", err)
	}
	if err := gw.Close(); err != nil {
		return fmt.Errorf("gzip close: %w", err)
	}

	return os.WriteFile(backupPath, buf.Bytes(), 0644)
}

func cleanupOldBackups(dir string, maxAge time.Duration) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-maxAge)
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(dir, info.Name()))
		}
	}
}
