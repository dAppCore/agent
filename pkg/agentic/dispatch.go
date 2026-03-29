// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"time"

	"dappco.re/go/agent/pkg/messages"
	core "dappco.re/go/core"
	"dappco.re/go/core/process"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// workspaceTracker is the interface runner.Service satisfies.
// Uses *WorkspaceStatus from agentic — runner imports agentic for the type.
type workspaceTracker interface {
	TrackWorkspace(name string, st any)
}

// DispatchInput is the input for agentic_dispatch.
//
//	input := agentic.DispatchInput{Repo: "go-io", Task: "Fix the failing tests", Agent: "codex", Issue: 15}
type DispatchInput struct {
	Repo         string            `json:"repo"`                    // Target repo (e.g. "go-io")
	Org          string            `json:"org,omitempty"`           // Forge org (default "core")
	Task         string            `json:"task"`                    // What the agent should do
	Agent        string            `json:"agent,omitempty"`         // "codex" (default), "claude", "gemini"
	Template     string            `json:"template,omitempty"`      // "conventions", "security", "coding" (default)
	PlanTemplate string            `json:"plan_template,omitempty"` // Plan template slug
	Variables    map[string]string `json:"variables,omitempty"`     // Template variable substitution
	Persona      string            `json:"persona,omitempty"`       // Persona slug
	Issue        int               `json:"issue,omitempty"`         // Forge issue number → workspace: task-{num}/
	PR           int               `json:"pr,omitempty"`            // PR number → workspace: pr-{num}/
	Branch       string            `json:"branch,omitempty"`        // Branch → workspace: {branch}/
	Tag          string            `json:"tag,omitempty"`           // Tag → workspace: {tag}/ (immutable)
	DryRun       bool              `json:"dry_run,omitempty"`       // Preview without executing
}

// DispatchOutput is the output for agentic_dispatch.
//
//	out := agentic.DispatchOutput{Success: true, Agent: "codex", Repo: "go-io", WorkspaceDir: ".core/workspace/core/go-io/task-15"}
type DispatchOutput struct {
	Success      bool   `json:"success"`
	Agent        string `json:"agent"`
	Repo         string `json:"repo"`
	WorkspaceDir string `json:"workspace_dir"`
	Prompt       string `json:"prompt,omitempty"`
	PID          int    `json:"pid,omitempty"`
	OutputFile   string `json:"output_file,omitempty"`
}

func (s *PrepSubsystem) registerDispatchTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_dispatch",
		Description: "Dispatch a subagent (Gemini, Codex, or Claude) to work on a task. Preps a sandboxed workspace first, then spawns the agent inside it. Templates: conventions, security, coding.",
	}, s.dispatch)
}

// agentCommand returns the command and args for a given agent type.
// Supports model variants: "gemini", "gemini:flash", "codex", "claude", "claude:haiku".
func agentCommand(agent, prompt string) (string, []string, error) {
	parts := core.SplitN(agent, ":", 2)
	base := parts[0]
	model := ""
	if len(parts) > 1 {
		model = parts[1]
	}

	switch base {
	case "gemini":
		args := []string{"-p", prompt, "--yolo", "--sandbox"}
		if model != "" {
			args = append(args, "-m", core.Concat("gemini-2.5-", model))
		}
		return "gemini", args, nil
	case "codex":
		if model == "review" {
			// Use exec with bypass — codex review subcommand has its own sandbox that blocks shell
			// No -o flag — stdout captured by process output, ../.meta path unreliable in sandbox
			return "codex", []string{
				"exec",
				"--dangerously-bypass-approvals-and-sandbox",
				"Review the last 2 commits via git diff HEAD~2. Check for bugs, security issues, missing tests, naming issues. Report pass/fail with specifics. Do NOT make changes.",
			}, nil
		}
		// Container IS the sandbox — let codex run unrestricted inside it
		args := []string{
			"exec",
			"--dangerously-bypass-approvals-and-sandbox",
			"-o", "../.meta/agent-codex.log",
		}
		if model != "" {
			args = append(args, "--model", model)
		}
		args = append(args, prompt)
		return "codex", args, nil
	case "claude":
		args := []string{
			"-p", prompt,
			"--output-format", "text",
			"--dangerously-skip-permissions",
			"--no-session-persistence",
			"--append-system-prompt", "SANDBOX: You are restricted to the current directory only. Do NOT use absolute paths. Do NOT navigate outside this repository.",
		}
		if model != "" {
			args = append(args, "--model", model)
		}
		return "claude", args, nil
	case "coderabbit":
		args := []string{"review", "--plain", "--base", "HEAD~1"}
		if model != "" {
			args = append(args, "--type", model)
		}
		if prompt != "" {
			args = append(args, "--config", "CLAUDE.md")
		}
		return "coderabbit", args, nil
	case "local":
		// Local model via codex --oss → Ollama. Default model: devstral-24b
		// socat proxies localhost:11434 → host.docker.internal:11434
		// because codex hardcodes localhost check for Ollama.
		localModel := model
		if localModel == "" {
			localModel = "devstral-24b"
		}
		script := core.Sprintf(
			`socat TCP-LISTEN:11434,fork,reuseaddr TCP:host.docker.internal:11434 & sleep 0.5 && codex exec --dangerously-bypass-approvals-and-sandbox --oss --local-provider ollama -m %s -o ../.meta/agent-codex.log %q`,
			localModel, prompt,
		)
		return "sh", []string{"-c", script}, nil
	default:
		return "", nil, core.E("agentCommand", core.Concat("unknown agent: ", agent), nil)
	}
}

// defaultDockerImage is the container image for agent dispatch.
// Override via AGENT_DOCKER_IMAGE env var.
const defaultDockerImage = "core-dev"

// containerCommand wraps an agent command to run inside a Docker container.
// All agents run containerised — no bare metal execution.
// agentType is the base agent name (e.g. "local", "codex", "claude").
//
//	cmd, args := containerCommand("local", "codex", []string{"exec", "..."}, repoDir, metaDir)
func containerCommand(agentType, command string, args []string, repoDir, metaDir string) (string, []string) {
	image := core.Env("AGENT_DOCKER_IMAGE")
	if image == "" {
		image = defaultDockerImage
	}

	home := core.Env("DIR_HOME")

	dockerArgs := []string{
		"run", "--rm",
		// Host access for Ollama (local models)
		"--add-host=host.docker.internal:host-gateway",
		// Workspace: repo + meta
		"-v", core.Concat(repoDir, ":/workspace"),
		"-v", core.Concat(metaDir, ":/workspace/.meta"),
		"-w", "/workspace",
		// Auth: agent configs only — NO SSH keys, git push runs on host
		"-v", core.Concat(core.JoinPath(home, ".codex"), ":/home/dev/.codex:ro"),
		// API keys — passed by name, Docker resolves from host env
		"-e", "OPENAI_API_KEY",
		"-e", "ANTHROPIC_API_KEY",
		"-e", "GEMINI_API_KEY",
		"-e", "GOOGLE_API_KEY",
		// Agent environment
		"-e", "TERM=dumb",
		"-e", "NO_COLOR=1",
		"-e", "CI=true",
		"-e", "GIT_USER_NAME=Virgil",
		"-e", "GIT_USER_EMAIL=virgil@lethean.io",
		// Go workspace — local modules bypass checksum verification
		"-e", "GONOSUMCHECK=dappco.re/*,forge.lthn.ai/*",
		"-e", "GOFLAGS=-mod=mod",
	}

	// Mount Claude config if dispatching claude agent
	if command == "claude" {
		dockerArgs = append(dockerArgs,
			"-v", core.Concat(core.JoinPath(home, ".claude"), ":/home/dev/.claude:ro"),
		)
	}

	// Mount Gemini config if dispatching gemini agent
	if command == "gemini" {
		dockerArgs = append(dockerArgs,
			"-v", core.Concat(core.JoinPath(home, ".gemini"), ":/home/dev/.gemini:ro"),
		)
	}

	// Wrap agent command in sh -c to chmod workspace after exit.
	// Docker runs as a different user — without this, host can't delete workspace files.
	quoted := core.NewBuilder()
	quoted.WriteString(command)
	for _, a := range args {
		quoted.WriteString(" '")
		quoted.WriteString(core.Replace(a, "'", "'\\''"))
		quoted.WriteString("'")
	}
	quoted.WriteString("; chmod -R a+w /workspace /workspace/.meta 2>/dev/null; true")

	dockerArgs = append(dockerArgs, image, "sh", "-c", quoted.String())

	return "docker", dockerArgs
}

// --- spawnAgent: decomposed into testable steps ---

// agentOutputFile returns the log file path for an agent's output.
func agentOutputFile(wsDir, agent string) string {
	agentBase := core.SplitN(agent, ":", 2)[0]
	return core.JoinPath(wsDir, ".meta", core.Sprintf("agent-%s.log", agentBase))
}

// detectFinalStatus reads workspace state after agent exit to determine outcome.
// Returns (status, question) — "completed", "blocked", or "failed".
func detectFinalStatus(repoDir string, exitCode int, procStatus string) (string, string) {
	blockedPath := core.JoinPath(repoDir, "BLOCKED.md")
	if r := fs.Read(blockedPath); r.OK && core.Trim(r.Value.(string)) != "" {
		return "blocked", core.Trim(r.Value.(string))
	}
	if exitCode != 0 || procStatus == "failed" || procStatus == "killed" {
		question := ""
		if exitCode != 0 {
			question = core.Sprintf("Agent exited with code %d", exitCode)
		}
		return "failed", question
	}
	return "completed", ""
}

// trackFailureRate detects fast consecutive failures and applies backoff.
// Returns true if backoff was triggered.
func (s *PrepSubsystem) trackFailureRate(agent, status string, startedAt time.Time) bool {
	pool := baseAgent(agent)
	if status == "failed" {
		elapsed := time.Since(startedAt)
		if elapsed < 60*time.Second {
			s.failCount[pool]++
			if s.failCount[pool] >= 3 {
				s.backoff[pool] = time.Now().Add(30 * time.Minute)
				core.Print(nil, "rate-limit detected for %s — pausing pool for 30 minutes", pool)
				return true
			}
		} else {
			s.failCount[pool] = 0 // slow failure = real failure, reset count
		}
	} else {
		s.failCount[pool] = 0 // success resets count
	}
	return false
}

// startIssueTracking starts a Forge stopwatch on the workspace's issue.
func (s *PrepSubsystem) startIssueTracking(wsDir string) {
	if s.forge == nil {
		return
	}
	st, _ := ReadStatus(wsDir)
	if st == nil || st.Issue == 0 {
		return
	}
	org := st.Org
	if org == "" {
		org = "core"
	}
	s.forge.Issues.StartStopwatch(context.Background(), org, st.Repo, int64(st.Issue))
}

// stopIssueTracking stops a Forge stopwatch on the workspace's issue.
func (s *PrepSubsystem) stopIssueTracking(wsDir string) {
	if s.forge == nil {
		return
	}
	st, _ := ReadStatus(wsDir)
	if st == nil || st.Issue == 0 {
		return
	}
	org := st.Org
	if org == "" {
		org = "core"
	}
	s.forge.Issues.StopStopwatch(context.Background(), org, st.Repo, int64(st.Issue))
}

// broadcastStart emits IPC + audit events for agent start.
func (s *PrepSubsystem) broadcastStart(agent, wsDir string) {
	st, _ := ReadStatus(wsDir)
	repo := ""
	if st != nil {
		repo = st.Repo
	}
	if s.ServiceRuntime != nil {
		s.Core().ACTION(messages.AgentStarted{
			Agent: agent, Repo: repo, Workspace: core.PathBase(wsDir),
		})
	}
	emitStartEvent(agent, core.PathBase(wsDir))
}

// broadcastComplete emits IPC + audit events for agent completion.
func (s *PrepSubsystem) broadcastComplete(agent, wsDir, finalStatus string) {
	emitCompletionEvent(agent, core.PathBase(wsDir), finalStatus)
	if s.ServiceRuntime != nil {
		st, _ := ReadStatus(wsDir)
		repo := ""
		if st != nil {
			repo = st.Repo
		}
		s.Core().ACTION(messages.AgentCompleted{
			Agent: agent, Repo: repo,
			Workspace: core.PathBase(wsDir), Status: finalStatus,
		})
	}
}

// onAgentComplete handles all post-completion logic for a spawned agent.
// Called from the monitoring goroutine after the process exits.
func (s *PrepSubsystem) onAgentComplete(agent, wsDir, outputFile string, exitCode int, procStatus, output string) {
	// Save output
	if output != "" {
		fs.Write(outputFile, output)
	}

	repoDir := core.JoinPath(wsDir, "repo")
	finalStatus, question := detectFinalStatus(repoDir, exitCode, procStatus)

	// Update workspace status (disk + registry)
	if st, err := ReadStatus(wsDir); err == nil {
		st.Status = finalStatus
		st.PID = 0
		st.Question = question
		writeStatus(wsDir, st)
		s.TrackWorkspace(WorkspaceName(wsDir), st)
	}

	// Rate-limit tracking
	if st, _ := ReadStatus(wsDir); st != nil {
		s.trackFailureRate(agent, finalStatus, st.StartedAt)
	}

	// Forge time tracking
	s.stopIssueTracking(wsDir)

	// Broadcast completion
	s.broadcastComplete(agent, wsDir, finalStatus)

	// Run completion pipeline via PerformAsync for successful agents.
	// Gets ActionTaskStarted/Completed broadcasts + WaitGroup integration for graceful shutdown.
	//
	//   c.PerformAsync("agentic.complete", opts) → runs agent.completion Task in background
	if finalStatus == "completed" && s.ServiceRuntime != nil {
		s.Core().PerformAsync("agentic.complete", core.NewOptions(
			core.Option{Key: "workspace", Value: wsDir},
		))
	}
}

// spawnAgent launches an agent inside a Docker container.
// The repo/ directory is mounted at /workspace, agent runs sandboxed.
// Output is captured and written to .meta/agent-{agent}.log on completion.
func (s *PrepSubsystem) spawnAgent(agent, prompt, wsDir string) (int, string, error) {
	command, args, err := agentCommand(agent, prompt)
	if err != nil {
		return 0, "", err
	}

	repoDir := core.JoinPath(wsDir, "repo")
	metaDir := core.JoinPath(wsDir, ".meta")
	outputFile := agentOutputFile(wsDir, agent)

	// Clean up stale BLOCKED.md from previous runs
	fs.Delete(core.JoinPath(repoDir, "BLOCKED.md"))

	// All agents run containerised
	agentBase := core.SplitN(agent, ":", 2)[0]
	command, args = containerCommand(agentBase, command, args, repoDir, metaDir)

	procSvc, ok := core.ServiceFor[*process.Service](s.Core(), "process")
	if !ok {
		return 0, "", core.E("dispatch.spawnAgent", "process service not registered", nil)
	}
	sr := procSvc.StartWithOptions(context.Background(), process.RunOptions{
		Command: command,
		Args:    args,
		Dir:     repoDir,
		Detach:  true,
	})
	if !sr.OK {
		return 0, "", core.E("dispatch.spawnAgent", core.Concat("failed to spawn ", agent), nil)
	}
	proc := sr.Value.(*process.Process)

	proc.CloseStdin()
	pid := proc.Info().PID

	s.broadcastStart(agent, wsDir)
	s.startIssueTracking(wsDir)

	// Register a one-shot Action that monitors this agent, then run it via PerformAsync.
	// PerformAsync tracks it in Core's WaitGroup — ServiceShutdown waits for it.
	monitorAction := core.Concat("agentic.monitor.", core.PathBase(wsDir))
	s.Core().Action(monitorAction, func(_ context.Context, _ core.Options) core.Result {
		<-proc.Done()
		s.onAgentComplete(agent, wsDir, outputFile,
			proc.Info().ExitCode, string(proc.Info().Status), proc.Output())
		return core.Result{OK: true}
	})
	s.Core().PerformAsync(monitorAction, core.NewOptions())

	return pid, outputFile, nil
}

// runQA runs build + test checks on the repo after agent completion.
// Returns true if QA passes, false if build or tests fail.
func (s *PrepSubsystem) runQA(wsDir string) bool {
	ctx := context.Background()
	repoDir := core.JoinPath(wsDir, "repo")

	if fs.IsFile(core.JoinPath(repoDir, "go.mod")) {
		for _, args := range [][]string{
			{"go", "build", "./..."},
			{"go", "vet", "./..."},
			{"go", "test", "./...", "-count=1", "-timeout", "120s"},
		} {
			if !s.runCmdOK(ctx, repoDir, args[0], args[1:]...) {
				core.Warn("QA failed", "cmd", core.Join(" ", args...))
				return false
			}
		}
		return true
	}

	if fs.IsFile(core.JoinPath(repoDir, "composer.json")) {
		if !s.runCmdOK(ctx, repoDir, "composer", "install", "--no-interaction") {
			return false
		}
		return s.runCmdOK(ctx, repoDir, "composer", "test")
	}

	if fs.IsFile(core.JoinPath(repoDir, "package.json")) {
		if !s.runCmdOK(ctx, repoDir, "npm", "install") {
			return false
		}
		return s.runCmdOK(ctx, repoDir, "npm", "test")
	}

	return true
}

func (s *PrepSubsystem) dispatch(ctx context.Context, req *mcp.CallToolRequest, input DispatchInput) (*mcp.CallToolResult, DispatchOutput, error) {
	if input.Repo == "" {
		return nil, DispatchOutput{}, core.E("dispatch", "repo is required", nil)
	}
	if input.Task == "" {
		return nil, DispatchOutput{}, core.E("dispatch", "task is required", nil)
	}
	if input.Org == "" {
		input.Org = "core"
	}
	if input.Agent == "" {
		input.Agent = "codex"
	}
	if input.Template == "" {
		input.Template = "coding"
	}

	// Concurrency check — ask the runner
	r := s.Core().Action("runner.dispatch").Run(ctx, core.NewOptions(
		core.Option{Key: "agent", Value: input.Agent},
		core.Option{Key: "repo", Value: input.Repo},
	))
	if !r.OK {
		reason, _ := r.Value.(string)
		out := DispatchOutput{
			Repo:       input.Repo,
			Success:    true,
			OutputFile: core.Concat("queued — ", reason),
		}
		return nil, out, nil
	}

	// Step 1: Prep workspace — clone + build prompt
	prepInput := PrepInput{
		Repo:         input.Repo,
		Org:          input.Org,
		Issue:        input.Issue,
		PR:           input.PR,
		Branch:       input.Branch,
		Tag:          input.Tag,
		Task:         input.Task,
		Agent:        input.Agent,
		Template:     input.Template,
		PlanTemplate: input.PlanTemplate,
		Variables:    input.Variables,
		Persona:      input.Persona,
	}
	_, prepOut, err := s.prepWorkspace(ctx, req, prepInput)
	if err != nil {
		return nil, DispatchOutput{}, core.E("dispatch", "prep workspace failed", err)
	}

	wsDir := prepOut.WorkspaceDir
	prompt := prepOut.Prompt

	if input.DryRun {
		return nil, DispatchOutput{
			Success:      true,
			Agent:        input.Agent,
			Repo:         input.Repo,
			WorkspaceDir: wsDir,
			Prompt:       prompt,
		}, nil
	}

	// Step 2: Ask runner service for permission (frozen + concurrency check).
	// Runner owns the gate — agentic owns the spawn.
	if s.ServiceRuntime != nil {
		r := s.Core().Action("runner.dispatch").Run(ctx, core.NewOptions(
			core.Option{Key: "agent", Value: input.Agent},
			core.Option{Key: "repo", Value: input.Repo},
		))
		if !r.OK {
			// Runner denied — queue it
			st := &WorkspaceStatus{
				Status:    "queued",
				Agent:     input.Agent,
				Repo:      input.Repo,
				Org:       input.Org,
				Task:      input.Task,
				Branch:    prepOut.Branch,
				StartedAt: time.Now(),
				Runs:      0,
			}
			writeStatus(wsDir, st)
			if runnerSvc, ok := core.ServiceFor[workspaceTracker](s.Core(), "runner"); ok {
				runnerSvc.TrackWorkspace(WorkspaceName(wsDir), st)
			}
			return nil, DispatchOutput{
				Success:      true,
				Agent:        input.Agent,
				Repo:         input.Repo,
				WorkspaceDir: wsDir,
				OutputFile:   "queued — at concurrency limit or frozen",
			}, nil
		}
	}

	// Step 3: Spawn agent in repo/ directory
	pid, outputFile, err := s.spawnAgent(input.Agent, prompt, wsDir)
	if err != nil {
		return nil, DispatchOutput{}, err
	}

	st := &WorkspaceStatus{
		Status:    "running",
		Agent:     input.Agent,
		Repo:      input.Repo,
		Org:       input.Org,
		Task:      input.Task,
		Branch:    prepOut.Branch,
		PID:       pid,
		StartedAt: time.Now(),
		Runs:      1,
	}
	writeStatus(wsDir, st)
	// Track in runner's registry (runner owns workspace state)
	if s.ServiceRuntime != nil {
		if runnerSvc, ok := core.ServiceFor[workspaceTracker](s.Core(), "runner"); ok {
			runnerSvc.TrackWorkspace(WorkspaceName(wsDir), st)
		}
	}

	return nil, DispatchOutput{
		Success:      true,
		Agent:        input.Agent,
		Repo:         input.Repo,
		WorkspaceDir: wsDir,
		PID:          pid,
		OutputFile:   outputFile,
	}, nil
}
