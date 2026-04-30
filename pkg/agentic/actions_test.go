// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/lib"
)

func TestActions_HandleDispatch_Good_Case(t *testing.T) {
	s := newPrepWithProcess()
	r := s.handleDispatch(context.Background(), core.NewOptions(
		core.Option{Key: "repo", Value: "go-io"},
		core.Option{Key: "task", Value: "fix tests"},
	))
	// Will fail (no local clone) but exercises the handler path
	core.AssertFalse(t, r.OK)
}

func TestActions_HandleDispatch_Bad_EntitlementDenied(t *testing.T) {
	c := core.New(core.WithService(ProcessRegister))
	c.ServiceStartup(context.Background(), nil)
	c.SetEntitlementChecker(func(action string, _ int, _ context.Context) core.Entitlement {
		if action == "agentic.concurrency" {
			return core.Entitlement{Allowed: false, Reason: "dispatch limit reached"}
		}
		return core.Entitlement{Allowed: true, Unlimited: true}
	})

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	r := s.handleDispatch(context.Background(), core.NewOptions(
		core.Option{Key: "repo", Value: "go-io"},
		core.Option{Key: "task", Value: "fix tests"},
	))

	core.AssertFalse(t, r.OK)
	err, ok := r.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "dispatch limit reached")
}

func TestActions_HandleDispatch_Good_RecordsUsage(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	t.Setenv("CORE_BRAIN_KEY", "")

	forgeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(core.JSONMarshalString(map[string]any{
			"title": "Issue",
			"body":  "Fix",
		})))
	}))
	t.Cleanup(forgeSrv.Close)

	s := newPrepWithProcess()
	s.Core().SetEntitlementChecker(func(_ string, _ int, _ context.Context) core.Entitlement {
		return core.Entitlement{Allowed: true, Unlimited: true}
	})

	srcRepo := core.JoinPath(t.TempDir(), "core", "go-io")
	core.RequireTrue(t, fs.EnsureDir(srcRepo).OK)
	process := s.Core().Process()
	core.RequireTrue(t, process.RunIn(context.Background(), srcRepo, "git", "init", "-b", "main").OK)
	core.RequireTrue(t, process.RunIn(context.Background(), srcRepo, "git", "config", "user.name", "Test").OK)
	core.RequireTrue(t, process.RunIn(context.Background(), srcRepo, "git", "config", "user.email", "test@test.com").OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(srcRepo, "go.mod"), "module test\ngo 1.22\n").OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(srcRepo, "README.md"), "hello\n").OK)
	core.RequireTrue(t, process.RunIn(context.Background(), srcRepo, "git", "add", ".").OK)
	core.RequireTrue(t, process.RunIn(
		context.Background(),
		srcRepo,
		"git",
		"commit",
		"-m", "initial commit",
	).OK)

	recorded := 0
	s.Core().SetUsageRecorder(func(action string, qty int, _ context.Context) {
		if action == "agentic.dispatch" && qty == 1 {
			recorded++
		}
	})

	s.forge = newForgeClient(forgeSrv.URL, "tok")
	s.codePath = core.PathDir(core.PathDir(srcRepo))

	r := s.handleDispatch(context.Background(), core.NewOptions(
		core.Option{Key: "repo", Value: "go-io"},
		core.Option{Key: "issue", Value: 42},
		core.Option{Key: "task", Value: "fix tests"},
		core.Option{Key: "dry-run", Value: true},
	))

	if !r.OK {
		t.Fatalf("dispatch failed: %#v", r.Value)
	}
	core.AssertEqual(t, 1, recorded)
}

func TestActions_HandleStatus_Good_Case(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	s := newPrepWithProcess()
	r := s.handleStatus(context.Background(), core.NewOptions())
	core.AssertTrue(t, r.OK)
}

func TestActions_HandlePrompt_Good_Case(t *testing.T) {
	s := newPrepWithProcess()
	r := s.handlePrompt(context.Background(), core.NewOptions(
		core.Option{Key: "slug", Value: "coding"},
	))
	core.AssertTrue(t, r.OK)
}

func TestActions_HandlePrompt_Bad_Case(t *testing.T) {
	s := newPrepWithProcess()
	r := s.handlePrompt(context.Background(), core.NewOptions(
		core.Option{Key: "slug", Value: "does-not-exist"},
	))
	core.AssertFalse(t, r.OK)
}

func TestActions_HandleTask_Good_Case(t *testing.T) {
	s := newPrepWithProcess()
	r := s.handleTask(context.Background(), core.NewOptions(
		core.Option{Key: "slug", Value: "bug-fix"},
	))
	core.AssertTrue(t, r.OK)
}

func TestActions_HandleFlow_Good_Case(t *testing.T) {
	s := newPrepWithProcess()
	r := s.handleFlow(context.Background(), core.NewOptions(
		core.Option{Key: "slug", Value: "go"},
	))
	core.AssertTrue(t, r.OK)
}

func TestActions_HandlePersona_Good_Case(t *testing.T) {
	personas := lib.ListPersonas()
	if len(personas) == 0 {
		t.Skip("no personas embedded")
	}

	s := newPrepWithProcess()
	r := s.handlePersona(context.Background(), core.NewOptions(
		core.Option{Key: `path`, Value: personas[0]},
	))
	core.AssertTrue(t, r.OK)
}

func TestActions_HandlePoke_Good_Case(t *testing.T) {
	s := newPrepWithProcess()
	s.pokeCh = make(chan struct{}, 1)
	r := s.handlePoke(context.Background(), core.NewOptions())
	core.AssertTrue(t, r.OK)
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
	core.RequireTrue(t, r.OK)
	core.AssertTrue(t, called)
}

func TestActions_HandleQA_Bad_NoWorkspace(t *testing.T) {
	s := newPrepWithProcess()
	r := s.handleQA(context.Background(), core.NewOptions())
	core.AssertFalse(t, r.OK)
}

func TestActions_HandleVerify_Bad_NoWorkspace(t *testing.T) {
	s := newPrepWithProcess()
	r := s.handleVerify(context.Background(), core.NewOptions())
	core.AssertFalse(t, r.OK)
}

func TestActions_HandleIngest_Bad_NoWorkspace(t *testing.T) {
	s := newPrepWithProcess()
	r := s.handleIngest(context.Background(), core.NewOptions())
	core.AssertFalse(t, r.OK)
}

func TestActions_HandleComplete_Bad_NoCore(t *testing.T) {
	s := &PrepSubsystem{}

	r := s.handleComplete(context.Background(), core.NewOptions())

	core.AssertFalse(t, r.OK)
	err, ok := r.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "core runtime is required")
}

func TestActions_HandleWorkspaceQuery_Good_Case(t *testing.T) {
	s := newPrepWithProcess()
	s.workspaces = core.NewRegistry[*WorkspaceStatus]()
	s.workspaces.Set("core/go-io/task-42", &WorkspaceStatus{Status: "blocked", Repo: "go-io"})

	r := s.handleWorkspaceQuery(nil, WorkspaceQuery{Status: "blocked"})
	core.RequireTrue(t, r.OK)

	names, ok := r.Value.([]string)
	core.RequireTrue(t, ok)
	core.AssertLen(t, names, 1)
	core.AssertEqual(t, "core/go-io/task-42", names[0])
}

func TestActions_HandleWorkspaceQuery_Bad_Case(t *testing.T) {
	s := newPrepWithProcess()
	r := s.handleWorkspaceQuery(nil, "not-a-workspace-query")
	core.AssertFalse(t, r.OK)
	core.AssertNil(t, r.Value)
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

	core.AssertEqual(t, "go-io", input.Repo)
	core.AssertEqual(t, "core", input.Org)
	core.AssertEqual(t, "Fix the failing tests", input.Task)
	core.AssertEqual(t, "codex:gpt-5.4", input.Agent)
	core.AssertEqual(t, "coding", input.Template)
	core.AssertEqual(t, "bug-fix", input.PlanTemplate)
	core.AssertEqual(t, map[string]string{"ISSUE": "42", "MODE": "deep"}, input.Variables)
	core.AssertEqual(t, "code/reviewer", input.Persona)
	core.AssertEqual(t, 42, input.Issue)
	core.AssertEqual(t, 7, input.PR)
	core.AssertEqual(t, "agent/fix-tests", input.Branch)
	core.AssertEqual(t, "v0.8.0", input.Tag)
	core.AssertTrue(t, input.DryRun)
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

	core.AssertEqual(t, "go-scm", input.Repo)
	core.AssertEqual(t, "core", input.Org)
	core.AssertEqual(t, "Prepare release branch", input.Task)
	core.AssertEqual(t, "claude", input.Agent)
	core.AssertEqual(t, 12, input.Issue)
	core.AssertEqual(t, "dev", input.Branch)
	core.AssertEqual(t, "security", input.Template)
	core.AssertEqual(t, "release", input.PlanTemplate)
	core.AssertEqual(t, map[string]string{"REPO": "go-scm", "MODE": "resume"}, input.Variables)
	core.AssertEqual(t, "code/security", input.Persona)
	core.AssertTrue(t, input.DryRun)
}

func TestActions_WatchInputFromOptions_Good_ParsesWorkspaceList(t *testing.T) {
	input := watchInputFromOptions(core.NewOptions(
		core.Option{Key: "workspaces", Value: []any{"core/go-io/task-5", " core/go-scm/task-6 "}},
		core.Option{Key: "poll-interval", Value: "15"},
		core.Option{Key: "timeout", Value: "900"},
	))

	core.AssertEqual(t, []string{"core/go-io/task-5", "core/go-scm/task-6"}, input.Workspaces)
	core.AssertEqual(t, 15, input.PollInterval)
	core.AssertEqual(t, 900, input.Timeout)
}

func TestActions_ReviewQueueInputFromOptions_Good_MapsLocalOnly(t *testing.T) {
	input := reviewQueueInputFromOptions(core.NewOptions(
		core.Option{Key: "limit", Value: "4"},
		core.Option{Key: "reviewer", Value: "both"},
		core.Option{Key: "dry_run", Value: true},
		core.Option{Key: "local_only", Value: "yes"},
	))

	core.AssertEqual(t, 4, input.Limit)
	core.AssertEqual(t, "both", input.Reviewer)
	core.AssertTrue(t, input.DryRun)
	core.AssertTrue(t, input.LocalOnly)
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

	core.AssertEqual(t, "go-io", input.Repo)
	core.AssertEqual(t, "core", input.Org)
	core.AssertEqual(t, "AX RFC follow-up", input.Title)
	core.AssertEqual(t, "Finish the remaining wrappers", input.Body)
	core.AssertEqual(t, []string{"Map action inputs", "Add tests"}, input.Tasks)
	core.AssertEqual(t, []string{"agentic", "ax"}, input.Labels)
	core.AssertTrue(t, input.Dispatch)
	core.AssertEqual(t, "codex", input.Agent)
	core.AssertEqual(t, "coding", input.Template)
}

func TestActions_NormaliseForgeActionOptions_Good_MapsRepoAndNumber(t *testing.T) {
	options := normaliseForgeActionOptions(core.NewOptions(
		core.Option{Key: "repo", Value: "go-io"},
		core.Option{Key: "number", Value: 12},
		core.Option{Key: "title", Value: "Fix watcher"},
	))

	core.AssertEqual(t, "go-io", options.String("_arg"))
	core.AssertEqual(t, "12", options.String("number"))
	core.AssertEqual(t, "Fix watcher", options.String("title"))
}

func TestActions_OptionHelpers_Ugly_IgnoreMalformedMapJSON(t *testing.T) {
	input := dispatchInputFromOptions(core.NewOptions(
		core.Option{Key: "repo", Value: "go-io"},
		core.Option{Key: "task", Value: "Review"},
		core.Option{Key: "variables", Value: "{\"BROKEN\""},
	))

	core.AssertNil(t, input.Variables)
}
