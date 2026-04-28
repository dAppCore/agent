// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core "dappco.re/go"
	"dappco.re/go/mcp/pkg/mcp/ide"
	"dappco.re/go/ws"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type bridgeCapture struct {
	err error
	msg ide.BridgeMessage
}

func init() {
	gin.SetMode(gin.TestMode)
}

func setupRouter(p *BrainProvider) *gin.Engine {
	r := gin.New()
	g := r.Group(p.BasePath())
	p.RegisterRoutes(g)
	return r
}

func providerRequest(t *testing.T, p *BrainProvider, method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	r := setupRouter(p)
	w := httptest.NewRecorder()
	var req *http.Request
	if body != nil {
		req, _ = http.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, _ = http.NewRequest(method, path, nil)
	}
	r.ServeHTTP(w, req)
	return w
}

func connectedBridge(t *testing.T) (*ide.Bridge, <-chan bridgeCapture, func()) {
	t.Helper()

	captures := make(chan bridgeCapture, 4)
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			captures <- bridgeCapture{err: err}
			return
		}
		defer conn.Close()

		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var msg ide.BridgeMessage
			if result := core.JSONUnmarshal(data, &msg); !result.OK {
				parseErr, _ := result.Value.(error)
				captures <- bridgeCapture{err: parseErr}
				return
			}
			captures <- bridgeCapture{msg: msg}
		}
	}))

	bridge := ide.NewBridge(nil, ide.Config{
		LaravelWSURL:         core.Replace(server.URL, "http://", "ws://"),
		ReconnectInterval:    10 * time.Millisecond,
		MaxReconnectInterval: 20 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())
	bridge.Start(ctx)
	requireEventually(t, bridge.Connected, time.Second, 10*time.Millisecond)

	cleanup := func() {
		cancel()
		bridge.Shutdown()
		server.Close()
	}

	return bridge, captures, cleanup
}

func providerResponseData(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var response map[string]any
	core.RequireTrue(t, core.JSONUnmarshal(w.Body.Bytes(), &response).OK)
	data, ok := response["data"].(map[string]any)
	core.RequireTrue(t, ok)
	return data
}

func receiveBridgeMessage(t *testing.T, captures <-chan bridgeCapture) ide.BridgeMessage {
	t.Helper()
	select {
	case capture := <-captures:
		core.RequireNoError(t, capture.err)
		return capture.msg
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for bridge message")
		return ide.BridgeMessage{}
	}
}

func providerRouteSignatures(r *gin.Engine) []string {
	routes := r.Routes()
	signatures := make([]string, 0, len(routes))
	for _, route := range routes {
		signatures = append(signatures, core.Concat(route.Method, " ", route.Path))
	}
	return signatures
}

func TestProvider_NewProvider_Good(t *testing.T) {
	bridge := ide.NewBridge(nil, ide.Config{})
	hub := ws.NewHub()

	p := NewProvider(bridge, hub)

	core.AssertNotNil(t, p)
	core.AssertSame(t, bridge, p.bridge)
	core.AssertSame(t, hub, p.hub)
}

func TestNilDependencies_NewProvider_Bad(t *testing.T) {
	p := NewProvider(nil, nil)

	core.AssertNotNil(t, p)
	core.AssertNil(t, p.bridge)
	core.AssertNil(t, p.hub)
}

func TestMixedDependencies_NewProvider_Ugly(t *testing.T) {
	bridge := ide.NewBridge(nil, ide.Config{})

	p := NewProvider(bridge, nil)

	core.AssertNotNil(t, p)
	core.AssertSame(t, bridge, p.bridge)
	core.AssertNil(t, p.hub)
}

func TestDefaultProvider_BrainProvider_Name_Good(t *testing.T) {
	got := NewProvider(nil, nil).Name()
	core.AssertEqual(t, "brain", got)
	core.AssertNotEmpty(t, got)
}

func TestZeroValueProvider_BrainProvider_Name_Bad(t *testing.T) {
	got := (&BrainProvider{}).Name()
	core.AssertEqual(t, "brain", got)
	core.AssertNotContains(t, got, "/")
}

func TestNilProvider_BrainProvider_Name_Ugly(t *testing.T) {
	var p *BrainProvider
	got := p.Name()
	core.AssertEqual(t, "brain", got)
	core.AssertContains(t, got, "brain")
}

func TestDefaultProvider_BrainProvider_BasePath_Good(t *testing.T) {
	got := NewProvider(nil, nil).BasePath()
	core.AssertEqual(t, "/api/brain", got)
	core.AssertContains(t, got, "/api/")
}

func TestZeroValueProvider_BrainProvider_BasePath_Bad(t *testing.T) {
	got := (&BrainProvider{}).BasePath()
	core.AssertEqual(t, "/api/brain", got)
	core.AssertContains(t, got, "brain")
}

func TestNilProvider_BrainProvider_BasePath_Ugly(t *testing.T) {
	var p *BrainProvider
	got := p.BasePath()
	core.AssertEqual(t, "/api/brain", got)
	core.AssertNotContains(t, got, "//api")
}

func TestDefaultProvider_BrainProvider_Channels_Good(t *testing.T) {
	channels := NewProvider(nil, nil).Channels()

	core.AssertEqual(t, []string{
		"brain.remember.complete",
		"brain.recall.complete",
		"brain.forget.complete",
	}, channels)
}

func TestZeroValueProvider_BrainProvider_Channels_Bad(t *testing.T) {
	channels := (&BrainProvider{}).Channels()
	core.AssertLen(t, channels, 3)
	core.AssertContains(t, channels, "brain.recall.complete")
}

func TestDetachedSlice_BrainProvider_Channels_Ugly(t *testing.T) {
	channels := NewProvider(nil, nil).Channels()
	channels[0] = "changed"

	core.AssertEqual(t, "brain.remember.complete", NewProvider(nil, nil).Channels()[0])
}

func TestDefaultProvider_BrainProvider_Element_Good(t *testing.T) {
	element := NewProvider(nil, nil).Element()

	core.AssertEqual(t, "core-brain-panel", element.Tag)
	core.AssertEqual(t, "/assets/brain-panel.js", element.Source)
}

func TestZeroValueProvider_BrainProvider_Element_Bad(t *testing.T) {
	element := (&BrainProvider{}).Element()

	core.AssertEqual(t, "core-brain-panel", element.Tag)
	core.AssertEqual(t, "/assets/brain-panel.js", element.Source)
}

func TestDetachedValue_BrainProvider_Element_Ugly(t *testing.T) {
	element := NewProvider(nil, nil).Element()
	element.Tag = "changed"

	core.AssertEqual(t, "core-brain-panel", NewProvider(nil, nil).Element().Tag)
}

func TestDefaultProvider_BrainProvider_RegisterRoutes_Good(t *testing.T) {
	signatures := providerRouteSignatures(setupRouter(NewProvider(nil, nil)))

	core.AssertElementsMatch(t, []string{
		"POST /api/brain/remember",
		"POST /api/brain/recall",
		"POST /api/brain/forget",
		"GET /api/brain/list",
		"GET /api/brain/status",
	}, signatures)
}

func TestZeroValueProvider_BrainProvider_RegisterRoutes_Bad(t *testing.T) {
	provider := &BrainProvider{}

	status := providerRequest(t, provider, "GET", "/api/brain/status", nil)
	list := providerRequest(t, provider, "GET", "/api/brain/list", nil)

	core.AssertEqual(t, http.StatusOK, status.Code)
	core.AssertEqual(t, http.StatusServiceUnavailable, list.Code)
}

func TestCustomGroup_BrainProvider_RegisterRoutes_Ugly(t *testing.T) {
	r := gin.New()
	NewProvider(nil, nil).RegisterRoutes(r.Group("/v1/brain"))

	core.AssertElementsMatch(t, []string{
		"POST /v1/brain/remember",
		"POST /v1/brain/recall",
		"POST /v1/brain/forget",
		"GET /v1/brain/list",
		"GET /v1/brain/status",
	}, providerRouteSignatures(r))
}

func TestDefaultProvider_BrainProvider_Describe_Good(t *testing.T) {
	descriptions := NewProvider(nil, nil).Describe()

	core.AssertLen(t, descriptions, 5)
	core.AssertEqual(t, "POST", descriptions[0].Method)
	core.AssertEqual(t, "/remember", descriptions[0].Path)
	core.AssertEqual(t, "GET", descriptions[4].Method)
	core.AssertEqual(t, "/status", descriptions[4].Path)
}

func TestZeroValueProvider_BrainProvider_Describe_Bad(t *testing.T) {
	descriptions := (&BrainProvider{}).Describe()

	core.AssertLen(t, descriptions, 5)
	core.AssertEqual(t, "/list", descriptions[3].Path)
}

func TestDetachedSlice_BrainProvider_Describe_Ugly(t *testing.T) {
	descriptions := NewProvider(nil, nil).Describe()
	descriptions[0].Path = "/changed"

	core.AssertEqual(t, "/remember", NewProvider(nil, nil).Describe()[0].Path)
}

func TestProvider_Describe_Good_BrainFields(t *testing.T) {
	descriptions := NewProvider(nil, nil).Describe()

	rememberBody, ok := descriptions[0].RequestBody.(map[string]any)
	core.RequireTrue(t, ok)
	rememberProps := rememberBody["properties"].(map[string]any)
	core.AssertContains(t, rememberProps, "supersedes")
	core.AssertContains(t, rememberProps, "expires_in")

	recallBody, ok := descriptions[1].RequestBody.(map[string]any)
	core.RequireTrue(t, ok)
	recallProps := recallBody["properties"].(map[string]any)
	filterProps := recallProps["filter"].(map[string]any)["properties"].(map[string]any)
	core.AssertContains(t, filterProps, "agent_id")
	core.AssertContains(t, filterProps, "min_confidence")
}

func TestProvider_Status_Good(t *testing.T) {
	bridge, _, cleanup := connectedBridge(t)
	defer cleanup()

	w := providerRequest(t, NewProvider(bridge, nil), "GET", "/api/brain/status", nil)

	core.AssertEqual(t, http.StatusOK, w.Code)
	core.AssertEqual(t, true, providerResponseData(t, w)["connected"])
}

func TestProvider_Status_Bad_NoBridge(t *testing.T) {
	w := providerRequest(t, NewProvider(nil, nil), "GET", "/api/brain/status", nil)

	core.AssertEqual(t, http.StatusOK, w.Code)
	core.AssertEqual(t, false, providerResponseData(t, w)["connected"])
}

func TestProvider_Status_Ugly_DisconnectedBridge(t *testing.T) {
	w := providerRequest(t, NewProvider(&ide.Bridge{}, nil), "GET", "/api/brain/status", nil)

	core.AssertEqual(t, http.StatusOK, w.Code)
	core.AssertEqual(t, false, providerResponseData(t, w)["connected"])
}

func TestProvider_Remember_Good(t *testing.T) {
	bridge, captures, cleanup := connectedBridge(t)
	defer cleanup()

	body := []byte(core.JSONMarshalString(map[string]any{
		"content":    "Use core.Env for system paths.",
		"type":       "convention",
		"project":    "agent",
		"confidence": 0.9,
		"tags":       []string{"ax", "paths"},
	}))
	w := providerRequest(t, NewProvider(bridge, nil), "POST", "/api/brain/remember", body)

	core.AssertEqual(t, http.StatusOK, w.Code)
	core.AssertEqual(t, true, providerResponseData(t, w)["success"])

	msg := receiveBridgeMessage(t, captures)
	data, ok := msg.Data.(map[string]any)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "brain_remember", msg.Type)
	core.AssertEqual(t, "Use core.Env for system paths.", data["content"])
	core.AssertEqual(t, "convention", data["type"])
	core.AssertEqual(t, "agent", data["project"])
}

func TestProvider_Remember_Bad_InvalidInput(t *testing.T) {
	w := providerRequest(t, NewProvider(&ide.Bridge{}, nil), "POST", "/api/brain/remember", []byte("not json"))
	core.AssertEqual(t, http.StatusBadRequest, w.Code)
	core.AssertNotEmpty(t, w.Body.String())
}

func TestProvider_Remember_Ugly_DisconnectedBridge(t *testing.T) {
	body := []byte(core.JSONMarshalString(map[string]any{"content": "test memory", "type": "observation"}))
	w := providerRequest(t, NewProvider(&ide.Bridge{}, nil), "POST", "/api/brain/remember", body)
	core.AssertEqual(t, http.StatusInternalServerError, w.Code)
}

func TestProvider_Recall_Good(t *testing.T) {
	bridge, captures, cleanup := connectedBridge(t)
	defer cleanup()

	body := []byte(core.JSONMarshalString(map[string]any{
		"query": "workspace path helpers",
		"top_k": 3,
		"filter": map[string]any{
			"project": "agent",
			"type":    "convention",
		},
	}))
	w := providerRequest(t, NewProvider(bridge, nil), "POST", "/api/brain/recall", body)

	core.AssertEqual(t, http.StatusOK, w.Code)
	data := providerResponseData(t, w)
	core.AssertEqual(t, true, data["success"])
	core.AssertEqual(t, 0.0, data["count"])

	msg := receiveBridgeMessage(t, captures)
	payload, ok := msg.Data.(map[string]any)
	core.RequireTrue(t, ok)
	filter, ok := payload["filter"].(map[string]any)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "brain_recall", msg.Type)
	core.AssertEqual(t, "workspace path helpers", payload["query"])
	core.AssertEqual(t, 3.0, payload["top_k"])
	core.AssertEqual(t, "agent", filter["project"])
	core.AssertEqual(t, "convention", filter["type"])
}

func TestProvider_Recall_Bad_InvalidInput(t *testing.T) {
	w := providerRequest(t, NewProvider(&ide.Bridge{}, nil), "POST", "/api/brain/recall", []byte("not json"))
	core.AssertEqual(t, http.StatusBadRequest, w.Code)
	core.AssertNotEmpty(t, w.Body.String())
}

func TestProvider_Recall_Ugly_DisconnectedBridge(t *testing.T) {
	body := []byte(core.JSONMarshalString(map[string]any{"query": "test"}))
	w := providerRequest(t, NewProvider(&ide.Bridge{}, nil), "POST", "/api/brain/recall", body)
	core.AssertEqual(t, http.StatusInternalServerError, w.Code)
}

func TestProvider_Forget_Good(t *testing.T) {
	bridge, captures, cleanup := connectedBridge(t)
	defer cleanup()

	body := []byte(core.JSONMarshalString(map[string]any{"id": "mem-123", "reason": "superseded"}))
	w := providerRequest(t, NewProvider(bridge, nil), "POST", "/api/brain/forget", body)

	core.AssertEqual(t, http.StatusOK, w.Code)
	core.AssertEqual(t, "mem-123", providerResponseData(t, w)["forgotten"])

	msg := receiveBridgeMessage(t, captures)
	data, ok := msg.Data.(map[string]any)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "brain_forget", msg.Type)
	core.AssertEqual(t, "mem-123", data["id"])
	core.AssertEqual(t, "superseded", data["reason"])
}

func TestProvider_Forget_Bad_InvalidInput(t *testing.T) {
	w := providerRequest(t, NewProvider(&ide.Bridge{}, nil), "POST", "/api/brain/forget", []byte("not json"))
	core.AssertEqual(t, http.StatusBadRequest, w.Code)
	core.AssertNotEmpty(t, w.Body.String())
}

func TestProvider_Forget_Ugly_DisconnectedBridge(t *testing.T) {
	body := []byte(core.JSONMarshalString(map[string]any{"id": "mem-123"}))
	w := providerRequest(t, NewProvider(&ide.Bridge{}, nil), "POST", "/api/brain/forget", body)
	core.AssertEqual(t, http.StatusInternalServerError, w.Code)
}

func TestProvider_List_Good(t *testing.T) {
	bridge, captures, cleanup := connectedBridge(t)
	defer cleanup()

	w := providerRequest(t, NewProvider(bridge, nil), "GET", "/api/brain/list?project=agent&type=convention&agent_id=codex&limit=2", nil)

	core.AssertEqual(t, http.StatusOK, w.Code)
	data := providerResponseData(t, w)
	core.AssertEqual(t, true, data["success"])
	core.AssertEqual(t, 0.0, data["count"])

	msg := receiveBridgeMessage(t, captures)
	payload, ok := msg.Data.(map[string]any)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "brain_list", msg.Type)
	core.AssertEqual(t, "agent", payload["project"])
	core.AssertEqual(t, "convention", payload["type"])
	core.AssertEqual(t, "codex", payload["agent_id"])
	core.AssertEqual(t, 2.0, payload["limit"])
}

func TestProvider_List_Bad_InvalidLimit(t *testing.T) {
	w := providerRequest(t, NewProvider(&ide.Bridge{}, nil), "GET", "/api/brain/list?limit=abc", nil)
	core.AssertEqual(t, http.StatusBadRequest, w.Code)
	core.AssertNotEmpty(t, w.Body.String())
}

func TestProvider_List_Ugly_DisconnectedBridge(t *testing.T) {
	w := providerRequest(t, NewProvider(&ide.Bridge{}, nil), "GET", "/api/brain/list?limit=2", nil)
	core.AssertEqual(t, http.StatusInternalServerError, w.Code)
	core.AssertNotEmpty(t, w.Body.String())
}

func TestProvider_EmitEvent_Good(t *testing.T) {
	core.AssertNotPanics(t, func() {
		NewProvider(nil, ws.NewHub()).emitEvent("brain.test", map[string]any{"foo": "bar"})
	})
}

func TestProvider_EmitEvent_Bad_EmptyChannel(t *testing.T) {
	core.AssertNotPanics(t, func() {
		NewProvider(nil, ws.NewHub()).emitEvent("", map[string]any{"foo": "bar"})
	})
}

func TestProvider_EmitEvent_Ugly_NilHub(t *testing.T) {
	core.AssertNotPanics(t, func() {
		NewProvider(nil, nil).emitEvent("brain.test", map[string]any{"foo": "bar"})
	})
}
