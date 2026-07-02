package cli

import (
	"fmt"

	"github.com/reeinharrrd/maestro/internal/heal"
	"github.com/spf13/cobra"
)

func newHealCmd(dbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "heal",
		Short: "Run auto-healing checks",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			report, err := heal.New(d).Run(cmd.Context())
			if err != nil {
				return err
			}
			fmt.Printf("Issues found: %d, fixed: %d\n", report.IssuesFound, report.IssuesFixed)
			for _, issue := range report.Issues {
				status := "FIXED"
				if !issue.Fixed {
					status = "WARN"
				}
				fmt.Printf("  [%s] [%s] %s\n", status, issue.Severity, issue.Message)
			}
			return nil
		},
	}
}
