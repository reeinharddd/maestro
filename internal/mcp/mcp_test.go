package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/reeinharrrd/maestro/internal/classifier"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockClassifier struct {
	result classifier.ClassificationResult
	err    error
}

func (m *mockClassifier) Classify(ctx context.Context, task classifier.Task) (classifier.ClassificationResult, error) {
	return m.result, m.err
}

func TestNewServer(t *testing.T) {
	t.Parallel()
	s := NewServer(&mockClassifier{}, nil)
	assert.NotNil(t, s)
	assert.NotNil(t, s.handlers)
	assert.Contains(t, s.handlers, "classify_task")
}

func TestHandleTools(t *testing.T) {
	t.Parallel()
	s := NewServer(&mockClassifier{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/tools", nil)
	w := httptest.NewRecorder()
	s.handleTools(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var tools []map[string]any
	err := json.NewDecoder(w.Body).Decode(&tools)
	require.NoError(t, err)
	require.Len(t, tools, 1)
	assert.Equal(t, "classify_task", tools[0]["id"])

	params, ok := tools[0]["parameters"].(map[string]any)
	require.True(t, ok)
	taskParam, ok := params["task"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "object", taskParam["type"])
}

func TestHandleTools_MethodNotAllowed(t *testing.T) {
	t.Parallel()
	s := NewServer(&mockClassifier{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/tools", nil)
	w := httptest.NewRecorder()
	s.handleTools(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestHandleExecute_UnknownTool(t *testing.T) {
	t.Parallel()
	s := NewServer(&mockClassifier{}, nil)

	body, _ := json.Marshal(ToolRequest{ToolID: "unknown_tool"})
	req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.handleExecute(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleExecute_InvalidJSON(t *testing.T) {
	t.Parallel()
	s := NewServer(&mockClassifier{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader([]byte(`not json`)))
	w := httptest.NewRecorder()
	s.handleExecute(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleExecute_ClassifyTask_Success(t *testing.T) {
	t.Parallel()
	mc := &mockClassifier{
		result: classifier.ClassificationResult{
			TaskID:     "task_1",
			Intent:     "coding",
			Confidence: 0.95,
			Entities:   map[string]string{"lang": "go"},
		},
	}
	s := NewServer(mc, nil)

	body, _ := json.Marshal(ToolRequest{
		ToolID:    "classify_task",
		Arguments: json.RawMessage(`{"input":"write a function","session_id":"sess_1"}`),
	})
	req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.handleExecute(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp ToolResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Empty(t, resp.Error)

	var result classifier.ClassificationResult
	b, _ := json.Marshal(resp.Output)
	err = json.Unmarshal(b, &result)
	require.NoError(t, err)
	assert.Equal(t, "coding", result.Intent)
	assert.Equal(t, 0.95, result.Confidence)
}

func TestHandleExecute_ClassifyTask_InvalidArgs(t *testing.T) {
	t.Parallel()
	s := NewServer(&mockClassifier{}, nil)

	body, _ := json.Marshal(ToolRequest{
		ToolID:    "classify_task",
		Arguments: json.RawMessage(`invalid`),
	})
	req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.handleExecute(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "decode request")
}

func TestHandleExecute_ClassifierError(t *testing.T) {
	t.Parallel()
	mc := &mockClassifier{err: assert.AnError}
	s := NewServer(mc, nil)

	body, _ := json.Marshal(ToolRequest{
		ToolID:    "classify_task",
		Arguments: json.RawMessage(`{"input":"test","session_id":"sess_1"}`),
	})
	req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.handleExecute(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp ToolResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Contains(t, resp.Error, "classify")
}

func TestHandleExecute_MethodNotAllowed(t *testing.T) {
	t.Parallel()
	s := NewServer(&mockClassifier{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/execute", nil)
	w := httptest.NewRecorder()
	s.handleExecute(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}
