// SPDX-License-Identifier: EUPL-1.2

// Gating-leg coverage for cleanupWorkspaceBranch — the early-return paths that
// decide whether a workspace's agent branch is eligible for forge deletion.
// Each leg returns OK without calling the forge, so they need only a workspace
// + status.json fixture (no forge mock). The eligible path is already covered
// by branch_cleanup_test.go via createPR / cmdComplete.

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go"
)

func cleanupGatePrep(t *testing.T) *PrepSubsystem {
	t.Helper()
	return &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
}

// TestCleanupWorkspaceBranch_Gate_EmptyWorkspace — a blank workspace string is
// a no-op success.
func TestCleanupWorkspaceBranch_Gate_EmptyWorkspace(t *testing.T) {
	s := cleanupGatePrep(t)
	core.AssertTrue(t, s.cleanupWorkspaceBranch(context.Background(), "  ").OK)
}

// TestCleanupWorkspaceBranch_Gate_NonexistentDir — a workspace that resolves to
// neither an absolute dir nor a name under the workspace root is a no-op.
func TestCleanupWorkspaceBranch_Gate_NonexistentDir(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	s := cleanupGatePrep(t)
	core.AssertTrue(t, s.cleanupWorkspaceBranch(context.Background(), "no-such-workspace").OK)
}

// TestCleanupWorkspaceBranch_Gate_NoStatus — an existing workspace dir without a
// status.json is a no-op (nothing to act on).
func TestCleanupWorkspaceBranch_Gate_NoStatus(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	ws := core.JoinPath(WorkspaceRoot(), "ws-nostatus")
	core.RequireTrue(t, fs.EnsureDir(ws).OK)

	s := cleanupGatePrep(t)
	core.AssertTrue(t, s.cleanupWorkspaceBranch(context.Background(), "ws-nostatus").OK)
}

// TestCleanupWorkspaceBranch_Gate_MissingRepoOrBranch — status present but with
// no repo/branch → not eligible, no-op.
func TestCleanupWorkspaceBranch_Gate_MissingRepoOrBranch(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	ws := core.JoinPath(WorkspaceRoot(), "ws-norepo")
	core.RequireTrue(t, fs.EnsureDir(ws).OK)
	core.RequireNoError(t, writeStatus(ws, &WorkspaceStatus{Status: "completed", Agent: "codex"}))

	s := cleanupGatePrep(t)
	core.AssertTrue(t, s.cleanupWorkspaceBranch(context.Background(), "ws-norepo").OK)
}

// TestCleanupWorkspaceBranch_Gate_NotMergedNoPR — repo+branch present but the
// branch has neither a PR URL nor a merged status → not yet eligible, no-op
// (the branch is only cleaned once its PR exists / it merged).
func TestCleanupWorkspaceBranch_Gate_NotMergedNoPR(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	ws := core.JoinPath(WorkspaceRoot(), "ws-pending")
	core.RequireTrue(t, fs.EnsureDir(ws).OK)
	core.RequireNoError(t, writeStatus(ws, &WorkspaceStatus{
		Status: "completed", Repo: "go-io", Org: "core", Branch: "agent/x",
	}))

	s := cleanupGatePrep(t)
	// No forge configured — if the gate let this through, cleanupBranch would
	// fail on the missing token. OK proves the gate short-circuited first.
	core.AssertTrue(t, s.cleanupWorkspaceBranch(context.Background(), "ws-pending").OK)
}

// TestCleanupWorkspaceBranch_Gate_AbsoluteDir — the same no-op gate works when
// the workspace is passed as an absolute path rather than a name.
func TestCleanupWorkspaceBranch_Gate_AbsoluteDir(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	ws := core.JoinPath(WorkspaceRoot(), "ws-abs")
	core.RequireTrue(t, fs.EnsureDir(ws).OK)
	core.RequireNoError(t, writeStatus(ws, &WorkspaceStatus{Status: "completed", Repo: "go-io"}))

	s := cleanupGatePrep(t)
	core.AssertTrue(t, s.cleanupWorkspaceBranch(context.Background(), ws).OK)
}
