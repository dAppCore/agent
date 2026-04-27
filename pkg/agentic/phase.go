// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"time"

	core "dappco.re/go/core"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// input := agentic.PhaseGetInput{PlanSlug: "my-plan-abc123", PhaseOrder: 1}
type PhaseGetInput struct {
	PlanSlug   string `json:"plan_slug"`
	PhaseOrder int    `json:"phase_order"`
}

// input := agentic.PhaseStatusInput{PlanSlug: "my-plan-abc123", PhaseOrder: 1, Status: "completed"}
type PhaseStatusInput struct {
	PlanSlug   string `json:"plan_slug"`
	PhaseOrder int    `json:"phase_order"`
	Status     string `json:"status"`
	Reason     string `json:"reason,omitempty"`
}

// input := agentic.PhaseCheckpointInput{PlanSlug: "my-plan-abc123", PhaseOrder: 1, Note: "Build passes"}
type PhaseCheckpointInput struct {
	PlanSlug   string         `json:"plan_slug"`
	PhaseOrder int            `json:"phase_order"`
	Note       string         `json:"note"`
	Context    map[string]any `json:"context,omitempty"`
}

// out := agentic.PhaseOutput{Success: true, Phase: agentic.Phase{Number: 1, Name: "Setup"}}
type PhaseOutput struct {
	Success bool  `json:"success"`
	Phase   Phase `json:"phase"`
}

// phase := agentic.AgentPhase{Number: 1, Name: "Build", Status: "in_progress"}
type AgentPhase = Phase

// result := c.Action("phase.get").Run(ctx, core.NewOptions(core.Option{Key: "plan_slug", Value: "my-plan-abc123"}))
func (s *PrepSubsystem) handlePhaseGet(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.phaseGet(ctx, nil, PhaseGetInput{
		PlanSlug:   optionStringValue(options, "plan_slug", "plan", "slug"),
		PhaseOrder: optionIntValue(options, "phase_order", "phase"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("phase.update_status").Run(ctx, core.NewOptions(core.Option{Key: "status", Value: "completed"}))
func (s *PrepSubsystem) handlePhaseUpdateStatus(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.phaseUpdateStatus(ctx, nil, PhaseStatusInput{
		PlanSlug:   optionStringValue(options, "plan_slug", "plan", "slug"),
		PhaseOrder: optionIntValue(options, "phase_order", "phase"),
		Status:     optionStringValue(options, "status"),
		Reason:     optionStringValue(options, "reason"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("phase.add_checkpoint").Run(ctx, core.NewOptions(core.Option{Key: "note", Value: "Build passes"}))
func (s *PrepSubsystem) handlePhaseAddCheckpoint(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.phaseAddCheckpoint(ctx, nil, PhaseCheckpointInput{
		PlanSlug:   optionStringValue(options, "plan_slug", "plan", "slug"),
		PhaseOrder: optionIntValue(options, "phase_order", "phase"),
		Note:       optionStringValue(options, "note"),
		Context:    optionAnyMapValue(options, "context"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) registerPhaseTools(svc *coremcp.Service) {
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "phase_get",
		Description: "Get a phase by plan slug and phase order.",
	}, s.phaseGet)

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "phase_update_status",
		Description: "Update a phase status by plan slug and phase order.",
	}, s.phaseUpdateStatus)

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "phase_add_checkpoint",
		Description: "Append a checkpoint note to a phase.",
	}, s.phaseAddCheckpoint)
}

func (s *PrepSubsystem) phaseGet(_ context.Context, _ *mcp.CallToolRequest, input PhaseGetInput) (*mcp.CallToolResult, PhaseOutput, error) {
	plan, phaseIndex, err := planPhaseByOrder(PlansRoot(), input.PlanSlug, input.PhaseOrder)
	if err != nil {
		return nil, PhaseOutput{}, err
	}

	return nil, PhaseOutput{
		Success: true,
		Phase:   plan.Phases[phaseIndex],
	}, nil
}

func (s *PrepSubsystem) phaseUpdateStatus(_ context.Context, _ *mcp.CallToolRequest, input PhaseStatusInput) (*mcp.CallToolResult, PhaseOutput, error) {
	if !validPhaseStatus(input.Status) {
		return nil, PhaseOutput{}, core.E("phaseUpdateStatus", core.Concat("invalid status: ", input.Status), nil)
	}

	plan, phaseIndex, err := planPhaseByOrder(PlansRoot(), input.PlanSlug, input.PhaseOrder)
	if err != nil {
		return nil, PhaseOutput{}, err
	}

	plan.Phases[phaseIndex].Status = input.Status
	if reason := core.Trim(input.Reason); reason != "" {
		plan.Phases[phaseIndex].Notes = appendPlanNote(plan.Phases[phaseIndex].Notes, reason)
	}
	plan.UpdatedAt = time.Now()

	if result := writePlanResult(PlansRoot(), plan); !result.OK {
		err, _ := result.Value.(error)
		if err == nil {
			err = core.E("phaseUpdateStatus", "failed to write plan", nil)
		}
		return nil, PhaseOutput{}, err
	}

	return nil, PhaseOutput{
		Success: true,
		Phase:   plan.Phases[phaseIndex],
	}, nil
}

func (s *PrepSubsystem) phaseAddCheckpoint(_ context.Context, _ *mcp.CallToolRequest, input PhaseCheckpointInput) (*mcp.CallToolResult, PhaseOutput, error) {
	if core.Trim(input.Note) == "" {
		return nil, PhaseOutput{}, core.E("phaseAddCheckpoint", "note is required", nil)
	}

	plan, phaseIndex, err := planPhaseByOrder(PlansRoot(), input.PlanSlug, input.PhaseOrder)
	if err != nil {
		return nil, PhaseOutput{}, err
	}

	plan.Phases[phaseIndex].Checkpoints = append(plan.Phases[phaseIndex].Checkpoints, PhaseCheckpoint{
		Note:      input.Note,
		Context:   input.Context,
		CreatedAt: time.Now().Format(time.RFC3339),
	})
	plan.UpdatedAt = time.Now()

	if result := writePlanResult(PlansRoot(), plan); !result.OK {
		err, _ := result.Value.(error)
		if err == nil {
			err = core.E("phaseAddCheckpoint", "failed to write plan", nil)
		}
		return nil, PhaseOutput{}, err
	}

	return nil, PhaseOutput{
		Success: true,
		Phase:   plan.Phases[phaseIndex],
	}, nil
}

func planPhaseByOrder(dir, planSlug string, phaseOrder int) (*Plan, int, error) {
	if core.Trim(planSlug) == "" {
		return nil, 0, core.E("planPhaseByOrder", "plan_slug is required", nil)
	}
	if phaseOrder <= 0 {
		return nil, 0, core.E("planPhaseByOrder", "phase_order is required", nil)
	}

	plan, err := readPlan(dir, planSlug)
	if err != nil {
		return nil, 0, err
	}

	for index := range plan.Phases {
		if plan.Phases[index].Number == phaseOrder {
			return plan, index, nil
		}
	}

	return nil, 0, core.E("planPhaseByOrder", core.Concat("phase not found: ", core.Sprint(phaseOrder)), nil)
}

func appendPlanNote(existing, note string) string {
	if existing == "" {
		return note
	}
	return core.Concat(existing, "\n", note)
}

func validPhaseStatus(status string) bool {
	switch status {
	case "pending", "in_progress", "completed", "blocked", "skipped":
		return true
	}
	return false
}
