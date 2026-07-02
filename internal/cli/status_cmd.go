package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newStatusCmd(dbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show system status",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()

			stats, err := d.GetStats()
			if err != nil {
				return err
			}

			fmt.Println("=== opencode-kit Status ===")
			fmt.Printf("DB: %s\n", d.DBPath())
			fmt.Println()
			fmt.Println("Models by status:")
			for _, s := range []string{"active", "error", "untested", "deprecated", "paid"} {
				if c, ok := stats[s]; ok {
					fmt.Printf("  %s: %d\n", s, c)
				}
			}
			if p, ok := stats["providers_active"]; ok {
				fmt.Printf("\nActive providers: %d\n", p)
			}

			providers, err := d.ListProviders()
			if err == nil && len(providers) > 0 {
				fmt.Println("\nProviders:")
				for _, p := range providers {
					fmt.Printf("  %s (status: %s)\n", p.ID, p.Status)
				}
			}
			return nil
		},
	}
}
