// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core "dappco.re/go"
	"dappco.re/go/forge"
)

// --- forgeMergePR ---

func TestVerify_ForgeMergePR_Good_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "POST", r.Method)
		core.AssertContains(t, r.URL.Path, "/pulls/42/merge")
		core.AssertEqual(t, "token test-forge-token", r.Header.Get("Authorization"))

		var body map[string]any
		core.JSONUnmarshalString(core.ReadAll(r.Body).Value.(string), &body)
		core.AssertEqual(t, "merge", body["Do"])
		core.AssertEqual(t, true, body["delete_branch_after_merge"])

		w.WriteHeader(200)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeURL:       srv.URL,
		forgeToken:     "test-forge-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	r := s.forgeMergePR(context.Background(), "core", "test-repo", 42)
	core.AssertTrue(t, r.OK)
}

func TestVerify_ForgeMergePR_Good_204Response(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204) // No Content — also valid success
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	r := s.forgeMergePR(context.Background(), "core", "test-repo", 1)
	core.AssertTrue(t, r.OK)
}

func TestVerify_ForgeMergePR_Bad_ConflictResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(409)
		w.Write([]byte(core.JSONMarshalString(map[string]any{
			"message": "merge conflict",
		})))
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	r := s.forgeMergePR(context.Background(), "core", "test-repo", 1)
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Value.(string), "merge conflict")
}

func TestVerify_ForgeMergePR_Bad_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(core.JSONMarshalString(map[string]any{
			"message": "internal server error",
		})))
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	r := s.forgeMergePR(context.Background(), "core", "test-repo", 1)
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Value.(string), "internal server error")
}

func TestVerify_ForgeMergePR_Bad_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close() // close immediately to cause connection error

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	r := s.forgeMergePR(context.Background(), "core", "test-repo", 1)
	core.AssertFalse(t, r.OK)
}

// --- extractPullRequestNumber (additional _Ugly cases) ---

func TestVerify_ExtractPullRequestNumber_Ugly_DoubleSlashEnd(t *testing.T) {
	number := extractPullRequestNumber("https://forge.lthn.ai/core/go-io/pulls/42/")
	core.AssertEqual(t, 0, number)
	core.AssertFalse(t, number > 0)
}

func TestVerify_ExtractPullRequestNumber_Ugly_VeryLargeNumber(t *testing.T) {
	number := extractPullRequestNumber("https://forge.lthn.ai/core/go-io/pulls/999999")
	core.AssertEqual(t, 999999, number)
	core.AssertTrue(t, number > 42)
}

func TestVerify_ExtractPullRequestNumber_Ugly_NegativeNumber(t *testing.T) {
	// atoi of "-5" is -5, parseInt wraps atoi
	number := extractPullRequestNumber("https://forge.lthn.ai/core/go-io/pulls/-5")
	core.AssertEqual(t, -5, number)
	core.AssertTrue(t, number < 0)
}

func TestVerify_ExtractPullRequestNumber_Ugly_ZeroExplicit(t *testing.T) {
	number := extractPullRequestNumber("https://forge.lthn.ai/core/go-io/pulls/0")
	core.AssertEqual(t, 0, number)
	core.AssertFalse(t, number > 0)
}

// --- ensureLabel ---

func TestVerify_EnsureLabel_Good_CreatesLabel(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "POST", r.Method)
		core.AssertContains(t, r.URL.Path, "/labels")
		called = true

		var body map[string]string
		core.JSONUnmarshalString(core.ReadAll(r.Body).Value.(string), &body)
		core.AssertEqual(t, "needs-review", body["name"])
		core.AssertEqual(t, "#e11d48", body["color"])

		w.WriteHeader(201)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	s.ensureLabel(context.Background(), "core", "test-repo", "needs-review", "e11d48")
	core.AssertTrue(t, called)
}

func TestVerify_EnsureLabel_Bad_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Should not panic
	core.AssertNotPanics(t, func() {
		s.ensureLabel(context.Background(), "core", "test-repo", "test-label", "abc123")
	})
}

// --- getLabelID ---

func TestVerify_GetLabelID_Good_Found(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString([]map[string]any{
			{"id": 10, "name": "agentic"},
			{"id": 20, "name": "needs-review"},
		})))
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	id := s.getLabelID(context.Background(), "core", "test-repo", "needs-review")
	core.AssertEqual(t, 20, id)
}

func TestVerify_GetLabelID_Bad_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString([]map[string]any{
			{"id": 10, "name": "agentic"},
		})))
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	id := s.getLabelID(context.Background(), "core", "test-repo", "missing-label")
	core.AssertEqual(t, 0, id)
}

func TestVerify_GetLabelID_Bad_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	id := s.getLabelID(context.Background(), "core", "test-repo", "any")
	core.AssertEqual(t, 0, id)
}

// --- runVerification ---

func TestVerify_RunVerification_Good_NoProjectFile(t *testing.T) {
	dir := t.TempDir() // No go.mod, composer.json, or package.json

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	result := s.runVerification(dir)
	core.AssertTrue(t, result.passed)
	core.AssertEqual(t, "none", result.testCmd)
}

func TestVerify_RunVerification_Good_GoProject(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "go.mod"), "module test").OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	result := s.runVerification(dir)
	core.AssertEqual(t, "go test ./...", result.testCmd)
	// It will fail because there's no real Go code, but we test the detection path
}

func TestVerify_RunVerification_Good_PHPProject(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "composer.json"), `{"require":{}}`).OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	result := s.runVerification(dir)
	// Will fail (no composer) but detection path is covered
	core.AssertContains(t, []string{"composer test", "vendor/bin/pest", "none"}, result.testCmd)
}

func TestVerify_RunVerification_Good_NodeProject(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "package.json"), `{"scripts":{"test":"echo ok"}}`).OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	result := s.runVerification(dir)
	core.AssertEqual(t, "npm test", result.testCmd)
}

func TestVerify_RunVerification_Good_NodeNoTestScript(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "package.json"), `{"scripts":{}}`).OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	result := s.runVerification(dir)
	core.AssertTrue(t, result.passed)
	core.AssertEqual(t, "none", result.testCmd)
}

// --- fileExists ---

func TestVerify_FileExists_Good_Exists(t *testing.T) {
	dir := t.TempDir()
	path := core.JoinPath(dir, "test.txt")
	core.RequireTrue(t, fs.Write(path, "hello").OK)

	core.AssertTrue(t, fileExists(path))
}

func TestVerify_FileExists_Bad_NotExists(t *testing.T) {
	path := "/nonexistent/path/file.txt"
	exists := fileExists(path)
	core.AssertFalse(t, exists)
	core.AssertEqual(t, false, exists)
}

func TestVerify_FileExists_Bad_IsDirectory(t *testing.T) {
	dir := t.TempDir()
	exists := fileExists(dir)
	core.AssertFalse(t, exists) // directories are not files
	core.AssertEqual(t, false, exists)
}

// --- autoVerifyAndMerge ---

func TestVerify_AutoVerifyAndMerge_Bad_NoStatus(t *testing.T) {
	dir := t.TempDir()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	// Should not panic when status.json is missing
	core.AssertNotPanics(t, func() {
		s.autoVerifyAndMerge(dir)
	})
}

func TestVerify_AutoVerifyAndMerge_Bad_NoPRURL(t *testing.T) {
	dir := t.TempDir()
	core.RequireNoError(t, writeStatus(dir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Branch: "agent/fix",
	}))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Should return early — no PR URL
	core.AssertNotPanics(t, func() {
		s.autoVerifyAndMerge(dir)
	})
}

func TestVerify_AutoVerifyAndMerge_Bad_EmptyRepo(t *testing.T) {
	dir := t.TempDir()
	core.RequireNoError(t, writeStatus(dir, &WorkspaceStatus{
		Status: "completed",
		PRURL:  "https://forge.test/core/go-io/pulls/1",
	}))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	core.AssertNotPanics(t, func() {
		s.autoVerifyAndMerge(dir)
	})
}

func TestVerify_AutoVerifyAndMerge_Bad_InvalidPRURL(t *testing.T) {
	dir := t.TempDir()
	core.RequireNoError(t, writeStatus(dir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Branch: "agent/fix",
		PRURL:  "not-a-url",
	}))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// extractPullRequestNumber returns 0 for invalid URL, so autoVerifyAndMerge returns early
	core.AssertNotPanics(t, func() {
		s.autoVerifyAndMerge(dir)
	})
}

// --- flagForReview ---

func TestVerify_FlagForReview_Good_AddsLabel(t *testing.T) {
	labelCalled := false
	commentCalled := false

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && containsStr(r.URL.Path, "/labels") {
			labelCalled = true
			if containsStr(r.URL.Path, "/issues/") {
				w.WriteHeader(200) // add label to issue
			} else {
				w.WriteHeader(201) // create label
			}
			return
		}
		if r.Method == "GET" && containsStr(r.URL.Path, "/labels") {
			w.Write([]byte(core.JSONMarshalString([]map[string]any{
				{"id": 99, "name": "needs-review"},
			})))
			return
		}
		if r.Method == "POST" && containsStr(r.URL.Path, "/comments") {
			commentCalled = true
			w.WriteHeader(201)
			return
		}
		w.WriteHeader(200)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	s.flagForReview("core", "test-repo", 42, testFailed)
	core.AssertTrue(t, labelCalled)
	core.AssertTrue(t, commentCalled)
}

func TestVerify_FlagForReview_Good_MergeConflictMessage(t *testing.T) {
	var commentBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && containsStr(r.URL.Path, "/labels") {
			w.Write([]byte(core.JSONMarshalString([]map[string]any{})))
			return
		}
		if r.Method == "POST" && containsStr(r.URL.Path, "/comments") {
			var body map[string]string
			core.JSONUnmarshalString(core.ReadAll(r.Body).Value.(string), &body)
			commentBody = body["body"]
			w.WriteHeader(201)
			return
		}
		w.WriteHeader(201) // default for label creation etc
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	s.flagForReview("core", "test-repo", 1, mergeConflict)
	core.AssertContains(t, commentBody, "Merge conflict")
}

// --- truncate ---

func TestAutopr_Truncate_Good_Short(t *testing.T) {
	result := truncate("hello", 10)
	core.AssertEqual(t, "hello", result)
	core.AssertLen(t, result, 5)
}

func TestAutopr_Truncate_Good_Exact(t *testing.T) {
	result := truncate("hello", 5)
	core.AssertEqual(t, "hello", result)
	core.AssertLen(t, result, 5)
}

func TestAutopr_Truncate_Good_Long(t *testing.T) {
	result := truncate("hello world", 3)
	core.AssertEqual(t, "hel...", result)
	core.AssertContains(t, result, "...")
}

func TestAutopr_Truncate_Bad_ZeroMax(t *testing.T) {
	result := truncate("hello", 0)
	core.AssertEqual(t, "...", result)
	core.AssertContains(t, result, "...")
}

func TestAutopr_Truncate_Ugly_EmptyString(t *testing.T) {
	result := truncate("", 10)
	core.AssertEqual(t, "", result)
	core.AssertEmpty(t, result)
}

// --- autoVerifyAndMerge (extended Ugly) ---

func TestVerify_AutoVerifyAndMerge_Ugly(t *testing.T) {
	// Workspace with status=completed, repo=test, PRURL="not-a-url"
	// extractPullRequestNumber returns 0 for "not-a-url" → early return, no panic
	dir := t.TempDir()
	core.RequireNoError(t, writeStatus(dir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "test",
		Branch: "agent/fix",
		PRURL:  "not-a-url",
	}))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// PR number is 0 → should return early without panicking
	core.AssertNotPanics(t, func() {
		s.autoVerifyAndMerge(dir)
	})

	// Status should remain unchanged (not "merged")
	st := mustReadStatus(t, dir)
	core.AssertEqual(t, "completed", st.Status)
}

// --- attemptVerifyAndMerge (Ugly — Go project that fails build) ---

func TestVerify_AttemptVerifyAndMerge_Ugly(t *testing.T) {
	// Go project that fails build (go.mod but no valid Go code)
	// with httptest Forge mock for comment API → returns testFailed
	commentCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && containsStr(r.URL.Path, "/comments") {
			commentCalled = true
			w.Write([]byte(core.JSONMarshalString(map[string]any{"id": 1})))
			return
		}
		w.WriteHeader(200)
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	// Write a go.mod so runVerification detects Go and runs "go test ./..."
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "go.mod"), "module broken-test\n\ngo 1.22").OK)
	// Write invalid Go code to force test failure
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "broken.go"), "package broken\n\nfunc Bad() { undeclared() }").OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	result := s.attemptVerifyAndMerge(dir, "core", "test-repo", "agent/fix", 42)
	core.AssertEqual(t, testFailed, result)
	core.AssertTrue(t, commentCalled, "should have posted a comment about test failure")
}

// --- extractPullRequestNumber (extended Ugly) ---

func TestVerify_ExtractPullRequestNumber_Ugly(t *testing.T) {
	// Just a bare number "5" → last segment is "5" → returns 5
	core.AssertEqual(t, 5, extractPullRequestNumber("5"))

	// Trailing slash → last segment is empty string → parseInt("") → 0
	core.AssertEqual(t, 0, extractPullRequestNumber("https://forge.lthn.ai/core/go-io/pulls/42/"))

	// Non-numeric string → parseInt("abc") → 0
	core.AssertEqual(t, 0, extractPullRequestNumber("abc"))
}

// --- EnsureLabel Ugly ---

func TestVerify_EnsureLabel_Ugly_AlreadyExists409(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Server returns 409 Conflict — label already exists
		w.WriteHeader(409)
		w.Write([]byte(core.JSONMarshalString(map[string]any{"message": "label already exists"})))
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Should not panic on 409 — ensureLabel is fire-and-forget
	core.AssertNotPanics(t, func() {
		s.ensureLabel(context.Background(), "core", "test-repo", "needs-review", "e11d48")
	})
}

// --- GetLabelID Ugly ---

func TestVerify_GetLabelID_Ugly_EmptyArray(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString([]map[string]any{})))
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	id := s.getLabelID(context.Background(), "core", "test-repo", "needs-review")
	core.AssertEqual(t, 0, id)
}

// --- ForgeMergePR Ugly ---

func TestVerify_ForgeMergePR_Ugly_EmptyBody200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		// Empty body — no JSON at all
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	r := s.forgeMergePR(context.Background(), "core", "test-repo", 42)
	core.AssertTrue(t, r.OK) // 200 is success even with empty body
}

// --- FileExists Ugly ---

func TestVerify_FileExists_Ugly_PathIsDirectory(t *testing.T) {
	dir := t.TempDir()
	sub := core.JoinPath(dir, "subdir")
	fs.EnsureDir(sub)

	// A directory is not a file — fileExists should return false
	core.AssertFalse(t, fileExists(sub))
}

// --- FlagForReview Bad/Ugly ---

func TestVerify_FlagForReview_Bad_AllAPICallsFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(core.JSONMarshalString(map[string]any{"message": "server error"})))
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Should not panic when all API calls (ensureLabel, getLabelID, add label, comment) fail
	core.AssertNotPanics(t, func() {
		s.flagForReview("core", "test-repo", 42, testFailed)
	})
}

func TestVerify_FlagForReview_Ugly_LabelNotFoundZeroID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && containsStr(r.URL.Path, "/labels") {
			// getLabelID returns empty array → label ID is 0
			w.Write([]byte(core.JSONMarshalString([]map[string]any{})))
			return
		}
		// All other calls succeed
		w.WriteHeader(201)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// label ID 0 is passed to "add labels" payload — should not panic
	core.AssertNotPanics(t, func() {
		s.flagForReview("core", "test-repo", 42, mergeConflict)
	})
}

// --- RunVerification Bad/Ugly ---

func TestVerify_RunVerification_Bad_GoModButNoGoFiles(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "go.mod"), "module test\n\ngo 1.22").OK)
	// go.mod exists but no .go files — go test should fail

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	result := s.runVerification(dir)
	core.AssertEqual(t, "go test ./...", result.testCmd)
	// Depending on go version, this may pass (no test files = pass) or fail
	// The important thing is we detect Go project type correctly
}

func TestVerify_RunVerification_Ugly_MultipleProjectFiles(t *testing.T) {
	dir := t.TempDir()
	// Both go.mod and package.json exist — Go takes priority
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "go.mod"), "module test\n\ngo 1.22").OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "package.json"), `{"scripts":{"test":"echo ok"}}`).OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	result := s.runVerification(dir)
	// Go takes priority because it's checked first
	core.AssertEqual(t, "go test ./...", result.testCmd)
}

// --- additional: go.mod + composer.json to verify priority ---

func TestVerify_RunVerification_Ugly_GoAndPHPProjectFiles(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "go.mod"), "module test\n\ngo 1.22").OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "composer.json"), `{"require":{}}`).OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	result := s.runVerification(dir)
	core.AssertEqual(t, "go test ./...", result.testCmd) // Go first in priority chain
}

// --- runGoTests ---

func TestVerify_RunGoTests_Good(t *testing.T) {
	dir := t.TempDir()
	// Create a valid Go project with a passing test
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "go.mod"), "module testproj\n\ngo 1.22\n").OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "main.go"), "package testproj\n\nfunc Add(a, b int) int { return a + b }\n").OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "main_test.go"), `package testproj

import "testing"

func TestVerify_Add_Good(t *testing.T) {
	if Add(1, 2) != 3 {
		t.Fatal("expected 3")
	}
}
`).OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	result := s.runGoTests(dir)
	core.AssertTrue(t, result.passed)
	core.AssertEqual(t, "go test ./...", result.testCmd)
	core.AssertEqual(t, 0, result.exitCode)
}

func TestVerify_RunGoTests_Bad(t *testing.T) {
	dir := t.TempDir()
	// Create a broken Go project — compilation error
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "go.mod"), "module broken\n\ngo 1.22\n").OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "broken.go"), "package broken\n\nfunc Bad() { undeclared() }\n").OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	result := s.runGoTests(dir)
	core.AssertFalse(t, result.passed)
	core.AssertEqual(t, "go test ./...", result.testCmd)
	core.AssertEqual(t, 1, result.exitCode)
}

func TestVerify_RunGoTests_Ugly(t *testing.T) {
	dir := t.TempDir()
	// go.mod but no test files — Go considers this a pass
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "go.mod"), "module empty\n\ngo 1.22\n").OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "main.go"), "package empty\n").OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	result := s.runGoTests(dir)
	// No test files is a pass in go test
	core.AssertTrue(t, result.passed)
	core.AssertEqual(t, "go test ./...", result.testCmd)
	core.AssertEqual(t, 0, result.exitCode)
}
