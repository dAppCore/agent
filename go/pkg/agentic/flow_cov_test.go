// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// --- executeFlowStep: capture error arm (captureFlowStepOutput returns err) ---

func TestFlow_ExecuteFlowStep_Ugly_CaptureError(t *testing.T) {
	s, c := newFlowCommandPrep()
	core.RequireTrue(t, c.Command("flow/cap-err", core.Command{Action: func(_ core.Options) core.Result {
		return core.Result{OK: true}
	}}).OK)

	orig := captureFlowStepOutput
	t.Cleanup(func() { captureFlowStepOutput = orig })
	captureFlowStepOutput = func(_ func() core.Result) (core.Result, string, string, error) {
		return core.Result{}, "", "", core.E("test", "pipe redirect failed", nil)
	}

	out := s.executeFlowStep(1, flowDefinitionStep{Name: "step-x", Cmd: "flow/cap-err"})
	core.AssertFalse(t, out.Success)
	core.AssertContains(t, out.Error, "pipe redirect failed")
}

// --- executeFlowStep: command fails, continueOnError prints "failed (continued)" ---

func TestFlow_ExecuteFlowStep_Bad_FailedContinueOnError(t *testing.T) {
	s, c := newFlowCommandPrep()
	core.RequireTrue(t, c.Command("flow/fails", core.Command{Action: func(_ core.Options) core.Result {
		return core.Result{Value: core.E("flow/fails", "boom", nil), OK: false}
	}}).OK)

	orig := captureFlowStepOutput
	t.Cleanup(func() { captureFlowStepOutput = orig })
	// Return the inner command result (not OK) with no captured streams + no err,
	// so executeFlowStep takes the result.OK == false branch.
	captureFlowStepOutput = func(run func() core.Result) (core.Result, string, string, error) {
		return run(), "", "", nil
	}

	out := s.executeFlowStep(2, flowDefinitionStep{Name: "flaky", Cmd: "flow/fails", ContinueOnError: true})
	core.AssertFalse(t, out.Success)
	core.AssertTrue(t, out.ContinueOnError)
	core.AssertNotEmpty(t, out.Error)
}

// --- executeFlowStep: streams surface on the step output ---

func TestFlow_ExecuteFlowStep_Good_CapturesStreams(t *testing.T) {
	s, c := newFlowCommandPrep()
	core.RequireTrue(t, c.Command("flow/streams", core.Command{Action: func(_ core.Options) core.Result {
		return core.Result{OK: true}
	}}).OK)

	orig := captureFlowStepOutput
	t.Cleanup(func() { captureFlowStepOutput = orig })
	captureFlowStepOutput = func(_ func() core.Result) (core.Result, string, string, error) {
		return core.Result{OK: true}, "captured stdout\n", "captured stderr\n", nil
	}

	out := captureStdout(t, func() {
		stepOut := s.executeFlowStep(1, flowDefinitionStep{Name: "noisy", Cmd: "flow/streams"})
		core.AssertTrue(t, stepOut.Success)
		core.AssertEqual(t, "captured stdout\n", stepOut.Stdout)
		core.AssertEqual(t, "captured stderr\n", stepOut.Stderr)
	})
	core.AssertContains(t, out, "captured stdout")
	core.AssertContains(t, out, "captured stderr")
}

// --- executeNestedFlowStep: unresolvable nested flow, abort (no continue) ---

func TestFlow_ExecuteNestedFlowStep_Bad_UnresolvableAborts(t *testing.T) {
	s, _ := newFlowCommandPrep()

	// A document whose only step references a flow that cannot be resolved.
	// Validation is bypassed by calling executeFlowDefinition directly, so the
	// resolution failure surfaces inside executeNestedFlowStep.
	document := flowRunDocument{
		Source: "/tmp/parent.yaml",
		Parsed: true,
		Definition: flowDefinition{
			Name: "parent",
			Steps: []flowDefinitionStep{
				{Name: "nested", Flow: "does/not/exist"},
			},
		},
	}
	ctx := flowExpansionContext{visited: map[string]bool{document.Source: true}}

	out := captureStdout(t, func() {
		summary := s.executeFlowDefinition(document, ctx)
		core.AssertFalse(t, summary.Success)
		core.AssertEqual(t, 1, summary.Executed)
		core.AssertEqual(t, 1, summary.Failed)
		core.AssertLen(t, summary.StepResults, 1)
		if len(summary.StepResults) == 1 {
			core.AssertContains(t, summary.StepResults[0].Error, "unresolvable flow")
		}
	})
	_ = out
}

// --- executeNestedFlowStep: unresolvable nested flow, continueOnError keeps going ---

func TestFlow_ExecuteNestedFlowStep_Ugly_UnresolvableContinues(t *testing.T) {
	s, c := newFlowCommandPrep()
	core.RequireTrue(t, c.Command("flow/after", core.Command{Action: func(_ core.Options) core.Result {
		return core.Result{OK: true}
	}}).OK)

	document := flowRunDocument{
		Source: "/tmp/parent2.yaml",
		Parsed: true,
		Definition: flowDefinition{
			Name: "parent2",
			Steps: []flowDefinitionStep{
				{Name: "nested", Flow: "does/not/exist", ContinueOnError: true},
				{Name: "after", Cmd: "flow/after"},
			},
		},
	}
	ctx := flowExpansionContext{visited: map[string]bool{document.Source: true}}

	out := captureStdout(t, func() {
		summary := s.executeFlowDefinition(document, ctx)
		// The nested step failed but continueOnError let the next step run + pass.
		core.AssertTrue(t, summary.Success)
		core.AssertEqual(t, 2, summary.Executed)
		core.AssertEqual(t, 1, summary.Failed)
		core.AssertEqual(t, 1, summary.Passed)
	})
	_ = out
}

// --- validateExecutableFlowStep: legacy run syntax + missing cmd ---

func TestFlow_ValidateExecutableFlowStep_Bad_LegacyRunSyntax(t *testing.T) {
	s, _ := newFlowCommandPrep()
	ctx := flowExpansionContext{visited: map[string]bool{"src": true}}

	err := s.validateExecutableFlowStep(1, flowDefinitionStep{Name: "legacy", Run: "echo hi"}, "src", ctx)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "legacy run syntax")
}

func TestFlow_ValidateExecutableFlowStep_Bad_MissingCmd(t *testing.T) {
	s, _ := newFlowCommandPrep()
	ctx := flowExpansionContext{visited: map[string]bool{"src": true}}

	err := s.validateExecutableFlowStep(2, flowDefinitionStep{Name: "empty"}, "src", ctx)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "must define cmd")
}

func TestFlow_ValidateExecutableFlowStep_Bad_UnknownCommand(t *testing.T) {
	s, _ := newFlowCommandPrep()
	ctx := flowExpansionContext{visited: map[string]bool{"src": true}}

	err := s.validateExecutableFlowStep(3, flowDefinitionStep{Name: "ghost", Cmd: "flow/never-registered"}, "src", ctx)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "unknown command")
}

func TestFlow_ValidateExecutableFlowStep_Ugly_NonExecutableCommand(t *testing.T) {
	s, c := newFlowCommandPrep()
	// A command registered with a nil Action is resolvable but not executable.
	core.RequireTrue(t, c.Command("flow/no-action", core.Command{}).OK)
	ctx := flowExpansionContext{visited: map[string]bool{"src": true}}

	err := s.validateExecutableFlowStep(4, flowDefinitionStep{Name: "inert", Cmd: "flow/no-action"}, "src", ctx)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "non-executable command")
}

// --- validateNestedFlowStep: depth guard rejects deep composition ---

func TestFlow_ValidateNestedFlowStep_Bad_DepthExceeded(t *testing.T) {
	s, _ := newFlowCommandPrep()
	// A context already at the nesting limit means depth+1 exceeds the guard.
	ctx := flowExpansionContext{visited: map[string]bool{"src": true}, depth: maxFlowNestingDepth}

	err := s.validateNestedFlowStep("deep", flowDefinitionStep{Name: "deep", Flow: "child"}, "src", ctx)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "depth exceeds limit")
}

// --- printFlowStepStream: empty stream is a no-op ---

func TestFlow_PrintFlowStepStream_Bad_EmptyStream(t *testing.T) {
	// A blank/whitespace stream prints nothing (early return).
	out := captureStdout(t, func() {
		printFlowStepStream("stdout", "")
		printFlowStepStream("stderr", "\n")
	})
	core.AssertEmpty(t, out)
}

func TestFlow_PrintFlowStepStream_Good_PrintsLines(t *testing.T) {
	out := captureStdout(t, func() {
		printFlowStepStream("stdout", "line one\nline two\n")
	})
	core.AssertContains(t, out, "stdout:")
	core.AssertContains(t, out, "line one")
	core.AssertContains(t, out, "line two")
}
