package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newExecLogCmd(dbPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec-log",
		Short: "View execution logs",
	}
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Show recent execution log entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			limit, _ := cmd.Flags().GetInt("limit")
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			logs, err := d.ListExecLogs(limit)
			if err != nil {
				return err
			}
			if len(logs) == 0 {
				fmt.Println("No execution logs found")
				return nil
			}
			fmt.Printf("%-4s %-20s %-30s %-8s %-9s %s\n", "ID", "Agent", "Model", "Tokens", "Duration", "Success")
			fmt.Println(strings.Repeat("-", 85))
			for _, l := range logs {
				succ := "yes"
				if !l.Success {
					succ = "no"
				}
				tokens := fmt.Sprintf("%d+%d", l.TokensIn, l.TokensOut)
				dur := fmt.Sprintf("%dms", l.DurationMs)
				fmt.Printf("%-4d %-20s %-30s %-8s %-9s %s\n", l.ID, l.Agent, l.Model, tokens, dur, succ)
			}
			return nil
		},
	}
	listCmd.Flags().Int("limit", 20, "Number of log entries to show")
	cmd.AddCommand(listCmd)
	return cmd
}
