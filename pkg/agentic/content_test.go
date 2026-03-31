// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContent_HandleContentGenerate_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/content/generate", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "Bearer secret-token", r.Header.Get("Authorization"))

		bodyResult := core.ReadAll(r.Body)
		require.True(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		require.True(t, parseResult.OK)
		require.Equal(t, "Draft a release note", payload["prompt"])
		require.Equal(t, "claude", payload["provider"])

		config, ok := payload["config"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, float64(4000), config["max_tokens"])

		_, _ = w.Write([]byte(`{"data":{"id":"gen_1","provider":"claude","model":"claude-3.7-sonnet","content":"Release notes draft","input_tokens":12,"output_tokens":48,"duration_ms":321}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleContentGenerate(context.Background(), core.NewOptions(
		core.Option{Key: "prompt", Value: "Draft a release note"},
		core.Option{Key: "provider", Value: "claude"},
		core.Option{Key: "config", Value: `{"max_tokens":4000}`},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(ContentGenerateOutput)
	require.True(t, ok)
	assert.Equal(t, "gen_1", output.Result.ID)
	assert.Equal(t, "claude", output.Result.Provider)
	assert.Equal(t, "claude-3.7-sonnet", output.Result.Model)
	assert.Equal(t, "Release notes draft", output.Result.Content)
	assert.Equal(t, 48, output.Result.OutputTokens)
}

func TestContent_HandleContentGenerate_Bad(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")

	result := subsystem.handleContentGenerate(context.Background(), core.NewOptions())
	assert.False(t, result.OK)
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
	assert.False(t, result.OK)
}

func TestContent_HandleContentBriefCreate_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/content/briefs", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)

		bodyResult := core.ReadAll(r.Body)
		require.True(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		require.True(t, parseResult.OK)
		require.Equal(t, "LinkHost brief", payload["title"])
		require.Equal(t, "LinkHost", payload["product"])

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
	require.True(t, result.OK)

	output, ok := result.Value.(ContentBriefOutput)
	require.True(t, ok)
	assert.Equal(t, "brief_1", output.Brief.ID)
	assert.Equal(t, "host-link", output.Brief.Slug)
	assert.Equal(t, "LinkHost", output.Brief.Product)
}

func TestContent_HandleContentBriefList_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/content/briefs", r.URL.Path)
		require.Equal(t, "product", r.URL.Query().Get("category"))
		require.Equal(t, "5", r.URL.Query().Get("limit"))
		_, _ = w.Write([]byte(`{"data":{"briefs":[{"id":"brief_1","slug":"host-link","title":"LinkHost brief","category":"product"}],"total":1}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleContentBriefList(context.Background(), core.NewOptions(
		core.Option{Key: "category", Value: "product"},
		core.Option{Key: "limit", Value: 5},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(ContentBriefListOutput)
	require.True(t, ok)
	assert.Equal(t, 1, output.Total)
	require.Len(t, output.Briefs, 1)
	assert.Equal(t, "host-link", output.Briefs[0].Slug)
}

func TestContent_HandleContentStatus_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/content/status/batch_123", r.URL.Path)
		_, _ = w.Write([]byte(`{"data":{"status":"running","batch_id":"batch_123","queued":2}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleContentStatus(context.Background(), core.NewOptions(
		core.Option{Key: "batch_id", Value: "batch_123"},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(ContentStatusOutput)
	require.True(t, ok)
	assert.Equal(t, "running", stringValue(output.Status["status"]))
	assert.Equal(t, "batch_123", stringValue(output.Status["batch_id"]))
}

func TestContent_HandleContentUsageStats_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/content/usage/stats", r.URL.Path)
		require.Equal(t, "claude", r.URL.Query().Get("provider"))
		require.Equal(t, "week", r.URL.Query().Get("period"))
		_, _ = w.Write([]byte(`{"data":{"calls":4,"tokens":1200}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleContentUsageStats(context.Background(), core.NewOptions(
		core.Option{Key: "provider", Value: "claude"},
		core.Option{Key: "period", Value: "week"},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(ContentUsageStatsOutput)
	require.True(t, ok)
	assert.Equal(t, 4, intValue(output.Usage["calls"]))
	assert.Equal(t, 1200, intValue(output.Usage["tokens"]))
}

func TestContent_HandleContentFromPlan_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/content/from-plan", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)

		bodyResult := core.ReadAll(r.Body)
		require.True(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		require.True(t, parseResult.OK)
		require.Equal(t, "release-notes", payload["plan_slug"])
		require.Equal(t, "openai", payload["provider"])

		_, _ = w.Write([]byte(`{"data":{"result":{"batch_id":"batch_123","provider":"openai","model":"gpt-5.4","content":"Plan-driven draft","status":"completed"}}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleContentFromPlan(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "release-notes"},
		core.Option{Key: "provider", Value: "openai"},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(ContentFromPlanOutput)
	require.True(t, ok)
	assert.Equal(t, "batch_123", output.Result.BatchID)
	assert.Equal(t, "completed", output.Result.Status)
	assert.Equal(t, "Plan-driven draft", output.Result.Content)
}
