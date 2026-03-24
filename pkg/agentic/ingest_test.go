// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ingestFindings ---

func TestIngestFindings_Good_WithFindings(t *testing.T) {
	// Track the issue creation call
	issueCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && containsStr(r.URL.Path, "/issues") {
			issueCalled = true
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			assert.Contains(t, body["title"], "Scan findings")
			w.WriteHeader(201)
			return
		}
		w.WriteHeader(200)
	}))
	t.Cleanup(srv.Close)

	// Create a workspace with status and log file
	wsDir := t.TempDir()
	require.NoError(t, writeStatus(wsDir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Agent:  "codex",
	}))

	// Write a log file with file:line references
	logContent := "Found issues:\n" +
		"- `pkg/core/app.go:42` has an unused variable\n" +
		"- `pkg/core/service.go:100` has a missing error check\n" +
		"- `pkg/core/config.go:25` needs documentation\n" +
		"This is padding to get past the 100 char minimum length requirement for the log file content parsing."
	require.True(t, fs.Write(filepath.Join(wsDir, "agent-codex.log"), logContent).OK)

	// Set up HOME for the agent-api.key read
	home := t.TempDir()
	t.Setenv("DIR_HOME", home)
	require.True(t, fs.EnsureDir(filepath.Join(home, ".claude")).OK)
	require.True(t, fs.Write(filepath.Join(home, ".claude", "agent-api.key"), "test-api-key").OK)

	s := &PrepSubsystem{
		brainURL:   srv.URL,
		brainKey:   "test-brain-key",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	s.ingestFindings(wsDir)
	assert.True(t, issueCalled, "should have created an issue via API")
}

func TestIngestFindings_Bad_NotCompleted(t *testing.T) {
	wsDir := t.TempDir()
	require.NoError(t, writeStatus(wsDir, &WorkspaceStatus{
		Status: "running",
		Repo:   "go-io",
	}))

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	// Should return early — status is not "completed"
	assert.NotPanics(t, func() {
		s.ingestFindings(wsDir)
	})
}

func TestIngestFindings_Bad_NoLogFile(t *testing.T) {
	wsDir := t.TempDir()
	require.NoError(t, writeStatus(wsDir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
	}))

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	// Should return early — no log files
	assert.NotPanics(t, func() {
		s.ingestFindings(wsDir)
	})
}

func TestIngestFindings_Bad_TooFewFindings(t *testing.T) {
	wsDir := t.TempDir()
	require.NoError(t, writeStatus(wsDir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
	}))

	// Only 1 finding (need >= 2 to ingest)
	logContent := "Found: `main.go:1` has an issue. This padding makes the content long enough to pass the 100 char minimum check."
	require.True(t, fs.Write(filepath.Join(wsDir, "agent-codex.log"), logContent).OK)

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	assert.NotPanics(t, func() {
		s.ingestFindings(wsDir)
	})
}

func TestIngestFindings_Bad_QuotaExhausted(t *testing.T) {
	wsDir := t.TempDir()
	require.NoError(t, writeStatus(wsDir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
	}))

	// Log contains quota error — should skip
	logContent := "QUOTA_EXHAUSTED: Rate limit exceeded. `main.go:1` `other.go:2` padding to ensure we pass length check and get past the threshold."
	require.True(t, fs.Write(filepath.Join(wsDir, "agent-codex.log"), logContent).OK)

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	assert.NotPanics(t, func() {
		s.ingestFindings(wsDir)
	})
}

func TestIngestFindings_Bad_NoStatusFile(t *testing.T) {
	wsDir := t.TempDir()

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	assert.NotPanics(t, func() {
		s.ingestFindings(wsDir)
	})
}

func TestIngestFindings_Bad_ShortLogFile(t *testing.T) {
	wsDir := t.TempDir()
	require.NoError(t, writeStatus(wsDir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
	}))

	// Log content is less than 100 bytes — should skip
	require.True(t, fs.Write(filepath.Join(wsDir, "agent-codex.log"), "short").OK)

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	assert.NotPanics(t, func() {
		s.ingestFindings(wsDir)
	})
}

// --- createIssueViaAPI ---

func TestCreateIssueViaAPI_Good_Success(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/v1/issues")
		// Auth header should be present (Bearer + some key)
		assert.Contains(t, r.Header.Get("Authorization"), "Bearer ")

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Test Issue", body["title"])
		assert.Equal(t, "bug", body["type"])
		assert.Equal(t, "high", body["priority"])

		w.WriteHeader(201)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		brainURL:   srv.URL,
		brainKey:   "test-brain-key",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	s.createIssueViaAPI("go-io", "Test Issue", "Description", "bug", "high", "scan")
	assert.True(t, called)
}

func TestCreateIssueViaAPI_Bad_NoBrainKey(t *testing.T) {
	s := &PrepSubsystem{
		brainKey:  "",
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	// Should return early without panic
	assert.NotPanics(t, func() {
		s.createIssueViaAPI("go-io", "Title", "Body", "task", "normal", "scan")
	})
}

func TestCreateIssueViaAPI_Bad_NoAPIKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("DIR_HOME", home)
	// No agent-api.key file

	s := &PrepSubsystem{
		brainURL:   "https://example.com",
		brainKey:   "test-brain-key",
		client:     &http.Client{},
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	// Should return early — no API key file
	assert.NotPanics(t, func() {
		s.createIssueViaAPI("go-io", "Title", "Body", "task", "normal", "scan")
	})
}

func TestCreateIssueViaAPI_Bad_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	home := t.TempDir()
	t.Setenv("DIR_HOME", home)
	require.True(t, fs.EnsureDir(filepath.Join(home, ".claude")).OK)
	require.True(t, fs.Write(filepath.Join(home, ".claude", "agent-api.key"), "test-key").OK)

	s := &PrepSubsystem{
		brainURL:   srv.URL,
		brainKey:   "test-brain-key",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	// Should not panic even on server error
	assert.NotPanics(t, func() {
		s.createIssueViaAPI("go-io", "Title", "Body", "task", "normal", "scan")
	})
}

// --- countFileRefs (additional security-related) ---

func TestCountFileRefs_Good_SecurityFindings(t *testing.T) {
	body := "Security scan found:\n" +
		"- `pkg/auth/token.go:55` hardcoded secret\n" +
		"- `pkg/auth/middleware.go:12` missing auth check\n"
	assert.Equal(t, 2, countFileRefs(body))
}

func TestCountFileRefs_Good_PHPSecurityFindings(t *testing.T) {
	body := "PHP audit:\n" +
		"- `src/Controller/Api.php:42` SQL injection risk\n" +
		"- `src/Service/Auth.php:100` session fixation\n" +
		"- `src/Config/routes.php:5` open redirect\n"
	assert.Equal(t, 3, countFileRefs(body))
}
