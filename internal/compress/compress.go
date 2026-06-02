package compress

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/reeinharrrd/opencode-kit/internal/db"
	"github.com/reeinharrrd/opencode-kit/pkg/models"
)

type Observation struct {
	Source    string
	Step      int
	Message   string
	Important bool
}

type Compressor struct {
	MaxLines int
	db       db.DBInterface
}

func New(maxLines int) *Compressor {
	if maxLines <= 0 {
		maxLines = 12
	}
	return &Compressor{MaxLines: maxLines}
}

func NewWithDB(database db.DBInterface, maxLines int) *Compressor {
	c := New(maxLines)
	c.db = database
	return c
}

func (c *Compressor) Compress(observations []Observation) string {
	if len(observations) == 0 {
		return ""
	}

	lines := make([]string, 0, len(observations))
	for _, obs := range observations {
		msg := strings.TrimSpace(obs.Message)
		if msg == "" {
			continue
		}
		if obs.Important || isSignal(msg) {
			lines = append(lines, fmt.Sprintf("[%s:%d] %s", obs.Source, obs.Step, msg))
		}
	}

	if len(lines) == 0 {
		return ""
	}
	if len(lines) <= c.MaxLines {
		return c.persist("session", strings.Join(lines, "\n"), "compress")
	}

	keep := c.MaxLines - 1
	if keep < 1 {
		keep = 1
	}
	head := lines[:keep]
	return c.persist("session", strings.Join(append(head, fmt.Sprintf("... %d more compressed item(s)", len(lines)-keep)), "\n"), "compress")
}

func (c *Compressor) persist(kind, content, source string) string {
	if c.db != nil {
		h := sha256.Sum256([]byte(content))
		id := hex.EncodeToString(h[:])[:24]
		now := time.Now().UTC().Format(time.RFC3339)
		_ = c.db.UpsertConfigFragment(&models.ConfigFragment{ID: id, ConfigType: kind, Content: content, Source: source, Hash: hex.EncodeToString(h[:]), CreatedAt: now, UpdatedAt: now})
	}
	return content
}

func PruneOutput(output string) string {
	var out []string
	s := bufio.NewScanner(strings.NewReader(output))
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		if keepLine(line) {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

func keepLine(line string) bool {
	lower := strings.ToLower(line)
	if strings.Contains(lower, "error") || strings.Contains(lower, "fail") || strings.Contains(lower, "warn") {
		return true
	}
	if strings.HasPrefix(lower, "[1/") || strings.HasPrefix(lower, "[2/") || strings.HasPrefix(lower, "[3/") || strings.HasPrefix(lower, "[4/") || strings.HasPrefix(lower, "[5/") || strings.HasPrefix(lower, "[6/") || strings.HasPrefix(lower, "[7/") || strings.HasPrefix(lower, "[8/") {
		return true
	}
	if strings.Contains(lower, "complete") || strings.Contains(lower, "summary") {
		return true
	}
	return false
}

func isSignal(msg string) bool {
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "error") || strings.Contains(lower, "warn") || strings.Contains(lower, "fail") {
		return true
	}
	if strings.Contains(lower, "missing") || strings.Contains(lower, "not found") || strings.Contains(lower, "fallback") {
		return true
	}
	return strings.Contains(lower, "complete") || strings.Contains(lower, "summary")
}
