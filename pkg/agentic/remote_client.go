// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go/core"
)

// client := agentic.NewRemoteClient("charon")
// core.Println(client.URL) // "http://10.69.69.165:9101/mcp"
type RemoteClient struct {
	Host    string
	Address string
	Token   string
	URL     string
}

// client := agentic.NewRemoteClient("charon")
func NewRemoteClient(host string) RemoteClient {
	address := resolveHost(host)
	return RemoteClient{
		Host:    host,
		Address: address,
		Token:   remoteToken(host),
		URL:     core.Sprintf("http://%s/mcp", address),
	}
}

// sessionID, err := client.Initialize(context.Background())
func (c RemoteClient) Initialize(ctx context.Context) (string, error) {
	result := mcpInitializeResult(ctx, c.URL, c.Token)
	if !result.OK {
		err, _ := result.Value.(error)
		if err == nil {
			err = core.E("remoteClient.Initialize", "MCP initialise failed", nil)
		}
		return "", err
	}
	sessionID, ok := result.Value.(string)
	if !ok || sessionID == "" {
		return "", core.E("remoteClient.Initialize", "invalid session id", nil)
	}
	return sessionID, nil
}

// response, err := client.Call(context.Background(), "session-1", core.JSONMarshalString(map[string]any{"method":"tools/call"}))
func (c RemoteClient) Call(ctx context.Context, sessionID string, body []byte) ([]byte, error) {
	result := mcpCallResult(ctx, c.URL, c.Token, sessionID, body)
	if !result.OK {
		err, _ := result.Value.(error)
		if err == nil {
			err = core.E("remoteClient.Call", "tool call failed", nil)
		}
		return nil, err
	}
	response, ok := result.Value.([]byte)
	if !ok {
		return nil, core.E("remoteClient.Call", "invalid response payload", nil)
	}
	return response, nil
}

// body := client.ToolCallBody(1, "agentic_dispatch", map[string]any{"repo": "go-io", "task": "Fix tests"})
func (c RemoteClient) ToolCallBody(id int, name string, arguments map[string]any) []byte {
	request := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      name,
			"arguments": arguments,
		},
	}
	return []byte(core.JSONMarshalString(request))
}
