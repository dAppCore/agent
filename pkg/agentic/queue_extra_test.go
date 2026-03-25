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
