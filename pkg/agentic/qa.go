// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"time"

	core "dappco.re/go"
	store "dappco.re/go/store"
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
	Errors   int  "json:\"errors\""
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
// Per RFC §7 Post-Run Analysis, the report contrasts the current cycle with
// previous journal entries for the same workspace to surface what changed:
// `New` lists findings absent from the previous cycle, `Resolved` lists
// findings the previous cycle had that this cycle cleared, and `Persistent`
// lists findings that appear across the last `persistentThreshold` cycles.
// When the journal has no history the diff lists are left nil so the first
// cycle behaves like a fresh baseline.
//
// Usage example: `report := DispatchReport{Summary: map[string]any{"finding": 12, "tool_run": 3}, Findings: findings, Tools: tools, BuildPassed: true, TestPassed: true}`
type DispatchReport struct {
	Workspace   string            `json:"workspace"`
	Commit      string            `json:"commit,omitempty"`
	Summary     map[string]any    `json:"summary"`
	SummaryText string            `json:"summary_text,omitempty"`
	Findings    []QAFinding       `json:"findings,omitempty"`
	Tools       []QAToolRun       `json:"tools,omitempty"`
	BuildPassed bool              `json:"build_passed"`
	TestPassed  bool              `json:"test_passed"`
	LintPassed  bool              `json:"lint_passed"`
	Passed      bool              `json:"passed"`
	GeneratedAt time.Time         `json:"generated_at"`
	New         []map[string]any  `json:"new,omitempty"`
	Resolved    []map[string]any  `json:"resolved,omitempty"`
	Persistent  []map[string]any  `json:"persistent,omitempty"`
	Clusters    []DispatchCluster `json:"clusters,omitempty"`
}

// DispatchCluster groups similar findings together so human reviewers can see
// recurring problem shapes without scanning every raw finding. Clusters are
// built from Poindexter KD-tree similarity over hashed finding features, then
// summarised back into the dominant tool/severity/category/rule identity with
// representative samples.
//
// Usage example: `cluster := DispatchCluster{Tool: "gosec", Severity: "error", Category: "security", Count: 3, RuleID: "G101"}`
type DispatchCluster struct {
	Tool     string                  `json:"tool,omitempty"`
	Severity string                  `json:"severity,omitempty"`
	Category string                  `json:"category,omitempty"`
	RuleID   string                  `json:"rule_id,omitempty"`
	Count    int                     `json:"count"`
	Samples  []DispatchClusterSample `json:"samples,omitempty"`
}

// DispatchClusterSample is a minimal projection of a finding inside a
// DispatchCluster so reviewers can jump to the file/line without
// re-scanning the full findings list.
//
// Usage example: `sample := DispatchClusterSample{File: "main.go", Line: 42, Message: "hardcoded secret"}`
type DispatchClusterSample struct {
	File    string `json:"file,omitempty"`
	Line    int    `json:"line,omitempty"`
	Message string `json:"message,omitempty"`
}

// persistentThreshold matches RFC §7 — findings that appear in this many
// consecutive cycles for the same workspace are classed as persistent and
// surfaced separately so Uptelligence can flag chronic issues.
const persistentThreshold = 5

// clusterSampleLimit caps how many representative findings accompany a
// DispatchCluster so the `.meta/report.json` payload stays bounded even when
// a single rule fires hundreds of times.
const clusterSampleLimit = 3

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
		if err := workspace.Put("finding", map[string]any{
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
		}); err != nil {
			core.Warn("agentic: failed to persist lint finding", "workspace", workspace.Name(), "reason", err)
		}
	}
	for _, tool := range report.Tools {
		if err := workspace.Put("tool_run", map[string]any{
			"name":     tool.Name,
			"version":  tool.Version,
			"status":   tool.Status,
			"duration": tool.Duration,
			"findings": tool.Findings,
		}); err != nil {
			core.Warn("agentic: failed to persist tool run", "workspace", workspace.Name(), "reason", err)
		}
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
	if err := workspace.Put(kind, map[string]any{
		"passed": passed,
		"output": output,
	}); err != nil {
		core.Warn("agentic: failed to persist build result", "workspace", workspace.Name(), "kind", kind, "reason", err)
	}
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

	workspaceName := WorkspaceName(workspaceDir)
	dispatchReport := s.analyseWorkspaceNamed(workspace, workspaceName)
	dispatchReport.BuildPassed = buildPassed
	dispatchReport.TestPassed = testPassed
	dispatchReport.LintPassed = lintPassed
	dispatchReport.Passed = buildPassed && testPassed
	dispatchReport.GeneratedAt = time.Now().UTC()

	writeDispatchReport(workspaceDir, dispatchReport)

	// Publish the full dispatch report to the journal (keyed by workspace name)
	// so the next cycle's readPreviousJournalCycles can diff against a
	// findings-level payload rather than only the aggregated counts produced
	// by workspace.Commit(). Matches RFC §7 "the intelligence survives in the
	// report and the journal".
	publishDispatchReport(storeInstance, workspaceName, dispatchReport)

	commitResult := workspace.Commit()
	if !commitResult.OK {
		// Commit failed — make sure the buffer does not leak on disk.
		workspace.Discard()
	}

	return dispatchReport.Passed
}

// publishDispatchReport writes the dispatch report's findings, tools, and
// per-kind summary to the journal using Store.CommitToJournal. The measurement
// is the workspace name so later reads can filter by workspace, and the tags
// let Uptelligence group cycles across repos.
//
// Usage example: `publishDispatchReport(store, "core/go-io/task-5", dispatchReport)`
func publishDispatchReport(storeInstance *store.Store, workspaceName string, dispatchReport DispatchReport) {
	if storeInstance == nil || workspaceName == "" {
		return
	}

	findings := make([]map[string]any, 0, len(dispatchReport.Findings))
	for _, finding := range dispatchReport.Findings {
		findings = append(findings, findingToMap(finding))
	}

	tools := make([]map[string]any, 0, len(dispatchReport.Tools))
	for _, tool := range dispatchReport.Tools {
		tools = append(tools, map[string]any{
			"name":     tool.Name,
			"version":  tool.Version,
			"status":   tool.Status,
			"duration": tool.Duration,
			"findings": tool.Findings,
		})
	}

	fields := map[string]any{
		"passed":       dispatchReport.Passed,
		"build_passed": dispatchReport.BuildPassed,
		"test_passed":  dispatchReport.TestPassed,
		"lint_passed":  dispatchReport.LintPassed,
		"summary":      dispatchReport.Summary,
		"summary_text": dispatchReport.SummaryText,
		"findings":     findings,
		"tools":        tools,
		"clusters":     dispatchReport.Clusters,
		"new":          dispatchReport.New,
		"resolved":     dispatchReport.Resolved,
		"persistent":   dispatchReport.Persistent,
		"generated_at": dispatchReport.GeneratedAt.Format(time.RFC3339Nano),
	}
	tags := map[string]string{"workspace": workspaceName}

	storeInstance.CommitToJournal(workspaceName, fields, tags)
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
	if ensureResult := fs.EnsureDir(metaDir); !ensureResult.OK {
		core.Warn("agentic: failed to prepare dispatch report directory", `path`, metaDir, "reason", ensureResult.Value)
		return
	}
	payload := core.JSONMarshalString(report)
	if payload == "" {
		return
	}
	reportPath := core.JoinPath(metaDir, "report.json")
	if writeResult := fs.WriteAtomic(reportPath, payload); !writeResult.OK {
		core.Warn("agentic: failed to write dispatch report", `path`, reportPath, "reason", writeResult.Value)
	}
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

// findingFingerprint returns a stable key for a single lint finding so the
// diff and cluster helpers can compare current and previous cycles without
// confusing "two G101 hits in the same file" with "two identical findings".
// The fingerprint mirrors what human reviewers use to recognise the same
// issue across cycles — tool, file, line, rule/code.
//
// Usage example: `key := findingFingerprint(QAFinding{Tool: "gosec", File: "main.go", Line: 42, Code: "G101"})`
func findingFingerprint(finding QAFinding) string {
	return core.Sprintf("%s|%s|%d|%s", finding.Tool, finding.File, finding.Line, firstNonEmpty(finding.Code, finding.RuleID))
}

// findingFingerprintFromMap extracts a fingerprint from a journal-restored
// finding (which is a `map[string]any` rather than a typed struct). Keeps the
// diff helpers agnostic to how the finding was stored.
//
// Usage example: `key := findingFingerprintFromMap(map[string]any{"tool": "gosec", "file": "main.go", "line": 42, "code": "G101"})`
func findingFingerprintFromMap(entry map[string]any) string {
	return core.Sprintf(
		"%s|%s|%d|%s",
		stringValue(entry["tool"]),
		stringValue(entry["file"]),
		intValue(entry["line"]),
		firstNonEmpty(stringValue(entry["code"]), stringValue(entry["rule_id"])),
	)
}

// firstNonEmpty returns the first non-empty value in the arguments, or the
// empty string if all are empty. Lets fingerprint helpers fall back from
// `code` to `rule_id` without nested conditionals.
//
// Usage example: `value := firstNonEmpty(finding.Code, finding.RuleID)`
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// findingToMap turns a QAFinding into the map shape used by the diff output
// so journal-backed previous findings and current typed findings share a
// single representation in `.meta/report.json`.
//
// Usage example: `entry := findingToMap(QAFinding{Tool: "gosec", File: "main.go"})`
func findingToMap(finding QAFinding) map[string]any {
	entry := map[string]any{
		"tool":     finding.Tool,
		"file":     finding.File,
		"line":     finding.Line,
		"severity": finding.Severity,
		"code":     finding.Code,
		"message":  finding.Message,
		"category": finding.Category,
	}
	if finding.Column != 0 {
		entry["column"] = finding.Column
	}
	if finding.RuleID != "" {
		entry["rule_id"] = finding.RuleID
	}
	if finding.Title != "" {
		entry["title"] = finding.Title
	}
	return entry
}

// diffFindingsAgainstJournal compares the current cycle's findings with the
// previous cycles captured in the journal and returns the three RFC §7 lists
// (New, Resolved, Persistent). Empty journal history produces nil slices so
// the first cycle acts like a baseline rather than flagging every finding
// as new.
//
// Usage example: `newList, resolvedList, persistentList := diffFindingsAgainstJournal(current, previous)`
func diffFindingsAgainstJournal(current []QAFinding, previous [][]map[string]any) (newList, resolvedList, persistentList []map[string]any) {
	newList, resolvedList = diffFindings(current, previous)
	persistentList = persistentFindings(current, previous)
	return newList, resolvedList, persistentList
}

// readPreviousJournalCycles fetches the findings from the most recent `limit`
// journal commits for this workspace. Each cycle is returned as the slice of
// finding maps that ws.Put("finding", ...) recorded, so the diff helpers can
// treat journal entries the same way as in-memory findings.
//
// Usage example: `cycles := readPreviousJournalCycles(store, "core/go-io/task-5", 5)`
func readPreviousJournalCycles(storeInstance *store.Store, workspaceName string, limit int) [][]map[string]any {
	if storeInstance == nil || workspaceName == "" || limit <= 0 {
		return nil
	}

	queryString := core.Sprintf(
		`SELECT fields_json FROM journal_entries WHERE measurement = '%s' ORDER BY committed_at DESC, entry_id DESC LIMIT %d`,
		escapeJournalLiteral(workspaceName),
		limit,
	)
	result := storeInstance.QueryJournal(queryString)
	if !result.OK || result.Value == nil {
		return nil
	}

	rows, ok := result.Value.([]map[string]any)
	if !ok {
		return nil
	}

	cycles := make([][]map[string]any, 0, len(rows))
	for i := len(rows) - 1; i >= 0; i-- {
		raw := stringValue(rows[i]["fields_json"])
		if raw == "" {
			continue
		}
		var payload map[string]any
		if parseResult := core.JSONUnmarshalString(raw, &payload); !parseResult.OK {
			continue
		}
		cycles = append(cycles, findingsFromJournalPayload(payload))
	}
	return cycles
}

// findingsFromJournalPayload decodes the finding list out of a journal
// payload. The workspace.Commit aggregate only carries counts by kind, so
// this helper reads the companion `.meta/report.json` payload when it was
// synced into the journal (as sync.go records). Missing entries return an
// empty slice so older cycles without the enriched payload still allow the
// diff to complete.
//
// Usage example: `findings := findingsFromJournalPayload(map[string]any{"findings": []any{...}})`
func findingsFromJournalPayload(payload map[string]any) []map[string]any {
	if payload == nil {
		return nil
	}
	if findings := anyMapSliceValue(payload["findings"]); len(findings) > 0 {
		return findings
	}
	// Older cycles stored the full report inline — accept both shapes so the
	// diff still sees history during the rollout.
	if report, ok := payload["report"].(map[string]any); ok {
		return anyMapSliceValue(report["findings"])
	}
	return nil
}

// escapeJournalLiteral escapes single quotes in a SQL literal so QueryJournal
// can accept workspace names that contain them (rare but possible with
// hand-authored paths).
//
// Usage example: `safe := escapeJournalLiteral("core/go-io/task's-5")`
func escapeJournalLiteral(value string) string {
	return core.Replace(value, "'", "''")
}
