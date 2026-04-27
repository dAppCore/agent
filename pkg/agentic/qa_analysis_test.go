// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyseWorkspace_Good_EmptyFindings(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	subsystem := newPrepWithProcess()
	t.Cleanup(subsystem.closeStateStore)

	workspaceDir := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-empty")
	workspaceName := WorkspaceName(workspaceDir)
	workspace, err := subsystem.stateStoreInstance().NewWorkspace(qaWorkspaceName(workspaceDir))
	require.NoError(t, err)
	t.Cleanup(workspace.Discard)

	report := subsystem.analyseWorkspaceNamed(workspace, workspaceName)

	assert.Equal(t, workspaceName, report.Workspace)
	assert.Empty(t, report.Findings)
	assert.Empty(t, report.Clusters)
	assert.Empty(t, report.New)
	assert.Empty(t, report.Resolved)
	assert.Empty(t, report.Persistent)
	assert.Equal(t, 0, report.Summary["clusters"])
	assert.Equal(t, "0 findings across 0 clusters; 0 new, 0 resolved, 0 persistent", report.SummaryText)
}

func TestAnalyseWorkspace_Good_FiveClusters(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	subsystem := newPrepWithProcess()
	t.Cleanup(subsystem.closeStateStore)

	workspaceDir := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-five")
	workspaceName := WorkspaceName(workspaceDir)
	workspace, err := subsystem.stateStoreInstance().NewWorkspace(qaWorkspaceName(workspaceDir))
	require.NoError(t, err)
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
		require.NoError(t, workspace.Put("finding", findingToMap(finding)))
	}

	report := subsystem.analyseWorkspaceNamed(workspace, workspaceName)

	if assert.Len(t, report.Clusters, 5) {
		for _, cluster := range report.Clusters {
			assert.Equal(t, 1, cluster.Count)
		}
	}
	assert.Len(t, report.New, 4)
	assert.Empty(t, report.Resolved)
	assert.Len(t, report.Persistent, 1)
	assert.Equal(t, 5, report.Summary["clusters"])
	assert.Equal(t, 1, report.Summary["persistent"])
}

func TestAnalyseWorkspace_Bad_NilWorkspace(t *testing.T) {
	var subsystem *PrepSubsystem

	assert.NotPanics(t, func() {
		report := subsystem.analyseWorkspace(nil)
		assert.Empty(t, report.Workspace)
		assert.Empty(t, report.Findings)
		assert.Empty(t, report.Clusters)
		assert.Empty(t, report.New)
		assert.Empty(t, report.Resolved)
		assert.Empty(t, report.Persistent)
		assert.Equal(t, "0 findings across 0 clusters; 0 new, 0 resolved, 0 persistent", report.SummaryText)
	})
}

func TestAnalyseWorkspace_Ugly_PoindexterPanic(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	subsystem := newPrepWithProcess()
	t.Cleanup(subsystem.closeStateStore)

	workspaceDir := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-panic")
	workspaceName := WorkspaceName(workspaceDir)
	workspace, err := subsystem.stateStoreInstance().NewWorkspace(qaWorkspaceName(workspaceDir))
	require.NoError(t, err)
	t.Cleanup(workspace.Discard)

	require.NoError(t, workspace.Put("finding", findingToMap(QAFinding{
		Tool:     "gosec",
		Severity: "error",
		Category: "security-secret",
		Code:     "G101",
		File:     "panic.go",
		Line:     10,
		Message:  "hardcoded secret",
	})))

	previousClusterer := qaAnalysisClusterer
	qaAnalysisClusterer = func([]QAFinding) []DispatchCluster {
		panic("poindexter panic")
	}
	t.Cleanup(func() { qaAnalysisClusterer = previousClusterer })

	assert.NotPanics(t, func() {
		report := subsystem.analyseWorkspaceNamed(workspace, workspaceName)
		if assert.Len(t, report.Clusters, 1) {
			assert.Equal(t, 1, report.Clusters[0].Count)
		}
		assert.Equal(t, 1, report.Summary["clusters"])
	})
}
