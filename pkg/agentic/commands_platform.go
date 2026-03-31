// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go/core"
)

func (s *PrepSubsystem) registerPlatformCommands() {
	c := s.Core()
	c.Command("sync/push", core.Command{Description: "Push completed dispatch state to the platform API", Action: s.cmdSyncPush})
	c.Command("sync/pull", core.Command{Description: "Pull shared fleet context from the platform API", Action: s.cmdSyncPull})
	c.Command("sync/status", core.Command{Description: "Show platform sync status for the current or named agent", Action: s.cmdSyncStatus})
	c.Command("auth/provision", core.Command{Description: "Provision a platform API key for an authenticated agent user", Action: s.cmdAuthProvision})
	c.Command("auth/revoke", core.Command{Description: "Revoke a platform API key", Action: s.cmdAuthRevoke})

	c.Command("fleet/register", core.Command{Description: "Register a fleet node with the platform API", Action: s.cmdFleetRegister})
	c.Command("fleet/heartbeat", core.Command{Description: "Send a heartbeat for a registered fleet node", Action: s.cmdFleetHeartbeat})
	c.Command("fleet/deregister", core.Command{Description: "Deregister a fleet node from the platform API", Action: s.cmdFleetDeregister})
	c.Command("fleet/nodes", core.Command{Description: "List registered fleet nodes", Action: s.cmdFleetNodes})
	c.Command("fleet/task/assign", core.Command{Description: "Assign a task to a fleet node", Action: s.cmdFleetTaskAssign})
	c.Command("fleet/task/complete", core.Command{Description: "Complete a fleet task and report findings", Action: s.cmdFleetTaskComplete})
	c.Command("fleet/task/next", core.Command{Description: "Ask the platform for the next fleet task", Action: s.cmdFleetTaskNext})
	c.Command("fleet/stats", core.Command{Description: "Show fleet activity statistics", Action: s.cmdFleetStats})
	c.Command("fleet/events", core.Command{Description: "Read the next fleet event from the platform SSE stream", Action: s.cmdFleetEvents})

	c.Command("credits/award", core.Command{Description: "Award credits to a fleet node", Action: s.cmdCreditsAward})
	c.Command("credits/balance", core.Command{Description: "Show credit balance for a fleet node", Action: s.cmdCreditsBalance})
	c.Command("credits/history", core.Command{Description: "Show credit history for a fleet node", Action: s.cmdCreditsHistory})

	c.Command("subscription/detect", core.Command{Description: "Detect available provider capabilities", Action: s.cmdSubscriptionDetect})
	c.Command("subscription/budget", core.Command{Description: "Show compute budget for a fleet node", Action: s.cmdSubscriptionBudget})
	c.Command("subscription/budget/update", core.Command{Description: "Update compute budget for a fleet node", Action: s.cmdSubscriptionUpdateBudget})
	c.Command("subscription/update-budget", core.Command{Description: "Update compute budget for a fleet node", Action: s.cmdSubscriptionUpdateBudget})
}

func (s *PrepSubsystem) cmdAuthProvision(options core.Options) core.Result {
	if optionStringValue(options, "oauth_user_id", "oauth-user-id", "user_id", "user-id", "_arg") == "" {
		core.Print(nil, "usage: core-agent auth provision <oauth-user-id> [--name=codex] [--permissions=plans:read,plans:write] [--rate-limit=60] [--expires-at=2026-04-01T00:00:00Z]")
		return core.Result{Value: core.E("agentic.cmdAuthProvision", "oauth_user_id is required", nil), OK: false}
	}

	result := s.handleAuthProvision(s.commandContext(), options)
	if !result.OK {
		err := commandResultError("agentic.cmdAuthProvision", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(AuthProvisionOutput)
	if !ok {
		err := core.E("agentic.cmdAuthProvision", "invalid auth provision output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "key id:      %d", output.Key.ID)
	core.Print(nil, "name:        %s", output.Key.Name)
	core.Print(nil, "prefix:      %s", output.Key.Prefix)
	if output.Key.Key != "" {
		core.Print(nil, "key:         %s", output.Key.Key)
	}
	if len(output.Key.Permissions) > 0 {
		core.Print(nil, "permissions: %s", core.Join(",", output.Key.Permissions...))
	}
	if output.Key.ExpiresAt != "" {
		core.Print(nil, "expires:     %s", output.Key.ExpiresAt)
	}
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdAuthRevoke(options core.Options) core.Result {
	if optionStringValue(options, "key_id", "key-id", "_arg") == "" {
		core.Print(nil, "usage: core-agent auth revoke <key-id>")
		return core.Result{Value: core.E("agentic.cmdAuthRevoke", "key_id is required", nil), OK: false}
	}

	result := s.handleAuthRevoke(s.commandContext(), options)
	if !result.OK {
		err := commandResultError("agentic.cmdAuthRevoke", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(AuthRevokeOutput)
	if !ok {
		err := core.E("agentic.cmdAuthRevoke", "invalid auth revoke output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "revoked: %s", output.KeyID)
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdSyncPush(options core.Options) core.Result {
	result := s.handleSyncPush(s.commandContext(), options)
	if !result.OK {
		err := commandResultError("agentic.cmdSyncPush", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(SyncPushOutput)
	if !ok {
		err := core.E("agentic.cmdSyncPush", "invalid sync push output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "synced: %d", output.Count)
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdSyncPull(options core.Options) core.Result {
	result := s.handleSyncPull(s.commandContext(), options)
	if !result.OK {
		err := commandResultError("agentic.cmdSyncPull", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(SyncPullOutput)
	if !ok {
		err := core.E("agentic.cmdSyncPull", "invalid sync pull output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "context items: %d", output.Count)
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdSyncStatus(options core.Options) core.Result {
	result := s.handleSyncStatus(s.commandContext(), options)
	if !result.OK {
		err := commandResultError("agentic.cmdSyncStatus", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(SyncStatusOutput)
	if !ok {
		err := core.E("agentic.cmdSyncStatus", "invalid sync status output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "agent:         %s", output.AgentID)
	core.Print(nil, "status:        %s", output.Status)
	core.Print(nil, "queued:        %d", output.Queued)
	core.Print(nil, "context items: %d", output.ContextCount)
	if output.LastPushAt != "" {
		core.Print(nil, "last push:     %s", output.LastPushAt)
	}
	if output.LastPullAt != "" {
		core.Print(nil, "last pull:     %s", output.LastPullAt)
	}
	if output.RemoteError != "" {
		core.Print(nil, "remote error:  %s", output.RemoteError)
	}

	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdFleetRegister(options core.Options) core.Result {
	if optionStringValue(options, "agent_id", "agent-id", "_arg") == "" {
		core.Print(nil, "usage: core-agent fleet register <agent-id> --platform=linux [--models=codex,gpt-5.4] [--capabilities=go,review]")
		return core.Result{Value: core.E("agentic.cmdFleetRegister", "agent_id is required", nil), OK: false}
	}

	result := s.handleFleetRegister(s.commandContext(), normalisePlatformCommandOptions(options))
	if !result.OK {
		err := commandResultError("agentic.cmdFleetRegister", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	node, ok := result.Value.(FleetNode)
	if !ok {
		err := core.E("agentic.cmdFleetRegister", "invalid fleet register output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "registered: %s", node.AgentID)
	core.Print(nil, "status:     %s", node.Status)
	core.Print(nil, "platform:   %s", node.Platform)
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdFleetHeartbeat(options core.Options) core.Result {
	if optionStringValue(options, "agent_id", "agent-id", "_arg") == "" || optionStringValue(options, "status") == "" {
		core.Print(nil, "usage: core-agent fleet heartbeat <agent-id> --status=online [--compute-budget='{\"max_daily_hours\":2}']")
		return core.Result{Value: core.E("agentic.cmdFleetHeartbeat", "agent_id and status are required", nil), OK: false}
	}

	result := s.handleFleetHeartbeat(s.commandContext(), normalisePlatformCommandOptions(options))
	if !result.OK {
		err := commandResultError("agentic.cmdFleetHeartbeat", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	node, ok := result.Value.(FleetNode)
	if !ok {
		err := core.E("agentic.cmdFleetHeartbeat", "invalid fleet heartbeat output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "heartbeat: %s", node.AgentID)
	core.Print(nil, "status:    %s", node.Status)
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdFleetDeregister(options core.Options) core.Result {
	if optionStringValue(options, "agent_id", "agent-id", "_arg") == "" {
		core.Print(nil, "usage: core-agent fleet deregister <agent-id>")
		return core.Result{Value: core.E("agentic.cmdFleetDeregister", "agent_id is required", nil), OK: false}
	}

	result := s.handleFleetDeregister(s.commandContext(), normalisePlatformCommandOptions(options))
	if !result.OK {
		err := commandResultError("agentic.cmdFleetDeregister", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	data, ok := result.Value.(map[string]any)
	if !ok {
		err := core.E("agentic.cmdFleetDeregister", "invalid fleet deregister output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "deregistered: %s", stringValue(data["agent_id"]))
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdFleetNodes(options core.Options) core.Result {
	result := s.handleFleetNodes(s.commandContext(), options)
	if !result.OK {
		err := commandResultError("agentic.cmdFleetNodes", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(FleetNodesOutput)
	if !ok {
		err := core.E("agentic.cmdFleetNodes", "invalid fleet nodes output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	if len(output.Nodes) == 0 {
		core.Print(nil, "no fleet nodes")
		return core.Result{OK: true}
	}

	for _, node := range output.Nodes {
		core.Print(nil, "  %-10s %-8s %-10s %s", node.Status, node.Platform, node.AgentID, core.Join(",", node.Models...))
	}
	core.Print(nil, "")
	core.Print(nil, "total: %d", output.Total)
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdFleetTaskAssign(options core.Options) core.Result {
	agentID := optionStringValue(options, "agent_id", "agent-id", "_arg")
	if agentID == "" || optionStringValue(options, "repo") == "" || optionStringValue(options, "task") == "" {
		core.Print(nil, "usage: core-agent fleet task assign <agent-id> --repo=core/go-io --task=\"...\" [--branch=dev] [--template=coding] [--agent-model=codex:gpt-5.4]")
		return core.Result{Value: core.E("agentic.cmdFleetTaskAssign", "agent_id, repo, and task are required", nil), OK: false}
	}

	result := s.handleFleetAssignTask(s.commandContext(), normalisePlatformCommandOptions(options))
	if !result.OK {
		err := commandResultError("agentic.cmdFleetTaskAssign", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	task, ok := result.Value.(FleetTask)
	if !ok {
		err := core.E("agentic.cmdFleetTaskAssign", "invalid fleet task output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	printFleetTask(task)
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdFleetTaskComplete(options core.Options) core.Result {
	agentID := optionStringValue(options, "agent_id", "agent-id", "_arg")
	taskID := optionIntValue(options, "task_id", "task-id")
	if agentID == "" || taskID == 0 {
		core.Print(nil, "usage: core-agent fleet task complete <agent-id> --task-id=N [--result='{\"status\":\"completed\"}'] [--findings='[{\"file\":\"x.go\"}]']")
		return core.Result{Value: core.E("agentic.cmdFleetTaskComplete", "agent_id and task_id are required", nil), OK: false}
	}

	result := s.handleFleetCompleteTask(s.commandContext(), normalisePlatformCommandOptions(options))
	if !result.OK {
		err := commandResultError("agentic.cmdFleetTaskComplete", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	task, ok := result.Value.(FleetTask)
	if !ok {
		err := core.E("agentic.cmdFleetTaskComplete", "invalid fleet task output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	printFleetTask(task)
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdFleetTaskNext(options core.Options) core.Result {
	if optionStringValue(options, "agent_id", "agent-id", "_arg") == "" {
		core.Print(nil, "usage: core-agent fleet task next <agent-id> [--capabilities=go,review]")
		return core.Result{Value: core.E("agentic.cmdFleetTaskNext", "agent_id is required", nil), OK: false}
	}

	result := s.handleFleetNextTask(s.commandContext(), normalisePlatformCommandOptions(options))
	if !result.OK {
		err := commandResultError("agentic.cmdFleetTaskNext", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	task, ok := result.Value.(*FleetTask)
	if !ok {
		err := core.E("agentic.cmdFleetTaskNext", "invalid fleet next-task output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	if task == nil {
		core.Print(nil, "no task available")
		return core.Result{OK: true}
	}

	printFleetTask(*task)
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdFleetStats(options core.Options) core.Result {
	result := s.handleFleetStats(s.commandContext(), options)
	if !result.OK {
		err := commandResultError("agentic.cmdFleetStats", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	stats, ok := result.Value.(FleetStats)
	if !ok {
		err := core.E("agentic.cmdFleetStats", "invalid fleet stats output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "nodes online:   %d", stats.NodesOnline)
	core.Print(nil, "tasks today:    %d", stats.TasksToday)
	core.Print(nil, "tasks week:     %d", stats.TasksWeek)
	core.Print(nil, "repos touched:  %d", stats.ReposTouched)
	core.Print(nil, "findings total: %d", stats.FindingsTotal)
	core.Print(nil, "compute hours:  %d", stats.ComputeHours)
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdFleetEvents(options core.Options) core.Result {
	result := s.handleFleetEvents(s.commandContext(), normalisePlatformCommandOptions(options))
	if !result.OK {
		err := commandResultError("agentic.cmdFleetEvents", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(FleetEventOutput)
	if !ok {
		err := core.E("agentic.cmdFleetEvents", "invalid fleet event output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "event:       %s", output.Event.Event)
	if output.Event.Type != "" && output.Event.Type != output.Event.Event {
		core.Print(nil, "type:        %s", output.Event.Type)
	}
	if output.Event.AgentID != "" {
		core.Print(nil, "agent:       %s", output.Event.AgentID)
	}
	if output.Event.Repo != "" {
		core.Print(nil, "repo:        %s", output.Event.Repo)
	}
	if output.Event.Branch != "" {
		core.Print(nil, "branch:      %s", output.Event.Branch)
	}
	if output.Event.TaskID > 0 {
		core.Print(nil, "task id:     %d", output.Event.TaskID)
	}
	if output.Event.Status != "" {
		core.Print(nil, "status:      %s", output.Event.Status)
	}
	if len(output.Event.Payload) > 0 {
		core.Print(nil, "payload:     %s", core.JSONMarshalString(output.Event.Payload))
	}
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdCreditsAward(options core.Options) core.Result {
	agentID := optionStringValue(options, "agent_id", "agent-id", "_arg")
	taskType := optionStringValue(options, "task_type", "task-type")
	amount := optionIntValue(options, "amount")
	if agentID == "" || taskType == "" || amount == 0 {
		core.Print(nil, "usage: core-agent credits award <agent-id> --task-type=fleet-task --amount=2 [--description=\"...\"]")
		return core.Result{Value: core.E("agentic.cmdCreditsAward", "agent_id, task_type, and amount are required", nil), OK: false}
	}

	result := s.handleCreditsAward(s.commandContext(), normalisePlatformCommandOptions(options))
	if !result.OK {
		err := commandResultError("agentic.cmdCreditsAward", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	entry, ok := result.Value.(CreditEntry)
	if !ok {
		err := core.E("agentic.cmdCreditsAward", "invalid credit award output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "entry:         %d", entry.ID)
	core.Print(nil, "task type:     %s", entry.TaskType)
	core.Print(nil, "amount:        %d", entry.Amount)
	core.Print(nil, "balance after: %d", entry.BalanceAfter)
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdCreditsBalance(options core.Options) core.Result {
	if optionStringValue(options, "agent_id", "agent-id", "_arg") == "" {
		core.Print(nil, "usage: core-agent credits balance <agent-id>")
		return core.Result{Value: core.E("agentic.cmdCreditsBalance", "agent_id is required", nil), OK: false}
	}

	result := s.handleCreditsBalance(s.commandContext(), normalisePlatformCommandOptions(options))
	if !result.OK {
		err := commandResultError("agentic.cmdCreditsBalance", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	balance, ok := result.Value.(CreditBalance)
	if !ok {
		err := core.E("agentic.cmdCreditsBalance", "invalid credit balance output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "agent:   %s", balance.AgentID)
	core.Print(nil, "balance: %d", balance.Balance)
	core.Print(nil, "entries: %d", balance.Entries)
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdCreditsHistory(options core.Options) core.Result {
	if optionStringValue(options, "agent_id", "agent-id", "_arg") == "" {
		core.Print(nil, "usage: core-agent credits history <agent-id> [--limit=50]")
		return core.Result{Value: core.E("agentic.cmdCreditsHistory", "agent_id is required", nil), OK: false}
	}

	result := s.handleCreditsHistory(s.commandContext(), normalisePlatformCommandOptions(options))
	if !result.OK {
		err := commandResultError("agentic.cmdCreditsHistory", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(CreditsHistoryOutput)
	if !ok {
		err := core.E("agentic.cmdCreditsHistory", "invalid credit history output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	if len(output.Entries) == 0 {
		core.Print(nil, "no credit entries")
		return core.Result{OK: true}
	}

	for _, entry := range output.Entries {
		core.Print(nil, "  #%-4d %-12s %+d  balance=%d", entry.ID, entry.TaskType, entry.Amount, entry.BalanceAfter)
	}
	core.Print(nil, "")
	core.Print(nil, "total: %d", output.Total)
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdSubscriptionDetect(options core.Options) core.Result {
	result := s.handleSubscriptionDetect(s.commandContext(), options)
	if !result.OK {
		err := commandResultError("agentic.cmdSubscriptionDetect", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	capabilities, ok := result.Value.(SubscriptionCapabilities)
	if !ok {
		err := core.E("agentic.cmdSubscriptionDetect", "invalid capability output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "available: %s", core.Join(",", capabilities.Available...))
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdSubscriptionBudget(options core.Options) core.Result {
	if optionStringValue(options, "agent_id", "agent-id", "_arg") == "" {
		core.Print(nil, "usage: core-agent subscription budget <agent-id>")
		return core.Result{Value: core.E("agentic.cmdSubscriptionBudget", "agent_id is required", nil), OK: false}
	}

	result := s.handleSubscriptionBudget(s.commandContext(), normalisePlatformCommandOptions(options))
	if !result.OK {
		err := commandResultError("agentic.cmdSubscriptionBudget", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	budget, ok := result.Value.(map[string]any)
	if !ok {
		err := core.E("agentic.cmdSubscriptionBudget", "invalid budget output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "%s", core.JSONMarshalString(budget))
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdSubscriptionUpdateBudget(options core.Options) core.Result {
	if optionStringValue(options, "agent_id", "agent-id", "_arg") == "" || len(optionAnyMapValue(options, "limits")) == 0 {
		core.Print(nil, "usage: core-agent subscription budget update <agent-id> --limits='{\"max_daily_hours\":2}'")
		return core.Result{Value: core.E("agentic.cmdSubscriptionUpdateBudget", "agent_id and limits are required", nil), OK: false}
	}

	result := s.handleSubscriptionBudgetUpdate(s.commandContext(), normalisePlatformCommandOptions(options))
	if !result.OK {
		err := commandResultError("agentic.cmdSubscriptionUpdateBudget", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	budget, ok := result.Value.(map[string]any)
	if !ok {
		err := core.E("agentic.cmdSubscriptionUpdateBudget", "invalid updated budget output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "%s", core.JSONMarshalString(budget))
	return core.Result{OK: true}
}

func normalisePlatformCommandOptions(options core.Options) core.Options {
	normalised := core.NewOptions(options.Items()...)
	if normalised.String("agent_id") == "" {
		if agentID := optionStringValue(options, "_arg", "agent-id"); agentID != "" {
			normalised.Set("agent_id", agentID)
		}
	}
	return normalised
}

func commandResultError(action string, result core.Result) error {
	if err, ok := result.Value.(error); ok && err != nil {
		return err
	}

	message := stringValue(result.Value)
	if message != "" {
		return core.E(action, message, nil)
	}

	return core.E(action, "command failed", nil)
}

func printFleetTask(task FleetTask) {
	core.Print(nil, "task:        %d", task.ID)
	core.Print(nil, "repo:        %s", task.Repo)
	core.Print(nil, "status:      %s", task.Status)
	if task.Branch != "" {
		core.Print(nil, "branch:      %s", task.Branch)
	}
	if task.AgentModel != "" {
		core.Print(nil, "agent model: %s", task.AgentModel)
	}
	core.Print(nil, "summary:     %s", task.Task)
}
