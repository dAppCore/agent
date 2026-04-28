// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"os"
	"testing"
	"time"

	core "dappco.re/go"
)

func newFlowCommandPrep() (*PrepSubsystem, *core.Core) {
	c := core.New()
	return &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}, c
}

func TestCommandsFlow_CmdFlowPreview_Good_ReadsYamlFlowFile(t *testing.T) {
	dir := t.TempDir()
	flowPath := core.JoinPath(dir, "pkg", "lib", "flow", "verify")
	core.RequireTrue(t, fs.EnsureDir(flowPath).OK)

	filePath := core.JoinPath(flowPath, "go-qa.yaml")
	core.RequireTrue(t, fs.Write(filePath, core.Concat(
		"name: Go QA\n",
		"description: Build and test a Go project\n",
		"steps:\n",
		"  - name: build\n",
		"    run: go build ./...\n",
		"  - name: verify\n",
		"    flow: verify/go-qa.yaml\n",
	)).OK)

	s := newTestPrep(t)
	output := captureStdout(t, func() {
		r := s.cmdFlowPreview(core.NewOptions(core.Option{Key: "_arg", Value: filePath}))
		core.RequireTrue(t, r.OK)

		flowOutput, ok := r.Value.(FlowRunOutput)
		core.RequireTrue(t, ok)
		core.AssertTrue(t, flowOutput.Success)
		core.AssertEqual(t, filePath, flowOutput.Source)
		core.AssertEqual(t, "Go QA", flowOutput.Name)
		core.AssertEqual(t, "Build and test a Go project", flowOutput.Description)
		core.AssertEqual(t, 2, flowOutput.Steps)
	})

	core.AssertContains(t, output, "steps: 2")
	core.AssertContains(t, output, "build: run go build ./...")
	core.AssertContains(t, output, "verify: flow verify/go-qa.yaml")
}

func TestCommandsFlow_CmdRunFlow_Good_ExecutesSequentialSteps(t *testing.T) {
	dir := t.TempDir()
	filePath := core.JoinPath(dir, "execute-flow.yaml")
	core.RequireTrue(t, fs.Write(filePath, core.Concat(
		"name: Execute Flow\n",
		"description: Run registered commands\n",
		"steps:\n",
		"  - name: first\n",
		"    cmd: flow/test-first\n",
		"  - name: second\n",
		"    cmd: flow/test-second\n",
		"    args:\n",
		"      - --mode=fast\n",
		"  - name: third\n",
		"    cmd: flow/test-third\n",
		"    args:\n",
		"      - payload\n",
	)).OK)

	s, c := newFlowCommandPrep()
	invoked := []string{}
	core.RequireTrue(t, c.Command("flow/test-first", core.Command{Action: func(_ core.Options) core.Result {
		invoked = append(invoked, "first")
		core.Print(nil, "first stdout")
		return core.Result{OK: true}
	}}).OK)
	core.RequireTrue(t, c.Command("flow/test-second", core.Command{Action: func(options core.Options) core.Result {
		invoked = append(invoked, "second")
		core.AssertEqual(t, "fast", options.String("mode"))
		_, _ = os.Stderr.WriteString("second stderr\n")
		return core.Result{OK: true}
	}}).OK)
	core.RequireTrue(t, c.Command("flow/test-third", core.Command{Action: func(options core.Options) core.Result {
		invoked = append(invoked, "third")
		core.AssertEqual(t, "payload", options.String("_arg"))
		return core.Result{OK: true}
	}}).OK)

	output := captureStdout(t, func() {
		r := s.cmdRunFlow(core.NewOptions(core.Option{Key: "_arg", Value: filePath}))
		core.RequireTrue(t, r.OK)

		flowOutput, ok := r.Value.(FlowRunOutput)
		core.RequireTrue(t, ok)
		core.AssertTrue(t, flowOutput.Success)
		core.AssertEqual(t, filePath, flowOutput.Source)
		core.AssertEqual(t, "Execute Flow", flowOutput.Name)
		core.AssertEqual(t, "Run registered commands", flowOutput.Description)
		core.AssertEqual(t, 3, flowOutput.Steps)
		core.AssertEqual(t, 3, flowOutput.Executed)
		core.AssertEqual(t, 3, flowOutput.Passed)
		core.AssertEqual(t, 0, flowOutput.Failed)
		core.AssertLen(t, flowOutput.StepResults, 3)
		core.AssertEqual(t, "first", flowOutput.StepResults[0].Name)
		core.AssertEqual(t, "flow/test-second", flowOutput.StepResults[1].Command)
		core.AssertEqual(t, "second stderr\n", flowOutput.StepResults[1].Stderr)
	})

	core.AssertEqual(t, []string{"first", "second", "third"}, invoked)
	core.AssertContains(t, output, "steps: 3")
	core.AssertContains(t, output, "first stdout")
	core.AssertContains(t, output, "second stderr")
	core.AssertContains(t, output, "totals: ran=3 passed=3 failed=0")
}

func TestCommandsFlow_CmdRunFlow_Good_RendersVariablesAndDryRun(t *testing.T) {
	dir := t.TempDir()
	flowPath := core.JoinPath(dir, "pkg", "lib", "flow", "verify")
	core.RequireTrue(t, fs.EnsureDir(flowPath).OK)

	filePath := core.JoinPath(flowPath, "go-qa.yaml")
	core.RequireTrue(t, fs.Write(filePath, core.Concat(
		"name: Go QA\n",
		"description: Build {{ repo }}\n",
		"steps:\n",
		"  - name: build\n",
		"    cmd: flow/test-build\n",
		"    args:\n",
		"      - --repo={{ repo }}\n",
	)).OK)

	s := newTestPrep(t)
	output := captureStdout(t, func() {
		r := s.cmdRunFlow(core.NewOptions(
			core.Option{Key: "_arg", Value: filePath},
			core.Option{Key: "dry-run", Value: true},
			core.Option{Key: "var", Value: "repo=core/go"},
		))
		core.RequireTrue(t, r.OK)

		flowOutput, ok := r.Value.(FlowRunOutput)
		core.RequireTrue(t, ok)
		core.AssertTrue(t, flowOutput.Success)
		core.AssertEqual(t, "Go QA", flowOutput.Name)
		core.AssertEqual(t, "Build core/go", flowOutput.Description)
		core.AssertEqual(t, 1, flowOutput.Steps)
	})

	core.AssertContains(t, output, "dry-run: true")
	core.AssertContains(t, output, "vars: 1")
	core.AssertContains(t, output, "desc:  Build core/go")
	core.AssertContains(t, output, "build: cmd flow/test-build --repo=core/go")
}

func TestCommandsFlow_CmdFlowPreview_Good_ResolvesNestedFlowReferences(t *testing.T) {
	dir := t.TempDir()
	flowRoot := core.JoinPath(dir, "pkg", "lib", "flow")
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(flowRoot, "verify")).OK)

	rootPath := core.JoinPath(flowRoot, "root.yaml")
	core.RequireTrue(t, fs.Write(rootPath, core.Concat(
		"name: Root Flow\n",
		"description: Resolve nested flow references\n",
		"steps:\n",
		"  - name: child\n",
		"    flow: verify/go-qa.yaml\n",
	)).OK)

	childPath := core.JoinPath(flowRoot, "verify", "go-qa.yaml")
	core.RequireTrue(t, fs.Write(childPath, core.Concat(
		"name: Child Flow\n",
		"description: Nested flow body\n",
		"steps:\n",
		"  - name: child-run\n",
		"    run: echo child\n",
	)).OK)

	s := newTestPrep(t)
	output := captureStdout(t, func() {
		r := s.cmdFlowPreview(core.NewOptions(core.Option{Key: "_arg", Value: rootPath}))
		core.RequireTrue(t, r.OK)

		flowOutput, ok := r.Value.(FlowRunOutput)
		core.RequireTrue(t, ok)
		core.AssertTrue(t, flowOutput.Success)
		core.AssertEqual(t, 1, flowOutput.Steps)
		core.AssertEqual(t, 2, flowOutput.ResolvedSteps)
	})

	core.AssertContains(t, output, "resolved:")
	core.AssertContains(t, output, "child-run: run echo child")
}

func TestCommandsFlow_CmdRunFlow_Bad_MissingPath(t *testing.T) {
	s := newTestPrep(t)

	r := s.cmdRunFlow(core.NewOptions())
	core.AssertFalse(t, r.OK)

	err, ok := r.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "flow path or slug is required")
}

func TestCommandsFlow_CmdRunFlow_Bad_RejectsUnknownCommandAtParseTime(t *testing.T) {
	dir := t.TempDir()
	filePath := core.JoinPath(dir, "missing-command.yaml")
	core.RequireTrue(t, fs.Write(filePath, core.Concat(
		"name: Missing Command\n",
		"steps:\n",
		"  - name: first\n",
		"    cmd: flow/known\n",
		"  - name: second\n",
		"    cmd: flow/missing\n",
	)).OK)

	s, c := newFlowCommandPrep()
	invoked := []string{}
	core.RequireTrue(t, c.Command("flow/known", core.Command{Action: func(_ core.Options) core.Result {
		invoked = append(invoked, "known")
		return core.Result{OK: true}
	}}).OK)

	r := s.cmdRunFlow(core.NewOptions(core.Option{Key: "_arg", Value: filePath}))
	core.AssertFalse(t, r.OK)

	err, ok := r.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "references unknown command: flow/missing")
	core.AssertEmpty(t, invoked)
}

func TestCommandsFlow_CmdRunFlow_Ugly_InvalidYaml(t *testing.T) {
	dir := t.TempDir()
	filePath := core.JoinPath(dir, "broken-flow.yaml")
	core.RequireTrue(t, fs.Write(filePath, "name: [broken\n").OK)

	s := newTestPrep(t)
	r := s.cmdRunFlow(core.NewOptions(core.Option{Key: "_arg", Value: filePath}))
	core.AssertFalse(t, r.OK)

	err, ok := r.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "invalid flow definition")
}

func TestCommandsFlow_CmdRunFlow_Ugly_StopsOnFirstFailure(t *testing.T) {
	dir := t.TempDir()
	filePath := core.JoinPath(dir, "stops-on-failure.yaml")
	core.RequireTrue(t, fs.Write(filePath, core.Concat(
		"name: Stop On Failure\n",
		"steps:\n",
		"  - name: first\n",
		"    cmd: flow/first\n",
		"  - name: second\n",
		"    cmd: flow/fail\n",
		"  - name: third\n",
		"    cmd: flow/third\n",
	)).OK)

	s, c := newFlowCommandPrep()
	invoked := []string{}
	core.RequireTrue(t, c.Command("flow/first", core.Command{Action: func(_ core.Options) core.Result {
		invoked = append(invoked, "first")
		return core.Result{OK: true}
	}}).OK)
	core.RequireTrue(t, c.Command("flow/fail", core.Command{Action: func(_ core.Options) core.Result {
		invoked = append(invoked, "second")
		return core.Result{Value: "boom", OK: false}
	}}).OK)
	core.RequireTrue(t, c.Command("flow/third", core.Command{Action: func(_ core.Options) core.Result {
		invoked = append(invoked, "third")
		return core.Result{OK: true}
	}}).OK)

	output := captureStdout(t, func() {
		r := s.cmdRunFlow(core.NewOptions(core.Option{Key: "_arg", Value: filePath}))
		core.AssertFalse(t, r.OK)

		flowOutput, ok := r.Value.(FlowRunOutput)
		core.RequireTrue(t, ok)
		core.AssertFalse(t, flowOutput.Success)
		core.AssertEqual(t, 2, flowOutput.Executed)
		core.AssertEqual(t, 1, flowOutput.Passed)
		core.AssertEqual(t, 1, flowOutput.Failed)
		core.AssertLen(t, flowOutput.StepResults, 2)
		core.AssertContains(t, flowOutput.StepResults[1].Error, "boom")
	})

	core.AssertEqual(t, []string{"first", "second"}, invoked)
	core.AssertContains(t, output, "second: failed")
	core.AssertContains(t, output, "totals: ran=2 passed=1 failed=1")
	core.AssertNotContains(t, output, "third: passed")
}

func TestCommandsFlow_CmdRunFlow_Ugly_ContinueOnErrorRunsRemainingSteps(t *testing.T) {
	dir := t.TempDir()
	filePath := core.JoinPath(dir, "continue-on-error.yaml")
	core.RequireTrue(t, fs.Write(filePath, core.Concat(
		"name: Continue On Error\n",
		"steps:\n",
		"  - name: first\n",
		"    cmd: flow/first\n",
		"  - name: second\n",
		"    cmd: flow/fail\n",
		"    continueOnError: true\n",
		"  - name: third\n",
		"    cmd: flow/third\n",
	)).OK)

	s, c := newFlowCommandPrep()
	invoked := []string{}
	core.RequireTrue(t, c.Command("flow/first", core.Command{Action: func(_ core.Options) core.Result {
		invoked = append(invoked, "first")
		return core.Result{OK: true}
	}}).OK)
	core.RequireTrue(t, c.Command("flow/fail", core.Command{Action: func(_ core.Options) core.Result {
		invoked = append(invoked, "second")
		return core.Result{Value: "boom", OK: false}
	}}).OK)
	core.RequireTrue(t, c.Command("flow/third", core.Command{Action: func(_ core.Options) core.Result {
		invoked = append(invoked, "third")
		return core.Result{OK: true}
	}}).OK)

	output := captureStdout(t, func() {
		r := s.cmdRunFlow(core.NewOptions(core.Option{Key: "_arg", Value: filePath}))
		core.RequireTrue(t, r.OK)

		flowOutput, ok := r.Value.(FlowRunOutput)
		core.RequireTrue(t, ok)
		core.AssertTrue(t, flowOutput.Success)
		core.AssertEqual(t, 3, flowOutput.Executed)
		core.AssertEqual(t, 2, flowOutput.Passed)
		core.AssertEqual(t, 1, flowOutput.Failed)
		core.AssertLen(t, flowOutput.StepResults, 3)
		core.AssertTrue(t, flowOutput.StepResults[1].ContinueOnError)
		core.AssertContains(t, flowOutput.StepResults[1].Error, "boom")
	})

	core.AssertEqual(t, []string{"first", "second", "third"}, invoked)
	core.AssertContains(t, output, "second: failed (continued)")
	core.AssertContains(t, output, "third: passed")
	core.AssertContains(t, output, "totals: ran=3 passed=2 failed=1")
}

func TestCommandsFlow_CmdRunFlow_Good_EmptyFlowIsNoop(t *testing.T) {
	dir := t.TempDir()
	filePath := core.JoinPath(dir, "empty-flow.yaml")
	core.RequireTrue(t, fs.Write(filePath, core.Concat(
		"name: Empty Flow\n",
		"steps: []\n",
	)).OK)

	s, _ := newFlowCommandPrep()
	output := captureStdout(t, func() {
		r := s.cmdRunFlow(core.NewOptions(core.Option{Key: "_arg", Value: filePath}))
		core.RequireTrue(t, r.OK)

		flowOutput, ok := r.Value.(FlowRunOutput)
		core.RequireTrue(t, ok)
		core.AssertTrue(t, flowOutput.Success)
		core.AssertEqual(t, 0, flowOutput.Steps)
		core.AssertEqual(t, 0, flowOutput.Executed)
		core.AssertEqual(t, 0, flowOutput.Passed)
		core.AssertEqual(t, 0, flowOutput.Failed)
		core.AssertEmpty(t, flowOutput.StepResults)
	})

	core.AssertContains(t, output, "steps: 0")
	core.AssertContains(t, output, "totals: ran=0 passed=0 failed=0")
}

func TestCommandsFlow_CmdFlowPreview_Good_VariablesAlias(t *testing.T) {
	root := t.TempDir()
	flowPath := core.JoinPath(root, "preview.yaml")
	fs.Write(flowPath, ""+
		"name: \"{{NAME}} deployment\"\n"+
		"description: \"Preview flow\"\n"+
		"steps:\n"+
		"  - name: \"{{STEP}}\"\n"+
		"    run: \"echo {{VALUE}}\"\n",
	)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(core.New(), AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	output := captureStdout(t, func() {
		r := s.cmdFlowPreview(core.NewOptions(
			core.Option{Key: "_arg", Value: flowPath},
			core.Option{Key: "variables", Value: map[string]any{
				"NAME":  "release",
				"STEP":  "lint",
				"VALUE": "ok",
			}},
		))
		core.AssertTrue(t, r.OK)
	})

	core.AssertContains(t, output, "name:  release deployment")
	core.AssertContains(t, output, "1. lint")
}

func TestCommandsFlow_CmdFlowPreview_Bad_MissingPath(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(core.New(), AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	r := s.cmdFlowPreview(core.NewOptions())
	core.AssertFalse(t, r.OK)
}

func TestCommandsFlow_CmdFlowPreview_Ugly_InvalidYaml(t *testing.T) {
	root := t.TempDir()
	flowPath := core.JoinPath(root, "broken.yaml")
	fs.Write(flowPath, "name: [broken\nsteps:\n  - name: test\n")

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(core.New(), AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	r := s.cmdFlowPreview(core.NewOptions(core.Option{Key: "_arg", Value: flowPath}))
	core.AssertFalse(t, r.OK)
}
