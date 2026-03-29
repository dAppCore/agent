// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"dappco.re/go/agent/pkg/agentic"
	"dappco.re/go/agent/pkg/messages"
	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type capturedChannelEvent struct {
	Channel string
	Data    any
}

type captureNotifier struct {
	events []capturedChannelEvent
}

func (n *captureNotifier) ChannelSend(_ context.Context, channel string, data any) {
	n.events = append(n.events, capturedChannelEvent{Channel: channel, Data: data})
}

// setupBrainKey creates a ~/.claude/brain.key file for API auth tests.
func setupBrainKey(t *testing.T, key string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	claudeDir := core.JoinPath(home, ".claude")
	fs.EnsureDir(claudeDir)
	fs.Write(core.JoinPath(claudeDir, "brain.key"), key)
}

// setupAPIEnv sets up brain key, CORE_API_URL, and AGENT_NAME for API tests.
func setupAPIEnv(t *testing.T, apiURL string) {
	t.Helper()
	setupBrainKey(t, "key")
	t.Setenv("CORE_API_URL", apiURL)
	t.Setenv("AGENT_NAME", "test-agent")
}

// writeWorkspaceStatus creates a workspace directory with a status.json file
// under the given root. Returns the workspace directory path.
func writeWorkspaceStatus(t *testing.T, wsRoot, name string, fields map[string]any) string {
	t.Helper()
	dir := core.JoinPath(wsRoot, "workspace", name)
	fs.EnsureDir(dir)
	fs.Write(core.JoinPath(dir, "status.json"), core.JSONMarshalString(fields))
	return dir
}

// --- New ---

func TestMonitor_New_Good_Defaults(t *testing.T) {
	t.Setenv("MONITOR_INTERVAL", "")
	mon := New()
	assert.Equal(t, 2*time.Minute, mon.interval)
	assert.NotNil(t, mon.poke)
}

func TestMonitor_New_Good_CustomInterval(t *testing.T) {
	mon := New(Options{Interval: 30 * time.Second})
	assert.Equal(t, 30*time.Second, mon.interval)
}

func TestMonitor_New_Bad_ZeroInterval(t *testing.T) {
	t.Setenv("MONITOR_INTERVAL", "")
	mon := New(Options{Interval: 0})
	assert.Equal(t, 2*time.Minute, mon.interval)
}

func TestMonitor_Name_Good(t *testing.T) {
	mon := New()
	assert.Equal(t, "monitor", mon.Name())
}

// --- Poke ---

func TestMonitor_Poke_Good(t *testing.T) {
	mon := New()
	mon.Poke()

	select {
	case <-mon.poke:
	default:
		t.Fatal("expected poke to send a value")
	}
}

func TestMonitor_Poke_Good_NonBlocking(t *testing.T) {
	mon := New()
	mon.Poke()
	mon.Poke() // second poke should be a no-op, not block

	select {
	case <-mon.poke:
	default:
		t.Fatal("expected at least one poke")
	}

	select {
	case <-mon.poke:
		t.Fatal("expected channel to be empty after drain")
	default:
	}
}

// --- Start / Shutdown ---

func TestMonitor_StartShutdown_Good(t *testing.T) {
	mon := New(Options{Interval: 1 * time.Hour})

	ctx := context.Background()
	mon.Start(ctx)

	err := mon.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestMonitor_Shutdown_Good_NilCancel(t *testing.T) {
	mon := New()
	err := mon.Shutdown(context.Background())
	assert.NoError(t, err)
}

// --- checkCompletions ---

func TestMonitor_CheckCompletions_Good_NewCompletions(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

	// Create Core with IPC handler to capture QueueDrained messages
	var drainEvents []messages.QueueDrained
	c := core.New()
	c.RegisterAction(func(_ *core.Core, msg core.Message) core.Result {
		if ev, ok := msg.(messages.QueueDrained); ok {
			drainEvents = append(drainEvents, ev)
		}
		return core.Result{OK: true}
	})

	mon := New()
	mon.ServiceRuntime = core.NewServiceRuntime(c, MonitorOptions{})
	assert.Equal(t, "", mon.checkCompletions())

	for i := 0; i < 2; i++ {
		writeWorkspaceStatus(t, wsRoot, fmt.Sprintf("ws-%d", i), map[string]any{
			"status": "completed",
			"repo":   fmt.Sprintf("repo-%d", i),
			"agent":  "claude:sonnet",
		})
	}

	msg := mon.checkCompletions()
	assert.Contains(t, msg, "2 agent(s) completed")

	// checkCompletions emits QueueDrained via c.ACTION() when running=0 and queued=0
	require.Len(t, drainEvents, 1)
	assert.Equal(t, 2, drainEvents[0].Completed)
}

func TestMonitor_CheckCompletions_Good_MixedStatuses(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

	mon := New()
	assert.Equal(t, "", mon.checkCompletions())

	for i, status := range []string{"completed", "running", "queued"} {
		writeWorkspaceStatus(t, wsRoot, fmt.Sprintf("ws-%d", i), map[string]any{
			"status": status,
			"repo":   fmt.Sprintf("repo-%d", i),
			"agent":  "claude:sonnet",
		})
	}

	msg := mon.checkCompletions()
	assert.Contains(t, msg, "1 agent(s) completed")
	assert.Contains(t, msg, "1 still running")
	assert.Contains(t, msg, "1 queued")
}

func TestMonitor_CheckCompletions_Good_NoNewCompletions(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	writeWorkspaceStatus(t, wsRoot, "ws-0", map[string]any{
		"status": "completed", "repo": "r", "agent": "a",
	})

	mon := New()
	mon.checkCompletions() // sets baseline

	msg := mon.checkCompletions()
	assert.Equal(t, "", msg)
}

func TestMonitor_CheckCompletions_Good_EmptyWorkspace(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

	mon := New()
	msg := mon.checkCompletions()
	assert.Equal(t, "", msg)
}

func TestMonitor_CheckCompletions_Bad_InvalidJSON(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	dir := core.JoinPath(wsRoot, "workspace", "ws-bad")
	fs.EnsureDir(dir)
	fs.Write(core.JoinPath(dir, "status.json"), "not json")

	mon := New()
	msg := mon.checkCompletions()
	assert.Equal(t, "", msg)
}

func TestMonitor_CheckCompletions_Good_NilRuntime(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

	mon := New()
	assert.Equal(t, "", mon.checkCompletions())

	writeWorkspaceStatus(t, wsRoot, "ws-0", map[string]any{
		"status": "completed", "repo": "r", "agent": "a",
	})

	msg := mon.checkCompletions()
	assert.Contains(t, msg, "1 agent(s) completed")
}

// --- checkInbox ---

func TestMonitor_CheckInbox_Good_UnreadMessages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/messages/inbox", r.URL.Path)
		assert.NotEmpty(t, r.URL.Query().Get("agent"))

		resp := map[string]any{
			"data": []map[string]any{
				{"id": 3, "read": false, "from": "clotho", "subject": "task done"},
				{"id": 2, "read": false, "from": "gemini", "subject": "review ready"},
				{"id": 1, "read": true, "from": "clotho", "subject": "old msg"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(resp)))
	}))
	defer srv.Close()

	setupBrainKey(t, "test-key")
	t.Setenv("CORE_API_URL", srv.URL)
	t.Setenv("AGENT_NAME", "test-agent")

	// Create Core with an MCP notifier to capture channel events.
	c := core.New()
	notifier := &captureNotifier{}
	require.True(t, c.RegisterService("mcp", notifier).OK)

	mon := New()
	mon.ServiceRuntime = core.NewServiceRuntime(c, MonitorOptions{})
	mon.inboxSeeded = true

	msg := mon.checkInbox()
	assert.Contains(t, msg, "2 unread message(s) in inbox")

	// 3 new messages (ids 1,2,3 all > prevMaxID=0), but only 2 unread.
	// Monitor sends one channel notification per new message.
	require.Len(t, notifier.events, 3)
	data := notifier.events[0].Data.(map[string]any)
	assert.Equal(t, "clotho", data["from"])
}

func TestMonitor_CheckInbox_Good_NoUnread(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": []map[string]any{
				{"id": 1, "read": true, "from": "clotho", "subject": "old"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(resp)))
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)

	mon := New()
	msg := mon.checkInbox()
	assert.Equal(t, "", msg)
}

func TestMonitor_CheckInbox_Good_SameCountNoRepeat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": []map[string]any{
				{"id": 1, "read": false, "from": "clotho", "subject": "msg"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(resp)))
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)

	mon := New()
	mon.checkInbox() // sets baseline

	msg := mon.checkInbox()
	assert.Equal(t, "", msg)
}

func TestMonitor_CheckInbox_Bad_NoBrainKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	mon := New()
	msg := mon.checkInbox()
	assert.Equal(t, "", msg)
}

func TestMonitor_CheckInbox_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)

	mon := New()
	msg := mon.checkInbox()
	assert.Equal(t, "", msg)
}

func TestMonitor_CheckInbox_Bad_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)

	mon := New()
	msg := mon.checkInbox()
	assert.Equal(t, "", msg)
}

func TestMonitor_CheckInbox_Good_MultipleSameSender(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": []map[string]any{
				{"id": 3, "read": false, "from": "clotho", "subject": "msg1"},
				{"id": 2, "read": false, "from": "clotho", "subject": "msg2"},
				{"id": 1, "read": false, "from": "gemini", "subject": "msg3"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(resp)))
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)

	// Create Core with an MCP notifier to capture channel events.
	c := core.New()
	notifier := &captureNotifier{}
	require.True(t, c.RegisterService("mcp", notifier).OK)

	mon := New()
	mon.ServiceRuntime = core.NewServiceRuntime(c, MonitorOptions{})
	mon.inboxSeeded = true

	msg := mon.checkInbox()
	assert.Contains(t, msg, "3 unread message(s)")

	require.True(t, len(notifier.events) > 0, "should capture at least one channel event")
}

// --- check (integration of sub-checks) ---

func TestMonitor_Check_Good_CombinesMessages(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	writeWorkspaceStatus(t, wsRoot, "ws-0", map[string]any{
		"status": "completed", "repo": "r", "agent": "a",
	})

	home := t.TempDir()
	t.Setenv("HOME", home)

	mon := New()
	mon.check(context.Background())

	mon.mu.Lock()
	assert.True(t, mon.completionsSeeded)
	assert.True(t, mon.seenCompleted["ws-0"])
	mon.mu.Unlock()
}

func TestMonitor_Check_Good_NoMessages(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

	home := t.TempDir()
	t.Setenv("HOME", home)

	mon := New()
	mon.check(context.Background())
}

// --- notify ---

func TestMonitor_Notify_Good_NilServer(t *testing.T) {
	mon := New()
	mon.notify(context.Background(), "test message")
}

// --- loop ---

func TestMonitor_Loop_Good_ImmediateCancel(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

	home := t.TempDir()
	t.Setenv("HOME", home)

	mon := New(Options{Interval: 1 * time.Hour})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		mon.loop(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("loop did not exit after context cancellation")
	}
}

func TestMonitor_Loop_Good_PokeTriggersCheck(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

	home := t.TempDir()
	t.Setenv("HOME", home)

	mon := New(Options{Interval: 1 * time.Hour})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mon.wg.Add(1)
	go func() {
		defer mon.wg.Done()
		mon.loop(ctx)
	}()

	// Wait for initial delay (5s) + first check + scheduler overhead
	time.Sleep(7 * time.Second)

	writeWorkspaceStatus(t, wsRoot, "ws-poke", map[string]any{
		"status": "completed", "repo": "poke-repo", "agent": "a",
	})

	mon.Poke()

	// Poll until the poke-triggered check updates the count
	require.Eventually(t, func() bool {
		mon.mu.Lock()
		defer mon.mu.Unlock()
		return mon.seenCompleted["ws-poke"]
	}, 5*time.Second, 50*time.Millisecond, "expected ws-poke completion to be recorded")

	cancel()
	mon.wg.Wait()
}

// --- initSyncTimestamp ---

func TestMonitor_InitSyncTimestamp_Good(t *testing.T) {
	mon := New()
	assert.Equal(t, int64(0), mon.lastSyncTimestamp)

	before := time.Now().Unix()
	mon.initSyncTimestamp()
	after := time.Now().Unix()

	mon.mu.Lock()
	ts := mon.lastSyncTimestamp
	mon.mu.Unlock()

	assert.GreaterOrEqual(t, ts, before)
	assert.LessOrEqual(t, ts, after)
}

func TestMonitor_InitSyncTimestamp_Good_NoOverwrite(t *testing.T) {
	mon := New()
	mon.lastSyncTimestamp = 12345

	mon.initSyncTimestamp()

	mon.mu.Lock()
	assert.Equal(t, int64(12345), mon.lastSyncTimestamp)
	mon.mu.Unlock()
}

// --- syncRepos ---

func TestMonitor_SyncRepos_Good_NoChanges(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/agent/checkin", r.URL.Path)
		resp := CheckinResponse{Timestamp: time.Now().Unix()}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(resp)))
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)

	mon := New()
	msg := mon.syncRepos()
	assert.Equal(t, "", msg)
}

func TestMonitor_SyncRepos_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)

	mon := New()
	msg := mon.syncRepos()
	assert.Equal(t, "", msg)
}

func TestMonitor_SyncRepos_Good_UpdatesTimestamp(t *testing.T) {
	newTS := time.Now().Unix() + 1000
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := CheckinResponse{Timestamp: newTS}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(resp)))
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)

	mon := New()
	mon.syncRepos()

	mon.mu.Lock()
	assert.Equal(t, newTS, mon.lastSyncTimestamp)
	mon.mu.Unlock()
}

// --- agentStatusResource ---

func TestMonitor_AgentStatusResource_Good(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	for i, status := range []string{"completed", "running"} {
		writeWorkspaceStatus(t, wsRoot, fmt.Sprintf("ws-%d", i), map[string]any{
			"status": status,
			"repo":   fmt.Sprintf("repo-%d", i),
			"agent":  "claude:sonnet",
		})
	}

	mon := New()
	result, err := mon.agentStatusResource(context.Background(), &mcp.ReadResourceRequest{})
	require.NoError(t, err)
	require.Len(t, result.Contents, 1)
	assert.Equal(t, "status://agents", result.Contents[0].URI)

	var workspaces []map[string]any
	require.True(t, core.JSONUnmarshalString(result.Contents[0].Text, &workspaces).OK)
	assert.Len(t, workspaces, 2)
}

func TestMonitor_AgentStatusResource_Good_Empty(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

	mon := New()
	result, err := mon.agentStatusResource(context.Background(), &mcp.ReadResourceRequest{})
	require.NoError(t, err)
	require.Len(t, result.Contents, 1)
	assert.Equal(t, "null", result.Contents[0].Text)
}

func TestMonitor_AgentStatusResource_Bad_InvalidJSON(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	dir := core.JoinPath(wsRoot, "workspace", "ws-bad")
	fs.EnsureDir(dir)
	fs.Write(core.JoinPath(dir, "status.json"), "bad")

	mon := New()
	result, err := mon.agentStatusResource(context.Background(), &mcp.ReadResourceRequest{})
	require.NoError(t, err)
	assert.Equal(t, "null", result.Contents[0].Text)
}

// --- syncRepos (git pull path) ---

func TestMonitor_SyncRepos_Good_PullsChangedRepo(t *testing.T) {
	remoteDir := core.JoinPath(t.TempDir(), "remote")
	fs.EnsureDir(remoteDir)
	run(t, remoteDir, "git", "init", "--bare")

	codeDir := t.TempDir()
	repoDir := core.JoinPath(codeDir, "test-repo")
	run(t, codeDir, "git", "clone", remoteDir, "test-repo")
	run(t, repoDir, "git", "checkout", "-b", "main")
	fs.Write(core.JoinPath(repoDir, "README.md"), "# test")
	run(t, repoDir, "git", "add", ".")
	run(t, repoDir, "git", "commit", "-m", "init")
	run(t, repoDir, "git", "push", "-u", "origin", "main")

	// Simulate another agent pushing work via a second clone
	clone2Parent := t.TempDir()
	tmpClone := core.JoinPath(clone2Parent, "clone2")
	run(t, clone2Parent, "git", "clone", remoteDir, "clone2")
	run(t, tmpClone, "git", "checkout", "main")
	fs.Write(core.JoinPath(tmpClone, "new.go"), "package main\n")
	run(t, tmpClone, "git", "add", ".")
	run(t, tmpClone, "git", "commit", "-m", "agent work")
	run(t, tmpClone, "git", "push", "origin", "main")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := CheckinResponse{
			Changed:   []ChangedRepo{{Repo: "test-repo", Branch: "main", SHA: "abc"}},
			Timestamp: time.Now().Unix() + 100,
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(resp)))
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)
	t.Setenv("CODE_PATH", codeDir)

	mon := New()
	mon.ServiceRuntime = testMon.ServiceRuntime
	msg := mon.syncRepos()
	assert.Contains(t, msg, "Synced 1 repo(s)")
	assert.Contains(t, msg, "test-repo")
}

func TestMonitor_SyncRepos_Good_SkipsDirtyRepo(t *testing.T) {
	remoteDir := core.JoinPath(t.TempDir(), "remote")
	fs.EnsureDir(remoteDir)
	run(t, remoteDir, "git", "init", "--bare")

	codeDir := t.TempDir()
	repoDir := core.JoinPath(codeDir, "dirty-repo")
	run(t, codeDir, "git", "clone", remoteDir, "dirty-repo")
	run(t, repoDir, "git", "checkout", "-b", "main")
	fs.Write(core.JoinPath(repoDir, "README.md"), "# test")
	run(t, repoDir, "git", "add", ".")
	run(t, repoDir, "git", "commit", "-m", "init")
	run(t, repoDir, "git", "push", "-u", "origin", "main")

	// Make the repo dirty
	fs.Write(core.JoinPath(repoDir, "dirty.txt"), "uncommitted")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := CheckinResponse{
			Changed:   []ChangedRepo{{Repo: "dirty-repo", Branch: "main", SHA: "abc"}},
			Timestamp: time.Now().Unix() + 100,
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(resp)))
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)
	t.Setenv("CODE_PATH", codeDir)

	mon := New()
	mon.ServiceRuntime = testMon.ServiceRuntime
	msg := mon.syncRepos()
	assert.Equal(t, "", msg)
}

func TestMonitor_SyncRepos_Good_SkipsNonMainBranch(t *testing.T) {
	remoteDir := core.JoinPath(t.TempDir(), "remote")
	fs.EnsureDir(remoteDir)
	run(t, remoteDir, "git", "init", "--bare")

	codeDir := t.TempDir()
	repoDir := core.JoinPath(codeDir, "feature-repo")
	run(t, codeDir, "git", "clone", remoteDir, "feature-repo")
	run(t, repoDir, "git", "checkout", "-b", "main")
	fs.Write(core.JoinPath(repoDir, "README.md"), "# test")
	run(t, repoDir, "git", "add", ".")
	run(t, repoDir, "git", "commit", "-m", "init")
	run(t, repoDir, "git", "push", "-u", "origin", "main")
	run(t, repoDir, "git", "checkout", "-b", "feature/wip")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := CheckinResponse{
			Changed:   []ChangedRepo{{Repo: "feature-repo", Branch: "main", SHA: "abc"}},
			Timestamp: time.Now().Unix() + 100,
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(resp)))
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)
	t.Setenv("CODE_PATH", codeDir)

	mon := New()
	mon.ServiceRuntime = testMon.ServiceRuntime
	msg := mon.syncRepos()
	assert.Equal(t, "", msg)
}

func TestMonitor_SyncRepos_Good_SkipsNonexistentRepo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := CheckinResponse{
			Changed:   []ChangedRepo{{Repo: "nonexistent", Branch: "main", SHA: "abc"}},
			Timestamp: time.Now().Unix() + 100,
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(resp)))
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)
	t.Setenv("CODE_PATH", t.TempDir())

	mon := New()
	msg := mon.syncRepos()
	assert.Equal(t, "", msg)
}

func TestMonitor_SyncRepos_Good_UsesEnvBrainKey(t *testing.T) {
	var authHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		resp := CheckinResponse{Timestamp: time.Now().Unix()}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(resp)))
	}))
	defer srv.Close()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CORE_BRAIN_KEY", "env-key-value")
	t.Setenv("CORE_API_URL", srv.URL)
	t.Setenv("AGENT_NAME", "test-agent")

	mon := New()
	mon.syncRepos()
	assert.Equal(t, "Bearer env-key-value", authHeader)
}

// --- harvestCompleted (full path) ---

func TestMonitor_HarvestCompleted_Good_MultipleWorkspaces(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	for i := 0; i < 2; i++ {
		name := fmt.Sprintf("ws-%d", i)
		wsDir := core.JoinPath(wsRoot, "workspace", name)

		sourceDir := core.JoinPath(wsRoot, fmt.Sprintf("source-%d", i))
		fs.EnsureDir(sourceDir)
		run(t, sourceDir, "git", "init")
		run(t, sourceDir, "git", "checkout", "-b", "main")
		fs.Write(core.JoinPath(sourceDir, "README.md"), "# test")
		run(t, sourceDir, "git", "add", ".")
		run(t, sourceDir, "git", "commit", "-m", "init")

		fs.EnsureDir(wsDir)
		run(t, wsDir, "git", "clone", sourceDir, "repo")
		repoDir := core.JoinPath(wsDir, "repo")
		run(t, repoDir, "git", "checkout", "-b", "agent/test-task")
		fs.Write(core.JoinPath(repoDir, "new.go"), "package main\n")
		run(t, repoDir, "git", "add", ".")
		run(t, repoDir, "git", "commit", "-m", "agent work")

		writeStatus(t, wsDir, "completed", fmt.Sprintf("repo-%d", i), "agent/test-task")
	}

	// Create Core with IPC handler to capture HarvestComplete messages
	var harvests []messages.HarvestComplete
	c := core.New(core.WithService(agentic.ProcessRegister))
	c.ServiceStartup(context.Background(), nil)
	c.RegisterAction(func(_ *core.Core, msg core.Message) core.Result {
		if ev, ok := msg.(messages.HarvestComplete); ok {
			harvests = append(harvests, ev)
		}
		return core.Result{OK: true}
	})

	mon := New()
	mon.ServiceRuntime = core.NewServiceRuntime(c, MonitorOptions{})

	msg := mon.harvestCompleted()
	assert.Contains(t, msg, "Harvested:")
	assert.Contains(t, msg, "repo-0")
	assert.Contains(t, msg, "repo-1")

	assert.GreaterOrEqual(t, len(harvests), 2)
}

func TestMonitor_HarvestCompleted_Good_Empty(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

	mon := New()
	mon.ServiceRuntime = testMon.ServiceRuntime
	msg := mon.harvestCompleted()
	assert.Equal(t, "", msg)
}

func TestMonitor_HarvestCompleted_Good_RejectedWorkspace(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	sourceDir := core.JoinPath(wsRoot, "source-rej")
	fs.EnsureDir(sourceDir)
	run(t, sourceDir, "git", "init")
	run(t, sourceDir, "git", "checkout", "-b", "main")
	fs.Write(core.JoinPath(sourceDir, "README.md"), "# test")
	run(t, sourceDir, "git", "add", ".")
	run(t, sourceDir, "git", "commit", "-m", "init")

	wsDir := core.JoinPath(wsRoot, "workspace", "ws-rej")
	fs.EnsureDir(wsDir)
	run(t, wsDir, "git", "clone", sourceDir, "repo")
	repoDir := core.JoinPath(wsDir, "repo")
	run(t, repoDir, "git", "checkout", "-b", "agent/test-task")
	fs.Write(core.JoinPath(repoDir, "new.go"), "package main\n")
	run(t, repoDir, "git", "add", ".")
	run(t, repoDir, "git", "commit", "-m", "agent work")

	// Add binary to trigger rejection
	fs.Write(core.JoinPath(repoDir, "app.exe"), "binary")
	run(t, repoDir, "git", "add", ".")
	run(t, repoDir, "git", "commit", "-m", "add binary")

	writeStatus(t, wsDir, "completed", "rej-repo", "agent/test-task")

	// Create Core with IPC handler to capture HarvestRejected messages
	var rejections []messages.HarvestRejected
	c := core.New(core.WithService(agentic.ProcessRegister))
	c.ServiceStartup(context.Background(), nil)
	c.RegisterAction(func(_ *core.Core, msg core.Message) core.Result {
		if ev, ok := msg.(messages.HarvestRejected); ok {
			rejections = append(rejections, ev)
		}
		return core.Result{OK: true}
	})

	mon := New()
	mon.ServiceRuntime = core.NewServiceRuntime(c, MonitorOptions{})

	msg := mon.harvestCompleted()
	assert.Contains(t, msg, "REJECTED")

	require.Len(t, rejections, 1)
	assert.Contains(t, rejections[0].Reason, "binary file added")
}
