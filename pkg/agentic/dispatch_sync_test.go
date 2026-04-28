// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestDispatchsync_ContainerCommand_Good(t *testing.T) {
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

	s.dispatchSyncPrep = func(ctx context.Context, _ *mcp.CallToolRequest, input PrepInput) (*mcp.CallToolResult, PrepOutput, error) {
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
	s.dispatchSyncPrep = func(context.Context, *mcp.CallToolRequest, PrepInput) (*mcp.CallToolResult, PrepOutput, error) {
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
	s.dispatchSyncPrep = func(context.Context, *mcp.CallToolRequest, PrepInput) (*mcp.CallToolResult, PrepOutput, error) {
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

	s.dispatchSyncPrep = func(context.Context, *mcp.CallToolRequest, PrepInput) (*mcp.CallToolResult, PrepOutput, error) {
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
