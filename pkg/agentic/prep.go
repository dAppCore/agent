// SPDX-License-Identifier: EUPL-1.2

// core.New(core.WithService(agentic.Register))
package agentic

import (
	"context"
	"encoding/base64"
	"sync"
	"time"

	"dappco.re/go/agent/pkg/lib"
	core "dappco.re/go/core"
	"dappco.re/go/core/forge"
	coremcp "forge.lthn.ai/core/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// options := agentic.AgentOptions{}
type AgentOptions struct{}

// core.New(core.WithService(agentic.Register))
type PrepSubsystem struct {
	*core.ServiceRuntime[AgentOptions]
	forge          *forge.Forge
	forgeURL       string
	forgeToken     string
	brainURL       string
	brainKey       string
	codePath       string
	startupContext context.Context
	dispatchMu     sync.Mutex
	drainMu        sync.Mutex
	pokeCh         chan struct{}
	frozen         bool
	backoff        map[string]time.Time
	failCount      map[string]int
	workspaces     *core.Registry[*WorkspaceStatus]
}

var _ coremcp.Subsystem = (*PrepSubsystem)(nil)

// subsystem := agentic.NewPrep()
// subsystem.SetCompletionNotifier(monitor)
func NewPrep() *PrepSubsystem {
	home := HomeDir()

	forgeToken := core.Env("FORGE_TOKEN")
	if forgeToken == "" {
		forgeToken = core.Env("GITEA_TOKEN")
	}

	brainKey := core.Env("CORE_BRAIN_KEY")
	if brainKey == "" {
		if r := fs.Read(core.JoinPath(home, ".claude", "brain.key")); r.OK {
			brainKey = core.Trim(r.Value.(string))
		}
	}

	forgeURL := envOr("FORGE_URL", "https://forge.lthn.ai")

	return &PrepSubsystem{
		forge:      forge.NewForge(forgeURL, forgeToken),
		forgeURL:   forgeURL,
		forgeToken: forgeToken,
		brainURL:   envOr("CORE_BRAIN_URL", "https://api.lthn.sh"),
		brainKey:   brainKey,
		codePath:   envOr("CODE_PATH", core.JoinPath(home, "Code")),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
		workspaces: core.NewRegistry[*WorkspaceStatus](),
	}
}

// c.Action("agentic.dispatch").Run(ctx, options)
// c.Actions() // ["agentic.dispatch", "agentic.prep", "agentic.status", "agentic.verify"]
func (s *PrepSubsystem) OnStartup(ctx context.Context) core.Result {
	c := s.Core()

	c.SetEntitlementChecker(func(action string, qty int, _ context.Context) core.Entitlement {
		if !core.HasPrefix(action, "agentic.") {
			return core.Entitlement{Allowed: true, Unlimited: true}
		}
		if core.HasPrefix(action, "agentic.monitor.") || core.HasPrefix(action, "agentic.complete") {
			return core.Entitlement{Allowed: true, Unlimited: true}
		}
		switch action {
		case "agentic.status", "agentic.scan", "agentic.watch",
			"agentic.issue.get", "agentic.issue.list", "agentic.pr.get", "agentic.pr.list",
			"agentic.prompt", "agentic.task", "agentic.flow", "agentic.persona",
			"agentic.sync.status", "agentic.fleet.nodes", "agentic.fleet.stats", "agentic.fleet.events",
			"agentic.credits.balance", "agentic.credits.history",
			"agentic.subscription.detect", "agentic.subscription.budget":
			return core.Entitlement{Allowed: true, Unlimited: true}
		}
		if s.frozen {
			return core.Entitlement{Allowed: false, Reason: "agent queue is frozen — shutting down"}
		}
		return core.Entitlement{Allowed: true}
	})

	lib.MountData(c)

	RegisterHTTPTransport(c)
	c.Drive().New(core.NewOptions(
		core.Option{Key: "name", Value: "forge"},
		core.Option{Key: "transport", Value: s.forgeURL},
		core.Option{Key: "token", Value: s.forgeToken},
	))
	c.Drive().New(core.NewOptions(
		core.Option{Key: "name", Value: "brain"},
		core.Option{Key: "transport", Value: s.brainURL},
		core.Option{Key: "token", Value: s.brainKey},
	))

	c.Action("agentic.sync.push", s.handleSyncPush).Description = "Push completed dispatch state to the platform API"
	c.Action("agent.sync.push", s.handleSyncPush).Description = "Push completed dispatch state to the platform API"
	c.Action("agentic.sync.pull", s.handleSyncPull).Description = "Pull fleet context from the platform API"
	c.Action("agent.sync.pull", s.handleSyncPull).Description = "Pull fleet context from the platform API"
	c.Action("agentic.sync.status", s.handleSyncStatus).Description = "Get fleet sync status from the platform API"
	c.Action("agent.sync.status", s.handleSyncStatus).Description = "Get fleet sync status from the platform API"
	c.Action("agentic.auth.provision", s.handleAuthProvision).Description = "Provision a platform API key for an authenticated agent user"
	c.Action("agent.auth.provision", s.handleAuthProvision).Description = "Provision a platform API key for an authenticated agent user"
	c.Action("agentic.auth.revoke", s.handleAuthRevoke).Description = "Revoke a platform API key"
	c.Action("agent.auth.revoke", s.handleAuthRevoke).Description = "Revoke a platform API key"
	c.Action("agentic.fleet.register", s.handleFleetRegister).Description = "Register a fleet node with the platform API"
	c.Action("agent.fleet.register", s.handleFleetRegister).Description = "Register a fleet node with the platform API"
	c.Action("agentic.fleet.heartbeat", s.handleFleetHeartbeat).Description = "Send a heartbeat for a fleet node"
	c.Action("agent.fleet.heartbeat", s.handleFleetHeartbeat).Description = "Send a heartbeat for a fleet node"
	c.Action("agentic.fleet.deregister", s.handleFleetDeregister).Description = "Deregister a fleet node from the platform API"
	c.Action("agent.fleet.deregister", s.handleFleetDeregister).Description = "Deregister a fleet node from the platform API"
	c.Action("agentic.fleet.nodes", s.handleFleetNodes).Description = "List registered fleet nodes"
	c.Action("agent.fleet.nodes", s.handleFleetNodes).Description = "List registered fleet nodes"
	c.Action("agentic.fleet.task.assign", s.handleFleetAssignTask).Description = "Assign a fleet task to an agent"
	c.Action("agent.fleet.task.assign", s.handleFleetAssignTask).Description = "Assign a fleet task to an agent"
	c.Action("agentic.fleet.task.complete", s.handleFleetCompleteTask).Description = "Complete a fleet task and report results"
	c.Action("agent.fleet.task.complete", s.handleFleetCompleteTask).Description = "Complete a fleet task and report results"
	c.Action("agentic.fleet.task.next", s.handleFleetNextTask).Description = "Ask the platform for the next fleet task"
	c.Action("agent.fleet.task.next", s.handleFleetNextTask).Description = "Ask the platform for the next fleet task"
	c.Action("agentic.fleet.stats", s.handleFleetStats).Description = "Get fleet activity statistics"
	c.Action("agent.fleet.stats", s.handleFleetStats).Description = "Get fleet activity statistics"
	c.Action("agentic.fleet.events", s.handleFleetEvents).Description = "Read fleet task assignment events from the platform API"
	c.Action("agent.fleet.events", s.handleFleetEvents).Description = "Read fleet task assignment events from the platform API"
	c.Action("agentic.credits.award", s.handleCreditsAward).Description = "Award credits to a fleet node"
	c.Action("agent.credits.award", s.handleCreditsAward).Description = "Award credits to a fleet node"
	c.Action("agentic.credits.balance", s.handleCreditsBalance).Description = "Get credit balance for a fleet node"
	c.Action("agent.credits.balance", s.handleCreditsBalance).Description = "Get credit balance for a fleet node"
	c.Action("agentic.credits.history", s.handleCreditsHistory).Description = "List credit entries for a fleet node"
	c.Action("agent.credits.history", s.handleCreditsHistory).Description = "List credit entries for a fleet node"
	c.Action("agentic.subscription.detect", s.handleSubscriptionDetect).Description = "Detect available provider capabilities"
	c.Action("agent.subscription.detect", s.handleSubscriptionDetect).Description = "Detect available provider capabilities"
	c.Action("agentic.subscription.budget", s.handleSubscriptionBudget).Description = "Get the compute budget for a fleet node"
	c.Action("agent.subscription.budget", s.handleSubscriptionBudget).Description = "Get the compute budget for a fleet node"
	c.Action("agentic.subscription.budget.update", s.handleSubscriptionBudgetUpdate).Description = "Update the compute budget for a fleet node"
	c.Action("agent.subscription.budget.update", s.handleSubscriptionBudgetUpdate).Description = "Update the compute budget for a fleet node"

	c.Action("agentic.dispatch", s.handleDispatch).Description = "Prep workspace and spawn a subagent"
	c.Action("agentic.prep", s.handlePrep).Description = "Clone repo and build agent prompt"
	c.Action("agentic.status", s.handleStatus).Description = "List workspace states (running/completed/blocked)"
	c.Action("agentic.resume", s.handleResume).Description = "Resume a blocked or completed workspace"
	c.Action("agentic.scan", s.handleScan).Description = "Scan Forge repos for actionable issues"
	c.Action("agentic.watch", s.handleWatch).Description = "Watch workspace for changes and report"

	c.Action("agentic.qa", s.handleQA).Description = "Run build + test QA checks on workspace"
	c.Action("agentic.auto-pr", s.handleAutoPR).Description = "Create PR from completed workspace"
	c.Action("agentic.verify", s.handleVerify).Description = "Verify PR and auto-merge if clean"
	c.Action("agentic.ingest", s.handleIngest).Description = "Create issues from agent findings"
	c.Action("agentic.poke", s.handlePoke).Description = "Drain next queued task from the queue"
	c.Action("agentic.mirror", s.handleMirror).Description = "Mirror agent branches to GitHub"

	c.Action("agentic.issue.get", s.handleIssueGet).Description = "Get a Forge issue by number"
	c.Action("agentic.issue.list", s.handleIssueList).Description = "List Forge issues for a repo"
	c.Action("agentic.issue.create", s.handleIssueCreate).Description = "Create a Forge issue"
	c.Action("agentic.pr.get", s.handlePRGet).Description = "Get a Forge PR by number"
	c.Action("agentic.pr.list", s.handlePRList).Description = "List Forge PRs for a repo"
	c.Action("agentic.pr.merge", s.handlePRMerge).Description = "Merge a Forge PR"
	c.Action("agentic.pr.close", s.handlePRClose).Description = "Close a Forge PR"

	c.Action("agentic.review-queue", s.handleReviewQueue).Description = "Run CodeRabbit review on completed workspaces"

	c.Action("agentic.epic", s.handleEpic).Description = "Create sub-issues from an epic plan"
	c.Action("plan.create", s.handlePlanCreate).Description = "Create a structured implementation plan"
	c.Action("plan.get", s.handlePlanGet).Description = "Read an implementation plan by ID or slug"
	c.Action("plan.read", s.handlePlanRead).Description = "Read an implementation plan by ID"
	c.Action("plan.update", s.handlePlanUpdate).Description = "Update plan status, phases, notes, or agent assignment"
	c.Action("plan.update_status", s.handlePlanUpdateStatus).Description = "Update an implementation plan lifecycle status by slug"
	c.Action("plan.archive", s.handlePlanArchive).Description = "Archive an implementation plan by slug"
	c.Action("plan.delete", s.handlePlanDelete).Description = "Delete an implementation plan by ID"
	c.Action("plan.list", s.handlePlanList).Description = "List implementation plans with optional filters"
	c.Action("phase.get", s.handlePhaseGet).Description = "Read a plan phase by slug and order"
	c.Action("phase.update_status", s.handlePhaseUpdateStatus).Description = "Update plan phase status by slug and order"
	c.Action("phase.add_checkpoint", s.handlePhaseAddCheckpoint).Description = "Append a checkpoint note to a plan phase"
	c.Action("task.update", s.handleTaskUpdate).Description = "Update a plan task by slug, phase, and identifier"
	c.Action("task.toggle", s.handleTaskToggle).Description = "Toggle a plan task between pending and completed"
	c.Action("session.start", s.handleSessionStart).Description = "Start an agent session for a plan"
	c.Action("session.get", s.handleSessionGet).Description = "Read a session by session ID"
	c.Action("session.list", s.handleSessionList).Description = "List sessions with optional plan or status filters"
	c.Action("session.continue", s.handleSessionContinue).Description = "Continue a session from its latest saved context"
	c.Action("session.end", s.handleSessionEnd).Description = "End a session with status and summary"
	c.Action("session.log", s.handleSessionLog).Description = "Append a typed work-log entry to a stored session"
	c.Action("session.artifact", s.handleSessionArtifact).Description = "Record a created, modified, deleted, or reviewed artifact for a session"
	c.Action("session.handoff", s.handleSessionHandoff).Description = "Pause a session with handoff notes for the next agent"
	c.Action("session.resume", s.handleSessionResume).Description = "Resume a paused or handed-off session from local cache"
	c.Action("session.replay", s.handleSessionReplay).Description = "Build replay context for a session from work logs and artifacts"
	c.Action("state.set", s.handleStateSet).Description = "Store shared plan state for later sessions"
	c.Action("state.get", s.handleStateGet).Description = "Read shared plan state by key"
	c.Action("state.list", s.handleStateList).Description = "List shared plan state for a plan"
	c.Action("template.list", s.handleTemplateList).Description = "List available YAML plan templates"
	c.Action("template.preview", s.handleTemplatePreview).Description = "Preview a YAML plan template with variable substitution"
	c.Action("template.create_plan", s.handleTemplateCreatePlan).Description = "Create a stored plan from a YAML template"
	c.Action("issue.create", s.handleIssueRecordCreate).Description = "Create a tracked platform issue"
	c.Action("issue.get", s.handleIssueRecordGet).Description = "Read a tracked platform issue by slug"
	c.Action("issue.list", s.handleIssueRecordList).Description = "List tracked platform issues with optional filters"
	c.Action("issue.update", s.handleIssueRecordUpdate).Description = "Update a tracked platform issue by slug"
	c.Action("issue.comment", s.handleIssueRecordComment).Description = "Add a comment to a tracked platform issue"
	c.Action("issue.archive", s.handleIssueRecordArchive).Description = "Archive a tracked platform issue by slug"
	c.Action("sprint.create", s.handleSprintCreate).Description = "Create a tracked platform sprint"
	c.Action("sprint.get", s.handleSprintGet).Description = "Read a tracked platform sprint by slug"
	c.Action("sprint.list", s.handleSprintList).Description = "List tracked platform sprints with optional filters"
	c.Action("sprint.update", s.handleSprintUpdate).Description = "Update a tracked platform sprint by slug"
	c.Action("sprint.archive", s.handleSprintArchive).Description = "Archive a tracked platform sprint by slug"
	c.Action("content.generate", s.handleContentGenerate).Description = "Generate content using the platform content pipeline"
	c.Action("content.batch", s.handleContentBatchGenerate).Description = "Start or continue batch content generation"
	c.Action("content.batch.generate", s.handleContentBatchGenerate).Description = "Start or continue batch content generation"
	c.Action("content.batch_generate", s.handleContentBatchGenerate).Description = "Start or continue batch content generation"
	c.Action("content_batch", s.handleContentBatchGenerate).Description = "Start or continue batch content generation"
	c.Action("content.brief.create", s.handleContentBriefCreate).Description = "Create a reusable content brief"
	c.Action("content.brief_create", s.handleContentBriefCreate).Description = "Create a reusable content brief"
	c.Action("content.brief.get", s.handleContentBriefGet).Description = "Read a content brief by ID or slug"
	c.Action("content.brief_get", s.handleContentBriefGet).Description = "Read a content brief by ID or slug"
	c.Action("content.brief.list", s.handleContentBriefList).Description = "List content briefs with optional filters"
	c.Action("content.brief_list", s.handleContentBriefList).Description = "List content briefs with optional filters"
	c.Action("content.status", s.handleContentStatus).Description = "Read batch content generation status"
	c.Action("content.usage.stats", s.handleContentUsageStats).Description = "Read content provider usage statistics"
	c.Action("content.usage_stats", s.handleContentUsageStats).Description = "Read content provider usage statistics"
	c.Action("content.from.plan", s.handleContentFromPlan).Description = "Generate content from plan context"
	c.Action("content.from_plan", s.handleContentFromPlan).Description = "Generate content from plan context"

	c.Action("agentic.prompt", s.handlePrompt).Description = "Read a system prompt by slug"
	c.Action("agentic.task", s.handleTask).Description = "Read a task plan by slug"
	c.Action("agentic.flow", s.handleFlow).Description = "Read a build/release flow by slug"
	c.Action("agentic.persona", s.handlePersona).Description = "Read a persona by path"

	c.Task("agent.completion", core.Task{
		Description: "QA → PR → Verify → Ingest → Poke",
		Steps: []core.Step{
			{Action: "agentic.qa"},
			{Action: "agentic.auto-pr"},
			{Action: "agentic.verify"},
			{Action: "agentic.ingest", Async: true},
			{Action: "agentic.poke", Async: true},
		},
	})

	c.Action("agentic.complete", s.handleComplete).Description = "Run completion pipeline (QA → PR → Verify → Ingest → Poke) in background"

	s.hydrateWorkspaces()

	c.RegisterQuery(s.handleWorkspaceQuery)

	s.StartRunner()
	s.registerCommands(ctx)
	s.registerWorkspaceCommands()
	s.registerForgeCommands()
	s.registerPlatformCommands()
	return core.Result{OK: true}
}

// s.registerCommands(ctx)

// subsystem := agentic.NewPrep()
// _ = subsystem.OnShutdown(context.Background())
func (s *PrepSubsystem) OnShutdown(ctx context.Context) core.Result {
	s.frozen = true
	return core.Result{OK: true}
}

// s.hydrateWorkspaces()
// s.workspaces.Names() // ["core/go-io/task-5", "ws-blocked", "ws-ready-for-review"]
func (s *PrepSubsystem) hydrateWorkspaces() {
	if s.workspaces == nil {
		s.workspaces = core.NewRegistry[*WorkspaceStatus]()
	}
	for _, path := range WorkspaceStatusPaths() {
		workspaceDir := core.PathDir(path)
		result := ReadStatusResult(workspaceDir)
		st, ok := workspaceStatusValue(result)
		if !ok {
			continue
		}
		s.workspaces.Set(WorkspaceName(workspaceDir), st)
	}
}

// s.TrackWorkspace("core/go-io/task-5", st)
func (s *PrepSubsystem) TrackWorkspace(name string, st *WorkspaceStatus) {
	if s.workspaces != nil {
		s.workspaces.Set(name, st)
	}
}

// s.Workspaces().Names()                        // all workspace names
// s.Workspaces().List("core/*")                 // org-scoped workspaces
// s.Workspaces().Each(func(name string, workspaceStatus *WorkspaceStatus) { core.Println(name, workspaceStatus.Status) })
func (s *PrepSubsystem) Workspaces() *core.Registry[*WorkspaceStatus] {
	return s.workspaces
}

func envOr(key, fallback string) string {
	if v := core.Env(key); v != "" {
		return v
	}
	return fallback
}

// subsystem := agentic.NewPrep()
// name := subsystem.Name()
// _ = name // "agentic"
func (s *PrepSubsystem) Name() string { return "agentic" }

// subsystem := agentic.NewPrep()
// subsystem.RegisterTools(server)
func (s *PrepSubsystem) RegisterTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_prep_workspace",
		Description: "Prepare an agent workspace: clone repo, create branch, build prompt with context.",
	}, s.prepWorkspace)

	s.registerDispatchTool(server)
	s.registerStatusTool(server)
	s.registerResumeTool(server)
	s.registerCreatePRTool(server)
	s.registerListPRsTool(server)
	s.registerEpicTool(server)
	s.registerMirrorTool(server)
	s.registerRemoteDispatchTool(server)
	s.registerRemoteStatusTool(server)
	s.registerReviewQueueTool(server)
	s.registerPlatformTools(server)
	s.registerShutdownTools(server)
	s.registerSessionTools(server)
	s.registerStateTools(server)
	s.registerPhaseTools(server)
	s.registerTaskTools(server)
	s.registerTemplateTools(server)
	s.registerIssueTools(server)
	s.registerSprintTools(server)
	s.registerContentTools(server)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_scan",
		Description: "Scan Forge repos for open issues with actionable labels (agentic, help-wanted, bug).",
	}, s.scan)

	s.registerPlanTools(server)
	s.registerWatchTool(server)
}

// subsystem := agentic.NewPrep()
// _ = subsystem.Shutdown(context.Background())
func (s *PrepSubsystem) Shutdown(_ context.Context) error { return nil }

// input := agentic.PrepInput{Repo: "go-io", Issue: 15, Task: "Migrate to Core primitives"}
type PrepInput struct {
	Repo         string            `json:"repo"`
	Org          string            `json:"org,omitempty"`
	Task         string            `json:"task,omitempty"`
	Agent        string            `json:"agent,omitempty"`
	Issue        int               `json:"issue,omitempty"`
	PR           int               `json:"pr,omitempty"`
	Branch       string            `json:"branch,omitempty"`
	Tag          string            `json:"tag,omitempty"`
	Template     string            `json:"template,omitempty"`
	PlanTemplate string            `json:"plan_template,omitempty"`
	Variables    map[string]string `json:"variables,omitempty"`
	Persona      string            `json:"persona,omitempty"`
	DryRun       bool              `json:"dry_run,omitempty"`
}

// out := agentic.PrepOutput{Success: true, WorkspaceDir: ".core/workspace/core/go-io/task-15"}
type PrepOutput struct {
	Success      bool   `json:"success"`
	WorkspaceDir string `json:"workspace_dir"`
	RepoDir      string `json:"repo_dir"`
	Branch       string `json:"branch"`
	Prompt       string `json:"prompt,omitempty"`
	Memories     int    `json:"memories"`
	Consumers    int    `json:"consumers"`
	Resumed      bool   `json:"resumed"`
}

// dir := workspaceDir("core", "go-io", PrepInput{Issue: 15})
// dir == ".core/workspace/core/go-io/task-15"
func workspaceDir(org, repo string, input PrepInput) (string, error) {
	r := workspaceDirResult(org, repo, input)
	if !r.OK {
		err, _ := r.Value.(error)
		if err == nil {
			err = core.E("workspaceDir", "failed to resolve workspace directory", nil)
		}
		return "", err
	}
	workspaceDir, ok := r.Value.(string)
	if !ok || workspaceDir == "" {
		return "", core.E("workspaceDir", "invalid workspace directory result", nil)
	}
	return workspaceDir, nil
}

// r := workspaceDirResult("core", "go-io", PrepInput{Issue: 15})
// if r.OK { workspaceDir := r.Value.(string) }
func workspaceDirResult(org, repo string, input PrepInput) core.Result {
	orgName := core.ValidateName(org)
	if !orgName.OK {
		err, _ := orgName.Value.(error)
		return core.Result{Value: core.E("workspaceDir", "invalid org name", err), OK: false}
	}

	repoName := core.ValidateName(repo)
	if !repoName.OK {
		err, _ := repoName.Value.(error)
		return core.Result{Value: core.E("workspaceDir", "invalid repo name", err), OK: false}
	}

	base := core.JoinPath(WorkspaceRoot(), orgName.Value.(string), repoName.Value.(string))
	switch {
	case input.PR > 0:
		return core.Result{Value: core.JoinPath(base, core.Sprintf("pr-%d", input.PR)), OK: true}
	case input.Issue > 0:
		return core.Result{Value: core.JoinPath(base, core.Sprintf("task-%d", input.Issue)), OK: true}
	case input.Branch != "":
		return core.Result{Value: core.JoinPath(base, input.Branch), OK: true}
	case input.Tag != "":
		return core.Result{Value: core.JoinPath(base, input.Tag), OK: true}
	default:
		return core.Result{Value: core.E("workspaceDir", "one of issue, pr, branch, or tag is required", nil), OK: false}
	}
}

func (s *PrepSubsystem) prepWorkspace(ctx context.Context, _ *mcp.CallToolRequest, input PrepInput) (*mcp.CallToolResult, PrepOutput, error) {
	if input.Repo == "" {
		return nil, PrepOutput{}, core.E("prepWorkspace", "repo is required", nil)
	}
	if input.Org == "" {
		input.Org = "core"
	}
	if input.Template == "" {
		input.Template = "coding"
	}

	workspaceResult := workspaceDirResult(input.Org, input.Repo, input)
	if !workspaceResult.OK {
		err, _ := workspaceResult.Value.(error)
		if err == nil {
			err = core.E("prepWorkspace", "workspace path not resolved", nil)
		}
		return nil, PrepOutput{}, err
	}
	workspaceDir, ok := workspaceResult.Value.(string)
	if !ok || workspaceDir == "" {
		return nil, PrepOutput{}, core.E("prepWorkspace", "invalid workspace path", nil)
	}

	repoDir := workspaceRepoDir(workspaceDir)
	metaDir := workspaceMetaDir(workspaceDir)
	out := PrepOutput{WorkspaceDir: workspaceDir, RepoDir: repoDir}

	repoPath := core.JoinPath(s.codePath, input.Org, input.Repo)
	process := s.Core().Process()

	if r := fs.EnsureDir(metaDir); !r.OK {
		return nil, PrepOutput{}, core.E("prep", "failed to create meta dir", nil)
	}

	resumed := fs.IsDir(core.JoinPath(repoDir, ".git"))
	out.Resumed = resumed

	if resumed {
		r := process.RunIn(ctx, repoDir, "git", "rev-parse", "--abbrev-ref", "HEAD")
		currentBranch := ""
		if r.OK {
			currentBranch = core.Trim(r.Value.(string))
		}
		defaultBranch := s.DefaultBranch(repoDir)
		if currentBranch == "" || currentBranch == "HEAD" {
			currentBranch = defaultBranch
		}
		if currentBranch != "" {
			process.RunIn(ctx, repoDir, "git", "checkout", currentBranch)
			if process.RunIn(ctx, repoDir, "git", "ls-remote", "--exit-code", "--heads", "origin", currentBranch).OK {
				process.RunIn(ctx, repoDir, "git", "pull", "--ff-only", "origin", currentBranch)
			} else if defaultBranch != "" {
				process.RunIn(ctx, repoDir, "git", "fetch", "origin", defaultBranch)
			}
		}
	}

	if result := lib.ExtractWorkspace("default", workspaceDir, &lib.WorkspaceData{
		Repo:   input.Repo,
		Branch: "",
		Task:   input.Task,
		Agent:  input.Agent,
	}); !result.OK {
		if err, ok := result.Value.(error); ok {
			return nil, PrepOutput{}, core.E("prepWorkspace", "extract default workspace template", err)
		}
		return nil, PrepOutput{}, core.E("prepWorkspace", "extract default workspace template", nil)
	}

	if !resumed {
		if r := process.RunIn(ctx, ".", "git", "clone", repoPath, repoDir); !r.OK {
			return nil, PrepOutput{}, core.E("prep", core.Concat("git clone failed for ", input.Repo), nil)
		}

		taskSlug := sanitiseBranchSlug(input.Task, 40)
		if taskSlug == "" {
			if input.Issue > 0 {
				taskSlug = core.Sprintf("issue-%d", input.Issue)
			} else if input.PR > 0 {
				taskSlug = core.Sprintf("pr-%d", input.PR)
			} else {
				taskSlug = core.Sprintf("work-%d", time.Now().Unix())
			}
		}
		branchName := core.Sprintf("agent/%s", taskSlug)

		if r := process.RunIn(ctx, repoDir, "git", "checkout", "-b", branchName); !r.OK {
			return nil, PrepOutput{}, core.E("prep.branch", core.Sprintf("failed to create branch %q", branchName), nil)
		}
		out.Branch = branchName
	} else {
		r := process.RunIn(ctx, repoDir, "git", "rev-parse", "--abbrev-ref", "HEAD")
		if r.OK {
			out.Branch = core.Trim(r.Value.(string))
		}
	}

	lang := detectLanguage(repoPath)
	if lang == "php" {
		if r := lib.WorkspaceFile("default", "CODEX-PHP.md.tmpl"); r.OK {
			codexPath := core.JoinPath(workspaceDir, "CODEX.md")
			fs.Write(codexPath, r.Value.(string))
		}
	}

	s.cloneWorkspaceDeps(ctx, workspaceDir, repoDir, input.Org)

	docsDir := core.JoinPath(workspaceDir, ".core", "reference", "docs")
	if !fs.IsDir(docsDir) {
		docsRepo := core.JoinPath(s.codePath, input.Org, "docs")
		if fs.IsDir(core.JoinPath(docsRepo, ".git")) {
			process.RunIn(ctx, ".", "git", "clone", "--depth", "1", docsRepo, docsDir)
		}
	}

	s.copyRepoSpecs(workspaceDir, input.Repo)

	out.Prompt, out.Memories, out.Consumers = s.buildPrompt(ctx, input, out.Branch, repoPath)

	out.Success = true
	return nil, out, nil
}

// s.copyRepoSpecs("/tmp/workspace", "go-io")   // copies plans/core/go/io/**/RFC*.md → /tmp/workspace/specs/
// s.copyRepoSpecs("/tmp/workspace", "core-bio") // copies plans/core/php/bio/**/RFC*.md → /tmp/workspace/specs/
func (s *PrepSubsystem) copyRepoSpecs(workspaceDir, repo string) {
	fs := (&core.Fs{}).NewUnrestricted()

	plansBase := core.JoinPath(s.codePath, "host-uk", "core", "plans")
	if !fs.IsDir(plansBase) {
		return
	}

	var specDir string
	switch {
	case core.HasPrefix(repo, "go-"):
		pkg := core.TrimPrefix(repo, "go-")
		specDir = core.JoinPath(plansBase, "core", "go", pkg)
	case core.HasPrefix(repo, "core-"):
		mod := core.TrimPrefix(repo, "core-")
		specDir = core.JoinPath(plansBase, "core", "php", mod)
	case repo == "go":
		specDir = core.JoinPath(plansBase, "core", "go")
	default:
		specDir = core.JoinPath(plansBase, "core", repo)
	}

	if !fs.IsDir(specDir) {
		return
	}

	specsDir := core.JoinPath(workspaceDir, "specs")
	fs.EnsureDir(specsDir)

	patterns := []string{
		core.JoinPath(specDir, "RFC*.md"),
		core.JoinPath(specDir, "*", "RFC*.md"),
		core.JoinPath(specDir, "*", "*", "RFC*.md"),
		core.JoinPath(specDir, "*", "*", "*", "RFC*.md"),
	}
	for _, pattern := range patterns {
		for _, entry := range core.PathGlob(pattern) {
			rel := entry[len(specDir)+1:]
			dst := core.JoinPath(specsDir, rel)
			fs.EnsureDir(core.PathDir(dst))
			r := fs.Read(entry)
			if r.OK {
				fs.Write(dst, r.Value.(string))
			}
		}
	}
}

// _, out, err := prep.PrepareWorkspace(ctx, input)
func (s *PrepSubsystem) PrepareWorkspace(ctx context.Context, input PrepInput) (*mcp.CallToolResult, PrepOutput, error) {
	return s.prepWorkspace(ctx, nil, input)
}

// prompt, memories, consumers := prep.BuildPrompt(ctx, input, "dev", repoPath)
func (s *PrepSubsystem) BuildPrompt(ctx context.Context, input PrepInput, branch, repoPath string) (string, int, int) {
	return s.buildPrompt(ctx, input, branch, repoPath)
}

// prompt, memories, consumers := prep.buildPrompt(ctx, input, "dev", "/srv/repos/go-io")
func (s *PrepSubsystem) buildPrompt(ctx context.Context, input PrepInput, branch, repoPath string) (string, int, int) {
	b := core.NewBuilder()
	memories := 0
	consumers := 0

	b.WriteString("TASK: ")
	b.WriteString(input.Task)
	b.WriteString("\n\n")

	b.WriteString(core.Sprintf("REPO: %s/%s on branch %s\n", input.Org, input.Repo, branch))
	b.WriteString(core.Sprintf("LANGUAGE: %s\n", detectLanguage(repoPath)))
	b.WriteString(core.Sprintf("BUILD: %s\n", detectBuildCmd(repoPath)))
	b.WriteString(core.Sprintf("TEST: %s\n\n", detectTestCmd(repoPath)))

	if input.Persona != "" {
		if r := lib.Persona(input.Persona); r.OK {
			b.WriteString("PERSONA:\n")
			b.WriteString(r.Value.(string))
			b.WriteString("\n\n")
		}
	}

	if r := lib.Flow(detectLanguage(repoPath)); r.OK {
		b.WriteString("WORKFLOW:\n")
		b.WriteString(r.Value.(string))
		b.WriteString("\n\n")
	}

	if input.Issue > 0 {
		if body := s.getIssueBody(ctx, input.Org, input.Repo, input.Issue); body != "" {
			b.WriteString("ISSUE:\n")
			b.WriteString(body)
			b.WriteString("\n\n")
		}
	}

	if recall, count := s.brainRecall(ctx, input.Repo); recall != "" {
		b.WriteString("CONTEXT (from OpenBrain):\n")
		b.WriteString(recall)
		b.WriteString("\n\n")
		memories = count
	}

	if list, count := s.findConsumersList(input.Repo); list != "" {
		b.WriteString("CONSUMERS (modules that import this repo):\n")
		b.WriteString(list)
		b.WriteString("\n\n")
		consumers = count
	}

	if log := s.getGitLog(repoPath); log != "" {
		b.WriteString("RECENT CHANGES:\n```\n")
		b.WriteString(log)
		b.WriteString("```\n\n")
	}

	if input.PlanTemplate != "" {
		if plan := s.renderPlan(input.PlanTemplate, input.Variables, input.Task); plan != "" {
			b.WriteString("PLAN:\n")
			b.WriteString(plan)
			b.WriteString("\n\n")
		}
	}

	b.WriteString("CONSTRAINTS:\n")
	b.WriteString("- Read CODEX.md for coding conventions (if it exists)\n")
	b.WriteString("- Read CLAUDE.md for project-specific instructions (if it exists)\n")
	b.WriteString("- Commit with conventional commit format: type(scope): description\n")
	b.WriteString("- Co-Authored-By: Virgil <virgil@lethean.io>\n")
	b.WriteString("- Run build and tests before committing\n")

	return b.String(), memories, consumers
}

func (s *PrepSubsystem) getIssueBody(ctx context.Context, org, repo string, issue int) string {
	idx := core.Sprintf("%d", issue)
	iss, err := s.forge.Issues.Get(ctx, forge.Params{"owner": org, "repo": repo, "index": idx})
	if err != nil {
		return ""
	}
	return core.Sprintf("# %s\n\n%s", iss.Title, iss.Body)
}

func (s *PrepSubsystem) brainRecall(ctx context.Context, repo string) (string, int) {
	if s.brainKey == "" {
		return "", 0
	}

	body := core.JSONMarshalString(map[string]any{
		"query":    core.Concat("architecture conventions key interfaces for ", repo),
		"top_k":    10,
		"project":  repo,
		"agent_id": "cladius",
	})

	r := HTTPPost(ctx, core.Concat(s.brainURL, "/v1/brain/recall"), body, s.brainKey, "Bearer")
	if !r.OK {
		return "", 0
	}

	var result struct {
		Memories []map[string]any `json:"memories"`
	}
	core.JSONUnmarshalString(r.Value.(string), &result)

	if len(result.Memories) == 0 {
		return "", 0
	}

	b := core.NewBuilder()
	for i, mem := range result.Memories {
		memType, _ := mem["type"].(string)
		memContent, _ := mem["content"].(string)
		memProject, _ := mem["project"].(string)
		b.WriteString(core.Sprintf("%d. [%s] %s: %s\n", i+1, memType, memProject, memContent))
	}

	return b.String(), len(result.Memories)
}

func (s *PrepSubsystem) findConsumersList(repo string) (string, int) {
	goWorkPath := core.JoinPath(s.codePath, "go.work")
	modulePath := core.Concat("forge.lthn.ai/core/", repo)

	r := fs.Read(goWorkPath)
	if !r.OK {
		return "", 0
	}
	workData := r.Value.(string)

	var consumers []string
	for _, line := range core.Split(workData, "\n") {
		line = core.Trim(line)
		if !core.HasPrefix(line, "./") {
			continue
		}
		dir := core.JoinPath(s.codePath, core.TrimPrefix(line, "./"))
		goMod := core.JoinPath(dir, "go.mod")
		mr := fs.Read(goMod)
		if !mr.OK {
			continue
		}
		modData := mr.Value.(string)
		if core.Contains(modData, modulePath) && !core.HasPrefix(modData, core.Concat("module ", modulePath)) {
			consumers = append(consumers, core.PathBase(dir))
		}
	}

	if len(consumers) == 0 {
		return "", 0
	}

	b := core.NewBuilder()
	for _, c := range consumers {
		b.WriteString(core.Concat("- ", c, "\n"))
	}
	b.WriteString(core.Sprintf("Breaking change risk: %d consumers.\n", len(consumers)))

	return b.String(), len(consumers)
}

func (s *PrepSubsystem) getGitLog(repoPath string) string {
	r := s.Core().Process().RunIn(context.Background(), repoPath, "git", "log", "--oneline", "-20")
	if !r.OK {
		return ""
	}
	return core.Trim(r.Value.(string))
}

func (s *PrepSubsystem) pullWikiContent(ctx context.Context, org, repo string) string {
	pages, err := s.forge.Wiki.ListPages(ctx, org, repo)
	if err != nil || len(pages) == 0 {
		return ""
	}

	b := core.NewBuilder()
	for _, meta := range pages {
		name := meta.SubURL
		if name == "" {
			name = meta.Title
		}
		page, pageErr := s.forge.Wiki.GetPage(ctx, org, repo, name)
		if pageErr != nil || page.ContentBase64 == "" {
			continue
		}
		content, _ := base64.StdEncoding.DecodeString(page.ContentBase64)
		b.WriteString(core.Concat("### ", meta.Title, "\n\n"))
		b.WriteString(string(content))
		b.WriteString("\n\n")
	}
	return b.String()
}

func (s *PrepSubsystem) renderPlan(templateSlug string, variables map[string]string, task string) string {
	definition, err := loadPlanTemplateDefinition(templateSlug, variables)
	if err != nil {
		return ""
	}
	return renderPlanMarkdown(definition, task)
}

func detectLanguage(repoPath string) string {
	checks := []struct {
		file string
		lang string
	}{
		{"go.mod", "go"},
		{"composer.json", "php"},
		{"package.json", "ts"},
		{"Cargo.toml", "rust"},
		{"requirements.txt", "py"},
		{"CMakeLists.txt", "cpp"},
		{"Dockerfile", "docker"},
	}
	for _, c := range checks {
		if fs.IsFile(core.JoinPath(repoPath, c.file)) {
			return c.lang
		}
	}
	return "go"
}

func detectBuildCmd(repoPath string) string {
	switch detectLanguage(repoPath) {
	case "go":
		return "go build ./..."
	case "php":
		return "composer install"
	case "ts":
		return "npm run build"
	case "py":
		return "pip install -e ."
	case "rust":
		return "cargo build"
	case "cpp":
		return "cmake --build ."
	default:
		return "go build ./..."
	}
}

func detectTestCmd(repoPath string) string {
	switch detectLanguage(repoPath) {
	case "go":
		return "go test ./..."
	case "php":
		return "composer test"
	case "ts":
		return "npm test"
	case "py":
		return "pytest"
	case "rust":
		return "cargo test"
	case "cpp":
		return "ctest"
	default:
		return "go test ./..."
	}
}
