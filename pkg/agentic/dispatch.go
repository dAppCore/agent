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

type workspaceTracker interface {
	TrackWorkspace(name string, status any)
}

// input := agentic.DispatchInput{Repo: "go-io", Task: "Fix the failing tests", Agent: "codex", Issue: 15}
type DispatchInput struct {
	Repo         string            `json:"repo"`
	Org          string            `json:"org,omitempty"`
	Task         string            `json:"task"`
	Agent        string            `json:"agent,omitempty"`
	Template     string            `json:"template,omitempty"`
	PlanTemplate string            `json:"plan_template,omitempty"`
	Variables    map[string]string `json:"variables,omitempty"`
	Persona      string            `json:"persona,omitempty"`
	Issue        int               `json:"issue,omitempty"`
	PR           int               `json:"pr,omitempty"`
	Branch       string            `json:"branch,omitempty"`
	Tag          string            `json:"tag,omitempty"`
	DryRun       bool              `json:"dry_run,omitempty"`
}

// out := agentic.DispatchOutput{Success: true, Agent: "codex", Repo: "go-io", WorkspaceDir: ".core/workspace/core/go-io/task-15"}
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

// command, args, err := agentCommand("codex:review", "Review the last 2 commits via git diff HEAD~2")
func agentCommand(agent, prompt string) (string, []string, error) {
	commandResult := agentCommandResult(agent, prompt)
	if !commandResult.OK {
		err, _ := commandResult.Value.(error)
		if err == nil {
			err = core.E("agentCommand", "failed to resolve command", nil)
		}
		return "", nil, err
	}

	result, ok := commandResult.Value.(agentCommandResultValue)
	if !ok {
		return "", nil, core.E("agentCommand", "invalid command result", nil)
	}

	return result.command, result.args, nil
}

type agentCommandResultValue struct {
	command string
	args    []string
}

func agentCommandResult(agent, prompt string) core.Result {
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
		return core.Result{Value: agentCommandResultValue{command: "gemini", args: args}, OK: true}
	case "codex":
		if model == "review" {
			return core.Result{Value: agentCommandResultValue{command: "codex", args: []string{
				"exec",
				"--dangerously-bypass-approvals-and-sandbox",
				"Review the last 2 commits via git diff HEAD~2. Check for bugs, security issues, missing tests, naming issues. Report pass/fail with specifics. Do NOT make changes.",
			}}, OK: true}
		}
		args := []string{
			"exec",
			"--dangerously-bypass-approvals-and-sandbox",
			"-o", "../.meta/agent-codex.log",
		}
		if model != "" {
			args = append(args, "--model", model)
		}
		args = append(args, prompt)
		return core.Result{Value: agentCommandResultValue{command: "codex", args: args}, OK: true}
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
		return core.Result{Value: agentCommandResultValue{command: "claude", args: args}, OK: true}
	case "coderabbit":
		args := []string{"review", "--plain", "--base", "HEAD~1"}
		if model != "" {
			args = append(args, "--type", model)
		}
		if prompt != "" {
			args = append(args, "--config", "CLAUDE.md")
		}
		return core.Result{Value: agentCommandResultValue{command: "coderabbit", args: args}, OK: true}
	case "local":
		localModel := model
		if localModel == "" {
			localModel = "devstral-24b"
		}
		script := core.Sprintf(
			`socat TCP-LISTEN:11434,fork,reuseaddr TCP:host.docker.internal:11434 & sleep 0.5 && codex exec --dangerously-bypass-approvals-and-sandbox --oss --local-provider ollama -m %s -o ../.meta/agent-codex.log %q`,
			localModel, prompt,
		)
		return core.Result{Value: agentCommandResultValue{command: "sh", args: []string{"-c", script}}, OK: true}
	default:
		return core.Result{Value: core.E("agentCommand", core.Concat("unknown agent: ", agent), nil), OK: false}
	}
}

const defaultDockerImage = "core-dev"

// command, args := containerCommand("local", "codex", []string{"exec", "--oss"}, "/srv/.core/workspace/core/go-io/task-5/repo", "/srv/.core/workspace/core/go-io/task-5/.meta")
func containerCommand(command string, args []string, repoDir, metaDir string) (string, []string) {
	image := core.Env("AGENT_DOCKER_IMAGE")
	if image == "" {
		image = defaultDockerImage
	}

	home := HomeDir()

	dockerArgs := []string{
		"run", "--rm",
		"--add-host=host.docker.internal:host-gateway",
		"-v", core.Concat(repoDir, ":/workspace"),
		"-v", core.Concat(metaDir, ":/workspace/.meta"),
		"-w", "/workspace",
		"-v", core.Concat(core.JoinPath(home, ".codex"), ":/home/dev/.codex:ro"),
		"-e", "OPENAI_API_KEY",
		"-e", "ANTHROPIC_API_KEY",
		"-e", "GEMINI_API_KEY",
		"-e", "GOOGLE_API_KEY",
		"-e", "TERM=dumb",
		"-e", "NO_COLOR=1",
		"-e", "CI=true",
		"-e", "GIT_USER_NAME=Virgil",
		"-e", "GIT_USER_EMAIL=virgil@lethean.io",
		"-e", "GONOSUMCHECK=dappco.re/*,forge.lthn.ai/*",
		"-e", "GOFLAGS=-mod=mod",
	}

	if command == "claude" {
		dockerArgs = append(dockerArgs,
			"-v", core.Concat(core.JoinPath(home, ".claude"), ":/home/dev/.claude:ro"),
		)
	}

	if command == "gemini" {
		dockerArgs = append(dockerArgs,
			"-v", core.Concat(core.JoinPath(home, ".gemini"), ":/home/dev/.gemini:ro"),
		)
	}

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

// outputFile := agentOutputFile(workspaceDir, "codex")
func agentOutputFile(workspaceDir, agent string) string {
	agentBase := core.SplitN(agent, ":", 2)[0]
	return core.JoinPath(WorkspaceMetaDir(workspaceDir), core.Sprintf("agent-%s.log", agentBase))
}

// status, question := detectFinalStatus(repoDir, 0, "completed")
func detectFinalStatus(repoDir string, exitCode int, processStatus string) (string, string) {
	blockedPath := core.JoinPath(repoDir, "BLOCKED.md")
	if blockedResult := fs.Read(blockedPath); blockedResult.OK && core.Trim(blockedResult.Value.(string)) != "" {
		return "blocked", core.Trim(blockedResult.Value.(string))
	}
	if exitCode != 0 || processStatus == "failed" || processStatus == "killed" {
		question := ""
		if exitCode != 0 {
			question = core.Sprintf("Agent exited with code %d", exitCode)
		}
		return "failed", question
	}
	return "completed", ""
}

// backoff := s.trackFailureRate("codex", "failed", time.Now().Add(-30*time.Second))
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
			s.failCount[pool] = 0
		}
	} else {
		s.failCount[pool] = 0
	}
	return false
}

func (s *PrepSubsystem) startIssueTracking(workspaceDir string) {
	if s.forge == nil {
		return
	}
	result := ReadStatusResult(workspaceDir)
	workspaceStatus, ok := workspaceStatusValue(result)
	if !ok || workspaceStatus.Issue == 0 {
		return
	}
	org := workspaceStatus.Org
	if org == "" {
		org = "core"
	}
	s.forge.Issues.StartStopwatch(context.Background(), org, workspaceStatus.Repo, int64(workspaceStatus.Issue))
}

func (s *PrepSubsystem) stopIssueTracking(workspaceDir string) {
	if s.forge == nil {
		return
	}
	result := ReadStatusResult(workspaceDir)
	workspaceStatus, ok := workspaceStatusValue(result)
	if !ok || workspaceStatus.Issue == 0 {
		return
	}
	org := workspaceStatus.Org
	if org == "" {
		org = "core"
	}
	s.forge.Issues.StopStopwatch(context.Background(), org, workspaceStatus.Repo, int64(workspaceStatus.Issue))
}

func (s *PrepSubsystem) broadcastStart(agent, workspaceDir string) {
	workspaceName := WorkspaceName(workspaceDir)
	result := ReadStatusResult(workspaceDir)
	workspaceStatus, ok := workspaceStatusValue(result)
	repo := ""
	if ok {
		repo = workspaceStatus.Repo
	}
	if s.ServiceRuntime != nil {
		s.Core().ACTION(messages.AgentStarted{
			Agent: agent, Repo: repo, Workspace: workspaceName,
		})
	}
	emitStartEvent(agent, workspaceName)
}

func (s *PrepSubsystem) broadcastComplete(agent, workspaceDir, finalStatus string) {
	workspaceName := WorkspaceName(workspaceDir)
	emitCompletionEvent(agent, workspaceName, finalStatus)
	if s.ServiceRuntime != nil {
		result := ReadStatusResult(workspaceDir)
		workspaceStatus, ok := workspaceStatusValue(result)
		repo := ""
		if ok {
			repo = workspaceStatus.Repo
		}
		s.Core().ACTION(messages.AgentCompleted{
			Agent: agent, Repo: repo,
			Workspace: workspaceName, Status: finalStatus,
		})
	}
}

func (s *PrepSubsystem) onAgentComplete(agent, workspaceDir, outputFile string, exitCode int, processStatus, output string) {
	if output != "" {
		fs.Write(outputFile, output)
	}

	repoDir := WorkspaceRepoDir(workspaceDir)
	finalStatus, question := detectFinalStatus(repoDir, exitCode, processStatus)

	result := ReadStatusResult(workspaceDir)
	workspaceStatus, ok := workspaceStatusValue(result)
	if ok {
		workspaceStatus.Status = finalStatus
		workspaceStatus.PID = 0
		workspaceStatus.Question = question
		writeStatusResult(workspaceDir, workspaceStatus)
		s.TrackWorkspace(WorkspaceName(workspaceDir), workspaceStatus)

		s.trackFailureRate(agent, finalStatus, workspaceStatus.StartedAt)
	}

	s.stopIssueTracking(workspaceDir)

	s.broadcastComplete(agent, workspaceDir, finalStatus)

	if finalStatus == "completed" && s.ServiceRuntime != nil {
		s.Core().PerformAsync("agentic.complete", core.NewOptions(
			core.Option{Key: "workspace", Value: workspaceDir},
		))
	}
}

// pid, processID, outputFile, err := s.spawnAgent(agent, prompt, workspaceDir)
func (s *PrepSubsystem) spawnAgent(agent, prompt, workspaceDir string) (int, string, string, error) {
	command, args, err := agentCommand(agent, prompt)
	if err != nil {
		return 0, "", "", err
	}

	repoDir := WorkspaceRepoDir(workspaceDir)
	metaDir := WorkspaceMetaDir(workspaceDir)
	outputFile := agentOutputFile(workspaceDir, agent)

	fs.Delete(WorkspaceBlockedPath(workspaceDir))

	command, args = containerCommand(command, args, repoDir, metaDir)

	procSvc, ok := core.ServiceFor[*process.Service](s.Core(), "process")
	if !ok {
		return 0, "", "", core.E("dispatch.spawnAgent", "process service not registered", nil)
	}
	proc, err := procSvc.StartWithOptions(context.Background(), process.RunOptions{
		Command: command,
		Args:    args,
		Dir:     repoDir,
		Detach:  true,
	})
	if err != nil {
		return 0, "", "", core.E("dispatch.spawnAgent", core.Concat("failed to spawn ", agent), err)
	}

	proc.CloseStdin()
	pid := proc.Info().PID
	processID := proc.ID

	s.broadcastStart(agent, workspaceDir)
	s.startIssueTracking(workspaceDir)

	monitorAction := core.Concat("agentic.monitor.", core.Replace(WorkspaceName(workspaceDir), "/", "."))
	monitor := &agentCompletionMonitor{
		service:      s,
		agent:        agent,
		workspaceDir: workspaceDir,
		outputFile:   outputFile,
		process:      proc,
	}
	s.Core().Action(monitorAction, monitor.run)
	s.Core().PerformAsync(monitorAction, core.NewOptions())

	return pid, processID, outputFile, nil
}

type completionProcess interface {
	Done() <-chan struct{}
	Info() process.Info
	Output() string
}

type agentCompletionMonitor struct {
	service      *PrepSubsystem
	agent        string
	workspaceDir string
	outputFile   string
	process      completionProcess
}

func (m *agentCompletionMonitor) run(_ context.Context, _ core.Options) core.Result {
	if m == nil || m.service == nil {
		return core.Result{Value: core.E("agentic.monitor", "service is required", nil), OK: false}
	}
	if m.process == nil {
		return core.Result{Value: core.E("agentic.monitor", "process is required", nil), OK: false}
	}

	<-m.process.Done()
	info := m.process.Info()
	m.service.onAgentComplete(m.agent, m.workspaceDir, m.outputFile, info.ExitCode, string(info.Status), m.process.Output())
	return core.Result{OK: true}
}

func (s *PrepSubsystem) runQA(workspaceDir string) bool {
	ctx := context.Background()
	repoDir := WorkspaceRepoDir(workspaceDir)
	process := s.Core().Process()

	if fs.IsFile(core.JoinPath(repoDir, "go.mod")) {
		for _, args := range [][]string{
			{"go", "build", "./..."},
			{"go", "vet", "./..."},
			{"go", "test", "./...", "-count=1", "-timeout", "120s"},
		} {
			if !process.RunIn(ctx, repoDir, args[0], args[1:]...).OK {
				core.Warn("QA failed", "cmd", core.Join(" ", args...))
				return false
			}
		}
		return true
	}

	if fs.IsFile(core.JoinPath(repoDir, "composer.json")) {
		if !process.RunIn(ctx, repoDir, "composer", "install", "--no-interaction").OK {
			return false
		}
		return process.RunIn(ctx, repoDir, "composer", "test").OK
	}

	if fs.IsFile(core.JoinPath(repoDir, "package.json")) {
		if !process.RunIn(ctx, repoDir, "npm", "install").OK {
			return false
		}
		return process.RunIn(ctx, repoDir, "npm", "test").OK
	}

	return true
}

func (s *PrepSubsystem) dispatch(ctx context.Context, callRequest *mcp.CallToolRequest, input DispatchInput) (*mcp.CallToolResult, DispatchOutput, error) {
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
	_, prepOut, err := s.prepWorkspace(ctx, callRequest, prepInput)
	if err != nil {
		return nil, DispatchOutput{}, core.E("dispatch", "prep workspace failed", err)
	}

	workspaceDir := prepOut.WorkspaceDir
	prompt := prepOut.Prompt

	if input.DryRun {
		return nil, DispatchOutput{
			Success:      true,
			Agent:        input.Agent,
			Repo:         input.Repo,
			WorkspaceDir: workspaceDir,
			Prompt:       prompt,
		}, nil
	}

	if s.ServiceRuntime != nil {
		dispatchResult := s.Core().Action("runner.dispatch").Run(ctx, core.NewOptions(
			core.Option{Key: "agent", Value: input.Agent},
			core.Option{Key: "repo", Value: input.Repo},
		))
		if !dispatchResult.OK {
			workspaceStatus := &WorkspaceStatus{
				Status:    "queued",
				Agent:     input.Agent,
				Repo:      input.Repo,
				Org:       input.Org,
				Task:      input.Task,
				Branch:    prepOut.Branch,
				StartedAt: time.Now(),
				Runs:      0,
			}
			writeStatusResult(workspaceDir, workspaceStatus)
			if runnerSvc, ok := core.ServiceFor[workspaceTracker](s.Core(), "runner"); ok {
				runnerSvc.TrackWorkspace(WorkspaceName(workspaceDir), workspaceStatus)
			}
			return nil, DispatchOutput{
				Success:      true,
				Agent:        input.Agent,
				Repo:         input.Repo,
				WorkspaceDir: workspaceDir,
				OutputFile:   "queued — at concurrency limit or frozen",
			}, nil
		}
	}

	pid, processID, outputFile, err := s.spawnAgent(input.Agent, prompt, workspaceDir)
	if err != nil {
		return nil, DispatchOutput{}, err
	}

	workspaceStatus := &WorkspaceStatus{
		Status:    "running",
		Agent:     input.Agent,
		Repo:      input.Repo,
		Org:       input.Org,
		Task:      input.Task,
		Branch:    prepOut.Branch,
		PID:       pid,
		ProcessID: processID,
		StartedAt: time.Now(),
		Runs:      1,
	}
	writeStatusResult(workspaceDir, workspaceStatus)
	// Track in runner's registry (runner owns workspace state)
	if s.ServiceRuntime != nil {
		if runnerSvc, ok := core.ServiceFor[workspaceTracker](s.Core(), "runner"); ok {
			runnerSvc.TrackWorkspace(WorkspaceName(workspaceDir), workspaceStatus)
		}
	}

	return nil, DispatchOutput{
		Success:      true,
		Agent:        input.Agent,
		Repo:         input.Repo,
		WorkspaceDir: workspaceDir,
		PID:          pid,
		OutputFile:   outputFile,
	}, nil
}
