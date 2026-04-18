// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDispatchsync_ContainerCommand_Good(t *testing.T) {
	cmd, args := containerCommand("codex", []string{"--model", "gpt-5.4"}, "/workspace/task-5", "/workspace/task-5/.meta")
	assert.Equal(t, "docker", cmd)
	assert.Contains(t, args, "run")
	assert.Contains(t, args, "/workspace/task-5:/workspace")
	assert.Contains(t, args, "/workspace/task-5/.meta:/workspace/.meta")
	assert.Contains(t, args, "/workspace/repo")
}

func TestDispatchsync_ContainerCommand_Bad_UnknownAgent(t *testing.T) {
	cmd, args := containerCommand("unknown", nil, "/workspace/task-5", "/workspace/task-5/.meta")
	assert.Equal(t, "docker", cmd)
	assert.NotEmpty(t, args)
}

func TestDispatchsync_ContainerCommand_Ugly_EmptyArgs(t *testing.T) {
	assert.NotPanics(t, func() {
		containerCommand("codex", nil, "", "")
	})
}

func TestDispatchsync_HandleDispatchSync_Good_Completed(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	workspaceDir := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-7")
	s := &PrepSubsystem{dispatchSyncTick: 10 * time.Millisecond}

	s.dispatchSyncPrep = func(ctx context.Context, _ *mcp.CallToolRequest, input PrepInput) (*mcp.CallToolResult, PrepOutput, error) {
		require.Equal(t, "core", input.Org)
		require.Equal(t, "go-io", input.Repo)
		require.Equal(t, "codex", input.Agent)
		require.Equal(t, "Fix tests", input.Task)
		require.Equal(t, 7, input.Issue)

		require.True(t, fs.EnsureDir(workspaceDir).OK)
		require.True(t, fs.Write(core.JoinPath(workspaceDir, "status.json"), core.JSONMarshalString(&WorkspaceStatus{
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
		require.Equal(t, "codex", agent)
		require.Equal(t, "prompt", prompt)
		require.Equal(t, workspaceDir, dir)
		return 321, "process-321", core.JoinPath(dir, ".meta", "agent.log"), nil
	}

	result := s.handleDispatchSync(context.Background(), core.NewOptions(
		core.Option{Key: "org", Value: "core"},
		core.Option{Key: "repo", Value: "go-io"},
		core.Option{Key: "agent", Value: "codex"},
		core.Option{Key: "task", Value: "Fix tests"},
		core.Option{Key: "issue", Value: "7"},
	))

	require.True(t, result.OK)
	output, ok := result.Value.(DispatchSyncResult)
	require.True(t, ok)
	assert.True(t, output.OK)
	assert.Equal(t, "completed", output.Status)
	assert.Equal(t, "https://forge.test/core/go-io/pulls/7", output.PRURL)
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

	assert.False(t, result.OK)
	require.Error(t, result.Value.(error))
	assert.Contains(t, result.Value.(error).Error(), "prep workspace failed")
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

	assert.False(t, result.OK)
	require.Error(t, result.Value.(error))
	assert.Contains(t, result.Value.(error).Error(), "prep failed")
}

func TestDispatchsync_HandleDispatchSync_Ugly_SpawnFailure(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	workspaceDir := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-7")
	s := &PrepSubsystem{dispatchSyncTick: 10 * time.Millisecond}

	s.dispatchSyncPrep = func(context.Context, *mcp.CallToolRequest, PrepInput) (*mcp.CallToolResult, PrepOutput, error) {
		require.True(t, fs.EnsureDir(workspaceDir).OK)
		require.True(t, fs.Write(core.JoinPath(workspaceDir, "status.json"), core.JSONMarshalString(&WorkspaceStatus{
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
		require.Equal(t, "codex", agent)
		return 0, "", "", core.E("spawn", "boom", nil)
	}

	result := s.handleDispatchSync(context.Background(), core.NewOptions(
		core.Option{Key: "repo", Value: "go-io"},
		core.Option{Key: "agent", Value: "codex"},
		core.Option{Key: "task", Value: "Fix tests"},
	))

	assert.False(t, result.OK)
	require.Error(t, result.Value.(error))
	assert.Contains(t, result.Value.(error).Error(), "spawn agent failed")
}
