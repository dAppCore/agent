// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go/core"
)

type SyncPushInput struct {
	AgentID string `json:"agent_id,omitempty"`
}

type SyncPushOutput struct {
	Success bool `json:"success"`
	Count   int  `json:"count"`
}

type SyncPullInput struct {
	AgentID string `json:"agent_id,omitempty"`
}

type SyncPullOutput struct {
	Success bool             `json:"success"`
	Count   int              `json:"count"`
	Context []map[string]any `json:"context"`
}

// result := c.Action("agent.sync.push").Run(ctx, core.NewOptions())
func (s *PrepSubsystem) handleSyncPush(ctx context.Context, options core.Options) core.Result {
	output, err := s.syncPush(ctx, options.String("agent_id"))
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("agent.sync.pull").Run(ctx, core.NewOptions())
func (s *PrepSubsystem) handleSyncPull(ctx context.Context, options core.Options) core.Result {
	output, err := s.syncPull(ctx, options.String("agent_id"))
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) syncPush(ctx context.Context, agentID string) (SyncPushOutput, error) {
	if agentID == "" {
		agentID = AgentName()
	}
	token := s.syncToken()
	if token == "" {
		return SyncPushOutput{}, core.E("agent.sync.push", "api token is required", nil)
	}

	dispatches := collectSyncDispatches()
	if len(dispatches) == 0 {
		return SyncPushOutput{Success: true, Count: 0}, nil
	}

	payload := map[string]any{
		"agent_id":   agentID,
		"dispatches": dispatches,
	}

	result := HTTPPost(ctx, core.Concat(s.syncAPIURL(), "/v1/agent/sync"), core.JSONMarshalString(payload), token, "Bearer")
	if !result.OK {
		err, _ := result.Value.(error)
		if err == nil {
			err = core.E("agent.sync.push", "sync push failed", nil)
		}
		return SyncPushOutput{}, err
	}

	return SyncPushOutput{Success: true, Count: len(dispatches)}, nil
}

func (s *PrepSubsystem) syncPull(ctx context.Context, agentID string) (SyncPullOutput, error) {
	if agentID == "" {
		agentID = AgentName()
	}
	token := s.syncToken()
	if token == "" {
		return SyncPullOutput{}, core.E("agent.sync.pull", "api token is required", nil)
	}

	endpoint := core.Concat(s.syncAPIURL(), "/v1/agent/context?agent_id=", agentID)
	result := HTTPGet(ctx, endpoint, token, "Bearer")
	if !result.OK {
		err, _ := result.Value.(error)
		if err == nil {
			err = core.E("agent.sync.pull", "sync pull failed", nil)
		}
		return SyncPullOutput{}, err
	}

	var response struct {
		Data []map[string]any `json:"data"`
	}
	parseResult := core.JSONUnmarshalString(result.Value.(string), &response)
	if !parseResult.OK {
		err, _ := parseResult.Value.(error)
		if err == nil {
			err = core.E("agent.sync.pull", "failed to parse sync response", nil)
		}
		return SyncPullOutput{}, err
	}

	return SyncPullOutput{
		Success: true,
		Count:   len(response.Data),
		Context: response.Data,
	}, nil
}

func (s *PrepSubsystem) syncAPIURL() string {
	if value := core.Env("CORE_API_URL"); value != "" {
		return value
	}
	if s != nil && s.brainURL != "" {
		return s.brainURL
	}
	return "https://api.lthn.sh"
}

func (s *PrepSubsystem) syncToken() string {
	if value := core.Env("CORE_AGENT_API_KEY"); value != "" {
		return value
	}
	if value := core.Env("CORE_BRAIN_KEY"); value != "" {
		return value
	}
	if s != nil && s.brainKey != "" {
		return s.brainKey
	}
	return ""
}

func collectSyncDispatches() []map[string]any {
	var dispatches []map[string]any
	for _, path := range WorkspaceStatusPaths() {
		workspaceDir := core.PathDir(path)
		statusResult := ReadStatusResult(workspaceDir)
		workspaceStatus, ok := workspaceStatusValue(statusResult)
		if !ok {
			continue
		}
		if !shouldSyncStatus(workspaceStatus.Status) {
			continue
		}
		dispatches = append(dispatches, map[string]any{
			"workspace":  WorkspaceName(workspaceDir),
			"repo":       workspaceStatus.Repo,
			"org":        workspaceStatus.Org,
			"task":       workspaceStatus.Task,
			"agent":      workspaceStatus.Agent,
			"branch":     workspaceStatus.Branch,
			"status":     workspaceStatus.Status,
			"pr_url":     workspaceStatus.PRURL,
			"started_at": workspaceStatus.StartedAt,
			"updated_at": workspaceStatus.UpdatedAt,
		})
	}
	return dispatches
}

func shouldSyncStatus(status string) bool {
	switch status {
	case "completed", "merged", "failed", "blocked":
		return true
	}
	return false
}
