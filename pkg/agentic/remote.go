// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	coreio "forge.lthn.ai/core/go-io"
	coreerr "forge.lthn.ai/core/go-log"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- agentic_dispatch_remote tool ---

// RemoteDispatchInput dispatches a task to a remote core-agent over HTTP.
type RemoteDispatchInput struct {
	Host     string            `json:"host"`               // Remote agent host (e.g. "charon", "10.69.69.165:9101")
	Repo     string            `json:"repo"`               // Target repo
	Task     string            `json:"task"`               // What the agent should do
	Agent    string            `json:"agent,omitempty"`     // Agent type (default: claude:opus)
	Template string            `json:"template,omitempty"`  // Prompt template
	Persona  string            `json:"persona,omitempty"`   // Persona slug
	Org      string            `json:"org,omitempty"`       // Forge org (default: core)
	Variables map[string]string `json:"variables,omitempty"` // Template variables
}

// RemoteDispatchOutput is the response from a remote dispatch.
type RemoteDispatchOutput struct {
	Success      bool   `json:"success"`
	Host         string `json:"host"`
	Repo         string `json:"repo"`
	Agent        string `json:"agent"`
	WorkspaceDir string `json:"workspace_dir,omitempty"`
	PID          int    `json:"pid,omitempty"`
	Error        string `json:"error,omitempty"`
}

func (s *PrepSubsystem) registerRemoteDispatchTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_dispatch_remote",
		Description: "Dispatch a task to a remote core-agent (e.g. Charon). The remote agent preps a workspace and spawns the task locally on its hardware.",
	}, s.dispatchRemote)
}

func (s *PrepSubsystem) dispatchRemote(ctx context.Context, _ *mcp.CallToolRequest, input RemoteDispatchInput) (*mcp.CallToolResult, RemoteDispatchOutput, error) {
	if input.Host == "" {
		return nil, RemoteDispatchOutput{}, coreerr.E("dispatchRemote", "host is required", nil)
	}
	if input.Repo == "" {
		return nil, RemoteDispatchOutput{}, coreerr.E("dispatchRemote", "repo is required", nil)
	}
	if input.Task == "" {
		return nil, RemoteDispatchOutput{}, coreerr.E("dispatchRemote", "task is required", nil)
	}

	// Resolve host aliases
	addr := resolveHost(input.Host)

	// Get auth token for remote agent
	token := remoteToken(input.Host)

	// Build the MCP JSON-RPC call to agentic_dispatch on the remote
	callParams := map[string]any{
		"repo": input.Repo,
		"task": input.Task,
	}
	if input.Agent != "" {
		callParams["agent"] = input.Agent
	}
	if input.Template != "" {
		callParams["template"] = input.Template
	}
	if input.Persona != "" {
		callParams["persona"] = input.Persona
	}
	if input.Org != "" {
		callParams["org"] = input.Org
	}
	if len(input.Variables) > 0 {
		callParams["variables"] = input.Variables
	}

	rpcReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "agentic_dispatch",
			"arguments": callParams,
		},
	}

	url := fmt.Sprintf("http://%s/mcp", addr)
	client := &http.Client{Timeout: 30 * time.Second}

	// Step 1: Initialize session
	sessionID, err := mcpInitialize(ctx, client, url, token)
	if err != nil {
		return nil, RemoteDispatchOutput{
			Host:  input.Host,
			Error: fmt.Sprintf("init failed: %v", err),
		}, coreerr.E("dispatchRemote", "MCP initialize failed", err)
	}

	// Step 2: Call the tool
	body, _ := json.Marshal(rpcReq)
	result, err := mcpCall(ctx, client, url, token, sessionID, body)
	if err != nil {
		return nil, RemoteDispatchOutput{
			Host:  input.Host,
			Error: fmt.Sprintf("call failed: %v", err),
		}, coreerr.E("dispatchRemote", "tool call failed", err)
	}

	// Parse result
	output := RemoteDispatchOutput{
		Success: true,
		Host:    input.Host,
		Repo:    input.Repo,
		Agent:   input.Agent,
	}

	var rpcResp struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(result, &rpcResp) == nil {
		if rpcResp.Error != nil {
			output.Success = false
			output.Error = rpcResp.Error.Message
		} else if len(rpcResp.Result.Content) > 0 {
			var dispatchOut DispatchOutput
			if json.Unmarshal([]byte(rpcResp.Result.Content[0].Text), &dispatchOut) == nil {
				output.WorkspaceDir = dispatchOut.WorkspaceDir
				output.PID = dispatchOut.PID
				output.Agent = dispatchOut.Agent
			}
		}
	}

	return nil, output, nil
}

// resolveHost maps friendly names to addresses.
func resolveHost(host string) string {
	// Known hosts
	aliases := map[string]string{
		"charon":  "10.69.69.165:9101",
		"cladius": "127.0.0.1:9101",
		"local":   "127.0.0.1:9101",
	}

	if addr, ok := aliases[strings.ToLower(host)]; ok {
		return addr
	}

	// If no port specified, add default
	if !strings.Contains(host, ":") {
		return host + ":9101"
	}

	return host
}

// remoteToken gets the auth token for a remote agent.
func remoteToken(host string) string {
	// Check environment first
	envKey := fmt.Sprintf("AGENT_TOKEN_%s", strings.ToUpper(host))
	if token := os.Getenv(envKey); token != "" {
		return token
	}

	// Fallback to shared agent token
	if token := os.Getenv("MCP_AUTH_TOKEN"); token != "" {
		return token
	}

	// Try reading from file
	home, _ := os.UserHomeDir()
	tokenFiles := []string{
		fmt.Sprintf("%s/.core/tokens/%s.token", home, strings.ToLower(host)),
		fmt.Sprintf("%s/.core/agent-token", home),
	}
	for _, f := range tokenFiles {
		if data, err := coreio.Local.Read(f); err == nil {
			return strings.TrimSpace(data)
		}
	}

	return ""
}
