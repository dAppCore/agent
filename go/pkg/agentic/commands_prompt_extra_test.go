// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// TestCommands_cmdPromptVersion_Bad_RequiresWorkspace — prompt version without
// a workspace prints usage and returns a workspace-required error.
func TestCommands_cmdPromptVersion_Bad_RequiresWorkspace(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	var r core.Result
	out := captureStdout(t, func() { r = s.cmdPromptVersion(core.NewOptions()) })
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, out, "usage: core-agent prompt version <workspace>")
}
