// SPDX-License-Identifier: EUPL-1.2

// service := monitor.New(monitor.MonitorOptions{Interval: 30 * time.Second})
// service.RegisterTools(server)
package monitor

import (
	"context"
	"time"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/agentic"
	"dappco.re/go/agent/pkg/messages"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// readResult := fs.Read(core.JoinPath(workspaceRoot, name, "status.json"))
// if text, ok := resultString(readResult); ok { _ = core.JSONUnmarshalString(text, &workspaceStatus) }
var fs = agentic.LocalFs()

func brainKeyPath(home string) string {
	return core.JoinPath(home, ".claude", "brain.key")
}

func monitorAPIURL() string {
	if u := core.Env("CORE_API_URL"); u != "" {
		return u
	}
	return "https://api.lthn.sh"
}

func monitorBrainKey() string {
	if k := core.Env("CORE_BRAIN_KEY"); k != "" {
		return k
	}
	if readResult := fs.Read(brainKeyPath(agentic.HomeDir())); readResult.OK {
		if value, ok := resultString(readResult); ok {
			return core.Trim(value)
		}
	}
	return ""
}

func resultString(result core.Result) (string, bool) {
	value, ok := result.Value.(string)
	if !ok {
		return "", false
	}
	return value, true
}

// service := monitor.New(Options{})
// service.Start(context.Background())
type Subsystem struct {
	*core.ServiceRuntime[Options]
	svc      *coremcp.Service
	interval time.Duration
	cancel   context.CancelFunc
	done     chan struct{}

	seenCompleted     map[string]bool
	seenRunning       map[string]bool
	completionsSeeded bool
	lastInboxMaxID    int
	inboxSeeded       bool
	lastSyncTimestamp int64
	lockCh            chan struct{}

	poke chan struct{}
}

// monitorLock acquires the monitor mutex — uses c.Lock("monitor") when
// Core is available, falls back to a channel-based lock for standalone use.
//
//	unlock := m.monitorLock()
//	defer unlock()
func (m *Subsystem) monitorLock() (unlock func()) {
	if m.ServiceRuntime != nil {
		mu := m.Core().Lock("monitor").Mutex
		mu.Lock()
		return mu.Unlock
	}
	m.lockCh <- struct{}{}
	return func() { <-m.lockCh }
}

// monitorRLock acquires a read-lock — uses c.Lock("monitor") when
// Core is available, falls back to the channel-based lock for standalone use.
//
//	unlock := m.monitorRLock()
//	defer unlock()
func (m *Subsystem) monitorRLock() (unlock func()) {
	if m.ServiceRuntime != nil {
		mu := m.Core().Lock("monitor").Mutex
		mu.RLock()
		return mu.RUnlock
	}
	m.lockCh <- struct{}{}
	return func() { <-m.lockCh }
}

var _ coremcp.Subsystem = (*Subsystem)(nil)

func (m *Subsystem) handleAgentStarted(ev messages.AgentStarted) {
	unlock := m.monitorLock()
	m.seenRunning[ev.Workspace] = true
	unlock()
}

func (m *Subsystem) handleAgentCompleted(ev messages.AgentCompleted) {
	unlock := m.monitorLock()
	m.seenCompleted[ev.Workspace] = true
	unlock()

	m.Poke()
	go m.checkIdleAfterDelay()
}

func (m *Subsystem) handleWorkspacePushed(ev messages.WorkspacePushed) {
	if m.ServiceRuntime == nil {
		return
	}
	m.syncWorkspacePush(ev.Repo, ev.Branch, ev.Org)
}

// c.ACTION(messages.AgentStarted{Agent: "codex", Repo: "go-io", Workspace: "core/go-io/task-5"})
// c.ACTION(messages.AgentCompleted{Agent: "codex", Repo: "go-io", Workspace: "core/go-io/task-5", Status: "completed"})
func (m *Subsystem) HandleIPCEvents(_ *core.Core, msg core.Message) core.Result {
	switch ev := msg.(type) {
	case messages.AgentCompleted:
		m.handleAgentCompleted(ev)
	case messages.AgentStarted:
		m.handleAgentStarted(ev)
	case messages.WorkspacePushed:
		m.handleWorkspacePushed(ev)
	}
	return core.Result{OK: true}
}

// options := monitor.MonitorOptions{Interval: 30 * time.Second}
// service := monitor.New(options)
type MonitorOptions struct {
	Interval time.Duration
}

// Options is kept as a compatibility alias for older callers.
type Options = MonitorOptions

// service := monitor.New(monitor.MonitorOptions{Interval: 30 * time.Second})
func New(options ...MonitorOptions) *Subsystem {
	interval := 2 * time.Minute
	if len(options) > 0 && options[0].Interval > 0 {
		interval = options[0].Interval
	}
	if envInterval := core.Env("MONITOR_INTERVAL"); envInterval != "" {
		if d, err := time.ParseDuration(envInterval); err == nil {
			interval = d
		}
	}
	return &Subsystem{
		interval:      interval,
		poke:          make(chan struct{}, 1),
		lockCh:        make(chan struct{}, 1),
		seenCompleted: make(map[string]bool),
		seenRunning:   make(map[string]bool),
	}
}

func (m *Subsystem) debug(msg string) {
	core.Debug(msg)
}

// name := service.Name() // "monitor"
func (m *Subsystem) Name() string { return "monitor" }

// service.RegisterTools(svc)
func (m *Subsystem) RegisterTools(svc *coremcp.Service) {
	m.svc = svc

	svc.Server().AddResource(&mcp.Resource{
		Name:        "Agent Status",
		URI:         "status://agents",
		Description: "Current status of all agent workspaces",
		MIMEType:    "application/json",
	}, m.agentStatusResource)
}

// service.Start(ctx)
func (m *Subsystem) Start(ctx context.Context) {
	loopContext, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.done = make(chan struct{})

	core.Info("monitor: started (interval=%s)", m.interval)

	go func() {
		defer close(m.done)
		m.loop(loopContext)
	}()
}

// result := service.OnStartup(context.Background())
// core.Println(result.OK)
func (m *Subsystem) OnStartup(ctx context.Context) core.Result {
	m.Start(ctx)
	return core.Result{OK: true}
}

// result := service.OnShutdown(context.Background())
// core.Println(result.OK)
func (m *Subsystem) OnShutdown(ctx context.Context) core.Result {
	if err := m.Shutdown(ctx); err != nil {
		return core.Fail(err)
	}
	return core.Result{OK: true}
}

// _ = service.Shutdown(ctx)
func (m *Subsystem) Shutdown(_ context.Context) error {
	if m.cancel != nil {
		m.cancel()
	}
	if m.done != nil {
		<-m.done
	}
	return nil
}

// service.Poke()
func (m *Subsystem) Poke() {
	select {
	case m.poke <- struct{}{}:
	default:
	}
}

func (m *Subsystem) checkIdleAfterDelay() {
	time.Sleep(5 * time.Second)
	if m.ServiceRuntime == nil {
		return
	}

	running, queued := m.countLiveWorkspaces()
	if running == 0 && queued == 0 {
		m.Core().ACTION(messages.QueueDrained{Completed: 0})
	}
}

func (m *Subsystem) countLiveWorkspaces() (running, queued int) {
	var runtime *core.Core
	if m.ServiceRuntime != nil {
		runtime = m.Core()
	}
	for _, path := range agentic.WorkspaceStatusPaths() {
		workspaceDir := core.PathDir(path)
		statusResult := agentic.ReadStatusResult(workspaceDir)
		if !statusResult.OK {
			continue
		}
		workspaceStatus, ok := statusResult.Value.(*agentic.WorkspaceStatus)
		if !ok || workspaceStatus == nil {
			continue
		}
		switch workspaceStatus.Status {
		case "running":
			if workspaceStatus.PID > 0 && processAlive(runtime, workspaceStatus.ProcessID, workspaceStatus.PID) {
				running++
			}
		case "queued":
			queued++
		}
	}
	return
}

func processAlive(coreApp *core.Core, processID string, pid int) bool {
	return agentic.ProcessAlive(coreApp, processID, pid)
}

func (m *Subsystem) loop(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(5 * time.Second):
	}

	m.initSyncTimestamp()

	m.check(ctx)

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.check(ctx)
		case <-m.poke:
			m.check(ctx)
		}
	}
}

func (m *Subsystem) check(ctx context.Context) {
	var statusMessages []string

	if statusMessage := m.checkCompletions(); statusMessage != "" {
		statusMessages = append(statusMessages, statusMessage)
	}

	if statusMessage := m.harvestCompleted(); statusMessage != "" {
		statusMessages = append(statusMessages, statusMessage)
	}

	if statusMessage := m.checkInbox(); statusMessage != "" {
		statusMessages = append(statusMessages, statusMessage)
	}

	if statusMessage := m.syncRepos(); statusMessage != "" {
		statusMessages = append(statusMessages, statusMessage)
	}

	if len(statusMessages) == 0 {
		return
	}

	combinedMessage := core.Join("\n", statusMessages...)
	m.notify(ctx, combinedMessage)

	if m.svc != nil {
		m.svc.Server().ResourceUpdated(ctx, &mcp.ResourceUpdatedNotificationParams{
			URI: "status://agents",
		})
	}
}

func (m *Subsystem) checkCompletions() string {
	entries := agentic.WorkspaceStatusPaths()

	running := 0
	queued := 0
	completed := 0
	var newlyCompleted []string

	unlock := m.monitorLock()
	seeded := m.completionsSeeded
	for _, entry := range entries {
		entryResult := fs.Read(entry)
		if !entryResult.OK {
			continue
		}
		entryData, ok := resultString(entryResult)
		if !ok {
			continue
		}
		var workspaceStatus struct {
			Status string `json:"status"`
			Repo   string `json:"repo"`
			Agent  string `json:"agent"`
		}
		if parseResult := core.JSONUnmarshalString(entryData, &workspaceStatus); !parseResult.OK {
			continue
		}

		workspaceName := agentic.WorkspaceName(core.PathDir(entry))

		switch workspaceStatus.Status {
		case "completed":
			completed++
			if !m.seenCompleted[workspaceName] {
				m.seenCompleted[workspaceName] = true
				if seeded {
					newlyCompleted = append(newlyCompleted, core.Sprintf("%s (%s)", workspaceStatus.Repo, workspaceStatus.Agent))
				}
			}
		case "running":
			running++
			if !m.seenRunning[workspaceName] && seeded {
				m.seenRunning[workspaceName] = true
			}
		case "queued":
			queued++
		case "blocked", "failed":
			if !m.seenCompleted[workspaceName] {
				m.seenCompleted[workspaceName] = true
				if seeded {
					newlyCompleted = append(newlyCompleted, core.Sprintf("%s (%s) [%s]", workspaceStatus.Repo, workspaceStatus.Agent, workspaceStatus.Status))
				}
			}
		}
	}
	m.completionsSeeded = true
	unlock()

	if len(newlyCompleted) == 0 {
		return ""
	}

	liveRunning, liveQueued := m.countLiveWorkspaces()
	if m.ServiceRuntime != nil && liveRunning == 0 && liveQueued == 0 {
		m.Core().ACTION(messages.QueueDrained{Completed: len(newlyCompleted)})
	}

	msg := core.Sprintf("%d agent(s) completed", len(newlyCompleted))
	if running > 0 {
		msg = core.Concat(msg, core.Sprintf(", %d still running", running))
	}
	if queued > 0 {
		msg = core.Concat(msg, core.Sprintf(", %d queued", queued))
	}
	return msg
}

func (m *Subsystem) checkInbox() string {
	brainKey := monitorBrainKey()
	if brainKey == "" {
		return ""
	}

	baseURL := monitorAPIURL()
	inboxURL := core.Concat(baseURL, "/v1/messages/inbox?agent=", core.URLEncode(agentic.AgentName()))
	httpResult := agentic.HTTPGet(context.Background(), inboxURL, core.Trim(brainKey), "Bearer")
	if !httpResult.OK {
		return ""
	}

	var inboxResponse struct {
		Data []struct {
			ID      int    `json:"id"`
			Read    bool   `json:"read"`
			From    string `json:"from"`
			Subject string `json:"subject"`
			Content string `json:"content"`
		} `json:"data"`
	}
	if parseResult := core.JSONUnmarshalString(httpResult.Value.(string), &inboxResponse); !parseResult.OK {
		m.debug("checkInbox: failed to decode response")
		return ""
	}

	maxID := 0
	unread := 0

	rUnlock := m.monitorRLock()
	prevMaxID := m.lastInboxMaxID
	seeded := m.inboxSeeded
	rUnlock()

	type inboxMessage struct {
		ID      int    `json:"id"`
		From    string `json:"from"`
		Subject string `json:"subject"`
		Content string `json:"content"`
	}
	var inboxMessages []inboxMessage

	for _, message := range inboxResponse.Data {
		if message.ID > maxID {
			maxID = message.ID
		}
		if !message.Read {
			unread++
		}
		if message.ID > prevMaxID {
			inboxMessages = append(inboxMessages, inboxMessage{
				ID:      message.ID,
				From:    message.From,
				Subject: message.Subject,
				Content: message.Content,
			})
		}
	}

	unlock := m.monitorLock()
	m.lastInboxMaxID = maxID
	m.inboxSeeded = true
	unlock()

	if !seeded {
		return ""
	}

	if maxID <= prevMaxID || len(inboxMessages) == 0 {
		return ""
	}

	if m.ServiceRuntime != nil {
		m.Core().ACTION(messages.InboxMessage{
			New:   len(inboxMessages),
			Total: unread,
		})
	}

	return core.Sprintf("%d unread message(s) in inbox", unread)
}

func (m *Subsystem) notify(ctx context.Context, message string) {
	if m.svc == nil {
		return
	}

	for session := range m.svc.Server().Sessions() {
		session.Log(ctx, &mcp.LoggingMessageParams{
			Level:  "info",
			Logger: "monitor",
			Data:   message,
		})
	}
}

func (m *Subsystem) agentStatusResource(_ context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	entries := agentic.WorkspaceStatusPaths()

	type workspaceInfo struct {
		Name   string `json:"name"`
		Status string `json:"status"`
		Repo   string `json:"repo"`
		Agent  string `json:"agent"`
		PRURL  string `json:"pr_url,omitempty"`
	}

	var workspaces []workspaceInfo
	for _, entry := range entries {
		entryResult := fs.Read(entry)
		if !entryResult.OK {
			continue
		}
		entryData, ok := resultString(entryResult)
		if !ok {
			continue
		}
		var workspaceStatus struct {
			Status string `json:"status"`
			Repo   string `json:"repo"`
			Agent  string `json:"agent"`
			PRURL  string `json:"pr_url"`
		}
		if parseResult := core.JSONUnmarshalString(entryData, &workspaceStatus); !parseResult.OK {
			continue
		}
		workspaces = append(workspaces, workspaceInfo{
			Name:   agentic.WorkspaceName(core.PathDir(entry)),
			Status: workspaceStatus.Status,
			Repo:   workspaceStatus.Repo,
			Agent:  workspaceStatus.Agent,
			PRURL:  workspaceStatus.PRURL,
		})
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      "status://agents",
				MIMEType: "application/json",
				Text:     core.JSONMarshalString(workspaces),
			},
		},
	}, nil
}
