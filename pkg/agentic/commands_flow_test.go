// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

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
