// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActions_OnStartup_Good(t *testing.T) {
	t.Setenv("CORE_BRAIN_URL", "https://api.lthn.sh")
	t.Setenv("CORE_BRAIN_KEY", "test-key")

	c := core.New(core.WithService(Register))
	result := c.ServiceStartup(context.Background(), nil)
	require.True(t, result.OK)

	assert.True(t, c.Action("brain.remember").Exists())
	assert.True(t, c.Action("brain.recall").Exists())
	assert.True(t, c.Action("brain.forget").Exists())
	assert.True(t, c.Action("brain.list").Exists())
	assert.True(t, c.Action("message.send").Exists())
	assert.True(t, c.Action("message.inbox").Exists())
	assert.True(t, c.Action("message.conversation").Exists())
	assert.True(t, c.Action("agent.send").Exists())
	assert.True(t, c.Action("agent.inbox").Exists())
	assert.True(t, c.Action("agent.conversation").Exists())
}

func TestActions_HandleList_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/v1/brain/list", r.URL.Path)
		assert.Equal(t, "core", r.URL.Query().Get("org"))
		assert.Equal(t, "agent", r.URL.Query().Get("project"))
		assert.Equal(t, "decision", r.URL.Query().Get("type"))
		assert.Equal(t, "cladius", r.URL.Query().Get("agent_id"))
		assert.Equal(t, "2", r.URL.Query().Get("limit"))

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(map[string]any{
			"memories": []any{
				map[string]any{
					"id":         "mem-1",
					"content":    "Use brain.list for filtered history",
					"type":       "decision",
					"project":    "agent",
					"agent_id":   "cladius",
					"source":     "manual",
					"confidence": 0.9,
					"created_at": "2026-03-31T00:00:00Z",
					"updated_at": "2026-03-31T00:00:00Z",
				},
			},
		})))
	}))
	defer srv.Close()

	t.Setenv("CORE_BRAIN_URL", srv.URL)
	t.Setenv("CORE_BRAIN_KEY", "test-key")

	c := core.New(core.WithService(Register))
	result := c.ServiceStartup(context.Background(), nil)
	require.True(t, result.OK)

	actionResult := c.Action("brain.list").Run(context.Background(), core.NewOptions(
		core.Option{Key: "org", Value: "core"},
		core.Option{Key: "project", Value: "agent"},
		core.Option{Key: "type", Value: "decision"},
		core.Option{Key: "agent_id", Value: "cladius"},
		core.Option{Key: "limit", Value: 2},
	))
	require.True(t, actionResult.OK)

	output, ok := actionResult.Value.(ListOutput)
	require.True(t, ok)
	assert.True(t, output.Success)
	assert.Equal(t, 1, output.Count)
	require.Len(t, output.Memories, 1)
	assert.Equal(t, "mem-1", output.Memories[0].ID)
	assert.Equal(t, "manual", output.Memories[0].Source)
}

func TestActions_HandleList_Bad(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"down"}`))
	}))
	defer srv.Close()

	t.Setenv("CORE_BRAIN_URL", srv.URL)
	t.Setenv("CORE_BRAIN_KEY", "test-key")

	c := core.New(core.WithService(Register))
	result := c.ServiceStartup(context.Background(), nil)
	require.True(t, result.OK)

	actionResult := c.Action("brain.list").Run(context.Background(), core.NewOptions())
	require.False(t, actionResult.OK)
	err, ok := actionResult.Value.(error)
	require.True(t, ok)
	assert.Contains(t, err.Error(), "upstream returned 500")
}

func TestActions_HandleRecall_Ugly_FilterMap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/brain/recall", r.URL.Path)

		var body map[string]any
		require.True(t, core.JSONUnmarshalString(core.ReadAll(r.Body).Value.(string), &body).OK)
		assert.Equal(t, "architecture", body["query"])
		assert.Equal(t, float64(3), body["top_k"])
		assert.Equal(t, "core", body["org"])
		assert.Equal(t, "agent", body["project"])
		assert.Equal(t, "decision", body["type"])
		assert.Equal(t, "clotho", body["agent_id"])
		assert.Equal(t, 0.75, body["min_confidence"])

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(map[string]any{"memories": []any{}})))
	}))
	defer srv.Close()

	t.Setenv("CORE_BRAIN_URL", srv.URL)
	t.Setenv("CORE_BRAIN_KEY", "test-key")

	c := core.New(core.WithService(Register))
	result := c.ServiceStartup(context.Background(), nil)
	require.True(t, result.OK)

	actionResult := c.Action("brain.recall").Run(context.Background(), core.NewOptions(
		core.Option{Key: "query", Value: "architecture"},
		core.Option{Key: "top_k", Value: 3},
		core.Option{Key: "filter", Value: map[string]any{
			"org":            "core",
			"project":        "agent",
			"type":           "decision",
			"agent_id":       "clotho",
			"min_confidence": 0.75,
		}},
	))
	require.True(t, actionResult.OK)

	output, ok := actionResult.Value.(RecallOutput)
	require.True(t, ok)
	assert.True(t, output.Success)
	assert.Equal(t, 0, output.Count)
}
