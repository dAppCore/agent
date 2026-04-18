// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go/core"
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
	}, s.resume)
}

func (s *PrepSubsystem) resume(ctx context.Context, _ *mcp.CallToolRequest, input ResumeInput) (*mcp.CallToolResult, ResumeOutput, error) {
	if input.Workspace == "" {
		return nil, ResumeOutput{}, core.E("resume", "workspace is required", nil)
	}

	workspaceDir := core.JoinPath(WorkspaceRoot(), input.Workspace)
	repoDir := WorkspaceRepoDir(workspaceDir)

	if !fs.IsDir(core.JoinPath(repoDir, ".git")) {
		return nil, ResumeOutput{}, core.E("resume", core.Concat("workspace not found: ", input.Workspace), nil)
	}

	result := ReadStatusResult(workspaceDir)
	workspaceStatus, ok := workspaceStatusValue(result)
	if !ok {
		err, _ := result.Value.(error)
		return nil, ResumeOutput{}, core.E("resume", "no status.json in workspace", err)
	}

	if workspaceStatus.Status != "blocked" && workspaceStatus.Status != "failed" && workspaceStatus.Status != "completed" {
		return nil, ResumeOutput{}, core.E("resume", core.Concat("workspace is ", workspaceStatus.Status, ", not resumable (must be blocked, failed, or completed)"), nil)
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
			return nil, ResumeOutput{}, core.E("resume", "failed to write ANSWER.md", err)
		}
	}

	prompt := core.Concat("You are resuming previous work.\n\nORIGINAL TASK:\n", workspaceStatus.Task)
	if input.Answer != "" {
		prompt = core.Concat(prompt, "\n\nANSWER TO YOUR QUESTION:\n", input.Answer)
	}
	prompt = core.Concat(prompt, "\n\nContinue working. Read BLOCKED.md to see what you were stuck on. Commit when done.")

	if input.DryRun {
		return nil, ResumeOutput{
			Success:   true,
			Workspace: input.Workspace,
			Agent:     agent,
			Prompt:    prompt,
		}, nil
	}

	pid, processID, _, err := s.spawnAgent(agent, prompt, workspaceDir)
	if err != nil {
		return nil, ResumeOutput{}, err
	}

	workspaceStatus.Status = "running"
	workspaceStatus.PID = pid
	workspaceStatus.ProcessID = processID
	workspaceStatus.Runs++
	workspaceStatus.Question = ""
	writeStatusResult(workspaceDir, workspaceStatus)

	return nil, ResumeOutput{
		Success:    true,
		Workspace:  input.Workspace,
		Agent:      agent,
		PID:        pid,
		OutputFile: agentOutputFile(workspaceDir, agent),
	}, nil
}
