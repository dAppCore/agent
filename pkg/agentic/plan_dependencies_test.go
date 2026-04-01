// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanDependencies_PlanCreate_Good_PreservesPhaseDependencies(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Dependency Plan",
		Objective: "Keep phase dependencies in the stored plan",
		Phases: []Phase{
			{
				Name:         "Build",
				Dependencies: []string{"Setup", "Lint"},
				Criteria:     []string{"tests pass"},
			},
		},
	})
	require.NoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	require.NoError(t, err)
	require.Len(t, plan.Phases, 1)
	assert.Equal(t, []string{"Setup", "Lint"}, plan.Phases[0].Dependencies)
}

func TestPlanDependencies_PhaseDependenciesValue_Bad_MixedTypesReturnsNil(t *testing.T) {
	dependencies := phaseDependenciesValue([]any{"Setup", 7})

	assert.Nil(t, dependencies)
}

func TestPlanDependencies_PhaseDependenciesValue_Ugly_NilInputReturnsNil(t *testing.T) {
	dependencies := phaseDependenciesValue(nil)

	assert.Nil(t, dependencies)
}
