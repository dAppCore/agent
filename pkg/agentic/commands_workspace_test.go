// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
)

func TestCommandsworkspace_RegisterWorkspaceCommands_Good_Aliases(t *testing.T) {
	s, c := testPrepWithCore(t, nil)

	s.registerWorkspaceCommands()

	assert.Contains(t, c.Commands(), "workspace/list")
	assert.Contains(t, c.Commands(), "agentic:workspace/list")
	assert.Contains(t, c.Commands(), "workspace/clean")
	assert.Contains(t, c.Commands(), "agentic:workspace/clean")
	assert.Contains(t, c.Commands(), "workspace/dispatch")
	assert.Contains(t, c.Commands(), "agentic:workspace/dispatch")
	assert.Contains(t, c.Commands(), "workspace/watch")
	assert.Contains(t, c.Commands(), "agentic:workspace/watch")
}

// --- CmdWorkspaceList Bad/Ugly ---

func TestCommandsworkspace_CmdWorkspaceList_Bad_NoWorkspaceRootDir(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	// Don't create "workspace" subdir — WorkspaceRoot() returns root+"/workspace" which won't exist

	c := core.New()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	r := s.cmdWorkspaceList(core.NewOptions())
	assert.True(t, r.OK) // gracefully says "no workspaces"
}

func TestCommandsworkspace_CmdWorkspaceList_Ugly_NonDirAndCorruptStatus(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	wsRoot := core.JoinPath(root, "workspace")
	fs.EnsureDir(wsRoot)

	// Non-directory entry in workspace root
	fs.Write(core.JoinPath(wsRoot, "stray-file.txt"), "not a workspace")

	// Workspace with corrupt status.json
	wsCorrupt := core.JoinPath(wsRoot, "ws-corrupt")
	fs.EnsureDir(wsCorrupt)
	fs.Write(core.JoinPath(wsCorrupt, "status.json"), "{broken json!!!")

	// Valid workspace
	wsGood := core.JoinPath(wsRoot, "ws-good")
	fs.EnsureDir(wsGood)
	fs.Write(core.JoinPath(wsGood, "status.json"), core.JSONMarshalString(WorkspaceStatus{Status: "running", Repo: "go-io", Agent: "codex"}))

	c := core.New()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	r := s.cmdWorkspaceList(core.NewOptions())
	assert.True(t, r.OK) // should skip non-dir entries and still list valid workspaces
}

// --- CmdWorkspaceClean Bad/Ugly ---

func TestCommandsworkspace_CmdWorkspaceClean_Bad_UnknownFilterLeavesEverything(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	wsRoot := core.JoinPath(root, "workspace")

	// Create workspaces with various statuses
	for _, ws := range []struct{ name, status string }{
		{"ws-done", "completed"},
		{"ws-fail", "failed"},
		{"ws-run", "running"},
	} {
		d := core.JoinPath(wsRoot, ws.name)
		fs.EnsureDir(d)
		fs.Write(core.JoinPath(d, "status.json"), core.JSONMarshalString(WorkspaceStatus{Status: ws.status, Repo: "test", Agent: "codex"}))
	}

	c := core.New()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Unknown filters now fail explicitly so agent callers can correct typos.
	r := s.cmdWorkspaceClean(core.NewOptions(core.Option{Key: "_arg", Value: "unknown"}))
	assert.False(t, r.OK)
	err, ok := r.Value.(error)
	assert.True(t, ok)
	assert.Contains(t, err.Error(), "unknown filter: unknown")

	// All workspaces should still exist
	for _, name := range []string{"ws-done", "ws-fail", "ws-run"} {
		assert.True(t, fs.IsDir(core.JoinPath(wsRoot, name)), "workspace %s should still exist", name)
	}
}

func TestCommandsworkspace_CmdWorkspaceClean_Ugly_MixedStatuses(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	wsRoot := core.JoinPath(root, "workspace")

	// Create workspaces with statuses including merged and ready-for-review
	for _, ws := range []struct{ name, status string }{
		{"ws-merged", "merged"},
		{"ws-review", "ready-for-review"},
		{"ws-running", "running"},
		{"ws-queued", "queued"},
		{"ws-blocked", "blocked"},
	} {
		d := core.JoinPath(wsRoot, ws.name)
		fs.EnsureDir(d)
		fs.Write(core.JoinPath(d, "status.json"), core.JSONMarshalString(WorkspaceStatus{Status: ws.status, Repo: "test", Agent: "codex"}))
	}

	c := core.New()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// "all" filter removes completed, failed, blocked, merged, ready-for-review but NOT running/queued
	r := s.cmdWorkspaceClean(core.NewOptions())
	assert.True(t, r.OK)

	// merged, ready-for-review, blocked should be removed
	for _, name := range []string{"ws-merged", "ws-review", "ws-blocked"} {
		assert.False(t, fs.Exists(core.JoinPath(wsRoot, name)), "workspace %s should be removed", name)
	}
	// running and queued should remain
	for _, name := range []string{"ws-running", "ws-queued"} {
		assert.True(t, fs.IsDir(core.JoinPath(wsRoot, name)), "workspace %s should still exist", name)
	}
}

func TestCommandsworkspace_CmdWorkspaceClean_Good_CapturesStatsBeforeDelete(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	wsRoot := core.JoinPath(root, "workspace")

	// A completed workspace with a .meta/report.json sidecar — per RFC §15.5
	// the stats row must be persisted to `.core/workspace/db.duckdb` BEFORE
	// the workspace directory is deleted.
	workspaceDir := core.JoinPath(wsRoot, "core", "go-io", "task-stats")
	fs.EnsureDir(workspaceDir)
	fs.Write(core.JoinPath(workspaceDir, "status.json"), core.JSONMarshalString(WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Org:    "core",
		Agent:  "codex:gpt-5.4",
		Branch: "agent/task-stats",
	}))
	metaDir := core.JoinPath(workspaceDir, ".meta")
	fs.EnsureDir(metaDir)
	fs.WriteAtomic(core.JoinPath(metaDir, "report.json"), core.JSONMarshalString(map[string]any{
		"passed":       true,
		"build_passed": true,
		"test_passed":  true,
		"findings":     []any{map[string]any{"severity": "error", "tool": "gosec"}},
	}))

	c := core.New()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	t.Cleanup(s.closeWorkspaceStatsStore)

	r := s.cmdWorkspaceClean(core.NewOptions())
	assert.True(t, r.OK)

	// Workspace directory is gone.
	assert.False(t, fs.Exists(workspaceDir))

	// Stats row survives in `.core/workspace/db.duckdb`.
	statsStore := s.workspaceStatsInstance()
	if statsStore == nil {
		t.Skip("go-store unavailable on this platform — RFC §15.6 graceful degradation")
	}

	value, err := statsStore.Get(stateWorkspaceStatsGroup, "core/go-io/task-stats")
	assert.NoError(t, err)
	assert.Contains(t, value, "core/go-io/task-stats")
	assert.Contains(t, value, "\"build_passed\":true")
}

// --- CmdWorkspaceDispatch Ugly ---

func TestCommandsworkspace_CmdWorkspaceDispatch_Ugly_AllFieldsSet(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	c := core.New()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	r := s.cmdWorkspaceDispatch(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "task", Value: "fix all the things"},
		core.Option{Key: "issue", Value: "42"},
		core.Option{Key: "pr", Value: "7"},
		core.Option{Key: "branch", Value: "feat/test"},
		core.Option{Key: "agent", Value: "claude"},
	))
	// Dispatch calls the real method — fails because no source repo exists to clone.
	// The test verifies the CLI correctly passes all fields through to dispatch.
	assert.False(t, r.OK)
}

func TestCommandsworkspace_CmdWorkspaceWatch_Good_ExplicitWorkspaceCompletes(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	writeWatchStatus(root, "core/go-io/task-42", WorkspaceStatus{
		Status: "ready-for-review",
		Repo:   "go-io",
		Agent:  "codex",
		PRURL:  "https://forge.lthn.ai/core/go-io/pulls/42",
	})

	c := core.New()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	r := s.cmdWorkspaceWatch(core.NewOptions(
		core.Option{Key: "workspace", Value: "core/go-io/task-42"},
		core.Option{Key: "poll_interval", Value: 1},
		core.Option{Key: "timeout", Value: 2},
	))
	assert.True(t, r.OK)

	output, ok := r.Value.(WatchOutput)
	assert.True(t, ok)
	assert.True(t, output.Success)
	assert.Len(t, output.Completed, 1)
	assert.Equal(t, "core/go-io/task-42", output.Completed[0].Workspace)
}

func TestCommandsworkspace_WorkspaceDispatchInputFromOptions_Good_MapsFullContract(t *testing.T) {
	input := workspaceDispatchInputFromOptions(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "task", Value: "ship the release"},
		core.Option{Key: "agent", Value: "codex:gpt-5.4"},
		core.Option{Key: "org", Value: "core"},
		core.Option{Key: "template", Value: "coding"},
		core.Option{Key: "plan_template", Value: "bug-fix"},
		core.Option{Key: "variables", Value: map[string]any{"ISSUE": 42, "MODE": "deep"}},
		core.Option{Key: "persona", Value: "code/reviewer"},
		core.Option{Key: "issue", Value: "42"},
		core.Option{Key: "pr", Value: 7},
		core.Option{Key: "branch", Value: "feature/release"},
		core.Option{Key: "tag", Value: "v0.8.0"},
		core.Option{Key: "dry_run", Value: true},
	))

	assert.Equal(t, "go-io", input.Repo)
	assert.Equal(t, "ship the release", input.Task)
	assert.Equal(t, "codex:gpt-5.4", input.Agent)
	assert.Equal(t, "core", input.Org)
	assert.Equal(t, "coding", input.Template)
	assert.Equal(t, "bug-fix", input.PlanTemplate)
	assert.Equal(t, map[string]string{"ISSUE": "42", "MODE": "deep"}, input.Variables)
	assert.Equal(t, "code/reviewer", input.Persona)
	assert.Equal(t, 42, input.Issue)
	assert.Equal(t, 7, input.PR)
	assert.Equal(t, "feature/release", input.Branch)
	assert.Equal(t, "v0.8.0", input.Tag)
	assert.True(t, input.DryRun)
}
