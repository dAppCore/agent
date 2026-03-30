// SPDX-License-Identifier: EUPL-1.2

// Package agentic provides MCP tools for agent orchestration.
// Prepares workspaces and dispatches subagents.
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
	"gopkg.in/yaml.v3"
)

// AgentOptions configures the agentic service.
//
//	options := agentic.AgentOptions{}
type AgentOptions struct{}

// PrepSubsystem provides agentic MCP tools for workspace orchestration.
// Agent lifecycle events are broadcast via s.Core().ACTION(messages.AgentCompleted{}).
//
//	core.New(core.WithService(agentic.Register))
type PrepSubsystem struct {
	*core.ServiceRuntime[AgentOptions]
	forge          *forge.Forge
	forgeURL       string
	forgeToken     string
	brainURL       string
	brainKey       string
	codePath       string
	startupContext context.Context
	dispatchMu     sync.Mutex // serialises concurrency check + spawn
	drainMu        sync.Mutex
	pokeCh         chan struct{}
	frozen         bool
	backoff        map[string]time.Time             // pool → paused until
	failCount      map[string]int                   // pool → consecutive fast failures
	workspaces     *core.Registry[*WorkspaceStatus] // in-memory workspace state
}

var _ coremcp.Subsystem = (*PrepSubsystem)(nil)

// NewPrep creates an agentic subsystem.
//
//	subsystem := agentic.NewPrep()
//	subsystem.SetCompletionNotifier(monitor)
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

// SetCore wires the Core framework instance via ServiceRuntime.
// Deprecated: Use Register with core.WithService(agentic.Register) instead.
//
//	prep.SetCore(c)
func (s *PrepSubsystem) SetCore(c *core.Core) {
	s.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})
}

// OnStartup implements core.Startable — registers named Actions, starts the queue runner,
// and registers CLI commands. The Action registry IS the capability map.
//
//	c.Action("agentic.dispatch").Run(ctx, options)
//	c.Actions() // ["agentic.dispatch", "agentic.prep", "agentic.status", ...]
func (s *PrepSubsystem) OnStartup(ctx context.Context) core.Result {
	c := s.Core()

	// Entitlement — gates agentic Actions when queue is frozen.
	// Per-agent concurrency is checked inside handlers (needs Options for agent name).
	// Entitlement gates the global capability: "can this Core dispatch at all?"
	//
	//   e := c.Entitled("agentic.dispatch")
	//   e.Allowed  // false when frozen
	//   e.Reason   // "agent queue is frozen"
	c.SetEntitlementChecker(func(action string, qty int, _ context.Context) core.Entitlement {
		// Only gate agentic.* actions
		if !core.HasPrefix(action, "agentic.") {
			return core.Entitlement{Allowed: true, Unlimited: true}
		}
		// Read-only + internal actions always allowed
		if core.HasPrefix(action, "agentic.monitor.") || core.HasPrefix(action, "agentic.complete") {
			return core.Entitlement{Allowed: true, Unlimited: true}
		}
		switch action {
		case "agentic.status", "agentic.scan", "agentic.watch",
			"agentic.issue.get", "agentic.issue.list", "agentic.pr.get", "agentic.pr.list",
			"agentic.prompt", "agentic.task", "agentic.flow", "agentic.persona":
			return core.Entitlement{Allowed: true, Unlimited: true}
		}
		// Write actions gated by frozen state
		if s.frozen {
			return core.Entitlement{Allowed: false, Reason: "agent queue is frozen — shutting down"}
		}
		return core.Entitlement{Allowed: true}
	})

	// Data — mount embedded content so other services can access it via c.Data()
	//
	//   c.Data().ReadString("prompts/coding.md")
	//   c.Data().ListNames("flows")
	lib.MountData(c)

	// Transport — register HTTP protocol + Drive endpoints
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

	// Dispatch & workspace
	c.Action("agentic.dispatch", s.handleDispatch).Description = "Prep workspace and spawn a subagent"
	c.Action("agentic.prep", s.handlePrep).Description = "Clone repo and build agent prompt"
	c.Action("agentic.status", s.handleStatus).Description = "List workspace states (running/completed/blocked)"
	c.Action("agentic.resume", s.handleResume).Description = "Resume a blocked or completed workspace"
	c.Action("agentic.scan", s.handleScan).Description = "Scan Forge repos for actionable issues"
	c.Action("agentic.watch", s.handleWatch).Description = "Watch workspace for changes and report"

	// Pipeline
	c.Action("agentic.qa", s.handleQA).Description = "Run build + test QA checks on workspace"
	c.Action("agentic.auto-pr", s.handleAutoPR).Description = "Create PR from completed workspace"
	c.Action("agentic.verify", s.handleVerify).Description = "Verify PR and auto-merge if clean"
	c.Action("agentic.ingest", s.handleIngest).Description = "Create issues from agent findings"
	c.Action("agentic.poke", s.handlePoke).Description = "Drain next queued task from the queue"
	c.Action("agentic.mirror", s.handleMirror).Description = "Mirror agent branches to GitHub"

	// Forge
	c.Action("agentic.issue.get", s.handleIssueGet).Description = "Get a Forge issue by number"
	c.Action("agentic.issue.list", s.handleIssueList).Description = "List Forge issues for a repo"
	c.Action("agentic.issue.create", s.handleIssueCreate).Description = "Create a Forge issue"
	c.Action("agentic.pr.get", s.handlePRGet).Description = "Get a Forge PR by number"
	c.Action("agentic.pr.list", s.handlePRList).Description = "List Forge PRs for a repo"
	c.Action("agentic.pr.merge", s.handlePRMerge).Description = "Merge a Forge PR"

	// Review
	c.Action("agentic.review-queue", s.handleReviewQueue).Description = "Run CodeRabbit review on completed workspaces"

	// Epic
	c.Action("agentic.epic", s.handleEpic).Description = "Create sub-issues from an epic plan"

	// Content — accessible via IPC, no lib import needed
	c.Action("agentic.prompt", s.handlePrompt).Description = "Read a system prompt by slug"
	c.Action("agentic.task", s.handleTask).Description = "Read a task plan by slug"
	c.Action("agentic.flow", s.handleFlow).Description = "Read a build/release flow by slug"
	c.Action("agentic.persona", s.handlePersona).Description = "Read a persona by path"

	// Completion pipeline — Task composition
	c.Task("agent.completion", core.Task{
		Description: "QA → PR → Verify → Merge",
		Steps: []core.Step{
			{Action: "agentic.qa"},
			{Action: "agentic.auto-pr"},
			{Action: "agentic.verify"},
			{Action: "agentic.ingest", Async: true},
			{Action: "agentic.poke", Async: true},
		},
	})

	// PerformAsync wrapper — runs the completion Task in background with progress tracking.
	// c.PerformAsync("agentic.complete", options) broadcasts ActionTaskStarted/Completed.
	c.Action("agentic.complete", s.handleComplete).Description = "Run completion pipeline (QA → PR → Verify) in background"

	// Hydrate workspace registry from disk
	s.hydrateWorkspaces()

	// QUERY handler — "what workspaces exist?"
	//
	//   r := c.QUERY(agentic.WorkspaceQuery{})
	//   if r.OK { workspaces := r.Value.(*core.Registry[*WorkspaceStatus]) }
	c.RegisterQuery(s.handleWorkspaceQuery)

	s.StartRunner()
	s.registerCommands(ctx)
	s.registerWorkspaceCommands()
	s.registerForgeCommands()
	return core.Result{OK: true}
}

// registerCommands is in commands.go

// OnShutdown implements core.Stoppable and freezes the queue.
//
//	subsystem := agentic.NewPrep()
//	_ = subsystem.OnShutdown(context.Background())
func (s *PrepSubsystem) OnShutdown(ctx context.Context) core.Result {
	s.frozen = true
	return core.Result{OK: true}
}

// hydrateWorkspaces scans disk and populates the workspace Registry on startup.
// Keyed by workspace name (relative path from workspace root).
//
//	s.hydrateWorkspaces()
//	s.workspaces.Names() // ["core/go-io/task-5", "ws-blocked", ...]
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

// TrackWorkspace registers or updates a workspace in the in-memory Registry.
//
//	s.TrackWorkspace("core/go-io/task-5", st)
func (s *PrepSubsystem) TrackWorkspace(name string, st *WorkspaceStatus) {
	if s.workspaces != nil {
		s.workspaces.Set(name, st)
	}
}

// Workspaces returns the workspace Registry for cross-cutting queries.
//
//	s.Workspaces().Names()                        // all workspace names
//	s.Workspaces().List("core/*")                 // org-scoped workspaces
//	s.Workspaces().Each(func(name string, st *WorkspaceStatus) { ... })
func (s *PrepSubsystem) Workspaces() *core.Registry[*WorkspaceStatus] {
	return s.workspaces
}

func envOr(key, fallback string) string {
	if v := core.Env(key); v != "" {
		return v
	}
	return fallback
}

// Name identifies the MCP subsystem.
//
//	subsystem := agentic.NewPrep()
//	name := subsystem.Name()
//	_ = name // "agentic"
func (s *PrepSubsystem) Name() string { return "agentic" }

// RegisterTools publishes the agentic MCP tools on the server.
//
//	subsystem := agentic.NewPrep()
//	subsystem.RegisterTools(server)
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
	s.registerShutdownTools(server)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_scan",
		Description: "Scan Forge repos for open issues with actionable labels (agentic, help-wanted, bug).",
	}, s.scan)

	s.registerPlanTools(server)
	s.registerWatchTool(server)
}

// Shutdown satisfies mcp.SubsystemWithShutdown for clean server teardown.
//
//	subsystem := agentic.NewPrep()
//	_ = subsystem.Shutdown(context.Background())
func (s *PrepSubsystem) Shutdown(_ context.Context) error { return nil }

// --- Input/Output types ---

// PrepInput is the input for agentic_prep_workspace.
// One of Issue, PR, Branch, or Tag is required.
//
//	input := agentic.PrepInput{Repo: "go-io", Issue: 15, Task: "Migrate to Core primitives"}
type PrepInput struct {
	Repo         string            `json:"repo"`                    // required: e.g. "go-io"
	Org          string            `json:"org,omitempty"`           // default "core"
	Task         string            `json:"task,omitempty"`          // task description
	Agent        string            `json:"agent,omitempty"`         // agent type
	Issue        int               `json:"issue,omitempty"`         // Forge issue → workspace: task-{num}/
	PR           int               `json:"pr,omitempty"`            // PR number → workspace: pr-{num}/
	Branch       string            `json:"branch,omitempty"`        // branch → workspace: {branch}/
	Tag          string            `json:"tag,omitempty"`           // tag → workspace: {tag}/ (immutable)
	Template     string            `json:"template,omitempty"`      // prompt template slug
	PlanTemplate string            `json:"plan_template,omitempty"` // plan template slug
	Variables    map[string]string `json:"variables,omitempty"`     // template variable substitution
	Persona      string            `json:"persona,omitempty"`       // persona slug
	DryRun       bool              `json:"dry_run,omitempty"`       // preview without executing
}

// PrepOutput is the output for agentic_prep_workspace.
//
//	out := agentic.PrepOutput{Success: true, WorkspaceDir: ".core/workspace/core/go-io/task-15"}
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

// workspaceDir resolves the workspace path from the input identifier.
//
//	dir := workspaceDir("core", "go-io", PrepInput{Issue: 15})
//	// → ".core/workspace/core/go-io/task-15"
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

// workspaceDirResult resolves the workspace path and returns core.Result.
//
//	r := workspaceDirResult("core", "go-io", PrepInput{Issue: 15})
//	if r.OK { workspaceDir := r.Value.(string) }
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

	// Resolve workspace directory from identifier
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

	// Source repo path — org and repo were validated by workspaceDirResult.
	repoPath := core.JoinPath(s.codePath, input.Org, input.Repo)
	process := s.Core().Process()

	// Ensure meta directory exists
	if r := fs.EnsureDir(metaDir); !r.OK {
		return nil, PrepOutput{}, core.E("prep", "failed to create meta dir", nil)
	}

	// Check for resume: if repo/ already has .git, pull latest instead of re-cloning
	resumed := fs.IsDir(core.JoinPath(repoDir, ".git"))
	out.Resumed = resumed

	if resumed {
		// Preserve the current branch on resume. Pull it only if it exists on
		// origin; otherwise refresh the default branch refs without abandoning the
		// workspace branch.
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

	// Extract default workspace template (go.work etc.)
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
		// Clone repo into repo/
		if r := process.RunIn(ctx, ".", "git", "clone", repoPath, repoDir); !r.OK {
			return nil, PrepOutput{}, core.E("prep", core.Concat("git clone failed for ", input.Repo), nil)
		}

		// Create feature branch
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
		// Resume: read branch from existing checkout
		r := process.RunIn(ctx, repoDir, "git", "rev-parse", "--abbrev-ref", "HEAD")
		if r.OK {
			out.Branch = core.Trim(r.Value.(string))
		}
	}

	// Overwrite CODEX.md with language-specific version if needed.
	// The default template is Go-focused. PHP repos get CODEX-PHP.md instead.
	lang := detectLanguage(repoPath)
	if lang == "php" {
		if r := lib.WorkspaceFile("default", "CODEX-PHP.md.tmpl"); r.OK {
			codexPath := core.JoinPath(workspaceDir, "CODEX.md")
			fs.Write(codexPath, r.Value.(string))
		}
	}

	// Clone workspace dependencies — Core modules needed to build the repo.
	// Reads go.mod, finds dappco.re/go/core/* imports, clones from Forge,
	// and updates go.work so the agent can build inside the workspace.
	s.cloneWorkspaceDeps(ctx, workspaceDir, repoDir, input.Org)

	// Clone ecosystem docs into .core/reference/ so agents have full documentation.
	// The docs site (core.help) has architecture guides, specs, and API references.
	docsDir := core.JoinPath(workspaceDir, ".core", "reference", "docs")
	if !fs.IsDir(docsDir) {
		docsRepo := core.JoinPath(s.codePath, input.Org, "docs")
		if fs.IsDir(core.JoinPath(docsRepo, ".git")) {
			process.RunIn(ctx, ".", "git", "clone", "--depth", "1", docsRepo, docsDir)
		}
	}

	// Copy RFC specs from plans repo into workspace specs/ folder.
	// Maps repo name to plans directory: go-io → core/go/io/, go-process → core/go/process/, etc.
	s.copyRepoSpecs(workspaceDir, input.Repo)

	// Build the rich prompt with all context
	out.Prompt, out.Memories, out.Consumers = s.buildPrompt(ctx, input, out.Branch, repoPath)

	out.Success = true
	return nil, out, nil
}

// --- Spec Injection ---

// copyRepoSpecs copies RFC spec files from the plans repo into the workspace specs/ folder.
// Maps repo name to plans directory: go-io → core/go/io, agent → core/agent, core-bio → core/php/bio.
// Preserves subdirectory structure so sub-package specs land in specs/{pkg}/RFC.md.
//
//	s.copyRepoSpecs("/tmp/workspace", "go-io")   // copies plans/core/go/io/**/RFC*.md → /tmp/workspace/specs/
//	s.copyRepoSpecs("/tmp/workspace", "core-bio") // copies plans/core/php/bio/**/RFC*.md → /tmp/workspace/specs/
func (s *PrepSubsystem) copyRepoSpecs(workspaceDir, repo string) {
	fs := (&core.Fs{}).NewUnrestricted()

	// Plans repo base — look for it relative to codePath
	plansBase := core.JoinPath(s.codePath, "host-uk", "core", "plans")
	if !fs.IsDir(plansBase) {
		return
	}

	// Map repo name to plans directory
	var specDir string
	switch {
	case core.HasPrefix(repo, "go-"):
		// go-io → core/go/io, go-process → core/go/process
		pkg := core.TrimPrefix(repo, "go-")
		specDir = core.JoinPath(plansBase, "core", "go", pkg)
	case core.HasPrefix(repo, "core-"):
		// core-bio → core/php/bio, core-social → core/php/social
		mod := core.TrimPrefix(repo, "core-")
		specDir = core.JoinPath(plansBase, "core", "php", mod)
	case repo == "go":
		specDir = core.JoinPath(plansBase, "core", "go")
	default:
		// agent → core/agent, mcp → core/mcp, cli → core/go/cli, ide → core/ide
		specDir = core.JoinPath(plansBase, "core", repo)
	}

	if !fs.IsDir(specDir) {
		return
	}

	// Glob RFC*.md at each depth level (root, 1 deep, 2 deep, 3 deep).
	// Preserves subdirectory structure: specDir/pkg/nested/RFC.md → specs/pkg/nested/RFC.md
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

// --- Public API for CLI testing ---

// TestPrepWorkspace exposes prepWorkspace for CLI testing.
//
//	_, out, err := prep.TestPrepWorkspace(ctx, input)
func (s *PrepSubsystem) TestPrepWorkspace(ctx context.Context, input PrepInput) (*mcp.CallToolResult, PrepOutput, error) {
	return s.prepWorkspace(ctx, nil, input)
}

// TestBuildPrompt exposes buildPrompt for CLI testing.
//
//	prompt, memories, consumers := prep.TestBuildPrompt(ctx, input, "dev", repoPath)
func (s *PrepSubsystem) TestBuildPrompt(ctx context.Context, input PrepInput, branch, repoPath string) (string, int, int) {
	return s.buildPrompt(ctx, input, branch, repoPath)
}

// --- Prompt Building ---

// buildPrompt assembles all context into a single prompt string.
// Context is gathered from: persona, flow, issue, brain, consumers, git log, wiki, plan.
func (s *PrepSubsystem) buildPrompt(ctx context.Context, input PrepInput, branch, repoPath string) (string, int, int) {
	b := core.NewBuilder()
	memories := 0
	consumers := 0

	// Task
	b.WriteString("TASK: ")
	b.WriteString(input.Task)
	b.WriteString("\n\n")

	// Repo info
	b.WriteString(core.Sprintf("REPO: %s/%s on branch %s\n", input.Org, input.Repo, branch))
	b.WriteString(core.Sprintf("LANGUAGE: %s\n", detectLanguage(repoPath)))
	b.WriteString(core.Sprintf("BUILD: %s\n", detectBuildCmd(repoPath)))
	b.WriteString(core.Sprintf("TEST: %s\n\n", detectTestCmd(repoPath)))

	// Persona
	if input.Persona != "" {
		if r := lib.Persona(input.Persona); r.OK {
			b.WriteString("PERSONA:\n")
			b.WriteString(r.Value.(string))
			b.WriteString("\n\n")
		}
	}

	// Flow
	if r := lib.Flow(detectLanguage(repoPath)); r.OK {
		b.WriteString("WORKFLOW:\n")
		b.WriteString(r.Value.(string))
		b.WriteString("\n\n")
	}

	// Issue body
	if input.Issue > 0 {
		if body := s.getIssueBody(ctx, input.Org, input.Repo, input.Issue); body != "" {
			b.WriteString("ISSUE:\n")
			b.WriteString(body)
			b.WriteString("\n\n")
		}
	}

	// Brain recall
	if recall, count := s.brainRecall(ctx, input.Repo); recall != "" {
		b.WriteString("CONTEXT (from OpenBrain):\n")
		b.WriteString(recall)
		b.WriteString("\n\n")
		memories = count
	}

	// Consumers
	if list, count := s.findConsumersList(input.Repo); list != "" {
		b.WriteString("CONSUMERS (modules that import this repo):\n")
		b.WriteString(list)
		b.WriteString("\n\n")
		consumers = count
	}

	// Recent git log
	if log := s.getGitLog(repoPath); log != "" {
		b.WriteString("RECENT CHANGES:\n```\n")
		b.WriteString(log)
		b.WriteString("```\n\n")
	}

	// Plan template
	if input.PlanTemplate != "" {
		if plan := s.renderPlan(input.PlanTemplate, input.Variables, input.Task); plan != "" {
			b.WriteString("PLAN:\n")
			b.WriteString(plan)
			b.WriteString("\n\n")
		}
	}

	// Constraints
	b.WriteString("CONSTRAINTS:\n")
	b.WriteString("- Read CODEX.md for coding conventions (if it exists)\n")
	b.WriteString("- Read CLAUDE.md for project-specific instructions (if it exists)\n")
	b.WriteString("- Commit with conventional commit format: type(scope): description\n")
	b.WriteString("- Co-Authored-By: Virgil <virgil@lethean.io>\n")
	b.WriteString("- Run build and tests before committing\n")

	return b.String(), memories, consumers
}

// --- Context Helpers (return strings, not write files) ---

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
	r := lib.Template(templateSlug)
	if !r.OK {
		return ""
	}

	content := r.Value.(string)
	for key, value := range variables {
		content = core.Replace(content, core.Concat("{{", key, "}}"), value)
		content = core.Replace(content, core.Concat("{{ ", key, " }}"), value)
	}

	var tmpl struct {
		Name        string   `yaml:"name"`
		Description string   `yaml:"description"`
		Guidelines  []string `yaml:"guidelines"`
		Phases      []struct {
			Name        string `yaml:"name"`
			Description string `yaml:"description"`
			Tasks       []any  `yaml:"tasks"`
		} `yaml:"phases"`
	}

	if err := yaml.Unmarshal([]byte(content), &tmpl); err != nil {
		return ""
	}

	plan := core.NewBuilder()
	plan.WriteString(core.Concat("# ", tmpl.Name, "\n\n"))
	if task != "" {
		plan.WriteString(core.Concat("**Task:** ", task, "\n\n"))
	}
	if tmpl.Description != "" {
		plan.WriteString(core.Concat(tmpl.Description, "\n\n"))
	}

	if len(tmpl.Guidelines) > 0 {
		plan.WriteString("## Guidelines\n\n")
		for _, g := range tmpl.Guidelines {
			plan.WriteString(core.Concat("- ", g, "\n"))
		}
		plan.WriteString("\n")
	}

	for i, phase := range tmpl.Phases {
		plan.WriteString(core.Sprintf("## Phase %d: %s\n\n", i+1, phase.Name))
		if phase.Description != "" {
			plan.WriteString(core.Concat(phase.Description, "\n\n"))
		}
		for _, t := range phase.Tasks {
			switch v := t.(type) {
			case string:
				plan.WriteString(core.Concat("- [ ] ", v, "\n"))
			case map[string]any:
				if name, ok := v["name"].(string); ok {
					plan.WriteString(core.Concat("- [ ] ", name, "\n"))
				}
			}
		}
		plan.WriteString("\n")
	}

	return plan.String()
}

// --- Detection helpers (unchanged) ---

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
