// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ShutdownInput is the input for agentic_dispatch_shutdown.
//
//	input := agentic.ShutdownInput{}
type ShutdownInput struct{}

// ShutdownOutput is the output for agentic_dispatch_shutdown.
//
//	out := agentic.ShutdownOutput{Success: true, Running: 3, Message: "draining"}
type ShutdownOutput struct {
	Success bool   `json:"success"`
	Running int    `json:"running"`
	Queued  int    `json:"queued"`
	Message string `json:"message"`
}

func (s *PrepSubsystem) registerShutdownTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_dispatch_start",
		Description: "Start the dispatch queue runner. Unfreezes the queue and begins draining.",
	}, s.dispatchStart)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_dispatch_shutdown",
		Description: "Graceful shutdown: stop accepting new jobs, let running agents finish. Queue is frozen.",
	}, s.shutdownGraceful)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_dispatch_shutdown_now",
		Description: "Hard shutdown: kill all running agents immediately. Queue is cleared.",
	}, s.shutdownNow)
}

// dispatchStart delegates to runner.start Action.
func (s *PrepSubsystem) dispatchStart(ctx context.Context, _ *mcp.CallToolRequest, input ShutdownInput) (*mcp.CallToolResult, ShutdownOutput, error) {
	if s.ServiceRuntime != nil {
		s.Core().Action("runner.start").Run(ctx, core.NewOptions())
	}
	return nil, ShutdownOutput{
		Success: true,
		Message: "dispatch started — queue unfrozen, draining",
	}, nil
}

// shutdownGraceful delegates to runner.stop Action.
func (s *PrepSubsystem) shutdownGraceful(ctx context.Context, _ *mcp.CallToolRequest, input ShutdownInput) (*mcp.CallToolResult, ShutdownOutput, error) {
	if s.ServiceRuntime != nil {
		s.Core().Action("runner.stop").Run(ctx, core.NewOptions())
	}
	return nil, ShutdownOutput{
		Success: true,
		Message: "queue frozen — running agents will finish, no new dispatches",
	}, nil
}

// shutdownNow delegates to runner.kill Action.
func (s *PrepSubsystem) shutdownNow(ctx context.Context, _ *mcp.CallToolRequest, input ShutdownInput) (*mcp.CallToolResult, ShutdownOutput, error) {
	if s.ServiceRuntime != nil {
		s.Core().Action("runner.kill").Run(ctx, core.NewOptions())
	}
	return nil, ShutdownOutput{
		Success: true,
		Running: 0,
		Queued:  0,
		Message: "killed all agents, cleared queue",
	}, nil
}
