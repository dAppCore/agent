// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"dappco.re/go/agent/pkg/messages"
	core "dappco.re/go/core"
)

// HandleIPCEvents applies agent lifecycle messages to the prep subsystem.
//
//	_ = prep.HandleIPCEvents(c, messages.AgentCompleted{Workspace: "core/go-io/task-5", Status: "completed"})
//	_ = prep.HandleIPCEvents(c, messages.SpawnQueued{Workspace: "core/go-io/task-5", Agent: "codex", Task: "fix tests"})
func (s *PrepSubsystem) HandleIPCEvents(c *core.Core, msg core.Message) core.Result {
	switch ev := msg.(type) {
	case messages.AgentCompleted:
		// Ingest findings (feature-flag gated)
		if c.Config().Enabled("auto-ingest") {
			if workspaceDir := resolveWorkspace(ev.Workspace); workspaceDir != "" {
				s.ingestFindings(workspaceDir)
			}
		}

	case messages.SpawnQueued:
		// Runner asks agentic to spawn a queued workspace
		workspaceDir := resolveWorkspace(ev.Workspace)
		if workspaceDir == "" {
			break
		}
		prompt := core.Concat("TASK: ", ev.Task, "\n\nResume from where you left off. Read CODEX.md for conventions. Commit when done.")
		pid, processID, outputFile, err := s.spawnAgent(ev.Agent, prompt, workspaceDir)
		if err != nil {
			break
		}
		// Update status with real PID
		if result := ReadStatusResult(workspaceDir); result.OK {
			workspaceStatus, ok := workspaceStatusValue(result)
			if !ok {
				break
			}
			workspaceStatus.PID = pid
			workspaceStatus.ProcessID = processID
			writeStatusResult(workspaceDir, workspaceStatus)
			if runnerSvc, ok := core.ServiceFor[workspaceTracker](c, "runner"); ok {
				runnerSvc.TrackWorkspace(WorkspaceName(workspaceDir), workspaceStatus)
			}
		}
		_ = outputFile
	}

	return core.Result{OK: true}
}

// SpawnFromQueue spawns an agent in a pre-prepped workspace.
// Called by runner.Service via ServiceFor interface matching.
//
//	spawnResult := prep.SpawnFromQueue("codex", prompt, workspaceDir)
//	pid := spawnResult.Value.(int)
func (s *PrepSubsystem) SpawnFromQueue(agent, prompt, workspaceDir string) core.Result {
	pid, _, _, err := s.spawnAgent(agent, prompt, workspaceDir)
	if err != nil {
		return core.Result{
			Value: core.E("agentic.SpawnFromQueue", "failed to spawn queued agent", err),
		}
	}
	return core.Result{Value: pid, OK: true}
}

// resolveWorkspace converts a workspace name back to the full path.
//
//	resolveWorkspace("core/go-io/task-5") → "/Users/snider/Code/.core/workspace/core/go-io/task-5"
func resolveWorkspace(name string) string {
	workspaceRoot := WorkspaceRoot()
	path := core.JoinPath(workspaceRoot, name)
	if fs.IsDir(path) {
		return path
	}
	return ""
}

// findWorkspaceByPR finds a workspace directory by repo name and branch.
// Scans running/completed workspaces for a matching repo+branch combination.
func findWorkspaceByPR(repo, branch string) string {
	for _, path := range WorkspaceStatusPaths() {
		workspaceDir := core.PathDir(path)
		statusResult := ReadStatusResult(workspaceDir)
		workspaceStatus, ok := workspaceStatusValue(statusResult)
		if !ok {
			continue
		}
		if workspaceStatus.Repo == repo && workspaceStatus.Branch == branch {
			return workspaceDir
		}
	}
	return ""
}
