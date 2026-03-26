// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- agentic_status_remote tool ---

// RemoteStatusInput queries a remote core-agent for workspace status.
//
//	input := agentic.RemoteStatusInput{Host: "charon"}
type RemoteStatusInput struct {
	Host string `json:"host"` // Remote agent host (e.g. "charon")
}

// RemoteStatusOutput is the response from a remote status check.
//
//	out := agentic.RemoteStatusOutput{Success: true, Host: "charon"}
type RemoteStatusOutput struct {
	Success bool         `json:"success"`
	Host    string       `json:"host"`
	Stats   StatusOutput `json:"stats"`
	Error   string       `json:"error,omitempty"`
}

func (s *PrepSubsystem) registerRemoteStatusTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_status_remote",
		Description: "Check workspace status on a remote core-agent (e.g. Charon). Shows running, completed, blocked, and failed agents.",
	}, s.statusRemote)
}

func (s *PrepSubsystem) statusRemote(ctx context.Context, _ *mcp.CallToolRequest, input RemoteStatusInput) (*mcp.CallToolResult, RemoteStatusOutput, error) {
	if input.Host == "" {
		return nil, RemoteStatusOutput{}, core.E("statusRemote", "host is required", nil)
	}

	addr := resolveHost(input.Host)
	token := remoteToken(input.Host)
	url := core.Concat("http://", addr, "/mcp")

	sessionID, err := mcpInitialize(ctx, url, token)
	if err != nil {
		return nil, RemoteStatusOutput{
			Host:  input.Host,
			Error: core.Concat("unreachable: ", err.Error()),
		}, nil
	}

	rpcReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "agentic_status",
			"arguments": map[string]any{},
		},
	}
	body := []byte(core.JSONMarshalString(rpcReq))

	result, err := mcpCall(ctx, url, token, sessionID, body)
	if err != nil {
		return nil, RemoteStatusOutput{
			Host:  input.Host,
			Error: core.Concat("call failed: ", err.Error()),
		}, nil
	}

	output := RemoteStatusOutput{
		Success: true,
		Host:    input.Host,
	}

	var rpcResp struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if r := core.JSONUnmarshal(result, &rpcResp); !r.OK {
		output.Success = false
		output.Error = "failed to parse response"
		return nil, output, nil
	}
	if rpcResp.Error != nil {
		output.Success = false
		output.Error = rpcResp.Error.Message
		return nil, output, nil
	}
	if len(rpcResp.Result.Content) > 0 {
		var statusOut StatusOutput
		if r := core.JSONUnmarshalString(rpcResp.Result.Content[0].Text, &statusOut); r.OK {
			output.Stats = statusOut
		}
	}

	return nil, output, nil
}
