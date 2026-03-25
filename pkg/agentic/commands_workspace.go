// SPDX-License-Identifier: EUPL-1.2

// Workspace CLI commands registered by the agentic service during OnStartup.

package agentic

import (
	"os"

	core "dappco.re/go/core"
)

// registerWorkspaceCommands adds workspace management commands.
func (s *PrepSubsystem) registerWorkspaceCommands() {
	c := s.core
	c.Command("workspace/list", core.Command{Description: "List all agent workspaces with status", Action: s.cmdWorkspaceList})
	c.Command("workspace/clean", core.Command{Description: "Remove completed/failed/blocked workspaces", Action: s.cmdWorkspaceClean})
	c.Command("workspace/dispatch", core.Command{Description: "Dispatch an agent to work on a repo task", Action: s.cmdWorkspaceDispatch})
}

func (s *PrepSubsystem) cmdWorkspaceList(opts core.Options) core.Result {
	wsRoot := WorkspaceRoot()
	fsys := s.core.Fs()

	r := fsys.List(wsRoot)
	if !r.OK {
		core.Print(nil, "no workspaces at %s", wsRoot)
		return core.Result{OK: true}
	}

	entries := r.Value.([]os.DirEntry)
	count := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		statusFile := core.JoinPath(wsRoot, e.Name(), "status.json")
		if sr := fsys.Read(statusFile); sr.OK {
			content := sr.Value.(string)
			status := extractField(content, "status")
			repo := extractField(content, "repo")
			agent := extractField(content, "agent")
			core.Print(nil, "  %-8s %-8s %-10s %s", status, agent, repo, e.Name())
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
	fsys := s.core.Fs()
	filter := opts.String("_arg")
	if filter == "" {
		filter = "all"
	}

	r := fsys.List(wsRoot)
	if !r.OK {
		core.Print(nil, "no workspaces")
		return core.Result{OK: true}
	}

	entries := r.Value.([]os.DirEntry)
	var toRemove []string

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		statusFile := core.JoinPath(wsRoot, e.Name(), "status.json")
		sr := fsys.Read(statusFile)
		if !sr.OK {
			continue
		}
		status := extractField(sr.Value.(string), "status")

		switch filter {
		case "all":
			if status == "completed" || status == "failed" || status == "blocked" || status == "merged" || status == "ready-for-review" {
				toRemove = append(toRemove, e.Name())
			}
		case "completed":
			if status == "completed" || status == "merged" || status == "ready-for-review" {
				toRemove = append(toRemove, e.Name())
			}
		case "failed":
			if status == "failed" {
				toRemove = append(toRemove, e.Name())
			}
		case "blocked":
			if status == "blocked" {
				toRemove = append(toRemove, e.Name())
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
