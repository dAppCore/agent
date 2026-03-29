// SPDX-License-Identifier: EUPL-1.2

// Package runner is the agent dispatch service.
// Owns concurrency, queue drain, workspace lifecycle, and frozen state.
// Communicates with other services via Core IPC — Actions, Tasks, and Messages.
//
//	core.New(core.WithService(runner.Register))
package runner

import (
	"context"
	"sync"
	"syscall"
	"time"

	"dappco.re/go/agent/pkg/agentic"
	"dappco.re/go/agent/pkg/messages"
	core "dappco.re/go/core"
)

// Options configures the runner service.
//
//	opts := runner.Options{}
type Options struct{}

// Service is the agent dispatch runner.
// Manages concurrency limits, queue drain, workspace lifecycle, and frozen state.
// All dispatch requests — MCP tool, CLI, or IPC — go through this service.
//
//	svc := runner.New()
//	svc.TrackWorkspace("core/go-io/task-5", &runner.WorkspaceStatus{Status: "running", Agent: "codex"})
type Service struct {
	*core.ServiceRuntime[Options]
	dispatchMu sync.Mutex
	drainMu    sync.Mutex
	pokeCh     chan struct{}
	frozen     bool
	backoff    map[string]time.Time
	failCount  map[string]int
	workspaces *core.Registry[*WorkspaceStatus]
}

type channelSender interface {
	ChannelSend(ctx context.Context, channel string, data any)
}

// New creates a runner service.
//
//	svc := runner.New()
func New() *Service {
	return &Service{
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
		workspaces: core.NewRegistry[*WorkspaceStatus](),
	}
}

// Register is the service factory for core.WithService.
//
//	core.New(core.WithService(runner.Register))
func Register(c *core.Core) core.Result {
	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})

	// Load agents config
	cfg := svc.loadAgentsConfig()
	c.Config().Set("agents.concurrency", cfg.Concurrency)
	c.Config().Set("agents.rates", cfg.Rates)
	c.Config().Set("agents.dispatch", cfg.Dispatch)
	c.Config().Set("agents.config_path", core.JoinPath(CoreRoot(), "agents.yaml"))
	codexTotal := 0
	if cl, ok := cfg.Concurrency["codex"]; ok {
		codexTotal = cl.Total
	}
	c.Config().Set("agents.codex_limit_debug", codexTotal)

	return core.Result{Value: svc, OK: true}
}

// OnStartup registers Actions and starts the queue runner.
//
//	c.Action("runner.dispatch").Run(ctx, core.NewOptions(
//		core.Option{Key: "repo", Value: "go-io"},
//		core.Option{Key: "agent", Value: "codex"},
//	))
//	c.Action("runner.status").Run(ctx, core.NewOptions())
func (s *Service) OnStartup(ctx context.Context) core.Result {
	c := s.Core()

	// Actions — the runner's capability map
	c.Action("runner.dispatch", s.actionDispatch).Description = "Dispatch a subagent (checks frozen + concurrency)"
	c.Action("runner.status", s.actionStatus).Description = "Query workspace status"
	c.Action("runner.start", s.actionStart).Description = "Unfreeze dispatch queue"
	c.Action("runner.stop", s.actionStop).Description = "Freeze dispatch queue (graceful)"
	c.Action("runner.kill", s.actionKill).Description = "Kill all running agents (hard stop)"
	c.Action("runner.poke", s.actionPoke).Description = "Drain next queued task"

	// Hydrate workspace registry from disk
	s.hydrateWorkspaces()

	// QUERY handler — workspace state queries
	c.RegisterQuery(s.handleWorkspaceQuery)

	// Start the background queue runner
	s.startRunner()

	return core.Result{OK: true}
}

// OnShutdown freezes the queue.
//
//	r := svc.OnShutdown(context.Background())
//	if r.OK {
//		core.Println(svc.IsFrozen())
//	}
func (s *Service) OnShutdown(_ context.Context) core.Result {
	s.frozen = true
	return core.Result{OK: true}
}

// HandleIPCEvents applies runner side-effects for IPC messages.
//
//	svc.HandleIPCEvents(c, messages.PokeQueue{})
//	svc.HandleIPCEvents(c, messages.AgentCompleted{
//		Agent: "codex", Repo: "go-io", Workspace: "core/go-io/task-5", Status: "completed",
//	})
func (s *Service) HandleIPCEvents(c *core.Core, msg core.Message) core.Result {
	switch ev := msg.(type) {
	case messages.AgentStarted:
		base := baseAgent(ev.Agent)
		running := s.countRunningByAgent(base)
		var limit int
		r := c.Config().Get("agents.concurrency")
		if r.OK {
			if concurrency, ok := r.Value.(map[string]ConcurrencyLimit); ok {
				if cl, has := concurrency[base]; has {
					limit = cl.Total
				}
			}
		}
		notification := &AgentNotification{
			Status:    "started",
			Repo:      ev.Repo,
			Agent:     ev.Agent,
			Workspace: ev.Workspace,
			Running:   running,
			Limit:     limit,
		}
		if notifier, ok := core.ServiceFor[channelSender](c, "mcp"); ok {
			notifier.ChannelSend(context.Background(), "agent.status", notification)
		}

	case messages.AgentCompleted:
		// Update workspace status in Registry so concurrency count drops
		if ev.Workspace != "" {
			if r := s.workspaces.Get(ev.Workspace); r.OK {
				if st, ok := r.Value.(*WorkspaceStatus); ok && st.Status == "running" {
					st.Status = ev.Status
					st.PID = 0
				}
			}
		} else {
			s.workspaces.Each(func(_ string, st *WorkspaceStatus) {
				if st.Repo == ev.Repo && st.Status == "running" {
					st.Status = ev.Status
					st.PID = 0
				}
			})
		}
		cBase := baseAgent(ev.Agent)
		cRunning := s.countRunningByAgent(cBase)
		var cLimit int
		cr := c.Config().Get("agents.concurrency")
		if cr.OK {
			if concurrency, ok := cr.Value.(map[string]ConcurrencyLimit); ok {
				if cl, has := concurrency[cBase]; has {
					cLimit = cl.Total
				}
			}
		}
		notification := &AgentNotification{
			Status:    ev.Status,
			Repo:      ev.Repo,
			Agent:     ev.Agent,
			Workspace: ev.Workspace,
			Running:   cRunning,
			Limit:     cLimit,
		}
		if notifier, ok := core.ServiceFor[channelSender](c, "mcp"); ok {
			notifier.ChannelSend(context.Background(), "agent.status", notification)
		}
		s.Poke()

	case messages.PokeQueue:
		s.drainQueue()
		_ = ev
	}
	return core.Result{OK: true}
}

// IsFrozen returns whether dispatch is currently frozen.
//
//	if s.IsFrozen() { return "queue is frozen" }
func (s *Service) IsFrozen() bool {
	return s.frozen
}

// Poke signals the runner to check the queue immediately.
//
//	s.Poke()
func (s *Service) Poke() {
	if s.pokeCh == nil {
		return
	}
	select {
	case s.pokeCh <- struct{}{}:
	default:
	}
}

// TrackWorkspace registers or updates a workspace in the in-memory Registry.
// Accepts any status type — agentic passes *agentic.WorkspaceStatus,
// runner stores its own *WorkspaceStatus copy.
//
//	s.TrackWorkspace("core/go-io/task-5", st)
func (s *Service) TrackWorkspace(name string, st any) {
	if s.workspaces == nil {
		return
	}
	// Convert from agentic's type to runner's via JSON round-trip
	json := core.JSONMarshalString(st)
	var ws WorkspaceStatus
	if r := core.JSONUnmarshalString(json, &ws); r.OK {
		s.workspaces.Set(name, &ws)
		// Remove pending reservation now that the real workspace is tracked
		s.workspaces.Delete(core.Concat("pending/", ws.Repo))
	}
}

// Workspaces returns the workspace Registry.
//
//	s.Workspaces().Each(func(name string, st *WorkspaceStatus) { ... })
func (s *Service) Workspaces() *core.Registry[*WorkspaceStatus] {
	return s.workspaces
}

// handleWorkspaceQuery answers workspace state queries from Core QUERY calls.
//
//	r := c.QUERY(runner.WorkspaceQuery{Name: "core/go-io/task-42"})
//	r := c.QUERY(runner.WorkspaceQuery{Status: "running"})
func (s *Service) handleWorkspaceQuery(_ *core.Core, q core.Query) core.Result {
	wq, ok := q.(WorkspaceQuery)
	if !ok {
		return core.Result{}
	}
	if wq.Name != "" {
		return s.workspaces.Get(wq.Name)
	}
	if wq.Status != "" {
		var names []string
		s.workspaces.Each(func(name string, st *WorkspaceStatus) {
			if st.Status == wq.Status {
				names = append(names, name)
			}
		})
		return core.Result{Value: names, OK: true}
	}
	return core.Result{Value: s.workspaces, OK: true}
}

// --- Actions ---

func (s *Service) actionDispatch(_ context.Context, opts core.Options) core.Result {
	if s.frozen {
		return core.Result{Value: "queue is frozen", OK: false}
	}

	agent := opts.String("agent")
	if agent == "" {
		agent = "codex"
	}
	repo := opts.String("repo")

	s.dispatchMu.Lock()
	defer s.dispatchMu.Unlock()

	can, reason := s.canDispatchAgent(agent)
	if !can {
		return core.Result{Value: core.Concat("queued — ", reason), OK: false}
	}

	// Reserve the slot immediately — before returning to agentic.
	name := core.Concat("pending/", repo)
	s.workspaces.Set(name, &WorkspaceStatus{
		Status: "running",
		Agent:  agent,
		Repo:   repo,
		PID:    -1,
	})

	return core.Result{OK: true}
}

func (s *Service) actionStatus(_ context.Context, _ core.Options) core.Result {
	running, queued, completed, failed := 0, 0, 0, 0
	s.workspaces.Each(func(_ string, st *WorkspaceStatus) {
		switch st.Status {
		case "running":
			running++
		case "queued":
			queued++
		case "completed", "merged", "ready-for-review":
			completed++
		case "failed", "blocked":
			failed++
		}
	})
	return core.Result{Value: map[string]int{
		"running": running, "queued": queued,
		"completed": completed, "failed": failed,
		"total": running + queued + completed + failed,
	}, OK: true}
}

func (s *Service) actionStart(_ context.Context, _ core.Options) core.Result {
	s.frozen = false
	s.Poke()
	return core.Result{Value: "dispatch started", OK: true}
}

func (s *Service) actionStop(_ context.Context, _ core.Options) core.Result {
	s.frozen = true
	return core.Result{Value: "queue frozen", OK: true}
}

func (s *Service) actionKill(_ context.Context, _ core.Options) core.Result {
	s.frozen = true
	killed := 0
	s.workspaces.Each(func(_ string, st *WorkspaceStatus) {
		if st.Status == "running" && st.PID > 0 {
			if syscall.Kill(st.PID, syscall.SIGTERM) == nil {
				killed++
			}
			st.Status = "failed"
			st.PID = 0
		}
		if st.Status == "queued" {
			st.Status = "failed"
		}
	})
	return core.Result{Value: core.Sprintf("killed %d agents", killed), OK: true}
}

func (s *Service) actionPoke(_ context.Context, _ core.Options) core.Result {
	s.drainQueue()
	return core.Result{OK: true}
}

// --- Queue runner ---

func (s *Service) startRunner() {
	s.pokeCh = make(chan struct{}, 1)

	if core.Env("CORE_AGENT_DISPATCH") == "1" {
		s.frozen = false
	} else {
		s.frozen = true
	}

	go s.runLoop()
}

func (s *Service) runLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.drainQueue()
		case <-s.pokeCh:
			s.drainQueue()
		}
	}
}

// --- Workspace hydration ---

func (s *Service) hydrateWorkspaces() {
	if s.workspaces == nil {
		s.workspaces = core.NewRegistry[*WorkspaceStatus]()
	}
	for _, path := range agentic.WorkspaceStatusPaths() {
		wsDir := core.PathDir(path)
		st, err := ReadStatus(wsDir)
		if err != nil || st == nil {
			continue
		}
		// Re-queue running agents on restart — process is dead, re-dispatch
		if st.Status == "running" {
			st.Status = "queued"
		}
		s.workspaces.Set(agentic.WorkspaceName(wsDir), st)
	}
}

// --- Types ---

// AgentNotification is the channel payload sent on `agent.status`.
//
//	n := runner.AgentNotification{
//		Status: "started", Repo: "go-io", Agent: "codex", Workspace: "core/go-io/task-5", Running: 1, Limit: 2,
//	}
//
// Field order is guaranteed by json tags so truncated notifications still show
// status and repo first.
type AgentNotification struct {
	Status    string `json:"status"`
	Repo      string `json:"repo"`
	Agent     string `json:"agent"`
	Workspace string `json:"workspace"`
	Running   int    `json:"running"`
	Limit     int    `json:"limit"`
}

// WorkspaceQuery is the QUERY type for workspace lookups.
//
//	r := c.QUERY(runner.WorkspaceQuery{Status: "running"})
type WorkspaceQuery struct {
	Name   string
	Status string
}

// WorkspaceStatus tracks the state of an agent workspace.
//
//	st := &runner.WorkspaceStatus{Status: "running", Agent: "codex", Repo: "go-io", PID: 12345}
type WorkspaceStatus struct {
	Status    string    `json:"status"`
	Agent     string    `json:"agent"`
	Repo      string    `json:"repo"`
	Org       string    `json:"org,omitempty"`
	Task      string    `json:"task,omitempty"`
	Branch    string    `json:"branch,omitempty"`
	PID       int       `json:"pid,omitempty"`
	Question  string    `json:"question,omitempty"`
	PRURL     string    `json:"pr_url,omitempty"`
	StartedAt time.Time `json:"started_at"`
	Runs      int       `json:"runs"`
}
