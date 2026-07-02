package skill

import (
	"fmt"
	"os"
	"path/filepath"
)

// Manager creates and removes symlinks in the opencode skills directory.
type Manager struct {
	opencodeSkillsDir string
}

// NewManager creates a Manager that operates on the given opencode skills path.
func NewManager(opencodeSkillsDir string) *Manager {
	return &Manager{opencodeSkillsDir: opencodeSkillsDir}
}

// Install creates a symlink at opencodeSkillsDir/name pointing to sourcePath.
// If a symlink already exists it is replaced; a real file/dir causes an error.
func (m *Manager) Install(name, sourcePath string) error {
	targetPath := filepath.Join(m.opencodeSkillsDir, name)

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

// Remove deletes the symlink at opencodeSkillsDir/name.
// Only removes symlinks — refuses to delete real files or directories.
func (m *Manager) Remove(name string) error {
	targetPath := filepath.Join(m.opencodeSkillsDir, name)

	info, err := os.Lstat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("skill %q not installed", name)
		}
		return fmt.Errorf("stat %s: %w", targetPath, err)
	}

	if info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("%s is not a symlink (refusing to remove)", targetPath)
	}

	return os.Remove(targetPath)
}
