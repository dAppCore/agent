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
	// QueueOnly skips the collectSyncDispatches() scan so the caller only
	// drains entries already queued. Used by the flush loop to avoid
	// re-adding the same completed workspaces on every tick.
	QueueOnly bool `json:"-"`
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
	Attempts    int              `json:"attempts,omitempty"`
	NextAttempt time.Time        `json:"next_attempt,omitempty"`
}

type syncStatusState struct {
	LastPushAt time.Time `json:"last_push_at,omitempty"`
	LastPullAt time.Time `json:"last_pull_at,omitempty"`
}

// syncBackoffSchedule implements RFC §16.5 — 1s → 5s → 15s → 60s → 5min max.
// schedule := syncBackoffSchedule(2) // 15s
// next := time.Now().Add(schedule)
func syncBackoffSchedule(attempts int) time.Duration {
	switch {
	case attempts <= 0:
		return 0
	case attempts == 1:
		return time.Second
	case attempts == 2:
		return 5 * time.Second
	case attempts == 3:
		return 15 * time.Second
	case attempts == 4:
		return 60 * time.Second
	default:
		return 5 * time.Minute
	}
}

// syncFlushScheduleInterval is the cadence at which queued pushes are retried
// when the agent has been unable to reach the platform. Per RFC §16.5 the
// retry window max is 5 minutes, so the scheduler wakes at that cadence and
// each queued entry enforces its own NextAttempt gate.
const syncFlushScheduleInterval = time.Minute

// ctx, cancel := context.WithCancel(context.Background())
// go s.runSyncFlushLoop(ctx, time.Minute)
func (s *PrepSubsystem) runSyncFlushLoop(ctx context.Context, interval time.Duration) {
	if ctx == nil || interval <= 0 {
		return
	}
	if s == nil || s.syncToken() == "" {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if len(s.readSyncQueue()) == 0 {
				continue
			}
			// QueueOnly keeps syncPushInput from re-scanning workspaces — the
			// flush loop only drains entries already queued.
			if _, err := s.syncPushInput(ctx, SyncPushInput{QueueOnly: true}); err != nil {
				core.Warn("sync flush loop failed", "error", err)
			}
		}
	}
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
	if len(dispatches) == 0 && !input.QueueOnly {
		dispatches = collectSyncDispatches()
	}
	token := s.syncToken()
	queuedPushes := s.readSyncQueue()
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
			s.writeSyncQueue(queuedPushes)
		}
		return SyncPushOutput{Success: true, Count: 0}, nil
	}
	if len(queuedPushes) == 0 {
		return SyncPushOutput{Success: true, Count: 0}, nil
	}

	synced := 0
	now := time.Now()
	for i, queued := range queuedPushes {
		if len(queued.Dispatches) == 0 {
			continue
		}
		if !queued.NextAttempt.IsZero() && queued.NextAttempt.After(now) {
			// Respect backoff — persist remaining tail so queue survives restart.
			s.writeSyncQueue(queuedPushes[i:])
			return SyncPushOutput{Success: true, Count: synced}, nil
		}
		if err := s.postSyncPush(ctx, queued.AgentID, queued.Dispatches, token); err != nil {
			remaining := append([]syncQueuedPush(nil), queuedPushes[i:]...)
			remaining[0].Attempts = queued.Attempts + 1
			remaining[0].NextAttempt = time.Now().Add(syncBackoffSchedule(remaining[0].Attempts))
			s.writeSyncQueue(remaining)
			return SyncPushOutput{Success: true, Count: synced}, nil
		}
		synced += len(queued.Dispatches)
		markDispatchesSynced(queued.Dispatches)
		recordSyncPush(time.Now())
		recordSyncHistory("push", queued.AgentID, queued.FleetNodeID, len(core.JSONMarshalString(map[string]any{
			"agent_id":   queued.AgentID,
			"dispatches": queued.Dispatches,
		})), len(queued.Dispatches), time.Now())
	}

	s.writeSyncQueue(nil)
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
	ledger := readSyncLedger()
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
		dispatchID := syncDispatchID(workspaceDir, workspaceStatus)
		if synced, ok := ledger[dispatchID]; ok && synced == syncDispatchFingerprint(workspaceStatus) {
			continue
		}
		record := syncDispatchRecord(workspaceDir, workspaceStatus)
		record["id"] = dispatchID
		dispatches = append(dispatches, record)
	}
	return dispatches
}

// id := syncDispatchID(workspaceDir, workspaceStatus) // "core/go-io/task-5"
func syncDispatchID(workspaceDir string, workspaceStatus *WorkspaceStatus) string {
	if workspaceStatus == nil {
		return WorkspaceName(workspaceDir)
	}
	return WorkspaceName(workspaceDir)
}

// fingerprint := syncDispatchFingerprint(workspaceStatus) // "2026-04-14T12:00:00Z#3"
// A dispatch is considered unchanged when (updated_at, runs) matches.
// Any new activity (re-dispatch, status change) generates a fresh fingerprint.
func syncDispatchFingerprint(workspaceStatus *WorkspaceStatus) string {
	if workspaceStatus == nil {
		return ""
	}
	return core.Concat(workspaceStatus.UpdatedAt.UTC().Format(time.RFC3339), "#", core.Sprintf("%d", workspaceStatus.Runs))
}

// ledger := readSyncLedger() // map[dispatchID]fingerprint of last push
func readSyncLedger() map[string]string {
	ledger := map[string]string{}
	result := fs.Read(syncLedgerPath())
	if !result.OK {
		return ledger
	}
	content := core.Trim(result.Value.(string))
	if content == "" {
		return ledger
	}
	if parseResult := core.JSONUnmarshalString(content, &ledger); !parseResult.OK {
		return map[string]string{}
	}
	return ledger
}

// writeSyncLedger persists the dispatched fingerprints so the next scan
// can skip workspaces that have already been pushed.
func writeSyncLedger(ledger map[string]string) {
	if len(ledger) == 0 {
		if deleteResult := fs.Delete(syncLedgerPath()); !deleteResult.OK {
			core.Warn("agentic: failed to delete sync ledger", "path", syncLedgerPath(), "reason", deleteResult.Value)
		}
		return
	}
	if ensureResult := fs.EnsureDir(syncStateDir()); !ensureResult.OK {
		core.Warn("agentic: failed to prepare sync ledger directory", "path", syncStateDir(), "reason", ensureResult.Value)
		return
	}
	if writeResult := fs.WriteAtomic(syncLedgerPath(), core.JSONMarshalString(ledger)); !writeResult.OK {
		core.Warn("agentic: failed to write sync ledger", "path", syncLedgerPath(), "reason", writeResult.Value)
	}
}

// markDispatchesSynced records which dispatches were successfully pushed so
// collectSyncDispatches skips them on the next scan.
func markDispatchesSynced(dispatches []map[string]any) {
	if len(dispatches) == 0 {
		return
	}
	ledger := readSyncLedger()
	changed := false
	for _, record := range dispatches {
		id := stringValue(record["id"])
		if id == "" {
			id = stringValue(record["workspace"])
		}
		if id == "" {
			continue
		}
		updatedAt := ""
		switch v := record["updated_at"].(type) {
		case time.Time:
			updatedAt = v.UTC().Format(time.RFC3339)
		case string:
			updatedAt = v
		}
		runs := 0
		if v, ok := record["runs"].(int); ok {
			runs = v
		}
		ledger[id] = core.Concat(updatedAt, "#", core.Sprintf("%d", runs))
		changed = true
	}
	if changed {
		writeSyncLedger(ledger)
	}
}

func syncLedgerPath() string {
	return core.JoinPath(syncStateDir(), "ledger.json")
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
		if deleteResult := fs.Delete(syncQueuePath()); !deleteResult.OK {
			core.Warn("agentic: failed to delete sync queue", "path", syncQueuePath(), "reason", deleteResult.Value)
		}
		return
	}
	if ensureResult := fs.EnsureDir(syncStateDir()); !ensureResult.OK {
		core.Warn("agentic: failed to prepare sync queue directory", "path", syncStateDir(), "reason", ensureResult.Value)
		return
	}
	if writeResult := fs.WriteAtomic(syncQueuePath(), core.JSONMarshalString(queued)); !writeResult.OK {
		core.Warn("agentic: failed to write sync queue", "path", syncQueuePath(), "reason", writeResult.Value)
	}
}

// syncQueueStoreKey is the canonical key for the sync queue inside go-store —
// the queue is a single JSON blob keyed under stateSyncQueueGroup so RFC §16.5
// "Queue persists across restarts in db.duckdb" holds.
//
// Usage example: `key := syncQueueStoreKey // "queue"`
const syncQueueStoreKey = "queue"

// readSyncQueue reads the queued sync pushes from go-store first (RFC §16.5)
// and falls back to the JSON file when the store is unavailable. Falling back
// keeps offline deployments working through the rollout.
//
// Usage example: `queued := s.readSyncQueue()`
func (s *PrepSubsystem) readSyncQueue() []syncQueuedPush {
	if s != nil {
		if value, ok := s.stateStoreGet(stateSyncQueueGroup, syncQueueStoreKey); ok {
			var queued []syncQueuedPush
			if result := core.JSONUnmarshalString(value, &queued); result.OK {
				return queued
			}
		}
	}
	return readSyncQueue()
}

// writeSyncQueue persists the queued sync pushes to go-store (RFC §16.5) and
// mirrors the JSON file so file-only consumers (debug tooling, manual recovery)
// continue to work.
//
// Usage example: `s.writeSyncQueue(queued)`
func (s *PrepSubsystem) writeSyncQueue(queued []syncQueuedPush) {
	if s != nil {
		if len(queued) == 0 {
			s.stateStoreDelete(stateSyncQueueGroup, syncQueueStoreKey)
		} else {
			s.stateStoreSet(stateSyncQueueGroup, syncQueueStoreKey, queued)
		}
	}
	writeSyncQueue(queued)
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
	if ensureResult := fs.EnsureDir(syncStateDir()); !ensureResult.OK {
		core.Warn("agentic: failed to prepare sync context directory", "path", syncStateDir(), "reason", ensureResult.Value)
		return
	}
	if writeResult := fs.WriteAtomic(syncContextPath(), core.JSONMarshalString(contextData)); !writeResult.OK {
		core.Warn("agentic: failed to write sync context", "path", syncContextPath(), "reason", writeResult.Value)
	}
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
	if ensureResult := fs.EnsureDir(syncStateDir()); !ensureResult.OK {
		core.Warn("agentic: failed to prepare sync status directory", "path", syncStateDir(), "reason", ensureResult.Value)
		return
	}
	if writeResult := fs.WriteAtomic(syncStatusPath(), core.JSONMarshalString(state)); !writeResult.OK {
		core.Warn("agentic: failed to write sync status", "path", syncStatusPath(), "reason", writeResult.Value)
	}
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
	if ensureResult := fs.EnsureDir(syncStateDir()); !ensureResult.OK {
		core.Warn("agentic: failed to prepare sync records directory", "path", syncStateDir(), "reason", ensureResult.Value)
		return
	}
	if writeResult := fs.WriteAtomic(syncRecordsPath(), core.JSONMarshalString(records)); !writeResult.OK {
		core.Warn("agentic: failed to write sync records", "path", syncRecordsPath(), "reason", writeResult.Value)
	}
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
