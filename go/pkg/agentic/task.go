// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"time"

	core "dappco.re/go"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// input := agentic.TaskUpdateInput{PlanSlug: "my-plan-abc123", PhaseOrder: 1, TaskIdentifier: "1", File: "pkg/agentic/task.go", Line: 128}
type TaskUpdateInput struct {
	PlanSlug       string `json:"plan_slug"`
	PhaseOrder     int    `json:"phase_order"`
	TaskIdentifier any    `json:"task_identifier"`
	Status         string `json:"status,omitempty"`
	Notes          string `json:"notes,omitempty"`
	Priority       string `json:"priority,omitempty"`
	Category       string `json:"category,omitempty"`
	File           string `json:"file,omitempty"`
	Line           int    `json:"line,omitempty"`
	FileRef        string `json:"file_ref,omitempty"`
	LineRef        int    `json:"line_ref,omitempty"`
}

// input := agentic.TaskToggleInput{PlanSlug: "my-plan-abc123", PhaseOrder: 1, TaskIdentifier: 1}
type TaskToggleInput struct {
	PlanSlug       string `json:"plan_slug"`
	PhaseOrder     int    `json:"phase_order"`
	TaskIdentifier any    `json:"task_identifier"`
}

// input := agentic.TaskCreateInput{PlanSlug: "my-plan-abc123", PhaseOrder: 1, Title: "Review imports", File: "pkg/agentic/task.go", Line: 153}
type TaskCreateInput struct {
	PlanSlug    string `json:"plan_slug"`
	PhaseOrder  int    `json:"phase_order"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
	Notes       string `json:"notes,omitempty"`
	Priority    string `json:"priority,omitempty"`
	Category    string `json:"category,omitempty"`
	File        string `json:"file,omitempty"`
	Line        int    `json:"line,omitempty"`
	FileRef     string `json:"file_ref,omitempty"`
	LineRef     int    `json:"line_ref,omitempty"`
}

// out := agentic.TaskOutput{Success: true, Task: agentic.PlanTask{ID: "1", Title: "Review imports", File: "pkg/agentic/task.go", Line: 128}}
type TaskOutput struct {
	Success bool     `json:"success"`
	Task    PlanTask `json:"task"`
}

// out := agentic.TaskCreateOutput{Success: true, Task: agentic.PlanTask{ID: "3", Title: "Review imports", File: "pkg/agentic/task.go", Line: 153}}
type TaskCreateOutput struct {
	Success bool     `json:"success"`
	Task    PlanTask `json:"task"`
}

// result := c.Action("task.create").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "plan_slug", Value: "my-plan-abc123"},
//	core.Option{Key: "phase_order", Value: 1},
//	core.Option{Key: "title", Value: "Review imports"},
//
// ))
func (s *PrepSubsystem) handleTaskCreate(ctx context.Context, options core.Options) core.Result {
	_, output, err := taskCreate(s, ctx, nil, TaskCreateInput{
		PlanSlug:    optionStringValue(options, "plan_slug", "plan", "slug"),
		PhaseOrder:  optionIntValue(options, "phase_order", "phase"),
		Title:       optionStringValue(options, "title", "task", "_arg"),
		Description: optionStringValue(options, "description"),
		Status:      optionStringValue(options, "status"),
		Notes:       optionStringValue(options, "notes"),
		Priority:    optionStringValue(options, "priority"),
		Category:    optionStringValue(options, "category"),
		File:        optionStringValue(options, "file"),
		Line:        optionIntValue(options, "line"),
		FileRef:     optionStringValue(options, "file_ref", "file-ref"),
		LineRef:     optionIntValue(options, "line_ref", "line-ref"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("task.update").Run(ctx, core.NewOptions(core.Option{Key: "plan_slug", Value: "my-plan-abc123"}))
func (s *PrepSubsystem) handleTaskUpdate(ctx context.Context, options core.Options) core.Result {
	_, output, err := taskUpdate(s, ctx, nil, TaskUpdateInput{
		PlanSlug:       optionStringValue(options, "plan_slug", "plan", "slug"),
		PhaseOrder:     optionIntValue(options, "phase_order", "phase"),
		TaskIdentifier: optionAnyValue(options, "task_identifier", "task"),
		Status:         optionStringValue(options, "status"),
		Notes:          optionStringValue(options, "notes"),
		Priority:       optionStringValue(options, "priority"),
		Category:       optionStringValue(options, "category"),
		File:           optionStringValue(options, "file"),
		Line:           optionIntValue(options, "line"),
		FileRef:        optionStringValue(options, "file_ref", "file-ref"),
		LineRef:        optionIntValue(options, "line_ref", "line-ref"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("task.toggle").Run(ctx, core.NewOptions(core.Option{Key: "plan_slug", Value: "my-plan-abc123"}))
func (s *PrepSubsystem) handleTaskToggle(ctx context.Context, options core.Options) core.Result {
	_, output, err := taskToggle(s, ctx, nil, TaskToggleInput{
		PlanSlug:       optionStringValue(options, "plan_slug", "plan", "slug"),
		PhaseOrder:     optionIntValue(options, "phase_order", "phase"),
		TaskIdentifier: optionAnyValue(options, "task_identifier", "task"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) registerTaskTools(svc *coremcp.Service) {
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "task_create",
		Description: "Create a plan task by plan slug and phase order.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, input TaskCreateInput) (*mcp.CallToolResult, TaskCreateOutput, error) {
		return taskCreate(s, ctx, request, input)
	})
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_task_create",
		Description: "Create a plan task by plan slug and phase order.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, input TaskCreateInput) (*mcp.CallToolResult, TaskCreateOutput, error) {
		return taskCreate(s, ctx, request, input)
	})

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "task_update",
		Description: "Update a plan task status or notes by plan slug, phase order, and task identifier.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, input TaskUpdateInput) (*mcp.CallToolResult, TaskOutput, error) {
		return taskUpdate(s, ctx, request, input)
	})
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_task_update",
		Description: "Update a plan task status or notes by plan slug, phase order, and task identifier.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, input TaskUpdateInput) (*mcp.CallToolResult, TaskOutput, error) {
		return taskUpdate(s, ctx, request, input)
	})

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "task_toggle",
		Description: "Toggle a plan task between pending and completed.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, input TaskToggleInput) (*mcp.CallToolResult, TaskOutput, error) {
		return taskToggle(s, ctx, request, input)
	})
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_task_toggle",
		Description: "Toggle a plan task between pending and completed.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, input TaskToggleInput) (*mcp.CallToolResult, TaskOutput, error) {
		return taskToggle(s, ctx, request, input)
	})
}

var taskUpdate = func(s *PrepSubsystem, _ context.Context, _ *mcp.CallToolRequest, input TaskUpdateInput) (*mcp.CallToolResult, TaskOutput, error) {
	if input.Status != "" && !validTaskStatus(input.Status) {
		return nil, TaskOutput{}, core.E("taskUpdate", core.Concat("invalid status: ", input.Status), nil)
	}
	if taskIdentifierValue(input.TaskIdentifier) == "" {
		return nil, TaskOutput{}, core.E("taskUpdate", "task_identifier is required", nil)
	}

	plan, phaseIndex, taskIndex, err := planTaskByIdentifier(PlansRoot(), input.PlanSlug, input.PhaseOrder, input.TaskIdentifier)
	if err != nil {
		return nil, TaskOutput{}, err
	}

	if input.Status != "" {
		plan.Phases[phaseIndex].Tasks[taskIndex].Status = input.Status
	}
	if notes := core.Trim(input.Notes); notes != "" {
		plan.Phases[phaseIndex].Tasks[taskIndex].Notes = notes
	}
	if priority := core.Trim(input.Priority); priority != "" {
		plan.Phases[phaseIndex].Tasks[taskIndex].Priority = priority
	}
	if category := core.Trim(input.Category); category != "" {
		plan.Phases[phaseIndex].Tasks[taskIndex].Category = category
	}
	if file := core.Trim(input.File); file != "" {
		plan.Phases[phaseIndex].Tasks[taskIndex].File = file
		plan.Phases[phaseIndex].Tasks[taskIndex].FileRef = file
	}
	if fileRef := core.Trim(input.FileRef); fileRef != "" {
		plan.Phases[phaseIndex].Tasks[taskIndex].FileRef = fileRef
		plan.Phases[phaseIndex].Tasks[taskIndex].File = fileRef
	}
	if input.Line > 0 {
		plan.Phases[phaseIndex].Tasks[taskIndex].Line = input.Line
		plan.Phases[phaseIndex].Tasks[taskIndex].LineRef = input.Line
	}
	if input.LineRef > 0 {
		plan.Phases[phaseIndex].Tasks[taskIndex].LineRef = input.LineRef
		plan.Phases[phaseIndex].Tasks[taskIndex].Line = input.LineRef
	}
	plan.UpdatedAt = time.Now()

	if result := writePlanResult(PlansRoot(), plan); !result.OK {
		err, _ := result.Value.(error)
		if err == nil {
			err = core.E("taskUpdate", "failed to write plan", nil)
		}
		return nil, TaskOutput{}, err
	}

	return nil, TaskOutput{
		Success: true,
		Task:    plan.Phases[phaseIndex].Tasks[taskIndex],
	}, nil
}

var taskCreate = func(s *PrepSubsystem, _ context.Context, _ *mcp.CallToolRequest, input TaskCreateInput) (*mcp.CallToolResult, TaskCreateOutput, error) {
	if core.Trim(input.Title) == "" {
		return nil, TaskCreateOutput{}, core.E("taskCreate", "title is required", nil)
	}
	if input.Status != "" && !validTaskStatus(input.Status) {
		return nil, TaskCreateOutput{}, core.E("taskCreate", core.Concat("invalid status: ", input.Status), nil)
	}

	plan, phaseIndex, err := planPhaseByOrder(PlansRoot(), input.PlanSlug, input.PhaseOrder)
	if err != nil {
		return nil, TaskCreateOutput{}, err
	}

	tasks := phaseTaskList(plan.Phases[phaseIndex])
	plan.Phases[phaseIndex].Tasks = tasks

	nextIndex := len(tasks) + 1
	newTask := PlanTask{
		ID:          core.Sprint(nextIndex),
		Title:       core.Trim(input.Title),
		Description: core.Trim(input.Description),
		Priority:    core.Trim(input.Priority),
		Category:    core.Trim(input.Category),
		Status:      input.Status,
		Notes:       core.Trim(input.Notes),
		File:        core.Trim(input.File),
		Line:        input.Line,
		FileRef:     core.Trim(input.FileRef),
		LineRef:     input.LineRef,
	}
	newTask = normalisePlanTask(newTask, nextIndex)

	plan.Phases[phaseIndex].Tasks = append(plan.Phases[phaseIndex].Tasks, newTask)
	plan.UpdatedAt = time.Now()

	if result := writePlanResult(PlansRoot(), plan); !result.OK {
		err, _ := result.Value.(error)
		if err == nil {
			err = core.E("taskCreate", "failed to write plan", nil)
		}
		return nil, TaskCreateOutput{}, err
	}

	return nil, TaskCreateOutput{
		Success: true,
		Task:    newTask,
	}, nil
}

var taskToggle = func(s *PrepSubsystem, _ context.Context, _ *mcp.CallToolRequest, input TaskToggleInput) (*mcp.CallToolResult, TaskOutput, error) {
	if taskIdentifierValue(input.TaskIdentifier) == "" {
		return nil, TaskOutput{}, core.E("taskToggle", "task_identifier is required", nil)
	}

	plan, phaseIndex, taskIndex, err := planTaskByIdentifier(PlansRoot(), input.PlanSlug, input.PhaseOrder, input.TaskIdentifier)
	if err != nil {
		return nil, TaskOutput{}, err
	}

	status := plan.Phases[phaseIndex].Tasks[taskIndex].Status
	if status == "completed" {
		plan.Phases[phaseIndex].Tasks[taskIndex].Status = "pending"
	} else {
		plan.Phases[phaseIndex].Tasks[taskIndex].Status = "completed"
	}
	plan.UpdatedAt = time.Now()

	if result := writePlanResult(PlansRoot(), plan); !result.OK {
		err, _ := result.Value.(error)
		if err == nil {
			err = core.E("taskToggle", "failed to write plan", nil)
		}
		return nil, TaskOutput{}, err
	}

	return nil, TaskOutput{
		Success: true,
		Task:    plan.Phases[phaseIndex].Tasks[taskIndex],
	}, nil
}

var planTaskByIdentifier = func(dir, planSlug string, phaseOrder int, taskIdentifier any) (*Plan, int, int, error) {
	plan, phaseIndex, err := planPhaseByOrder(dir, planSlug, phaseOrder)
	if err != nil {
		return nil, 0, 0, err
	}

	tasks := phaseTaskList(plan.Phases[phaseIndex])
	if len(tasks) == 0 {
		return nil, 0, 0, core.E("planTaskByIdentifier", "phase has no tasks", nil)
	}
	plan.Phases[phaseIndex].Tasks = tasks

	identifier := taskIdentifierValue(taskIdentifier)
	if identifier == "" {
		return nil, 0, 0, core.E("planTaskByIdentifier", "task_identifier is required", nil)
	}

	for index := range plan.Phases[phaseIndex].Tasks {
		task := plan.Phases[phaseIndex].Tasks[index]
		if task.ID == identifier || task.Title == identifier || core.Sprint(index+1) == identifier {
			return plan, phaseIndex, index, nil
		}
	}

	return nil, 0, 0, core.E("planTaskByIdentifier", core.Concat("task not found: ", identifier), nil)
}

func taskIdentifierValue(value any) string {
	switch typed := value.(type) {
	case string:
		return core.Trim(typed)
	case int:
		return core.Sprint(typed)
	case int64:
		return core.Sprint(typed)
	case float64:
		return core.Sprint(int(typed))
	}
	return ""
}

func validTaskStatus(status string) bool {
	switch status {
	case "pending", "completed":
		return true
	}
	return false
}
