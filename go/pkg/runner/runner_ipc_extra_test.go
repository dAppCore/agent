// SPDX-License-Identifier: EUPL-1.2

package runner

import (
	"testing"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/messages"
)

// TestRunner_HandleIPCEvents_AgentStarted — the AgentStarted branch builds
// a "started" notification and calls the notifier. With no mcp service
// registered sendNotification no-ops, so this exercises the branch (running
// count + limit lookup + notification build) without any side effect.
func TestRunner_HandleIPCEvents_AgentStarted(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))
	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})

	r := svc.HandleIPCEvents(c, messages.AgentStarted{
		Agent: "codex", Repo: "go-io", Workspace: "core/go-io/task-1",
	})
	core.AssertTrue(t, r.OK)
}

// TestRunner_HandleIPCEvents_AgentCompleted_NoWorkspace — AgentCompleted
// with an EMPTY Workspace takes the repo-sweep else-branch: every tracked
// workspace for that repo whose status is "running" flips to the event
// status. Covers the s.workspaces.Each path the workspace-set test skips.
func TestRunner_HandleIPCEvents_AgentCompleted_NoWorkspace(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))
	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})

	svc.TrackWorkspace("core/go-io/task-a", &WorkspaceStatus{
		Status: "running", Agent: "codex", Repo: "go-io", PID: 1,
	})
	svc.TrackWorkspace("core/other/task-b", &WorkspaceStatus{
		Status: "running", Agent: "codex", Repo: "other", PID: 2,
	})

	// No Workspace → sweep by Repo. Only the go-io running workspace flips.
	r := svc.HandleIPCEvents(c, messages.AgentCompleted{
		Agent: "codex", Repo: "go-io", Status: "failed",
	})
	core.AssertTrue(t, r.OK)

	a := svc.workspaces.Get("core/go-io/task-a").Value.(*WorkspaceStatus)
	b := svc.workspaces.Get("core/other/task-b").Value.(*WorkspaceStatus)
	core.AssertEqual(t, "failed", a.Status)
	core.AssertEqual(t, 0, a.PID)
	// Different repo — untouched.
	core.AssertEqual(t, "running", b.Status)
	core.AssertEqual(t, 2, b.PID)
}
