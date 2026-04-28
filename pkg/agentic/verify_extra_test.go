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

// --- commentOnIssue ---

func TestPr_CommentOnIssue_Good_PostsCommentOnPR(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "POST", r.Method)
		core.AssertContains(t, r.URL.Path, "/issues/7/comments")

		var body map[string]string
		bodyStr := core.ReadAll(r.Body)
		core.JSONUnmarshalString(bodyStr.Value.(string), &body)
		core.AssertEqual(t, "Test comment", body["body"])

		w.Write([]byte(core.JSONMarshalString(map[string]any{"id": 99})))
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

	s.commentOnIssue(context.Background(), "core", "repo", 7, "Test comment")
}

// --- autoVerifyAndMerge integration (extended) ---

func TestVerify_AutoVerifyAndMerge_Good_FullPipeline(t *testing.T) {
	// Mock Forge API for merge + comment
	mergeOK := false
	commented := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/api/v1/repos/core/test-repo/pulls/5/merge":
			mergeOK = true
			w.WriteHeader(200)
		case r.Method == "POST" && r.URL.Path == "/api/v1/repos/core/test-repo/issues/5/comments":
			commented = true
			w.Write([]byte(core.JSONMarshalString(map[string]any{"id": 1})))
		default:
			w.WriteHeader(200)
		}
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	wsDir := core.JoinPath(dir, "ws")
	repoDir := core.JoinPath(wsDir, "repo")
	fs.EnsureDir(repoDir)

	// No go.mod, composer.json, or package.json = no test runner = passes
	st := &WorkspaceStatus{
		Status: "completed",
		Repo:   "test-repo",
		Org:    "core",
		Branch: "agent/fix",
		PRURL:  "https://forge.lthn.ai/core/test-repo/pulls/5",
	}
	fs.Write(core.JoinPath(wsDir, "status.json"), core.JSONMarshalString(st))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	s.autoVerifyAndMerge(wsDir)
	core.AssertTrue(t, mergeOK, "should have called merge API")
	core.AssertTrue(t, commented, "should have posted comment")

	// Status should be marked as merged
	updated := mustReadStatus(t, wsDir)
	core.AssertEqual(t, "merged", updated.Status)
}

// --- attemptVerifyAndMerge ---

func TestVerify_AttemptVerifyAndMerge_Good_TestsPassMergeSucceeds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/repos/core/test/pulls/1/merge" {
			w.WriteHeader(200)
		} else {
			w.Write([]byte(core.JSONMarshalString(map[string]any{"id": 1})))
		}
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir() // No project files = passes verification

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	result := s.attemptVerifyAndMerge(dir, "core", "test", "agent/fix", 1)
	core.AssertEqual(t, mergeSuccess, result)
}

func TestVerify_AttemptVerifyAndMerge_Bad_MergeFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/repos/core/test/pulls/1/merge" {
			w.WriteHeader(409)
			w.Write([]byte(core.JSONMarshalString(map[string]any{"message": "conflict"})))
		} else {
			w.Write([]byte(core.JSONMarshalString(map[string]any{"id": 1})))
		}
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	result := s.attemptVerifyAndMerge(dir, "core", "test", "agent/fix", 1)
	core.AssertEqual(t, mergeConflict, result)
}
