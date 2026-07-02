package cli

import (
	"context"
	"log/slog"

	"github.com/reeinharrrd/maestro/internal/db"
	"github.com/reeinharrrd/maestro/internal/heal"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	var dbPath string
	var verbose bool

	root := &cobra.Command{
		Use:   "maestro",
		Short: "opencode-kit - OpenCode infrastructure manager",
		Long: `opencode-kit manages OpenCode configuration: discovers models,
audits capabilities, generates optimal config, and auto-heals issues.

  maestro discover     Fetch models from provider catalogs
  maestro audit        Test model capabilities
  maestro generate     Generate OpenCode config files
  maestro daily        Full daily pipeline
  maestro status       Show system status
  maestro query        Run SQL query against DB`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Name() == "help" || cmd.Name() == "completion" {
				return nil
			}
			if verbose {
				slog.SetLogLoggerLevel(slog.LevelDebug)
			}
			_ = LoadEnvFile()

			if d, err := openDB(&dbPath); err == nil {
				if providers, listErr := d.ListProviders(); listErr == nil {
					InjectKeysFromAuth(providers)
				}
				d.Close()
			}
			return nil
		},
	}

	root.PersistentFlags().StringVar(&dbPath, "db", "", "Path to SQLite database (default: auto-detect)")
	root.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable debug logging")

	root.AddCommand(newDiscoverCmd(&dbPath))
	root.AddCommand(newAuditCmd(&dbPath))
	root.AddCommand(newGenerateCmd(&dbPath))
	root.AddCommand(newDailyCmd(&dbPath))
	root.AddCommand(newStatusCmd(&dbPath))
	root.AddCommand(newQueryCmd(&dbPath))
	root.AddCommand(newProvidersCmd(&dbPath))
	root.AddCommand(newModelsCmd(&dbPath))
	root.AddCommand(newSyncCmd(&dbPath))
	root.AddCommand(newSourcesCmd(&dbPath))
	root.AddCommand(newProfileCmd(&dbPath))
	root.AddCommand(newRouteCmd(&dbPath))
	root.AddCommand(newHealCmd(&dbPath))
	root.AddCommand(newKeysCmd())
	root.AddCommand(newVerifyCmd())
	root.AddCommand(newDoctorCmd())
	root.AddCommand(newInitCmd())
	root.AddCommand(newMcpCmd())
	root.AddCommand(newMCPServersCmd(&dbPath))
	root.AddCommand(newCompressCmd())
	root.AddCommand(newValidateCmd(&dbPath))
	root.AddCommand(newModelsViewCmd(&dbPath))
	root.AddCommand(newBudgetCmd(&dbPath))
	root.AddCommand(newLSPServersCmd(&dbPath))
	root.AddCommand(newSnapshotsCmd(&dbPath))
	root.AddCommand(newPreferencesCmd(&dbPath))
	root.AddCommand(newSkillsCmd(&dbPath))
	root.AddCommand(newSkillCmd(&dbPath))
	root.AddCommand(newSourceItemsCmd(&dbPath))
	root.AddCommand(newExecLogCmd(&dbPath))
	root.AddCommand(newModelProfilesCmd(&dbPath))
	root.AddCommand(newAgentsCmd(&dbPath))
	root.AddCommand(newCommandsCmd(&dbPath))
	root.AddCommand(newDaemonCmd(&dbPath))

	return root
}

func openDB(path *string) (*db.DB, error) {
	p := ""
	if path != nil {
		p = *path
	}
	if p == "" {
		p = db.DefaultPath()
	}
	return db.Open(p)
}

func runHeal(d *db.DB) (*heal.HealReport, error) {
	return heal.New(d).Run(context.Background())
}

func newProvidersCmd(dbPath *string) *cobra.Command {
	return newProvidersCmdImpl(dbPath)
}

func newModelsCmd(dbPath *string) *cobra.Command {
	return newModelsCmdImpl(dbPath)
}

func newCommandsCmd(dbPath *string) *cobra.Command {
	return newCommandsCmdImpl(dbPath)
}
