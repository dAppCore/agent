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
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"dappco.re/go/agent/pkg/agentic"
	coreio "forge.lthn.ai/core/go-io"
	coreerr "forge.lthn.ai/core/go-log"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ChannelNotifier pushes events to connected MCP sessions.
// Matches the Notifier interface in core/mcp without importing it.
type ChannelNotifier interface {
	ChannelSend(ctx context.Context, channel string, data any)
}

// Subsystem implements mcp.Subsystem for background monitoring.
type Subsystem struct {
	server   *mcp.Server
	notifier ChannelNotifier
	interval time.Duration
	cancel   context.CancelFunc
	wg       sync.WaitGroup

	// Track last seen state to only notify on changes
	seenCompleted      map[string]bool // workspace names we've already notified about
	completionsSeeded  bool            // true after first completions check
	lastInboxMaxID     int             // highest message ID seen
	inboxSeeded        bool            // true after first inbox check
	lastSyncTimestamp  int64
	mu                 sync.Mutex

	// Event-driven poke channel — dispatch goroutine sends here on completion
	poke chan struct{}
}

// SetNotifier wires up channel event broadcasting.
func (m *Subsystem) SetNotifier(n ChannelNotifier) {
	m.notifier = n
}

// Options configures the monitor.
type Options struct {
	// Interval between checks (default: 2 minutes)
	Interval time.Duration
}

// New creates a monitor subsystem.
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

func (m *Subsystem) Name() string { return "monitor" }

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

// Start begins the background monitoring loop.
// Called after the MCP server is running and sessions are active.
func (m *Subsystem) Start(ctx context.Context) {
	monCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel

	fmt.Fprintf(os.Stderr, "monitor: started (interval=%s, notifier=%v)\n", m.interval, m.notifier != nil)

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.loop(monCtx)
	}()
}

// Shutdown stops the monitoring loop.
func (m *Subsystem) Shutdown(_ context.Context) error {
	if m.cancel != nil {
		m.cancel()
	}
	m.wg.Wait()
	return nil
}

// Poke triggers an immediate check cycle. Non-blocking — if a poke is already
// pending it's a no-op. Call this from dispatch when an agent completes.
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

	combined := strings.Join(messages, "\n")
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
	entries, err := filepath.Glob(filepath.Join(wsRoot, "*/status.json"))
	if err != nil {
		return ""
	}

	running := 0
	queued := 0
	var newlyCompleted []string

	m.mu.Lock()
	seeded := m.completionsSeeded
	for _, entry := range entries {
		data, err := coreio.Local.Read(entry)
		if err != nil {
			continue
		}
		var st struct {
			Status string `json:"status"`
			Repo   string `json:"repo"`
			Agent  string `json:"agent"`
		}
		if json.Unmarshal([]byte(data), &st) != nil {
			continue
		}

		wsName := filepath.Base(filepath.Dir(entry))

		switch st.Status {
		case "completed":
			if !m.seenCompleted[wsName] {
				m.seenCompleted[wsName] = true
				if seeded {
					newlyCompleted = append(newlyCompleted, fmt.Sprintf("%s (%s)", st.Repo, st.Agent))
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
					newlyCompleted = append(newlyCompleted, fmt.Sprintf("%s (%s) [%s]", st.Repo, st.Agent, st.Status))
				}
			}
		}
	}
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

	msg := fmt.Sprintf("%d agent(s) completed", len(newlyCompleted))
	if running > 0 {
		msg += fmt.Sprintf(", %d still running", running)
	}
	if queued > 0 {
		msg += fmt.Sprintf(", %d queued", queued)
	}
	return msg
}

// checkInbox checks for unread messages.
func (m *Subsystem) checkInbox() string {
	apiKeyStr := os.Getenv("CORE_BRAIN_KEY")
	if apiKeyStr == "" {
		home, _ := os.UserHomeDir()
		keyFile := filepath.Join(home, ".claude", "brain.key")
		data, err := coreio.Local.Read(keyFile)
		if err != nil {
			return ""
		}
		apiKeyStr = data
	}

	// Call the API to check inbox
	apiURL := os.Getenv("CORE_API_URL")
	if apiURL == "" {
		apiURL = "https://api.lthn.sh"
	}
	req, err := http.NewRequest("GET", apiURL+"/v1/messages/inbox?agent="+url.QueryEscape(agentic.AgentName()), nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKeyStr))

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
		} `json:"data"`
	}
	if json.NewDecoder(httpResp.Body).Decode(&resp) != nil {
		m.debugChannel("checkInbox: failed to decode response")
		return ""
	}

	// Find max ID and count unread
	maxID := 0
	unread := 0
	senders := make(map[string]int)
	latestSubject := ""
	for _, msg := range resp.Data {
		if msg.ID > maxID {
			maxID = msg.ID
		}
		if !msg.Read {
			unread++
			if msg.From != "" {
				senders[msg.From]++
			}
			if latestSubject == "" {
				latestSubject = msg.Subject
			}
		}
	}

	m.mu.Lock()
	prevMaxID := m.lastInboxMaxID
	seeded := m.inboxSeeded
	m.lastInboxMaxID = maxID
	m.inboxSeeded = true
	m.mu.Unlock()

	// First check after startup: seed, don't fire
	if !seeded {
		return ""
	}

	// Only fire if there are new messages (higher ID than last seen)
	if maxID <= prevMaxID || unread == 0 {
		return ""
	}

	// Write marker file for the PostToolUse hook to pick up
	var senderList []string
	for s, count := range senders {
		if count > 1 {
			senderList = append(senderList, fmt.Sprintf("%s (%d)", s, count))
		} else {
			senderList = append(senderList, s)
		}
	}
	// Push channel event for new messages
	newCount := maxID - prevMaxID
	if m.notifier != nil {
		m.notifier.ChannelSend(context.Background(), "inbox.message", map[string]any{
			"new":     newCount,
			"total":   unread,
			"senders": senderList,
			"subject": latestSubject,
		})
	}

	return fmt.Sprintf("%d unread message(s) in inbox", unread)
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
	entries, err := filepath.Glob(filepath.Join(wsRoot, "*/status.json"))
	if err != nil {
		return nil, coreerr.E("monitor.agentStatus", "failed to scan workspaces", err)
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
		data, err := coreio.Local.Read(entry)
		if err != nil {
			continue
		}
		var st struct {
			Status string `json:"status"`
			Repo   string `json:"repo"`
			Agent  string `json:"agent"`
			PRURL  string `json:"pr_url"`
		}
		if json.Unmarshal([]byte(data), &st) != nil {
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

	result, _ := json.Marshal(workspaces)
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
