package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
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
