// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"syscall"
	"time"

	core "dappco.re/go/core"
	"dappco.re/go/core/process"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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
			args = append(args, "-m", "gemini-2.5-"+model)
		}
		return "gemini", args, nil
	case "codex":
		if model == "review" {
			return "codex", []string{"review", "--base", "HEAD~1"}, nil
		}
		// Codex runs from repo/ which IS a git repo — no --skip-git-repo-check
		args := []string{
			"exec",
			"--full-auto",
			"-o", "../.meta/agent-codex.log",
			prompt,
		}
		if model != "" {
			args = append(args[:3], append([]string{"--model", model}, args[3:]...)...)
		}
		return "codex", args, nil
	case "claude":
		args := []string{
			"-p", prompt,
			"--output-format", "text",
			"--dangerously-skip-permissions",
			"--no-session-persistence",
			"--append-system-prompt", "SANDBOX: You are restricted to the current directory only. " +
				"Do NOT use absolute paths. Do NOT navigate outside this repository.",
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
		script := core.JoinPath(core.Env("DIR_HOME"), "Code", "core", "agent", "scripts", "local-agent.sh")
		return "bash", []string{script, prompt}, nil
	default:
		return "", nil, core.E("agentCommand", "unknown agent: "+agent, nil)
	}
}

// spawnAgent launches an agent process in the repo/ directory.
// Output is captured and written to .meta/agent-{agent}.log on completion.
func (s *PrepSubsystem) spawnAgent(agent, prompt, wsDir string) (int, string, error) {
	command, args, err := agentCommand(agent, prompt)
	if err != nil {
		return 0, "", err
	}

	repoDir := core.JoinPath(wsDir, "repo")
	metaDir := core.JoinPath(wsDir, ".meta")
	outputFile := core.JoinPath(metaDir, core.Sprintf("agent-%s.log", agent))

	// Clean up stale BLOCKED.md from previous runs
	fs.Delete(core.JoinPath(repoDir, "BLOCKED.md"))

	proc, err := process.StartWithOptions(context.Background(), process.RunOptions{
		Command: command,
		Args:    args,
		Dir:     repoDir,
		Env:     []string{"TERM=dumb", "NO_COLOR=1", "CI=true"},
		Detach:  true,
	})
	if err != nil {
		return 0, "", core.E("dispatch.spawnAgent", "failed to spawn "+agent, err)
	}

	proc.CloseStdin()
	pid := proc.Info().PID

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-proc.Done():
				goto done
			case <-ticker.C:
				if err := syscall.Kill(pid, 0); err != nil {
					goto done
				}
			}
		}
	done:

		if output := proc.Output(); output != "" {
			fs.Write(outputFile, output)
		}

		finalStatus := "completed"
		exitCode := proc.Info().ExitCode
		procStatus := proc.Info().Status
		question := ""

		blockedPath := core.JoinPath(repoDir, "BLOCKED.md")
		if r := fs.Read(blockedPath); r.OK && core.Trim(r.Value.(string)) != "" {
			finalStatus = "blocked"
			question = core.Trim(r.Value.(string))
		} else if exitCode != 0 || procStatus == "failed" || procStatus == "killed" {
			finalStatus = "failed"
			if exitCode != 0 {
				question = core.Sprintf("Agent exited with code %d", exitCode)
			}
		}

		if st, stErr := readStatus(wsDir); stErr == nil {
			st.Status = finalStatus
			st.PID = 0
			st.Question = question
			writeStatus(wsDir, st)
		}

		emitCompletionEvent(agent, core.PathBase(wsDir), finalStatus)

		if s.onComplete != nil {
			s.onComplete.Poke()
		}

		if finalStatus == "completed" {
			s.autoCreatePR(wsDir)
			s.autoVerifyAndMerge(wsDir)
		}

		s.ingestFindings(wsDir)
		s.drainQueue()
	}()

	return pid, outputFile, nil
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

	// Step 2: Check per-agent concurrency limit
	if !s.canDispatchAgent(input.Agent) {
		writeStatus(wsDir, &WorkspaceStatus{
			Status:    "queued",
			Agent:     input.Agent,
			Repo:      input.Repo,
			Org:       input.Org,
			Task:      input.Task,
			Branch:    prepOut.Branch,
			StartedAt: time.Now(),
			Runs:      0,
		})
		return nil, DispatchOutput{
			Success:      true,
			Agent:        input.Agent,
			Repo:         input.Repo,
			WorkspaceDir: wsDir,
			OutputFile:   "queued — waiting for a slot",
		}, nil
	}

	// Step 3: Spawn agent in repo/ directory
	pid, outputFile, err := s.spawnAgent(input.Agent, prompt, wsDir)
	if err != nil {
		return nil, DispatchOutput{}, err
	}

	writeStatus(wsDir, &WorkspaceStatus{
		Status:    "running",
		Agent:     input.Agent,
		Repo:      input.Repo,
		Org:       input.Org,
		Task:      input.Task,
		Branch:    prepOut.Branch,
		PID:       pid,
		StartedAt: time.Now(),
		Runs:      1,
	})

	return nil, DispatchOutput{
		Success:      true,
		Agent:        input.Agent,
		Repo:         input.Repo,
		WorkspaceDir: wsDir,
		PID:          pid,
		OutputFile:   outputFile,
	}, nil
}
