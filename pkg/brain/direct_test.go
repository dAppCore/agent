// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestDirect returns a DirectSubsystem wired to the given test server.
func newTestDirect(srv *httptest.Server) *DirectSubsystem {
	return &DirectSubsystem{apiURL: srv.URL, apiKey: "test-key"}
}

// jsonHandler returns an http.Handler that responds with the given JSON payload.
func jsonHandler(payload any) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	})
}

// errorHandler returns an http.Handler that responds with the given status and body.
func errorHandler(status int, body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(body))
	})
}

// --- DirectSubsystem construction ---

func TestNewDirect_Good_Defaults(t *testing.T) {
	t.Setenv("CORE_BRAIN_URL", "")
	t.Setenv("CORE_BRAIN_KEY", "")

	sub := NewDirect()
	assert.Equal(t, "https://api.lthn.sh", sub.apiURL)
	assert.NotEmpty(t, sub.apiURL)
}

func TestNewDirect_Good_CustomEnv(t *testing.T) {
	t.Setenv("CORE_BRAIN_URL", "https://custom.api.test")
	t.Setenv("CORE_BRAIN_KEY", "test-key-123")

	sub := NewDirect()
	assert.Equal(t, "https://custom.api.test", sub.apiURL)
	assert.Equal(t, "test-key-123", sub.apiKey)
}

func TestNewDirect_Good_KeyFromFile(t *testing.T) {
	t.Setenv("CORE_BRAIN_URL", "")
	t.Setenv("CORE_BRAIN_KEY", "")

	tmpHome := t.TempDir()
	t.Setenv("CORE_HOME", tmpHome)
	keyDir := core.JoinPath(tmpHome, ".claude")
	require.True(t, fs.EnsureDir(keyDir).OK)
	require.True(t, fs.Write(core.JoinPath(keyDir, "brain.key"), "  file-key-456  \n").OK)

	sub := NewDirect()
	assert.Equal(t, "file-key-456", sub.apiKey)
}

func TestDirectSubsystem_Good_Name(t *testing.T) {
	sub := &DirectSubsystem{}
	assert.Equal(t, "brain", sub.Name())
}

func TestDirectSubsystem_Good_Shutdown(t *testing.T) {
	sub := &DirectSubsystem{}
	assert.NoError(t, sub.Shutdown(context.Background()))
}

// --- apiCall ---

func TestApiCall_Bad_NoAPIKey(t *testing.T) {
	sub := &DirectSubsystem{apiURL: "http://localhost", apiKey: ""}
	_, err := sub.apiCall(context.Background(), "GET", "/test", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no API key")
}

func TestApiCall_Good_GET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/v1/test", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
	}))
	defer srv.Close()

	result, err := newTestDirect(srv).apiCall(context.Background(), "GET", "/v1/test", nil)
	require.NoError(t, err)
	assert.Equal(t, "ok", result["status"])
}

func TestApiCall_Good_POSTWithBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "hello", body["content"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": "mem-123"})
	}))
	defer srv.Close()

	result, err := newTestDirect(srv).apiCall(context.Background(), "POST", "/v1/brain/remember", map[string]any{"content": "hello"})
	require.NoError(t, err)
	assert.Equal(t, "mem-123", result["id"])
}

func TestApiCall_Bad_ServerError(t *testing.T) {
	srv := httptest.NewServer(errorHandler(http.StatusInternalServerError, `{"error":"internal"}`))
	defer srv.Close()

	_, err := newTestDirect(srv).apiCall(context.Background(), "GET", "/v1/test", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API call failed")
}

func TestApiCall_Bad_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	_, err := newTestDirect(srv).apiCall(context.Background(), "GET", "/v1/test", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse response")
}

func TestApiCall_Bad_ConnectionRefused(t *testing.T) {
	sub := &DirectSubsystem{apiURL: "http://127.0.0.1:1", apiKey: "test-key"}
	_, err := sub.apiCall(context.Background(), "GET", "/v1/test", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API call failed")
}

func TestApiCall_Bad_BadRequest(t *testing.T) {
	srv := httptest.NewServer(errorHandler(http.StatusBadRequest, `{"error":"bad input"}`))
	defer srv.Close()

	_, err := newTestDirect(srv).apiCall(context.Background(), "GET", "/v1/test", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API call failed")
}

// --- remember ---

func TestRemember_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/brain/remember", r.URL.Path)

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "test content", body["content"])
		assert.Equal(t, "observation", body["type"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": "mem-abc"})
	}))
	defer srv.Close()

	_, out, err := newTestDirect(srv).remember(context.Background(), nil, RememberInput{
		Content: "test content",
		Type:    "observation",
		Tags:    []string{"test"},
		Project: "core",
	})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Equal(t, "mem-abc", out.MemoryID)
	assert.False(t, out.Timestamp.IsZero())
}

func TestRemember_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(errorHandler(http.StatusInternalServerError, `{"error":"db down"}`))
	defer srv.Close()

	_, out, err := newTestDirect(srv).remember(context.Background(), nil, RememberInput{
		Content: "test",
		Type:    "fact",
	})
	require.Error(t, err)
	assert.False(t, out.Success)
}

// --- recall ---

func TestRecall_Good_WithMemories(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/brain/recall", r.URL.Path)

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "architecture", body["query"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"memories": []any{
				map[string]any{
					"id":         "mem-1",
					"content":    "Use Qdrant for vector search",
					"type":       "decision",
					"project":    "agent",
					"agent_id":   "virgil",
					"score":      0.95,
					"source":     "manual",
					"created_at": "2026-03-03T12:00:00Z",
				},
				map[string]any{
					"id":         "mem-2",
					"content":    "DuckDB for embedded use",
					"type":       "architecture",
					"project":    "agent",
					"agent_id":   "cladius",
					"score":      0.88,
					"created_at": "2026-03-04T10:00:00Z",
				},
			},
		})
	}))
	defer srv.Close()

	_, out, err := newTestDirect(srv).recall(context.Background(), nil, RecallInput{
		Query: "architecture",
		TopK:  5,
	})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Equal(t, 2, out.Count)
	require.Len(t, out.Memories, 2)

	assert.Equal(t, "mem-1", out.Memories[0].ID)
	assert.Equal(t, "Use Qdrant for vector search", out.Memories[0].Content)
	assert.Equal(t, "decision", out.Memories[0].Type)
	assert.Equal(t, "virgil", out.Memories[0].AgentID)
	assert.Equal(t, 0.95, out.Memories[0].Confidence)
	assert.Contains(t, out.Memories[0].Tags, "source:manual")

	assert.Equal(t, "mem-2", out.Memories[1].ID)
}

func TestRecall_Good_DefaultTopK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, float64(10), body["top_k"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"memories": []any{}})
	}))
	defer srv.Close()

	_, out, err := newTestDirect(srv).recall(context.Background(), nil, RecallInput{
		Query: "test",
		TopK:  0,
	})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Equal(t, 0, out.Count)
}

func TestRecall_Good_WithFilters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "cladius", body["agent_id"])
		assert.Equal(t, "eaas", body["project"])
		assert.Equal(t, "decision", body["type"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"memories": []any{}})
	}))
	defer srv.Close()

	_, _, err := newTestDirect(srv).recall(context.Background(), nil, RecallInput{
		Query: "scoring",
		TopK:  5,
		Filter: RecallFilter{
			AgentID: "cladius",
			Project: "eaas",
			Type:    "decision",
		},
	})
	require.NoError(t, err)
}

func TestRecall_Good_EmptyMemories(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(map[string]any{"memories": []any{}}))
	defer srv.Close()

	_, out, err := newTestDirect(srv).recall(context.Background(), nil, RecallInput{Query: "nothing"})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Equal(t, 0, out.Count)
	assert.Empty(t, out.Memories)
}

func TestRecall_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(errorHandler(http.StatusServiceUnavailable, `{"error":"qdrant down"}`))
	defer srv.Close()

	_, out, err := newTestDirect(srv).recall(context.Background(), nil, RecallInput{Query: "test"})
	require.Error(t, err)
	assert.False(t, out.Success)
}

// --- forget ---

func TestForget_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/v1/brain/forget/mem-123", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"deleted": true})
	}))
	defer srv.Close()

	_, out, err := newTestDirect(srv).forget(context.Background(), nil, ForgetInput{
		ID:     "mem-123",
		Reason: "outdated",
	})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Equal(t, "mem-123", out.Forgotten)
	assert.False(t, out.Timestamp.IsZero())
}

func TestForget_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(errorHandler(http.StatusNotFound, `{"error":"not found"}`))
	defer srv.Close()

	_, out, err := newTestDirect(srv).forget(context.Background(), nil, ForgetInput{ID: "nonexistent"})
	require.Error(t, err)
	assert.False(t, out.Success)
}
