// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go"
)

func (s *PrepSubsystem) registerWorkspaceCommands() core.Result {
	c := s.Core()
	if r := c.Command("workspace/list", core.Command{Description: "List all agent workspaces with status", Action: s.cmdWorkspaceList}); !r.OK {
		return r
	}
	if r := c.Command("agentic:workspace/list", core.Command{Description: "List all agent workspaces with status", Action: s.cmdWorkspaceList}); !r.OK {
		return r
	}
	if r := c.Command("workspace/clean", core.Command{Description: "Remove completed/failed/blocked workspaces", Action: s.cmdWorkspaceClean}); !r.OK {
		return r
	}
	if r := c.Command("agentic:workspace/clean", core.Command{Description: "Remove completed/failed/blocked workspaces", Action: s.cmdWorkspaceClean}); !r.OK {
		return r
	}
	if r := c.Command("workspace/stats", core.Command{Description: "List permanent dispatch stats from .core/workspace/db.duckdb", Action: s.cmdWorkspaceStats}); !r.OK {
		return r
	}
	if r := c.Command("agentic:workspace/stats", core.Command{Description: "List permanent dispatch stats from .core/workspace/db.duckdb", Action: s.cmdWorkspaceStats}); !r.OK {
		return r
	}
	if r := c.Command("workspace/dispatch", core.Command{Description: "Dispatch an agent to work on a repo task", Action: s.cmdWorkspaceDispatch}); !r.OK {
		return r
	}
	if r := c.Command("agentic:workspace/dispatch", core.Command{Description: "Dispatch an agent to work on a repo task", Action: s.cmdWorkspaceDispatch}); !r.OK {
		return r
	}
	if r := c.Command("workspace/watch", core.Command{Description: "Watch workspaces until they complete", Action: s.cmdWorkspaceWatch}); !r.OK {
		return r
	}
	if r := c.Command("agentic:workspace/watch", core.Command{Description: "Watch workspaces until they complete", Action: s.cmdWorkspaceWatch}); !r.OK {
		return r
	}
	if r := c.Command("watch", core.Command{Description: "Watch workspaces until they complete", Action: s.cmdWorkspaceWatch}); !r.OK {
		return r
	}
	if r := c.Command("agentic:watch", core.Command{Description: "Watch workspaces until they complete", Action: s.cmdWorkspaceWatch}); !r.OK {
		return r
	}
	return core.Ok(nil)
}

// workspaceListItem is the JSON shape of `workspace/list --json` — one row
// per tracked workspace, what the desktop CLI adapter parses.
type workspaceListItem struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Agent    string `json:"agent"`
	Repo     string `json:"repo"`
	Org      string `json:"org,omitempty"`
	Task     string `json:"task,omitempty"`
	Branch   string `json:"branch,omitempty"`
	Issue    int    `json:"issue,omitempty"`
	Question string `json:"question,omitempty"`
	Runs     int    `json:"runs"`
	PRURL    string `json:"pr_url,omitempty"`
}

func (s *PrepSubsystem) cmdWorkspaceList(options core.Options) core.Result {
	statusFiles := WorkspaceStatusPaths()
	items := make([]workspaceListItem, 0, len(statusFiles))
	for _, sf := range statusFiles {
		workspaceDir := core.PathDir(sf)
		workspaceName := WorkspaceName(workspaceDir)
		result := ReadStatusResult(workspaceDir)
		workspaceStatus, ok := workspaceStatusValue(result)
		if !ok {
			continue
		}
		items = append(items, workspaceListItem{
			Name:     workspaceName,
			Status:   workspaceStatus.Status,
			Agent:    workspaceStatus.Agent,
			Repo:     workspaceStatus.Repo,
			Org:      workspaceStatus.Org,
			Task:     workspaceStatus.Task,
			Branch:   workspaceStatus.Branch,
			Issue:    workspaceStatus.Issue,
			Question: workspaceStatus.Question,
			Runs:     workspaceStatus.Runs,
			PRURL:    workspaceStatus.PRURL,
		})
	}

	if emitCommandJSON(options, items) {
		return core.Result{OK: true}
	}

	for _, it := range items {
		core.Print(nil, "  %-8s %-8s %-10s %s", it.Status, it.Agent, it.Repo, it.Name)
	}
	if len(items) == 0 {
		core.Print(nil, "  no workspaces")
	}
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdWorkspaceClean(options core.Options) core.Result {
	workspaceRoot := WorkspaceRoot()
	filesystem := s.Core().Fs()
	filter := options.String("_arg")
	if filter == "" {
		filter = "all"
	}
	if !workspaceCleanFilterValid(filter) {
		core.Print(nil, "usage: core-agent workspace clean [all|completed|failed|blocked]")
		return core.Result{Value: core.E("agentic.cmdWorkspaceClean", core.Concat("unknown filter: ", filter), nil), OK: false}
	}

	statusFiles := WorkspaceStatusPaths()
	var toRemove []string

	for _, sf := range statusFiles {
		workspaceDir := core.PathDir(sf)
		workspaceName := WorkspaceName(workspaceDir)
		result := ReadStatusResult(workspaceDir)
		workspaceStatus, ok := workspaceStatusValue(result)
		if !ok {
			continue
		}
		status := workspaceStatus.Status

		switch filter {
		case "all":
			if status == "completed" || status == "failed" || status == "blocked" || status == "merged" || status == "ready-for-review" {
				toRemove = append(toRemove, workspaceName)
			}
		case "completed":
			if status == "completed" || status == "merged" || status == "ready-for-review" {
				toRemove = append(toRemove, workspaceName)
			}
		case "failed":
			if status == "failed" {
				toRemove = append(toRemove, workspaceName)
			}
		case "blocked":
			if status == "blocked" {
				toRemove = append(toRemove, workspaceName)
			}
		}
	}

	if len(toRemove) == 0 {
		core.Print(nil, "nothing to clean")
		return core.Result{OK: true}
	}

	for _, name := range toRemove {
		path := core.JoinPath(workspaceRoot, name)
		// RFC §15.5 — stats MUST be captured to `.core/workspace/db.duckdb`
		// before the workspace directory is deleted so the permanent record
		// of the dispatch survives cleanup.
		if result := ReadStatusResult(path); result.OK {
			if st, ok := workspaceStatusValue(result); ok {
				s.recordWorkspaceStats(path, st)
			}
		}
		if result := filesystem.DeleteAll(path); !result.OK {
			return result
		}
		core.Print(nil, "  removed %s", name)
	}
	core.Print(nil, "\n  %d workspaces removed", len(toRemove))
	return core.Result{OK: true}
}

func workspaceCleanFilterValid(filter string) bool {
	switch filter {
	case "all", "completed", "failed", "blocked":
		return true
	default:
		return false
	}
}

// cmdWorkspaceStats prints the last N dispatch stats rows persisted in the
// parent workspace store. `core-agent workspace stats` answers "what
// happened in the last 50 dispatches?" — the exact use case RFC §15.5 names
// as the reason for the permanent record. The default limit is 50 to match
// the spec.
//
// Usage example: `core-agent workspace stats --repo=go-io --status=completed --limit=20`
func (s *PrepSubsystem) cmdWorkspaceStats(options core.Options) core.Result {
	limit := options.Int("limit")
	if limit <= 0 {
		limit = 50
	}
	repo := options.String("repo")
	status := options.String("status")

	rows := filterWorkspaceStats(s.listWorkspaceStats(), repo, status, limit)
	if len(rows) == 0 {
		core.Print(nil, "  no recorded dispatches")
		return core.Result{OK: true}
	}

	core.Print(nil, "  %-30s %-12s %-18s %-10s %-6s %s", "WORKSPACE", "STATUS", "AGENT", "DURATION", "FINDS", "COMPLETED")
	for _, row := range rows {
		core.Print(nil, "  %-30s %-12s %-18s %-10s %-6d %s",
			row.Workspace,
			row.Status,
			row.Agent,
			core.Sprintf("%dms", row.DurationMS),
			row.FindingsTotal,
			row.CompletedAt,
		)
	}
	core.Print(nil, "\n  %d rows", len(rows))
	return core.Result{OK: true}
}

// input := DispatchInput{Repo: "go-io", Task: "Fix the failing tests", Issue: 12}
func (s *PrepSubsystem) cmdWorkspaceDispatch(options core.Options) core.Result {
	input := workspaceDispatchInputFromOptions(options)
	if input.Repo == "" {
		core.Print(nil, "usage: core-agent workspace dispatch <repo> --task=\"...\" --issue=N|--pr=N|--branch=X [--agent=codex] [--template=coding] [--plan-template=bug-fix] [--persona=code/reviewer] [--tag=v0.8.0] [--dry-run]")
		return core.Result{Value: core.E("agentic.cmdWorkspaceDispatch", "repo is required", nil), OK: false}
	}

	_, out, err := dispatch(s, context.Background(), nil, input)
	if err != nil {
		core.Print(nil, "dispatch failed: %s", err.Error())
		return core.Result{Value: err, OK: false}
	}

	if emitCommandJSON(options, out) {
		return core.Result{Value: out, OK: true}
	}

	agent := out.Agent
	if agent == "" {
		agent = "codex"
	}
	core.Print(nil, "dispatched %s to %s", agent, input.Repo)
	if out.WorkspaceDir != "" {
		core.Print(nil, "  workspace: %s", out.WorkspaceDir)
	}
	if out.PID > 0 {
		core.Print(nil, "  pid:       %d", out.PID)
	}
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdWorkspaceWatch(options core.Options) core.Result {
	watchOptions := core.NewOptions(options.Items()...)
	if watchOptions.String("workspace") == "" && len(optionStringSliceValue(watchOptions, "workspaces")) == 0 {
		if workspace := optionStringValue(options, "_arg"); workspace != "" {
			watchOptions.Set("workspace", workspace)
		}
	}
	input := watchInputFromOptions(watchOptions)

	_, output, err := watch(s, s.commandContext(), nil, input)
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	if emitCommandJSON(options, output) {
		return core.Result{Value: output, OK: output.Success}
	}

	core.Print(nil, "completed: %d", len(output.Completed))
	core.Print(nil, "failed:    %d", len(output.Failed))
	core.Print(nil, "duration:  %s", output.Duration)
	return core.Result{Value: output, OK: output.Success}
}

func workspaceDispatchInputFromOptions(options core.Options) DispatchInput {
	dispatchOptions := core.NewOptions(options.Items()...)
	if dispatchOptions.String("repo") == "" {
		if repo := optionStringValue(options, "_arg", "repo"); repo != "" {
			dispatchOptions.Set("repo", repo)
		}
	}
	return dispatchInputFromOptions(dispatchOptions)
}
