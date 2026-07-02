package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/reeinharrrd/maestro/internal/routing"
	"github.com/reeinharrrd/maestro/pkg/models"
	"github.com/spf13/cobra"
)

func newRouteCmd(dbPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "route",
		Short: "Show or reassign model routing",
		RunE: func(cmd *cobra.Command, args []string) error {
			task, _ := cmd.Flags().GetString("task")
			reassign, _ := cmd.Flags().GetBool("reassign")
			shadow, _ := cmd.Flags().GetBool("shadow")
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			r := routing.New(d)
			if reassign {
				return r.ReassignAll(cmd.Context(), shadow)
			}
			if task != "" {
				budget, _ := d.GetBudget()
				if budget == nil {
					budget = &models.BudgetConfig{ID: "default"}
				}
				rule, err := r.SelectBestModel(task, *budget, shadow)
				if err != nil {
					return err
				}
				fmt.Printf("Best model for %s: %s\n", task, rule.CurrentModelID)
				if rule.FallbackIDs != "" {
					var fallbacks []string
					if err := json.Unmarshal([]byte(rule.FallbackIDs), &fallbacks); err == nil && len(fallbacks) > 0 {
						fmt.Printf("Fallback chain: %s\n", strings.Join(fallbacks, " -> "))
					}
				}
				return nil
			}
			rules, err := d.ListRoutingRules()
			if err != nil {
				return err
			}
			for _, rule := range rules {
				fmt.Printf("%s: %s\n", rule.TaskKey, rule.CurrentModelID)
			}
			return nil
		},
	}
	cmd.AddCommand(newRouteReportCmd(dbPath))
	cmd.Flags().String("task", "", "Task type (coding_complex, coding_fast, reasoning, vision, long_context, fastest)")
	cmd.Flags().Bool("reassign", false, "Reassign all routing rules")
	cmd.Flags().Bool("shadow", false, "Log routing decisions without writing to DB")
	return cmd
}

func newRouteReportCmd(dbPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Show recent routing decisions",
		RunE: func(cmd *cobra.Command, args []string) error {
			limit, _ := cmd.Flags().GetInt("limit")
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()

			events, err := d.ListRoutingEvents(limit)
			if err != nil {
				return err
			}
			if len(events) == 0 {
				fmt.Println("No routing events found")
				return nil
			}
			fmt.Printf("%-4s %-14s %-24s %-7s %s\n", "ID", "Task", "Model", "Shadow", "Reason")
			fmt.Println(strings.Repeat("-", 80))
			for _, e := range events {
				shadow := "no"
				if e.Shadow {
					shadow = "yes"
				}
				fmt.Printf("%-4d %-14s %-24s %-7s %s | %s\n", e.ID, e.TaskKey, e.SelectedModel, shadow, e.Reason, routing.FormatCandidateSummary(e.Candidates))
			}
			return nil
		},
	}
	cmd.Flags().Int("limit", 20, "Number of routing events to show")
	return cmd
}
