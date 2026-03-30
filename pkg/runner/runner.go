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
	"time"

	"dappco.re/go/agent/pkg/agentic"
	"dappco.re/go/agent/pkg/messages"
	core "dappco.re/go/core"
)

// Options configures the runner service.
//
//	options := runner.Options{}
type Options struct{}

// Service is the agent dispatch runner.
// Manages concurrency limits, queue drain, workspace lifecycle, and frozen state.
// All dispatch requests — MCP tool, CLI, or IPC — go through this service.
//
//	service := runner.New()
//	service.TrackWorkspace("core/go-io/task-5", &runner.WorkspaceStatus{Status: "running", Agent: "codex"})
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
//	service := runner.New()
func New() *Service {
	return &Service{
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
		workspaces: core.NewRegistry[*WorkspaceStatus](),
	}
}

//	c := core.New(core.WithService(runner.Register))
//	service, _ := core.ServiceFor[*runner.Service](c, "runner")
func Register(coreApp *core.Core) core.Result {
	service := New()
	service.ServiceRuntime = core.NewServiceRuntime(coreApp, Options{})

	// Load agents config
	config := service.loadAgentsConfig()
	coreApp.Config().Set("agents.concurrency", config.Concurrency)
	coreApp.Config().Set("agents.rates", config.Rates)
	coreApp.Config().Set("agents.dispatch", config.Dispatch)
	coreApp.Config().Set("agents.config_path", core.JoinPath(CoreRoot(), "agents.yaml"))
	codexTotal := 0
	if limit, ok := config.Concurrency["codex"]; ok {
		codexTotal = limit.Total
	}
	coreApp.Config().Set("agents.codex_limit_debug", codexTotal)

	return core.Result{Value: service, OK: true}
}

// OnStartup registers Actions and starts the queue runner.
//
//	c.Action("runner.dispatch").Run(ctx, core.NewOptions(
//		core.Option{Key: "repo", Value: "go-io"},
//		core.Option{Key: "agent", Value: "codex"},
//	))
//	c.Action("runner.status").Run(ctx, core.NewOptions())
func (s *Service) OnStartup(ctx context.Context) core.Result {
	coreApp := s.Core()

	// Actions — the runner's capability map
	coreApp.Action("runner.dispatch", s.actionDispatch).Description = "Dispatch a subagent (checks frozen + concurrency)"
	coreApp.Action("runner.status", s.actionStatus).Description = "Query workspace status"
	coreApp.Action("runner.start", s.actionStart).Description = "Unfreeze dispatch queue"
	coreApp.Action("runner.stop", s.actionStop).Description = "Freeze dispatch queue (graceful)"
	coreApp.Action("runner.kill", s.actionKill).Description = "Kill all running agents (hard stop)"
	coreApp.Action("runner.poke", s.actionPoke).Description = "Drain next queued task"

	// Hydrate workspace registry from disk
	s.hydrateWorkspaces()

	// QUERY handler — workspace state queries
	coreApp.RegisterQuery(s.handleWorkspaceQuery)

	// Start the background queue runner
	s.startRunner()

	return core.Result{OK: true}
}

// OnShutdown freezes the queue.
//
//	result := service.OnShutdown(context.Background())
//	if result.OK {
//		core.Println(service.IsFrozen())
//	}
func (s *Service) OnShutdown(_ context.Context) core.Result {
	s.frozen = true
	return core.Result{OK: true}
}

// HandleIPCEvents applies runner side-effects for IPC messages.
//
//	service.HandleIPCEvents(c, messages.PokeQueue{})
//	service.HandleIPCEvents(c, messages.AgentCompleted{
//		Agent: "codex", Repo: "go-io", Workspace: "core/go-io/task-5", Status: "completed",
//	})
func (s *Service) HandleIPCEvents(coreApp *core.Core, msg core.Message) core.Result {
	switch ev := msg.(type) {
	case messages.AgentStarted:
		baseAgentName := baseAgent(ev.Agent)
		runningCount := s.countRunningByAgent(baseAgentName)
		var limit int
		concurrencyResult := coreApp.Config().Get("agents.concurrency")
		if concurrencyResult.OK {
			if concurrency, ok := concurrencyResult.Value.(map[string]ConcurrencyLimit); ok {
				if concurrencyLimit, has := concurrency[baseAgentName]; has {
					limit = concurrencyLimit.Total
				}
			}
		}
		notification := &AgentNotification{
			Status:    "started",
			Repo:      ev.Repo,
			Agent:     ev.Agent,
			Workspace: ev.Workspace,
			Running:   runningCount,
			Limit:     limit,
		}
		if notifier, ok := core.ServiceFor[channelSender](coreApp, "mcp"); ok {
			notifier.ChannelSend(context.Background(), "agent.status", notification)
		}

	case messages.AgentCompleted:
		// Update workspace status in Registry so concurrency count drops
		if ev.Workspace != "" {
			if workspaceResult := s.workspaces.Get(ev.Workspace); workspaceResult.OK {
				if workspaceStatus, ok := workspaceResult.Value.(*WorkspaceStatus); ok && workspaceStatus.Status == "running" {
					workspaceStatus.Status = ev.Status
					workspaceStatus.PID = 0
				}
			}
		} else {
			s.workspaces.Each(func(_ string, workspaceStatus *WorkspaceStatus) {
				if workspaceStatus.Repo == ev.Repo && workspaceStatus.Status == "running" {
					workspaceStatus.Status = ev.Status
					workspaceStatus.PID = 0
				}
			})
		}
		completedBaseAgentName := baseAgent(ev.Agent)
		runningCount := s.countRunningByAgent(completedBaseAgentName)
		var limit int
		completionResult := coreApp.Config().Get("agents.concurrency")
		if completionResult.OK {
			if concurrency, ok := completionResult.Value.(map[string]ConcurrencyLimit); ok {
				if concurrencyLimit, has := concurrency[completedBaseAgentName]; has {
					limit = concurrencyLimit.Total
				}
			}
		}
		notification := &AgentNotification{
			Status:    ev.Status,
			Repo:      ev.Repo,
			Agent:     ev.Agent,
			Workspace: ev.Workspace,
			Running:   runningCount,
			Limit:     limit,
		}
		if notifier, ok := core.ServiceFor[channelSender](coreApp, "mcp"); ok {
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
// Accepts the runner projection directly and the agentic projection from IPC.
//
//	s.TrackWorkspace("core/go-io/task-5", &WorkspaceStatus{Status: "running", Agent: "codex"})
//	s.TrackWorkspace("core/go-io/task-5", &agentic.WorkspaceStatus{Status: "running", Agent: "codex"})
func (s *Service) TrackWorkspace(name string, status any) {
	if s.workspaces == nil {
		return
	}
	var workspaceStatus *WorkspaceStatus
	switch value := status.(type) {
	case *WorkspaceStatus:
		workspaceStatus = value
	case *agentic.WorkspaceStatus:
		workspaceStatus = runnerWorkspaceStatusFromAgentic(value)
	default:
		statusJSON := core.JSONMarshalString(status)
		var decodedWorkspace WorkspaceStatus
		if result := core.JSONUnmarshalString(statusJSON, &decodedWorkspace); result.OK {
			workspaceStatus = &decodedWorkspace
		}
	}
	if workspaceStatus == nil {
		return
	}
	s.workspaces.Set(name, workspaceStatus)
	// Remove pending reservation now that the real workspace is tracked
	s.workspaces.Delete(core.Concat("pending/", workspaceStatus.Repo))
}

// Workspaces returns the workspace Registry.
//
//	s.Workspaces().Each(func(name string, workspaceStatus *WorkspaceStatus) { ... })
func (s *Service) Workspaces() *core.Registry[*WorkspaceStatus] {
	return s.workspaces
}

// handleWorkspaceQuery answers workspace state queries from Core QUERY calls.
//
//	result := c.QUERY(runner.WorkspaceQuery{Name: "core/go-io/task-42"})
//	result := c.QUERY(runner.WorkspaceQuery{Status: "running"})
func (s *Service) handleWorkspaceQuery(_ *core.Core, query core.Query) core.Result {
	workspaceQuery, ok := query.(WorkspaceQuery)
	if !ok {
		return core.Result{}
	}
	if workspaceQuery.Name != "" {
		return s.workspaces.Get(workspaceQuery.Name)
	}
	if workspaceQuery.Status != "" {
		var names []string
		s.workspaces.Each(func(name string, workspaceStatus *WorkspaceStatus) {
			if workspaceStatus.Status == workspaceQuery.Status {
				names = append(names, name)
			}
		})
		return core.Result{Value: names, OK: true}
	}
	return core.Result{Value: s.workspaces, OK: true}
}

// --- Actions ---

func (s *Service) actionDispatch(_ context.Context, options core.Options) core.Result {
	if s.frozen {
		return core.Result{Value: core.E("runner.actionDispatch", "queue is frozen", nil), OK: false}
	}

	agent := options.String("agent")
	if agent == "" {
		agent = "codex"
	}
	repo := options.String("repo")

	s.dispatchMu.Lock()
	defer s.dispatchMu.Unlock()

	can, reason := s.canDispatchAgent(agent)
	if !can {
		return core.Result{Value: core.E("runner.actionDispatch", core.Concat("queue at capacity: ", reason), nil), OK: false}
	}

	// Reserve the slot immediately — before returning to agentic.
	workspaceName := core.Concat("pending/", repo)
	s.workspaces.Set(workspaceName, &WorkspaceStatus{
		Status: "running",
		Agent:  agent,
		Repo:   repo,
		PID:    -1,
	})

	return core.Result{OK: true}
}

func (s *Service) actionStatus(_ context.Context, _ core.Options) core.Result {
	running, queued, completed, failed := 0, 0, 0, 0
	s.workspaces.Each(func(_ string, workspaceStatus *WorkspaceStatus) {
		switch workspaceStatus.Status {
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
	var runtime *core.Core
	if s.ServiceRuntime != nil {
		runtime = s.Core()
	}
	killed := 0
	s.workspaces.Each(func(_ string, workspaceStatus *WorkspaceStatus) {
		if workspaceStatus.Status == "running" && workspaceStatus.PID > 0 {
			if agentic.ProcessTerminate(runtime, "", workspaceStatus.PID) {
				killed++
			}
			workspaceStatus.Status = "failed"
			workspaceStatus.PID = 0
		}
		if workspaceStatus.Status == "queued" {
			workspaceStatus.Status = "failed"
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
		workspaceDir := core.PathDir(path)
		statusResult := ReadStatusResult(workspaceDir)
		if !statusResult.OK {
			continue
		}
		workspaceStatus, ok := statusResult.Value.(*WorkspaceStatus)
		if !ok || workspaceStatus == nil {
			continue
		}
		// Re-queue running agents on restart — process is dead, re-dispatch
		if workspaceStatus.Status == "running" {
			workspaceStatus.Status = "queued"
		}
		s.workspaces.Set(agentic.WorkspaceName(workspaceDir), workspaceStatus)
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
//	result := c.QUERY(runner.WorkspaceQuery{Status: "running"})
type WorkspaceQuery struct {
	Name   string
	Status string
}

// WorkspaceStatus tracks the state of an agent workspace.
//
//	workspaceStatus := &runner.WorkspaceStatus{Status: "running", Agent: "codex", Repo: "go-io", PID: 12345}
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
