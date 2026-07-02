package cli

import (
	"fmt"

	"github.com/reeinharrrd/maestro/internal/discover"
	"github.com/spf13/cobra"
)

func newDiscoverCmd(dbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "discover",
		Short: "Discover models from provider catalogs",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()

			dis := discover.NewService(discover.NewServiceParams{DB: d})
			if err := dis.Discover(cmd.Context()); err != nil {
				return fmt.Errorf("discover: %w", err)
			}
			fmt.Println("Discovery complete.")
			return nil
		},
	}
}
