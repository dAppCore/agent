// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestCommandsDispatchCov_CmdDispatchStart_Ugly_StartFails overrides the start
// seam to fail, exercising the error arm of cmdDispatchStart.
func TestCommandsDispatchCov_CmdDispatchStart_Ugly_StartFails(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	original := dispatchStart
	t.Cleanup(func() { dispatchStart = original })
	dispatchStart = func(_ *PrepSubsystem, _ context.Context, _ *mcp.CallToolRequest, _ ShutdownInput) (*mcp.CallToolResult, ShutdownOutput, error) {
		return nil, ShutdownOutput{}, core.E("agentic.dispatchStart", "runner unavailable", nil)
	}

	var r core.Result
	output := captureStdout(t, func() { r = s.cmdDispatchStart(core.NewOptions()) })
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Value.(error).Error(), "runner unavailable")
	core.AssertContains(t, output, "error:")
}

// TestCommandsDispatchCov_CmdDispatchShutdown_Ugly_ShutdownFails overrides the
// graceful-shutdown seam to fail, exercising its error arm.
func TestCommandsDispatchCov_CmdDispatchShutdown_Ugly_ShutdownFails(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	original := shutdownGraceful
	t.Cleanup(func() { shutdownGraceful = original })
	shutdownGraceful = func(_ *PrepSubsystem, _ context.Context, _ *mcp.CallToolRequest, _ ShutdownInput) (*mcp.CallToolResult, ShutdownOutput, error) {
		return nil, ShutdownOutput{}, core.E("agentic.shutdownGraceful", "freeze failed", nil)
	}

	var r core.Result
	output := captureStdout(t, func() { r = s.cmdDispatchShutdown(core.NewOptions()) })
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Value.(error).Error(), "freeze failed")
	core.AssertContains(t, output, "error:")
}

// TestCommandsDispatchCov_CmdDispatchShutdownNow_Ugly_KillFails overrides the
// kill seam to fail, exercising its error arm.
func TestCommandsDispatchCov_CmdDispatchShutdownNow_Ugly_KillFails(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	original := shutdownNow
	t.Cleanup(func() { shutdownNow = original })
	shutdownNow = func(_ *PrepSubsystem, _ context.Context, _ *mcp.CallToolRequest, _ ShutdownInput) (*mcp.CallToolResult, ShutdownOutput, error) {
		return nil, ShutdownOutput{}, core.E("agentic.shutdownNow", "kill failed", nil)
	}

	var r core.Result
	output := captureStdout(t, func() { r = s.cmdDispatchShutdownNow(core.NewOptions()) })
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Value.(error).Error(), "kill failed")
	core.AssertContains(t, output, "error:")
}

// TestCommandsDispatchCov_CmdDispatchShutdownNow_Good_PrintsRunningQueued — a
// shutdown-now result with running/queued counts prints those extra lines.
func TestCommandsDispatchCov_CmdDispatchShutdownNow_Good_PrintsRunningQueued(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	original := shutdownNow
	t.Cleanup(func() { shutdownNow = original })
	shutdownNow = func(_ *PrepSubsystem, _ context.Context, _ *mcp.CallToolRequest, _ ShutdownInput) (*mcp.CallToolResult, ShutdownOutput, error) {
		return nil, ShutdownOutput{Success: true, Message: "killed all agents", Running: 2, Queued: 5}, nil
	}

	var r core.Result
	output := captureStdout(t, func() { r = s.cmdDispatchShutdownNow(core.NewOptions()) })
	core.RequireTrue(t, r.OK)
	core.AssertContains(t, output, "killed all agents")
	core.AssertContains(t, output, "running: 2")
	core.AssertContains(t, output, "queued:  5")
}
