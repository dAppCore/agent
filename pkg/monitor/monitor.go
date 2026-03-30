// SPDX-License-Identifier: EUPL-1.2

// Package monitor polls workspace state, repo drift, and agent inboxes, then
// pushes the current view to connected MCP clients.
//
//	mon := monitor.New(monitor.Options{Interval: 30 * time.Second})
//	mon.RegisterTools(server)
package monitor

import (
	"context"
	"sync"
	"time"

	"dappco.re/go/agent/pkg/agentic"
	"dappco.re/go/agent/pkg/messages"
	core "dappco.re/go/core"
	coremcp "forge.lthn.ai/core/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// fs provides unrestricted filesystem access (root "/" = no sandbox).
//
//	r := fs.Read(core.JoinPath(wsRoot, name, "status.json"))
//	if text, ok := resultString(r); ok { _ = core.JSONUnmarshalString(text, &st) }
var fs = agentic.LocalFs()

type channelSender interface {
	ChannelSend(ctx context.Context, channel string, data any)
}

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
	if r := fs.Read(brainKeyPath(agentic.HomeDir())); r.OK {
		if value, ok := resultString(r); ok {
			return core.Trim(value)
		}
	}
	return ""
}

func resultString(r core.Result) (string, bool) {
	value, ok := r.Value.(string)
	if !ok {
		return "", false
	}
	return value, true
}

// Subsystem owns the long-running monitor loop and MCP resource surface.
//
//	mon := monitor.New()
//	mon.Start(context.Background())
type Subsystem struct {
	*core.ServiceRuntime[Options]
	server   *mcp.Server
	interval time.Duration
	cancel   context.CancelFunc
	wg       sync.WaitGroup

	// Track last seen state to only notify on changes
	lastCompletedCount int             // completed workspaces seen on the last scan
	seenCompleted      map[string]bool // workspace names we've already notified about
	seenRunning        map[string]bool // workspace names we've already sent start notification for
	completionsSeeded  bool            // true after first completions check
	lastInboxMaxID     int             // highest message ID seen
	inboxSeeded        bool            // true after first inbox check
	lastSyncTimestamp  int64
	mu                 sync.Mutex

	// Event-driven poke channel — dispatch goroutine sends here on completion
	poke chan struct{}
}

var _ coremcp.Subsystem = (*Subsystem)(nil)

// SetCore preserves direct test setup without going through core.WithService.
// Deprecated: prefer Register with core.WithService(monitor.Register).
//
//	mon.SetCore(c)
func (m *Subsystem) SetCore(c *core.Core) {
	m.ServiceRuntime = core.NewServiceRuntime(c, Options{})
}

// handleAgentStarted tracks started agents.
func (m *Subsystem) handleAgentStarted(ev messages.AgentStarted) {
	m.mu.Lock()
	m.seenRunning[ev.Workspace] = true
	m.mu.Unlock()
}

// handleAgentCompleted processes agent completion — emits notifications and checks queue drain.
func (m *Subsystem) handleAgentCompleted(ev messages.AgentCompleted) {
	m.mu.Lock()
	m.seenCompleted[ev.Workspace] = true
	m.mu.Unlock()

	m.Poke()
	go m.checkIdleAfterDelay()
}

// HandleIPCEvents lets Core auto-wire monitor side-effects for IPC messages.
//
//	c.ACTION(messages.AgentStarted{Agent: "codex", Repo: "go-io", Workspace: "core/go-io/task-5"})
//	c.ACTION(messages.AgentCompleted{Agent: "codex", Repo: "go-io", Workspace: "core/go-io/task-5", Status: "completed"})
func (m *Subsystem) HandleIPCEvents(_ *core.Core, msg core.Message) core.Result {
	switch ev := msg.(type) {
	case messages.AgentCompleted:
		m.handleAgentCompleted(ev)
	case messages.AgentStarted:
		m.handleAgentStarted(ev)
	}
	return core.Result{OK: true}
}

// Options configures the monitor polling interval.
//
//	opts := monitor.Options{Interval: 30 * time.Second}
//	mon := monitor.New(opts)
type Options struct {
	// Interval between checks (default: 2 minutes)
	Interval time.Duration
}

// New builds the monitor with a polling interval and poke channel.
//
//	mon := monitor.New(monitor.Options{Interval: 30 * time.Second})
func New(opts ...Options) *Subsystem {
	interval := 2 * time.Minute
	if len(opts) > 0 && opts[0].Interval > 0 {
		interval = opts[0].Interval
	}
	// Override via env for debugging
	if envInterval := core.Env("MONITOR_INTERVAL"); envInterval != "" {
		if d, err := time.ParseDuration(envInterval); err == nil {
			interval = d
		}
	}
	return &Subsystem{
		interval:      interval,
		poke:          make(chan struct{}, 1),
		seenCompleted: make(map[string]bool),
		seenRunning:   make(map[string]bool),
	}
}

// debugChannel logs a debug message.
func (m *Subsystem) debugChannel(msg string) {
	core.Debug(msg)
}

// Name keeps the monitor address stable for MCP and core.WithService lookups.
//
//	name := mon.Name() // "monitor"
func (m *Subsystem) Name() string { return "monitor" }

// RegisterTools publishes the monitor status resource on an MCP server.
//
//	mon.RegisterTools(server)
func (m *Subsystem) RegisterTools(server *mcp.Server) {
	m.server = server

	// Register a resource that clients can read for current status
	server.AddResource(&mcp.Resource{
		Name:        "Agent Status",
		URI:         "status://agents",
		Description: "Current status of all agent workspaces",
		MIMEType:    "application/json",
	}, m.agentStatusResource)
}

// Start launches the background polling loop.
//
//	mon.Start(ctx)
func (m *Subsystem) Start(ctx context.Context) {
	monCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel

	core.Info("monitor: started (interval=%s)", m.interval)

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.loop(monCtx)
	}()
}

// OnStartup starts the monitor when Core starts the service lifecycle.
//
//	r := mon.OnStartup(context.Background())
//	core.Println(r.OK)
func (m *Subsystem) OnStartup(ctx context.Context) core.Result {
	m.Start(ctx)
	return core.Result{OK: true}
}

// OnShutdown stops the monitor through the Core lifecycle hook.
//
//	r := mon.OnShutdown(context.Background())
//	core.Println(r.OK)
func (m *Subsystem) OnShutdown(ctx context.Context) core.Result {
	_ = m.Shutdown(ctx)
	return core.Result{OK: true}
}

// Shutdown cancels the monitor loop and waits for the goroutine to exit.
//
//	_ = mon.Shutdown(ctx)
func (m *Subsystem) Shutdown(_ context.Context) error {
	if m.cancel != nil {
		m.cancel()
	}
	m.wg.Wait()
	return nil
}

// Poke asks the loop to run a check immediately instead of waiting for the ticker.
//
//	mon.Poke()
func (m *Subsystem) Poke() {
	select {
	case m.poke <- struct{}{}:
	default:
	}
}

// checkIdleAfterDelay waits briefly then checks if the fleet is genuinely idle.
// Only emits queue.drained when there are truly zero running or queued agents,
// verified by checking managed processes are alive, not just trusting status files.
func (m *Subsystem) checkIdleAfterDelay() {
	time.Sleep(5 * time.Second) // wait for queue drain to fill slots
	if m.ServiceRuntime == nil {
		return
	}

	running, queued := m.countLiveWorkspaces()
	if running == 0 && queued == 0 {
		m.Core().ACTION(messages.QueueDrained{Completed: 0})
	}
}

// countLiveWorkspaces counts workspaces that are genuinely active.
// For "running" status, verifies the managed process is still alive.
func (m *Subsystem) countLiveWorkspaces() (running, queued int) {
	var runtime *core.Core
	if m.ServiceRuntime != nil {
		runtime = m.Core()
	}
	for _, path := range agentic.WorkspaceStatusPaths() {
		wsDir := core.PathDir(path)
		r := agentic.ReadStatusResult(wsDir)
		if !r.OK {
			continue
		}
		st, ok := r.Value.(*agentic.WorkspaceStatus)
		if !ok || st == nil {
			continue
		}
		switch st.Status {
		case "running":
			if st.PID > 0 && processAlive(runtime, st.ProcessID, st.PID) {
				running++
			}
		case "queued":
			queued++
		}
	}
	return
}

// processAlive checks whether a managed process is still running.
func processAlive(c *core.Core, processID string, pid int) bool {
	return agentic.ProcessAlive(c, processID, pid)
}

func (m *Subsystem) loop(ctx context.Context) {
	// Initial check after short delay (let server fully start)
	select {
	case <-ctx.Done():
		return
	case <-time.After(5 * time.Second):
	}

	// Initialise sync timestamp to now (don't pull everything on first run)
	m.initSyncTimestamp()

	// Run first check immediately
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
	var messages []string

	// Check agent completions
	if msg := m.checkCompletions(); msg != "" {
		messages = append(messages, msg)
	}

	// Harvest completed workspaces — push branches, check for binaries
	if msg := m.harvestCompleted(); msg != "" {
		messages = append(messages, msg)
	}

	// Check inbox
	if msg := m.checkInbox(); msg != "" {
		messages = append(messages, msg)
	}

	// Sync repos from other agents' changes
	if msg := m.syncRepos(); msg != "" {
		messages = append(messages, msg)
	}

	// Only notify if there's something new
	if len(messages) == 0 {
		return
	}

	combined := core.Join("\n", messages...)
	m.notify(ctx, combined)

	// Notify resource subscribers that agent status changed
	if m.server != nil {
		m.server.ResourceUpdated(ctx, &mcp.ResourceUpdatedNotificationParams{
			URI: "status://agents",
		})
	}
}

// checkCompletions scans workspace for newly completed agents.
// Tracks by workspace name (not count) so harvest status rewrites
// don't suppress future notifications.
func (m *Subsystem) checkCompletions() string {
	entries := agentic.WorkspaceStatusPaths()

	running := 0
	queued := 0
	completed := 0
	var newlyCompleted []string

	m.mu.Lock()
	seeded := m.completionsSeeded
	for _, entry := range entries {
		r := fs.Read(entry)
		if !r.OK {
			continue
		}
		entryData, ok := resultString(r)
		if !ok {
			continue
		}
		var st struct {
			Status string `json:"status"`
			Repo   string `json:"repo"`
			Agent  string `json:"agent"`
		}
		if r := core.JSONUnmarshalString(entryData, &st); !r.OK {
			continue
		}

		wsName := agentic.WorkspaceName(core.PathDir(entry))

		switch st.Status {
		case "completed":
			completed++
			if !m.seenCompleted[wsName] {
				m.seenCompleted[wsName] = true
				if seeded {
					newlyCompleted = append(newlyCompleted, core.Sprintf("%s (%s)", st.Repo, st.Agent))
				}
			}
		case "running":
			running++
			if !m.seenRunning[wsName] && seeded {
				m.seenRunning[wsName] = true
				// No individual start notification — too noisy
			}
		case "queued":
			queued++
		case "blocked", "failed":
			if !m.seenCompleted[wsName] {
				m.seenCompleted[wsName] = true
				if seeded {
					newlyCompleted = append(newlyCompleted, core.Sprintf("%s (%s) [%s]", st.Repo, st.Agent, st.Status))
				}
			}
		}
	}
	m.lastCompletedCount = completed
	m.completionsSeeded = true
	m.mu.Unlock()

	if len(newlyCompleted) == 0 {
		return ""
	}

	// Only emit queue.drained when genuinely empty — verified by live process checks
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

// checkInbox checks for unread messages.
func (m *Subsystem) checkInbox() string {
	apiKeyStr := monitorBrainKey()
	if apiKeyStr == "" {
		return ""
	}

	// Call the API to check inbox
	apiURL := monitorAPIURL()
	inboxURL := core.Concat(apiURL, "/v1/messages/inbox?agent=", core.Replace(agentic.AgentName(), " ", "%20"))
	hr := agentic.HTTPGet(context.Background(), inboxURL, core.Trim(apiKeyStr), "Bearer")
	if !hr.OK {
		return ""
	}

	var resp struct {
		Data []struct {
			ID      int    `json:"id"`
			Read    bool   `json:"read"`
			From    string `json:"from"`
			Subject string `json:"subject"`
			Content string `json:"content"`
		} `json:"data"`
	}
	if r := core.JSONUnmarshalString(hr.Value.(string), &resp); !r.OK {
		m.debugChannel("checkInbox: failed to decode response")
		return ""
	}

	// Find max ID, count unread, collect new messages
	maxID := 0
	unread := 0

	m.mu.Lock()
	prevMaxID := m.lastInboxMaxID
	seeded := m.inboxSeeded
	m.mu.Unlock()

	type newMessage struct {
		ID      int    `json:"id"`
		From    string `json:"from"`
		Subject string `json:"subject"`
		Content string `json:"content"`
	}
	var newMessages []newMessage

	for _, msg := range resp.Data {
		if msg.ID > maxID {
			maxID = msg.ID
		}
		if !msg.Read {
			unread++
		}
		// Collect messages newer than what we've seen
		if msg.ID > prevMaxID {
			newMessages = append(newMessages, newMessage{
				ID:      msg.ID,
				From:    msg.From,
				Subject: msg.Subject,
				Content: msg.Content,
			})
		}
	}

	m.mu.Lock()
	m.lastInboxMaxID = maxID
	m.inboxSeeded = true
	m.mu.Unlock()

	// First check after startup: seed, don't fire
	if !seeded {
		return ""
	}

	// Only fire if there are new messages (higher ID than last seen)
	if maxID <= prevMaxID || len(newMessages) == 0 {
		return ""
	}

	// Push each message as a channel event so it lands in the session
	if m.ServiceRuntime != nil {
		if notifier, ok := core.ServiceFor[channelSender](m.Core(), "mcp"); ok {
			for _, msg := range newMessages {
				notifier.ChannelSend(context.Background(), "inbox.message", map[string]any{
					"from":    msg.From,
					"subject": msg.Subject,
					"content": msg.Content,
				})
			}
		}
	}

	return core.Sprintf("%d unread message(s) in inbox", unread)
}

// notify sends a log notification to all connected MCP sessions.
func (m *Subsystem) notify(ctx context.Context, message string) {
	if m.server == nil {
		return
	}

	// Use the server's session list to broadcast
	for ss := range m.server.Sessions() {
		ss.Log(ctx, &mcp.LoggingMessageParams{
			Level:  "info",
			Logger: "monitor",
			Data:   message,
		})
	}
}

// agentStatusResource returns current workspace status as a JSON resource.
func (m *Subsystem) agentStatusResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	entries := agentic.WorkspaceStatusPaths()

	type wsInfo struct {
		Name   string `json:"name"`
		Status string `json:"status"`
		Repo   string `json:"repo"`
		Agent  string `json:"agent"`
		PRURL  string `json:"pr_url,omitempty"`
	}

	var workspaces []wsInfo
	for _, entry := range entries {
		r := fs.Read(entry)
		if !r.OK {
			continue
		}
		entryData, ok := resultString(r)
		if !ok {
			continue
		}
		var st struct {
			Status string `json:"status"`
			Repo   string `json:"repo"`
			Agent  string `json:"agent"`
			PRURL  string `json:"pr_url"`
		}
		if r := core.JSONUnmarshalString(entryData, &st); !r.OK {
			continue
		}
		workspaces = append(workspaces, wsInfo{
			Name:   agentic.WorkspaceName(core.PathDir(entry)),
			Status: st.Status,
			Repo:   st.Repo,
			Agent:  st.Agent,
			PRURL:  st.PRURL,
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
