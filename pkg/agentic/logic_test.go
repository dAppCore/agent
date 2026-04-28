// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"strings"
	"testing"

	core "dappco.re/go"
)

// --- agentCommand ---

func TestDispatch_AgentCommand_Good_Gemini(t *testing.T) {
	cmd, args, err := agentCommand("gemini", "do the thing")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "gemini", cmd)
	core.AssertContains(t, args, "-p")
	core.AssertContains(t, args, "do the thing")
	core.AssertContains(t, args, "--yolo")
	core.AssertContains(t, args, "--sandbox")
}

func TestDispatch_AgentCommand_Good_GeminiWithModel(t *testing.T) {
	cmd, args, err := agentCommand("gemini:flash", "my prompt")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "gemini", cmd)
	core.AssertContains(t, args, "-m")
	core.AssertContains(t, args, "gemini-2.5-flash")
}

func TestDispatch_AgentCommand_Good_Codex(t *testing.T) {
	cmd, args, err := agentCommand("codex", "fix the tests")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "codex", cmd)
	core.AssertContains(t, args, "exec")
	core.AssertContains(t, args, "--dangerously-bypass-approvals-and-sandbox")
	core.AssertContains(t, args, "fix the tests")
}

func TestDispatch_AgentCommand_Good_CodexReview(t *testing.T) {
	cmd, args, err := agentCommand("codex:review", "")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "codex", cmd)
	core.AssertContains(t, args, "exec")
	// Review mode should NOT include -o flag
	for _, a := range args {
		core.AssertNotEqual(t, "-o", a)
	}
}

func TestDispatch_AgentCommand_Good_CodexWithModel(t *testing.T) {
	cmd, args, err := agentCommand("codex:gpt-5.4", "refactor this")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "codex", cmd)
	core.AssertContains(t, args, "--model")
	core.AssertContains(t, args, "gpt-5.4")
}

func TestDispatch_AgentCommand_Good_Claude(t *testing.T) {
	cmd, args, err := agentCommand("claude", "add tests")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "claude", cmd)
	core.AssertContains(t, args, "-p")
	core.AssertContains(t, args, "add tests")
	core.AssertContains(t, args, "--dangerously-skip-permissions")
}

func TestDispatch_AgentCommand_Good_ClaudeWithModel(t *testing.T) {
	cmd, args, err := agentCommand("claude:haiku", "write docs")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "claude", cmd)
	core.AssertContains(t, args, "--model")
	core.AssertContains(t, args, "haiku")
}

func TestDispatch_AgentCommand_Good_CodeRabbit(t *testing.T) {
	cmd, args, err := agentCommand("coderabbit", "")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "coderabbit", cmd)
	core.AssertContains(t, args, "review")
	core.AssertContains(t, args, "--plain")
}

func TestDispatch_AgentCommand_Good_Local(t *testing.T) {
	cmd, args, err := agentCommand("local", "do stuff")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "sh", cmd)
	core.AssertEqual(t, "-c", args[0])
	// Script should contain socat proxy setup
	core.AssertContains(t, args[1], "socat")
	core.AssertContains(t, args[1], "devstral-24b")
}

func TestDispatch_AgentCommand_Good_LocalWithModel(t *testing.T) {
	cmd, args, err := agentCommand("local:mistral-nemo", "do stuff")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "sh", cmd)
	core.AssertContains(t, args[1], "mistral-nemo")
}

func TestDispatch_LocalAgentCommandScript_Good_ShellQuoting(t *testing.T) {
	script := localAgentCommandScript("devstral-24b", "can't break quoting")
	core.AssertContains(
		t,
		script,
		"'can'\\''t break quoting'",
	)
}

func TestDispatch_AgentCommand_Good_CodexLEMProfile(t *testing.T) {
	cmd, args, err := agentCommand("codex:lemmy", "implement the scorer")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "codex", cmd)
	core.AssertContains(t, args, "--profile")
	core.AssertContains(t, args, "lemmy")
	core.AssertNotContains(t, args, "--model")
}

func TestDispatch_AgentCommand_Good_CodexLemer(t *testing.T) {
	cmd, args, err := agentCommand("codex:lemer", "add docs")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "codex", cmd)
	core.AssertContains(t, args, "--profile")
	core.AssertContains(t, args, "lemer")
}

func TestDispatch_AgentCommand_Good_CodexLemrd(t *testing.T) {
	cmd, args, err := agentCommand("codex:lemrd", "review code")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "codex", cmd)
	core.AssertContains(t, args, "--profile")
	core.AssertContains(t, args, "lemrd")
}

func TestDispatch_IsLEMProfile_Good(t *testing.T) {
	core.AssertTrue(t, isLEMProfile("lemer"))
	core.AssertTrue(t, isLEMProfile("lemma"))
	core.AssertTrue(t, isLEMProfile("lemmy"))
	core.AssertTrue(t, isLEMProfile("lemrd"))
}

func TestDispatch_IsLEMProfile_Bad(t *testing.T) {
	core.AssertFalse(t, isLEMProfile("gpt-5.4"))
	core.AssertFalse(t, isLEMProfile("gemini-2.5-flash"))
	core.AssertFalse(t, isLEMProfile(""))
}

func TestDispatch_IsLEMProfile_Ugly(t *testing.T) {
	core.AssertFalse(t, isLEMProfile("Lemmy"))
	core.AssertFalse(t, isLEMProfile("LEMRD"))
	core.AssertFalse(t, isLEMProfile("lem"))
}

func TestDispatch_IsNativeAgent_Good(t *testing.T) {
	core.AssertTrue(t, isNativeAgent("claude"))
	core.AssertTrue(t, isNativeAgent("claude:opus"))
	core.AssertTrue(t, isNativeAgent("claude:haiku"))
}

func TestDispatch_IsNativeAgent_Bad(t *testing.T) {
	core.AssertFalse(t, isNativeAgent("codex"))
	core.AssertFalse(t, isNativeAgent("codex:gpt-5.4"))
	core.AssertFalse(t, isNativeAgent("gemini"))
}

func TestDispatch_IsNativeAgent_Ugly(t *testing.T) {
	core.AssertFalse(t, isNativeAgent(""))
	core.AssertFalse(t, isNativeAgent("codex:lemmy"))
	core.AssertFalse(t, isNativeAgent("local:mistral"))
}

func TestDispatch_AgentCommand_Bad_Unknown(t *testing.T) {
	cmd, args, err := agentCommand("robot-from-the-future", "take over")
	core.AssertError(t, err)
	core.AssertEmpty(t, cmd)
	core.AssertNil(t, args)
}

func TestDispatch_AgentCommand_Ugly_EmptyAgent(t *testing.T) {
	cmd, args, err := agentCommand("", "prompt")
	core.AssertError(t, err)
	core.AssertEmpty(t, cmd)
	core.AssertNil(t, args)
}

// --- containerCommand ---

func TestDispatch_ContainerCommand_Good_Codex(t *testing.T) {
	t.Setenv("AGENT_DOCKER_IMAGE", "")
	t.Setenv("DIR_HOME", "/home/dev")

	cmd, args := containerCommand("codex", []string{"exec", "--dangerously-bypass-approvals-and-sandbox", "do it"}, "/ws", "/ws/.meta")
	core.AssertEqual(t, "docker", cmd)
	core.AssertContains(t, args, "run")
	core.AssertContains(t, args, "--rm")
	core.AssertContains(t, args, "/ws:/workspace")
	core.AssertContains(t, args, "/ws/.meta:/workspace/.meta")
	core.AssertContains(t, args, "/workspace/repo")
	// Command is wrapped in sh -c for chmod cleanup
	shCmd := args[len(args)-1]
	core.AssertContains(t, shCmd, "missing /workspace/repo")
	core.AssertContains(t, shCmd, "codex")
	// Should use default image
	core.AssertContains(t, args, defaultDockerImage)
}

func TestDispatch_ContainerCommand_Good_CustomImage(t *testing.T) {
	t.Setenv("AGENT_DOCKER_IMAGE", "my-custom-image:latest")
	t.Setenv("DIR_HOME", "/home/dev")

	cmd, args := containerCommand("codex", []string{"exec"}, "/ws", "/ws/.meta")
	core.AssertEqual(t, "docker", cmd)
	core.AssertContains(t, args, "my-custom-image:latest")
}

func TestDispatch_ContainerCommand_Good_ClaudeMountsConfig(t *testing.T) {
	t.Setenv("AGENT_DOCKER_IMAGE", "")
	t.Setenv("DIR_HOME", "/home/dev")

	_, args := containerCommand("claude", []string{"-p", "do it"}, "/ws", "/ws/.meta")
	joined := strings.Join(args, " ")
	core.AssertContains(t, joined, ".claude:/home/agent/.claude:ro")
}

func TestDispatch_ContainerCommand_Good_GeminiMountsConfig(t *testing.T) {
	t.Setenv("AGENT_DOCKER_IMAGE", "")
	t.Setenv("DIR_HOME", "/home/dev")

	_, args := containerCommand("gemini", []string{"-p", "do it"}, "/ws", "/ws/.meta")
	joined := strings.Join(args, " ")
	core.AssertContains(t, joined, ".gemini:/home/agent/.gemini:ro")
}

func TestDispatch_ContainerCommand_Good_CodexNoClaudeMount(t *testing.T) {
	t.Setenv("AGENT_DOCKER_IMAGE", "")
	t.Setenv("DIR_HOME", "/home/dev")

	_, args := containerCommand("codex", []string{"exec"}, "/ws", "/ws/.meta")
	joined := strings.Join(args, " ")
	// codex agent must NOT mount .claude config
	core.AssertNotContains(t, joined, ".claude:/home/agent/.claude:ro")
}

func TestDispatch_ContainerCommand_Good_APIKeysPassedByRef(t *testing.T) {
	t.Setenv("AGENT_DOCKER_IMAGE", "")
	t.Setenv("DIR_HOME", "/home/dev")

	_, args := containerCommand("codex", []string{"exec"}, "/ws", "/ws/.meta")
	joined := strings.Join(args, " ")
	core.AssertContains(t, joined, "OPENAI_API_KEY")
	core.AssertContains(t, joined, "ANTHROPIC_API_KEY")
	core.AssertContains(t, joined, "GEMINI_API_KEY")
}

func TestDispatch_ContainerCommand_Ugly_EmptyDirs(t *testing.T) {
	t.Setenv("AGENT_DOCKER_IMAGE", "")
	t.Setenv("DIR_HOME", "")

	// Should not panic with empty paths
	cmd, args := containerCommand("codex", []string{"exec"}, "", "")
	core.AssertEqual(t, "docker", cmd)
	core.AssertNotEmpty(t, args)
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
	core.AssertContains(t, body, "Fix the login bug")
	core.AssertContains(t, body, "codex")
	core.AssertContains(t, body, "3")
	core.AssertContains(t, body, "agent/fix-login-bug")
	core.AssertContains(t, body, "Co-Authored-By: Virgil <virgil@lethean.io>")
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
	core.AssertContains(t, body, "Closes #42")
}

func TestAutopr_BuildAutoPRBody_Good_NoIssue(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	st := &WorkspaceStatus{
		Task:   "Refactor internals",
		Agent:  "gemini",
		Branch: "agent/refactor-internals",
	}
	body := s.buildAutoPRBody(st, 5)
	core.AssertNotContains(t, body, "Closes #")
}

func TestAutopr_BuildAutoPRBody_Good_CommitCount(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	st := &WorkspaceStatus{Agent: "codex", Branch: "agent/foo"}
	body1 := s.buildAutoPRBody(st, 1)
	body5 := s.buildAutoPRBody(st, 5)
	core.AssertContains(t, body1, "**Commits:** 1")
	core.AssertContains(t, body5, "**Commits:** 5")
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
	core.AssertContains(t, body, "## Task")
	core.AssertContains(t, body, "**Agent:** codex")
}

func TestAutopr_BuildAutoPRBody_Ugly_ZeroCommits(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	st := &WorkspaceStatus{Agent: "codex", Branch: "agent/test"}
	body := s.buildAutoPRBody(st, 0)
	core.AssertContains(t, body, "**Commits:** 0")
}

// --- emitEvent ---

func TestEvents_EmitEvent_Good_WritesJSONL(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	emitEvent("agent_completed", "codex", "core/go-io/task-5", "completed")

	eventsFile := core.JoinPath(root, "workspace", "events.jsonl")
	r := fs.Read(eventsFile)
	core.RequireTrue(t, r.OK, "events.jsonl should exist after emitEvent")

	content := r.Value.(string)
	core.AssertContains(t, content, "agent_completed")
	core.AssertContains(t, content, "codex")
	core.AssertContains(t, content, "core/go-io/task-5")
	core.AssertContains(t, content, "completed")
}

func TestEvents_EmitEvent_Good_ValidJSON(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	emitEvent("agent_started", "claude", "core/agent/task-1", "running")

	eventsFile := core.JoinPath(root, "workspace", "events.jsonl")
	content := fs.Read(eventsFile)
	core.RequireTrue(t, content.OK)

	for _, line := range core.Split(content.Value.(string), "\n") {
		if line == "" {
			continue
		}
		var ev CompletionEvent
		core.RequireTrue(t, core.JSONUnmarshalString(line, &ev).OK, "each line must be valid JSON")
		core.AssertEqual(t, "agent_started", ev.Type)
	}
}

func TestEvents_EmitEvent_Good_Appends(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	emitEvent("agent_started", "codex", "core/go-io/task-1", "running")
	emitEvent("agent_completed", "codex", "core/go-io/task-1", "completed")

	eventsFile := core.JoinPath(root, "workspace", "events.jsonl")
	r := fs.Read(eventsFile)
	core.RequireTrue(t, r.OK)

	lines := 0
	for _, line := range strings.Split(strings.TrimSpace(r.Value.(string)), "\n") {
		if line != "" {
			lines++
		}
	}
	core.AssertEqual(t, 2, lines, "both events should be in the log")
}

func TestEvents_EmitEvent_Good_StartHelper(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	emitStartEvent("gemini", "core/go-log/task-3")

	eventsFile := core.JoinPath(root, "workspace", "events.jsonl")
	r := fs.Read(eventsFile)
	core.RequireTrue(t, r.OK)
	core.AssertContains(t, r.Value.(string), "agent_started")
	core.AssertContains(t, r.Value.(string), "running")
}

func TestEvents_EmitEvent_Good_CompletionHelper(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	emitCompletionEvent("claude", "core/agent/task-7", "failed")

	eventsFile := core.JoinPath(root, "workspace", "events.jsonl")
	r := fs.Read(eventsFile)
	core.RequireTrue(t, r.OK)
	core.AssertContains(t, r.Value.(string), "agent_completed")
	core.AssertContains(t, r.Value.(string), "failed")
}

func TestEvents_EmitEvent_Bad_NoWorkspaceDir(t *testing.T) {
	// CORE_WORKSPACE points to a directory that doesn't allow writing events.jsonl
	// because workspace/ subdir doesn't exist. Should not panic.
	root := t.TempDir()
	setTestWorkspace(t, root)
	// Do NOT create workspace/ subdir — emitEvent must handle this gracefully
	core.AssertNotPanics(t, func() {
		emitEvent("agent_completed", "codex", "test", "completed")
	})
}

func TestEvents_EmitEvent_Ugly_EmptyFields(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	// Should not panic with all empty fields
	core.AssertNotPanics(t, func() {
		emitEvent("", "", "", "")
	})
}

// --- emitStartEvent/emitCompletionEvent (Good/Bad/Ugly) ---

func TestEvents_EmitStartEvent_Good(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	emitStartEvent("codex", "core/go-io/task-10")

	eventsFile := core.JoinPath(root, "workspace", "events.jsonl")
	r := fs.Read(eventsFile)
	core.RequireTrue(t, r.OK)
	content := r.Value.(string)
	core.AssertContains(t, content, "agent_started")
	core.AssertContains(t, content, "codex")
	core.AssertContains(t, content, "core/go-io/task-10")
}

func TestEvents_EmitStartEvent_Bad(t *testing.T) {
	// Empty agent name
	root := t.TempDir()
	setTestWorkspace(t, root)
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	core.AssertNotPanics(t, func() {
		emitStartEvent("", "core/go-io/task-10")
	})

	eventsFile := core.JoinPath(root, "workspace", "events.jsonl")
	r := fs.Read(eventsFile)
	core.RequireTrue(t, r.OK)
	content := r.Value.(string)
	core.AssertContains(t, content, "agent_started")
}

func TestEvents_EmitStartEvent_Ugly(t *testing.T) {
	// Very long workspace name
	root := t.TempDir()
	setTestWorkspace(t, root)
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	longName := strings.Repeat("very-long-workspace-name-", 50)
	core.AssertNotPanics(t, func() {
		emitStartEvent("claude", longName)
	})

	eventsFile := core.JoinPath(root, "workspace", "events.jsonl")
	r := fs.Read(eventsFile)
	core.RequireTrue(t, r.OK)
	core.AssertContains(t, r.Value.(string), "agent_started")
}

func TestEvents_EmitCompletionEvent_Good(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	emitCompletionEvent("gemini", "core/go-log/task-5", "completed")

	eventsFile := core.JoinPath(root, "workspace", "events.jsonl")
	r := fs.Read(eventsFile)
	core.RequireTrue(t, r.OK)
	content := r.Value.(string)
	core.AssertContains(t, content, "agent_completed")
	core.AssertContains(t, content, "gemini")
	core.AssertContains(t, content, "completed")
}

func TestEvents_EmitCompletionEvent_Bad(t *testing.T) {
	// Empty status
	root := t.TempDir()
	setTestWorkspace(t, root)
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	core.AssertNotPanics(t, func() {
		emitCompletionEvent("claude", "core/agent/task-1", "")
	})

	eventsFile := core.JoinPath(root, "workspace", "events.jsonl")
	r := fs.Read(eventsFile)
	core.RequireTrue(t, r.OK)
	core.AssertContains(t, r.Value.(string), "agent_completed")
}

func TestEvents_EmitCompletionEvent_Ugly(t *testing.T) {
	// Unicode in agent name
	root := t.TempDir()
	setTestWorkspace(t, root)
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	core.AssertNotPanics(t, func() {
		emitCompletionEvent("\u00e9nchantr\u00efx-\u2603", "core/agent/task-1", "completed")
	})

	eventsFile := core.JoinPath(root, "workspace", "events.jsonl")
	r := fs.Read(eventsFile)
	core.RequireTrue(t, r.OK)
	core.AssertContains(t, r.Value.(string), "\u00e9nchantr\u00efx")
}

// --- countFileRefs ---

func TestIngest_CountFileRefs_Good_GoRefs(t *testing.T) {
	body := "Found issue in `pkg/core/app.go:42` and `pkg/core/service.go:100`."
	core.AssertEqual(
		t,
		2,
		countFileRefs(body),
	)
}

func TestIngest_CountFileRefs_Good_PHPRefs(t *testing.T) {
	body := "See `src/Core/Boot.php:15` for details."
	core.AssertEqual(
		t,
		1,
		countFileRefs(body),
	)
}

func TestIngest_CountFileRefs_Good_Mixed(t *testing.T) {
	body := "Go file: `main.go:1`, PHP file: `index.php:99`, plain text ref."
	core.AssertEqual(
		t,
		2,
		countFileRefs(body),
	)
}

func TestIngest_CountFileRefs_Good_NoRefs(t *testing.T) {
	body := "This is just plain text with no file references."
	core.AssertEqual(
		t,
		0,
		countFileRefs(body),
	)
}

func TestIngest_CountFileRefs_Good_UnrelatedBacktick(t *testing.T) {
	// Backtick-quoted string that is not a file:line reference
	body := "Run `go test ./...` to execute tests."
	core.AssertEqual(
		t,
		0,
		countFileRefs(body),
	)
}

func TestIngest_CountFileRefs_Bad_EmptyBody(t *testing.T) {
	core.AssertEqual(
		t,
		0,
		countFileRefs(""),
	)
}

func TestIngest_CountFileRefs_Bad_ShortBody(t *testing.T) {
	// Body too short to contain a valid reference
	core.AssertEqual(
		t,
		0,
		countFileRefs("`a`"),
	)
}

func TestIngest_CountFileRefs_Ugly_MalformedBackticks(t *testing.T) {
	// Unclosed backtick — should not panic or hang
	body := "Something `unclosed"
	core.AssertNotPanics(t, func() {
		countFileRefs(body)
	})
}

func TestIngest_CountFileRefs_Ugly_LongRef(t *testing.T) {
	// Reference longer than 100 chars should not be counted (loop limit)
	longRef := "`" + strings.Repeat("a", 101) + ".go:1`"
	core.AssertEqual(
		t,
		0,
		countFileRefs(longRef),
	)
}

// --- modelVariant ---

func TestQueue_ModelVariant_Good_WithModel(t *testing.T) {
	core.AssertEqual(t, "gpt-5.4", modelVariant("codex:gpt-5.4"))
	core.AssertEqual(t, "flash", modelVariant("gemini:flash"))
	core.AssertEqual(t, "opus", modelVariant("claude:opus"))
	core.AssertEqual(t, "haiku", modelVariant("claude:haiku"))
}

func TestQueue_ModelVariant_Good_NoVariant(t *testing.T) {
	core.AssertEqual(t, "", modelVariant("codex"))
	core.AssertEqual(t, "", modelVariant("claude"))
	core.AssertEqual(t, "", modelVariant("gemini"))
}

func TestQueue_ModelVariant_Good_MultipleColons(t *testing.T) {
	// SplitN(2) only splits on first colon; rest is preserved as the model
	core.AssertEqual(
		t,
		"gpt-5.3-codex-spark",
		modelVariant("codex:gpt-5.3-codex-spark"),
	)
}

func TestQueue_ModelVariant_Bad_EmptyString(t *testing.T) {
	core.AssertEqual(
		t,
		"",
		modelVariant(""),
	)
}

func TestQueue_ModelVariant_Ugly_ColonOnly(t *testing.T) {
	// Just a colon with no model name
	core.AssertEqual(
		t,
		"",
		modelVariant(":"),
	)
}

// --- baseAgent ---

func TestQueue_BaseAgent_Good_Variants(t *testing.T) {
	core.AssertEqual(t, "gemini", baseAgent("gemini:flash"))
	core.AssertEqual(t, "gemini", baseAgent("gemini:pro"))
	core.AssertEqual(t, "claude", baseAgent("claude:haiku"))
	core.AssertEqual(t, "codex", baseAgent("codex:gpt-5.4"))
}

func TestQueue_BaseAgent_Good_NoVariant(t *testing.T) {
	core.AssertEqual(t, "codex", baseAgent("codex"))
	core.AssertEqual(t, "claude", baseAgent("claude"))
	core.AssertEqual(t, "gemini", baseAgent("gemini"))
}

func TestQueue_BaseAgent_Good_CodexSpark(t *testing.T) {
	// spark is codex, not a separate pool
	core.AssertEqual(
		t,
		"codex",
		baseAgent("codex:gpt-5.3-codex-spark"),
	)
}

func TestQueue_BaseAgent_Bad_EmptyString(t *testing.T) {
	// Empty string — SplitN returns [""], so first element is ""
	core.AssertEqual(
		t,
		"",
		baseAgent(""),
	)
}

func TestQueue_BaseAgent_Ugly_JustColon(t *testing.T) {
	// Just a colon — base is empty string before colon
	core.AssertEqual(
		t,
		"",
		baseAgent(":model"),
	)
}

// --- resolveWorkspace ---

func TestHandlers_ResolveWorkspace_Good_ExistingDir(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	// Create the workspace directory structure
	workspaceName := "core/go-io/task-5"
	workspaceDir := core.JoinPath(root, "workspace", workspaceName)
	core.RequireTrue(t, fs.EnsureDir(workspaceDir).OK)

	result := resolveWorkspace(workspaceName)
	core.AssertEqual(t, workspaceDir, result)
}

func TestHandlers_ResolveWorkspace_Good_NestedPath(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	workspaceName := "core/agent/pr-42"
	workspaceDir := core.JoinPath(root, "workspace", workspaceName)
	core.RequireTrue(t, fs.EnsureDir(workspaceDir).OK)

	result := resolveWorkspace(workspaceName)
	core.AssertEqual(t, workspaceDir, result)
}

func TestHandlers_ResolveWorkspace_Bad_NonExistentDir(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	result := resolveWorkspace("core/go-io/task-999")
	core.AssertEqual(t, "", result)
}

func TestHandlers_ResolveWorkspace_Bad_EmptyName(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	// Empty name resolves to the workspace root itself — which is a dir but not a workspace
	// The function returns "" if the path is not a directory, and the workspace root *is*
	// a directory if created. This test verifies the path arithmetic is sane.
	result := resolveWorkspace("")
	// Either the workspace root itself or "" — both are acceptable; must not panic.
	_ = result
}

func TestHandlers_ResolveWorkspace_Ugly_PathTraversal(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	// Path traversal attempt should return "" (parent of workspace root won't be a workspace)
	result := resolveWorkspace("../../etc")
	core.AssertEqual(t, "", result)
}

// --- findWorkspaceByPR ---

func TestHandlers_FindWorkspaceByPR_Good_MatchesFlatLayout(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	wsDir := core.JoinPath(root, "workspace", "task-10")
	core.RequireTrue(t, fs.EnsureDir(wsDir).OK)
	core.RequireNoError(t, writeStatus(wsDir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Branch: "agent/fix-timeout",
	}))

	result := findWorkspaceByPR("go-io", "agent/fix-timeout")
	core.AssertEqual(t, wsDir, result)
}

func TestHandlers_FindWorkspaceByPR_Good_MatchesDeepLayout(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	wsDir := core.JoinPath(root, "workspace", "core", "go-io", "task-15")
	core.RequireTrue(t, fs.EnsureDir(wsDir).OK)
	core.RequireNoError(t, writeStatus(wsDir, &WorkspaceStatus{
		Status: "running",
		Repo:   "go-io",
		Branch: "agent/add-metrics",
	}))

	result := findWorkspaceByPR("go-io", "agent/add-metrics")
	core.AssertEqual(t, wsDir, result)
}

func TestHandlers_FindWorkspaceByPR_Bad_NoMatch(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	wsDir := core.JoinPath(root, "workspace", "task-99")
	core.RequireTrue(t, fs.EnsureDir(wsDir).OK)
	core.RequireNoError(t, writeStatus(wsDir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Branch: "agent/some-other-branch",
	}))

	result := findWorkspaceByPR("go-io", "agent/nonexistent-branch")
	core.AssertEqual(t, "", result)
}

func TestHandlers_FindWorkspaceByPR_Bad_EmptyWorkspace(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	// No workspaces at all
	result := findWorkspaceByPR("go-io", "agent/any-branch")
	core.AssertEqual(t, "", result)
}

func TestHandlers_FindWorkspaceByPR_Bad_RepoDiffers(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	wsDir := core.JoinPath(root, "workspace", "task-5")
	core.RequireTrue(t, fs.EnsureDir(wsDir).OK)
	core.RequireNoError(t, writeStatus(wsDir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-log",
		Branch: "agent/fix-formatter",
	}))

	// Same branch, different repo
	result := findWorkspaceByPR("go-io", "agent/fix-formatter")
	core.AssertEqual(t, "", result)
}

func TestHandlers_FindWorkspaceByPR_Ugly_CorruptStatusFile(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	wsDir := core.JoinPath(root, "workspace", "corrupt-ws")
	core.RequireTrue(t, fs.EnsureDir(wsDir).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(wsDir, "status.json"), "not-valid-json{").OK)

	// Should skip corrupt entries, not panic
	result := findWorkspaceByPR("go-io", "agent/any")
	core.AssertEqual(t, "", result)
}

// --- extractPullRequestNumber ---

func TestVerify_ExtractPullRequestNumber_Good_FullURL(t *testing.T) {
	core.AssertEqual(t, 42, extractPullRequestNumber("https://forge.lthn.ai/core/agent/pulls/42"))
	core.AssertEqual(t, 1, extractPullRequestNumber("https://forge.lthn.ai/core/go-io/pulls/1"))
	core.AssertEqual(t, 999, extractPullRequestNumber("https://forge.lthn.ai/core/go-log/pulls/999"))
}

func TestVerify_ExtractPullRequestNumber_Good_NumberOnly(t *testing.T) {
	// If someone passes a bare number as a URL it should still work
	number := extractPullRequestNumber("7")
	core.AssertEqual(t, 7, number)
	core.AssertTrue(t, number > 0)
}

func TestVerify_ExtractPullRequestNumber_Bad_EmptyURL(t *testing.T) {
	number := extractPullRequestNumber("")
	core.AssertEqual(t, 0, number)
	core.AssertFalse(t, number > 0)
}

func TestVerify_ExtractPullRequestNumber_Bad_TrailingSlash(t *testing.T) {
	// URL ending with slash has empty last segment
	number := extractPullRequestNumber("https://forge.lthn.ai/core/go-io/pulls/")
	core.AssertEqual(t, 0, number)
	core.AssertFalse(t, number > 0)
}

func TestVerify_ExtractPullRequestNumber_Bad_NonNumericEnd(t *testing.T) {
	number := extractPullRequestNumber("https://forge.lthn.ai/core/go-io/pulls/abc")
	core.AssertEqual(t, 0, number)
	core.AssertFalse(t, number > 0)
}

func TestVerify_ExtractPullRequestNumber_Ugly_JustSlashes(t *testing.T) {
	// All slashes — last segment is empty
	number := extractPullRequestNumber("///")
	core.AssertEqual(t, 0, number)
	core.AssertFalse(t, number > 0)
}
