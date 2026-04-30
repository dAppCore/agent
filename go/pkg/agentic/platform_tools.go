// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go"
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
	}, toolHandlerFor[SyncPushInput, SyncPushOutput]("agentic.sync.push", "invalid sync push output", s.syncPushTool))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_sync_pull",
		Description: "Pull fleet-wide context from the platform API into the local cache.",
	}, toolHandlerFor[SyncPullInput, SyncPullOutput]("agentic.sync.pull", "invalid sync pull output", s.syncPullTool))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_sync_status",
		Description: "Read platform sync status for an agent, including queued items and last push/pull times.",
	}, toolHandlerFor[SyncStatusInput, SyncStatusOutput]("agentic.sync.status", "invalid sync status output", s.syncStatusTool))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_auth_provision",
		Description: "Provision a platform API key for an authenticated agent user.",
	}, toolHandlerFor[AuthProvisionInput, AuthProvisionOutput]("agentic.auth.provision", "invalid auth provision output", s.authProvisionTool))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_auth_revoke",
		Description: "Revoke a platform API key by key ID.",
	}, toolHandlerFor[AuthRevokeInput, AuthRevokeOutput]("agentic.auth.revoke", "invalid auth revoke output", s.authRevokeTool))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_auth_login",
		Description: "Exchange a 6-digit pairing code (generated at app.lthn.ai/device) for an AgentApiKey. Bootstraps a fleet node without requiring an existing API key — RFC §9 Fleet Mode.",
	}, toolHandlerFor[AuthLoginInput, AuthLoginOutput]("agentic.auth.login", "invalid auth login output", s.authLoginTool))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_fleet_register",
		Description: "Register a fleet node with models, capabilities, and platform metadata.",
	}, toolHandlerFor[FleetNode, FleetNode]("agentic.fleet.register", "invalid fleet register output", s.fleetRegisterTool))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_fleet_heartbeat",
		Description: "Send a fleet heartbeat update with status and optional compute budget.",
	}, toolHandlerFor[FleetNode, FleetNode]("agentic.fleet.heartbeat", "invalid fleet heartbeat output", s.fleetHeartbeatTool))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_fleet_deregister",
		Description: "Deregister a fleet node from the platform API.",
	}, toolHandlerFor[FleetDeregisterInput, map[string]any]("agentic.fleet.deregister", "invalid fleet deregister output", s.fleetDeregisterTool))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_fleet_nodes",
		Description: "List registered fleet nodes with optional status and platform filters.",
	}, toolHandlerFor[FleetNodesInput, FleetNodesOutput]("agentic.fleet.nodes", "invalid fleet nodes output", s.fleetNodesTool))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_fleet_task_assign",
		Description: "Assign a task to a fleet node.",
	}, toolHandlerFor[FleetTaskAssignInput, FleetTask]("agentic.fleet.task.assign", "invalid fleet task output", s.fleetTaskAssignTool))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_fleet_task_complete",
		Description: "Complete a fleet task and report result, findings, changes, and report data.",
	}, toolHandlerFor[FleetTaskCompleteInput, FleetTask]("agentic.fleet.task.complete", "invalid fleet task output", s.fleetTaskCompleteTool))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_fleet_task_next",
		Description: "Ask the platform for the next available fleet task for an agent.",
	}, toolHandlerFor[FleetTaskNextInput, *FleetTask]("agentic.fleet.task.next", "invalid fleet next-task output", s.fleetTaskNextTool))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_fleet_stats",
		Description: "Read aggregate fleet activity statistics.",
	}, toolHandlerFor[struct{}, FleetStats]("agentic.fleet.stats", "invalid fleet stats output", s.fleetStatsTool))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_fleet_events",
		Description: "Read the next fleet event from the platform SSE stream, falling back to polling when needed.",
	}, toolHandlerFor[FleetEventsInput, FleetEventOutput]("agentic.fleet.events", "invalid fleet event output", s.fleetEventsTool))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_credits_award",
		Description: "Award credits to a fleet node for completed work.",
	}, toolHandlerFor[CreditsAwardInput, CreditEntry]("agentic.credits.award", "invalid credit award output", s.creditsAwardTool))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_credits_balance",
		Description: "Read the current credit balance for a fleet node.",
	}, toolHandlerFor[CreditsBalanceInput, CreditBalance]("agentic.credits.balance", "invalid credit balance output", s.creditsBalanceTool))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_credits_history",
		Description: "List credit history entries for a fleet node.",
	}, toolHandlerFor[CreditsHistoryInput, CreditsHistoryOutput]("agentic.credits.history", "invalid credit history output", s.creditsHistoryTool))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_subscription_detect",
		Description: "Detect provider capabilities available to a fleet node.",
	}, toolHandlerFor[SubscriptionDetectInput, SubscriptionCapabilities]("agentic.subscription.detect", "invalid capability output", s.subscriptionDetectTool))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_subscription_budget",
		Description: "Read the current compute budget for a fleet node.",
	}, toolHandlerFor[SubscriptionBudgetInput, map[string]any]("agentic.subscription.budget", "invalid budget output", s.subscriptionBudgetTool))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_subscription_budget_update",
		Description: "Update the compute budget limits for a fleet node.",
	}, toolHandlerFor[SubscriptionBudgetUpdateInput, map[string]any]("agentic.subscription.budget.update", "invalid updated budget output", s.subscriptionBudgetUpdateTool))
}

func (s *PrepSubsystem) syncPushTool(ctx context.Context, input SyncPushInput) core.Result {
	output, err := syncPushInput(s, ctx, input)
	if err != nil {
		return core.Fail(err)
	}
	return core.Ok(output)
}

func (s *PrepSubsystem) syncPullTool(ctx context.Context, input SyncPullInput) core.Result {
	output, err := syncPullInput(s, ctx, input)
	if err != nil {
		return core.Fail(err)
	}
	return core.Ok(output)
}

func (s *PrepSubsystem) syncStatusTool(ctx context.Context, input SyncStatusInput) core.Result {
	return typedResultValue[SyncStatusOutput]("agentic.sync.status", "invalid sync status output", s.handleSyncStatus(ctx, platformOptions(core.Option{Key: "agent_id", Value: input.AgentID})))
}

func (s *PrepSubsystem) authProvisionTool(ctx context.Context, input AuthProvisionInput) core.Result {
	options := platformOptions(
		core.Option{Key: "oauth_user_id", Value: input.OAuthUserID},
		core.Option{Key: "name", Value: input.Name},
		core.Option{Key: "permissions", Value: input.Permissions},
		core.Option{Key: "ip_restrictions", Value: input.IPRestrictions},
		core.Option{Key: "rate_limit", Value: input.RateLimit},
		core.Option{Key: "expires_at", Value: input.ExpiresAt},
	)
	return typedResultValue[AuthProvisionOutput]("agentic.auth.provision", "invalid auth provision output", s.handleAuthProvision(ctx, options))
}

func (s *PrepSubsystem) authRevokeTool(ctx context.Context, input AuthRevokeInput) core.Result {
	return typedResultValue[AuthRevokeOutput]("agentic.auth.revoke", "invalid auth revoke output", s.handleAuthRevoke(ctx, platformOptions(core.Option{Key: "key_id", Value: input.KeyID})))
}

// authLoginTool handles the MCP-side of the RFC §9 pairing-code bootstrap.
// Callers pass a 6-digit pairing code generated at app.lthn.ai/device and
// receive the provisioned AgentApiKey so the node can authenticate future
// platform calls. The code itself is the proof — no existing API key is
// required.
//
// Usage example:
//
//	out, _ := clientSession.CallTool(ctx, &mcp.CallToolParams{
//	    Name:      "agentic_auth_login",
//	    Arguments: json.RawMessage(`{"code": "123456"}`),
//	})
func (s *PrepSubsystem) authLoginTool(ctx context.Context, input AuthLoginInput) core.Result {
	return typedResultValue[AuthLoginOutput]("agentic.auth.login", "invalid auth login output", s.handleAuthLogin(ctx, platformOptions(core.Option{Key: "code", Value: input.Code})))
}

func (s *PrepSubsystem) fleetRegisterTool(ctx context.Context, input FleetNode) core.Result {
	options := platformOptions(
		core.Option{Key: "agent_id", Value: input.AgentID},
		core.Option{Key: "platform", Value: input.Platform},
		core.Option{Key: "models", Value: input.Models},
		core.Option{Key: "capabilities", Value: input.Capabilities},
	)
	return typedResultValue[FleetNode]("agentic.fleet.register", "invalid fleet register output", s.handleFleetRegister(ctx, options))
}

func (s *PrepSubsystem) fleetHeartbeatTool(ctx context.Context, input FleetNode) core.Result {
	options := platformOptions(
		core.Option{Key: "agent_id", Value: input.AgentID},
		core.Option{Key: "status", Value: input.Status},
		core.Option{Key: "compute_budget", Value: computeBudgetMapValue(input.ComputeBudget)},
	)
	return typedResultValue[FleetNode]("agentic.fleet.heartbeat", "invalid fleet heartbeat output", s.handleFleetHeartbeat(ctx, options))
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

func (s *PrepSubsystem) fleetDeregisterTool(ctx context.Context, input FleetDeregisterInput) core.Result {
	return typedResultValue[map[string]any]("agentic.fleet.deregister", "invalid fleet deregister output", s.handleFleetDeregister(ctx, platformOptions(core.Option{Key: "agent_id", Value: input.AgentID})))
}

func (s *PrepSubsystem) fleetNodesTool(ctx context.Context, input FleetNodesInput) core.Result {
	return typedResultValue[FleetNodesOutput]("agentic.fleet.nodes", "invalid fleet nodes output", s.handleFleetNodes(ctx, platformOptions(
		core.Option{Key: "status", Value: input.Status},
		core.Option{Key: "platform", Value: input.Platform},
	)))
}

func (s *PrepSubsystem) fleetTaskAssignTool(ctx context.Context, input FleetTaskAssignInput) core.Result {
	options := platformOptions(
		core.Option{Key: "agent_id", Value: input.AgentID},
		core.Option{Key: "repo", Value: input.Repo},
		core.Option{Key: "branch", Value: input.Branch},
		core.Option{Key: "task", Value: input.Task},
		core.Option{Key: "template", Value: input.Template},
		core.Option{Key: "agent_model", Value: input.AgentModel},
	)
	return typedResultValue[FleetTask]("agentic.fleet.task.assign", "invalid fleet task output", s.handleFleetAssignTask(ctx, options))
}

func (s *PrepSubsystem) fleetTaskCompleteTool(ctx context.Context, input FleetTaskCompleteInput) core.Result {
	return typedResultValue[FleetTask]("agentic.fleet.task.complete", "invalid fleet task output", s.handleFleetCompleteTask(ctx, platformOptions(
		core.Option{Key: "agent_id", Value: input.AgentID},
		core.Option{Key: "task_id", Value: input.TaskID},
		core.Option{Key: "result", Value: input.Result},
		core.Option{Key: "findings", Value: input.Findings},
		core.Option{Key: "changes", Value: input.Changes},
		core.Option{Key: "report", Value: input.Report},
	)))
}

func (s *PrepSubsystem) fleetTaskNextTool(ctx context.Context, input FleetTaskNextInput) core.Result {
	return typedResultValue[*FleetTask]("agentic.fleet.task.next", "invalid fleet next-task output", s.handleFleetNextTask(ctx, platformOptions(
		core.Option{Key: "agent_id", Value: input.AgentID},
		core.Option{Key: "capabilities", Value: input.Capabilities},
	)))
}

func (s *PrepSubsystem) fleetStatsTool(ctx context.Context, _ struct{}) core.Result {
	return typedResultValue[FleetStats]("agentic.fleet.stats", "invalid fleet stats output", s.handleFleetStats(ctx, core.NewOptions()))
}

func (s *PrepSubsystem) fleetEventsTool(ctx context.Context, input FleetEventsInput) core.Result {
	return typedResultValue[FleetEventOutput]("agentic.fleet.events", "invalid fleet event output", s.handleFleetEvents(ctx, platformOptions(
		core.Option{Key: "agent_id", Value: input.AgentID},
		core.Option{Key: "capabilities", Value: input.Capabilities},
	)))
}

func (s *PrepSubsystem) creditsAwardTool(ctx context.Context, input CreditsAwardInput) core.Result {
	return typedResultValue[CreditEntry]("agentic.credits.award", "invalid credit award output", s.handleCreditsAward(ctx, platformOptions(
		core.Option{Key: "agent_id", Value: input.AgentID},
		core.Option{Key: "task_type", Value: input.TaskType},
		core.Option{Key: "amount", Value: input.Amount},
		core.Option{Key: "fleet_node_id", Value: input.FleetNodeID},
		core.Option{Key: "description", Value: input.Description},
	)))
}

func (s *PrepSubsystem) creditsBalanceTool(ctx context.Context, input CreditsBalanceInput) core.Result {
	return typedResultValue[CreditBalance]("agentic.credits.balance", "invalid credit balance output", s.handleCreditsBalance(ctx, platformOptions(core.Option{Key: "agent_id", Value: input.AgentID})))
}

func (s *PrepSubsystem) creditsHistoryTool(ctx context.Context, input CreditsHistoryInput) core.Result {
	return typedResultValue[CreditsHistoryOutput]("agentic.credits.history", "invalid credit history output", s.handleCreditsHistory(ctx, platformOptions(
		core.Option{Key: "agent_id", Value: input.AgentID},
		core.Option{Key: "limit", Value: input.Limit},
	)))
}

func (s *PrepSubsystem) subscriptionDetectTool(ctx context.Context, input SubscriptionDetectInput) core.Result {
	return typedResultValue[SubscriptionCapabilities]("agentic.subscription.detect", "invalid capability output", s.handleSubscriptionDetect(ctx, platformOptions(core.Option{Key: "api_keys", Value: input.APIKeys})))
}

func (s *PrepSubsystem) subscriptionBudgetTool(ctx context.Context, input SubscriptionBudgetInput) core.Result {
	return typedResultValue[map[string]any]("agentic.subscription.budget", "invalid budget output", s.handleSubscriptionBudget(ctx, platformOptions(core.Option{Key: "agent_id", Value: input.AgentID})))
}

func (s *PrepSubsystem) subscriptionBudgetUpdateTool(ctx context.Context, input SubscriptionBudgetUpdateInput) core.Result {
	return typedResultValue[map[string]any]("agentic.subscription.budget.update", "invalid updated budget output", s.handleSubscriptionBudgetUpdate(ctx, platformOptions(
		core.Option{Key: "agent_id", Value: input.AgentID},
		core.Option{Key: "limits", Value: input.Limits},
	)))
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
