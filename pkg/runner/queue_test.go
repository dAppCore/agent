// SPDX-License-Identifier: EUPL-1.2

package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

// --- ConcurrencyLimit UnmarshalYAML ---

func TestQueue_ConcurrencyLimit_Good_Int(t *testing.T) {
	var cl ConcurrencyLimit
	node := &yaml.Node{Kind: yaml.ScalarNode, Value: "5", Tag: "!!int"}
	err := cl.UnmarshalYAML(node)
	assert.NoError(t, err)
	assert.Equal(t, 5, cl.Total)
	assert.Nil(t, cl.Models)
}

func TestQueue_ConcurrencyLimit_Good_Map(t *testing.T) {
	input := `
total: 5
gpt-5.4: 1
gpt-5.3-codex-spark: 3
`
	var cl ConcurrencyLimit
	err := yaml.Unmarshal([]byte(input), &cl)
	assert.NoError(t, err)
	assert.Equal(t, 5, cl.Total)
	assert.Equal(t, 1, cl.Models["gpt-5.4"])
	assert.Equal(t, 3, cl.Models["gpt-5.3-codex-spark"])
}

func TestQueue_ConcurrencyLimit_Bad_InvalidYAML(t *testing.T) {
	node := &yaml.Node{Kind: yaml.SequenceNode}
	var cl ConcurrencyLimit
	err := cl.UnmarshalYAML(node)
	assert.Error(t, err)
}

func TestQueue_ConcurrencyLimit_Ugly_ZeroTotal(t *testing.T) {
	input := `
total: 0
gpt-5.4: 1
`
	var cl ConcurrencyLimit
	err := yaml.Unmarshal([]byte(input), &cl)
	assert.NoError(t, err)
	assert.Equal(t, 0, cl.Total)
}

// --- baseAgent ---

func TestQueue_BaseAgent_Good_Simple(t *testing.T) {
	assert.Equal(t, "codex", baseAgent("codex"))
	assert.Equal(t, "claude", baseAgent("claude"))
	assert.Equal(t, "gemini", baseAgent("gemini"))
}

func TestQueue_BaseAgent_Good_WithModel(t *testing.T) {
	assert.Equal(t, "codex", baseAgent("codex:gpt-5.4"))
	assert.Equal(t, "claude", baseAgent("claude:haiku"))
}

func TestQueue_BaseAgent_Bad_CodexSpark(t *testing.T) {
	assert.Equal(t, "codex-spark", baseAgent("codex:gpt-5.3-codex-spark"))
}

func TestQueue_BaseAgent_Ugly_Empty(t *testing.T) {
	assert.Equal(t, "", baseAgent(""))
}

func TestQueue_BaseAgent_Ugly_MultipleColons(t *testing.T) {
	assert.Equal(t, "codex", baseAgent("codex:model:extra"))
}

// --- modelVariant ---

func TestQueue_ModelVariant_Good(t *testing.T) {
	assert.Equal(t, "gpt-5.4", modelVariant("codex:gpt-5.4"))
	assert.Equal(t, "haiku", modelVariant("claude:haiku"))
}

func TestQueue_ModelVariant_Bad_NoColon(t *testing.T) {
	assert.Equal(t, "", modelVariant("codex"))
	assert.Equal(t, "", modelVariant("claude"))
}

func TestQueue_ModelVariant_Ugly_Empty(t *testing.T) {
	assert.Equal(t, "", modelVariant(""))
}

// --- canDispatchAgent ---

func TestQueue_CanDispatchAgent_Good_NoConfig(t *testing.T) {
	svc := New()
	assert.True(t, svc.canDispatchAgent("codex"), "no config → unlimited")
}

func TestQueue_CanDispatchAgent_Good_UnknownAgent(t *testing.T) {
	svc := New()
	assert.True(t, svc.canDispatchAgent("unknown-agent"))
}

func TestQueue_CanDispatchAgent_Bad_AtLimit(t *testing.T) {
	svc := New()
	// Simulate 5 running codex workspaces
	for i := 0; i < 5; i++ {
		svc.TrackWorkspace("ws-"+string(rune('a'+i)), &WorkspaceStatus{
			Status: "running", Agent: "codex", PID: 99999, // PID won't be alive
		})
	}
	// Since PIDs aren't alive, count should be 0
	assert.True(t, svc.canDispatchAgent("codex"), "dead PIDs don't count")
}

func TestQueue_CanDispatchAgent_Ugly_ZeroLimit(t *testing.T) {
	svc := New()
	// Zero total means unlimited
	assert.True(t, svc.canDispatchAgent("codex"))
}

// --- countRunningByAgent ---

func TestQueue_CountRunningByAgent_Good_Empty(t *testing.T) {
	svc := New()
	assert.Equal(t, 0, svc.countRunningByAgent("codex"))
}

func TestQueue_CountRunningByAgent_Good_SkipsNonRunning(t *testing.T) {
	svc := New()
	svc.TrackWorkspace("ws-1", &WorkspaceStatus{Status: "completed", Agent: "codex"})
	svc.TrackWorkspace("ws-2", &WorkspaceStatus{Status: "queued", Agent: "codex"})
	assert.Equal(t, 0, svc.countRunningByAgent("codex"))
}

func TestQueue_CountRunningByAgent_Bad_SkipsMismatchedAgent(t *testing.T) {
	svc := New()
	svc.TrackWorkspace("ws-1", &WorkspaceStatus{Status: "running", Agent: "claude", PID: 1})
	assert.Equal(t, 0, svc.countRunningByAgent("codex"))
}

func TestQueue_CountRunningByAgent_Ugly_DeadPIDsIgnored(t *testing.T) {
	svc := New()
	svc.TrackWorkspace("ws-1", &WorkspaceStatus{
		Status: "running", Agent: "codex", PID: 999999999, // almost certainly not alive
	})
	assert.Equal(t, 0, svc.countRunningByAgent("codex"))
}

// --- countRunningByModel ---

func TestQueue_CountRunningByModel_Good_Empty(t *testing.T) {
	svc := New()
	assert.Equal(t, 0, svc.countRunningByModel("codex:gpt-5.4"))
}

func TestQueue_CountRunningByModel_Bad_WrongModel(t *testing.T) {
	svc := New()
	svc.TrackWorkspace("ws-1", &WorkspaceStatus{
		Status: "running", Agent: "codex:gpt-5.3", PID: 1,
	})
	assert.Equal(t, 0, svc.countRunningByModel("codex:gpt-5.4"))
}

func TestQueue_CountRunningByModel_Ugly_ExactMatch(t *testing.T) {
	svc := New()
	svc.TrackWorkspace("ws-1", &WorkspaceStatus{
		Status: "running", Agent: "codex:gpt-5.4", PID: 999999999,
	})
	// PID is dead so count is 0
	assert.Equal(t, 0, svc.countRunningByModel("codex:gpt-5.4"))
}

// --- drainQueue ---

func TestQueue_DrainQueue_Good_FrozenDoesNothing(t *testing.T) {
	svc := New()
	svc.frozen = true
	assert.NotPanics(t, func() {
		svc.drainQueue()
	})
}

func TestQueue_DrainQueue_Bad_EmptyWorkspace(t *testing.T) {
	svc := New()
	svc.frozen = false
	assert.NotPanics(t, func() {
		svc.drainQueue()
	})
}

func TestQueue_DrainQueue_Ugly_NoStatusFiles(t *testing.T) {
	svc := New()
	svc.frozen = false
	// drainOne scans disk — with no workspace root, it finds nothing
	assert.False(t, svc.drainOne())
}

// --- loadAgentsConfig ---

func TestQueue_LoadAgentsConfig_Good_ReadsFile(t *testing.T) {
	svc := New()
	cfg := svc.loadAgentsConfig()
	assert.NotNil(t, cfg)
	assert.GreaterOrEqual(t, cfg.Version, 0)
}

func TestQueue_LoadAgentsConfig_Bad_NoFile(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	svc := New()
	cfg := svc.loadAgentsConfig()
	assert.NotNil(t, cfg, "should return defaults when no file found")
}

func TestQueue_LoadAgentsConfig_Ugly_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)
	fs.Write(dir+"/agents.yaml", "{{{{invalid yaml")
	svc := New()
	cfg := svc.loadAgentsConfig()
	assert.NotNil(t, cfg, "should return defaults on parse failure")
}

// --- AgentsConfig full parse ---

func TestQueue_AgentsConfig_Good_FullParse(t *testing.T) {
	input := `
version: 1
dispatch:
  default_agent: claude
  default_template: coding
concurrency:
  claude: 3
  codex:
    total: 5
    gpt-5.4: 1
rates:
  gemini:
    reset_utc: "06:00"
    sustained_delay: 120
`
	var cfg AgentsConfig
	err := yaml.Unmarshal([]byte(input), &cfg)
	assert.NoError(t, err)
	assert.Equal(t, 1, cfg.Version)
	assert.Equal(t, 3, cfg.Concurrency["claude"].Total)
	assert.Equal(t, 5, cfg.Concurrency["codex"].Total)
	assert.Equal(t, 1, cfg.Concurrency["codex"].Models["gpt-5.4"])
	assert.Equal(t, 120, cfg.Rates["gemini"].SustainedDelay)
}
