// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipelineBudget_Good_RootHelp(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	output := captureStdout(t, func() {
		result := s.cmdPipelineBudget(core.NewOptions())
		require.True(t, result.OK)
	})

	assert.Contains(t, output, "core-agent pipeline/budget/plan")
	assert.Contains(t, output, "core-agent pipeline/budget/log")
}

func TestPipelineBudget_Good_PlanShowsRemainingBudgetByPool(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	require.True(t, fs.Write(core.JoinPath(CoreRoot(), "agents.yaml"), core.Concat(
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
	require.True(t, result.OK)

	var rows []pipelineBudgetPlanRow
	output := captureStdout(t, func() {
		plan := s.cmdPipelineBudgetPlan(core.NewOptions())
		require.True(t, plan.OK)
		var ok bool
		rows, ok = plan.Value.([]pipelineBudgetPlanRow)
		require.True(t, ok)
	})

	require.Len(t, rows, 1)
	assert.Contains(t, output, "codex")
	assert.Equal(t, 1, rows[0].UsedToday)
	assert.Equal(t, "3", rows[0].DailyLimit)
	assert.Equal(t, "2", rows[0].Remaining)
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

	require.True(t, first.OK)
	require.True(t, second.OK)

	lines := readJSONLLines(pipelineBudgetJournalPath())
	require.Len(t, lines, 2)
	assert.Contains(t, lines[0], `"repo":"go-io"`)
	assert.Contains(t, lines[1], `"repo":"go-log"`)
}

func TestPipelineBudget_Bad_LogRequiresRepoAndAgent(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	result := s.cmdPipelineBudgetLog(core.NewOptions(
		core.Option{Key: "repo", Value: "go-io"},
	))

	require.False(t, result.OK)
	err, ok := result.Value.(error)
	require.True(t, ok)
	assert.Contains(t, err.Error(), "repo and agent are required")
}

func TestPipelineBudget_Ugly_PlanSkipsCorruptJournalRows(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	require.True(t, fs.Write(core.JoinPath(CoreRoot(), "agents.yaml"), core.Concat(
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
	require.NoError(t, ensureParentDir(pipelineBudgetJournalPath()))
	require.True(t, fs.WriteAtomic(pipelineBudgetJournalPath(), core.Concat(
		"{not-json}\n",
		core.JSONMarshalString(valid),
		"\n",
	)).OK)

	rows := s.pipelineBudgetPlanRows(time.Now().UTC())
	require.Len(t, rows, 1)
	assert.Equal(t, 1, rows[0].UsedToday)
}
