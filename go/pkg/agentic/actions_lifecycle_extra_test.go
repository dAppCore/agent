// SPDX-License-Identifier: EUPL-1.2

// Happy-path + enabled-leg coverage for the agentic action handlers whose
// side-effecting cores (prep / watch / review-queue / epic) reach docker,
// a real forge, or a spawned dispatch loop. Each handler's underlying op is
// already a package var, so we override it, defer-restore, drive the wrapper
// with options, and assert the OK envelope — no real forge, git, or process.
//
// handleResume is driven through a real blocked-workspace fixture (status.json
// + a git repo dir) with DryRun, so the wrapper's option-mapping and the
// resume body both run without spawning an agent.

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestActions_HandlePrep_Good_MockedPrep — handlePrep maps options to a
// PrepInput, calls the (mocked) prepWorkspace op, and surfaces its output.
func TestActions_HandlePrep_Good_MockedPrep(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx := context.Background()

	orig := prepWorkspace
	defer func() { prepWorkspace = orig }()
	prepWorkspace = func(_ *PrepSubsystem, _ context.Context, _ *mcp.CallToolRequest, input PrepInput) (*mcp.CallToolResult, PrepOutput, error) {
		// Verify the wrapper mapped the option through to the input.
		core.AssertEqual(t, "go-io", input.Repo)
		return nil, PrepOutput{Success: true, WorkspaceDir: "core/go-io/task-1", Branch: "agent/x"}, nil
	}

	r := s.handlePrep(ctx, core.NewOptions(core.Option{Key: "repo", Value: "go-io"}))
	core.AssertTrue(t, r.OK)
	out, ok := r.Value.(PrepOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, out.Success)
	core.AssertEqual(t, "agent/x", out.Branch)
}

// TestActions_HandlePrep_Bad_PrepErrors — when prepWorkspace returns an
// error the handler propagates a failed Result carrying it.
func TestActions_HandlePrep_Bad_PrepErrors(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx := context.Background()

	orig := prepWorkspace
	defer func() { prepWorkspace = orig }()
	prepWorkspace = func(_ *PrepSubsystem, _ context.Context, _ *mcp.CallToolRequest, _ PrepInput) (*mcp.CallToolResult, PrepOutput, error) {
		return nil, PrepOutput{}, core.E("agentic.prep", "boom", nil)
	}

	r := s.handlePrep(ctx, core.NewOptions(core.Option{Key: "repo", Value: "go-io"}))
	core.AssertFalse(t, r.OK)
}

// TestActions_HandleWatch_Good_MockedWatch — handleWatch builds a WatchInput
// from the workspace option and returns the mocked watch output.
func TestActions_HandleWatch_Good_MockedWatch(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx := context.Background()

	orig := watch
	defer func() { watch = orig }()
	watch = func(_ *PrepSubsystem, _ context.Context, _ *mcp.CallToolRequest, input WatchInput) (*mcp.CallToolResult, WatchOutput, error) {
		core.AssertEqual(t, 1, len(input.Workspaces))
		core.AssertEqual(t, "core/go-io/task-5", input.Workspaces[0])
		return nil, WatchOutput{}, nil
	}

	r := s.handleWatch(ctx, core.NewOptions(core.Option{Key: "workspace", Value: "core/go-io/task-5"}))
	core.AssertTrue(t, r.OK)
}

// TestActions_HandleWatch_Bad_WatchErrors — a watch error surfaces as a
// failed Result.
func TestActions_HandleWatch_Bad_WatchErrors(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx := context.Background()

	orig := watch
	defer func() { watch = orig }()
	watch = func(_ *PrepSubsystem, _ context.Context, _ *mcp.CallToolRequest, _ WatchInput) (*mcp.CallToolResult, WatchOutput, error) {
		return nil, WatchOutput{}, core.E("agentic.watch", "watch failed", nil)
	}

	r := s.handleWatch(ctx, core.NewOptions(core.Option{Key: "workspace", Value: "core/go-io/task-5"}))
	core.AssertFalse(t, r.OK)
}

// TestActions_HandleReviewQueue_Good_MockedQueue — handleReviewQueue maps the
// options to a ReviewQueueInput, calls the mocked reviewQueue op, returns OK.
func TestActions_HandleReviewQueue_Good_MockedQueue(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx := context.Background()

	orig := reviewQueue
	defer func() { reviewQueue = orig }()
	reviewQueue = func(_ *PrepSubsystem, _ context.Context, _ *mcp.CallToolRequest, input ReviewQueueInput) (*mcp.CallToolResult, ReviewQueueOutput, error) {
		// Verify the wrapper mapped the reviewer + limit options through.
		core.AssertEqual(t, "cerberus", input.Reviewer)
		core.AssertEqual(t, 5, input.Limit)
		return nil, ReviewQueueOutput{Success: true}, nil
	}

	r := s.handleReviewQueue(ctx, core.NewOptions(
		core.Option{Key: "reviewer", Value: "cerberus"},
		core.Option{Key: "limit", Value: 5},
	))
	core.AssertTrue(t, r.OK)
	out, ok := r.Value.(ReviewQueueOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, out.Success)
}

// TestActions_HandleReviewQueue_Bad_QueueErrors — review-queue errors
// surface as a failed Result.
func TestActions_HandleReviewQueue_Bad_QueueErrors(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx := context.Background()

	orig := reviewQueue
	defer func() { reviewQueue = orig }()
	reviewQueue = func(_ *PrepSubsystem, _ context.Context, _ *mcp.CallToolRequest, _ ReviewQueueInput) (*mcp.CallToolResult, ReviewQueueOutput, error) {
		return nil, ReviewQueueOutput{}, core.E("agentic.review-queue", "queue failed", nil)
	}

	r := s.handleReviewQueue(ctx, core.NewOptions(core.Option{Key: "repo", Value: "go-io"}))
	core.AssertFalse(t, r.OK)
}

// TestActions_HandleEpicAndCmdEpic_Good_MockedEpic — both the action handler
// and the cmd wrapper map options to an EpicInput and return the mocked epic
// output. cmdEpic delegates to handleEpic via s.commandContext().
func TestActions_HandleEpicAndCmdEpic_Good_MockedEpic(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx := context.Background()

	orig := createEpic
	defer func() { createEpic = orig }()
	createEpic = func(_ *PrepSubsystem, _ context.Context, _ *mcp.CallToolRequest, input EpicInput) (*mcp.CallToolResult, EpicOutput, error) {
		core.AssertEqual(t, "Stabilise dispatch", input.Title)
		core.AssertEqual(t, "go-io", input.Repo)
		return nil, EpicOutput{Success: true, EpicNumber: 7}, nil
	}

	opts := core.NewOptions(
		core.Option{Key: "title", Value: "Stabilise dispatch"},
		core.Option{Key: "repo", Value: "go-io"},
	)
	core.AssertTrue(t, s.handleEpic(ctx, opts).OK)
	core.AssertTrue(t, s.cmdEpic(opts).OK)
}

// TestActions_HandleEpic_Bad_EpicErrors — epic creation errors surface as a
// failed Result.
func TestActions_HandleEpic_Bad_EpicErrors(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx := context.Background()

	orig := createEpic
	defer func() { createEpic = orig }()
	createEpic = func(_ *PrepSubsystem, _ context.Context, _ *mcp.CallToolRequest, _ EpicInput) (*mcp.CallToolResult, EpicOutput, error) {
		return nil, EpicOutput{}, core.E("agentic.epic", "epic failed", nil)
	}

	r := s.handleEpic(ctx, core.NewOptions(core.Option{Key: "title", Value: "x"}))
	core.AssertFalse(t, r.OK)
}

// TestActions_HandleResume_Good_DryRunWrapper — handleResume maps the options
// to a ResumeInput and runs the resume body against a real blocked-workspace
// fixture in DryRun mode (no agent spawned). Covers the wrapper end-to-end.
func TestActions_HandleResume_Good_DryRunWrapper(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	wsRoot := WorkspaceRoot()
	ws := core.JoinPath(wsRoot, "ws-blocked")
	repoDir := core.JoinPath(ws, "repo")
	fs.EnsureDir(repoDir)
	testCore.Process().Run(context.Background(), "git", "init", repoDir)

	st := &WorkspaceStatus{Status: "blocked", Repo: "go-io", Agent: "codex", Task: "Fix the queue"}
	fs.Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(st))

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), backoff: make(map[string]time.Time), failCount: make(map[string]int)}

	r := s.handleResume(context.Background(), core.NewOptions(
		core.Option{Key: "workspace", Value: "ws-blocked"},
		core.Option{Key: "answer", Value: "Use the new queue config"},
		core.Option{Key: "dry_run", Value: true},
	))
	core.RequireTrue(t, r.OK)
	out, ok := r.Value.(ResumeOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, out.Success)
	core.AssertEqual(t, "codex", out.Agent)
	core.AssertContains(t, out.Prompt, "Fix the queue")
}

// TestActions_HandleResume_Bad_MissingWorkspace — the wrapper surfaces the
// resume body's typed failure when no workspace is given.
func TestActions_HandleResume_Bad_MissingWorkspace(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	r := s.handleResume(context.Background(), core.NewOptions())
	core.AssertFalse(t, r.OK)
}

// TestActions_HandleAutoPR_Good_EnabledEarlyReturn — with auto-pr enabled and a
// status fixture that has no branch, handleAutoPR runs its body (autoCreatePR
// early-returns on the empty branch before any git/forge call) and the
// post-action block (PRURL empty → no ACTION emitted), returning OK.
func TestActions_HandleAutoPR_Good_EnabledEarlyReturn(t *testing.T) {
	s, c := testPrepWithCore(t, nil)
	c.Config().Enable("auto-pr")

	root := t.TempDir()
	setTestWorkspace(t, root)
	ws := core.JoinPath(WorkspaceRoot(), "ws-ap")
	fs.EnsureDir(ws)
	// Status with no branch → autoCreatePR returns before touching git.
	st := &WorkspaceStatus{Status: "completed", Repo: "go-io", Agent: "codex"}
	fs.Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(st))

	captureStdout(t, func() {
		core.AssertTrue(t, s.handleAutoPR(context.Background(), core.NewOptions(
			core.Option{Key: "workspace", Value: ws},
		)).OK)
	})
}

// TestActions_HandleAutoPR_Bad_NoWorkspace — auto-pr enabled but no workspace
// option → typed failure.
func TestActions_HandleAutoPR_Bad_NoWorkspace(t *testing.T) {
	s, c := testPrepWithCore(t, nil)
	c.Config().Enable("auto-pr")
	r := s.handleAutoPR(context.Background(), core.NewOptions())
	core.AssertFalse(t, r.OK)
}

// TestActions_HandleVerify_Good_EnabledEarlyReturn — with auto-merge enabled and
// a status fixture with no branch, handleVerify runs its body (autoVerifyAndMerge
// early-returns) plus the post-action status read, returning OK.
func TestActions_HandleVerify_Good_EnabledEarlyReturn(t *testing.T) {
	s, c := testPrepWithCore(t, nil)
	c.Config().Enable("auto-merge")

	root := t.TempDir()
	setTestWorkspace(t, root)
	ws := core.JoinPath(WorkspaceRoot(), "ws-vf")
	fs.EnsureDir(ws)
	st := &WorkspaceStatus{Status: "completed", Repo: "go-io", Agent: "codex"}
	fs.Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(st))

	captureStdout(t, func() {
		core.AssertTrue(t, s.handleVerify(context.Background(), core.NewOptions(
			core.Option{Key: "workspace", Value: ws},
		)).OK)
	})
}

// TestActions_CompleteTool_Bad_MissingWorkspace — completeTool short-circuits
// with a typed failure when the workspace is empty (before any task run).
func TestActions_CompleteTool_Bad_MissingWorkspace(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.completeTool(context.Background(), CompleteInput{})
	core.AssertFalse(t, r.OK)
}
