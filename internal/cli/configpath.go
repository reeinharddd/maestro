package cli

import (
	"os"
	"path/filepath"

	"github.com/reeinharddd/okit/internal/config"
)

func OpenCodeConfigDir() string {
	return config.ConfigDir()
}

func opencodeConfigName() string {
	dir := OpenCodeConfigDir()
	if _, err := os.Stat(filepath.Join(dir, "opencode.json")); err == nil {
		return "opencode.json"
	}
	return "opencode.jsonc"
}

func OpenCodeConfigPath() string {
	return filepath.Join(OpenCodeConfigDir(), opencodeConfigName())
}

func OpenCodeEnvPath() string {
	return filepath.Join(OpenCodeConfigDir(), "opencode.env")
}

func OpenCodeDBPath() string {
	return filepath.Join(OpenCodeConfigDir(), "opencode-kit.db")
}
