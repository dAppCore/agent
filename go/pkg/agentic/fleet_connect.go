// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"bufio"
	"context"
	"net/http"
	"sync"
	"time"

	core "dappco.re/go"
)

var fleetBackoffSchedule = []time.Duration{
	time.Second,
	2 * time.Second,
	4 * time.Second,
	8 * time.Second,
	16 * time.Second,
	30 * time.Second,
}

var fleetPollInterval = 30 * time.Second
var fleetHeartbeatInterval = 60 * time.Second
var fleetPollingFailureThreshold = 3

const fleetPollAction = "agentic.fleet.poll"

var fleetSleep = func(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		select {
		case <-ctx.Done():
			return false
		default:
			return true
		}
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

type fleetClientConfig struct {
	APIURL       string
	AgentID      string
	AgentAPIKey  string
	Status       string
	Capabilities []string
}

type fleetRuntimeSnapshot struct {
	APIURL           string
	AgentID          string
	State            string
	Transport        string
	LastError        string
	LastHeartbeatAt  string
	LastConnectedAt  string
	LastEventAt      string
	LastTaskReceived string
	LastTask         FleetTask
}

var fleetRuntimeState = struct {
	mu       sync.RWMutex
	snapshot fleetRuntimeSnapshot
}{
	snapshot: fleetRuntimeSnapshot{State: "offline"},
}

// result := subsystem.Connect(ctx, core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))
func (s *PrepSubsystem) Connect(ctx context.Context, options core.Options) core.Result {
	config := fleetClientConfigFromOptions(s, options)
	if validation := validateFleetClientConfig("agentic.fleet.connect", config, true); !validation.OK {
		return validation
	}

	fleetRememberBase(config)
	fleetRememberState("connecting", "sse", "")

	heartbeatContext, cancelHeartbeat := context.WithCancel(ctx)
	defer cancelHeartbeat()

	if fleetHeartbeatInterval > 0 {
		go func() {
			if result := s.runFleetHeartbeat(heartbeatContext, config); !result.OK {
				core.Warn("agentic.fleet.heartbeat: stopped", "reason", result.Error())
			}
		}()
	}

	var pollingCancel context.CancelFunc
	var pollingDone chan struct{}
	consecutiveFailures := 0

	for ctx.Err() == nil {
		fleetClearCompletedPollFallback(&pollingCancel, &pollingDone)

		result := s.connectFleetEventStream(ctx, config)
		if result.OK {
			consecutiveFailures = 0
			fleetStopPollFallback(pollingCancel, pollingDone)
			pollingCancel = nil
			pollingDone = nil
			continue
		}

		if ctx.Err() != nil {
			break
		}

		consecutiveFailures++
		err := commandResultError("agentic.fleet.connect", result)
		fleetRememberState("disconnected", "sse", err.Error())

		if consecutiveFailures >= fleetPollingFailureThreshold && pollingCancel == nil {
			pollingCancel, pollingDone = s.startFleetPollFallback(ctx, config)
		}

		if !fleetSleep(ctx, fleetBackoffDelay(consecutiveFailures)) {
			break
		}
	}

	fleetStopPollFallback(pollingCancel, pollingDone)

	fleetRememberState("offline", fleetRuntimeSnapshotValue().Transport, "")
	return core.Result{OK: true}
}

func fleetClearCompletedPollFallback(cancel *context.CancelFunc, done *chan struct{}) {
	if *done == nil {
		return
	}

	select {
	case <-*done:
		*done = nil
		*cancel = nil
	default:
	}
}

func fleetStopPollFallback(cancel context.CancelFunc, done chan struct{}) {
	if cancel == nil {
		return
	}

	cancel()
	if done != nil {
		<-done
	}
}

func (s *PrepSubsystem) startFleetPollFallback(ctx context.Context, config fleetClientConfig) (context.CancelFunc, chan struct{}) {
	pollingContext, cancelPolling := context.WithCancel(ctx)
	pollingDone := make(chan struct{})
	go func() {
		defer close(pollingDone)
		if result := s.runFleetPollFallback(pollingContext, config); !result.OK {
			core.Warn("fleet poll fallback exited", "reason", result.Value)
		}
	}()
	return cancelPolling, pollingDone
}

// result := subsystem.PollFallback(ctx, core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))
func (s *PrepSubsystem) PollFallback(ctx context.Context, options core.Options) core.Result {
	config := fleetClientConfigFromOptions(s, options)
	if validation := validateFleetClientConfig(fleetPollAction, config, true); !validation.OK {
		return validation
	}
	return s.runFleetPollFallback(ctx, config)
}

// result := subsystem.Heartbeat(ctx, core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))
func (s *PrepSubsystem) Heartbeat(ctx context.Context, options core.Options) core.Result {
	config := fleetClientConfigFromOptions(s, options)
	if validation := validateFleetClientConfig("agentic.fleet.heartbeat", config, true); !validation.OK {
		return validation
	}
	return s.runFleetHeartbeat(ctx, config)
}

func fleetClientConfigFromOptions(s *PrepSubsystem, options core.Options) fleetClientConfig {
	status := optionStringValue(options, "status")
	if status == "" {
		status = "online"
	}

	token := optionStringValue(options, "agent_api_key", "agent-api-key", "token")
	if token == "" {
		token = fleetAgentAPIKey(s)
	}

	agentID := optionStringValue(options, "agent_id", "agent-id", "_arg")
	if agentID == "" {
		agentID = AgentName()
	}

	return fleetClientConfig{
		APIURL:       fleetAPIURLFromOptions(s, options),
		AgentID:      agentID,
		AgentAPIKey:  token,
		Status:       status,
		Capabilities: optionStringSliceValue(options, "capabilities"),
	}
}

func validateFleetClientConfig(action string, config fleetClientConfig, requireToken bool) core.Result {
	if core.Trim(config.APIURL) == "" {
		return core.Result{Value: core.E(action, "api url is required", nil), OK: false}
	}

	if core.Trim(config.AgentID) == "" {
		return core.Result{Value: core.E(action, "agent_id is required", nil), OK: false}
	}

	if requireToken && core.Trim(config.AgentAPIKey) == "" {
		return core.Result{Value: core.E(action, core.Concat("no fleet api key configured at ", fleetAgentKeyPath()), nil), OK: false}
	}

	return core.Result{OK: true}
}

func fleetAPIURLFromOptions(s *PrepSubsystem, options core.Options) string {
	if apiURL := optionStringValue(options, "api", "api_url", "api-url"); apiURL != "" {
		return core.TrimSuffix(apiURL, "/")
	}
	if envURL := core.Env("CORE_API_URL"); envURL != "" {
		return core.TrimSuffix(envURL, "/")
	}
	if s != nil && s.brainURL != "" {
		return core.TrimSuffix(s.brainURL, "/")
	}
	return "https://api.lthn.sh"
}

func fleetAgentKeyPath() string {
	return core.JoinPath(HomeDir(), ".core", "agent.key")
}

func fleetStatusSnapshotPath() string {
	return core.JoinPath(HomeDir(), ".core", "fleet.status.json")
}

func fleetAgentAPIKey(s *PrepSubsystem) string {
	if value := core.Env("CORE_AGENT_API_KEY"); value != "" {
		return core.Trim(value)
	}
	if value := core.Env("CORE_BRAIN_KEY"); value != "" {
		return core.Trim(value)
	}
	if readResult := fs.Read(fleetAgentKeyPath()); readResult.OK {
		return core.Trim(readResult.Value.(string))
	}
	if readResult := fs.Read(core.JoinPath(HomeDir(), ".claude", "agent-api.key")); readResult.OK {
		return core.Trim(readResult.Value.(string))
	}
	if readResult := fs.Read(core.JoinPath(HomeDir(), ".claude", "brain.key")); readResult.OK {
		return core.Trim(readResult.Value.(string))
	}
	if s != nil {
		return core.Trim(s.brainKey)
	}
	return ""
}

func (s *PrepSubsystem) fleetJSONRequest(ctx context.Context, action string, config fleetClientConfig, method, path string, body any) core.Result {
	bodyString := ""
	if body != nil {
		bodyString = core.JSONMarshalString(body)
	}

	requestResult := HTTPDo(ctx, method, fleetURL(config.APIURL, path), bodyString, config.AgentAPIKey, "Bearer")
	if !requestResult.OK {
		return core.Result{Value: platformResultError(action, requestResult), OK: false}
	}

	rawBody := core.Trim(stringValue(requestResult.Value))
	if rawBody == "" {
		return core.Result{Value: map[string]any{}, OK: true}
	}

	var payload map[string]any
	parseResult := core.JSONUnmarshalString(rawBody, &payload)
	if !parseResult.OK {
		err, _ := parseResult.Value.(error)
		return core.Result{Value: core.E(action, "failed to parse fleet response", err), OK: false}
	}

	return core.Result{Value: payload, OK: true}
}

func (s *PrepSubsystem) fleetEventRequest(ctx context.Context, action string, config fleetClientConfig) core.Result {
	path := appendQueryParam("/v1/fleet/events", "agent_id", config.AgentID)
	path = appendQuerySlice(path, "capabilities[]", config.Capabilities)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fleetURL(config.APIURL, path), nil)
	if err != nil {
		return core.Result{Value: core.E(action, "create request", err), OK: false}
	}

	request.Header.Set("Accept", "text/event-stream, application/json")
	if config.AgentAPIKey != "" {
		request.Header.Set("Authorization", core.Concat("Bearer ", config.AgentAPIKey))
	}

	response, err := defaultClient.Do(request)
	if err != nil {
		return core.Result{Value: core.E(action, "request failed", err), OK: false}
	}

	if response.StatusCode >= 400 {
		readResult := core.ReadAll(response.Body)
		if !readResult.OK {
			return core.Result{Value: core.E(action, core.Sprintf("HTTP %d", response.StatusCode), nil), OK: false}
		}
		return core.Result{
			Value: platformResultError(action, core.Result{Value: readResult.Value, OK: false}),
			OK:    false,
		}
	}

	return core.Result{Value: response, OK: true}
}

func (s *PrepSubsystem) connectFleetEventStream(ctx context.Context, config fleetClientConfig) core.Result {
	requestResult := s.fleetEventRequest(ctx, "agentic.fleet.connect", config)
	if !requestResult.OK {
		return requestResult
	}

	response, ok := requestResult.Value.(*http.Response)
	if !ok || response == nil {
		return core.Result{Value: core.E("agentic.fleet.connect", "invalid event stream response", nil), OK: false}
	}
	defer core.CloseStream(response.Body)

	fleetRememberBase(config)
	fleetRememberState("connected", "sse", "")
	fleetRememberConnected()

	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, 0, 4096), 1024*1024)

	eventCount := 0
	rawLines := make([]string, 0, 4)

	flushEvent := func() {
		if len(rawLines) == 0 {
			return
		}

		eventBody := core.Join("\n", rawLines...)
		rawLines = rawLines[:0]

		payload := s.eventPayloadValue(eventBody)
		output, err := parseFleetEventOutput(payload)
		if err != nil {
			return
		}

		eventCount++
		fleetRememberState("connected", "sse", "")
		fleetRememberEvent(output.Event)
	}

	for scanner.Scan() {
		line := core.Trim(scanner.Text())
		if line == "" {
			flushEvent()
			continue
		}
		rawLines = append(rawLines, line)
	}

	flushEvent()

	if err := scanner.Err(); err != nil {
		if eventCount > 0 {
			return core.Result{Value: eventCount, OK: true}
		}
		return core.Result{Value: core.E("agentic.fleet.connect", "read event stream", err), OK: false}
	}

	if ctx.Err() != nil {
		return core.Result{OK: true}
	}

	if eventCount > 0 {
		return core.Result{Value: eventCount, OK: true}
	}

	return core.Result{Value: core.E("agentic.fleet.connect", "event stream closed before any events arrived", nil), OK: false}
}

func (s *PrepSubsystem) runFleetPollFallback(ctx context.Context, config fleetClientConfig) core.Result {
	fleetRememberBase(config)
	fleetRememberState("polling", "poll", "")

	for ctx.Err() == nil {
		result := s.pollFleetNextTask(ctx, config)
		if result.OK {
			task, _ := result.Value.(*FleetTask)
			if task != nil {
				return core.Result{Value: task, OK: true}
			}
		} else {
			err := commandResultError(fleetPollAction, result)
			fleetRememberState("polling", "poll", err.Error())
		}

		if !fleetSleep(ctx, fleetPollInterval) {
			break
		}
	}

	return core.Result{OK: true}
}

func (s *PrepSubsystem) pollFleetNextTask(ctx context.Context, config fleetClientConfig) core.Result {
	path := appendQueryParam("/v1/fleet/task/next", "agent_id", config.AgentID)
	path = appendQuerySlice(path, "capabilities[]", config.Capabilities)

	result := s.fleetJSONRequest(ctx, fleetPollAction, config, http.MethodGet, path, nil)
	if !result.OK {
		return result
	}

	payload, ok := result.Value.(map[string]any)
	if !ok {
		return core.Result{Value: core.E(fleetPollAction, "invalid fleet polling payload", nil), OK: false}
	}

	taskValues := payloadResourceMap(payload, "task")
	if len(taskValues) == 0 {
		return core.Result{OK: true}
	}

	task := parseFleetTask(taskValues)
	fleetRememberTask(task)
	return core.Result{Value: &task, OK: true}
}

func (s *PrepSubsystem) runFleetHeartbeat(ctx context.Context, config fleetClientConfig) core.Result {
	if fleetHeartbeatInterval <= 0 {
		return core.Result{OK: true}
	}

	fleetRememberBase(config)

	for ctx.Err() == nil {
		result := s.postFleetHeartbeat(ctx, config)
		if !result.OK {
			err := commandResultError("agentic.fleet.heartbeat", result)
			fleetRememberState(fleetRuntimeSnapshotValue().State, fleetRuntimeSnapshotValue().Transport, err.Error())
		}

		if !fleetSleep(ctx, fleetHeartbeatInterval) {
			break
		}
	}

	return core.Result{OK: true}
}

func (s *PrepSubsystem) postFleetHeartbeat(ctx context.Context, config fleetClientConfig) core.Result {
	result := s.fleetJSONRequest(ctx, "agentic.fleet.heartbeat", config, http.MethodPost, "/v1/fleet/heartbeat", map[string]any{
		"agent_id": config.AgentID,
		"status":   config.Status,
	})
	if !result.OK {
		return result
	}

	fleetRememberHeartbeat()
	return result
}

func fleetURL(apiURL, path string) string {
	return core.Concat(core.TrimSuffix(apiURL, "/"), path)
}

func fleetBackoffDelay(failures int) time.Duration {
	if len(fleetBackoffSchedule) == 0 {
		return 30 * time.Second
	}

	index := failures - 1
	if index < 0 {
		index = 0
	}
	if index >= len(fleetBackoffSchedule) {
		index = len(fleetBackoffSchedule) - 1
	}
	return fleetBackoffSchedule[index]
}

func fleetRuntimeSnapshotValue() fleetRuntimeSnapshot {
	fleetRuntimeState.mu.RLock()
	snapshot := fleetRuntimeState.snapshot
	fleetRuntimeState.mu.RUnlock()

	if !fleetSnapshotEmpty(snapshot) {
		return snapshot
	}

	persisted := loadFleetRuntimeSnapshot()
	if persisted.State == "" {
		persisted.State = "offline"
	}
	return persisted
}

func fleetRememberBase(config fleetClientConfig) {
	fleetRuntimeState.mu.Lock()
	if config.APIURL != "" {
		fleetRuntimeState.snapshot.APIURL = config.APIURL
	}
	if config.AgentID != "" {
		fleetRuntimeState.snapshot.AgentID = config.AgentID
	}
	snapshot := fleetRuntimeState.snapshot
	fleetRuntimeState.mu.Unlock()
	persistFleetRuntimeSnapshot(snapshot)
}

func fleetRememberState(state, transport, lastError string) {
	fleetRuntimeState.mu.Lock()
	if state != "" {
		fleetRuntimeState.snapshot.State = state
	}
	if transport != "" {
		fleetRuntimeState.snapshot.Transport = transport
	}
	fleetRuntimeState.snapshot.LastError = core.Trim(lastError)
	snapshot := fleetRuntimeState.snapshot
	fleetRuntimeState.mu.Unlock()
	persistFleetRuntimeSnapshot(snapshot)
}

func fleetRememberConnected() {
	fleetRuntimeState.mu.Lock()
	fleetRuntimeState.snapshot.LastConnectedAt = time.Now().Format(time.RFC3339)
	snapshot := fleetRuntimeState.snapshot
	fleetRuntimeState.mu.Unlock()
	persistFleetRuntimeSnapshot(snapshot)
}

func fleetRememberHeartbeat() {
	fleetRuntimeState.mu.Lock()
	fleetRuntimeState.snapshot.LastHeartbeatAt = time.Now().Format(time.RFC3339)
	fleetRuntimeState.snapshot.LastError = ""
	snapshot := fleetRuntimeState.snapshot
	fleetRuntimeState.mu.Unlock()
	persistFleetRuntimeSnapshot(snapshot)
}

func fleetRememberEvent(event FleetEvent) {
	fleetRuntimeState.mu.Lock()
	fleetRuntimeState.snapshot.LastEventAt = time.Now().Format(time.RFC3339)
	fleetRuntimeState.snapshot.LastError = ""

	task := fleetTaskFromEvent(event)
	if task.ID > 0 || task.Repo != "" || task.Task != "" {
		fleetRuntimeState.snapshot.LastTask = task
		fleetRuntimeState.snapshot.LastTaskReceived = fleetRuntimeState.snapshot.LastEventAt
	}
	snapshot := fleetRuntimeState.snapshot
	fleetRuntimeState.mu.Unlock()
	persistFleetRuntimeSnapshot(snapshot)
}

func fleetRememberTask(task FleetTask) {
	fleetRuntimeState.mu.Lock()
	fleetRuntimeState.snapshot.LastTask = task
	fleetRuntimeState.snapshot.LastTaskReceived = time.Now().Format(time.RFC3339)
	fleetRuntimeState.snapshot.LastError = ""
	snapshot := fleetRuntimeState.snapshot
	fleetRuntimeState.mu.Unlock()
	persistFleetRuntimeSnapshot(snapshot)
}

func fleetTaskFromEvent(event FleetEvent) FleetTask {
	payload := event.Payload
	return FleetTask{
		ID:         event.TaskID,
		Repo:       event.Repo,
		Branch:     event.Branch,
		Status:     event.Status,
		Task:       stringValue(payload["task"]),
		Template:   stringValue(payload["template"]),
		AgentModel: stringValue(payload["agent_model"]),
	}
}

func resetFleetRuntimeState() {
	fleetRuntimeState.mu.Lock()
	fleetRuntimeState.snapshot = fleetRuntimeSnapshot{State: "offline"}
	fleetRuntimeState.mu.Unlock()
	if deleteResult := fs.Delete(fleetStatusSnapshotPath()); !deleteResult.OK && fs.Exists(fleetStatusSnapshotPath()).OK {
		core.Warn("agentic: failed to delete fleet status snapshot", `path`, fleetStatusSnapshotPath(), "reason", deleteResult.Value)
	}
}

func fleetSnapshotEmpty(snapshot fleetRuntimeSnapshot) bool {
	return snapshot.APIURL == "" &&
		snapshot.AgentID == "" &&
		snapshot.Transport == "" &&
		snapshot.LastError == "" &&
		snapshot.LastHeartbeatAt == "" &&
		snapshot.LastConnectedAt == "" &&
		snapshot.LastEventAt == "" &&
		snapshot.LastTaskReceived == "" &&
		snapshot.LastTask.ID == 0 &&
		snapshot.LastTask.Repo == "" &&
		snapshot.LastTask.Task == "" &&
		(snapshot.State == "" || snapshot.State == "offline")
}

func persistFleetRuntimeSnapshot(snapshot fleetRuntimeSnapshot) {
	if ensureResult := fs.EnsureDir(core.PathDir(fleetStatusSnapshotPath())); !ensureResult.OK {
		return
	}
	if writeResult := fs.WriteMode(fleetStatusSnapshotPath(), core.JSONMarshalString(snapshot), 0644); !writeResult.OK {
		core.Warn("agentic: failed to write fleet status snapshot", `path`, fleetStatusSnapshotPath(), "reason", writeResult.Value)
	}
}

func loadFleetRuntimeSnapshot() fleetRuntimeSnapshot {
	readResult := fs.Read(fleetStatusSnapshotPath())
	if !readResult.OK {
		return fleetRuntimeSnapshot{State: "offline"}
	}

	var snapshot fleetRuntimeSnapshot
	parseResult := core.JSONUnmarshalString(readResult.Value.(string), &snapshot)
	if !parseResult.OK {
		return fleetRuntimeSnapshot{State: "offline"}
	}
	return snapshot
}
