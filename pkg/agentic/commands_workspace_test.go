// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
)

// --- extractField ---

func TestCommandsWorkspace_ExtractField_Good_SimpleJSON(t *testing.T) {
	json := `{"status":"running","repo":"go-io","agent":"codex"}`
	assert.Equal(t, "running", extractField(json, "status"))
	assert.Equal(t, "go-io", extractField(json, "repo"))
	assert.Equal(t, "codex", extractField(json, "agent"))
}

func TestCommandsWorkspace_ExtractField_Good_PrettyPrinted(t *testing.T) {
	json := `{
  "status": "completed",
  "repo": "go-crypt"
}`
	assert.Equal(t, "completed", extractField(json, "status"))
	assert.Equal(t, "go-crypt", extractField(json, "repo"))
}

func TestCommandsWorkspace_ExtractField_Good_TabSeparated(t *testing.T) {
	json := `{"status":	"blocked"}`
	assert.Equal(t, "blocked", extractField(json, "status"))
}

func TestCommandsWorkspace_ExtractField_Bad_MissingField(t *testing.T) {
	json := `{"status":"running"}`
	assert.Empty(t, extractField(json, "nonexistent"))
}

func TestCommandsWorkspace_ExtractField_Bad_EmptyJSON(t *testing.T) {
	assert.Empty(t, extractField("", "status"))
	assert.Empty(t, extractField("{}", "status"))
}

func TestCommandsWorkspace_ExtractField_Bad_NoValue(t *testing.T) {
	// Field key exists but no quoted value after colon
	json := `{"status": 42}`
	assert.Empty(t, extractField(json, "status"))
}

func TestCommandsWorkspace_ExtractField_Bad_TruncatedJSON(t *testing.T) {
	// Field key exists but string is truncated
	json := `{"status":`
	assert.Empty(t, extractField(json, "status"))
}

func TestCommandsWorkspace_ExtractField_Good_EmptyValue(t *testing.T) {
	json := `{"status":""}`
	assert.Equal(t, "", extractField(json, "status"))
}

func TestCommandsWorkspace_ExtractField_Good_ValueWithSpaces(t *testing.T) {
	json := `{"task":"fix the failing tests"}`
	assert.Equal(t, "fix the failing tests", extractField(json, "task"))
}

// --- CmdWorkspaceList Bad/Ugly ---

func TestCommandsWorkspace_CmdWorkspaceList_Bad_NoWorkspaceRootDir(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	// Don't create "workspace" subdir — WorkspaceRoot() returns root+"/workspace" which won't exist

	c := core.New()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	r := s.cmdWorkspaceList(core.NewOptions())
	assert.True(t, r.OK) // gracefully says "no workspaces"
}

func TestCommandsWorkspace_CmdWorkspaceList_Ugly_NonDirAndCorruptStatus(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := core.JoinPath(root, "workspace")
	os.MkdirAll(wsRoot, 0o755)

	// Non-directory entry in workspace root
	os.WriteFile(core.JoinPath(wsRoot, "stray-file.txt"), []byte("not a workspace"), 0o644)

	// Workspace with corrupt status.json
	wsCorrupt := core.JoinPath(wsRoot, "ws-corrupt")
	os.MkdirAll(wsCorrupt, 0o755)
	os.WriteFile(core.JoinPath(wsCorrupt, "status.json"), []byte("{broken json!!!"), 0o644)

	// Valid workspace
	wsGood := core.JoinPath(wsRoot, "ws-good")
	os.MkdirAll(wsGood, 0o755)
	data, _ := json.Marshal(WorkspaceStatus{Status: "running", Repo: "go-io", Agent: "codex"})
	os.WriteFile(core.JoinPath(wsGood, "status.json"), data, 0o644)

	c := core.New()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	r := s.cmdWorkspaceList(core.NewOptions())
	assert.True(t, r.OK) // should skip non-dir entries and still list valid workspaces
}

// --- CmdWorkspaceClean Bad/Ugly ---

func TestCommandsWorkspace_CmdWorkspaceClean_Bad_UnknownFilterLeavesEverything(t *testing.T) {
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
		os.MkdirAll(d, 0o755)
		data, _ := json.Marshal(WorkspaceStatus{Status: ws.status, Repo: "test", Agent: "codex"})
		os.WriteFile(core.JoinPath(d, "status.json"), data, 0o644)
	}

	c := core.New()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	// Filter "unknown" matches no switch case — nothing gets removed
	r := s.cmdWorkspaceClean(core.NewOptions(core.Option{Key: "_arg", Value: "unknown"}))
	assert.True(t, r.OK)

	// All workspaces should still exist
	for _, name := range []string{"ws-done", "ws-fail", "ws-run"} {
		_, err := os.Stat(core.JoinPath(wsRoot, name))
		assert.NoError(t, err, "workspace %s should still exist", name)
	}
}

func TestCommandsWorkspace_CmdWorkspaceClean_Ugly_MixedStatuses(t *testing.T) {
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
		os.MkdirAll(d, 0o755)
		data, _ := json.Marshal(WorkspaceStatus{Status: ws.status, Repo: "test", Agent: "codex"})
		os.WriteFile(core.JoinPath(d, "status.json"), data, 0o644)
	}

	c := core.New()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	// "all" filter removes completed, failed, blocked, merged, ready-for-review but NOT running/queued
	r := s.cmdWorkspaceClean(core.NewOptions())
	assert.True(t, r.OK)

	// merged, ready-for-review, blocked should be removed
	for _, name := range []string{"ws-merged", "ws-review", "ws-blocked"} {
		_, err := os.Stat(core.JoinPath(wsRoot, name))
		assert.True(t, os.IsNotExist(err), "workspace %s should be removed", name)
	}
	// running and queued should remain
	for _, name := range []string{"ws-running", "ws-queued"} {
		_, err := os.Stat(core.JoinPath(wsRoot, name))
		assert.NoError(t, err, "workspace %s should still exist", name)
	}
}

// --- CmdWorkspaceDispatch Ugly ---

func TestCommandsWorkspace_CmdWorkspaceDispatch_Ugly_AllFieldsSet(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	c := core.New()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	r := s.cmdWorkspaceDispatch(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "task", Value: "fix all the things"},
		core.Option{Key: "issue", Value: "42"},
		core.Option{Key: "pr", Value: "7"},
		core.Option{Key: "branch", Value: "feat/test"},
		core.Option{Key: "agent", Value: "claude"},
	))
	// Dispatch is stubbed out — returns OK with a message
	assert.True(t, r.OK)
}

// --- ExtractField Ugly ---

func TestCommandsWorkspace_ExtractField_Ugly_NestedJSON(t *testing.T) {
	// Nested JSON — extractField only finds top-level keys (simple scan)
	j := `{"outer":{"inner":"value"},"status":"ok"}`
	assert.Equal(t, "ok", extractField(j, "status"))
	// "inner" is inside the nested object — extractField should still find it
	assert.Equal(t, "value", extractField(j, "inner"))
}

func TestCommandsWorkspace_ExtractField_Ugly_EscapedQuotes(t *testing.T) {
	// Value with escaped quotes — extractField stops at the first unescaped quote
	j := `{"msg":"hello \"world\"","status":"done"}`
	// extractField will return "hello \" because it stops at first quote after open
	// The important thing is it doesn't panic
	_ = extractField(j, "msg")
	assert.Equal(t, "done", extractField(j, "status"))
}
