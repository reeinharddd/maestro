package compress

import (
	"testing"

	"github.com/reeinharrrd/opencode-kit/internal/db"
)

func TestCompress_KeptSignals(t *testing.T) {
	c := New(10)
	out := c.Compress([]Observation{
		{Source: "cli", Step: 1, Message: "starting"},
		{Source: "db", Step: 2, Message: "Warning: backup failed"},
		{Source: "route", Step: 3, Message: "Best model selected", Important: true},
	})

	if out == "" {
		t.Fatal("expected compressed output")
	}
	if got := len(splitLines(out)); got != 2 {
		t.Fatalf("got %d lines, want 2", got)
	}
}

func TestCompress_Truncates(t *testing.T) {
	c := New(2)
	out := c.Compress([]Observation{
		{Source: "a", Step: 1, Message: "warn one"},
		{Source: "a", Step: 2, Message: "warn two"},
		{Source: "a", Step: 3, Message: "warn three"},
	})

	lines := splitLines(out)
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(lines))
	}
	if lines[1] == "" || lines[1][0:3] != "..." {
		t.Fatalf("expected truncation notice, got %q", lines[1])
	}
}

func TestPruneOutput_KeptSignals(t *testing.T) {
	out := PruneOutput("ok\nWARN: something\ncomplete\nnoise")
	lines := splitLines(out)
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(lines))
	}
}

func TestCompress_PersistsFragmentWhenDBPresent(t *testing.T) {
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	c := NewWithDB(d, 4)
	out := c.Compress([]Observation{{Source: "cli", Step: 1, Message: "warn one"}})
	if out == "" {
		t.Fatal("expected output")
	}
	frags, err := d.ListConfigFragments(10)
	if err != nil {
		t.Fatalf("list config fragments: %v", err)
	}
	if len(frags) == 0 {
		t.Fatal("expected persisted config fragment")
	}
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}
