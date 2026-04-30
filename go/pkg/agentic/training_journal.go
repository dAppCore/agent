// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"sort"

	core "dappco.re/go"
)

// entry := PipelineTrainingEntry{Repo: "go-io", PRNumber: 42, CodeRabbitFindings: 0}
type PipelineTrainingEntry struct {
	CapturedAt            string `json:"captured_at"`
	Org                   string `json:"org,omitempty"`
	Repo                  string `json:"repo"`
	PRNumber              int    `json:"pr_number"`
	PRURL                 string `json:"pr_url,omitempty"`
	State                 string `json:"state,omitempty"`
	Mergeable             string `json:"mergeable,omitempty"`
	BaseBranch            string `json:"base_branch,omitempty"`
	HeadBranch            string `json:"head_branch,omitempty"`
	HeadSHA               string `json:"head_sha,omitempty"`
	ChecksTotal           int    `json:"checks_total,omitempty"`
	ChecksPassing         int    `json:"checks_passing,omitempty"`
	ChecksFailing         int    `json:"checks_failing,omitempty"`
	ReviewThreadsTotal    int    `json:"review_threads_total,omitempty"`
	ReviewThreadsResolved int    `json:"review_threads_resolved,omitempty"`
	CodeRabbitFindings    int    `json:"coderabbit_findings"`
	FindingSource         string `json:"finding_source,omitempty"`
	Diff                  string `json:"diff,omitempty"`
	DiffSource            string `json:"diff_source,omitempty"`
}

// stats := PipelineTrainingStats{TotalPRs: 5, ZeroFindingPRs: 2, ByRepo: map[string]int{"go-io": 3}}
type PipelineTrainingStats struct {
	TotalPRs        int            `json:"total_prs"`
	ZeroFindingPRs  int            `json:"zero_finding_prs"`
	ByRepo          map[string]int `json:"by_repo,omitempty"`
	ByRepoZeroClean map[string]int `json:"by_repo_zero_finding,omitempty"`
}

// path := pipelineTrainingJournalPath()
func pipelineTrainingJournalPath() string {
	return core.JoinPath(CoreRoot(), "training", "journal.jsonl")
}

// path := pipelineTrainingExportPath()
func pipelineTrainingExportPath() string {
	return core.JoinPath(CoreRoot(), "training", "export.jsonl")
}

// result := ensureParentDir("/tmp/.core/training/journal.jsonl")
var ensureParentDir = func(path string) error {
	if ensureResult := fs.EnsureDir(core.PathDir(path)); !ensureResult.OK {
		if err, ok := ensureResult.Value.(error); ok {
			return core.E("agentic.ensureParentDir", "prepare journal directory", err)
		}
		return core.E("agentic.ensureParentDir", "prepare journal directory", nil)
	}
	return nil
}

// result := appendJSONLRecord("/tmp/test.jsonl", map[string]any{"repo": "go-io"})
var appendJSONLRecord = func(path string, value any) error {
	if err := ensureParentDir(path); err != nil {
		return err
	}
	handle := fs.Append(path)
	if !handle.OK {
		if err, ok := handle.Value.(error); ok {
			return core.E("agentic.appendJSONLRecord", "open journal", err)
		}
		return core.E("agentic.appendJSONLRecord", "open journal", nil)
	}
	if writeResult := core.WriteAll(handle.Value, core.Concat(core.JSONMarshalString(value), "\n")); !writeResult.OK {
		if err, ok := writeResult.Value.(error); ok {
			return core.E("agentic.appendJSONLRecord", "append journal entry", err)
		}
		return core.E("agentic.appendJSONLRecord", "append journal entry", nil)
	}
	return nil
}

// lines := readJSONLLines("/tmp/test.jsonl")
func readJSONLLines(path string) []string {
	readResult := fs.Read(path)
	if !readResult.OK {
		return nil
	}
	lines := []string{}
	for _, line := range core.Split(readResult.Value.(string), "\n") {
		trimmed := core.Trim(line)
		if trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	return lines
}

// entries := readPipelineTrainingJournal("/tmp/.core/training/journal.jsonl")
func readPipelineTrainingJournal(path string) []PipelineTrainingEntry {
	lines := readJSONLLines(path)
	if len(lines) == 0 {
		return nil
	}

	entries := make([]PipelineTrainingEntry, 0, len(lines))
	for _, line := range lines {
		var entry PipelineTrainingEntry
		if parseResult := core.JSONUnmarshalString(line, &entry); !parseResult.OK {
			continue
		}
		if entry.Repo == "" || entry.PRNumber <= 0 {
			continue
		}
		entries = append(entries, entry)
	}
	return entries
}

// stats := summarisePipelineTraining(entries)
func summarisePipelineTraining(entries []PipelineTrainingEntry) PipelineTrainingStats {
	stats := PipelineTrainingStats{
		ByRepo:          map[string]int{},
		ByRepoZeroClean: map[string]int{},
	}
	for _, entry := range entries {
		stats.TotalPRs++
		stats.ByRepo[entry.Repo]++
		if entry.CodeRabbitFindings == 0 {
			stats.ZeroFindingPRs++
			stats.ByRepoZeroClean[entry.Repo]++
		}
	}
	if len(stats.ByRepo) == 0 {
		stats.ByRepo = nil
	}
	if len(stats.ByRepoZeroClean) == 0 {
		stats.ByRepoZeroClean = nil
	}
	return stats
}

// clean := filterZeroFindingTrainingEntries(entries)
func filterZeroFindingTrainingEntries(entries []PipelineTrainingEntry) []PipelineTrainingEntry {
	if len(entries) == 0 {
		return nil
	}
	clean := []PipelineTrainingEntry{}
	for _, entry := range entries {
		if entry.CodeRabbitFindings == 0 {
			clean = append(clean, entry)
		}
	}
	return clean
}

// result := writePipelineTrainingExport("/tmp/.core/training/export.jsonl", entries)
var writePipelineTrainingExport = func(path string, entries []PipelineTrainingEntry) error {
	if err := ensureParentDir(path); err != nil {
		return err
	}
	builder := core.NewBuilder()
	for _, entry := range entries {
		builder.WriteString(core.JSONMarshalString(entry))
		builder.WriteString("\n")
	}
	if writeResult := fs.WriteAtomic(path, builder.String()); !writeResult.OK {
		if err, ok := writeResult.Value.(error); ok {
			return core.E("agentic.writePipelineTrainingExport", "write export", err)
		}
		return core.E("agentic.writePipelineTrainingExport", "write export", nil)
	}
	return nil
}

// names := sortedTrainingRepos(map[string]int{"go-log": 1, "go-io": 2})
func sortedTrainingRepos(values map[string]int) []string {
	if len(values) == 0 {
		return nil
	}
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
