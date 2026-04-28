// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core "dappco.re/go"
)

// --- ingestFindings ---

func TestIngest_IngestFindings_Good_WithFindings(t *testing.T) {
	// Track the issue creation call
	issueCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && containsStr(r.URL.Path, "/issues") {
			issueCalled = true
			var body map[string]string
			core.JSONUnmarshalString(core.ReadAll(r.Body).Value.(string), &body)
			core.AssertContains(t, body["title"], "Scan findings")
			w.WriteHeader(201)
			return
		}
		w.WriteHeader(200)
	}))
	t.Cleanup(srv.Close)

	// Create a workspace with status and log file
	wsDir := t.TempDir()
	core.RequireNoError(t, writeStatus(wsDir, &WorkspaceStatus{
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
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(wsDir, ".meta")).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(wsDir, ".meta", "agent-codex.log"), logContent).OK)

	// Set up HOME for the agent-api.key read
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CORE_HOME", home)
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(home, ".claude")).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(home, ".claude", "agent-api.key"), "test-api-key").OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:       srv.URL,
		brainKey:       "test-brain-key",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	s.ingestFindings(wsDir)
	core.AssertTrue(t, issueCalled, "should have created an issue via API")
}

func TestIngest_IngestFindings_Bad_NotCompleted(t *testing.T) {
	wsDir := t.TempDir()
	core.RequireNoError(t, writeStatus(wsDir, &WorkspaceStatus{
		Status: "running",
		Repo:   "go-io",
	}))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Should return early — status is not "completed"
	core.AssertNotPanics(t, func() {
		s.ingestFindings(wsDir)
	})
}

func TestIngest_IngestFindings_Bad_NoLogFile(t *testing.T) {
	wsDir := t.TempDir()
	core.RequireNoError(t, writeStatus(wsDir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
	}))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Should return early — no log files
	core.AssertNotPanics(t, func() {
		s.ingestFindings(wsDir)
	})
}

func TestIngest_IngestFindings_Bad_TooFewFindings(t *testing.T) {
	wsDir := t.TempDir()
	core.RequireNoError(t, writeStatus(wsDir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
	}))

	// Only 1 finding (need >= 2 to ingest)
	logContent := "Found: `main.go:1` has an issue. This padding makes the content long enough to pass the 100 char minimum check."
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(wsDir, ".meta")).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(wsDir, ".meta", "agent-codex.log"), logContent).OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	core.AssertNotPanics(t, func() {
		s.ingestFindings(wsDir)
	})
}

func TestIngest_IngestFindings_Bad_QuotaExhausted(t *testing.T) {
	wsDir := t.TempDir()
	core.RequireNoError(t, writeStatus(wsDir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
	}))

	// Log contains quota error — should skip
	logContent := "QUOTA_EXHAUSTED: Rate limit exceeded. `main.go:1` `other.go:2` padding to ensure we pass length check and get past the threshold."
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(wsDir, ".meta")).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(wsDir, ".meta", "agent-codex.log"), logContent).OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	core.AssertNotPanics(t, func() {
		s.ingestFindings(wsDir)
	})
}

func TestIngest_IngestFindings_Bad_NoStatusFile(t *testing.T) {
	wsDir := t.TempDir()

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	core.AssertNotPanics(t, func() {
		s.ingestFindings(wsDir)
	})
}

func TestIngest_IngestFindings_Bad_ShortLogFile(t *testing.T) {
	wsDir := t.TempDir()
	core.RequireNoError(t, writeStatus(wsDir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
	}))

	// Log content is less than 100 bytes — should skip
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(wsDir, ".meta")).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(wsDir, ".meta", "agent-codex.log"), "short").OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	core.AssertNotPanics(t, func() {
		s.ingestFindings(wsDir)
	})
}

// --- createIssueViaAPI ---

func TestIngest_CreateIssueViaAPI_Good_Success(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		core.AssertEqual(t, "POST", r.Method)
		core.AssertContains(t, r.URL.Path, "/v1/issues")
		// Auth header should be present (Bearer + some key)
		core.AssertContains(t, r.Header.Get("Authorization"), "Bearer ")

		var body map[string]string
		core.JSONUnmarshalString(core.ReadAll(r.Body).Value.(string), &body)
		core.AssertEqual(t, "Test Issue", body["title"])
		core.AssertEqual(t, "bug", body["type"])
		core.AssertEqual(t, "high", body["priority"])

		w.WriteHeader(201)
	}))
	t.Cleanup(srv.Close)

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CORE_HOME", home)
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(home, ".claude")).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(home, ".claude", "agent-api.key"), "test-key").OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:       srv.URL,
		brainKey:       "test-brain-key",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	s.createIssueViaAPI("Test Issue", "Description", "bug", "high")
	core.AssertTrue(t, called)
}

func TestIngest_CreateIssueViaAPI_Bad_NoBrainKey(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainKey:       "",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Should return early without panic
	core.AssertNotPanics(t, func() {
		s.createIssueViaAPI("Title", "Body", "task", "normal")
	})
}

func TestIngest_CreateIssueViaAPI_Bad_NoAPIKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CORE_HOME", home)
	// No agent-api.key file

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:       "https://example.com",
		brainKey:       "test-brain-key",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Should return early — no API key file
	core.AssertNotPanics(t, func() {
		s.createIssueViaAPI("Title", "Body", "task", "normal")
	})
}

func TestIngest_CreateIssueViaAPI_Bad_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CORE_HOME", home)
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(home, ".claude")).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(home, ".claude", "agent-api.key"), "test-key").OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:       srv.URL,
		brainKey:       "test-brain-key",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Should not panic even on server error
	core.AssertNotPanics(t, func() {
		s.createIssueViaAPI("Title", "Body", "task", "normal")
	})
}

// --- countFileRefs (additional security-related) ---

func TestIngest_CountFileRefs_Good_SecurityFindings(t *testing.T) {
	body := "Security scan found:\n" +
		"- `pkg/auth/token.go:55` hardcoded secret\n" +
		"- `pkg/auth/middleware.go:12` missing auth check\n"
	core.AssertEqual(t, 2, countFileRefs(body))
}

// --- IngestFindings Ugly ---

func TestIngest_IngestFindings_Ugly(t *testing.T) {
	// Workspace with no findings file (completed but empty meta dir)
	wsDir := t.TempDir()
	core.RequireNoError(t, writeStatus(wsDir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Agent:  "codex",
	}))
	// No agent-*.log files at all

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Should return early without panic — no log files
	core.AssertNotPanics(t, func() {
		s.ingestFindings(wsDir)
	})
}

// --- CreateIssueViaAPI Ugly ---

func TestIngest_CreateIssueViaAPI_Ugly(t *testing.T) {
	// Issue body with HTML injection chars — should be passed as-is without panic
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		var body map[string]string
		core.JSONUnmarshalString(core.ReadAll(r.Body).Value.(string), &body)
		// Verify the body preserved HTML chars
		core.AssertContains(t, body["description"], "<script>")
		core.AssertContains(t, body["description"], "alert('xss')")
		w.WriteHeader(201)
	}))
	t.Cleanup(srv.Close)

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CORE_HOME", home)
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(home, ".claude")).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(home, ".claude", "agent-api.key"), "test-key").OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:       srv.URL,
		brainKey:       "test-brain-key",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	s.createIssueViaAPI("XSS Test", "<script>alert('xss')</script><b>bold</b>&amp;", "bug", "high")
	core.AssertTrue(t, called)
}

func TestIngest_CountFileRefs_Good_PHPSecurityFindings(t *testing.T) {
	body := "PHP audit:\n" +
		"- `src/Controller/Api.php:42` SQL injection risk\n" +
		"- `src/Service/Auth.php:100` session fixation\n" +
		"- `src/Config/routes.php:5` open redirect\n"
	core.AssertEqual(t, 3, countFileRefs(body))
}
