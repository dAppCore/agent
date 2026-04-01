// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	"dappco.re/go/agent/pkg/messages"
	core "dappco.re/go/core"
)

// RegisterHandlers(c, subsystem)
// c.ACTION(messages.AgentCompleted{Workspace: "core/go-io/task-5", Repo: "go-io", Status: "completed"})
func RegisterHandlers(c *core.Core, s *PrepSubsystem) {
	if c == nil || s == nil {
		return
	}

	c.RegisterActions(
		func(coreApp *core.Core, msg core.Message) core.Result {
			return handleCompletionQA(coreApp, msg)
		},
		func(coreApp *core.Core, msg core.Message) core.Result {
			return handleCompletionAutoPR(coreApp, msg)
		},
		func(coreApp *core.Core, msg core.Message) core.Result {
			return handleCompletionVerify(coreApp, msg)
		},
		func(coreApp *core.Core, msg core.Message) core.Result {
			return handleCompletionCommit(coreApp, msg)
		},
		func(coreApp *core.Core, msg core.Message) core.Result {
			return handleCompletionIngest(coreApp, msg)
		},
		func(coreApp *core.Core, msg core.Message) core.Result {
			return handleCompletionPoke(coreApp, msg)
		},
	)
}

// _ = prep.HandleIPCEvents(c, messages.AgentCompleted{Workspace: "core/go-io/task-5", Status: "completed"})
// _ = prep.HandleIPCEvents(c, messages.SpawnQueued{Workspace: "core/go-io/task-5", Agent: "codex", Task: "fix tests"})
func (s *PrepSubsystem) HandleIPCEvents(c *core.Core, msg core.Message) core.Result {
	switch ev := msg.(type) {
	case messages.SpawnQueued:
		workspaceDir := resolveWorkspace(ev.Workspace)
		if workspaceDir == "" {
			break
		}
		prompt := core.Concat("TASK: ", ev.Task, "\n\nResume from where you left off. Read CODEX.md for conventions. Commit when done.")
		pid, processID, outputFile, err := s.spawnAgent(ev.Agent, prompt, workspaceDir)
		if err != nil {
			break
		}
		if result := ReadStatusResult(workspaceDir); result.OK {
			workspaceStatus, ok := workspaceStatusValue(result)
			if !ok {
				break
			}
			workspaceStatus.PID = pid
			workspaceStatus.ProcessID = processID
			writeStatusResult(workspaceDir, workspaceStatus)
			if runnerResult := c.Service("runner"); runnerResult.OK {
				if runnerSvc, ok := runnerResult.Value.(workspaceTracker); ok {
					runnerSvc.TrackWorkspace(WorkspaceName(workspaceDir), workspaceStatus)
				}
			}
		}
		_ = outputFile
	}

	return core.Result{OK: true}
}

func handleCompletionQA(c *core.Core, msg core.Message) core.Result {
	ev, ok := msg.(messages.AgentCompleted)
	if !ok || ev.Status != "completed" {
		return core.Result{OK: true}
	}

	workspaceDir := resolveWorkspace(ev.Workspace)
	if workspaceDir == "" {
		return core.Result{OK: true}
	}

	performAsyncIfRegistered(c, "agentic.qa", workspaceActionOptions(workspaceDir))
	return core.Result{OK: true}
}

func handleCompletionAutoPR(c *core.Core, msg core.Message) core.Result {
	ev, ok := msg.(messages.QAResult)
	if !ok || !ev.Passed {
		return core.Result{OK: true}
	}

	workspaceDir := resolveWorkspace(ev.Workspace)
	if workspaceDir == "" {
		return core.Result{OK: true}
	}

	performAsyncIfRegistered(c, "agentic.auto-pr", workspaceActionOptions(workspaceDir))
	return core.Result{OK: true}
}

func handleCompletionVerify(c *core.Core, msg core.Message) core.Result {
	ev, ok := msg.(messages.PRCreated)
	if !ok {
		return core.Result{OK: true}
	}

	workspaceDir := findWorkspaceByPR(ev.Repo, ev.Branch)
	if workspaceDir == "" {
		return core.Result{OK: true}
	}

	performAsyncIfRegistered(c, "agentic.verify", workspaceActionOptions(workspaceDir))
	return core.Result{OK: true}
}

func handleCompletionCommit(c *core.Core, msg core.Message) core.Result {
	switch ev := msg.(type) {
	case messages.PRMerged:
		workspaceDir := findWorkspaceByPR(ev.Repo, "")
		if workspaceDir != "" {
			if c.Action("agentic.commit").Exists() {
				c.Action("agentic.commit").Run(context.Background(), workspaceActionOptions(workspaceDir))
			}
		}
	case messages.PRNeedsReview:
		workspaceDir := findWorkspaceByPR(ev.Repo, "")
		if workspaceDir != "" {
			if c.Action("agentic.commit").Exists() {
				c.Action("agentic.commit").Run(context.Background(), workspaceActionOptions(workspaceDir))
			}
		}
	}

	return core.Result{OK: true}
}

func handleCompletionIngest(c *core.Core, msg core.Message) core.Result {
	ev, ok := msg.(messages.AgentCompleted)
	if !ok || c == nil || !c.Config().Enabled("auto-ingest") {
		return core.Result{OK: true}
	}

	workspaceDir := resolveWorkspace(ev.Workspace)
	if workspaceDir == "" {
		return core.Result{OK: true}
	}

	performAsyncIfRegistered(c, "agentic.ingest", workspaceActionOptions(workspaceDir))
	return core.Result{OK: true}
}

func handleCompletionPoke(c *core.Core, msg core.Message) core.Result {
	if _, ok := msg.(messages.AgentCompleted); !ok {
		return core.Result{OK: true}
	}

	if performAsyncIfRegistered(c, "runner.poke", core.NewOptions()).OK {
		return core.Result{OK: true}
	}
	performAsyncIfRegistered(c, "agentic.poke", core.NewOptions())
	return core.Result{OK: true}
}

func workspaceActionOptions(workspaceDir string) core.Options {
	return core.NewOptions(core.Option{Key: "workspace", Value: workspaceDir})
}

func performAsyncIfRegistered(c *core.Core, action string, options core.Options) core.Result {
	if c == nil || !c.Action(action).Exists() {
		return core.Result{}
	}
	return c.PerformAsync(action, options)
}

// spawnResult := prep.SpawnFromQueue("codex", prompt, workspaceDir)
// pid := spawnResult.Value.(int)
func (s *PrepSubsystem) SpawnFromQueue(agent, prompt, workspaceDir string) core.Result {
	pid, _, _, err := s.spawnAgent(agent, prompt, workspaceDir)
	if err != nil {
		return core.Result{
			Value: core.E("agentic.SpawnFromQueue", "failed to spawn queued agent", err),
		}
	}
	return core.Result{Value: pid, OK: true}
}

// workspaceDir := resolveWorkspace("core/go-io/task-5")
// core.Println(workspaceDir) // "/srv/.core/workspace/core/go-io/task-5"
func resolveWorkspace(name string) string {
	workspaceRoot := WorkspaceRoot()
	path := core.JoinPath(workspaceRoot, name)
	if fs.IsDir(path) {
		return path
	}
	return ""
}

func findWorkspaceByPR(repo, branch string) string {
	for _, path := range WorkspaceStatusPaths() {
		workspaceDir := core.PathDir(path)
		statusResult := ReadStatusResult(workspaceDir)
		workspaceStatus, ok := workspaceStatusValue(statusResult)
		if !ok {
			continue
		}
		if workspaceStatus.Repo != repo {
			continue
		}
		if branch != "" && workspaceStatus.Branch != branch {
			continue
		}
		if branch == "" || workspaceStatus.Branch == branch {
			return workspaceDir
		}
	}
	return ""
}
