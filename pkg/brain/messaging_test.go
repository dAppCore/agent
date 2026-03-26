// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// localDirect returns a DirectSubsystem that never hits the network.
// Suitable for tests that validate input before making API calls.
func localDirect() *DirectSubsystem {
	return &DirectSubsystem{apiURL: "http://localhost", apiKey: "test-key"}
}

// --- sendMessage ---

func TestSendMessage_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/messages/send", r.URL.Path)

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "charon", body["to"])
		assert.Equal(t, "deploy complete", body["content"])
		assert.Equal(t, "status update", body["subject"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": float64(42)},
		})
	}))
	defer srv.Close()

	_, out, err := newTestDirect(srv).sendMessage(context.Background(), nil, SendInput{
		To:      "charon",
		Content: "deploy complete",
		Subject: "status update",
	})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Equal(t, 42, out.ID)
	assert.Equal(t, "charon", out.To)
}

func TestSendMessage_Bad_EmptyTo(t *testing.T) {
	_, _, err := localDirect().sendMessage(context.Background(), nil, SendInput{
		To:      "",
		Content: "hello",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "to and content are required")
}

func TestSendMessage_Bad_EmptyContent(t *testing.T) {
	_, _, err := localDirect().sendMessage(context.Background(), nil, SendInput{
		To:      "charon",
		Content: "",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "to and content are required")
}

func TestSendMessage_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(errorHandler(http.StatusInternalServerError, `{"error":"queue full"}`))
	defer srv.Close()

	_, out, err := newTestDirect(srv).sendMessage(context.Background(), nil, SendInput{
		To:      "charon",
		Content: "hello",
	})
	require.Error(t, err)
	assert.False(t, out.Success)
}

// --- inbox ---

func TestInbox_Good_WithMessages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/v1/messages/inbox")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []any{
				map[string]any{
					"id":         float64(1),
					"from":       "charon",
					"to":         "cladius",
					"subject":    "status",
					"content":    "deploy done",
					"read":       true,
					"created_at": "2026-03-10T12:00:00Z",
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
		})
	}))
	defer srv.Close()

	_, out, err := newTestDirect(srv).inbox(context.Background(), nil, InboxInput{Agent: "cladius"})
	require.NoError(t, err)
	assert.True(t, out.Success)
	require.Len(t, out.Messages, 2)
	assert.Equal(t, 1, out.Messages[0].ID)
	assert.Equal(t, "charon", out.Messages[0].From)
	assert.Equal(t, "deploy done", out.Messages[0].Content)
	assert.True(t, out.Messages[0].Read)
	assert.Equal(t, 2, out.Messages[1].ID)
	assert.False(t, out.Messages[1].Read)
}

func TestInbox_Good_EmptyInbox(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(map[string]any{"data": []any{}}))
	defer srv.Close()

	_, out, err := newTestDirect(srv).inbox(context.Background(), nil, InboxInput{Agent: "cladius"})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Empty(t, out.Messages)
}

func TestInbox_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(errorHandler(http.StatusInternalServerError, `{"error":"db down"}`))
	defer srv.Close()

	_, out, err := newTestDirect(srv).inbox(context.Background(), nil, InboxInput{})
	require.Error(t, err)
	assert.False(t, out.Success)
}

// --- conversation ---

func TestConversation_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/v1/messages/conversation/charon")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
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
		})
	}))
	defer srv.Close()

	_, out, err := newTestDirect(srv).conversation(context.Background(), nil, ConversationInput{Agent: "charon"})
	require.NoError(t, err)
	assert.True(t, out.Success)
	require.Len(t, out.Messages, 2)
	assert.Equal(t, "how is the deploy?", out.Messages[0].Content)
	assert.Equal(t, "all green", out.Messages[1].Content)
}

func TestConversation_Bad_EmptyAgent(t *testing.T) {
	_, _, err := localDirect().conversation(context.Background(), nil, ConversationInput{Agent: ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent is required")
}

func TestConversation_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(errorHandler(http.StatusNotFound, `{"error":"agent not found"}`))
	defer srv.Close()

	_, out, err := newTestDirect(srv).conversation(context.Background(), nil, ConversationInput{Agent: "nonexistent"})
	require.Error(t, err)
	assert.False(t, out.Success)
}

// --- parseMessages ---

func TestParseMessages_Good(t *testing.T) {
	result := map[string]any{
		"data": []any{
			map[string]any{
				"id":         float64(5),
				"from":       "alice",
				"to":         "bob",
				"subject":    "hello",
				"content":    "hi there",
				"read":       true,
				"created_at": "2026-03-10T10:00:00Z",
			},
		},
	}
	msgs := parseMessages(result)
	require.Len(t, msgs, 1)
	assert.Equal(t, 5, msgs[0].ID)
	assert.Equal(t, "alice", msgs[0].From)
	assert.Equal(t, "bob", msgs[0].To)
	assert.Equal(t, "hello", msgs[0].Subject)
	assert.Equal(t, "hi there", msgs[0].Content)
	assert.True(t, msgs[0].Read)
	assert.Equal(t, "2026-03-10T10:00:00Z", msgs[0].CreatedAt)
}

func TestParseMessages_Good_EmptyData(t *testing.T) {
	msgs := parseMessages(map[string]any{"data": []any{}})
	assert.Empty(t, msgs)
}

func TestParseMessages_Good_NoDataKey(t *testing.T) {
	msgs := parseMessages(map[string]any{"other": "value"})
	assert.Empty(t, msgs)
}

func TestParseMessages_Good_NilResult(t *testing.T) {
	assert.Empty(t, parseMessages(nil))
}

// --- toInt ---

func TestToInt_Good_Float64(t *testing.T) {
	assert.Equal(t, 42, toInt(float64(42)))
}

func TestToInt_Good_Zero(t *testing.T) {
	assert.Equal(t, 0, toInt(float64(0)))
}

func TestToInt_Bad_String(t *testing.T) {
	assert.Equal(t, 0, toInt("not a number"))
}

func TestToInt_Bad_Nil(t *testing.T) {
	assert.Equal(t, 0, toInt(nil))
}

func TestToInt_Bad_Int(t *testing.T) {
	// Go JSON decode always uses float64, so int returns 0.
	assert.Equal(t, 0, toInt(42))
}

// --- Messaging struct round-trips ---

func TestSendInput_Good_RoundTrip(t *testing.T) {
	in := SendInput{To: "charon", Content: "hello", Subject: "test"}
	var out SendInput
	roundTrip(t, in, &out)
	assert.Equal(t, in, out)
}

func TestSendOutput_Good_RoundTrip(t *testing.T) {
	in := SendOutput{Success: true, ID: 42, To: "charon"}
	var out SendOutput
	roundTrip(t, in, &out)
	assert.Equal(t, in, out)
}

func TestInboxOutput_Good_RoundTrip(t *testing.T) {
	in := InboxOutput{
		Success: true,
		Messages: []MessageItem{
			{ID: 1, From: "a", To: "b", Content: "hi", Read: false, CreatedAt: "2026-03-10T12:00:00Z"},
		},
	}
	var out InboxOutput
	roundTrip(t, in, &out)
	assert.Equal(t, in.Success, out.Success)
	require.Len(t, out.Messages, 1)
	assert.Equal(t, "a", out.Messages[0].From)
}

func TestConversationOutput_Good_RoundTrip(t *testing.T) {
	in := ConversationOutput{
		Success: true,
		Messages: []MessageItem{
			{ID: 10, From: "x", To: "y", Content: "thread", Read: true, CreatedAt: "2026-03-10T14:00:00Z"},
		},
	}
	var out ConversationOutput
	roundTrip(t, in, &out)
	assert.True(t, out.Success)
	require.Len(t, out.Messages, 1)
}
