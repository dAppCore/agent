// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"time"

	core "dappco.re/go/core"
)

// input := agentic.DispatchSyncInput{Repo: "go-crypt", Agent: "codex:gpt-5.3-codex-spark", Task: "fix it", Issue: 7}
type DispatchSyncInput struct {
	Org   string
	Repo  string
	Agent string
	Task  string
	Issue int
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
	// Prep workspace
	prepInput := PrepInput{
		Org:   input.Org,
		Repo:  input.Repo,
		Task:  input.Task,
		Agent: input.Agent,
		Issue: input.Issue,
	}

	prepContext, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	_, prepOut, err := s.prepWorkspace(prepContext, nil, prepInput)
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

	// Spawn agent directly — no queue, no concurrency check
	pid, processID, _, err := s.spawnAgent(input.Agent, prompt, workspaceDir)
	if err != nil {
		return DispatchSyncResult{Error: core.E("agentic.DispatchSync", "spawn agent failed", err)}
	}

	core.Print(nil, "  pid:       %d", pid)
	core.Print(nil, "  waiting for completion...")

	var runtime *core.Core
	if s.ServiceRuntime != nil {
		runtime = s.Core()
	}

	// Poll for process exit
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return DispatchSyncResult{Error: core.E("agentic.DispatchSync", "cancelled", ctx.Err())}
		case <-ticker.C:
			if pid > 0 && !ProcessAlive(runtime, processID, pid) {
				// Process exited — read final status
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
