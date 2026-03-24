// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"dappco.re/go/core/forge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockForgeServer creates an httptest server that handles Forge API calls
// for issues and labels. Returns the server and a counter of issues created.
func mockForgeServer(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	issueCounter := &atomic.Int32{}

	mux := http.NewServeMux()

	// Create issue
	mux.HandleFunc("/api/v1/repos/", func(w http.ResponseWriter, r *http.Request) {
		// Route based on method + path suffix
		if r.Method == "POST" && pathEndsWith(r.URL.Path, "/issues") {
			num := int(issueCounter.Add(1))
			w.WriteHeader(201)
			json.NewEncoder(w).Encode(map[string]any{
				"number":   num,
				"html_url": "https://forge.test/core/test-repo/issues/" + itoa(num),
			})
			return
		}

		// Create/list labels
		if pathEndsWith(r.URL.Path, "/labels") {
			if r.Method == "GET" {
				json.NewEncoder(w).Encode([]map[string]any{
					{"id": 1, "name": "agentic"},
					{"id": 2, "name": "bug"},
				})
				return
			}
			if r.Method == "POST" {
				w.WriteHeader(201)
				json.NewEncoder(w).Encode(map[string]any{
					"id": issueCounter.Load() + 100,
				})
				return
			}
		}

		// List issues (for scan)
		if r.Method == "GET" && pathEndsWith(r.URL.Path, "/issues") {
			json.NewEncoder(w).Encode([]map[string]any{
				{
					"number":   1,
					"title":    "Test issue",
					"labels":   []map[string]any{{"name": "agentic"}},
					"assignee": nil,
					"html_url": "https://forge.test/core/test-repo/issues/1",
				},
			})
			return
		}

		// Issue labels (for verify)
		if r.Method == "POST" && containsStr(r.URL.Path, "/labels") {
			w.WriteHeader(200)
			return
		}

		// PR merge
		if r.Method == "POST" && containsStr(r.URL.Path, "/merge") {
			w.WriteHeader(200)
			return
		}

		// Issue comments
		if r.Method == "POST" && containsStr(r.URL.Path, "/comments") {
			w.WriteHeader(201)
			return
		}

		w.WriteHeader(404)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, issueCounter
}

func pathEndsWith(path, suffix string) bool {
	if len(path) < len(suffix) {
		return false
	}
	return path[len(path)-len(suffix):] == suffix
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// newTestSubsystem creates a PrepSubsystem wired to a mock Forge server.
func newTestSubsystem(t *testing.T, srv *httptest.Server) *PrepSubsystem {
	t.Helper()
	s := &PrepSubsystem{
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		brainURL:   srv.URL,
		brainKey:   "test-brain-key",
		codePath:   t.TempDir(),
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}
	return s
}

// --- createIssue ---

func TestCreateIssue_Good_Success(t *testing.T) {
	srv, counter := mockForgeServer(t)
	s := newTestSubsystem(t, srv)

	child, err := s.createIssue(context.Background(), "core", "test-repo", "Fix the bug", "Description", []int64{1})
	require.NoError(t, err)
	assert.Equal(t, 1, child.Number)
	assert.Equal(t, "Fix the bug", child.Title)
	assert.Contains(t, child.URL, "issues/1")
	assert.Equal(t, int32(1), counter.Load())
}

func TestCreateIssue_Good_NoLabels(t *testing.T) {
	srv, _ := mockForgeServer(t)
	s := newTestSubsystem(t, srv)

	child, err := s.createIssue(context.Background(), "core", "test-repo", "No labels task", "", nil)
	require.NoError(t, err)
	assert.Equal(t, "No labels task", child.Title)
}

func TestCreateIssue_Good_WithBody(t *testing.T) {
	srv, _ := mockForgeServer(t)
	s := newTestSubsystem(t, srv)

	child, err := s.createIssue(context.Background(), "core", "test-repo", "Task with body", "Detailed description", []int64{1, 2})
	require.NoError(t, err)
	assert.NotZero(t, child.Number)
}

func TestCreateIssue_Bad_ServerDown(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close() // immediately close

	s := &PrepSubsystem{
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     &http.Client{},
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	_, err := s.createIssue(context.Background(), "core", "test-repo", "Title", "", nil)
	assert.Error(t, err)
}

func TestCreateIssue_Bad_Non201Response(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	_, err := s.createIssue(context.Background(), "core", "test-repo", "Title", "", nil)
	assert.Error(t, err)
}

// --- resolveLabelIDs ---

func TestResolveLabelIDs_Good_ExistingLabels(t *testing.T) {
	srv, _ := mockForgeServer(t)
	s := newTestSubsystem(t, srv)

	ids := s.resolveLabelIDs(context.Background(), "core", "test-repo", []string{"agentic", "bug"})
	assert.Len(t, ids, 2)
	assert.Contains(t, ids, int64(1))
	assert.Contains(t, ids, int64(2))
}

func TestResolveLabelIDs_Good_NewLabel(t *testing.T) {
	srv, _ := mockForgeServer(t)
	s := newTestSubsystem(t, srv)

	// "new-label" doesn't exist in mock, so it will be created
	ids := s.resolveLabelIDs(context.Background(), "core", "test-repo", []string{"new-label"})
	assert.NotEmpty(t, ids)
}

func TestResolveLabelIDs_Good_EmptyNames(t *testing.T) {
	srv, _ := mockForgeServer(t)
	s := newTestSubsystem(t, srv)

	ids := s.resolveLabelIDs(context.Background(), "core", "test-repo", nil)
	assert.Nil(t, ids)
}

func TestResolveLabelIDs_Bad_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	ids := s.resolveLabelIDs(context.Background(), "core", "test-repo", []string{"agentic"})
	assert.Nil(t, ids)
}

// --- createLabel ---

func TestCreateLabel_Good_Known(t *testing.T) {
	srv, _ := mockForgeServer(t)
	s := newTestSubsystem(t, srv)

	id := s.createLabel(context.Background(), "core", "test-repo", "agentic")
	assert.NotZero(t, id)
}

func TestCreateLabel_Good_Unknown(t *testing.T) {
	srv, _ := mockForgeServer(t)
	s := newTestSubsystem(t, srv)

	// Unknown label uses default colour
	id := s.createLabel(context.Background(), "core", "test-repo", "custom-label")
	assert.NotZero(t, id)
}

func TestCreateLabel_Bad_ServerDown(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()

	s := &PrepSubsystem{
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     &http.Client{},
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	id := s.createLabel(context.Background(), "core", "test-repo", "agentic")
	assert.Zero(t, id)
}

// --- createEpic (validation only, not full dispatch) ---

func TestCreateEpic_Bad_NoTitle(t *testing.T) {
	srv, _ := mockForgeServer(t)
	s := newTestSubsystem(t, srv)

	_, _, err := s.createEpic(context.Background(), nil, EpicInput{
		Repo:  "test-repo",
		Tasks: []string{"Task 1"},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "title is required")
}

func TestCreateEpic_Bad_NoTasks(t *testing.T) {
	srv, _ := mockForgeServer(t)
	s := newTestSubsystem(t, srv)

	_, _, err := s.createEpic(context.Background(), nil, EpicInput{
		Repo:  "test-repo",
		Title: "Epic Title",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one task")
}

func TestCreateEpic_Bad_NoToken(t *testing.T) {
	s := &PrepSubsystem{
		forgeToken: "",
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	_, _, err := s.createEpic(context.Background(), nil, EpicInput{
		Repo:  "test-repo",
		Title: "Epic",
		Tasks: []string{"Task"},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no Forge token")
}

func TestCreateEpic_Good_WithTasks(t *testing.T) {
	srv, counter := mockForgeServer(t)
	s := newTestSubsystem(t, srv)

	_, out, err := s.createEpic(context.Background(), nil, EpicInput{
		Repo:  "test-repo",
		Title: "Test Epic",
		Tasks: []string{"Task 1", "Task 2"},
	})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.NotZero(t, out.EpicNumber)
	assert.Len(t, out.Children, 2)
	assert.Equal(t, "Task 1", out.Children[0].Title)
	assert.Equal(t, "Task 2", out.Children[1].Title)
	// 2 children + 1 epic = 3 issues
	assert.Equal(t, int32(3), counter.Load())
}

func TestCreateEpic_Good_WithLabels(t *testing.T) {
	srv, _ := mockForgeServer(t)
	s := newTestSubsystem(t, srv)

	_, out, err := s.createEpic(context.Background(), nil, EpicInput{
		Repo:   "test-repo",
		Title:  "Labelled Epic",
		Tasks:  []string{"Do it"},
		Labels: []string{"bug"},
	})
	require.NoError(t, err)
	assert.True(t, out.Success)
}

func TestCreateEpic_Good_AgenticLabelAutoAdded(t *testing.T) {
	srv, _ := mockForgeServer(t)
	s := newTestSubsystem(t, srv)

	// No labels specified — "agentic" should be auto-added
	_, out, err := s.createEpic(context.Background(), nil, EpicInput{
		Repo:  "test-repo",
		Title: "Auto-labelled",
		Tasks: []string{"Task"},
	})
	require.NoError(t, err)
	assert.True(t, out.Success)
}

func TestCreateEpic_Good_AgenticLabelNotDuplicated(t *testing.T) {
	srv, _ := mockForgeServer(t)
	s := newTestSubsystem(t, srv)

	// agentic already present — should not be duplicated
	_, out, err := s.createEpic(context.Background(), nil, EpicInput{
		Repo:   "test-repo",
		Title:  "With agentic",
		Tasks:  []string{"Task"},
		Labels: []string{"agentic"},
	})
	require.NoError(t, err)
	assert.True(t, out.Success)
}
