// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go/core"
	"dappco.re/go/core/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func startManagedProcess(t *testing.T, c *core.Core) *process.Process {
	t.Helper()

	r := c.Process().Start(context.Background(), core.NewOptions(
		core.Option{Key: "command", Value: "sleep"},
		core.Option{Key: "args", Value: []string{"30"}},
		core.Option{Key: "detach", Value: true},
	))
	require.True(t, r.OK)

	proc, ok := r.Value.(*process.Process)
	require.True(t, ok)
	t.Cleanup(func() {
		_ = proc.Kill()
	})
	return proc
}

// --- UnmarshalYAML for ConcurrencyLimit ---

func TestQueue_ConcurrencyLimit_Good_IntForm(t *testing.T) {
	var cfg struct {
		Limit ConcurrencyLimit `yaml:"limit"`
	}
	err := yaml.Unmarshal([]byte("limit: 3"), &cfg)
	require.NoError(t, err)
	assert.Equal(t, 3, cfg.Limit.Total)
	assert.Nil(t, cfg.Limit.Models)
}

func TestQueue_ConcurrencyLimit_Good_MapForm(t *testing.T) {
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

func TestQueue_ConcurrencyLimit_Good_MapNoTotal(t *testing.T) {
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

func TestQueue_ConcurrencyLimit_Good_FullConfig(t *testing.T) {
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
	fs.Write(core.JoinPath(root, "agents.yaml"), cfg)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	d := s.delayForAgent("codex:gpt-5.4")
	assert.True(t, d == 120*time.Second || d == 15*time.Second,
		"expected 120s or 15s, got %v", d)
}

func TestQueue_DelayForAgent_Good_MinDelayFloor(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	cfg := `version: 1
rates:
  codex:
    reset_utc: "06:00"
    min_delay: 90
    sustained_delay: 30
    burst_window: 0
    burst_delay: 0`
	require.True(t, fs.Write(core.JoinPath(root, "agents.yaml"), cfg).OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	d := s.delayForAgent("codex:gpt-5.4")
	assert.Equal(t, 90*time.Second, d)
}

func TestQueue_CanDispatchAgent_Bad_DailyLimitReached(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	require.True(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	cfg := `version: 1
rates:
  codex:
    reset_utc: "06:00"
    daily_limit: 2
    sustained_delay: 30`
	require.True(t, fs.Write(core.JoinPath(root, "agents.yaml"), cfg).OK)

	now := time.Now().UTC().Format(time.RFC3339)
	eventsPath := core.JoinPath(root, "workspace", "events.jsonl")
	require.True(t, fs.Write(eventsPath, core.Concat(
		core.JSONMarshalString(CompletionEvent{Type: "agent_started", Agent: "codex:gpt-5.4", Timestamp: now}),
		"\n",
		core.JSONMarshalString(CompletionEvent{Type: "agent_started", Agent: "codex:gpt-5.4", Timestamp: now}),
		"\n",
	)).OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	assert.False(t, s.canDispatchAgent("codex:gpt-5.4"))
	until, ok := s.backoff["codex"]
	require.True(t, ok)
	assert.True(t, until.After(time.Now()))
}

// --- countRunningByModel ---

func TestQueue_CountRunningByModel_Good_NoWorkspaces(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	fs.EnsureDir(core.JoinPath(root, "workspace"))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	assert.Equal(t, 0, s.countRunningByModel("codex:gpt-5.4"))
}

// --- drainQueue / drainOne ---

func TestQueue_DrainQueue_Good_NoCoreFallsBackToMutex(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	fs.EnsureDir(core.JoinPath(root, "workspace"))

	s := &PrepSubsystem{
		ServiceRuntime: nil,
		frozen:         false,
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	assert.NotPanics(t, func() { s.drainQueue() })
}

func TestQueue_DrainOne_Good_NoWorkspaces(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	fs.EnsureDir(core.JoinPath(root, "workspace"))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	assert.False(t, s.drainOne())
}

func TestQueue_DrainOne_Good_SkipsNonQueued(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := core.JoinPath(root, "workspace")

	ws := core.JoinPath(wsRoot, "ws-done")
	fs.EnsureDir(ws)
	st := &WorkspaceStatus{Status: "completed", Agent: "codex", Repo: "test"}
	fs.Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(st))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	assert.False(t, s.drainOne())
}

func TestQueue_DrainOne_Good_SkipsBackedOffPool(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := core.JoinPath(root, "workspace")

	ws := core.JoinPath(wsRoot, "ws-queued")
	fs.EnsureDir(ws)
	st := &WorkspaceStatus{Status: "queued", Agent: "codex", Repo: "test", Task: "do it"}
	fs.Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(st))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
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
	fs.EnsureDir(core.JoinPath(root, "workspace"))

	c := core.New()
	// Set concurrency on Core.Config() — same path that Register() uses
	c.Config().Set("agents.concurrency", map[string]ConcurrencyLimit{
		"claude": {Total: 1},
		"gemini": {Total: 3},
	})

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
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
	fs.EnsureDir(core.JoinPath(root, "workspace"))

	c := core.New()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		frozen:         false,
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Not frozen, Core is present, empty workspace → drainQueue runs the Core lock path without panic
	assert.NotPanics(t, func() { s.drainQueue() })
}

// --- CanDispatchAgent Bad ---

func TestQueue_CanDispatchAgent_Bad_AgentAtLimit(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := core.JoinPath(root, "workspace")

	c := core.New(core.WithService(ProcessRegister))
	c.ServiceStartup(context.Background(), nil)

	// Create a running workspace backed by a managed process.
	ws := core.JoinPath(wsRoot, "ws-running")
	fs.EnsureDir(ws)
	proc := startManagedProcess(t, c)
	st := &WorkspaceStatus{
		Status:    "running",
		Agent:     "claude",
		Repo:      "go-io",
		PID:       proc.Info().PID,
		ProcessID: proc.ID,
	}
	fs.Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(st))

	c.Config().Set("agents.concurrency", map[string]ConcurrencyLimit{
		"claude": {Total: 1},
	})

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Agent at limit (1 running, limit is 1) — cannot dispatch
	assert.False(t, s.canDispatchAgent("claude"))
}

// --- CountRunningByAgent Bad/Ugly ---

func TestQueue_CountRunningByAgent_Bad_WrongAgentType(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := core.JoinPath(root, "workspace")

	// Create a running workspace for a different agent type
	ws := core.JoinPath(wsRoot, "ws-gemini")
	fs.EnsureDir(ws)
	proc := startManagedProcess(t, testCore)
	st := &WorkspaceStatus{
		Status:    "running",
		Agent:     "gemini",
		Repo:      "go-io",
		PID:       proc.Info().PID,
		ProcessID: proc.ID,
	}
	fs.Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(st))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Counting for "claude" when only "gemini" is running → 0
	assert.Equal(t, 0, s.countRunningByAgent("claude"))
}

func TestQueue_CountRunningByAgent_Ugly_CorruptStatusJSON(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := core.JoinPath(root, "workspace")

	// Create a workspace with corrupt status.json
	ws := core.JoinPath(wsRoot, "ws-corrupt")
	fs.EnsureDir(ws)
	fs.Write(core.JoinPath(ws, "status.json"), "{not valid json!!!")

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Corrupt status.json → ReadStatusResult fails → skipped → count is 0
	assert.Equal(t, 0, s.countRunningByAgent("codex"))
}

// --- CountRunningByModel Bad/Ugly ---

func TestQueue_CountRunningByModel_Bad_NoMatchingModel(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := core.JoinPath(root, "workspace")

	ws := core.JoinPath(wsRoot, "ws-1")
	fs.EnsureDir(ws)
	proc := startManagedProcess(t, testCore)
	st := &WorkspaceStatus{
		Status:    "running",
		Agent:     "codex:gpt-5.4",
		Repo:      "go-io",
		PID:       proc.Info().PID,
		ProcessID: proc.ID,
	}
	fs.Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(st))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Looking for a model that doesn't match any running workspace
	assert.Equal(t, 0, s.countRunningByModel("codex:gpt-5.3-codex-spark"))
}

func TestQueue_CountRunningByModel_Ugly_ModelMismatch(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := core.JoinPath(root, "workspace")

	// Two workspaces, different models of same agent
	for _, ws := range []struct {
		name, agent string
	}{
		{"ws-a", "codex:gpt-5.4"},
		{"ws-b", "codex:gpt-5.3-codex-spark"},
	} {
		d := core.JoinPath(wsRoot, ws.name)
		fs.EnsureDir(d)
		proc := startManagedProcess(t, testCore)
		st := &WorkspaceStatus{Status: "running", Agent: ws.agent, Repo: "test", PID: proc.Info().PID, ProcessID: proc.ID}
		fs.Write(core.JoinPath(d, "status.json"), core.JSONMarshalString(st))
	}

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
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
	fs.Write(core.JoinPath(root, "agents.yaml"), cfg)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
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
	fs.Write(core.JoinPath(root, "agents.yaml"), cfg)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
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
	wsRoot := core.JoinPath(root, "workspace")

	// Create a running workspace backed by a managed process.
	wsRunning := core.JoinPath(wsRoot, "ws-running")
	fs.EnsureDir(wsRunning)
	proc := startManagedProcess(t, testCore)
	stRunning := &WorkspaceStatus{
		Status:    "running",
		Agent:     "claude",
		Repo:      "go-io",
		PID:       proc.Info().PID,
		ProcessID: proc.ID,
	}
	fs.Write(core.JoinPath(wsRunning, "status.json"), core.JSONMarshalString(stRunning))

	// Create a queued workspace
	wsQueued := core.JoinPath(wsRoot, "ws-queued")
	fs.EnsureDir(wsQueued)
	stQueued := &WorkspaceStatus{Status: "queued", Agent: "claude", Repo: "go-log", Task: "do it"}
	fs.Write(core.JoinPath(wsQueued, "status.json"), core.JSONMarshalString(stQueued))

	c := core.New()
	c.Config().Set("agents.concurrency", map[string]ConcurrencyLimit{
		"claude": {Total: 1},
	})

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Queued workspace exists but agent is at concurrency limit → drainOne returns false
	assert.False(t, s.drainOne())
}

func TestQueue_DrainOne_Ugly_QueuedButInBackoffWindow(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := core.JoinPath(root, "workspace")

	// Create a queued workspace
	ws := core.JoinPath(wsRoot, "ws-queued")
	fs.EnsureDir(ws)
	st := &WorkspaceStatus{Status: "queued", Agent: "codex", Repo: "go-io", Task: "fix it"}
	fs.Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(st))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff: map[string]time.Time{
			"codex": time.Now().Add(1 * time.Hour), // pool is backed off
		},
		failCount: make(map[string]int),
	}

	// Agent pool is in backoff → drainOne skips and returns false
	assert.False(t, s.drainOne())
}

// --- DrainQueue Bad ---

// --- UnmarshalYAML (renamed convention) ---

func TestQueue_UnmarshalYAML_Good(t *testing.T) {
	var cfg struct {
		Limit ConcurrencyLimit `yaml:"limit"`
	}
	err := yaml.Unmarshal([]byte("limit: 5"), &cfg)
	require.NoError(t, err)
	assert.Equal(t, 5, cfg.Limit.Total)
	assert.Nil(t, cfg.Limit.Models)
}

func TestQueue_UnmarshalYAML_Bad(t *testing.T) {
	var cfg struct {
		Limit ConcurrencyLimit `yaml:"limit"`
	}
	// Invalid YAML — nested map with bad types
	err := yaml.Unmarshal([]byte("limit: [1, 2, 3]"), &cfg)
	assert.Error(t, err)
}

func TestQueue_UnmarshalYAML_Ugly(t *testing.T) {
	var cfg struct {
		Limit ConcurrencyLimit `yaml:"limit"`
	}
	// Float value — YAML truncates to int, so 3.5 becomes 3
	err := yaml.Unmarshal([]byte("limit: 3.5"), &cfg)
	require.NoError(t, err)
	assert.Equal(t, 3, cfg.Limit.Total)
	assert.Nil(t, cfg.Limit.Models)
}

// --- loadAgentsConfig ---

func TestQueue_LoadAgentsConfig_Good(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	cfg := `version: 1
concurrency:
  claude: 1
  codex: 2
rates:
  codex:
    sustained_delay: 60`
	require.True(t, fs.Write(core.JoinPath(root, "agents.yaml"), cfg).OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	loaded := s.loadAgentsConfig()
	assert.NotNil(t, loaded)
	assert.Equal(t, 1, loaded.Version)
	assert.Equal(t, 1, loaded.Concurrency["claude"].Total)
	assert.Equal(t, 2, loaded.Concurrency["codex"].Total)
	assert.Equal(t, 60, loaded.Rates["codex"].SustainedDelay)
}

func TestQueue_LoadAgentsConfig_Bad(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	// Corrupt YAML file
	require.True(t, fs.Write(core.JoinPath(root, "agents.yaml"), "{{{not yaml!!!").OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Should return defaults when YAML is corrupt
	loaded := s.loadAgentsConfig()
	assert.NotNil(t, loaded)
	assert.Equal(t, "claude", loaded.Dispatch.DefaultAgent)
}

func TestQueue_LoadAgentsConfig_Ugly(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	// No agents.yaml file at all — should return defaults

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	loaded := s.loadAgentsConfig()
	assert.NotNil(t, loaded)
	assert.Equal(t, "claude", loaded.Dispatch.DefaultAgent)
	assert.Equal(t, "coding", loaded.Dispatch.DefaultTemplate)
	assert.NotNil(t, loaded.Concurrency)
}

// --- DrainQueue Bad ---

func TestQueue_DrainQueue_Bad_FrozenQueueDoesNothing(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := core.JoinPath(root, "workspace")

	// Create a queued workspace that would normally be drained
	ws := core.JoinPath(wsRoot, "ws-queued")
	fs.EnsureDir(ws)
	st := &WorkspaceStatus{Status: "queued", Agent: "codex", Repo: "go-io", Task: "fix it"}
	fs.Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(st))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		frozen:         true, // queue is frozen
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Frozen queue returns immediately without draining
	assert.NotPanics(t, func() { s.drainQueue() })

	// Workspace should still be queued
	updated := mustReadStatus(t, ws)
	assert.Equal(t, "queued", updated.Status)
}
