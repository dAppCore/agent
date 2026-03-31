// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"time"

	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// input := agentic.TaskUpdateInput{PlanSlug: "my-plan-abc123", PhaseOrder: 1, TaskIdentifier: "1"}
type TaskUpdateInput struct {
	PlanSlug       string `json:"plan_slug"`
	PhaseOrder     int    `json:"phase_order"`
	TaskIdentifier any    `json:"task_identifier"`
	Status         string `json:"status,omitempty"`
	Notes          string `json:"notes,omitempty"`
}

// input := agentic.TaskToggleInput{PlanSlug: "my-plan-abc123", PhaseOrder: 1, TaskIdentifier: 1}
type TaskToggleInput struct {
	PlanSlug       string `json:"plan_slug"`
	PhaseOrder     int    `json:"phase_order"`
	TaskIdentifier any    `json:"task_identifier"`
}

// out := agentic.TaskOutput{Success: true, Task: agentic.PlanTask{ID: "1", Title: "Review imports"}}
type TaskOutput struct {
	Success bool     `json:"success"`
	Task    PlanTask `json:"task"`
}

// result := c.Action("task.update").Run(ctx, core.NewOptions(core.Option{Key: "plan_slug", Value: "my-plan-abc123"}))
func (s *PrepSubsystem) handleTaskUpdate(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.taskUpdate(ctx, nil, TaskUpdateInput{
		PlanSlug:       optionStringValue(options, "plan_slug", "plan", "slug"),
		PhaseOrder:     optionIntValue(options, "phase_order", "phase"),
		TaskIdentifier: optionAnyValue(options, "task_identifier", "task"),
		Status:         optionStringValue(options, "status"),
		Notes:          optionStringValue(options, "notes"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("task.toggle").Run(ctx, core.NewOptions(core.Option{Key: "plan_slug", Value: "my-plan-abc123"}))
func (s *PrepSubsystem) handleTaskToggle(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.taskToggle(ctx, nil, TaskToggleInput{
		PlanSlug:       optionStringValue(options, "plan_slug", "plan", "slug"),
		PhaseOrder:     optionIntValue(options, "phase_order", "phase"),
		TaskIdentifier: optionAnyValue(options, "task_identifier", "task"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) registerTaskTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "task_update",
		Description: "Update a plan task status or notes by plan slug, phase order, and task identifier.",
	}, s.taskUpdate)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "task_toggle",
		Description: "Toggle a plan task between pending and completed.",
	}, s.taskToggle)
}

func (s *PrepSubsystem) taskUpdate(_ context.Context, _ *mcp.CallToolRequest, input TaskUpdateInput) (*mcp.CallToolResult, TaskOutput, error) {
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

func (s *PrepSubsystem) taskToggle(_ context.Context, _ *mcp.CallToolRequest, input TaskToggleInput) (*mcp.CallToolResult, TaskOutput, error) {
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

func planTaskByIdentifier(dir, planSlug string, phaseOrder int, taskIdentifier any) (*Plan, int, int, error) {
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
