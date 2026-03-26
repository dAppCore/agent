// SPDX-License-Identifier: EUPL-1.2

// IPC handler for agent lifecycle events.
// Auto-discovered by Core's WithService via the HandleIPCEvents interface.
// No manual RegisterHandlers call needed — Core wires it during service registration.

package agentic

import (
	"dappco.re/go/agent/pkg/messages"
	core "dappco.re/go/core"
)

// HandleIPCEvents implements Core's IPC handler interface.
// Auto-registered by WithService — no manual wiring needed.
//
// Handles:
//
//	AgentCompleted → ingest findings (runner handles channel push + queue poke)
//	SpawnQueued → runner asks agentic to spawn a queued workspace
func (s *PrepSubsystem) HandleIPCEvents(c *core.Core, msg core.Message) core.Result {
	switch ev := msg.(type) {
	case messages.AgentCompleted:
		// Ingest findings (feature-flag gated)
		if c.Config().Enabled("auto-ingest") {
			if wsDir := resolveWorkspace(ev.Workspace); wsDir != "" {
				s.ingestFindings(wsDir)
			}
		}
	}

	return core.Result{OK: true}
}

// resolveWorkspace converts a workspace name back to the full path.
//
//	resolveWorkspace("core/go-io/task-5") → "/Users/snider/Code/.core/workspace/core/go-io/task-5"
func resolveWorkspace(name string) string {
	wsRoot := WorkspaceRoot()
	path := core.JoinPath(wsRoot, name)
	if fs.IsDir(path) {
		return path
	}
	return ""
}

// findWorkspaceByPR finds a workspace directory by repo name and branch.
// Scans running/completed workspaces for a matching repo+branch combination.
func findWorkspaceByPR(repo, branch string) string {
	wsRoot := WorkspaceRoot()
	old := core.PathGlob(core.JoinPath(wsRoot, "*", "status.json"))
	deep := core.PathGlob(core.JoinPath(wsRoot, "*", "*", "*", "status.json"))
	for _, path := range append(old, deep...) {
		wsDir := core.PathDir(path)
		st, err := ReadStatus(wsDir)
		if err != nil {
			continue
		}
		if st.Repo == repo && st.Branch == branch {
			return wsDir
		}
	}
	return ""
}
