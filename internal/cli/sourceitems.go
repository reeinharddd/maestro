package cli

import (
	"fmt"
	"strings"

	"github.com/reeinharrrd/maestro/internal/sources"
	"github.com/spf13/cobra"
)

func newSourceItemsCmd(dbPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "source-items",
		Short: "Manage source items",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List source items",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			items, err := d.ListSourceItems()
			if err != nil {
				return err
			}
			if len(items) == 0 {
				fmt.Println("No source items found")
				return nil
			}
			fmt.Printf("%-30s %-15s %-10s %s\n", "ID", "Source ID", "Type", "Status")
			fmt.Println(strings.Repeat("-", 70))
			for _, item := range items {
				fmt.Printf("%-30s %-15s %-10s %s\n", item.ID, item.SourceID, item.Type, item.Status)
			}
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "import <registry.json>",
		Short: "Import source registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			svc := sources.New(d)
			if err := svc.ImportSourceRegistry(args[0]); err != nil {
				return err
			}
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "report",
		Short: "Show imported source items",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			items, err := d.ListSourceItems()
			if err != nil {
				return err
			}
			if len(items) == 0 {
				fmt.Printf("Source items: 0\n")
				return nil
			}
			for _, item := range items {
				fmt.Printf("%s | %s | %s | %s\n", item.ID, item.SourceID, item.Type, item.Status)
			}
			return nil
		},
	})
	return cmd
}
