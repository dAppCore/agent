// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

func TestTrainingJournal_PipelineTrainingEntry_Good(t *testing.T) {
	entry := PipelineTrainingEntry{Repo: "go-io", PRNumber: 42, CodeRabbitFindings: 0}
	text := core.JSONMarshalString(entry)

	core.AssertContains(t, text, `"repo":"go-io"`)
	core.AssertContains(t, text, `"pr_number":42`)
}

func TestTrainingJournal_summarisePipelineTraining_Bad(t *testing.T) {
	stats := summarisePipelineTraining(nil)

	core.AssertEqual(t, 0, stats.TotalPRs)
	core.AssertNil(t, stats.ByRepo)
	core.AssertNil(t, stats.ByRepoZeroClean)
}

func TestTrainingJournal_filterZeroFindingTrainingEntries_Ugly(t *testing.T) {
	entries := []PipelineTrainingEntry{
		{Repo: "go-io", CodeRabbitFindings: 1},
		{Repo: "go-log", CodeRabbitFindings: 0},
	}

	filtered := filterZeroFindingTrainingEntries(entries)
	core.AssertLen(t, filtered, 1)
	core.AssertEqual(t, "go-log", filtered[0].Repo)
}
