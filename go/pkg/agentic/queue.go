// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"strconv"
	"time"

	core "dappco.re/go"
	agentcompat "dappco.re/go/agent/pkg/agentcompat"
	"gopkg.in/yaml.v3"
)

const defaultDispatchTimeoutMinutes = 60

// config := agentic.DispatchConfig{DefaultAgent: "claude", DefaultTemplate: "coding", Runtime: "auto", Image: "core-dev", TimeoutMinutes: 60}
type DispatchConfig struct {
	DefaultAgent    string `yaml:"default_agent"`
	DefaultTemplate string `yaml:"default_template"`
	WorkspaceRoot   string `yaml:"workspace_root"`
	// TimeoutMinutes bounds agent runtime before dispatch marks the workspace
	// failed and go-process shuts the process tree down.
	TimeoutMinutes int `yaml:"timeout_minutes"`
	// Runtime selects the container runtime — auto | apple | docker | podman.
	// auto detects in preference order: Apple Container -> Docker -> Podman.
	// Apple Containers (macOS 26+) provide hardware VM isolation and sub-second
	// startup; Docker is the cross-platform fallback; Podman is the rootless
	// option for Linux environments where Docker is unavailable.
	Runtime string `yaml:"runtime"`
	// Image is the default container image for non-native agent dispatch.
	// Used by go-build LinuxKit images such as "core-dev", "core-ml", "core-minimal".
	Image string `yaml:"image"`
	// GPU enables GPU passthrough — Metal on Apple Containers (when available),
	// NVIDIA on Docker. Default false.
	GPU bool `yaml:"gpu"`
}

// rate := agentic.RateConfig{ResetUTC: "06:00", DailyLimit: 200, MinDelay: 15, SustainedDelay: 120, BurstWindow: 2, BurstDelay: 15}
type RateConfig struct {
	ResetUTC       string `yaml:"reset_utc"`
	DailyLimit     int    `yaml:"daily_limit"`
	MinDelay       int    `yaml:"min_delay"`
	SustainedDelay int    `yaml:"sustained_delay"`
	BurstWindow    int    `yaml:"burst_window"`
	BurstDelay     int    `yaml:"burst_delay"`
}

// claude: 1                       → Total=1, Models=nil
// codex:                          → Total=2, Models={"gpt-5.4": 1, "gpt-5.3-codex-spark": 1}
//
//	total: 2
//	gpt-5.4: 1
//	gpt-5.3-codex-spark: 1
type ConcurrencyLimit = agentcompat.ConcurrencyLimit

// identity := agentic.AgentIdentity{Host: "local", Runner: "claude", Active: true, Roles: []string{"dispatch", "review"}}
// AgentIdentity represents one entry in the agents.yaml `agents:` block —
// the named identity (e.g. cladius, charon, codex) that can dispatch work.
type AgentIdentity struct {
	// Host is "local", "cloud", "remote", or an explicit IP/hostname.
	//   identity := agentic.AgentIdentity{Host: "local"}
	Host string `yaml:"host"`
	// Runner is the runtime that backs this identity ("claude", "openai", "gemini").
	//   identity := agentic.AgentIdentity{Runner: "claude"}
	Runner string `yaml:"runner"`
	// Active reports whether this identity participates in dispatch.
	//   identity := agentic.AgentIdentity{Active: true}
	Active bool `yaml:"active"`
	// Roles enumerates the workflows this identity can handle:
	// dispatch, worker, review, qa, plan.
	//   identity := agentic.AgentIdentity{Roles: []string{"dispatch", "review", "plan"}}
	Roles []string `yaml:"roles"`
}

// config := agentic.AgentsConfig{Version: 1, Dispatch: agentic.DispatchConfig{DefaultAgent: "claude"}}
type AgentsConfig struct {
	Version     int                         `yaml:"version"`
	Dispatch    DispatchConfig              `yaml:"dispatch"`
	Concurrency map[string]ConcurrencyLimit `yaml:"concurrency"`
	Rates       map[string]RateConfig       `yaml:"rates"`
	// Agents declares named identities (cladius, charon, codex, clotho)
	// keyed by name. Each identity carries host/runner/roles metadata used
	// by message routing, brain attribution, and session ownership.
	Agents map[string]AgentIdentity `yaml:"agents"`
}

func normaliseDispatchConfig(config DispatchConfig) DispatchConfig {
	if config.TimeoutMinutes <= 0 {
		config.TimeoutMinutes = defaultDispatchTimeoutMinutes
	}
	return config
}

// config := s.loadAgentsConfig()
func (s *PrepSubsystem) loadAgentsConfig() *AgentsConfig {
	paths := []string{
		// Operator config first (~/Lethean/conf/agents.yaml), then the
		// CORE_WORKSPACE-relative config (CoreRoot()/agents.yaml — multi-tenant
		// tenants drop their own agents.yaml in their workspace root), then the
		// shipped repo config (core/agent/.core/agents.yaml — the .core convention
		// is fine for the in-repo default), then legacy config/agents.yaml for
		// back-compat. Without a found config dispatch falls back to the hardcoded
		// default (no opencode entry → opencode unlimited).
		AgentsConfigPath(),
		core.JoinPath(CoreRoot(), "agents.yaml"),
		core.JoinPath(s.codePath, "core", "agent", ".core", "agents.yaml"),
		core.JoinPath(s.codePath, "core", "agent", "config", "agents.yaml"),
	}

	for _, path := range paths {
		readResult := fs.Read(path)
		if !readResult.OK {
			continue
		}
		var config AgentsConfig
		if err := yaml.Unmarshal([]byte(readResult.Value.(string)), &config); err != nil {
			continue
		}
		config.Dispatch = normaliseDispatchConfig(config.Dispatch)
		setWorkspaceRootOverride(config.Dispatch.WorkspaceRoot)
		return &config
	}

	setWorkspaceRootOverride("")
	return &AgentsConfig{
		Dispatch: DispatchConfig{
			DefaultAgent:    "claude",
			DefaultTemplate: "coding",
			TimeoutMinutes:  defaultDispatchTimeoutMinutes,
		},
		Concurrency: map[string]ConcurrencyLimit{
			"claude": {Total: 1},
			"gemini": {Total: 3},
		},
	}
}

// delay := s.delayForAgent("codex:gpt-5.4")
func (s *PrepSubsystem) delayForAgent(agent string) time.Duration {
	var rates map[string]RateConfig
	if s.ServiceRuntime != nil {
		if configured, ok := s.Core().Config().Get("agents.rates").Value.(map[string]RateConfig); ok {
			rates = configured
		}
	}
	if rates == nil {
		config := s.loadAgentsConfig()
		rates = config.Rates
	}
	base := baseAgent(agent)
	rate, ok := rates[base]
	if !ok || rate.SustainedDelay == 0 {
		return 0
	}

	resetHour, resetMin := 6, 0
	parts := core.Split(rate.ResetUTC, ":")
	if len(parts) >= 2 {
		if hour, err := strconv.Atoi(core.Trim(parts[0])); err == nil {
			resetHour = hour
		}
		if min, err := strconv.Atoi(core.Trim(parts[1])); err == nil {
			resetMin = min
		}
	}

	now := time.Now().UTC()
	resetToday := time.Date(now.Year(), now.Month(), now.Day(), resetHour, resetMin, 0, 0, time.UTC)
	if now.Before(resetToday) {
		resetToday = resetToday.AddDate(0, 0, -1)
	}
	nextReset := resetToday.AddDate(0, 0, 1)
	hoursUntilReset := nextReset.Sub(now).Hours()

	delay := time.Duration(rate.SustainedDelay) * time.Second
	if rate.BurstWindow > 0 && hoursUntilReset <= float64(rate.BurstWindow) {
		delay = time.Duration(rate.BurstDelay) * time.Second
	}

	minDelay := time.Duration(rate.MinDelay) * time.Second
	if minDelay > delay {
		delay = minDelay
	}

	return delay
}

// n := s.countRunningByAgent("codex")
func (s *PrepSubsystem) countRunningByAgent(agent string) int {
	var runtime *core.Core
	if s.ServiceRuntime != nil {
		runtime = s.Core()
	}
	if s.workspaces != nil && s.workspaces.Len() > 0 {
		count := 0
		s.workspaces.Each(func(_ string, workspaceStatus *WorkspaceStatus) {
			if workspaceStatus.Status == "running" && baseAgent(workspaceStatus.Agent) == agent && workspaceRunning(runtime, workspaceStatus) {
				count++
			}
		})
		return count
	}

	return s.countRunningByAgentDisk(runtime, agent)
}

func (s *PrepSubsystem) countRunningByAgentDisk(runtime *core.Core, agent string) int {
	count := 0
	for _, statusPath := range WorkspaceStatusPaths() {
		result := ReadStatusResult(core.PathDir(statusPath))
		workspaceStatus, ok := workspaceStatusValue(result)
		if !ok || workspaceStatus.Status != "running" {
			continue
		}
		if baseAgent(workspaceStatus.Agent) != agent {
			continue
		}
		if workspaceRunning(runtime, workspaceStatus) {
			count++
		}
	}
	return count
}

// n := s.countRunningByModel("codex:gpt-5.4")
func (s *PrepSubsystem) countRunningByModel(agent string) int {
	var runtime *core.Core
	if s.ServiceRuntime != nil {
		runtime = s.Core()
	}
	if s.workspaces != nil && s.workspaces.Len() > 0 {
		count := 0
		s.workspaces.Each(func(_ string, workspaceStatus *WorkspaceStatus) {
			if workspaceStatus.Status == "running" && workspaceStatus.Agent == agent && workspaceRunning(runtime, workspaceStatus) {
				count++
			}
		})
		return count
	}

	return s.countRunningByModelDisk(runtime, agent)
}

func (s *PrepSubsystem) countRunningByModelDisk(runtime *core.Core, agent string) int {
	count := 0
	for _, statusPath := range WorkspaceStatusPaths() {
		result := ReadStatusResult(core.PathDir(statusPath))
		workspaceStatus, ok := workspaceStatusValue(result)
		if !ok || workspaceStatus.Status != "running" {
			continue
		}
		if workspaceStatus.Agent != agent {
			continue
		}
		if workspaceRunning(runtime, workspaceStatus) {
			count++
		}
	}
	return count
}

// workspaceRunning reports whether a running-status workspace counts toward the
// concurrency limit. A VZ dispatch (Runtime=="vz") always counts: the VM lives
// in-process under a sentinel PID, so ProcessAlive cannot see it. Every other
// dispatch counts only while its host process is alive (the unchanged OCI/native
// rule). Callers must have already checked Status=="running".
func workspaceRunning(runtime *core.Core, workspaceStatus *WorkspaceStatus) bool {
	if workspaceStatus.Runtime == vzRuntimeName {
		return true
	}
	return ProcessAlive(runtime, workspaceStatus.ProcessID, workspaceStatus.PID)
}

// base := baseAgent("gemini:flash") // "gemini"
func baseAgent(agent string) string {
	return core.SplitN(agent, ":", 2)[0]
}

// codex: {total: 2, models: {gpt-5.4: 1}} → max 2 codex total, max 1 gpt-5.4
func (s *PrepSubsystem) canDispatchAgent(agent string) bool {
	var concurrency map[string]ConcurrencyLimit
	if s.ServiceRuntime != nil {
		configurationResult := s.Core().Config().Get("agents.concurrency")
		if configurationResult.OK {
			if configured, ok := configurationResult.Value.(map[string]ConcurrencyLimit); ok {
				concurrency = configured
			}
		}
	}
	if concurrency == nil {
		config := s.loadAgentsConfig()
		concurrency = config.Concurrency
	}

	base := baseAgent(agent)
	limit, ok := concurrency[base]
	if !ok || limit.Total <= 0 {
		if blocked, until := s.dailyRateLimitBackoff(agent); blocked {
			if s.backoff == nil {
				s.backoff = make(map[string]time.Time)
			}
			s.backoff[baseAgent(agent)] = until
			s.persistRuntimeState()
			return false
		}
		return true
	}

	if s.countRunningByAgent(base) >= limit.Total {
		return false
	}

	if limit.Models != nil {
		model := modelVariant(agent)
		if model != "" {
			if modelLimit, has := limit.Models[model]; has && modelLimit > 0 {
				if s.countRunningByModel(agent) >= modelLimit {
					return false
				}
			}
		}
	}

	if blocked, until := s.dailyRateLimitBackoff(agent); blocked {
		if s.backoff == nil {
			s.backoff = make(map[string]time.Time)
		}
		s.backoff[base] = until
		s.persistRuntimeState()
		return false
	}

	return true
}

func (s *PrepSubsystem) dailyRateLimitBackoff(agent string) (bool, time.Time) {
	rates := s.loadAgentsConfig().Rates
	rate, ok := rates[baseAgent(agent)]
	if !ok || rate.DailyLimit <= 0 {
		return false, time.Time{}
	}

	if s.dailyDispatchCount(agent) < rate.DailyLimit {
		return false, time.Time{}
	}

	resetHour, resetMin := 6, 0
	parts := core.Split(rate.ResetUTC, ":")
	if len(parts) >= 2 {
		if hour, err := strconv.Atoi(core.Trim(parts[0])); err == nil {
			resetHour = hour
		}
		if min, err := strconv.Atoi(core.Trim(parts[1])); err == nil {
			resetMin = min
		}
	}

	now := time.Now().UTC()
	resetToday := time.Date(now.Year(), now.Month(), now.Day(), resetHour, resetMin, 0, 0, time.UTC)
	if now.Before(resetToday) {
		resetToday = resetToday.AddDate(0, 0, -1)
	}
	nextReset := resetToday.AddDate(0, 0, 1)
	if nextReset.Before(now) {
		nextReset = now
	}
	return true, nextReset
}

func (s *PrepSubsystem) dailyDispatchCount(agent string) int {
	eventsPath := core.JoinPath(WorkspaceRoot(), "events.jsonl")
	result := fs.Read(eventsPath)
	if !result.OK {
		return 0
	}

	targetDay := time.Now().UTC().Format("2006-01-02")
	base := baseAgent(agent)
	count := 0

	for _, line := range core.Split(result.Value.(string), "\n") {
		line = core.Trim(line)
		if line == "" {
			continue
		}

		var event CompletionEvent
		if parseResult := core.JSONUnmarshalString(line, &event); !parseResult.OK {
			continue
		}
		if event.Type != "agent_started" {
			continue
		}
		if baseAgent(event.Agent) != base {
			continue
		}

		timestamp, err := time.Parse(time.RFC3339, event.Timestamp)
		if err != nil {
			continue
		}
		if timestamp.UTC().Format("2006-01-02") != targetDay {
			continue
		}

		count++
	}

	return count
}

// model := modelVariant("codex:gpt-5.4")
// core.Println(model) // "gpt-5.4"
func modelVariant(agent string) string {
	parts := core.SplitN(agent, ":", 2)
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}

// s.drainQueue()
func (s *PrepSubsystem) drainQueue() {
	if s.frozen {
		return
	}
	if s.ServiceRuntime != nil {
		s.Core().Lock("drain").Mutex.Lock()
		defer s.Core().Lock("drain").Mutex.Unlock()
	} else if s.drainCh != nil {
		s.drainCh <- struct{}{}
		defer func() { <-s.drainCh }()
	}

	for s.drainOne() {
	}
}

// spawned := s.drainOne()
func (s *PrepSubsystem) drainOne() bool {
	for _, statusPath := range WorkspaceStatusPaths() {
		workspaceDir := core.PathDir(statusPath)
		result := ReadStatusResult(workspaceDir)
		workspaceStatus, ok := workspaceStatusValue(result)
		if !ok || workspaceStatus.Status != "queued" {
			continue
		}

		if !s.canDispatchAgent(workspaceStatus.Agent) {
			continue
		}

		pool := baseAgent(workspaceStatus.Agent)
		if until, ok := s.backoff[pool]; ok && time.Now().Before(until) {
			continue
		}

		delay := s.delayForAgent(workspaceStatus.Agent)
		if delay > 0 {
			time.Sleep(delay)
		}

		if !s.canDispatchAgent(workspaceStatus.Agent) {
			continue
		}

		prompt := core.Concat("TASK: ", workspaceStatus.Task, "\n\nResume from where you left off. Read CODEX.md for conventions. Commit when done.")

		pid, processID, _, err := spawnAgent(s, workspaceStatus.Agent, prompt, workspaceDir)
		if err != nil {
			continue
		}

		workspaceStatus.Status = "running"
		workspaceStatus.PID = pid
		workspaceStatus.ProcessID = processID
		workspaceStatus.Runs++
		preserveStatusNote(workspaceDir, workspaceStatus) // keep VZ→OCI downgrade note (SP2.4)
		// Reported: the status file is what the monitor polls, so a silent write
		// failure leaves the workspace looking stuck indefinitely.
		if r := writeStatusResult(workspaceDir, workspaceStatus); !r.OK {
			core.Warn("agentic: failed to write workspace status", "reason", r.Value)
		}
		s.TrackWorkspace(WorkspaceName(workspaceDir), workspaceStatus)

		return true
	}

	return false
}
