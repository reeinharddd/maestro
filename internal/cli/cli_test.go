package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMaskKey_ShortKey(t *testing.T) {
	got := maskKey("abc")
	want := "***"
	if got != want {
		t.Errorf("maskKey(\"abc\") = %q, want %q", got, want)
	}
}

func TestMaskKey_LongKey(t *testing.T) {
	got := maskKey("sk-ant-abcdefghijklmnop")
	if len(got) != 23 {
		t.Errorf("maskKey len = %d, want 22", len(got))
	}
	if got[:4] != "sk-a" || got[len(got)-4:] != "mnop" {
		t.Errorf("maskKey long = %q, want sk-a****mnop pattern", got)
	}
}

func TestMaskKey_Exact8(t *testing.T) {
	got := maskKey("12345678")
	want := "********"
	if got != want {
		t.Errorf("maskKey(\"12345678\") = %q, want %q", got, want)
	}
}

func TestMaskKey_Empty(t *testing.T) {
	got := maskKey("")
	want := ""
	if got != want {
		t.Errorf("maskKey empty = %q, want %q", got, want)
	}
}

func TestParseEnvFile_Valid(t *testing.T) {
	content := `OPENAI_API_KEY=sk-abc123
MISTRAL_API_KEY=xyz789
EMPTY_LINE=

# comment
`
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	vars, err := parseEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if vars["OPENAI_API_KEY"] != "sk-abc123" {
		t.Errorf("OPENAI_API_KEY = %q, want %q", vars["OPENAI_API_KEY"], "sk-abc123")
	}
	if vars["MISTRAL_API_KEY"] != "xyz789" {
		t.Errorf("MISTRAL_API_KEY = %q, want %q", vars["MISTRAL_API_KEY"], "xyz789")
	}
	if val, ok := vars["EMPTY_LINE"]; ok && val != "" {
		t.Errorf("EMPTY_LINE should be empty, got %q", val)
	}
	if _, ok := vars["# comment"]; ok {
		t.Error("comment should not be parsed")
	}
}

func TestParseEnvFile_ExportPrefix(t *testing.T) {
	content := `export OPENAI_API_KEY="sk-abc"`
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	vars, err := parseEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if vars["OPENAI_API_KEY"] != "sk-abc" {
		t.Errorf("export-prefixed key = %q, want %q", vars["OPENAI_API_KEY"], "sk-abc")
	}
}

func TestParseEnvFile_QuotedValues(t *testing.T) {
	content := `KEY1='single-quoted'
KEY2="double-quoted"
KEY3=no-quotes`
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	vars, err := parseEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if vars["KEY1"] != "single-quoted" {
		t.Errorf("KEY1 = %q, want %q", vars["KEY1"], "single-quoted")
	}
	if vars["KEY2"] != "double-quoted" {
		t.Errorf("KEY2 = %q, want %q", vars["KEY2"], "double-quoted")
	}
	if vars["KEY3"] != "no-quotes" {
		t.Errorf("KEY3 = %q, want %q", vars["KEY3"], "no-quotes")
	}
}

func TestParseEnvFile_Missing(t *testing.T) {
	_, err := parseEnvFile("/nonexistent/.env")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestFindConfigPath_PrefersConfigDir(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "data", "okit.db")
	os.MkdirAll(filepath.Dir(dbPath), 0755)

	configDir := "/home/reeinharrrd/.config/opencode"
	jsoncPath := filepath.Join(configDir, "opencode.jsonc")
	os.MkdirAll(configDir, 0755)
	os.WriteFile(jsoncPath, []byte("{}"), 0644)
	defer os.Remove(jsoncPath)

	d, err := openDB(&dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	got := findConfigPath(d)
	if got != jsoncPath {
		t.Errorf("findConfigPath = %q, want %q", got, jsoncPath)
	}
}

func TestFindConfigPath_FallbackToDBDir(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "okit.db")
	jsoncPath := filepath.Join(dir, "opencode.jsonc")
	os.WriteFile(jsoncPath, []byte("{}"), 0644)

	d, err := openDB(&dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	got := findConfigPath(d)
	if got != jsoncPath {
		t.Errorf("findConfigPath = %q, want %q", got, jsoncPath)
	}
}

func TestCheckFileExists_Found(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("hello"), 0644)

	got := checkFileExists(path)
	if got == "(not found)" {
		t.Errorf("checkFileExists found file, got %q", got)
	}
}
