// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"
	"time"

	core "dappco.re/go"
)

func TestAnalyseWorkspace_Good_EmptyFindings(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	subsystem := newPrepWithProcess()
	t.Cleanup(subsystem.closeStateStore)

	workspaceDir := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-empty")
	workspaceName := WorkspaceName(workspaceDir)
	workspace, result := subsystem.stateStoreInstance().NewWorkspace(qaWorkspaceName(workspaceDir))
	if !result.OK {
		t.Fatalf("create QA workspace: %v", resultErrorValue("TestAnalyseWorkspace_Good_EmptyFindings", result))
	}
	t.Cleanup(workspace.Discard)

	report := subsystem.analyseWorkspaceNamed(workspace, workspaceName)

	core.AssertEqual(t, workspaceName, report.Workspace)
	core.AssertEmpty(t, report.Findings)
	core.AssertEmpty(t, report.Clusters)
	core.AssertEmpty(t, report.New)
	core.AssertEmpty(t, report.Resolved)
	core.AssertEmpty(t, report.Persistent)
	core.AssertEqual(t, 0, report.Summary["clusters"])
	core.AssertEqual(t, "0 findings across 0 clusters; 0 new, 0 resolved, 0 persistent", report.SummaryText)
}

func TestAnalyseWorkspace_Good_FiveClusters(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	subsystem := newPrepWithProcess()
	t.Cleanup(subsystem.closeStateStore)

	workspaceDir := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-five")
	workspaceName := WorkspaceName(workspaceDir)
	workspace, result := subsystem.stateStoreInstance().NewWorkspace(qaWorkspaceName(workspaceDir))
	if !result.OK {
		t.Fatalf("create QA workspace: %v", resultErrorValue("TestAnalyseWorkspace_Good_FiveClusters", result))
	}
	t.Cleanup(workspace.Discard)

	repeated := QAFinding{Tool: "gosec", Severity: "error", Category: "security-secret", Code: "G101", File: "secret.go", Line: 10, Message: "hardcoded secret"}
	for cycle := 0; cycle < persistentThreshold-1; cycle++ {
		publishDispatchReport(subsystem.stateStoreInstance(), workspaceName, DispatchReport{
			Workspace:   workspaceName,
			Findings:    []QAFinding{repeated},
			GeneratedAt: time.Now().UTC(),
		})
	}

	currentFindings := []QAFinding{
		repeated,
		{Tool: "gosec", Severity: "error", Category: "security-path", Code: "G304", File: "path.go", Line: 20, Message: "tainted path"},
		{Tool: "staticcheck", Severity: "warning", Category: "correctness-regexp", Code: "SA1000", File: "regexp.go", Line: 30, Message: "invalid regexp"},
		{Tool: "govet", Severity: "warning", Category: "printf", Code: "printf", File: "printf.go", Line: 40, Message: "printf mismatch"},
		{Tool: "revive", Severity: "info", Category: "var-naming", Code: "var-naming", File: "style.go", Line: 50, Message: "bad variable name"},
	}
	for _, finding := range currentFindings {
		if result := workspace.Put("finding", findingToMap(finding)); !result.OK {
			t.Fatalf("put finding: %v", resultErrorValue("TestAnalyseWorkspace_Good_FiveClusters", result))
		}
	}

	report := subsystem.analyseWorkspaceNamed(workspace, workspaceName)

	core.AssertLen(t, report.Clusters, 5)
	if len(report.Clusters) == 5 {
		for _, cluster := range report.Clusters {
			core.AssertEqual(t, 1, cluster.Count)
		}
	}
	core.AssertLen(t, report.New, 4)
	core.AssertEmpty(t, report.Resolved)
	core.AssertLen(t, report.Persistent, 1)
	core.AssertEqual(t, 5, report.Summary["clusters"])
	core.AssertEqual(t, 1, report.Summary["persistent"])
}

func TestAnalyseWorkspace_Bad_NilWorkspace(t *testing.T) {
	var subsystem *PrepSubsystem

	core.AssertNotPanics(t, func() {
		report := subsystem.analyseWorkspace(nil)
		core.AssertEmpty(t, report.Workspace)
		core.AssertEmpty(t, report.Findings)
		core.AssertEmpty(t, report.Clusters)
		core.AssertEmpty(t, report.New)
		core.AssertEmpty(t, report.Resolved)
		core.AssertEmpty(t, report.Persistent)
		core.AssertEqual(t, "0 findings across 0 clusters; 0 new, 0 resolved, 0 persistent", report.SummaryText)
	})
}

func TestAnalyseWorkspace_Ugly_PoindexterPanic(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	subsystem := newPrepWithProcess()
	t.Cleanup(subsystem.closeStateStore)

	workspaceDir := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-panic")
	workspaceName := WorkspaceName(workspaceDir)
	workspace, result := subsystem.stateStoreInstance().NewWorkspace(qaWorkspaceName(workspaceDir))
	if !result.OK {
		t.Fatalf("create QA workspace: %v", resultErrorValue("TestAnalyseWorkspace_Ugly_PoindexterPanic", result))
	}
	t.Cleanup(workspace.Discard)

	if result := workspace.Put("finding", findingToMap(QAFinding{
		Tool:     "gosec",
		Severity: "error",
		Category: "security-secret",
		Code:     "G101",
		File:     "panic.go",
		Line:     10,
		Message:  "hardcoded secret",
	})); !result.OK {
		t.Fatalf("put finding: %v", resultErrorValue("TestAnalyseWorkspace_Ugly_PoindexterPanic", result))
	}

	previousClusterer := qaAnalysisClusterer
	qaAnalysisClusterer = func([]QAFinding) []DispatchCluster {
		panic("poindexter panic")
	}
	t.Cleanup(func() { qaAnalysisClusterer = previousClusterer })

	core.AssertNotPanics(t, func() {
		report := subsystem.analyseWorkspaceNamed(workspace, workspaceName)
		core.AssertLen(t, report.Clusters, 1)
		if len(report.Clusters) == 1 {
			core.AssertEqual(t, 1, report.Clusters[0].Count)
		}
		core.AssertEqual(t, 1, report.Summary["clusters"])
	})
}
