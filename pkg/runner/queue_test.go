// SPDX-License-Identifier: EUPL-1.2

package runner

import (
	"testing"

	core "dappco.re/go"
	"gopkg.in/yaml.v3"
)

// --- ConcurrencyLimit UnmarshalYAML ---

func TestInt_ConcurrencyLimit_UnmarshalYAML_Good(t *testing.T) {
	var cl ConcurrencyLimit
	node := &yaml.Node{Kind: yaml.ScalarNode, Value: "5", Tag: "!!int"}
	err := cl.UnmarshalYAML(node)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 5, cl.Total)
	core.AssertNil(t, cl.Models)
}

func TestMap_ConcurrencyLimit_UnmarshalYAML_Good(t *testing.T) {
	input := `
total: 5
gpt-5.4: 1
gpt-5.3-codex-spark: 3
`
	var cl ConcurrencyLimit
	err := yaml.Unmarshal([]byte(input), &cl)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 5, cl.Total)
	core.AssertEqual(t, 1, cl.Models["gpt-5.4"])
	core.AssertEqual(t, 3, cl.Models["gpt-5.3-codex-spark"])
}

func TestInvalidYAML_ConcurrencyLimit_UnmarshalYAML_Bad(t *testing.T) {
	node := &yaml.Node{Kind: yaml.SequenceNode}
	var cl ConcurrencyLimit
	err := cl.UnmarshalYAML(node)
	core.AssertError(t, err)
}

func TestZeroTotal_ConcurrencyLimit_UnmarshalYAML_Ugly(t *testing.T) {
	input := `
total: 0
gpt-5.4: 1
`
	var cl ConcurrencyLimit
	err := yaml.Unmarshal([]byte(input), &cl)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, cl.Total)
}

// --- baseAgent ---

func TestQueue_BaseAgent_Good_Simple(t *testing.T) {
	core.AssertEqual(t, "codex", baseAgent("codex"))
	core.AssertEqual(t, "claude", baseAgent("claude"))
	core.AssertEqual(t, "gemini", baseAgent("gemini"))
}

func TestQueue_BaseAgent_Good_WithModel(t *testing.T) {
	core.AssertEqual(t, "codex", baseAgent("codex:gpt-5.4"))
	core.AssertEqual(t, "claude", baseAgent("claude:haiku"))
	core.AssertNotEmpty(t, baseAgent("codex:gpt-5.4"))
}

func TestQueue_BaseAgent_Good_CodexSpark(t *testing.T) {
	got := baseAgent("codex:gpt-5.3-codex-spark")
	core.AssertEqual(t, "codex", got)
	core.AssertNotContains(t, got, ":")
}

func TestQueue_BaseAgent_Ugly_Empty(t *testing.T) {
	got := baseAgent("")
	core.AssertEqual(t, "", got)
	core.AssertEmpty(t, got)
}

func TestQueue_BaseAgent_Ugly_MultipleColons(t *testing.T) {
	got := baseAgent("codex:model:extra")
	core.AssertEqual(t, "codex", got)
	core.AssertNotContains(t, got, ":")
}

// --- modelVariant ---

func TestQueue_ModelVariant_Good(t *testing.T) {
	core.AssertEqual(t, "gpt-5.4", modelVariant("codex:gpt-5.4"))
	core.AssertEqual(t, "haiku", modelVariant("claude:haiku"))
	core.AssertNotEmpty(t, modelVariant("codex:gpt-5.4"))
}

func TestQueue_ModelVariant_Bad_NoColon(t *testing.T) {
	core.AssertEqual(t, "", modelVariant("codex"))
	core.AssertEqual(t, "", modelVariant("claude"))
	core.AssertEqual(t, "", modelVariant("gemini"))
}

func TestQueue_ModelVariant_Ugly_Empty(t *testing.T) {
	got := modelVariant("")
	core.AssertEqual(t, "", got)
	core.AssertEmpty(t, got)
}

// --- canDispatchAgent ---

func TestQueue_CanDispatchAgent_Good_NoConfig(t *testing.T) {
	svc := New()
	can, _ := svc.canDispatchAgent("codex")
	core.AssertTrue(t, can, "no config → unlimited")
}

func TestQueue_CanDispatchAgent_Good_UnknownAgent(t *testing.T) {
	svc := New()
	can, _ := svc.canDispatchAgent("unknown-agent")
	core.AssertTrue(t, can)
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
	can, _ := svc.canDispatchAgent("codex")
	core.AssertTrue(t, can, "dead PIDs don't count")
}

func TestQueue_CanDispatchAgent_Ugly_ZeroLimit(t *testing.T) {
	svc := New()
	// Zero total means unlimited
	can, _ := svc.canDispatchAgent("codex")
	core.AssertTrue(t, can)
}

// --- countRunningByAgent ---

func TestQueue_CountRunningByAgent_Good_Empty(t *testing.T) {
	svc := New()
	// Add a non-running entry so Registry is non-empty (avoids disk fallback)
	svc.TrackWorkspace("ws-seed", &WorkspaceStatus{Status: "completed", Agent: "claude"})
	core.AssertEqual(t, 0, svc.countRunningByAgent("codex"))
}

func TestQueue_CountRunningByAgent_Good_SkipsNonRunning(t *testing.T) {
	svc := New()
	svc.TrackWorkspace("ws-1", &WorkspaceStatus{Status: "completed", Agent: "codex"})
	svc.TrackWorkspace("ws-2", &WorkspaceStatus{Status: "queued", Agent: "codex"})
	core.AssertEqual(t, 0, svc.countRunningByAgent("codex"))
}

func TestQueue_CountRunningByAgent_Bad_SkipsMismatchedAgent(t *testing.T) {
	svc := New()
	svc.TrackWorkspace("ws-1", &WorkspaceStatus{Status: "running", Agent: "claude", PID: 1})
	core.AssertEqual(t, 0, svc.countRunningByAgent("codex"))
}

func TestQueue_CountRunningByAgent_Ugly_DeadPIDsIgnored(t *testing.T) {
	svc := New()
	svc.TrackWorkspace("ws-1", &WorkspaceStatus{
		Status: "running", Agent: "codex", PID: 999999999, // almost certainly not alive
	})
	core.AssertEqual(t, 0, svc.countRunningByAgent("codex"))
}

// --- countRunningByModel ---

func TestQueue_CountRunningByModel_Good_Empty(t *testing.T) {
	svc := New()
	count := svc.countRunningByModel("codex:gpt-5.4")
	core.AssertEqual(t, 0, count)
	core.AssertGreaterOrEqual(t, count, 0)
}

func TestQueue_CountRunningByModel_Bad_WrongModel(t *testing.T) {
	svc := New()
	svc.TrackWorkspace("ws-1", &WorkspaceStatus{
		Status: "running", Agent: "codex:gpt-5.3", PID: 1,
	})
	core.AssertEqual(t, 0, svc.countRunningByModel("codex:gpt-5.4"))
}

func TestQueue_CountRunningByModel_Ugly_ExactMatch(t *testing.T) {
	svc := New()
	svc.TrackWorkspace("ws-1", &WorkspaceStatus{
		Status: "running", Agent: "codex:gpt-5.4", PID: 999999999,
	})
	// PID is dead so count is 0
	core.AssertEqual(t, 0, svc.countRunningByModel("codex:gpt-5.4"))
}

// --- drainQueue ---

func TestQueue_DrainQueue_Good_FrozenDoesNothing(t *testing.T) {
	svc := New()
	svc.frozen = true
	core.AssertNotPanics(t, func() {
		svc.drainQueue()
	})
}

func TestQueue_DrainQueue_Bad_EmptyWorkspace(t *testing.T) {
	svc := New()
	svc.frozen = false
	core.AssertNotPanics(t, func() {
		svc.drainQueue()
	})
}

func TestQueue_DrainQueue_Ugly_NoStatusFiles(t *testing.T) {
	svc := New()
	svc.frozen = false
	// drainOne scans disk — with no workspace root, it finds nothing
	core.AssertFalse(t, svc.drainOne())
}

// --- loadAgentsConfig ---

func TestQueue_LoadAgentsConfig_Good_ReadsFile(t *testing.T) {
	svc := New()
	cfg := svc.loadAgentsConfig()
	core.AssertNotNil(t, cfg)
	core.AssertGreaterOrEqual(t, cfg.Version, 0)
}

func TestQueue_LoadAgentsConfig_Bad_NoFile(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	svc := New()
	cfg := svc.loadAgentsConfig()
	core.AssertNotNil(t, cfg, "should return defaults when no file found")
}

func TestQueue_LoadAgentsConfig_Ugly_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)
	fs.Write(dir+"/agents.yaml", "{{{{invalid yaml")
	svc := New()
	cfg := svc.loadAgentsConfig()
	core.AssertNotNil(t, cfg, "should return defaults on parse failure")
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
	core.AssertNoError(t, err)
	core.AssertEqual(t, 1, cfg.Version)
	core.AssertEqual(t, 3, cfg.Concurrency["claude"].Total)
	core.AssertEqual(t, 5, cfg.Concurrency["codex"].Total)
	core.AssertEqual(t, 1, cfg.Concurrency["codex"].Models["gpt-5.4"])
	core.AssertEqual(t, 120, cfg.Rates["gemini"].SustainedDelay)
}

// --- DispatchConfig runtime/image/gpu ---

func TestQueue_DispatchConfig_Good_RuntimeImageGPU(t *testing.T) {
	input := `
version: 1
dispatch:
  default_agent: claude
  runtime: apple
  image: core-ml
  gpu: true
`
	var cfg AgentsConfig
	err := yaml.Unmarshal([]byte(input), &cfg)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "apple", cfg.Dispatch.Runtime)
	core.AssertEqual(t, "core-ml", cfg.Dispatch.Image)
	core.AssertTrue(t, cfg.Dispatch.GPU)
}

func TestQueue_DispatchConfig_Bad_OmittedRuntimeFields(t *testing.T) {
	// When runtime / image / gpu are missing the yaml unmarshals into the
	// struct's zero values. Callers treat empty runtime as "auto".
	input := `
version: 1
dispatch:
  default_agent: claude
`
	var cfg AgentsConfig
	err := yaml.Unmarshal([]byte(input), &cfg)
	core.AssertNoError(t, err)
	core.AssertEmpty(t, cfg.Dispatch.Runtime)
	core.AssertEmpty(t, cfg.Dispatch.Image)
	core.AssertFalse(t, cfg.Dispatch.GPU)
}

func TestQueue_DispatchConfig_Ugly_PartialRuntimeBlock(t *testing.T) {
	// Only runtime is set; image keeps its zero value, gpu defaults to false.
	input := `
version: 1
dispatch:
  runtime: docker
`
	var cfg AgentsConfig
	err := yaml.Unmarshal([]byte(input), &cfg)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "docker", cfg.Dispatch.Runtime)
	core.AssertEmpty(t, cfg.Dispatch.Image)
	core.AssertFalse(t, cfg.Dispatch.GPU)
}

// --- AgentIdentity ---

func TestQueue_AgentIdentity_Good_FullParse(t *testing.T) {
	input := `
version: 1
agents:
  cladius:
    host: local
    runner: claude
    active: true
    roles: [dispatch, review, plan]
  codex:
    host: cloud
    runner: openai
    active: true
    roles: [worker]
`
	var cfg AgentsConfig
	err := yaml.Unmarshal([]byte(input), &cfg)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "local", cfg.Agents["cladius"].Host)
	core.AssertEqual(t, "claude", cfg.Agents["cladius"].Runner)
	core.AssertTrue(t, cfg.Agents["cladius"].Active)
	core.AssertContains(t, cfg.Agents["cladius"].Roles, "dispatch")
	core.AssertContains(t, cfg.Agents["cladius"].Roles, "review")
	core.AssertEqual(t, "cloud", cfg.Agents["codex"].Host)
}

func TestQueue_AgentIdentity_Bad_MissingAgentsBlock(t *testing.T) {
	input := `
version: 1
`
	var cfg AgentsConfig
	err := yaml.Unmarshal([]byte(input), &cfg)
	core.AssertNoError(t, err)
	core.AssertEmpty(t, cfg.Agents)
}

func TestQueue_AgentIdentity_Ugly_OnlyHostSet(t *testing.T) {
	// An identity with only host set populates host and leaves zero values
	// for runner / active / roles. Routing code treats Active=false as a
	// disabled identity and SHOULD NOT crash on missing fields.
	input := `
agents:
  ghost:
    host: 192.168.0.42
`
	var cfg AgentsConfig
	err := yaml.Unmarshal([]byte(input), &cfg)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "192.168.0.42", cfg.Agents["ghost"].Host)
	core.AssertEmpty(t, cfg.Agents["ghost"].Runner)
	core.AssertFalse(t, cfg.Agents["ghost"].Active)
	core.AssertEmpty(t, cfg.Agents["ghost"].Roles)
}
