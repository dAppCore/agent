# OpenCode Local Harness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an OpenCode-based local coding harness runner so CoreAgent can dispatch Gemma/Qwen local models with file, shell, and LSP tool access.

**Architecture:** CoreAgent keeps owning workspace prep, queueing, process supervision, status files, and logs. The new `opencode:<profile>` runner executes OpenCode in non-interactive mode on the host, using inline `OPENCODE_CONFIG_CONTENT` to point OpenCode at a local OpenAI-compatible endpoint such as vLLM Metal. The first pass only resolves profile configuration and process arguments; vLLM launch management remains external.

**Tech Stack:** Go, CoreAgent dispatch runner, OpenCode CLI, OpenAI-compatible local model servers.

---

### File Structure

- Modify `go/pkg/agentic/dispatch.go`: recognise `opencode` as a native runner and route `opencode:<profile>` through the new command helper.
- Create `go/pkg/agentic/opencode.go`: profile defaults, environment overrides, inline OpenCode JSON config, and shell command assembly.
- Create `go/pkg/agentic/opencode_test.go`: focused Good/Bad/Ugly tests for profile resolution and command generation.
- Modify `go/pkg/agentic/logic_test.go`: add one dispatch-level test proving `agentCommand("opencode:gemma4-agentic", prompt)` returns a host OpenCode command.

### Task 1: Profile Resolution Tests

- [ ] **Step 1: Write failing tests**

Create `go/pkg/agentic/opencode_test.go` with tests that expect:

```go
profile := opencodeProfileConfig("gemma4-agentic")
core.AssertEqual(t, "core-local", profile.Provider)
core.AssertEqual(t, "http://127.0.0.1:8001/v1", profile.BaseURL)
core.AssertEqual(t, "google/gemma-4-26B-A4B-it", profile.Model)
```

Also test environment overrides:

```go
t.Setenv("CORE_OPENCODE_GEMMA4_AGENTIC_BASE_URL", "http://127.0.0.1:9001/v1")
t.Setenv("CORE_OPENCODE_GEMMA4_AGENTIC_MODEL", "lthn/lemma-gemma-4-26b")
profile := opencodeProfileConfig("gemma4-agentic")
core.AssertEqual(t, "http://127.0.0.1:9001/v1", profile.BaseURL)
core.AssertEqual(t, "lthn/lemma-gemma-4-26b", profile.Model)
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./go/pkg/agentic -run 'TestOpenCode_Profile' -count=1`

Expected: compile failure because `opencodeProfileConfig` does not exist.

- [ ] **Step 3: Implement profile resolution**

Create `opencode.go` with:

```go
type opencodeProfile struct {
    Provider string
    BaseURL string
    Model string
    SmallModel string
    Agent string
}
```

Implement `opencodeProfileConfig(profile string) opencodeProfile` with defaults for `gemma4-agentic`, `gemma4-xhigh`, `gemma4-chatter`, `gemma4-e4b`, and `qwen36`, plus `CORE_OPENCODE_<PROFILE>_{PROVIDER,BASE_URL,MODEL,SMALL_MODEL,AGENT}` overrides.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./go/pkg/agentic -run 'TestOpenCode_Profile' -count=1`

Expected: PASS.

### Task 2: OpenCode Command Tests

- [ ] **Step 1: Write failing tests**

Extend `opencode_test.go` with tests that expect:

```go
script := opencodeAgentCommandScript("gemma4-agentic", "fix tests")
core.AssertContains(t, script, "OPENCODE_CONFIG_CONTENT=")
core.AssertContains(t, script, "opencode run")
core.AssertContains(t, script, "--dangerously-skip-permissions")
core.AssertContains(t, script, "--model")
core.AssertContains(t, script, "core-local/google/gemma-4-26B-A4B-it")
core.AssertContains(t, script, "'fix tests'")
```

Add a shell quoting test:

```go
script := opencodeAgentCommandScript("gemma4-agentic", "can't break")
core.AssertContains(t, script, "'can'\\''t break'")
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./go/pkg/agentic -run 'TestOpenCode_Command' -count=1`

Expected: compile failure because `opencodeAgentCommandScript` does not exist.

- [ ] **Step 3: Implement command generation**

Add `opencodeAgentCommandScript(profile, prompt string) string`. It should build inline OpenCode config with provider `npm: "@ai-sdk/openai-compatible"`, `options.baseURL`, `options.apiKey: "sk-local"`, `model`, `small_model`, `tools` enabled, and `permission` entries allowing edit/bash/read/grep/glob/lsp for non-interactive CoreAgent runs.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./go/pkg/agentic -run 'TestOpenCode_Command' -count=1`

Expected: PASS.

### Task 3: Dispatch Integration

- [ ] **Step 1: Write failing dispatch test**

Modify `go/pkg/agentic/logic_test.go` with:

```go
func TestDispatch_AgentCommand_Good_OpenCodeGemma(t *testing.T) {
    cmd, args, err := agentCommand("opencode:gemma4-agentic", "fix it")
    core.RequireNoError(t, err)
    core.AssertEqual(t, "sh", cmd)
    core.AssertEqual(t, "-c", args[0])
    core.AssertContains(t, args[1], "opencode run")
    core.AssertContains(t, args[1], "core-local/google/gemma-4-26B-A4B-it")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./go/pkg/agentic -run 'TestDispatch_AgentCommand_Good_OpenCodeGemma' -count=1`

Expected: failure with `unknown agent: opencode:gemma4-agentic`.

- [ ] **Step 3: Implement dispatch integration**

Modify `agentCommandResult` in `dispatch.go` to add `case "opencode":` returning `sh -c opencodeAgentCommandScript(profile, prompt)`. Modify `isNativeAgent` so `opencode` runs on the host rather than inside the container.

- [ ] **Step 4: Run focused tests**

Run: `go test ./go/pkg/agentic -run 'Test(OpenCode|Dispatch_AgentCommand_Good_OpenCode|Dispatch_IsNativeAgent)' -count=1`

Expected: PASS.

### Task 4: Package Verification

- [ ] **Step 1: Run agentic package tests**

Run: `go test ./go/pkg/agentic -count=1`

Expected: PASS or clearly identified pre-existing failures.

- [ ] **Step 2: Run runner package tests**

Run: `go test ./go/pkg/runner -count=1`

Expected: PASS or clearly identified pre-existing failures.

### Self-Review

- Spec coverage: OpenCode harness profile support, direct local endpoint config, and host-native dispatch are covered. vLLM process launch, health checks, and direct `/v1/chat/completions` provider calls are intentionally out of scope for this first pass.
- Placeholder scan: no deferred implementation placeholders remain.
- Type consistency: `opencodeProfile`, `opencodeProfileConfig`, and `opencodeAgentCommandScript` are used consistently across tasks.
