package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/reeinharrrd/maestro/internal/config"
)

func TestConfigDir(t *testing.T) {

	home := os.Getenv("HOME")

	tests := []struct {
		name        string
		opencodeDir string
		xdgConfig   string
		want        string
	}{
		{
			name:        "opencode_env_override",
			opencodeDir: "/custom/config",
			xdgConfig:   "/xdg/config",
			want:        "/custom/config",
		},
		{
			name:        "xdg_fallback",
			opencodeDir: "",
			xdgConfig:   "/xdg/config",
			want:        filepath.Join("/xdg/config", "opencode"),
		},
		{
			name:        "home_fallback",
			opencodeDir: "",
			xdgConfig:   "",
			want:        filepath.Join(home, ".config", "opencode"),
		},
		{
			name:        "relative_path",
			opencodeDir: "relative/path",
			xdgConfig:   "",
			want:        "relative/path",
		},
		{
			name:        "opencode_precedence_over_xdg",
			opencodeDir: "/precedence",
			xdgConfig:   "/ignored",
			want:        "/precedence",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("OPENCODE_CONFIG_DIR", tt.opencodeDir)
			t.Setenv("XDG_CONFIG_HOME", tt.xdgConfig)
			got := config.ConfigDir()
			if got != tt.want {
				t.Errorf("ConfigDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDataDir(t *testing.T) {

	home := os.Getenv("HOME")

	tests := []struct {
		name    string
		dataDir string
		xdgData string
		want    string
	}{
		{
			name:    "opencode_data_override",
			dataDir: "/custom/data",
			xdgData: "/xdg/data",
			want:    "/custom/data",
		},
		{
			name:    "xdg_data_fallback",
			dataDir: "",
			xdgData: "/xdg/data",
			want:    filepath.Join("/xdg/data", "maestro"),
		},
		{
			name:    "home_data_fallback",
			dataDir: "",
			xdgData: "",
			want:    filepath.Join(home, ".local", "share", "maestro"),
		},
		{
			name:    "relative_data_path",
			dataDir: "relative/data",
			xdgData: "",
			want:    "relative/data",
		},
		{
			name:    "opencode_precedence_over_xdg",
			dataDir: "/precedence",
			xdgData: "/ignored",
			want:    "/precedence",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("OPENCODE_DATA_DIR", tt.dataDir)
			t.Setenv("XDG_DATA_HOME", tt.xdgData)
			got := config.DataDir()
			if got != tt.want {
				t.Errorf("DataDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCacheDir(t *testing.T) {

	home := os.Getenv("HOME")

	tests := []struct {
		name     string
		cacheDir string
		xdgCache string
		want     string
	}{
		{
			name:     "opencode_cache_override",
			cacheDir: "/custom/cache",
			xdgCache: "/xdg/cache",
			want:     "/custom/cache",
		},
		{
			name:     "xdg_cache_fallback",
			cacheDir: "",
			xdgCache: "/xdg/cache",
			want:     filepath.Join("/xdg/cache", "maestro"),
		},
		{
			name:     "home_cache_fallback",
			cacheDir: "",
			xdgCache: "",
			want:     filepath.Join(home, ".cache", "maestro"),
		},
		{
			name:     "opencode_precedence_over_xdg",
			cacheDir: "/precedence",
			xdgCache: "/ignored",
			want:     "/precedence",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("OPENCODE_CACHE_DIR", tt.cacheDir)
			t.Setenv("XDG_CACHE_HOME", tt.xdgCache)
			got := config.CacheDir()
			if got != tt.want {
				t.Errorf("CacheDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSubdirsDerivedFromDataDir(t *testing.T) {

	t.Setenv("OPENCODE_DATA_DIR", "/test/data")
	t.Setenv("XDG_DATA_HOME", "")

	t.Run("SourcesDir", func(t *testing.T) {
		want := filepath.Join("/test/data", "sources")
		if got := config.SourcesDir(); got != want {
			t.Errorf("SourcesDir() = %q, want %q", got, want)
		}
	})

	t.Run("SkillsDir", func(t *testing.T) {
		want := filepath.Join("/test/data", "skills")
		if got := config.SkillsDir(); got != want {
			t.Errorf("SkillsDir() = %q, want %q", got, want)
		}
	})

	t.Run("CredentialsDir", func(t *testing.T) {
		want := filepath.Join("/test/data", "credentials")
		if got := config.CredentialsDir(); got != want {
			t.Errorf("CredentialsDir() = %q, want %q", got, want)
		}
	})
}

func TestSourcesDir_follows_XDG_DATA_HOME(t *testing.T) {
	t.Setenv("OPENCODE_DATA_DIR", "")
	t.Setenv("XDG_DATA_HOME", "/xdg/data")

	want := filepath.Join("/xdg/data", "maestro", "sources")
	if got := config.SourcesDir(); got != want {
		t.Errorf("SourcesDir() = %q, want %q", got, want)
	}
}

func TestSkillsDir_follows_OPENCODE_DATA_DIR(t *testing.T) {
	t.Setenv("OPENCODE_DATA_DIR", "/custom/data")
	t.Setenv("XDG_DATA_HOME", "")

	want := filepath.Join("/custom/data", "skills")
	if got := config.SkillsDir(); got != want {
		t.Errorf("SkillsDir() = %q, want %q", got, want)
	}
}

func TestEnsureDirs_creates_all_dirs(t *testing.T) {

	base := t.TempDir()
	t.Setenv("OPENCODE_CONFIG_DIR", filepath.Join(base, "config"))
	t.Setenv("OPENCODE_DATA_DIR", filepath.Join(base, "data"))
	t.Setenv("OPENCODE_CACHE_DIR", filepath.Join(base, "cache"))
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")

	if err := config.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs() first call: %v", err)
	}

	expected := []string{
		config.ConfigDir(),
		config.DataDir(),
		config.SourcesDir(),
		config.SkillsDir(),
		config.CacheDir(),
		config.CredentialsDir(),
	}

	for _, d := range expected {
		fi, err := os.Stat(d)
		if err != nil {
			t.Errorf("expected %q to exist: %v", d, err)
			continue
		}
		if !fi.IsDir() {
			t.Errorf("%q is not a directory", d)
		}
	}
}

func TestEnsureDirs_idempotent(t *testing.T) {

	base := t.TempDir()
	t.Setenv("OPENCODE_CONFIG_DIR", filepath.Join(base, "config"))
	t.Setenv("OPENCODE_DATA_DIR", filepath.Join(base, "data"))
	t.Setenv("OPENCODE_CACHE_DIR", filepath.Join(base, "cache"))
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")

	if err := config.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs() first call: %v", err)
	}

	if err := config.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs() second call should be idempotent: %v", err)
	}
}
