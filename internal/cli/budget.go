package cli

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/reeinharrrd/maestro/pkg/models"
)

func newBudgetCmd(dbPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "budget",
		Short: "Manage daily budget configuration",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show current budget config",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			b, err := d.GetBudget()
			if err != nil {
				return err
			}
			fmt.Printf("Daily Global USD: $%.2f\n", b.DailyGlobalUSD)
			fmt.Printf("Preferred Tier:   %s\n", b.PreferredTier)
			return nil
		},
	})
	setCmd := &cobra.Command{
		Use:   "set",
		Short: "Set budget config",
		RunE: func(cmd *cobra.Command, args []string) error {
			daily, _ := cmd.Flags().GetFloat64("daily")
			tier, _ := cmd.Flags().GetString("tier")
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			b, err := d.GetBudget()
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("daily") {
				b.DailyGlobalUSD = daily
			}
			if cmd.Flags().Changed("tier") {
				b.PreferredTier = tier
			}
			if err := d.UpsertBudget(&models.BudgetConfig{
				ID:             b.ID,
				DailyGlobalUSD: b.DailyGlobalUSD,
				PreferredTier:  b.PreferredTier,
			}); err != nil {
				return err
			}
			fmt.Println("Budget updated.")
			return nil
		},
	}
	setCmd.Flags().Float64("daily", 0, "Daily global budget in USD")
	setCmd.Flags().String("tier", "", "Preferred tier (free_only, budget, quality)")
	cmd.AddCommand(setCmd)
	return cmd
}
