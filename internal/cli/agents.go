package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/reeinharrrd/maestro/pkg/models"
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
	cmd.AddCommand(newAgentAddCmd(dbPath))
	cmd.AddCommand(newAgentUpdateCmd(dbPath))
	return cmd
}

func newAgentAddCmd(dbPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a custom agent",
		Long: `Add an agent and auto-sync to opencode config.

Example:
  maestro agents add --id my-agent --task-type coding --model gpt-4 --description "Coding agent"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			id, _ := cmd.Flags().GetString("id")
			if id == "" {
				return fmt.Errorf("required: --id")
			}

			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()

			a := &models.Agent{
				ID:     id,
				Status: "active",
				Source: "manual",
			}

			if v, _ := cmd.Flags().GetString("task-type"); v != "" {
				a.TaskType = v
			}
			if v, _ := cmd.Flags().GetString("description"); v != "" {
				a.Description = v
			}
			if v, _ := cmd.Flags().GetString("model"); v != "" {
				a.CurrentModelID = v
			}
			if v, _ := cmd.Flags().GetString("fallback-ids"); v != "" {
				a.FallbackIDs = v
			}
			if v, _ := cmd.Flags().GetString("prompt-file"); v != "" {
				a.PromptFile = v
			}
			if v, _ := cmd.Flags().GetFloat64("temperature"); cmd.Flags().Changed("temperature") {
				a.Temperature = v
			}
			if v, _ := cmd.Flags().GetInt("max-steps"); v > 0 {
				a.MaxSteps = v
			}
			if v, _ := cmd.Flags().GetString("permission"); v != "" {
				a.Permission = v
			}
			if v, _ := cmd.Flags().GetString("color"); v != "" {
				a.Color = v
			}
			if v, _ := cmd.Flags().GetString("mode"); v != "" {
				a.Mode = v
			}
			if v, _ := cmd.Flags().GetBool("hidden"); cmd.Flags().Changed("hidden") {
				a.Hidden = v
			}
			if v, _ := cmd.Flags().GetString("source"); v != "" {
				a.Source = v
			}

			if err := d.UpsertAgent(a); err != nil {
				return fmt.Errorf("db insert: %w", err)
			}
			fmt.Printf("Agent %s added to database.\n", id)

			return syncConfig(d)
		},
	}

	flagSet := pflag.NewFlagSet("agent", pflag.ExitOnError)
	flagSet.String("id", "", "Agent ID (required)")
	flagSet.String("task-type", "", "Task type (e.g. coding, reasoning)")
	flagSet.String("description", "", "Agent description")
	flagSet.String("model", "", "Current model ID")
	flagSet.String("fallback-ids", "", "Fallback model IDs (comma-separated)")
	flagSet.String("prompt-file", "", "Path to agent prompt file")
	flagSet.Float64("temperature", 0, "Model temperature")
	flagSet.Int("max-steps", 0, "Maximum steps")
	flagSet.String("permission", "", "Agent permission (JSON)")
	flagSet.String("color", "", "Agent color")
	flagSet.String("mode", "", "Agent mode (subagent, primary, all)")
	flagSet.Bool("hidden", false, "Hide agent from lists")
	flagSet.String("source", "", "Agent source (default: manual)")
	cmd.Flags().AddFlagSet(flagSet)
	return cmd
}

func newAgentUpdateCmd(dbPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update an existing agent",
		Long: `Update agent fields. Only provided flags are changed.
Auto-syncs to opencode config.

Example:
  maestro agents update --id my-agent --model gpt-4-turbo`,
		RunE: func(cmd *cobra.Command, args []string) error {
			id, _ := cmd.Flags().GetString("id")
			if id == "" {
				return fmt.Errorf("required: --id")
			}

			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()

			existing, err := d.GetAgent(id)
			if err != nil {
				return fmt.Errorf("agent %q not found: %w", id, err)
			}

			changed := false
			if v, _ := cmd.Flags().GetString("task-type"); cmd.Flags().Changed("task-type") {
				existing.TaskType = v; changed = true
			}
			if v, _ := cmd.Flags().GetString("description"); cmd.Flags().Changed("description") {
				existing.Description = v; changed = true
			}
			if v, _ := cmd.Flags().GetString("model"); cmd.Flags().Changed("model") {
				existing.CurrentModelID = v; changed = true
			}
			if v, _ := cmd.Flags().GetString("fallback-ids"); cmd.Flags().Changed("fallback-ids") {
				existing.FallbackIDs = v; changed = true
			}
			if v, _ := cmd.Flags().GetString("prompt-file"); cmd.Flags().Changed("prompt-file") {
				existing.PromptFile = v; changed = true
			}
			if v, _ := cmd.Flags().GetFloat64("temperature"); cmd.Flags().Changed("temperature") {
				existing.Temperature = v; changed = true
			}
			if v, _ := cmd.Flags().GetInt("max-steps"); cmd.Flags().Changed("max-steps") {
				existing.MaxSteps = v; changed = true
			}
			if v, _ := cmd.Flags().GetString("permission"); cmd.Flags().Changed("permission") {
				existing.Permission = v; changed = true
			}
			if v, _ := cmd.Flags().GetString("color"); cmd.Flags().Changed("color") {
				existing.Color = v; changed = true
			}
			if v, _ := cmd.Flags().GetString("mode"); cmd.Flags().Changed("mode") {
				existing.Mode = v; changed = true
			}
			if v, _ := cmd.Flags().GetBool("hidden"); cmd.Flags().Changed("hidden") {
				existing.Hidden = v; changed = true
			}
			if v, _ := cmd.Flags().GetString("source"); cmd.Flags().Changed("source") {
				existing.Source = v; changed = true
			}

			if !changed {
				return fmt.Errorf("no fields to update (specify at least one flag)")
			}

			if err := d.UpsertAgent(existing); err != nil {
				return fmt.Errorf("db update: %w", err)
			}
			fmt.Printf("Agent %s updated in database.\n", id)

			return syncConfig(d)
		},
	}

	flagSet := pflag.NewFlagSet("agent", pflag.ExitOnError)
	flagSet.String("id", "", "Agent ID (required)")
	flagSet.String("task-type", "", "Task type (e.g. coding, reasoning)")
	flagSet.String("description", "", "Agent description")
	flagSet.String("model", "", "Current model ID")
	flagSet.String("fallback-ids", "", "Fallback model IDs (comma-separated)")
	flagSet.String("prompt-file", "", "Path to agent prompt file")
	flagSet.Float64("temperature", 0, "Model temperature")
	flagSet.Int("max-steps", 0, "Maximum steps")
	flagSet.String("permission", "", "Agent permission (JSON)")
	flagSet.String("color", "", "Agent color")
	flagSet.String("mode", "", "Agent mode (subagent, primary, all)")
	flagSet.Bool("hidden", false, "Hide agent from lists")
	flagSet.String("source", "", "Agent source")
	cmd.Flags().AddFlagSet(flagSet)
	return cmd
}
