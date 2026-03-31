// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSync_HandleSyncPush_Good(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	t.Setenv("CORE_AGENT_API_KEY", "secret-token")

	workspaceDir := core.JoinPath(root, "workspace", "core", "go-io", "task-5")
	fs.EnsureDir(workspaceDir)
	writeStatusResult(workspaceDir, &WorkspaceStatus{
		Status:    "completed",
		Agent:     "codex",
		Repo:      "go-io",
		Org:       "core",
		Task:      "Fix tests",
		Branch:    "agent/fix-tests",
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/agent/sync", r.URL.Path)
		require.Equal(t, "Bearer secret-token", r.Header.Get("Authorization"))
		bodyResult := core.ReadAll(r.Body)
		require.True(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		require.True(t, parseResult.OK)
		require.Equal(t, AgentName(), payload["agent_id"])
		dispatches, ok := payload["dispatches"].([]any)
		require.True(t, ok)
		require.Len(t, dispatches, 1)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"synced":1}}`))
	}))
	defer server.Close()

	subsystem := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:       server.URL,
	}
	output, err := subsystem.syncPush(context.Background(), "")
	require.NoError(t, err)
	assert.True(t, output.Success)
	assert.Equal(t, 1, output.Count)
}

func TestSync_HandleSyncPull_Bad(t *testing.T) {
	t.Setenv("CORE_AGENT_API_KEY", "secret-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/agent/context", r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	}))
	defer server.Close()

	subsystem := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:       server.URL,
	}
	_, err := subsystem.syncPull(context.Background(), "codex")
	require.Error(t, err)
}

func TestSync_HandleSyncPull_Ugly(t *testing.T) {
	t.Setenv("CORE_AGENT_API_KEY", "secret-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/agent/context", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{this is not json`))
	}))
	defer server.Close()

	subsystem := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:       server.URL,
	}
	_, err := subsystem.syncPull(context.Background(), "codex")
	require.Error(t, err)
}
