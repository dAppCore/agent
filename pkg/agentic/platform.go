// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"time"

	core "dappco.re/go/core"
)

// node := agentic.FleetNode{AgentID: "charon", Platform: "linux", Status: "online"}
type FleetNode struct {
	ID              int            `json:"id"`
	WorkspaceID     int            `json:"workspace_id,omitempty"`
	AgentID         string         `json:"agent_id"`
	Platform        string         `json:"platform"`
	Models          []string       `json:"models,omitempty"`
	Capabilities    []string       `json:"capabilities,omitempty"`
	Status          string         `json:"status"`
	ComputeBudget   map[string]any `json:"compute_budget,omitempty"`
	CurrentTaskID   int            `json:"current_task_id,omitempty"`
	LastHeartbeatAt string         `json:"last_heartbeat_at,omitempty"`
	RegisteredAt    string         `json:"registered_at,omitempty"`
}

// task := agentic.FleetTask{ID: 7, Repo: "go-io", Task: "Fix tests", Status: "assigned"}
type FleetTask struct {
	ID          int              `json:"id"`
	WorkspaceID int              `json:"workspace_id,omitempty"`
	FleetNodeID int              `json:"fleet_node_id,omitempty"`
	Repo        string           `json:"repo"`
	Branch      string           `json:"branch,omitempty"`
	Task        string           `json:"task"`
	Template    string           `json:"template,omitempty"`
	AgentModel  string           `json:"agent_model,omitempty"`
	Status      string           `json:"status"`
	Result      map[string]any   `json:"result,omitempty"`
	Findings    []map[string]any `json:"findings,omitempty"`
	Changes     map[string]any   `json:"changes,omitempty"`
	Report      map[string]any   `json:"report,omitempty"`
	StartedAt   string           `json:"started_at,omitempty"`
	CompletedAt string           `json:"completed_at,omitempty"`
}

// out := agentic.FleetNodesOutput{Total: 2, Nodes: []agentic.FleetNode{{AgentID: "charon"}}}
type FleetNodesOutput struct {
	Total int         `json:"total"`
	Nodes []FleetNode `json:"nodes"`
}

// stats := agentic.FleetStats{NodesOnline: 2, TasksToday: 5}
type FleetStats struct {
	NodesOnline   int `json:"nodes_online"`
	TasksToday    int `json:"tasks_today"`
	TasksWeek     int `json:"tasks_week"`
	ReposTouched  int `json:"repos_touched"`
	FindingsTotal int `json:"findings_total"`
	ComputeHours  int `json:"compute_hours"`
}

// event := agentic.FleetEvent{Type: "task.assigned", AgentID: "charon", Repo: "core/go-io"}
type FleetEvent struct {
	Type       string         `json:"type,omitempty"`
	Event      string         `json:"event,omitempty"`
	AgentID    string         `json:"agent_id,omitempty"`
	TaskID     int            `json:"task_id,omitempty"`
	Repo       string         `json:"repo,omitempty"`
	Branch     string         `json:"branch,omitempty"`
	Status     string         `json:"status,omitempty"`
	ReceivedAt string         `json:"received_at,omitempty"`
	Payload    map[string]any `json:"payload,omitempty"`
}

// out := agentic.FleetEventOutput{Success: true, Event: agentic.FleetEvent{Type: "task.assigned"}}
type FleetEventOutput struct {
	Success bool       `json:"success"`
	Event   FleetEvent `json:"event"`
	Raw     string     `json:"raw,omitempty"`
}

// status := agentic.SyncStatusOutput{AgentID: "charon", Status: "online"}
type SyncStatusOutput struct {
	AgentID      string `json:"agent_id"`
	Status       string `json:"status"`
	LastPushAt   string `json:"last_push_at,omitempty"`
	LastPullAt   string `json:"last_pull_at,omitempty"`
	Queued       int    `json:"queued"`
	ContextCount int    `json:"context_count"`
	RemoteError  string `json:"remote_error,omitempty"`
}

// balance := agentic.CreditBalance{AgentID: "charon", Balance: 12}
type CreditBalance struct {
	AgentID string `json:"agent_id"`
	Balance int    `json:"balance"`
	Entries int    `json:"entries"`
}

// entry := agentic.CreditEntry{ID: 4, TaskType: "fleet-task", Amount: 2}
type CreditEntry struct {
	ID           int    `json:"id"`
	WorkspaceID  int    `json:"workspace_id,omitempty"`
	FleetNodeID  int    `json:"fleet_node_id,omitempty"`
	TaskType     string `json:"task_type"`
	Amount       int    `json:"amount"`
	BalanceAfter int    `json:"balance_after"`
	Description  string `json:"description,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
}

// out := agentic.CreditsHistoryOutput{Total: 1, Entries: []agentic.CreditEntry{{ID: 1}}}
type CreditsHistoryOutput struct {
	Total   int           `json:"total"`
	Entries []CreditEntry `json:"entries"`
}

// caps := agentic.SubscriptionCapabilities{Available: []string{"claude", "openai"}}
type SubscriptionCapabilities struct {
	Providers map[string]bool `json:"providers,omitempty"`
	Available []string        `json:"available,omitempty"`
}

// result := c.Action("agentic.sync.status").Run(ctx, core.NewOptions())
func (s *PrepSubsystem) handleSyncStatus(ctx context.Context, options core.Options) core.Result {
	agentID := optionStringValue(options, "agent_id", "agent-id", "_arg")
	if agentID == "" {
		agentID = AgentName()
	}

	output := SyncStatusOutput{
		AgentID:      agentID,
		Status:       "offline",
		Queued:       len(readSyncQueue()),
		ContextCount: len(readSyncContext()),
	}
	localStatus := readSyncStatusState()
	if !localStatus.LastPushAt.IsZero() {
		output.LastPushAt = localStatus.LastPushAt.Format(time.RFC3339)
	}
	if !localStatus.LastPullAt.IsZero() {
		output.LastPullAt = localStatus.LastPullAt.Format(time.RFC3339)
	}

	if s.syncToken() == "" {
		return core.Result{Value: output, OK: true}
	}

	path := appendQueryParam("/v1/agent/status", "agent_id", agentID)
	result := s.platformPayload(ctx, "agentic.sync.status", "GET", path, nil)
	if !result.OK {
		err, _ := result.Value.(error)
		if err != nil {
			output.RemoteError = err.Error()
		}
		return core.Result{Value: output, OK: true}
	}

	data := payloadResourceMap(result.Value.(map[string]any), "status")
	if len(data) == 0 {
		return core.Result{Value: output, OK: true}
	}

	if remoteAgentID := stringValue(data["agent_id"]); remoteAgentID != "" {
		output.AgentID = remoteAgentID
	}
	output.Status = stringValue(data["status"])
	if lastPushAt := stringValue(data["last_push_at"]); lastPushAt != "" {
		output.LastPushAt = lastPushAt
	}
	if lastPullAt := stringValue(data["last_pull_at"]); lastPullAt != "" {
		output.LastPullAt = lastPullAt
	}
	if queued, ok := intValueOK(data["queued"]); ok {
		output.Queued = queued
	}
	if contextCount, ok := intValueOK(data["context_count"]); ok {
		output.ContextCount = contextCount
	}
	if output.Status == "" {
		output.Status = "online"
	}

	return core.Result{Value: output, OK: true}
}

// result := c.Action("agentic.fleet.register").Run(ctx, core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))
func (s *PrepSubsystem) handleFleetRegister(ctx context.Context, options core.Options) core.Result {
	agentID := optionStringValue(options, "agent_id", "agent-id", "_arg")
	if agentID == "" {
		return core.Result{Value: core.E("agentic.fleet.register", "agent_id is required", nil), OK: false}
	}

	platform := optionStringValue(options, "platform")
	if platform == "" {
		platform = "unknown"
	}

	body := map[string]any{
		"agent_id": agentID,
		"platform": platform,
	}
	if models := optionStringSliceValue(options, "models"); len(models) > 0 {
		body["models"] = models
	}
	if capabilities := optionStringSliceValue(options, "capabilities"); len(capabilities) > 0 {
		body["capabilities"] = capabilities
	}

	result := s.platformPayload(ctx, "agentic.fleet.register", "POST", "/v1/fleet/register", body)
	if !result.OK {
		return result
	}

	return core.Result{Value: parseFleetNode(payloadResourceMap(result.Value.(map[string]any), "node")), OK: true}
}

// result := c.Action("agentic.fleet.heartbeat").Run(ctx, core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))
func (s *PrepSubsystem) handleFleetHeartbeat(ctx context.Context, options core.Options) core.Result {
	agentID := optionStringValue(options, "agent_id", "agent-id", "_arg")
	status := optionStringValue(options, "status")
	if agentID == "" || status == "" {
		return core.Result{Value: core.E("agentic.fleet.heartbeat", "agent_id and status are required", nil), OK: false}
	}

	body := map[string]any{
		"agent_id": agentID,
		"status":   status,
	}
	if budget := optionAnyMapValue(options, "compute_budget", "compute-budget"); len(budget) > 0 {
		body["compute_budget"] = budget
	}

	result := s.platformPayload(ctx, "agentic.fleet.heartbeat", "POST", "/v1/fleet/heartbeat", body)
	if !result.OK {
		return result
	}

	return core.Result{Value: parseFleetNode(payloadResourceMap(result.Value.(map[string]any), "node")), OK: true}
}

// result := c.Action("agentic.fleet.deregister").Run(ctx, core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))
func (s *PrepSubsystem) handleFleetDeregister(ctx context.Context, options core.Options) core.Result {
	agentID := optionStringValue(options, "agent_id", "agent-id", "_arg")
	if agentID == "" {
		return core.Result{Value: core.E("agentic.fleet.deregister", "agent_id is required", nil), OK: false}
	}

	result := s.platformPayload(ctx, "agentic.fleet.deregister", "POST", "/v1/fleet/deregister", map[string]any{
		"agent_id": agentID,
	})
	if !result.OK {
		return result
	}

	return core.Result{Value: map[string]any{
		"agent_id":     agentID,
		"deregistered": true,
	}, OK: true}
}

// result := c.Action("agentic.fleet.nodes").Run(ctx, core.NewOptions())
func (s *PrepSubsystem) handleFleetNodes(ctx context.Context, options core.Options) core.Result {
	path := "/v1/fleet/nodes"
	path = appendQueryParam(path, "status", optionStringValue(options, "status"))
	path = appendQueryParam(path, "platform", optionStringValue(options, "platform"))

	result := s.platformPayload(ctx, "agentic.fleet.nodes", "GET", path, nil)
	if !result.OK {
		return result
	}

	return core.Result{Value: parseFleetNodesOutput(result.Value.(map[string]any)), OK: true}
}

// result := c.Action("agentic.fleet.task.assign").Run(ctx, core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))
func (s *PrepSubsystem) handleFleetAssignTask(ctx context.Context, options core.Options) core.Result {
	agentID := optionStringValue(options, "agent_id", "agent-id", "_arg")
	repo := optionStringValue(options, "repo")
	task := optionStringValue(options, "task")
	if agentID == "" || repo == "" || task == "" {
		return core.Result{Value: core.E("agentic.fleet.task.assign", "agent_id, repo, and task are required", nil), OK: false}
	}

	body := map[string]any{
		"agent_id": agentID,
		"repo":     repo,
		"task":     task,
	}
	if branch := optionStringValue(options, "branch"); branch != "" {
		body["branch"] = branch
	}
	if template := optionStringValue(options, "template"); template != "" {
		body["template"] = template
	}
	if agentModel := optionStringValue(options, "agent_model", "agent-model"); agentModel != "" {
		body["agent_model"] = agentModel
	}

	result := s.platformPayload(ctx, "agentic.fleet.task.assign", "POST", "/v1/fleet/task/assign", body)
	if !result.OK {
		return result
	}

	return core.Result{Value: parseFleetTask(payloadResourceMap(result.Value.(map[string]any), "task")), OK: true}
}

// result := c.Action("agentic.fleet.task.complete").Run(ctx, core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))
func (s *PrepSubsystem) handleFleetCompleteTask(ctx context.Context, options core.Options) core.Result {
	agentID := optionStringValue(options, "agent_id", "agent-id", "_arg")
	taskID := optionIntValue(options, "task_id", "task-id")
	if agentID == "" || taskID == 0 {
		return core.Result{Value: core.E("agentic.fleet.task.complete", "agent_id and task_id are required", nil), OK: false}
	}

	body := map[string]any{
		"agent_id": agentID,
		"task_id":  taskID,
	}
	if resultMap := optionAnyMapValue(options, "result"); len(resultMap) > 0 {
		body["result"] = resultMap
	}
	if findings := optionAnyMapSliceValue(options, "findings"); len(findings) > 0 {
		body["findings"] = findings
	}
	if changes := optionAnyMapValue(options, "changes"); len(changes) > 0 {
		body["changes"] = changes
	}
	if report := optionAnyMapValue(options, "report"); len(report) > 0 {
		body["report"] = report
	}

	result := s.platformPayload(ctx, "agentic.fleet.task.complete", "POST", "/v1/fleet/task/complete", body)
	if !result.OK {
		return result
	}

	return core.Result{Value: parseFleetTask(payloadResourceMap(result.Value.(map[string]any), "task")), OK: true}
}

// result := c.Action("agentic.fleet.task.next").Run(ctx, core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))
func (s *PrepSubsystem) handleFleetNextTask(ctx context.Context, options core.Options) core.Result {
	agentID := optionStringValue(options, "agent_id", "agent-id", "_arg")
	if agentID == "" {
		return core.Result{Value: core.E("agentic.fleet.task.next", "agent_id is required", nil), OK: false}
	}

	path := appendQueryParam("/v1/fleet/task/next", "agent_id", agentID)
	path = appendQuerySlice(path, "capabilities[]", optionStringSliceValue(options, "capabilities"))

	result := s.platformPayload(ctx, "agentic.fleet.task.next", "GET", path, nil)
	if !result.OK {
		return result
	}

	data := payloadResourceMap(result.Value.(map[string]any), "task")
	if len(data) == 0 {
		var task *FleetTask
		return core.Result{Value: task, OK: true}
	}

	task := parseFleetTask(data)
	return core.Result{Value: &task, OK: true}
}

// result := c.Action("agentic.fleet.stats").Run(ctx, core.NewOptions())
func (s *PrepSubsystem) handleFleetStats(ctx context.Context, options core.Options) core.Result {
	result := s.platformPayload(ctx, "agentic.fleet.stats", "GET", "/v1/fleet/stats", nil)
	if !result.OK {
		return result
	}

	return core.Result{Value: parseFleetStats(payloadResourceMap(result.Value.(map[string]any), "stats")), OK: true}
}

// result := c.Action("agentic.fleet.events").Run(ctx, core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))
func (s *PrepSubsystem) handleFleetEvents(ctx context.Context, options core.Options) core.Result {
	agentID := optionStringValue(options, "agent_id", "agent-id", "_arg")
	path := "/v1/fleet/events"
	path = appendQueryParam(path, "agent_id", agentID)

	result := s.platformEventPayload(ctx, "agentic.fleet.events", path)
	if result.OK {
		output, err := parseFleetEventOutput(result.Value.(map[string]any))
		if err == nil {
			return core.Result{Value: output, OK: true}
		}
	}

	fallbackResult := s.handleFleetNextTask(ctx, core.NewOptions(
		core.Option{Key: "agent_id", Value: agentID},
		core.Option{Key: "capabilities", Value: optionStringSliceValue(options, "capabilities")},
	))
	if !fallbackResult.OK {
		if result.OK {
			return result
		}
		return fallbackResult
	}

	task, ok := fallbackResult.Value.(*FleetTask)
	if !ok {
		return core.Result{Value: core.E("agentic.fleet.events", "invalid fleet task output", nil), OK: false}
	}
	if task == nil {
		if result.OK {
			return core.Result{Value: core.E("agentic.fleet.events", "no fleet event payload returned", nil), OK: false}
		}
		return core.Result{Value: core.E("agentic.fleet.events", "no fleet task available", nil), OK: false}
	}

	return core.Result{Value: fleetEventOutputFromTask(agentID, task), OK: true}
}

// result := c.Action("agentic.credits.award").Run(ctx, core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))
func (s *PrepSubsystem) handleCreditsAward(ctx context.Context, options core.Options) core.Result {
	agentID := optionStringValue(options, "agent_id", "agent-id", "_arg")
	taskType := optionStringValue(options, "task_type", "task-type")
	amount := optionIntValue(options, "amount")
	if agentID == "" || taskType == "" || amount == 0 {
		return core.Result{Value: core.E("agentic.credits.award", "agent_id, task_type, and amount are required", nil), OK: false}
	}

	body := map[string]any{
		"agent_id":  agentID,
		"task_type": taskType,
		"amount":    amount,
	}
	if fleetNodeID := optionIntValue(options, "fleet_node_id", "fleet-node-id"); fleetNodeID > 0 {
		body["fleet_node_id"] = fleetNodeID
	}
	if description := optionStringValue(options, "description"); description != "" {
		body["description"] = description
	}

	result := s.platformPayload(ctx, "agentic.credits.award", "POST", "/v1/credits/award", body)
	if !result.OK {
		return result
	}

	return core.Result{Value: parseCreditEntry(payloadResourceMap(result.Value.(map[string]any), "entry")), OK: true}
}

// result := c.Action("agentic.credits.balance").Run(ctx, core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))
func (s *PrepSubsystem) handleCreditsBalance(ctx context.Context, options core.Options) core.Result {
	agentID := optionStringValue(options, "agent_id", "agent-id", "_arg")
	if agentID == "" {
		return core.Result{Value: core.E("agentic.credits.balance", "agent_id is required", nil), OK: false}
	}

	path := core.Concat("/v1/credits/balance/", agentID)
	result := s.platformPayload(ctx, "agentic.credits.balance", "GET", path, nil)
	if !result.OK {
		return result
	}

	return core.Result{Value: parseCreditBalance(payloadResourceMap(result.Value.(map[string]any), "balance")), OK: true}
}

// result := c.Action("agentic.credits.history").Run(ctx, core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))
func (s *PrepSubsystem) handleCreditsHistory(ctx context.Context, options core.Options) core.Result {
	agentID := optionStringValue(options, "agent_id", "agent-id", "_arg")
	if agentID == "" {
		return core.Result{Value: core.E("agentic.credits.history", "agent_id is required", nil), OK: false}
	}

	path := core.Concat("/v1/credits/history/", agentID)
	if limit := optionIntValue(options, "limit"); limit > 0 {
		path = appendQueryParam(path, "limit", core.Sprint(limit))
	}

	result := s.platformPayload(ctx, "agentic.credits.history", "GET", path, nil)
	if !result.OK {
		return result
	}

	return core.Result{Value: parseCreditsHistoryOutput(result.Value.(map[string]any)), OK: true}
}

// result := c.Action("agentic.subscription.detect").Run(ctx, core.NewOptions())
func (s *PrepSubsystem) handleSubscriptionDetect(ctx context.Context, options core.Options) core.Result {
	body := map[string]any{}
	if apiKeys := optionStringMapValue(options, "api_keys", "api-keys"); len(apiKeys) > 0 {
		body["api_keys"] = apiKeys
	}

	result := s.platformPayload(ctx, "agentic.subscription.detect", "POST", "/v1/subscription/detect", body)
	if !result.OK {
		return result
	}

	return core.Result{Value: parseSubscriptionCapabilities(payloadResourceMap(result.Value.(map[string]any), "capabilities", "subscription")), OK: true}
}

// result := c.Action("agentic.subscription.budget").Run(ctx, core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))
func (s *PrepSubsystem) handleSubscriptionBudget(ctx context.Context, options core.Options) core.Result {
	agentID := optionStringValue(options, "agent_id", "agent-id", "_arg")
	if agentID == "" {
		return core.Result{Value: core.E("agentic.subscription.budget", "agent_id is required", nil), OK: false}
	}

	path := core.Concat("/v1/subscription/budget/", agentID)
	result := s.platformPayload(ctx, "agentic.subscription.budget", "GET", path, nil)
	if !result.OK {
		return result
	}

	return core.Result{Value: payloadResourceMap(result.Value.(map[string]any), "budget"), OK: true}
}

// result := c.Action("agentic.subscription.budget.update").Run(ctx, core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))
func (s *PrepSubsystem) handleSubscriptionBudgetUpdate(ctx context.Context, options core.Options) core.Result {
	agentID := optionStringValue(options, "agent_id", "agent-id", "_arg")
	limits := optionAnyMapValue(options, "limits")
	if agentID == "" || len(limits) == 0 {
		return core.Result{Value: core.E("agentic.subscription.budget.update", "agent_id and limits are required", nil), OK: false}
	}

	path := core.Concat("/v1/subscription/budget/", agentID)
	result := s.platformPayload(ctx, "agentic.subscription.budget.update", "PUT", path, map[string]any{
		"limits": limits,
	})
	if !result.OK {
		return result
	}

	return core.Result{Value: payloadResourceMap(result.Value.(map[string]any), "budget"), OK: true}
}

func (s *PrepSubsystem) platformPayload(ctx context.Context, action, method, path string, body any) core.Result {
	token := s.syncToken()
	if token == "" {
		return core.Result{Value: core.E(action, "no platform API key configured", nil), OK: false}
	}

	bodyString := ""
	if body != nil {
		bodyString = core.JSONMarshalString(body)
	}

	requestResult := HTTPDo(ctx, method, core.Concat(s.syncAPIURL(), path), bodyString, token, "Bearer")
	if !requestResult.OK {
		return core.Result{Value: platformResultError(action, requestResult), OK: false}
	}

	var payload map[string]any
	parseResult := core.JSONUnmarshalString(requestResult.Value.(string), &payload)
	if !parseResult.OK {
		err, _ := parseResult.Value.(error)
		return core.Result{Value: core.E(action, "failed to parse platform response", err), OK: false}
	}

	return core.Result{Value: payload, OK: true}
}

func (s *PrepSubsystem) platformEventPayload(ctx context.Context, action, path string) core.Result {
	token := s.syncToken()
	if token == "" {
		return core.Result{Value: core.E(action, "no platform API key configured", nil), OK: false}
	}

	request, err := http.NewRequestWithContext(ctx, "GET", core.Concat(s.syncAPIURL(), path), nil)
	if err != nil {
		return core.Result{Value: core.E(action, "create request", err), OK: false}
	}
	request.Header.Set("Accept", "text/event-stream, application/json")
	request.Header.Set("Authorization", core.Concat("Bearer ", token))

	response, err := defaultClient.Do(request)
	if err != nil {
		return core.Result{Value: core.E(action, "request failed", err), OK: false}
	}
	defer response.Body.Close()

	if response.StatusCode >= 400 {
		readResult := core.ReadAll(response.Body)
		if !readResult.OK {
			return core.Result{Value: core.E(action, core.Sprintf("HTTP %d", response.StatusCode), nil), OK: false}
		}
		body := core.Trim(readResult.Value.(string))
		if body == "" {
			return core.Result{Value: core.E(action, core.Sprintf("HTTP %d", response.StatusCode), nil), OK: false}
		}
		return core.Result{Value: platformResultError(action, core.Result{Value: body, OK: false}), OK: false}
	}

	eventBody, readErr := readFleetEventBody(response.Body)
	if readErr != nil {
		return core.Result{Value: core.E(action, "failed to read event stream", readErr), OK: false}
	}

	payload := s.eventPayloadValue(eventBody)
	if len(payload) == 0 {
		return core.Result{Value: core.E(action, "no fleet event payload returned", nil), OK: false}
	}

	return core.Result{Value: payload, OK: true}
}

func platformResultError(action string, result core.Result) error {
	if err, ok := result.Value.(error); ok && err != nil {
		return core.E(action, "platform request failed", err)
	}

	body := core.Trim(stringValue(result.Value))
	if body == "" {
		return core.E(action, "platform request failed", nil)
	}

	var payload map[string]any
	if parseResult := core.JSONUnmarshalString(body, &payload); parseResult.OK {
		if message := stringValue(payload["error"]); message != "" {
			return core.E(action, message, nil)
		}
	}

	return core.E(action, body, nil)
}

func payloadDataMap(payload map[string]any) map[string]any {
	return anyMapValue(payload["data"])
}

func payloadDataSlice(payload map[string]any, keys ...string) []map[string]any {
	if values := anyMapSliceValue(payload["data"]); len(values) > 0 {
		return values
	}

	if data := payloadDataMap(payload); len(data) > 0 {
		for _, key := range keys {
			if values := anyMapSliceValue(data[key]); len(values) > 0 {
				return values
			}
		}
	}

	for _, key := range keys {
		if values := anyMapSliceValue(payload[key]); len(values) > 0 {
			return values
		}
	}

	return nil
}

func payloadResourceMap(payload map[string]any, keys ...string) map[string]any {
	if data := payloadDataMap(payload); len(data) > 0 {
		for _, key := range keys {
			if values := anyMapValue(data[key]); len(values) > 0 {
				return values
			}
		}
		return data
	}

	for _, key := range keys {
		if values := anyMapValue(payload[key]); len(values) > 0 {
			return values
		}
	}

	for key, value := range payload {
		switch key {
		case "data", "error", "code", "message":
			continue
		}
		if value != nil {
			return payload
		}
	}

	return nil
}

func mapIntValue(values map[string]any, keys ...string) int {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			return intValue(value)
		}
	}
	return 0
}

func intValueOK(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	case string:
		trimmed := core.Trim(typed)
		parsed := parseInt(trimmed)
		if parsed != 0 || trimmed == "0" {
			return parsed, true
		}
	}
	return 0, false
}

func parseFleetNode(values map[string]any) FleetNode {
	return FleetNode{
		ID:              intValue(values["id"]),
		WorkspaceID:     intValue(values["workspace_id"]),
		AgentID:         stringValue(values["agent_id"]),
		Platform:        stringValue(values["platform"]),
		Models:          listValue(values["models"]),
		Capabilities:    listValue(values["capabilities"]),
		Status:          stringValue(values["status"]),
		ComputeBudget:   anyMapValue(values["compute_budget"]),
		CurrentTaskID:   intValue(values["current_task_id"]),
		LastHeartbeatAt: stringValue(values["last_heartbeat_at"]),
		RegisteredAt:    stringValue(values["registered_at"]),
	}
}

func parseFleetTask(values map[string]any) FleetTask {
	return FleetTask{
		ID:          intValue(values["id"]),
		WorkspaceID: intValue(values["workspace_id"]),
		FleetNodeID: intValue(values["fleet_node_id"]),
		Repo:        stringValue(values["repo"]),
		Branch:      stringValue(values["branch"]),
		Task:        stringValue(values["task"]),
		Template:    stringValue(values["template"]),
		AgentModel:  stringValue(values["agent_model"]),
		Status:      stringValue(values["status"]),
		Result:      anyMapValue(values["result"]),
		Findings:    anyMapSliceValue(values["findings"]),
		Changes:     anyMapValue(values["changes"]),
		Report:      anyMapValue(values["report"]),
		StartedAt:   stringValue(values["started_at"]),
		CompletedAt: stringValue(values["completed_at"]),
	}
}

func parseFleetNodesOutput(payload map[string]any) FleetNodesOutput {
	nodesData := payloadDataSlice(payload, "nodes")
	nodes := make([]FleetNode, 0, len(nodesData))
	for _, values := range nodesData {
		nodes = append(nodes, parseFleetNode(values))
	}

	total := mapIntValue(payload, "total", "count")
	if total == 0 {
		total = mapIntValue(payloadDataMap(payload), "total", "count")
	}
	if total == 0 {
		total = len(nodes)
	}

	return FleetNodesOutput{
		Total: total,
		Nodes: nodes,
	}
}

func parseFleetStats(values map[string]any) FleetStats {
	return FleetStats{
		NodesOnline:   intValue(values["nodes_online"]),
		TasksToday:    intValue(values["tasks_today"]),
		TasksWeek:     intValue(values["tasks_week"]),
		ReposTouched:  intValue(values["repos_touched"]),
		FindingsTotal: intValue(values["findings_total"]),
		ComputeHours:  intValue(values["compute_hours"]),
	}
}

func parseFleetEventOutput(values map[string]any) (FleetEventOutput, error) {
	eventValues := payloadResourceMap(values, "event")
	if len(eventValues) == 0 {
		eventValues = values
	}

	event := parseFleetEvent(eventValues)
	if event.Event == "" && event.Type == "" && event.AgentID == "" && event.Repo == "" {
		return FleetEventOutput{}, core.E("parseFleetEventOutput", "fleet event payload is empty", nil)
	}

	return FleetEventOutput{
		Success: true,
		Event:   event,
		Raw:     core.Trim(stringValue(values["raw"])),
	}, nil
}

func (s *PrepSubsystem) eventPayloadValue(body string) map[string]any {
	trimmed := core.Trim(body)
	if trimmed == "" {
		return nil
	}

	if core.HasPrefix(trimmed, "data: ") {
		trimmed = core.Trim(core.TrimPrefix(trimmed, "data: "))
	}

	var payload map[string]any
	if parseResult := core.JSONUnmarshalString(trimmed, &payload); parseResult.OK {
		if payload != nil {
			payload["raw"] = trimmed
		}
		return payload
	}

	return map[string]any{
		"raw": trimmed,
	}
}

func readFleetEventBody(body io.ReadCloser) (string, error) {
	reader := bufio.NewReader(body)
	rawLines := make([]string, 0, 4)
	dataLines := make([]string, 0, 4)

	for {
		line, err := reader.ReadString('\n')
		if line != "" {
			trimmed := core.Trim(line)
			if trimmed != "" {
				rawLines = append(rawLines, trimmed)
				if core.HasPrefix(trimmed, "data:") {
					dataLines = append(dataLines, core.Trim(core.TrimPrefix(trimmed, "data:")))
				}
			} else if len(dataLines) > 0 {
				return core.Join("\n", dataLines...), nil
			}
		}

		if err == io.EOF {
			if len(dataLines) > 0 {
				return core.Join("\n", dataLines...), nil
			}
			return core.Join("\n", rawLines...), nil
		}
		if err != nil {
			return "", err
		}
	}
}

func parseFleetEvent(values map[string]any) FleetEvent {
	payload := map[string]any{}
	for key, value := range values {
		switch key {
		case "type", "event", "agent_id", "task_id", "repo", "branch", "status", "received_at":
			continue
		default:
			payload[key] = value
		}
	}
	if len(payload) == 0 {
		payload = nil
	}

	event := FleetEvent{
		Type:       stringValue(values["type"]),
		Event:      stringValue(values["event"]),
		AgentID:    stringValue(values["agent_id"]),
		TaskID:     intValue(values["task_id"]),
		Repo:       stringValue(values["repo"]),
		Branch:     stringValue(values["branch"]),
		Status:     stringValue(values["status"]),
		ReceivedAt: stringValue(values["received_at"]),
		Payload:    payload,
	}

	if event.Event == "" {
		event.Event = event.Type
	}
	if event.Type == "" {
		event.Type = event.Event
	}

	return event
}

func fleetEventOutputFromTask(agentID string, task *FleetTask) FleetEventOutput {
	payload := map[string]any{
		"task_id":       task.ID,
		"repo":          task.Repo,
		"branch":        task.Branch,
		"task":          task.Task,
		"template":      task.Template,
		"agent_model":   task.AgentModel,
		"status":        task.Status,
		"source":        "polling",
		"fleet_node_id": task.FleetNodeID,
	}

	return FleetEventOutput{
		Success: true,
		Event: FleetEvent{
			Type:    "task.assigned",
			Event:   "task.assigned",
			AgentID: agentID,
			TaskID:  task.ID,
			Repo:    task.Repo,
			Branch:  task.Branch,
			Status:  "assigned",
			Payload: payload,
		},
		Raw: "polling fallback",
	}
}

func parseCreditEntry(values map[string]any) CreditEntry {
	return CreditEntry{
		ID:           intValue(values["id"]),
		WorkspaceID:  intValue(values["workspace_id"]),
		FleetNodeID:  intValue(values["fleet_node_id"]),
		TaskType:     stringValue(values["task_type"]),
		Amount:       intValue(values["amount"]),
		BalanceAfter: intValue(values["balance_after"]),
		Description:  stringValue(values["description"]),
		CreatedAt:    stringValue(values["created_at"]),
	}
}

func parseCreditBalance(values map[string]any) CreditBalance {
	return CreditBalance{
		AgentID: stringValue(values["agent_id"]),
		Balance: intValue(values["balance"]),
		Entries: intValue(values["entries"]),
	}
}

func parseCreditsHistoryOutput(payload map[string]any) CreditsHistoryOutput {
	entriesData := payloadDataSlice(payload, "entries", "history")
	entries := make([]CreditEntry, 0, len(entriesData))
	for _, values := range entriesData {
		entries = append(entries, parseCreditEntry(values))
	}

	total := mapIntValue(payload, "total", "count")
	if total == 0 {
		total = mapIntValue(payloadDataMap(payload), "total", "count")
	}
	if total == 0 {
		total = len(entries)
	}

	return CreditsHistoryOutput{
		Total:   total,
		Entries: entries,
	}
}

func parseSubscriptionCapabilities(values map[string]any) SubscriptionCapabilities {
	capabilities := SubscriptionCapabilities{
		Providers: boolMapValue(values["providers"]),
		Available: listValue(values["available"]),
	}
	if len(capabilities.Available) == 0 && len(capabilities.Providers) > 0 {
		for name, enabled := range capabilities.Providers {
			if enabled {
				capabilities.Available = append(capabilities.Available, name)
			}
		}
		capabilities.Available = cleanStrings(capabilities.Available)
	}
	return capabilities
}

func appendQueryParam(path, key, value string) string {
	value = core.Trim(value)
	if value == "" {
		return path
	}

	separator := "?"
	if core.Contains(path, "?") {
		separator = "&"
	}

	return core.Concat(path, separator, key, "=", value)
}

func appendQuerySlice(path, key string, values []string) string {
	for _, value := range values {
		path = appendQueryParam(path, key, value)
	}
	return path
}

func optionAnyMapValue(options core.Options, keys ...string) map[string]any {
	for _, key := range keys {
		result := options.Get(key)
		if !result.OK {
			continue
		}
		values := anyMapValue(result.Value)
		if len(values) > 0 {
			return values
		}
	}
	return nil
}

func optionAnyMapSliceValue(options core.Options, keys ...string) []map[string]any {
	for _, key := range keys {
		result := options.Get(key)
		if !result.OK {
			continue
		}
		values := anyMapSliceValue(result.Value)
		if len(values) > 0 {
			return values
		}
	}
	return nil
}

func anyMapValue(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return typed
	case map[string]string:
		values := make(map[string]any, len(typed))
		for key, item := range typed {
			values[key] = item
		}
		return values
	case string:
		trimmed := core.Trim(typed)
		if trimmed == "" {
			return nil
		}
		if core.HasPrefix(trimmed, "{") {
			var values map[string]any
			if result := core.JSONUnmarshalString(trimmed, &values); result.OK {
				return values
			}
			var stringValues map[string]string
			if result := core.JSONUnmarshalString(trimmed, &stringValues); result.OK {
				return anyMapValue(stringValues)
			}
		}
		values := stringMapValue(trimmed)
		if len(values) > 0 {
			return anyMapValue(values)
		}
	}
	return nil
}

func anyMapSliceValue(value any) []map[string]any {
	switch typed := value.(type) {
	case []map[string]any:
		return typed
	case []any:
		values := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			if mapValue := anyMapValue(item); len(mapValue) > 0 {
				values = append(values, mapValue)
			}
		}
		return values
	case string:
		trimmed := core.Trim(typed)
		if trimmed == "" {
			return nil
		}
		if core.HasPrefix(trimmed, "[") {
			var values []map[string]any
			if result := core.JSONUnmarshalString(trimmed, &values); result.OK {
				return values
			}
			var generic []any
			if result := core.JSONUnmarshalString(trimmed, &generic); result.OK {
				return anyMapSliceValue(generic)
			}
		}
	}
	return nil
}

func boolMapValue(value any) map[string]bool {
	switch typed := value.(type) {
	case map[string]bool:
		return typed
	case map[string]any:
		values := make(map[string]bool, len(typed))
		for key, item := range typed {
			switch resolved := item.(type) {
			case bool:
				values[key] = resolved
			case string:
				values[key] = core.Lower(core.Trim(resolved)) == "true"
			default:
				values[key] = intValue(resolved) > 0
			}
		}
		return values
	case string:
		trimmed := core.Trim(typed)
		if trimmed == "" {
			return nil
		}
		if core.HasPrefix(trimmed, "{") {
			var values map[string]bool
			if result := core.JSONUnmarshalString(trimmed, &values); result.OK {
				return values
			}
			var generic map[string]any
			if result := core.JSONUnmarshalString(trimmed, &generic); result.OK {
				return boolMapValue(generic)
			}
		}
	}
	return nil
}

func listValue(value any) []string {
	switch typed := value.(type) {
	case map[string]any:
		values := make([]string, 0, len(typed))
		for key, item := range typed {
			if item == true || core.Trim(stringValue(item)) != "" {
				values = append(values, key)
			}
		}
		return cleanStrings(values)
	default:
		return stringSliceValue(value)
	}
}

func intValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		parsed := parseInt(typed)
		if parsed != 0 || core.Trim(typed) == "0" {
			return parsed
		}
	}
	return 0
}
