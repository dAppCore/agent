// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go"
)

// --- resume ---

func TestResume_Resume_Good_Case(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	wsRoot := WorkspaceRoot()
	ws := core.JoinPath(wsRoot, "ws-blocked")
	repoDir := core.JoinPath(ws, "repo")
	fs.EnsureDir(repoDir)
	testCore.Process().Run(context.Background(), "git", "init", repoDir)

	st := &WorkspaceStatus{Status: "blocked", Repo: "go-io", Agent: "codex", Task: "Fix the tests"}
	fs.Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(st))

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	result := s.resume(context.Background(), ResumeInput{
		Workspace: "ws-blocked", Answer: "Use the new Core API", DryRun: true,
	})
	core.RequireTrue(t, result.OK)
	out, ok := result.Value.(ResumeOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, out.Success)
	core.AssertEqual(t, "ws-blocked", out.Workspace)
	core.AssertEqual(t, "codex", out.Agent)
	core.AssertContains(t, out.Prompt, "Fix the tests")
	core.AssertContains(t, out.Prompt, "Use the new Core API")

	answerR := fs.Read(core.JoinPath(repoDir, "ANSWER.md"))
	core.AssertContains(t, answerR.Value.(string), "Use the new Core API")

	// Agent override
	result = s.resume(context.Background(), ResumeInput{
		Workspace: "ws-blocked", Agent: "claude:opus", DryRun: true,
	})
	out2, ok := result.Value.(ResumeOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "claude:opus", out2.Agent)

	// Completed workspace is resumable too
	ws2 := core.JoinPath(wsRoot, "ws-done")
	fs.EnsureDir(core.JoinPath(ws2, "repo"))
	testCore.Process().Run(context.Background(), "git", "init", core.JoinPath(ws2, "repo"))
	st2 := &WorkspaceStatus{Status: "completed", Repo: "go-io", Agent: "codex", Task: "Review code"}
	fs.Write(core.JoinPath(ws2, "status.json"), core.JSONMarshalString(st2))

	result = s.resume(context.Background(), ResumeInput{Workspace: "ws-done", DryRun: true})
	core.RequireTrue(t, result.OK)
	out3, ok := result.Value.(ResumeOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, out3.Success)
}

func TestResume_Resume_Bad_Case(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), backoff: make(map[string]time.Time), failCount: make(map[string]int)}

	// Empty workspace
	result := s.resume(context.Background(), ResumeInput{})
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "workspace is required")

	// Workspace not found
	result = s.resume(context.Background(), ResumeInput{Workspace: "nonexistent"})
	err, ok = result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "workspace not found")

	// Not resumable (running)
	ws := core.JoinPath(WorkspaceRoot(), "ws-running")
	fs.EnsureDir(core.JoinPath(ws, "repo"))
	testCore.Process().Run(context.Background(), "git", "init", core.JoinPath(ws, "repo"))
	st := &WorkspaceStatus{Status: "running", Repo: "test", Agent: "codex"}
	fs.Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(st))

	result = s.resume(context.Background(), ResumeInput{Workspace: "ws-running"})
	err, ok = result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "not resumable")
}

func TestResume_Resume_Ugly_Case(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	// Workspace exists but no status.json
	ws := core.JoinPath(WorkspaceRoot(), "ws-nostatus")
	fs.EnsureDir(core.JoinPath(ws, "repo"))
	testCore.Process().Run(context.Background(), "git", "init", core.JoinPath(ws, "repo"))

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	result := s.resume(context.Background(), ResumeInput{Workspace: "ws-nostatus"})
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "no status.json")

	// No answer provided — prompt has no ANSWER section
	ws2 := core.JoinPath(WorkspaceRoot(), "ws-noanswer")
	fs.EnsureDir(core.JoinPath(ws2, "repo"))
	testCore.Process().Run(context.Background(), "git", "init", core.JoinPath(ws2, "repo"))
	st := &WorkspaceStatus{Status: "blocked", Repo: "test", Agent: "codex", Task: "Fix"}
	fs.Write(core.JoinPath(ws2, "status.json"), core.JSONMarshalString(st))

	result = s.resume(context.Background(), ResumeInput{Workspace: "ws-noanswer", DryRun: true})
	core.RequireTrue(t, result.OK)
	out, ok := result.Value.(ResumeOutput)
	core.RequireTrue(t, ok)
	core.AssertNotContains(t, out.Prompt, "ANSWER TO YOUR QUESTION")
}
