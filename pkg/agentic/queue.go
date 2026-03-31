// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"strconv"
	"time"

	core "dappco.re/go/core"
	"gopkg.in/yaml.v3"
)

// config := agentic.DispatchConfig{DefaultAgent: "claude", DefaultTemplate: "coding"}
type DispatchConfig struct {
	DefaultAgent    string `yaml:"default_agent"`
	DefaultTemplate string `yaml:"default_template"`
	WorkspaceRoot   string `yaml:"workspace_root"`
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
type ConcurrencyLimit struct {
	Total  int
	Models map[string]int
}

// var limit ConcurrencyLimit
// _ = yaml.Unmarshal([]byte("total: 2\ngpt-5.4: 1\n"), &limit)
func (c *ConcurrencyLimit) UnmarshalYAML(value *yaml.Node) error {
	var n int
	if err := value.Decode(&n); err == nil {
		c.Total = n
		return nil
	}
	var m map[string]int
	if err := value.Decode(&m); err != nil {
		return err
	}
	c.Total = m["total"]
	c.Models = make(map[string]int)
	for k, v := range m {
		if k != "total" {
			c.Models[k] = v
		}
	}
	return nil
}

// config := agentic.AgentsConfig{Version: 1, Dispatch: agentic.DispatchConfig{DefaultAgent: "claude"}}
type AgentsConfig struct {
	Version     int                         `yaml:"version"`
	Dispatch    DispatchConfig              `yaml:"dispatch"`
	Concurrency map[string]ConcurrencyLimit `yaml:"concurrency"`
	Rates       map[string]RateConfig       `yaml:"rates"`
}

// config := s.loadAgentsConfig()
func (s *PrepSubsystem) loadAgentsConfig() *AgentsConfig {
	paths := []string{
		core.JoinPath(CoreRoot(), "agents.yaml"),
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
		return &config
	}

	return &AgentsConfig{
		Dispatch: DispatchConfig{
			DefaultAgent:    "claude",
			DefaultTemplate: "coding",
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
		rates, _ = s.Core().Config().Get("agents.rates").Value.(map[string]RateConfig)
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

	if rate.BurstWindow > 0 && hoursUntilReset <= float64(rate.BurstWindow) {
		return time.Duration(rate.BurstDelay) * time.Second
	}

	return time.Duration(rate.SustainedDelay) * time.Second
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
			if workspaceStatus.Status == "running" && baseAgent(workspaceStatus.Agent) == agent && ProcessAlive(runtime, workspaceStatus.ProcessID, workspaceStatus.PID) {
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
		if ProcessAlive(runtime, workspaceStatus.ProcessID, workspaceStatus.PID) {
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
			if workspaceStatus.Status == "running" && workspaceStatus.Agent == agent && ProcessAlive(runtime, workspaceStatus.ProcessID, workspaceStatus.PID) {
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
		if ProcessAlive(runtime, workspaceStatus.ProcessID, workspaceStatus.PID) {
			count++
		}
	}
	return count
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
			concurrency, _ = configurationResult.Value.(map[string]ConcurrencyLimit)
		}
	}
	if concurrency == nil {
		config := s.loadAgentsConfig()
		concurrency = config.Concurrency
	}

	base := baseAgent(agent)
	limit, ok := concurrency[base]
	if !ok || limit.Total <= 0 {
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

	return true
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
	} else {
		s.drainMu.Lock()
		defer s.drainMu.Unlock()
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

		pid, processID, _, err := s.spawnAgent(workspaceStatus.Agent, prompt, workspaceDir)
		if err != nil {
			continue
		}

		workspaceStatus.Status = "running"
		workspaceStatus.PID = pid
		workspaceStatus.ProcessID = processID
		workspaceStatus.Runs++
		writeStatusResult(workspaceDir, workspaceStatus)
		s.TrackWorkspace(WorkspaceName(workspaceDir), workspaceStatus)

		return true
	}

	return false
}
