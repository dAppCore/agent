// SPDX-License-Identifier: EUPL-1.2

// core.New(core.WithService(agentic.Register))
package agentic

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"sync"
	"time"

	"dappco.re/go/agent/pkg/lib"
	core "dappco.re/go/core"
	"dappco.re/go/core/forge"
	coremcp "dappco.re/go/mcp/pkg/mcp"
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
	drainCh        chan struct{}
	pokeCh         chan struct{}
	frozen         bool
	backoff        map[string]time.Time
	failCount      map[string]int
	providers      *ProviderManager
	workspaces     *core.Registry[*WorkspaceStatus]
	stateOnce      sync.Once
	state          *stateStoreRef
	workspaceStatsOnce sync.Once
	workspaceStats     *workspaceStatsRef
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

	subsystem := &PrepSubsystem{
		forge:      forge.NewForge(forgeURL, forgeToken),
		forgeURL:   forgeURL,
		forgeToken: forgeToken,
		brainURL:   envOr("CORE_BRAIN_URL", "https://api.lthn.sh"),
		brainKey:   brainKey,
		codePath:   envOr("CODE_PATH", core.JoinPath(home, "Code")),
		drainCh:    make(chan struct{}, 1),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
		workspaces: core.NewRegistry[*WorkspaceStatus](),
	}
	subsystem.loadRuntimeState()
	return subsystem
}

// c.Action("agentic.dispatch").Run(ctx, options)
// core.Println(c.Actions()) // inspect the registered action names at startup
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
		case "agentic.status", "agentic.scan", "agentic.watch", "agentic.workspace.stats",
			"agentic.issue.get", "agentic.issue.list", "agentic.issue.assign", "agentic.pr.get", "agentic.pr.list",
			"agentic.prompt", "agentic.task", "agentic.flow", "agentic.persona",
			"agentic.prompt.version", "agentic.setup",
			"agentic.sync.status", "agentic.fleet.nodes", "agentic.fleet.stats", "agentic.fleet.events",
			"agentic.credits.balance", "agentic.credits.history",
			"agentic.subscription.detect", "agentic.subscription.budget",
			"agentic.message.send", "agentic.message.inbox", "agentic.message.conversation":
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
	c.Action("agentic.auth.login", s.handleAuthLogin).Description = "Exchange a 6-digit pairing code for an AgentApiKey"
	c.Action("agent.auth.login", s.handleAuthLogin).Description = "Exchange a 6-digit pairing code for an AgentApiKey"
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
	c.Action("agentic.message.send", s.handleMessageSend).Description = "Send a direct message between agents"
	c.Action("agent.message.send", s.handleMessageSend).Description = "Send a direct message between agents"
	c.Action("agentic.message.inbox", s.handleMessageInbox).Description = "List direct messages for an agent"
	c.Action("agent.message.inbox", s.handleMessageInbox).Description = "List direct messages for an agent"
	c.Action("agentic.message.conversation", s.handleMessageConversation).Description = "List a direct conversation between two agents"
	c.Action("agent.message.conversation", s.handleMessageConversation).Description = "List a direct conversation between two agents"

	c.Action("agentic.dispatch", s.handleDispatch).Description = "Prep workspace and spawn a subagent"
	c.Action("agentic.dispatch.sync", s.handleDispatchSync).Description = "Dispatch a single task synchronously and block until it completes"
	c.Action("agentic.dispatch.start", s.handleDispatchStart).Description = "Start the dispatch queue runner"
	c.Action("agentic.dispatch.shutdown", s.handleDispatchShutdown).Description = "Freeze the dispatch queue gracefully"
	c.Action("agentic.dispatch.shutdown_now", s.handleDispatchShutdownNow).Description = "Hard stop the dispatch queue and kill running agents"
	c.Action("agentic.prep", s.handlePrep).Description = "Clone repo and build agent prompt"
	c.Action("agentic.status", s.handleStatus).Description = "List workspace states (running/completed/blocked)"
	c.Action("agentic.resume", s.handleResume).Description = "Resume a blocked or completed workspace"
	c.Action("agentic.scan", s.handleScan).Description = "Scan Forge repos for actionable issues"
	c.Action("agentic.watch", s.handleWatch).Description = "Watch workspace for changes and report"
	c.Action("agentic.workspace.stats", s.handleWorkspaceStats).Description = "List permanent dispatch stats from the parent workspace store"
	c.Action("workspace.stats", s.handleWorkspaceStats).Description = "List permanent dispatch stats from the parent workspace store"

	c.Action("agentic.qa", s.handleQA).Description = "Run build + test QA checks on workspace"
	c.Action("agentic.auto-pr", s.handleAutoPR).Description = "Create PR from completed workspace"
	c.Action("agentic.verify", s.handleVerify).Description = "Verify PR and auto-merge if clean"
	c.Action("agentic.commit", s.handleCommit).Description = "Write the final dispatch record to the workspace journal"
	c.Action("agentic.ingest", s.handleIngest).Description = "Create issues from agent findings"
	c.Action("agentic.poke", s.handlePoke).Description = "Drain next queued task from the queue"
	c.Action("agentic.mirror", s.handleMirror).Description = "Mirror agent branches to GitHub"
	c.Action("agentic.setup", s.handleSetup).Description = "Scaffold a workspace with .core config files and optional templates"

	c.Action("agentic.issue.get", s.handleIssueGet).Description = "Get a Forge issue by number"
	c.Action("agentic.issue.list", s.handleIssueList).Description = "List Forge issues for a repo"
	c.Action("agentic.issue.create", s.handleIssueCreate).Description = "Create a Forge issue"
	c.Action("agentic.issue.update", s.handleIssueRecordUpdate).Description = "Update a tracked platform issue by slug"
	c.Action("agentic.issue.comment", s.handleIssueRecordComment).Description = "Add a comment to a tracked platform issue"
	c.Action("agentic.issue.archive", s.handleIssueRecordArchive).Description = "Archive a tracked platform issue by slug"
	c.Action("agentic.pr.get", s.handlePRGet).Description = "Get a Forge PR by number"
	c.Action("agentic.pr.list", s.handlePRList).Description = "List Forge PRs for a repo"
	c.Action("agentic.pr.merge", s.handlePRMerge).Description = "Merge a Forge PR"
	c.Action("agentic.pr.close", s.handlePRClose).Description = "Close a Forge PR"
	c.Action("agentic.branch.delete", s.handleBranchDelete).Description = "Delete a branch on the Forge remote"
	c.Action("agent.branch.delete", s.handleBranchDelete).Description = "Delete a branch on the Forge remote"

	c.Action("agentic.review-queue", s.handleReviewQueue).Description = "Run CodeRabbit review on completed workspaces"

	c.Action("agentic.epic", s.handleEpic).Description = "Create sub-issues from an epic plan"
	c.Action("plan.create", s.handlePlanCreate).Description = "Create a structured implementation plan"
	c.Action("plan.get", s.handlePlanGet).Description = "Read an implementation plan by ID or slug"
	c.Action("plan.read", s.handlePlanRead).Description = "Read an implementation plan by ID"
	c.Action("plan.update", s.handlePlanUpdate).Description = "Update plan status, phases, notes, or agent assignment"
	c.Action("plan.update_status", s.handlePlanUpdateStatus).Description = "Update an implementation plan lifecycle status by slug"
	c.Action("plan.from.issue", s.handlePlanFromIssue).Description = "Create a plan from a tracked issue"
	c.Action("plan.check", s.handlePlanCheck).Description = "Check whether a plan or phase is complete"
	c.Action("plan.archive", s.handlePlanArchive).Description = "Archive an implementation plan by slug"
	c.Action("plan.delete", s.handlePlanDelete).Description = "Delete an implementation plan by ID"
	c.Action("plan.list", s.handlePlanList).Description = "List implementation plans with optional filters"
	c.Action("agentic.plan.create", s.handlePlanCreate).Description = "Create a structured implementation plan"
	c.Action("agentic.plan.get", s.handlePlanGet).Description = "Read an implementation plan by ID or slug"
	c.Action("agentic.plan.read", s.handlePlanRead).Description = "Read an implementation plan by ID"
	c.Action("agentic.plan.update", s.handlePlanUpdate).Description = "Update plan status, phases, notes, or agent assignment"
	c.Action("agentic.plan.update_status", s.handlePlanUpdateStatus).Description = "Update an implementation plan lifecycle status by slug"
	c.Action("agentic.plan.from.issue", s.handlePlanFromIssue).Description = "Create a plan from a tracked issue"
	c.Action("agentic.plan.check", s.handlePlanCheck).Description = "Check whether a plan or phase is complete"
	c.Action("agentic.plan.archive", s.handlePlanArchive).Description = "Archive an implementation plan by slug"
	c.Action("agentic.plan.delete", s.handlePlanDelete).Description = "Delete an implementation plan by ID"
	c.Action("agentic.plan.list", s.handlePlanList).Description = "List implementation plans with optional filters"
	c.Action("phase.get", s.handlePhaseGet).Description = "Read a plan phase by slug and order"
	c.Action("phase.update_status", s.handlePhaseUpdateStatus).Description = "Update plan phase status by slug and order"
	c.Action("phase.add_checkpoint", s.handlePhaseAddCheckpoint).Description = "Append a checkpoint note to a plan phase"
	c.Action("agentic.phase.get", s.handlePhaseGet).Description = "Read a plan phase by slug and order"
	c.Action("agentic.phase.update_status", s.handlePhaseUpdateStatus).Description = "Update plan phase status by slug and order"
	c.Action("agentic.phase.add_checkpoint", s.handlePhaseAddCheckpoint).Description = "Append a checkpoint note to a plan phase"
	c.Action("task.create", s.handleTaskCreate).Description = "Create a plan task in a phase"
	c.Action("task.update", s.handleTaskUpdate).Description = "Update a plan task by slug, phase, and identifier"
	c.Action("task.toggle", s.handleTaskToggle).Description = "Toggle a plan task between pending and completed"
	c.Action("agentic.task.create", s.handleTaskCreate).Description = "Create a plan task in a phase"
	c.Action("agentic.task.update", s.handleTaskUpdate).Description = "Update a plan task by slug, phase, and identifier"
	c.Action("agentic.task.toggle", s.handleTaskToggle).Description = "Toggle a plan task between pending and completed"
	c.Action("session.start", s.handleSessionStart).Description = "Start an agent session for a plan"
	c.Action("session.get", s.handleSessionGet).Description = "Read a session by session ID"
	c.Action("session.list", s.handleSessionList).Description = "List sessions with optional plan or status filters"
	c.Action("session.continue", s.handleSessionContinue).Description = "Continue a session from its latest saved context"
	c.Action("session.end", s.handleSessionEnd).Description = "End a session with status and summary"
	c.Action("session.complete", s.handleSessionEnd).Description = "Mark a session completed with status, summary, and handoff notes"
	c.Action("session.log", s.handleSessionLog).Description = "Append a typed work-log entry to a stored session"
	c.Action("session.artifact", s.handleSessionArtifact).Description = "Record a created, modified, deleted, or reviewed artifact for a session"
	c.Action("session.handoff", s.handleSessionHandoff).Description = "Hand off a session with notes for the next agent"
	c.Action("session.resume", s.handleSessionResume).Description = "Resume a paused or handed-off session from local cache"
	c.Action("session.replay", s.handleSessionReplay).Description = "Build replay context for a session from work logs and artifacts"
	c.Action("agentic.session.start", s.handleSessionStart).Description = "Start an agent session for a plan"
	c.Action("agentic.session.get", s.handleSessionGet).Description = "Read a session by session ID"
	c.Action("agentic.session.list", s.handleSessionList).Description = "List sessions with optional plan or status filters"
	c.Action("agentic.session.continue", s.handleSessionContinue).Description = "Continue a session from its latest saved context"
	c.Action("agentic.session.end", s.handleSessionEnd).Description = "End a session with status and summary"
	c.Action("agentic.session.complete", s.handleSessionEnd).Description = "Mark a session completed with status, summary, and handoff notes"
	c.Action("agentic.session.log", s.handleSessionLog).Description = "Append a typed work-log entry to a stored session"
	c.Action("agentic.session.artifact", s.handleSessionArtifact).Description = "Record a created, modified, deleted, or reviewed artifact for a session"
	c.Action("agentic.session.handoff", s.handleSessionHandoff).Description = "Hand off a session with notes for the next agent"
	c.Action("agentic.session.resume", s.handleSessionResume).Description = "Resume a paused or handed-off session from local cache"
	c.Action("agentic.session.replay", s.handleSessionReplay).Description = "Build replay context for a session from work logs and artifacts"
	c.Action("state.set", s.handleStateSet).Description = "Store shared plan state for later sessions"
	c.Action("state.get", s.handleStateGet).Description = "Read shared plan state by key"
	c.Action("state.list", s.handleStateList).Description = "List shared plan state for a plan"
	c.Action("state.delete", s.handleStateDelete).Description = "Delete shared plan state by key"
	c.Action("agentic.state.set", s.handleStateSet).Description = "Store shared plan state for later sessions"
	c.Action("agentic.state.get", s.handleStateGet).Description = "Read shared plan state by key"
	c.Action("agentic.state.list", s.handleStateList).Description = "List shared plan state for a plan"
	c.Action("agentic.state.delete", s.handleStateDelete).Description = "Delete shared plan state by key"
	c.Action("template.list", s.handleTemplateList).Description = "List available YAML plan templates"
	c.Action("agentic.template.list", s.handleTemplateList).Description = "List available YAML plan templates"
	c.Action("template.preview", s.handleTemplatePreview).Description = "Preview a YAML plan template with variable substitution"
	c.Action("agentic.template.preview", s.handleTemplatePreview).Description = "Preview a YAML plan template with variable substitution"
	c.Action("template.create_plan", s.handleTemplateCreatePlan).Description = "Create a stored plan from a YAML template"
	c.Action("agentic.template.create_plan", s.handleTemplateCreatePlan).Description = "Create a stored plan from a YAML template"
	c.Action("issue.create", s.handleIssueRecordCreate).Description = "Create a tracked platform issue"
	c.Action("issue.get", s.handleIssueRecordGet).Description = "Read a tracked platform issue by slug"
	c.Action("issue.list", s.handleIssueRecordList).Description = "List tracked platform issues with optional filters"
	c.Action("issue.update", s.handleIssueRecordUpdate).Description = "Update a tracked platform issue by slug"
	c.Action("issue.assign", s.handleIssueRecordAssign).Description = "Assign an agent or user to a tracked platform issue"
	c.Action("issue.comment", s.handleIssueRecordComment).Description = "Add a comment to a tracked platform issue"
	c.Action("issue.report", s.handleIssueRecordReport).Description = "Post a structured report comment to a tracked platform issue"
	c.Action("issue.archive", s.handleIssueRecordArchive).Description = "Archive a tracked platform issue by slug"
	c.Action("agentic.issue.create", s.handleIssueRecordCreate).Description = "Create a tracked platform issue"
	c.Action("agentic.issue.assign", s.handleIssueRecordAssign).Description = "Assign an agent or user to a tracked platform issue"
	c.Action("agentic.issue.comment", s.handleIssueRecordComment).Description = "Add a comment to a tracked platform issue"
	c.Action("agentic.issue.report", s.handleIssueRecordReport).Description = "Post a structured report comment to a tracked platform issue"
	c.Action("sprint.create", s.handleSprintCreate).Description = "Create a tracked platform sprint"
	c.Action("sprint.get", s.handleSprintGet).Description = "Read a tracked platform sprint by slug"
	c.Action("sprint.list", s.handleSprintList).Description = "List tracked platform sprints with optional filters"
	c.Action("sprint.update", s.handleSprintUpdate).Description = "Update a tracked platform sprint by slug"
	c.Action("sprint.archive", s.handleSprintArchive).Description = "Archive a tracked platform sprint by slug"
	c.Action("content.generate", s.handleContentGenerate).Description = "Generate content using the platform content pipeline"
	c.Action("agentic.generate", s.handleContentGenerate).Description = "Generate content using the platform content pipeline"
	c.Action("agentic.content.generate", s.handleContentGenerate).Description = "Generate content using the platform content pipeline"
	c.Action("content.batch", s.handleContentBatchGenerate).Description = "Start or continue batch content generation"
	c.Action("content.batch.generate", s.handleContentBatchGenerate).Description = "Start or continue batch content generation"
	c.Action("content.batch_generate", s.handleContentBatchGenerate).Description = "Start or continue batch content generation"
	c.Action("content_batch", s.handleContentBatchGenerate).Description = "Start or continue batch content generation"
	c.Action("agentic.content.batch", s.handleContentBatchGenerate).Description = "Start or continue batch content generation"
	c.Action("agentic.content.batch.generate", s.handleContentBatchGenerate).Description = "Start or continue batch content generation"
	c.Action("agentic.content.batch_generate", s.handleContentBatchGenerate).Description = "Start or continue batch content generation"
	c.Action("agentic.content_batch", s.handleContentBatchGenerate).Description = "Start or continue batch content generation"
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
	c.Action("content.schema.generate", s.handleContentSchemaGenerate).Description = "Generate SEO schema JSON-LD for article, FAQ, or how-to content"
	c.Action("agentic.content.brief.create", s.handleContentBriefCreate).Description = "Create a reusable content brief"
	c.Action("agentic.content.brief.get", s.handleContentBriefGet).Description = "Read a content brief by ID or slug"
	c.Action("agentic.content.brief.list", s.handleContentBriefList).Description = "List content briefs with optional filters"
	c.Action("agentic.content.status", s.handleContentStatus).Description = "Read batch content generation status"
	c.Action("agentic.content.usage.stats", s.handleContentUsageStats).Description = "Read content provider usage statistics"
	c.Action("agentic.content.usage_stats", s.handleContentUsageStats).Description = "Read content provider usage statistics"
	c.Action("agentic.content.from.plan", s.handleContentFromPlan).Description = "Generate content from plan context"
	c.Action("agentic.content.from_plan", s.handleContentFromPlan).Description = "Generate content from plan context"
	c.Action("agentic.content.schema.generate", s.handleContentSchemaGenerate).Description = "Generate SEO schema JSON-LD for article, FAQ, or how-to content"

	c.Action("agentic.prompt", s.handlePrompt).Description = "Read a system prompt by slug"
	c.Action("agentic.prompt.version", s.handlePromptVersion).Description = "Read the current prompt snapshot for a workspace"
	c.Action("agentic.task", s.handleTask).Description = "Read a task plan by slug"
	c.Action("agentic.flow", s.handleFlow).Description = "Read a build/release flow by slug"
	c.Action("agentic.persona", s.handlePersona).Description = "Read a persona by path"

	c.Task("agent.completion", core.Task{
		Description: "QA → PR → Verify → Commit → Ingest → Poke",
		Steps: []core.Step{
			{Action: "agentic.qa"},
			{Action: "agentic.auto-pr"},
			{Action: "agentic.verify"},
			{Action: "agentic.commit", Async: true},
			{Action: "agentic.ingest", Async: true},
			{Action: "agentic.poke", Async: true},
		},
	})

	c.Action("agentic.complete", s.handleComplete).Description = "Run completion pipeline (QA → PR → Verify → Commit → Ingest → Poke) in background"

	s.hydrateWorkspaces()
	// RFC §15.5 — startup scans `.core/state/` for orphaned QA workspace
	// buffers (leftover DuckDB files from dispatches that crashed before
	// commit) and releases them so the next cycle starts clean.
	s.recoverStateOrphans()
	if planRetentionDays(core.NewOptions()) > 0 {
		go s.runPlanCleanupLoop(ctx, planRetentionScheduleInterval)
	}
	if s.forgeToken != "" {
		go s.runPRManageLoop(ctx, prManageScheduleInterval)
	}
	if s.syncToken() != "" {
		go s.runSyncFlushLoop(ctx, syncFlushScheduleInterval)
	}

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
	s.closeStateStore()
	s.closeWorkspaceStatsStore()
	return core.Result{OK: true}
}

// s.hydrateWorkspaces()
// s.workspaces.Names() // ["core/go-io/task-5", "ws-blocked", "ws-ready-for-review"]
func (s *PrepSubsystem) hydrateWorkspaces() {
	if s.workspaces == nil {
		s.workspaces = core.NewRegistry[*WorkspaceStatus]()
	}

	// Registry hydration is filesystem-first — workspace status.json is
	// authoritative. The go-store registry group caches last-known status
	// so ghost agents can be detected after crashes even when the JSON
	// status file has been rotated. Filesystem wins on conflict (§15.3).
	s.stateStoreRestore(stateRegistryGroup, func(key, value string) bool {
		if s.workspaces.Get(key).OK {
			return true
		}
		var status WorkspaceStatus
		if result := core.JSONUnmarshalString(value, &status); !result.OK {
			return true
		}
		// Dead agents are marked failed so the next dispatch does not
		// block on a PID that disappeared.
		if status.Status == "running" && !ProcessAlive(nil, status.ProcessID, status.PID) {
			status.Status = "failed"
			if status.Question == "" {
				status.Question = "Agent process died during restart"
			}
		}
		s.workspaces.Set(key, &status)
		return true
	})

	for _, path := range WorkspaceStatusPaths() {
		workspaceDir := core.PathDir(path)
		result := ReadStatusResult(workspaceDir)
		st, ok := workspaceStatusValue(result)
		if !ok {
			continue
		}
		// Reap ghost agents: a status.json that claims `running` for a
		// PID that no longer exists is a stale artifact from a crashed
		// dispatch — RFC §15.3 requires no ghost agents after restart.
		// Persist the reaped status back to disk so cmdStatus and any
		// out-of-process consumer see a coherent view.
		if st.Status == "running" && !ProcessAlive(nil, st.ProcessID, st.PID) {
			st.Status = "failed"
			if st.Question == "" {
				st.Question = "Agent process died during restart"
			}
			writeStatusResult(workspaceDir, st)
		}
		s.workspaces.Set(WorkspaceName(workspaceDir), st)
	}
}

// s.TrackWorkspace("core/go-io/task-5", st)
//
// TrackWorkspace mirrors the workspace status into the go-store registry
// group, the queue group (when status="queued") and the concurrency group
// (running counts per agent type) so a restart restores all three slices of
// dispatch state described in RFC §15.3.
func (s *PrepSubsystem) TrackWorkspace(name string, st *WorkspaceStatus) {
	if s.workspaces != nil {
		s.workspaces.Set(name, st)
	}
	if st == nil {
		s.stateStoreDelete(stateRegistryGroup, name)
		s.stateStoreDelete(stateQueueGroup, name)
		s.refreshConcurrencySnapshot()
		return
	}
	s.stateStoreSet(stateRegistryGroup, name, st)

	// Queue group keeps the spec-shaped `{repo}/{branch}` index of
	// dispatch slots that have not started running yet (RFC §15.3).
	if st.Status == "queued" {
		s.stateStoreSet(stateQueueGroup, name, queueEntryFromStatus(st))
	} else {
		s.stateStoreDelete(stateQueueGroup, name)
	}

	s.refreshConcurrencySnapshot()
}

// queueEntry is the JSON shape persisted under stateQueueGroup so the dispatch
// queue survives restart per RFC §15.3.
//
// Usage example: `entry := queueEntryFromStatus(workspaceStatus)`
type queueEntry struct {
	Repo     string    `json:"repo"`
	Branch   string    `json:"branch,omitempty"`
	Org      string    `json:"org,omitempty"`
	Task     string    `json:"task,omitempty"`
	Agent    string    `json:"agent,omitempty"`
	Status   string    `json:"status,omitempty"`
	Priority int       `json:"priority,omitempty"`
	QueuedAt time.Time `json:"queued_at"`
}

// queueEntryFromStatus projects the dispatch fields RFC §15.3 records into the
// queue group from the live WorkspaceStatus.
//
// Usage example: `entry := queueEntryFromStatus(&WorkspaceStatus{Repo: "go-io", Branch: "agent/fix-tests"})`
func queueEntryFromStatus(st *WorkspaceStatus) queueEntry {
	if st == nil {
		return queueEntry{}
	}
	queuedAt := st.UpdatedAt
	if queuedAt.IsZero() {
		queuedAt = st.StartedAt
	}
	return queueEntry{
		Repo:     st.Repo,
		Branch:   st.Branch,
		Org:      st.Org,
		Task:     st.Task,
		Agent:    st.Agent,
		Status:   st.Status,
		QueuedAt: queuedAt,
	}
}

// refreshConcurrencySnapshot writes a `{agent-type}` snapshot of currently
// running dispatch counts into stateConcurrencyGroup so RFC §15.3 ghost-agent
// detection has authoritative pre-restart counts to compare against.
//
// Usage example: `s.refreshConcurrencySnapshot()`
func (s *PrepSubsystem) refreshConcurrencySnapshot() {
	if s == nil || s.workspaces == nil {
		return
	}
	if s.stateStoreInstance() == nil {
		return
	}

	counts := map[string]int{}
	totals := map[string]int{}
	s.workspaces.Each(func(_ string, workspaceStatus *WorkspaceStatus) {
		if workspaceStatus == nil || workspaceStatus.Agent == "" {
			return
		}
		base := baseAgent(workspaceStatus.Agent)
		totals[base]++
		if workspaceStatus.Status == "running" {
			counts[base]++
		}
	})

	seen := map[string]struct{}{}
	for base, running := range counts {
		entry := map[string]any{
			"running":     running,
			"tracked":     totals[base],
			"snapshot_at": time.Now().UTC(),
		}
		s.stateStoreSet(stateConcurrencyGroup, base, entry)
		seen[base] = struct{}{}
	}
	for base, tracked := range totals {
		if _, ok := seen[base]; ok {
			continue
		}
		entry := map[string]any{
			"running":     0,
			"tracked":     tracked,
			"snapshot_at": time.Now().UTC(),
		}
		s.stateStoreSet(stateConcurrencyGroup, base, entry)
	}

	// Drop entries for agent types we no longer track so the snapshot
	// never grows beyond active dispatch pools.
	s.stateStoreRestore(stateConcurrencyGroup, func(key, _ string) bool {
		if _, alive := totals[key]; !alive {
			s.stateStoreDelete(stateConcurrencyGroup, key)
		}
		return true
	})
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
// subsystem.SetCore(core.New(core.WithOption("name", "core-agent")))
func (s *PrepSubsystem) SetCore(c *core.Core) {
	if s == nil || c == nil {
		return
	}
	s.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})
}

// subsystem := agentic.NewPrep()
// subsystem.RegisterTools(svc)
func (s *PrepSubsystem) RegisterTools(svc *coremcp.Service) {
	mcp.AddTool(svc.Server(), &mcp.Tool{
		Name:        "agentic_prep_workspace",
		Description: "Prepare an agent workspace: clone repo, create branch, build prompt with context.",
	}, s.prepWorkspace)
	s.registerDispatchTool(svc)
	s.registerStatusTool(svc)
	s.registerResumeTool(svc)
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_complete",
		Description: "Run the completion pipeline (QA → PR → Verify → Commit → Ingest → Poke) in the background.",
	}, s.completeTool)
	s.registerCommitTool(svc)
	s.registerCreatePRTool(svc)
	s.registerListPRsTool(svc)
	s.registerClosePRTool(svc)
	s.registerDeleteBranchTool(svc)
	s.registerMirrorTool(svc)
	s.registerShutdownTools(svc)
	s.registerPlanTools(svc)
	s.registerWatchTool(svc)
	s.registerIssueTools(svc)
	s.registerPRTools(svc)
	mcp.AddTool(svc.Server(), &mcp.Tool{
		Name:        "agentic_scan",
		Description: "Scan Forge repos for open issues with actionable labels (agentic, help-wanted, bug).",
	}, s.scan)

	// Extended tools — only when CORE_MCP_FULL=1
	if core.Env("CORE_MCP_FULL") != "1" {
		return
	}
	s.registerEpicTool(svc)
	s.registerRemoteDispatchTool(svc)
	s.registerRemoteStatusTool(svc)
	s.registerReviewQueueTool(svc)
	s.registerPlatformTools(svc)
	s.registerSessionTools(svc)
	s.registerStateTools(svc)
	s.registerPhaseTools(svc)
	s.registerTaskTools(svc)
	s.registerPromptTools(svc)
	s.registerTemplateTools(svc)
	s.registerMessageTools(svc)
	s.registerSprintTools(svc)
	s.registerContentTools(svc)
	s.registerLanguageTools(svc)
	s.registerSetupTool(svc)

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_scan",
		Description: "Scan Forge repos for open issues with actionable labels (agentic, help-wanted, bug).",
	}, s.scan)

	s.registerPlanTools(svc)
	s.registerWatchTool(svc)
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
	Success       bool   `json:"success"`
	WorkspaceDir  string `json:"workspace_dir"`
	RepoDir       string `json:"repo_dir"`
	Branch        string `json:"branch"`
	Prompt        string `json:"prompt,omitempty"`
	PromptVersion string `json:"prompt_version,omitempty"`
	Memories      int    `json:"memories"`
	Consumers     int    `json:"consumers"`
	Resumed       bool   `json:"resumed"`
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
	orgName, ok := validateName(org)
	if !ok {
		return core.Result{Value: core.E("workspaceDir", "invalid org name", nil), OK: false}
	}

	repoName, ok := validateName(repo)
	if !ok {
		return core.Result{Value: core.E("workspaceDir", "invalid repo name", nil), OK: false}
	}

	base := core.JoinPath(WorkspaceRoot(), orgName, repoName)
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
	if err := s.runWorkspaceLanguagePrep(ctx, workspaceDir, repoDir); err != nil {
		return nil, PrepOutput{}, err
	}

	docsDir := core.JoinPath(workspaceDir, ".core", "reference", "docs")
	if !fs.IsDir(docsDir) {
		docsRepo := core.JoinPath(s.codePath, input.Org, "docs")
		if fs.IsDir(core.JoinPath(docsRepo, ".git")) {
			process.RunIn(ctx, ".", "git", "clone", "--depth", "1", docsRepo, docsDir)
		}
	}

	s.copyRepoSpecs(workspaceDir, input.Repo)

	out.Prompt, out.Memories, out.Consumers = s.buildPrompt(ctx, input, out.Branch, repoPath)
	if versionResult := writePromptSnapshot(workspaceDir, out.Prompt); !versionResult.OK {
		err, _ := versionResult.Value.(error)
		if err == nil {
			err = core.E("prepWorkspace", "failed to write prompt snapshot", nil)
		}
		return nil, PrepOutput{}, err
	} else if version, ok := versionResult.Value.(string); ok {
		out.PromptVersion = version
	}

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

// _, out, err := prep.TestPrepWorkspace(ctx, input)
func (s *PrepSubsystem) TestPrepWorkspace(ctx context.Context, input PrepInput) (*mcp.CallToolResult, PrepOutput, error) {
	return s.prepWorkspace(ctx, nil, input)
}

// prompt, memories, consumers := prep.BuildPrompt(ctx, input, "dev", repoPath)
func (s *PrepSubsystem) BuildPrompt(ctx context.Context, input PrepInput, branch, repoPath string) (string, int, int) {
	return s.buildPrompt(ctx, input, branch, repoPath)
}

// prompt, memories, consumers := prep.TestBuildPrompt(ctx, input, "dev", repoPath)
func (s *PrepSubsystem) TestBuildPrompt(ctx context.Context, input PrepInput, branch, repoPath string) (string, int, int) {
	return s.buildPrompt(ctx, input, branch, repoPath)
}

// prompt, memoryCount, consumerCount := prep.buildPrompt(ctx, input, "dev", "/srv/repos/go-io")
func (s *PrepSubsystem) buildPrompt(ctx context.Context, input PrepInput, branch, repoPath string) (string, int, int) {
	promptBuilder := core.NewBuilder()
	memoryCount := 0
	consumerCount := 0

	promptBuilder.WriteString("TASK: ")
	promptBuilder.WriteString(input.Task)
	promptBuilder.WriteString("\n\n")

	promptBuilder.WriteString(core.Sprintf("REPO: %s/%s on branch %s\n", input.Org, input.Repo, branch))
	promptBuilder.WriteString(core.Sprintf("LANGUAGE: %s\n", detectLanguage(repoPath)))
	promptBuilder.WriteString(core.Sprintf("BUILD: %s\n", detectBuildCmd(repoPath)))
	promptBuilder.WriteString(core.Sprintf("TEST: %s\n\n", detectTestCmd(repoPath)))

	if input.Persona != "" {
		if personaResult := lib.Persona(input.Persona); personaResult.OK {
			promptBuilder.WriteString("PERSONA:\n")
			promptBuilder.WriteString(personaResult.Value.(string))
			promptBuilder.WriteString("\n\n")
		}
	}

	if workflowResult := lib.Flow(detectLanguage(repoPath)); workflowResult.OK {
		promptBuilder.WriteString("WORKFLOW:\n")
		promptBuilder.WriteString(workflowResult.Value.(string))
		promptBuilder.WriteString("\n\n")
	}

	if input.Issue > 0 {
		if issueBody := s.getIssueBody(ctx, input.Org, input.Repo, input.Issue); issueBody != "" {
			promptBuilder.WriteString("ISSUE:\n")
			promptBuilder.WriteString(issueBody)
			promptBuilder.WriteString("\n\n")
		}
	}

	if brainContext, count := s.brainRecall(ctx, input.Repo); brainContext != "" {
		promptBuilder.WriteString("CONTEXT (from OpenBrain):\n")
		promptBuilder.WriteString(brainContext)
		promptBuilder.WriteString("\n\n")
		memoryCount = count
	}

	if consumerList, count := s.findConsumersList(input.Repo); consumerList != "" {
		promptBuilder.WriteString("CONSUMERS (modules that import this repo):\n")
		promptBuilder.WriteString(consumerList)
		promptBuilder.WriteString("\n\n")
		consumerCount = count
	}

	if gitLog := s.getGitLog(repoPath); gitLog != "" {
		promptBuilder.WriteString("RECENT CHANGES:\n```\n")
		promptBuilder.WriteString(gitLog)
		promptBuilder.WriteString("```\n\n")
	}

	if input.PlanTemplate != "" {
		if planText := s.renderPlan(input.PlanTemplate, input.Variables, input.Task); planText != "" {
			promptBuilder.WriteString("PLAN:\n")
			promptBuilder.WriteString(planText)
			promptBuilder.WriteString("\n\n")
		}
	}

	promptBuilder.WriteString("CONSTRAINTS:\n")
	promptBuilder.WriteString("- Read CODEX.md for coding conventions (if it exists)\n")
	promptBuilder.WriteString("- Read CLAUDE.md for project-specific instructions (if it exists)\n")
	promptBuilder.WriteString("- Commit with conventional commit format: type(scope): description\n")
	promptBuilder.WriteString("- Co-Authored-By: Virgil <virgil@lethean.io>\n")
	promptBuilder.WriteString("- Run build and tests before committing\n")

	return promptBuilder.String(), memoryCount, consumerCount
}

// writePromptSnapshot stores an immutable prompt snapshot for a workspace.
//
//	snapshot := writePromptSnapshot("/srv/.core/workspace/core/go-io/task-42", "TASK: Fix tests")
func writePromptSnapshot(workspaceDir, prompt string) core.Result {
	if workspaceDir == "" || core.Trim(prompt) == "" {
		return core.Result{OK: true}
	}

	hash := promptSnapshotHash(prompt)
	snapshot := PromptVersionSnapshot{
		Hash:      hash,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Content:   prompt,
	}

	metaDir := WorkspaceMetaDir(workspaceDir)
	snapshotDir := core.JoinPath(metaDir, "prompt-versions")
	if r := fs.EnsureDir(snapshotDir); !r.OK {
		err, _ := r.Value.(error)
		if err == nil {
			err = core.E("prepWorkspace", "failed to create prompt snapshot directory", nil)
		}
		return core.Result{Value: err, OK: false}
	}

	snapshotPath := core.JoinPath(snapshotDir, core.Concat(hash, ".json"))
	if !fs.Exists(snapshotPath) {
		if r := fs.WriteAtomic(snapshotPath, core.JSONMarshalString(snapshot)); !r.OK {
			err, _ := r.Value.(error)
			if err == nil {
				err = core.E("prepWorkspace", "failed to write prompt snapshot", nil)
			}
			return core.Result{Value: err, OK: false}
		}
	}

	if r := fs.WriteAtomic(core.JoinPath(metaDir, "prompt-version.json"), core.JSONMarshalString(snapshot)); !r.OK {
		err, _ := r.Value.(error)
		if err == nil {
			err = core.E("prepWorkspace", "failed to write prompt version index", nil)
		}
		return core.Result{Value: err, OK: false}
	}

	return core.Result{Value: hash, OK: true}
}

// snapshot := readPromptSnapshot("/srv/.core/workspace/core/go-io/task-42")
func readPromptSnapshot(workspaceDir string) (PromptVersionSnapshot, error) {
	if workspaceDir == "" {
		return PromptVersionSnapshot{}, core.E("readPromptSnapshot", "workspace is required", nil)
	}

	snapshotPath := core.JoinPath(WorkspaceMetaDir(workspaceDir), "prompt-version.json")
	result := fs.Read(snapshotPath)
	if !result.OK {
		err, _ := result.Value.(error)
		if err == nil {
			err = core.E("readPromptSnapshot", "prompt snapshot not found", nil)
		}
		return PromptVersionSnapshot{}, err
	}

	var snapshot PromptVersionSnapshot
	if parseResult := core.JSONUnmarshalString(result.Value.(string), &snapshot); !parseResult.OK {
		err, _ := parseResult.Value.(error)
		return PromptVersionSnapshot{}, core.E("readPromptSnapshot", "failed to parse prompt snapshot", err)
	}
	return snapshot, nil
}

// snapshot := PromptVersionSnapshot{Hash: "f2c8...", Content: "TASK: Fix tests"}
type PromptVersionSnapshot struct {
	Hash      string `json:"hash"`
	CreatedAt string `json:"created_at"`
	Content   string `json:"content"`
}

func promptSnapshotHash(prompt string) string {
	sum := sha256.Sum256([]byte(prompt))
	return hex.EncodeToString(sum[:])
}

// _ = s.runWorkspaceLanguagePrep(ctx, "/srv/.core/workspace/core/go-io/task-42", "/srv/Code/core/go-io")
func (s *PrepSubsystem) runWorkspaceLanguagePrep(ctx context.Context, workspaceDir, repoDir string) error {
	process := s.Core().Process()

	goEnv := []string{
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=url.https://forge.lthn.ai/.insteadOf",
		"GIT_CONFIG_VALUE_0=ssh://git@forge.lthn.ai:2223/",
		"GONOSUMCHECK=forge.lthn.ai/*,dappco.re/*",
		"GOPRIVATE=forge.lthn.ai/*,dappco.re/*",
		"GOFLAGS=-mod=mod",
	}

	if fs.IsFile(core.JoinPath(repoDir, "go.mod")) {
		if result := process.RunWithEnv(ctx, repoDir, goEnv, "go", "mod", "download"); !result.OK {
			return core.E("prepWorkspace", "go mod download failed", nil)
		}
	}

	if fs.IsFile(core.JoinPath(repoDir, "go.mod")) && (fs.IsFile(core.JoinPath(workspaceDir, "go.work")) || fs.IsFile(core.JoinPath(repoDir, "go.work"))) {
		if result := process.RunWithEnv(ctx, repoDir, goEnv, "go", "work", "sync"); !result.OK {
			return core.E("prepWorkspace", "go work sync failed", nil)
		}
	}

	if fs.IsFile(core.JoinPath(repoDir, "composer.json")) {
		if result := process.RunIn(ctx, repoDir, "composer", "install"); !result.OK {
			return core.E("prepWorkspace", "composer install failed", nil)
		}
	}

	if fs.IsFile(core.JoinPath(repoDir, "package.json")) {
		if result := process.RunIn(ctx, repoDir, "npm", "install"); !result.OK {
			return core.E("prepWorkspace", "npm install failed", nil)
		}
	}

	return nil
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
	modulePath := core.Concat("dappco.re/go/core/", repo)

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
	definition, _, err := loadPlanTemplateDefinition(templateSlug, variables)
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
