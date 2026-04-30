// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"
	"time"

	core "dappco.re/go"
)

func TestPipelineBudget_Good_RootHelp(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	output := captureStdout(t, func() {
		result := s.cmdPipelineBudget(core.NewOptions())
		core.RequireTrue(t, result.OK)
	})

	core.AssertContains(t, output, "core-agent pipeline/budget/plan")
	core.AssertContains(t, output, "core-agent pipeline/budget/log")
}

func TestPipelineBudget_Good_PlanShowsRemainingBudgetByPool(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	core.RequireTrue(t, fs.Write(core.JoinPath(CoreRoot(), "agents.yaml"), core.Concat(
		"version: 1\n",
		"concurrency:\n",
		"  codex: 1\n",
		"rates:\n",
		"  codex:\n",
		"    reset_utc: \"00:00\"\n",
		"    daily_limit: 3\n",
		"    sustained_delay: 30\n",
	)).OK)

	result := s.cmdPipelineBudgetLog(core.NewOptions(
		core.Option{Key: "repo", Value: "go-io"},
		core.Option{Key: "agent", Value: "codex"},
		core.Option{Key: "model", Value: "gpt-5.4"},
	))
	core.RequireTrue(t, result.OK)

	var rows []pipelineBudgetPlanRow
	output := captureStdout(t, func() {
		plan := s.cmdPipelineBudgetPlan(core.NewOptions())
		core.RequireTrue(t, plan.OK)
		var ok bool
		rows, ok = plan.Value.([]pipelineBudgetPlanRow)
		core.RequireTrue(t, ok)
	})

	core.AssertLen(t, rows, 1)
	core.AssertContains(t, output, "codex")
	core.AssertEqual(t, 1, rows[0].UsedToday)
	core.AssertEqual(t, "3", rows[0].DailyLimit)
	core.AssertEqual(t, "2", rows[0].Remaining)
}

func TestPipelineBudget_Good_LogAppendsJournal(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	first := s.cmdPipelineBudgetLog(core.NewOptions(
		core.Option{Key: "repo", Value: "go-io"},
		core.Option{Key: "agent", Value: "codex"},
		core.Option{Key: "model", Value: "gpt-5.4"},
	))
	second := s.cmdPipelineBudgetLog(core.NewOptions(
		core.Option{Key: "repo", Value: "go-log"},
		core.Option{Key: "agent", Value: "claude"},
		core.Option{Key: "status", Value: "queued"},
	))

	core.RequireTrue(t, first.OK)
	core.RequireTrue(t, second.OK)

	lines := readJSONLLines(pipelineBudgetJournalPath())
	core.AssertLen(t, lines, 2)
	core.AssertContains(t, lines[0], `"repo":"go-io"`)
	core.AssertContains(t, lines[1], `"repo":"go-log"`)
}

func TestPipelineBudget_Bad_LogRequiresRepoAndAgent(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	result := s.cmdPipelineBudgetLog(core.NewOptions(
		core.Option{Key: "repo", Value: "go-io"},
	))

	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "repo and agent are required")
}

func TestPipelineBudget_Ugly_PlanSkipsCorruptJournalRows(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	core.RequireTrue(t, fs.Write(core.JoinPath(CoreRoot(), "agents.yaml"), core.Concat(
		"version: 1\n",
		"rates:\n",
		"  codex:\n",
		"    reset_utc: \"00:00\"\n",
		"    daily_limit: 2\n",
		"    sustained_delay: 30\n",
	)).OK)

	valid := pipelineBudgetEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Repo:      "go-io",
		Agent:     "codex",
		Model:     "gpt-5.4",
		Status:    "started",
	}
	core.RequireNoError(t, ensureParentDir(pipelineBudgetJournalPath()))
	core.RequireTrue(t, fs.WriteAtomic(pipelineBudgetJournalPath(), core.Concat(
		"{not-json}\n",
		core.JSONMarshalString(valid),
		"\n",
	)).OK)

	rows := s.pipelineBudgetPlanRows(time.Now().UTC())
	core.AssertLen(t, rows, 1)
	core.AssertEqual(t, 1, rows[0].UsedToday)
}
