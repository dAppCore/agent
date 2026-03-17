// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	coreerr "forge.lthn.ai/core/go-log"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// WatchInput is the input for agentic_watch.
type WatchInput struct {
	// Workspaces to watch. If empty, watches all running/queued workspaces.
	Workspaces []string `json:"workspaces,omitempty"`
	// PollInterval in seconds (default: 5)
	PollInterval int `json:"poll_interval,omitempty"`
	// Timeout in seconds (default: 600 = 10 minutes)
	Timeout int `json:"timeout,omitempty"`
}

// WatchOutput is the result when all watched workspaces complete.
type WatchOutput struct {
	Success   bool          `json:"success"`
	Completed []WatchResult `json:"completed"`
	Failed    []WatchResult `json:"failed,omitempty"`
	Duration  string        `json:"duration"`
}

// WatchResult describes one completed workspace.
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

func (s *PrepSubsystem) watch(ctx context.Context, req *mcp.CallToolRequest, input WatchInput) (*mcp.CallToolResult, WatchOutput, error) {
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

	// Find workspaces to watch
	targets := input.Workspaces
	if len(targets) == 0 {
		targets = s.findActiveWorkspaces()
	}

	if len(targets) == 0 {
		return nil, WatchOutput{
			Success:  true,
			Duration: "0s",
		}, nil
	}

	var completed []WatchResult
	var failed []WatchResult
	remaining := make(map[string]bool)
	for _, ws := range targets {
		remaining[ws] = true
	}

	progressCount := float64(0)
	total := float64(len(targets))

	// Get progress token from request
	progressToken := req.Params.GetProgressToken()

	// Poll until all complete or timeout
	for len(remaining) > 0 {
		if time.Now().After(deadline) {
			for ws := range remaining {
				failed = append(failed, WatchResult{
					Workspace: ws,
					Status:    "timeout",
				})
			}
			break
		}

		select {
		case <-ctx.Done():
			return nil, WatchOutput{}, coreerr.E("watch", "cancelled", ctx.Err())
		case <-time.After(pollInterval):
		}

		for ws := range remaining {
			wsDir := s.resolveWorkspaceDir(ws)
			st, err := readStatus(wsDir)
			if err != nil {
				continue
			}

			switch st.Status {
			case "completed":
				result := WatchResult{
					Workspace: ws,
					Agent:     st.Agent,
					Repo:      st.Repo,
					Status:    "completed",
					PRURL:     st.PRURL,
				}
				completed = append(completed, result)
				delete(remaining, ws)
				progressCount++

				if progressToken != nil && req.Session != nil {
					req.Session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
						ProgressToken: progressToken,
						Progress:      progressCount,
						Total:         total,
						Message:       fmt.Sprintf("%s completed (%s)", st.Repo, st.Agent),
					})
				}

			case "failed", "blocked":
				result := WatchResult{
					Workspace: ws,
					Agent:     st.Agent,
					Repo:      st.Repo,
					Status:    st.Status,
				}
				failed = append(failed, result)
				delete(remaining, ws)
				progressCount++

				if progressToken != nil && req.Session != nil {
					req.Session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
						ProgressToken: progressToken,
						Progress:      progressCount,
						Total:         total,
						Message:       fmt.Sprintf("%s %s (%s)", st.Repo, st.Status, st.Agent),
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
	wsRoot := s.workspaceRoot()
	entries, err := filepath.Glob(filepath.Join(wsRoot, "*/status.json"))
	if err != nil {
		return nil
	}

	var active []string
	for _, entry := range entries {
		wsDir := filepath.Dir(entry)
		st, err := readStatus(wsDir)
		if err != nil {
			continue
		}
		if st.Status == "running" || st.Status == "queued" {
			active = append(active, filepath.Base(wsDir))
		}
	}
	return active
}

// resolveWorkspaceDir converts a workspace name to full path.
func (s *PrepSubsystem) resolveWorkspaceDir(name string) string {
	if filepath.IsAbs(name) {
		return name
	}
	return filepath.Join(s.workspaceRoot(), name)
}

// workspaceRoot returns the root directory for agent workspaces.
func (s *PrepSubsystem) workspaceRoot() string {
	return WorkspaceRoot()
}
