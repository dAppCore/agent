// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"time"

	core "dappco.re/go"
	store "dappco.re/go/store"
)

// stateWorkspaceStatsGroup is the group key inside the parent workspace store
// used to persist per-dispatch stat rows per RFC §15.5. The top-level state
// store already has `dispatch_history`, which is volatile (drains when pushed
// to the platform). The parent stats store is the permanent record so the
// "what happened in the last 50 dispatches" query described in RFC §15.5 stays
// answerable even after sync has drained the dispatch history.
//
// Usage example: `s.workspaceStatsInstance().Set(stateWorkspaceStatsGroup, workspaceName, payload)`
const stateWorkspaceStatsGroup = "stats"

// workspaceStatsRef carries the lazily-initialised go-store handle for the
// parent `.core/workspace/db.duckdb` stats database. The reference is kept
// separate from the top-level `stateStoreRef` so the two stores open
// independently — a missing parent DB does not disable top-level state.
type workspaceStatsRef struct {
	once     core.Once
	instance *store.Store
	err      error
}

// workspaceStatsPath returns the canonical path for the parent workspace
// stats database described in RFC §15.5 — `.core/workspace/db.duckdb`.
//
// Usage example: `path := workspaceStatsPath() // "/.core/workspace/db.duckdb"`
func workspaceStatsPath() string {
	return core.JoinPath(WorkspaceRoot(), "db.duckdb")
}

// workspaceStatsInstance lazily opens the parent workspace stats store.
// Returns nil when go-store is unavailable so callers can fall back to the
// file-system journal under RFC §15.6 graceful degradation.
//
// Usage example: `if stats := s.workspaceStatsInstance(); stats != nil { stats.Set("stats", name, payload) }`
func (s *PrepSubsystem) workspaceStatsInstance() *store.Store {
	if s == nil {
		return nil
	}
	ref := s.workspaceStatsReference()
	if ref == nil {
		return nil
	}
	ref.once.Do(func() {
		ref.instance, ref.err = openWorkspaceStatsStore()
	})
	if ref.err != nil {
		return nil
	}
	return ref.instance
}

// workspaceStatsReference allocates the lazy reference — tests that use a
// zero-value subsystem can still call stats helpers without panicking.
func (s *PrepSubsystem) workspaceStatsReference() *workspaceStatsRef {
	if s == nil {
		return nil
	}
	s.workspaceStatsOnce.Do(func() {
		s.workspaceStats = &workspaceStatsRef{}
	})
	return s.workspaceStats
}

// closeWorkspaceStatsStore releases the parent stats handle so the file
// descriptor is not left open during shutdown.
//
// Usage example: `s.closeWorkspaceStatsStore()`
func (s *PrepSubsystem) closeWorkspaceStatsStore() {
	if s == nil {
		return
	}
	ref := s.workspaceStats
	if ref == nil {
		return
	}
	if ref.instance != nil {
		if result := ref.instance.Close(); !result.OK {
			core.Warn("agentic.workspaceStats: failed to close workspace stats store", `path`, workspaceStatsPath(), "reason", resultErrorValue("agentic.workspaceStats", result))
		}
		ref.instance = nil
	}
	ref.err = nil
	s.workspaceStats = nil
	s.workspaceStatsOnce.Reset()
}

// openWorkspaceStatsStore opens the parent workspace stats database,
// creating the containing directory first so the first call on a clean
// machine succeeds. Errors are returned instead of panicking so the agent
// still boots without the parent stats DB per RFC §15.6.
//
// Usage example: `st, err := openWorkspaceStatsStore()`
var openWorkspaceStatsStore = func() (*store.Store, error) {
	path := workspaceStatsPath()
	directory := core.PathDir(path)
	if ensureResult := fs.EnsureDir(directory); !ensureResult.OK {
		if err, ok := ensureResult.Value.(error); ok {
			return nil, core.E("agentic.workspaceStats", "prepare workspace stats directory", err)
		}
		return nil, core.E("agentic.workspaceStats", "prepare workspace stats directory", nil)
	}
	result := store.New(path)
	if !result.OK {
		return nil, core.E("agentic.workspaceStats", "open workspace stats store", resultErrorValue("agentic.workspaceStats", result))
	}
	return result.Value.(*store.Store), nil
}

// workspaceStatsRecord is the shape persisted for each dispatch cycle. The
// fields mirror RFC §15.5 — dispatch duration, agent, model, repo, branch,
// findings counts by severity/tool/category, build/test pass-fail, changes,
// and the dispatch report summary (clusters, new, resolved, persistent).
//
// Usage example:
//
//	record := workspaceStatsRecord{
//	    Workspace:   "core/go-io/task-5",
//	    Repo:        "go-io",
//	    Branch:      "agent/task-5",
//	    Agent:       "codex:gpt-5.4-mini",
//	    Status:      "completed",
//	    DurationMS:  12843,
//	    BuildPassed: true,
//	    TestPassed:  true,
//	}
type workspaceStatsRecord struct {
	Workspace       string         `json:"workspace"`
	Repo            string         `json:"repo,omitempty"`
	Org             string         `json:"org,omitempty"`
	Branch          string         `json:"branch,omitempty"`
	Agent           string         `json:"agent,omitempty"`
	Model           string         `json:"model,omitempty"`
	Task            string         `json:"task,omitempty"`
	Status          string         `json:"status,omitempty"`
	Runs            int            `json:"runs,omitempty"`
	StartedAt       string         `json:"started_at,omitempty"`
	UpdatedAt       string         `json:"updated_at,omitempty"`
	CompletedAt     string         `json:"completed_at,omitempty"`
	DurationMS      int64          `json:"duration_ms,omitempty"`
	BuildPassed     bool           `json:"build_passed"`
	TestPassed      bool           `json:"test_passed"`
	LintPassed      bool           `json:"lint_passed"`
	Passed          bool           `json:"passed"`
	FindingsTotal   int            `json:"findings_total,omitempty"`
	BySeverity      map[string]int `json:"findings_by_severity,omitempty"`
	ByTool          map[string]int `json:"findings_by_tool,omitempty"`
	ByCategory      map[string]int `json:"findings_by_category,omitempty"`
	Insertions      int            `json:"insertions,omitempty"`
	Deletions       int            `json:"deletions,omitempty"`
	FilesChanged    int            `json:"files_changed,omitempty"`
	ClustersCount   int            `json:"clusters_count,omitempty"`
	NewCount        int            `json:"new_count,omitempty"`
	ResolvedCount   int            `json:"resolved_count,omitempty"`
	PersistentCount int            `json:"persistent_count,omitempty"`
}

// recordWorkspaceStats writes a stats row for a dispatch cycle into the
// parent workspace store (RFC §15.5). The caller typically invokes this
// immediately before deleting the workspace directory so the permanent
// record survives cleanup. No-op when go-store is unavailable.
//
// Usage example: `s.recordWorkspaceStats(workspaceDir, workspaceStatus)`
func (s *PrepSubsystem) recordWorkspaceStats(workspaceDir string, workspaceStatus *WorkspaceStatus) {
	if s == nil || workspaceDir == "" || workspaceStatus == nil {
		return
	}
	statsStore := s.workspaceStatsInstance()
	if statsStore == nil {
		return
	}
	record := buildWorkspaceStatsRecord(workspaceDir, workspaceStatus)
	payload := core.JSONMarshalString(record)
	if payload == "" {
		return
	}
	if result := statsStore.Set(stateWorkspaceStatsGroup, record.Workspace, payload); !result.OK {
		core.Warn("agentic.workspaceStats: failed to persist workspace stats", "workspace", record.Workspace, "reason", resultErrorValue("agentic.workspaceStats", result))
	}
}

// buildWorkspaceStatsRecord projects the WorkspaceStatus and the dispatch
// report sidecar (`.meta/report.json`) into the stats row shape documented in
// RFC §15.5. The report is optional — older cycles that predate the QA
// capture pipeline still write a row using just the status fields.
//
// Usage example: `record := buildWorkspaceStatsRecord(workspaceDir, workspaceStatus)`
func buildWorkspaceStatsRecord(workspaceDir string, workspaceStatus *WorkspaceStatus) workspaceStatsRecord {
	record := workspaceStatsRecord{
		Workspace:   WorkspaceName(workspaceDir),
		Repo:        workspaceStatus.Repo,
		Org:         workspaceStatus.Org,
		Branch:      workspaceStatus.Branch,
		Agent:       workspaceStatus.Agent,
		Model:       extractModelFromAgent(workspaceStatus.Agent),
		Task:        workspaceStatus.Task,
		Status:      workspaceStatus.Status,
		Runs:        workspaceStatus.Runs,
		StartedAt:   formatTimeRFC3339(workspaceStatus.StartedAt),
		UpdatedAt:   formatTimeRFC3339(workspaceStatus.UpdatedAt),
		CompletedAt: time.Now().UTC().Format(time.RFC3339),
		DurationMS:  dispatchDurationMS(workspaceStatus.StartedAt, workspaceStatus.UpdatedAt),
	}

	if report := readSyncWorkspaceReport(workspaceDir); len(report) > 0 {
		if passed, ok := report["passed"].(bool); ok {
			record.Passed = passed
		}
		if buildPassed, ok := report["build_passed"].(bool); ok {
			record.BuildPassed = buildPassed
		}
		if testPassed, ok := report["test_passed"].(bool); ok {
			record.TestPassed = testPassed
		}
		if lintPassed, ok := report["lint_passed"].(bool); ok {
			record.LintPassed = lintPassed
		}
		findings := anyMapSliceValue(report["findings"])
		record.FindingsTotal = len(findings)
		record.BySeverity = countFindingsBy(findings, "severity")
		record.ByTool = countFindingsBy(findings, "tool")
		record.ByCategory = countFindingsBy(findings, "category")
		if clusters := anyMapSliceValue(report["clusters"]); len(clusters) > 0 {
			record.ClustersCount = len(clusters)
		}
		if newList := anyMapSliceValue(report["new"]); len(newList) > 0 {
			record.NewCount = len(newList)
		}
		if resolvedList := anyMapSliceValue(report["resolved"]); len(resolvedList) > 0 {
			record.ResolvedCount = len(resolvedList)
		}
		if persistentList := anyMapSliceValue(report["persistent"]); len(persistentList) > 0 {
			record.PersistentCount = len(persistentList)
		}
		if changes := anyMapValue(report["changes"]); len(changes) > 0 {
			record.Insertions = intValue(changes["insertions"])
			record.Deletions = intValue(changes["deletions"])
			record.FilesChanged = intValue(changes["files_changed"])
		}
	}

	return record
}

// extractModelFromAgent splits an agent identifier like `codex:gpt-5.4-mini`
// into the model suffix so the stats row records the concrete model without
// parsing elsewhere. Agent strings without a colon leave Model empty so the
// upstream Agent field carries the full value.
//
// Usage example: `model := extractModelFromAgent("codex:gpt-5.4-mini") // "gpt-5.4-mini"`
func extractModelFromAgent(agent string) string {
	if agent == "" {
		return ""
	}
	parts := core.SplitN(agent, ":", 2)
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}

// formatTimeRFC3339 renders a time.Time as RFC3339 UTC, returning an empty
// string when the time is zero so the stats row does not record a bogus
// "0001-01-01" timestamp for dispatches that never started.
//
// Usage example: `ts := formatTimeRFC3339(time.Now())`
func formatTimeRFC3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// dispatchDurationMS returns the elapsed milliseconds between StartedAt and
// UpdatedAt when both are populated. Zero is returned when either side is
// missing so the stats row skips the field instead of reporting a negative
// value.
//
// Usage example: `ms := dispatchDurationMS(status.StartedAt, status.UpdatedAt)`
func dispatchDurationMS(startedAt, updatedAt time.Time) int64 {
	if startedAt.IsZero() || updatedAt.IsZero() {
		return 0
	}
	if !updatedAt.After(startedAt) {
		return 0
	}
	return updatedAt.Sub(startedAt).Milliseconds()
}

// countFindingsBy groups a slice of finding maps by the value at `field` and
// returns a count per distinct value. Missing or empty values are skipped so
// the resulting map only contains keys that appeared in the data.
//
// Usage example: `bySev := countFindingsBy(findings, "severity") // {"error": 3, "warning": 7}`
func countFindingsBy(findings []map[string]any, field string) map[string]int {
	if len(findings) == 0 || field == "" {
		return nil
	}
	counts := map[string]int{}
	for _, entry := range findings {
		value := stringValue(entry[field])
		if value == "" {
			continue
		}
		counts[value]++
	}
	if len(counts) == 0 {
		return nil
	}
	return counts
}

// listWorkspaceStats returns every stats row currently persisted in the
// parent workspace store — the list is unsorted so callers decide how to
// present the data (recent first, grouped by repo, etc.). Returns nil when
// go-store is unavailable so RFC §15.6 graceful degradation holds.
//
// Usage example: `rows := s.listWorkspaceStats() // [{Workspace: "core/go-io/task-5", ...}, ...]`
func (s *PrepSubsystem) listWorkspaceStats() []workspaceStatsRecord {
	if s == nil {
		return nil
	}
	statsStore := s.workspaceStatsInstance()
	if statsStore == nil {
		return nil
	}

	var rows []workspaceStatsRecord
	for entry, err := range statsStore.AllSeq(stateWorkspaceStatsGroup) {
		if err != nil {
			return rows
		}
		var record workspaceStatsRecord
		if parseResult := core.JSONUnmarshalString(entry.Value, &record); !parseResult.OK {
			continue
		}
		rows = append(rows, record)
	}
	return rows
}

// workspaceStatsMatches reports whether a stats record passes the given
// filters. Empty filters act as wildcards, so `matches("", "")` returns true
// for every row. Keeping the filter semantics local to this helper means the
// CLI, MCP tool and action handler stay a single line each.
//
// Usage example: `if workspaceStatsMatches(row, "go-io", "completed") { ... }`
func workspaceStatsMatches(record workspaceStatsRecord, repo, status string) bool {
	if repo != "" && record.Repo != repo {
		return false
	}
	if status != "" && record.Status != status {
		return false
	}
	return true
}

// filterWorkspaceStats returns the subset of records that match the given
// repo and status filters. Limit <= 0 returns every match. Callers wire the
// order before slicing so `limit=50` always returns the 50 most relevant
// rows.
//
// Usage example: `rows := filterWorkspaceStats(all, "go-io", "completed", 50)`
func filterWorkspaceStats(records []workspaceStatsRecord, repo, status string, limit int) []workspaceStatsRecord {
	if len(records) == 0 {
		return nil
	}
	out := make([]workspaceStatsRecord, 0, len(records))
	for _, record := range records {
		if !workspaceStatsMatches(record, repo, status) {
			continue
		}
		out = append(out, record)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}
