// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// --- UnmarshalYAML for ConcurrencyLimit ---

func TestConcurrencyLimit_Good_IntForm(t *testing.T) {
	var cfg struct {
		Limit ConcurrencyLimit `yaml:"limit"`
	}
	err := yaml.Unmarshal([]byte("limit: 3"), &cfg)
	require.NoError(t, err)
	assert.Equal(t, 3, cfg.Limit.Total)
	assert.Nil(t, cfg.Limit.Models)
}

func TestConcurrencyLimit_Good_MapForm(t *testing.T) {
	data := `limit:
  total: 2
  gpt-5.4: 1
  gpt-5.3-codex-spark: 1`

	var cfg struct {
		Limit ConcurrencyLimit `yaml:"limit"`
	}
	err := yaml.Unmarshal([]byte(data), &cfg)
	require.NoError(t, err)
	assert.Equal(t, 2, cfg.Limit.Total)
	assert.Equal(t, 1, cfg.Limit.Models["gpt-5.4"])
	assert.Equal(t, 1, cfg.Limit.Models["gpt-5.3-codex-spark"])
}

func TestConcurrencyLimit_Good_MapNoTotal(t *testing.T) {
	data := `limit:
  flash: 2
  pro: 1`

	var cfg struct {
		Limit ConcurrencyLimit `yaml:"limit"`
	}
	err := yaml.Unmarshal([]byte(data), &cfg)
	require.NoError(t, err)
	assert.Equal(t, 0, cfg.Limit.Total)
	assert.Equal(t, 2, cfg.Limit.Models["flash"])
}

func TestConcurrencyLimit_Good_FullConfig(t *testing.T) {
	data := `version: 1
concurrency:
  claude: 1
  codex:
    total: 2
    gpt-5.4: 1
    gpt-5.3-codex-spark: 1
  gemini: 3`

	var cfg AgentsConfig
	err := yaml.Unmarshal([]byte(data), &cfg)
	require.NoError(t, err)
	assert.Equal(t, 1, cfg.Concurrency["claude"].Total)
	assert.Equal(t, 2, cfg.Concurrency["codex"].Total)
	assert.Equal(t, 1, cfg.Concurrency["codex"].Models["gpt-5.4"])
	assert.Equal(t, 3, cfg.Concurrency["gemini"].Total)
}

// --- delayForAgent (extended — sustained mode) ---

func TestQueue_DelayForAgent_Good_SustainedMode(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	cfg := `version: 1
concurrency:
  codex: 2
rates:
  codex:
    reset_utc: "06:00"
    sustained_delay: 120
    burst_window: 2
    burst_delay: 15`
	os.WriteFile(filepath.Join(root, "agents.yaml"), []byte(cfg), 0o644)

	s := &PrepSubsystem{
		codePath:  t.TempDir(),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	d := s.delayForAgent("codex:gpt-5.4")
	assert.True(t, d == 120*time.Second || d == 15*time.Second,
		"expected 120s or 15s, got %v", d)
}

// --- countRunningByModel ---

func TestQueue_CountRunningByModel_Good_NoWorkspaces(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	os.MkdirAll(filepath.Join(root, "workspace"), 0o755)

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	assert.Equal(t, 0, s.countRunningByModel("codex:gpt-5.4"))
}

// --- drainQueue / drainOne ---

func TestQueue_DrainQueue_Good_NoCoreFallsBackToMutex(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	os.MkdirAll(filepath.Join(root, "workspace"), 0o755)

	s := &PrepSubsystem{
		frozen:    false,
		core:      nil,
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	assert.NotPanics(t, func() { s.drainQueue() })
}

func TestQueue_DrainOne_Good_NoWorkspaces(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	os.MkdirAll(filepath.Join(root, "workspace"), 0o755)

	s := &PrepSubsystem{
		codePath:  t.TempDir(),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	assert.False(t, s.drainOne())
}

func TestQueue_DrainOne_Good_SkipsNonQueued(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := filepath.Join(root, "workspace")

	ws := filepath.Join(wsRoot, "ws-done")
	os.MkdirAll(ws, 0o755)
	st := &WorkspaceStatus{Status: "completed", Agent: "codex", Repo: "test"}
	data, _ := json.Marshal(st)
	os.WriteFile(filepath.Join(ws, "status.json"), data, 0o644)

	s := &PrepSubsystem{
		codePath:  t.TempDir(),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	assert.False(t, s.drainOne())
}

func TestQueue_DrainOne_Good_SkipsBackedOffPool(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := filepath.Join(root, "workspace")

	ws := filepath.Join(wsRoot, "ws-queued")
	os.MkdirAll(ws, 0o755)
	st := &WorkspaceStatus{Status: "queued", Agent: "codex", Repo: "test", Task: "do it"}
	data, _ := json.Marshal(st)
	os.WriteFile(filepath.Join(ws, "status.json"), data, 0o644)

	s := &PrepSubsystem{
		codePath: t.TempDir(),
		backoff: map[string]time.Time{
			"codex": time.Now().Add(1 * time.Hour),
		},
		failCount: make(map[string]int),
	}
	assert.False(t, s.drainOne())
}

// --- canDispatchAgent (Ugly — with Core.Config concurrency) ---

func TestQueue_CanDispatchAgent_Ugly(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	os.MkdirAll(filepath.Join(root, "workspace"), 0o755)

	c := core.New()
	// Set concurrency on Core.Config() — same path that Register() uses
	c.Config().Set("agents.concurrency", map[string]ConcurrencyLimit{
		"claude": {Total: 1},
		"gemini": {Total: 3},
	})

	s := &PrepSubsystem{
		core:      c,
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	// No running workspaces → should be able to dispatch
	assert.True(t, s.canDispatchAgent("claude"))
	assert.True(t, s.canDispatchAgent("gemini:flash"))
	// Agent with no limit configured → always allowed
	assert.True(t, s.canDispatchAgent("codex:gpt-5.4"))
}

// --- drainQueue (Ugly — with Core lock path) ---

func TestQueue_DrainQueue_Ugly(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	os.MkdirAll(filepath.Join(root, "workspace"), 0o755)

	c := core.New()
	s := &PrepSubsystem{
		core:      c,
		frozen:    false,
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	// Not frozen, Core is present, empty workspace → drainQueue runs the Core lock path without panic
	assert.NotPanics(t, func() { s.drainQueue() })
}

// --- CanDispatchAgent Bad ---

func TestQueue_CanDispatchAgent_Bad_AgentAtLimit(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := filepath.Join(root, "workspace")

	// Create a running workspace with a valid-looking PID (use our own PID)
	ws := filepath.Join(wsRoot, "ws-running")
	os.MkdirAll(ws, 0o755)
	st := &WorkspaceStatus{
		Status: "running",
		Agent:  "claude",
		Repo:   "go-io",
		PID:    os.Getpid(), // Our own PID so Kill(pid, 0) succeeds
	}
	data, _ := json.Marshal(st)
	os.WriteFile(filepath.Join(ws, "status.json"), data, 0o644)

	c := core.New()
	c.Config().Set("agents.concurrency", map[string]ConcurrencyLimit{
		"claude": {Total: 1},
	})

	s := &PrepSubsystem{
		core:      c,
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	// Agent at limit (1 running, limit is 1) — cannot dispatch
	assert.False(t, s.canDispatchAgent("claude"))
}

// --- CountRunningByAgent Bad/Ugly ---

func TestQueue_CountRunningByAgent_Bad_WrongAgentType(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := filepath.Join(root, "workspace")

	// Create a running workspace for a different agent type
	ws := filepath.Join(wsRoot, "ws-gemini")
	os.MkdirAll(ws, 0o755)
	st := &WorkspaceStatus{
		Status: "running",
		Agent:  "gemini",
		Repo:   "go-io",
		PID:    os.Getpid(),
	}
	data, _ := json.Marshal(st)
	os.WriteFile(filepath.Join(ws, "status.json"), data, 0o644)

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	// Counting for "claude" when only "gemini" is running → 0
	assert.Equal(t, 0, s.countRunningByAgent("claude"))
}

func TestQueue_CountRunningByAgent_Ugly_CorruptStatusJSON(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := filepath.Join(root, "workspace")

	// Create a workspace with corrupt status.json
	ws := filepath.Join(wsRoot, "ws-corrupt")
	os.MkdirAll(ws, 0o755)
	os.WriteFile(filepath.Join(ws, "status.json"), []byte("{not valid json!!!"), 0o644)

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	// Corrupt status.json → ReadStatus fails → skipped → count is 0
	assert.Equal(t, 0, s.countRunningByAgent("codex"))
}

// --- CountRunningByModel Bad/Ugly ---

func TestQueue_CountRunningByModel_Bad_NoMatchingModel(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := filepath.Join(root, "workspace")

	ws := filepath.Join(wsRoot, "ws-1")
	os.MkdirAll(ws, 0o755)
	st := &WorkspaceStatus{
		Status: "running",
		Agent:  "codex:gpt-5.4",
		Repo:   "go-io",
		PID:    os.Getpid(),
	}
	data, _ := json.Marshal(st)
	os.WriteFile(filepath.Join(ws, "status.json"), data, 0o644)

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	// Looking for a model that doesn't match any running workspace
	assert.Equal(t, 0, s.countRunningByModel("codex:gpt-5.3-codex-spark"))
}

func TestQueue_CountRunningByModel_Ugly_ModelMismatch(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := filepath.Join(root, "workspace")

	// Two workspaces, different models of same agent
	for _, ws := range []struct {
		name, agent string
	}{
		{"ws-a", "codex:gpt-5.4"},
		{"ws-b", "codex:gpt-5.3-codex-spark"},
	} {
		d := filepath.Join(wsRoot, ws.name)
		os.MkdirAll(d, 0o755)
		st := &WorkspaceStatus{Status: "running", Agent: ws.agent, Repo: "test", PID: os.Getpid()}
		data, _ := json.Marshal(st)
		os.WriteFile(filepath.Join(d, "status.json"), data, 0o644)
	}

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	// countRunningByModel does exact match on agent string
	assert.Equal(t, 1, s.countRunningByModel("codex:gpt-5.4"))
	assert.Equal(t, 1, s.countRunningByModel("codex:gpt-5.3-codex-spark"))
	assert.Equal(t, 0, s.countRunningByModel("codex:nonexistent"))
}

// --- DelayForAgent Bad/Ugly ---

func TestQueue_DelayForAgent_Bad_ZeroSustainedDelay(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	cfg := `version: 1
rates:
  codex:
    reset_utc: "06:00"
    sustained_delay: 0
    burst_window: 0
    burst_delay: 0`
	os.WriteFile(filepath.Join(root, "agents.yaml"), []byte(cfg), 0o644)

	s := &PrepSubsystem{
		codePath:  t.TempDir(),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	// sustained_delay is 0 → delayForAgent returns 0
	d := s.delayForAgent("codex:gpt-5.4")
	assert.Equal(t, time.Duration(0), d)
}

func TestQueue_DelayForAgent_Ugly_MalformedResetUTC(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	cfg := `version: 1
rates:
  codex:
    reset_utc: "not-a-time"
    sustained_delay: 60
    burst_window: 2
    burst_delay: 10`
	os.WriteFile(filepath.Join(root, "agents.yaml"), []byte(cfg), 0o644)

	s := &PrepSubsystem{
		codePath:  t.TempDir(),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	// Malformed reset_utc — strconv.Atoi fails → defaults to hour=6, min=0
	// Should still return a valid delay without panicking
	d := s.delayForAgent("codex:gpt-5.4")
	assert.True(t, d == 60*time.Second || d == 10*time.Second,
		"expected 60s or 10s, got %v", d)
}

// --- DrainOne Bad/Ugly ---

func TestQueue_DrainOne_Bad_QueuedButAtConcurrencyLimit(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := filepath.Join(root, "workspace")

	// Create a running workspace that uses our PID
	wsRunning := filepath.Join(wsRoot, "ws-running")
	os.MkdirAll(wsRunning, 0o755)
	stRunning := &WorkspaceStatus{
		Status: "running",
		Agent:  "claude",
		Repo:   "go-io",
		PID:    os.Getpid(),
	}
	dataRunning, _ := json.Marshal(stRunning)
	os.WriteFile(filepath.Join(wsRunning, "status.json"), dataRunning, 0o644)

	// Create a queued workspace
	wsQueued := filepath.Join(wsRoot, "ws-queued")
	os.MkdirAll(wsQueued, 0o755)
	stQueued := &WorkspaceStatus{Status: "queued", Agent: "claude", Repo: "go-log", Task: "do it"}
	dataQueued, _ := json.Marshal(stQueued)
	os.WriteFile(filepath.Join(wsQueued, "status.json"), dataQueued, 0o644)

	c := core.New()
	c.Config().Set("agents.concurrency", map[string]ConcurrencyLimit{
		"claude": {Total: 1},
	})

	s := &PrepSubsystem{
		core:     c,
		codePath: t.TempDir(),
		backoff:  make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	// Queued workspace exists but agent is at concurrency limit → drainOne returns false
	assert.False(t, s.drainOne())
}

func TestQueue_DrainOne_Ugly_QueuedButInBackoffWindow(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := filepath.Join(root, "workspace")

	// Create a queued workspace
	ws := filepath.Join(wsRoot, "ws-queued")
	os.MkdirAll(ws, 0o755)
	st := &WorkspaceStatus{Status: "queued", Agent: "codex", Repo: "go-io", Task: "fix it"}
	data, _ := json.Marshal(st)
	os.WriteFile(filepath.Join(ws, "status.json"), data, 0o644)

	s := &PrepSubsystem{
		codePath: t.TempDir(),
		backoff: map[string]time.Time{
			"codex": time.Now().Add(1 * time.Hour), // pool is backed off
		},
		failCount: make(map[string]int),
	}

	// Agent pool is in backoff → drainOne skips and returns false
	assert.False(t, s.drainOne())
}

// --- DrainQueue Bad ---

func TestQueue_DrainQueue_Bad_FrozenQueueDoesNothing(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := filepath.Join(root, "workspace")

	// Create a queued workspace that would normally be drained
	ws := filepath.Join(wsRoot, "ws-queued")
	os.MkdirAll(ws, 0o755)
	st := &WorkspaceStatus{Status: "queued", Agent: "codex", Repo: "go-io", Task: "fix it"}
	data, _ := json.Marshal(st)
	os.WriteFile(filepath.Join(ws, "status.json"), data, 0o644)

	s := &PrepSubsystem{
		frozen:    true, // queue is frozen
		codePath:  t.TempDir(),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	// Frozen queue returns immediately without draining
	assert.NotPanics(t, func() { s.drainQueue() })

	// Workspace should still be queued
	updated, err := ReadStatus(ws)
	require.NoError(t, err)
	assert.Equal(t, "queued", updated.Status)
}
