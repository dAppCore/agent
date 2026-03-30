// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"strconv"
	"time"

	core "dappco.re/go/core"
	"gopkg.in/yaml.v3"
)

// DispatchConfig controls agent dispatch behaviour.
//
//	config := agentic.DispatchConfig{DefaultAgent: "claude", DefaultTemplate: "coding"}
type DispatchConfig struct {
	DefaultAgent    string `yaml:"default_agent"`
	DefaultTemplate string `yaml:"default_template"`
	WorkspaceRoot   string `yaml:"workspace_root"`
}

// RateConfig controls pacing between task dispatches.
//
//	rate := agentic.RateConfig{ResetUTC: "06:00", SustainedDelay: 120, BurstWindow: 2, BurstDelay: 15}
type RateConfig struct {
	ResetUTC       string `yaml:"reset_utc"`       // Daily quota reset time (UTC), e.g. "06:00"
	DailyLimit     int    `yaml:"daily_limit"`     // Max requests per day (0 = unknown)
	MinDelay       int    `yaml:"min_delay"`       // Minimum seconds between task starts
	SustainedDelay int    `yaml:"sustained_delay"` // Delay when pacing for full-day use
	BurstWindow    int    `yaml:"burst_window"`    // Hours before reset where burst kicks in
	BurstDelay     int    `yaml:"burst_delay"`     // Delay during burst window
}

// ConcurrencyLimit supports both flat (int) and nested (map with total + per-model) formats.
//
//	claude: 1                       → Total=1, Models=nil
//	codex:                          → Total=2, Models={"gpt-5.4": 1, "gpt-5.3-codex-spark": 1}
//	  total: 2
//	  gpt-5.4: 1
//	  gpt-5.3-codex-spark: 1
type ConcurrencyLimit struct {
	Total  int
	Models map[string]int
}

// UnmarshalYAML handles both int and map forms for concurrency limits.
//
//	var limit ConcurrencyLimit
//	_ = yaml.Unmarshal([]byte("total: 2\ngpt-5.4: 1\n"), &limit)
func (c *ConcurrencyLimit) UnmarshalYAML(value *yaml.Node) error {
	// Try int first
	var n int
	if err := value.Decode(&n); err == nil {
		c.Total = n
		return nil
	}
	// Try map
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

// AgentsConfig is the root of config/agents.yaml.
//
//	config := agentic.AgentsConfig{Version: 1, Dispatch: agentic.DispatchConfig{DefaultAgent: "claude"}}
type AgentsConfig struct {
	Version     int                         `yaml:"version"`
	Dispatch    DispatchConfig              `yaml:"dispatch"`
	Concurrency map[string]ConcurrencyLimit `yaml:"concurrency"`
	Rates       map[string]RateConfig       `yaml:"rates"`
}

// loadAgentsConfig reads config/agents.yaml from the code path.
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

// delayForAgent calculates how long to wait before spawning the next task
// for a given agent type, based on rate config and time of day.
func (s *PrepSubsystem) delayForAgent(agent string) time.Duration {
	// Read from Core Config (loaded once at registration)
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

	// Parse reset time
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
		// Reset hasn't happened yet today — reset was yesterday
		resetToday = resetToday.AddDate(0, 0, -1)
	}
	nextReset := resetToday.AddDate(0, 0, 1)
	hoursUntilReset := nextReset.Sub(now).Hours()

	// Burst mode: if within burst window of reset, use burst delay
	if rate.BurstWindow > 0 && hoursUntilReset <= float64(rate.BurstWindow) {
		return time.Duration(rate.BurstDelay) * time.Second
	}

	// Sustained mode
	return time.Duration(rate.SustainedDelay) * time.Second
}

// countRunningByAgent counts running workspaces for a specific agent type
// using the in-memory Registry. Falls back to disk scan if Registry is empty.
//
//	n := s.countRunningByAgent("codex") // counts all codex:* variants
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

	// Fallback: scan disk (cold start before hydration)
	return s.countRunningByAgentDisk(runtime, agent)
}

// countRunningByAgentDisk scans workspace status.json files on disk.
// Used only as fallback before Registry hydration completes.
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

// countRunningByModel counts running workspaces for a specific agent:model string
// using the in-memory Registry.
//
//	n := s.countRunningByModel("codex:gpt-5.4") // counts only that model
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

	// Fallback: scan disk
	return s.countRunningByModelDisk(runtime, agent)
}

// countRunningByModelDisk scans workspace status.json files on disk.
// Used only as fallback before Registry hydration completes.
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

// baseAgent strips the model variant (gemini:flash → gemini).
func baseAgent(agent string) string {
	return core.SplitN(agent, ":", 2)[0]
}

// canDispatchAgent checks both pool-level and per-model concurrency limits.
//
//	codex: {total: 2, models: {gpt-5.4: 1}} → max 2 codex total, max 1 gpt-5.4
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

	// Check pool total
	if s.countRunningByAgent(base) >= limit.Total {
		return false
	}

	// Check per-model limit if configured
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

// modelVariant extracts the model name from an agent string.
//
//	codex:gpt-5.4 → gpt-5.4
//	codex:gpt-5.3-codex-spark → gpt-5.3-codex-spark
//	claude → ""
func modelVariant(agent string) string {
	parts := core.SplitN(agent, ":", 2)
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}

// drainQueue fills all available concurrency slots from queued workspaces.
// Serialised via c.Lock("drain") when Core is available, falls back to local mutex.
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
		// keep filling slots
	}
}

// drainOne finds the oldest queued workspace and spawns it if a slot is available.
// Returns true if a task was spawned, false if nothing to do.
func (s *PrepSubsystem) drainOne() bool {
	for _, statusPath := range WorkspaceStatusPaths() {
		wsDir := core.PathDir(statusPath)
		result := ReadStatusResult(wsDir)
		workspaceStatus, ok := workspaceStatusValue(result)
		if !ok || workspaceStatus.Status != "queued" {
			continue
		}

		if !s.canDispatchAgent(workspaceStatus.Agent) {
			continue
		}

		// Skip if agent pool is in rate-limit backoff
		pool := baseAgent(workspaceStatus.Agent)
		if until, ok := s.backoff[pool]; ok && time.Now().Before(until) {
			continue
		}

		// Apply rate delay before spawning
		delay := s.delayForAgent(workspaceStatus.Agent)
		if delay > 0 {
			time.Sleep(delay)
		}

		// Re-check concurrency after delay (another task may have started)
		if !s.canDispatchAgent(workspaceStatus.Agent) {
			continue
		}

		prompt := core.Concat("TASK: ", workspaceStatus.Task, "\n\nResume from where you left off. Read CODEX.md for conventions. Commit when done.")

		pid, processID, _, err := s.spawnAgent(workspaceStatus.Agent, prompt, wsDir)
		if err != nil {
			continue
		}

		workspaceStatus.Status = "running"
		workspaceStatus.PID = pid
		workspaceStatus.ProcessID = processID
		workspaceStatus.Runs++
		writeStatusResult(wsDir, workspaceStatus)
		s.TrackWorkspace(WorkspaceName(wsDir), workspaceStatus)

		return true
	}

	return false
}
