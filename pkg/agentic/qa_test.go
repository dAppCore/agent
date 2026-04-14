// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
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
