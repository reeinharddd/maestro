//go:build ignore

package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

func main() {
	dbPath := filepath.Join(os.Getenv("HOME"), ".config/opencode/opencode-kit.db")
	database, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		fmt.Fprintf(os.Stderr, "db open: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	sourceID := os.Args[1]
	rows, err := database.Query("SELECT id, type, source_path, status FROM source_items WHERE source_id=?", sourceID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "query: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	var installed []string
	var notInstalled []string

	for rows.Next() {
		var id, typ, sourcePath, status string
		rows.Scan(&id, &typ, &sourcePath, &status)
		if typ != "skill" {
			continue
		}
		if status == "installed" {
			installed = append(installed, fmt.Sprintf("  %s (%s)", filepath.Base(sourcePath), id))
		} else {
			notInstalled = append(notInstalled, fmt.Sprintf("  %s (%s)", filepath.Base(sourcePath), id))
		}
	}

	fmt.Printf("Installed skills: %d\n", len(installed))
	fmt.Printf("Not installed skills: %d\n", len(notInstalled))
	fmt.Println("\n--- Not installed ---")
	for _, s := range notInstalled {
		fmt.Println(s)
	}
	fmt.Println("\n--- Installed ---")
	for _, s := range installed {
		fmt.Println(s)
	}

	fmt.Println("\n--- Skill Directories ---")
	dirs := make(map[string]int)
	rows2, _ := database.Query("SELECT source_path FROM source_items WHERE source_id=? AND type='skill'", sourceID)
	for rows2.Next() {
		var p string
		rows2.Scan(&p)
		rel := strings.TrimPrefix(p, filepath.Dir(p)+"/")
		parts := strings.Split(rel, "/")
		if len(parts) > 1 {
			dirs[parts[len(parts)-2]]++
		} else {
			dirs["root"]++
		}
	}
	rows2.Close()
	for d, c := range dirs {
		fmt.Printf("  %s: %d\n", d, c)
	}
}
