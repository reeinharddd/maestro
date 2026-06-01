package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newSnapshotsCmd(dbPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshots",
		Short: "Manage DB snapshots",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List recent snapshots",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			snapshots, err := d.ListSnapshots(10)
			if err != nil {
				return err
			}
			if len(snapshots) == 0 {
				fmt.Println("No snapshots found")
				return nil
			}
			fmt.Printf("%-6s %-20s %-30s\n", "ID", "Hash", "Created At")
			fmt.Println(strings.Repeat("-", 60))
			for _, s := range snapshots {
				prefix := s.Hash
				if len(prefix) > 16 {
					prefix = prefix[:16]
				}
				fmt.Printf("%-6d %-20s %-30s\n", s.ID, prefix, s.CreatedAt)
			}
			return nil
		},
	})
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show snapshot content",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			var id int64
			if _, err := fmt.Sscanf(args[0], "%d", &id); err != nil {
				return fmt.Errorf("invalid snapshot id: %s", args[0])
			}
			s, err := d.GetSnapshot(id)
			if err != nil {
				return err
			}
			fmt.Printf("ID:        %d\n", s.ID)
			fmt.Printf("Hash:      %s\n", s.Hash)
			fmt.Printf("Created:   %s\n", s.CreatedAt)
			fmt.Printf("Content:\n%s\n", s.Content)
			return nil
		},
	}
	cmd.AddCommand(showCmd)
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a snapshot",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			var id int64
			if _, err := fmt.Sscanf(args[0], "%d", &id); err != nil {
				return fmt.Errorf("invalid snapshot id: %s", args[0])
			}
			if err := d.DeleteSnapshot(id); err != nil {
				return err
			}
			fmt.Printf("Snapshot %d deleted.\n", id)
			return nil
		},
	}
	cmd.AddCommand(deleteCmd)
	return cmd
}
