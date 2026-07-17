// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go"
	store "dappco.re/go/store"
)

// uniqueWorkspaceName returns a collision-free workspace name. NewWorkspace
// writes a real `<name>.duckdb` under the CWD-relative `.core/state/` dir and
// refuses to recreate an existing file, so a fixed name leaks across repeated
// runs (`-count=2`). A nanosecond suffix keeps each invocation distinct.
func uniqueWorkspaceName(prefix string) string {
	return core.Concat(prefix, "-", core.Itoa(int(time.Now().UnixNano())))
}

// --- runQALegacy (direct, bypassing the go-store report path) ---

func TestQa_RunQALegacy_Good_GoRepoPasses(t *testing.T) {
	wsDir := t.TempDir()
	repoDir := core.JoinPath(wsDir, "repo")
	core.RequireTrue(t, fs.EnsureDir(repoDir).OK)
	fs.Write(core.JoinPath(repoDir, "go.mod"), "module testmod\n\ngo 1.22\n")
	fs.Write(core.JoinPath(repoDir, "main.go"), "package main\nfunc main() {}\n")

	s := newPrepWithProcess()
	core.AssertTrue(t, s.runQALegacy(context.Background(), wsDir))
}

func TestQa_RunQALegacy_Bad_GoBuildFails(t *testing.T) {
	wsDir := t.TempDir()
	repoDir := core.JoinPath(wsDir, "repo")
	core.RequireTrue(t, fs.EnsureDir(repoDir).OK)
	fs.Write(core.JoinPath(repoDir, "go.mod"), "module testmod\n\ngo 1.22\n")
	// Syntactically broken Go — build fails on the first cascade step.
	fs.Write(core.JoinPath(repoDir, "main.go"), "package main\nfunc main( {\n}\n")

	s := newPrepWithProcess()
	core.AssertFalse(t, s.runQALegacy(context.Background(), wsDir))
}

func TestQa_RunQALegacy_Ugly_NoBuildSystem(t *testing.T) {
	// No go.mod / composer.json / package.json → passes (nothing to check).
	wsDir := t.TempDir()
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(wsDir, "repo")).OK)

	s := newPrepWithProcess()
	core.AssertTrue(t, s.runQALegacy(context.Background(), wsDir))

	// Composer project with composer unavailable — install fails → false. This
	// is deterministic because the fixture has no vendor dir and composer is not
	// on the test PATH; mirror runQA's existing Bad composer assertion.
	wsDir2 := t.TempDir()
	repoDir2 := core.JoinPath(wsDir2, "repo")
	core.RequireTrue(t, fs.EnsureDir(repoDir2).OK)
	fs.Write(core.JoinPath(repoDir2, "composer.json"), `{"name":"test"}`)
	core.AssertFalse(t, s.runQALegacy(context.Background(), wsDir2))
}

// --- runLintReport (fake core-lint emits a parseable report) ---

func TestQa_RunLintReport_Good_ParsesJSONReport(t *testing.T) {
	binDir := t.TempDir()
	scriptPath := core.JoinPath(binDir, "core-lint")
	// Emit a minimal but valid lint report JSON on stdout.
	body := "#!/bin/sh\ncat <<'LINT_EOF'\n" +
		`{"project":"go-io","tools":[{"name":"gosec","status":"ok","findings":1}],` +
		`"findings":[{"tool":"gosec","file":"a.go","line":10,"severity":"error","code":"G101","message":"secret"}],` +
		`"summary":{"total":1,"errors":1}}` +
		"\nLINT_EOF\n"
	core.RequireTrue(t, core.WriteFile(scriptPath, []byte(body), 0o755).OK)
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

	repoDir := t.TempDir()
	s := newPrepWithProcess()
	report := s.runLintReport(context.Background(), repoDir)

	core.AssertLen(t, report.Findings, 1)
	core.AssertEqual(t, "gosec", report.Findings[0].Tool)
	core.AssertEqual(t, "G101", report.Findings[0].Code)
	core.AssertLen(t, report.Tools, 1)
	core.AssertEqual(t, 1, report.Summary.Errors)
}

func TestQa_RunLintReport_Ugly_NonJSONOutputDegrades(t *testing.T) {
	binDir := t.TempDir()
	scriptPath := core.JoinPath(binDir, "core-lint")
	// Non-JSON stdout → JSON unmarshal fails → empty report (graceful degrade).
	core.RequireTrue(t, core.WriteFile(scriptPath, []byte("#!/bin/sh\necho 'not json at all'\n"), 0o755).OK)
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

	s := newPrepWithProcess()
	report := s.runLintReport(context.Background(), t.TempDir())
	core.AssertEmpty(t, report.Findings)
	core.AssertEmpty(t, report.Tools)
}

// --- recordLintFindings (real :memory: workspace) ---

func TestQa_RecordLintFindings_Good_PersistsFindingsAndTools(t *testing.T) {
	result := store.New(":memory:")
	core.RequireTrue(t, result.OK)
	storeInstance := result.Value.(*store.Store)
	t.Cleanup(func() { _ = storeInstance.Close() })

	workspace, wsResult := storeInstance.NewWorkspace(uniqueWorkspaceName("qa-record-good"))
	core.RequireTrue(t, wsResult.OK)
	t.Cleanup(workspace.Discard)

	report := QAReport{
		Findings: []QAFinding{
			{Tool: "gosec", File: "a.go", Line: 10, Severity: "error", Code: "G101", Message: "secret"},
			{Tool: "staticcheck", File: "b.go", Line: 5, Severity: "warning", Code: "SA1000"},
		},
		Tools: []QAToolRun{
			{Name: "gosec", Status: "ok", Findings: 1},
			{Name: "staticcheck", Status: "ok", Findings: 1},
		},
	}

	s := newPrepWithProcess()
	s.recordLintFindings(workspace, report)

	// All four rows (2 findings + 2 tool runs) land in the buffer before commit.
	count, countResult := workspace.Count()
	core.RequireTrue(t, countResult.OK)
	core.AssertEqual(t, 4, count)

	// The per-kind aggregate records both finding and tool_run kinds.
	aggregate := workspace.Aggregate()
	core.AssertEqual(t, 2, intValue(aggregate["finding"]))
	core.AssertEqual(t, 2, intValue(aggregate["tool_run"]))
}

func TestQa_RecordLintFindings_Bad_NilWorkspace(t *testing.T) {
	// nil workspace is a no-op (graceful degradation path).
	s := newPrepWithProcess()
	core.AssertNotPanics(t, func() {
		s.recordLintFindings(nil, QAReport{Findings: []QAFinding{{Tool: "gosec"}}})
	})
}

func TestQa_RecordLintFindings_Ugly_EmptyReport(t *testing.T) {
	result := store.New(":memory:")
	core.RequireTrue(t, result.OK)
	storeInstance := result.Value.(*store.Store)
	t.Cleanup(func() { _ = storeInstance.Close() })

	workspace, wsResult := storeInstance.NewWorkspace(uniqueWorkspaceName("qa-record-empty"))
	core.RequireTrue(t, wsResult.OK)
	t.Cleanup(workspace.Discard)

	s := newPrepWithProcess()
	// Empty report records nothing but must not panic.
	core.AssertNotPanics(t, func() {
		s.recordLintFindings(workspace, QAReport{})
	})
}

// --- recordBuildResult (real :memory: workspace happy path) ---

func TestQa_RecordBuildResult_Good_PersistsRow(t *testing.T) {
	result := store.New(":memory:")
	core.RequireTrue(t, result.OK)
	storeInstance := result.Value.(*store.Store)
	t.Cleanup(func() { _ = storeInstance.Close() })

	workspace, wsResult := storeInstance.NewWorkspace(uniqueWorkspaceName("qa-build-good"))
	core.RequireTrue(t, wsResult.OK)
	t.Cleanup(workspace.Discard)

	s := newPrepWithProcess()
	s.recordBuildResult(workspace, "build", true, "ok output")
	s.recordBuildResult(workspace, "test", false, "1 failure")

	aggregate := workspace.Aggregate()
	core.AssertEqual(t, 1, intValue(aggregate["build"]))
	core.AssertEqual(t, 1, intValue(aggregate["test"]))
}

// --- findingsFromJournalPayload (report-inline + nil arms) ---

func TestQa_FindingsFromJournalPayload_Good_TopLevelFindings(t *testing.T) {
	payload := map[string]any{
		"findings": []any{
			map[string]any{"tool": "gosec", "file": "a.go"},
		},
	}
	findings := findingsFromJournalPayload(payload)
	core.AssertLen(t, findings, 1)
	core.AssertEqual(t, "gosec", findings[0]["tool"])
}

func TestQa_FindingsFromJournalPayload_Good_NestedReportFallback(t *testing.T) {
	// Older cycles stored findings under a nested "report" key.
	payload := map[string]any{
		"report": map[string]any{
			"findings": []any{
				map[string]any{"tool": "staticcheck", "file": "b.go"},
			},
		},
	}
	findings := findingsFromJournalPayload(payload)
	core.AssertLen(t, findings, 1)
	core.AssertEqual(t, "staticcheck", findings[0]["tool"])
}

func TestQa_FindingsFromJournalPayload_Bad_NilAndEmpty(t *testing.T) {
	core.AssertNil(t, findingsFromJournalPayload(nil))
	core.AssertNil(t, findingsFromJournalPayload(map[string]any{}))
	// A report key with no findings still returns nil, not a panic.
	core.AssertNil(t, findingsFromJournalPayload(map[string]any{"report": map[string]any{}}))
}

// --- findingToMap (Column + RuleID + Title arms) ---

func TestQa_FindingToMap_Good_FullFinding(t *testing.T) {
	entry := findingToMap(QAFinding{
		Tool:     "gosec",
		File:     "a.go",
		Line:     42,
		Column:   7,
		Severity: "error",
		Code:     "G101",
		Message:  "hardcoded secret",
		Category: "security",
		RuleID:   "HARDCODED",
		Title:    "Hardcoded credentials",
	})

	core.AssertEqual(t, "gosec", entry["tool"])
	core.AssertEqual(t, 7, entry["column"])
	core.AssertEqual(t, "HARDCODED", entry["rule_id"])
	core.AssertEqual(t, "Hardcoded credentials", entry["title"])
}

func TestQa_FindingToMap_Bad_MinimalFinding(t *testing.T) {
	// Zero Column/RuleID/Title are omitted from the map.
	entry := findingToMap(QAFinding{Tool: "gosec", File: "a.go", Line: 1})
	_, hasColumn := entry["column"]
	_, hasRuleID := entry["rule_id"]
	_, hasTitle := entry["title"]
	core.AssertFalse(t, hasColumn)
	core.AssertFalse(t, hasRuleID)
	core.AssertFalse(t, hasTitle)
	core.AssertEqual(t, "gosec", entry["tool"])
}

// --- firstNonEmpty (all-empty arm) ---

func TestQa_FirstNonEmpty_Good_ReturnsFirstSet(t *testing.T) {
	core.AssertEqual(t, "b", firstNonEmpty("", "b", "c"))
	core.AssertEqual(t, "a", firstNonEmpty("a", "b"))
}

func TestQa_FirstNonEmpty_Bad_AllEmpty(t *testing.T) {
	core.AssertEqual(t, "", firstNonEmpty("", "", ""))
	core.AssertEqual(t, "", firstNonEmpty())
}

// --- qaWorkspaceName (empty WorkspaceName fallback to PathBase) ---

func TestQa_QaWorkspaceName_Ugly_RootEqualsWorkspace(t *testing.T) {
	// When the workspace dir equals the configured root, WorkspaceName returns
	// empty and the helper falls back to PathBase of the dir.
	previous := workspaceRootOverride
	t.Cleanup(func() { workspaceRootOverride = previous })
	setWorkspaceRootOverride("/srv/work")
	core.AssertEqual(t, "qa-work", qaWorkspaceName("/srv/work"))
}
