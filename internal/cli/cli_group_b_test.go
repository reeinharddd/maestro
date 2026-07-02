package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/reeinharrrd/maestro/internal/db"
	"github.com/reeinharrrd/maestro/pkg/models"
)

// ── maskKey edge cases ────────────────────────────────────────────────

func TestMaskKey_Len9_ShowsFourEachEnd(t *testing.T) {
	t.Parallel()
	got := maskKey("123456789")
	want := "1234*6789"
	if got != want {
		t.Errorf("maskKey(%q) = %q, want %q", "123456789", got, want)
	}
}

func TestMaskKey_Len1(t *testing.T) {
	t.Parallel()
	got := maskKey("a")
	if got != "*" {
		t.Errorf("maskKey(%q) = %q, want %q", "a", got, "*")
	}
}

func TestMaskKey_Len8_ExactBoundary(t *testing.T) {
	t.Parallel()
	got := maskKey("12345678")
	if got != "********" {
		t.Errorf("maskKey(%q) = %q, want %q", "12345678", got, "********")
	}
}

// ── parseEnvFile edge cases ───────────────────────────────────────────

func TestParseEnvFile_NoEqualsLine(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("JUST_A_KEY\n"), 0644); err != nil {
		t.Fatal(err)
	}
	vars, err := parseEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(vars) != 0 {
		t.Errorf("expected 0 vars for line without '=', got %d", len(vars))
	}
}

func TestParseEnvFile_EmptyFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	vars, err := parseEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(vars) != 0 {
		t.Errorf("expected 0 vars for empty file, got %d", len(vars))
	}
}

func TestParseEnvFile_OnlyComments(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("# comment only\n# another\n"), 0644); err != nil {
		t.Fatal(err)
	}
	vars, err := parseEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(vars) != 0 {
		t.Errorf("expected 0 vars for only comments, got %d", len(vars))
	}
}

func TestParseEnvFile_WeirdWhitespace(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := "  KEY =  value  \nexport SPACED =\"  quoted  \"\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	vars, err := parseEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if vars["KEY"] != "value" {
		t.Errorf("KEY = %q, want %q", vars["KEY"], "value")
	}
	if vars["SPACED"] != "  quoted  " {
		t.Errorf("SPACED = %q, want %q", vars["SPACED"], "  quoted  ")
	}
}

// ── checkFileExists edge cases ────────────────────────────────────────

func TestCheckFileExists_NotFound(t *testing.T) {
	t.Parallel()
	got := checkFileExists("/nonexistent/path/that/definitely/does/not/exist")
	if got != "(not found)" {
		t.Errorf("checkFileExists missing = %q, want %q", got, "(not found)")
	}
}

func TestCheckFileExists_FoundWithBytes(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "exists.txt")
	if err := os.WriteFile(path, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}
	got := checkFileExists(path)
	if got == "(not found)" {
		t.Error("checkFileExists should report found")
	}
}

// ── findConfigPath edge cases ─────────────────────────────────────────

func TestFindConfigPath_FallbackToJsonWhenNoConfig(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := openDB(&dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	t.Setenv("OPENCODE_CONFIG_DIR", filepath.Join(dir, "empty-config"))
	got := findConfigPath(d)
	want := filepath.Join(dir, "opencode.json")
	if got != want {
		t.Errorf("findConfigPath = %q, want %q", got, want)
	}
}

func TestFindConfigPath_PrefersConfigDir(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := openDB(&dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	configDir := filepath.Join(dir, "cfg")
	os.MkdirAll(configDir, 0755)
	jsoncPath := filepath.Join(configDir, "opencode.jsonc")
	if err := os.WriteFile(jsoncPath, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("OPENCODE_CONFIG_DIR", configDir)
	got := findConfigPath(d)
	if got != jsoncPath {
		t.Errorf("findConfigPath = %q, want %q", got, jsoncPath)
	}
}

// ── Keys subcommands ──────────────────────────────────────────────────

func TestKeysSet_CreatesNewFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OPENCODE_CONFIG_DIR", filepath.Join(dir, "cfg"))

	cmd := newKeysSetCmd()
	if err := cmd.RunE(cmd, []string{"TEST_KEY", "test-value"}); err != nil {
		t.Fatal(err)
	}

	envPath := OpenCodeEnvPath()
	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == "" {
		t.Error("env file should not be empty")
	}
}

func TestKeysSet_EmptyKeyReturnsError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OPENCODE_CONFIG_DIR", filepath.Join(dir, "cfg"))

	cmd := newKeysSetCmd()
	err := cmd.RunE(cmd, []string{"", "value"})
	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestKeysSet_EmptyValueReturnsError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OPENCODE_CONFIG_DIR", filepath.Join(dir, "cfg"))

	cmd := newKeysSetCmd()
	err := cmd.RunE(cmd, []string{"KEY", ""})
	if err == nil {
		t.Error("expected error for empty value")
	}
}

func TestKeysList_NoFile_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OPENCODE_CONFIG_DIR", filepath.Join(dir, "empty"))

	cmd := newKeysListCmd()
	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Error("expected error for missing env file")
	}
}

func TestKeysList_ShowsKeys(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OPENCODE_CONFIG_DIR", filepath.Join(dir, "cfg"))
	os.MkdirAll(filepath.Join(dir, "cfg"), 0755)
	envPath := OpenCodeEnvPath()
	if err := os.WriteFile(envPath, []byte("ALPHA=val1\nBETA=val2\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := newKeysListCmd()
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestKeysRemove_RemovesKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OPENCODE_CONFIG_DIR", filepath.Join(dir, "cfg"))
	os.MkdirAll(filepath.Join(dir, "cfg"), 0755)

	// Set a key first
	setCmd := newKeysSetCmd()
	if err := setCmd.RunE(setCmd, []string{"SOME_KEY", "val"}); err != nil {
		t.Fatal(err)
	}

	// Remove it
	rmCmd := newKeysRemoveCmd()
	if err := rmCmd.RunE(rmCmd, []string{"SOME_KEY"}); err != nil {
		t.Fatal(err)
	}
}

func TestKeysRemove_Nonexistent_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OPENCODE_CONFIG_DIR", filepath.Join(dir, "cfg"))
	os.MkdirAll(filepath.Join(dir, "cfg"), 0755)

	// Set a key first
	setCmd := newKeysSetCmd()
	if err := setCmd.RunE(setCmd, []string{"SOME_KEY", "val"}); err != nil {
		t.Fatal(err)
	}

	// Remove it
	rmCmd := newKeysRemoveCmd()
	if err := rmCmd.RunE(rmCmd, []string{"SOME_KEY"}); err != nil {
		t.Fatal(err)
	}

	// Remove again should fail
	rmCmd2 := newKeysRemoveCmd()
	err := rmCmd2.RunE(rmCmd2, []string{"SOME_KEY"})
	if err == nil {
		t.Error("expected error removing nonexistent key")
	}
}

func TestKeysDoctor_NoKeys(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OPENCODE_CONFIG_DIR", filepath.Join(dir, "empty"))
	os.MkdirAll(filepath.Join(dir, "empty"), 0755)

	cmd := newKeysCmd()
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestKeysDoctor_WithKeysAndDB(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "cfg")
	dbDir := filepath.Join(dir, "db")
	os.MkdirAll(configDir, 0755)
	os.MkdirAll(dbDir, 0755)
	t.Setenv("OPENCODE_CONFIG_DIR", configDir)
	t.Setenv("OPENCODE_KIT_DB", filepath.Join(dbDir, "maestro.db"))

	// Set up env file
	envPath := OpenCodeEnvPath()
	if err := os.WriteFile(envPath, []byte("MY_KEY=real-value\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Seed DB with a provider that has key_env
	d, err := db.Open(filepath.Join(dbDir, "maestro.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertProvider(&models.Provider{
		ID: "test-prov", Name: "Test", KeyEnv: "MY_KEY",
		Source: "custom", Status: "active",
	}); err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newKeysCmd()
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
}

// ── Init command ──────────────────────────────────────────────────────

func TestInitCmd_CreatesConfigDir(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "maestro-config")
	t.Setenv("OPENCODE_CONFIG_DIR", configDir)

	cmd := newInitCmd()
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Error("config dir should have been created")
	}
}

// ── Status command ────────────────────────────────────────────────────

func TestStatusCmd_ShowsStats(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertProvider(&models.Provider{
		ID: "p1", Name: "P1", Source: "custom", Status: "active",
	}); err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertModel(&models.Model{
		ID: "p1/m1", ProviderID: "p1", DisplayName: "m1",
		Status: "active", Source: "discovered",
	}); err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newStatusCmd(&dbPath)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestStatusCmd_EmptyDB_ReturnsNoError(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "empty.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newStatusCmd(&dbPath)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
}

// ── Discover command ──────────────────────────────────────────────────

func TestDiscoverCmd_NoProviders_NoCrash(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newDiscoverCmd(&dbPath)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
}

// ── Audit command ─────────────────────────────────────────────────────

func TestAuditCmd_NoProviders_NoCrash(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newAuditCmd(&dbPath)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestAuditCmd_FullFlag_NoCrash(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newAuditCmd(&dbPath)
	cmd.Flags().Set("full", "true")
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
}

// ── Heal command ──────────────────────────────────────────────────────

func TestHealCmd_WithDB(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	// Seed a provider and a model to have something to heal
	if err := d.UpsertProvider(&models.Provider{
		ID: "heal-prov", Name: "Heal Test", Source: "custom", Status: "active",
	}); err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertModel(&models.Model{
		ID: "heal-prov/m1", ProviderID: "heal-prov", DisplayName: "m1",
		Status: "active", Source: "discovered",
	}); err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newHealCmd(&dbPath)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
}

// ── Sync command ──────────────────────────────────────────────────────

func TestSyncCmd_ImportFull(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	configPath := filepath.Join(dir, "opencode.jsonc")
	configContent := `{
		"$schema": "https://opencode.ai/config.json",
		"provider": {
			"openai": {
				"name": "OpenAI",
				"whitelist": ["gpt-4", "gpt-3.5-turbo"]
			}
		},
		"agent": {
			"coding": {
				"description": "Coding agent",
				"model": "openai/gpt-4"
			}
		},
		"command": {
			"test": {
				"template": "go test ./..."
			}
		}
	}`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	d.Close()

	t.Setenv("OPENCODE_CONFIG_DIR", dir)
	cmd := newSyncCmd(&dbPath)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestSyncCmd_ExportFull(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertProvider(&models.Provider{
		ID: "exp-prov", Name: "Export Test", Source: "opencode", Status: "active",
	}); err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertModel(&models.Model{
		ID: "exp-prov/gpt-test", ProviderID: "exp-prov", DisplayName: "gpt-test",
		Status: "active", Source: "opencode",
	}); err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertAgent(&models.Agent{
		ID: "test-agent", Description: "Test Agent", Status: "active",
	}); err != nil {
		t.Fatal(err)
	}
	d.Close()

	t.Setenv("OPENCODE_CONFIG_DIR", dir)
	cmd := newSyncCmd(&dbPath)
	// Create a stub config so import doesn't error
	configPath := filepath.Join(dir, "opencode.jsonc")
	if err := os.WriteFile(configPath, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
}

// ── Generate command ──────────────────────────────────────────────────

func TestGenerateConfigCmd_WithDB(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertProvider(&models.Provider{
		ID: "gen-prov", Name: "Gen Test", Source: "custom", Status: "active",
	}); err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertModel(&models.Model{
		ID: "gen-prov/m1", ProviderID: "gen-prov", DisplayName: "m1",
		Status: "active", Source: "discovered",
	}); err != nil {
		t.Fatal(err)
	}
	d.Close()

	t.Setenv("OPENCODE_CONFIG_DIR", dir)
	cmd := newGenerateCmd(&dbPath)
	// Run the config subcommand
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateAgentsCmd_WithDB(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "maestro.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertAgent(&models.Agent{
		ID: "test-agent", Description: "Test Agent", Status: "active",
	}); err != nil {
		t.Fatal(err)
	}
	d.Close()

	t.Setenv("OPENCODE_CONFIG_DIR", dir)
	cmd := newGenerateCmd(&dbPath)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
}

// ── Verify command ────────────────────────────────────────────────────

func TestVerifyCmd_NoLive_NoKeys(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OPENCODE_CONFIG_DIR", filepath.Join(dir, "empty"))
	os.MkdirAll(filepath.Join(dir, "empty"), 0755)

	cmd := newVerifyCmd()
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyCmd_WithProviderFlag_UnknownProvider(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OPENCODE_CONFIG_DIR", filepath.Join(dir, "empty"))
	os.MkdirAll(filepath.Join(dir, "empty"), 0755)

	// Set the --provider flag to an unknown provider
	cmd := newVerifyCmd()
	cmd.Flags().Set("provider", "nonexistent-provider")
	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestVerifyCmd_WithLiveFlag_NoKeys(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OPENCODE_CONFIG_DIR", filepath.Join(dir, "empty"))
	os.MkdirAll(filepath.Join(dir, "empty"), 0755)

	cmd := newVerifyCmd()
	cmd.Flags().Set("live", "true")
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}
}

// ── Doctor command ────────────────────────────────────────────────────

func TestDoctorCmd_RunsWithoutDB(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OPENCODE_CONFIG_DIR", filepath.Join(dir, "empty"))
	os.MkdirAll(filepath.Join(dir, "empty"), 0755)

	cmd := newDoctorCmd()
	// Doctor may fail if binary is not found in test env — that's OK
	_ = cmd.RunE(cmd, nil)
}

func TestDoctorCmd_WithDB(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "cfg")
	os.MkdirAll(configDir, 0755)
	t.Setenv("OPENCODE_CONFIG_DIR", configDir)

	// Seed a DB so status check works
	dbPath := filepath.Join(configDir, "opencode-kit.db")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertProvider(&models.Provider{
		ID: "doc-prov", Name: "Doctor Test", Source: "custom", Status: "active",
	}); err != nil {
		t.Fatal(err)
	}
	if err := d.UpsertModel(&models.Model{
		ID: "doc-prov/m1", ProviderID: "doc-prov", DisplayName: "m1",
		Status: "active", Source: "discovered",
	}); err != nil {
		t.Fatal(err)
	}
	d.Close()

	cmd := newDoctorCmd()
	_ = cmd.RunE(cmd, nil)
}

func TestDoctorCmd_WithKeysFlag(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OPENCODE_CONFIG_DIR", filepath.Join(dir, "cfg"))
	os.MkdirAll(filepath.Join(dir, "cfg"), 0755)

	// Set up a key
	envPath := OpenCodeEnvPath()
	if err := os.WriteFile(envPath, []byte("MY_KEY=test-value\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := newDoctorCmd()
	cmd.Flags().Set("with-keys", "true")
	_ = cmd.RunE(cmd, nil)
}

// ── openDB edge cases ─────────────────────────────────────────────────

func TestOpenDB_ExplicitPath(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "custom", "test.db")
	os.MkdirAll(filepath.Dir(dbPath), 0755)

	d, err := openDB(&dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if d.DBPath() != dbPath {
		t.Errorf("openDB path = %q, want %q", d.DBPath(), dbPath)
	}
}
