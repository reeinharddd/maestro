package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

func newPreferencesCmd(dbPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prefs",
		Short: "Manage key-value preferences",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all preferences",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			prefs, err := d.ListPreferences()
			if err != nil {
				return err
			}
			if len(prefs) == 0 {
				fmt.Println("No preferences set")
				return nil
			}
			keys := make([]string, 0, len(prefs))
			for k := range prefs {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Printf("%s = %s\n", k, prefs[k])
			}
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "get",
		Short: "Get a preference value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			val, err := d.GetPreference(args[0])
			if err != nil {
				return fmt.Errorf("preference %q not found", args[0])
			}
			fmt.Println(val)
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "set",
		Short: "Set a preference",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			if err := d.SetPreference(args[0], args[1]); err != nil {
				return err
			}
			fmt.Printf("Preference %q set.\n", args[0])
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "delete",
		Short: "Delete a preference",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			if _, err := d.GetPreference(args[0]); err != nil {
				return fmt.Errorf("preference %q not found", args[0])
			}
			if err := d.DeletePreference(args[0]); err != nil {
				return err
			}
			fmt.Printf("Preference %q deleted.\n", args[0])
			return nil
		},
	})
	return cmd
}
