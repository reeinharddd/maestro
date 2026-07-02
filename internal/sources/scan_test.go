package sources_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/reeinharrrd/maestro/internal/sources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllScanners(t *testing.T) {
	scanners := sources.AllScanners()
	assert.Len(t, scanners, 6)
}

func TestScanAll(t *testing.T) {
	t.Run("aggregates items from all scanners", func(t *testing.T) {
		tmp := t.TempDir()

		require.NoError(t, os.MkdirAll(filepath.Join(tmp, "skills"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(tmp, "skills", "my-skill.md"), []byte("# S"), 0644))
		require.NoError(t, os.MkdirAll(filepath.Join(tmp, "agents"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(tmp, "agents", "my-agent.md"), []byte("# A"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("# agents"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(tmp, "SKILL.md"), []byte("# root skill"), 0644))
		require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".opencode", "commands"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(tmp, ".opencode", "commands", "my-cmd.md"), []byte("# C"), 0644))
		require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".claude", "skills"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(tmp, ".claude", "skills", "cl-skill.md"), []byte("# CS"), 0644))

		items, err := sources.ScanAll(tmp)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(items), 8)
	})

	t.Run("returns empty for empty directory", func(t *testing.T) {
		tmp := t.TempDir()
		items, err := sources.ScanAll(tmp)
		require.NoError(t, err)
		assert.Empty(t, items)
	})

	t.Run("handles missing root path", func(t *testing.T) {
		_, err := sources.ScanAll("/nonexistent-path-12345")
		if err != nil {
			assert.Error(t, err)
		}
	})
}

func TestScanAll_DirScanner(t *testing.T) {
	tmp := t.TempDir()
	for _, d := range []string{"skills", "agents", "commands", "mcps", "plugins", "workflows"} {
		require.NoError(t, os.MkdirAll(filepath.Join(tmp, d), 0755))
	}
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "skills", "s.md"), []byte("# S"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "agents", "a.md"), []byte("# A"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "commands", "c.md"), []byte("# C"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "mcps", "m.md"), []byte("# M"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "plugins", "p.md"), []byte("# P"), 0644))

	items, err := sources.ScanAll(tmp)
	require.NoError(t, err)

	found := make(map[string]string)
	for _, it := range items {
		found[it.ID] = it.Type
	}
	assert.Equal(t, "skill", found["skills-s"])
	assert.Equal(t, "agent", found["agents-a"])
	assert.Equal(t, "command", found["commands-c"])
	assert.Equal(t, "mcp", found["mcps-m"])
	assert.Equal(t, "plugin", found["plugins-p"])
}

func TestScanAll_RootHarnessScanner(t *testing.T) {
	t.Run("detects AGENTS.md as harness", func(t *testing.T) {
		tmp := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmp, "AGENTS.md"), []byte("# agents"), 0644))

		items, err := sources.ScanAll(tmp)
		require.NoError(t, err)

		hasHarness := false
		for _, it := range items {
			if it.Type == "harness" {
				hasHarness = true
			}
		}
		assert.True(t, hasHarness)
	})

	t.Run("empty for plain repo", func(t *testing.T) {
		tmp := t.TempDir()
		items, err := sources.ScanAll(tmp)
		require.NoError(t, err)
		assert.Empty(t, items)
	})
}

func TestScanAll_OpenCodeScanner(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".opencode", "skills"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, ".opencode", "skills", "oc-skill.md"), []byte("# OC"), 0644))

	items, err := sources.ScanAll(tmp)
	require.NoError(t, err)

	found := false
	for _, it := range items {
		if it.Scanner == "opencode" {
			found = true
		}
	}
	assert.True(t, found)
}

func TestScanAll_ClaudeScanner(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, ".claude", "skills"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmp, ".claude", "skills", "cs.md"), []byte("# CS"), 0644))

	items, err := sources.ScanAll(tmp)
	require.NoError(t, err)

	found := false
	for _, it := range items {
		if it.Scanner == "claude" {
			found = true
		}
	}
	assert.True(t, found)
}

func TestScanAll_Dedup(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "SKILL.md"), []byte("# S"), 0644))

	items, err := sources.ScanAll(tmp)
	require.NoError(t, err)

	count := 0
	for _, it := range items {
		if it.ID == "root-SKILL" {
			count++
		}
	}
	assert.Equal(t, 1, count)
}
func TestSanitizeFile(t *testing.T) {
	t.Run("passes through clean files unchanged", func(t *testing.T) {
		sourceDir := t.TempDir()
		sourcePath := filepath.Join(sourceDir, "clean.md")
		require.NoError(t, os.WriteFile(sourcePath, []byte("# Clean Skill\n"), 0644))

		result, err := sources.SanitizeFile(sourcePath, "source-1")
		require.NoError(t, err)
		// Clean file returns original path
		assert.Equal(t, sourcePath, result)
	})

	t.Run("sanitizes hardcoded home paths", func(t *testing.T) {
		sourceDir := t.TempDir()
		sourcePath := filepath.Join(sourceDir, "skill-with-path.md")
		content := "# Skill\nPath: /home/reeinharrrd/projects/gstack/\n"
		require.NoError(t, os.WriteFile(sourcePath, []byte(content), 0644))

		result, err := sources.SanitizeFile(sourcePath, "source-2")
		require.NoError(t, err)
		// Processed file created
		assert.NotEqual(t, sourcePath, result)
		// Verify content was sanitized
		data, err := os.ReadFile(result)
		require.NoError(t, err)
		assert.NotContains(t, string(data), "/home/reeinharrrd")
		assert.Contains(t, string(data), "{MAESTRO_USER_HOME}")
	})

	t.Run("sanitizes $HOME references", func(t *testing.T) {
		sourceDir := t.TempDir()
		sourcePath := filepath.Join(sourceDir, "home-ref.md")
		content := "# Agent\nInstall: $HOME/.config/opencode/skills\n"
		require.NoError(t, os.WriteFile(sourcePath, []byte(content), 0644))

		result, err := sources.SanitizeFile(sourcePath, "source-3")
		require.NoError(t, err)
		assert.NotEqual(t, sourcePath, result)
		data, err := os.ReadFile(result)
		require.NoError(t, err)
		assert.Contains(t, string(data), "{MAESTRO_SANITIZED_PATH}")
	})
}
