// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"testing"

	"dappco.re/go"
	agentpkg "dappco.re/go/agent"
	"dappco.re/go/agent/pkg/agentic"
)

// newTestCore creates a minimal Core with application commands registered.
func newTestCore(t *testing.T) *core.Core {
	t.Helper()
	c := core.New(core.WithOption("name", "core-agent"))
	c.App().Version = "test"
	registerApplicationCommands(c)
	c.Cli().SetOutput(core.NewBuffer())
	return c
}

func withArgs(t *testing.T, args ...string) {
	t.Helper()
	previous := startupArgv
	startupArgv = func() []string {
		return append([]string(nil), args...)
	}
	t.Cleanup(func() {
		startupArgv = previous
	})
}

func captureStdout(t *testing.T, run func()) string {
	t.Helper()
	buffer := core.NewBuffer()
	previous := applicationPrint
	applicationPrint = func(format string, args ...any) {
		core.Print(buffer, format, args...)
	}
	defer func() {
		applicationPrint = previous
	}()

	run()
	return buffer.String()
}

func TestCommands_ApplyLogLevel_Good_Case(t *testing.T) {
	defer core.SetLevel(core.LevelInfo)

	args := applyLogLevel([]string{"--quiet", "version"})
	core.AssertEqual(t, []string{"version"}, args)
}

func TestCommands_ApplyLogLevel_Bad_Case(t *testing.T) {
	defer core.SetLevel(core.LevelInfo)

	args := applyLogLevel([]string{"status"})
	core.AssertEqual(t, []string{"status"}, args)
}

func TestCommands_ApplyLogLevel_Ugly_Case(t *testing.T) {
	defer core.SetLevel(core.LevelInfo)

	args := applyLogLevel([]string{"version", "-q"})
	core.AssertEqual(t, []string{"version"}, args)
}

func TestCommands_StartupArgs_Good_Case(t *testing.T) {
	defer core.SetLevel(core.LevelInfo)

	withArgs(t, "core-agent", "--debug", "check")
	args := startupArgs()
	core.AssertEqual(t, []string{"check"}, args)
}

func TestCommands_StartupArgs_Bad_Case(t *testing.T) {
	defer core.SetLevel(core.LevelInfo)

	withArgs(t, "core-agent", "status")
	args := startupArgs()
	core.AssertEqual(t, []string{"status"}, args)
}

func TestCommands_StartupArgs_Ugly_Case(t *testing.T) {
	defer core.SetLevel(core.LevelInfo)

	withArgs(t, "core-agent", "version", "-q")
	args := startupArgs()
	core.AssertEqual(t, []string{"version"}, args)
}

func TestCommands_RegisterApplicationCommands_Good_Case(t *testing.T) {
	c := newTestCore(t)
	cmds := c.Commands()
	core.AssertContains(t, cmds, "version")
	core.AssertContains(t, cmds, "check")
	core.AssertContains(t, cmds, "env")
}

func TestCommands_Version_Good(t *testing.T) {
	c := newTestCore(t)
	agentpkg.Version = "0.8.0"
	t.Cleanup(func() {
		agentpkg.Version = ""
	})

	r := c.Cli().Run("version")
	core.AssertTrue(t, r.OK)
}

func TestCommands_VersionDev_Bad_Case(t *testing.T) {
	c := newTestCore(t)
	agentpkg.Version = ""
	c.App().Version = "dev"

	r := c.Cli().Run("version")
	core.AssertTrue(t, r.OK)
}

func TestCommands_Check_Good_Case(t *testing.T) {
	c := newTestCore(t)

	r := c.Cli().Run("check")
	core.AssertTrue(t, r.OK)
}

func TestCommands_Check_Good_BranchWorkspaceCount(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	c := newTestCore(t)

	wsRoot := core.JoinPath(root, "workspace")
	ws := core.JoinPath(wsRoot, "core", "go-io", "feature", "new-ui")
	core.AssertTrue(t, agentic.LocalFs().EnsureDir(agentic.WorkspaceRepoDir(ws)).OK)
	core.AssertTrue(t, agentic.LocalFs().EnsureDir(agentic.WorkspaceMetaDir(ws)).OK)
	core.AssertTrue(t, agentic.LocalFs().Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(agentic.WorkspaceStatus{
		Status: "running",
		Repo:   "go-io",
		Agent:  "codex",
	})).OK)

	output := captureStdout(t, func() {
		r := c.Cli().Run("check")
		core.AssertTrue(t, r.OK)
	})

	core.AssertContains(t, output, "1 workspaces")
}

func TestCommands_Env_Good_Case(t *testing.T) {
	c := newTestCore(t)

	r := c.Cli().Run("env")
	core.AssertTrue(t, r.OK)
}

func TestCommands_CliUnknown_Bad_Case(t *testing.T) {
	c := newTestCore(t)
	r := c.Cli().Run("nonexistent")
	core.AssertFalse(t, r.OK)
}

func TestCommands_CliBanner_Good_Case(t *testing.T) {
	c := newTestCore(t)
	c.Cli().SetBanner(func(_ *core.Cli) string {
		return "core-agent test"
	})
	r := c.Cli().Run()
	_ = r
}

func TestCommands_CliEmptyArgs_Ugly_Case(t *testing.T) {
	c := newTestCore(t)
	r := c.Cli().Run()
	_ = r
}
