// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"
	"time"

	core "dappco.re/go"
)

func TestWorkspacestats_ExtractModelFromAgent_Good_Case(t *testing.T) {
	codexModel := extractModelFromAgent("codex:gpt-5.4-mini")
	claudeModel := extractModelFromAgent("claude:sonnet")
	core.AssertEqual(t, "gpt-5.4-mini", codexModel)
	core.AssertEqual(t, "sonnet", claudeModel)
}

func TestWorkspacestats_ExtractModelFromAgent_Bad_NoColon(t *testing.T) {
	model := extractModelFromAgent("codex")
	core.AssertEqual(t, "", model)
	core.AssertEmpty(t, model)
}

func TestWorkspacestats_ExtractModelFromAgent_Ugly_EmptyAndMultipleColons(t *testing.T) {
	empty := extractModelFromAgent("")
	model := extractModelFromAgent("codex:gpt:5.4:mini")
	core.AssertEqual(t, "", empty)
	// Multiple colons — the model preserves the remainder unchanged.
	core.AssertEqual(t, "gpt:5.4:mini", model)
}

func TestWorkspacestats_DispatchDurationMS_Good_Case(t *testing.T) {
	started := time.Now()
	updated := started.Add(2500 * time.Millisecond)
	core.AssertEqual(t, int64(2500), dispatchDurationMS(started, updated))
}

func TestWorkspacestats_DispatchDurationMS_Bad_ZeroStart(t *testing.T) {
	startedAt := time.Time{}
	duration := dispatchDurationMS(startedAt, time.Now())
	core.AssertEqual(t, int64(0), duration)
	core.AssertTrue(t, duration >= 0)
}

func TestWorkspacestats_DispatchDurationMS_Ugly_UpdatedBeforeStarted(t *testing.T) {
	started := time.Now()
	updated := started.Add(-5 * time.Second)
	// When UpdatedAt is before StartedAt we return 0 rather than a negative value.
	core.AssertEqual(t, int64(0), dispatchDurationMS(started, updated))
}

func TestWorkspacestats_CountFindingsBy_Good_Case(t *testing.T) {
	findings := []map[string]any{
		{"severity": "error", "tool": "gosec"},
		{"severity": "error", "tool": "gosec"},
		{"severity": "warning", "tool": "golangci-lint"},
	}
	counts := countFindingsBy(findings, "severity")
	core.AssertEqual(t, 2, counts["error"])
	core.AssertEqual(t, 1, counts["warning"])
}

func TestWorkspacestats_CountFindingsBy_Bad_EmptySlice(t *testing.T) {
	nilCounts := countFindingsBy(nil, "severity")
	emptyCounts := countFindingsBy([]map[string]any{}, "severity")
	core.AssertNil(t, nilCounts)
	core.AssertNil(t, emptyCounts)
}

func TestWorkspacestats_CountFindingsBy_Ugly_MissingFieldValues(t *testing.T) {
	findings := []map[string]any{
		{"severity": "error"},
		{"severity": ""},
		{"severity": nil},
		{"tool": "gosec"}, // no severity at all
	}
	counts := countFindingsBy(findings, "severity")
	core.AssertEqual(t, 1, counts["error"])
	// Empty and missing values are skipped, so the map only holds "error".
	core.AssertEqual(t, 1, len(counts))
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

	core.AssertEqual(t, "core/go-io/task-5", record.Workspace)
	core.AssertEqual(t, "go-io", record.Repo)
	core.AssertEqual(t, "agent/task-5", record.Branch)
	core.AssertEqual(t, "codex:gpt-5.4-mini", record.Agent)
	core.AssertEqual(t, "gpt-5.4-mini", record.Model)
	core.AssertEqual(t, "completed", record.Status)
	core.AssertEqual(t, 2, record.Runs)
	core.AssertEqual(t, int64(3500), record.DurationMS)
	core.AssertNotEmpty(t, record.CompletedAt)
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

	core.AssertTrue(t, record.Passed)
	core.AssertTrue(t, record.BuildPassed)
	core.AssertTrue(t, record.TestPassed)
	core.AssertTrue(t, record.LintPassed)
	core.AssertEqual(t, 2, record.FindingsTotal)
	core.AssertEqual(t, 1, record.BySeverity["error"])
	core.AssertEqual(t, 1, record.BySeverity["warning"])
	core.AssertEqual(t, 1, record.ByTool["gosec"])
	core.AssertEqual(t, 1, record.ByTool["golangci-lint"])
	core.AssertEqual(t, 1, record.ClustersCount)
	core.AssertEqual(t, 1, record.NewCount)
	core.AssertEqual(t, 1, record.ResolvedCount)
	core.AssertEqual(t, 0, record.PersistentCount)
	core.AssertEqual(t, 12, record.Insertions)
	core.AssertEqual(t, 3, record.Deletions)
	core.AssertEqual(t, 2, record.FilesChanged)
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

	core.AssertEqual(t, "core/go-io/task-5", record.Workspace)
	core.AssertFalse(t, record.Passed)
	core.AssertEqual(t, 0, record.FindingsTotal)
	core.AssertNil(t, record.BySeverity)
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

	value, result := statsStore.Get(stateWorkspaceStatsGroup, "core/go-io/task-5")
	if !result.OK {
		t.Fatalf("read workspace stats: %v", resultErrorValue("TestWorkspacestats_RecordWorkspaceStats_Good_WritesToStore", result))
	}
	core.AssertContains(t, value, "core/go-io/task-5")
	core.AssertContains(t, value, "go-io")
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

func TestWorkspacestats_WorkspaceStatsPath_Good_Case(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	expected := core.JoinPath(root, "workspace", "db.duckdb")
	core.AssertEqual(t, expected, workspaceStatsPath())
}

func TestWorkspacestats_WorkspaceStatsMatches_Good_Case(t *testing.T) {
	record := workspaceStatsRecord{Repo: "go-io", Status: "completed"}
	core.AssertTrue(t, workspaceStatsMatches(record, "", ""))
	core.AssertTrue(t, workspaceStatsMatches(record, "go-io", ""))
	core.AssertTrue(t, workspaceStatsMatches(record, "", "completed"))
	core.AssertTrue(t, workspaceStatsMatches(record, "go-io", "completed"))
}

func TestWorkspacestats_WorkspaceStatsMatches_Bad_RepoMismatch(t *testing.T) {
	record := workspaceStatsRecord{Repo: "go-io", Status: "completed"}
	core.AssertFalse(t, workspaceStatsMatches(record, "go-log", ""))
	core.AssertFalse(t, workspaceStatsMatches(record, "", "failed"))
}

func TestWorkspacestats_FilterWorkspaceStats_Good_AppliesLimit(t *testing.T) {
	records := []workspaceStatsRecord{
		{Workspace: "a", Repo: "go-io", Status: "completed"},
		{Workspace: "b", Repo: "go-io", Status: "completed"},
		{Workspace: "c", Repo: "go-io", Status: "completed"},
	}

	filtered := filterWorkspaceStats(records, "go-io", "completed", 2)
	core.AssertLen(t, filtered, 2)
	core.AssertEqual(t, "a", filtered[0].Workspace)
	core.AssertEqual(t, "b", filtered[1].Workspace)
}

func TestWorkspacestats_FilterWorkspaceStats_Ugly_FilterSkipsMismatches(t *testing.T) {
	records := []workspaceStatsRecord{
		{Workspace: "a", Repo: "go-io", Status: "completed"},
		{Workspace: "b", Repo: "go-io", Status: "failed"},
		{Workspace: "c", Repo: "go-log", Status: "completed"},
	}

	// Repo filter drops the go-log row, status filter drops the failed one.
	filtered := filterWorkspaceStats(records, "go-io", "completed", 0)
	core.AssertLen(t, filtered, 1)
	core.AssertEqual(t, "a", filtered[0].Workspace)

	// Empty filters return everything.
	core.AssertLen(t, filterWorkspaceStats(records, "", "", 0), 3)

	// Nil input returns nil.
	core.AssertNil(t, filterWorkspaceStats(nil, "", "", 0))
}

func TestWorkspacestats_ListWorkspaceStats_Ugly_StoreUnavailableReturnsNil(t *testing.T) {
	var s *PrepSubsystem
	rows := s.listWorkspaceStats()
	core.AssertNil(t, rows)
	core.AssertEqual(t, 0, len(rows))
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
	core.AssertNotNil(t, second)
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
	core.AssertTrue(t, result.OK)
	out, ok := result.Value.(WorkspaceStatsOutput)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, 0, out.Count)
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
	core.AssertTrue(t, result.OK)
	out := result.Value.(WorkspaceStatsOutput)
	core.AssertEqual(t, 2, out.Count)

	// Filter by repo + status.
	result = s.handleWorkspaceStats(nil, core.NewOptions(
		core.Option{Key: "repo", Value: "go-io"},
		core.Option{Key: "status", Value: "completed"},
	))
	out = result.Value.(WorkspaceStatsOutput)
	core.AssertEqual(t, 1, out.Count)

	// Limit trims the result set.
	result = s.handleWorkspaceStats(nil, core.NewOptions(
		core.Option{Key: "limit", Value: 1},
	))
	out = result.Value.(WorkspaceStatsOutput)
	core.AssertEqual(t, 1, out.Count)
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
	core.AssertTrue(t, result.OK)
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
	core.AssertTrue(t, result.OK)
}

func TestWorkspacestats_RegisterWorkspaceStatsCommand_Good_Case(t *testing.T) {
	s, c := testPrepWithCore(t, nil)

	s.registerWorkspaceCommands()

	core.AssertContains(t, c.Commands(), "workspace/stats")
	core.AssertContains(t, c.Commands(), "agentic:workspace/stats")
}
