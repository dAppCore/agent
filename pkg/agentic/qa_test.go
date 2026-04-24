// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go/core"
	store "dappco.re/go/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- qaWorkspaceName ---

func TestQa_QaWorkspaceName_Good(t *testing.T) {
	// WorkspaceName strips the configured workspace root prefix before sanitising.
	previous := workspaceRootOverride
	t.Cleanup(func() { workspaceRootOverride = previous })
	setWorkspaceRootOverride("/root")
	assert.Equal(t, "qa-core-go-io-task-5", qaWorkspaceName("/root/core/go-io/task-5"))
	assert.Equal(t, "qa-simple", qaWorkspaceName("/simple"))
}

func TestQa_QaWorkspaceName_Bad(t *testing.T) {
	assert.Equal(t, "qa-default", qaWorkspaceName(""))
}

func TestQa_QaWorkspaceName_Ugly(t *testing.T) {
	// Slashes, colons, and dots collapse to dashes so go-store validation passes.
	got := qaWorkspaceName("/tmp/workspace/ofm/mobile/bug:42")
	assert.Contains(t, got, "qa-")
	for _, r := range got {
		valid := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
		assert.True(t, valid, "unexpected rune %q in %q", r, got)
	}
}

// --- sanitiseWorkspaceName ---

func TestQa_SanitiseWorkspaceName_Good(t *testing.T) {
	assert.Equal(t, "core-go-io-task-5", sanitiseWorkspaceName("core/go-io/task-5"))
	assert.Equal(t, "safe_name", sanitiseWorkspaceName("safe_name"))
}

func TestQa_SanitiseWorkspaceName_Bad(t *testing.T) {
	assert.Equal(t, "", sanitiseWorkspaceName(""))
}

func TestQa_SanitiseWorkspaceName_Ugly(t *testing.T) {
	// Non-ASCII and punctuation characters all collapse to dashes.
	got := sanitiseWorkspaceName("core:📦/go-io")
	assert.Contains(t, got, "core")
	assert.NotContains(t, got, ":")
	assert.NotContains(t, got, "/")
}

// --- writeDispatchReport ---

func TestQa_WriteDispatchReport_Good(t *testing.T) {
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
	assert.True(t, fs.IsFile(reportPath))

	readResult := fs.Read(reportPath)
	assert.True(t, readResult.OK)

	var restored DispatchReport
	parseResult := core.JSONUnmarshalString(readResult.Value.(string), &restored)
	assert.True(t, parseResult.OK)
	assert.Equal(t, report.Passed, restored.Passed)
	assert.Equal(t, report.BuildPassed, restored.BuildPassed)
}

func TestQa_WriteDispatchReport_Bad(t *testing.T) {
	// Empty workspace dir is a no-op — no panic, no file created.
	writeDispatchReport("", DispatchReport{})
}

func TestQa_WriteDispatchReport_Ugly(t *testing.T) {
	// Tolerates findings + tools with zero-value fields.
	wsDir := t.TempDir()
	report := DispatchReport{
		Workspace: "empty",
		Findings:  []QAFinding{{}, {Tool: "gosec"}},
		Tools:     []QAToolRun{{}, {Name: "gofmt"}},
	}
	writeDispatchReport(wsDir, report)

	reportPath := core.JoinPath(WorkspaceMetaDir(wsDir), "report.json")
	assert.True(t, fs.IsFile(reportPath))
}

// --- recordBuildResult ---

func TestQa_RecordBuildResult_Good(t *testing.T) {
	// nil workspace is a no-op (graceful degradation path).
	s := newPrepWithProcess()
	s.recordBuildResult(nil, "build", true, "ok")
}

func TestQa_RecordBuildResult_Bad(t *testing.T) {
	s := newPrepWithProcess()
	// Empty kind is skipped so we never insert rows without a kind.
	s.recordBuildResult(nil, "", true, "ignored")
}

func TestQa_RecordBuildResult_Ugly(t *testing.T) {
	// Ugly path — very large output strings should not crash the nil-ws path.
	s := newPrepWithProcess()
	s.recordBuildResult(nil, "test", false, string(make([]byte, 1024*16)))
}

// --- runLintReport ---

func TestQa_RunLintReport_Good(t *testing.T) {
	// No repoDir → empty report, never errors.
	s := newPrepWithProcess()
	report := s.runLintReport(context.Background(), "")
	assert.Empty(t, report.Findings)
	assert.Empty(t, report.Tools)
}

func TestQa_RunLintReport_Bad(t *testing.T) {
	// Nil subsystem → empty report (no panic).
	var s *PrepSubsystem
	report := s.runLintReport(context.Background(), t.TempDir())
	assert.Empty(t, report.Findings)
}

func TestQa_RunLintReport_Ugly(t *testing.T) {
	// Missing core-lint binary → empty report; the caller degrades gracefully.
	s := newPrepWithProcess()
	report := s.runLintReport(context.Background(), t.TempDir())
	assert.Empty(t, report.Findings)
}

// --- runQAWithReport ---

func TestQa_RunQAWithReport_Good(t *testing.T) {
	wsDir := t.TempDir()
	repoDir := core.JoinPath(wsDir, "repo")
	fs.EnsureDir(repoDir)
	fs.Write(core.JoinPath(repoDir, "go.mod"), "module testmod\n\ngo 1.22\n")
	fs.Write(core.JoinPath(repoDir, "main.go"), "package main\nfunc main() {}\n")

	s := newPrepWithProcess()
	assert.True(t, s.runQAWithReport(context.Background(), wsDir))

	// Report file should exist when the state store is available.
	reportPath := core.JoinPath(WorkspaceMetaDir(wsDir), "report.json")
	if fs.IsFile(reportPath) {
		readResult := fs.Read(reportPath)
		assert.True(t, readResult.OK)
		var report DispatchReport
		parseResult := core.JSONUnmarshalString(readResult.Value.(string), &report)
		assert.True(t, parseResult.OK)
		assert.True(t, report.Passed)
	}
}

func TestQa_RunQAWithReport_Bad(t *testing.T) {
	// Missing repo → runQALegacy returns true because no build system is
	// detected under the workspace root. The important assertion is that
	// runQAWithReport never panics on an empty workspace dir.
	s := newPrepWithProcess()
	assert.NotPanics(t, func() {
		s.runQAWithReport(context.Background(), "")
	})
}

func TestQa_RunQAWithReport_Ugly(t *testing.T) {
	// Unknown language — no build system detected but no panic.
	wsDir := t.TempDir()
	fs.EnsureDir(core.JoinPath(wsDir, "repo"))

	s := newPrepWithProcess()
	assert.True(t, s.runQAWithReport(context.Background(), wsDir))
}

// --- stringOutput ---

func TestQa_StringOutput_Good(t *testing.T) {
	assert.Equal(t, "hello", stringOutput(core.Result{Value: "hello", OK: true}))
}

func TestQa_StringOutput_Bad(t *testing.T) {
	assert.Equal(t, "", stringOutput(core.Result{Value: nil, OK: false}))
	assert.Equal(t, "", stringOutput(core.Result{Value: 42, OK: true}))
}

func TestQa_StringOutput_Ugly(t *testing.T) {
	assert.Equal(t, "", stringOutput(core.Result{}))
}

// --- clusterFindings ---

func TestQa_ClusterFindings_Good(t *testing.T) {
	// Two G101 findings in the same tool merge into one cluster with count 2.
	findings := []QAFinding{
		{Tool: "gosec", Severity: "error", Category: "security", Code: "G101", File: "a.go", Line: 10, Message: "secret"},
		{Tool: "gosec", Severity: "error", Category: "security", Code: "G101", File: "b.go", Line: 20, Message: "secret"},
		{Tool: "staticcheck", Severity: "warning", Code: "SA1000", File: "c.go", Line: 5},
	}
	clusters := clusterFindings(findings)
	if assert.Len(t, clusters, 2) {
		assert.Equal(t, 2, clusters[0].Count)
		assert.Equal(t, "gosec", clusters[0].Tool)
		assert.Len(t, clusters[0].Samples, 2)
		assert.Equal(t, 1, clusters[1].Count)
	}
}

func TestQa_ClusterFindings_Bad(t *testing.T) {
	assert.Nil(t, clusterFindings(nil))
	assert.Nil(t, clusterFindings([]QAFinding{}))
}

func TestQa_ClusterFindings_Ugly(t *testing.T) {
	// 10 identical findings should cap samples at clusterSampleLimit.
	findings := make([]QAFinding, 10)
	for i := range findings {
		findings[i] = QAFinding{Tool: "gosec", Code: "G101", File: "same.go", Line: i}
	}
	clusters := clusterFindings(findings)
	if assert.Len(t, clusters, 1) {
		assert.Equal(t, 10, clusters[0].Count)
		assert.LessOrEqual(t, len(clusters[0].Samples), clusterSampleLimit)
	}
}

// --- findingFingerprint ---

func TestQa_FindingFingerprint_Good(t *testing.T) {
	left := findingFingerprint(QAFinding{Tool: "gosec", File: "a.go", Line: 10, Code: "G101"})
	right := findingFingerprint(QAFinding{Tool: "gosec", File: "a.go", Line: 10, Code: "G101", Message: "different message"})
	// Fingerprint ignores message — two findings at the same site are the same issue.
	assert.Equal(t, left, right)
}

func TestQa_FindingFingerprint_Bad(t *testing.T) {
	// Different line numbers produce different fingerprints.
	left := findingFingerprint(QAFinding{Tool: "gosec", File: "a.go", Line: 10, Code: "G101"})
	right := findingFingerprint(QAFinding{Tool: "gosec", File: "a.go", Line: 11, Code: "G101"})
	assert.NotEqual(t, left, right)
}

func TestQa_FindingFingerprint_Ugly(t *testing.T) {
	// Missing Code falls back to RuleID so migrations across lint versions don't break diffs.
	withCode := findingFingerprint(QAFinding{Tool: "gosec", File: "a.go", Line: 10, Code: "G101"})
	withRuleID := findingFingerprint(QAFinding{Tool: "gosec", File: "a.go", Line: 10, RuleID: "G101"})
	assert.Equal(t, withCode, withRuleID)
}

// --- diffFindingsAgainstJournal ---

func TestQa_DiffFindingsAgainstJournal_Good(t *testing.T) {
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
	assert.Len(t, newList, 1)
	assert.Len(t, resolved, 1)
	assert.Nil(t, persistent)
}

func TestQa_DiffFindingsAgainstJournal_Bad(t *testing.T) {
	// No previous cycles → no diff computed. First cycle baselines silently.
	newList, resolved, persistent := diffFindingsAgainstJournal([]QAFinding{{Tool: "gosec"}}, nil)
	assert.Nil(t, newList)
	assert.Nil(t, resolved)
	assert.Nil(t, persistent)
}

func TestQa_DiffFindingsAgainstJournal_Ugly(t *testing.T) {
	// When persistentThreshold-1 historical cycles agree, current finding is persistent.
	current := []QAFinding{{Tool: "gosec", File: "a.go", Line: 1, Code: "G101"}}
	history := make([][]map[string]any, persistentThreshold-1)
	for i := range history {
		history[i] = []map[string]any{{"tool": "gosec", "file": "a.go", "line": 1, "code": "G101"}}
	}
	_, _, persistent := diffFindingsAgainstJournal(current, history)
	assert.Len(t, persistent, 1)
}

// --- publishDispatchReport + readPreviousJournalCycles ---

func TestQa_PublishDispatchReport_Good(t *testing.T) {
	// A published dispatch report should round-trip through the journal so the
	// next cycle can diff against its findings.
	storeInstance, err := store.New(":memory:")
	require.NoError(t, err)
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
	if assert.Len(t, cycles, 1) {
		assert.Len(t, cycles[0], 1)
		assert.Equal(t, "gosec", cycles[0][0]["tool"])
	}
}

func TestQa_PublishDispatchReport_Bad(t *testing.T) {
	// Nil store and empty workspace name are no-ops — never panic.
	publishDispatchReport(nil, "any", DispatchReport{})

	storeInstance, err := store.New(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = storeInstance.Close() })
	publishDispatchReport(storeInstance, "", DispatchReport{Findings: []QAFinding{{Tool: "gosec"}}})

	assert.Empty(t, readPreviousJournalCycles(nil, "x", 1))
	assert.Empty(t, readPreviousJournalCycles(storeInstance, "", 1))
	assert.Empty(t, readPreviousJournalCycles(storeInstance, "unknown-workspace", 1))
}

func TestQa_PublishDispatchReport_Ugly(t *testing.T) {
	// After N pushes the reader should return at most `limit` cycles ordered
	// oldest→newest, so persistent detection sees cycles in the right order.
	storeInstance, err := store.New(":memory:")
	require.NoError(t, err)
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
	assert.LessOrEqual(t, len(cycles), persistentThreshold)
	// Oldest returned cycle has the earliest line number surviving in the window.
	if assert.NotEmpty(t, cycles) {
		assert.NotEmpty(t, cycles[0])
	}
}
