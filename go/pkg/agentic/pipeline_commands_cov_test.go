// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// covPipeNoTokenPrep builds a subsystem with no Forge token so any routed
// network subcommand (audit/monitor/onboard) fails fast at its token guard,
// keeping router-arm tests free of real HTTP.
func covPipeNoTokenPrep(t *testing.T) *PrepSubsystem {
	t.Helper()
	s, _ := testPrepWithCore(t, nil)
	s.forgeToken = ""
	return s
}

// TestPipelineCommandsCov_CmdPipeline_Good_RoutesAuditArm — the top router
// dispatches the "audit" action into cmdPipelineAudit, which (token-less) fails
// fast; this exercises the case "audit" routing line.
func TestPipelineCommandsCov_CmdPipeline_Good_RoutesAuditArm(t *testing.T) {
	s := covPipeNoTokenPrep(t)

	var result core.Result
	captureStdout(t, func() {
		result = s.cmdPipeline(core.NewOptions(
			core.Option{Key: "_arg", Value: "audit"},
			core.Option{Key: "repo", Value: "go-io"},
		))
	})

	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "no Forge token configured")
}

// TestPipelineCommandsCov_CmdPipeline_Good_RoutesMonitorArm — the "monitor"
// action routes into cmdPipelineMonitor (token-less fail-fast).
func TestPipelineCommandsCov_CmdPipeline_Good_RoutesMonitorArm(t *testing.T) {
	s := covPipeNoTokenPrep(t)

	var result core.Result
	captureStdout(t, func() {
		result = s.cmdPipeline(core.NewOptions(
			core.Option{Key: "_arg", Value: "monitor"},
			core.Option{Key: "repo", Value: "go-io"},
		))
	})

	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Value.(error).Error(), "no Forge token configured")
}

// TestPipelineCommandsCov_CmdPipeline_Good_RoutesOnboardArm — the "onboard"
// action routes into cmdPipelineOnboard (token-less fail-fast).
func TestPipelineCommandsCov_CmdPipeline_Good_RoutesOnboardArm(t *testing.T) {
	s := covPipeNoTokenPrep(t)

	var result core.Result
	captureStdout(t, func() {
		result = s.cmdPipeline(core.NewOptions(
			core.Option{Key: "_arg", Value: "onboard"},
			core.Option{Key: "repo", Value: "go-io"},
		))
	})

	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Value.(error).Error(), "no Forge token configured")
}

// TestPipelineCommandsCov_CmdPipeline_Good_RoutesNestedHelpArms — routing into
// the epic/fix/budget/training sub-routers with the router keyword as the action
// lands each sub-router's default arm, returning its own "unknown" envelope.
// This covers the epic/fix/budget/training routing lines of cmdPipeline.
func TestPipelineCommandsCov_CmdPipeline_Good_RoutesNestedHelpArms(t *testing.T) {
	s := covPipeNoTokenPrep(t)

	cases := []struct {
		action  string
		wantErr string
	}{
		{"epic", "unknown pipeline epic command: epic"},
		{"fix", "unknown pipeline fix command: fix"},
		{"budget", "unknown pipeline budget command: budget"},
		{"training", "unknown pipeline training command: training"},
	}
	for _, tc := range cases {
		var result core.Result
		captureStdout(t, func() {
			result = s.cmdPipeline(core.NewOptions(core.Option{Key: "_arg", Value: tc.action}))
		})
		core.AssertFalse(t, result.OK)
		err, ok := result.Value.(error)
		core.RequireTrue(t, ok)
		core.AssertContains(t, err.Error(), tc.wantErr)
	}
}

// TestPipelineCommandsCov_CmdPipelineEpic_Good_RoutesSubcommands — the epic
// router dispatches create/run/status/sync. run/status/sync hit the wrapper's
// repo+number guard; create treats the action keyword as a repo name and so
// reaches the seam, which (token-less) fails fast. All network-free.
func TestPipelineCommandsCov_CmdPipelineEpic_Good_RoutesSubcommands(t *testing.T) {
	s := covPipeNoTokenPrep(t)

	cases := []struct {
		action  string
		wantErr string
	}{
		{"create", "no Forge token configured"},
		{"run", "repo and epic number are required"},
		{"status", "repo and epic number are required"},
		{"sync", "repo and epic number are required"},
	}
	for _, tc := range cases {
		var result core.Result
		captureStdout(t, func() {
			result = s.cmdPipelineEpic(core.NewOptions(core.Option{Key: "_arg", Value: tc.action}))
		})
		core.AssertFalse(t, result.OK)
		err, ok := result.Value.(error)
		core.RequireTrue(t, ok)
		core.AssertContains(t, err.Error(), tc.wantErr)
	}
}

// TestPipelineCommandsCov_CmdPipelineFix_Good_RoutesSubcommands — the fix router
// dispatches reviews/conflicts/threads (which hit the number guard) and format
// (which hits the workspace guard after the number guard). The format case here
// supplies a number so it reaches its own workspace guard, all network-free.
func TestPipelineCommandsCov_CmdPipelineFix_Good_RoutesSubcommands(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	// reviews/conflicts/threads: number guard fires (no number).
	for _, action := range []string{"reviews", "conflicts", "threads"} {
		var result core.Result
		captureStdout(t, func() {
			result = s.cmdPipelineFix(core.NewOptions(core.Option{Key: "_arg", Value: action}))
		})
		core.AssertFalse(t, result.OK)
		core.AssertContains(t, result.Value.(error).Error(), "repo and pull request number are required")
	}

	// format: supply a number and repo so the number guard passes and the
	// workspace guard fires instead (still no subprocess).
	var fmtResult core.Result
	captureStdout(t, func() {
		fmtResult = s.cmdPipelineFix(core.NewOptions(
			core.Option{Key: "_arg", Value: "format"},
			core.Option{Key: "repo", Value: "go-io"},
			core.Option{Key: "number", Value: "12"},
		))
	})
	core.AssertFalse(t, fmtResult.OK)
	core.AssertContains(t, fmtResult.Value.(error).Error(), "workspace or repo_dir is required")
}

// TestPipelineCommandsCov_CmdPipelineBudget_Good_RoutesPlan — the budget router
// dispatches "plan" into cmdPipelineBudgetPlan (journal-backed, no network).
func TestPipelineCommandsCov_CmdPipelineBudget_Good_RoutesPlan(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	var result core.Result
	captureStdout(t, func() {
		result = s.cmdPipelineBudget(core.NewOptions(core.Option{Key: "_arg", Value: "plan"}))
	})

	core.RequireTrue(t, result.OK)
}

// TestPipelineCommandsCov_CmdPipelineBudget_Good_RoutesLog — the budget router
// dispatches "log" into cmdPipelineBudgetLog, which (no repo/agent) fails fast
// at its own guard; this covers the case "log" routing line.
func TestPipelineCommandsCov_CmdPipelineBudget_Good_RoutesLog(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	var result core.Result
	captureStdout(t, func() {
		result = s.cmdPipelineBudget(core.NewOptions(core.Option{Key: "_arg", Value: "log"}))
	})

	core.AssertFalse(t, result.OK)
}

// TestPipelineCommandsCov_CmdPipelineTraining_Good_RoutesCapture — the training
// router dispatches "capture" into cmdPipelineTrainingCapture, which (no
// repo/number) fails fast at its own guard; this covers the case "capture" line.
func TestPipelineCommandsCov_CmdPipelineTraining_Good_RoutesCapture(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	var result core.Result
	captureStdout(t, func() {
		result = s.cmdPipelineTraining(core.NewOptions(core.Option{Key: "_arg", Value: "capture"}))
	})

	core.AssertFalse(t, result.OK)
}

// TestPipelineCommandsCov_CmdPipelineBudget_Bad_UnknownAction — an unrecognised
// budget action prints usage and returns the "unknown" envelope.
func TestPipelineCommandsCov_CmdPipelineBudget_Bad_UnknownAction(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	var result core.Result
	output := captureStdout(t, func() {
		result = s.cmdPipelineBudget(core.NewOptions(core.Option{Key: "_arg", Value: "bogus"}))
	})

	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Value.(error).Error(), "unknown pipeline budget command: bogus")
	core.AssertContains(t, output, "usage: core-agent pipeline/budget")
}

// TestPipelineCommandsCov_CmdPipelineTraining_Good_RoutesStats — the training
// router dispatches "stats" into cmdPipelineTrainingStats (journal-backed).
func TestPipelineCommandsCov_CmdPipelineTraining_Good_RoutesStats(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	var result core.Result
	captureStdout(t, func() {
		result = s.cmdPipelineTraining(core.NewOptions(core.Option{Key: "_arg", Value: "stats"}))
	})

	core.RequireTrue(t, result.OK)
}

// TestPipelineCommandsCov_CmdPipelineTraining_Good_RoutesExport — the training
// router dispatches "export" into cmdPipelineTrainingExport (writes the export
// file under the test workspace).
func TestPipelineCommandsCov_CmdPipelineTraining_Good_RoutesExport(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	var result core.Result
	captureStdout(t, func() {
		result = s.cmdPipelineTraining(core.NewOptions(core.Option{Key: "_arg", Value: "export"}))
	})

	core.RequireTrue(t, result.OK)
}

// TestPipelineCommandsCov_CmdPipelineTraining_Bad_UnknownAction — an
// unrecognised training action prints usage and returns the "unknown" envelope.
func TestPipelineCommandsCov_CmdPipelineTraining_Bad_UnknownAction(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	var result core.Result
	output := captureStdout(t, func() {
		result = s.cmdPipelineTraining(core.NewOptions(core.Option{Key: "_arg", Value: "bogus"}))
	})

	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Value.(error).Error(), "unknown pipeline training command: bogus")
	core.AssertContains(t, output, "usage: core-agent pipeline/training")
}

// TestPipelineCommandsCov_PipelineSlug_Good_NormalisesAndCollapses — letters and
// digits pass through, runs of other characters collapse to a single dash, and
// leading/trailing dashes are trimmed.
func TestPipelineCommandsCov_PipelineSlug_Good_NormalisesAndCollapses(t *testing.T) {
	core.AssertEqual(t, "go-io-security", pipelineSlug("  Go/IO   Security!!  "))
	core.AssertEqual(t, "abc123", pipelineSlug("abc123"))
}

// TestPipelineCommandsCov_PipelineSlug_Bad_EmptyAndSeparatorOnly — an empty
// value and a separators-only value both fall back to the "pipeline" default.
func TestPipelineCommandsCov_PipelineSlug_Bad_EmptyAndSeparatorOnly(t *testing.T) {
	core.AssertEqual(t, "pipeline", pipelineSlug(""))
	core.AssertEqual(t, "pipeline", pipelineSlug("   "))
	core.AssertEqual(t, "pipeline", pipelineSlug("///---"))
}
