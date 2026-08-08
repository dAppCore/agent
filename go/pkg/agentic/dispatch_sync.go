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
	// the agent succeeded.
	//
	// Fill-missing rather than write-if-absent: the VZ fork's success path
	// pre-writes a MINIMAL status (Status/Agent/StartedAt/Runtime) inside
	// spawnAgentVZ, which would make a plain write-if-absent skip and leave
	// Repo/Branch/PID empty — auto-PR (autoCreatePR requires both) then no-ops.
	// Reading and filling only the empty fields restores the full record for VZ
	// while still preserving a complete status a resume/mock already placed (a
	// full pre-existing status makes every fill a no-op).
	dispatched := &WorkspaceStatus{
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
	if existing, ok := workspaceStatusValue(ReadStatusResult(workspaceDir)); ok {
		fillMissingDispatchStatus(dispatched, existing)
	}
	// Reported: the status file is what the monitor polls, so a silent write
	// failure leaves the workspace looking stuck indefinitely.
	if r := writeStatusResult(workspaceDir, dispatched); !r.OK {
		core.Warn("agentic: failed to write workspace status", "reason", r.Value)
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

// fillMissingDispatchStatus overlays the non-empty fields of an existing
// on-disk status onto the freshly-built dispatch status. A complete pre-existing
// status (resume/mock) thus wins on every field it sets — preserved unchanged;
// a minimal status (the VZ fork's success-path pre-write, which carries only
// Status/Agent/StartedAt/Runtime) contributes just those fields, so the dispatch
// input fills the rest (Repo/Org/Task/Branch/PID/ProcessID/Runs). This keeps the
// VZ Runtime tag while restoring the full record auto-PR + tracking need.
func fillMissingDispatchStatus(dst, existing *WorkspaceStatus) {
	if dst == nil || existing == nil {
		return
	}
	if existing.Status != "" {
		dst.Status = existing.Status
	}
	if existing.Agent != "" {
		dst.Agent = existing.Agent
	}
	if existing.Repo != "" {
		dst.Repo = existing.Repo
	}
	if existing.Org != "" {
		dst.Org = existing.Org
	}
	if existing.Task != "" {
		dst.Task = existing.Task
	}
	if existing.Branch != "" {
		dst.Branch = existing.Branch
	}
	if existing.Issue != 0 {
		dst.Issue = existing.Issue
	}
	if existing.PID != 0 {
		dst.PID = existing.PID
	}
	if existing.ProcessID != "" {
		dst.ProcessID = existing.ProcessID
	}
	if !existing.StartedAt.IsZero() {
		dst.StartedAt = existing.StartedAt
	}
	if existing.Runs != 0 {
		dst.Runs = existing.Runs
	}
	if existing.PRURL != "" {
		dst.PRURL = existing.PRURL
	}
	if existing.Question != "" {
		dst.Question = existing.Question
	}
	if existing.Note != "" {
		dst.Note = existing.Note
	}
	if existing.Runtime != "" {
		dst.Runtime = existing.Runtime
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
