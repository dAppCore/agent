// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
)

func TestPlanDependencies_PlanCreate_Good_PreservesPhaseDependencies(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

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
	core.RequireNoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	core.RequireNoError(t, err)
	core.AssertLen(t, plan.Phases, 1)
	core.AssertEqual(t, []string{"Setup", "Lint"}, plan.Phases[0].Dependencies)
}

func TestPlanDependencies_PhaseDependenciesValue_Bad_MixedTypesReturnsNil(t *testing.T) {
	dependencies := phaseDependenciesValue([]any{"Setup", 7})
	valid := phaseDependenciesValue([]any{"Setup", "Lint"})
	core.AssertNil(t, dependencies)
	core.AssertEqual(t, []string{"Setup", "Lint"}, valid)
}

func TestPlanDependencies_PhaseDependenciesValue_Ugly_NilInputReturnsNil(t *testing.T) {
	dependencies := phaseDependenciesValue(nil)
	core.AssertNil(t, dependencies)
	core.AssertEqual(t, 0, len(dependencies))
}
