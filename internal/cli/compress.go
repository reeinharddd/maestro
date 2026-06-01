package cli

import (
	"fmt"
	"strings"

	"github.com/reeinharrrd/opencode-kit/internal/compress"
	"github.com/spf13/cobra"
)

func newCompressCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compress",
		Short: "Compress session observations",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "demo",
		Short: "Show a compression example",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := compress.New(6)
			out := c.Compress([]compress.Observation{
				{Source: "cli", Step: 1, Message: "starting session"},
				{Source: "db", Step: 2, Message: "warning: backup failed"},
				{Source: "route", Step: 3, Message: "fallback chain selected", Important: true},
			})
			if out == "" {
				return fmt.Errorf("compression produced no output")
			}
			fmt.Println(strings.TrimSpace(out))
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "prune <text>",
		Short: "Prune command output",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := compress.PruneOutput(args[0])
			fmt.Println(out)
			return nil
		},
	})

	return cmd
}
