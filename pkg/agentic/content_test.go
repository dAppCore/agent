// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

func TestContent_HandleContentGenerate_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/content/generate", r.URL.Path)
		core.AssertEqual(t, http.MethodPost, r.Method)
		core.AssertEqual(t, "Bearer secret-token", r.Header.Get("Authorization"))

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		core.RequireTrue(t, parseResult.OK)
		core.AssertEqual(t, "Draft a release note", payload["prompt"])
		core.AssertEqual(t, "claude", payload["provider"])

		config, ok := payload["config"].(map[string]any)
		core.RequireTrue(t, ok)
		core.AssertEqual(t, float64(4000), config["max_tokens"])

		_, _ = w.Write([]byte(`{"data":{"id":"gen_1","provider":"claude","model":"claude-3.7-sonnet","content":"Release notes draft","input_tokens":12,"output_tokens":48,"duration_ms":321}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleContentGenerate(context.Background(), core.NewOptions(
		core.Option{Key: "prompt", Value: "Draft a release note"},
		core.Option{Key: "provider", Value: "claude"},
		core.Option{Key: "config", Value: `{"max_tokens":4000}`},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(ContentGenerateOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "gen_1", output.Result.ID)
	core.AssertEqual(t, "claude", output.Result.Provider)
	core.AssertEqual(t, "claude-3.7-sonnet", output.Result.Model)
	core.AssertEqual(t, "Release notes draft", output.Result.Content)
	core.AssertEqual(t, 48, output.Result.OutputTokens)
}

func TestContent_HandleContentGenerate_Good_BriefTemplate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/content/generate", r.URL.Path)
		core.AssertEqual(t, http.MethodPost, r.Method)

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		core.RequireTrue(t, parseResult.OK)
		core.AssertEqual(t, "brief_1", payload["brief_id"])
		core.AssertEqual(t, "help-article", payload["template"])
		core.AssertNotContains(t, payload, "prompt")

		_, _ = w.Write([]byte(`{"data":{"result":{"id":"gen_2","provider":"claude","model":"claude-3.7-sonnet","content":"Template draft","status":"completed"}}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleContentGenerate(context.Background(), core.NewOptions(
		core.Option{Key: "brief_id", Value: "brief_1"},
		core.Option{Key: "template", Value: "help-article"},
		core.Option{Key: "provider", Value: "claude"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(ContentGenerateOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "gen_2", output.Result.ID)
	core.AssertEqual(t, "Template draft", output.Result.Content)
}

func TestContent_HandleContentGenerate_Bad(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")

	result := subsystem.handleContentGenerate(context.Background(), core.NewOptions())
	core.AssertFalse(t, result.OK)
}

func TestContent_HandleContentGenerate_Ugly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleContentGenerate(context.Background(), core.NewOptions(
		core.Option{Key: "prompt", Value: "Draft a release note"},
	))
	core.AssertFalse(t, result.OK)
}

func TestContent_HandleContentBriefCreate_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/content/briefs", r.URL.Path)
		core.AssertEqual(t, http.MethodPost, r.Method)

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		core.RequireTrue(t, parseResult.OK)
		core.AssertEqual(t, "LinkHost brief", payload["title"])
		core.AssertEqual(t, "LinkHost", payload["product"])

		_, _ = w.Write([]byte(`{"data":{"brief":{"id":"brief_1","slug":"host-link","title":"LinkHost brief","product":"LinkHost","category":"product","brief":"Core context"}}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleContentBriefCreate(context.Background(), core.NewOptions(
		core.Option{Key: "title", Value: "LinkHost brief"},
		core.Option{Key: "product", Value: "LinkHost"},
		core.Option{Key: "category", Value: "product"},
		core.Option{Key: "brief", Value: "Core context"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(ContentBriefOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "brief_1", output.Brief.ID)
	core.AssertEqual(t, "host-link", output.Brief.Slug)
	core.AssertEqual(t, "LinkHost", output.Brief.Product)
}

func TestContent_HandleContentBriefList_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/content/briefs", r.URL.Path)
		core.AssertEqual(t, "product", r.URL.Query().Get("category"))
		core.AssertEqual(t, "5", r.URL.Query().Get("limit"))
		_, _ = w.Write([]byte(`{"data":{"briefs":[{"id":"brief_1","slug":"host-link","title":"LinkHost brief","category":"product"}],"total":1}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleContentBriefList(context.Background(), core.NewOptions(
		core.Option{Key: "category", Value: "product"},
		core.Option{Key: "limit", Value: 5},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(ContentBriefListOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 1, output.Total)
	core.AssertLen(t, output.Briefs, 1)
	core.AssertEqual(t, "host-link", output.Briefs[0].Slug)
}

func TestContent_HandleContentStatus_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/content/status/batch_123", r.URL.Path)
		_, _ = w.Write([]byte(`{"data":{"status":"running","batch_id":"batch_123","queued":2}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleContentStatus(context.Background(), core.NewOptions(
		core.Option{Key: "batch_id", Value: "batch_123"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(ContentStatusOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "running", stringValue(output.Status["status"]))
	core.AssertEqual(t, "batch_123", stringValue(output.Status["batch_id"]))
}

func TestContent_HandleContentUsageStats_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/content/usage/stats", r.URL.Path)
		core.AssertEqual(t, "claude", r.URL.Query().Get("provider"))
		core.AssertEqual(t, "week", r.URL.Query().Get("period"))
		_, _ = w.Write([]byte(`{"data":{"calls":4,"tokens":1200}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleContentUsageStats(context.Background(), core.NewOptions(
		core.Option{Key: "provider", Value: "claude"},
		core.Option{Key: "period", Value: "week"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(ContentUsageStatsOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 4, intValue(output.Usage["calls"]))
	core.AssertEqual(t, 1200, intValue(output.Usage["tokens"]))
}

func TestContent_HandleContentFromPlan_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/content/from-plan", r.URL.Path)
		core.AssertEqual(t, http.MethodPost, r.Method)

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		core.RequireTrue(t, parseResult.OK)
		core.AssertEqual(t, "release-notes", payload["plan_slug"])
		core.AssertEqual(t, "openai", payload["provider"])

		_, _ = w.Write([]byte(`{"data":{"result":{"batch_id":"batch_123","provider":"openai","model":"gpt-5.4","content":"Plan-driven draft","status":"completed"}}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleContentFromPlan(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "release-notes"},
		core.Option{Key: "provider", Value: "openai"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(ContentFromPlanOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "batch_123", output.Result.BatchID)
	core.AssertEqual(t, "completed", output.Result.Status)
	core.AssertEqual(t, "Plan-driven draft", output.Result.Content)
}

func TestContent_HandleContentSchemaGenerate_Good_Article(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	result := subsystem.handleContentSchemaGenerate(context.Background(), core.NewOptions(
		core.Option{Key: "type", Value: "article"},
		core.Option{Key: "title", Value: "Release notes"},
		core.Option{Key: "description", Value: "What changed in this release"},
		core.Option{Key: "url", Value: "https://example.test/releases/1"},
		core.Option{Key: "author", Value: "Virgil"},
		core.Option{Key: "language", Value: "en-GB"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(ContentSchemaOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "BlogPosting", output.SchemaType)
	core.AssertEqual(t, "Release notes", output.Schema["headline"])
	core.AssertEqual(t, "What changed in this release", output.Schema["description"])
	core.AssertEqual(t, "https://example.test/releases/1", output.Schema["url"])
	core.AssertEqual(t, "en-GB", output.Schema["inLanguage"])

	author, ok := output.Schema["author"].(map[string]any)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "Virgil", author["name"])
	core.AssertContains(t, output.SchemaJSON, `"@type":"BlogPosting"`)
}

func TestContent_HandleContentSchemaGenerate_Good_FAQ(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	result := subsystem.handleContentSchemaGenerate(context.Background(), core.NewOptions(
		core.Option{Key: "kind", Value: "faq"},
		core.Option{Key: "title", Value: "Release notes FAQ"},
		core.Option{Key: "questions", Value: `[{"question":"What changed?","answer":"We added schema generation."}]`},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(ContentSchemaOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "FAQPage", output.SchemaType)

	entries, ok := output.Schema["mainEntity"].([]map[string]any)
	core.RequireTrue(t, ok)
	core.AssertLen(t, entries, 1)
	core.AssertEqual(t, "Question", entries[0]["@type"])
	core.AssertEqual(t, "What changed?", entries[0]["name"])
	answer, ok := entries[0]["acceptedAnswer"].(map[string]any)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "Answer", answer["@type"])
	core.AssertEqual(t, "We added schema generation.", answer["text"])
}

func TestContent_HandleContentSchemaGenerate_Good_HowTo(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	result := subsystem.handleContentSchemaGenerate(context.Background(), core.NewOptions(
		core.Option{Key: "kind", Value: "howto"},
		core.Option{Key: "title", Value: "Set up the workspace"},
		core.Option{Key: "steps", Value: `[{"name":"Prepare","text":"Clone the repo"},{"name":"Run","text":"Start the agent"}]`},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(ContentSchemaOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "HowTo", output.SchemaType)

	steps, ok := output.Schema["step"].([]map[string]any)
	core.RequireTrue(t, ok)
	core.AssertLen(t, steps, 2)
	core.AssertEqual(t, "HowToStep", steps[0]["@type"])
	core.AssertEqual(t, "Prepare", steps[0]["name"])
	core.AssertEqual(t, "Clone the repo", steps[0]["text"])
}

func TestContent_HandleContentSchemaGenerate_Bad(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	result := subsystem.handleContentSchemaGenerate(context.Background(), core.NewOptions(
		core.Option{Key: "type", Value: "article"},
	))
	core.AssertFalse(t, result.OK)
}

func TestContent_HandleContentSchemaGenerate_Ugly(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	result := subsystem.handleContentSchemaGenerate(context.Background(), core.NewOptions(
		core.Option{Key: "kind", Value: "faq"},
		core.Option{Key: "title", Value: "FAQ"},
		core.Option{Key: "questions", Value: `[{"question":"What changed?"`},
	))
	core.AssertFalse(t, result.OK)
}

func TestContent_contentSchemaType_Good(t *testing.T) {
	core.AssertEqual(t, "BlogPosting", contentSchemaType("article"))
	core.AssertEqual(t, "FAQPage", contentSchemaType("faq"))
	core.AssertEqual(t, "HowTo", contentSchemaType("how-to"))
}

func TestContent_contentSchemaType_Bad(t *testing.T) {
	result := contentSchemaType("press-release")
	core.AssertEmpty(t, result)
	core.AssertEqual(t, "", result)
}

func TestContent_contentSchemaType_Ugly(t *testing.T) {
	techArticle := contentSchemaType("  TECH-ARTICLE  ")
	empty := contentSchemaType("")
	core.AssertEqual(t, "TechArticle", techArticle)
	core.AssertEmpty(t, empty)
}
