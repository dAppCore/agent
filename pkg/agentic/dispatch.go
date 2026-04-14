// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"os/exec"
	"runtime"
	"time"

	"dappco.re/go/agent/pkg/messages"
	core "dappco.re/go/core"
	"dappco.re/go/core/process"
	coremcp "dappco.re/go/mcp/pkg/mcp"
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

func (s *PrepSubsystem) registerDispatchTool(svc *coremcp.Service) {
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_dispatch",
		Description: "Dispatch a subagent (Gemini, Codex, or Claude) to work on a task. Preps a sandboxed workspace first, then spawns the agent inside it. Templates: conventions, security, coding.",
	}, s.dispatch)
}

// isNativeAgent returns true for agents that run directly on the host (no Docker).
//
//	isNativeAgent("claude")      // true
//	isNativeAgent("coderabbit")  // true
//	isNativeAgent("codex")       // false — runs in Docker
//	isNativeAgent("codex:gpt-5.4-mini") // false
func isNativeAgent(agent string) bool {
	base := agent
	if parts := core.SplitN(agent, ":", 2); len(parts) > 0 {
		base = parts[0]
	}
	return base == "claude" || base == "coderabbit"
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
			if isLEMProfile(model) {
				args = append(args, "--profile", model)
			} else {
				args = append(args, "--model", model)
			}
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
		script := localAgentCommandScript(localModel, prompt)
		return core.Result{Value: agentCommandResultValue{command: "sh", args: []string{"-c", script}}, OK: true}
	default:
		return core.Result{Value: core.E("agentCommand", core.Concat("unknown agent: ", agent), nil), OK: false}
	}
}

// isLEMProfile returns true if the model name is a known LEM profile
// (lemer, lemma, lemmy, lemrd) configured in codex config.toml.
//
//	isLEMProfile("lemmy")      // true
//	isLEMProfile("gpt-5.4")   // false
func isLEMProfile(model string) bool {
	switch model {
	case "lemer", "lemma", "lemmy", "lemrd":
		return true
	default:
		return false
	}
}

// localAgentCommandScript("devstral-24b", "Review the last 2 commits")
func localAgentCommandScript(model, prompt string) string {
	builder := core.NewBuilder()
	builder.WriteString("socat TCP-LISTEN:11434,fork,reuseaddr TCP:host.docker.internal:11434 & sleep 0.5")
	builder.WriteString(" && codex exec --dangerously-bypass-approvals-and-sandbox")
	if isLEMProfile(model) {
		builder.WriteString(" --profile ")
	} else {
		builder.WriteString(" --oss --local-provider ollama -m ")
	}
	builder.WriteString(shellQuote(model))
	builder.WriteString(" -o ../.meta/agent-codex.log ")
	builder.WriteString(shellQuote(prompt))
	return builder.String()
}

func shellQuote(value string) string {
	return core.Concat("'", core.Replace(value, "'", "'\\''"), "'")
}

const defaultDockerImage = "core-dev"

// Container runtime identifiers used by dispatch to route agent containers to
// the correct backend. Apple Container provides hardware VM isolation on
// macOS 26+, Docker is the cross-platform default, Podman is the rootless
// fallback for Linux environments.
const (
	// RuntimeAuto picks the first available runtime in preference order.
	//   resolved := resolveContainerRuntime("auto")  // → "apple" on macOS 26+, "docker" elsewhere
	RuntimeAuto = "auto"
	// RuntimeApple uses Apple Containers (macOS 26+, Virtualisation.framework).
	//   resolved := resolveContainerRuntime("apple")  // → "apple" if /usr/bin/container or `container` in PATH
	RuntimeApple = "apple"
	// RuntimeDocker uses Docker Engine (Docker Desktop on macOS, dockerd on Linux).
	//   resolved := resolveContainerRuntime("docker")  // → "docker" if `docker` in PATH
	RuntimeDocker = "docker"
	// RuntimePodman uses Podman (rootless containers, popular on RHEL/Fedora).
	//   resolved := resolveContainerRuntime("podman")  // → "podman" if `podman` in PATH
	RuntimePodman = "podman"
)

// containerRuntimeBinary returns the executable name for a runtime identifier.
//
//	containerRuntimeBinary("apple")   // "container"
//	containerRuntimeBinary("docker")  // "docker"
//	containerRuntimeBinary("podman")  // "podman"
func containerRuntimeBinary(runtime string) string {
	switch runtime {
	case RuntimeApple:
		return "container"
	case RuntimePodman:
		return "podman"
	default:
		return "docker"
	}
}

// goosIsDarwin reports whether the running process is on macOS. Captured at
// package init so tests can compare against a fixed value without taking a
// dependency on the `runtime` package themselves.
var goosIsDarwin = runtime.GOOS == "darwin"

// runtimeAvailable reports whether the runtime's binary is available on PATH
// or via known absolute paths. Apple Container additionally requires macOS as
// the host operating system because the binary is a thin wrapper over
// Virtualisation.framework.
//
//	runtimeAvailable("docker")  // true if `docker` binary on PATH
//	runtimeAvailable("apple")   // true on macOS when `container` binary on PATH
func runtimeAvailable(name string) bool {
	switch name {
	case RuntimeApple:
		if !goosIsDarwin {
			return false
		}
	case RuntimeDocker, RuntimePodman:
		// supported on every platform that ships the binary
	default:
		return false
	}
	binary := containerRuntimeBinary(name)
	if _, err := exec.LookPath(binary); err == nil {
		return true
	}
	return false
}

// resolveContainerRuntime returns the concrete runtime identifier for the
// requested runtime preference. "auto" picks the first available runtime in
// the preferred order (apple → docker → podman). An explicit runtime is
// honoured if the binary is on PATH; otherwise it falls back to docker so
// dispatch never silently breaks.
//
//	resolveContainerRuntime("")        // → "docker" (fallback)
//	resolveContainerRuntime("auto")    // → "apple" on macOS 26+, "docker" elsewhere
//	resolveContainerRuntime("apple")   // → "apple" if available, else "docker"
//	resolveContainerRuntime("podman")  // → "podman" if available, else "docker"
func resolveContainerRuntime(preferred string) string {
	switch preferred {
	case RuntimeApple, RuntimeDocker, RuntimePodman:
		if runtimeAvailable(preferred) {
			return preferred
		}
	}
	for _, candidate := range []string{RuntimeApple, RuntimeDocker, RuntimePodman} {
		if runtimeAvailable(candidate) {
			return candidate
		}
	}
	return RuntimeDocker
}

// dispatchRuntime returns the configured runtime preference (yaml
// `dispatch.runtime`) or the default ("auto"). The CORE_AGENT_RUNTIME
// environment variable wins for ad-hoc overrides during tests or CI.
//
//	rt := s.dispatchRuntime()  // "auto" | "apple" | "docker" | "podman"
func (s *PrepSubsystem) dispatchRuntime() string {
	if envValue := core.Env("CORE_AGENT_RUNTIME"); envValue != "" {
		return envValue
	}
	if s == nil || s.ServiceRuntime == nil {
		return RuntimeAuto
	}
	dispatchConfig, ok := s.Core().Config().Get("agents.dispatch").Value.(DispatchConfig)
	if !ok || dispatchConfig.Runtime == "" {
		return RuntimeAuto
	}
	return dispatchConfig.Runtime
}

// dispatchImage returns the configured container image (yaml `dispatch.image`)
// falling back to AGENT_DOCKER_IMAGE and finally `core-dev`.
//
//	image := s.dispatchImage()  // "core-dev" | "core-ml" | configured value
func (s *PrepSubsystem) dispatchImage() string {
	if envValue := core.Env("AGENT_DOCKER_IMAGE"); envValue != "" {
		return envValue
	}
	if s != nil && s.ServiceRuntime != nil {
		dispatchConfig, ok := s.Core().Config().Get("agents.dispatch").Value.(DispatchConfig)
		if ok && dispatchConfig.Image != "" {
			return dispatchConfig.Image
		}
	}
	return defaultDockerImage
}

// dispatchGPU reports whether GPU passthrough is enabled (yaml `dispatch.gpu`).
// When true, dispatch adds Metal passthrough on Apple Containers (when
// available) or `--gpus=all` on Docker for NVIDIA passthrough.
//
//	gpu := s.dispatchGPU()  // false unless agents.yaml sets dispatch.gpu: true
func (s *PrepSubsystem) dispatchGPU() bool {
	if s == nil || s.ServiceRuntime == nil {
		return false
	}
	dispatchConfig, ok := s.Core().Config().Get("agents.dispatch").Value.(DispatchConfig)
	if !ok {
		return false
	}
	return dispatchConfig.GPU
}

// command, args := containerCommand("codex", []string{"exec", "--model", "gpt-5.4"}, "/srv/.core/workspace/core/go-io/task-5", "/srv/.core/workspace/core/go-io/task-5/.meta")
func containerCommand(command string, args []string, workspaceDir, metaDir string) (string, []string) {
	return containerCommandFor(RuntimeDocker, defaultDockerImage, false, command, args, workspaceDir, metaDir)
}

// containerCommandFor builds the runtime-specific command line for executing
// an agent inside a container. Docker and Podman share an identical CLI
// surface (run/-rm/-v/-e), so they only differ in binary name. Apple
// Containers use the same flag shape (`container run -v ...`) per the
// Virtualisation.framework wrapper introduced in macOS 26.
//
//	command, args := containerCommandFor(RuntimeDocker, "core-dev", false, "codex", []string{"exec"}, ws, meta)
//	command, args := containerCommandFor(RuntimeApple, "core-dev", true, "claude", nil, ws, meta)
func containerCommandFor(containerRuntime, image string, gpu bool, command string, args []string, workspaceDir, metaDir string) (string, []string) {
	if image == "" {
		image = defaultDockerImage
	}
	if envImage := core.Env("AGENT_DOCKER_IMAGE"); envImage != "" {
		image = envImage
	}

	home := HomeDir()

	containerArgs := []string{"run", "--rm"}
	// Apple Containers don't support `--add-host=host-gateway`; the host-gateway
	// alias is a Docker-only convenience for reaching the host loopback.
	if containerRuntime != RuntimeApple {
		containerArgs = append(containerArgs, "--add-host=host.docker.internal:host-gateway")
	}
	if gpu {
		switch containerRuntime {
		case RuntimeDocker, RuntimePodman:
			// NVIDIA passthrough — `--gpus=all` is the standard NVIDIA Container Toolkit flag.
			containerArgs = append(containerArgs, "--gpus=all")
		case RuntimeApple:
			// Metal passthrough — flagged for the macOS 26 roadmap; emit the
			// flag so Apple's runtime can opt-in once it ships GPU support.
			containerArgs = append(containerArgs, "--gpu=metal")
		}
	}
	containerArgs = append(containerArgs,
		"-v", core.Concat(workspaceDir, ":/workspace"),
		"-v", core.Concat(metaDir, ":/workspace/.meta"),
		"-w", "/workspace/repo",
		"-v", core.Concat(core.JoinPath(home, ".codex"), ":/home/agent/.codex"),
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
	)

	if command == "claude" {
		containerArgs = append(containerArgs,
			"-v", core.Concat(core.JoinPath(home, ".claude"), ":/home/agent/.claude:ro"),
		)
	}

	if command == "gemini" {
		containerArgs = append(containerArgs,
			"-v", core.Concat(core.JoinPath(home, ".gemini"), ":/home/agent/.gemini:ro"),
		)
	}

	quoted := core.NewBuilder()
	quoted.WriteString("if [ ! -d /workspace/repo ]; then echo 'missing /workspace/repo' >&2; exit 1; fi")
	if command != "" {
		quoted.WriteString("; ")
		quoted.WriteString(command)
		for _, a := range args {
			quoted.WriteString(" '")
			quoted.WriteString(core.Replace(a, "'", "'\\''"))
			quoted.WriteString("'")
		}
	}
	quoted.WriteString("; chmod -R a+w /workspace /workspace/.meta 2>/dev/null; true")

	containerArgs = append(containerArgs, image, "sh", "-c", quoted.String())

	return containerRuntimeBinary(containerRuntime), containerArgs
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
				backoffDuration := 30 * time.Minute
				until := time.Now().Add(backoffDuration)
				s.backoff[pool] = until
				s.persistRuntimeState()
				if s.ServiceRuntime != nil {
					s.Core().ACTION(messages.RateLimitDetected{
						Pool:     pool,
						Duration: backoffDuration.String(),
					})
				}
				core.Print(nil, "rate-limit detected for %s — pausing pool for 30 minutes", pool)
				return true
			}
		} else {
			s.failCount[pool] = 0
		}
	} else {
		s.failCount[pool] = 0
	}
	s.persistRuntimeState()
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
		// Push to MCP channel so Claude Code receives the notification
		s.Core().ACTION(coremcp.ChannelPush{
			Channel: coremcp.ChannelAgentStatus,
			Data: map[string]any{
				"agent": agent, "repo": repo,
				"workspace": workspaceName, "status": "running",
			},
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
		// Push to MCP channel so Claude Code receives the notification
		s.Core().ACTION(coremcp.ChannelPush{
			Channel: coremcp.ChannelAgentComplete,
			Data: map[string]any{
				"agent": agent, "repo": repo,
				"workspace": workspaceName, "status": finalStatus,
			},
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
}

// pid, processID, outputFile, err := s.spawnAgent(agent, prompt, workspaceDir)
func (s *PrepSubsystem) spawnAgent(agent, prompt, workspaceDir string) (int, string, string, error) {
	command, args, err := agentCommand(agent, prompt)
	if err != nil {
		return 0, "", "", err
	}

	metaDir := WorkspaceMetaDir(workspaceDir)
	outputFile := agentOutputFile(workspaceDir, agent)

	fs.Delete(WorkspaceBlockedPath(workspaceDir))

	if !isNativeAgent(agent) {
		runtimeName := resolveContainerRuntime(s.dispatchRuntime())
		command, args = containerCommandFor(runtimeName, s.dispatchImage(), s.dispatchGPU(), command, args, workspaceDir, metaDir)
	}

	processResult := s.Core().Service("process")
	if !processResult.OK {
		return 0, "", "", core.E("dispatch.spawnAgent", "process service not registered", nil)
	}
	procSvc, ok := processResult.Value.(*process.Service)
	if !ok {
		return 0, "", "", core.E("dispatch.spawnAgent", "process service has unexpected type", nil)
	}
	// Native agents run in repo/ (the git checkout).
	// Docker agents run in workspaceDir (container maps it to /workspace).
	runDir := workspaceDir
	if isNativeAgent(agent) {
		runDir = WorkspaceRepoDir(workspaceDir)
	}

	proc, err := procSvc.StartWithOptions(context.Background(), process.RunOptions{
		Command: command,
		Args:    args,
		Dir:     runDir,
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

// runQA executes the RFC §7 completion pipeline QA step — captures every
// lint finding, build, and test result into a go-store workspace buffer and
// commits the cycle to the journal when a store is available. Falls back to
// the legacy build/vet/test cascade when go-store is not loaded (RFC §15.6).
//
// Usage example: `passed := s.runQA("/workspace/core/go-io/task-5")`
func (s *PrepSubsystem) runQA(workspaceDir string) bool {
	return s.runQAWithReport(context.Background(), workspaceDir)
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
			if runnerResult := s.Core().Service("runner"); runnerResult.OK {
				if runnerSvc, ok := runnerResult.Value.(workspaceTracker); ok {
					runnerSvc.TrackWorkspace(WorkspaceName(workspaceDir), workspaceStatus)
				}
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
	if s.ServiceRuntime != nil {
		if runnerResult := s.Core().Service("runner"); runnerResult.OK {
			if runnerSvc, ok := runnerResult.Value.(workspaceTracker); ok {
				runnerSvc.TrackWorkspace(WorkspaceName(workspaceDir), workspaceStatus)
			}
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
