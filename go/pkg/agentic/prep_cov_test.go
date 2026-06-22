// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core "dappco.re/go"
)

// TestPrepCov_WritePromptSnapshot_Good_NoopOnBlankInput — a blank workspace dir
// or blank prompt is a silent OK no-op that writes nothing.
func TestPrepCov_WritePromptSnapshot_Good_NoopOnBlankInput(t *testing.T) {
	core.AssertTrue(t, writePromptSnapshot("", "TASK: x").OK)

	workspaceDir := t.TempDir()
	core.AssertTrue(t, writePromptSnapshot(workspaceDir, "   ").OK)
	// Nothing was written: the meta dir's prompt-version.json is absent.
	core.AssertFalse(t, fs.Exists(core.JoinPath(WorkspaceMetaDir(workspaceDir), "prompt-version.json")))
}

// TestPrepCov_WritePromptSnapshot_Good_SecondCallReusesSnapshot — calling twice
// with the same prompt re-uses the existing immutable snapshot file (the
// fs.Exists(snapshotPath) true branch) while still refreshing the index.
func TestPrepCov_WritePromptSnapshot_Good_SecondCallReusesSnapshot(t *testing.T) {
	workspaceDir := t.TempDir()
	prompt := "TASK: cover writePromptSnapshot reuse\n\nRead the RFC."

	first := writePromptSnapshot(workspaceDir, prompt)
	core.RequireTrue(t, first.OK)
	hash, ok := first.Value.(string)
	core.RequireTrue(t, ok)

	snapshotPath := core.JoinPath(WorkspaceMetaDir(workspaceDir), "prompt-versions", core.Concat(hash, ".json"))
	core.RequireTrue(t, fs.Exists(snapshotPath))

	// Second call with identical content takes the already-exists path.
	second := writePromptSnapshot(workspaceDir, prompt)
	core.RequireTrue(t, second.OK)
	core.AssertEqual(t, hash, second.Value.(string))

	// The persisted snapshot round-trips back through readPromptSnapshot.
	snapshot, err := readPromptSnapshot(workspaceDir)
	core.RequireNoError(t, err)
	core.AssertEqual(t, hash, snapshot.Hash)
	core.AssertEqual(t, prompt, snapshot.Content)
}

// TestPrepCov_WritePromptSnapshot_Ugly_EnsureDirFails — when the .meta path is
// occupied by a regular file the snapshot-directory creation fails and the
// error is returned.
func TestPrepCov_WritePromptSnapshot_Ugly_EnsureDirFails(t *testing.T) {
	workspaceDir := t.TempDir()
	// Occupy the meta dir path with a file so EnsureDir(.meta/prompt-versions) fails.
	core.RequireTrue(t, fs.Write(WorkspaceMetaDir(workspaceDir), "not a directory").OK)

	result := writePromptSnapshot(workspaceDir, "TASK: trigger the ensure-dir failure")
	core.AssertFalse(t, result.OK)

	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	// fs.EnsureDir surfaces the raw mkdir error when the .meta path is a file.
	core.AssertContains(t, err.Error(), "not a directory")
}

// TestPrepCov_BuildPrompt_Good_IncludesIssueAndGitLog — buildPrompt injects the
// fetched issue body and a recent-changes git log when an issue number and a
// real git checkout are supplied.
func TestPrepCov_BuildPrompt_Good_IncludesIssueAndGitLog(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(core.JSONMarshalString(map[string]any{
			"number": 42,
			"title":  "Fix the broken build",
			"body":   "The workspace build fails on a stale pin.",
		})))
	}))
	t.Cleanup(srv.Close)

	repoDir := prepCovGitRepo(t)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          newForgeClient(srv.URL, "test-token"),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	prompt, memories, consumers := s.buildPrompt(context.Background(), PrepInput{
		Task:  "Fix the build",
		Org:   "core",
		Repo:  "go-io",
		Issue: 42,
	}, "dev", repoDir)

	core.AssertContains(t, prompt, "TASK: Fix the build")
	core.AssertContains(t, prompt, "ISSUE:")
	core.AssertContains(t, prompt, "Fix the broken build")
	core.AssertContains(t, prompt, "RECENT CHANGES:")
	core.AssertEqual(t, 0, memories)
	core.AssertEqual(t, 0, consumers)
}

// TestPrepCov_BuildPrompt_Good_IncludesBrainContextAndConsumers — buildPrompt
// folds in OpenBrain recalled memories (with a non-zero memory count) and the
// consumer list derived from the workspace go.work.
func TestPrepCov_BuildPrompt_Good_IncludesBrainContextAndConsumers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/brain/recall", r.URL.Path)
		_, _ = w.Write([]byte(`{"memories":[{"type":"architecture","project":"go-io","content":"Uses Core.Process for all IO."}]}`))
	}))
	t.Cleanup(srv.Close)

	codePath := t.TempDir()
	// A consumer module that requires dappco.re/go/go-io.
	consumerDir := core.JoinPath(codePath, "consumer")
	core.RequireTrue(t, fs.EnsureDir(consumerDir).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(consumerDir, "go.mod"),
		"module dappco.re/go/consumer\n\ngo 1.26\n\nrequire dappco.re/go/go-io v0.0.0\n").OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(codePath, "go.work"),
		"go 1.26\n\nuse (\n\t./consumer\n)\n").OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:       srv.URL,
		brainKey:       "brain-key",
		codePath:       codePath,
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	prompt, memories, consumers := s.buildPrompt(context.Background(), PrepInput{
		Task: "Update IO paths",
		Org:  "core",
		Repo: "go-io",
	}, "dev", t.TempDir())

	core.AssertContains(t, prompt, "CONTEXT (from OpenBrain):")
	core.AssertContains(t, prompt, "Uses Core.Process for all IO.")
	core.AssertEqual(t, 1, memories)

	core.AssertContains(t, prompt, "CONSUMERS (modules that import this repo):")
	core.AssertContains(t, prompt, "- consumer")
	core.AssertEqual(t, 1, consumers)
}

// TestPrepCov_BrainRecall_Bad_NoKey — with no brain key brainRecall returns an
// empty context and zero count without any request.
func TestPrepCov_BrainRecall_Bad_NoKey(t *testing.T) {
	s := &PrepSubsystem{}
	recall, count := s.brainRecall(context.Background(), "go-io")
	core.AssertEqual(t, "", recall)
	core.AssertEqual(t, 0, count)
}

// TestPrepCov_BrainRecall_Bad_RecallRequestFails — a failing recall endpoint
// yields an empty context (the !r.OK arm).
func TestPrepCov_BrainRecall_Bad_RecallRequestFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "down", http.StatusBadRequest)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{brainURL: srv.URL, brainKey: "brain-key"}
	recall, count := s.brainRecall(context.Background(), "go-io")
	core.AssertEqual(t, "", recall)
	core.AssertEqual(t, 0, count)
}

// TestPrepCov_PullWikiContent_Ugly_SkipsEmptyBase64 — a wiki page whose
// content_base64 is empty is skipped, leaving the aggregate empty when it is
// the only page.
func TestPrepCov_PullWikiContent_Ugly_SkipsEmptyBase64(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/repos/core/go-io/wiki/pages":
			_, _ = w.Write([]byte(core.JSONMarshalString([]map[string]any{
				{"title": "Empty", "sub_url": "Empty"},
			})))
		case "/api/v1/repos/core/go-io/wiki/page/Empty":
			_, _ = w.Write([]byte(core.JSONMarshalString(map[string]any{
				"title":          "Empty",
				"content_base64": "",
			})))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          newForgeClient(srv.URL, "test-token"),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	content := s.pullWikiContent(context.Background(), "core", "go-io")
	core.AssertEmpty(t, content)
}

// TestPrepCov_PullWikiContent_Ugly_SkipsFailedPageFetch — a page whose detail
// fetch fails is skipped while a sibling page still contributes.
func TestPrepCov_PullWikiContent_Ugly_SkipsFailedPageFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/repos/core/go-io/wiki/pages":
			_, _ = w.Write([]byte(core.JSONMarshalString([]map[string]any{
				{"title": "Broken", "sub_url": "Broken"},
				{"title": "Good", "sub_url": "Good"},
			})))
		case "/api/v1/repos/core/go-io/wiki/page/Broken":
			w.WriteHeader(http.StatusInternalServerError)
		case "/api/v1/repos/core/go-io/wiki/page/Good":
			_, _ = w.Write([]byte(core.JSONMarshalString(map[string]any{
				"title":          "Good",
				"content_base64": "R29vZCBwYWdl", // "Good page"
			})))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          newForgeClient(srv.URL, "test-token"),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	content := s.pullWikiContent(context.Background(), "core", "go-io")
	core.AssertContains(t, content, "Good page")
	core.AssertNotContains(t, content, "Broken")
}

// prepCovGitRepo creates a tiny git checkout with one commit so getGitLog has a
// non-empty log to return.
func prepCovGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	core.RequireTrue(t, testCore.Process().RunIn(ctx, dir, "git", "init").OK)
	core.RequireTrue(t, testCore.Process().RunIn(ctx, dir, "git", "config", "user.name", "Test").OK)
	core.RequireTrue(t, testCore.Process().RunIn(ctx, dir, "git", "config", "user.email", "test@example.com").OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "go.mod"), "module test\n\ngo 1.26\n").OK)
	core.RequireTrue(t, testCore.Process().RunIn(ctx, dir, "git", "add", ".").OK)
	core.RequireTrue(t, testCore.Process().RunIn(ctx, dir, "git", "commit", "-m", "feat: initial commit").OK)
	return dir
}
