// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestCommandsWorkspaceCov_CmdWorkspaceDispatch_Good_PrintsHumanOutput overrides
// the injectable dispatch seam to succeed, exercising the human-readable output
// branch (dispatched / workspace / pid lines).
func TestCommandsWorkspaceCov_CmdWorkspaceDispatch_Good_PrintsHumanOutput(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	original := dispatch
	t.Cleanup(func() { dispatch = original })
	var gotInput DispatchInput
	dispatch = func(_ *PrepSubsystem, _ context.Context, _ *mcp.CallToolRequest, input DispatchInput) (*mcp.CallToolResult, DispatchOutput, error) {
		gotInput = input
		return nil, DispatchOutput{
			Success:      true,
			Agent:        "codex",
			Repo:         input.Repo,
			WorkspaceDir: "/tmp/ws/core/go-io/task-42",
			PID:          4321,
		}, nil
	}

	s := newTestPrep(t)
	var r core.Result
	output := captureStdout(t, func() {
		r = s.cmdWorkspaceDispatch(core.NewOptions(
			core.Option{Key: "_arg", Value: "go-io"},
			core.Option{Key: "task", Value: "fix it"},
			core.Option{Key: "branch", Value: "dev"},
		))
	})
	core.RequireTrue(t, r.OK)
	core.AssertEqual(t, "go-io", gotInput.Repo)
	core.AssertContains(t, output, "dispatched codex to go-io")
	core.AssertContains(t, output, "workspace: /tmp/ws/core/go-io/task-42")
	core.AssertContains(t, output, "pid:       4321")
}

// TestCommandsWorkspaceCov_CmdWorkspaceDispatch_Good_JSONOutput exercises the
// --json branch which prints the marshalled DispatchOutput.
func TestCommandsWorkspaceCov_CmdWorkspaceDispatch_Good_JSONOutput(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	original := dispatch
	t.Cleanup(func() { dispatch = original })
	dispatch = func(_ *PrepSubsystem, _ context.Context, _ *mcp.CallToolRequest, input DispatchInput) (*mcp.CallToolResult, DispatchOutput, error) {
		return nil, DispatchOutput{Success: true, Agent: "codex", Repo: input.Repo, WorkspaceDir: "/tmp/ws"}, nil
	}

	s := newTestPrep(t)
	var r core.Result
	output := captureStdout(t, func() {
		r = s.cmdWorkspaceDispatch(core.NewOptions(
			core.Option{Key: "_arg", Value: "go-io"},
			core.Option{Key: "task", Value: "fix it"},
			core.Option{Key: "json", Value: true},
		))
	})
	core.RequireTrue(t, r.OK)
	core.AssertContains(t, output, `"agent":"codex"`)
	core.AssertContains(t, output, `"workspace_dir":"/tmp/ws"`)
}

// TestCommandsWorkspaceCov_CmdWorkspaceDispatch_Bad_MissingRepo — no repo prints
// usage and returns the required-field error.
func TestCommandsWorkspaceCov_CmdWorkspaceDispatch_Bad_MissingRepo(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	var r core.Result
	output := captureStdout(t, func() { r = s.cmdWorkspaceDispatch(core.NewOptions()) })
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Value.(error).Error(), "repo is required")
	core.AssertContains(t, output, "usage: core-agent workspace dispatch")
}

// TestCommandsWorkspaceCov_CmdWorkspaceDispatch_Ugly_DispatchFails overrides the
// dispatch seam to fail, exercising the failure-output arm.
func TestCommandsWorkspaceCov_CmdWorkspaceDispatch_Ugly_DispatchFails(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	original := dispatch
	t.Cleanup(func() { dispatch = original })
	dispatch = func(_ *PrepSubsystem, _ context.Context, _ *mcp.CallToolRequest, _ DispatchInput) (*mcp.CallToolResult, DispatchOutput, error) {
		return nil, DispatchOutput{}, core.E("agentic.dispatch", "clone failed", nil)
	}

	s := newTestPrep(t)
	var r core.Result
	output := captureStdout(t, func() {
		r = s.cmdWorkspaceDispatch(core.NewOptions(
			core.Option{Key: "_arg", Value: "go-io"},
			core.Option{Key: "task", Value: "fix it"},
		))
	})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, output, "dispatch failed:")
	core.AssertContains(t, output, "clone failed")
}

// TestCommandsWorkspaceCov_CmdWorkspaceWatch_Ugly_WatchFails overrides the watch
// seam to fail, exercising the error arm of cmdWorkspaceWatch.
func TestCommandsWorkspaceCov_CmdWorkspaceWatch_Ugly_WatchFails(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	original := watch
	t.Cleanup(func() { watch = original })
	watch = func(_ *PrepSubsystem, _ context.Context, _ *mcp.CallToolRequest, _ WatchInput) (*mcp.CallToolResult, WatchOutput, error) {
		return nil, WatchOutput{}, core.E("agentic.watch", "watch aborted", nil)
	}

	s := newTestPrep(t)
	var r core.Result
	output := captureStdout(t, func() {
		r = s.cmdWorkspaceWatch(core.NewOptions(core.Option{Key: "_arg", Value: "core/go-io/task-1"}))
	})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, output, "error:")
}

// TestCommandsWorkspaceCov_CmdWorkspaceWatch_Good_JSONOutput exercises the --json
// branch of cmdWorkspaceWatch via the injectable watch seam.
func TestCommandsWorkspaceCov_CmdWorkspaceWatch_Good_JSONOutput(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	original := watch
	t.Cleanup(func() { watch = original })
	watch = func(_ *PrepSubsystem, _ context.Context, _ *mcp.CallToolRequest, input WatchInput) (*mcp.CallToolResult, WatchOutput, error) {
		return nil, WatchOutput{
			Success:   true,
			Completed: []WatchResult{{Workspace: "core/go-io/task-1"}},
			Duration:  "1s",
		}, nil
	}

	s := newTestPrep(t)
	var r core.Result
	output := captureStdout(t, func() {
		r = s.cmdWorkspaceWatch(core.NewOptions(
			core.Option{Key: "_arg", Value: "core/go-io/task-1"},
			core.Option{Key: "json", Value: true},
		))
	})
	core.RequireTrue(t, r.OK)
	core.AssertContains(t, output, `"success":true`)
	core.AssertContains(t, output, "core/go-io/task-1")
}

// TestCommandsWorkspaceCov_CmdWorkspaceWatch_Good_HumanOutput exercises the
// human-readable completed/failed/duration print lines.
func TestCommandsWorkspaceCov_CmdWorkspaceWatch_Good_HumanOutput(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	original := watch
	t.Cleanup(func() { watch = original })
	watch = func(_ *PrepSubsystem, _ context.Context, _ *mcp.CallToolRequest, _ WatchInput) (*mcp.CallToolResult, WatchOutput, error) {
		return nil, WatchOutput{
			Success:   true,
			Completed: []WatchResult{{Workspace: "core/go-io/task-1"}},
			Failed:    []WatchResult{{Workspace: "core/go-io/task-2"}},
			Duration:  "3s",
		}, nil
	}

	s := newTestPrep(t)
	var r core.Result
	output := captureStdout(t, func() {
		r = s.cmdWorkspaceWatch(core.NewOptions(core.Option{Key: "workspace", Value: "core/go-io/task-1"}))
	})
	core.RequireTrue(t, r.OK)
	core.AssertContains(t, output, "completed: 1")
	core.AssertContains(t, output, "failed:    1")
	core.AssertContains(t, output, "duration:  3s")
}

// TestCommandsWorkspaceCov_RegisterWorkspaceCommands_Ugly_DuplicateConflict — a
// second registration fails on the first duplicate command.
func TestCommandsWorkspaceCov_RegisterWorkspaceCommands_Ugly_DuplicateConflict(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	core.RequireTrue(t, s.registerWorkspaceCommands().OK)
	core.AssertFalse(t, s.registerWorkspaceCommands().OK)
}
