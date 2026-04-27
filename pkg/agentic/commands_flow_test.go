// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"os"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.True(t, fs.EnsureDir(flowPath).OK)

	filePath := core.JoinPath(flowPath, "go-qa.yaml")
	require.True(t, fs.Write(filePath, core.Concat(
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
		require.True(t, r.OK)

		flowOutput, ok := r.Value.(FlowRunOutput)
		require.True(t, ok)
		assert.True(t, flowOutput.Success)
		assert.Equal(t, filePath, flowOutput.Source)
		assert.Equal(t, "Go QA", flowOutput.Name)
		assert.Equal(t, "Build and test a Go project", flowOutput.Description)
		assert.Equal(t, 2, flowOutput.Steps)
	})

	assert.Contains(t, output, "steps: 2")
	assert.Contains(t, output, "build: run go build ./...")
	assert.Contains(t, output, "verify: flow verify/go-qa.yaml")
}

func TestCommandsFlow_CmdRunFlow_Good_ExecutesSequentialSteps(t *testing.T) {
	dir := t.TempDir()
	filePath := core.JoinPath(dir, "execute-flow.yaml")
	require.True(t, fs.Write(filePath, core.Concat(
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
	require.True(t, c.Command("flow/test-first", core.Command{Action: func(_ core.Options) core.Result {
		invoked = append(invoked, "first")
		core.Print(nil, "first stdout")
		return core.Result{OK: true}
	}}).OK)
	require.True(t, c.Command("flow/test-second", core.Command{Action: func(options core.Options) core.Result {
		invoked = append(invoked, "second")
		assert.Equal(t, "fast", options.String("mode"))
		_, _ = os.Stderr.WriteString("second stderr\n")
		return core.Result{OK: true}
	}}).OK)
	require.True(t, c.Command("flow/test-third", core.Command{Action: func(options core.Options) core.Result {
		invoked = append(invoked, "third")
		assert.Equal(t, "payload", options.String("_arg"))
		return core.Result{OK: true}
	}}).OK)

	output := captureStdout(t, func() {
		r := s.cmdRunFlow(core.NewOptions(core.Option{Key: "_arg", Value: filePath}))
		require.True(t, r.OK)

		flowOutput, ok := r.Value.(FlowRunOutput)
		require.True(t, ok)
		assert.True(t, flowOutput.Success)
		assert.Equal(t, filePath, flowOutput.Source)
		assert.Equal(t, "Execute Flow", flowOutput.Name)
		assert.Equal(t, "Run registered commands", flowOutput.Description)
		assert.Equal(t, 3, flowOutput.Steps)
		assert.Equal(t, 3, flowOutput.Executed)
		assert.Equal(t, 3, flowOutput.Passed)
		assert.Equal(t, 0, flowOutput.Failed)
		require.Len(t, flowOutput.StepResults, 3)
		assert.Equal(t, "first", flowOutput.StepResults[0].Name)
		assert.Equal(t, "flow/test-second", flowOutput.StepResults[1].Command)
		assert.Equal(t, "second stderr\n", flowOutput.StepResults[1].Stderr)
	})

	assert.Equal(t, []string{"first", "second", "third"}, invoked)
	assert.Contains(t, output, "steps: 3")
	assert.Contains(t, output, "first stdout")
	assert.Contains(t, output, "second stderr")
	assert.Contains(t, output, "totals: ran=3 passed=3 failed=0")
}

func TestCommandsFlow_CmdRunFlow_Good_RendersVariablesAndDryRun(t *testing.T) {
	dir := t.TempDir()
	flowPath := core.JoinPath(dir, "pkg", "lib", "flow", "verify")
	require.True(t, fs.EnsureDir(flowPath).OK)

	filePath := core.JoinPath(flowPath, "go-qa.yaml")
	require.True(t, fs.Write(filePath, core.Concat(
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
		require.True(t, r.OK)

		flowOutput, ok := r.Value.(FlowRunOutput)
		require.True(t, ok)
		assert.True(t, flowOutput.Success)
		assert.Equal(t, "Go QA", flowOutput.Name)
		assert.Equal(t, "Build core/go", flowOutput.Description)
		assert.Equal(t, 1, flowOutput.Steps)
	})

	assert.Contains(t, output, "dry-run: true")
	assert.Contains(t, output, "vars: 1")
	assert.Contains(t, output, "desc:  Build core/go")
	assert.Contains(t, output, "build: cmd flow/test-build --repo=core/go")
}

func TestCommandsFlow_CmdFlowPreview_Good_ResolvesNestedFlowReferences(t *testing.T) {
	dir := t.TempDir()
	flowRoot := core.JoinPath(dir, "pkg", "lib", "flow")
	require.True(t, fs.EnsureDir(core.JoinPath(flowRoot, "verify")).OK)

	rootPath := core.JoinPath(flowRoot, "root.yaml")
	require.True(t, fs.Write(rootPath, core.Concat(
		"name: Root Flow\n",
		"description: Resolve nested flow references\n",
		"steps:\n",
		"  - name: child\n",
		"    flow: verify/go-qa.yaml\n",
	)).OK)

	childPath := core.JoinPath(flowRoot, "verify", "go-qa.yaml")
	require.True(t, fs.Write(childPath, core.Concat(
		"name: Child Flow\n",
		"description: Nested flow body\n",
		"steps:\n",
		"  - name: child-run\n",
		"    run: echo child\n",
	)).OK)

	s := newTestPrep(t)
	output := captureStdout(t, func() {
		r := s.cmdFlowPreview(core.NewOptions(core.Option{Key: "_arg", Value: rootPath}))
		require.True(t, r.OK)

		flowOutput, ok := r.Value.(FlowRunOutput)
		require.True(t, ok)
		assert.True(t, flowOutput.Success)
		assert.Equal(t, 1, flowOutput.Steps)
		assert.Equal(t, 2, flowOutput.ResolvedSteps)
	})

	assert.Contains(t, output, "resolved:")
	assert.Contains(t, output, "child-run: run echo child")
}

func TestCommandsFlow_CmdRunFlow_Bad_MissingPath(t *testing.T) {
	s := newTestPrep(t)

	r := s.cmdRunFlow(core.NewOptions())
	require.False(t, r.OK)

	err, ok := r.Value.(error)
	require.True(t, ok)
	assert.Contains(t, err.Error(), "flow path or slug is required")
}

func TestCommandsFlow_CmdRunFlow_Bad_RejectsUnknownCommandAtParseTime(t *testing.T) {
	dir := t.TempDir()
	filePath := core.JoinPath(dir, "missing-command.yaml")
	require.True(t, fs.Write(filePath, core.Concat(
		"name: Missing Command\n",
		"steps:\n",
		"  - name: first\n",
		"    cmd: flow/known\n",
		"  - name: second\n",
		"    cmd: flow/missing\n",
	)).OK)

	s, c := newFlowCommandPrep()
	invoked := []string{}
	require.True(t, c.Command("flow/known", core.Command{Action: func(_ core.Options) core.Result {
		invoked = append(invoked, "known")
		return core.Result{OK: true}
	}}).OK)

	r := s.cmdRunFlow(core.NewOptions(core.Option{Key: "_arg", Value: filePath}))
	require.False(t, r.OK)

	err, ok := r.Value.(error)
	require.True(t, ok)
	assert.Contains(t, err.Error(), "references unknown command: flow/missing")
	assert.Empty(t, invoked)
}

func TestCommandsFlow_CmdRunFlow_Ugly_InvalidYaml(t *testing.T) {
	dir := t.TempDir()
	filePath := core.JoinPath(dir, "broken-flow.yaml")
	require.True(t, fs.Write(filePath, "name: [broken\n").OK)

	s := newTestPrep(t)
	r := s.cmdRunFlow(core.NewOptions(core.Option{Key: "_arg", Value: filePath}))
	require.False(t, r.OK)

	err, ok := r.Value.(error)
	require.True(t, ok)
	assert.Contains(t, err.Error(), "invalid flow definition")
}

func TestCommandsFlow_CmdRunFlow_Ugly_StopsOnFirstFailure(t *testing.T) {
	dir := t.TempDir()
	filePath := core.JoinPath(dir, "stops-on-failure.yaml")
	require.True(t, fs.Write(filePath, core.Concat(
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
	require.True(t, c.Command("flow/first", core.Command{Action: func(_ core.Options) core.Result {
		invoked = append(invoked, "first")
		return core.Result{OK: true}
	}}).OK)
	require.True(t, c.Command("flow/fail", core.Command{Action: func(_ core.Options) core.Result {
		invoked = append(invoked, "second")
		return core.Result{Value: "boom", OK: false}
	}}).OK)
	require.True(t, c.Command("flow/third", core.Command{Action: func(_ core.Options) core.Result {
		invoked = append(invoked, "third")
		return core.Result{OK: true}
	}}).OK)

	output := captureStdout(t, func() {
		r := s.cmdRunFlow(core.NewOptions(core.Option{Key: "_arg", Value: filePath}))
		require.False(t, r.OK)

		flowOutput, ok := r.Value.(FlowRunOutput)
		require.True(t, ok)
		assert.False(t, flowOutput.Success)
		assert.Equal(t, 2, flowOutput.Executed)
		assert.Equal(t, 1, flowOutput.Passed)
		assert.Equal(t, 1, flowOutput.Failed)
		require.Len(t, flowOutput.StepResults, 2)
		assert.Equal(t, "boom", flowOutput.StepResults[1].Error)
	})

	assert.Equal(t, []string{"first", "second"}, invoked)
	assert.Contains(t, output, "second: failed")
	assert.Contains(t, output, "totals: ran=2 passed=1 failed=1")
	assert.NotContains(t, output, "third: passed")
}

func TestCommandsFlow_CmdRunFlow_Ugly_ContinueOnErrorRunsRemainingSteps(t *testing.T) {
	dir := t.TempDir()
	filePath := core.JoinPath(dir, "continue-on-error.yaml")
	require.True(t, fs.Write(filePath, core.Concat(
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
	require.True(t, c.Command("flow/first", core.Command{Action: func(_ core.Options) core.Result {
		invoked = append(invoked, "first")
		return core.Result{OK: true}
	}}).OK)
	require.True(t, c.Command("flow/fail", core.Command{Action: func(_ core.Options) core.Result {
		invoked = append(invoked, "second")
		return core.Result{Value: "boom", OK: false}
	}}).OK)
	require.True(t, c.Command("flow/third", core.Command{Action: func(_ core.Options) core.Result {
		invoked = append(invoked, "third")
		return core.Result{OK: true}
	}}).OK)

	output := captureStdout(t, func() {
		r := s.cmdRunFlow(core.NewOptions(core.Option{Key: "_arg", Value: filePath}))
		require.True(t, r.OK)

		flowOutput, ok := r.Value.(FlowRunOutput)
		require.True(t, ok)
		assert.True(t, flowOutput.Success)
		assert.Equal(t, 3, flowOutput.Executed)
		assert.Equal(t, 2, flowOutput.Passed)
		assert.Equal(t, 1, flowOutput.Failed)
		require.Len(t, flowOutput.StepResults, 3)
		assert.True(t, flowOutput.StepResults[1].ContinueOnError)
		assert.Equal(t, "boom", flowOutput.StepResults[1].Error)
	})

	assert.Equal(t, []string{"first", "second", "third"}, invoked)
	assert.Contains(t, output, "second: failed (continued)")
	assert.Contains(t, output, "third: passed")
	assert.Contains(t, output, "totals: ran=3 passed=2 failed=1")
}

func TestCommandsFlow_CmdRunFlow_Good_EmptyFlowIsNoop(t *testing.T) {
	dir := t.TempDir()
	filePath := core.JoinPath(dir, "empty-flow.yaml")
	require.True(t, fs.Write(filePath, core.Concat(
		"name: Empty Flow\n",
		"steps: []\n",
	)).OK)

	s, _ := newFlowCommandPrep()
	output := captureStdout(t, func() {
		r := s.cmdRunFlow(core.NewOptions(core.Option{Key: "_arg", Value: filePath}))
		require.True(t, r.OK)

		flowOutput, ok := r.Value.(FlowRunOutput)
		require.True(t, ok)
		assert.True(t, flowOutput.Success)
		assert.Equal(t, 0, flowOutput.Steps)
		assert.Equal(t, 0, flowOutput.Executed)
		assert.Equal(t, 0, flowOutput.Passed)
		assert.Equal(t, 0, flowOutput.Failed)
		assert.Empty(t, flowOutput.StepResults)
	})

	assert.Contains(t, output, "steps: 0")
	assert.Contains(t, output, "totals: ran=0 passed=0 failed=0")
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
		assert.True(t, r.OK)
	})

	assert.Contains(t, output, "name:  release deployment")
	assert.Contains(t, output, "1. lint")
}

func TestCommandsFlow_CmdFlowPreview_Bad_MissingPath(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(core.New(), AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	r := s.cmdFlowPreview(core.NewOptions())
	assert.False(t, r.OK)
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
	assert.False(t, r.OK)
}
