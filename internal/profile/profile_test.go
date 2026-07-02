package profile_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
	"testing"

	"github.com/reeinharrrd/maestro/internal/db"
	"github.com/reeinharrrd/maestro/internal/profile"
	"github.com/reeinharrrd/maestro/pkg/models"
	"github.com/stretchr/testify/assert"
)

type mockDB struct {
	providers []models.Provider
	models    []models.Model
	profile   *models.ModelProfile
	returnErr bool
}

func (m *mockDB) ListProviders() ([]models.Provider, error) {
	if m.returnErr {
		return nil, assert.AnError
	}
	return m.providers, nil
}

func (m *mockDB) ListModelsByProvider(providerID string) ([]models.Model, error) {
	if m.returnErr {
		return nil, assert.AnError
	}
	return m.models, nil
}

func (m *mockDB) UpsertModelProfile(p *models.ModelProfile) error {
	m.profile = p
	return nil
}

func (m *mockDB) UpsertProvider(p *models.Provider) error                     { return nil }
func (m *mockDB) GetProvider(id string) (*models.Provider, error)             { return nil, nil }
func (m *mockDB) DeleteProvider(id string) error                              { return nil }
func (m *mockDB) UpsertModel(mdl *models.Model) error                         { return nil }
func (m *mockDB) ListModels(opts ...db.ModelFilter) ([]models.Model, error)              { return nil, nil }
func (m *mockDB) GetModel(id string) (*models.Model, error)                   { return nil, nil }
func (m *mockDB) DeleteModel(id string) error                                 { return nil }
func (m *mockDB) UpsertCommand(c *models.Command) error                       { return nil }
func (m *mockDB) ListCommands() ([]models.Command, error)                     { return nil, nil }
func (m *mockDB) DeleteCommand(id string) error                               { return nil }
func (m *mockDB) UpsertMCP(mcp *models.MCPServer) error                       { return nil }
func (m *mockDB) ListMCPs() ([]models.MCPServer, error)                       { return nil, nil }
func (m *mockDB) DeleteMCP(id string) error                                   { return nil }
func (m *mockDB) UpsertSkill(s *models.Skill) error                           { return nil }
func (m *mockDB) ListSkills() ([]models.Skill, error)                         { return nil, nil }
func (m *mockDB) UpsertSourceItem(s *models.SourceItem) error                 { return nil }
func (m *mockDB) ListSourceItems() ([]models.SourceItem, error)               { return nil, nil }
func (m *mockDB) GetSourceItem(id string) (*models.SourceItem, error)         { return nil, nil }
func (m *mockDB) DeleteSourceItem(id string) error                            { return nil }
func (m *mockDB) UpsertLSPServer(l *models.LSPServer) error                   { return nil }
func (m *mockDB) ListLSPServers() ([]models.LSPServer, error)                 { return nil, nil }
func (m *mockDB) GetLSPServer(id string) (*models.LSPServer, error)           { return nil, nil }
func (m *mockDB) DeleteLSPServer(id string) error                             { return nil }
func (m *mockDB) UpsertConfigFragment(f *models.ConfigFragment) error         { return nil }
func (m *mockDB) ListConfigFragments(limit int) ([]models.ConfigFragment, error) { return nil, nil }
func (m *mockDB) GetConfigFragment(id string) (*models.ConfigFragment, error) { return nil, nil }
func (m *mockDB) ListRoutingRules() ([]models.RoutingRule, error)            { return nil, nil }

func (m *mockDB) ListModelProfiles() ([]models.ModelProfile, error)           { return nil, nil }
func (m *mockDB) GetModelProfile(modelID string) (*models.ModelProfile, error) { return nil, nil }
func (m *mockDB) UpsertSource(src *models.Source) error                       { return nil }
func (m *mockDB) GetSource(id string) (*models.Source, error)                 { return nil, nil }
func (m *mockDB) DeleteSource(id string) error                                { return nil }
func (m *mockDB) ListSources() ([]models.Source, error)                       { return nil, nil }
func (m *mockDB) UpsertAgent(a *models.Agent) error                           { return nil }
func (m *mockDB) ListAgents() ([]models.Agent, error)                         { return nil, nil }
func (m *mockDB) GetAgent(id string) (*models.Agent, error)                   { return nil, nil }
func (m *mockDB) DeleteAgent(id string) error                                 { return nil }
func (m *mockDB) GetSmallFastModels(ctx context.Context) ([]models.Model, error) { return nil, nil }
func (m *mockDB) DBPath() string                                             { return "" }
func (m *mockDB) SearchModels(query string) ([]models.Model, error)          { return nil, nil }
func (m *mockDB) GetStats() (map[string]int, error)                          { return nil, nil }
func (m *mockDB) UpdateSkillMeta(id string, updates map[string]any) error    { return nil }
func (m *mockDB) SearchSkills(query string) ([]models.Skill, error)          { return nil, nil }
func (m *mockDB) DeleteSkill(id string) error                                { return nil }
func (m *mockDB) UpsertRoutingRule(r *models.RoutingRule) error              { return nil }
func (m *mockDB) DeleteRoutingRule(key string) error                         { return nil }
func (m *mockDB) GetBudget() (*models.BudgetConfig, error)                   { return nil, nil }
func (m *mockDB) GetRoutingRule(key string) (*models.RoutingRule, error)     { return nil, nil }
func (m *mockDB) InsertRoutingEvent(e *models.RoutingEvent) error            { return nil }
func (m *mockDB) ListRoutingEvents(limit int) ([]models.RoutingEvent, error) { return nil, nil }
func (m *mockDB) UpsertBudget(b *models.BudgetConfig) error                  { return nil }
func (m *mockDB) SetPreference(key, value string) error                      { return nil }
func (m *mockDB) ListPreferences() (map[string]string, error)                { return nil, nil }
func (m *mockDB) GetPreference(key string) (string, error)                   { return "", nil }
func (m *mockDB) DeletePreference(key string) error                          { return nil }
func (m *mockDB) CleanupProviderPrefs() (int, error)                         { return 0, nil }
func (m *mockDB) CleanupInvalidPreferences() (int, error)                    { return 0, nil }
func (m *mockDB) InsertSyncLog(phase, status, details string, durationMs int64) error { return nil }
func (m *mockDB) ListSyncLogs(limit int) ([]models.SyncLog, error)           { return nil, nil }
func (m *mockDB) InsertExecLog(l *models.ExecLog) error                      { return nil }
func (m *mockDB) ListExecLogs(limit int) ([]models.ExecLog, error)           { return nil, nil }
func (m *mockDB) InsertSnapshot(hash, content string) error                  { return nil }
func (m *mockDB) ListSnapshots(limit int) ([]models.Snapshot, error)         { return nil, nil }
func (m *mockDB) GetSnapshot(id int64) (*models.Snapshot, error)             { return nil, nil }
func (m *mockDB) DeleteSnapshot(id int64) error                              { return nil }
func (m *mockDB) Query(query string, args ...any) (*sql.Rows, error)        { return nil, nil }
func (m *mockDB) Exec(query string, args ...any) (sql.Result, error)        { return nil, nil }
func (m *mockDB) UpsertProject(p *models.Project) error                     { return nil }
func (m *mockDB) ListProjects() ([]models.Project, error)                   { return nil, nil }
func (m *mockDB) GetProject(id string) (*models.Project, error)             { return nil, nil }
func (m *mockDB) DeleteProject(id string) error                             { return nil }
func (m *mockDB) UpsertDetectedStack(d *models.DetectedStack) error         { return nil }
func (m *mockDB) ListDetectedStacks(projectID string) ([]models.DetectedStack, error) { return nil, nil }
func (m *mockDB) DeleteDetectedStacks(projectID string) error               { return nil }
func (m *mockDB) UpsertProjectConfig(p *models.ProjectConfig) error         { return nil }
func (m *mockDB) ListProjectConfigs(projectID string) ([]models.ProjectConfig, error) { return nil, nil }
func (m *mockDB) GetProjectConfig(projectID, configType string) (*models.ProjectConfig, error) { return nil, nil }
func (m *mockDB) DeleteProjectConfigs(projectID string) error               { return nil }
func (m *mockDB) UpdateSourceItemStatus(id, status string) error            { return nil }
func (m *mockDB) UpdateSourceItemTarget(id, targetPath string) error        { return nil }
func (m *mockDB) ListSourceItemsBySource(sourceID string) ([]models.SourceItem, error) { return nil, nil }

func TestNew(t *testing.T) {
	t.Parallel()
	s := profile.New(&mockDB{})
	assert.NotNil(t, s)
}

func TestProfileAll_EmptyProviders(t *testing.T) {
	t.Parallel()
	m := &mockDB{providers: []models.Provider{}}
	s := profile.New(m)
	err := s.ProfileAll(context.Background(), false)
	assert.NoError(t, err)
}

func TestProfileAll_OnlyInactiveProviders(t *testing.T) {
	t.Parallel()
	m := &mockDB{
		providers: []models.Provider{
			{ID: "test", Name: "Test", Status: "inactive"},
		},
	}
	s := profile.New(m)
	err := s.ProfileAll(context.Background(), false)
	assert.NoError(t, err)
}

func TestProfileAll_DBError(t *testing.T) {
	t.Parallel()
	m := &mockDB{returnErr: true}
	s := profile.New(m)
	err := s.ProfileAll(context.Background(), false)
	assert.Error(t, err)
}

func TestProfileModel_NoAPIKey(t *testing.T) {
	t.Parallel()
	m := &mockDB{}
	s := profile.New(m)
	prov := models.Provider{ID: "test", KeyEnv: "NONEXISTENT_KEY_ENV"}
	mdl := models.Model{ID: "test/model", DisplayName: "test-model"}
	_, err := s.ProfileModel(context.Background(), prov, mdl)
	assert.ErrorContains(t, err, "no API key")
}

func TestTestStreaming_ServerError(t *testing.T) {
	t.Parallel()
	s := profile.New(&mockDB{})
	tps, ok := s.TestStreaming("http://127.0.0.1:1/nonexistent", "key", "model")
	assert.False(t, ok)
	assert.Equal(t, float64(0), tps)
}

func TestTestStreaming_NonStreamingResponse(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"id":"test"}`)
	}))
	defer srv.Close()

	s := profile.New(&mockDB{})
	tps, ok := s.TestStreaming(srv.URL, "fake-key", "test-model")
	assert.False(t, ok)
	assert.Equal(t, float64(0), tps)
}

func TestTestStreaming_WithValidStreamData(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"token%d\"}}]}\n", 0)
		w.(http.Flusher).Flush()
		time.Sleep(150 * time.Millisecond)
		for i := 1; i < 5; i++ {
			fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"token%d\"}}]}\n", i)
		}
		fmt.Fprint(w, "data: [DONE]\n")
	}))
	defer srv.Close()

	s := profile.New(&mockDB{})
	tps, ok := s.TestStreaming(srv.URL, "fake-key", "test-model")
	assert.True(t, ok)
	assert.Greater(t, tps, float64(0))
}

func TestTestStreaming_Non200(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	s := profile.New(&mockDB{})
	tps, ok := s.TestStreaming(srv.URL, "fake-key", "test-model")
	assert.False(t, ok)
	assert.Equal(t, float64(0), tps)
}

func TestTestStructuredOutput_InvalidEndpoint(t *testing.T) {
	t.Parallel()
	s := profile.New(&mockDB{})
	ok := s.TestStructuredOutput("http://127.0.0.1:1/nonexistent", "key", "model")
	assert.False(t, ok)
}

func TestTestStructuredOutput_Non200(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	s := profile.New(&mockDB{})
	ok := s.TestStructuredOutput(srv.URL, "key", "model")
	assert.False(t, ok)
}

func TestTestStructuredOutput_ValidJSONResponse(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": `{"hello":"world"}`}},
			},
		})
	}))
	defer srv.Close()

	s := profile.New(&mockDB{})
	ok := s.TestStructuredOutput(srv.URL, "key", "model")
	assert.True(t, ok)
}

func TestTestStructuredOutput_InvalidJSONResponse(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"choices":[{"message":{"content":"not-json"}}]}`)
	}))
	defer srv.Close()

	s := profile.New(&mockDB{})
	ok := s.TestStructuredOutput(srv.URL, "key", "model")
	assert.False(t, ok)
}

func TestTestStructuredOutput_EmptyChoices(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{},
		})
	}))
	defer srv.Close()

	s := profile.New(&mockDB{})
	ok := s.TestStructuredOutput(srv.URL, "key", "model")
	assert.False(t, ok)
}

func TestEstimateContext_ServerError(t *testing.T) {
	t.Parallel()
	s := profile.New(&mockDB{})
	ctx := s.EstimateContext("http://127.0.0.1:1/nonexistent", "key", "model")
	assert.Equal(t, 131072, ctx)
}

func TestEstimateContext_200Response(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"id":"test"}`)
	}))
	defer srv.Close()

	s := profile.New(&mockDB{})
	ctx := s.EstimateContext(srv.URL, "key", "model")
	assert.Equal(t, 131072, ctx)
}

func TestEstimateContext_Non200Response(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":"context length exceeded"}`)
	}))
	defer srv.Close()

	s := profile.New(&mockDB{})
	ctx := s.EstimateContext(srv.URL, "key", "model")
	assert.Equal(t, 8192, ctx)
}

func TestEstimateContext_CheckRequestBody(t *testing.T) {
	t.Parallel()
	var receivedBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		receivedBody = string(bodyBytes)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	}))
	defer srv.Close()

	s := profile.New(&mockDB{})
	ctx := s.EstimateContext(srv.URL, "key", "test-model")
	assert.Equal(t, 131072, ctx)
	assert.Contains(t, receivedBody, "test-model")
	assert.Contains(t, receivedBody, strings.Repeat("A", 5000))
}
