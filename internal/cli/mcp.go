package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/reeinharrrd/maestro/pkg/models"
)

func newMcpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage MCP servers",
	}

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start an MCP server",
	}
	startCmd.AddCommand(&cobra.Command{
		Use:   "filesystem [roots...]",
		Short: "Start the filesystem MCP server",
		Long: `Start @modelcontextprotocol/server-filesystem with auto-detected roots.
If no roots given, auto-detects in order:
  1. $OPENCODE_MCP_FILESYSTEM_ROOTS env var
  2. git rev-parse --show-toplevel (if in a git repo)
  3. Current working directory`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var roots []string
			if len(args) > 0 {
				roots = args
			} else {
				roots = detectFilesystemRoots()
			}

			if len(roots) == 0 {
				pwd, _ := os.Getwd()
				roots = []string{pwd}
			}

			fmt.Printf("Starting filesystem MCP with roots: %s\n", strings.Join(roots, ", "))

			npxArgs := append([]string{"-y", "@modelcontextprotocol/server-filesystem"}, roots...)
			xcmd := exec.Command("bunx", npxArgs...)
			xcmd.Stdout = os.Stdout
			xcmd.Stderr = os.Stderr
			xcmd.Stdin = os.Stdin

			return xcmd.Run()
		},
	})

	cmd.AddCommand(startCmd)

	// MCP profile subcommand (absorb mcp-profile.sh)
	cmd.AddCommand(newMcpProfileCmd())

	return cmd
}

func detectFilesystemRoots() []string {
	roots := os.Getenv("OPENCODE_MCP_FILESYSTEM_ROOTS")
	if roots != "" {
		parts := strings.Split(roots, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		return parts
	}

	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err == nil {
		dir := strings.TrimSpace(string(out))
		if dir != "" {
			return []string{dir}
		}
	}

	pwd, _ := os.Getwd()
	if pwd != "" {
		return []string{pwd}
	}

	return nil
}

// ─── DB-backed MCP server management (mcps) ────────────────────────────

func newMCPServersCmd(dbPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcps",
		Short: "Manage MCP servers (DB auto-sync)",
		Long: `Manage MCP servers. Every change automatically syncs to opencode config
so both DB and config stay in sync.`,
	}

	cmd.AddCommand(newMCPServerListCmd(dbPath))
	cmd.AddCommand(newMCPServerAddCmd(dbPath))
	cmd.AddCommand(newMCPServerUpdateCmd(dbPath))
	cmd.AddCommand(newMCPServerRemoveCmd(dbPath))

	cmd.PersistentFlags().String("id", "", "MCP server ID")
	cmd.PersistentFlags().String("type", "", "Type (stdio/url)")
	cmd.PersistentFlags().String("command", "", "Command for stdio type (JSON array)")
	cmd.PersistentFlags().String("url", "", "URL for URL type")
	cmd.PersistentFlags().Bool("enabled", true, "Server enabled")
	cmd.PersistentFlags().String("env-vars", "", "Environment variables (JSON map)")
	cmd.PersistentFlags().Int("timeout", 60, "Timeout in seconds")

	return cmd
}

func newMCPServerListCmd(dbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all MCP servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()

			servers, err := d.ListMCPs()
			if err != nil {
				return err
			}

			fmt.Printf("%-20s %-8s %-30s %-8s %-8s %-10s\n", "ID", "Type", "Command/URL", "Enabled", "Timeout", "Source")
			fmt.Println(strings.Repeat("-", 90))
			for _, s := range servers {
				en := "yes"
				if !s.Enabled {
					en = "no"
				}
				display := s.Command
				if display == "" {
					display = s.URL
				}
				fmt.Printf("%-20s %-8s %-30s %-8s %-8d %-10s\n", s.ID, s.Type, display, en, s.Timeout/1000, s.Source)
			}
			return nil
		},
	}
}

func newMCPServerAddCmd(dbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "add",
		Short: "Add an MCP server",
		Long: `Add an MCP server and auto-sync to opencode config.

Example:
  maestro mcps add --id my-server --type stdio --command '["npx","-y","pkg"]'
  maestro mcps add --id my-server --type url --url http://localhost:3000/mcp`,
		RunE: func(cmd *cobra.Command, args []string) error {
			id, _ := cmd.Flags().GetString("id")
			mcpType, _ := cmd.Flags().GetString("type")
			if id == "" || mcpType == "" {
				return fmt.Errorf("required: --id, --type")
			}
			command, _ := cmd.Flags().GetString("command")
			url, _ := cmd.Flags().GetString("url")
			if mcpType == "stdio" && command == "" {
				return fmt.Errorf("--command required for stdio type")
			}
			if mcpType == "url" && url == "" {
				return fmt.Errorf("--url required for url type")
			}

			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()

			enabled, _ := cmd.Flags().GetBool("enabled")
			timeout, _ := cmd.Flags().GetInt("timeout")
			envVars, _ := cmd.Flags().GetString("env-vars")

			// Map user-facing type values to DB enum
			dbType := mcpType
			if mcpType == "stdio" {
				dbType = "local"
			} else if mcpType == "url" {
				dbType = "remote"
			}

			m := &models.MCPServer{
				ID:      id,
				Type:    dbType,
				Command: command,
				URL:     url,
				Enabled: enabled,
				EnvVars: envVars,
				Timeout: timeout * 1000, // seconds -> ms
				Source:  "manual",
			}

			if err := d.UpsertMCP(m); err != nil {
				return fmt.Errorf("db insert: %w", err)
			}
			fmt.Printf("MCP server %s added to database.\n", id)

			return syncConfig(d)
		},
	}
}

func newMCPServerUpdateCmd(dbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update an existing MCP server",
		Long: `Update MCP server fields. Only provided flags are changed.
Auto-syncs to opencode config.

Example:
  maestro mcps update --id my-server --timeout 120
  maestro mcps update --id my-server --type stdio --command '["npx","-y","pkg"]'`,
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

			servers, err := d.ListMCPs()
			if err != nil {
				return err
			}
			var existing *models.MCPServer
			for i := range servers {
				if servers[i].ID == id {
					existing = &servers[i]
					break
				}
			}
			if existing == nil {
				return fmt.Errorf("MCP server %q not found", id)
			}

			changed := false
			if mcpType, _ := cmd.Flags().GetString("type"); cmd.Flags().Changed("type") {
				dbType := mcpType
				if mcpType == "stdio" {
					dbType = "local"
				} else if mcpType == "url" {
					dbType = "remote"
				}
				existing.Type = dbType
				changed = true
			}
			if v, _ := cmd.Flags().GetString("command"); cmd.Flags().Changed("command") {
				existing.Command = v
				changed = true
			}
			if v, _ := cmd.Flags().GetString("url"); cmd.Flags().Changed("url") {
				existing.URL = v
				changed = true
			}
			if v, _ := cmd.Flags().GetBool("enabled"); cmd.Flags().Changed("enabled") {
				existing.Enabled = v
				changed = true
			}
			if v, _ := cmd.Flags().GetString("env-vars"); cmd.Flags().Changed("env-vars") {
				existing.EnvVars = v
				changed = true
			}
			if v, _ := cmd.Flags().GetInt("timeout"); cmd.Flags().Changed("timeout") {
				existing.Timeout = v * 1000
				changed = true
			}

			if !changed {
				return fmt.Errorf("no fields to update (specify at least one flag)")
			}

			if err := d.UpsertMCP(existing); err != nil {
				return fmt.Errorf("db update: %w", err)
			}
			fmt.Printf("MCP server %s updated in database.\n", id)

			return syncConfig(d)
		},
	}
}

func newMCPServerRemoveCmd(dbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "remove",
		Short: "Remove an MCP server",
		Long: `Remove an MCP server from DB.
Auto-syncs to opencode config.

Example:
  maestro mcps remove --id my-server`,
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

			// Verify it exists before deleting
			servers, err := d.ListMCPs()
			if err != nil {
				return err
			}
			found := false
			for _, s := range servers {
				if s.ID == id {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("MCP server %q not found", id)
			}

			if err := d.DeleteMCP(id); err != nil {
				return fmt.Errorf("db delete: %w", err)
			}
			fmt.Printf("MCP server %s removed from database.\n", id)

			return syncConfig(d)
		},
	}
}
