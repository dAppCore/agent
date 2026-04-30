// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

func TestActions_OnStartup_Good_Case(t *testing.T) {
	t.Setenv("CORE_BRAIN_URL", "https://api.lthn.sh")
	t.Setenv("CORE_BRAIN_KEY", "test-key")

	c := core.New(core.WithService(Register))
	result := c.ServiceStartup(context.Background(), nil)
	core.RequireTrue(t, result.OK)

	core.AssertTrue(t, c.Action("brain.remember").Exists())
	core.AssertTrue(t, c.Action("brain.recall").Exists())
	core.AssertTrue(t, c.Action("brain.forget").Exists())
	core.AssertTrue(t, c.Action("brain.list").Exists())
	core.AssertTrue(t, c.Action("message.send").Exists())
	core.AssertTrue(t, c.Action("message.inbox").Exists())
	core.AssertTrue(t, c.Action("message.conversation").Exists())
	core.AssertTrue(t, c.Action("agent.send").Exists())
	core.AssertTrue(t, c.Action("agent.inbox").Exists())
	core.AssertTrue(t, c.Action("agent.conversation").Exists())
}

func TestActions_HandleList_Good_Case(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "GET", r.Method)
		core.AssertEqual(t, "/v1/brain/list", r.URL.Path)
		core.AssertEqual(t, "core", r.URL.Query().Get("org"))
		core.AssertEqual(t, "agent", r.URL.Query().Get("project"))
		core.AssertEqual(t, "decision", r.URL.Query().Get("type"))
		core.AssertEqual(t, "cladius", r.URL.Query().Get("agent_id"))
		core.AssertEqual(t, "2", r.URL.Query().Get("limit"))

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
	core.RequireTrue(t, result.OK)

	actionResult := c.Action("brain.list").Run(context.Background(), core.NewOptions(
		core.Option{Key: "org", Value: "core"},
		core.Option{Key: "project", Value: "agent"},
		core.Option{Key: "type", Value: "decision"},
		core.Option{Key: "agent_id", Value: "cladius"},
		core.Option{Key: "limit", Value: 2},
	))
	core.RequireTrue(t, actionResult.OK)

	output, ok := actionResult.Value.(ListOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, 1, output.Count)
	core.AssertLen(t, output.Memories, 1)
	core.AssertEqual(t, "mem-1", output.Memories[0].ID)
	core.AssertEqual(t, "manual", output.Memories[0].Source)
}

func TestActions_HandleList_Bad_Case(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"down"}`))
	}))
	defer srv.Close()

	t.Setenv("CORE_BRAIN_URL", srv.URL)
	t.Setenv("CORE_BRAIN_KEY", "test-key")

	c := core.New(core.WithService(Register))
	result := c.ServiceStartup(context.Background(), nil)
	core.RequireTrue(t, result.OK)

	actionResult := c.Action("brain.list").Run(context.Background(), core.NewOptions())
	core.AssertFalse(t, actionResult.OK)
	err, ok := actionResult.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "upstream returned 500")
}

func TestActions_HandleRecall_Ugly_FilterMap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "POST", r.Method)
		core.AssertEqual(t, "/v1/brain/recall", r.URL.Path)

		var body map[string]any
		core.RequireTrue(t, core.JSONUnmarshalString(core.ReadAll(r.Body).Value.(string), &body).OK)
		core.AssertEqual(t, "architecture", body["query"])
		core.AssertEqual(t, float64(3), body["top_k"])
		core.AssertEqual(t, "core", body["org"])
		core.AssertEqual(t, "agent", body["project"])
		core.AssertEqual(t, "decision", body["type"])
		core.AssertEqual(t, "clotho", body["agent_id"])
		core.AssertEqual(t, 0.75, body["min_confidence"])

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(map[string]any{"memories": []any{}})))
	}))
	defer srv.Close()

	t.Setenv("CORE_BRAIN_URL", srv.URL)
	t.Setenv("CORE_BRAIN_KEY", "test-key")

	c := core.New(core.WithService(Register))
	result := c.ServiceStartup(context.Background(), nil)
	core.RequireTrue(t, result.OK)

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
	core.RequireTrue(t, actionResult.OK)

	output, ok := actionResult.Value.(RecallOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, 0, output.Count)
}

func TestActions_DirectSubsystem_OnStartup_Good(t *testing.T) {
	t.Setenv("CORE_BRAIN_URL", "https://api.lthn.sh")
	t.Setenv("CORE_BRAIN_KEY", "test-key")
	c := core.New()
	sub := NewDirect()
	sub.ServiceRuntime = core.NewServiceRuntime(c, DirectOptions{})

	result := sub.OnStartup(context.Background())
	core.AssertTrue(t, result.OK)
	core.AssertTrue(t, c.Action("brain.remember").Exists())
}

func TestActions_DirectSubsystem_OnStartup_Bad(t *testing.T) {
	result := (&DirectSubsystem{}).OnStartup(context.Background())
	core.AssertTrue(t, result.OK)
	core.AssertNil(t, result.Value)
}

func TestActions_DirectSubsystem_OnStartup_Ugly(t *testing.T) {
	sub := &DirectSubsystem{ServiceRuntime: core.NewServiceRuntime(nil, DirectOptions{})}
	result := sub.OnStartup(context.Background())
	core.AssertTrue(t, result.OK)
	core.AssertNil(t, result.Value)
}
