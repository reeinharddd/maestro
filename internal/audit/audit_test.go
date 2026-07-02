package audit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/reeinharrrd/maestro/internal/db"
	"github.com/reeinharrrd/maestro/pkg/models"
)

func TestNew(t *testing.T) {
	s := New(nil, 0)
	if s.workers != 5 {
		t.Errorf("default workers = %d, want 5", s.workers)
	}
	if s.db != nil {
		t.Error("db should be nil")
	}
}

func TestNew_CustomWorkers(t *testing.T) {
	s := New(nil, 3)
	if s.workers != 3 {
		t.Errorf("workers = %d, want 3", s.workers)
	}
}

func TestTestModel_NoBaseURL(t *testing.T) {
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	s := New(d, 1)
	prov := models.Provider{ID: "test", BaseURL: ""}
	m := models.Model{ID: "test/m", DisplayName: "m"}

	result := s.testModel(context.Background(), prov, m)
	if result.Status != "error" {
		t.Errorf("status = %q, want error", result.Status)
	}
	if !strings.Contains(result.ErrorMessage, "base_url") {
		t.Errorf("error = %q, want contains base_url", result.ErrorMessage)
	}
}

func TestTestModel_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			w.WriteHeader(401)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"x","choices":[{"message":{"content":"ok"}}],"usage":{}}`))
	}))
	defer srv.Close()

	os.Setenv("TEST_KEY", "test-key-123")
	defer os.Unsetenv("TEST_KEY")

	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	s := New(d, 1)
	prov := models.Provider{ID: "test", BaseURL: srv.URL, KeyEnv: "TEST_KEY"}
	m := models.Model{ID: "test/model", DisplayName: "model", ContextWindow: 1000}

	result := s.testModel(context.Background(), prov, m)
	if result.Status == "error" {
		t.Fatalf("model should not error: %s", result.ErrorMessage)
	}
	if result.Status != "active" {
		t.Errorf("status = %q, want active", result.Status)
	}
	if !result.FunctionCalling {
		t.Error("FunctionCalling should be true for successful test")
	}
	if result.LatencyP50Ms <= 0 {
		t.Errorf("LatencyP50Ms = %f, want > 0", result.LatencyP50Ms)
	}
}

func TestTestModel_ReasoningModel(t *testing.T) {
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"x","choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()

	os.Setenv("TEST_KEY", "key")
	defer os.Unsetenv("TEST_KEY")

	d, _ := db.Open(":memory:")
	defer d.Close()

	s := New(d, 1)
	prov := models.Provider{ID: "test", BaseURL: srv.URL, KeyEnv: "TEST_KEY"}

	// o3 model should use max_completion_tokens
	m := models.Model{ID: "test/o3-mini", DisplayName: "o3-mini", ContextWindow: 200000}
	result := s.testModel(context.Background(), prov, m)
	if result.Status == "error" {
		t.Fatalf("o3 should not error: %s", result.ErrorMessage)
	}
	if _, ok := gotBody["max_completion_tokens"]; !ok {
		t.Error("reasoning model should use max_completion_tokens")
	}
	if _, ok := gotBody["max_tokens"]; ok {
		t.Error("reasoning model should NOT use max_tokens")
	}
}

func TestTestModel_DeepseekR1(t *testing.T) {
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"x","choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()

	os.Setenv("TEST_KEY", "key")
	defer os.Unsetenv("TEST_KEY")

	d, _ := db.Open(":memory:")
	defer d.Close()

	s := New(d, 1)
	prov := models.Provider{ID: "test", BaseURL: srv.URL, KeyEnv: "TEST_KEY"}
	m := models.Model{ID: "test/deepseek-r1", DisplayName: "deepseek-r1", ContextWindow: 100000}

	result := s.testModel(context.Background(), prov, m)
	if result.Status == "error" {
		t.Fatalf("deepseek-r1 should not error: %s", result.ErrorMessage)
	}
	if _, ok := gotBody["max_completion_tokens"]; !ok {
		t.Error("deepseek-r1 should use max_completion_tokens")
	}
}

func TestTestModel_RateLimit(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(429)
		w.Write([]byte(`{"error":"rate limited"}`))
	}))
	defer srv.Close()

	os.Setenv("TEST_KEY", "key")
	defer os.Unsetenv("TEST_KEY")

	d, _ := db.Open(":memory:")
	defer d.Close()

	s := New(d, 1)
	prov := models.Provider{ID: "test", BaseURL: srv.URL, KeyEnv: "TEST_KEY"}
	m := models.Model{ID: "test/model", DisplayName: "model", ContextWindow: 1000}

	result := s.testModel(context.Background(), prov, m)
	if result.Status != "error" {
		t.Errorf("status = %q, want error", result.Status)
	}
	if !strings.Contains(result.ErrorMessage, "rate_limited") {
		t.Errorf("error = %q, want rate_limited", result.ErrorMessage)
	}
	if result.Tier != "unknown" {
		t.Errorf("tier = %q, want unknown for errored model", result.Tier)
	}
}

func TestTestModel_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`{"error":"internal"}`))
	}))
	defer srv.Close()

	os.Setenv("TEST_KEY", "key")
	defer os.Unsetenv("TEST_KEY")

	d, _ := db.Open(":memory:")
	defer d.Close()

	s := New(d, 1)
	prov := models.Provider{ID: "test", BaseURL: srv.URL, KeyEnv: "TEST_KEY"}
	m := models.Model{ID: "test/model", DisplayName: "model", ContextWindow: 1000}

	result := s.testModel(context.Background(), prov, m)
	if result.Status != "error" {
		t.Errorf("status = %q, want error", result.Status)
	}
	if result.Tier != "unknown" {
		t.Errorf("tier = %q, want unknown", result.Tier)
	}
}

func TestTestModel_ClaudePricing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"x","choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()

	os.Setenv("TEST_KEY", "key")
	defer os.Unsetenv("TEST_KEY")

	d, _ := db.Open(":memory:")
	defer d.Close()

	s := New(d, 1)
	prov := models.Provider{ID: "anthropic", BaseURL: srv.URL, KeyEnv: "TEST_KEY"}
	m := models.Model{ID: "anthropic/claude-sonnet-4", DisplayName: "claude-sonnet-4", ContextWindow: 200000}

	result := s.testModel(context.Background(), prov, m)
	if result.Status == "error" {
		t.Fatalf("claude should not error: %s", result.ErrorMessage)
	}
	if result.Tier != "paid" {
		t.Errorf("claude tier = %q, want paid", result.Tier)
	}
}

func TestTestModel_ContextWindowDefaults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"x","choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()

	os.Setenv("TEST_KEY", "key")
	defer os.Unsetenv("TEST_KEY")

	tests := []struct {
		id   string
		want int
	}{
		{"test/codestral-latest", 256000},
		{"test/gemini-pro", 1048576},
		{"test/gpt-4.1-nano", 1048576},
		{"test/deepseek-v3", 1048576},
		{"test/other-model", 131072},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			d, _ := db.Open(":memory:")
			defer d.Close()

			s := New(d, 1)
			prov := models.Provider{ID: "test", BaseURL: srv.URL, KeyEnv: "TEST_KEY"}
			m := models.Model{ID: tt.id, DisplayName: tt.id, ContextWindow: 0}

			result := s.testModel(context.Background(), prov, m)
			if result.ContextWindow != tt.want {
				t.Errorf("ContextWindow = %d, want %d", result.ContextWindow, tt.want)
			}
		})
	}
}
