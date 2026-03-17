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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	coreerr "forge.lthn.ai/core/go-log"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Subsystem implements mcp.Subsystem for background monitoring.
type Subsystem struct {
	server   *mcp.Server
	interval time.Duration
	cancel   context.CancelFunc
	wg       sync.WaitGroup

	// Track last seen state to only notify on changes
	lastCompletedCount int
	lastInboxCount     int
	lastSyncTimestamp  int64
	mu                 sync.Mutex

	// Event-driven poke channel — dispatch goroutine sends here on completion
	poke chan struct{}
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
	return &Subsystem{
		interval: interval,
		poke:     make(chan struct{}, 1),
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
func (m *Subsystem) checkCompletions() string {
	wsRoot := workspaceRoot()
	entries, err := filepath.Glob(filepath.Join(wsRoot, "*/status.json"))
	if err != nil {
		return ""
	}

	completed := 0
	running := 0
	queued := 0
	var recentlyCompleted []string

	for _, entry := range entries {
		data, err := os.ReadFile(entry)
		if err != nil {
			continue
		}
		var st struct {
			Status string `json:"status"`
			Repo   string `json:"repo"`
			Agent  string `json:"agent"`
		}
		if json.Unmarshal(data, &st) != nil {
			continue
		}

		switch st.Status {
		case "completed":
			completed++
			recentlyCompleted = append(recentlyCompleted, fmt.Sprintf("%s (%s)", st.Repo, st.Agent))
		case "running":
			running++
		case "queued":
			queued++
		}
	}

	m.mu.Lock()
	prevCompleted := m.lastCompletedCount
	m.lastCompletedCount = completed
	m.mu.Unlock()

	newCompletions := completed - prevCompleted
	if newCompletions <= 0 {
		return ""
	}

	msg := fmt.Sprintf("%d agent(s) completed", newCompletions)
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
	home, _ := os.UserHomeDir()
	keyFile := filepath.Join(home, ".claude", "brain.key")
	apiKey, err := os.ReadFile(keyFile)
	if err != nil {
		return ""
	}

	// Call the API to check inbox
	cmd := exec.Command("curl", "-sf",
		"-H", "Authorization: Bearer "+strings.TrimSpace(string(apiKey)),
		"https://api.lthn.sh/v1/messages/inbox?agent="+agentName(),
	)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	var resp struct {
		Data []struct {
			Read    bool   `json:"read"`
			From    string `json:"from_agent"`
			Subject string `json:"subject"`
		} `json:"data"`
	}
	if json.Unmarshal(out, &resp) != nil {
		return ""
	}

	unread := 0
	senders := make(map[string]int)
	latestSubject := ""
	for _, msg := range resp.Data {
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
	prevInbox := m.lastInboxCount
	m.lastInboxCount = unread
	m.mu.Unlock()

	if unread <= 0 || unread == prevInbox {
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
	notify := fmt.Sprintf("📬 %d new message(s) from %s", unread-prevInbox, strings.Join(senderList, ", "))
	if latestSubject != "" {
		notify += fmt.Sprintf(" — \"%s\"", latestSubject)
	}
	os.WriteFile("/tmp/claude-inbox-notify", []byte(notify), 0644)

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
	wsRoot := workspaceRoot()
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
		data, err := os.ReadFile(entry)
		if err != nil {
			continue
		}
		var st struct {
			Status string `json:"status"`
			Repo   string `json:"repo"`
			Agent  string `json:"agent"`
			PRURL  string `json:"pr_url"`
		}
		if json.Unmarshal(data, &st) != nil {
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

func workspaceRoot() string {
	if root := os.Getenv("CORE_WORKSPACE"); root != "" {
		return filepath.Join(root, "workspace")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Code", ".core", "workspace")
}

func agentName() string {
	if name := os.Getenv("AGENT_NAME"); name != "" {
		return name
	}
	hostname, _ := os.Hostname()
	h := strings.ToLower(hostname)
	if strings.Contains(h, "snider") || strings.Contains(h, "studio") || strings.Contains(h, "mac") {
		return "cladius"
	}
	return "charon"
}
