package cli

import (
	"fmt"
	"strings"

	"github.com/reeinharrrd/maestro/pkg/models"
	"github.com/spf13/cobra"
)

func newCommandsCmdImpl(dbPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "commands",
		Short: "Manage slash commands (DB auto-sync)",
		Long: `Manage OpenCode slash commands. Every change auto-syncs to opencode config
so both DB and config stay in sync.`,
	}

	cmd.AddCommand(newCommandListCmd(dbPath))
	cmd.AddCommand(newCommandAddCmd(dbPath))
	cmd.AddCommand(newCommandUpdateCmd(dbPath))
	cmd.AddCommand(newCommandRemoveCmd(dbPath))
	cmd.PersistentFlags().String("id", "", "Command ID (e.g. /my-command)")
	cmd.PersistentFlags().String("template", "", "Command template/prompt")
	cmd.PersistentFlags().String("description", "", "Command description")
	cmd.PersistentFlags().String("agent", "", "Agent to assign")
	cmd.PersistentFlags().String("model", "", "Model override")
	cmd.PersistentFlags().Bool("subtask", false, "Subtask mode")
	cmd.PersistentFlags().String("source", "manual", "Command source")
	cmd.PersistentFlags().String("status", "active", "Command status")

	return cmd
}

func newCommandListCmd(dbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()

			commands, err := d.ListCommands()
			if err != nil {
				return err
			}

			fmt.Printf("%-30s %-20s %-15s %-8s\n", "ID", "Agent", "Model", "Status")
			fmt.Println(strings.Repeat("-", 80))
			for _, c := range commands {
				fmt.Printf("%-30s %-20s %-15s %-8s\n", c.ID, c.Agent, c.Model, c.Status)
			}
			return nil
		},
	}
}

func newCommandAddCmd(dbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "add",
		Short: "Add a custom command",
		Long: `Add a command and auto-sync to opencode config.

Example:
  maestro commands add --id /my-command --template "Do something with {input}" --agent my-agent`,
		RunE: func(cmd *cobra.Command, args []string) error {
			id, _ := cmd.Flags().GetString("id")
			template, _ := cmd.Flags().GetString("template")
			if id == "" || template == "" {
				return fmt.Errorf("required: --id, --template")
			}

			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()

			c := &models.Command{
				ID:       id,
				Template: template,
				Source:   "manual",
				Status:   "active",
			}

			if desc, _ := cmd.Flags().GetString("description"); desc != "" {
				c.Description = desc
			}
			if agent, _ := cmd.Flags().GetString("agent"); agent != "" {
				c.Agent = agent
			}
			if model, _ := cmd.Flags().GetString("model"); model != "" {
				c.Model = model
			}
			if cmd.Flags().Changed("subtask") {
				c.Subtask, _ = cmd.Flags().GetBool("subtask")
			}
			if src, _ := cmd.Flags().GetString("source"); src != "manual" {
				c.Source = src
			}
			if st, _ := cmd.Flags().GetString("status"); st != "active" {
				c.Status = st
			}

			if err := d.UpsertCommand(c); err != nil {
				return fmt.Errorf("db insert: %w", err)
			}
			fmt.Printf("Command %s added to database.\n", id)

			return syncConfig(d)
		},
	}
}

func newCommandUpdateCmd(dbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update an existing command",
		Long: `Update command fields. Only provided flags are changed.
Auto-syncs to opencode config.

Example:
  maestro commands update --id /my-command --agent new-agent`,
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

			all, err := d.ListCommands()
			if err != nil {
				return err
			}
			var existing *models.Command
			for i := range all {
				if all[i].ID == id {
					existing = &all[i]
					break
				}
			}
			if existing == nil {
				return fmt.Errorf("command %q not found", id)
			}

			changed := false
			if v, _ := cmd.Flags().GetString("template"); cmd.Flags().Changed("template") {
				existing.Template = v
				changed = true
			}
			if v, _ := cmd.Flags().GetString("description"); cmd.Flags().Changed("description") {
				existing.Description = v
				changed = true
			}
			if v, _ := cmd.Flags().GetString("agent"); cmd.Flags().Changed("agent") {
				existing.Agent = v
				changed = true
			}
			if v, _ := cmd.Flags().GetString("model"); cmd.Flags().Changed("model") {
				existing.Model = v
				changed = true
			}
			if cmd.Flags().Changed("subtask") {
				existing.Subtask, _ = cmd.Flags().GetBool("subtask")
				changed = true
			}
			if v, _ := cmd.Flags().GetString("source"); cmd.Flags().Changed("source") {
				existing.Source = v
				changed = true
			}
			if v, _ := cmd.Flags().GetString("status"); cmd.Flags().Changed("status") {
				existing.Status = v
				changed = true
			}

			if !changed {
				return fmt.Errorf("no fields to update (specify at least one flag)")
			}

			if err := d.UpsertCommand(existing); err != nil {
				return fmt.Errorf("db update: %w", err)
			}
			fmt.Printf("Command %s updated in database.\n", id)

			return syncConfig(d)
		},
	}
}

func newCommandRemoveCmd(dbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "remove",
		Short: "Remove a command",
		Long: `Remove a command from DB.
Auto-syncs to opencode config.

Example:
  maestro commands remove --id /my-command`,
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

			all, err := d.ListCommands()
			if err != nil {
				return err
			}
			found := false
			for _, c := range all {
				if c.ID == id {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("command %q not found", id)
			}

			if err := d.DeleteCommand(id); err != nil {
				return fmt.Errorf("db delete: %w", err)
			}
			fmt.Printf("Command %s removed from database.\n", id)

			return syncConfig(d)
		},
	}
}
