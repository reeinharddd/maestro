package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize opencode-kit setup",
		Long: `Setup wizard that:
  1. Checks ~/.config/opencode/ exists
  2. Checks opencode.jsonc exists and parses it
  3. Creates opencode.env from opencode.env.example if needed
  4. Seeds the model registry
  5. Runs auto-healing
  6. Prints summary + next steps`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configDir := OpenCodeConfigDir()

			fmt.Println("=== opencode-kit Init ===")
			fmt.Println()

			fmt.Println("[1/6] Config directory...")
			_, err := os.Stat(configDir)
			if os.IsNotExist(err) {
				fmt.Printf("  Creating %s...\n", configDir)
				if err := os.MkdirAll(configDir, 0755); err != nil {
					return fmt.Errorf("create config dir: %w", err)
				}
			} else if err != nil {
				return fmt.Errorf("stat config dir: %w", err)
			} else {
				fmt.Printf("  OK: %s\n", configDir)
			}
			fmt.Println()

			fmt.Println("[2/6] Config file...")
			configPath := filepath.Join(configDir, "opencode.jsonc")
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				fmt.Println("  WARN: opencode.jsonc not found")
				fmt.Println("  Run 'okit generate config' to create it")
			} else {
				data, err := os.ReadFile(configPath)
				if err != nil {
					fmt.Printf("  WARN: cannot read: %v\n", err)
				} else {
					fmt.Printf("  OK: %s (%d bytes)\n", configPath, len(data))
				}
			}
			fmt.Println()

			fmt.Println("[3/6] Environment file...")
			envPath := filepath.Join(configDir, "opencode.env")
			envExample := filepath.Join(configDir, "opencode.env.example")
			if _, err := os.Stat(envPath); os.IsNotExist(err) {
				if _, err := os.Stat(envExample); os.IsNotExist(err) {
					fmt.Println("  WARN: no opencode.env or opencode.env.example found")
				} else {
					data, _ := os.ReadFile(envExample)
					if err := os.WriteFile(envPath, data, 0600); err != nil {
						fmt.Printf("  WARN: cannot create opencode.env: %v\n", err)
					} else {
						fmt.Printf("  Created: %s (fill in your API keys)\n", envPath)
					}
				}
			} else {
				fmt.Printf("  OK: %s\n", envPath)
			}
			fmt.Println()

			fmt.Println("[4/6] Seeding model registry...")
			d, err := openDB(nil)
			if err != nil {
				fmt.Printf("  WARN: open db: %v\n", err)
			} else {
				fmt.Printf("  OK: database at %s\n", d.DBPath())
				d.Close()
			}
			fmt.Println()

			fmt.Println("[5/6] Running auto-heal...")
			d, err = openDB(nil)
			if err != nil {
				fmt.Printf("  WARN: open db: %v\n", err)
			} else {
				report, err := runHeal(d)
				if err != nil {
					fmt.Printf("  WARN: heal: %v\n", err)
				} else if report.IssuesFound > 0 {
					fmt.Printf("  Healed: %d issues found, %d fixed\n", report.IssuesFound, report.IssuesFixed)
				} else {
					fmt.Println("  OK: no issues found")
				}
				d.Close()
			}
			fmt.Println()

			fmt.Println("[6/6] Summary")
			fmt.Printf("  Configuration: %s\n", configDir)
			fmt.Printf("  Config file:   opencode.jsonc %s\n", checkFileExists(configPath))
			fmt.Printf("  Env file:      opencode.env %s\n", checkFileExists(envPath))
			fmt.Println()
			fmt.Println("Next steps:")
			fmt.Println("  1. Edit opencode.env with your API keys")
			fmt.Println("  2. Run 'okit daily' to discover and audit models")
			fmt.Println("  3. Run 'okit doctor' to verify everything works")
			fmt.Println()
			fmt.Printf("Tip: run 'source opencode.env' from %s to load API keys\n", configDir)

			return nil
		},
	}
}

func checkFileExists(path string) string {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return "(not found)"
	}
	if err != nil {
		return fmt.Sprintf("(error: %v)", err)
	}
	return fmt.Sprintf("(%d bytes)", info.Size())
}
