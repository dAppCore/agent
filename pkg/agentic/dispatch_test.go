// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"dappco.re/go/core/forge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- dispatch (validation) ---

func TestDispatch_Bad_NoRepo(t *testing.T) {
	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, _, err := s.dispatch(context.Background(), nil, DispatchInput{
		Task: "Fix the bug",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "repo is required")
}

func TestDispatch_Bad_NoTask(t *testing.T) {
	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, _, err := s.dispatch(context.Background(), nil, DispatchInput{
		Repo: "go-io",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task is required")
}

func TestDispatch_Good_DefaultsApplied(t *testing.T) {
	// We can't test full dispatch without Docker, but we can verify defaults
	// by using DryRun and checking the workspace prep
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	// Mock forge server for issue fetching
	forgeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"title": "Test issue",
			"body":  "Fix the thing",
		})
	}))
	t.Cleanup(forgeSrv.Close)

	// Create source repo to clone from
	srcRepo := filepath.Join(t.TempDir(), "core", "go-io")
	require.NoError(t, exec.Command("git", "init", "-b", "main", srcRepo).Run())
	gitCmd := exec.Command("git", "config", "user.name", "Test")
	gitCmd.Dir = srcRepo
	gitCmd.Run()
	gitCmd = exec.Command("git", "config", "user.email", "test@test.com")
	gitCmd.Dir = srcRepo
	gitCmd.Run()
	require.True(t, fs.Write(filepath.Join(srcRepo, "go.mod"), "module test\n\ngo 1.22").OK)
	gitCmd = exec.Command("git", "add", ".")
	gitCmd.Dir = srcRepo
	gitCmd.Run()
	gitCmd = exec.Command("git", "commit", "-m", "init")
	gitCmd.Dir = srcRepo
	gitCmd.Env = append(gitCmd.Environ(),
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@test.com",
	)
	gitCmd.Run()

	s := &PrepSubsystem{
		forge:     forge.NewForge(forgeSrv.URL, "test-token"),
		codePath:  filepath.Dir(filepath.Dir(srcRepo)), // parent of core/go-io
		client:    forgeSrv.Client(),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, out, err := s.dispatch(context.Background(), nil, DispatchInput{
		Repo:   "go-io",
		Task:   "Fix stuff",
		Issue:  42,
		DryRun: true,
	})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Equal(t, "codex", out.Agent) // default agent
	assert.Equal(t, "go-io", out.Repo)
	assert.NotEmpty(t, out.WorkspaceDir)
	assert.NotEmpty(t, out.Prompt)
}

// --- runQA ---

func TestRunQA_Good_GoProject(t *testing.T) {
	// Create a minimal valid Go project
	wsDir := t.TempDir()
	repoDir := filepath.Join(wsDir, "repo")
	require.True(t, fs.EnsureDir(repoDir).OK)

	require.True(t, fs.Write(filepath.Join(repoDir, "go.mod"), "module testmod\n\ngo 1.22\n").OK)
	require.True(t, fs.Write(filepath.Join(repoDir, "main.go"), "package main\n\nfunc main() {}\n").OK)

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	// go build, go vet, go test should all pass on this minimal project
	result := s.runQA(wsDir)
	assert.True(t, result)
}

func TestRunQA_Bad_GoBrokenCode(t *testing.T) {
	wsDir := t.TempDir()
	repoDir := filepath.Join(wsDir, "repo")
	require.True(t, fs.EnsureDir(repoDir).OK)

	require.True(t, fs.Write(filepath.Join(repoDir, "go.mod"), "module testmod\n\ngo 1.22\n").OK)
	// Deliberately broken Go code — won't compile
	require.True(t, fs.Write(filepath.Join(repoDir, "main.go"), "package main\n\nfunc main( {\n}\n").OK)

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	result := s.runQA(wsDir)
	assert.False(t, result)
}

func TestRunQA_Good_UnknownLanguage(t *testing.T) {
	// No go.mod, composer.json, or package.json → passes QA (no checks)
	wsDir := t.TempDir()
	repoDir := filepath.Join(wsDir, "repo")
	require.True(t, fs.EnsureDir(repoDir).OK)

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	result := s.runQA(wsDir)
	assert.True(t, result)
}

func TestRunQA_Good_GoVetFailure(t *testing.T) {
	wsDir := t.TempDir()
	repoDir := filepath.Join(wsDir, "repo")
	require.True(t, fs.EnsureDir(repoDir).OK)

	require.True(t, fs.Write(filepath.Join(repoDir, "go.mod"), "module testmod\n\ngo 1.22\n").OK)
	// Code that compiles but has a vet issue (unreachable code after return)
	code := `package main

import "fmt"

func main() {
	fmt.Printf("%d", "not a number")
}
`
	require.True(t, fs.Write(filepath.Join(repoDir, "main.go"), code).OK)

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	result := s.runQA(wsDir)
	// go vet should catch the Printf format mismatch
	assert.False(t, result)
}

// --- workspaceDir ---

func TestWorkspaceDir_Good_Issue(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	dir, err := workspaceDir("core", "go-io", PrepInput{Issue: 42})
	require.NoError(t, err)
	assert.Contains(t, dir, "task-42")
}

func TestWorkspaceDir_Good_PR(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	dir, err := workspaceDir("core", "go-io", PrepInput{PR: 7})
	require.NoError(t, err)
	assert.Contains(t, dir, "pr-7")
}

func TestWorkspaceDir_Good_Branch(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	dir, err := workspaceDir("core", "go-io", PrepInput{Branch: "feature/new-api"})
	require.NoError(t, err)
	assert.Contains(t, dir, "feature/new-api")
}

func TestWorkspaceDir_Good_Tag(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	dir, err := workspaceDir("core", "go-io", PrepInput{Tag: "v1.0.0"})
	require.NoError(t, err)
	assert.Contains(t, dir, "v1.0.0")
}

func TestWorkspaceDir_Bad_NoIdentifier(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	_, err := workspaceDir("core", "go-io", PrepInput{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "one of issue, pr, branch, or tag is required")
}

// --- DispatchInput defaults ---

func TestDispatchInput_Good_Defaults(t *testing.T) {
	input := DispatchInput{
		Repo: "go-io",
		Task: "Fix it",
	}
	// Verify default values are empty until dispatch applies them
	assert.Empty(t, input.Org)
	assert.Empty(t, input.Agent)
	assert.Empty(t, input.Template)
}

// --- buildPRBody ---

func TestBuildPRBody_Good_AllFields(t *testing.T) {
	s := &PrepSubsystem{}
	st := &WorkspaceStatus{
		Task:   "Implement new feature",
		Agent:  "claude",
		Issue:  15,
		Branch: "agent/implement-new-feature",
		Runs:   3,
	}
	body := s.buildPRBody(st)
	assert.Contains(t, body, "Implement new feature")
	assert.Contains(t, body, "Closes #15")
	assert.Contains(t, body, "**Agent:** claude")
	assert.Contains(t, body, "**Runs:** 3")
}

func TestBuildPRBody_Good_NoIssue(t *testing.T) {
	s := &PrepSubsystem{}
	st := &WorkspaceStatus{
		Task:  "Refactor internals",
		Agent: "codex",
		Runs:  1,
	}
	body := s.buildPRBody(st)
	assert.Contains(t, body, "Refactor internals")
	assert.NotContains(t, body, "Closes #")
}

func TestBuildPRBody_Bad_EmptyStatus(t *testing.T) {
	s := &PrepSubsystem{}
	st := &WorkspaceStatus{}
	body := s.buildPRBody(st)
	// Should still produce valid markdown, just with empty fields
	assert.Contains(t, body, "## Summary")
}

// --- canDispatchAgent ---

func TestCanDispatchAgent_Good_NoLimitsConfigured(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	s := &PrepSubsystem{
		codePath:  t.TempDir(),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	// No config, no running agents — should allow dispatch
	assert.True(t, s.canDispatchAgent("claude"))
}
