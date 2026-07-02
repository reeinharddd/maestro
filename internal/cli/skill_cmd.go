package cli

import (
	"fmt"
	"strings"

	"github.com/reeinharrrd/maestro/internal/config"
	"github.com/reeinharrrd/maestro/internal/skill"
	"github.com/spf13/cobra"
)

func newSkillCmd(dbPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Manage skills in the registry",
		Long: `Inspect, search, install, remove, and estimate context usage of skills.

  maestro skill list              List all skills
  maestro skill search <query>    Search skills by name, description, tags, or category
  maestro skill context-estimate  Show estimated context usage breakdown
  maestro skill install <name>    Install a skill symlink into opencode
  maestro skill remove <name>     Remove a skill symlink from opencode`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all skills in the registry",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()

			skills, err := d.ListSkills()
			if err != nil {
				return err
			}
			if len(skills) == 0 {
				fmt.Println("No skills found")
				return nil
			}
			fmt.Printf("%-30s %-20s %-25s %s\n", "ID", "Source", "Category", "Size")
			fmt.Println(strings.Repeat("-", 85))
			for _, s := range skills {
				size := formatSize(s.SizeBytes)
				fmt.Printf("%-30s %-20s %-25s %s\n", s.ID, s.Source, s.Category, size)
			}
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "search",
		Short: "Search skills by name, description, tags, or category",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()

			results, err := d.SearchSkills(args[0])
			if err != nil {
				return err
			}
			if len(results) == 0 {
				fmt.Printf("No skills matching %q\n", args[0])
				return nil
			}
			fmt.Printf("%-30s %-20s %-25s %s\n", "ID", "Source", "Category", "Size")
			fmt.Println(strings.Repeat("-", 85))
			for _, s := range results {
				size := formatSize(s.SizeBytes)
				fmt.Printf("%-30s %-20s %-25s %s\n", s.ID, s.Source, s.Category, size)
			}
			return nil
		},
	})

	{
		installCmd := &cobra.Command{
			Use:   "install",
			Short: "Install a skill by creating a symlink into opencode's skills directory",
			Long: `Install a skill: create a symlink from opencode's skills directory to the skill's source path.

The skill must exist in the database first (use 'maestro skill list' or 'maestro skill search').
Use --source-path to specify a custom source path, or it uses the stored source_path from DB.`,
			Args: cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				name := args[0]
				d, err := openDB(dbPath)
				if err != nil {
					return err
				}
				defer d.Close()

				sourcePath, _ := cmd.Flags().GetString("source-path")
				if sourcePath == "" {
					skills, err := d.ListSkills()
					if err != nil {
						return err
					}
					var found bool
					for _, s := range skills {
						if s.ID == name && s.SourcePath != "" {
							sourcePath = s.SourcePath
							found = true
							break
						}
					}
					if !found {
						return fmt.Errorf("skill %q not found in database and --source-path not provided", name)
					}
				}

				mgr := skill.NewManager(config.SkillsDir())
				if err := mgr.Install(name, sourcePath); err != nil {
					return err
				}
				fmt.Printf("Skill %q installed (%s)\n", name, sourcePath)
				return nil
			},
		}
		installCmd.Flags().String("source-path", "", "Source path to link from (overrides DB stored path)")
		cmd.AddCommand(installCmd)
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "remove",
		Short: "Remove a skill symlink from opencode's skills directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := skill.NewManager(config.SkillsDir())
			if err := mgr.Remove(args[0]); err != nil {
				return err
			}
			fmt.Printf("Skill %q removed\n", args[0])
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "context-estimate",
		Short: "Estimate how much context skills consume",
		Long: `Calculate and display an estimate of how many bytes all skills consume
in the system prompt, broken down by source and category.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()

			est := skill.NewEstimator(d)
			ce, err := est.Estimate()
			if err != nil {
				return err
			}

			fmt.Printf("Total skills: %d\n", ce.TotalSkills)
			fmt.Printf("Total estimated size: %s\n", formatSize(ce.TotalBytes))
			fmt.Println()

			if len(ce.BySource) > 0 {
				fmt.Println("By Source:")
				for src, size := range ce.BySource {
					fmt.Printf("  %-25s %s\n", src, formatSize(size))
				}
				fmt.Println()
			}

			if len(ce.ByCategory) > 0 {
				fmt.Println("By Category:")
				for cat, size := range ce.ByCategory {
					fmt.Printf("  %-25s %s\n", cat, formatSize(size))
				}
				fmt.Println()
			}

			if len(ce.Heaviest) > 0 {
				fmt.Println("Heaviest Skills:")
				fmt.Printf("%-30s %-20s %-25s %s\n", "ID", "Source", "Category", "Size")
				fmt.Println(strings.Repeat("-", 85))
				for _, h := range ce.Heaviest {
					fmt.Printf("%-30s %-20s %-25s %s\n", h.ID, h.Source, h.Category, formatSize(h.SizeBytes))
				}
			}
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "auto",
		Short: "[placeholder] Auto-install relevant skills for the current project",
		Long: `Auto-detect the current project's stack and install relevant skills.
This is a placeholder for future context-aware resolution.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Auto skill resolution is not yet implemented.")
			fmt.Println("Future behavior: detect stack → resolve relevant skills → install symlinks")
			return nil
		},
	})

	return cmd
}

func formatSize(bytes int64) string {
	switch {
	case bytes >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	case bytes >= 1024:
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
