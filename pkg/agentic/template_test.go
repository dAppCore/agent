// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

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
		assert.Equal(t, 1, summary.Version.Version)
		assert.NotEmpty(t, summary.Version.ContentHash)
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
	assert.Equal(t, 1, output.Version.Version)
	assert.NotEmpty(t, output.Version.ContentHash)
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

func TestTemplate_TemplatePlanTask_Good_FileLineReference(t *testing.T) {
	task := templatePlanTask(map[string]any{
		"title":  "Review RFC",
		"status": "pending",
		"file":   "pkg/agentic/template.go",
		"line":   411,
	}, 1)

	assert.Equal(t, "Review RFC", task.Title)
	assert.Equal(t, "pending", task.Status)
	assert.Equal(t, "pkg/agentic/template.go", task.File)
	assert.Equal(t, 411, task.Line)
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
	assert.Equal(t, 1, output.Version.Version)
	assert.NotEmpty(t, output.Version.ContentHash)

	plan, err := readPlan(PlansRoot(), "auth-rollout")
	require.NoError(t, err)
	assert.Equal(t, "Authentication Rollout", plan.Title)
	assert.Equal(t, "in_progress", plan.Status)
	assert.Equal(t, 1, plan.TemplateVersion.Version)
	assert.Equal(t, "new-feature", plan.TemplateVersion.Slug)
	assert.NotEmpty(t, plan.TemplateVersion.ContentHash)
	require.NotEmpty(t, plan.Phases)
	require.NotEmpty(t, plan.Phases[0].Tasks)
	assert.Equal(t, "pending", plan.Phases[0].Tasks[0].Status)
	assert.Equal(t, "new-feature", stringValue(plan.Context["template"]))
}

func TestTemplate_HandleTemplateCreatePlan_Good_NoVariables(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleTemplateCreatePlan(context.Background(), core.NewOptions(
		core.Option{Key: "template", Value: "api-consistency"},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(TemplateCreatePlanOutput)
	require.True(t, ok)
	assert.True(t, output.Success)
	assert.NotEmpty(t, output.Plan.Slug)
	assert.Equal(t, "API Consistency Audit", output.Plan.Title)
	assert.Equal(t, "draft", output.Plan.Status)
	assert.Equal(t, 1, output.Version.Version)
	assert.NotEmpty(t, output.Version.ContentHash)

	plan, err := readPlan(PlansRoot(), output.Plan.Slug)
	require.NoError(t, err)
	assert.Equal(t, "api-consistency", stringValue(plan.Context["template"]))
	assert.Empty(t, plan.Context["variables"])
	assert.Equal(t, 1, plan.TemplateVersion.Version)
	assert.Equal(t, "api-consistency", plan.TemplateVersion.Slug)
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

func TestTemplate_TemplateVersionFromContent_Good_ReusesExistingVersion(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	content := "name: New Feature\nphases:\n  - name: Setup\n"
	sum := sha256.Sum256([]byte(content))
	hash := hex.EncodeToString(sum[:])

	require.True(t, writePlanResult(PlansRoot(), &Plan{
		ID:     "plan-template-version-good",
		Slug:   "existing-plan",
		Title:  "Existing Plan",
		Status: "draft",
		TemplateVersion: PlanTemplateVersion{
			Slug:        "new-feature",
			Version:     3,
			Name:        "New Feature",
			ContentHash: hash,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}).OK)

	version := templateVersionFromContent("new-feature", "New Feature", content)

	assert.Equal(t, 3, version.Version)
	assert.Equal(t, hash, version.ContentHash)
}

func TestTemplate_TemplateVersionFromContent_Bad_IncrementsOnChangedContent(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	existingContent := "name: New Feature\nphases:\n  - name: Setup\n"
	sum := sha256.Sum256([]byte(existingContent))
	hash := hex.EncodeToString(sum[:])

	require.True(t, writePlanResult(PlansRoot(), &Plan{
		ID:     "plan-template-version-bad",
		Slug:   "existing-plan",
		Title:  "Existing Plan",
		Status: "draft",
		TemplateVersion: PlanTemplateVersion{
			Slug:        "new-feature",
			Version:     3,
			Name:        "New Feature",
			ContentHash: hash,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}).OK)

	version := templateVersionFromContent("new-feature", "New Feature", "name: New Feature\nphases:\n  - name: Discovery\n")

	assert.Equal(t, 4, version.Version)
	assert.NotEqual(t, hash, version.ContentHash)
}

func TestTemplate_TemplateVersionFromContent_Ugly_IgnoresCorruptPlans(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	content := "name: New Feature\nphases:\n  - name: Setup\n"
	sum := sha256.Sum256([]byte(content))
	hash := hex.EncodeToString(sum[:])

	require.True(t, writePlanResult(PlansRoot(), &Plan{
		ID:     "plan-template-version-ugly",
		Slug:   "existing-plan",
		Title:  "Existing Plan",
		Status: "draft",
		TemplateVersion: PlanTemplateVersion{
			Slug:        "new-feature",
			Version:     3,
			Name:        "New Feature",
			ContentHash: hash,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}).OK)

	require.True(t, fs.Write(core.JoinPath(PlansRoot(), "broken.json"), "{").OK)

	version := templateVersionFromContent("new-feature", "New Feature", "name: New Feature\nphases:\n  - name: Discovery\n")

	assert.Equal(t, 4, version.Version)
}
