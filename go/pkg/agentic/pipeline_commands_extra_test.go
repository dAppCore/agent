// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// TestPipelineCommands_pipelineWorkspaceDir_Good — a --workspace resolves under
// WorkspaceRoot()/.../repo; otherwise the explicit --repo-dir is used.
func TestPipelineCommands_pipelineWorkspaceDir_Good(t *testing.T) {
	t.Cleanup(func() { SetWorkspaceRootOverride("") })
	SetWorkspaceRootOverride("/ws")
	got := pipelineWorkspaceDir(core.NewOptions(core.Option{Key: "workspace", Value: "core/go-io/t5"}))
	core.AssertEqual(t, core.JoinPath("/ws", "core/go-io/t5", "repo"), got)

	got2 := pipelineWorkspaceDir(core.NewOptions(core.Option{Key: "repo_dir", Value: "/explicit"}))
	core.AssertEqual(t, "/explicit", got2)
}

// TestPipelineCommands_PrintUsages_Good — the pipeline usage printers emit
// their command synopses.
func TestPipelineCommands_PrintUsages_Good(t *testing.T) {
	out := captureStdout(t, func() {
		printPipelineEpicUsage()
		printPipelineFixUsage()
		printPipelineBudgetUsage()
	})
	core.AssertContains(t, out, "pipeline/epic/create")
	core.AssertContains(t, out, "pipeline/fix/reviews")
	core.AssertContains(t, out, "pipeline/budget")
}
