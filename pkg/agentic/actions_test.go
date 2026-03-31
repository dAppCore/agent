// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	"dappco.re/go/agent/pkg/lib"
	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActions_HandleDispatch_Good(t *testing.T) {
	s := newPrepWithProcess()
	r := s.handleDispatch(context.Background(), core.NewOptions(
		core.Option{Key: "repo", Value: "go-io"},
		core.Option{Key: "task", Value: "fix tests"},
	))
	// Will fail (no local clone) but exercises the handler path
	assert.False(t, r.OK)
}

func TestActions_HandleStatus_Good(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	s := newPrepWithProcess()
	r := s.handleStatus(context.Background(), core.NewOptions())
	assert.True(t, r.OK)
}

func TestActions_HandlePrompt_Good(t *testing.T) {
	s := newPrepWithProcess()
	r := s.handlePrompt(context.Background(), core.NewOptions(
		core.Option{Key: "slug", Value: "coding"},
	))
	assert.True(t, r.OK)
}

func TestActions_HandlePrompt_Bad(t *testing.T) {
	s := newPrepWithProcess()
	r := s.handlePrompt(context.Background(), core.NewOptions(
		core.Option{Key: "slug", Value: "does-not-exist"},
	))
	assert.False(t, r.OK)
}

func TestActions_HandleTask_Good(t *testing.T) {
	s := newPrepWithProcess()
	r := s.handleTask(context.Background(), core.NewOptions(
		core.Option{Key: "slug", Value: "bug-fix"},
	))
	assert.True(t, r.OK)
}

func TestActions_HandleFlow_Good(t *testing.T) {
	s := newPrepWithProcess()
	r := s.handleFlow(context.Background(), core.NewOptions(
		core.Option{Key: "slug", Value: "go"},
	))
	assert.True(t, r.OK)
}

func TestActions_HandlePersona_Good(t *testing.T) {
	personas := lib.ListPersonas()
	if len(personas) == 0 {
		t.Skip("no personas embedded")
	}

	s := newPrepWithProcess()
	r := s.handlePersona(context.Background(), core.NewOptions(
		core.Option{Key: "path", Value: personas[0]},
	))
	assert.True(t, r.OK)
}

func TestActions_HandlePoke_Good(t *testing.T) {
	s := newPrepWithProcess()
	s.pokeCh = make(chan struct{}, 1)
	r := s.handlePoke(context.Background(), core.NewOptions())
	assert.True(t, r.OK)
}

func TestActions_HandlePoke_Good_DelegatesToRunner(t *testing.T) {
	called := false
	c := core.New()
	c.Action("runner.poke", func(_ context.Context, _ core.Options) core.Result {
		called = true
		return core.Result{OK: true}
	})

	s := NewPrep()
	s.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})

	r := s.handlePoke(context.Background(), core.NewOptions())
	require.True(t, r.OK)
	assert.True(t, called)
}

func TestActions_HandleQA_Bad_NoWorkspace(t *testing.T) {
	s := newPrepWithProcess()
	r := s.handleQA(context.Background(), core.NewOptions())
	assert.False(t, r.OK)
}

func TestActions_HandleVerify_Bad_NoWorkspace(t *testing.T) {
	s := newPrepWithProcess()
	r := s.handleVerify(context.Background(), core.NewOptions())
	assert.False(t, r.OK)
}

func TestActions_HandleIngest_Bad_NoWorkspace(t *testing.T) {
	s := newPrepWithProcess()
	r := s.handleIngest(context.Background(), core.NewOptions())
	assert.False(t, r.OK)
}

func TestActions_HandleWorkspaceQuery_Good(t *testing.T) {
	s := newPrepWithProcess()
	s.workspaces = core.NewRegistry[*WorkspaceStatus]()
	s.workspaces.Set("core/go-io/task-42", &WorkspaceStatus{Status: "blocked", Repo: "go-io"})

	r := s.handleWorkspaceQuery(nil, WorkspaceQuery{Status: "blocked"})
	require.True(t, r.OK)

	names, ok := r.Value.([]string)
	require.True(t, ok)
	require.Len(t, names, 1)
	assert.Equal(t, "core/go-io/task-42", names[0])
}

func TestActions_HandleWorkspaceQuery_Bad(t *testing.T) {
	s := newPrepWithProcess()
	r := s.handleWorkspaceQuery(nil, "not-a-workspace-query")
	assert.False(t, r.OK)
	assert.Nil(t, r.Value)
}

func TestActions_DispatchInputFromOptions_Good_MapsRFCFields(t *testing.T) {
	input := dispatchInputFromOptions(core.NewOptions(
		core.Option{Key: "repo", Value: "go-io"},
		core.Option{Key: "org", Value: "core"},
		core.Option{Key: "task", Value: "Fix the failing tests"},
		core.Option{Key: "agent", Value: "codex:gpt-5.4"},
		core.Option{Key: "template", Value: "coding"},
		core.Option{Key: "plan_template", Value: "bug-fix"},
		core.Option{Key: "variables", Value: map[string]any{"ISSUE": 42, "MODE": "deep"}},
		core.Option{Key: "persona", Value: "code/reviewer"},
		core.Option{Key: "issue", Value: "42"},
		core.Option{Key: "pr", Value: 7},
		core.Option{Key: "branch", Value: "agent/fix-tests"},
		core.Option{Key: "tag", Value: "v0.8.0"},
		core.Option{Key: "dry-run", Value: "true"},
	))

	assert.Equal(t, "go-io", input.Repo)
	assert.Equal(t, "core", input.Org)
	assert.Equal(t, "Fix the failing tests", input.Task)
	assert.Equal(t, "codex:gpt-5.4", input.Agent)
	assert.Equal(t, "coding", input.Template)
	assert.Equal(t, "bug-fix", input.PlanTemplate)
	assert.Equal(t, map[string]string{"ISSUE": "42", "MODE": "deep"}, input.Variables)
	assert.Equal(t, "code/reviewer", input.Persona)
	assert.Equal(t, 42, input.Issue)
	assert.Equal(t, 7, input.PR)
	assert.Equal(t, "agent/fix-tests", input.Branch)
	assert.Equal(t, "v0.8.0", input.Tag)
	assert.True(t, input.DryRun)
}

func TestActions_PrepInputFromOptions_Good_MapsRFCFields(t *testing.T) {
	input := prepInputFromOptions(core.NewOptions(
		core.Option{Key: "repo", Value: "go-scm"},
		core.Option{Key: "org", Value: "core"},
		core.Option{Key: "task", Value: "Prepare release branch"},
		core.Option{Key: "agent", Value: "claude"},
		core.Option{Key: "issue", Value: 12},
		core.Option{Key: "branch", Value: "dev"},
		core.Option{Key: "template", Value: "security"},
		core.Option{Key: "plan-template", Value: "release"},
		core.Option{Key: "variables", Value: "{\"REPO\":\"go-scm\",\"MODE\":\"resume\"}"},
		core.Option{Key: "persona", Value: "code/security"},
		core.Option{Key: "dry_run", Value: true},
	))

	assert.Equal(t, "go-scm", input.Repo)
	assert.Equal(t, "core", input.Org)
	assert.Equal(t, "Prepare release branch", input.Task)
	assert.Equal(t, "claude", input.Agent)
	assert.Equal(t, 12, input.Issue)
	assert.Equal(t, "dev", input.Branch)
	assert.Equal(t, "security", input.Template)
	assert.Equal(t, "release", input.PlanTemplate)
	assert.Equal(t, map[string]string{"REPO": "go-scm", "MODE": "resume"}, input.Variables)
	assert.Equal(t, "code/security", input.Persona)
	assert.True(t, input.DryRun)
}

func TestActions_WatchInputFromOptions_Good_ParsesWorkspaceList(t *testing.T) {
	input := watchInputFromOptions(core.NewOptions(
		core.Option{Key: "workspaces", Value: []any{"core/go-io/task-5", " core/go-scm/task-6 "}},
		core.Option{Key: "poll-interval", Value: "15"},
		core.Option{Key: "timeout", Value: "900"},
	))

	assert.Equal(t, []string{"core/go-io/task-5", "core/go-scm/task-6"}, input.Workspaces)
	assert.Equal(t, 15, input.PollInterval)
	assert.Equal(t, 900, input.Timeout)
}

func TestActions_ReviewQueueInputFromOptions_Good_MapsLocalOnly(t *testing.T) {
	input := reviewQueueInputFromOptions(core.NewOptions(
		core.Option{Key: "limit", Value: "4"},
		core.Option{Key: "reviewer", Value: "both"},
		core.Option{Key: "dry_run", Value: true},
		core.Option{Key: "local_only", Value: "yes"},
	))

	assert.Equal(t, 4, input.Limit)
	assert.Equal(t, "both", input.Reviewer)
	assert.True(t, input.DryRun)
	assert.True(t, input.LocalOnly)
}

func TestActions_EpicInputFromOptions_Good_ParsesListFields(t *testing.T) {
	input := epicInputFromOptions(core.NewOptions(
		core.Option{Key: "repo", Value: "go-io"},
		core.Option{Key: "org", Value: "core"},
		core.Option{Key: "title", Value: "AX RFC follow-up"},
		core.Option{Key: "body", Value: "Finish the remaining wrappers"},
		core.Option{Key: "tasks", Value: "[\"Map action inputs\",\"Add tests\"]"},
		core.Option{Key: "labels", Value: "agentic, ax"},
		core.Option{Key: "dispatch", Value: "true"},
		core.Option{Key: "agent", Value: "codex"},
		core.Option{Key: "template", Value: "coding"},
	))

	assert.Equal(t, "go-io", input.Repo)
	assert.Equal(t, "core", input.Org)
	assert.Equal(t, "AX RFC follow-up", input.Title)
	assert.Equal(t, "Finish the remaining wrappers", input.Body)
	assert.Equal(t, []string{"Map action inputs", "Add tests"}, input.Tasks)
	assert.Equal(t, []string{"agentic", "ax"}, input.Labels)
	assert.True(t, input.Dispatch)
	assert.Equal(t, "codex", input.Agent)
	assert.Equal(t, "coding", input.Template)
}

func TestActions_NormaliseForgeActionOptions_Good_MapsRepoAndNumber(t *testing.T) {
	options := normaliseForgeActionOptions(core.NewOptions(
		core.Option{Key: "repo", Value: "go-io"},
		core.Option{Key: "number", Value: 12},
		core.Option{Key: "title", Value: "Fix watcher"},
	))

	assert.Equal(t, "go-io", options.String("_arg"))
	assert.Equal(t, "12", options.String("number"))
	assert.Equal(t, "Fix watcher", options.String("title"))
}

func TestActions_OptionHelpers_Ugly_IgnoreMalformedMapJSON(t *testing.T) {
	input := dispatchInputFromOptions(core.NewOptions(
		core.Option{Key: "repo", Value: "go-io"},
		core.Option{Key: "task", Value: "Review"},
		core.Option{Key: "variables", Value: "{\"BROKEN\""},
	))

	assert.Nil(t, input.Variables)
}
