// SPDX-License-Identifier: EUPL-1.2

// Workspace CLI commands registered by the agentic service during OnStartup.

package agentic

import (
	core "dappco.re/go/core"
)

// registerWorkspaceCommands adds workspace management commands.
func (s *PrepSubsystem) registerWorkspaceCommands() {
	c := s.Core()
	c.Command("workspace/list", core.Command{Description: "List all agent workspaces with status", Action: s.cmdWorkspaceList})
	c.Command("workspace/clean", core.Command{Description: "Remove completed/failed/blocked workspaces", Action: s.cmdWorkspaceClean})
	c.Command("workspace/dispatch", core.Command{Description: "Dispatch an agent to work on a repo task", Action: s.cmdWorkspaceDispatch})
}

func (s *PrepSubsystem) cmdWorkspaceList(opts core.Options) core.Result {
	wsRoot := WorkspaceRoot()
	fsys := s.Core().Fs()

	statusFiles := core.PathGlob(core.JoinPath(wsRoot, "*", "status.json"))
	count := 0
	for _, sf := range statusFiles {
		wsName := core.PathBase(core.PathDir(sf))
		if sr := fsys.Read(sf); sr.OK {
			content := sr.Value.(string)
			status := extractField(content, "status")
			repo := extractField(content, "repo")
			agent := extractField(content, "agent")
			core.Print(nil, "  %-8s %-8s %-10s %s", status, agent, repo, wsName)
			count++
		}
	}
	if count == 0 {
		core.Print(nil, "  no workspaces")
	}
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdWorkspaceClean(opts core.Options) core.Result {
	wsRoot := WorkspaceRoot()
	fsys := s.Core().Fs()
	filter := opts.String("_arg")
	if filter == "" {
		filter = "all"
	}

	statusFiles := core.PathGlob(core.JoinPath(wsRoot, "*", "status.json"))
	var toRemove []string

	for _, sf := range statusFiles {
		wsName := core.PathBase(core.PathDir(sf))
		sr := fsys.Read(sf)
		if !sr.OK {
			continue
		}
		status := extractField(sr.Value.(string), "status")

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

func (s *PrepSubsystem) cmdWorkspaceDispatch(opts core.Options) core.Result {
	repo := opts.String("_arg")
	if repo == "" {
		core.Print(nil, "usage: core-agent workspace dispatch <repo> --task=\"...\" --issue=N|--pr=N|--branch=X [--agent=codex]")
		return core.Result{OK: false}
	}
	core.Print(nil, "dispatch via CLI not yet wired — use MCP agentic_dispatch tool")
	core.Print(nil, "repo: %s, task: %s", repo, opts.String("task"))
	return core.Result{OK: true}
}

// extractField does a quick JSON field extraction without full unmarshal.
func extractField(jsonStr, field string) string {
	needle := core.Concat("\"", field, "\"")
	idx := -1
	for i := 0; i <= len(jsonStr)-len(needle); i++ {
		if jsonStr[i:i+len(needle)] == needle {
			idx = i + len(needle)
			break
		}
	}
	if idx < 0 {
		return ""
	}
	for idx < len(jsonStr) && (jsonStr[idx] == ':' || jsonStr[idx] == ' ' || jsonStr[idx] == '\t') {
		idx++
	}
	if idx >= len(jsonStr) || jsonStr[idx] != '"' {
		return ""
	}
	idx++
	end := idx
	for end < len(jsonStr) && jsonStr[end] != '"' {
		end++
	}
	return jsonStr[idx:end]
}
