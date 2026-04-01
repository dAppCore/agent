// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"sort"
	"time"

	core "dappco.re/go/core"
)

const planRetentionDefaultDays = 90
const planRetentionScheduleInterval = 24 * time.Hour

type PlanCleanupOutput struct {
	Success  bool   `json:"success"`
	Disabled bool   `json:"disabled,omitempty"`
	DryRun   bool   `json:"dry_run,omitempty"`
	Deleted  int    `json:"deleted,omitempty"`
	Matched  int    `json:"matched,omitempty"`
	Cutoff   string `json:"cutoff,omitempty"`
}

// result := c.Command("plan-cleanup").Run(ctx, core.NewOptions(core.Option{Key: "dry-run", Value: true}))
func (s *PrepSubsystem) cmdPlanCleanup(options core.Options) core.Result {
	result := s.planCleanup(options)
	if !result.OK {
		if err, ok := result.Value.(error); ok {
			core.Print(nil, "error: %v", err)
		}
		return result
	}

	output, ok := result.Value.(PlanCleanupOutput)
	if !ok {
		err := core.E("planCleanup", "invalid plan cleanup output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	if output.Disabled {
		return core.Result{Value: output, OK: true}
	}

	if output.Matched == 0 {
		core.Print(nil, "No archived plans found past the retention period.")
		return core.Result{Value: output, OK: true}
	}

	if output.DryRun {
		core.Print(nil, "DRY RUN: %d archived plan(s) would be permanently deleted (archived before %s).", output.Matched, output.Cutoff)
		return core.Result{Value: output, OK: true}
	}

	core.Print(nil, "Permanently deleted %d archived plan(s) archived before %s.", output.Deleted, output.Cutoff)
	return core.Result{Value: output, OK: true}
}

// ctx, cancel := context.WithCancel(context.Background())
// go s.runPlanCleanupLoop(ctx, time.Minute)
func (s *PrepSubsystem) runPlanCleanupLoop(ctx context.Context, interval time.Duration) {
	if ctx == nil || interval <= 0 {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.planCleanup(core.NewOptions())
		}
	}
}

func (s *PrepSubsystem) planCleanup(options core.Options) core.Result {
	days := planRetentionDays(options)
	if days <= 0 {
		core.Print(nil, "Retention cleanup is disabled (plan_retention_days is 0).")
		return core.Result{Value: PlanCleanupOutput{Success: true, Disabled: true}, OK: true}
	}

	cutoff := time.Now().AddDate(0, 0, -days)
	candidates := planRetentionCandidates(PlansRoot(), cutoff)
	output := PlanCleanupOutput{
		Success: true,
		DryRun:  optionBoolValue(options, "dry_run", "dry-run"),
		Matched: len(candidates),
		Cutoff:  cutoff.Format("2006-01-02"),
	}

	if len(candidates) == 0 {
		return core.Result{Value: output, OK: true}
	}

	if output.DryRun {
		return core.Result{Value: output, OK: true}
	}

	deleted := 0
	for _, candidate := range candidates {
		if r := fs.Delete(candidate.path); !r.OK {
			err, _ := r.Value.(error)
			if err == nil {
				err = core.E("planCleanup", core.Concat("failed to delete plan: ", candidate.path), nil)
			}
			return core.Result{Value: err, OK: false}
		}
		deleted++
	}

	output.Deleted = deleted
	return core.Result{Value: output, OK: true}
}

type planRetentionCandidate struct {
	path       string
	archivedAt time.Time
}

func planRetentionCandidates(dir string, cutoff time.Time) []planRetentionCandidate {
	jsonFiles := core.PathGlob(core.JoinPath(dir, "*.json"))
	sort.Strings(jsonFiles)

	var candidates []planRetentionCandidate
	for _, path := range jsonFiles {
		id := core.TrimSuffix(core.PathBase(path), ".json")
		planResult := readPlanResult(dir, id)
		if !planResult.OK {
			continue
		}

		plan, ok := planResult.Value.(*Plan)
		if !ok || plan == nil {
			continue
		}
		if plan.Status != "archived" {
			continue
		}

		archivedAt := planArchivedAt(path, plan)
		if archivedAt.IsZero() || !archivedAt.Before(cutoff) {
			continue
		}

		candidates = append(candidates, planRetentionCandidate{
			path:       path,
			archivedAt: archivedAt,
		})
	}

	return candidates
}

func planArchivedAt(path string, plan *Plan) time.Time {
	if plan != nil && !plan.ArchivedAt.IsZero() {
		return plan.ArchivedAt
	}

	if stat := fs.Stat(path); stat.OK {
		if info, ok := stat.Value.(interface{ ModTime() time.Time }); ok {
			return info.ModTime()
		}
	}

	return time.Time{}
}

func planRetentionDays(options core.Options) int {
	if result := options.Get("days"); result.OK {
		switch value := result.Value.(type) {
		case int:
			return value
		case int64:
			return int(value)
		case float64:
			return int(value)
		case string:
			trimmed := core.Trim(value)
			if trimmed != "" {
				return parseInt(trimmed)
			}
		}
	}

	if value := core.Env("AGENTIC_PLAN_RETENTION_DAYS"); value != "" {
		return parseInt(value)
	}

	return planRetentionDefaultDays
}
