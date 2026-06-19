// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// TestAgentic_PipelineCmd_Guards — the pipeline epic/fix/monitor command
// wrappers reject empty options (missing repo / PR number) before dispatching.
func TestAgentic_PipelineCmd_Guards(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	captureStdout(t, func() {
		core.AssertFalse(t, s.cmdPipelineEpicCreate(core.NewOptions()).OK)
		core.AssertFalse(t, s.cmdPipelineEpicStatus(core.NewOptions()).OK)
		core.AssertFalse(t, s.cmdPipelineEpicSync(core.NewOptions()).OK)
		core.AssertFalse(t, s.cmdPipelineFixReviews(core.NewOptions()).OK)
		core.AssertFalse(t, s.cmdPipelineFixConflicts(core.NewOptions()).OK)
		core.AssertFalse(t, s.cmdPipelineFixFormat(core.NewOptions()).OK)
		core.AssertFalse(t, s.cmdPipelineFixThreads(core.NewOptions()).OK)
		core.AssertFalse(t, s.cmdPipelineMonitor(core.NewOptions()).OK)
	})
}
