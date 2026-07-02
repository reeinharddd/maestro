package cli

import (
	"github.com/reeinharrrd/maestro/internal/profile"
	"github.com/spf13/cobra"
)

func newProfileCmd(dbPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Profile all active models",
		RunE: func(cmd *cobra.Command, args []string) error {
			full, _ := cmd.Flags().GetBool("full")
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			return profile.New(d).ProfileAll(cmd.Context(), full)
		},
	}
	cmd.Flags().Bool("full", false, "Full detailed profiling")
	return cmd
}
