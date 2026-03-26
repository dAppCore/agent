// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ResumeInput is the input for agentic_resume.
//
//	input := agentic.ResumeInput{Workspace: "go-scm-1773581173", Answer: "Use the existing queue config"}
type ResumeInput struct {
	Workspace string `json:"workspace"`         // workspace name (e.g. "go-scm-1773581173")
	Answer    string `json:"answer,omitempty"`  // answer to the blocked question (written to ANSWER.md)
	Agent     string `json:"agent,omitempty"`   // override agent type (default: same as original)
	DryRun    bool   `json:"dry_run,omitempty"` // preview without executing
}

// ResumeOutput is the output for agentic_resume.
//
//	out := agentic.ResumeOutput{Success: true, Workspace: "go-scm-1773581173", Agent: "codex"}
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
	repoDir := core.JoinPath(wsDir, "repo")

	// Verify workspace exists
	if !fs.IsDir(core.JoinPath(repoDir, ".git")) {
		return nil, ResumeOutput{}, core.E("resume", core.Concat("workspace not found: ", input.Workspace), nil)
	}

	// Read current status
	st, err := ReadStatus(wsDir)
	if err != nil {
		return nil, ResumeOutput{}, core.E("resume", "no status.json in workspace", err)
	}

	if st.Status != "blocked" && st.Status != "failed" && st.Status != "completed" {
		return nil, ResumeOutput{}, core.E("resume", core.Concat("workspace is ", st.Status, ", not resumable (must be blocked, failed, or completed)"), nil)
	}

	// Determine agent
	agent := st.Agent
	if input.Agent != "" {
		agent = input.Agent
	}

	// Write ANSWER.md if answer provided
	if input.Answer != "" {
		answerPath := core.JoinPath(repoDir, "ANSWER.md")
		content := core.Sprintf("# Answer\n\n%s\n", input.Answer)
		if r := fs.Write(answerPath, content); !r.OK {
			err, _ := r.Value.(error)
			return nil, ResumeOutput{}, core.E("resume", "failed to write ANSWER.md", err)
		}
	}

	// Build resume prompt — inline the task and answer, no file references
	prompt := core.Concat("You are resuming previous work.\n\nORIGINAL TASK:\n", st.Task)
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
	pid, _, err := s.spawnAgent(agent, prompt, wsDir)
	if err != nil {
		return nil, ResumeOutput{}, err
	}

	// Update status
	st.Status = "running"
	st.PID = pid
	st.Runs++
	st.Question = ""
	writeStatus(wsDir, st)

	return nil, ResumeOutput{
		Success:    true,
		Workspace:  input.Workspace,
		Agent:      agent,
		PID:        pid,
		OutputFile: core.JoinPath(wsDir, core.Sprintf("agent-%s.log", agent)),
	}, nil
}
