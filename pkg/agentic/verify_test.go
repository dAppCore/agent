// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"dappco.re/go/core/forge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- forgeMergePR ---

func TestVerify_ForgeMergePR_Good_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/pulls/42/merge")
		assert.Equal(t, "token test-forge-token", r.Header.Get("Authorization"))

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "merge", body["Do"])
		assert.Equal(t, true, body["delete_branch_after_merge"])

		w.WriteHeader(200)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		forgeURL:   srv.URL,
		forgeToken: "test-forge-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	err := s.forgeMergePR(context.Background(), "core", "test-repo", 42)
	assert.NoError(t, err)
}

func TestVerify_ForgeMergePR_Good_204Response(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204) // No Content — also valid success
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	err := s.forgeMergePR(context.Background(), "core", "test-repo", 1)
	assert.NoError(t, err)
}

func TestVerify_ForgeMergePR_Bad_ConflictResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(409)
		json.NewEncoder(w).Encode(map[string]any{
			"message": "merge conflict",
		})
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	err := s.forgeMergePR(context.Background(), "core", "test-repo", 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "409")
	assert.Contains(t, err.Error(), "merge conflict")
}

func TestVerify_ForgeMergePR_Bad_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]any{
			"message": "internal server error",
		})
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	err := s.forgeMergePR(context.Background(), "core", "test-repo", 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestVerify_ForgeMergePR_Bad_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close() // close immediately to cause connection error

	s := &PrepSubsystem{
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     &http.Client{},
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	err := s.forgeMergePR(context.Background(), "core", "test-repo", 1)
	assert.Error(t, err)
}

// --- extractPRNumber (additional _Ugly cases) ---

func TestVerify_ExtractPRNumber_Ugly_DoubleSlashEnd(t *testing.T) {
	assert.Equal(t, 0, extractPRNumber("https://forge.lthn.ai/core/go-io/pulls/42/"))
}

func TestVerify_ExtractPRNumber_Ugly_VeryLargeNumber(t *testing.T) {
	assert.Equal(t, 999999, extractPRNumber("https://forge.lthn.ai/core/go-io/pulls/999999"))
}

func TestVerify_ExtractPRNumber_Ugly_NegativeNumber(t *testing.T) {
	// atoi of "-5" is -5, parseInt wraps atoi
	assert.Equal(t, -5, extractPRNumber("https://forge.lthn.ai/core/go-io/pulls/-5"))
}

func TestVerify_ExtractPRNumber_Ugly_ZeroExplicit(t *testing.T) {
	assert.Equal(t, 0, extractPRNumber("https://forge.lthn.ai/core/go-io/pulls/0"))
}

// --- ensureLabel ---

func TestVerify_EnsureLabel_Good_CreatesLabel(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/labels")
		called = true

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "needs-review", body["name"])
		assert.Equal(t, "#e11d48", body["color"])

		w.WriteHeader(201)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	s.ensureLabel(context.Background(), "core", "test-repo", "needs-review", "e11d48")
	assert.True(t, called)
}

func TestVerify_EnsureLabel_Bad_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()

	s := &PrepSubsystem{
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     &http.Client{},
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	// Should not panic
	assert.NotPanics(t, func() {
		s.ensureLabel(context.Background(), "core", "test-repo", "test-label", "abc123")
	})
}

// --- getLabelID ---

func TestVerify_GetLabelID_Good_Found(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": 10, "name": "agentic"},
			{"id": 20, "name": "needs-review"},
		})
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	id := s.getLabelID(context.Background(), "core", "test-repo", "needs-review")
	assert.Equal(t, 20, id)
}

func TestVerify_GetLabelID_Bad_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": 10, "name": "agentic"},
		})
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	id := s.getLabelID(context.Background(), "core", "test-repo", "missing-label")
	assert.Equal(t, 0, id)
}

func TestVerify_GetLabelID_Bad_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()

	s := &PrepSubsystem{
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     &http.Client{},
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	id := s.getLabelID(context.Background(), "core", "test-repo", "any")
	assert.Equal(t, 0, id)
}

// --- runVerification ---

func TestVerify_RunVerification_Good_NoProjectFile(t *testing.T) {
	dir := t.TempDir() // No go.mod, composer.json, or package.json

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	result := s.runVerification(dir)
	assert.True(t, result.passed)
	assert.Equal(t, "none", result.testCmd)
}

func TestVerify_RunVerification_Good_GoProject(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(filepath.Join(dir, "go.mod"), "module test").OK)

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	result := s.runVerification(dir)
	assert.Equal(t, "go test ./...", result.testCmd)
	// It will fail because there's no real Go code, but we test the detection path
}

func TestVerify_RunVerification_Good_PHPProject(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(filepath.Join(dir, "composer.json"), `{"require":{}}`).OK)

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	result := s.runVerification(dir)
	// Will fail (no composer) but detection path is covered
	assert.Contains(t, []string{"composer test", "vendor/bin/pest", "none"}, result.testCmd)
}

func TestVerify_RunVerification_Good_NodeProject(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(filepath.Join(dir, "package.json"), `{"scripts":{"test":"echo ok"}}`).OK)

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	result := s.runVerification(dir)
	assert.Equal(t, "npm test", result.testCmd)
}

func TestVerify_RunVerification_Good_NodeNoTestScript(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(filepath.Join(dir, "package.json"), `{"scripts":{}}`).OK)

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	result := s.runVerification(dir)
	assert.True(t, result.passed)
	assert.Equal(t, "none", result.testCmd)
}

// --- fileExists ---

func TestVerify_FileExists_Good_Exists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	require.True(t, fs.Write(path, "hello").OK)

	assert.True(t, fileExists(path))
}

func TestVerify_FileExists_Bad_NotExists(t *testing.T) {
	assert.False(t, fileExists("/nonexistent/path/file.txt"))
}

func TestVerify_FileExists_Bad_IsDirectory(t *testing.T) {
	dir := t.TempDir()
	assert.False(t, fileExists(dir)) // directories are not files
}

// --- autoVerifyAndMerge ---

func TestVerify_AutoVerifyAndMerge_Bad_NoStatus(t *testing.T) {
	dir := t.TempDir()
	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	// Should not panic when status.json is missing
	assert.NotPanics(t, func() {
		s.autoVerifyAndMerge(dir)
	})
}

func TestVerify_AutoVerifyAndMerge_Bad_NoPRURL(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, writeStatus(dir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Branch: "agent/fix",
	}))

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	// Should return early — no PR URL
	assert.NotPanics(t, func() {
		s.autoVerifyAndMerge(dir)
	})
}

func TestVerify_AutoVerifyAndMerge_Bad_EmptyRepo(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, writeStatus(dir, &WorkspaceStatus{
		Status: "completed",
		PRURL:  "https://forge.test/core/go-io/pulls/1",
	}))

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	assert.NotPanics(t, func() {
		s.autoVerifyAndMerge(dir)
	})
}

func TestVerify_AutoVerifyAndMerge_Bad_InvalidPRURL(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, writeStatus(dir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Branch: "agent/fix",
		PRURL:  "not-a-url",
	}))

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	// extractPRNumber returns 0 for invalid URL, so autoVerifyAndMerge returns early
	assert.NotPanics(t, func() {
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
			json.NewEncoder(w).Encode([]map[string]any{
				{"id": 99, "name": "needs-review"},
			})
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
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	s.flagForReview("core", "test-repo", 42, testFailed)
	assert.True(t, labelCalled)
	assert.True(t, commentCalled)
}

func TestVerify_FlagForReview_Good_MergeConflictMessage(t *testing.T) {
	var commentBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && containsStr(r.URL.Path, "/labels") {
			json.NewEncoder(w).Encode([]map[string]any{})
			return
		}
		if r.Method == "POST" && containsStr(r.URL.Path, "/comments") {
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			commentBody = body["body"]
			w.WriteHeader(201)
			return
		}
		w.WriteHeader(201) // default for label creation etc
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	s.flagForReview("core", "test-repo", 1, mergeConflict)
	assert.Contains(t, commentBody, "Merge conflict")
}

// --- truncate ---

func TestAutoPr_Truncate_Good_Short(t *testing.T) {
	assert.Equal(t, "hello", truncate("hello", 10))
}

func TestAutoPr_Truncate_Good_Exact(t *testing.T) {
	assert.Equal(t, "hello", truncate("hello", 5))
}

func TestAutoPr_Truncate_Good_Long(t *testing.T) {
	assert.Equal(t, "hel...", truncate("hello world", 3))
}

func TestAutoPr_Truncate_Bad_ZeroMax(t *testing.T) {
	assert.Equal(t, "...", truncate("hello", 0))
}

func TestAutoPr_Truncate_Ugly_EmptyString(t *testing.T) {
	assert.Equal(t, "", truncate("", 10))
}

// --- autoVerifyAndMerge (extended Ugly) ---

func TestVerify_AutoVerifyAndMerge_Ugly(t *testing.T) {
	// Workspace with status=completed, repo=test, PRURL="not-a-url"
	// extractPRNumber returns 0 for "not-a-url" → early return, no panic
	dir := t.TempDir()
	require.NoError(t, writeStatus(dir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "test",
		Branch: "agent/fix",
		PRURL:  "not-a-url",
	}))

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	// PR number is 0 → should return early without panicking
	assert.NotPanics(t, func() {
		s.autoVerifyAndMerge(dir)
	})

	// Status should remain unchanged (not "merged")
	st, err := ReadStatus(dir)
	require.NoError(t, err)
	assert.Equal(t, "completed", st.Status)
}

// --- attemptVerifyAndMerge (Ugly — Go project that fails build) ---

func TestVerify_AttemptVerifyAndMerge_Ugly(t *testing.T) {
	// Go project that fails build (go.mod but no valid Go code)
	// with httptest Forge mock for comment API → returns testFailed
	commentCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && containsStr(r.URL.Path, "/comments") {
			commentCalled = true
			json.NewEncoder(w).Encode(map[string]any{"id": 1})
			return
		}
		w.WriteHeader(200)
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	// Write a go.mod so runVerification detects Go and runs "go test ./..."
	require.True(t, fs.Write(filepath.Join(dir, "go.mod"), "module broken-test\n\ngo 1.22").OK)
	// Write invalid Go code to force test failure
	require.True(t, fs.Write(filepath.Join(dir, "broken.go"), "package broken\n\nfunc Bad() { undeclared() }").OK)

	s := &PrepSubsystem{
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	result := s.attemptVerifyAndMerge(dir, "core", "test-repo", "agent/fix", 42)
	assert.Equal(t, testFailed, result)
	assert.True(t, commentCalled, "should have posted a comment about test failure")
}

// --- extractPRNumber (extended Ugly) ---

func TestVerify_ExtractPRNumber_Ugly(t *testing.T) {
	// Just a bare number "5" → last segment is "5" → returns 5
	assert.Equal(t, 5, extractPRNumber("5"))

	// Trailing slash → last segment is empty string → parseInt("") → 0
	assert.Equal(t, 0, extractPRNumber("https://forge.lthn.ai/core/go-io/pulls/42/"))

	// Non-numeric string → parseInt("abc") → 0
	assert.Equal(t, 0, extractPRNumber("abc"))
}

// --- EnsureLabel Ugly ---

func TestVerify_EnsureLabel_Ugly_AlreadyExists409(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Server returns 409 Conflict — label already exists
		w.WriteHeader(409)
		json.NewEncoder(w).Encode(map[string]any{"message": "label already exists"})
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	// Should not panic on 409 — ensureLabel is fire-and-forget
	assert.NotPanics(t, func() {
		s.ensureLabel(context.Background(), "core", "test-repo", "needs-review", "e11d48")
	})
}

// --- GetLabelID Ugly ---

func TestVerify_GetLabelID_Ugly_EmptyArray(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{})
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	id := s.getLabelID(context.Background(), "core", "test-repo", "needs-review")
	assert.Equal(t, 0, id)
}

// --- ForgeMergePR Ugly ---

func TestVerify_ForgeMergePR_Ugly_EmptyBody200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		// Empty body — no JSON at all
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	err := s.forgeMergePR(context.Background(), "core", "test-repo", 42)
	assert.NoError(t, err) // 200 is success even with empty body
}

// --- FileExists Ugly ---

func TestVerify_FileExists_Ugly_PathIsDirectory(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "subdir")
	require.NoError(t, os.MkdirAll(sub, 0o755))

	// A directory is not a file — fileExists should return false
	assert.False(t, fileExists(sub))
}

// --- FlagForReview Bad/Ugly ---

func TestVerify_FlagForReview_Bad_AllAPICallsFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]any{"message": "server error"})
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	// Should not panic when all API calls (ensureLabel, getLabelID, add label, comment) fail
	assert.NotPanics(t, func() {
		s.flagForReview("core", "test-repo", 42, testFailed)
	})
}

func TestVerify_FlagForReview_Ugly_LabelNotFoundZeroID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && containsStr(r.URL.Path, "/labels") {
			// getLabelID returns empty array → label ID is 0
			json.NewEncoder(w).Encode([]map[string]any{})
			return
		}
		// All other calls succeed
		w.WriteHeader(201)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	// label ID 0 is passed to "add labels" payload — should not panic
	assert.NotPanics(t, func() {
		s.flagForReview("core", "test-repo", 42, mergeConflict)
	})
}

// --- RunVerification Bad/Ugly ---

func TestVerify_RunVerification_Bad_GoModButNoGoFiles(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(filepath.Join(dir, "go.mod"), "module test\n\ngo 1.22").OK)
	// go.mod exists but no .go files — go test should fail

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	result := s.runVerification(dir)
	assert.Equal(t, "go test ./...", result.testCmd)
	// Depending on go version, this may pass (no test files = pass) or fail
	// The important thing is we detect Go project type correctly
}

func TestVerify_RunVerification_Ugly_MultipleProjectFiles(t *testing.T) {
	dir := t.TempDir()
	// Both go.mod and package.json exist — Go takes priority
	require.True(t, fs.Write(filepath.Join(dir, "go.mod"), "module test\n\ngo 1.22").OK)
	require.True(t, fs.Write(filepath.Join(dir, "package.json"), `{"scripts":{"test":"echo ok"}}`).OK)

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	result := s.runVerification(dir)
	// Go takes priority because it's checked first
	assert.Equal(t, "go test ./...", result.testCmd)
}

// --- additional: go.mod + composer.json to verify priority ---

func TestVerify_RunVerification_Ugly_GoAndPHPProjectFiles(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(filepath.Join(dir, "go.mod"), "module test\n\ngo 1.22").OK)
	require.True(t, fs.Write(filepath.Join(dir, "composer.json"), `{"require":{}}`).OK)

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	result := s.runVerification(dir)
	assert.Equal(t, "go test ./...", result.testCmd) // Go first in priority chain
}
