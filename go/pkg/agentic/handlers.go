// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/messages"
)

// c := core.New(core.WithService(agentic.ProcessRegister))
// agentic.RegisterHandlers(c, agentic.NewPrep())
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
		func(coreApp *core.Core, msg core.Message) core.Result {
			return handleHarvestAutoPR(coreApp, msg)
		},
	)
}

// completed := prep.HandleIPCEvents(c, messages.AgentCompleted{Workspace: "core/go-io/task-5", Status: "completed"})
// queued := prep.HandleIPCEvents(c, messages.SpawnQueued{Workspace: "core/go-io/task-5", Agent: "codex", Task: "fix tests"})
func (s *PrepSubsystem) HandleIPCEvents(c *core.Core, msg core.Message) core.Result {
	switch ev := msg.(type) {
	case messages.SpawnQueued:
		workspaceDir := resolveWorkspace(ev.Workspace)
		if workspaceDir == "" {
			break
		}
		prompt := core.Concat("TASK: ", ev.Task, "\n\nResume from where you left off. Read CODEX.md for conventions. Commit when done.")
		pid, processID, _, err := spawnAgent(s, ev.Agent, prompt, workspaceDir)
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
		workspaceDir := findWorkspaceByPRWithInfo(ev.Repo, "", ev.PRNum, ev.PRURL)
		if workspaceDir != "" {
			if c.Action("agentic.commit").Exists() {
				if result := c.Action("agentic.commit").Run(context.Background(), workspaceActionOptions(workspaceDir)); !result.OK {
					return result
				}
			}
		}
	case messages.PRNeedsReview:
		workspaceDir := findWorkspaceByPRWithInfo(ev.Repo, "", ev.PRNum, ev.PRURL)
		if workspaceDir != "" {
			if c.Action("agentic.commit").Exists() {
				if result := c.Action("agentic.commit").Run(context.Background(), workspaceActionOptions(workspaceDir)); !result.OK {
					return result
				}
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

	if c != nil && c.Action("runner.poke").Exists() {
		if result := c.ACTION(messages.PokeQueue{}); !result.OK {
			return result
		}
		return core.Result{OK: true}
	}
	performAsyncIfRegistered(c, "agentic.poke", core.NewOptions())
	return core.Result{OK: true}
}

// handleHarvestAutoPR re-dispatches a completed harvest into the closeout
// pipeline: the harvested branch's workspace runs agentic.auto-pr — the same
// entry the QA→PR flow uses — so a harvest joins the normal PR path instead of
// stopping at the harvest step. The runner notifies on harvest.status from the
// same broadcast (H3). Unknown messages pass through OK.
func handleHarvestAutoPR(c *core.Core, msg core.Message) core.Result {
	ev, ok := msg.(messages.HarvestComplete)
	if !ok {
		return core.Result{OK: true}
	}
	workspaceDir := findWorkspaceByPR(ev.Repo, ev.Branch)
	if workspaceDir == "" {
		return core.Result{OK: true}
	}
	performAsyncIfRegistered(c, "agentic.auto-pr", workspaceActionOptions(workspaceDir))
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
	pid, _, _, err := spawnAgent(s, agent, prompt, workspaceDir)
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
	if fs.IsDir(path).OK {
		return path
	}
	return ""
}

func findWorkspaceByPR(repo, branch string) string {
	return findWorkspaceByPRWithInfo(repo, branch, 0, "")
}

func findWorkspaceByPRWithInfo(repo, branch string, prNum int, prURL string) string {
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
		if prNum > 0 {
			if workspaceStatus.PRURL != "" && extractPullRequestNumber(workspaceStatus.PRURL) == prNum {
				return workspaceDir
			}
			if prURL != "" && workspaceStatus.PRURL == prURL {
				return workspaceDir
			}
			continue
		}
		if branch == "" || workspaceStatus.Branch == branch {
			return workspaceDir
		}
	}
	return ""
}
