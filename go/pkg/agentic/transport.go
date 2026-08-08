// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http" // Note: AX-6 — structural HTTP transport boundary for core.API protocol streams and raw MCP POST/SSE exchange; no exported core/api generic request wrapper covers this file.
	"time"

	core "dappco.re/go"
	agentcompat "dappco.re/go/agent/pkg/agentcompat"
)

// defaultClient centralises the low-level HTTP boundary owned by this file.
// Higher layers should go through core.Drive/core.API; this transport package
// is the intrinsic implementation behind those abstractions.
var defaultClient = &http.Client{Timeout: 30 * time.Second}

type httpStream = agentcompat.HTTPStream

// agentic.RegisterHTTPTransport(c)
func RegisterHTTPTransport(c *core.Core) {
	factory := func(handle *core.DriveHandle) (core.Stream, error) {
		token := handle.Options.String("token")
		return &httpStream{
			Client: defaultClient,
			URL:    handle.Transport,
			Token:  token,
			Method: "POST",
		}, nil
	}
	c.API().RegisterProtocol("http", factory)
	c.API().RegisterProtocol("https", factory)
}

// result := agentic.HTTPGet(ctx, "https://forge.lthn.ai/api/v1/repos", "my-token", "token")
func HTTPGet(ctx context.Context, url, token, authScheme string) core.Result {
	return httpDo(ctx, "GET", url, "", token, authScheme)
}

// result := agentic.HTTPPost(ctx, url, core.JSONMarshalString(payload), token, "token")
func HTTPPost(ctx context.Context, url, body, token, authScheme string) core.Result {
	return httpDo(ctx, "POST", url, body, token, authScheme)
}

// result := agentic.HTTPPatch(ctx, url, body, token, "token")
func HTTPPatch(ctx context.Context, url, body, token, authScheme string) core.Result {
	return httpDo(ctx, "PATCH", url, body, token, authScheme)
}

// result := agentic.HTTPDelete(ctx, url, body, token, "Bearer")
func HTTPDelete(ctx context.Context, url, body, token, authScheme string) core.Result {
	return httpDo(ctx, "DELETE", url, body, token, authScheme)
}

// result := agentic.HTTPDo(ctx, "PUT", url, body, token, "token")
func HTTPDo(ctx context.Context, method, url, body, token, authScheme string) core.Result {
	return httpDo(ctx, method, url, body, token, authScheme)
}

// result := DriveGet(c, "forge", "/api/v1/repos/core/go-io", "token")
func DriveGet(c *core.Core, drive, path, authScheme string) core.Result {
	base, token := driveEndpoint(c, drive)
	if base == "" {
		return core.Result{Value: core.E("DriveGet", core.Concat("drive not found: ", drive), nil), OK: false}
	}
	return httpDo(context.Background(), "GET", core.Concat(base, path), "", token, authScheme)
}

// result := DrivePost(c, "forge", "/api/v1/repos/core/go-io/issues", body, "token")
func DrivePost(c *core.Core, drive, path, body, authScheme string) core.Result {
	base, token := driveEndpoint(c, drive)
	if base == "" {
		return core.Result{Value: core.E("DrivePost", core.Concat("drive not found: ", drive), nil), OK: false}
	}
	return httpDo(context.Background(), "POST", core.Concat(base, path), body, token, authScheme)
}

// result := DriveDo(c, "forge", "PATCH", "/api/v1/repos/core/go-io/pulls/5", body, "token")
func DriveDo(c *core.Core, drive, method, path, body, authScheme string) core.Result {
	base, token := driveEndpoint(c, drive)
	if base == "" {
		return core.Result{Value: core.E("DriveDo", core.Concat("drive not found: ", drive), nil), OK: false}
	}
	return httpDo(context.Background(), method, core.Concat(base, path), body, token, authScheme)
}

func driveEndpoint(c *core.Core, name string) (base, token string) {
	driveResult := c.Drive().Get(name)
	if !driveResult.OK {
		return "", ""
	}
	driveHandle := driveResult.Value.(*core.DriveHandle)
	return driveHandle.Transport, driveHandle.Options.String("token")
}

func httpDo(ctx context.Context, method, url, body, token, authScheme string) core.Result {
	var request *http.Request
	var requestErr error

	if body != "" {
		request, requestErr = http.NewRequestWithContext(ctx, method, url, core.NewReader(body))
	} else {
		request, requestErr = http.NewRequestWithContext(ctx, method, url, nil)
	}
	if requestErr != nil {
		return core.Result{Value: core.E("httpDo", "create request", requestErr), OK: false}
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	if token != "" {
		if authScheme == "" {
			authScheme = "token"
		}
		request.Header.Set("Authorization", core.Concat(authScheme, " ", token))
	}

	response, requestErr := defaultClient.Do(request)
	if requestErr != nil {
		return core.Result{Value: core.E("httpDo", "request failed", requestErr), OK: false}
	}
	defer func() { _ = response.Body.Close() }() // read side of the response

	readResult := core.ReadAll(response.Body)
	if !readResult.OK {
		readErr, _ := readResult.Value.(error)
		return core.Result{Value: core.E("httpDo", "failed to read response", readErr), OK: false}
	}

	return core.Result{Value: readResult.Value.(string), OK: response.StatusCode < 400}
}

// sessionID, err := mcpInitialize(ctx, url, token)
var mcpInitialize = func(ctx context.Context, url, token string) (string, error) {
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
	initializeRequest := map[string]any{
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

	body := core.JSONMarshalString(initializeRequest)
	request, err := http.NewRequestWithContext(ctx, "POST", url, core.NewReader(body))
	if err != nil {
		return core.Result{Value: core.E("mcpInitialize", "create request", err), OK: false}
	}
	mcpHeaders(request, token, "")

	response, err := defaultClient.Do(request)
	if err != nil {
		return core.Result{Value: core.E("mcpInitialize", "request failed", err), OK: false}
	}
	defer func() { _ = response.Body.Close() }() // read side of the response

	if response.StatusCode != 200 {
		return core.Result{Value: core.E("mcpInitialize", core.Sprintf("HTTP %d", response.StatusCode), nil), OK: false}
	}

	sessionID := response.Header.Get("Mcp-Session-Id")
	if sessionID == "" {
		return core.Result{Value: core.E("mcpInitialize", "missing session id", nil), OK: false}
	}

	drainSSE(response)

	notification := core.JSONMarshalString(map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	})
	notificationRequest, err := http.NewRequestWithContext(ctx, "POST", url, core.NewReader(notification))
	if err != nil {
		return core.Result{Value: core.E("mcpInitialize", "create notification request", err), OK: false}
	}
	mcpHeaders(notificationRequest, token, sessionID)
	notificationResponse, err := defaultClient.Do(notificationRequest)
	if err == nil {
		// Read side of the response.
		_ = notificationResponse.Body.Close()
	}

	return core.Result{Value: sessionID, OK: true}
}

var mcpCall = func(ctx context.Context, url, token, sessionID string, body []byte) ([]byte, error) {
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
	request, err := http.NewRequestWithContext(ctx, "POST", url, core.NewReader(string(body)))
	if err != nil {
		return core.Result{Value: core.E("mcpCall", "create request", err), OK: false}
	}
	mcpHeaders(request, token, sessionID)

	response, err := defaultClient.Do(request)
	if err != nil {
		return core.Result{Value: core.E("mcpCall", "request failed", err), OK: false}
	}
	defer func() { _ = response.Body.Close() }() // read side of the response

	if response.StatusCode != 200 {
		return core.Result{Value: core.E("mcpCall", core.Sprintf("HTTP %d", response.StatusCode), nil), OK: false}
	}

	return readSSEDataResult(response)
}

var readSSEData = func(response *http.Response) ([]byte, error) {
	result := readSSEDataResult(response)
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

func readSSEDataResult(response *http.Response) core.Result {
	readResult := core.ReadAll(response.Body)
	if !readResult.OK {
		err, _ := readResult.Value.(error)
		return core.Result{Value: core.E("readSSEData", "failed to read response", err), OK: false}
	}
	for _, line := range core.Split(readResult.Value.(string), "\n") {
		if core.HasPrefix(line, "data: ") {
			return core.Result{Value: []byte(core.TrimPrefix(line, "data: ")), OK: true}
		}
	}
	return core.Result{Value: core.E("readSSEData", "no data in SSE response", nil), OK: false}
}

func mcpHeaders(request *http.Request, token, sessionID string) {
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	if token != "" {
		request.Header.Set("Authorization", core.Concat("Bearer ", token))
	}
	if sessionID != "" {
		request.Header.Set("Mcp-Session-Id", sessionID)
	}
}

func drainSSE(response *http.Response) {
	// Drained purely so the connection can be reused — the body is unwanted and
	// a read failure here changes nothing.
	_ = core.ReadAll(response.Body)
}
