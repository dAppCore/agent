// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestQueue_BaseAgent_Ugly_Empty(t *testing.T) {
	agent := baseAgent("")
	core.AssertEqual(t, "", agent)
	core.AssertEmpty(t, agent)
}

func TestQueue_BaseAgent_Ugly_MultipleColons(t *testing.T) {
	// SplitN with N=2 should only split on first colon
	agent := baseAgent("claude:opus:extra")
	core.AssertEqual(t, "claude", agent)
	core.AssertNotContains(t, agent, ":")
}

func TestQueue_DispatchConfig_Good_Defaults(t *testing.T) {
	// loadAgentsConfig falls back to defaults when no config file exists
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), codePath: t.TempDir()}
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	cfg := s.loadAgentsConfig()
	core.AssertEqual(t, "claude", cfg.Dispatch.DefaultAgent)
	core.AssertEqual(t, "coding", cfg.Dispatch.DefaultTemplate)
	core.AssertEqual(t, 60, cfg.Dispatch.TimeoutMinutes)
	core.AssertEqual(t, 1, cfg.Concurrency["claude"].Total)
	core.AssertEqual(t, 3, cfg.Concurrency["gemini"].Total)
}

func TestQueue_Config_Good_TimeoutDefault(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	core.RequireTrue(t, fs.Write(core.JoinPath(root, "agents.yaml"), "version: 1\ndispatch:\n  default_agent: codex\n").OK)
	t.Cleanup(func() { setWorkspaceRootOverride("") })

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), codePath: t.TempDir()}
	cfg := s.loadAgentsConfig()

	core.AssertEqual(t, 60, cfg.Dispatch.TimeoutMinutes)
}

func TestQueue_DispatchConfig_Good_RuntimeImageGPUFromYAML(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	core.RequireTrue(t, fs.Write(core.JoinPath(root, "agents.yaml"), core.Concat(
		"version: 1\n",
		"dispatch:\n",
		"  runtime: apple\n",
		"  image: core-ml\n",
		"  gpu: true\n",
		"  timeout_minutes: 45\n",
	)).OK)

	t.Cleanup(func() {
		setWorkspaceRootOverride("")
	})

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), codePath: t.TempDir()}
	cfg := s.loadAgentsConfig()

	core.AssertEqual(t, "apple", cfg.Dispatch.Runtime)
	core.AssertEqual(t, "core-ml", cfg.Dispatch.Image)
	core.AssertTrue(t, cfg.Dispatch.GPU)
	core.AssertEqual(t, 45, cfg.Dispatch.TimeoutMinutes)
}

func TestQueue_DispatchConfig_Bad_OmittedRuntimeFields(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	core.RequireTrue(t, fs.Write(core.JoinPath(root, "agents.yaml"), "version: 1\ndispatch:\n  default_agent: codex\n").OK)
	t.Cleanup(func() { setWorkspaceRootOverride("") })

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), codePath: t.TempDir()}
	cfg := s.loadAgentsConfig()

	core.AssertEmpty(t, cfg.Dispatch.Runtime)
	core.AssertEmpty(t, cfg.Dispatch.Image)
	core.AssertFalse(t, cfg.Dispatch.GPU)
}

func TestQueue_DispatchConfig_Ugly_PartialRuntimeBlock(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	core.RequireTrue(t, fs.Write(core.JoinPath(root, "agents.yaml"), "version: 1\ndispatch:\n  runtime: docker\n").OK)
	t.Cleanup(func() { setWorkspaceRootOverride("") })

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), codePath: t.TempDir()}
	cfg := s.loadAgentsConfig()

	core.AssertEqual(t, "docker", cfg.Dispatch.Runtime)
	core.AssertEmpty(t, cfg.Dispatch.Image)
	core.AssertFalse(t, cfg.Dispatch.GPU)
}

// --- AgentIdentity ---

func TestQueue_AgentIdentity_Good_FullParseFromYAML(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	core.RequireTrue(t, fs.Write(core.JoinPath(root, "agents.yaml"), core.Concat(
		"version: 1\n",
		"agents:\n",
		"  cladius:\n",
		"    host: local\n",
		"    runner: claude\n",
		"    active: true\n",
		"    roles: [dispatch, review, plan]\n",
		"  codex:\n",
		"    host: cloud\n",
		"    runner: openai\n",
		"    active: true\n",
		"    roles: [worker]\n",
	)).OK)
	t.Cleanup(func() { setWorkspaceRootOverride("") })

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), codePath: t.TempDir()}
	cfg := s.loadAgentsConfig()

	core.AssertEqual(t, "local", cfg.Agents["cladius"].Host)
	core.AssertEqual(t, "claude", cfg.Agents["cladius"].Runner)
	core.AssertTrue(t, cfg.Agents["cladius"].Active)
	core.AssertContains(t, cfg.Agents["cladius"].Roles, "dispatch")
	core.AssertEqual(t, "cloud", cfg.Agents["codex"].Host)
}

func TestQueue_AgentIdentity_Bad_MissingAgentsBlock(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	core.RequireTrue(t, fs.Write(core.JoinPath(root, "agents.yaml"), "version: 1\n").OK)
	t.Cleanup(func() { setWorkspaceRootOverride("") })

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), codePath: t.TempDir()}
	cfg := s.loadAgentsConfig()
	core.AssertEmpty(t, cfg.Agents)
}

func TestQueue_AgentIdentity_Ugly_OnlyHostSet(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	core.RequireTrue(t, fs.Write(core.JoinPath(root, "agents.yaml"), core.Concat(
		"agents:\n",
		"  ghost:\n",
		"    host: 192.168.0.42\n",
	)).OK)
	t.Cleanup(func() { setWorkspaceRootOverride("") })

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), codePath: t.TempDir()}
	cfg := s.loadAgentsConfig()

	core.AssertEqual(t, "192.168.0.42", cfg.Agents["ghost"].Host)
	core.AssertEmpty(t, cfg.Agents["ghost"].Runner)
	core.AssertFalse(t, cfg.Agents["ghost"].Active)
}

func TestQueue_DispatchConfig_Good_WorkspaceRootOverride(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	customRoot := core.JoinPath(root, "agent-workspaces")
	core.RequireTrue(t, fs.Write(core.JoinPath(root, "agents.yaml"), core.Concat(
		"version: 1\n",
		"dispatch:\n",
		"  workspace_root: ", customRoot, "\n",
	)).OK)

	t.Cleanup(func() {
		setWorkspaceRootOverride("")
	})

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), codePath: t.TempDir()}
	cfg := s.loadAgentsConfig()

	core.AssertEqual(t, customRoot, cfg.Dispatch.WorkspaceRoot)
	core.AssertEqual(t, customRoot, WorkspaceRoot())
}

func TestQueue_CanDispatchAgent_Good_NoConfig(t *testing.T) {
	// With no running workspaces and default config, should be able to dispatch
	root := t.TempDir()
	setTestWorkspace(t, root)
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), codePath: t.TempDir()}
	core.AssertTrue(t, s.canDispatchAgent("gemini"))
}

func TestQueue_CanDispatchAgent_Good_UnknownAgent(t *testing.T) {
	// Unknown agent has no limit, so always allowed
	root := t.TempDir()
	setTestWorkspace(t, root)
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), codePath: t.TempDir()}
	core.AssertTrue(t, s.canDispatchAgent("unknown-agent"))
}

func TestQueue_CountRunningByAgent_Good_EmptyWorkspace(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	core.AssertEqual(t, 0, s.countRunningByAgent("gemini"))
	core.AssertEqual(t, 0, s.countRunningByAgent("claude"))
}

func TestQueue_CountRunningByAgent_Good_NoRunning(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	// Create a workspace with completed status under workspace/
	ws := core.JoinPath(root, "workspace", "test-ws")
	core.RequireTrue(t, fs.EnsureDir(ws).OK)
	core.RequireNoError(t, writeStatus(ws, &WorkspaceStatus{
		Status: "completed",
		Agent:  "gemini",
		PID:    0,
	}))

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	core.AssertEqual(t, 0, s.countRunningByAgent("gemini"))
}

func TestQueue_DelayForAgent_Good_NoConfig(t *testing.T) {
	// With no config, delay should be 0
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), codePath: t.TempDir()}
	core.AssertEqual(t, 0, int(s.delayForAgent("gemini").Seconds()))
}

func TestQueue_ConcurrencyLimit_UnmarshalYAML_Good(t *testing.T) {
	var limit ConcurrencyLimit
	err := yaml.Unmarshal([]byte("total: 3\ngpt-5.4: 2\ngpt-5.3-codex-spark: 1\n"), &limit)

	core.RequireNoError(t, err)
	core.AssertEqual(t, 3, limit.Total)
	core.AssertEqual(t, 2, limit.Models["gpt-5.4"])
	core.AssertEqual(t, 1, limit.Models["gpt-5.3-codex-spark"])
}

func TestQueue_ConcurrencyLimit_UnmarshalYAML_Bad(t *testing.T) {
	var limit ConcurrencyLimit
	err := yaml.Unmarshal([]byte("2\n"), &limit)

	core.RequireNoError(t, err)
	core.AssertEqual(t, 2, limit.Total)
	core.AssertNil(t, limit.Models)
}

func TestQueue_ConcurrencyLimit_UnmarshalYAML_Ugly(t *testing.T) {
	var limit ConcurrencyLimit
	err := yaml.Unmarshal([]byte("total: nope\n"), &limit)

	core.AssertError(t, err)
	core.AssertEqual(t, 0, limit.Total)
}
