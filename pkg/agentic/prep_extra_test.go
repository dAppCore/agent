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
)

// --- Shutdown ---

func TestShutdown_Good(t *testing.T) {
	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	err := s.Shutdown(context.Background())
	assert.NoError(t, err)
}

// --- Name ---

func TestName_Good(t *testing.T) {
	s := &PrepSubsystem{}
	assert.Equal(t, "agentic", s.Name())
}

// --- findConsumersList ---

func TestFindConsumersList_Good_HasConsumers(t *testing.T) {
	dir := t.TempDir()

	// Create go.work
	goWork := `go 1.22

use (
	./core/go
	./core/agent
	./core/mcp
)`
	os.WriteFile(filepath.Join(dir, "go.work"), []byte(goWork), 0o644)

	// Create module dirs with go.mod
	for _, mod := range []struct {
		path    string
		content string
	}{
		{"core/go", "module forge.lthn.ai/core/go\n\ngo 1.22\n"},
		{"core/agent", "module forge.lthn.ai/core/agent\n\nrequire forge.lthn.ai/core/go v0.7.0\n"},
		{"core/mcp", "module forge.lthn.ai/core/mcp\n\nrequire forge.lthn.ai/core/go v0.7.0\n"},
	} {
		modDir := filepath.Join(dir, mod.path)
		os.MkdirAll(modDir, 0o755)
		os.WriteFile(filepath.Join(modDir, "go.mod"), []byte(mod.content), 0o644)
	}

	s := &PrepSubsystem{
		codePath:  dir,
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	list, count := s.findConsumersList("go")
	assert.Equal(t, 2, count)
	assert.Contains(t, list, "agent")
	assert.Contains(t, list, "mcp")
	assert.Contains(t, list, "Breaking change risk")
}

func TestFindConsumersList_Good_NoConsumers(t *testing.T) {
	dir := t.TempDir()

	goWork := `go 1.22

use (
	./core/go
)`
	os.WriteFile(filepath.Join(dir, "go.work"), []byte(goWork), 0o644)

	modDir := filepath.Join(dir, "core", "go")
	os.MkdirAll(modDir, 0o755)
	os.WriteFile(filepath.Join(modDir, "go.mod"), []byte("module forge.lthn.ai/core/go\n"), 0o644)

	s := &PrepSubsystem{
		codePath:  dir,
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	list, count := s.findConsumersList("go")
	assert.Equal(t, 0, count)
	assert.Empty(t, list)
}

func TestFindConsumersList_Bad_NoGoWork(t *testing.T) {
	s := &PrepSubsystem{
		codePath:  t.TempDir(),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	list, count := s.findConsumersList("go")
	assert.Equal(t, 0, count)
	assert.Empty(t, list)
}

// --- pullWikiContent ---

func TestPullWikiContent_Good_WithPages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/repos/core/go-io/wiki/pages":
			json.NewEncoder(w).Encode([]map[string]any{
				{"title": "Home", "sub_url": "Home"},
				{"title": "Architecture", "sub_url": "Architecture"},
			})
		case r.URL.Path == "/api/v1/repos/core/go-io/wiki/page/Home":
			// "Hello World" base64
			json.NewEncoder(w).Encode(map[string]any{
				"title":          "Home",
				"content_base64": "SGVsbG8gV29ybGQ=",
			})
		case r.URL.Path == "/api/v1/repos/core/go-io/wiki/page/Architecture":
			json.NewEncoder(w).Encode(map[string]any{
				"title":          "Architecture",
				"content_base64": "TGF5ZXJlZA==",
			})
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		forge:     forge.NewForge(srv.URL, "test-token"),
		client:    srv.Client(),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	content := s.pullWikiContent(context.Background(), "core", "go-io")
	assert.Contains(t, content, "Hello World")
	assert.Contains(t, content, "Layered")
	assert.Contains(t, content, "### Home")
	assert.Contains(t, content, "### Architecture")
}

func TestPullWikiContent_Good_NoPages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{})
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		forge:     forge.NewForge(srv.URL, "test-token"),
		client:    srv.Client(),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	content := s.pullWikiContent(context.Background(), "core", "go-io")
	assert.Empty(t, content)
}

// --- getIssueBody ---

func TestGetIssueBody_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"number": 15,
			"title":  "Fix tests",
			"body":   "The tests are broken in pkg/core",
		})
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		forge:     forge.NewForge(srv.URL, "test-token"),
		client:    srv.Client(),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	body := s.getIssueBody(context.Background(), "core", "go-io", 15)
	assert.Contains(t, body, "tests are broken")
}

func TestGetIssueBody_Bad_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		forge:     forge.NewForge(srv.URL, "test-token"),
		client:    srv.Client(),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	body := s.getIssueBody(context.Background(), "core", "go-io", 999)
	assert.Empty(t, body)
}

// --- buildPrompt ---

func TestBuildPrompt_Good_BasicFields(t *testing.T) {
	dir := t.TempDir()
	// Create go.mod to detect language
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n\ngo 1.22\n"), 0o644)

	s := &PrepSubsystem{
		codePath:  t.TempDir(),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
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

func TestBuildPrompt_Good_WithIssue(t *testing.T) {
	dir := t.TempDir()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"number": 42,
			"title":  "Bug report",
			"body":   "Steps to reproduce the bug",
		})
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		forge:     forge.NewForge(srv.URL, "test-token"),
		codePath:  t.TempDir(),
		client:    srv.Client(),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
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

// --- runQA ---

func TestRunQA_Good_PHPNoComposer(t *testing.T) {
	dir := t.TempDir()
	repoDir := filepath.Join(dir, "repo")
	os.MkdirAll(repoDir, 0o755)
	// composer.json present but no composer binary
	os.WriteFile(filepath.Join(repoDir, "composer.json"), []byte(`{"name":"test"}`), 0o644)

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	// Will fail (composer not found) — that's the expected path
	result := s.runQA(dir)
	assert.False(t, result)
}
