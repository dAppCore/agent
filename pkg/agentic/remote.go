// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- agentic_dispatch_remote tool ---

// RemoteDispatchInput dispatches a task to a remote core-agent over HTTP.
//
//	input := agentic.RemoteDispatchInput{Host: "charon", Repo: "go-io", Task: "Run the review queue"}
type RemoteDispatchInput struct {
	Host      string            `json:"host"`                // Remote agent host (e.g. "charon", "10.69.69.165:9101")
	Repo      string            `json:"repo"`                // Target repo
	Task      string            `json:"task"`                // What the agent should do
	Agent     string            `json:"agent,omitempty"`     // Agent type (default: claude:opus)
	Template  string            `json:"template,omitempty"`  // Prompt template
	Persona   string            `json:"persona,omitempty"`   // Persona slug
	Org       string            `json:"org,omitempty"`       // Forge org (default: core)
	Variables map[string]string `json:"variables,omitempty"` // Template variables
}

// RemoteDispatchOutput is the response from a remote dispatch.
//
//	out := agentic.RemoteDispatchOutput{Success: true, Host: "charon", Repo: "go-io", Agent: "claude:opus"}
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
		return nil, RemoteDispatchOutput{}, core.E("dispatchRemote", "host is required", nil)
	}
	if input.Repo == "" {
		return nil, RemoteDispatchOutput{}, core.E("dispatchRemote", "repo is required", nil)
	}
	if input.Task == "" {
		return nil, RemoteDispatchOutput{}, core.E("dispatchRemote", "task is required", nil)
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

	rpcRequest := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "agentic_dispatch",
			"arguments": callParams,
		},
	}
	body := []byte(core.JSONMarshalString(rpcRequest))

	url := core.Sprintf("http://%s/mcp", addr)

	// Step 1: Initialize session
	sessionResult := mcpInitializeResult(ctx, url, token)
	if !sessionResult.OK {
		err, _ := sessionResult.Value.(error)
		if err == nil {
			err = core.E("dispatchRemote", "MCP initialize failed", nil)
		}
		return nil, RemoteDispatchOutput{
			Host:  input.Host,
			Error: core.Sprintf("init failed: %v", err),
		}, core.E("dispatchRemote", "MCP initialize failed", err)
	}
	sessionID, ok := sessionResult.Value.(string)
	if !ok || sessionID == "" {
		err := core.E("dispatchRemote", "invalid session id", nil)
		return nil, RemoteDispatchOutput{
			Host:  input.Host,
			Error: core.Sprintf("init failed: %v", err),
		}, err
	}

	callResult := mcpCallResult(ctx, url, token, sessionID, body)
	if !callResult.OK {
		err, _ := callResult.Value.(error)
		if err == nil {
			err = core.E("dispatchRemote", "tool call failed", nil)
		}
		return nil, RemoteDispatchOutput{
			Host:  input.Host,
			Error: core.Sprintf("call failed: %v", err),
		}, core.E("dispatchRemote", "tool call failed", err)
	}
	result, ok := callResult.Value.([]byte)
	if !ok {
		err := core.E("dispatchRemote", "invalid tool response", nil)
		return nil, RemoteDispatchOutput{
			Host:  input.Host,
			Error: core.Sprintf("call failed: %v", err),
		}, err
	}

	// Parse result
	output := RemoteDispatchOutput{
		Success: true,
		Host:    input.Host,
		Repo:    input.Repo,
		Agent:   input.Agent,
	}

	var rpcResponse struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if r := core.JSONUnmarshal(result, &rpcResponse); r.OK {
		if rpcResponse.Error != nil {
			output.Success = false
			output.Error = rpcResponse.Error.Message
		} else if len(rpcResponse.Result.Content) > 0 {
			var dispatchOut DispatchOutput
			if r := core.JSONUnmarshalString(rpcResponse.Result.Content[0].Text, &dispatchOut); r.OK {
				output.Success = dispatchOut.Success
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

	if addr, ok := aliases[core.Lower(host)]; ok {
		return addr
	}

	// If no port specified, add default
	if !core.Contains(host, ":") {
		return core.Concat(host, ":9101")
	}

	return host
}

// remoteToken gets the auth token for a remote agent.
func remoteToken(host string) string {
	// Check environment first
	envKey := core.Sprintf("AGENT_TOKEN_%s", core.Upper(host))
	if token := core.Env(envKey); token != "" {
		return token
	}

	// Fallback to shared agent token
	if token := core.Env("MCP_AUTH_TOKEN"); token != "" {
		return token
	}

	// Try reading from file
	home := HomeDir()
	tokenFiles := []string{
		core.Sprintf("%s/.core/tokens/%s.token", home, core.Lower(host)),
		core.Sprintf("%s/.core/agent-token", home),
	}
	for _, f := range tokenFiles {
		if r := fs.Read(f); r.OK {
			return core.Trim(r.Value.(string))
		}
	}

	return ""
}
