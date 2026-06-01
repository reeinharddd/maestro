package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newLSPServersCmd(dbPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lsp",
		Short: "Manage LSP servers",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List LSP servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			servers, err := d.ListLSPServers()
			if err != nil {
				return err
			}
			if len(servers) == 0 {
				fmt.Println("No LSP servers configured")
				return nil
			}
			fmt.Printf("%-25s %-30s %-12s\n", "ID", "Command", "Disabled")
			fmt.Println(strings.Repeat("-", 70))
			for _, s := range servers {
				dis := ""
				if s.Disabled {
					dis = "yes"
				}
				fmt.Printf("%-25s %-30s %-12s\n", s.ID, s.Command, dis)
			}
			return nil
		},
	})
	return cmd
}
