// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// input := agentic.ResumeInput{Workspace: "core/go-scm/task-42", Answer: "Use the existing queue config"}
type ResumeInput struct {
	Workspace string `json:"workspace"`
	Answer    string `json:"answer,omitempty"`
	Agent     string `json:"agent,omitempty"`
	DryRun    bool   `json:"dry_run,omitempty"`
}

// out := agentic.ResumeOutput{Success: true, Workspace: "core/go-scm/task-42", Agent: "codex"}
type ResumeOutput struct {
	Success    bool   `json:"success"`
	Workspace  string `json:"workspace"`
	Agent      string `json:"agent"`
	PID        int    `json:"pid,omitempty"`
	OutputFile string `json:"output_file,omitempty"`
	Prompt     string `json:"prompt,omitempty"`
}

func (s *PrepSubsystem) registerResumeTool(svc *coremcp.Service) {
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_resume",
		Description: "Resume a blocked agent workspace. Writes ANSWER.md if an answer is provided, then relaunches the agent with instructions to read it and continue.",
	}, toolHandlerFor[ResumeInput, ResumeOutput]("resume", "invalid resume output", s.resume))
}

func (s *PrepSubsystem) resume(ctx context.Context, input ResumeInput) core.Result {
	if input.Workspace == "" {
		return core.Fail(core.E("resume", "workspace is required", nil))
	}

	workspaceDir := core.JoinPath(WorkspaceRoot(), input.Workspace)
	repoDir := WorkspaceRepoDir(workspaceDir)

	if !fs.IsDir(core.JoinPath(repoDir, ".git")).OK {
		return core.Fail(core.E("resume", core.Concat("workspace not found: ", input.Workspace), nil))
	}

	result := ReadStatusResult(workspaceDir)
	workspaceStatus, ok := workspaceStatusValue(result)
	if !ok {
		err, _ := result.Value.(error)
		return core.Fail(core.E("resume", "no status.json in workspace", err))
	}

	if workspaceStatus.Status != "blocked" && workspaceStatus.Status != "failed" && workspaceStatus.Status != "completed" {
		return core.Fail(core.E("resume", core.Concat("workspace is ", workspaceStatus.Status, ", not resumable (must be blocked, failed, or completed)"), nil))
	}

	agent := workspaceStatus.Agent
	if input.Agent != "" {
		agent = input.Agent
	}

	if input.Answer != "" {
		answerPath := workspaceAnswerPath(workspaceDir)
		content := core.Sprintf("# Answer\n\n%s\n", input.Answer)
		if writeResult := fs.Write(answerPath, content); !writeResult.OK {
			err, _ := writeResult.Value.(error)
			return core.Fail(core.E("resume", "failed to write ANSWER.md", err))
		}
	}

	prompt := core.Concat("You are resuming previous work.\n\nORIGINAL TASK:\n", workspaceStatus.Task)
	if input.Answer != "" {
		prompt = core.Concat(prompt, "\n\nANSWER TO YOUR QUESTION:\n", input.Answer)
	}
	prompt = core.Concat(prompt, "\n\nContinue working. Read BLOCKED.md to see what you were stuck on. Commit when done.")

	if input.DryRun {
		return core.Ok(ResumeOutput{
			Success:   true,
			Workspace: input.Workspace,
			Agent:     agent,
			Prompt:    prompt,
		})
	}

	pid, processID, _, err := spawnAgent(s, agent, prompt, workspaceDir)
	if err != nil {
		return core.Fail(err)
	}

	workspaceStatus.Status = "running"
	workspaceStatus.PID = pid
	workspaceStatus.ProcessID = processID
	workspaceStatus.Runs++
	workspaceStatus.Question = ""
	preserveStatusNote(workspaceDir, workspaceStatus) // keep VZ→OCI downgrade note (SP2.4)
	writeStatusResult(workspaceDir, workspaceStatus)

	return core.Ok(ResumeOutput{
		Success:    true,
		Workspace:  input.Workspace,
		Agent:      agent,
		PID:        pid,
		OutputFile: agentOutputFile(workspaceDir, agent),
	})
}
