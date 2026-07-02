package sync

import "strings"

// normalizeAgentID transforms machine-specific agent IDs (derived from
// absolute file paths or GitHub URLs) into portable names.
//
// Examples:
//
//	file-home-user-tools-superpowers-root-AGENTS -> superpowers-agents
//	github-com-owner-repo-root-AGENTS           -> repo-agents
func normalizeAgentID(id string) string {
	if !strings.HasPrefix(id, "file-") && !strings.HasPrefix(id, "github-com-") {
		return id
	}

	parts := strings.Split(id, "-")

	// Walk backwards to find the meaningful project/repo name.
	// Pattern: file-<path-segments>-root-AGENTS or github-com-<owner>-<repo>-root-AGENTS
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == "root" && i > 0 {
			return parts[i-1] + "-agents"
		}
	}

	// Fallback: use the segment just before "AGENTS"
	if len(parts) >= 2 && parts[len(parts)-1] == "AGENTS" {
		return parts[len(parts)-2] + "-agents"
	}

	return id
}
