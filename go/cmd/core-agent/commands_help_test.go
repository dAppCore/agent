// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"testing"

	core "dappco.re/go"
)

// A bare help request routes to the `help` command, which is what makes the
// grouped listing (rather than Cli.PrintHelp's flat 355-line wall) the thing a
// person sees, and what makes it exit 0.
func TestCommands_RouteHelp_Good_BareHelpRoutesToHelpCommand(t *testing.T) {
	core.AssertEqual(t, []string{"help"}, routeHelp(nil))
	for _, arg := range []string{"help", "-h", "--help"} {
		core.AssertEqual(t, []string{"help"}, routeHelp([]string{arg}))
	}
}

// A help token that belongs to a subcommand is left alone — `serve --help`
// must still reach serve, which owns its own help.
func TestCommands_RouteHelp_Bad_SubcommandHelpSurvives(t *testing.T) {
	core.AssertEqual(t, []string{"serve", "--help"}, routeHelp([]string{"serve", "--help"}))
	core.AssertEqual(t, []string{"mcp", "-h"}, routeHelp([]string{"mcp", "-h"}))
	core.AssertEqual(t, []string{"help", "plan"}, routeHelp([]string{"help", "plan"}))
}

// Non-help arguments pass through untouched.
func TestCommands_RouteHelp_Ugly_NonHelpArgsUntouched(t *testing.T) {
	core.AssertEqual(t, []string{"version"}, routeHelp([]string{"version"}))
	core.AssertEqual(t, []string{"hub", "help"}, routeHelp([]string{"hub", "help"}))
}

// startupArgs is the real entry point: argv holding only --help must come out
// as the help command.
func TestCommands_StartupArgs_Good_TopLevelHelpBecomesHelpCommand(t *testing.T) {
	defer core.SetLevel(core.LevelInfo)

	withArgs(t, "core-agent", "--help")
	core.AssertEqual(t, []string{"help"}, startupArgs())
}

// The three aliasing habits the listing folds. Every spelling still resolves
// when typed; canonicalHelpPath decides which one is shown.
func TestCommandsHelp_CanonicalHelpPath_Good_FoldsEveryAliasSpelling(t *testing.T) {
	cases := map[string]string{
		"agentic:plan/list":           "plan/list",    // plane-parity prefix
		"core/pipeline/fix":           "pipeline/fix", // plane-parity prefix
		"brain:recall":                "brain/recall", // colon as group separator
		"phase/add_checkpoint":        "phase/add-checkpoint",
		"agentic:phase/update_status": "phase/update-status", // both at once
		"plan/list":                   "plan/list",           // already canonical
		"version":                     "version",
	}
	for input, want := range cases {
		core.AssertEqual(t, want, canonicalHelpPath(input))
	}
}

func TestCommandsHelp_HelpGroupOf_Good_SplitsOnFirstSegment(t *testing.T) {
	core.AssertEqual(t, "plan", helpGroupOf("plan/list"))
	core.AssertEqual(t, "pipeline", helpGroupOf("pipeline/fix/threads"))
	core.AssertEqual(t, "", helpGroupOf("version"))
	core.AssertEqual(t, "fix/threads", helpLeafOf("pipeline/fix/threads"))
	core.AssertEqual(t, "version", helpLeafOf("version"))
}

// Two groups offering exactly the same leaves are one family registered twice
// (`message` / `messages`). The shorter name survives.
func TestCommandsHelp_MergeTwinGroups_Good_FoldsIdenticalLeafSets(t *testing.T) {
	leaves := func(names ...string) []helpCommand {
		out := make([]helpCommand, 0, len(names))
		for _, n := range names {
			out = append(out, helpCommand{leaf: n})
		}
		return out
	}
	merged := mergeTwinGroups(map[string][]helpCommand{
		"message":  leaves("send", "inbox", "conversation"),
		"messages": leaves("conversation", "send", "inbox"),
		"plan":     leaves("list", "create"),
	})

	core.AssertEqual(t, 2, len(merged))
	if _, kept := merged["message"]; !kept {
		t.Fatal("shorter group name should survive the merge")
	}
	if _, dropped := merged["messages"]; dropped {
		t.Fatal("longer twin group should have been folded away")
	}
}

// A group whose leaves differ is a different family and must not be merged.
func TestCommandsHelp_MergeTwinGroups_Bad_KeepsDistinctFamilies(t *testing.T) {
	merged := mergeTwinGroups(map[string][]helpCommand{
		"plan":  {{leaf: "list"}},
		"phase": {{leaf: "get"}},
	})
	core.AssertEqual(t, 2, len(merged))
}

// Same action, two spellings, one description — show it once, keep the short
// name. `plan/status` and `plan/update-status` are the live example.
func TestCommandsHelp_DropDescriptionTwins_Good_KeepsShortestSpelling(t *testing.T) {
	kept := dropDescriptionTwins([]helpCommand{
		{path: "plan/update-status", description: "Read or update an implementation plan status"},
		{path: "plan/status", description: "Read or update an implementation plan status"},
		{path: "plan/list", description: "List implementation plans"},
	})

	core.AssertEqual(t, 2, len(kept))
	paths := []string{kept[0].path, kept[1].path}
	core.AssertContains(t, paths, "plan/status")
	core.AssertContains(t, paths, "plan/list")
}

// Commands with no description carry no evidence of being twins, so they must
// all survive rather than collapsing onto each other.
func TestCommandsHelp_DropDescriptionTwins_Ugly_EmptyDescriptionsAllSurvive(t *testing.T) {
	kept := dropDescriptionTwins([]helpCommand{
		{path: "a"}, {path: "b"}, {path: "c"},
	})
	core.AssertEqual(t, 3, len(kept))
}

// `dispatch` is both a group and a runnable command. It must not be listed
// twice, and its description must not be lost — it moves onto the group's
// index line.
func TestCommandsHelp_LiftGroupHeadings_Good_MovesDescriptionOntoGroup(t *testing.T) {
	groups := []helpGroup{{name: "dispatch", commands: []helpCommand{{path: "dispatch/start"}}}}
	top := liftGroupHeadings([]helpCommand{
		{path: "dispatch", description: "Dispatch queued agents"},
		{path: "version", description: "Print version and build info"},
	}, groups)

	core.AssertEqual(t, 1, len(top))
	core.AssertEqual(t, "version", top[0].path)
	core.AssertEqual(t, "Dispatch queued agents", groups[0].description)
	core.AssertEqual(t, "Dispatch queued agents", helpGroupSummary(groups[0]))
}

// A group whose name is not itself a command falls back to naming its leaves,
// so the index line still says what the family is for.
func TestCommandsHelp_HelpGroupSummary_Bad_FallsBackToLeafSample(t *testing.T) {
	core.AssertEqual(t, "get, list, sync", helpGroupSummary(helpGroup{
		name:     "repo",
		commands: []helpCommand{{leaf: "get"}, {leaf: "list"}, {leaf: "sync"}},
	}))
}

// More leaves than fit are elided rather than printed in full.
func TestCommandsHelp_HelpSample_Ugly_ElidesBeyondFour(t *testing.T) {
	group := helpGroup{commands: []helpCommand{
		{leaf: "a"}, {leaf: "b"}, {leaf: "c"}, {leaf: "d"}, {leaf: "e"},
	}}
	core.AssertEqual(t, "a, b, c, d, …", helpSample(group))
}

// newGroupedTestCore adds the shape the services register at startup — a
// family under `plan/`, the same family under its `agentic:` alias, and a
// bare `plan` heading. newTestCore alone carries only the twelve top-level
// application commands, so it has no groups to exercise.
func newGroupedTestCore(t *testing.T) *core.Core {
	t.Helper()
	c := newTestCore(t)
	noop := func(core.Options) core.Result { return core.Result{OK: true} }
	for path, desc := range map[string]string{
		"plan":                "Manage implementation plans",
		"plan/list":           "List implementation plans",
		"plan/create":         "Create an implementation plan",
		"agentic:plan/list":   "List implementation plans",
		"agentic:plan/create": "Create an implementation plan",
		"brain:recall":        "Recall memories from OpenBrain",
		"brain/recall":        "Recall memories from OpenBrain",
	} {
		if r := c.Command(path, core.Command{Description: desc, Action: noop}); !r.OK {
			t.Fatalf("registering %q: %v", path, r.Value)
		}
	}
	return c
}

// The end-to-end shape: groups exist, every listed path is already canonical
// (no alias spelling survives), and the aliases really did fold rather than
// being listed twice.
func TestCommandsHelp_CollectHelp_Good_RegistryIsGroupedAndFolded(t *testing.T) {
	top, groups := collectHelp(newGroupedTestCore(t))

	if len(groups) == 0 {
		t.Fatal("expected the registry to produce command groups")
	}
	if len(top) == 0 {
		t.Fatal("expected top-level commands")
	}
	for _, group := range groups {
		for _, cmd := range group.commands {
			core.AssertEqual(t, cmd.path, canonicalHelpPath(cmd.path))
		}
	}
	for _, cmd := range top {
		core.AssertEqual(t, cmd.path, canonicalHelpPath(cmd.path))
	}

	// plan/{list,create} once each, not twice via agentic:.
	for _, group := range groups {
		if group.name == "plan" {
			core.AssertEqual(t, 2, len(group.commands))
			core.AssertEqual(t, "Manage implementation plans", group.description)
		}
	}
	// The bare `plan` heading was lifted onto the group, not listed above it.
	for _, cmd := range top {
		if cmd.path == "plan" {
			t.Fatal("group heading `plan` should have been lifted onto the plan group")
		}
	}
}

// A nil core must not panic — collectHelp is called from a command action, and
// a coded empty result beats a crash.
func TestCommandsHelp_CollectHelp_Ugly_NilCoreReturnsNothing(t *testing.T) {
	top, groups := collectHelp(nil)
	core.AssertEqual(t, 0, len(top))
	core.AssertEqual(t, 0, len(groups))
}

// Naming a group prints it; naming a group that does not exist is a typo and
// must fail rather than quietly exit 0.
func TestCommandsHelp_Help_Good_KnownGroupSucceeds(t *testing.T) {
	commands := applicationCommandSet{coreApp: newGroupedTestCore(t)}
	opts := core.NewOptions()
	opts.Set("_arg", "plan")
	core.AssertTrue(t, commands.help(opts).OK)
}

func TestCommandsHelp_Help_Bad_UnknownGroupFails(t *testing.T) {
	commands := applicationCommandSet{coreApp: newTestCore(t)}
	opts := core.NewOptions()
	opts.Set("_arg", "nosuchgroup")
	core.AssertFalse(t, commands.help(opts).OK)
}

func TestCommandsHelp_Help_Ugly_NoArgumentPrintsOverview(t *testing.T) {
	commands := applicationCommandSet{coreApp: newTestCore(t)}
	core.AssertTrue(t, commands.help(core.NewOptions()).OK)
}

// runApp must not report failure for a bare invocation or a bare --help.
func TestMain_RunApp_Good_HelpAndBareInvocationSucceed(t *testing.T) {
	defer core.SetLevel(core.LevelInfo)

	for _, args := range [][]string{nil, {"help"}} {
		c := newTestCore(t)
		if err := runApp(c, args); err != nil {
			t.Fatalf("runApp(%v) returned %v, want nil", args, err)
		}
	}
}

// An argument that matches no command is a real failure — a typo must not
// exit 0 just because help was printed alongside it.
func TestMain_RunApp_Bad_UnknownCommandStillFails(t *testing.T) {
	defer core.SetLevel(core.LevelInfo)

	c := newTestCore(t)
	if err := runApp(c, []string{"nosuchcommand"}); err == nil {
		t.Fatal("runApp returned nil for an unknown command, want an error")
	}
}
