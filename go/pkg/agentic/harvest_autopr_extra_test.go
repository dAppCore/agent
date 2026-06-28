// SPDX-License-Identifier: EUPL-1.2

// handleHarvestAutoPR (H3 in PLAN-cli-square-up.md): a completed harvest
// re-dispatches the harvested branch's workspace into agentic.auto-pr — the same
// closeout entry the QA→PR flow uses — so a harvest joins the normal PR path.
// Mirrors the completion-pipeline harness (setTestWorkspace + writeStatus +
// RegisterHandlers + requireEventually).

package agentic

import (
	"context"
	"sync"
	"testing"
	"time"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/messages"
)

// TestHandlers_HarvestComplete_RedispatchesAutoPR — HarvestComplete for a
// repo+branch with a matching workspace re-dispatches agentic.auto-pr with that
// workspace dir.
func TestHandlers_HarvestComplete_RedispatchesAutoPR(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	workspaceDir := core.JoinPath(root, "workspace", "core", "go-io", "task-7")
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(workspaceDir, "repo")).OK)
	core.RequireNoError(t, writeStatus(workspaceDir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Branch: "agent/fix-tests",
		Agent:  "codex",
	}))

	var mu sync.Mutex
	called := false
	var gotWorkspace string

	c := core.New()
	RegisterHandlers(c, &PrepSubsystem{})
	c.Action("agentic.auto-pr", func(_ context.Context, options core.Options) core.Result {
		mu.Lock()
		called = true
		gotWorkspace = options.String("workspace")
		mu.Unlock()
		return core.Result{OK: true}
	})

	c.ACTION(messages.HarvestComplete{Repo: "go-io", Branch: "agent/fix-tests", Files: 3})

	requireEventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return called
	}, time.Second, 10*time.Millisecond)

	mu.Lock()
	core.AssertEqual(t, workspaceDir, gotWorkspace)
	mu.Unlock()
}

// TestHandlers_HarvestComplete_NoWorkspace_NoDispatch — HarvestComplete with no
// matching workspace is a clean no-op: the handler returns before any
// re-dispatch (broadcast is synchronous, so auto-pr is provably never called).
func TestHandlers_HarvestComplete_NoWorkspace_NoDispatch(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	called := false
	c := core.New()
	RegisterHandlers(c, &PrepSubsystem{})
	c.Action("agentic.auto-pr", func(_ context.Context, _ core.Options) core.Result {
		called = true
		return core.Result{OK: true}
	})

	core.AssertNotPanics(t, func() {
		c.ACTION(messages.HarvestComplete{Repo: "ghost", Branch: "none", Files: 0})
	})
	core.AssertFalse(t, called)
}
