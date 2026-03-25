// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- mcpInitialize ---

func TestRemoteClient_McpInitialize_Good(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		if callCount == 1 {
			// Initialize request
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			assert.Equal(t, "initialize", body["method"])

			w.Header().Set("Mcp-Session-Id", "session-abc")
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprintf(w, "data: {\"result\":{}}\n\n")
		} else {
			// Initialized notification
			w.WriteHeader(200)
		}
	}))
	t.Cleanup(srv.Close)

	sessionID, err := mcpInitialize(context.Background(), srv.Client(), srv.URL, "test-token")
	require.NoError(t, err)
	assert.Equal(t, "session-abc", sessionID)
	assert.Equal(t, 2, callCount, "should make init + notification requests")
}

func TestRemoteClient_McpInitialize_Bad_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	_, err := mcpInitialize(context.Background(), srv.Client(), srv.URL, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 500")
}

func TestRemoteClient_McpInitialize_Bad_Unreachable(t *testing.T) {
	_, err := mcpInitialize(context.Background(), http.DefaultClient, "http://127.0.0.1:1", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "request failed")
}

// --- mcpCall ---

func TestRemoteClient_McpCall_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer mytoken", r.Header.Get("Authorization"))
		assert.Equal(t, "sess-123", r.Header.Get("Mcp-Session-Id"))

		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "event: message\ndata: {\"result\":{\"content\":[{\"text\":\"hello\"}]}}\n\n")
	}))
	t.Cleanup(srv.Close)

	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call"}`)
	result, err := mcpCall(context.Background(), srv.Client(), srv.URL, "mytoken", "sess-123", body)
	require.NoError(t, err)
	assert.Contains(t, string(result), "hello")
}

func TestRemoteClient_McpCall_Bad_HTTP500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	_, err := mcpCall(context.Background(), srv.Client(), srv.URL, "", "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 500")
}

func TestRemoteClient_McpCall_Bad_NoSSEData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "event: ping\n\n") // No data: line
	}))
	t.Cleanup(srv.Close)

	_, err := mcpCall(context.Background(), srv.Client(), srv.URL, "", "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no data")
}

// --- setHeaders ---

func TestRemoteClient_SetHeaders_Good_All(t *testing.T) {
	req, _ := http.NewRequest("POST", "http://example.com", nil)
	setHeaders(req, "my-token", "my-session")

	assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
	assert.Equal(t, "application/json, text/event-stream", req.Header.Get("Accept"))
	assert.Equal(t, "Bearer my-token", req.Header.Get("Authorization"))
	assert.Equal(t, "my-session", req.Header.Get("Mcp-Session-Id"))
}

func TestRemoteClient_SetHeaders_Good_NoToken(t *testing.T) {
	req, _ := http.NewRequest("POST", "http://example.com", nil)
	setHeaders(req, "", "")

	assert.Empty(t, req.Header.Get("Authorization"))
	assert.Empty(t, req.Header.Get("Mcp-Session-Id"))
}

// --- setHeaders Bad ---

func TestRemoteClient_SetHeaders_Bad(t *testing.T) {
	// Both token and session empty — only Content-Type and Accept are set
	req, _ := http.NewRequest("POST", "http://example.com", nil)
	setHeaders(req, "", "")

	assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
	assert.Equal(t, "application/json, text/event-stream", req.Header.Get("Accept"))
	assert.Empty(t, req.Header.Get("Authorization"), "no auth header when token is empty")
	assert.Empty(t, req.Header.Get("Mcp-Session-Id"), "no session header when session is empty")
}

// --- readSSEData ---

func TestRemoteClient_ReadSSEData_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "event: message\ndata: {\"key\":\"value\"}\n\n")
	}))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	data, err := readSSEData(resp)
	require.NoError(t, err)
	assert.Equal(t, `{"key":"value"}`, string(data))
}

func TestRemoteClient_ReadSSEData_Bad_NoData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "event: ping\n\n")
	}))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	_, err = readSSEData(resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no data")
}

// --- drainSSE ---

func TestRemoteClient_DrainSSE_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "data: line1\ndata: line2\n\n")
	}))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should not panic
	drainSSE(resp)
}

// --- McpInitialize Ugly ---

func TestRemoteClient_McpInitialize_Ugly_NonJSONSSE(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Mcp-Session-Id", "sess-ugly")
		w.Header().Set("Content-Type", "text/event-stream")
		// Return non-JSON in SSE data line
		fmt.Fprintf(w, "data: this is not json at all\n\n")
	}))
	t.Cleanup(srv.Close)

	// mcpInitialize drains the SSE body but doesn't parse it — should succeed
	sessionID, err := mcpInitialize(context.Background(), srv.Client(), srv.URL, "tok")
	require.NoError(t, err)
	assert.Equal(t, "sess-ugly", sessionID)
}

// --- McpCall Ugly ---

func TestRemoteClient_McpCall_Ugly_EmptyResponseBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Write nothing — empty body
	}))
	t.Cleanup(srv.Close)

	_, err := mcpCall(context.Background(), srv.Client(), srv.URL, "", "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no data")
}

// --- ReadSSEData Ugly ---

func TestRemoteClient_ReadSSEData_Ugly_OnlyEventLines(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Only event: lines, no data: lines
		fmt.Fprintf(w, "event: message\nevent: done\n\n")
	}))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	_, err = readSSEData(resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no data")
}

// --- SetHeaders Ugly ---

func TestRemoteClient_SetHeaders_Ugly_VeryLongToken(t *testing.T) {
	req, _ := http.NewRequest("POST", "http://example.com", nil)
	longToken := strings.Repeat("a", 10000)
	setHeaders(req, longToken, "sess-123")

	assert.Equal(t, "Bearer "+longToken, req.Header.Get("Authorization"))
	assert.Equal(t, "sess-123", req.Header.Get("Mcp-Session-Id"))
	assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
}

// --- DrainSSE Bad/Ugly ---

func TestRemoteClient_DrainSSE_Bad_EmptyBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write nothing — empty body
	}))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should not panic on empty body
	assert.NotPanics(t, func() { drainSSE(resp) })
}

func TestRemoteClient_DrainSSE_Ugly_VeryLargeResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write many SSE lines
		for i := 0; i < 1000; i++ {
			fmt.Fprintf(w, "data: line-%d padding-text-to-make-it-bigger-and-test-scanner-handling\n", i)
		}
		fmt.Fprintf(w, "\n")
	}))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should drain all lines without panic
	assert.NotPanics(t, func() { drainSSE(resp) })
}
