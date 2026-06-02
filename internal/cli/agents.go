package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newAgentsCmd(dbPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agents",
		Short: "Manage agents",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			agents, err := d.ListAgents()
			if err != nil {
				return err
			}
			if len(agents) == 0 {
				fmt.Println("No agents found")
				return nil
			}
			fmt.Printf("%-20s %-12s %-10s %s\n", "ID", "Model", "Status", "Task Type")
			fmt.Println(strings.Repeat("-", 60))
			for _, a := range agents {
				model := a.CurrentModelID
				if model == "" {
					model = "-"
				}
				fmt.Printf("%-20s %-12s %-10s %s\n", a.ID, model, a.Status, a.TaskType)
			}
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "get",
		Short: "Show agent details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			a, err := d.GetAgent(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("ID:              %s\n", a.ID)
			fmt.Printf("Task Type:       %s\n", a.TaskType)
			fmt.Printf("Description:     %s\n", a.Description)
			fmt.Printf("Model:           %s\n", a.CurrentModelID)
			fmt.Printf("Fallbacks:       %s\n", a.FallbackIDs)
			fmt.Printf("Prompt File:     %s\n", a.PromptFile)
			fmt.Printf("Temperature:     %.2f\n", a.Temperature)
			fmt.Printf("Max Steps:       %d\n", a.MaxSteps)
			fmt.Printf("Permission:      %s\n", a.Permission)
			fmt.Printf("Color:           %s\n", a.Color)
			fmt.Printf("Mode:            %s\n", a.Mode)
			fmt.Printf("Hidden:          %t\n", a.Hidden)
			fmt.Printf("Status:          %s\n", a.Status)
			fmt.Printf("Source:          %s\n", a.Source)
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "delete",
		Short: "Delete an agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			if err := d.DeleteAgent(args[0]); err != nil {
				return err
			}
			fmt.Printf("Agent %q deleted.\n", args[0])
			return nil
		},
	})
	return cmd
}
