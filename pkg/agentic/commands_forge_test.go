// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- parseForgeArgs ---

func TestCommandsforge_ParseForgeArgs_Good_AllFields(t *testing.T) {
	opts := core.NewOptions(
		core.Option{Key: "org", Value: "myorg"},
		core.Option{Key: "_arg", Value: "myrepo"},
		core.Option{Key: "number", Value: "42"},
	)
	org, repo, num := parseForgeArgs(opts)
	assert.Equal(t, "myorg", org)
	assert.Equal(t, "myrepo", repo)
	assert.Equal(t, int64(42), num)
}

func TestCommandsforge_ParseForgeArgs_Good_DefaultOrg(t *testing.T) {
	opts := core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
	)
	org, repo, num := parseForgeArgs(opts)
	assert.Equal(t, "core", org, "should default to 'core'")
	assert.Equal(t, "go-io", repo)
	assert.Equal(t, int64(0), num, "no number provided")
}

func TestCommandsforge_ParseForgeArgs_Bad_EmptyOpts(t *testing.T) {
	opts := core.NewOptions()
	org, repo, num := parseForgeArgs(opts)
	assert.Equal(t, "core", org, "should default to 'core'")
	assert.Empty(t, repo)
	assert.Equal(t, int64(0), num)
}

func TestCommandsforge_ParseForgeArgs_Bad_InvalidNumber(t *testing.T) {
	opts := core.NewOptions(
		core.Option{Key: "_arg", Value: "repo"},
		core.Option{Key: "number", Value: "not-a-number"},
	)
	_, _, num := parseForgeArgs(opts)
	assert.Equal(t, int64(0), num, "invalid number should parse as 0")
}

// --- formatIndex ---

func TestCommandsforge_FormatIndex_Good(t *testing.T) {
	assert.Equal(t, "1", formatIndex(1))
	assert.Equal(t, "42", formatIndex(42))
	assert.Equal(t, "0", formatIndex(0))
	assert.Equal(t, "999999", formatIndex(999999))
}

// --- parseForgeArgs Ugly ---

func TestCommandsforge_ParseForgeArgs_Ugly_OrgSetButNoRepo(t *testing.T) {
	opts := core.NewOptions(
		core.Option{Key: "org", Value: "custom-org"},
	)
	org, repo, num := parseForgeArgs(opts)
	assert.Equal(t, "custom-org", org)
	assert.Empty(t, repo, "repo should be empty when only org is set")
	assert.Equal(t, int64(0), num)
}

func TestCommandsforge_ParseForgeArgs_Ugly_NegativeNumber(t *testing.T) {
	opts := core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "number", Value: "-5"},
	)
	_, _, num := parseForgeArgs(opts)
	assert.Equal(t, int64(-5), num, "negative numbers parse but are semantically invalid")
}

func TestCommandsforge_ParseForgeArgs_Ugly_InvalidNames(t *testing.T) {
	opts := core.NewOptions(
		core.Option{Key: "org", Value: "bad/org"},
		core.Option{Key: "_arg", Value: "repo/with/slashes"},
	)
	org, repo, num := parseForgeArgs(opts)
	assert.Empty(t, org)
	assert.Empty(t, repo)
	assert.Equal(t, int64(0), num)
}

// --- formatIndex Bad/Ugly ---

func TestCommandsforge_FormatIndex_Bad_Negative(t *testing.T) {
	result := formatIndex(-1)
	assert.Equal(t, "-1", result, "negative should format as negative string")
}

func TestCommandsforge_FormatIndex_Ugly_VeryLarge(t *testing.T) {
	result := formatIndex(9999999999)
	assert.Equal(t, "9999999999", result)
}

func TestCommandsforge_FormatIndex_Ugly_MaxInt64(t *testing.T) {
	result := formatIndex(9223372036854775807) // math.MaxInt64
	assert.NotEmpty(t, result)
	assert.Equal(t, "9223372036854775807", result)
}

// --- Forge commands Ugly (special chars → API returns 404/error) ---

func TestCommandsforge_CmdIssueGet_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) }))
	t.Cleanup(srv.Close)
	s, _ := testPrepWithCore(t, srv)
	r := s.cmdIssueGet(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io/<script>"},
		core.Option{Key: "number", Value: "1"},
	))
	assert.False(t, r.OK)
}

func TestCommandsforge_CmdIssueList_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	t.Cleanup(srv.Close)
	s, _ := testPrepWithCore(t, srv)
	r := s.cmdIssueList(core.NewOptions(core.Option{Key: "_arg", Value: "repo&evil=true"}))
	assert.False(t, r.OK)
}

func TestCommandsforge_CmdIssueComment_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	t.Cleanup(srv.Close)
	s, _ := testPrepWithCore(t, srv)
	r := s.cmdIssueComment(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "number", Value: "1"},
		core.Option{Key: "body", Value: "Hello <b>world</b> & \"quotes\""},
	))
	assert.False(t, r.OK)
}

func TestCommandsforge_CmdIssueCreate_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	t.Cleanup(srv.Close)
	s, _ := testPrepWithCore(t, srv)
	r := s.cmdIssueCreate(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "title", Value: "Fix <b>bug</b> #123"},
	))
	assert.False(t, r.OK)
}

func TestCommandsforge_CmdIssueUpdate_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/issues/fix-auth", r.URL.Path)
		require.Equal(t, http.MethodPatch, r.Method)

		bodyResult := core.ReadAll(r.Body)
		require.True(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		require.True(t, parseResult.OK)
		require.Equal(t, "Fix auth middleware", payload["title"])
		require.Equal(t, "in_progress", payload["status"])

		_, _ = w.Write([]byte(`{"data":{"issue":{"slug":"fix-auth","title":"Fix auth middleware","status":"in_progress","priority":"high","labels":["auth","backend"]}}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.cmdIssueUpdate(core.NewOptions(
		core.Option{Key: "_arg", Value: "fix-auth"},
		core.Option{Key: "title", Value: "Fix auth middleware"},
		core.Option{Key: "status", Value: "in_progress"},
		core.Option{Key: "priority", Value: "high"},
		core.Option{Key: "labels", Value: "auth,backend"},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(IssueOutput)
	require.True(t, ok)
	assert.Equal(t, "fix-auth", output.Issue.Slug)
	assert.Equal(t, "in_progress", output.Issue.Status)
	assert.Equal(t, []string{"auth", "backend"}, output.Issue.Labels)
}

func TestCommandsforge_CmdIssueUpdate_Bad_MissingSlug(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	result := subsystem.cmdIssueUpdate(core.NewOptions())
	assert.False(t, result.OK)
	assert.EqualError(t, result.Value.(error), "agentic.cmdIssueUpdate: slug or id is required")
}

func TestCommandsforge_CmdIssueAssign_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/issues/fix-auth", r.URL.Path)
		require.Equal(t, http.MethodPatch, r.Method)

		bodyResult := core.ReadAll(r.Body)
		require.True(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		require.True(t, parseResult.OK)
		require.Equal(t, "codex", payload["assignee"])

		_, _ = w.Write([]byte(`{"data":{"issue":{"slug":"fix-auth","title":"Fix auth middleware","status":"open","assignee":"codex"}}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.cmdIssueAssign(core.NewOptions(
		core.Option{Key: "slug", Value: "fix-auth"},
		core.Option{Key: "assignee", Value: "codex"},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(IssueOutput)
	require.True(t, ok)
	assert.Equal(t, "fix-auth", output.Issue.Slug)
	assert.Equal(t, "codex", output.Issue.Assignee)
}

func TestCommandsforge_CmdIssueAssign_Bad_MissingAssignee(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	result := subsystem.cmdIssueAssign(core.NewOptions(core.Option{Key: "slug", Value: "fix-auth"}))
	assert.False(t, result.OK)
	assert.EqualError(t, result.Value.(error), "agentic.cmdIssueAssign: slug or id and assignee are required")
}

func TestCommandsforge_CmdIssueReport_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/issues/fix-auth/comments", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)

		bodyResult := core.ReadAll(r.Body)
		require.True(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		require.True(t, parseResult.OK)

		reportBody := core.JSONMarshalString(map[string]any{
			"summary": "Build failed",
		})
		require.Equal(t, "codex", payload["author"])
		require.Equal(t, core.Concat("```json\n", reportBody, "\n```"), payload["body"])

		_, _ = w.Write([]byte("{\"data\":{\"comment\":{\"id\":7,\"author\":\"codex\",\"body\":\"report received\"}}}"))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.cmdIssueReport(core.NewOptions(
		core.Option{Key: "slug", Value: "fix-auth"},
		core.Option{Key: "report", Value: map[string]any{"summary": "Build failed"}},
		core.Option{Key: "author", Value: "codex"},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(IssueReportOutput)
	require.True(t, ok)
	assert.Equal(t, 7, output.Comment.ID)
	assert.Equal(t, "codex", output.Comment.Author)
}

func TestCommandsforge_CmdIssueReport_Bad_MissingSlug(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	result := subsystem.cmdIssueReport(core.NewOptions())
	assert.False(t, result.OK)
	assert.EqualError(t, result.Value.(error), "agentic.cmdIssueReport: slug or id is required")
}

func TestCommandsforge_CmdIssueArchive_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/issues/fix-auth", r.URL.Path)
		require.Equal(t, http.MethodDelete, r.Method)

		_, _ = w.Write([]byte(`{"data":{"result":{"slug":"fix-auth","success":true}}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.cmdIssueArchive(core.NewOptions(
		core.Option{Key: "slug", Value: "fix-auth"},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(IssueArchiveOutput)
	require.True(t, ok)
	assert.True(t, output.Success)
	assert.Equal(t, "fix-auth", output.Archived)
}

func TestCommandsforge_CmdIssueArchive_Ugly_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.cmdIssueArchive(core.NewOptions(
		core.Option{Key: "_arg", Value: "fix-auth"},
	))
	assert.False(t, result.OK)
}

func TestCommandsforge_CmdPRGet_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) }))
	t.Cleanup(srv.Close)
	s, _ := testPrepWithCore(t, srv)
	r := s.cmdPRGet(core.NewOptions(
		core.Option{Key: "_arg", Value: "../../../etc/passwd"},
		core.Option{Key: "number", Value: "1"},
	))
	assert.False(t, r.OK)
}

func TestCommandsforge_CmdPRList_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	t.Cleanup(srv.Close)
	s, _ := testPrepWithCore(t, srv)
	r := s.cmdPRList(core.NewOptions(core.Option{Key: "_arg", Value: "repo%00null"}))
	assert.False(t, r.OK)
}

func TestCommandsforge_CmdPRMerge_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(422) }))
	t.Cleanup(srv.Close)
	s, _ := testPrepWithCore(t, srv)
	r := s.cmdPRMerge(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "number", Value: "1"},
		core.Option{Key: "method", Value: "invalid-method"},
	))
	assert.False(t, r.OK)
}

func TestCommandsforge_CmdRepoGet_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) }))
	t.Cleanup(srv.Close)
	s, _ := testPrepWithCore(t, srv)
	r := s.cmdRepoGet(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "org", Value: "org/with/slashes"},
	))
	assert.False(t, r.OK)
}

func TestCommandsforge_CmdRepoList_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	t.Cleanup(srv.Close)
	s, _ := testPrepWithCore(t, srv)
	r := s.cmdRepoList(core.NewOptions(core.Option{Key: "org", Value: "<script>alert(1)</script>"}))
	assert.False(t, r.OK)
}

func TestCommandsforge_CmdRepoSync_Bad_MissingRepo(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdRepoSync(core.NewOptions())
	assert.False(t, r.OK)
}

func TestCommandsforge_CmdRepoSync_Good_ResetLocalRepo(t *testing.T) {
	codeDir := t.TempDir()
	orgDir := core.JoinPath(codeDir, "core")
	fs.EnsureDir(orgDir)
	repoDir := core.JoinPath(orgDir, "test-repo")
	fs.EnsureDir(repoDir)

	binDir := t.TempDir()
	logPath := core.JoinPath(t.TempDir(), "git.log")
	gitPath := core.JoinPath(binDir, "git")
	fs.Write(gitPath, core.Concat("#!/bin/sh\nprintf '%s\\n' \"$*\" >> ", logPath, "\nexit 0\n"))
	assert.True(t, testCore.Process().RunIn(context.Background(), binDir, "chmod", "+x", gitPath).OK)
	oldPath := core.Env("PATH")
	t.Setenv("PATH", core.Concat(binDir, ":", oldPath))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       codeDir,
	}

	output := captureStdout(t, func() {
		r := s.cmdRepoSync(core.NewOptions(
			core.Option{Key: "_arg", Value: "test-repo"},
			core.Option{Key: "org", Value: "core"},
			core.Option{Key: "branch", Value: "main"},
			core.Option{Key: "reset", Value: true},
		))
		assert.True(t, r.OK)
	})

	assert.Contains(t, output, "fetched core/test-repo@main")
	assert.Contains(t, output, "reset")

	logResult := fs.Read(logPath)
	assert.True(t, logResult.OK)
	assert.Contains(t, logResult.Value.(string), "fetch origin")
	assert.Contains(t, logResult.Value.(string), "reset --hard origin/main")
}

func TestCommandsforge_RegisterForgeCommands_Good_RepoSyncRegistered(t *testing.T) {
	s, c := testPrepWithCore(t, nil)
	s.registerForgeCommands()
	assert.Contains(t, c.Commands(), "repo/sync")
	assert.Contains(t, c.Commands(), "agentic:repo/sync")
	assert.Contains(t, c.Commands(), "issue/get")
	assert.Contains(t, c.Commands(), "agentic:issue/get")
	assert.Contains(t, c.Commands(), "issue/list")
	assert.Contains(t, c.Commands(), "agentic:issue/list")
	assert.Contains(t, c.Commands(), "issue/comment")
	assert.Contains(t, c.Commands(), "agentic:issue/comment")
	assert.Contains(t, c.Commands(), "issue/create")
	assert.Contains(t, c.Commands(), "agentic:issue/create")
	assert.Contains(t, c.Commands(), "issue/assign")
	assert.Contains(t, c.Commands(), "agentic:issue/assign")
	assert.Contains(t, c.Commands(), "issue/report")
	assert.Contains(t, c.Commands(), "agentic:issue/report")
	assert.Contains(t, c.Commands(), "issue/update")
	assert.Contains(t, c.Commands(), "agentic:issue/update")
	assert.Contains(t, c.Commands(), "issue/archive")
	assert.Contains(t, c.Commands(), "agentic:issue/archive")
	assert.Contains(t, c.Commands(), "pr/get")
	assert.Contains(t, c.Commands(), "agentic:pr/get")
	assert.Contains(t, c.Commands(), "pr/list")
	assert.Contains(t, c.Commands(), "agentic:pr/list")
	assert.Contains(t, c.Commands(), "pr/merge")
	assert.Contains(t, c.Commands(), "agentic:pr/merge")
	assert.Contains(t, c.Commands(), "pr/close")
	assert.Contains(t, c.Commands(), "agentic:pr/close")
	assert.Contains(t, c.Commands(), "repo/get")
	assert.Contains(t, c.Commands(), "agentic:repo/get")
	assert.Contains(t, c.Commands(), "repo/list")
	assert.Contains(t, c.Commands(), "agentic:repo/list")
}
