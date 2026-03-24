// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"syscall"

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

// dispatchStart unfreezes the queue and starts draining.
func (s *PrepSubsystem) dispatchStart(ctx context.Context, _ *mcp.CallToolRequest, input ShutdownInput) (*mcp.CallToolResult, ShutdownOutput, error) {
	s.frozen = false
	s.Poke() // trigger immediate drain

	return nil, ShutdownOutput{
		Success: true,
		Message: "dispatch started — queue unfrozen, draining",
	}, nil
}

// shutdownGraceful freezes the queue — running agents finish, no new dispatches.
func (s *PrepSubsystem) shutdownGraceful(ctx context.Context, _ *mcp.CallToolRequest, input ShutdownInput) (*mcp.CallToolResult, ShutdownOutput, error) {
	s.frozen = true

	running := s.countRunningByAgent("codex") + s.countRunningByAgent("claude") +
		s.countRunningByAgent("gemini") + s.countRunningByAgent("codex-spark")

	return nil, ShutdownOutput{
		Success: true,
		Running: running,
		Message: "queue frozen — running agents will finish, no new dispatches",
	}, nil
}

// shutdownNow kills all running agents and clears the queue.
func (s *PrepSubsystem) shutdownNow(ctx context.Context, _ *mcp.CallToolRequest, input ShutdownInput) (*mcp.CallToolResult, ShutdownOutput, error) {
	s.frozen = true

	wsRoot := WorkspaceRoot()
	old := core.PathGlob(core.JoinPath(wsRoot, "*", "status.json"))
	deep := core.PathGlob(core.JoinPath(wsRoot, "*", "*", "*", "status.json"))
	statusFiles := append(old, deep...)

	killed := 0
	cleared := 0

	for _, statusPath := range statusFiles {
		wsDir := core.PathDir(statusPath)
		st, err := ReadStatus(wsDir)
		if err != nil {
			continue
		}

		// Kill running agents
		if st.Status == "running" && st.PID > 0 {
			if syscall.Kill(st.PID, syscall.SIGTERM) == nil {
				killed++
			}
			st.Status = "failed"
			st.Question = "killed by shutdown_now"
			st.PID = 0
			writeStatus(wsDir, st)
		}

		// Clear queued tasks
		if st.Status == "queued" {
			st.Status = "failed"
			st.Question = "cleared by shutdown_now"
			writeStatus(wsDir, st)
			cleared++
		}
	}

	return nil, ShutdownOutput{
		Success: true,
		Running: 0,
		Queued:  0,
		Message: core.Sprintf("killed %d agents, cleared %d queued tasks", killed, cleared),
	}, nil
}
