// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPhase_PhaseGet_Good(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Phase Get",
		Description: "Read phase",
		Phases:      []Phase{{Number: 1, Name: "Setup"}},
	})
	require.NoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	require.NoError(t, err)

	_, output, err := s.phaseGet(context.Background(), nil, PhaseGetInput{
		PlanSlug:   plan.Slug,
		PhaseOrder: 1,
	})
	require.NoError(t, err)
	assert.True(t, output.Success)
	assert.Equal(t, "Setup", output.Phase.Name)
}

func TestPhase_PhaseUpdateStatus_Bad_InvalidStatus(t *testing.T) {
	s := newTestPrep(t)
	_, _, err := s.phaseUpdateStatus(context.Background(), nil, PhaseStatusInput{
		PlanSlug:   "my-plan",
		PhaseOrder: 1,
		Status:     "invalid",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status")
}

func TestPhase_PhaseAddCheckpoint_Ugly_AppendsCheckpoint(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Checkpoint Phase",
		Description: "Append checkpoint",
		Phases:      []Phase{{Number: 1, Name: "Setup"}},
	})
	require.NoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	require.NoError(t, err)

	_, output, err := s.phaseAddCheckpoint(context.Background(), nil, PhaseCheckpointInput{
		PlanSlug:   plan.Slug,
		PhaseOrder: 1,
		Note:       "Build passes",
	})
	require.NoError(t, err)
	assert.True(t, output.Success)
	require.Len(t, output.Phase.Checkpoints, 1)
	assert.Equal(t, "Build passes", output.Phase.Checkpoints[0].Note)
}
