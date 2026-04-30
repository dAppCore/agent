// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	core "dappco.re/go"
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
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_status_remote",
		Description: "Check workspace status on a remote core-agent (e.g. Charon). Shows running, completed, blocked, and failed agents.",
	}, toolHandlerFor[RemoteStatusInput, RemoteStatusOutput]("statusRemote", "invalid remote status output", s.statusRemote))
}

func (s *PrepSubsystem) statusRemote(ctx context.Context, input RemoteStatusInput) core.Result {
	if input.Host == "" {
		return core.Fail(core.E("statusRemote", "host is required", nil))
	}

	output := RemoteStatusOutput{
		Success: true,
		Host:    input.Host,
	}

	client := NewRemoteClient(input.Host)

	sessionID, err := InitializeRemoteClient(client, ctx)
	if err != nil {
		return core.Ok(RemoteStatusOutput{
			Host:  input.Host,
			Error: core.Concat("unreachable: ", err.Error()),
		})
	}

	result, err := CallRemoteClient(client, ctx, sessionID, client.ToolCallBody(2, "agentic_status", map[string]any{}))
	if err != nil {
		return core.Ok(RemoteStatusOutput{
			Host:  input.Host,
			Error: core.Concat("call failed: ", err.Error()),
		})
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
		return core.Ok(output)
	}
	if rpcResponse.Error != nil {
		output.Success = false
		output.Error = rpcResponse.Error.Message
		return core.Ok(output)
	}
	if len(rpcResponse.Result.Content) > 0 {
		var statusOut StatusOutput
		if r := core.JSONUnmarshalString(rpcResponse.Result.Content[0].Text, &statusOut); r.OK {
			output.Stats = statusOut
		}
	}

	return core.Ok(output)
}
