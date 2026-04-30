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
	"dappco.re/go/mcp/pkg/mcp/ide"
	providerws "dappco.re/go/ws"
	"github.com/gorilla/websocket"
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

	wsURL := "ws" + core.TrimPrefix(srv.URL, "http")
	hub := providerws.NewHub()
	bridge := ide.NewBridge(hub, ide.Config{
		LaravelWSURL:      wsURL,
		ReconnectInterval: 100 * time.Millisecond,
	})
	bridge.Start(context.Background())

	requireEventually(t, func() bool {
		return bridge.Connected()
	}, 2*time.Second, 10*time.Millisecond, "bridge did not connect")

	t.Cleanup(bridge.Shutdown)
	return bridge
}

// --- RegisterTools ---

func TestBrain_RegisterTools_Good(t *testing.T) {
	sub := New(nil)
	svc, err := coremcp.New(coremcp.Options{Unrestricted: true})
	core.RequireNoError(t, err)
	sub.RegisterTools(svc)
}

func TestDirect_RegisterTools_Good(t *testing.T) {
	t.Setenv("CORE_BRAIN_URL", "http://localhost")
	t.Setenv("CORE_BRAIN_KEY", "test-key")
	sub := NewDirect()
	svc, err := coremcp.New(coremcp.Options{Unrestricted: true})
	core.RequireNoError(t, err)
	sub.RegisterTools(svc)
}

// --- Subsystem with connected bridge ---

func TestBrain_RememberBridge_Good_Case(t *testing.T) {
	sub := New(testBridge(t))
	_, out, err := sub.brainRemember(context.Background(), nil, RememberInput{
		Content: "test memory",
		Type:    "observation",
		Tags:    []string{"test"},
		Project: "core",
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertFalse(t, out.Timestamp.IsZero())
}

func TestBrain_RecallBridge_Good_Case(t *testing.T) {
	sub := New(testBridge(t))
	_, out, err := sub.brainRecall(context.Background(), nil, RecallInput{
		Query: "architecture",
		TopK:  5,
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertEmpty(t, out.Memories)
}

func TestBrain_ForgetBridge_Good_Case(t *testing.T) {
	sub := New(testBridge(t))
	_, out, err := sub.brainForget(context.Background(), nil, ForgetInput{
		ID:     "mem-123",
		Reason: "outdated",
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertEqual(t, "mem-123", out.Forgotten)
	core.AssertFalse(t, out.Timestamp.IsZero())
}

func TestBrain_ListBridge_Good_Case(t *testing.T) {
	sub := New(testBridge(t))
	_, out, err := sub.brainList(context.Background(), nil, ListInput{
		Project: "core",
		Type:    "decision",
		Limit:   10,
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertEmpty(t, out.Memories)
}

// --- Provider handlers with connected bridge ---

func TestProvider_RememberBridge_Good_Case(t *testing.T) {
	p := NewProvider(testBridge(t), nil)
	body := []byte(core.JSONMarshalString(RememberInput{
		Content:    "provider test memory",
		Type:       "fact",
		Tags:       []string{"test"},
		Project:    "agent",
		Confidence: 0.9,
	}))
	w := providerRequest(t, p, "POST", "/api/brain/remember", body)
	core.AssertEqual(t, http.StatusOK, w.Code)
}

func TestProvider_RememberInvalid_Bad_Case(t *testing.T) {
	p := NewProvider(testBridge(t), nil)
	w := providerRequest(t, p, "POST", "/api/brain/remember", []byte("{"))
	core.AssertEqual(t, http.StatusBadRequest, w.Code)
}

func TestProvider_RecallBridge_Good_Case(t *testing.T) {
	p := NewProvider(testBridge(t), nil)
	body := []byte(core.JSONMarshalString(RecallInput{Query: "test", TopK: 5}))
	w := providerRequest(t, p, "POST", "/api/brain/recall", body)
	core.AssertEqual(t, http.StatusOK, w.Code)
}

func TestProvider_RecallInvalid_Bad_Case(t *testing.T) {
	p := NewProvider(testBridge(t), nil)
	w := providerRequest(t, p, "POST", "/api/brain/recall", []byte("bad"))
	core.AssertEqual(t, http.StatusBadRequest, w.Code)
}

func TestProvider_ForgetBridge_Good_Case(t *testing.T) {
	p := NewProvider(testBridge(t), nil)
	body := []byte(core.JSONMarshalString(ForgetInput{ID: "mem-abc", Reason: "outdated"}))
	w := providerRequest(t, p, "POST", "/api/brain/forget", body)
	core.AssertEqual(t, http.StatusOK, w.Code)
}

func TestProvider_ForgetInvalid_Bad_Case(t *testing.T) {
	p := NewProvider(testBridge(t), nil)
	w := providerRequest(t, p, "POST", "/api/brain/forget", []byte("{"))
	core.AssertEqual(t, http.StatusBadRequest, w.Code)
}

func TestProvider_ListBridge_Good_Case(t *testing.T) {
	p := NewProvider(testBridge(t), nil)
	w := providerRequest(t, p, "GET", "/api/brain/list?project=core&type=decision&limit=10", nil)
	core.AssertEqual(t, http.StatusOK, w.Code)
}

func TestProvider_StatusBridge_Good_Case(t *testing.T) {
	p := NewProvider(testBridge(t), nil)
	w := providerRequest(t, p, "GET", "/api/brain/status", nil)

	core.AssertEqual(t, http.StatusOK, w.Code)
	var resp map[string]any
	core.RequireTrue(t, core.JSONUnmarshal(w.Body.Bytes(), &resp).OK)
	data, _ := resp["data"].(map[string]any)
	core.AssertEqual(t, true, data["connected"])
}

// --- emitEvent with hub ---

func TestProvider_EmitEventHub_Good_Case(t *testing.T) {
	hub := providerws.NewHub()
	p := NewProvider(nil, hub)
	p.emitEvent("brain.test", map[string]any{"key": "value"})
}
