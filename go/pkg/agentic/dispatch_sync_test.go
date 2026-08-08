// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestDispatchsync_ContainerCommand_Good_Case(t *testing.T) {
	cmd, args := containerCommand("codex", []string{"--model", "gpt-5.4"}, "/workspace/task-5", "/workspace/task-5/.meta")
	core.AssertEqual(t, "docker", cmd)
	core.AssertContains(t, args, "run")
	core.AssertContains(t, args, "/workspace/task-5:/workspace")
	core.AssertContains(t, args, "/workspace/task-5/.meta:/workspace/.meta")
	core.AssertContains(t, args, "/workspace/repo")
}

func TestDispatchsync_ContainerCommand_Bad_UnknownAgent(t *testing.T) {
	cmd, args := containerCommand("unknown", nil, "/workspace/task-5", "/workspace/task-5/.meta")
	core.AssertEqual(t, "docker", cmd)
	core.AssertNotEmpty(t, args)
}

func TestDispatchsync_ContainerCommand_Ugly_EmptyArgs(t *testing.T) {
	core.AssertNotPanics(t, func() {
		containerCommand("codex", nil, "", "")
	})
}

func TestDispatchsync_HandleDispatchSync_Good_Completed(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	workspaceDir := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-7")
	s := &PrepSubsystem{dispatchSyncTick: 10 * time.Millisecond}

	s.dispatchSyncPrep = func(ctx context.Context, _ *mcpsdk.CallToolRequest, input PrepInput) (*mcpsdk.CallToolResult, PrepOutput, error) {
		core.AssertEqual(t, "core", input.Org)
		core.AssertEqual(t, "go-io", input.Repo)
		core.AssertEqual(t, "codex", input.Agent)
		core.AssertEqual(t, "Fix tests", input.Task)
		core.AssertEqual(t, 7, input.Issue)

		core.RequireTrue(t, fs.EnsureDir(workspaceDir).OK)
		core.RequireTrue(t, fs.Write(core.JoinPath(workspaceDir, "status.json"), core.JSONMarshalString(&WorkspaceStatus{
			Status: "completed",
			PRURL:  "https://forge.test/core/go-io/pulls/7",
		})).OK)

		return nil, PrepOutput{
			Success:      true,
			WorkspaceDir: workspaceDir,
			Branch:       "agent/fix-tests",
			Prompt:       "prompt",
		}, nil
	}
	s.dispatchSyncSpawn = func(agent, prompt, dir string) (int, string, string, error) {
		core.AssertEqual(t, "codex", agent)
		core.AssertEqual(t, "prompt", prompt)
		core.AssertEqual(t, workspaceDir, dir)
		return 321, "process-321", core.JoinPath(dir, ".meta", "agent.log"), nil
	}

	result := s.handleDispatchSync(context.Background(), core.NewOptions(
		core.Option{Key: "org", Value: "core"},
		core.Option{Key: "repo", Value: "go-io"},
		core.Option{Key: "agent", Value: "codex"},
		core.Option{Key: "task", Value: "Fix tests"},
		core.Option{Key: "issue", Value: "7"},
	))

	core.RequireTrue(t, result.OK)
	output, ok := result.Value.(DispatchSyncResult)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.OK)
	core.AssertEqual(t, "completed", output.Status)
	core.AssertEqual(t, "https://forge.test/core/go-io/pulls/7", output.PRURL)
}

func TestDispatchsync_HandleDispatchSync_Bad_PrepFailure(t *testing.T) {
	s := &PrepSubsystem{}
	s.dispatchSyncPrep = func(context.Context, *mcpsdk.CallToolRequest, PrepInput) (*mcpsdk.CallToolResult, PrepOutput, error) {
		return nil, PrepOutput{}, core.E("prepWorkspace", "boom", nil)
	}

	result := s.handleDispatchSync(context.Background(), core.NewOptions(
		core.Option{Key: "repo", Value: "go-io"},
		core.Option{Key: "task", Value: "Fix tests"},
	))

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "prep workspace failed")
}

func TestDispatchsync_HandleDispatchSync_Bad_PrepIncomplete(t *testing.T) {
	s := &PrepSubsystem{}
	s.dispatchSyncPrep = func(context.Context, *mcpsdk.CallToolRequest, PrepInput) (*mcpsdk.CallToolResult, PrepOutput, error) {
		return nil, PrepOutput{
			Success: false,
		}, nil
	}

	result := s.handleDispatchSync(context.Background(), core.NewOptions(
		core.Option{Key: "repo", Value: "go-io"},
		core.Option{Key: "task", Value: "Fix tests"},
	))

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "prep failed")
}

func TestDispatchsync_HandleDispatchSync_Ugly_SpawnFailure(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	workspaceDir := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-7")
	s := &PrepSubsystem{dispatchSyncTick: 10 * time.Millisecond}

	s.dispatchSyncPrep = func(context.Context, *mcpsdk.CallToolRequest, PrepInput) (*mcpsdk.CallToolResult, PrepOutput, error) {
		core.RequireTrue(t, fs.EnsureDir(workspaceDir).OK)
		core.RequireTrue(t, fs.Write(core.JoinPath(workspaceDir, "status.json"), core.JSONMarshalString(&WorkspaceStatus{
			Status: "running",
		})).OK)

		return nil, PrepOutput{
			Success:      true,
			WorkspaceDir: workspaceDir,
			Branch:       "agent/fix-tests",
			Prompt:       "prompt",
		}, nil
	}
	s.dispatchSyncSpawn = func(agent, prompt, dir string) (int, string, string, error) {
		core.AssertEqual(t, "codex", agent)
		return 0, "", "", core.E("spawn", "boom", nil)
	}

	result := s.handleDispatchSync(context.Background(), core.NewOptions(
		core.Option{Key: "repo", Value: "go-io"},
		core.Option{Key: "agent", Value: "codex"},
		core.Option{Key: "task", Value: "Fix tests"},
	))

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "spawn agent failed")
}

func TestDispatchSync_PrepSubsystem_DispatchSync_Bad(t *testing.T) {
	subsystem := &PrepSubsystem{}
	subsystem.dispatchSyncPrep = func(context.Context, *mcpsdk.CallToolRequest, PrepInput) (*mcpsdk.CallToolResult, PrepOutput, error) {
		return nil, PrepOutput{}, core.E("prepWorkspace", "boom", nil)
	}

	result := subsystem.DispatchSync(context.Background(), DispatchSyncInput{Repo: "go-io", Task: "Fix tests"})
	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Error)
	core.AssertContains(t, result.Error.Error(), "prep workspace failed")
}

func TestDispatchSync_PrepSubsystem_DispatchSync_Ugly(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	workspaceDir := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-10")
	subsystem := &PrepSubsystem{dispatchSyncTick: 10 * time.Millisecond}
	subsystem.dispatchSyncPrep = func(context.Context, *mcpsdk.CallToolRequest, PrepInput) (*mcpsdk.CallToolResult, PrepOutput, error) {
		core.RequireTrue(t, fs.EnsureDir(workspaceDir).OK)
		core.RequireTrue(t, fs.Write(core.JoinPath(workspaceDir, "status.json"), core.JSONMarshalString(&WorkspaceStatus{Status: "running"})).OK)
		return nil, PrepOutput{Success: true, WorkspaceDir: workspaceDir, Prompt: "prompt"}, nil
	}
	subsystem.dispatchSyncSpawn = func(string, string, string) (int, string, string, error) {
		return 0, "", "", core.E("spawn", "boom", nil)
	}

	result := subsystem.DispatchSync(context.Background(), DispatchSyncInput{Repo: "go-io", Agent: "codex", Task: "Fix tests"})
	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Error)
	core.AssertContains(t, result.Error.Error(), "spawn agent failed")
}

func TestDispatchSync_PrepSubsystem_DispatchSync_Ugly_WritesInitialStatusWhenPrepDoesnt(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	workspaceDir := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-11")
	s := &PrepSubsystem{dispatchSyncTick: 10 * time.Millisecond}

	// Real-like prep: creates the workspace but does NOT pre-write status.json
	// (the actual prepWorkspace doesn't — the async dispatch() writes it after
	// spawn, which the sync path used to skip → "status not found" crash).
	s.dispatchSyncPrep = func(context.Context, *mcpsdk.CallToolRequest, PrepInput) (*mcpsdk.CallToolResult, PrepOutput, error) {
		core.RequireTrue(t, fs.EnsureDir(workspaceDir).OK)
		return nil, PrepOutput{Success: true, WorkspaceDir: workspaceDir, Branch: "agent/x", Prompt: "prompt"}, nil
	}
	s.dispatchSyncSpawn = func(string, string, string) (int, string, string, error) {
		return 42, "process-x", core.JoinPath(workspaceDir, ".meta", "agent.log"), nil
	}

	result := s.DispatchSync(context.Background(), DispatchSyncInput{
		Repo: "go-io", Agent: "codex", Task: "Fix tests", Branch: "x",
	})

	// The fix: DispatchSync wrote the initial "running" status, so the poll
	// reads it instead of erroring — no "status not found".
	core.AssertNil(t, result.Error)
	core.AssertEqual(t, "running", result.Status)
}

func TestDispatchSync_PrepSubsystem_DispatchSync_Good(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	workspaceDir := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-9")
	subsystem := &PrepSubsystem{dispatchSyncTick: 10 * time.Millisecond}

	subsystem.dispatchSyncPrep = func(_ context.Context, _ *mcpsdk.CallToolRequest, input PrepInput) (*mcpsdk.CallToolResult, PrepOutput, error) {
		core.AssertEqual(t, "core", input.Org)
		core.AssertEqual(t, "go-io", input.Repo)
		core.AssertEqual(t, "codex", input.Agent)
		core.AssertEqual(t, "Fix tests", input.Task)
		core.AssertEqual(t, 9, input.Issue)

		core.RequireTrue(t, fs.EnsureDir(workspaceDir).OK)
		core.RequireTrue(t, fs.Write(core.JoinPath(workspaceDir, "status.json"), core.JSONMarshalString(&WorkspaceStatus{
			Status: "completed",
			PRURL:  "https://forge.test/core/go-io/pulls/9",
		})).OK)

		return nil, PrepOutput{
			Success:      true,
			WorkspaceDir: workspaceDir,
			Branch:       "agent/fix-tests",
			Prompt:       "prompt",
		}, nil
	}
	subsystem.dispatchSyncSpawn = func(agent, prompt, workspaceDir string) (int, string, string, error) {
		core.AssertEqual(t, "codex", agent)
		core.AssertEqual(t, "prompt", prompt)
		core.AssertContains(t, workspaceDir, "task-9")
		return 42, "process-42", core.JoinPath(workspaceDir, ".meta", "agent.log"), nil
	}

	result := subsystem.DispatchSync(context.Background(), DispatchSyncInput{
		Org:   "core",
		Repo:  "go-io",
		Agent: "codex",
		Task:  "Fix tests",
		Issue: 9,
	})

	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, "completed", result.Status)
	core.AssertEqual(t, "https://forge.test/core/go-io/pulls/9", result.PRURL)
}

// --- fillMissingDispatchStatus (VZ minimal status must be completed; a full
//     resume/mock status must be preserved) ---

// The VZ fork's success path pre-writes a MINIMAL status (Status/Agent/StartedAt/
// Runtime). The sync caller must fill the dispatch input's Repo/Branch/PID into
// it — otherwise auto-PR (which requires Repo+Branch) no-ops on the VZ+sync path.
func TestDispatchSync_FillMissingDispatchStatus_Good_CompletesVZMinimal(t *testing.T) {
	started := time.Now().Add(-time.Minute)
	existing := &WorkspaceStatus{Status: "running", Agent: "codex", StartedAt: started, Runtime: vzRuntimeName}
	dispatched := &WorkspaceStatus{
		Status: "running", Agent: "codex", Repo: "go-io", Org: "core",
		Task: "Fix tests", Branch: "agent/fix", PID: vzSentinelPID, ProcessID: "vz-x", Runs: 1,
	}

	fillMissingDispatchStatus(dispatched, existing)

	// Dispatch input filled the fields the minimal status lacked.
	core.AssertEqual(t, "go-io", dispatched.Repo)
	core.AssertEqual(t, "agent/fix", dispatched.Branch)
	core.AssertEqual(t, vzSentinelPID, dispatched.PID)
	core.AssertEqual(t, 1, dispatched.Runs)
	// The VZ Runtime tag + true StartedAt from the pre-write are carried.
	core.AssertEqual(t, vzRuntimeName, dispatched.Runtime)
	core.AssertEqual(t, started, dispatched.StartedAt)
}

// A complete pre-existing status (a resume or mock placed it) must win on every
// field it sets — the merge must not clobber it with the fresh dispatch struct.
func TestDispatchSync_FillMissingDispatchStatus_Ugly_PreservesFullExisting(t *testing.T) {
	existing := &WorkspaceStatus{
		Status: "completed", Agent: "claude", Repo: "go-log", Org: "dAppCore",
		Branch: "feat/done", PID: 4242, ProcessID: "proc-1", Runs: 3,
		PRURL: "https://forge.test/x/pulls/1",
	}
	dispatched := &WorkspaceStatus{
		Status: "running", Agent: "codex", Repo: "go-io", Branch: "agent/new", PID: 99, Runs: 1,
	}

	fillMissingDispatchStatus(dispatched, existing)

	core.AssertEqual(t, "completed", dispatched.Status) // existing wins
	core.AssertEqual(t, "go-log", dispatched.Repo)
	core.AssertEqual(t, "feat/done", dispatched.Branch)
	core.AssertEqual(t, 4242, dispatched.PID)
	core.AssertEqual(t, 3, dispatched.Runs)
	core.AssertEqual(t, "https://forge.test/x/pulls/1", dispatched.PRURL)
}

// End-to-end on the VZ+sync path: a spawn that pre-writes a minimal VZ status
// (as recordVZRuntime does) and returns the sentinel PID must leave a status.json
// carrying Repo+Branch+Runtime. The sync poll never fires for a sentinel PID
// (pre-existing limitation), so a short context deadline ends the call after the
// status write under test.
func TestDispatchSync_PrepSubsystem_DispatchSync_Ugly_VZFillsFullStatus(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	workspaceDir := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-vz")
	s := &PrepSubsystem{dispatchSyncTick: 5 * time.Millisecond}

	s.dispatchSyncPrep = func(context.Context, *mcpsdk.CallToolRequest, PrepInput) (*mcpsdk.CallToolResult, PrepOutput, error) {
		core.RequireTrue(t, fs.EnsureDir(workspaceDir).OK)
		return nil, PrepOutput{Success: true, WorkspaceDir: workspaceDir, Branch: "agent/vz", Prompt: "prompt"}, nil
	}
	// Simulate the VZ fork: pre-write a minimal status (recordVZRuntime) and
	// return the sentinel PID.
	s.dispatchSyncSpawn = func(_, _, ws string) (int, string, string, error) {
		writeStatusResult(ws, &WorkspaceStatus{Status: "running", Agent: "codex", StartedAt: time.Now(), Runtime: vzRuntimeName})
		return vzSentinelPID, "vz-task", core.JoinPath(ws, ".meta", "agent.log"), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	_ = s.DispatchSync(ctx, DispatchSyncInput{Repo: "go-io", Org: "core", Agent: "codex", Task: "Fix", Branch: "x"})

	// The status the sync caller wrote carries the full record + the VZ tag.
	updated := mustReadStatus(t, workspaceDir)
	core.AssertEqual(t, "go-io", updated.Repo)
	core.AssertEqual(t, "agent/vz", updated.Branch) // from prep output
	core.AssertEqual(t, "core", updated.Org)
	core.AssertEqual(t, vzRuntimeName, updated.Runtime)
}
