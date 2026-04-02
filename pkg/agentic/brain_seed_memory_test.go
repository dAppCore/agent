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

func TestBrainSeedMemory_CmdBrainSeedMemory_Good(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CORE_HOME", home)

	projectsDir := core.JoinPath(home, ".claude", "projects")
	memoryDir := core.JoinPath(home, ".claude", "projects", "-Users-snider-Code-eaas", "memory")
	require.True(t, fs.EnsureDir(memoryDir).OK)
	require.True(t, fs.Write(core.JoinPath(memoryDir, "MEMORY.md"), "# Memory\n\n## Architecture\nUse Core.Process().\n\n## Decision\nPrefer named actions.").OK)
	require.True(t, fs.Write(core.JoinPath(memoryDir, "notes.md"), "## Convention\nUse UK English.\n").OK)

	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/brain/remember", r.URL.Path)
		bodyResult := core.ReadAll(r.Body)
		require.True(t, bodyResult.OK)
		var payload map[string]any
		require.True(t, core.JSONUnmarshalString(bodyResult.Value.(string), &payload).OK)
		bodies = append(bodies, payload)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"memory":{"id":"mem-1"}}`))
	}))
	defer srv.Close()

	subsystem := &PrepSubsystem{
		brainURL: srv.URL,
		brainKey: "brain-key",
	}

	result := subsystem.cmdBrainSeedMemory(core.NewOptions(
		core.Option{Key: "workspace", Value: "42"},
		core.Option{Key: "path", Value: projectsDir},
		core.Option{Key: "agent", Value: "virgil"},
	))

	require.True(t, result.OK)
	output, ok := result.Value.(BrainSeedMemoryOutput)
	require.True(t, ok)
	assert.Equal(t, 1, output.Files)
	assert.Equal(t, 2, output.Imported)
	assert.Equal(t, 0, output.Skipped)
	assert.Equal(t, false, output.DryRun)
	assert.Equal(t, projectsDir, output.Path)
	require.Len(t, bodies, 2)

	assert.Equal(t, float64(42), bodies[0]["workspace_id"])
	assert.Equal(t, "virgil", bodies[0]["agent_id"])
	assert.Equal(t, "architecture", bodies[0]["type"])
	assert.Equal(t, "eaas", bodies[0]["project"])
	assert.Contains(t, bodies[0]["content"].(string), "Architecture")
	assert.Equal(t, []any{"memory-import"}, bodies[0]["tags"])

	assert.Equal(t, "decision", bodies[1]["type"])
	assert.Equal(t, []any{"memory-import"}, bodies[1]["tags"])
}

func TestBrainSeedMemory_CmdBrainSeedMemory_Good_GlobPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CORE_HOME", home)

	projectsDir := core.JoinPath(home, ".claude", "projects")
	firstMemoryDir := core.JoinPath(projectsDir, "-Users-snider-Code-eaas", "memory")
	secondMemoryDir := core.JoinPath(projectsDir, "-Users-snider-Code-agent", "memory")
	require.True(t, fs.EnsureDir(firstMemoryDir).OK)
	require.True(t, fs.EnsureDir(secondMemoryDir).OK)
	require.True(t, fs.Write(core.JoinPath(firstMemoryDir, "MEMORY.md"), "# Memory\n\n## Architecture\nUse Core.Process().").OK)
	require.True(t, fs.Write(core.JoinPath(secondMemoryDir, "MEMORY.md"), "# Memory\n\n## Decision\nPrefer named actions.").OK)

	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/brain/remember", r.URL.Path)
		bodyResult := core.ReadAll(r.Body)
		require.True(t, bodyResult.OK)
		var payload map[string]any
		require.True(t, core.JSONUnmarshalString(bodyResult.Value.(string), &payload).OK)
		bodies = append(bodies, payload)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"memory":{"id":"mem-1"}}`))
	}))
	defer srv.Close()

	subsystem := &PrepSubsystem{
		brainURL: srv.URL,
		brainKey: "brain-key",
	}

	result := subsystem.cmdBrainSeedMemory(core.NewOptions(
		core.Option{Key: "workspace", Value: "42"},
		core.Option{Key: "path", Value: core.JoinPath(projectsDir, "*", "memory")},
	))

	require.True(t, result.OK)
	output, ok := result.Value.(BrainSeedMemoryOutput)
	require.True(t, ok)
	assert.Equal(t, 2, output.Files)
	assert.Equal(t, 2, output.Imported)
	assert.Equal(t, 0, output.Skipped)
	assert.Equal(t, core.JoinPath(projectsDir, "*", "memory"), output.Path)
	require.Len(t, bodies, 2)
	assert.ElementsMatch(t, []any{"architecture", "decision"}, []any{bodies[0]["type"], bodies[1]["type"]})
}

func TestBrainSeedMemory_CmdBrainIngest_Good(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CORE_HOME", home)

	memoryDir := core.JoinPath(home, ".claude", "projects", "-Users-snider-Code-eaas", "memory")
	require.True(t, fs.EnsureDir(memoryDir).OK)
	require.True(t, fs.Write(core.JoinPath(memoryDir, "MEMORY.md"), "# Memory\n\n## Architecture\nUse Core.Process().").OK)

	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/brain/remember", r.URL.Path)
		bodyResult := core.ReadAll(r.Body)
		require.True(t, bodyResult.OK)
		var payload map[string]any
		require.True(t, core.JSONUnmarshalString(bodyResult.Value.(string), &payload).OK)
		bodies = append(bodies, payload)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"memory":{"id":"mem-1"}}`))
	}))
	defer srv.Close()

	subsystem := &PrepSubsystem{
		brainURL: srv.URL,
		brainKey: "brain-key",
	}

	result := subsystem.cmdBrainIngest(core.NewOptions(
		core.Option{Key: "workspace", Value: "42"},
		core.Option{Key: "path", Value: memoryDir},
	))

	require.True(t, result.OK)
	output, ok := result.Value.(BrainSeedMemoryOutput)
	require.True(t, ok)
	assert.Equal(t, 1, output.Files)
	assert.Equal(t, 1, output.Imported)
	assert.Equal(t, 0, output.Skipped)
	require.Len(t, bodies, 1)
	assert.Equal(t, float64(42), bodies[0]["workspace_id"])
	assert.Equal(t, "architecture", bodies[0]["type"])
}

func TestBrainSeedMemory_CmdBrainIngest_Good_DirectMarkdownFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CORE_HOME", home)

	memoryFile := core.JoinPath(home, "notes.md")
	require.True(t, fs.Write(memoryFile, "# Memory\n\n## Convention\nUse named actions.\n").OK)

	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/brain/remember", r.URL.Path)
		bodyResult := core.ReadAll(r.Body)
		require.True(t, bodyResult.OK)
		var payload map[string]any
		require.True(t, core.JSONUnmarshalString(bodyResult.Value.(string), &payload).OK)
		bodies = append(bodies, payload)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"memory":{"id":"mem-1"}}`))
	}))
	defer srv.Close()

	subsystem := &PrepSubsystem{
		brainURL: srv.URL,
		brainKey: "brain-key",
	}

	result := subsystem.cmdBrainIngest(core.NewOptions(
		core.Option{Key: "workspace", Value: "42"},
		core.Option{Key: "path", Value: memoryFile},
	))

	require.True(t, result.OK)
	output, ok := result.Value.(BrainSeedMemoryOutput)
	require.True(t, ok)
	assert.Equal(t, 1, output.Files)
	assert.Equal(t, 1, output.Imported)
	assert.Equal(t, 0, output.Skipped)
	assert.Equal(t, memoryFile, output.Path)
	require.Len(t, bodies, 1)
	assert.Equal(t, "convention", bodies[0]["type"])
	assert.Contains(t, bodies[0]["content"].(string), "Use named actions.")
}

func TestBrainSeedMemory_CmdBrainSeedMemory_Bad_MissingWorkspace(t *testing.T) {
	subsystem := &PrepSubsystem{brainURL: "https://example.com", brainKey: "brain-key"}

	result := subsystem.cmdBrainSeedMemory(core.NewOptions(
		core.Option{Key: "path", Value: "/tmp/memory"},
	))

	require.False(t, result.OK)
	err, ok := result.Value.(error)
	require.True(t, ok)
	assert.Contains(t, err.Error(), "workspace is required")
}

func TestBrainSeedMemory_CmdBrainIngest_Bad_MissingWorkspace(t *testing.T) {
	subsystem := &PrepSubsystem{brainURL: "https://example.com", brainKey: "brain-key"}

	result := subsystem.cmdBrainIngest(core.NewOptions(
		core.Option{Key: "path", Value: "/tmp/memory"},
	))

	require.False(t, result.OK)
	err, ok := result.Value.(error)
	require.True(t, ok)
	assert.Contains(t, err.Error(), "workspace is required")
}

func TestBrainSeedMemory_CmdBrainSeedMemory_Ugly_PartialImportFailure(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CORE_HOME", home)

	memoryDir := core.JoinPath(home, ".claude", "projects", "-Users-snider-Code-eaas", "memory")
	require.True(t, fs.EnsureDir(memoryDir).OK)
	require.True(t, fs.Write(core.JoinPath(memoryDir, "MEMORY.md"), "## Architecture\nUse Core.Process().\n\n## Decision\nPrefer named actions.").OK)

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		bodyResult := core.ReadAll(r.Body)
		require.True(t, bodyResult.OK)
		var payload map[string]any
		require.True(t, core.JSONUnmarshalString(bodyResult.Value.(string), &payload).OK)
		if calls == 1 {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"memory":{"id":"mem-2"}}`))
	}))
	defer srv.Close()

	subsystem := &PrepSubsystem{
		brainURL: srv.URL,
		brainKey: "brain-key",
	}

	result := subsystem.brainSeedMemory(context.Background(), BrainSeedMemoryInput{
		WorkspaceID: 42,
		AgentID:     "virgil",
		Path:        memoryDir,
	}, true)

	require.True(t, result.OK)
	output, ok := result.Value.(BrainSeedMemoryOutput)
	require.True(t, ok)
	assert.Equal(t, 1, output.Imported)
	assert.Equal(t, 1, output.Skipped)
	assert.Equal(t, 2, calls)
}
