package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newQueryCmd(dbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "query",
		Short: "Run SQL query against DB",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()

			query := strings.Join(args, " ")
			if strings.TrimSpace(query) == "" {
				return fmt.Errorf("query must not be empty")
			}
			rows, err := d.Query(query)
			if err != nil {
				return fmt.Errorf("query: %w", err)
			}
			defer rows.Close()

			cols, _ := rows.Columns()
			fmt.Println(strings.Join(cols, " | "))
			fmt.Println(strings.Repeat("-", len(cols)*12))

			var rowVals []any = make([]any, len(cols))
			rowPtrs := make([]any, len(cols))
			for i := range rowVals {
				rowPtrs[i] = &rowVals[i]
			}

			for rows.Next() {
				if err := rows.Scan(rowPtrs...); err != nil {
					return err
				}
				strVals := make([]string, len(cols))
				for i, v := range rowVals {
					if v == nil {
						strVals[i] = "NULL"
					} else {
						strVals[i] = fmt.Sprintf("%v", v)
					}
				}
				fmt.Println(strings.Join(strVals, " | "))
			}
			return nil
		},
	}
}
