// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// --- pure flow helpers (no test references existed) ---

// TestCommandsCov_FlowStepSummary_Good_LabelPrecedence verifies the label
// fallback chain (name → flow → cmd → agent → run → "step") and the per-kind
// suffix rendering (flow/cmd/agent/run/gate).
func TestCommandsCov_FlowStepSummary_Good_LabelPrecedence(t *testing.T) {
	core.AssertEqual(t, "build: flow ci.yaml", flowStepSummary(flowDefinitionStep{Name: "build", Flow: "ci.yaml"}))
	// No name → label falls through to the flow value.
	core.AssertEqual(t, "ci.yaml: flow ci.yaml", flowStepSummary(flowDefinitionStep{Flow: "ci.yaml"}))
	core.AssertEqual(t, "lint: cmd task lint", flowStepSummary(flowDefinitionStep{Name: "lint", Cmd: "task", Args: []string{"lint"}}))
	core.AssertEqual(t, "review: agent codex", flowStepSummary(flowDefinitionStep{Name: "review", Agent: "codex"}))
	core.AssertEqual(t, "smoke: run ./smoke.sh", flowStepSummary(flowDefinitionStep{Name: "smoke", Run: "./smoke.sh"}))
	core.AssertEqual(t, "gate-it: gate qa", flowStepSummary(flowDefinitionStep{Name: "gate-it", Gate: "qa"}))
}

// TestCommandsCov_FlowStepSummary_Ugly_EmptyStepIsLabelledStep — a step with no
// distinguishing field still produces the "step" sentinel and hits the default
// switch arm.
func TestCommandsCov_FlowStepSummary_Ugly_EmptyStepIsLabelledStep(t *testing.T) {
	core.AssertEqual(t, "step", flowStepSummary(flowDefinitionStep{}))
	// A bare name with no action kind hits the default arm and returns the label verbatim.
	core.AssertEqual(t, "just-a-name", flowStepSummary(flowDefinitionStep{Name: "just-a-name"}))
}

// TestCommandsCov_FlowSlugFromPath_Good_StripsKnownSuffixes verifies the slug
// derivation strips .yaml/.yml/.md and the directory.
func TestCommandsCov_FlowSlugFromPath_Good_StripsKnownSuffixes(t *testing.T) {
	core.AssertEqual(t, "ci", flowSlugFromPath("pkg/lib/flow/ci.yaml"))
	core.AssertEqual(t, "release", flowSlugFromPath("release.yml"))
	core.AssertEqual(t, "onboard", flowSlugFromPath("flows/onboard.md"))
	core.AssertEqual(t, "bare", flowSlugFromPath("bare"))
}

// TestCommandsCov_FlowInputLooksYaml_Good_ExtensionDetection — only .yaml/.yml
// are treated as YAML, so .md parse failures fall back to raw content.
func TestCommandsCov_FlowInputLooksYaml_Good_ExtensionDetection(t *testing.T) {
	core.AssertTrue(t, flowInputLooksYaml("a.yaml"))
	core.AssertTrue(t, flowInputLooksYaml("a.yml"))
	core.AssertFalse(t, flowInputLooksYaml("a.md"))
	core.AssertFalse(t, flowInputLooksYaml("noext"))
}

// TestCommandsCov_FlowRootPath_Good_FindsFlowRoot verifies the pkg/lib/flow
// anchor is detected, and otherwise the parent directory is returned.
func TestCommandsCov_FlowRootPath_Good_FindsFlowRoot(t *testing.T) {
	core.AssertEqual(t, core.JoinPath("pkg", "lib", "flow"), flowRootPath("pkg/lib/flow/sub/ci.yaml"))
	// No flow anchor → parent directory of the source.
	core.AssertEqual(t, core.JoinPath("flows", "team"), flowRootPath("flows/team/onboard.yaml"))
	// Backslashes are normalised to forward slashes before splitting.
	core.AssertEqual(t, core.JoinPath("pkg", "lib", "flow"), flowRootPath("pkg\\lib\\flow\\ci.yaml"))
}

// TestCommandsCov_FlowRootPath_Ugly_EmptyAndBareSources — empty source yields
// empty; a bare filename with no directory yields empty (PathDir returns "").
func TestCommandsCov_FlowRootPath_Ugly_EmptyAndBareSources(t *testing.T) {
	core.AssertEqual(t, "", flowRootPath(""))
	core.AssertEqual(t, "", flowRootPath("   "))
}

// --- extractAgentOutputContent (no test references existed) ---

// TestCommandsCov_ExtractAgentOutputContent_Good_JSONPassthrough — content that
// already starts as a JSON object/array is returned verbatim (trimmed).
func TestCommandsCov_ExtractAgentOutputContent_Good_JSONPassthrough(t *testing.T) {
	core.AssertEqual(t, `{"ok":true}`, extractAgentOutputContent("  {\"ok\":true}  "))
	core.AssertEqual(t, `[1,2,3]`, extractAgentOutputContent("[1,2,3]"))
}

// TestCommandsCov_ExtractAgentOutputContent_Good_FencedBlockWithLanguage — a
// fenced code block with a single-word language tag drops the tag and returns
// the body.
func TestCommandsCov_ExtractAgentOutputContent_Good_FencedBlockWithLanguage(t *testing.T) {
	content := "Here is the result:\n```json\n{\"plan\":\"x\"}\n```\nthanks"
	core.AssertEqual(t, `{"plan":"x"}`, extractAgentOutputContent(content))
}

// TestCommandsCov_ExtractAgentOutputContent_Ugly_NoExtractableContent — prose
// with no JSON and no fenced block returns empty, and a fence whose first line
// is multi-word (not a language) is kept intact.
func TestCommandsCov_ExtractAgentOutputContent_Ugly_NoExtractableContent(t *testing.T) {
	core.AssertEqual(t, "", extractAgentOutputContent("just some prose, nothing to extract"))
	core.AssertEqual(t, "", extractAgentOutputContent("   "))
	// First fence line has a space → treated as content, not a language tag.
	core.AssertEqual(t, "two words here", extractAgentOutputContent("```\ntwo words here\n```"))
}

// --- brain output decoders (no test references existed) ---

// TestCommandsCov_BrainListOutputFromPayload_Good_DecodesEntries verifies count
// + memory entries are decoded from a generic map, including the float64 count
// path that JSON decoding produces.
func TestCommandsCov_BrainListOutputFromPayload_Good_DecodesEntries(t *testing.T) {
	payload := map[string]any{
		"count": float64(3),
		"memories": []any{
			// float64 confidence + int supersedes_count + tags + deleted_at.
			map[string]any{
				"id": "m1", "type": "fact", "content": "alpha", "project": "core", "agent_id": "cladius",
				"confidence": float64(0.9), "supersedes_count": 2, "deleted_at": "2026-06-01T00:00:00Z",
				"tags": []any{"x", "y"},
			},
			// int confidence + float64 supersedes_count.
			map[string]any{"id": "m2", "type": "note", "content": "beta", "confidence": 1, "supersedes_count": float64(4)},
			// no confidence → falls back to the score field (int arm).
			map[string]any{"id": "m3", "type": "note", "content": "gamma", "score": 5},
			"not-a-map", // skipped
		},
	}

	out := brainListOutputFromPayload(payload)
	core.AssertEqual(t, 3, out.Count)
	core.RequireTrue(t, len(out.Memories) == 3)
	core.AssertEqual(t, "m1", out.Memories[0].ID)
	core.AssertEqual(t, "core", out.Memories[0].Project)
	core.AssertEqual(t, "cladius", out.Memories[0].AgentID)
	core.AssertEqual(t, 2, out.Memories[0].SupersedesCount)
	core.AssertEqual(t, "2026-06-01T00:00:00Z", out.Memories[0].DeletedAt)
	core.AssertEqual(t, []string{"x", "y"}, out.Memories[0].Tags)
	core.AssertEqual(t, float64(1), out.Memories[1].Confidence)
	core.AssertEqual(t, 4, out.Memories[1].SupersedesCount)
	core.AssertEqual(t, float64(5), out.Memories[2].Confidence)
}

// TestCommandsCov_BrainListOutputFromPayload_Good_CountFallsBackToLen — when the
// payload omits count, it is derived from the number of decoded memories.
func TestCommandsCov_BrainListOutputFromPayload_Good_CountFallsBackToLen(t *testing.T) {
	out := brainListOutputFromPayload(map[string]any{
		"memories": []any{
			map[string]any{"id": "only"},
		},
	})
	core.AssertEqual(t, 1, out.Count)
}

// TestCommandsCov_BrainListOutputFromPayload_Ugly_IntCountAndNoMemories — the
// int-typed count arm and a payload missing the memories key.
func TestCommandsCov_BrainListOutputFromPayload_Ugly_IntCountAndNoMemories(t *testing.T) {
	out := brainListOutputFromPayload(map[string]any{"count": 5})
	core.AssertEqual(t, 5, out.Count)
	core.AssertEqual(t, 0, len(out.Memories))
}

// TestCommandsCov_BrainRecallOutputFromResult_Good_TypedAndPointer — the typed
// value and non-nil pointer arms both return the value with ok=true.
func TestCommandsCov_BrainRecallOutputFromResult_Good_TypedAndPointer(t *testing.T) {
	value := brainRecallOutput{Count: 3}
	got, ok := brainRecallOutputFromResult(value)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 3, got.Count)

	got, ok = brainRecallOutputFromResult(&value)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 3, got.Count)
}

// TestCommandsCov_BrainRecallOutputFromResult_Good_JSONFallback — an arbitrary
// map is JSON round-tripped into the output shape.
func TestCommandsCov_BrainRecallOutputFromResult_Good_JSONFallback(t *testing.T) {
	got, ok := brainRecallOutputFromResult(map[string]any{"count": 7})
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 7, got.Count)
}

// TestCommandsCov_BrainRecallOutputFromResult_Ugly_NilPointerAndUnmarshalable —
// a nil typed pointer returns ok=false, and a value that cannot JSON-decode
// into the output also returns ok=false.
func TestCommandsCov_BrainRecallOutputFromResult_Ugly_NilPointerAndUnmarshalable(t *testing.T) {
	var nilPtr *brainRecallOutput
	_, ok := brainRecallOutputFromResult(nilPtr)
	core.AssertFalse(t, ok)

	// A bare string marshals to a JSON scalar that cannot decode into the
	// struct → the unmarshal arm returns ok=false.
	_, ok = brainRecallOutputFromResult("not-a-recall-object")
	core.AssertFalse(t, ok)
}

// --- runFlowCommand / readFlowDocument / printFlowSteps / resolveFlowReference ---

// TestCommandsCov_CmdRunFlow_Good_ParsedFlowWithSteps drives a real YAML flow on
// disk through the full preview path: header line, var count, name/desc, and a
// per-step summary line.
func TestCommandsCov_CmdRunFlow_Good_ParsedFlowWithSteps(t *testing.T) {
	dir := t.TempDir()
	flowPath := core.JoinPath(dir, "ci.yaml")
	core.RequireTrue(t, fs.Write(flowPath, "name: CI\ndescription: Build and test {{repo}}\nsteps:\n  - name: build\n    cmd: task\n    args: [build]\n  - name: test\n    run: ./test.sh\n").OK)

	s := newTestPrep(t)
	output := captureStdout(t, func() {
		r := s.cmdFlowPreview(core.NewOptions(
			core.Option{Key: "_arg", Value: flowPath},
			core.Option{Key: "dry-run", Value: true},
			core.Option{Key: "var", Value: "repo=go-io"},
		))
		core.RequireTrue(t, r.OK)
		out, ok := r.Value.(FlowRunOutput)
		core.RequireTrue(t, ok)
		core.AssertEqual(t, "CI", out.Name)
		core.AssertEqual(t, 2, out.Steps)
	})

	core.AssertContains(t, output, "flow:  "+flowPath)
	core.AssertContains(t, output, "dry-run: true")
	core.AssertContains(t, output, "vars: 1")
	core.AssertContains(t, output, "name:  CI")
	core.AssertContains(t, output, "desc:  Build and test go-io")
	core.AssertContains(t, output, "steps: 2")
	core.AssertContains(t, output, "1. build: cmd task build")
	core.AssertContains(t, output, "2. test: run ./test.sh")
}

// TestCommandsCov_CmdRunFlow_Good_ResolvesNestedFlow — a step that references a
// sibling flow on disk is resolved and its steps printed inline.
func TestCommandsCov_CmdRunFlow_Good_ResolvesNestedFlow(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "child.yaml"), "name: Child\nsteps:\n  - name: childstep\n    run: ./child.sh\n").OK)
	parentPath := core.JoinPath(dir, "parent.yaml")
	core.RequireTrue(t, fs.Write(parentPath, "name: Parent\nsteps:\n  - name: callchild\n    flow: child.yaml\n").OK)

	s := newTestPrep(t)
	output := captureStdout(t, func() {
		r := s.cmdFlowPreview(core.NewOptions(core.Option{Key: "_arg", Value: parentPath}))
		core.RequireTrue(t, r.OK)
		out, ok := r.Value.(FlowRunOutput)
		core.RequireTrue(t, ok)
		// One parent step + one resolved child step.
		core.AssertEqual(t, 2, out.ResolvedSteps)
	})

	core.AssertContains(t, output, "1. callchild: flow child.yaml")
	core.AssertContains(t, output, "resolved: "+core.JoinPath(dir, "child.yaml"))
	core.AssertContains(t, output, "childstep: run ./child.sh")
}

// TestCommandsCov_CmdRunFlow_Ugly_CycleDetected — a flow that references itself
// is resolved once, then the cycle guard fires on the second visit.
func TestCommandsCov_CmdRunFlow_Ugly_CycleDetected(t *testing.T) {
	dir := t.TempDir()
	selfPath := core.JoinPath(dir, "loop.yaml")
	core.RequireTrue(t, fs.Write(selfPath, "name: Loop\nsteps:\n  - name: again\n    flow: loop.yaml\n").OK)

	s := newTestPrep(t)
	output := captureStdout(t, func() {
		r := s.cmdFlowPreview(core.NewOptions(core.Option{Key: "_arg", Value: selfPath}))
		core.RequireTrue(t, r.OK)
	})

	core.AssertContains(t, output, "cycle: "+selfPath)
}

// TestCommandsCov_CmdRunFlow_Good_ParallelStepsAndRawContent — a parsed flow with
// a parallel block prints the nested parallel summaries; a non-YAML .md file with
// no parseable definition falls back to the raw-content branch.
func TestCommandsCov_CmdRunFlow_Good_ParallelStepsAndRawContent(t *testing.T) {
	dir := t.TempDir()
	parallelPath := core.JoinPath(dir, "fan.yaml")
	core.RequireTrue(t, fs.Write(parallelPath, "name: Fan\nsteps:\n  - name: spread\n    parallel:\n      - name: a\n        run: ./a.sh\n      - name: b\n        run: ./b.sh\n").OK)

	s := newTestPrep(t)
	output := captureStdout(t, func() {
		r := s.cmdFlowPreview(core.NewOptions(core.Option{Key: "_arg", Value: parallelPath}))
		core.RequireTrue(t, r.OK)
	})
	core.AssertContains(t, output, "parallel:")
	core.AssertContains(t, output, "1. a: run ./a.sh")
	core.AssertContains(t, output, "2. b: run ./b.sh")

	// Raw markdown (no flow definition) → unparsed branch + content char count.
	rawPath := core.JoinPath(dir, "notes.md")
	core.RequireTrue(t, fs.Write(rawPath, "# Just notes\nno yaml here").OK)
	rawOutput := captureStdout(t, func() {
		r := s.cmdFlowPreview(core.NewOptions(core.Option{Key: "_arg", Value: rawPath}))
		core.RequireTrue(t, r.OK)
		out, ok := r.Value.(FlowRunOutput)
		core.RequireTrue(t, ok)
		core.AssertFalse(t, out.Parsed)
	})
	core.AssertContains(t, rawOutput, "content:")
}

// TestCommandsCov_CmdRunFlow_Bad_MissingPath — no path/slug argument prints usage
// and returns an error envelope.
func TestCommandsCov_CmdRunFlow_Bad_MissingPath(t *testing.T) {
	s := newTestPrep(t)
	output := captureStdout(t, func() {
		r := s.cmdFlowPreview(core.NewOptions())
		core.AssertFalse(t, r.OK)
		core.AssertContains(t, r.Value.(error).Error(), "flow path or slug is required")
	})
	core.AssertContains(t, output, "usage: core-agent flow preview")
}

// TestCommandsCov_CmdRunFlow_Ugly_InvalidYamlFails — a .yaml file that is not a
// valid flow definition surfaces the parse error (the YAML-extension branch of
// readFlowDocument).
func TestCommandsCov_CmdRunFlow_Ugly_InvalidYamlFails(t *testing.T) {
	dir := t.TempDir()
	badPath := core.JoinPath(dir, "broken.yaml")
	// Valid YAML scalar but no Name field → parseFlowDefinition rejects it.
	core.RequireTrue(t, fs.Write(badPath, "description: nameless\n").OK)

	s := newTestPrep(t)
	output := captureStdout(t, func() {
		r := s.cmdFlowPreview(core.NewOptions(core.Option{Key: "_arg", Value: badPath}))
		core.AssertFalse(t, r.OK)
	})
	core.AssertContains(t, output, "error:")
}

// TestCommandsCov_CmdRunFlow_Ugly_FlowNotFound — a slug that resolves to nothing
// on disk and is not in the embedded library returns "flow not found".
func TestCommandsCov_CmdRunFlow_Ugly_FlowNotFound(t *testing.T) {
	s := newTestPrep(t)
	r := s.cmdFlowPreview(core.NewOptions(core.Option{Key: "_arg", Value: "definitely-not-a-real-flow-slug-xyz"}))
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Value.(error).Error(), "flow not found")
}

// TestCommandsCov_ResolveFlowReference_Bad_EmptyReference — an empty reference is
// rejected before any disk lookup.
func TestCommandsCov_ResolveFlowReference_Bad_EmptyReference(t *testing.T) {
	s := newTestPrep(t)
	r := s.resolveFlowReference("pkg/lib/flow/ci.yaml", "   ", nil)
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Value.(error).Error(), "flow reference is required")
}

// TestCommandsCov_ResolveFlowReference_Ugly_AllCandidatesMissing — a reference
// that exists in none of the candidate roots returns "flow not found".
func TestCommandsCov_ResolveFlowReference_Ugly_AllCandidatesMissing(t *testing.T) {
	s := newTestPrep(t)
	r := s.resolveFlowReference(core.JoinPath(t.TempDir(), "base.yaml"), "nope-missing.yaml", nil)
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Value.(error).Error(), "flow not found")
}

// --- cmdPromptVersion ---

// TestCommandsCov_CmdPromptVersion_Good_PrintsSnapshot writes a real prompt
// snapshot under an absolute workspace dir and asserts every printed field.
func TestCommandsCov_CmdPromptVersion_Good_PrintsSnapshot(t *testing.T) {
	workspaceDir := t.TempDir()
	prompt := "TASK: cover the prompt version command\n\nRead the RFC and commit."
	core.RequireTrue(t, writePromptSnapshot(workspaceDir, prompt).OK)

	s := newTestPrep(t)
	output := captureStdout(t, func() {
		r := s.cmdPromptVersion(core.NewOptions(core.Option{Key: "_arg", Value: workspaceDir}))
		core.RequireTrue(t, r.OK)
		out, ok := r.Value.(PromptVersionOutput)
		core.RequireTrue(t, ok)
		core.AssertEqual(t, promptSnapshotHash(prompt), out.Snapshot.Hash)
	})

	core.AssertContains(t, output, "workspace: "+workspaceDir)
	core.AssertContains(t, output, "hash:      "+promptSnapshotHash(prompt))
	core.AssertContains(t, output, "created:")
	core.AssertContains(t, output, core.Sprintf("chars:     %d", len(prompt)))
}

// TestCommandsCov_CmdPromptVersion_Bad_MissingWorkspace — no workspace argument
// prints usage and returns an error envelope.
func TestCommandsCov_CmdPromptVersion_Bad_MissingWorkspace(t *testing.T) {
	s := newTestPrep(t)
	output := captureStdout(t, func() {
		r := s.cmdPromptVersion(core.NewOptions())
		core.AssertFalse(t, r.OK)
		core.AssertContains(t, r.Value.(error).Error(), "workspace is required")
	})
	core.AssertContains(t, output, "usage: core-agent prompt version")
}

// TestCommandsCov_CmdPromptVersion_Ugly_CorruptSnapshot — a workspace whose
// snapshot JSON is corrupt surfaces the handler error (the !result.OK arm).
func TestCommandsCov_CmdPromptVersion_Ugly_CorruptSnapshot(t *testing.T) {
	workspaceDir := t.TempDir()
	metaDir := WorkspaceMetaDir(workspaceDir)
	core.RequireTrue(t, fs.EnsureDir(metaDir).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(metaDir, "prompt-version.json"), "{not-json").OK)

	s := newTestPrep(t)
	output := captureStdout(t, func() {
		r := s.cmdPromptVersion(core.NewOptions(core.Option{Key: "_arg", Value: workspaceDir}))
		core.AssertFalse(t, r.OK)
	})
	core.AssertContains(t, output, "error:")
}

// --- cmdMirror ---

// TestCommandsCov_CmdMirror_Good_SkippedNoGithubRemote drives the real mirror
// over a git repo that has no `github` remote, exercising the skipped-output
// loop and the count line.
func TestCommandsCov_CmdMirror_Good_SkippedNoGithubRemote(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	// codePath/core/<repo> is where mirror looks; create a repo with a git dir
	// but no github remote so it is reported as skipped.
	repoDir := core.JoinPath(s.codePath, "core", "go-io")
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(repoDir, ".git")).OK)

	output := captureStdout(t, func() {
		r := s.cmdMirror(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
		core.RequireTrue(t, r.OK)
		out, ok := r.Value.(MirrorOutput)
		core.RequireTrue(t, ok)
		core.AssertEqual(t, 0, out.Count)
		core.RequireTrue(t, len(out.Skipped) == 1)
		core.AssertContains(t, out.Skipped[0], "no github remote")
	})

	core.AssertContains(t, output, "count: 0")
	core.AssertContains(t, output, "skipped: go-io: no github remote")
}
