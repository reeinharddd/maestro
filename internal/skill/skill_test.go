package skill

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/reeinharrrd/maestro/internal/db"
	"github.com/reeinharrrd/maestro/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// parseFrontmatter
// ---------------------------------------------------------------------------

func TestParseFrontmatter_AllFields(t *testing.T) {
	data := []byte(`---
description: A test skill
category: testing
tags: go,test
triggers: build
---
# Content here
`)
	desc, cat, tags, triggers := parseFrontmatter(data)
	assert.Equal(t, "A test skill", desc)
	assert.Equal(t, "testing", cat)
	assert.Equal(t, "go,test", tags)
	assert.Equal(t, "build", triggers)
}

func TestParseFrontmatter_NoFrontmatter(t *testing.T) {
	data := []byte(`# Just content
no frontmatter here`)
	desc, cat, tags, triggers := parseFrontmatter(data)
	assert.Empty(t, desc)
	assert.Empty(t, cat)
	assert.Empty(t, tags)
	assert.Empty(t, triggers)
}

func TestParseFrontmatter_EmptyFrontmatter(t *testing.T) {
	data := []byte(`---
---
content`)
	desc, cat, tags, triggers := parseFrontmatter(data)
	assert.Empty(t, desc)
	assert.Empty(t, cat)
	assert.Empty(t, tags)
	assert.Empty(t, triggers)
}

func TestParseFrontmatter_PartialFields(t *testing.T) {
	data := []byte(`---
description: only description
---
body`)
	desc, cat, tags, triggers := parseFrontmatter(data)
	assert.Equal(t, "only description", desc)
	assert.Empty(t, cat)
	assert.Empty(t, tags)
	assert.Empty(t, triggers)
}

func TestParseFrontmatter_UnknownFieldsIgnored(t *testing.T) {
	data := []byte(`---
description: known
unknown: should be ignored
category: misc
---
`)
	desc, cat, _, _ := parseFrontmatter(data)
	assert.Equal(t, "known", desc)
	assert.Equal(t, "misc", cat)
}

// ---------------------------------------------------------------------------
// detectSource
// ---------------------------------------------------------------------------

func TestDetectSource_MultiLevel(t *testing.T) {
	src := detectSource("/root", "/root/mysource/myskill.md")
	assert.Equal(t, "mysource", src)
}

func TestDetectSource_RootLevel(t *testing.T) {
	src := detectSource("/root", "/root/myskill.md")
	assert.Equal(t, "unknown", src)
}

func TestDetectSource_DeepPath(t *testing.T) {
	src := detectSource("/base", "/base/a/b/c/skill.md")
	assert.Equal(t, "a", src)
}

// ---------------------------------------------------------------------------
// Manager (filesystem operations)
// ---------------------------------------------------------------------------

func TestManager_Install(t *testing.T) {
	skillsDir := t.TempDir()
	m := NewManager(skillsDir)

	sourceFile := filepath.Join(t.TempDir(), "test-skill.md")
	require.NoError(t, os.WriteFile(sourceFile, []byte("# skill"), 0644))

	err := m.Install("my-skill", sourceFile)
	require.NoError(t, err)

	target := filepath.Join(skillsDir, "my-skill")
	info, err := os.Lstat(target)
	require.NoError(t, err)
	assert.True(t, info.Mode()&os.ModeSymlink != 0, "expected a symlink")

	got, err := os.Readlink(target)
	require.NoError(t, err)
	assert.Equal(t, sourceFile, got)
}

func TestManager_Install_ReplaceSymlink(t *testing.T) {
	skillsDir := t.TempDir()
	m := NewManager(skillsDir)

	first := filepath.Join(t.TempDir(), "first.md")
	require.NoError(t, os.WriteFile(first, []byte("# first"), 0644))
	second := filepath.Join(t.TempDir(), "second.md")
	require.NoError(t, os.WriteFile(second, []byte("# second"), 0644))

	require.NoError(t, m.Install("replaced", first))
	require.NoError(t, m.Install("replaced", second))

	got, _ := os.Readlink(filepath.Join(skillsDir, "replaced"))
	assert.Equal(t, second, got)
}

func TestManager_Install_ErrorOnRealFile(t *testing.T) {
	skillsDir := t.TempDir()
	m := NewManager(skillsDir)

	realFile := filepath.Join(skillsDir, "blocker")
	require.NoError(t, os.WriteFile(realFile, []byte("real"), 0644))

	sourceFile := filepath.Join(t.TempDir(), "skill.md")
	require.NoError(t, os.WriteFile(sourceFile, []byte("# skill"), 0644))

	err := m.Install("blocker", sourceFile)
	assert.ErrorContains(t, err, "already exists and is not a symlink")
}

func TestManager_Remove(t *testing.T) {
	skillsDir := t.TempDir()
	m := NewManager(skillsDir)

	sourceFile := filepath.Join(t.TempDir(), "skill.md")
	require.NoError(t, os.WriteFile(sourceFile, []byte("# skill"), 0644))
	require.NoError(t, m.Install("to-remove", sourceFile))

	err := m.Remove("to-remove")
	require.NoError(t, err)

	_, err = os.Lstat(filepath.Join(skillsDir, "to-remove"))
	assert.True(t, os.IsNotExist(err), "should not exist after removal")
}

func TestManager_Remove_NotInstalled(t *testing.T) {
	m := NewManager(t.TempDir())
	err := m.Remove("never-installed")
	assert.ErrorContains(t, err, "not installed")
}

func TestManager_Remove_RefusesRealFile(t *testing.T) {
	skillsDir := t.TempDir()
	m := NewManager(skillsDir)

	realFile := filepath.Join(skillsDir, "real-file")
	require.NoError(t, os.WriteFile(realFile, []byte("content"), 0644))

	err := m.Remove("real-file")
	assert.ErrorContains(t, err, "not a symlink")
}

// ---------------------------------------------------------------------------
// Scanner.ScanDir
// ---------------------------------------------------------------------------

func TestScanner_ScanDir_FindsMDFiles(t *testing.T) {
	d, err := db.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { d.Close() })

	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "src"), 0755))

	skillFile := filepath.Join(root, "src", "test-skill.md")
	require.NoError(t, os.WriteFile(skillFile, []byte(`---
description: my skill
category: util
tags: go
triggers: build
---
# Hello
`), 0644))

	s := NewScanner(d)
	count, err := s.ScanDir(root)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestScanner_ScanDir_SkipsNonMarkdown(t *testing.T) {
	d, err := db.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { d.Close() })

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "readme.txt"), []byte("hello"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "script.sh"), []byte("#!/bin/sh"), 0644))

	s := NewScanner(d)
	count, err := s.ScanDir(root)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestScanner_ScanDir_SkipsHiddenDirs(t *testing.T) {
	d, err := db.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { d.Close() })

	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".hidden"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".hidden", "skill.md"), []byte("---\ndescription: hidden\n---"), 0644))
	// Visible file at root should still be picked up
	require.NoError(t, os.WriteFile(filepath.Join(root, "visible.md"), []byte("---\ndescription: visible\n---"), 0644))

	s := NewScanner(d)
	count, err := s.ScanDir(root)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "should skip hidden dir but find visible.md")
}

func TestScanner_ScanDir_UpsertsSkillToDB(t *testing.T) {
	d, err := db.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { d.Close() })

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "greeter.md"), []byte(`---
description: A greeting skill
category: demo
tags: hello,world
triggers: greet
---
# Greeter
`), 0644))

	s := NewScanner(d)
	_, err = s.ScanDir(root)
	require.NoError(t, err)

	skills, err := d.ListSkills()
	require.NoError(t, err)
	require.Len(t, skills, 1)

	assert.Equal(t, "greeter", skills[0].ID)
	assert.Equal(t, "A greeting skill", skills[0].Description)
	assert.Equal(t, "demo", skills[0].Category)
	assert.Equal(t, "hello,world", skills[0].Tags)
	assert.Equal(t, "greet", skills[0].Triggers)
	assert.Equal(t, "skill", skills[0].Type)
	assert.Equal(t, "active", skills[0].Status)
	assert.NotEmpty(t, skills[0].Hash)
	assert.Greater(t, skills[0].SizeBytes, int64(0))
}

// ---------------------------------------------------------------------------
// Estimator
// ---------------------------------------------------------------------------

func TestEstimator_Estimate_Empty(t *testing.T) {
	d, err := db.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { d.Close() })

	e := NewEstimator(d)
	est, err := e.Estimate()
	require.NoError(t, err)

	assert.Equal(t, int64(0), est.TotalBytes)
	assert.Equal(t, 0, est.TotalSkills)
	assert.Empty(t, est.BySource)
	assert.Empty(t, est.ByCategory)
	assert.Empty(t, est.Heaviest)
}

func TestEstimator_Estimate_AggregatesBySourceAndCategory(t *testing.T) {
	d, err := db.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { d.Close() })

	// Insert skills directly into DB
	for _, sk := range []struct {
		id, source, category string
		bytes                int64
	}{
		{"s1", "github", "dev", 100},
		{"s2", "github", "dev", 200},
		{"s3", "local", "ops", 150},
		{"s4", "local", "dev", 50},
	} {
		require.NoError(t, d.UpsertSkill(&models.Skill{
			ID:        sk.id,
			Source:    sk.source,
			Category:  sk.category,
			SizeBytes: sk.bytes,
			Type:      "skill",
			Status:    "active",
		}))
	}

	e := NewEstimator(d)
	est, err := e.Estimate()
	require.NoError(t, err)

	assert.Equal(t, int64(500), est.TotalBytes)
	assert.Equal(t, 4, est.TotalSkills)
	assert.Equal(t, int64(300), est.BySource["github"])
	assert.Equal(t, int64(200), est.BySource["local"])
	assert.Equal(t, int64(350), est.ByCategory["dev"])
	assert.Equal(t, int64(150), est.ByCategory["ops"])
}
func TestEstimator_Estimate_HeaviestOrdering(t *testing.T) {
	d, err := db.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { d.Close() })

	for i := 0; i < 15; i++ {
		require.NoError(t, d.UpsertSkill(&models.Skill{
			ID:        intToSkillID(i),
			Source:    "test",
			SizeBytes: int64((15 - i) * 10), // descending: 140, 130, ...
			Type:      "skill",
			Status:    "active",
		}))
	}

	e := NewEstimator(d)
	est, err := e.Estimate()
	require.NoError(t, err)

	require.Len(t, est.Heaviest, 10)
	// Heaviest should be sorted descending
	for i := 1; i < len(est.Heaviest); i++ {
		assert.GreaterOrEqual(t, est.Heaviest[i-1].SizeBytes, est.Heaviest[i].SizeBytes,
			"heaviest list must be sorted descending")
	}
	assert.Equal(t, int64(150), est.Heaviest[0].SizeBytes, "first should be the heaviest")
	assert.Equal(t, int64(60), est.Heaviest[9].SizeBytes, "tenth should be the 10th heaviest")
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func intToSkillID(i int) string {
	return string(rune('a' + i))
}
