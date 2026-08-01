// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"cmp"
	"slices"

	core "dappco.re/go"
)

// Actions are projected onto three planes — CLI, API and MCP — so the command
// registry legitimately carries several hundred entries, and each one is also
// registered under an `agentic:` (or `core/`) prefix so the CLI answers to the
// same names the other two planes use.
//
// Printed as one flat list that is a wall of ~355 lines. This file renders it
// the way a normal CLI does: the handful of top-level commands, then the
// groups, then one group at a time on request.
//
//	core-agent            → top-level commands + the group index
//	core-agent help       → the same
//	core-agent help plan  → just the plan group

// helpTopLevelOrder pins the commands a person actually types to the top of
// the listing, in a deliberate order, rather than alphabetically among 355.
var helpTopLevelOrder = []string{
	"version", "check", "env",
	"mcp", "serve", "hub", "chat", "shell",
	"setup", "status", "dispatch", "scan",
}

// helpAliasPrefixes are the plane-parity prefixes. `agentic:plan/list` is the
// same command as `plan/list`; showing both doubles the listing for no reader
// benefit, so the prefixed form is folded away and mentioned once as a note.
var helpAliasPrefixes = []string{"agentic:", "core/"}

// helpGroup is one command family — "plan", "fleet", "pipeline". A family's
// name is often an invocable command in its own right (`dispatch` dispatches;
// `dispatch/start` starts the runner), in which case description carries what
// the bare command does so the index line still says it.
type helpGroup struct {
	name        string
	description string
	commands    []helpCommand
}

type helpCommand struct {
	path        string
	leaf        string
	description string
}

// canonicalHelpPath reduces a registered path to the one spelling the listing
// shows. Three separate aliasing habits are folded away:
//
//   - the plane-parity prefixes, `agentic:` and `core/`
//   - `:` as a group separator, registered alongside `/`
//   - `_` in a leaf, registered alongside `-`
//
// Every spelling still works when typed; only the listing collapses them.
//
//	canonicalHelpPath("agentic:plan/list")   // "plan/list"
//	canonicalHelpPath("core/pipeline/fix")   // "pipeline/fix"
//	canonicalHelpPath("brain:recall")        // "brain/recall"
//	canonicalHelpPath("phase/add_checkpoint") // "phase/add-checkpoint"
func canonicalHelpPath(path string) string {
	for _, prefix := range helpAliasPrefixes {
		if core.HasPrefix(path, prefix) {
			path = path[len(prefix):]
			break
		}
	}
	return core.Replace(core.Replace(path, ":", "/"), "_", "-")
}

// helpGroupOf returns the family a command belongs to — its first path
// segment, or "" for a top-level command with no segments.
//
//	helpGroupOf("plan/list") // "plan"
//	helpGroupOf("version")   // ""
func helpGroupOf(path string) string {
	if idx := core.Index(path, "/"); idx > 0 {
		return path[:idx]
	}
	return ""
}

// helpLeafOf returns the part of the path below the group.
//
//	helpLeafOf("pipeline/fix/threads") // "fix/threads"
func helpLeafOf(path string) string {
	if idx := core.Index(path, "/"); idx > 0 {
		return path[idx+1:]
	}
	return path
}

// collectHelp walks the command registry once and returns the top-level
// commands and the grouped remainder, aliases folded and each list sorted.
func collectHelp(c *core.Core) ([]helpCommand, []helpGroup) {
	if c == nil {
		return nil, nil
	}

	seen := make(map[string]helpCommand)
	for _, path := range c.Commands() {
		canonical := canonicalHelpPath(path)
		if _, done := seen[canonical]; done {
			continue
		}
		r := c.Command(path)
		if !r.OK {
			continue
		}
		cmd, ok := r.Value.(*core.Command)
		if !ok || cmd == nil || cmd.Hidden {
			continue
		}
		// A group parent with no action of its own ("plan", "fleet") is a
		// heading, not something to invoke — the group index already names it.
		if cmd.Action == nil && !cmd.IsManaged() {
			continue
		}
		seen[canonical] = helpCommand{
			path:        canonical,
			leaf:        helpLeafOf(canonical),
			description: cmd.Description,
		}
	}

	var top []helpCommand
	grouped := make(map[string][]helpCommand)
	for canonical, entry := range seen {
		if group := helpGroupOf(canonical); group != "" {
			grouped[group] = append(grouped[group], entry)
			continue
		}
		top = append(top, entry)
	}

	groups := sortHelpGroups(mergeTwinGroups(grouped))
	top = dropDescriptionTwins(top)
	return sortHelpTopLevel(liftGroupHeadings(top, groups)), groups
}

// dropDescriptionTwins removes a command that is another one under a second
// name. `prep` / `prep-workspace` and `plan/status` / `plan/update-status`
// each share an action and a description; only the spelling differs. The
// shorter name is kept — every spelling still works when typed.
func dropDescriptionTwins(cmds []helpCommand) []helpCommand {
	slices.SortFunc(cmds, func(a, b helpCommand) int {
		if n := cmp.Compare(len(a.path), len(b.path)); n != 0 {
			return n
		}
		return cmp.Compare(a.path, b.path)
	})
	seen := make(map[string]bool, len(cmds))
	kept := cmds[:0]
	for _, cmd := range cmds {
		// An empty description carries no evidence of being a twin.
		if cmd.description != "" {
			if seen[cmd.description] {
				continue
			}
			seen[cmd.description] = true
		}
		kept = append(kept, cmd)
	}
	return kept
}

// mergeTwinGroups folds a group into another that offers exactly the same
// leaves — `messages/{conversation,inbox,send}` is the same family as
// `message/{conversation,inbox,send}`, registered twice. The shorter name
// wins, and the listing stops showing the family twice.
func mergeTwinGroups(grouped map[string][]helpCommand) map[string][]helpCommand {
	signature := func(cmds []helpCommand) string {
		leaves := make([]string, 0, len(cmds))
		for _, cmd := range cmds {
			leaves = append(leaves, cmd.leaf)
		}
		slices.Sort(leaves)
		return core.Join("\x00", leaves...)
	}

	keep := make(map[string]string) // signature → surviving group name
	for name, cmds := range grouped {
		sig := signature(cmds)
		winner, seen := keep[sig]
		if !seen || len(name) < len(winner) || (len(name) == len(winner) && name < winner) {
			keep[sig] = name
		}
	}

	merged := make(map[string][]helpCommand, len(keep))
	for name, cmds := range grouped {
		if keep[signature(cmds)] == name {
			merged[name] = cmds
		}
	}
	return merged
}

// liftGroupHeadings moves a top-level command that names a group onto that
// group, so `dispatch` is described on the dispatch index line rather than
// listed a second time above it. Its description is not lost — that was the
// bug in listing it twice *and* the bug in simply dropping it.
func liftGroupHeadings(top []helpCommand, groups []helpGroup) []helpCommand {
	byName := make(map[string]*helpGroup, len(groups))
	for i := range groups {
		byName[groups[i].name] = &groups[i]
	}
	kept := top[:0]
	for _, cmd := range top {
		if group, isHeading := byName[cmd.path]; isHeading {
			group.description = cmd.description
			continue
		}
		kept = append(kept, cmd)
	}
	return kept
}

// sortHelpTopLevel puts the helpTopLevelOrder entries first, in that order,
// then everything else alphabetically.
func sortHelpTopLevel(top []helpCommand) []helpCommand {
	rank := make(map[string]int, len(helpTopLevelOrder))
	for i, name := range helpTopLevelOrder {
		rank[name] = i
	}
	slices.SortFunc(top, func(a, b helpCommand) int {
		ra, oka := rank[a.path]
		rb, okb := rank[b.path]
		switch {
		case oka && okb:
			return ra - rb
		case oka:
			return -1
		case okb:
			return 1
		}
		return cmp.Compare(a.path, b.path)
	})
	return top
}

// sortHelpGroups orders the groups by name and each group's commands by path.
func sortHelpGroups(grouped map[string][]helpCommand) []helpGroup {
	groups := make([]helpGroup, 0, len(grouped))
	for name, cmds := range grouped {
		cmds = dropDescriptionTwins(cmds)
		slices.SortFunc(cmds, func(a, b helpCommand) int { return cmp.Compare(a.path, b.path) })
		groups = append(groups, helpGroup{name: name, commands: cmds})
	}
	slices.SortFunc(groups, func(a, b helpGroup) int { return cmp.Compare(a.name, b.name) })
	return groups
}

// help is the `help` command action. With no argument it prints the overview;
// with one it prints that group, or reports the group as unknown.
func (commands applicationCommandSet) help(opts core.Options) core.Result {
	c := commands.coreApp
	top, groups := collectHelp(c)

	if group := core.Trim(opts.String("_arg")); group != "" {
		return printHelpGroup(group, groups)
	}
	printHelpOverview(c, top, groups)
	return core.Result{OK: true}
}

// printHelpOverview prints the banner, the top-level commands, and a one-line
// index of every group with its command count and a sample of its leaves.
func printHelpOverview(c *core.Core, top []helpCommand, groups []helpGroup) {
	name := "core-agent"
	if c != nil && c.App() != nil && c.App().Name != "" {
		name = c.App().Name
	}
	if c != nil && c.Cli() != nil {
		applicationPrint("%s", c.Cli().Banner())
	}
	applicationPrint("")
	applicationPrint("Usage:")
	applicationPrint("  %s <command> [options]", name)
	applicationPrint("  %s help <group>          the commands in one group", name)
	applicationPrint("")

	applicationPrint("Commands:")
	for _, cmd := range top {
		applicationPrint("  %-22s %s", cmd.path, cmd.description)
	}

	applicationPrint("")
	applicationPrint("Command groups — %s help <group>:", name)
	for _, group := range groups {
		applicationPrint("  %-14s %4d  %s", group.name, len(group.commands), helpGroupSummary(group))
	}

	applicationPrint("")
	applicationPrint("Every command also answers to `agentic:<command>` (and `core/<command>` for")
	applicationPrint("pipeline) — the same names the API and MCP planes use for the action.")
}

// printHelpGroup prints one group in full.
func printHelpGroup(name string, groups []helpGroup) core.Result {
	for _, group := range groups {
		if group.name != name {
			continue
		}
		applicationPrint("%s — %d commands", group.name, len(group.commands))
		applicationPrint("")
		for _, cmd := range group.commands {
			applicationPrint("  %-34s %s", cmd.path, cmd.description)
		}
		return core.Result{OK: true}
	}

	// An unknown group is a typo, and a typo must not exit 0.
	names := make([]string, 0, len(groups))
	for _, group := range groups {
		names = append(names, group.name)
	}
	return core.Result{
		Value: core.E("main.help", core.Concat("unknown command group: ", name,
			" (known: ", core.Join(", ", names...), ")"), nil),
		OK: false,
	}
}

// helpGroupSummary is the right-hand side of an index line: what the bare
// command does when the group name is itself invocable, otherwise a sample of
// the leaves so the line still says what the family is for.
func helpGroupSummary(group helpGroup) string {
	if group.description != "" {
		return group.description
	}
	return helpSample(group)
}

// helpSample names the first few leaves in a group so the index line says what
// the group is for without the reader having to open it.
func helpSample(group helpGroup) string {
	const max = 4
	leaves := make([]string, 0, max)
	for _, cmd := range group.commands {
		if len(leaves) == max {
			return core.Concat(core.Join(", ", leaves...), ", …")
		}
		leaves = append(leaves, cmd.leaf)
	}
	return core.Join(", ", leaves...)
}
