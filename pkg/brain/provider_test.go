// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// setupRouter creates a gin engine with provider routes registered.
func setupRouter(p *BrainProvider) *gin.Engine {
	r := gin.New()
	g := r.Group(p.BasePath())
	p.RegisterRoutes(g)
	return r
}

// providerRequest performs an HTTP request against the provider router and returns the recorder.
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

// --- Provider construction ---

func TestNewProvider_Good(t *testing.T) {
	p := NewProvider(nil, nil)
	assert.NotNil(t, p)
	assert.Nil(t, p.bridge)
	assert.Nil(t, p.hub)
}

func TestBrainProvider_Good_Name(t *testing.T) {
	assert.Equal(t, "brain", NewProvider(nil, nil).Name())
}

func TestBrainProvider_Good_BasePath(t *testing.T) {
	assert.Equal(t, "/api/brain", NewProvider(nil, nil).BasePath())
}

func TestBrainProvider_Good_Channels(t *testing.T) {
	channels := NewProvider(nil, nil).Channels()
	assert.Len(t, channels, 3)
	assert.Contains(t, channels, "brain.remember.complete")
	assert.Contains(t, channels, "brain.recall.complete")
	assert.Contains(t, channels, "brain.forget.complete")
}

func TestBrainProvider_Good_Element(t *testing.T) {
	el := NewProvider(nil, nil).Element()
	assert.Equal(t, "core-brain-panel", el.Tag)
	assert.Equal(t, "/assets/brain-panel.js", el.Source)
}

func TestBrainProvider_Good_Describe(t *testing.T) {
	descs := NewProvider(nil, nil).Describe()
	assert.Len(t, descs, 5)

	paths := make([]string, len(descs))
	for i, d := range descs {
		paths[i] = d.Method + " " + d.Path
	}
	assert.Contains(t, paths, "POST /remember")
	assert.Contains(t, paths, "POST /recall")
	assert.Contains(t, paths, "POST /forget")
	assert.Contains(t, paths, "GET /list")
	assert.Contains(t, paths, "GET /status")
}

// --- Handler: status ---

func TestStatus_Good_NilBridge(t *testing.T) {
	p := NewProvider(nil, nil)
	w := providerRequest(t, p, "GET", "/api/brain/status", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data, _ := resp["data"].(map[string]any)
	assert.Equal(t, false, data["connected"])
}

// --- Nil bridge handlers return 503 ---

func TestRememberHandler_Bad_NilBridge(t *testing.T) {
	body, _ := json.Marshal(map[string]any{"content": "test memory", "type": "observation"})
	w := providerRequest(t, NewProvider(nil, nil), "POST", "/api/brain/remember", body)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestRememberHandler_Bad_NilBridgeInvalidBody(t *testing.T) {
	// nil bridge returns 503 before JSON validation.
	w := providerRequest(t, NewProvider(nil, nil), "POST", "/api/brain/remember", []byte("not json"))
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestRecallHandler_Bad_NilBridge(t *testing.T) {
	body, _ := json.Marshal(map[string]any{"query": "test"})
	w := providerRequest(t, NewProvider(nil, nil), "POST", "/api/brain/recall", body)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestForgetHandler_Bad_NilBridge(t *testing.T) {
	body, _ := json.Marshal(map[string]any{"id": "mem-123"})
	w := providerRequest(t, NewProvider(nil, nil), "POST", "/api/brain/forget", body)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestListHandler_Bad_NilBridge(t *testing.T) {
	w := providerRequest(t, NewProvider(nil, nil), "GET", "/api/brain/list", nil)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// --- emitEvent ---

func TestEmitEvent_Good_NilHub(t *testing.T) {
	p := NewProvider(nil, nil)
	p.emitEvent("brain.test", map[string]any{"foo": "bar"})
}
