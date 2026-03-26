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

	core "dappco.re/go/core"
	"dappco.re/go/core/forge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- commentOnIssue ---

func TestPr_CommentOnIssue_Good_PostsCommentOnPR(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/issues/7/comments")

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Test comment", body["body"])

		json.NewEncoder(w).Encode(map[string]any{"id": 99})
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
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
			json.NewEncoder(w).Encode(map[string]any{"id": 1})
		default:
			w.WriteHeader(200)
		}
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	wsDir := filepath.Join(dir, "ws")
	repoDir := filepath.Join(wsDir, "repo")
	os.MkdirAll(repoDir, 0o755)

	// No go.mod, composer.json, or package.json = no test runner = passes
	st := &WorkspaceStatus{
		Status: "completed",
		Repo:   "test-repo",
		Org:    "core",
		Branch: "agent/fix",
		PRURL:  "https://forge.lthn.ai/core/test-repo/pulls/5",
	}
	data, _ := json.Marshal(st)
	os.WriteFile(filepath.Join(wsDir, "status.json"), data, 0o644)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	s.autoVerifyAndMerge(wsDir)
	assert.True(t, mergeOK, "should have called merge API")
	assert.True(t, commented, "should have posted comment")

	// Status should be marked as merged
	updated, err := ReadStatus(wsDir)
	require.NoError(t, err)
	assert.Equal(t, "merged", updated.Status)
}

// --- attemptVerifyAndMerge ---

func TestVerify_AttemptVerifyAndMerge_Good_TestsPassMergeSucceeds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/repos/core/test/pulls/1/merge" {
			w.WriteHeader(200)
		} else {
			json.NewEncoder(w).Encode(map[string]any{"id": 1})
		}
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir() // No project files = passes verification

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	result := s.attemptVerifyAndMerge(dir, "core", "test", "agent/fix", 1)
	assert.Equal(t, mergeSuccess, result)
}

func TestVerify_AttemptVerifyAndMerge_Bad_MergeFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/repos/core/test/pulls/1/merge" {
			w.WriteHeader(409)
			json.NewEncoder(w).Encode(map[string]any{"message": "conflict"})
		} else {
			json.NewEncoder(w).Encode(map[string]any{"id": 1})
		}
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	result := s.attemptVerifyAndMerge(dir, "core", "test", "agent/fix", 1)
	assert.Equal(t, mergeConflict, result)
}
