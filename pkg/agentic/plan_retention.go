// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"slices"
	"time"

	core "dappco.re/go"
)

const planRetentionDefaultDays = 90
const planRetentionScheduleInterval = 24 * time.Hour

type PlanCleanupOutput struct {
	Success  bool   `json:"success"`
	Disabled bool   `json:"disabled,omitempty"`
	DryRun   bool   `json:"dry_run,omitempty"`
	Archived int    `json:"archived,omitempty"`
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
		core.Print(nil, "No plans found past the retention period.")
		return core.Result{Value: output, OK: true}
	}

	if output.DryRun {
		core.Print(nil, "DRY RUN: %d plan(s) would be archived or deleted (cutoff %s).", output.Matched, output.Cutoff)
		return core.Result{Value: output, OK: true}
	}

	if output.Archived > 0 && output.Deleted > 0 {
		core.Print(nil, "Archived %d plan(s) and permanently deleted %d stale archive(s) before %s.", output.Archived, output.Deleted, output.Cutoff)
		return core.Result{Value: output, OK: true}
	}
	if output.Archived > 0 {
		core.Print(nil, "Archived %d plan(s) before %s.", output.Archived, output.Cutoff)
		return core.Result{Value: output, OK: true}
	}

	core.Print(nil, "Permanently deleted %d stale archive(s) before %s.", output.Deleted, output.Cutoff)
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

	archived := 0
	if output.DryRun {
		for _, candidate := range candidates {
			if planRetentionShouldArchive(candidate.plan.Status) {
				archived++
			}
		}
		output.Archived = archived
		return core.Result{Value: output, OK: true}
	}

	deleted := 0
	for _, candidate := range candidates {
		if planRetentionShouldArchive(candidate.plan.Status) {
			_, archiveErr := archivePlanResult(PlanDeleteInput{
				ID:     candidate.plan.ID,
				Reason: "plan retention cleanup",
			}, "id is required", "planCleanup")
			if archiveErr != nil {
				return core.Result{Value: archiveErr, OK: false}
			}
			archived++
			continue
		}

		if r := fs.Delete(candidate.path); !r.OK {
			err, _ := r.Value.(error)
			if err == nil {
				err = core.E("planCleanup", core.Concat("failed to delete plan: ", candidate.path), nil)
			}
			return core.Result{Value: err, OK: false}
		}
		deleted++
	}

	output.Archived = archived
	output.Deleted = deleted
	return core.Result{Value: output, OK: true}
}

type planRetentionCandidate struct {
	path       string
	plan       *Plan
	retainedAt time.Time
}

func planRetentionCandidates(dir string, cutoff time.Time) []planRetentionCandidate {
	jsonFiles := core.PathGlob(core.JoinPath(dir, "*.json"))
	slices.Sort(jsonFiles)

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
		if !planRetentionShouldArchive(plan.Status) && plan.Status != "archived" {
			continue
		}

		retainedAt := planRetentionAt(path, plan)
		if retainedAt.IsZero() || !retainedAt.Before(cutoff) {
			continue
		}

		candidates = append(candidates, planRetentionCandidate{
			path:       path,
			plan:       plan,
			retainedAt: retainedAt,
		})
	}

	return candidates
}

func planRetentionShouldArchive(status string) bool {
	switch status {
	case "approved", "completed":
		return true
	default:
		return false
	}
}

func planRetentionAt(path string, plan *Plan) time.Time {
	if plan == nil {
		return time.Time{}
	}

	if !plan.ArchivedAt.IsZero() {
		return plan.ArchivedAt
	}
	if !plan.UpdatedAt.IsZero() {
		return plan.UpdatedAt
	}
	return planArchivedAt(path, plan)
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
