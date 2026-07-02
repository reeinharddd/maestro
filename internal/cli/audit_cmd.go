package cli

import (
	"github.com/reeinharrrd/maestro/internal/audit"
	"github.com/spf13/cobra"
)

func newAuditCmd(dbPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Test model capabilities via real API calls",
		Long: `Tests every model against its provider API.

By default, skips already-active models (cache hit). Use --full to re-test everything.

Error models are always re-tested — errors may be transient (rate limits, outages).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			full, _ := cmd.Flags().GetBool("full")
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()

			svc := audit.New(d, 5)
			return svc.Run(cmd.Context(), full)
		},
	}
	cmd.Flags().Bool("full", false, "Re-test already-active models too")
	cmd.AddCommand(newAuditLiveCmd(dbPath))
	return cmd
}
