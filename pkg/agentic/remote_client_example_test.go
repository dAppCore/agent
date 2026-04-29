// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"syscall"

	core "dappco.re/go"
)

func ExampleDispatchInput_remote() {
	input := DispatchInput{Repo: "go-io", Task: "Fix tests", Agent: "codex"}
	core.Println(input.Agent)
	// Output: codex
}

func ExampleNewRemoteClient() {
	if err := syscall.Setenv("AGENT_TOKEN_CHARON", "token-123"); err != nil {
		panic(err)
	}
	defer func() {
		if err := syscall.Unsetenv("AGENT_TOKEN_CHARON"); err != nil {
			panic(err)
		}
	}()
	client := NewRemoteClient("charon")
	core.Println(client.Host)
	// Output: charon
}

func ExampleRemoteClient_Initialize() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Mcp-Session-Id", "example-session")
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"result\":{}}\n\n"))
	}))
	defer srv.Close()
	client := RemoteClient{URL: srv.URL, Token: "token"}
	sessionID, _ := client.Initialize(context.Background())
	core.Println(sessionID)
	// Output: example-session
}

func ExampleRemoteClient_Call() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: message\ndata: {\"result\":{\"content\":[{\"text\":\"hello\"}]}}\n\n"))
	}))
	defer srv.Close()
	client := RemoteClient{URL: srv.URL, Token: "token"}
	body := []byte(`{"jsonrpc":"2.0","id":1}`)
	result, _ := client.Call(context.Background(), "sess-1", body)
	core.Println(core.Contains(string(result), "hello"))
	// Output: true
}

func ExampleRemoteClient_ToolCallBody() {
	body := NewRemoteClient("local").ToolCallBody(1, "agentic_status", map[string]any{"workspace": "core/go-io/task-5"})
	core.Println(core.Contains(string(body), "agentic_status"))
	// Output: true
}
