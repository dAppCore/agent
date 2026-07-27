// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestCommandsPrepCov_CmdPrep_Good_PrintsAllFields overrides the injectable
// PrepareWorkspace seam to succeed with a fully-populated output, exercising the
// human-readable print block (workspace/repo/branch/prompt-version/resumed/
// memories/consumers + the prompt dump).
func TestCommandsPrepCov_CmdPrep_Good_PrintsAllFields(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	original := PrepareWorkspace
	t.Cleanup(func() { PrepareWorkspace = original })
	var gotInput PrepInput
	PrepareWorkspace = func(_ *PrepSubsystem, _ context.Context, input PrepInput) (*mcp.CallToolResult, PrepOutput, error) {
		gotInput = input
		return nil, PrepOutput{
			Success:       true,
			WorkspaceDir:  "/tmp/ws/core/go-io/task-42",
			RepoDir:       "/tmp/ws/core/go-io/task-42/repo",
			Branch:        "dev",
			PromptVersion: "abc123",
			Prompt:        "TASK: fix the build",
			Memories:      3,
			Consumers:     2,
			Resumed:       true,
		}, nil
	}

	s := newTestPrep(t)
	var r core.Result
	output := captureStdout(t, func() {
		r = s.cmdPrep(core.NewOptions(
			core.Option{Key: "_arg", Value: "go-io"},
			core.Option{Key: "task", Value: "fix the build"},
			core.Option{Key: "issue", Value: "42"},
		))
	})
	core.RequireTrue(t, r.OK)
	core.AssertEqual(t, "go-io", gotInput.Repo)
	core.AssertContains(t, output, "workspace: /tmp/ws/core/go-io/task-42")
	core.AssertContains(t, output, "repo:      /tmp/ws/core/go-io/task-42/repo")
	core.AssertContains(t, output, "branch:    dev")
	core.AssertContains(t, output, "prompt:    abc123")
	core.AssertContains(t, output, "resumed:   true")
	core.AssertContains(t, output, "memories:  3")
	core.AssertContains(t, output, "consumers: 2")
	core.AssertContains(t, output, "--- prompt (19 chars) ---")
	core.AssertContains(t, output, "TASK: fix the build")
}

// TestCommandsPrepCov_CmdPrep_Good_JSONOutput exercises the --json branch which
// prints the marshalled PrepOutput instead of the human block.
func TestCommandsPrepCov_CmdPrep_Good_JSONOutput(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	original := PrepareWorkspace
	t.Cleanup(func() { PrepareWorkspace = original })
	PrepareWorkspace = func(_ *PrepSubsystem, _ context.Context, _ PrepInput) (*mcp.CallToolResult, PrepOutput, error) {
		return nil, PrepOutput{Success: true, WorkspaceDir: "/tmp/ws", Branch: "dev"}, nil
	}

	s := newTestPrep(t)
	var r core.Result
	output := captureStdout(t, func() {
		r = s.cmdPrep(core.NewOptions(
			core.Option{Key: "_arg", Value: "go-io"},
			core.Option{Key: "task", Value: "x"},
			core.Option{Key: "json", Value: true},
		))
	})
	core.RequireTrue(t, r.OK)
	core.AssertContains(t, output, `"workspace_dir":"/tmp/ws"`)
	core.AssertContains(t, output, `"branch":"dev"`)
}

// TestCommandsPrepCov_CmdPrep_Good_DefaultsBranchWhenUnspecified — with no
// issue/pr/branch/tag the input branch defaults to "dev" before dispatch.
func TestCommandsPrepCov_CmdPrep_Good_DefaultsBranchWhenUnspecified(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	original := PrepareWorkspace
	t.Cleanup(func() { PrepareWorkspace = original })
	var gotInput PrepInput
	PrepareWorkspace = func(_ *PrepSubsystem, _ context.Context, input PrepInput) (*mcp.CallToolResult, PrepOutput, error) {
		gotInput = input
		return nil, PrepOutput{Success: true, WorkspaceDir: "/tmp/ws", Branch: "dev"}, nil
	}

	s := newTestPrep(t)
	var r core.Result
	captureStdout(t, func() {
		r = s.cmdPrep(core.NewOptions(
			core.Option{Key: "_arg", Value: "go-io"},
			core.Option{Key: "task", Value: "x"},
		))
	})
	core.RequireTrue(t, r.OK)
	core.AssertEqual(t, "dev", gotInput.Branch)
}
