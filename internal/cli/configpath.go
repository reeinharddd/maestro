package cli

import (
	"os"
	"path/filepath"
)

func OpenCodeConfigDir() string {
	if dir := os.Getenv("OPENCODE_CONFIG_DIR"); dir != "" {
		return dir
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "opencode")
	}
	return filepath.Join(os.Getenv("HOME"), ".config", "opencode")
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
