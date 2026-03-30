// SPDX-License-Identifier: EUPL-1.2

// HTTP transport for Core API streams.
// This is the ONE file in core/agent that imports net/http.
// All other files use the exported helpers: HTTPGet, HTTPPost, HTTPCall.

package agentic

import (
	"context"
	"net/http"
	"time"

	core "dappco.re/go/core"
)

// defaultClient is the shared HTTP client for all transport calls.
var defaultClient = &http.Client{Timeout: 30 * time.Second}

// httpStream implements core.Stream over HTTP request/response.
type httpStream struct {
	client   *http.Client
	url      string
	token    string
	method   string
	response []byte
}

// Send issues the configured HTTP request and caches the response body for Receive.
//
//	stream := &httpStream{client: defaultClient, url: "https://forge.lthn.ai/api/v1/version", method: "GET"}
//	_ = stream.Send(nil)
func (s *httpStream) Send(data []byte) error {
	req, err := http.NewRequestWithContext(context.Background(), s.method, s.url, core.NewReader(string(data)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if s.token != "" {
		req.Header.Set("Authorization", core.Concat("token ", s.token))
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}

	r := core.ReadAll(resp.Body)
	if !r.OK {
		err, _ := r.Value.(error)
		return core.E("httpStream.Send", "failed to read response", err)
	}
	s.response = []byte(r.Value.(string))
	return nil
}

// Receive returns the cached response body from the last Send call.
//
//	stream := &httpStream{response: []byte(`{"ok":true}`)}
//	data, _ := stream.Receive()
//	_ = data
func (s *httpStream) Receive() ([]byte, error) {
	return s.response, nil
}

// Close satisfies core.Stream for one-shot HTTP requests.
//
//	stream := &httpStream{}
//	_ = stream.Close()
func (s *httpStream) Close() error {
	return nil
}

// RegisterHTTPTransport registers the HTTP/HTTPS protocol handler with Core API.
//
//	agentic.RegisterHTTPTransport(c)
func RegisterHTTPTransport(c *core.Core) {
	factory := func(handle *core.DriveHandle) (core.Stream, error) {
		token := handle.Options.String("token")
		return &httpStream{
			client: defaultClient,
			url:    handle.Transport,
			token:  token,
			method: "POST",
		}, nil
	}
	c.API().RegisterProtocol("http", factory)
	c.API().RegisterProtocol("https", factory)
}

// --- REST helpers — all HTTP in core/agent routes through these ---

// HTTPGet performs a GET request. Returns Result{Value: string (response body), OK: bool}.
// Auth is "token {token}" for Forge, "Bearer {token}" for Brain.
//
//	r := agentic.HTTPGet(ctx, "https://forge.lthn.ai/api/v1/repos", "my-token", "token")
func HTTPGet(ctx context.Context, url, token, authScheme string) core.Result {
	return httpDo(ctx, "GET", url, "", token, authScheme)
}

// HTTPPost performs a POST request with a JSON body. Returns Result{Value: string, OK: bool}.
//
//	r := agentic.HTTPPost(ctx, url, core.JSONMarshalString(payload), token, "token")
func HTTPPost(ctx context.Context, url, body, token, authScheme string) core.Result {
	return httpDo(ctx, "POST", url, body, token, authScheme)
}

// HTTPPatch performs a PATCH request with a JSON body.
//
//	r := agentic.HTTPPatch(ctx, url, body, token, "token")
func HTTPPatch(ctx context.Context, url, body, token, authScheme string) core.Result {
	return httpDo(ctx, "PATCH", url, body, token, authScheme)
}

// HTTPDelete performs a DELETE request.
//
//	r := agentic.HTTPDelete(ctx, url, body, token, "Bearer")
func HTTPDelete(ctx context.Context, url, body, token, authScheme string) core.Result {
	return httpDo(ctx, "DELETE", url, body, token, authScheme)
}

// HTTPDo performs an HTTP request with the specified method.
//
//	r := agentic.HTTPDo(ctx, "PUT", url, body, token, "token")
func HTTPDo(ctx context.Context, method, url, body, token, authScheme string) core.Result {
	return httpDo(ctx, method, url, body, token, authScheme)
}

// --- Drive-aware REST helpers — route through c.Drive() for endpoint resolution ---

// DriveGet performs a GET request using a named Drive endpoint.
// Reads base URL and token from the Drive handle registered in Core.
//
//	r := DriveGet(c, "forge", "/api/v1/repos/core/go-io", "token")
func DriveGet(c *core.Core, drive, path, authScheme string) core.Result {
	base, token := driveEndpoint(c, drive)
	if base == "" {
		return core.Result{Value: core.E("DriveGet", core.Concat("drive not found: ", drive), nil), OK: false}
	}
	return httpDo(context.Background(), "GET", core.Concat(base, path), "", token, authScheme)
}

// DrivePost performs a POST request using a named Drive endpoint.
//
//	r := DrivePost(c, "forge", "/api/v1/repos/core/go-io/issues", body, "token")
func DrivePost(c *core.Core, drive, path, body, authScheme string) core.Result {
	base, token := driveEndpoint(c, drive)
	if base == "" {
		return core.Result{Value: core.E("DrivePost", core.Concat("drive not found: ", drive), nil), OK: false}
	}
	return httpDo(context.Background(), "POST", core.Concat(base, path), body, token, authScheme)
}

// DriveDo performs an HTTP request using a named Drive endpoint.
//
//	r := DriveDo(c, "forge", "PATCH", "/api/v1/repos/core/go-io/pulls/5", body, "token")
func DriveDo(c *core.Core, drive, method, path, body, authScheme string) core.Result {
	base, token := driveEndpoint(c, drive)
	if base == "" {
		return core.Result{Value: core.E("DriveDo", core.Concat("drive not found: ", drive), nil), OK: false}
	}
	return httpDo(context.Background(), method, core.Concat(base, path), body, token, authScheme)
}

// driveEndpoint reads base URL and token from a named Drive handle.
func driveEndpoint(c *core.Core, name string) (base, token string) {
	r := c.Drive().Get(name)
	if !r.OK {
		return "", ""
	}
	h := r.Value.(*core.DriveHandle)
	return h.Transport, h.Options.String("token")
}

// httpDo is the single HTTP execution point. Every HTTP call in core/agent routes here.
func httpDo(ctx context.Context, method, url, body, token, authScheme string) core.Result {
	var req *http.Request
	var err error

	if body != "" {
		req, err = http.NewRequestWithContext(ctx, method, url, core.NewReader(body))
	} else {
		req, err = http.NewRequestWithContext(ctx, method, url, nil)
	}
	if err != nil {
		return core.Result{Value: core.E("httpDo", "create request", err), OK: false}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if token != "" {
		if authScheme == "" {
			authScheme = "token"
		}
		req.Header.Set("Authorization", core.Concat(authScheme, " ", token))
	}

	resp, err := defaultClient.Do(req)
	if err != nil {
		return core.Result{Value: core.E("httpDo", "request failed", err), OK: false}
	}

	r := core.ReadAll(resp.Body)
	if !r.OK {
		err, _ := r.Value.(error)
		return core.Result{Value: core.E("httpDo", "failed to read response", err), OK: false}
	}

	return core.Result{Value: r.Value.(string), OK: resp.StatusCode < 400}
}

// --- MCP Streamable HTTP Transport ---

// mcpInitialize performs the MCP initialise handshake over Streamable HTTP.
// Returns the session ID from the Mcp-Session-Id header.
func mcpInitialize(ctx context.Context, url, token string) (string, error) {
	result := mcpInitializeResult(ctx, url, token)
	if !result.OK {
		err, _ := result.Value.(error)
		if err == nil {
			return "", core.E("mcpInitialize", "failed", nil)
		}
		return "", err
	}
	sessionID, ok := result.Value.(string)
	if !ok {
		return "", core.E("mcpInitialize", "invalid session id result", nil)
	}
	return sessionID, nil
}

func mcpInitializeResult(ctx context.Context, url, token string) core.Result {
	initReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-03-26",
			"capabilities":    map[string]any{},
			"clientInfo": map[string]any{
				"name":    "core-agent-remote",
				"version": "0.2.0",
			},
		},
	}

	body := core.JSONMarshalString(initReq)
	req, err := http.NewRequestWithContext(ctx, "POST", url, core.NewReader(body))
	if err != nil {
		return core.Result{Value: core.E("mcpInitialize", "create request", err), OK: false}
	}
	mcpHeaders(req, token, "")

	resp, err := defaultClient.Do(req)
	if err != nil {
		return core.Result{Value: core.E("mcpInitialize", "request failed", err), OK: false}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return core.Result{Value: core.E("mcpInitialize", core.Sprintf("HTTP %d", resp.StatusCode), nil), OK: false}
	}

	sessionID := resp.Header.Get("Mcp-Session-Id")

	// Drain SSE response
	drainSSE(resp)

	// Send initialised notification
	notif := core.JSONMarshalString(map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	})
	notifReq, err := http.NewRequestWithContext(ctx, "POST", url, core.NewReader(notif))
	if err != nil {
		return core.Result{Value: core.E("mcpInitialize", "create notification request", err), OK: false}
	}
	mcpHeaders(notifReq, token, sessionID)
	notifResp, err := defaultClient.Do(notifReq)
	if err == nil {
		notifResp.Body.Close()
	}

	return core.Result{Value: sessionID, OK: true}
}

// mcpCall sends a JSON-RPC request and returns the parsed response.
func mcpCall(ctx context.Context, url, token, sessionID string, body []byte) ([]byte, error) {
	result := mcpCallResult(ctx, url, token, sessionID, body)
	if !result.OK {
		err, _ := result.Value.(error)
		if err == nil {
			return nil, core.E("mcpCall", "failed", nil)
		}
		return nil, err
	}
	data, ok := result.Value.([]byte)
	if !ok {
		return nil, core.E("mcpCall", "invalid call result", nil)
	}
	return data, nil
}

func mcpCallResult(ctx context.Context, url, token, sessionID string, body []byte) core.Result {
	req, err := http.NewRequestWithContext(ctx, "POST", url, core.NewReader(string(body)))
	if err != nil {
		return core.Result{Value: core.E("mcpCall", "create request", err), OK: false}
	}
	mcpHeaders(req, token, sessionID)

	resp, err := defaultClient.Do(req)
	if err != nil {
		return core.Result{Value: core.E("mcpCall", "request failed", err), OK: false}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return core.Result{Value: core.E("mcpCall", core.Sprintf("HTTP %d", resp.StatusCode), nil), OK: false}
	}

	return readSSEDataResult(resp)
}

// readSSEData reads an SSE response and extracts JSON from data: lines.
func readSSEData(resp *http.Response) ([]byte, error) {
	result := readSSEDataResult(resp)
	if !result.OK {
		err, _ := result.Value.(error)
		if err == nil {
			return nil, core.E("readSSEData", "failed", nil)
		}
		return nil, err
	}
	data, ok := result.Value.([]byte)
	if !ok {
		return nil, core.E("readSSEData", "invalid data result", nil)
	}
	return data, nil
}

// readSSEDataResult parses an SSE response and extracts the first data: payload as core.Result.
func readSSEDataResult(resp *http.Response) core.Result {
	r := core.ReadAll(resp.Body)
	if !r.OK {
		err, _ := r.Value.(error)
		return core.Result{Value: core.E("readSSEData", "failed to read response", err), OK: false}
	}
	for _, line := range core.Split(r.Value.(string), "\n") {
		if core.HasPrefix(line, "data: ") {
			return core.Result{Value: []byte(core.TrimPrefix(line, "data: ")), OK: true}
		}
	}
	return core.Result{Value: core.E("readSSEData", "no data in SSE response", nil), OK: false}
}

// mcpHeaders applies standard MCP HTTP headers.
func mcpHeaders(req *http.Request, token, sessionID string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if token != "" {
		req.Header.Set("Authorization", core.Concat("Bearer ", token))
	}
	if sessionID != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
	}
}

// drainSSE reads and discards an SSE response body.
func drainSSE(resp *http.Response) {
	core.ReadAll(resp.Body)
}
