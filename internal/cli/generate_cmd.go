package cli

import (
	"github.com/reeinharrrd/maestro/internal/generator"
	"github.com/spf13/cobra"
)

func newGenerateCmd(dbPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate OpenCode configuration files",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "config",
		Short: "Generate opencode config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			svc := generator.NewService(d, "")
			return svc.GenerateConfig()
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "agents",
		Short: "Generate agents/*.md files",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			svc := generator.NewService(d, "")
			return svc.GenerateAgents()
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "commands",
		Short: "Generate commands/*.md files",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			svc := generator.NewService(d, "")
			return svc.GenerateCommands()
		},
	})
	return cmd
}
