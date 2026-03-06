package jobrunner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Journal: NewJournal error path ---

func TestNewJournal_Bad_EmptyBaseDir(t *testing.T) {
	_, err := NewJournal("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base directory is required")
}

func TestNewJournal_Good(t *testing.T) {
	dir := t.TempDir()
	j, err := NewJournal(dir)
	require.NoError(t, err)
	assert.NotNil(t, j)
}

// --- Journal: sanitizePathComponent additional cases ---

func TestSanitizePathComponent_Good_ValidNames(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"host-uk", "host-uk"},
		{"core", "core"},
		{"my_repo", "my_repo"},
		{"repo.v2", "repo.v2"},
		{"A123", "A123"},
	}

	for _, tc := range tests {
		got, err := sanitizePathComponent(tc.input)
		require.NoError(t, err, "input: %q", tc.input)
		assert.Equal(t, tc.want, got)
	}
}

func TestSanitizePathComponent_Bad_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"spaces", "   "},
		{"dotdot", ".."},
		{"dot", "."},
		{"slash", "foo/bar"},
		{"backslash", `foo\bar`},
		{"special", "org$bad"},
		{"leading-dot", ".hidden"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := sanitizePathComponent(tc.input)
			assert.Error(t, err, "input: %q", tc.input)
		})
	}
}

// --- Journal: Append with readonly directory ---

func TestJournal_Append_Bad_ReadonlyDir(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("chmod does not restrict root")
	}
	// Create a dir that we then make readonly (only works as non-root).
	dir := t.TempDir()
	readonlyDir := filepath.Join(dir, "readonly")
	require.NoError(t, os.MkdirAll(readonlyDir, 0o755))
	require.NoError(t, os.Chmod(readonlyDir, 0o444))
	t.Cleanup(func() { _ = os.Chmod(readonlyDir, 0o755) })

	j, err := NewJournal(readonlyDir)
	require.NoError(t, err)

	signal := &PipelineSignal{
		RepoOwner: "test-owner",
		RepoName:  "test-repo",
	}
	result := &ActionResult{
		Action:    "test",
		Timestamp: time.Now(),
	}

	err = j.Append(signal, result)
	// Should fail because MkdirAll cannot create subdirectories in readonly dir.
	assert.Error(t, err)
}

// --- Poller: error-returning source ---

type errorSource struct {
	name string
}

func (e *errorSource) Name() string { return e.name }
func (e *errorSource) Poll(_ context.Context) ([]*PipelineSignal, error) {
	return nil, fmt.Errorf("poll error")
}
func (e *errorSource) Report(_ context.Context, _ *ActionResult) error { return nil }

func TestPoller_RunOnce_Good_SourceError(t *testing.T) {
	src := &errorSource{name: "broken-source"}
	handler := &mockHandler{name: "test"}

	p := NewPoller(PollerConfig{
		Sources:  []JobSource{src},
		Handlers: []JobHandler{handler},
	})

	err := p.RunOnce(context.Background())
	require.NoError(t, err) // Poll errors are logged, not returned

	handler.mu.Lock()
	defer handler.mu.Unlock()
	assert.Empty(t, handler.executed, "handler should not be called when poll fails")
}

// --- Poller: error-returning handler ---

type errorHandler struct {
	name string
}

func (e *errorHandler) Name() string                  { return e.name }
func (e *errorHandler) Match(_ *PipelineSignal) bool   { return true }
func (e *errorHandler) Execute(_ context.Context, _ *PipelineSignal) (*ActionResult, error) {
	return nil, fmt.Errorf("handler error")
}

func TestPoller_RunOnce_Good_HandlerError(t *testing.T) {
	sig := &PipelineSignal{
		EpicNumber:  1,
		ChildNumber: 1,
		PRNumber:    1,
		RepoOwner:   "test",
		RepoName:    "repo",
	}

	src := &mockSource{
		name:    "test-source",
		signals: []*PipelineSignal{sig},
	}

	handler := &errorHandler{name: "broken-handler"}

	p := NewPoller(PollerConfig{
		Sources:  []JobSource{src},
		Handlers: []JobHandler{handler},
	})

	err := p.RunOnce(context.Background())
	require.NoError(t, err) // Handler errors are logged, not returned

	// Source should not have received a report (handler errored out).
	src.mu.Lock()
	defer src.mu.Unlock()
	assert.Empty(t, src.reports)
}

// --- Poller: with Journal integration ---

func TestPoller_RunOnce_Good_WithJournal(t *testing.T) {
	dir := t.TempDir()
	journal, err := NewJournal(dir)
	require.NoError(t, err)

	sig := &PipelineSignal{
		EpicNumber:  10,
		ChildNumber: 3,
		PRNumber:    55,
		RepoOwner:   "host-uk",
		RepoName:    "core",
		PRState:     "OPEN",
		CheckStatus: "SUCCESS",
		Mergeable:   "MERGEABLE",
	}

	src := &mockSource{
		name:    "test-source",
		signals: []*PipelineSignal{sig},
	}

	handler := &mockHandler{
		name: "test-handler",
		matchFn: func(s *PipelineSignal) bool {
			return true
		},
	}

	p := NewPoller(PollerConfig{
		Sources:  []JobSource{src},
		Handlers: []JobHandler{handler},
		Journal:  journal,
	})

	err = p.RunOnce(context.Background())
	require.NoError(t, err)

	handler.mu.Lock()
	require.Len(t, handler.executed, 1)
	handler.mu.Unlock()

	// Verify the journal file was written.
	date := time.Now().UTC().Format("2006-01-02")
	journalPath := filepath.Join(dir, "host-uk", "core", date+".jsonl")
	_, statErr := os.Stat(journalPath)
	assert.NoError(t, statErr, "journal file should exist at %s", journalPath)
}

// --- Poller: error-returning Report ---

type reportErrorSource struct {
	name    string
	signals []*PipelineSignal
	mu      sync.Mutex
}

func (r *reportErrorSource) Name() string { return r.name }
func (r *reportErrorSource) Poll(_ context.Context) ([]*PipelineSignal, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.signals, nil
}
func (r *reportErrorSource) Report(_ context.Context, _ *ActionResult) error {
	return fmt.Errorf("report error")
}

func TestPoller_RunOnce_Good_ReportError(t *testing.T) {
	sig := &PipelineSignal{
		EpicNumber:  1,
		ChildNumber: 1,
		PRNumber:    1,
		RepoOwner:   "test",
		RepoName:    "repo",
	}

	src := &reportErrorSource{
		name:    "report-fail-source",
		signals: []*PipelineSignal{sig},
	}

	handler := &mockHandler{
		name:    "test-handler",
		matchFn: func(s *PipelineSignal) bool { return true },
	}

	p := NewPoller(PollerConfig{
		Sources:  []JobSource{src},
		Handlers: []JobHandler{handler},
	})

	err := p.RunOnce(context.Background())
	require.NoError(t, err) // Report errors are logged, not returned

	handler.mu.Lock()
	defer handler.mu.Unlock()
	assert.Len(t, handler.executed, 1, "handler should still execute even though report fails")
}

// --- Poller: multiple sources and handlers ---

func TestPoller_RunOnce_Good_MultipleSources(t *testing.T) {
	sig1 := &PipelineSignal{
		EpicNumber: 1, ChildNumber: 1, PRNumber: 1,
		RepoOwner: "org1", RepoName: "repo1",
	}
	sig2 := &PipelineSignal{
		EpicNumber: 2, ChildNumber: 2, PRNumber: 2,
		RepoOwner: "org2", RepoName: "repo2",
	}

	src1 := &mockSource{name: "source-1", signals: []*PipelineSignal{sig1}}
	src2 := &mockSource{name: "source-2", signals: []*PipelineSignal{sig2}}

	handler := &mockHandler{
		name:    "catch-all",
		matchFn: func(s *PipelineSignal) bool { return true },
	}

	p := NewPoller(PollerConfig{
		Sources:  []JobSource{src1, src2},
		Handlers: []JobHandler{handler},
	})

	err := p.RunOnce(context.Background())
	require.NoError(t, err)

	handler.mu.Lock()
	defer handler.mu.Unlock()
	assert.Len(t, handler.executed, 2)
}

// --- Poller: Run with immediate cancellation ---

func TestPoller_Run_Good_ImmediateCancel(t *testing.T) {
	src := &mockSource{name: "source", signals: nil}

	p := NewPoller(PollerConfig{
		Sources:      []JobSource{src},
		PollInterval: 1 * time.Hour, // long interval
	})

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel after the first RunOnce completes.
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := p.Run(ctx)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, p.Cycle()) // One cycle from the initial RunOnce
}

// --- Journal: Append with journal error logging ---

func TestPoller_RunOnce_Good_JournalAppendError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("chmod does not restrict root")
	}
	// Use a directory that will cause journal writes to fail.
	dir := t.TempDir()
	journal, err := NewJournal(dir)
	require.NoError(t, err)

	// Make the journal directory read-only to trigger append errors.
	require.NoError(t, os.Chmod(dir, 0o444))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	sig := &PipelineSignal{
		EpicNumber:  1,
		ChildNumber: 1,
		PRNumber:    1,
		RepoOwner:   "test",
		RepoName:    "repo",
	}

	src := &mockSource{
		name:    "test-source",
		signals: []*PipelineSignal{sig},
	}

	handler := &mockHandler{
		name:    "test-handler",
		matchFn: func(s *PipelineSignal) bool { return true },
	}

	p := NewPoller(PollerConfig{
		Sources:  []JobSource{src},
		Handlers: []JobHandler{handler},
		Journal:  journal,
	})

	err = p.RunOnce(context.Background())
	// Journal errors are logged, not returned.
	require.NoError(t, err)

	handler.mu.Lock()
	defer handler.mu.Unlock()
	assert.Len(t, handler.executed, 1, "handler should still execute even when journal fails")
}

// --- Poller: Cycle counter increments ---

func TestPoller_Cycle_Good_Increments(t *testing.T) {
	src := &mockSource{name: "source", signals: nil}

	p := NewPoller(PollerConfig{
		Sources: []JobSource{src},
	})

	assert.Equal(t, 0, p.Cycle())

	_ = p.RunOnce(context.Background())
	assert.Equal(t, 1, p.Cycle())

	_ = p.RunOnce(context.Background())
	assert.Equal(t, 2, p.Cycle())
}
