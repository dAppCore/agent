// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"time"

	core "dappco.re/go/core"
	store "dappco.re/go/core/store"
)

// QAFinding mirrors the lint.Finding shape produced by `core-lint run --output json`.
// Only the fields consumed by the agent pipeline are captured — the full lint
// report is persisted to the workspace buffer for post-run analysis.
//
// Usage example: `finding := QAFinding{Tool: "gosec", File: "main.go", Line: 42, Severity: "error", Code: "G101", Message: "hardcoded secret"}`
type QAFinding struct {
	Tool     string `json:"tool,omitempty"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	Severity string `json:"severity,omitempty"`
	Code     string `json:"code,omitempty"`
	Message  string `json:"message,omitempty"`
	Category string `json:"category,omitempty"`
	RuleID   string `json:"rule_id,omitempty"`
	Title    string `json:"title,omitempty"`
}

// QAToolRun mirrors lint.ToolRun — captures each adapter's execution status so
// the journal records which linters participated in the cycle.
//
// Usage example: `toolRun := QAToolRun{Name: "gosec", Status: "ok", Duration: "2.1s", Findings: 3}`
type QAToolRun struct {
	Name     string `json:"name"`
	Version  string `json:"version,omitempty"`
	Status   string `json:"status"`
	Duration string `json:"duration,omitempty"`
	Findings int    `json:"findings"`
}

// QASummary mirrors lint.Summary — the aggregate counts by severity.
//
// Usage example: `summary := QASummary{Total: 12, Errors: 3, Warnings: 5, Info: 4, Passed: false}`
type QASummary struct {
	Total    int  `json:"total"`
	Errors   int  `json:"errors"`
	Warnings int  `json:"warnings"`
	Info     int  `json:"info"`
	Passed   bool `json:"passed"`
}

// QAReport mirrors lint.Report — the shape emitted by `core-lint run --output json`.
// The agent parses this JSON to capture raw findings into the workspace buffer.
//
// Usage example: `report := QAReport{}; core.JSONUnmarshalString(jsonOutput, &report)`
type QAReport struct {
	Project   string      `json:"project"`
	Timestamp time.Time   `json:"timestamp"`
	Duration  string      `json:"duration"`
	Languages []string    `json:"languages"`
	Tools     []QAToolRun `json:"tools"`
	Findings  []QAFinding `json:"findings"`
	Summary   QASummary   `json:"summary"`
}

// DispatchReport summarises the raw findings captured in the workspace buffer
// before the cycle commits to the journal. Written to `.meta/report.json` so
// human reviewers and downstream tooling (Uptelligence, Poindexter) can read
// the cycle without re-scanning the buffer.
//
// Usage example: `report := DispatchReport{Summary: map[string]any{"finding": 12, "tool_run": 3}, Findings: findings, Tools: tools, BuildPassed: true, TestPassed: true}`
type DispatchReport struct {
	Workspace   string         `json:"workspace"`
	Commit      string         `json:"commit,omitempty"`
	Summary     map[string]any `json:"summary"`
	Findings    []QAFinding    `json:"findings,omitempty"`
	Tools       []QAToolRun    `json:"tools,omitempty"`
	BuildPassed bool           `json:"build_passed"`
	TestPassed  bool           `json:"test_passed"`
	LintPassed  bool           `json:"lint_passed"`
	Passed      bool           `json:"passed"`
	GeneratedAt time.Time      `json:"generated_at"`
}

// qaWorkspaceName returns the buffer name used to accumulate a single QA cycle.
// Mirrors RFC §7 — workspaces are namespaced `qa-<workspace-name>` so multiple
// concurrent dispatches produce distinct DuckDB files.
//
// Usage example: `name := qaWorkspaceName("/tmp/workspace/core/go-io/task-5") // "qa-core-go-io-task-5"`
func qaWorkspaceName(workspaceDir string) string {
	if workspaceDir == "" {
		return "qa-default"
	}
	name := WorkspaceName(workspaceDir)
	if name == "" {
		name = core.PathBase(workspaceDir)
	}
	return core.Concat("qa-", sanitiseWorkspaceName(name))
}

// sanitiseWorkspaceName replaces characters that collide with the go-store
// workspace name validator (`^[a-zA-Z0-9_-]+$`).
//
// Usage example: `clean := sanitiseWorkspaceName("core/go-io/task-5") // "core-go-io-task-5"`
func sanitiseWorkspaceName(name string) string {
	runes := []rune(name)
	for index, value := range runes {
		switch {
		case (value >= 'a' && value <= 'z') || (value >= 'A' && value <= 'Z'),
			value >= '0' && value <= '9',
			value == '-', value == '_':
			continue
		default:
			runes[index] = '-'
		}
	}
	return string(runes)
}

// runLintReport runs `core-lint run --output json` against the repo directory
// and returns the decoded QAReport. When the binary is missing or fails, the
// report comes back empty so the QA pipeline still records build/test without
// crashing — RFC §15.6 graceful degradation applies to the QA step too.
//
// Usage example: `report := s.runLintReport(ctx, "/workspace/repo")`
func (s *PrepSubsystem) runLintReport(ctx context.Context, repoDir string) QAReport {
	if repoDir == "" {
		return QAReport{}
	}
	if s == nil || s.Core() == nil {
		return QAReport{}
	}

	result := s.Core().Process().RunIn(ctx, repoDir, "core-lint", "run", "--output", "json", "--path", repoDir)
	if !result.OK || result.Value == nil {
		return QAReport{}
	}

	output, ok := result.Value.(string)
	if !ok || core.Trim(output) == "" {
		return QAReport{}
	}

	var report QAReport
	if parseResult := core.JSONUnmarshalString(output, &report); !parseResult.OK {
		return QAReport{}
	}
	return report
}

// recordLintFindings streams every lint.Finding and every lint.ToolRun into the
// workspace buffer per RFC §7 "QA handler — runs lint, captures all findings
// to workspace store". The store is optional — when go-store is not loaded the
// caller skips this step and falls back to the simple pass/fail run.
//
// Usage example: `s.recordLintFindings(workspace, report)`
func (s *PrepSubsystem) recordLintFindings(workspace *store.Workspace, report QAReport) {
	if workspace == nil {
		return
	}
	for _, finding := range report.Findings {
		_ = workspace.Put("finding", map[string]any{
			"tool":     finding.Tool,
			"file":     finding.File,
			"line":     finding.Line,
			"column":   finding.Column,
			"severity": finding.Severity,
			"code":     finding.Code,
			"message":  finding.Message,
			"category": finding.Category,
			"rule_id":  finding.RuleID,
			"title":    finding.Title,
		})
	}
	for _, tool := range report.Tools {
		_ = workspace.Put("tool_run", map[string]any{
			"name":     tool.Name,
			"version":  tool.Version,
			"status":   tool.Status,
			"duration": tool.Duration,
			"findings": tool.Findings,
		})
	}
}

// recordBuildResult persists a build/test cycle row so downstream analysis can
// correlate failures with specific findings.
//
// Usage example: `s.recordBuildResult(workspace, "build", true, "")`
func (s *PrepSubsystem) recordBuildResult(workspace *store.Workspace, kind string, passed bool, output string) {
	if workspace == nil || kind == "" {
		return
	}
	_ = workspace.Put(kind, map[string]any{
		"passed": passed,
		"output": output,
	})
}

// runQAWithReport extends runQA with the RFC §7 capture pipeline — it opens a
// go-store workspace buffer, records every lint finding, build, and test
// result, writes `.meta/report.json`, and commits the cycle to the journal.
// The returned bool matches the existing runQA contract so callers need no
// migration. When go-store is unavailable, the function degrades to the
// simple build/vet/test pass/fail path per RFC §15.6.
//
// Usage example: `passed := s.runQAWithReport(ctx, "/workspace/core/go-io/task-5")`
func (s *PrepSubsystem) runQAWithReport(ctx context.Context, workspaceDir string) bool {
	if workspaceDir == "" {
		return false
	}

	repoDir := WorkspaceRepoDir(workspaceDir)
	if !fs.IsDir(repoDir) {
		return s.runQALegacy(ctx, workspaceDir)
	}

	storeInstance := s.stateStoreInstance()
	if storeInstance == nil {
		return s.runQALegacy(ctx, workspaceDir)
	}

	workspace, err := storeInstance.NewWorkspace(qaWorkspaceName(workspaceDir))
	if err != nil {
		return s.runQALegacy(ctx, workspaceDir)
	}

	report := s.runLintReport(ctx, repoDir)
	s.recordLintFindings(workspace, report)

	buildPassed, testPassed := s.runBuildAndTest(ctx, workspace, repoDir)
	lintPassed := report.Summary.Errors == 0

	dispatchReport := DispatchReport{
		Workspace:   WorkspaceName(workspaceDir),
		Summary:     workspace.Aggregate(),
		Findings:    report.Findings,
		Tools:       report.Tools,
		BuildPassed: buildPassed,
		TestPassed:  testPassed,
		LintPassed:  lintPassed,
		Passed:      buildPassed && testPassed,
		GeneratedAt: time.Now().UTC(),
	}

	writeDispatchReport(workspaceDir, dispatchReport)

	commitResult := workspace.Commit()
	if !commitResult.OK {
		// Commit failed — make sure the buffer does not leak on disk.
		workspace.Discard()
	}

	return dispatchReport.Passed
}

// runBuildAndTest executes the language-specific build/test cycle, recording
// each outcome into the workspace buffer. Mirrors the existing runQA decision
// tree (Go > composer > npm > passthrough) so the captured data matches what
// previously determined pass/fail.
//
// Usage example: `buildPassed, testPassed := s.runBuildAndTest(ctx, ws, "/workspace/repo")`
func (s *PrepSubsystem) runBuildAndTest(ctx context.Context, workspace *store.Workspace, repoDir string) (bool, bool) {
	process := s.Core().Process()

	switch {
	case fs.IsFile(core.JoinPath(repoDir, "go.mod")):
		buildResult := process.RunIn(ctx, repoDir, "go", "build", "./...")
		s.recordBuildResult(workspace, "build", buildResult.OK, stringOutput(buildResult))
		if !buildResult.OK {
			return false, false
		}
		vetResult := process.RunIn(ctx, repoDir, "go", "vet", "./...")
		s.recordBuildResult(workspace, "vet", vetResult.OK, stringOutput(vetResult))
		if !vetResult.OK {
			return false, false
		}
		testResult := process.RunIn(ctx, repoDir, "go", "test", "./...", "-count=1", "-timeout", "120s")
		s.recordBuildResult(workspace, "test", testResult.OK, stringOutput(testResult))
		return true, testResult.OK
	case fs.IsFile(core.JoinPath(repoDir, "composer.json")):
		installResult := process.RunIn(ctx, repoDir, "composer", "install", "--no-interaction")
		s.recordBuildResult(workspace, "build", installResult.OK, stringOutput(installResult))
		if !installResult.OK {
			return false, false
		}
		testResult := process.RunIn(ctx, repoDir, "composer", "test")
		s.recordBuildResult(workspace, "test", testResult.OK, stringOutput(testResult))
		return true, testResult.OK
	case fs.IsFile(core.JoinPath(repoDir, "package.json")):
		installResult := process.RunIn(ctx, repoDir, "npm", "install")
		s.recordBuildResult(workspace, "build", installResult.OK, stringOutput(installResult))
		if !installResult.OK {
			return false, false
		}
		testResult := process.RunIn(ctx, repoDir, "npm", "test")
		s.recordBuildResult(workspace, "test", testResult.OK, stringOutput(testResult))
		return true, testResult.OK
	default:
		// No build system detected — record a passthrough outcome so the
		// journal still sees the cycle.
		s.recordBuildResult(workspace, "build", true, "no build system detected")
		s.recordBuildResult(workspace, "test", true, "no build system detected")
		return true, true
	}
}

// runQALegacy is the original build/vet/test cascade used when the go-store
// buffer is unavailable. Keeps the old behaviour so offline deployments and
// tests without a state store still pass.
//
// Usage example: `passed := s.runQALegacy(ctx, "/workspace/core/go-io/task-5")`
func (s *PrepSubsystem) runQALegacy(ctx context.Context, workspaceDir string) bool {
	repoDir := WorkspaceRepoDir(workspaceDir)
	process := s.Core().Process()

	if fs.IsFile(core.JoinPath(repoDir, "go.mod")) {
		for _, args := range [][]string{
			{"go", "build", "./..."},
			{"go", "vet", "./..."},
			{"go", "test", "./...", "-count=1", "-timeout", "120s"},
		} {
			if !process.RunIn(ctx, repoDir, args[0], args[1:]...).OK {
				core.Warn("QA failed", "cmd", core.Join(" ", args...))
				return false
			}
		}
		return true
	}

	if fs.IsFile(core.JoinPath(repoDir, "composer.json")) {
		if !process.RunIn(ctx, repoDir, "composer", "install", "--no-interaction").OK {
			return false
		}
		return process.RunIn(ctx, repoDir, "composer", "test").OK
	}

	if fs.IsFile(core.JoinPath(repoDir, "package.json")) {
		if !process.RunIn(ctx, repoDir, "npm", "install").OK {
			return false
		}
		return process.RunIn(ctx, repoDir, "npm", "test").OK
	}

	return true
}

// writeDispatchReport serialises the DispatchReport to `.meta/report.json` so
// the human-readable record survives the workspace buffer being committed and
// discarded. Per RFC §7: "the intelligence survives in the report and the
// journal".
//
// Usage example: `writeDispatchReport("/workspace/core/go-io/task-5", report)`
func writeDispatchReport(workspaceDir string, report DispatchReport) {
	if workspaceDir == "" {
		return
	}
	metaDir := WorkspaceMetaDir(workspaceDir)
	if !fs.EnsureDir(metaDir).OK {
		return
	}
	payload := core.JSONMarshalString(report)
	if payload == "" {
		return
	}
	fs.WriteAtomic(core.JoinPath(metaDir, "report.json"), payload)
}

// stringOutput extracts the process output from a core.Result, returning an
// empty string when the value is not a string (e.g. nil on spawn failure).
//
// Usage example: `output := stringOutput(process.RunIn(ctx, dir, "go", "build"))`
func stringOutput(result core.Result) string {
	if result.Value == nil {
		return ""
	}
	if value, ok := result.Value.(string); ok {
		return value
	}
	return ""
}
