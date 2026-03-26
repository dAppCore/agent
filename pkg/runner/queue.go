// SPDX-License-Identifier: EUPL-1.2

package runner

import (
	"strconv"
	"syscall"
	"time"

	core "dappco.re/go/core"
	"gopkg.in/yaml.v3"
)

// DispatchConfig controls agent dispatch behaviour.
//
//	cfg := runner.DispatchConfig{DefaultAgent: "claude", DefaultTemplate: "coding"}
type DispatchConfig struct {
	DefaultAgent    string `yaml:"default_agent"`
	DefaultTemplate string `yaml:"default_template"`
	WorkspaceRoot   string `yaml:"workspace_root"`
}

// RateConfig controls pacing between task dispatches.
//
//	rate := runner.RateConfig{ResetUTC: "06:00", SustainedDelay: 120}
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

// UnmarshalYAML handles both int and map forms.
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

// AgentsConfig is the root of agents.yaml.
type AgentsConfig struct {
	Version     int                          `yaml:"version"`
	Dispatch    DispatchConfig               `yaml:"dispatch"`
	Concurrency map[string]ConcurrencyLimit  `yaml:"concurrency"`
	Rates       map[string]RateConfig        `yaml:"rates"`
}

// loadAgentsConfig reads agents.yaml from known paths.
func (s *Service) loadAgentsConfig() *AgentsConfig {
	paths := []string{
		core.JoinPath(CoreRoot(), "agents.yaml"),
	}
	for _, path := range paths {
		r := fs.Read(path)
		if !r.OK {
			continue
		}
		var cfg AgentsConfig
		if err := yaml.Unmarshal([]byte(r.Value.(string)), &cfg); err != nil {
			continue
		}
		return &cfg
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

// canDispatchAgent checks both pool-level and per-model concurrency limits.
//
//	if !s.canDispatchAgent("codex") { /* queue it */ }
func (s *Service) canDispatchAgent(agent string) bool {
	var concurrency map[string]ConcurrencyLimit
	if s.ServiceRuntime != nil {
		r := s.Core().Config().Get("agents.concurrency")
		if r.OK {
			concurrency, _ = r.Value.(map[string]ConcurrencyLimit)
		}
	}
	if concurrency == nil {
		cfg := s.loadAgentsConfig()
		concurrency = cfg.Concurrency
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

// countRunningByAgent counts running workspaces using the in-memory Registry.
//
//	n := s.countRunningByAgent("codex")
func (s *Service) countRunningByAgent(agent string) int {
	if s.workspaces != nil && s.workspaces.Len() > 0 {
		count := 0
		s.workspaces.Each(func(_ string, st *WorkspaceStatus) {
			if st.Status == "running" && baseAgent(st.Agent) == agent {
				// PID < 0 = reservation (pending spawn), always count
				// PID > 0 = verify process is alive
				if st.PID < 0 || (st.PID > 0 && syscall.Kill(st.PID, 0) == nil) {
					count++
				}
			}
		})
		return count
	}
	return s.countRunningByAgentDisk(agent)
}

func (s *Service) countRunningByAgentDisk(agent string) int {
	wsRoot := WorkspaceRoot()
	old := core.PathGlob(core.JoinPath(wsRoot, "*", "status.json"))
	deep := core.PathGlob(core.JoinPath(wsRoot, "*", "*", "*", "status.json"))

	count := 0
	for _, statusPath := range append(old, deep...) {
		st, err := ReadStatus(core.PathDir(statusPath))
		if err != nil || st.Status != "running" {
			continue
		}
		if baseAgent(st.Agent) != agent {
			continue
		}
		if st.PID > 0 && syscall.Kill(st.PID, 0) == nil {
			count++
		}
	}
	return count
}

// countRunningByModel counts running workspaces for a specific agent:model.
func (s *Service) countRunningByModel(agent string) int {
	if s.workspaces != nil && s.workspaces.Len() > 0 {
		count := 0
		s.workspaces.Each(func(_ string, st *WorkspaceStatus) {
			if st.Status == "running" && st.Agent == agent {
				if st.PID < 0 || (st.PID > 0 && syscall.Kill(st.PID, 0) == nil) {
					count++
				}
			}
		})
		return count
	}

	wsRoot := WorkspaceRoot()
	old := core.PathGlob(core.JoinPath(wsRoot, "*", "status.json"))
	deep := core.PathGlob(core.JoinPath(wsRoot, "*", "*", "*", "status.json"))

	count := 0
	for _, statusPath := range append(old, deep...) {
		st, err := ReadStatus(core.PathDir(statusPath))
		if err != nil || st.Status != "running" {
			continue
		}
		if st.Agent != agent {
			continue
		}
		if st.PID > 0 && syscall.Kill(st.PID, 0) == nil {
			count++
		}
	}
	return count
}

// drainQueue fills available concurrency slots from queued workspaces.
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
	wsRoot := WorkspaceRoot()
	old := core.PathGlob(core.JoinPath(wsRoot, "*", "status.json"))
	deep := core.PathGlob(core.JoinPath(wsRoot, "*", "*", "*", "status.json"))

	for _, statusPath := range append(old, deep...) {
		wsDir := core.PathDir(statusPath)
		st, err := ReadStatus(wsDir)
		if err != nil || st.Status != "queued" {
			continue
		}

		if !s.canDispatchAgent(st.Agent) {
			continue
		}

		pool := baseAgent(st.Agent)
		if until, ok := s.backoff[pool]; ok && time.Now().Before(until) {
			continue
		}

		delay := s.delayForAgent(st.Agent)
		if delay > 0 {
			time.Sleep(delay)
		}

		if !s.canDispatchAgent(st.Agent) {
			continue
		}

		// Ask agentic to spawn — runner doesn't own the spawn logic,
		// just the gate. Send IPC to trigger the actual spawn.
		// Workspace name is relative path from workspace root (e.g. "core/go-ai/dev")
		wsRoot := WorkspaceRoot()
		wsName := wsDir
		if len(wsDir) > len(wsRoot)+1 {
			wsName = wsDir[len(wsRoot)+1:]
		}
		core.Info("drainOne: found queued workspace", "workspace", wsName, "agent", st.Agent)

		// Spawn directly — agentic is a Core service, use ServiceFor to get it
		if s.ServiceRuntime == nil {
			continue
		}
		type spawner interface {
			SpawnFromQueue(agent, prompt, wsDir string) (int, error)
		}
		prep, ok := core.ServiceFor[spawner](s.Core(), "agentic")
		if !ok {
			core.Error("drainOne: agentic service not found")
			continue
		}
		prompt := core.Concat("TASK: ", st.Task, "\n\nResume from where you left off. Read CODEX.md for conventions. Commit when done.")
		pid, err := prep.SpawnFromQueue(st.Agent, prompt, wsDir)
		if err != nil {
			core.Error("drainOne: spawn failed", "err", err)
			continue
		}

		// Only mark running AFTER successful spawn
		st.Status = "running"
		st.PID = pid
		st.Runs++
		WriteStatus(wsDir, st)
		s.TrackWorkspace(wsName, st)
		core.Info("drainOne: spawned", "pid", pid, "workspace", wsName)

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
		cfg := s.loadAgentsConfig()
		rates = cfg.Rates
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

// --- Helpers ---

func baseAgent(agent string) string {
	if core.Contains(agent, "codex-spark") {
		return "codex-spark"
	}
	return core.SplitN(agent, ":", 2)[0]
}

func modelVariant(agent string) string {
	parts := core.SplitN(agent, ":", 2)
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}
