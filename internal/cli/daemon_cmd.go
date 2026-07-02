package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/reeinharrrd/maestro/internal/sources"
	"github.com/spf13/cobra"
)

func newDaemonCmd(dbPath *string) *cobra.Command {
	var interval time.Duration
	var autoInstall bool

	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Start the auto-sync daemon",
		Long: `Periodically sync all registered sources and optionally install
discovered items. The daemon runs continuously until interrupted.

Examples:
  maestro daemon                              # sync every 5 minutes
  maestro daemon --interval 30s               # sync every 30 seconds
  maestro daemon --install                    # sync and auto-install
  maestro daemon --interval 10m --install     # sync every 10 min + install`,
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()

			svc := sources.New(d)
			inst := sources.NewInstaller(d)

			fmt.Printf("Maestro daemon started (interval: %s, auto-install: %v)\n", interval, autoInstall)
			fmt.Println("Press Ctrl+C to stop.")

			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			// Run once immediately
			runSyncCycle(svc, inst, autoInstall)

			for range ticker.C {
				runSyncCycle(svc, inst, autoInstall)
			}

			return nil
		},
	}

	cmd.Flags().DurationVar(&interval, "interval", 5*time.Minute, "Sync interval (e.g. 30s, 5m, 1h)")
	cmd.Flags().BoolVar(&autoInstall, "install", false, "Auto-install discovered items")

	return cmd
}

func runSyncCycle(svc *sources.Service, inst *sources.Installer, autoInstall bool) {
	fmt.Printf("\n[%s] Syncing sources...\n", time.Now().Format("15:04:05"))

	if err := svc.Sync(context.Background()); err != nil {
		fmt.Printf("  Error: sync: %v\n", err)
		return
	}

	if !autoInstall {
		return
	}

	sources, err := svc.AllSources()
	if err != nil {
		fmt.Printf("  Error: list sources: %v\n", err)
		return
	}

	for _, src := range sources {
		if err := inst.InstallAll(src.ID); err != nil {
			fmt.Printf("  Error: install %s: %v\n", src.ID, err)
		}
	}
}
