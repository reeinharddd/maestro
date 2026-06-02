package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/reeinharrrd/opencode-kit/internal/util"
	"github.com/spf13/cobra"
)

var knownMCPServers = []string{
	"context7", "engram", "agentmemory", "firecrawl", "opensearch", "github",
	"exa", "git-mcp", "filesystem", "sequential-thinking", "angular-cli",
	"upgrade-pilot", "token-optimizer", "browserbase", "evalview",
	"fal-ai", "magic", "railway", "playwright", "docker-mcp", "confluence",
	"memory", "postgres-wedo", "supabase", "browserbase", "longhand",
	"ecc-memory", "ecc-playwright", "ecc-railway", "ecc-token-optimizer",
}

func newMcpProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage MCP server profiles",
		Long: `Manage MCP server profiles in opencode.jsonc.
Sets enabled/disabled flags on individual MCP servers.

Profiles:
  light     - No MCP servers enabled
  balanced  - filesystem, sequential-thinking, git-mcp, token-optimizer
  full      - All known MCP servers`,
	}

	setCmd := &cobra.Command{
		Use:   "set {light|balanced|full}",
		Short: "Set MCP profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profile := args[0]
			var enabled []string
			switch profile {
			case "light":
			case "balanced":
				enabled = []string{"filesystem", "sequential-thinking", "git-mcp", "token-optimizer"}
			case "full":
				enabled = knownMCPServers
			default:
				return fmt.Errorf("unknown profile: %s (use light, balanced, or full)", profile)
			}
			return applyMCPProfile(profile, enabled)
		},
	}

	enableCmd := &cobra.Command{
		Use:   "enable <name>...",
		Short: "Enable one or more MCP servers",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return setMCPEnabled(args, true)
		},
	}

	disableCmd := &cobra.Command{
		Use:   "disable <name>...",
		Short: "Disable one or more MCP servers",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return setMCPEnabled(args, false)
		},
	}

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show MCP server status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showMCPStatus()
		},
	}

	reportCmd := &cobra.Command{
		Use:   "report",
		Short: "Show MCP profile/import report",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := readConfigJSON()
			if err != nil {
				return err
			}
			fmt.Printf("Config keys: %d\n", len(cfg))
			if mcps, ok := cfg["mcp"].(map[string]interface{}); ok {
				fmt.Printf("MCP servers: %d\n", len(mcps))
			}
			return nil
		},
	}

	cmd.AddCommand(setCmd, enableCmd, disableCmd, statusCmd, reportCmd)
	return cmd
}

func opencodeConfigPath() string {
	return OpenCodeConfigPath()
}

func readConfigJSON() (map[string]interface{}, error) {
	path := opencodeConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	cleaned := util.StripJSONC(data)
	var cfg map[string]interface{}
	if err := json.Unmarshal(cleaned, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return cfg, nil
}

func writeConfigJSON(cfg map[string]interface{}) error {
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(opencodeConfigPath(), out, 0644)
}

func applyMCPProfile(profile string, enabled []string) error {
	cfg, err := readConfigJSON()
	if err != nil {
		return err
	}

	mcps, _ := cfg["mcp"].(map[string]interface{})
	if mcps == nil {
		return fmt.Errorf("no mcp section found in config")
	}

	enabledSet := make(map[string]bool)
	for _, name := range enabled {
		enabledSet[name] = true
	}

	var actual []string
	for name, entry := range mcps {
		if entryMap, ok := entry.(map[string]interface{}); ok {
			entryMap["enabled"] = enabledSet[name]
			if enabledSet[name] {
				actual = append(actual, name)
			}
		}
	}

	sort.Strings(actual)

	if err := writeConfigJSON(cfg); err != nil {
		return err
	}

	fmt.Printf("MCP profile: %s (%d enabled)\n", profile, len(actual))
	if len(actual) > 0 {
		fmt.Println("Enabled: " + joinStrings(actual, ", "))
	}
	return nil
}

func setMCPEnabled(names []string, enabled bool) error {
	cfg, err := readConfigJSON()
	if err != nil {
		return err
	}

	mcps, _ := cfg["mcp"].(map[string]interface{})
	if mcps == nil {
		return fmt.Errorf("no mcp section found in config")
	}

	var missing []string
	for _, name := range names {
		if entry, ok := mcps[name]; ok {
			if entryMap, ok := entry.(map[string]interface{}); ok {
				entryMap["enabled"] = enabled
			}
		} else {
			missing = append(missing, name)
		}
	}

	if err := writeConfigJSON(cfg); err != nil {
		return err
	}

	action := "Enabled"
	if !enabled {
		action = "Disabled"
	}
	for _, name := range names {
		found := true
		for _, m := range missing {
			if m == name {
				found = false
				break
			}
		}
		if found {
			fmt.Printf("%s: %s\n", action, name)
		}
	}
	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "Warning: not found in config: %s\n", joinStrings(missing, ", "))
	}
	return nil
}

func showMCPStatus() error {
	cfg, err := readConfigJSON()
	if err != nil {
		return err
	}

	mcps, _ := cfg["mcp"].(map[string]interface{})
	if mcps == nil {
		fmt.Println("No MCP servers configured")
		return nil
	}

	var names []string
	for name := range mcps {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		entry := mcps[name]
		if entryMap, ok := entry.(map[string]interface{}); ok {
			enabled, _ := entryMap["enabled"].(bool)
			status := "false"
			if enabled {
				status = "true"
			}
			fmt.Printf("%s: %s\n", name, status)
		}
	}
	return nil
}

func joinStrings(items []string, sep string) string {
	result := ""
	for i, item := range items {
		if i > 0 {
			result += sep
		}
		result += item
	}
	return result
}
