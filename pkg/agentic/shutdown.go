// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go/core"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ShutdownInput struct{}

type ShutdownOutput struct {
	Success bool   `json:"success"`
	Running int    `json:"running"`
	Queued  int    `json:"queued"`
	Message string `json:"message"`
}

func (s *PrepSubsystem) registerShutdownTools(svc *coremcp.Service) {
	mcp.AddTool(svc.Server(), &mcp.Tool{
		Name:        "agentic_dispatch_start",
		Description: "Start the dispatch queue runner. Unfreezes the queue and begins draining.",
	}, s.dispatchStart)

	mcp.AddTool(svc.Server(), &mcp.Tool{
		Name:        "agentic_dispatch_shutdown",
		Description: "Graceful shutdown: stop accepting new jobs, let running agents finish. Queue is frozen.",
	}, s.shutdownGraceful)

	mcp.AddTool(svc.Server(), &mcp.Tool{
		Name:        "agentic_dispatch_shutdown_now",
		Description: "Hard shutdown: kill all running agents immediately. Queue is cleared.",
	}, s.shutdownNow)
}

// result := c.Action("agentic.dispatch.start").Run(ctx, core.NewOptions())
func (s *PrepSubsystem) handleDispatchStart(ctx context.Context, _ core.Options) core.Result {
	_, output, err := s.dispatchStart(ctx, nil, ShutdownInput{})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("agentic.dispatch.shutdown").Run(ctx, core.NewOptions())
func (s *PrepSubsystem) handleDispatchShutdown(ctx context.Context, _ core.Options) core.Result {
	_, output, err := s.shutdownGraceful(ctx, nil, ShutdownInput{})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("agentic.dispatch.shutdown_now").Run(ctx, core.NewOptions())
func (s *PrepSubsystem) handleDispatchShutdownNow(ctx context.Context, _ core.Options) core.Result {
	_, output, err := s.shutdownNow(ctx, nil, ShutdownInput{})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) dispatchStart(ctx context.Context, _ *mcp.CallToolRequest, input ShutdownInput) (*mcp.CallToolResult, ShutdownOutput, error) {
	if s.ServiceRuntime != nil {
		s.Core().Action("runner.start").Run(ctx, core.NewOptions())
	}
	return nil, ShutdownOutput{
		Success: true,
		Message: "dispatch started — queue unfrozen, draining",
	}, nil
}

func (s *PrepSubsystem) shutdownGraceful(ctx context.Context, _ *mcp.CallToolRequest, input ShutdownInput) (*mcp.CallToolResult, ShutdownOutput, error) {
	if s.ServiceRuntime != nil {
		s.Core().Action("runner.stop").Run(ctx, core.NewOptions())
	}
	return nil, ShutdownOutput{
		Success: true,
		Message: "queue frozen — running agents will finish, no new dispatches",
	}, nil
}

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
