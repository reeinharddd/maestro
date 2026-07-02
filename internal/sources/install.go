package sources

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/reeinharrrd/maestro/internal/config"
	"github.com/reeinharrrd/maestro/pkg/models"
)

// entityDirs maps entity types to their target subdirectory under
// maestro's data directory (~/.local/share/maestro/).
// ALL items stay external to opencode — maestro manages its own tree.
var entityDirs = map[string]string{
	"skill":    "skills",
	"agent":    "agents",
	"command":  "commands",
	"mcp":      "mcps",
	"plugin":   "plugins",
	"workflow": "workflows",
	"prompt":   "prompts",
	"rule":     "rules",
}

// Installer creates and removes symlinks in opencode directories for discovered items.
type Installer struct {
	db dbForInstaller
}

// dbForInstaller is the minimal DB interface the installer needs.
type dbForInstaller interface {
	GetSourceItem(id string) (*models.SourceItem, error)
	UpdateSourceItemStatus(id, status string) error
	UpdateSourceItemTarget(id, targetPath string) error
	ListSourceItemsBySource(sourceID string) ([]models.SourceItem, error)
}

// NewInstaller creates an Installer backed by the given DB.
func NewInstaller(database dbForInstaller) *Installer {
	return &Installer{db: database}
}

// installSymlink creates a symlink at targetPath pointing to sourcePath.
// It replaces existing symlinks but refuses to replace regular files.
func installSymlink(sourcePath, targetPath string) error {
	if existing, err := os.Lstat(targetPath); err == nil {
		if existing.Mode()&os.ModeSymlink != 0 {
			if err := os.Remove(targetPath); err != nil {
				return fmt.Errorf("remove existing symlink %s: %w", targetPath, err)
			}
		} else {
			return fmt.Errorf("target %s already exists and is not a symlink", targetPath)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat %s: %w", targetPath, err)
	}

	if err := os.Symlink(sourcePath, targetPath); err != nil {
		return fmt.Errorf("symlink %s -> %s: %w", targetPath, sourcePath, err)
	}
	return nil
}

// InstallItem creates a symlink for a single source item.
// All types use a unified path under ~/.local/share/maestro/<type>/,
// namespaced with SourceID. Items stay external to opencode.
func (inst *Installer) InstallItem(item *models.SourceItem) error {
	if item.SourcePath == "" {
		return fmt.Errorf("source item %q has no source path", item.ID)
	}

	// Capture original basename + extension for readability.
	// The item.ID is used as the unique symlink name (already namespaced by sourceID),
	// but we append the original extension so files look like proper filenames.
	origBase := filepath.Base(item.SourcePath)
	ext := filepath.Ext(origBase)

	// Sanitize hardcoded paths before symlinking.
	safePath, err := SanitizeFile(item.SourcePath, item.SourceID)
	if err != nil {
		return fmt.Errorf("sanitize %s: %w", item.ID, err)
	}

	subdir, ok := entityDirs[item.Type]
	if !ok {
		return fmt.Errorf("unsupported item type %q for install", item.Type)
	}
	targetDir := filepath.Join(config.DataDir(), subdir)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("create target directory %s: %w", targetDir, err)
	}

	// Use item.ID as the unique symlink name: it already contains the sourceID +
	// structured relative path, preventing collisions when multiple items share
	// the same basename (e.g., SKILL.md from different subdirectories).
	symlinkName := item.ID + ext
	targetPath := filepath.Join(targetDir, symlinkName)

	if err := installSymlink(safePath, targetPath); err != nil {
		return fmt.Errorf("install symlink %s: %w", item.ID, err)
	}

	_ = inst.db.UpdateSourceItemTarget(item.ID, targetPath)
	_ = inst.db.UpdateSourceItemStatus(item.ID, "installed")
	return nil
}

// UninstallItem removes the symlink for a single source item.
// Uses targetPath from DB for all types (including skills).
func (inst *Installer) UninstallItem(item *models.SourceItem) error {
	if item.TargetPath == "" {
		return fmt.Errorf("item %q has no target path (not installed)", item.ID)
	}

	info, err := os.Lstat(item.TargetPath)
	if err != nil {
		if os.IsNotExist(err) {
			_ = inst.db.UpdateSourceItemStatus(item.ID, "active")
			return nil
		}
		return fmt.Errorf("stat %s: %w", item.TargetPath, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("%s is not a symlink (refusing to remove)", item.TargetPath)
	}
	if err := os.Remove(item.TargetPath); err != nil {
		return fmt.Errorf("remove %s: %w", item.TargetPath, err)
	}

	_ = inst.db.UpdateSourceItemStatus(item.ID, "active")
	return nil
}

// InstallAll installs all items from a source that have a matching entity type.
func (inst *Installer) InstallAll(sourceID string) error {
	items, err := inst.db.ListSourceItemsBySource(sourceID)
	if err != nil {
		return fmt.Errorf("list items for source %q: %w", sourceID, err)
	}
	var count int
	for _, item := range items {
		if _, ok := entityDirs[item.Type]; !ok {
			continue
		}
		if item.Status == "installed" {
			continue
		}
		if err := inst.InstallItem(&item); err != nil {
			fmt.Printf("  Warning: install %s (%s): %v\n", item.ID, item.Type, err)
			continue
		}
		count++
	}
	fmt.Printf("  Installed %d items from source %s\n", count, sourceID)
	return nil
}

// UninstallAll removes all installed symlinks from a source.
func (inst *Installer) UninstallAll(sourceID string) error {
	items, err := inst.db.ListSourceItemsBySource(sourceID)
	if err != nil {
		return fmt.Errorf("list items for source %q: %w", sourceID, err)
	}
	var count int
	for _, item := range items {
		if item.Status != "installed" {
			continue
		}
		if err := inst.UninstallItem(&item); err != nil {
			fmt.Printf("  Warning: uninstall %s: %v\n", item.ID, err)
			continue
		}
		count++
	}
	fmt.Printf("  Uninstalled %d items from source %s\n", count, sourceID)
	return nil
}
