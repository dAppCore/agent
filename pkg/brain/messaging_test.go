// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

// localDirect returns a DirectSubsystem that never hits the network.
// Suitable for tests that validate input before making API calls.
func localDirect() *DirectSubsystem {
	return &DirectSubsystem{apiURL: "http://localhost", apiKey: "test-key"}
}

// --- sendMessage ---

func TestMessaging_SendMessage_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "POST", r.Method)
		core.AssertEqual(t, "/v1/messages/send", r.URL.Path)

		var body map[string]any
		core.JSONUnmarshalString(core.ReadAll(r.Body).Value.(string), &body)
		core.AssertEqual(t, "charon", body["to"])
		core.AssertEqual(t, "deploy complete", body["content"])
		core.AssertEqual(t, "status update", body["subject"])

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(map[string]any{
			"data": map[string]any{"id": float64(42)},
		})))
	}))
	defer srv.Close()

	_, out, err := newTestDirect(srv).sendMessage(context.Background(), nil, SendInput{
		To:      "charon",
		Content: "deploy complete",
		Subject: "status update",
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertEqual(t, 42, out.ID)
	core.AssertEqual(t, "charon", out.To)
}

func TestMessaging_SendMessage_Bad_EmptyTo(t *testing.T) {
	_, _, err := localDirect().sendMessage(context.Background(), nil, SendInput{
		To:      "",
		Content: "hello",
	})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "to and content are required")
}

func TestMessaging_SendMessage_Bad_EmptyContent(t *testing.T) {
	_, _, err := localDirect().sendMessage(context.Background(), nil, SendInput{
		To:      "charon",
		Content: "",
	})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "to and content are required")
}

func TestMessaging_SendMessage_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(errorHandler(http.StatusInternalServerError, `{"error":"queue full"}`))
	defer srv.Close()

	_, out, err := newTestDirect(srv).sendMessage(context.Background(), nil, SendInput{
		To:      "charon",
		Content: "hello",
	})
	core.AssertError(t, err)
	core.AssertFalse(t, out.Success)
}

// --- inbox ---

func TestMessaging_Inbox_Good_WithMessages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "GET", r.Method)
		core.AssertContains(t, r.URL.Path, "/v1/messages/inbox")

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(map[string]any{
			"data": []any{
				map[string]any{
					"workspace_id": float64(3),
					"id":           float64(1),
					"from_agent":   "charon",
					"to_agent":     "cladius",
					"subject":      "status",
					"content":      "deploy done",
					"read":         true,
					"read_at":      "2026-03-10T12:05:00Z",
					"created_at":   "2026-03-10T12:00:00Z",
				},
				map[string]any{
					"id":         float64(2),
					"from":       "clotho",
					"to":         "cladius",
					"subject":    "review",
					"content":    "PR ready",
					"read":       false,
					"created_at": "2026-03-10T13:00:00Z",
				},
			},
		})))
	}))
	defer srv.Close()

	_, out, err := newTestDirect(srv).inbox(context.Background(), nil, InboxInput{Agent: "cladius"})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertLen(t, out.Messages, 2)
	core.AssertEqual(t, 1, out.Messages[0].ID)
	core.AssertEqual(t, 3, out.Messages[0].WorkspaceID)
	core.AssertEqual(t, "charon", out.Messages[0].From)
	core.AssertEqual(t, "charon", out.Messages[0].FromAgent)
	core.AssertEqual(t, "cladius", out.Messages[0].ToAgent)
	core.AssertEqual(t, "deploy done", out.Messages[0].Content)
	core.AssertTrue(t, out.Messages[0].Read)
	core.AssertEqual(t, "2026-03-10T12:05:00Z", out.Messages[0].ReadAt)
	core.AssertEqual(t, 2, out.Messages[1].ID)
	core.AssertFalse(t, out.Messages[1].Read)
}

func TestMessaging_Inbox_Good_EmptyInbox(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(map[string]any{"data": []any{}}))
	defer srv.Close()

	_, out, err := newTestDirect(srv).inbox(context.Background(), nil, InboxInput{Agent: "cladius"})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertEmpty(t, out.Messages)
}

func TestMessaging_Inbox_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(errorHandler(http.StatusInternalServerError, `{"error":"db down"}`))
	defer srv.Close()

	_, out, err := newTestDirect(srv).inbox(context.Background(), nil, InboxInput{})
	core.AssertError(t, err)
	core.AssertFalse(t, out.Success)
}

// --- conversation ---

func TestMessaging_Conversation_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "GET", r.Method)
		core.AssertContains(t, r.URL.Path, "/v1/messages/conversation/charon")

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(map[string]any{
			"data": []any{
				map[string]any{
					"id":         float64(10),
					"from":       "cladius",
					"to":         "charon",
					"content":    "how is the deploy?",
					"created_at": "2026-03-10T12:00:00Z",
				},
				map[string]any{
					"id":         float64(11),
					"from":       "charon",
					"to":         "cladius",
					"content":    "all green",
					"created_at": "2026-03-10T12:01:00Z",
				},
			},
		})))
	}))
	defer srv.Close()

	_, out, err := newTestDirect(srv).conversation(context.Background(), nil, ConversationInput{Agent: "charon"})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertLen(t, out.Messages, 2)
	core.AssertEqual(t, "how is the deploy?", out.Messages[0].Content)
	core.AssertEqual(t, "all green", out.Messages[1].Content)
}

func TestMessaging_Conversation_Bad_EmptyAgent(t *testing.T) {
	_, _, err := localDirect().conversation(context.Background(), nil, ConversationInput{Agent: ""})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "agent is required")
}

func TestMessaging_Conversation_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(errorHandler(http.StatusNotFound, `{"error":"agent not found"}`))
	defer srv.Close()

	_, out, err := newTestDirect(srv).conversation(context.Background(), nil, ConversationInput{Agent: "nonexistent"})
	core.AssertError(t, err)
	core.AssertFalse(t, out.Success)
}

// --- parseMessages ---

func TestMessaging_ParseMessages_Good(t *testing.T) {
	result := map[string]any{
		"data": []any{
			map[string]any{
				"id":           float64(5),
				"workspace_id": float64(8),
				"from":         "alice",
				"to":           "bob",
				"subject":      "hello",
				"content":      "hi there",
				"read":         true,
				"read_at":      "2026-03-10T10:01:00Z",
				"created_at":   "2026-03-10T10:00:00Z",
			},
		},
	}
	msgs := parseMessages(result)
	core.AssertLen(t, msgs, 1)
	core.AssertEqual(t, 5, msgs[0].ID)
	core.AssertEqual(t, 8, msgs[0].WorkspaceID)
	core.AssertEqual(t, "alice", msgs[0].From)
	core.AssertEqual(t, "bob", msgs[0].To)
	core.AssertEqual(t, "hello", msgs[0].Subject)
	core.AssertEqual(t, "hi there", msgs[0].Content)
	core.AssertTrue(t, msgs[0].Read)
	core.AssertEqual(t, "2026-03-10T10:01:00Z", msgs[0].ReadAt)
	core.AssertEqual(t, "2026-03-10T10:00:00Z", msgs[0].CreatedAt)
}

func TestMessaging_ParseMessages_Good_EmptyData(t *testing.T) {
	msgs := parseMessages(map[string]any{"data": []any{}})
	core.AssertEmpty(t, msgs)
	core.AssertLen(t, msgs, 0)
}

func TestMessaging_ParseMessages_Good_NoDataKey(t *testing.T) {
	msgs := parseMessages(map[string]any{"other": "value"})
	core.AssertEmpty(t, msgs)
	core.AssertLen(t, msgs, 0)
}

func TestMessaging_ParseMessages_Good_NilResult(t *testing.T) {
	msgs := parseMessages(nil)
	core.AssertEmpty(t, msgs)
	core.AssertLen(t, msgs, 0)
}

// --- toInt ---

func TestMessaging_ToInt_Good_Float64(t *testing.T) {
	got := toInt(float64(42))
	core.AssertEqual(t, 42, got)
	core.AssertGreater(t, got, 0)
}

func TestMessaging_ToInt_Good_Zero(t *testing.T) {
	got := toInt(float64(0))
	core.AssertEqual(t, 0, got)
	core.AssertGreaterOrEqual(t, got, 0)
}

func TestMessaging_ToInt_Bad_String(t *testing.T) {
	got := toInt("not a number")
	core.AssertEqual(t, 0, got)
	core.AssertEmpty(t, got)
}

func TestMessaging_ToInt_Bad_Nil(t *testing.T) {
	got := toInt(nil)
	core.AssertEqual(t, 0, got)
	core.AssertEmpty(t, got)
}

func TestMessaging_ToInt_Bad_Int(t *testing.T) {
	// Go JSON decode always uses float64, so int returns 0.
	got := toInt(42)
	core.AssertEqual(t, 0, got)
	core.AssertEmpty(t, got)
}

// --- Messaging struct round-trips ---

func TestMessaging_SendInput_Good_RoundTrip(t *testing.T) {
	in := SendInput{To: "charon", Content: "hello", Subject: "test"}
	var out SendInput
	roundTrip(t, in, &out)
	core.AssertEqual(t, in, out)
}

func TestMessaging_SendOutput_Good_RoundTrip(t *testing.T) {
	in := SendOutput{Success: true, ID: 42, To: "charon"}
	var out SendOutput
	roundTrip(t, in, &out)
	core.AssertEqual(t, in, out)
}

func TestMessaging_InboxOutput_Good_RoundTrip(t *testing.T) {
	in := InboxOutput{
		Success: true,
		Messages: []MessageItem{
			{ID: 1, WorkspaceID: 7, FromAgent: "a", ToAgent: "b", Content: "hi", Read: false, CreatedAt: "2026-03-10T12:00:00Z"},
		},
	}
	var out InboxOutput
	roundTrip(t, in, &out)
	core.AssertEqual(t, in.Success, out.Success)
	core.AssertLen(t, out.Messages, 1)
	core.AssertEqual(t, 7, out.Messages[0].WorkspaceID)
	core.AssertEqual(t, "a", out.Messages[0].FromAgent)
}

func TestMessaging_ConversationOutput_Good_RoundTrip(t *testing.T) {
	in := ConversationOutput{
		Success: true,
		Messages: []MessageItem{
			{ID: 10, From: "x", To: "y", Content: "thread", Read: true, ReadAt: "2026-03-10T14:01:00Z", CreatedAt: "2026-03-10T14:00:00Z"},
		},
	}
	var out ConversationOutput
	roundTrip(t, in, &out)
	core.AssertTrue(t, out.Success)
	core.AssertLen(t, out.Messages, 1)
}
