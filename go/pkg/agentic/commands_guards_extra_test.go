// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// TestCommands_cmdExtract_Bad_UnreadableSource — extract with an unreadable
// --source path surfaces the read error (no write, no side effects).
func TestCommands_cmdExtract_Bad_UnreadableSource(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	var r core.Result
	out := captureStdout(t, func() {
		r = s.cmdExtract(core.NewOptions(core.Option{Key: "source", Value: "/no/such/agent-output.txt"}))
	})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, out, "error:")
}

// TestCommandsForge_cmdBranchDelete_Bad_RequiresRepoAndBranch — branch delete
// without repo+branch prints usage and errors before any forge call.
func TestCommandsForge_cmdBranchDelete_Bad_RequiresRepoAndBranch(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	var r core.Result
	out := captureStdout(t, func() { r = s.cmdBranchDelete(core.NewOptions()) })
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, out, "usage: core-agent branch delete")
}
