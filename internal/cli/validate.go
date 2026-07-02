package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/reeinharrrd/maestro/internal/generator"
	"github.com/reeinharrrd/maestro/internal/util"
	"github.com/spf13/cobra"
)

func newValidateCmd(dbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate generated config",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()

			outputDir := filepath.Dir(d.DBPath())
			gen := generator.NewService(d, outputDir)
			if err := gen.GenerateConfig(); err != nil {
				return fmt.Errorf("generate config: %w", err)
			}

			configName := "opencode.jsonc"
			if _, err := os.Stat(filepath.Join(outputDir, "opencode.json")); err == nil {
				configName = "opencode.json"
			}
			configPath := filepath.Join(outputDir, configName)
			data, err := os.ReadFile(configPath)
			if err != nil {
				return fmt.Errorf("read generated config: %w", err)
			}

			cleaned := util.StripJSONC(data)
			var cfg map[string]interface{}
			if err := json.Unmarshal(cleaned, &cfg); err != nil {
				return fmt.Errorf("parse generated config: %w", err)
			}

			required := []string{"provider", "agent", "mcp"}
			missing := []string{}
			present := 0
			for _, s := range required {
				if _, ok := cfg[s]; ok {
					present++
				} else {
					missing = append(missing, s)
				}
			}
			if len(missing) > 0 {
				return fmt.Errorf("generated config missing required sections: %s", strings.Join(missing, ", "))
			}

			fmt.Printf("Validated config at %s\n", configPath)
			return nil
		},
	}
}
