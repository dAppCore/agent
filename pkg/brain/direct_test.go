// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"
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
		w.Write([]byte(core.JSONMarshalString(payload)))
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

func TestDirect_NewDirect_Good_Defaults(t *testing.T) {
	t.Setenv("CORE_BRAIN_URL", "")
	t.Setenv("CORE_BRAIN_KEY", "")

	sub := NewDirect()
	assert.Equal(t, "https://api.lthn.sh", sub.apiURL)
	assert.NotEmpty(t, sub.apiURL)
}

func TestDirect_NewDirect_Good_CustomEnv(t *testing.T) {
	t.Setenv("CORE_BRAIN_URL", "https://custom.api.test")
	t.Setenv("CORE_BRAIN_KEY", "test-key-123")

	sub := NewDirect()
	assert.Equal(t, "https://custom.api.test", sub.apiURL)
	assert.Equal(t, "test-key-123", sub.apiKey)
}

func TestDirect_NewDirect_Good_KeyFromFile(t *testing.T) {
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

func TestDirect_NewDirect_Good_HomeFallback(t *testing.T) {
	t.Setenv("CORE_BRAIN_URL", "")
	t.Setenv("CORE_BRAIN_KEY", "")
	t.Setenv("CORE_HOME", "")
	t.Setenv("DIR_HOME", "")

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	keyDir := core.JoinPath(tmpHome, ".claude")
	require.True(t, fs.EnsureDir(keyDir).OK)
	require.True(t, fs.Write(core.JoinPath(keyDir, "brain.key"), "  home-key-789  \n").OK)

	sub := NewDirect()
	assert.Equal(t, "home-key-789", sub.apiKey)
}

func TestDirect_Subsystem_Good_Name(t *testing.T) {
	sub := &DirectSubsystem{}
	assert.Equal(t, "brain", sub.Name())
}

func TestDirect_Subsystem_Good_Shutdown(t *testing.T) {
	sub := &DirectSubsystem{}
	assert.NoError(t, sub.Shutdown(context.Background()))
}

// --- apiCall ---

func TestDirect_ApiCall_Bad_NoAPIKey(t *testing.T) {
	sub := &DirectSubsystem{apiURL: "http://localhost", apiKey: ""}
	result := sub.apiCall(context.Background(), "GET", "/test", nil)
	require.False(t, result.OK)
	err, _ := result.Value.(error)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no API key")
}

func TestDirect_ApiCall_Good_GET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/v1/test", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(map[string]any{"status": "ok"})))
	}))
	defer srv.Close()

	result := newTestDirect(srv).apiCall(context.Background(), "GET", "/v1/test", nil)
	require.True(t, result.OK)
	payload, _ := result.Value.(map[string]any)
	assert.Equal(t, "ok", payload["status"])
}

func TestDirect_ApiCall_Good_POSTWithBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var body map[string]any
		core.JSONUnmarshalString(core.ReadAll(r.Body).Value.(string), &body)
		assert.Equal(t, "hello", body["content"])

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(map[string]any{"id": "mem-123"})))
	}))
	defer srv.Close()

	result := newTestDirect(srv).apiCall(context.Background(), "POST", "/v1/brain/remember", map[string]any{"content": "hello"})
	require.True(t, result.OK)
	payload, _ := result.Value.(map[string]any)
	assert.Equal(t, "mem-123", payload["id"])
}

func TestDirect_ApiCall_Bad_ServerError(t *testing.T) {
	srv := httptest.NewServer(errorHandler(http.StatusInternalServerError, `{"error":"internal"}`))
	defer srv.Close()

	result := newTestDirect(srv).apiCall(context.Background(), "GET", "/v1/test", nil)
	require.False(t, result.OK)
	err, _ := result.Value.(error)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API call failed")
}

func TestDirect_ApiCall_Bad_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	result := newTestDirect(srv).apiCall(context.Background(), "GET", "/v1/test", nil)
	require.False(t, result.OK)
	err, _ := result.Value.(error)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse response")
}

func TestDirect_ApiCall_Bad_ConnectionRefused(t *testing.T) {
	sub := &DirectSubsystem{apiURL: "http://127.0.0.1:1", apiKey: "test-key"}
	result := sub.apiCall(context.Background(), "GET", "/v1/test", nil)
	require.False(t, result.OK)
	err, _ := result.Value.(error)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API call failed")
}

func TestDirect_ApiCall_Bad_BadRequest(t *testing.T) {
	srv := httptest.NewServer(errorHandler(http.StatusBadRequest, `{"error":"bad input"}`))
	defer srv.Close()

	result := newTestDirect(srv).apiCall(context.Background(), "GET", "/v1/test", nil)
	require.False(t, result.OK)
	err, _ := result.Value.(error)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API call failed")
}

// --- remember ---

func TestDirect_Remember_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/brain/remember", r.URL.Path)

		var body map[string]any
		core.JSONUnmarshalString(core.ReadAll(r.Body).Value.(string), &body)
		assert.Equal(t, "test content", body["content"])
		assert.Equal(t, "observation", body["type"])

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(map[string]any{
			"data": map[string]any{"id": "mem-abc"},
		})))
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

func TestDirect_Remember_Ugly_LegacyTopLevelID(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(map[string]any{"id": "mem-legacy"}))
	defer srv.Close()

	_, out, err := newTestDirect(srv).remember(context.Background(), nil, RememberInput{
		Content: "legacy payload",
		Type:    "observation",
	})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Equal(t, "mem-legacy", out.MemoryID)
}

func TestDirect_Remember_Bad_APIError(t *testing.T) {
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

func TestDirect_Recall_Good_WithMemories(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/brain/recall", r.URL.Path)

		var body map[string]any
		core.JSONUnmarshalString(core.ReadAll(r.Body).Value.(string), &body)
		assert.Equal(t, "architecture", body["query"])

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(map[string]any{
			"data": map[string]any{
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
			},
		})))
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

func TestDirect_Recall_Good_DefaultTopK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		core.JSONUnmarshalString(core.ReadAll(r.Body).Value.(string), &body)
		assert.Equal(t, float64(10), body["top_k"])

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(map[string]any{
			"data": map[string]any{"memories": []any{}},
		})))
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

func TestDirect_Recall_Good_WithFilters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		core.JSONUnmarshalString(core.ReadAll(r.Body).Value.(string), &body)
		assert.Equal(t, "cladius", body["agent_id"])
		assert.Equal(t, "eaas", body["project"])
		assert.Equal(t, "decision", body["type"])

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(map[string]any{
			"data": map[string]any{"memories": []any{}},
		})))
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

func TestDirect_Recall_Good_EmptyMemories(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(map[string]any{
		"data": map[string]any{"memories": []any{}},
	}))
	defer srv.Close()

	_, out, err := newTestDirect(srv).recall(context.Background(), nil, RecallInput{Query: "nothing"})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Equal(t, 0, out.Count)
	assert.Empty(t, out.Memories)
}

func TestDirect_Recall_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(errorHandler(http.StatusServiceUnavailable, `{"error":"qdrant down"}`))
	defer srv.Close()

	_, out, err := newTestDirect(srv).recall(context.Background(), nil, RecallInput{Query: "test"})
	require.Error(t, err)
	assert.False(t, out.Success)
}

// --- forget ---

func TestDirect_Forget_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/v1/brain/forget/mem-123", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(map[string]any{"deleted": true})))
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

func TestDirect_Forget_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(errorHandler(http.StatusNotFound, `{"error":"not found"}`))
	defer srv.Close()

	_, out, err := newTestDirect(srv).forget(context.Background(), nil, ForgetInput{ID: "nonexistent"})
	require.Error(t, err)
	assert.False(t, out.Success)
}

// --- list ---

func TestDirect_List_Good_WithMemories(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/v1/brain/list", r.URL.Path)
		assert.Equal(t, "agent", r.URL.Query().Get("project"))
		assert.Equal(t, "decision", r.URL.Query().Get("type"))
		assert.Equal(t, "codex", r.URL.Query().Get("agent_id"))
		assert.Equal(t, "2", r.URL.Query().Get("limit"))

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(map[string]any{
			"data": map[string]any{
				"memories": []any{
					map[string]any{
						"id":            "mem-list-1",
						"content":       "Use the review queue for completed workspaces",
						"type":          "decision",
						"project":       "agent",
						"agent_id":      "codex",
						"confidence":    0.73,
						"tags":          []any{"queue", "review"},
						"updated_at":    "2026-03-30T10:00:00Z",
						"created_at":    "2026-03-30T09:00:00Z",
						"expires_at":    "2026-04-01T00:00:00Z",
						"source":        "manual",
						"supersedes_id": "mem-old",
					},
					map[string]any{
						"id":         "mem-list-2",
						"content":    "AgentCompleted should key on workspace",
						"type":       "architecture",
						"project":    "agent",
						"agent_id":   "cladius",
						"score":      0.91,
						"created_at": "2026-03-31T08:00:00Z",
					},
				},
			},
		})))
	}))
	defer srv.Close()

	_, out, err := newTestDirect(srv).list(context.Background(), nil, ListInput{
		Project: "agent",
		Type:    "decision",
		AgentID: "codex",
		Limit:   2,
	})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Equal(t, 2, out.Count)
	require.Len(t, out.Memories, 2)

	assert.Equal(t, "mem-list-1", out.Memories[0].ID)
	assert.Equal(t, 0.73, out.Memories[0].Confidence)
	assert.Equal(t, "mem-old", out.Memories[0].SupersedesID)
	assert.Equal(t, "2026-03-30T10:00:00Z", out.Memories[0].UpdatedAt)
	assert.Equal(t, "2026-04-01T00:00:00Z", out.Memories[0].ExpiresAt)
	assert.Contains(t, out.Memories[0].Tags, "queue")
	assert.Contains(t, out.Memories[0].Tags, "source:manual")

	assert.Equal(t, "mem-list-2", out.Memories[1].ID)
	assert.Equal(t, 0.91, out.Memories[1].Confidence)
}

func TestDirect_List_Good_EmptyMemories(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(map[string]any{
		"data": map[string]any{"memories": []any{}},
	}))
	defer srv.Close()

	_, out, err := newTestDirect(srv).list(context.Background(), nil, ListInput{})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Equal(t, 0, out.Count)
	assert.Empty(t, out.Memories)
}

func TestDirect_List_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(errorHandler(http.StatusServiceUnavailable, `{"error":"list unavailable"}`))
	defer srv.Close()

	_, out, err := newTestDirect(srv).list(context.Background(), nil, ListInput{Project: "agent"})
	require.Error(t, err)
	assert.False(t, out.Success)
}
