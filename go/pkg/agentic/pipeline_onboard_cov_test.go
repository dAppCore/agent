// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// TestPipelineOnboardCov_CmdOnboard_Good_ChainsAndPrintsSummary — the onboard
// command wrapper runs audit -> epic-create -> dispatch and prints the summary
// (with an epic-run line per epic), returning the typed output. The dry-run
// flag keeps dispatch from spawning a subprocess, so captureStdout is safe.
func TestPipelineOnboardCov_CmdOnboard_Good_ChainsAndPrintsSummary(t *testing.T) {
	repo := newPipelineTestRepo()
	repo.Issues[1] = &pipelineTestIssue{
		Number: 1,
		Title:  "[Audit] Security",
		Body:   "- Validate tokens\n- Sanitize input\n- Add rate limiting",
		State:  "open",
		Labels: []string{"audit", "security"},
	}
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": repo})
	s, _ := testPrepWithCore(t, srv)

	var result core.Result
	output := captureStdout(t, func() {
		result = s.cmdPipelineOnboard(core.NewOptions(
			core.Option{Key: "_arg", Value: "go-io"},
			core.Option{Key: "dry-run", Value: "true"},
		))
	})

	core.RequireTrue(t, result.OK)
	typed, ok := result.Value.(PipelineOnboardOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, typed.Success)
	core.AssertLen(t, typed.Audit.Created, 3)
	core.AssertLen(t, typed.Runs, 1)
	core.AssertContains(t, output, "repo:          core/go-io")
	core.AssertContains(t, output, "audit created: 3")
	core.AssertContains(t, output, "epic runs:     1")
	core.AssertContains(t, output, "dispatched 3 issue(s)")
}

// TestPipelineOnboardCov_CmdOnboard_Good_DirectDispatchSummary — when too few
// candidates exist to form an epic, the wrapper reports the direct-dispatch
// path (no epic runs) and prints the direct count.
func TestPipelineOnboardCov_CmdOnboard_Good_DirectDispatchSummary(t *testing.T) {
	repo := newPipelineTestRepo()
	repo.Issues[1] = &pipelineTestIssue{
		Number: 1,
		Title:  "[Audit] Security",
		Body:   "- Validate tokens\n- Sanitize input",
		State:  "open",
		Labels: []string{"audit", "security"},
	}
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": repo})
	s, _ := testPrepWithCore(t, srv)

	var result core.Result
	output := captureStdout(t, func() {
		result = s.cmdPipelineOnboard(core.NewOptions(
			core.Option{Key: "_arg", Value: "go-io"},
			core.Option{Key: "dry-run", Value: "true"},
		))
	})

	core.RequireTrue(t, result.OK)
	typed, ok := result.Value.(PipelineOnboardOutput)
	core.RequireTrue(t, ok)
	core.AssertEmpty(t, typed.Runs)
	core.AssertLen(t, typed.Direct, 2)
	core.AssertContains(t, output, "epic runs:     0")
	core.AssertContains(t, output, "direct:        2")
}

// TestPipelineOnboardCov_CmdOnboard_Ugly_NoTokenPrintsError — without a Forge
// token the underlying audit fails; the wrapper prints the error and fails.
func TestPipelineOnboardCov_CmdOnboard_Ugly_NoTokenPrintsError(t *testing.T) {
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": newPipelineTestRepo()})
	s, _ := testPrepWithCore(t, srv)
	s.forgeToken = ""

	var result core.Result
	output := captureStdout(t, func() {
		result = s.cmdPipelineOnboard(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	})

	core.AssertFalse(t, result.OK)
	core.AssertContains(t, output, "error:")
	core.AssertContains(t, output, "no Forge token configured")
}
