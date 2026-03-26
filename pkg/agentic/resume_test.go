// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
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
	ws := filepath.Join(wsRoot, "ws-blocked")
	repoDir := filepath.Join(ws, "repo")
	os.MkdirAll(repoDir, 0o755)
	exec.Command("git", "init", repoDir).Run()

	st := &WorkspaceStatus{Status: "blocked", Repo: "go-io", Agent: "codex", Task: "Fix the tests"}
	data, _ := json.Marshal(st)
	os.WriteFile(filepath.Join(ws, "status.json"), data, 0o644)

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

	answerContent, _ := os.ReadFile(filepath.Join(repoDir, "ANSWER.md"))
	assert.Contains(t, string(answerContent), "Use the new Core API")

	// Agent override
	_, out2, _ := s.resume(context.Background(), nil, ResumeInput{
		Workspace: "ws-blocked", Agent: "claude:opus", DryRun: true,
	})
	assert.Equal(t, "claude:opus", out2.Agent)

	// Completed workspace is resumable too
	ws2 := filepath.Join(wsRoot, "ws-done")
	os.MkdirAll(filepath.Join(ws2, "repo"), 0o755)
	exec.Command("git", "init", filepath.Join(ws2, "repo")).Run()
	st2 := &WorkspaceStatus{Status: "completed", Repo: "go-io", Agent: "codex", Task: "Review code"}
	data2, _ := json.Marshal(st2)
	os.WriteFile(filepath.Join(ws2, "status.json"), data2, 0o644)

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
	ws := filepath.Join(WorkspaceRoot(), "ws-running")
	os.MkdirAll(filepath.Join(ws, "repo"), 0o755)
	exec.Command("git", "init", filepath.Join(ws, "repo")).Run()
	st := &WorkspaceStatus{Status: "running", Repo: "test", Agent: "codex"}
	data, _ := json.Marshal(st)
	os.WriteFile(filepath.Join(ws, "status.json"), data, 0o644)

	_, _, err = s.resume(context.Background(), nil, ResumeInput{Workspace: "ws-running"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not resumable")
}

func TestResume_Resume_Ugly(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	// Workspace exists but no status.json
	ws := filepath.Join(WorkspaceRoot(), "ws-nostatus")
	os.MkdirAll(filepath.Join(ws, "repo"), 0o755)
	exec.Command("git", "init", filepath.Join(ws, "repo")).Run()

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	_, _, err := s.resume(context.Background(), nil, ResumeInput{Workspace: "ws-nostatus"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no status.json")

	// No answer provided — prompt has no ANSWER section
	ws2 := filepath.Join(WorkspaceRoot(), "ws-noanswer")
	os.MkdirAll(filepath.Join(ws2, "repo"), 0o755)
	exec.Command("git", "init", filepath.Join(ws2, "repo")).Run()
	st := &WorkspaceStatus{Status: "blocked", Repo: "test", Agent: "codex", Task: "Fix"}
	data, _ := json.Marshal(st)
	os.WriteFile(filepath.Join(ws2, "status.json"), data, 0o644)

	_, out, err := s.resume(context.Background(), nil, ResumeInput{Workspace: "ws-noanswer", DryRun: true})
	require.NoError(t, err)
	assert.NotContains(t, out.Prompt, "ANSWER TO YOUR QUESTION")
}
