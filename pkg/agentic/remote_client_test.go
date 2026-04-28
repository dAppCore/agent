// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	core "dappco.re/go"
)

// --- mcpInitialize ---

func TestMcpInitialize_RemoteClient_Initialize_Good(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		core.AssertEqual(t, "POST", r.Method)
		core.AssertEqual(t, "application/json", r.Header.Get("Content-Type"))
		core.AssertEqual(t, "Bearer test-token", r.Header.Get("Authorization"))

		if callCount == 1 {
			// Initialize request
			var body map[string]any
			bodyStr := core.ReadAll(r.Body)
			core.JSONUnmarshalString(bodyStr.Value.(string), &body)
			core.AssertEqual(t, "initialize", body["method"])

			w.Header().Set("Mcp-Session-Id", "session-abc")
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprintf(w, "data: {\"result\":{}}\n\n")
		} else {
			// Initialized notification
			w.WriteHeader(200)
		}
	}))
	t.Cleanup(srv.Close)

	sessionID, err := mcpInitialize(context.Background(), srv.URL, "test-token")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "session-abc", sessionID)
	core.AssertEqual(t, 2, callCount, "should make init + notification requests")
}

func TestRemoteclient_McpInitialize_Bad_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	_, err := mcpInitialize(context.Background(), srv.URL, "")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "HTTP 500")
}

func TestMcpInitialize_RemoteClient_Initialize_Bad(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: {\"result\":{}}\n\n")
	}))
	t.Cleanup(srv.Close)

	_, err := mcpInitialize(context.Background(), srv.URL, "")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "missing session id")
}

func TestRemoteclient_McpInitialize_Bad_Unreachable(t *testing.T) {
	_, err := mcpInitialize(context.Background(), "http://127.0.0.1:1", "")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "request failed")
}

// --- mcpCall ---

func TestMcpCall_RemoteClient_Call_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "Bearer mytoken", r.Header.Get("Authorization"))
		core.AssertEqual(t, "sess-123", r.Header.Get("Mcp-Session-Id"))

		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "event: message\ndata: {\"result\":{\"content\":[{\"text\":\"hello\"}]}}\n\n")
	}))
	t.Cleanup(srv.Close)

	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call"}`)
	result, err := mcpCall(context.Background(), srv.URL, "mytoken", "sess-123", body)
	core.RequireNoError(t, err)
	core.AssertContains(t, string(result), "hello")
}

func TestMcpCall_RemoteClient_Call_Bad(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	_, err := mcpCall(context.Background(), srv.URL, "", "", nil)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "HTTP 500")
}

func TestRemoteclient_McpCall_Bad_NoSSEData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "event: ping\n\n") // No data: line
	}))
	t.Cleanup(srv.Close)

	_, err := mcpCall(context.Background(), srv.URL, "", "", nil)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "no data")
}

// --- setHeaders ---

func TestRemoteclient_SetHeaders_Good_All(t *testing.T) {
	req, _ := http.NewRequest("POST", "http://example.com", nil)
	mcpHeaders(req, "my-token", "my-session")

	core.AssertEqual(t, "application/json", req.Header.Get("Content-Type"))
	core.AssertEqual(t, "application/json, text/event-stream", req.Header.Get("Accept"))
	core.AssertEqual(t, "Bearer my-token", req.Header.Get("Authorization"))
	core.AssertEqual(t, "my-session", req.Header.Get("Mcp-Session-Id"))
}

func TestRemoteclient_SetHeaders_Good_NoToken(t *testing.T) {
	req, _ := http.NewRequest("POST", "http://example.com", nil)
	mcpHeaders(req, "", "")

	core.AssertEmpty(t, req.Header.Get("Authorization"))
	core.AssertEmpty(t, req.Header.Get("Mcp-Session-Id"))
}

// --- setHeaders Bad ---

func TestRemoteclient_SetHeaders_Bad(t *testing.T) {
	// Both token and session empty — only Content-Type and Accept are set
	req, _ := http.NewRequest("POST", "http://example.com", nil)
	mcpHeaders(req, "", "")

	core.AssertEqual(t, "application/json", req.Header.Get("Content-Type"))
	core.AssertEqual(t, "application/json, text/event-stream", req.Header.Get("Accept"))
	core.AssertEmpty(t, req.Header.Get("Authorization"), "no auth header when token is empty")
	core.AssertEmpty(t, req.Header.Get("Mcp-Session-Id"), "no session header when session is empty")
}

// --- readSSEData ---

func TestRemoteclient_ReadSSEData_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "event: message\ndata: {\"key\":\"value\"}\n\n")
	}))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL)
	core.RequireNoError(t, err)
	defer resp.Body.Close()

	data, err := readSSEData(resp)
	core.RequireNoError(t, err)
	core.AssertEqual(t, `{"key":"value"}`, string(data))
}

func TestRemoteclient_ReadSSEData_Bad_NoData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "event: ping\n\n")
	}))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL)
	core.RequireNoError(t, err)
	defer resp.Body.Close()

	_, err = readSSEData(resp)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "no data")
}

// --- drainSSE ---

func TestRemoteclient_DrainSSE_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "data: line1\ndata: line2\n\n")
	}))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL)
	core.RequireNoError(t, err)
	defer resp.Body.Close()

	// Should not panic
	drainSSE(resp)
}

// --- McpInitialize Ugly ---

func TestMcpInitialize_RemoteClient_Initialize_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Mcp-Session-Id", "sess-ugly")
		w.Header().Set("Content-Type", "text/event-stream")
		// Return non-JSON in SSE data line
		fmt.Fprintf(w, "data: this is not json at all\n\n")
	}))
	t.Cleanup(srv.Close)

	// mcpInitialize drains the SSE body but doesn't parse it — should succeed
	sessionID, err := mcpInitialize(context.Background(), srv.URL, "tok")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "sess-ugly", sessionID)
}

// --- McpCall Ugly ---

func TestMcpCall_RemoteClient_Call_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Write nothing — empty body
	}))
	t.Cleanup(srv.Close)

	_, err := mcpCall(context.Background(), srv.URL, "", "", nil)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "no data")
}

// --- ReadSSEData Ugly ---

func TestRemoteclient_ReadSSEData_Ugly_OnlyEventLines(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// Only event: lines, no data: lines
		fmt.Fprintf(w, "event: message\nevent: done\n\n")
	}))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL)
	core.RequireNoError(t, err)
	defer resp.Body.Close()

	_, err = readSSEData(resp)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "no data")
}

// --- SetHeaders Ugly ---

func TestRemoteclient_SetHeaders_Ugly_VeryLongToken(t *testing.T) {
	req, _ := http.NewRequest("POST", "http://example.com", nil)
	longToken := strings.Repeat("a", 10000)
	mcpHeaders(req, longToken, "sess-123")

	core.AssertEqual(t, "Bearer "+longToken, req.Header.Get("Authorization"))
	core.AssertEqual(t, "sess-123", req.Header.Get("Mcp-Session-Id"))
	core.AssertEqual(t, "application/json", req.Header.Get("Content-Type"))
}

// --- DrainSSE Bad/Ugly ---

func TestRemoteclient_DrainSSE_Bad_EmptyBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write nothing — empty body
	}))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL)
	core.RequireNoError(t, err)
	defer resp.Body.Close()

	// Should not panic on empty body
	core.AssertNotPanics(t, func() { drainSSE(resp) })
}

func TestRemoteclient_DrainSSE_Ugly_VeryLargeResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write many SSE lines
		for i := 0; i < 1000; i++ {
			fmt.Fprintf(w, "data: line-%d padding-text-to-make-it-bigger-and-test-scanner-handling\n", i)
		}
		fmt.Fprintf(w, "\n")
	}))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL)
	core.RequireNoError(t, err)
	defer resp.Body.Close()

	// Should drain all lines without panic
	core.AssertNotPanics(t, func() { drainSSE(resp) })
}

// --- RemoteClient ---

func TestRemoteClient_NewRemoteClient_Good(t *testing.T) {
	t.Setenv("AGENT_TOKEN_CHARON", "token-123")

	client := NewRemoteClient("charon")

	core.AssertEqual(t, "charon", client.Host)
	core.AssertEqual(t, "10.69.69.165:9101", client.Address)
	core.AssertEqual(t, "token-123", client.Token)
	core.AssertEqual(t, "http://10.69.69.165:9101/mcp", client.URL)
}

func TestTrimmedInput_NewRemoteClient_Ugly(t *testing.T) {
	t.Setenv("AGENT_TOKEN_CHARON", "token-123")

	client := NewRemoteClient("  charon  ")

	core.AssertEqual(t, "charon", client.Host)
	core.AssertEqual(t, "10.69.69.165:9101", client.Address)
	core.AssertEqual(t, "token-123", client.Token)
	core.AssertEqual(t, "http://10.69.69.165:9101/mcp", client.URL)
}

func TestToolCallBody_RemoteClient_ToolCallBody_Good(t *testing.T) {
	client := NewRemoteClient("local")

	body := client.ToolCallBody(7, "agentic_status", map[string]any{
		"workspace": "core/go-io/task-5",
	})

	var payload map[string]any
	result := core.JSONUnmarshal(body, &payload)
	core.RequireTrue(t, result.OK)
	core.AssertEqual(t, "2.0", payload["jsonrpc"])
	core.AssertEqual(t, float64(7), payload["id"])
	core.AssertEqual(t, "tools/call", payload["method"])
	params, ok := payload["params"].(map[string]any)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "agentic_status", params["name"])
}

func TestRemoteClient_NewRemoteClient_Bad(t *testing.T) {
	client := NewRemoteClient("")

	core.AssertEqual(t, ":9101", client.Address)
	core.AssertEqual(t, "http://:9101/mcp", client.URL)
}

func TestToolCallBody_RemoteClient_ToolCallBody_Ugly(t *testing.T) {
	client := NewRemoteClient("my-host.local")

	body := client.ToolCallBody(0, "", nil)

	var payload map[string]any
	result := core.JSONUnmarshal(body, &payload)
	core.RequireTrue(t, result.OK)
	params, ok := payload["params"].(map[string]any)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "", params["name"])
	core.AssertNil(t, params["arguments"])
}
