// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupBrainKey creates a ~/.claude/brain.key file for API auth tests.
func setupBrainKey(t *testing.T, key string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	claudeDir := filepath.Join(home, ".claude")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(claudeDir, "brain.key"), []byte(key), 0644))
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
	dir := filepath.Join(wsRoot, "workspace", name)
	require.NoError(t, os.MkdirAll(dir, 0755))
	data, _ := json.Marshal(fields)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "status.json"), data, 0644))
	return dir
}

// --- New ---

func TestNew_Good_Defaults(t *testing.T) {
	mon := New()
	assert.Equal(t, 2*time.Minute, mon.interval)
	assert.NotNil(t, mon.poke)
}

func TestNew_Good_CustomInterval(t *testing.T) {
	mon := New(Options{Interval: 30 * time.Second})
	assert.Equal(t, 30*time.Second, mon.interval)
}

func TestNew_Bad_ZeroInterval(t *testing.T) {
	mon := New(Options{Interval: 0})
	assert.Equal(t, 2*time.Minute, mon.interval)
}

func TestName_Good(t *testing.T) {
	mon := New()
	assert.Equal(t, "monitor", mon.Name())
}

// --- Poke ---

func TestPoke_Good(t *testing.T) {
	mon := New()
	mon.Poke()

	select {
	case <-mon.poke:
	default:
		t.Fatal("expected poke to send a value")
	}
}

func TestPoke_Good_NonBlocking(t *testing.T) {
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

func TestStartShutdown_Good(t *testing.T) {
	mon := New(Options{Interval: 1 * time.Hour})

	ctx := context.Background()
	mon.Start(ctx)

	err := mon.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestShutdown_Good_NilCancel(t *testing.T) {
	mon := New()
	err := mon.Shutdown(context.Background())
	assert.NoError(t, err)
}

// --- checkCompletions ---

func TestCheckCompletions_Good_NewCompletions(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	for i := 0; i < 2; i++ {
		writeWorkspaceStatus(t, wsRoot, fmt.Sprintf("ws-%d", i), map[string]any{
			"status": "completed",
			"repo":   fmt.Sprintf("repo-%d", i),
			"agent":  "claude:sonnet",
		})
	}

	mon := New()
	notifier := &mockNotifier{}
	mon.SetNotifier(notifier)

	msg := mon.checkCompletions()
	assert.Contains(t, msg, "2 agent(s) completed")

	events := notifier.Events()
	require.Len(t, events, 1)
	assert.Equal(t, "agent.complete", events[0].channel)
	eventData := events[0].data.(map[string]any)
	assert.Equal(t, 2, eventData["count"])
}

func TestCheckCompletions_Good_MixedStatuses(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	for i, status := range []string{"completed", "running", "queued"} {
		writeWorkspaceStatus(t, wsRoot, fmt.Sprintf("ws-%d", i), map[string]any{
			"status": status,
			"repo":   fmt.Sprintf("repo-%d", i),
			"agent":  "claude:sonnet",
		})
	}

	mon := New()
	notifier := &mockNotifier{}
	mon.SetNotifier(notifier)

	msg := mon.checkCompletions()
	assert.Contains(t, msg, "1 agent(s) completed")
	assert.Contains(t, msg, "1 still running")
	assert.Contains(t, msg, "1 queued")
}

func TestCheckCompletions_Good_NoNewCompletions(t *testing.T) {
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

func TestCheckCompletions_Good_EmptyWorkspace(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	require.NoError(t, os.MkdirAll(filepath.Join(wsRoot, "workspace"), 0755))

	mon := New()
	msg := mon.checkCompletions()
	assert.Equal(t, "", msg)
}

func TestCheckCompletions_Bad_InvalidJSON(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	dir := filepath.Join(wsRoot, "workspace", "ws-bad")
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "status.json"), []byte("not json"), 0644))

	mon := New()
	msg := mon.checkCompletions()
	assert.Equal(t, "", msg)
}

func TestCheckCompletions_Good_NoNotifierSet(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	writeWorkspaceStatus(t, wsRoot, "ws-0", map[string]any{
		"status": "completed", "repo": "r", "agent": "a",
	})

	mon := New()
	msg := mon.checkCompletions()
	assert.Contains(t, msg, "1 agent(s) completed")
}

// --- checkInbox ---

func TestCheckInbox_Good_UnreadMessages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/messages/inbox", r.URL.Path)
		assert.NotEmpty(t, r.URL.Query().Get("agent"))

		resp := map[string]any{
			"data": []map[string]any{
				{"read": false, "from_agent": "clotho", "subject": "task done"},
				{"read": false, "from_agent": "gemini", "subject": "review ready"},
				{"read": true, "from_agent": "clotho", "subject": "old msg"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	setupBrainKey(t, "test-key")
	t.Setenv("CORE_API_URL", srv.URL)
	t.Setenv("AGENT_NAME", "test-agent")

	mon := New()
	notifier := &mockNotifier{}
	mon.SetNotifier(notifier)

	msg := mon.checkInbox()
	assert.Contains(t, msg, "2 unread message(s) in inbox")

	events := notifier.Events()
	require.Len(t, events, 1)
	assert.Equal(t, "inbox.message", events[0].channel)
	eventData := events[0].data.(map[string]any)
	assert.Equal(t, 2, eventData["new"])
	assert.Equal(t, 2, eventData["total"])
	assert.Equal(t, "task done", eventData["subject"])
}

func TestCheckInbox_Good_NoUnread(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": []map[string]any{
				{"read": true, "from_agent": "clotho", "subject": "old"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)

	mon := New()
	msg := mon.checkInbox()
	assert.Equal(t, "", msg)
}

func TestCheckInbox_Good_SameCountNoRepeat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": []map[string]any{
				{"read": false, "from_agent": "clotho", "subject": "msg"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)

	mon := New()
	mon.checkInbox() // sets baseline

	msg := mon.checkInbox()
	assert.Equal(t, "", msg)
}

func TestCheckInbox_Bad_NoBrainKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	mon := New()
	msg := mon.checkInbox()
	assert.Equal(t, "", msg)
}

func TestCheckInbox_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)

	mon := New()
	msg := mon.checkInbox()
	assert.Equal(t, "", msg)
}

func TestCheckInbox_Bad_InvalidJSON(t *testing.T) {
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

func TestCheckInbox_Good_MultipleSameSender(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": []map[string]any{
				{"read": false, "from_agent": "clotho", "subject": "msg1"},
				{"read": false, "from_agent": "clotho", "subject": "msg2"},
				{"read": false, "from_agent": "gemini", "subject": "msg3"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)

	mon := New()
	notifier := &mockNotifier{}
	mon.SetNotifier(notifier)

	msg := mon.checkInbox()
	assert.Contains(t, msg, "3 unread message(s)")

	events := notifier.Events()
	require.Len(t, events, 1)
	eventData := events[0].data.(map[string]any)
	senders := eventData["senders"].([]string)
	found := false
	for _, s := range senders {
		if s == "clotho (2)" {
			found = true
		}
	}
	assert.True(t, found, "expected clotho (2) in senders, got %v", senders)
}

// --- check (integration of sub-checks) ---

func TestCheck_Good_CombinesMessages(t *testing.T) {
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
	assert.Equal(t, 1, len(mon.seenCompleted))
	mon.mu.Unlock()
}

func TestCheck_Good_NoMessages(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	require.NoError(t, os.MkdirAll(filepath.Join(wsRoot, "workspace"), 0755))

	home := t.TempDir()
	t.Setenv("HOME", home)

	mon := New()
	mon.check(context.Background())
}

// --- notify ---

func TestNotify_Good_NilServer(t *testing.T) {
	mon := New()
	mon.notify(context.Background(), "test message")
}

// --- loop ---

func TestLoop_Good_ImmediateCancel(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	require.NoError(t, os.MkdirAll(filepath.Join(wsRoot, "workspace"), 0755))

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

func TestLoop_Good_PokeTriggersCheck(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	require.NoError(t, os.MkdirAll(filepath.Join(wsRoot, "workspace"), 0755))

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

	// Poll until the poke-triggered check sees the completion
	require.Eventually(t, func() bool {
		mon.mu.Lock()
		defer mon.mu.Unlock()
		return len(mon.seenCompleted) == 1
	}, 5*time.Second, 50*time.Millisecond, "expected seenCompleted to have 1 entry")

	cancel()
	mon.wg.Wait()
}

// --- initSyncTimestamp ---

func TestInitSyncTimestamp_Good(t *testing.T) {
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

func TestInitSyncTimestamp_Good_NoOverwrite(t *testing.T) {
	mon := New()
	mon.lastSyncTimestamp = 12345

	mon.initSyncTimestamp()

	mon.mu.Lock()
	assert.Equal(t, int64(12345), mon.lastSyncTimestamp)
	mon.mu.Unlock()
}

// --- syncRepos ---

func TestSyncRepos_Good_NoChanges(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/agent/checkin", r.URL.Path)
		resp := CheckinResponse{Timestamp: time.Now().Unix()}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)

	mon := New()
	msg := mon.syncRepos()
	assert.Equal(t, "", msg)
}

func TestSyncRepos_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)

	mon := New()
	msg := mon.syncRepos()
	assert.Equal(t, "", msg)
}

func TestSyncRepos_Good_UpdatesTimestamp(t *testing.T) {
	newTS := time.Now().Unix() + 1000
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := CheckinResponse{Timestamp: newTS}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
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

func TestAgentStatusResource_Good(t *testing.T) {
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
	require.NoError(t, json.Unmarshal([]byte(result.Contents[0].Text), &workspaces))
	assert.Len(t, workspaces, 2)
}

func TestAgentStatusResource_Good_Empty(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	require.NoError(t, os.MkdirAll(filepath.Join(wsRoot, "workspace"), 0755))

	mon := New()
	result, err := mon.agentStatusResource(context.Background(), &mcp.ReadResourceRequest{})
	require.NoError(t, err)
	require.Len(t, result.Contents, 1)
	assert.Equal(t, "null", result.Contents[0].Text)
}

func TestAgentStatusResource_Bad_InvalidJSON(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	dir := filepath.Join(wsRoot, "workspace", "ws-bad")
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "status.json"), []byte("bad"), 0644))

	mon := New()
	result, err := mon.agentStatusResource(context.Background(), &mcp.ReadResourceRequest{})
	require.NoError(t, err)
	assert.Equal(t, "null", result.Contents[0].Text)
}

// --- syncRepos (git pull path) ---

func TestSyncRepos_Good_PullsChangedRepo(t *testing.T) {
	remoteDir := filepath.Join(t.TempDir(), "remote")
	require.NoError(t, os.MkdirAll(remoteDir, 0755))
	run(t, remoteDir, "git", "init", "--bare")

	codeDir := t.TempDir()
	repoDir := filepath.Join(codeDir, "test-repo")
	run(t, codeDir, "git", "clone", remoteDir, "test-repo")
	run(t, repoDir, "git", "checkout", "-b", "main")
	os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("# test"), 0644)
	run(t, repoDir, "git", "add", ".")
	run(t, repoDir, "git", "commit", "-m", "init")
	run(t, repoDir, "git", "push", "-u", "origin", "main")

	// Simulate another agent pushing work via a second clone
	clone2Parent := t.TempDir()
	tmpClone := filepath.Join(clone2Parent, "clone2")
	run(t, clone2Parent, "git", "clone", remoteDir, "clone2")
	run(t, tmpClone, "git", "checkout", "main")
	os.WriteFile(filepath.Join(tmpClone, "new.go"), []byte("package main\n"), 0644)
	run(t, tmpClone, "git", "add", ".")
	run(t, tmpClone, "git", "commit", "-m", "agent work")
	run(t, tmpClone, "git", "push", "origin", "main")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := CheckinResponse{
			Changed:   []ChangedRepo{{Repo: "test-repo", Branch: "main", SHA: "abc"}},
			Timestamp: time.Now().Unix() + 100,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)
	t.Setenv("CODE_PATH", codeDir)

	mon := New()
	msg := mon.syncRepos()
	assert.Contains(t, msg, "Synced 1 repo(s)")
	assert.Contains(t, msg, "test-repo")
}

func TestSyncRepos_Good_SkipsDirtyRepo(t *testing.T) {
	remoteDir := filepath.Join(t.TempDir(), "remote")
	require.NoError(t, os.MkdirAll(remoteDir, 0755))
	run(t, remoteDir, "git", "init", "--bare")

	codeDir := t.TempDir()
	repoDir := filepath.Join(codeDir, "dirty-repo")
	run(t, codeDir, "git", "clone", remoteDir, "dirty-repo")
	run(t, repoDir, "git", "checkout", "-b", "main")
	os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("# test"), 0644)
	run(t, repoDir, "git", "add", ".")
	run(t, repoDir, "git", "commit", "-m", "init")
	run(t, repoDir, "git", "push", "-u", "origin", "main")

	// Make the repo dirty
	os.WriteFile(filepath.Join(repoDir, "dirty.txt"), []byte("uncommitted"), 0644)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := CheckinResponse{
			Changed:   []ChangedRepo{{Repo: "dirty-repo", Branch: "main", SHA: "abc"}},
			Timestamp: time.Now().Unix() + 100,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)
	t.Setenv("CODE_PATH", codeDir)

	mon := New()
	msg := mon.syncRepos()
	assert.Equal(t, "", msg)
}

func TestSyncRepos_Good_SkipsNonMainBranch(t *testing.T) {
	remoteDir := filepath.Join(t.TempDir(), "remote")
	require.NoError(t, os.MkdirAll(remoteDir, 0755))
	run(t, remoteDir, "git", "init", "--bare")

	codeDir := t.TempDir()
	repoDir := filepath.Join(codeDir, "feature-repo")
	run(t, codeDir, "git", "clone", remoteDir, "feature-repo")
	run(t, repoDir, "git", "checkout", "-b", "main")
	os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("# test"), 0644)
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
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)
	t.Setenv("CODE_PATH", codeDir)

	mon := New()
	msg := mon.syncRepos()
	assert.Equal(t, "", msg)
}

func TestSyncRepos_Good_SkipsNonexistentRepo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := CheckinResponse{
			Changed:   []ChangedRepo{{Repo: "nonexistent", Branch: "main", SHA: "abc"}},
			Timestamp: time.Now().Unix() + 100,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)
	t.Setenv("CODE_PATH", t.TempDir())

	mon := New()
	msg := mon.syncRepos()
	assert.Equal(t, "", msg)
}

func TestSyncRepos_Good_UsesEnvBrainKey(t *testing.T) {
	var authHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		resp := CheckinResponse{Timestamp: time.Now().Unix()}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
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

func TestHarvestCompleted_Good_MultipleWorkspaces(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	for i := 0; i < 2; i++ {
		name := fmt.Sprintf("ws-%d", i)
		wsDir := filepath.Join(wsRoot, "workspace", name)

		sourceDir := filepath.Join(wsRoot, fmt.Sprintf("source-%d", i))
		require.NoError(t, os.MkdirAll(sourceDir, 0755))
		run(t, sourceDir, "git", "init")
		run(t, sourceDir, "git", "checkout", "-b", "main")
		os.WriteFile(filepath.Join(sourceDir, "README.md"), []byte("# test"), 0644)
		run(t, sourceDir, "git", "add", ".")
		run(t, sourceDir, "git", "commit", "-m", "init")

		require.NoError(t, os.MkdirAll(wsDir, 0755))
		run(t, wsDir, "git", "clone", sourceDir, "src")
		srcDir := filepath.Join(wsDir, "src")
		run(t, srcDir, "git", "checkout", "-b", "agent/test-task")
		os.WriteFile(filepath.Join(srcDir, "new.go"), []byte("package main\n"), 0644)
		run(t, srcDir, "git", "add", ".")
		run(t, srcDir, "git", "commit", "-m", "agent work")

		writeStatus(t, wsDir, "completed", fmt.Sprintf("repo-%d", i), "agent/test-task")
	}

	mon := New()
	notifier := &mockNotifier{}
	mon.SetNotifier(notifier)

	msg := mon.harvestCompleted()
	assert.Contains(t, msg, "Harvested:")
	assert.Contains(t, msg, "repo-0")
	assert.Contains(t, msg, "repo-1")

	events := notifier.Events()
	assert.GreaterOrEqual(t, len(events), 2)
}

func TestHarvestCompleted_Good_Empty(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	require.NoError(t, os.MkdirAll(filepath.Join(wsRoot, "workspace"), 0755))

	mon := New()
	msg := mon.harvestCompleted()
	assert.Equal(t, "", msg)
}

func TestHarvestCompleted_Good_RejectedWorkspace(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	sourceDir := filepath.Join(wsRoot, "source-rej")
	require.NoError(t, os.MkdirAll(sourceDir, 0755))
	run(t, sourceDir, "git", "init")
	run(t, sourceDir, "git", "checkout", "-b", "main")
	os.WriteFile(filepath.Join(sourceDir, "README.md"), []byte("# test"), 0644)
	run(t, sourceDir, "git", "add", ".")
	run(t, sourceDir, "git", "commit", "-m", "init")

	wsDir := filepath.Join(wsRoot, "workspace", "ws-rej")
	require.NoError(t, os.MkdirAll(wsDir, 0755))
	run(t, wsDir, "git", "clone", sourceDir, "src")
	srcDir := filepath.Join(wsDir, "src")
	run(t, srcDir, "git", "checkout", "-b", "agent/test-task")
	os.WriteFile(filepath.Join(srcDir, "new.go"), []byte("package main\n"), 0644)
	run(t, srcDir, "git", "add", ".")
	run(t, srcDir, "git", "commit", "-m", "agent work")

	// Add binary to trigger rejection
	os.WriteFile(filepath.Join(srcDir, "app.exe"), []byte("binary"), 0644)
	run(t, srcDir, "git", "add", ".")
	run(t, srcDir, "git", "commit", "-m", "add binary")

	writeStatus(t, wsDir, "completed", "rej-repo", "agent/test-task")

	mon := New()
	notifier := &mockNotifier{}
	mon.SetNotifier(notifier)

	msg := mon.harvestCompleted()
	assert.Contains(t, msg, "REJECTED")

	events := notifier.Events()
	require.Len(t, events, 1)
	assert.Equal(t, "harvest.rejected", events[0].channel)
}
