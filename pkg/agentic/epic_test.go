// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	core "dappco.re/go"
	"dappco.re/go/forge"
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
			w.Write([]byte(core.JSONMarshalString(map[string]any{
				"number":   num,
				"html_url": "https://forge.test/core/test-repo/issues/" + itoa(num),
			})))
			return
		}

		// Create/list labels
		if pathEndsWith(r.URL.Path, "/labels") {
			if r.Method == "GET" {
				w.Write([]byte(core.JSONMarshalString([]map[string]any{
					{"id": 1, "name": "agentic"},
					{"id": 2, "name": "bug"},
				})))
				return
			}
			if r.Method == "POST" {
				w.WriteHeader(201)
				w.Write([]byte(core.JSONMarshalString(map[string]any{
					"id": issueCounter.Load() + 100,
				})))
				return
			}
		}

		// List issues (for scan)
		if r.Method == "GET" && pathEndsWith(r.URL.Path, "/issues") {
			w.Write([]byte(core.JSONMarshalString([]map[string]any{
				{
					"number":   1,
					"title":    "Test issue",
					"labels":   []map[string]any{{"name": "agentic"}},
					"assignee": nil,
					"html_url": "https://forge.test/core/test-repo/issues/1",
				},
			})))
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
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		brainURL:       srv.URL,
		brainKey:       "test-brain-key",
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	return s
}

// --- createIssue ---

func TestEpic_CreateIssue_Good_Success(t *testing.T) {
	srv, counter := mockForgeServer(t)
	s := newTestSubsystem(t, srv)

	child, err := s.createIssue(context.Background(), "core", "test-repo", "Fix the bug", "Description", []int64{1})
	core.RequireNoError(t, err)
	core.AssertEqual(t, 1, child.Number)
	core.AssertEqual(t, "Fix the bug", child.Title)
	core.AssertContains(t, child.URL, "issues/1")
	core.AssertEqual(t, int32(1), counter.Load())
}

func TestEpic_CreateIssue_Good_NoLabels(t *testing.T) {
	srv, _ := mockForgeServer(t)
	s := newTestSubsystem(t, srv)

	child, err := s.createIssue(context.Background(), "core", "test-repo", "No labels task", "", nil)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "No labels task", child.Title)
}

func TestEpic_CreateIssue_Good_WithBody(t *testing.T) {
	srv, _ := mockForgeServer(t)
	s := newTestSubsystem(t, srv)

	child, err := s.createIssue(context.Background(), "core", "test-repo", "Task with body", "Detailed description", []int64{1, 2})
	core.RequireNoError(t, err)
	assertNotZero(t, child.Number)
}

func TestEpic_CreateIssue_Bad_ServerDown(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close() // immediately close

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, err := s.createIssue(context.Background(), "core", "test-repo", "Title", "", nil)
	core.AssertError(t, err)
}

func TestEpic_CreateIssue_Bad_Non201Response(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, err := s.createIssue(context.Background(), "core", "test-repo", "Title", "", nil)
	core.AssertError(t, err)
}

// --- resolveLabelIDs ---

func TestEpic_ResolveLabelIDs_Good_ExistingLabels(t *testing.T) {
	srv, _ := mockForgeServer(t)
	s := newTestSubsystem(t, srv)

	ids := s.resolveLabelIDs(context.Background(), "core", "test-repo", []string{"agentic", "bug"})
	core.AssertLen(t, ids, 2)
	core.AssertContains(t, ids, int64(1))
	core.AssertContains(t, ids, int64(2))
}

func TestEpic_ResolveLabelIDs_Good_NewLabel(t *testing.T) {
	srv, _ := mockForgeServer(t)
	s := newTestSubsystem(t, srv)

	// "new-label" doesn't exist in mock, so it will be created
	ids := s.resolveLabelIDs(context.Background(), "core", "test-repo", []string{"new-label"})
	core.AssertNotEmpty(t, ids)
}

func TestEpic_ResolveLabelIDs_Good_EmptyNames(t *testing.T) {
	srv, _ := mockForgeServer(t)
	s := newTestSubsystem(t, srv)

	ids := s.resolveLabelIDs(context.Background(), "core", "test-repo", nil)
	core.AssertNil(t, ids)
}

func TestEpic_ResolveLabelIDs_Bad_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	ids := s.resolveLabelIDs(context.Background(), "core", "test-repo", []string{"agentic"})
	core.AssertNil(t, ids)
}

// --- createLabel ---

func TestEpic_CreateLabel_Good_Known(t *testing.T) {
	srv, _ := mockForgeServer(t)
	s := newTestSubsystem(t, srv)

	id := s.createLabel(context.Background(), "core", "test-repo", "agentic")
	assertNotZero(t, id)
}

func TestEpic_CreateLabel_Good_Unknown(t *testing.T) {
	srv, _ := mockForgeServer(t)
	s := newTestSubsystem(t, srv)

	// Unknown label uses default colour
	id := s.createLabel(context.Background(), "core", "test-repo", "custom-label")
	assertNotZero(t, id)
}

func TestEpic_CreateLabel_Bad_ServerDown(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	id := s.createLabel(context.Background(), "core", "test-repo", "agentic")
	assertZero(t, id)
}

// --- createEpic (validation only, not full dispatch) ---

func TestEpic_CreateEpic_Bad_NoTitle(t *testing.T) {
	srv, _ := mockForgeServer(t)
	s := newTestSubsystem(t, srv)

	_, _, err := createEpic(s, context.Background(), nil, EpicInput{
		Repo:  "test-repo",
		Tasks: []string{"Task 1"},
	})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "title is required")
}

func TestEpic_CreateEpic_Bad_NoTasks(t *testing.T) {
	srv, _ := mockForgeServer(t)
	s := newTestSubsystem(t, srv)

	_, _, err := createEpic(s, context.Background(), nil, EpicInput{
		Repo:  "test-repo",
		Title: "Epic Title",
	})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "at least one task")
}

func TestEpic_CreateEpic_Bad_NoToken(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeToken:     "",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, _, err := createEpic(s, context.Background(), nil, EpicInput{
		Repo:  "test-repo",
		Title: "Epic",
		Tasks: []string{"Task"},
	})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "no Forge token")
}

func TestEpic_CreateEpic_Good_WithTasks(t *testing.T) {
	srv, counter := mockForgeServer(t)
	s := newTestSubsystem(t, srv)

	_, out, err := createEpic(s, context.Background(), nil, EpicInput{
		Repo:  "test-repo",
		Title: "Test Epic",
		Tasks: []string{"Task 1", "Task 2"},
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	assertNotZero(t, out.EpicNumber)
	core.AssertLen(t, out.Children, 2)
	core.AssertEqual(t, "Task 1", out.Children[0].Title)
	core.AssertEqual(t, "Task 2", out.Children[1].Title)
	// 2 children + 1 epic = 3 issues
	core.AssertEqual(t, int32(3), counter.Load())
}

func TestEpic_CreateEpic_Good_WithLabels(t *testing.T) {
	srv, _ := mockForgeServer(t)
	s := newTestSubsystem(t, srv)

	_, out, err := createEpic(s, context.Background(), nil, EpicInput{
		Repo:   "test-repo",
		Title:  "Labelled Epic",
		Tasks:  []string{"Do it"},
		Labels: []string{"bug"},
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
}

func TestEpic_CreateEpic_Good_AgenticLabelAutoAdded(t *testing.T) {
	srv, _ := mockForgeServer(t)
	s := newTestSubsystem(t, srv)

	// No labels specified — "agentic" should be auto-added
	_, out, err := createEpic(s, context.Background(), nil, EpicInput{
		Repo:  "test-repo",
		Title: "Auto-labelled",
		Tasks: []string{"Task"},
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
}

func TestEpic_CreateEpic_Good_AgenticLabelNotDuplicated(t *testing.T) {
	srv, _ := mockForgeServer(t)
	s := newTestSubsystem(t, srv)

	// agentic already present — should not be duplicated
	_, out, err := createEpic(s, context.Background(), nil, EpicInput{
		Repo:   "test-repo",
		Title:  "With agentic",
		Tasks:  []string{"Task"},
		Labels: []string{"agentic"},
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
}

// --- Ugly tests ---

func TestEpic_CreateEpic_Ugly_Case(t *testing.T) {
	// Very long title/description
	srv, _ := mockForgeServer(t)
	s := newTestSubsystem(t, srv)

	longTitle := repeatString("Very Long Epic Title ", 50)
	longBody := repeatString("Detailed description of the epic work to be done. ", 100)

	_, out, err := createEpic(s, context.Background(), nil, EpicInput{
		Repo:  "test-repo",
		Title: longTitle,
		Body:  longBody,
		Tasks: []string{"Task 1"},
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	assertNotZero(t, out.EpicNumber)
}

func TestEpic_CreateIssue_Ugly_Case(t *testing.T) {
	// Issue with HTML in body
	srv, _ := mockForgeServer(t)
	s := newTestSubsystem(t, srv)

	htmlBody := "<h1>Issue</h1><p>This has <b>bold</b> and <script>alert('xss')</script></p>"
	child, err := s.createIssue(context.Background(), "core", "test-repo", "HTML Issue", htmlBody, []int64{1})
	core.RequireNoError(t, err)
	core.AssertEqual(t, "HTML Issue", child.Title)
	assertNotZero(t, child.Number)
}

func TestEpic_ResolveLabelIDs_Ugly_Case(t *testing.T) {
	// Label names with special chars
	srv, _ := mockForgeServer(t)
	s := newTestSubsystem(t, srv)

	ids := s.resolveLabelIDs(context.Background(), "core", "test-repo", []string{"bug/fix", "feature:new", "label with spaces"})
	// These will all be created as new labels since they don't match existing ones
	core.AssertNotNil(t, ids)
}

func TestEpic_CreateLabel_Ugly_Case(t *testing.T) {
	// Label with unicode name
	srv, _ := mockForgeServer(t)
	s := newTestSubsystem(t, srv)

	id := s.createLabel(context.Background(), "core", "test-repo", "\u00e9nhancement-\u00fc\u00f1ic\u00f6de")
	assertNotZero(t, id)
}
