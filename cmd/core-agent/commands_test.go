// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"bytes"
	"os"
	"testing"

	agentpkg "dappco.re/go/agent"
	"dappco.re/go/core"
	"github.com/stretchr/testify/assert"
)

// newTestCore creates a minimal Core with app commands registered.
func newTestCore(t *testing.T) *core.Core {
	t.Helper()
	c := core.New(core.WithOption("name", "core-agent"))
	c.App().Version = "test"
	registerAppCommands(c)
	c.Cli().SetOutput(&bytes.Buffer{})
	return c
}

func withArgs(t *testing.T, args ...string) {
	t.Helper()
	previous := os.Args
	os.Args = append([]string(nil), args...)
	t.Cleanup(func() {
		os.Args = previous
	})
}

func TestCommands_ApplyLogLevel_Good(t *testing.T) {
	defer core.SetLevel(core.LevelInfo)

	args := applyLogLevel([]string{"--quiet", "version"})
	assert.Equal(t, []string{"version"}, args)
}

func TestCommands_ApplyLogLevel_Bad(t *testing.T) {
	defer core.SetLevel(core.LevelInfo)

	args := applyLogLevel([]string{"status"})
	assert.Equal(t, []string{"status"}, args)
}

func TestCommands_ApplyLogLevel_Ugly(t *testing.T) {
	defer core.SetLevel(core.LevelInfo)

	args := applyLogLevel([]string{"version", "-q"})
	assert.Equal(t, []string{"version"}, args)
}

func TestCommands_StartupArgs_Good(t *testing.T) {
	defer core.SetLevel(core.LevelInfo)

	withArgs(t, "core-agent", "--debug", "check")
	args := startupArgs()
	assert.Equal(t, []string{"check"}, args)
}

func TestCommands_StartupArgs_Bad(t *testing.T) {
	defer core.SetLevel(core.LevelInfo)

	withArgs(t, "core-agent", "status")
	args := startupArgs()
	assert.Equal(t, []string{"status"}, args)
}

func TestCommands_StartupArgs_Ugly(t *testing.T) {
	defer core.SetLevel(core.LevelInfo)

	withArgs(t, "core-agent", "version", "-q")
	args := startupArgs()
	assert.Equal(t, []string{"version"}, args)
}

func TestCommands_RegisterAppCommands_Good(t *testing.T) {
	c := newTestCore(t)
	cmds := c.Commands()
	assert.Contains(t, cmds, "version")
	assert.Contains(t, cmds, "check")
	assert.Contains(t, cmds, "env")
}

func TestCommands_Version_Good(t *testing.T) {
	c := newTestCore(t)
	agentpkg.Version = "0.8.0"
	t.Cleanup(func() {
		agentpkg.Version = ""
	})

	r := c.Cli().Run("version")
	assert.True(t, r.OK)
}

func TestCommands_VersionDev_Bad(t *testing.T) {
	c := newTestCore(t)
	agentpkg.Version = ""
	c.App().Version = "dev"

	r := c.Cli().Run("version")
	assert.True(t, r.OK)
}

func TestCommands_Check_Good(t *testing.T) {
	c := newTestCore(t)

	r := c.Cli().Run("check")
	assert.True(t, r.OK)
}

func TestCommands_Env_Good(t *testing.T) {
	c := newTestCore(t)

	r := c.Cli().Run("env")
	assert.True(t, r.OK)
}

func TestCommands_CliUnknown_Bad(t *testing.T) {
	c := newTestCore(t)
	r := c.Cli().Run("nonexistent")
	assert.False(t, r.OK)
}

func TestCommands_CliBanner_Good(t *testing.T) {
	c := newTestCore(t)
	c.Cli().SetBanner(func(_ *core.Cli) string {
		return "core-agent test"
	})
	r := c.Cli().Run()
	_ = r
}

func TestCommands_CliEmptyArgs_Ugly(t *testing.T) {
	c := newTestCore(t)
	r := c.Cli().Run()
	_ = r
}
