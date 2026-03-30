// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ResumeInput is the input for agentic_resume.
//
//	input := agentic.ResumeInput{Workspace: "core/go-scm/task-42", Answer: "Use the existing queue config"}
type ResumeInput struct {
	Workspace string `json:"workspace"`         // workspace name (e.g. "core/go-scm/task-42")
	Answer    string `json:"answer,omitempty"`  // answer to the blocked question (written to ANSWER.md)
	Agent     string `json:"agent,omitempty"`   // override agent type (default: same as original)
	DryRun    bool   `json:"dry_run,omitempty"` // preview without executing
}

// ResumeOutput is the output for agentic_resume.
//
//	out := agentic.ResumeOutput{Success: true, Workspace: "core/go-scm/task-42", Agent: "codex"}
type ResumeOutput struct {
	Success    bool   `json:"success"`
	Workspace  string `json:"workspace"`
	Agent      string `json:"agent"`
	PID        int    `json:"pid,omitempty"`
	OutputFile string `json:"output_file,omitempty"`
	Prompt     string `json:"prompt,omitempty"`
}

func (s *PrepSubsystem) registerResumeTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_resume",
		Description: "Resume a blocked agent workspace. Writes ANSWER.md if an answer is provided, then relaunches the agent with instructions to read it and continue.",
	}, s.resume)
}

func (s *PrepSubsystem) resume(ctx context.Context, _ *mcp.CallToolRequest, input ResumeInput) (*mcp.CallToolResult, ResumeOutput, error) {
	if input.Workspace == "" {
		return nil, ResumeOutput{}, core.E("resume", "workspace is required", nil)
	}

	wsDir := core.JoinPath(WorkspaceRoot(), input.Workspace)
	repoDir := WorkspaceRepoDir(wsDir)

	// Verify workspace exists
	if !fs.IsDir(core.JoinPath(repoDir, ".git")) {
		return nil, ResumeOutput{}, core.E("resume", core.Concat("workspace not found: ", input.Workspace), nil)
	}

	// Read current status
	result := ReadStatusResult(wsDir)
	workspaceStatus, ok := workspaceStatusValue(result)
	if !ok {
		err, _ := result.Value.(error)
		return nil, ResumeOutput{}, core.E("resume", "no status.json in workspace", err)
	}

	if workspaceStatus.Status != "blocked" && workspaceStatus.Status != "failed" && workspaceStatus.Status != "completed" {
		return nil, ResumeOutput{}, core.E("resume", core.Concat("workspace is ", workspaceStatus.Status, ", not resumable (must be blocked, failed, or completed)"), nil)
	}

	// Determine agent
	agent := workspaceStatus.Agent
	if input.Agent != "" {
		agent = input.Agent
	}

	// Write ANSWER.md if answer provided
	if input.Answer != "" {
		answerPath := workspaceAnswerPath(wsDir)
		content := core.Sprintf("# Answer\n\n%s\n", input.Answer)
		if writeResult := fs.Write(answerPath, content); !writeResult.OK {
			err, _ := writeResult.Value.(error)
			return nil, ResumeOutput{}, core.E("resume", "failed to write ANSWER.md", err)
		}
	}

	// Build resume prompt — inline the task and answer, no file references
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

	// Spawn agent via go-process
	pid, processID, _, err := s.spawnAgent(agent, prompt, wsDir)
	if err != nil {
		return nil, ResumeOutput{}, err
	}

	// Update status
	workspaceStatus.Status = "running"
	workspaceStatus.PID = pid
	workspaceStatus.ProcessID = processID
	workspaceStatus.Runs++
	workspaceStatus.Question = ""
	writeStatusResult(wsDir, workspaceStatus)

	return nil, ResumeOutput{
		Success:    true,
		Workspace:  input.Workspace,
		Agent:      agent,
		PID:        pid,
		OutputFile: agentOutputFile(wsDir, agent),
	}, nil
}
