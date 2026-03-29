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

	case messages.SpawnQueued:
		// Runner asks agentic to spawn a queued workspace
		// Runner asks agentic to spawn a queued workspace
		wsDir := resolveWorkspace(ev.Workspace)
		if wsDir == "" {
				break
		}
		prompt := core.Concat("TASK: ", ev.Task, "\n\nResume from where you left off. Read CODEX.md for conventions. Commit when done.")
		pid, outputFile, err := s.spawnAgent(ev.Agent, prompt, wsDir)
		if err != nil {
				break
		}
		// Update status with real PID
		if st, serr := ReadStatus(wsDir); serr == nil {
			st.PID = pid
			writeStatus(wsDir, st)
			if runnerSvc, ok := core.ServiceFor[workspaceTracker](c, "runner"); ok {
				runnerSvc.TrackWorkspace(WorkspaceName(wsDir), st)
			}
		}
		_ = outputFile
	}

	return core.Result{OK: true}
}

// SpawnFromQueue spawns an agent in a pre-prepped workspace.
// Called by runner.Service via ServiceFor interface matching.
//
//	pid, err := prep.SpawnFromQueue("codex", prompt, wsDir)
func (s *PrepSubsystem) SpawnFromQueue(agent, prompt, wsDir string) (int, error) {
	pid, _, err := s.spawnAgent(agent, prompt, wsDir)
	return pid, err
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
