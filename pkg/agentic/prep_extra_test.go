// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	core "dappco.re/go/core"
	"dappco.re/go/core/forge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Shutdown ---

func TestPrep_Shutdown_Good(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	err := s.Shutdown(context.Background())
	assert.NoError(t, err)
}

// --- Name ---

func TestPrep_Name_Good(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	assert.Equal(t, "agentic", s.Name())
}

// --- findConsumersList ---

func TestPrep_FindConsumersList_Good_HasConsumers(t *testing.T) {
	dir := t.TempDir()

	// Create go.work
	goWork := `go 1.22

use (
	./core/go
	./core/agent
	./core/mcp
)`
	fs.Write(core.JoinPath(dir, "go.work"), goWork)

	// Create module dirs with go.mod
	for _, mod := range []struct {
		path    string
		content string
	}{
		{"core/go", "module dappco.re/go/core/go\n\ngo 1.22\n"},
		{"core/agent", "module dappco.re/go/core/agent\n\nrequire dappco.re/go/core/go v0.7.0\n"},
		{"core/mcp", "module dappco.re/go/core/mcp\n\nrequire dappco.re/go/core/go v0.7.0\n"},
	} {
		modDir := core.JoinPath(dir, mod.path)
		fs.EnsureDir(modDir)
		fs.Write(core.JoinPath(modDir, "go.mod"), mod.content)
	}

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       dir,
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	list, count := s.findConsumersList("go")
	assert.Equal(t, 2, count)
	assert.Contains(t, list, "agent")
	assert.Contains(t, list, "mcp")
	assert.Contains(t, list, "Breaking change risk")
}

func TestPrep_FindConsumersList_Good_NoConsumers(t *testing.T) {
	dir := t.TempDir()

	goWork := `go 1.22

use (
	./core/go
)`
	fs.Write(core.JoinPath(dir, "go.work"), goWork)

	modDir := core.JoinPath(dir, "core", "go")
	fs.EnsureDir(modDir)
	fs.Write(core.JoinPath(modDir, "go.mod"), "module forge.lthn.ai/core/go\n")

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       dir,
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	list, count := s.findConsumersList("go")
	assert.Equal(t, 0, count)
	assert.Empty(t, list)
}

func TestPrep_FindConsumersList_Bad_NoGoWork(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	list, count := s.findConsumersList("go")
	assert.Equal(t, 0, count)
	assert.Empty(t, list)
}

// --- copyRepoSpecs ---

func TestPrep_CopyRepoSpecs_Good(t *testing.T) {
	root := t.TempDir()
	codePath := core.JoinPath(root, "src")
	plansBase := core.JoinPath(codePath, "host-uk", "core", "plans")
	specDir := core.JoinPath(plansBase, "core", "go", "test-repo")

	require.True(t, fs.EnsureDir(core.JoinPath(specDir, "sub")).OK)
	require.True(t, fs.Write(core.JoinPath(specDir, "RFC.md"), "root-spec").OK)
	require.True(t, fs.Write(core.JoinPath(specDir, "sub", "RFC-2.md"), "nested-spec").OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       codePath,
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	require.NoError(t, s.copyRepoSpecs(core.JoinPath(root, "workspace"), "go-test-repo"))

	rootCopy := fs.Read(core.JoinPath(root, "workspace", "specs", "RFC.md"))
	require.True(t, rootCopy.OK)
	assert.Equal(t, "root-spec", rootCopy.Value.(string))

	nestedCopy := fs.Read(core.JoinPath(root, "workspace", "specs", "sub", "RFC-2.md"))
	require.True(t, nestedCopy.OK)
	assert.Equal(t, "nested-spec", nestedCopy.Value.(string))
}

func TestPrep_CopyRepoSpecs_Bad_SpecsDirBlocked(t *testing.T) {
	root := t.TempDir()
	codePath := core.JoinPath(root, "src")
	plansBase := core.JoinPath(codePath, "host-uk", "core", "plans")
	specDir := core.JoinPath(plansBase, "core", "go", "test-repo")

	require.True(t, fs.EnsureDir(specDir).OK)
	require.True(t, fs.Write(core.JoinPath(specDir, "RFC.md"), "root-spec").OK)
	require.True(t, fs.Write(core.JoinPath(root, "workspace", "specs"), "blocked").OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       codePath,
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	err := s.copyRepoSpecs(core.JoinPath(root, "workspace"), "go-test-repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create specs dir")
}

func TestPrep_CopyRepoSpecs_Ugly_ParentDirBlocked(t *testing.T) {
	root := t.TempDir()
	codePath := core.JoinPath(root, "src")
	plansBase := core.JoinPath(codePath, "host-uk", "core", "plans")
	specDir := core.JoinPath(plansBase, "core", "go", "test-repo")

	require.True(t, fs.EnsureDir(core.JoinPath(specDir, "sub")).OK)
	require.True(t, fs.Write(core.JoinPath(specDir, "sub", "RFC-2.md"), "nested-spec").OK)
	require.True(t, fs.EnsureDir(core.JoinPath(root, "workspace", "specs")).OK)
	require.True(t, fs.Write(core.JoinPath(root, "workspace", "specs", "sub"), "blocked").OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       codePath,
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	err := s.copyRepoSpecs(core.JoinPath(root, "workspace"), "go-test-repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create specs parent dir")
}

func writeFakeLanguageCommand(t *testing.T, dir, name, logPath string, exitCode int) {
	t.Helper()

	script := core.Concat(
		"#!/bin/sh\n",
		"printf '%s %s\\n' '",
		name,
		"' \"$*\" >> '",
		logPath,
		"'\n",
		"exit ",
		core.Sprint(exitCode),
		"\n",
	)
	require.True(t, fs.WriteMode(core.JoinPath(dir, name), script, 0755).OK)
}

// --- runWorkspaceLanguagePrep ---

func TestPrep_RunWorkspaceLanguagePrep_Good_Polyglot(t *testing.T) {
	root := t.TempDir()
	binDir := core.JoinPath(root, "bin")
	require.True(t, fs.EnsureDir(binDir).OK)

	logPath := core.JoinPath(root, "commands.log")
	writeFakeLanguageCommand(t, binDir, "go", logPath, 0)
	writeFakeLanguageCommand(t, binDir, "composer", logPath, 0)
	writeFakeLanguageCommand(t, binDir, "npm", logPath, 0)

	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", core.Concat(binDir, ":", oldPath))

	workspaceDir := core.JoinPath(root, "workspace")
	repoDir := core.JoinPath(root, "repo")
	require.True(t, fs.EnsureDir(workspaceDir).OK)
	require.True(t, fs.EnsureDir(repoDir).OK)
	require.True(t, fs.Write(core.JoinPath(repoDir, "go.mod"), "module example.com/test\n").OK)
	require.True(t, fs.Write(core.JoinPath(repoDir, "composer.json"), "{}").OK)
	require.True(t, fs.Write(core.JoinPath(repoDir, "package.json"), "{}").OK)
	require.True(t, fs.Write(core.JoinPath(workspaceDir, "go.work"), "go 1.22\n").OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	err := s.runWorkspaceLanguagePrep(context.Background(), workspaceDir, repoDir)
	require.NoError(t, err)

	logResult := fs.Read(logPath)
	require.True(t, logResult.OK)
	assert.Equal(t, "go mod download\ngo work sync\ncomposer install\nnpm install\n", logResult.Value.(string))
}

func TestPrep_RunWorkspaceLanguagePrep_Bad_NoLanguageManifests(t *testing.T) {
	root := t.TempDir()
	binDir := core.JoinPath(root, "bin")
	require.True(t, fs.EnsureDir(binDir).OK)

	logPath := core.JoinPath(root, "commands.log")
	writeFakeLanguageCommand(t, binDir, "go", logPath, 0)
	writeFakeLanguageCommand(t, binDir, "composer", logPath, 0)
	writeFakeLanguageCommand(t, binDir, "npm", logPath, 0)

	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", core.Concat(binDir, ":", oldPath))

	workspaceDir := core.JoinPath(root, "workspace")
	repoDir := core.JoinPath(root, "repo")
	require.True(t, fs.EnsureDir(workspaceDir).OK)
	require.True(t, fs.EnsureDir(repoDir).OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	err := s.runWorkspaceLanguagePrep(context.Background(), workspaceDir, repoDir)
	require.NoError(t, err)

	logResult := fs.Read(logPath)
	assert.False(t, logResult.OK)
}

func TestPrep_RunWorkspaceLanguagePrep_Ugly_CommandFailure(t *testing.T) {
	root := t.TempDir()
	binDir := core.JoinPath(root, "bin")
	require.True(t, fs.EnsureDir(binDir).OK)

	logPath := core.JoinPath(root, "commands.log")
	writeFakeLanguageCommand(t, binDir, "go", logPath, 0)
	writeFakeLanguageCommand(t, binDir, "composer", logPath, 1)
	writeFakeLanguageCommand(t, binDir, "npm", logPath, 0)

	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", core.Concat(binDir, ":", oldPath))

	workspaceDir := core.JoinPath(root, "workspace")
	repoDir := core.JoinPath(root, "repo")
	require.True(t, fs.EnsureDir(workspaceDir).OK)
	require.True(t, fs.EnsureDir(repoDir).OK)
	require.True(t, fs.Write(core.JoinPath(repoDir, "composer.json"), "{}").OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	err := s.runWorkspaceLanguagePrep(context.Background(), workspaceDir, repoDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "composer install failed")
}

// --- pullWikiContent ---

func TestPrep_PullWikiContent_Good_WithPages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/repos/core/go-io/wiki/pages":
			w.Write([]byte(core.JSONMarshalString([]map[string]any{
				{"title": "Home", "sub_url": "Home"},
				{"title": "Architecture", "sub_url": "Architecture"},
			})))
		case r.URL.Path == "/api/v1/repos/core/go-io/wiki/page/Home":
			// "Hello World" base64
			w.Write([]byte(core.JSONMarshalString(map[string]any{
				"title":          "Home",
				"content_base64": "SGVsbG8gV29ybGQ=",
			})))
		case r.URL.Path == "/api/v1/repos/core/go-io/wiki/page/Architecture":
			w.Write([]byte(core.JSONMarshalString(map[string]any{
				"title":          "Architecture",
				"content_base64": "TGF5ZXJlZA==",
			})))
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge(srv.URL, "test-token"),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	content := s.pullWikiContent(context.Background(), "core", "go-io")
	assert.Contains(t, content, "Hello World")
	assert.Contains(t, content, "Layered")
	assert.Contains(t, content, "### Home")
	assert.Contains(t, content, "### Architecture")
}

func TestPrep_PullWikiContent_Good_NoPages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString([]map[string]any{})))
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge(srv.URL, "test-token"),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	content := s.pullWikiContent(context.Background(), "core", "go-io")
	assert.Empty(t, content)
}

// --- getIssueBody ---

func TestPrep_GetIssueBody_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString(map[string]any{
			"number": 15,
			"title":  "Fix tests",
			"body":   "The tests are broken in pkg/core",
		})))
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge(srv.URL, "test-token"),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	body := s.getIssueBody(context.Background(), "core", "go-io", 15)
	assert.Contains(t, body, "tests are broken")
}

func TestPrep_GetIssueBody_Bad_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge(srv.URL, "test-token"),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	body := s.getIssueBody(context.Background(), "core", "go-io", 999)
	assert.Empty(t, body)
}

// --- buildPrompt ---

func TestPrep_BuildPrompt_Good_BasicFields(t *testing.T) {
	dir := t.TempDir()
	// Create go.mod to detect language
	fs.Write(core.JoinPath(dir, "go.mod"), "module test\n\ngo 1.22\n")

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	prompt, memories, consumers := s.buildPrompt(context.Background(), PrepInput{
		Task: "Fix the tests",
		Org:  "core",
		Repo: "go-io",
	}, "dev", dir)

	assert.Contains(t, prompt, "TASK: Fix the tests")
	assert.Contains(t, prompt, "REPO: core/go-io on branch dev")
	assert.Contains(t, prompt, "LANGUAGE: go")
	assert.Contains(t, prompt, "CONSTRAINTS:")
	assert.Contains(t, prompt, "CODEX.md")
	assert.Equal(t, 0, memories)
	assert.Equal(t, 0, consumers)
}

func TestPrep_TestBuildPrompt_Good_BasicFields(t *testing.T) {
	dir := t.TempDir()
	fs.Write(core.JoinPath(dir, "go.mod"), "module test\n\ngo 1.22\n")

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	prompt, memories, consumers := s.TestBuildPrompt(context.Background(), PrepInput{
		Task: "Fix the tests",
		Org:  "core",
		Repo: "go-io",
	}, "dev", dir)

	assert.Contains(t, prompt, "TASK: Fix the tests")
	assert.Contains(t, prompt, "REPO: core/go-io on branch dev")
	assert.Contains(t, prompt, "LANGUAGE: go")
	assert.Contains(t, prompt, "CONSTRAINTS:")
	assert.Equal(t, 0, memories)
	assert.Equal(t, 0, consumers)
}

func TestPrep_BuildPrompt_Good_WithIssue(t *testing.T) {
	dir := t.TempDir()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString(map[string]any{
			"number": 42,
			"title":  "Bug report",
			"body":   "Steps to reproduce the bug",
		})))
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge(srv.URL, "test-token"),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	prompt, _, _ := s.buildPrompt(context.Background(), PrepInput{
		Task:  "Fix the bug",
		Org:   "core",
		Repo:  "go-io",
		Issue: 42,
	}, "dev", dir)

	assert.Contains(t, prompt, "ISSUE:")
	assert.Contains(t, prompt, "Steps to reproduce")
}

func TestPrep_TestBuildPrompt_Bad_EmptyRepoPath(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	prompt, memories, consumers := s.TestBuildPrompt(context.Background(), PrepInput{
		Task: "Add unit tests",
		Org:  "core",
		Repo: "go-io",
	}, "dev", "")

	assert.Contains(t, prompt, "TASK: Add unit tests")
	assert.Equal(t, 0, memories)
	assert.Equal(t, 0, consumers)
}

func TestPrep_TestBuildPrompt_Ugly_StillBuildsPrompt(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	prompt, memories, consumers := s.TestBuildPrompt(context.Background(), PrepInput{}, "", "")

	assert.Contains(t, prompt, "TASK:")
	assert.Equal(t, 0, memories)
	assert.Equal(t, 0, consumers)
}

// --- buildPrompt (naming convention tests) ---

func TestPrep_BuildPromptNaming_Good(t *testing.T) {
	dir := t.TempDir()
	// Create go.mod to detect language as "go"
	fs.Write(core.JoinPath(dir, "go.mod"), "module test\n\ngo 1.22\n")

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	prompt, memories, consumers := s.buildPrompt(context.Background(), PrepInput{
		Task: "Add unit tests",
		Org:  "core",
		Repo: "go-io",
	}, "dev", dir)

	assert.Contains(t, prompt, "TASK: Add unit tests")
	assert.Contains(t, prompt, "REPO: core/go-io on branch dev")
	assert.Contains(t, prompt, "LANGUAGE: go")
	assert.Contains(t, prompt, "BUILD: go build ./...")
	assert.Contains(t, prompt, "TEST: go test ./...")
	assert.Contains(t, prompt, "CONSTRAINTS:")
	assert.Equal(t, 0, memories)
	assert.Equal(t, 0, consumers)
}

func TestPrep_BuildPromptNaming_Bad(t *testing.T) {
	// Empty repo path — still produces a prompt (no crash)
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	prompt, memories, consumers := s.buildPrompt(context.Background(), PrepInput{
		Task: "Do something",
		Org:  "core",
		Repo: "go-io",
	}, "main", "")

	assert.Contains(t, prompt, "TASK: Do something")
	assert.Contains(t, prompt, "CONSTRAINTS:")
	assert.Equal(t, 0, memories)
	assert.Equal(t, 0, consumers)
}

func TestPrep_BuildPromptNaming_Ugly(t *testing.T) {
	dir := t.TempDir()
	fs.Write(core.JoinPath(dir, "go.mod"), "module test\n\ngo 1.22\n")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString(map[string]any{
			"number": 99,
			"title":  "Critical bug",
			"body":   "Server crashes on startup",
		})))
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge(srv.URL, "test-token"),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	prompt, _, _ := s.buildPrompt(context.Background(), PrepInput{
		Task:         "Fix critical bug",
		Org:          "core",
		Repo:         "go-io",
		Persona:      "reviewer",
		PlanTemplate: "nonexistent-plan",
		Issue:        99,
	}, "agent/fix-bug", dir)

	// Persona may or may not resolve, but prompt must still contain core fields
	assert.Contains(t, prompt, "TASK: Fix critical bug")
	assert.Contains(t, prompt, "REPO: core/go-io on branch agent/fix-bug")
	assert.Contains(t, prompt, "ISSUE:")
	assert.Contains(t, prompt, "Server crashes on startup")
	assert.Contains(t, prompt, "CONSTRAINTS:")
}

func TestPrep_BuildPrompt_Ugly_WithGitLog(t *testing.T) {
	dir := t.TempDir()
	fs.Write(core.JoinPath(dir, "go.mod"), "module test\n\ngo 1.22\n")

	// Init a real git repo with commits so git log path is covered
	testCore.Process().Run(context.Background(), "git", "init", "-b", "main", dir)
	testCore.Process().RunIn(context.Background(), dir, "git", "config", "user.email", "t@t.com")
	testCore.Process().RunIn(context.Background(), dir, "git", "config", "user.name", "T")
	testCore.Process().RunIn(context.Background(), dir, "git", "add", ".")
	testCore.Process().RunIn(context.Background(), dir, "git", "commit", "-m", "init")

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	prompt, _, _ := s.buildPrompt(context.Background(), PrepInput{
		Task: "Fix tests", Org: "core", Repo: "go-io",
	}, "dev", dir)

	assert.Contains(t, prompt, "RECENT CHANGES:")
	assert.Contains(t, prompt, "init")
}

// --- runQA ---

func TestDispatch_RunQA_Good_PHPNoComposer(t *testing.T) {
	dir := t.TempDir()
	repoDir := core.JoinPath(dir, "repo")
	fs.EnsureDir(repoDir)
	// composer.json present but no composer binary
	fs.Write(core.JoinPath(repoDir, "composer.json"), `{"name":"test"}`)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	// Will fail (composer not found) — that's the expected path
	result := s.runQA(dir)
	assert.False(t, result)
}

// --- pullWikiContent Bad/Ugly ---

func TestPrep_PullWikiContent_Bad(t *testing.T) {
	// Forge returns error on wiki pages
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge(srv.URL, "test-token"),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	content := s.pullWikiContent(context.Background(), "core", "go-io")
	assert.Empty(t, content)
}

func TestPrep_PullWikiContent_Ugly(t *testing.T) {
	// Forge returns pages with empty content
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/repos/core/go-io/wiki/pages":
			w.Write([]byte(core.JSONMarshalString([]map[string]any{
				{"title": "EmptyPage", "sub_url": "EmptyPage"},
			})))
		case r.URL.Path == "/api/v1/repos/core/go-io/wiki/page/EmptyPage":
			w.Write([]byte(core.JSONMarshalString(map[string]any{
				"title":          "EmptyPage",
				"content_base64": "",
			})))
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge(srv.URL, "test-token"),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	content := s.pullWikiContent(context.Background(), "core", "go-io")
	// Empty content_base64 means the page is skipped
	assert.Empty(t, content)
}

// --- renderPlan Ugly ---

func TestPrep_RenderPlan_Ugly(t *testing.T) {
	// Template with variables that don't exist in template — variables just won't match
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Passing variables that won't match any {{placeholder}} in the template
	result := s.renderPlan("coding", map[string]string{
		"nonexistent_var": "value1",
		"another_missing": "value2",
	}, "Test task")
	// Should return the template rendered without substitution (if template exists)
	// or empty if template doesn't exist. Either way, no panic.
	_ = result
}

// --- brainRecall Ugly ---

func TestPrep_BrainRecall_Ugly(t *testing.T) {
	// Server returns unexpected JSON structure
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		// Return JSON that doesn't have "memories" key
		w.Write([]byte(core.JSONMarshalString(map[string]any{
			"unexpected_key": "unexpected_value",
			"data":           []string{"not", "the", "right", "shape"},
		})))
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:       srv.URL,
		brainKey:       "test-key",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	recall, count := s.brainRecall(context.Background(), "go-io")
	assert.Empty(t, recall, "unexpected JSON structure should yield no memories")
	assert.Equal(t, 0, count)
}

// --- PrepWorkspace Ugly ---

func TestPrep_PrepWorkspace_Ugly(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Repo name "." — should be rejected as invalid
	_, _, err := s.prepWorkspace(context.Background(), nil, PrepInput{
		Repo:  ".",
		Issue: 1,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid repo name")

	// Repo name ".." — should be rejected as invalid
	_, _, err2 := s.prepWorkspace(context.Background(), nil, PrepInput{
		Repo:  "..",
		Issue: 1,
	})
	assert.Error(t, err2)
	assert.Contains(t, err2.Error(), "invalid repo name")
}

// --- findConsumersList Ugly ---

func TestPrep_FindConsumersList_Ugly(t *testing.T) {
	// go.work with modules that don't have go.mod
	dir := t.TempDir()

	goWork := "go 1.22\n\nuse (\n\t./core/go\n\t./core/missing\n)"
	fs.Write(core.JoinPath(dir, "go.work"), goWork)

	// Create only the first module dir with go.mod
	modDir := core.JoinPath(dir, "core", "go")
	fs.EnsureDir(modDir)
	fs.Write(core.JoinPath(modDir, "go.mod"), "module forge.lthn.ai/core/go\n")

	// core/missing has no go.mod
	fs.EnsureDir(core.JoinPath(dir, "core", "missing"))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       dir,
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Should not panic, just skip missing go.mod entries
	list, count := s.findConsumersList("go")
	assert.Equal(t, 0, count)
	assert.Empty(t, list)
}

// --- getIssueBody Ugly ---

func TestPrep_GetIssueBody_Ugly(t *testing.T) {
	// Issue body with HTML/special chars
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString(map[string]any{
			"number": 99,
			"title":  "Issue with <script>alert('xss')</script>",
			"body":   "Body has &amp; HTML &lt;tags&gt; and \"quotes\" and 'apostrophes' <b>bold</b>",
		})))
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge(srv.URL, "test-token"),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	body := s.getIssueBody(context.Background(), "core", "go-io", 99)
	assert.NotEmpty(t, body)
	assert.Contains(t, body, "HTML")
	assert.Contains(t, body, "<script>")
}
