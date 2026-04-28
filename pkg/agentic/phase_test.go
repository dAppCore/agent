// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
)

func TestPhase_PhaseGet_Good(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Phase Get",
		Description: "Read phase",
		Phases:      []Phase{{Number: 1, Name: "Setup"}},
	})
	core.RequireNoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	core.RequireNoError(t, err)

	_, output, err := s.phaseGet(context.Background(), nil, PhaseGetInput{
		PlanSlug:   plan.Slug,
		PhaseOrder: 1,
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, "Setup", output.Phase.Name)
}

func TestPhase_PhaseUpdateStatus_Bad_InvalidStatus(t *testing.T) {
	s := newTestPrep(t)
	_, _, err := s.phaseUpdateStatus(context.Background(), nil, PhaseStatusInput{
		PlanSlug:   "my-plan",
		PhaseOrder: 1,
		Status:     "invalid",
	})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "invalid status")
}

func TestPhase_PhaseAddCheckpoint_Ugly_AppendsCheckpoint(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Checkpoint Phase",
		Description: "Append checkpoint",
		Phases:      []Phase{{Number: 1, Name: "Setup"}},
	})
	core.RequireNoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	core.RequireNoError(t, err)

	_, output, err := s.phaseAddCheckpoint(context.Background(), nil, PhaseCheckpointInput{
		PlanSlug:   plan.Slug,
		PhaseOrder: 1,
		Note:       "Build passes",
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, output.Success)
	core.AssertLen(t, output.Phase.Checkpoints, 1)
	core.AssertEqual(t, "Build passes", output.Phase.Checkpoints[0].Note)
}
