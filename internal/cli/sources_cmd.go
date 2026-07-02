package cli

import (
	"context"
	"fmt"

	"github.com/reeinharrrd/maestro/internal/sources"
	"github.com/reeinharrrd/maestro/pkg/models"
	"github.com/spf13/cobra"
)

func newSourcesCmd(dbPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sources",
		Short: "Manage external sources",
		Long: `Register and manage external git sources. Maestro clones or pulls
the repository, then discovers any skills, agents, commands, MCPs,
and plugins inside it.`,
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List external sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			list, err := d.ListSources()
			if err != nil {
				return err
			}
			if len(list) == 0 {
				fmt.Println("No sources configured")
				return nil
			}
			for _, s := range list {
				status := s.Status
				if s.LastSynced > 0 {
					fmt.Printf("%s: %s (%s, last synced: %d)\n", s.ID, s.RemoteURL, status, s.LastSynced)
				} else {
					fmt.Printf("%s: %s (%s)\n", s.ID, s.RemoteURL, status)
				}
			}
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "add <git-url>",
		Short: "Register a new source from a git URL",
		Long: `Clone a git repository and register it as a source. Maestro
will discover skills, agents, commands, MCPs, and plugins
inside the repository.

Examples:
  maestro sources add https://github.com/user/my-skills
  maestro sources add git@github.com:user/repo.git`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			svc := sources.New(d)
			src, err := svc.AddSource(context.Background(), args[0])
			if err != nil {
				return fmt.Errorf("add source: %w", err)
			}
			fmt.Printf("Added source %s (%s)\n", src.ID, src.RemoteURL)
			if src.LocalPath != "" {
				fmt.Printf("  Local path: %s\n", src.LocalPath)
			}
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "remove <id>",
		Short: "Remove a source by ID",
		Long: `Remove a source and its local clone. All skills, agents,
commands, MCPs, and plugins discovered from this source
will also be removed from the database.

Examples:
  maestro sources list            # get the ID
  maestro sources remove github-com-user-my-skills`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			svc := sources.New(d)
			if err := svc.RemoveSource(context.Background(), args[0]); err != nil {
				return fmt.Errorf("remove source: %w", err)
			}
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "sync [<id>]",
		Short: "Sync (clone/pull) one or all sources",
		Long: `Clone missing or pull existing sources. If a source ID is
provided, only that source is synced. Otherwise all active
sources are synced. After syncing, re-discovers items.

Examples:
  maestro sources sync                    # sync all sources
  maestro sources sync github-com-user-repo  # sync one source`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			svc := sources.New(d)

			if len(args) == 1 {
				fmt.Printf("Syncing source %s...\n", args[0])
				if err := svc.SyncSourceByID(context.Background(), args[0]); err != nil {
					return fmt.Errorf("sync source: %w", err)
				}
				fmt.Printf("  Source %s synced.\n", args[0])
			} else {
				fmt.Println("Syncing all sources...")
				if err := svc.Sync(context.Background()); err != nil {
					return fmt.Errorf("sync all: %w", err)
				}
				fmt.Println("  All sources synced.")
			}
			return nil
		},
	})

	// Discover command with --force flag
	discoverForce := false
	discoverCmd := &cobra.Command{
		Use:   "discover [<id>]",
		Short: "Discover items in one or all sources",
		Long: `Scan source repositories and report what skills, agents, commands,
MCPs, plugins, harnesses, and configs were discovered.
If a source ID is provided, only that source is scanned.
Otherwise all active sources are scanned.

This command re-scans and re-imports discovered items into the database.
Use --force to clean existing items before re-discovering (useful after scanner changes).

Examples:
  maestro sources discover
  maestro sources discover github-com-user-repo
  maestro sources discover --force`,
		Args: cobra.MaximumNArgs(1),
		RunE: runDiscover(dbPath, &discoverForce),
	}
	discoverCmd.Flags().BoolVar(&discoverForce, "force", false, "Delete existing items before re-discovering")
	cmd.AddCommand(discoverCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "install [id]",
		Short: "Install all items from a source",
		Long: `Create symlinks in opencode directories for all discoverable
	items (skills, agents, commands, MCPs) from the given source.
	If no ID is provided, installs items from all active sources.

	Examples:
	  maestro sources install
	  maestro sources install github-com-user-repo`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			inst := sources.NewInstaller(d)
			if len(args) == 1 {
				fmt.Printf("Installing items from source %s...\n", args[0])
				return inst.InstallAll(args[0])
			}
			svcs, err := d.ListSources()
			if err != nil {
				return err
			}
			for _, src := range svcs {
				fmt.Printf("Installing items from source %s...\n", src.ID)
				if err := inst.InstallAll(src.ID); err != nil {
					fmt.Printf("  Warning: install %s: %v\n", src.ID, err)
				}
			}
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "uninstall [id]",
		Short: "Uninstall all items from a source",
		Long: `Remove symlinks from opencode directories for all installed
	items from the given source.
	If no ID is provided, uninstalls items from all sources.

	Examples:
	  maestro sources uninstall
	  maestro sources uninstall github-com-user-repo`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			inst := sources.NewInstaller(d)
			if len(args) == 1 {
				fmt.Printf("Uninstalling items from source %s...\n", args[0])
				return inst.UninstallAll(args[0])
			}
			svcs, err := d.ListSources()
			if err != nil {
				return err
			}
			for _, src := range svcs {
				fmt.Printf("Uninstalling items from source %s...\n", src.ID)
				if err := inst.UninstallAll(src.ID); err != nil {
					fmt.Printf("  Warning: uninstall %s: %v\n", src.ID, err)
				}
			}
			return nil
		},
	})
	return cmd
}

func runDiscover(dbPath *string, force *bool) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		d, err := openDB(dbPath)
		if err != nil {
			return err
		}
		defer d.Close()
		svc := sources.New(d)

		if len(args) == 1 {
			return discoverSource(svc, d, args[0], *force)
		}
		return discoverAll(svc, d, *force)
	}
}

func discoverSource(svc *sources.Service, d interface {
	GetSource(string) (*models.Source, error)
	ListSourceItemsBySource(string) ([]models.SourceItem, error)
	DeleteSourceItem(string) error
}, id string, force bool) error {
	src, err := d.GetSource(id)
	if err != nil {
		return fmt.Errorf("source %q not found: %w", id, err)
	}
	if force {
		items, _ := d.ListSourceItemsBySource(id)
		for _, it := range items {
			_ = d.DeleteSourceItem(it.ID)
		}
		fmt.Printf("  Cleaned existing items for %s\n", id)
	}
	items, err := svc.DiscoverItems(context.Background(), *src)
	if err != nil {
		return fmt.Errorf("discover %s: %w", id, err)
	}
	printDiscoveryReport(id, items)
	return nil
}

func discoverAll(svc *sources.Service, d interface {
	ListSources() ([]models.Source, error)
	GetSource(string) (*models.Source, error)
	ListSourceItemsBySource(string) ([]models.SourceItem, error)
	DeleteSourceItem(string) error
}, force bool) error {
	list, err := d.ListSources()
	if err != nil {
		return fmt.Errorf("list sources: %w", err)
	}
	if len(list) == 0 {
		fmt.Println("No sources configured")
		return nil
	}
	var totalItems int
	for _, src := range list {
		if force {
			items, _ := d.ListSourceItemsBySource(src.ID)
			for _, it := range items {
				_ = d.DeleteSourceItem(it.ID)
			}
			fmt.Printf("  Cleaned existing items for %s\n", src.ID)
		}
		items, err := svc.DiscoverItems(context.Background(), src)
		if err != nil {
			fmt.Printf("  Warning: discover %s: %v\n", src.ID, err)
			continue
		}
		printDiscoveryReport(src.ID, items)
		totalItems += len(items)
	}
	fmt.Printf("\nTotal: %d items across %d sources\n", totalItems, len(list))
	return nil
}

func printDiscoveryReport(sourceID string, items []models.SourceItem) {
	fmt.Printf("\n=== %s ===\n", sourceID)
	if len(items) == 0 {
		fmt.Println("  No items discovered")
		return
	}
	byType := make(map[string]int)
	for _, item := range items {
		byType[item.Type]++
	}
	for _, t := range []string{"skill", "agent", "command", "mcp", "plugin", "workflow", "harness", "config", "prompt", "rule", "doc"} {
		if n := byType[t]; n > 0 {
			fmt.Printf("  %s: %d\n", t, n)
		}
	}
	fmt.Printf("  Total: %d items\n", len(items))
}
