// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core "dappco.re/go"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	brainclient "dappco.re/go/mcp/pkg/mcp/brain/client"
)

// newTestDirect returns a DirectSubsystem wired to the given test server.
func newTestDirect(srv *httptest.Server) *DirectSubsystem {
	return &DirectSubsystem{
		apiURL: srv.URL,
		apiKey: "test-key",
		apiClient: brainclient.New(brainclient.Options{
			URL:         srv.URL,
			Key:         "test-key",
			HTTPClient:  srv.Client(),
			MaxAttempts: 1,
			BaseDelay:   time.Nanosecond,
		}),
	}
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
	core.AssertEqual(t, "https://api.lthn.sh", sub.apiURL)
	core.AssertNotEmpty(t, sub.apiURL)
}

func TestDirect_NewDirect_Good_CustomEnv(t *testing.T) {
	t.Setenv("CORE_BRAIN_URL", "https://custom.api.test")
	t.Setenv("CORE_BRAIN_KEY", "test-key-123")

	sub := NewDirect()
	core.AssertEqual(t, "https://custom.api.test", sub.apiURL)
	core.AssertEqual(t, "test-key-123", sub.apiKey)
}

func TestDirect_NewDirect_Good_KeyFromFile(t *testing.T) {
	t.Setenv("CORE_BRAIN_URL", "")
	t.Setenv("CORE_BRAIN_KEY", "")

	tmpHome := t.TempDir()
	t.Setenv("CORE_HOME", tmpHome)
	keyDir := core.JoinPath(tmpHome, ".claude")
	core.RequireTrue(t, fs.EnsureDir(keyDir).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(keyDir, "brain.key"), "  file-key-456  \n").OK)

	sub := NewDirect()
	core.AssertEqual(t, "file-key-456", sub.apiKey)
}

func TestDirect_NewDirect_Good_HomeFallback(t *testing.T) {
	t.Setenv("CORE_BRAIN_URL", "")
	t.Setenv("CORE_BRAIN_KEY", "")
	t.Setenv("CORE_HOME", "")
	t.Setenv("DIR_HOME", "")

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	keyDir := core.JoinPath(tmpHome, ".claude")
	core.RequireTrue(t, fs.EnsureDir(keyDir).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(keyDir, "brain.key"), "  home-key-789  \n").OK)

	sub := NewDirect()
	core.AssertEqual(t, "home-key-789", sub.apiKey)
}

func TestDirect_Subsystem_Good_Name(t *testing.T) {
	sub := &DirectSubsystem{}
	got := sub.Name()
	core.AssertEqual(t, "brain", got)
	core.AssertNotEmpty(t, got)
}

func TestDirect_Subsystem_Good_Shutdown(t *testing.T) {
	sub := &DirectSubsystem{}
	err := sub.Shutdown(context.Background())
	core.AssertNoError(t, err)
	core.AssertNil(t, err)
}

// --- apiCall ---

func TestDirect_ApiCall_Bad_NoAPIKey(t *testing.T) {
	sub := &DirectSubsystem{apiURL: "http://localhost", apiKey: ""}
	result := sub.apiCall(context.Background(), "GET", "/test", nil)
	core.AssertFalse(t, result.OK)
	err, _ := result.Value.(error)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "no API key")
}

func TestDirect_ApiCall_Good_GET(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "GET", r.Method)
		core.AssertEqual(t, "/v1/test", r.URL.Path)
		core.AssertEqual(t, "Bearer test-key", r.Header.Get("Authorization"))
		core.AssertEqual(t, "application/json", r.Header.Get("Accept"))

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(map[string]any{"status": "ok"})))
	}))
	defer srv.Close()

	result := newTestDirect(srv).apiCall(context.Background(), "GET", "/v1/test", nil)
	core.RequireTrue(t, result.OK)
	payload, _ := result.Value.(map[string]any)
	core.AssertEqual(t, "ok", payload["status"])
}

func TestDirect_ApiCall_Good_POSTWithBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "POST", r.Method)
		core.AssertEqual(t, "application/json", r.Header.Get("Content-Type"))

		var body map[string]any
		core.JSONUnmarshalString(core.ReadAll(r.Body).Value.(string), &body)
		core.AssertEqual(t, "hello", body["content"])

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(map[string]any{"id": "mem-123"})))
	}))
	defer srv.Close()

	result := newTestDirect(srv).apiCall(context.Background(), "POST", "/v1/brain/remember", map[string]any{"content": "hello"})
	core.RequireTrue(t, result.OK)
	payload, _ := result.Value.(map[string]any)
	core.AssertEqual(t, "mem-123", payload["id"])
}

func TestDirect_ApiCall_Bad_ServerError(t *testing.T) {
	srv := httptest.NewServer(errorHandler(http.StatusInternalServerError, `{"error":"internal"}`))
	defer srv.Close()

	result := newTestDirect(srv).apiCall(context.Background(), "GET", "/v1/test", nil)
	core.AssertFalse(t, result.OK)
	err, _ := result.Value.(error)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "upstream returned 500")
}

func TestDirect_ApiCall_Bad_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	result := newTestDirect(srv).apiCall(context.Background(), "GET", "/v1/test", nil)
	core.AssertFalse(t, result.OK)
	err, _ := result.Value.(error)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "parse response")
}

func TestDirect_ApiCall_Bad_ConnectionRefused(t *testing.T) {
	sub := &DirectSubsystem{
		apiURL: "http://127.0.0.1:1",
		apiKey: "test-key",
		apiClient: brainclient.New(brainclient.Options{
			URL:         "http://127.0.0.1:1",
			Key:         "test-key",
			MaxAttempts: 1,
			BaseDelay:   time.Nanosecond,
		}),
	}
	result := sub.apiCall(context.Background(), "GET", "/v1/test", nil)
	core.AssertFalse(t, result.OK)
	err, _ := result.Value.(error)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "request failed")
}

func TestDirect_ApiCall_Bad_BadRequest(t *testing.T) {
	srv := httptest.NewServer(errorHandler(http.StatusBadRequest, `{"error":"bad input"}`))
	defer srv.Close()

	result := newTestDirect(srv).apiCall(context.Background(), "GET", "/v1/test", nil)
	core.AssertFalse(t, result.OK)
	err, _ := result.Value.(error)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "upstream returned 400")
}

// --- remember ---

func TestDirect_Remember_Good_Case(t *testing.T) {
	t.Setenv("CORE_BRAIN_AGENT_ID", "codex")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "POST", r.Method)
		core.AssertEqual(t, "/v1/brain/remember", r.URL.Path)

		var body map[string]any
		core.JSONUnmarshalString(core.ReadAll(r.Body).Value.(string), &body)
		core.AssertEqual(t, "test content", body["content"])
		core.AssertEqual(t, "observation", body["type"])
		core.AssertEqual(t, "core", body["org"])
		core.AssertEqual(t, "codex", body["agent_id"])

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
		Org:     "core",
		Project: "core",
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertEqual(t, "mem-abc", out.MemoryID)
	core.AssertFalse(t, out.Timestamp.IsZero())
}

func TestDirect_Remember_Ugly_LegacyTopLevelID(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(map[string]any{"id": "mem-legacy"}))
	defer srv.Close()

	_, out, err := newTestDirect(srv).remember(context.Background(), nil, RememberInput{
		Content: "legacy payload",
		Type:    "observation",
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertEqual(t, "mem-legacy", out.MemoryID)
}

func TestDirect_Remember_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(errorHandler(http.StatusInternalServerError, `{"error":"db down"}`))
	defer srv.Close()

	_, out, err := newTestDirect(srv).remember(context.Background(), nil, RememberInput{
		Content: "test",
		Type:    "fact",
	})
	core.AssertError(t, err)
	core.AssertFalse(t, out.Success)
}

// --- recall ---

func TestDirect_Recall_Good_WithMemories(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "POST", r.Method)
		core.AssertEqual(t, "/v1/brain/recall", r.URL.Path)

		var body map[string]any
		core.JSONUnmarshalString(core.ReadAll(r.Body).Value.(string), &body)
		core.AssertEqual(t, "architecture", body["query"])
		core.AssertEqual(t, "core", body["org"])

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
		Filter: RecallFilter{
			Org: "core",
		},
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertEqual(t, 2, out.Count)
	core.AssertLen(t, out.Memories, 2)

	core.AssertEqual(t, "mem-1", out.Memories[0].ID)
	core.AssertEqual(t, "Use Qdrant for vector search", out.Memories[0].Content)
	core.AssertEqual(t, "decision", out.Memories[0].Type)
	core.AssertEqual(t, "virgil", out.Memories[0].AgentID)
	core.AssertEqual(t, 0.95, out.Memories[0].Confidence)
	core.AssertEqual(t, "manual", out.Memories[0].Source)
	core.AssertContains(t, out.Memories[0].Tags, "source:manual")

	core.AssertEqual(t, "mem-2", out.Memories[1].ID)
}

func TestDirect_Recall_Good_DefaultTopK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		core.JSONUnmarshalString(core.ReadAll(r.Body).Value.(string), &body)
		core.AssertEqual(t, float64(10), body["top_k"])

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
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertEqual(t, 0, out.Count)
}

func TestDirect_Recall_Good_WithFilters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		core.JSONUnmarshalString(core.ReadAll(r.Body).Value.(string), &body)
		core.AssertEqual(t, "cladius", body["agent_id"])
		core.AssertEqual(t, "core", body["org"])
		core.AssertEqual(t, "eaas", body["project"])
		core.AssertEqual(t, "decision", body["type"])

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
			Org:     "core",
			Project: "eaas",
			Type:    "decision",
		},
	})
	core.RequireNoError(t, err)
}

func TestDirect_Recall_Good_EmptyMemories(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(map[string]any{
		"data": map[string]any{"memories": []any{}},
	}))
	defer srv.Close()

	_, out, err := newTestDirect(srv).recall(context.Background(), nil, RecallInput{Query: "nothing"})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertEqual(t, 0, out.Count)
	core.AssertEmpty(t, out.Memories)
}

func TestDirect_Recall_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(errorHandler(http.StatusServiceUnavailable, `{"error":"qdrant down"}`))
	defer srv.Close()

	_, out, err := newTestDirect(srv).recall(context.Background(), nil, RecallInput{Query: "test"})
	core.AssertError(t, err)
	core.AssertFalse(t, out.Success)
}

// --- forget ---

func TestDirect_Forget_Good_Case(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "DELETE", r.Method)
		core.AssertEqual(t, "/v1/brain/forget/mem-123", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(map[string]any{"deleted": true})))
	}))
	defer srv.Close()

	_, out, err := newTestDirect(srv).forget(context.Background(), nil, ForgetInput{
		ID:     "mem-123",
		Reason: "outdated",
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertEqual(t, "mem-123", out.Forgotten)
	core.AssertFalse(t, out.Timestamp.IsZero())
}

func TestDirect_Forget_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(errorHandler(http.StatusNotFound, `{"error":"not found"}`))
	defer srv.Close()

	_, out, err := newTestDirect(srv).forget(context.Background(), nil, ForgetInput{ID: "nonexistent"})
	core.AssertError(t, err)
	core.AssertFalse(t, out.Success)
}

// --- list ---

func TestDirect_List_Good_WithMemories(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "GET", r.Method)
		core.AssertEqual(t, "/v1/brain/list", r.URL.Path)
		core.AssertEqual(t, "core", r.URL.Query().Get("org"))
		core.AssertEqual(t, "agent", r.URL.Query().Get("project"))
		core.AssertEqual(t, "decision", r.URL.Query().Get("type"))
		core.AssertEqual(t, "codex", r.URL.Query().Get("agent_id"))
		core.AssertEqual(t, "2", r.URL.Query().Get("limit"))

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(map[string]any{
			"data": map[string]any{
				"memories": []any{
					map[string]any{
						"id":               "mem-list-1",
						"content":          "Use the review queue for completed workspaces",
						"type":             "decision",
						"project":          "agent",
						"agent_id":         "codex",
						"confidence":       0.73,
						"supersedes_count": 2,
						"deleted_at":       "2026-03-31T12:30:00Z",
						"tags":             []any{"queue", "review"},
						"updated_at":       "2026-03-30T10:00:00Z",
						"created_at":       "2026-03-30T09:00:00Z",
						"expires_at":       "2026-04-01T00:00:00Z",
						"source":           "manual",
						"supersedes_id":    "mem-old",
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
		Org:     "core",
		Project: "agent",
		Type:    "decision",
		AgentID: "codex",
		Limit:   2,
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertEqual(t, 2, out.Count)
	core.AssertLen(t, out.Memories, 2)

	core.AssertEqual(t, "mem-list-1", out.Memories[0].ID)
	core.AssertEqual(t, 0.73, out.Memories[0].Confidence)
	core.AssertEqual(t, 2, out.Memories[0].SupersedesCount)
	core.AssertEqual(t, "mem-old", out.Memories[0].SupersedesID)
	core.AssertEqual(t, "manual", out.Memories[0].Source)
	core.AssertEqual(t, "2026-03-30T10:00:00Z", out.Memories[0].UpdatedAt)
	core.AssertEqual(t, "2026-04-01T00:00:00Z", out.Memories[0].ExpiresAt)
	core.AssertEqual(t, "2026-03-31T12:30:00Z", out.Memories[0].DeletedAt)
	core.AssertContains(t, out.Memories[0].Tags, "queue")
	core.AssertContains(t, out.Memories[0].Tags, "source:manual")

	core.AssertEqual(t, "mem-list-2", out.Memories[1].ID)
	core.AssertEqual(t, 0.91, out.Memories[1].Confidence)
}

func TestDirect_List_Good_EmptyMemories(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(map[string]any{
		"data": map[string]any{"memories": []any{}},
	}))
	defer srv.Close()

	_, out, err := newTestDirect(srv).list(context.Background(), nil, ListInput{})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertEqual(t, 0, out.Count)
	core.AssertEmpty(t, out.Memories)
}

func TestDirect_List_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(errorHandler(http.StatusServiceUnavailable, `{"error":"list unavailable"}`))
	defer srv.Close()

	_, out, err := newTestDirect(srv).list(context.Background(), nil, ListInput{Project: "agent"})
	core.AssertError(t, err)
	core.AssertFalse(t, out.Success)
}

func TestDirect_NewDirect_Good(t *testing.T) {
	t.Setenv("CORE_BRAIN_URL", "")
	t.Setenv("CORE_BRAIN_KEY", "")

	sub := NewDirect()
	core.AssertEqual(t, "https://api.lthn.sh", sub.apiURL)
	core.AssertNotEmpty(t, sub.apiURL)
}

func TestDirect_NewDirect_Bad(t *testing.T) {
	t.Setenv("CORE_BRAIN_URL", "")
	t.Setenv("CORE_BRAIN_KEY", "")
	t.Setenv("CORE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	sub := NewDirect()
	core.AssertEqual(t, "https://api.lthn.sh", sub.apiURL)
	core.AssertEmpty(t, sub.apiKey)
}

func TestDirect_NewDirect_Ugly(t *testing.T) {
	t.Setenv("CORE_BRAIN_URL", "")
	t.Setenv("CORE_BRAIN_KEY", "")
	t.Setenv("CORE_HOME", "")
	t.Setenv("DIR_HOME", "")

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	keyDir := core.JoinPath(tmpHome, ".claude")
	core.RequireTrue(t, fs.EnsureDir(keyDir).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(keyDir, "brain.key"), "  home-key-789  \n").OK)

	sub := NewDirect()
	core.AssertEqual(t, "home-key-789", sub.apiKey)
	core.AssertNotNil(t, sub.apiClient)
}

func TestDirect_DirectSubsystem_Name_Good(t *testing.T) {
	t.Setenv("CORE_BRAIN_URL", "https://custom.api.test")
	t.Setenv("CORE_BRAIN_KEY", "test-key")
	got := NewDirect().Name()
	core.AssertEqual(t, "brain", got)
	core.AssertNotEmpty(t, got)
}

func TestDirect_DirectSubsystem_Name_Bad(t *testing.T) {
	got := (&DirectSubsystem{}).Name()
	core.AssertEqual(t, "brain", got)
	core.AssertContains(t, got, "brain")
}

func TestDirect_DirectSubsystem_Name_Ugly(t *testing.T) {
	var sub *DirectSubsystem
	got := sub.Name()
	core.AssertEqual(t, "brain", got)
	core.AssertNotContains(t, got, "/")
}

func TestDirect_DirectSubsystem_RegisterTools_Good(t *testing.T) {
	names := listedToolNames(t, NewDirect().RegisterTools)
	core.AssertContains(t, names, "brain_remember")
	core.AssertContains(t, names, "agent_send")
}

func TestDirect_DirectSubsystem_RegisterTools_Bad(t *testing.T) {
	names := listedToolNames(t, (&DirectSubsystem{}).RegisterTools)
	core.AssertContains(t, names, "brain_list")
	core.AssertContains(t, names, "agent_inbox")
}

func TestDirect_DirectSubsystem_RegisterTools_Ugly(t *testing.T) {
	names := listedToolNames(t, func(svc *coremcp.Service) {
		sub := &DirectSubsystem{}
		sub.RegisterTools(svc)
		sub.RegisterTools(svc)
	})
	core.AssertContains(t, names, "brain_recall")
	core.AssertContains(t, names, "agent_conversation")
}

func TestDirect_DirectSubsystem_Shutdown_Good(t *testing.T) {
	err := NewDirect().Shutdown(context.Background())
	core.AssertNoError(t, err)
	core.AssertNil(t, err)
}

func TestDirect_DirectSubsystem_Shutdown_Bad(t *testing.T) {
	err := (&DirectSubsystem{}).Shutdown(context.Background())
	core.AssertNoError(t, err)
	core.AssertNil(t, err)
}

func TestDirect_DirectSubsystem_Shutdown_Ugly(t *testing.T) {
	var sub *DirectSubsystem
	core.AssertNotPanics(t, func() {
		core.AssertNoError(t, sub.Shutdown(context.Background()))
	})
}
