// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplate_HandleTemplateList_Good(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleTemplateList(context.Background(), core.NewOptions(
		core.Option{Key: "category", Value: "development"},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(TemplateListOutput)
	require.True(t, ok)
	assert.True(t, output.Success)
	assert.NotZero(t, output.Total)

	found := false
	for _, summary := range output.Templates {
		assert.Equal(t, "development", summary.Category)
		if summary.Slug == "bug-fix" {
			found = true
			assert.Equal(t, "Bug Fix", summary.Name)
			assert.NotZero(t, summary.PhasesCount)
			assert.NotEmpty(t, summary.Variables)
		}
	}
	assert.True(t, found)
}

func TestTemplate_HandleTemplatePreview_Good(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleTemplatePreview(context.Background(), core.NewOptions(
		core.Option{Key: "template", Value: "new-feature"},
		core.Option{Key: "variables", Value: `{"feature_name":"Authentication"}`},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(TemplatePreviewOutput)
	require.True(t, ok)
	assert.True(t, output.Success)
	assert.Equal(t, "new-feature", output.Template)
	assert.Contains(t, output.Preview, "Authentication")
	assert.Contains(t, output.Preview, "Phase 1")
}

func TestTemplate_HandleTemplatePreview_Bad(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleTemplatePreview(context.Background(), core.NewOptions())
	assert.False(t, result.OK)
}

func TestTemplate_HandleTemplatePreview_Ugly_MissingVariables(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleTemplatePreview(context.Background(), core.NewOptions(
		core.Option{Key: "template", Value: "new-feature"},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(TemplatePreviewOutput)
	require.True(t, ok)
	assert.Contains(t, output.Preview, "{{ feature_name }}")
}

func TestTemplate_HandleTemplateCreatePlan_Good(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleTemplateCreatePlan(context.Background(), core.NewOptions(
		core.Option{Key: "template", Value: "new-feature"},
		core.Option{Key: "variables", Value: `{"feature_name":"Authentication"}`},
		core.Option{Key: "title", Value: "Authentication Rollout"},
		core.Option{Key: "plan_slug", Value: "auth-rollout"},
		core.Option{Key: "activate", Value: true},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(TemplateCreatePlanOutput)
	require.True(t, ok)
	assert.True(t, output.Success)
	assert.Equal(t, "auth-rollout", output.Plan.Slug)
	assert.Equal(t, "active", output.Plan.Status)

	plan, err := readPlan(PlansRoot(), "auth-rollout")
	require.NoError(t, err)
	assert.Equal(t, "Authentication Rollout", plan.Title)
	assert.Equal(t, "in_progress", plan.Status)
	require.NotEmpty(t, plan.Phases)
	require.NotEmpty(t, plan.Phases[0].Tasks)
	assert.Equal(t, "pending", plan.Phases[0].Tasks[0].Status)
	assert.Equal(t, "new-feature", stringValue(plan.Context["template"]))
}

func TestTemplate_HandleTemplateCreatePlan_Bad(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleTemplateCreatePlan(context.Background(), core.NewOptions(
		core.Option{Key: "template", Value: "new-feature"},
	))
	assert.False(t, result.OK)
}

func TestTemplate_HandleTemplateCreatePlan_Ugly_UnknownTemplate(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleTemplateCreatePlan(context.Background(), core.NewOptions(
		core.Option{Key: "template", Value: "unknown-template"},
		core.Option{Key: "variables", Value: `{"feature_name":"Authentication"}`},
	))
	assert.False(t, result.OK)
}
