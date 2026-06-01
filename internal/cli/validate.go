package cli

import (
	"fmt"
	"path/filepath"

	"github.com/reeinharrrd/opencode-kit/internal/generator"
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
			fmt.Printf("Validated config at %s\n", configPath)
			return nil
		},
	}
}
