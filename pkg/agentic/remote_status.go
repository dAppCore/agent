// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	core "dappco.re/go/core"
	coremcp "dappco.re/go/mcp/pkg/mcp"
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

func (s *PrepSubsystem) registerRemoteStatusTool(svc *coremcp.Service) {
	mcp.AddTool(svc.Server(), &mcp.Tool{
		Name:        "agentic_status_remote",
		Description: "Check workspace status on a remote core-agent (e.g. Charon). Shows running, completed, blocked, and failed agents.",
	}, s.statusRemote)
}

func (s *PrepSubsystem) statusRemote(ctx context.Context, _ *mcp.CallToolRequest, input RemoteStatusInput) (*mcp.CallToolResult, RemoteStatusOutput, error) {
	if input.Host == "" {
		return nil, RemoteStatusOutput{}, core.E("statusRemote", "host is required", nil)
	}

	output := RemoteStatusOutput{
		Success: true,
		Host:    input.Host,
	}

	client := NewRemoteClient(input.Host)

	sessionID, err := client.Initialize(ctx)
	if err != nil {
		return nil, RemoteStatusOutput{
			Host:  input.Host,
			Error: core.Concat("unreachable: ", err.Error()),
		}, nil
	}

	result, err := client.Call(ctx, sessionID, client.ToolCallBody(2, "agentic_status", map[string]any{}))
	if err != nil {
		return nil, RemoteStatusOutput{
			Host:  input.Host,
			Error: core.Concat("call failed: ", err.Error()),
		}, nil
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
