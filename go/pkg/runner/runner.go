// SPDX-License-Identifier: EUPL-1.2

// service := runner.New()
// service.TrackWorkspace("core/go-io/task-5", &runner.WorkspaceStatus{Status: "running", Agent: "codex"})
package runner

import (
	"context"
	"time"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/agentic"
	"dappco.re/go/agent/pkg/messages"
)

// options := runner.Options{}
type Options struct{}

// service := runner.New()
// service.TrackWorkspace("core/go-io/task-5", &runner.WorkspaceStatus{Status: "running", Agent: "codex"})
type Service struct {
	*core.ServiceRuntime[Options]
	pokeCh       chan struct{}
	dispatchLock chan struct{}
	drainLock    chan struct{}
	frozen       bool
	backoff      map[string]time.Time
	failCount    map[string]int
	workspaces   *core.Registry[*WorkspaceStatus]
}

// lock acquires a named mutex — uses c.Lock(name) when Core is
// available, falls back to a channel-based lock for standalone use.
//
//	unlock := s.lock("runner.dispatch", s.dispatchLock)
//	defer unlock()
func (s *Service) lock(name string, fallback chan struct{}) (unlock func()) {
	if s.ServiceRuntime != nil {
		mu := s.Core().Lock(name).Mutex
		mu.Lock()
		return mu.Unlock
	}
	fallback <- struct{}{}
	return func() { <-fallback }
}

type channelSender interface {
	ChannelSend(ctx context.Context, channel string, data any)
}

// service := runner.New()
// service.TrackWorkspace("core/go-io/task-5", &runner.WorkspaceStatus{Status: "running", Agent: "codex"})
func New() *Service {
	return &Service{
		dispatchLock: make(chan struct{}, 1),
		drainLock:    make(chan struct{}, 1),
		backoff:      make(map[string]time.Time),
		failCount:    make(map[string]int),
		workspaces:   core.NewRegistry[*WorkspaceStatus](),
	}
}

// c := core.New(core.WithService(runner.Register))
// service := c.Service("runner")
func Register(coreApp *core.Core) core.Result {
	service := New()
	service.ServiceRuntime = core.NewServiceRuntime(coreApp, Options{})

	config := service.loadAgentsConfig()
	coreApp.Config().Set("agents.concurrency", config.Concurrency)
	coreApp.Config().Set("agents.rates", config.Rates)
	coreApp.Config().Set("agents.dispatch", config.Dispatch)
	coreApp.Config().Set("agents.config_path", AgentsConfigPath())
	codexTotal := 0
	if limit, ok := config.Concurrency["codex"]; ok {
		codexTotal = limit.Total
	}
	coreApp.Config().Set("agents.codex_limit_debug", codexTotal)

	return core.Result{Value: service, OK: true}
}

// c.Action("runner.dispatch").Run(ctx, core.NewOptions(
//
// core.Option{Key: "repo", Value: "go-io"},
// core.Option{Key: "agent", Value: "codex"},
//
// ))
// c.Action("runner.status").Run(ctx, core.NewOptions())
func (s *Service) OnStartup(ctx context.Context) core.Result {
	coreApp := s.Core()

	coreApp.Action("runner.dispatch", s.actionDispatch).Description = "Dispatch a subagent (checks frozen + concurrency)"
	coreApp.Action("runner.status", s.actionStatus).Description = "Query workspace status"
	coreApp.Action("runner.start", s.actionStart).Description = "Unfreeze dispatch queue"
	coreApp.Action("runner.stop", s.actionStop).Description = "Freeze dispatch queue (graceful)"
	coreApp.Action("runner.kill", s.actionKill).Description = "Kill all running agents (hard stop)"
	coreApp.Action("runner.poke", s.actionPoke).Description = "Drain next queued task"

	s.hydrateWorkspaces()

	coreApp.RegisterQuery(s.handleWorkspaceQuery)

	s.startRunner()

	return core.Result{OK: true}
}

// result := service.OnShutdown(context.Background())
//
//	if result.OK {
//		core.Println(service.IsFrozen())
//	}
func (s *Service) OnShutdown(_ context.Context) core.Result {
	s.frozen = true
	return core.Result{OK: true}
}

// service.HandleIPCEvents(c, messages.PokeQueue{})
//
//	service.HandleIPCEvents(c, messages.AgentCompleted{
//		Agent: "codex", Repo: "go-io", Workspace: "core/go-io/task-5", Status: "completed",
//	})
func (s *Service) HandleIPCEvents(coreApp *core.Core, msg core.Message) core.Result {
	sendNotification := func(channel string, data any) {
		serviceResult := coreApp.Service("mcp")
		if !serviceResult.OK {
			return
		}
		notifier, ok := serviceResult.Value.(channelSender)
		if !ok {
			return
		}
		notifier.ChannelSend(context.Background(), channel, data)
	}

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
		sendNotification("agent.status", notification)

	case messages.AgentCompleted:
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
		sendNotification("agent.status", notification)
		s.Poke()

	case messages.PokeQueue:
		s.drainQueueAndNotify(coreApp)

	case messages.QueueDrained:
		// H1: the queue lifecycle becomes observable — previously this event
		// was broadcast to no handler. Notify only; do not re-drain (no loop).
		sendNotification("queue.status", &QueueNotification{Completed: ev.Completed})

	case messages.HarvestRejected:
		// H4: a rejected harvest is surfaced, not silently dropped.
		sendNotification("harvest.status", &HarvestNotification{
			Status: "rejected",
			Repo:   ev.Repo,
			Branch: ev.Branch,
			Reason: ev.Reason,
		})

	case messages.InboxMessage:
		// H5: OpenBrain inbox arrivals surface on their own channel.
		sendNotification("inbox.status", &InboxNotification{
			New:   ev.New,
			Total: ev.Total,
		})

	case messages.RateLimitDetected:
		// H2: honour the rate limit into the runner's per-pool backoff so
		// drainOne pauses that pool until it lapses (the backoff map was read
		// at queue.go but never written). Written under the same "runner.drain"
		// lock drainQueue holds while reading s.backoff, so no map-race.
		if ev.Pool != "" {
			if duration, err := time.ParseDuration(ev.Duration); err == nil && duration > 0 {
				unlock := s.lock("runner.drain", s.drainLock)
				s.backoff[ev.Pool] = time.Now().Add(duration)
				unlock()
			}
		}
		sendNotification("ratelimit.status", &RateLimitNotification{Pool: ev.Pool, Duration: ev.Duration})

	case messages.HarvestComplete:
		// H3: surface the completed harvest here; the agentic handler reacting
		// to the same broadcast re-dispatches it into agentic.auto-pr.
		sendNotification("harvest.status", &HarvestNotification{
			Status: "complete",
			Repo:   ev.Repo,
			Branch: ev.Branch,
			Files:  ev.Files,
		})
	}
	return core.Result{OK: true}
}

// if s.IsFrozen() { return "queue is frozen" }
func (s *Service) IsFrozen() bool {
	return s.frozen
}

// s.Poke()
func (s *Service) Poke() {
	if s.pokeCh == nil {
		return
	}
	select {
	case s.pokeCh <- struct{}{}:
	default:
	}
}

// s.TrackWorkspace("core/go-io/task-5", &WorkspaceStatus{Status: "running", Agent: "codex"})
// s.TrackWorkspace("core/go-io/task-5", &agentic.WorkspaceStatus{Status: "running", Agent: "codex"})
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
	if r := s.workspaces.Set(name, workspaceStatus); !r.OK {
		core.Warn("runner: failed to track workspace", "workspace", name, "reason", r.Value)
	}
	_ = s.workspaces.Delete(core.Concat("pending/", workspaceStatus.Repo)) // best-effort: the pending marker may already be gone
}

// s.Workspaces().Each(func(name string, workspaceStatus *WorkspaceStatus) { core.Println(name, workspaceStatus.Status) })
func (s *Service) Workspaces() *core.Registry[*WorkspaceStatus] {
	return s.workspaces
}

// result := c.QUERY(runner.WorkspaceQuery{Name: "core/go-io/task-42"})
// result := c.QUERY(runner.WorkspaceQuery{Status: "running"})
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

func (s *Service) actionDispatch(_ context.Context, options core.Options) core.Result {
	if s.frozen {
		return core.Result{Value: core.E("runner.actionDispatch", "queue is frozen", nil), OK: false}
	}

	agent := options.String("agent")
	if agent == "" {
		agent = "codex"
	}
	repo := options.String("repo")

	unlock := s.lock("runner.dispatch", s.dispatchLock)
	defer unlock()

	can, reason := s.canDispatchAgent(agent)
	if !can {
		return core.Result{Value: core.E("runner.actionDispatch", core.Concat("queue at capacity: ", reason), nil), OK: false}
	}

	workspaceName := core.Concat("pending/", repo)
	// Reported: Set only fails on a locked or sealed registry, so a
	// failure here means this workspace silently stops being tracked.
	if r := s.workspaces.Set(workspaceName, &WorkspaceStatus{
		Status: "running",
		Agent:  agent,
		Repo:   repo,
		PID:    -1,
	}); !r.OK {
		core.Warn("runner: failed to track workspace", "reason", r.Value)
	}

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
	cleared := 0
	seenQueued := make(map[string]bool)

	for _, statusPath := range agentic.WorkspaceStatusPaths() {
		workspaceDir := core.PathDir(statusPath)
		statusResult := ReadStatusResult(workspaceDir)
		workspaceStatus, ok := statusResult.Value.(*WorkspaceStatus)
		if !ok || workspaceStatus == nil {
			continue
		}

		switch workspaceStatus.Status {
		case "running":
			if workspaceStatus.PID > 0 && agentic.ProcessTerminate(runtime, "", workspaceStatus.PID) {
				killed++
			}
			workspaceStatus.Status = "failed"
			workspaceStatus.PID = 0
			if writeResult := WriteStatus(workspaceDir, workspaceStatus); !writeResult.OK {
				core.Warn("runner.actionKill: failed to write failed status", "workspace", agentic.WorkspaceName(workspaceDir), "reason", writeResult.Value)
			}
			if s.workspaces != nil {
				// Reported: Set only fails on a locked or sealed registry, so a
				// failure here means this workspace silently stops being tracked.
				if r := s.workspaces.Set(agentic.WorkspaceName(workspaceDir), workspaceStatus); !r.OK {
					core.Warn("runner: failed to track workspace", "reason", r.Value)
				}
			}
		case "queued":
			workspaceName := agentic.WorkspaceName(workspaceDir)
			if seenQueued[workspaceName] {
				continue
			}
			seenQueued[workspaceName] = true
			if deleteResult := fs.DeleteAll(workspaceDir); !deleteResult.OK {
				core.Warn("runner.actionKill: failed to delete queued workspace", "workspace", workspaceName, "reason", core.Sprint(deleteResult.Value))
				continue
			}
			cleared++
			if s.workspaces != nil {
				_ = s.workspaces.Delete(workspaceName) // best-effort: the key may already be gone
			}
		}
	}

	return core.Result{Value: core.Sprintf("killed %d agents, cleared %d queued", killed, cleared), OK: true}
}

func (s *Service) actionPoke(_ context.Context, _ core.Options) core.Result {
	var coreApp *core.Core
	if s.ServiceRuntime != nil {
		coreApp = s.Core()
	}
	s.drainQueueAndNotify(coreApp)
	return core.Result{OK: true}
}

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
			s.drainQueueAndNotify(s.Core())
		case <-s.pokeCh:
			s.drainQueueAndNotify(s.Core())
		}
	}
}

func (s *Service) drainQueueAndNotify(coreApp *core.Core) {
	completed := s.drainQueue()
	if coreApp != nil {
		// Best-effort: a listener that has gone away must not fail this.
		_ = coreApp.ACTION(messages.QueueDrained{Completed: completed})
	}
}

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
		if workspaceStatus.Status == "running" {
			workspaceStatus.Status = "queued"
		}
		// Reported: Set only fails on a locked or sealed registry, so a
		// failure here means this workspace silently stops being tracked.
		if r := s.workspaces.Set(agentic.WorkspaceName(workspaceDir), workspaceStatus); !r.OK {
			core.Warn("runner: failed to track workspace", "reason", r.Value)
		}
	}
}

// notification := runner.AgentNotification{Status: "started", Repo: "go-io", Agent: "codex", Workspace: "core/go-io/task-5", Running: 1, Limit: 2}
type AgentNotification struct {
	Status    string `json:"status"`
	Repo      string `json:"repo"`
	Agent     string `json:"agent"`
	Workspace string `json:"workspace"`
	Running   int    `json:"running"`
	Limit     int    `json:"limit"`
}

// notification := runner.QueueNotification{Completed: 3}
type QueueNotification struct {
	Completed int `json:"completed"`
}

// notification := runner.HarvestNotification{Status: "rejected", Repo: "go-io", Branch: "agent/fix", Reason: "binary detected"}
type HarvestNotification struct {
	Status string `json:"status"`
	Repo   string `json:"repo"`
	Branch string `json:"branch"`
	Files  int    `json:"files,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// notification := runner.InboxNotification{New: 2, Total: 5}
type InboxNotification struct {
	New   int `json:"new"`
	Total int `json:"total"`
}

// notification := runner.RateLimitNotification{Pool: "codex", Duration: "30m0s"}
type RateLimitNotification struct {
	Pool     string `json:"pool"`
	Duration string `json:"duration"`
}

// result := c.QUERY(runner.WorkspaceQuery{Status: "running"})
type WorkspaceQuery struct {
	Name   string
	Status string
}

// workspaceStatus := &runner.WorkspaceStatus{Status: "running", Agent: "codex", Repo: "go-io", PID: 12345}
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
	// Runtime mirrors agentic.WorkspaceStatus.Runtime — "vz" for in-process
	// Virtualization.framework dispatches. Kept in sync so a VZ status survives a
	// round-trip through this struct without dropping the tag.
	Runtime string `json:"runtime,omitempty"`
}
