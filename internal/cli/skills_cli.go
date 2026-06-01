package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newSkillsCmd(dbPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Manage skills",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all skills",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			skills, err := d.ListSkills()
			if err != nil {
				return err
			}
			if len(skills) == 0 {
				fmt.Println("No skills found")
				return nil
			}
			fmt.Printf("%-30s %-15s %-10s %s\n", "ID", "Source", "Type", "Status")
			fmt.Println(strings.Repeat("-", 70))
			for _, s := range skills {
				fmt.Printf("%-30s %-15s %-10s %s\n", s.ID, s.Source, s.Type, s.Status)
			}
			return nil
		},
	})
	return cmd
}
