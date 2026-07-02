package config

import (
	"os"
	"path/filepath"
)

func ConfigDir() string {
	if dir := os.Getenv("OPENCODE_CONFIG_DIR"); dir != "" {
		return dir
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "opencode")
	}
	return filepath.Join(os.Getenv("HOME"), ".config", "opencode")
}

func DataDir() string {
	if dir := os.Getenv("OPENCODE_DATA_DIR"); dir != "" {
		return dir
	}
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "maestro")
	}
	return filepath.Join(os.Getenv("HOME"), ".local", "share", "maestro")
}

func SourcesDir() string {
	return filepath.Join(DataDir(), "sources")
}

func SkillsDir() string {
	return filepath.Join(DataDir(), "skills")
}

func CacheDir() string {
	if dir := os.Getenv("OPENCODE_CACHE_DIR"); dir != "" {
		return dir
	}
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "maestro")
	}
	return filepath.Join(os.Getenv("HOME"), ".cache", "maestro")
}

func CredentialsDir() string {
	return filepath.Join(DataDir(), "credentials")
}

func EnsureDirs() error {
	dirs := []string{
		ConfigDir(),
		DataDir(),
		SourcesDir(),
		SkillsDir(),
		CacheDir(),
		CredentialsDir(),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}
	return nil
}
