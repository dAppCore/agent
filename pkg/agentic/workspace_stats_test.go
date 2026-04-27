// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
)

func TestWorkspacestats_ExtractModelFromAgent_Good(t *testing.T) {
	assert.Equal(t, "gpt-5.4-mini", extractModelFromAgent("codex:gpt-5.4-mini"))
	assert.Equal(t, "sonnet", extractModelFromAgent("claude:sonnet"))
}

func TestWorkspacestats_ExtractModelFromAgent_Bad_NoColon(t *testing.T) {
	assert.Equal(t, "", extractModelFromAgent("codex"))
}

func TestWorkspacestats_ExtractModelFromAgent_Ugly_EmptyAndMultipleColons(t *testing.T) {
	assert.Equal(t, "", extractModelFromAgent(""))
	// Multiple colons — the model preserves the remainder unchanged.
	assert.Equal(t, "gpt:5.4:mini", extractModelFromAgent("codex:gpt:5.4:mini"))
}

func TestWorkspacestats_DispatchDurationMS_Good(t *testing.T) {
	started := time.Now()
	updated := started.Add(2500 * time.Millisecond)
	assert.Equal(t, int64(2500), dispatchDurationMS(started, updated))
}

func TestWorkspacestats_DispatchDurationMS_Bad_ZeroStart(t *testing.T) {
	assert.Equal(t, int64(0), dispatchDurationMS(time.Time{}, time.Now()))
}

func TestWorkspacestats_DispatchDurationMS_Ugly_UpdatedBeforeStarted(t *testing.T) {
	started := time.Now()
	updated := started.Add(-5 * time.Second)
	// When UpdatedAt is before StartedAt we return 0 rather than a negative value.
	assert.Equal(t, int64(0), dispatchDurationMS(started, updated))
}

func TestWorkspacestats_CountFindingsBy_Good(t *testing.T) {
	findings := []map[string]any{
		{"severity": "error", "tool": "gosec"},
		{"severity": "error", "tool": "gosec"},
		{"severity": "warning", "tool": "golangci-lint"},
	}
	counts := countFindingsBy(findings, "severity")
	assert.Equal(t, 2, counts["error"])
	assert.Equal(t, 1, counts["warning"])
}

func TestWorkspacestats_CountFindingsBy_Bad_EmptySlice(t *testing.T) {
	assert.Nil(t, countFindingsBy(nil, "severity"))
	assert.Nil(t, countFindingsBy([]map[string]any{}, "severity"))
}

func TestWorkspacestats_CountFindingsBy_Ugly_MissingFieldValues(t *testing.T) {
	findings := []map[string]any{
		{"severity": "error"},
		{"severity": ""},
		{"severity": nil},
		{"tool": "gosec"}, // no severity at all
	}
	counts := countFindingsBy(findings, "severity")
	assert.Equal(t, 1, counts["error"])
	// Empty and missing values are skipped, so the map only holds "error".
	assert.Equal(t, 1, len(counts))
}

func TestWorkspacestats_BuildWorkspaceStatsRecord_Good_FromStatus(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	workspaceDir := core.JoinPath(root, "workspace", "core", "go-io", "task-5")
	fs.EnsureDir(workspaceDir)

	started := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	updated := started.Add(3500 * time.Millisecond)

	record := buildWorkspaceStatsRecord(workspaceDir, &WorkspaceStatus{
		Repo:      "go-io",
		Org:       "core",
		Branch:    "agent/task-5",
		Agent:     "codex:gpt-5.4-mini",
		Task:      "fix the thing",
		Status:    "completed",
		Runs:      2,
		StartedAt: started,
		UpdatedAt: updated,
	})

	assert.Equal(t, "core/go-io/task-5", record.Workspace)
	assert.Equal(t, "go-io", record.Repo)
	assert.Equal(t, "agent/task-5", record.Branch)
	assert.Equal(t, "codex:gpt-5.4-mini", record.Agent)
	assert.Equal(t, "gpt-5.4-mini", record.Model)
	assert.Equal(t, "completed", record.Status)
	assert.Equal(t, 2, record.Runs)
	assert.Equal(t, int64(3500), record.DurationMS)
	assert.NotEmpty(t, record.CompletedAt)
}

func TestWorkspacestats_BuildWorkspaceStatsRecord_Good_FromReport(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	workspaceDir := core.JoinPath(root, "workspace", "core", "go-io", "task-5")
	metaDir := core.JoinPath(workspaceDir, ".meta")
	fs.EnsureDir(metaDir)

	report := map[string]any{
		"passed":       true,
		"build_passed": true,
		"test_passed":  true,
		"lint_passed":  true,
		"findings": []any{
			map[string]any{"severity": "error", "tool": "gosec", "category": "security"},
			map[string]any{"severity": "warning", "tool": "golangci-lint", "category": "style"},
		},
		"clusters":   []any{map[string]any{"tool": "gosec"}},
		"new":        []any{map[string]any{"tool": "gosec"}},
		"resolved":   []any{map[string]any{"tool": "golangci-lint"}},
		"persistent": []any{},
		"changes":    map[string]any{"insertions": 12, "deletions": 3, "files_changed": 2},
	}
	fs.WriteAtomic(core.JoinPath(metaDir, "report.json"), core.JSONMarshalString(report))

	record := buildWorkspaceStatsRecord(workspaceDir, &WorkspaceStatus{
		Repo:   "go-io",
		Org:    "core",
		Branch: "agent/task-5",
		Agent:  "codex:gpt-5.4",
		Status: "completed",
	})

	assert.True(t, record.Passed)
	assert.True(t, record.BuildPassed)
	assert.True(t, record.TestPassed)
	assert.True(t, record.LintPassed)
	assert.Equal(t, 2, record.FindingsTotal)
	assert.Equal(t, 1, record.BySeverity["error"])
	assert.Equal(t, 1, record.BySeverity["warning"])
	assert.Equal(t, 1, record.ByTool["gosec"])
	assert.Equal(t, 1, record.ByTool["golangci-lint"])
	assert.Equal(t, 1, record.ClustersCount)
	assert.Equal(t, 1, record.NewCount)
	assert.Equal(t, 1, record.ResolvedCount)
	assert.Equal(t, 0, record.PersistentCount)
	assert.Equal(t, 12, record.Insertions)
	assert.Equal(t, 3, record.Deletions)
	assert.Equal(t, 2, record.FilesChanged)
}

func TestWorkspacestats_BuildWorkspaceStatsRecord_Ugly_MissingReport(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	workspaceDir := core.JoinPath(root, "workspace", "core", "go-io", "task-5")
	fs.EnsureDir(workspaceDir)

	// No .meta/report.json — build record from status only.
	record := buildWorkspaceStatsRecord(workspaceDir, &WorkspaceStatus{
		Repo:   "go-io",
		Branch: "agent/task-5",
		Agent:  "codex:gpt-5.4",
		Status: "failed",
	})

	assert.Equal(t, "core/go-io/task-5", record.Workspace)
	assert.False(t, record.Passed)
	assert.Equal(t, 0, record.FindingsTotal)
	assert.Nil(t, record.BySeverity)
}

func TestWorkspacestats_RecordWorkspaceStats_Good_WritesToStore(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	workspaceDir := core.JoinPath(root, "workspace", "core", "go-io", "task-5")
	fs.EnsureDir(workspaceDir)

	c := core.New()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	t.Cleanup(s.closeWorkspaceStatsStore)

	status := &WorkspaceStatus{
		Repo:   "go-io",
		Org:    "core",
		Branch: "agent/task-5",
		Agent:  "codex:gpt-5.4",
		Status: "completed",
	}

	s.recordWorkspaceStats(workspaceDir, status)

	statsStore := s.workspaceStatsInstance()
	if statsStore == nil {
		t.Skip("go-store unavailable on this platform — RFC §15.6 graceful degradation")
	}

	value, err := statsStore.Get(stateWorkspaceStatsGroup, "core/go-io/task-5")
	assert.NoError(t, err)
	assert.Contains(t, value, "core/go-io/task-5")
	assert.Contains(t, value, "go-io")
}

func TestWorkspacestats_RecordWorkspaceStats_Bad_NilInputs(t *testing.T) {
	var s *PrepSubsystem
	// Nil receiver is a no-op — no panic.
	s.recordWorkspaceStats("/tmp/workspace", &WorkspaceStatus{})

	c := core.New()
	s = &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	// Empty workspace directory — no-op.
	s.recordWorkspaceStats("", &WorkspaceStatus{Repo: "go-io"})
	// Nil status — no-op.
	s.recordWorkspaceStats("/tmp/workspace", nil)
}

func TestWorkspacestats_WorkspaceStatsPath_Good(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	expected := core.JoinPath(root, "workspace", "db.duckdb")
	assert.Equal(t, expected, workspaceStatsPath())
}

func TestWorkspacestats_WorkspaceStatsMatches_Good(t *testing.T) {
	record := workspaceStatsRecord{Repo: "go-io", Status: "completed"}
	assert.True(t, workspaceStatsMatches(record, "", ""))
	assert.True(t, workspaceStatsMatches(record, "go-io", ""))
	assert.True(t, workspaceStatsMatches(record, "", "completed"))
	assert.True(t, workspaceStatsMatches(record, "go-io", "completed"))
}

func TestWorkspacestats_WorkspaceStatsMatches_Bad_RepoMismatch(t *testing.T) {
	record := workspaceStatsRecord{Repo: "go-io", Status: "completed"}
	assert.False(t, workspaceStatsMatches(record, "go-log", ""))
	assert.False(t, workspaceStatsMatches(record, "", "failed"))
}

func TestWorkspacestats_FilterWorkspaceStats_Good_AppliesLimit(t *testing.T) {
	records := []workspaceStatsRecord{
		{Workspace: "a", Repo: "go-io", Status: "completed"},
		{Workspace: "b", Repo: "go-io", Status: "completed"},
		{Workspace: "c", Repo: "go-io", Status: "completed"},
	}

	filtered := filterWorkspaceStats(records, "go-io", "completed", 2)
	assert.Len(t, filtered, 2)
	assert.Equal(t, "a", filtered[0].Workspace)
	assert.Equal(t, "b", filtered[1].Workspace)
}

func TestWorkspacestats_FilterWorkspaceStats_Ugly_FilterSkipsMismatches(t *testing.T) {
	records := []workspaceStatsRecord{
		{Workspace: "a", Repo: "go-io", Status: "completed"},
		{Workspace: "b", Repo: "go-io", Status: "failed"},
		{Workspace: "c", Repo: "go-log", Status: "completed"},
	}

	// Repo filter drops the go-log row, status filter drops the failed one.
	filtered := filterWorkspaceStats(records, "go-io", "completed", 0)
	assert.Len(t, filtered, 1)
	assert.Equal(t, "a", filtered[0].Workspace)

	// Empty filters return everything.
	assert.Len(t, filterWorkspaceStats(records, "", "", 0), 3)

	// Nil input returns nil.
	assert.Nil(t, filterWorkspaceStats(nil, "", "", 0))
}

func TestWorkspacestats_ListWorkspaceStats_Ugly_StoreUnavailableReturnsNil(t *testing.T) {
	var s *PrepSubsystem
	assert.Nil(t, s.listWorkspaceStats())
}

func TestWorkspacestats_WorkspaceStatsInstance_Ugly_ReopenAfterClose(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	c := core.New()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	t.Cleanup(s.closeWorkspaceStatsStore)

	first := s.workspaceStatsInstance()
	if first == nil {
		t.Skip("go-store unavailable on this platform — RFC §15.6 graceful degradation")
	}

	s.closeWorkspaceStatsStore()

	second := s.workspaceStatsInstance()
	assert.NotNil(t, second)
	// After close the reference is reset so a new instance is opened — the
	// old pointer is stale but the store handle is re-used transparently.
}

func TestWorkspacestats_HandleWorkspaceStats_Good_ReturnsEmptyWhenNoRows(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	c := core.New()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	t.Cleanup(s.closeWorkspaceStatsStore)

	result := s.handleWorkspaceStats(nil, core.NewOptions())
	assert.True(t, result.OK)
	out, ok := result.Value.(WorkspaceStatsOutput)
	assert.True(t, ok)
	assert.Equal(t, 0, out.Count)
}

func TestWorkspacestats_HandleWorkspaceStats_Good_AppliesFilters(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	c := core.New()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	t.Cleanup(s.closeWorkspaceStatsStore)

	// Seed two stats rows by recording two workspaces.
	for _, ws := range []struct{ name, repo, status string }{
		{"core/go-io/task-1", "go-io", "completed"},
		{"core/go-io/task-2", "go-io", "failed"},
		{"core/go-log/task-3", "go-log", "completed"},
	} {
		workspaceDir := core.JoinPath(root, "workspace", ws.name)
		fs.EnsureDir(workspaceDir)
		s.recordWorkspaceStats(workspaceDir, &WorkspaceStatus{
			Repo:   ws.repo,
			Status: ws.status,
			Agent:  "codex:gpt-5.4",
		})
	}

	if s.workspaceStatsInstance() == nil {
		t.Skip("go-store unavailable on this platform — RFC §15.6 graceful degradation")
	}

	// Filter by repo only.
	result := s.handleWorkspaceStats(nil, core.NewOptions(
		core.Option{Key: "repo", Value: "go-io"},
	))
	assert.True(t, result.OK)
	out := result.Value.(WorkspaceStatsOutput)
	assert.Equal(t, 2, out.Count)

	// Filter by repo + status.
	result = s.handleWorkspaceStats(nil, core.NewOptions(
		core.Option{Key: "repo", Value: "go-io"},
		core.Option{Key: "status", Value: "completed"},
	))
	out = result.Value.(WorkspaceStatsOutput)
	assert.Equal(t, 1, out.Count)

	// Limit trims the result set.
	result = s.handleWorkspaceStats(nil, core.NewOptions(
		core.Option{Key: "limit", Value: 1},
	))
	out = result.Value.(WorkspaceStatsOutput)
	assert.Equal(t, 1, out.Count)
}

func TestWorkspacestats_CmdWorkspaceStats_Good_NoRowsPrintsFriendlyMessage(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	c := core.New()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	t.Cleanup(s.closeWorkspaceStatsStore)

	result := s.cmdWorkspaceStats(core.NewOptions())
	assert.True(t, result.OK)
}

func TestWorkspacestats_CmdWorkspaceStats_Good_PrintsTable(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	c := core.New()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	t.Cleanup(s.closeWorkspaceStatsStore)

	workspaceDir := core.JoinPath(root, "workspace", "core", "go-io", "task-1")
	fs.EnsureDir(workspaceDir)
	s.recordWorkspaceStats(workspaceDir, &WorkspaceStatus{
		Repo:   "go-io",
		Status: "completed",
		Agent:  "codex:gpt-5.4",
	})

	if s.workspaceStatsInstance() == nil {
		t.Skip("go-store unavailable on this platform — RFC §15.6 graceful degradation")
	}

	result := s.cmdWorkspaceStats(core.NewOptions())
	assert.True(t, result.OK)
}

func TestWorkspacestats_RegisterWorkspaceStatsCommand_Good(t *testing.T) {
	s, c := testPrepWithCore(t, nil)

	s.registerWorkspaceCommands()

	assert.Contains(t, c.Commands(), "workspace/stats")
	assert.Contains(t, c.Commands(), "agentic:workspace/stats")
}
