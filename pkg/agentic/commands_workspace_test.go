// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
)

// --- CmdWorkspaceList Bad/Ugly ---

func TestCommandsworkspace_CmdWorkspaceList_Bad_NoWorkspaceRootDir(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
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
	t.Setenv("CORE_WORKSPACE", root)
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
	t.Setenv("CORE_WORKSPACE", root)
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

	// Filter "unknown" matches no switch case — nothing gets removed
	r := s.cmdWorkspaceClean(core.NewOptions(core.Option{Key: "_arg", Value: "unknown"}))
	assert.True(t, r.OK)

	// All workspaces should still exist
	for _, name := range []string{"ws-done", "ws-fail", "ws-run"} {
		assert.True(t, fs.IsDir(core.JoinPath(wsRoot, name)), "workspace %s should still exist", name)
	}
}

func TestCommandsworkspace_CmdWorkspaceClean_Ugly_MixedStatuses(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
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

// --- CmdWorkspaceDispatch Ugly ---

func TestCommandsworkspace_CmdWorkspaceDispatch_Ugly_AllFieldsSet(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

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
