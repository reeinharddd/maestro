package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/reeinharrrd/maestro/pkg/models"
)

func newLSPServersCmd(dbPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lsp",
		Short: "Manage LSP servers",
	}

	cmd.AddCommand(newLSPListCmd(dbPath))
	cmd.AddCommand(newLSPAddCmd(dbPath))
	cmd.AddCommand(newLSPUpdateCmd(dbPath))
	cmd.AddCommand(newLSPRemoveCmd(dbPath))

	cmd.PersistentFlags().String("id", "", "LSP server ID")
	cmd.PersistentFlags().String("command", "", `Command (JSON array, e.g. '["typescript-language-server","--stdio"]')`)
	cmd.PersistentFlags().String("extensions", "", "Comma-separated file extensions (optional)")
	cmd.PersistentFlags().String("env", "", "Environment variables JSON (optional)")
	cmd.PersistentFlags().String("init", "", "Initialization options JSON (optional)")
	cmd.PersistentFlags().Bool("disabled", false, "Disable this LSP server")

	return cmd
}

func newLSPListCmd(dbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all LSP servers",
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
	}
}

func newLSPAddCmd(dbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "add",
		Short: "Add an LSP server",
		Long: `Add an LSP server configuration.

Example:
  maestro lsp add --id typescript --command '["typescript-language-server","--stdio"]' --extensions ".ts,.tsx"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			id, _ := cmd.Flags().GetString("id")
			command, _ := cmd.Flags().GetString("command")
			if id == "" || command == "" {
				return fmt.Errorf("required: --id, --command")
			}

			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()

			l := &models.LSPServer{
				ID:      id,
				Command: command,
			}

			if v, _ := cmd.Flags().GetString("extensions"); v != "" {
				l.Extensions = v
			}
			if v, _ := cmd.Flags().GetString("env"); v != "" {
				l.Env = v
			}
			if v, _ := cmd.Flags().GetString("init"); v != "" {
				l.Initialization = v
			}
			if v, _ := cmd.Flags().GetBool("disabled"); cmd.Flags().Changed("disabled") {
				l.Disabled = v
			}

			if err := d.UpsertLSPServer(l); err != nil {
				return fmt.Errorf("db insert: %w", err)
			}
			fmt.Printf("LSP server %s added to database.\n", id)
			return nil
		},
	}
}

func newLSPUpdateCmd(dbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update an existing LSP server",
		Long: `Update LSP server fields. Only provided flags are changed.

Example:
  maestro lsp update --id typescript --command '["typescript-language-server","--stdio"]'`,
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

			existing, err := d.GetLSPServer(id)
			if err != nil {
				return fmt.Errorf("LSP server %q not found: %w", id, err)
			}

			changed := false
			if v, _ := cmd.Flags().GetString("command"); cmd.Flags().Changed("command") {
				existing.Command = v
				changed = true
			}
			if v, _ := cmd.Flags().GetString("extensions"); cmd.Flags().Changed("extensions") {
				existing.Extensions = v
				changed = true
			}
			if v, _ := cmd.Flags().GetString("env"); cmd.Flags().Changed("env") {
				existing.Env = v
				changed = true
			}
			if v, _ := cmd.Flags().GetString("init"); cmd.Flags().Changed("init") {
				existing.Initialization = v
				changed = true
			}
			if v, _ := cmd.Flags().GetBool("disabled"); cmd.Flags().Changed("disabled") {
				existing.Disabled = v
				changed = true
			}

			if !changed {
				return fmt.Errorf("no fields to update (specify at least one flag)")
			}

			if err := d.UpsertLSPServer(existing); err != nil {
				return fmt.Errorf("db update: %w", err)
			}
			fmt.Printf("LSP server %s updated in database.\n", id)
			return nil
		},
	}
}

func newLSPRemoveCmd(dbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "remove",
		Short: "Remove an LSP server",
		Long: `Remove an LSP server from the database.

Example:
  maestro lsp remove --id typescript`,
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

			if _, err := d.GetLSPServer(id); err != nil {
				return fmt.Errorf("LSP server %q not found", id)
			}
			if err := d.DeleteLSPServer(id); err != nil {
				return fmt.Errorf("db delete: %w", err)
			}
			fmt.Printf("LSP server %s removed from database.\n", id)
			return nil
		},
	}
}
