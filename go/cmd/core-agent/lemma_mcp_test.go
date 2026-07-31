// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"testing"

	"dappco.re/go/agent/pkg/chathistory"
	"dappco.re/go/agent/pkg/lemma"
)

// fakeLemmaServer returns an httptest server that echoes user turns
// back as the assistant. Sufficient for round-trip + continuation
// tests without needing lthn-mlx running.
func fakeLemmaServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.Error(w, "wrong path", http.StatusNotFound)
			return
		}
		var req struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var lastUser string
		for _, v := range slices.Backward(req.Messages) {
			if v.Role == "user" {
				lastUser = v.Content
				break
			}
		}
		resp := map[string]any{
			"id":      "fake",
			"model":   "test",
			"choices": []map[string]any{{"index": 0, "message": map[string]string{"role": "assistant", "content": "echo: " + lastUser}, "finish_reason": "stop"}},
			"usage":   map[string]int{"prompt_tokens": 5, "completion_tokens": 3, "total_tokens": 8},
		}
		w.Header().Set("content-type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

// TestLemmaSubsystem_Name — the subsystem id is "lemma" so the
// tool registers under the "lemma" group.
func TestLemmaSubsystem_Name(t *testing.T) {
	sub := newLemmaSubsystem()
	if got := sub.Name(); got != "lemma" {
		t.Errorf("Name() = %q, want %q", got, "lemma")
	}
}

// TestRegisterLemmaSubsystem — the core.WithService factory returns
// a *lemmaSubsystem wrapped in a successful Result.
func TestRegisterLemmaSubsystem(t *testing.T) {
	result := registerLemmaSubsystem(nil)
	if !result.OK {
		t.Fatalf("registerLemmaSubsystem: OK=false, value=%v", result.Value)
	}
	if _, ok := result.Value.(*lemmaSubsystem); !ok {
		t.Errorf("unexpected value type: %T", result.Value)
	}
}

// TestLemmaSubsystem_HandleSend_RequiresAgentID — empty agent_id
// is rejected before any I/O happens.
func TestLemmaSubsystem_HandleSend_RequiresAgentID(t *testing.T) {
	sub := &lemmaSubsystem{historyDir: t.TempDir()}
	_, _, err := sub.handleSend(context.Background(), LemmaSendInput{Message: "hi"})
	if err == nil {
		t.Fatal("expected error for empty agent_id, got nil")
	}
}

// TestLemmaSubsystem_HandleSend_RequiresMessage — empty message
// is rejected before any I/O happens.
func TestLemmaSubsystem_HandleSend_RequiresMessage(t *testing.T) {
	sub := &lemmaSubsystem{historyDir: t.TempDir()}
	_, _, err := sub.handleSend(context.Background(), LemmaSendInput{AgentID: "cladius"})
	if err == nil {
		t.Fatal("expected error for empty message, got nil")
	}
}

// TestLemmaSubsystem_HandleSend_FreshConversation — calling
// lemma_send with no conversation_id starts a fresh thread and
// returns the new conversation_id.
func TestLemmaSubsystem_HandleSend_FreshConversation(t *testing.T) {
	srv := fakeLemmaServer(t)
	defer srv.Close()

	sub := &lemmaSubsystem{
		cfg: lemma.Config{
			BaseURL: srv.URL + "/v1",
			ModelID: "test",
		},
		historyDir: t.TempDir(),
	}

	_, out, err := sub.handleSend(context.Background(), LemmaSendInput{
		AgentID: "cladius",
		Message: "hello",
		Title:   "smoke",
	})
	if err != nil {
		t.Fatalf("handleSend: %v", err)
	}
	if out.Reply != "echo: hello" {
		t.Errorf("Reply = %q, want %q", out.Reply, "echo: hello")
	}
	if out.ConversationID == "" {
		t.Error("ConversationID empty — caller can't continue the thread")
	}
}

// TestLemmaSubsystem_HandleSend_ContinuesConversation — passing the
// conversation_id from a previous call appends to the same thread
// (verified by LoadTurns showing both user turns in order).
func TestLemmaSubsystem_HandleSend_ContinuesConversation(t *testing.T) {
	srv := fakeLemmaServer(t)
	defer srv.Close()

	tmp := t.TempDir()
	sub := &lemmaSubsystem{
		cfg: lemma.Config{
			BaseURL: srv.URL + "/v1",
			ModelID: "test",
		},
		historyDir: tmp,
	}

	_, first, err := sub.handleSend(context.Background(), LemmaSendInput{
		AgentID: "cladius",
		Message: "first message",
	})
	if err != nil {
		t.Fatalf("first send: %v", err)
	}

	_, second, err := sub.handleSend(context.Background(), LemmaSendInput{
		AgentID:        "cladius",
		Message:        "second message",
		ConversationID: first.ConversationID,
	})
	if err != nil {
		t.Fatalf("second send: %v", err)
	}
	if second.ConversationID != first.ConversationID {
		t.Errorf("ConversationID changed across continuation: %q -> %q",
			first.ConversationID, second.ConversationID)
	}

	// LoadTurns must show 4 turns (user, assistant, user, assistant) in order.
	histPath := filepath.Join(tmp, "cladius", "chats.duckdb")
	hist, err := chathistory.Open("cladius", histPath)
	if err != nil {
		t.Fatalf("re-open history: %v", err)
	}
	defer hist.Close()
	turns, err := hist.LoadTurns(first.ConversationID)
	if err != nil {
		t.Fatalf("LoadTurns: %v", err)
	}
	if len(turns) != 4 {
		t.Fatalf("expected 4 turns after two sends, got %d", len(turns))
	}
	want := []struct{ role, content string }{
		{"user", "first message"},
		{"assistant", "echo: first message"},
		{"user", "second message"},
		{"assistant", "echo: second message"},
	}
	for i, w := range want {
		if turns[i].Role != w.role || turns[i].Content != w.content {
			t.Errorf("turn[%d]: got (%s, %s) want (%s, %s)",
				i, turns[i].Role, turns[i].Content, w.role, w.content)
		}
	}
}
