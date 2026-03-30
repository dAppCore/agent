// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"time"

	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// WatchInput is the input for agentic_watch.
//
//	input := agentic.WatchInput{Workspaces: []string{"core/go-io/task-42"}, PollInterval: 5, Timeout: 600}
type WatchInput struct {
	// Workspaces to watch. If empty, watches all running/queued workspaces.
	Workspaces []string `json:"workspaces,omitempty"`
	// PollInterval in seconds (default: 5)
	PollInterval int `json:"poll_interval,omitempty"`
	// Timeout in seconds (default: 600 = 10 minutes)
	Timeout int `json:"timeout,omitempty"`
}

// WatchOutput is the result when all watched workspaces complete.
//
//	out := agentic.WatchOutput{Success: true, Completed: []agentic.WatchResult{{Workspace: "core/go-io/task-42", Status: "completed"}}}
type WatchOutput struct {
	Success   bool          `json:"success"`
	Completed []WatchResult `json:"completed"`
	Failed    []WatchResult `json:"failed,omitempty"`
	Duration  string        `json:"duration"`
}

// WatchResult describes one completed workspace.
//
//	result := agentic.WatchResult{Workspace: "core/go-io/task-42", Agent: "codex", Repo: "go-io", Status: "completed"}
type WatchResult struct {
	Workspace string `json:"workspace"`
	Agent     string `json:"agent"`
	Repo      string `json:"repo"`
	Status    string `json:"status"`
	PRURL     string `json:"pr_url,omitempty"`
}

func (s *PrepSubsystem) registerWatchTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_watch",
		Description: "Watch running/queued agent workspaces until they all complete. Sends progress notifications as each agent finishes. Returns summary when all are done.",
	}, s.watch)
}

func (s *PrepSubsystem) watch(ctx context.Context, request *mcp.CallToolRequest, input WatchInput) (*mcp.CallToolResult, WatchOutput, error) {
	pollInterval := time.Duration(input.PollInterval) * time.Second
	if pollInterval <= 0 {
		pollInterval = 5 * time.Second
	}
	timeout := time.Duration(input.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}

	start := time.Now()
	deadline := start.Add(timeout)

	workspaceNames := input.Workspaces
	if len(workspaceNames) == 0 {
		workspaceNames = s.findActiveWorkspaces()
	}

	if len(workspaceNames) == 0 {
		return nil, WatchOutput{
			Success:  true,
			Duration: "0s",
		}, nil
	}

	var completed []WatchResult
	var failed []WatchResult
	pendingWorkspaces := make(map[string]bool)
	for _, workspaceName := range workspaceNames {
		pendingWorkspaces[workspaceName] = true
	}

	progressCount := float64(0)
	total := float64(len(workspaceNames))

	// MCP tests and internal callers may not provide a full request envelope.
	progressToken := any(nil)
	if request != nil && request.Params != nil {
		progressToken = request.Params.GetProgressToken()
	}

	for len(pendingWorkspaces) > 0 {
		if time.Now().After(deadline) {
			for workspaceName := range pendingWorkspaces {
				failed = append(failed, WatchResult{
					Workspace: workspaceName,
					Status:    "timeout",
				})
			}
			break
		}

		select {
		case <-ctx.Done():
			return nil, WatchOutput{}, core.E("watch", "cancelled", ctx.Err())
		case <-time.After(pollInterval):
		}

		for workspaceName := range pendingWorkspaces {
			workspaceDir := s.resolveWorkspaceDir(workspaceName)
			statusResult := ReadStatusResult(workspaceDir)
			workspaceStatus, ok := workspaceStatusValue(statusResult)
			if !ok {
				continue
			}

			switch workspaceStatus.Status {
			case "completed":
				watchResult := WatchResult{
					Workspace: workspaceName,
					Agent:     workspaceStatus.Agent,
					Repo:      workspaceStatus.Repo,
					Status:    "completed",
					PRURL:     workspaceStatus.PRURL,
				}
				completed = append(completed, watchResult)
				delete(pendingWorkspaces, workspaceName)
				progressCount++

				if request != nil && progressToken != nil && request.Session != nil {
					request.Session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
						ProgressToken: progressToken,
						Progress:      progressCount,
						Total:         total,
						Message:       core.Sprintf("%s completed (%s)", workspaceStatus.Repo, workspaceStatus.Agent),
					})
				}

			case "merged", "ready-for-review":
				watchResult := WatchResult{
					Workspace: workspaceName,
					Agent:     workspaceStatus.Agent,
					Repo:      workspaceStatus.Repo,
					Status:    workspaceStatus.Status,
					PRURL:     workspaceStatus.PRURL,
				}
				completed = append(completed, watchResult)
				delete(pendingWorkspaces, workspaceName)
				progressCount++

				if request != nil && progressToken != nil && request.Session != nil {
					request.Session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
						ProgressToken: progressToken,
						Progress:      progressCount,
						Total:         total,
						Message:       core.Sprintf("%s %s (%s)", workspaceStatus.Repo, workspaceStatus.Status, workspaceStatus.Agent),
					})
				}

			case "failed", "blocked":
				watchResult := WatchResult{
					Workspace: workspaceName,
					Agent:     workspaceStatus.Agent,
					Repo:      workspaceStatus.Repo,
					Status:    workspaceStatus.Status,
				}
				failed = append(failed, watchResult)
				delete(pendingWorkspaces, workspaceName)
				progressCount++

				if request != nil && progressToken != nil && request.Session != nil {
					request.Session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
						ProgressToken: progressToken,
						Progress:      progressCount,
						Total:         total,
						Message:       core.Sprintf("%s %s (%s)", workspaceStatus.Repo, workspaceStatus.Status, workspaceStatus.Agent),
					})
				}
			}
		}
	}

	return nil, WatchOutput{
		Success:   len(failed) == 0,
		Completed: completed,
		Failed:    failed,
		Duration:  time.Since(start).Round(time.Second).String(),
	}, nil
}

// findActiveWorkspaces returns workspace names that are running or queued.
func (s *PrepSubsystem) findActiveWorkspaces() []string {
	var active []string
	for _, entry := range WorkspaceStatusPaths() {
		workspaceDir := core.PathDir(entry)
		statusResult := ReadStatusResult(workspaceDir)
		workspaceStatus, ok := workspaceStatusValue(statusResult)
		if !ok {
			continue
		}
		if workspaceStatus.Status == "running" || workspaceStatus.Status == "queued" {
			active = append(active, WorkspaceName(workspaceDir))
		}
	}
	return active
}

// resolveWorkspaceDir converts a workspace name to full path.
func (s *PrepSubsystem) resolveWorkspaceDir(workspaceName string) string {
	if core.PathIsAbs(workspaceName) {
		return workspaceName
	}
	return core.JoinPath(WorkspaceRoot(), workspaceName)
}
