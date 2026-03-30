// SPDX-License-Identifier: EUPL-1.2

// Workspace CLI commands registered by the agentic service during OnStartup.

package agentic

import (
	"context"

	core "dappco.re/go/core"
)

// registerWorkspaceCommands adds workspace management commands.
func (s *PrepSubsystem) registerWorkspaceCommands() {
	c := s.Core()
	c.Command("workspace/list", core.Command{Description: "List all agent workspaces with status", Action: s.cmdWorkspaceList})
	c.Command("workspace/clean", core.Command{Description: "Remove completed/failed/blocked workspaces", Action: s.cmdWorkspaceClean})
	c.Command("workspace/dispatch", core.Command{Description: "Dispatch an agent to work on a repo task", Action: s.cmdWorkspaceDispatch})
}

func (s *PrepSubsystem) cmdWorkspaceList(_ core.Options) core.Result {
	statusFiles := WorkspaceStatusPaths()
	count := 0
	for _, sf := range statusFiles {
		wsDir := core.PathDir(sf)
		wsName := WorkspaceName(wsDir)
		result := ReadStatusResult(wsDir)
		workspaceStatus, ok := workspaceStatusValue(result)
		if !ok {
			continue
		}
		core.Print(nil, "  %-8s %-8s %-10s %s", workspaceStatus.Status, workspaceStatus.Agent, workspaceStatus.Repo, wsName)
		count++
	}
	if count == 0 {
		core.Print(nil, "  no workspaces")
	}
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdWorkspaceClean(options core.Options) core.Result {
	wsRoot := WorkspaceRoot()
	fsys := s.Core().Fs()
	filter := options.String("_arg")
	if filter == "" {
		filter = "all"
	}

	statusFiles := WorkspaceStatusPaths()
	var toRemove []string

	for _, sf := range statusFiles {
		wsDir := core.PathDir(sf)
		wsName := WorkspaceName(wsDir)
		result := ReadStatusResult(wsDir)
		workspaceStatus, ok := workspaceStatusValue(result)
		if !ok {
			continue
		}
		status := workspaceStatus.Status

		switch filter {
		case "all":
			if status == "completed" || status == "failed" || status == "blocked" || status == "merged" || status == "ready-for-review" {
				toRemove = append(toRemove, wsName)
			}
		case "completed":
			if status == "completed" || status == "merged" || status == "ready-for-review" {
				toRemove = append(toRemove, wsName)
			}
		case "failed":
			if status == "failed" {
				toRemove = append(toRemove, wsName)
			}
		case "blocked":
			if status == "blocked" {
				toRemove = append(toRemove, wsName)
			}
		}
	}

	if len(toRemove) == 0 {
		core.Print(nil, "nothing to clean")
		return core.Result{OK: true}
	}

	for _, name := range toRemove {
		path := core.JoinPath(wsRoot, name)
		fsys.DeleteAll(path)
		core.Print(nil, "  removed %s", name)
	}
	core.Print(nil, "\n  %d workspaces removed", len(toRemove))
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdWorkspaceDispatch(options core.Options) core.Result {
	repo := options.String("_arg")
	if repo == "" {
		core.Print(nil, "usage: core-agent workspace dispatch <repo> --task=\"...\" --issue=N|--pr=N|--branch=X [--agent=codex]")
		return core.Result{Value: core.E("agentic.cmdWorkspaceDispatch", "repo is required", nil), OK: false}
	}

	// Call dispatch directly — CLI is an explicit user action,
	// not gated by the frozen-queue entitlement.
	input := DispatchInput{
		Repo:     repo,
		Task:     options.String("task"),
		Agent:    options.String("agent"),
		Org:      options.String("org"),
		Template: options.String("template"),
		Branch:   options.String("branch"),
		Issue:    parseIntStr(options.String("issue")),
		PR:       parseIntStr(options.String("pr")),
	}
	_, out, err := s.dispatch(context.Background(), nil, input)
	if err != nil {
		core.Print(nil, "dispatch failed: %s", err.Error())
		return core.Result{Value: err, OK: false}
	}
	agent := out.Agent
	if agent == "" {
		agent = "codex"
	}
	core.Print(nil, "dispatched %s to %s", agent, repo)
	if out.WorkspaceDir != "" {
		core.Print(nil, "  workspace: %s", out.WorkspaceDir)
	}
	if out.PID > 0 {
		core.Print(nil, "  pid:       %d", out.PID)
	}
	return core.Result{OK: true}
}
