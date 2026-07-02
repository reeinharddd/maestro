package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/reeinharrrd/maestro/internal/audit"
	"github.com/spf13/cobra"
)

func newAuditLiveCmd(dbPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "live",
		Short: "Live audit: real provider catalogs + smoke tests",
		Long: `Phase 1+2: fetch the real /v1/models catalog from every active provider
using the API key in your env, and diff against the local DB.
Phase 3 (--smoke): send a 1-token chat completion to each active model
to verify it really works (rate-limit aware, max ~$0.01 of tokens).`,
		RunE: func(cmd *cobra.Command, args []string) error {
		smoke, _ := cmd.Flags().GetBool("smoke")
		fix, _ := cmd.Flags().GetBool("fix")
		d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()

			l := audit.NewLive(d, 4)

			fmt.Println("=== Phase 1+2: Real catalog vs DB ===")
			fmt.Println()
			diffs, err := l.DiffAll(cmd.Context())
			if err != nil {
				return err
			}

			totals := struct{ real, db, phantom, missing int }{}
			for _, d := range diffs {
				totals.real += d.RealCount
				totals.db += d.DBCount
				totals.phantom += len(d.Phantom)
				totals.missing += len(d.Missing)
			}

			fmt.Printf("%-14s %5s %5s %8s %8s  %s\n", "PROVIDER", "REAL", "DB", "PHANTOM", "MISSING", "STATUS")
			fmt.Println(strings.Repeat("-", 78))
			for _, d := range diffs {
				if d.FetchError != "" {
					fmt.Printf("%-14s %5s %5d %8s %8s  SKIP (%s)\n", d.ProviderID, "-", d.DBCount, "-", "-", d.FetchError)
					continue
				}
				status := "OK"
				if len(d.Phantom) > 0 || len(d.Missing) > 0 {
					status = fmt.Sprintf("DRIFT -%d/+%d", len(d.Phantom), len(d.Missing))
				}
				fmt.Printf("%-14s %5d %5d %8d %8d  %s\n", d.ProviderID, d.RealCount, d.DBCount, len(d.Phantom), len(d.Missing), status)
			}
			fmt.Println(strings.Repeat("-", 78))
			fmt.Printf("%-14s %5d %5d %8d %8d\n", "TOTAL", totals.real, totals.db, totals.phantom, totals.missing)
			fmt.Println()

			anyDrift := false
			for _, d := range diffs {
				if len(d.Phantom) == 0 && len(d.Missing) == 0 {
					continue
				}
				anyDrift = true
				fmt.Println(strings.Repeat("=", 78))
				fmt.Printf("[%s]\n", d.ProviderID)
				if len(d.Phantom) > 0 {
					sort.Strings(d.Phantom)
					fmt.Printf("  PHANTOM (%d) in DB but not in real catalog:\n", len(d.Phantom))
					for i, p := range d.Phantom {
						if i >= 10 {
							fmt.Printf("    ... +%d more\n", len(d.Phantom)-10)
							break
						}
						fmt.Printf("    - %s/%s\n", d.ProviderID, p)
					}
				}
				if len(d.Missing) > 0 {
					sort.Strings(d.Missing)
					fmt.Printf("  MISSING (%d) in real catalog but not in DB:\n", len(d.Missing))
					for i, p := range d.Missing {
						if i >= 10 {
							fmt.Printf("    ... +%d more\n", len(d.Missing)-10)
							break
						}
						fmt.Printf("    + %s\n", p)
					}
				}
				fmt.Println()
			}
			if !anyDrift {
				fmt.Println("No drift detected.")
			}

			if fix {
				fmt.Println("=== Apply fixes ===")
				fixes, err := l.FixAll(cmd.Context())
				if err != nil {
					return err
				}
				grandPhantom, grandMissing, grandSkip := 0, 0, 0
				for _, f := range fixes {
					if f.FetchError != "" {
						fmt.Printf("  %-14s  SKIP (%s)\n", f.ProviderID+":", f.FetchError)
						continue
					}
					fmt.Printf("  %-14s  phantoms fixed: %d, missing added: %d, skipped (non-chat): %d\n",
						f.ProviderID+":", f.PhantomFixed, f.MissingAdded, f.Skipped)
					grandPhantom += f.PhantomFixed
					grandMissing += f.MissingAdded
					grandSkip += f.Skipped
				}
				fmt.Printf("  %-14s  total fixed: %d, added: %d, skipped: %d\n",
					"TOTAL:", grandPhantom, grandMissing, grandSkip)
				fmt.Println()
				fmt.Println("Tip: run `maestro audit live --smoke` to verify the new untested models.")
			}

			if !smoke {
				fmt.Println()
				fmt.Println("Tip: pass --smoke to send 1-token chat completions and verify each active model.")
				return nil
			}

			fmt.Println("=== Phase 3: Smoke test (1-token completion per active model) ===")
			fmt.Println()
			smokeRes, err := l.SmokeAll(cmd.Context(), audit.SmokeOpts{})
			if err != nil {
				return err
			}
			byStatus := map[string]int{}
			failed := []audit.LiveModelResult{}
			for _, r := range smokeRes {
				byStatus[r.Status]++
				if r.Status != "ok" && r.Status != "rate_limited" {
					failed = append(failed, r)
				}
			}
			order := []string{"ok", "not_found", "rate_limited", "unauthorized", "err"}
			for _, s := range order {
				if c, ok := byStatus[s]; ok {
					fmt.Printf("  %-15s %d\n", s+":", c)
				}
			}
			for s, c := range byStatus {
				skip := false
				for _, o := range order {
					if o == s {
						skip = true
						break
					}
				}
				if !skip {
					fmt.Printf("  %-15s %d\n", s+":", c)
				}
			}
			fmt.Println()
			if len(failed) > 0 {
				fmt.Println("--- FAILED MODELS (non-ok, non-rate-limited) ---")
				sort.Slice(failed, func(i, j int) bool {
					if failed[i].Status != failed[j].Status {
						return failed[i].Status < failed[j].Status
					}
					return failed[i].ModelID < failed[j].ModelID
				})
				for _, f := range failed {
					fmt.Printf("  %-12s  %s/%s  %s\n", f.Status, f.Provider, f.ModelID, truncate(f.ErrorMsg, 80))
				}
			} else {
				fmt.Println("No broken models detected by smoke test.")
			}

			return nil
		},
	}
	cmd.Flags().Bool("smoke", false, "Run 1-token chat completion per active model")
	cmd.Flags().Bool("fix", false, "Apply fixes: mark phantoms as error, insert missing free chat models as untested")
	return cmd
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
