package sources

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/reeinharrrd/maestro/internal/config"
)

// Patterns for detecting hardcoded absolute paths that need sanitization.
var hardcodedPathPatterns = []*regexp.Regexp{
	// /home/username/... or /Users/username/... patterns
	regexp.MustCompile(`(?:file://)?/[/a-zA-Z0-9_.-]+/(?:home|Users|u[0-9]+)(?:/[^\s"',)]+)+`),
	// C:\Users\... Windows paths
	regexp.MustCompile(`[A-Za-z]:\\Users\\[^\s"',)]+`),
	// $HOME or ~/ patterns (keep placeholder but mark them)
	regexp.MustCompile(`(?:\$HOME|~)(?:/[^\s"',)]*)?`),
}

// replacePairs are static string replacements for common user-specific values.
// These are detected at initialization time from the running environment.
var replacePairs []struct{ old, new string }

func init() {
	home := os.Getenv("HOME")
	if home == "" {
		home = "/home/user"
	}
	userName := filepath.Base(home)

	replacePairs = []struct{ old, new string }{
		{home, "{MAESTRO_USER_HOME}"},
		{strings.ReplaceAll(home, "/", "\\"), "{MAESTRO_USER_HOME}"},
		{userName, "{MAESTRO_USER}"},
	}
}

// ProcessedDir returns the directory where sanitized copies of source files are stored.
func ProcessedDir() string {
	return filepath.Join(config.DataDir(), "processed")
}

// SanitizeFile copies a source file to the processed directory, sanitizing
// hardcoded absolute paths and environment-specific values. Returns the path
// to the sanitized copy, or the original path if no sanitization was needed.
func SanitizeFile(sourcePath, sourceID string) (string, error) {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		// If we can't read it, just return the original path
		return sourcePath, nil
	}

	content := string(data)
	original := content

	// Replace static pairs (HOME, username)
	for _, pair := range replacePairs {
		content = strings.ReplaceAll(content, pair.old, pair.new)
	}

	// Replace regex patterns with placeholders
	for _, re := range hardcodedPathPatterns {
		content = re.ReplaceAllString(content, "{MAESTRO_SANITIZED_PATH}")
	}

	// No changes needed — return original
	if content == original {
		return sourcePath, nil
	}

	// Write sanitized copy
	processedDir := ProcessedDir()
	if err := os.MkdirAll(processedDir, 0755); err != nil {
		return "", fmt.Errorf("create processed dir: %w", err)
	}

	// Preserve the relative path structure to avoid name collisions
	relName := fmt.Sprintf("%s-%s", sourceID, filepath.Base(sourcePath))
	destPath := filepath.Join(processedDir, relName)

	if err := os.WriteFile(destPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("write sanitized file %s: %w", destPath, err)
	}

	return destPath, nil
}
