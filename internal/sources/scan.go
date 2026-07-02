package sources

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)
// ScannedItem represents a single discovered item from scanning a source.
type ScannedItem struct {
	ID         string
	Type       string // skill, agent, command, mcp, plugin, workflow, harness, config, prompt, rule
	SourcePath string
	Scanner    string // name of the scanner that found it
}

// Scanner defines a strategy for discovering opencode-mappable items
// inside a cloned source repository.
type Scanner interface {
	Name() string
	Scan(rootPath string) ([]ScannedItem, error)
}

// ─── DirScanner ───────────────────────────────────────────────────────────────

type dirMapping struct {
	Dir  string
	Type string
}

var discoverDirs = []dirMapping{
	{"skills", "skill"},
	{"agents", "agent"},
	{"commands", "command"},
	{"mcps", "mcp"},
	{"mcp", "mcp"},
	{"plugins", "plugin"},
	{"workflows", "workflow"},
}

// DirScanner scans well-known subdirectories (skills/, agents/, commands/, …)
// for .md files and maps them to their entity type.
type DirScanner struct{}

func (d *DirScanner) Name() string { return "dir" }

func (d *DirScanner) Scan(rootPath string) ([]ScannedItem, error) {
	var items []ScannedItem
	for _, dd := range discoverDirs {
		dir := filepath.Join(rootPath, dd.Dir)
		// Walk recursively to handle both flat .md files and subdirectory-nested
		// (e.g., skills/brainstorming/SKILL.md)
		filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return nil // skip permission errors etc.
			}
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				return nil
			}
			// Build unique ID from relative path
			rel, _ := filepath.Rel(dir, path)
			relName := strings.TrimSuffix(rel, ".md")
			relName = strings.ReplaceAll(relName, string(os.PathSeparator), "-")
			items = append(items, ScannedItem{
				ID:         dd.Dir + "-" + relName,
				Type:       dd.Type,
				SourcePath: path,
				Scanner:    "dir",
			})
			return nil
		})
	}
	return items, nil
}

// ─── RootScanner ──────────────────────────────────────────────────────────────

// RootScanner scans root-level configuration and harness files.
// It detects well-known filenames and maps them by convention.
type RootScanner struct{}

func (r *RootScanner) Name() string { return "root" }

func (r *RootScanner) Scan(rootPath string) ([]ScannedItem, error) {
	var items []ScannedItem
	entries, err := os.ReadDir(rootPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".md")
		fullPath := filepath.Join(rootPath, entry.Name())

		// Detect by well-known filename conventions
		var itemType string
		switch {
		case strings.EqualFold(name, "SKILL") || strings.EqualFold(name, "SKILLS"):
			itemType = "skill"
		case strings.EqualFold(name, "AGENT") || strings.EqualFold(name, "AGENTS"):
			itemType = "agent"
		case strings.EqualFold(name, "COMMAND") || strings.EqualFold(name, "COMMANDS"):
			itemType = "command"
		case strings.EqualFold(name, "MCP") || strings.EqualFold(name, "MCPS"):
			itemType = "mcp"
		case strings.EqualFold(name, "CLAUDE"):
			itemType = "config"
		case strings.EqualFold(name, "GEMINI"):
			itemType = "config"
		case strings.EqualFold(name, "README"):
			itemType = "doc"
		case strings.EqualFold(name, "CHANGELOG"):
			itemType = "doc"
		case strings.EqualFold(name, "CONSTITUTION"):
			itemType = "rule"
		case strings.EqualFold(name, "LICENSE"):
			itemType = "doc"
		case strings.EqualFold(name, "CODE_OF_CONDUCT") || strings.EqualFold(name, "CONTRIBUTING") || strings.EqualFold(name, "RELEASE-NOTES") || strings.EqualFold(name, "RELEASE_NOTES") || strings.EqualFold(name, "REPORT"):
			itemType = "doc"
		case strings.EqualFold(name, "TODO") || strings.EqualFold(name, "TODOS") || strings.EqualFold(name, "ROADMAP"):
			itemType = "doc"
		default:
			// Flat skill repos: remaining .md files in root are likely skills
			itemType = "skill"
		}

		items = append(items, ScannedItem{
			ID:         "root-" + name,
			Type:       itemType,
			SourcePath: fullPath,
			Scanner:    "root",
		})
	}

	return items, nil
}

// ─── OpenCodeScanner ──────────────────────────────────────────────────────────

// OpenCodeScanner scans the .opencode/ directory for nested skills, agents,
// commands, and other harness-specific items.
type OpenCodeScanner struct{}

func (o *OpenCodeScanner) Name() string { return "opencode" }

var opencodeDirs = []dirMapping{
	{"skills", "skill"},
	{"agents", "agent"},
	{"commands", "command"},
	{"mcps", "mcp"},
	{"mcp", "mcp"},
	{"plugins", "plugin"},
	{"workflows", "workflow"},
	{"rules", "rule"},
	{"prompts", "prompt"},
}

func (o *OpenCodeScanner) Scan(rootPath string) ([]ScannedItem, error) {
	opencodeDir := filepath.Join(rootPath, ".opencode")
	if info, err := os.Stat(opencodeDir); err != nil || !info.IsDir() {
		return nil, nil
	}

	var items []ScannedItem

	// Mark the .opencode directory itself as a harness
	items = append(items, ScannedItem{
		ID:         "opencode",
		Type:       "harness",
		SourcePath: opencodeDir,
		Scanner:    "opencode",
	})

	// Scan each well-known subdirectory (recursively to handle nested SKILL.md)
	for _, dd := range opencodeDirs {
		subdir := filepath.Join(opencodeDir, dd.Dir)
		filepath.WalkDir(subdir, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				return nil
			}
			rel, _ := filepath.Rel(subdir, path)
			relName := strings.TrimSuffix(rel, ".md")
			relName = strings.ReplaceAll(relName, string(os.PathSeparator), "-")
			items = append(items, ScannedItem{
				ID:         "opencode-" + dd.Dir + "-" + relName,
				Type:       dd.Type,
				SourcePath: path,
				Scanner:    "opencode",
			})
			return nil
		})
	}

	return items, nil
}

// ─── ClaudeScanner ────────────────────────────────────────────────────────────

// ClaudeScanner scans .claude/ and .claude-plugin/ directories which may contain
// skills, agents, hooks, or plugin definitions.
type ClaudeScanner struct{}

func (c *ClaudeScanner) Name() string { return "claude" }

func (c *ClaudeScanner) Scan(rootPath string) ([]ScannedItem, error) {
	var items []ScannedItem

	claudeDirs := []string{".claude", ".claude-plugin"}
	for _, dirName := range claudeDirs {
		dir := filepath.Join(rootPath, dirName)
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}

		items = append(items, ScannedItem{
			ID:         strings.TrimPrefix(dirName, "."),
			Type:       "plugin",
			SourcePath: dir,
			Scanner:    "claude",
		})

		// Scan for skills subdirectory
		skillsDir := filepath.Join(dir, "skills")
		if entries, err := os.ReadDir(skillsDir); err == nil {
			for _, entry := range entries {
				if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
					continue
				}
				name := strings.TrimSuffix(entry.Name(), ".md")
				items = append(items, ScannedItem{
					ID:         dirName + "-skill-" + name,
					Type:       "skill",
					SourcePath: filepath.Join(skillsDir, entry.Name()),
					Scanner:    "claude",
				})
			}
		}
	}

	return items, nil
}

// ─── RootHarnessScanner ───────────────────────────────────────────────────────

// RootHarnessScanner detects that the root repository itself is a harness
// by looking for AGENTS.md, CLAUDE.md, package.json with opencode/agent metadata.
type RootHarnessScanner struct{}

func (h *RootHarnessScanner) Name() string { return "root-harness" }

func (h *RootHarnessScanner) Scan(rootPath string) ([]ScannedItem, error) {
	var items []ScannedItem

	isHarness := false

	// Check for AGENTS.md or CLAUDE.md at root
	for _, name := range []string{"AGENTS.md", "CLAUDE.md", "GEMINI.md"} {
		if _, err := os.Stat(filepath.Join(rootPath, name)); err == nil {
			isHarness = true
			itemType := "config"
			if name == "AGENTS.md" {
				itemType = "agent"
			}
			items = append(items, ScannedItem{
				ID:         "root-" + strings.TrimSuffix(name, ".md"),
				Type:       itemType,
				SourcePath: filepath.Join(rootPath, name),
				Scanner:    "root-harness",
			})
		}
	}

	// Check for package.json
	packagePath := filepath.Join(rootPath, "package.json")
	if data, err := os.ReadFile(packagePath); err == nil {
		if strings.Contains(string(data), `"opencode`) ||
			strings.Contains(string(data), `"agent`) ||
			strings.Contains(string(data), `"skills`) {
			isHarness = true
			items = append(items, ScannedItem{
				ID:         "package-json",
				Type:       "config",
				SourcePath: packagePath,
				Scanner:    "root-harness",
			})
		}
	}

	// If AGENTS.md exists, this repo is a harness itself
	if isHarness {
		items = append([]ScannedItem{{
			ID:         "harness",
			Type:       "harness",
			SourcePath: rootPath,
			Scanner:    "root-harness",
		}}, items...)
	}

	return items, nil
}

// --- RootSkillDirScanner ---

type RootSkillDirScanner struct{}

func (d *RootSkillDirScanner) Name() string { return "root-skills" }

func (d *RootSkillDirScanner) Scan(rootPath string) ([]ScannedItem, error) {
  var items []ScannedItem
  entries, err := os.ReadDir(rootPath)
  if err != nil {
    return nil, err
  }
  skipDirs := map[string]bool{
    "skills": true, "agents": true, "commands": true,
    "mcps": true, "mcp": true, "plugins": true, "workflows": true,
    "node_modules": true, "vendor": true, ".git": true,
  }
  for _, entry := range entries {
    if !entry.IsDir() { continue }
    if strings.HasPrefix(entry.Name(), ".") { continue }
    if skipDirs[entry.Name()] { continue }
    for _, sf := range []string{"SKILL.md", "SKILLS.md"} {
      if _, err2 := os.Stat(filepath.Join(rootPath, entry.Name(), sf)); err2 == nil {
        items = append(items, ScannedItem{
          ID: "skill-" + strings.ToLower(entry.Name()),
          Type: "skill",
          SourcePath: filepath.Join(rootPath, entry.Name()),
          Scanner: "root-skills",
        })
        break
      }
    }
  }
  return items, nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// AllScanners returns the default set of scanners used during discovery.
func AllScanners() []Scanner {
	return []Scanner{
&DirScanner{},
&RootSkillDirScanner{},
&RootHarnessScanner{},
&RootScanner{},
&OpenCodeScanner{},
&ClaudeScanner{},
	}
}

// ScanAll runs all scanners against rootPath and returns deduplicated items.
// Items from earlier scanners take priority on ID collision.
func ScanAll(rootPath string) ([]ScannedItem, error) {
	var all []ScannedItem
	seen := make(map[string]bool)

	for _, scanner := range AllScanners() {
		items, err := scanner.Scan(rootPath)
		if err != nil {
			continue
		}
		for _, item := range items {
			if seen[item.ID] {
				continue
			}
			seen[item.ID] = true
			all = append(all, item)
		}
	}

	return all, nil
}
