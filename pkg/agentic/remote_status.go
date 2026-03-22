// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	coreerr "forge.lthn.ai/core/go-log"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- agentic_status_remote tool ---

// RemoteStatusInput queries a remote core-agent for workspace status.
type RemoteStatusInput struct {
	Host string `json:"host"` // Remote agent host (e.g. "charon")
}

// RemoteStatusOutput is the response from a remote status check.
type RemoteStatusOutput struct {
	Success    bool            `json:"success"`
	Host       string          `json:"host"`
	Workspaces []WorkspaceInfo `json:"workspaces"`
	Count      int             `json:"count"`
	Error      string          `json:"error,omitempty"`
}

func (s *PrepSubsystem) registerRemoteStatusTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_status_remote",
		Description: "Check workspace status on a remote core-agent (e.g. Charon). Shows running, completed, blocked, and failed agents.",
	}, s.statusRemote)
}

func (s *PrepSubsystem) statusRemote(ctx context.Context, _ *mcp.CallToolRequest, input RemoteStatusInput) (*mcp.CallToolResult, RemoteStatusOutput, error) {
	if input.Host == "" {
		return nil, RemoteStatusOutput{}, coreerr.E("statusRemote", "host is required", nil)
	}

	addr := resolveHost(input.Host)
	token := remoteToken(input.Host)
	url := "http://" + addr + "/mcp"

	client := &http.Client{Timeout: 15 * time.Second}

	sessionID, err := mcpInitialize(ctx, client, url, token)
	if err != nil {
		return nil, RemoteStatusOutput{
			Host:  input.Host,
			Error: "unreachable: " + err.Error(),
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
	body, _ := json.Marshal(rpcReq)

	result, err := mcpCall(ctx, client, url, token, sessionID, body)
	if err != nil {
		return nil, RemoteStatusOutput{
			Host:  input.Host,
			Error: "call failed: " + err.Error(),
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
	if json.Unmarshal(result, &rpcResp) != nil {
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
		if json.Unmarshal([]byte(rpcResp.Result.Content[0].Text), &statusOut) == nil {
			output.Workspaces = statusOut.Workspaces
			output.Count = statusOut.Count
		}
	}

	return nil, output, nil
}
