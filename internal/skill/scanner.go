package skill

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/reeinharrrd/maestro/internal/db"
	"github.com/reeinharrrd/maestro/pkg/models"
)

// Scanner reads SKILL.md files from directories and upserts their metadata into the DB.
type Scanner struct {
	db db.DBInterface
}

// NewScanner creates a Scanner backed by the given database.
func NewScanner(database db.DBInterface) *Scanner {
	return &Scanner{db: database}
}

// ScanDir walks root looking for .md files, parses frontmatter, and upserts skills.
// Returns the count of skills found.
func (s *Scanner) ScanDir(root string) (int, error) {
	var count int
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible files
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".md") && !strings.HasSuffix(info.Name(), ".MD") {
			return nil
		}
		// Skip files inside hidden directories
		rel, _ := filepath.Rel(root, path)
		if strings.Contains(rel, string(filepath.Separator)+".") || strings.HasPrefix(rel, ".") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		id := strings.TrimSuffix(info.Name(), ".md")
		id = strings.TrimSuffix(id, ".MD")

		description, category, tags, triggers := parseFrontmatter(data)
		hash := sha256.Sum256(data)
		hashStr := hex.EncodeToString(hash[:])
		source := detectSource(root, path)

		skill := &models.Skill{
			ID:          id,
			Source:      source,
			SourcePath:  path,
			Type:        "skill",
			Status:      "active",
			Hash:        hashStr,
			Description: description,
			Category:    category,
			Tags:        tags,
			Triggers:    triggers,
			SizeBytes:   int64(len(data)),
			Filename:    info.Name(),
		}

		if err := s.db.UpsertSkill(skill); err != nil {
			return fmt.Errorf("upsert skill %s: %w", id, err)
		}
		count++
		return nil
	})
	return count, err
}

// parseFrontmatter extracts metadata from YAML frontmatter blocks (--- ... ---).
func parseFrontmatter(data []byte) (description, category, tags, triggers string) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	inFrontmatter := false
	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			}
			break
		}
		if !inFrontmatter {
			continue
		}
		if idx := strings.Index(line, ":"); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			val := strings.TrimSpace(line[idx+1:])
			switch strings.ToLower(key) {
			case "description":
				description = val
			case "category":
				category = val
			case "tags":
				tags = val
			case "triggers":
				triggers = val
			}
		}
	}
	return
}

// detectSource derives a source name from the first path component under root.
func detectSource(root, path string) string {
	rel, _ := filepath.Rel(root, path)
	parts := strings.SplitN(rel, string(filepath.Separator), 2)
	if len(parts) > 1 {
		return parts[0]
	}
	return "unknown"
}
