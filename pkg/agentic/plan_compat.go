// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"time"

	core "dappco.re/go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// out := agentic.PlanCompatibilitySummary{Slug: "my-plan-abc123", Title: "My Plan", Status: "draft", Phases: 3}
type PlanCompatibilitySummary struct {
	Slug   string `json:"slug"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Phases int    `json:"phases"`
}

// progress := agentic.PlanProgress{Total: 5, Completed: 2, Percentage: 40}
type PlanProgress struct {
	Total      int `json:"total"`
	Completed  int `json:"completed"`
	Percentage int `json:"percentage"`
}

// out := agentic.PlanCompatibilityCreateOutput{Success: true, Plan: agentic.PlanCompatibilitySummary{Slug: "my-plan-abc123"}}
type PlanCompatibilityCreateOutput struct {
	Success bool                     `json:"success"`
	Plan    PlanCompatibilitySummary `json:"plan"`
}

// out := agentic.PlanCompatibilityGetOutput{Success: true}
type PlanCompatibilityGetOutput struct {
	Success bool                  `json:"success"`
	Plan    PlanCompatibilityView `json:"plan"`
}

// out := agentic.PlanCompatibilityListOutput{Success: true, Count: 1}
type PlanCompatibilityListOutput struct {
	Success bool                       `json:"success"`
	Plans   []PlanCompatibilitySummary `json:"plans"`
	Count   int                        `json:"count"`
}

// view := agentic.PlanCompatibilityView{Slug: "my-plan-abc123", Title: "My Plan", Status: "active"}
type PlanCompatibilityView struct {
	Slug            string              `json:"slug"`
	Title           string              `json:"title"`
	Description     string              `json:"description,omitempty"`
	Status          string              `json:"status"`
	Progress        PlanProgress        `json:"progress"`
	TemplateVersion PlanTemplateVersion `json:"template_version,omitempty"`
	Phases          []Phase             `json:"phases,omitempty"`
	Context         map[string]any      `json:"context,omitempty"`
}

// input := agentic.PlanStatusUpdateInput{Slug: "my-plan-abc123", Status: "active"}
type PlanStatusUpdateInput struct {
	Slug   string `json:"slug"`
	Status string `json:"status"`
}

// out := agentic.PlanArchiveOutput{Success: true, Archived: "my-plan-abc123"}
type PlanArchiveOutput struct {
	Success  bool   `json:"success"`
	Archived string `json:"archived"`
}

// input := agentic.PlanCheckInput{Slug: "my-plan-abc123", Phase: 1}
type PlanCheckInput struct {
	Slug  string `json:"slug"`
	Phase int    `json:"phase,omitempty"`
}

// out := agentic.PlanCheckOutput{Success: true, Complete: true}
type PlanCheckOutput struct {
	Success   bool                  `json:"success"`
	Complete  bool                  `json:"complete"`
	Plan      PlanCompatibilityView `json:"plan"`
	Phase     int                   `json:"phase,omitempty"`
	PhaseName string                `json:"phase_name,omitempty"`
	Pending   []string              `json:"pending,omitempty"`
}

// result := c.Action("plan.get").Run(ctx, core.NewOptions(core.Option{Key: "slug", Value: "my-plan-abc123"}))
func (s *PrepSubsystem) handlePlanGet(ctx context.Context, options core.Options) core.Result {
	return s.handlePlanRead(ctx, options)
}

// result := c.Action("plan.archive").Run(ctx, core.NewOptions(core.Option{Key: "slug", Value: "my-plan-abc123"}))
func (s *PrepSubsystem) handlePlanArchive(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.planArchiveCompat(ctx, nil, PlanDeleteInput{
		ID:     optionStringValue(options, "id", "_arg"),
		Slug:   optionStringValue(options, "slug"),
		Reason: optionStringValue(options, "reason"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("plan.update_status").Run(ctx, core.NewOptions(core.Option{Key: "slug", Value: "my-plan-abc123"}))
func (s *PrepSubsystem) handlePlanUpdateStatus(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.planUpdateStatusCompat(ctx, nil, PlanStatusUpdateInput{
		Slug:   optionStringValue(options, "slug", "_arg"),
		Status: optionStringValue(options, "status"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("plan.check").Run(ctx, core.NewOptions(core.Option{Key: "slug", Value: "my-plan-abc123"}))
func (s *PrepSubsystem) handlePlanCheck(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.planCheck(ctx, nil, PlanCheckInput{
		Slug:  optionStringValue(options, "slug", "_arg"),
		Phase: optionIntValue(options, "phase", "phase_order"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) planCreateCompat(ctx context.Context, _ *mcp.CallToolRequest, input PlanCreateInput) (*mcp.CallToolResult, PlanCompatibilityCreateOutput, error) {
	_, created, err := s.planCreate(ctx, nil, input)
	if err != nil {
		return nil, PlanCompatibilityCreateOutput{}, err
	}

	plan, err := readPlan(PlansRoot(), created.ID)
	if err != nil {
		return nil, PlanCompatibilityCreateOutput{}, err
	}

	return nil, PlanCompatibilityCreateOutput{
		Success: true,
		Plan:    planCompatibilitySummary(*plan),
	}, nil
}

func (s *PrepSubsystem) planGetCompat(ctx context.Context, _ *mcp.CallToolRequest, input PlanReadInput) (*mcp.CallToolResult, PlanCompatibilityGetOutput, error) {
	if input.Slug == "" && input.ID == "" {
		return nil, PlanCompatibilityGetOutput{}, core.E("planGetCompat", "slug is required", nil)
	}

	_, output, err := s.planRead(ctx, nil, input)
	if err != nil {
		return nil, PlanCompatibilityGetOutput{}, err
	}

	return nil, PlanCompatibilityGetOutput{
		Success: true,
		Plan:    planCompatibilityView(output.Plan),
	}, nil
}

func (s *PrepSubsystem) planListCompat(ctx context.Context, _ *mcp.CallToolRequest, input PlanListInput) (*mcp.CallToolResult, PlanCompatibilityListOutput, error) {
	if input.Status != "" {
		input.Status = planCompatibilityInputStatus(input.Status)
	}

	_, output, err := s.planList(ctx, nil, input)
	if err != nil {
		return nil, PlanCompatibilityListOutput{}, err
	}

	summaries := make([]PlanCompatibilitySummary, 0, len(output.Plans))
	for _, plan := range output.Plans {
		summaries = append(summaries, planCompatibilitySummary(plan))
	}

	return nil, PlanCompatibilityListOutput{
		Success: true,
		Plans:   summaries,
		Count:   len(summaries),
	}, nil
}

func (s *PrepSubsystem) planUpdateStatusCompat(ctx context.Context, _ *mcp.CallToolRequest, input PlanStatusUpdateInput) (*mcp.CallToolResult, PlanCompatibilityGetOutput, error) {
	if input.Slug == "" {
		return nil, PlanCompatibilityGetOutput{}, core.E("planUpdateStatusCompat", "slug is required", nil)
	}
	if input.Status == "" {
		return nil, PlanCompatibilityGetOutput{}, core.E("planUpdateStatusCompat", "status is required", nil)
	}

	internalStatus := planCompatibilityInputStatus(input.Status)
	_, output, err := s.planUpdate(ctx, nil, PlanUpdateInput{
		Slug:   input.Slug,
		Status: internalStatus,
	})
	if err != nil {
		return nil, PlanCompatibilityGetOutput{}, err
	}

	return nil, PlanCompatibilityGetOutput{
		Success: true,
		Plan:    planCompatibilityView(output.Plan),
	}, nil
}

func (s *PrepSubsystem) planCheck(ctx context.Context, _ *mcp.CallToolRequest, input PlanCheckInput) (*mcp.CallToolResult, PlanCheckOutput, error) {
	if input.Slug == "" {
		return nil, PlanCheckOutput{}, core.E("planCheck", "slug is required", nil)
	}

	_, output, err := s.planGetCompat(ctx, nil, PlanReadInput{Slug: input.Slug})
	if err != nil {
		return nil, PlanCheckOutput{}, err
	}

	return nil, planCheckOutput(output.Plan, input.Phase), nil
}

func (s *PrepSubsystem) planArchiveCompat(ctx context.Context, _ *mcp.CallToolRequest, input PlanDeleteInput) (*mcp.CallToolResult, PlanArchiveOutput, error) {
	plan, err := archivePlanResult(input, "slug is required", "planArchiveCompat")
	if err != nil {
		return nil, PlanArchiveOutput{}, err
	}

	return nil, PlanArchiveOutput{
		Success:  true,
		Archived: plan.Slug,
	}, nil
}

func planCompatibilitySummary(plan Plan) PlanCompatibilitySummary {
	return PlanCompatibilitySummary{
		Slug:   plan.Slug,
		Title:  plan.Title,
		Status: planCompatibilityOutputStatus(plan.Status),
		Phases: len(plan.Phases),
	}
}

func planCompatibilityView(plan Plan) PlanCompatibilityView {
	return PlanCompatibilityView{
		Slug:            plan.Slug,
		Title:           plan.Title,
		Description:     plan.Description,
		Status:          planCompatibilityOutputStatus(plan.Status),
		Progress:        planProgress(plan),
		TemplateVersion: plan.TemplateVersion,
		Phases:          plan.Phases,
		Context:         plan.Context,
	}
}

func archiveReasonValue(reason string) string {
	trimmed := core.Trim(reason)
	if trimmed == "" {
		return ""
	}
	return core.Concat("Archived: ", trimmed)
}

func planProgress(plan Plan) PlanProgress {
	total := 0
	completed := 0

	for _, phase := range plan.Phases {
		tasks := phaseTaskList(phase)
		if len(tasks) > 0 {
			total += len(tasks)
			for _, task := range tasks {
				if task.Status == "completed" {
					completed++
				}
			}
			continue
		}

		total++
		switch phase.Status {
		case "completed", "done", "approved":
			completed++
		}
	}

	percentage := 0
	if total > 0 {
		percentage = (completed * 100) / total
	}

	return PlanProgress{
		Total:      total,
		Completed:  completed,
		Percentage: percentage,
	}
}

func phaseTaskList(phase Phase) []PlanTask {
	if len(phase.Tasks) > 0 {
		tasks := make([]PlanTask, 0, len(phase.Tasks))
		for i := range phase.Tasks {
			tasks = append(tasks, normalisePlanTask(phase.Tasks[i], i+1))
		}
		return tasks
	}

	criteria := phaseCriteriaList(phase)
	if len(criteria) == 0 {
		return nil
	}

	tasks := make([]PlanTask, 0, len(criteria))
	for index, criterion := range criteria {
		tasks = append(tasks, normalisePlanTask(PlanTask{Title: criterion}, index+1))
	}
	return tasks
}

func planCompatibilityInputStatus(status string) string {
	switch status {
	case "active":
		return "in_progress"
	case "completed":
		return "approved"
	default:
		return status
	}
}

func planCompatibilityOutputStatus(status string) string {
	switch status {
	case "in_progress", "needs_verification", "verified":
		return "active"
	case "approved":
		return "completed"
	default:
		return status
	}
}

func archivePlanResult(input PlanDeleteInput, missingMessage, op string) (*Plan, error) {
	ref := planReference(input.ID, input.Slug)
	if ref == "" {
		return nil, core.E(op, missingMessage, nil)
	}

	plan, err := readPlan(PlansRoot(), ref)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	plan.Status = "archived"
	plan.ArchivedAt = now
	plan.UpdatedAt = now
	if notes := archiveReasonValue(input.Reason); notes != "" {
		plan.Notes = appendPlanNote(plan.Notes, notes)
	}
	if result := writePlanResult(PlansRoot(), plan); !result.OK {
		err, _ := result.Value.(error)
		if err == nil {
			err = core.E(op, "failed to write plan", nil)
		}
		return nil, err
	}

	return plan, nil
}

func deletePlanResult(input PlanDeleteInput, missingMessage, op string) (*Plan, error) {
	ref := planReference(input.ID, input.Slug)
	if ref == "" {
		return nil, core.E(op, missingMessage, nil)
	}

	plan, err := readPlan(PlansRoot(), ref)
	if err != nil {
		return nil, err
	}

	planPath := planPath(PlansRoot(), plan.ID)
	if deleteResult := fs.Delete(planPath); !deleteResult.OK {
		deleteErr, _ := deleteResult.Value.(error)
		return nil, core.E(op, "failed to delete plan", deleteErr)
	}

	if plan.Slug != "" {
		stateFile := statePath(plan.Slug)
		if fs.Exists(stateFile) {
			if deleteResult := fs.Delete(stateFile); !deleteResult.OK {
				deleteErr, _ := deleteResult.Value.(error)
				return nil, core.E(op, "failed to delete plan state", deleteErr)
			}
		}
	}

	return plan, nil
}
