// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go/core"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// input := agentic.SyncStatusInput{AgentID: "charon"}
type SyncStatusInput struct {
	AgentID string `json:"agent_id,omitempty"`
}

// input := agentic.FleetDeregisterInput{AgentID: "charon"}
type FleetDeregisterInput struct {
	AgentID string `json:"agent_id"`
}

// input := agentic.FleetTaskAssignInput{AgentID: "charon", Repo: "core/go-io", Task: "Fix tests"}
type FleetTaskAssignInput struct {
	AgentID    string `json:"agent_id"`
	Repo       string `json:"repo"`
	Branch     string `json:"branch,omitempty"`
	Task       string `json:"task"`
	Template   string `json:"template,omitempty"`
	AgentModel string `json:"agent_model,omitempty"`
}

// input := agentic.FleetTaskCompleteInput{AgentID: "charon", TaskID: 7}
type FleetTaskCompleteInput struct {
	AgentID  string           `json:"agent_id"`
	TaskID   int              `json:"task_id"`
	Result   map[string]any   `json:"result,omitempty"`
	Findings []map[string]any `json:"findings,omitempty"`
	Changes  map[string]any   `json:"changes,omitempty"`
	Report   map[string]any   `json:"report,omitempty"`
}

// input := agentic.FleetEventsInput{AgentID: "charon", Capabilities: []string{"go", "review"}}
type FleetEventsInput struct {
	AgentID      string   `json:"agent_id,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
}

// input := agentic.FleetNodesInput{Status: "online", Platform: "linux"}
type FleetNodesInput struct {
	Status   string `json:"status,omitempty"`
	Platform string `json:"platform,omitempty"`
}

// input := agentic.FleetTaskNextInput{AgentID: "charon", Capabilities: []string{"go", "review"}}
type FleetTaskNextInput struct {
	AgentID      string   `json:"agent_id"`
	Capabilities []string `json:"capabilities,omitempty"`
}

// input := agentic.CreditsAwardInput{AgentID: "charon", TaskType: "fleet-task", Amount: 2}
type CreditsAwardInput struct {
	AgentID     string `json:"agent_id"`
	TaskType    string `json:"task_type"`
	Amount      int    `json:"amount"`
	FleetNodeID int    `json:"fleet_node_id,omitempty"`
	Description string `json:"description,omitempty"`
}

// input := agentic.CreditsBalanceInput{AgentID: "charon"}
type CreditsBalanceInput struct {
	AgentID string `json:"agent_id"`
}

// input := agentic.CreditsHistoryInput{AgentID: "charon", Limit: 50}
type CreditsHistoryInput struct {
	AgentID string `json:"agent_id"`
	Limit   int    `json:"limit,omitempty"`
}

// input := agentic.SubscriptionDetectInput{APIKeys: map[string]string{"openai": "sk-..."}}
type SubscriptionDetectInput struct {
	APIKeys map[string]string `json:"api_keys,omitempty"`
}

// input := agentic.SubscriptionBudgetInput{AgentID: "charon"}
type SubscriptionBudgetInput struct {
	AgentID string `json:"agent_id"`
}

// input := agentic.SubscriptionBudgetUpdateInput{AgentID: "charon", Limits: map[string]any{"max_daily_hours": 2}}
type SubscriptionBudgetUpdateInput struct {
	AgentID string         `json:"agent_id"`
	Limits  map[string]any `json:"limits"`
}

func (s *PrepSubsystem) registerPlatformTools(svc *coremcp.Service) {
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_sync_push",
		Description: "Push completed dispatch state to the platform API for fleet-wide context sharing.",
	}, s.syncPushTool)

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_sync_pull",
		Description: "Pull fleet-wide context from the platform API into the local cache.",
	}, s.syncPullTool)

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_sync_status",
		Description: "Read platform sync status for an agent, including queued items and last push/pull times.",
	}, s.syncStatusTool)

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_auth_provision",
		Description: "Provision a platform API key for an authenticated agent user.",
	}, s.authProvisionTool)

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_auth_revoke",
		Description: "Revoke a platform API key by key ID.",
	}, s.authRevokeTool)

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_fleet_register",
		Description: "Register a fleet node with models, capabilities, and platform metadata.",
	}, s.fleetRegisterTool)

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_fleet_heartbeat",
		Description: "Send a fleet heartbeat update with status and optional compute budget.",
	}, s.fleetHeartbeatTool)

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_fleet_deregister",
		Description: "Deregister a fleet node from the platform API.",
	}, s.fleetDeregisterTool)

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_fleet_nodes",
		Description: "List registered fleet nodes with optional status and platform filters.",
	}, s.fleetNodesTool)

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_fleet_task_assign",
		Description: "Assign a task to a fleet node.",
	}, s.fleetTaskAssignTool)

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_fleet_task_complete",
		Description: "Complete a fleet task and report result, findings, changes, and report data.",
	}, s.fleetTaskCompleteTool)

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_fleet_task_next",
		Description: "Ask the platform for the next available fleet task for an agent.",
	}, s.fleetTaskNextTool)

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_fleet_stats",
		Description: "Read aggregate fleet activity statistics.",
	}, s.fleetStatsTool)

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_fleet_events",
		Description: "Read the next fleet event from the platform SSE stream, falling back to polling when needed.",
	}, s.fleetEventsTool)

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_credits_award",
		Description: "Award credits to a fleet node for completed work.",
	}, s.creditsAwardTool)

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_credits_balance",
		Description: "Read the current credit balance for a fleet node.",
	}, s.creditsBalanceTool)

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_credits_history",
		Description: "List credit history entries for a fleet node.",
	}, s.creditsHistoryTool)

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_subscription_detect",
		Description: "Detect provider capabilities available to a fleet node.",
	}, s.subscriptionDetectTool)

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_subscription_budget",
		Description: "Read the current compute budget for a fleet node.",
	}, s.subscriptionBudgetTool)

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_subscription_budget_update",
		Description: "Update the compute budget limits for a fleet node.",
	}, s.subscriptionBudgetUpdateTool)
}

func (s *PrepSubsystem) syncPushTool(ctx context.Context, _ *mcp.CallToolRequest, input SyncPushInput) (*mcp.CallToolResult, SyncPushOutput, error) {
	output, err := s.syncPushInput(ctx, input)
	if err != nil {
		return nil, SyncPushOutput{}, err
	}
	return nil, output, nil
}

func (s *PrepSubsystem) syncPullTool(ctx context.Context, _ *mcp.CallToolRequest, input SyncPullInput) (*mcp.CallToolResult, SyncPullOutput, error) {
	output, err := s.syncPullInput(ctx, input)
	if err != nil {
		return nil, SyncPullOutput{}, err
	}
	return nil, output, nil
}

func (s *PrepSubsystem) syncStatusTool(ctx context.Context, _ *mcp.CallToolRequest, input SyncStatusInput) (*mcp.CallToolResult, SyncStatusOutput, error) {
	result := s.handleSyncStatus(ctx, platformOptions(core.Option{Key: "agent_id", Value: input.AgentID}))
	if !result.OK {
		return nil, SyncStatusOutput{}, resultErrorValue("agentic.sync.status", result)
	}
	output, ok := result.Value.(SyncStatusOutput)
	if !ok {
		return nil, SyncStatusOutput{}, core.E("agentic.sync.status", "invalid sync status output", nil)
	}
	return nil, output, nil
}

func (s *PrepSubsystem) authProvisionTool(ctx context.Context, _ *mcp.CallToolRequest, input AuthProvisionInput) (*mcp.CallToolResult, AuthProvisionOutput, error) {
	options := platformOptions(
		core.Option{Key: "oauth_user_id", Value: input.OAuthUserID},
		core.Option{Key: "name", Value: input.Name},
		core.Option{Key: "permissions", Value: input.Permissions},
		core.Option{Key: "ip_restrictions", Value: input.IPRestrictions},
		core.Option{Key: "rate_limit", Value: input.RateLimit},
		core.Option{Key: "expires_at", Value: input.ExpiresAt},
	)
	result := s.handleAuthProvision(ctx, options)
	if !result.OK {
		return nil, AuthProvisionOutput{}, resultErrorValue("agentic.auth.provision", result)
	}
	output, ok := result.Value.(AuthProvisionOutput)
	if !ok {
		return nil, AuthProvisionOutput{}, core.E("agentic.auth.provision", "invalid auth provision output", nil)
	}
	return nil, output, nil
}

func (s *PrepSubsystem) authRevokeTool(ctx context.Context, _ *mcp.CallToolRequest, input AuthRevokeInput) (*mcp.CallToolResult, AuthRevokeOutput, error) {
	result := s.handleAuthRevoke(ctx, platformOptions(core.Option{Key: "key_id", Value: input.KeyID}))
	if !result.OK {
		return nil, AuthRevokeOutput{}, resultErrorValue("agentic.auth.revoke", result)
	}
	output, ok := result.Value.(AuthRevokeOutput)
	if !ok {
		return nil, AuthRevokeOutput{}, core.E("agentic.auth.revoke", "invalid auth revoke output", nil)
	}
	return nil, output, nil
}

func (s *PrepSubsystem) fleetRegisterTool(ctx context.Context, _ *mcp.CallToolRequest, input FleetNode) (*mcp.CallToolResult, FleetNode, error) {
	options := platformOptions(
		core.Option{Key: "agent_id", Value: input.AgentID},
		core.Option{Key: "platform", Value: input.Platform},
		core.Option{Key: "models", Value: input.Models},
		core.Option{Key: "capabilities", Value: input.Capabilities},
	)
	result := s.handleFleetRegister(ctx, options)
	if !result.OK {
		return nil, FleetNode{}, resultErrorValue("agentic.fleet.register", result)
	}
	output, ok := result.Value.(FleetNode)
	if !ok {
		return nil, FleetNode{}, core.E("agentic.fleet.register", "invalid fleet register output", nil)
	}
	return nil, output, nil
}

func (s *PrepSubsystem) fleetHeartbeatTool(ctx context.Context, _ *mcp.CallToolRequest, input FleetNode) (*mcp.CallToolResult, FleetNode, error) {
	options := platformOptions(
		core.Option{Key: "agent_id", Value: input.AgentID},
		core.Option{Key: "status", Value: input.Status},
		core.Option{Key: "compute_budget", Value: computeBudgetMapValue(input.ComputeBudget)},
	)
	result := s.handleFleetHeartbeat(ctx, options)
	if !result.OK {
		return nil, FleetNode{}, resultErrorValue("agentic.fleet.heartbeat", result)
	}
	output, ok := result.Value.(FleetNode)
	if !ok {
		return nil, FleetNode{}, core.E("agentic.fleet.heartbeat", "invalid fleet heartbeat output", nil)
	}
	return nil, output, nil
}

func computeBudgetMapValue(budget *ComputeBudget) map[string]any {
	if budget == nil || computeBudgetIsZero(*budget) {
		return nil
	}

	values := map[string]any{}
	if budget.MaxDailyHours != 0 {
		values["max_daily_hours"] = budget.MaxDailyHours
	}
	if budget.MaxWeeklyCostUSD != 0 {
		values["max_weekly_cost_usd"] = budget.MaxWeeklyCostUSD
	}
	if trimmed := core.Trim(budget.QuietStart); trimmed != "" {
		values["quiet_start"] = trimmed
	}
	if trimmed := core.Trim(budget.QuietEnd); trimmed != "" {
		values["quiet_end"] = trimmed
	}
	if len(budget.PreferModels) > 0 {
		values["prefer_models"] = cleanStrings(budget.PreferModels)
	}
	if len(budget.AvoidModels) > 0 {
		values["avoid_models"] = cleanStrings(budget.AvoidModels)
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

func (s *PrepSubsystem) fleetDeregisterTool(ctx context.Context, _ *mcp.CallToolRequest, input FleetDeregisterInput) (*mcp.CallToolResult, map[string]any, error) {
	result := s.handleFleetDeregister(ctx, platformOptions(core.Option{Key: "agent_id", Value: input.AgentID}))
	if !result.OK {
		return nil, nil, resultErrorValue("agentic.fleet.deregister", result)
	}
	output, ok := result.Value.(map[string]any)
	if !ok {
		return nil, nil, core.E("agentic.fleet.deregister", "invalid fleet deregister output", nil)
	}
	return nil, output, nil
}

func (s *PrepSubsystem) fleetNodesTool(ctx context.Context, _ *mcp.CallToolRequest, input FleetNodesInput) (*mcp.CallToolResult, FleetNodesOutput, error) {
	result := s.handleFleetNodes(ctx, platformOptions(
		core.Option{Key: "status", Value: input.Status},
		core.Option{Key: "platform", Value: input.Platform},
	))
	if !result.OK {
		return nil, FleetNodesOutput{}, resultErrorValue("agentic.fleet.nodes", result)
	}
	output, ok := result.Value.(FleetNodesOutput)
	if !ok {
		return nil, FleetNodesOutput{}, core.E("agentic.fleet.nodes", "invalid fleet nodes output", nil)
	}
	return nil, output, nil
}

func (s *PrepSubsystem) fleetTaskAssignTool(ctx context.Context, _ *mcp.CallToolRequest, input FleetTaskAssignInput) (*mcp.CallToolResult, FleetTask, error) {
	options := platformOptions(
		core.Option{Key: "agent_id", Value: input.AgentID},
		core.Option{Key: "repo", Value: input.Repo},
		core.Option{Key: "branch", Value: input.Branch},
		core.Option{Key: "task", Value: input.Task},
		core.Option{Key: "template", Value: input.Template},
		core.Option{Key: "agent_model", Value: input.AgentModel},
	)
	result := s.handleFleetAssignTask(ctx, options)
	if !result.OK {
		return nil, FleetTask{}, resultErrorValue("agentic.fleet.task.assign", result)
	}
	output, ok := result.Value.(FleetTask)
	if !ok {
		return nil, FleetTask{}, core.E("agentic.fleet.task.assign", "invalid fleet task output", nil)
	}
	return nil, output, nil
}

func (s *PrepSubsystem) fleetTaskCompleteTool(ctx context.Context, _ *mcp.CallToolRequest, input FleetTaskCompleteInput) (*mcp.CallToolResult, FleetTask, error) {
	result := s.handleFleetCompleteTask(ctx, platformOptions(
		core.Option{Key: "agent_id", Value: input.AgentID},
		core.Option{Key: "task_id", Value: input.TaskID},
		core.Option{Key: "result", Value: input.Result},
		core.Option{Key: "findings", Value: input.Findings},
		core.Option{Key: "changes", Value: input.Changes},
		core.Option{Key: "report", Value: input.Report},
	))
	if !result.OK {
		return nil, FleetTask{}, resultErrorValue("agentic.fleet.task.complete", result)
	}
	output, ok := result.Value.(FleetTask)
	if !ok {
		return nil, FleetTask{}, core.E("agentic.fleet.task.complete", "invalid fleet task output", nil)
	}
	return nil, output, nil
}

func (s *PrepSubsystem) fleetTaskNextTool(ctx context.Context, _ *mcp.CallToolRequest, input FleetTaskNextInput) (*mcp.CallToolResult, *FleetTask, error) {
	result := s.handleFleetNextTask(ctx, platformOptions(
		core.Option{Key: "agent_id", Value: input.AgentID},
		core.Option{Key: "capabilities", Value: input.Capabilities},
	))
	if !result.OK {
		return nil, nil, resultErrorValue("agentic.fleet.task.next", result)
	}
	output, ok := result.Value.(*FleetTask)
	if !ok {
		return nil, nil, core.E("agentic.fleet.task.next", "invalid fleet next-task output", nil)
	}
	return nil, output, nil
}

func (s *PrepSubsystem) fleetStatsTool(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, FleetStats, error) {
	result := s.handleFleetStats(ctx, core.NewOptions())
	if !result.OK {
		return nil, FleetStats{}, resultErrorValue("agentic.fleet.stats", result)
	}
	output, ok := result.Value.(FleetStats)
	if !ok {
		return nil, FleetStats{}, core.E("agentic.fleet.stats", "invalid fleet stats output", nil)
	}
	return nil, output, nil
}

func (s *PrepSubsystem) fleetEventsTool(ctx context.Context, _ *mcp.CallToolRequest, input FleetEventsInput) (*mcp.CallToolResult, FleetEventOutput, error) {
	result := s.handleFleetEvents(ctx, platformOptions(
		core.Option{Key: "agent_id", Value: input.AgentID},
		core.Option{Key: "capabilities", Value: input.Capabilities},
	))
	if !result.OK {
		return nil, FleetEventOutput{}, resultErrorValue("agentic.fleet.events", result)
	}
	output, ok := result.Value.(FleetEventOutput)
	if !ok {
		return nil, FleetEventOutput{}, core.E("agentic.fleet.events", "invalid fleet event output", nil)
	}
	return nil, output, nil
}

func (s *PrepSubsystem) creditsAwardTool(ctx context.Context, _ *mcp.CallToolRequest, input CreditsAwardInput) (*mcp.CallToolResult, CreditEntry, error) {
	result := s.handleCreditsAward(ctx, platformOptions(
		core.Option{Key: "agent_id", Value: input.AgentID},
		core.Option{Key: "task_type", Value: input.TaskType},
		core.Option{Key: "amount", Value: input.Amount},
		core.Option{Key: "fleet_node_id", Value: input.FleetNodeID},
		core.Option{Key: "description", Value: input.Description},
	))
	if !result.OK {
		return nil, CreditEntry{}, resultErrorValue("agentic.credits.award", result)
	}
	output, ok := result.Value.(CreditEntry)
	if !ok {
		return nil, CreditEntry{}, core.E("agentic.credits.award", "invalid credit award output", nil)
	}
	return nil, output, nil
}

func (s *PrepSubsystem) creditsBalanceTool(ctx context.Context, _ *mcp.CallToolRequest, input CreditsBalanceInput) (*mcp.CallToolResult, CreditBalance, error) {
	result := s.handleCreditsBalance(ctx, platformOptions(core.Option{Key: "agent_id", Value: input.AgentID}))
	if !result.OK {
		return nil, CreditBalance{}, resultErrorValue("agentic.credits.balance", result)
	}
	output, ok := result.Value.(CreditBalance)
	if !ok {
		return nil, CreditBalance{}, core.E("agentic.credits.balance", "invalid credit balance output", nil)
	}
	return nil, output, nil
}

func (s *PrepSubsystem) creditsHistoryTool(ctx context.Context, _ *mcp.CallToolRequest, input CreditsHistoryInput) (*mcp.CallToolResult, CreditsHistoryOutput, error) {
	result := s.handleCreditsHistory(ctx, platformOptions(
		core.Option{Key: "agent_id", Value: input.AgentID},
		core.Option{Key: "limit", Value: input.Limit},
	))
	if !result.OK {
		return nil, CreditsHistoryOutput{}, resultErrorValue("agentic.credits.history", result)
	}
	output, ok := result.Value.(CreditsHistoryOutput)
	if !ok {
		return nil, CreditsHistoryOutput{}, core.E("agentic.credits.history", "invalid credit history output", nil)
	}
	return nil, output, nil
}

func (s *PrepSubsystem) subscriptionDetectTool(ctx context.Context, _ *mcp.CallToolRequest, input SubscriptionDetectInput) (*mcp.CallToolResult, SubscriptionCapabilities, error) {
	result := s.handleSubscriptionDetect(ctx, platformOptions(core.Option{Key: "api_keys", Value: input.APIKeys}))
	if !result.OK {
		return nil, SubscriptionCapabilities{}, resultErrorValue("agentic.subscription.detect", result)
	}
	output, ok := result.Value.(SubscriptionCapabilities)
	if !ok {
		return nil, SubscriptionCapabilities{}, core.E("agentic.subscription.detect", "invalid capability output", nil)
	}
	return nil, output, nil
}

func (s *PrepSubsystem) subscriptionBudgetTool(ctx context.Context, _ *mcp.CallToolRequest, input SubscriptionBudgetInput) (*mcp.CallToolResult, map[string]any, error) {
	result := s.handleSubscriptionBudget(ctx, platformOptions(core.Option{Key: "agent_id", Value: input.AgentID}))
	if !result.OK {
		return nil, nil, resultErrorValue("agentic.subscription.budget", result)
	}
	output, ok := result.Value.(map[string]any)
	if !ok {
		return nil, nil, core.E("agentic.subscription.budget", "invalid budget output", nil)
	}
	return nil, output, nil
}

func (s *PrepSubsystem) subscriptionBudgetUpdateTool(ctx context.Context, _ *mcp.CallToolRequest, input SubscriptionBudgetUpdateInput) (*mcp.CallToolResult, map[string]any, error) {
	result := s.handleSubscriptionBudgetUpdate(ctx, platformOptions(
		core.Option{Key: "agent_id", Value: input.AgentID},
		core.Option{Key: "limits", Value: input.Limits},
	))
	if !result.OK {
		return nil, nil, resultErrorValue("agentic.subscription.budget.update", result)
	}
	output, ok := result.Value.(map[string]any)
	if !ok {
		return nil, nil, core.E("agentic.subscription.budget.update", "invalid updated budget output", nil)
	}
	return nil, output, nil
}

func platformOptions(options ...core.Option) core.Options {
	filtered := make([]core.Option, 0, len(options))
	for _, option := range options {
		switch value := option.Value.(type) {
		case string:
			if core.Trim(value) == "" {
				continue
			}
		case []string:
			if len(value) == 0 {
				continue
			}
		case map[string]any:
			if len(value) == 0 {
				continue
			}
		case map[string]string:
			if len(value) == 0 {
				continue
			}
		case int:
			if value == 0 {
				continue
			}
		}
		filtered = append(filtered, option)
	}
	return core.NewOptions(filtered...)
}
