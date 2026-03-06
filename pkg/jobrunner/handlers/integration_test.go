package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"forge.lthn.ai/core/go-scm/forge"
	"forge.lthn.ai/core/agent/pkg/jobrunner"
)

// --- Integration: full signal -> handler -> result flow ---
// These tests exercise the complete pipeline: a signal is created,
// matched by a handler, executed against a mock Forgejo server,
// and the result is verified.

// mockForgejoServer creates a comprehensive mock Forgejo API server
// for integration testing. It supports issues, PRs, labels, comments,
// and tracks all API calls made.
type apiCall struct {
	Method string
	Path   string
	Body   string
}

type forgejoMock struct {
	epicBody    string
	calls       []apiCall
	srv         *httptest.Server
	closedChild bool
	editedBody  string
	comments    []string
}

func newForgejoMock(t *testing.T, epicBody string) *forgejoMock {
	t.Helper()
	m := &forgejoMock{
		epicBody: epicBody,
	}

	m.srv = httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		m.calls = append(m.calls, apiCall{
			Method: r.Method,
			Path:   r.URL.Path,
			Body:   string(bodyBytes),
		})

		w.Header().Set("Content-Type", "application/json")
		path := r.URL.Path

		switch {
		// GET epic issue.
		case r.Method == http.MethodGet && strings.Contains(path, "/issues/") && !strings.Contains(path, "/comments"):
			issueNum := path[strings.LastIndex(path, "/")+1:]
			_ = json.NewEncoder(w).Encode(map[string]any{
				"number": json.Number(issueNum),
				"body":   m.epicBody,
				"title":  "Epic: Phase 3",
				"state":  "open",
				"labels": []map[string]any{{"name": "epic", "id": 1}},
			})

		// PATCH epic issue (edit body or close child).
		case r.Method == http.MethodPatch && strings.Contains(path, "/issues/"):
			var body map[string]any
			_ = json.Unmarshal(bodyBytes, &body)

			if bodyStr, ok := body["body"].(string); ok {
				m.editedBody = bodyStr
			}
			if state, ok := body["state"].(string); ok && state == "closed" {
				m.closedChild = true
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"number": 1,
				"body":   m.editedBody,
				"state":  "open",
			})

		// POST comment.
		case r.Method == http.MethodPost && strings.Contains(path, "/comments"):
			var body map[string]string
			_ = json.Unmarshal(bodyBytes, &body)
			m.comments = append(m.comments, body["body"])
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "body": body["body"]})

		// GET labels.
		case r.Method == http.MethodGet && strings.Contains(path, "/labels"):
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": 1, "name": "epic", "color": "#ff0000"},
				{"id": 2, "name": "in-progress", "color": "#1d76db"},
			})

		// POST labels.
		case r.Method == http.MethodPost && strings.HasSuffix(path, "/labels"):
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 10, "name": "new-label"})

		// POST issue labels.
		case r.Method == http.MethodPost && strings.Contains(path, "/issues/") && strings.Contains(path, "/labels"):
			_ = json.NewEncoder(w).Encode([]map[string]any{})

		// DELETE issue label.
		case r.Method == http.MethodDelete && strings.Contains(path, "/labels/"):
			w.WriteHeader(http.StatusNoContent)

		// POST merge PR.
		case r.Method == http.MethodPost && strings.Contains(path, "/merge"):
			w.WriteHeader(http.StatusOK)

		// PATCH PR (publish draft).
		case r.Method == http.MethodPatch && strings.Contains(path, "/pulls/"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))

		// GET reviews.
		case r.Method == http.MethodGet && strings.Contains(path, "/reviews"):
			_ = json.NewEncoder(w).Encode([]map[string]any{})

		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{})
		}
	})))

	return m
}

func (m *forgejoMock) close() {
	m.srv.Close()
}

func (m *forgejoMock) client(t *testing.T) *forge.Client {
	t.Helper()
	c, err := forge.New(m.srv.URL, "test-token")
	require.NoError(t, err)
	return c
}

// --- TickParent integration: signal -> execute -> verify epic updated ---

func TestIntegration_TickParent_Good_FullFlow(t *testing.T) {
	epicBody := "## Tasks\n- [x] #1\n- [ ] #7\n- [ ] #8\n- [x] #3\n"

	mock := newForgejoMock(t, epicBody)
	defer mock.close()

	h := NewTickParentHandler(mock.client(t))

	// Create signal representing a merged PR for child #7.
	signal := &jobrunner.PipelineSignal{
		EpicNumber:  42,
		ChildNumber: 7,
		PRNumber:    99,
		RepoOwner:   "host-uk",
		RepoName:    "core-php",
		PRState:     "MERGED",
		CheckStatus: "SUCCESS",
		Mergeable:   "UNKNOWN",
	}

	// Verify the handler matches.
	assert.True(t, h.Match(signal))

	// Execute.
	result, err := h.Execute(context.Background(), signal)
	require.NoError(t, err)

	// Verify result.
	assert.True(t, result.Success)
	assert.Equal(t, "tick_parent", result.Action)
	assert.Equal(t, "host-uk", result.RepoOwner)
	assert.Equal(t, "core-php", result.RepoName)
	assert.Equal(t, 99, result.PRNumber)

	// Verify the epic body was updated: #7 should now be checked.
	assert.Contains(t, mock.editedBody, "- [x] #7")
	// #8 should still be unchecked.
	assert.Contains(t, mock.editedBody, "- [ ] #8")
	// #1 and #3 should remain checked.
	assert.Contains(t, mock.editedBody, "- [x] #1")
	assert.Contains(t, mock.editedBody, "- [x] #3")

	// Verify the child issue was closed.
	assert.True(t, mock.closedChild)
}

// --- TickParent integration: epic progress tracking ---

func TestIntegration_TickParent_Good_TrackEpicProgress(t *testing.T) {
	// Start with 4 tasks, 1 checked.
	epicBody := "## Tasks\n- [x] #1\n- [ ] #2\n- [ ] #3\n- [ ] #4\n"

	mock := newForgejoMock(t, epicBody)
	defer mock.close()

	h := NewTickParentHandler(mock.client(t))

	// Tick child #2.
	signal := &jobrunner.PipelineSignal{
		EpicNumber:  10,
		ChildNumber: 2,
		PRNumber:    20,
		RepoOwner:   "org",
		RepoName:    "repo",
		PRState:     "MERGED",
	}

	result, err := h.Execute(context.Background(), signal)
	require.NoError(t, err)
	assert.True(t, result.Success)

	// Verify #2 is now checked.
	assert.Contains(t, mock.editedBody, "- [x] #2")
	// #3 and #4 should still be unchecked.
	assert.Contains(t, mock.editedBody, "- [ ] #3")
	assert.Contains(t, mock.editedBody, "- [ ] #4")

	// Count progress: 2 out of 4 now checked.
	checked := strings.Count(mock.editedBody, "- [x]")
	unchecked := strings.Count(mock.editedBody, "- [ ]")
	assert.Equal(t, 2, checked)
	assert.Equal(t, 2, unchecked)
}

// --- EnableAutoMerge integration: full flow ---

func TestIntegration_EnableAutoMerge_Good_FullFlow(t *testing.T) {
	var mergeMethod string

	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/merge") {
			bodyBytes, _ := io.ReadAll(r.Body)
			var body map[string]any
			_ = json.Unmarshal(bodyBytes, &body)
			if do, ok := body["Do"].(string); ok {
				mergeMethod = do
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
	})))
	defer srv.Close()

	client := newTestForgeClient(t, srv.URL)
	h := NewEnableAutoMergeHandler(client)

	signal := &jobrunner.PipelineSignal{
		EpicNumber:  1,
		ChildNumber: 5,
		PRNumber:    42,
		RepoOwner:   "host-uk",
		RepoName:    "core-tenant",
		PRState:     "OPEN",
		IsDraft:     false,
		Mergeable:   "MERGEABLE",
		CheckStatus: "SUCCESS",
	}

	// Verify match.
	assert.True(t, h.Match(signal))

	// Execute.
	result, err := h.Execute(context.Background(), signal)
	require.NoError(t, err)

	assert.True(t, result.Success)
	assert.Equal(t, "enable_auto_merge", result.Action)
	assert.Equal(t, "host-uk", result.RepoOwner)
	assert.Equal(t, "core-tenant", result.RepoName)
	assert.Equal(t, 42, result.PRNumber)
	assert.Equal(t, "squash", mergeMethod)
}

// --- PublishDraft integration: full flow ---

func TestIntegration_PublishDraft_Good_FullFlow(t *testing.T) {
	var patchedDraft bool

	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodPatch && strings.Contains(r.URL.Path, "/pulls/") {
			bodyBytes, _ := io.ReadAll(r.Body)
			if strings.Contains(string(bodyBytes), `"draft":false`) {
				patchedDraft = true
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	})))
	defer srv.Close()

	client := newTestForgeClient(t, srv.URL)
	h := NewPublishDraftHandler(client)

	signal := &jobrunner.PipelineSignal{
		EpicNumber:  3,
		ChildNumber: 8,
		PRNumber:    15,
		RepoOwner:   "org",
		RepoName:    "repo",
		PRState:     "OPEN",
		IsDraft:     true,
		CheckStatus: "SUCCESS",
		Mergeable:   "MERGEABLE",
	}

	// Verify match.
	assert.True(t, h.Match(signal))

	// Execute.
	result, err := h.Execute(context.Background(), signal)
	require.NoError(t, err)

	assert.True(t, result.Success)
	assert.Equal(t, "publish_draft", result.Action)
	assert.True(t, patchedDraft)
}

// --- SendFixCommand integration: conflict message ---

func TestIntegration_SendFixCommand_Good_ConflictFlow(t *testing.T) {
	var commentBody string

	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/comments") {
			bodyBytes, _ := io.ReadAll(r.Body)
			var body map[string]string
			_ = json.Unmarshal(bodyBytes, &body)
			commentBody = body["body"]
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 1})
			return
		}
		w.WriteHeader(http.StatusOK)
	})))
	defer srv.Close()

	client := newTestForgeClient(t, srv.URL)
	h := NewSendFixCommandHandler(client)

	signal := &jobrunner.PipelineSignal{
		EpicNumber:  1,
		ChildNumber: 3,
		PRNumber:    10,
		RepoOwner:   "org",
		RepoName:    "repo",
		PRState:     "OPEN",
		Mergeable:   "CONFLICTING",
		CheckStatus: "SUCCESS",
	}

	assert.True(t, h.Match(signal))

	result, err := h.Execute(context.Background(), signal)
	require.NoError(t, err)

	assert.True(t, result.Success)
	assert.Equal(t, "send_fix_command", result.Action)
	assert.Contains(t, commentBody, "fix the merge conflict")
}

// --- SendFixCommand integration: code review message ---

func TestIntegration_SendFixCommand_Good_ReviewFlow(t *testing.T) {
	var commentBody string

	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/comments") {
			bodyBytes, _ := io.ReadAll(r.Body)
			var body map[string]string
			_ = json.Unmarshal(bodyBytes, &body)
			commentBody = body["body"]
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 1})
			return
		}
		w.WriteHeader(http.StatusOK)
	})))
	defer srv.Close()

	client := newTestForgeClient(t, srv.URL)
	h := NewSendFixCommandHandler(client)

	signal := &jobrunner.PipelineSignal{
		EpicNumber:      1,
		ChildNumber:     3,
		PRNumber:        10,
		RepoOwner:       "org",
		RepoName:        "repo",
		PRState:         "OPEN",
		Mergeable:       "MERGEABLE",
		CheckStatus:     "FAILURE",
		ThreadsTotal:    3,
		ThreadsResolved: 1,
	}

	assert.True(t, h.Match(signal))

	result, err := h.Execute(context.Background(), signal)
	require.NoError(t, err)

	assert.True(t, result.Success)
	assert.Contains(t, commentBody, "fix the code reviews")
}

// --- Completion integration: success flow ---

func TestIntegration_Completion_Good_SuccessFlow(t *testing.T) {
	var labelAdded bool
	var labelRemoved bool
	var commentBody string

	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		// GetLabelByName — GET repo labels.
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/core/go-scm/labels":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": 1, "name": "in-progress", "color": "#1d76db"},
			})

		// RemoveIssueLabel.
		case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/labels/"):
			labelRemoved = true
			w.WriteHeader(http.StatusNoContent)

		// EnsureLabel — POST to create repo label.
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/core/go-scm/labels":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 2, "name": "agent-completed", "color": "#0e8a16"})

		// AddIssueLabels — POST to issue labels.
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/core/go-scm/issues/12/labels":
			labelAdded = true
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 2, "name": "agent-completed"}})

		// CreateIssueComment.
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/comments"):
			bodyBytes, _ := io.ReadAll(r.Body)
			var body map[string]string
			_ = json.Unmarshal(bodyBytes, &body)
			commentBody = body["body"]
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 1})

		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{})
		}
	})))
	defer srv.Close()

	client := newTestForgeClient(t, srv.URL)
	h := NewCompletionHandler(client)

	signal := &jobrunner.PipelineSignal{
		Type:        "agent_completion",
		EpicNumber:  5,
		ChildNumber: 12,
		RepoOwner:   "core",
		RepoName:    "go-scm",
		Success:     true,
		Message:     "PR created and tests passing",
	}

	assert.True(t, h.Match(signal))

	result, err := h.Execute(context.Background(), signal)
	require.NoError(t, err)

	assert.True(t, result.Success)
	assert.Equal(t, "completion", result.Action)
	assert.Equal(t, "core", result.RepoOwner)
	assert.Equal(t, "go-scm", result.RepoName)
	assert.Equal(t, 5, result.EpicNumber)
	assert.Equal(t, 12, result.ChildNumber)
	assert.True(t, labelRemoved, "in-progress label should be removed")
	assert.True(t, labelAdded, "agent-completed label should be added")
	assert.Contains(t, commentBody, "PR created and tests passing")
}

// --- Full pipeline integration: signal -> match -> execute -> journal ---

func TestIntegration_FullPipeline_Good_TickParentWithJournal(t *testing.T) {
	epicBody := "## Tasks\n- [ ] #7\n- [ ] #8\n"

	mock := newForgejoMock(t, epicBody)
	defer mock.close()

	dir := t.TempDir()
	journal, err := jobrunner.NewJournal(dir)
	require.NoError(t, err)

	client := mock.client(t)
	h := NewTickParentHandler(client)

	signal := &jobrunner.PipelineSignal{
		EpicNumber:  10,
		ChildNumber: 7,
		PRNumber:    55,
		RepoOwner:   "host-uk",
		RepoName:    "core-tenant",
		PRState:     "MERGED",
		CheckStatus: "SUCCESS",
		Mergeable:   "UNKNOWN",
	}

	// Verify match.
	assert.True(t, h.Match(signal))

	// Execute.
	start := time.Now()
	result, err := h.Execute(context.Background(), signal)
	require.NoError(t, err)

	assert.True(t, result.Success)

	// Write to journal (simulating what the poller does).
	result.EpicNumber = signal.EpicNumber
	result.ChildNumber = signal.ChildNumber
	result.Cycle = 1
	result.Duration = time.Since(start)

	err = journal.Append(signal, result)
	require.NoError(t, err)

	// Verify the journal file exists and contains the entry.
	date := time.Now().UTC().Format("2006-01-02")
	journalPath := filepath.Join(dir, "host-uk", "core-tenant", date+".jsonl")

	_, statErr := os.Stat(journalPath)
	require.NoError(t, statErr)

	f, err := os.Open(journalPath)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	var entry jobrunner.JournalEntry
	err = json.NewDecoder(f).Decode(&entry)
	require.NoError(t, err)

	assert.Equal(t, "tick_parent", entry.Action)
	assert.Equal(t, "host-uk/core-tenant", entry.Repo)
	assert.Equal(t, 10, entry.Epic)
	assert.Equal(t, 7, entry.Child)
	assert.Equal(t, 55, entry.PR)
	assert.Equal(t, 1, entry.Cycle)
	assert.True(t, entry.Result.Success)
	assert.Equal(t, "MERGED", entry.Signals.PRState)

	// Verify the epic was properly updated.
	assert.Contains(t, mock.editedBody, "- [x] #7")
	assert.Contains(t, mock.editedBody, "- [ ] #8")
	assert.True(t, mock.closedChild)
}

// --- Handler matching priority: first match wins ---

func TestIntegration_HandlerPriority_Good_FirstMatchWins(t *testing.T) {
	// Test that when multiple handlers could match, the first one wins.
	// This exercises the poller's findHandler logic.

	// Signal with OPEN, not draft, MERGEABLE, SUCCESS, no threads:
	// This matches enable_auto_merge.
	signal := &jobrunner.PipelineSignal{
		PRState:         "OPEN",
		IsDraft:         false,
		Mergeable:       "MERGEABLE",
		CheckStatus:     "SUCCESS",
		ThreadsTotal:    0,
		ThreadsResolved: 0,
	}

	autoMerge := NewEnableAutoMergeHandler(nil)
	publishDraft := NewPublishDraftHandler(nil)
	fixCommand := NewSendFixCommandHandler(nil)

	// enable_auto_merge should match.
	assert.True(t, autoMerge.Match(signal))
	// publish_draft should NOT match (not a draft).
	assert.False(t, publishDraft.Match(signal))
	// send_fix_command should NOT match (mergeable and passing).
	assert.False(t, fixCommand.Match(signal))
}

// --- Handler matching: draft PR path ---

func TestIntegration_HandlerPriority_Good_DraftPRPath(t *testing.T) {
	signal := &jobrunner.PipelineSignal{
		PRState:         "OPEN",
		IsDraft:         true,
		Mergeable:       "MERGEABLE",
		CheckStatus:     "SUCCESS",
		ThreadsTotal:    0,
		ThreadsResolved: 0,
	}

	autoMerge := NewEnableAutoMergeHandler(nil)
	publishDraft := NewPublishDraftHandler(nil)
	fixCommand := NewSendFixCommandHandler(nil)

	// enable_auto_merge should NOT match (is draft).
	assert.False(t, autoMerge.Match(signal))
	// publish_draft should match (draft + open + success).
	assert.True(t, publishDraft.Match(signal))
	// send_fix_command should NOT match.
	assert.False(t, fixCommand.Match(signal))
}

// --- Handler matching: merged PR only matches tick_parent ---

func TestIntegration_HandlerPriority_Good_MergedPRPath(t *testing.T) {
	signal := &jobrunner.PipelineSignal{
		PRState:         "MERGED",
		IsDraft:         false,
		Mergeable:       "UNKNOWN",
		CheckStatus:     "SUCCESS",
		ThreadsTotal:    0,
		ThreadsResolved: 0,
	}

	autoMerge := NewEnableAutoMergeHandler(nil)
	publishDraft := NewPublishDraftHandler(nil)
	fixCommand := NewSendFixCommandHandler(nil)
	tickParent := NewTickParentHandler(nil)

	assert.False(t, autoMerge.Match(signal))
	assert.False(t, publishDraft.Match(signal))
	assert.False(t, fixCommand.Match(signal))
	assert.True(t, tickParent.Match(signal))
}

// --- Handler matching: conflicting PR matches send_fix_command ---

func TestIntegration_HandlerPriority_Good_ConflictingPRPath(t *testing.T) {
	signal := &jobrunner.PipelineSignal{
		PRState:         "OPEN",
		IsDraft:         false,
		Mergeable:       "CONFLICTING",
		CheckStatus:     "SUCCESS",
		ThreadsTotal:    0,
		ThreadsResolved: 0,
	}

	autoMerge := NewEnableAutoMergeHandler(nil)
	fixCommand := NewSendFixCommandHandler(nil)

	// enable_auto_merge should NOT match (conflicting).
	assert.False(t, autoMerge.Match(signal))
	// send_fix_command should match (conflicting).
	assert.True(t, fixCommand.Match(signal))
}

// --- Completion integration: failure flow ---

func TestIntegration_Completion_Good_FailureFlow(t *testing.T) {
	var commentBody string

	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		// GetLabelByName — GET repo labels.
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/core/go-scm/labels":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": 1, "name": "in-progress", "color": "#1d76db"},
			})

		// RemoveIssueLabel.
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)

		// EnsureLabel — POST to create repo label.
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/core/go-scm/labels":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 3, "name": "agent-failed", "color": "#c0392b"})

		// AddIssueLabels — POST to issue labels.
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/core/go-scm/issues/12/labels":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 3, "name": "agent-failed"}})

		// CreateIssueComment.
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/comments"):
			bodyBytes, _ := io.ReadAll(r.Body)
			var body map[string]string
			_ = json.Unmarshal(bodyBytes, &body)
			commentBody = body["body"]
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 1})

		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{})
		}
	})))
	defer srv.Close()

	client := newTestForgeClient(t, srv.URL)
	h := NewCompletionHandler(client)

	signal := &jobrunner.PipelineSignal{
		Type:        "agent_completion",
		EpicNumber:  5,
		ChildNumber: 12,
		RepoOwner:   "core",
		RepoName:    "go-scm",
		Success:     false,
		Error:       "tests failed: 3 assertions",
	}

	result, err := h.Execute(context.Background(), signal)
	require.NoError(t, err)

	assert.True(t, result.Success) // The handler itself succeeded.
	assert.Contains(t, commentBody, "Agent reported failure")
	assert.Contains(t, commentBody, "tests failed: 3 assertions")
}

// --- Multiple handlers execute in sequence for different signals ---

func TestIntegration_MultipleHandlers_Good_DifferentSignals(t *testing.T) {
	var commentBodies []string
	var mergedPRs []int64

	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/merge"):
			// Extract PR number from path.
			parts := strings.Split(r.URL.Path, "/")
			for i, p := range parts {
				if p == "pulls" && i+1 < len(parts) {
					var prNum int64
					_ = json.Unmarshal([]byte(parts[i+1]), &prNum)
					mergedPRs = append(mergedPRs, prNum)
				}
			}
			w.WriteHeader(http.StatusOK)

		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/comments"):
			bodyBytes, _ := io.ReadAll(r.Body)
			var body map[string]string
			_ = json.Unmarshal(bodyBytes, &body)
			commentBodies = append(commentBodies, body["body"])
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 1})

		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/issues/"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"number": 42,
				"body":   "## Tasks\n- [ ] #7\n- [ ] #8\n",
				"title":  "Epic",
			})

		case r.Method == http.MethodPatch:
			_ = json.NewEncoder(w).Encode(map[string]any{"number": 1, "body": "", "state": "open"})

		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{})
		}
	})))
	defer srv.Close()

	client := newTestForgeClient(t, srv.URL)

	autoMergeHandler := NewEnableAutoMergeHandler(client)
	fixCommandHandler := NewSendFixCommandHandler(client)

	// Signal 1: should trigger auto merge.
	sig1 := &jobrunner.PipelineSignal{
		PRState: "OPEN", IsDraft: false, Mergeable: "MERGEABLE",
		CheckStatus: "SUCCESS", PRNumber: 10,
		RepoOwner: "org", RepoName: "repo",
	}

	// Signal 2: should trigger fix command.
	sig2 := &jobrunner.PipelineSignal{
		PRState: "OPEN", Mergeable: "CONFLICTING",
		CheckStatus: "SUCCESS", PRNumber: 20,
		RepoOwner: "org", RepoName: "repo",
	}

	assert.True(t, autoMergeHandler.Match(sig1))
	assert.False(t, autoMergeHandler.Match(sig2))

	assert.False(t, fixCommandHandler.Match(sig1))
	assert.True(t, fixCommandHandler.Match(sig2))

	// Execute both.
	result1, err := autoMergeHandler.Execute(context.Background(), sig1)
	require.NoError(t, err)
	assert.True(t, result1.Success)

	result2, err := fixCommandHandler.Execute(context.Background(), sig2)
	require.NoError(t, err)
	assert.True(t, result2.Success)

	// Verify correct comment was posted for the conflicting PR.
	require.Len(t, commentBodies, 1)
	assert.Contains(t, commentBodies[0], "fix the merge conflict")
}
