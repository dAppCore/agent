// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type RemoteStatusInput struct {
	Host string `json:"host"`
}

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

	sessionResult := mcpInitializeResult(ctx, url, token)
	if !sessionResult.OK {
		err, _ := sessionResult.Value.(error)
		if err == nil {
			err = core.E("statusRemote", "MCP initialize failed", nil)
		}
		return nil, RemoteStatusOutput{
			Host:  input.Host,
			Error: core.Concat("unreachable: ", err.Error()),
		}, nil
	}
	sessionID, ok := sessionResult.Value.(string)
	if !ok || sessionID == "" {
		err := core.E("statusRemote", "invalid session id", nil)
		return nil, RemoteStatusOutput{
			Host:  input.Host,
			Error: core.Concat("unreachable: ", err.Error()),
		}, nil
	}

	rpcRequest := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "agentic_status",
			"arguments": map[string]any{},
		},
	}
	body := []byte(core.JSONMarshalString(rpcRequest))

	callResult := mcpCallResult(ctx, url, token, sessionID, body)
	if !callResult.OK {
		err, _ := callResult.Value.(error)
		if err == nil {
			err = core.E("statusRemote", "tool call failed", nil)
		}
		return nil, RemoteStatusOutput{
			Host:  input.Host,
			Error: core.Concat("call failed: ", err.Error()),
		}, nil
	}
	result, ok := callResult.Value.([]byte)
	if !ok {
		err := core.E("statusRemote", "invalid tool response", nil)
		return nil, RemoteStatusOutput{
			Host:  input.Host,
			Error: core.Concat("call failed: ", err.Error()),
		}, nil
	}

	output := RemoteStatusOutput{
		Success: true,
		Host:    input.Host,
	}

	var rpcResponse struct {
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
	if r := core.JSONUnmarshal(result, &rpcResponse); !r.OK {
		output.Success = false
		output.Error = "failed to parse response"
		return nil, output, nil
	}
	if rpcResponse.Error != nil {
		output.Success = false
		output.Error = rpcResponse.Error.Message
		return nil, output, nil
	}
	if len(rpcResponse.Result.Content) > 0 {
		var statusOut StatusOutput
		if r := core.JSONUnmarshalString(rpcResponse.Result.Content[0].Text, &statusOut); r.OK {
			output.Stats = statusOut
		}
	}

	return nil, output, nil
}
