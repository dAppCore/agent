// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"time"

	core "dappco.re/go/core"
)

type SyncPushInput struct {
	AgentID     string           `json:"agent_id,omitempty"`
	FleetNodeID int              `json:"fleet_node_id,omitempty"`
	Dispatches  []map[string]any `json:"dispatches,omitempty"`
}

type SyncPushOutput struct {
	Success bool `json:"success"`
	Count   int  `json:"count"`
}

type SyncPullInput struct {
	AgentID     string `json:"agent_id,omitempty"`
	FleetNodeID int    `json:"fleet_node_id,omitempty"`
	Since       string `json:"since,omitempty"`
}

type SyncPullOutput struct {
	Success bool             `json:"success"`
	Count   int              `json:"count"`
	Context []map[string]any `json:"context"`
}

// record := agentic.SyncRecord{AgentID: "codex", Direction: "push", ItemsCount: 3, PayloadSize: 512, SyncedAt: "2026-03-31T12:00:00Z"}
type SyncRecord struct {
	AgentID     string `json:"agent_id,omitempty"`
	FleetNodeID int    `json:"fleet_node_id,omitempty"`
	Direction   string `json:"direction"`
	PayloadSize int    `json:"payload_size"`
	ItemsCount  int    `json:"items_count"`
	SyncedAt    string `json:"synced_at"`
}

type syncQueuedPush struct {
	AgentID     string           `json:"agent_id"`
	FleetNodeID int              `json:"fleet_node_id,omitempty"`
	Dispatches  []map[string]any `json:"dispatches"`
	QueuedAt    time.Time        `json:"queued_at"`
}

type syncStatusState struct {
	LastPushAt time.Time `json:"last_push_at,omitempty"`
	LastPullAt time.Time `json:"last_pull_at,omitempty"`
}

// result := c.Action("agentic.sync.push").Run(ctx, core.NewOptions())
func (s *PrepSubsystem) handleSyncPush(ctx context.Context, options core.Options) core.Result {
	output, err := s.syncPushInput(ctx, SyncPushInput{
		AgentID:     optionStringValue(options, "agent_id", "agent-id", "_arg"),
		FleetNodeID: optionIntValue(options, "fleet_node_id", "fleet-node-id"),
		Dispatches:  optionAnyMapSliceValue(options, "dispatches"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("agentic.sync.pull").Run(ctx, core.NewOptions())
func (s *PrepSubsystem) handleSyncPull(ctx context.Context, options core.Options) core.Result {
	output, err := s.syncPullInput(ctx, SyncPullInput{
		AgentID:     optionStringValue(options, "agent_id", "agent-id", "_arg"),
		FleetNodeID: optionIntValue(options, "fleet_node_id", "fleet-node-id"),
		Since:       optionStringValue(options, "since"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) syncPush(ctx context.Context, agentID string) (SyncPushOutput, error) {
	return s.syncPushInput(ctx, SyncPushInput{AgentID: agentID})
}

func (s *PrepSubsystem) syncPushInput(ctx context.Context, input SyncPushInput) (SyncPushOutput, error) {
	agentID := input.AgentID
	if agentID == "" {
		agentID = AgentName()
	}
	dispatches := input.Dispatches
	if len(dispatches) == 0 {
		dispatches = collectSyncDispatches()
	}
	token := s.syncToken()
	queuedPushes := readSyncQueue()
	if len(dispatches) > 0 {
		queuedPushes = append(queuedPushes, syncQueuedPush{
			AgentID:     agentID,
			FleetNodeID: input.FleetNodeID,
			Dispatches:  dispatches,
			QueuedAt:    time.Now(),
		})
	}
	if token == "" {
		if len(input.Dispatches) > 0 {
			writeSyncQueue(queuedPushes)
		}
		return SyncPushOutput{Success: true, Count: 0}, nil
	}
	if len(queuedPushes) == 0 {
		return SyncPushOutput{Success: true, Count: 0}, nil
	}

	synced := 0
	for i, queued := range queuedPushes {
		if len(queued.Dispatches) == 0 {
			continue
		}
		if err := s.postSyncPush(ctx, queued.AgentID, queued.Dispatches, token); err != nil {
			writeSyncQueue(queuedPushes[i:])
			return SyncPushOutput{Success: true, Count: synced}, nil
		}
		synced += len(queued.Dispatches)
		recordSyncPush(time.Now())
		recordSyncHistory("push", queued.AgentID, queued.FleetNodeID, len(core.JSONMarshalString(map[string]any{
			"agent_id":   queued.AgentID,
			"dispatches": queued.Dispatches,
		})), len(queued.Dispatches), time.Now())
	}

	writeSyncQueue(nil)
	return SyncPushOutput{Success: true, Count: synced}, nil
}

func (s *PrepSubsystem) syncPull(ctx context.Context, agentID string) (SyncPullOutput, error) {
	return s.syncPullInput(ctx, SyncPullInput{AgentID: agentID})
}

func (s *PrepSubsystem) syncPullInput(ctx context.Context, input SyncPullInput) (SyncPullOutput, error) {
	agentID := input.AgentID
	if agentID == "" {
		agentID = AgentName()
	}
	token := s.syncToken()
	if token == "" {
		cached := readSyncContext()
		return SyncPullOutput{Success: true, Count: len(cached), Context: cached}, nil
	}

	path := appendQueryParam("/v1/agent/context", "agent_id", agentID)
	path = appendQueryParam(path, "since", input.Since)
	endpoint := core.Concat(s.syncAPIURL(), path)
	result := HTTPGet(ctx, endpoint, token, "Bearer")
	if !result.OK {
		cached := readSyncContext()
		return SyncPullOutput{Success: true, Count: len(cached), Context: cached}, nil
	}

	var response map[string]any
	parseResult := core.JSONUnmarshalString(result.Value.(string), &response)
	if !parseResult.OK {
		cached := readSyncContext()
		return SyncPullOutput{Success: true, Count: len(cached), Context: cached}, nil
	}

	contextData := syncContextPayload(response)
	writeSyncContext(contextData)
	recordSyncPull(time.Now())
	recordSyncHistory("pull", agentID, input.FleetNodeID, len(result.Value.(string)), len(contextData), time.Now())

	return SyncPullOutput{
		Success: true,
		Count:   len(contextData),
		Context: contextData,
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
		dispatches = append(dispatches, syncDispatchRecord(workspaceDir, workspaceStatus))
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

func (s *PrepSubsystem) postSyncPush(ctx context.Context, agentID string, dispatches []map[string]any, token string) error {
	payload := map[string]any{
		"agent_id":   agentID,
		"dispatches": dispatches,
	}

	result := HTTPPost(ctx, core.Concat(s.syncAPIURL(), "/v1/agent/sync"), core.JSONMarshalString(payload), token, "Bearer")
	if result.OK {
		return nil
	}

	err, _ := result.Value.(error)
	if err == nil {
		err = core.E("agentic.sync.push", "sync push failed", nil)
	}
	return err
}

func syncStateDir() string {
	return core.JoinPath(CoreRoot(), "sync")
}

func syncQueuePath() string {
	return core.JoinPath(syncStateDir(), "queue.json")
}

func syncContextPath() string {
	return core.JoinPath(syncStateDir(), "context.json")
}

func syncStatusPath() string {
	return core.JoinPath(syncStateDir(), "status.json")
}

func readSyncQueue() []syncQueuedPush {
	var queued []syncQueuedPush
	result := fs.Read(syncQueuePath())
	if !result.OK {
		return queued
	}
	parseResult := core.JSONUnmarshalString(result.Value.(string), &queued)
	if !parseResult.OK {
		return []syncQueuedPush{}
	}
	return queued
}

func writeSyncQueue(queued []syncQueuedPush) {
	if len(queued) == 0 {
		fs.Delete(syncQueuePath())
		return
	}
	fs.EnsureDir(syncStateDir())
	fs.WriteAtomic(syncQueuePath(), core.JSONMarshalString(queued))
}

func readSyncContext() []map[string]any {
	var contextData []map[string]any
	result := fs.Read(syncContextPath())
	if !result.OK {
		return contextData
	}
	parseResult := core.JSONUnmarshalString(result.Value.(string), &contextData)
	if !parseResult.OK {
		return []map[string]any{}
	}
	return contextData
}

func writeSyncContext(contextData []map[string]any) {
	fs.EnsureDir(syncStateDir())
	fs.WriteAtomic(syncContextPath(), core.JSONMarshalString(contextData))
}

func syncContextPayload(payload map[string]any) []map[string]any {
	if contextData := payloadDataSlice(payload, "context", "items", "memories"); len(contextData) > 0 {
		return contextData
	}
	return nil
}

func syncDispatchRecord(workspaceDir string, workspaceStatus *WorkspaceStatus) map[string]any {
	record := map[string]any{
		"workspace":  WorkspaceName(workspaceDir),
		"repo":       workspaceStatus.Repo,
		"org":        workspaceStatus.Org,
		"task":       workspaceStatus.Task,
		"agent":      workspaceStatus.Agent,
		"branch":     workspaceStatus.Branch,
		"status":     workspaceStatus.Status,
		"question":   workspaceStatus.Question,
		"issue":      workspaceStatus.Issue,
		"runs":       workspaceStatus.Runs,
		"process_id": workspaceStatus.ProcessID,
		"pr_url":     workspaceStatus.PRURL,
		"started_at": workspaceStatus.StartedAt,
		"updated_at": workspaceStatus.UpdatedAt,
	}

	if report := readSyncWorkspaceReport(workspaceDir); len(report) > 0 {
		record["report"] = report
		if findings := anyMapSliceValue(report["findings"]); len(findings) > 0 {
			record["findings"] = findings
		}
		if changes := anyMapValue(report["changes"]); len(changes) > 0 {
			record["changes"] = changes
		}
	}

	return record
}

func readSyncWorkspaceReport(workspaceDir string) map[string]any {
	reportPath := core.JoinPath(WorkspaceMetaDir(workspaceDir), "report.json")
	result := fs.Read(reportPath)
	if !result.OK {
		return nil
	}

	var report map[string]any
	parseResult := core.JSONUnmarshalString(result.Value.(string), &report)
	if !parseResult.OK {
		return nil
	}

	return report
}

func readSyncStatusState() syncStatusState {
	var state syncStatusState
	result := fs.Read(syncStatusPath())
	if !result.OK {
		return state
	}

	parseResult := core.JSONUnmarshalString(result.Value.(string), &state)
	if !parseResult.OK {
		return syncStatusState{}
	}

	return state
}

func writeSyncStatusState(state syncStatusState) {
	fs.EnsureDir(syncStateDir())
	fs.WriteAtomic(syncStatusPath(), core.JSONMarshalString(state))
}

func syncRecordsPath() string {
	return core.JoinPath(syncStateDir(), "records.json")
}

func readSyncRecords() []SyncRecord {
	var records []SyncRecord
	result := fs.Read(syncRecordsPath())
	if !result.OK {
		return records
	}

	parseResult := core.JSONUnmarshalString(result.Value.(string), &records)
	if !parseResult.OK {
		return []SyncRecord{}
	}

	return records
}

func writeSyncRecords(records []SyncRecord) {
	fs.EnsureDir(syncStateDir())
	fs.WriteAtomic(syncRecordsPath(), core.JSONMarshalString(records))
}

func recordSyncHistory(direction, agentID string, fleetNodeID, payloadSize, itemsCount int, at time.Time) {
	direction = core.Trim(direction)
	if direction == "" {
		return
	}

	record := SyncRecord{
		AgentID:     core.Trim(agentID),
		FleetNodeID: fleetNodeID,
		Direction:   direction,
		PayloadSize: payloadSize,
		ItemsCount:  itemsCount,
		SyncedAt:    at.UTC().Format(time.RFC3339),
	}

	records := readSyncRecords()
	records = append(records, record)
	if len(records) > 100 {
		records = records[len(records)-100:]
	}
	writeSyncRecords(records)
}

func recordSyncPush(at time.Time) {
	state := readSyncStatusState()
	state.LastPushAt = at
	writeSyncStatusState(state)
}

func recordSyncPull(at time.Time) {
	state := readSyncStatusState()
	state.LastPullAt = at
	writeSyncStatusState(state)
}
