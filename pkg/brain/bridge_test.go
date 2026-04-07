// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	core "dappco.re/go/core"
	providerws "dappco.re/go/core/ws"
	"dappco.re/go/mcp/pkg/mcp/ide"
	"github.com/gorilla/websocket"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testWSServer creates a WS server that accepts connections and discards messages.
func testWSServer(t *testing.T) *httptest.Server {
	t.Helper()
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// testBridge creates a bridge connected to a test WS server.
func testBridge(t *testing.T) *ide.Bridge {
	t.Helper()
	srv := testWSServer(t)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	hub := providerws.NewHub()
	bridge := ide.NewBridge(hub, ide.Config{
		LaravelWSURL:      wsURL,
		ReconnectInterval: 100 * time.Millisecond,
	})
	bridge.Start(context.Background())

	require.Eventually(t, func() bool {
		return bridge.Connected()
	}, 2*time.Second, 10*time.Millisecond, "bridge did not connect")

	t.Cleanup(bridge.Shutdown)
	return bridge
}

// --- RegisterTools ---

func TestBrain_RegisterTools_Good(t *testing.T) {
	sub := New(nil)
	srv := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "test", Version: "0.1.0"}, nil)
	sub.RegisterTools(srv)
}

func TestDirect_RegisterTools_Good(t *testing.T) {
	t.Setenv("CORE_BRAIN_URL", "http://localhost")
	t.Setenv("CORE_BRAIN_KEY", "test-key")
	sub := NewDirect()
	srv := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "test", Version: "0.1.0"}, nil)
	sub.RegisterTools(srv)
}

// --- Subsystem with connected bridge ---

func TestBrain_RememberBridge_Good(t *testing.T) {
	sub := New(testBridge(t))
	_, out, err := sub.brainRemember(context.Background(), nil, RememberInput{
		Content: "test memory",
		Type:    "observation",
		Tags:    []string{"test"},
		Project: "core",
	})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.False(t, out.Timestamp.IsZero())
}

func TestBrain_RecallBridge_Good(t *testing.T) {
	sub := New(testBridge(t))
	_, out, err := sub.brainRecall(context.Background(), nil, RecallInput{
		Query: "architecture",
		TopK:  5,
	})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Empty(t, out.Memories)
}

func TestBrain_ForgetBridge_Good(t *testing.T) {
	sub := New(testBridge(t))
	_, out, err := sub.brainForget(context.Background(), nil, ForgetInput{
		ID:     "mem-123",
		Reason: "outdated",
	})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Equal(t, "mem-123", out.Forgotten)
	assert.False(t, out.Timestamp.IsZero())
}

func TestBrain_ListBridge_Good(t *testing.T) {
	sub := New(testBridge(t))
	_, out, err := sub.brainList(context.Background(), nil, ListInput{
		Project: "core",
		Type:    "decision",
		Limit:   10,
	})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Empty(t, out.Memories)
}

// --- Provider handlers with connected bridge ---

func TestProvider_RememberBridge_Good(t *testing.T) {
	p := NewProvider(testBridge(t), nil)
	body := []byte(core.JSONMarshalString(RememberInput{
		Content:    "provider test memory",
		Type:       "fact",
		Tags:       []string{"test"},
		Project:    "agent",
		Confidence: 0.9,
	}))
	w := providerRequest(t, p, "POST", "/api/brain/remember", body)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProvider_RememberInvalid_Bad(t *testing.T) {
	p := NewProvider(testBridge(t), nil)
	w := providerRequest(t, p, "POST", "/api/brain/remember", []byte("{"))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProvider_RecallBridge_Good(t *testing.T) {
	p := NewProvider(testBridge(t), nil)
	body := []byte(core.JSONMarshalString(RecallInput{Query: "test", TopK: 5}))
	w := providerRequest(t, p, "POST", "/api/brain/recall", body)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProvider_RecallInvalid_Bad(t *testing.T) {
	p := NewProvider(testBridge(t), nil)
	w := providerRequest(t, p, "POST", "/api/brain/recall", []byte("bad"))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProvider_ForgetBridge_Good(t *testing.T) {
	p := NewProvider(testBridge(t), nil)
	body := []byte(core.JSONMarshalString(ForgetInput{ID: "mem-abc", Reason: "outdated"}))
	w := providerRequest(t, p, "POST", "/api/brain/forget", body)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProvider_ForgetInvalid_Bad(t *testing.T) {
	p := NewProvider(testBridge(t), nil)
	w := providerRequest(t, p, "POST", "/api/brain/forget", []byte("{"))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProvider_ListBridge_Good(t *testing.T) {
	p := NewProvider(testBridge(t), nil)
	w := providerRequest(t, p, "GET", "/api/brain/list?project=core&type=decision&limit=10", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestProvider_StatusBridge_Good(t *testing.T) {
	p := NewProvider(testBridge(t), nil)
	w := providerRequest(t, p, "GET", "/api/brain/status", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.True(t, core.JSONUnmarshal(w.Body.Bytes(), &resp).OK)
	data, _ := resp["data"].(map[string]any)
	assert.Equal(t, true, data["connected"])
}

// --- emitEvent with hub ---

func TestProvider_EmitEventHub_Good(t *testing.T) {
	hub := providerws.NewHub()
	p := NewProvider(nil, hub)
	p.emitEvent("brain.test", map[string]any{"key": "value"})
}
