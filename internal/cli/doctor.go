package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/reeinharrrd/opencode-kit/internal/util"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Comprehensive system validation",
		Long:  `Runs a full system check: binary, config, database, and optionally keys.`,
	}
	withKeys := cmd.Flags().Bool("with-keys", false, "Also check API keys")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		fail := 0

		fmt.Println("=== opencode-kit Doctor ===")
		fmt.Println()

		fmt.Println("[1/5] Binary check...")
		if err := checkBinary(); err != nil {
			fmt.Printf("  FAIL: %v\n", err)
			fail++
		} else {
			fmt.Println("  OK")
		}
		fmt.Println()

		fmt.Println("[2/5] Config check...")
		if err := checkConfig(); err != nil {
			fmt.Printf("  FAIL: %v\n", err)
			fail++
		} else {
			fmt.Println("  OK")
		}
		fmt.Println()

		fmt.Println("[3/5] Database check...")
		configDir := OpenCodeConfigDir()
		dbPath := OpenCodeDBPath()
		if err := checkDatabase(dbPath); err != nil {
			fmt.Printf("  FAIL: %v\n", err)
		}
		fmt.Println()

		fmt.Println("[4/5] Status check...")
		if err := checkStatus(); err != nil {
			fmt.Printf("  FAIL: %v\n", err)
			fail++
		} else {
			fmt.Println("  OK")
		}
		fmt.Println()

		if *withKeys {
			fmt.Println("[5/5] Keys check...")
			envPath := filepath.Join(configDir, "opencode.env")
			if _, err := os.Stat(envPath); os.IsNotExist(err) {
				fmt.Println("  WARN: opencode.env not found")
			} else {
				envVars, _ := parseEnvFile(envPath)
				missing := 0
				for _, k := range expectedKeys {
					val := envVars[k.EnvVar]
					if val == "" {
						val = os.Getenv(k.EnvVar)
					}
					if val == "" {
						fmt.Printf("  MISS: %s (%s)\n", k.EnvVar, k.Name)
						missing++
					}
				}
				if missing == 0 {
					fmt.Println("  OK: all keys present")
				} else {
					fmt.Printf("  %d key(s) missing\n", missing)
				}
			}
			fmt.Println()
		}

		if fail > 0 {
			fmt.Printf("Doctor: %d check(s) FAILED\n", fail)
			return fmt.Errorf("%d check(s) failed", fail)
		}
		fmt.Println("Doctor: ALL CHECKS PASSED")
		return nil
	}

	return cmd
}

func checkBinary() error {
	path, err := exec.LookPath("okit")
	if err != nil {
		return fmt.Errorf("okit not found in PATH")
	}
	out, err := exec.Command(path, "--help").Output()
	if err == nil {
		first := strings.SplitN(string(out), "\n", 2)[0]
		fmt.Printf("  Binary: %s\n", path)
		fmt.Printf("  Info:   %s\n", first)
	} else {
		fmt.Printf("  Binary: %s\n", path)
	}
	return nil
}

func checkConfig() error {
	configPath := OpenCodeConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read opencode.jsonc: %w", err)
	}

	cleaned := util.StripJSONC(data)
	var cfg map[string]interface{}
	if err := json.Unmarshal(cleaned, &cfg); err != nil {
		return fmt.Errorf("parse opencode.jsonc: %w", err)
	}

	sections := []string{"provider", "agent", "command", "mcp", "permission", "experimental"}
	missing := []string{}
	present := 0
	for _, s := range sections {
		if _, ok := cfg[s]; ok {
			present++
		} else {
			missing = append(missing, s)
		}
	}
	fmt.Printf("  File:  %s\n", configPath)
	fmt.Printf("  Size:  %d bytes\n", len(data))
	fmt.Printf("  Sections: %d/%d critical sections present\n", present, len(sections))
	if len(missing) > 0 {
		fmt.Printf("  Missing: %s\n", strings.Join(missing, ", "))
	}

	return nil
}

func checkDatabase(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		fmt.Println("  WARN: opencode-kit.db not found (run 'okit daily' to initialize)")
		return nil
	}
	if err != nil {
		return fmt.Errorf("stat db: %w", err)
	}
	size := info.Size()
	fmt.Printf("  Path: %s\n", path)
	fmt.Printf("  Size: %d bytes (%.1f KB)\n", size, float64(size)/1024)
	return nil
}

func checkStatus() error {
	d, err := openDB(nil)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer d.Close()

	stats, err := d.GetStats()
	if err != nil {
		return fmt.Errorf("get stats: %w", err)
	}

	fmt.Printf("  DB: %s\n", d.DBPath())
	for _, s := range []string{"active", "error", "untested", "deprecated"} {
		if c, ok := stats[s]; ok && c > 0 {
			fmt.Printf("  %s models: %d\n", s, c)
		}
	}
	if p, ok := stats["providers_active"]; ok {
		fmt.Printf("  Active providers: %d\n", p)
	}
	return nil
}
