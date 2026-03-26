// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"syscall"
	"time"

	core "dappco.re/go/core"
)

// DispatchSyncInput is the input for a synchronous (blocking) task run.
//
//	input := agentic.DispatchSyncInput{Repo: "go-crypt", Agent: "codex:gpt-5.3-codex-spark", Task: "fix it", Issue: 7}
type DispatchSyncInput struct {
	Org   string
	Repo  string
	Agent string
	Task  string
	Issue int
}

// DispatchSyncResult is the output of a synchronous task run.
//
//	if result.OK { fmt.Println("done:", result.Status) }
type DispatchSyncResult struct {
	OK     bool
	Status string
	Error  string
	PRURL  string
}

// DispatchSync preps a workspace, spawns the agent directly (no queue, no concurrency check),
// and blocks until the agent completes.
//
//	result := prep.DispatchSync(ctx, input)
func (s *PrepSubsystem) DispatchSync(ctx context.Context, input DispatchSyncInput) DispatchSyncResult {
	// Prep workspace
	prepInput := PrepInput{
		Org:   input.Org,
		Repo:  input.Repo,
		Task:  input.Task,
		Agent: input.Agent,
		Issue: input.Issue,
	}

	prepCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	_, prepOut, err := s.prepWorkspace(prepCtx, nil, prepInput)
	if err != nil {
		return DispatchSyncResult{Error: err.Error()}
	}
	if !prepOut.Success {
		return DispatchSyncResult{Error: "prep failed"}
	}

	wsDir := prepOut.WorkspaceDir
	prompt := prepOut.Prompt

	core.Print(nil, "  workspace: %s", wsDir)
	core.Print(nil, "  branch:    %s", prepOut.Branch)

	// Spawn agent directly — no queue, no concurrency check
	pid, _, err := s.spawnAgent(input.Agent, prompt, wsDir)
	if err != nil {
		return DispatchSyncResult{Error: err.Error()}
	}

	core.Print(nil, "  pid:       %d", pid)
	core.Print(nil, "  waiting for completion...")

	// Poll for process exit
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return DispatchSyncResult{Error: "cancelled"}
		case <-ticker.C:
			if pid > 0 && syscall.Kill(pid, 0) != nil {
				// Process exited — read final status
				st, err := ReadStatus(wsDir)
				if err != nil {
					return DispatchSyncResult{Error: "can't read final status"}
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
