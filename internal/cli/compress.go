package cli

import (
	"encoding/json"
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
			d, err := openDB(nil)
			if err != nil {
				return err
			}
			defer d.Close()
			c := compress.NewWithDB(d, 6)
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
		Use:   "report",
		Short: "Show recent compressed fragments",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(nil)
			if err != nil {
				return err
			}
			defer d.Close()
			fragments, err := d.ListConfigFragments(10)
			if err != nil {
				return err
			}
			if len(fragments) == 0 {
				fmt.Println("No compressed fragments found")
				return nil
			}
			fmt.Printf("%-24s %-12s %-12s %s\n", "ID", "Type", "Source", "Content")
			fmt.Println(strings.Repeat("-", 80))
			for _, f := range fragments {
				preview := f.Content
				if len(preview) > 48 {
					preview = preview[:48] + "..."
				}
				fmt.Printf("%-24s %-12s %-12s %s\n", f.ID, f.ConfigType, f.Source, preview)
			}
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "save <content> <type>",
		Short: "Persist a compressed fragment",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(nil)
			if err != nil {
				return err
			}
			defer d.Close()
			payload := map[string]string{"content": args[0], "type": args[1]}
			raw, _ := json.Marshal(payload)
			c := compress.NewWithDB(d, 12)
			out := c.Compress([]compress.Observation{{Source: "cli", Step: 1, Message: string(raw), Important: true}})
			fmt.Println(out)
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
