// SPDX-License-Identifier: EUPL-1.2

// Extra coverage for the pipeline command dispatchers in
// pipeline_commands.go. The sub-handlers and the epic/fix usage paths are
// covered elsewhere; the remaining gaps are the budget/training dispatchers'
// usage cases and the unknown-action (default) arm of every dispatcher.
// These are pure routing branches with no infra.

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// TestPipeline_Dispatchers_UsageBudgetTraining — the budget + training
// dispatchers print usage and succeed when invoked with no action.
func TestPipeline_Dispatchers_UsageBudgetTraining(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	captureStdout(t, func() {
		core.AssertTrue(t, s.cmdPipelineBudget(core.NewOptions()).OK)
		core.AssertTrue(t, s.cmdPipelineTraining(core.NewOptions()).OK)
	})
}

// TestPipeline_Dispatchers_UnknownAction — every pipeline dispatcher returns a
// non-OK result carrying the unknown-command error on an unrecognised action.
func TestPipeline_Dispatchers_UnknownAction(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	dispatchers := map[string]func(core.Options) core.Result{
		"pipeline":          s.cmdPipeline,
		"pipeline-epic":     s.cmdPipelineEpic,
		"pipeline-fix":      s.cmdPipelineFix,
		"pipeline-budget":   s.cmdPipelineBudget,
		"pipeline-training": s.cmdPipelineTraining,
	}

	for name, fn := range dispatchers {
		fn := fn
		t.Run(name, func(t *testing.T) {
			var r core.Result
			captureStdout(t, func() {
				r = fn(core.NewOptions(core.Option{Key: "action", Value: "frobnicate"}))
			})
			core.AssertFalse(t, r.OK)
			core.AssertContains(t, r.Value.(error).Error(), "unknown")
		})
	}
}

// TestPipeline_CmdPipeline_HelpAction — the top dispatcher's explicit "help"
// action prints usage and returns OK (distinct from the empty-action arm).
func TestPipeline_CmdPipeline_HelpAction(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	var r core.Result
	out := captureStdout(t, func() {
		r = s.cmdPipeline(core.NewOptions(core.Option{Key: "action", Value: "help"}))
	})
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "usage: core-agent pipeline")
}

// TestPipeline_Dispatchers_HelpAction — the epic/fix/budget/training
// dispatchers also accept an explicit "help" action (the same arm as empty).
func TestPipeline_Dispatchers_HelpAction(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	captureStdout(t, func() {
		core.AssertTrue(t, s.cmdPipelineEpic(core.NewOptions(core.Option{Key: "action", Value: "help"})).OK)
		core.AssertTrue(t, s.cmdPipelineFix(core.NewOptions(core.Option{Key: "action", Value: "help"})).OK)
		core.AssertTrue(t, s.cmdPipelineBudget(core.NewOptions(core.Option{Key: "action", Value: "help"})).OK)
		core.AssertTrue(t, s.cmdPipelineTraining(core.NewOptions(core.Option{Key: "action", Value: "help"})).OK)
	})
}
