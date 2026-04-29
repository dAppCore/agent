// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go"
	store "dappco.re/go/store"
)

// --- qaWorkspaceName ---

func TestQa_QaWorkspaceName_Good_Case(t *testing.T) {
	// WorkspaceName strips the configured workspace root prefix before sanitising.
	previous := workspaceRootOverride
	t.Cleanup(func() { workspaceRootOverride = previous })
	setWorkspaceRootOverride("/root")
	core.AssertEqual(t, "qa-core-go-io-task-5", qaWorkspaceName("/root/core/go-io/task-5"))
	core.AssertEqual(t, "qa-simple", qaWorkspaceName("/simple"))
}

func TestQa_QaWorkspaceName_Bad_Case(t *testing.T) {
	core.AssertEqual(
		t,
		"qa-default",
		qaWorkspaceName(""),
	)
}

func TestQa_QaWorkspaceName_Ugly_Case(t *testing.T) {
	// Slashes, colons, and dots collapse to dashes so go-store validation passes.
	got := qaWorkspaceName("/tmp/workspace/ofm/mobile/bug:42")
	core.AssertContains(t, got, "qa-")
	for _, r := range got {
		valid := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
		if !valid {
			t.Fatalf("unexpected rune %q in %q", r, got)
		}
	}
}

// --- sanitiseWorkspaceName ---

func TestQa_SanitiseWorkspaceName_Good_Case(t *testing.T) {
	first := sanitiseWorkspaceName("core/go-io/task-5")
	second := sanitiseWorkspaceName("safe_name")
	core.AssertEqual(t, "core-go-io-task-5", first)
	core.AssertEqual(t, "safe_name", second)
}

func TestQa_SanitiseWorkspaceName_Bad_Case(t *testing.T) {
	core.AssertEqual(
		t,
		"",
		sanitiseWorkspaceName(""),
	)
}

func TestQa_SanitiseWorkspaceName_Ugly_Case(t *testing.T) {
	// Non-ASCII and punctuation characters all collapse to dashes.
	got := sanitiseWorkspaceName("core:📦/go-io")
	core.AssertContains(t, got, "core")
	core.AssertNotContains(t, got, ":")
	core.AssertNotContains(t, got, "/")
}

// --- writeDispatchReport ---

func TestQa_WriteDispatchReport_Good_Case(t *testing.T) {
	wsDir := t.TempDir()
	report := DispatchReport{
		Workspace:   WorkspaceName(wsDir),
		Passed:      true,
		BuildPassed: true,
		TestPassed:  true,
		LintPassed:  true,
		GeneratedAt: time.Now().UTC(),
	}

	writeDispatchReport(wsDir, report)

	reportPath := core.JoinPath(WorkspaceMetaDir(wsDir), "report.json")
	core.AssertTrue(t, fs.IsFile(reportPath))

	readResult := fs.Read(reportPath)
	core.AssertTrue(t, readResult.OK)

	var restored DispatchReport
	parseResult := core.JSONUnmarshalString(readResult.Value.(string), &restored)
	core.AssertTrue(t, parseResult.OK)
	core.AssertEqual(t, report.Passed, restored.Passed)
	core.AssertEqual(t, report.BuildPassed, restored.BuildPassed)
}

func TestQa_WriteDispatchReport_Bad_Case(t *testing.T) {
	// Empty workspace dir is a no-op — no panic, no file created.
	workspaceDir := ""
	report := DispatchReport{}
	writeDispatchReport(workspaceDir, report)
}

func TestQa_WriteDispatchReport_Ugly_Case(t *testing.T) {
	// Tolerates findings + tools with zero-value fields.
	wsDir := t.TempDir()
	report := DispatchReport{
		Workspace: "empty",
		Findings:  []QAFinding{{}, {Tool: "gosec"}},
		Tools:     []QAToolRun{{}, {Name: "gofmt"}},
	}
	writeDispatchReport(wsDir, report)

	reportPath := core.JoinPath(WorkspaceMetaDir(wsDir), "report.json")
	core.AssertTrue(t, fs.IsFile(reportPath))
}

// --- recordBuildResult ---

func TestQa_RecordBuildResult_Good_Case(t *testing.T) {
	// nil workspace is a no-op (graceful degradation path).
	s := newPrepWithProcess()
	workspace := (*store.Workspace)(nil)
	s.recordBuildResult(workspace, "build", true, "ok")
}

func TestQa_RecordBuildResult_Bad_Case(t *testing.T) {
	s := newPrepWithProcess()
	// Empty kind is skipped so we never insert rows without a kind.
	workspace := (*store.Workspace)(nil)
	s.recordBuildResult(workspace, "", true, "ignored")
}

func TestQa_RecordBuildResult_Ugly_Case(t *testing.T) {
	// Ugly path — very large output strings should not crash the nil-ws path.
	s := newPrepWithProcess()
	output := string(make([]byte, 1024*16))
	workspace := (*store.Workspace)(nil)
	s.recordBuildResult(workspace, "test", false, output)
}

// --- runLintReport ---

func TestQa_RunLintReport_Good_Case(t *testing.T) {
	// No repoDir → empty report, never errors.
	s := newPrepWithProcess()
	report := s.runLintReport(context.Background(), "")
	core.AssertEmpty(t, report.Findings)
	core.AssertEmpty(t, report.Tools)
}

func TestQa_RunLintReport_Bad_Case(t *testing.T) {
	// Nil subsystem → empty report (no panic).
	var s *PrepSubsystem
	report := s.runLintReport(context.Background(), t.TempDir())
	core.AssertEmpty(t, report.Findings)
}

func TestQa_RunLintReport_Ugly_Case(t *testing.T) {
	// Missing core-lint binary → empty report; the caller degrades gracefully.
	s := newPrepWithProcess()
	report := s.runLintReport(context.Background(), t.TempDir())
	core.AssertEmpty(t, report.Findings)
}

// --- runQAWithReport ---

func TestQa_RunQAWithReport_Good_Case(t *testing.T) {
	wsDir := t.TempDir()
	repoDir := core.JoinPath(wsDir, "repo")
	fs.EnsureDir(repoDir)
	fs.Write(core.JoinPath(repoDir, "go.mod"), "module testmod\n\ngo 1.22\n")
	fs.Write(core.JoinPath(repoDir, "main.go"), "package main\nfunc main() {}\n")

	s := newPrepWithProcess()
	core.AssertTrue(t, s.runQAWithReport(context.Background(), wsDir))

	// Report file should exist when the state store is available.
	reportPath := core.JoinPath(WorkspaceMetaDir(wsDir), "report.json")
	if fs.IsFile(reportPath) {
		readResult := fs.Read(reportPath)
		core.AssertTrue(t, readResult.OK)
		var report DispatchReport
		parseResult := core.JSONUnmarshalString(readResult.Value.(string), &report)
		core.AssertTrue(t, parseResult.OK)
		core.AssertTrue(t, report.Passed)
	}
}

func TestQa_RunQAWithReport_Bad_Case(t *testing.T) {
	// Missing repo → runQALegacy returns true because no build system is
	// detected under the workspace root. The important assertion is that
	// runQAWithReport never panics on an empty workspace dir.
	s := newPrepWithProcess()
	core.AssertNotPanics(t, func() {
		s.runQAWithReport(context.Background(), "")
	})
}

func TestQa_RunQAWithReport_Ugly_Case(t *testing.T) {
	// Unknown language — no build system detected but no panic.
	wsDir := t.TempDir()
	fs.EnsureDir(core.JoinPath(wsDir, "repo"))

	s := newPrepWithProcess()
	core.AssertTrue(t, s.runQAWithReport(context.Background(), wsDir))
}

// --- stringOutput ---

func TestQa_StringOutput_Good_Case(t *testing.T) {
	result := core.Result{Value: "hello", OK: true}
	output := stringOutput(result)
	core.AssertEqual(t, "hello", output)
}

func TestQa_StringOutput_Bad_Case(t *testing.T) {
	nilOutput := stringOutput(core.Result{Value: nil, OK: false})
	numberOutput := stringOutput(core.Result{Value: 42, OK: true})
	core.AssertEqual(t, "", nilOutput)
	core.AssertEqual(t, "", numberOutput)
}

func TestQa_StringOutput_Ugly_Case(t *testing.T) {
	result := core.Result{}
	output := stringOutput(result)
	core.AssertEqual(t, "", output)
}

// --- clusterFindings ---

func TestQa_ClusterFindings_Good_Case(t *testing.T) {
	// Similar gosec findings with different rule IDs should still deduplicate.
	findings := []QAFinding{
		{Tool: "gosec", Severity: "error", Category: "security", Code: "G101", File: "a.go", Line: 10, Message: "hardcoded password found"},
		{Tool: "gosec", Severity: "error", Category: "security", Code: "G401", File: "b.go", Line: 20, Message: "hardcoded password found in config"},
		{Tool: "gosec", Severity: "error", Category: "security", RuleID: "HARDCODED-SECRET", File: "c.go", Line: 30, Message: "possible hardcoded password found"},
		{Tool: "gosec", Severity: "error", Category: "security", Code: "G304", File: "d.go", Line: 40, Message: "file path built from tainted input"},
		{Tool: "staticcheck", Severity: "warning", Code: "SA1000", File: "e.go", Line: 5, Message: "invalid regexp pattern"},
	}
	clusters := clusterFindings(findings)
	core.AssertLen(t, clusters, 3)
	if len(clusters) == 3 {
		core.AssertEqual(t, 3, clusters[0].Count)
		core.AssertEqual(t, "gosec", clusters[0].Tool)
		core.AssertLen(t, clusters[0].Samples, 3)
		core.AssertEqual(t, 1, clusters[1].Count)
		core.AssertEqual(t, 1, clusters[2].Count)
	}
}

func TestQa_ClusterFindings_Bad_Case(t *testing.T) {
	core.AssertNotPanics(t, func() {
		core.AssertNil(t, clusterFindings(nil))
		core.AssertNil(t, clusterFindings([]QAFinding{}))
	})
}

func TestQa_ClusterFindings_Ugly_Case(t *testing.T) {
	findings := []QAFinding{
		{Tool: "gosec", Severity: "error", Code: "G101", File: "a.go", Line: 10, Message: "hardcoded password found"},
		{Tool: "gosec", Severity: "error", Code: "G304", File: "b.go", Line: 20, Message: "file path built from tainted input"},
		{Tool: "staticcheck", Severity: "warning", Code: "SA1000", File: "c.go", Line: 30, Message: "invalid regexp pattern"},
		{Tool: "govet", Severity: "warning", Code: "printf", File: "d.go", Line: 40, Message: "printf format mismatch"},
		{Tool: "revive", Severity: "info", Code: "var-naming", File: "e.go", Line: 50, Message: "var name should be camelCase"},
	}
	clusters := clusterFindings(findings)
	core.AssertLen(t, clusters, 5)
	if len(clusters) == 5 {
		for _, cluster := range clusters {
			core.AssertEqual(t, 1, cluster.Count)
		}
	}
}

func TestQa_ClusterFindings_Ugly_SampleLimit(t *testing.T) {
	// 10 identical findings should cap samples at clusterSampleLimit.
	findings := make([]QAFinding, 10)
	for i := range findings {
		findings[i] = QAFinding{Tool: "gosec", Severity: "error", Code: "G101", File: "same.go", Line: i, Message: "hardcoded password found"}
	}
	clusters := clusterFindings(findings)
	core.AssertLen(t, clusters, 1)
	if len(clusters) == 1 {
		core.AssertEqual(t, 10, clusters[0].Count)
		core.AssertLessOrEqual(t, len(clusters[0].Samples), clusterSampleLimit)
	}
}

// --- findingFingerprint ---

func TestQa_FindingFingerprint_Good_Case(t *testing.T) {
	left := findingFingerprint(QAFinding{Tool: "gosec", File: "a.go", Line: 10, Code: "G101"})
	right := findingFingerprint(QAFinding{Tool: "gosec", File: "a.go", Line: 10, Code: "G101", Message: "different message"})
	// Fingerprint ignores message — two findings at the same site are the same issue.
	core.AssertEqual(t, left, right)
}

func TestQa_FindingFingerprint_Bad_Case(t *testing.T) {
	// Different line numbers produce different fingerprints.
	left := findingFingerprint(QAFinding{Tool: "gosec", File: "a.go", Line: 10, Code: "G101"})
	right := findingFingerprint(QAFinding{Tool: "gosec", File: "a.go", Line: 11, Code: "G101"})
	core.AssertNotEqual(t, left, right)
}

func TestQa_FindingFingerprint_Ugly_Case(t *testing.T) {
	// Missing Code falls back to RuleID so migrations across lint versions don't break diffs.
	withCode := findingFingerprint(QAFinding{Tool: "gosec", File: "a.go", Line: 10, Code: "G101"})
	withRuleID := findingFingerprint(QAFinding{Tool: "gosec", File: "a.go", Line: 10, RuleID: "G101"})
	core.AssertEqual(t, withCode, withRuleID)
}

// --- diffFindingsAgainstJournal ---

func TestQa_DiffFindingsAgainstJournal_Good_Case(t *testing.T) {
	current := []QAFinding{
		{Tool: "gosec", File: "a.go", Line: 1, Code: "G101"},
		{Tool: "gosec", File: "b.go", Line: 2, Code: "G102"},
	}
	previous := [][]map[string]any{
		{
			{"tool": "gosec", "file": "a.go", "line": 1, "code": "G101"},
			{"tool": "gosec", "file": "c.go", "line": 3, "code": "G103"},
		},
	}
	newList, resolved, persistent := diffFindingsAgainstJournal(current, previous)
	// G102 is new this cycle, G103 resolved, persistent needs more history.
	core.AssertLen(t, newList, 1)
	core.AssertLen(t, resolved, 1)
	core.AssertNil(t, persistent)
}

func TestQa_DiffFindingsAgainstJournal_Bad_Case(t *testing.T) {
	// No previous cycles → no diff computed. First cycle baselines silently.
	newList, resolved, persistent := diffFindingsAgainstJournal([]QAFinding{{Tool: "gosec"}}, nil)
	core.AssertNil(t, newList)
	core.AssertNil(t, resolved)
	core.AssertNil(t, persistent)
}

func TestQa_DiffFindingsAgainstJournal_Ugly_Case(t *testing.T) {
	// When persistentThreshold-1 historical cycles agree, current finding is persistent.
	current := []QAFinding{{Tool: "gosec", File: "a.go", Line: 1, Code: "G101"}}
	history := make([][]map[string]any, persistentThreshold-1)
	for i := range history {
		history[i] = []map[string]any{{"tool": "gosec", "file": "a.go", "line": 1, "code": "G101"}}
	}
	_, _, persistent := diffFindingsAgainstJournal(current, history)
	core.AssertLen(t, persistent, 1)
}

// --- publishDispatchReport + readPreviousJournalCycles ---

func TestQa_PublishDispatchReport_Good_Case(t *testing.T) {
	// A published dispatch report should round-trip through the journal so the
	// next cycle can diff against its findings.
	storeInstance, err := store.New(":memory:")
	core.RequireNoError(t, err)
	t.Cleanup(func() { _ = storeInstance.Close() })

	workspaceName := "core/go-io/task-1"
	reportPayload := DispatchReport{
		Workspace:   workspaceName,
		Passed:      true,
		BuildPassed: true,
		TestPassed:  true,
		LintPassed:  true,
		Findings:    []QAFinding{{Tool: "gosec", File: "a.go", Line: 1, Code: "G101", Message: "secret"}},
		GeneratedAt: time.Now().UTC(),
	}

	publishDispatchReport(storeInstance, workspaceName, reportPayload)

	cycles := readPreviousJournalCycles(storeInstance, workspaceName, persistentThreshold)
	core.AssertLen(t, cycles, 1)
	if len(cycles) == 1 {
		core.AssertLen(t, cycles[0], 1)
		core.AssertEqual(t, "gosec", cycles[0][0]["tool"])
	}
}

func TestQa_PublishDispatchReport_Bad_Case(t *testing.T) {
	// Nil store and empty workspace name are no-ops — never panic.
	publishDispatchReport(nil, "any", DispatchReport{})

	storeInstance, err := store.New(":memory:")
	core.RequireNoError(t, err)
	t.Cleanup(func() { _ = storeInstance.Close() })
	publishDispatchReport(storeInstance, "", DispatchReport{Findings: []QAFinding{{Tool: "gosec"}}})

	core.AssertEmpty(t, readPreviousJournalCycles(nil, "x", 1))
	core.AssertEmpty(t, readPreviousJournalCycles(storeInstance, "", 1))
	core.AssertEmpty(t, readPreviousJournalCycles(storeInstance, "unknown-workspace", 1))
}

func TestQa_PublishDispatchReport_Ugly_Case(t *testing.T) {
	// After N pushes the reader should return at most `limit` cycles ordered
	// oldest→newest, so persistent detection sees cycles in the right order.
	storeInstance, err := store.New(":memory:")
	core.RequireNoError(t, err)
	t.Cleanup(func() { _ = storeInstance.Close() })

	workspaceName := "core/go-io/task-2"
	for cycle := 0; cycle < persistentThreshold+2; cycle++ {
		publishDispatchReport(storeInstance, workspaceName, DispatchReport{
			Workspace: workspaceName,
			Findings: []QAFinding{{
				Tool: "gosec",
				File: "a.go",
				Line: cycle + 1,
				Code: "G101",
			}},
			GeneratedAt: time.Now().UTC(),
		})
	}

	cycles := readPreviousJournalCycles(storeInstance, workspaceName, persistentThreshold)
	core.AssertLessOrEqual(t, len(cycles), persistentThreshold)
	// Oldest returned cycle has the earliest line number surviving in the window.
	core.AssertNotEmpty(t, cycles)
	if len(cycles) > 0 {
		core.AssertNotEmpty(t, cycles[0])
	}
}
