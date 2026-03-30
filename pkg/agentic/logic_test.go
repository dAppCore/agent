// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"strings"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- agentCommand ---

func TestDispatch_AgentCommand_Good_Gemini(t *testing.T) {
	cmd, args, err := agentCommand("gemini", "do the thing")
	require.NoError(t, err)
	assert.Equal(t, "gemini", cmd)
	assert.Contains(t, args, "-p")
	assert.Contains(t, args, "do the thing")
	assert.Contains(t, args, "--yolo")
	assert.Contains(t, args, "--sandbox")
}

func TestDispatch_AgentCommand_Good_GeminiWithModel(t *testing.T) {
	cmd, args, err := agentCommand("gemini:flash", "my prompt")
	require.NoError(t, err)
	assert.Equal(t, "gemini", cmd)
	assert.Contains(t, args, "-m")
	assert.Contains(t, args, "gemini-2.5-flash")
}

func TestDispatch_AgentCommand_Good_Codex(t *testing.T) {
	cmd, args, err := agentCommand("codex", "fix the tests")
	require.NoError(t, err)
	assert.Equal(t, "codex", cmd)
	assert.Contains(t, args, "exec")
	assert.Contains(t, args, "--dangerously-bypass-approvals-and-sandbox")
	assert.Contains(t, args, "fix the tests")
}

func TestDispatch_AgentCommand_Good_CodexReview(t *testing.T) {
	cmd, args, err := agentCommand("codex:review", "")
	require.NoError(t, err)
	assert.Equal(t, "codex", cmd)
	assert.Contains(t, args, "exec")
	// Review mode should NOT include -o flag
	for _, a := range args {
		assert.NotEqual(t, "-o", a)
	}
}

func TestDispatch_AgentCommand_Good_CodexWithModel(t *testing.T) {
	cmd, args, err := agentCommand("codex:gpt-5.4", "refactor this")
	require.NoError(t, err)
	assert.Equal(t, "codex", cmd)
	assert.Contains(t, args, "--model")
	assert.Contains(t, args, "gpt-5.4")
}

func TestDispatch_AgentCommand_Good_Claude(t *testing.T) {
	cmd, args, err := agentCommand("claude", "add tests")
	require.NoError(t, err)
	assert.Equal(t, "claude", cmd)
	assert.Contains(t, args, "-p")
	assert.Contains(t, args, "add tests")
	assert.Contains(t, args, "--dangerously-skip-permissions")
}

func TestDispatch_AgentCommand_Good_ClaudeWithModel(t *testing.T) {
	cmd, args, err := agentCommand("claude:haiku", "write docs")
	require.NoError(t, err)
	assert.Equal(t, "claude", cmd)
	assert.Contains(t, args, "--model")
	assert.Contains(t, args, "haiku")
}

func TestDispatch_AgentCommand_Good_CodeRabbit(t *testing.T) {
	cmd, args, err := agentCommand("coderabbit", "")
	require.NoError(t, err)
	assert.Equal(t, "coderabbit", cmd)
	assert.Contains(t, args, "review")
	assert.Contains(t, args, "--plain")
}

func TestDispatch_AgentCommand_Good_Local(t *testing.T) {
	cmd, args, err := agentCommand("local", "do stuff")
	require.NoError(t, err)
	assert.Equal(t, "sh", cmd)
	assert.Equal(t, "-c", args[0])
	// Script should contain socat proxy setup
	assert.Contains(t, args[1], "socat")
	assert.Contains(t, args[1], "devstral-24b")
}

func TestDispatch_AgentCommand_Good_LocalWithModel(t *testing.T) {
	cmd, args, err := agentCommand("local:mistral-nemo", "do stuff")
	require.NoError(t, err)
	assert.Equal(t, "sh", cmd)
	assert.Contains(t, args[1], "mistral-nemo")
}

func TestDispatch_AgentCommand_Bad_Unknown(t *testing.T) {
	cmd, args, err := agentCommand("robot-from-the-future", "take over")
	assert.Error(t, err)
	assert.Empty(t, cmd)
	assert.Nil(t, args)
}

func TestDispatch_AgentCommand_Ugly_EmptyAgent(t *testing.T) {
	cmd, args, err := agentCommand("", "prompt")
	assert.Error(t, err)
	assert.Empty(t, cmd)
	assert.Nil(t, args)
}

// --- containerCommand ---

func TestDispatch_ContainerCommand_Good_Codex(t *testing.T) {
	t.Setenv("AGENT_DOCKER_IMAGE", "")
	t.Setenv("DIR_HOME", "/home/dev")

	cmd, args := containerCommand("codex", "codex", []string{"exec", "--dangerously-bypass-approvals-and-sandbox", "do it"}, "/ws/repo", "/ws/.meta")
	assert.Equal(t, "docker", cmd)
	assert.Contains(t, args, "run")
	assert.Contains(t, args, "--rm")
	assert.Contains(t, args, "/ws/repo:/workspace")
	assert.Contains(t, args, "/ws/.meta:/workspace/.meta")
	// Command is wrapped in sh -c for chmod cleanup
	shCmd := args[len(args)-1]
	assert.Contains(t, shCmd, "codex")
	// Should use default image
	assert.Contains(t, args, defaultDockerImage)
}

func TestDispatch_ContainerCommand_Good_CustomImage(t *testing.T) {
	t.Setenv("AGENT_DOCKER_IMAGE", "my-custom-image:latest")
	t.Setenv("DIR_HOME", "/home/dev")

	cmd, args := containerCommand("codex", "codex", []string{"exec"}, "/ws/repo", "/ws/.meta")
	assert.Equal(t, "docker", cmd)
	assert.Contains(t, args, "my-custom-image:latest")
}

func TestDispatch_ContainerCommand_Good_ClaudeMountsConfig(t *testing.T) {
	t.Setenv("AGENT_DOCKER_IMAGE", "")
	t.Setenv("DIR_HOME", "/home/dev")

	_, args := containerCommand("claude", "claude", []string{"-p", "do it"}, "/ws/repo", "/ws/.meta")
	joined := strings.Join(args, " ")
	assert.Contains(t, joined, ".claude:/home/dev/.claude:ro")
}

func TestDispatch_ContainerCommand_Good_GeminiMountsConfig(t *testing.T) {
	t.Setenv("AGENT_DOCKER_IMAGE", "")
	t.Setenv("DIR_HOME", "/home/dev")

	_, args := containerCommand("gemini", "gemini", []string{"-p", "do it"}, "/ws/repo", "/ws/.meta")
	joined := strings.Join(args, " ")
	assert.Contains(t, joined, ".gemini:/home/dev/.gemini:ro")
}

func TestDispatch_ContainerCommand_Good_CodexNoClaudeMount(t *testing.T) {
	t.Setenv("AGENT_DOCKER_IMAGE", "")
	t.Setenv("DIR_HOME", "/home/dev")

	_, args := containerCommand("codex", "codex", []string{"exec"}, "/ws/repo", "/ws/.meta")
	joined := strings.Join(args, " ")
	// codex agent must NOT mount .claude config
	assert.NotContains(t, joined, ".claude:/home/dev/.claude:ro")
}

func TestDispatch_ContainerCommand_Good_APIKeysPassedByRef(t *testing.T) {
	t.Setenv("AGENT_DOCKER_IMAGE", "")
	t.Setenv("DIR_HOME", "/home/dev")

	_, args := containerCommand("codex", "codex", []string{"exec"}, "/ws/repo", "/ws/.meta")
	joined := strings.Join(args, " ")
	assert.Contains(t, joined, "OPENAI_API_KEY")
	assert.Contains(t, joined, "ANTHROPIC_API_KEY")
	assert.Contains(t, joined, "GEMINI_API_KEY")
}

func TestDispatch_ContainerCommand_Ugly_EmptyDirs(t *testing.T) {
	t.Setenv("AGENT_DOCKER_IMAGE", "")
	t.Setenv("DIR_HOME", "")

	// Should not panic with empty paths
	cmd, args := containerCommand("codex", "codex", []string{"exec"}, "", "")
	assert.Equal(t, "docker", cmd)
	assert.NotEmpty(t, args)
}

// --- buildAutoPRBody ---

func TestAutopr_BuildAutoPRBody_Good_Basic(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	st := &WorkspaceStatus{
		Task:   "Fix the login bug",
		Agent:  "codex",
		Branch: "agent/fix-login-bug",
	}
	body := s.buildAutoPRBody(st, 3)
	assert.Contains(t, body, "Fix the login bug")
	assert.Contains(t, body, "codex")
	assert.Contains(t, body, "3")
	assert.Contains(t, body, "agent/fix-login-bug")
	assert.Contains(t, body, "Co-Authored-By: Virgil <virgil@lethean.io>")
}

func TestAutopr_BuildAutoPRBody_Good_WithIssue(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	st := &WorkspaceStatus{
		Task:   "Add rate limiting",
		Agent:  "claude",
		Branch: "agent/add-rate-limiting",
		Issue:  42,
	}
	body := s.buildAutoPRBody(st, 1)
	assert.Contains(t, body, "Closes #42")
}

func TestAutopr_BuildAutoPRBody_Good_NoIssue(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	st := &WorkspaceStatus{
		Task:   "Refactor internals",
		Agent:  "gemini",
		Branch: "agent/refactor-internals",
	}
	body := s.buildAutoPRBody(st, 5)
	assert.NotContains(t, body, "Closes #")
}

func TestAutopr_BuildAutoPRBody_Good_CommitCount(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	st := &WorkspaceStatus{Agent: "codex", Branch: "agent/foo"}
	body1 := s.buildAutoPRBody(st, 1)
	body5 := s.buildAutoPRBody(st, 5)
	assert.Contains(t, body1, "**Commits:** 1")
	assert.Contains(t, body5, "**Commits:** 5")
}

func TestAutopr_BuildAutoPRBody_Bad_EmptyTask(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	st := &WorkspaceStatus{
		Task:   "",
		Agent:  "codex",
		Branch: "agent/something",
	}
	// Should not panic; body should still have the structure
	body := s.buildAutoPRBody(st, 0)
	assert.Contains(t, body, "## Task")
	assert.Contains(t, body, "**Agent:** codex")
}

func TestAutopr_BuildAutoPRBody_Ugly_ZeroCommits(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	st := &WorkspaceStatus{Agent: "codex", Branch: "agent/test"}
	body := s.buildAutoPRBody(st, 0)
	assert.Contains(t, body, "**Commits:** 0")
}

// --- emitEvent ---

func TestEvents_EmitEvent_Good_WritesJSONL(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	require.True(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	emitEvent("agent_completed", "codex", "core/go-io/task-5", "completed")

	eventsFile := core.JoinPath(root, "workspace", "events.jsonl")
	r := fs.Read(eventsFile)
	require.True(t, r.OK, "events.jsonl should exist after emitEvent")

	content := r.Value.(string)
	assert.Contains(t, content, "agent_completed")
	assert.Contains(t, content, "codex")
	assert.Contains(t, content, "core/go-io/task-5")
	assert.Contains(t, content, "completed")
}

func TestEvents_EmitEvent_Good_ValidJSON(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	require.True(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	emitEvent("agent_started", "claude", "core/agent/task-1", "running")

	eventsFile := core.JoinPath(root, "workspace", "events.jsonl")
	content := fs.Read(eventsFile)
	require.True(t, content.OK)

	for _, line := range core.Split(content.Value.(string), "\n") {
		if line == "" {
			continue
		}
		var ev CompletionEvent
		require.True(t, core.JSONUnmarshalString(line, &ev).OK, "each line must be valid JSON")
		assert.Equal(t, "agent_started", ev.Type)
	}
}

func TestEvents_EmitEvent_Good_Appends(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	require.True(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	emitEvent("agent_started", "codex", "core/go-io/task-1", "running")
	emitEvent("agent_completed", "codex", "core/go-io/task-1", "completed")

	eventsFile := core.JoinPath(root, "workspace", "events.jsonl")
	r := fs.Read(eventsFile)
	require.True(t, r.OK)

	lines := 0
	for _, line := range strings.Split(strings.TrimSpace(r.Value.(string)), "\n") {
		if line != "" {
			lines++
		}
	}
	assert.Equal(t, 2, lines, "both events should be in the log")
}

func TestEvents_EmitEvent_Good_StartHelper(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	require.True(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	emitStartEvent("gemini", "core/go-log/task-3")

	eventsFile := core.JoinPath(root, "workspace", "events.jsonl")
	r := fs.Read(eventsFile)
	require.True(t, r.OK)
	assert.Contains(t, r.Value.(string), "agent_started")
	assert.Contains(t, r.Value.(string), "running")
}

func TestEvents_EmitEvent_Good_CompletionHelper(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	require.True(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	emitCompletionEvent("claude", "core/agent/task-7", "failed")

	eventsFile := core.JoinPath(root, "workspace", "events.jsonl")
	r := fs.Read(eventsFile)
	require.True(t, r.OK)
	assert.Contains(t, r.Value.(string), "agent_completed")
	assert.Contains(t, r.Value.(string), "failed")
}

func TestEvents_EmitEvent_Bad_NoWorkspaceDir(t *testing.T) {
	// CORE_WORKSPACE points to a directory that doesn't allow writing events.jsonl
	// because workspace/ subdir doesn't exist. Should not panic.
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	// Do NOT create workspace/ subdir — emitEvent must handle this gracefully
	assert.NotPanics(t, func() {
		emitEvent("agent_completed", "codex", "test", "completed")
	})
}

func TestEvents_EmitEvent_Ugly_EmptyFields(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	require.True(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	// Should not panic with all empty fields
	assert.NotPanics(t, func() {
		emitEvent("", "", "", "")
	})
}

// --- emitStartEvent/emitCompletionEvent (Good/Bad/Ugly) ---

func TestEvents_EmitStartEvent_Good(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	require.True(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	emitStartEvent("codex", "core/go-io/task-10")

	eventsFile := core.JoinPath(root, "workspace", "events.jsonl")
	r := fs.Read(eventsFile)
	require.True(t, r.OK)
	content := r.Value.(string)
	assert.Contains(t, content, "agent_started")
	assert.Contains(t, content, "codex")
	assert.Contains(t, content, "core/go-io/task-10")
}

func TestEvents_EmitStartEvent_Bad(t *testing.T) {
	// Empty agent name
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	require.True(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	assert.NotPanics(t, func() {
		emitStartEvent("", "core/go-io/task-10")
	})

	eventsFile := core.JoinPath(root, "workspace", "events.jsonl")
	r := fs.Read(eventsFile)
	require.True(t, r.OK)
	content := r.Value.(string)
	assert.Contains(t, content, "agent_started")
}

func TestEvents_EmitStartEvent_Ugly(t *testing.T) {
	// Very long workspace name
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	require.True(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	longName := strings.Repeat("very-long-workspace-name-", 50)
	assert.NotPanics(t, func() {
		emitStartEvent("claude", longName)
	})

	eventsFile := core.JoinPath(root, "workspace", "events.jsonl")
	r := fs.Read(eventsFile)
	require.True(t, r.OK)
	assert.Contains(t, r.Value.(string), "agent_started")
}

func TestEvents_EmitCompletionEvent_Good(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	require.True(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	emitCompletionEvent("gemini", "core/go-log/task-5", "completed")

	eventsFile := core.JoinPath(root, "workspace", "events.jsonl")
	r := fs.Read(eventsFile)
	require.True(t, r.OK)
	content := r.Value.(string)
	assert.Contains(t, content, "agent_completed")
	assert.Contains(t, content, "gemini")
	assert.Contains(t, content, "completed")
}

func TestEvents_EmitCompletionEvent_Bad(t *testing.T) {
	// Empty status
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	require.True(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	assert.NotPanics(t, func() {
		emitCompletionEvent("claude", "core/agent/task-1", "")
	})

	eventsFile := core.JoinPath(root, "workspace", "events.jsonl")
	r := fs.Read(eventsFile)
	require.True(t, r.OK)
	assert.Contains(t, r.Value.(string), "agent_completed")
}

func TestEvents_EmitCompletionEvent_Ugly(t *testing.T) {
	// Unicode in agent name
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	require.True(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	assert.NotPanics(t, func() {
		emitCompletionEvent("\u00e9nchantr\u00efx-\u2603", "core/agent/task-1", "completed")
	})

	eventsFile := core.JoinPath(root, "workspace", "events.jsonl")
	r := fs.Read(eventsFile)
	require.True(t, r.OK)
	assert.Contains(t, r.Value.(string), "\u00e9nchantr\u00efx")
}

// --- countFileRefs ---

func TestIngest_CountFileRefs_Good_GoRefs(t *testing.T) {
	body := "Found issue in `pkg/core/app.go:42` and `pkg/core/service.go:100`."
	assert.Equal(t, 2, countFileRefs(body))
}

func TestIngest_CountFileRefs_Good_PHPRefs(t *testing.T) {
	body := "See `src/Core/Boot.php:15` for details."
	assert.Equal(t, 1, countFileRefs(body))
}

func TestIngest_CountFileRefs_Good_Mixed(t *testing.T) {
	body := "Go file: `main.go:1`, PHP file: `index.php:99`, plain text ref."
	assert.Equal(t, 2, countFileRefs(body))
}

func TestIngest_CountFileRefs_Good_NoRefs(t *testing.T) {
	body := "This is just plain text with no file references."
	assert.Equal(t, 0, countFileRefs(body))
}

func TestIngest_CountFileRefs_Good_UnrelatedBacktick(t *testing.T) {
	// Backtick-quoted string that is not a file:line reference
	body := "Run `go test ./...` to execute tests."
	assert.Equal(t, 0, countFileRefs(body))
}

func TestIngest_CountFileRefs_Bad_EmptyBody(t *testing.T) {
	assert.Equal(t, 0, countFileRefs(""))
}

func TestIngest_CountFileRefs_Bad_ShortBody(t *testing.T) {
	// Body too short to contain a valid reference
	assert.Equal(t, 0, countFileRefs("`a`"))
}

func TestIngest_CountFileRefs_Ugly_MalformedBackticks(t *testing.T) {
	// Unclosed backtick — should not panic or hang
	body := "Something `unclosed"
	assert.NotPanics(t, func() {
		countFileRefs(body)
	})
}

func TestIngest_CountFileRefs_Ugly_LongRef(t *testing.T) {
	// Reference longer than 100 chars should not be counted (loop limit)
	longRef := "`" + strings.Repeat("a", 101) + ".go:1`"
	assert.Equal(t, 0, countFileRefs(longRef))
}

// --- modelVariant ---

func TestQueue_ModelVariant_Good_WithModel(t *testing.T) {
	assert.Equal(t, "gpt-5.4", modelVariant("codex:gpt-5.4"))
	assert.Equal(t, "flash", modelVariant("gemini:flash"))
	assert.Equal(t, "opus", modelVariant("claude:opus"))
	assert.Equal(t, "haiku", modelVariant("claude:haiku"))
}

func TestQueue_ModelVariant_Good_NoVariant(t *testing.T) {
	assert.Equal(t, "", modelVariant("codex"))
	assert.Equal(t, "", modelVariant("claude"))
	assert.Equal(t, "", modelVariant("gemini"))
}

func TestQueue_ModelVariant_Good_MultipleColons(t *testing.T) {
	// SplitN(2) only splits on first colon; rest is preserved as the model
	assert.Equal(t, "gpt-5.3-codex-spark", modelVariant("codex:gpt-5.3-codex-spark"))
}

func TestQueue_ModelVariant_Bad_EmptyString(t *testing.T) {
	assert.Equal(t, "", modelVariant(""))
}

func TestQueue_ModelVariant_Ugly_ColonOnly(t *testing.T) {
	// Just a colon with no model name
	assert.Equal(t, "", modelVariant(":"))
}

// --- baseAgent ---

func TestQueue_BaseAgent_Good_Variants(t *testing.T) {
	assert.Equal(t, "gemini", baseAgent("gemini:flash"))
	assert.Equal(t, "gemini", baseAgent("gemini:pro"))
	assert.Equal(t, "claude", baseAgent("claude:haiku"))
	assert.Equal(t, "codex", baseAgent("codex:gpt-5.4"))
}

func TestQueue_BaseAgent_Good_NoVariant(t *testing.T) {
	assert.Equal(t, "codex", baseAgent("codex"))
	assert.Equal(t, "claude", baseAgent("claude"))
	assert.Equal(t, "gemini", baseAgent("gemini"))
}

func TestQueue_BaseAgent_Good_CodexSpark(t *testing.T) {
	// spark is codex, not a separate pool
	assert.Equal(t, "codex", baseAgent("codex:gpt-5.3-codex-spark"))
}

func TestQueue_BaseAgent_Bad_EmptyString(t *testing.T) {
	// Empty string — SplitN returns [""], so first element is ""
	assert.Equal(t, "", baseAgent(""))
}

func TestQueue_BaseAgent_Ugly_JustColon(t *testing.T) {
	// Just a colon — base is empty string before colon
	assert.Equal(t, "", baseAgent(":model"))
}

// --- resolveWorkspace ---

func TestHandlers_ResolveWorkspace_Good_ExistingDir(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	// Create the workspace directory structure
	workspaceName := "core/go-io/task-5"
	workspaceDir := core.JoinPath(root, "workspace", workspaceName)
	require.True(t, fs.EnsureDir(workspaceDir).OK)

	result := resolveWorkspace(workspaceName)
	assert.Equal(t, workspaceDir, result)
}

func TestHandlers_ResolveWorkspace_Good_NestedPath(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	workspaceName := "core/agent/pr-42"
	workspaceDir := core.JoinPath(root, "workspace", workspaceName)
	require.True(t, fs.EnsureDir(workspaceDir).OK)

	result := resolveWorkspace(workspaceName)
	assert.Equal(t, workspaceDir, result)
}

func TestHandlers_ResolveWorkspace_Bad_NonExistentDir(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	result := resolveWorkspace("core/go-io/task-999")
	assert.Equal(t, "", result)
}

func TestHandlers_ResolveWorkspace_Bad_EmptyName(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	// Empty name resolves to the workspace root itself — which is a dir but not a workspace
	// The function returns "" if the path is not a directory, and the workspace root *is*
	// a directory if created. This test verifies the path arithmetic is sane.
	result := resolveWorkspace("")
	// Either the workspace root itself or "" — both are acceptable; must not panic.
	_ = result
}

func TestHandlers_ResolveWorkspace_Ugly_PathTraversal(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	// Path traversal attempt should return "" (parent of workspace root won't be a workspace)
	result := resolveWorkspace("../../etc")
	assert.Equal(t, "", result)
}

// --- findWorkspaceByPR ---

func TestHandlers_FindWorkspaceByPR_Good_MatchesFlatLayout(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	wsDir := core.JoinPath(root, "workspace", "task-10")
	require.True(t, fs.EnsureDir(wsDir).OK)
	require.NoError(t, writeStatus(wsDir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Branch: "agent/fix-timeout",
	}))

	result := findWorkspaceByPR("go-io", "agent/fix-timeout")
	assert.Equal(t, wsDir, result)
}

func TestHandlers_FindWorkspaceByPR_Good_MatchesDeepLayout(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	wsDir := core.JoinPath(root, "workspace", "core", "go-io", "task-15")
	require.True(t, fs.EnsureDir(wsDir).OK)
	require.NoError(t, writeStatus(wsDir, &WorkspaceStatus{
		Status: "running",
		Repo:   "go-io",
		Branch: "agent/add-metrics",
	}))

	result := findWorkspaceByPR("go-io", "agent/add-metrics")
	assert.Equal(t, wsDir, result)
}

func TestHandlers_FindWorkspaceByPR_Bad_NoMatch(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	wsDir := core.JoinPath(root, "workspace", "task-99")
	require.True(t, fs.EnsureDir(wsDir).OK)
	require.NoError(t, writeStatus(wsDir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Branch: "agent/some-other-branch",
	}))

	result := findWorkspaceByPR("go-io", "agent/nonexistent-branch")
	assert.Equal(t, "", result)
}

func TestHandlers_FindWorkspaceByPR_Bad_EmptyWorkspace(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	// No workspaces at all
	result := findWorkspaceByPR("go-io", "agent/any-branch")
	assert.Equal(t, "", result)
}

func TestHandlers_FindWorkspaceByPR_Bad_RepoDiffers(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	wsDir := core.JoinPath(root, "workspace", "task-5")
	require.True(t, fs.EnsureDir(wsDir).OK)
	require.NoError(t, writeStatus(wsDir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-log",
		Branch: "agent/fix-formatter",
	}))

	// Same branch, different repo
	result := findWorkspaceByPR("go-io", "agent/fix-formatter")
	assert.Equal(t, "", result)
}

func TestHandlers_FindWorkspaceByPR_Ugly_CorruptStatusFile(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	wsDir := core.JoinPath(root, "workspace", "corrupt-ws")
	require.True(t, fs.EnsureDir(wsDir).OK)
	require.True(t, fs.Write(core.JoinPath(wsDir, "status.json"), "not-valid-json{").OK)

	// Should skip corrupt entries, not panic
	result := findWorkspaceByPR("go-io", "agent/any")
	assert.Equal(t, "", result)
}

// --- extractPullRequestNumber ---

func TestVerify_ExtractPullRequestNumber_Good_FullURL(t *testing.T) {
	assert.Equal(t, 42, extractPullRequestNumber("https://forge.lthn.ai/core/agent/pulls/42"))
	assert.Equal(t, 1, extractPullRequestNumber("https://forge.lthn.ai/core/go-io/pulls/1"))
	assert.Equal(t, 999, extractPullRequestNumber("https://forge.lthn.ai/core/go-log/pulls/999"))
}

func TestVerify_ExtractPullRequestNumber_Good_NumberOnly(t *testing.T) {
	// If someone passes a bare number as a URL it should still work
	assert.Equal(t, 7, extractPullRequestNumber("7"))
}

func TestVerify_ExtractPullRequestNumber_Bad_EmptyURL(t *testing.T) {
	assert.Equal(t, 0, extractPullRequestNumber(""))
}

func TestVerify_ExtractPullRequestNumber_Bad_TrailingSlash(t *testing.T) {
	// URL ending with slash has empty last segment
	assert.Equal(t, 0, extractPullRequestNumber("https://forge.lthn.ai/core/go-io/pulls/"))
}

func TestVerify_ExtractPullRequestNumber_Bad_NonNumericEnd(t *testing.T) {
	assert.Equal(t, 0, extractPullRequestNumber("https://forge.lthn.ai/core/go-io/pulls/abc"))
}

func TestVerify_ExtractPullRequestNumber_Ugly_JustSlashes(t *testing.T) {
	// All slashes — last segment is empty
	assert.Equal(t, 0, extractPullRequestNumber("///"))
}
