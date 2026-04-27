// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core "dappco.re/go/core"
	"dappco.re/go/ws"
	"dappco.re/go/mcp/pkg/mcp/ide"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.Eventually(t, bridge.Connected, time.Second, 10*time.Millisecond)

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
	require.True(t, core.JSONUnmarshal(w.Body.Bytes(), &response).OK)
	data, ok := response["data"].(map[string]any)
	require.True(t, ok)
	return data
}

func receiveBridgeMessage(t *testing.T, captures <-chan bridgeCapture) ide.BridgeMessage {
	t.Helper()
	select {
	case capture := <-captures:
		require.NoError(t, capture.err)
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

	require.NotNil(t, p)
	assert.Same(t, bridge, p.bridge)
	assert.Same(t, hub, p.hub)
}

func TestProvider_NewProvider_Bad_NilDependencies(t *testing.T) {
	p := NewProvider(nil, nil)

	require.NotNil(t, p)
	assert.Nil(t, p.bridge)
	assert.Nil(t, p.hub)
}

func TestProvider_NewProvider_Ugly_MixedDependencies(t *testing.T) {
	bridge := ide.NewBridge(nil, ide.Config{})

	p := NewProvider(bridge, nil)

	require.NotNil(t, p)
	assert.Same(t, bridge, p.bridge)
	assert.Nil(t, p.hub)
}

func TestProvider_Name_Good(t *testing.T) {
	assert.Equal(t, "brain", NewProvider(nil, nil).Name())
}

func TestProvider_Name_Bad_ZeroValueReceiver(t *testing.T) {
	assert.Equal(t, "brain", (&BrainProvider{}).Name())
}

func TestProvider_Name_Ugly_NilReceiver(t *testing.T) {
	var p *BrainProvider
	assert.Equal(t, "brain", p.Name())
}

func TestProvider_BasePath_Good(t *testing.T) {
	assert.Equal(t, "/api/brain", NewProvider(nil, nil).BasePath())
}

func TestProvider_BasePath_Bad_ZeroValueReceiver(t *testing.T) {
	assert.Equal(t, "/api/brain", (&BrainProvider{}).BasePath())
}

func TestProvider_BasePath_Ugly_NilReceiver(t *testing.T) {
	var p *BrainProvider
	assert.Equal(t, "/api/brain", p.BasePath())
}

func TestProvider_Channels_Good(t *testing.T) {
	channels := NewProvider(nil, nil).Channels()

	assert.Equal(t, []string{
		"brain.remember.complete",
		"brain.recall.complete",
		"brain.forget.complete",
	}, channels)
}

func TestProvider_Channels_Bad_ZeroValueReceiver(t *testing.T) {
	channels := (&BrainProvider{}).Channels()
	assert.Len(t, channels, 3)
}

func TestProvider_Channels_Ugly_ReturnSliceIsDetached(t *testing.T) {
	channels := NewProvider(nil, nil).Channels()
	channels[0] = "changed"

	assert.Equal(t, "brain.remember.complete", NewProvider(nil, nil).Channels()[0])
}

func TestProvider_Element_Good(t *testing.T) {
	element := NewProvider(nil, nil).Element()

	assert.Equal(t, "core-brain-panel", element.Tag)
	assert.Equal(t, "/assets/brain-panel.js", element.Source)
}

func TestProvider_Element_Bad_ZeroValueReceiver(t *testing.T) {
	element := (&BrainProvider{}).Element()

	assert.Equal(t, "core-brain-panel", element.Tag)
	assert.Equal(t, "/assets/brain-panel.js", element.Source)
}

func TestProvider_Element_Ugly_ReturnValueIsDetached(t *testing.T) {
	element := NewProvider(nil, nil).Element()
	element.Tag = "changed"

	assert.Equal(t, "core-brain-panel", NewProvider(nil, nil).Element().Tag)
}

func TestProvider_RegisterRoutes_Good(t *testing.T) {
	signatures := providerRouteSignatures(setupRouter(NewProvider(nil, nil)))

	assert.ElementsMatch(t, []string{
		"POST /api/brain/remember",
		"POST /api/brain/recall",
		"POST /api/brain/forget",
		"GET /api/brain/list",
		"GET /api/brain/status",
	}, signatures)
}

func TestProvider_RegisterRoutes_Bad_ZeroValueProvider(t *testing.T) {
	provider := &BrainProvider{}

	status := providerRequest(t, provider, "GET", "/api/brain/status", nil)
	list := providerRequest(t, provider, "GET", "/api/brain/list", nil)

	assert.Equal(t, http.StatusOK, status.Code)
	assert.Equal(t, http.StatusServiceUnavailable, list.Code)
}

func TestProvider_RegisterRoutes_Ugly_CustomGroup(t *testing.T) {
	r := gin.New()
	NewProvider(nil, nil).RegisterRoutes(r.Group("/v1/brain"))

	assert.ElementsMatch(t, []string{
		"POST /v1/brain/remember",
		"POST /v1/brain/recall",
		"POST /v1/brain/forget",
		"GET /v1/brain/list",
		"GET /v1/brain/status",
	}, providerRouteSignatures(r))
}

func TestProvider_Describe_Good(t *testing.T) {
	descriptions := NewProvider(nil, nil).Describe()

	assert.Len(t, descriptions, 5)
	assert.Equal(t, "POST", descriptions[0].Method)
	assert.Equal(t, "/remember", descriptions[0].Path)
	assert.Equal(t, "GET", descriptions[4].Method)
	assert.Equal(t, "/status", descriptions[4].Path)
}

func TestProvider_Describe_Bad_ZeroValueReceiver(t *testing.T) {
	descriptions := (&BrainProvider{}).Describe()

	assert.Len(t, descriptions, 5)
	assert.Equal(t, "/list", descriptions[3].Path)
}

func TestProvider_Describe_Ugly_ReturnSliceIsDetached(t *testing.T) {
	descriptions := NewProvider(nil, nil).Describe()
	descriptions[0].Path = "/changed"

	assert.Equal(t, "/remember", NewProvider(nil, nil).Describe()[0].Path)
}

func TestProvider_Describe_Good_BrainFields(t *testing.T) {
	descriptions := NewProvider(nil, nil).Describe()

	rememberProps := descriptions[0].RequestBody["properties"].(map[string]any)
	assert.Contains(t, rememberProps, "supersedes")
	assert.Contains(t, rememberProps, "expires_in")

	recallProps := descriptions[1].RequestBody["properties"].(map[string]any)
	filterProps := recallProps["filter"].(map[string]any)["properties"].(map[string]any)
	assert.Contains(t, filterProps, "agent_id")
	assert.Contains(t, filterProps, "min_confidence")
}

func TestProvider_Status_Good(t *testing.T) {
	bridge, _, cleanup := connectedBridge(t)
	defer cleanup()

	w := providerRequest(t, NewProvider(bridge, nil), "GET", "/api/brain/status", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, true, providerResponseData(t, w)["connected"])
}

func TestProvider_Status_Bad_NoBridge(t *testing.T) {
	w := providerRequest(t, NewProvider(nil, nil), "GET", "/api/brain/status", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, false, providerResponseData(t, w)["connected"])
}

func TestProvider_Status_Ugly_DisconnectedBridge(t *testing.T) {
	w := providerRequest(t, NewProvider(&ide.Bridge{}, nil), "GET", "/api/brain/status", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, false, providerResponseData(t, w)["connected"])
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

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, true, providerResponseData(t, w)["success"])

	msg := receiveBridgeMessage(t, captures)
	data, ok := msg.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "brain_remember", msg.Type)
	assert.Equal(t, "Use core.Env for system paths.", data["content"])
	assert.Equal(t, "convention", data["type"])
	assert.Equal(t, "agent", data["project"])
}

func TestProvider_Remember_Bad_InvalidInput(t *testing.T) {
	w := providerRequest(t, NewProvider(&ide.Bridge{}, nil), "POST", "/api/brain/remember", []byte("not json"))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProvider_Remember_Ugly_DisconnectedBridge(t *testing.T) {
	body := []byte(core.JSONMarshalString(map[string]any{"content": "test memory", "type": "observation"}))
	w := providerRequest(t, NewProvider(&ide.Bridge{}, nil), "POST", "/api/brain/remember", body)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
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

	assert.Equal(t, http.StatusOK, w.Code)
	data := providerResponseData(t, w)
	assert.Equal(t, true, data["success"])
	assert.Equal(t, 0.0, data["count"])

	msg := receiveBridgeMessage(t, captures)
	payload, ok := msg.Data.(map[string]any)
	require.True(t, ok)
	filter, ok := payload["filter"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "brain_recall", msg.Type)
	assert.Equal(t, "workspace path helpers", payload["query"])
	assert.Equal(t, 3.0, payload["top_k"])
	assert.Equal(t, "agent", filter["project"])
	assert.Equal(t, "convention", filter["type"])
}

func TestProvider_Recall_Bad_InvalidInput(t *testing.T) {
	w := providerRequest(t, NewProvider(&ide.Bridge{}, nil), "POST", "/api/brain/recall", []byte("not json"))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProvider_Recall_Ugly_DisconnectedBridge(t *testing.T) {
	body := []byte(core.JSONMarshalString(map[string]any{"query": "test"}))
	w := providerRequest(t, NewProvider(&ide.Bridge{}, nil), "POST", "/api/brain/recall", body)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestProvider_Forget_Good(t *testing.T) {
	bridge, captures, cleanup := connectedBridge(t)
	defer cleanup()

	body := []byte(core.JSONMarshalString(map[string]any{"id": "mem-123", "reason": "superseded"}))
	w := providerRequest(t, NewProvider(bridge, nil), "POST", "/api/brain/forget", body)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "mem-123", providerResponseData(t, w)["forgotten"])

	msg := receiveBridgeMessage(t, captures)
	data, ok := msg.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "brain_forget", msg.Type)
	assert.Equal(t, "mem-123", data["id"])
	assert.Equal(t, "superseded", data["reason"])
}

func TestProvider_Forget_Bad_InvalidInput(t *testing.T) {
	w := providerRequest(t, NewProvider(&ide.Bridge{}, nil), "POST", "/api/brain/forget", []byte("not json"))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProvider_Forget_Ugly_DisconnectedBridge(t *testing.T) {
	body := []byte(core.JSONMarshalString(map[string]any{"id": "mem-123"}))
	w := providerRequest(t, NewProvider(&ide.Bridge{}, nil), "POST", "/api/brain/forget", body)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestProvider_List_Good(t *testing.T) {
	bridge, captures, cleanup := connectedBridge(t)
	defer cleanup()

	w := providerRequest(t, NewProvider(bridge, nil), "GET", "/api/brain/list?project=agent&type=convention&agent_id=codex&limit=2", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	data := providerResponseData(t, w)
	assert.Equal(t, true, data["success"])
	assert.Equal(t, 0.0, data["count"])

	msg := receiveBridgeMessage(t, captures)
	payload, ok := msg.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "brain_list", msg.Type)
	assert.Equal(t, "agent", payload["project"])
	assert.Equal(t, "convention", payload["type"])
	assert.Equal(t, "codex", payload["agent_id"])
	assert.Equal(t, 2.0, payload["limit"])
}

func TestProvider_List_Bad_InvalidLimit(t *testing.T) {
	w := providerRequest(t, NewProvider(&ide.Bridge{}, nil), "GET", "/api/brain/list?limit=abc", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestProvider_List_Ugly_DisconnectedBridge(t *testing.T) {
	w := providerRequest(t, NewProvider(&ide.Bridge{}, nil), "GET", "/api/brain/list?limit=2", nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestProvider_EmitEvent_Good(t *testing.T) {
	assert.NotPanics(t, func() {
		NewProvider(nil, ws.NewHub()).emitEvent("brain.test", map[string]any{"foo": "bar"})
	})
}

func TestProvider_EmitEvent_Bad_EmptyChannel(t *testing.T) {
	assert.NotPanics(t, func() {
		NewProvider(nil, ws.NewHub()).emitEvent("", map[string]any{"foo": "bar"})
	})
}

func TestProvider_EmitEvent_Ugly_NilHub(t *testing.T) {
	assert.NotPanics(t, func() {
		NewProvider(nil, nil).emitEvent("brain.test", map[string]any{"foo": "bar"})
	})
}
