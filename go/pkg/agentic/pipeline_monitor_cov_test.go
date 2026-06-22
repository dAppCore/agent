// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// TestPipelineMonitorCov_CheckStatus_Good_MapsRawStates — every recognised raw
// CI state maps to its normalised status bucket; the default arm yields "".
func TestPipelineMonitorCov_CheckStatus_Good_MapsRawStates(t *testing.T) {
	core.AssertEqual(t, "completed", pipelineCheckStatus("success"))
	core.AssertEqual(t, "completed", pipelineCheckStatus("failure"))
	core.AssertEqual(t, "completed", pipelineCheckStatus("error"))
	core.AssertEqual(t, "queued", pipelineCheckStatus("pending"))
	core.AssertEqual(t, "queued", pipelineCheckStatus("queued"))
	core.AssertEqual(t, "in_progress", pipelineCheckStatus("running"))
	core.AssertEqual(t, "in_progress", pipelineCheckStatus("in_progress"))
	core.AssertEqual(t, "", pipelineCheckStatus("skipped"))
	core.AssertEqual(t, "", pipelineCheckStatus(""))
}

// TestPipelineMonitorCov_CheckStatus_Ugly_CaseInsensitive — the mapping lowers
// the raw state first, so mixed-case input still resolves.
func TestPipelineMonitorCov_CheckStatus_Ugly_CaseInsensitive(t *testing.T) {
	core.AssertEqual(t, "completed", pipelineCheckStatus("SUCCESS"))
	core.AssertEqual(t, "in_progress", pipelineCheckStatus("In_Progress"))
}

// TestPipelineMonitorCov_CmdMonitor_Good_RepoScopePrintsActions — the monitor
// command wrapper, scoped to one repo, prints the repo header and each
// intervention line and returns the typed output. HTTP-only path.
func TestPipelineMonitorCov_CmdMonitor_Good_RepoScopePrintsActions(t *testing.T) {
	repo := newPipelineTestRepo()
	repo.Pulls[1] = &pipelineTestPR{
		Number:         1,
		Title:          "Conflicting PR",
		State:          "open",
		Mergeable:      boolPtr(false),
		MergeableState: "dirty",
		HeadRef:        "agent/conflict",
		HeadSHA:        "sha-conflict",
		BaseRef:        "dev",
	}
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": repo})
	s, _ := testPrepWithCore(t, srv)

	var result core.Result
	output := captureStdout(t, func() {
		result = s.cmdPipelineMonitor(core.NewOptions(
			core.Option{Key: "_arg", Value: "go-io"},
			core.Option{Key: "dry-run", Value: "true"},
		))
	})

	core.RequireTrue(t, result.OK)
	typed, ok := result.Value.(PipelineMonitorOutput)
	core.RequireTrue(t, ok)
	core.AssertLen(t, typed.Actions, 1)
	core.AssertEqual(t, "fix/conflicts", typed.Actions[0].Action)
	core.AssertContains(t, output, "repo:    core/go-io")
	core.AssertContains(t, output, "actions: 1")
	core.AssertContains(t, output, "go-io #1 fix/conflicts")
}

// TestPipelineMonitorCov_CmdMonitor_Good_OrgScopeNoInterventions — without a
// repo the wrapper lists the org's repos, prints the org header, and reports
// "no interventions" when nothing is actionable.
func TestPipelineMonitorCov_CmdMonitor_Good_OrgScopeNoInterventions(t *testing.T) {
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": newPipelineTestRepo()})
	s, _ := testPrepWithCore(t, srv)

	var result core.Result
	output := captureStdout(t, func() {
		result = s.cmdPipelineMonitor(core.NewOptions(core.Option{Key: "dry-run", Value: "true"}))
	})

	core.RequireTrue(t, result.OK)
	typed, ok := result.Value.(PipelineMonitorOutput)
	core.RequireTrue(t, ok)
	core.AssertEmpty(t, typed.Actions)
	core.AssertContains(t, output, "org:     core")
	core.AssertContains(t, output, "no interventions")
}

// TestPipelineMonitorCov_CmdMonitor_Bad_NoTokenPrintsError — with no Forge
// token the wrapper prints the error and returns a failed result.
func TestPipelineMonitorCov_CmdMonitor_Bad_NoTokenPrintsError(t *testing.T) {
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": newPipelineTestRepo()})
	s, _ := testPrepWithCore(t, srv)
	s.forgeToken = ""

	var result core.Result
	output := captureStdout(t, func() {
		result = s.cmdPipelineMonitor(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	})

	core.AssertFalse(t, result.OK)
	core.AssertContains(t, output, "error:")
	core.AssertContains(t, output, "no Forge token configured")
}
