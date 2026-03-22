// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	coreerr "dappco.re/go/core/log"
)

// mcpInitialize performs the MCP initialize handshake over Streamable HTTP.
// Returns the session ID from the Mcp-Session-Id header.
func mcpInitialize(ctx context.Context, client *http.Client, url, token string) (string, error) {
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

	body, _ := json.Marshal(initReq)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", coreerr.E("mcpInitialize", "create request", err)
	}
	setHeaders(req, token, "")

	resp, err := client.Do(req)
	if err != nil {
		return "", coreerr.E("mcpInitialize", "request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", coreerr.E("mcpInitialize", fmt.Sprintf("HTTP %d", resp.StatusCode), nil)
	}

	sessionID := resp.Header.Get("Mcp-Session-Id")

	// Drain the SSE response (we don't need the initialize result)
	drainSSE(resp)

	// Send initialized notification
	notif := map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	}
	notifBody, _ := json.Marshal(notif)

	notifReq, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(notifBody))
	setHeaders(notifReq, token, sessionID)

	notifResp, err := client.Do(notifReq)
	if err == nil {
		notifResp.Body.Close()
	}

	return sessionID, nil
}

// mcpCall sends a JSON-RPC request and returns the parsed response.
// Handles the SSE response format (text/event-stream with data: lines).
func mcpCall(ctx context.Context, client *http.Client, url, token, sessionID string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, coreerr.E("mcpCall", "create request", err)
	}
	setHeaders(req, token, sessionID)

	resp, err := client.Do(req)
	if err != nil {
		return nil, coreerr.E("mcpCall", "request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, coreerr.E("mcpCall", fmt.Sprintf("HTTP %d", resp.StatusCode), nil)
	}

	// Parse SSE response — extract data: lines
	return readSSEData(resp)
}

// readSSEData reads an SSE response and extracts the JSON from data: lines.
func readSSEData(resp *http.Response) ([]byte, error) {
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			return []byte(strings.TrimPrefix(line, "data: ")), nil
		}
	}
	return nil, coreerr.E("readSSEData", "no data in SSE response", nil)
}

// setHeaders applies standard MCP HTTP headers.
func setHeaders(req *http.Request, token, sessionID string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if sessionID != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
	}
}

// drainSSE reads and discards an SSE response body.
func drainSSE(resp *http.Response) {
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		// Discard
	}
}
