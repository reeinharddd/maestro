package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/reeinharrrd/maestro/internal/db"
	"github.com/reeinharrrd/maestro/internal/sync"
	"github.com/spf13/cobra"
)

func newSyncCmd(dbPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Bidirectional sync between DB and config file",
		Long: `Bidirectional sync: imports changes from opencode config into DB,
then exports DB back to opencode config. Keeps both in sync.

Use after manually editing the config file, or after running
discover/audit/heal to push results to the config file.`,
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "bidirectional",
		Short: "Sync both directions (import \u2192 export)",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()

			configPath := findConfigPath(d)
			svc := sync.New(d)

			fmt.Printf("Syncing %s \u2192 DB...\n", opencodeConfigName())
			inDiff, err := svc.ImportFromOpenCodeConfig(configPath)
			if err != nil {
				return fmt.Errorf("import: %w", err)
			}
			fmt.Printf("  New providers: %d, new models: %d, new agents: %d, new commands: %d\n",
				len(inDiff.AddedProviders), len(inDiff.AddedModels), len(inDiff.AddedAgents), len(inDiff.AddedCommands))

			fmt.Printf("Syncing DB \u2192 %s...\n", opencodeConfigName())
			if err := svc.ExportToOpenCodeConfig(configPath); err != nil {
				return fmt.Errorf("export: %w", err)
			}
			fmt.Printf("  %s updated.\n", opencodeConfigName())

			fmt.Println("Sync complete.")
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "import",
		Short: "Import config file \u2192 DB only",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			svc := sync.New(d)
			configPath := findConfigPath(d)
			diff, err := svc.ImportFromOpenCodeConfig(configPath)
			if err != nil {
				return fmt.Errorf("import: %w", err)
			}
			fmt.Printf("Import complete. New providers: %d, new models: %d, new agents: %d, new commands: %d\n",
				len(diff.AddedProviders), len(diff.AddedModels), len(diff.AddedAgents), len(diff.AddedCommands))
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "export",
		Short: "Export DB \u2192 config file only",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			svc := sync.New(d)
			configPath := findConfigPath(d)
			if err := svc.ExportToOpenCodeConfig(configPath); err != nil {
				return fmt.Errorf("export: %w", err)
			}
			fmt.Printf("Export complete: %s updated.\n", opencodeConfigName())
			return nil
		},
	})
	return cmd
}

func findConfigPath(d *db.DB) string {
	configPath := OpenCodeConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configName := "opencode.json"
		if _, err := os.Stat(filepath.Join(filepath.Dir(d.DBPath()), "opencode.jsonc")); err == nil {
			configName = "opencode.jsonc"
		}
		configPath = filepath.Join(filepath.Dir(d.DBPath()), configName)
	}
	return configPath
}
