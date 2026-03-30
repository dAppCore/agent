// SPDX-License-Identifier: EUPL-1.2

package runner

import (
	"strconv"
	"time"

	"dappco.re/go/agent/pkg/agentic"
	core "dappco.re/go/core"
	"gopkg.in/yaml.v3"
)

//	config := runner.DispatchConfig{
//	    DefaultAgent: "codex", DefaultTemplate: "coding", WorkspaceRoot: "/srv/core/workspace",
//	}
type DispatchConfig struct {
	DefaultAgent    string `yaml:"default_agent"`
	DefaultTemplate string `yaml:"default_template"`
	WorkspaceRoot   string `yaml:"workspace_root"`
}

//	rate := runner.RateConfig{
//	    ResetUTC: "06:00", DailyLimit: 200, SustainedDelay: 120, BurstWindow: 2, BurstDelay: 300,
//	}
type RateConfig struct {
	ResetUTC       string `yaml:"reset_utc"`
	DailyLimit     int    `yaml:"daily_limit"`
	MinDelay       int    `yaml:"min_delay"`
	SustainedDelay int    `yaml:"sustained_delay"`
	BurstWindow    int    `yaml:"burst_window"`
	BurstDelay     int    `yaml:"burst_delay"`
}

// ConcurrencyLimit supports both flat (int) and nested (map with total + per-model) formats.
//
//	claude: 1                       → Total=1, Models=nil
//	codex:                          → Total=5, Models={"gpt-5.4": 1}
//	  total: 5
//	  gpt-5.4: 1
type ConcurrencyLimit struct {
	Total  int
	Models map[string]int
}

// var limit ConcurrencyLimit
// _ = yaml.Unmarshal([]byte("total: 5\ngpt-5.4: 1\n"), &limit)
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

//	config := runner.AgentsConfig{
//	    Version: 1,
//	    Dispatch: runner.DispatchConfig{DefaultAgent: "codex", DefaultTemplate: "coding"},
//	}
type AgentsConfig struct {
	Version     int                         `yaml:"version"`
	Dispatch    DispatchConfig              `yaml:"dispatch"`
	Concurrency map[string]ConcurrencyLimit `yaml:"concurrency"`
	Rates       map[string]RateConfig       `yaml:"rates"`
}

// config := s.loadAgentsConfig()
// core.Println(config.Dispatch.DefaultAgent)
func (s *Service) loadAgentsConfig() *AgentsConfig {
	paths := []string{
		core.JoinPath(CoreRoot(), "agents.yaml"),
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

// if can, reason := s.canDispatchAgent("codex"); !can { _ = reason }
func (s *Service) canDispatchAgent(agent string) (bool, string) {
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
		return true, ""
	}

	running := s.countRunningByAgent(base)
	if running >= limit.Total {
		return false, core.Sprintf("total %d/%d", running, limit.Total)
	}

	if limit.Models != nil {
		model := modelVariant(agent)
		if model != "" {
			modelRunning := s.countRunningByModel(agent)
			if modelLimit, has := limit.Models[model]; has && modelLimit > 0 {
				if modelRunning >= modelLimit {
					return false, core.Sprintf("model %s %d/%d", model, modelRunning, modelLimit)
				}
			}
		}
	}

	return true, ""
}

// n := s.countRunningByAgent("codex")
func (s *Service) countRunningByAgent(agent string) int {
	var runtime *core.Core
	if s.ServiceRuntime != nil {
		runtime = s.Core()
	}
	count := 0
	s.workspaces.Each(func(_ string, workspaceStatus *WorkspaceStatus) {
		if workspaceStatus.Status != "running" || baseAgent(workspaceStatus.Agent) != agent {
			return
		}
		switch {
		case workspaceStatus.PID < 0:
			count++
		case workspaceStatus.PID > 0 && agentic.ProcessAlive(runtime, "", workspaceStatus.PID):
			count++
		}
	})
	return count
}

// n := s.countRunningByModel("codex:gpt-5.4")
func (s *Service) countRunningByModel(agent string) int {
	var runtime *core.Core
	if s.ServiceRuntime != nil {
		runtime = s.Core()
	}
	count := 0
	s.workspaces.Each(func(_ string, workspaceStatus *WorkspaceStatus) {
		if workspaceStatus.Status != "running" || workspaceStatus.Agent != agent {
			return
		}
		switch {
		case workspaceStatus.PID < 0:
			count++
		case workspaceStatus.PID > 0 && agentic.ProcessAlive(runtime, "", workspaceStatus.PID):
			count++
		}
	})
	return count
}

// s.drainQueue()
func (s *Service) drainQueue() {
	if s.frozen {
		return
	}
	s.drainMu.Lock()
	defer s.drainMu.Unlock()

	for s.drainOne() {
		// keep filling slots
	}
}

func (s *Service) drainOne() bool {
	for _, statusPath := range agentic.WorkspaceStatusPaths() {
		workspaceDir := core.PathDir(statusPath)
		statusResult := ReadStatusResult(workspaceDir)
		if !statusResult.OK {
			continue
		}
		workspaceStatus, ok := statusResult.Value.(*WorkspaceStatus)
		if !ok || workspaceStatus == nil || workspaceStatus.Status != "queued" {
			continue
		}

		if can, _ := s.canDispatchAgent(workspaceStatus.Agent); !can {
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

		if can, _ := s.canDispatchAgent(workspaceStatus.Agent); !can {
			continue
		}

		// Ask agentic to spawn — runner owns the gate,
		// agentic owns the actual process launch.
		// Workspace name is relative path from workspace root (e.g. "core/go-ai/dev")
		workspaceName := agentic.WorkspaceName(workspaceDir)
		core.Info("drainOne: found queued workspace", "workspace", workspaceName, "agent", workspaceStatus.Agent)

		// Spawn directly — agentic is a Core service, use ServiceFor to get it
		if s.ServiceRuntime == nil {
			continue
		}
		type spawner interface {
			SpawnFromQueue(agent, prompt, workspaceDir string) core.Result
		}
		agenticService, ok := core.ServiceFor[spawner](s.Core(), "agentic")
		if !ok {
			core.Error("drainOne: agentic service not found")
			continue
		}
		prompt := core.Concat("TASK: ", workspaceStatus.Task, "\n\nResume from where you left off. Read CODEX.md for conventions. Commit when done.")
		spawnResult := agenticService.SpawnFromQueue(workspaceStatus.Agent, prompt, workspaceDir)
		if !spawnResult.OK {
			core.Error("drainOne: spawn failed", "workspace", workspaceName, "reason", core.Sprint(spawnResult.Value))
			continue
		}
		pid, ok := spawnResult.Value.(int)
		if !ok {
			core.Error("drainOne: spawn returned non-int pid", "workspace", workspaceName)
			continue
		}

		// Only mark running AFTER successful spawn
		workspaceStatus.Status = "running"
		workspaceStatus.PID = pid
		workspaceStatus.Runs++
		if writeResult := WriteStatus(workspaceDir, workspaceStatus); !writeResult.OK {
			core.Error("drainOne: failed to write workspace status", "workspace", workspaceName, "reason", core.Sprint(writeResult.Value))
			continue
		}
		s.TrackWorkspace(workspaceName, workspaceStatus)
		core.Info("drainOne: spawned", "pid", pid, "workspace", workspaceName)

		return true
	}

	return false
}

func (s *Service) delayForAgent(agent string) time.Duration {
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

func baseAgent(agent string) string {
	return core.SplitN(agent, ":", 2)[0]
}

func modelVariant(agent string) string {
	parts := core.SplitN(agent, ":", 2)
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}
