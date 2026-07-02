package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/reeinharrrd/maestro/internal/db"
	"github.com/reeinharrrd/maestro/pkg/models"
	"github.com/spf13/cobra"
)

func newSkillsCmd(dbPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Manage skills",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all skills",
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
			fmt.Printf("%-30s %-15s %-10s %s\n", "ID", "Source", "Type", "Status")
			fmt.Println(strings.Repeat("-", 70))
			for _, s := range skills {
				fmt.Printf("%-30s %-15s %-10s %s\n", s.ID, s.Source, s.Type, s.Status)
			}
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "report",
		Short: "Show skill import summary",
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
			fmt.Printf("Skills: %d\n", len(skills))
			for _, s := range skills {
				fmt.Printf("%s | %s | %s | %s\n", s.ID, s.Source, s.Type, s.Status)
			}
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "sync",
		Short: "Sync skills from a directory into the DB",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, _ := cmd.Flags().GetString("dir")
			if dir == "" {
				dir = filepath.Join(filepath.Dir(db.DefaultPath()), "skills")
			}
			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()
			entries, err := os.ReadDir(dir)
			if err != nil {
				return fmt.Errorf("read skills dir %s: %w", dir, err)
			}
			added, updated, skipped := 0, 0, 0
			now := time.Now().UTC().Format(time.RFC3339)
			existing, _ := d.ListSkills()
			index := make(map[string]models.Skill, len(existing))
			for _, s := range existing {
				index[s.ID] = s
			}
			for _, e := range entries {
				if !e.IsDir() {
					skipped++
					continue
				}
				name := e.Name()
				id := name
				path := filepath.Join(dir, name)
				hash := hashDir(path)
				prev, ok := index[id]
				typ := "directory"
				if _, err := os.Stat(filepath.Join(path, "SKILL.md")); err == nil {
					typ = "skill"
				}
				if ok && prev.Hash == hash {
					skipped++
					continue
				}
				skill := &models.Skill{
					ID:         id,
					Source:     "filesystem",
					SourcePath: path,
					Type:       typ,
					Status:     "active",
					Hash:       hash,
					LastSynced: time.Now().Unix(),
				}
				if err := d.UpsertSkill(skill); err != nil {
					return fmt.Errorf("upsert skill %s: %w", id, err)
				}
				if ok {
					updated++
				} else {
					added++
				}
				_ = now
			}
			fmt.Printf("Skills synced: %d added, %d updated, %d skipped\n", added, updated, skipped)
			return nil
		},
	})
	cmd.Flags().String("dir", "", "Skills directory to sync (default: $OPENCODE_CONFIG_DIR/skills)")
	cmd.PersistentFlags().String("id", "", "Skill ID")
	cmd.PersistentFlags().String("source", "", "Skill source (manual, registry, reference)")
	cmd.PersistentFlags().String("source-path", "", "Source path")
	cmd.PersistentFlags().String("target-path", "", "Target path")
	cmd.PersistentFlags().String("type", "", "Skill type (skill, agent, command, mcp)")
	cmd.PersistentFlags().String("status", "active", "Skill status")
	cmd.AddCommand(newSkillAddCmd(dbPath))
	cmd.AddCommand(newSkillUpdateCmd(dbPath))
	cmd.AddCommand(newSkillRemoveCmd(dbPath))
	return cmd
}

func hashDir(path string) string {
	h := sha256.New()
	filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(path, p)
		h.Write([]byte(rel))
		if data, err := os.ReadFile(p); err == nil {
			h.Write(data)
		}
		return nil
	})
	return hex.EncodeToString(h.Sum(nil))
}

func newSkillAddCmd(dbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "add",
		Short: "Add a skill to the database",
		Long: `Add a skill and auto-sync to opencode config.

Example:
  maestro skills add --id my-skill --source manual --type skill --source-path /path/to/skill`,
		RunE: func(cmd *cobra.Command, args []string) error {
			id, _ := cmd.Flags().GetString("id")
			source, _ := cmd.Flags().GetString("source")
			typ, _ := cmd.Flags().GetString("type")
			if id == "" || source == "" || typ == "" {
				return fmt.Errorf("required: --id, --source, --type")
			}

			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()

			s := &models.Skill{
				ID:     id,
				Source: source,
				Type:   typ,
				Status: "active",
			}
			if v, _ := cmd.Flags().GetString("source-path"); v != "" {
				s.SourcePath = v
			}
			if v, _ := cmd.Flags().GetString("target-path"); v != "" {
				s.TargetPath = v
			}
			if v, _ := cmd.Flags().GetString("status"); v != "active" {
				s.Status = v
			}
			if err := d.UpsertSkill(s); err != nil {
				return fmt.Errorf("db insert: %w", err)
			}
			fmt.Printf("Skill %s added to database.\n", id)
			return syncConfig(d)
		},
	}
}

func newSkillUpdateCmd(dbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update an existing skill",
		Long: `Update skill fields. Only provided flags are changed.
Auto-syncs to opencode config.

Example:
  maestro skills update --id my-skill --status inactive`,
		RunE: func(cmd *cobra.Command, args []string) error {
			id, _ := cmd.Flags().GetString("id")
			if id == "" {
				return fmt.Errorf("required: --id")
			}

			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()

			skills, err := d.ListSkills()
			if err != nil {
				return err
			}
			var existing *models.Skill
			for i, s := range skills {
				if s.ID == id {
					existing = &skills[i]
					break
				}
			}
			if existing == nil {
				return fmt.Errorf("skill %q not found", id)
			}

			changed := false
			if v, _ := cmd.Flags().GetString("source"); cmd.Flags().Changed("source") {
				existing.Source = v
				changed = true
			}
			if v, _ := cmd.Flags().GetString("source-path"); cmd.Flags().Changed("source-path") {
				existing.SourcePath = v
				changed = true
			}
			if v, _ := cmd.Flags().GetString("target-path"); cmd.Flags().Changed("target-path") {
				existing.TargetPath = v
				changed = true
			}
			if v, _ := cmd.Flags().GetString("type"); cmd.Flags().Changed("type") {
				existing.Type = v
				changed = true
			}
			if v, _ := cmd.Flags().GetString("status"); cmd.Flags().Changed("status") {
				existing.Status = v
				changed = true
			}
			if !changed {
				return fmt.Errorf("no fields to update (specify at least one flag)")
			}
			if err := d.UpsertSkill(existing); err != nil {
				return fmt.Errorf("db update: %w", err)
			}
			fmt.Printf("Skill %s updated in database.\n", id)
			return syncConfig(d)
		},
	}
}

func newSkillRemoveCmd(dbPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "remove",
		Short: "Remove a skill",
		Long: `Remove a skill from the database.
Auto-syncs to opencode config.

Example:
  maestro skills remove --id my-skill`,
		RunE: func(cmd *cobra.Command, args []string) error {
			id, _ := cmd.Flags().GetString("id")
			if id == "" {
				return fmt.Errorf("required: --id")
			}

			d, err := openDB(dbPath)
			if err != nil {
				return err
			}
			defer d.Close()

			skills, err := d.ListSkills()
			if err != nil {
				return err
			}
			found := false
			for _, s := range skills {
				if s.ID == id {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("skill %q not found", id)
			}
			if err := d.DeleteSkill(id); err != nil {
				return fmt.Errorf("db delete: %w", err)
			}
			fmt.Printf("Skill %s removed from database.\n", id)
			return syncConfig(d)
		},
	}
}
