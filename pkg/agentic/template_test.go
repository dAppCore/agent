// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	core "dappco.re/go"
)

func TestTemplate_HandleTemplateList_Good(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleTemplateList(context.Background(), core.NewOptions(
		core.Option{Key: "category", Value: "development"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(TemplateListOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	assertNotZero(t, output.Total)

	found := false
	for _, summary := range output.Templates {
		core.AssertEqual(t, "development", summary.Category)
		core.AssertEqual(t, 1, summary.Version.Version)
		core.AssertNotEmpty(t, summary.Version.ContentHash)
		core.AssertNotEmpty(t, summary.Version.Content.Name)
		if summary.Slug == "bug-fix" {
			found = true
			core.AssertEqual(t, "Bug Fix", summary.Name)
			assertNotZero(t, summary.PhasesCount)
			core.AssertNotEmpty(t, summary.Variables)
		}
	}
	core.AssertTrue(t, found)
}

func TestTemplate_HandleTemplatePreview_Good(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleTemplatePreview(context.Background(), core.NewOptions(
		core.Option{Key: "template", Value: "new-feature"},
		core.Option{Key: "variables", Value: `{"feature_name":"Authentication"}`},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(TemplatePreviewOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, "new-feature", output.Template)
	core.AssertEqual(t, 1, output.Version.Version)
	core.AssertNotEmpty(t, output.Version.ContentHash)
	core.AssertEqual(t, "new-feature", output.Version.Content.Slug)
	core.AssertEqual(t, "New Feature", output.Version.Content.Name)
	core.AssertContains(t, output.Preview, "Authentication")
	core.AssertContains(t, output.Preview, "Phase 1")
}

func TestTemplate_HandleTemplatePreview_Bad(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleTemplatePreview(context.Background(), core.NewOptions())
	core.AssertFalse(t, result.OK)
}

func TestTemplate_HandleTemplatePreview_Ugly_MissingVariables(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleTemplatePreview(context.Background(), core.NewOptions(
		core.Option{Key: "template", Value: "new-feature"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(TemplatePreviewOutput)
	core.RequireTrue(t, ok)
	core.AssertContains(t, output.Preview, "{{ feature_name }}")
}

func TestTemplate_TemplatePlanTask_Good_FileLineReference(t *testing.T) {
	task := templatePlanTask(map[string]any{
		"title":  "Review RFC",
		"status": "pending",
		"file":   "pkg/agentic/template.go",
		"line":   411,
	}, 1)

	core.AssertEqual(t, "Review RFC", task.Title)
	core.AssertEqual(t, "pending", task.Status)
	core.AssertEqual(t, "pkg/agentic/template.go", task.File)
	core.AssertEqual(t, 411, task.Line)
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
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(TemplateCreatePlanOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, "auth-rollout", output.Plan.Slug)
	core.AssertEqual(t, "active", output.Plan.Status)
	core.AssertEqual(t, 1, output.Version.Version)
	core.AssertNotEmpty(t, output.Version.ContentHash)
	core.AssertEqual(t, "new-feature", output.Version.Content.Slug)
	core.AssertEqual(t, "New Feature", output.Version.Content.Name)

	plan, err := readPlan(PlansRoot(), "auth-rollout")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "Authentication Rollout", plan.Title)
	core.AssertEqual(t, "in_progress", plan.Status)
	core.AssertEqual(t, 1, plan.TemplateVersion.Version)
	core.AssertEqual(t, "new-feature", plan.TemplateVersion.Slug)
	core.AssertNotEmpty(t, plan.TemplateVersion.ContentHash)
	core.RequireNotEmpty(t, plan.Phases)
	core.RequireNotEmpty(t, plan.Phases[0].Tasks)
	core.AssertEqual(t, "pending", plan.Phases[0].Tasks[0].Status)
	core.AssertEqual(t, "new-feature", stringValue(plan.Context["template"]))
}

func TestTemplate_HandleTemplateCreatePlan_Good_NoVariables(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleTemplateCreatePlan(context.Background(), core.NewOptions(
		core.Option{Key: "template", Value: "api-consistency"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(TemplateCreatePlanOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertNotEmpty(t, output.Plan.Slug)
	core.AssertEqual(t, "API Consistency Audit", output.Plan.Title)
	core.AssertEqual(t, "draft", output.Plan.Status)
	core.AssertEqual(t, 1, output.Version.Version)
	core.AssertNotEmpty(t, output.Version.ContentHash)
	core.AssertEqual(t, "api-consistency", output.Version.Content.Slug)

	plan, err := readPlan(PlansRoot(), output.Plan.Slug)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "api-consistency", stringValue(plan.Context["template"]))
	core.AssertEmpty(t, plan.Context["variables"])
	core.AssertEqual(t, 1, plan.TemplateVersion.Version)
	core.AssertEqual(t, "api-consistency", plan.TemplateVersion.Slug)
}

func TestTemplate_HandleTemplateCreatePlan_Bad(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleTemplateCreatePlan(context.Background(), core.NewOptions(
		core.Option{Key: "template", Value: "new-feature"},
	))
	core.AssertFalse(t, result.OK)
}

func TestTemplate_HandleTemplateCreatePlan_Ugly_UnknownTemplate(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleTemplateCreatePlan(context.Background(), core.NewOptions(
		core.Option{Key: "template", Value: "unknown-template"},
		core.Option{Key: "variables", Value: `{"feature_name":"Authentication"}`},
	))
	core.AssertFalse(t, result.OK)
}

func TestTemplate_TemplateVersionFromContent_Good_ReusesExistingVersion(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	content := "name: New Feature\nphases:\n  - name: Setup\n"
	sum := sha256.Sum256([]byte(content))
	hash := hex.EncodeToString(sum[:])

	core.RequireTrue(t, writePlanResult(PlansRoot(), &Plan{
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

	core.AssertEqual(t, 3, version.Version)
	core.AssertEqual(t, hash, version.ContentHash)
	core.AssertEqual(t, "new-feature", version.Content.Slug)
	core.AssertEqual(t, "New Feature", version.Content.Name)
}

func TestTemplate_TemplateVersionFromContent_Bad_IncrementsOnChangedContent(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	existingContent := "name: New Feature\nphases:\n  - name: Setup\n"
	sum := sha256.Sum256([]byte(existingContent))
	hash := hex.EncodeToString(sum[:])

	core.RequireTrue(t, writePlanResult(PlansRoot(), &Plan{
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

	core.AssertEqual(t, 4, version.Version)
	core.AssertNotEqual(t, hash, version.ContentHash)
	core.AssertEqual(t, "new-feature", version.Content.Slug)
}

func TestTemplate_TemplateVersionFromContent_Ugly_IgnoresCorruptPlans(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	content := "name: New Feature\nphases:\n  - name: Setup\n"
	sum := sha256.Sum256([]byte(content))
	hash := hex.EncodeToString(sum[:])

	core.RequireTrue(t, writePlanResult(PlansRoot(), &Plan{
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

	core.RequireTrue(t, fs.Write(core.JoinPath(PlansRoot(), "broken.json"), "{").OK)

	version := templateVersionFromContent("new-feature", "New Feature", "name: New Feature\nphases:\n  - name: Discovery\n")

	core.AssertEqual(t, 4, version.Version)
	core.AssertEqual(t, "new-feature", version.Content.Slug)
}
