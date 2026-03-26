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
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gopkg.in/yaml.v3"
)

// AgentOptions configures the agentic service.
type AgentOptions struct{}

// PrepSubsystem provides agentic MCP tools for workspace orchestration.
// Agent lifecycle events are broadcast via s.Core().ACTION(messages.AgentCompleted{}).
//
//	core.New(core.WithService(agentic.Register))
type PrepSubsystem struct {
	*core.ServiceRuntime[AgentOptions]
	forge      *forge.Forge
	forgeURL   string
	forgeToken string
	brainURL   string
	brainKey   string
	codePath   string
	drainMu    sync.Mutex
	pokeCh     chan struct{}
	frozen     bool
	backoff    map[string]time.Time // pool → paused until
	failCount  map[string]int       // pool → consecutive fast failures
}

var _ coremcp.Subsystem = (*PrepSubsystem)(nil)

// NewPrep creates an agentic subsystem.
//
//	sub := agentic.NewPrep()
//	sub.SetCompletionNotifier(monitor)
func NewPrep() *PrepSubsystem {
	home := core.Env("DIR_HOME")

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
//	c.Action("agentic.dispatch").Run(ctx, opts)
//	c.Actions() // ["agentic.dispatch", "agentic.prep", "agentic.status", ...]
func (s *PrepSubsystem) OnStartup(ctx context.Context) core.Result {
	c := s.Core()

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
	c.Action("agentic.dispatch", s.handleDispatch)
	c.Action("agentic.prep", s.handlePrep)
	c.Action("agentic.status", s.handleStatus)
	c.Action("agentic.resume", s.handleResume)
	c.Action("agentic.scan", s.handleScan)
	c.Action("agentic.watch", s.handleWatch)

	// Pipeline
	c.Action("agentic.qa", s.handleQA)
	c.Action("agentic.auto-pr", s.handleAutoPR)
	c.Action("agentic.verify", s.handleVerify)
	c.Action("agentic.ingest", s.handleIngest)
	c.Action("agentic.poke", s.handlePoke)
	c.Action("agentic.mirror", s.handleMirror)

	// Forge
	c.Action("agentic.issue.get", s.handleIssueGet)
	c.Action("agentic.issue.list", s.handleIssueList)
	c.Action("agentic.issue.create", s.handleIssueCreate)
	c.Action("agentic.pr.get", s.handlePRGet)
	c.Action("agentic.pr.list", s.handlePRList)
	c.Action("agentic.pr.merge", s.handlePRMerge)

	// Review
	c.Action("agentic.review-queue", s.handleReviewQueue)

	// Epic
	c.Action("agentic.epic", s.handleEpic)

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

	s.StartRunner()
	s.registerCommands(ctx)
	s.registerWorkspaceCommands()
	s.registerForgeCommands()
	return core.Result{OK: true}
}

// registerCommands is in commands.go

// OnShutdown implements core.Stoppable — freezes the queue.
func (s *PrepSubsystem) OnShutdown(ctx context.Context) core.Result {
	s.frozen = true
	return core.Result{OK: true}
}

func envOr(key, fallback string) string {
	if v := core.Env(key); v != "" {
		return v
	}
	return fallback
}

// Name implements mcp.Subsystem.
func (s *PrepSubsystem) Name() string { return "agentic" }

// RegisterTools implements mcp.Subsystem.
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

// Shutdown implements mcp.SubsystemWithShutdown.
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
	base := core.JoinPath(WorkspaceRoot(), org, repo)
	switch {
	case input.PR > 0:
		return core.JoinPath(base, core.Sprintf("pr-%d", input.PR)), nil
	case input.Issue > 0:
		return core.JoinPath(base, core.Sprintf("task-%d", input.Issue)), nil
	case input.Branch != "":
		return core.JoinPath(base, input.Branch), nil
	case input.Tag != "":
		return core.JoinPath(base, input.Tag), nil
	default:
		return "", core.E("workspaceDir", "one of issue, pr, branch, or tag is required", nil)
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
	wsDir, err := workspaceDir(input.Org, input.Repo, input)
	if err != nil {
		return nil, PrepOutput{}, err
	}

	repoDir := core.JoinPath(wsDir, "repo")
	metaDir := core.JoinPath(wsDir, ".meta")
	out := PrepOutput{WorkspaceDir: wsDir, RepoDir: repoDir}

	// Source repo path — sanitise to prevent path traversal
	repoName := core.PathBase(input.Repo)
	if repoName == "." || repoName == ".." || repoName == "" {
		return nil, PrepOutput{}, core.E("prep", "invalid repo name: "+input.Repo, nil)
	}
	repoPath := core.JoinPath(s.codePath, input.Org, repoName)

	// Ensure meta directory exists
	if r := fs.EnsureDir(metaDir); !r.OK {
		return nil, PrepOutput{}, core.E("prep", "failed to create meta dir", nil)
	}

	// Check for resume: if repo/ already has .git, skip clone
	resumed := fs.IsDir(core.JoinPath(repoDir, ".git"))
	out.Resumed = resumed

	// Extract default workspace template (go.work etc.)
	lib.ExtractWorkspace("default", wsDir, &lib.WorkspaceData{
		Repo:   input.Repo,
		Branch: "",
		Task:   input.Task,
		Agent:  input.Agent,
	})

	if !resumed {
		// Clone repo into repo/
		if r := s.gitCmd(ctx, ".", "clone", repoPath, repoDir); !r.OK {
			return nil, PrepOutput{}, core.E("prep", "git clone failed for "+input.Repo, nil)
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

		if r := s.gitCmd(ctx, repoDir, "checkout", "-b", branchName); !r.OK {
			return nil, PrepOutput{}, core.E("prep.branch", core.Sprintf("failed to create branch %q", branchName), nil)
		}
		out.Branch = branchName
	} else {
		// Resume: read branch from existing checkout
		out.Branch = s.gitOutput(ctx, repoDir, "rev-parse", "--abbrev-ref", "HEAD")
	}

	// Build the rich prompt with all context
	out.Prompt, out.Memories, out.Consumers = s.buildPrompt(ctx, input, out.Branch, repoPath)

	out.Success = true
	return nil, out, nil
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
		if core.Contains(modData, modulePath) && !core.HasPrefix(modData, "module "+modulePath) {
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
	return s.gitOutput(context.Background(), repoPath, "log", "--oneline", "-20")
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
		page, pErr := s.forge.Wiki.GetPage(ctx, org, repo, name)
		if pErr != nil || page.ContentBase64 == "" {
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
		content = core.Replace(content, "{{"+key+"}}", value)
		content = core.Replace(content, "{{ "+key+" }}", value)
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
