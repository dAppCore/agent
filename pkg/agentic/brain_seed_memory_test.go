// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

func TestBrainSeedMemory_CmdBrainSeedMemory_Good_Case(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CORE_HOME", home)

	projectsDir := core.JoinPath(home, ".claude", "projects")
	memoryDir := core.JoinPath(home, ".claude", "projects", "-Users-snider-Code-eaas", "memory")
	core.RequireTrue(t, fs.EnsureDir(memoryDir).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(memoryDir, "MEMORY.md"), "# Memory\n\n## Architecture\nUse Core.Process().\n\n## Decision\nPrefer named actions.").OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(memoryDir, "notes.md"), "## Convention\nUse UK English.\n").OK)

	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/brain/remember", r.URL.Path)
		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)
		var payload map[string]any
		core.RequireTrue(t, core.JSONUnmarshalString(bodyResult.Value.(string), &payload).OK)
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
		core.Option{Key: `path`, Value: projectsDir},
		core.Option{Key: "agent", Value: "virgil"},
	))

	core.RequireTrue(t, result.OK)
	output, ok := result.Value.(BrainSeedMemoryOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 1, output.Files)
	core.AssertEqual(t, 2, output.Imported)
	core.AssertEqual(t, 0, output.Skipped)
	core.AssertEqual(t, false, output.DryRun)
	core.AssertEqual(t, projectsDir, output.Path)
	core.AssertLen(t, bodies, 2)
	if len(bodies) != 2 {
		return
	}

	core.AssertEqual(t, float64(42), bodies[0]["workspace_id"])
	core.AssertEqual(t, "virgil", bodies[0]["agent_id"])
	core.AssertEqual(t, "architecture", bodies[0]["type"])
	core.AssertEqual(t, "eaas", bodies[0]["project"])
	core.AssertContains(t, bodies[0]["content"].(string), "Architecture")
	core.AssertEqual(t, []any{"memory-import"}, bodies[0]["tags"])

	core.AssertEqual(t, "decision", bodies[1]["type"])
	core.AssertEqual(t, []any{"memory-import"}, bodies[1]["tags"])
}

func TestBrainSeedMemory_CmdBrainSeedMemory_Good_GlobPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CORE_HOME", home)

	projectsDir := core.JoinPath(home, ".claude", "projects")
	firstMemoryDir := core.JoinPath(projectsDir, "-Users-snider-Code-eaas", "memory")
	secondMemoryDir := core.JoinPath(projectsDir, "-Users-snider-Code-agent", "memory")
	core.RequireTrue(t, fs.EnsureDir(firstMemoryDir).OK)
	core.RequireTrue(t, fs.EnsureDir(secondMemoryDir).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(firstMemoryDir, "MEMORY.md"), "# Memory\n\n## Architecture\nUse Core.Process().").OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(secondMemoryDir, "MEMORY.md"), "# Memory\n\n## Decision\nPrefer named actions.").OK)

	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/brain/remember", r.URL.Path)
		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)
		var payload map[string]any
		core.RequireTrue(t, core.JSONUnmarshalString(bodyResult.Value.(string), &payload).OK)
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
		core.Option{Key: `path`, Value: core.JoinPath(projectsDir, "*", "memory")},
	))

	core.RequireTrue(t, result.OK)
	output, ok := result.Value.(BrainSeedMemoryOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 2, output.Files)
	core.AssertEqual(t, 2, output.Imported)
	core.AssertEqual(t, 0, output.Skipped)
	core.AssertEqual(t, core.JoinPath(projectsDir, "*", "memory"), output.Path)
	core.AssertLen(t, bodies, 2)
	if len(bodies) != 2 {
		return
	}
	core.AssertElementsMatch(t, []any{"architecture", "decision"}, []any{bodies[0]["type"], bodies[1]["type"]})
}

func TestBrainSeedMemory_CmdBrainIngest_Good_Case(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CORE_HOME", home)

	memoryDir := core.JoinPath(home, ".claude", "projects", "-Users-snider-Code-eaas", "memory")
	core.RequireTrue(t, fs.EnsureDir(memoryDir).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(memoryDir, "MEMORY.md"), "# Memory\n\n## Architecture\nUse Core.Process().").OK)

	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/brain/remember", r.URL.Path)
		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)
		var payload map[string]any
		core.RequireTrue(t, core.JSONUnmarshalString(bodyResult.Value.(string), &payload).OK)
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
		core.Option{Key: `path`, Value: memoryDir},
	))

	core.RequireTrue(t, result.OK)
	output, ok := result.Value.(BrainSeedMemoryOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 1, output.Files)
	core.AssertEqual(t, 1, output.Imported)
	core.AssertEqual(t, 0, output.Skipped)
	core.AssertLen(t, bodies, 1)
	if len(bodies) != 1 {
		return
	}
	core.AssertEqual(t, float64(42), bodies[0]["workspace_id"])
	core.AssertEqual(t, "architecture", bodies[0]["type"])
}

func TestBrainSeedMemory_CmdBrainIngest_Good_DirectMarkdownFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CORE_HOME", home)

	memoryFile := core.JoinPath(home, "notes.md")
	core.RequireTrue(t, fs.Write(memoryFile, "# Memory\n\n## Convention\nUse named actions.\n").OK)

	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/brain/remember", r.URL.Path)
		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)
		var payload map[string]any
		core.RequireTrue(t, core.JSONUnmarshalString(bodyResult.Value.(string), &payload).OK)
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
		core.Option{Key: `path`, Value: memoryFile},
	))

	core.RequireTrue(t, result.OK)
	output, ok := result.Value.(BrainSeedMemoryOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 1, output.Files)
	core.AssertEqual(t, 1, output.Imported)
	core.AssertEqual(t, 0, output.Skipped)
	core.AssertEqual(t, memoryFile, output.Path)
	core.AssertLen(t, bodies, 1)
	if len(bodies) != 1 {
		return
	}
	core.AssertEqual(t, "convention", bodies[0]["type"])
	core.AssertContains(t, bodies[0]["content"].(string), "Use named actions.")
}

func TestBrainSeedMemory_CmdBrainSeedMemory_Bad_MissingWorkspace(t *testing.T) {
	subsystem := &PrepSubsystem{brainURL: "https://example.com", brainKey: "brain-key"}

	result := subsystem.cmdBrainSeedMemory(core.NewOptions(
		core.Option{Key: `path`, Value: "/tmp/memory"},
	))

	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "workspace is required")
}

func TestBrainSeedMemory_CmdBrainIngest_Bad_MissingWorkspace(t *testing.T) {
	subsystem := &PrepSubsystem{brainURL: "https://example.com", brainKey: "brain-key"}

	result := subsystem.cmdBrainIngest(core.NewOptions(
		core.Option{Key: `path`, Value: "/tmp/memory"},
	))

	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "workspace is required")
}

func TestBrainSeedMemory_CmdBrainSeedMemory_Ugly_PartialImportFailure(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CORE_HOME", home)

	memoryDir := core.JoinPath(home, ".claude", "projects", "-Users-snider-Code-eaas", "memory")
	core.RequireTrue(t, fs.EnsureDir(memoryDir).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(memoryDir, "MEMORY.md"), "## Architecture\nUse Core.Process().\n\n## Decision\nPrefer named actions.").OK)

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)
		var payload map[string]any
		core.RequireTrue(t, core.JSONUnmarshalString(bodyResult.Value.(string), &payload).OK)
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

	core.RequireTrue(t, result.OK)
	output, ok := result.Value.(BrainSeedMemoryOutput)
	core.RequireTrue(t, ok)
	// The brain client retries transient 5xx failures, so the first section
	// succeeds on retry and the second section imports normally.
	core.AssertEqual(t, 2, output.Imported)
	core.AssertEqual(t, 0, output.Skipped)
	core.AssertEqual(t, 3, calls)
}
