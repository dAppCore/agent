// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/messages"
)

// TestRepoSyncCov_HandleRepoSyncIPC_Bad_IgnoresOtherMessage — a message that is
// not WorkspacePushed is a no-op that returns OK without touching any repo.
func TestRepoSyncCov_HandleRepoSyncIPC_Bad_IgnoresOtherMessage(t *testing.T) {
	s, c, _ := repoSyncTestPrep(t)

	result := s.handleRepoSyncIPC(c, otherIPCMessage{})
	core.AssertTrue(t, result.OK)
}

// TestRepoSyncCov_HandleRepoSyncIPC_Ugly_WarnsOnFailedSync — a WorkspacePushed
// for a missing repo drives the failure-warn branch and returns the failure.
func TestRepoSyncCov_HandleRepoSyncIPC_Ugly_WarnsOnFailedSync(t *testing.T) {
	s, c, _ := repoSyncTestPrep(t)

	result := s.handleRepoSyncIPC(c, messages.WorkspacePushed{
		Repo:   "missing-repo",
		Branch: "main",
		Org:    "core",
	})
	core.AssertFalse(t, result.OK)
}

// TestRepoSyncCov_RepoSyncContext_Good_NilFallsBackToBackground — a nil context
// is replaced with context.Background(); a live context passes through.
func TestRepoSyncCov_RepoSyncContext_Good_NilFallsBackToBackground(t *testing.T) {
	//nolint:staticcheck // SA1012: the nil Context IS the input under test —
	// the nil branch is the only reason repoSyncContext exists, and a real
	// Context here just repeats the passthrough case asserted below.
	core.AssertNotNil(t, repoSyncContext(nil))

	ctx := context.Background()
	core.AssertEqual(t, ctx, repoSyncContext(ctx))
}

// TestRepoSyncCov_CmdRepoSyncLocal_Bad_InvalidTarget — an invalid repo name
// makes target resolution fail; the command prints usage and returns a failure
// before any git work.
func TestRepoSyncCov_CmdRepoSyncLocal_Bad_InvalidTarget(t *testing.T) {
	s, _, _ := repoSyncTestPrep(t)

	var result core.Result
	output := captureStdout(t, func() {
		result = s.cmdRepoSyncLocal(core.NewOptions(
			core.Option{Key: "repo", Value: ".."},
		))
	})
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, output, "usage: core-agent repo/sync")
}

// TestRepoSyncCov_CmdRepoSyncLocal_Ugly_SyncFails — a target whose local repo
// does not exist makes runRepoSync fail; the command prints the error and
// returns a failure.
func TestRepoSyncCov_CmdRepoSyncLocal_Ugly_SyncFails(t *testing.T) {
	s, _, _ := repoSyncTestPrep(t)

	var result core.Result
	output := captureStdout(t, func() {
		result = s.cmdRepoSyncLocal(core.NewOptions(
			core.Option{Key: "repo", Value: "ghost-repo"},
		))
	})
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, output, "error:")
}

// TestRepoSyncCov_CmdRepoSyncLocal_Good_NoReset — a fetch-only sync (no --reset,
// no --branch) prints the fetched line without a branch and without a reset
// line, and counts one repo.
func TestRepoSyncCov_CmdRepoSyncLocal_Good_NoReset(t *testing.T) {
	s, c, _ := repoSyncTestPrep(t)
	_, _ = repoSyncCreateTrackedRepo(t, c, s.codePath, "core", "test-repo")

	s.registerRepoSyncSupport()
	var result core.Result
	output := captureStdout(t, func() {
		result = s.cmdRepoSyncLocal(core.NewOptions(
			core.Option{Key: "repo", Value: "test-repo"},
		))
	})
	core.RequireTrue(t, result.OK)

	commandOutput, ok := result.Value.(RepoSyncCommandOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 1, commandOutput.Count)
	core.AssertLen(t, commandOutput.Synced, 1)
	core.AssertFalse(t, commandOutput.Synced[0].Reset)
	core.AssertContains(t, output, "fetched core/test-repo")
	core.AssertContains(t, output, "count: 1")
	core.AssertNotContains(t, output, "reset ")
}

// TestRepoSyncCov_HandleRepoSyncFetch_Good_WithBranch — the fetch action
// records the requested branch in its output when given one.
func TestRepoSyncCov_HandleRepoSyncFetch_Good_WithBranch(t *testing.T) {
	s, _, _ := repoSyncTestPrep(t)
	_, _ = repoSyncCreateTrackedRepo(t, s.Core(), s.codePath, "core", "test-repo")

	result := s.handleRepoSyncFetch(context.Background(), core.NewOptions(
		core.Option{Key: "repo", Value: "test-repo"},
		core.Option{Key: "branch", Value: "main"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(RepoSyncOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "main", output.Branch)
	core.AssertEqual(t, "test-repo", output.Repo)
}

// TestRepoSyncCov_HandleRepoSyncFetch_Bad_InvalidTarget — an invalid repo name
// fails target resolution before any git fetch.
func TestRepoSyncCov_HandleRepoSyncFetch_Bad_InvalidTarget(t *testing.T) {
	s, _, _ := repoSyncTestPrep(t)

	result := s.handleRepoSyncFetch(context.Background(), core.NewOptions(
		core.Option{Key: "repo", Value: ".."},
	))
	core.AssertFalse(t, result.OK)
}

// TestRepoSyncCov_HandleRepoSyncFetch_Ugly_RepoDirMissing — a valid name with no
// local checkout fails the repoSyncRepoDir guard.
func TestRepoSyncCov_HandleRepoSyncFetch_Ugly_RepoDirMissing(t *testing.T) {
	s, _, _ := repoSyncTestPrep(t)

	result := s.handleRepoSyncFetch(context.Background(), core.NewOptions(
		core.Option{Key: "repo", Value: "absent"},
	))
	core.AssertFalse(t, result.OK)

	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "local repo not found")
}

// TestRepoSyncCov_HandleRepoSyncReset_Bad_InvalidTarget — an invalid repo name
// fails target resolution before any git reset.
func TestRepoSyncCov_HandleRepoSyncReset_Bad_InvalidTarget(t *testing.T) {
	s, _, _ := repoSyncTestPrep(t)

	result := s.handleRepoSyncReset(context.Background(), core.NewOptions(
		core.Option{Key: "repo", Value: ".."},
	))
	core.AssertFalse(t, result.OK)
}

// TestRepoSyncCov_HandleRepoSyncReset_Good_SameBranchNoCheckout — resetting the
// already-checked-out branch skips the checkout step and hard-resets in place.
func TestRepoSyncCov_HandleRepoSyncReset_Good_SameBranchNoCheckout(t *testing.T) {
	s, c, _ := repoSyncTestPrep(t)
	remoteDir, repoDir := repoSyncCreateTrackedRepo(t, c, s.codePath, "core", "test-repo")
	_, remoteHead := repoSyncPushCommit(t, c, remoteDir, "main", "reset.go", "package reset\n")

	// Bring the fetch refs up to date so origin/main has the new commit.
	core.RequireTrue(t, c.Process().RunIn(context.Background(), repoDir, "git", "fetch", "origin", "main").OK)

	result := s.handleRepoSyncReset(context.Background(), core.NewOptions(
		core.Option{Key: "repo", Value: "test-repo"},
		core.Option{Key: "branch", Value: "main"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(RepoSyncOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Reset)
	core.AssertEqual(t, "main", output.Branch)
	core.AssertEqual(t, remoteHead, repoSyncGitOutput(t, c, repoDir, "rev-parse", "HEAD"))
}

// TestRepoSyncCov_RunRepoSync_Good_BranchOnlyFetchNoReset — passing a branch
// without --reset still resolves the branch and fetches it, but does not reset.
func TestRepoSyncCov_RunRepoSync_Good_BranchOnlyFetchNoReset(t *testing.T) {
	s, c, _ := repoSyncTestPrep(t)
	_, _ = repoSyncCreateTrackedRepo(t, c, s.codePath, "core", "test-repo")

	result := s.runRepoSync(context.Background(), fetchRepoRef{Org: "core", Repo: "test-repo"}, "main", false)
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(RepoSyncOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "main", output.Branch)
	core.AssertFalse(t, output.Reset)
}

// TestRepoSyncCov_RegisterRepoSyncSupport_Good_IdempotentSecondCall — a second
// registration call short-circuits on the "registered" config flag and stays OK.
func TestRepoSyncCov_RegisterRepoSyncSupport_Good_IdempotentSecondCall(t *testing.T) {
	s, _, _ := repoSyncTestPrep(t)

	core.RequireTrue(t, s.registerRepoSyncSupport().OK)
	core.AssertTrue(t, s.registerRepoSyncSupport().OK) // second call hits the early-return guard
}

// TestRepoSyncCov_HandleRepoSyncReset_Ugly_DifferentBranchCheckout — when the
// working copy is on a different branch, reset performs a `git checkout -B` to
// the target before the hard reset.
func TestRepoSyncCov_HandleRepoSyncReset_Ugly_DifferentBranchCheckout(t *testing.T) {
	s, c, _ := repoSyncTestPrep(t)
	remoteDir, repoDir := repoSyncCreateTrackedRepo(t, c, s.codePath, "core", "test-repo")
	_, remoteHead := repoSyncPushCommit(t, c, remoteDir, "main", "checkout.go", "package checkout\n")

	// Move the working copy onto a feature branch and refresh origin refs.
	core.RequireTrue(t, c.Process().RunIn(context.Background(), repoDir, "git", "checkout", "-b", "feature/wip").OK)
	core.RequireTrue(t, c.Process().RunIn(context.Background(), repoDir, "git", "fetch", "origin", "main").OK)

	result := s.handleRepoSyncReset(context.Background(), core.NewOptions(
		core.Option{Key: "repo", Value: "test-repo"},
		core.Option{Key: "branch", Value: "main"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(RepoSyncOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Reset)
	core.AssertEqual(t, "main", repoSyncGitOutput(t, c, repoDir, "rev-parse", "--abbrev-ref", "HEAD"))
	core.AssertEqual(t, remoteHead, repoSyncGitOutput(t, c, repoDir, "rev-parse", "HEAD"))
}

// TestRepoSyncCov_OnWorkspacePushed_Ugly_InvalidRepoName — a WorkspacePushed
// carrying an invalid repo name fails at target resolution (before any sync).
func TestRepoSyncCov_OnWorkspacePushed_Ugly_InvalidRepoName(t *testing.T) {
	s, _, _ := repoSyncTestPrep(t)

	result := s.onWorkspacePushed(context.Background(), messages.WorkspacePushed{
		Repo:   "..",
		Branch: "main",
		Org:    "core",
	})
	core.AssertFalse(t, result.OK)

	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "invalid repo name")
}

// otherIPCMessage is a non-WorkspacePushed message used to drive the IPC
// handler's type-assert miss arm.
type otherIPCMessage struct{}

func (otherIPCMessage) MessageType() string { return "agentic.test.other" }
