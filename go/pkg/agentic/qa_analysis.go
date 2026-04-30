// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"hash/fnv"
	"maps"
	"time"

	core "dappco.re/go"
	store "dappco.re/go/store"
	poindexter "forge.lthn.ai/Snider/Poindexter"
)

const qaAnalysisClusterDistance = 0.15

type qaAnalysisPoint struct {
	Index     int
	ToolID    float64
	Severity  float64
	FileHash  float64
	Category  float64
	Frequency float64
}

var qaAnalysisClusterer = qaAnalysisClusters

// analyseWorkspace reads the buffered QA findings from the workspace DuckDB
// and returns the RFC §7 dispatch report. When called without the caller's
// original workspace name, the journal comparison falls back to the QA buffer
// name with the `qa-` prefix removed.
//
// Usage example: `report := s.analyseWorkspace(workspace)`
func (s *PrepSubsystem) analyseWorkspace(workspace *store.Workspace) DispatchReport {
	return s.analyseWorkspaceNamed(workspace, qaAnalysisMeasurementName(qaAnalysisWorkspaceName(workspace)))
}

func (s *PrepSubsystem) analyseWorkspaceNamed(workspace *store.Workspace, workspaceName string) DispatchReport {
	report := DispatchReport{
		Workspace:   core.Trim(workspaceName),
		Summary:     map[string]any{},
		GeneratedAt: time.Now().UTC(),
	}
	if report.Workspace == "" {
		report.Workspace = qaAnalysisMeasurementName(qaAnalysisWorkspaceName(workspace))
	}
	if workspace == nil {
		report.SummaryText = qaAnalysisSummaryText(report)
		return report
	}

	report.Findings = qaAnalysisWorkspaceFindings(workspace)
	report.Tools = qaAnalysisWorkspaceToolRuns(workspace)
	report.Clusters = qaAnalysisSafeClusters(report.Findings)

	previousCycles := readPreviousJournalCycles(s.stateStoreInstance(), report.Workspace, persistentThreshold)
	report.New, report.Resolved = diffFindings(report.Findings, previousCycles)
	report.Persistent = persistentFindings(report.Findings, previousCycles)
	report.Summary = qaAnalysisSummary(workspace.Aggregate(), report)
	report.SummaryText = qaAnalysisSummaryText(report)
	return report
}

func qaAnalysisWorkspaceName(workspace *store.Workspace) string {
	if workspace == nil {
		return ""
	}
	return workspace.Name()
}

func qaAnalysisMeasurementName(name string) string {
	trimmed := core.Trim(name)
	if core.HasPrefix(trimmed, "qa-") {
		return core.TrimPrefix(trimmed, "qa-")
	}
	return trimmed
}

func qaAnalysisWorkspaceFindings(workspace *store.Workspace) []QAFinding {
	rows := qaAnalysisWorkspaceRows(workspace, "finding")
	findings := make([]QAFinding, 0, len(rows))
	for _, row := range rows {
		var finding QAFinding
		if parseResult := core.JSONUnmarshalString(row, &finding); parseResult.OK {
			findings = append(findings, finding)
		}
	}
	return findings
}

func qaAnalysisWorkspaceToolRuns(workspace *store.Workspace) []QAToolRun {
	rows := qaAnalysisWorkspaceRows(workspace, "tool_run")
	toolRuns := make([]QAToolRun, 0, len(rows))
	for _, row := range rows {
		var toolRun QAToolRun
		if parseResult := core.JSONUnmarshalString(row, &toolRun); parseResult.OK {
			toolRuns = append(toolRuns, toolRun)
		}
	}
	return toolRuns
}

func qaAnalysisWorkspaceRows(workspace *store.Workspace, kind string) []string {
	if workspace == nil || kind == "" {
		return nil
	}

	result := workspace.Query(
		core.Sprintf(
			"SELECT data FROM entries WHERE kind = '%s' ORDER BY id",
			escapeJournalLiteral(kind),
		),
	)
	if !result.OK || result.Value == nil {
		return nil
	}

	rows, ok := result.Value.([]map[string]any)
	if !ok {
		return nil
	}

	values := make([]string, 0, len(rows))
	for _, row := range rows {
		if payload := stringValue(row["data"]); payload != "" {
			values = append(values, payload)
		}
	}
	return values
}

// findingToPoint projects a finding into the RFC §7 clustering dimensions.
// Frequency defaults to 1 for direct callers; the cluster builder supplies the
// observed per-fingerprint frequency for each point.
func qaAnalysisPointCoords(finding QAFinding, frequency float64) []float64 {
	return []float64{
		qaAnalysisHash(core.Lower(finding.Tool)),
		qaAnalysisSeverityScore(finding.Severity),
		qaAnalysisHash(core.Lower(finding.File)),
		qaAnalysisHash(core.Lower(firstNonEmpty(finding.Category, finding.Code, finding.RuleID))),
		frequency,
	}
}

func diffFindings(current []QAFinding, previous [][]map[string]any) (newList, resolvedList []map[string]any) {
	if len(previous) == 0 {
		return nil, nil
	}

	currentByKey := make(map[string]QAFinding, len(current))
	for _, finding := range current {
		currentByKey[findingFingerprint(finding)] = finding
	}

	lastCycle := previous[len(previous)-1]
	lastCycleByKey := make(map[string]map[string]any, len(lastCycle))
	for _, entry := range lastCycle {
		lastCycleByKey[findingFingerprintFromMap(entry)] = entry
	}

	for key, finding := range currentByKey {
		if _, ok := lastCycleByKey[key]; !ok {
			newList = append(newList, findingToMap(finding))
		}
	}

	for key, entry := range lastCycleByKey {
		if _, ok := currentByKey[key]; !ok {
			resolvedList = append(resolvedList, entry)
		}
	}

	return newList, resolvedList
}

func persistentFindings(current []QAFinding, previous [][]map[string]any) []map[string]any {
	if len(previous) < persistentThreshold-1 || len(current) == 0 {
		return nil
	}

	currentByKey := make(map[string]QAFinding, len(current))
	for _, finding := range current {
		currentByKey[findingFingerprint(finding)] = finding
	}

	window := previous
	if len(window) > persistentThreshold-1 {
		window = window[len(window)-(persistentThreshold-1):]
	}

	counts := make(map[string]int, len(currentByKey))
	for _, cycle := range window {
		seen := make(map[string]bool, len(cycle))
		for _, entry := range cycle {
			key := findingFingerprintFromMap(entry)
			if seen[key] {
				continue
			}
			seen[key] = true
			counts[key]++
		}
	}

	persistentList := make([]map[string]any, 0, len(currentByKey))
	for key, finding := range currentByKey {
		if counts[key] == len(window) {
			persistentList = append(persistentList, findingToMap(finding))
		}
	}
	if len(persistentList) == 0 {
		return nil
	}
	return persistentList
}

func qaAnalysisSafeClusters(findings []QAFinding) (clusters []DispatchCluster) {
	if len(findings) == 0 {
		return nil
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			core.Warn("agentic: Poindexter workspace analysis panicked", "reason", recovered)
			clusters = clusterFindingsFallback(findings)
		}
	}()

	clusters = qaAnalysisClusterer(findings)
	if len(clusters) > 0 {
		return clusters
	}
	return clusterFindingsFallback(findings)
}

func qaAnalysisClusters(findings []QAFinding) []DispatchCluster {
	if len(findings) == 0 {
		return nil
	}

	frequencies := qaAnalysisFrequencies(findings)
	items := make([]qaAnalysisPoint, len(findings))
	for index, finding := range findings {
		coords := qaAnalysisPointCoords(finding, frequencies[findingFingerprint(finding)])
		items[index] = qaAnalysisPoint{
			Index:     index,
			ToolID:    coords[0],
			Severity:  coords[1],
			FileHash:  coords[2],
			Category:  coords[3],
			Frequency: coords[4],
		}
	}

	points, err := poindexter.BuildND(items,
		func(item qaAnalysisPoint) string { return core.Sprintf("finding-%d", item.Index) },
		[]func(qaAnalysisPoint) float64{
			func(item qaAnalysisPoint) float64 { return item.ToolID },
			func(item qaAnalysisPoint) float64 { return item.Severity },
			func(item qaAnalysisPoint) float64 { return item.FileHash },
			func(item qaAnalysisPoint) float64 { return item.Category },
			func(item qaAnalysisPoint) float64 { return item.Frequency },
		},
		[]float64{1, 1, 1, 1, 1},
		[]bool{false, false, false, false, false},
	)
	if err != nil || len(points) == 0 {
		return clusterFindingsFallback(findings)
	}

	union := qaAnalysisClusterByDistance(points, findings, qaAnalysisClusterDistance)
	if union == nil {
		return clusterFindingsFallback(findings)
	}
	return qaClusterDispatchClusters(findings, union)
}

func qaAnalysisClusterByDistance(points []poindexter.KDPoint[qaAnalysisPoint], findings []QAFinding, distance float64) *qaClusterUnion {
	tree, err := poindexter.NewKDTree(points, poindexter.WithMetric(poindexter.EuclideanDistance{}))
	if err != nil {
		return nil
	}

	union := newQAClusterUnion(len(points))
	for _, point := range points {
		neighbours, _ := tree.Radius(point.Coords, distance)
		for _, neighbour := range neighbours {
			leftIndex := point.Value.Index
			rightIndex := neighbour.Value.Index
			if leftIndex == rightIndex {
				continue
			}
			if !qaAnalysisCompatible(findings[leftIndex], findings[rightIndex]) {
				continue
			}
			union.Union(leftIndex, rightIndex)
		}
	}
	return union
}

func qaAnalysisCompatible(left, right QAFinding) bool {
	if !qaClusterCompatible(left, right) {
		return false
	}

	leftCategory := firstNonEmpty(left.Category, left.Code, left.RuleID)
	rightCategory := firstNonEmpty(right.Category, right.Code, right.RuleID)
	if leftCategory != "" && rightCategory != "" && leftCategory != rightCategory {
		return false
	}
	return true
}

func qaAnalysisFrequencies(findings []QAFinding) map[string]float64 {
	frequencies := make(map[string]float64, len(findings))
	for _, finding := range findings {
		frequencies[findingFingerprint(finding)]++
	}
	return frequencies
}

func qaAnalysisSummary(base map[string]any, report DispatchReport) map[string]any {
	summary := maps.Clone(base)
	if summary == nil {
		summary = map[string]any{}
	}
	summary["clusters"] = len(report.Clusters)
	summary["new"] = len(report.New)
	summary["resolved"] = len(report.Resolved)
	summary["persistent"] = len(report.Persistent)
	return summary
}

func qaAnalysisSummaryText(report DispatchReport) string {
	return core.Sprintf(
		"%d findings across %d clusters; %d new, %d resolved, %d persistent",
		len(report.Findings),
		len(report.Clusters),
		len(report.New),
		len(report.Resolved),
		len(report.Persistent),
	)
}

func qaAnalysisSeverityScore(severity string) float64 {
	switch core.Lower(core.Trim(severity)) {
	case "critical", "error", "high":
		return 1
	case "warning", "warn", "medium":
		return 0.6
	case "info", "low":
		return 0.3
	default:
		return 0.1
	}
}

func qaAnalysisHash(value string) float64 {
	if core.Trim(value) == "" {
		return 0
	}
	hash := fnv.New32a()
	if _, err := hash.Write([]byte(value)); err != nil {
		return 0
	}
	return float64(hash.Sum32())
}
