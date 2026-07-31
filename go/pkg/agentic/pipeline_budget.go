// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"maps"
	"slices"
	"time"

	core "dappco.re/go"
)

const pipelineBudgetStoreGroup = "pipeline_budget_dispatch"

// entry := pipelineBudgetEntry{Repo: "go-io", Agent: "codex", Model: "gpt-5.4", Status: "started"}
type pipelineBudgetEntry struct {
	Timestamp string `json:"timestamp"`
	Repo      string `json:"repo"`
	Agent     string `json:"agent"`
	Model     string `json:"model,omitempty"`
	Ticket    string `json:"ticket,omitempty"`
	Status    string `json:"status,omitempty"`
}

// row := pipelineBudgetPlanRow{Pool: "codex", UsedToday: 1, Remaining: "2"}
type pipelineBudgetPlanRow struct {
	Pool        string
	UsedToday   int
	DailyLimit  string
	Remaining   string
	Running     int
	Concurrency string
	ResetUTC    string
}

// path := pipelineBudgetJournalPath()
func pipelineBudgetJournalPath() string {
	return core.JoinPath(CoreRoot(), "journal", "dispatch.jsonl")
}

func (s *PrepSubsystem) cmdPipelineBudgetPlan(_ core.Options) core.Result {
	rows := s.pipelineBudgetPlanRows(time.Now().UTC())
	if len(rows) == 0 {
		core.Print(nil, "no configured dispatch pools")
		return core.Result{OK: true}
	}

	core.Print(nil, "  %-12s %-6s %-12s %-12s %-8s %-11s %s", "POOL", "USED", "DAILY_LIMIT", "REMAINING", "RUNNING", "CONCURRENCY", "RESET")
	for _, row := range rows {
		core.Print(nil, "  %-12s %-6d %-12s %-12s %-8d %-11s %s",
			row.Pool,
			row.UsedToday,
			row.DailyLimit,
			row.Remaining,
			row.Running,
			row.Concurrency,
			row.ResetUTC,
		)
	}
	if s.stateStoreInstance() == nil && stateStoreErr(s) != nil {
		core.Print(nil, "  note: using %s fallback because .core/db.duckdb is unavailable", pipelineBudgetJournalPath())
	}
	return core.Result{Value: rows, OK: true}
}

func (s *PrepSubsystem) cmdPipelineBudgetLog(options core.Options) core.Result {
	repo := pipelineRepoValue(options)
	agent := optionStringValue(options, "agent")
	if repo == "" || agent == "" {
		core.Print(nil, "usage: core-agent pipeline/budget/log --repo=<repo> --agent=<agent> [--model=<model>] [--ticket=<id>] [--status=started]")
		return core.Result{Value: core.E("agentic.cmdPipelineBudgetLog", "repo and agent are required", nil), OK: false}
	}

	model := optionStringValue(options, "model")
	if model == "" {
		model = modelVariant(agent)
	}
	entry := pipelineBudgetEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Repo:      repo,
		Agent:     baseAgent(agent),
		Model:     model,
		Ticket:    optionStringValue(options, "ticket"),
		Status:    optionStringValue(options, "status"),
	}
	if entry.Status == "" {
		entry.Status = "started"
	}

	if err := appendJSONLRecord(pipelineBudgetJournalPath(), entry); err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}
	s.pipelineBudgetMirrorToStore(entry)

	core.Print(nil, "timestamp: %s", entry.Timestamp)
	core.Print(nil, "repo:      %s", entry.Repo)
	core.Print(nil, "agent:     %s", entry.Agent)
	core.Print(nil, "model:     %s", firstNonEmpty(entry.Model, "-"))
	core.Print(nil, "ticket:    %s", firstNonEmpty(entry.Ticket, "-"))
	core.Print(nil, "status:    %s", entry.Status)
	core.Print(nil, "journal:   %s", pipelineBudgetJournalPath())
	return core.Result{Value: entry, OK: true}
}

// rows := s.pipelineBudgetPlanRows(time.Now().UTC())
func (s *PrepSubsystem) pipelineBudgetPlanRows(now time.Time) []pipelineBudgetPlanRow {
	config := s.loadAgentsConfig()
	pools := map[string]bool{}
	for pool := range config.Concurrency {
		pools[pool] = true
	}
	for pool := range config.Rates {
		pools[pool] = true
	}
	if len(pools) == 0 {
		return nil
	}

	names := slices.Sorted(maps.Keys(pools))

	entries := s.pipelineBudgetEntries()
	rows := make([]pipelineBudgetPlanRow, 0, len(names))
	for _, pool := range names {
		rate := config.Rates[pool]
		windowStart, _ := pipelineBudgetWindow(rate.ResetUTC, now)
		used := 0
		for _, entry := range entries {
			if baseAgent(entry.Agent) != pool {
				continue
			}
			timestamp, err := time.Parse(time.RFC3339Nano, entry.Timestamp)
			if err != nil {
				continue
			}
			if timestamp.Before(windowStart) {
				continue
			}
			used++
		}

		row := pipelineBudgetPlanRow{
			Pool:        pool,
			UsedToday:   used,
			DailyLimit:  "unlimited",
			Remaining:   "unlimited",
			Running:     s.countRunningByAgent(pool),
			Concurrency: pipelineBudgetConcurrencyString(config.Concurrency[pool]),
			ResetUTC:    firstNonEmpty(rate.ResetUTC, "00:00"),
		}
		if rate.DailyLimit > 0 {
			row.DailyLimit = core.Sprint(rate.DailyLimit)
			remaining := max(rate.DailyLimit-used, 0)
			row.Remaining = core.Sprint(remaining)
		}
		rows = append(rows, row)
	}
	return rows
}

// entries := s.pipelineBudgetEntries()
func (s *PrepSubsystem) pipelineBudgetEntries() []pipelineBudgetEntry {
	entries := []pipelineBudgetEntry{}
	seen := map[string]bool{}

	s.stateStoreRestore(pipelineBudgetStoreGroup, func(_ string, value string) bool {
		var entry pipelineBudgetEntry
		if parseResult := core.JSONUnmarshalString(value, &entry); !parseResult.OK {
			return true
		}
		if entry.Repo == "" || entry.Agent == "" || entry.Timestamp == "" {
			return true
		}
		key := pipelineBudgetEntryKey(entry)
		if !seen[key] {
			seen[key] = true
			entries = append(entries, entry)
		}
		return true
	})

	for _, line := range readJSONLLines(pipelineBudgetJournalPath()) {
		var entry pipelineBudgetEntry
		if parseResult := core.JSONUnmarshalString(line, &entry); !parseResult.OK {
			continue
		}
		if entry.Repo == "" || entry.Agent == "" || entry.Timestamp == "" {
			continue
		}
		key := pipelineBudgetEntryKey(entry)
		if !seen[key] {
			seen[key] = true
			entries = append(entries, entry)
		}
	}

	return entries
}

// result := s.pipelineBudgetMirrorToStore(entry)
func (s *PrepSubsystem) pipelineBudgetMirrorToStore(entry pipelineBudgetEntry) {
	key := pipelineBudgetEntryKey(entry)
	s.stateStoreSet(pipelineBudgetStoreGroup, key, entry)
}

// key := pipelineBudgetEntryKey(entry)
func pipelineBudgetEntryKey(entry pipelineBudgetEntry) string {
	return core.Sprintf("%s|%s|%s|%s|%s|%s", entry.Timestamp, entry.Repo, entry.Agent, entry.Model, entry.Ticket, entry.Status)
}

// start, end := pipelineBudgetWindow("06:00", time.Now().UTC())
func pipelineBudgetWindow(resetUTC string, now time.Time) (time.Time, time.Time) {
	resetHour, resetMinute := 0, 0
	parts := core.Split(resetUTC, ":")
	if len(parts) >= 2 {
		resetHour = parseIntString(parts[0])
		resetMinute = parseIntString(parts[1])
	}
	windowStart := time.Date(now.Year(), now.Month(), now.Day(), resetHour, resetMinute, 0, 0, time.UTC)
	if now.Before(windowStart) {
		windowStart = windowStart.AddDate(0, 0, -1)
	}
	return windowStart, windowStart.AddDate(0, 0, 1)
}

// text := pipelineBudgetConcurrencyString(ConcurrencyLimit{Total: 2})
func pipelineBudgetConcurrencyString(limit ConcurrencyLimit) string {
	if limit.Total <= 0 {
		return "unlimited"
	}
	return core.Sprint(limit.Total)
}
