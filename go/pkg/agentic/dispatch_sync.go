// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"time"

	core "dappco.re/go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// input := agentic.DispatchSyncInput{Repo: "go-crypt", Agent: "codex:gpt-5.3-codex-spark", Task: "fix it", Issue: 7}
type DispatchSyncInput struct {
	Org    string
	Repo   string
	Agent  string
	Task   string
	Issue  int
	Branch string
}

// if result.OK { core.Print(nil, "done: %s", result.Status) }
// if !result.OK { core.Print(nil, "%v", result.Error) }
type DispatchSyncResult struct {
	OK     bool
	Status string
	Error  error
	PRURL  string
}

// result := prep.DispatchSync(ctx, input)
func (s *PrepSubsystem) DispatchSync(ctx context.Context, input DispatchSyncInput) DispatchSyncResult {
	prepInput := PrepInput{
		Org:    input.Org,
		Repo:   input.Repo,
		Task:   input.Task,
		Agent:  input.Agent,
		Issue:  input.Issue,
		Branch: input.Branch,
	}

	prepContext, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	prepWorkspaceFn := func(ctx context.Context, request *mcp.CallToolRequest, input PrepInput) (*mcp.CallToolResult, PrepOutput, error) {
		return prepWorkspace(s, ctx, request, input)
	}
	if s.dispatchSyncPrep != nil {
		prepWorkspaceFn = s.dispatchSyncPrep
	}

	_, prepOut, err := prepWorkspaceFn(prepContext, nil, prepInput)
	if err != nil {
		return DispatchSyncResult{Error: core.E("agentic.DispatchSync", "prep workspace failed", err)}
	}
	if !prepOut.Success {
		return DispatchSyncResult{Error: core.E("agentic.DispatchSync", "prep failed", nil)}
	}

	workspaceDir := prepOut.WorkspaceDir
	prompt := prepOut.Prompt

	core.Print(nil, "  workspace: %s", workspaceDir)
	core.Print(nil, "  branch:    %s", prepOut.Branch)

	spawnAgentFn := func(subsystem *PrepSubsystem, agent, prompt, dir string) (int, string, string, error) {
		return spawnAgent(subsystem, agent, prompt, dir)
	}
	if s.dispatchSyncSpawn != nil {
		spawnAgentFn = func(_ *PrepSubsystem, agent, prompt, dir string) (int, string, string, error) {
			return s.dispatchSyncSpawn(agent, prompt, dir)
		}
	}

	pid, processID, _, err := spawnAgentFn(s, input.Agent, prompt, workspaceDir)
	if err != nil {
		return DispatchSyncResult{Error: core.E("agentic.DispatchSync", "spawn agent failed", err)}
	}

	// The async dispatch() writes the initial "running" status after spawn; the
	// sync path must too. A native dispatch (opencode runs on the host with no
	// in-container wrapper to create status.json) would otherwise leave the
	// workspace status-less, and both the poll below and the completion monitor
	// fail to read a final status — surfacing as "status not found" even when
	// the agent succeeded. Write-if-absent so a status a resume/mock already
	// placed is preserved.
	if _, ok := workspaceStatusValue(ReadStatusResult(workspaceDir)); !ok {
		writeStatusResult(workspaceDir, &WorkspaceStatus{
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
		})
	}

	core.Print(nil, "  pid:       %d", pid)
	core.Print(nil, "  waiting for completion...")

	var runtime *core.Core
	if s.ServiceRuntime != nil {
		runtime = s.Core()
	}

	tick := 3 * time.Second
	if s.dispatchSyncTick > 0 {
		tick = s.dispatchSyncTick
	}

	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return DispatchSyncResult{Error: core.E("agentic.DispatchSync", "cancelled", ctx.Err())}
		case <-ticker.C:
			if pid > 0 && !ProcessAlive(runtime, processID, pid) {
				result := ReadStatusResult(workspaceDir)
				st, ok := workspaceStatusValue(result)
				if !ok {
					err, _ := result.Value.(error)
					return DispatchSyncResult{Error: core.E("agentic.DispatchSync", "can't read final status", err)}
				}
				return DispatchSyncResult{
					OK:     st.Status == "completed",
					Status: st.Status,
					PRURL:  st.PRURL,
				}
			}
		}
	}
}

// result := c.Action("agentic.dispatch.sync").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "repo", Value: "go-io"},
//	core.Option{Key: "task", Value: "Fix the failing tests"},
//
// ))
func (s *PrepSubsystem) handleDispatchSync(ctx context.Context, options core.Options) core.Result {
	input := dispatchSyncInputFromOptions(options)
	result := s.DispatchSync(ctx, input)
	if result.Error != nil {
		return core.Result{Value: result.Error, OK: false}
	}
	return core.Result{Value: result, OK: result.OK}
}

func dispatchSyncInputFromOptions(options core.Options) DispatchSyncInput {
	return DispatchSyncInput{
		Org:    optionStringValue(options, "org"),
		Repo:   optionStringValue(options, "repo", "_arg"),
		Agent:  optionStringValue(options, "agent"),
		Task:   optionStringValue(options, "task"),
		Issue:  optionIntValue(options, "issue"),
		Branch: optionStringValue(options, "branch"),
	}
}
