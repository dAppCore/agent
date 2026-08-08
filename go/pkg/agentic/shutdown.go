// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go"
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
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_dispatch_start",
		Description: "Start the dispatch queue runner. Unfreezes the queue and begins draining.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, input ShutdownInput) (*mcp.CallToolResult, ShutdownOutput, error) {
		return dispatchStart(s, ctx, request, input)
	})

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_dispatch_shutdown",
		Description: "Graceful shutdown: stop accepting new jobs, let running agents finish. Queue is frozen.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, input ShutdownInput) (*mcp.CallToolResult, ShutdownOutput, error) {
		return shutdownGraceful(s, ctx, request, input)
	})

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_dispatch_shutdown_now",
		Description: "Hard shutdown: kill all running agents immediately. Queue is cleared.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, input ShutdownInput) (*mcp.CallToolResult, ShutdownOutput, error) {
		return shutdownNow(s, ctx, request, input)
	})
}

// result := c.Action("agentic.dispatch.start").Run(ctx, core.NewOptions())
func (s *PrepSubsystem) handleDispatchStart(ctx context.Context, _ core.Options) core.Result {
	_, output, err := dispatchStart(s, ctx, nil, ShutdownInput{})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("agentic.dispatch.shutdown").Run(ctx, core.NewOptions())
func (s *PrepSubsystem) handleDispatchShutdown(ctx context.Context, _ core.Options) core.Result {
	_, output, err := shutdownGraceful(s, ctx, nil, ShutdownInput{})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("agentic.dispatch.shutdown_now").Run(ctx, core.NewOptions())
func (s *PrepSubsystem) handleDispatchShutdownNow(ctx context.Context, _ core.Options) core.Result {
	_, output, err := shutdownNow(s, ctx, nil, ShutdownInput{})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

var dispatchStart = func(s *PrepSubsystem, ctx context.Context, _ *mcp.CallToolRequest, input ShutdownInput) (*mcp.CallToolResult, ShutdownOutput, error) {
	if s.ServiceRuntime != nil {
		// Reported: a lifecycle transition that fails silently leaves the
		if r := s.Core().Action("runner.start").Run(ctx, core.NewOptions()); !r.OK {
			core.Warn("agentic: runner.start action failed", "reason", r.Value)
		}
	}
	return nil, ShutdownOutput{
		Success: true,
		Message: "dispatch started — queue unfrozen, draining",
	}, nil
}

var shutdownGraceful = func(s *PrepSubsystem, ctx context.Context, _ *mcp.CallToolRequest, input ShutdownInput) (*mcp.CallToolResult, ShutdownOutput, error) {
	if s.ServiceRuntime != nil {
		// Reported: a lifecycle transition that fails silently leaves the
		if r := s.Core().Action("runner.stop").Run(ctx, core.NewOptions()); !r.OK {
			core.Warn("agentic: runner.stop action failed", "reason", r.Value)
		}
	}
	return nil, ShutdownOutput{
		Success: true,
		Message: "queue frozen — running agents will finish, no new dispatches",
	}, nil
}

var shutdownNow = func(s *PrepSubsystem, ctx context.Context, _ *mcp.CallToolRequest, input ShutdownInput) (*mcp.CallToolResult, ShutdownOutput, error) {
	if s.ServiceRuntime != nil {
		// Reported: a lifecycle transition that fails silently leaves the
		if r := s.Core().Action("runner.kill").Run(ctx, core.NewOptions()); !r.OK {
			core.Warn("agentic: runner.kill action failed", "reason", r.Value)
		}
	}
	return nil, ShutdownOutput{
		Success: true,
		Running: 0,
		Queued:  0,
		Message: "killed all agents, cleared queue",
	}, nil
}
