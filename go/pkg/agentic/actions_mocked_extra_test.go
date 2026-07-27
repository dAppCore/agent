// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestAgentic_DispatchBranch_Handlers_Mocked — branch-delete + dispatch-start
// wrap their (mocked) underlying ops and surface a successful Result without
// touching a real forge or spawning a dispatch loop.
func TestAgentic_DispatchBranch_Handlers_Mocked(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx := context.Background()

	origDel, origStart := deleteBranch, dispatchStart
	defer func() { deleteBranch, dispatchStart = origDel, origStart }()
	deleteBranch = func(_ *PrepSubsystem, _ context.Context, _ *mcp.CallToolRequest, _ DeleteBranchInput) (*mcp.CallToolResult, DeleteBranchOutput, error) {
		return nil, DeleteBranchOutput{}, nil
	}
	dispatchStart = func(_ *PrepSubsystem, _ context.Context, _ *mcp.CallToolRequest, _ ShutdownInput) (*mcp.CallToolResult, ShutdownOutput, error) {
		return nil, ShutdownOutput{}, nil
	}

	captureStdout(t, func() {
		core.AssertTrue(t, s.handleBranchDelete(ctx, core.NewOptions()).OK)
		core.AssertTrue(t, s.handleDispatchStart(ctx, core.NewOptions()).OK)
	})
}
