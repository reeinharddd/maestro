package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newModelsViewCmd(dbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "models-view",
		Short: "View model registry entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()

			models, err := d.ListModels()
			if err != nil {
				return err
			}
			if len(models) == 0 {
				fmt.Println("No models found")
				return nil
			}

			fmt.Printf("%-35s %-15s %-8s %-8s %-8s %-8s\n", "ID", "Provider", "Ctx", "FC", "Vision", "Tier")
			fmt.Println(strings.Repeat("-", 90))
			for _, m := range models {
				fc := "-"
				if m.FunctionCalling {
					fc = "yes"
				}
				vision := "-"
				if m.Vision {
					vision = "yes"
				}
				fmt.Printf("%-35s %-15s %-8d %-8s %-8s %-8s\n", m.ID, m.ProviderID, m.ContextWindow, fc, vision, m.Tier)
			}
			return nil
		},
	}
}
