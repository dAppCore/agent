// SPDX-License-Identifier: EUPL-1.2

// HandleIPCEvents concurrency-config + workspace-targeted coverage. The
// existing IPC tests fire AgentStarted / AgentCompleted with NO
// agents.concurrency config set, so the inner "config present → resolve
// limit" branch (the map type-assert + per-agent lookup) and the
// non-empty-Workspace AgentCompleted branch were uncovered. These set the
// config + track a matching workspace so those legs run, and exercise the
// PokeQueue case.

package runner

import (
	"testing"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/messages"
)

// TestRunner_HandleIPCEvents_AgentStarted_ConcurrencyConfig — AgentStarted
// with agents.concurrency set resolves the per-agent limit out of config
// (the map type-assert + lookup branch). A tracked running codex workspace
// makes the running count non-zero so the notification carries real
// numbers.
func TestRunner_HandleIPCEvents_AgentStarted_ConcurrencyConfig(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))
	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})
	c.Config().Set("agents.concurrency", map[string]ConcurrencyLimit{
		"codex": {Total: 3},
	})
	svc.TrackWorkspace("core/go-io/task-r", &WorkspaceStatus{
		Status: "running", Agent: "codex", Repo: "go-io", PID: -1,
	})

	r := svc.HandleIPCEvents(c, messages.AgentStarted{
		Agent: "codex", Repo: "go-io", Workspace: "core/go-io/task-2",
	})
	core.AssertTrue(t, r.OK)
}

// TestRunner_HandleIPCEvents_AgentCompleted_Workspace_ConcurrencyConfig —
// AgentCompleted with a NON-empty Workspace flips that specific tracked
// workspace (the if ev.Workspace != "" branch) AND resolves the limit from
// config (the completion-side map lookup). Covers both legs the
// no-workspace test skips.
func TestRunner_HandleIPCEvents_AgentCompleted_Workspace_ConcurrencyConfig(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))
	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})
	c.Config().Set("agents.concurrency", map[string]ConcurrencyLimit{
		"codex": {Total: 2},
	})
	svc.TrackWorkspace("core/go-io/task-x", &WorkspaceStatus{
		Status: "running", Agent: "codex", Repo: "go-io", PID: 99,
	})

	r := svc.HandleIPCEvents(c, messages.AgentCompleted{
		Agent: "codex", Repo: "go-io", Workspace: "core/go-io/task-x", Status: "completed",
	})
	core.AssertTrue(t, r.OK)

	// The targeted workspace flipped to the event status with PID cleared.
	st := svc.workspaces.Get("core/go-io/task-x").Value.(*WorkspaceStatus)
	core.AssertEqual(t, "completed", st.Status)
	core.AssertEqual(t, 0, st.PID)
}

// TestRunner_HandleIPCEvents_PokeQueue — the PokeQueue case routes to
// drainQueueAndNotify. With no workspace root nothing is dispatched, but
// the case body runs (the message-switch arm the other IPC tests skip).
func TestRunner_HandleIPCEvents_PokeQueue(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))
	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	r := svc.HandleIPCEvents(c, messages.PokeQueue{})
	core.AssertTrue(t, r.OK)
}
