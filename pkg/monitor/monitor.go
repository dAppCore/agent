// SPDX-License-Identifier: EUPL-1.2

// Package monitor provides a background subsystem that watches the ecosystem
// and pushes notifications to connected MCP clients.
//
// Checks performed on each tick:
//   - Agent completions: scans workspace for newly completed agents
//   - Repo drift: checks forge for repos with unpushed/unpulled changes
//   - Inbox: checks for unread agent messages
package monitor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"dappco.re/go/agent/pkg/agentic"
	core "dappco.re/go/core"
	coremcp "forge.lthn.ai/core/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// fs provides unrestricted filesystem access (root "/" = no sandbox).
//
//	r := fs.Read(core.Concat(wsRoot, "/", name, "/status.json"))
//	if text, ok := resultString(r); ok { json.Unmarshal([]byte(text), &st) }
var fs = agentic.LocalFs()

func workspaceStatusGlob(wsRoot string) string {
	return core.Concat(wsRoot, "/*/status.json")
}

func workspaceStatusPath(wsDir string) string {
	return core.Concat(wsDir, "/status.json")
}

func brainKeyPath(home string) string {
	return filepath.Join(home, ".claude", "brain.key")
}

func resultString(r core.Result) (string, bool) {
	value, ok := r.Value.(string)
	if !ok {
		return "", false
	}
	return value, true
}

// ChannelNotifier pushes events to connected MCP sessions.
//
//	mon.SetNotifier(notifier)
type ChannelNotifier interface {
	ChannelSend(ctx context.Context, channel string, data any)
}

// Subsystem implements mcp.Subsystem for background monitoring.
//
//	mon := monitor.New(monitor.Options{Interval: 2 * time.Minute})
//	mon.SetNotifier(notifier)
//	mon.Start(ctx)
type Subsystem struct {
	server   *mcp.Server
	notifier ChannelNotifier
	interval time.Duration
	cancel   context.CancelFunc
	wg       sync.WaitGroup

	// Track last seen state to only notify on changes
	lastCompletedCount int             // completed workspaces seen on the last scan
	seenCompleted      map[string]bool // workspace names we've already notified about
	completionsSeeded  bool            // true after first completions check
	lastInboxMaxID     int             // highest message ID seen
	inboxSeeded        bool            // true after first inbox check
	lastSyncTimestamp  int64
	mu                 sync.Mutex

	// Event-driven poke channel — dispatch goroutine sends here on completion
	poke chan struct{}
}

var _ coremcp.Subsystem = (*Subsystem)(nil)
var _ agentic.CompletionNotifier = (*Subsystem)(nil)

// SetNotifier wires up channel event broadcasting.
//
//	mon.SetNotifier(notifier)
func (m *Subsystem) SetNotifier(n ChannelNotifier) {
	m.notifier = n
}

// Options configures the monitor interval.
//
//	monitor.New(monitor.Options{Interval: 30 * time.Second})
type Options struct {
	// Interval between checks (default: 2 minutes)
	Interval time.Duration
}

// New creates a monitor subsystem.
//
//	mon := monitor.New(monitor.Options{Interval: 30 * time.Second})
func New(opts ...Options) *Subsystem {
	interval := 2 * time.Minute
	if len(opts) > 0 && opts[0].Interval > 0 {
		interval = opts[0].Interval
	}
	// Override via env for debugging
	if envInterval := os.Getenv("MONITOR_INTERVAL"); envInterval != "" {
		if d, err := time.ParseDuration(envInterval); err == nil {
			interval = d
		}
	}
	return &Subsystem{
		interval:      interval,
		poke:          make(chan struct{}, 1),
		seenCompleted: make(map[string]bool),
	}
}

// debugChannel sends a debug message via the notifier so it arrives as a channel event.
func (m *Subsystem) debugChannel(msg string) {
	if m.notifier != nil {
		m.notifier.ChannelSend(context.Background(), "monitor.debug", map[string]any{"msg": msg})
	}
}

// Name returns the subsystem identifier used by MCP registration.
//
//	mon.Name() // "monitor"
func (m *Subsystem) Name() string { return "monitor" }

// RegisterTools binds the monitor resource to an MCP server.
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

// Start begins the background monitoring loop after MCP startup.
//
//	mon.Start(ctx)
func (m *Subsystem) Start(ctx context.Context) {
	monCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel

	core.Print(os.Stderr, "monitor: started (interval=%s, notifier=%v)", m.interval, m.notifier != nil)

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.loop(monCtx)
	}()
}

// Shutdown stops the monitoring loop and waits for it to exit.
//
//	_ = mon.Shutdown(ctx)
func (m *Subsystem) Shutdown(_ context.Context) error {
	if m.cancel != nil {
		m.cancel()
	}
	m.wg.Wait()
	return nil
}

// Poke triggers an immediate check cycle.
//
//	mon.Poke()
func (m *Subsystem) Poke() {
	select {
	case m.poke <- struct{}{}:
	default:
	}
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
	wsRoot := agentic.WorkspaceRoot()
	entries, err := filepath.Glob(workspaceStatusGlob(wsRoot))
	if err != nil {
		return ""
	}

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
		if json.Unmarshal([]byte(entryData), &st) != nil {
			continue
		}

		wsName := filepath.Base(filepath.Dir(entry))

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

	// Push channel events
	if m.notifier != nil {
		m.notifier.ChannelSend(context.Background(), "agent.complete", map[string]any{
			"count":     len(newlyCompleted),
			"completed": newlyCompleted,
			"running":   running,
			"queued":    queued,
		})
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
	apiKeyStr := os.Getenv("CORE_BRAIN_KEY")
	if apiKeyStr == "" {
		home, _ := os.UserHomeDir()
		keyFile := brainKeyPath(home)
		r := fs.Read(keyFile)
		if !r.OK {
			return ""
		}
		value, ok := resultString(r)
		if !ok {
			return ""
		}
		apiKeyStr = value
	}

	// Call the API to check inbox
	apiURL := os.Getenv("CORE_API_URL")
	if apiURL == "" {
		apiURL = "https://api.lthn.sh"
	}
	req, err := http.NewRequest("GET", core.Concat(apiURL, "/v1/messages/inbox?agent=", url.QueryEscape(agentic.AgentName())), nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", core.Concat("Bearer ", core.Trim(apiKeyStr)))

	client := &http.Client{Timeout: 10 * time.Second}
	httpResp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != 200 {
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
	if json.NewDecoder(httpResp.Body).Decode(&resp) != nil {
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

	// Push channel event with full message content
	if m.notifier != nil {
		m.notifier.ChannelSend(context.Background(), "inbox.message", map[string]any{
			"new":      len(newMessages),
			"total":    unread,
			"messages": newMessages,
		})
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
	wsRoot := agentic.WorkspaceRoot()
	entries, err := filepath.Glob(workspaceStatusGlob(wsRoot))
	if err != nil {
		return nil, core.E("monitor.agentStatus", "failed to scan workspaces", err)
	}

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
		if json.Unmarshal([]byte(entryData), &st) != nil {
			continue
		}
		workspaces = append(workspaces, wsInfo{
			Name:   filepath.Base(filepath.Dir(entry)),
			Status: st.Status,
			Repo:   st.Repo,
			Agent:  st.Agent,
			PRURL:  st.PRURL,
		})
	}

	result, err := json.Marshal(workspaces)
	if err != nil {
		return nil, core.E("monitor.agentStatus", "failed to encode workspace status", err)
	}
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      "status://agents",
				MIMEType: "application/json",
				Text:     string(result),
			},
		},
	}, nil
}
