package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/reeinharrrd/opencode-kit/internal/generator"
	"github.com/reeinharrrd/opencode-kit/internal/util"
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

			gen := generator.NewService(d, filepath.Dir(d.DBPath()))
			if err := gen.GenerateConfig(); err != nil {
				return fmt.Errorf("generate config: %w", err)
			}
			configPath := filepath.Join(filepath.Dir(d.DBPath()), "opencode.jsonc")
			data, err := os.ReadFile(configPath)
			if err != nil {
				return fmt.Errorf("read generated config: %w", err)
			}

			cleaned := util.StripJSONC(data)
			var cfg map[string]interface{}
			if err := json.Unmarshal(cleaned, &cfg); err != nil {
				return fmt.Errorf("parse generated config: %w", err)
			}

			// Validate critical sections are present (same as doctor check)
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
			if len(missing) > 0 {
				return fmt.Errorf("generated config missing sections: %s", strings.Join(missing, ", "))
			}

			fmt.Printf("Validated config at %s\n", configPath)
			return nil
		},
	}
}
