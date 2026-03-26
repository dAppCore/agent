// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- resume ---

func TestResume_Resume_Good(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	wsRoot := WorkspaceRoot()
	ws := core.JoinPath(wsRoot, "ws-blocked")
	repoDir := core.JoinPath(ws, "repo")
	fs.EnsureDir(repoDir)
	testCore.Process().Run(context.Background(), "git", "init", repoDir)

	st := &WorkspaceStatus{Status: "blocked", Repo: "go-io", Agent: "codex", Task: "Fix the tests"}
	fs.Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(st))

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	_, out, err := s.resume(context.Background(), nil, ResumeInput{
		Workspace: "ws-blocked", Answer: "Use the new Core API", DryRun: true,
	})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Equal(t, "ws-blocked", out.Workspace)
	assert.Equal(t, "codex", out.Agent)
	assert.Contains(t, out.Prompt, "Fix the tests")
	assert.Contains(t, out.Prompt, "Use the new Core API")

	answerR := fs.Read(core.JoinPath(repoDir, "ANSWER.md"))
	assert.Contains(t, answerR.Value.(string), "Use the new Core API")

	// Agent override
	_, out2, _ := s.resume(context.Background(), nil, ResumeInput{
		Workspace: "ws-blocked", Agent: "claude:opus", DryRun: true,
	})
	assert.Equal(t, "claude:opus", out2.Agent)

	// Completed workspace is resumable too
	ws2 := core.JoinPath(wsRoot, "ws-done")
	fs.EnsureDir(core.JoinPath(ws2, "repo"))
	testCore.Process().Run(context.Background(), "git", "init", core.JoinPath(ws2, "repo"))
	st2 := &WorkspaceStatus{Status: "completed", Repo: "go-io", Agent: "codex", Task: "Review code"}
	fs.Write(core.JoinPath(ws2, "status.json"), core.JSONMarshalString(st2))

	_, out3, err3 := s.resume(context.Background(), nil, ResumeInput{Workspace: "ws-done", DryRun: true})
	require.NoError(t, err3)
	assert.True(t, out3.Success)
}

func TestResume_Resume_Bad(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), backoff: make(map[string]time.Time), failCount: make(map[string]int)}

	// Empty workspace
	_, _, err := s.resume(context.Background(), nil, ResumeInput{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workspace is required")

	// Workspace not found
	_, _, err = s.resume(context.Background(), nil, ResumeInput{Workspace: "nonexistent"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workspace not found")

	// Not resumable (running)
	ws := core.JoinPath(WorkspaceRoot(), "ws-running")
	fs.EnsureDir(core.JoinPath(ws, "repo"))
	testCore.Process().Run(context.Background(), "git", "init", core.JoinPath(ws, "repo"))
	st := &WorkspaceStatus{Status: "running", Repo: "test", Agent: "codex"}
	fs.Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(st))

	_, _, err = s.resume(context.Background(), nil, ResumeInput{Workspace: "ws-running"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not resumable")
}

func TestResume_Resume_Ugly(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	// Workspace exists but no status.json
	ws := core.JoinPath(WorkspaceRoot(), "ws-nostatus")
	fs.EnsureDir(core.JoinPath(ws, "repo"))
	testCore.Process().Run(context.Background(), "git", "init", core.JoinPath(ws, "repo"))

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	_, _, err := s.resume(context.Background(), nil, ResumeInput{Workspace: "ws-nostatus"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no status.json")

	// No answer provided — prompt has no ANSWER section
	ws2 := core.JoinPath(WorkspaceRoot(), "ws-noanswer")
	fs.EnsureDir(core.JoinPath(ws2, "repo"))
	testCore.Process().Run(context.Background(), "git", "init", core.JoinPath(ws2, "repo"))
	st := &WorkspaceStatus{Status: "blocked", Repo: "test", Agent: "codex", Task: "Fix"}
	fs.Write(core.JoinPath(ws2, "status.json"), core.JSONMarshalString(st))

	_, out, err := s.resume(context.Background(), nil, ResumeInput{Workspace: "ws-noanswer", DryRun: true})
	require.NoError(t, err)
	assert.NotContains(t, out.Prompt, "ANSWER TO YOUR QUESTION")
}
