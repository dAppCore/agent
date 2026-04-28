// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

// --- parseForgeArgs ---

func TestCommandsforge_ParseForgeArgs_Good_AllFields(t *testing.T) {
	opts := core.NewOptions(
		core.Option{Key: "org", Value: "myorg"},
		core.Option{Key: "_arg", Value: "myrepo"},
		core.Option{Key: "number", Value: "42"},
	)
	org, repo, num := parseForgeArgs(opts)
	core.AssertEqual(t, "myorg", org)
	core.AssertEqual(t, "myrepo", repo)
	core.AssertEqual(t, int64(42), num)
}

func TestCommandsforge_ParseForgeArgs_Good_DefaultOrg(t *testing.T) {
	opts := core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
	)
	org, repo, num := parseForgeArgs(opts)
	core.AssertEqual(t, "core", org, "should default to 'core'")
	core.AssertEqual(t, "go-io", repo)
	core.AssertEqual(t, int64(0), num, "no number provided")
}

func TestCommandsforge_ParseForgeArgs_Bad_EmptyOpts(t *testing.T) {
	opts := core.NewOptions()
	org, repo, num := parseForgeArgs(opts)
	core.AssertEqual(t, "core", org, "should default to 'core'")
	core.AssertEmpty(t, repo)
	core.AssertEqual(t, int64(0), num)
}

func TestCommandsforge_ParseForgeArgs_Bad_InvalidNumber(t *testing.T) {
	opts := core.NewOptions(
		core.Option{Key: "_arg", Value: "repo"},
		core.Option{Key: "number", Value: "not-a-number"},
	)
	_, _, num := parseForgeArgs(opts)
	core.AssertEqual(t, int64(0), num, "invalid number should parse as 0")
}

// --- formatIndex ---

func TestCommandsforge_FormatIndex_Good(t *testing.T) {
	core.AssertEqual(t, "1", formatIndex(1))
	core.AssertEqual(t, "42", formatIndex(42))
	core.AssertEqual(t, "0", formatIndex(0))
	core.AssertEqual(t, "999999", formatIndex(999999))
}

// --- parseForgeArgs Ugly ---

func TestCommandsforge_ParseForgeArgs_Ugly_OrgSetButNoRepo(t *testing.T) {
	opts := core.NewOptions(
		core.Option{Key: "org", Value: "custom-org"},
	)
	org, repo, num := parseForgeArgs(opts)
	core.AssertEqual(t, "custom-org", org)
	core.AssertEmpty(t, repo, "repo should be empty when only org is set")
	core.AssertEqual(t, int64(0), num)
}

func TestCommandsforge_ParseForgeArgs_Ugly_NegativeNumber(t *testing.T) {
	opts := core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "number", Value: "-5"},
	)
	_, _, num := parseForgeArgs(opts)
	core.AssertEqual(t, int64(-5), num, "negative numbers parse but are semantically invalid")
}

func TestCommandsforge_ParseForgeArgs_Ugly_InvalidNames(t *testing.T) {
	opts := core.NewOptions(
		core.Option{Key: "org", Value: "bad/org"},
		core.Option{Key: "_arg", Value: "repo/with/slashes"},
	)
	org, repo, num := parseForgeArgs(opts)
	core.AssertEmpty(t, org)
	core.AssertEmpty(t, repo)
	core.AssertEqual(t, int64(0), num)
}

// --- formatIndex Bad/Ugly ---

func TestCommandsforge_FormatIndex_Bad_Negative(t *testing.T) {
	result := formatIndex(-1)
	core.AssertEqual(t, "-1", result, "negative should format as negative string")
	core.AssertContains(t, result, "-")
}

func TestCommandsforge_FormatIndex_Ugly_VeryLarge(t *testing.T) {
	result := formatIndex(9999999999)
	core.AssertEqual(t, "9999999999", result)
	core.AssertLen(t, result, 10)
}

func TestCommandsforge_FormatIndex_Ugly_MaxInt64(t *testing.T) {
	result := formatIndex(9223372036854775807) // math.MaxInt64
	core.AssertNotEmpty(t, result)
	core.AssertEqual(t, "9223372036854775807", result)
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
	core.AssertFalse(t, r.OK)
}

func TestCommandsforge_CmdIssueList_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	t.Cleanup(srv.Close)
	s, _ := testPrepWithCore(t, srv)
	r := s.cmdIssueList(core.NewOptions(core.Option{Key: "_arg", Value: "repo&evil=true"}))
	core.AssertFalse(t, r.OK)
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
	core.AssertFalse(t, r.OK)
}

func TestCommandsforge_CmdIssueCreate_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	t.Cleanup(srv.Close)
	s, _ := testPrepWithCore(t, srv)
	r := s.cmdIssueCreate(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "title", Value: "Fix <b>bug</b> #123"},
	))
	core.AssertFalse(t, r.OK)
}

func TestCommandsforge_CmdIssueUpdate_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/issues/fix-auth", r.URL.Path)
		core.AssertEqual(t, http.MethodPatch, r.Method)

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		core.RequireTrue(t, parseResult.OK)
		core.AssertEqual(t, "Fix auth middleware", payload["title"])
		core.AssertEqual(t, "in_progress", payload["status"])

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
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(IssueOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "fix-auth", output.Issue.Slug)
	core.AssertEqual(t, "in_progress", output.Issue.Status)
	core.AssertEqual(t, []string{"auth", "backend"}, output.Issue.Labels)
}

func TestCommandsforge_CmdIssueUpdate_Bad_MissingSlug(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	result := subsystem.cmdIssueUpdate(core.NewOptions())
	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error), "agentic.cmdIssueUpdate: slug or id is required")
}

func TestCommandsforge_CmdIssueAssign_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/issues/fix-auth", r.URL.Path)
		core.AssertEqual(t, http.MethodPatch, r.Method)

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		core.RequireTrue(t, parseResult.OK)
		core.AssertEqual(t, "codex", payload["assignee"])

		_, _ = w.Write([]byte(`{"data":{"issue":{"slug":"fix-auth","title":"Fix auth middleware","status":"open","assignee":"codex"}}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.cmdIssueAssign(core.NewOptions(
		core.Option{Key: "slug", Value: "fix-auth"},
		core.Option{Key: "assignee", Value: "codex"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(IssueOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "fix-auth", output.Issue.Slug)
	core.AssertEqual(t, "codex", output.Issue.Assignee)
}

func TestCommandsforge_CmdIssueAssign_Bad_MissingAssignee(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	result := subsystem.cmdIssueAssign(core.NewOptions(core.Option{Key: "slug", Value: "fix-auth"}))
	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error), "agentic.cmdIssueAssign: slug or id and assignee are required")
}

func TestCommandsforge_CmdIssueReport_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/issues/fix-auth/comments", r.URL.Path)
		core.AssertEqual(t, http.MethodPost, r.Method)

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		core.RequireTrue(t, parseResult.OK)

		reportBody := core.JSONMarshalString(map[string]any{
			"summary": "Build failed",
		})
		core.AssertEqual(t, "codex", payload["author"])
		core.AssertEqual(t, core.Concat("```json\n", reportBody, "\n```"), payload["body"])

		_, _ = w.Write([]byte("{\"data\":{\"comment\":{\"id\":7,\"author\":\"codex\",\"body\":\"report received\"}}}"))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.cmdIssueReport(core.NewOptions(
		core.Option{Key: "slug", Value: "fix-auth"},
		core.Option{Key: "report", Value: map[string]any{"summary": "Build failed"}},
		core.Option{Key: "author", Value: "codex"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(IssueReportOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 7, output.Comment.ID)
	core.AssertEqual(t, "codex", output.Comment.Author)
}

func TestCommandsforge_CmdIssueReport_Bad_MissingSlug(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	result := subsystem.cmdIssueReport(core.NewOptions())
	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error), "agentic.cmdIssueReport: slug or id is required")
}

func TestCommandsforge_CmdIssueArchive_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/issues/fix-auth", r.URL.Path)
		core.AssertEqual(t, http.MethodDelete, r.Method)

		_, _ = w.Write([]byte(`{"data":{"result":{"slug":"fix-auth","success":true}}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.cmdIssueArchive(core.NewOptions(
		core.Option{Key: "slug", Value: "fix-auth"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(IssueArchiveOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, "fix-auth", output.Archived)
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
	core.AssertFalse(t, result.OK)
}

func TestCommandsforge_CmdPRGet_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) }))
	t.Cleanup(srv.Close)
	s, _ := testPrepWithCore(t, srv)
	r := s.cmdPRGet(core.NewOptions(
		core.Option{Key: "_arg", Value: "../../../etc/passwd"},
		core.Option{Key: "number", Value: "1"},
	))
	core.AssertFalse(t, r.OK)
}

func TestCommandsforge_CmdPRList_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	t.Cleanup(srv.Close)
	s, _ := testPrepWithCore(t, srv)
	r := s.cmdPRList(core.NewOptions(core.Option{Key: "_arg", Value: "repo%00null"}))
	core.AssertFalse(t, r.OK)
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
	core.AssertFalse(t, r.OK)
}

func TestCommandsforge_CmdRepoGet_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) }))
	t.Cleanup(srv.Close)
	s, _ := testPrepWithCore(t, srv)
	r := s.cmdRepoGet(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "org", Value: "org/with/slashes"},
	))
	core.AssertFalse(t, r.OK)
}

func TestCommandsforge_CmdRepoList_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	t.Cleanup(srv.Close)
	s, _ := testPrepWithCore(t, srv)
	r := s.cmdRepoList(core.NewOptions(core.Option{Key: "org", Value: "<script>alert(1)</script>"}))
	core.AssertFalse(t, r.OK)
}

func TestCommandsforge_CmdRepoSync_Bad_MissingRepo(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdRepoSync(core.NewOptions())
	core.AssertFalse(t, r.OK)
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
	core.AssertTrue(t, testCore.Process().RunIn(context.Background(), binDir, "chmod", "+x", gitPath).OK)
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
		core.AssertTrue(t, r.OK)
	})

	core.AssertContains(t, output, "fetched core/test-repo@main")
	core.AssertContains(t, output, "reset")

	logResult := fs.Read(logPath)
	core.AssertTrue(t, logResult.OK)
	core.AssertContains(t, logResult.Value.(string), "fetch origin")
	core.AssertContains(t, logResult.Value.(string), "reset --hard origin/main")
}

func TestCommandsforge_RegisterForgeCommands_Good_RepoSyncRegistered(t *testing.T) {
	s, c := testPrepWithCore(t, nil)
	s.registerForgeCommands()
	core.AssertContains(t, c.Commands(), "repo/sync")
	core.AssertContains(t, c.Commands(), "agentic:repo/sync")
	core.AssertContains(t, c.Commands(), "issue/get")
	core.AssertContains(t, c.Commands(), "agentic:issue/get")
	core.AssertContains(t, c.Commands(), "issue/list")
	core.AssertContains(t, c.Commands(), "agentic:issue/list")
	core.AssertContains(t, c.Commands(), "issue/comment")
	core.AssertContains(t, c.Commands(), "agentic:issue/comment")
	core.AssertContains(t, c.Commands(), "issue/create")
	core.AssertContains(t, c.Commands(), "agentic:issue/create")
	core.AssertContains(t, c.Commands(), "issue/assign")
	core.AssertContains(t, c.Commands(), "agentic:issue/assign")
	core.AssertContains(t, c.Commands(), "issue/report")
	core.AssertContains(t, c.Commands(), "agentic:issue/report")
	core.AssertContains(t, c.Commands(), "issue/update")
	core.AssertContains(t, c.Commands(), "agentic:issue/update")
	core.AssertContains(t, c.Commands(), "issue/archive")
	core.AssertContains(t, c.Commands(), "agentic:issue/archive")
	core.AssertContains(t, c.Commands(), "pr/get")
	core.AssertContains(t, c.Commands(), "agentic:pr/get")
	core.AssertContains(t, c.Commands(), "pr/list")
	core.AssertContains(t, c.Commands(), "agentic:pr/list")
	core.AssertContains(t, c.Commands(), "pr/merge")
	core.AssertContains(t, c.Commands(), "agentic:pr/merge")
	core.AssertContains(t, c.Commands(), "pr/close")
	core.AssertContains(t, c.Commands(), "agentic:pr/close")
	core.AssertContains(t, c.Commands(), "repo/get")
	core.AssertContains(t, c.Commands(), "agentic:repo/get")
	core.AssertContains(t, c.Commands(), "repo/list")
	core.AssertContains(t, c.Commands(), "agentic:repo/list")
}
