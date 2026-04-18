// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandsFlow_CmdRunFlow_Good_ReadsYamlFlowFile(t *testing.T) {
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
		r := s.cmdRunFlow(core.NewOptions(core.Option{Key: "_arg", Value: filePath}))
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
		"    run: go build ./...\n",
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
}

func TestCommandsFlow_CmdRunFlow_Good_ResolvesNestedFlowReferences(t *testing.T) {
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
		r := s.cmdRunFlow(core.NewOptions(core.Option{Key: "_arg", Value: rootPath}))
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
