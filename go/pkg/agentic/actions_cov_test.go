// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- optionIntValue (int64 / float64 / string-zero arms) ---

func TestActions_OptionIntValue_Good_NumericTypes(t *testing.T) {
	core.AssertEqual(t, 7, optionIntValue(core.NewOptions(core.Option{Key: "n", Value: int64(7)}), "n"))
	core.AssertEqual(t, 9, optionIntValue(core.NewOptions(core.Option{Key: "n", Value: float64(9)}), "n"))
	core.AssertEqual(t, 0, optionIntValue(core.NewOptions(core.Option{Key: "n", Value: "0"}), "n"))
	core.AssertEqual(t, 42, optionIntValue(core.NewOptions(core.Option{Key: "n", Value: "42"}), "n"))
}

func TestActions_OptionIntValue_Bad_MissingAndUnparseable(t *testing.T) {
	core.AssertEqual(t, 0, optionIntValue(core.NewOptions(), "absent"))
	// A non-numeric string yields 0 via the parseIntString fallback.
	core.AssertEqual(t, 0, optionIntValue(core.NewOptions(core.Option{Key: "n", Value: "abc"}), "n"))
}

// --- stringValue (int / int64 / float64 / bool arms) ---

func TestActions_StringValue_Good_AllScalarTypes(t *testing.T) {
	core.AssertEqual(t, "5", stringValue(5))
	core.AssertEqual(t, "6", stringValue(int64(6)))
	core.AssertEqual(t, "7", stringValue(float64(7)))
	core.AssertEqual(t, "true", stringValue(true))
	core.AssertEqual(t, "text", stringValue("text"))
}

func TestActions_StringValue_Bad_UnsupportedType(t *testing.T) {
	core.AssertEqual(t, "", stringValue([]int{1, 2}))
	core.AssertEqual(t, "", stringValue(nil))
}

// --- stringSliceValue ([]any / JSON-array string / generic fallback) ---

func TestActions_StringSliceValue_Good_AnySlice(t *testing.T) {
	got := stringSliceValue([]any{"a", " b ", "", "c"})
	core.AssertEqual(t, []string{"a", "b", "c"}, got)
}

func TestActions_StringSliceValue_Good_JSONArrayString(t *testing.T) {
	core.AssertEqual(t, []string{"x", "y"}, stringSliceValue(`["x","y"]`))
}

func TestActions_StringSliceValue_Ugly_GenericArrayFallback(t *testing.T) {
	// A JSON array of mixed scalars falls back to the generic []any decode.
	got := stringSliceValue(`[1,2,3]`)
	core.AssertEqual(t, []string{"1", "2", "3"}, got)
}

func TestActions_StringSliceValue_Bad_ScalarFallback(t *testing.T) {
	// A non-collection scalar becomes a single-element slice.
	core.AssertEqual(t, []string{"7"}, stringSliceValue(7))
	core.AssertNil(t, stringSliceValue(""))
}

// --- normaliseOptionValue (object / array / bool / int / string arms) ---

func TestActions_NormaliseOptionValue_Good_AllArms(t *testing.T) {
	obj, ok := normaliseOptionValue(`{"k":"v"}`).(map[string]any)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "v", obj["k"])

	arr, ok := normaliseOptionValue(`[1,2]`).([]any)
	core.RequireTrue(t, ok)
	core.AssertLen(t, arr, 2)

	core.AssertEqual(t, true, normaliseOptionValue("true"))
	core.AssertEqual(t, false, normaliseOptionValue("false"))
	core.AssertEqual(t, 5, normaliseOptionValue("5"))
	core.AssertEqual(t, "plain", normaliseOptionValue("plain"))
}

func TestActions_NormaliseOptionValue_Bad_EmptyAndNonString(t *testing.T) {
	core.AssertEqual(t, "", normaliseOptionValue(""))
	// Non-string passes through untouched.
	core.AssertEqual(t, 42, normaliseOptionValue(42))
}

// --- stringMapValue (map[string]any / []any / JSON-object generic fallback) ---

func TestActions_StringMapValue_Good_MapAnyValues(t *testing.T) {
	got := stringMapValue(map[string]any{"a": 1, "b": "two", "c": ""})
	core.AssertEqual(t, "1", got["a"])
	core.AssertEqual(t, "two", got["b"])
	_, hasC := got["c"]
	core.AssertFalse(t, hasC)
}

func TestActions_StringMapValue_Good_AnySliceOfPairs(t *testing.T) {
	got := stringMapValue([]any{"k1=v1", "k2=v2"})
	core.AssertEqual(t, map[string]string{"k1": "v1", "k2": "v2"}, got)
}

func TestActions_StringMapValue_Ugly_JSONObjectGenericFallback(t *testing.T) {
	// A JSON object with non-string values decodes via the generic map fallback.
	got := stringMapValue(`{"n":1,"s":"x"}`)
	core.AssertEqual(t, "1", got["n"])
	core.AssertEqual(t, "x", got["s"])
}

func TestActions_StringMapValue_Bad_EmptyAndUnsupported(t *testing.T) {
	core.AssertNil(t, stringMapValue(""))
	core.AssertNil(t, stringMapValue(42))
}

// --- mergeStringMapEntry (no '=' / empty key or value) ---

func TestActions_MergeStringMapEntry_Bad_RejectsMalformed(t *testing.T) {
	out := map[string]string{}
	mergeStringMapEntry(out, "no-equals-here")
	mergeStringMapEntry(out, "=novalue")
	mergeStringMapEntry(out, "nokey=")
	mergeStringMapEntry(out, "  ")
	core.AssertLen(t, out, 0)

	mergeStringMapEntry(out, "key = value")
	core.AssertEqual(t, "value", out["key"])
}

// --- handleQA (passing go repo, then failing repo) ---

func TestActions_HandleQA_Good_PassingRepo(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	wsDir := core.JoinPath(root, "ws-qa-pass")
	repoDir := core.JoinPath(wsDir, "repo")
	core.RequireTrue(t, fs.EnsureDir(repoDir).OK)
	fs.Write(core.JoinPath(repoDir, "go.mod"), "module testmod\n\ngo 1.22\n")
	fs.Write(core.JoinPath(repoDir, "main.go"), "package main\nfunc main() {}\n")
	fs.Write(core.JoinPath(wsDir, "status.json"), core.JSONMarshalString(&WorkspaceStatus{Status: "running", Repo: "go-io"}))

	// testCore has auto-qa enabled + process registered, so the QA + ACTION
	// emission path both run.
	s := newPrepWithProcess()
	r := s.handleQA(context.Background(), core.NewOptions(core.Option{Key: "workspace", Value: wsDir}))
	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, true, r.Value)
}

func TestActions_HandleQA_Bad_FailingRepoFlipsStatus(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	wsDir := core.JoinPath(root, "ws-qa-fail")
	repoDir := core.JoinPath(wsDir, "repo")
	core.RequireTrue(t, fs.EnsureDir(repoDir).OK)
	fs.Write(core.JoinPath(repoDir, "go.mod"), "module testmod\n\ngo 1.22\n")
	// Broken source — build fails so QA returns false.
	fs.Write(core.JoinPath(repoDir, "main.go"), "package main\nfunc main( {\n}\n")
	fs.Write(core.JoinPath(wsDir, "status.json"), core.JSONMarshalString(&WorkspaceStatus{Status: "running", Repo: "go-io"}))

	s := newPrepWithProcess()
	r := s.handleQA(context.Background(), core.NewOptions(core.Option{Key: "workspace", Value: wsDir}))
	core.AssertFalse(t, r.OK)

	// The failure path writes the QA-failed status back to disk.
	updated := mustReadStatus(t, wsDir)
	core.AssertEqual(t, "failed", updated.Status)
	core.AssertContains(t, updated.Question, "QA check failed")
}

func TestActions_HandleQA_Ugly_DisabledGateShortCircuits(t *testing.T) {
	// A fresh core without auto-qa enabled returns OK immediately, never
	// touching the workspace.
	c := core.New()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	r := s.handleQA(context.Background(), core.NewOptions(core.Option{Key: "workspace", Value: "/does/not/matter"}))
	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, true, r.Value)
}

// --- completeTool (success: agent.completion task runs through stub steps) ---

func TestActions_CompleteTool_Good_RunsCompletionTask(t *testing.T) {
	c := core.New()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Register the completion task + stub step actions so completeTool's
	// success envelope is exercised without running real QA/PR/merge.
	var ran []string
	for _, name := range []string{"agentic.qa", "agentic.auto-pr", "agentic.verify", "agentic.commit", "agentic.ingest", "agentic.poke"} {
		stepName := name
		c.Action(stepName, func(_ context.Context, _ core.Options) core.Result {
			ran = append(ran, stepName)
			return core.Result{OK: true}
		})
	}
	c.Task("agent.completion", core.Task{
		Steps: []core.Step{
			{Action: "agentic.qa"},
			{Action: "agentic.auto-pr"},
			{Action: "agentic.verify"},
		},
	})

	result := s.completeTool(context.Background(), CompleteInput{Workspace: "core/go-io/task-9"})
	core.RequireTrue(t, result.OK)
	out, ok := result.Value.(CompleteOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, out.Success)
	core.AssertEqual(t, "core/go-io/task-9", out.Workspace)
	core.AssertEqual(t, []string{"agentic.qa", "agentic.auto-pr", "agentic.verify"}, ran)
}

func TestActions_CompleteTool_Ugly_TaskStepFails(t *testing.T) {
	c := core.New()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	c.Action("agentic.qa", func(_ context.Context, _ core.Options) core.Result {
		return core.Result{Value: core.E("agentic.qa", "qa exploded", nil), OK: false}
	})
	c.Task("agent.completion", core.Task{Steps: []core.Step{{Action: "agentic.qa"}}})

	result := s.completeTool(context.Background(), CompleteInput{Workspace: "core/go-io/task-9"})
	core.AssertFalse(t, result.OK)
}

// --- handleIngest (enabled path runs ingestFindings on a bare workspace) ---

func TestActions_HandleIngest_Good_RunsOnBareWorkspace(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	wsDir := core.JoinPath(root, "ws-ingest")
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(wsDir, ".meta")).OK)

	// ingestFindings degrades gracefully when there are no findings to ingest;
	// the handler still returns OK.
	s := newPrepWithProcess()
	r := s.handleIngest(context.Background(), core.NewOptions(core.Option{Key: "workspace", Value: wsDir}))
	core.AssertTrue(t, r.OK)
}

// --- handleAutoPR (enabled path on a workspace with no PR yet) ---

func TestActions_HandleAutoPR_Good_NoPRURLStillOK(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	wsDir := core.JoinPath(root, "ws-autopr")
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(wsDir, "repo")).OK)
	fs.Write(core.JoinPath(wsDir, "status.json"), core.JSONMarshalString(&WorkspaceStatus{Status: "completed", Repo: "go-io"}))

	// auto-pr is enabled on testCore; autoCreatePR finds no committable work
	// and leaves PRURL empty, so the PRCreated emission is skipped but the
	// handler still returns OK.
	s := newPrepWithProcess()
	r := s.handleAutoPR(context.Background(), core.NewOptions(core.Option{Key: "workspace", Value: wsDir}))
	core.AssertTrue(t, r.OK)
}

// --- handleVerify (enabled path on a workspace, no merge happens) ---

func TestActions_HandleVerify_Good_NoMergeStillOK(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	wsDir := core.JoinPath(root, "ws-verify")
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(wsDir, "repo")).OK)
	fs.Write(core.JoinPath(wsDir, "status.json"), core.JSONMarshalString(&WorkspaceStatus{Status: "completed", Repo: "go-io"}))

	s := newPrepWithProcess()
	r := s.handleVerify(context.Background(), core.NewOptions(core.Option{Key: "workspace", Value: wsDir}))
	core.AssertTrue(t, r.OK)
}

// --- handleBranchDelete (success via the deleteBranch seam) ---

func TestActions_HandleBranchDelete_Good_DispatchesDelete(t *testing.T) {
	s := newPrepWithProcess()

	orig := deleteBranch
	t.Cleanup(func() { deleteBranch = orig })
	deleteBranch = func(_ *PrepSubsystem, _ context.Context, _ *mcp.CallToolRequest, input DeleteBranchInput) (*mcp.CallToolResult, DeleteBranchOutput, error) {
		core.AssertEqual(t, "go-io", input.Repo)
		core.AssertEqual(t, "agent/fix", input.Branch)
		return nil, DeleteBranchOutput{Success: true}, nil
	}

	r := s.handleBranchDelete(context.Background(), core.NewOptions(
		core.Option{Key: "repo", Value: "go-io"},
		core.Option{Key: "branch", Value: "agent/fix"},
	))
	core.RequireTrue(t, r.OK)
	out, ok := r.Value.(DeleteBranchOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, out.Success)
}

func TestActions_HandleBranchDelete_Bad_SeamErrors(t *testing.T) {
	s := newPrepWithProcess()

	orig := deleteBranch
	t.Cleanup(func() { deleteBranch = orig })
	deleteBranch = func(_ *PrepSubsystem, _ context.Context, _ *mcp.CallToolRequest, _ DeleteBranchInput) (*mcp.CallToolResult, DeleteBranchOutput, error) {
		return nil, DeleteBranchOutput{}, core.E("agentic.branch.delete", "branch not found", nil)
	}

	r := s.handleBranchDelete(context.Background(), core.NewOptions(core.Option{Key: "repo", Value: "go-io"}))
	core.AssertFalse(t, r.OK)
}

// --- handleComplete (nil core guard already covered; success path) ---

func TestActions_HandleComplete_Good_DelegatesToTask(t *testing.T) {
	c := core.New()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	ran := false
	c.Action("agentic.qa", func(_ context.Context, _ core.Options) core.Result {
		ran = true
		return core.Result{OK: true}
	})
	c.Task("agent.completion", core.Task{Steps: []core.Step{{Action: "agentic.qa"}}})

	r := s.handleComplete(context.Background(), core.NewOptions(core.Option{Key: "workspace", Value: "core/go-io/task-1"}))
	core.AssertTrue(t, r.OK)
	core.AssertTrue(t, ran)
}
