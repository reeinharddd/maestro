package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/reeinharrrd/opencode-kit/internal/db"
	"github.com/reeinharrrd/opencode-kit/pkg/models"
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
