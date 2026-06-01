package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newModelProfilesCmd(dbPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profiles",
		Short: "View model profiles",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all model profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			profiles, err := d.ListModelProfiles()
			if err != nil {
				return err
			}
			if len(profiles) == 0 {
				fmt.Println("No model profiles found")
				return nil
			}
			fmt.Printf("%-35s %-10s %-11s %s\n", "Model ID", "Stream TPS", "Real Ctx", "Profiled At")
			fmt.Println(strings.Repeat("-", 70))
			for _, p := range profiles {
				tps := fmt.Sprintf("%.1f", p.StreamTPS)
				ctx := fmt.Sprintf("%dK", p.RealContext/1000)
				fmt.Printf("%-35s %-10s %-11s %d\n", p.ModelID, tps, ctx, p.ProfiledAt)
			}
			return nil
		},
	})
	return cmd
}
