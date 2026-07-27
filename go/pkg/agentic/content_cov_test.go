// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

// TestContentCov_HandleContentBatchGenerate_Good_DryRun — the batch generate
// handler posts batch_id + dry_run, and the batch payload comes back from the
// "batch" envelope key.
func TestContentCov_HandleContentBatchGenerate_Good_DryRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/content/batch/generate", r.URL.Path)
		core.AssertEqual(t, http.MethodPost, r.Method)

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)
		var payload map[string]any
		core.RequireTrue(t, core.JSONUnmarshalString(bodyResult.Value.(string), &payload).OK)
		core.AssertEqual(t, "batch_123", payload["batch_id"])
		core.AssertEqual(t, true, payload["dry_run"])

		_, _ = w.Write([]byte(`{"data":{"batch":{"batch_id":"batch_123","status":"queued","items":3}}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleContentBatchGenerate(context.Background(), core.NewOptions(
		core.Option{Key: "batch_id", Value: "batch_123"},
		core.Option{Key: "dry_run", Value: true},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(ContentBatchOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, "batch_123", stringValue(output.Batch["batch_id"]))
	core.AssertEqual(t, "queued", stringValue(output.Batch["status"]))
}

// TestContentCov_ContentBatchGenerate_Bad_MissingBatchID — an empty batch_id is
// rejected before any request is emitted.
func TestContentCov_ContentBatchGenerate_Bad_MissingBatchID(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	result := subsystem.contentBatchGenerate(context.Background(), ContentBatchGenerateInput{BatchID: "  "})
	core.AssertFalse(t, result.OK)

	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "batch_id is required")
}

// TestContentCov_ContentBatchGenerate_Ugly_RequestFails — a 5xx from the
// platform surfaces as a failure result through failureResult.
func TestContentCov_ContentBatchGenerate_Ugly_RequestFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"batch backend down"}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleContentBatchGenerate(context.Background(), core.NewOptions(
		core.Option{Key: "batch_id", Value: "batch_err"},
	))
	core.AssertFalse(t, result.OK)
}

// TestContentCov_ContentBatchGenerate_Bad_ProviderRejected — when a provider is
// supplied and validateContentProvider rejects it, the batch fails before any
// request (the provider-validation guard).
func TestContentCov_ContentBatchGenerate_Bad_ProviderRejected(t *testing.T) {
	covMiscRestoreValidateContentProvider(t, core.E("contentGenerate", "unknown provider: ghost", nil))

	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	result := subsystem.contentBatchGenerate(context.Background(), ContentBatchGenerateInput{
		BatchID:  "batch_1",
		Provider: "ghost",
	})
	core.AssertFalse(t, result.OK)

	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "unknown provider")
}

// TestContentCov_ContentFromPlan_Bad_ProviderRejected — the from-plan provider
// guard fails the call before any request when the provider is invalid.
func TestContentCov_ContentFromPlan_Bad_ProviderRejected(t *testing.T) {
	covMiscRestoreValidateContentProvider(t, core.E("contentGenerate", "provider unavailable: ghost", nil))

	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	result := subsystem.contentFromPlan(context.Background(), ContentFromPlanInput{
		PlanSlug: "release-notes",
		Provider: "ghost",
	})
	core.AssertFalse(t, result.OK)

	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "provider unavailable")
}

// TestContentCov_ContentFromPlan_Bad_MissingPlanSlug — an empty plan_slug is
// rejected before the request.
func TestContentCov_ContentFromPlan_Bad_MissingPlanSlug(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	result := subsystem.contentFromPlan(context.Background(), ContentFromPlanInput{PlanSlug: "   "})
	core.AssertFalse(t, result.OK)

	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "plan_slug is required")
}

// TestContentCov_ContentFromPlan_Ugly_RequestFails — a failing platform call
// surfaces as a failure result.
func TestContentCov_ContentFromPlan_Ugly_RequestFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleContentFromPlan(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "release-notes"},
	))
	core.AssertFalse(t, result.OK)
}

// TestContentCov_HandleContentFromPlan_Good_PromptTemplatePayloadMerge — the
// from-plan handler merges prompt, template, config and the extra payload map
// into the request body; non-nil payload keys win.
func TestContentCov_HandleContentFromPlan_Good_PromptTemplatePayloadMerge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/content/from-plan", r.URL.Path)

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)
		var payload map[string]any
		core.RequireTrue(t, core.JSONUnmarshalString(bodyResult.Value.(string), &payload).OK)
		core.AssertEqual(t, "release-notes", payload["plan_slug"])
		core.AssertEqual(t, "Summarise the changes", payload["prompt"])
		core.AssertEqual(t, "release-template", payload["template"])
		core.AssertEqual(t, "extra-value", payload["extra_key"])

		config, ok := payload["config"].(map[string]any)
		core.RequireTrue(t, ok)
		core.AssertEqual(t, float64(2000), config["max_tokens"])

		_, _ = w.Write([]byte(`{"data":{"result":{"batch_id":"b9","content":"Plan draft","status":"completed"}}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleContentFromPlan(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "release-notes"},
		core.Option{Key: "prompt", Value: "Summarise the changes"},
		core.Option{Key: "template", Value: "release-template"},
		core.Option{Key: "config", Value: `{"max_tokens":2000}`},
		core.Option{Key: "payload", Value: `{"extra_key":"extra-value"}`},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(ContentFromPlanOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "b9", output.Result.BatchID)
	core.AssertEqual(t, "completed", output.Result.Status)
}

// TestContentCov_ContentStatus_Bad_MissingBatchID — an empty batch_id is
// rejected before the request.
func TestContentCov_ContentStatus_Bad_MissingBatchID(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	result := subsystem.contentStatus(context.Background(), ContentStatusInput{BatchID: ""})
	core.AssertFalse(t, result.OK)

	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "batch_id is required")
}

// TestContentCov_ContentStatus_Ugly_RequestFails — a 503 from the status
// endpoint surfaces as a failure result.
func TestContentCov_ContentStatus_Ugly_RequestFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleContentStatus(context.Background(), core.NewOptions(
		core.Option{Key: "batch_id", Value: "batch_x"},
	))
	core.AssertFalse(t, result.OK)
}

// TestContentCov_ContentBriefCreate_Bad_NoData — with every field blank and no
// payload the body is empty and the create is rejected before any request.
func TestContentCov_ContentBriefCreate_Bad_NoData(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	result := subsystem.contentBriefCreate(context.Background(), ContentBriefCreateInput{})
	core.AssertFalse(t, result.OK)

	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "content brief data is required")
}

// TestContentCov_ContentBriefCreate_Ugly_RequestFails — a 500 from the briefs
// endpoint surfaces as a failure result (request emitted, then fails).
func TestContentCov_ContentBriefCreate_Ugly_RequestFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleContentBriefCreate(context.Background(), core.NewOptions(
		core.Option{Key: "name", Value: "n"},
	))
	core.AssertFalse(t, result.OK)
}

// TestContentCov_ContentBriefGet_Bad_MissingID — an empty brief_id is rejected.
func TestContentCov_ContentBriefGet_Bad_MissingID(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	result := subsystem.contentBriefGet(context.Background(), ContentBriefGetInput{BriefID: ""})
	core.AssertFalse(t, result.OK)
}

// TestContentCov_ContentBriefList_Ugly_RequestFails — a failing list call
// surfaces as a failure result.
func TestContentCov_ContentBriefList_Ugly_RequestFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleContentBriefList(context.Background(), core.NewOptions())
	core.AssertFalse(t, result.OK)
}

// TestContentCov_ContentUsageStats_Ugly_RequestFails — a failing usage call
// surfaces as a failure result.
func TestContentCov_ContentUsageStats_Ugly_RequestFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleContentUsageStats(context.Background(), core.NewOptions())
	core.AssertFalse(t, result.OK)
}

// TestContentCov_MergeContentPayload_Good_NilTargetAndNilValues — a nil target
// is allocated, non-nil extra keys are copied, and nil extra values are
// dropped.
func TestContentCov_MergeContentPayload_Good_NilTargetAndNilValues(t *testing.T) {
	merged := mergeContentPayload(nil, map[string]any{
		"keep": "value",
		"drop": nil,
	})
	core.AssertEqual(t, "value", merged["keep"])
	_, hasDrop := merged["drop"]
	core.AssertFalse(t, hasDrop)
}

// TestContentCov_MergeContentPayload_Ugly_OverwritesTarget — an extra key with
// the same name as a target key overwrites the target value.
func TestContentCov_MergeContentPayload_Ugly_OverwritesTarget(t *testing.T) {
	merged := mergeContentPayload(map[string]any{"k": "old"}, map[string]any{"k": "new"})
	core.AssertEqual(t, "new", merged["k"])
}

// TestContentCov_ContentSchemaGenerate_Good_TechArticleWithStepsAndQuestions —
// a TechArticle carries both how-to steps and FAQ entries when supplied.
func TestContentCov_ContentSchemaGenerate_Good_TechArticleWithStepsAndQuestions(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	result := subsystem.handleContentSchemaGenerate(context.Background(), core.NewOptions(
		core.Option{Key: "type", Value: "tech-article"},
		core.Option{Key: "title", Value: "Wiring the workspace"},
		core.Option{Key: "image", Value: "https://example.test/cover.png"},
		core.Option{Key: "published_at", Value: "2026-01-01"},
		core.Option{Key: "modified_at", Value: "2026-02-01"},
		core.Option{Key: "steps", Value: `[{"name":"Clone","text":"git clone"}]`},
		core.Option{Key: "questions", Value: `[{"question":"Why?","answer":"Because."}]`},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(ContentSchemaOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "TechArticle", output.SchemaType)
	core.AssertEqual(t, "https://example.test/cover.png", output.Schema["image"])
	core.AssertEqual(t, "2026-01-01", output.Schema["datePublished"])
	core.AssertEqual(t, "2026-02-01", output.Schema["dateModified"])

	steps, ok := output.Schema["step"].([]map[string]any)
	core.RequireTrue(t, ok)
	core.AssertLen(t, steps, 1)
	entries, ok := output.Schema["mainEntity"].([]map[string]any)
	core.RequireTrue(t, ok)
	core.AssertLen(t, entries, 1)
}

// TestContentCov_ContentSchemaGenerate_Bad_HowToMissingSteps — a HowTo with no
// steps is rejected.
func TestContentCov_ContentSchemaGenerate_Bad_HowToMissingSteps(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	result := subsystem.contentSchemaGenerate(context.Background(), ContentSchemaInput{
		Type:  "howto",
		Title: "No steps",
	})
	core.AssertFalse(t, result.OK)

	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "steps are required")
}

// TestContentCov_ContentSchemaFAQEntries_Ugly_SkipsBlank — entries with a blank
// question or answer are dropped.
func TestContentCov_ContentSchemaFAQEntries_Ugly_SkipsBlank(t *testing.T) {
	entries := contentSchemaFAQEntries([]ContentSchemaQuestion{
		{Question: "Real?", Answer: "Yes"},
		{Question: "  ", Answer: "Orphan"},
		{Question: "Orphan", Answer: " "},
	})
	core.AssertLen(t, entries, 1)
	core.AssertEqual(t, "Real?", entries[0]["name"])
}

// TestContentCov_ContentSchemaHowToSteps_Ugly_PartialFields — a step with only
// a name (no text/url) still emits, but a fully-blank step is dropped.
func TestContentCov_ContentSchemaHowToSteps_Ugly_PartialFields(t *testing.T) {
	steps := contentSchemaHowToSteps([]ContentSchemaStep{
		{Name: "NameOnly"},
		{Text: "TextOnly", URL: "https://example.test/s"},
		{Name: " ", Text: " "},
	})
	core.AssertLen(t, steps, 2)
	core.AssertEqual(t, "NameOnly", steps[0]["name"])
	_, hasText := steps[0]["text"]
	core.AssertFalse(t, hasText)
	core.AssertEqual(t, "https://example.test/s", steps[1]["url"])
}

// TestContentCov_ContentSchemaQuestionsValue_Bad_SkipsIncomplete — typed
// question values missing a question or answer are filtered out.
func TestContentCov_ContentSchemaQuestionsValue_Bad_SkipsIncomplete(t *testing.T) {
	questions := contentSchemaQuestionsValue([]ContentSchemaQuestion{
		{Question: "Q1", Answer: "A1"},
		{Question: "Q2", Answer: ""},
	})
	core.AssertLen(t, questions, 1)
	core.AssertEqual(t, "Q1", questions[0].Question)
}

// TestContentCov_ContentSchemaStepsValue_Bad_SkipsEmpty — typed step values
// with neither a name nor text are filtered out.
func TestContentCov_ContentSchemaStepsValue_Bad_SkipsEmpty(t *testing.T) {
	steps := contentSchemaStepsValue([]ContentSchemaStep{
		{Name: "Keep", Text: "body"},
		{},
	})
	core.AssertLen(t, steps, 1)
	core.AssertEqual(t, "Keep", steps[0].Name)
}

// TestContentCov_ParseContentBriefListOutput_Good_TotalFromBriefs — when the
// payload omits a total/count the brief length is used as the total.
func TestContentCov_ParseContentBriefListOutput_Good_TotalFromBriefs(t *testing.T) {
	output := parseContentBriefListOutput(map[string]any{
		"data": map[string]any{
			"briefs": []any{
				map[string]any{"id": "b1", "slug": "first"},
				map[string]any{"id": "b2", "slug": "second"},
			},
		},
	})
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, 2, output.Total)
	core.AssertLen(t, output.Briefs, 2)
}

// covMiscRestoreValidateContentProvider swaps the validateContentProvider seam
// for one that always returns the supplied error, restoring it after the test.
func covMiscRestoreValidateContentProvider(t *testing.T, err error) {
	t.Helper()
	previous := validateContentProvider
	validateContentProvider = func(_ *PrepSubsystem, _ string) error { return err }
	t.Cleanup(func() { validateContentProvider = previous })
}
