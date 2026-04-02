// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"time"

	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// plan := &Plan{ID: "id-1-a3f2b1", Title: "Migrate Core", Status: "draft", Objective: "Replace raw process calls with Core.Process()"}
// r := writePlanResult(PlansRoot(), plan)
type Plan struct {
	ID              string              `json:"id"`
	Slug            string              `json:"slug,omitempty"`
	Title           string              `json:"title"`
	Status          string              `json:"status"`
	Repo            string              `json:"repo,omitempty"`
	Org             string              `json:"org,omitempty"`
	Objective       string              `json:"objective"`
	Description     string              `json:"description,omitempty"`
	Context         map[string]any      `json:"context,omitempty"`
	TemplateVersion PlanTemplateVersion `json:"template_version,omitempty"`
	Phases          []Phase             `json:"phases,omitempty"`
	Notes           string              `json:"notes,omitempty"`
	Agent           string              `json:"agent,omitempty"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
	ArchivedAt      time.Time           `json:"archived_at,omitempty"`
}

// AgentPlan is the RFC-named alias for Plan.
type AgentPlan = Plan

// phase := agentic.Phase{Number: 1, Name: "Migrate strings", Status: "in_progress"}
type Phase struct {
	Number       int               `json:"number"`
	Name         string            `json:"name"`
	Description  string            `json:"description,omitempty"`
	Status       string            `json:"status"`
	Criteria     []string          `json:"criteria,omitempty"`
	Dependencies []string          `json:"dependencies,omitempty"`
	Tasks        []PlanTask        `json:"tasks,omitempty"`
	Checkpoints  []PhaseCheckpoint `json:"checkpoints,omitempty"`
	Tests        int               `json:"tests,omitempty"`
	Notes        string            `json:"notes,omitempty"`
}

// task := agentic.PlanTask{ID: "1", Title: "Review imports", Status: "pending", File: "pkg/agentic/plan.go", Line: 46}
type PlanTask struct {
	ID          string `json:"id,omitempty"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Priority    string `json:"priority,omitempty"`
	Category    string `json:"category,omitempty"`
	Status      string `json:"status,omitempty"`
	Notes       string `json:"notes,omitempty"`
	File        string `json:"file,omitempty"`
	Line        int    `json:"line,omitempty"`
	FileRef     string `json:"file_ref,omitempty"`
	LineRef     int    `json:"line_ref,omitempty"`
}

// checkpoint := agentic.PhaseCheckpoint{Note: "Build passes", CreatedAt: "2026-03-31T00:00:00Z"}
type PhaseCheckpoint struct {
	Note      string         `json:"note"`
	Context   map[string]any `json:"context,omitempty"`
	CreatedAt string         `json:"created_at,omitempty"`
}

type PlanCreateInput struct {
	Title           string              `json:"title"`
	Slug            string              `json:"slug,omitempty"`
	Objective       string              `json:"objective,omitempty"`
	Description     string              `json:"description,omitempty"`
	Context         map[string]any      `json:"context,omitempty"`
	TemplateVersion PlanTemplateVersion `json:"template_version,omitempty"`
	Repo            string              `json:"repo,omitempty"`
	Org             string              `json:"org,omitempty"`
	Phases          []Phase             `json:"phases,omitempty"`
	Notes           string              `json:"notes,omitempty"`
}

type PlanCreateOutput struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
	Path    string `json:"path"`
}

type PlanReadInput struct {
	ID   string `json:"id,omitempty"`
	Slug string `json:"slug,omitempty"`
}

type PlanReadOutput struct {
	Success bool `json:"success"`
	Plan    Plan `json:"plan"`
}

type PlanUpdateInput struct {
	ID          string         `json:"id,omitempty"`
	Slug        string         `json:"slug,omitempty"`
	Status      string         `json:"status,omitempty"`
	Title       string         `json:"title,omitempty"`
	Objective   string         `json:"objective,omitempty"`
	Description string         `json:"description,omitempty"`
	Context     map[string]any `json:"context,omitempty"`
	Phases      []Phase        `json:"phases,omitempty"`
	Notes       string         `json:"notes,omitempty"`
	Agent       string         `json:"agent,omitempty"`
}

type PlanUpdateOutput struct {
	Success bool `json:"success"`
	Plan    Plan `json:"plan"`
}

type PlanDeleteInput struct {
	ID     string `json:"id,omitempty"`
	Slug   string `json:"slug,omitempty"`
	Reason string `json:"reason,omitempty"`
}

type PlanDeleteOutput struct {
	Success bool   `json:"success"`
	Deleted string `json:"deleted"`
}

type PlanListInput struct {
	Status string `json:"status,omitempty"`
	Repo   string `json:"repo,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

type PlanListOutput struct {
	Success bool   `json:"success"`
	Count   int    `json:"count"`
	Plans   []Plan `json:"plans"`
}

const planListDefaultLimit = 20

// result := c.Action("plan.create").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "title", Value: "AX RFC follow-up"},
//	core.Option{Key: "objective", Value: "Register plan actions"},
//
// ))
func (s *PrepSubsystem) handlePlanCreate(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.planCreate(ctx, nil, PlanCreateInput{
		Title:       optionStringValue(options, "title"),
		Slug:        optionStringValue(options, "slug"),
		Objective:   optionStringValue(options, "objective"),
		Description: optionStringValue(options, "description"),
		Context:     optionAnyMapValue(options, "context"),
		Repo:        optionStringValue(options, "repo"),
		Org:         optionStringValue(options, "org"),
		Phases:      planPhasesValue(options, "phases"),
		Notes:       optionStringValue(options, "notes"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("plan.read").Run(ctx, core.NewOptions(core.Option{Key: "id", Value: "id-42-a3f2b1"}))
func (s *PrepSubsystem) handlePlanRead(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.planRead(ctx, nil, PlanReadInput{
		ID:   optionStringValue(options, "id", "_arg"),
		Slug: optionStringValue(options, "slug"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("plan.update").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "id", Value: "id-42-a3f2b1"},
//	core.Option{Key: "status", Value: "ready"},
//
// ))
func (s *PrepSubsystem) handlePlanUpdate(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.planUpdate(ctx, nil, PlanUpdateInput{
		ID:          optionStringValue(options, "id", "_arg"),
		Slug:        optionStringValue(options, "slug"),
		Status:      optionStringValue(options, "status"),
		Title:       optionStringValue(options, "title"),
		Objective:   optionStringValue(options, "objective"),
		Description: optionStringValue(options, "description"),
		Context:     optionAnyMapValue(options, "context"),
		Phases:      planPhasesValue(options, "phases"),
		Notes:       optionStringValue(options, "notes"),
		Agent:       optionStringValue(options, "agent"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("plan.delete").Run(ctx, core.NewOptions(core.Option{Key: "id", Value: "id-42-a3f2b1"}))
func (s *PrepSubsystem) handlePlanDelete(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.planDelete(ctx, nil, PlanDeleteInput{
		ID:     optionStringValue(options, "id", "_arg"),
		Slug:   optionStringValue(options, "slug"),
		Reason: optionStringValue(options, "reason"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("plan.list").Run(ctx, core.NewOptions(core.Option{Key: "repo", Value: "go-io"}))
func (s *PrepSubsystem) handlePlanList(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.planList(ctx, nil, PlanListInput{
		Status: optionStringValue(options, "status"),
		Repo:   optionStringValue(options, "repo"),
		Limit:  optionIntValue(options, "limit"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) registerPlanTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_plan_create",
		Description: "Create an implementation plan. Plans track phased work with acceptance criteria, status lifecycle (draft → ready → in_progress → needs_verification → verified → approved), and per-phase progress.",
	}, s.planCreate)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_plan_read",
		Description: "Read an implementation plan by ID. Returns the full plan with all phases, criteria, and status.",
	}, s.planRead)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_plan_update",
		Description: "Update an implementation plan. Supports partial updates — only provided fields are changed. Use this to advance status, update phases, or add notes.",
	}, s.planUpdate)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_plan_delete",
		Description: "Delete an implementation plan by ID. Permanently removes the plan file.",
	}, s.planDelete)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_plan_list",
		Description: "List implementation plans. Supports filtering by status (draft, ready, in_progress, etc.) and repo.",
	}, s.planList)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "plan_create",
		Description: "Create a plan using the slug-based compatibility surface described by the platform RFC.",
	}, s.planCreateCompat)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "plan_read",
		Description: "Read a plan using the legacy plain-name MCP alias.",
	}, s.planRead)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "plan_get",
		Description: "Read a plan by slug with progress details and full phases.",
	}, s.planGetCompat)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "plan_list",
		Description: "List plans using the compatibility surface with slug and progress summaries.",
	}, s.planListCompat)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "plan_check",
		Description: "Check whether a plan or phase is complete using the compatibility surface.",
	}, s.planCheck)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "plan_update",
		Description: "Update a plan using the legacy plain-name MCP alias.",
	}, s.planUpdate)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "plan_update_status",
		Description: "Update a plan lifecycle status by slug.",
	}, s.planUpdateStatusCompat)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "plan_delete",
		Description: "Delete a plan using the legacy plain-name MCP alias.",
	}, s.planDelete)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "plan_archive",
		Description: "Archive a plan by slug without deleting the local record.",
	}, s.planArchiveCompat)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "plan_from_issue",
		Description: "Create an implementation plan from a tracked issue slug or ID.",
	}, s.planFromIssue)
}

func (s *PrepSubsystem) planCreate(_ context.Context, _ *mcp.CallToolRequest, input PlanCreateInput) (*mcp.CallToolResult, PlanCreateOutput, error) {
	if input.Title == "" {
		return nil, PlanCreateOutput{}, core.E("planCreate", "title is required", nil)
	}
	description := input.Description
	if description == "" {
		description = input.Objective
	}
	objective := input.Objective
	if objective == "" {
		objective = description
	}
	if objective == "" {
		return nil, PlanCreateOutput{}, core.E("planCreate", "objective is required", nil)
	}

	id := core.ID()
	plan := Plan{
		ID:              id,
		Slug:            planSlugValue(input.Slug, input.Title, id),
		Title:           input.Title,
		Status:          "draft",
		Repo:            input.Repo,
		Org:             input.Org,
		Objective:       objective,
		Description:     description,
		Context:         input.Context,
		TemplateVersion: input.TemplateVersion,
		Phases:          input.Phases,
		Notes:           input.Notes,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	plan = normalisePlan(plan)

	writeResult := writePlanResult(PlansRoot(), &plan)
	if !writeResult.OK {
		err, _ := writeResult.Value.(error)
		if err == nil {
			err = core.E("planCreate", "failed to write plan", nil)
		}
		return nil, PlanCreateOutput{}, err
	}
	path, ok := writeResult.Value.(string)
	if !ok {
		return nil, PlanCreateOutput{}, core.E("planCreate", "invalid plan write result", nil)
	}

	return nil, PlanCreateOutput{
		Success: true,
		ID:      id,
		Path:    path,
	}, nil
}

func (s *PrepSubsystem) planRead(_ context.Context, _ *mcp.CallToolRequest, input PlanReadInput) (*mcp.CallToolResult, PlanReadOutput, error) {
	ref := planReference(input.ID, input.Slug)
	if ref == "" {
		return nil, PlanReadOutput{}, core.E("planRead", "id is required", nil)
	}

	planResult := readPlanResult(PlansRoot(), ref)
	if !planResult.OK {
		err, _ := planResult.Value.(error)
		if err == nil {
			err = core.E("planRead", "failed to read plan", nil)
		}
		return nil, PlanReadOutput{}, err
	}
	plan, ok := planResult.Value.(*Plan)
	if !ok || plan == nil {
		return nil, PlanReadOutput{}, core.E("planRead", "invalid plan payload", nil)
	}

	return nil, PlanReadOutput{
		Success: true,
		Plan:    *plan,
	}, nil
}

func (s *PrepSubsystem) planUpdate(_ context.Context, _ *mcp.CallToolRequest, input PlanUpdateInput) (*mcp.CallToolResult, PlanUpdateOutput, error) {
	ref := planReference(input.ID, input.Slug)
	if ref == "" {
		return nil, PlanUpdateOutput{}, core.E("planUpdate", "id is required", nil)
	}

	planResult := readPlanResult(PlansRoot(), ref)
	if !planResult.OK {
		err, _ := planResult.Value.(error)
		if err == nil {
			err = core.E("planUpdate", "failed to read plan", nil)
		}
		return nil, PlanUpdateOutput{}, err
	}
	plan, ok := planResult.Value.(*Plan)
	if !ok || plan == nil {
		return nil, PlanUpdateOutput{}, core.E("planUpdate", "invalid plan payload", nil)
	}

	if input.Status != "" {
		if !validPlanStatus(input.Status) {
			return nil, PlanUpdateOutput{}, core.E("planUpdate", core.Concat("invalid status: ", input.Status, " (valid: draft, ready, in_progress, needs_verification, verified, approved)"), nil)
		}
		plan.Status = input.Status
	}
	if input.Title != "" {
		plan.Title = input.Title
	}
	if input.Slug != "" {
		plan.Slug = planSlugValue(input.Slug, plan.Title, plan.ID)
	}
	if input.Objective != "" {
		plan.Objective = input.Objective
		if plan.Description == "" {
			plan.Description = input.Objective
		}
	}
	if input.Description != "" {
		plan.Description = input.Description
		if plan.Objective == "" || input.Objective == "" {
			plan.Objective = input.Description
		}
	}
	if input.Context != nil {
		plan.Context = input.Context
	}
	if input.Phases != nil {
		plan.Phases = input.Phases
	}
	if input.Notes != "" {
		plan.Notes = input.Notes
	}
	if input.Agent != "" {
		plan.Agent = input.Agent
	}

	*plan = normalisePlan(*plan)
	plan.UpdatedAt = time.Now()

	writeResult := writePlanResult(PlansRoot(), plan)
	if !writeResult.OK {
		err, _ := writeResult.Value.(error)
		if err == nil {
			err = core.E("planUpdate", "failed to write plan", nil)
		}
		return nil, PlanUpdateOutput{}, err
	}

	return nil, PlanUpdateOutput{
		Success: true,
		Plan:    *plan,
	}, nil
}

func (s *PrepSubsystem) planDelete(_ context.Context, _ *mcp.CallToolRequest, input PlanDeleteInput) (*mcp.CallToolResult, PlanDeleteOutput, error) {
	plan, err := deletePlanResult(input, "id is required", "planDelete")
	if err != nil {
		return nil, PlanDeleteOutput{}, err
	}

	return nil, PlanDeleteOutput{
		Success: true,
		Deleted: plan.ID,
	}, nil
}

func (s *PrepSubsystem) planList(_ context.Context, _ *mcp.CallToolRequest, input PlanListInput) (*mcp.CallToolResult, PlanListOutput, error) {
	dir := PlansRoot()
	if r := fs.EnsureDir(dir); !r.OK {
		err, _ := r.Value.(error)
		return nil, PlanListOutput{}, core.E("planList", "failed to access plans directory", err)
	}

	limit := input.Limit
	if limit <= 0 {
		limit = planListDefaultLimit
	}

	jsonFiles := core.PathGlob(core.JoinPath(dir, "*.json"))

	var plans []Plan
	for _, f := range jsonFiles {
		id := core.TrimSuffix(core.PathBase(f), ".json")
		planResult := readPlanResult(dir, id)
		if !planResult.OK {
			continue
		}
		plan, ok := planResult.Value.(*Plan)
		if !ok || plan == nil {
			continue
		}

		if input.Status != "" && plan.Status != input.Status {
			continue
		}
		if input.Repo != "" && plan.Repo != input.Repo {
			continue
		}

		plans = append(plans, *plan)
		if len(plans) >= limit {
			break
		}
	}

	return nil, PlanListOutput{
		Success: true,
		Count:   len(plans),
		Plans:   plans,
	}, nil
}

func planPath(dir, id string) string {
	safe := core.SanitisePath(id)
	return core.JoinPath(dir, core.Concat(safe, ".json"))
}

func planPhasesValue(options core.Options, keys ...string) []Phase {
	for _, key := range keys {
		result := options.Get(key)
		if !result.OK {
			continue
		}
		phases := phaseSliceValue(result.Value)
		if len(phases) > 0 {
			return phases
		}
	}
	return nil
}

func phaseSliceValue(value any) []Phase {
	switch typed := value.(type) {
	case []Phase:
		return typed
	case []any:
		phases := make([]Phase, 0, len(typed))
		for _, item := range typed {
			phase, ok := phaseValue(item)
			if ok {
				phases = append(phases, phase)
			}
		}
		return phases
	case string:
		trimmed := core.Trim(typed)
		if trimmed == "" {
			return nil
		}
		if core.HasPrefix(trimmed, "[") {
			var phases []Phase
			if result := core.JSONUnmarshalString(trimmed, &phases); result.OK {
				return phases
			}
			if values := anyMapSliceValue(trimmed); len(values) > 0 {
				return phaseSliceValue(values)
			}
			var generic []any
			if result := core.JSONUnmarshalString(trimmed, &generic); result.OK {
				return phaseSliceValue(generic)
			}
		}
	case []map[string]any:
		phases := make([]Phase, 0, len(typed))
		for _, item := range typed {
			phase, ok := phaseValue(item)
			if ok {
				phases = append(phases, phase)
			}
		}
		return phases
	}
	if phase, ok := phaseValue(value); ok {
		return []Phase{phase}
	}
	return nil
}

func phaseValue(value any) (Phase, bool) {
	switch typed := value.(type) {
	case Phase:
		return typed, true
	case map[string]any:
		return Phase{
			Number:       intValue(typed["number"]),
			Name:         stringValue(typed["name"]),
			Description:  stringValue(typed["description"]),
			Status:       stringValue(typed["status"]),
			Criteria:     stringSliceValue(typed["criteria"]),
			Dependencies: phaseDependenciesValue(typed["dependencies"]),
			Tasks:        planTaskSliceValue(typed["tasks"]),
			Checkpoints:  phaseCheckpointSliceValue(typed["checkpoints"]),
			Tests:        intValue(typed["tests"]),
			Notes:        stringValue(typed["notes"]),
		}, true
	case map[string]string:
		return phaseValue(anyMapValue(typed))
	case string:
		trimmed := core.Trim(typed)
		if trimmed == "" || !core.HasPrefix(trimmed, "{") {
			return Phase{}, false
		}
		if values := anyMapValue(trimmed); len(values) > 0 {
			return phaseValue(values)
		}
	}
	return Phase{}, false
}

func phaseDependenciesValue(value any) []string {
	switch typed := value.(type) {
	case []string:
		return cleanStrings(typed)
	case []any:
		dependencies := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok {
				return nil
			}
			if text = core.Trim(text); text != "" {
				dependencies = append(dependencies, text)
			}
		}
		return dependencies
	case string:
		trimmed := core.Trim(typed)
		if trimmed == "" {
			return nil
		}
		if core.HasPrefix(trimmed, "[") {
			var dependencies []string
			if result := core.JSONUnmarshalString(trimmed, &dependencies); result.OK {
				return cleanStrings(dependencies)
			}
			return nil
		}
		return cleanStrings(core.Split(trimmed, ","))
	default:
		if text := stringValue(value); text != "" {
			return []string{text}
		}
	}
	return nil
}

func planTaskSliceValue(value any) []PlanTask {
	switch typed := value.(type) {
	case []PlanTask:
		return typed
	case []string:
		tasks := make([]PlanTask, 0, len(typed))
		for _, title := range cleanStrings(typed) {
			tasks = append(tasks, PlanTask{Title: title})
		}
		return tasks
	case []any:
		tasks := make([]PlanTask, 0, len(typed))
		for _, item := range typed {
			if task, ok := planTaskValue(item); ok {
				tasks = append(tasks, task)
			}
		}
		return tasks
	case string:
		trimmed := core.Trim(typed)
		if trimmed == "" {
			return nil
		}
		if core.HasPrefix(trimmed, "[") {
			var tasks []PlanTask
			if result := core.JSONUnmarshalString(trimmed, &tasks); result.OK {
				return tasks
			}
			var generic []any
			if result := core.JSONUnmarshalString(trimmed, &generic); result.OK {
				return planTaskSliceValue(generic)
			}
			var titles []string
			if result := core.JSONUnmarshalString(trimmed, &titles); result.OK {
				return planTaskSliceValue(titles)
			}
		}
	case []map[string]any:
		tasks := make([]PlanTask, 0, len(typed))
		for _, item := range typed {
			if task, ok := planTaskValue(item); ok {
				tasks = append(tasks, task)
			}
		}
		return tasks
	}
	if task, ok := planTaskValue(value); ok {
		return []PlanTask{task}
	}
	return nil
}

func planTaskValue(value any) (PlanTask, bool) {
	switch typed := value.(type) {
	case PlanTask:
		return typed, true
	case map[string]any:
		title := stringValue(typed["title"])
		if title == "" {
			title = stringValue(typed["name"])
		}
		file := stringValue(typed["file"])
		if file == "" {
			file = stringValue(typed["file_ref"])
		}
		line := intValue(typed["line"])
		if line == 0 {
			line = intValue(typed["line_ref"])
		}
		return PlanTask{
			ID:          stringValue(typed["id"]),
			Title:       title,
			Description: stringValue(typed["description"]),
			Priority:    stringValue(typed["priority"]),
			Category:    stringValue(typed["category"]),
			Status:      stringValue(typed["status"]),
			Notes:       stringValue(typed["notes"]),
			File:        file,
			Line:        line,
			FileRef:     file,
			LineRef:     line,
		}, title != ""
	case map[string]string:
		return planTaskValue(anyMapValue(typed))
	case string:
		title := core.Trim(typed)
		if title == "" {
			return PlanTask{}, false
		}
		if core.HasPrefix(title, "{") {
			if values := anyMapValue(title); len(values) > 0 {
				return planTaskValue(values)
			}
		}
		return PlanTask{Title: title}, true
	}
	return PlanTask{}, false
}

func phaseCheckpointSliceValue(value any) []PhaseCheckpoint {
	switch typed := value.(type) {
	case []PhaseCheckpoint:
		return typed
	case []any:
		checkpoints := make([]PhaseCheckpoint, 0, len(typed))
		for _, item := range typed {
			if checkpoint, ok := phaseCheckpointValue(item); ok {
				checkpoints = append(checkpoints, checkpoint)
			}
		}
		return checkpoints
	case string:
		trimmed := core.Trim(typed)
		if trimmed == "" {
			return nil
		}
		if core.HasPrefix(trimmed, "[") {
			var checkpoints []PhaseCheckpoint
			if result := core.JSONUnmarshalString(trimmed, &checkpoints); result.OK {
				return checkpoints
			}
			var generic []any
			if result := core.JSONUnmarshalString(trimmed, &generic); result.OK {
				return phaseCheckpointSliceValue(generic)
			}
		}
	case []map[string]any:
		checkpoints := make([]PhaseCheckpoint, 0, len(typed))
		for _, item := range typed {
			if checkpoint, ok := phaseCheckpointValue(item); ok {
				checkpoints = append(checkpoints, checkpoint)
			}
		}
		return checkpoints
	}
	if checkpoint, ok := phaseCheckpointValue(value); ok {
		return []PhaseCheckpoint{checkpoint}
	}
	return nil
}

func phaseCheckpointValue(value any) (PhaseCheckpoint, bool) {
	switch typed := value.(type) {
	case PhaseCheckpoint:
		return typed, typed.Note != ""
	case map[string]any:
		note := stringValue(typed["note"])
		return PhaseCheckpoint{
			Note:      note,
			Context:   anyMapValue(typed["context"]),
			CreatedAt: stringValue(typed["created_at"]),
		}, note != ""
	case map[string]string:
		return phaseCheckpointValue(anyMapValue(typed))
	case string:
		note := core.Trim(typed)
		if note == "" {
			return PhaseCheckpoint{}, false
		}
		if core.HasPrefix(note, "{") {
			if values := anyMapValue(note); len(values) > 0 {
				return phaseCheckpointValue(values)
			}
		}
		return PhaseCheckpoint{Note: note}, true
	}
	return PhaseCheckpoint{}, false
}

// result := readPlanResult(PlansRoot(), "plan-id")
// if result.OK { plan := result.Value.(*Plan) }
func readPlanResult(dir, id string) core.Result {
	path := planPath(dir, id)
	r := fs.Read(path)
	if r.OK {
		return planFromReadResult(r, id)
	}

	if found := findPlanBySlugResult(dir, id); found.OK {
		return found
	}

	err, _ := r.Value.(error)
	if err == nil {
		return core.Result{Value: core.E("readPlan", core.Concat("plan not found: ", id), nil), OK: false}
	}
	return core.Result{Value: core.E("readPlan", core.Concat("plan not found: ", id), err), OK: false}
}

// plan, err := readPlan(PlansRoot(), "plan-id")
func readPlan(dir, id string) (*Plan, error) {
	r := readPlanResult(dir, id)
	if !r.OK {
		err, _ := r.Value.(error)
		if err == nil {
			return nil, core.E("readPlan", "failed to read plan", nil)
		}
		return nil, err
	}
	plan, ok := r.Value.(*Plan)
	if !ok || plan == nil {
		return nil, core.E("readPlan", "invalid plan payload", nil)
	}
	return plan, nil
}

// result := writePlanResult(PlansRoot(), plan)
// if result.OK { path := result.Value.(string) }
func writePlanResult(dir string, plan *Plan) core.Result {
	if plan == nil {
		return core.Result{Value: core.E("writePlan", "plan is required", nil), OK: false}
	}
	normalised := normalisePlan(*plan)
	plan = &normalised
	if r := fs.EnsureDir(dir); !r.OK {
		err, _ := r.Value.(error)
		if err == nil {
			return core.Result{Value: core.E("writePlan", "failed to create plans directory", nil), OK: false}
		}
		return core.Result{Value: core.E("writePlan", "failed to create plans directory", err), OK: false}
	}

	path := planPath(dir, plan.ID)

	if r := fs.WriteAtomic(path, core.JSONMarshalString(plan)); !r.OK {
		err, _ := r.Value.(error)
		if err == nil {
			return core.Result{Value: core.E("writePlan", "failed to write plan", nil), OK: false}
		}
		return core.Result{Value: core.E("writePlan", "failed to write plan", err), OK: false}
	}
	return core.Result{Value: path, OK: true}
}

// path, err := writePlan(PlansRoot(), plan)
func writePlan(dir string, plan *Plan) (string, error) {
	r := writePlanResult(dir, plan)
	if !r.OK {
		err, _ := r.Value.(error)
		if err == nil {
			return "", core.E("writePlan", "failed to write plan", nil)
		}
		return "", err
	}
	path, ok := r.Value.(string)
	if !ok {
		return "", core.E("writePlan", "invalid plan write result", nil)
	}
	return path, nil
}

func validPlanStatus(status string) bool {
	switch status {
	case "draft", "ready", "in_progress", "needs_verification", "verified", "approved":
		return true
	}
	return false
}

func normalisePlan(plan Plan) Plan {
	if plan.Slug == "" {
		plan.Slug = planSlugValue("", plan.Title, plan.ID)
	}
	if plan.Description == "" {
		plan.Description = plan.Objective
	}
	if plan.Objective == "" {
		plan.Objective = plan.Description
	}
	for i := range plan.Phases {
		plan.Phases[i] = normalisePhase(plan.Phases[i], i+1)
	}
	return plan
}

func normalisePhase(phase Phase, number int) Phase {
	if phase.Number == 0 {
		phase.Number = number
	}
	if phase.Status == "" {
		phase.Status = "pending"
	}
	for i := range phase.Tasks {
		phase.Tasks[i] = normalisePlanTask(phase.Tasks[i], i+1)
	}
	for i := range phase.Checkpoints {
		if phase.Checkpoints[i].CreatedAt == "" {
			phase.Checkpoints[i].CreatedAt = time.Now().Format(time.RFC3339)
		}
	}
	return phase
}

func normalisePlanTask(task PlanTask, index int) PlanTask {
	if task.ID == "" {
		task.ID = core.Sprint(index)
	}
	if task.Status == "" {
		task.Status = "pending"
	}
	if task.Title == "" {
		task.Title = task.Description
	}
	task.Priority = core.Trim(task.Priority)
	task.Category = core.Trim(task.Category)
	if task.File == "" {
		task.File = task.FileRef
	}
	if task.FileRef == "" {
		task.FileRef = task.File
	}
	if task.Line == 0 {
		task.Line = task.LineRef
	}
	if task.LineRef == 0 {
		task.LineRef = task.Line
	}
	return task
}

func planReference(id, slug string) string {
	if id != "" {
		return id
	}
	return slug
}

func planFromReadResult(result core.Result, ref string) core.Result {
	var plan Plan
	if ur := core.JSONUnmarshalString(result.Value.(string), &plan); !ur.OK {
		err, _ := ur.Value.(error)
		if err == nil {
			return core.Result{Value: core.E("readPlan", core.Concat("failed to parse plan ", ref), nil), OK: false}
		}
		return core.Result{Value: core.E("readPlan", core.Concat("failed to parse plan ", ref), err), OK: false}
	}
	normalised := normalisePlan(plan)
	return core.Result{Value: &normalised, OK: true}
}

func findPlanBySlugResult(dir, slug string) core.Result {
	ref := core.Trim(slug)
	if ref == "" {
		return core.Result{Value: core.E("readPlan", "plan not found: invalid", nil), OK: false}
	}

	for _, path := range core.PathGlob(core.JoinPath(dir, "*.json")) {
		result := fs.Read(path)
		if !result.OK {
			continue
		}
		planResult := planFromReadResult(result, ref)
		if !planResult.OK {
			continue
		}
		plan, ok := planResult.Value.(*Plan)
		if !ok || plan == nil {
			continue
		}
		if plan.Slug == ref || plan.ID == ref {
			return core.Result{Value: plan, OK: true}
		}
	}

	return core.Result{Value: core.E("readPlan", core.Concat("plan not found: ", ref), nil), OK: false}
}

func planSlugValue(input, title, id string) string {
	slug := cleanPlanSlug(input)
	if slug != "" {
		return slug
	}

	base := cleanPlanSlug(title)
	if base == "" {
		base = "plan"
	}
	suffix := planSlugSuffix(id)
	if suffix == "" {
		return base
	}
	return core.Concat(base, "-", suffix)
}

func cleanPlanSlug(value string) string {
	slug := core.Lower(core.Trim(value))
	if slug == "" {
		return ""
	}
	for _, old := range []string{"/", "\\", "_", ".", ":", ";", ",", " ", "\t", "\n", "\r"} {
		slug = core.Replace(slug, old, "-")
	}
	for core.Contains(slug, "--") {
		slug = core.Replace(slug, "--", "-")
	}
	for core.HasPrefix(slug, "-") {
		slug = slug[1:]
	}
	for core.HasSuffix(slug, "-") {
		slug = slug[:len(slug)-1]
	}
	if slug == "" || slug == "invalid" {
		return ""
	}
	return slug
}

func planSlugSuffix(id string) string {
	parts := core.Split(id, "-")
	if len(parts) == 0 {
		return ""
	}
	return core.Trim(parts[len(parts)-1])
}
